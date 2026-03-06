import React, { useCallback, useEffect, useState, lazy, Suspense } from 'react';
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
  Spin,
  TabPane,
  Tabs,
  Tag,
  Typography,
  Progress,
  Table,
  Descriptions,
  Banner,
  InputNumber,
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
  Box,
  List,
  ArrowUp,
  ScrollText,
  Fish,
  TrendingUp,
  Factory,
  Trophy,
  ClipboardList,
  Star,
  Landmark,
  Warehouse,
} from 'lucide-react';

import { VChart } from '@visactor/react-vchart';

const Farm3DView = lazy(() => import('./Farm3D'));

const { Text, Title } = Typography;

const formatDuration = (secs) => {
  if (!secs || secs <= 0) return '0分';
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  if (h > 0) return `${h}时${m}分`;
  return `${m}分`;
};

const formatBalance = (val) => {
  if (val == null) return '$0.00';
  if (val >= 1e12) return `$${(val / 1e12).toFixed(2)}T`;
  if (val >= 1e9) return `$${(val / 1e9).toFixed(2)}B`;
  if (val >= 1e6) return `$${(val / 1e6).toFixed(2)}M`;
  if (val >= 1e4) return `$${(val / 1e3).toFixed(2)}K`;
  return `$${val.toFixed(2)}`;
};

const statusColors = { 0: 'default', 1: 'blue', 2: 'green', 3: 'red', 4: 'orange' };

