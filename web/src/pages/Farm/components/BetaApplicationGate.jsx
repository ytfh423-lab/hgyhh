import React, { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, Input, Typography, Spin } from '@douyinfe/semi-ui';
import { Lock, Clock, FileText, CheckCircle, XCircle, Send } from 'lucide-react';
import { API, showError, showSuccess } from '../../../helpers';
import { Link } from 'react-router-dom';

const { TextArea } = Input;
const { Text } = Typography;

const btnStyle = {
  padding: '10px 28px', borderRadius: 8, border: '1px solid rgba(251,191,36,0.3)',
  background: 'linear-gradient(135deg, var(--farm-harvest), var(--farm-soil))', color: '#fff',
  fontWeight: 700, fontSize: 14, cursor: 'pointer',
};

const btnDisabledStyle = {
  ...btnStyle,
  opacity: 0.5, cursor: 'not-allowed',
};

const cardStyle = {
  textAlign: 'center', padding: '48px 32px', maxWidth: 480,
  background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(251,191,36,0.12)',
  borderRadius: 16,
};

const formatTime = (ts) => {
  if (!ts) return '';
  const d = new Date(ts * 1000);
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
};

const BetaApplicationGate = () => {
  const { t } = useTranslation();
  const [appData, setAppData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [modalVisible, setModalVisible] = useState(false);
  const [reason, setReason] = useState('');
  const [linuxdoUrl, setLinuxdoUrl] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const loadStatus = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/beta/application/status');
      if (res.success) {
        setAppData(res.data);
      }
    } catch (e) { /* ignore */ }
    setLoading(false);
  }, []);

  useEffect(() => { loadStatus(); }, [loadStatus]);

  const handleSubmit = async () => {
    const trimReason = reason.trim();
    const trimUrl = linuxdoUrl.trim();

    if (trimReason.length < 10) {
      showError(t('申请理由不能少于10个字'));
      return;
    }
    if (trimReason.length > 300) {
      showError(t('申请理由不能超过300个字'));
      return;
    }
    if (trimUrl && !trimUrl.startsWith('https://linux.do/') && !trimUrl.startsWith('https://www.linux.do/')) {
      showError(t('LinuxDo 链接格式不正确，请填写完整的个人主页链接'));
      return;
    }

    setSubmitting(true);
    try {
      const { data: res } = await API.post('/api/farm/beta/application/apply', {
        reason: trimReason,
        linuxdo_profile: trimUrl,
      });
      if (res.success) {
        showSuccess(t('申请已提交，请等待审核结果'));
        setModalVisible(false);
        setReason('');
        setLinuxdoUrl('');
        loadStatus();
      } else {
        showError(res.message);
      }
    } catch (e) {
      showError(t('提交失败，请稍后重试'));
    }
    setSubmitting(false);
  };

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#0a0a0a' }}>
        <Spin size='large' />
      </div>
    );
  }

  const status = appData?.app_status || 'not_applied';
  const canApply = appData?.can_apply !== false;

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#0a0a0a' }}>
      <div style={cardStyle}>

        {/* 状态1：未申请 */}
        {status === 'not_applied' && (
          <>
            <Lock size={44} style={{ color: '#fbbf24', marginBottom: 16 }} />
            <h2 style={{ color: '#fde68a', fontSize: 22, fontWeight: 700, marginBottom: 8 }}>
              {t('你当前暂无农场内测资格')}
            </h2>
            <p style={{ color: '#a8a29e', fontSize: 14, lineHeight: 1.6, marginBottom: 24 }}>
              {t('农场目前处于内测阶段，你可以提交申请获得内测资格。')}
            </p>
            <button style={btnStyle} onClick={() => setModalVisible(true)}>
              {t('申请内测资格')}
            </button>
            <p style={{ color: '#78716c', fontSize: 12, marginTop: 12, lineHeight: 1.5 }}>
              {t('之前没有预约的，也可以通过申请获得内测资格')}
            </p>
          </>
        )}

        {/* 状态2：待审核 */}
        {status === 'pending' && (
          <>
            <Clock size={44} style={{ color: '#fbbf24', marginBottom: 16 }} />
            <h2 style={{ color: '#fde68a', fontSize: 22, fontWeight: 700, marginBottom: 8 }}>
              {t('你已提交农场内测资格申请')}
            </h2>

            <div style={{
              background: 'rgba(251,191,36,0.08)', border: '1px solid rgba(251,191,36,0.2)',
              borderRadius: 12, padding: '16px 20px', marginBottom: 20, textAlign: 'left',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                <FileText size={16} style={{ color: '#fbbf24' }} />
                <span style={{ color: '#fde68a', fontWeight: 600, fontSize: 14 }}>{t('当前审核进度')}</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#fbbf24', animation: 'pulse 2s infinite' }} />
                <span style={{ color: '#fde68a', fontSize: 14, fontWeight: 600 }}>{t('待审核')}</span>
              </div>
              {appData?.submitted_at > 0 && (
                <p style={{ color: '#a8a29e', fontSize: 12, marginTop: 8, marginBottom: 0 }}>
                  {t('提交时间')}: {formatTime(appData.submitted_at)}
                </p>
              )}
            </div>

            {appData?.linuxdo_profile ? (
              <p style={{ color: '#a8a29e', fontSize: 12, lineHeight: 1.5, marginBottom: 16 }}>
                {t('审核通过后，管理员将通过你提供的 LinuxDo 链接手动私信通知')}
              </p>
            ) : (
              <p style={{ color: '#78716c', fontSize: 12, lineHeight: 1.5, marginBottom: 16 }}>
                {t('你未填写 LinuxDo 链接，审核通过后将不做私信通知')}
              </p>
            )}

            <button style={btnDisabledStyle} disabled>
              {t('申请审核中...')}
            </button>
          </>
        )}

        {/* 状态3：已拒绝 */}
        {status === 'rejected' && (
          <>
            <XCircle size={44} style={{ color: '#ef4444', marginBottom: 16 }} />
            <h2 style={{ color: '#fca5a5', fontSize: 22, fontWeight: 700, marginBottom: 8 }}>
              {t('你的申请暂未通过审核')}
            </h2>

            <div style={{
              background: 'rgba(239,68,68,0.08)', border: '1px solid rgba(239,68,68,0.2)',
              borderRadius: 12, padding: '16px 20px', marginBottom: 20, textAlign: 'left',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                <FileText size={16} style={{ color: '#ef4444' }} />
                <span style={{ color: '#fca5a5', fontWeight: 600, fontSize: 14 }}>{t('当前审核进度')}</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#ef4444' }} />
                <span style={{ color: '#fca5a5', fontSize: 14, fontWeight: 600 }}>{t('已拒绝')}</span>
              </div>
              {appData?.review_note && (
                <p style={{ color: '#a8a29e', fontSize: 12, marginTop: 8, marginBottom: 0 }}>
                  {t('审核备注')}: {appData.review_note}
                </p>
              )}
              {appData?.reviewed_at > 0 && (
                <p style={{ color: '#78716c', fontSize: 12, marginTop: 4, marginBottom: 0 }}>
                  {t('审核时间')}: {formatTime(appData.reviewed_at)}
                </p>
              )}
            </div>

            {canApply ? (
              <>
                <button style={btnStyle} onClick={() => setModalVisible(true)}>
                  {t('重新申请')}
                </button>
                <p style={{ color: '#78716c', fontSize: 12, marginTop: 12, lineHeight: 1.5 }}>
                  {t('第')} {(appData?.application_round || 1)}/3 {t('次申请')}
                </p>
              </>
            ) : (
              <>
                <button style={btnDisabledStyle} disabled>
                  {t('暂时无法重新申请')}
                </button>
                {appData?.retry_after && (
                  <p style={{ color: '#78716c', fontSize: 12, marginTop: 12, lineHeight: 1.5 }}>
                    {t('可在')} {formatTime(appData.retry_after)} {t('后重新申请')}
                  </p>
                )}
              </>
            )}
          </>
        )}

        <div style={{ marginTop: 20 }}>
          <Link to='/'>
            <button style={{
              padding: '8px 20px', borderRadius: 8, border: '1px solid rgba(255,255,255,0.1)',
              background: 'transparent', color: '#a8a29e', fontWeight: 500, fontSize: 13, cursor: 'pointer',
            }}>
              {t('返回首页')}
            </button>
          </Link>
        </div>
      </div>

      {/* 申请弹窗 */}
      <Modal
        title={t('申请农场内测资格')}
        visible={modalVisible}
        onCancel={() => { if (!submitting) setModalVisible(false); }}
        footer={null}
        width={480}
        closable={!submitting}
        maskClosable={!submitting}
      >
        <div style={{ marginBottom: 16 }}>
          <Text type='tertiary' size='small' style={{ lineHeight: 1.6 }}>
            {t('请填写你的申请理由，并补充 LinuxDo 论坛个人主页链接。')}
            <br />
            {t('审核通过后，管理员将会手动私信通知；如未填写链接，将不做通知。')}
          </Text>
        </div>

        <div style={{ marginBottom: 16 }}>
          <Text strong size='small' style={{ display: 'block', marginBottom: 6 }}>{t('申请理由')}</Text>
          <TextArea
            value={reason}
            onChange={setReason}
            placeholder={t('请输入你的申请理由，例如你为什么想体验农场玩法、你会如何参与测试、你愿意提供哪些反馈')}
            maxCount={300}
            showClear
            autosize={{ minRows: 4, maxRows: 8 }}
          />
          <Text type='tertiary' size='small' style={{ marginTop: 4, display: 'block' }}>
            {reason.length}/300 {t('字')}（{t('最少10字')}）
          </Text>
        </div>

        <div style={{ marginBottom: 20 }}>
          <Text strong size='small' style={{ display: 'block', marginBottom: 6 }}>
            LinuxDo {t('论坛个人主页链接')}
            <Text type='tertiary' size='small' style={{ marginLeft: 6 }}>（{t('选填')}）</Text>
          </Text>
          <Input
            value={linuxdoUrl}
            onChange={setLinuxdoUrl}
            placeholder='https://linux.do/u/your-username'
            showClear
          />
          <Text type='tertiary' size='small' style={{ marginTop: 4, display: 'block', lineHeight: 1.5 }}>
            {t('请填写你的 LinuxDo 论坛个人主页链接，审核通过后管理员将会手动私信通知；如不填写，将不做通知。')}
          </Text>
        </div>

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12 }}>
          <button
            onClick={() => setModalVisible(false)}
            disabled={submitting}
            style={{
              padding: '8px 20px', borderRadius: 8, border: '1px solid rgba(255,255,255,0.15)',
              background: 'transparent', color: '#a8a29e', fontWeight: 500, fontSize: 13, cursor: 'pointer',
            }}
          >
            {t('取消')}
          </button>
          <button
            onClick={handleSubmit}
            disabled={submitting}
            style={{
              ...btnStyle,
              padding: '8px 24px',
              display: 'flex', alignItems: 'center', gap: 6,
              opacity: submitting ? 0.6 : 1,
            }}
          >
            <Send size={14} />
            {submitting ? t('提交中...') : t('提交申请')}
          </button>
        </div>
      </Modal>
    </div>
  );
};

export default BetaApplicationGate;
