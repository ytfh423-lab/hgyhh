import React, { useCallback, useEffect, useState } from 'react';
import { Modal, Button, Typography, Toast, Spin } from '@douyinfe/semi-ui';
import { API } from '../../../helpers';

const { Title, Text, Paragraph } = Typography;

// 突发事件 Modal（A-3）
// 负责拉取 pending 事件、展示三选项、提交玩家选择、展示结果。
// 轮询：Overview 挂载期间每 60s 刷新一次，确保新事件能弹出。

const RandomEventModal = ({ t, loadFarm }) => {
  const [pending, setPending] = useState(null);
  const [busy, setBusy]       = useState(false);
  const [result, setResult]   = useState(null);  // { outcome, chosen }

  const fetchPending = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/event/view');
      if (res.success) setPending(res.data.pending || null);
    } catch { /* silent */ }
  }, []);

  useEffect(() => {
    fetchPending();
    const id = setInterval(fetchPending, 60 * 1000);
    // 支持外部立即触发刷新（教程完成、QA 触发等）
    const onExternalRefresh = () => fetchPending();
    window.addEventListener('farm-random-event-refresh', onExternalRefresh);
    return () => {
      clearInterval(id);
      window.removeEventListener('farm-random-event-refresh', onExternalRefresh);
    };
  }, [fetchPending]);

  const choose = async (optIdx) => {
    if (!pending || busy) return;
    setBusy(true);
    try {
      const { data: res } = await API.post('/api/farm/event/choose', {
        event_id: pending.id,
        option_index: optIdx,
      });
      if (res.success) {
        setResult({ outcome: res.data.outcome, chosen: optIdx });
        loadFarm && loadFarm({ silent: true });
      } else {
        Toast.error(res.message || t('操作失败'));
      }
    } catch { Toast.error(t('操作失败')); }
    finally { setBusy(false); }
  };

  const close = () => {
    setPending(null);
    setResult(null);
  };

  if (!pending) return null;

  const opts = pending.options || [];

  return (
    <Modal
      visible
      title={null}
      footer={null}
      closable={!!result}
      maskClosable={false}
      onCancel={close}
      width={480}
    >
      <div style={{ padding: 8 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
          <span style={{ fontSize: 40 }}>{pending.emoji}</span>
          <Title heading={4} style={{ margin: 0 }}>{pending.title}</Title>
        </div>
        <Paragraph style={{ color: 'var(--farm-text-1)', lineHeight: 1.7, marginBottom: 16 }}>
          {pending.narrative}
        </Paragraph>

        {!result ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {opts.map((label, idx) => (
              <Button
                key={idx}
                block
                theme='solid'
                type='primary'
                loading={busy}
                style={{ textAlign: 'left', justifyContent: 'flex-start', padding: '8px 14px', height: 'auto', whiteSpace: 'normal' }}
                onClick={() => choose(idx)}
              >
                {String.fromCharCode(65 + idx)}. {label}
              </Button>
            ))}
          </div>
        ) : (
          <div style={{
            padding: 14,
            background: 'rgba(109,187,92,0.12)',
            border: '1px solid rgba(109,187,92,0.3)',
            borderRadius: 8,
          }}>
            <Text strong style={{ color: 'var(--farm-leaf)' }}>{t('结果')}:</Text>
            <Paragraph style={{ margin: '6px 0 12px 0', lineHeight: 1.6 }}>
              {result.outcome}
            </Paragraph>
            <Button block theme='solid' onClick={close}>{t('知道了')}</Button>
          </div>
        )}
      </div>
    </Modal>
  );
};

export default RandomEventModal;
