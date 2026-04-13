/**
 * SocialPanel — 可拖动浮动社交卡片
 * - 拖动标题栏移动位置，位置持久化到 localStorage
 * - 首次访问显示新手引导弹窗
 * - 全站挂载（PageLayout），所有页面可用
 */
import React, {
  useState, useEffect, useCallback, useRef,
  forwardRef, useImperativeHandle,
} from 'react';
import { Input, Avatar } from '@douyinfe/semi-ui';
import {
  Users, X, MessageCircle, UserPlus, UserCheck, UserX,
  Trash2, Tractor, Search, Send, Minus, Maximize2,
  GripHorizontal, ChevronRight, Bell,
} from 'lucide-react';
import { API, showSuccess, showError } from '../../helpers';
import { useNavigate } from 'react-router-dom';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { farmConfirm } from '../../pages/Farm/components/farmConfirm';
import './social.css';

/* ─────────────────────── helpers ─────────────────────── */
const OnlineDot = ({ online }) => (
  <span style={{
    display: 'inline-block', width: 9, height: 9, borderRadius: '50%',
    background: online ? '#4caf50' : 'rgba(160,160,160,0.4)',
    boxShadow: online ? '0 0 6px rgba(76,175,80,0.85)' : 'none',
    flexShrink: 0, transition: 'background .3s',
  }} />
);

/* ═══════════════════════════════════════════════════════
   新手引导弹窗
═══════════════════════════════════════════════════════ */
const ONBOARD_KEY = 'social_onboarding_v1';

const STEPS = [
  {
    emoji: '🎉',
    title: '好友功能上线啦！',
    desc: '我们为你带来全新的社交体验。添加好友、一起种菜、实时聊天，快来探索吧！',
  },
  {
    emoji: '🔍',
    title: '搜索并添加好友',
    desc: '前往农场里的「好友」页面，在「搜索用户」中输入对方用户名即可发送好友申请，对方接受后正式成为好友。',
    highlight: '搜索用户',
  },
  {
    emoji: '🔔',
    title: '接受好友申请',
    desc: '当有人申请加你为好友时，右下角会弹出通知。你也可以在农场「好友」页面的「好友申请」标签页随时查看并一键接受或拒绝。',
    highlight: '好友申请',
  },
  {
    emoji: '💬',
    title: '和好友实时聊天',
    desc: '在好友列表点击「发消息」即可打开聊天窗，支持实时消息推送、输入状态显示和已读回执。',
    highlight: '发消息',
  },
  {
    emoji: '🚜',
    title: '邀请好友一起种菜',
    desc: '当好友在线时，点击「邀请种菜」按钮，对方的屏幕上会立即弹出邀请通知，一键前往农场！',
    highlight: '邀请种菜',
  },
];

