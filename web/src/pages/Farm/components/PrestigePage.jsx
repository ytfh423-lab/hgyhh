import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Banner, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './utils';
import FarmConfirmModal from './FarmConfirmModal';

const { Text, Title } = Typography;

const PrestigePage = ({ loadFarm, t }) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [prestigeLoading, setPrestigeLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/prestige');
      if (res.success) setData(res.data);
    } catch (err) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const doPrestige = async () => {
    setPrestigeLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/prestige');
      if (res.success) { showSuccess(res.message); load(); loadFarm(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
    finally { setPrestigeLoading(false); setShowConfirm(false); }
  };

  if (loading && !data) return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  if (!data) return null;

  return (
    <div className='farm-card farm-card-glow-purple'>
      <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 14 }}>
        <span style={{ fontSize: 36 }}>🔄</span>
        <div>
          <Title heading={5} style={{ margin: 0 }}>{t('转生系统')}</Title>
          <Text type='tertiary' size='small'>{t('满级后重置换取永久加成')}</Text>
        </div>
      </div>
      <div className='farm-kv-grid' style={{ marginBottom: 14 }}>
        <div className='farm-kv'>
          <div className='farm-kv-label'>{t('当前等级')}</div>
          <div className='farm-kv-value'>Lv.{data.current_level}</div>
        </div>
        <div className='farm-kv'>
          <div className='farm-kv-label'>{t('转生次数')}</div>
          <div className='farm-kv-value'>{data.prestige_level}/{data.max_times}</div>
        </div>
        <div className='farm-kv'>
          <div className='farm-kv-label'>{t('当前加成')}</div>
          <div className='farm-kv-value' style={{ color: 'var(--farm-leaf)' }}>+{data.current_bonus}%</div>
        </div>
        <div className='farm-kv'>
          <div className='farm-kv-label'>{t('转生后加成')}</div>
          <div className='farm-kv-value' style={{ color: '#8a6cb0' }}>+{data.next_bonus}%</div>
        </div>
        <div className='farm-kv'>
          <div className='farm-kv-label'>{t('本次价格')}</div>
          <div className='farm-kv-value'>${data.next_price?.toFixed(2)}</div>
        </div>
        <div className='farm-kv'>
          <div className='farm-kv-label'>{t('每次加成')}</div>
          <div className='farm-kv-value'>+{data.bonus_per_level}%</div>
        </div>
        <div className='farm-kv'>
          <div className='farm-kv-label'>{t('需要等级')}</div>
          <div className='farm-kv-value'>Lv.{data.min_level}</div>
        </div>
      </div>
      <Banner type='warning' style={{ marginBottom: 12, borderRadius: 10 }}
        description={`${t('转生将收费，首次转生也需要支付。转生后重置：余额到')} ${data.reset_balance?.toFixed(2)}${t('、等级、地块、仓库、狗、牧场、加工及现有物品。保留：成就、图鉴。获得永久收入加成。')}`} />
      <Button theme='solid' type='warning' disabled={!data.can_prestige} onClick={() => setShowConfirm(true)} className='farm-btn'>
        {data.can_prestige
          ? `🔄 ${t('转生')} (+${data.bonus_per_level}%)`
          : data.prestige_level >= data.max_times
            ? t('已达上限')
            : `${t('需要')} Lv.${data.min_level}`}
      </Button>

      <FarmConfirmModal
        visible={showConfirm}
        title={t('确认转生')}
        message={`${t('转生将把余额重置到')} ${data.reset_balance?.toFixed(2)}${t('，并清空现有物品与大部分进度，仅保留成就和图鉴，确定吗？')}`}
        icon={<span style={{ fontSize: 28 }}>🔄</span>}
        confirmText={t('确认转生')}
        cancelText={t('取消')}
        confirmType='warning'
        loading={prestigeLoading}
        onConfirm={doPrestige}
        onCancel={() => setShowConfirm(false)}
      />
    </div>
  );
};

export default PrestigePage;
