import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Empty, Tag, Banner, InputNumber, Typography } from '@douyinfe/semi-ui';
import { API, showError, formatBalance } from './utils';
import FarmConfirmModal from './FarmConfirmModal';

const { Text } = Typography;

const BankPage = ({ farmData, actionLoading, doAction, loadFarm, t }) => {
  const [bankData, setBankData] = useState(null);
  const [bankLoading, setBankLoading] = useState(true);
  const [mortgageAmount, setMortgageAmount] = useState(100);
  const [confirmState, setConfirmState] = useState({ visible: false, title: '', message: '', action: null, type: 'warning' });

  const loadBank = useCallback(async () => {
    setBankLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/bank');
      if (res.success) setBankData(res.data);
      else showError(res.message);
    } catch (err) { /* ignore */ }
    finally { setBankLoading(false); }
  }, []);

  useEffect(() => { loadBank(); }, [loadBank]);

  const showConfirm = (title, message, action, type = 'warning') => {
    setConfirmState({ visible: true, title, message, action, type });
  };
  const closeConfirm = () => setConfirmState(s => ({ ...s, visible: false }));
  const executeConfirm = async () => {
    if (confirmState.action) await confirmState.action();
    closeConfirm();
  };

  const handleLoan = () => {
    if (!bankData) return;
    showConfirm(
      '💵 ' + t('确认申请信用贷款？'),
      <div>
        <div style={{ marginBottom: 8 }}>
          <span style={{ fontWeight: 600 }}>{t('贷款金额')}: </span>{formatBalance(bankData.max_loan)}
        </div>
        <div style={{ marginBottom: 8 }}>
          <span style={{ fontWeight: 600 }}>{t('利息')}: </span>{formatBalance(bankData.interest)} ({bankData.interest_rate}%)
        </div>
        <div style={{ marginBottom: 8 }}>
          <span style={{ fontWeight: 600 }}>{t('还款期限')}: </span>{bankData.loan_days} {t('天')}
        </div>
        <div style={{ color: 'var(--farm-warning)', fontSize: 12 }}>
          ⚠️ {t('逾期未还将影响信用评分')}
        </div>
      </div>,
      async () => {
        const res = await doAction('/api/farm/bank/loan', {});
        if (res) { loadBank(); loadFarm(); }
      },
      'primary'
    );
  };

  const mortgageMaxDollar = bankData ? Math.floor(bankData.mortgage_max) : 1000;

  const handleMortgage = () => {
    if (!mortgageAmount || mortgageAmount < 1 || mortgageAmount > mortgageMaxDollar) {
      showError(t('金额必须在 $1 ~ $') + mortgageMaxDollar + t(' 之间'));
      return;
    }
    const interestAmt = (mortgageAmount * (bankData?.mortgage_interest_rate || 15) / 100).toFixed(2);
    const totalDue = (mortgageAmount * (1 + (bankData?.mortgage_interest_rate || 15) / 100)).toFixed(2);
    showConfirm(
      '🏠 ' + t('确认申请抵押贷款？'),
      <div>
        <div style={{ marginBottom: 8 }}>
          <span style={{ fontWeight: 600 }}>{t('贷款金额')}: </span>${mortgageAmount}
        </div>
        <div style={{ marginBottom: 8 }}>
          <span style={{ fontWeight: 600 }}>{t('利息')}: </span>${interestAmt} ({bankData?.mortgage_interest_rate}%)
        </div>
        <div style={{ marginBottom: 8 }}>
          <span style={{ fontWeight: 600 }}>{t('应还总额')}: </span>${totalDue}
        </div>
        <div style={{ color: 'var(--farm-danger)', fontSize: 12, lineHeight: 1.5 }}>
          ⚠️ {t('抵押贷款不能用于升级！逾期未还：10级以下永久禁升10级，10级以上封禁账号')}
        </div>
      </div>,
      async () => {
        const res = await doAction('/api/farm/bank/mortgage', { amount: mortgageAmount });
        if (res) { loadBank(); loadFarm(); }
      },
      'danger'
    );
  };

  const handleRepay = (percent) => {
    if (!bankData?.active_loan) return;
    const loan = bankData.active_loan;
    const repayAmt = percent === 100 ? loan.remaining : loan.remaining / 2;
    const extendInfo = percent === 50 ? `\n📅 ${t('还一半可延长期限2天')}` : '';
    showConfirm(
      '💰 ' + t('确认还款？'),
      <div>
        <div style={{ marginBottom: 8 }}>
          <span style={{ fontWeight: 600 }}>{t('还款比例')}: </span>{percent}%
        </div>
        <div style={{ marginBottom: 8 }}>
          <span style={{ fontWeight: 600 }}>{t('还款金额')}: </span>{formatBalance(repayAmt)}
        </div>
        <div style={{ marginBottom: 8 }}>
          <span style={{ fontWeight: 600 }}>{t('还后剩余')}: </span>{formatBalance(loan.remaining - repayAmt)}
        </div>
        {percent === 50 && loan.remaining - repayAmt > 0.01 && (
          <div style={{ color: 'var(--farm-primary)', fontSize: 12 }}>
            📅 {t('还一半可延长期限2天')}
          </div>
        )}
      </div>,
      async () => {
        const res = await doAction('/api/farm/bank/repay', { percent });
        if (res) { loadBank(); loadFarm(); }
      },
      'primary'
    );
  };

  if (bankLoading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin size='large' /></div>;
  }
  if (!bankData) return <Empty description={t('银行功能不可用')} />;

  const loan = bankData.active_loan;
  const history = bankData.history || [];

  return (
    <div>
      {bankData.mortgage_blocked && (
        <Banner type='danger' description={t('由于抵押贷款违约，你已被永久禁止升级到10级及以上等级')}
          style={{ marginBottom: 14, borderRadius: 12 }} />
      )}

      <div className='farm-card'>
        <div className='farm-section-title'>🏦 {t('银行信息')}</div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginBottom: 12 }}>
          <div className='farm-pill farm-pill-green'>💰 {t('余额')}: {formatBalance(bankData.balance)}</div>
          <div className='farm-pill farm-pill-cyan'>📊 {t('信用评分')}: {bankData.credit_score}/{bankData.max_score}</div>
        </div>
        <div className='farm-kv-grid'>
          <div className='farm-kv'>
            <div className='farm-kv-label'>{t('信用贷额度')}</div>
            <div className='farm-kv-value'>{formatBalance(bankData.max_loan)}</div>
          </div>
          <div className='farm-kv'>
            <div className='farm-kv-label'>{t('信用贷利率')}</div>
            <div className='farm-kv-value'>{bankData.interest_rate}%</div>
          </div>
          <div className='farm-kv'>
            <div className='farm-kv-label'>{t('抵押贷上限')}</div>
            <div className='farm-kv-value'>{formatBalance(bankData.mortgage_max)}</div>
          </div>
          <div className='farm-kv'>
            <div className='farm-kv-label'>{t('抵押贷利率')}</div>
            <div className='farm-kv-value'>{bankData.mortgage_interest_rate}%</div>
          </div>
          <div className='farm-kv'>
            <div className='farm-kv-label'>{t('还款期限')}</div>
            <div className='farm-kv-value'>{bankData.loan_days} {t('天')}</div>
          </div>
        </div>
      </div>

      <div className='farm-card'>
        {bankData.has_active_loan && loan ? (
          <div>
            <div className='farm-section-title'>
              📋 {t('当前贷款')} {loan.loan_type === 1 ? <Tag size='small' color='orange'>🏠 {t('抵押')}</Tag> : <Tag size='small' color='blue'>{t('信用')}</Tag>}
            </div>
            {loan.overdue && (
              <Banner type='danger' description={loan.loan_type === 1 ? t('抵押贷款已逾期！逾期将执行惩罚！') : t('贷款已逾期！请尽快还款')}
                style={{ marginBottom: 10, borderRadius: 10 }} />
            )}
            <div className='farm-kv-grid' style={{ marginBottom: 12 }}>
              <div className='farm-kv'>
                <div className='farm-kv-label'>{t('本金')}</div>
                <div className='farm-kv-value'>{formatBalance(loan.principal)}</div>
              </div>
              <div className='farm-kv'>
                <div className='farm-kv-label'>{t('利息')}</div>
                <div className='farm-kv-value'>{formatBalance(loan.interest)}</div>
              </div>
              <div className='farm-kv'>
                <div className='farm-kv-label'>{t('应还')}</div>
                <div className='farm-kv-value'>{formatBalance(loan.total_due)}</div>
              </div>
              <div className='farm-kv'>
                <div className='farm-kv-label'>{t('已还')}</div>
                <div className='farm-kv-value'>{formatBalance(loan.repaid)}</div>
              </div>
              <div className='farm-kv'>
                <div className='farm-kv-label'>{t('剩余')}</div>
                <div className='farm-kv-value' style={{ color: 'var(--farm-danger)' }}>{formatBalance(loan.remaining)}</div>
              </div>
              <div className='farm-kv'>
                <div className='farm-kv-label'>{t('剩余天数')}</div>
                <div className='farm-kv-value'>
                  {loan.overdue ? <Tag size='small' color='red'>{t('已逾期')}</Tag> : `${loan.days_left} ${t('天')}`}
                </div>
              </div>
            </div>
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
              <Button theme='solid' type='primary' onClick={() => handleRepay(100)}
                loading={actionLoading} className='farm-btn'>
                💰 {t('全额还款')} ({formatBalance(loan.remaining)})
              </Button>
              {loan.remaining > 0.01 && (
                <Button theme='light' type='primary' onClick={() => handleRepay(50)}
                  loading={actionLoading} className='farm-btn'>
                  💰 {t('还一半')} ({formatBalance(loan.remaining / 2)})
                </Button>
              )}
            </div>
          </div>
        ) : (
          <div>
            <div className='farm-section-title'>✅ {t('当前无贷款')}</div>

            <div className='farm-card-flat' style={{ marginBottom: 10, padding: '12px 14px' }}>
              <Text strong size='small' style={{ display: 'block', marginBottom: 6 }}>💵 {t('信用贷款')}</Text>
              <Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 8 }}>
                {t('额度')} {formatBalance(bankData.max_loan)}，{t('利息')} {formatBalance(bankData.interest)}，{t('期限')} {bankData.loan_days}{t('天')}
              </Text>
              <Button theme='solid' type='primary' onClick={handleLoan}
                loading={actionLoading} className='farm-btn'>
                💵 {t('申请信用贷款')} ({formatBalance(bankData.max_loan)})
              </Button>
            </div>

            <div className='farm-card-flat' style={{ padding: '12px 14px' }}>
              <Text strong size='small' style={{ display: 'block', marginBottom: 6 }}>🏠 {t('抵押贷款')}</Text>
              <Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 4 }}>
                {t('以10级升级权为抵押，最高')} {formatBalance(bankData.mortgage_max)}，{t('利率')} {bankData.mortgage_interest_rate}%
              </Text>
              <Banner type='warning' style={{ marginBottom: 8, borderRadius: 8 }}
                description={t('抵押贷款不能用于升级！逾期未还：10级以下永久禁升10级，10级以上封禁账号')} />
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                <Text size='small'>$</Text>
                <InputNumber value={mortgageAmount} onChange={setMortgageAmount}
                  min={1} max={mortgageMaxDollar} style={{ width: 120 }} />
                <Button theme='solid' type='warning' onClick={handleMortgage}
                  loading={actionLoading} className='farm-btn'>
                  🏠 {t('申请抵押贷款')}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>

      {history.length > 0 && (
        <div className='farm-card'>
          <div className='farm-section-title'>📜 {t('贷款历史')}</div>
          {history.map((h, idx) => (
            <div key={h.id || idx} className='farm-row'>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap', marginBottom: 2 }}>
                  {h.loan_type === 1
                    ? <Tag size='small' color='orange'>{t('抵押')}</Tag>
                    : <Tag size='small' color='blue'>{t('信用')}</Tag>}
                  {h.status === 1
                    ? <Tag size='small' color='green'>{t('已还清')}</Tag>
                    : h.status === 2
                      ? <Tag size='small' color='red'>{t('违约')}</Tag>
                      : <Tag size='small' color='orange'>{t('还款中')}</Tag>}
                  <Text type='tertiary' size='small'>{new Date(h.created_at * 1000).toLocaleDateString()}</Text>
                </div>
                <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
                  <Text size='small'>{t('本金')} {formatBalance(h.principal)}</Text>
                  <Text size='small'>{t('应还')} {formatBalance(h.total_due)}</Text>
                  <Text size='small'>{t('已还')} {formatBalance(h.repaid)}</Text>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
      <FarmConfirmModal
        visible={confirmState.visible}
        title={confirmState.title}
        message={confirmState.message}
        confirmType={confirmState.type}
        confirmText={t('确认')}
        cancelText={t('取消')}
        loading={actionLoading}
        onConfirm={executeConfirm}
        onCancel={closeConfirm}
      />
    </div>
  );
};

export default BankPage;
