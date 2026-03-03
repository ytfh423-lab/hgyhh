import React, { useEffect, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Modal,
  Table,
  Tag,
  TextArea,
  Typography,
  Select,
  Space,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';

const STATUS_OPTIONS = [
  { value: '', label: '全部' },
  { value: '0', label: '待审核' },
  { value: '1', label: '已通过' },
  { value: '2', label: '已拒绝' },
];

const statusTagMap = {
  0: { color: 'orange', text: '待审核' },
  1: { color: 'green', text: '已通过' },
  2: { color: 'red', text: '已拒绝' },
};

const DeletionRequestPage = () => {
  const { t } = useTranslation();
  const [requests, setRequests] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [remarkModal, setRemarkModal] = useState({ visible: false, id: 0, action: '' });
  const [adminRemark, setAdminRemark] = useState('');

  const loadData = async () => {
    setLoading(true);
    try {
      let url = `/api/deletion-request/?page=${page}&page_size=${pageSize}`;
      if (statusFilter !== '') {
        url += `&status=${statusFilter}`;
      }
      const res = await API.get(url);
      if (res.data.success) {
        setRequests(res.data.data || []);
        setTotal(res.data.total || 0);
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('加载失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [page, statusFilter]);

  const handleAction = (id, action) => {
    setRemarkModal({ visible: true, id, action });
    setAdminRemark('');
  };

  const confirmAction = async () => {
    setActionLoading(true);
    try {
      const url = `/api/deletion-request/${remarkModal.id}/${remarkModal.action}`;
      const res = await API.post(url, { admin_remark: adminRemark });
      if (res.data.success) {
        showSuccess(res.data.message);
        setRemarkModal({ visible: false, id: 0, action: '' });
        await loadData();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('操作失败'));
    } finally {
      setActionLoading(false);
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: t('用户名'),
      dataIndex: 'username',
      width: 120,
    },
    {
      title: t('用户ID'),
      dataIndex: 'user_id',
      width: 80,
    },
    {
      title: t('注销理由'),
      dataIndex: 'reason',
      render: (text) => (
        <Typography.Paragraph
          ellipsis={{ rows: 2, expandable: true }}
          style={{ marginBottom: 0 }}
        >
          {text}
        </Typography.Paragraph>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 90,
      render: (status) => {
        const info = statusTagMap[status] || { color: 'grey', text: '未知' };
        return <Tag color={info.color}>{t(info.text)}</Tag>;
      },
    },
    {
      title: t('管理员备注'),
      dataIndex: 'admin_remark',
      width: 150,
      render: (text) => text || '-',
    },
    {
      title: t('提交时间'),
      dataIndex: 'created_at',
      width: 160,
      render: (ts) => ts ? new Date(ts * 1000).toLocaleString() : '-',
    },
    {
      title: t('操作'),
      width: 160,
      fixed: 'right',
      render: (_, record) => {
        if (record.status !== 0) return <Typography.Text type='tertiary'>-</Typography.Text>;
        return (
          <Space>
            <Button
              size='small'
              type='danger'
              theme='solid'
              onClick={() => handleAction(record.id, 'approve')}
            >
              {t('通过')}
            </Button>
            <Button
              size='small'
              type='tertiary'
              onClick={() => handleAction(record.id, 'reject')}
            >
              {t('拒绝')}
            </Button>
          </Space>
        );
      },
    },
  ];

  return (
    <div className='mt-[60px] px-2'>
      <Card
        className='!rounded-2xl'
        style={{
          boxShadow: '0 1px 3px rgba(0,0,0,0.04), 0 4px 16px rgba(0,0,0,0.02)',
          border: '1px solid var(--semi-color-border)',
        }}
      >
        <div className='flex items-center justify-between mb-4 flex-wrap gap-2'>
          <Typography.Title heading={5} style={{ marginBottom: 0 }}>
            {t('注销审核')}
          </Typography.Title>
          <Select
            value={statusFilter}
            onChange={setStatusFilter}
            style={{ width: 120 }}
            optionList={STATUS_OPTIONS.map((o) => ({ ...o, label: t(o.label) }))}
          />
        </div>

        <Table
          columns={columns}
          dataSource={requests}
          loading={loading}
          rowKey='id'
          pagination={{
            currentPage: page,
            pageSize,
            total,
            onPageChange: setPage,
          }}
          scroll={{ x: 900 }}
          empty={
            <div className='py-8 text-center text-gray-400'>
              {t('暂无注销申请')}
            </div>
          }
        />
      </Card>

      <Modal
        title={remarkModal.action === 'approve' ? t('确认通过注销申请') : t('确认拒绝注销申请')}
        visible={remarkModal.visible}
        onCancel={() => setRemarkModal({ visible: false, id: 0, action: '' })}
        onOk={confirmAction}
        okButtonProps={{ loading: actionLoading }}
        centered
        size='small'
      >
        <div className='space-y-3 py-2'>
          {remarkModal.action === 'approve' && (
            <Banner
              type='danger'
              description={t('通过后该用户账户将被永久删除，此操作不可逆')}
              closeIcon={null}
              className='!rounded-lg'
            />
          )}
          <div>
            <Typography.Text strong className='block mb-2'>
              {t('管理员备注')}（{t('可选')}）
            </Typography.Text>
            <TextArea
              placeholder={t('输入备注...')}
              value={adminRemark}
              onChange={setAdminRemark}
              rows={3}
              className='!rounded-lg'
            />
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default DeletionRequestPage;
