import React, { useState } from 'react';
import { Button, Empty, Tag, Typography } from '@douyinfe/semi-ui';
import { formatDuration } from './utils';
import { showError } from './utils';
import tutorialEvents from './tutorialEvents';

const { Text } = Typography;

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
        <div className='farm-grid farm-grid-2'>
          {crops.map((crop) => (
            <div key={crop.key}
              className={`farm-item-card ${selectedCrop === crop.key ? 'selected' : ''}`}
              onClick={() => { setSelectedCrop(crop.key); tutorialEvents.emitAction('select-crop', { cropKey: crop.key }); }}
              style={{ textAlign: 'left', padding: '10px 14px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                <span style={{ fontSize: 22 }}>{crop.emoji}</span>
                <Text strong style={{ fontSize: 14 }}>{crop.name}</Text>
                <span className='farm-pill farm-pill-green' style={{ padding: '1px 8px', fontSize: 11 }}>${crop.seed_cost?.toFixed(2)}</span>
              </div>
              <div style={{ display: 'flex', gap: 10, fontSize: 11, flexWrap: 'wrap', paddingLeft: 30 }}>
                <Text type='tertiary'>⏱{formatDuration(crop.grow_secs)}</Text>
                <Text type='tertiary'>📦1~{crop.max_yield}</Text>
                <Text type='tertiary'>🏆${crop.max_value?.toFixed(2)}</Text>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Selected crop detail */}
      {activeCrop && (
        <div className='farm-card farm-card-glow-blue'>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <span style={{ fontSize: 28 }}>{activeCrop.emoji}</span>
            <div>
              <Text strong style={{ fontSize: 15 }}>{activeCrop.name}</Text>
              <div style={{ fontSize: 12 }}>
                <Text type='tertiary'>
                  {t('种子')} ${activeCrop.seed_cost?.toFixed(2)} · {t('生长')} {formatDuration(activeCrop.grow_secs)} · {t('产量')} 1~{activeCrop.max_yield} · {t('最高')} ${activeCrop.max_value?.toFixed(2)}
                </Text>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Plot selection */}
      <div className='farm-card' data-tutorial='plot-selection'>
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
