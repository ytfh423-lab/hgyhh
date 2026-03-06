import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './utils';

const { Text } = Typography;

const AutomationPage = ({ loadFarm, t }) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/automation');
      if (res.success) setData(res.data);
    } catch (err) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const buy = async (type) => {
    try {
      const { data: res } = await API.post('/api/farm/automation/buy', { type });
      if (res.success) { showSuccess(res.message); load(); loadFarm(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
  };

  if (loading && !data) return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  if (!data) return null;

  return (
    <div className='farm-card'>
      <div className='farm-section-title'>⚡ {t('自动化设施')}</div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {data.map(item => (
          <div key={item.type} className='farm-row' style={{
            background: item.installed ? 'rgba(34,197,94,0.08)' : undefined,
            marginBottom: 0,
          }}>
            <span style={{ fontSize: 28 }}>{item.emoji}</span>
            <div style={{ flex: 1 }}>
              <Text strong>{item.name}</Text>
              <Text type='tertiary' size='small' style={{ display: 'block' }}>{item.desc}</Text>
            </div>
            {item.installed ? (
              <Tag size='large' color='green'>✅ {t('已安装')}</Tag>
            ) : (
              <Button theme='solid' onClick={() => buy(item.type)} className='farm-btn'>
                ${item.price.toFixed(2)}
              </Button>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default AutomationPage;
