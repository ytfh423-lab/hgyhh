/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import {
  getUserIdFromLocalStorage,
  showError,
  formatMessageForAPI,
  isValidMessage,
} from './utils';
import axios from 'axios';
import { MESSAGE_ROLES } from '../constants/playground.constants';
import { getFarmRecaptchaV3Token } from './recaptcha';

export let API = axios.create({
  baseURL: import.meta.env.VITE_REACT_APP_SERVER_URL
    ? import.meta.env.VITE_REACT_APP_SERVER_URL
    : 'https://www.codex1.cn',
  headers: {
    'New-API-User': getUserIdFromLocalStorage(),
    'Cache-Control': 'no-store',
  },
});

// 生成 32 字符随机 hex（用于农场防重放 Nonce）
function generateFarmNonce() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID().replace(/-/g, '');
  }
  const arr = new Uint8Array(16);
  crypto.getRandomValues(arr);
  return Array.from(arr, (b) => b.toString(16).padStart(2, '0')).join('');
}

function patchAPIInstance(instance) {
  const originalGet = instance.get.bind(instance);
  const inFlightGetRequests = new Map();

  const genKey = (url, config = {}) => {
    const params = config.params ? JSON.stringify(config.params) : '{}';
    return `${url}?${params}`;
  };

  instance.get = (url, config = {}) => {
    if (config?.skipDeduplication || config?.disableDuplicate === false) {
      return originalGet(url, config);
    }

    const key = genKey(url, config);
    if (inFlightGetRequests.has(key)) {
      return inFlightGetRequests.get(key);
    }

    const reqPromise = originalGet(url, config).finally(() => {
      inFlightGetRequests.delete(key);
    });

    inFlightGetRequests.set(key, reqPromise);
    return reqPromise;
  };

  // 农场防脚本：对 /api/farm、/api/ranch、/api/tree 的写请求自动注入 Nonce
  // 并在启用 reCAPTCHA v3 时异步拿 token 注入 Header，实现"无感风控"
  instance.interceptors.request.use(async (config) => {
    const url = config.url || '';
    const method = (config.method || 'get').toLowerCase();
    const isFarmWrite =
      method !== 'get' &&
      method !== 'head' &&
      method !== 'options' &&
      (url.startsWith('/api/farm') ||
        url.startsWith('/api/ranch') ||
        url.startsWith('/api/tree'));
    if (!isFarmWrite) return config;

    config.headers['X-Farm-Nonce'] = generateFarmNonce();

    // 若请求已显式带上 v2/其他 token（来自 step-up 重试），不再覆盖
    const hasExistingToken = !!config.headers['X-Farm-Captcha-Token'];
    if (hasExistingToken) return config;

    // 异步尝试拿 v3 token（失败返回空字符串，不阻塞请求）
    try {
      const action = url.replace(/^\/api\//, '').replace(/\//g, '_');
      const token = await getFarmRecaptchaV3Token(action);
      if (token) {
        config.headers['X-Farm-Captcha-Token'] = token;
        config.headers['X-Farm-Captcha-Action'] = action;
        config.headers['X-Farm-Captcha-Version'] = 'v3';
      }
    } catch (_) {
      // 静默失败：后端会用 burst 逻辑决定是否 step-up
    }
    return config;
  });

  // 农场 step-up 人机验证拦截：后端返回 FARM_STEP_UP_REQUIRED / FARM_VERIFICATION_FAILED 时
  // 自动弹出验证窗口，验证通过后通过 Header 把 token 附加到原请求并重试
  instance.interceptors.response.use(async (response) => {
    try {
      const data = response?.data;
      const config = response?.config;
      if (!data || typeof data !== 'object' || !config) return response;
      const url = config.url || '';
      const isFarmUrl =
        url.startsWith('/api/farm') ||
        url.startsWith('/api/ranch') ||
        url.startsWith('/api/tree');
      if (!isFarmUrl) return response;
      if (data.success) return response;
      const code = data.code;
      if (code !== 'FARM_STEP_UP_REQUIRED' && code !== 'FARM_VERIFICATION_FAILED') {
        return response;
      }
      if (config._farmStepUpRetried) return response;

      const d = data.data || {};
      const provider = d.provider || 'turnstile';
      const version = d.version || (provider === 'recaptcha' ? 'v3' : '');
      // v2 用 checkbox，v3 用 score（invisible），Turnstile 用默认 checkbox
      let mode = 'checkbox';
      if (provider === 'recaptcha') {
        mode = version === 'v2' ? 'checkbox' : 'score';
      }
      const isV2Fallback = provider === 'recaptcha' && version === 'v2';
      const message = isV2Fallback
        ? (d.v3_reason
            ? `风险检测触发（${d.v3_reason}），请完成人机验证`
            : '当前操作触发风控，请完成人机验证')
        : code === 'FARM_VERIFICATION_FAILED'
        ? '验证未通过，请重新完成人机验证'
        : '当前操作需要完成人机验证';

      const mod = await import('../pages/Farm/components/farmConfirm');
      const result = await mod.farmVerificationConfirm({
        title: isV2Fallback ? '安全验证（补充）' : '安全验证',
        message,
        icon: '🛡️',
        confirmText: '验证并继续',
        verification: {
          enabled: true,
          provider,
          siteKey: d.site_key || '',
          mode,
          action: d.action || '',
          version,
        },
      });
      if (!result || !result.token) {
        // 用户取消：修改 code 避免调用方再次触发 step-up 分支
        try {
          response.data = {
            ...response.data,
            code: 'FARM_STEP_UP_CANCELLED',
            message: response.data?.message || '已取消验证',
          };
        } catch (_) {}
        return response;
      }

      const retryConfig = {
        ...config,
        _farmStepUpRetried: true,
        headers: {
          ...(config.headers || {}),
          'X-Farm-Captcha-Token': result.token,
          'X-Farm-Captcha-Action': d.action || '',
          'X-Farm-Captcha-Version': version || '',
        },
      };
      return instance.request(retryConfig);
    } catch (_) {
      return response;
    }
  });
}

patchAPIInstance(API);

const sharedCacheMemory = new Map();
const sharedCachePending = new Map();

const readSharedCache = (cacheKey, ttlMs) => {
  const now = Date.now();
  const memoryValue = sharedCacheMemory.get(cacheKey);
  if (memoryValue && now-memoryValue.ts < ttlMs) {
    return memoryValue.data;
  }
  try {
    const raw = sessionStorage.getItem(`shared_cache:${cacheKey}`);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw);
    if (!parsed || now-parsed.ts >= ttlMs) {
      sessionStorage.removeItem(`shared_cache:${cacheKey}`);
      return null;
    }
    sharedCacheMemory.set(cacheKey, parsed);
    return parsed.data;
  } catch (_) {
    return null;
  }
};

