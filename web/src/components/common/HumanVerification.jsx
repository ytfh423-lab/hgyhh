import React from 'react';
import Turnstile from 'react-turnstile';
import ReCAPTCHA from 'react-google-recaptcha';

const HumanVerification = ({
  enabled,
  provider,
  siteKey,
  onVerify,
  onExpire,
  widgetKey,
}) => {
  if (!enabled || !siteKey) {
    return null;
  }

  if (provider === 'recaptcha') {
    return (
      <ReCAPTCHA
        key={widgetKey}
        sitekey={siteKey}
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
