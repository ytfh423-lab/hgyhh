import React, { useState, useCallback, useEffect, useRef, memo } from 'react';
import { Button, Typography, Select } from '@douyinfe/semi-ui';
import { RefreshCw, Droplets, FlaskConical, ArrowUp, Pill, Plus, Sprout, Trash2, Zap, Users } from 'lucide-react';
import { formatBalance, formatDuration, confirmAction } from './utils';
import { farmConfirm } from './farmConfirm';
import { useTutorial } from './TutorialProvider';
import FarmAnnouncementBar from './FarmAnnouncementBar';
import { API } from '../../../helpers';

const { Text } = Typography;

/* ═══════════════════════════════════════════════════════════════
   StatCard — dashboard 统计指标卡片
   ═══════════════════════════════════════════════════════════════ */
const StatCard = ({ emoji, label, value, accent }) => (
  <div className='farm-stat-card'>
    <span style={{ fontSize: 24, lineHeight: 1, flexShrink: 0 }}>{emoji}</span>
    <div style={{ minWidth: 0 }}>
      <div style={{ fontSize: 11, color: 'var(--farm-text-2)', whiteSpace: 'nowrap', letterSpacing: 0.3 }}>{label}</div>
      <div style={{ fontSize: 17, fontWeight: 700, color: accent || 'var(--farm-text-0)', marginTop: 1 }}>{value}</div>
    </div>
  </div>
);

/* ═══════════════════════════════════════════════════════════════
   PlotCard — 单个地块交互卡片（支持 empty / growing / mature / event / wilting）
   ═══════════════════════════════════════════════════════════════ */
const statusClassMap = {
  0: 'farm-plot-card farm-plot-empty',
  1: 'farm-plot-card farm-plot-growing',
  2: 'farm-plot-card farm-plot-mature',
  3: 'farm-plot-card farm-plot-event',
  4: 'farm-plot-card farm-plot-wilting',
};

