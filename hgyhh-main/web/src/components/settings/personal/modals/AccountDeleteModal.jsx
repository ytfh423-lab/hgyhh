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

import React, { useEffect, useState } from 'react';
import { Banner, Input, Modal, Typography, Tag, TextArea } from '@douyinfe/semi-ui';
import { IconDelete } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../../../helpers';

const AccountDeleteModal = ({
  t,
  showAccountDeleteModal,
  setShowAccountDeleteModal,
}) => {
  const [reason, setReason] = useState('');
  const [pendingRequest, setPendingRequest] = useState(null);
  const [loading, setLoading] = useState(false);

  const loadPendingRequest = async () => {
    try {
      const res = await API.get('/api/user/deletion-request');
      if (res.data.success && res.data.data) {
        setPendingRequest(res.data.data);
      } else {
        setPendingRequest(null);
      }
    } catch {
      setPendingRequest(null);
    }
  };

  useEffect(() => {
    if (showAccountDeleteModal) {
      loadPendingRequest();
    }
  }, [showAccountDeleteModal]);

  const submitDeletionRequest = async () => {
    if (!reason.trim()) {
      showError(t('请填写注销理由'));
      return;
    }
    setLoading(true);
    try {
      const res = await API.delete('/api/user/self', { data: { reason: reason.trim() } });
      if (res.data.success) {
        showSuccess(res.data.message || t('注销申请已提交'));
        setReason('');
        await loadPendingRequest();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('提交失败'));
    } finally {
      setLoading(false);
    }
  };

  const cancelDeletionRequest = async () => {
    setLoading(true);
    try {
      const res = await API.post('/api/user/deletion-request/cancel');
      if (res.data.success) {
        showSuccess(res.data.message || t('注销申请已取消'));
        setPendingRequest(null);
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('操作失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={
        <div className='flex items-center'>
          <IconDelete className='mr-2 text-red-500' />
          {t('注销账户')}
        </div>
      }
      visible={showAccountDeleteModal}
      onCancel={() => setShowAccountDeleteModal(false)}
      onOk={pendingRequest ? cancelDeletionRequest : submitDeletionRequest}
      okText={pendingRequest ? t('取消申请') : t('提交注销申请')}
      okButtonProps={{
        type: pendingRequest ? 'tertiary' : 'danger',
        loading,
      }}
      size={'small'}
      centered={true}
      className='modern-modal'
    >
      <div className='space-y-4 py-4'>
        {pendingRequest ? (
          <>
            <Banner
              type='warning'
              description={t('您已提交注销申请，正在等待管理员审核')}
              closeIcon={null}
              className='!rounded-lg'
            />
            <div className='space-y-2'>
              <div>
                <Typography.Text strong>{t('注销理由')}：</Typography.Text>
                <Typography.Text>{pendingRequest.reason}</Typography.Text>
              </div>
              <div>
                <Typography.Text strong>{t('状态')}：</Typography.Text>
                <Tag color='orange' className='ml-1'>{t('待审核')}</Tag>
              </div>
              <div>
                <Typography.Text strong>{t('提交时间')}：</Typography.Text>
                <Typography.Text>
                  {new Date(pendingRequest.created_at * 1000).toLocaleString()}
                </Typography.Text>
              </div>
            </div>
          </>
        ) : (
          <>
            <Banner
              type='danger'
              description={t('注销账户需要管理员审核，审核通过后账户将被永久删除且不可恢复')}
              closeIcon={null}
              className='!rounded-lg'
            />
            <div>
              <Typography.Text strong className='block mb-2'>
                {t('请填写注销理由')}
              </Typography.Text>
              <TextArea
                placeholder={t('请说明您注销账户的原因...')}
                value={reason}
                onChange={setReason}
                maxCount={500}
                rows={4}
                className='!rounded-lg'
              />
            </div>
          </>
        )}
      </div>
    </Modal>
  );
};

export default AccountDeleteModal;
