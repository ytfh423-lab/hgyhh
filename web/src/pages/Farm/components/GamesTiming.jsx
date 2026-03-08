import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Button } from '@douyinfe/semi-ui';

/* ═══════════════════════════════════════════════════════════════
   6. FishingBarGame — 钓鱼 🎣
   Flappy-Bird 式反重力条：点击让钩子上升，松开下降
   必须让钩子保持在移动的鱼区域内，坚持 10 秒
   ═══════════════════════════════════════════════════════════════ */
export const FishingBarGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(10);
  const stateRef = useRef(null);
  const animRef = useRef(null);
  const timerRef = useRef(null);
  const [display, setDisplay] = useState({ hookY: 50, zoneY: 40, zoneH: 22, overlap: 0 });

  const BAR_H = 100;
  const HOOK_H = 6;
  const ZONE_H = 22;

  const initState = () => ({
    hookY: 50,
    hookVel: 0,
    zoneY: 40,
    zoneDir: 1,
    zoneSpeed: 0.4,
    pressing: false,
    overlapFrames: 0,
    totalFrames: 0,
  });

  const start = () => {
    stateRef.current = initState();
    setTimeLeft(10);
    setPhase('playing');

    timerRef.current = setInterval(() => {
      setTimeLeft(prev => {
        if (prev <= 1) {
          clearInterval(timerRef.current);
          cancelAnimationFrame(animRef.current);
          setPhase('done');
          const s = stateRef.current;
          const pct = s.totalFrames > 0 ? Math.round((s.overlapFrames / s.totalFrames) * 100) : 0;
          onComplete(pct / 100, pct);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  useEffect(() => {
    return () => {
      clearInterval(timerRef.current);
      if (animRef.current) cancelAnimationFrame(animRef.current);
    };
  }, []);

  useEffect(() => {
    if (phase !== 'playing') return;
    const loop = () => {
      const s = stateRef.current;
      if (!s) return;

      // Hook physics: gravity pulls down, pressing pushes up
      if (s.pressing) {
        s.hookVel -= 0.35;
      } else {
        s.hookVel += 0.2;
      }
      s.hookVel *= 0.92;
      s.hookY += s.hookVel;
      s.hookY = Math.max(0, Math.min(BAR_H - HOOK_H, s.hookY));

      // Zone movement (bouncing)
      s.zoneY += s.zoneDir * s.zoneSpeed;
      if (s.zoneY <= 0) { s.zoneY = 0; s.zoneDir = 1; s.zoneSpeed = 0.3 + Math.random() * 0.3; }
      if (s.zoneY >= BAR_H - ZONE_H) { s.zoneY = BAR_H - ZONE_H; s.zoneDir = -1; s.zoneSpeed = 0.3 + Math.random() * 0.3; }

      // Check overlap
      s.totalFrames++;
      const hookTop = s.hookY;
      const hookBot = s.hookY + HOOK_H;
      const zoneTop = s.zoneY;
      const zoneBot = s.zoneY + ZONE_H;
      if (hookBot >= zoneTop && hookTop <= zoneBot) {
        s.overlapFrames++;
      }

      const pct = s.totalFrames > 0 ? Math.round((s.overlapFrames / s.totalFrames) * 100) : 0;
      setDisplay({
        hookY: s.hookY,
        zoneY: s.zoneY,
        zoneH: ZONE_H,
        overlap: pct,
      });

      animRef.current = requestAnimationFrame(loop);
    };
    animRef.current = requestAnimationFrame(loop);
    return () => { if (animRef.current) cancelAnimationFrame(animRef.current); };
  }, [phase]);

  // Input handling
  useEffect(() => {
    const down = (e) => {
      if (phase === 'ready' && (e.code === 'Space' || e.type === 'mousedown' || e.type === 'touchstart')) {
        e.preventDefault(); start(); return;
      }
      if (phase === 'playing' && stateRef.current) {
        e.preventDefault();
        stateRef.current.pressing = true;
      }
    };
    const up = () => {
      if (stateRef.current) stateRef.current.pressing = false;
    };
    window.addEventListener('keydown', down);
    window.addEventListener('keyup', up);
    return () => {
      window.removeEventListener('keydown', down);
      window.removeEventListener('keyup', up);
    };
  }, [phase]);

  if (phase === 'ready') {
    return (
      <div className='farm-gc-ready'>
        <div className='farm-gc-ready-emoji'>{game.emoji}</div>
        <div className='farm-gc-ready-desc'>🎣 {t('按住上浮，松开下沉')} — {t('保持在鱼区域内')}!</div>
        <div className='farm-gc-ready-hint'>10s · {t('空格/长按屏幕')}</div>
        <Button theme='solid' size='large' onClick={start} className='farm-btn'
          style={{ fontWeight: 700, minWidth: 140 }}>▶ {t('开始')}</Button>
      </div>
    );
  }

  if (phase === 'done') {
    const pct = display.overlap;
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-gc-score-big' style={{ marginBottom: 8 }}>
          {pct >= 70 ? '🐟' : pct >= 40 ? '🎣' : '🌊'}
        </div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 4 }}>
          {t('命中率')}: {pct}%
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>
          {pct >= 70 ? t('大丰收') + '!' : pct >= 40 ? t('还不错') : t('鱼跑了')}
        </div>
      </div>
    );
  }

  // Playing
  const barHeight = 260;
  const hookPx = (display.hookY / BAR_H) * barHeight;
  const zonePx = (display.zoneY / BAR_H) * barHeight;
  const zoneHPx = (display.zoneH / BAR_H) * barHeight;
  const hookInZone = (display.hookY + HOOK_H >= display.zoneY) && (display.hookY <= display.zoneY + ZONE_H);

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12 }}
      onMouseDown={() => { if (stateRef.current) stateRef.current.pressing = true; }}
      onMouseUp={() => { if (stateRef.current) stateRef.current.pressing = false; }}
      onTouchStart={(e) => { e.preventDefault(); if (stateRef.current) stateRef.current.pressing = true; }}
      onTouchEnd={() => { if (stateRef.current) stateRef.current.pressing = false; }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%', maxWidth: 200 }}>
        <span className='farm-gc-countdown' style={{ color: timeLeft <= 3 ? 'var(--farm-danger)' : 'var(--farm-text-0)', fontSize: 22 }}>
          {timeLeft}s
        </span>
        <span style={{ fontSize: 14, fontWeight: 700, color: display.overlap >= 50 ? 'var(--farm-leaf)' : 'var(--farm-harvest)' }}>
          {display.overlap}%
        </span>
      </div>

      <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
        {/* Fish bar */}
        <div className='farm-gc-fishbar' style={{ height: barHeight }}>
          <div className='farm-gc-fishbar-zone' style={{ top: zonePx, height: zoneHPx }}>
            <span style={{ position: 'absolute', left: '50%', top: '50%', transform: 'translate(-50%,-50%)', fontSize: 16 }}>🐟</span>
          </div>
          <div className='farm-gc-fishbar-hook' style={{ top: hookPx, background: hookInZone ? 'var(--farm-leaf)' : 'var(--farm-danger)', borderRadius: '50%' }}>
            🪝
          </div>
        </div>

        {/* Instructions */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, fontSize: 13, color: 'var(--farm-text-2)' }}>
          <div>⬆️ {t('按住')} = {t('上浮')}</div>
          <div>⬇️ {t('松开')} = {t('下沉')}</div>
          <div style={{ marginTop: 8, fontSize: 20 }}>
            {hookInZone ? '✅' : '❌'}
          </div>
        </div>
      </div>

      <div style={{ fontSize: 12, color: 'var(--farm-text-3)' }}>
        {t('按住空格/屏幕让钩子上浮')}
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   7. PowerChopGame — 砍树 / 抢收 👨‍🌾
   力量条来回移动，在红色甜区按空格，3 刀砍完
   视觉：原木被砍裂动画
   ═══════════════════════════════════════════════════════════════ */
export const PowerChopGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [round, setRound] = useState(0);
  const [scores, setScores] = useState([]);
  const [cursorPos, setCursorPos] = useState(0);
  const [flashResult, setFlashResult] = useState(null);
  const posRef = useRef(0);
  const dirRef = useRef(1);
  const animRef = useRef(null);
  const maxRounds = 5;
  const sweetCenter = 50;
  const sweetHalf = 8;

  useEffect(() => {
    return () => { if (animRef.current) cancelAnimationFrame(animRef.current); };
  }, []);

  const startRound = useCallback(() => {
    posRef.current = 0;
    dirRef.current = 1;
    setFlashResult(null);
    const speed = 1.2 + round * 0.4;
    const animate = () => {
      posRef.current += dirRef.current * speed;
      if (posRef.current >= 100) { posRef.current = 100; dirRef.current = -1; }
      if (posRef.current <= 0) { posRef.current = 0; dirRef.current = 1; }
      setCursorPos(posRef.current);
      animRef.current = requestAnimationFrame(animate);
    };
    animRef.current = requestAnimationFrame(animate);
  }, [round]);

  const startGame = () => {
    setRound(0);
    setScores([]);
    setPhase('playing');
    setTimeout(() => startRound(), 100);
  };

  useEffect(() => {
    if (phase === 'playing' && round > 0) {
      startRound();
    }
  }, [round, phase]);

  const handleChop = () => {
    if (phase !== 'playing') return;
    cancelAnimationFrame(animRef.current);
    const dist = Math.abs(posRef.current - sweetCenter);
    let s = 0;
    if (dist <= sweetHalf) s = 100;
    else if (dist <= sweetHalf * 2) s = 70;
    else if (dist <= sweetHalf * 3) s = 40;
    else s = 10;

    const newScores = [...scores, s];
    setScores(newScores);
    setFlashResult(s >= 70 ? 'good' : 'bad');

    const nextRound = round + 1;
    if (nextRound >= maxRounds) {
      setPhase('done');
      const avg = newScores.reduce((a, b) => a + b, 0) / maxRounds;
      onComplete(avg / 100, Math.round(avg));
    } else {
      setTimeout(() => setRound(nextRound), 600);
    }
  };

  useEffect(() => {
    const handler = (e) => {
      if ((e.code === 'Space' || e.code === 'Enter') && phase === 'ready') { e.preventDefault(); startGame(); }
      if ((e.code === 'Space' || e.code === 'Enter') && phase === 'playing') { e.preventDefault(); handleChop(); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [phase, round, scores]);

  if (phase === 'ready') {
    return (
      <div className='farm-gc-ready'>
        <div className='farm-gc-ready-emoji'>{game.emoji}</div>
        <div className='farm-gc-ready-desc'>🎯 {t('在指针到达红色区域时按下')}!</div>
        <div className='farm-gc-ready-hint'>{maxRounds} {t('刀')} · {t('速度递增')} · {t('空格键')}</div>
        <Button theme='solid' size='large' onClick={startGame} className='farm-btn'
          style={{ fontWeight: 700, minWidth: 140 }}>▶ {t('开始')}</Button>
      </div>
    );
  }

  if (phase === 'done') {
    const avg = scores.reduce((a, b) => a + b, 0) / maxRounds;
    const perfects = scores.filter(s => s === 100).length;
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-gc-score-big' style={{ marginBottom: 8 }}>
          {avg >= 80 ? '🪓' : avg >= 50 ? '👍' : '😅'}
        </div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 4 }}>
          {t('精准度')}: {Math.round(avg)}%
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>
          {scores.map((s, i) => `#${i + 1}:${s}`).join(' · ')}
          {perfects > 0 && ` · 🎯 ×${perfects}`}
        </div>
      </div>
    );
  }

  // Playing
  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 16 }}>
      <div style={{ fontSize: 14, fontWeight: 700 }}>
        {t('第')} {round + 1}/{maxRounds} {t('刀')}
        {flashResult && (
          <span style={{ marginLeft: 8, color: flashResult === 'good' ? 'var(--farm-leaf)' : 'var(--farm-danger)' }}>
            {flashResult === 'good' ? '✅ ' + t('好砍') + '!' : '❌ ' + t('偏了')}
          </span>
        )}
      </div>

      {/* Power bar */}
      <div className='farm-gc-bar power' style={{ height: 32, borderRadius: 16 }}>
        {/* Sweet spot indicator */}
        <div style={{
          position: 'absolute', top: 0, bottom: 0,
          left: `${sweetCenter - sweetHalf}%`,
          width: `${sweetHalf * 2}%`,
          background: 'rgba(184, 66, 51, 0.25)',
          borderLeft: '2px solid var(--farm-danger)',
          borderRight: '2px solid var(--farm-danger)',
          zIndex: 1,
        }} />
        {/* Cursor */}
        <div className='farm-gc-bar-marker' style={{ left: `${cursorPos}%` }} />
      </div>

      {/* Log visualization */}
      <div style={{ display: 'flex', gap: 4, justifyContent: 'center' }}>
        {scores.map((s, i) => (
          <span key={i} style={{
            fontSize: 24,
            opacity: 0.3 + (s / 100) * 0.7,
            filter: s >= 70 ? 'none' : 'grayscale(0.6)',
          }}>
            {s >= 70 ? '🪵' : '🌲'}
          </span>
        ))}
        {Array.from({ length: maxRounds - scores.length }, (_, i) => (
          <span key={`rem-${i}`} style={{ fontSize: 24, opacity: 0.2 }}>🌲</span>
        ))}
      </div>

      <Button theme='solid' size='large' className='farm-btn'
        style={{ fontWeight: 700, minWidth: 160, fontSize: 16 }}
        onClick={handleChop}>
        🪓 {t('砍')}!
      </Button>

      <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', justifyContent: 'center' }}>
        {scores.map((s, i) => (
          <span key={i} className={`farm-pill ${s >= 70 ? 'farm-pill-green' : s >= 40 ? 'farm-pill-blue' : 'farm-pill-red'}`}>
            #{i + 1}: {s}
          </span>
        ))}
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   8. LassoGame — 套牛 🐮
   牛在屏幕上跑，按住蓄力决定抛出距离，松开抛出
   必须预判牛的走位，3 次机会
   ═══════════════════════════════════════════════════════════════ */
