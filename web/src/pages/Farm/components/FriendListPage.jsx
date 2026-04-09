import React, { useState, useEffect, useCallback } from 'react';
import { Button, Input, Tabs, TabPane, Avatar, Badge } from '@douyinfe/semi-ui';
import { Search, UserPlus, UserCheck, UserX, Trash2, MessageCircle, Tractor } from 'lucide-react';
import { API, showSuccess, showError } from './utils';
import { farmConfirm } from './farmConfirm';

/* ─── 在线状态圆点 ─── */
const OnlineDot = ({ online }) => (
  <span style={{
    display: 'inline-block', width: 8, height: 8, borderRadius: '50%', flexShrink: 0,
    background: online ? '#4caf50' : '#888',
    boxShadow: online ? '0 0 5px rgba(76,175,80,0.7)' : 'none',
  }} />
);

const dispatchVisitFarm = (friendId, friendName) => {
  window.dispatchEvent(new CustomEvent('farm:visit-friend', { detail: { friendId, friendName } }));
};

/* ─── 好友卡片 ─── */
const FriendCard = ({ friend, onChat, onInvite, onRemove, t }) => (
  <div className='farm-row' style={{
    display: 'flex', alignItems: 'center', gap: 10,
    padding: '10px 12px', borderRadius: 8,
    background: 'rgba(255,255,255,0.03)',
    border: '1px solid rgba(255,255,255,0.07)',
    marginBottom: 6,
  }}>
    <Avatar size='small' style={{ background: 'var(--farm-leaf)', flexShrink: 0 }}>
      {(friend.display_name || friend.username || '?')[0].toUpperCase()}
    </Avatar>
    <div style={{ flex: 1, minWidth: 0 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--farm-text-0)' }}>
          {friend.display_name || friend.username}
        </span>
        <OnlineDot online={friend.online} />
        {friend.unread_count > 0 && (
          <span style={{
            fontSize: 10, background: 'var(--farm-danger)', color: '#fff',
            borderRadius: 8, padding: '1px 5px', fontWeight: 700,
          }}>{friend.unread_count}</span>
        )}
      </div>
      <div style={{ fontSize: 11, color: 'var(--farm-text-3)' }}>
        {friend.online ? '在线' : '离线'} · @{friend.username}
      </div>
    </div>
    <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
      <Button size='small' icon={<MessageCircle size={12} />}
        className='farm-btn'
        style={{ background: 'rgba(90,143,180,0.12)', border: '1px solid rgba(90,143,180,0.25)',
          color: 'var(--farm-sky)' }}
        onClick={() => onChat(friend)}>
        {t('聊天')}
      </Button>
      {friend.online && (
        <Button size='small' icon={<Tractor size={12} />}
          className='farm-btn'
          style={{ background: 'rgba(74,124,63,0.12)', border: '1px solid rgba(74,124,63,0.25)',
            color: 'var(--farm-leaf)' }}
          onClick={() => onInvite(friend)}>
          {t('邀请种菜')}
        </Button>
      )}
      <Button size='small' icon={<span style={{ fontSize: 12 }}>🌾</span>}
        className='farm-btn'
        style={{ background: 'rgba(200,146,42,0.1)', border: '1px solid rgba(200,146,42,0.28)',
          color: 'var(--farm-harvest)' }}
        onClick={() => dispatchVisitFarm(friend.user_id, friend.display_name || friend.username)}>
        {t('访问农场')}
      </Button>
      <Button size='small' icon={<Trash2 size={12} />}
        className='farm-btn'
        style={{ background: 'rgba(184,66,51,0.08)', border: '1px solid rgba(184,66,51,0.2)',
          color: 'var(--farm-danger)' }}
        onClick={() => onRemove(friend)}>
      </Button>
    </div>
  </div>
);

/* ─── 申请卡片 ─── */
const RequestCard = ({ req, onAccept, onReject, t }) => (
  <div className='farm-row' style={{
    display: 'flex', alignItems: 'center', gap: 10,
    padding: '10px 12px', borderRadius: 8,
    background: 'rgba(90,143,180,0.06)',
    border: '1px solid rgba(90,143,180,0.18)',
    marginBottom: 6,
  }}>
    <Avatar size='small' style={{ background: 'var(--farm-sky)', flexShrink: 0 }}>
      {(req.display_name || req.username || '?')[0].toUpperCase()}
    </Avatar>
    <div style={{ flex: 1, minWidth: 0 }}>
      <div style={{ fontSize: 13, fontWeight: 700, color: 'var(--farm-text-0)' }}>
        {req.display_name || req.username}
      </div>
      <div style={{ fontSize: 11, color: 'var(--farm-text-3)' }}>@{req.username}</div>
    </div>
    <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
      <Button size='small' icon={<UserCheck size={12} />}
        theme='solid'
        style={{ background: 'var(--farm-leaf)', border: 'none', fontSize: 12 }}
        onClick={() => onAccept(req)}>
        {t('接受')}
      </Button>
      <Button size='small' icon={<UserX size={12} />}
        className='farm-btn'
        style={{ background: 'rgba(184,66,51,0.08)', border: '1px solid rgba(184,66,51,0.2)',
          color: 'var(--farm-danger)', fontSize: 12 }}
        onClick={() => onReject(req)}>
        {t('拒绝')}
      </Button>
    </div>
  </div>
);

/* ─── 搜索结果卡片 ─── */
const SearchCard = ({ user, onRequest, t }) => {
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState(user.req_status || (user.is_friend ? 'friend' : ''));

  const send = async () => {
    setLoading(true);
    try {
      const { data: res } = await API.post('/api/social/friends/request', { friend_id: user.user_id });
      if (res.success) {
        showSuccess(res.message);
        setStatus('pending');
      } else {
        showError(res.message);
      }
    } finally {
      setLoading(false);
    }
  };

  const btnContent = () => {
    if (status === 'friend') return { label: t('已是好友'), disabled: true };
    if (status === 'accepted') return { label: t('已是好友'), disabled: true };
    if (status === 'pending') return { label: t('已申请'), disabled: true };
    return { label: t('加好友'), disabled: false };
  };
  const btn = btnContent();

  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 10,
      padding: '10px 12px', borderRadius: 8,
      background: 'rgba(255,255,255,0.03)',
      border: '1px solid rgba(255,255,255,0.07)',
      marginBottom: 6,
    }}>
      <Avatar size='small' style={{ background: '#888', flexShrink: 0 }}>
        {(user.display_name || user.username || '?')[0].toUpperCase()}
      </Avatar>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--farm-text-0)' }}>
            {user.display_name || user.username}
          </span>
          <OnlineDot online={user.online} />
        </div>
        <div style={{ fontSize: 11, color: 'var(--farm-text-3)' }}>@{user.username}</div>
      </div>
      <Button size='small' icon={<UserPlus size={12} />}
        disabled={btn.disabled}
        loading={loading}
        className='farm-btn'
        style={{ fontSize: 12, flexShrink: 0,
          background: btn.disabled ? 'rgba(128,128,128,0.1)' : 'rgba(90,143,180,0.12)',
          border: btn.disabled ? '1px solid rgba(128,128,128,0.2)' : '1px solid rgba(90,143,180,0.3)',
          color: btn.disabled ? 'var(--farm-text-3)' : 'var(--farm-sky)' }}
        onClick={send}>
        {btn.label}
      </Button>
    </div>
  );
};

