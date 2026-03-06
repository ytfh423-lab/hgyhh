import React, { useCallback, useEffect, useState } from 'react';
import { Button, Empty, Spin, Tag, Banner, Typography } from '@douyinfe/semi-ui';
import { API, seasonNames, seasonEmojis } from './utils';

const { Text } = Typography;

const WarehousePage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [whData, setWhData] = useState(null);
  const [whLoading, setWhLoading] = useState(true);

  const loadWarehouse = useCallback(async () => {
    setWhLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/warehouse');
      if (res.success) setWhData(res.data);
    } catch (err) { /* ignore */ }
    finally { setWhLoading(false); }
  }, []);

  useEffect(() => { loadWarehouse(); }, [loadWarehouse]);

  const handleSell = async (cropKey) => {
    const res = await doAction('/api/farm/warehouse/sell', { crop_key: cropKey });
    if (res) { loadWarehouse(); loadFarm(); }
  };

  const handleSellAll = async () => {
    const res = await doAction('/api/farm/warehouse/sellall', {});
    if (res) { loadWarehouse(); loadFarm(); }
  };

  if (whLoading) return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  if (!whData) return <Empty description={t('加载失败')} />;

  const items = whData.items || [];
  const season = whData.season ?? 0;

  return (
    <div>
      <div className='farm-card'>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 8 }}>
          <Text strong>{seasonEmojis[season]} {t('当前')}: {seasonNames[season]}{t('季')} ({t('剩余')} {whData.days_left} {t('天')})</Text>
          <div className='farm-pill farm-pill-blue'>{t('容量')}: {whData.total}/{whData.max_slots}</div>
        </div>
        <Banner type='info' style={{ borderRadius: 10 }}
          description={t('应季作物价格低，反季价格高。建议应季存入仓库，等反季再出售！')} />
      </div>

      {items.length === 0 ? (
        <Empty description={t('仓库空空如也，收获时选择「收获到仓库」来存储作物')} />
      ) : (
        <div className='farm-card'>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <div className='farm-section-title' style={{ marginBottom: 0 }}>📦 {t('库存')}</div>
            <Button theme='solid' type='warning' size='small' onClick={handleSellAll}
              loading={actionLoading} className='farm-btn'>
              💰 {t('全部出售')}
            </Button>
          </div>
          {items.map((r) => (
            <div key={r.crop_key} className='farm-row'>
              <span style={{ fontSize: 20 }}>{r.emoji}</span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                  <Text strong size='small'>{r.crop_name}</Text>
                  <span className='farm-pill' style={{ fontSize: 11 }}>×{r.quantity}</span>
                  <Tag size='small' color={r.in_season ? 'green' : 'orange'} style={{ borderRadius: 12 }}>
                    {r.in_season ? t('应季') : t('反季')} {r.season_pct}%
                  </Tag>
                </div>
                <Text type='tertiary' size='small'>
                  {t('单价')} ${r.unit_price?.toFixed(2)} · {t('总值')} <Text size='small' strong style={{ color: '#f59e0b' }}>${r.total_value?.toFixed(2)}</Text>
                </Text>
              </div>
              <Button size='small' theme='solid' type='warning' onClick={() => handleSell(r.crop_key)}
                loading={actionLoading} className='farm-btn'>{t('出售')}</Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default WarehousePage;
