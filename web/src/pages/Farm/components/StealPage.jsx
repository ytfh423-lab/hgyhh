import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Empty, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { RefreshCw } from 'lucide-react';
import { API } from './utils';

const { Text } = Typography;

const StealPage = ({ actionLoading, doAction, t }) => {
  const [targets, setTargets] = useState([]);
  const [stealInfo, setStealInfo] = useState(null);
  const [stealLoading, setStealLoading] = useState(true);
  const [refreshingTargets, setRefreshingTargets] = useState(false);
  const [stealResults, setStealResults] = useState([]);
  const [rules, setRules] = useState(null);
  const [stealDisabled, setStealDisabled] = useState(false);
  const [victimCooldowns, setVictimCooldowns] = useState({});
  const cooldownTimersRef = useRef({});

  const clearVictimCooldown = useCallback((victimId) => {
    if (cooldownTimersRef.current[victimId]) {
      clearTimeout(cooldownTimersRef.current[victimId]);
      delete cooldownTimersRef.current[victimId];
    }
    setVictimCooldowns((prev) => {
      if (!prev[victimId]) {
        return prev;
      }
      const next = { ...prev };
      delete next[victimId];
      return next;
    });
  }, []);

  const markVictimCooldown = useCallback((victimId, cooldownSeconds) => {
    clearVictimCooldown(victimId);
    if (!cooldownSeconds || cooldownSeconds <= 0) {
      return;
    }
    setVictimCooldowns((prev) => ({
      ...prev,
      [victimId]: Date.now() + (cooldownSeconds * 1000),
    }));
    cooldownTimersRef.current[victimId] = setTimeout(() => {
      clearVictimCooldown(victimId);
    }, cooldownSeconds * 1000);
  }, [clearVictimCooldown]);

  const loadTargets = useCallback(async ({ background = false } = {}) => {
    if (background) {
      setRefreshingTargets(true);
    } else {
      setStealLoading(true);
    }
    try {
      const { data: res } = await API.get('/api/farm/steal/targets');
      if (res.success) {
        setTargets(res.data || []);
        setStealInfo(res.steal_info || null);
        setStealDisabled(!!res.steal_disabled);
      }
    } catch (err) { /* ignore */ }
    finally {
      if (background) {
        setRefreshingTargets(false);
      } else {
        setStealLoading(false);
      }
    }
  }, []);

  const loadRules = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/steal/rules');
      if (res.success) setRules(res.data);
    } catch (err) { /* ignore */ }
  }, []);

  useEffect(() => { loadTargets(); loadRules(); }, [loadTargets, loadRules]);

  useEffect(() => () => {
    Object.values(cooldownTimersRef.current).forEach((timer) => clearTimeout(timer));
  }, []);

  const handleSteal = async (victimId) => {
    const victimLabel = targets.find((target) => target.id === victimId)?.label;
    const res = await doAction('/api/farm/steal', { victim_id: victimId });
    if (res) {
      setStealResults(prev => [{
        time: new Date().toLocaleTimeString(),
        message: res.message,
        data: res.data,
        victim: res.data?.victim || victimLabel,
      }, ...prev]);
      setStealInfo((prev) => {
        if (!prev) {
          return prev;
        }
        return {
          ...prev,
          today_count: Math.min(prev.daily_limit, (prev.today_count || 0) + 1),
        };
      });
      markVictimCooldown(victimId, stealInfo?.cooldown_secs || 0);
      loadTargets({ background: true });
    }
  };

  if (stealLoading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }

  if (stealDisabled) {
    return (
      <div className='farm-card' style={{ textAlign: 'center', padding: 40 }}>
        <Text type='tertiary'>🚫 {t('偷菜功能当前已关闭')}</Text>
      </div>
    );
  }

  return (
    <div>
      {/* 规则摘要 */}
      {rules && (
        <div className='farm-card' style={{ marginBottom: 8 }}>
          <div className='farm-section-title'>📋 {t('偷菜规则')}</div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, fontSize: 11 }}>
            <Tag size='small' color='blue'>{t('主人保底')} {rules.owner_keep_pct}%</Tag>
            <Tag size='small' color='orange'>{t('可偷')} {rules.stealable_pct}%</Tag>
            <Tag size='small' color='cyan'>{t('保护期')} {rules.protection_minutes}{t('分钟')}</Tag>
            <Tag size='small' color='green'>{t('每地最多偷')} {rules.max_steal_per_plot}{t('次')}</Tag>
            <Tag size='small' color='violet'>{t('每日上限')} {rules.max_steal_per_day}{t('次')}</Tag>
            {rules.long_crop_hours > 0 && (
              <Tag size='small' color='red'>{rules.long_crop_hours}h+{t('作物保底')} {rules.long_crop_keep_pct}%</Tag>
            )}
          </div>
          <Text type='tertiary' size='small' style={{ display: 'block', marginTop: 6 }}>
            💡 {t('偷菜只能摘取额外收益，主人的基础收益始终受保护。')}
          </Text>
        </div>
      )}

      {/* 今日状态 */}
      {stealInfo && (
        <div className='farm-card' style={{ marginBottom: 8 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: 12 }}>
            <Text size='small'>{t('今日已偷')}: <Text strong>{stealInfo.today_count}</Text> / {stealInfo.daily_limit}</Text>
            <Text type='tertiary' size='small'>{t('冷却')}: {Math.floor(stealInfo.cooldown_secs / 60)}{t('分钟')}</Text>
          </div>
        </div>
      )}

      {/* Targets */}
      <div className='farm-card'>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
          <div className='farm-section-title' style={{ marginBottom: 0 }}>🕵️ {t('可偷菜的农场')}</div>
          <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={() => loadTargets({ background: true })} loading={refreshingTargets} className='farm-btn' />
        </div>

        {targets.length === 0 ? (
          <Empty description={t('暂时没有可偷的菜地')} style={{ padding: '20px 0' }} />
        ) : (
          <div>
            {targets.map((target) => (
              <div key={target.id} className='farm-row'>
                <Text strong size='small'>👤 {target.label}</Text>
                <Tag size='small' color='green'>{target.count}{t('块')}</Tag>
                {(victimCooldowns[target.id] || 0) > Date.now() && <Tag size='small' color='grey'>{t('冷却中')}</Tag>}
                <div style={{ flex: 1 }} />
                <Button size='small' type='warning' theme='solid' onClick={() => handleSteal(target.id)}
                  loading={actionLoading} className='farm-btn'
                  disabled={(stealInfo && stealInfo.today_count >= stealInfo.daily_limit) || (victimCooldowns[target.id] || 0) > Date.now()}>
                  {(victimCooldowns[target.id] || 0) > Date.now() ? t('冷却中') : t('摘取额外收益')}
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Results */}
      {stealResults.length > 0 && (
        <div className='farm-card'>
          <div className='farm-section-title'>📜 {t('偷菜记录')}</div>
          {stealResults.map((r, i) => (
            <div key={i} className='farm-row' style={{ marginBottom: 4 }}>
              <Text size='small'><Text type='tertiary' size='small'>{r.time}</Text> {r.message}</Text>
              {r.victim && <Tag size='small' color='blue'>👤 {r.victim}</Tag>}
              {r.data && <Tag size='small' color='green'>${r.data.value?.toFixed(2)}</Tag>}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default StealPage;
