import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { RefreshCw } from 'lucide-react';
import { API, showError } from './utils';

const { Text } = Typography;

const actionEmojis = {
  plant: '🌱', harvest: '🌾', shop: '🏪', steal: '🕵️',
  buy_plot: '🏗️', buy_dog: '🐶', upgrade_soil: '⬆️',
  ranch_buy: '🐄', ranch_feed: '🌾', ranch_water: '💧',
  ranch_sell: '🔪', ranch_clean: '🧹',
  fish: '🎣', fish_sell: '💰',
  craft: '🏭', craft_sell: '📥',
  task: '📝', achieve: '🏆',
  levelup: '⬆️',
};

const formatTime = (ts) => {
  if (!ts) return '';
  const d = new Date(ts * 1000);
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  const hh = String(d.getHours()).padStart(2, '0');
  const mi = String(d.getMinutes()).padStart(2, '0');
  return `${mm}-${dd} ${hh}:${mi}`;
};

const LogsPage = ({ t }) => {
  const [logs, setLogs] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [logsLoading, setLogsLoading] = useState(false);

  const loadLogs = useCallback(async (p = 1) => {
    setLogsLoading(true);
    try {
      const { data: res } = await API.get(`/api/farm/logs?page=${p}&page_size=20`);
      if (res.success) {
        setLogs(res.data.logs || []);
        setTotal(res.data.total || 0);
        setPage(p);
      }
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setLogsLoading(false);
    }
  }, [t]);

  useEffect(() => { loadLogs(1); }, [loadLogs]);

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
        <div className='farm-section-title' style={{ marginBottom: 0 }}>{t('消费记录')} ({total})</div>
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={() => loadLogs(1)} loading={logsLoading} className='farm-btn' />
      </div>

      {logsLoading && logs.length === 0 ? (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>
      ) : logs.length === 0 ? (
        <div className='farm-card' style={{ textAlign: 'center', padding: 30 }}>
          <Text type='tertiary'>{t('暂无消费记录')}</Text>
        </div>
      ) : (
        <>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {logs.map((log) => (
              <div key={log.id} className='farm-row' style={{ flexWrap: 'wrap' }}>
                <span style={{ fontSize: 18, width: 24, textAlign: 'center', flexShrink: 0 }}>
                  {actionEmojis[log.action] || '📋'}
                </span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                    <Tag size='small' color='grey' style={{ borderRadius: 4 }}>{log.action_label}</Tag>
                    <Text size='small' style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {log.detail}
                    </Text>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginLeft: 'auto' }}>
                  <Text strong size='small' style={{ color: log.amount >= 0 ? 'var(--farm-leaf)' : 'var(--farm-danger)' }}>
                    {log.amount >= 0 ? '+' : ''}{log.amount.toFixed(2)}
                  </Text>
                  <Text type='tertiary' size='small'>
                    {formatTime(log.created_at)}
                  </Text>
                </div>
              </div>
            ))}
          </div>
          {total > 20 && (
            <div style={{ display: 'flex', justifyContent: 'center', gap: 8, marginTop: 14 }}>
              <Button size='small' disabled={page <= 1} onClick={() => loadLogs(page - 1)} className='farm-btn'>{t('上一页')}</Button>
              <Text type='tertiary' size='small' style={{ lineHeight: '32px' }}>{page}/{Math.ceil(total / 20)}</Text>
              <Button size='small' disabled={page >= Math.ceil(total / 20)} onClick={() => loadLogs(page + 1)} className='farm-btn'>{t('下一页')}</Button>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default LogsPage;
