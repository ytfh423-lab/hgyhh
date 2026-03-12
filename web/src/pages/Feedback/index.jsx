import React, { useCallback, useEffect, useState } from 'react';
import {
  Button, Card, Input, Select, Spin, Switch, Table, TabPane, Tabs, Tag,
  TextArea, Typography, Notification,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';

const { Text, Title, Paragraph } = Typography;

const CATEGORY_OPTIONS = [
  { value: 'bug', label: 'Bug 反馈', emoji: '🐛' },
  { value: 'suggestion', label: '功能建议', emoji: '💡' },
  { value: 'feedback', label: '体验反馈', emoji: '💬' },
  { value: 'other', label: '其他', emoji: '📝' },
];

const CATEGORY_MAP = { bug: '🐛 Bug', suggestion: '💡 建议', feedback: '💬 反馈', other: '📝 其他' };
const CATEGORY_COLOR = { bug: 'red', suggestion: 'blue', feedback: 'green', other: 'grey' };

const STATUS_MAP = {
  pending: '待处理', viewed: '已查看', processing: '处理中', resolved: '已解决', rejected: '已拒绝',
};
const STATUS_COLOR = {
  pending: 'orange', viewed: 'blue', processing: 'light-blue', resolved: 'green', rejected: 'red',
};

const FeedbackPage = () => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('submit');

  // ========== 发布留言 ==========
  const [form, setForm] = useState({ title: '', content: '', category: 'feedback', contact_info: '', is_public: false });
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    if (!form.title.trim()) return showError(t('标题不能为空'));
    if (!form.content.trim() || form.content.trim().length < 10) return showError(t('内容至少10个字符'));
    setSubmitting(true);
    try {
      const { data: res } = await API.post('/api/feedback/', form);
      if (res.success) {
        showSuccess(t('提交成功'));
        setForm({ title: '', content: '', category: 'feedback', contact_info: '', is_public: false });
        setActiveTab('my');
      } else {
        showError(res.message);
      }
    } catch { showError(t('提交失败')); }
    finally { setSubmitting(false); }
  };

  // ========== 我的留言 ==========
  const [myPosts, setMyPosts] = useState([]);
  const [myTotal, setMyTotal] = useState(0);
  const [myPage, setMyPage] = useState(1);
  const [myLoading, setMyLoading] = useState(false);

  const loadMyPosts = useCallback(async () => {
    setMyLoading(true);
    try {
      const { data: res } = await API.get(`/api/feedback/my?page=${myPage}&page_size=10`);
      if (res.success) { setMyPosts(res.data || []); setMyTotal(res.total || 0); }
    } catch { showError(t('加载失败')); }
    finally { setMyLoading(false); }
  }, [myPage, t]);

  useEffect(() => { if (activeTab === 'my') loadMyPosts(); }, [activeTab, loadMyPosts]);

  // ========== 公开留言板 ==========
  const [pubPosts, setPubPosts] = useState([]);
  const [pubTotal, setPubTotal] = useState(0);
  const [pubPage, setPubPage] = useState(1);
  const [pubCategory, setPubCategory] = useState('');
  const [pubLoading, setPubLoading] = useState(false);

  const loadPubPosts = useCallback(async () => {
    setPubLoading(true);
    try {
      let url = `/api/feedback/public?page=${pubPage}&page_size=10`;
      if (pubCategory) url += `&category=${pubCategory}`;
      const { data: res } = await API.get(url);
      if (res.success) { setPubPosts(res.data || []); setPubTotal(res.total || 0); }
    } catch { showError(t('加载失败')); }
    finally { setPubLoading(false); }
  }, [pubPage, pubCategory, t]);

  useEffect(() => { if (activeTab === 'public') loadPubPosts(); }, [activeTab, loadPubPosts]);

  const myColumns = [
    { title: t('标题'), dataIndex: 'title', width: 200,
      render: (text) => <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 200 }}>{text}</Text> },
    { title: t('类型'), dataIndex: 'category', width: 100,
      render: (v) => <Tag color={CATEGORY_COLOR[v]}>{CATEGORY_MAP[v] || v}</Tag> },
    { title: t('状态'), dataIndex: 'status', width: 100,
      render: (v) => <Tag color={STATUS_COLOR[v]}>{STATUS_MAP[v] || v}</Tag> },
    { title: t('提交时间'), dataIndex: 'created_at', width: 160,
      render: (ts) => ts ? new Date(ts * 1000).toLocaleString() : '-' },
    { title: t('管理员回复'), dataIndex: 'admin_reply', width: 200,
      render: (text) => text ? (
        <Paragraph ellipsis={{ rows: 2, expandable: true }} style={{ marginBottom: 0 }}>{text}</Paragraph>
      ) : <Text type='tertiary'>-</Text> },
  ];

  const pubColumns = [
    { title: t('标题'), dataIndex: 'title', width: 220,
      render: (text) => <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 220 }}>{text}</Text> },
    { title: t('类型'), dataIndex: 'category', width: 100,
      render: (v) => <Tag color={CATEGORY_COLOR[v]}>{CATEGORY_MAP[v] || v}</Tag> },
    { title: t('状态'), dataIndex: 'status', width: 100,
      render: (v) => <Tag color={STATUS_COLOR[v]}>{STATUS_MAP[v] || v}</Tag> },
    { title: t('时间'), dataIndex: 'created_at', width: 160,
      render: (ts) => ts ? new Date(ts * 1000).toLocaleString() : '-' },
    { title: t('管理员回复'), dataIndex: 'admin_reply',
      render: (text) => text ? (
        <Paragraph ellipsis={{ rows: 2, expandable: true }} style={{ marginBottom: 0 }}>{text}</Paragraph>
      ) : <Text type='tertiary'>-</Text> },
  ];

  return (
    <div className='mt-[60px] px-2'>
      <Card className='!rounded-2xl' style={{
        boxShadow: '0 1px 3px rgba(0,0,0,0.04), 0 4px 16px rgba(0,0,0,0.02)',
        border: '1px solid var(--semi-color-border)',
      }}>
        <Title heading={5} style={{ marginBottom: 16 }}>📮 {t('留言板 / 反馈中心')}</Title>
        <Tabs activeKey={activeTab} onChange={setActiveTab}>

          {/* Tab 1: 发布留言 */}
          <TabPane tab={t('✏️ 发布留言')} itemKey='submit'>
            <div style={{ maxWidth: 600, display: 'flex', flexDirection: 'column', gap: 14, padding: '16px 0' }}>
              <div>
                <Text strong>{t('类型')}</Text>
                <Select value={form.category} onChange={(v) => setForm({ ...form, category: v })}
                  style={{ width: '100%', marginTop: 4 }}
                  optionList={CATEGORY_OPTIONS.map(o => ({ value: o.value, label: `${o.emoji} ${t(o.label)}` }))} />
              </div>
              <div>
                <Text strong>{t('标题')} <Text type='danger'>*</Text></Text>
                <Input value={form.title} onChange={(v) => setForm({ ...form, title: v })}
                  placeholder={t('请输入标题')} maxLength={128} showClear style={{ marginTop: 4 }} />
              </div>
              <div>
                <Text strong>{t('内容')} <Text type='danger'>*</Text></Text>
                <TextArea value={form.content} onChange={(v) => setForm({ ...form, content: v })}
                  placeholder={t('请详细描述你的反馈或建议（至少10个字符）')}
                  autosize={{ minRows: 4, maxRows: 12 }} style={{ marginTop: 4 }} />
                <Text type='tertiary' size='small'>{form.content.length} {t('字符')}</Text>
              </div>
              <div>
                <Text strong>{t('联系方式')} <Text type='tertiary'>({t('可选')})</Text></Text>
                <Input value={form.contact_info} onChange={(v) => setForm({ ...form, contact_info: v })}
                  placeholder={t('邮箱、Telegram 等')} style={{ marginTop: 4 }} />
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <Switch checked={form.is_public} onChange={(v) => setForm({ ...form, is_public: v })} />
                <Text>{t('允许公开展示')}</Text>
              </div>
              <Button theme='solid' type='primary' onClick={handleSubmit} loading={submitting}
                style={{ width: 'fit-content' }}>
                {t('提交留言')}
              </Button>
            </div>
          </TabPane>

          {/* Tab 2: 我的留言 */}
          <TabPane tab={t('📋 我的留言')} itemKey='my'>
            <Table columns={myColumns} dataSource={myPosts} loading={myLoading} rowKey='id'
              pagination={{ currentPage: myPage, pageSize: 10, total: myTotal, onPageChange: setMyPage }}
              expandedRowRender={(record) => (
                <div style={{ padding: '8px 16px' }}>
                  <Text strong>{t('内容')}:</Text>
                  <Paragraph style={{ marginTop: 4, whiteSpace: 'pre-wrap' }}>{record.content}</Paragraph>
                  {record.admin_note && (
                    <><Text strong>{t('管理员备注')}:</Text><Paragraph>{record.admin_note}</Paragraph></>
                  )}
                </div>
              )} />
          </TabPane>

          {/* Tab 3: 公开留言板 */}
          <TabPane tab={t('🌐 公开留言板')} itemKey='public'>
            <div style={{ marginBottom: 12 }}>
              <Select value={pubCategory} onChange={(v) => { setPubCategory(v); setPubPage(1); }}
                style={{ width: 160 }} placeholder={t('全部类型')} showClear
                optionList={CATEGORY_OPTIONS.map(o => ({ value: o.value, label: `${o.emoji} ${t(o.label)}` }))} />
            </div>
            <Table columns={pubColumns} dataSource={pubPosts} loading={pubLoading} rowKey='id'
              pagination={{ currentPage: pubPage, pageSize: 10, total: pubTotal, onPageChange: setPubPage }}
              expandedRowRender={(record) => (
                <div style={{ padding: '8px 16px' }}>
                  <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{record.content}</Paragraph>
                  {record.admin_reply && (
                    <div style={{ marginTop: 8, padding: 8, background: 'var(--semi-color-fill-0)', borderRadius: 6 }}>
                      <Text strong>💬 {t('管理员回复')}:</Text>
                      <Paragraph style={{ marginTop: 4 }}>{record.admin_reply}</Paragraph>
                    </div>
                  )}
                </div>
              )} />
          </TabPane>
        </Tabs>
      </Card>
    </div>
  );
};

export default FeedbackPage;
