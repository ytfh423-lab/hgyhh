import React from 'react';
import { Button } from '@douyinfe/semi-ui';
import { AlertTriangle } from 'lucide-react';

/**
 * FarmConfirmModal — 农场风格自定义确认弹窗
 *
 * Props:
 *   visible     {boolean}   是否显示
 *   title       {string}    标题
 *   message     {string|ReactNode}  内容
 *   icon        {ReactNode} 自定义图标（默认 ⚠️ AlertTriangle）
 *   confirmText {string}    确认按钮文字（默认 "确定"）
 *   cancelText  {string}    取消按钮文字（默认 "取消"）
 *   confirmType {string}    确认按钮类型：'danger' | 'warning' | 'primary'（默认 'warning'）
 *   loading     {boolean}   确认按钮 loading 状态
 *   onConfirm   {function}  点击确认
 *   onCancel    {function}  点击取消 / 关闭
 */
const FarmConfirmModal = ({
  visible,
  title,
  message,
  icon,
  confirmText = '确定',
  cancelText = '取消',
  confirmType = 'warning',
  loading = false,
  onConfirm,
  onCancel,
}) => {
  if (!visible) return null;

  return (
    <div className='farm-modal-overlay' onClick={onCancel}>
      <div className='farm-modal-container' onClick={e => e.stopPropagation()}>
        {/* Icon */}
        <div className='farm-modal-icon'>
          {icon || <AlertTriangle size={28} />}
        </div>

        {/* Title */}
        {title && <div className='farm-modal-title'>{title}</div>}

        {/* Message */}
        <div className='farm-modal-message'>{message}</div>

        {/* Buttons */}
        <div className='farm-modal-buttons'>
          <Button
            className='farm-btn farm-modal-btn-cancel'
            theme='borderless'
            onClick={onCancel}
            disabled={loading}
          >
            {cancelText}
          </Button>
          <Button
            className='farm-btn farm-modal-btn-confirm'
            theme='solid'
            type={confirmType}
            onClick={onConfirm}
            loading={loading}
          >
            {confirmText}
          </Button>
        </div>
      </div>
    </div>
  );
};

export default FarmConfirmModal;