const writeSharedCache = (cacheKey, data) => {
  const payload = { data, ts: Date.now() };
  sharedCacheMemory.set(cacheKey, payload);
  try {
    sessionStorage.setItem(`shared_cache:${cacheKey}`, JSON.stringify(payload));
  } catch (_) {}
};

const loadSharedCachedResource = async (cacheKey, loader, ttlMs = 30000) => {
  const cachedData = readSharedCache(cacheKey, ttlMs);
  if (cachedData) {
    return { data: cachedData, fromCache: true };
  }
  if (sharedCachePending.has(cacheKey)) {
    return sharedCachePending.get(cacheKey);
  }
  const request = loader()
    .then((response) => {
      if (response?.data?.success) {
        writeSharedCache(cacheKey, response.data);
      }
      return response;
    })
    .finally(() => {
      sharedCachePending.delete(cacheKey);
    });
  sharedCachePending.set(cacheKey, request);
  return request;
};

export const getUserModelsCached = async (ttlMs = 30000) => {
  return loadSharedCachedResource(
    'user-models',
    () => API.get('/api/user/models', { disableDuplicate: true }),
    ttlMs,
  );
};

export const getUserGroupsCached = async (ttlMs = 30000) => {
  return loadSharedCachedResource(
    'user-groups',
    () => API.get('/api/user/self/groups', { disableDuplicate: true }),
    ttlMs,
  );
};

export const invalidateSharedCache = (cacheKey) => {
  sharedCacheMemory.delete(cacheKey);
  sharedCachePending.delete(cacheKey);
  try {
    sessionStorage.removeItem(`shared_cache:${cacheKey}`);
  } catch (_) {}
};

