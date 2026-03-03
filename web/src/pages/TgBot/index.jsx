import React, { useEffect, useState, useCallback } from 'react';
import {
  Banner,
  Button,
  Card,
  Descriptions,
  Form,
  Modal,
  Space,
  Spin,
  Table,
  Tag,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';

const PURPOSE_OPTIONS = [
  { value: 1, label: '余额兑换码' },
  { value: 2, label: '注册邀请码' },
];

const STATUS_OPTIONS = [
  { value: 1, label: '启用' },
  { value: 2, label: '禁用' },
];

const TgBotPage = () => {
  const { t } = useTranslation();

  // ===== Bot 设置 =====
  const [botToken, setBotToken] = useState('');
  const [botName, setBotName] = useState('');
  const [settingsLoading, setSettingsLoading] = useState(false);
  const [savingSettings, setSavingSettings] = useState(false);
  const [webhookInfo, setWebhookInfo] = useState(null);
  const [settingWebhook, setSettingWebhook] = useState(false);
  const [registeringCommands, setRegisteringCommands] = useState(false);

  // ===== 分类管理 =====
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingCategory, setEditingCategory] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  // ===== 库存管理 =====
  const [inventoryModalVisible, setInventoryModalVisible] = useState(false);
  const [inventoryCategory, setInventoryCategory] = useState(null);
  const [inventoryItems, setInventoryItems] = useState([]);
  const [inventoryLoading, setInventoryLoading] = useState(false);
  const [addCodesText, setAddCodesText] = useState('');
  const [addingCodes, setAddingCodes] = useState(false);

  // 加载系统设置
  const loadSettings = useCallback(async () => {
    setSettingsLoading(true);
    try {
      const res = await API.get('/api/option/');
      if (res.data.success) {
        const data = res.data.data || [];
        data.forEach((item) => {
          if (item.key === 'TelegramBotToken') setBotToken(item.value || '');
          if (item.key === 'TelegramBotName') setBotName(item.value || '');
        });
      }
    } catch (err) {
      showError(t('加载设置失败'));
    } finally {
      setSettingsLoading(false);
    }
  }, [t]);

  // 保存 Bot 设置
  const saveSettings = async () => {
    setSavingSettings(true);
    try {
      const options = [
        { key: 'TelegramBotToken', value: botToken },
        { key: 'TelegramBotName', value: botName },
      ];
      for (const opt of options) {
        const res = await API.put('/api/option/', opt);
        if (!res.data.success) {
          showError(res.data.message);
          setSavingSettings(false);
          return;
        }
      }
      showSuccess(t('保存成功'));
    } catch (err) {
      showError(t('保存失败'));
    } finally {
      setSavingSettings(false);
    }
  };

  // 注册命令菜单
  const registerCommands = async () => {
    if (!botToken) {
      showError(t('请先填写并保存 Bot Token'));
      return;
    }
    setRegisteringCommands(true);
    try {
      const res = await API.post('/api/tgbot/register-commands');
      if (res.data.success) {
        showSuccess(res.data.message || t('命令菜单注册成功'));
      } else {
        showError(res.data.message || t('注册失败'));
      }
    } catch (err) {
      showError(err.response?.data?.message || t('注册失败'));
    } finally {
      setRegisteringCommands(false);
    }
  };

  // 设置 Webhook
  const setupWebhook = async () => {
    if (!botToken) {
      showError(t('请先填写并保存 Bot Token'));
      return;
    }
    setSettingWebhook(true);
    try {
      const res = await API.post('/api/tgbot/setup-webhook');
      if (res.data.success) {
        showSuccess(t('Webhook 设置成功'));
        loadWebhookInfo();
      } else {
        showError(res.data.message || t('设置失败'));
      }
    } catch (err) {
      showError(err.response?.data?.message || t('设置失败'));
    } finally {
      setSettingWebhook(false);
    }
  };

  // 获取 Webhook 信息
  const loadWebhookInfo = useCallback(async () => {
    try {
      const res = await API.get('/api/tgbot/webhook-info');
      if (res.data.success) {
        setWebhookInfo(res.data.data);
      }
    } catch {
      // ignore
    }
  }, []);

  // 加载分类
  const loadCategories = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/tgbot/category/');
      if (res.data.success) {
        setCategories(res.data.data || []);
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
    loadSettings();
    loadCategories();
    loadWebhookInfo();
  }, [loadSettings, loadWebhookInfo]);

  // ===== 分类 CRUD =====
  const openCreateModal = () => {
    setEditingCategory(null);
    setModalVisible(true);
  };

  const openEditModal = (record) => {
    setEditingCategory(record);
    setModalVisible(true);
  };

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      const payload = {
        ...values,
        max_claims: Number(values.max_claims) || 1,
      };
      if (editingCategory) {
        payload.id = editingCategory.id;
        const res = await API.put('/api/tgbot/category/', payload);
        if (res.data.success) {
          showSuccess(t('更新成功'));
        } else {
          showError(res.data.message);
          return;
        }
      } else {
        const res = await API.post('/api/tgbot/category/', payload);
        if (res.data.success) {
          showSuccess(t('创建成功'));
        } else {
          showError(res.data.message);
          return;
        }
      }
      setModalVisible(false);
      loadCategories();
    } catch (err) {
      showError(err.response?.data?.message || t('操作失败'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id) => {
    Modal.confirm({
      title: t('确认删除'),
      content: t('删除后不可恢复，确定要删除该分类吗？'),
      onOk: async () => {
        try {
          const res = await API.delete(`/api/tgbot/category/${id}`);
          if (res.data.success) {
            showSuccess(t('删除成功'));
            loadCategories();
          } else {
            showError(res.data.message);
          }
        } catch (err) {
          showError(err.response?.data?.message || t('删除失败'));
        }
      },
    });
  };

  const handleToggleStatus = async (record) => {
    const newStatus = record.status === 1 ? 2 : 1;
    try {
      const res = await API.put('/api/tgbot/category/', {
        id: record.id,
        name: record.name,
        description: record.description,
        max_claims: record.max_claims,
        purpose: record.purpose,
        status: newStatus,
      });
      if (res.data.success) {
        showSuccess(newStatus === 1 ? t('已启用') : t('已禁用'));
        loadCategories();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('操作失败'));
    }
  };

  // ===== 库存管理 API =====
  const openInventoryModal = async (category) => {
    setInventoryCategory(category);
    setInventoryModalVisible(true);
    setAddCodesText('');
    await loadInventory(category.id);
  };

  const loadInventory = async (categoryId) => {
    setInventoryLoading(true);
    try {
      const res = await API.get(`/api/tgbot/inventory/?category_id=${categoryId}`);
      if (res.data.success) {
        setInventoryItems(res.data.data || []);
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('加载库存失败'));
    } finally {
      setInventoryLoading(false);
    }
  };

  const handleAddCodes = async () => {
    if (!addCodesText.trim() || !inventoryCategory) return;
    setAddingCodes(true);
    try {
      const res = await API.post('/api/tgbot/inventory/', {
        category_id: inventoryCategory.id,
        codes: addCodesText,
      });
      if (res.data.success) {
        showSuccess(res.data.message || t('添加成功'));
        setAddCodesText('');
        await loadInventory(inventoryCategory.id);
        loadCategories();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('添加失败'));
    } finally {
      setAddingCodes(false);
    }
  };

  const handleDeleteInventoryItem = async (itemId) => {
    Modal.confirm({
      title: t('确认删除'),
      content: t('确定要删除该兑换码吗？'),
      onOk: async () => {
        try {
          const res = await API.delete(`/api/tgbot/inventory/${itemId}`);
          if (res.data.success) {
            showSuccess(t('删除成功'));
            if (inventoryCategory) {
              await loadInventory(inventoryCategory.id);
            }
            loadCategories();
          } else {
            showError(res.data.message);
          }
        } catch (err) {
          showError(err.response?.data?.message || t('删除失败'));
        }
      },
    });
  };

  const inventoryColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    {
      title: t('兑换码/邀请码'),
      dataIndex: 'code',
      width: 250,
      render: (text) => (
        <Typography.Text copyable style={{ fontFamily: 'monospace' }}>
          {text}
        </Typography.Text>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 100,
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'grey'}>
          {status === 1 ? t('可用') : t('已发放')}
        </Tag>
      ),
    },
    {
      title: t('领取者'),
      dataIndex: 'claimed_by',
      width: 150,
      render: (text) => text || '-',
    },
    {
      title: t('操作'),
      width: 80,
      render: (_, record) => (
        <Button
          size='small'
          type='danger'
          onClick={() => handleDeleteInventoryItem(record.id)}
        >
          {t('删除')}
        </Button>
      ),
    },
  ];

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: t('分类名称'), dataIndex: 'name', width: 150 },
    {
      title: t('描述'),
      dataIndex: 'description',
      width: 200,
      render: (text) => text || '-',
    },
    {
      title: t('兑换码类型'),
      dataIndex: 'purpose',
      width: 120,
      render: (purpose) => {
        const opt = PURPOSE_OPTIONS.find((o) => o.value === purpose);
        return (
          <Tag color={purpose === 2 ? 'blue' : 'green'}>
            {opt ? t(opt.label) : t('未知')}
          </Tag>
        );
      },
    },
    {
      title: t('库存(可用/总数)'),
      width: 130,
      render: (_, record) => {
        const available = record.stock_available ?? 0;
        const total = record.stock_total ?? 0;
        const color = available === 0 ? 'red' : available <= 5 ? 'orange' : 'green';
        return (
          <Tag color={color}>
            {available} / {total}
          </Tag>
        );
      },
    },
    { title: t('每人可领取次数'), dataIndex: 'max_claims', width: 130 },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 80,
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'grey'}>
          {status === 1 ? t('启用') : t('禁用')}
        </Tag>
      ),
    },
    {
      title: t('操作'),
      width: 300,
      fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button
            size='small'
            theme='light'
            type='primary'
            onClick={() => openInventoryModal(record)}
          >
            {t('管理库存')}
          </Button>
          <Button size='small' onClick={() => openEditModal(record)}>
            {t('编辑')}
          </Button>
          <Button
            size='small'
            type={record.status === 1 ? 'warning' : 'primary'}
            onClick={() => handleToggleStatus(record)}
          >
            {record.status === 1 ? t('禁用') : t('启用')}
          </Button>
          <Button
            size='small'
            type='danger'
            onClick={() => handleDelete(record.id)}
          >
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  const cardStyle = {
    boxShadow: '0 1px 3px rgba(0,0,0,0.04), 0 4px 16px rgba(0,0,0,0.02)',
    border: '1px solid var(--semi-color-border)',
  };

  return (
    <div className='mt-[60px] px-2 space-y-4'>
      {/* ===== Bot 基本设置 ===== */}
      <Card className='!rounded-2xl' style={cardStyle}>
        <Typography.Title heading={5} style={{ marginBottom: 16 }}>
          {t('TG 机器人设置')}
        </Typography.Title>

        <Spin spinning={settingsLoading}>
          <Form labelPosition='left' labelWidth={140}>
            <Form.Input
              field='TelegramBotToken'
              label='Bot Token'
              placeholder={t('从 @BotFather 获取的 Bot Token')}
              type='password'
              mode='password'
              value={botToken}
              onChange={setBotToken}
              extraText={t('在 Telegram 中找 @BotFather 创建机器人获取 Token')}
            />
            <Form.Input
              field='TelegramBotName'
              label={t('Bot 用户名')}
              placeholder={t('如：my_cool_bot')}
              value={botName}
              onChange={setBotName}
              extraText={t('机器人的用户名（不含 @）')}
            />
          </Form>

          <div className='flex items-center gap-3 mt-4'>
            <Button
              theme='solid'
              type='primary'
              loading={savingSettings}
              onClick={saveSettings}
            >
              {t('保存设置')}
            </Button>
            <Button
              theme='light'
              type='tertiary'
              loading={settingWebhook}
              onClick={setupWebhook}
            >
              {t('设置 Webhook')}
            </Button>
            <Button
              theme='light'
              type='tertiary'
              loading={registeringCommands}
              onClick={registerCommands}
            >
              {t('注册命令菜单')}
            </Button>
          </div>

          {webhookInfo && (
            <div className='mt-4'>
              <Descriptions
                size='small'
                row
                data={[
                  {
                    key: 'Webhook URL',
                    value: webhookInfo.url || t('未设置'),
                  },
                  {
                    key: t('状态'),
                    value: webhookInfo.url ? (
                      <Tag color='green'>{t('已设置')}</Tag>
                    ) : (
                      <Tag color='red'>{t('未设置')}</Tag>
                    ),
                  },
                  {
                    key: t('待处理更新'),
                    value: webhookInfo.pending_update_count ?? '-',
                  },
                ]}
              />
            </div>
          )}

          <Banner
            type='info'
            className='!rounded-lg mt-4'
            closeIcon={null}
            description={t(
              '用户在 Telegram 群组中与机器人交互时，会自动创建系统账户。管理员需要在下方添加分类，每个分类对应一个按钮，用户点击按钮即可领取对应的兑换码。',
            )}
          />
        </Spin>
      </Card>

      {/* ===== 分类管理 ===== */}
      <Card className='!rounded-2xl' style={cardStyle}>
        <div className='flex items-center justify-between mb-4 flex-wrap gap-2'>
          <Typography.Title heading={5} style={{ marginBottom: 0 }}>
            {t('领取分类管理')}
          </Typography.Title>
          <Button theme='solid' type='primary' onClick={openCreateModal}>
            {t('添加分类')}
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={categories}
          loading={loading}
          rowKey='id'
          pagination={false}
          scroll={{ x: 1100 }}
          empty={
            <div className='py-8 text-center text-gray-400'>
              {t('暂无分类，请添加')}
            </div>
          }
        />
      </Card>

      {/* ===== 添加/编辑分类弹窗 ===== */}
      <Modal
        title={editingCategory ? t('编辑分类') : t('添加分类')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        centered
        size='small'
      >
        <Form
          onSubmit={handleSubmit}
          initValues={
            editingCategory || {
              name: '',
              description: '',
              max_claims: 1,
              purpose: 1,
              status: 1,
            }
          }
          labelPosition='top'
        >
          <Form.Input
            field='name'
            label={t('分类名称')}
            placeholder={t('如：新手福利、每日签到奖励')}
            rules={[{ required: true, message: t('请输入分类名称') }]}
          />
          <Form.Input
            field='description'
            label={t('描述')}
            placeholder={t('可选，分类的简要描述')}
          />
          <Form.Select
            field='purpose'
            label={t('兑换码类型')}
            optionList={PURPOSE_OPTIONS.map((o) => ({
              ...o,
              label: t(o.label),
            }))}
            rules={[{ required: true, message: t('请选择兑换码类型') }]}
          />
          <Form.InputNumber
            field='max_claims'
            label={t('每人可领取次数')}
            min={1}
            max={9999}
            rules={[{ required: true, message: t('请输入领取次数') }]}
          />
          <Form.Select
            field='status'
            label={t('状态')}
            optionList={STATUS_OPTIONS.map((o) => ({
              ...o,
              label: t(o.label),
            }))}
          />
          <div className='flex justify-end gap-2 mt-4'>
            <Button onClick={() => setModalVisible(false)}>{t('取消')}</Button>
            <Button
              theme='solid'
              type='primary'
              htmlType='submit'
              loading={submitting}
            >
              {editingCategory ? t('更新') : t('创建')}
            </Button>
          </div>
        </Form>
      </Modal>

      {/* ===== 库存管理弹窗 ===== */}
      <Modal
        title={
          inventoryCategory
            ? `${t('库存管理')} - ${inventoryCategory.name}（${inventoryCategory.purpose === 2 ? t('邀请码') : t('兑换码')}）`
            : t('库存管理')
        }
        visible={inventoryModalVisible}
        onCancel={() => setInventoryModalVisible(false)}
        footer={null}
        centered
        width={700}
      >
        {/* 添加兑换码/邀请码 */}
        <div className='mb-4'>
          <Typography.Text strong className='mb-2 block'>
            {inventoryCategory?.purpose === 2
              ? t('添加邀请码')
              : t('添加兑换码')}
          </Typography.Text>
          <TextArea
            value={addCodesText}
            onChange={setAddCodesText}
            placeholder={t('每行一个兑换码/邀请码，支持批量添加')}
            rows={4}
            style={{ marginBottom: 8 }}
          />
          <Button
            theme='solid'
            type='primary'
            loading={addingCodes}
            onClick={handleAddCodes}
            disabled={!addCodesText.trim()}
          >
            {t('批量添加')}
          </Button>
        </div>

        {/* 库存列表 */}
        <div className='mb-2'>
          <Typography.Text strong>
            {t('当前库存')}
            {inventoryItems.length > 0 && (
              <Tag color='blue' style={{ marginLeft: 8 }}>
                {t('可用')} {inventoryItems.filter((i) => i.status === 1).length} / {t('总数')} {inventoryItems.length}
              </Tag>
            )}
          </Typography.Text>
        </div>
        <Table
          columns={inventoryColumns}
          dataSource={inventoryItems}
          loading={inventoryLoading}
          rowKey='id'
          pagination={{ pageSize: 10 }}
          size='small'
          scroll={{ y: 300 }}
          empty={
            <div className='py-4 text-center text-gray-400'>
              {t('暂无库存，请添加')}
            </div>
          }
        />
      </Modal>
    </div>
  );
};

export default TgBotPage;
