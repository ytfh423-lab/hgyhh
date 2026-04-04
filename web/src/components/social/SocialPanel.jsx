/**
 * SocialPanel — 全站右侧社交浮窗
 * 挂在 PageLayout，所有页面均可用。
 * 通过 window CustomEvent 与其他页面组件通信：
 *   - 'social:open-chat'  { detail: { friendId, friendName } }
 */
import React, {
  useState, useEffect, useCallback, useRef,
  forwardRef, useImperativeHandle,
} from 'react';
import { Button, Input, Avatar, Badge, Tabs, TabPane } from '@douyinfe/semi-ui';
import {
  Users, X, MessageCircle, UserPlus, UserCheck, UserX,
  Trash2, Tractor, Search, Send, Minus, ChevronRight,
} from 'lucide-react';
import { API, showSuccess, showError } from '../../helpers';
import { useNavigate } from 'react-router-dom';
import './social.css';

/* ══════════════════════════════════════
   小工具
══════════════════════════════════════ */
const OnlineDot = ({ online, size = 9 }) => (
  <span style={{
    display: 'inline-block', width: size, height: size,
    borderRadius: '50%', flexShrink: 0,
    background: online ? '#4caf50' : 'rgba(180,180,180,0.5)',
    boxShadow: online ? '0 0 6px rgba(76,175,80,0.8)' : 'none',
    transition: 'background 0.3s',
  }} />
);

/* ══════════════════════════════════════
   通知弹窗（右下角）
══════════════════════════════════════ */
const NotifCard = ({ notif, onDismiss, onAction }) => {
  const [out, setOut] = useState(false);
  const dismiss = () => { setOut(true); setTimeout(() => onDismiss(notif.id), 280); };

  const cfg = {
    friend_request:  { icon: <UserPlus  size={22} />, color: '#5a8fb4', bg: 'rgba(90,143,180,0.13)',  border: 'rgba(90,143,180,0.35)',  title: '好友申请',  body: `${notif.from_name} 想加你为好友`, actions: [{ label: '接受', type: 'accept' }, { label: '拒绝', type: 'reject' }] },
    friend_accepted: { icon: <UserCheck size={22} />, color: '#4a7c3f', bg: 'rgba(74,124,63,0.12)',   border: 'rgba(74,124,63,0.3)',    title: '好友通知',  body: `${notif.from_name} 接受了你的好友申请`, actions: [] },
    farm_invite:     { icon: <Tractor   size={22} />, color: '#c8921a', bg: 'rgba(200,146,42,0.11)',  border: 'rgba(200,146,42,0.3)',   title: '农场邀请',  body: `${notif.from_name} 邀请你一起来农场种菜`, actions: [{ label: '去农场', type: 'go_farm' }] },
    chat_message:    { icon: <MessageCircle size={22} />, color: '#5a8fb4', bg: 'rgba(90,143,180,0.10)', border: 'rgba(90,143,180,0.25)', title: '新消息', body: `${notif.from_name}：${(notif.payload?.content ?? '').slice(0, 50)}`, actions: [{ label: '查看', type: 'open_chat' }] },
  }[notif.type] || null;
  if (!cfg) return null;

  return (
    <div className={`sp-notif-card${out ? ' sp-notif-out' : ''}`}
      style={{ background: cfg.bg, border: `1px solid ${cfg.border}`, borderLeft: `4px solid ${cfg.color}` }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
        <span style={{ color: cfg.color, flexShrink: 0, marginTop: 2 }}>{cfg.icon}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 14, fontWeight: 700, color: 'var(--sp-text-0)', marginBottom: 3 }}>{cfg.title}</div>
          <div style={{ fontSize: 13, color: 'var(--sp-text-1)', wordBreak: 'break-word' }}>{cfg.body}</div>
          {cfg.actions.length > 0 && (
            <div style={{ display: 'flex', gap: 8, marginTop: 10 }}>
              {cfg.actions.map(a => (
                <button key={a.type} className='sp-action-btn'
                  style={{ background: cfg.color }}
                  onClick={() => { onAction(notif, a.type); dismiss(); }}>
                  {a.label}
                </button>
              ))}
            </div>
          )}
        </div>
        <button className='sp-icon-btn' onClick={dismiss}><X size={16} /></button>
      </div>
    </div>
  );
};

