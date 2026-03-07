import React, { useState } from 'react';
import { Button, Spin } from '@douyinfe/semi-ui';
import { HorseRaceGame, TugOfWarGame, ClickBlitzGame, RhythmKeysGame, CircleDrawGame } from './GamesMashing';
import { FishingBarGame, PowerChopGame, LassoGame, AnglePowerGame, WaterCatchGame } from './GamesTiming';

/* ═══════════════════════════════════════════════════════════════
   PlaceholderGame — 尚未实现的游戏占位符
   ═══════════════════════════════════════════════════════════════ */
const PlaceholderGame = ({ game, t }) => {
  return (
    <div className='farm-gc-ready'>
      <div className='farm-gc-ready-emoji'>{game.emoji}</div>
      <div className='farm-gc-ready-desc'>🚧 {t('即将推出')}...</div>
      <div className='farm-gc-ready-hint'>{game.name} — {game.desc}</div>
      <div style={{ fontSize: 13, color: 'var(--farm-text-2)', marginTop: 8 }}>
        {t('该游戏尚未上线，敬请期待')}
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════════════
   GAME_REGISTRY — 30 个游戏 Key → 组件 + 分类
   动态导入对应引擎组件
   ═══════════════════════════════════════════════════════════════ */
const GAME_REGISTRY = {
  // ═══ 第一类: ⚡ 极限手速与耐力 ═══
  horserace:  { get: () => HorseRaceGame,  cat: 'cat1', catLabel: '⚡手速' },
  woodchop:   { get: () => TugOfWarGame,    cat: 'cat1', catLabel: '⚡手速' },
  weed:       { get: () => ClickBlitzGame,  cat: 'cat1', catLabel: '⚡手速' },
  milking:    { get: () => RhythmKeysGame,  cat: 'cat1', catLabel: '⚡手速' },
  thresh:     { get: () => CircleDrawGame,  cat: 'cat1', catLabel: '⚡手速' },
  // ═══ 第二类: 🎯 精准时机与预判 ═══
  fishcomp:   { get: () => FishingBarGame,   cat: 'cat2', catLabel: '🎯时机' },
  harvest:    { get: () => PowerChopGame,    cat: 'cat2', catLabel: '🎯时机' },
  lasso:      { get: () => LassoGame,        cat: 'cat2', catLabel: '🎯时机' },
  pullcarrot: { get: () => AnglePowerGame,   cat: 'cat2', catLabel: '🎯时机' },
  seedling:   { get: () => WaterCatchGame,   cat: 'cat2', catLabel: '🎯时机' },
  // ═══ 第三类: 👀 动态反应与捕捉 (待实现) ═══
  egghunt:    { get: () => null, cat: 'cat3', catLabel: '👀反应' },
  bugcatch:   { get: () => null, cat: 'cat3', catLabel: '👀反应' },
  duckherd:   { get: () => null, cat: 'cat3', catLabel: '👀反应' },
  fruitpick:  { get: () => null, cat: 'cat3', catLabel: '👀反应' },
  foxhunt:    { get: () => null, cat: 'cat3', catLabel: '👀反应' },
  // ═══ 第四类: 🧠 记忆与益智 (待实现) ═══
  sheepcount: { get: () => null, cat: 'cat4', catLabel: '🧠益智' },
  mushroom:   { get: () => null, cat: 'cat4', catLabel: '🧠益智' },
  scarecrow:  { get: () => null, cat: 'cat4', catLabel: '🧠益智' },
  pumpkin:    { get: () => null, cat: 'cat4', catLabel: '🧠益智' },
  produce:    { get: () => null, cat: 'cat4', catLabel: '🧠益智' },
  // ═══ 第五类: ⚖️ 平衡与控制 (待实现) ═══
  cornrace:   { get: () => null, cat: 'cat5', catLabel: '⚖️平衡' },
  pigchase:   { get: () => null, cat: 'cat5', catLabel: '⚖️平衡' },
  grape:      { get: () => null, cat: 'cat5', catLabel: '⚖️平衡' },
  beekeep:    { get: () => null, cat: 'cat5', catLabel: '⚖️平衡' },
  hatchegg:   { get: () => null, cat: 'cat5', catLabel: '⚖️平衡' },
  // ═══ 第六类: 🎰 物理模拟 (待实现) ═══
  rooster:    { get: () => null, cat: 'cat6', catLabel: '🎰物理' },
  sunflower:  { get: () => null, cat: 'cat6', catLabel: '🎰物理' },
  tame:       { get: () => null, cat: 'cat6', catLabel: '🎰物理' },
  weather:    { get: () => null, cat: 'cat6', catLabel: '🎰物理' },
  sheepdog:   { get: () => null, cat: 'cat6', catLabel: '🎰物理' },
};

/* ═══════════════════════════════════════════════════════════════
   EngineModal — 通用游戏引擎模态弹窗
   生命周期: engine(玩游戏) → calling(API结算) → result(显示结果)
   ═══════════════════════════════════════════════════════════════ */
const EngineModal = ({ game, onClose, onPlayApi, gameLoading, t }) => {
  const [phase, setPhase] = useState('engine');
  const [gameScore, setGameScore] = useState(null);
  const [apiResult, setApiResult] = useState(null);
  const [playCount, setPlayCount] = useState(0);

  const entry = GAME_REGISTRY[game.key];
  const GameComponent = entry?.get?.() || null;
  const FinalComponent = GameComponent || PlaceholderGame;

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
    <div className='farm-game-overlay'
      onClick={(e) => { if (e.target === e.currentTarget && phase !== 'calling') onClose(); }}>
      <div className='farm-game-modal'>
        {/* Header */}
        <div className='farm-game-modal-header'>
          <span className='farm-game-modal-emoji'>{game.emoji}</span>
          <div>
            <div className='farm-game-modal-title'>{game.name}</div>
            <div className='farm-game-modal-sub'>
              {game.desc} · ${game.price.toFixed(2)}
              {entry && (
                <span className={`farm-game-tile-badge ${entry.cat}`}
                  style={{ marginLeft: 6, verticalAlign: 'middle' }}>
                  {entry.catLabel}
                </span>
              )}
            </div>
          </div>
          {phase !== 'calling' && (
            <div className='farm-game-modal-close' onClick={onClose}>✕</div>
          )}
        </div>

        {/* Body */}
        <div className='farm-game-modal-body'>
          {phase === 'engine' && (
            <FinalComponent
              key={playCount}
              game={game}
              onComplete={handleEngineComplete}
              t={t}
            />
          )}

          {phase === 'calling' && (
            <div style={{ padding: 40, textAlign: 'center' }}>
              <Spin size='large' />
              <div style={{ marginTop: 12, fontSize: 13, color: 'var(--farm-text-2)' }}>
                {t('结算中')}...
              </div>
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

        {/* Footer */}
        {phase === 'result' && (
          <div className='farm-game-modal-footer'>
            <Button theme='light' onClick={onClose} className='farm-btn'>
              {t('关闭')}
            </Button>
            <Button theme='solid' onClick={handleReplay} loading={gameLoading}
              className='farm-btn' style={{ fontWeight: 700 }}>
              🔄 {t('再来一次')} (${game.price.toFixed(2)})
            </Button>
          </div>
        )}
      </div>
    </div>
  );
};

export { GAME_REGISTRY, EngineModal };
