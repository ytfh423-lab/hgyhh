import React, { useState, useEffect } from 'react';
import { formatBalance } from './utils';

function useBetaCountdown(endTimeStr) {
  const [remaining, setRemaining] = useState(null);

  useEffect(() => {
    if (!endTimeStr) { setRemaining(null); return; }
    const endMs = new Date(endTimeStr).getTime();
    if (isNaN(endMs)) { setRemaining(null); return; }

    const tick = () => {
      const diff = endMs - Date.now();
      setRemaining(diff > 0 ? diff : 0);
    };
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [endTimeStr]);

  if (remaining === null || remaining <= 0) return null;
  const d = Math.floor(remaining / 86400000);
  const h = Math.floor((remaining % 86400000) / 3600000);
  const m = Math.floor((remaining % 3600000) / 60000);
  const s = Math.floor((remaining % 60000) / 1000);
  return { days: d, hours: h, minutes: m, seconds: s };
}

const BetaCountdownBanner = ({ farmData, t }) => {
  const countdown = useBetaCountdown(farmData?.beta_end_time);
  if (!farmData?.beta_enabled || !countdown) return null;

  const urgent = countdown.days === 0 && countdown.hours < 6;
  const bannerStyle = {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    padding: '6px 16px',
    borderRadius: 10,
    fontSize: 13,
    fontWeight: 600,
    background: urgent
      ? 'linear-gradient(90deg, rgba(239,68,68,0.15), rgba(239,68,68,0.05))'
      : 'linear-gradient(90deg, rgba(251,191,36,0.12), rgba(251,191,36,0.04))',
    border: urgent
      ? '1px solid rgba(239,68,68,0.25)'
      : '1px solid rgba(251,191,36,0.18)',
    color: urgent ? '#fca5a5' : '#fde68a',
    animation: urgent ? 'farm-beta-pulse 2s ease-in-out infinite' : 'none',
  };

  const unitStyle = {
    display: 'inline-flex',
    flexDirection: 'column',
    alignItems: 'center',
    minWidth: 32,
  };
  const numStyle = { fontSize: 16, fontWeight: 700, fontVariantNumeric: 'tabular-nums' };
  const labelStyle = { fontSize: 9, opacity: 0.7, fontWeight: 400 };

  return (
    <div style={bannerStyle}>
      <span style={{ fontSize: 15 }}>⏱️</span>
      <span>{t('内测剩余')}</span>
      <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
        {countdown.days > 0 && (
          <>
            <span style={unitStyle}>
              <span style={numStyle}>{countdown.days}</span>
              <span style={labelStyle}>{t('天')}</span>
            </span>
            <span style={{ opacity: 0.4 }}>:</span>
          </>
        )}
        <span style={unitStyle}>
          <span style={numStyle}>{String(countdown.hours).padStart(2, '0')}</span>
          <span style={labelStyle}>{t('时')}</span>
        </span>
        <span style={{ opacity: 0.4 }}>:</span>
        <span style={unitStyle}>
          <span style={numStyle}>{String(countdown.minutes).padStart(2, '0')}</span>
          <span style={labelStyle}>{t('分')}</span>
        </span>
        <span style={{ opacity: 0.4 }}>:</span>
        <span style={unitStyle}>
          <span style={numStyle}>{String(countdown.seconds).padStart(2, '0')}</span>
          <span style={labelStyle}>{t('秒')}</span>
        </span>
      </div>
    </div>
  );
};

const StatusBar = ({ farmData, t }) => {
  if (!farmData) return null;

  const hasExtra = farmData.prestige_level > 0 || farmData.dog || (farmData.items && farmData.items.length > 0);

  return (
    <>
      <BetaCountdownBanner farmData={farmData} t={t} />
      <div className='farm-statusbar'>
        <div className='farm-pill farm-pill-green'>
          <span>💰</span>
          <span>{formatBalance(farmData.balance)}</span>
        </div>
        <div className='farm-pill farm-pill-blue'>
          <span>⭐</span>
          <span>Lv.{farmData.user_level || 1}</span>
        </div>
        <div className='farm-pill'>
          <span>🌾</span>
          <span>{farmData.plot_count}/{farmData.max_plots}</span>
        </div>
        {farmData.weather && (
          <div className='farm-pill farm-pill-cyan'>
            <span>{farmData.weather.emoji}</span>
            <span>{farmData.weather.name}</span>
          </div>
        )}
        {hasExtra && <div className='farm-stat-sep' />}
        {farmData.prestige_level > 0 && (
          <div className='farm-pill farm-pill-purple'>
            <span>🔄</span>
            <span>P{farmData.prestige_level} (+{farmData.prestige_bonus}%)</span>
          </div>
        )}
        {farmData.dog && (
          <div className={`farm-pill ${farmData.dog.hunger > 0 ? 'farm-pill-green' : 'farm-pill-red'}`}>
            <span>{farmData.dog.level === 2 ? '🐕' : '🐶'}</span>
            <span>{farmData.dog.hunger}%</span>
          </div>
        )}
        {farmData.items && farmData.items.map((item) => (
          <div key={item.key} className='farm-pill farm-pill-amber'>
            <span>{item.emoji}</span>
            <span>{item.name} ×{item.quantity}</span>
          </div>
        ))}
      </div>
    </>
  );
};

export default StatusBar;
