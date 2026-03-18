import React, { useCallback, useEffect, useState, useRef } from 'react';
import { Button, Spin, Typography } from '@douyinfe/semi-ui';
import { API, showError } from './utils';
import { GAME_REGISTRY, EngineModal } from './GameEngineModal';

const { Text } = Typography;

/* ═══════════════════════════════════════════════════════════════
   WHEEL_SECTORS — 转盘扇区颜色
   ═══════════════════════════════════════════════════════════════ */
const WHEEL_COLORS = [
  '#b84233', '#5a8fb4', '#4a7c3f', '#c8922a',
  '#8a6cb0', '#5a8fb4', '#a0845e', '#6fa85e',
];

/* ═══════════════════════════════════════════════════════════════
   SpinningWheel — 真实旋转转盘（CSS transform）
   ═══════════════════════════════════════════════════════════════ */
const WHEEL_PRIZES = [
  { symbol: '🎁', label: '$0', color: '#b84233' },
  { symbol: '💰', label: '$0.50', color: '#5a8fb4' },
  { symbol: '🍀', label: '$1', color: '#4a7c3f' },
  { symbol: '⭐', label: '$1.50', color: '#c8922a' },
  { symbol: '🎯', label: '$2', color: '#8a6cb0' },
  { symbol: '🏆', label: '$3', color: '#5a8fb4' },
  { symbol: '💎', label: '$5', color: '#a0845e' },
  { symbol: '🌟', label: '$10', color: '#4a7c3f' },
];

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
    const prizeIdx = data.sector_index ?? Math.floor(Math.random() * sectorCount);
    // Pointer is at top (0°). Sector 0 starts at 0°.
    // To land on prizeIdx, rotate so that sector's center aligns with top.
    // Keep rotation continuous by solving against current angle modulo 360.
    const targetSectorCenter = prizeIdx * sectorAngle + sectorAngle / 2;
    const desiredAngle = (360 - targetSectorCenter + 360) % 360;
    const currentAngle = ((rotation % 360) + 360) % 360;
    let delta = desiredAngle - currentAngle;
    if (delta <= 0) delta += 360; // always spin forward
    const extraSpins = 5 * 360; // 5 full rotations
    const newRotation = rotation + extraSpins + delta;
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
    grad.addColorStop(0, '#6b5d4f');
    grad.addColorStop(1, '#8a7a6a');
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
    ctx.arc(pos.x, pos.y, 30, 0, Math.PI * 2);
    ctx.fill();
    // Check percentage scratched
    const imgData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    let cleared = 0;
    for (let i = 3; i < imgData.data.length; i += 4) {
      if (imgData.data[i] === 0) cleared++;
    }
    const pct = cleared / (imgData.data.length / 4);
    if (pct > 0.35) {
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
              <span style={{ fontSize: 12, color: result.net >= 0 ? 'var(--farm-leaf)' : 'var(--farm-danger)', fontWeight: 700 }}>
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
  const playApi = async (gameKey, score) => {
    setGameLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/game/play', { game_key: gameKey, score: score ?? 0 });
      if (res.success) { loadFarm(); loadHistory(); return res.data; }
      else { showError(res.message); return null; }
    } catch (err) { showError(t('操作失败')); return null; }
    finally { setGameLoading(false); }
  };

  const openGame = (game) => {
    if (gameLoading) return;
    const entry = GAME_REGISTRY[game.key];
    if (!entry?.get?.()) {
      showError(t('该游戏尚未上线，敬请期待'));
      return;
    }
    setModalGame(game);
  };

  const closeModal = () => { setModalGame(null); };

  return (
    <div>
      {/* ═══ Wheel / Prize Pool / Scratch — 3-column grid ═══ */}
      <div className='farm-games-top-grid'>
        {/* Col 1: 幸运转盘 */}
        <div className='farm-card farm-games-col'>
          <div className='farm-section-title'>🎡 {t('幸运转盘')}</div>
          <SpinningWheel onSpin={spinWheel} spinning={gameLoading}
            result={wheelResult} gameLoading={gameLoading} t={t} />
        </div>

        {/* Col 2: 奖池一览 */}
        <div className='farm-card farm-games-col farm-prize-pool-card'>
          <div className='farm-section-title'>🏆 {t('奖池一览')}</div>
          <div className='farm-prize-pool-list'>
            {WHEEL_PRIZES.map((p, i) => (
              <div key={i} className='farm-prize-pool-row'>
                <span className='farm-prize-pool-symbol'>{p.symbol}</span>
                <span className='farm-prize-pool-label' style={{ color: p.color }}>{p.label}</span>
                <span className='farm-prize-pool-bar' style={{ background: p.color, width: `${Math.max(12, parseFloat(p.label.replace('$', '')) * 10)}%` }} />
              </div>
            ))}
          </div>
        </div>

        {/* Col 3: 刮刮卡 */}
        <div className='farm-card farm-games-col'>
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
              const reg = GAME_REGISTRY[g.key];
              const online = !!reg?.get?.();
              return (
                <div key={g.key} className='farm-game-tile'
                  onClick={() => openGame(g)}
                  style={{ opacity: gameLoading ? 0.6 : online ? 1 : 0.45, cursor: gameLoading ? 'not-allowed' : 'pointer' }}>
                  <span className='farm-game-tile-emoji'>{g.emoji}</span>
                  <span className='farm-game-tile-name'>{g.name}</span>
                  <span className='farm-game-tile-price'>${g.price.toFixed(2)}</span>
                  {online && reg && <span className={`farm-game-tile-badge ${reg.cat}`}>{reg.catLabel}</span>}
                  {!online && <span className='farm-game-tile-badge' style={{ background: 'rgba(255,255,255,0.08)', color: '#888' }}>🚧{t('未上线')}</span>}
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
                <Text size='small' strong style={{ color: h.net >= 0 ? 'var(--farm-leaf)' : 'var(--farm-danger)' }}>
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
