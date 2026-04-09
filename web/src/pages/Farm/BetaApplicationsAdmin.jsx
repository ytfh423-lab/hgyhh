import React, { useEffect, useState, useCallback } from 'react';
import {
  Button, Card, Input, Modal, Table, Tag, Typography, Select, Space, Descriptions, Spin, Empty, Pagination,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './components/utils';
import { useTranslation } from 'react-i18next';

const { Text, Title } = Typography;

const STATUS_MAP = {
  pending: { text: '待审核', color: 'orange' },
  approved: { text: '已通过', color: 'green' },
  rejected: { text: '已拒绝', color: 'red' },
};

const AI_DECISION_MAP = {
  approve: { text: '建议通过', color: 'green' },
  reject: { text: '建议拒绝', color: 'red' },
  manual_review: { text: '转人工', color: 'orange' },
  error: { text: 'AI错误', color: 'grey' },
};

const formatTime = (ts) => {
  if (!ts) return '-';
  return new Date(ts * 1000).toLocaleString('zh-CN');
};

const BetaApplicationsAdmin = () => {
  const { t } = useTranslation();
  const [list, setList] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(15);
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(false);

  // 详情弹窗
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailData, setDetailData] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);

  // 审核操作
  const [reviewNote, setReviewNote] = useState('');
  const [actionLoading, setActionLoading] = useState(false);

  const loadList = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page, page_size: pageSize });
      if (statusFilter) params.set('status', statusFilter);
      const { data: res } = await API.get(`/api/tgbot/farm/beta-applications?${params}`);
      if (res.success) {
        setList(res.data.list || []);
        setTotal(res.data.total || 0);
      } else {
        showError(res.message);
      }
    } catch (e) {
      showError('加载失败');
    }
    setLoading(false);
  }, [page, pageSize, statusFilter]);

  useEffect(() => { loadList(); }, [loadList]);

  const loadDetail = async (id) => {
    setDetailLoading(true);
    setDetailVisible(true);
    setReviewNote('');
    try {
      const { data: res } = await API.get(`/api/tgbot/farm/beta-application/detail?id=${id}`);
      if (res.success) {
        setDetailData(res.data);
      } else {
        showError(res.message);
        setDetailVisible(false);
      }
    } catch (e) {
      showError('加载详情失败');
      setDetailVisible(false);
    }
    setDetailLoading(false);
  };

  const handleApprove = async () => {
    if (!detailData) return;
    setActionLoading(true);
    try {
      const { data: res } = await API.post('/api/tgbot/farm/beta-application/approve', {
        app_id: detailData.id,
        review_note: reviewNote,
      });
      if (res.success) {
        showSuccess(res.message);
        setDetailVisible(false);
        loadList();
      } else {
        showError(res.message);
      }
    } catch (e) {
      showError('操作失败');
    }
    setActionLoading(false);
  };

  const handleReject = async () => {
    if (!detailData) return;
    setActionLoading(true);
    try {
      const { data: res } = await API.post('/api/tgbot/farm/beta-application/reject', {
        app_id: detailData.id,
        review_note: reviewNote,
      });
      if (res.success) {
        showSuccess(res.message);
        setDetailVisible(false);
        loadList();
      } else {
        showError(res.message);
      }
    } catch (e) {
      showError('操作失败');
    }
    setActionLoading(false);
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '用户',
      dataIndex: 'username',
      width: 140,
      render: (text, record) => (
        <div>
          <div style={{ fontWeight: 600 }}>{record.display_name || record.username || '-'}</div>
          <Text type='tertiary' size='small'>ID: {record.user_id}</Text>
        </div>
      ),
    },
    {
      title: '申请理由',
      dataIndex: 'reason',
      render: (text) => (
        <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 280 }}>
          {text}
        </Text>
      ),
    },
    {
      title: 'LinuxDo',
      dataIndex: 'linuxdo_profile',
      width: 120,
      render: (text, record) => {
        if (!text) return <Tag color='grey' size='small'>未填写</Tag>;
        return (
          <a href={text} target='_blank' rel='noopener noreferrer' style={{ fontSize: 12 }}>
            查看链接
          </a>
        );
      },
    },
    {
      title: '通知',
      dataIndex: 'notify_status',
      width: 110,
      render: (val) => val === 'available'
        ? <Tag color='green' size='small'>可私信通知</Tag>
        : <Tag color='grey' size='small'>不做通知</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const s = STATUS_MAP[val] || { text: val, color: 'default' };
        return <Tag color={s.color} size='small'>{s.text}</Tag>;
      },
    },
    {
      title: '申请时间',
      dataIndex: 'submitted_at',
      width: 150,
      render: formatTime,
    },
    {
      title: '轮次',
      dataIndex: 'application_round',
      width: 60,
      render: (val) => `${val}/3`,
    },
    {
      title: 'AI建议',
      dataIndex: 'ai_decision',
      width: 100,
      render: (val, record) => {
        if (!val) return <Tag color='default' size='small'>未审</Tag>;
        const d = AI_DECISION_MAP[val] || { text: val, color: 'default' };
        return (
          <span>
            <Tag color={d.color} size='small'>{d.text}</Tag>
            {record.ai_confidence > 0 && (
              <Text type='tertiary' size='small' style={{ marginLeft: 4 }}>{(record.ai_confidence * 100).toFixed(0)}%</Text>
            )}
          </span>
        );
      },
    },
    {
      title: '操作',
      width: 80,
      render: (_, record) => (
        <Button size='small' theme='light' onClick={() => loadDetail(record.id)}>
          详情
        </Button>
      ),
    },
  ];

  return (
    <div style={{ padding: '20px 24px' }}>
      <Title heading={4} style={{ marginBottom: 16 }}>农场内测资格申请管理</Title>

      <Card style={{ marginBottom: 16 }}>
        <Space>
          <Text strong>状态筛选:</Text>
          <Select
            value={statusFilter}
            onChange={(val) => { setStatusFilter(val); setPage(1); }}
            style={{ width: 140 }}
            optionList={[
              { label: '全部', value: '' },
              { label: '待审核', value: 'pending' },
              { label: '已通过', value: 'approved' },
              { label: '已拒绝', value: 'rejected' },
            ]}
          />
          <Button onClick={loadList} loading={loading}>刷新</Button>
        </Space>
      </Card>

      <Card>
        <Table
          columns={columns}
          dataSource={list}
          rowKey='id'
          loading={loading}
          pagination={false}
          size='small'
          empty={<Empty description='暂无申请记录' />}
        />
        {total > pageSize && (
          <div style={{ textAlign: 'right', marginTop: 16 }}>
            <Pagination
              total={total}
              pageSize={pageSize}
              currentPage={page}
              onChange={(p) => setPage(p)}
            />
          </div>
        )}
      </Card>

      {/* 详情弹窗 */}
      <Modal
        title='申请详情'
        visible={detailVisible}
        onCancel={() => { if (!actionLoading) setDetailVisible(false); }}
        footer={null}
        width={600}
        closable={!actionLoading}
      >
        {detailLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>
        ) : detailData ? (
          <div>
            <Descriptions
              data={[
                { key: '用户ID', value: detailData.user_id },
                { key: '用户名', value: detailData.username || '-' },
                { key: '昵称', value: detailData.display_name || '-' },
                { key: '邮箱', value: detailData.email || '-' },
                { key: '已有资格', value: detailData.has_existing_access ? '是 ✅' : '否' },
                { key: '申请轮次', value: `${detailData.application_round}/3` },
                { key: '状态', value: (STATUS_MAP[detailData.status] || {}).text || detailData.status },
                { key: '申请时间', value: formatTime(detailData.submitted_at) },
              ]}
              row
              size='small'
              style={{ marginBottom: 16 }}
            />

            <Card title='申请理由' size='small' style={{ marginBottom: 12 }}>
              <Text style={{ whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>{detailData.reason}</Text>
            </Card>

            <Card title='LinuxDo 论坛链接' size='small' style={{ marginBottom: 12 }}>
              {detailData.linuxdo_profile ? (
                <div>
                  <a href={detailData.linuxdo_profile} target='_blank' rel='noopener noreferrer'>
                    {detailData.linuxdo_profile}
                  </a>
                  <br />
                  <Tag color='green' size='small' style={{ marginTop: 6 }}>{detailData.notify_message}</Tag>
                </div>
              ) : (
                <Tag color='grey' size='small'>{detailData.notify_message}</Tag>
              )}
            </Card>

            {detailData.ai_decision && (
              <Card title='AI 审核结果' size='small' style={{ marginBottom: 12 }}>
                <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 8 }}>
                  <Tag color={(AI_DECISION_MAP[detailData.ai_decision] || {}).color || 'default'} size='small'>
                    {(AI_DECISION_MAP[detailData.ai_decision] || {}).text || detailData.ai_decision}
                  </Tag>
                  {detailData.ai_confidence > 0 && (
                    <Text size='small'>置信度: {(detailData.ai_confidence * 100).toFixed(0)}%</Text>
                  )}
                </div>
                {detailData.ai_summary && (
                  <Text size='small' style={{ display: 'block', marginBottom: 4 }}>
                    <strong>摘要:</strong> {detailData.ai_summary}
                  </Text>
                )}
                {detailData.ai_logs && detailData.ai_logs.length > 0 && detailData.ai_logs.map((log) => (
                  <div key={log.id} style={{ fontSize: 12, marginTop: 4, padding: '6px 8px', background: 'var(--semi-color-fill-0)', borderRadius: 4 }}>
                    <Space>
                      <Text type='tertiary' size='small'>模型: {log.model_name}</Text>
                      <Text type='tertiary' size='small'>评分: {log.ai_score || '-'}</Text>
                      <Text type='tertiary' size='small'>Prompt v{log.prompt_version}</Text>
                      <Text type='tertiary' size='small'>{formatTime(log.created_at)}</Text>
                    </Space>
                    {log.ai_reasons && (
                      <div style={{ marginTop: 4 }}>
                        <Text type='tertiary' size='small'>理由: {log.ai_reasons}</Text>
                      </div>
                    )}
                    {log.error_message && (
                      <div style={{ marginTop: 4 }}>
                        <Text type='danger' size='small'>错误: {log.error_message}</Text>
                      </div>
                    )}
                  </div>
                ))}
              </Card>
            )}

            {detailData.reviewed_at > 0 && (
              <Card title='审核信息' size='small' style={{ marginBottom: 12 }}>
                <Descriptions
                  data={[
                    { key: '审核人ID', value: detailData.reviewed_by === 0 ? 'AI 自动' : (detailData.reviewed_by || '-') },
                    { key: '审核时间', value: formatTime(detailData.reviewed_at) },
                    { key: '审核备注', value: detailData.review_note || '-' },
                  ]}
                  row
                  size='small'
                />
              </Card>
            )}

            {detailData.history && detailData.history.length > 1 && (
              <Card title='历史申请记录' size='small' style={{ marginBottom: 12 }}>
                {detailData.history.map((h) => (
                  <div key={h.id} style={{
                    padding: '8px 0', borderBottom: '1px solid var(--semi-color-border)',
                    fontSize: 13,
                  }}>
                    <Tag color={(STATUS_MAP[h.status] || {}).color || 'default'} size='small'>
                      {(STATUS_MAP[h.status] || {}).text || h.status}
                    </Tag>
                    <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>
                      第{h.round}轮 · {formatTime(h.submitted_at)}
                    </Text>
                    {h.review_note && (
                      <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>
                        备注: {h.review_note}
                      </Text>
                    )}
                  </div>
                ))}
              </Card>
            )}

            {detailData.status === 'pending' && (
              <Card
                title={
                  <span>
                    审核操作
                    {detailData.linuxdo_profile && (
                      <Text type='warning' size='small' style={{ marginLeft: 8 }}>
                        💡 通过后请管理员手动私信通知用户
                      </Text>
                    )}
                  </span>
                }
                size='small'
                style={{ marginTop: 12 }}
              >
                <div style={{ marginBottom: 12 }}>
                  <Text strong size='small' style={{ display: 'block', marginBottom: 4 }}>审核备注（选填）</Text>
                  <Input
                    value={reviewNote}
                    onChange={setReviewNote}
                    placeholder='可填写审核备注，拒绝时建议填写原因'
                  />
                </div>
                <Space>
                  <Button type='primary' theme='solid' loading={actionLoading} onClick={handleApprove}>
                    ✅ 审核通过
                  </Button>
                  <Button type='danger' theme='solid' loading={actionLoading} onClick={handleReject}>
                    ❌ 审核拒绝
                  </Button>
                </Space>
              </Card>
            )}
          </div>
        ) : null}
      </Modal>
    </div>
  );
};

export default BetaApplicationsAdmin;
