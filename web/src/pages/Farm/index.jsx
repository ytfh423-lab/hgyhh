import React, { useCallback, useEffect, useState, useContext } from 'react';
import { useTranslation } from 'react-i18next';
import { Spin, Typography, Button } from '@douyinfe/semi-ui';
import { Sprout, Lock, Clock, ShieldAlert } from 'lucide-react';
import { API, showError, showSuccess } from '../../helpers';
import { StatusContext } from '../../context/Status';
import { Link } from 'react-router-dom';
import './farm.css';

import Sidebar, { navGroups } from './components/Sidebar';
import StatusBar from './components/StatusBar';
import FarmOverview from './components/FarmOverview';
import PlantPage from './components/PlantPage';
import RanchPage from './components/RanchPage';
import FishPage from './components/FishPage';
import WorkshopPage from './components/WorkshopPage';
import MarketPage from './components/MarketPage';
import ShopPage from './components/ShopPage';
import WarehousePage from './components/WarehousePage';
import TradingPage from './components/TradingPage';
import BankPage from './components/BankPage';
import LevelPage from './components/LevelPage';
import TasksPage from './components/TasksPage';
import AchievementsPage from './components/AchievementsPage';
import EncyclopediaPage from './components/EncyclopediaPage';
import LeaderboardPage from './components/LeaderboardPage';
import StealPage from './components/StealPage';
import GamesPage from './components/GamesPage';
import DogPage from './components/DogPage';
import AutomationPage from './components/AutomationPage';
import PrestigePage from './components/PrestigePage';
import LogsPage from './components/LogsPage';
import FarmAnnouncementBar from './components/FarmAnnouncementBar';

const { Text, Title } = Typography;

class FarmErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null };
  }
  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }
  componentDidCatch(error, info) {
    console.error('[Farm] Page crash:', error, info);
  }
  componentDidUpdate(prevProps) {
    if (prevProps.resetKey !== this.props.resetKey) {
      this.setState({ hasError: false, error: null });
    }
  }
  render() {
    if (this.state.hasError) {
      return (
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          minHeight: 'calc(100vh - 120px)', padding: 24,
        }}>
          <div style={{
            textAlign: 'center', padding: '48px 36px', maxWidth: 420, width: '100%',
            background: 'rgba(15, 23, 20, 0.85)', backdropFilter: 'blur(16px)',
            border: '1px solid rgba(239, 68, 68, 0.3)', borderRadius: 16,
            boxShadow: '0 0 30px rgba(239, 68, 68, 0.08), 0 8px 32px rgba(0,0,0,0.3)',
          }}>
            <div style={{ fontSize: 48, marginBottom: 16 }}>⚠️</div>
            <h3 style={{ margin: '0 0 8px', fontSize: 18, fontWeight: 700, color: '#f1f5f9' }}>
              页面加载出错
            </h3>
            <p style={{ margin: '0 0 16px', color: '#94a3b8', fontSize: 13 }}>
              {this.state.error?.message || '未知错误'}
            </p>
            <button
              onClick={() => this.setState({ hasError: false, error: null })}
              style={{
                padding: '8px 24px', borderRadius: 8, border: '1px solid rgba(59,130,246,0.3)',
                background: 'rgba(59,130,246,0.15)', color: '#60a5fa',
                fontWeight: 600, fontSize: 13, cursor: 'pointer',
              }}
            >
              重试
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}

const seasonCssVar = { 0: 'var(--farm-spring)', 1: 'var(--farm-summer)', 2: 'var(--farm-autumn)', 3: 'var(--farm-winter)' };

const mobileQuickTabs = [
  { key: 'overview', emoji: '🏠', label: '总览' },
  { key: 'plant', emoji: '🌱', label: '种植' },
  { key: 'ranch', emoji: '🐄', label: '牧场' },
  { key: 'market', emoji: '📈', label: '市场' },
  { key: 'more', emoji: '☰', label: '更多' },
];

