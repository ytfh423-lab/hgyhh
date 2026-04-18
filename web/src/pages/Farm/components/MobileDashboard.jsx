import React, { useMemo } from 'react';
import { formatBalance } from './utils';
import { FEATURE_LEVEL_MAP } from '../constants';

/**
 * 移动端 Dashboard 首页 —— 替代传统侧边栏导航。
 * 主卡片展示高频模块入口 + 实时状态（来自 farmData，零额外请求）；
 * 次级清单用一行小 Row 展示低频入口；视觉层级清晰，一屏可见所有重要信息。
 */

const Card = ({ emoji, title, subtitle, accent, locked, lockLevel, onClick, badge }) => (
  <div
    className={`farm-home-card ${locked ? 'locked' : ''}`}
    style={{ '--card-accent': accent }}
    onClick={locked ? undefined : onClick}
  >
    <div className='farm-home-card-emoji'>{locked ? '🔒' : emoji}</div>
    <div className='farm-home-card-body'>
      <div className='farm-home-card-title'>{title}</div>
      <div className='farm-home-card-subtitle'>
        {locked ? `Lv.${lockLevel} 解锁` : subtitle || ' '}
      </div>
    </div>
    {badge != null && <div className='farm-home-card-badge'>{badge}</div>}
  </div>
);

const MiniRow = ({ emoji, label, locked, lockLevel, onClick }) => (
  <div
    className={`farm-home-mini ${locked ? 'locked' : ''}`}
    onClick={locked ? undefined : onClick}
  >
    <span className='farm-home-mini-emoji'>{locked ? '🔒' : emoji}</span>
    <span className='farm-home-mini-label'>{label}</span>
    {locked && <span className='farm-home-mini-lock'>Lv.{lockLevel}</span>}
  </div>
);

