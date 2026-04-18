import React, { useEffect } from 'react';
import { ArrowLeft, Search } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { getLogo } from '../../../helpers';
import { FEATURE_LEVEL_MAP } from '../constants';

const navGroups = [
  {
    key: 'agri',
    label: '种植养殖',
    emoji: '\u{1F331}',
    accent: '#6dbb5c',
    soft: 'rgba(109, 187, 92, 0.16)',
    items: [
      { key: 'overview', label: '总览', emoji: '🏠' },
      { key: 'plant', label: '种植', emoji: '🌱' },
      { key: 'soil', label: '土壤', emoji: '🟫' },
      { key: 'ranch', label: '牧场', emoji: '🐄' },
      { key: 'breeding', label: '育种', emoji: '🧬' },
      { key: 'fish', label: '钓鱼', emoji: '🎣' },
      { key: 'workshop', label: '加工坊', emoji: '🏭' },
      { key: 'treefarm', label: '树场', emoji: '🌲' },
    ],
  },
  {
    key: 'economy',
    label: '经济贸易',
    emoji: '💰',
    accent: '#d1a03a',
    soft: 'rgba(209, 160, 58, 0.16)',
    items: [
      { key: 'market', label: '市场', emoji: '📈' },
      { key: 'shop', label: '商店', emoji: '🏪' },
      { key: 'warehouse', label: '仓库', emoji: '📦' },
      { key: 'trading', label: '交易所', emoji: '🔄' },
      { key: 'entrust', label: '委托', emoji: '🤝' },
      { key: 'bank', label: '银行', emoji: '🏦' },
    ],
  },
  {
    key: 'growth',
    label: '成长进度',
    emoji: '📈',
    accent: '#4f8ff7',
    soft: 'rgba(79, 143, 247, 0.14)',
    items: [
      { key: 'profile', label: '个人中心', emoji: '👤' },
      { key: 'level', label: '等级', emoji: '📊' },
      { key: 'tasks', label: '任务', emoji: '📝' },
      { key: 'achievements', label: '成就', emoji: '🏆' },
      { key: 'encyclopedia', label: '图鉴', emoji: '📖' },
      { key: 'leaderboard', label: '排行榜', emoji: '🏅' },
    ],
  },
  {
    key: 'social',
    label: '好友',
    emoji: '\u{1F465}',
    accent: '#2fb7bf',
    soft: 'rgba(47, 183, 191, 0.14)',
    items: [
      { key: 'friends', label: '好友', emoji: '\u{1F46B}' },
    ],
  },
  {
    key: 'fun',
    label: '趣味玩法',
    emoji: '\u{1F3B2}',
    accent: '#9a72d8',
    soft: 'rgba(154, 114, 216, 0.14)',
    items: [
      { key: 'steal', label: '偷菜', emoji: '\u{1F575}\u{FE0F}' },
      { key: 'games', label: '小游戏', emoji: '🎰' },
      { key: 'dog', label: '狗狗', emoji: '\u{1F436}' },
    ],
  },
  {
    key: 'system',
    label: '系统功能',
    emoji: '⚙️',
    accent: '#7f8a78',
    soft: 'rgba(127, 138, 120, 0.16)',
    items: [
      { key: 'automation', label: '自动化', emoji: '⚡' },
      { key: 'prestige', label: '转生', emoji: '🔄' },
      { key: 'logs', label: '日志', emoji: '📜' },
      { key: 'feedback', label: '留言板', emoji: '💬', href: '/feedback' },
    ],
  },
];

export { navGroups };

