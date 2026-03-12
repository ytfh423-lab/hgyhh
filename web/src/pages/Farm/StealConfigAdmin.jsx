import React, { useCallback, useEffect, useState } from 'react';
import {
  Button, Card, Form, InputNumber, Switch, Spin, Toast, Typography,
  Divider, Tag, Table, Modal,
} from '@douyinfe/semi-ui';
import { API } from './components/utils';

const { Text, Title } = Typography;

const DEFAULTS = {
  steal_enabled: true,
  steal_bonus_only_enabled: false,
  long_crop_protection_enabled: true,
  owner_base_keep_ratio: 0.80,
  stealable_ratio: 0.20,
  owner_protection_minutes: 60,
  max_steal_per_plot: 1,
  max_steal_per_user_per_day: 10,
  max_steal_per_farm_per_day: 5,
  steal_cooldown_seconds: 1800,
  max_daily_loss_ratio_per_farm: 0.20,
  steal_success_rate: 100,
  scarecrow_block_rate: 30,
  dog_guard_rate: 50,
  long_crop_hours_threshold: 8,
  super_long_crop_hours_threshold: 12,
  long_crop_owner_keep_ratio: 0.90,
  super_long_crop_bonus_only: true,
  long_crop_protection_extra_min: 60,
  enable_steal_log: true,
  notify_owner_when_stolen: true,
  compensation_ratio: 0,
};

