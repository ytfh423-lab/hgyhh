import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Descriptions, Banner, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './utils';

const { Text, Title } = Typography;

const PrestigePage = ({ loadFarm, t }) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);

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
    if (!window.confirm(t('转生将重置所有进度（保留成就和图鉴），确定吗？'))) return;
    try {
      const { data: res } = await API.post('/api/farm/prestige');
      if (res.success) { showSuccess(res.message); load(); loadFarm(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
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
      <Descriptions size='small' row data={[
        { key: t('当前等级'), value: `Lv.${data.current_level}` },
        { key: t('转生次数'), value: data.prestige_level },
        { key: t('当前加成'), value: `+${data.current_bonus}%` },
        { key: t('转生后加成'), value: `+${data.next_bonus}%` },
        { key: t('每次加成'), value: `+${data.bonus_per_level}%` },
        { key: t('需要等级'), value: `Lv.${data.min_level}` },
      ]} />
      <Banner type='warning' style={{ marginTop: 12, marginBottom: 12, borderRadius: 10 }}
        description={t('转生将重置：等级、地块、仓库、狗、牧场、加工。保留：成就、图鉴。获得永久收入加成。')} />
      <Button theme='solid' type='warning' disabled={!data.can_prestige} onClick={doPrestige} className='farm-btn'>
        {data.can_prestige ? `🔄 ${t('转生')} (+${data.bonus_per_level}%)` : `${t('需要')} Lv.${data.min_level}`}
      </Button>
    </div>
  );
};

export default PrestigePage;
