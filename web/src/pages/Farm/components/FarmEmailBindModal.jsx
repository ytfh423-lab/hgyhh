import React from 'react';
import { Button, Input } from '@douyinfe/semi-ui';
import { IconMail, IconKey } from '@douyinfe/semi-icons';
import HumanVerification from '../../../components/common/HumanVerification';

const FarmEmailBindModal = ({
  t,
  visible,
  onClose,
  inputs,
  onInputChange,
  onSendCode,
  onSubmit,
  disableButton,
  loading,
  countdown,
  humanVerificationEnabled,
  humanVerificationProvider,
  humanVerificationSiteKey,
  setHumanVerificationToken,
}) => {
  if (!visible) return null;

  return (
    <div
      className='farm-email-modal-overlay'
      onClick={(e) => {
        if (e.target === e.currentTarget && !loading) {
          onClose();
        }
      }}
    >
      <div className='farm-email-modal'>
        <div className='farm-email-modal-header'>
          <div className='farm-email-modal-header-main'>
            <span className='farm-email-modal-emoji'>📧</span>
            <div>
              <div className='farm-email-modal-title'>{t('绑定邮箱地址')}</div>
              <div className='farm-email-modal-subtitle'>
                {t('绑定后即可在农场内接收活动提醒邮件')}
              </div>
            </div>
          </div>
          {!loading && (
            <button
              type='button'
              className='farm-email-modal-close'
              onClick={onClose}
              aria-label={t('关闭')}
            >
              ✕
            </button>
          )}
        </div>

        <div className='farm-email-modal-body'>
          <div className='farm-email-modal-tip farm-pill farm-pill-blue'>
            {t('绑定逻辑与个人中心一致，需先获取邮箱验证码后再完成绑定。')}
          </div>
          <div className='farm-email-modal-row'>
            <Input
              placeholder={t('输入邮箱地址')}
              onChange={(value) => onInputChange('email', value)}
              value={inputs.email}
              name='email'
              type='email'
              size='large'
              className='farm-email-modal-input'
              prefix={<IconMail />}
              disabled={loading}
            />
            <Button
              onClick={onSendCode}
              disabled={disableButton || loading}
              className='farm-btn farm-email-modal-send'
              theme='light'
              size='large'
            >
              {disableButton
                ? `${t('重新发送')} (${countdown})`
                : t('获取验证码')}
            </Button>
          </div>

          <Input
            placeholder={t('验证码')}
            name='email_verification_code'
            value={inputs.email_verification_code}
            onChange={(value) => onInputChange('email_verification_code', value)}
            size='large'
            className='farm-email-modal-input'
            prefix={<IconKey />}
            disabled={loading}
          />

          {humanVerificationEnabled && (
            <div className='farm-email-modal-turnstile'>
              <HumanVerification
                provider={humanVerificationProvider}
                enabled={humanVerificationEnabled}
                siteKey={humanVerificationSiteKey}
                onVerify={setHumanVerificationToken}
              />
            </div>
          )}
        </div>

        <div className='farm-email-modal-footer'>
          <Button theme='light' onClick={onClose} className='farm-btn' disabled={loading}>
            {t('取消')}
          </Button>
          <Button theme='solid' onClick={onSubmit} loading={loading} className='farm-btn'>
            {t('立即绑定')}
          </Button>
        </div>
      </div>
    </div>
  );
};

export default FarmEmailBindModal;
