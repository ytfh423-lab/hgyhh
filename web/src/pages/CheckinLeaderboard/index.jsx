import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Typography,
  Avatar,
  Spin,
  Tag,
  Input,
} from '@douyinfe/semi-ui';
import { Trophy, Medal, Award, Search } from 'lucide-react';
import { API, showError, renderQuota } from '../../helpers';
import { useTranslation } from 'react-i18next';

const { Title, Text } = Typography;

const PAGE_SIZE = 50;

const CheckinLeaderboard = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [leaderboard, setLeaderboard] = useState([]);
  const [total, setTotal] = useState(0);
  const [limit, setLimit] = useState(100);
  const [currentPage, setCurrentPage] = useState(1);
  const [keyword, setKeyword] = useState('');
  const [searchMode, setSearchMode] = useState(false);
  const [searchTimer, setSearchTimer] = useState(null);
  const userId = parseInt(localStorage.getItem('user_id') || '0');

  const fetchLeaderboard = async (page = 1, search = '') => {
    setLoading(true);
    try {
      let url = `/api/user/checkin/leaderboard?page=${page}`;
      if (search) {
        url = `/api/user/checkin/leaderboard?page=${page}&keyword=${encodeURIComponent(search)}`;
      }
      const res = await API.get(url);
      const { success, data, message } = res.data;
      if (success) {
        setLeaderboard(data || []);
        setTotal(res.data.total || 0);
        setLimit(res.data.limit || 100);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('获取排行榜失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLeaderboard(currentPage, searchMode ? keyword.trim() : '');
  }, [currentPage]);

  const handleSearch = (value) => {
    setKeyword(value);
    if (searchTimer) clearTimeout(searchTimer);
    if (!value.trim()) {
      setSearchMode(false);
      setCurrentPage(1);
      fetchLeaderboard(1);
      return;
    }
    const timer = setTimeout(() => {
      setSearchMode(true);
      setCurrentPage(1);
      fetchLeaderboard(1, value.trim());
    }, 400);
    setSearchTimer(timer);
  };

  const getRankIcon = (rank) => {
    if (rank === 1) return <Trophy size={20} style={{ color: '#FFD700' }} />;
    if (rank === 2) return <Medal size={20} style={{ color: '#C0C0C0' }} />;
    if (rank === 3) return <Award size={20} style={{ color: '#CD7F32' }} />;
    return null;
  };

  const getRankStyle = (rank) => {
    if (rank === 1)
      return {
        background: 'linear-gradient(135deg, #FFF8E1, #FFE082)',
        fontWeight: 800,
        color: '#F57F17',
      };
    if (rank === 2)
      return {
        background: 'linear-gradient(135deg, #F5F5F5, #E0E0E0)',
        fontWeight: 700,
        color: '#616161',
      };
    if (rank === 3)
      return {
        background: 'linear-gradient(135deg, #FBE9E7, #FFCCBC)',
        fontWeight: 700,
        color: '#BF360C',
      };
    return {};
  };

  const columns = [
    {
      title: t('排名'),
      dataIndex: 'rank',
      width: 80,
      align: 'center',
      render: (rank) => {
        const icon = getRankIcon(rank);
        const style = getRankStyle(rank);
        return (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4 }}>
            {icon}
            <span
              style={{
                fontSize: rank <= 3 ? '18px' : '14px',
                fontWeight: rank <= 3 ? 800 : 500,
                color: style.color || 'var(--semi-color-text-0)',
              }}
            >
              {rank}
            </span>
          </div>
        );
      },
    },
    {
      title: t('用户'),
      dataIndex: 'username',
      render: (_, record) => {
        const isMe = record.user_id === userId;
        const displayName = record.display_name || record.username || '?';
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Avatar
              size='small'
              style={{
                background: record.rank <= 3
                  ? ['', '#FFD700', '#C0C0C0', '#CD7F32'][record.rank]
                  : 'var(--semi-color-primary)',
              }}
            >
              {displayName[0].toUpperCase()}
            </Avatar>
            <span style={{ fontWeight: isMe ? 700 : 400 }}>
              {displayName}
              {isMe && (
                <Tag size='small' color='green' style={{ marginLeft: 6 }}>
                  {t('我')}
                </Tag>
              )}
            </span>
          </div>
        );
      },
    },
    {
      title: t('累计签到额度'),
      dataIndex: 'total_quota',
      align: 'right',
      render: (val) => (
        <Text strong style={{ color: 'var(--semi-color-warning)', fontSize: '14px' }}>
          {renderQuota(val)}
        </Text>
      ),
    },
    {
      title: t('签到天数'),
      dataIndex: 'total_days',
      width: 100,
      align: 'center',
      render: (val) => (
        <Tag color='blue' size='small'>
          {val} {t('天')}
        </Tag>
      ),
    },
  ];

  return (
    <div style={{ maxWidth: 800, margin: '20px auto', padding: '20px 16px' }}>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Trophy size={22} style={{ color: '#FFD700' }} />
            <Title heading={5} style={{ margin: 0 }}>
              {t('屯屯鼠排行榜')}
            </Title>
          </div>
        }
        headerExtraContent={
          <Text type='secondary' size='small'>
            {t('签到累计额度 TOP {{limit}}', { limit })}
          </Text>
        }
      >
        <div style={{ marginBottom: 12 }}>
          <Input
            prefix={<Search size={16} />}
            placeholder={t('搜索用户名查找排名')}
            value={keyword}
            onChange={handleSearch}
            showClear
          />
        </div>
        <Spin spinning={loading}>
          <Table
            columns={columns}
            dataSource={leaderboard}
            rowKey='rank'
            size='small'
            empty={<Text type='tertiary'>{t('暂无排行数据')}</Text>}
            pagination={
              total > PAGE_SIZE
                ? {
                    currentPage,
                    pageSize: PAGE_SIZE,
                    total,
                    onPageChange: (page) => setCurrentPage(page),
                  }
                : false
            }
            onRow={(record) => {
              const style = {};
              if (record.user_id === userId) {
                style.background = 'var(--semi-color-primary-light-default)';
              }
              if (record.rank <= 3) {
                style.background = getRankStyle(record.rank).background;
              }
              return { style };
            }}
          />
        </Spin>
      </Card>
    </div>
  );
};

export default CheckinLeaderboard;
