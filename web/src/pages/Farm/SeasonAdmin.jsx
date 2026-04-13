import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Modal, Space, Spin, Table, Tag, TextArea, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './components/utils';

const { Title, Text } = Typography;

const templates = {
  season: { code: '', weeks_per_season: 4, rush_days: 7, rest_days: 1, status: 0, start_at: 0, end_at: 0, points_multiplier: 100 },
  tier: { tier_key: '', tier_name: '', tier_level: 0, min_points: 0, initial_balance: 0, gift_items: '[]', emoji: '🏅', color: '#999999' },
  points: { action: '', action_name: '', points: 1, daily_cap: 0, enabled: true },
  anti: { rule_key: '', rule_name: '', enabled: true, threshold: 0, window_secs: 600, action: 1, ban_duration: 3600 },
};

const urls = {
  season: '/api/farm/season/admin/seasons',
  tier: '/api/farm/season/admin/tiers',
  points: '/api/farm/season/admin/points-rules',
  anti: '/api/farm/season/admin/anti-cheat/rules',
  logs: '/api/farm/season/admin/anti-cheat/logs',
};

const statusMap = { 0: ['未开始', 'grey'], 1: ['冲榜期', 'orange'], 2: ['休赛期', 'blue'], 3: ['已结束', 'green'] };
const actionMap = { 1: '警告', 2: '临时封禁', 3: '永久封禁' };
const fmtTime = (ts) => ts ? new Date(ts * 1000).toLocaleString('zh-CN') : '-';

