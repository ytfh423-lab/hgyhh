import React, { useEffect, useMemo, useRef, useState } from 'react';
import { clamp, randInt, ReadyPanel, ResultPanel, StatRow, shuffle } from './GamesCommon';

const panelStyle = {
  width: '100%',
  maxWidth: 340,
  margin: '0 auto',
  padding: 12,
  borderRadius: 16,
  border: '2px solid var(--farm-border)',
  background: 'var(--farm-surface-alt)',
};

export const SheepCountGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [round, setRound] = useState(1);
  const [stage, setStage] = useState('show');
  const [count, setCount] = useState(5);
  const [options, setOptions] = useState([]);
  const [timeLeft, setTimeLeft] = useState(4);
  const [correct, setCorrect] = useState(0);
  const correctRef = useRef(0);
  const answerRef = useRef(null);
  const timerRef = useRef(null);
  const revealRef = useRef(null);
  const totalRounds = 4;

  const clearAll = () => {
    if (timerRef.current) clearInterval(timerRef.current);
    if (revealRef.current) clearTimeout(revealRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = () => {
    clearAll();
    const score = Math.round((correctRef.current / totalRounds) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const nextRound = (next) => {
    clearAll();
    if (next > totalRounds) {
      finish();
      return;
    }
    const answer = randInt(4, 9);
    answerRef.current = answer;
    const opts = shuffle(Array.from(new Set([answer, answer - 1, answer + 1, randInt(3, 10)])).values()).slice(0, 4);
    while (opts.length < 4) {
      const candidate = randInt(3, 10);
      if (!opts.includes(candidate)) opts.push(candidate);
    }
    setRound(next);
    setCount(answer);
    setOptions(shuffle(opts));
    setStage('show');
    setTimeLeft(4);
    revealRef.current = setTimeout(() => {
      setStage('answer');
      timerRef.current = setInterval(() => {
        setTimeLeft((prev) => {
          if (prev <= 1) {
            setTimeout(() => nextRound(next + 1), 250);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    }, 1200);
    setPhase('playing');
  };

  const start = () => {
    correctRef.current = 0;
    setCorrect(0);
    nextRound(1);
  };

  const choose = (value) => {
    if (phase !== 'playing' || stage !== 'answer') return;
    clearAll();
    if (value === answerRef.current) {
      correctRef.current += 1;
      setCorrect(correctRef.current);
    }
    setTimeout(() => nextRound(round + 1), 250);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🐑 ${t('先看一眼羊群，再选出正确数量')}`} hint={`4 ${t('轮')} · ${t('瞬时记数')} · ${t('每轮限时')} 4s`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const score = Math.round((correctRef.current / totalRounds) * 100);
    return <ResultPanel emoji={score >= 75 ? '🌙' : score >= 50 ? '🐑' : '😴'} title={`${t('答对')} ${correctRef.current}/${totalRounds}`} detail={`${t('得分')}: ${score}`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${t('第')} ${round}/${totalRounds} ${t('轮')}`} right={stage === 'show' ? `👀 ${t('看清楚')}` : `${timeLeft}s · ✅ ${correctRef.current}`} />
      <div style={{ ...panelStyle, minHeight: 120, display: 'flex', alignItems: 'center', justifyContent: 'center', flexWrap: 'wrap', gap: 8, fontSize: 28 }}>
        {stage === 'show' ? Array.from({ length: count }, (_, index) => <span key={index}>🐑</span>) : <span style={{ fontSize: 18, fontWeight: 700 }}>{t('刚才有几只羊？')}</span>}
      </div>
      {stage === 'answer' && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 8, width: '100%', maxWidth: 320 }}>
          {options.map((value) => (
            <button key={value} type='button' onClick={() => choose(value)} style={{ minHeight: 48, borderRadius: 14, border: '2px solid var(--farm-border)', background: 'var(--farm-surface)', fontWeight: 700, fontSize: 18, cursor: 'pointer' }}>
              {value}
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export const MushroomGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [round, setRound] = useState(1);
  const [revealing, setRevealing] = useState(false);
  const [path, setPath] = useState([]);
  const [picked, setPicked] = useState([]);
  const [scores, setScores] = useState([]);
  const pathRef = useRef([]);
  const picksRef = useRef([]);
  const revealRef = useRef(null);
  const totalRounds = 3;

  const clearAll = () => {
    if (revealRef.current) clearTimeout(revealRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = (values) => {
    const avg = Math.round(values.reduce((sum, val) => sum + val, 0) / values.length);
    setPhase('done');
    onComplete(avg / 100, avg);
  };

  const beginRound = (next) => {
    clearAll();
    if (next > totalRounds) {
      finish(scores.length ? scores : [0]);
      return;
    }
    const safePath = Array.from({ length: 4 }, () => randInt(0, 2));
    pathRef.current = safePath;
    picksRef.current = [];
    setRound(next);
    setPath(safePath);
    setPicked([]);
    setRevealing(true);
    setPhase('playing');
    revealRef.current = setTimeout(() => setRevealing(false), 1200);
  };

  const start = () => {
    setScores([]);
    beginRound(1);
  };

  const chooseCell = (row, col) => {
    if (phase !== 'playing' || revealing) return;
    if (picked.some((item) => item.row === row)) return;
    const nextPicked = [...picked, { row, col }];
    picksRef.current = nextPicked;
    setPicked(nextPicked);
    if (nextPicked.length >= 4) {
      const correctCount = nextPicked.filter((item) => pathRef.current[item.row] === item.col).length;
      const roundScore = Math.round((correctCount / 4) * 100);
      const nextScores = [...scores, roundScore];
      setScores(nextScores);
      setTimeout(() => {
        if (round >= totalRounds) finish(nextScores);
        else beginRound(round + 1);
      }, 320);
    }
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🍄 ${t('记住安全蘑菇路径，再按行走完森林')}`} hint={`3 ${t('轮')} · ${t('每行只能选一个')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const avg = Math.round((scores.reduce((sum, val) => sum + val, 0) / scores.length) || 0);
    return <ResultPanel emoji={avg >= 80 ? '🌲' : avg >= 50 ? '🍄' : '☠️'} title={`${t('采菇得分')}: ${avg}`} detail={scores.map((score, index) => `R${index + 1}:${score}`).join(' · ')} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${t('第')} ${round}/${totalRounds} ${t('轮')}`} right={revealing ? `👀 ${t('看路径')}` : `${t('已选')} ${picked.length}/4`} />
      <div style={{ ...panelStyle, display: 'flex', flexDirection: 'column', gap: 8 }}>
        {Array.from({ length: 4 }, (_, row) => (
          <div key={row} style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 8 }}>
            {Array.from({ length: 3 }, (_, col) => {
              const isSafe = path[row] === col;
              const chosen = picked.find((item) => item.row === row)?.col === col;
              return (
                <button
                  key={`${row}-${col}`}
                  type='button'
                  onClick={() => chooseCell(row, col)}
                  style={{
                    minHeight: 54,
                    borderRadius: 14,
                    border: `2px solid ${chosen ? 'var(--farm-leaf)' : revealing && isSafe ? 'var(--farm-harvest)' : 'var(--farm-border)'}`,
                    background: chosen ? 'rgba(74,124,63,0.14)' : revealing && isSafe ? 'rgba(200,146,42,0.14)' : 'var(--farm-surface)',
                    fontSize: 22,
                    cursor: 'pointer',
                  }}
                >
                  {chosen ? '👣' : revealing && isSafe ? '🍄' : '🌫️'}
                </button>
              );
            })}
          </div>
        ))}
      </div>
    </div>
  );
};

const buildCrowRound = () => {
  const crowCount = randInt(4, 5);
  return shuffle(Array.from({ length: 9 }, (_, index) => index)).slice(0, crowCount);
};

export const ScarecrowGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [round, setRound] = useState(1);
  const [crows, setCrows] = useState([]);
  const [placed, setPlaced] = useState([]);
  const [scores, setScores] = useState([]);
  const crowsRef = useRef([]);
  const placedRef = useRef([]);
  const totalRounds = 3;

  const coverage = useMemo(() => {
    const covered = crows.filter((cell) => placed.some((p) => Math.floor(p / 3) === Math.floor(cell / 3) || p % 3 === cell % 3));
    return covered.length;
  }, [crows, placed]);

  const finish = (values) => {
    const avg = Math.round(values.reduce((sum, val) => sum + val, 0) / values.length);
    setPhase('done');
    onComplete(avg / 100, avg);
  };

  const beginRound = (next) => {
    if (next > totalRounds) {
      finish(scores.length ? scores : [0]);
      return;
    }
    const nextCrows = buildCrowRound();
    crowsRef.current = nextCrows;
    placedRef.current = [];
    setRound(next);
    setCrows(nextCrows);
    setPlaced([]);
    setPhase('playing');
  };

  const start = () => {
    setScores([]);
    beginRound(1);
  };

  const toggleCell = (cell) => {
    if (phase !== 'playing') return;
    let nextPlaced = placedRef.current.includes(cell)
      ? placedRef.current.filter((item) => item !== cell)
      : [...placedRef.current, cell].slice(-3);
    placedRef.current = nextPlaced;
    setPlaced(nextPlaced);
  };

  const submit = () => {
    if (phase !== 'playing') return;
    const covered = crowsRef.current.filter((cell) => placedRef.current.some((p) => Math.floor(p / 3) === Math.floor(cell / 3) || p % 3 === cell % 3)).length;
    const roundScore = Math.round((covered / crowsRef.current.length) * 100);
    const nextScores = [...scores, roundScore];
    setScores(nextScores);
    setTimeout(() => {
      if (round >= totalRounds) finish(nextScores);
      else beginRound(round + 1);
    }, 280);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`👒 ${t('最多放 3 个稻草人，让它们覆盖尽可能多的乌鸦')}`} hint={`3 ${t('轮')} · ${t('稻草人覆盖同一行和同一列')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const avg = Math.round((scores.reduce((sum, val) => sum + val, 0) / scores.length) || 0);
    return <ResultPanel emoji={avg >= 80 ? '🧑‍🌾' : avg >= 50 ? '👒' : '🐦'} title={`${t('防守得分')}: ${avg}`} detail={scores.map((score, index) => `R${index + 1}:${score}`).join(' · ')} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${t('第')} ${round}/${totalRounds} ${t('轮')}`} right={`${t('覆盖')} ${coverage}/${crows.length} · ${t('已放')} ${placed.length}/3`} />
      <div style={{ ...panelStyle, display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
        {Array.from({ length: 9 }, (_, index) => {
          const hasCrow = crows.includes(index);
          const hasScarecrow = placed.includes(index);
          return (
            <button
              key={index}
              type='button'
              onClick={() => toggleCell(index)}
              style={{
                aspectRatio: '1/1',
                borderRadius: 16,
                border: `2px solid ${hasScarecrow ? 'var(--farm-leaf)' : hasCrow ? 'var(--farm-danger)' : 'var(--farm-border)'}`,
                background: hasScarecrow ? 'rgba(74,124,63,0.14)' : hasCrow ? 'rgba(184,66,51,0.08)' : 'var(--farm-surface)',
                fontSize: 26,
                cursor: 'pointer',
              }}
            >
              {hasScarecrow ? '👒' : hasCrow ? '🐦' : '🌾'}
            </button>
          );
        })}
      </div>
      <button type='button' onClick={submit} style={{ minWidth: 160, minHeight: 44, borderRadius: 14, border: '2px solid var(--farm-border)', background: 'var(--farm-surface)', fontWeight: 700, cursor: 'pointer' }}>
        {t('提交布局')}
      </button>
    </div>
  );
};
