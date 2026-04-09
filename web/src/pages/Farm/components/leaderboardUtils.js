import { formatBalance } from './utils';

export const FARM_LEADERBOARD_TYPES = [
  { key: 'balance', icon: '💰', label: '资产' },
  { key: 'level', icon: '⭐', label: '等级' },
  { key: 'harvest', icon: '🌾', label: '收获' },
  { key: 'prestige', icon: '🔄', label: '转生' },
  { key: 'steal', icon: '🕵️', label: '偷菜' },
];

export const FARM_LEADERBOARD_SCOPES = [
  { key: 'global', icon: '🌍', label: '全服' },
  { key: 'friends', icon: '👫', label: '好友' },
];

export const FARM_LEADERBOARD_PERIODS = [
  { key: 'all', icon: '🏆', label: '总榜' },
  { key: 'weekly', icon: '📅', label: '周榜' },
];

const FARM_LEADERBOARD_REWARDS = {
  1: { emoji: '👑', title: '冠军勋章', shortTitle: '冠军勋章' },
  2: { emoji: '🥈', title: '亚军勋章', shortTitle: '亚军勋章' },
  3: { emoji: '🥉', title: '季军勋章', shortTitle: '季军勋章' },
};

export const getFarmLeaderboardReward = (rank) => FARM_LEADERBOARD_REWARDS[rank] || null;

export const formatFarmLeaderboardValue = (boardType, value) => {
  if (value == null) {
    return boardType === 'balance' || boardType === 'steal' ? formatBalance(0) : '0';
  }
  if (boardType === 'balance' || boardType === 'steal') {
    return formatBalance(value);
  }
  return `${Math.round(value)}`;
};
