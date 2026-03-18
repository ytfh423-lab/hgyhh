import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Banner, Button, Progress, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { Fish } from 'lucide-react';
import { API, showError, showSuccess } from './utils';

const { Text } = Typography;

const rarityColors = {
  普通: 'grey',
  优良: 'green',
  稀有: 'blue',
  史诗: 'purple',
  传说: 'orange',
};

const GoldenDragonEffect = ({ show, onDone }) => {
  useEffect(() => {
    if (!show) return;
    const timer = setTimeout(onDone, 4000);
    return () => clearTimeout(timer);
  }, [show, onDone]);

  if (!show) return null;
  return (
    <div className='fish-golden-dragon-overlay' onClick={onDone}>
      <div className='fish-golden-dragon-content'>
        <div className='fish-golden-dragon-glow' />
        <div className='fish-golden-dragon-emoji'>🐉</div>
        <div className='fish-golden-dragon-title'>金 龙 鱼</div>
        <div className='fish-golden-dragon-subtitle'>传说级 · 极其稀有</div>
        <div className='fish-golden-dragon-sparkles'>
          {Array.from({ length: 20 }).map((_, i) => (
            <span key={i} className='fish-golden-sparkle' style={{
              left: `${Math.random() * 100}%`,
              top: `${Math.random() * 100}%`,
              animationDelay: `${Math.random() * 2}s`,
              fontSize: `${10 + Math.random() * 14}px`,
            }}>✨</span>
          ))}
        </div>
      </div>
    </div>
  );
};