export default function SeasonAdmin() {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [seasons, setSeasons] = useState([]);
  const [tiers, setTiers] = useState([]);
  const [pointsRules, setPointsRules] = useState([]);
  const [antiRules, setAntiRules] = useState([]);
  const [logs, setLogs] = useState([]);
  const [editor, setEditor] = useState({ open: false, kind: 'season', text: '' });

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const [a, b, c, d, e] = await Promise.all([
        API.get(urls.season), API.get(urls.tier), API.get(urls.points), API.get(urls.anti), API.get(urls.logs),
      ]);
      if (a.data?.success) setSeasons(a.data.data || []);
      if (b.data?.success) setTiers(b.data.data || []);
      if (c.data?.success) setPointsRules(c.data.data || []);
      if (d.data?.success) setAntiRules(d.data.data || []);
      if (e.data?.success) setLogs(e.data.data || []);
    } catch (e) {
      showError(e?.message || '加载赛季后台数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadAll(); }, [loadAll]);

  const openEditor = (kind, row) => setEditor({ open: true, kind, text: JSON.stringify(row || templates[kind], null, 2) });
  const closeEditor = () => setEditor({ open: false, kind: 'season', text: '' });

  const saveEditor = async () => {
    setSaving(true);
    try {
      const payload = JSON.parse(editor.text);
      const method = (editor.kind === 'points' || editor.kind === 'anti') ? 'post' : (payload.id ? 'put' : 'post');
      const { data: res } = await API[method](urls[editor.kind], payload);
      if (!res.success) throw new Error(res.message || '保存失败');
      showSuccess('保存成功');
      closeEditor();
      loadAll();
    } catch (e) {
      showError(e?.message || '保存失败，请检查 JSON 格式');
    } finally {
      setSaving(false);
    }
  };

  const removeItem = async (kind, id) => {
    try {
      const { data: res } = await API.delete(urls[kind], { data: { id } });
      if (!res.success) throw new Error(res.message || '删除失败');
      showSuccess('删除成功');
      loadAll();
    } catch (e) {
      showError(e?.message || '删除失败');
    }
  };

  const seasonAction = async (mode, id) => {
    try {
      const { data: res } = await API.post(`/api/farm/season/admin/seasons/${mode}`, { id });
      if (!res.success) throw new Error(res.message || '操作失败');
      showSuccess(mode === 'start' ? '赛季已启动' : '赛季已结束');
      loadAll();
    } catch (e) {
      showError(e?.message || '操作失败');
    }
  };

  const seasonColumns = useMemo(() => [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '代号', dataIndex: 'code' },
    { title: '倍率', render: (_, r) => `${r.points_multiplier}%` },
    { title: '状态', render: (_, r) => <Tag color={(statusMap[r.status] || [null, 'grey'])[1]}>{(statusMap[r.status] || ['未知'])[0]}</Tag> },
    { title: '开始', render: (_, r) => fmtTime(r.start_at) },
    { title: '结束', render: (_, r) => fmtTime(r.end_at) },
    { title: '操作', render: (_, r) => <Space><Button size='small' theme='light' onClick={() => openEditor('season', r)}>编辑</Button><Button size='small' type='primary' theme='light' onClick={() => seasonAction('start', r.id)}>启动</Button><Button size='small' type='warning' theme='light' onClick={() => seasonAction('end', r.id)}>结束</Button><Button size='small' type='danger' theme='light' onClick={() => removeItem('season', r.id)}>删除</Button></Space> },
  ], []);

  const tierColumns = useMemo(() => [
    { title: '段位', render: (_, r) => `${r.emoji || '🏅'} ${r.tier_name}` },
    { title: 'Key', dataIndex: 'tier_key' },
    { title: '层级', dataIndex: 'tier_level' },
    { title: '最低积分', dataIndex: 'min_points' },
    { title: '操作', render: (_, r) => <Space><Button size='small' theme='light' onClick={() => openEditor('tier', r)}>编辑</Button><Button size='small' type='danger' theme='light' onClick={() => removeItem('tier', r.id)}>删除</Button></Space> },
  ], []);

  const pointsColumns = useMemo(() => [
    { title: '动作', dataIndex: 'action' },
    { title: '名称', dataIndex: 'action_name' },
    { title: '积分', dataIndex: 'points' },
    { title: '日上限', dataIndex: 'daily_cap' },
    { title: '启用', render: (_, r) => <Tag color={r.enabled ? 'green' : 'grey'}>{r.enabled ? '是' : '否'}</Tag> },
    { title: '操作', render: (_, r) => <Space><Button size='small' theme='light' onClick={() => openEditor('points', r)}>编辑</Button><Button size='small' type='danger' theme='light' onClick={() => removeItem('points', r.id)}>删除</Button></Space> },
  ], []);

  const antiColumns = useMemo(() => [
    { title: '规则', dataIndex: 'rule_name' },
    { title: 'Key', dataIndex: 'rule_key' },
    { title: '阈值', dataIndex: 'threshold' },
    { title: '窗口', render: (_, r) => `${r.window_secs}s` },
    { title: '动作', render: (_, r) => actionMap[r.action] || r.action },
    { title: '操作', render: (_, r) => <Space><Button size='small' theme='light' onClick={() => openEditor('anti', r)}>编辑</Button><Button size='small' type='danger' theme='light' onClick={() => removeItem('anti', r.id)}>删除</Button></Space> },
  ], []);

  const logColumns = useMemo(() => [
    { title: '用户', dataIndex: 'telegram_id' },
    { title: '规则', dataIndex: 'rule_key' },
    { title: '级别', render: (_, r) => actionMap[r.severity] || r.severity },
    { title: '详情', dataIndex: 'detail' },
    { title: '时间', render: (_, r) => fmtTime(r.created_at) },
  ], []);

  return (
    <Spin spinning={loading}>
      <div style={{ padding: 8 }}>
        <Title heading={4}>🏟️ 赛季配置后台</Title>
        <Text type='secondary'>先提供可用的配置与排障入口，后续可以继续细化成更完整的表单编辑器。</Text>
        <Card style={{ marginTop: 16, marginBottom: 16 }} title='赛季列表' extra={<Space><Button theme='light' onClick={loadAll}>刷新</Button><Button type='primary' onClick={() => openEditor('season')}>新增赛季</Button></Space>}><Table rowKey='id' columns={seasonColumns} dataSource={seasons} pagination={false} /></Card>
        <Card style={{ marginBottom: 16 }} title='段位配置' extra={<Button type='primary' onClick={() => openEditor('tier')}>新增段位</Button>}><Table rowKey='id' columns={tierColumns} dataSource={tiers} pagination={false} /></Card>
        <Card style={{ marginBottom: 16 }} title='积分规则' extra={<Button type='primary' onClick={() => openEditor('points')}>新增积分规则</Button>}><Table rowKey='id' columns={pointsColumns} dataSource={pointsRules} pagination={false} /></Card>
        <Card style={{ marginBottom: 16 }} title='防作弊规则' extra={<Button type='primary' onClick={() => openEditor('anti')}>新增防作弊规则</Button>}><Table rowKey='id' columns={antiColumns} dataSource={antiRules} pagination={false} /></Card>
        <Card title='防作弊触发日志'><Table rowKey='id' columns={logColumns} dataSource={logs} pagination={false} /></Card>
      </div>
      <Modal visible={editor.open} title='编辑配置 JSON' onCancel={closeEditor} onOk={saveEditor} confirmLoading={saving} width={720}>
        <TextArea autosize value={editor.text} onChange={(v) => setEditor((prev) => ({ ...prev, text: v }))} />
      </Modal>
    </Spin>
  );
}
