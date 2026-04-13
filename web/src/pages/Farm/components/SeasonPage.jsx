import React, { useCallback, useEffect, useState } from 'react';
import { Spin } from '@douyinfe/semi-ui';
import { API, showError } from './utils';

const TIER_FALLBACK_EMOJI = { bronze: '🥉', silver: '🥈', gold: '🥇', platinum: '💎', diamond: '💠', rich: '👑' };

const SeasonPage = ({ t }) => {
  const [overview, setOverview] = useState(null);
  const [leaderboard, setLeaderboard] = useState(null);
  const [tiers, setTiers] = useState([]);
  const [history, setHistory] = useState([]);
  const [pointsLogs, setPointsLogs] = useState([]);
  const [tab, setTab] = useState('overview');
  const [loading, setLoading] = useState(true);

  const loadOverview = useCallback(async () => {
    try {
      const [ovRes, tierRes] = await Promise.all([
        API.get('/api/farm/season/overview'),
        API.get('/api/farm/season/tiers'),
      ]);
      if (ovRes.data?.success) setOverview(ovRes.data.data);
      if (tierRes.data?.success) setTiers(tierRes.data.data || []);
    } catch (e) { showError(t('加载赛季数据失败')); }
    finally { setLoading(false); }
  }, [t]);

  const loadLeaderboard = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/season/leaderboard');
      if (res.success) setLeaderboard(res.data);
    } catch (e) { /* ignore */ }
  }, []);

  const loadHistory = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/season/history');
      if (res.success) setHistory(res.data || []);
    } catch (e) { /* ignore */ }
  }, []);

  const loadPointsLogs = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/season/points-logs');
      if (res.success) setPointsLogs(res.data || []);
    } catch (e) { /* ignore */ }
  }, []);

  useEffect(() => { loadOverview(); }, [loadOverview]);

  useEffect(() => {
    if (tab === 'leaderboard') loadLeaderboard();
    else if (tab === 'history') loadHistory();
    else if (tab === 'logs') loadPointsLogs();
  }, [tab, loadLeaderboard, loadHistory, loadPointsLogs]);

  if (loading) return <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>;

  const tabs = [
    { key: 'overview', label: '总览', emoji: '🏟️' },
    { key: 'leaderboard', label: '冲榜', emoji: '🏆' },
    { key: 'tiers', label: '段位', emoji: '🎖️' },
    { key: 'history', label: '历史', emoji: '📜' },
    { key: 'logs', label: '积分明细', emoji: '📊' },
  ];

  const tierEmoji = (key) => {
    const tier = tiers.find(t => t.tier_key === key);
    return tier?.emoji || TIER_FALLBACK_EMOJI[key] || '🏅';
  };
  const tierName = (key) => {
    const tier = tiers.find(t => t.tier_key === key);
    return tier?.tier_name || key || '无';
  };
  const tierColor = (key) => {
    const tier = tiers.find(t => t.tier_key === key);
    return tier?.color || '#888';
  };
  const tierMinPoints = (key) => {
    const tier = tiers.find(t => t.tier_key === key);
    return tier?.min_points || 0;
  };
  const nextTierProgress = () => {
    if (!overview?.next_tier?.key) return 0;
    const currentMin = tierMinPoints(overview.current_tier?.key);
    const nextMin = overview.next_tier?.min_points || currentMin;
    const range = Math.max(1, nextMin - currentMin);
    const progress = ((overview.points - currentMin) / range) * 100;
    return Math.min(100, Math.max(0, progress));
  };

  const statusLabel = (s) => {
    switch (s) {
      case 1: return '🔥 冲榜期';
      case 2: return '😴 休赛期';
      case 3: return '✅ 已结束';
      default: return '⏳ 未开始';
    }
  };

  return (
    <div>
      {/* Tab bar */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 14, flexWrap: 'wrap' }}>
        {tabs.map(tb => (
          <div
            key={tb.key}
            className={`farm-pill ${tab === tb.key ? 'farm-pill-blue' : ''}`}
            style={{ cursor: 'pointer' }}
            onClick={() => setTab(tb.key)}
          >
            {tb.emoji} {t(tb.label)}
          </div>
        ))}
      </div>

      {/* Overview */}
      {tab === 'overview' && (
        overview?.active ? (
          <div className="farm-card" style={{ padding: 20 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <h3 style={{ margin: 0 }}>{overview.season_code}</h3>
              <span className="farm-pill" style={{ fontSize: 13 }}>{statusLabel(overview.status)}</span>
            </div>

            {/* Tier badge */}
            <div style={{ textAlign: 'center', margin: '20px 0' }}>
              <div style={{ fontSize: 48 }}>{tierEmoji(overview.current_tier?.key)}</div>
              <div style={{
                fontSize: 20, fontWeight: 700,
                color: tierColor(overview.current_tier?.key),
                marginTop: 4
              }}>
                {overview.current_tier?.name || '青铜'}
              </div>
              <div style={{ color: '#888', marginTop: 4, fontSize: 13 }}>
                赛季积分: <strong>{overview.points}</strong> · 排名: <strong>#{overview.rank}</strong>
              </div>
            </div>

            {/* Progress to next tier */}
            {overview.next_tier?.key && (
              <div style={{ marginBottom: 16 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, color: '#666', marginBottom: 4 }}>
                  <span>{tierEmoji(overview.current_tier?.key)} {overview.current_tier?.name}</span>
                  <span>{tierEmoji(overview.next_tier?.key)} {overview.next_tier?.name}</span>
                </div>
                <div style={{ height: 8, background: '#eee', borderRadius: 4, overflow: 'hidden' }}>
                  <div style={{
                    height: '100%',
                    borderRadius: 4,
                    background: `linear-gradient(90deg, ${tierColor(overview.current_tier?.key)}, ${tierColor(overview.next_tier?.key)})`,
                    width: `${nextTierProgress()}%`,
                    transition: 'width 0.5s',
                  }} />
                </div>
                <div style={{ fontSize: 12, color: '#999', marginTop: 2, textAlign: 'right' }}>
                  还需 {overview.next_tier?.points_needed} 积分
                </div>
              </div>
            )}

            {/* Stats */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10, marginTop: 12 }}>
              <div className="farm-card" style={{ textAlign: 'center', padding: '10px 6px' }}>
                <div style={{ fontSize: 12, color: '#888' }}>剩余天数</div>
                <div style={{ fontSize: 20, fontWeight: 700 }}>{overview.days_left}</div>
              </div>
              <div className="farm-card" style={{ textAlign: 'center', padding: '10px 6px' }}>
                <div style={{ fontSize: 12, color: '#888' }}>参与人数</div>
                <div style={{ fontSize: 20, fontWeight: 700 }}>{overview.player_count}</div>
              </div>
              <div className="farm-card" style={{ textAlign: 'center', padding: '10px 6px' }}>
                <div style={{ fontSize: 12, color: '#888' }}>积分倍率</div>
                <div style={{ fontSize: 20, fontWeight: 700 }}>{overview.multiplier}%</div>
              </div>
            </div>

            {/* Inherited tier info */}
            {overview.inherited_from && (
              <div style={{ marginTop: 14, padding: '8px 12px', background: '#f6f0e8', borderRadius: 8, fontSize: 13 }}>
                历史最高段位继承: {tierEmoji(overview.inherited_from)} <strong>{tierName(overview.inherited_from)}</strong>
              </div>
            )}
          </div>
        ) : (
          <div className="farm-card" style={{ padding: 30, textAlign: 'center', color: '#888' }}>
            <div style={{ fontSize: 48, marginBottom: 12 }}>🏟️</div>
            <div>{t('当前没有进行中的赛季')}</div>
          </div>
        )
      )}

      {/* Leaderboard */}
      {tab === 'leaderboard' && (
        <div className="farm-card" style={{ padding: 16 }}>
          <h4 style={{ margin: '0 0 12px' }}>🏆 {t('赛季冲榜排名')}</h4>
          {leaderboard?.entries?.length > 0 ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {leaderboard.entries.map((entry, i) => (
                <div key={i} style={{
                  display: 'flex', alignItems: 'center', gap: 10,
                  padding: '8px 12px', borderRadius: 8,
                  background: i < 3 ? '#fffbe6' : (i % 2 === 0 ? '#fafafa' : '#fff'),
                }}>
                  <span style={{ width: 28, fontWeight: 700, fontSize: i < 3 ? 18 : 14 }}>
                    {i === 0 ? '🥇' : i === 1 ? '🥈' : i === 2 ? '🥉' : `${entry.rank}`}
                  </span>
                  <span style={{ fontSize: 16 }}>{tierEmoji(entry.tier_key)}</span>
                  <span style={{ flex: 1, fontWeight: i < 3 ? 600 : 400 }}>{entry.username}</span>
                  <span style={{ fontWeight: 600, color: '#c97a30' }}>{entry.points} 分</span>
                </div>
              ))}
            </div>
          ) : (
            <div style={{ textAlign: 'center', color: '#999', padding: 20 }}>{t('暂无数据')}</div>
          )}
        </div>
      )}

      {/* Tiers */}
      {tab === 'tiers' && (
        <div className="farm-card" style={{ padding: 16 }}>
          <h4 style={{ margin: '0 0 12px' }}>🎖️ {t('段位体系')}</h4>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {tiers.map(tier => (
              <div key={tier.tier_key} style={{
                display: 'flex', alignItems: 'center', gap: 12,
                padding: '12px 14px', borderRadius: 10,
                border: `2px solid ${tier.color || '#eee'}`,
                background: overview?.current_tier?.key === tier.tier_key ? `${tier.color}15` : '#fff',
              }}>
                <span style={{ fontSize: 28 }}>{tier.emoji}</span>
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 700, color: tier.color }}>{tier.tier_name}</div>
                  <div style={{ fontSize: 12, color: '#888' }}>
                    {tier.min_points} 积分以上 · 继承资金 ${(tier.initial_balance / 500000).toFixed(2)}
                  </div>
                </div>
                {overview?.current_tier?.key === tier.tier_key && (
                  <span className="farm-pill farm-pill-blue" style={{ fontSize: 11 }}>当前</span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* History */}
      {tab === 'history' && (
        <div className="farm-card" style={{ padding: 16 }}>
          <h4 style={{ margin: '0 0 12px' }}>📜 {t('赛季历史')}</h4>
          {history.length > 0 ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {history.map((h, i) => (
                <div key={i} style={{
                  display: 'flex', alignItems: 'center', gap: 10,
                  padding: '8px 12px', borderRadius: 8, background: '#fafafa',
                }}>
                  <span style={{ fontSize: 20 }}>{tierEmoji(h.final_tier_key)}</span>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontWeight: 600 }}>赛季 #{h.season_id}</div>
                    <div style={{ fontSize: 12, color: '#888' }}>
                      {tierName(h.final_tier_key)} · {h.final_points} 分 · 排名 #{h.final_rank}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div style={{ textAlign: 'center', color: '#999', padding: 20 }}>{t('暂无赛季历史')}</div>
          )}
        </div>
      )}

      {/* Points Logs */}
      {tab === 'logs' && (
        <div className="farm-card" style={{ padding: 16 }}>
          <h4 style={{ margin: '0 0 12px' }}>📊 {t('积分明细')}</h4>
          {pointsLogs.length > 0 ? (
            <div style={{ maxHeight: 400, overflowY: 'auto' }}>
              {pointsLogs.map((log, i) => (
                <div key={i} style={{
                  display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                  padding: '6px 10px', borderBottom: '1px solid #f0f0f0', fontSize: 13,
                }}>
                  <div>
                    <span style={{ color: '#555' }}>{log.detail || log.action}</span>
                  </div>
                  <span style={{ fontWeight: 600, color: log.points > 0 ? '#52c41a' : '#f5222d' }}>
                    {log.points > 0 ? '+' : ''}{log.points}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <div style={{ textAlign: 'center', color: '#999', padding: 20 }}>{t('暂无积分记录')}</div>
          )}
        </div>
      )}
    </div>
  );
};

export default SeasonPage;
