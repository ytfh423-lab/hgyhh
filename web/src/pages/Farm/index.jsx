import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Spin, Typography } from '@douyinfe/semi-ui';
import { Sprout } from 'lucide-react';
import { API, showError, showSuccess } from '../../helpers';
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

const { Text, Title } = Typography;

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
  const [loading, setLoading] = useState(true);
  const [farmData, setFarmData] = useState(null);
  const [crops, setCrops] = useState([]);
  const [actionLoading, setActionLoading] = useState(false);
  const [activePage, setActivePage] = useState('overview');
  const [mobileSheetOpen, setMobileSheetOpen] = useState(false);

  const loadFarm = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await API.get('/api/farm/view');
      if (res.success) {
        setFarmData(res.data);
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

  if (loading && !farmData) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <Spin size='large' />
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

  const renderPage = () => {
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
      <Sidebar activeKey={activePage} onNavigate={setActivePage} t={t} farmData={farmData} />
      <div className='farm-main' style={{ background: seasonCssVar[currentSeason] || seasonCssVar[0] }}>
        <StatusBar farmData={farmData} t={t} />
        <div className='farm-content'>
          <div className='farm-content-inner'>
            {renderPage()}
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
