import React, { useEffect, useState } from 'react';
import { API } from '../../../helpers';
import { formatDuration } from './utils';

// 天气事件横幅（A-2）
// 叠加在基础天气横幅之下，只在有活跃事件时渲染。
// 轻量轮询（60s）刷新，失败静默。

const severityStyle = {
  1: { bg: 'rgba(109,187,92,0.12)',  border: 'rgba(109,187,92,0.35)',  text: '#3d7d2f' },
  2: { bg: 'rgba(209,160,58,0.12)',  border: 'rgba(209,160,58,0.35)',  text: '#a27016' },
  3: { bg: 'rgba(209,67,67,0.12)',   border: 'rgba(209,67,67,0.35)',   text: '#a82525' },
};

const WeatherEventBanner = ({ t }) => {
  const [active, setActive] = useState(null);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const { data: res } = await API.get('/api/farm/weather/event');
        if (!alive) return;
        if (res.success) setActive(res.data.active || null);
      } catch { /* silent */ }
    };
    load();
    const id = setInterval(load, 60 * 1000);
    return () => { alive = false; clearInterval(id); };
  }, []);

  if (!active) return null;

  const st = severityStyle[active.severity] || severityStyle[1];
  return (
    <div
      className='farm-card'
      style={{
        padding: '8px 14px',
        marginBottom: 12,
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        background: st.bg,
        border: `1px solid ${st.border}`,
      }}
    >
      <span style={{ fontSize: 22 }}>{active.emoji}</span>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 13, fontWeight: 700, color: st.text }}>
          {t('天气事件')} · {active.name}
        </div>
        <div style={{ fontSize: 11, color: 'var(--farm-text-2)' }}>
          {active.narrative}
        </div>
      </div>
      {active.remain > 0 && (
        <div style={{ fontSize: 11, color: 'var(--farm-text-3)', whiteSpace: 'nowrap' }}>
          ⏱ {formatDuration(active.remain)}
        </div>
      )}
    </div>
  );
};

export default WeatherEventBanner;