const PlotCard = memo(({ plot, farmData, handlers, actionLoading, expanded, onToggle, t }) => {
  const { handleWater, handleFertilize, handleTreat, handleUpgradeSoil, handleClearPlot } = handlers;
  const soilLv = plot.soil_level || 1;
  const soilMax = farmData.soil_max_level || 5;
  const st = plot.status;

  /* ── Empty plot ── */
  if (st === 0) {
    return (
      <div className={statusClassMap[0]} style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: 130, textAlign: 'center' }}>
        <Sprout size={26} strokeWidth={1.5} style={{ color: 'var(--farm-text-3)', marginBottom: 6 }} />
        <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--farm-text-3)' }}>
          {plot.plot_index + 1}{t('号地')}
        </span>
        <span style={{ fontSize: 11, color: 'var(--farm-text-3)', marginTop: 2 }}>{t('空地')}</span>
      </div>
    );
  }

  const emoji = st === 4 ? '🥀' : plot.crop_emoji;
  const needsWater = st === 1 || st === 4 || (st === 3 && plot.event_type === 'drought');

  return (
    <div className={statusClassMap[st] || statusClassMap[1]} onClick={onToggle}>
      {/* ── Plot number badge ── */}
      <div style={{
        position: 'absolute', top: 8, right: 10, fontSize: 10, fontWeight: 700,
        color: 'var(--farm-text-3)', opacity: 0.6,
      }}>
        #{plot.plot_index + 1}
      </div>

      {/* ── Crop icon + name ── */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
        <span style={{ fontSize: 32, lineHeight: 1 }}>{emoji}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 13, fontWeight: 700, color: 'var(--farm-text-0)', display: 'flex', alignItems: 'center', gap: 4, flexWrap: 'wrap' }}>
            {plot.crop_name}
            {soilLv > 1 && (
              <span style={{ fontSize: 10, padding: '1px 5px', borderRadius: 4, background: 'rgba(138,108,176,0.15)', color: '#b094d0', fontWeight: 600 }}>
                Lv.{soilLv}
              </span>
            )}
            {plot.fertilized === 1 && (
              <span style={{ fontSize: 10, padding: '1px 5px', borderRadius: 4, background: 'rgba(90,143,180,0.15)', color: 'var(--farm-sky)', fontWeight: 600 }}>
                🧴
              </span>
            )}
          </div>
          {/* Status sub-label */}
          <div style={{ fontSize: 11, marginTop: 2, color: 'var(--farm-text-2)' }}>
            {st === 1 && (
              <>
                ⏳ {formatDuration(plot.remaining)}
                {plot.ready_at > 0 && (
                  <span style={{ marginLeft: 6, opacity: 0.7 }}>
                    ({new Date(plot.ready_at * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })})
                  </span>
                )}
              </>
            )}
            {st === 2 && (plot.stolen_count > 0 ? `⚠️ ${t('已被偷')} ${plot.stolen_count}` : `✅ ${t('可收获')}`)}
            {st === 3 && (plot.event_type === 'drought' ? `🏜️ ${t('干旱')}` : `🐛 ${t('虫害')}`)}
            {st === 4 && `🥀 ${t('枯萎中')}`}
          </div>
        </div>
      </div>

      {/* ── Growing: progress bar ── */}
      {st === 1 && (
        <div>
          <div className='farm-progress' style={{ marginBottom: 6, height: 7 }}>
            <div className='farm-progress-fill' style={{
              width: `${plot.progress}%`,
              background: 'linear-gradient(90deg, var(--farm-sky), var(--farm-leaf))',
            }} />
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11 }}>
            <span style={{ color: 'var(--farm-text-2)' }}>{plot.progress}%</span>
            {plot.last_watered_at > 0 && (
              <span style={{ color: plot.water_remain <= 0 ? 'var(--farm-danger)' : 'var(--farm-text-2)' }}>
                💧 {plot.water_remain > 0 ? formatDuration(plot.water_remain) : t('需浇水')}
              </span>
            )}
          </div>
        </div>
      )}

      {/* ── Mature: ready indicator ── */}
      {st === 2 && (
        <div style={{
          padding: '6px 10px', borderRadius: 8, textAlign: 'center',
          background: 'rgba(74,124,63,0.1)', border: '1px solid rgba(74,124,63,0.2)',
          fontSize: 12, fontWeight: 600, color: 'var(--farm-leaf)',
        }}>
          🌾 {t('已成熟，可收获')}
        </div>
      )}

      {/* ── Event / Wilting: death timer ── */}
      {(st === 3 || st === 4) && (
        <div style={{
          padding: '6px 10px', borderRadius: 8, textAlign: 'center',
          background: st === 3 ? 'rgba(184,66,51,0.1)' : 'rgba(200,146,42,0.1)',
          border: `1px solid ${st === 3 ? 'rgba(184,66,51,0.2)' : 'rgba(200,146,42,0.2)'}`,
          fontSize: 12, fontWeight: 600, color: st === 3 ? 'var(--farm-danger)' : 'var(--farm-harvest)',
        }}>
          💀 {formatDuration(plot.death_remain)} {t('后死亡')}
        </div>
      )}

      {/* ── Action panel (always visible) ── */}
      <div className='farm-plot-actions'>
        {needsWater && (
          <Button size='small' icon={<Droplets size={12} />}
            onClick={e => { e.stopPropagation(); handleWater(plot.plot_index); }}
            loading={actionLoading} theme='solid' className='farm-btn'>
            {t('浇水')}
          </Button>
        )}
        {st === 1 && plot.fertilized === 0 && (
          <Button size='small' icon={<FlaskConical size={12} />}
            onClick={e => { e.stopPropagation(); handleFertilize(plot.plot_index); }}
            loading={actionLoading} theme='solid' className='farm-btn'>
            {t('施肥')}
          </Button>
        )}
        {st === 3 && plot.event_type !== 'drought' && (
          <Button size='small' icon={<Pill size={12} />}
            onClick={e => { e.stopPropagation(); handleTreat(plot.plot_index); }}
            loading={actionLoading} theme='solid' className='farm-btn'>
            {t('治疗')}
          </Button>
        )}
        {soilLv < soilMax && (
          <Button size='small' icon={<ArrowUp size={12} />}
            onClick={e => { e.stopPropagation(); handleUpgradeSoil(plot.plot_index); }}
            loading={actionLoading} theme='solid' className='farm-btn'>
            {t('升级')} Lv.{soilLv + 1}
          </Button>
        )}
        <Button size='small' icon={<Trash2 size={12} />}
          onClick={async e => { e.stopPropagation(); if (await farmConfirm(t('铲除作物'), t('确定要铲除这块地的作物吗？'), { icon: '🗑', confirmType: 'danger', confirmText: t('铲除') })) handleClearPlot(plot.plot_index); }}
          loading={actionLoading} theme='light' type='danger' className='farm-btn'>
          {t('铲除')}
        </Button>
      </div>
    </div>
  );
});

/* ═══════════════════════════════════════════════════════════════
   BuyLandCard — 购买新地块入口
   ═══════════════════════════════════════════════════════════════ */
