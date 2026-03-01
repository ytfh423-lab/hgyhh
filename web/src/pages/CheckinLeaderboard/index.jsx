import React, { useState, useEffect, useContext } from 'react';
import {
  Card,
  Table,
  Typography,
  Avatar,
  Spin,
  Tag,
} from '@douyinfe/semi-ui';
import { Trophy, Medal, Award } from 'lucide-react';
import { API, showError, renderQuota } from '../../helpers';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../context/Status';

const { Title, Text } = Typography;

const CheckinLeaderboard = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [leaderboard, setLeaderboard] = useState([]);
  const [statusState] = useContext(StatusContext);
  const userId = parseInt(localStorage.getItem('user_id') || '0');

  const fetchLeaderboard = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/user/self/checkin/leaderboard');
      const { success, data, message } = res.data;
      if (success) {
        setLeaderboard(data || []);
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
    fetchLeaderboard();
  }, []);

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

  const getMaskedName = (username, displayName) => {
    const name = displayName || username || '';
    if (name.length <= 1) return name + '**';
    if (name.length <= 3) return name[0] + '**';
    return name[0] + '**' + name[name.length - 1];
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
                ...style,
                background: undefined,
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
        const name = getMaskedName(record.username, record.display_name);
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
              {(record.display_name || record.username || '?')[0].toUpperCase()}
            </Avatar>
            <span style={{ fontWeight: isMe ? 700 : 400 }}>
              {name}
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
    <div style={{ maxWidth: 800, margin: '0 auto', padding: '20px 16px' }}>
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
            {t('签到累计额度 TOP 100')}
          </Text>
        }
      >
        <Spin spinning={loading}>
          <Table
            columns={columns}
            dataSource={leaderboard}
            rowKey='rank'
            pagination={false}
            size='small'
            empty={<Text type='tertiary'>{t('暂无排行数据')}</Text>}
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
