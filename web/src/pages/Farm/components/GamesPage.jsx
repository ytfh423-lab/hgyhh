import React, { useCallback, useEffect, useState, useRef } from 'react';
import { Button, Spin, Typography } from '@douyinfe/semi-ui';
import { API, showError } from './utils';

const { Text } = Typography;

/* ═══════════════════════════════════════════════════════════════
   WHEEL_SECTORS — 转盘扇区颜色
   ═══════════════════════════════════════════════════════════════ */
const WHEEL_COLORS = [
  '#ef4444', '#3b82f6', '#22c55e', '#f59e0b',
  '#8b5cf6', '#06b6d4', '#ec4899', '#14b8a6',
];

/* ═══════════════════════════════════════════════════════════════
   SpinningWheel — 真实旋转转盘（CSS transform）
   ═══════════════════════════════════════════════════════════════ */
const SpinningWheel = ({ onSpin, spinning, result, gameLoading, t }) => {
  const wheelRef = useRef(null);
  const [rotation, setRotation] = useState(0);
  const [showResult, setShowResult] = useState(false);
  const sectors = result?.sectors || [
    '🎁', '💰', '🍀', '⭐', '🎯', '🏆', '💎', '🌟',
  ];
  const sectorCount = sectors.length;
  const sectorAngle = 360 / sectorCount;

  const handleSpin = async () => {
    if (spinning || gameLoading) return;
    setShowResult(false);
    const data = await onSpin();
    if (!data) return;
    // Calculate target angle: prize_index tells us where to land
    const prizeIdx = data.prize_index ?? Math.floor(Math.random() * sectorCount);
    // Pointer is at top (0°). Sector 0 starts at 0°.
    // To land on prizeIdx, rotate so that sector's center aligns with top.
    const targetSectorCenter = prizeIdx * sectorAngle + sectorAngle / 2;
    const extraSpins = 5 * 360; // 5 full rotations
    const newRotation = rotation + extraSpins + (360 - targetSectorCenter);
    setRotation(newRotation);
    // Show result after spin animation
    setTimeout(() => setShowResult(true), 4200);
  };

  // Draw sectors with conic-gradient
  const conicStops = sectors.map((_, i) => {
    const color = WHEEL_COLORS[i % WHEEL_COLORS.length];
    const start = (i * sectorAngle).toFixed(1);
    const end = ((i + 1) * sectorAngle).toFixed(1);
    return `${color} ${start}deg ${end}deg`;
  }).join(', ');

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 16 }}>
      <div className='farm-wheel-container'>
        <div className='farm-wheel-pointer'>📍</div>
        <div ref={wheelRef}
          className={`farm-wheel ${rotation > 0 ? 'spinning' : ''}`}
          style={{
            background: `conic-gradient(${conicStops})`,
            transform: `rotate(${rotation}deg)`,
          }}>
          {/* Sector labels */}
          {sectors.map((label, i) => {
            const angle = i * sectorAngle + sectorAngle / 2;
            return (
              <div key={i} style={{
                position: 'absolute',
                top: '50%', left: '50%',
                transform: `rotate(${angle}deg) translateY(-90px)`,
                transformOrigin: '0 0',
                fontSize: 20,
                marginLeft: -10,
                marginTop: -10,
              }}>
                {label}
              </div>
            );
          })}
        </div>
        <div className='farm-wheel-center'>🎰</div>
      </div>

      <Button theme='solid' size='large' onClick={handleSpin}
        loading={gameLoading} className='farm-btn'
        style={{ minWidth: 140, fontWeight: 700 }}>
        🎡 {t('转一次')} ($1)
      </Button>

      {showResult && result && (
        <div className='farm-game-result'>
          <div style={{ fontSize: 16, fontWeight: 700, marginBottom: 4 }}>
            {result.prize_label}
          </div>
          <span className={`farm-ws-profit-badge ${result.net >= 0 ? 'positive' : 'negative'}`}
            style={{ fontSize: 14 }}>
            {result.net >= 0 ? '+' : ''}{result.net.toFixed(2)}
          </span>
        </div>
      )}
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   ScratchCard — Canvas 刮刮卡
   ═══════════════════════════════════════════════════════════════ */
