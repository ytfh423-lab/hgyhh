import React, { useState } from 'react';
import { Button, Typography } from '@douyinfe/semi-ui';
import { RefreshCw, Droplets, FlaskConical, Wheat, Package, ArrowUp, Pill, Plus, Sprout } from 'lucide-react';
import { formatBalance, formatDuration } from './utils';

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

const PlotCard = ({ plot, farmData, handlers, actionLoading, expanded, onToggle, t }) => {
  const { handleWater, handleFertilize, handleTreat, handleUpgradeSoil } = handlers;
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
            {st === 1 && `⏳ ${formatDuration(plot.remaining)}`}
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
          🌾 {t('点击展开操作')}
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

      {/* ── Expanded action panel ── */}
      {expanded && (
        <div className='farm-plot-actions'>
          {needsWater && (
            <Button size='small' icon={<Droplets size={12} />}
              onClick={e => { e.stopPropagation(); handleWater(plot.plot_index); }}
              loading={actionLoading} className='farm-btn'
              style={{ background: 'rgba(90,143,180,0.12)', border: '1px solid rgba(90,143,180,0.3)', color: 'var(--farm-sky)' }}>
              {t('浇水')}
            </Button>
          )}
          {st === 1 && plot.fertilized === 0 && (
            <Button size='small' icon={<FlaskConical size={12} />}
              onClick={e => { e.stopPropagation(); handleFertilize(plot.plot_index); }}
              loading={actionLoading} className='farm-btn'
              style={{ background: 'rgba(74,124,63,0.12)', border: '1px solid rgba(74,124,63,0.3)', color: 'var(--farm-leaf)' }}>
              {t('施肥')}
            </Button>
          )}
          {st === 3 && plot.event_type !== 'drought' && (
            <Button size='small' icon={<Pill size={12} />}
              onClick={e => { e.stopPropagation(); handleTreat(plot.plot_index); }}
              loading={actionLoading} className='farm-btn'
              style={{ background: 'rgba(200,146,42,0.12)', border: '1px solid rgba(200,146,42,0.3)', color: 'var(--farm-harvest)' }}>
              {t('治疗')}
            </Button>
          )}
          {soilLv < soilMax && (
            <Button size='small' icon={<ArrowUp size={12} />}
              onClick={e => { e.stopPropagation(); handleUpgradeSoil(plot.plot_index); }}
              loading={actionLoading} className='farm-btn'
              style={{ background: 'rgba(138,108,176,0.12)', border: '1px solid rgba(138,108,176,0.3)', color: '#b094d0' }}>
              {t('升级')} Lv.{soilLv + 1}
            </Button>
          )}
        </div>
      )}
    </div>
  );
};

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
const FarmOverview = ({ farmData, loading, loadFarm, actionLoading, doAction, t }) => {
  const [expandedPlot, setExpandedPlot] = useState(null);

  if (!farmData) return null;

  const handleWater = (idx) => doAction('/api/farm/water', { plot_index: idx });
  const handleWaterAll = () => doAction('/api/farm/water/all', {});
  const handleTreat = (idx) => doAction('/api/farm/treat', { plot_index: idx });
  const handleFertilize = (idx) => doAction('/api/farm/fertilize', { plot_index: idx });
  const handleFertilizeAll = () => doAction('/api/farm/fertilize/all', {});
  const handleHarvest = () => doAction('/api/farm/harvest', {});
  const handleHarvestStore = () => doAction('/api/farm/harvest/store', {});
  const handleBuyLand = () => doAction('/api/farm/buyland', {});
  const handleUpgradeSoil = (idx) => doAction('/api/farm/upgrade-soil', { plot_index: idx });

  const handlers = { handleWater, handleFertilize, handleTreat, handleUpgradeSoil };

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
      {/* ═══ Dashboard Stats ═══ */}
      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 14 }}>
        <StatCard emoji='💰' label={t('余额')} value={formatBalance(farmData.balance)} accent='var(--farm-leaf)' />
        <StatCard emoji='🌾' label={t('地块')} value={`${farmData.plot_count} / ${farmData.max_plots}`} />
        {growingCount > 0 && <StatCard emoji='🌱' label={t('种植中')} value={growingCount} accent='var(--farm-sky)' />}
        {matureCount > 0 && <StatCard emoji='✅' label={t('可收获')} value={matureCount} accent='var(--farm-leaf)' />}
        {eventCount > 0 && <StatCard emoji='⚠️' label={t('需处理')} value={eventCount} accent='var(--farm-danger)' />}
        {emptyCount > 0 && <StatCard emoji='⬜' label={t('空地')} value={emptyCount} />}
      </div>

      {/* ═══ Quick Actions Bar ═══ */}
      <div className='farm-overview-actions'>
        <Button size='small' icon={<RefreshCw size={13} />} theme='borderless'
          onClick={loadFarm} loading={loading} className='farm-btn'
          style={{ color: 'var(--farm-text-2)' }} />
        <div style={{ width: 1, height: 20, background: 'var(--farm-border-strong)', margin: '0 2px' }} />

        {matureCount > 0 && (
          <>
            <Button size='small' icon={<Wheat size={13} />} theme='solid'
              style={{ background: 'linear-gradient(135deg, var(--farm-harvest), var(--farm-soil))', borderRadius: 6 }}
              onClick={handleHarvest} loading={actionLoading} className='farm-btn'>
              🌾 {t('收获出售')} ({matureCount})
            </Button>
            <Button size='small' icon={<Package size={13} />} theme='solid'
              style={{ background: 'linear-gradient(135deg, var(--farm-soil-light, #a0845e), var(--farm-soil))', borderRadius: 6 }}
              onClick={handleHarvestStore} loading={actionLoading} className='farm-btn'>
              📦 {t('收获入仓')}
            </Button>
          </>
        )}
        {needsWaterCount > 0 && (
          <Button size='small' icon={<Droplets size={13} />}
            style={{ background: 'rgba(90,143,180,0.1)', border: '1px solid rgba(90,143,180,0.25)', color: 'var(--farm-sky)', borderRadius: 6 }}
            onClick={handleWaterAll} loading={actionLoading} className='farm-btn'>
            💧 {t('全部浇水')} ({needsWaterCount})
          </Button>
        )}
        {canFertilizeCount > 0 && (
          <Button size='small' icon={<FlaskConical size={13} />}
            style={{ background: 'rgba(74,124,63,0.1)', border: '1px solid rgba(74,124,63,0.25)', color: 'var(--farm-leaf)', borderRadius: 6 }}
            onClick={handleFertilizeAll} loading={actionLoading} className='farm-btn'>
            🧪 {t('全部施肥')} ({canFertilizeCount})
          </Button>
        )}
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
    </div>
  );
};

export default FarmOverview;