// compact 模式渲染：只显示分组图标条，hover 时浮出子菜单；
// 点击分组图标 = 直接跳到该组第一个未锁定子项；
// 单独的「主页」图标在最顶部常驻。
const CompactNav = ({ activeKey, onNavigate, t, userLevel, friendRequestCount, farmData, navigate }) => {
  const firstUnlocked = (group) => {
    for (const item of group.items) {
      const req = FEATURE_LEVEL_MAP[item.key];
      if (!(req && userLevel < req.level)) return item;
    }
    return group.items[0];
  };

  const handleGroupClick = (group) => {
    const item = firstUnlocked(group);
    if (item) {
      if (item.href) navigate(item.href);
      else onNavigate(item.key);
    }
  };

  const groupContainsActive = (group) =>
    group.items.some((item) => item.key === activeKey);

  // 鼠标进入图标时，根据图标在视口的位置决定浮层向上 / 向下对齐
  // 防止在屏幕底部浮层被截断看不到
  const handleIconEnter = (e) => {
    const icon = e.currentTarget;
    const popover = icon.querySelector('.farm-compact-popover');
    if (!popover) return;
    const iconRect = icon.getBoundingClientRect();
    // 先清除类好测真实高度
    popover.classList.remove('align-bottom');
    const popoverHeight = popover.offsetHeight || 240;
    const spaceBelow = window.innerHeight - iconRect.top;
    if (spaceBelow < popoverHeight + 20) {
      popover.classList.add('align-bottom');
    }
  };

  return (
    <div className='farm-sidebar-nav farm-sidebar-nav-compact'>
      <div
        className={`farm-compact-icon ${activeKey === 'home' ? 'active' : ''}`}
        onClick={() => onNavigate('home')}
        title={t('主页')}
      >
        <span className='farm-compact-icon-emoji'>🏠</span>
      </div>
      {navGroups.map((group) => {
        const unread = group.key === 'social' ? friendRequestCount : 0;
        return (
          <div
            key={group.key}
            className={`farm-compact-icon ${groupContainsActive(group) ? 'active' : ''}`}
            style={{
              '--farm-nav-group-accent': group.accent,
              '--farm-nav-group-soft': group.soft,
            }}
            onClick={() => handleGroupClick(group)}
            onMouseEnter={handleIconEnter}
            title={t(group.label)}
          >
            <span className='farm-compact-icon-emoji'>{group.emoji}</span>
            {unread > 0 && <span className='farm-compact-icon-dot' />}
            {/* Hover 浮层：展示该组所有子项 */}
            <div className='farm-compact-popover'>
              <div className='farm-compact-popover-title'>{group.emoji} {t(group.label)}</div>
              {group.items.map((item) => {
                const req = FEATURE_LEVEL_MAP[item.key];
                const locked = req && userLevel < req.level;
                return (
                  <div
                    key={item.key}
                    className={`farm-compact-popover-item ${activeKey === item.key ? 'active' : ''} ${locked ? 'locked' : ''}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      if (locked) return;
                      if (item.href) navigate(item.href);
                      else onNavigate(item.key);
                    }}
                  >
                    <span className='farm-compact-popover-emoji'>{locked ? '🔒' : item.emoji}</span>
                    <span>{t(item.label)}</span>
                    {locked && <span className='farm-compact-popover-lock'>Lv.{req.level}</span>}
                    {item.key === 'friends' && friendRequestCount > 0 && (
                      <span className='farm-compact-popover-badge'>{friendRequestCount}</span>
                    )}
                    {item.key === 'tasks' && !locked && farmData?.task_summary && (
                      <span className='farm-compact-popover-badge' style={{
                        background: farmData.task_summary.done >= farmData.task_summary.total
                          ? 'var(--farm-leaf)' : 'var(--farm-sky)',
                      }}>
                        {farmData.task_summary.done}/{farmData.task_summary.total}
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
};

// 新样式：始终 compact 图标侧边栏，无切换入口。
// 完整的页面列表通过 Ctrl+K 命令面板或 hover 图标浮层访问。
const Sidebar = ({ activeKey, onNavigate, t, farmData, userLevel = 1, friendRequestCount = 0, onOpenCommand }) => {
  const navigate = useNavigate();

  // 同步 body class 让 CSS 变量 --farm-sidebar-w 生效（60px）
  useEffect(() => {
    document.body.classList.add('farm-sidebar-compact');
    return () => document.body.classList.remove('farm-sidebar-compact');
  }, []);

  return (
    <nav className='farm-sidebar is-compact'>
      <div className='farm-sidebar-header'>
        <div
          className='farm-sidebar-brand'
          onClick={() => onNavigate('home')}
          title={t('主页')}
        >
          <img src={getLogo()} alt='logo' className='farm-sidebar-logo' style={{ objectFit: 'contain' }} />
        </div>
      </div>
      {onOpenCommand && (
        <div
          className='farm-sidebar-search-trigger'
          onClick={(e) => { e.stopPropagation(); onOpenCommand(); }}
          title={`${t('搜索页面')} (Ctrl+K)`}
        >
          <Search size={18} />
        </div>
      )}
      <CompactNav
        activeKey={activeKey}
        onNavigate={onNavigate}
        t={t}
        userLevel={userLevel}
        friendRequestCount={friendRequestCount}
        farmData={farmData}
        navigate={navigate}
      />
      <div className='farm-sidebar-footer'>
        <div
          className='farm-nav-item'
          onClick={() => navigate('/')}
          style={{ paddingLeft: 0, justifyContent: 'center' }}
          title={t('返回控制台')}
        >
          <ArrowLeft size={14} style={{ color: 'var(--farm-sb-text-dim)' }} />
        </div>
      </div>
    </nav>
  );
};

export default Sidebar;