const OnboardingModal = ({ onClose }) => {
  const [step, setStep] = useState(0);
  const s = STEPS[step];
  const isLast = step === STEPS.length - 1;

  const finish = () => {
    localStorage.setItem(ONBOARD_KEY, '1');
    onClose();
  };

  return (
    <div className='sp-onboard-overlay'>
      <div className='sp-onboard-modal'>
        {/* 关闭 */}
        <button className='sp-onboard-close' onClick={finish}><X size={16} /></button>

        {/* 步骤指示 */}
        <div className='sp-onboard-steps'>
          {STEPS.map((_, i) => (
            <span key={i} className={`sp-onboard-dot${i === step ? ' active' : ''}`} />
          ))}
        </div>

        {/* 内容 */}
        <div className='sp-onboard-emoji'>{s.emoji}</div>
        <div className='sp-onboard-title'>{s.title}</div>
        <div className='sp-onboard-desc'>{s.desc}</div>
        {s.highlight && (
          <div className='sp-onboard-highlight'>
            <ChevronRight size={14} style={{ marginRight: 4 }} />
            在农场的「好友」页中找到「{s.highlight}」
          </div>
        )}

        {/* 导航按钮 */}
        <div className='sp-onboard-nav'>
          {step > 0 && (
            <button className='sp-onboard-btn sp-onboard-btn--ghost'
              onClick={() => setStep(s => s - 1)}>上一步</button>
          )}
          <div style={{ flex: 1 }} />
          {!isLast ? (
            <button className='sp-onboard-btn sp-onboard-btn--primary'
              onClick={() => setStep(s => s + 1)}>下一步</button>
          ) : (
            <button className='sp-onboard-btn sp-onboard-btn--primary' onClick={finish}>
              开始使用 🚀
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════
   通知弹窗
═══════════════════════════════════════════════════════ */
const NotifCard = ({ notif, onDismiss, onAction }) => {
  const [out, setOut] = useState(false);
  const dismissTimerRef = useRef(null);
  const removeTimerRef = useRef(null);
  const dismiss = useCallback(() => {
    if (dismissTimerRef.current) {
      clearTimeout(dismissTimerRef.current);
      dismissTimerRef.current = null;
    }
    if (removeTimerRef.current) {
      return;
    }
    setOut(true);
    removeTimerRef.current = window.setTimeout(() => {
      onDismiss(notif.id);
    }, 280);
  }, [notif.id, onDismiss]);

  useEffect(() => {
    const duration = typeof notif.duration === 'number' ? notif.duration : 3000;
    dismissTimerRef.current = window.setTimeout(() => {
      dismiss();
    }, duration);
    return () => {
      if (dismissTimerRef.current) {
        clearTimeout(dismissTimerRef.current);
      }
      if (removeTimerRef.current) {
        clearTimeout(removeTimerRef.current);
      }
    };
  }, [dismiss, notif.duration]);

  const cfg = {
    friend_request:  { icon: '👋', color: '#5a8fb4', bg: 'rgba(90,143,180,0.14)',  border: 'rgba(90,143,180,0.38)',  title: '好友申请',  body: `${notif.from_name} 想加你为好友`,           actions: [{ label: '✅ 接受', type: 'accept' }, { label: '❌ 拒绝', type: 'reject' }] },
    friend_accepted: { icon: '🤝', color: '#4a7c3f', bg: 'rgba(74,124,63,0.13)',   border: 'rgba(74,124,63,0.32)',   title: '好友通知',  body: `${notif.from_name} 接受了你的好友申请`,   actions: [] },
    farm_invite:     { icon: '🌾', color: '#c8921a', bg: 'rgba(200,146,42,0.12)',  border: 'rgba(200,146,42,0.32)',  title: '农场邀请',  body: `${notif.from_name} 邀请你一起来农场种菜`, actions: [{ label: '🚜 去农场', type: 'go_farm' }] },
    chat_message:    { icon: '💬', color: '#5a8fb4', bg: 'rgba(90,143,180,0.10)', border: 'rgba(90,143,180,0.26)',  title: '新消息',    body: `${notif.from_name}：${(notif.payload?.content ?? '').slice(0, 50)}`, actions: [{ label: '📖 查看', type: 'open_chat' }] },
    farm_success:    { icon: '✅', color: '#3e8f58', bg: 'linear-gradient(135deg, rgba(248,255,246,0.76), rgba(241,250,239,0.62))', border: 'rgba(92,167,116,0.36)', title: notif.title || '操作成功', body: `${notif.payload?.message ?? ''}`, actions: [], className: 'sp-notif-card--farm-success', titleColor: '#16351f', bodyColor: '#314238', closeColor: '#5f7467' },
  }[notif.type] || null;
  if (!cfg) return null;

  return (
    <div className={`sp-notif-card${cfg.className ? ` ${cfg.className}` : ''}${out ? ' sp-notif-out' : ''}`}
      style={{ background: cfg.bg, border: `1px solid ${cfg.border}`, borderLeft: `4px solid ${cfg.color}` }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
        <span style={{ fontSize: 22, flexShrink: 0, lineHeight: 1, marginTop: 1 }}>{cfg.icon}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 14, fontWeight: 700, color: cfg.titleColor || 'var(--sp-text-0)', marginBottom: 4 }}>{cfg.title}</div>
          <div style={{ fontSize: 13, color: cfg.bodyColor || 'var(--sp-text-1)', wordBreak: 'break-word', lineHeight: 1.5 }}>{cfg.body}</div>
          {cfg.actions.length > 0 && (
            <div style={{ display: 'flex', gap: 8, marginTop: 10, flexWrap: 'wrap' }}>
              {cfg.actions.map(a => (
                <button key={a.type} className='sp-notif-action-btn'
                  style={{ borderColor: cfg.color, color: cfg.color }}
                  onClick={() => { onAction(notif, a.type); dismiss(); }}>
                  {a.label}
                </button>
              ))}
            </div>
          )}
        </div>
        <button className='sp-close-btn' onClick={dismiss} style={{ color: cfg.closeColor || undefined }}><X size={15} /></button>
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════
   聊天窗口（带 typing + 已读）
═══════════════════════════════════════════════════════ */
const Bubble = ({ msg, isMine }) => (
  <div style={{ display: 'flex', justifyContent: isMine ? 'flex-end' : 'flex-start', marginBottom: 8 }}>
    <div style={{
      maxWidth: '78%', padding: '9px 14px', fontSize: 13.5, lineHeight: 1.55,
      wordBreak: 'break-word', color: 'var(--sp-text-0)',
      borderRadius: isMine ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
      background: isMine ? 'rgba(74,124,63,0.28)' : 'rgba(255,255,255,0.09)',
      border: isMine ? '1px solid rgba(74,124,63,0.4)' : '1px solid rgba(255,255,255,0.12)',
    }}>
      {msg.content}
      <div style={{
        fontSize: 11, color: 'var(--sp-text-3)', marginTop: 3,
        display: 'flex', justifyContent: isMine ? 'flex-end' : 'flex-start', gap: 6,
      }}>
        <span>{new Date(msg.created_at * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
        {isMine && (
          <span style={{ color: msg.is_read ? '#4caf50' : 'var(--sp-text-3)', fontWeight: 600 }}>
            {msg.is_read ? '已读' : '未读'}
          </span>
        )}
      </div>
    </div>
  </div>
);

const ChatWidget = forwardRef(({ friendId, friendName, currentUserId, onClose }, ref) => {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [minimized, setMinimized] = useState(false);
  const [friendTyping, setFriendTyping] = useState(false);
  const typingTimer = useRef(null);
  const typingDebounce = useRef(null);
  const bottomRef = useRef(null);

  const loadHistory = useCallback(async () => {
    try {
      const { data: res } = await API.get(`/api/social/chat/${friendId}`, { disableDuplicate: true });
      if (res.success) setMessages(res.data || []);
    } catch { /* ignore */ }
  }, [friendId]);

  useEffect(() => { loadHistory(); }, [loadHistory]);

  useImperativeHandle(ref, () => ({
    pushMessage(payload) {
      setMessages(prev => {
        if (prev.some(m => m.id === payload.msg_id)) return prev;
        return [...prev, {
          id: payload.msg_id ?? Date.now(), from_user_id: friendId, to_user_id: currentUserId,
          content: payload.content, created_at: payload.created_at ?? Math.floor(Date.now() / 1000), is_read: false,
        }];
      });
      setMinimized(false);
    },
    showTyping() {
      setFriendTyping(true);
      clearTimeout(typingTimer.current);
      typingTimer.current = setTimeout(() => setFriendTyping(false), 3500);
    },
    markRead() {
      setMessages(prev => prev.map(m => m.from_user_id === currentUserId ? { ...m, is_read: true } : m));
    },
  }), [friendId, currentUserId]);

  useEffect(() => {
    if (!minimized) setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: 'smooth' }), 50);
  }, [messages, minimized, friendTyping]);

  const handleInputChange = (v) => {
    setInput(v);
    clearTimeout(typingDebounce.current);
    if (v.trim()) typingDebounce.current = setTimeout(() => {
      API.post('/api/social/chat/typing', { friend_id: friendId }).catch(() => {});
    }, 600);
  };

  const send = async () => {
    const text = input.trim();
    if (!text || sending) return;
    setSending(true); setInput('');
    clearTimeout(typingDebounce.current);
    setMessages(prev => [...prev, {
      id: Date.now(), from_user_id: currentUserId, to_user_id: friendId,
      content: text, created_at: Math.floor(Date.now() / 1000), is_read: false,
    }]);
    try { await API.post(`/api/social/chat/${friendId}`, { content: text }); }
    catch { /* ignore */ }
    setSending(false);
  };

  return (
    <div className='sp-chat-widget'>
      <div className='sp-chat-header' onClick={() => setMinimized(v => !v)}>
        <span style={{ fontSize: 14, fontWeight: 700, flex: 1 }}>💬 与 {friendName} 聊天</span>
        <button className='sp-close-btn' onClick={e => { e.stopPropagation(); setMinimized(v => !v); }}><Minus size={14} /></button>
        <button className='sp-close-btn' onClick={e => { e.stopPropagation(); onClose(); }}><X size={14} /></button>
      </div>
      {!minimized && (
        <>
          <div className='sp-chat-messages'>
            {messages.length === 0 && (
              <div style={{ textAlign: 'center', color: 'var(--sp-text-3)', fontSize: 13, padding: '30px 0' }}>开始聊天吧 👋</div>
            )}
            {messages.map(m => <Bubble key={m.id} msg={m} isMine={m.from_user_id === currentUserId} />)}
            {friendTyping && (
              <div style={{ display: 'flex', justifyContent: 'flex-start', marginBottom: 8 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '8px 14px',
                  borderRadius: '16px 16px 16px 4px', background: 'rgba(255,255,255,0.08)',
                  border: '1px solid rgba(255,255,255,0.1)', fontSize: 12, color: 'var(--sp-text-2)' }}>
                  <div className='sp-typing-indicator'><span /><span /><span /></div>
                  <span>{friendName} 正在输入…</span>
                </div>
              </div>
            )}
            <div ref={bottomRef} />
          </div>
          <div className='sp-chat-input-row'>
            <Input value={input} onChange={handleInputChange} onEnterPress={send}
              placeholder='输入消息… (Enter 发送)' maxLength={300} style={{ flex: 1, fontSize: 13 }} />
            <button className='sp-send-btn' onClick={send} disabled={sending}>
              {sending ? '…' : <Send size={15} />}
            </button>
          </div>
        </>
      )}
    </div>
  );
});
ChatWidget.displayName = 'ChatWidget';

/* ═══════════════════════════════════════════════════════
   好友面板内容（卡片内部）
═══════════════════════════════════════════════════════ */
const FriendPanel = ({ currentUserId, onChatOpen }) => {
  const [friends, setFriends] = useState([]);
  const [requests, setRequests] = useState([]);
  const [searchQ, setSearchQ] = useState('');
  const [searchResults, setSearchResults] = useState(null);
  const [searchLoading, setSearchLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('friends');

  const load = useCallback(async () => {
    try {
      const [fr, rr] = await Promise.all([
        API.get('/api/social/friends'),
        API.get('/api/social/friends/requests'),
      ]);
      if (fr.data.success) setFriends(fr.data.data || []);
      if (rr.data.success) setRequests(rr.data.data || []);
    } catch { /* ignore */ }
  }, []);

  useEffect(() => { load(); const t = setInterval(load, 12000); return () => clearInterval(t); }, [load]);

  const [onlineLoading, setOnlineLoading] = useState(false);
  const [searchLabel, setSearchLabel] = useState('');   // 当前结果来源描述

  const doSearch = async () => {
    if (!searchQ.trim()) return;
    setSearchLoading(true);
    setSearchLabel('');
    try {
      const { data: res } = await API.get(`/api/social/friends/search?q=${encodeURIComponent(searchQ)}`);
      if (res.success) { setSearchResults(res.data || []); setSearchLabel(`搜索「${searchQ}」的结果`); }
      else showError(res.message);
    } finally { setSearchLoading(false); }
  };

  const loadOnlineUsers = async () => {
    setOnlineLoading(true);
    setSearchQ('');
    try {
      const { data: res } = await API.get('/api/social/online-users');
      if (res.success) { setSearchResults(res.data || []); setSearchLabel('当前在线用户'); }
      else showError(res.message);
    } finally { setOnlineLoading(false); }
  };

  const respond = async (req, action) => {
    const { data: res } = await API.post('/api/social/friends/respond', { request_id: req.request_id, action });
    if (res.success) { showSuccess(res.message); load(); } else showError(res.message);
  };

  const removeFriend = async (f) => {
    if (!await farmConfirm('删除好友', `确定要删除好友「${f.display_name || f.username}」吗？删除后需重新申请。`, { icon: '👤', confirmType: 'danger', confirmText: '删除好友' })) return;
    const { data: res } = await API.delete(`/api/social/friends/${f.user_id}`);
    if (res.success) { showSuccess(res.message); load(); } else showError(res.message);
  };

  const invite = async (f) => {
    const { data: res } = await API.post('/api/social/invite', { friend_id: f.user_id });
    if (res.success) showSuccess(res.message); else showError(res.message);
  };

  const TABS = [
    { key: 'friends',  label: `👫 好友列表`,   badge: 0 },
    { key: 'requests', label: `🔔 好友申请`,  badge: requests.length },
    { key: 'search',   label: `🔍 搜索用户`,   badge: 0 },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* 标签栏 */}
      <div className='sp-tab-bar'>
        {TABS.map(tab => (
          <button key={tab.key}
            className={`sp-tab-btn${activeTab === tab.key ? ' active' : ''}`}
            onClick={() => setActiveTab(tab.key)}>
            {tab.label}
            {tab.badge > 0 && <span className='sp-tab-badge'>{tab.badge}</span>}
          </button>
        ))}
      </div>

      {/* 内容区 */}
      <div className='sp-panel-scroll'>
        {/* 好友列表 */}
        {activeTab === 'friends' && (
          friends.length === 0
            ? <div className='sp-empty'>还没有好友，去「搜索用户」添加吧 👀</div>
            : friends.map(f => (
              <div key={f.user_id} className='sp-friend-card'>
                <div className='sp-friend-card__top'>
                  <Avatar size='small' style={{ background: '#4a7c3f', flexShrink: 0 }}>
                    {(f.display_name || f.username || '?')[0].toUpperCase()}
                  </Avatar>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
                      <span style={{ fontWeight: 700, fontSize: 13, color: 'var(--sp-text-0)' }}>
                        {f.display_name || f.username}
                      </span>
                      <OnlineDot online={f.online} />
                      {f.unread_count > 0 && <span className='sp-badge'>{f.unread_count}</span>}
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--sp-text-3)', marginTop: 1 }}>
                      {f.online ? '🟢 在线' : '⚫ 离线'} · @{f.username}
                    </div>
                  </div>
                </div>
                <div className='sp-friend-card__actions'>
                  <button className='sp-txt-btn sp-txt-btn--sky'
                    onClick={() => onChatOpen(f.user_id, f.display_name || f.username)}>
                    <MessageCircle size={13} /> 发消息
                  </button>
                  <button className='sp-txt-btn sp-txt-btn--harvest'
                    onClick={() => {
                      window.dispatchEvent(new CustomEvent('farm:visit-friend',
                        { detail: { friendId: f.user_id, friendName: f.display_name || f.username } }));
                    }}>
                    🌾 访问农场
                  </button>
                  {f.online && (
                    <button className='sp-txt-btn sp-txt-btn--leaf' onClick={() => invite(f)}>
                      <Tractor size={13} /> 邀请种菜
                    </button>
                  )}
                  <button className='sp-txt-btn sp-txt-btn--danger' onClick={() => removeFriend(f)}>
                    <Trash2 size={12} /> 删除好友
                  </button>
                </div>
              </div>
            ))
        )}

        {/* 好友申请 */}
        {activeTab === 'requests' && (
          requests.length === 0
            ? <div className='sp-empty'>暂无待处理的好友申请 ✅</div>
            : requests.map(r => (
              <div key={r.request_id} className='sp-friend-card' style={{ borderColor: 'rgba(90,143,180,0.25)' }}>
                <div className='sp-friend-card__top'>
                  <Avatar size='small' style={{ background: '#5a8fb4', flexShrink: 0 }}>
                    {(r.display_name || r.username || '?')[0].toUpperCase()}
                  </Avatar>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontWeight: 700, fontSize: 13, color: 'var(--sp-text-0)' }}>{r.display_name || r.username}</div>
                    <div style={{ fontSize: 11, color: 'var(--sp-text-3)', marginTop: 1 }}>想加你为好友 · @{r.username}</div>
                  </div>
                </div>
                <div className='sp-friend-card__actions'>
                  <button className='sp-txt-btn sp-txt-btn--leaf' onClick={() => respond(r, 'accept')}>
                    <UserCheck size={13} /> 接受申请
                  </button>
                  <button className='sp-txt-btn sp-txt-btn--danger' onClick={() => respond(r, 'reject')}>
                    <UserX size={13} /> 拒绝
                  </button>
                </div>
              </div>
            ))
        )}

        {/* 搜索 */}
        {activeTab === 'search' && (
          <div>
            {/* 搜索栏 */}
            <div style={{ display: 'flex', gap: 8, marginBottom: 8, marginTop: 4 }}>
              <Input prefix={<Search size={13} />} placeholder='输入用户名或昵称搜索'
                value={searchQ} onChange={setSearchQ} onEnterPress={doSearch} style={{ flex: 1, fontSize: 13 }} />
              <button className='sp-txt-btn sp-txt-btn--leaf' style={{ flexShrink: 0 }} onClick={doSearch}>
                {searchLoading ? '搜索中…' : <><Search size={13} /> 搜索</>}
              </button>
            </div>
            {/* 查看在线用户按钮 */}
            <button className='sp-online-users-btn' onClick={loadOnlineUsers} disabled={onlineLoading}>
              <span className='sp-online-dot-anim' />
              {onlineLoading ? '加载中…' : '查看当前所有在线用户'}
            </button>
            {/* 结果标题 */}
            {searchResults !== null && searchLabel && (
              <div style={{ fontSize: 11, color: 'var(--sp-text-3)', marginBottom: 8, marginTop: 4 }}>
                {searchLabel} · 共 {searchResults.length} 人
              </div>
            )}
            {/* 结果列表 */}
            {searchResults === null
              ? <div className='sp-empty'>输入关键词搜索，或点击上方按钮查看在线用户</div>
              : searchResults.length === 0
                ? <div className='sp-empty'>没有找到用户</div>
                : searchResults.map(u => <SearchRow key={u.user_id} user={u} />)
            }
          </div>
        )}
      </div>
    </div>
  );
};

const SearchRow = ({ user }) => {
  const [status, setStatus] = useState(user.req_status || (user.is_friend ? 'accepted' : ''));
  const [loading, setLoading] = useState(false);
  const isFriend = status === 'accepted' || user.is_friend;
  const isPending = status === 'pending';
  const send = async () => {
    setLoading(true);
    try {
      const { data: res } = await API.post('/api/social/friends/request', { friend_id: user.user_id });
      if (res.success) { showSuccess(res.message); setStatus('pending'); } else showError(res.message);
    } finally { setLoading(false); }
  };
  return (
    <div className='sp-friend-card'>
      <div className='sp-friend-card__top'>
        <Avatar size='small' style={{ background: '#777', flexShrink: 0 }}>
          {(user.display_name || user.username || '?')[0].toUpperCase()}
        </Avatar>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
            <span style={{ fontWeight: 700, fontSize: 13, color: 'var(--sp-text-0)' }}>{user.display_name || user.username}</span>
            <OnlineDot online={user.online} />
          </div>
          <div style={{ fontSize: 11, color: 'var(--sp-text-3)', marginTop: 1 }}>
            {user.online ? '🟢 在线' : '⚫ 离线'} · @{user.username}
          </div>
        </div>
      </div>
      <div className='sp-friend-card__actions'>
        {isFriend
          ? <span style={{ fontSize: 12, color: 'var(--sp-text-3)' }}>✅ 已是好友</span>
          : isPending
            ? <span style={{ fontSize: 12, color: 'var(--sp-text-3)' }}>⏳ 申请已发送</span>
            : (
              <button className='sp-txt-btn sp-txt-btn--sky' onClick={send} disabled={loading}>
                <UserPlus size={13} /> {loading ? '发送中…' : '发送好友申请'}
              </button>
            )
        }
      </div>
    </div>
  );
};

/* ═══════════════════════════════════════════════════════
   可拖动浮动卡片
═══════════════════════════════════════════════════════ */
const CARD_KEY = 'social_card_pos';
const CARD_W_COLLAPSED = 340;
const CARD_W_EXPANDED  = 380;
const HEADER_H = 46;
const QUICK_H  = 60;
const GAP      = 12; // 距屏幕边缘最小留白

/** 将 (x, y) 夹紧到视口内，保证标题栏始终可见 */
function clampToViewport(x, y, cardW) {
  return {
    x: Math.max(GAP, Math.min(window.innerWidth  - cardW - GAP, x)),
    y: Math.max(GAP, Math.min(window.innerHeight - HEADER_H - GAP, y)),
  };
}

/** 从 localStorage 读取并立刻校验是否在当前视口内 */
function loadPos() {
  try {
    const s = JSON.parse(localStorage.getItem(CARD_KEY) || 'null');
    if (s && typeof s.x === 'number' && typeof s.y === 'number') {
      return clampToViewport(s.x, s.y, CARD_W_COLLAPSED);
    }
  } catch { /* ignore */ }
  // 默认：右下角，距边缘 GAP
  return {
    x: window.innerWidth  - CARD_W_COLLAPSED - GAP,
    y: window.innerHeight - HEADER_H - QUICK_H - GAP * 2,
  };
}

const DraggableCard = ({ children, requestCount }) => {
  const [pos, setPos]         = useState(loadPos);
  const [expanded, setExpanded] = useState(false);
  const dragging = useRef(false);
  const offset   = useRef({ x: 0, y: 0 });
  const cardRef  = useRef(null);

  // 挂载后再次校验（应对 SSR 或视口尺寸变化）
  useEffect(() => {
    setPos(prev => clampToViewport(prev.x, prev.y, CARD_W_COLLAPSED));
  }, []);

  // 视口 resize 时重新夹紧
  useEffect(() => {
    const onResize = () => {
      setPos(prev => {
        const w = expanded ? CARD_W_EXPANDED : CARD_W_COLLAPSED;
        return clampToViewport(prev.x, prev.y, w);
      });
    };
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, [expanded]);

  // 展开时确保卡片不超出屏幕右边
  useEffect(() => {
    if (expanded) {
      setPos(prev => clampToViewport(prev.x, prev.y, CARD_W_EXPANDED));
    }
  }, [expanded]);

  const savePos = (p) => { localStorage.setItem(CARD_KEY, JSON.stringify(p)); return p; };

  const onMouseDown = (e) => {
    if (e.button !== 0) return;
    dragging.current = true;
    offset.current = { x: e.clientX - pos.x, y: e.clientY - pos.y };
    document.body.style.userSelect = 'none';
  };
  const onMouseMove = useCallback((e) => {
    if (!dragging.current) return;
    const w = cardRef.current?.offsetWidth || CARD_W_COLLAPSED;
    setPos(clampToViewport(e.clientX - offset.current.x, e.clientY - offset.current.y, w));
  }, []);
  const onMouseUp = useCallback(() => {
    if (!dragging.current) return;
    dragging.current = false;
    document.body.style.userSelect = '';
    setPos(prev => savePos(prev));
  }, []);

  const onTouchStart = (e) => {
    const t = e.touches[0];
    dragging.current = true;
    offset.current = { x: t.clientX - pos.x, y: t.clientY - pos.y };
  };
  const onTouchMove = useCallback((e) => {
    if (!dragging.current) return;
    const t = e.touches[0];
    const w = cardRef.current?.offsetWidth || CARD_W_COLLAPSED;
    setPos(clampToViewport(t.clientX - offset.current.x, t.clientY - offset.current.y, w));
  }, []);
  const onTouchEnd = useCallback(() => {
    dragging.current = false;
    setPos(prev => savePos(prev));
  }, []);

  useEffect(() => {
    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup',   onMouseUp);
    window.addEventListener('touchmove', onTouchMove, { passive: false });
    window.addEventListener('touchend',  onTouchEnd);
    return () => {
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup',   onMouseUp);
      window.removeEventListener('touchmove', onTouchMove);
      window.removeEventListener('touchend',  onTouchEnd);
    };
  }, [onMouseMove, onMouseUp, onTouchMove, onTouchEnd]);

  // 展开时动态计算可用高度，避免溢出底部
  const bodyMaxH = expanded
    ? Math.max(180, window.innerHeight - pos.y - HEADER_H - GAP * 2)
    : 0;

  return (
    <div ref={cardRef}
      className={`sp-card${expanded ? ' sp-card--expanded' : ''}`}
      style={{ left: pos.x, top: pos.y }}>

      {/* ── 拖动标题栏 ── */}
      <div className='sp-card-header' onMouseDown={onMouseDown} onTouchStart={onTouchStart}>
        <GripHorizontal size={14} style={{ color: 'var(--sp-text-3)', flexShrink: 0 }} />
        <span style={{ fontWeight: 700, fontSize: 13, flex: 1 }}>👥 好友社交</span>
        {requestCount > 0 && !expanded && (
          <span className='sp-card-badge'>{requestCount > 9 ? '9+' : requestCount}</span>
        )}
        <button className='sp-close-btn'
          onClick={e => { e.stopPropagation(); setExpanded(v => !v); }}
          title={expanded ? '收起' : '展开'}>
          {expanded ? <Minus size={14} /> : <Maximize2 size={13} />}
        </button>
      </div>

      {/* ── 折叠态：快捷按钮 ── */}
      {!expanded && (
        <div className='sp-card-quick'>
          <button className='sp-quick-btn' onClick={() => setExpanded(true)}>
            <Users size={13} /> 好友列表
          </button>
          <button className='sp-quick-btn sp-quick-btn--accent' style={{ position: 'relative' }}
            onClick={() => setExpanded(true)}>
            <Bell size={13} /> 好友申请
            {requestCount > 0 && <span className='sp-mini-badge'>{requestCount}</span>}
          </button>
          <button className='sp-quick-btn' onClick={() => setExpanded(true)}>
            <Search size={13} /> 搜索用户
          </button>
        </div>
      )}

      {/* ── 展开态：面板内容 ── */}
      {expanded && (
        <div className='sp-card-body' style={{ maxHeight: bodyMaxH }}>
          {children}
        </div>
      )}
    </div>
  );
};

/* ═══════════════════════════════════════════════════════
   主组件
═══════════════════════════════════════════════════════ */
const SocialPanel = () => {
  const isMobile = useIsMobile();
  const [currentUserId, setCurrentUserId] = useState(0);
  const [chat, setChat] = useState(null);
  const [notifs, setNotifs] = useState([]);
  const [showOnboarding, setShowOnboarding] = useState(false);
  const chatRef = useRef(null);
  const nextId = useRef(0);
  const navigate = useNavigate();

  // 读取用户 ID
  useEffect(() => {
    const read = () => {
      try { setCurrentUserId(JSON.parse(localStorage.getItem('user') || '{}').id || 0); }
      catch { /* ignore */ }
    };
    read();
    window.addEventListener('storage', read);
    return () => window.removeEventListener('storage', read);
  }, []);

  // 首次使用引导
  useEffect(() => {
    if (!currentUserId) return;
    if (!localStorage.getItem(ONBOARD_KEY)) {
      const t = setTimeout(() => setShowOnboarding(true), 1200);
      return () => clearTimeout(t);
    }
  }, [currentUserId]);

  // 监听 open-chat 事件
  useEffect(() => {
    const h = (e) => { const { friendId, friendName } = e.detail || {}; if (friendId) setChat({ friendId, friendName }); };
    window.addEventListener('social:open-chat', h);
    return () => window.removeEventListener('social:open-chat', h);
  }, []);

  // 通知
  const addNotif = useCallback((ev) => {
    const id = nextId.current++;
    const maxVisible = window.innerWidth <= 768 ? 2 : 3;
    setNotifs(prev => {
      const next = [...prev, { ...ev, id }];
      return next.length > maxVisible ? next.slice(-maxVisible) : next;
    });
  }, []);

  useEffect(() => {
    const handler = (event) => {
      const detail = event.detail || {};
      const message = typeof detail.message === 'string'
        ? detail.message.trim()
        : String(detail.message || '').trim();
      if (!message) return;
      addNotif({
        type: 'farm_success',
        title: typeof detail.title === 'string' && detail.title.trim() ? detail.title.trim() : '操作成功',
        payload: { message },
        duration: typeof detail.duration === 'number' ? detail.duration : 2000,
      });
    };
    window.addEventListener('farm:success-notify', handler);
    return () => window.removeEventListener('farm:success-notify', handler);
  }, [addNotif]);

  const handleNotifAction = useCallback((notif, type) => {
    if (type === 'accept') API.post('/api/social/friends/respond', { request_id: notif.payload?.request_id, action: 'accept' });
    else if (type === 'reject') API.post('/api/social/friends/respond', { request_id: notif.payload?.request_id, action: 'reject' });
    else if (type === 'go_farm') navigate('/farm');
    else if (type === 'open_chat') setChat({ friendId: notif.from_id, friendName: notif.from_name });
  }, [navigate]);

  // 事件轮询
  useEffect(() => {
    if (!currentUserId) return;
    let alive = true;
    let timer = null;
    const schedule = (delay) => {
      if (!alive) return;
      timer = window.setTimeout(run, delay);
    };
    const run = async () => {
      if (!alive) return;
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') {
        schedule(12000);
        return;
      }
      try {
        const { data: res } = await API.get('/api/social/events/poll', { disableDuplicate: true });
        if (!alive || !res.success) return;
        for (const ev of (res.data?.events ?? [])) {
          if (ev.type === 'chat_message') {
            if (chat?.friendId === ev.from_id) chatRef.current?.pushMessage(ev.payload);
            else { addNotif(ev); setChat({ friendId: ev.from_id, friendName: ev.from_name }); setTimeout(() => chatRef.current?.pushMessage(ev.payload), 120); }
          } else if (ev.type === 'typing') {
            if (chat?.friendId === ev.from_id) chatRef.current?.showTyping();
          } else if (ev.type === 'messages_read') {
            if (chat?.friendId === ev.from_id) chatRef.current?.markRead();
          } else { addNotif(ev); }
        }
      } catch { /* ignore */ }
      finally {
        schedule(3500);
      }
    };
    run();
    return () => {
      alive = false;
      if (timer) {
        clearTimeout(timer);
      }
    };
  }, [currentUserId, addNotif, chat]);

  if (!currentUserId) return null;

  return (
    <>
      {/* 新手引导 */}
      {showOnboarding && !isMobile && <OnboardingModal onClose={() => setShowOnboarding(false)} />}

      {/* 通知弹窗 */}
      {notifs.length > 0 && (
        <div className='sp-notif-container'>
          {notifs.map(n => (
            <NotifCard key={n.id} notif={n} onDismiss={(id) => setNotifs(p => p.filter(x => x.id !== id))} onAction={handleNotifAction} />
          ))}
        </div>
      )}

      {/* 聊天窗口 */}
      {chat && (
        <ChatWidget ref={chatRef} friendId={chat.friendId} friendName={chat.friendName}
          currentUserId={currentUserId} onClose={() => setChat(null)} />
      )}
    </>
  );
};

export default SocialPanel;
