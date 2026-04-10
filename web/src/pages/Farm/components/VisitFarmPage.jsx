import React, { useState, useEffect, useCallback, memo } from 'react';
import { Button, Select, Spin } from '@douyinfe/semi-ui';
import { RefreshCw, Droplets, FlaskConical, Wheat, Sprout, Pill, ArrowLeft } from 'lucide-react';
import { API, showSuccess, showError, formatDuration } from './utils';

/* ─── 只读地块卡片 ─── */
const statusClassMap = {
  0: 'farm-plot-card farm-plot-empty',
  1: 'farm-plot-card farm-plot-growing',
  2: 'farm-plot-card farm-plot-mature',
  3: 'farm-plot-card farm-plot-event',
  4: 'farm-plot-card farm-plot-wilting',
};

const VisitPlotCard = memo(({ plot }) => {
  const st = plot.status;
  if (st === 0) {
    return (
      <div className={statusClassMap[0]}
        style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: 110, textAlign: 'center' }}>
        <span style={{ fontSize: 22, color: 'var(--farm-text-3)', marginBottom: 4 }}>🌱</span>
        <span style={{ fontSize: 12, color: 'var(--farm-text-3)' }}>空地 #{plot.plot_index + 1}</span>
      </div>
    );
  }
  const emoji = st === 4 ? '🥀' : plot.crop_emoji;
  return (
    <div className={statusClassMap[st] || statusClassMap[1]}>
      <div style={{ position: 'absolute', top: 6, right: 8, fontSize: 10, color: 'var(--farm-text-3)', opacity: 0.6 }}>#{plot.plot_index + 1}</div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
        <span style={{ fontSize: 28, lineHeight: 1 }}>{emoji}</span>
        <div>
          <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--farm-text-0)' }}>{plot.crop_name}</div>
          <div style={{ fontSize: 11, color: 'var(--farm-text-2)', marginTop: 1 }}>
            {st === 1 && `⏳ ${formatDuration(plot.remaining)}`}
            {st === 2 && '✅ 可收获'}
            {st === 3 && (plot.event_type === 'drought' ? '🏜️ 干旱' : '🐛 虫害')}
            {st === 4 && '🥀 枯萎中'}
          </div>
        </div>
      </div>
      {st === 1 && (
        <div className='farm-progress' style={{ height: 6 }}>
          <div className='farm-progress-fill'
            style={{ width: `${plot.progress}%`, background: 'linear-gradient(90deg, var(--farm-sky), var(--farm-leaf))' }} />
        </div>
      )}
    </div>
  );
});