export const LassoGame = ({ game, onComplete, t }) => {
  const canvasRef = useRef(null);
  const stateRef = useRef(null);
  const animRef = useRef(null);
  const [phase, setPhase] = useState('ready');
  const [result, setResult] = useState(null);

  const W = 370, H = 200;
  const MAX_ATTEMPTS = 3;

  const initState = () => ({
    cowX: 30 + Math.random() * 40,
    cowDir: Math.random() < 0.5 ? 1 : -1,
    cowSpeed: 0.3 + Math.random() * 0.2,
    playerX: 50,
    charging: false,
    charge: 0,
    lassoFlying: false,
    lassoX: 50,
    lassoTargetX: 50,
    lassoProgress: 0,
    attempts: 0,
    catches: 0,
    roundPhase: 'aim',
    tick: 0,
  });

  const start = () => {
    stateRef.current = initState();
    setResult(null);
    setPhase('playing');
  };

  useEffect(() => {
    if (phase !== 'playing') return;
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    canvas.width = W;
    canvas.height = H;

    const loop = () => {
      const s = stateRef.current;
      if (!s) return;
      s.tick++;

      // Cow movement (bounce)
      s.cowX += s.cowDir * s.cowSpeed;
      if (s.cowX >= 90) { s.cowDir = -1; s.cowSpeed = 0.3 + Math.random() * 0.25; }
      if (s.cowX <= 10) { s.cowDir = 1; s.cowSpeed = 0.3 + Math.random() * 0.25; }

      // Charging
      if (s.charging && s.roundPhase === 'aim') {
        s.charge = Math.min(100, s.charge + 1.5);
      }

      // Lasso flying
      if (s.lassoFlying) {
        s.lassoProgress += 4;
        if (s.lassoProgress >= 100) {
          // Check catch
          const landX = s.lassoTargetX;
          const dist = Math.abs(landX - s.cowX);
          if (dist < 8) {
            s.catches++;
          }
          s.attempts++;
          s.lassoFlying = false;
          s.charge = 0;

          if (s.attempts >= MAX_ATTEMPTS) {
            setPhase('done');
            const score = Math.round((s.catches / MAX_ATTEMPTS) * 100);
            setResult({ catches: s.catches, attempts: MAX_ATTEMPTS, score });
            onComplete(score / 100, score);
            return;
          }
          // Reset for next attempt
          s.roundPhase = 'aim';
          s.cowSpeed = 0.3 + Math.random() * 0.3;
        }
      }

      // Render
      ctx.clearRect(0, 0, W, H);
      ctx.fillStyle = '#1a2332';
      ctx.fillRect(0, 0, W, H);

      // Ground
      ctx.fillStyle = '#2d4a2e';
      ctx.fillRect(0, H - 40, W, 40);

      // Fence posts
      for (let i = 0; i < W; i += 50) {
        ctx.fillStyle = '#8b6914';
        ctx.fillRect(i, H - 55, 4, 20);
      }
      ctx.strokeStyle = '#8b6914';
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.moveTo(0, H - 48);
      ctx.lineTo(W, H - 48);
      ctx.stroke();

      // Cow
      const cowPx = (s.cowX / 100) * W;
      const cowY = H - 70;
      const cowBounce = Math.sin(s.tick * 0.1) * 2;
      ctx.font = '32px serif';
      ctx.textAlign = 'center';
      ctx.save();
      if (s.cowDir < 0) {
        ctx.translate(cowPx, cowY + cowBounce);
        ctx.scale(-1, 1);
        ctx.fillText('🐮', 0, 0);
      } else {
        ctx.fillText('🐮', cowPx, cowY + cowBounce);
      }
      ctx.restore();

      // Player
      const playerPx = (s.playerX / 100) * W;
      ctx.font = '28px serif';
      ctx.fillText('🤠', playerPx, H - 20);

      // Charge bar
      if (s.charging || s.charge > 0) {
        ctx.fillStyle = 'rgba(0,0,0,0.5)';
        ctx.fillRect(playerPx - 25, H - 10, 50, 8);
        const chargeColor = s.charge < 40 ? '#6fa85e' : s.charge < 75 ? '#c8922a' : '#b84233';
        ctx.fillStyle = chargeColor;
        ctx.fillRect(playerPx - 25, H - 10, s.charge * 0.5, 8);
      }

      // Lasso in flight
      if (s.lassoFlying) {
        const startPx = playerPx;
        const endPx = (s.lassoTargetX / 100) * W;
        const currentPx = startPx + (endPx - startPx) * (s.lassoProgress / 100);
        const arcY = cowY - 10 - Math.sin((s.lassoProgress / 100) * Math.PI) * 40;

        ctx.beginPath();
        ctx.moveTo(startPx, H - 30);
        ctx.quadraticCurveTo(currentPx, arcY - 20, currentPx, arcY);
        ctx.strokeStyle = '#d4a';
        ctx.lineWidth = 2;
        ctx.stroke();

        ctx.font = '20px serif';
        ctx.fillText('🔵', currentPx, arcY);
      }

      // Attempts display
      ctx.fillStyle = '#fff';
      ctx.font = 'bold 12px sans-serif';
      ctx.textAlign = 'left';
      ctx.fillText(`${t('机会')}: ${MAX_ATTEMPTS - s.attempts}  ✅ ${s.catches}`, 10, 18);

      animRef.current = requestAnimationFrame(loop);
    };
    animRef.current = requestAnimationFrame(loop);
    return () => { if (animRef.current) cancelAnimationFrame(animRef.current); };
  }, [phase]);

  const throwLasso = () => {
    const s = stateRef.current;
    if (!s || s.roundPhase !== 'aim' || s.lassoFlying) return;
    const throwDist = s.charge * 0.8;
    s.lassoTargetX = s.playerX + throwDist * (s.cowX > s.playerX ? 1 : -1) * 0.01 * 100;
    s.lassoTargetX = Math.max(5, Math.min(95, s.playerX + (s.cowX - s.playerX) * (s.charge / 80)));
    s.lassoFlying = true;
    s.lassoProgress = 0;
    s.charging = false;
    s.roundPhase = 'throwing';
  };

  useEffect(() => {
    const down = (e) => {
      if (phase === 'ready' && e.code === 'Space') { e.preventDefault(); start(); return; }
      if (phase === 'playing' && stateRef.current && !stateRef.current.lassoFlying) {
        e.preventDefault();
        stateRef.current.charging = true;
      }
    };
    const up = (e) => {
      if (phase === 'playing' && stateRef.current && stateRef.current.charging) {
        throwLasso();
      }
    };
    window.addEventListener('keydown', down);
    window.addEventListener('keyup', up);
    return () => {
      window.removeEventListener('keydown', down);
      window.removeEventListener('keyup', up);
    };
  }, [phase]);

  if (phase === 'ready') {
    return (
      <div className='farm-gc-ready'>
        <div className='farm-gc-ready-emoji'>{game.emoji}</div>
        <div className='farm-gc-ready-desc'>🤠 {t('按住蓄力，松开抛出套索')}!</div>
        <div className='farm-gc-ready-hint'>{MAX_ATTEMPTS} {t('次机会')} · {t('预判牛的走位')} · {t('空格/长按')}</div>
        <Button theme='solid' size='large' onClick={start} className='farm-btn'
          style={{ fontWeight: 700, minWidth: 140 }}>▶ {t('开始')}</Button>
      </div>
    );
  }

  if (phase === 'done' && result) {
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-gc-score-big' style={{ marginBottom: 8 }}>
          {result.catches >= 3 ? '🏆' : result.catches >= 2 ? '🤠' : result.catches >= 1 ? '👍' : '😢'}
        </div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 4 }}>
          {t('套中')}: {result.catches}/{result.attempts}
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>{t('得分')}: {result.score}</div>
      </div>
    );
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10 }}>
      <div className='farm-gc' style={{ aspectRatio: `${W}/${H}` }}>
        <canvas ref={canvasRef} />
      </div>
      <Button theme='solid' size='default' className='farm-btn'
        style={{ fontWeight: 700, minWidth: 140 }}
        onMouseDown={() => { if (stateRef.current && !stateRef.current.lassoFlying) stateRef.current.charging = true; }}
        onMouseUp={() => throwLasso()}
        onTouchStart={(e) => { e.preventDefault(); if (stateRef.current && !stateRef.current.lassoFlying) stateRef.current.charging = true; }}
        onTouchEnd={() => throwLasso()}>
        🤠 {t('按住蓄力')}
      </Button>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   9. AnglePowerGame — 拔萝卜 / 高尔夫 🥕
   两阶段判定：先定角度，再定力度，把萝卜/石子打入目标区
   ═══════════════════════════════════════════════════════════════ */
