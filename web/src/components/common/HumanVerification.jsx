import React, { useEffect, useRef, useState } from 'react';
import Turnstile from 'react-turnstile';
import ReCAPTCHA from 'react-google-recaptcha';
import { getRecaptchaV3Token } from '../../helpers/recaptcha';

// 从 localStorage.status 读取 recaptcha v2 siteKey（后端在 /api/status 返回）
function readRecaptchaV2SiteKey() {
  try {
    const raw = typeof window !== 'undefined' ? localStorage.getItem('status') : null;
    if (!raw) return '';
    const status = JSON.parse(raw);
    return status?.recaptcha_v2_site_key || '';
  } catch (_) {
    return '';
  }
}

const HumanVerification = ({
  enabled,
  provider,
  siteKey,
  onVerify,
  onExpire,
  widgetKey,
  mode,
  action,
}) => {
  const recaptchaRef = useRef(null);
  const [v3Error, setV3Error] = useState('');
  const [v3Running, setV3Running] = useState(false);

  // 智能模式判断（调用方未显式传 mode 时）：
  //   provider='recaptcha' + 后端配了 v2 siteKey → v2 checkbox（可见验证，用户体验直观）
  //   provider='recaptcha' 仅有 v3            → v3 score（静默）
  //   provider='turnstile'                     → checkbox
  // 显式传入 mode 的场景（step-up 弹窗等）不受影响。
  let effectiveMode = mode;
  let effectiveSiteKey = siteKey;
  if (!effectiveMode && provider === 'recaptcha') {
    const v2SiteKey = readRecaptchaV2SiteKey();
    if (v2SiteKey) {
      effectiveMode = 'checkbox';
      effectiveSiteKey = v2SiteKey;
    } else {
      effectiveMode = 'score';
    }
  }
  if (!effectiveMode) {
    effectiveMode = 'checkbox';
  }

  // reCAPTCHA v3（score 模式）：手动加载脚本并执行，不使用 react-google-recaptcha
  useEffect(() => {
    if (provider !== 'recaptcha' || effectiveMode !== 'score' || !enabled || !effectiveSiteKey) {
      return undefined;
    }
    let cancelled = false;
    setV3Error('');
    setV3Running(true);
    getRecaptchaV3Token(effectiveSiteKey, action || 'farm')
      .then((token) => {
        if (cancelled) return;
        setV3Running(false);
        onVerify(token || '');
      })
      .catch((err) => {
        if (cancelled) return;
        setV3Running(false);
        setV3Error(err?.message || 'reCAPTCHA v3 执行失败');
        onVerify('');
      });
    return () => {
      cancelled = true;
    };
  }, [action, enabled, effectiveMode, effectiveSiteKey, onVerify, provider, widgetKey]);

  if (!enabled || !effectiveSiteKey) {
    return null;
  }

  if (provider === 'recaptcha') {
    if (effectiveMode === 'score') {
      // v3：隐形验证，无可见 UI，仅在异常时提示
      return (
        <div style={{ fontSize: 12, color: '#888', textAlign: 'center', padding: '6px 0' }}>
          {v3Running && '正在进行人机验证…'}
          {!v3Running && !v3Error && '已通过 reCAPTCHA v3 隐形验证'}
          {v3Error && (
            <span style={{ color: '#c0392b' }}>
              reCAPTCHA v3 执行失败：{v3Error}
            </span>
          )}
        </div>
      );
    }
    return (
      <ReCAPTCHA
        key={widgetKey}
        ref={recaptchaRef}
        sitekey={effectiveSiteKey}
        size='normal'
        onChange={(token) => {
          onVerify(token || '');
        }}
        onExpired={() => {
          onExpire?.();
          onVerify('');
        }}
      />
    );
  }

  return (
    <Turnstile
      key={widgetKey}
      sitekey={effectiveSiteKey}
      onVerify={(token) => {
        onVerify(token);
      }}
      onExpire={() => {
        onExpire?.();
        onVerify('');
      }}
    />
  );
};

export default HumanVerification;
