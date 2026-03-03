import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
} from '../../helpers';
import {
  Button,
  Card,
  Empty,
  Select,
  Spin,
  TabPane,
  Tabs,
  Tag,
  Typography,
  Progress,
  Table,
  Descriptions,
  Banner,
} from '@douyinfe/semi-ui';
import {
  Sprout,
  Wheat,
  Store,
  Droplets,
  Dog,
  RefreshCw,
  Swords,
  Pill,
  FlaskConical,
  LandPlot,
  Package,
} from 'lucide-react';

const { Text, Title } = Typography;

const formatDuration = (secs) => {
  if (!secs || secs <= 0) return '0分';
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  if (h > 0) return `${h}时${m}分`;
  return `${m}分`;
};

const statusColors = {
  0: 'default',
  1: 'blue',
  2: 'green',
  3: 'red',
  4: 'orange',
};

const statusEmojis = {
  0: '⬜',
  1: '🌱',
  2: '✅',
  3: '⚠️',
  4: '🥀',
};

// ===================== Sub-page: Farm Overview =====================
const FarmOverview = ({ farmData, loading, loadFarm, actionLoading, doAction, t }) => {
  if (!farmData) return null;

  const handleWater = (idx) => doAction('/api/farm/water', { plot_index: idx });
  const handleTreat = (idx) => doAction('/api/farm/treat', { plot_index: idx });
  const handleFertilize = (idx) => doAction('/api/farm/fertilize', { plot_index: idx });
  const handleHarvest = () => doAction('/api/farm/harvest', {});
  const handleBuyLand = () => doAction('/api/farm/buyland', {});

  const matureCount = (farmData.plots || []).filter(p => p.status === 2).length;

  return (
    <div>
      {/* Status bar */}
      <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 20, alignItems: 'center' }}>
          <div>
            <Text type='tertiary' size='small'>💰 {t('余额')}</Text>
            <div><Text strong style={{ fontSize: 18 }}>${farmData.balance?.toFixed(2)}</Text></div>
          </div>
          <div>
            <Text type='tertiary' size='small'>📊 {t('土地')}</Text>
            <div><Text strong style={{ fontSize: 18 }}>{farmData.plot_count}/{farmData.max_plots}</Text></div>
          </div>
          <div style={{ flex: 1 }} />
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <Button icon={<RefreshCw size={14} />} theme='borderless' onClick={loadFarm} loading={loading}>
              {t('刷新')}
            </Button>
            {matureCount > 0 && (
              <Button icon={<Wheat size={14} />} theme='solid' style={{ borderRadius: 10, background: '#f59e0b' }}
                onClick={handleHarvest} loading={actionLoading}>
                {t('一键收获')} ({matureCount})
              </Button>
            )}
            {farmData.plot_count < farmData.max_plots && (
              <Button icon={<LandPlot size={14} />} theme='light' onClick={handleBuyLand}
                loading={actionLoading} style={{ borderRadius: 10 }}>
                {t('购买土地')} (${farmData.plot_price?.toFixed(2)})
              </Button>
            )}
          </div>
        </div>
      </Card>

      {/* Backpack */}
      {farmData.items && farmData.items.length > 0 && (
        <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
            <Package size={16} />
            <Text strong>{t('背包')}</Text>
          </div>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            {farmData.items.map((item) => (
              <Tag key={item.key} size='large' color='blue' style={{ padding: '4px 12px', borderRadius: 8 }}>
                {item.emoji} {item.name} ×{item.quantity}
              </Tag>
            ))}
          </div>
        </Card>
      )}

      {/* Dog summary */}
      {farmData.dog && (
        <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <span style={{ fontSize: 28 }}>{farmData.dog.level === 2 ? '🐕' : '🐶'}</span>
            <div>
              <Text strong style={{ fontSize: 15 }}>「{farmData.dog.name}」</Text>
              <div>
                <Tag size='small' color={farmData.dog.hunger > 0 ? 'green' : 'red'} style={{ marginRight: 6 }}>
                  {farmData.dog.level_name}
                </Tag>
                <Text type='tertiary' size='small'>{farmData.dog.status} · {t('饱食度')} {farmData.dog.hunger}%</Text>
              </div>
            </div>
          </div>
        </Card>
      )}

      {/* Plot grid */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 12 }}>
        {(farmData.plots || []).map((plot) => (
          <Card key={plot.plot_index} className='!rounded-xl'
            style={{
              border: `2px solid ${plot.status === 3 || plot.status === 4 ? 'var(--semi-color-danger)' : plot.status === 2 ? 'var(--semi-color-success)' : 'var(--semi-color-border)'}`,
              background: plot.status === 0 ? 'var(--semi-color-fill-0)' : undefined,
            }}>
            {/* Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
              <Text strong style={{ fontSize: 15 }}>
                {statusEmojis[plot.status]} {plot.plot_index + 1}{t('号地')}
              </Text>
              <Tag size='small' color={statusColors[plot.status]}>{plot.status_label}</Tag>
            </div>

            {/* Empty */}
            {plot.status === 0 && (
              <div style={{ padding: '12px 0', textAlign: 'center' }}>
                <Text type='tertiary'>{t('空地，等待种植')}</Text>
              </div>
            )}

            {/* Growing */}
            {plot.status === 1 && (
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                  <span style={{ fontSize: 20 }}>{plot.crop_emoji}</span>
                  <Text strong>{plot.crop_name}</Text>
                  {plot.fertilized === 1 && <Tag size='small' color='cyan'>🧴 {t('已施肥')}</Tag>}
                </div>
                <div style={{ marginBottom: 6 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                    <Text type='tertiary' size='small'>{t('生长进度')}</Text>
                    <Text type='tertiary' size='small'>{plot.progress}%</Text>
                  </div>
                  <Progress percent={plot.progress} size='small' />
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 10 }}>
                  <Text type='tertiary' size='small'>⏳ {t('剩余')} {formatDuration(plot.remaining)}</Text>
                  {plot.last_watered_at > 0 && (
                    <Text type={plot.water_remain <= 0 ? 'danger' : 'tertiary'} size='small'>
                      💧 {plot.water_remain > 0 ? formatDuration(plot.water_remain) : '⚠️ ' + t('需浇水')}
                    </Text>
                  )}
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  <Button size='small' icon={<Droplets size={12} />} onClick={() => handleWater(plot.plot_index)}
                    loading={actionLoading} style={{ borderRadius: 8 }}>{t('浇水')}</Button>
                  {plot.fertilized === 0 && (
                    <Button size='small' icon={<FlaskConical size={12} />} onClick={() => handleFertilize(plot.plot_index)}
                      loading={actionLoading} style={{ borderRadius: 8 }}>{t('施肥')}</Button>
                  )}
                </div>
              </div>
            )}

            {/* Mature */}
            {plot.status === 2 && (
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                  <span style={{ fontSize: 20 }}>{plot.crop_emoji}</span>
                  <Text strong>{plot.crop_name}</Text>
                  <Tag size='small' color='green'>{t('已成熟')}</Tag>
                </div>
                {plot.stolen_count > 0 && (
                  <Banner type='warning' description={`⚠️ ${t('已被偷')} ${plot.stolen_count} ${t('次')}`}
                    style={{ borderRadius: 8, marginTop: 4 }} />
                )}
              </div>
            )}

            {/* Event */}
            {plot.status === 3 && (
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                  <span style={{ fontSize: 20 }}>{plot.crop_emoji}</span>
                  <Text strong>{plot.crop_name}</Text>
                </div>
                {plot.event_type === 'drought' ? (
                  <div>
                    <Banner type='danger' description={`🏜️ ${t('天灾干旱！快浇水救命！')}`}
                      style={{ borderRadius: 8, marginBottom: 8 }} />
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Text type='danger' size='small'>💀 {formatDuration(plot.death_remain)} {t('后死亡')}</Text>
                      <Button size='small' type='danger' icon={<Droplets size={12} />}
                        onClick={() => handleWater(plot.plot_index)} loading={actionLoading}
                        style={{ borderRadius: 8 }}>{t('浇水救命')}</Button>
                    </div>
                  </div>
                ) : (
                  <div>
                    <Banner type='warning' description={`🐛 ${t('虫害发作！需要杀虫剂治疗')}`}
                      style={{ borderRadius: 8, marginBottom: 8 }} />
                    <Button size='small' type='warning' icon={<Pill size={12} />}
                      onClick={() => handleTreat(plot.plot_index)} loading={actionLoading}
                      style={{ borderRadius: 8 }}>{t('使用杀虫剂')}</Button>
                  </div>
                )}
              </div>
            )}

            {/* Wilting */}
            {plot.status === 4 && (
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                  <span style={{ fontSize: 20 }}>{plot.crop_emoji}</span>
                  <Text strong>{plot.crop_name}</Text>
                </div>
                <Banner type='danger' description={`🥀 ${t('作物枯萎中！快浇水！')}`}
                  style={{ borderRadius: 8, marginBottom: 8 }} />
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text type='danger' size='small'>💀 {formatDuration(plot.death_remain)} {t('后死亡')}</Text>
                  <Button size='small' type='danger' icon={<Droplets size={12} />}
                    onClick={() => handleWater(plot.plot_index)} loading={actionLoading}
                    style={{ borderRadius: 8 }}>{t('浇水救命')}</Button>
                </div>
              </div>
            )}
          </Card>
        ))}
      </div>
    </div>
  );
};