export const AnglePowerGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [round, setRound] = useState(0);
  const [step, setStep] = useState('angle');
  const [angleVal, setAngleVal] = useState(0);
  const [powerVal, setPowerVal] = useState(0);
  const [lockedAngle, setLockedAngle] = useState(null);
  const [scores, setScores] = useState([]);
  const [flyResult, setFlyResult] = useState(null);
  const angleRef = useRef(0);
  const powerRef = useRef(0);
  const dirRef = useRef(1);
  const animRef = useRef(null);
  const maxRounds = 3;
  const targetAngle = 45;
  const targetPower = 65;

  useEffect(() => {
    return () => { if (animRef.current) cancelAnimationFrame(animRef.current); };
  }, []);

  const startGame = () => {
    setRound(0);
    setScores([]);
    setPhase('playing');
    setStep('angle');
    startAngleSweep();
  };

  const startAngleSweep = () => {
    angleRef.current = 0;
    dirRef.current = 1;
    setLockedAngle(null);
    setFlyResult(null);
    const animate = () => {
      angleRef.current += dirRef.current * 1.2;
      if (angleRef.current >= 90) { dirRef.current = -1; }
      if (angleRef.current <= 0) { dirRef.current = 1; }
      setAngleVal(angleRef.current);
      animRef.current = requestAnimationFrame(animate);
    };
    animRef.current = requestAnimationFrame(animate);
  };

  const startPowerSweep = () => {
    powerRef.current = 0;
    dirRef.current = 1;
    const animate = () => {
      powerRef.current += dirRef.current * 1.5;
      if (powerRef.current >= 100) { dirRef.current = -1; }
      if (powerRef.current <= 0) { dirRef.current = 1; }
      setPowerVal(powerRef.current);
      animRef.current = requestAnimationFrame(animate);
    };
    animRef.current = requestAnimationFrame(animate);
  };

  const handleLock = () => {
    if (phase !== 'playing') return;
    cancelAnimationFrame(animRef.current);

    if (step === 'angle') {
      setLockedAngle(angleRef.current);
      setStep('power');
      setTimeout(() => startPowerSweep(), 300);
    } else if (step === 'power') {
      const angleDist = Math.abs(angleRef.current ? lockedAngle : angleVal - targetAngle);
      const aAngle = Math.abs((lockedAngle || 0) - targetAngle);
      const aPower = Math.abs(powerRef.current - targetPower);
      const angleScore = Math.max(0, 100 - aAngle * 2.5);
      const powerScore = Math.max(0, 100 - aPower * 2);
      const combined = Math.round((angleScore + powerScore) / 2);
      setFlyResult(combined);

      const newScores = [...scores, combined];
      setScores(newScores);

      const nextRound = round + 1;
      if (nextRound >= maxRounds) {
        setPhase('done');
        const avg = newScores.reduce((a, b) => a + b, 0) / maxRounds;
        onComplete(avg / 100, Math.round(avg));
      } else {
        setTimeout(() => {
          setRound(nextRound);
          setStep('angle');
          startAngleSweep();
        }, 1000);
      }
    }
  };

  useEffect(() => {
    const handler = (e) => {
      if ((e.code === 'Space' || e.code === 'Enter') && phase === 'ready') { e.preventDefault(); startGame(); }
      if ((e.code === 'Space' || e.code === 'Enter') && phase === 'playing') { e.preventDefault(); handleLock(); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [phase, step, scores, round, lockedAngle]);

  if (phase === 'ready') {
    return (
      <div className='farm-gc-ready'>
        <div className='farm-gc-ready-emoji'>{game.emoji}</div>
        <div className='farm-gc-ready-desc'>🎯 {t('先定角度，再定力度')}!</div>
        <div className='farm-gc-ready-hint'>{maxRounds} {t('轮')} · {t('目标')}: 45° + 65% · {t('空格锁定')}</div>
        <Button theme='solid' size='large' onClick={startGame} className='farm-btn'
          style={{ fontWeight: 700, minWidth: 140 }}>▶ {t('开始')}</Button>
      </div>
    );
  }

  if (phase === 'done') {
    const avg = scores.reduce((a, b) => a + b, 0) / maxRounds;
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-gc-score-big' style={{ marginBottom: 8 }}>
          {avg >= 75 ? '🎯' : avg >= 45 ? '👍' : '😅'}
        </div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 4 }}>
          {t('精准度')}: {Math.round(avg)}%
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>
          {scores.map((s, i) => `R${i + 1}:${s}`).join(' · ')}
        </div>
      </div>
    );
  }

  // Playing
  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 14 }}>
      <div style={{ fontSize: 14, fontWeight: 700 }}>
        {t('第')} {round + 1}/{maxRounds} {t('轮')} —
        {step === 'angle' ? ` 📐 ${t('定角度')}` : ` 💪 ${t('定力度')}`}
        {flyResult !== null && (
          <span style={{ marginLeft: 8, color: flyResult >= 60 ? 'var(--farm-leaf)' : 'var(--farm-danger)' }}>
            → {flyResult}{t('分')}!
          </span>
        )}
      </div>

      {/* Angle indicator */}
      <div style={{
        width: 140, height: 80, position: 'relative',
        borderBottom: '2px solid var(--farm-border)',
      }}>
        <div style={{
          position: 'absolute', bottom: 0, left: '50%',
          width: 3, height: 70,
          background: step === 'angle' ? 'var(--farm-danger)' : (lockedAngle !== null ? 'var(--farm-leaf)' : '#6b7280'),
          transformOrigin: 'bottom center',
          transform: `rotate(${-(step === 'angle' ? angleVal : (lockedAngle || 0)) + 45}deg)`,
          transition: step === 'power' ? 'none' : undefined,
          borderRadius: 2,
        }} />
        <div style={{ position: 'absolute', bottom: -18, left: '50%', transform: 'translateX(-50%)', fontSize: 11, color: 'var(--farm-text-2)' }}>
          {Math.round(step === 'angle' ? angleVal : (lockedAngle || 0))}°
          {lockedAngle !== null && <span style={{ color: 'var(--farm-leaf)' }}> ✓</span>}
        </div>
        {/* Target marker */}
        <div style={{
          position: 'absolute', bottom: 0, left: '50%',
          width: 2, height: 70,
          background: 'rgba(74, 124, 63, 0.3)',
          transformOrigin: 'bottom center',
          transform: `rotate(${-targetAngle + 45}deg)`,
          borderRadius: 2,
        }} />
      </div>

      {/* Power bar */}
      {step === 'power' && (
        <div style={{ width: '100%', maxWidth: 300 }}>
          <div style={{ fontSize: 12, color: 'var(--farm-text-3)', marginBottom: 4, textAlign: 'center' }}>
            {t('力度')} ({t('目标')}: {targetPower}%)
          </div>
          <div className='farm-gc-bar power' style={{ height: 24, borderRadius: 12 }}>
            <div className='farm-gc-bar-fill' style={{ width: `${powerVal}%` }} />
            <div className='farm-gc-bar-marker' style={{ left: `${targetPower}%`, background: 'var(--farm-leaf)', width: 3 }} />
          </div>
          <div style={{ textAlign: 'center', fontSize: 13, fontWeight: 700, marginTop: 4 }}>
            {Math.round(powerVal)}%
          </div>
        </div>
      )}

      <Button theme='solid' size='large' className='farm-btn'
        style={{ fontWeight: 700, minWidth: 160, fontSize: 15 }}
        onClick={handleLock}>
        {step === 'angle' ? `📐 ${t('锁定角度')}` : `💪 ${t('锁定力度')}`}
      </Button>

      <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', justifyContent: 'center' }}>
        {scores.map((s, i) => (
          <span key={i} className={`farm-pill ${s >= 60 ? 'farm-pill-green' : s >= 30 ? 'farm-pill-blue' : 'farm-pill-red'}`}>
            R{i + 1}: {s}
          </span>
        ))}
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   10. WaterCatchGame — 接水滴 / 育苗 🌱
   水滴从上方随机位置掉落，玩家用鼠标/方向键控制桶左右接
   漏接 3 个失败
   ═══════════════════════════════════════════════════════════════ */