/* ─── 主组件 ─── */
const FriendListPage = ({ onChatOpen, t }) => {
  const [friends, setFriends] = useState([]);
  const [requests, setRequests] = useState([]);
  const [searchQ, setSearchQ] = useState('');
  const [searchResults, setSearchResults] = useState(null);
  const [searchLoading, setSearchLoading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('friends');

  const loadFriends = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/social/friends');
      if (res.success) setFriends(res.data || []);
    } catch { /* ignore */ }
  }, []);

  const loadRequests = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/social/friends/requests');
      if (res.success) setRequests(res.data || []);
    } catch { /* ignore */ }
  }, []);

  useEffect(() => {
    loadFriends();
    loadRequests();
    const timer = setInterval(() => { loadFriends(); loadRequests(); }, 10000);
    return () => clearInterval(timer);
  }, [loadFriends, loadRequests]);

  const [onlineLoading, setOnlineLoading] = useState(false);
  const [searchLabel, setSearchLabel] = useState('');

  const doSearch = async () => {
    if (!searchQ.trim()) return;
    setSearchLoading(true);
    setSearchLabel('');
    try {
      const { data: res } = await API.get(`/api/social/friends/search?q=${encodeURIComponent(searchQ)}`);
      if (res.success) { setSearchResults(res.data || []); setSearchLabel(`搜索「${searchQ}」`); }
      else showError(res.message);
    } finally {
      setSearchLoading(false);
    }
  };

  const loadOnlineUsers = async () => {
    setOnlineLoading(true);
    setSearchQ('');
    try {
      const { data: res } = await API.get('/api/social/online-users');
      if (res.success) { setSearchResults(res.data || []); setSearchLabel('当前在线用户'); }
      else showError(res.message);
    } finally {
      setOnlineLoading(false);
    }
  };

  const handleAccept = async (req) => {
    setLoading(true);
    try {
      const { data: res } = await API.post('/api/social/friends/respond',
        { request_id: req.request_id, action: 'accept' });
      if (res.success) {
        showSuccess(res.message);
        setRequests((prev) => prev.filter((r) => r.request_id !== req.request_id));
        loadFriends();
      } else showError(res.message);
    } finally { setLoading(false); }
  };

  const handleReject = async (req) => {
    setLoading(true);
    try {
      const { data: res } = await API.post('/api/social/friends/respond',
        { request_id: req.request_id, action: 'reject' });
      if (res.success) {
        setRequests((prev) => prev.filter((r) => r.request_id !== req.request_id));
      } else showError(res.message);
    } finally { setLoading(false); }
  };

  const handleRemove = async (friend) => {
    if (!await farmConfirm('删除好友', `确定要删除好友「${friend.display_name || friend.username}」吗？删除后需重新申请。`, { icon: '👤', confirmType: 'danger', confirmText: '删除好友' })) return;
    const { data: res } = await API.delete(`/api/social/friends/${friend.user_id}`);
    if (res.success) {
      showSuccess(res.message);
      setFriends((prev) => prev.filter((f) => f.user_id !== friend.user_id));
    } else showError(res.message);
  };

  const handleInvite = async (friend) => {
    const { data: res } = await API.post('/api/social/invite', { friend_id: friend.user_id });
    if (res.success) showSuccess(res.message);
    else showError(res.message);
  };

  return (
    <div>
      <Tabs activeKey={activeTab} onChange={setActiveTab} size='small'>
        {/* ── 好友列表 ── */}
        <TabPane tab={`👫 ${t('好友')} (${friends.length})`} itemKey='friends'>
          <div style={{ marginTop: 10 }}>
            {friends.length === 0 ? (
              <div style={{ textAlign: 'center', color: 'var(--farm-text-3)', padding: '40px 0', fontSize: 13 }}>
                还没有好友，快去搜索添加吧 👀
              </div>
            ) : (
              friends.map((f) => (
                <FriendCard
                  key={f.user_id}
                  friend={f}
                  onChat={() => onChatOpen(f.user_id, f.display_name || f.username)}
                  onInvite={() => handleInvite(f)}
                  onRemove={() => handleRemove(f)}
                  t={t}
                />
              ))
            )}
          </div>
        </TabPane>

        {/* ── 好友申请 ── */}
        <TabPane
          tab={
            <span>
              🔔 {t('申请')}
              {requests.length > 0 && (
                <Badge count={requests.length} style={{ marginLeft: 6 }} />
              )}
            </span>
          }
          itemKey='requests'
        >
          <div style={{ marginTop: 10 }}>
            {requests.length === 0 ? (
              <div style={{ textAlign: 'center', color: 'var(--farm-text-3)', padding: '40px 0', fontSize: 13 }}>
                暂无好友申请
              </div>
            ) : (
              requests.map((r) => (
                <RequestCard
                  key={r.request_id}
                  req={r}
                  onAccept={handleAccept}
                  onReject={handleReject}
                  t={t}
                />
              ))
            )}
          </div>
        </TabPane>

        {/* ── 搜索用户 ── */}
        <TabPane tab={`🔍 ${t('搜索')}`} itemKey='search'>
          <div style={{ marginTop: 10 }}>
            <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
              <Input
                prefix={<Search size={14} />}
                placeholder={t('搜索用户名或昵称')}
                value={searchQ}
                onChange={setSearchQ}
                onEnterPress={doSearch}
                style={{ flex: 1 }}
              />
              <Button
                icon={<Search size={14} />}
                loading={searchLoading}
                onClick={doSearch}
                theme='solid'
                style={{ background: 'var(--farm-leaf)', border: 'none' }}>
                {t('搜索')}
              </Button>
            </div>
            {/* 查看在线用户 */}
            <button
              onClick={loadOnlineUsers}
              disabled={onlineLoading}
              style={{
                width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center',
                gap: 8, padding: '8px 14px', marginBottom: 10, borderRadius: 9,
                border: '1px dashed rgba(76,175,80,0.45)',
                background: 'rgba(74,124,63,0.07)',
                color: 'var(--farm-leaf)', fontSize: 13, fontWeight: 600,
                cursor: onlineLoading ? 'default' : 'pointer',
                opacity: onlineLoading ? 0.6 : 1,
                transition: 'background .15s',
              }}
            >
              <span style={{
                display: 'inline-block', width: 9, height: 9, borderRadius: '50%',
                background: '#4caf50', boxShadow: '0 0 0 2px rgba(76,175,80,0.3)',
                animation: 'farm-pulse 1.6s infinite', flexShrink: 0,
              }} />
              {onlineLoading ? '加载中…' : t('查看当前所有在线用户')}
            </button>
            {searchResults !== null && searchLabel && (
              <div style={{ fontSize: 11, color: 'var(--farm-text-3)', marginBottom: 8 }}>
                {searchLabel} · 共 {searchResults.length} 人
              </div>
            )}
            {searchResults === null ? (
              <div style={{ textAlign: 'center', color: 'var(--farm-text-3)', padding: '24px 0', fontSize: 13 }}>
                输入关键词搜索，或点击上方按钮查看在线用户
              </div>
            ) : searchResults.length === 0 ? (
              <div style={{ textAlign: 'center', color: 'var(--farm-text-3)', padding: '24px 0', fontSize: 13 }}>
                没有找到用户
              </div>
            ) : (
              searchResults.map((u) => (
                <SearchCard key={u.user_id} user={u} t={t} />
              ))
            )}
          </div>
        </TabPane>
      </Tabs>
    </div>
  );
};

export default FriendListPage;
