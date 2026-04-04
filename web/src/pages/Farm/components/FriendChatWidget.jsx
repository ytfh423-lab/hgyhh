import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Input, Button } from '@douyinfe/semi-ui';
import { X, Send, Minus } from 'lucide-react';
import { API } from '../../../helpers';

/* ─── 单条消息气泡 ─── */
const Bubble = ({ msg, isMine }) => (
  <div style={{
    display: 'flex',
    justifyContent: isMine ? 'flex-end' : 'flex-start',
    marginBottom: 6,
  }}>
    <div style={{
      maxWidth: '75%',
      padding: '7px 11px',
      borderRadius: isMine ? '14px 14px 4px 14px' : '14px 14px 14px 4px',
      background: isMine ? 'rgba(74,124,63,0.25)' : 'rgba(255,255,255,0.08)',
      border: isMine
        ? '1px solid rgba(74,124,63,0.35)'
        : '1px solid rgba(255,255,255,0.1)',
      fontSize: 13,
      color: 'var(--farm-text-0)',
      wordBreak: 'break-word',
      lineHeight: 1.5,
    }}>
      {msg.content}
      <div style={{ fontSize: 10, color: 'var(--farm-text-3)', marginTop: 3, textAlign: 'right' }}>
        {new Date(msg.created_at * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
      </div>
    </div>
  </div>
);

/* ─── 聊天窗口主组件 ─── */
const FriendChatWidget = ({ friendId, friendName, currentUserId, onClose, newMessages }) => {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [minimized, setMinimized] = useState(false);
  const bottomRef = useRef(null);

  const loadHistory = useCallback(async () => {
    try {
      const { data: res } = await API.get(`/api/farm/chat/${friendId}`, { disableDuplicate: true });
      if (res.success) setMessages(res.data || []);
    } catch { /* ignore */ }
  }, [friendId]);

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  // 收到新消息时追加（来自 FarmNotification 的事件）
  useEffect(() => {
    if (!newMessages || newMessages.length === 0) return;
    setMessages((prev) => {
      const existingIds = new Set(prev.map((m) => m.id));
      const fresh = newMessages.filter((m) => !existingIds.has(m.msg_id));
      if (fresh.length === 0) return prev;
      return [
        ...prev,
        ...fresh.map((m) => ({
          id: m.msg_id,
          from_user_id: friendId,
          to_user_id: currentUserId,
          content: m.content,
          created_at: m.created_at,
        })),
      ];
    });
    setMinimized(false); // 收到消息自动展开
  }, [newMessages, friendId, currentUserId]);

  // 滚到底部
  useEffect(() => {
    if (!minimized) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, minimized]);

  const send = async () => {
    const text = input.trim();
    if (!text || sending) return;
    setSending(true);
    setInput('');
    // 乐观更新
    const optimistic = {
      id: Date.now(),
      from_user_id: currentUserId,
      to_user_id: friendId,
      content: text,
      created_at: Math.floor(Date.now() / 1000),
    };
    setMessages((prev) => [...prev, optimistic]);
    try {
      await API.post(`/api/farm/chat/${friendId}`, { content: text });
    } catch { /* ignore */ }
    setSending(false);
  };

  return (
    <div className='farm-chat-widget'>
      {/* 标题栏 */}
      <div className='farm-chat-header' onClick={() => setMinimized((v) => !v)}>
        <span style={{ fontSize: 13, fontWeight: 700, flex: 1 }}>
          💬 {friendName}
        </span>
        <button
          className='farm-chat-btn'
          onClick={(e) => { e.stopPropagation(); setMinimized((v) => !v); }}
        >
          <Minus size={14} />
        </button>
        <button
          className='farm-chat-btn'
          onClick={(e) => { e.stopPropagation(); onClose(); }}
        >
          <X size={14} />
        </button>
      </div>

      {!minimized && (
        <>
          {/* 消息列表 */}
          <div className='farm-chat-messages'>
            {messages.length === 0 && (
              <div style={{ textAlign: 'center', color: 'var(--farm-text-3)',
                fontSize: 12, padding: '20px 0' }}>
                开始聊天吧 👋
              </div>
            )}
            {messages.map((msg) => (
              <Bubble
                key={msg.id}
                msg={msg}
                isMine={msg.from_user_id === currentUserId}
              />
            ))}
            <div ref={bottomRef} />
          </div>

          {/* 输入框 */}
          <div className='farm-chat-input-row'>
            <Input
              value={input}
              onChange={setInput}
              onEnterPress={send}
              placeholder='输入消息…'
              maxLength={300}
              style={{ flex: 1, fontSize: 12 }}
            />
            <Button
              icon={<Send size={14} />}
              loading={sending}
              onClick={send}
              theme='solid'
              style={{ background: 'var(--farm-leaf)', border: 'none',
                width: 34, height: 34, padding: 0, flexShrink: 0 }}
            />
          </div>
        </>
      )}
    </div>
  );
};

export default FriendChatWidget;
