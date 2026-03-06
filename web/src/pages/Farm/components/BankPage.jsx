import React, { useCallback, useEffect, useState } from 'react';
import { Button, Spin, Empty, Tag, Descriptions, Banner, InputNumber, Table, Typography } from '@douyinfe/semi-ui';
import { API, showError, formatBalance } from './utils';

const { Text } = Typography;

const BankPage = ({ farmData, actionLoading, doAction, loadFarm, t }) => {
  const [bankData, setBankData] = useState(null);
  const [bankLoading, setBankLoading] = useState(true);
  const [mortgageAmount, setMortgageAmount] = useState(100);

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

  const handleLoan = async () => {
    const res = await doAction('/api/farm/bank/loan', {});
    if (res) { loadBank(); loadFarm(); }
  };

  const mortgageMaxDollar = bankData ? Math.floor(bankData.mortgage_max) : 1000;

  const handleMortgage = async () => {
    if (!mortgageAmount || mortgageAmount < 1 || mortgageAmount > mortgageMaxDollar) {
      showError(t('金额必须在 $1 ~ $') + mortgageMaxDollar + t(' 之间'));
      return;
    }
    const res = await doAction('/api/farm/bank/mortgage', { amount: mortgageAmount });
    if (res) { loadBank(); loadFarm(); }
  };

  const handleRepay = async (percent) => {
    const res = await doAction('/api/farm/bank/repay', { percent });
    if (res) { loadBank(); loadFarm(); }
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

      {/* Bank info */}
      <div className='farm-card'>
        <div className='farm-section-title'>🏦 {t('银行信息')}</div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginBottom: 10 }}>
          <div className='farm-pill farm-pill-green'>💰 {t('余额')}: {formatBalance(bankData.balance)}</div>
          <div className='farm-pill farm-pill-cyan'>📊 {t('信用评分')}: {bankData.credit_score}/{bankData.max_score}</div>
        </div>
        <Descriptions size='small' row data={[
          { key: t('信用贷额度'), value: formatBalance(bankData.max_loan) },
          { key: t('信用贷利率'), value: `${bankData.interest_rate}%` },
          { key: t('抵押贷上限'), value: formatBalance(bankData.mortgage_max) },
          { key: t('抵押贷利率'), value: `${bankData.mortgage_interest_rate}%` },
          { key: t('还款期限'), value: `${bankData.loan_days} ${t('天')}` },
        ]} />
      </div>

      {/* Active loan or apply */}
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
            <Descriptions size='small' row data={[
              { key: t('本金'), value: formatBalance(loan.principal) },
              { key: t('利息'), value: formatBalance(loan.interest) },
              { key: t('应还'), value: formatBalance(loan.total_due) },
              { key: t('已还'), value: formatBalance(loan.repaid) },
              { key: t('剩余'), value: <Text type='danger' strong>{formatBalance(loan.remaining)}</Text> },
              { key: t('剩余天数'), value: loan.overdue ? <Tag color='red'>{t('已逾期')}</Tag> : `${loan.days_left} ${t('天')}` },
            ]} />
            <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
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
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
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

      {/* History */}
      {history.length > 0 && (
        <div className='farm-card'>
          <div className='farm-section-title'>📜 {t('贷款历史')}</div>
          <Table dataSource={history} pagination={false} size='small' columns={[
            { title: t('日期'), dataIndex: 'created_at', width: 100, render: v => new Date(v * 1000).toLocaleDateString() },
            { title: t('类型'), dataIndex: 'loan_type', width: 60, render: v => v === 1 ? <Tag size='small' color='orange'>{t('抵押')}</Tag> : <Tag size='small' color='blue'>{t('信用')}</Tag> },
            { title: t('本金'), dataIndex: 'principal', width: 90, render: v => formatBalance(v) },
            { title: t('应还'), dataIndex: 'total_due', width: 90, render: v => formatBalance(v) },
            { title: t('已还'), dataIndex: 'repaid', width: 90, render: v => formatBalance(v) },
            { title: t('状态'), dataIndex: 'status', width: 70, render: v => v === 1 ? <Tag size='small' color='green'>{t('已还清')}</Tag> : v === 2 ? <Tag size='small' color='red'>{t('违约')}</Tag> : <Tag size='small' color='orange'>{t('还款中')}</Tag> },
          ]} />
        </div>
      )}
    </div>
  );
};

export default BankPage;