const FishPage = ({ loadFarm, t }) => {
  const [fishData, setFishData] = useState(null);
  const [fishLoading, setFishLoading] = useState(false);
  const [lastCatch, setLastCatch] = useState(null);
  const [cooldown, setCooldown] = useState(0);
  const tickRef = useRef(null);
  const [showGoldenDragon, setShowGoldenDragon] = useState(false);

  const loadFish = useCallback(async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/fish');
      if (res.success) {
        setFishData(res.data);
        setCooldown(res.data.cooldown || 0);
      } else {
        showError(res.message || t('加载失败'));
      }
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setFishLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadFish();
  }, [loadFish]);

  useEffect(() => {
    if (tickRef.current) {
      clearInterval(tickRef.current);
    }
    if (cooldown <= 0) {
      return;
    }
    tickRef.current = setInterval(() => {
      setCooldown((prev) => Math.max(0, prev - 1));
    }, 1000);
    return () => {
      if (tickRef.current) {
        clearInterval(tickRef.current);
      }
    };
  }, [cooldown > 0]);

  const doFish = async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/fish');
      if (res.success) {
        setLastCatch(res.data);
        if (res.data.special_effect === 'golden_dragon') {
          setShowGoldenDragon(true);
        }
        showSuccess(res.message);
        loadFish();
      } else {
        showError(res.message || t('操作失败'));
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setFishLoading(false);
    }
  };

  const doSellAll = async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/fish/sell');
      if (res.success) {
        showSuccess(res.message);
        setLastCatch(null);
        loadFish();
        loadFarm();
      } else {
        showError(res.message || t('操作失败'));
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setFishLoading(false);
    }
  };

  const doStoreAll = async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/fish/store');
      if (res.success) {
        showSuccess(res.message);
        setLastCatch(null);
        loadFish();
        loadFarm();
      } else {
        showError(res.message || t('操作失败'));
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setFishLoading(false);
    }
  };

  if (fishLoading && !fishData) {
    return (
      <div style={{ textAlign: 'center', padding: 40 }}>
        <Spin size='large' />
      </div>
    );
  }
  if (!fishData) return null;

  const dailyCount = fishData.daily_count || 0;
  const dailyIncome = fishData.daily_income || 0;
  const dailyMaxIncome = fishData.daily_max_income || 200;
  const fatigueActive = fishData.fatigue_active || false;
  const fatigueThreshold = fishData.fatigue_threshold || 30;
  const fatigueDecay = fishData.fatigue_decay || 50;

  const capEnabled = fishData.cap_enabled || false;
  const dailyIncomeCap = fishData.daily_income_cap || 100;
  const overCap = fishData.over_cap || false;
  const totalBaitCount = (fishData.bait_count || 0) + (fishData.premium_bait_count || 0);
  const incomeCapPct =
    capEnabled && dailyIncomeCap > 0
      ? Math.min(100, Math.round((dailyIncome / dailyIncomeCap) * 100))
      : 0;

  let btnText = t('开始钓鱼');
  let btnDisabled = false;
  if (capEnabled) {
    if (overCap) {
      btnText = t('今日收益已达上限');
      btnDisabled = true;
    } else if (cooldown > 0) {
      btnText = `⏱️ ${cooldown}s`;
      btnDisabled = true;
    } else if (totalBaitCount === 0) {
      btnText = t('没有鱼饵');
      btnDisabled = true;
    }
  } else if (dailyIncome >= dailyMaxIncome) {
    btnText = t('今日收益已达上限');
    btnDisabled = true;
  } else if (cooldown > 0) {
    btnText = `⏱️ ${cooldown}s`;
    btnDisabled = true;
  } else if (totalBaitCount === 0) {
    btnText = t('没有鱼饵');
    btnDisabled = true;
  }

  return (
    <div>
      <GoldenDragonEffect show={showGoldenDragon} onDone={() => setShowGoldenDragon(false)} />
      {capEnabled && (
        <div className='farm-card' style={{ marginBottom: 14, padding: '12px 16px' }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: 6,
            }}
          >
            <Text strong>
              💵 {t('今日钓鱼收益')}: ${dailyIncome.toFixed(2)} / ${dailyIncomeCap.toFixed(2)}
            </Text>
            {overCap && (
              <Tag size='small' color='red'>
                🚫 {t('已达上限')}
              </Tag>
            )}
          </div>
          <Progress
            percent={incomeCapPct}
            showInfo={false}
            size='large'
            stroke={overCap ? '#f59e0b' : incomeCapPct > 80 ? '#eab308' : '#22c55e'}
          />
          {!overCap && dailyIncomeCap > dailyIncome && (
            <Text size='small' type='tertiary' style={{ marginTop: 4, display: 'block' }}>
              📳 {t('距离上限还差')}: ${(dailyIncomeCap - dailyIncome).toFixed(2)}
            </Text>
          )}
        </div>
      )}

      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 14 }}>
        <div className='farm-pill farm-pill-amber'>🪱 {t('鱼饵')}: {fishData.bait_count}</div>
        <div className='farm-pill farm-pill-purple'>✨ {t('高级鱼饵')}: {fishData.premium_bait_count || 0}</div>
        <div className='farm-pill farm-pill-cyan'>📳 {t('今日次数')}: {dailyCount}</div>
        {!capEnabled && (
          <div className='farm-pill farm-pill-amber'>
            💵 ${dailyIncome.toFixed(2)} / ${dailyMaxIncome.toFixed(2)}
          </div>
        )}
        {fatigueActive ? (
          <div className='farm-pill farm-pill-red'>😵 {t('疲劳中')} -{fatigueDecay}%</div>
        ) : (
          <div className='farm-pill farm-pill-green'>😊 {dailyCount}/{fatigueThreshold}</div>
        )}
        {(fishData.premium_bait_count || 0) > 0 && (
          <div className='farm-pill farm-pill-purple'>✨ {t('优先消耗，史诗/传说额外概率+5%')}</div>
        )}
        {fishData.total_value > 0 && (
          <div className='farm-pill farm-pill-cyan'>
            💵 {t('鱼仓价值')}: ${fishData.total_value.toFixed(2)}
          </div>
        )}
      </div>

      <div style={{ display: 'flex', gap: 8, marginBottom: 14 }}>
        <Button
          theme='solid'
          type={overCap && capEnabled ? 'warning' : 'primary'}
          loading={fishLoading}
          disabled={btnDisabled}
          onClick={doFish}
          icon={<Fish size={14} />}
          className='farm-btn'
        >
          {btnText}
        </Button>
        {fishData.total_value > 0 && (
          <Button
            theme='light'
            type='secondary'
            loading={fishLoading}
            onClick={doStoreAll}
            className='farm-btn'
          >
            {t('存入仓库')}
          </Button>
        )}
        {fishData.total_value > 0 && (
          <Button
            theme='light'
            type='warning'
            loading={fishLoading}
            onClick={doSellAll}
            className='farm-btn'
          >
            💵 {t('出售全部')} (${fishData.total_value.toFixed(2)})
          </Button>
        )}
      </div>

      {lastCatch && (
        <Banner
          type={lastCatch.caught ? (lastCatch.cap_reached_after_catch ? 'warning' : 'success') : 'warning'}
          closeIcon={null}
          style={{ marginBottom: 14, borderRadius: 12 }}
          description={
            lastCatch.caught ? (
              <span style={{ fontSize: 15 }}>
                {lastCatch.fish_emoji} {t('钓到了')} <strong>{lastCatch.fish_name}</strong>{' '}
                <Tag size='small' color={rarityColors[lastCatch.rarity]}>[{lastCatch.rarity}]</Tag>{' '}
                {lastCatch.cap_reached_after_catch ? (
                  <>
                    {t('价值')} <Text type='warning'>${lastCatch.effective_price.toFixed(2)}</Text>{' '}
                    <Tag size='small' color='orange'>
                      {t('今日已满')}
                    </Tag>
                  </>
                ) : (
                  <>
                    {t('价值')} ${lastCatch.sell_price.toFixed(2)}
                  </>
                )}
              </span>
            ) : (
              <span style={{ fontSize: 15 }}>🗑️ {t('空军！什么都没钓到...')}</span>
            )
          }
        />
      )}

      {fishData.inventory && fishData.inventory.length > 0 && (
        <div className='farm-card'>
          <div className='farm-section-title'>📦 {t('鱼仓库')}</div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {fishData.inventory.map((fish) => (
              <div
                key={fish.key}
                className='farm-card-flat'
                style={{ display: 'flex', alignItems: 'center', gap: 8 }}
              >
                <span style={{ fontSize: 20 }}>{fish.emoji}</span>
                <div>
                  <Text size='small' strong>
                    {fish.name} x{fish.quantity}
                  </Text>
                  <Tag size='small' color={rarityColors[fish.rarity]} style={{ marginLeft: 4 }}>
                    {fish.rarity}
                  </Tag>
                  <Text size='small' type='tertiary' style={{ display: 'block' }}>
                    ${fish.total_value.toFixed(2)}
                  </Text>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className='farm-card'>
        <div className='farm-section-title'>📳 {t('鱼种图鉴')}</div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
          {fishData.fish_types &&
            fishData.fish_types.map((ft) => (
              <div
                key={ft.key}
                className='farm-card-flat'
                style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 120 }}
              >
                <span style={{ fontSize: 18 }}>{ft.emoji}</span>
                <div>
                  <Text size='small'>{ft.name}</Text>
                  <Tag size='small' color={rarityColors[ft.rarity]} style={{ marginLeft: 4 }}>
                    {ft.rarity}
                  </Tag>
                  <Text size='small' type='tertiary' style={{ display: 'block' }}>
                    {ft.chance}% / ${ft.sell_price.toFixed(2)}
                  </Text>
                </div>
              </div>
            ))}
          <div
            className='farm-card-flat'
            style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 120 }}
          >
            <span style={{ fontSize: 18 }}>🗑️</span>
            <div>
              <Text size='small'>{t('空军')}</Text>
              <Text size='small' type='tertiary' style={{ display: 'block' }}>
                {fishData.nothing_chance}%
              </Text>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default FishPage;
