import React, { useState } from 'react';
import { ChevronRight, ArrowLeft } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { getLogo } from '../../../helpers';
import { FEATURE_LEVEL_MAP } from '../constants';

const navGroups = [
  {
    key: 'agri',
    label: '种植养殖',
    emoji: '🌾',
    items: [
      { key: 'overview', label: '总览', emoji: '🏠' },
      { key: 'plant', label: '种植', emoji: '🌱' },
      { key: 'ranch', label: '牧场', emoji: '🐄' },
      { key: 'fish', label: '钓鱼', emoji: '🎣' },
      { key: 'workshop', label: '加工坊', emoji: '🏭' },
      { key: 'treefarm', label: '树场', emoji: '🌲' },
    ],
  },
  {
    key: 'economy',
    label: '经济贸易',
    emoji: '💰',
    items: [
      { key: 'market', label: '市场', emoji: '📈' },
      { key: 'shop', label: '商店', emoji: '🏪' },
      { key: 'warehouse', label: '仓库', emoji: '📦' },
      { key: 'trading', label: '交易所', emoji: '🔄' },
      { key: 'bank', label: '银行', emoji: '🏦' },
    ],
  },
  {
    key: 'growth',
    label: '成长进度',
    emoji: '⭐',
    items: [
      { key: 'level', label: '等级', emoji: '📊' },
      { key: 'tasks', label: '任务', emoji: '📝' },
      { key: 'achievements', label: '成就', emoji: '🏆' },
      { key: 'encyclopedia', label: '图鉴', emoji: '📖' },
      { key: 'leaderboard', label: '排行榜', emoji: '🏅' },
    ],
  },
  {
    key: 'fun',
    label: '趣味玩法',
    emoji: '🎮',
    items: [
      { key: 'steal', label: '偷菜', emoji: '🕵️' },
      { key: 'games', label: '小游戏', emoji: '🎰' },
      { key: 'dog', label: '狗狗', emoji: '🐶' },
    ],
  },
  {
    key: 'system',
    label: '系统功能',
    emoji: '⚙️',
    items: [
      { key: 'automation', label: '自动化', emoji: '⚡' },
      { key: 'prestige', label: '转生', emoji: '🔄' },
      { key: 'logs', label: '日志', emoji: '📜' },
    ],
  },
];

export { navGroups };

const Sidebar = ({ activeKey, onNavigate, t, farmData, userLevel = 1 }) => {
  const [collapsed, setCollapsed] = useState({});
  const navigate = useNavigate();

  const toggle = (groupKey) => {
    setCollapsed(prev => ({ ...prev, [groupKey]: !prev[groupKey] }));
  };

  return (
    <nav className='farm-sidebar'>
      <div className='farm-sidebar-header'>
        <div className='farm-sidebar-brand' onClick={() => onNavigate('overview')}>
          <img src={getLogo()} alt='logo' className='farm-sidebar-logo' style={{ objectFit: 'contain' }} />
          <div>
            <div className='farm-sidebar-title'>NPC {t('农场')}</div>
            <div className='farm-sidebar-subtitle'>
              {farmData ? `Lv.${farmData.user_level || 1}` : ''}
              {farmData?.prestige_level > 0 ? ` · P${farmData.prestige_level}` : ''}
            </div>
          </div>
        </div>
      </div>
      <div className='farm-sidebar-nav'>
        {navGroups.map((group) => (
          <div key={group.key} className='farm-nav-group'>
            <div className='farm-nav-header' onClick={() => toggle(group.key)}>
              <span>{group.emoji}</span>
              <span>{t(group.label)}</span>
              <ChevronRight size={12} className={`chevron ${collapsed[group.key] ? '' : 'open'}`} />
            </div>
            {!collapsed[group.key] && group.items.map((item) => {
              const req = FEATURE_LEVEL_MAP[item.key];
              const locked = req && userLevel < req.level;
              return (
                <div
                  key={item.key}
                  data-tutorial={`nav-${item.key}`}
                  className={`farm-nav-item ${activeKey === item.key ? 'active' : ''} ${locked ? 'locked' : ''}`}
                  onClick={locked ? undefined : () => onNavigate(item.key)}
                  title={locked ? `${t('需要')} Lv.${req.level} ${t('解锁')}` : ''}
                >
                  <span style={{ fontSize: 15 }}>{locked ? '🔒' : item.emoji}</span>
                  <span>{t(item.label)}</span>
                  {item.key === 'tasks' && !locked && farmData?.task_summary && (
                    <span className='farm-pill' style={{
                      marginLeft: 'auto',
                      fontSize: 10,
                      padding: '1px 6px',
                      background: farmData.task_summary.done >= farmData.task_summary.total
                        ? 'var(--farm-leaf)' : 'var(--farm-sky)',
                      color: '#fff',
                      borderRadius: 8,
                      fontWeight: 700,
                      lineHeight: '16px',
                    }}>
                      {farmData.task_summary.done}/{farmData.task_summary.total}
                    </span>
                  )}
                  {locked && <span className='nav-lock'>Lv.{req.level}</span>}
                </div>
              );
            })}
          </div>
        ))}
      </div>
      <div className='farm-sidebar-footer'>
        <div className='farm-nav-item' onClick={() => navigate('/')} style={{ paddingLeft: 14 }}>
          <ArrowLeft size={14} style={{ color: 'var(--farm-sb-text-dim)' }} />
          <span>{t('返回控制台')}</span>
        </div>
      </div>
    </nav>
  );
};

export default Sidebar;
