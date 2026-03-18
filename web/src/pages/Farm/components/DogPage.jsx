import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { API, confirmAction } from './utils';

const { Text } = Typography;

const DogPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [dogData, setDogData] = useState(null);
  const [dogLoading, setDogLoading] = useState(true);

  const loadDog = useCallback(async () => {
    setDogLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/dog');
      if (res.success) setDogData(res.data);
    } catch (err) { /* ignore */ }
    finally { setDogLoading(false); }
  }, []);

  useEffect(() => { loadDog(); }, [loadDog]);

  const handleBuyDog = async () => {
    if (!await confirmAction(t('购买确认'), t('确认购买看门狗？'))) return;
    const res = await doAction('/api/farm/buydog', {});
    if (res) { loadDog(); loadFarm(); }
  };

  const handleFeedDog = async () => {
    if (!await confirmAction(t('喂狗确认'), t('确认花费喂狗粮？'))) return;
    const res = await doAction('/api/farm/feeddog', {});
    if (res) { loadDog(); loadFarm(); }
  };

  if (dogLoading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }

  if (!dogData || !dogData.has_dog) {
    return (
      <div className='farm-card'>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 16 }}>
          <span style={{ fontSize: 36 }}>🐶</span>
          <div>
            <Text strong style={{ fontSize: 15 }}>{t('你还没有看门狗')}</Text>
            <div><Text type='tertiary' size='small'>{t('养大后可拦截偷菜者')}</Text></div>
          </div>
        </div>
        <div className='farm-kv-grid' style={{ marginBottom: 16 }}>
          {[
            { label: t('价格'), value: `$${dogData?.dog_price?.toFixed(2)}` },
            { label: t('成长'), value: `${dogData?.grow_hours}${t('小时')}` },
            { label: t('拦截率'), value: `${dogData?.guard_rate}%` },
            { label: t('狗粮'), value: `$${dogData?.food_price?.toFixed(2)}` },
          ].map(s => (
            <div key={s.label} className='farm-kv'>
              <div className='farm-kv-label'>{s.label}</div>
              <div className='farm-kv-value'>{s.value}</div>
            </div>
          ))}
        </div>
        <Button theme='solid' onClick={handleBuyDog} loading={actionLoading} className='farm-btn'>
          🐶 {t('购买小狗')} (${dogData?.dog_price?.toFixed(2)})
        </Button>
      </div>
    );
  }

  return (
    <div className='farm-card'>
      {/* Profile */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 14, flexWrap: 'wrap' }}>
        <span style={{ fontSize: 36 }}>{dogData.level === 2 ? '🐕' : '🐶'}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
            <Text strong style={{ fontSize: 16 }}>「{dogData.name}」</Text>
            <Tag size='small' color={dogData.hunger > 0 ? 'green' : 'red'}>
              {dogData.level_name} · {dogData.status}
            </Tag>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
            <Text type='tertiary' size='small'>{t('饱食度')}</Text>
            <div className='farm-progress' style={{ flex: 1, maxWidth: 180, height: 8 }}>
              <div className='farm-progress-fill' style={{
                width: `${dogData.hunger}%`,
                background: dogData.hunger > 30 ? 'linear-gradient(90deg, var(--farm-leaf), var(--farm-leaf))' : 'linear-gradient(90deg, var(--farm-danger), var(--farm-danger))',
              }} />
            </div>
            <Text strong size='small'>{dogData.hunger}%</Text>
          </div>
        </div>
        {dogData.hunger < 100 && (
          <Button size='small' theme='solid' onClick={handleFeedDog} loading={actionLoading} className='farm-btn'>
            🦴 {t('喂食')}
          </Button>
        )}
      </div>

      {/* Stats */}
      <div className='farm-kv-grid' style={{ marginBottom: 12 }}>
        {[
          { label: t('等级'), value: dogData.level_name },
          { label: t('饱食度'), value: `${dogData.hunger}%` },
          { label: t('拦截率'), value: `${dogData.guard_rate}%` },
          { label: t('狗粮'), value: `$${dogData.food_price?.toFixed(2)}` },
        ].map(s => (
          <div key={s.label} className='farm-kv'>
            <div className='farm-kv-label'>{s.label}</div>
            <div className='farm-kv-value'>{s.value}</div>
          </div>
        ))}
      </div>

      <Text type='tertiary' size='small'>
        💡 {t('饱食度为0时无法看门，每小时-1点，请定期喂食狗粮')}
      </Text>
    </div>
  );
};

export default DogPage;