const BuyLandCard = ({ price, onClick, actionLoading, t }) => (
  <div className='farm-plot-card farm-plot-buy'
    onClick={onClick}
    style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: 130, textAlign: 'center' }}>
    <Plus size={30} strokeWidth={1.5} style={{ color: 'var(--farm-leaf)', marginBottom: 8 }} />
    <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--farm-leaf)' }}>{t('购买土地')}</span>
    <span style={{ fontSize: 12, color: 'var(--farm-text-2)', marginTop: 4 }}>{price}</span>
  </div>
);

/* ═══════════════════════════════════════════════════════════════
   FarmOverview — 主组件
   ═══════════════════════════════════════════════════════════════ */
const TutorialRestartButton = ({ t }) => {
  const tutorial = useTutorial();
  const [showMenu, setShowMenu] = useState(false);
  if (!tutorial || !tutorial.loaded) return null;
  if (tutorial.isActive) return null; // 教程进行中不显示

  const flows = tutorial.tutorialFlows || {};

  // 只有基础教程完成后才显示
  const basicDone = tutorial.featuresState?.farm_basic?.tutorial_completed;
  if (!basicDone) return null;

  return (
    <div style={{ position: 'relative' }}>
      <button className='tutorial-restart-btn' onClick={() => setShowMenu(!showMenu)} title={t('教学回看')}>
        📖 {t('教学回看')}
      </button>
      {showMenu && (
        <div className='tutorial-replay-menu'>
          {Object.entries(flows).map(([key, flow]) => {
            const fs = tutorial.featuresState?.[key];
            const completed = fs && fs.tutorial_completed;
            return (
              <button key={key} className='tutorial-replay-item'
                onClick={() => { setShowMenu(false); tutorial.restartTutorial(key); }}>
                <span>{flow.emoji}</span>
                <span>{t(flow.label)}</span>
                {completed && <span className='tutorial-replay-check'>✅</span>}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
};

const OnlineBadge = ({ t }) => {
  const [count, setCount] = useState(0);

  useEffect(() => {
    let alive = true;
    const fetchOnline = async () => {
      try {
        const { data: res } = await API.get('/api/farm/online');
        if (alive && res.success) setCount(res.data.online_count ?? 0);
      } catch { /* ignore */ }
    };
    fetchOnline();
    const timer = setInterval(fetchOnline, 30000);
    return () => { alive = false; clearInterval(timer); };
  }, []);

  return (
    <div className='farm-stat-card' style={{ background: 'rgba(74,124,63,0.08)', border: '1px solid rgba(74,124,63,0.18)' }}>
      <Users size={20} strokeWidth={2} style={{ color: 'var(--farm-leaf)', flexShrink: 0 }} />
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 11, color: 'var(--farm-text-2)', whiteSpace: 'nowrap', letterSpacing: 0.3 }}>{t('当前在线')}</div>
        <div style={{ fontSize: 17, fontWeight: 700, color: 'var(--farm-leaf)', marginTop: 1, display: 'flex', alignItems: 'center', gap: 5 }}>
          <span style={{
            display: 'inline-block', width: 8, height: 8, borderRadius: '50%',
            background: '#4caf50', boxShadow: '0 0 6px rgba(76,175,80,0.6)',
          }} />
          {count}
        </div>
      </div>
    </div>
  );
};

const FarmOverview = ({ farmData, crops, loading, loadFarm, actionLoading, doAction, t }) => {
  const [expandedPlot, setExpandedPlot] = useState(null);
  const [plantCrop, setPlantCrop] = useState('');

  const handleWater = useCallback((idx) => doAction('/api/farm/water', { plot_index: idx }), [doAction]);
  const handleWaterAll = useCallback(() => doAction('/api/farm/water/all', {}), [doAction]);
  const handleTreat = useCallback((idx) => doAction('/api/farm/treat', { plot_index: idx }), [doAction]);
  const handleFertilize = useCallback((idx) => doAction('/api/farm/fertilize', { plot_index: idx }), [doAction]);
  const handleFertilizeAll = useCallback(() => doAction('/api/farm/fertilize/all', {}), [doAction]);
  const handleHarvest = useCallback(() => doAction('/api/farm/harvest', {}), [doAction]);
  const handleHarvestStore = useCallback(() => doAction('/api/farm/harvest/store', {}), [doAction]);
  const handleBuyLand = useCallback(async () => {
    const plotPrice = Number(farmData?.plot_price);
    const price = Number.isFinite(plotPrice) ? `$${plotPrice.toFixed(2)}` : '$0.00';
    if (await confirmAction(t('购买农田'), t('确认花费') + ` ${price} ` + t('购买一块新农田？')))
      doAction('/api/farm/buyland', {});
  }, [doAction, farmData, t]);
  const handleUpgradeSoil = useCallback(async (idx) => {
    const plot = (farmData?.plots || [])[idx];
    const nextLv = (plot?.soil_level || 1) + 1;
    const prices = farmData?.soil_upgrade_prices || {};
    const rawPrice = Number(prices[String(nextLv)]);
    const price = Number.isFinite(rawPrice) ? `$${rawPrice.toFixed(2)}` : '$0.00';
    if (await confirmAction(t('升级农田'), t('确认花费') + ` ${price} ` + t('升级农田土壤？升级后该地块上的作物收获速度将加快。')))
      doAction('/api/farm/upgrade-soil', { plot_index: idx });
  }, [doAction, farmData, t]);
  const handleClearPlot = useCallback((idx) => doAction('/api/farm/clear-plot', { plot_index: idx }), [doAction]);

  // Stable handlers object — only recreated when any handler changes
  const handlers = React.useMemo(
    () => ({ handleWater, handleFertilize, handleTreat, handleUpgradeSoil, handleClearPlot }),
    [handleWater, handleFertilize, handleTreat, handleUpgradeSoil, handleClearPlot]
  );

  if (!farmData) return null;

  const plots = farmData.plots || [];
  const matureCount = plots.filter(p => p.status === 2).length;
  const growingCount = plots.filter(p => p.status === 1).length;
  const eventCount = plots.filter(p => p.status === 3 || p.status === 4).length;
  const emptyCount = plots.filter(p => p.status === 0).length;
  const needsWaterCount = plots.filter(p => p.status === 1 || p.status === 4 || (p.status === 3 && p.event_type === 'drought')).length;
  const canFertilizeCount = plots.filter(p => p.status === 1 && p.fertilized === 0).length;
  const canBuyMore = farmData.plot_count < farmData.max_plots;
  const hasQuickActions = matureCount > 0 || needsWaterCount > 0 || canFertilizeCount > 0;

  return (
    <div>
      {/* ═══ Announcement ═══ */}
      <FarmAnnouncementBar t={t} inline />

      {/* ═══ Weather Banner ═══ */}
      {farmData.weather && (
        <div className='farm-card' style={{ padding: '8px 14px', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 10, background: 'rgba(90,143,180,0.06)', border: '1px solid rgba(90,143,180,0.15)' }}>
          <span style={{ fontSize: 22 }}>{farmData.weather.emoji}</span>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, fontWeight: 700, color: 'var(--farm-text-0)' }}>{farmData.weather.name}</div>
            <div style={{ fontSize: 11, color: 'var(--farm-text-2)' }}>{farmData.weather.effects}</div>
          </div>
          {farmData.weather.ends_in > 0 && (
            <div style={{ fontSize: 11, color: 'var(--farm-text-3)', whiteSpace: 'nowrap' }}>
              ⏱ {formatDuration(farmData.weather.ends_in)}
            </div>
          )}
        </div>
      )}

      {/* ═══ Dashboard Stats ═══ */}
      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 14, alignItems: 'center' }}>
        <StatCard emoji='💰' label={t('余额')} value={formatBalance(farmData.balance)} accent='var(--farm-leaf)' />
        <StatCard emoji='🌾' label={t('地块')} value={`${farmData.plot_count} / ${farmData.max_plots}`} />
        {growingCount > 0 && <StatCard emoji='🌱' label={t('种植中')} value={growingCount} accent='var(--farm-sky)' />}
        {matureCount > 0 && <StatCard emoji='✅' label={t('可收获')} value={matureCount} accent='var(--farm-leaf)' />}
        {eventCount > 0 && <StatCard emoji='⚠️' label={t('需处理')} value={eventCount} accent='var(--farm-danger)' />}
        {emptyCount > 0 && <StatCard emoji='⬜' label={t('空地')} value={emptyCount} />}
        <OnlineBadge t={t} />
        <div style={{ marginLeft: 'auto' }}>
          <TutorialRestartButton t={t} />
        </div>
      </div>

      {/* ═══ 一键操作面板 ═══ */}
      <div className='farm-card' style={{ marginBottom: 12 }} data-tutorial='quick-actions'>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
          <div className='farm-section-title' style={{ marginBottom: 0 }}>
            <Zap size={14} style={{ marginRight: 4, verticalAlign: 'middle' }} />
            {t('一键操作')}
          </div>
          <Button size='small' icon={<RefreshCw size={12} />} theme='borderless'
            onClick={loadFarm} loading={loading} className='farm-btn'
            style={{ color: 'var(--farm-text-3)' }} />
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 8, marginBottom: 8 }}>
          <Button size='small'
            disabled={matureCount === 0 || actionLoading}
            loading={actionLoading}
            theme='solid'
            style={{ width: '100%' }}
            onClick={handleHarvest} className='farm-btn'>
            🌾 {t('一键收获出售')}{matureCount > 0 ? ` (${matureCount})` : ''}
          </Button>
          <Button size='small'
            disabled={matureCount === 0 || actionLoading}
            loading={actionLoading}
            theme='solid'
            style={{ width: '100%' }}
            onClick={handleHarvestStore} className='farm-btn'>
            📦 {t('一键收获入仓')}{matureCount > 0 ? ` (${matureCount})` : ''}
          </Button>
          <Button size='small'
            disabled={needsWaterCount === 0 || actionLoading}
            loading={actionLoading}
            theme='solid'
            style={{ width: '100%' }}
            onClick={handleWaterAll} className='farm-btn'>
            💧 {t('一键浇水')}{needsWaterCount > 0 ? ` (${needsWaterCount})` : ''}
          </Button>
          <Button size='small'
            disabled={canFertilizeCount === 0 || actionLoading}
            loading={actionLoading}
            theme='solid'
            style={{ width: '100%' }}
            onClick={handleFertilizeAll} className='farm-btn'>
            🧪 {t('一键施肥')}{canFertilizeCount > 0 ? ` (${canFertilizeCount})` : ''}
          </Button>
          <Button size='small'
            disabled={eventCount === 0 || actionLoading}
            loading={actionLoading}
            theme='solid'
            style={{ width: '100%' }}
            onClick={() => doAction('/api/farm/treat/all', {})} className='farm-btn'>
            💊 {t('一键治疗')}{eventCount > 0 ? ` (${eventCount})` : ''}
          </Button>
        </div>

        {/* 一键种植行 */}
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <Select
            size='small'
            placeholder={t('选择作物')}
            value={plantCrop || undefined}
            onChange={v => setPlantCrop(v)}
            style={{ minWidth: 140, flex: 1, maxWidth: 200 }}
            showClear
          >
            {(crops || []).map(crop => (
              <Select.Option key={crop.key} value={crop.key}>
                {crop.emoji} {crop.name}
              </Select.Option>
            ))}
          </Select>
          <Button size='small'
            disabled={emptyCount === 0 || !plantCrop || actionLoading}
            loading={actionLoading}
            theme='solid'
            onClick={() => doAction('/api/farm/plant/all', { crop_key: plantCrop })} className='farm-btn'>
            🌱 {t('一键种植')}{emptyCount > 0 ? ` (${emptyCount}${t('块空地')})` : ''}
          </Button>
        </div>
      </div>

      {/* ═══ Plot Grid ═══ */}
      <div className='farm-plot-grid'>
        {plots.map(plot => (
          <PlotCard
            key={plot.plot_index}
            plot={plot}
            farmData={farmData}
            handlers={handlers}
            actionLoading={actionLoading}
            expanded={expandedPlot === plot.plot_index && plot.status !== 0}
            onToggle={() => setExpandedPlot(expandedPlot === plot.plot_index ? null : plot.plot_index)}
            t={t}
          />
        ))}
        {canBuyMore && (
          <BuyLandCard
            price={formatBalance(farmData.plot_price)}
            onClick={handleBuyLand}
            actionLoading={actionLoading}
            t={t}
          />
        )}
      </div>

      {/* ═══ Backpack: Seeds & Items ═══ */}
      {(farmData.items || []).length > 0 && (
        <div className='farm-card' style={{ marginTop: 14 }}>
          <div className='farm-section-title' style={{ marginBottom: 8 }}>📦 {t('背包')}</div>
          {(farmData.items || []).map(item => (
            <div key={item.key} className='farm-row' style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontSize: 20 }}>{item.emoji}</span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <Text strong size='small'>{item.name}</Text>
                <span className='farm-pill' style={{ fontSize: 11, marginLeft: 6 }}>×{item.quantity}</span>
                {item.category === 'seed' && (
                  <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>
                    {t('买入')} ${item.seed_cost?.toFixed(2)}
                  </Text>
                )}
              </div>
              {item.category === 'seed' && (
                <Button size='small' theme='solid' type='warning'
                  onClick={() => doAction('/api/farm/sell/seed', { seed_key: item.crop_key, quantity: item.quantity })}
                  loading={actionLoading} className='farm-btn'>
                  💰 {t('出售')}
                </Button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default FarmOverview;