const MobileBottomNav = ({ activeKey, onNavigate, showSheet, t }) => {
  return (
    <div className='farm-mobile-nav'>
      {mobileQuickTabs.map((tab) => (
        <div
          key={tab.key}
          className={`farm-mobile-nav-item ${activeKey === tab.key || (tab.key === 'more' && !mobileQuickTabs.slice(0, 4).some(q => q.key === activeKey)) ? 'active' : ''}`}
          onClick={() => tab.key === 'more' ? showSheet() : onNavigate(tab.key)}
        >
          <span className='nav-emoji'>{tab.emoji}</span>
          <span className='nav-label'>{t(tab.label)}</span>
        </div>
      ))}
    </div>
  );
};

const MobileSheet = ({ activeKey, onNavigate, onClose, t }) => {
  return (
    <div className='farm-mobile-sheet-overlay' onClick={onClose}>
      <div className='farm-mobile-sheet' onClick={(e) => e.stopPropagation()}>
        <div className='farm-mobile-sheet-handle' />
        {navGroups.map((group) => (
          <div key={group.key} className='farm-mobile-sheet-group'>
            <div className='farm-mobile-sheet-group-title'>{group.emoji} {t(group.label)}</div>
            <div className='farm-mobile-sheet-grid'>
              {group.items.map((item) => (
                <div
                  key={item.key}
                  className={`farm-mobile-sheet-item ${activeKey === item.key ? 'active' : ''}`}
                  onClick={() => { onNavigate(item.key); onClose(); }}
                >
                  <span className='sheet-emoji'>{item.emoji}</span>
                  <span>{t(item.label)}</span>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

const Farm = () => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const [loading, setLoading] = useState(true);
  const [farmData, setFarmData] = useState(null);
  const [crops, setCrops] = useState([]);
  const [actionLoading, setActionLoading] = useState(false);
  const [activePage, setActivePage] = useState('overview');
  const [mobileSheetOpen, setMobileSheetOpen] = useState(false);
  const [betaGate, setBetaGate] = useState(null); // null | 'BETA_NOT_STARTED' | 'BETA_NO_ACCESS'
  const [betaMessage, setBetaMessage] = useState('');

  const loadFarm = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/view');
      if (res.success) {
        setFarmData(res.data);
        setBetaGate(null);
      } else if (res.code === 'BETA_NOT_STARTED' || res.code === 'BETA_NO_ACCESS') {
        setBetaGate(res.code);
        setBetaMessage(res.message);
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const loadCrops = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/crops');
      if (res.success) setCrops(res.data || []);
    } catch (err) { /* ignore */ }
  }, []);

  useEffect(() => {
    loadFarm();
    loadCrops();
  }, [loadFarm, loadCrops]);

  useEffect(() => {
    const interval = setInterval(loadFarm, 30000);
    return () => clearInterval(interval);
  }, [loadFarm]);

  const doAction = async (url, body) => {
    setActionLoading(true);
    try {
      const { data: res } = await API.post(url, body);
      if (res.success) {
        showSuccess(res.message || t('操作成功'));
        loadFarm();
        return res;
      } else {
        showError(res.message);
        return null;
      }
    } catch (err) {
      showError(t('操作失败'));
      return null;
    } finally {
      setActionLoading(false);
    }
  };

  if (loading && !farmData && !betaGate) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <Spin size='large' />
      </div>
    );
  }

  if (betaGate) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#0a0a0a' }}>
        <div style={{
          textAlign: 'center', padding: '48px 32px', maxWidth: 440,
          background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(251,191,36,0.12)',
          borderRadius: 16, backdropFilter: 'blur(16px)',
        }}>
          {betaGate === 'BETA_NOT_STARTED' ? (
            <>
              <Clock size={44} style={{ color: '#fbbf24', marginBottom: 16 }} />
              <h2 style={{ color: '#fde68a', fontSize: 22, fontWeight: 700, marginBottom: 8 }}>
                {t('农场内测尚未开启')}
              </h2>
              <p style={{ color: '#a8a29e', fontSize: 14, lineHeight: 1.6, marginBottom: 24 }}>
                {t('内测倒计时正在进行中，请返回首页查看倒计时并预约内测资格。')}
              </p>
              <Link to='/'>
                <button style={{
                  padding: '10px 28px', borderRadius: 8, border: '1px solid rgba(251,191,36,0.3)',
                  background: 'linear-gradient(135deg, #fbbf24, #d97706)', color: '#000',
                  fontWeight: 700, fontSize: 14, cursor: 'pointer',
                }}>
                  {t('返回首页')}
                </button>
              </Link>
            </>
          ) : (
            <>
              <Lock size={44} style={{ color: '#ef4444', marginBottom: 16 }} />
              <h2 style={{ color: '#fca5a5', fontSize: 22, fontWeight: 700, marginBottom: 8 }}>
                {t('暂无内测资格')}
              </h2>
              <p style={{ color: '#a8a29e', fontSize: 14, lineHeight: 1.6, marginBottom: 24 }}>
                {betaMessage || t('你没有内测资格，无法访问农场。请返回首页预约内测名额。')}
              </p>
              <Link to='/'>
                <button style={{
                  padding: '10px 28px', borderRadius: 8, border: '1px solid rgba(251,191,36,0.3)',
                  background: 'linear-gradient(135deg, #fbbf24, #d97706)', color: '#000',
                  fontWeight: 700, fontSize: 14, cursor: 'pointer',
                }}>
                  {t('返回首页')}
                </button>
              </Link>
            </>
          )}
        </div>
      </div>
    );
  }

  if (!farmData) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <div className='farm-card' style={{ textAlign: 'center', padding: 40, maxWidth: 400 }}>
          <Sprout size={36} style={{ color: 'var(--semi-color-text-3)', marginBottom: 12 }} />
          <Title heading={6}>{t('农场不可用')}</Title>
          <Text type='tertiary' size='small'>{t('请先绑定 Telegram 账号')}</Text>
        </div>
      </div>
    );
  }

  const currentSeason = farmData.weather?.season ?? 0;
  const commonProps = { farmData, loadFarm, actionLoading, doAction, t };

  // 功能解锁等级映射
  const featureLevelMap = {
    steal: { level: 2, name: '偷菜', emoji: '🕵️' },
    dog: { level: 2, name: '狗狗', emoji: '🐶' },
    market: { level: 2, name: '市场', emoji: '📈' },
    encyclopedia: { level: 2, name: '图鉴', emoji: '📖' },
    ranch: { level: 3, name: '牧场', emoji: '🐄' },
    fish: { level: 3, name: '钓鱼', emoji: '🎣' },
    bank: { level: 3, name: '银行', emoji: '🏦' },
    leaderboard: { level: 3, name: '排行榜', emoji: '🏅' },
    workshop: { level: 4, name: '加工坊', emoji: '🏭' },
    games: { level: 4, name: '小游戏', emoji: '🎰' },
    trading: { level: 5, name: '交易所', emoji: '🔄' },
    automation: { level: 6, name: '自动化', emoji: '⚡' },
  };

  const userLevel = farmData.user_level || 1;

  const renderPage = () => {
    const req = featureLevelMap[activePage];
    if (req && userLevel < req.level) {
      return (
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          minHeight: 'calc(100vh - 120px)', padding: 24,
        }}>
          <div style={{
            textAlign: 'center', padding: '48px 36px', maxWidth: 420, width: '100%',
            background: 'rgba(15, 23, 20, 0.85)', backdropFilter: 'blur(16px)',
            border: '1px solid rgba(245, 158, 11, 0.25)', borderRadius: 16,
            boxShadow: '0 0 30px rgba(245, 158, 11, 0.08), 0 8px 32px rgba(0,0,0,0.3)',
          }}>
            <div style={{
              width: 72, height: 72, margin: '0 auto 20px', borderRadius: '50%',
              background: 'rgba(245, 158, 11, 0.12)', border: '2px solid rgba(245, 158, 11, 0.3)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>
              <Lock size={32} style={{ color: '#fbbf24' }} />
            </div>
            <div style={{ fontSize: 36, marginBottom: 8 }}>{req.emoji}</div>
            <h3 style={{ margin: '0 0 8px', fontSize: 20, fontWeight: 700, color: '#f1f5f9' }}>
              {t(req.name)}
            </h3>
            <p style={{ margin: '0 0 20px', color: '#94a3b8', fontSize: 14, lineHeight: 1.6 }}>
              {t('需要等级')} <strong style={{ color: '#fbbf24' }}>Lv.{req.level}</strong> {t('才能解锁')}
            </p>
            <div style={{
              display: 'inline-flex', alignItems: 'center', gap: 6,
              padding: '6px 16px', borderRadius: 9999, fontSize: 13, fontWeight: 600,
              background: 'rgba(59, 130, 246, 0.15)', color: '#60a5fa',
              border: '1px solid rgba(59, 130, 246, 0.25)',
            }}>
              {t('当前等级')}: Lv.{userLevel}
            </div>
          </div>
        </div>
      );
    }

    switch (activePage) {
      case 'overview':
        return <FarmOverview {...commonProps} loading={loading} />;
      case 'plant':
        return <PlantPage {...commonProps} crops={crops} />;
      case 'ranch':
        return <RanchPage {...commonProps} />;
      case 'fish':
        return <FishPage {...commonProps} />;
      case 'workshop':
        return <WorkshopPage {...commonProps} />;
      case 'market':
        return <MarketPage t={t} />;
      case 'shop':
        return <ShopPage {...commonProps} />;
      case 'warehouse':
        return <WarehousePage {...commonProps} />;
      case 'trading':
        return <TradingPage {...commonProps} />;
      case 'bank':
        return <BankPage {...commonProps} />;
      case 'level':
        return <LevelPage actionLoading={actionLoading} loadFarm={loadFarm} t={t} />;
      case 'tasks':
        return <TasksPage actionLoading={actionLoading} loadFarm={loadFarm} t={t} />;
      case 'achievements':
        return <AchievementsPage actionLoading={actionLoading} loadFarm={loadFarm} t={t} />;
      case 'encyclopedia':
        return <EncyclopediaPage actionLoading={actionLoading} loadFarm={loadFarm} t={t} />;
      case 'leaderboard':
        return <LeaderboardPage t={t} />;
      case 'steal':
        return <StealPage {...commonProps} />;
      case 'games':
        return <GamesPage loadFarm={loadFarm} t={t} />;
      case 'dog':
        return <DogPage {...commonProps} />;
      case 'automation':
        return <AutomationPage loadFarm={loadFarm} t={t} />;
      case 'prestige':
        return <PrestigePage loadFarm={loadFarm} t={t} />;
      case 'logs':
        return <LogsPage t={t} />;
      default:
        return <FarmOverview {...commonProps} loading={loading} />;
    }
  };

  return (
    <div className='farm-layout'>
      <FarmAnnouncementBar t={t} />
      <Sidebar activeKey={activePage} onNavigate={setActivePage} t={t} farmData={farmData} />
      <div className='farm-main' style={{ background: seasonCssVar[currentSeason] || seasonCssVar[0] }}>
        <StatusBar farmData={farmData} t={t} />
        <div className='farm-content'>
          <div className='farm-content-inner'>
            <FarmErrorBoundary resetKey={activePage}>
              {renderPage()}
            </FarmErrorBoundary>
          </div>
        </div>
      </div>
      <MobileBottomNav
        activeKey={activePage}
        onNavigate={setActivePage}
        showSheet={() => setMobileSheetOpen(true)}
        t={t}
      />
      {mobileSheetOpen && (
        <MobileSheet
          activeKey={activePage}
          onNavigate={setActivePage}
          onClose={() => setMobileSheetOpen(false)}
          t={t}
        />
      )}
    </div>
  );
};

export default Farm;
