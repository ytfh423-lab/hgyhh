import React, { useCallback, useEffect, useState } from 'react';
import {
  Button, Card, Input, Modal, Select, Space, Table, Tag, TextArea,
  Typography, Switch,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';

const { Text, Title, Paragraph } = Typography;

const CATEGORY_OPTIONS = [
  { value: '', label: '全部类型' },
  { value: 'bug', label: '🐛 Bug' },
  { value: 'suggestion', label: '💡 建议' },
  { value: 'feedback', label: '💬 反馈' },
  { value: 'other', label: '📝 其他' },
];
const STATUS_OPTIONS = [
  { value: '', label: '全部状态' },
  { value: 'pending', label: '待处理' },
  { value: 'viewed', label: '已查看' },
  { value: 'processing', label: '处理中' },
  { value: 'resolved', label: '已解决' },
  { value: 'rejected', label: '已拒绝' },
];
const CATEGORY_MAP = { bug: '🐛 Bug', suggestion: '💡 建议', feedback: '💬 反馈', other: '📝 其他' };
const CATEGORY_COLOR = { bug: 'red', suggestion: 'blue', feedback: 'green', other: 'grey' };
const STATUS_MAP = { pending: '待处理', viewed: '已查看', processing: '处理中', resolved: '已解决', rejected: '已拒绝' };
const STATUS_COLOR = { pending: 'orange', viewed: 'blue', processing: 'light-blue', resolved: 'green', rejected: 'red' };

const FeedbackAdminPage = () => {
  const { t } = useTranslation();

  const [posts, setPosts] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');
  const [keyword, setKeyword] = useState('');
  const [loading, setLoading] = useState(false);

  // Detail modal
  const [detailVisible, setDetailVisible] = useState(false);
  const [detail, setDetail] = useState(null);
  const [actionLoading, setActionLoading] = useState(false);

  // Action fields
  const [replyText, setReplyText] = useState('');
  const [noteText, setNoteText] = useState('');

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      let url = `/api/feedback/admin/?page=${page}&page_size=${pageSize}`;
      if (statusFilter) url += `&status=${statusFilter}`;
      if (categoryFilter) url += `&category=${categoryFilter}`;
      if (keyword) url += `&keyword=${encodeURIComponent(keyword)}`;
      const { data: res } = await API.get(url);
      if (res.success) { setPosts(res.data || []); setTotal(res.total || 0); }
      else showError(res.message);
    } catch { showError(t('加载失败')); }
    finally { setLoading(false); }
  }, [page, pageSize, statusFilter, categoryFilter, keyword, t]);

  useEffect(() => { loadData(); }, [loadData]);

  const openDetail = async (id) => {
    try {
      const { data: res } = await API.get(`/api/feedback/admin/${id}`);
      if (res.success) {
        setDetail(res.data);
        setReplyText(res.data.admin_reply || '');
        setNoteText(res.data.admin_note || '');
        setDetailVisible(true);
      } else showError(res.message);
    } catch { showError(t('加载失败')); }
  };

  const updateStatus = async (id, status) => {
    setActionLoading(true);
    try {
      const { data: res } = await API.put(`/api/feedback/admin/${id}/status`, { status });
      if (res.success) {
        showSuccess(t('状态已更新'));
        setDetail({ ...detail, status });
        loadData();
      } else showError(res.message);
    } catch { showError(t('操作失败')); }
    finally { setActionLoading(false); }
  };

  const saveReply = async () => {
    if (!detail) return;
    setActionLoading(true);
    try {
      const { data: res } = await API.put(`/api/feedback/admin/${detail.id}/reply`, { reply: replyText });
      if (res.success) { showSuccess(t('回复已保存')); setDetail({ ...detail, admin_reply: replyText }); loadData(); }
      else showError(res.message);
    } catch { showError(t('操作失败')); }
    finally { setActionLoading(false); }
  };

  const saveNote = async () => {
    if (!detail) return;
    setActionLoading(true);
    try {
      const { data: res } = await API.put(`/api/feedback/admin/${detail.id}/note`, { note: noteText });
      if (res.success) { showSuccess(t('备注已保存')); setDetail({ ...detail, admin_note: noteText }); loadData(); }
      else showError(res.message);
    } catch { showError(t('操作失败')); }
    finally { setActionLoading(false); }
  };

  const togglePublic = async (id, isPublic) => {
    try {
      const { data: res } = await API.put(`/api/feedback/admin/${id}/public`, { is_public: isPublic });
      if (res.success) {
        showSuccess(t('已更新'));
        if (detail && detail.id === id) setDetail({ ...detail, is_public: isPublic });
        loadData();
      } else showError(res.message);
    } catch { showError(t('操作失败')); }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: t('用户'), dataIndex: 'username', width: 100,
      render: (v, r) => <Text>{v || `#${r.user_id}`}</Text> },
    { title: t('标题'), dataIndex: 'title', width: 200,
      render: (text, r) => (
        <a onClick={() => openDetail(r.id)} style={{ cursor: 'pointer', color: 'var(--semi-color-link)' }}>
          <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 200 }}>{text}</Text>
        </a>
      ) },
    { title: t('类型'), dataIndex: 'category', width: 90,
      render: (v) => <Tag color={CATEGORY_COLOR[v]} size='small'>{CATEGORY_MAP[v] || v}</Tag> },
    { title: t('状态'), dataIndex: 'status', width: 90,
      render: (v) => <Tag color={STATUS_COLOR[v]} size='small'>{STATUS_MAP[v] || v}</Tag> },
    { title: t('公开'), dataIndex: 'is_public', width: 70,
      render: (v, r) => <Switch size='small' checked={v} onChange={(checked) => togglePublic(r.id, checked)} /> },
    { title: t('提交时间'), dataIndex: 'created_at', width: 160,
      render: (ts) => ts ? new Date(ts * 1000).toLocaleString() : '-' },
    { title: t('操作'), width: 80, fixed: 'right',
      render: (_, r) => (
        <Button size='small' theme='light' onClick={() => openDetail(r.id)}>{t('详情')}</Button>
      ) },
  ];

  return (
    <div className='mt-[60px] px-2'>
      <Card className='!rounded-2xl' style={{
        boxShadow: '0 1px 3px rgba(0,0,0,0.04), 0 4px 16px rgba(0,0,0,0.02)',
        border: '1px solid var(--semi-color-border)',
      }}>
        <div className='flex items-center justify-between mb-4 flex-wrap gap-2'>
          <Title heading={5} style={{ marginBottom: 0 }}>📮 {t('留言板管理')}</Title>
          <Space>
            <Select value={statusFilter} onChange={(v) => { setStatusFilter(v); setPage(1); }}
              style={{ width: 120 }}
              optionList={STATUS_OPTIONS.map(o => ({ ...o, label: t(o.label) }))} />
            <Select value={categoryFilter} onChange={(v) => { setCategoryFilter(v); setPage(1); }}
              style={{ width: 120 }}
              optionList={CATEGORY_OPTIONS.map(o => ({ ...o, label: t(o.label) }))} />
            <Input placeholder={t('搜索标题/内容')} value={keyword} showClear
              onChange={(v) => setKeyword(v)}
              onEnterPress={() => { setPage(1); loadData(); }}
              style={{ width: 180 }} />
          </Space>
        </div>

        <Table columns={columns} dataSource={posts} loading={loading} rowKey='id'
          pagination={{ currentPage: page, pageSize, total, onPageChange: setPage }}
          scroll={{ x: 900 }} />
      </Card>

      {/* Detail Modal */}
      <Modal title={t('留言详情')} visible={detailVisible} onCancel={() => setDetailVisible(false)}
        footer={null} width={640} style={{ maxHeight: '80vh' }}>
        {detail && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {/* Info */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
              <div><Text type='tertiary'>{t('用户')}:</Text> <Text>{detail.username || `#${detail.user_id}`}</Text></div>
              <div><Text type='tertiary'>{t('类型')}:</Text> <Tag color={CATEGORY_COLOR[detail.category]} size='small'>{CATEGORY_MAP[detail.category]}</Tag></div>
              <div><Text type='tertiary'>{t('状态')}:</Text> <Tag color={STATUS_COLOR[detail.status]} size='small'>{STATUS_MAP[detail.status]}</Tag></div>
              <div><Text type='tertiary'>{t('提交时间')}:</Text> <Text>{new Date(detail.created_at * 1000).toLocaleString()}</Text></div>
              {detail.contact_info && <div><Text type='tertiary'>{t('联系方式')}:</Text> <Text>{detail.contact_info}</Text></div>}
              <div><Text type='tertiary'>{t('公开')}:</Text> <Switch size='small' checked={detail.is_public} onChange={(v) => togglePublic(detail.id, v)} /></div>
            </div>

            {/* Title + Content */}
            <div style={{ padding: 12, background: 'var(--semi-color-fill-0)', borderRadius: 8 }}>
              <Text strong style={{ fontSize: 15 }}>{detail.title}</Text>
              <Paragraph style={{ marginTop: 8, whiteSpace: 'pre-wrap' }}>{detail.content}</Paragraph>
            </div>

            {/* Status actions */}
            <div>
              <Text strong>{t('标记状态')}</Text>
              <div style={{ display: 'flex', gap: 6, marginTop: 4, flexWrap: 'wrap' }}>
                {['viewed', 'processing', 'resolved', 'rejected'].map(s => (
                  <Button key={s} size='small' disabled={detail.status === s || actionLoading}
                    type={s === 'rejected' ? 'danger' : s === 'resolved' ? 'primary' : 'tertiary'}
                    theme={detail.status === s ? 'solid' : 'light'}
                    onClick={() => updateStatus(detail.id, s)}>
                    {STATUS_MAP[s]}
                  </Button>
                ))}
              </div>
            </div>

            {/* Reply */}
            <div>
              <Text strong>{t('管理员回复')}</Text>
              <TextArea value={replyText} onChange={setReplyText}
                autosize={{ minRows: 2, maxRows: 6 }} style={{ marginTop: 4 }} />
              <Button size='small' theme='solid' onClick={saveReply} loading={actionLoading}
                style={{ marginTop: 4 }}>{t('保存回复')}</Button>
            </div>

            {/* Note */}
            <div>
              <Text strong>{t('处理备注')} <Text type='tertiary'>({t('仅管理员可见')})</Text></Text>
              <TextArea value={noteText} onChange={setNoteText}
                autosize={{ minRows: 2, maxRows: 4 }} style={{ marginTop: 4 }} />
              <Button size='small' theme='light' onClick={saveNote} loading={actionLoading}
                style={{ marginTop: 4 }}>{t('保存备注')}</Button>
            </div>

            {detail.resolved_at > 0 && (
              <Text type='tertiary' size='small'>
                {t('处理时间')}: {new Date(detail.resolved_at * 1000).toLocaleString()} | {t('处理人')}: #{detail.resolved_by}
              </Text>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default FeedbackAdminPage;