// ===================== Sub-page: Planting =====================
const PlantPage = ({ farmData, crops, actionLoading, doAction, loadFarm, t }) => {
  const [selectedCrop, setSelectedCrop] = useState(null);

  if (!farmData) return null;

  const emptyPlots = (farmData.plots || []).filter(p => p.status === 0);
  const activeCrop = crops.find(c => c.key === selectedCrop);

  const handlePlant = async (plotIndex) => {
    if (!selectedCrop) {
      showError(t('请先选择作物'));
      return;
    }
    const res = await doAction('/api/farm/plant', { crop_key: selectedCrop, plot_index: plotIndex });
    if (res) loadFarm();
  };

  return (
    <div>
      {/* Crop selection */}
      <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
        <Title heading={6} style={{ marginBottom: 12 }}>🌱 {t('选择要种植的作物')}</Title>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 10 }}>
          {crops.map((crop) => (
            <div key={crop.key}
              onClick={() => setSelectedCrop(crop.key)}
              style={{
                padding: '14px 16px', borderRadius: 12, cursor: 'pointer',
                border: `2px solid ${selectedCrop === crop.key ? 'var(--semi-color-primary)' : 'var(--semi-color-border)'}`,
                background: selectedCrop === crop.key ? 'var(--semi-color-primary-light-default)' : 'var(--semi-color-fill-0)',
                transition: 'all 0.2s',
              }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
                <span style={{ fontSize: 28 }}>{crop.emoji}</span>
                <div>
                  <Text strong style={{ fontSize: 15 }}>{crop.name}</Text>
                  <div><Tag size='small' color='green'>${crop.seed_cost?.toFixed(2)}</Tag></div>
                </div>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 4, fontSize: 12 }}>
                <Text type='tertiary'>⏱ {formatDuration(crop.grow_secs)}</Text>
                <Text type='tertiary'>📦 {t('产量')} 1~{crop.max_yield}</Text>
                <Text type='tertiary'>💎 {t('单价')} ${crop.unit_price?.toFixed(2)}</Text>
                <Text type='tertiary'>🏆 {t('最高')} ${crop.max_value?.toFixed(2)}</Text>
              </div>
            </div>
          ))}
        </div>
      </Card>

      {/* Selected crop info */}
      {activeCrop && (
        <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
          <Title heading={6} style={{ marginBottom: 12 }}>
            {activeCrop.emoji} {t('正在种植')}: {activeCrop.name}
          </Title>
          <Descriptions row size='small' data={[
            { key: t('种子价格'), value: `$${activeCrop.seed_cost?.toFixed(2)}` },
            { key: t('生长时间'), value: formatDuration(activeCrop.grow_secs) },
            { key: t('产量范围'), value: `1 ~ ${activeCrop.max_yield}` },
            { key: t('单位价格'), value: `$${activeCrop.unit_price?.toFixed(2)}` },
            { key: t('最高收益'), value: `$${activeCrop.max_value?.toFixed(2)}` },
          ]} />
        </Card>
      )}

      {/* Plot selection */}
      <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)' }}>
        <Title heading={6} style={{ marginBottom: 12 }}>📍 {t('选择空地种植')}</Title>
        {emptyPlots.length === 0 ? (
          <Empty description={t('没有空地了！请先收获或购买新土地')} />
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 10 }}>
            {emptyPlots.map((plot) => (
              <Button key={plot.plot_index}
                theme='light' size='large'
                disabled={!selectedCrop || actionLoading}
                loading={actionLoading}
                onClick={() => handlePlant(plot.plot_index)}
                style={{
                  borderRadius: 12, height: 64, width: '100%',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                }}>
                <span style={{ fontSize: 20, marginRight: 8 }}>⬜</span>
                {plot.plot_index + 1}{t('号地')}
                {selectedCrop && <span style={{ marginLeft: 8, fontSize: 12 }}>→ {t('点击种植')}</span>}
              </Button>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
};

// ===================== Sub-page: Shop =====================
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

  const handleBuyItem = async (key) => {
    const res = await doAction('/api/farm/buy', { item_key: key });
    if (res) { loadShop(); loadFarm(); }
  };

  const handleBuyDog = async () => {
    const res = await doAction('/api/farm/buydog', {});
    if (res) { loadShop(); loadFarm(); }
  };

  if (shopLoading) {
    return <div style={{ textAlign: 'center', padding: 60 }}><Spin size='large' /></div>;
  }

  return (
    <div>
      {/* Balance */}
      <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Text type='tertiary'>💰 {t('当前余额')}:</Text>
          <Text strong style={{ fontSize: 20 }}>${farmData?.balance?.toFixed(2)}</Text>
        </div>
      </Card>

      {/* Seeds catalog */}
      <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
        <Title heading={6} style={{ marginBottom: 12 }}>🌱 {t('种子目录')} <Text type='tertiary' size='small'>({t('在种植页面直接购买并种下')})</Text></Title>
        <Table
          dataSource={crops}
          pagination={false}
          size='small'
          columns={[
            { title: t('作物'), dataIndex: 'name', render: (_, r) => <span>{r.emoji} {r.name}</span> },
            { title: t('价格'), dataIndex: 'seed_cost', render: v => `$${v?.toFixed(2)}` },
            { title: t('生长时间'), dataIndex: 'grow_secs', render: v => formatDuration(v) },
            { title: t('产量'), dataIndex: 'max_yield', render: (v, r) => `1~${v} × $${r.unit_price?.toFixed(2)}` },
            { title: t('最高收益'), dataIndex: 'max_value', render: v => <Tag color='green'>${v?.toFixed(2)}</Tag> },
          ]}
        />
      </Card>

      {/* Items */}
      <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
        <Title heading={6} style={{ marginBottom: 12 }}>📦 {t('道具')}</Title>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 10 }}>
          {(shopData?.items || []).map((item) => (
            <div key={item.key} style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              padding: '16px 18px', borderRadius: 12,
              border: '1px solid var(--semi-color-border)',
              background: 'var(--semi-color-fill-0)',
            }}>
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{ fontSize: 24 }}>{item.emoji}</span>
                  <div>
                    <Text strong style={{ fontSize: 15 }}>{item.name}</Text>
                    <div><Text type='tertiary' size='small'>{item.desc}</Text></div>
                  </div>
                </div>
              </div>
              <Button theme='solid' onClick={() => handleBuyItem(item.key)}
                loading={actionLoading} style={{ borderRadius: 10 }}>
                ${t('购买')} ${item.cost?.toFixed(2)}
              </Button>
            </div>
          ))}
        </div>
      </Card>

      {/* Dog purchase */}
      {shopData && !shopData.has_dog && (
        <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)' }}>
          <Title heading={6} style={{ marginBottom: 12 }}>🐕 {t('看门狗')}</Title>
          <div style={{
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            padding: '20px 24px', borderRadius: 12,
            border: '1px solid var(--semi-color-border)',
            background: 'var(--semi-color-fill-0)',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <span style={{ fontSize: 32 }}>🐶</span>
              <div>
                <Text strong style={{ fontSize: 15 }}>{t('小狗')}</Text>
                <div><Text type='tertiary' size='small'>{t('长大后可拦截偷菜者')}</Text></div>
              </div>
            </div>
            <Button theme='solid' onClick={handleBuyDog}
              loading={actionLoading} style={{ borderRadius: 10 }}>
              {t('购买')} ${shopData.dog_price?.toFixed(2)}
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
};

// ===================== Sub-page: Steal =====================
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
    return <div style={{ textAlign: 'center', padding: 60 }}><Spin size='large' /></div>;
  }

  return (
    <div>
      {/* Steal history */}
      {stealResults.length > 0 && (
        <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
          <Title heading={6} style={{ marginBottom: 12 }}>📜 {t('本次偷菜记录')}</Title>
          {stealResults.map((r, i) => (
            <div key={i} style={{
              padding: '10px 14px', borderRadius: 10, marginBottom: 6,
              background: 'var(--semi-color-fill-0)',
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
            }}>
              <div>
                <Text type='tertiary' size='small'>{r.time}</Text>
                <div><Text>{r.message}</Text></div>
              </div>
              {r.data && <Tag color='green' size='large'>${r.data.value?.toFixed(2)}</Tag>}
            </div>
          ))}
        </Card>
      )}

      {/* Targets */}
      <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
          <Title heading={6} style={{ margin: 0 }}>🕵️ {t('可偷菜的农场')}</Title>
          <Button icon={<RefreshCw size={14} />} theme='borderless' onClick={loadTargets} loading={stealLoading}>
            {t('刷新')}
          </Button>
        </div>

        {targets.length === 0 ? (
          <Empty description={t('暂时没有可偷的菜地，等其他玩家作物成熟后再来！')} />
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 10 }}>
            {targets.map((target) => (
              <div key={target.id} style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                padding: '16px 18px', borderRadius: 12,
                border: '1px solid var(--semi-color-border)',
                background: 'var(--semi-color-fill-0)',
              }}>
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ fontSize: 24 }}>👤</span>
                    <div>
                      <Text strong style={{ fontSize: 15 }}>{target.label}</Text>
                      <div>
                        <Tag size='small' color='green'>{target.count} {t('块成熟')}</Tag>
                      </div>
                    </div>
                  </div>
                </div>
                <Button type='warning' theme='solid' onClick={() => handleSteal(target.id)}
                  loading={actionLoading} style={{ borderRadius: 10 }}>
                  🕵️ {t('偷菜')}
                </Button>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
};

