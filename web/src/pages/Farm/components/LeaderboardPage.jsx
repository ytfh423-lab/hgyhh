import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Empty, Spin, Typography } from '@douyinfe/semi-ui';
import { API } from './utils';
import {
  FARM_LEADERBOARD_PERIODS,
  FARM_LEADERBOARD_SCOPES,
  FARM_LEADERBOARD_TYPES,
  formatFarmLeaderboardValue,
  getFarmLeaderboardReward,
} from './leaderboardUtils';

const { Text } = Typography;

const LeaderboardPage = ({ t }) => {
  const [data, setData] = useState(null);
  const [boardType, setBoardType] = useState('balance');
  const [scope, setScope] = useState('global');
  const [period, setPeriod] = useState('all');
  const [loading, setLoading] = useState(false);

  const load = useCallback(async (type, boardScope, boardPeriod) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ type, scope: boardScope, period: boardPeriod });
      const { data: res } = await API.get(`/api/farm/leaderboard?${params.toString()}`);
      if (res.success) setData(res.data);
    } catch (err) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(boardType, scope, period); }, [load, boardType, scope, period]);

  const medals = ['🥇', '🥈', '🥉'];
  const myReward = useMemo(() => getFarmLeaderboardReward(data?.my_rank), [data?.my_rank]);

  return (
    <div>
      <div style={{ display: 'flex', gap: 6, marginBottom: 14, flexWrap: 'wrap' }}>
        {FARM_LEADERBOARD_TYPES.map(tp => (
          <div key={tp.key}
            className={`farm-pill ${boardType === tp.key ? 'farm-pill-blue' : ''}`}
            style={{ cursor: 'pointer' }} onClick={() => setBoardType(tp.key)}>
            {tp.icon} {t(tp.label)}
          </div>
        ))}
      </div>

      <div style={{ display: 'flex', gap: 6, marginBottom: 10, flexWrap: 'wrap' }}>
        {FARM_LEADERBOARD_SCOPES.map(tp => (
          <div key={tp.key}
            className={`farm-pill ${scope === tp.key ? 'farm-pill-green' : ''}`}
            style={{ cursor: 'pointer' }} onClick={() => setScope(tp.key)}>
            {tp.icon} {t(tp.label)}
          </div>
        ))}
      </div>

      <div style={{ display: 'flex', gap: 6, marginBottom: 14, flexWrap: 'wrap' }}>
        {FARM_LEADERBOARD_PERIODS.map(tp => (
          <div key={tp.key}
            className={`farm-pill ${period === tp.key ? 'farm-pill-amber' : ''}`}
            style={{ cursor: 'pointer' }} onClick={() => setPeriod(tp.key)}>
            {tp.icon} {t(tp.label)}
          </div>
        ))}
      </div>

      {data?.title && (
        <div style={{ marginBottom: 10 }}>
          <div className='farm-pill farm-pill-blue'>{t('当前榜单')}: {data.title}</div>
        </div>
      )}

      <div className='farm-card' style={{ marginBottom: 14 }}>
        <div className='farm-section-title'>🎁 {t('榜单荣誉')}</div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {[1, 2, 3].map((rank) => {
            const reward = getFarmLeaderboardReward(rank);
            return (
              <div key={`reward-${rank}`} className='farm-pill farm-pill-amber'>
                {reward.emoji} #{rank} {t(reward.title)}
              </div>
            );
          })}
        </div>
      </div>

      {data?.my_rank && (
        <div style={{ display: 'flex', gap: 8, marginBottom: 14, flexWrap: 'wrap' }}>
          <div className='farm-pill farm-pill-cyan'>📊 {t('我的排名')}: #{data.my_rank}</div>
          <div className='farm-pill farm-pill-blue'>{t('我的')}{t(data?.label || '数值')}: {formatFarmLeaderboardValue(boardType, data.my_value)}</div>
          {myReward && (
            <div className='farm-pill farm-pill-amber'>{myReward.emoji} {t(myReward.title)}</div>
          )}
          {data.gap_to_prev > 0 && (
            <div className='farm-pill farm-pill-amber'>{t('距上一名还差')}: {formatFarmLeaderboardValue(boardType, data.gap_to_prev)}</div>
          )}
          {data.gap_to_prev <= 0 && data.my_rank === 1 && (
            <div className='farm-pill farm-pill-green'>{t('你目前就是第一名')}</div>
          )}
        </div>
      )}

      {loading ? <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div> : (
        <div className='farm-card'>
          {data?.title && (
            <div className='farm-section-title' style={{ marginBottom: 10 }}>🏅 {data.title}</div>
          )}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {(data?.items || []).map(item => {
              const reward = getFarmLeaderboardReward(item.rank);
              return (
                <div key={item.rank} className='farm-row' style={{
                  marginBottom: 0,
                  borderColor: item.is_me ? 'var(--semi-color-primary)' : undefined,
                  boxShadow: item.is_me ? 'var(--farm-shadow), var(--farm-glow-blue)' : undefined,
                  fontWeight: item.is_me ? 600 : 400,
                }}>
                  <span style={{ fontSize: 18, width: 30, textAlign: 'center' }}>
                    {item.rank <= 3 ? medals[item.rank - 1] : `#${item.rank}`}
                  </span>
                  <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 8, minWidth: 0, flexWrap: 'wrap' }}>
                    <Text>{item.name}</Text>
                    {reward && (
                      <div className='farm-pill farm-pill-amber'>
                        {reward.emoji} {t(reward.shortTitle)}
                      </div>
                    )}
                  </div>
                  <Text strong>{formatFarmLeaderboardValue(boardType, item.value)}</Text>
                </div>
              );
            })}
            {(!data?.items || data.items.length === 0) && (
              <Empty description={t('暂无数据')} />
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default LeaderboardPage;
