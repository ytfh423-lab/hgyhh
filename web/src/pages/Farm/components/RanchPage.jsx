import React, { useCallback, useEffect, useState } from 'react';
import { Button, Empty, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { RefreshCw } from 'lucide-react';
import { API, formatBalance, formatDuration } from './utils';
import { farmConfirm } from './farmConfirm';

const { Text, Title } = Typography;

const RanchPage = ({ actionLoading, doAction, t }) => {
  const [ranchData, setRanchData] = useState(null);
  const [ranchLoading, setRanchLoading] = useState(true);

  const loadRanch = useCallback(async () => {
    setRanchLoading(true);
    try {
      const { data: res } = await API.get('/api/ranch/view');
      if (res.success) setRanchData(res.data);
    } catch (err) { /* ignore */ }
    finally { setRanchLoading(false); }
  }, []);

  useEffect(() => { loadRanch(); }, [loadRanch]);

  useEffect(() => {
    const interval = setInterval(() => {
      if (!document.hidden) loadRanch();
    }, 60000);
    return () => clearInterval(interval);
  }, [loadRanch]);

  const doRanchAction = async (url, body) => {
    const res = await doAction(url, body);
    if (res) { loadRanch(); }
    return res;
  };

  if (ranchLoading && !ranchData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!ranchData) return null;

  const animals = ranchData.animals || [];
  const animalTypes = ranchData.animal_types || [];
  const deadAnimals = animals.filter(a => a.status === 5);
  const aliveAnimals = animals.filter(a => a.status !== 5);
  const dirtyAnimals = aliveAnimals.filter(a => a.is_dirty);

  const statusLabels = { 1: '生长中', 2: '已成熟', 3: '饥饿', 4: '口渴', 5: '已死亡' };
  const statusTagColors = { 1: 'blue', 2: 'green', 3: 'orange', 4: 'red', 5: 'grey' };

  return (
    <div>
      {/* Status bar */}
      <div className='farm-card' style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 8, padding: '10px 16px' }}>
        <div className='farm-pill farm-pill-green'>💰 {formatBalance(ranchData.balance)}</div>
        <div className='farm-pill'>🐄 {ranchData.alive_count}/{ranchData.max_animals}</div>
        <div className='farm-pill farm-pill-cyan'>🌾 {formatBalance(ranchData.feed_price)}/{t('次')}</div>
        <div className='farm-pill farm-pill-blue'>💧 {formatBalance(ranchData.water_price)}/{t('次')}</div>
        <div style={{ flex: 1 }} />
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadRanch} loading={ranchLoading} className='farm-btn' />
        {dirtyAnimals.length > 0 && (
          <Button size='small' theme='light' onClick={() => doRanchAction('/api/ranch/clean', {})}
            loading={actionLoading} className='farm-btn' style={{ color: 'var(--farm-soil)', borderColor: 'var(--farm-harvest)' }}>
            🧹 {t('清理粪便')}({formatBalance(ranchData.manure_clean_price)})
          </Button>
        )}
        {deadAnimals.length > 0 && (
          <Button size='small' theme='light' type='danger' onClick={() => doRanchAction('/api/ranch/cleanup', {})}
            loading={actionLoading} className='farm-btn'>
            🗑️ {t('清理')}({deadAnimals.length})
          </Button>
        )}
      </div>

      {/* Animal list */}
      {animals.length === 0 ? (
        <div className='farm-card' style={{ textAlign: 'center', padding: 30 }}>
          <span style={{ fontSize: 36 }}>🏚️</span>
          <Title heading={6} style={{ marginTop: 8 }}>{t('牧场空空如也')}</Title>
          <Text type='tertiary' size='small'>{t('去下方购买动物开始养殖吧！')}</Text>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {animals.map((animal) => (
            <div key={animal.id} className='farm-card' style={{ marginBottom: 0, padding: '12px 16px' }}>
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12, flexWrap: 'wrap' }}>
                <span style={{ fontSize: 32 }}>{animal.animal_emoji}</span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4, flexWrap: 'wrap' }}>
                    <Text strong style={{ fontSize: 15 }}>{animal.animal_name}</Text>
                    <Tag size='small' color={statusTagColors[animal.status] || 'grey'}>
                      {statusLabels[animal.status] || animal.status_label}
                    </Tag>
                    {animal.needs_feed && animal.status !== 5 && <Tag size='small' color='orange'>⚠️ {t('需喂食')}</Tag>}
                    {animal.needs_water && animal.status !== 5 && <Tag size='small' color='red'>⚠️ {t('需喂水')}</Tag>}
                    {animal.is_dirty && animal.status !== 5 && <Tag size='small' color='amber'>💩 {t('脏污')}</Tag>}
                  </div>
                  {animal.status === 1 && (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <div className='farm-progress' style={{ flex: 1, maxWidth: 200 }}>
                        <div className='farm-progress-fill' style={{ width: `${animal.progress}%`, background: 'linear-gradient(90deg, var(--farm-sky), var(--farm-leaf))' }} />
                      </div>
                      <Text type='tertiary' size='small'>{animal.progress}% · {formatDuration(animal.remaining)}</Text>
                    </div>
                  )}
                  {animal.status === 2 && (
                    <Text type='success' size='small'>🥩 {t('肉价')} {formatBalance(animal.meat_price)}</Text>
                  )}
                  {(animal.status === 1 || animal.status === 2) && (
                    <div style={{ display: 'flex', gap: 8, marginTop: 2 }}>
                      {!animal.needs_feed && animal.feed_remaining > 0 && <Text type='tertiary' size='small'>🌾 {formatDuration(animal.feed_remaining)}</Text>}
                      {!animal.needs_water && animal.water_remaining > 0 && <Text type='tertiary' size='small'>💧 {formatDuration(animal.water_remaining)}</Text>}
                      {!animal.is_dirty && animal.clean_remaining > 0 && <Text type='tertiary' size='small'>🧹 {formatDuration(animal.clean_remaining)}</Text>}
                    </div>
                  )}
                </div>
                <div style={{ display: 'flex', gap: 4, flexShrink: 0, flexWrap: 'wrap', justifyContent: 'flex-end' }}>
                  {animal.status !== 5 && (
                    <>
                      <Button size='small' theme='light' onClick={() => doRanchAction('/api/ranch/feed', { animal_id: animal.id })}
                        loading={actionLoading} className='farm-btn' disabled={!animal.needs_feed}>🌾</Button>
                      <Button size='small' theme='light' onClick={() => doRanchAction('/api/ranch/water', { animal_id: animal.id })}
                        loading={actionLoading} className='farm-btn' disabled={!animal.needs_water}>💧</Button>
                    </>
                  )}
                  {animal.status === 2 && (
                    <>
                      <Button size='small' theme='solid' type='warning'
                        onClick={() => doRanchAction('/api/ranch/slaughter', { animal_id: animal.id })}
                        loading={actionLoading} className='farm-btn'>
                        💰 {t('出售')}
                      </Button>
                      <Button size='small' theme='light'
                        onClick={() => doRanchAction('/api/ranch/slaughter/store', { animal_id: animal.id })}
                        loading={actionLoading} className='farm-btn' style={{ color: 'var(--farm-sky)', borderColor: 'var(--farm-sky)' }}>
                        📦 {t('存仓库')}
                      </Button>
                    </>
                  )}
                  {animal.status !== 5 && (
                    <Button size='small' theme='light' type='danger'
                      onClick={async () => { if (await farmConfirm(t('放生动物'), t('确定要放生这只动物吗？不会退款。'), { icon: '🐾', confirmType: 'danger', confirmText: t('放生') })) doRanchAction('/api/ranch/release', { animal_id: animal.id }); }}
                      loading={actionLoading} className='farm-btn'>
                      🔓 {t('放生')}
                    </Button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Buy animals */}
      {aliveAnimals.length < ranchData.max_animals && (
        <div className='farm-card' style={{ marginTop: 8 }}>
          <div className='farm-section-title'>🛒 {t('购买动物')}</div>
          <div className='farm-grid farm-grid-4'>
            {animalTypes.map((at) => (
              <div key={at.key} className='farm-item-card' onClick={() => doRanchAction('/api/ranch/buy', { animal_type: at.key })}>
                <span style={{ fontSize: 28, display: 'block', marginBottom: 4 }}>{at.emoji}</span>
                <Text strong size='small' style={{ display: 'block' }}>{at.name}</Text>
                <Text type='tertiary' size='small' style={{ display: 'block' }}>{formatBalance(at.buy_price)}</Text>
                <Text type='tertiary' size='small' style={{ display: 'block' }}>⏱️ {Math.round(at.grow_secs / 3600)}h</Text>
                <Text type='success' size='small' style={{ display: 'block' }}>🥩 {formatBalance(at.meat_price)}</Text>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default RanchPage;
