import React, { useCallback, useEffect, useMemo, useState } from 'react';
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
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Space,
  Table,
  Tag,
} from '@douyinfe/semi-ui';

const RegistrationCode = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [records, setRecords] = useState([]);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [formApi, setFormApi] = useState(null);

  const loadData = useCallback(
    async (page = activePage, size = pageSize, kw = keyword) => {
      setLoading(true);
      try {
        const path = kw
          ? `/api/registration-code/search?keyword=${encodeURIComponent(kw)}&p=${page}&page_size=${size}`
          : `/api/registration-code/?p=${page}&page_size=${size}`;
        const { data: resData } = await API.get(path);
        const { success, message, data } = resData;
        if (!success) {
          showError(message || t('加载失败'));
          return;
        }
        setRecords(data?.items || []);
        setTotal(data?.total || 0);
      } catch (error) {
        showError(error.message || t('加载失败'));
      } finally {
        setLoading(false);
      }
    },
    [activePage, keyword, pageSize, t],
  );

  useEffect(() => {
    loadData();
  }, [loadData]);

  const createCodes = async () => {
    const values = formApi?.getValues?.() || {};
    if (!values.name) {
      showError(t('请输入名称'));
      return;
    }
    const count = Number(values.count || 1);
    if (!count || count < 1) {
      showError(t('生成数量必须大于 0'));
      return;
    }
    setSubmitting(true);
    try {
      const payload = {
        name: values.name,
        count,
        expired_time: Number(values.expired_time || 0),
      };
      const { data: resData } = await API.post('/api/registration-code/', payload);
      const { success, message, data } = resData;
      if (!success) {
        showError(message || t('生成失败'));
        return;
      }
      if (Array.isArray(data) && data.length > 0) {
        copy(data.join('\n'));
        showSuccess(t('生成成功，已复制到剪贴板'));
      } else {
        showSuccess(t('生成成功'));
      }
      formApi?.reset?.();
      loadData(1, pageSize, keyword);
    } catch (error) {
      showError(error.message || t('生成失败'));
    } finally {
      setSubmitting(false);
    }
  };

  const updateStatus = async (row, status) => {
    try {
      const { data: resData } = await API.put('/api/registration-code/?status_only=true', {
        id: row.id,
        status,
      });
      if (!resData.success) {
        showError(resData.message || t('操作失败'));
        return;
      }
      showSuccess(t('操作成功'));
      loadData();
    } catch (error) {
      showError(error.message || t('操作失败'));
    }
  };

  const deleteCode = async (id) => {
    try {
      const { data: resData } = await API.delete(`/api/registration-code/${id}`);
      if (!resData.success) {
        showError(resData.message || t('删除失败'));
        return;
      }
      showSuccess(t('删除成功'));
      loadData();
    } catch (error) {
      showError(error.message || t('删除失败'));
    }
  };

  const columns = useMemo(
    () => [
      { title: 'ID', dataIndex: 'id', width: 80 },
      {
        title: t('注册码'),
        dataIndex: 'key',
        render: (value) => (
          <Space>
            <span>{value}</span>
            <Button size='small' type='tertiary' onClick={() => copy(value)}>
              {t('复制')}
            </Button>
          </Space>
        ),
      },
      { title: t('名称'), dataIndex: 'name', width: 160 },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 120,
        render: (status) => {
          if (status === 1) return <Tag color='green'>{t('可用')}</Tag>;
          if (status === 2) return <Tag color='orange'>{t('禁用')}</Tag>;
          return <Tag color='red'>{t('已使用')}</Tag>;
        },
      },
      {
        title: t('创建时间'),
        dataIndex: 'created_time',
        width: 180,
        render: (v) => (v ? timestamp2string(v) : '-'),
      },
      {
        title: t('过期时间'),
        dataIndex: 'expired_time',
        width: 180,
        render: (v) => (v ? timestamp2string(v) : t('永不过期')),
      },
      {
        title: t('使用者ID'),
        dataIndex: 'used_user_id',
        width: 120,
        render: (v) => (v || '-'),
      },
      {
        title: t('操作'),
        dataIndex: 'id',
        width: 220,
        render: (_, row) => (
          <Space>
            {row.status === 1 ? (
              <Button size='small' onClick={() => updateStatus(row, 2)}>
                {t('禁用')}
              </Button>
            ) : row.status === 2 ? (
              <Button size='small' onClick={() => updateStatus(row, 1)}>
                {t('启用')}
              </Button>
            ) : null}
            <Popconfirm
              title={t('确认删除该注册码？')}
              onConfirm={() => deleteCode(row.id)}
            >
              <Button size='small' type='danger'>
                {t('删除')}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [t],
  );

  return (
    <div className='mt-[60px] px-2'>
      <Card title={t('管理员注册码管理')}>
        <Form getFormApi={setFormApi} layout='horizontal'>
          <Space wrap>
            <Form.Input field='name' label={t('名称')} placeholder={t('例如：LinuxDO 新用户')} />
            <Form.InputNumber field='count' label={t('生成数量')} min={1} max={100} initValue={1} />
            <Form.Input
              field='expired_time'
              label={t('过期时间戳')}
              placeholder={t('可选，秒级时间戳，留空表示不过期')}
            />
            <Button type='primary' loading={submitting} onClick={createCodes}>
              {t('生成注册码')}
            </Button>
          </Space>
        </Form>

        <div className='mt-4 mb-3 flex gap-2'>
          <Input
            value={keyword}
            onChange={setKeyword}
            placeholder={t('按名称或ID搜索')}
            style={{ maxWidth: 300 }}
          />
          <Button
            onClick={() => {
              setActivePage(1);
              loadData(1, pageSize, keyword);
            }}
          >
            {t('搜索')}
          </Button>
          <Button
            onClick={() => {
              setKeyword('');
              setActivePage(1);
              loadData(1, pageSize, '');
            }}
          >
            {t('重置')}
          </Button>
        </div>

        <Table
          loading={loading}
          columns={columns}
          dataSource={records}
          pagination={{
            currentPage: activePage,
            pageSize,
            total,
            onPageChange: (page) => {
              setActivePage(page);
              loadData(page, pageSize, keyword);
            },
            onPageSizeChange: (size) => {
              setPageSize(size);
              setActivePage(1);
              loadData(1, size, keyword);
            },
            showSizeChanger: true,
          }}
        />
      </Card>
    </div>
  );
};

export default RegistrationCode;
