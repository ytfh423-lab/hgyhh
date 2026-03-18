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
import { Modal, Typography, Input, InputNumber } from '@douyinfe/semi-ui';
import { ArrowRightLeft } from 'lucide-react';

const TransferModal = ({
  t,
  openTransfer,
  transfer,
  handleTransferCancel,
  userState,
  renderQuota,
  getQuotaPerUnit,
  transferAmount,
  setTransferAmount,
}) => {
  return (
    <Modal
      title={
        <div className='flex items-center gap-2'>
          <div style={{
            width: '28px',
            height: '28px',
            borderRadius: '8px',
            background: 'linear-gradient(135deg, #10b981, #059669)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            <ArrowRightLeft size={14} style={{ color: 'white' }} />
          </div>
          {t('划转邀请额度')}
        </div>
      }
      visible={openTransfer}
      onOk={transfer}
      onCancel={handleTransferCancel}
      maskClosable={false}
      centered
    >
      <div style={{
        borderRadius: '16px',
        background: 'var(--semi-color-bg-1)',
        border: '1px solid var(--semi-color-border)',
        overflow: 'hidden',
      }}>
        <div style={{ padding: '16px 20px' }}>
          <Typography.Text style={{ color: 'var(--semi-color-text-2)', fontSize: '13px', display: 'block', marginBottom: '8px' }}>
            {t('可用邀请额度')}
          </Typography.Text>
          <Input
            value={renderQuota(userState?.user?.aff_quota)}
            disabled
            style={{ borderRadius: '10px' }}
          />
        </div>
        <div style={{ height: '1px', background: 'var(--semi-color-border)', margin: '0 20px' }} />
        <div style={{ padding: '16px 20px' }}>
          <Typography.Text style={{ color: 'var(--semi-color-text-2)', fontSize: '13px', display: 'block', marginBottom: '8px' }}>
            {t('划转额度')} · {t('最低') + renderQuota(getQuotaPerUnit())}
          </Typography.Text>
          <InputNumber
            min={getQuotaPerUnit()}
            max={userState?.user?.aff_quota || 0}
            value={transferAmount}
            onChange={(value) => setTransferAmount(value)}
            style={{ width: '100%', borderRadius: '10px' }}
          />
        </div>
      </div>
    </Modal>
  );
};

export default TransferModal;
