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

/**
 * 从 localStorage 读 status，如果启用了 recaptcha v3 则拿 token；否则返回空
 * 不抛错，失败时返回空字符串
 */
export async function getFarmRecaptchaV3Token(action) {
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
    return await getRecaptchaV3Token(siteKey, action);
  } catch (_) {
    return '';
  }
}
