import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Typography } from '@douyinfe/semi-ui';
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
    markLine: [{ y: 100, line: { style: { stroke: '#b84233', lineWidth: 1.5, lineDash: [6, 4] } }, label: { visible: true, text: '100%', style: { fill: '#b84233', fontSize: 11, fontWeight: 'bold' } } }],
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
        <Button size='small' theme='light' className='farm-btn' style={{ fontSize: 12, color: 'var(--farm-leaf)' }}
          onClick={() => selectPreset('up')}>📈 {t('涨')}</Button>
        <Button size='small' theme='light' className='farm-btn' style={{ fontSize: 12, color: 'var(--farm-danger)' }}
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
                <span style={{ color: pct >= 100 ? 'var(--farm-leaf)' : 'var(--farm-danger)', fontWeight: 600 }}>{pct}%</span>
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

/* ═══════════════════════════════════════════════════════════════
   SortIcon — 排序方向指示器
   ═══════════════════════════════════════════════════════════════ */
const SortIcon = ({ field, sortBy, sortDir }) => {
  if (sortBy !== field) return <span style={{ opacity: 0.3, fontSize: 10 }}>⇅</span>;
  return <span style={{ fontSize: 10 }}>{sortDir === 'asc' ? '▲' : '▼'}</span>;
};

/* ═══════════════════════════════════════════════════════════════
   MarketPage — 交易所风格市场主组件
   ═══════════════════════════════════════════════════════════════ */
