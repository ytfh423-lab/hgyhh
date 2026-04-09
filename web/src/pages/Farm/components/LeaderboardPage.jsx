import React, { useCallback, useEffect, useState } from 'react';
import { Empty, Spin, Typography } from '@douyinfe/semi-ui';
import { API, formatBalance } from './utils';

const { Text } = Typography;

const formatBoardValue = (boardType, value) => {
  if (value == null) {
    return boardType === 'balance' || boardType === 'steal' ? formatBalance(0) : '0';
  }
  if (boardType === 'balance' || boardType === 'steal') {
    return formatBalance(value);
  }
  return `${Math.round(value)}`;
};

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

  const types = [
    { key: 'balance', label: '💰 ' + t('资产') },
    { key: 'level', label: '⭐ ' + t('等级') },
    { key: 'harvest', label: '🌾 ' + t('收获') },
    { key: 'prestige', label: '🔄 ' + t('转生') },
    { key: 'steal', label: '🕵️ ' + t('偷菜') },
  ];
  const scopes = [
    { key: 'global', label: '🌍 ' + t('全服') },
    { key: 'friends', label: '👫 ' + t('好友') },
  ];
  const periods = [
    { key: 'all', label: '🏆 ' + t('总榜') },
    { key: 'weekly', label: '📅 ' + t('周榜') },
  ];

  const medals = ['🥇', '🥈', '🥉'];

  return (
    <div>
      <div style={{ display: 'flex', gap: 6, marginBottom: 14, flexWrap: 'wrap' }}>
        {types.map(tp => (
          <div key={tp.key}
            className={`farm-pill ${boardType === tp.key ? 'farm-pill-blue' : ''}`}
            style={{ cursor: 'pointer' }} onClick={() => setBoardType(tp.key)}>
            {tp.label}
          </div>
        ))}
      </div>

      <div style={{ display: 'flex', gap: 6, marginBottom: 10, flexWrap: 'wrap' }}>
        {scopes.map(tp => (
          <div key={tp.key}
            className={`farm-pill ${scope === tp.key ? 'farm-pill-green' : ''}`}
            style={{ cursor: 'pointer' }} onClick={() => setScope(tp.key)}>
            {tp.label}
          </div>
        ))}
      </div>

      <div style={{ display: 'flex', gap: 6, marginBottom: 14, flexWrap: 'wrap' }}>
        {periods.map(tp => (
          <div key={tp.key}
            className={`farm-pill ${period === tp.key ? 'farm-pill-amber' : ''}`}
            style={{ cursor: 'pointer' }} onClick={() => setPeriod(tp.key)}>
            {tp.label}
          </div>
        ))}
      </div>

      {data?.title && (
        <div style={{ marginBottom: 10 }}>
          <div className='farm-pill farm-pill-blue'>{t('当前榜单')}: {t(data.title)}</div>
        </div>
      )}

      {data?.my_rank && (
        <div style={{ display: 'flex', gap: 8, marginBottom: 14, flexWrap: 'wrap' }}>
          <div className='farm-pill farm-pill-cyan'>📊 {t('我的排名')}: #{data.my_rank}</div>
          <div className='farm-pill farm-pill-blue'>{t('我的')}{t(data?.label || '数值')}: {formatBoardValue(boardType, data.my_value)}</div>
          {data.gap_to_prev > 0 && (
            <div className='farm-pill farm-pill-amber'>{t('距上一名还差')}: {formatBoardValue(boardType, data.gap_to_prev)}</div>
          )}
          {data.gap_to_prev <= 0 && data.my_rank === 1 && (
            <div className='farm-pill farm-pill-green'>{t('你目前就是第一名')}</div>
          )}
        </div>
      )}

      {data?.nearby_items?.length > 0 && (
        <div className='farm-card' style={{ marginBottom: 14 }}>
          <div className='farm-section-title'>🎯 {t('附近排名')}</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {data.nearby_items.map((item) => (
              <div key={`nearby-${item.rank}`} className='farm-row' style={{
                marginBottom: 0,
                borderColor: item.is_me ? 'var(--semi-color-primary)' : undefined,
                boxShadow: item.is_me ? 'var(--farm-shadow), var(--farm-glow-blue)' : undefined,
                fontWeight: item.is_me ? 600 : 400,
              }}>
                <span style={{ width: 36, textAlign: 'center' }}>#{item.rank}</span>
                <Text style={{ flex: 1 }}>{item.name}</Text>
                <Text strong>{formatBoardValue(boardType, item.value)}</Text>
              </div>
            ))}
          </div>
        </div>
      )}

      {loading ? <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div> : (
        <div className='farm-card'>
          {data?.title && (
            <div className='farm-section-title' style={{ marginBottom: 10 }}>🏅 {t(data.title)}</div>
          )}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {(data?.items || []).map(item => (
              <div key={item.rank} className='farm-row' style={{
                marginBottom: 0,
                borderColor: item.is_me ? 'var(--semi-color-primary)' : undefined,
                boxShadow: item.is_me ? 'var(--farm-shadow), var(--farm-glow-blue)' : undefined,
                fontWeight: item.is_me ? 600 : 400,
              }}>
                <span style={{ fontSize: 18, width: 30, textAlign: 'center' }}>
                  {item.rank <= 3 ? medals[item.rank - 1] : `#${item.rank}`}
                </span>
                <Text style={{ flex: 1 }}>{item.name}</Text>
                <Text strong>{formatBoardValue(boardType, item.value)}</Text>
              </div>
            ))}
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
