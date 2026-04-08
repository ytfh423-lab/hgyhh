import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './utils';

const { Text } = Typography;

const AchievementsPage = ({ actionLoading, loadFarm, t }) => {
  const [achData, setAchData] = useState(null);
  const [loading, setLoading] = useState(false);

  const loadAch = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/achievements');
      if (res.success) setAchData(res.data);
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => { loadAch(); }, [loadAch]);

  const claimAch = async (key) => {
    try {
      const { data: res } = await API.post('/api/farm/achievements/claim', { key });
      if (res.success) {
        showSuccess(res.message);
        loadAch();
        loadFarm({ silent: true });
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    }
  };

  if (loading && !achData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!achData) return null;

  const unlockCount = (achData.achievements || []).filter(a => a.unlocked).length;
  const totalCount = (achData.achievements || []).length;

  return (
    <div>
      <div className='farm-pill farm-pill-blue' style={{ marginBottom: 14 }}>🏆 {unlockCount}/{totalCount} {t('已解锁')}</div>

      <div className='farm-card'>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {(achData.achievements || []).map((ach) => (
            <div key={ach.key} className='farm-row' style={{
              background: ach.unlocked ? 'rgba(74,124,63,0.08)' : undefined,
              opacity: ach.unlocked ? 1 : ach.done ? 1 : 0.6,
              marginBottom: 0,
            }}>
              <span style={{ fontSize: 24 }}>{ach.emoji}</span>
              <div style={{ flex: 1 }}>
                <Text strong>{ach.name}</Text>
                <Text size='small' type='tertiary' style={{ display: 'block' }}>{ach.description}</Text>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
                  <div className='farm-progress' style={{ width: 80 }}>
                    <div className='farm-progress-fill' style={{
                      width: `${Math.min(100, (ach.progress / ach.target) * 100)}%`,
                      background: ach.unlocked ? 'var(--farm-leaf)' : ach.done ? 'var(--farm-harvest)' : 'var(--farm-sky)',
                    }} />
                  </div>
                  <Text size='small' type='tertiary'>{ach.progress}/{ach.target}</Text>
                  <Text size='small' type='tertiary'>· ${ach.reward.toFixed(2)}</Text>
                </div>
              </div>
              {ach.unlocked ? (
                <Tag size='small' color='green'>✅</Tag>
              ) : ach.done ? (
                <Button size='small' theme='solid' type='warning' onClick={() => claimAch(ach.key)} className='farm-btn'>
                  {t('领取')}
                </Button>
              ) : null}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default AchievementsPage;
