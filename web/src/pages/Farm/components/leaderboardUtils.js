import { formatBalance } from './utils';

export const FARM_LEADERBOARD_TYPES = [
  { key: 'balance', icon: '💰', label: '资产' },
  { key: 'level', icon: '⭐', label: '等级' },
  { key: 'harvest', icon: '🌾', label: '最佳收获' },
  { key: 'prestige', icon: '🔄', label: '转生' },
  { key: 'steal', icon: '🕵️', label: '最佳偷菜' },
];

export const FARM_LEADERBOARD_SCOPES = [
  { key: 'global', icon: '🌍', label: '全服' },
  { key: 'friends', icon: '👫', label: '好友' },
];

export const FARM_LEADERBOARD_PERIODS = [
  { key: 'all', icon: '🏆', label: '总榜' },
  { key: 'weekly', icon: '📅', label: '周榜' },
];

// 新老玩家分榜：普通玩家只能看自己所属的 cohort（后端强制），
// 管理员可以切换 old/new/all 对比查看。
export const FARM_LEADERBOARD_COHORTS = [
  { key: 'all', icon: '👥', label: '全部玩家' },
  { key: 'old', icon: '🪵', label: '老玩家榜' },
  { key: 'new', icon: '🌱', label: '新玩家榜' },
];

export const formatFarmLeaderboardValue = (boardType, value, valueKind) => {
  if (value == null) {
    return valueKind === 'quota' || boardType === 'balance' || boardType === 'steal' || boardType === 'harvest'
      ? formatBalance(0)
      : '0';
  }
  if (valueKind === 'quota' || boardType === 'balance' || boardType === 'steal' || boardType === 'harvest') {
    return formatBalance(value);
  }
  return `${Math.round(value)}`;
};
