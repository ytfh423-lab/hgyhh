import React, { useCallback, useEffect, useState } from 'react';
import { Button, Empty, Spin, Tag, InputNumber, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from './utils';

const { Text } = Typography;

const TradingPage = ({ actionLoading, doAction, loadFarm, t }) => {
  const [trades, setTrades] = useState([]);
  const [myTrades, setMyTrades] = useState([]);
  const [loading, setLoading] = useState(false);
  const [view, setView] = useState('market');
  const [whItems, setWhItems] = useState([]);
  const [sellForm, setSellForm] = useState({ crop_type: '', quantity: 1, price: 1 });

  const loadTrades = useCallback(async () => {
    setLoading(true);
    try {
      const [mktRes, histRes, whRes] = await Promise.all([
        API.get('/api/farm/trade'),
        API.get('/api/farm/trade/history'),
        API.get('/api/farm/warehouse'),
      ]);
      if (mktRes.data.success) setTrades(mktRes.data.data?.trades || []);
      if (histRes.data.success) setMyTrades(histRes.data.data || []);
      if (whRes.data.success) setWhItems(whRes.data.data?.items || []);
    } catch (err) { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadTrades(); }, [loadTrades]);

  const buyTrade = async (tradeId) => {
    try {
      const { data: res } = await API.post('/api/farm/trade/buy', { trade_id: tradeId });
      if (res.success) { showSuccess(res.message); loadTrades(); loadFarm({ silent: true }); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
  };

  const cancelTrade = async (tradeId) => {
    try {
      const { data: res } = await API.post('/api/farm/trade/cancel', { trade_id: tradeId });
      if (res.success) { showSuccess(res.message); loadTrades(); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
  };

  const createTrade = async () => {
    if (!sellForm.crop_type || sellForm.quantity < 1 || sellForm.price <= 0) {
      showError(t('请填写完整'));
      return;
    }
    try {
      const { data: res } = await API.post('/api/farm/trade/create', sellForm);
      if (res.success) { showSuccess(res.message); loadTrades(); loadFarm({ silent: true }); setSellForm({ crop_type: '', quantity: 1, price: 1 }); }
      else showError(res.message);
    } catch (err) { showError(t('操作失败')); }
  };

  if (loading && trades.length === 0) return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;

  return (
    <div>
      <div style={{ display: 'flex', gap: 6, marginBottom: 14, flexWrap: 'wrap' }}>
        {[
          { key: 'market', label: '🏪 ' + t('市场') },
          { key: 'sell', label: '📤 ' + t('挂单') },
          { key: 'history', label: '📜 ' + t('历史') },
        ].map(v => (
          <div key={v.key}
            className={`farm-pill ${view === v.key ? 'farm-pill-blue' : ''}`}
            style={{ cursor: 'pointer' }} onClick={() => setView(v.key)}>
            {v.label}
          </div>
        ))}
      </div>

      {view === 'market' && (
        <div className='farm-card'>
          {trades.length === 0 ? <Empty description={t('暂无挂单')} /> : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {trades.map(tr => (
                <div key={tr.id} className='farm-row'>
                  <span style={{ fontSize: 20 }}>{tr.item_emoji}</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <Text strong size='small'>{tr.item_name} ×{tr.quantity}</Text>
                    <Text type='tertiary' size='small' style={{ display: 'block' }}>
                      {t('卖家')}: {tr.seller_name} · ${tr.price_per_unit.toFixed(2)}/{t('个')}
                    </Text>
                  </div>
                  <div style={{ textAlign: 'right', flexShrink: 0 }}>
                    <Text strong style={{ color: 'var(--farm-harvest)' }}>${tr.total_price.toFixed(2)}</Text>
                    <Text type='tertiary' size='small' style={{ display: 'block' }}>+{tr.fee.toFixed(2)}{t('手续费')}</Text>
                  </div>
                  <Button size='small' theme='solid' onClick={() => buyTrade(tr.id)} className='farm-btn'>{t('购买')}</Button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {view === 'sell' && (
        <div className='farm-card'>
          <div className='farm-section-title'>📤 {t('从仓库挂单出售')}</div>
          {whItems.length === 0 ? <Empty description={t('仓库为空')} /> : (
            <div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 12 }}>
                {whItems.map(it => (
                  <div key={it.crop_key}
                    className={`farm-pill ${sellForm.crop_type === it.crop_key ? 'farm-pill-blue' : ''}`}
                    style={{ cursor: 'pointer' }}
                    onClick={() => setSellForm({ ...sellForm, crop_type: it.crop_key, quantity: Math.min(sellForm.quantity, it.quantity) })}>
                    {it.emoji} {it.crop_name} ×{it.quantity}
                  </div>
                ))}
              </div>
              {sellForm.crop_type && (
                <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
                  <Text size='small'>{t('数量')}:</Text>
                  <InputNumber value={sellForm.quantity} onChange={v => setSellForm({ ...sellForm, quantity: v })}
                    min={1} max={whItems.find(i => i.crop_key === sellForm.crop_type)?.quantity || 99} style={{ width: 80 }} />
                  <Text size='small'>{t('单价')} $:</Text>
                  <InputNumber value={sellForm.price} onChange={v => setSellForm({ ...sellForm, price: v })}
                    min={0.01} step={0.1} style={{ width: 100 }} />
                  <Button theme='solid' onClick={createTrade} className='farm-btn'>{t('挂单')}</Button>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {view === 'history' && (
        <div className='farm-card'>
          {myTrades.length === 0 ? <Empty description={t('暂无记录')} /> : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {myTrades.map((r, idx) => (
                <div key={r.id || idx} className='farm-row'>
                  <span style={{ fontSize: 18 }}>{r.item_emoji}</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                      <Text strong size='small'>{r.item_name} ×{r.quantity}</Text>
                      {r.is_seller
                        ? <Tag size='small' color='orange'>{t('卖出')}</Tag>
                        : <Tag size='small' color='green'>{t('买入')}</Tag>}
                      {r.status === 1
                        ? <Tag size='small' color='green'>{t('成交')}</Tag>
                        : <Tag size='small' color='grey'>{t('取消')}</Tag>}
                    </div>
                    <Text type='tertiary' size='small'>{t('单价')} ${r.price?.toFixed(2)}</Text>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default TradingPage;
