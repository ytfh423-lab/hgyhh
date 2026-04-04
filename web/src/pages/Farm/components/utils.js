export { API, showError, showSuccess } from '../../../helpers';
export { farmConfirm as confirmAction } from './farmConfirm';

export const formatDuration = (secs) => {
  if (!secs || secs <= 0) return '0分';
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  if (h > 0) return `${h}时${m}分`;
  return `${m}分`;
};

export const formatBalance = (val) => {
  if (val == null) return '$0.00';
  if (val >= 1e12) return `$${(val / 1e12).toFixed(2)}T`;
  if (val >= 1e9) return `$${(val / 1e9).toFixed(2)}B`;
  if (val >= 1e6) return `$${(val / 1e6).toFixed(2)}M`;
  if (val >= 1e4) return `$${(val / 1e3).toFixed(2)}K`;
  return `$${val.toFixed(2)}`;
};

export const seasonNames = ['春', '夏', '秋', '冬'];
export const seasonEmojis = ['🌸', '☀️', '🍂', '❄️'];

export const CHART_PALETTE = [
  '#5a8fb4', '#b84233', '#4a7c3f', '#c8922a', '#8a6cb0', '#a0845e',
  '#6fa85e', '#d4956a', '#5a8fb4', '#7a9e4e', '#b84233', '#8a6cb0',
];
