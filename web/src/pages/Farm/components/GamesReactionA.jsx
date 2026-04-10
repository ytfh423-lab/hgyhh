import React, { useEffect, useRef, useState } from 'react';
import { clamp, pickOne, randInt, ReadyPanel, ResultPanel, StatRow, ChoiceButton, shuffle } from './GamesCommon';

const panelStyle = {
  width: '100%',
  maxWidth: 340,
  margin: '0 auto',
  padding: 12,
  borderRadius: 16,
  border: '2px solid var(--farm-border)',
  background: 'var(--farm-surface-alt)',
};

export const EggHuntGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [wave, setWave] = useState(1);
  const [cells, setCells] = useState([]);
  const [revealing, setRevealing] = useState(false);
  const [found, setFound] = useState(0);
  const [mistakes, setMistakes] = useState(0);
  const [timeLeft, setTimeLeft] = useState(5);
  const scoresRef = useRef([]);
  const eggsRef = useRef([]);
  const foundRef = useRef(0);
  const mistakesRef = useRef(0);
  const waveRef = useRef(1);
  const timerRef = useRef(null);
  const revealRef = useRef(null);
  const totalWaves = 3;

  const clearAll = () => {
    if (timerRef.current) clearInterval(timerRef.current);
    if (revealRef.current) clearTimeout(revealRef.current);
  };

  useEffect(() => clearAll, []);

  const finishGame = (scores) => {
    const avg = Math.round(scores.reduce((sum, val) => sum + val, 0) / scores.length);
    setPhase('done');
    onComplete(avg / 100, avg);
  };

  const finishWave = () => {
    clearAll();
    const eggCount = eggsRef.current.length;
    const waveScore = Math.round(clamp((foundRef.current / eggCount) - mistakesRef.current * 0.08, 0, 1) * 100);
    const nextScores = [...scoresRef.current, waveScore];
    scoresRef.current = nextScores;
    if (waveRef.current >= totalWaves) {
      setTimeout(() => finishGame(nextScores), 320);
      return;
    }
    setTimeout(() => beginWave(waveRef.current + 1), 420);
  };

  const beginWave = (nextWave) => {
    clearAll();
    waveRef.current = nextWave;
    const eggSlots = shuffle(Array.from({ length: 9 }, (_, index) => index)).slice(0, 2 + nextWave);
    eggsRef.current = eggSlots;
    foundRef.current = 0;
    mistakesRef.current = 0;
    setWave(nextWave);
    setFound(0);
    setMistakes(0);
    setTimeLeft(5);
    setRevealing(true);
    setCells(Array.from({ length: 9 }, (_, index) => ({ id: index, open: false, egg: eggSlots.includes(index) })));
    setPhase('playing');
    revealRef.current = setTimeout(() => {
      setRevealing(false);
      timerRef.current = setInterval(() => {
        setTimeLeft((prev) => {
          if (prev <= 1) {
            finishWave();
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    }, 1200);
  };

  const start = () => {
    scoresRef.current = [];
    beginWave(1);
  };

  const hitCell = (id) => {
    if (phase !== 'playing' || revealing) return;
    const isEgg = eggsRef.current.includes(id);
    setCells((prev) => prev.map((cell) => (cell.id === id ? { ...cell, open: true } : cell)));
    if (!isEgg) {
      mistakesRef.current += 1;
      setMistakes(mistakesRef.current);
      return;
    }
    const alreadyOpen = cells.find((cell) => cell.id === id)?.open;
    if (alreadyOpen) return;
    foundRef.current += 1;
    setFound(foundRef.current);
    if (foundRef.current >= eggsRef.current.length) finishWave();
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🥚 ${t('先记住鸡蛋位置，再把它们找出来')}`} hint={`3 ${t('轮')} · ${t('记忆')} + ${t('点击')} · ${t('失误扣分')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const avg = Math.round(scoresRef.current.reduce((sum, val) => sum + val, 0) / scoresRef.current.length);
    return <ResultPanel emoji={avg >= 80 ? '🐣' : avg >= 50 ? '🥚' : '😵'} title={`${t('记忆得分')}: ${avg}`} detail={`${t('完成')} ${scoresRef.current.length}/${totalWaves} ${t('轮')}`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${t('第')} ${wave}/${totalWaves} ${t('轮')}`} right={revealing ? `👀 ${t('观察中')}` : `${timeLeft}s · ✅ ${found}/${eggsRef.current.length}`} />
      <div style={{ ...panelStyle, display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
        {cells.map((cell) => (
          <button
            key={cell.id}
            type='button'
            onClick={() => hitCell(cell.id)}
            style={{
              aspectRatio: '1/1',
              borderRadius: 16,
              border: `2px solid ${(cell.open || revealing) && cell.egg ? 'var(--farm-harvest)' : 'var(--farm-border)'}`,
              background: (cell.open || revealing) && cell.egg ? 'rgba(200,146,42,0.14)' : 'var(--farm-surface)',
              fontSize: 28,
              cursor: revealing ? 'default' : 'pointer',
            }}
          >
            {(cell.open || revealing) && cell.egg ? '🥚' : '🌾'}
          </button>
        ))}
      </div>
      <div style={{ fontSize: 12, color: 'var(--farm-text-3)' }}>{revealing ? t('记住鸡蛋位置') : `${t('失误')}: ${mistakes}`}</div>
    </div>
  );
};

export const BugCatchGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(8);
  const [bugs, setBugs] = useState([]);
  const [stats, setStats] = useState({ hits: 0, bad: 0, total: 0 });
  const bugsRef = useRef([]);
  const statsRef = useRef({ hits: 0, bad: 0, total: 0 });
  const moveRef = useRef(null);
  const spawnRef = useRef(null);
  const timerRef = useRef(null);
  const idRef = useRef(1);

  const clearAll = () => {
    if (moveRef.current) clearInterval(moveRef.current);
    if (spawnRef.current) clearInterval(spawnRef.current);
    if (timerRef.current) clearInterval(timerRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = () => {
    clearAll();
    const s = statsRef.current;
    const ratio = s.total > 0 ? s.hits / s.total : 0;
    const score = Math.round(clamp(ratio - s.bad * 0.07, 0, 1) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const start = () => {
    bugsRef.current = [];
    statsRef.current = { hits: 0, bad: 0, total: 0 };
    setStats(statsRef.current);
    setBugs([]);
    setTimeLeft(8);
    setPhase('playing');
    spawnRef.current = setInterval(() => {
      const hostile = Math.random() < 0.72;
      if (hostile) statsRef.current.total += 1;
      bugsRef.current = [...bugsRef.current, {
        id: idRef.current++,
        x: randInt(20, 300),
        y: randInt(24, 180),
        dx: (Math.random() < 0.5 ? -1 : 1) * (0.8 + Math.random() * 1.4),
        dy: (Math.random() < 0.5 ? -1 : 1) * (0.6 + Math.random() * 1.2),
        ttl: 34,
        hostile,
        emoji: hostile ? pickOne(['🐛', '🪲', '🐜']) : pickOne(['🐝', '🦋']),
      }];
      setStats({ ...statsRef.current });
    }, 320);
    moveRef.current = setInterval(() => {
      bugsRef.current = bugsRef.current
        .map((bug) => ({
          ...bug,
          x: clamp(bug.x + bug.dx, 18, 302),
          y: clamp(bug.y + bug.dy, 18, 182),
          dx: bug.x <= 18 || bug.x >= 302 ? -bug.dx : bug.dx,
          dy: bug.y <= 18 || bug.y >= 182 ? -bug.dy : bug.dy,
          ttl: bug.ttl - 1,
        }))
        .filter((bug) => bug.ttl > 0);
      setBugs([...bugsRef.current]);
    }, 80);
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

  const tapBug = (id) => {
    if (phase !== 'playing') return;
    const target = bugsRef.current.find((bug) => bug.id === id);
    if (!target) return;
    if (target.hostile) statsRef.current.hits += 1;
    else statsRef.current.bad += 1;
    bugsRef.current = bugsRef.current.filter((bug) => bug.id !== id);
    setStats({ ...statsRef.current });
    setBugs([...bugsRef.current]);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🐛 ${t('只抓害虫，不要误点蜜蜂和蝴蝶')}`} hint={`8s · ${t('高速反应')} · ${t('误抓扣分')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const ratio = stats.total > 0 ? Math.round((stats.hits / stats.total) * 100) : 0;
    const score = Math.round(clamp((stats.total > 0 ? stats.hits / stats.total : 0) - stats.bad * 0.07, 0, 1) * 100);
    return <ResultPanel emoji={score >= 80 ? '🧹' : score >= 50 ? '🐛' : '😵'} title={`${t('捕虫得分')}: ${score}`} detail={`${t('命中')} ${stats.hits}/${stats.total} · ${t('误抓')} ${stats.bad} · ${t('准确率')} ${ratio}%`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${timeLeft}s`} right={`✅ ${stats.hits}/${stats.total} · ⚠️ ${stats.bad}`} />
      <div style={{ ...panelStyle, position: 'relative', height: 200, overflow: 'hidden' }}>
        {bugs.map((bug) => (
          <button
            key={bug.id}
            type='button'
            onClick={() => tapBug(bug.id)}
            style={{
              position: 'absolute',
              left: bug.x,
              top: bug.y,
              width: 34,
              height: 34,
              marginLeft: -17,
              marginTop: -17,
              borderRadius: '50%',
              border: `2px solid ${bug.hostile ? 'var(--farm-danger)' : 'var(--farm-sky)'}`,
              background: bug.hostile ? 'rgba(184,66,51,0.14)' : 'rgba(90,143,180,0.14)',
              fontSize: 18,
              cursor: 'pointer',
            }}
          >
            {bug.emoji}
          </button>
        ))}
      </div>
    </div>
  );
};

export const DuckHerdGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [lane, setLane] = useState(1);
  const [ducks, setDucks] = useState([]);
  const [stats, setStats] = useState({ saved: 0, escaped: 0, total: 12 });
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
    const score = Math.round((s.saved / s.total) * 100);
    setPhase('done');
    onComplete(score / 100, s.saved);
  };

  const start = () => {
    stateRef.current = { lane: 1, ducks: [], saved: 0, escaped: 0, total: 12, spawned: 0, tick: 0 };
    setLane(1);
    setDucks([]);
    setStats({ saved: 0, escaped: 0, total: 12 });
    setPhase('playing');
    loopRef.current = setInterval(() => {
      const s = stateRef.current;
      s.tick += 1;
      if (s.spawned < s.total && s.tick % 6 === 0) {
        s.ducks.push({ id: idRef.current++, lane: randInt(0, 2), x: 6, speed: 2 + Math.random() * 1.3 });
        s.spawned += 1;
      }
      s.ducks = s.ducks.reduce((list, duck) => {
        const next = { ...duck, x: duck.x + duck.speed };
        if (next.x >= 94) {
          if (next.lane === s.lane) s.saved += 1;
          else s.escaped += 1;
          return list;
        }
        list.push(next);
        return list;
      }, []);
      setDucks([...s.ducks]);
      setStats({ saved: s.saved, escaped: s.escaped, total: s.total });
      if (s.spawned >= s.total && s.ducks.length === 0) finish();
    }, 80);
  };

  const pickLane = (nextLane) => {
    setLane(nextLane);
    if (stateRef.current) stateRef.current.lane = nextLane;
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🦆 ${t('切换池塘入口，把鸭子赶进正确通道')}`} hint={`${t('只能开一个入口')} · 12 ${t('只鸭子')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const score = Math.round((stats.saved / stats.total) * 100);
    return <ResultPanel emoji={score >= 80 ? '🏞️' : score >= 50 ? '🦆' : '💦'} title={`${t('入池')} ${stats.saved}/${stats.total}`} detail={`${t('逃走')} ${stats.escaped} · ${t('得分')} ${score}`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`✅ ${stats.saved}`} right={`💨 ${stats.escaped}/${stats.total}`} />
      <div style={{ ...panelStyle, display: 'flex', flexDirection: 'column', gap: 8 }}>
        {[0, 1, 2].map((row) => (
          <div key={row} style={{ position: 'relative', height: 54, borderRadius: 14, background: row === lane ? 'rgba(90,143,180,0.12)' : 'var(--farm-surface)' }}>
            <div style={{ position: 'absolute', left: 10, top: 14, fontSize: 22 }}>🌾</div>
            <div style={{ position: 'absolute', right: 10, top: 10, fontSize: 28 }}>{row === lane ? '💧' : '🚧'}</div>
            {ducks.filter((duck) => duck.lane === row).map((duck) => (
              <div key={duck.id} style={{ position: 'absolute', left: `${duck.x}%`, top: 10, transform: 'translateX(-50%)', fontSize: 28 }}>🦆</div>
            ))}
          </div>
        ))}
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        {[0, 1, 2].map((row) => (
          <ChoiceButton key={row} active={lane === row} onClick={() => pickLane(row)}>{t('通道')} {row + 1}</ChoiceButton>
        ))}
      </div>
    </div>
  );
};
