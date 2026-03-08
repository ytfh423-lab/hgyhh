import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './utils';

const { Text } = Typography;

const TasksPage = ({ actionLoading, loadFarm, t }) => {
  const [taskData, setTaskData] = useState(null);
  const [loading, setLoading] = useState(false);

  const loadTasks = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/tasks');
      if (res.success) setTaskData(res.data);
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => { loadTasks(); }, [loadTasks]);

  const claimTask = async (index) => {
    try {
      const { data: res } = await API.post('/api/farm/tasks/claim', { index });
      if (res.success) {
        showSuccess(res.message);
        loadTasks();
        loadFarm();
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    }
  };

  if (loading && !taskData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!taskData) return null;

  const dateStr = taskData.date || '';
  const dateDisplay = dateStr.length === 8 ? `${dateStr.slice(0,4)}-${dateStr.slice(4,6)}-${dateStr.slice(6)}` : dateStr;

  return (
    <div>
      <div className='farm-pill farm-pill-blue' style={{ marginBottom: 14 }}>📅 {dateDisplay}</div>

      <div className='farm-card'>
        <div className='farm-section-title'>📝 {t('今日任务')}</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {(taskData.tasks || []).map((task) => (
            <div key={task.index} className='farm-row' style={{
              background: task.claimed ? 'rgba(74,124,63,0.08)' : undefined,
              marginBottom: 0,
            }}>
              <span style={{ fontSize: 22 }}>{task.emoji}</span>
              <div style={{ flex: 1 }}>
                <Text strong>{task.name}</Text>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
                  <div className='farm-progress' style={{ width: 80 }}>
                    <div className='farm-progress-fill' style={{
                      width: `${Math.min(100, (task.progress / task.target) * 100)}%`,
                      background: task.claimed ? 'var(--farm-leaf)' : task.done ? 'var(--farm-harvest)' : 'var(--farm-sky)',
                    }} />
                  </div>
                  <Text size='small' type='tertiary'>{task.progress}/{task.target}</Text>
                </div>
                <Text size='small' type='tertiary'>{t('奖励')}: ${task.reward.toFixed(2)}</Text>
              </div>
              {task.claimed ? (
                <Tag size='small' color='green'>✅</Tag>
              ) : task.done ? (
                <Button size='small' theme='solid' type='warning' onClick={() => claimTask(task.index)} className='farm-btn'>
                  {t('领取')}
                </Button>
              ) : (
                <Tag size='small' color='grey'>{t('未完成')}</Tag>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default TasksPage;
