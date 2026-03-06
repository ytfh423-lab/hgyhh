import React, { useCallback, useEffect, useState } from 'react';
import { Empty, Spin, Typography } from '@douyinfe/semi-ui';
import { API, formatBalance } from './utils';

const { Text } = Typography;

const LeaderboardPage = ({ t }) => {
  const [data, setData] = useState(null);
  const [boardType, setBoardType] = useState('balance');
  const [loading, setLoading] = useState(false);

  const load = useCallback(async (type) => {
    setLoading(true);
    try {
      const { data: res } = await API.get(`/api/farm/leaderboard?type=${type}`);
      if (res.success) setData(res.data);
    } catch (err) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(boardType); }, [load, boardType]);

  const types = [
    { key: 'balance', label: '💰 ' + t('资产') },
    { key: 'level', label: '⭐ ' + t('等级') },
    { key: 'harvest', label: '🌾 ' + t('收获') },
    { key: 'prestige', label: '🔄 ' + t('转生') },
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

      {data?.my_rank && (
        <div className='farm-pill farm-pill-cyan' style={{ marginBottom: 14 }}>📊 {t('我的排名')}: #{data.my_rank}</div>
      )}

      {loading ? <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div> : (
        <div className='farm-card'>
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
                <Text strong>{boardType === 'balance' ? formatBalance(item.value) : item.value}</Text>
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