const ScratchCard = ({ onScratch, result, gameLoading, t }) => {
  const canvasRef = useRef(null);
  const [scratching, setScratching] = useState(false);
  const [revealed, setRevealed] = useState(false);
  const [started, setStarted] = useState(false);
  const isDrawing = useRef(false);

  const initCanvas = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const rect = canvas.parentElement.getBoundingClientRect();
    canvas.width = rect.width;
    canvas.height = rect.height;
    // Fill with scratch coating
    const grad = ctx.createLinearGradient(0, 0, canvas.width, canvas.height);
    grad.addColorStop(0, '#6366f1');
    grad.addColorStop(1, '#8b5cf6');
    ctx.fillStyle = grad;
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    // Add text hint
    ctx.fillStyle = 'rgba(255,255,255,0.5)';
    ctx.font = 'bold 16px sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText('✨ ' + t('刮开此处') + ' ✨', canvas.width / 2, canvas.height / 2);
  }, [t]);

  useEffect(() => {
    if (result && !revealed) {
      setStarted(true);
      setTimeout(() => initCanvas(), 50);
    }
  }, [result, revealed, initCanvas]);

  const getPos = (e) => {
    const canvas = canvasRef.current;
    const rect = canvas.getBoundingClientRect();
    const touch = e.touches ? e.touches[0] : e;
    return { x: touch.clientX - rect.left, y: touch.clientY - rect.top };
  };

  const scratch = (pos) => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    ctx.globalCompositeOperation = 'destination-out';
    ctx.beginPath();
    ctx.arc(pos.x, pos.y, 22, 0, Math.PI * 2);
    ctx.fill();
    // Check percentage scratched
    const imgData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    let cleared = 0;
    for (let i = 3; i < imgData.data.length; i += 4) {
      if (imgData.data[i] === 0) cleared++;
    }
    const pct = cleared / (imgData.data.length / 4);
    if (pct > 0.55) {
      setRevealed(true);
    }
  };

  const onPointerDown = (e) => { e.preventDefault(); isDrawing.current = true; scratch(getPos(e)); };
  const onPointerMove = (e) => { e.preventDefault(); if (isDrawing.current) scratch(getPos(e)); };
  const onPointerUp = () => { isDrawing.current = false; };

  const handleBuy = async () => {
    setRevealed(false);
    setStarted(false);
    await onScratch();
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 16 }}>
      {started && result ? (
        <>
          <div className='farm-scratch-container'
            onMouseDown={onPointerDown} onMouseMove={onPointerMove} onMouseUp={onPointerUp} onMouseLeave={onPointerUp}
            onTouchStart={onPointerDown} onTouchMove={onPointerMove} onTouchEnd={onPointerUp}>
            {/* Prize layer underneath */}
            <div className='farm-scratch-prize'>
              <span className='farm-scratch-prize-emoji'>{result.win_symbol || '🎁'}</span>
              <span className='farm-scratch-prize-label'>{result.prize_label}</span>
              <span style={{ fontSize: 12, color: result.net >= 0 ? '#4ade80' : '#ef4444', fontWeight: 700 }}>
                {result.net >= 0 ? '+' : ''}{result.net.toFixed(2)}
              </span>
            </div>
            {/* Scratch canvas on top */}
            {!revealed && <canvas ref={canvasRef} className='farm-scratch-canvas' />}
          </div>
          {revealed && (
            <div className='farm-game-result'>
              <div style={{ fontSize: 14, fontWeight: 600 }}>
                {result.win_symbol}×3 → {result.prize_label}
              </div>
              <div className='farm-game-result-pills'>
                <span className={`farm-ws-profit-badge ${result.net >= 0 ? 'positive' : 'negative'}`}>
                  {result.net >= 0 ? '+' : ''}{result.net.toFixed(2)}
                </span>
              </div>
            </div>
          )}
        </>
      ) : (
        <div style={{ padding: 20, textAlign: 'center' }}>
          <div style={{ fontSize: 48, marginBottom: 12 }}>🎰</div>
          <Button theme='solid' size='large' onClick={handleBuy}
            loading={gameLoading} className='farm-btn'
            style={{ minWidth: 140, fontWeight: 700 }}>
            🃏 {t('刮一张')} ($0.50)
          </Button>
        </div>
      )}
      {started && revealed && (
        <Button size='small' theme='light' onClick={handleBuy}
          loading={gameLoading} className='farm-btn'>
          🔄 {t('再来一张')}
        </Button>
      )}
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   GAME_ENGINE_MAP — 游戏 Key → 引擎 + 配置
   ═══════════════════════════════════════════════════════════════ */
