import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Empty, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { RefreshCw } from 'lucide-react';
import { API, formatBalance, formatDuration } from './utils';

const { Text } = Typography;

const qualityColor = {
  1: 'grey',
  2: 'green',
  3: 'blue',
  4: 'purple',
  5: 'amber',
};

const BreedingPage = ({ doAction, actionLoading, t }) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [parentA, setParentA] = useState(null);
  const [parentB, setParentB] = useState(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/ranch/breed/view');
      if (res.success) setData(res.data);
    } catch (e) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const animals = data?.animals || [];
  const breedings = data?.breedings || [];
  const readyAnimals = useMemo(() => animals.filter((a) => a.status === 2 && !a.breed_cooldown_remaining), [animals]);
  const selectedA = readyAnimals.find((a) => a.id === parentA);
  const candidatesB = readyAnimals.filter((a) => selectedA && a.id !== selectedA.id && a.animal_type === selectedA.animal_type);

  const start = async () => {
    if (!parentA || !parentB) return;
    const res = await doAction('/api/ranch/breed/start', { parent_a_id: parentA, parent_b_id: parentB });
    if (res) {
      setParentA(null);
      setParentB(null);
      load();
    }
  };

  const claim = async (id) => {
    const res = await doAction('/api/ranch/breed/claim', { breeding_id: id });
    if (res) load();
  };

  if (loading && !data) return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  if (!data) return null;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div className='farm-card' style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center', padding: '10px 16px' }}>
        <div className='farm-pill farm-pill-green'>💰 {formatBalance(data.balance)}</div>
        <div className='farm-pill'>🧬 {data.active_count}/{data.max_active}</div>
        <div className='farm-pill farm-pill-cyan'>🐄 {data.alive_count}/{data.max_animals}</div>
        <div style={{ flex: 1 }} />
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={load} loading={loading} className='farm-btn' />
      </div>

      <div className='farm-card'>
        <div className='farm-section-title'>🧬 {t('育种')}</div>
        <Text type='tertiary' size='small'>{t('选择两只成熟且同种的动物，等待后领取后代。')}</Text>
      </div>

      <div className='farm-card'>
        <div className='farm-section-title'>{t('进行中的育种')}</div>
        {breedings.length === 0 ? (
          <Empty description={t('暂无育种记录')} image={<div style={{ fontSize: 42 }}>🥚</div>} />
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {breedings.map((item) => (
              <div key={item.id} className='farm-row' style={{ justifyContent: 'space-between', gap: 10, padding: 12, borderRadius: 14, background: 'rgba(0,0,0,0.02)' }}>
                <div>
                  <div style={{ fontWeight: 700 }}>{item.animal_emoji} {item.animal_name} #{item.parent_a_id} × #{item.parent_b_id}</div>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 4 }}>
                    <Tag size='small' color={qualityColor[item.parent_a_quality] || 'grey'}>{item.parent_a_quality_label}</Tag>
                    <Tag size='small' color={qualityColor[item.parent_b_quality] || 'grey'}>{item.parent_b_quality_label}</Tag>
                    <Tag size='small' color={item.status === 2 ? 'green' : 'blue'}>{item.status_label}</Tag>
                    {item.offspring_quality > 0 && <Tag size='small' color={qualityColor[item.offspring_quality] || 'grey'}>{item.offspring_quality_label}</Tag>}
                  </div>
                  <Text type='tertiary' size='small'>
                    {item.status === 1 ? `⏳ ${formatDuration(item.remaining)}` : `G${item.offspring_generation} · ${t('投入')} ${formatBalance(item.cost)}`}
                  </Text>
                </div>
                {item.status === 2 && (
                  <Button size='small' theme='solid' type='primary' onClick={() => claim(item.id)} loading={actionLoading} className='farm-btn'>
                    {t('领取')}
                  </Button>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      <div className='farm-card'>
        <div className='farm-section-title'>{t('选择亲本')}</div>
        {readyAnimals.length < 2 ? (
          <Empty description={t('至少需要两只成熟且未冷却的动物')} image={<div style={{ fontSize: 40 }}>🐣</div>} />
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <div>
              <div style={{ fontWeight: 700, marginBottom: 8 }}>{t('第一只')}</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                {readyAnimals.map((animal) => (
                  <div key={animal.id} className='farm-row' onClick={() => { setParentA(animal.id); setParentB(null); }} style={{ cursor: 'pointer', padding: 10, borderRadius: 12, background: parentA === animal.id ? 'rgba(0,0,0,0.06)' : 'rgba(0,0,0,0.02)' }}>
                    <span style={{ fontSize: 24 }}>{animal.animal_emoji}</span>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontWeight: 700 }}>{animal.animal_name}</div>
                      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 4 }}>
                        <Tag size='small' color={qualityColor[animal.quality] || 'grey'}>{animal.quality_label}</Tag>
                        <Tag size='small' color='grey'>G{animal.generation}</Tag>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
            <div>
              <div style={{ fontWeight: 700, marginBottom: 8 }}>{t('第二只')}</div>
              {!selectedA ? (
                <Empty description={t('先选择第一只动物')} image={<div style={{ fontSize: 36 }}>👉</div>} />
              ) : candidatesB.length === 0 ? (
                <Empty description={t('没有可配对的同种动物')} image={<div style={{ fontSize: 36 }}>🐄</div>} />
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  {candidatesB.map((animal) => (
                    <div key={animal.id} className='farm-row' onClick={() => setParentB(animal.id)} style={{ cursor: 'pointer', padding: 10, borderRadius: 12, background: parentB === animal.id ? 'rgba(0,0,0,0.06)' : 'rgba(0,0,0,0.02)' }}>
                      <span style={{ fontSize: 24 }}>{animal.animal_emoji}</span>
                      <div style={{ flex: 1 }}>
                        <div style={{ fontWeight: 700 }}>{animal.animal_name}</div>
                        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 4 }}>
                          <Tag size='small' color={qualityColor[animal.quality] || 'grey'}>{animal.quality_label}</Tag>
                          <Tag size='small' color='grey'>G{animal.generation}</Tag>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
        <div style={{ marginTop: 12, display: 'flex', justifyContent: 'flex-end' }}>
          <Button theme='solid' type='primary' className='farm-btn' disabled={!parentA || !parentB} loading={actionLoading} onClick={start}>
            {t('开始育种')}
          </Button>
        </div>
      </div>
    </div>
  );
};

export default BreedingPage;