// ===================== Sub-page: Farm Overview =====================
const FarmOverview = ({ farmData, loading, loadFarm, actionLoading, doAction, t }) => {
  const [viewMode, setViewMode] = useState('3d');
  const [selectedPlotIndex, setSelectedPlotIndex] = useState(null);

  if (!farmData) return null;

  const handleWater = (idx) => doAction('/api/farm/water', { plot_index: idx });
  const handleTreat = (idx) => doAction('/api/farm/treat', { plot_index: idx });
  const handleFertilize = (idx) => doAction('/api/farm/fertilize', { plot_index: idx });
  const handleHarvest = () => doAction('/api/farm/harvest', {});
  const handleHarvestStore = () => doAction('/api/farm/harvest/store', {});
  const handleBuyLand = () => doAction('/api/farm/buyland', {});
  const handleUpgradeSoil = (idx) => doAction('/api/farm/upgrade-soil', { plot_index: idx });

  const plots = farmData.plots || [];
  const matureCount = plots.filter(p => p.status === 2).length;
  const activePlots = plots.filter(p => p.status !== 0);
  const emptyPlots = plots.filter(p => p.status === 0);

  return (
    <div>
      {/* Compact status + actions bar */}
      <div style={{
        display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10,
        padding: '10px 14px', borderRadius: 10, marginBottom: 12,
        background: 'var(--semi-color-fill-0)', border: '1px solid var(--semi-color-border)',
      }}>
        <Tag size='large' color='light-blue' style={{ borderRadius: 6 }}>💰 {formatBalance(farmData.balance)}</Tag>
        <Tag size='large' color='grey' style={{ borderRadius: 6 }}>📊 {farmData.plot_count}/{farmData.max_plots}</Tag>
        {farmData.items && farmData.items.map((item) => (
          <Tag key={item.key} size='large' color='blue' style={{ borderRadius: 6 }}>
            {item.emoji} {item.name} ×{item.quantity}
          </Tag>
        ))}
        {farmData.dog && (
          <Tag size='large' color={farmData.dog.hunger > 0 ? 'green' : 'red'} style={{ borderRadius: 6 }}>
            {farmData.dog.level === 2 ? '🐕' : '🐶'} {farmData.dog.name} · {farmData.dog.level_name} · {farmData.dog.hunger}%
          </Tag>
        )}
        <div style={{ flex: 1 }} />
        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', alignItems: 'center' }}>
          <Button size='small' icon={viewMode === '3d' ? <List size={12} /> : <Box size={12} />}
            theme='borderless' onClick={() => setViewMode(viewMode === '3d' ? 'list' : '3d')}
            style={{ borderRadius: 6 }}>
            {viewMode === '3d' ? t('列表') : '3D'}
          </Button>
          <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadFarm} loading={loading} />
          {matureCount > 0 && (
            <>
              <Button size='small' icon={<Wheat size={12} />} theme='solid'
                style={{ borderRadius: 6, background: '#f59e0b' }}
                onClick={handleHarvest} loading={actionLoading}>
                {t('收获出售')}({matureCount})
              </Button>
              <Button size='small' icon={<Package size={12} />} theme='solid'
                style={{ borderRadius: 6, background: '#6366f1' }}
                onClick={handleHarvestStore} loading={actionLoading}>
                {t('收获入仓')}
              </Button>
            </>
          )}
          {farmData.plot_count < farmData.max_plots && (
            <Button size='small' icon={<LandPlot size={12} />} theme='light' onClick={handleBuyLand}
              loading={actionLoading} style={{ borderRadius: 6 }}>
              {t('买地')}({formatBalance(farmData.plot_price)})
            </Button>
          )}
        </div>
      </div>

      {/* 3D Farm View */}
      {viewMode === '3d' && (
        <div style={{ marginBottom: 12 }}>
          <Suspense fallback={
            <div style={{
              width: '100%', height: 500, borderRadius: 12, display: 'flex',
              alignItems: 'center', justifyContent: 'center',
              background: 'linear-gradient(180deg, #bae6fd 0%, #e0f2fe 40%, #dcfce7 100%)',
              border: '2px solid var(--semi-color-border)',
            }}>
              <Spin size='large' />
            </div>
          }>
            <Farm3DView
              farmData={farmData}
              doAction={doAction}
              t={t}
              selectedPlotIndex={selectedPlotIndex}
              setSelectedPlotIndex={setSelectedPlotIndex}
            />
          </Suspense>
          <div style={{
            textAlign: 'center', marginTop: 6,
          }}>
            <Text type='tertiary' size='small'>🖱️ {t('拖拽旋转 · 滚轮缩放 · 点击地块查看详情')}</Text>
          </div>
        </div>
      )}

      {/* List view (fallback / alternative) */}
      {viewMode === 'list' && (
        <>
          {/* Active plots - card grid */}
          {activePlots.length > 0 && (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 10, marginBottom: emptyPlots.length > 0 ? 12 : 0 }}>
              {activePlots.map((plot) => (
                <div key={plot.plot_index} style={{
                  padding: '12px 14px', borderRadius: 10,
                  border: `1.5px solid ${plot.status === 3 || plot.status === 4 ? 'var(--semi-color-danger)' : plot.status === 2 ? 'var(--semi-color-success)' : 'var(--semi-color-border)'}`,
                }}>
                  {/* Growing */}
                  {plot.status === 1 && (
                    <div>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
                        <span style={{ fontSize: 13, fontWeight: 600 }}>
                          {plot.crop_emoji} {plot.plot_index + 1}{t('号地')} · {plot.crop_name}
                          {plot.fertilized === 1 && <Tag size='small' color='cyan' style={{ marginLeft: 4 }}>🧴</Tag>}
                          {(plot.soil_level || 1) > 1 && <Tag size='small' color='violet' style={{ marginLeft: 4 }}>🌱Lv.{plot.soil_level}</Tag>}
                        </span>
                        <Tag size='small' color='blue'>{plot.progress}%</Tag>
                      </div>
                      <Progress percent={plot.progress} size='small' style={{ marginBottom: 4 }} />
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6, fontSize: 12 }}>
                        <Text type='tertiary' size='small'>⏳ {formatDuration(plot.remaining)}</Text>
                        {plot.last_watered_at > 0 && (
                          <Text type={plot.water_remain <= 0 ? 'danger' : 'tertiary'} size='small'>
                            💧 {plot.water_remain > 0 ? formatDuration(plot.water_remain) : '⚠️ ' + t('需浇水')}
                          </Text>
                        )}
                      </div>
                      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                        <Button size='small' icon={<Droplets size={11} />} onClick={() => handleWater(plot.plot_index)}
                          loading={actionLoading} style={{ borderRadius: 6, fontSize: 12 }}>{t('浇水')}</Button>
                        {plot.fertilized === 0 && (
                          <Button size='small' icon={<FlaskConical size={11} />} onClick={() => handleFertilize(plot.plot_index)}
                            loading={actionLoading} style={{ borderRadius: 6, fontSize: 12 }}>{t('施肥')}</Button>
                        )}
                        {(plot.soil_level || 1) < (farmData.soil_max_level || 5) && (
                          <Button size='small' icon={<ArrowUp size={11} />} onClick={() => handleUpgradeSoil(plot.plot_index)}
                            loading={actionLoading} style={{ borderRadius: 6, fontSize: 12 }}>{t('升级泥土')} Lv.{(plot.soil_level || 1)+1}</Button>
                        )}
                      </div>
                    </div>
                  )}

                  {/* Mature */}
                  {plot.status === 2 && (
                    <div>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <span style={{ fontSize: 13, fontWeight: 600 }}>
                          {plot.crop_emoji} {plot.plot_index + 1}{t('号地')} · {plot.crop_name}
                          {(plot.soil_level || 1) > 1 && <Tag size='small' color='violet' style={{ marginLeft: 4 }}>🌱Lv.{plot.soil_level}</Tag>}
                        </span>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          {plot.stolen_count > 0 && (
                            <Text type='warning' size='small'>⚠️ -{plot.stolen_count}</Text>
                          )}
                          <Tag size='small' color='green'>✅ {t('已成熟')}</Tag>
                        </div>
                      </div>
                      {(plot.soil_level || 1) < (farmData.soil_max_level || 5) && (
                        <div style={{ display: 'flex', gap: 6, marginTop: 6 }}>
                          <Button size='small' icon={<ArrowUp size={11} />} onClick={() => handleUpgradeSoil(plot.plot_index)}
                            loading={actionLoading} style={{ borderRadius: 6, fontSize: 12 }}>{t('升级泥土')} Lv.{(plot.soil_level || 1)+1}</Button>
                        </div>
                      )}
                    </div>
                  )}

                  {/* Event */}
                  {plot.status === 3 && (
                    <div>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
                        <span style={{ fontSize: 13, fontWeight: 600 }}>
                          {plot.crop_emoji} {plot.plot_index + 1}{t('号地')} · {plot.crop_name}
                        </span>
                        <Tag size='small' color='red'>{plot.event_type === 'drought' ? '🏜️ ' + t('干旱') : '🐛 ' + t('虫害')}</Tag>
                      </div>
                      {plot.event_type === 'drought' ? (
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Text type='danger' size='small'>💀 {formatDuration(plot.death_remain)} {t('后死亡')}</Text>
                          <Button size='small' type='danger' icon={<Droplets size={11} />}
                            onClick={() => handleWater(plot.plot_index)} loading={actionLoading}
                            style={{ borderRadius: 6, fontSize: 12 }}>{t('浇水')}</Button>
                        </div>
                      ) : (
                        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                          <Button size='small' type='warning' icon={<Pill size={11} />}
                            onClick={() => handleTreat(plot.plot_index)} loading={actionLoading}
                            style={{ borderRadius: 6, fontSize: 12 }}>{t('治疗')}</Button>
                        </div>
                      )}
                    </div>
                  )}

                  {/* Wilting */}
                  {plot.status === 4 && (
                    <div>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
                        <span style={{ fontSize: 13, fontWeight: 600 }}>
                          🥀 {plot.plot_index + 1}{t('号地')} · {plot.crop_name}
                        </span>
                        <Tag size='small' color='orange'>{t('枯萎')}</Tag>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Text type='danger' size='small'>💀 {formatDuration(plot.death_remain)} {t('后死亡')}</Text>
                        <Button size='small' type='danger' icon={<Droplets size={11} />}
                          onClick={() => handleWater(plot.plot_index)} loading={actionLoading}
                          style={{ borderRadius: 6, fontSize: 12 }}>{t('浇水')}</Button>
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Empty plots - compact inline list */}
          {emptyPlots.length > 0 && (
            <div style={{
              display: 'flex', flexWrap: 'wrap', gap: 6, padding: '10px 14px',
              borderRadius: 10, border: '1px dashed var(--semi-color-border)', background: 'var(--semi-color-fill-0)',
            }}>
              <Text type='tertiary' size='small' style={{ lineHeight: '24px', marginRight: 4 }}>⬜ {t('空地')}:</Text>
              {emptyPlots.map((p) => (
                <Tag key={p.plot_index} size='small' color='default' style={{ borderRadius: 4 }}>
                  {p.plot_index + 1}{t('号')}
                </Tag>
              ))}
              <Text type='tertiary' size='small' style={{ lineHeight: '24px', marginLeft: 4 }}>
                ({emptyPlots.length}{t('块')} — {t('去种植页种菜')})
              </Text>
            </div>
          )}
        </>
      )}
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
      <Card className='!rounded-xl' bodyStyle={{ padding: '14px 16px' }}
        style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
        <Text strong size='small' style={{ display: 'block', marginBottom: 10 }}>🌱 {t('选择作物')}</Text>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))', gap: 8 }}>
          {crops.map((crop) => (
            <div key={crop.key}
              onClick={() => setSelectedCrop(crop.key)}
              style={{
                padding: '10px 12px', borderRadius: 8, cursor: 'pointer',
                border: `2px solid ${selectedCrop === crop.key ? 'var(--semi-color-primary)' : 'var(--semi-color-border)'}`,
                background: selectedCrop === crop.key ? 'var(--semi-color-primary-light-default)' : 'var(--semi-color-fill-0)',
                transition: 'all 0.15s',
              }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                <span style={{ fontSize: 22 }}>{crop.emoji}</span>
                <Text strong style={{ fontSize: 14 }}>{crop.name}</Text>
                <Tag size='small' color='green'>${crop.seed_cost?.toFixed(2)}</Tag>
              </div>
              <div style={{ display: 'flex', gap: 8, fontSize: 11, flexWrap: 'wrap' }}>
                <Text type='tertiary'>⏱{formatDuration(crop.grow_secs)}</Text>
                <Text type='tertiary'>📦1~{crop.max_yield}</Text>
                <Text type='tertiary'>🏆${crop.max_value?.toFixed(2)}</Text>
              </div>
            </div>
          ))}
        </div>
      </Card>

      {/* Selected crop detail + plot selection */}
      {activeCrop && (
        <Card className='!rounded-xl' bodyStyle={{ padding: '14px 16px' }}
          style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 10 }}>
            <span style={{ fontSize: 24 }}>{activeCrop.emoji}</span>
            <div>
              <Text strong>{activeCrop.name}</Text>
              <div style={{ fontSize: 12 }}>
                <Text type='tertiary'>
                  {t('种子')} ${activeCrop.seed_cost?.toFixed(2)} · {t('生长')} {formatDuration(activeCrop.grow_secs)} · {t('产量')} 1~{activeCrop.max_yield} · {t('最高')} ${activeCrop.max_value?.toFixed(2)}
                </Text>
              </div>
            </div>
          </div>
        </Card>
      )}

      {/* Plot selection */}
      <Card className='!rounded-xl' bodyStyle={{ padding: '14px 16px' }}
        style={{ border: '1px solid var(--semi-color-border)' }}>
        <Text strong size='small' style={{ display: 'block', marginBottom: 10 }}>📍 {t('选择空地种植')}</Text>
        {emptyPlots.length === 0 ? (
          <Empty description={t('没有空地了')} style={{ padding: '20px 0' }} />
        ) : (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {emptyPlots.map((plot) => (
              <Button key={plot.plot_index}
                theme={selectedCrop ? 'solid' : 'light'}
                disabled={!selectedCrop || actionLoading}
                loading={actionLoading}
                onClick={() => handlePlant(plot.plot_index)}
                style={{ borderRadius: 8, minWidth: 80 }}>
                ⬜ {plot.plot_index + 1}{t('号地')}
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
      {/* Balance inline */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12,
        padding: '8px 14px', borderRadius: 8, background: 'var(--semi-color-fill-0)',
        border: '1px solid var(--semi-color-border)',
      }}>
        <Text type='tertiary' size='small'>💰 {t('余额')}:</Text>
        <Text strong>{formatBalance(farmData?.balance)}</Text>
      </div>

      {/* Seeds table */}
      <Card className='!rounded-xl' bodyStyle={{ padding: '12px 14px' }}
        style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
        <Text strong size='small' style={{ display: 'block', marginBottom: 8 }}>🌱 {t('种子目录')}</Text>
        <Table dataSource={crops} pagination={false} size='small' columns={[
          { title: t('作物'), dataIndex: 'name', render: (_, r) => <span>{r.emoji} {r.name}</span>, width: 100 },
          { title: t('价格'), dataIndex: 'seed_cost', render: v => `$${v?.toFixed(2)}`, width: 80 },
          { title: t('时间'), dataIndex: 'grow_secs', render: v => formatDuration(v), width: 80 },
          { title: t('产量'), dataIndex: 'max_yield', render: (v, r) => `1~${v}×$${r.unit_price?.toFixed(2)}`, width: 120 },
          { title: t('最高'), dataIndex: 'max_value', render: v => <Tag size='small' color='green'>${v?.toFixed(2)}</Tag>, width: 80 },
        ]} />
      </Card>

      {/* Items */}
      <Card className='!rounded-xl' bodyStyle={{ padding: '12px 14px' }}
        style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
        <Text strong size='small' style={{ display: 'block', marginBottom: 8 }}>📦 {t('道具')}</Text>
        {(shopData?.items || []).map((item) => (
          <div key={item.key} style={{
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            padding: '8px 10px', borderRadius: 8, marginBottom: 4,
            border: '1px solid var(--semi-color-border)', background: 'var(--semi-color-fill-0)',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontSize: 18 }}>{item.emoji}</span>
              <Text strong size='small'>{item.name}</Text>
              <Text type='tertiary' size='small'>{item.desc}</Text>
            </div>
            <div style={{ display: 'flex', gap: 4 }}>
              {[1, 5, 10].map(qty => (
                <Button key={qty} size='small' theme={qty === 1 ? 'solid' : 'light'}
                  onClick={() => handleBuyItem(item.key, qty)}
                  loading={actionLoading} style={{ borderRadius: 6, minWidth: 60 }}>
                  ×{qty} ${(item.cost * qty)?.toFixed(2)}
                </Button>
              ))}
            </div>
          </div>
        ))}
      </Card>

      {/* Dog purchase */}
      {shopData && !shopData.has_dog && (
        <Card className='!rounded-xl' bodyStyle={{ padding: '12px 14px' }}
          style={{ border: '1px solid var(--semi-color-border)' }}>
          <div style={{
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            padding: '8px 10px', borderRadius: 8,
            border: '1px solid var(--semi-color-border)', background: 'var(--semi-color-fill-0)',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontSize: 18 }}>🐶</span>
              <Text strong size='small'>{t('小狗')}</Text>
              <Text type='tertiary' size='small'>{t('长大后拦截偷菜')}</Text>
            </div>
            <Button size='small' theme='solid' onClick={handleBuyDog}
              loading={actionLoading} style={{ borderRadius: 6 }}>
              ${shopData.dog_price?.toFixed(2)}
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
};

// ===================== Sub-page: Bank =====================
const BankPage = ({ farmData, actionLoading, doAction, loadFarm, t }) => {
  const [bankData, setBankData] = useState(null);
  const [bankLoading, setBankLoading] = useState(true);
  const [mortgageAmount, setMortgageAmount] = useState(100);

  const loadBank = useCallback(async () => {
    setBankLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/bank');
      if (res.success) setBankData(res.data);
      else showError(res.message);
    } catch (err) { /* ignore */ }
    finally { setBankLoading(false); }
  }, []);

  useEffect(() => { loadBank(); }, [loadBank]);

  const handleLoan = async () => {
    const res = await doAction('/api/farm/bank/loan', {});
    if (res) { loadBank(); loadFarm(); }
  };

  const mortgageMaxDollar = bankData ? Math.floor(bankData.mortgage_max) : 1000;

  const handleMortgage = async () => {
    if (!mortgageAmount || mortgageAmount < 1 || mortgageAmount > mortgageMaxDollar) {
      showError(t('金额必须在 $1 ~ $') + mortgageMaxDollar + t(' 之间'));
      return;
    }
    const res = await doAction('/api/farm/bank/mortgage', { amount: mortgageAmount });
    if (res) { loadBank(); loadFarm(); }
  };

  const handleRepay = async (percent) => {
    const res = await doAction('/api/farm/bank/repay', { percent });
    if (res) { loadBank(); loadFarm(); }
  };

  if (bankLoading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }

  if (!bankData) {
    return <Empty description={t('银行功能不可用')} />;
  }

  const loan = bankData.active_loan;
  const history = bankData.history || [];

  return (
    <div>
      {/* Mortgage blocked warning */}
      {bankData.mortgage_blocked && (
        <Banner type='danger' description={t('由于抵押贷款违约，你已被永久禁止升级到10级及以上等级')}
          style={{ marginBottom: 12, borderRadius: 8 }} />
      )}

      {/* Balance & Credit info */}
      <Card className='!rounded-xl' bodyStyle={{ padding: '12px 14px' }}
        style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
        <Text strong size='small' style={{ display: 'block', marginBottom: 8 }}>🏦 {t('银行信息')}</Text>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginBottom: 8 }}>
          <Tag size='large' color='light-blue' style={{ borderRadius: 6 }}>💰 {t('余额')}: {formatBalance(bankData.balance)}</Tag>
          <Tag size='large' color='cyan' style={{ borderRadius: 6 }}>📊 {t('信用评分')}: {bankData.credit_score}/{bankData.max_score}</Tag>
        </div>
        <Descriptions size='small' row data={[
          { key: t('信用贷额度'), value: formatBalance(bankData.max_loan) },
          { key: t('信用贷利率'), value: `${bankData.interest_rate}%` },
          { key: t('抵押贷上限'), value: formatBalance(bankData.mortgage_max) },
          { key: t('抵押贷利率'), value: `${bankData.mortgage_interest_rate}%` },
          { key: t('还款期限'), value: `${bankData.loan_days} ${t('天')}` },
        ]} />
      </Card>

      {/* Active loan or apply */}
      <Card className='!rounded-xl' bodyStyle={{ padding: '12px 14px' }}
        style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
        {bankData.has_active_loan && loan ? (
          <div>
            <Text strong size='small' style={{ display: 'block', marginBottom: 8 }}>
              📋 {t('当前贷款')} {loan.loan_type === 1 ? <Tag size='small' color='orange'>🏠 {t('抵押')}</Tag> : <Tag size='small' color='blue'>{t('信用')}</Tag>}
            </Text>
            {loan.overdue && (
              <Banner type='danger' description={loan.loan_type === 1 ? t('抵押贷款已逾期！逾期将执行惩罚！') : t('贷款已逾期！请尽快还款')}
                style={{ marginBottom: 8, borderRadius: 8 }} />
            )}
            <Descriptions size='small' row data={[
              { key: t('本金'), value: formatBalance(loan.principal) },
              { key: t('利息'), value: formatBalance(loan.interest) },
              { key: t('应还'), value: formatBalance(loan.total_due) },
              { key: t('已还'), value: formatBalance(loan.repaid) },
              { key: t('剩余'), value: <Text type='danger' strong>{formatBalance(loan.remaining)}</Text> },
              { key: t('剩余天数'), value: loan.overdue ? <Tag color='red'>{t('已逾期')}</Tag> : `${loan.days_left} ${t('天')}` },
            ]} />
            <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
              <Button theme='solid' type='primary' onClick={() => handleRepay(100)}
                loading={actionLoading} style={{ borderRadius: 8 }}>
                💰 {t('全额还款')} ({formatBalance(loan.remaining)})
              </Button>
              {loan.remaining > 0.01 && (
                <Button theme='light' type='primary' onClick={() => handleRepay(50)}
                  loading={actionLoading} style={{ borderRadius: 8 }}>
                  💰 {t('还一半')} ({formatBalance(loan.remaining / 2)})
                </Button>
              )}
            </div>
          </div>
        ) : (
          <div>
            <Text strong size='small' style={{ display: 'block', marginBottom: 8 }}>✅ {t('当前无贷款')}</Text>

            {/* Credit loan */}
            <div style={{ padding: '10px 12px', borderRadius: 8, border: '1px solid var(--semi-color-border)',
              background: 'var(--semi-color-fill-0)', marginBottom: 10 }}>
              <Text strong size='small' style={{ display: 'block', marginBottom: 6 }}>💵 {t('信用贷款')}</Text>
              <Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 8 }}>
                {t('额度')} {formatBalance(bankData.max_loan)}，{t('利息')} {formatBalance(bankData.interest)}，{t('期限')} {bankData.loan_days}{t('天')}
              </Text>
              <Button theme='solid' type='primary' onClick={handleLoan}
                loading={actionLoading} style={{ borderRadius: 8 }}>
                💵 {t('申请信用贷款')} ({formatBalance(bankData.max_loan)})
              </Button>
            </div>

            {/* Mortgage loan */}
            <div style={{ padding: '10px 12px', borderRadius: 8, border: '1px solid var(--semi-color-border)',
              background: 'var(--semi-color-fill-0)' }}>
              <Text strong size='small' style={{ display: 'block', marginBottom: 6 }}>🏠 {t('抵押贷款')}</Text>
              <Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 4 }}>
                {t('以10级升级权为抵押，最高')} {formatBalance(bankData.mortgage_max)}，{t('利率')} {bankData.mortgage_interest_rate}%
              </Text>
              <Banner type='warning' style={{ marginBottom: 8, borderRadius: 6 }}
                description={t('抵押贷款不能用于升级！逾期未还：10级以下永久禁升10级，10级以上封禁账号')} />
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <Text size='small'>$</Text>
                <InputNumber value={mortgageAmount} onChange={setMortgageAmount}
                  min={1} max={mortgageMaxDollar} style={{ width: 120 }} />
                <Button theme='solid' type='warning' onClick={handleMortgage}
                  loading={actionLoading} style={{ borderRadius: 8 }}>
                  🏠 {t('申请抵押贷款')}
                </Button>
              </div>
            </div>
          </div>
        )}
      </Card>

      {/* Loan history */}
      {history.length > 0 && (
        <Card className='!rounded-xl' bodyStyle={{ padding: '12px 14px' }}
          style={{ border: '1px solid var(--semi-color-border)' }}>
          <Text strong size='small' style={{ display: 'block', marginBottom: 8 }}>📜 {t('贷款历史')}</Text>
          <Table dataSource={history} pagination={false} size='small' columns={[
            { title: t('日期'), dataIndex: 'created_at', width: 100,
              render: v => new Date(v * 1000).toLocaleDateString() },
            { title: t('类型'), dataIndex: 'loan_type', width: 60,
              render: v => v === 1
                ? <Tag size='small' color='orange'>{t('抵押')}</Tag>
                : <Tag size='small' color='blue'>{t('信用')}</Tag> },
            { title: t('本金'), dataIndex: 'principal', width: 90,
              render: v => formatBalance(v) },
            { title: t('应还'), dataIndex: 'total_due', width: 90,
              render: v => formatBalance(v) },
            { title: t('已还'), dataIndex: 'repaid', width: 90,
              render: v => formatBalance(v) },
            { title: t('状态'), dataIndex: 'status', width: 70,
              render: v => v === 1
                ? <Tag size='small' color='green'>{t('已还清')}</Tag>
                : v === 2
                  ? <Tag size='small' color='red'>{t('违约')}</Tag>
                  : <Tag size='small' color='orange'>{t('还款中')}</Tag> },
          ]} />
        </Card>
      )}
    </div>
  );
};

