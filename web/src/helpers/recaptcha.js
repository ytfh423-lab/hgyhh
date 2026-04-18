/**
 * reCAPTCHA v3 脚本加载 & token 获取工具
 *
 * - loadRecaptchaV3Script(siteKey): 单例加载 v3 脚本，返回 Promise
 * - getRecaptchaV3Token(siteKey, action): 加载脚本 + 执行获取 token
 * - getFarmRecaptchaV3Token(action): 从 localStorage 读 status，若配置为 recaptcha 则拿 v3 token
 */

const v3ScriptCache = new Map(); // siteKey -> Promise<void>

export function loadRecaptchaV3Script(siteKey) {
  if (!siteKey) return Promise.reject(new Error('missing siteKey'));
  if (v3ScriptCache.has(siteKey)) return v3ScriptCache.get(siteKey);

  const p = new Promise((resolve, reject) => {
    if (typeof window === 'undefined') {
      reject(new Error('no window'));
      return;
    }
    const existing = document.querySelector(`script[data-recaptcha-v3="${siteKey}"]`);
    if (existing) {
      if (window.grecaptcha && window.grecaptcha.execute) {
        window.grecaptcha.ready(() => resolve());
        return;
      }
      existing.addEventListener('load', () => {
        if (window.grecaptcha && window.grecaptcha.ready) {
          window.grecaptcha.ready(() => resolve());
        } else {
          reject(new Error('grecaptcha not ready'));
        }
      });
      existing.addEventListener('error', () => reject(new Error('recaptcha v3 script load failed')));
      return;
    }
    const script = document.createElement('script');
    script.src = `https://www.google.com/recaptcha/api.js?render=${encodeURIComponent(siteKey)}`;
    script.async = true;
    script.defer = true;
    script.setAttribute('data-recaptcha-v3', siteKey);
    script.onload = () => {
      if (window.grecaptcha && window.grecaptcha.ready) {
        window.grecaptcha.ready(() => resolve());
      } else {
        reject(new Error('grecaptcha not available after load'));
      }
    };
    script.onerror = () => reject(new Error('recaptcha v3 script load failed'));
    document.head.appendChild(script);
  });
  v3ScriptCache.set(siteKey, p);
  return p;
}

export async function getRecaptchaV3Token(siteKey, action) {
  if (!siteKey) throw new Error('missing siteKey');
  await loadRecaptchaV3Script(siteKey);
  return new Promise((resolve, reject) => {
    try {
      window.grecaptcha
        .execute(siteKey, { action: action || 'farm' })
        .then((token) => resolve(token || ''), (err) => reject(err));
    } catch (err) {
      reject(err);
    }
  });
}

// v3 token 缓存：同 action 在 90 秒内复用（Google 官方 2 分钟有效，留 30 秒安全窗口）
// 这样同一个会话里农场连续操作不再每次都往返 Google 拿 token，请求延迟显著下降
const v3TokenCache = new Map(); // action -> { token, expireAt }
const V3_TOKEN_TTL_MS = 90 * 1000;

/**
 * 从 localStorage 读 status，如果启用了 recaptcha v3 则拿 token；否则返回空
 * 不抛错，失败/超时返回空字符串（请求继续，后端走 burst 兜底）
 *
 * @param {string} action - reCAPTCHA v3 action 名，用于 action 校验
 * @param {number} timeoutMs - 超时毫秒数，默认 300ms（大多数农场操作可容忍，
 *                              超时后 token 为空请求继续，后端走 burst 兜底）
 */
export async function getFarmRecaptchaV3Token(action, timeoutMs = 300) {
  try {
    if (typeof window === 'undefined') return '';
    const raw = localStorage.getItem('status');
    if (!raw) return '';
    let status;
    try {
      status = JSON.parse(raw);
    } catch (_) {
      return '';
    }
    const enabled =
      status?.human_verification_enabled ?? status?.turnstile_check ?? false;
    const provider = status?.human_verification_provider || 'turnstile';
    const siteKey =
      status?.human_verification_site_key || status?.turnstile_site_key || '';
    if (!enabled || provider !== 'recaptcha' || !siteKey) return '';

    // 命中缓存：直接返回，零延迟
    const cacheKey = `${siteKey}:${action || 'farm'}`;
    const cached = v3TokenCache.get(cacheKey);
    const now = Date.now();
    if (cached && cached.expireAt > now) {
      return cached.token;
    }

    const tokenPromise = getRecaptchaV3Token(siteKey, action).then((t) => {
      if (t) {
        v3TokenCache.set(cacheKey, { token: t, expireAt: Date.now() + V3_TOKEN_TTL_MS });
      }
      return t;
    });
    const timeoutPromise = new Promise((resolve) =>
      setTimeout(() => resolve(''), timeoutMs),
    );
    return await Promise.race([tokenPromise, timeoutPromise]);
  } catch (_) {
    return '';
  }
}
