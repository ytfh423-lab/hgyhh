import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { X, UserPlus, Tractor, MessageCircle } from 'lucide-react';
import { API } from '../../../helpers';
import { useNavigate } from 'react-router-dom';

/* ─── 单条通知 ─── */
const NotifCard = ({ notif, onDismiss, onAction }) => {
  const [visible, setVisible] = useState(true);

  const dismiss = () => {
    setVisible(false);
    setTimeout(() => onDismiss(notif.id), 300);
  };

  if (!visible) return null;

  const cfg = {
    friend_request: {
      icon: <UserPlus size={18} />,
      color: 'var(--farm-sky)',
      bg: 'rgba(90,143,180,0.12)',
      border: 'rgba(90,143,180,0.3)',
      title: '好友申请',
      body: `${notif.from_name} 想加你为好友`,
      actions: [
        { label: '接受', type: 'accept' },
        { label: '拒绝', type: 'reject' },
      ],
    },
    friend_accepted: {
      icon: <UserPlus size={18} />,
      color: 'var(--farm-leaf)',
      bg: 'rgba(74,124,63,0.1)',
      border: 'rgba(74,124,63,0.25)',
      title: '好友通知',
      body: `${notif.from_name} 接受了你的好友申请`,
      actions: [],
    },
    farm_invite: {
      icon: <Tractor size={18} />,
      color: 'var(--farm-harvest)',
      bg: 'rgba(200,146,42,0.1)',
      border: 'rgba(200,146,42,0.25)',
      title: '农场邀请',
      body: `${notif.from_name} 邀请你一起来农场种菜`,
      actions: [
        { label: '去农场', type: 'go_farm' },
      ],
    },
    chat_message: {
      icon: <MessageCircle size={18} />,
      color: 'var(--farm-sky)',
      bg: 'rgba(90,143,180,0.10)',
      border: 'rgba(90,143,180,0.2)',
      title: '新消息',
      body: `${notif.from_name}：${notif.payload?.content?.slice(0, 40) ?? ''}`,
      actions: [
        { label: '查看', type: 'open_chat' },
      ],
    },
  }[notif.type] || null;

  if (!cfg) return null;

  return (
    <div
      className='farm-notif-card'
      style={{
        background: cfg.bg,
        border: `1px solid ${cfg.border}`,
        borderLeft: `3px solid ${cfg.color}`,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
        <span style={{ color: cfg.color, flexShrink: 0, marginTop: 2 }}>{cfg.icon}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--farm-text-0)', marginBottom: 2 }}>
            {cfg.title}
          </div>
          <div style={{ fontSize: 12, color: 'var(--farm-text-1)', wordBreak: 'break-all' }}>
            {cfg.body}
          </div>
          {cfg.actions.length > 0 && (
            <div style={{ display: 'flex', gap: 6, marginTop: 8 }}>
              {cfg.actions.map((a) => (
                <Button
                  key={a.type}
                  size='small'
                  theme='solid'
                  style={{ fontSize: 11, height: 24, padding: '0 10px',
                    background: cfg.color, border: 'none', borderRadius: 5 }}
                  onClick={() => { onAction(notif, a.type); dismiss(); }}
                >
                  {a.label}
                </Button>
              ))}
            </div>
          )}
        </div>
        <button
          onClick={dismiss}
          style={{ background: 'none', border: 'none', cursor: 'pointer',
            color: 'var(--farm-text-3)', padding: 0, flexShrink: 0 }}
        >
          <X size={14} />
        </button>
      </div>
    </div>
  );
};

/* ─── 全局通知容器（挂在 FarmPage 或 PageLayout 中） ─── */
const FarmNotification = ({ userId, onChatOpen }) => {
  const [notifs, setNotifs] = useState([]);
  const nextId = useRef(0);
  const navigate = useNavigate();

  const addNotif = useCallback((ev) => {
    const id = nextId.current++;
    setNotifs((prev) => [...prev, { ...ev, id }]);
    // 8 秒自动消失
    setTimeout(() => {
      setNotifs((prev) => prev.filter((n) => n.id !== id));
    }, 8000);
  }, []);

  const dismiss = useCallback((id) => {
    setNotifs((prev) => prev.filter((n) => n.id !== id));
  }, []);

  const handleAction = useCallback((notif, actionType) => {
    if (actionType === 'accept') {
      API.post('/api/farm/friends/respond', {
        request_id: notif.payload?.request_id,
        action: 'accept',
      });
    } else if (actionType === 'reject') {
      API.post('/api/farm/friends/respond', {
        request_id: notif.payload?.request_id,
        action: 'reject',
      });
    } else if (actionType === 'go_farm') {
      navigate('/farm');
    } else if (actionType === 'open_chat') {
      onChatOpen && onChatOpen(notif.from_id, notif.from_name);
    }
  }, [navigate, onChatOpen]);

  // 轮询事件 — 每 3 秒
  useEffect(() => {
    if (!userId) return;
    let alive = true;
    const poll = async () => {
      try {
        const { data: res } = await API.get('/api/farm/events/poll', { disableDuplicate: true });
        if (!alive || !res.success) return;
        for (const ev of (res.data?.events ?? [])) {
          addNotif(ev);
        }
      } catch { /* ignore */ }
    };
    poll();
    const timer = setInterval(poll, 3000);
    return () => { alive = false; clearInterval(timer); };
  }, [userId, addNotif]);

  if (notifs.length === 0) return null;

  return (
    <div className='farm-notif-container'>
      {notifs.map((n) => (
        <NotifCard key={n.id} notif={n} onDismiss={dismiss} onAction={handleAction} />
      ))}
    </div>
  );
};

export default FarmNotification;
