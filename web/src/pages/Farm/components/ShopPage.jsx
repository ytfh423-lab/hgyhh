import React, { useCallback, useEffect, useState, useMemo } from 'react';
import { Button, Spin, Typography } from '@douyinfe/semi-ui';
import { API, formatBalance, formatDuration } from './utils';

const { Text } = Typography;

/* ═══════════════════════════════════════════════════════════════
   ShopPage — 赛博贩卖机 Master-Detail 布局
   ═══════════════════════════════════════════════════════════════ */
const ShopPage = ({ farmData, actionLoading, doAction, loadFarm, t }) => {
  const [shopData, setShopData] = useState(null);
  const [shopLoading, setShopLoading] = useState(true);
  const [crops, setCrops] = useState([]);
  const [activeTab, setActiveTab] = useState('seed');
  const [selectedKey, setSelectedKey] = useState(null);
  const [quantity, setQuantity] = useState(1);

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

  const handleBuyItem = async (key, qty) => {
    const res = await doAction('/api/farm/buy', { item_key: key, quantity: qty });
    if (res) { loadShop(); loadFarm(); }
  };

  const handleBuyDog = async () => {
    const res = await doAction('/api/farm/buydog', {});
    if (res) { loadShop(); loadFarm(); }
  };

  // Normalize all items into a unified list with category tags
  const allItems = useMemo(() => {
    const list = [];
    // Seeds
    crops.forEach(cr => list.push({
      key: cr.key || cr.name,
      category: 'seed',
      emoji: cr.emoji,
      name: cr.name,
      price: cr.seed_cost,
      desc: `${formatDuration(cr.grow_secs)} · 1~${cr.max_yield}${t('个')}`,
      isSeed: true,
      grow_secs: cr.grow_secs,
      max_yield: cr.max_yield,
      unit_price: cr.unit_price,
      max_value: cr.max_value,
      maxQty: 50,
    }));
    // Shop items
    (shopData?.items || []).forEach(item => list.push({
      key: item.key,
      category: 'tool',
      emoji: item.emoji,
      name: item.name,
      price: item.cost,
      desc: item.desc,
      isSeed: false,
      maxQty: 99,
    }));
    // Dog (special)
    if (shopData && !shopData.has_dog) {
      list.push({
        key: '__dog__',
        category: 'livestock',
        emoji: '🐶',
        name: t('小狗'),
        price: shopData.dog_price,
        desc: t('长大后拦截偷菜'),
        isDog: true,
        isSeed: false,
        maxQty: 1,
      });
    }
    return list;
  }, [crops, shopData, t]);

  const tabs = [
    { key: 'seed', label: '🌱 ' + t('种子'), count: allItems.filter(i => i.category === 'seed').length },
    { key: 'tool', label: '🔧 ' + t('道具'), count: allItems.filter(i => i.category === 'tool').length },
    { key: 'livestock', label: '🐾 ' + t('牲畜'), count: allItems.filter(i => i.category === 'livestock').length },
  ];

  const filteredItems = allItems.filter(i => i.category === activeTab);
  const selected = allItems.find(i => i.key === selectedKey) || null;

  // Auto-select first item when switching tabs
  useEffect(() => {
    const items = allItems.filter(i => i.category === activeTab);
    if (items.length > 0 && (!selected || selected.category !== activeTab)) {
      setSelectedKey(items[0].key);
      setQuantity(1);
    }
  }, [activeTab, allItems]);

  const handleSelect = (key) => {
    setSelectedKey(key);
    setQuantity(1);
  };

  const handleBuy = () => {
    if (!selected) return;
    if (selected.isDog) { handleBuyDog(); return; }
    handleBuyItem(selected.key, quantity);
  };

  const totalPrice = selected ? (selected.price * quantity) : 0;
  const canAfford = farmData?.balance >= totalPrice;

  if (shopLoading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }

  return (
    <div>
      {/* ═══ Balance Bar ═══ */}
      <div className='farm-card' style={{ padding: '8px 16px', marginBottom: 12 }}>
        <Text type='tertiary' size='small'>💰 {t('余额')}: </Text>
        <Text strong>{formatBalance(farmData?.balance)}</Text>
      </div>

      {/* ═══ Category Tabs ═══ */}
      <div style={{ marginBottom: 12 }}>
        <div className='farm-market-tabs'>
          {tabs.map(tab => tab.count > 0 && (
            <div key={tab.key}
              className={`farm-market-tab ${activeTab === tab.key ? 'active' : ''}`}
              onClick={() => setActiveTab(tab.key)}>
              {tab.label}
              <span style={{ marginLeft: 4, opacity: 0.5, fontSize: 11 }}>({tab.count})</span>
            </div>
          ))}
        </div>
      </div>

      {/* ═══ Master-Detail Layout ═══ */}
      <div className='farm-shop-layout'>
        {/* Left: Item List */}
        <div className='farm-shop-list'>
          {filteredItems.length === 0 ? (
            <div className='farm-shop-empty'>
              <div className='farm-shop-empty-icon'>📭</div>
              <span>{t('暂无商品')}</span>
            </div>
          ) : filteredItems.map(item => (
            <div key={item.key}
              className={`farm-shop-item ${selectedKey === item.key ? 'active' : ''}`}
              onClick={() => handleSelect(item.key)}>
              <span className='farm-shop-item-emoji'>{item.emoji}</span>
              <div className='farm-shop-item-info'>
                <div className='farm-shop-item-name'>{item.name}</div>
                <div className='farm-shop-item-price'>${item.price?.toFixed(2)}</div>
              </div>
            </div>
          ))}
        </div>

        {/* Right: Detail Panel */}
        <div className='farm-shop-detail'>
          {!selected ? (
            <div className='farm-shop-empty'>
              <div className='farm-shop-empty-icon'>👈</div>
              <span>{t('选择一个商品查看详情')}</span>
            </div>
          ) : (
            <>
              {/* Header */}
              <div className='farm-shop-detail-header'>
                <span className='farm-shop-detail-emoji'>{selected.emoji}</span>
                <div>
                  <div className='farm-shop-detail-title'>{selected.name}</div>
                  <div className='farm-shop-detail-desc'>{selected.desc}</div>
                </div>
              </div>

              {/* Stats */}
              <div className='farm-shop-detail-body'>
                <div className='farm-shop-stats'>
                  <div className='farm-shop-stat'>
                    <div className='farm-shop-stat-label'>💰 {t('单价')}</div>
                    <div className='farm-shop-stat-value'>${selected.price?.toFixed(2)}</div>
                  </div>
                  {selected.isSeed && (
                    <>
                      <div className='farm-shop-stat'>
                        <div className='farm-shop-stat-label'>⏱ {t('生长周期')}</div>
                        <div className='farm-shop-stat-value'>{formatDuration(selected.grow_secs)}</div>
                      </div>
                      <div className='farm-shop-stat'>
                        <div className='farm-shop-stat-label'>📦 {t('产量范围')}</div>
                        <div className='farm-shop-stat-value'>1~{selected.max_yield}</div>
                      </div>
                      <div className='farm-shop-stat'>
                        <div className='farm-shop-stat-label'>📈 {t('最高收益')}</div>
                        <div className='farm-shop-stat-value' style={{ color: '#4ade80' }}>
                          ${selected.max_value?.toFixed(2)}
                        </div>
                      </div>
                    </>
                  )}
                  {!selected.isSeed && !selected.isDog && (
                    <div className='farm-shop-stat'>
                      <div className='farm-shop-stat-label'>� {t('说明')}</div>
                      <div className='farm-shop-stat-value' style={{ fontSize: 12 }}>{selected.desc}</div>
                    </div>
                  )}
                </div>

                {/* Quantity Controls */}
                {!selected.isDog && (
                  <div className='farm-shop-qty-section'>
                    <div className='farm-shop-qty-label'>
                      <span>{t('购买数量')}</span>
                      <span style={{ fontWeight: 700, fontSize: 16, color: 'var(--farm-text-0)' }}>{quantity}</span>
                    </div>

                    <input type='range'
                      className='farm-shop-slider'
                      min={1} max={selected.maxQty}
                      value={quantity}
                      onChange={(e) => setQuantity(Number(e.target.value))} />

                    {/* Preset buttons */}
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 8 }}>
                      <div className='farm-shop-presets'>
                        {[1, 5, 10, 25].filter(v => v <= selected.maxQty).map(v => (
                          <div key={v}
                            className={`farm-shop-preset ${quantity === v ? 'active' : ''}`}
                            onClick={() => setQuantity(v)}>
                            ×{v}
                          </div>
                        ))}
                        {selected.maxQty > 25 && (
                          <div
                            className={`farm-shop-preset ${quantity === selected.maxQty ? 'active' : ''}`}
                            onClick={() => setQuantity(selected.maxQty)}>
                            MAX
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                )}
              </div>

              {/* Footer: Total + Buy */}
              <div className='farm-shop-total'>
                <div>
                  <div style={{ fontSize: 10, color: 'var(--farm-text-3)', textTransform: 'uppercase', letterSpacing: 0.5 }}>
                    {t('总价')}
                  </div>
                  <div className='farm-shop-total-price'>${totalPrice.toFixed(2)}</div>
                </div>
                <Button theme='solid' size='large'
                  onClick={handleBuy}
                  loading={actionLoading}
                  disabled={!canAfford}
                  className='farm-btn'
                  style={{ minWidth: 100, fontWeight: 700 }}>
                  {!canAfford ? '💸 ' + t('余额不足') : selected.isDog ? '🐶 ' + t('购买') : '🛒 ' + t('购买')}
                </Button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default ShopPage;
