import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button, Spin, Tag, Typography, InputNumber, Popover, Progress,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './utils';

const { Text, Title } = Typography;

// 土壤肥力页面（A-1）
// 数据来源：GET /api/farm/soil/view
// 交互：
//   - 选一块地，展示六维参数与评分
//   - 选一种肥料施用
//   - 空地块可休耕 24-72 小时

const PARAM_META = [
  { key: 'n',       label: '氮 N',    max: 100, unit: '' },
  { key: 'p',       label: '磷 P',    max: 100, unit: '' },
  { key: 'k',       label: '钾 K',    max: 100, unit: '' },
  { key: 'ph',      label: 'PH',      max: 8.5, unit: '', min: 4.5 },
  { key: 'om',      label: '有机质',   max: 100, unit: '' },
  { key: 'fatigue', label: '连作疲劳', max: 100, unit: '', reverse: true },
];

const scoreColor = (score) => {
  if (score >= 85) return '#22a55b';
  if (score >= 70) return '#4caf50';
  if (score >= 50) return '#f7b500';
  if (score >= 30) return '#e08a3c';
  return '#d14343';
};

const yieldLabel = (mult) => {
  const pct = Math.round((mult - 1) * 100);
  if (pct >= 0) return `产量 +${pct}%`;
  return `产量 ${pct}%`;
};

