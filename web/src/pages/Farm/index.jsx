import React, { useCallback, useContext, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
} from '../../helpers';
import {
  Button,
  Card,
  Modal,
  Select,
  Space,
  Spin,
  Tag,
  Tooltip,
  Typography,
  Progress,
} from '@douyinfe/semi-ui';
import {
  Sprout,
  Wheat,
  Store,
  Droplets,
  Bug,
  Dog,
  Shovel,
  RefreshCw,
  ShoppingCart,
  Swords,
  Pill,
  FlaskConical,
  LandPlot,
  Package,
} from 'lucide-react';
import { UserContext } from '../../context/User';

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

const Farm = () => {
  const { t } = useTranslation();
  const [userState] = useContext(UserContext);
  const [loading, setLoading] = useState(true);
  const [farmData, setFarmData] = useState(null);
  const [crops, setCrops] = useState([]);
  const [actionLoading, setActionLoading] = useState(false);

  // Modals
  const [plantModal, setPlantModal] = useState(false);
  const [shopModal, setShopModal] = useState(false);
  const [stealModal, setStealModal] = useState(false);
  const [dogModal, setDogModal] = useState(false);
  const [harvestResult, setHarvestResult] = useState(null);

  // Plant state
  const [selectedCrop, setSelectedCrop] = useState(null);
  const [selectedPlot, setSelectedPlot] = useState(null);

  // Shop state
  const [shopData, setShopData] = useState(null);

  // Steal state
  const [stealTargets, setStealTargets] = useState([]);

  // Dog state
  const [dogData, setDogData] = useState(null);

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

  const loadCrops = async () => {
    try {
      const { data: res } = await API.get('/api/farm/crops');
      if (res.success) setCrops(res.data || []);
    } catch (err) { /* ignore */ }
  };

  useEffect(() => {
    loadFarm();
    loadCrops();
  }, [loadFarm]);

  // Auto-refresh every 30s
  useEffect(() => {
    const interval = setInterval(loadFarm, 30000);
    return () => clearInterval(interval);
  }, [loadFarm]);

  const doAction = async (url, body, successMsg) => {
    setActionLoading(true);
    try {
      const { data: res } = await API.post(url, body);
      if (res.success) {
        showSuccess(res.message || successMsg || t('操作成功'));
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

  // ---- Plant ----
  const openPlant = () => {
    setSelectedCrop(null);
    setSelectedPlot(null);
    setPlantModal(true);
  };

  const handlePlant = async () => {
    if (!selectedCrop || selectedPlot === null) {
      showError(t('请选择作物和地块'));
      return;
    }
    const res = await doAction('/api/farm/plant', { crop_key: selectedCrop, plot_index: selectedPlot });
    if (res) setPlantModal(false);
  };

  // ---- Harvest ----
  const handleHarvest = async () => {
    const res = await doAction('/api/farm/harvest', {});
    if (res && res.data) {
      setHarvestResult(res.data);
    }
  };

  // ---- Shop ----
  const openShop = async () => {
    setShopModal(true);
    try {
      const { data: res } = await API.get('/api/farm/shop');
      if (res.success) setShopData(res.data);
    } catch (err) { /* ignore */ }
  };

  const handleBuyItem = async (itemKey) => {
    await doAction('/api/farm/buy', { item_key: itemKey });
    // refresh shop
    try {
      const { data: res } = await API.get('/api/farm/shop');
      if (res.success) setShopData(res.data);
    } catch (err) { /* ignore */ }
  };

  // ---- Water ----
  const handleWater = async (plotIndex) => {
    await doAction('/api/farm/water', { plot_index: plotIndex });
  };

  // ---- Treat ----
  const handleTreat = async (plotIndex) => {
    await doAction('/api/farm/treat', { plot_index: plotIndex });
  };

  // ---- Fertilize ----
  const handleFertilize = async (plotIndex) => {
    await doAction('/api/farm/fertilize', { plot_index: plotIndex });
  };

  // ---- Buy Land ----
  const handleBuyLand = async () => {
    await doAction('/api/farm/buyland', {});
  };

  // ---- Steal ----
  const openSteal = async () => {
    setStealModal(true);
    try {
      const { data: res } = await API.get('/api/farm/steal/targets');
      if (res.success) setStealTargets(res.data || []);
    } catch (err) { /* ignore */ }
  };

  const handleSteal = async (victimId) => {
    const res = await doAction('/api/farm/steal', { victim_id: victimId });
    if (res) {
      // refresh targets
      try {
        const { data: r } = await API.get('/api/farm/steal/targets');
        if (r.success) setStealTargets(r.data || []);
      } catch (err) { /* ignore */ }
    }
  };

  // ---- Dog ----
  const openDog = async () => {
    setDogModal(true);
    try {
      const { data: res } = await API.get('/api/farm/dog');
      if (res.success) setDogData(res.data);
    } catch (err) { /* ignore */ }
  };

  const handleBuyDog = async () => {
    const res = await doAction('/api/farm/buydog', {});
    if (res) {
      try {
        const { data: r } = await API.get('/api/farm/dog');
        if (r.success) setDogData(r.data);
      } catch (err) { /* ignore */ }
    }
  };

  const handleFeedDog = async () => {
    const res = await doAction('/api/farm/feeddog', {});
    if (res) {
      try {
        const { data: r } = await API.get('/api/farm/dog');
        if (r.success) setDogData(r.data);
      } catch (err) { /* ignore */ }
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
      <div className='mt-[60px] px-2' style={{ maxWidth: 960, margin: '60px auto 0' }}>
        <Card className='!rounded-2xl shadow-sm' style={{ border: '1px solid var(--semi-color-border)', textAlign: 'center', padding: 40 }}>
          <Sprout size={48} style={{ color: 'var(--semi-color-text-3)', marginBottom: 16 }} />
          <Title heading={5}>{t('农场不可用')}</Title>
          <Text type='tertiary'>{t('请先绑定 Telegram 账号后才能使用农场功能')}</Text>
        </Card>
      </div>
    );
  }

  const emptyPlots = (farmData.plots || []).filter(p => p.status === 0);
  const maturePlots = (farmData.plots || []).filter(p => p.status === 2);
  const eventPlots = (farmData.plots || []).filter(p => p.status === 3 && p.event_type !== 'drought');
  const waterablePlots = (farmData.plots || []).filter(p =>
    p.status === 1 || p.status === 4 || (p.status === 3 && p.event_type === 'drought')
  );
  const fertilizablePlots = (farmData.plots || []).filter(p => p.status === 1 && p.fertilized === 0);

  return (
    <div className='mt-[60px] px-2' style={{ maxWidth: 960, margin: '60px auto 0', paddingBottom: 40 }}>
      {/* Header */}
      <Card className='!rounded-2xl shadow-sm' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
          <div style={{
            width: 44, height: 44, borderRadius: 14,
            background: 'linear-gradient(135deg, #22c55e, #16a34a)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: '0 2px 8px rgba(34,197,94,0.3)', flexShrink: 0, marginRight: 14,
          }}>
            <Wheat size={22} style={{ color: 'white' }} />
          </div>
          <div style={{ flex: 1 }}>
            <Title heading={5} style={{ margin: 0 }}>🌾 {t('我的农场')}</Title>
            <Text type='tertiary' size='small'>
              💰 {t('余额')}: ${farmData.balance?.toFixed(2)} &nbsp;|&nbsp;
              📊 {t('土地')} {farmData.plot_count}/{farmData.max_plots}
            </Text>
          </div>
          <Button icon={<RefreshCw size={14} />} theme='borderless' onClick={loadFarm} loading={loading}>
            {t('刷新')}
          </Button>
        </div>

        {/* Action buttons */}
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
          <Button icon={<Sprout size={14} />} theme='solid' type='primary' onClick={openPlant}
            disabled={emptyPlots.length === 0} style={{ borderRadius: 10 }}>
            {t('种植')}
          </Button>
          <Button icon={<Wheat size={14} />} theme='solid' style={{ borderRadius: 10, background: '#f59e0b' }}
            onClick={handleHarvest} disabled={maturePlots.length === 0} loading={actionLoading}>
            {t('收获')} {maturePlots.length > 0 && `(${maturePlots.length})`}
          </Button>
          <Button icon={<Store size={14} />} theme='light' onClick={openShop} style={{ borderRadius: 10 }}>
            {t('商店')}
          </Button>
          <Button icon={<Swords size={14} />} theme='light' type='warning' onClick={openSteal} style={{ borderRadius: 10 }}>
            {t('偷菜')}
          </Button>
          <Button icon={<Dog size={14} />} theme='light' onClick={openDog} style={{ borderRadius: 10 }}>
            {t('狗狗')}
          </Button>
          {farmData.plot_count < farmData.max_plots && (
            <Button icon={<LandPlot size={14} />} theme='light' onClick={handleBuyLand}
              loading={actionLoading} style={{ borderRadius: 10 }}>
              {t('买地')} (${farmData.plot_price?.toFixed(2)})
            </Button>
          )}
        </div>
      </Card>

      {/* Backpack */}
      {farmData.items && farmData.items.length > 0 && (
        <Card className='!rounded-2xl shadow-sm' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
            <Package size={16} />
            <Text strong>{t('背包')}</Text>
          </div>
          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            {farmData.items.map((item) => (
              <Tag key={item.key} size='large' color='blue'>
                {item.emoji} {item.name} ×{item.quantity}
              </Tag>
            ))}
          </div>
        </Card>
      )}

      {/* Dog info inline */}
      {farmData.dog && (
        <Card className='!rounded-2xl shadow-sm' style={{ border: '1px solid var(--semi-color-border)', marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <span style={{ fontSize: 24 }}>{farmData.dog.level === 2 ? '🐕' : '🐶'}</span>
            <div>
              <Text strong>{farmData.dog.name}</Text>
              <Text type='tertiary' size='small'> — {farmData.dog.level_name} · {farmData.dog.status} · {t('饱食度')} {farmData.dog.hunger}%</Text>
            </div>
          </div>
        </Card>
      )}

      {/* Plots Grid */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 12 }}>
        {(farmData.plots || []).map((plot) => (
          <Card key={plot.plot_index} className='!rounded-xl'
            style={{
              border: `1px solid ${plot.status === 3 || plot.status === 4 ? 'var(--semi-color-danger)' : 'var(--semi-color-border)'}`,
              background: plot.status === 0 ? 'var(--semi-color-fill-0)' : undefined,
            }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
              <Text strong>
                {statusEmojis[plot.status]} {plot.plot_index + 1}{t('号地')}
              </Text>
              <Tag size='small' color={statusColors[plot.status]}>
                {plot.status_label}
              </Tag>
            </div>

            {plot.status === 0 && (
              <Text type='tertiary' size='small'>{t('空地，可以种植作物')}</Text>
            )}

            {plot.status === 1 && (
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6 }}>
                  <span>{plot.crop_emoji}</span>
                  <Text>{plot.crop_name}</Text>
                  {plot.fertilized === 1 && <Tag size='small' color='cyan'>🧴{t('已施肥')}</Tag>}
                </div>
                <Progress percent={plot.progress} size='small' style={{ marginBottom: 4 }} />
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text type='tertiary' size='small'>⏳ {formatDuration(plot.remaining)}</Text>
                  {plot.water_remain !== undefined && (
                    <Text type={plot.water_remain <= 0 ? 'danger' : 'tertiary'} size='small'>
                      💧 {plot.water_remain > 0 ? formatDuration(plot.water_remain) : '⚠️' + t('需浇水')}
                    </Text>
                  )}
                </div>
                <div style={{ marginTop: 8, display: 'flex', gap: 6 }}>
                  <Button size='small' icon={<Droplets size={12} />} onClick={() => handleWater(plot.plot_index)}
                    loading={actionLoading} style={{ borderRadius: 8 }}>
                    {t('浇水')}
                  </Button>
                  {plot.fertilized === 0 && (
                    <Button size='small' icon={<FlaskConical size={12} />} onClick={() => handleFertilize(plot.plot_index)}
                      loading={actionLoading} style={{ borderRadius: 8 }}>
                      {t('施肥')}
                    </Button>
                  )}
                </div>
              </div>
            )}

            {plot.status === 2 && (
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span>{plot.crop_emoji}</span>
                  <Text>{plot.crop_name}</Text>
                  <Tag size='small' color='green'>{t('可收获')}</Tag>
                </div>
                {plot.stolen_count > 0 && (
                  <Text type='warning' size='small'>⚠️ {t('被偷')} {plot.stolen_count} {t('次')}</Text>
                )}
              </div>
            )}

            {plot.status === 3 && (
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6 }}>
                  <span>{plot.crop_emoji}</span>
                  <Text>{plot.crop_name}</Text>
                </div>
                {plot.event_type === 'drought' ? (
                  <>
                    <Text type='danger' size='small'>🏜️ {t('天灾干旱！快浇水救命！')}</Text>
                    <Text type='danger' size='small' style={{ display: 'block' }}>💀 {formatDuration(plot.death_remain)} {t('后死亡')}</Text>
                    <Button size='small' type='danger' icon={<Droplets size={12} />}
                      onClick={() => handleWater(plot.plot_index)} loading={actionLoading}
                      style={{ marginTop: 6, borderRadius: 8 }}>
                      {t('浇水救命')}
                    </Button>
                  </>
                ) : (
                  <>
                    <Text type='danger' size='small'>🐛 {t('虫害！需要治疗')}</Text>
                    <Button size='small' type='warning' icon={<Pill size={12} />}
                      onClick={() => handleTreat(plot.plot_index)} loading={actionLoading}
                      style={{ marginTop: 6, borderRadius: 8 }}>
                      {t('治疗')}
                    </Button>
                  </>
                )}
              </div>
            )}

            {plot.status === 4 && (
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6 }}>
                  <span>{plot.crop_emoji}</span>
                  <Text>{plot.crop_name}</Text>
                </div>
                <Text type='danger' size='small'>🥀 {t('枯萎中！快浇水！')}</Text>
                <Text type='danger' size='small' style={{ display: 'block' }}>💀 {formatDuration(plot.death_remain)} {t('后死亡')}</Text>
                <Button size='small' type='danger' icon={<Droplets size={12} />}
                  onClick={() => handleWater(plot.plot_index)} loading={actionLoading}
                  style={{ marginTop: 6, borderRadius: 8 }}>
                  {t('浇水救命')}
                </Button>
              </div>
            )}
          </Card>
        ))}
      </div>

      {/* Plant Modal */}
      <Modal title={`🌱 ${t('种植')}`} visible={plantModal}
        onOk={handlePlant} onCancel={() => setPlantModal(false)}
        okText={t('种植')} confirmLoading={actionLoading}
        style={{ maxWidth: 500 }}>
        <div style={{ marginBottom: 12 }}>
          <Text strong style={{ display: 'block', marginBottom: 6 }}>{t('选择作物')}</Text>
          <Select placeholder={t('选择作物')} value={selectedCrop} onChange={setSelectedCrop}
            style={{ width: '100%' }}
            optionList={crops.map(c => ({
              label: `${c.emoji} ${c.name} — $${c.seed_cost.toFixed(2)} | ${formatDuration(c.grow_secs)} | ${t('最高')} $${c.max_value.toFixed(2)}`,
              value: c.key,
            }))}
          />
        </div>
        <div>
          <Text strong style={{ display: 'block', marginBottom: 6 }}>{t('选择地块')}</Text>
          <Select placeholder={t('选择空地')} value={selectedPlot} onChange={setSelectedPlot}
            style={{ width: '100%' }}
            optionList={emptyPlots.map(p => ({
              label: `⬜ ${p.plot_index + 1}${t('号地')}`,
              value: p.plot_index,
            }))}
          />
        </div>
        {selectedCrop && (
          <div style={{
            marginTop: 12, padding: 10, borderRadius: 10,
            background: 'var(--semi-color-fill-0)',
          }}>
            {(() => {
              const c = crops.find(x => x.key === selectedCrop);
              if (!c) return null;
              return (
                <div style={{ fontSize: 13 }}>
                  <Text>{c.emoji} <strong>{c.name}</strong></Text><br />
                  <Text type='tertiary'>
                    {t('种子')}: ${c.seed_cost.toFixed(2)} | {t('生长')}: {formatDuration(c.grow_secs)} |
                    {t('产量')}: 1~{c.max_yield}×${c.unit_price.toFixed(2)} | {t('最高')}: ${c.max_value.toFixed(2)}
                  </Text>
                </div>
              );
            })()}
          </div>
        )}
      </Modal>

      {/* Harvest Result Modal */}
      <Modal title={`🌾 ${t('收获结果')}`} visible={!!harvestResult}
        onOk={() => setHarvestResult(null)} onCancel={() => setHarvestResult(null)}
        cancelButtonProps={{ style: { display: 'none' } }}
        okText={t('好的')} style={{ maxWidth: 480 }}>
        {harvestResult && (
          <div>
            <div style={{ textAlign: 'center', marginBottom: 16 }}>
              <Title heading={4} style={{ color: '#22c55e' }}>💰 ${harvestResult.total?.toFixed(2)}</Title>
              <Text type='tertiary'>{t('共收获')} {harvestResult.count} {t('块作物')}</Text>
            </div>
            {(harvestResult.details || []).map((d, i) => (
              <div key={i} style={{
                padding: '8px 12px', borderRadius: 8, marginBottom: 6,
                background: 'var(--semi-color-fill-0)',
                display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              }}>
                <Text>{d.crop_emoji} {d.crop_name}: {t('产量')}{d.yield}
                  {d.fert_bonus > 0 && <span style={{ color: '#06b6d4' }}> +{t('化肥')}{d.fert_bonus}</span>}
                  {d.stolen > 0 && <span style={{ color: '#ef4444' }}> -{t('被偷')}{d.stolen}</span>}
                </Text>
                <Tag color='green'>${d.value?.toFixed(2)}</Tag>
              </div>
            ))}
          </div>
        )}
      </Modal>

      {/* Shop Modal */}
      <Modal title={`🏪 ${t('商店')}`} visible={shopModal}
        onCancel={() => setShopModal(false)} footer={null}
        style={{ maxWidth: 500 }}>
        {!shopData ? (
          <div style={{ textAlign: 'center', padding: 20 }}><Spin /></div>
        ) : (
          <div>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>📦 {t('道具')}</Text>
            {(shopData.items || []).map((item) => (
              <div key={item.key} style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                padding: '10px 12px', borderRadius: 10, marginBottom: 6,
                background: 'var(--semi-color-fill-0)',
              }}>
                <div>
                  <Text>{item.emoji} <strong>{item.name}</strong></Text>
                  <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>{item.desc}</Text>
                </div>
                <Button size='small' theme='solid' onClick={() => handleBuyItem(item.key)}
                  loading={actionLoading} style={{ borderRadius: 8 }}>
                  ${item.cost?.toFixed(2)}
                </Button>
              </div>
            ))}
            {!shopData.has_dog && (
              <>
                <Text strong style={{ display: 'block', marginTop: 16, marginBottom: 8 }}>🐕 {t('看门狗')}</Text>
                <div style={{
                  display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                  padding: '10px 12px', borderRadius: 10,
                  background: 'var(--semi-color-fill-0)',
                }}>
                  <Text>🐶 {t('小狗')} <Text type='tertiary' size='small'>({t('长大后拦截偷菜')})</Text></Text>
                  <Button size='small' theme='solid' onClick={handleBuyDog}
                    loading={actionLoading} style={{ borderRadius: 8 }}>
                    ${shopData.dog_price?.toFixed(2)}
                  </Button>
                </div>
              </>
            )}
          </div>
        )}
      </Modal>

      {/* Steal Modal */}
      <Modal title={`🕵️ ${t('偷菜')}`} visible={stealModal}
        onCancel={() => setStealModal(false)} footer={null}
        style={{ maxWidth: 460 }}>
        {stealTargets.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 20 }}>
            <Text type='tertiary'>{t('暂时没有可偷的菜地')}</Text>
          </div>
        ) : (
          <div>
            {stealTargets.map((target) => (
              <div key={target.id} style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                padding: '10px 12px', borderRadius: 10, marginBottom: 6,
                background: 'var(--semi-color-fill-0)',
              }}>
                <div>
                  <Text>👤 {target.label}</Text>
                  <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>{target.count} {t('块成熟')}</Text>
                </div>
                <Button size='small' type='warning' theme='solid' onClick={() => handleSteal(target.id)}
                  loading={actionLoading} style={{ borderRadius: 8 }}>
                  {t('偷菜')}
                </Button>
              </div>
            ))}
          </div>
        )}
      </Modal>

      {/* Dog Modal */}
      <Modal title={`🐕 ${t('狗狗')}`} visible={dogModal}
        onCancel={() => setDogModal(false)} footer={null}
        style={{ maxWidth: 460 }}>
        {!dogData ? (
          <div style={{ textAlign: 'center', padding: 20 }}><Spin /></div>
        ) : !dogData.has_dog ? (
          <div style={{ textAlign: 'center', padding: 20 }}>
            <div style={{ fontSize: 48, marginBottom: 12 }}>🐶</div>
            <Text>{t('你还没有狗狗！')}</Text>
            <div style={{ marginTop: 8 }}>
              <Text type='tertiary' size='small'>
                {t('购买小狗后')} {dogData.grow_hours} {t('小时长大，拦截率')} {dogData.guard_rate}%
              </Text>
            </div>
            <Button theme='solid' style={{ marginTop: 16, borderRadius: 10 }}
              onClick={handleBuyDog} loading={actionLoading}>
              🐶 {t('购买小狗')} (${dogData.dog_price?.toFixed(2)})
            </Button>
          </div>
        ) : (
          <div>
            <div style={{ textAlign: 'center', marginBottom: 16 }}>
              <div style={{ fontSize: 48 }}>{dogData.level === 2 ? '🐕' : '🐶'}</div>
              <Title heading={5} style={{ margin: '8px 0 0' }}>「{dogData.name}」</Title>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
              <div style={{ padding: 10, borderRadius: 8, background: 'var(--semi-color-fill-0)', textAlign: 'center' }}>
                <Text type='tertiary' size='small'>{t('等级')}</Text>
                <div><Text strong>{dogData.level_name}</Text></div>
              </div>
              <div style={{ padding: 10, borderRadius: 8, background: 'var(--semi-color-fill-0)', textAlign: 'center' }}>
                <Text type='tertiary' size='small'>{t('状态')}</Text>
                <div><Text strong>{dogData.status}</Text></div>
              </div>
              <div style={{ padding: 10, borderRadius: 8, background: 'var(--semi-color-fill-0)', textAlign: 'center' }}>
                <Text type='tertiary' size='small'>{t('饱食度')}</Text>
                <div><Text strong>{dogData.hunger}%</Text></div>
              </div>
              <div style={{ padding: 10, borderRadius: 8, background: 'var(--semi-color-fill-0)', textAlign: 'center' }}>
                <Text type='tertiary' size='small'>{t('拦截率')}</Text>
                <div><Text strong>{dogData.guard_rate}%</Text></div>
              </div>
            </div>
            {dogData.hunger < 100 && (
              <Button theme='solid' block style={{ marginTop: 16, borderRadius: 10 }}
                onClick={handleFeedDog} loading={actionLoading}>
                🦴 {t('喂狗粮')}
              </Button>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default Farm;