// ===================== Sub-page: Dog =====================
const DogPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [dogData, setDogData] = useState(null);
  const [dogLoading, setDogLoading] = useState(true);

  const loadDog = useCallback(async () => {
    setDogLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/dog');
      if (res.success) setDogData(res.data);
    } catch (err) { /* ignore */ }
    finally { setDogLoading(false); }
  }, []);

  useEffect(() => { loadDog(); }, [loadDog]);

  const handleBuyDog = async () => {
    const res = await doAction('/api/farm/buydog', {});
    if (res) { loadDog(); loadFarm(); }
  };

  const handleFeedDog = async () => {
    const res = await doAction('/api/farm/feeddog', {});
    if (res) { loadDog(); loadFarm(); }
  };

  if (dogLoading) {
    return <div style={{ textAlign: 'center', padding: 60 }}><Spin size='large' /></div>;
  }

  if (!dogData || !dogData.has_dog) {
    return (
      <div>
        <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', textAlign: 'center', padding: '40px 20px' }}>
          <div style={{ fontSize: 64, marginBottom: 16 }}>🐶</div>
          <Title heading={5}>{t('你还没有看门狗！')}</Title>
          <Text type='tertiary' style={{ display: 'block', marginBottom: 8 }}>
            {t('购买一只小狗，养大后可以帮你看门拦截偷菜者！')}
          </Text>

          <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', marginTop: 20, textAlign: 'left', maxWidth: 400, margin: '20px auto 0' }}>
            <Descriptions row size='small' data={[
              { key: t('价格'), value: `$${dogData?.dog_price?.toFixed(2)}` },
              { key: t('幼犬成长时间'), value: `${dogData?.grow_hours} ${t('小时')}` },
              { key: t('成犬拦截率'), value: `${dogData?.guard_rate}%` },
              { key: t('狗粮价格'), value: `$${dogData?.food_price?.toFixed(2)}` },
            ]} />
          </Card>

          <Button theme='solid' size='large' style={{ marginTop: 24, borderRadius: 12 }}
            onClick={handleBuyDog} loading={actionLoading}>
            🐶 {t('购买小狗')} (${dogData?.dog_price?.toFixed(2)})
          </Button>
        </Card>
      </div>
    );
  }

  return (
    <div>
      {/* Dog profile */}
      <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
        <div style={{ textAlign: 'center', paddingTop: 12, paddingBottom: 12 }}>
          <div style={{ fontSize: 64, marginBottom: 8 }}>{dogData.level === 2 ? '🐕' : '🐶'}</div>
          <Title heading={4} style={{ margin: 0 }}>「{dogData.name}」</Title>
          <Tag size='large' color={dogData.hunger > 0 ? 'green' : 'red'} style={{ marginTop: 8 }}>
            {dogData.level_name} · {dogData.status}
          </Tag>
        </div>
      </Card>

      {/* Stats */}
      <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: 12 }}>
          <div style={{ padding: 16, borderRadius: 12, background: 'var(--semi-color-fill-0)', textAlign: 'center' }}>
            <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('等级')}</Text>
            <Text strong style={{ fontSize: 20 }}>{dogData.level_name}</Text>
          </div>
          <div style={{ padding: 16, borderRadius: 12, background: 'var(--semi-color-fill-0)', textAlign: 'center' }}>
            <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('饱食度')}</Text>
            <Text strong style={{ fontSize: 20 }}>{dogData.hunger}%</Text>
            <Progress percent={dogData.hunger} size='small' style={{ marginTop: 4 }}
              stroke={dogData.hunger > 30 ? '#22c55e' : '#ef4444'} />
          </div>
          <div style={{ padding: 16, borderRadius: 12, background: 'var(--semi-color-fill-0)', textAlign: 'center' }}>
            <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('拦截率')}</Text>
            <Text strong style={{ fontSize: 20 }}>{dogData.guard_rate}%</Text>
          </div>
          <div style={{ padding: 16, borderRadius: 12, background: 'var(--semi-color-fill-0)', textAlign: 'center' }}>
            <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('状态')}</Text>
            <Text strong style={{ fontSize: 20 }}>{dogData.status}</Text>
          </div>
        </div>
      </Card>

      {/* Actions */}
      <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)' }}>
        <Title heading={6} style={{ marginBottom: 12 }}>{t('操作')}</Title>
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          {dogData.hunger < 100 ? (
            <Button theme='solid' size='large' icon={<span>🦴</span>}
              onClick={handleFeedDog} loading={actionLoading}
              style={{ borderRadius: 12, flex: 1, minWidth: 200 }}>
              {t('喂狗粮')} ({t('恢复饱食度到')} 100%)
            </Button>
          ) : (
            <Banner type='success' description={`✅ ${t('狗狗吃饱了，不需要喂食！')}`}
              style={{ borderRadius: 10, width: '100%' }} />
          )}
        </div>
        <div style={{ marginTop: 12 }}>
          <Text type='tertiary' size='small'>
            💡 {t('提示：饱食度为0时狗狗无法看门。每小时消耗1点饱食度，请定期喂食。')}
          </Text>
          <br />
          <Text type='tertiary' size='small'>
            🦴 {t('狗粮价格')}: ${dogData.food_price?.toFixed(2)} — {t('可在商店购买')}
          </Text>
        </div>
      </Card>
    </div>
  );
};

