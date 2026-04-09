import React, { useEffect, useState, useCallback } from 'react';
import {
  Button, Card, Input, Switch, Typography, Select, Space, Spin, TextArea, InputNumber, Modal, Tag, Table, Empty, Pagination,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './components/utils';
import { useTranslation } from 'react-i18next';

const { Text, Title } = Typography;

const DEFAULT_PROMPT = `你是农场内测资格审核助手。请根据用户的申请信息，判断是否应该给予内测资格。

## 审核标准
1. **申请理由质量**：理由是否真诚、具体，表达了对农场玩法的兴趣和参与意愿
2. **内容有效性**：理由不能是无意义的重复文字、乱码或敷衍内容
3. **LinuxDo 社区参与**：如果提供了 LinuxDo 论坛链接，可作为加分项
4. **申请历史**：多次被拒绝后重新申请需要更充分的理由

## 评分规则
- 90-100分：理由详细具体，表达了明确的参与意愿和反馈承诺 → approve
- 70-89分：理由基本合理，但不够详细 → manual_review
- 50-69分：理由模糊或过于简短 → manual_review
- 0-49分：明显无意义、乱码、恶意内容 → reject

## 输出要求
请严格按以下 JSON 格式输出，不要输出任何其他内容：
{
  "decision": "approve 或 reject 或 manual_review",
  "confidence": 0.0到1.0的置信度,
  "score": 0到100的评分,
  "summary": "一句话总结审核结论",
  "reasons": ["理由1", "理由2"],
  "risk_flags": ["风险标记，如无则为空数组"],
  "suggested_review_note": "建议的审核备注"
}`;

const formatTime = (ts) => {
  if (!ts) return '-';
  return new Date(ts * 1000).toLocaleString('zh-CN');
};

const DECISION_MAP = {
  approve: { text: '通过', color: 'green' },
  reject: { text: '拒绝', color: 'red' },
  manual_review: { text: '转人工', color: 'orange' },
  error: { text: '错误', color: 'grey' },
};

const ACTION_MAP = {
  auto_approved: { text: '自动通过', color: 'green' },
  auto_rejected: { text: '自动拒绝', color: 'red' },
  manual_review: { text: '转人工', color: 'orange' },
  error_manual_review: { text: '错误转人工', color: 'grey' },
};

const BetaAIConfigAdmin = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  // 配置表单
  const [config, setConfig] = useState({
    enabled: false,
    api_base_url: 'https://codex.hyw.me',
    model_name: 'gpt5.2',
    api_key: '',
    api_key_configured: false,
    system_prompt: DEFAULT_PROMPT,
    auto_approve_confidence: 85,
    auto_reject_confidence: 80,
    allow_auto_apply_result: true,
    log_raw_response: true,
    timeout_ms: 30000,
    json_mode: true,
    daily_quota: 0,
    prompt_version: 1,
  });

  // 测试
  const [testConnLoading, setTestConnLoading] = useState(false);
  const [testPromptLoading, setTestPromptLoading] = useState(false);
  const [testPromptVisible, setTestPromptVisible] = useState(false);
  const [testReason, setTestReason] = useState('我非常喜欢这个农场玩法，希望能参与内测，帮助发现问题和提供改进建议。');
  const [testLinuxdo, setTestLinuxdo] = useState('');
  const [testResult, setTestResult] = useState(null);

  // 日志
  const [logsVisible, setLogsVisible] = useState(false);
  const [logs, setLogs] = useState([]);
  const [logsTotal, setLogsTotal] = useState(0);
  const [logsPage, setLogsPage] = useState(1);
  const [logsLoading, setLogsLoading] = useState(false);
  const [logDetailVisible, setLogDetailVisible] = useState(false);
  const [logDetail, setLogDetail] = useState(null);

  const loadConfig = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/tgbot/farm/beta-ai-config');
      if (res.success) {
        setConfig((prev) => ({ ...prev, ...res.data, api_key: '' }));
      }
    } catch (e) { /* ignore */ }
    setLoading(false);
  }, []);

  useEffect(() => { loadConfig(); }, [loadConfig]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const payload = {
        enabled: config.enabled,
        api_base_url: config.api_base_url,
        model_name: config.model_name,
        api_key: config.api_key,
        system_prompt: config.system_prompt,
        auto_approve_confidence: config.auto_approve_confidence,
        auto_reject_confidence: config.auto_reject_confidence,
        allow_auto_apply_result: config.allow_auto_apply_result,
        log_raw_response: config.log_raw_response,
        timeout_ms: config.timeout_ms,
        json_mode: config.json_mode,
        daily_quota: config.daily_quota,
      };
      const { data: res } = await API.post('/api/tgbot/farm/beta-ai-config', payload);
      if (res.success) {
        showSuccess(res.message);
        if (res.data?.prompt_version) {
          setConfig((prev) => ({ ...prev, prompt_version: res.data.prompt_version, api_key: '', api_key_configured: true }));
        }
      } else {
        showError(res.message);
      }
    } catch (e) {
      showError('保存失败');
    }
    setSaving(false);
  };

  const handleTestConnection = async () => {
    setTestConnLoading(true);
    try {
      const { data: res } = await API.post('/api/tgbot/farm/beta-ai-config/test', {
        api_base_url: config.api_base_url,
        model_name: config.model_name,
        api_key: config.api_key,
      });
      if (res.success) {
        showSuccess(res.message);
      } else {
        showError(res.message);
      }
    } catch (e) {
      showError('测试失败');
    }
    setTestConnLoading(false);
  };

  const handleTestPrompt = async () => {
    setTestPromptLoading(true);
    setTestResult(null);
    try {
      const { data: res } = await API.post('/api/tgbot/farm/beta-ai-config/test-prompt', {
        system_prompt: config.system_prompt,
        test_data: { reason: testReason, linuxdo_profile: testLinuxdo },
        api_base_url: config.api_base_url,
        model_name: config.model_name,
        api_key: config.api_key,
        json_mode: config.json_mode,
      });
      if (res.success) {
        setTestResult(res.data);
      } else {
        showError(res.message);
      }
    } catch (e) {
      showError('测试失败');
    }
    setTestPromptLoading(false);
  };

  const loadLogs = async (page = 1) => {
    setLogsLoading(true);
    try {
      const { data: res } = await API.get(`/api/tgbot/farm/beta-ai-review-logs?page=${page}&page_size=15`);
      if (res.success) {
        setLogs(res.data.list || []);
        setLogsTotal(res.data.total || 0);
        setLogsPage(page);
      }
    } catch (e) { /* ignore */ }
    setLogsLoading(false);
  };

  const loadLogDetail = async (id) => {
    try {
      const { data: res } = await API.get(`/api/tgbot/farm/beta-ai-review-log/detail?id=${id}`);
      if (res.success) {
        setLogDetail(res.data);
        setLogDetailVisible(true);
      }
    } catch (e) { /* ignore */ }
  };

  const updateField = (field, value) => {
    setConfig((prev) => ({ ...prev, [field]: value }));
  };

  if (loading) {
    return (
      <div style={{ padding: 40, textAlign: 'center' }}><Spin size='large' /></div>
    );
  }

  return (
    <div style={{ padding: '20px 24px', maxWidth: 900 }}>
      <Title heading={4} style={{ marginBottom: 16 }}>AI 自动审核配置</Title>

      {/* 基础开关 */}
      <Card title='基础开关' style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
          <Switch checked={config.enabled} onChange={(v) => updateField('enabled', v)} />
          <Text strong>{config.enabled ? '已开启 AI 自动审核' : '已关闭 AI 自动审核'}</Text>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Text>每日审批名额限制：</Text>
          <InputNumber value={config.daily_quota} onChange={(v) => updateField('daily_quota', v || 0)} min={0} style={{ width: 120 }} />
          <Text type='tertiary' size='small'>0 表示不限制</Text>
        </div>
        {config.prompt_version > 0 && (
          <div style={{ marginTop: 12 }}>
            <Tag color='blue' size='small'>提示词版本 v{config.prompt_version}</Tag>
          </div>
        )}
      </Card>

      {/* 接口配置 */}
      <Card title='接口配置' style={{ marginBottom: 16 }}>
        <div style={{ marginBottom: 12 }}>
          <Text strong size='small' style={{ display: 'block', marginBottom: 4 }}>API 地址</Text>
          <Input value={config.api_base_url} onChange={(v) => updateField('api_base_url', v)} placeholder='https://api.openai.com' />
        </div>
        <div style={{ marginBottom: 12 }}>
          <Text strong size='small' style={{ display: 'block', marginBottom: 4 }}>模型名称</Text>
          <Input value={config.model_name} onChange={(v) => updateField('model_name', v)} placeholder='gpt-4o' />
        </div>
        <div style={{ marginBottom: 12 }}>
          <Text strong size='small' style={{ display: 'block', marginBottom: 4 }}>
            API Key
            {config.api_key_configured && !config.api_key && (
              <Tag color='green' size='small' style={{ marginLeft: 8 }}>已配置，重新输入则覆盖</Tag>
            )}
          </Text>
          <Input mode='password' value={config.api_key} onChange={(v) => updateField('api_key', v)} placeholder={config.api_key_configured ? '已配置，留空保持不变' : '输入 API Key'} />
        </div>
        <div style={{ display: 'flex', gap: 12, marginBottom: 12 }}>
          <div style={{ flex: 1 }}>
            <Text strong size='small' style={{ display: 'block', marginBottom: 4 }}>请求超时（毫秒）</Text>
            <InputNumber value={config.timeout_ms} onChange={(v) => updateField('timeout_ms', v || 30000)} min={5000} max={120000} step={1000} style={{ width: '100%' }} />
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 22 }}>
              <Switch checked={config.json_mode} onChange={(v) => updateField('json_mode', v)} size='small' />
              <Text size='small'>JSON 输出模式</Text>
            </div>
          </div>
        </div>
        <Button onClick={handleTestConnection} loading={testConnLoading} theme='light'>测试连接</Button>
      </Card>

      {/* 提示词配置 */}
      <Card title='前置提示词 / System Prompt' style={{ marginBottom: 16 }}>
        <TextArea
          value={config.system_prompt}
          onChange={(v) => updateField('system_prompt', v)}
          autosize={{ minRows: 12, maxRows: 30 }}
          style={{ fontFamily: 'monospace', fontSize: 13 }}
        />
        <div style={{ marginTop: 8, display: 'flex', gap: 8 }}>
          <Button size='small' theme='light' onClick={() => updateField('system_prompt', DEFAULT_PROMPT)}>恢复默认模板</Button>
          <Text type='tertiary' size='small' style={{ lineHeight: '32px' }}>
            提示词将作为 system role 发送给模型，申请数据作为 user role
          </Text>
        </div>
      </Card>

      {/* 策略配置 */}
      <Card title='审核策略' style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 24, marginBottom: 16 }}>
          <div style={{ flex: 1 }}>
            <Text strong size='small' style={{ display: 'block', marginBottom: 4 }}>自动通过阈值（置信度 %）</Text>
            <InputNumber value={config.auto_approve_confidence} onChange={(v) => updateField('auto_approve_confidence', v || 85)} min={0} max={100} style={{ width: '100%' }} />
            <Text type='tertiary' size='small'>AI 判定 approve 且置信度 ≥ 此值时自动通过</Text>
          </div>
          <div style={{ flex: 1 }}>
            <Text strong size='small' style={{ display: 'block', marginBottom: 4 }}>自动拒绝阈值（置信度 %）</Text>
            <InputNumber value={config.auto_reject_confidence} onChange={(v) => updateField('auto_reject_confidence', v || 80)} min={0} max={100} style={{ width: '100%' }} />
            <Text type='tertiary' size='small'>AI 判定 reject 且置信度 ≥ 此值时自动拒绝</Text>
          </div>
        </div>
        <div style={{ display: 'flex', gap: 24 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Switch checked={config.allow_auto_apply_result} onChange={(v) => updateField('allow_auto_apply_result', v)} size='small' />
            <Text size='small'>允许 AI 直接改审核状态（关闭则仅记录建议）</Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Switch checked={config.log_raw_response} onChange={(v) => updateField('log_raw_response', v)} size='small' />
            <Text size='small'>记录 AI 原始返回结果</Text>
          </div>
        </div>
      </Card>

      {/* 调试功能 */}
      <Card title='调试 & 测试' style={{ marginBottom: 16 }}>
        <div style={{ marginBottom: 12 }}>
          <Text strong size='small' style={{ display: 'block', marginBottom: 4 }}>模拟申请理由</Text>
          <TextArea
            value={testReason}
            onChange={setTestReason}
            autosize={{ minRows: 2, maxRows: 4 }}
            placeholder='输入模拟的申请理由'
          />
        </div>
        <div style={{ marginBottom: 12 }}>
          <Text strong size='small' style={{ display: 'block', marginBottom: 4 }}>模拟 LinuxDo 链接（选填）</Text>
          <Input value={testLinuxdo} onChange={setTestLinuxdo} placeholder='https://linux.do/u/test' />
        </div>
        <Space>
          <Button onClick={handleTestPrompt} loading={testPromptLoading} theme='light'>测试提示词</Button>
          <Button onClick={() => { setLogsVisible(true); loadLogs(1); }} theme='light'>查看审核日志</Button>
        </Space>
        {testResult && (
          <Card size='small' style={{ marginTop: 12, background: 'var(--semi-color-fill-0)' }}>
            <Title heading={6}>AI 返回结果</Title>
            {testResult.ai_result && (
              <div style={{ marginBottom: 8 }}>
                <Space>
                  <Tag color={(DECISION_MAP[testResult.ai_result.decision] || {}).color || 'default'}>
                    {(DECISION_MAP[testResult.ai_result.decision] || {}).text || testResult.ai_result.decision}
                  </Tag>
                  <Text size='small'>置信度: {(testResult.ai_result.confidence * 100).toFixed(0)}%</Text>
                  <Text size='small'>评分: {testResult.ai_result.score}</Text>
                </Space>
                <div style={{ marginTop: 8 }}>
                  <Text size='small'><strong>摘要:</strong> {testResult.ai_result.summary}</Text>
                </div>
                {testResult.ai_result.reasons?.length > 0 && (
                  <div style={{ marginTop: 4 }}>
                    <Text size='small'><strong>理由:</strong> {testResult.ai_result.reasons.join('；')}</Text>
                  </div>
                )}
                {testResult.ai_result.risk_flags?.length > 0 && (
                  <div style={{ marginTop: 4 }}>
                    <Text type='danger' size='small'><strong>风险标记:</strong> {testResult.ai_result.risk_flags.join('；')}</Text>
                  </div>
                )}
                {testResult.ai_result.suggested_review_note && (
                  <div style={{ marginTop: 4 }}>
                    <Text type='tertiary' size='small'><strong>建议备注:</strong> {testResult.ai_result.suggested_review_note}</Text>
                  </div>
                )}
              </div>
            )}
            {testResult.raw_response && (
              <details style={{ marginTop: 8 }}>
                <summary style={{ cursor: 'pointer', fontSize: 12, color: 'var(--semi-color-text-2)' }}>原始返回</summary>
                <pre style={{ fontSize: 11, maxHeight: 200, overflow: 'auto', background: 'var(--semi-color-fill-1)', padding: 8, borderRadius: 4, marginTop: 4 }}>
                  {typeof testResult.raw_response === 'string' ? testResult.raw_response : JSON.stringify(testResult.raw_response, null, 2)}
                </pre>
              </details>
            )}
          </Card>
        )}
      </Card>

      {/* 保存按钮 */}
      <div style={{ textAlign: 'right' }}>
        <Button type='primary' theme='solid' loading={saving} onClick={handleSave} size='large'>保存配置</Button>
      </div>

      {/* 审核日志弹窗 */}
      <Modal
        title='AI 审核日志'
        visible={logsVisible}
        onCancel={() => setLogsVisible(false)}
        footer={null}
        width={800}
      >
        <Table
          columns={[
            { title: 'ID', dataIndex: 'id', width: 60 },
            { title: '申请ID', dataIndex: 'application_id', width: 70 },
            { title: '模型', dataIndex: 'model_name', width: 100 },
            {
              title: 'AI 建议', dataIndex: 'ai_decision', width: 80,
              render: (val) => <Tag color={(DECISION_MAP[val] || {}).color || 'default'} size='small'>{(DECISION_MAP[val] || {}).text || val || '-'}</Tag>,
            },
            {
              title: '最终动作', dataIndex: 'final_action', width: 90,
              render: (val) => <Tag color={(ACTION_MAP[val] || {}).color || 'default'} size='small'>{(ACTION_MAP[val] || {}).text || val || '-'}</Tag>,
            },
            {
              title: '置信度', dataIndex: 'ai_confidence', width: 70,
              render: (val) => val ? `${(val * 100).toFixed(0)}%` : '-',
            },
            { title: '时间', dataIndex: 'created_at', width: 140, render: formatTime },
            {
              title: '操作', width: 60,
              render: (_, record) => <Button size='small' theme='borderless' onClick={() => loadLogDetail(record.id)}>详情</Button>,
            },
          ]}
          dataSource={logs}
          rowKey='id'
          loading={logsLoading}
          pagination={false}
          size='small'
          empty={<Empty description='暂无日志' />}
        />
        {logsTotal > 15 && (
          <div style={{ textAlign: 'right', marginTop: 12 }}>
            <Pagination total={logsTotal} pageSize={15} currentPage={logsPage} onChange={(p) => loadLogs(p)} />
          </div>
        )}
      </Modal>

      {/* 日志详情弹窗 */}
      <Modal
        title='AI 审核日志详情'
        visible={logDetailVisible}
        onCancel={() => setLogDetailVisible(false)}
        footer={null}
        width={650}
      >
        {logDetail && (
          <div>
            <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 12 }}>
              <Tag color={(DECISION_MAP[logDetail.ai_decision] || {}).color}>AI建议: {(DECISION_MAP[logDetail.ai_decision] || {}).text || logDetail.ai_decision}</Tag>
              <Tag color={(ACTION_MAP[logDetail.final_action] || {}).color}>最终: {(ACTION_MAP[logDetail.final_action] || {}).text || logDetail.final_action}</Tag>
              <Text size='small'>置信度: {logDetail.ai_confidence ? `${(logDetail.ai_confidence * 100).toFixed(0)}%` : '-'}</Text>
              <Text size='small'>评分: {logDetail.ai_score || '-'}</Text>
              <Text size='small'>模型: {logDetail.model_name}</Text>
              <Text size='small'>Prompt v{logDetail.prompt_version}</Text>
            </div>
            {logDetail.ai_summary && <Card size='small' title='AI 摘要' style={{ marginBottom: 8 }}><Text>{logDetail.ai_summary}</Text></Card>}
            {logDetail.ai_reasons && (
              <Card size='small' title='AI 理由' style={{ marginBottom: 8 }}>
                <Text style={{ whiteSpace: 'pre-wrap' }}>{logDetail.ai_reasons}</Text>
              </Card>
            )}
            {logDetail.error_message && (
              <Card size='small' title='错误信息' style={{ marginBottom: 8 }}>
                <Text type='danger'>{logDetail.error_message}</Text>
              </Card>
            )}
            {logDetail.system_prompt_snapshot && (
              <details style={{ marginBottom: 8 }}>
                <summary style={{ cursor: 'pointer', fontSize: 12 }}>使用的提示词快照</summary>
                <pre style={{ fontSize: 11, maxHeight: 200, overflow: 'auto', background: 'var(--semi-color-fill-1)', padding: 8, borderRadius: 4, marginTop: 4 }}>
                  {logDetail.system_prompt_snapshot}
                </pre>
              </details>
            )}
            {logDetail.request_payload && (
              <details style={{ marginBottom: 8 }}>
                <summary style={{ cursor: 'pointer', fontSize: 12 }}>请求数据</summary>
                <pre style={{ fontSize: 11, maxHeight: 200, overflow: 'auto', background: 'var(--semi-color-fill-1)', padding: 8, borderRadius: 4, marginTop: 4 }}>
                  {logDetail.request_payload}
                </pre>
              </details>
            )}
            {logDetail.ai_raw_response && (
              <details style={{ marginBottom: 8 }}>
                <summary style={{ cursor: 'pointer', fontSize: 12 }}>原始返回 JSON</summary>
                <pre style={{ fontSize: 11, maxHeight: 200, overflow: 'auto', background: 'var(--semi-color-fill-1)', padding: 8, borderRadius: 4, marginTop: 4 }}>
                  {logDetail.ai_raw_response}
                </pre>
              </details>
            )}
            <Text type='tertiary' size='small'>时间: {formatTime(logDetail.created_at)} | 申请ID: {logDetail.application_id} | 用户ID: {logDetail.user_id}</Text>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default BetaAIConfigAdmin;
