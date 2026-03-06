import React, { useCallback, useEffect, useState } from 'react';
import { Button, Empty, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { RefreshCw } from 'lucide-react';
import { API } from './utils';

const { Text } = Typography;

const StealPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [targets, setTargets] = useState([]);
  const [stealLoading, setStealLoading] = useState(true);
  const [stealResults, setStealResults] = useState([]);

  const loadTargets = useCallback(async () => {
    setStealLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/steal/targets');
      if (res.success) setTargets(res.data || []);
    } catch (err) { /* ignore */ }
    finally { setStealLoading(false); }
  }, []);

  useEffect(() => { loadTargets(); }, [loadTargets]);

  const handleSteal = async (victimId) => {
    const res = await doAction('/api/farm/steal', { victim_id: victimId });
    if (res) {
      setStealResults(prev => [{
        time: new Date().toLocaleTimeString(),
        message: res.message,
        data: res.data,
      }, ...prev]);
      loadTargets();
      loadFarm();
    }
  };

  if (stealLoading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }

  return (
    <div>
      {/* Results */}
      {stealResults.length > 0 && (
        <div className='farm-card'>
          <div className='farm-section-title'>📜 {t('偷菜记录')}</div>
          {stealResults.map((r, i) => (
            <div key={i} className='farm-row' style={{ marginBottom: 4 }}>
              <Text size='small'><Text type='tertiary' size='small'>{r.time}</Text> {r.message}</Text>
              {r.data && <Tag size='small' color='green'>${r.data.value?.toFixed(2)}</Tag>}
            </div>
          ))}
        </div>
      )}

      {/* Targets */}
      <div className='farm-card'>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
          <div className='farm-section-title' style={{ marginBottom: 0 }}>🕵️ {t('可偷菜的农场')}</div>
          <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadTargets} loading={stealLoading} className='farm-btn' />
        </div>

        {targets.length === 0 ? (
          <Empty description={t('暂时没有可偷的菜地')} style={{ padding: '20px 0' }} />
        ) : (
          <div>
            {targets.map((target) => (
              <div key={target.id} className='farm-row'>
                <Text strong size='small'>👤 {target.label}</Text>
                <Tag size='small' color='green'>{target.count}{t('块')}</Tag>
                <div style={{ flex: 1 }} />
                <Button size='small' type='warning' theme='solid' onClick={() => handleSteal(target.id)}
                  loading={actionLoading} className='farm-btn'>
                  {t('偷菜')}
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default StealPage;
