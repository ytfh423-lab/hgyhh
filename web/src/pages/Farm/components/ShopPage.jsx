import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Table, Tag, Typography } from '@douyinfe/semi-ui';
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
      {/* Balance */}
      <div className='farm-card' style={{ padding: '8px 16px' }}>
        <Text type='tertiary' size='small'>💰 {t('余额')}: </Text>
        <Text strong>{formatBalance(farmData?.balance)}</Text>
      </div>

      {/* Seeds */}
      <div className='farm-card'>
        <div className='farm-section-title'>🌱 {t('种子目录')}</div>
        <Table dataSource={crops} pagination={false} size='small' columns={[
          { title: t('作物'), dataIndex: 'name', render: (_, r) => <span>{r.emoji} {r.name}</span>, width: 100 },
          { title: t('价格'), dataIndex: 'seed_cost', render: v => `$${v?.toFixed(2)}`, width: 80 },
          { title: t('时间'), dataIndex: 'grow_secs', render: v => formatDuration(v), width: 80 },
          { title: t('产量'), dataIndex: 'max_yield', render: (v, r) => `1~${v}×$${r.unit_price?.toFixed(2)}`, width: 120 },
          { title: t('最高'), dataIndex: 'max_value', render: v => <Tag size='small' color='green'>${v?.toFixed(2)}</Tag>, width: 80 },
        ]} />
      </div>

      {/* Items */}
      <div className='farm-card'>
        <div className='farm-section-title'>📦 {t('道具')}</div>
        {(shopData?.items || []).map((item) => (
          <div key={item.key} className='farm-row'>
            <span style={{ fontSize: 18 }}>{item.emoji}</span>
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text strong size='small'>{item.name}</Text>
              <Text type='tertiary' size='small'>{item.desc}</Text>
            </div>
            <div style={{ display: 'flex', gap: 4 }}>
              {[1, 5, 10].map(qty => (
                <Button key={qty} size='small' theme={qty === 1 ? 'solid' : 'light'}
                  onClick={() => handleBuyItem(item.key, qty)}
                  loading={actionLoading} className='farm-btn' style={{ minWidth: 60 }}>
                  ×{qty} ${(item.cost * qty)?.toFixed(2)}
                </Button>
              ))}
            </div>
          </div>
        ))}
      </div>

      {/* Dog */}
      {shopData && !shopData.has_dog && (
        <div className='farm-card'>
          <div className='farm-row'>
            <span style={{ fontSize: 18 }}>🐶</span>
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text strong size='small'>{t('小狗')}</Text>
              <Text type='tertiary' size='small'>{t('长大后拦截偷菜')}</Text>
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
