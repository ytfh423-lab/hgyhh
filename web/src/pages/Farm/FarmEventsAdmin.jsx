import React, { useCallback, useEffect, useState } from 'react';
import {
  Button, Card, Table, Tabs, TabPane, Typography, Spin,
  Input, Tag, Modal,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './components/utils';

const { Title, Text } = Typography;

// 事件后端日志面板（A-4）— admin 只读
// 两个 Tab：天气事件 / 突发事件
// 底部：手动触发按钮（指定 tg_id）

const fmt = (ts) => ts ? new Date(ts * 1000).toLocaleString() : '—';

const severityColor = (s) => {
  if (s >= 3) return 'red';
  if (s === 2) return 'orange';
  return 'green';
};

const FarmEventsAdmin = () => {
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState({ weather: [], random: [] });
  const [triggerTgId, setTriggerTgId] = useState('');
  const [triggering, setTriggering] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/tgbot/farm/events?limit=100');
      if (res.success) setData(res.data);
      else showError(res.message || '加载失败');
    } catch (e) { showError('网络错误'); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const triggerForUser = async () => {
    if (!triggerTgId.trim()) return;
    setTriggering(true);
    try {
      const { data: res } = await API.post('/api/tgbot/farm/events/trigger-random', {
        tg_id: triggerTgId.trim(),
      });
      if (res.success) {
        showSuccess(res.message);
        setTriggerTgId('');
        load();
      } else {
        showError(res.message);
      }
    } catch (e) { showError('网络错误'); }
    finally { setTriggering(false); }
  };

  const weatherColumns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    {
      title: '事件', dataIndex: 'name', width: 140,
      render: (_, rec) => (
        <span><span style={{ marginRight: 6 }}>{rec.emoji}</span>{rec.name}</span>
      ),
    },
    {
      title: '等级', dataIndex: 'severity', width: 90,
      render: (v) => <Tag color={severityColor(v)}>Lv.{v}</Tag>,
    },
    { title: '描述', dataIndex: 'narrative', ellipsis: true },
    { title: '开始', dataIndex: 'started_at', width: 160, render: fmt },
    { title: '结束', dataIndex: 'ends_at', width: 160, render: fmt },
    {
      title: '状态', dataIndex: 'ended', width: 100,
      render: (v) => v ? <Tag color='grey'>已结束</Tag> : <Tag color='blue'>进行中</Tag>,
    },
  ];

  const randomColumns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '玩家', dataIndex: 'tg_id', width: 140 },
    {
      title: '事件', dataIndex: 'title', width: 160,
      render: (_, rec) => (
        <span><span style={{ marginRight: 6 }}>{rec.emoji}</span>{rec.title}</span>
      ),
    },
    {
      title: '选项', dataIndex: 'chosen_idx', width: 100,
      render: (v) => v >= 0
        ? <Tag color='green'>{String.fromCharCode(65 + v)}</Tag>
        : <Tag color='grey'>未选</Tag>,
    },
    { title: '结果文案', dataIndex: 'outcome', ellipsis: true },
    { title: '触发', dataIndex: 'started_at', width: 160, render: fmt },
    { title: '结算', dataIndex: 'resolved_at', width: 160, render: fmt },
  ];

  return (
    <div style={{ padding: 20 }}>
      <Title heading={3}>🌦️ 事件后端日志面板</Title>
      <Text type='tertiary'>A-2 天气事件 · A-3 突发事件（只读 + 手动触发）</Text>

      <Card style={{ marginTop: 16 }}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 12 }}>
          <Input
            placeholder='Telegram ID'
            value={triggerTgId}
            onChange={setTriggerTgId}
            style={{ width: 200 }}
          />
          <Button theme='solid' onClick={triggerForUser} loading={triggering}
                  disabled={!triggerTgId.trim()}>
            给此玩家触发一条突发事件
          </Button>
          <Button icon={<span>🔄</span>} onClick={load}>刷新</Button>
          <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>
            管理员强制触发会忽略玩家 12 小时节流，但玩家若已有未结算事件仍会失败
          </Text>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>
        ) : (
          <Tabs type='line'>
            <TabPane tab={`天气事件 (${data.weather.length})`} itemKey='weather'>
              <Table
                dataSource={data.weather}
                columns={weatherColumns}
                pagination={{ pageSize: 20 }}
                rowKey='id'
                size='small'
              />
            </TabPane>
            <TabPane tab={`突发事件 (${data.random.length})`} itemKey='random'>
              <Table
                dataSource={data.random}
                columns={randomColumns}
                pagination={{ pageSize: 20 }}
                rowKey='id'
                size='small'
              />
            </TabPane>
          </Tabs>
        )}
      </Card>
    </div>
  );
};

export default FarmEventsAdmin;
