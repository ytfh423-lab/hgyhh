import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { API, formatBalance, formatDuration } from './utils';

const { Text } = Typography;

const ShopPage = ({ farmData, actionLoading, doAction, loadFarm, t }) => {
  const [shopData, setShopData] = useState(null);
  const [shopLoading, setShopLoading] = useState(true);
  const [crops, setCrops] = useState([]);

  const loadShop = useCallback(async () => {
    setShopLoading(true);
    try {
      const [shopRes, cropRes] = await Promise.all([
        API.get('/api/farm/shop'),
        API.get('/api/farm/crops'),
      ]);
      if (shopRes.data.success) setShopData(shopRes.data.data);
      if (cropRes.data.success) setCrops(cropRes.data.data || []);
    } catch (err) { /* ignore */ }
    finally { setShopLoading(false); }
  }, []);

  useEffect(() => { loadShop(); }, [loadShop]);

  const handleBuyItem = async (key, quantity = 1) => {
    const res = await doAction('/api/farm/buy', { item_key: key, quantity });
    if (res) { loadShop(); loadFarm(); }
  };

  const handleBuyDog = async () => {
    const res = await doAction('/api/farm/buydog', {});
    if (res) { loadShop(); loadFarm(); }
  };

  if (shopLoading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }

  return (
    <div>
      <div className='farm-card' style={{ padding: '8px 16px' }}>
        <Text type='tertiary' size='small'>💰 {t('余额')}: </Text>
        <Text strong>{formatBalance(farmData?.balance)}</Text>
      </div>

      <div className='farm-card'>
        <div className='farm-section-title'>🌱 {t('种子目录')}</div>
        <div className='farm-grid farm-grid-2'>
          {crops.map((cr) => (
            <div key={cr.key || cr.name} className='farm-card-flat' style={{ padding: '12px 14px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                <span style={{ fontSize: 22 }}>{cr.emoji}</span>
                <Text strong size='small'>{cr.name}</Text>
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginBottom: 4 }}>
                <span className='farm-pill' style={{ fontSize: 11 }}>💰 ${cr.seed_cost?.toFixed(2)}</span>
                <span className='farm-pill' style={{ fontSize: 11 }}>⏱ {formatDuration(cr.grow_secs)}</span>
                <span className='farm-pill' style={{ fontSize: 11 }}>📦 1~{cr.max_yield}×${cr.unit_price?.toFixed(2)}</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                <Tag size='small' color='green' style={{ borderRadius: 12 }}>{t('最高')} ${cr.max_value?.toFixed(2)}</Tag>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className='farm-card'>
        <div className='farm-section-title'>📦 {t('道具')}</div>
        {(shopData?.items || []).map((item) => (
          <div key={item.key} className='farm-row'>
            <span style={{ fontSize: 18 }}>{item.emoji}</span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <Text strong size='small'>{item.name}</Text>
              <Text type='tertiary' size='small' style={{ display: 'block' }}>{item.desc}</Text>
            </div>
            <div style={{ display: 'flex', gap: 4, flexShrink: 0, flexWrap: 'wrap', justifyContent: 'flex-end' }}>
              {[1, 5, 10].map(qty => (
                <Button key={qty} size='small' theme={qty === 1 ? 'solid' : 'light'}
                  onClick={() => handleBuyItem(item.key, qty)}
                  loading={actionLoading} className='farm-btn' style={{ minWidth: 56 }}>
                  ×{qty} ${(item.cost * qty)?.toFixed(2)}
                </Button>
              ))}
            </div>
          </div>
        ))}
      </div>

      {shopData && !shopData.has_dog && (
        <div className='farm-card'>
          <div className='farm-row'>
            <span style={{ fontSize: 18 }}>🐶</span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <Text strong size='small'>{t('小狗')}</Text>
              <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('长大后拦截偷菜')}</Text>
            </div>
            <Button size='small' theme='solid' onClick={handleBuyDog}
              loading={actionLoading} className='farm-btn'>
              ${shopData.dog_price?.toFixed(2)}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};

export default ShopPage;
