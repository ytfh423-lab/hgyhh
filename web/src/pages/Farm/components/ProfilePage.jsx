import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Empty, Spin, Typography } from '@douyinfe/semi-ui';
import { API, formatBalance } from './utils';
import {
  FARM_LEADERBOARD_PERIODS,
  FARM_LEADERBOARD_SCOPES,
  FARM_LEADERBOARD_TYPES,
  formatFarmLeaderboardValue,
  getFarmLeaderboardReward,
} from './leaderboardUtils';

const { Text } = Typography;

const ProfilePage = ({ farmData, t }) => {
  const [scope, setScope] = useState('global');
  const [period, setPeriod] = useState('all');
  const [loading, setLoading] = useState(false);
  const [rankSummaries, setRankSummaries] = useState([]);

  const loadRankSummaries = useCallback(async (boardScope, boardPeriod) => {
    setLoading(true);
    try {
      const results = await Promise.allSettled(
        FARM_LEADERBOARD_TYPES.map(async (board) => {
          const params = new URLSearchParams({ type: board.key, scope: boardScope, period: boardPeriod });
          const { data: res } = await API.get(`/api/farm/leaderboard?${params.toString()}`);
          if (!res.success || !res.data) {
            return null;
          }
          return {
            ...res.data,
            boardType: board.key,
            boardIcon: board.icon,
            boardLabel: board.label,
          };
        }),
      );
      const next = results
        .filter((result) => result.status === 'fulfilled' && result.value)
        .map((result) => result.value);
      setRankSummaries(next);
    } catch (error) {
      setRankSummaries([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadRankSummaries(scope, period);
  }, [loadRankSummaries, period, scope]);

  const bestRankEntry = useMemo(() => {
    return rankSummaries
      .filter((item) => item.my_rank > 0)
      .sort((a, b) => a.my_rank - b.my_rank)[0] || null;
  }, [rankSummaries]);

  const honorEntries = useMemo(() => {
    return rankSummaries
      .filter((item) => item.my_rank > 0 && item.my_rank <= 3)
      .map((item) => ({
        ...item,
        reward: getFarmLeaderboardReward(item.my_rank),
      }));
  }, [rankSummaries]);

  return (
    <div>
      <div className='farm-card' style={{ marginBottom: 14 }}>
        <div className='farm-section-title'>👤 {t('个人中心')}</div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 10 }}>
          <div className='farm-row' style={{ marginBottom: 0, flexDirection: 'column', alignItems: 'flex-start', gap: 4 }}>
            <Text type='tertiary'>{t('当前资产')}</Text>
            <Text strong>{formatBalance(farmData?.balance || 0)}</Text>
          </div>
          <div className='farm-row' style={{ marginBottom: 0, flexDirection: 'column', alignItems: 'flex-start', gap: 4 }}>
            <Text type='tertiary'>{t('农场等级')}</Text>
            <Text strong>Lv.{farmData?.user_level || 1}</Text>
          </div>
          <div className='farm-row' style={{ marginBottom: 0, flexDirection: 'column', alignItems: 'flex-start', gap: 4 }}>
            <Text type='tertiary'>{t('转生等级')}</Text>
            <Text strong>P{farmData?.prestige_level || 0}</Text>
          </div>
          <div className='farm-row' style={{ marginBottom: 0, flexDirection: 'column', alignItems: 'flex-start', gap: 4 }}>
            <Text type='tertiary'>{t('地块进度')}</Text>
            <Text strong>{farmData?.plot_count || 0}/{farmData?.max_plots || 0}</Text>
          </div>
          <div className='farm-row' style={{ marginBottom: 0, flexDirection: 'column', alignItems: 'flex-start', gap: 4 }}>
            <Text type='tertiary'>{t('任务完成')}</Text>
            <Text strong>
              {farmData?.task_summary ? `${farmData.task_summary.done}/${farmData.task_summary.total}` : '-'}
            </Text>
          </div>
          <div className='farm-row' style={{ marginBottom: 0, flexDirection: 'column', alignItems: 'flex-start', gap: 4 }}>
            <Text type='tertiary'>{t('转生加成')}</Text>
            <Text strong>+{farmData?.prestige_bonus || 0}%</Text>
          </div>
        </div>
        {bestRankEntry && (
          <div style={{ display: 'flex', gap: 8, marginTop: 12, flexWrap: 'wrap' }}>
            <div className='farm-pill farm-pill-cyan'>🏅 {t('当前最佳排名')}: #{bestRankEntry.my_rank}</div>
            <div className='farm-pill farm-pill-blue'>{bestRankEntry.boardIcon} {t(bestRankEntry.boardLabel)}</div>
            {getFarmLeaderboardReward(bestRankEntry.my_rank) && (
              <div className='farm-pill farm-pill-amber'>
                {getFarmLeaderboardReward(bestRankEntry.my_rank).emoji} {t(getFarmLeaderboardReward(bestRankEntry.my_rank).title)}
              </div>
            )}
          </div>
        )}
      </div>

      <div style={{ display: 'flex', gap: 6, marginBottom: 10, flexWrap: 'wrap' }}>
        {FARM_LEADERBOARD_SCOPES.map((item) => (
          <div
            key={item.key}
            className={`farm-pill ${scope === item.key ? 'farm-pill-green' : ''}`}
            style={{ cursor: 'pointer' }}
            onClick={() => setScope(item.key)}
          >
            {item.icon} {t(item.label)}
          </div>
        ))}
      </div>

      <div style={{ display: 'flex', gap: 6, marginBottom: 14, flexWrap: 'wrap' }}>
        {FARM_LEADERBOARD_PERIODS.map((item) => (
          <div
            key={item.key}
            className={`farm-pill ${period === item.key ? 'farm-pill-amber' : ''}`}
            style={{ cursor: 'pointer' }}
            onClick={() => setPeriod(item.key)}
          >
            {item.icon} {t(item.label)}
          </div>
        ))}
      </div>

      <div className='farm-card' style={{ marginBottom: 14 }}>
        <div className='farm-section-title'>🏅 {t('勋章墙')}</div>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 24 }}><Spin /></div>
        ) : honorEntries.length > 0 ? (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 10 }}>
            {honorEntries.map((item) => (
              <div key={`honor-${item.boardType}`} className='farm-row' style={{ marginBottom: 0, gap: 10 }}>
                <span style={{ fontSize: 22 }}>{item.reward?.emoji || '🏅'}</span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontWeight: 700 }}>{t(item.reward?.title || '上榜勋章')}</div>
                  <div style={{ color: 'var(--semi-color-text-2)', fontSize: 12 }}>
                    {item.boardIcon} {t(item.boardLabel)} · #{item.my_rank}
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <Empty description={t('当前维度暂未获得前三勋章')} />
        )}
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>
      ) : (
        <div className='farm-card'>
          <div className='farm-section-title'>📊 {t('榜单成绩')}</div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: 10 }}>
            {rankSummaries.map((item) => {
              const reward = getFarmLeaderboardReward(item.my_rank);
              return (
                <div key={item.boardType} className='farm-row' style={{ marginBottom: 0, flexDirection: 'column', alignItems: 'stretch', gap: 8 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                    <Text strong>{item.boardIcon} {t(item.boardLabel)}</Text>
                    {reward && (
                      <div className='farm-pill farm-pill-amber' style={{ marginLeft: 'auto' }}>
                        {reward.emoji} {t(reward.shortTitle)}
                      </div>
                    )}
                  </div>
                  <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                    <div className={`farm-pill ${item.my_rank > 0 ? 'farm-pill-cyan' : ''}`}>
                      {item.my_rank > 0 ? `#${item.my_rank}` : t('未上榜')}
                    </div>
                    <div className='farm-pill farm-pill-blue'>
                      {t('我的数值')}: {formatFarmLeaderboardValue(item.boardType, item.my_value)}
                    </div>
                    {item.gap_to_prev > 0 && (
                      <div className='farm-pill farm-pill-amber'>
                        {t('距上一名')}: {formatFarmLeaderboardValue(item.boardType, item.gap_to_prev)}
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
            {rankSummaries.length === 0 && (
              <div style={{ gridColumn: '1 / -1' }}>
                <Empty description={t('暂无数据')} />
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default ProfilePage;
