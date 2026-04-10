import React, { useEffect, useRef, useState } from 'react';
import { clamp, ChoiceButton, ReadyPanel, ResultPanel, StatRow, pickOne } from './GamesCommon';

const panelStyle = {
  width: '100%',
  maxWidth: 340,
  margin: '0 auto',
  padding: 12,
  borderRadius: 16,
  border: '2px solid var(--farm-border)',
  background: 'var(--farm-surface-alt)',
};

export const CornRaceGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(8);
  const [pos, setPos] = useState(50);
  const [safeRate, setSafeRate] = useState(0);
  const stateRef = useRef(null);
  const loopRef = useRef(null);
  const timerRef = useRef(null);

  const clearAll = () => {
    if (loopRef.current) clearInterval(loopRef.current);
    if (timerRef.current) clearInterval(timerRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = () => {
    clearAll();
    const s = stateRef.current;
    const score = Math.round((s.safeFrames / Math.max(1, s.totalFrames)) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const start = () => {
    stateRef.current = { pos: 50, drift: 0.6, safeFrames: 0, totalFrames: 0, tick: 0 };
    setPos(50);
    setSafeRate(0);
    setTimeLeft(8);
    setPhase('playing');
    loopRef.current = setInterval(() => {
      const s = stateRef.current;
      s.tick += 1;
      if (s.tick % 12 === 0) s.drift = pickOne([-1.3, -0.9, -0.5, 0.5, 0.9, 1.3]);
      s.pos = clamp(s.pos + s.drift, 0, 100);
      s.totalFrames += 1;
      if (s.pos >= 35 && s.pos <= 65) s.safeFrames += 1;
      setPos(s.pos);
      setSafeRate(Math.round((s.safeFrames / s.totalFrames) * 100));
    }, 90);
    timerRef.current = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          finish();
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  const nudge = (dir) => {
    if (phase !== 'playing') return;
    const s = stateRef.current;
    s.pos = clamp(s.pos + dir * 6, 0, 100);
    setPos(s.pos);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🌽 ${t('稳住装满玉米的小车，不要翻车')}`} hint={`8s · ${t('风会把车吹偏')} · ${t('左右调整')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    return <ResultPanel emoji={safeRate >= 80 ? '🚜' : safeRate >= 50 ? '🌽' : '💥'} title={`${t('稳定率')}: ${safeRate}%`} detail={t('小车在安全区内停留越久得分越高')} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${timeLeft}s`} right={`✅ ${safeRate}%`} />
      <div style={{ ...panelStyle }}>
        <div className='farm-gc-bar progress' style={{ height: 30, borderRadius: 16 }}>
          <div style={{ position: 'absolute', inset: '0 35% 0 35%', background: 'rgba(74,124,63,0.12)', borderLeft: '2px dashed var(--farm-leaf)', borderRight: '2px dashed var(--farm-leaf)' }} />
          <div className='farm-gc-bar-marker' style={{ left: `${pos}%`, width: 8, background: 'var(--farm-harvest)' }} />
        </div>
        <div style={{ marginTop: 18, textAlign: 'center', fontSize: 30 }}>🚜</div>
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        <ChoiceButton onClick={() => nudge(-1)}>⬅️</ChoiceButton>
        <ChoiceButton onClick={() => nudge(1)}>➡️</ChoiceButton>
      </div>
    </div>
  );
};

export const PigChaseGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(8);
  const [pigX, setPigX] = useState(50);
  const [netX, setNetX] = useState(50);
  const [catchRate, setCatchRate] = useState(0);
  const stateRef = useRef(null);
  const loopRef = useRef(null);
  const timerRef = useRef(null);

  const clearAll = () => {
    if (loopRef.current) clearInterval(loopRef.current);
    if (timerRef.current) clearInterval(timerRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = () => {
    clearAll();
    const s = stateRef.current;
    const score = Math.round((s.catchFrames / Math.max(1, s.totalFrames)) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const start = () => {
    stateRef.current = { pigX: 50, pigV: 1.5, netX: 50, catchFrames: 0, totalFrames: 0, tick: 0 };
    setPigX(50);
    setNetX(50);
    setCatchRate(0);
    setTimeLeft(8);
    setPhase('playing');
    loopRef.current = setInterval(() => {
      const s = stateRef.current;
      s.tick += 1;
      if (s.tick % 10 === 0) s.pigV += pickOne([-1.8, -1.2, 1.2, 1.8]);
      s.pigV = clamp(s.pigV, -3.2, 3.2);
      s.pigX += s.pigV;
      if (s.pigX <= 5 || s.pigX >= 95) {
        s.pigV = -s.pigV;
        s.pigX = clamp(s.pigX, 5, 95);
      }
      s.totalFrames += 1;
      if (Math.abs(s.pigX - s.netX) <= 10) s.catchFrames += 1;
      setPigX(s.pigX);
      setNetX(s.netX);
      setCatchRate(Math.round((s.catchFrames / s.totalFrames) * 100));
    }, 90);
    timerRef.current = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          finish();
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  const moveNet = (dir) => {
    if (phase !== 'playing') return;
    const s = stateRef.current;
    s.netX = clamp(s.netX + dir * 7, 0, 100);
    setNetX(s.netX);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🐷 ${t('追上乱跑的小猪，让捕网尽量贴住它')}`} hint={`8s · ${t('越接近小猪得分越高')} · ${t('左右追踪')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    return <ResultPanel emoji={catchRate >= 80 ? '🪤' : catchRate >= 50 ? '🐷' : '💨'} title={`${t('贴身率')}: ${catchRate}%`} detail={t('捕网与小猪重合时间越久分越高')} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${timeLeft}s`} right={`🎯 ${catchRate}%`} />
      <div style={{ ...panelStyle, position: 'relative', height: 120 }}>
        <div style={{ position: 'absolute', left: `${pigX}%`, top: 20, transform: 'translateX(-50%)', fontSize: 32 }}>🐷</div>
        <div style={{ position: 'absolute', left: `${netX}%`, top: 70, transform: 'translateX(-50%)', fontSize: 30 }}>🪤</div>
        <div style={{ position: 'absolute', inset: '54px 6% 0 6%', borderTop: '2px dashed var(--farm-border)' }} />
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        <ChoiceButton onClick={() => moveNet(-1)}>⬅️</ChoiceButton>
        <ChoiceButton onClick={() => moveNet(1)}>➡️</ChoiceButton>
      </div>
    </div>
  );
};

export const GrapeGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(8);
  const [juice, setJuice] = useState(0);
  const [wobble, setWobble] = useState(50);
  const stateRef = useRef(null);
  const timerRef = useRef(null);
  const loopRef = useRef(null);

  const clearAll = () => {
    if (timerRef.current) clearInterval(timerRef.current);
    if (loopRef.current) clearInterval(loopRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = () => {
    clearAll();
    const s = stateRef.current;
    const stability = 1 - Math.abs(s.wobble - 50) / 50;
    const score = Math.round(clamp((s.juice / 100) * 0.7 + stability * 0.3, 0, 1) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const start = () => {
    stateRef.current = { juice: 0, wobble: 50, lastSide: '' };
    setJuice(0);
    setWobble(50);
    setTimeLeft(8);
    setPhase('playing');
    loopRef.current = setInterval(() => {
      const s = stateRef.current;
      const recenter = s.wobble > 50 ? -1.2 : 1.2;
      s.wobble = clamp(s.wobble + recenter, 0, 100);
      setWobble(s.wobble);
    }, 140);
    timerRef.current = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          finish();
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  const stomp = (side) => {
    if (phase !== 'playing') return;
    const s = stateRef.current;
    s.juice = clamp(s.juice + (s.lastSide && s.lastSide !== side ? 7 : 3), 0, 100);
    s.wobble = clamp(s.wobble + (side === 'L' ? -8 : 8) + (s.lastSide === side ? (side === 'L' ? -6 : 6) : 0), 0, 100);
    s.lastSide = side;
    setJuice(s.juice);
    setWobble(s.wobble);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🍇 ${t('左右交替踩踏，榨汁越快越好，但别失去平衡')}`} hint={`8s · ${t('连续踩同一边会更容易失衡')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const stability = Math.round((1 - Math.abs(wobble - 50) / 50) * 100);
    const score = Math.round(clamp((juice / 100) * 0.7 + (stability / 100) * 0.3, 0, 1) * 100);
    return <ResultPanel emoji={score >= 80 ? '🍷' : score >= 50 ? '🍇' : '😵'} title={`${t('酿汁得分')}: ${score}`} detail={`${t('榨汁')} ${juice}% · ${t('稳定')} ${stability}%`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${timeLeft}s`} right={`🍷 ${juice}%`} />
      <div style={{ ...panelStyle }}>
        <div className='farm-gc-bar progress' style={{ height: 28, borderRadius: 16, marginBottom: 12 }}>
          <div style={{ position: 'absolute', inset: '0 40% 0 40%', background: 'rgba(74,124,63,0.12)', borderLeft: '2px dashed var(--farm-leaf)', borderRight: '2px dashed var(--farm-leaf)' }} />
          <div className='farm-gc-bar-marker' style={{ left: `${wobble}%`, width: 8, background: '#8a6cb0' }} />
        </div>
        <div className='farm-gc-bar power' style={{ height: 18, borderRadius: 10 }}>
          <div className='farm-gc-bar-fill' style={{ width: `${juice}%` }} />
        </div>
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        <ChoiceButton onClick={() => stomp('L')}>👣 L</ChoiceButton>
        <ChoiceButton onClick={() => stomp('R')}>👣 R</ChoiceButton>
      </div>
    </div>
  );
};
