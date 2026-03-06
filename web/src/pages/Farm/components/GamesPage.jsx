import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Banner, Typography } from '@douyinfe/semi-ui';
import { API, showError } from './utils';

const { Text } = Typography;

const GamesPage = ({ loadFarm, t }) => {
  const [gameLoading, setGameLoading] = useState(false);
  const [wheelResult, setWheelResult] = useState(null);
  const [scratchResult, setScratchResult] = useState(null);
  const [scratchRevealed, setScratchRevealed] = useState(false);
  const [history, setHistory] = useState([]);
  const [miniGames, setMiniGames] = useState([]);
  const [miniResult, setMiniResult] = useState(null);

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
    setMiniResult(null);
    try {
      const { data: res } = await API.post('/api/farm/game/wheel');
      if (res.success) { setWheelResult(res.data); loadFarm(); loadHistory(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
    finally { setGameLoading(false); }
  };

  const doScratch = async () => {
    setGameLoading(true);
    setScratchResult(null);
    setScratchRevealed(false);
    setMiniResult(null);
    try {
      const { data: res } = await API.post('/api/farm/game/scratch');
      if (res.success) { setScratchResult(res.data); loadFarm(); loadHistory(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
    finally { setGameLoading(false); }
  };

  const playMiniGame = async (gameKey) => {
    setGameLoading(true);
    setMiniResult(null);
    setWheelResult(null);
    setScratchResult(null);
    try {
      const { data: res } = await API.post('/api/farm/game/play', { game_key: gameKey });
      if (res.success) { setMiniResult(res.data); loadFarm(); loadHistory(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
    finally { setGameLoading(false); }
  };

  return (
    <div>
      {/* Wheel & Scratch */}
      <div className='farm-grid farm-grid-2'>
        <div className='farm-card' style={{ marginBottom: 0 }}>
          <div className='farm-section-title'>🎡 {t('幸运转盘')} ($1/{t('次')})</div>
          <Button theme='solid' onClick={spinWheel} loading={gameLoading} className='farm-btn' style={{ marginBottom: 8 }}>
            🎰 {t('转一次')}
          </Button>
          {wheelResult && (
            <Banner type={wheelResult.net >= 0 ? 'success' : 'warning'} closeIcon={null} style={{ borderRadius: 10 }}
              description={<span>{t('中奖')}: <strong>{wheelResult.prize_label}</strong> ({wheelResult.net >= 0 ? '+' : ''}{wheelResult.net.toFixed(2)})</span>} />
          )}
        </div>

        <div className='farm-card' style={{ marginBottom: 0 }}>
          <div className='farm-section-title'>🎰 {t('刮刮卡')} ($0.50/{t('次')})</div>
          <Button theme='solid' onClick={doScratch} loading={gameLoading} className='farm-btn' style={{ marginBottom: 8 }}>
            🃏 {t('刮一张')}
          </Button>
          {scratchResult && (
            <div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 4, marginBottom: 8, maxWidth: 180 }}>
                {scratchResult.grid.flat().map((sym, i) => (
                  <div key={i} onClick={() => setScratchRevealed(true)} style={{
                    width: 50, height: 50, borderRadius: 10, display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: 24, cursor: 'pointer',
                    background: scratchRevealed ? 'var(--farm-glass-bg)' : 'linear-gradient(135deg, #6366f1, #8b5cf6)',
                    border: '1px solid var(--farm-glass-border)', transition: 'all 0.3s',
                  }}>
                    {scratchRevealed ? sym : '?'}
                  </div>
                ))}
              </div>
              {scratchRevealed && (
                <Banner type={scratchResult.net >= 0 ? 'success' : 'warning'} closeIcon={null} style={{ borderRadius: 10 }}
                  description={<span>{scratchResult.win_symbol}×3 → <strong>{scratchResult.prize_label}</strong></span>} />
              )}
            </div>
          )}
        </div>
      </div>

      {/* Mini-Game Result */}
      {miniResult && (
        <div className='farm-card farm-card-glow-purple' style={{ marginTop: 14 }}>
          <div className='farm-section-title' style={{ fontSize: 16 }}>
            {miniResult.game_emoji} {miniResult.game_name}
          </div>
          <div style={{ whiteSpace: 'pre-wrap', marginBottom: 12, padding: '10px 14px', borderRadius: 10,
            background: 'var(--farm-glass-bg)', fontSize: 13, lineHeight: 1.6 }}>
            {miniResult.result_text}
          </div>
          <div style={{ display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
            <div className='farm-pill farm-pill-blue'>{t('下注')}: ${miniResult.bet.toFixed(2)}</div>
            <div className={`farm-pill ${miniResult.net >= 0 ? 'farm-pill-green' : 'farm-pill-red'}`}>
              {t('收益')}: {miniResult.net >= 0 ? '+' : ''}{miniResult.net.toFixed(2)}
            </div>
            <div className='farm-pill farm-pill-purple'>{miniResult.multi}x</div>
            <Button size='small' theme='solid' onClick={() => playMiniGame(miniResult.game_key)}
              loading={gameLoading} className='farm-btn'>
              🔄 {t('再来一次')}
            </Button>
          </div>
        </div>
      )}

      {/* All Mini-Games */}
      {miniGames.length > 0 && (
        <div className='farm-card' style={{ marginTop: 14 }}>
          <div className='farm-section-title'>🎮 {t('农场小游戏')} ({miniGames.length})</div>
          <div className='farm-grid farm-grid-4'>
            {miniGames.map((g) => (
              <div key={g.key} className='farm-item-card'
                onClick={() => !gameLoading && playMiniGame(g.key)}
                style={{ opacity: gameLoading ? 0.6 : 1, cursor: gameLoading ? 'not-allowed' : 'pointer' }}>
                <span style={{ fontSize: 28, display: 'block', marginBottom: 4 }}>{g.emoji}</span>
                <Text strong size='small' style={{ display: 'block' }}>{g.name}</Text>
                <Text type='tertiary' size='small' style={{ display: 'block', fontSize: 11 }}>{g.desc}</Text>
                <Text type='success' size='small' style={{ display: 'block', marginTop: 2 }}>${g.price.toFixed(2)}</Text>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* History */}
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
    </div>
  );
};

export default GamesPage;
