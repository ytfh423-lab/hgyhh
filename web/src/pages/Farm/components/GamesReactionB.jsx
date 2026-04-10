import React, { useEffect, useRef, useState } from 'react';
import { clamp, ReadyPanel, ResultPanel, StatRow, ChoiceButton, shuffle } from './GamesCommon';

const panelStyle = {
  width: '100%',
  maxWidth: 340,
  margin: '0 auto',
  padding: 12,
  borderRadius: 16,
  border: '2px solid var(--farm-border)',
  background: 'var(--farm-surface-alt)',
};

export const FruitPickGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(10);
  const [fruits, setFruits] = useState([]);
  const [stats, setStats] = useState({ ripeTotal: 0, harvested: 0, bad: 0 });
  const timerRef = useRef(null);
  const tickRef = useRef(null);
  const stateRef = useRef(null);

  const clearAll = () => {
    if (timerRef.current) clearInterval(timerRef.current);
    if (tickRef.current) clearInterval(tickRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = () => {
    clearAll();
    const s = stateRef.current;
    const ratio = s.ripeTotal > 0 ? s.harvested / s.ripeTotal : 0;
    const score = Math.round(clamp(ratio - s.bad * 0.06, 0, 1) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const start = () => {
    const initial = Array.from({ length: 9 }, (_, index) => ({ id: index, stage: 0, flash: false }));
    stateRef.current = { fruits: initial, ripeTotal: 0, harvested: 0, bad: 0 };
    setFruits(initial);
    setStats({ ripeTotal: 0, harvested: 0, bad: 0 });
    setTimeLeft(10);
    setPhase('playing');
    timerRef.current = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          finish();
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    tickRef.current = setInterval(() => {
      const s = stateRef.current;
      const picks = shuffle(Array.from({ length: s.fruits.length }, (_, index) => index)).slice(0, 3);
      s.fruits = s.fruits.map((fruit, index) => {
        if (!picks.includes(index)) return { ...fruit, flash: false };
        if (fruit.stage === 3) return { ...fruit, stage: 0, flash: false };
        const nextStage = fruit.stage + 1;
        if (nextStage === 2) s.ripeTotal += 1;
        return { ...fruit, stage: nextStage, flash: nextStage === 2 };
      });
      setStats({ ripeTotal: s.ripeTotal, harvested: s.harvested, bad: s.bad });
      setFruits([...s.fruits]);
    }, 850);
  };

  const pickFruit = (id) => {
    if (phase !== 'playing') return;
    const s = stateRef.current;
    s.fruits = s.fruits.map((fruit) => {
      if (fruit.id !== id) return fruit;
      if (fruit.stage === 2) {
        s.harvested += 1;
        return { ...fruit, stage: 0, flash: false };
      }
      s.bad += 1;
      return { ...fruit, flash: true };
    });
    setStats({ ripeTotal: s.ripeTotal, harvested: s.harvested, bad: s.bad });
    setFruits([...s.fruits]);
  };

  const iconForStage = (stage) => {
    if (stage === 0) return '🌿';
    if (stage === 1) return '🟡';
    if (stage === 2) return '🍎';
    return '🪰';
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🍎 ${t('只摘成熟果子，青果和烂果都会扣分')}`} hint={`10s · ${t('观察成熟时机')} · ${t('果子会继续变化')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const ratio = stats.ripeTotal > 0 ? Math.round((stats.harvested / stats.ripeTotal) * 100) : 0;
    const score = Math.round(clamp((stats.ripeTotal > 0 ? stats.harvested / stats.ripeTotal : 0) - stats.bad * 0.06, 0, 1) * 100);
    return <ResultPanel emoji={score >= 80 ? '🧺' : score >= 50 ? '🍎' : '😵'} title={`${t('采摘得分')}: ${score}`} detail={`${t('摘对')} ${stats.harvested}/${stats.ripeTotal} · ${t('误摘')} ${stats.bad} · ${t('成熟命中率')} ${ratio}%`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${timeLeft}s`} right={`✅ ${stats.harvested}/${stats.ripeTotal || 0} · ⚠️ ${stats.bad}`} />
      <div style={{ ...panelStyle, display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
        {fruits.map((fruit) => (
          <button
            key={fruit.id}
            type='button'
            onClick={() => pickFruit(fruit.id)}
            style={{
              aspectRatio: '1/1',
              borderRadius: 16,
              border: `2px solid ${fruit.stage === 2 ? 'var(--farm-harvest)' : fruit.stage === 3 ? 'var(--farm-danger)' : 'var(--farm-border)'}`,
              background: fruit.stage === 2 ? 'rgba(200,146,42,0.14)' : fruit.stage === 3 ? 'rgba(184,66,51,0.1)' : 'var(--farm-surface)',
              fontSize: 28,
              cursor: 'pointer',
            }}
          >
            {iconForStage(fruit.stage)}
          </button>
        ))}
      </div>
      <div style={{ fontSize: 12, color: 'var(--farm-text-3)' }}>{t('只在出现红苹果时点击')}</div>
    </div>
  );
};

export const FoxHuntGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [lane, setLane] = useState(0);
  const [foxes, setFoxes] = useState([]);
  const [stats, setStats] = useState({ blocked: 0, lost: 0, total: 10 });
  const [cooldownLane, setCooldownLane] = useState(null);
  const stateRef = useRef(null);
  const loopRef = useRef(null);
  const idRef = useRef(1);

  const clearLoop = () => {
    if (loopRef.current) clearInterval(loopRef.current);
  };

  useEffect(() => clearLoop, []);

  const finish = () => {
    clearLoop();
    const s = stateRef.current;
    const score = Math.round((s.blocked / s.total) * 100);
    setPhase('done');
    onComplete(score / 100, s.blocked);
  };

  const start = () => {
    stateRef.current = { foxes: [], blocked: 0, lost: 0, total: 10, spawned: 0, tick: 0 };
    setLane(0);
    setCooldownLane(null);
    setFoxes([]);
    setStats({ blocked: 0, lost: 0, total: 10 });
    setPhase('playing');
    loopRef.current = setInterval(() => {
      const s = stateRef.current;
      s.tick += 1;
      if (s.spawned < s.total && s.tick % 10 === 0) {
        s.foxes.push({ id: idRef.current++, lane: Math.floor(Math.random() * 4), x: 0, speed: 6 + Math.random() * 2 });
        s.spawned += 1;
      }
      s.foxes = s.foxes.reduce((list, fox) => {
        const next = { ...fox, x: fox.x + fox.speed };
        if (next.x >= 80) {
          s.lost += 1;
          return list;
        }
        list.push(next);
        return list;
      }, []);
      setFoxes([...s.foxes]);
      setStats({ blocked: s.blocked, lost: s.lost, total: s.total });
      if (s.spawned >= s.total && s.foxes.length === 0) finish();
    }, 80);
  };

  const guardLane = (nextLane) => {
    if (phase !== 'playing') return;
    setLane(nextLane);
    setCooldownLane(nextLane);
    setTimeout(() => setCooldownLane(null), 220);
    const s = stateRef.current;
    let blocked = 0;
    s.foxes = s.foxes.filter((fox) => {
      const inRange = fox.x >= 55 && fox.x <= 78;
      if (fox.lane === nextLane && inRange) {
        blocked += 1;
        return false;
      }
      return true;
    });
    if (blocked > 0) s.blocked += blocked;
    setFoxes([...s.foxes]);
    setStats({ blocked: s.blocked, lost: s.lost, total: s.total });
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🦊 ${t('看准狐狸冲刺的鸡舍，及时关门赶走它')}`} hint={`10 ${t('次进攻')} · ${t('只在接近鸡舍时拦截有效')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const score = Math.round((stats.blocked / stats.total) * 100);
    return <ResultPanel emoji={score >= 80 ? '🛡️' : score >= 50 ? '🦊' : '🐔'} title={`${t('成功拦截')} ${stats.blocked}/${stats.total}`} detail={`${t('失守')} ${stats.lost} · ${t('得分')} ${score}`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`🛡️ ${stats.blocked}`} right={`⚠️ ${stats.lost}/${stats.total}`} />
      <div style={{ ...panelStyle, display: 'flex', flexDirection: 'column', gap: 8 }}>
        {Array.from({ length: 4 }, (_, row) => (
          <div key={row} style={{ position: 'relative', height: 48, borderRadius: 14, background: row === lane ? 'rgba(184,66,51,0.08)' : 'var(--farm-surface)' }}>
            <div style={{ position: 'absolute', left: 8, top: 8, fontSize: 24 }}>🌾</div>
            <div style={{ position: 'absolute', right: 8, top: 6, fontSize: 28 }}>{cooldownLane === row ? '🚪' : '🐔'}</div>
            {foxes.filter((fox) => fox.lane === row).map((fox) => (
              <div key={fox.id} style={{ position: 'absolute', left: `${fox.x}%`, top: 8, transform: 'translateX(-50%)', fontSize: 26 }}>🦊</div>
            ))}
          </div>
        ))}
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 8, width: '100%', maxWidth: 320 }}>
        {Array.from({ length: 4 }, (_, row) => (
          <ChoiceButton key={row} active={lane === row} onClick={() => guardLane(row)}>{t('鸡舍')} {row + 1}</ChoiceButton>
        ))}
      </div>
    </div>
  );
};
