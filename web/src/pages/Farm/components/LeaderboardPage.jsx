import React, { useCallback, useEffect, useState } from 'react';
import { Empty, Spin, Typography } from '@douyinfe/semi-ui';
import { API } from './utils';
import {
  FARM_LEADERBOARD_COHORTS,
  FARM_LEADERBOARD_PERIODS,
  FARM_LEADERBOARD_SCOPES,
  FARM_LEADERBOARD_TYPES,
  formatFarmLeaderboardValue,
} from './leaderboardUtils';

const { Text } = Typography;

const LeaderboardPage = ({ t }) => {
  const [data, setData] = useState(null);
  const [boardType, setBoardType] = useState('balance');
  const [scope, setScope] = useState('global');
  const [period, setPeriod] = useState('all');
  // cohort 初始为空串，首次请求让后端自动决定（普通玩家=自己所属，admin=all）。
  // 后端响应里带 can_switch_cohort + cohort 字段，前端据此决定是否展示切换 pills。
  const [cohort, setCohort] = useState('');
  const [loading, setLoading] = useState(false);

  const load = useCallback(async (type, boardScope, boardPeriod, boardCohort) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ type, scope: boardScope, period: boardPeriod });
      if (boardCohort) params.append('cohort', boardCohort);
      const { data: res } = await API.get(`/api/farm/leaderboard?${params.toString()}`);
      if (res.success) setData(res.data);
    } catch (err) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(boardType, scope, period, cohort); }, [load, boardType, scope, period, cohort]);

  // 管理员专属 cohort 切换；普通玩家后端会无视 query 强制绑定到自己所属的 cohort，
  // 所以即使前端误传也不会越权看到另一边。
  const canSwitchCohort = !!data?.can_switch_cohort;
  const effectiveCohort = data?.cohort || cohort || '';

  const medals = ['🥇', '🥈', '🥉'];

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

      {canSwitchCohort && (
        <div style={{ display: 'flex', gap: 6, marginBottom: 14, flexWrap: 'wrap', alignItems: 'center' }}>
          <Text type='tertiary' size='small' style={{ marginRight: 4 }}>
            {t('管理员视角')}:
          </Text>
          {FARM_LEADERBOARD_COHORTS.map(tp => (
            <div key={tp.key}
              className={`farm-pill ${effectiveCohort === tp.key ? 'farm-pill-cyan' : ''}`}
              style={{ cursor: 'pointer' }} onClick={() => setCohort(tp.key)}>
              {tp.icon} {t(tp.label)}
            </div>
          ))}
        </div>
      )}

      {data?.title && (
        <div style={{ display: 'flex', gap: 8, marginBottom: 10, flexWrap: 'wrap' }}>
          <div className='farm-pill farm-pill-blue'>{t('当前榜单')}: {data.title}</div>
          {data?.cohort_label && (
            <div className='farm-pill farm-pill-cyan'>
              {data.cohort === 'new' ? '🌱' : data.cohort === 'old' ? '🪵' : '👥'} {t(data.cohort_label)}
            </div>
          )}
          {data?.group_label && (
            <div className='farm-pill farm-pill-cyan'>🏷️ {data.group_label} {data.group_range_label ? `· ${data.group_range_label}` : ''}</div>
          )}
          {typeof data?.total_players === 'number' && (
            <div className='farm-pill farm-pill-green'>👥 {t('上榜人数')}: {data.total_players}</div>
          )}
        </div>
      )}

      <div className='farm-card' style={{ marginBottom: 14 }}>
        <div className='farm-section-title'>🎁 {t('榜单荣誉')}</div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {(data?.reward_bands || []).map((band) => (
            <div key={`reward-${band.key}-${band.start_rank}`} className='farm-pill farm-pill-amber'>
              {band.emoji} #{band.start_rank}-#{band.end_rank} {t(band.title)}
            </div>
          ))}
          {(!data?.reward_bands || data.reward_bands.length === 0) && (
            <div className='farm-pill'>{t('当前分组暂无奖励带数据')}</div>
          )}
        </div>
      </div>

      {data?.my_rank && (
        <div style={{ display: 'flex', gap: 8, marginBottom: 14, flexWrap: 'wrap' }}>
          <div className='farm-pill farm-pill-cyan'>📊 {t('我的排名')}: #{data.my_rank}</div>
          <div className='farm-pill farm-pill-blue'>{t('我的')}{t(data?.label || '数值')}: {formatFarmLeaderboardValue(boardType, data.my_value, data?.value_kind)}</div>
          {data?.my_reward && (
            <div className='farm-pill farm-pill-amber'>{data.my_reward.emoji} {t(data.my_reward.title)}</div>
          )}
          {data.gap_to_prev > 0 && (
            <div className='farm-pill farm-pill-amber'>{t('距上一名还差')}: {formatFarmLeaderboardValue(boardType, data.gap_to_prev, data?.value_kind)}</div>
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
                    {item.reward && (
                      <div className='farm-pill farm-pill-amber'>
                        {item.reward.emoji} {t(item.reward.short_title || item.reward.shortTitle || item.reward.title)}
                      </div>
                    )}
                  </div>
                  <Text strong>{formatFarmLeaderboardValue(boardType, item.value, data?.value_kind)}</Text>
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
