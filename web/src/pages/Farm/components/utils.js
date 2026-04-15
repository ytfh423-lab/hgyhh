import { API, showError, showSuccess as showGlobalSuccess } from '../../../helpers';

export { API, showError };

const isFarmSuccessRoute = (pathname) => pathname.startsWith('/farm') || pathname.startsWith('/console/farm-');

export const showSuccess = (message, options = {}) => {
  if (typeof window !== 'undefined' && isFarmSuccessRoute(window.location.pathname)) {
    window.dispatchEvent(new CustomEvent('farm:success-notify', {
      detail: {
        message,
        ...options,
      },
    }));
    return;
  }
  showGlobalSuccess(message);
};

export { farmConfirm as confirmAction } from './farmConfirm';

export const formatDuration = (secs) => {
  if (!secs || secs <= 0) return '0分';
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  if (h > 0) return `${h}时${m}分`;
  return `${m}分`;
};

export const formatBalance = (val) => {
  const num = Number(val);
  if (!Number.isFinite(num)) return '$0.00';
  if (num >= 1e12) return `$${(num / 1e12).toFixed(2)}T`;
  if (num >= 1e9) return `$${(num / 1e9).toFixed(2)}B`;
  if (num >= 1e6) return `$${(num / 1e6).toFixed(2)}M`;
  if (num >= 1e4) return `$${(num / 1e3).toFixed(2)}K`;
  return `$${num.toFixed(2)}`;
};

export const seasonNames = ['春', '夏', '秋', '冬'];
export const seasonEmojis = ['🌸', '☀️', '🍂', '❄️'];

export const CHART_PALETTE = [
  '#5a8fb4', '#b84233', '#4a7c3f', '#c8922a', '#8a6cb0', '#a0845e',
  '#6fa85e', '#d4956a', '#5a8fb4', '#7a9e4e', '#b84233', '#8a6cb0',
];
