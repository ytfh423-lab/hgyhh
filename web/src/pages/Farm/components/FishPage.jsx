import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Banner, Typography } from '@douyinfe/semi-ui';
import { Fish } from 'lucide-react';
import { API, showError, showSuccess, formatDuration } from './utils';

const { Text } = Typography;

const rarityColors = {
  '普通': 'grey', '优良': 'green', '稀有': 'blue', '史诗': 'purple', '传说': 'orange',
};

const FishPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [fishData, setFishData] = useState(null);
  const [fishLoading, setFishLoading] = useState(false);
  const [lastCatch, setLastCatch] = useState(null);
  const [cooldown, setCooldown] = useState(0);

  const loadFish = useCallback(async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/fish');
      if (res.success) {
        setFishData(res.data);
        setCooldown(res.data.cooldown || 0);
      }
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setFishLoading(false);
    }
  }, [t]);

  useEffect(() => { loadFish(); }, [loadFish]);

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setInterval(() => {
      setCooldown(prev => {
        if (prev <= 1) { clearInterval(timer); return 0; }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [cooldown]);

  const doFish = async () => {
    setFishLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/fish');
      if (res.success) {
        setLastCatch(res.data);
        if (res.data.caught) showSuccess(res.message);
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

  return (
    <div>
      {/* Status */}
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 14 }}>
        <div className='farm-pill farm-pill-amber'>🪱 {t('鱼饵')}: {fishData.bait_count}</div>
        <div className={`farm-pill ${cooldown > 0 ? 'farm-pill-red' : 'farm-pill-green'}`}>
          {cooldown > 0 ? `⏱️ ${cooldown}s` : `✅ ${t('可以钓鱼')}`}
        </div>
        {fishData.total_value > 0 && (
          <div className='farm-pill farm-pill-cyan'>💰 {t('鱼仓价值')}: ${fishData.total_value.toFixed(2)}</div>
        )}
      </div>

      {/* Actions */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 14 }}>
        <Button theme='solid' type='primary' loading={fishLoading}
          disabled={cooldown > 0 || fishData.bait_count === 0}
          onClick={doFish} icon={<Fish size={14} />} className='farm-btn'>
          {cooldown > 0 ? `${t('冷却中')} ${cooldown}s` : fishData.bait_count === 0 ? t('没有鱼饵') : t('开始钓鱼')}
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