const MobileDashboard = ({ farmData, userLevel = 1, onNavigate, friendRequestCount = 0, t }) => {
  const plots = farmData?.plots || [];
  const matureCount = plots.filter((p) => p.status === 2).length;
  const growingCount = plots.filter((p) => p.status === 1).length;
  const eventCount = plots.filter((p) => p.status === 3 || p.status === 4).length;
  const plotSubtitle = useMemo(() => {
    if (matureCount > 0) return t('{{n}} 块可收获', { n: matureCount });
    if (eventCount > 0) return t('{{n}} 块需处理', { n: eventCount });
    if (growingCount > 0) return t('{{n}} 块生长中', { n: growingCount });
    if (plots.length === 0) return t('开始开垦');
    return t('{{a}}/{{b}} 地块', { a: farmData?.plot_count || 0, b: farmData?.max_plots || 0 });
  }, [matureCount, eventCount, growingCount, plots.length, farmData, t]);

  const taskSummary = farmData?.task_summary || { done: 0, total: 0 };
  const taskSubtitle = taskSummary.total > 0
    ? t('{{a}}/{{b}} 完成', { a: taskSummary.done, b: taskSummary.total })
    : t('暂无任务');
  const taskBadge = (taskSummary.total - taskSummary.done) > 0
    ? (taskSummary.total - taskSummary.done)
    : null;

  // 牧场详情（alive_count/max_animals）走 /api/ranch 独立 API，Dashboard 为避免额外请求
  // 只展示提示文案；等级≥10 后按钮仍会进入牧场页查看
  const ranchSubtitle = t('喂养动物');

  // 等级锁定判断
  const lockOf = (key) => {
    const req = FEATURE_LEVEL_MAP[key];
    if (req && userLevel < req.level) return req.level;
    return 0;
  };

  const primaryCards = [
    {
      key: 'overview', emoji: '🌾', title: t('我的农田'),
      subtitle: plotSubtitle, accent: '#6dbb5c',
      badge: matureCount > 0 ? matureCount : null,
    },
    {
      key: 'plant', emoji: '🌱', title: t('种植'),
      subtitle: t('播种收获'), accent: '#7bc46f',
    },
    {
      key: 'ranch', emoji: '🐄', title: t('牧场'),
      subtitle: ranchSubtitle, accent: '#d1a03a',
    },
    {
      key: 'fish', emoji: '🎣', title: t('钓鱼'),
      subtitle: t('甩竿赚钱'), accent: '#4f8ff7',
    },
    {
      key: 'workshop', emoji: '🏭', title: t('加工坊'),
      subtitle: t('提升附加值'), accent: '#9a72d8',
    },
    {
      key: 'market', emoji: '📈', title: t('市场行情'),
      subtitle: t('买卖最佳时机'), accent: '#c8922a',
    },
    {
      key: 'shop', emoji: '🏪', title: t('商店'),
      subtitle: t('种子与道具'), accent: '#2fb7bf',
    },
    {
      key: 'warehouse', emoji: '📦', title: t('仓库'),
      subtitle: farmData?.balance != null ? `💰 ${formatBalance(farmData.balance)}` : t('存储物品'),
      accent: '#7f8a78',
    },
    {
      key: 'tasks', emoji: '📝', title: t('今日任务'),
      subtitle: taskSubtitle, accent: '#e07b4c',
      badge: taskBadge,
    },
  ];

  const secondaryRows = [
    { key: 'trading', emoji: '🔄', label: t('交易所') },
    { key: 'entrust', emoji: '🤝', label: t('委托') },
    { key: 'bank', emoji: '🏦', label: t('银行') },
    { key: 'soil', emoji: '🟫', label: t('土壤') },
    { key: 'breeding', emoji: '🧬', label: t('育种') },
    { key: 'treefarm', emoji: '🌲', label: t('树场') },
    { key: 'profile', emoji: '👤', label: t('个人中心') },
    { key: 'level', emoji: '📊', label: t('等级') },
    { key: 'achievements', emoji: '🏆', label: t('成就') },
    { key: 'encyclopedia', emoji: '📖', label: t('图鉴') },
    { key: 'leaderboard', emoji: '🏅', label: t('排行榜') },
    { key: 'friends', emoji: '👥', label: t('好友'), badge: friendRequestCount },
    { key: 'steal', emoji: '🕵️', label: t('偷菜') },
    { key: 'games', emoji: '🎰', label: t('小游戏') },
    { key: 'dog', emoji: '🐕', label: t('狗狗') },
    { key: 'automation', emoji: '⚡', label: t('自动化') },
    { key: 'prestige', emoji: '🔄', label: t('转生') },
    { key: 'logs', emoji: '📜', label: t('日志') },
  ];

  return (
    <div className='farm-home-dashboard'>
      <div className='farm-home-greeting'>
        <div className='farm-home-greeting-kicker'>{t('欢迎回来')}</div>
        <div className='farm-home-greeting-title'>
          ⭐ Lv.{userLevel}
          {farmData?.prestige_level > 0 && (
            <span className='farm-home-greeting-prestige'>· P{farmData.prestige_level}</span>
          )}
        </div>
        {farmData?.weather && (
          <div className='farm-home-greeting-weather'>
            <span>{farmData.weather.emoji}</span>
            <span>{farmData.weather.name}</span>
          </div>
        )}
      </div>

      <div className='farm-home-grid'>
        {primaryCards.map((c) => {
          const lockLv = lockOf(c.key);
          return (
            <Card
              key={c.key}
              emoji={c.emoji}
              title={c.title}
              subtitle={c.subtitle}
              accent={c.accent}
              locked={lockLv > 0}
              lockLevel={lockLv}
              badge={c.badge}
              onClick={() => onNavigate(c.key)}
            />
          );
        })}
      </div>

      <div className='farm-home-section-title'>{t('更多功能')}</div>
      <div className='farm-home-mini-grid'>
        {secondaryRows.map((r) => {
          const lockLv = lockOf(r.key);
          return (
            <MiniRow
              key={r.key}
              emoji={r.emoji}
              label={r.label}
              locked={lockLv > 0}
              lockLevel={lockLv}
              onClick={() => onNavigate(r.key)}
            />
          );
        })}
      </div>
    </div>
  );
};

export default MobileDashboard;