const MarketPage = ({ t }) => {
  const [marketData, setMarketData] = useState(null);
  const [marketLoading, setMarketLoading] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [activeTab, setActiveTab] = useState('all');
  const [sortBy, setSortBy] = useState('change');
  const [sortDir, setSortDir] = useState('desc');

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

  const tabs = [
    { key: 'all', label: '📊 ' + t('全部') },
    { key: 'crop', label: '🌾 ' + t('作物') },
    { key: 'fish', label: '🐟 ' + t('鱼类') },
    { key: 'meat', label: '🥩 ' + t('肉类') },
    { key: 'recipe', label: '🏭 ' + t('加工品') },
  ];

  const allPrices = marketData.prices || [];
  const filtered = activeTab === 'all' ? allPrices : allPrices.filter(p => p.category === activeTab);

  const handleSort = (field) => {
    if (sortBy === field) setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    else { setSortBy(field); setSortDir('desc'); }
  };

  const sorted = [...filtered].sort((a, b) => {
    let va, vb;
    switch (sortBy) {
      case 'name': va = a.name; vb = b.name; return sortDir === 'asc' ? va.localeCompare(vb) : vb.localeCompare(va);
      case 'price': va = a.cur_price; vb = b.cur_price; break;
      case 'change': va = a.multiplier; vb = b.multiplier; break;
      case 'base': va = a.base_price; vb = b.base_price; break;
      default: va = a.multiplier; vb = b.multiplier;
    }
    return sortDir === 'asc' ? va - vb : vb - va;
  });

  const upCount = allPrices.filter(p => p.multiplier > 100).length;
  const downCount = allPrices.filter(p => p.multiplier < 100).length;
  const flatCount = allPrices.filter(p => p.multiplier === 100).length;

  const changeClass = (m) => m > 100 ? 'farm-market-change-up' : m < 100 ? 'farm-market-change-down' : 'farm-market-change-flat';
  const changeText = (m) => {
    const diff = m - 100;
    if (diff > 0) return `+${diff.toFixed(0)}%`;
    if (diff < 0) return `${diff.toFixed(0)}%`;
    return '0%';
  };
  const trendArrow = (m) => {
    if (m >= 150) return '🔥';
    if (m >= 120) return '↑↑';
    if (m > 100) return '↑';
    if (m === 100) return '—';
    if (m >= 80) return '↓';
    if (m >= 50) return '↓↓';
    return '💀';
  };

  return (
    <div>
      {/* ═══ Top Bar: countdown + refresh ═══ */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
          <div className='farm-pill farm-pill-blue'>⏱️ {t('下次刷新')}: {formatCountdown(countdown)}</div>
          <div className='farm-pill'>🔄 {t('每')} {marketData.refresh_hours}h</div>
          <div className='farm-market-summary'>
            <span className='farm-market-summary-item'>
              <span style={{ color: 'var(--farm-leaf)', fontWeight: 700 }}>▲ {upCount}</span>
            </span>
            <span className='farm-market-summary-item'>
              <span style={{ color: 'var(--farm-danger)', fontWeight: 700 }}>▼ {downCount}</span>
            </span>
            {flatCount > 0 && (
              <span className='farm-market-summary-item'>
                <span style={{ fontWeight: 700 }}>— {flatCount}</span>
              </span>
            )}
          </div>
        </div>
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadMarket} loading={marketLoading} className='farm-btn' />
      </div>

      {/* ═══ Chart ═══ */}
      <MarketChart t={t} />

      {/* ═══ Category Tabs ═══ */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 16, marginBottom: 10 }}>
        <div className='farm-market-tabs'>
          {tabs.map(tab => (
            <div key={tab.key}
              className={`farm-market-tab ${activeTab === tab.key ? 'active' : ''}`}
              onClick={() => setActiveTab(tab.key)}>
              {tab.label}
              {tab.key !== 'all' && (
                <span style={{ marginLeft: 4, opacity: 0.6, fontSize: 11 }}>
                  ({allPrices.filter(p => p.category === tab.key).length})
                </span>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* ═══ Exchange Table ═══ */}
      <div className='farm-market-table'>
        {/* Header */}
        <div className='farm-market-thead'>
          <div className='farm-market-col-name'>
            <div className={`farm-market-th ${sortBy === 'name' ? 'active' : ''}`} onClick={() => handleSort('name')}>
              {t('名称')} <SortIcon field='name' sortBy={sortBy} sortDir={sortDir} />
            </div>
          </div>
          <div className='farm-market-col-price'>
            <div className={`farm-market-th ${sortBy === 'price' ? 'active' : ''}`} onClick={() => handleSort('price')} style={{ marginLeft: 'auto' }}>
              {t('现价')} <SortIcon field='price' sortBy={sortBy} sortDir={sortDir} />
            </div>
          </div>
          <div className='farm-market-col-change'>
            <div className={`farm-market-th ${sortBy === 'change' ? 'active' : ''}`} onClick={() => handleSort('change')} style={{ marginLeft: 'auto' }}>
              {t('涨跌')} <SortIcon field='change' sortBy={sortBy} sortDir={sortDir} />
            </div>
          </div>
          <div className='farm-market-col-base'>
            <div className={`farm-market-th ${sortBy === 'base' ? 'active' : ''}`} onClick={() => handleSort('base')} style={{ marginLeft: 'auto' }}>
              {t('基价')} <SortIcon field='base' sortBy={sortBy} sortDir={sortDir} />
            </div>
          </div>
          <div className='farm-market-col-trend'>
            <span style={{ fontSize: 11 }}>{t('趋势')}</span>
          </div>
        </div>

        {/* Rows */}
        {sorted.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 32, color: 'var(--farm-text-2)' }}>
            {t('暂无数据')}
          </div>
        ) : sorted.map(p => (
          <div key={p.key} className='farm-market-row'>
            <div className='farm-market-col-name'>
              <span style={{ fontSize: 20, lineHeight: 1, flexShrink: 0 }}>{p.emoji}</span>
              <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--farm-text-0)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {p.name}
              </span>
            </div>
            <div className='farm-market-col-price'>
              <span style={{ fontSize: 14, fontWeight: 700, color: p.multiplier >= 100 ? 'var(--farm-leaf)' : 'var(--farm-danger)' }}>
                ${p.cur_price.toFixed(2)}
              </span>
            </div>
            <div className='farm-market-col-change'>
              <span className={`farm-market-change-pill ${changeClass(p.multiplier)}`}>
                {changeText(p.multiplier)}
              </span>
            </div>
            <div className='farm-market-col-base'>
              <span style={{ fontSize: 12, color: 'var(--farm-text-2)' }}>
                ${p.base_price.toFixed(2)}
              </span>
            </div>
            <div className='farm-market-col-trend'>
              <span style={{ fontSize: 14 }}>{trendArrow(p.multiplier)}</span>
            </div>
          </div>
        ))}
      </div>

      {/* ═══ Footer info ═══ */}
      <div style={{ textAlign: 'center', padding: '12px 0', fontSize: 11, color: 'var(--farm-text-3)' }}>
        {t('共')} {filtered.length} {t('项')} · {t('点击表头排序')}
      </div>
    </div>
  );
};

export default MarketPage;