/* ══════════════════════════════════════
   聊天窗口（带 typing + 已读）
══════════════════════════════════════ */
const Bubble = ({ msg, isMine }) => (
  <div style={{ display: 'flex', justifyContent: isMine ? 'flex-end' : 'flex-start', marginBottom: 8 }}>
    <div style={{
      maxWidth: '78%', padding: '9px 14px', fontSize: 14, lineHeight: 1.55,
      wordBreak: 'break-word', color: 'var(--sp-text-0)',
      borderRadius: isMine ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
      background: isMine ? 'rgba(74,124,63,0.28)' : 'rgba(255,255,255,0.09)',
      border: isMine ? '1px solid rgba(74,124,63,0.4)' : '1px solid rgba(255,255,255,0.12)',
    }}>
      {msg.content}
      <div style={{ fontSize: 11, color: 'var(--sp-text-3)', marginTop: 3,
        display: 'flex', justifyContent: isMine ? 'flex-end' : 'flex-start', gap: 6, alignItems: 'center' }}>
        <span>{new Date(msg.created_at * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
        {isMine && (
          <span style={{ color: msg.is_read ? '#4caf50' : 'var(--sp-text-3)', fontWeight: msg.is_read ? 600 : 400 }}>
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
          id: payload.msg_id ?? Date.now(),
          from_user_id: friendId, to_user_id: currentUserId,
          content: payload.content,
          created_at: payload.created_at ?? Math.floor(Date.now() / 1000),
          is_read: false,
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
      setMessages(prev => prev.map(m =>
        m.from_user_id === currentUserId ? { ...m, is_read: true } : m
      ));
    },
  }), [friendId, currentUserId]);

  useEffect(() => {
    if (!minimized)
      setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: 'smooth' }), 50);
  }, [messages, minimized, friendTyping]);

  const handleInputChange = (v) => {
    setInput(v);
    // 防抖发送 typing 事件
    clearTimeout(typingDebounce.current);
    if (v.trim()) {
      typingDebounce.current = setTimeout(() => {
        API.post('/api/social/chat/typing', { friend_id: friendId }).catch(() => {});
      }, 600);
    }
  };

  const send = async () => {
    const text = input.trim();
    if (!text || sending) return;
    setSending(true);
    setInput('');
    clearTimeout(typingDebounce.current);
    const optimistic = {
      id: Date.now(), from_user_id: currentUserId, to_user_id: friendId,
      content: text, created_at: Math.floor(Date.now() / 1000), is_read: false,
    };
    setMessages(prev => [...prev, optimistic]);
    try { await API.post(`/api/social/chat/${friendId}`, { content: text }); }
    catch { /* ignore */ }
    setSending(false);
  };

  return (
    <div className='sp-chat-widget'>
      <div className='sp-chat-header' onClick={() => setMinimized(v => !v)}>
        <span style={{ fontSize: 14, fontWeight: 700, flex: 1 }}>💬 {friendName}</span>
        <button className='sp-icon-btn' onClick={e => { e.stopPropagation(); setMinimized(v => !v); }}><Minus size={15} /></button>
        <button className='sp-icon-btn' onClick={e => { e.stopPropagation(); onClose(); }}><X size={15} /></button>
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
                <div className='sp-typing-indicator'>
                  <span /><span /><span />
                </div>
              </div>
            )}
            <div ref={bottomRef} />
          </div>
          <div className='sp-chat-input-row'>
            <Input value={input} onChange={handleInputChange} onEnterPress={send}
              placeholder='输入消息… (Enter 发送)' maxLength={300}
              style={{ flex: 1, fontSize: 13 }} />
            <Button icon={<Send size={15} />} loading={sending} onClick={send}
              theme='solid' style={{ background: '#4a7c3f', border: 'none', width: 38, height: 38, padding: 0, flexShrink: 0 }} />
          </div>
        </>
      )}
    </div>
  );
});
ChatWidget.displayName = 'ChatWidget';