export const WaterCatchGame = ({ game, onComplete, t }) => {
  const canvasRef = useRef(null);
  const stateRef = useRef(null);
  const animRef = useRef(null);
  const [phase, setPhase] = useState('ready');
  const [display, setDisplay] = useState({ caught: 0, missed: 0, lives: 3 });

  const W = 340, H = 260;
  const BUCKET_W = 40;
  const MAX_MISSES = 3;

  const initState = () => ({
    bucketX: W / 2,
    drops: [],
    caught: 0,
    missed: 0,
    tick: 0,
    spawnRate: 50,
    dropSpeed: 2,
    mouseX: W / 2,
    keysDown: {},
  });

  const start = () => {
    stateRef.current = initState();
    setDisplay({ caught: 0, missed: 0, lives: MAX_MISSES });
    setPhase('playing');
  };

  useEffect(() => {
    return () => { if (animRef.current) cancelAnimationFrame(animRef.current); };
  }, []);

  useEffect(() => {
    if (phase !== 'playing') return;
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    canvas.width = W;
    canvas.height = H;

    const loop = () => {
      const s = stateRef.current;
      if (!s) return;
      s.tick++;

      // Keyboard movement
      if (s.keysDown['ArrowLeft'] || s.keysDown['KeyA']) s.bucketX -= 4;
      if (s.keysDown['ArrowRight'] || s.keysDown['KeyD']) s.bucketX += 4;
      s.bucketX = Math.max(BUCKET_W / 2, Math.min(W - BUCKET_W / 2, s.bucketX));

      // Difficulty ramp
      if (s.tick % 300 === 0) {
        s.spawnRate = Math.max(20, s.spawnRate - 5);
        s.dropSpeed = Math.min(5, s.dropSpeed + 0.3);
      }

      // Spawn drops
      if (s.tick % s.spawnRate === 0) {
        const isSpecial = Math.random() < 0.1;
        s.drops.push({
          x: 20 + Math.random() * (W - 40),
          y: -10,
          speed: s.dropSpeed + Math.random() * 0.5,
          emoji: isSpecial ? '⭐' : '💧',
          value: isSpecial ? 3 : 1,
        });
      }

      // Update drops
      for (let i = s.drops.length - 1; i >= 0; i--) {
        const d = s.drops[i];
        d.y += d.speed;

        // Catch check
        if (d.y >= H - 35 && d.y <= H - 15) {
          if (Math.abs(d.x - s.bucketX) < BUCKET_W / 2 + 8) {
            s.caught += d.value;
            s.drops.splice(i, 1);
            continue;
          }
        }

        // Miss check
        if (d.y > H + 10) {
          s.missed++;
          s.drops.splice(i, 1);
          if (s.missed >= MAX_MISSES) {
            setPhase('done');
            const score = Math.min(100, s.caught * 5);
            setDisplay({ caught: s.caught, missed: s.missed, lives: 0 });
            onComplete(score / 100, s.caught);
            return;
          }
        }
      }

      setDisplay({ caught: s.caught, missed: s.missed, lives: MAX_MISSES - s.missed });

      // Render
      ctx.clearRect(0, 0, W, H);
      ctx.fillStyle = '#0f172a';
      ctx.fillRect(0, 0, W, H);

      // Rain effect background
      for (let i = 0; i < 8; i++) {
        const rx = (s.tick * 2 + i * 47) % W;
        const ry = (s.tick * 3 + i * 31) % H;
        ctx.fillStyle = 'rgba(96, 165, 250, 0.1)';
        ctx.fillRect(rx, ry, 1, 8);
      }

      // Drops
      for (const d of s.drops) {
        ctx.font = '18px serif';
        ctx.textAlign = 'center';
        ctx.fillText(d.emoji, d.x, d.y);
      }

      // Bucket
      ctx.font = '28px serif';
      ctx.textAlign = 'center';
      ctx.fillText('🪣', s.bucketX, H - 18);

      // Ground
      ctx.fillStyle = '#2d4a2e';
      ctx.fillRect(0, H - 8, W, 8);

      // HUD
      ctx.fillStyle = '#fff';
      ctx.font = 'bold 12px sans-serif';
      ctx.textAlign = 'left';
      ctx.fillText(`💧 ${s.caught}`, 8, 16);
      ctx.textAlign = 'right';
      const heartsText = '❤️'.repeat(MAX_MISSES - s.missed) + '🖤'.repeat(s.missed);
      ctx.fillText(heartsText, W - 8, 16);

      animRef.current = requestAnimationFrame(loop);
    };
    animRef.current = requestAnimationFrame(loop);
    return () => { if (animRef.current) cancelAnimationFrame(animRef.current); };
  }, [phase]);

  // Mouse/touch tracking
  useEffect(() => {
    if (phase !== 'playing') return;
    const canvas = canvasRef.current;
    if (!canvas) return;
    const handleMouse = (e) => {
      const rect = canvas.getBoundingClientRect();
      const x = (e.clientX - rect.left) * (W / rect.width);
      if (stateRef.current) stateRef.current.bucketX = Math.max(BUCKET_W / 2, Math.min(W - BUCKET_W / 2, x));
    };
    const handleTouch = (e) => {
      e.preventDefault();
      const rect = canvas.getBoundingClientRect();
      const x = (e.touches[0].clientX - rect.left) * (W / rect.width);
      if (stateRef.current) stateRef.current.bucketX = Math.max(BUCKET_W / 2, Math.min(W - BUCKET_W / 2, x));
    };
    canvas.addEventListener('mousemove', handleMouse);
    canvas.addEventListener('touchmove', handleTouch, { passive: false });
    return () => {
      canvas.removeEventListener('mousemove', handleMouse);
      canvas.removeEventListener('touchmove', handleTouch);
    };
  }, [phase]);

  // Keyboard
  useEffect(() => {
    const down = (e) => {
      if (phase === 'ready' && e.code === 'Space') { e.preventDefault(); start(); return; }
      if (phase === 'playing' && stateRef.current) {
        stateRef.current.keysDown[e.code] = true;
      }
    };
    const up = (e) => {
      if (stateRef.current) delete stateRef.current.keysDown[e.code];
    };
    window.addEventListener('keydown', down);
    window.addEventListener('keyup', up);
    return () => {
      window.removeEventListener('keydown', down);
      window.removeEventListener('keyup', up);
    };
  }, [phase]);

  if (phase === 'ready') {
    return (
      <div className='farm-gc-ready'>
        <div className='farm-gc-ready-emoji'>{game.emoji}</div>
        <div className='farm-gc-ready-desc'>💧 {t('移动水桶接住水滴')}!</div>
        <div className='farm-gc-ready-hint'>{t('鼠标/方向键移动')} · {t('漏接')} {MAX_MISSES} {t('个失败')} · ⭐=3{t('分')}</div>
        <Button theme='solid' size='large' onClick={start} className='farm-btn'
          style={{ fontWeight: 700, minWidth: 140 }}>▶ {t('开始')}</Button>
      </div>
    );
  }

  if (phase === 'done') {
    const score = Math.min(100, display.caught * 5);
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-gc-score-big' style={{ marginBottom: 8 }}>
          {display.caught >= 15 ? '🌊' : display.caught >= 8 ? '💧' : '🏜️'}
        </div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 4 }}>
          {t('接到')}: {display.caught} 💧
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>
          {t('得分')}: {score}
        </div>
      </div>
    );
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%', maxWidth: W }}>
        <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--farm-sky)' }}>💧 {display.caught}</span>
        <span style={{ fontSize: 14 }}>
          {'❤️'.repeat(display.lives)}{'🖤'.repeat(MAX_MISSES - display.lives)}
        </span>
      </div>
      <div className='farm-gc' style={{ maxWidth: W, aspectRatio: `${W}/${H}`, cursor: 'none' }}>
        <canvas ref={canvasRef} />
      </div>
      <div style={{ fontSize: 12, color: 'var(--farm-text-3)' }}>
        {t('鼠标/触摸移动')} · ← → {t('方向键')}
      </div>
    </div>
  );
};
