import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Button } from '@douyinfe/semi-ui';


/* ═══════════════════════════════════════════════════════════════
   2. TugOfWarGame — 拔河 / 劈柴 🪓
   NPC 不断把绳子拉向右边，玩家狂点把它拉回左边安全区
   ═══════════════════════════════════════════════════════════════ */
export const TugOfWarGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [pos, setPos] = useState(50);
  const [timeLeft, setTimeLeft] = useState(10);
  const posRef = useRef(50);
  const npcForceRef = useRef(0.15);
  const timerRef = useRef(null);
  const animRef = useRef(null);
  const samplesRef = useRef([]);

  const start = () => {
    posRef.current = 50;
    npcForceRef.current = 0.15;
    samplesRef.current = [];
    setPos(50);
    setTimeLeft(10);
    setPhase('playing');
  };

  useEffect(() => {
    if (phase !== 'playing') return;
    timerRef.current = setInterval(() => {
      setTimeLeft(prev => {
        if (prev <= 1) {
          clearInterval(timerRef.current);
          cancelAnimationFrame(animRef.current);
          setPhase('done');
          const avg = samplesRef.current.length > 0
            ? samplesRef.current.reduce((a, b) => a + b, 0) / samplesRef.current.length : 50;
          const score = Math.round(Math.max(0, 100 - avg));
          onComplete(score / 100, score);
          return 0;
        }
        npcForceRef.current = 0.15 + (10 - prev) * 0.02;
        return prev - 1;
      });
    }, 1000);

    const loop = () => {
      posRef.current = Math.min(100, posRef.current + npcForceRef.current);
      if (posRef.current >= 100) {
        clearInterval(timerRef.current);
        setPhase('done');
        onComplete(0, 0);
        return;
      }
      samplesRef.current.push(posRef.current);
      setPos(posRef.current);
      animRef.current = requestAnimationFrame(loop);
    };
    animRef.current = requestAnimationFrame(loop);

    return () => {
      clearInterval(timerRef.current);
      cancelAnimationFrame(animRef.current);
    };
  }, [phase]);

  useEffect(() => {
    const handler = (e) => {
      if (e.code === 'Space' || e.code === 'KeyF') {
        e.preventDefault();
        if (phase === 'ready') start();
        else if (phase === 'playing') pull();
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [phase]);

  const pull = () => {
    if (phase !== 'playing') return;
    posRef.current = Math.max(0, posRef.current - 2.5);
    setPos(posRef.current);
  };

  if (phase === 'ready') {
    return (
      <div className='farm-gc-ready'>
        <div className='farm-gc-ready-emoji'>{game.emoji}</div>
        <div className='farm-gc-ready-desc'>💪 {t('疯狂点击把绳子拉回左侧安全区')}!</div>
        <div className='farm-gc-ready-hint'>10s · {t('NPC力量递增')} · {t('空格/点击')}</div>
        <Button theme='solid' size='large' onClick={start} className='farm-btn'
          style={{ fontWeight: 700, minWidth: 140 }}>▶ {t('开始')}</Button>
      </div>
    );
  }

  if (phase === 'done') {
    const avg = samplesRef.current.length > 0
      ? samplesRef.current.reduce((a, b) => a + b, 0) / samplesRef.current.length : 50;
    const score = Math.round(Math.max(0, 100 - avg));
    const won = score >= 50;
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-gc-score-big' style={{ marginBottom: 8 }}>{won ? '💪' : '😵'}</div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 4 }}>
          {won ? t('拉赢了') + '!' : t('被拉走了') + '!'}
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>{t('得分')}: {score}</div>
      </div>
    );
  }

  const posColor = pos < 35 ? 'var(--farm-leaf)' : pos < 65 ? 'var(--farm-harvest)' : 'var(--farm-danger)';
  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
        <span className='farm-gc-countdown' style={{ color: timeLeft <= 3 ? 'var(--farm-danger)' : 'var(--farm-text-0)', fontSize: 22 }}>
          {timeLeft}s
        </span>
        <span style={{ fontSize: 14, fontWeight: 700, color: posColor }}>
          {pos < 35 ? '✅ ' + t('安全') : pos < 65 ? '⚠️ ' + t('危险') : '🚨 ' + t('快拉')}
        </span>
      </div>

      <div className='farm-gc-rope'>
        <div className='farm-gc-rope-track'>
          <div className='farm-gc-rope-zone' style={{ left: 0, width: '35%' }} />
          <div className='farm-gc-rope-knot' style={{ left: `${pos}%` }}>🪢</div>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 8, fontSize: 22 }}>
        <span>💪 {t('你')}</span>
        <span style={{ flex: 1, textAlign: 'center', fontSize: 14, color: 'var(--farm-text-3)' }}>← {t('拉回来')} →</span>
        <span>👹 NPC</span>
      </div>

      <Button theme='solid' size='large' className='farm-btn'
        style={{ fontWeight: 700, minWidth: 160, fontSize: 16 }}
        onMouseDown={pull} onTouchStart={(e) => { e.preventDefault(); pull(); }}>
        💥 {t('拉')}!
      </Button>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   3. ClickBlitzGame — 除草 / 耕地 🌿
   5 秒内点掉所有 20 个随机冒出的杂草目标
   ═══════════════════════════════════════════════════════════════ */
