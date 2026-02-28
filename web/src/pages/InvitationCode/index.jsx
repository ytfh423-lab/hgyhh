import React, { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  copy,
  showError,
  showSuccess,
  timestamp2string,
} from '../../helpers';
import {
  Button,
  Card,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { Gift, Copy, AlertCircle } from 'lucide-react';
import { StatusContext } from '../../context/Status';

const { Text, Title } = Typography;

const InvitationCode = () => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const enabled = statusState?.status?.invitation_code_enabled;
  const [loading, setLoading] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [records, setRecords] = useState([]);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  const loadData = useCallback(
    async (page = activePage, size = pageSize) => {
      setLoading(true);
      try {
        const { data: resData } = await API.get(
          `/api/invitation-code/?p=${page}&page_size=${size}`,
        );
        const { success, message, data } = resData;
        if (!success) {
          showError(message || t('加载失败'));
          return;
        }
        setRecords(data?.items || []);
        setTotal(data?.total || 0);
      } catch (error) {
        showError(error.message || t('加载失败'));
      } finally {
        setLoading(false);
      }
    },
    [activePage, pageSize, t],
  );

  useEffect(() => {
    loadData();
  }, [loadData]);

  const generateCode = async () => {
    setGenerating(true);
    try {
      const { data: resData } = await API.post('/api/invitation-code/');
      const { success, message, data } = resData;
      if (!success) {
        showError(message || t('生成失败'));
        return;
      }
      if (data?.key) {
        copy(data.key);
        showSuccess(t('邀请码生成成功，已复制到剪贴板'));
      } else {
        showSuccess(t('邀请码生成成功'));
      }
      loadData(1, pageSize);
      setActivePage(1);
    } catch (error) {
      showError(error.message || t('生成失败'));
    } finally {
      setGenerating(false);
    }
  };

  const getStatusInfo = (row) => {
    const now = Math.floor(Date.now() / 1000);
    if (row.status === 3) return { color: 'red', text: t('已使用') };
    if (row.status === 2) return { color: 'orange', text: t('禁用') };
    if (row.expired_time && row.expired_time > 0 && row.expired_time < now)
      return { color: 'grey', text: t('已过期') };
    return { color: 'green', text: t('可用') };
  };

  const columns = useMemo(
    () => [
      {
        title: t('邀请码'),
        dataIndex: 'key',
        render: (value) => (
          <Space>
            <Text copyable={{ content: value }} style={{ fontFamily: 'monospace', fontSize: '13px' }}>
              {value}
            </Text>
          </Space>
        ),
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 100,
        render: (_, row) => {
          const info = getStatusInfo(row);
          return <Tag color={info.color}>{info.text}</Tag>;
        },
      },
      {
        title: t('创建时间'),
        dataIndex: 'created_time',
        width: 180,
        render: (v) => (v ? timestamp2string(v) : '-'),
      },
      {
        title: t('过期时间'),
        dataIndex: 'expired_time',
        width: 180,
        render: (v) =>
          v && v > 0 ? timestamp2string(v) : t('永不过期'),
      },
      {
        title: t('使用者ID'),
        dataIndex: 'used_user_id',
        width: 100,
        render: (v) => v || '-',
      },
    ],
    [t],
  );

  return (
    <div className='mt-[60px] px-2' style={{ maxWidth: 900, margin: '60px auto 0' }}>
      <Card
        className='!rounded-2xl shadow-sm'
        style={{ border: '1px solid var(--semi-color-border)' }}
      >
        <div className='flex items-center mb-4'>
          <div
            className='mr-3'
            style={{
              width: '40px',
              height: '40px',
              borderRadius: '12px',
              background: 'linear-gradient(135deg, #6366f1, #8b5cf6)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              boxShadow: '0 2px 8px rgba(99, 102, 241, 0.3)',
              flexShrink: 0,
            }}
          >
            <Gift size={20} style={{ color: 'white' }} />
          </div>
          <div style={{ flex: 1 }}>
            <Title heading={5} style={{ margin: 0 }}>
              {t('邀请码生成')}
            </Title>
            <Text type='tertiary' size='small'>
              {t('生成邀请码分享给好友注册，每日最多 2 个，有效期 24 小时')}
            </Text>
          </div>
          <Button
            theme='solid'
            type='primary'
            icon={<Gift size={16} />}
            loading={generating}
            onClick={generateCode}
            disabled={!enabled}
            style={{ borderRadius: '10px' }}
          >
            {t('生成邀请码')}
          </Button>
        </div>

        {!enabled && (
          <div
            style={{
              background: 'rgba(239, 68, 68, 0.06)',
              border: '1px solid rgba(239, 68, 68, 0.2)',
              borderRadius: '10px',
              padding: '10px 14px',
              marginBottom: '16px',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}
          >
            <AlertCircle size={16} style={{ color: '#dc2626', flexShrink: 0 }} />
            <Text size='small' style={{ color: '#dc2626' }}>
              {t('管理员已关闭邀请码功能')}
            </Text>
          </div>
        )}

        <div
          style={{
            background: 'rgba(245, 158, 11, 0.06)',
            border: '1px solid rgba(245, 158, 11, 0.2)',
            borderRadius: '10px',
            padding: '10px 14px',
            marginBottom: '16px',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
          }}
        >
          <AlertCircle size={16} style={{ color: '#b45309', flexShrink: 0 }} />
          <Text size='small' style={{ color: '#b45309' }}>
            {t('每个邀请码仅可使用一次，过期后自动作废。请将邀请码分享给信任的朋友。')}
          </Text>
        </div>

        <Table
          loading={loading}
          columns={columns}
          dataSource={records}
          rowKey='id'
          size='small'
          pagination={{
            currentPage: activePage,
            pageSize,
            total,
            onPageChange: (page) => {
              setActivePage(page);
              loadData(page, pageSize);
            },
            onPageSizeChange: (size) => {
              setPageSize(size);
              setActivePage(1);
              loadData(1, size);
            },
            showSizeChanger: true,
          }}
          empty={
            <div style={{ padding: '40px 0', textAlign: 'center' }}>
              <Gift size={40} style={{ color: 'var(--semi-color-text-3)', marginBottom: '12px' }} />
              <div>
                <Text type='tertiary'>{t('还没有生成过邀请码')}</Text>
              </div>
            </div>
          }
        />
      </Card>
    </div>
  );
};

export default InvitationCode;