const GAME_ENGINE_MAP = {
  // ⚡ Masher — 连点爆发
  milking:    { engine: 'masher', duration: 7, threshold: 35, actionEmoji: '🥛' },
  cornrace:   { engine: 'masher', duration: 8, threshold: 40, actionEmoji: '🌽' },
  horserace:  { engine: 'masher', duration: 6, threshold: 45, actionEmoji: '🏇' },
  pigchase:   { engine: 'masher', duration: 7, threshold: 35, actionEmoji: '💨' },
  grape:      { engine: 'masher', duration: 8, threshold: 40, actionEmoji: '🍇' },
  weed:       { engine: 'masher', duration: 8, threshold: 35, actionEmoji: '🌿' },
  woodchop:   { engine: 'masher', duration: 7, threshold: 40, actionEmoji: '🪓' },
  thresh:     { engine: 'masher', duration: 8, threshold: 40, actionEmoji: '🌾' },
  pullcarrot: { engine: 'masher', duration: 6, threshold: 30, actionEmoji: '🥕' },
  tame:       { engine: 'masher', duration: 8, threshold: 45, actionEmoji: '🐴' },
  harvest:    { engine: 'masher', duration: 7, threshold: 40, actionEmoji: '🌾' },
  lasso:      { engine: 'masher', duration: 6, threshold: 35, actionEmoji: '🤠' },
  // 🎯 Timing — 精准时机
  fishcomp:   { engine: 'timing', speed: 3.5, sweetSpot: 14 },
  beekeep:    { engine: 'timing', speed: 4.5, sweetSpot: 12 },
  sunflower:  { engine: 'timing', speed: 2.5, sweetSpot: 16 },
  rooster:    { engine: 'timing', speed: 5, sweetSpot: 10 },
  sheepdog:   { engine: 'timing', speed: 3.5, sweetSpot: 14 },
  seedling:   { engine: 'timing', speed: 2.5, sweetSpot: 18 },
  pumpkin:    { engine: 'timing', speed: 3, sweetSpot: 14 },
  duckherd:   { engine: 'timing', speed: 4, sweetSpot: 12 },
  hatchegg:   { engine: 'timing', speed: 2.5, sweetSpot: 16 },
  weather:    { engine: 'timing', speed: 2, sweetSpot: 20 },
  scarecrow:  { engine: 'timing', speed: 3.5, sweetSpot: 14 },
  // 🧠 Reaction — 反应寻找
  bugcatch:   { engine: 'reaction', gridSize: 4, duration: 12, targetEmoji: '🐛', bombEmoji: '💣', spawnMs: 900 },
  egghunt:    { engine: 'reaction', gridSize: 4, duration: 12, targetEmoji: '🥚', bombEmoji: '💩', spawnMs: 1000 },
  fruitpick:  { engine: 'reaction', gridSize: 4, duration: 12, targetEmoji: '🍎', bombEmoji: '🐍', spawnMs: 900 },
  sheepcount: { engine: 'reaction', gridSize: 4, duration: 10, targetEmoji: '🐑', bombEmoji: '🐺', spawnMs: 850 },
  mushroom:   { engine: 'reaction', gridSize: 4, duration: 12, targetEmoji: '🍄', bombEmoji: '☠️', spawnMs: 900 },
  foxhunt:    { engine: 'reaction', gridSize: 4, duration: 10, targetEmoji: '🦊', bombEmoji: '🐔', spawnMs: 800 },
  produce:    { engine: 'reaction', gridSize: 4, duration: 15, targetEmoji: '🏆', bombEmoji: '💩', spawnMs: 1000 },
};

/* ═══════════════════════════════════════════════════════════════
   ⚡ MasherEngine — 连点爆发引擎
   限时疯狂点击，真实统计 clickCount
   ═══════════════════════════════════════════════════════════════ */