/* ─── 主组件 ─── */
const VisitFarmPage = ({ friendId, friendName, onBack, t }) => {
  const [farmData, setFarmData] = useState(null);
  const [inventory, setInventory] = useState(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [plantCrop, setPlantCrop] = useState('');
  const [crops, setCrops] = useState([]);

  const base = `/api/farm/visit/${friendId}`;

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const [farmRes, invRes, cropRes] = await Promise.all([
        API.get(`${base}`),
        API.get(`${base}/inventory`),
        API.get('/api/farm/crops'),
      ]);
      if (farmRes.data.success) setFarmData(farmRes.data.data);
      if (invRes.data.success)  setInventory(invRes.data.data);
      if (cropRes.data.success) setCrops(cropRes.data.data || []);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, [base]);

  useEffect(() => { loadAll(); }, [loadAll]);

  const doAction = useCallback(async (path, body = {}) => {
    setActionLoading(true);
    try {
      const { data: res } = await API.post(`${base}${path}`, body);
      if (res.success) { showSuccess(res.message); loadAll(); }
      else showError(res.message);
    } catch { showError('操作失败'); }
    finally { setActionLoading(false); }
  }, [base, loadAll]);

  if (loading && !farmData) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }

  const plots = farmData?.plots || [];
  const matureCount = plots.filter(p => p.status === 2).length;
  const needsWaterCount = plots.filter(p => p.status === 1 || p.status === 4 || (p.status === 3 && p.event_type === 'drought')).length;
  const emptyCount = plots.filter(p => p.status === 0).length;
  const eventCount = plots.filter(p => p.status === 3 && p.event_type !== 'drought').length;
  const canFertilize = plots.some(p => p.status === 1 && p.fertilized === 0) && inventory?.has_fertilizer;

  return (
    <div>
      {/* 访问横幅 */}
      <div className='farm-card' style={{
        marginBottom: 14, padding: '10px 16px',
        background: 'rgba(74,124,63,0.1)', border: '1px solid rgba(74,124,63,0.25)',
        display: 'flex', alignItems: 'center', gap: 10,
      }}>
        <button onClick={onBack} style={{
          background: 'none', border: 'none', cursor: 'pointer',
          color: 'var(--farm-text-2)', display: 'flex', alignItems: 'center', gap: 4, fontSize: 13,
        }}>
          <ArrowLeft size={15} /> 返回
        </button>
        <span style={{ flex: 1, fontSize: 14, fontWeight: 700, color: 'var(--farm-leaf)', textAlign: 'center' }}>
          🌾 正在访问 {friendName} 的农场
        </span>
        <Button size='small' icon={<RefreshCw size={12} />} theme='borderless'
          onClick={loadAll} loading={loading} className='farm-btn'
          style={{ color: 'var(--farm-text-3)' }} />
      </div>

      {/* 农场概况 */}
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 14 }}>
        <div className='farm-stat-card'>
          <span style={{ fontSize: 20 }}>🌾</span>
          <div><div style={{ fontSize: 11, color: 'var(--farm-text-2)' }}>地块</div>
            <div style={{ fontSize: 15, fontWeight: 700 }}>{farmData?.plot_count} 块</div></div>
        </div>
        {matureCount > 0 && (
          <div className='farm-stat-card'>
            <span style={{ fontSize: 20 }}>✅</span>
            <div><div style={{ fontSize: 11, color: 'var(--farm-text-2)' }}>可收获</div>
              <div style={{ fontSize: 15, fontWeight: 700, color: 'var(--farm-leaf)' }}>{matureCount}</div></div>
          </div>
        )}
        {eventCount > 0 && (
          <div className='farm-stat-card'>
            <span style={{ fontSize: 20 }}>⚠️</span>
            <div><div style={{ fontSize: 11, color: 'var(--farm-text-2)' }}>需治疗</div>
              <div style={{ fontSize: 15, fontWeight: 700, color: 'var(--farm-danger)' }}>{eventCount}</div></div>
          </div>
        )}
        <div className='farm-stat-card'>
          <span style={{ fontSize: 20 }}>📊</span>
          <div><div style={{ fontSize: 11, color: 'var(--farm-text-2)' }}>农场等级</div>
            <div style={{ fontSize: 15, fontWeight: 700 }}>Lv.{farmData?.user_level || 1}</div></div>
        </div>
      </div>

      {/* 帮助操作面板 */}
      <div className='farm-card' style={{ marginBottom: 14 }}>
        <div className='farm-section-title' style={{ marginBottom: 10 }}>
          🤝 帮助好友操作
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 8, marginBottom: 10 }}>
          <Button size='small' icon={<Wheat size={13} />}
            disabled={matureCount === 0 || actionLoading} loading={actionLoading}
            theme='solid'
            style={{ background: matureCount > 0 ? 'linear-gradient(135deg,var(--farm-harvest),var(--farm-soil))' : undefined, borderRadius: 6 }}
            onClick={() => doAction('/harvest')} className='farm-btn'>
            🌾 帮助收获{matureCount > 0 ? ` (${matureCount})` : ''}
          </Button>

          <Button size='small' icon={<Droplets size={13} />}
            disabled={needsWaterCount === 0 || actionLoading} loading={actionLoading}
            style={{ background: 'rgba(138, 128, 106, 0.18)', border: '1px solid rgba(138, 128, 106, 0.34)', color: 'var(--farm-text-0)', borderRadius: 6 }}
            onClick={() => doAction('/water')} className='farm-btn'>
            💧 帮助浇水{needsWaterCount > 0 ? ` (${needsWaterCount})` : ''} <span style={{ fontSize: 10, opacity: 0.7 }}>免费</span>
          </Button>

          <Button size='small' icon={<FlaskConical size={13} />}
            disabled={!canFertilize || actionLoading} loading={actionLoading}
            style={{ background: 'rgba(111, 122, 99, 0.18)', border: '1px solid rgba(111, 122, 99, 0.34)', color: 'var(--farm-text-0)', borderRadius: 6 }}
            onClick={() => doAction('/fertilize')} className='farm-btn'>
            🧴 帮助施肥 <span style={{ fontSize: 10, opacity: 0.7 }}>用我的化肥</span>
          </Button>

          <Button size='small' icon={<Pill size={13} />}
            disabled={eventCount === 0 || !inventory?.has_medicine || actionLoading} loading={actionLoading}
            style={{ background: 'rgba(181, 51, 51, 0.12)', border: '1px solid rgba(181, 51, 51, 0.24)', color: 'var(--farm-danger)', borderRadius: 6 }}
            onClick={() => doAction('/treat')} className='farm-btn'>
            💊 帮助治疗{eventCount > 0 ? ` (${eventCount})` : ''} <span style={{ fontSize: 10, opacity: 0.7 }}>用我的药</span>
          </Button>
        </div>

        {/* 种植行 */}
        {emptyCount > 0 && (
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <Select size='small' placeholder='选择要帮种的作物'
              value={plantCrop || undefined} onChange={v => setPlantCrop(v)}
              style={{ flex: 1, maxWidth: 200 }} showClear>
              {/* 优先显示访客有种子的作物 */}
              {inventory?.seeds?.length > 0 && inventory.seeds.map(s => (
                <Select.Option key={s.key} value={s.key}>
                  {s.emoji} {s.name} ×{s.quantity}（库存种子）
                </Select.Option>
              ))}
              {crops.filter(c => !inventory?.seeds?.some(s => s.key === c.key)).map(c => (
                <Select.Option key={c.key} value={c.key}>
                  {c.emoji} {c.name}（从余额购买）
                </Select.Option>
              ))}
            </Select>
            <Button size='small' icon={<Sprout size={13} />}
              disabled={!plantCrop || actionLoading} loading={actionLoading}
              theme='solid'
              style={{ background: plantCrop ? 'linear-gradient(135deg,var(--farm-leaf),#2d6a2e)' : undefined, borderRadius: 6 }}
              onClick={() => doAction('/plant', { crop_key: plantCrop })} className='farm-btn'>
              🌱 帮种 ({emptyCount} 块空地)
            </Button>
          </div>
        )}

        {/* 访客道具提示 */}
        <div style={{ marginTop: 10, padding: '6px 10px', borderRadius: 8,
          background: 'rgba(255,255,255,0.04)', fontSize: 11, color: 'var(--farm-text-3)' }}>
          💡 浇水免费 · 施肥/种植/治疗消耗你自己的道具 · 收益全归农场主
        </div>
      </div>

      {/* 地块网格 */}
      <div className='farm-plot-grid'>
        {plots.map(plot => (
          <VisitPlotCard key={plot.plot_index} plot={plot} />
        ))}
      </div>
    </div>
  );
};

export default VisitFarmPage;
