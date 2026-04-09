import React, { useCallback, useEffect, useState } from 'react';
import { Button, Empty, Modal, Spin, Tag, Typography, InputNumber, Select, Radio, RadioGroup } from '@douyinfe/semi-ui';
import { RefreshCw } from 'lucide-react';
import { API, showError, showSuccess, formatDuration, formatBalance } from './utils';
import { farmConfirm } from './farmConfirm';

const { Text, Title } = Typography;

const entrustActionTypes = [
  { key: 'water', module: 'farm', label: '浇水', emoji: '💧', unit: '块地' },
  { key: 'fertilize', module: 'farm', label: '施肥', emoji: '🧴', unit: '块地' },
  { key: 'harvest', module: 'farm', label: '收获', emoji: '🌾', unit: '块地' },
  { key: 'treat', module: 'farm', label: '治疗', emoji: '💊', unit: '块地' },
  { key: 'ranch_feed', module: 'ranch', label: '牧场喂食', emoji: '🌾', unit: '只' },
  { key: 'ranch_water', module: 'ranch', label: '牧场喂水', emoji: '💧', unit: '只' },
  { key: 'ranch_clean', module: 'ranch', label: '牧场清理', emoji: '🧹', unit: '次' },
  { key: 'tree_water', module: 'tree', label: '树场浇水', emoji: '💧', unit: '棵' },
  { key: 'tree_harvest', module: 'tree', label: '树场采收', emoji: '🍎', unit: '棵' },
  { key: 'tree_chop', module: 'tree', label: '树场伐木', emoji: '🪓', unit: '棵' },
];
const actionMap = {};
entrustActionTypes.forEach(a => { actionMap[a.key] = a; });

const statusLabels = {
  published: '招募中', in_progress: '进行中', completed: '已完成',
  cancelled: '已取消', expired: '已过期',
};
const statusColors = {
  published: 'green', in_progress: 'blue', completed: 'light-green',
  cancelled: 'grey', expired: 'red',
};
const workerStatusLabels = {
  accepted: '已接单', working: '工作中', completed: '已完成', abandoned: '已放弃',
};

const ENTRUST_CACHE_TTL = 15000;
const entrustDataCache = {
  hall: null,
  published: null,
  accepted: null,
};
const entrustDataCacheOwnerId = {
  hall: 0,
  published: 0,
  accepted: 0,
};
const entrustDataCacheExpiresAt = {
  hall: 0,
  published: 0,
  accepted: 0,
};
const entrustPendingMap = {
  hall: null,
  published: null,
  accepted: null,
};
const entrustPendingOwnerId = {
  hall: 0,
  published: 0,
  accepted: 0,
};

const getCurrentEntrustCacheOwnerId = () => {
  if (typeof window === 'undefined') {
    return 0;
  }
  try {
    return JSON.parse(window.localStorage.getItem('user') || '{}')?.id || 0;
  } catch (_) {
    return 0;
  }
};

const readEntrustCache = (key) => {
  if (
    entrustDataCache[key] &&
    entrustDataCacheExpiresAt[key] > Date.now() &&
    entrustDataCacheOwnerId[key] === getCurrentEntrustCacheOwnerId()
  ) {
    return entrustDataCache[key];
  }
  return null;
};

const writeEntrustCache = (key, data) => {
  entrustDataCache[key] = data;
  entrustDataCacheExpiresAt[key] = Date.now() + ENTRUST_CACHE_TTL;
  entrustDataCacheOwnerId[key] = getCurrentEntrustCacheOwnerId();
  return data;
};

const invalidateEntrustCache = (...keys) => {
  keys.forEach((key) => {
    entrustDataCache[key] = null;
    entrustDataCacheExpiresAt[key] = 0;
    entrustDataCacheOwnerId[key] = 0;
  });
};

/* ═══════════════════════════════════════════
   TaskCard — 任务卡片（大厅 / 我的委托通用）
   ═══════════════════════════════════════════ */
