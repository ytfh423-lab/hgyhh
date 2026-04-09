import React, { useEffect, useMemo } from 'react';

const FarmMedalDropOverlay = ({ drop, onClose, t }) => {
  useEffect(() => {
    if (!drop) return undefined;
    const timer = setTimeout(() => {
      onClose?.();
    }, 3600);
    return () => clearTimeout(timer);
  }, [drop, onClose]);

  const sparkles = useMemo(() => {
    if (!drop) return [];
    const sparkleCount = drop.rarity === 'epic' ? 18 : drop.rarity === 'rare' ? 14 : 10;
    return Array.from({ length: sparkleCount }).map((_, index) => ({
      id: `${drop.key}-${drop.quantity}-${index}`,
      left: `${8 + Math.random() * 84}%`,
      top: `${8 + Math.random() * 72}%`,
      delay: `${Math.random() * 1.2}s`,
      duration: `${1.8 + Math.random() * 1.4}s`,
      size: `${10 + Math.random() * 16}px`,
    }));
  }, [drop]);

  if (!drop) return null;

  const footerText = drop.is_new
    ? t('首次收录，已加入你的勋章墙')
    : `${t('再次掉落')} · ${t('已拥有')} ×${drop.quantity}`;

  return (
    <div
      className={`farm-medal-overlay farm-medal-overlay--${drop.animation || 'bloom'} farm-medal-overlay--${drop.rarity || 'common'}`}
      style={{
        '--farm-medal-from': drop.color_from || '#22c55e',
        '--farm-medal-to': drop.color_to || '#86efac',
        '--farm-medal-glow': drop.glow_color || 'rgba(34, 197, 94, 0.45)',
      }}
      onClick={onClose}
    >
      <div className='farm-medal-card' onClick={(event) => event.stopPropagation()}>
        <div className='farm-medal-backlight' />
        <div className='farm-medal-ring farm-medal-ring-outer' />
        <div className='farm-medal-ring farm-medal-ring-inner' />
        <div className='farm-medal-chip'>
          <span className='farm-medal-chip-emoji'>{drop.emoji}</span>
        </div>
        <div className='farm-medal-kicker'>
          {drop.is_new ? t('勋章掉落') : t('勋章再次闪耀')}
        </div>
        <div className='farm-medal-title'>{drop.name}</div>
        <div className='farm-medal-description'>{drop.description}</div>
        <div className='farm-medal-meta'>
          <span className='farm-medal-pill'>{drop.rarity_label}</span>
          {drop.source_label && <span className='farm-medal-pill'>{drop.source_label}</span>}
          <span className='farm-medal-pill'>×{drop.quantity}</span>
        </div>
        <div className='farm-medal-footer'>{footerText}</div>
        <div className='farm-medal-sparkles'>
          {sparkles.map((sparkle) => (
            <span
              key={sparkle.id}
              className='farm-medal-sparkle'
              style={{
                left: sparkle.left,
                top: sparkle.top,
                animationDelay: sparkle.delay,
                animationDuration: sparkle.duration,
                fontSize: sparkle.size,
              }}
            >
              ✦
            </span>
          ))}
        </div>
      </div>
    </div>
  );
};

export default FarmMedalDropOverlay;
