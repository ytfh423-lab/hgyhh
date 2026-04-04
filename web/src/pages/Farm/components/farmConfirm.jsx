/**
 * farmConfirm — 全局自定义确认弹窗（农场风格）
 *
 * 使用方式：
 *   import { farmConfirm } from './farmConfirm';
 *   const ok = await farmConfirm('标题', '内容');
 *   const ok = await farmConfirm('标题', '内容', { icon: '🗑', confirmType: 'danger', confirmText: '删除' });
 *
 * 需要在 React 树的顶层挂载 <FarmConfirmProvider />（PageLayout 或 Farm/index 均可）。
 */
import React, { useState, useEffect } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { AlertTriangle } from 'lucide-react';

/* ── 模块级单例状态 ── */
let _resolve = null;
let _state = { visible: false, title: '', message: '', icon: null, confirmText: '确定', cancelText: '取消', confirmType: 'warning' };
const _listeners = new Set();

function _notify() {
  _listeners.forEach(fn => fn({ ..._state }));
}

/**
 * 调用后弹出自定义确认框，返回 Promise<boolean>。
 * @param {string} title
 * @param {string|ReactNode} message
 * @param {object} [opts]  icon / confirmText / cancelText / confirmType('danger'|'warning'|'primary')
 */
export function farmConfirm(title, message, opts = {}) {
  return new Promise(resolve => {
    _resolve = resolve;
    _state = {
      visible: true,
      title,
      message,
      icon: opts.icon ?? null,
      confirmText: opts.confirmText ?? '确定',
      cancelText:  opts.cancelText  ?? '取消',
      confirmType: opts.confirmType ?? 'warning',
    };
    _notify();
  });
}

function _close(result) {
  _state = { ..._state, visible: false };
  _notify();
  if (_resolve) { _resolve(result); _resolve = null; }
}

/* ── Provider（挂载一次即可） ── */
export const FarmConfirmProvider = () => {
  const [s, setS] = useState({ ..._state });

  useEffect(() => {
    _listeners.add(setS);
    return () => _listeners.delete(setS);
  }, []);

  if (!s.visible) return null;

  const colorMap = {
    danger:  'var(--farm-danger)',
    warning: 'var(--farm-harvest)',
    primary: 'var(--farm-leaf)',
  };
  const accentColor = colorMap[s.confirmType] || colorMap.warning;

  return (
    <div className='farm-modal-overlay' onClick={() => _close(false)}>
      <div className='farm-modal-container' onClick={e => e.stopPropagation()}>
        <div className='farm-modal-icon' style={{ color: accentColor }}>
          {s.icon
            ? (typeof s.icon === 'string'
                ? <span style={{ fontSize: 32 }}>{s.icon}</span>
                : s.icon)
            : <AlertTriangle size={28} />}
        </div>
        {s.title && <div className='farm-modal-title'>{s.title}</div>}
        <div className='farm-modal-message'>{s.message}</div>
        <div className='farm-modal-buttons'>
          <Button className='farm-btn farm-modal-btn-cancel' theme='borderless' onClick={() => _close(false)}>
            {s.cancelText}
          </Button>
          <Button
            className='farm-btn farm-modal-btn-confirm'
            theme='solid'
            type={s.confirmType}
            onClick={() => _close(true)}
          >
            {s.confirmText}
          </Button>
        </div>
      </div>
    </div>
  );
};
