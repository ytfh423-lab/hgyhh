import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './utils';

const { Text, Title } = Typography;

const LevelPage = ({ actionLoading, loadFarm, t }) => {
  const [lvData, setLvData] = useState(null);
  const [loading, setLoading] = useState(false);

  const loadLevel = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/level');
      if (res.success) setLvData(res.data);
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => { loadLevel(); }, [loadLevel]);

  const doLevelUp = async () => {
    setLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/levelup');
      if (res.success) {
        showSuccess(res.message);
        loadLevel();
        loadFarm();
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setLoading(false);
    }
  };

  if (loading && !lvData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!lvData) return null;

  const isMax = lvData.level >= lvData.max_level;
  const pct = Math.round((lvData.level / lvData.max_level) * 100);

  return (
    <div>
      {/* Level header */}
      <div className='farm-card farm-card-glow-amber'>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontSize: 36 }}>⭐</span>
          <div style={{ flex: 1 }}>
            <Title heading={4} style={{ margin: 0 }}>Lv.{lvData.level}</Title>
            <div className='farm-progress' style={{ height: 8, marginTop: 6 }}>
              <div className='farm-progress-fill' style={{
                width: `${pct}%`,
                background: isMax ? 'linear-gradient(90deg, #16a34a, #22c55e)' : 'linear-gradient(90deg, #f59e0b, #eab308)',
              }} />
            </div>
            <Text size='small' type='tertiary'>{lvData.level}/{lvData.max_level}</Text>
          </div>
          {!isMax && (
            <Button theme='solid' type='warning' loading={loading} onClick={doLevelUp} className='farm-btn'>
              ⬆️ {t('升级')} ${lvData.next_price.toFixed(2)}
            </Button>
          )}
          {isMax && <Tag size='large' color='green'>MAX</Tag>}
        </div>
      </div>

      {/* Feature unlocks */}
      <div className='farm-card'>
        <div className='farm-section-title'>🔓 {t('功能解锁')}</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {(lvData.unlocks || []).map((u) => (
            <div key={u.key} className='farm-row' style={{
              background: u.unlocked ? 'rgba(34,197,94,0.08)' : undefined,
              marginBottom: 0,
            }}>
              <span>{u.unlocked ? '✅' : '🔒'}</span>
              <Text style={{ flex: 1 }}>{u.name}</Text>
              <Tag size='small' color={u.unlocked ? 'green' : 'grey'}>Lv.{u.level}</Tag>
            </div>
          ))}
        </div>
      </div>

      {/* Price table */}
      <div className='farm-card'>
        <div className='farm-section-title'>📊 {t('升级价格表')}</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {(lvData.prices || []).map((p) => (
            <div key={p.level} style={{
              display: 'flex', alignItems: 'center', gap: 8, padding: '3px 8px',
              borderRadius: 8,
              background: p.level === lvData.level + 1 ? 'rgba(245,158,11,0.1)' : 'transparent',
              fontWeight: p.level === lvData.level + 1 ? 600 : 400,
            }}>
              <Text style={{ width: 50 }}>Lv.{p.level}</Text>
              <Text>${p.price.toFixed(2)}</Text>
              {p.level <= lvData.level && <Tag size='small' color='green' style={{ marginLeft: 'auto' }}>✅</Tag>}
              {p.level === lvData.level + 1 && <Tag size='small' color='orange' style={{ marginLeft: 'auto' }}>👉 {t('下一级')}</Tag>}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default LevelPage;
