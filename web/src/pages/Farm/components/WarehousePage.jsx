import React, { useCallback, useEffect, useState } from 'react';
import { Button, Empty, Spin, Tag, Banner, Typography } from '@douyinfe/semi-ui';
import { API, seasonNames, seasonEmojis } from './utils';

const { Text } = Typography;

const fmtExpiry = (secs) => {
  if (!secs || secs <= 0) return '已过期';
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  if (h >= 24) return `${Math.floor(h / 24)}天${h % 24}时`;
  return `${h}时${m}分`;
};

const WarehousePage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [whData, setWhData] = useState(null);
  const [whLoading, setWhLoading] = useState(true);
  const [upgrading, setUpgrading] = useState(false);

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

  const handleUpgrade = async () => {
    setUpgrading(true);
    const res = await doAction('/api/farm/warehouse/upgrade', {});
    if (res) { loadWarehouse(); loadFarm(); }
    setUpgrading(false);
  };

  if (whLoading) return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  if (!whData) return <Empty description={t('加载失败')} />;

  const items = whData.items || [];
  const season = whData.season ?? 0;
  const whLevel = whData.warehouse_level ?? 1;
  const maxLevel = whData.max_level ?? 10;
  const canUpgrade = whData.can_upgrade ?? false;
  const expiryPct = whData.expiry_pct ?? 100;

  return (
    <div>
      {/* Warehouse level card */}
      <div className='farm-card'>
        <div className='farm-section-title' style={{ marginBottom: 8 }}>🏗️ {t('仓库等级')}</div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 8, marginBottom: 8 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
            <div className='farm-pill farm-pill-gold' style={{ fontSize: 14, padding: '4px 14px' }}>
              Lv.{whLevel}
            </div>
            <Text type='tertiary' size='small'>
              {t('容量')}: <Text strong size='small'>{whData.total}/{whData.max_slots}</Text>
            </Text>
            <Text type='tertiary' size='small'>
              {t('保质期')}: <Text strong size='small' style={{ color: expiryPct > 100 ? 'var(--farm-leaf)' : undefined }}>{expiryPct}%</Text>
            </Text>
          </div>
          {canUpgrade && (
            <Button theme='solid' type='tertiary' size='small' onClick={handleUpgrade}
              loading={upgrading} className='farm-btn'>
              ⬆️ {t('升级')} ${whData.upgrade_price?.toFixed(2)}
            </Button>
          )}
          {!canUpgrade && (
            <Tag color='gold' style={{ borderRadius: 12 }}>MAX</Tag>
          )}
        </div>
        {canUpgrade && (
          <div style={{ background: 'var(--semi-color-fill-0)', borderRadius: 8, padding: '8px 12px' }}>
            <Text type='tertiary' size='small'>
              {t('下一级')}: {t('容量')} {whData.next_capacity} · {t('保质期')} {whData.next_expiry_pct}%
            </Text>
          </div>
        )}
      </div>

      {/* Season info */}
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
        <div className='farm-card' data-tutorial='warehouse-items'>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <div className='farm-section-title' style={{ marginBottom: 0 }}>📦 {t('库存')}</div>
            <Button theme='solid' type='warning' size='small' onClick={handleSellAll}
              loading={actionLoading} className='farm-btn'>
              💰 {t('全部出售')}
            </Button>
          </div>
          {items.map((r) => (
            <div key={r.item_key} className='farm-row'>
              <span style={{ fontSize: 20 }}>{r.emoji}</span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                  <Text strong size='small'>{r.name}</Text>
                  <span className='farm-pill' style={{ fontSize: 11 }}>×{r.quantity}</span>
                  {r.category === 'crop' && r.in_season !== undefined && (
                    <Tag size='small' color={r.in_season ? 'green' : 'orange'} style={{ borderRadius: 12 }}>
                      {r.in_season ? t('应季') : t('反季')} {r.season_pct}%
                    </Tag>
                  )}
                  {(r.category === 'meat' || r.category === 'recipe') && r.expire_remain !== undefined && (
                    <Tag size='small' color={r.expire_remain > 86400 ? 'blue' : r.expire_remain > 0 ? 'orange' : 'red'} style={{ borderRadius: 12 }}>
                      ⏳ {fmtExpiry(r.expire_remain)}
                    </Tag>
                  )}
                </div>
                <Text type='tertiary' size='small'>
                  {t('单价')} ${r.unit_price?.toFixed(2)} · {t('总值')} <Text size='small' strong style={{ color: 'var(--farm-harvest)' }}>${r.total_value?.toFixed(2)}</Text>
                </Text>
              </div>
              <Button size='small' theme='solid' type='warning' onClick={() => handleSell(r.item_key)}
                loading={actionLoading} className='farm-btn'>{t('出售')}</Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default WarehousePage;
