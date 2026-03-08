import React, { useCallback, useEffect, useState } from 'react';
import { Button, Empty, Spin, Tag, Typography, Modal } from '@douyinfe/semi-ui';
import { RefreshCw, Droplets, Sprout, Axe, Trash2, Plus, TreePine } from 'lucide-react';
import { API, formatBalance, formatDuration, showError } from './utils';

const { Text, Title } = Typography;

const statusLabels = { 0: '空地', 1: '生长中', 2: '已成熟', 3: '树桩' };
const statusColors = { 0: 'grey', 1: 'blue', 2: 'green', 3: 'orange' };
const treeStageEmojis = { 0: '⬜', 1: '🌱', 2: '🌳', 3: '🪵' };

const TreeFarmPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [treeData, setTreeData] = useState(null);
  const [treeTypes, setTreeTypes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [plantModal, setPlantModal] = useState({ visible: false, slotIndex: null });
  const [detailSlot, setDetailSlot] = useState(null);

  const loadTree = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/tree/view');
      if (res.success) setTreeData(res.data);
      else if (res.message) showError(res.message);
    } catch (err) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  const loadTypes = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/tree/types');
      if (res.success) setTreeTypes(res.data || []);
    } catch (err) { /* ignore */ }
  }, []);

  useEffect(() => { loadTree(); loadTypes(); }, [loadTree, loadTypes]);

  useEffect(() => {
    const interval = setInterval(loadTree, 30000);
    return () => clearInterval(interval);
  }, [loadTree]);

  const doTreeAction = async (url, body) => {
    const res = await doAction(url, body);
    if (res) { loadTree(); loadFarm(); }
    return res;
  };

  if (loading && !treeData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!treeData) return null;

  const slots = treeData.slots || [];
  const emptySlots = slots.filter(s => s.status === 0);
  const activeSlots = slots.filter(s => s.status > 0);

  const handlePlant = async (treeKey, slotIndex) => {
    setPlantModal({ visible: false, slotIndex: null });
    await doTreeAction('/api/tree/plant', { tree_key: treeKey, slot_index: slotIndex });
  };

  return (
    <div>
      {/* Status bar */}
      <div className='farm-card' style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 8, padding: '10px 16px' }}>
        <div className='farm-pill farm-pill-green'>💰 {formatBalance(treeData.balance)}</div>
        <div className='farm-pill'>🌲 {slots.length}/{treeData.max_slots}</div>
        <div className='farm-pill farm-pill-cyan'>💧 {t('浇水加速')} {treeData.water_bonus}%</div>
        <div className='farm-pill farm-pill-blue'>🧪 {t('施肥加速')} {treeData.fert_bonus}%</div>
        <div style={{ flex: 1 }} />
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless' onClick={loadTree} loading={loading} className='farm-btn' />
        {slots.length < treeData.max_slots && (
          <Button size='small' theme='light' onClick={() => doTreeAction('/api/tree/buyslot', {})}
            loading={actionLoading} className='farm-btn' style={{ color: 'var(--farm-soil)', borderColor: 'var(--farm-harvest)' }}>
            <Plus size={12} /> {t('开垦树位')} ({formatBalance(treeData.slot_price)})
          </Button>
        )}
      </div>

      {/* Tree grid */}
      <div className='farm-card'>
        <div className='farm-section-title'>🌲 {t('树场')}</div>
        {slots.length === 0 ? (
          <Empty description={t('还没有树位')} style={{ padding: '20px 0' }} />
        ) : (
          <div className='tree-farm-grid'>
            {slots.map((slot) => {
              const treeType = treeTypes.find(tt => tt.key === slot.tree_type);
              return (
                <div
                  key={slot.slot_index}
                  className={`tree-slot tree-slot-status-${slot.status} ${detailSlot?.slot_index === slot.slot_index ? 'tree-slot-selected' : ''}`}
                  onClick={() => setDetailSlot(slot)}
                >
                  <div className='tree-slot-stage'>
                    {slot.status === 0 ? (
                      <span className='tree-slot-empty-icon'>⬜</span>
                    ) : (
                      <span className='tree-slot-emoji'>{slot.tree_emoji || treeStageEmojis[slot.status]}</span>
                    )}
                  </div>
                  <div className='tree-slot-label'>
                    {slot.status === 0 ? (
                      <Text type='tertiary' size='small'>#{slot.slot_index + 1} {t('空地')}</Text>
                    ) : (
                      <Text size='small' strong>{slot.tree_name}</Text>
                    )}
                  </div>
                  {slot.status === 1 && (
                    <div className='tree-slot-progress'>
                      <div className='tree-slot-progress-bar' style={{ width: `${slot.progress}%` }} />
                    </div>
                  )}
                  {slot.status === 1 && (
                    <Text type='tertiary' size='small' className='tree-slot-time'>{slot.progress}%</Text>
                  )}
                  {slot.status === 2 && (
                    <Tag color='green' size='small' className='tree-slot-tag'>{t('成熟')}</Tag>
                  )}
                  {slot.status === 3 && (
                    <Tag color='orange' size='small' className='tree-slot-tag'>{t('树桩')}</Tag>
                  )}
                  {slot.status === 2 && slot.can_harvest && (
                    <div className='tree-slot-harvest-dot' />
                  )}
                  <div className='tree-slot-indicators'>
                    {slot.fertilized === 1 && <span title={t('已施肥')}>🧪</span>}
                    {slot.water_remain > 0 && <span title={t('浇水中')}>💧</span>}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Detail panel */}
      {detailSlot && (
        <div className='farm-card tree-detail-panel'>
          {detailSlot.status === 0 ? (
            <div style={{ textAlign: 'center', padding: 16 }}>
              <span style={{ fontSize: 32 }}>⬜</span>
              <Title heading={6} style={{ marginTop: 8 }}>#{detailSlot.slot_index + 1} {t('空地')}</Title>
              <Text type='tertiary' size='small'>{t('点击下方按钮种植树苗')}</Text>
              <div style={{ marginTop: 16 }}>
                <Button theme='solid' onClick={() => setPlantModal({ visible: true, slotIndex: detailSlot.slot_index })}
                  className='farm-btn' icon={<Sprout size={14} />}>
                  {t('种植树苗')}
                </Button>
              </div>
            </div>
          ) : detailSlot.status === 3 ? (
            <div style={{ textAlign: 'center', padding: 16 }}>
              <span style={{ fontSize: 32 }}>🪵</span>
              <Title heading={6} style={{ marginTop: 8 }}>{detailSlot.tree_name} - {t('树桩')}</Title>
              {detailSlot.stump_remain > 0 ? (
                <Text type='tertiary' size='small'>{t('冷却中')}... {formatDuration(detailSlot.stump_remain)}</Text>
              ) : (
                <div style={{ marginTop: 16 }}>
                  <Button theme='solid' type='warning' onClick={() => doTreeAction('/api/tree/clear', { slot_index: detailSlot.slot_index })}
                    loading={actionLoading} className='farm-btn' icon={<Trash2 size={14} />}>
                    {t('清理树桩')}
                  </Button>
                </div>
              )}
            </div>
          ) : (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 12 }}>
                <span style={{ fontSize: 36 }}>{detailSlot.tree_emoji}</span>
                <div>
                  <Title heading={6} style={{ margin: 0 }}>
                    {detailSlot.tree_name}
                    <Tag color={statusColors[detailSlot.status]} size='small' style={{ marginLeft: 8 }}>
                      {t(statusLabels[detailSlot.status])}
                    </Tag>
                  </Title>
                  <Text type='tertiary' size='small'>
                    #{detailSlot.slot_index + 1} · {t('采收')} {detailSlot.harvest_count} {t('次')}
                  </Text>
                </div>
              </div>

              {detailSlot.status === 1 && (
                <div style={{ marginBottom: 12 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 4 }}>
                    <Text type='tertiary'>{t('生长进度')}</Text>
                    <Text type='tertiary'>{detailSlot.progress}% · {formatDuration(detailSlot.remaining)}</Text>
                  </div>
                  <div className='tree-detail-progress'>
                    <div className='tree-detail-progress-bar' style={{ width: `${detailSlot.progress}%` }} />
                  </div>
                </div>
              )}

              {detailSlot.status === 2 && detailSlot.repeatable && detailSlot.harvest_cooldown > 0 && (
                <div style={{ marginBottom: 12 }}>
                  <Text type='tertiary' size='small'>⏳ {t('下次可采收')}: {formatDuration(detailSlot.harvest_cooldown)}</Text>
                </div>
              )}

              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {detailSlot.status === 1 && (
                  <>
                    <Button size='small' theme='light' onClick={() => doTreeAction('/api/tree/water', { slot_index: detailSlot.slot_index })}
                      loading={actionLoading} disabled={detailSlot.water_remain > 0} className='farm-btn'
                      icon={<Droplets size={13} />}>
                      {detailSlot.water_remain > 0 ? `💧 ${formatDuration(detailSlot.water_remain)}` : t('浇水')}
                    </Button>
                    <Button size='small' theme='light' onClick={() => doTreeAction('/api/tree/fertilize', { slot_index: detailSlot.slot_index })}
                      loading={actionLoading} disabled={detailSlot.fertilized === 1} className='farm-btn'>
                      {detailSlot.fertilized === 1 ? `🧪 ${t('已施肥')}` : `🧪 ${t('施肥')}`}
                    </Button>
                  </>
                )}
                {detailSlot.status === 2 && detailSlot.can_harvest && (
                  <Button size='small' theme='solid' type='primary' onClick={() => doTreeAction('/api/tree/harvest', { slot_index: detailSlot.slot_index })}
                    loading={actionLoading} className='farm-btn'>
                    🍎 {t('采收果实')}
                  </Button>
                )}
                {detailSlot.status === 2 && detailSlot.can_chop && (
                  <Button size='small' theme='solid' type='warning' onClick={() => doTreeAction('/api/tree/chop', { slot_index: detailSlot.slot_index })}
                    loading={actionLoading} className='farm-btn' icon={<Axe size={13} />}>
                    {t('伐木')}
                  </Button>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Tree types reference */}
      <div className='farm-card'>
        <div className='farm-section-title'>📖 {t('树种图鉴')}</div>
        <div className='tree-types-list'>
          {treeTypes.map((tt) => (
            <div key={tt.key} className='tree-type-card'>
              <div className='tree-type-header'>
                <span className='tree-type-emoji'>{tt.emoji}</span>
                <div>
                  <Text strong>{tt.name}</Text>
                  <div style={{ fontSize: 11 }}>
                    <Text type='tertiary'>{tt.description}</Text>
                  </div>
                </div>
                <span className='farm-pill farm-pill-green' style={{ marginLeft: 'auto', padding: '1px 8px', fontSize: 11 }}>
                  {formatBalance(tt.seed_cost)}
                </span>
              </div>
              <div className='tree-type-stats'>
                <span>⏱ {formatDuration(tt.grow_secs)}</span>
                {tt.repeatable && <span>🔄 {t('可反复采收')}</span>}
                {tt.repeatable && <span>⏳ {formatDuration(tt.harvest_cooldown)}/{t('次')}</span>}
                {tt.can_chop && <span>🪓 {t('可伐木')}</span>}
              </div>
              {tt.harvest_yield && tt.harvest_yield.length > 0 && (
                <div className='tree-type-yields'>
                  <Text type='tertiary' size='small'>🍎 {t('采收')}:</Text>
                  {tt.harvest_yield.map((y, i) => (
                    <span key={i} className='tree-yield-tag'>{y.emoji} {y.name} {y.amount_min}~{y.amount_max}</span>
                  ))}
                </div>
              )}
              {tt.chop_yield && tt.chop_yield.length > 0 && (
                <div className='tree-type-yields'>
                  <Text type='tertiary' size='small'>🪓 {t('伐木')}:</Text>
                  {tt.chop_yield.map((y, i) => (
                    <span key={i} className='tree-yield-tag'>{y.emoji} {y.name} {y.amount_min}~{y.amount_max}</span>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Plant modal */}
      <Modal
        title={`🌱 ${t('选择树种')} - #${(plantModal.slotIndex ?? 0) + 1} ${t('号树位')}`}
        visible={plantModal.visible}
        onCancel={() => setPlantModal({ visible: false, slotIndex: null })}
        footer={null}
        width={520}
        bodyStyle={{ maxHeight: 400, overflowY: 'auto' }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {treeTypes.map((tt) => (
            <div key={tt.key}
              className='tree-plant-option'
              onClick={() => handlePlant(tt.key, plantModal.slotIndex)}
            >
              <span style={{ fontSize: 28 }}>{tt.emoji}</span>
              <div style={{ flex: 1 }}>
                <Text strong>{tt.name}</Text>
                <div style={{ fontSize: 11 }}>
                  <Text type='tertiary'>
                    ⏱ {formatDuration(tt.grow_secs)}
                    {tt.repeatable ? ` · 🔄 ${t('可反复采收')}` : ` · 🪓 ${t('一次性伐木')}`}
                  </Text>
                </div>
              </div>
              <span className='farm-pill farm-pill-green' style={{ fontSize: 12 }}>
                {formatBalance(tt.seed_cost)}
              </span>
            </div>
          ))}
        </div>
      </Modal>
    </div>
  );
};

export default TreeFarmPage;
