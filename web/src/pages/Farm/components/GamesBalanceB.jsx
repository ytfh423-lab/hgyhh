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

export const BeekeepGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(9);
  const [hives, setHives] = useState([50, 50, 50]);
  const [stableRate, setStableRate] = useState(0);
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
    const score = Math.round((s.stableFrames / Math.max(1, s.totalFrames)) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const start = () => {
    stateRef.current = { hives: [50, 50, 50], stableFrames: 0, totalFrames: 0 };
    setHives([50, 50, 50]);
    setStableRate(0);
    setTimeLeft(9);
    setPhase('playing');
    loopRef.current = setInterval(() => {
      const s = stateRef.current;
      s.hives = s.hives.map((value, index) => clamp(value + (index === 1 ? 1.6 : 1.2) + Math.random() * 1.2, 0, 100));
      s.totalFrames += 1;
      if (s.hives.every((value) => value >= 30 && value <= 70)) s.stableFrames += 1;
      setHives([...s.hives]);
      setStableRate(Math.round((s.stableFrames / s.totalFrames) * 100));
    }, 180);
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

  const smoke = (index) => {
    if (phase !== 'playing') return;
    const s = stateRef.current;
    s.hives = s.hives.map((value, i) => {
      if (i === index) return clamp(value - 20, 0, 100);
      return clamp(value + 4, 0, 100);
    });
    setHives([...s.hives]);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🐝 ${t('轮流给蜂箱熏烟，保持三个蜂箱都在安全区')}`} hint={`9s · ${t('一次操作会让其他蜂箱压力上升')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    return <ResultPanel emoji={stableRate >= 80 ? '🍯' : stableRate >= 50 ? '🐝' : '😖'} title={`${t('稳态率')}: ${stableRate}%`} detail={t('三个蜂箱同时处于安全区的时间越久越高分')} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${timeLeft}s`} right={`✅ ${stableRate}%`} />
      <div style={{ ...panelStyle, display: 'flex', flexDirection: 'column', gap: 10 }}>
        {hives.map((value, index) => (
          <button key={index} type='button' onClick={() => smoke(index)} style={{ border: 'none', background: 'transparent', cursor: 'pointer' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ width: 32, fontSize: 22 }}>🐝</span>
              <div className='farm-gc-bar progress' style={{ flex: 1, height: 22, borderRadius: 12 }}>
                <div style={{ position: 'absolute', inset: '0 30% 0 30%', background: 'rgba(74,124,63,0.12)' }} />
                <div className='farm-gc-bar-fill' style={{ width: `${value}%`, background: value > 70 ? 'linear-gradient(90deg, var(--farm-harvest), var(--farm-danger))' : undefined }} />
              </div>
              <span style={{ width: 42, fontWeight: 700 }}>{Math.round(value)}</span>
            </div>
          </button>
        ))}
      </div>
      <div style={{ fontSize: 12, color: 'var(--farm-text-3)' }}>{t('点击某个蜂箱给它降压')}</div>
    </div>
  );
};

export const HatchEggGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(10);
  const [temp, setTemp] = useState(50);
  const [humidity, setHumidity] = useState(50);
  const [stableRate, setStableRate] = useState(0);
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
    const score = Math.round((s.goodFrames / Math.max(1, s.totalFrames)) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const start = () => {
    stateRef.current = { temp: 50, humidity: 50, goodFrames: 0, totalFrames: 0, wind: 0.8 };
    setTemp(50);
    setHumidity(50);
    setStableRate(0);
    setTimeLeft(10);
    setPhase('playing');
    loopRef.current = setInterval(() => {
      const s = stateRef.current;
      s.wind = clamp(s.wind + pickOne([-0.4, 0, 0.4]), -1.6, 1.6);
      s.temp = clamp(s.temp + s.wind + 0.4, 0, 100);
      s.humidity = clamp(s.humidity - 0.5 + Math.random() * 1.2, 0, 100);
      s.totalFrames += 1;
      if (s.temp >= 45 && s.temp <= 55 && s.humidity >= 45 && s.humidity <= 60) s.goodFrames += 1;
      setTemp(s.temp);
      setHumidity(s.humidity);
      setStableRate(Math.round((s.goodFrames / s.totalFrames) * 100));
    }, 180);
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

  const adjust = (key) => {
    if (phase !== 'playing') return;
    const s = stateRef.current;
    if (key === 'warm') s.temp = clamp(s.temp + 8, 0, 100);
    if (key === 'cool') s.temp = clamp(s.temp - 8, 0, 100);
    if (key === 'mist') s.humidity = clamp(s.humidity + 9, 0, 100);
    if (key === 'vent') s.humidity = clamp(s.humidity - 9, 0, 100);
    setTemp(s.temp);
    setHumidity(s.humidity);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🐣 ${t('同时把温度和湿度维持在最佳孵化区间')}`} hint={`10s · ${t('双参数控制')} · ${t('越稳越高分')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    return <ResultPanel emoji={stableRate >= 80 ? '🐥' : stableRate >= 50 ? '🥚' : '🥶'} title={`${t('孵化稳态')}: ${stableRate}%`} detail={`${t('温度与湿度同时在目标范围内的时间越久越高分')}`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${timeLeft}s`} right={`✅ ${stableRate}%`} />
      <div style={{ ...panelStyle, display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div>
          <div style={{ fontSize: 12, marginBottom: 6 }}>{t('温度')}</div>
          <div className='farm-gc-bar progress' style={{ height: 22, borderRadius: 12 }}>
            <div style={{ position: 'absolute', inset: '0 45% 0 55%', background: 'rgba(74,124,63,0.12)' }} />
            <div style={{ position: 'absolute', inset: '0 45% 0 45%', background: 'rgba(74,124,63,0.08)' }} />
            <div className='farm-gc-bar-fill' style={{ width: `${temp}%` }} />
          </div>
        </div>
        <div>
          <div style={{ fontSize: 12, marginBottom: 6 }}>{t('湿度')}</div>
          <div className='farm-gc-bar progress' style={{ height: 22, borderRadius: 12 }}>
            <div style={{ position: 'absolute', inset: '0 40% 0 55%', background: 'rgba(74,124,63,0.12)' }} />
            <div className='farm-gc-bar-fill' style={{ width: `${humidity}%`, background: 'linear-gradient(90deg, #7cc4ff, #5a8fb4)' }} />
          </div>
        </div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 8, width: '100%', maxWidth: 320 }}>
        <ChoiceButton onClick={() => adjust('warm')}>🔥 {t('升温')}</ChoiceButton>
        <ChoiceButton onClick={() => adjust('cool')}>🧊 {t('降温')}</ChoiceButton>
        <ChoiceButton onClick={() => adjust('mist')}>💧 {t('加湿')}</ChoiceButton>
        <ChoiceButton onClick={() => adjust('vent')}>🌬️ {t('通风')}</ChoiceButton>
      </div>
    </div>
  );
};