const MasherEngine = ({ config, game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(config.duration || 7);
  const [displayClicks, setDisplayClicks] = useState(0);
  const clicksRef = useRef(0);
  const timerRef = useRef(null);
  const threshold = config.threshold || 40;

  useEffect(() => {
    return () => { if (timerRef.current) clearInterval(timerRef.current); };
  }, []);

  useEffect(() => {
    if (phase === 'done') {
      const c = clicksRef.current;
      onComplete(Math.min(c / threshold, 1), c);
    }
  }, [phase]);

  // Spacebar support
  useEffect(() => {
    const handler = (e) => {
      if (e.code === 'Space' && phase === 'playing') { e.preventDefault(); doClick(); }
      if (e.code === 'Space' && phase === 'ready') { e.preventDefault(); start(); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [phase]);

  const start = () => {
    clicksRef.current = 0;
    setDisplayClicks(0);
    setTimeLeft(config.duration || 7);
    setPhase('playing');
    timerRef.current = setInterval(() => {
      setTimeLeft(prev => {
        if (prev <= 1) {
          clearInterval(timerRef.current);
          setPhase('done');
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  const doClick = () => {
    if (phase !== 'playing') return;
    clicksRef.current++;
    setDisplayClicks(clicksRef.current);
  };

  const progress = Math.min(displayClicks / threshold * 100, 100);

  if (phase === 'ready') {
    return (
      <div style={{ textAlign: 'center', padding: 20 }}>
        <div style={{ fontSize: 56, marginBottom: 12 }}>{game.emoji}</div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)', marginBottom: 6 }}>
          ⚡ {t('在')} <strong>{config.duration}s</strong> {t('内疯狂点击')}
        </div>
        <div style={{ fontSize: 12, color: 'var(--farm-text-3)', marginBottom: 20 }}>
          {t('目标')}: {threshold} {t('次')} · {t('支持空格键')}
        </div>
        <Button theme='solid' size='large' onClick={start} className='farm-btn'
          style={{ fontWeight: 700, fontSize: 16, minWidth: 140 }}>
          ▶ {t('开始')}
        </Button>
      </div>
    );
  }

  if (phase === 'done') {
    const won = displayClicks >= threshold;
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-engine-score-big' style={{ marginBottom: 8 }}>{won ? '🎉' : '😢'}</div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 6 }}>
          {won ? t('太棒了') + '!' : t('差一点') + '!'}
        </div>
        <div style={{ fontSize: 14, color: 'var(--farm-text-2)' }}>
          {t('点击次数')}: <strong>{displayClicks}</strong> / {threshold}
        </div>
      </div>
    );
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 14 }}>
      <div className='farm-engine-countdown' style={{ color: timeLeft <= 3 ? '#ef4444' : 'var(--farm-text-0)' }}>
        {timeLeft}s
      </div>
      <div style={{ width: '100%', height: 14, borderRadius: 7, background: 'var(--farm-surface-alt)', overflow: 'hidden' }}>
        <div style={{
          height: '100%', borderRadius: 7,
          background: progress >= 100 ? 'linear-gradient(90deg, #22c55e, #4ade80)' : 'linear-gradient(90deg, #3b82f6, #8b5cf6)',
          width: `${progress}%`, transition: 'width 0.08s',
        }} />
      </div>
      <div style={{ fontSize: 13, color: 'var(--farm-text-2)', fontVariantNumeric: 'tabular-nums' }}>
        {displayClicks} / {threshold}
      </div>
      <div className='farm-masher-btn'
        onClick={doClick}
        onTouchStart={(e) => { e.preventDefault(); doClick(); }}>
        <span style={{ fontSize: 48, lineHeight: 1 }}>{config.actionEmoji || game.emoji}</span>
      </div>
      <div style={{ fontSize: 11, color: 'var(--farm-text-3)' }}>
        {t('疯狂点击')} / {t('按空格')}
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   🎯 TimingEngine — 精准时机引擎
   指针来回移动，玩家在 Sweet Spot 区域按下停止
   ═══════════════════════════════════════════════════════════════ */
const TimingEngine = ({ config, game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [round, setRound] = useState(0);
  const [scores, setScores] = useState([]);
  const [flashKey, setFlashKey] = useState(0);
  const [flashHit, setFlashHit] = useState(true);
  const [cursorDisplay, setCursorDisplay] = useState(0);
  const posRef = useRef(0);
  const dirRef = useRef(1);
  const animRef = useRef(null);
  const maxRounds = 3;
  const sweetCenter = 50;
  const sweetHalf = (config.sweetSpot || 14) / 2;
  const baseSpeed = config.speed || 3.5;

  useEffect(() => {
    return () => { if (animRef.current) cancelAnimationFrame(animRef.current); };
  }, []);

  useEffect(() => {
    if (phase === 'playing') {
      const spd = baseSpeed * (1 + round * 0.25);
      const animate = () => {
        posRef.current += dirRef.current * spd * 0.45;
        if (posRef.current >= 100) { posRef.current = 100; dirRef.current = -1; }
        if (posRef.current <= 0) { posRef.current = 0; dirRef.current = 1; }
        setCursorDisplay(posRef.current);
        animRef.current = requestAnimationFrame(animate);
      };
      posRef.current = 0;
      dirRef.current = 1;
      animRef.current = requestAnimationFrame(animate);
      return () => { if (animRef.current) cancelAnimationFrame(animRef.current); };
    }
  }, [phase, round]);

  // Spacebar / Enter support
  useEffect(() => {
    const handler = (e) => {
      if ((e.code === 'Space' || e.code === 'Enter') && phase === 'playing') { e.preventDefault(); handleStop(); }
      if ((e.code === 'Space' || e.code === 'Enter') && phase === 'ready') { e.preventDefault(); startGame(); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [phase, round, scores]);

  const startGame = () => {
    setRound(0);
    setScores([]);
    setPhase('playing');
  };

  const handleStop = () => {
    if (phase !== 'playing') return;
    cancelAnimationFrame(animRef.current);
    const dist = Math.abs(posRef.current - sweetCenter);
    let roundScore = 0;
    if (dist <= sweetHalf) roundScore = 100;
    else if (dist <= sweetHalf * 2) roundScore = 60;
    else if (dist <= sweetHalf * 3) roundScore = 30;
    else roundScore = 0;

    const newScores = [...scores, roundScore];
    setScores(newScores);
    setFlashHit(roundScore > 0);
    setFlashKey(k => k + 1);
    const newRound = round + 1;

    if (newRound >= maxRounds) {
      setPhase('done');
      const avg = newScores.reduce((a, b) => a + b, 0) / maxRounds;
      onComplete(avg / 100, Math.round(avg));
    } else {
      setPhase('paused');
      setTimeout(() => { setRound(newRound); setPhase('playing'); }, 900);
    }
  };

  if (phase === 'ready') {
    return (
      <div style={{ textAlign: 'center', padding: 20 }}>
        <div style={{ fontSize: 56, marginBottom: 12 }}>{game.emoji}</div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)', marginBottom: 6 }}>
          🎯 {t('在指针经过绿色区域时按下停止')}
        </div>
        <div style={{ fontSize: 12, color: 'var(--farm-text-3)', marginBottom: 20 }}>
          {maxRounds} {t('轮')} · {t('速度递增')} · {t('支持空格键')}
        </div>
        <Button theme='solid' size='large' onClick={startGame} className='farm-btn'
          style={{ fontWeight: 700, fontSize: 16, minWidth: 140 }}>
          ▶ {t('开始')}
        </Button>
      </div>
    );
  }

  if (phase === 'done') {
    const avg = scores.reduce((a, b) => a + b, 0) / maxRounds;
    const perfect = scores.filter(s => s === 100).length;
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-engine-score-big' style={{ marginBottom: 8 }}>
          {avg >= 80 ? '🎯' : avg >= 40 ? '👍' : '😅'}
        </div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 6 }}>
          {t('精准度')}: {Math.round(avg)}%
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>
          {scores.map((s, i) => `R${i + 1}: ${s}`).join(' · ')}
          {perfect > 0 && ` · 🎯 Perfect ×${perfect}`}
        </div>
      </div>
    );
  }

  // Playing / Paused
  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 16 }}>
      <div style={{ fontSize: 14, fontWeight: 700, color: 'var(--farm-text-0)' }}>
        {t('第')} {round + 1} / {maxRounds} {t('轮')}
        {phase === 'paused' && <span style={{ color: '#4ade80', marginLeft: 8 }}>
          {scores[scores.length - 1] === 100 ? '🎯 Perfect!' : scores[scores.length - 1] > 0 ? '✓ ' + t('不错') : '✗ ' + t('偏了')}
        </span>}
      </div>

      <div className='farm-timing-bar'>
        <div className='farm-timing-sweet' style={{
          left: `${sweetCenter - sweetHalf}%`,
          width: `${sweetHalf * 2}%`,
        }} />
        <div className='farm-timing-cursor' style={{ left: `calc(${cursorDisplay}% - 2px)` }} />
        {flashKey > 0 && (
          <div key={flashKey} className={`farm-timing-flash ${flashHit ? '' : 'miss'}`} />
        )}
      </div>

      {phase === 'playing' && (
        <Button theme='solid' size='large' onClick={handleStop} className='farm-btn'
          style={{ fontWeight: 700, minWidth: 160, fontSize: 15 }}>
          ⏱ {t('停')}!
        </Button>
      )}

      <div style={{ display: 'flex', gap: 6 }}>
        {scores.map((s, i) => (
          <span key={i} className={`farm-pill ${s === 100 ? 'farm-pill-green' : s > 0 ? 'farm-pill-blue' : 'farm-pill-red'}`}>
            R{i + 1}: {s}
          </span>
        ))}
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   🧠 ReactionEngine — 反应/寻找引擎
   4x4 网格，目标随机冒出，限时点击
   ═══════════════════════════════════════════════════════════════ */
const ReactionEngine = ({ config, game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(config.duration || 12);
  const [score, setScore] = useState(0);
  const [misses, setMisses] = useState(0);
  const [cells, setCells] = useState({});
  const [shakeKey, setShakeKey] = useState(0);
  const scoreRef = useRef(0);
  const missRef = useRef(0);
  const timerRef = useRef(null);
  const spawnRef = useRef(null);
  const cleanRef = useRef(null);
  const nextId = useRef(0);
  const gridSize = config.gridSize || 4;
  const totalCells = gridSize * gridSize;

  useEffect(() => {
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      if (spawnRef.current) clearInterval(spawnRef.current);
      if (cleanRef.current) clearInterval(cleanRef.current);
    };
  }, []);

  useEffect(() => {
    if (phase === 'done') {
      const finalScore = Math.max(0, scoreRef.current);
      const maxPossible = Math.floor((config.duration || 12) * 1000 / (config.spawnMs || 900)) * 0.7;
      onComplete(Math.min(finalScore / Math.max(maxPossible, 1), 1), finalScore);
    }
  }, [phase]);

  // Spacebar to start
  useEffect(() => {
    const handler = (e) => {
      if (e.code === 'Space' && phase === 'ready') { e.preventDefault(); start(); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [phase]);

  const start = () => {
    scoreRef.current = 0;
    missRef.current = 0;
    setScore(0);
    setMisses(0);
    setCells({});
    setTimeLeft(config.duration || 12);
    setPhase('playing');

    timerRef.current = setInterval(() => {
      setTimeLeft(prev => {
        if (prev <= 1) {
          clearInterval(timerRef.current);
          clearInterval(spawnRef.current);
          clearInterval(cleanRef.current);
          setPhase('done');
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    spawnRef.current = setInterval(() => {
      setCells(prev => {
        const now = Date.now();
        const used = new Set(Object.values(prev).map(v => v.cellIdx));
        const free = [];
        for (let i = 0; i < totalCells; i++) { if (!used.has(i)) free.push(i); }
        if (free.length === 0) return prev;
        const cellIdx = free[Math.floor(Math.random() * free.length)];
        const isBomb = Math.random() < 0.2;
        const id = nextId.current++;
        return {
          ...prev,
          [id]: {
            id, cellIdx,
            type: isBomb ? 'bomb' : 'target',
            emoji: isBomb ? (config.bombEmoji || '💣') : (config.targetEmoji || '⭐'),
            spawnTime: now,
          },
        };
      });
    }, config.spawnMs || 900);

    cleanRef.current = setInterval(() => {
      const now = Date.now();
      setCells(prev => {
        const next = {};
        for (const [k, v] of Object.entries(prev)) {
          if (now - v.spawnTime < 1300) next[k] = v;
        }
        return next;
      });
    }, 250);
  };

  const handleCellClick = (cellIdx) => {
    if (phase !== 'playing') return;
    const item = Object.values(cells).find(v => v.cellIdx === cellIdx);
    if (!item) return;
    if (item.type === 'target') {
      scoreRef.current++;
      setScore(scoreRef.current);
    } else {
      missRef.current++;
      setMisses(missRef.current);
      setShakeKey(k => k + 1);
    }
    setCells(prev => {
      const next = { ...prev };
      delete next[item.id];
      return next;
    });
  };

  if (phase === 'ready') {
    return (
      <div style={{ textAlign: 'center', padding: 20 }}>
        <div style={{ fontSize: 56, marginBottom: 12 }}>{game.emoji}</div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)', marginBottom: 6 }}>
          🧠 {t('点击冒出的')} {config.targetEmoji || '⭐'} · {t('避开')} {config.bombEmoji || '💣'}
        </div>
        <div style={{ fontSize: 12, color: 'var(--farm-text-3)', marginBottom: 20 }}>
          {config.duration}s · {gridSize}×{gridSize} {t('网格')}
        </div>
        <Button theme='solid' size='large' onClick={start} className='farm-btn'
          style={{ fontWeight: 700, fontSize: 16, minWidth: 140 }}>
          ▶ {t('开始')}
        </Button>
      </div>
    );
  }

  if (phase === 'done') {
    const finalScore = Math.max(0, score);
    return (
      <div className='farm-game-result' style={{ width: '100%', textAlign: 'center' }}>
        <div className='farm-engine-score-big' style={{ marginBottom: 8 }}>
          {finalScore >= 8 ? '🏆' : finalScore >= 4 ? '👏' : '🤔'}
        </div>
        <div style={{ fontSize: 18, fontWeight: 700, marginBottom: 6 }}>
          {t('得分')}: {finalScore}
        </div>
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>
          ✅ {t('命中')}: {score} · ❌ {t('踩雷')}: {misses}
        </div>
      </div>
    );
  }

  // Playing
  const cellMap = {};
  for (const item of Object.values(cells)) {
    cellMap[item.cellIdx] = item;
  }

  return (
    <div key={shakeKey > 0 ? shakeKey : undefined}
      className={shakeKey > 0 ? 'farm-engine-shake' : ''}
      style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%', maxWidth: 300 }}>
        <span className='farm-engine-countdown' style={{ color: timeLeft <= 3 ? '#ef4444' : 'var(--farm-text-0)', fontSize: 24 }}>
          {timeLeft}s
        </span>
        <span style={{ fontSize: 14, fontWeight: 700, color: '#4ade80' }}>
          ✅ {score}
        </span>
        <span style={{ fontSize: 14, fontWeight: 700, color: '#ef4444' }}>
          ❌ {misses}
        </span>
      </div>

      <div className='farm-reaction-grid' style={{ gridTemplateColumns: `repeat(${gridSize}, 1fr)` }}>
        {Array.from({ length: totalCells }, (_, i) => {
          const item = cellMap[i];
          return (
            <div key={i} className='farm-reaction-cell'
              onClick={() => handleCellClick(i)}>
              {item && (
                <span className='farm-rc-pop'>{item.emoji}</span>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   EngineModal — 引擎模态弹窗
   包裹引擎 + 游戏结束后调用 API
   ═══════════════════════════════════════════════════════════════ */
const EngineModal = ({ game, onClose, onPlayApi, gameLoading, t }) => {
  const [phase, setPhase] = useState('engine');
  const [gameScore, setGameScore] = useState(null);
  const [apiResult, setApiResult] = useState(null);
  const [playCount, setPlayCount] = useState(0);

  const engineCfg = GAME_ENGINE_MAP[game.key] || { engine: 'masher', duration: 7, threshold: 35, actionEmoji: game.emoji };

  const handleEngineComplete = async (normalized, raw) => {
    setGameScore({ normalized, raw });
    setPhase('calling');
    const result = await onPlayApi(game.key);
    setApiResult(result);
    setPhase('result');
  };

  const handleReplay = () => {
    setPhase('engine');
    setGameScore(null);
    setApiResult(null);
    setPlayCount(c => c + 1);
  };

  return (
    <div className='farm-game-overlay' onClick={(e) => { if (e.target === e.currentTarget && phase !== 'calling') onClose(); }}>
      <div className='farm-game-modal'>
        <div className='farm-game-modal-header'>
          <span className='farm-game-modal-emoji'>{game.emoji}</span>
          <div>
            <div className='farm-game-modal-title'>{game.name}</div>
            <div className='farm-game-modal-sub'>{game.desc} · ${game.price.toFixed(2)}</div>
          </div>
          {phase !== 'calling' && (
            <div className='farm-game-modal-close' onClick={onClose}>✕</div>
          )}
        </div>

        <div className='farm-game-modal-body'>
          {phase === 'engine' && (
            <>
              {engineCfg.engine === 'masher' && (
                <MasherEngine key={playCount} config={engineCfg} game={game} onComplete={handleEngineComplete} t={t} />
              )}
              {engineCfg.engine === 'timing' && (
                <TimingEngine key={playCount} config={engineCfg} game={game} onComplete={handleEngineComplete} t={t} />
              )}
              {engineCfg.engine === 'reaction' && (
                <ReactionEngine key={playCount} config={engineCfg} game={game} onComplete={handleEngineComplete} t={t} />
              )}
            </>
          )}

          {phase === 'calling' && (
            <div style={{ padding: 40, textAlign: 'center' }}>
              <Spin size='large' />
              <div style={{ marginTop: 12, fontSize: 13, color: 'var(--farm-text-2)' }}>{t('结算中')}...</div>
            </div>
          )}

          {phase === 'result' && (
            <div className='farm-game-result' style={{ width: '100%' }}>
              {gameScore && (
                <div style={{ textAlign: 'center', marginBottom: 12 }}>
                  <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--farm-text-2)', marginBottom: 4 }}>
                    🎮 {t('你的表现')}: <strong style={{ color: 'var(--farm-text-0)' }}>{gameScore.raw}</strong>
                  </div>
                </div>
              )}
              {apiResult && (
                <>
                  <div className='farm-game-result-text'>{apiResult.result_text}</div>
                  <div className='farm-game-result-pills'>
                    <span className='farm-pill farm-pill-blue'>{t('下注')}: ${apiResult.bet.toFixed(2)}</span>
                    <span className={`farm-pill ${apiResult.net >= 0 ? 'farm-pill-green' : 'farm-pill-red'}`}>
                      {apiResult.net >= 0 ? '+' : ''}{apiResult.net.toFixed(2)}
                    </span>
                    <span className='farm-pill farm-pill-purple'>{apiResult.multi}x</span>
                  </div>
                </>
              )}
              {!apiResult && (
                <div style={{ textAlign: 'center', color: 'var(--farm-text-2)', fontSize: 13 }}>
                  {t('结算失败，请重试')}
                </div>
              )}
            </div>
          )}
        </div>

        {phase === 'result' && (
          <div className='farm-game-modal-footer'>
            <Button theme='light' onClick={onClose} className='farm-btn'>{t('关闭')}</Button>
            <Button theme='solid' onClick={handleReplay} loading={gameLoading} className='farm-btn'
              style={{ fontWeight: 700 }}>
              🔄 {t('再来一次')} (${game.price.toFixed(2)})
            </Button>
          </div>
        )}
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   GamesPage — 农场游乐场主组件
   ═══════════════════════════════════════════════════════════════ */
const GamesPage = ({ loadFarm, t }) => {
  const [gameLoading, setGameLoading] = useState(false);
  const [wheelResult, setWheelResult] = useState(null);
  const [scratchResult, setScratchResult] = useState(null);
  const [history, setHistory] = useState([]);
  const [miniGames, setMiniGames] = useState([]);
  const [modalGame, setModalGame] = useState(null);

  const loadHistory = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/game/history');
      if (res.success) setHistory(res.data || []);
    } catch (err) { /* ignore */ }
  }, []);

  const loadMiniGames = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/game/list');
      if (res.success) setMiniGames(res.data || []);
    } catch (err) { /* ignore */ }
  }, []);

  useEffect(() => { loadHistory(); loadMiniGames(); }, [loadHistory, loadMiniGames]);

  const spinWheel = async () => {
    setGameLoading(true);
    setWheelResult(null);
    try {
      const { data: res } = await API.post('/api/farm/game/wheel');
      if (res.success) { setWheelResult(res.data); loadFarm(); loadHistory(); return res.data; }
      else { showError(res.message); return null; }
    } catch (err) { showError(t('操作失败')); return null; }
    finally { setGameLoading(false); }
  };

  const doScratch = async () => {
    setGameLoading(true);
    setScratchResult(null);
    try {
      const { data: res } = await API.post('/api/farm/game/scratch');
      if (res.success) { setScratchResult(res.data); loadFarm(); loadHistory(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
    finally { setGameLoading(false); }
  };

  // Called by EngineModal after game engine completes
  const playApi = async (gameKey) => {
    setGameLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/game/play', { game_key: gameKey });
      if (res.success) { loadFarm(); loadHistory(); return res.data; }
      else { showError(res.message); return null; }
    } catch (err) { showError(t('操作失败')); return null; }
    finally { setGameLoading(false); }
  };

  const openGame = (game) => {
    if (gameLoading) return;
    setModalGame(game);
  };

  const closeModal = () => { setModalGame(null); };

  return (
    <div>
      {/* ═══ Wheel & Scratch ═══ */}
      <div className='farm-grid farm-grid-2'>
        <div className='farm-card' style={{ marginBottom: 0 }}>
          <div className='farm-section-title'>🎡 {t('幸运转盘')}</div>
          <SpinningWheel onSpin={spinWheel} spinning={gameLoading}
            result={wheelResult} gameLoading={gameLoading} t={t} />
        </div>
        <div className='farm-card' style={{ marginBottom: 0 }}>
          <div className='farm-section-title'>🎰 {t('刮刮卡')}</div>
          <ScratchCard onScratch={doScratch} result={scratchResult}
            gameLoading={gameLoading} t={t} />
        </div>
      </div>

      {/* ═══ Mini-Games Grid ═══ */}
      {miniGames.length > 0 && (
        <div className='farm-card' style={{ marginTop: 14 }}>
          <div className='farm-section-title'>🎮 {t('农场小游戏')} ({miniGames.length})</div>
          <div className='farm-game-grid'>
            {miniGames.map((g) => {
              const eng = GAME_ENGINE_MAP[g.key];
              const badge = eng ? (eng.engine === 'masher' ? '⚡' : eng.engine === 'timing' ? '🎯' : '🧠') : '🎮';
              return (
                <div key={g.key} className='farm-game-tile'
                  onClick={() => openGame(g)}
                  style={{ opacity: gameLoading ? 0.6 : 1, cursor: gameLoading ? 'not-allowed' : 'pointer' }}>
                  <span className='farm-game-tile-emoji'>{g.emoji}</span>
                  <span className='farm-game-tile-name'>{g.name}</span>
                  <span className='farm-game-tile-price'>{badge} ${g.price.toFixed(2)}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* ═══ History ═══ */}
      {history.length > 0 && (
        <div className='farm-card' style={{ marginTop: 14 }}>
          <div className='farm-section-title'>📜 {t('游戏记录')}</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {history.slice(0, 15).map((h, i) => (
              <div key={i} className='farm-row' style={{ marginBottom: 0, padding: '6px 10px' }}>
                <Text size='small'>{h.game_type === 'wheel' ? '🎡' : h.game_type === 'scratch' ? '🎰' : '🎮'}</Text>
                <Text size='small' style={{ flex: 1 }}>{t('下注')} ${h.bet.toFixed(2)} → ${h.win.toFixed(2)}</Text>
                <Text size='small' strong style={{ color: h.net >= 0 ? '#22c55e' : '#ef4444' }}>
                  {h.net >= 0 ? '+' : ''}{h.net.toFixed(2)}
                </Text>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ═══ Engine Modal ═══ */}
      {modalGame && (
        <EngineModal game={modalGame} onClose={closeModal}
          onPlayApi={playApi} gameLoading={gameLoading} t={t} />
      )}
    </div>
  );
};

export default GamesPage;
