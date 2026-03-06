import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { RefreshCw } from 'lucide-react';
import { VChart } from '@visactor/react-vchart';
import { API, showError, CHART_PALETTE } from './utils';

const { Text } = Typography;

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
      <div className='farm-card'>
        <Text type='tertiary' style={{ display: 'block', textAlign: 'center', padding: 20 }}>
          📊 {t('市场需要至少刷新2次才能显示波动图')}
        </Text>
      </div>
    );
  }

  const catItems = (historyData.items || []).filter(it => it.category === chartCat);
  const latestSnap = historyData.history[historyData.history.length - 1];
  const itemColorMap = {};
  catItems.forEach((it, idx) => { itemColorMap[it.key] = CHART_PALETTE[idx % CHART_PALETTE.length]; });

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
      if (val !== undefined) chartData.push({ time: timeStr, name: item.emoji + item.name, value: val });
    }
  }

  const displayColors = displayItems.map(it => itemColorMap[it.key]);

  const spec = {
    type: 'line',
    data: { values: chartData },
    xField: 'time', yField: 'value', seriesField: 'name',
    point: { visible: true, size: 5, style: { lineWidth: 1.5, stroke: '#fff' } },
    line: { style: { lineWidth: 2.5, lineCap: 'round' } },
    legends: { visible: false },
    crosshair: { xField: { visible: true, line: { style: { stroke: 'var(--semi-color-text-3)', lineWidth: 1, lineDash: [4, 4] } } } },
    axes: [
      { orient: 'left', title: { visible: true, text: t('倍率') + ' %' }, min: 0 },
      { orient: 'bottom', title: { visible: false }, label: { autoRotate: true, autoRotateAngle: [-45] } },
    ],
    markLine: [{ y: 100, line: { style: { stroke: '#ef4444', lineWidth: 1.5, lineDash: [6, 4] } }, label: { visible: true, text: '100%', style: { fill: '#ef4444', fontSize: 11, fontWeight: 'bold' } } }],
    tooltip: {
      dimension: {
        content: (data) => {
          const sorted = [...data].sort((a, b) => (b.datum?.value ?? b.value ?? 0) - (a.datum?.value ?? a.value ?? 0));
          return sorted.map(d => ({ key: d.datum?.name || d.name, value: (d.datum?.value ?? d.value) + '%', hasShape: true, shapeType: 'circle' }));
        },
      },
    },
    color: displayColors, height: 420,
    padding: { left: 10, right: 10, top: 10, bottom: 10 },
    animation: false,
  };

  const toggleItem = (key) => {
    const base = new Set(activeKeys);
    if (base.has(key)) { if (base.size <= 1) return; base.delete(key); } else { base.add(key); }
    setVisibleKeys(base);
  };

  const selectPreset = (mode) => {
    if (mode === 'top5') setVisibleKeys(null);
    else if (mode === 'all') setVisibleKeys(new Set(catItems.map(it => it.key)));
    else if (mode === 'up') {
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
    <div className='farm-card'>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
        <div className='farm-section-title' style={{ marginBottom: 0 }}>📊 {t('市场波动图')}</div>
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadHistory} loading={loading} className='farm-btn' />
      </div>
      <div style={{ display: 'flex', gap: 6, marginBottom: 8, flexWrap: 'wrap' }}>
        {cats.map(c => (
          <div key={c.key} className={`farm-pill ${chartCat === c.key ? 'farm-pill-blue' : ''}`}
            style={{ cursor: 'pointer' }} onClick={() => setChartCat(c.key)}>
            {c.label}
          </div>
        ))}
      </div>
      <div style={{ display: 'flex', gap: 4, marginBottom: 8, flexWrap: 'wrap' }}>
        <Button size='small' theme={!visibleKeys ? 'solid' : 'light'} className='farm-btn' style={{ fontSize: 12 }}
          onClick={() => selectPreset('top5')}>🔥 Top 5</Button>
        <Button size='small' theme={visibleKeys?.size === catItems.length ? 'solid' : 'light'} className='farm-btn' style={{ fontSize: 12 }}
          onClick={() => selectPreset('all')}>{t('全部')}</Button>
        <Button size='small' theme='light' className='farm-btn' style={{ fontSize: 12, color: '#22c55e' }}
          onClick={() => selectPreset('up')}>📈 {t('涨')}</Button>
        <Button size='small' theme='light' className='farm-btn' style={{ fontSize: 12, color: '#ef4444' }}
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
              border: `1.5px solid ${active ? clr : 'var(--farm-glass-border)'}`,
              background: active ? clr + '18' : 'transparent',
              opacity: active ? 1 : 0.45, transition: 'all 0.15s',
            }}>
              <span style={{ width: 8, height: 8, borderRadius: '50%', background: active ? clr : 'var(--semi-color-text-3)', flexShrink: 0 }} />
              <span>{it.emoji}{it.name}</span>
              {pct !== undefined && (
                <span style={{ color: pct >= 100 ? '#22c55e' : '#ef4444', fontWeight: 600 }}>{pct}%</span>
              )}
            </div>
          );
        })}
      </div>
      {chartData.length > 0 ? <VChart spec={spec} /> : (
        <Text type='tertiary' style={{ display: 'block', textAlign: 'center', padding: 20 }}>{t('暂无数据')}</Text>
      )}
    </div>
  );
};

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
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <div className='farm-pill farm-pill-blue'>⏱️ {t('下次刷新')}: {formatCountdown(countdown)}</div>
          <div className='farm-pill'>🔄 {t('每')} {marketData.refresh_hours}h</div>
        </div>
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadMarket} loading={marketLoading} className='farm-btn' />
      </div>

      <MarketChart t={t} />

      {categories.map(cat => {
        const items = (marketData.prices || []).filter(p => p.category === cat.key);
        if (items.length === 0) return null;
        return (
          <div key={cat.key} className='farm-card'>
            <div className='farm-section-title'>{cat.label}</div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
              {items.map(p => (
                <div key={p.key} className='farm-card-flat' style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 140 }}>
                  <span style={{ fontSize: 20 }}>{p.emoji}</span>
                  <div style={{ flex: 1 }}>
                    <Text size='small' strong>{p.name}</Text>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      <Text size='small' style={{ color: mColor(p.multiplier), fontWeight: 700 }}>{p.multiplier}%</Text>
                      <Text size='small' type='tertiary'>{mLabel(p.multiplier)}</Text>
                    </div>
                    <Text size='small' type='tertiary'>
                      ${p.base_price.toFixed(2)} → <span style={{ color: mColor(p.multiplier), fontWeight: 600 }}>${p.cur_price.toFixed(2)}</span>
                    </Text>
                  </div>
                </div>
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default MarketPage;