/* ══════════════════════════════════════
   好友面板内容
══════════════════════════════════════ */
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

  useEffect(() => {
    load();
    const t = setInterval(load, 12000);
    return () => clearInterval(t);
  }, [load]);

  const doSearch = async () => {
    if (!searchQ.trim()) return;
    setSearchLoading(true);
    try {
      const { data: res } = await API.get(`/api/social/friends/search?q=${encodeURIComponent(searchQ)}`);
      if (res.success) setSearchResults(res.data || []);
      else showError(res.message);
    } finally { setSearchLoading(false); }
  };

  const respond = async (req, action) => {
    const { data: res } = await API.post('/api/social/friends/respond', { request_id: req.request_id, action });
    if (res.success) { showSuccess(res.message); load(); }
    else showError(res.message);
  };

  const removeFriend = async (friend) => {
    if (!window.confirm(`确定删除好友 ${friend.display_name || friend.username}？`)) return;
    const { data: res } = await API.delete(`/api/social/friends/${friend.user_id}`);
    if (res.success) { showSuccess(res.message); load(); }
    else showError(res.message);
  };

  const invite = async (friend) => {
    const { data: res } = await API.post('/api/social/invite', { friend_id: friend.user_id });
    if (res.success) showSuccess(res.message);
    else showError(res.message);
  };

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Tabs activeKey={activeTab} onChange={setActiveTab} size='small' style={{ flex: '0 0 auto' }}>
        <TabPane tab={`👫 好友 (${friends.length})`} itemKey='friends' />
        <TabPane tab={<span>🔔 申请{requests.length > 0 && <Badge count={requests.length} style={{ marginLeft: 5 }} />}</span>} itemKey='requests' />
        <TabPane tab='🔍 搜索' itemKey='search' />
      </Tabs>

      <div style={{ flex: 1, overflowY: 'auto', padding: '10px 0' }}>
        {/* 好友列表 */}
        {activeTab === 'friends' && (
          friends.length === 0
            ? <div className='sp-empty'>还没有好友，快去搜索添加吧 👀</div>
            : friends.map(f => (
              <div key={f.user_id} className='sp-friend-row'>
                <Avatar size='small' style={{ background: '#4a7c3f', flexShrink: 0 }}>
                  {(f.display_name || f.username || '?')[0].toUpperCase()}
                </Avatar>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--sp-text-0)' }}>
                      {f.display_name || f.username}
                    </span>
                    <OnlineDot online={f.online} />
                    {f.unread_count > 0 && (
                      <span className='sp-badge'>{f.unread_count}</span>
                    )}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--sp-text-3)' }}>
                    {f.online ? '在线' : '离线'} · @{f.username}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 5, flexShrink: 0 }}>
                  <button className='sp-icon-btn sp-icon-btn--sky'
                    title='聊天' onClick={() => onChatOpen(f.user_id, f.display_name || f.username)}>
                    <MessageCircle size={14} />
                  </button>
                  {f.online && (
                    <button className='sp-icon-btn sp-icon-btn--leaf'
                      title='邀请种菜' onClick={() => invite(f)}>
                      <Tractor size={14} />
                    </button>
                  )}
                  <button className='sp-icon-btn sp-icon-btn--danger'
                    title='删除好友' onClick={() => removeFriend(f)}>
                    <Trash2 size={13} />
                  </button>
                </div>
              </div>
            ))
        )}

        {/* 好友申请 */}
        {activeTab === 'requests' && (
          requests.length === 0
            ? <div className='sp-empty'>暂无好友申请</div>
            : requests.map(r => (
              <div key={r.request_id} className='sp-friend-row'
                style={{ background: 'rgba(90,143,180,0.07)', borderColor: 'rgba(90,143,180,0.2)' }}>
                <Avatar size='small' style={{ background: '#5a8fb4', flexShrink: 0 }}>
                  {(r.display_name || r.username || '?')[0].toUpperCase()}
                </Avatar>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 13, fontWeight: 700, color: 'var(--sp-text-0)' }}>
                    {r.display_name || r.username}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--sp-text-3)' }}>@{r.username}</div>
                </div>
                <div style={{ display: 'flex', gap: 5, flexShrink: 0 }}>
                  <button className='sp-icon-btn sp-icon-btn--leaf' title='接受'
                    onClick={() => respond(r, 'accept')}><UserCheck size={14} /></button>
                  <button className='sp-icon-btn sp-icon-btn--danger' title='拒绝'
                    onClick={() => respond(r, 'reject')}><UserX size={14} /></button>
                </div>
              </div>
            ))
        )}

        {/* 搜索用户 */}
        {activeTab === 'search' && (
          <div>
            <div style={{ display: 'flex', gap: 8, marginBottom: 10 }}>
              <Input prefix={<Search size={13} />} placeholder='搜索用户名或昵称'
                value={searchQ} onChange={setSearchQ} onEnterPress={doSearch}
                style={{ flex: 1, fontSize: 13 }} />
              <button className='sp-action-btn' style={{ background: '#4a7c3f', padding: '0 14px' }}
                onClick={doSearch}>{searchLoading ? '…' : '搜索'}</button>
            </div>
            {searchResults === null
              ? <div className='sp-empty'>输入关键词搜索用户</div>
              : searchResults.length === 0
                ? <div className='sp-empty'>未找到用户</div>
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
  const send = async () => {
    setLoading(true);
    try {
      const { data: res } = await API.post('/api/social/friends/request', { friend_id: user.user_id });
      if (res.success) { showSuccess(res.message); setStatus('pending'); }
      else showError(res.message);
    } finally { setLoading(false); }
  };
  const isFriend = status === 'accepted' || user.is_friend;
  const isPending = status === 'pending';
  return (
    <div className='sp-friend-row'>
      <Avatar size='small' style={{ background: '#888', flexShrink: 0 }}>
        {(user.display_name || user.username || '?')[0].toUpperCase()}
      </Avatar>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--sp-text-0)' }}>
            {user.display_name || user.username}
          </span>
          <OnlineDot online={user.online} />
        </div>
        <div style={{ fontSize: 11, color: 'var(--sp-text-3)' }}>@{user.username}</div>
      </div>
      <button className='sp-icon-btn'
        disabled={isFriend || isPending || loading}
        title={isFriend ? '已是好友' : isPending ? '已申请' : '加好友'}
        style={{ opacity: isFriend || isPending ? 0.5 : 1 }}
        onClick={send}>
        <UserPlus size={14} />
      </button>
    </div>
  );
};

/* ══════════════════════════════════════
   主组件 SocialPanel
══════════════════════════════════════ */
const SocialPanel = () => {
  const [currentUserId, setCurrentUserId] = useState(0);
  useEffect(() => {
    const read = () => {
      try { setCurrentUserId(JSON.parse(localStorage.getItem('user') || '{}').id || 0); }
      catch { /* ignore */ }
    };
    read();
    window.addEventListener('storage', read);
    return () => window.removeEventListener('storage', read);
  }, []);
  const [panelOpen, setPanelOpen] = useState(false);
  const [chat, setChat] = useState(null);           // {friendId, friendName}
  const [notifs, setNotifs] = useState([]);
  const [requestCount, setRequestCount] = useState(0);
  const chatRef = useRef(null);
  const nextNotifId = useRef(0);
  const navigate = useNavigate();

  /* ── 接收来自其他组件的打开聊天请求 ── */
  useEffect(() => {
    const handler = (e) => {
      const { friendId, friendName } = e.detail || {};
      if (friendId) setChat({ friendId, friendName });
    };
    window.addEventListener('social:open-chat', handler);
    return () => window.removeEventListener('social:open-chat', handler);
  }, []);

  /* ── 通知 ── */
  const addNotif = useCallback((ev) => {
    const id = nextNotifId.current++;
    setNotifs(prev => [...prev, { ...ev, id }]);
    setTimeout(() => setNotifs(prev => prev.filter(n => n.id !== id)), 10000);
  }, []);

  const dismissNotif = useCallback((id) => setNotifs(prev => prev.filter(n => n.id !== id)), []);

  const handleNotifAction = useCallback((notif, actionType) => {
    if (actionType === 'accept') {
      API.post('/api/social/friends/respond', { request_id: notif.payload?.request_id, action: 'accept' });
    } else if (actionType === 'reject') {
      API.post('/api/social/friends/respond', { request_id: notif.payload?.request_id, action: 'reject' });
    } else if (actionType === 'go_farm') {
      navigate('/farm');
    } else if (actionType === 'open_chat') {
      setChat({ friendId: notif.from_id, friendName: notif.from_name });
    }
  }, [navigate]);

  /* ── 事件轮询 ── */
  useEffect(() => {
    if (!currentUserId) return;
    let alive = true;
    const poll = async () => {
      try {
        const { data: res } = await API.get('/api/social/events/poll', { disableDuplicate: true });
        if (!alive || !res.success) return;
        for (const ev of (res.data?.events ?? [])) {
          if (ev.type === 'chat_message') {
            // 推入已打开的聊天窗
            if (chat?.friendId === ev.from_id) {
              chatRef.current?.pushMessage(ev.payload);
            } else {
              // 没打开 → 显示通知
              addNotif(ev);
              // 自动打开聊天
              setChat({ friendId: ev.from_id, friendName: ev.from_name });
              setTimeout(() => chatRef.current?.pushMessage(ev.payload), 120);
            }
          } else if (ev.type === 'typing') {
            if (chat?.friendId === ev.from_id) {
              chatRef.current?.showTyping();
            }
          } else if (ev.type === 'messages_read') {
            if (chat?.friendId === ev.from_id) {
              chatRef.current?.markRead();
            }
          } else {
            addNotif(ev);
          }
        }
      } catch { /* ignore */ }
    };
    poll();
    const timer = setInterval(poll, 3000);
    return () => { alive = false; clearInterval(timer); };
  }, [currentUserId, addNotif, chat]);

  /* ── 统计申请数（更新角标）── */
  useEffect(() => {
    if (!currentUserId) return;
    const poll = async () => {
      try {
        const { data: res } = await API.get('/api/social/friends/requests', { disableDuplicate: true });
        if (res.success) setRequestCount((res.data || []).length);
      } catch { /* ignore */ }
    };
    poll();
    const t = setInterval(poll, 15000);
    return () => clearInterval(t);
  }, [currentUserId]);

  if (!currentUserId) return null;

  return (
    <>
      {/* 右侧浮动按钮 */}
      <button
        className={`sp-fab${panelOpen ? ' sp-fab--open' : ''}`}
        onClick={() => setPanelOpen(v => !v)}
        title='社交'
      >
        <Users size={20} />
        {requestCount > 0 && !panelOpen && (
          <span className='sp-fab-badge'>{requestCount > 9 ? '9+' : requestCount}</span>
        )}
      </button>

      {/* 滑出面板 */}
      <div className={`sp-panel${panelOpen ? ' sp-panel--open' : ''}`}>
        <div className='sp-panel-header'>
          <span style={{ fontWeight: 700, fontSize: 15 }}>👥 社交</span>
          <button className='sp-icon-btn' onClick={() => setPanelOpen(false)}>
            <X size={16} />
          </button>
        </div>
        <div className='sp-panel-body'>
          <FriendPanel currentUserId={currentUserId} onChatOpen={(id, name) => {
            setChat({ friendId: id, friendName: name });
            setPanelOpen(false);
          }} />
        </div>
      </div>

      {/* 面板遮罩 */}
      {panelOpen && <div className='sp-overlay' onClick={() => setPanelOpen(false)} />}

      {/* 通知弹窗 */}
      {notifs.length > 0 && (
        <div className='sp-notif-container'>
          {notifs.map(n => (
            <NotifCard key={n.id} notif={n} onDismiss={dismissNotif} onAction={handleNotifAction} />
          ))}
        </div>
      )}

      {/* 聊天窗口 */}
      {chat && (
        <ChatWidget
          ref={chatRef}
          friendId={chat.friendId}
          friendName={chat.friendName}
          currentUserId={currentUserId}
          onClose={() => setChat(null)}
        />
      )}
    </>
  );
};

export default SocialPanel;
