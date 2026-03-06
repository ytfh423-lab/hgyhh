import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess, formatDuration } from './utils';

const { Text } = Typography;

const WorkshopPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [wsData, setWsData] = useState(null);
  const [wsLoading, setWsLoading] = useState(false);
  const [tick, setTick] = useState(0);

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
    }
  };

  const doCollect = async () => {
    setWsLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/workshop/collect');
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
    }
  };

  if (wsLoading && !wsData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!wsData) return null;

  const hasCollectable = (wsData.active || []).some(p => p.status === 2);
  const slotsAvailable = wsData.used_slots < wsData.max_slots;
  const profitColor = (v) => v >= 0 ? '#16a34a' : '#dc2626';

  return (
    <div>
      {/* Slots */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 14, alignItems: 'center', flexWrap: 'wrap' }}>
        <div className='farm-pill farm-pill-blue'>🏭 {t('槽位')}: {wsData.used_slots}/{wsData.max_slots}</div>
        {hasCollectable && (
          <Button theme='solid' type='warning' size='small' loading={wsLoading} onClick={doCollect} className='farm-btn'>
            📥 {t('收取全部')}
          </Button>
        )}
      </div>

      {/* Active processes */}
      {wsData.active && wsData.active.length > 0 && (
        <div className='farm-card'>
          <div className='farm-section-title'>⏳ {t('加工中')}</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {wsData.active.map((p) => (
              <div key={p.id} className='farm-row'>
                <span style={{ fontSize: 24 }}>{p.emoji}</span>
                <div style={{ flex: 1 }}>
                  <Text strong>{p.name}</Text>
                  {p.status === 2 ? (
                    <Tag size='small' color='green' style={{ marginLeft: 6 }}>✅ {t('已完成')}</Tag>
                  ) : (
                    <Tag size='small' color='blue' style={{ marginLeft: 6 }}>{p.progress}% · {formatDuration(p.remaining)}</Tag>
                  )}
                  <Text size='small' type='tertiary' style={{ display: 'block' }}>{t('价值')}: ${p.sell_price.toFixed(2)}</Text>
                </div>
                {p.status === 1 && (
                  <div className='farm-progress' style={{ width: 80 }}>
                    <div className='farm-progress-fill' style={{ width: `${p.progress}%`, background: 'linear-gradient(90deg, #3b82f6, #06b6d4)' }} />
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Recipes */}
      <div className='farm-card'>
        <div className='farm-section-title'>📋 {t('配方列表')}</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {(wsData.recipes || []).map((r) => (
            <div key={r.key} className='farm-row'>
              <span style={{ fontSize: 22 }}>{r.emoji}</span>
              <div style={{ flex: 1 }}>
                <Text strong>{r.name}</Text>
                <Text size='small' type='tertiary' style={{ display: 'block' }}>
                  {t('成本')} ${r.cost.toFixed(2)} → {t('售价')} ${r.sell_price.toFixed(2)} ({r.multiplier}%)
                  · <span style={{ color: profitColor(r.profit), fontWeight: 600 }}>{r.profit >= 0 ? '+' : ''}${r.profit.toFixed(2)}</span>
                  · {formatDuration(r.time_secs)}
                </Text>
              </div>
              <Button size='small' theme='solid' disabled={!slotsAvailable || wsLoading}
                onClick={() => doCraft(r.key)} className='farm-btn'>
                {t('加工')}
              </Button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default WorkshopPage;