const ParamBar = ({ meta, value }) => {
  let percent;
  if (meta.key === 'ph') {
    // PH 偏离 6.5（中性）越远越差
    const diff = Math.abs(value - 6.5);
    percent = Math.max(0, 100 - diff * 20);
  } else if (meta.reverse) {
    percent = 100 - value;
  } else {
    percent = value;
  }
  return (
    <div style={{ marginBottom: 6 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
        <Text type='tertiary'>{meta.label}</Text>
        <Text strong>{meta.key === 'ph' ? value.toFixed(1) : value}</Text>
      </div>
      <Progress percent={percent} stroke={scoreColor(percent)} showInfo={false} size='small' />
    </div>
  );
};

const SoilPage = ({ loadFarm, t }) => {
  const [data, setData]       = useState(null);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy]       = useState(false);
  const [selected, setSelected] = useState(null);  // plot_index
  const [fallowHours, setFallowHours] = useState(24);
  const [weatherEvents, setWeatherEvents] = useState({ active: null, recent: [] });

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/soil/view');
      if (res.success) {
        setData(res.data);
        if (res.data.plots.length > 0 && selected === null) {
          setSelected(res.data.plots[0].plot_index);
        }
      } else {
        showError(res.message || t('加载失败'));
      }
    } catch (err) { showError(t('加载失败')); }
    finally { setLoading(false); }
  }, [selected, t]);

  const loadWeatherEvents = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/weather/event');
      if (res.success) setWeatherEvents(res.data);
    } catch { /* silent */ }
  }, []);

  useEffect(() => { load(); }, [load]);
  useEffect(() => { loadWeatherEvents(); }, [loadWeatherEvents]);

  const currentPlot = useMemo(() => {
    if (!data) return null;
    return data.plots.find(p => p.plot_index === selected) || data.plots[0];
  }, [data, selected]);

  const fertilize = async (code) => {
    if (!currentPlot || busy) return;
    setBusy(true);
    try {
      const { data: res } = await API.post('/api/farm/soil/fertilize', {
        plot_index: currentPlot.plot_index,
        code,
      });
      if (res.success) {
        showSuccess(res.message);
        await load();
        loadFarm && loadFarm({ silent: true });
      } else {
        showError(res.message);
      }
    } catch (err) { showError(t('操作失败')); }
    finally { setBusy(false); }
  };

  const doFallow = async () => {
    if (!currentPlot || busy) return;
    if (currentPlot.status !== 0) {
      showError(t('地块有作物，无法休耕'));
      return;
    }
    setBusy(true);
    try {
      const { data: res } = await API.post('/api/farm/soil/fallow', {
        plot_index: currentPlot.plot_index,
        hours: fallowHours,
      });
      if (res.success) { showSuccess(res.message); await load(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
    finally { setBusy(false); }
  };

  const cancelFallow = async () => {
    if (!currentPlot || busy) return;
    setBusy(true);
    try {
      const { data: res } = await API.post('/api/farm/soil/fallow/cancel', {
        plot_index: currentPlot.plot_index,
      });
      if (res.success) { showSuccess(res.message); await load(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
    finally { setBusy(false); }
  };

  if (loading && !data) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!data) return null;

  const fmtRemain = (secs) => {
    if (secs <= 0) return '';
    const h = Math.floor(secs / 3600);
    const m = Math.floor((secs % 3600) / 60);
    return `${h}时${m}分`;
  };

  return (
    <div className='farm-card'>
      <div className='farm-section-title'>🌱 {t('土壤肥力')}</div>

      {/* 地块选择条 */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginBottom: 16 }}>
        {data.plots.map(p => {
          const active = currentPlot && p.plot_index === currentPlot.plot_index;
          const score = p.soil.score;
          return (
            <div
              key={p.plot_index}
              onClick={() => setSelected(p.plot_index)}
              style={{
                cursor: 'pointer',
                padding: '8px 12px',
                borderRadius: 10,
                border: `2px solid ${active ? scoreColor(score) : 'rgba(0,0,0,0.08)'}`,
                background: active ? 'rgba(109,187,92,0.08)' : 'rgba(0,0,0,0.02)',
                minWidth: 84,
                textAlign: 'center',
              }}
            >
              <div style={{ fontWeight: 700, fontSize: 13 }}>{p.plot_index + 1}{t('号地')}</div>
              <div style={{ color: scoreColor(score), fontWeight: 700, fontSize: 18, lineHeight: '20px' }}>{score}</div>
              <div style={{ fontSize: 10, color: '#888' }}>
                {p.status === 0
                  ? (p.fallow_remain > 0 ? `💤 ${fmtRemain(p.fallow_remain)}` : t('空地'))
                  : (p.crop_type || t('种植中'))}
              </div>
            </div>
          );
        })}
      </div>

      {currentPlot && (
        <>
          {/* 当前地块详情 */}
          <div className='farm-row' style={{ flexDirection: 'column', alignItems: 'stretch', gap: 8 }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <Title heading={5} style={{ margin: 0 }}>
                  {currentPlot.plot_index + 1}{t('号地')}
                </Title>
                <Text type='tertiary' size='small'>
                  {t('土壤等级')} Lv.{currentPlot.soil_level}
                  {currentPlot.last_crop_type ? ` · ${t('上一轮')}: ${currentPlot.last_crop_type}` : ''}
                </Text>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{
                  color: scoreColor(currentPlot.soil.score),
                  fontWeight: 800, fontSize: 28, lineHeight: '30px',
                }}>{currentPlot.soil.score}</div>
                <Tag size='small' color='green'>{yieldLabel(currentPlot.soil.yield)}</Tag>
              </div>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '0 16px' }}>
              {PARAM_META.map(meta => (
                <ParamBar key={meta.key} meta={meta} value={currentPlot.soil[meta.key]} />
              ))}
            </div>
          </div>

          {/* 肥料区 */}
          <div style={{ marginTop: 20, fontSize: 14, fontWeight: 700, marginBottom: 10 }}>
            💧 {t('施肥')}
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: 10 }}>
            {data.fertilizers.map(f => (
              <Popover
                key={f.code}
                content={<div style={{ maxWidth: 180, fontSize: 12 }}>{f.effect}</div>}
                position='top'
              >
                <div
                  className='farm-row'
                  style={{
                    flexDirection: 'column', alignItems: 'stretch', gap: 6,
                    padding: 10, marginBottom: 0,
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span style={{ fontSize: 22 }}>{f.emoji}</span>
                    <Text strong style={{ fontSize: 13 }}>{f.name}</Text>
                  </div>
                  <Text type='tertiary' size='small' style={{ fontSize: 11, minHeight: 30 }}>
                    {f.effect}
                  </Text>
                  <Button
                    size='small' theme='solid' className='farm-btn'
                    loading={busy}
                    onClick={() => fertilize(f.code)}
                  >
                    ${(f.price / 500000).toFixed(2)}
                  </Button>
                </div>
              </Popover>
            ))}
          </div>

          {/* 休耕区 */}
          <div style={{ marginTop: 20, fontSize: 14, fontWeight: 700, marginBottom: 10 }}>
            💤 {t('休耕')}
          </div>
          <div className='farm-row' style={{ alignItems: 'center', gap: 12 }}>
            {currentPlot.fallow_remain > 0 ? (
              <>
                <Tag size='large' color='blue'>
                  💤 {t('休耕中')} · {t('剩余')} {fmtRemain(currentPlot.fallow_remain)}
                </Tag>
                <Button onClick={cancelFallow} loading={busy}>{t('结束休耕')}</Button>
              </>
            ) : currentPlot.status !== 0 ? (
              <Text type='tertiary'>{t('地块有作物，请先收获后再休耕')}</Text>
            ) : (
              <>
                <Text>{t('时长')}:</Text>
                <InputNumber
                  min={data.fallow.min_hours}
                  max={data.fallow.max_hours}
                  value={fallowHours}
                  onChange={v => setFallowHours(Number(v) || data.fallow.min_hours)}
                  suffix={t('小时')}
                  style={{ width: 140 }}
                />
                <Button theme='solid' onClick={doFallow} loading={busy} className='farm-btn'>
                  {t('开始休耕')}
                </Button>
                <Text type='tertiary' size='small'>
                  {t('每小时自动回补 N/P/K +2，疲劳 -3')}
                </Text>
              </>
            )}
          </div>

          {/* 最近天气事件（A-2） */}
          {(weatherEvents.active || weatherEvents.recent.length > 0) && (
            <>
              <div style={{ marginTop: 20, fontSize: 14, fontWeight: 700, marginBottom: 10 }}>
                🌦️ {t('最近天气事件')}
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                {weatherEvents.active && (
                  <div style={{
                    padding: '6px 10px', borderRadius: 6, display: 'flex',
                    alignItems: 'center', gap: 8,
                    background: 'rgba(209,160,58,0.12)',
                    border: '1px solid rgba(209,160,58,0.3)',
                  }}>
                    <span style={{ fontSize: 20 }}>{weatherEvents.active.emoji}</span>
                    <div style={{ flex: 1 }}>
                      <Text strong style={{ fontSize: 12 }}>{weatherEvents.active.name}</Text>
                      <Text type='tertiary' size='small' style={{ display: 'block', fontSize: 11 }}>
                        {weatherEvents.active.narrative}
                      </Text>
                    </div>
                    <Tag size='small' color='orange'>{t('进行中')}</Tag>
                  </div>
                )}
                {weatherEvents.recent
                  .filter(ev => ev.ended)
                  .slice(0, 5)
                  .map(ev => (
                    <div key={ev.id} style={{
                      padding: '6px 10px', borderRadius: 6, display: 'flex',
                      alignItems: 'center', gap: 8,
                      background: 'rgba(0,0,0,0.03)',
                    }}>
                      <span style={{ fontSize: 18, opacity: 0.7 }}>{ev.emoji}</span>
                      <Text size='small' style={{ flex: 1, fontSize: 11 }}>{ev.name}</Text>
                      <Text type='tertiary' size='small' style={{ fontSize: 10 }}>
                        {new Date(ev.started_at * 1000).toLocaleString()}
                      </Text>
                    </div>
                  ))}
              </div>
            </>
          )}

          {/* 提示 */}
          <div style={{ marginTop: 16, padding: 12, background: 'rgba(79,143,247,0.08)', borderRadius: 8, fontSize: 12 }}>
            <Text type='tertiary' size='small'>
              💡 {t('土壤评分 ≥ 70 时收获数量提升；种植同一作物会加重连作疲劳，建议轮作或休耕。稀有天气事件会临时改变土壤参数。')}
            </Text>
          </div>
        </>
      )}
    </div>
  );
};

export default SoilPage;