// ===================== Sub-page: Warehouse =====================
const seasonNames = ['春', '夏', '秋', '冬'];
const seasonEmojis = ['🌸', '☀️', '🍂', '❄️'];

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

  if (whLoading) return <Spin />;
  if (!whData) return <Empty description={t('加载失败')} />;

  const items = whData.items || [];
  const season = whData.season ?? 0;

  return (
    <div>
      <Card style={{ marginBottom: 12 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <Text strong>{seasonEmojis[season]} {t('当前')}: {seasonNames[season]}{t('季')} ({t('剩余')} {whData.days_left} {t('天')})</Text>
          <Tag color='blue'>{t('容量')}: {whData.total}/{whData.max_slots}</Tag>
        </div>
        <Banner type='info' style={{ marginBottom: 8, borderRadius: 6 }}
          description={t('应季作物价格低，反季价格高。建议应季存入仓库，等反季再出售！')} />
      </Card>

      {items.length === 0 ? (
        <Empty description={t('仓库空空如也，收获时选择「收获到仓库」来存储作物')} />
      ) : (
        <>
          <Table dataSource={items} pagination={false} size='small' rowKey='crop_key'
            columns={[
              { title: t('作物'), dataIndex: 'crop_name', render: (_, r) => `${r.emoji} ${r.crop_name}` },
              { title: t('数量'), dataIndex: 'quantity' },
              { title: t('单价'), dataIndex: 'unit_price', render: v => `$${v.toFixed(2)}` },
              { title: t('季节'), dataIndex: 'in_season', render: (v, r) => (
                <Tag color={v ? 'green' : 'orange'}>{v ? t('应季') : t('反季')} {r.season_pct}%</Tag>
              )},
              { title: t('总值'), dataIndex: 'total_value', render: v => `$${v.toFixed(2)}` },
              { title: t('操作'), render: (_, r) => (
                <Button size='small' theme='solid' type='warning' onClick={() => handleSell(r.crop_key)}
                  loading={actionLoading}>{t('出售')}</Button>
              )},
            ]}
          />
          <div style={{ marginTop: 8, textAlign: 'right' }}>
            <Button theme='solid' type='warning' onClick={handleSellAll} loading={actionLoading}>
              💰 {t('全部出售')}
            </Button>
          </div>
        </>
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
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }

  return (
    <div>
      {/* Steal history */}
      {stealResults.length > 0 && (
        <Card className='!rounded-xl' bodyStyle={{ padding: '10px 14px' }}
          style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
          <Text strong size='small' style={{ display: 'block', marginBottom: 6 }}>📜 {t('偷菜记录')}</Text>
          {stealResults.map((r, i) => (
            <div key={i} style={{
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              padding: '6px 10px', borderRadius: 6, marginBottom: 4, background: 'var(--semi-color-fill-0)',
            }}>
              <Text size='small'><Text type='tertiary' size='small'>{r.time}</Text> {r.message}</Text>
              {r.data && <Tag size='small' color='green'>${r.data.value?.toFixed(2)}</Tag>}
            </div>
          ))}
        </Card>
      )}

      {/* Targets */}
      <Card className='!rounded-xl' bodyStyle={{ padding: '10px 14px' }}
        style={{ border: '1px solid var(--semi-color-border)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <Text strong size='small'>🕵️ {t('可偷菜的农场')}</Text>
          <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadTargets} loading={stealLoading} />
        </div>

        {targets.length === 0 ? (
          <Empty description={t('暂时没有可偷的菜地')} style={{ padding: '20px 0' }} />
        ) : (
          <div>
            {targets.map((target) => (
              <div key={target.id} style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                padding: '8px 10px', borderRadius: 8, marginBottom: 4,
                border: '1px solid var(--semi-color-border)', background: 'var(--semi-color-fill-0)',
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Text strong size='small'>👤 {target.label}</Text>
                  <Tag size='small' color='green'>{target.count}{t('块')}</Tag>
                </div>
                <Button size='small' type='warning' theme='solid' onClick={() => handleSteal(target.id)}
                  loading={actionLoading} style={{ borderRadius: 6 }}>
                  {t('偷菜')}
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
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }

  if (!dogData || !dogData.has_dog) {
    return (
      <Card className='!rounded-xl' bodyStyle={{ padding: '20px 24px' }}
        style={{ border: '1px solid var(--semi-color-border)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 16 }}>
          <span style={{ fontSize: 36 }}>🐶</span>
          <div>
            <Text strong style={{ fontSize: 15 }}>{t('你还没有看门狗')}</Text>
            <div><Text type='tertiary' size='small'>{t('养大后可拦截偷菜者')}</Text></div>
          </div>
        </div>
        <div style={{
          display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8, marginBottom: 16,
          padding: '10px 12px', borderRadius: 8, background: 'var(--semi-color-fill-0)',
        }}>
          <div style={{ textAlign: 'center' }}>
            <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('价格')}</Text>
            <Text strong size='small'>${dogData?.dog_price?.toFixed(2)}</Text>
          </div>
          <div style={{ textAlign: 'center' }}>
            <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('成长')}</Text>
            <Text strong size='small'>{dogData?.grow_hours}{t('小时')}</Text>
          </div>
          <div style={{ textAlign: 'center' }}>
            <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('拦截率')}</Text>
            <Text strong size='small'>{dogData?.guard_rate}%</Text>
          </div>
          <div style={{ textAlign: 'center' }}>
            <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('狗粮')}</Text>
            <Text strong size='small'>${dogData?.food_price?.toFixed(2)}</Text>
          </div>
        </div>
        <Button theme='solid' onClick={handleBuyDog} loading={actionLoading} style={{ borderRadius: 8 }}>
          🐶 {t('购买小狗')} (${dogData?.dog_price?.toFixed(2)})
        </Button>
      </Card>
    );
  }

  return (
    <Card className='!rounded-xl' bodyStyle={{ padding: '16px 20px' }}
      style={{ border: '1px solid var(--semi-color-border)' }}>
      {/* Profile row */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 14 }}>
        <span style={{ fontSize: 36 }}>{dogData.level === 2 ? '🐕' : '🐶'}</span>
        <div style={{ flex: 1 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Text strong style={{ fontSize: 16 }}>「{dogData.name}」</Text>
            <Tag size='small' color={dogData.hunger > 0 ? 'green' : 'red'}>
              {dogData.level_name} · {dogData.status}
            </Tag>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
            <Text type='tertiary' size='small'>{t('饱食度')}</Text>
            <div style={{ flex: 1, maxWidth: 180 }}>
              <Progress percent={dogData.hunger} size='small'
                stroke={dogData.hunger > 30 ? '#22c55e' : '#ef4444'} />
            </div>
            <Text strong size='small'>{dogData.hunger}%</Text>
          </div>
        </div>
        {dogData.hunger < 100 && (
          <Button size='small' theme='solid' onClick={handleFeedDog} loading={actionLoading}
            style={{ borderRadius: 6 }}>
            🦴 {t('喂食')}
          </Button>
        )}
      </div>

      {/* Stats row */}
      <div style={{
        display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8,
        padding: '10px 12px', borderRadius: 8, background: 'var(--semi-color-fill-0)', marginBottom: 10,
      }}>
        <div style={{ textAlign: 'center' }}>
          <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('等级')}</Text>
          <Text strong size='small'>{dogData.level_name}</Text>
        </div>
        <div style={{ textAlign: 'center' }}>
          <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('饱食度')}</Text>
          <Text strong size='small'>{dogData.hunger}%</Text>
        </div>
        <div style={{ textAlign: 'center' }}>
          <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('拦截率')}</Text>
          <Text strong size='small'>{dogData.guard_rate}%</Text>
        </div>
        <div style={{ textAlign: 'center' }}>
          <Text type='tertiary' size='small' style={{ display: 'block' }}>{t('狗粮')}</Text>
          <Text strong size='small'>${dogData.food_price?.toFixed(2)}</Text>
        </div>
      </div>

      <Text type='tertiary' size='small'>
        💡 {t('饱食度为0时无法看门，每小时-1点，请定期喂食狗粮')}
      </Text>
    </Card>
  );
};

// ===================== Sub-page: Ranch =====================
const RanchPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [ranchData, setRanchData] = useState(null);
  const [ranchLoading, setRanchLoading] = useState(true);

  const loadRanch = useCallback(async () => {
    setRanchLoading(true);
    try {
      const { data: res } = await API.get('/api/ranch/view');
      if (res.success) setRanchData(res.data);
    } catch (err) { /* ignore */ }
    finally { setRanchLoading(false); }
  }, []);

  useEffect(() => { loadRanch(); }, [loadRanch]);

  useEffect(() => {
    const interval = setInterval(loadRanch, 15000);
    return () => clearInterval(interval);
  }, [loadRanch]);

  const doRanchAction = async (url, body) => {
    const res = await doAction(url, body);
    if (res) { loadRanch(); loadFarm(); }
    return res;
  };

  if (ranchLoading && !ranchData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!ranchData) return null;

  const animals = ranchData.animals || [];
  const animalTypes = ranchData.animal_types || [];
  const deadAnimals = animals.filter(a => a.status === 5);
  const aliveAnimals = animals.filter(a => a.status !== 5);
  const dirtyAnimals = aliveAnimals.filter(a => a.is_dirty);

  const statusLabels = { 1: '生长中', 2: '已成熟', 3: '饥饿', 4: '口渴', 5: '已死亡' };
  const statusTagColors = { 1: 'blue', 2: 'green', 3: 'orange', 4: 'red', 5: 'grey' };

  return (
    <div>
      {/* Status bar */}
      <div style={{
        display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10,
        padding: '10px 14px', borderRadius: 10, marginBottom: 12,
        background: 'var(--semi-color-fill-0)', border: '1px solid var(--semi-color-border)',
      }}>
        <Tag size='large' color='light-blue' style={{ borderRadius: 6 }}>💰 {formatBalance(ranchData.balance)}</Tag>
        <Tag size='large' color='grey' style={{ borderRadius: 6 }}>🐄 {ranchData.alive_count}/{ranchData.max_animals}</Tag>
        <Tag size='large' color='cyan' style={{ borderRadius: 6 }}>🌾 {formatBalance(ranchData.feed_price)}/次</Tag>
        <Tag size='large' color='blue' style={{ borderRadius: 6 }}>💧 {formatBalance(ranchData.water_price)}/次</Tag>
        <div style={{ flex: 1 }} />
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadRanch} loading={ranchLoading} />
        {dirtyAnimals.length > 0 && (
          <Button size='small' theme='light' onClick={() => doRanchAction('/api/ranch/clean', {})}
            loading={actionLoading} style={{ borderRadius: 6, color: '#92400e', borderColor: '#f59e0b' }}>
            🧹 {t('清理粪便')}({formatBalance(ranchData.manure_clean_price)})
          </Button>
        )}
        {deadAnimals.length > 0 && (
          <Button size='small' theme='light' type='danger' onClick={() => doRanchAction('/api/ranch/cleanup', {})}
            loading={actionLoading} style={{ borderRadius: 6 }}>
            🗑️ {t('清理')}({deadAnimals.length})
          </Button>
        )}
      </div>

      {/* Animal list */}
      {animals.length === 0 ? (
        <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', textAlign: 'center', padding: 30, marginBottom: 12 }}>
          <span style={{ fontSize: 36 }}>🏚️</span>
          <Title heading={6} style={{ marginTop: 8 }}>{t('牧场空空如也')}</Title>
          <Text type='tertiary' size='small'>{t('去下方购买动物开始养殖吧！')}</Text>
        </Card>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 12 }}>
          {animals.map((animal) => (
            <Card key={animal.id} className='!rounded-xl' bodyStyle={{ padding: '12px 16px' }}
              style={{ border: '1px solid var(--semi-color-border)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <span style={{ fontSize: 32 }}>{animal.animal_emoji}</span>
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                    <Text strong style={{ fontSize: 15 }}>{animal.animal_name}</Text>
                    <Tag size='small' color={statusTagColors[animal.status] || 'grey'}>
                      {statusLabels[animal.status] || animal.status_label}
                    </Tag>
                    {animal.needs_feed && animal.status !== 5 && (
                      <Tag size='small' color='orange'>⚠️ {t('需喂食')}</Tag>
                    )}
                    {animal.needs_water && animal.status !== 5 && (
                      <Tag size='small' color='red'>⚠️ {t('需喂水')}</Tag>
                    )}
                    {animal.is_dirty && animal.status !== 5 && (
                      <Tag size='small' color='amber'>💩 {t('脏污')}</Tag>
                    )}
                  </div>
                  {animal.status === 1 && (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <div style={{ flex: 1, maxWidth: 200 }}>
                        <Progress percent={animal.progress} size='small' stroke='#3b82f6' />
                      </div>
                      <Text type='tertiary' size='small'>{animal.progress}% · {formatDuration(animal.remaining)}</Text>
                    </div>
                  )}
                  {animal.status === 2 && (
                    <Text type='success' size='small'>🥩 {t('肉价')} {formatBalance(animal.meat_price)}</Text>
                  )}
                  {(animal.status === 1 || animal.status === 2) && (
                    <div style={{ display: 'flex', gap: 8, marginTop: 2 }}>
                      {!animal.needs_feed && animal.feed_remaining > 0 && (
                        <Text type='tertiary' size='small'>🌾 {formatDuration(animal.feed_remaining)}</Text>
                      )}
                      {!animal.needs_water && animal.water_remaining > 0 && (
                        <Text type='tertiary' size='small'>💧 {formatDuration(animal.water_remaining)}</Text>
                      )}
                      {!animal.is_dirty && animal.clean_remaining > 0 && (
                        <Text type='tertiary' size='small'>🧹 {formatDuration(animal.clean_remaining)}</Text>
                      )}
                    </div>
                  )}
                </div>
                {/* Action buttons */}
                <div style={{ display: 'flex', gap: 4, flexShrink: 0 }}>
                  {animal.status !== 5 && (
                    <>
                      <Button size='small' theme='light' onClick={() => doRanchAction('/api/ranch/feed', { animal_id: animal.id })}
                        loading={actionLoading} style={{ borderRadius: 6 }} disabled={!animal.needs_feed}>
                        🌾
                      </Button>
                      <Button size='small' theme='light' onClick={() => doRanchAction('/api/ranch/water', { animal_id: animal.id })}
                        loading={actionLoading} style={{ borderRadius: 6 }} disabled={!animal.needs_water}>
                        💧
                      </Button>
                    </>
                  )}
                  {animal.status === 2 && (
                    <Button size='small' theme='solid' type='warning'
                      onClick={() => doRanchAction('/api/ranch/slaughter', { animal_id: animal.id })}
                      loading={actionLoading} style={{ borderRadius: 6 }}>
                      🔪 {t('出售')}
                    </Button>
                  )}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Buy animals section */}
      {aliveAnimals.length < ranchData.max_animals && (
        <Card className='!rounded-xl' bodyStyle={{ padding: '14px 18px' }}
          style={{ border: '1px solid var(--semi-color-border)' }}
          title={<span>🛒 {t('购买动物')}</span>}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: 8 }}>
            {animalTypes.map((at) => (
              <Card key={at.key} className='!rounded-lg' bodyStyle={{ padding: '10px 12px', textAlign: 'center' }}
                style={{ border: '1px solid var(--semi-color-border)', cursor: 'pointer' }}
                onClick={() => doRanchAction('/api/ranch/buy', { animal_type: at.key })}>
                <span style={{ fontSize: 28, display: 'block', marginBottom: 4 }}>{at.emoji}</span>
                <Text strong size='small' style={{ display: 'block' }}>{at.name}</Text>
                <Text type='tertiary' size='small' style={{ display: 'block' }}>{formatBalance(at.buy_price)}</Text>
                <Text type='tertiary' size='small' style={{ display: 'block' }}>⏱️ {Math.round(at.grow_secs / 3600)}h</Text>
                <Text type='success' size='small' style={{ display: 'block' }}>🥩 {formatBalance(at.meat_price)}</Text>
              </Card>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
};

// ===================== Level Page =====================
const LevelPage = ({ actionLoading, loadFarm, t }) => {
  const [lvData, setLvData] = useState(null);
  const [loading, setLoading] = useState(false);

  const loadLevel = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/level');
      if (res.success) setLvData(res.data);
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => { loadLevel(); }, [loadLevel]);

  const doLevelUp = async () => {
    setLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/levelup');
      if (res.success) {
        showSuccess(res.message);
        loadLevel();
        loadFarm();
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setLoading(false);
    }
  };

  if (loading && !lvData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!lvData) return null;

  const isMax = lvData.level >= lvData.max_level;
  const pct = Math.round((lvData.level / lvData.max_level) * 100);

  return (
    <div>
      {/* Level header */}
      <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontSize: 36 }}>⭐</span>
          <div style={{ flex: 1 }}>
            <Title heading={4} style={{ margin: 0 }}>Lv.{lvData.level}</Title>
            <div style={{ width: '100%', height: 8, background: 'var(--semi-color-fill-2)', borderRadius: 4, marginTop: 4 }}>
              <div style={{ width: `${pct}%`, height: '100%', background: isMax ? '#16a34a' : '#f59e0b', borderRadius: 4 }} />
            </div>
            <Text size='small' type='tertiary'>{lvData.level}/{lvData.max_level}</Text>
          </div>
          {!isMax && (
            <Button theme='solid' type='warning' loading={loading} onClick={doLevelUp}>
              ⬆️ {t('升级')} ${lvData.next_price.toFixed(2)}
            </Button>
          )}
          {isMax && <Tag size='large' color='green'>MAX</Tag>}
        </div>
      </Card>

      {/* Feature unlocks */}
      <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
        <Text strong style={{ display: 'block', marginBottom: 8 }}>🔓 {t('功能解锁')}</Text>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {(lvData.unlocks || []).map((u) => (
            <div key={u.key} style={{
              display: 'flex', alignItems: 'center', gap: 8, padding: '4px 8px',
              borderRadius: 6, background: u.unlocked ? 'var(--semi-color-success-light-default)' : 'var(--semi-color-fill-0)',
            }}>
              <span>{u.unlocked ? '✅' : '🔒'}</span>
              <Text style={{ flex: 1 }}>{u.name}</Text>
              <Tag size='small' color={u.unlocked ? 'green' : 'grey'}>Lv.{u.level}</Tag>
            </div>
          ))}
        </div>
      </Card>

      {/* Price table */}
      <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)' }}>
        <Text strong style={{ display: 'block', marginBottom: 8 }}>📊 {t('升级价格表')}</Text>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {(lvData.prices || []).map((p) => (
            <div key={p.level} style={{
              display: 'flex', alignItems: 'center', gap: 8, padding: '3px 8px',
              borderRadius: 6,
              background: p.level === lvData.level + 1 ? 'var(--semi-color-warning-light-default)' : 'transparent',
              fontWeight: p.level === lvData.level + 1 ? 600 : 400,
            }}>
              <Text style={{ width: 50 }}>Lv.{p.level}</Text>
              <Text>${p.price.toFixed(2)}</Text>
              {p.level <= lvData.level && <Tag size='small' color='green' style={{ marginLeft: 'auto' }}>✅</Tag>}
              {p.level === lvData.level + 1 && <Tag size='small' color='orange' style={{ marginLeft: 'auto' }}>👉 {t('下一级')}</Tag>}
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
};

// ===================== Tasks Page =====================
const TasksPage = ({ actionLoading, loadFarm, t }) => {
  const [taskData, setTaskData] = useState(null);
  const [loading, setLoading] = useState(false);

  const loadTasks = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/tasks');
      if (res.success) setTaskData(res.data);
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => { loadTasks(); }, [loadTasks]);

  const claimTask = async (index) => {
    try {
      const { data: res } = await API.post('/api/farm/tasks/claim', { index });
      if (res.success) {
        showSuccess(res.message);
        loadTasks();
        loadFarm();
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    }
  };

  if (loading && !taskData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!taskData) return null;

  const dateStr = taskData.date || '';
  const dateDisplay = dateStr.length === 8 ? `${dateStr.slice(0,4)}-${dateStr.slice(4,6)}-${dateStr.slice(6)}` : dateStr;

  return (
    <div>
      <Tag size='large' color='blue' style={{ marginBottom: 12 }}>📅 {dateDisplay}</Tag>
      <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
        <Text strong style={{ display: 'block', marginBottom: 10 }}>📝 {t('今日任务')}</Text>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {(taskData.tasks || []).map((task) => (
            <div key={task.index} style={{
              padding: '10px 14px', borderRadius: 8,
              background: task.claimed ? 'var(--semi-color-success-light-default)' : 'var(--semi-color-fill-0)',
              border: '1px solid var(--semi-color-border)',
              display: 'flex', alignItems: 'center', gap: 10,
            }}>
              <span style={{ fontSize: 22 }}>{task.emoji}</span>
              <div style={{ flex: 1 }}>
                <Text strong>{task.name}</Text>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
                  <div style={{ width: 80, height: 6, background: 'var(--semi-color-fill-2)', borderRadius: 3 }}>
                    <div style={{
                      width: `${Math.min(100, (task.progress / task.target) * 100)}%`,
                      height: '100%',
                      background: task.claimed ? '#16a34a' : task.done ? '#eab308' : '#3b82f6',
                      borderRadius: 3,
                    }} />
                  </div>
                  <Text size='small' type='tertiary'>{task.progress}/{task.target}</Text>
                </div>
                <Text size='small' type='tertiary'>{t('奖励')}: ${task.reward.toFixed(2)}</Text>
              </div>
              {task.claimed ? (
                <Tag size='small' color='green'>✅</Tag>
              ) : task.done ? (
                <Button size='small' theme='solid' type='warning' onClick={() => claimTask(task.index)}>
                  {t('领取')}
                </Button>
              ) : (
                <Tag size='small' color='grey'>{t('未完成')}</Tag>
              )}
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
};

// ===================== Achievements Page =====================
const AchievementsPage = ({ actionLoading, loadFarm, t }) => {
  const [achData, setAchData] = useState(null);
  const [loading, setLoading] = useState(false);

  const loadAch = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/achievements');
      if (res.success) setAchData(res.data);
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => { loadAch(); }, [loadAch]);

  const claimAch = async (key) => {
    try {
      const { data: res } = await API.post('/api/farm/achievements/claim', { key });
      if (res.success) {
        showSuccess(res.message);
        loadAch();
        loadFarm();
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    }
  };

  if (loading && !achData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!achData) return null;

  const unlockCount = (achData.achievements || []).filter(a => a.unlocked).length;
  const totalCount = (achData.achievements || []).length;

  return (
    <div>
      <Tag size='large' color='blue' style={{ marginBottom: 12 }}>🏆 {unlockCount}/{totalCount} {t('已解锁')}</Tag>
      <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {(achData.achievements || []).map((ach) => (
            <div key={ach.key} style={{
              padding: '10px 14px', borderRadius: 8,
              background: ach.unlocked ? 'var(--semi-color-success-light-default)' : 'var(--semi-color-fill-0)',
              border: '1px solid var(--semi-color-border)',
              display: 'flex', alignItems: 'center', gap: 10,
              opacity: ach.unlocked ? 1 : ach.done ? 1 : 0.7,
            }}>
              <span style={{ fontSize: 24 }}>{ach.emoji}</span>
              <div style={{ flex: 1 }}>
                <Text strong>{ach.name}</Text>
                <Text size='small' type='tertiary' style={{ display: 'block' }}>{ach.description}</Text>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
                  <div style={{ width: 80, height: 6, background: 'var(--semi-color-fill-2)', borderRadius: 3 }}>
                    <div style={{
                      width: `${Math.min(100, (ach.progress / ach.target) * 100)}%`,
                      height: '100%',
                      background: ach.unlocked ? '#16a34a' : ach.done ? '#eab308' : '#3b82f6',
                      borderRadius: 3,
                    }} />
                  </div>
                  <Text size='small' type='tertiary'>{ach.progress}/{ach.target}</Text>
                  <Text size='small' type='tertiary'>· ${ach.reward.toFixed(2)}</Text>
                </div>
              </div>
              {ach.unlocked ? (
                <Tag size='small' color='green'>✅</Tag>
              ) : ach.done ? (
                <Button size='small' theme='solid' type='warning' onClick={() => claimAch(ach.key)}>
                  {t('领取')}
                </Button>
              ) : null}
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
};

// ===================== Workshop Page =====================
const WorkshopPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [wsData, setWsData] = useState(null);
  const [wsLoading, setWsLoading] = useState(false);
  const [tick, setTick] = useState(0);

  const loadWorkshop = useCallback(async () => {
    setWsLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/workshop');
      if (res.success) setWsData(res.data);
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setWsLoading(false);
    }
  }, [t]);

  useEffect(() => { loadWorkshop(); }, [loadWorkshop]);

  // Auto-refresh every 5s for progress
  useEffect(() => {
    const timer = setInterval(() => setTick(p => p + 1), 5000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    if (tick > 0) loadWorkshop();
  }, [tick, loadWorkshop]);

  const doCraft = async (key) => {
    setWsLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/workshop/craft', { recipe_key: key });
      if (res.success) {
        showSuccess(res.message);
        loadWorkshop();
        loadFarm();
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setWsLoading(false);
    }
  };

  const doCollect = async () => {
    setWsLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/workshop/collect');
      if (res.success) {
        showSuccess(res.message);
        loadWorkshop();
        loadFarm();
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setWsLoading(false);
    }
  };

  if (wsLoading && !wsData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!wsData) return null;

  const hasCollectable = (wsData.active || []).some(p => p.status === 2);
  const slotsAvailable = wsData.used_slots < wsData.max_slots;

  const profitColor = (v) => v >= 0 ? '#16a34a' : '#dc2626';

  return (
    <div>
      {/* Slots info */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 12, alignItems: 'center' }}>
        <Tag size='large' color='blue'>🏭 {t('槽位')}: {wsData.used_slots}/{wsData.max_slots}</Tag>
        {hasCollectable && (
          <Button theme='solid' type='warning' size='small' loading={wsLoading} onClick={doCollect}>
            📥 {t('收取全部')}
          </Button>
        )}
      </div>

      {/* Active processes */}
      {wsData.active && wsData.active.length > 0 && (
        <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
          <Text strong style={{ display: 'block', marginBottom: 8 }}>⏳ {t('加工中')}</Text>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {wsData.active.map((p) => (
              <div key={p.id} style={{
                padding: '8px 12px', borderRadius: 8,
                background: 'var(--semi-color-fill-0)',
                border: '1px solid var(--semi-color-border)',
                display: 'flex', alignItems: 'center', gap: 10,
              }}>
                <span style={{ fontSize: 24 }}>{p.emoji}</span>
                <div style={{ flex: 1 }}>
                  <Text strong>{p.name}</Text>
                  {p.status === 2 ? (
                    <Tag size='small' color='green' style={{ marginLeft: 6 }}>✅ {t('已完成')}</Tag>
                  ) : (
                    <Tag size='small' color='blue' style={{ marginLeft: 6 }}>{p.progress}% · {formatDuration(p.remaining)}</Tag>
                  )}
                  <Text size='small' type='tertiary' style={{ display: 'block' }}>{t('价值')}: ${p.sell_price.toFixed(2)}</Text>
                </div>
                {p.status === 1 && (
                  <div style={{ width: 80, height: 6, background: 'var(--semi-color-fill-2)', borderRadius: 3 }}>
                    <div style={{ width: `${p.progress}%`, height: '100%', background: '#3b82f6', borderRadius: 3 }} />
                  </div>
                )}
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Recipes */}
      <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)' }}>
        <Text strong style={{ display: 'block', marginBottom: 8 }}>📋 {t('配方列表')}</Text>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {(wsData.recipes || []).map((r) => (
            <div key={r.key} style={{
              padding: '8px 12px', borderRadius: 8,
              background: 'var(--semi-color-fill-0)',
              border: '1px solid var(--semi-color-border)',
              display: 'flex', alignItems: 'center', gap: 10,
            }}>
              <span style={{ fontSize: 22 }}>{r.emoji}</span>
              <div style={{ flex: 1 }}>
                <Text strong>{r.name}</Text>
                <Text size='small' type='tertiary' style={{ display: 'block' }}>
                  {t('成本')} ${r.cost.toFixed(2)} → {t('售价')} ${r.sell_price.toFixed(2)} ({r.multiplier}%)
                  · <span style={{ color: profitColor(r.profit), fontWeight: 600 }}>{r.profit >= 0 ? '+' : ''}${r.profit.toFixed(2)}</span>
                  · {formatDuration(r.time_secs)}
                </Text>
              </div>
              <Button size='small' theme='solid' disabled={!slotsAvailable || wsLoading}
                onClick={() => doCraft(r.key)}>
                {t('加工')}
              </Button>
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
};

// ===================== Market Chart =====================
const CHART_PALETTE = [
  '#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6', '#ec4899',
  '#14b8a6', '#f97316', '#06b6d4', '#84cc16', '#e11d48', '#6366f1',
];

const MarketChart = ({ t }) => {
  const [historyData, setHistoryData] = useState(null);
  const [chartCat, setChartCat] = useState('crop');
  const [loading, setLoading] = useState(false);
  const [visibleKeys, setVisibleKeys] = useState(null);

  const loadHistory = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/market/history');
      if (res.success) setHistoryData(res.data);
    } catch (e) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadHistory(); }, [loadHistory]);
  useEffect(() => { setVisibleKeys(null); }, [chartCat]);

  if (loading && !historyData) return <div style={{ textAlign: 'center', padding: 20 }}><Spin /></div>;
  if (!historyData || !historyData.history || historyData.history.length < 2) {
    return (
      <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
        <Text type='tertiary' style={{ display: 'block', textAlign: 'center', padding: 20 }}>
          📊 {t('市场需要至少刷新2次才能显示波动图')}
        </Text>
      </Card>
    );
  }

  const catItems = (historyData.items || []).filter(it => it.category === chartCat);
  const latestSnap = historyData.history[historyData.history.length - 1];

  const itemColorMap = {};
  catItems.forEach((it, idx) => {
    itemColorMap[it.key] = CHART_PALETTE[idx % CHART_PALETTE.length];
  });

  const sortedByVolatility = [...catItems].sort((a, b) => {
    const aD = Math.abs((latestSnap.prices?.[a.key] || 100) - 100);
    const bD = Math.abs((latestSnap.prices?.[b.key] || 100) - 100);
    return bD - aD;
  });

  const defaultTopN = Math.min(5, catItems.length);
  const defaultKeys = new Set(sortedByVolatility.slice(0, defaultTopN).map(it => it.key));
  const activeKeys = visibleKeys || defaultKeys;
  const displayItems = catItems.filter(it => activeKeys.has(it.key));

  const chartData = [];
  for (const snap of historyData.history) {
    const timeStr = new Date(snap.timestamp * 1000).toLocaleString(undefined, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
    for (const item of displayItems) {
      const val = snap.prices?.[item.key];
      if (val !== undefined) {
        chartData.push({ time: timeStr, name: item.emoji + item.name, value: val });
      }
    }
  }

  const displayColors = displayItems.map(it => itemColorMap[it.key]);

  const spec = {
    type: 'line',
    data: { values: chartData },
    xField: 'time',
    yField: 'value',
    seriesField: 'name',
    point: { visible: true, size: 5, style: { lineWidth: 1.5, stroke: '#fff' } },
    line: { style: { lineWidth: 2.5, lineCap: 'round' } },
    legends: { visible: false },
    crosshair: {
      xField: { visible: true, line: { style: { stroke: 'var(--semi-color-text-3)', lineWidth: 1, lineDash: [4, 4] } } },
    },
    axes: [
      { orient: 'left', title: { visible: true, text: t('倍率') + ' %' }, min: 0 },
      { orient: 'bottom', title: { visible: false }, label: { autoRotate: true, autoRotateAngle: [-45] } },
    ],
    markLine: [{
      y: 100,
      line: { style: { stroke: '#ef4444', lineWidth: 1.5, lineDash: [6, 4] } },
      label: { visible: true, text: '100%', style: { fill: '#ef4444', fontSize: 11, fontWeight: 'bold' } },
    }],
    tooltip: {
      dimension: {
        content: (data) => {
          const sorted = [...data].sort((a, b) => (b.datum?.value ?? b.value ?? 0) - (a.datum?.value ?? a.value ?? 0));
          return sorted.map(d => ({
            key: d.datum?.name || d.name,
            value: (d.datum?.value ?? d.value) + '%',
            hasShape: true,
            shapeType: 'circle',
          }));
        },
      },
    },
    color: displayColors,
    height: 420,
    padding: { left: 10, right: 10, top: 10, bottom: 10 },
    animation: false,
  };

  const toggleItem = (key) => {
    const base = new Set(activeKeys);
    if (base.has(key)) {
      if (base.size <= 1) return;
      base.delete(key);
    } else {
      base.add(key);
    }
    setVisibleKeys(base);
  };

  const selectPreset = (mode) => {
    if (mode === 'top5') {
      setVisibleKeys(null);
    } else if (mode === 'all') {
      setVisibleKeys(new Set(catItems.map(it => it.key)));
    } else if (mode === 'up') {
      const keys = catItems.filter(it => (latestSnap.prices?.[it.key] || 100) > 100).map(it => it.key);
      setVisibleKeys(new Set(keys.length ? keys : defaultKeys));
    } else if (mode === 'down') {
      const keys = catItems.filter(it => (latestSnap.prices?.[it.key] || 100) < 100).map(it => it.key);
      setVisibleKeys(new Set(keys.length ? keys : defaultKeys));
    }
  };

  const cats = [
    { key: 'crop', label: '🌾 ' + t('作物') },
    { key: 'fish', label: '🐟 ' + t('鱼类') },
    { key: 'meat', label: '🥩 ' + t('肉类') },
    { key: 'recipe', label: '🏭 ' + t('加工品') },
  ];

  return (
    <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
        <Text strong>📊 {t('市场波动图')}</Text>
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadHistory} loading={loading} />
      </div>
      <div style={{ display: 'flex', gap: 6, marginBottom: 8, flexWrap: 'wrap' }}>
        {cats.map(c => (
          <Tag key={c.key} size='large' color={chartCat === c.key ? 'blue' : 'grey'}
            style={{ cursor: 'pointer' }} onClick={() => setChartCat(c.key)}>
            {c.label}
          </Tag>
        ))}
      </div>
      <div style={{ display: 'flex', gap: 4, marginBottom: 8, flexWrap: 'wrap' }}>
        <Button size='small' theme={!visibleKeys ? 'solid' : 'light'} style={{ borderRadius: 6, fontSize: 12 }}
          onClick={() => selectPreset('top5')}>🔥 Top 5</Button>
        <Button size='small' theme={visibleKeys?.size === catItems.length ? 'solid' : 'light'} style={{ borderRadius: 6, fontSize: 12 }}
          onClick={() => selectPreset('all')}>{t('全部')}</Button>
        <Button size='small' theme='light' style={{ borderRadius: 6, fontSize: 12, color: '#22c55e' }}
          onClick={() => selectPreset('up')}>📈 {t('涨')}</Button>
        <Button size='small' theme='light' style={{ borderRadius: 6, fontSize: 12, color: '#ef4444' }}
          onClick={() => selectPreset('down')}>📉 {t('跌')}</Button>
      </div>
      <div style={{ display: 'flex', gap: 4, marginBottom: 10, flexWrap: 'wrap' }}>
        {catItems.map((it) => {
          const active = activeKeys.has(it.key);
          const clr = itemColorMap[it.key];
          const pct = latestSnap.prices?.[it.key];
          return (
            <div key={it.key} onClick={() => toggleItem(it.key)} style={{
              display: 'inline-flex', alignItems: 'center', gap: 4,
              padding: '2px 8px', borderRadius: 6, cursor: 'pointer', fontSize: 12,
              border: `1.5px solid ${active ? clr : 'var(--semi-color-border)'}`,
              background: active ? clr + '18' : 'transparent',
              opacity: active ? 1 : 0.45,
              transition: 'all 0.15s',
            }}>
              <span style={{
                width: 8, height: 8, borderRadius: '50%',
                background: active ? clr : 'var(--semi-color-text-3)',
                flexShrink: 0,
              }} />
              <span>{it.emoji}{it.name}</span>
              {pct !== undefined && (
                <span style={{ color: pct >= 100 ? '#22c55e' : '#ef4444', fontWeight: 600 }}>{pct}%</span>
              )}
            </div>
          );
        })}
      </div>
      {chartData.length > 0 ? (
        <VChart spec={spec} />
      ) : (
        <Text type='tertiary' style={{ display: 'block', textAlign: 'center', padding: 20 }}>
          {t('暂无数据')}
        </Text>
      )}
    </Card>
  );
};

// ===================== Market Page =====================
const MarketPage = ({ t }) => {
  const [marketData, setMarketData] = useState(null);
  const [marketLoading, setMarketLoading] = useState(false);
  const [countdown, setCountdown] = useState(0);

  const loadMarket = useCallback(async () => {
    setMarketLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/market');
      if (res.success) {
        setMarketData(res.data);
        setCountdown(res.data.next_refresh || 0);
      }
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setMarketLoading(false);
    }
  }, [t]);

  useEffect(() => { loadMarket(); }, [loadMarket]);

  useEffect(() => {
    if (countdown <= 0) return;
    const timer = setInterval(() => {
      setCountdown(prev => {
        if (prev <= 1) { clearInterval(timer); loadMarket(); return 0; }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [countdown, loadMarket]);

  if (marketLoading && !marketData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!marketData) return null;

  const formatCountdown = (s) => {
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    return h > 0 ? `${h}h${m}m` : `${m}m`;
  };

  const mColor = (m) => {
    if (m >= 150) return '#16a34a';
    if (m >= 120) return '#65a30d';
    if (m >= 80) return '#ca8a04';
    if (m >= 50) return '#dc2626';
    return '#991b1b';
  };

  const mLabel = (m) => {
    if (m >= 180) return '🔥暴涨';
    if (m >= 140) return '📈大涨';
    if (m >= 110) return '📈上涨';
    if (m >= 90) return '➡️平稳';
    if (m >= 60) return '📉下跌';
    return '📉暴跌';
  };

  const categories = [
    { key: 'crop', label: '🌾 ' + t('作物') },
    { key: 'fish', label: '🐟 ' + t('鱼类') },
    { key: 'meat', label: '🥩 ' + t('肉类') },
    { key: 'recipe', label: '🏭 ' + t('加工品') },
  ];

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <Tag size='large' color='blue'>⏱️ {t('下次刷新')}: {formatCountdown(countdown)}</Tag>
          <Tag size='large' color='grey'>🔄 {t('每')} {marketData.refresh_hours}h</Tag>
        </div>
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadMarket} loading={marketLoading} />
      </div>

      <MarketChart t={t} />

      {categories.map(cat => {
        const items = (marketData.prices || []).filter(p => p.category === cat.key);
        if (items.length === 0) return null;
        return (
          <Card key={cat.key} className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 10 }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>{cat.label}</Text>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
              {items.map(p => (
                <div key={p.key} style={{
                  padding: '6px 12px', borderRadius: 8, minWidth: 140,
                  background: 'var(--semi-color-fill-0)',
                  border: '1px solid var(--semi-color-border)',
                  display: 'flex', alignItems: 'center', gap: 8,
                }}>
                  <span style={{ fontSize: 20 }}>{p.emoji}</span>
                  <div style={{ flex: 1 }}>
                    <Text size='small' strong>{p.name}</Text>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      <Text size='small' style={{ color: mColor(p.multiplier), fontWeight: 700 }}>
                        {p.multiplier}%
                      </Text>
                      <Text size='small' type='tertiary'>{mLabel(p.multiplier)}</Text>
                    </div>
                    <Text size='small' type='tertiary'>
                      ${p.base_price.toFixed(2)} → <span style={{ color: mColor(p.multiplier), fontWeight: 600 }}>${p.cur_price.toFixed(2)}</span>
                    </Text>
                  </div>
                </div>
              ))}
            </div>
          </Card>
        );
      })}
    </div>
  );
};

// ===================== Fish Page =====================
const FishPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [fishData, setFishData] = useState(null);
  const [fishLoading, setFishLoading] = useState(false);
  const [lastCatch, setLastCatch] = useState(null);
  const [cooldown, setCooldown] = useState(0);

  const loadFish = useCallback(async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/fish');
      if (res.success) {
        setFishData(res.data);
        setCooldown(res.data.cooldown || 0);
      }
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setFishLoading(false);
    }
  }, [t]);

  useEffect(() => { loadFish(); }, [loadFish]);

  // Cooldown timer
  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setInterval(() => {
      setCooldown(prev => {
        if (prev <= 1) { clearInterval(timer); return 0; }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [cooldown]);

  const doFish = async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/fish');
      if (res.success) {
        setLastCatch(res.data);
        if (res.data.caught) {
          showSuccess(res.message);
        }
        loadFish();
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setFishLoading(false);
    }
  };

  const doSellAll = async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/fish/sell');
      if (res.success) {
        showSuccess(res.message);
        setLastCatch(null);
        loadFish();
        loadFarm();
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setFishLoading(false);
    }
  };

  if (fishLoading && !fishData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!fishData) return null;

  const rarityColors = {
    '普通': 'grey', '优良': 'green', '稀有': 'blue', '史诗': 'purple', '传说': 'orange',
  };

  return (
    <div>
      {/* Status bar */}
      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 12 }}>
        <Tag size='large' color='amber'>🪱 {t('鱼饵')}: {fishData.bait_count}</Tag>
        <Tag size='large' color={cooldown > 0 ? 'red' : 'green'}>
          {cooldown > 0 ? `⏱️ ${cooldown}s` : `✅ ${t('可以钓鱼')}`}
        </Tag>
        {fishData.total_value > 0 && (
          <Tag size='large' color='cyan'>💰 {t('鱼仓价值')}: ${fishData.total_value.toFixed(2)}</Tag>
        )}
      </div>

      {/* Action buttons */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
        <Button theme='solid' type='primary' loading={fishLoading}
          disabled={cooldown > 0 || fishData.bait_count === 0}
          onClick={doFish}
          icon={<Fish size={14} />}>
          {cooldown > 0 ? `${t('冷却中')} ${cooldown}s` : fishData.bait_count === 0 ? t('没有鱼饵') : t('开始钓鱼')}
        </Button>
        {fishData.total_value > 0 && (
          <Button theme='light' type='warning' loading={fishLoading} onClick={doSellAll}>
            💰 {t('出售全部')} (${fishData.total_value.toFixed(2)})
          </Button>
        )}
      </div>

      {/* Last catch result */}
      {lastCatch && (
        <Banner type={lastCatch.caught ? 'success' : 'warning'} closeIcon={null}
          style={{ marginBottom: 12, borderRadius: 8 }}
          description={lastCatch.caught
            ? <span style={{ fontSize: 16 }}>{lastCatch.fish_emoji} {t('钓到了')} <strong>{lastCatch.fish_name}</strong> <Tag size='small' color={rarityColors[lastCatch.rarity]}>[{lastCatch.rarity}]</Tag> {t('价值')} ${lastCatch.sell_price.toFixed(2)}</span>
            : <span style={{ fontSize: 16 }}>🗑️ {t('空军！什么都没钓到...')}</span>
          }
        />
      )}

      {/* Fish inventory */}
      {fishData.inventory && fishData.inventory.length > 0 && (
        <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 12 }}>
          <Text strong style={{ display: 'block', marginBottom: 8 }}>📦 {t('鱼仓库')}</Text>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {fishData.inventory.map((fish) => (
              <div key={fish.key} style={{
                padding: '6px 12px', borderRadius: 8,
                background: 'var(--semi-color-fill-0)',
                border: '1px solid var(--semi-color-border)',
                display: 'flex', alignItems: 'center', gap: 6,
              }}>
                <span style={{ fontSize: 20 }}>{fish.emoji}</span>
                <div>
                  <Text size='small' strong>{fish.name} ×{fish.quantity}</Text>
                  <Tag size='small' color={rarityColors[fish.rarity]} style={{ marginLeft: 4 }}>{fish.rarity}</Tag>
                  <Text size='small' type='tertiary' style={{ display: 'block' }}>${fish.total_value.toFixed(2)}</Text>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Fish types reference */}
      <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)' }}>
        <Text strong style={{ display: 'block', marginBottom: 8 }}>📊 {t('鱼种图鉴')}</Text>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
          {fishData.fish_types && fishData.fish_types.map((ft) => (
            <div key={ft.key} style={{
              padding: '4px 10px', borderRadius: 6, minWidth: 120,
              background: 'var(--semi-color-fill-0)',
              border: '1px solid var(--semi-color-border)',
              display: 'flex', alignItems: 'center', gap: 6,
            }}>
              <span style={{ fontSize: 18 }}>{ft.emoji}</span>
              <div>
                <Text size='small'>{ft.name}</Text>
                <Tag size='small' color={rarityColors[ft.rarity]} style={{ marginLeft: 4 }}>{ft.rarity}</Tag>
                <Text size='small' type='tertiary' style={{ display: 'block' }}>{ft.chance}% · ${ft.sell_price.toFixed(2)}</Text>
              </div>
            </div>
          ))}
          <div style={{
            padding: '4px 10px', borderRadius: 6, minWidth: 120,
            background: 'var(--semi-color-fill-0)',
            border: '1px solid var(--semi-color-border)',
            display: 'flex', alignItems: 'center', gap: 6,
          }}>
            <span style={{ fontSize: 18 }}>🗑️</span>
            <div>
              <Text size='small'>{t('空军')}</Text>
              <Text size='small' type='tertiary' style={{ display: 'block' }}>{fishData.nothing_chance}%</Text>
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
};

// ===================== Logs Page =====================
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

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
        <Text strong>{t('消费记录')} ({total})</Text>
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={() => loadLogs(1)} loading={logsLoading} />
      </div>
      {logsLoading && logs.length === 0 ? (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>
      ) : logs.length === 0 ? (
        <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', textAlign: 'center', padding: 30 }}>
          <Text type='tertiary'>{t('暂无消费记录')}</Text>
        </Card>
      ) : (
        <>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {logs.map((log) => (
              <div key={log.id} style={{
                display: 'flex', alignItems: 'center', gap: 10,
                padding: '8px 12px', borderRadius: 8,
                background: 'var(--semi-color-fill-0)',
                border: '1px solid var(--semi-color-border)',
              }}>
                <span style={{ fontSize: 18, width: 28, textAlign: 'center', flexShrink: 0 }}>
                  {actionEmojis[log.action] || '📋'}
                </span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <Tag size='small' color='grey' style={{ borderRadius: 4 }}>{log.action_label}</Tag>
                    <Text size='small' style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {log.detail}
                    </Text>
                  </div>
                </div>
                <Text strong size='small' style={{
                  color: log.amount >= 0 ? '#16a34a' : '#dc2626',
                  flexShrink: 0, minWidth: 60, textAlign: 'right',
                }}>
                  {log.amount >= 0 ? '+' : ''}{log.amount.toFixed(2)}
                </Text>
                <Text type='tertiary' size='small' style={{ flexShrink: 0, minWidth: 80, textAlign: 'right' }}>
                  {formatTime(log.created_at)}
                </Text>
              </div>
            ))}
          </div>
          {total > 20 && (
            <div style={{ display: 'flex', justifyContent: 'center', gap: 8, marginTop: 12 }}>
              <Button size='small' disabled={page <= 1} onClick={() => loadLogs(page - 1)}>{t('上一页')}</Button>
              <Text type='tertiary' size='small' style={{ lineHeight: '32px' }}>{page}/{Math.ceil(total / 20)}</Text>
              <Button size='small' disabled={page >= Math.ceil(total / 20)} onClick={() => loadLogs(page + 1)}>{t('下一页')}</Button>
            </div>
          )}
        </>
      )}
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
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 300, marginTop: 60 }}>
        <Spin size='large' />
      </div>
    );
  }

  if (!farmData) {
    return (
      <div style={{ maxWidth: 960, margin: '80px auto 0', padding: '0 16px' }}>
        <Card className='!rounded-xl' style={{ border: '1px solid var(--semi-color-border)', textAlign: 'center', padding: 30 }}>
          <Sprout size={36} style={{ color: 'var(--semi-color-text-3)', marginBottom: 12 }} />
          <Title heading={6}>{t('农场不可用')}</Title>
          <Text type='tertiary' size='small'>{t('请先绑定 Telegram 账号')}</Text>
        </Card>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 960, margin: '70px auto 0', padding: '0 16px 40px' }}>
      {/* Compact page header */}
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 14 }}>
        <div style={{
          width: 36, height: 36, borderRadius: 10,
          background: 'linear-gradient(135deg, #22c55e, #16a34a)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          flexShrink: 0, marginRight: 10,
        }}>
          <Wheat size={18} style={{ color: 'white' }} />
        </div>
        <div>
          <Title heading={5} style={{ margin: 0 }}>🌾 {t('我的农场')}</Title>
        </div>
      </div>

      {/* Tabs */}
      <Tabs type='button' size='small' activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab={<span><Wheat size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('总览')}</span>} itemKey='overview'>
          <FarmOverview farmData={farmData} loading={loading} loadFarm={loadFarm}
            actionLoading={actionLoading} doAction={doAction} t={t} />
        </TabPane>
        <TabPane tab={<span><Sprout size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('种植')}</span>} itemKey='plant'>
          <PlantPage farmData={farmData} crops={crops} actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Store size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('商店')}</span>} itemKey='shop'>
          <ShopPage farmData={farmData} actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Swords size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('偷菜')}</span>} itemKey='steal'>
          <StealPage actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Dog size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('狗狗')}</span>} itemKey='dog'>
          <DogPage actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span>🐄 {t('牧场')}</span>} itemKey='ranch'>
          <RanchPage actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Fish size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('钓鱼')}</span>} itemKey='fish'>
          <FishPage actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Factory size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('加工')}</span>} itemKey='workshop'>
          <WorkshopPage actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Star size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('等级')}</span>} itemKey='level'>
          <LevelPage actionLoading={actionLoading} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><ClipboardList size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('任务')}</span>} itemKey='tasks'>
          <TasksPage actionLoading={actionLoading} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Trophy size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('成就')}</span>} itemKey='achievements'>
          <AchievementsPage actionLoading={actionLoading} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Landmark size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('银行')}</span>} itemKey='bank'>
          <BankPage farmData={farmData} actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><Warehouse size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('仓库')}</span>} itemKey='warehouse'>
          <WarehousePage actionLoading={actionLoading}
            doAction={doAction} loadFarm={loadFarm} t={t} />
        </TabPane>
        <TabPane tab={<span><TrendingUp size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('市场')}</span>} itemKey='market'>
          <MarketPage t={t} />
        </TabPane>
        <TabPane tab={<span><ScrollText size={13} style={{ marginRight: 3, verticalAlign: -2 }} />{t('记录')}</span>} itemKey='logs'>
          <LogsPage t={t} />
        </TabPane>
      </Tabs>
    </div>
  );
};

export default Farm;