export function updateAPI() {
  API = axios.create({
    baseURL: import.meta.env.VITE_REACT_APP_SERVER_URL
      ? import.meta.env.VITE_REACT_APP_SERVER_URL
      : 'https://www.codex1.cn',
    headers: {
      'New-API-User': getUserIdFromLocalStorage(),
      'Cache-Control': 'no-store',
    },
  });

  patchAPIInstance(API);
}

API.interceptors.response.use(
  (response) => response,
  (error) => {
    // 如果请求配置中显式要求跳过全局错误处理，则不弹出默认错误提示
    if (error.config && error.config.skipErrorHandler) {
      return Promise.reject(error);
    }
    showError(error);
    return Promise.reject(error);
  },
);

// playground

// 构建API请求负载
export const buildApiPayload = (
  messages,
  systemPrompt,
  inputs,
  parameterEnabled,
) => {
  const processedMessages = messages
    .filter(isValidMessage)
    .map(formatMessageForAPI)
    .filter(Boolean);

  // 如果有系统提示，插入到消息开头
  if (systemPrompt && systemPrompt.trim()) {
    processedMessages.unshift({
      role: MESSAGE_ROLES.SYSTEM,
      content: systemPrompt.trim(),
    });
  }

  const payload = {
    model: inputs.model,
    group: inputs.group,
    messages: processedMessages,
    stream: inputs.stream,
  };

  // 添加启用的参数
  const parameterMappings = {
    temperature: 'temperature',
    top_p: 'top_p',
    max_tokens: 'max_tokens',
    frequency_penalty: 'frequency_penalty',
    presence_penalty: 'presence_penalty',
    seed: 'seed',
  };

  Object.entries(parameterMappings).forEach(([key, param]) => {
    const enabled = parameterEnabled[key];
    const value = inputs[param];
    const hasValue = value !== undefined && value !== null;

    if (enabled && hasValue) {
      payload[param] = value;
    }
  });

  return payload;
};

// 处理API错误响应
export const handleApiError = (error, response = null) => {
  const errorInfo = {
    error: error.message || '未知错误',
    timestamp: new Date().toISOString(),
    stack: error.stack,
  };

  if (response) {
    errorInfo.status = response.status;
    errorInfo.statusText = response.statusText;
  }

  if (error.message.includes('HTTP error')) {
    errorInfo.details = '服务器返回了错误状态码';
  } else if (error.message.includes('Failed to fetch')) {
    errorInfo.details = '网络连接失败或服务器无响应';
  }

  return errorInfo;
};

// 处理模型数据
export const processModelsData = (data, currentModel) => {
  const modelOptions = data.map((model) => ({
    label: model,
    value: model,
  }));

  const hasCurrentModel = modelOptions.some(
    (option) => option.value === currentModel,
  );
  const selectedModel =
    hasCurrentModel && modelOptions.length > 0
      ? currentModel
      : modelOptions[0]?.value;

  return { modelOptions, selectedModel };
};

// 处理分组数据
export const processGroupsData = (data, userGroup) => {
  let groupOptions = Object.entries(data).map(([group, info]) => ({
    label:
      info.desc.length > 20 ? info.desc.substring(0, 20) + '...' : info.desc,
    value: group,
    ratio: info.ratio,
    fullLabel: info.desc,
  }));

  if (groupOptions.length === 0) {
    groupOptions = [
      {
        label: '用户分组',
        value: '',
        ratio: 1,
      },
    ];
  } else if (userGroup) {
    const userGroupIndex = groupOptions.findIndex((g) => g.value === userGroup);
    if (userGroupIndex > -1) {
      const userGroupOption = groupOptions.splice(userGroupIndex, 1)[0];
      groupOptions.unshift(userGroupOption);
    }
  }

  return groupOptions;
};

// 原来components中的utils.js

export async function getOAuthState() {
  let path = '/api/oauth/state';
  let affCode = localStorage.getItem('aff');
  if (affCode && affCode.length > 0) {
    path += `?aff=${affCode}`;
  }
  const res = await API.get(path);
  const { success, message, data } = res.data;
  if (success) {
    return data;
  } else {
    showError(message);
    return '';
  }
}

async function prepareOAuthState(options = {}) {
  const { shouldLogout = false } = options;
  if (shouldLogout) {
    try {
      await API.get('/api/user/logout', { skipErrorHandler: true });
    } catch (err) {}
    localStorage.removeItem('user');
    updateAPI();
  }
  return await getOAuthState();
}

