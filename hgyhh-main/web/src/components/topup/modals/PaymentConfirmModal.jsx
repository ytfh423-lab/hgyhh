/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Modal, Typography, Card, Skeleton } from '@douyinfe/semi-ui';
import { SiAlipay, SiWechat, SiStripe } from 'react-icons/si';
import { CreditCard } from 'lucide-react';

const { Text } = Typography;

const PaymentConfirmModal = ({
  t,
  open,
  onlineTopUp,
  handleCancel,
  confirmLoading,
  topUpCount,
  renderQuotaWithAmount,
  amountLoading,
  renderAmount,
  payWay,
  payMethods,
  // 新增：用于显示折扣明细
  amountNumber,
  discountRate,
}) => {
  const hasDiscount =
    discountRate && discountRate > 0 && discountRate < 1 && amountNumber > 0;
  const originalAmount = hasDiscount ? amountNumber / discountRate : 0;
  const discountAmount = hasDiscount ? originalAmount - amountNumber : 0;
  const getPayIcon = () => {
    const payMethod = payMethods.find((method) => method.type === payWay);
    const type = payMethod?.type || payWay;
    const name = payMethod?.name || (type === 'alipay' ? t('支付宝') : type === 'stripe' ? 'Stripe' : type === 'wxpay' ? t('微信') : type);
    const icon = type === 'alipay' ? <SiAlipay size={16} color='#1677FF' />
      : type === 'wxpay' ? <SiWechat size={16} color='#07C160' />
      : type === 'stripe' ? <SiStripe size={16} color='#635BFF' />
      : <CreditCard size={16} color={payMethod?.color || 'var(--semi-color-text-2)'} />;
    return { icon, name };
  };

  const { icon: payIcon, name: payName } = getPayIcon();

  return (
    <Modal
      title={
        <div className='flex items-center gap-2'>
          <div style={{
            width: '28px',
            height: '28px',
            borderRadius: '8px',
            background: 'linear-gradient(135deg, #3b82f6, #6366f1)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            <CreditCard size={15} style={{ color: 'white' }} />
          </div>
          {t('充值确认')}
        </div>
      }
      visible={open}
      onOk={onlineTopUp}
      onCancel={handleCancel}
      maskClosable={false}
      size='small'
      centered
      confirmLoading={confirmLoading}
    >
      <div style={{
        borderRadius: '16px',
        background: 'var(--semi-color-bg-1)',
        border: '1px solid var(--semi-color-border)',
        overflow: 'hidden',
      }}>
        {/* 充值数量 */}
        <div style={{ padding: '16px 20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Text style={{ color: 'var(--semi-color-text-2)', fontSize: '14px' }}>
            {t('充值数量')}
          </Text>
          <Text strong style={{ fontSize: '14px' }}>
            {renderQuotaWithAmount(topUpCount)}
          </Text>
        </div>

        <div style={{ height: '1px', background: 'var(--semi-color-border)', margin: '0 20px' }} />

        {/* 支付方式 */}
        <div style={{ padding: '16px 20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Text style={{ color: 'var(--semi-color-text-2)', fontSize: '14px' }}>
            {t('支付方式')}
          </Text>
          <div className='flex items-center gap-2'>
            {payIcon}
            <Text strong style={{ fontSize: '14px' }}>{payName}</Text>
          </div>
        </div>

        {hasDiscount && !amountLoading && (
          <>
            <div style={{ height: '1px', background: 'var(--semi-color-border)', margin: '0 20px' }} />
            <div style={{ padding: '16px 20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Text style={{ color: 'var(--semi-color-text-2)', fontSize: '14px' }}>
                {t('原价')}
              </Text>
              <Text delete style={{ color: 'var(--semi-color-text-2)', fontSize: '14px' }}>
                {`${originalAmount.toFixed(2)} ${t('元')}`}
              </Text>
            </div>
            <div style={{ height: '1px', background: 'var(--semi-color-border)', margin: '0 20px' }} />
            <div style={{ padding: '16px 20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Text style={{ color: '#10b981', fontSize: '14px' }}>
                {t('优惠')}
              </Text>
              <Text style={{ color: '#10b981', fontSize: '14px', fontWeight: 600 }}>
                {`-${discountAmount.toFixed(2)} ${t('元')}`}
              </Text>
            </div>
          </>
        )}

        <div style={{ height: '1px', background: 'var(--semi-color-border)', margin: '0 20px' }} />

        {/* 实付金额 */}
        <div style={{
          padding: '18px 20px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          background: 'var(--semi-color-bg-2)',
        }}>
          <Text strong style={{ fontSize: '15px' }}>
            {t('实付金额')}
          </Text>
          {amountLoading ? (
            <Skeleton.Title style={{ width: '80px', height: '20px' }} />
          ) : (
            <div className='flex items-baseline gap-2'>
              <Text strong style={{ fontSize: '20px', color: '#ef4444', fontWeight: 700 }}>
                {renderAmount()}
              </Text>
              {hasDiscount && (
                <span style={{
                  background: 'linear-gradient(135deg, #10b981, #059669)',
                  color: 'white',
                  fontSize: '11px',
                  fontWeight: 600,
                  padding: '2px 8px',
                  borderRadius: '6px',
                }}>
                  {Math.round(discountRate * 100)}%
                </span>
              )}
            </div>
          )}
        </div>
      </div>
    </Modal>
  );
};

export default PaymentConfirmModal;