const TaskCard = ({ task, mode, onAccept, onCancel, onWork, actionLoading, t }) => {
  const actDef = actionMap[task.target_action] || {};
  const remaining = task.remaining_secs > 0 ? formatDuration(task.remaining_secs) : '已过期';
  return (
    <div className='farm-card' style={{ marginBottom: 8, padding: '12px 16px' }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10, flexWrap: 'wrap' }}>
        <span style={{ fontSize: 28 }}>{actDef.emoji || '📋'}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4, flexWrap: 'wrap' }}>
            <Text strong style={{ fontSize: 14 }}>{task.title}</Text>
            <Tag size='small' color={statusColors[task.status] || 'grey'}>{statusLabels[task.status] || task.status}</Tag>
            {task.max_workers > 1 && <Tag size='small' color='blue'>👥 {t('多人')}</Tag>}
          </div>
          <div style={{ fontSize: 12, color: 'var(--farm-text-2)', display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            <span>{actDef.label || task.target_action}</span>
            <span>🎯 {task.progress_count}/{task.target_count}{actDef.unit || ''}</span>
            <span>💰 {task.reward_display}</span>
            <span>⏱ {remaining}</span>
          </div>
          {/* 进度条 */}
          {task.target_count > 0 && (
            <div className='farm-progress' style={{ marginTop: 6, height: 5, maxWidth: 200 }}>
              <div className='farm-progress-fill' style={{
                width: `${Math.min(100, task.progress_count / task.target_count * 100)}%`,
                background: task.progress_count >= task.target_count
                  ? 'var(--farm-leaf)' : 'linear-gradient(90deg, var(--farm-sky), var(--farm-leaf))',
              }} />
            </div>
          )}
          {/* 我的委托：显示已结算 */}
          {mode === 'owner' && task.settled_amount != null && (
            <div style={{ fontSize: 11, color: 'var(--farm-text-3)', marginTop: 4 }}>
              {t('已结算')}: ${(task.settled_amount || 0).toFixed(2)} · {t('可退回')}: ${(task.refundable || 0).toFixed(2)}
            </div>
          )}
        </div>
        <div style={{ display: 'flex', gap: 4, flexShrink: 0, flexWrap: 'wrap' }}>
          {mode === 'hall' && (task.status === 'published' || task.status === 'in_progress') && (
            <Button size='small' theme='solid' loading={actionLoading}
              onClick={() => onAccept(task.id)} className='farm-btn'
              style={{ background: 'linear-gradient(135deg, var(--farm-sky), var(--farm-leaf))' }}>
              ✋ {t('接单')}
            </Button>
          )}
          {mode === 'owner' && (task.status === 'published' || task.status === 'in_progress') && (
            <Button size='small' theme='light' type='danger' loading={actionLoading}
              onClick={async () => { if (await farmConfirm(t('取消委托'), t('确定取消该委托？已完成部分将按比例结算。'), { icon: '📋', confirmType: 'danger', confirmText: t('取消委托') })) onCancel(task.id); }}
              className='farm-btn'>
              ✕ {t('取消')}
            </Button>
          )}
          {mode === 'worker' && onWork && (
            <Button size='small' theme='solid' loading={actionLoading}
              onClick={() => onWork(task.id)} className='farm-btn'
              style={{ background: 'linear-gradient(135deg, var(--farm-harvest), var(--farm-soil))' }}>
              🔨 {t('工作')}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════
   CreateForm — 发布委托表单
   ═══════════════════════════════════════════ */
const CreateForm = ({ balance, onCreated, actionLoading, t }) => {
  const [form, setForm] = useState({
    target_action: 'water', target_count: 5, reward_amount: 0.5,
    deadline_hours: 24, settlement_mode: 'partial', max_workers: 1,
  });
  const [confirmVisible, setConfirmVisible] = useState(false);
  const [creating, setCreating] = useState(false);

  const actDef = actionMap[form.target_action] || {};
  const rewardDisplay = form.reward_amount.toFixed(2);
  const balanceDisplay = (balance || 0).toFixed(2);
  const afterBalance = ((balance || 0) - form.reward_amount).toFixed(2);
  const canAfford = (balance || 0) >= form.reward_amount;

  const handlePublish = () => {
    if (!canAfford) { showError(t('余额不足')); return; }
    if (form.target_count < 1) { showError(t('目标数量不能为0')); return; }
    if (form.reward_amount < 0.01) { showError(t('报酬最低 $0.01')); return; }
    setConfirmVisible(true);
  };

  const handleConfirm = async () => {
    setCreating(true);
    try {
      const { data: res } = await API.post('/api/farm/entrust/create', {
        target_action: form.target_action,
        target_count: form.target_count,
        reward_amount: form.reward_amount,
        deadline_hours: form.deadline_hours,
        settlement_mode: form.settlement_mode,
        max_workers: form.max_workers,
      });
      if (res.success) {
        showSuccess(res.message);
        setConfirmVisible(false);
        setForm({ target_action: 'water', target_count: 5, reward_amount: 0.5, deadline_hours: 24, settlement_mode: 'partial', max_workers: 1 });
        onCreated();
      } else {
        showError(res.message);
      }
    } catch (err) { showError(t('操作失败')); }
    finally { setCreating(false); }
  };

  return (
    <div className='farm-card'>
      <div className='farm-section-title'>📝 {t('发布委托')}</div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {/* 任务类型 */}
        <div>
          <Text size='small' type='tertiary'>{t('任务类型')}</Text>
          <div className='farm-grid farm-grid-2' style={{ marginTop: 4 }}>
            {entrustActionTypes.map(a => (
              <div key={a.key}
                className={`farm-item-card ${form.target_action === a.key ? 'selected' : ''}`}
                onClick={() => setForm(f => ({ ...f, target_action: a.key }))}
                style={{ padding: '8px 10px', textAlign: 'left', cursor: 'pointer' }}>
                <span style={{ fontSize: 18 }}>{a.emoji}</span>
                <Text strong size='small' style={{ marginLeft: 6 }}>{a.label}</Text>
                <Text type='tertiary' size='small' style={{ marginLeft: 4 }}>({a.unit})</Text>
              </div>
            ))}
          </div>
        </div>
        {/* 数量 + 报酬 */}
        <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
          <div style={{ flex: 1, minWidth: 120 }}>
            <Text size='small' type='tertiary'>{t('目标数量')} ({actDef.unit || ''})</Text>
            <InputNumber value={form.target_count} min={1} max={50}
              onChange={v => setForm(f => ({ ...f, target_count: v }))}
              style={{ width: '100%', marginTop: 4 }} />
          </div>
          <div style={{ flex: 1, minWidth: 120 }}>
            <Text size='small' type='tertiary'>{t('报酬 ($)')}</Text>
            <InputNumber value={form.reward_amount} min={0.01} max={500} step={0.1}
              onChange={v => setForm(f => ({ ...f, reward_amount: v }))}
              formatter={v => `$${Number(v).toFixed(2)}`}
              parser={v => parseFloat(v.replace('$', '')) || 0}
              style={{ width: '100%', marginTop: 4 }} />
          </div>
        </div>
        {/* 截止时间 + 结算模式 */}
        <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
          <div style={{ flex: 1, minWidth: 120 }}>
            <Text size='small' type='tertiary'>{t('截止时间')}</Text>
            <Select value={form.deadline_hours}
              onChange={v => setForm(f => ({ ...f, deadline_hours: v }))}
              style={{ width: '100%', marginTop: 4 }}>
              <Select.Option value={6}>6 {t('小时')}</Select.Option>
              <Select.Option value={12}>12 {t('小时')}</Select.Option>
              <Select.Option value={24}>24 {t('小时')}</Select.Option>
              <Select.Option value={48}>48 {t('小时')}</Select.Option>
              <Select.Option value={72}>72 {t('小时')}</Select.Option>
            </Select>
          </div>
          <div style={{ flex: 1, minWidth: 120 }}>
            <Text size='small' type='tertiary'>{t('结算模式')}</Text>
            <RadioGroup value={form.settlement_mode} type='button' size='small'
              onChange={e => setForm(f => ({ ...f, settlement_mode: e.target.value }))}
              style={{ marginTop: 4 }}>
              <Radio value='partial'>{t('按进度')}</Radio>
              <Radio value='full'>{t('全部完成')}</Radio>
            </RadioGroup>
          </div>
        </div>
        {/* 最大接单人数 */}
        <div>
          <Text size='small' type='tertiary'>{t('最大接单人数')}</Text>
          <InputNumber value={form.max_workers} min={1} max={5}
            onChange={v => setForm(f => ({ ...f, max_workers: v }))}
            style={{ width: 80, marginTop: 4 }} />
        </div>
        {/* 余额提示 */}
        <div style={{ padding: '8px 12px', borderRadius: 8, background: canAfford ? 'rgba(74,124,63,0.08)' : 'rgba(184,66,51,0.08)',
          border: `1px solid ${canAfford ? 'rgba(74,124,63,0.2)' : 'rgba(184,66,51,0.2)'}`, fontSize: 12 }}>
          <div>💰 {t('当前余额')}: <strong>${balanceDisplay}</strong></div>
          <div>📤 {t('需托管')}: <strong>${rewardDisplay}</strong></div>
          {canAfford
            ? <div style={{ color: 'var(--farm-leaf)' }}>✅ {t('托管后余额')}: ${afterBalance}</div>
            : <div style={{ color: 'var(--farm-danger)' }}>❌ {t('余额不足，无法发布')}</div>
          }
        </div>
        <Button theme='solid' disabled={!canAfford || actionLoading} loading={actionLoading}
          onClick={handlePublish} className='farm-btn'
          style={{ background: 'linear-gradient(135deg, var(--farm-harvest), var(--farm-soil))', width: '100%' }}>
          📋 {t('发布委托')}
        </Button>
      </div>

      {/* ═══ 二次确认弹窗 ═══ */}
      <Modal
        title={`✅ ${t('确认发布委托任务')}`}
        visible={confirmVisible}
        onCancel={() => setConfirmVisible(false)}
        footer={
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <Button onClick={() => setConfirmVisible(false)}>{t('取消')}</Button>
            <Button theme='solid' type='warning' loading={creating} onClick={handleConfirm}>
              💰 {t('确认支付并发布')}
            </Button>
          </div>
        }
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10, fontSize: 13 }}>
          <div className='farm-card' style={{ margin: 0, padding: '10px 14px' }}>
            <div><strong>{t('任务类型')}：</strong>{actDef.emoji} {actDef.label}</div>
            <div><strong>{t('目标数量')}：</strong>{form.target_count} {actDef.unit}</div>
            <div><strong>{t('截止时间')}：</strong>{form.deadline_hours} {t('小时')}</div>
            <div><strong>{t('结算模式')}：</strong>{form.settlement_mode === 'partial' ? t('按进度结算') : t('全部完成后结算')}</div>
          </div>
          <div style={{ padding: '10px 14px', borderRadius: 8, background: 'rgba(200,146,42,0.1)', border: '1px solid rgba(200,146,42,0.2)' }}>
            <div style={{ fontWeight: 700, marginBottom: 4 }}>💰 {t('支付信息')}</div>
            <div>{t('托管报酬')}：<strong style={{ color: 'var(--farm-harvest)' }}>${rewardDisplay}</strong></div>
            <div>{t('当前余额')}：${balanceDisplay}</div>
            <div>{t('托管后余额')}：${afterBalance}</div>
          </div>
          <div style={{ fontSize: 11, color: 'var(--farm-text-3)', lineHeight: 1.6 }}>
            ⚠️ {t('发布后报酬将进入平台托管账户。任务完成后系统自动结算给接单玩家。如任务取消或过期，未结算部分按规则退回。')}
          </div>
        </div>
      </Modal>
    </div>
  );
};

/* ═══════════════════════════════════════════
   EntrustPage — 主页面（Tab 切换）
   ═══════════════════════════════════════════ */
const EntrustPage = ({ farmData, actionLoading, doAction, loadFarm, onEnterWork, t }) => {
  const [tab, setTab] = useState('hall');
  const [hallData, setHallData] = useState(null);
  const [myPublished, setMyPublished] = useState(null);
  const [myAccepted, setMyAccepted] = useState(null);
  const [loading, setLoading] = useState(false);

  const loadEntrustData = useCallback(async (key, url, setter, options = {}) => {
    const force = options.force === true;
    const currentOwnerId = getCurrentEntrustCacheOwnerId();
    const cached = force ? null : readEntrustCache(key);
    if (cached) {
      setter(cached);
      setLoading(false);
      return cached;
    }
    setLoading(true);
    try {
      if (!force && entrustPendingMap[key] && entrustPendingOwnerId[key] === currentOwnerId) {
        const pendingData = await entrustPendingMap[key];
        if (pendingData) {
          setter(pendingData);
        }
        return pendingData;
      }
      entrustPendingOwnerId[key] = currentOwnerId;
      entrustPendingMap[key] = API.get(url, { disableDuplicate: true })
        .then(({ data: res }) => {
          if (res.success) {
            return writeEntrustCache(key, res.data);
          }
          return null;
        })
        .catch(() => null)
        .finally(() => {
          entrustPendingMap[key] = null;
          entrustPendingOwnerId[key] = 0;
        });
      const nextData = await entrustPendingMap[key];
      if (nextData) {
        setter(nextData);
      }
      return nextData;
    } catch (err) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  const loadHall = useCallback(async (options = {}) => {
    return loadEntrustData('hall', '/api/farm/entrust/hall', setHallData, options);
  }, [loadEntrustData]);

  const loadMyPublished = useCallback(async (options = {}) => {
    return loadEntrustData('published', '/api/farm/entrust/my-published', setMyPublished, options);
  }, [loadEntrustData]);

  const loadMyAccepted = useCallback(async (options = {}) => {
    return loadEntrustData('accepted', '/api/farm/entrust/my-accepted', setMyAccepted, options);
  }, [loadEntrustData]);

  useEffect(() => {
    if (tab === 'hall') loadHall();
    else if (tab === 'published') loadMyPublished();
    else if (tab === 'accepted') loadMyAccepted();
  }, [tab, loadHall, loadMyPublished, loadMyAccepted]);

  const handleAccept = async (taskId) => {
    setLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/entrust/accept', { task_id: taskId });
      if (res.success) {
        showSuccess(res.message);
        invalidateEntrustCache('hall', 'accepted');
        await loadHall({ force: true });
      }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
    finally { setLoading(false); }
  };

  const handleCancel = async (taskId) => {
    try {
      const { data: res } = await API.post('/api/farm/entrust/cancel', { task_id: taskId });
      if (res.success) {
        showSuccess(res.message);
        invalidateEntrustCache('published', 'hall');
        await loadMyPublished({ force: true });
        loadFarm({ silent: true });
      }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
  };

  const handleAbandon = async (taskId) => {
    if (!await farmConfirm(t('放弃委托'), t('确定放弃该委托？'), { icon: '🤝', confirmType: 'danger', confirmText: t('放弃') })) return;
    try {
      const { data: res } = await API.post('/api/farm/entrust/abandon', { task_id: taskId });
      if (res.success) {
        showSuccess(res.message);
        invalidateEntrustCache('accepted', 'hall');
        await loadMyAccepted({ force: true });
      }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
  };

  const tabs = [
    { key: 'hall', label: '🏛️ ' + t('任务大厅') },
    { key: 'create', label: '📝 ' + t('发布') },
    { key: 'published', label: '📤 ' + t('我的委托') },
    { key: 'accepted', label: '📥 ' + t('我的接单') },
  ];

  return (
    <div>
      {/* Tab 导航 */}
      <div style={{ display: 'flex', gap: 4, marginBottom: 12, flexWrap: 'wrap' }}>
        {tabs.map(tb => (
          <div key={tb.key}
            className={`farm-shop-preset ${tab === tb.key ? 'active' : ''}`}
            onClick={() => setTab(tb.key)}
            style={{ cursor: 'pointer', padding: '6px 14px', fontSize: 13, fontWeight: 600 }}>
            {tb.label}
          </div>
        ))}
        {tab !== 'create' && (
          <Button size='small' icon={<RefreshCw size={12} />} theme='borderless'
            onClick={() => { if (tab === 'hall') loadHall({ force: true }); else if (tab === 'published') loadMyPublished({ force: true }); else loadMyAccepted({ force: true }); }}
            loading={loading} className='farm-btn' style={{ marginLeft: 'auto' }} />
        )}
      </div>

      {/* ═══ 任务大厅 ═══ */}
      {tab === 'hall' && (
        loading && !hallData ? <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div> : (
          <div>
            {hallData?.tasks?.length > 0 ? hallData.tasks.map(task => (
              <TaskCard key={task.id} task={task} mode='hall' onAccept={handleAccept}
                actionLoading={actionLoading} t={t} />
            )) : (
              <div className='farm-card' style={{ textAlign: 'center', padding: 30 }}>
                <span style={{ fontSize: 36 }}>📋</span>
                <Title heading={6} style={{ marginTop: 8 }}>{t('暂无委托任务')}</Title>
                <Text type='tertiary' size='small'>{t('去发布一个吧！')}</Text>
              </div>
            )}
          </div>
        )
      )}

      {/* ═══ 发布委托 ═══ */}
      {tab === 'create' && (
        <CreateForm balance={farmData?.balance || 0} onCreated={() => {
          invalidateEntrustCache('hall', 'published');
          setTab('published');
          loadMyPublished({ force: true });
          loadFarm({ silent: true });
        }}
          actionLoading={actionLoading} t={t} />
      )}

      {/* ═══ 我的委托 ═══ */}
      {tab === 'published' && (
        loading && !myPublished ? <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div> : (
          <div>
            {myPublished?.length > 0 ? myPublished.map(task => (
              <TaskCard key={task.id} task={task} mode='owner' onCancel={handleCancel}
                actionLoading={actionLoading} t={t} />
            )) : (
              <Empty description={t('还没有发布过委托')} style={{ padding: 30 }} />
            )}
          </div>
        )
      )}

      {/* ═══ 我的接单 ═══ */}
      {tab === 'accepted' && (
        loading && !myAccepted ? <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div> : (
          <div>
            {myAccepted?.length > 0 ? myAccepted.map(item => (
              <div key={item.worker_id} className='farm-card' style={{ marginBottom: 8, padding: '12px 16px' }}>
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10, flexWrap: 'wrap' }}>
                  <span style={{ fontSize: 28 }}>{actionMap[item.task?.target_action]?.emoji || '📋'}</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4, flexWrap: 'wrap' }}>
                      <Text strong style={{ fontSize: 14 }}>{item.task?.title}</Text>
                      <Tag size='small' color={item.status === 'working' ? 'blue' : item.status === 'completed' ? 'green' : 'grey'}>
                        {workerStatusLabels[item.status] || item.status}
                      </Tag>
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--farm-text-2)' }}>
                      🎯 {t('我的进度')}: {item.my_progress}/{item.task?.target_count}
                      {item.reward_earned > 0 && <span> · 💰 {t('已获')}: ${(item.reward_earned || 0).toFixed(2)}</span>}
                    </div>
                    {item.task?.target_count > 0 && (
                      <div className='farm-progress' style={{ marginTop: 6, height: 5, maxWidth: 200 }}>
                        <div className='farm-progress-fill' style={{
                          width: `${Math.min(100, item.my_progress / item.task.target_count * 100)}%`,
                          background: 'linear-gradient(90deg, var(--farm-sky), var(--farm-leaf))',
                        }} />
                      </div>
                    )}
                  </div>
                  <div style={{ display: 'flex', gap: 4, flexShrink: 0 }}>
                    {(item.status === 'accepted' || item.status === 'working') && (
                      <>
                        <Button size='small' theme='solid' loading={actionLoading}
                          onClick={() => onEnterWork && onEnterWork(item.task?.id)}
                          className='farm-btn'
                          style={{ background: 'linear-gradient(135deg, var(--farm-harvest), var(--farm-soil))' }}>
                          🔨 {t('工作')}
                        </Button>
                        <Button size='small' theme='light' type='danger' loading={actionLoading}
                          onClick={() => handleAbandon(item.task?.id)} className='farm-btn'>
                          ✕
                        </Button>
                      </>
                    )}
                  </div>
                </div>
              </div>
            )) : (
              <Empty description={t('还没有接过委托')} style={{ padding: 30 }} />
            )}
          </div>
        )
      )}
    </div>
  );
};

export default EntrustPage;
