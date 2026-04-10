import React from 'react';
import { Button } from '@douyinfe/semi-ui';

export const clamp = (value, min = 0, max = 1) => Math.max(min, Math.min(max, value));

export const randInt = (min, max) => Math.floor(Math.random() * (max - min + 1)) + min;

export const pickOne = (items) => items[Math.floor(Math.random() * items.length)];

export const shuffle = (items) => {
  const arr = [...items];
  for (let i = arr.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [arr[i], arr[j]] = [arr[j], arr[i]];
  }
  return arr;
};

export const ReadyPanel = ({ game, desc, hint, onStart, t }) => (
  <div className='farm-gc-ready'>
    <div className='farm-gc-ready-emoji'>{game.emoji}</div>
    <div className='farm-gc-ready-desc'>{desc}</div>
    <div className='farm-gc-ready-hint'>{hint}</div>
    <Button theme='solid' size='large' onClick={onStart} className='farm-btn' style={{ fontWeight: 700, minWidth: 140 }}>
      ▶ {t('开始')}
    </Button>
  </div>
);

export const ResultPanel = ({ emoji, title, detail, extra }) => (
  <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
    <div className='farm-gc-score-big' style={{ marginBottom: 8 }}>{emoji}</div>
    <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 4 }}>{title}</div>
    {detail && <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>{detail}</div>}
    {extra && <div style={{ marginTop: 10 }}>{extra}</div>}
  </div>
);

export const StatRow = ({ left, right, maxWidth = 340 }) => (
  <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%', maxWidth }}>
    <span className='farm-gc-countdown' style={{ fontSize: 22 }}>{left}</span>
    <span style={{ fontSize: 14, fontWeight: 700 }}>{right}</span>
  </div>
);

export const Pill = ({ children, tone = 'blue' }) => (
  <span className={`farm-pill farm-pill-${tone}`} style={{ whiteSpace: 'nowrap' }}>{children}</span>
);

export const ChoiceButton = ({ active, onClick, children, danger = false, disabled = false, style = {} }) => (
  <button
    type='button'
    onClick={onClick}
    disabled={disabled}
    style={{
      minWidth: 54,
      minHeight: 42,
      borderRadius: 12,
      border: `2px solid ${danger ? 'var(--farm-danger)' : active ? 'var(--farm-leaf)' : 'var(--farm-border)'}`,
      background: danger ? 'rgba(184,66,51,0.1)' : active ? 'rgba(74,124,63,0.12)' : 'var(--farm-surface-alt)',
      color: 'var(--farm-text-0)',
      fontWeight: 700,
      padding: '8px 12px',
      cursor: disabled ? 'not-allowed' : 'pointer',
      opacity: disabled ? 0.5 : 1,
      ...style,
    }}
  >
    {children}
  </button>
);
