import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Form,
  InputNumber,
  Modal,
  Select,
  Space,
  Table,
  Tag,
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
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingCategory, setEditingCategory] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const [formApi, setFormApi] = useState(null);

  const loadData = async () => {
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
    loadData();
  }, []);

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
      loadData();
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
            loadData();
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
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('操作失败'));
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: t('分类名称'),
      dataIndex: 'name',
      width: 150,
    },
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
      title: t('每人可领取次数'),
      dataIndex: 'max_claims',
      width: 130,
    },
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
      width: 220,
      fixed: 'right',
      render: (_, record) => (
        <Space>
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

  return (
    <div className='mt-[60px] px-2'>
      <Card
        className='!rounded-2xl'
        style={{
          boxShadow:
            '0 1px 3px rgba(0,0,0,0.04), 0 4px 16px rgba(0,0,0,0.02)',
          border: '1px solid var(--semi-color-border)',
        }}
      >
        <div className='flex items-center justify-between mb-4 flex-wrap gap-2'>
          <Typography.Title heading={5} style={{ marginBottom: 0 }}>
            {t('TG 机器人分类管理')}
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
          scroll={{ x: 900 }}
          empty={
            <div className='py-8 text-center text-gray-400'>
              {t('暂无分类，请添加')}
            </div>
          }
        />
      </Card>

      <Modal
        title={editingCategory ? t('编辑分类') : t('添加分类')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        centered
        size='small'
      >
        <Form
          getFormApi={setFormApi}
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
    </div>
  );
};

export default TgBotPage;