const StealConfigAdmin = () => {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState(DEFAULTS);
  const [logs, setLogs] = useState([]);
  const [logsTotal, setLogsTotal] = useState(0);
  const [logsPage, setLogsPage] = useState(1);
  const [showLogs, setShowLogs] = useState(false);

  const loadConfig = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/tgbot/farm/steal-config');
      if (res.success && res.data) setConfig(res.data);
    } catch (e) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  const loadLogs = useCallback(async (page = 1) => {
    try {
      const { data: res } = await API.get(`/api/tgbot/farm/steal-config/logs?page=${page}&page_size=10`);
      if (res.success) {
        setLogs(res.data || []);
        setLogsTotal(res.total || 0);
      }
    } catch (e) { /* ignore */ }
  }, []);

  useEffect(() => { loadConfig(); }, [loadConfig]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const { data: res } = await API.post('/api/tgbot/farm/steal-config', config);
      if (res.success) {
        Toast.success('保存成功');
        loadConfig();
      } else {
        Toast.error(res.message || '保存失败');
      }
    } catch (e) {
      Toast.error('网络错误');
    } finally { setSaving(false); }
  };

  const handleReset = async () => {
    Modal.confirm({
      title: '恢复默认配置',
      content: '确定要将所有偷菜配置恢复为默认值吗？此操作不可撤销。',
      onOk: async () => {
        try {
          const { data: res } = await API.post('/api/tgbot/farm/steal-config/reset');
          if (res.success) {
            Toast.success('已恢复默认配置');
            loadConfig();
          } else {
            Toast.error(res.message || '重置失败');
          }
        } catch (e) { Toast.error('网络错误'); }
      },
    });
  };

  const handleShowLogs = () => {
    setShowLogs(true);
    loadLogs(1);
    setLogsPage(1);
  };

  const update = (key, val) => setConfig(prev => ({ ...prev, [key]: val }));

  if (loading) return <div style={{ textAlign: 'center', padding: 80 }}><Spin size='large' /></div>;

  return (
    <div style={{ maxWidth: 800, margin: '0 auto', padding: '20px 16px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <Title heading={4}>偷菜机制配置</Title>
        <div style={{ display: 'flex', gap: 8 }}>
          <Button onClick={handleShowLogs} theme='borderless'>修改日志</Button>
          <Button onClick={handleReset} type='danger' theme='light'>恢复默认</Button>
          <Button onClick={handleSave} loading={saving} theme='solid' type='primary'>保存配置</Button>
        </div>
      </div>

      {/* 基础开关 */}
      <Card title='基础开关' style={{ marginBottom: 16 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          <div>
            <Text strong>偷菜功能总开关</Text>
            <div><Switch checked={config.steal_enabled} onChange={v => update('steal_enabled', v)} /></div>
          </div>
          <div>
            <Text strong>仅允许偷bonus</Text>
            <Text type='tertiary' size='small' style={{ display: 'block' }}>开启后只能偷取加成部分</Text>
            <div><Switch checked={config.steal_bonus_only_enabled} onChange={v => update('steal_bonus_only_enabled', v)} /></div>
          </div>
          <div>
            <Text strong>长周期作物保护</Text>
            <div><Switch checked={config.long_crop_protection_enabled} onChange={v => update('long_crop_protection_enabled', v)} /></div>
          </div>
          <div>
            <Text strong>偷菜日志</Text>
            <div><Switch checked={config.enable_steal_log} onChange={v => update('enable_steal_log', v)} /></div>
          </div>
          <div>
            <Text strong>被偷通知主人</Text>
            <div><Switch checked={config.notify_owner_when_stolen} onChange={v => update('notify_owner_when_stolen', v)} /></div>
          </div>
        </div>
      </Card>

      {/* 收益拆分 */}
      <Card title='收益拆分' style={{ marginBottom: 16 }}>
        <Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 12 }}>
          主人保底 + 可偷比例 之和不能超过 100%。保底部分永远归主人，不受偷菜影响。
        </Text>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          <div>
            <Text strong>主人保底比例</Text>
            <InputNumber value={Math.round(config.owner_base_keep_ratio * 100)} min={50} max={100} suffix='%'
              onChange={v => update('owner_base_keep_ratio', v / 100)} style={{ width: '100%' }} />
          </div>
          <div>
            <Text strong>可偷比例</Text>
            <InputNumber value={Math.round(config.stealable_ratio * 100)} min={0} max={50} suffix='%'
              onChange={v => update('stealable_ratio', v / 100)} style={{ width: '100%' }} />
          </div>
        </div>
      </Card>

      {/* 保护期 */}
      <Card title='成熟保护期' style={{ marginBottom: 16 }}>
        <Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 12 }}>
          作物成熟后的一段时间内，只有主人可以收获，其他人无法偷取。
        </Text>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          <div>
            <Text strong>基础保护时间</Text>
            <InputNumber value={config.owner_protection_minutes} min={0} max={1440} suffix='分钟'
              onChange={v => update('owner_protection_minutes', v)} style={{ width: '100%' }} />
          </div>
        </div>
      </Card>

      {/* 次数限制 */}
      <Card title='偷取次数与频率限制' style={{ marginBottom: 16 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          <div>
            <Text strong>每块地最多被偷次数</Text>
            <InputNumber value={config.max_steal_per_plot} min={1} max={10}
              onChange={v => update('max_steal_per_plot', v)} style={{ width: '100%' }} />
          </div>
          <div>
            <Text strong>每人每天偷菜上限</Text>
            <InputNumber value={config.max_steal_per_user_per_day} min={1} max={100}
              onChange={v => update('max_steal_per_user_per_day', v)} style={{ width: '100%' }} />
          </div>
          <div>
            <Text strong>每农场每天被偷上限</Text>
            <InputNumber value={config.max_steal_per_farm_per_day} min={1} max={100}
              onChange={v => update('max_steal_per_farm_per_day', v)} style={{ width: '100%' }} />
          </div>
          <div>
            <Text strong>偷同一人冷却</Text>
            <InputNumber value={config.steal_cooldown_seconds} min={0} max={86400} suffix='秒'
              onChange={v => update('steal_cooldown_seconds', v)} style={{ width: '100%' }} />
          </div>
        </div>
      </Card>

      {/* 损失上限 */}
      <Card title='损失上限' style={{ marginBottom: 16 }}>
        <div>
          <Text strong>单农场每日最大损失比例</Text>
          <InputNumber value={Math.round(config.max_daily_loss_ratio_per_farm * 100)} min={0} max={100} suffix='%'
            onChange={v => update('max_daily_loss_ratio_per_farm', v / 100)} style={{ width: 300 }} />
        </div>
      </Card>

      {/* 概率 */}
      <Card title='概率参数' style={{ marginBottom: 16 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 16 }}>
          <div>
            <Text strong>偷取成功率</Text>
            <InputNumber value={config.steal_success_rate} min={0} max={100} suffix='%'
              onChange={v => update('steal_success_rate', v)} style={{ width: '100%' }} />
          </div>
          <div>
            <Text strong>稻草人拦截率</Text>
            <InputNumber value={config.scarecrow_block_rate} min={0} max={100} suffix='%'
              onChange={v => update('scarecrow_block_rate', v)} style={{ width: '100%' }} />
          </div>
          <div>
            <Text strong>看门狗拦截率</Text>
            <InputNumber value={config.dog_guard_rate} min={0} max={100} suffix='%'
              onChange={v => update('dog_guard_rate', v)} style={{ width: '100%' }} />
          </div>
        </div>
      </Card>

      {/* 长周期作物保护 */}
      <Card title='长周期作物保护' style={{ marginBottom: 16 }}>
        <Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 12 }}>
          种植时间较长的作物自动获得更高保底比例和额外保护时间。
        </Text>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          <div>
            <Text strong>长周期判定阈值</Text>
            <InputNumber value={config.long_crop_hours_threshold} min={1} max={48} suffix='小时'
              onChange={v => update('long_crop_hours_threshold', v)} style={{ width: '100%' }} />
          </div>
          <div>
            <Text strong>超长周期判定阈值</Text>
            <InputNumber value={config.super_long_crop_hours_threshold} min={1} max={72} suffix='小时'
              onChange={v => update('super_long_crop_hours_threshold', v)} style={{ width: '100%' }} />
          </div>
          <div>
            <Text strong>长周期保底比例</Text>
            <InputNumber value={Math.round(config.long_crop_owner_keep_ratio * 100)} min={50} max={100} suffix='%'
              onChange={v => update('long_crop_owner_keep_ratio', v / 100)} style={{ width: '100%' }} />
          </div>
          <div>
            <Text strong>超长周期仅可偷bonus</Text>
            <div><Switch checked={config.super_long_crop_bonus_only} onChange={v => update('super_long_crop_bonus_only', v)} /></div>
          </div>
          <div>
            <Text strong>长周期额外保护时间</Text>
            <InputNumber value={config.long_crop_protection_extra_min} min={0} max={1440} suffix='分钟'
              onChange={v => update('long_crop_protection_extra_min', v)} style={{ width: '100%' }} />
          </div>
        </div>
      </Card>

      {/* 补偿 */}
      <Card title='被偷补偿' style={{ marginBottom: 16 }}>
        <div>
          <Text strong>被偷补偿比例</Text>
          <Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 4 }}>被偷后系统补偿给主人的金额比例（基于偷取金额）</Text>
          <InputNumber value={Math.round(config.compensation_ratio * 100)} min={0} max={50} suffix='%'
            onChange={v => update('compensation_ratio', v / 100)} style={{ width: 300 }} />
        </div>
      </Card>

      <div style={{ textAlign: 'right', marginTop: 20 }}>
        <Button onClick={handleSave} loading={saving} theme='solid' type='primary' size='large'>
          保存配置
        </Button>
      </div>

      {/* 修改日志弹窗 */}
      <Modal title='配置修改日志' visible={showLogs} onCancel={() => setShowLogs(false)} footer={null} width={700}>
        <Table dataSource={logs} pagination={{
          total: logsTotal, pageSize: 10, currentPage: logsPage,
          onPageChange: (p) => { setLogsPage(p); loadLogs(p); },
        }} columns={[
          { title: 'ID', dataIndex: 'id', width: 60 },
          { title: '操作者', dataIndex: 'operator_id', width: 80 },
          { title: '变更内容', dataIndex: 'changed_fields', render: v => (
            <Text size='small' style={{ wordBreak: 'break-all', maxWidth: 400, display: 'block' }}>{v}</Text>
          )},
          { title: '时间', dataIndex: 'created_at', width: 160, render: v => v ? new Date(v * 1000).toLocaleString() : '-' },
        ]} size='small' />
      </Modal>
    </div>
  );
};

export default StealConfigAdmin;
