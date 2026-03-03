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
  Empty,
  Modal,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { Gift, Copy, Share2, Trash2, RefreshCw } from 'lucide-react';
import { UserContext } from '../../context/User';

const { Text, Title } = Typography;

const PublicInviteCode = () => {
  const { t } = useTranslation();
  const [userState] = useContext(UserContext);
  const isLoggedIn = !!userState?.user?.username;
  const [codes, setCodes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [shareModalVisible, setShareModalVisible] = useState(false);
  const [myCodes, setMyCodes] = useState([]);
  const [myCodesLoading, setMyCodesLoading] = useState(false);
  const [selectedCodeId, setSelectedCodeId] = useState(null);
  const [sharing, setSharing] = useState(false);

  const loadCodes = useCallback(async () => {
    setLoading(true);
    try {
      const { data: resData } = await API.get('/api/public_invcode/');
      if (resData.success) {
        setCodes(resData.data || []);
      }
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadCodes();
  }, [loadCodes]);

  const loadMyCodes = async () => {
    setMyCodesLoading(true);
    try {
      const { data: resData } = await API.get('/api/public_invcode/my');
      if (resData.success) {
        setMyCodes(resData.data || []);
      } else {
        showError(resData.message || t('加载失败'));
      }
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setMyCodesLoading(false);
    }
  };

  const handleShare = async () => {
    if (!selectedCodeId) {
      showError(t('请选择要分享的邀请码'));
      return;
    }
    setSharing(true);
    try {
      const { data: resData } = await API.post('/api/public_invcode/', {
        code_id: selectedCodeId,
      });
      if (resData.success) {
        showSuccess(t('分享成功'));
        setShareModalVisible(false);
        setSelectedCodeId(null);
        loadCodes();
      } else {
        showError(resData.message || t('分享失败'));
      }
    } catch (err) {
      showError(t('分享失败'));
    } finally {
      setSharing(false);
    }
  };

  const handleDelete = async (id) => {
    try {
      const { data: resData } = await API.delete(`/api/public_invcode/${id}`);
      if (resData.success) {
        showSuccess(t('已删除'));
        loadCodes();
      } else {
        showError(resData.message || t('删除失败'));
      }
    } catch (err) {
      showError(t('删除失败'));
    }
  };

  const getStatusTag = (status) => {
    switch (status) {
      case 1:
        return <Tag color='green' size='small'>{t('可用')}</Tag>;
      case 2:
        return <Tag color='red' size='small'>{t('已使用')}</Tag>;
      case 3:
        return <Tag color='grey' size='small'>{t('已过期')}</Tag>;
      default:
        return <Tag color='grey' size='small'>{t('未知')}</Tag>;
    }
  };

  const columns = useMemo(
    () => [
      {
        title: t('分享者'),
        dataIndex: 'username',
        width: 120,
        render: (v) => (
          <Text strong style={{ fontSize: 13 }}>
            {v || '-'}
          </Text>
        ),
      },
      {
        title: t('邀请码'),
        dataIndex: 'code',
        render: (value, row) => (
          <Space>
            <Text
              style={{
                fontFamily: 'monospace',
                fontSize: '13px',
                letterSpacing: '0.5px',
              }}
            >
              {value}
            </Text>
            {row.status === 1 && (
              <Tooltip content={t('复制邀请码')}>
                <Button
                  icon={<Copy size={14} />}
                  size='small'
                  theme='borderless'
                  onClick={() => {
                    copy(value);
                    showSuccess(t('已复制到剪贴板'));
                  }}
                />
              </Tooltip>
            )}
          </Space>
        ),
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 90,
        render: (v) => getStatusTag(v),
      },
      {
        title: t('过期时间'),
        dataIndex: 'expired_time',
        width: 170,
        render: (v) => (v && v > 0 ? timestamp2string(v) : t('永不过期')),
      },
      ...(isLoggedIn
        ? [
            {
              title: t('操作'),
              width: 70,
              render: (_, row) => {
                if (row.user_id === userState?.user?.id) {
                  return (
                    <Tooltip content={t('取消分享')}>
                      <Button
                        icon={<Trash2 size={14} />}
                        size='small'
                        type='danger'
                        theme='borderless'
                        onClick={() => handleDelete(row.id)}
                      />
                    </Tooltip>
                  );
                }
                return null;
              },
            },
          ]
        : []),
    ],
    [t, isLoggedIn, userState],
  );

  return (
    <div className='mt-[60px] px-2' style={{ maxWidth: 960, margin: '60px auto 0' }}>
      <Card
        className='!rounded-2xl shadow-sm'
        style={{ border: '1px solid var(--semi-color-border)' }}
      >
        <div className='flex items-center mb-4' style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
          <div
            className='mr-3'
            style={{
              width: '40px',
              height: '40px',
              borderRadius: '12px',
              background: 'linear-gradient(135deg, #10b981, #059669)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              boxShadow: '0 2px 8px rgba(16, 185, 129, 0.3)',
              flexShrink: 0,
              marginRight: 12,
            }}
          >
            <Gift size={20} style={{ color: 'white' }} />
          </div>
          <div style={{ flex: 1 }}>
            <Title heading={5} style={{ margin: 0 }}>
              {t('公开邀请码')}
            </Title>
            <Text type='tertiary' size='small'>
              {t('用户分享的公开邀请码，可直接复制使用注册账号')}
            </Text>
          </div>
          <Space>
            <Button
              icon={<RefreshCw size={14} />}
              theme='borderless'
              onClick={loadCodes}
              style={{ borderRadius: '10px' }}
            >
              {t('刷新')}
            </Button>
            {isLoggedIn && (
              <Button
                theme='solid'
                type='primary'
                icon={<Share2 size={16} />}
                onClick={() => {
                  setShareModalVisible(true);
                  loadMyCodes();
                }}
                style={{ borderRadius: '10px' }}
              >
                {t('分享邀请码')}
              </Button>
            )}
          </Space>
        </div>

        <div
          style={{
            background: 'rgba(16, 185, 129, 0.06)',
            border: '1px solid rgba(16, 185, 129, 0.2)',
            borderRadius: '10px',
            padding: '10px 14px',
            marginBottom: '16px',
          }}
        >
          <Text size='small' style={{ color: '#059669' }}>
            💡 {t('提示：绿色「可用」状态的邀请码可以直接复制，用于注册新账号。每个邀请码只能使用一次。')}
          </Text>
        </div>

        <Table
          loading={loading}
          columns={columns}
          dataSource={codes}
          rowKey='id'
          size='small'
          pagination={false}
          empty={
            <div style={{ padding: '40px 0', textAlign: 'center' }}>
              <Gift size={40} style={{ color: 'var(--semi-color-text-3)', marginBottom: '12px' }} />
              <div>
                <Text type='tertiary'>{t('暂时没有公开的邀请码')}</Text>
              </div>
              {isLoggedIn && (
                <div style={{ marginTop: 8 }}>
                  <Text type='tertiary' size='small'>
                    {t('登录后可以生成邀请码并分享到这里')}
                  </Text>
                </div>
              )}
            </div>
          }
        />
      </Card>

      <Modal
        title={t('分享邀请码')}
        visible={shareModalVisible}
        onOk={handleShare}
        onCancel={() => {
          setShareModalVisible(false);
          setSelectedCodeId(null);
        }}
        okText={t('分享到公开列表')}
        cancelText={t('取消')}
        confirmLoading={sharing}
        style={{ borderRadius: '16px' }}
      >
        <div style={{ marginBottom: 12 }}>
          <Text type='tertiary' size='small'>
            {t('选择一个可用的邀请码分享到公开列表，其他用户可以直接复制使用。')}
          </Text>
        </div>
        {myCodesLoading ? (
          <div style={{ textAlign: 'center', padding: 20 }}>
            <Spin />
          </div>
        ) : myCodes.length === 0 ? (
          <Empty
            description={t('没有可分享的邀请码，请先到邀请码页面生成')}
          />
        ) : (
          <Select
            placeholder={t('选择邀请码')}
            value={selectedCodeId}
            onChange={setSelectedCodeId}
            style={{ width: '100%' }}
            renderOptionItem={({ disabled, selected, label, value, onMouseEnter, onClick, style: optionStyle, className: optionClassName }) => {
              const code = myCodes.find((c) => c.id === value);
              if (!code) return null;
              const statusColors = {
                '可用': 'green',
                '已使用': 'red',
                '已过期': 'grey',
                '已分享': 'blue',
                '已禁用': 'orange',
              };
              return (
                <div
                  className={optionClassName}
                  style={{
                    ...optionStyle,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '8px 12px',
                    cursor: disabled ? 'not-allowed' : 'pointer',
                    opacity: disabled ? 0.5 : 1,
                  }}
                  onClick={disabled ? undefined : onClick}
                  onMouseEnter={onMouseEnter}
                >
                  <span style={{ fontFamily: 'monospace', fontSize: 13 }}>
                    {code.key.substring(0, 12)}...
                  </span>
                  <Space spacing={4}>
                    <Tag size='small' color={statusColors[code.status_label] || 'grey'}>
                      {t(code.status_label)}
                    </Tag>
                    <Text type='tertiary' size='small'>
                      {code.expired_time && code.expired_time > 0
                        ? timestamp2string(code.expired_time)
                        : t('永不过期')}
                    </Text>
                  </Space>
                </div>
              );
            }}
            optionList={myCodes.map((code) => ({
              label: `${code.key.substring(0, 12)}... [${code.status_label}]`,
              value: code.id,
              disabled: !code.shareable,
            }))}
          />
        )}
      </Modal>
    </div>
  );
};

export default PublicInviteCode;
