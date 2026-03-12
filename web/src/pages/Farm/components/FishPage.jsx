import React, { useCallback, useEffect, useState, useRef } from 'react';
import { Button, Spin, Tag, Banner, Typography, Progress } from '@douyinfe/semi-ui';
import { Fish } from 'lucide-react';
import { API, showError, showSuccess } from './utils';

const { Text } = Typography;

const rarityColors = {
  '普通': 'grey', '优良': 'green', '稀有': 'blue', '史诗': 'purple', '传说': 'orange',
};

const FishPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [fishData, setFishData] = useState(null);
  const [fishLoading, setFishLoading] = useState(false);
  const [lastCatch, setLastCatch] = useState(null);
  const [cooldown, setCooldown] = useState(0);
  const [recoverIn, setRecoverIn] = useState(0);
  const [stamina, setStamina] = useState(0);
  const tickRef = useRef(null);

  const loadFish = useCallback(async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/fish');
      if (res.success) {
        setFishData(res.data);
        setCooldown(res.data.cooldown || 0);
        setRecoverIn(res.data.recover_in || 0);
        setStamina(res.data.stamina ?? 0);
      }
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setFishLoading(false);
    }
  }, [t]);

  useEffect(() => { loadFish(); }, [loadFish]);

  // Tick timer: countdown cooldown and recovery
  useEffect(() => {
    if (tickRef.current) clearInterval(tickRef.current);
    if (cooldown <= 0 && recoverIn <= 0) return;
    tickRef.current = setInterval(() => {
      setCooldown(prev => Math.max(0, prev - 1));
      setRecoverIn(prev => {
        if (prev <= 1 && prev > 0) {
          // Recovery tick
          setStamina(s => {
            const max = fishData?.stamina_max || 20;
            const amount = fishData?.recover_amount || 1;
            return Math.min(s + amount, max);
          });
          const interval = fishData?.recover_in || 300;
          return interval;
        }
        return Math.max(0, prev - 1);
      });
    }, 1000);
    return () => clearInterval(tickRef.current);
  }, [cooldown > 0, recoverIn > 0, fishData?.stamina_max, fishData?.recover_amount]);

  // Stop recovery when full
  useEffect(() => {
    if (fishData && stamina >= (fishData.stamina_max || 20)) {
      setRecoverIn(0);
    }
  }, [stamina, fishData]);

  const doFish = async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/fish');
      if (res.success) {
        setLastCatch(res.data);
        if (res.data.caught) showSuccess(res.message);
        if (res.data.stamina !== undefined) setStamina(res.data.stamina);
        if (res.data.recover_in !== undefined) setRecoverIn(res.data.recover_in);
        loadFish();
      } else {
        showError(res.message);
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
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setFishLoading(false);
    }
  };

  if (fishLoading && !fishData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!fishData) return null;

  const staminaMax = fishData.stamina_max || 20;
  const staminaCost = fishData.stamina_cost || 1;
  const staminaPct = Math.min(100, Math.round(stamina / staminaMax * 100));
  const dailyCount = fishData.daily_count || 0;
  const dailyMax = fishData.daily_max || 60;
  const dailyIncome = fishData.daily_income || 0;
  const dailyMaxIncome = fishData.daily_max_income || 200;
  const fatigueActive = fishData.fatigue_active || false;
  const fatigueThreshold = fishData.fatigue_threshold || 30;
  const fatigueDecay = fishData.fatigue_decay || 50;
  const dailyLimitReached = dailyCount >= dailyMax;
  const noStamina = stamina < staminaCost;

  // Button text priority: daily limit > no stamina > CD > no bait > ready
  let btnText = t('开始钓鱼');
  let btnDisabled = false;
  if (dailyLimitReached) {
    btnText = t('今日已达上限');
    btnDisabled = true;
  } else if (noStamina) {
    btnText = `⚡ ${t('体力不足')}`;
    btnDisabled = true;
  } else if (cooldown > 0) {
    btnText = `⏱️ ${cooldown}s`;
    btnDisabled = true;
  } else if (fishData.bait_count === 0) {
    btnText = t('没有鱼饵');
    btnDisabled = true;
  }

  return (
    <div>
      {/* Stamina bar */}
      <div className='farm-card' style={{ marginBottom: 14, padding: '12px 16px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
          <Text strong>⚡ {t('体力')}: {stamina}/{staminaMax}</Text>
          {stamina < staminaMax && recoverIn > 0 && (
            <Text size='small' type='tertiary'>🔄 {recoverIn}s (+{fishData.recover_amount || 1})</Text>
          )}
        </div>
        <Progress percent={staminaPct} showInfo={false} size='large'
          stroke={staminaPct > 50 ? '#22c55e' : staminaPct > 20 ? '#eab308' : '#ef4444'} />
      </div>

      {/* Status pills */}
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 14 }}>
        <div className='farm-pill farm-pill-amber'>🪱 {t('鱼饵')}: {fishData.bait_count}</div>
        <div className='farm-pill farm-pill-cyan'>
          📊 {t('今日')}: {dailyCount}/{dailyMax}
        </div>
        <div className='farm-pill farm-pill-amber'>
          💰 ${dailyIncome.toFixed(2)} / ${dailyMaxIncome.toFixed(2)}
        </div>
        {fatigueActive ? (
          <div className='farm-pill farm-pill-red'>😰 {t('疲劳中')} -{fatigueDecay}%</div>
        ) : fatigueThreshold > 0 && (
          <div className='farm-pill farm-pill-green'>😊 {dailyCount}/{fatigueThreshold}</div>
        )}
        {fishData.total_value > 0 && (
          <div className='farm-pill farm-pill-cyan'>💰 {t('鱼仓价值')}: ${fishData.total_value.toFixed(2)}</div>
        )}
      </div>

      {/* Actions */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 14 }}>
        <Button theme='solid' type='primary' loading={fishLoading}
          disabled={btnDisabled}
          onClick={doFish} icon={<Fish size={14} />} className='farm-btn'>
          {btnText}
        </Button>
        {fishData.total_value > 0 && (
          <Button theme='light' type='warning' loading={fishLoading} onClick={doSellAll} className='farm-btn'>
            💰 {t('出售全部')} (${fishData.total_value.toFixed(2)})
          </Button>
        )}
      </div>

      {/* Last catch */}
      {lastCatch && (
        <Banner type={lastCatch.caught ? 'success' : 'warning'} closeIcon={null}
          style={{ marginBottom: 14, borderRadius: 12 }}
          description={lastCatch.caught
            ? <span style={{ fontSize: 15 }}>{lastCatch.fish_emoji} {t('钓到了')} <strong>{lastCatch.fish_name}</strong> <Tag size='small' color={rarityColors[lastCatch.rarity]}>[{lastCatch.rarity}]</Tag> {t('价值')} ${lastCatch.sell_price.toFixed(2)}</span>
            : <span style={{ fontSize: 15 }}>🗑️ {t('空军！什么都没钓到...')}</span>
          }
        />
      )}

      {/* Fish inventory */}
      {fishData.inventory && fishData.inventory.length > 0 && (
        <div className='farm-card'>
          <div className='farm-section-title'>📦 {t('鱼仓库')}</div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {fishData.inventory.map((fish) => (
              <div key={fish.key} className='farm-card-flat' style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ fontSize: 20 }}>{fish.emoji}</span>
                <div>
                  <Text size='small' strong>{fish.name} ×{fish.quantity}</Text>
                  <Tag size='small' color={rarityColors[fish.rarity]} style={{ marginLeft: 4 }}>{fish.rarity}</Tag>
                  <Text size='small' type='tertiary' style={{ display: 'block' }}>${fish.total_value.toFixed(2)}</Text>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Fish types */}
      <div className='farm-card'>
        <div className='farm-section-title'>📊 {t('鱼种图鉴')}</div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
          {fishData.fish_types && fishData.fish_types.map((ft) => (
            <div key={ft.key} className='farm-card-flat' style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 120 }}>
              <span style={{ fontSize: 18 }}>{ft.emoji}</span>
              <div>
                <Text size='small'>{ft.name}</Text>
                <Tag size='small' color={rarityColors[ft.rarity]} style={{ marginLeft: 4 }}>{ft.rarity}</Tag>
                <Text size='small' type='tertiary' style={{ display: 'block' }}>{ft.chance}% · ${ft.sell_price.toFixed(2)}</Text>
              </div>
            </div>
          ))}
          <div className='farm-card-flat' style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 120 }}>
            <span style={{ fontSize: 18 }}>🗑️</span>
            <div>
              <Text size='small'>{t('空军')}</Text>
              <Text size='small' type='tertiary' style={{ display: 'block' }}>{fishData.nothing_chance}%</Text>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default FishPage;
