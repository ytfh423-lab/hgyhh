import React, { useEffect, useRef, useState } from 'react';
import Turnstile from 'react-turnstile';
import ReCAPTCHA from 'react-google-recaptcha';
import { getRecaptchaV3Token } from '../../helpers/recaptcha';

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

  // 默认模式：provider='recaptcha' 时默认走 v3(score) —— 项目主要配置 v3 siteKey；
  // 其他 provider（Turnstile）继续走 checkbox；
  // step-up 弹窗等需要显式 v2 checkbox 的场景由调用方传入 mode='checkbox'。
  const effectiveMode = mode || (provider === 'recaptcha' ? 'score' : 'checkbox');

  // reCAPTCHA v3（score 模式）：手动加载脚本并执行，不使用 react-google-recaptcha
  useEffect(() => {
    if (provider !== 'recaptcha' || effectiveMode !== 'score' || !enabled || !siteKey) {
      return undefined;
    }
    let cancelled = false;
    setV3Error('');
    setV3Running(true);
    getRecaptchaV3Token(siteKey, action || 'farm')
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
  }, [action, enabled, effectiveMode, onVerify, provider, siteKey, widgetKey]);

  if (!enabled || !siteKey) {
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
        sitekey={siteKey}
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
      sitekey={siteKey}
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