export const ClickBlitzGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [targets, setTargets] = useState([]);
  const [cleared, setCleared] = useState(0);
  const [timeLeft, setTimeLeft] = useState(5);
  const totalTargets = 20;
  const timerRef = useRef(null);
  const waveRef = useRef(null);
  const clearedRef = useRef(0);

  const emojis = ['🌿', '🌱', '🍂', '☘️', '🌾'];

  const spawnWave = () => {
    const batch = [];
    for (let i = 0; i < 5; i++) {
      batch.push({
        id: Date.now() + i + Math.random(),
        x: 10 + Math.random() * 80,
        y: 10 + Math.random() * 80,
        emoji: emojis[Math.floor(Math.random() * emojis.length)],
        hit: false,
      });
    }
    setTargets(prev => [...prev.filter(t => !t.hit), ...batch]);
  };

  const start = () => {
    clearedRef.current = 0;
    setCleared(0);
    setTargets([]);
    setTimeLeft(5);
    setPhase('playing');
    spawnWave();
    let wave = 1;
    waveRef.current = setInterval(() => {
      wave++;
      if (wave <= 4) spawnWave();
    }, 1200);

    timerRef.current = setInterval(() => {
      setTimeLeft(prev => {
        if (prev <= 1) {
          clearInterval(timerRef.current);
          clearInterval(waveRef.current);
          setPhase('done');
          const score = Math.round((clearedRef.current / totalTargets) * 100);
          onComplete(score / 100, clearedRef.current);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  useEffect(() => {
    return () => {
      clearInterval(timerRef.current);
      clearInterval(waveRef.current);
    };
  }, []);

  useEffect(() => {
    const handler = (e) => {
      if (e.code === 'Space' && phase === 'ready') { e.preventDefault(); start(); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [phase]);

  const hitTarget = (id) => {
    if (phase !== 'playing') return;
    setTargets(prev => prev.map(t => t.id === id ? { ...t, hit: true } : t));
    clearedRef.current++;
    setCleared(clearedRef.current);
    if (clearedRef.current >= totalTargets) {
      clearInterval(timerRef.current);
      clearInterval(waveRef.current);
      setPhase('done');
      onComplete(1, totalTargets);
    }
  };

  if (phase === 'ready') {
    return (
      <div className='farm-gc-ready'>
        <div className='farm-gc-ready-emoji'>{game.emoji}</div>
        <div className='farm-gc-ready-desc'>⚡ {t('5秒内点掉所有杂草')}!</div>
        <div className='farm-gc-ready-hint'>{totalTargets} {t('个目标')} · {t('分批冒出')}</div>
        <Button theme='solid' size='large' onClick={start} className='farm-btn'
          style={{ fontWeight: 700, minWidth: 140 }}>▶ {t('开始')}</Button>
      </div>
    );
  }

  if (phase === 'done') {
    const score = Math.round((cleared / totalTargets) * 100);
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-gc-score-big' style={{ marginBottom: 8 }}>
          {cleared >= totalTargets ? '🏆' : cleared >= 14 ? '👏' : '🤔'}
        </div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 4 }}>
          {cleared}/{totalTargets} {t('已清除')}
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>{t('得分')}: {score}</div>
      </div>
    );
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%', maxWidth: 340 }}>
        <span className='farm-gc-countdown' style={{ color: timeLeft <= 2 ? 'var(--farm-danger)' : 'var(--farm-text-0)', fontSize: 22 }}>
          {timeLeft}s
        </span>
        <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--farm-leaf)' }}>
          ✅ {cleared}/{totalTargets}
        </span>
      </div>
      <div className='farm-gc-target-area'>
        {targets.filter(t => !t.hit).map(tgt => (
          <div key={tgt.id} className='farm-gc-target'
            style={{ left: `${tgt.x}%`, top: `${tgt.y}%` }}
            onClick={() => hitTarget(tgt.id)}
            onTouchStart={(e) => { e.preventDefault(); hitTarget(tgt.id); }}>
            {tgt.emoji}
          </div>
        ))}
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   4. RhythmKeysGame — 挤奶 🐄
   屏幕提示 Q-E-Q-E，必须按对且按快，按错扣分
   ═══════════════════════════════════════════════════════════════ */
export const RhythmKeysGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [seqIdx, setSeqIdx] = useState(0);
  const [score, setScore] = useState(0);
  const [combo, setCombo] = useState(0);
  const [maxCombo, setMaxCombo] = useState(0);
  const [lastKey, setLastKey] = useState(null);
  const [keyState, setKeyState] = useState('');
  const [timeLeft, setTimeLeft] = useState(8);
  const scoreRef = useRef(0);
  const comboRef = useRef(0);
  const seqIdxRef = useRef(0);
  const timerRef = useRef(null);
  const seqLen = 30;
  const [sequence] = useState(() => {
    const seq = [];
    for (let i = 0; i < seqLen; i++) seq.push(Math.random() < 0.5 ? 'Q' : 'E');
    return seq;
  });

  const start = () => {
    scoreRef.current = 0;
    comboRef.current = 0;
    seqIdxRef.current = 0;
    setScore(0);
    setCombo(0);
    setMaxCombo(0);
    setSeqIdx(0);
    setTimeLeft(8);
    setKeyState('');
    setPhase('playing');

    timerRef.current = setInterval(() => {
      setTimeLeft(prev => {
        if (prev <= 1) {
          clearInterval(timerRef.current);
          setPhase('done');
          const s = scoreRef.current;
          onComplete(Math.min(s / (seqLen * 0.7), 1), s);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  useEffect(() => {
    return () => clearInterval(timerRef.current);
  }, []);

  const processKey = (key) => {
    if (phase !== 'playing') return;
    const idx = seqIdxRef.current;
    if (idx >= seqLen) return;
    const expected = sequence[idx];
    setLastKey(key);
    if (key === expected) {
      scoreRef.current++;
      comboRef.current++;
      setScore(scoreRef.current);
      setCombo(comboRef.current);
      setMaxCombo(m => Math.max(m, comboRef.current));
      setKeyState('active');
    } else {
      comboRef.current = 0;
      setCombo(0);
      setKeyState('wrong');
    }
    seqIdxRef.current++;
    setSeqIdx(seqIdxRef.current);

    setTimeout(() => setKeyState(''), 150);

    if (seqIdxRef.current >= seqLen) {
      clearInterval(timerRef.current);
      setPhase('done');
      const s = scoreRef.current;
      onComplete(Math.min(s / (seqLen * 0.7), 1), s);
    }
  };

  useEffect(() => {
    const handler = (e) => {
      if (phase === 'ready' && e.code === 'Space') { e.preventDefault(); start(); return; }
      if (phase !== 'playing') return;
      if (e.code === 'KeyQ' || e.code === 'KeyA') { e.preventDefault(); processKey('Q'); }
      if (e.code === 'KeyE' || e.code === 'KeyD') { e.preventDefault(); processKey('E'); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [phase]);

  if (phase === 'ready') {
    return (
      <div className='farm-gc-ready'>
        <div className='farm-gc-ready-emoji'>{game.emoji}</div>
        <div className='farm-gc-ready-desc'>🎵 {t('按照提示交替按 Q 和 E')}!</div>
        <div className='farm-gc-ready-hint'>{seqLen} {t('次')} · {t('按错扣连击')} · 8s</div>
        <Button theme='solid' size='large' onClick={start} className='farm-btn'
          style={{ fontWeight: 700, minWidth: 140 }}>▶ {t('开始')}</Button>
      </div>
    );
  }

  if (phase === 'done') {
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-gc-score-big' style={{ marginBottom: 8 }}>
          {score >= seqLen * 0.9 ? '🎯' : score >= seqLen * 0.6 ? '👏' : '😅'}
        </div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 4 }}>
          {score}/{seqLen} {t('正确')}
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>
          {t('最高连击')}: {maxCombo} · {t('得分')}: {score}
        </div>
      </div>
    );
  }

  const nextKeys = sequence.slice(seqIdx, seqIdx + 5);
  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
        <span className='farm-gc-countdown' style={{ color: timeLeft <= 2 ? 'var(--farm-danger)' : 'var(--farm-text-0)', fontSize: 22 }}>
          {timeLeft}s
        </span>
        <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--farm-leaf)' }}>
          ✅ {score} · 🔥 {combo}
        </span>
      </div>
      <div className='farm-gc-bar progress' style={{ height: 10, borderRadius: 5 }}>
        <div className='farm-gc-bar-fill' style={{ width: `${(seqIdx / seqLen) * 100}%` }} />
      </div>

      {/* Upcoming keys */}
      <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
        {nextKeys.map((k, i) => (
          <div key={seqIdx + i} style={{
            width: i === 0 ? 56 : 40,
            height: i === 0 ? 56 : 40,
            borderRadius: 10,
            background: i === 0 ? (k === 'Q' ? 'rgba(90,143,180,0.2)' : 'rgba(138,108,176,0.2)') : 'var(--farm-surface-alt)',
            border: i === 0 ? `3px solid ${k === 'Q' ? 'var(--farm-sky)' : '#8a6cb0'}` : '2px solid var(--farm-border)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: i === 0 ? 22 : 16, fontWeight: 800,
            color: i === 0 ? (k === 'Q' ? 'var(--farm-sky)' : '#b094d0') : 'var(--farm-text-3)',
            opacity: 1 - i * 0.15,
            transition: 'all 0.1s',
          }}>
            {k}
          </div>
        ))}
      </div>

      {/* Touch keys */}
      <div className='farm-gc-keys'>
        <div className={`farm-gc-key ${lastKey === 'Q' ? keyState : ''}`}
          onClick={() => processKey('Q')}
          onTouchStart={(e) => { e.preventDefault(); processKey('Q'); }}
          style={{ fontSize: 24 }}>Q</div>
        <div className={`farm-gc-key ${lastKey === 'E' ? keyState : ''}`}
          onClick={() => processKey('E')}
          onTouchStart={(e) => { e.preventDefault(); processKey('E'); }}
          style={{ fontSize: 24 }}>E</div>
      </div>

      {combo >= 5 && (
        <div style={{ fontSize: 16, fontWeight: 800, color: 'var(--farm-harvest)', animation: 'farm-gc-pulse 0.5s infinite' }}>
          🔥 {combo} COMBO!
        </div>
      )}
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   5. CircleDrawGame — 打谷 🌾
   鼠标画圆圈，转得越快脱粒进度越快，限时 8 秒
   ═══════════════════════════════════════════════════════════════ */
export const CircleDrawGame = ({ game, onComplete, t }) => {
  const canvasRef = useRef(null);
  const stateRef = useRef(null);
  const animRef = useRef(null);
  const [phase, setPhase] = useState('ready');
  const [progress, setProgress] = useState(0);
  const [timeLeft, setTimeLeft] = useState(8);
  const timerRef = useRef(null);

  const initState = () => ({
    points: [],
    totalAngle: 0,
    lastAngle: null,
    progress: 0,
    mouseDown: false,
  });

  const start = () => {
    stateRef.current = initState();
    setProgress(0);
    setTimeLeft(8);
    setPhase('playing');

    timerRef.current = setInterval(() => {
      setTimeLeft(prev => {
        if (prev <= 1) {
          clearInterval(timerRef.current);
          setPhase('done');
          const p = stateRef.current ? stateRef.current.progress : 0;
          const score = Math.round(Math.min(p, 100));
          onComplete(score / 100, score);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  useEffect(() => {
    return () => clearInterval(timerRef.current);
  }, []);

  useEffect(() => {
    if (phase !== 'playing') return;
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const W = 300, H = 300;
    canvas.width = W; canvas.height = H;
    const cx = W / 2, cy = H / 2, radius = 100;

    const render = () => {
      const s = stateRef.current;
      if (!s) return;
      ctx.clearRect(0, 0, W, H);

      // Background
      ctx.fillStyle = '#111827';
      ctx.fillRect(0, 0, W, H);

      // Guide circle
      ctx.beginPath();
      ctx.arc(cx, cy, radius, 0, Math.PI * 2);
      ctx.strokeStyle = 'rgba(75, 85, 99, 0.5)';
      ctx.lineWidth = 20;
      ctx.stroke();

      // Progress arc
      const pct = s.progress / 100;
      ctx.beginPath();
      ctx.arc(cx, cy, radius, -Math.PI / 2, -Math.PI / 2 + Math.PI * 2 * pct);
      ctx.strokeStyle = pct >= 1 ? '#6fa85e' : '#8a6cb0';
      ctx.lineWidth = 20;
      ctx.lineCap = 'round';
      ctx.stroke();

      // Trail points
      const trail = s.points.slice(-40);
      if (trail.length > 1) {
        ctx.beginPath();
        ctx.moveTo(trail[0].x, trail[0].y);
        for (let i = 1; i < trail.length; i++) {
          ctx.lineTo(trail[i].x, trail[i].y);
        }
        ctx.strokeStyle = 'rgba(251, 191, 36, 0.6)';
        ctx.lineWidth = 3;
        ctx.lineCap = 'round';
        ctx.stroke();
      }

      // Center text
      ctx.fillStyle = '#fff';
      ctx.font = 'bold 28px sans-serif';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText(Math.round(s.progress) + '%', cx, cy - 8);
      ctx.font = '14px sans-serif';
      ctx.fillStyle = '#9ca3af';
      ctx.fillText('🌾 ' + t('画圆圈'), cx, cy + 18);

      animRef.current = requestAnimationFrame(render);
    };
    animRef.current = requestAnimationFrame(render);

    return () => { if (animRef.current) cancelAnimationFrame(animRef.current); };
  }, [phase]);

  const handleMove = (clientX, clientY) => {
    if (phase !== 'playing' || !stateRef.current) return;
    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    const x = (clientX - rect.left) * (300 / rect.width);
    const y = (clientY - rect.top) * (300 / rect.height);
    const s = stateRef.current;

    s.points.push({ x, y });
    if (s.points.length > 60) s.points.shift();

    const cx = 150, cy = 150;
    const angle = Math.atan2(y - cy, x - cx);
    if (s.lastAngle !== null) {
      let diff = angle - s.lastAngle;
      if (diff > Math.PI) diff -= Math.PI * 2;
      if (diff < -Math.PI) diff += Math.PI * 2;
      s.totalAngle += Math.abs(diff);
      s.progress = Math.min(100, (s.totalAngle / (Math.PI * 12)) * 100);
      setProgress(s.progress);

      if (s.progress >= 100) {
        clearInterval(timerRef.current);
        setPhase('done');
        onComplete(1, 100);
      }
    }
    s.lastAngle = angle;
  };

  useEffect(() => {
    const handler = (e) => {
      if (e.code === 'Space' && phase === 'ready') { e.preventDefault(); start(); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [phase]);

  if (phase === 'ready') {
    return (
      <div className='farm-gc-ready'>
        <div className='farm-gc-ready-emoji'>{game.emoji}</div>
        <div className='farm-gc-ready-desc'>🔄 {t('在圆圈轨道上画圈')} — {t('转得越快进度越高')}!</div>
        <div className='farm-gc-ready-hint'>8s · {t('鼠标/触摸拖动')} · {t('目标')} 100%</div>
        <Button theme='solid' size='large' onClick={start} className='farm-btn'
          style={{ fontWeight: 700, minWidth: 140 }}>▶ {t('开始')}</Button>
      </div>
    );
  }

  if (phase === 'done') {
    const score = Math.round(Math.min(progress, 100));
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-gc-score-big' style={{ marginBottom: 8 }}>
          {score >= 100 ? '🏆' : score >= 60 ? '👏' : '😅'}
        </div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 4 }}>
          {t('脱粒进度')}: {score}%
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>{t('得分')}: {score}</div>
      </div>
    );
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%', maxWidth: 300 }}>
        <span className='farm-gc-countdown' style={{ color: timeLeft <= 2 ? 'var(--farm-danger)' : 'var(--farm-text-0)', fontSize: 22 }}>
          {timeLeft}s
        </span>
        <span style={{ fontSize: 14, fontWeight: 700, color: progress >= 100 ? 'var(--farm-leaf)' : '#8a6cb0' }}>
          {Math.round(progress)}%
        </span>
      </div>
      <div className='farm-gc' style={{ maxWidth: 300, aspectRatio: '1/1', cursor: 'crosshair' }}>
        <canvas ref={canvasRef}
          onMouseMove={(e) => handleMove(e.clientX, e.clientY)}
          onTouchMove={(e) => {
            e.preventDefault();
            const touch = e.touches[0];
            handleMove(touch.clientX, touch.clientY);
          }}
        />
      </div>
      <div style={{ fontSize: 12, color: 'var(--farm-text-3)' }}>{t('沿圆圈快速拖动鼠标/手指')}</div>
    </div>
  );
};
