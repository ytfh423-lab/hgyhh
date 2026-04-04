import React, { useState, useEffect, useRef, useCallback, useImperativeHandle, forwardRef } from 'react';
import { Input, Button } from '@douyinfe/semi-ui';
import { X, Send, Minus } from 'lucide-react';
import { API } from '../../../helpers';

/* ─── 单条消息气泡 ─── */
const Bubble = ({ msg, isMine }) => (
  <div style={{
    display: 'flex',
    justifyContent: isMine ? 'flex-end' : 'flex-start',
    marginBottom: 8,
  }}>
    <div style={{
      maxWidth: '78%',
      padding: '9px 14px',
      borderRadius: isMine ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
      background: isMine ? 'rgba(74,124,63,0.28)' : 'rgba(255,255,255,0.09)',
      border: isMine
        ? '1px solid rgba(74,124,63,0.4)'
        : '1px solid rgba(255,255,255,0.12)',
      fontSize: 14,
      color: 'var(--farm-text-0)',
      wordBreak: 'break-word',
      lineHeight: 1.55,
    }}>
      {msg.content}
      <div style={{ fontSize: 11, color: 'var(--farm-text-3)', marginTop: 4, textAlign: 'right' }}>
        {new Date(msg.created_at * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
      </div>
    </div>
  </div>
);

/* ─── 聊天窗口主组件 ─── */
// ref 暴露 pushMessage(payload) 让父组件直接推入新消息
const FriendChatWidget = forwardRef(({ friendId, friendName, currentUserId, onClose }, ref) => {
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

  // 暴露给父组件：推入一条对方发来的新消息
  useImperativeHandle(ref, () => ({
    pushMessage(payload) {
      setMessages((prev) => {
        if (prev.some((m) => m.id === payload.msg_id)) return prev;
        return [
          ...prev,
          {
            id: payload.msg_id ?? Date.now(),
            from_user_id: friendId,
            to_user_id: currentUserId,
            content: payload.content,
            created_at: payload.created_at ?? Math.floor(Date.now() / 1000),
          },
        ];
      });
      setMinimized(false);
    },
  }), [friendId, currentUserId]);

  // 滚到底部
  useEffect(() => {
    if (!minimized) {
      setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: 'smooth' }), 50);
    }
  }, [messages, minimized]);

  const send = async () => {
    const text = input.trim();
    if (!text || sending) return;
    setSending(true);
    setInput('');
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
        <span style={{ fontSize: 14, fontWeight: 700, flex: 1 }}>
          💬 {friendName}
        </span>
        <button className='farm-chat-btn'
          onClick={(e) => { e.stopPropagation(); setMinimized((v) => !v); }}>
          <Minus size={15} />
        </button>
        <button className='farm-chat-btn'
          onClick={(e) => { e.stopPropagation(); onClose(); }}>
          <X size={15} />
        </button>
      </div>

      {!minimized && (
        <>
          <div className='farm-chat-messages'>
            {messages.length === 0 && (
              <div style={{ textAlign: 'center', color: 'var(--farm-text-3)',
                fontSize: 13, padding: '30px 0' }}>
                开始聊天吧 👋
              </div>
            )}
            {messages.map((msg) => (
              <Bubble key={msg.id} msg={msg} isMine={msg.from_user_id === currentUserId} />
            ))}
            <div ref={bottomRef} />
          </div>

          <div className='farm-chat-input-row'>
            <Input
              value={input}
              onChange={setInput}
              onEnterPress={send}
              placeholder='输入消息… (Enter 发送)'
              maxLength={300}
              style={{ flex: 1, fontSize: 13 }}
            />
            <Button
              icon={<Send size={15} />}
              loading={sending}
              onClick={send}
              theme='solid'
              style={{ background: 'var(--farm-leaf)', border: 'none',
                width: 38, height: 38, padding: 0, flexShrink: 0 }}
            />
          </div>
        </>
      )}
    </div>
  );
});

FriendChatWidget.displayName = 'FriendChatWidget';
export default FriendChatWidget;
