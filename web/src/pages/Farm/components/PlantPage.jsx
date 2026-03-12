import React, { useState } from 'react';
import { Button, Empty, Tag, Typography } from '@douyinfe/semi-ui';
import { formatDuration } from './utils';
import { showError } from './utils';
import tutorialEvents from './tutorialEvents';

const { Text } = Typography;

const TIER_COLORS = {
  sprint: '#faad14',
  active: '#1890ff',
  balanced: '#52c41a',
  afk: '#722ed1',
};

const TAG_COLORS = {
  '睡前种植': '#722ed1',
  '适合离线': '#722ed1',
  '高总收益': '#f5222d',
  '快速回本': '#faad14',
  '当季作物': '#52c41a',
  '高效快刷': '#fa8c16',
};

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
    <div data-tutorial='plant-page'>
      {/* Crop selection */}
      <div className='farm-card'>
        <div className='farm-section-title'>🌱 {t('选择作物')}</div>
        <div className='farm-grid farm-grid-2' data-tutorial='crop-grid'>
          {crops.map((crop) => (
            <div key={crop.key}
              className={`farm-item-card ${selectedCrop === crop.key ? 'selected' : ''}`}
              onClick={() => { setSelectedCrop(crop.key); tutorialEvents.emitSuccess('select-crop', { cropKey: crop.key }); }}
              style={{ textAlign: 'left', padding: '10px 14px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4, flexWrap: 'wrap' }}>
                <span style={{ fontSize: 22 }}>{crop.emoji}</span>
                <Text strong style={{ fontSize: 14 }}>{crop.name}</Text>
                {crop.tier_name && (
                  <span style={{ padding: '1px 6px', fontSize: 10, borderRadius: 3, background: TIER_COLORS[crop.tier] || '#999', color: '#fff' }}>{crop.tier_name}</span>
                )}
                {crop.in_season !== undefined && (
                  crop.in_season
                    ? <span style={{ padding: '1px 6px', fontSize: 10, borderRadius: 3, background: '#52c41a', color: '#fff' }}>🏷️{t('应季')}</span>
                    : <span style={{ padding: '1px 6px', fontSize: 10, borderRadius: 3, background: '#fa8c16', color: '#fff' }}>📈{t('反季')}</span>
                )}
                <span className='farm-pill farm-pill-green' style={{ padding: '1px 8px', fontSize: 11 }}>${crop.seed_cost?.toFixed(2)}</span>
              </div>
              <div style={{ display: 'flex', gap: 8, fontSize: 11, flexWrap: 'wrap', paddingLeft: 30 }}>
                <Text type='tertiary'>⏱{formatDuration(crop.grow_secs)}</Text>
                <Text type='tertiary'>📦1~{crop.max_yield}</Text>
                <Text type='tertiary'>💰${crop.max_profit?.toFixed(2)}</Text>
              </div>
              {crop.tags && crop.tags.length > 0 && (
                <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', paddingLeft: 30, marginTop: 3 }}>
                  {crop.tags.map((tag) => (
                    <span key={tag} style={{ fontSize: 9, padding: '0 5px', borderRadius: 2, background: (TAG_COLORS[tag] || '#999') + '22', color: TAG_COLORS[tag] || '#999', border: `1px solid ${TAG_COLORS[tag] || '#999'}44` }}>{tag}</span>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Selected crop detail */}
      {activeCrop && (
        <div className='farm-card farm-card-glow-blue'>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <span style={{ fontSize: 28 }}>{activeCrop.emoji}</span>
            <div style={{ flex: 1 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 2 }}>
                <Text strong style={{ fontSize: 15 }}>{activeCrop.name}</Text>
                {activeCrop.tier_name && (
                  <span style={{ padding: '1px 6px', fontSize: 10, borderRadius: 3, background: TIER_COLORS[activeCrop.tier] || '#999', color: '#fff' }}>{activeCrop.tier_name}</span>
                )}
              </div>
              <div style={{ fontSize: 12 }}>
                <Text type='tertiary'>
                  {t('种子')} ${activeCrop.seed_cost?.toFixed(2)} · {t('生长')} {formatDuration(activeCrop.grow_secs)} · {t('产量')} 1~{activeCrop.max_yield}
                </Text>
              </div>
              <div style={{ fontSize: 12 }}>
                <Text type='tertiary'>
                  {t('最高利润')} <Text style={{ color: '#52c41a', fontSize: 12 }}>${activeCrop.max_profit?.toFixed(2)}</Text>
                  {activeCrop.avg_profit_per_hour > 0 && (
                    <> · {t('时均')} ${activeCrop.avg_profit_per_hour?.toFixed(2)}/h</>
                  )}
                </Text>
              </div>
              {activeCrop.tags && activeCrop.tags.length > 0 && (
                <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginTop: 4 }}>
                  {activeCrop.tags.map((tag) => (
                    <span key={tag} style={{ fontSize: 10, padding: '1px 6px', borderRadius: 3, background: (TAG_COLORS[tag] || '#999') + '22', color: TAG_COLORS[tag] || '#999', border: `1px solid ${TAG_COLORS[tag] || '#999'}44` }}>{tag}</span>
                  ))}
                </div>
              )}
              {activeCrop.in_season !== undefined && (
                <div style={{ marginTop: 6, padding: '6px 10px', borderRadius: 6, background: activeCrop.in_season ? '#f6ffed' : '#fff7e6', border: `1px solid ${activeCrop.in_season ? '#b7eb8f' : '#ffd591'}` }}>
                  <Text style={{ fontSize: 12, color: activeCrop.in_season ? '#389e0d' : '#d46b08' }}>
                    {activeCrop.in_season ? '🏷️ 应季优势：' : '📈 反季效果：'}
                    {t('生长')}{activeCrop.season_grow_pct}% · {t('产量')}{activeCrop.season_yield_pct}% · {t('事件')}{activeCrop.season_event_info}
                  </Text>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Plot selection */}
      <div className='farm-card' data-tutorial='plot-buttons'>
        <div className='farm-section-title'>📍 {t('选择空地种植')}</div>
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
                className='farm-btn'
                style={{ minWidth: 80 }}>
                ⬜ {plot.plot_index + 1}{t('号地')}
              </Button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default PlantPage;