export async function onDiscordOAuthClicked(client_id, options = {}) {
  const state = await prepareOAuthState(options);
  if (!state) return;
  const redirect_uri = `${window.location.origin}/oauth/discord`;
  const response_type = 'code';
  const scope = 'identify+openid';
  window.open(
    `https://discord.com/oauth2/authorize?client_id=${client_id}&redirect_uri=${redirect_uri}&response_type=${response_type}&scope=${scope}&state=${state}`,
  );
}

export async function onOIDCClicked(
  auth_url,
  client_id,
  openInNewTab = false,
  options = {},
) {
  const state = await prepareOAuthState(options);
  if (!state) return;
  const url = new URL(auth_url);
  url.searchParams.set('client_id', client_id);
  url.searchParams.set('redirect_uri', `${window.location.origin}/oauth/oidc`);
  url.searchParams.set('response_type', 'code');
  url.searchParams.set('scope', 'openid profile email');
  url.searchParams.set('state', state);
  if (openInNewTab) {
    window.open(url.toString(), '_blank');
  } else {
    window.location.href = url.toString();
  }
}

export async function onGitHubOAuthClicked(github_client_id, options = {}) {
  const state = await prepareOAuthState(options);
  if (!state) return;
  window.open(
    `https://github.com/login/oauth/authorize?client_id=${github_client_id}&state=${state}&scope=user:email`,
  );
}

export async function onLinuxDOOAuthClicked(
  linuxdo_client_id,
  options = { shouldLogout: false },
) {
  const state = await prepareOAuthState(options);
  if (!state) return;
  window.open(
    `https://connect.linux.do/oauth2/authorize?response_type=code&client_id=${linuxdo_client_id}&state=${state}`,
  );
}

/**
 * Initiate custom OAuth login
 * @param {Object} provider - Custom OAuth provider config from status API
 * @param {string} provider.slug - Provider slug (used for callback URL)
 * @param {string} provider.client_id - OAuth client ID
 * @param {string} provider.authorization_endpoint - Authorization URL
 * @param {string} provider.scopes - OAuth scopes (space-separated)
 * @param {Object} options - Options
 * @param {boolean} options.shouldLogout - Whether to logout first
 */
export async function onCustomOAuthClicked(provider, options = {}) {
  const state = await prepareOAuthState(options);
  if (!state) return;
  
  try {
    const redirect_uri = `${window.location.origin}/oauth/${provider.slug}`;
    
    // Check if authorization_endpoint is a full URL or relative path
    let authUrl;
    if (provider.authorization_endpoint.startsWith('http://') || 
        provider.authorization_endpoint.startsWith('https://')) {
      authUrl = new URL(provider.authorization_endpoint);
    } else {
      // Relative path - this is a configuration error, show error message
      console.error('Custom OAuth authorization_endpoint must be a full URL:', provider.authorization_endpoint);
      showError('OAuth 配置错误：授权端点必须是完整的 URL（以 http:// 或 https:// 开头）');
      return;
    }
    
    authUrl.searchParams.set('client_id', provider.client_id);
    authUrl.searchParams.set('redirect_uri', redirect_uri);
    authUrl.searchParams.set('response_type', 'code');
    authUrl.searchParams.set('scope', provider.scopes || 'openid profile email');
    authUrl.searchParams.set('state', state);
    
    window.open(authUrl.toString());
  } catch (error) {
    console.error('Failed to initiate custom OAuth:', error);
    showError('OAuth 登录失败：' + (error.message || '未知错误'));
  }
}

let channelModels = undefined;
export async function loadChannelModels() {
  const res = await API.get('/api/models');
  const { success, data } = res.data;
  if (!success) {
    return;
  }
  channelModels = data;
  localStorage.setItem('channel_models', JSON.stringify(data));
}

export function getChannelModels(type) {
  if (channelModels !== undefined && type in channelModels) {
    if (!channelModels[type]) {
      return [];
    }
    return channelModels[type];
  }
  let models = localStorage.getItem('channel_models');
  if (!models) {
    return [];
  }
  channelModels = JSON.parse(models);
  if (type in channelModels) {
    return channelModels[type];
  }
  return [];
}
