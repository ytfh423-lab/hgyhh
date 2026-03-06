import React, { useState, lazy, Suspense } from 'react';
import { Button, Tag, Spin, Typography } from '@douyinfe/semi-ui';
import { RefreshCw, Droplets, FlaskConical, LandPlot, Wheat, Package, Box, List, ArrowUp, Pill } from 'lucide-react';
import { formatBalance, formatDuration } from './utils';

const { Text } = Typography;

const Farm3DView = lazy(() => import('../Farm3D'));

const FarmOverview = ({ farmData, loading, loadFarm, actionLoading, doAction, t }) => {
  const [viewMode, setViewMode] = useState('3d');
  const [selectedPlotIndex, setSelectedPlotIndex] = useState(null);

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

  const plots = farmData.plots || [];
  const matureCount = plots.filter(p => p.status === 2).length;
  const needsWaterCount = plots.filter(p => p.status === 1 || p.status === 4 || (p.status === 3 && p.event_type === 'drought')).length;
  const canFertilizeCount = plots.filter(p => p.status === 1 && p.fertilized === 0).length;
  const activePlots = plots.filter(p => p.status !== 0);
  const emptyPlots = plots.filter(p => p.status === 0);

  return (
    <div>
      {/* Action Bar */}
      <div className='farm-card farm-action-bar'>
        <div className='farm-action-bar-left'>
          <Button size='small' icon={viewMode === '3d' ? <List size={14} /> : <Box size={14} />}
            theme='borderless' onClick={() => setViewMode(viewMode === '3d' ? 'list' : '3d')}
            className='farm-btn' style={{ fontWeight: 600 }}>
            {viewMode === '3d' ? t('列表') : '3D'}
          </Button>
          <Button size='small' icon={<RefreshCw size={14} />} theme='borderless' onClick={loadFarm} loading={loading} className='farm-btn' />
        </div>
        <div className='farm-action-bar-right'>
          {matureCount > 0 && (
            <>
              <Button size='small' icon={<Wheat size={14} />} theme='solid'
                style={{ background: 'linear-gradient(135deg, #f59e0b, #d97706)', borderRadius: 8 }}
                onClick={handleHarvest} loading={actionLoading} className='farm-btn'>
                {t('收获出售')} ({matureCount})
              </Button>
              <Button size='small' icon={<Package size={14} />} theme='solid'
                style={{ background: 'linear-gradient(135deg, #6366f1, #4f46e5)', borderRadius: 8 }}
                onClick={handleHarvestStore} loading={actionLoading} className='farm-btn'>
                {t('收获入仓')}
              </Button>
              <div className='farm-action-divider' />
            </>
          )}
          {needsWaterCount > 0 && (
            <Button size='small' icon={<Droplets size={14} />} theme='light' onClick={handleWaterAll}
              loading={actionLoading} className='farm-btn'
              style={{ color: '#0284c7', borderColor: 'rgba(56,189,248,0.4)', borderRadius: 8 }}>
              💧 {t('全部浇水')} ({needsWaterCount})
            </Button>
          )}
          {canFertilizeCount > 0 && (
            <Button size='small' icon={<FlaskConical size={14} />} theme='light' onClick={handleFertilizeAll}
              loading={actionLoading} className='farm-btn'
              style={{ color: '#059669', borderColor: 'rgba(52,211,153,0.4)', borderRadius: 8 }}>
              🧪 {t('全部施肥')} ({canFertilizeCount})
            </Button>
          )}
          {farmData.plot_count < farmData.max_plots && (
            <Button size='small' icon={<LandPlot size={14} />} theme='light' onClick={handleBuyLand}
              loading={actionLoading} className='farm-btn' style={{ borderRadius: 8 }}>
              {t('买地')} ({formatBalance(farmData.plot_price)})
            </Button>
          )}
        </div>
      </div>

      {/* 3D View */}
      {viewMode === '3d' && (
        <div style={{ marginTop: 16 }}>
          <Suspense fallback={
            <div className='farm-3d-canvas' style={{
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              background: 'linear-gradient(180deg, #bae6fd 0%, #e0f2fe 40%, #dcfce7 100%)',
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
          <div style={{ textAlign: 'center', marginTop: 8, opacity: 0.6 }}>
            <Text type='tertiary' size='small'>🖱️ {t('拖拽旋转 · 滚轮缩放 · 点击地块查看详情')}</Text>
          </div>
        </div>
      )}

      {/* List View */}
      {viewMode === 'list' && (
        <div style={{ marginTop: 16 }}>
          {activePlots.length > 0 && (
            <div className='farm-grid farm-grid-2'>
              {activePlots.map((plot) => (
                <div key={plot.plot_index} className='farm-card' style={{
                  marginBottom: 0, padding: '16px 18px',
                  borderColor: plot.status === 3 || plot.status === 4 ? 'rgba(239,68,68,0.4)' : plot.status === 2 ? 'rgba(34,197,94,0.4)' : undefined,
                  boxShadow: plot.status === 2 ? 'var(--farm-shadow), var(--farm-glow-green)' : undefined,
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
                        <span className='farm-pill farm-pill-blue' style={{ padding: '2px 8px', fontSize: 11 }}>{plot.progress}%</span>
                      </div>
                      <div className='farm-progress' style={{ marginBottom: 6 }}>
                        <div className='farm-progress-fill' style={{ width: `${plot.progress}%`, background: 'linear-gradient(90deg, #3b82f6, #06b6d4)' }} />
                      </div>
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
                          loading={actionLoading} className='farm-btn'>{t('浇水')}</Button>
                        {plot.fertilized === 0 && (
                          <Button size='small' icon={<FlaskConical size={11} />} onClick={() => handleFertilize(plot.plot_index)}
                            loading={actionLoading} className='farm-btn'>{t('施肥')}</Button>
                        )}
                        {(plot.soil_level || 1) < (farmData.soil_max_level || 5) && (
                          <Button size='small' icon={<ArrowUp size={11} />} onClick={() => handleUpgradeSoil(plot.plot_index)}
                            loading={actionLoading} className='farm-btn'>{t('升级泥土')} Lv.{(plot.soil_level || 1)+1}</Button>
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
                          {plot.stolen_count > 0 && <Text type='warning' size='small'>⚠️ -{plot.stolen_count}</Text>}
                          <span className='farm-pill farm-pill-green' style={{ padding: '2px 8px', fontSize: 11 }}>✅ {t('已成熟')}</span>
                        </div>
                      </div>
                      {(plot.soil_level || 1) < (farmData.soil_max_level || 5) && (
                        <div style={{ display: 'flex', gap: 6, marginTop: 6 }}>
                          <Button size='small' icon={<ArrowUp size={11} />} onClick={() => handleUpgradeSoil(plot.plot_index)}
                            loading={actionLoading} className='farm-btn'>{t('升级泥土')} Lv.{(plot.soil_level || 1)+1}</Button>
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
                        <span className='farm-pill farm-pill-red' style={{ padding: '2px 8px', fontSize: 11 }}>
                          {plot.event_type === 'drought' ? '🏜️ ' + t('干旱') : '🐛 ' + t('虫害')}
                        </span>
                      </div>
                      {plot.event_type === 'drought' ? (
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Text type='danger' size='small'>💀 {formatDuration(plot.death_remain)} {t('后死亡')}</Text>
                          <Button size='small' type='danger' icon={<Droplets size={11} />}
                            onClick={() => handleWater(plot.plot_index)} loading={actionLoading} className='farm-btn'>{t('浇水')}</Button>
                        </div>
                      ) : (
                        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                          <Button size='small' type='warning' icon={<Pill size={11} />}
                            onClick={() => handleTreat(plot.plot_index)} loading={actionLoading} className='farm-btn'>{t('治疗')}</Button>
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
                        <span className='farm-pill farm-pill-amber' style={{ padding: '2px 8px', fontSize: 11 }}>{t('枯萎')}</span>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Text type='danger' size='small'>💀 {formatDuration(plot.death_remain)} {t('后死亡')}</Text>
                        <Button size='small' type='danger' icon={<Droplets size={11} />}
                          onClick={() => handleWater(plot.plot_index)} loading={actionLoading} className='farm-btn'>{t('浇水')}</Button>
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Empty plots */}
          {emptyPlots.length > 0 && (
            <div className='farm-card' style={{
              display: 'flex', flexWrap: 'wrap', gap: 6,
              border: '1px dashed var(--farm-glass-border)',
              marginTop: activePlots.length > 0 ? 14 : 0,
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
        </div>
      )}
    </div>
  );
};

export default FarmOverview;
