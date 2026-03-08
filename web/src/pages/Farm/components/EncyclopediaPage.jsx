import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './utils';

const { Text } = Typography;

const EncyclopediaPage = ({ actionLoading, loadFarm, t }) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/encyclopedia');
      if (res.success) setData(res.data);
    } catch (err) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const claim = async (category) => {
    try {
      const { data: res } = await API.post('/api/farm/encyclopedia/claim', { category });
      if (res.success) { showSuccess(res.message); load(); loadFarm(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
  };

  if (loading && !data) return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  if (!data) return null;

  return (
    <div>
      <div style={{ display: 'flex', gap: 8, marginBottom: 14, alignItems: 'center' }}>
        <div className='farm-pill farm-pill-blue'>📖 {data.total_unlocked}/{data.total_items} {t('已发现')}</div>
        {data.all_complete && !data.grand_claimed && (
          <Button size='small' theme='solid' type='warning' onClick={() => claim('grand')} className='farm-btn'>
            🏆 {t('领取大师奖励')} (${data.grand_reward})
          </Button>
        )}
      </div>

      {(data.categories || []).map(cat => (
        <div key={cat.key} className='farm-card'>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
            <div className='farm-section-title' style={{ marginBottom: 0 }}>{cat.name} ({cat.unlocked}/{cat.total})</div>
            {cat.complete && !cat.claimed && (
              <Button size='small' theme='solid' type='warning' onClick={() => claim(cat.key)} className='farm-btn'>
                {t('领取')} ${cat.reward}
              </Button>
            )}
            {cat.claimed && <span className='farm-pill farm-pill-green' style={{ padding: '2px 8px', fontSize: 11 }}>✅</span>}
          </div>
          <div className='farm-progress' style={{ height: 4, marginBottom: 10 }}>
            <div className='farm-progress-fill' style={{
              width: `${Math.round(cat.unlocked / cat.total * 100)}%`,
              background: 'linear-gradient(90deg, var(--farm-sky), var(--farm-harvest))',
            }} />
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {(cat.items || []).map(it => (
              <div key={it.key} style={{
                padding: '8px 12px', borderRadius: 10, minWidth: 80, textAlign: 'center',
                backdropFilter: 'var(--farm-blur)',
                background: it.unlocked
                  ? 'linear-gradient(135deg, rgba(74,124,63,0.1), rgba(74,124,63,0.05))'
                  : 'var(--farm-glass-bg)',
                border: it.unlocked ? '1.5px solid rgba(74,124,63,0.3)' : '1px dashed var(--farm-glass-border)',
                opacity: it.unlocked ? 1 : 0.5,
                transition: 'all 0.2s',
              }}>
                {it.unlocked ? (
                  <>
                    <span style={{ fontSize: 22, display: 'block', marginBottom: 2 }}>{it.emoji}</span>
                    <Text size='small' strong style={{ display: 'block' }}>{it.name}</Text>
                  </>
                ) : (
                  <>
                    <span style={{ fontSize: 18, display: 'block', marginBottom: 2, filter: 'grayscale(1)' }}>🔒</span>
                    <Text size='small' type='tertiary' style={{ display: 'block', fontSize: 11 }}>{t('未解锁')}</Text>
                  </>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};

export default EncyclopediaPage;
