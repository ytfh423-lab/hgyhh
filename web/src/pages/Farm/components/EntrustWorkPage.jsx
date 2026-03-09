import React, { useCallback, useEffect, useState } from 'react';
import { Button, Empty, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { ArrowLeft, RefreshCw } from 'lucide-react';
import { API, showError, showSuccess, formatDuration } from './utils';

const { Text, Title } = Typography;

const actionMap = {
  water: { label: '浇水', emoji: '💧' },
  fertilize: { label: '施肥', emoji: '🧴' },
  harvest: { label: '收获', emoji: '🌾' },
  treat: { label: '治疗', emoji: '💊' },
  ranch_feed: { label: '牧场喂食', emoji: '🌾' },
  ranch_water: { label: '牧场喂水', emoji: '💧' },
  ranch_clean: { label: '牧场清理', emoji: '🧹' },
  tree_water: { label: '树场浇水', emoji: '💧' },
  tree_harvest: { label: '树场采收', emoji: '🍎' },
  tree_chop: { label: '树场伐木', emoji: '🪓' },
};

const EntrustWorkPage = ({ taskId, onBack, t }) => {
  const [workData, setWorkData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [executing, setExecuting] = useState(null);
  const [completed, setCompleted] = useState(false);

  const loadWork = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get(`/api/farm/entrust/work?id=${taskId}`);
      if (res.success) {
        setWorkData(res.data);
        if (res.data.task?.status === 'completed') setCompleted(true);
      } else {
        showError(res.message);
      }
    } catch (err) { showError(t('加载失败')); }
    finally { setLoading(false); }
  }, [taskId, t]);

  useEffect(() => { loadWork(); }, [loadWork]);

  const handleExecute = async (entityId) => {
    setExecuting(entityId);
    try {
      const { data: res } = await API.post('/api/farm/entrust/work/execute', {
        task_id: taskId,
        entity_id: entityId,
      });
      if (res.success) {
        showSuccess(res.message);
        if (res.completed) {
          setCompleted(true);
        }
        loadWork();
      } else {
        showError(res.message);
      }
    } catch (err) { showError(t('操作失败')); }
    finally { setExecuting(null); }
  };

  if (loading && !workData) {
    return <div style={{ textAlign: 'center', padding: 60 }}><Spin size='large' /></div>;
  }

  if (!workData) {
    return (
      <div className='farm-card' style={{ textAlign: 'center', padding: 30 }}>
        <Empty description={t('无法加载工作数据')} />
        <Button onClick={onBack} style={{ marginTop: 12 }}>{t('返回')}</Button>
      </div>
    );
  }

  const task = workData.task || {};
  const entities = workData.entities || [];
  const myProgress = workData.my_progress || 0;
  const actDef = actionMap[task.target_action] || {};
  const remaining = task.remaining_secs > 0 ? formatDuration(task.remaining_secs) : t('已过期');
  const progressPct = task.target_count > 0 ? Math.min(100, task.progress_count / task.target_count * 100) : 0;
  const doneCount = entities.filter(e => e.done).length;
  const actionableCount = entities.filter(e => e.actionable).length;

  return (
    <div>
      {/* 顶部导航 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
        <Button icon={<ArrowLeft size={16} />} theme='borderless' onClick={onBack}
          className='farm-btn' />
        <Title heading={6} style={{ margin: 0 }}>
          {actDef.emoji || '📋'} {t('委托工作模式')}
        </Title>
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless'
          onClick={loadWork} loading={loading} className='farm-btn' style={{ marginLeft: 'auto' }} />
      </div>

      {/* 完成提示 */}
      {completed && (
        <div className='farm-card' style={{
          background: 'linear-gradient(135deg, rgba(74,124,63,0.15), rgba(111,168,94,0.1))',
          border: '1px solid rgba(74,124,63,0.3)', marginBottom: 12, textAlign: 'center', padding: '16px',
        }}>
          <span style={{ fontSize: 40 }}>🎉</span>
          <Title heading={5} style={{ margin: '8px 0 4px' }}>{t('任务已完成！')}</Title>
          <Text type='tertiary'>{t('报酬将自动结算到你的账户')}</Text>
          <div style={{ marginTop: 12 }}>
            <Button theme='solid' onClick={onBack} className='farm-btn'
              style={{ background: 'linear-gradient(135deg, var(--farm-leaf), var(--farm-sky))' }}>
              {t('返回委托列表')}
            </Button>
          </div>
        </div>
      )}

      {/* 任务信息卡 */}
      <div className='farm-card' style={{ marginBottom: 12 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 8, marginBottom: 8 }}>
          <div>
            <Text strong style={{ fontSize: 15 }}>{task.title}</Text>
            <Tag size='small' color='blue' style={{ marginLeft: 8 }}>{t('工作中')}</Tag>
          </div>
          <Text type='tertiary' size='small'>⏱ {remaining}</Text>
        </div>
        <div style={{ display: 'flex', gap: 16, fontSize: 12, color: 'var(--farm-text-2)', flexWrap: 'wrap', marginBottom: 8 }}>
          <span>🎯 {t('总进度')}: {task.progress_count}/{task.target_count}</span>
          <span>👤 {t('我的进度')}: {myProgress}</span>
          <span>💰 {task.reward_display}</span>
        </div>
        {/* 总进度条 */}
        <div className='farm-progress' style={{ height: 8 }}>
          <div className='farm-progress-fill' style={{
            width: `${progressPct}%`,
            background: progressPct >= 100
              ? 'var(--farm-leaf)'
              : 'linear-gradient(90deg, var(--farm-sky), var(--farm-leaf))',
            transition: 'width 0.3s ease',
          }} />
        </div>
        <div style={{ fontSize: 11, color: 'var(--farm-text-3)', marginTop: 4 }}>
          ✅ {doneCount} {t('已完成')} · ⚡ {actionableCount} {t('可操作')}
        </div>
      </div>

      {/* 操作实体列表 */}
      <div className='farm-card'>
        <div className='farm-section-title'>
          {actDef.emoji} {t('操作列表')} — {t('点击按钮执行操作')}
        </div>
        {entities.length === 0 ? (
          <Empty description={t('暂无可操作的目标')} style={{ padding: 20 }} />
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {entities.map((entity, idx) => (
              <div key={entity.id ?? idx} style={{
                display: 'flex', alignItems: 'center', gap: 10, padding: '8px 12px',
                borderRadius: 8,
                background: entity.done
                  ? 'rgba(74,124,63,0.06)'
                  : entity.actionable
                    ? 'rgba(90,143,180,0.06)'
                    : 'rgba(128,128,128,0.04)',
                border: entity.actionable
                  ? '1px solid rgba(90,143,180,0.2)'
                  : '1px solid transparent',
                transition: 'all 0.2s',
              }}>
                <span style={{ fontSize: 18, flexShrink: 0 }}>
                  {entity.done ? '✅' : entity.actionable ? actDef.emoji || '⚡' : '⬜'}
                </span>
                <Text style={{ flex: 1, fontSize: 13, opacity: entity.done ? 0.6 : 1 }}>
                  {entity.label}
                </Text>
                {entity.done ? (
                  <Tag size='small' color='green'>{t('已完成')}</Tag>
                ) : entity.actionable ? (
                  <Button size='small' theme='solid' loading={executing === entity.id}
                    disabled={executing !== null}
                    onClick={() => handleExecute(entity.id)}
                    className='farm-btn'
                    style={{ background: 'linear-gradient(135deg, var(--farm-sky), var(--farm-leaf))', minWidth: 60 }}>
                    {actDef.emoji} {t('执行')}
                  </Button>
                ) : (
                  <Tag size='small' color='grey'>{t('不可操作')}</Tag>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 底部提示 */}
      <div style={{ marginTop: 12, padding: '8px 12px', borderRadius: 8, background: 'rgba(200,146,42,0.06)',
        border: '1px solid rgba(200,146,42,0.15)', fontSize: 11, color: 'var(--farm-text-3)', lineHeight: 1.6 }}>
        ℹ️ {t('你正在代替雇主执行操作。所有产出归雇主所有，你将获得任务报酬。操作消耗的物品/费用由雇主承担。')}
      </div>
    </div>
  );
};

export default EntrustWorkPage;
