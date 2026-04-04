import React, { useCallback, useEffect, useState, useMemo } from 'react';
import { Button, Spin, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess, formatDuration } from './utils';
import { farmConfirm } from './farmConfirm';

const { Text } = Typography;

/* ═══════════════════════════════════════════════════════════════
   ProfitBadge — 利润分级徽章
   ═══════════════════════════════════════════════════════════════ */
const ProfitBadge = ({ profit, multiplier, t }) => {
  let cls = 'positive';
  let label = `+$${profit.toFixed(2)}`;
  if (profit < 0) { cls = 'negative'; label = `-$${Math.abs(profit).toFixed(2)}`; }
  else if (multiplier >= 150) { cls = 'gold'; label = `🔥 +$${profit.toFixed(2)}`; }
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 2 }}>
      <span className={`farm-ws-profit-badge ${cls}`}>{label}</span>
      <span className='farm-ws-ratio'>{multiplier}%</span>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   WorkshopPage — 智能工厂控制台
   ═══════════════════════════════════════════════════════════════ */
const WorkshopPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [wsData, setWsData] = useState(null);
  const [wsLoading, setWsLoading] = useState(false);
  const [tick, setTick] = useState(0);
  const [craftingKey, setCraftingKey] = useState(null);
  const [sortBy, setSortBy] = useState('profit');

  const loadWorkshop = useCallback(async () => {
    setWsLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/workshop');
      if (res.success) setWsData(res.data);
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setWsLoading(false);
    }
  }, [t]);

  useEffect(() => { loadWorkshop(); }, [loadWorkshop]);

  useEffect(() => {
    const timer = setInterval(() => setTick(p => p + 1), 5000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    if (tick > 0) loadWorkshop();
  }, [tick, loadWorkshop]);

  const doCraft = async (key) => {
    setCraftingKey(key);
    setWsLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/workshop/craft', { recipe_key: key });
      if (res.success) {
        showSuccess(res.message);
        loadWorkshop();
        loadFarm();
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setWsLoading(false);
      setCraftingKey(null);
    }
  };

  const doCollect = async () => {
    setWsLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/workshop/collect');
      if (res.success) { showSuccess(res.message); loadWorkshop(); loadFarm(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
    finally { setWsLoading(false); }
  };

  const doCollectStore = async () => {
    setWsLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/workshop/collect/store');
      if (res.success) { showSuccess(res.message); loadWorkshop(); loadFarm(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
    finally { setWsLoading(false); }
  };

  const sortedRecipes = useMemo(() => {
    const list = [...(wsData?.recipes || [])];
    if (sortBy === 'profit') list.sort((a, b) => b.profit - a.profit);
    else if (sortBy === 'time') list.sort((a, b) => a.time_secs - b.time_secs);
    else if (sortBy === 'ratio') list.sort((a, b) => b.multiplier - a.multiplier);
    return list;
  }, [wsData?.recipes, sortBy]);

  if (wsLoading && !wsData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!wsData) return null;

  const hasCollectable = (wsData.active || []).some(p => p.status === 2);
  const slotsAvailable = wsData.used_slots < wsData.max_slots;

  return (
    <div>
      {/* ═══ Factory Status Bar ═══ */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 14, alignItems: 'center', flexWrap: 'wrap' }}>
        <div className='farm-pill farm-pill-blue'>
          🏭 {t('产线')}: {wsData.used_slots}/{wsData.max_slots}
        </div>
        {slotsAvailable
          ? <div className='farm-pill farm-pill-green'>✅ {t('有空闲槽位')}</div>
          : <div className='farm-pill farm-pill-red'>⛔ {t('满载运转')}</div>
        }
        {hasCollectable && (
          <>
            <Button theme='solid' type='warning' size='small' loading={wsLoading}
              onClick={doCollect} className='farm-btn'>
              📥 {t('收取成品出售')}
            </Button>
            <Button theme='light' type='primary' size='small' loading={wsLoading}
              onClick={doCollectStore} className='farm-btn'>
              📦 {t('收取成品入仓')}
            </Button>
          </>
        )}
      </div>

      {/* ═══ Active Processes ═══ */}
      {wsData.active && wsData.active.length > 0 && (
        <div className='farm-card' style={{ marginBottom: 14 }}>
          <div className='farm-section-title'>
            <span className='farm-ws-gear'>⚙️</span> {t('生产线运行中')}
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {wsData.active.map((p) => (
              <div key={p.id} className='farm-ws-active'>
                <span className='farm-ws-active-emoji'>{p.emoji}</span>
                <div className='farm-ws-active-info'>
                  <div className='farm-ws-active-name'>{p.name}</div>
                  <div className='farm-ws-active-bar'>
                    <div className={`farm-ws-active-fill ${p.status === 2 ? 'done' : ''}`}
                      style={{ width: `${p.status === 2 ? 100 : p.progress}%` }} />
                  </div>
                  <div className='farm-ws-active-meta'>
                    <span>
                      {p.status === 2
                        ? `✅ ${t('已完成')} — ${t('等待收取')}`
                        : `⏱ ${formatDuration(p.remaining)} · ${p.progress}%`
                      }
                    </span>
                    <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <span>${p.sell_price.toFixed(2)}</span>
                      {p.status !== 2 && (
                        <Button size='small' theme='light' type='danger'
                          onClick={async () => { if (await farmConfirm(t('取消加工'), t('确定要取消这个加工任务吗？材料不会退还。'), { icon: '🏭', confirmType: 'danger', confirmText: t('取消加工') })) { doAction('/api/farm/workshop/cancel', { process_id: p.id }).then(r => { if (r) { loadWorkshop(); loadFarm(); } }); } }}
                          loading={wsLoading} className='farm-btn' style={{ padding: '0 6px', fontSize: 11 }}>
                          ✕
                        </Button>
                      )}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ═══ Sort Controls ═══ */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 10, alignItems: 'center' }}>
        <Text type='tertiary' size='small' style={{ marginRight: 4 }}>📋 {t('排序')}:</Text>
        {[
          { key: 'profit', label: '💰 ' + t('利润') },
          { key: 'ratio', label: '📊 ' + t('利润率') },
          { key: 'time', label: '⏱ ' + t('耗时') },
        ].map(s => (
          <div key={s.key}
            className={`farm-shop-preset ${sortBy === s.key ? 'active' : ''}`}
            onClick={() => setSortBy(s.key)}
            style={{ cursor: 'pointer' }}>
            {s.label}
          </div>
        ))}
      </div>

      {/* ═══ Recipe Pipeline ═══ */}
      <div className='farm-ws-pipeline'>
        {sortedRecipes.map((r) => (
          <div key={r.key}
            className={`farm-ws-recipe ${r.multiplier >= 150 ? 'profit-high' : ''}`}>
            {/* Input Node */}
            <div className='farm-ws-node'>
              <span className='farm-ws-node-emoji'>💰</span>
              <span className='farm-ws-node-label'>{t('成本')}</span>
              <span className='farm-ws-node-value'>${r.cost.toFixed(2)}</span>
            </div>

            {/* Arrow 1 */}
            <div className='farm-ws-arrow'>
              <span className='farm-ws-arrow-line'>→</span>
            </div>

            {/* Process Node */}
            <div className='farm-ws-node'>
              <span className='farm-ws-node-emoji'>{r.emoji}</span>
              <span className='farm-ws-node-label'>{r.name}</span>
              <span className='farm-ws-arrow-time'>⏱ {formatDuration(r.time_secs)}</span>
            </div>

            {/* Arrow 2 */}
            <div className='farm-ws-arrow'>
              <span className='farm-ws-arrow-line'>→</span>
            </div>

            {/* Output Node */}
            <div className='farm-ws-node'>
              <span className='farm-ws-node-emoji'>📦</span>
              <span className='farm-ws-node-label'>{t('售价')}</span>
              <span className='farm-ws-node-value'>${r.sell_price.toFixed(2)}</span>
            </div>

            {/* Profit + Action */}
            <div className='farm-ws-profit'>
              <ProfitBadge profit={r.profit} multiplier={r.multiplier} t={t} />
              <Button size='small' theme='solid'
                disabled={!slotsAvailable || wsLoading}
                loading={craftingKey === r.key}
                onClick={() => doCraft(r.key)}
                className='farm-btn'
                style={{ minWidth: 60 }}>
                {craftingKey === r.key ? t('投料中') : '⚙️ ' + t('加工')}
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default WorkshopPage;