// ===================== Main Farm Component =====================
const Farm = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [farmData, setFarmData] = useState(null);
  const [crops, setCrops] = useState([]);
  const [actionLoading, setActionLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('overview');

  const loadFarm = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/view');
      if (res.success) {
        setFarmData(res.data);
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const loadCrops = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/crops');
      if (res.success) setCrops(res.data || []);
    } catch (err) { /* ignore */ }
  }, []);

  useEffect(() => {
    loadFarm();
    loadCrops();
  }, [loadFarm, loadCrops]);

  useEffect(() => {
    const interval = setInterval(loadFarm, 30000);
    return () => clearInterval(interval);
  }, [loadFarm]);

  const doAction = async (url, body) => {
    setActionLoading(true);
    try {
      const { data: res } = await API.post(url, body);
      if (res.success) {
        showSuccess(res.message || t('操作成功'));
        loadFarm();
        return res;
      } else {
        showError(res.message);
        return null;
      }
    } catch (err) {
      showError(t('操作失败'));
      return null;
    } finally {
      setActionLoading(false);
    }
  };

  if (loading && !farmData) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 400, marginTop: 60 }}>
        <Spin size='large' />
      </div>
    );
  }

  if (!farmData) {
    return (
      <div style={{ maxWidth: 960, margin: '80px auto 0', padding: '0 16px' }}>
        <Card className='!rounded-2xl' style={{ border: '1px solid var(--semi-color-border)', textAlign: 'center', padding: 40 }}>
          <Sprout size={48} style={{ color: 'var(--semi-color-text-3)', marginBottom: 16 }} />
          <Title heading={5}>{t('农场不可用')}</Title>
          <Text type='tertiary'>{t('请先绑定 Telegram 账号后才能使用农场功能')}</Text>
        </Card>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 1060, margin: '70px auto 0', padding: '0 16px 40px' }}>
      {/* Page title */}
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 20 }}>
        <div style={{
          width: 48, height: 48, borderRadius: 14,
          background: 'linear-gradient(135deg, #22c55e, #16a34a)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          boxShadow: '0 2px 8px rgba(34,197,94,0.3)', flexShrink: 0, marginRight: 14,
        }}>
          <Wheat size={24} style={{ color: 'white' }} />
        </div>
        <div>
          <Title heading={4} style={{ margin: 0 }}>🌾 {t('我的农场')}</Title>
          <Text type='tertiary' size='small'>
            {t('种植、收获、偷菜，在线经营你的农场！')}
          </Text>
        </div>
      </div>

      {/* Tab navigation */}
      <Tabs type='button' activeKey={activeTab} onChange={setActiveTab}
        style={{ marginBottom: 16 }}>
        <TabPane tab={<span><Wheat size={14} style={{ marginRight: 4, verticalAlign: -2 }} /> {t('农场总览')}</span>} itemKey='overview'>
          <FarmOverview farmData={farmData} loading={loading} loadFarm={loadFarm}
            actionLoading={actionLoading} doAction={doAction} t={t} />
        </TabPane>
        <TabPane tab={<span><Sprout size={14} style={{ marginRight: 4, verticalAlign: -2 }} /> {t('种植')}</span>} itemKey='plant'>
          <PlantPage farmData={farmData} crops={crops} actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Store size={14} style={{ marginRight: 4, verticalAlign: -2 }} /> {t('商店')}</span>} itemKey='shop'>
          <ShopPage farmData={farmData} actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Swords size={14} style={{ marginRight: 4, verticalAlign: -2 }} /> {t('偷菜')}</span>} itemKey='steal'>
          <StealPage actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Dog size={14} style={{ marginRight: 4, verticalAlign: -2 }} /> {t('狗狗')}</span>} itemKey='dog'>
          <DogPage actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
      </Tabs>
    </div>
  );
};

export default Farm;
