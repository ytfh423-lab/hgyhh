import React, { Suspense, lazy, useCallback, useEffect, useState, useContext, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Typography, Button } from '@douyinfe/semi-ui';
import { Sprout, Lock, Clock, ShieldAlert, ScrollText, CheckCircle, TimerOff } from 'lucide-react';
import { API, showError, showSuccess } from '../../helpers';
import { StatusContext } from '../../context/Status';
import { Link, useNavigate } from 'react-router-dom';
import Loading from '../../components/common/ui/Loading';
import './farm.css';

import Sidebar, { navGroups } from './components/Sidebar';
import StatusBar from './components/StatusBar';
import FarmOverview from './components/FarmOverview';
import BetaApplicationGate from './components/BetaApplicationGate';
import TutorialProvider from './components/TutorialProvider';
import tutorialEvents from './components/tutorialEvents';
import { FEATURE_LEVEL_MAP } from './constants';
const PlantPage = lazy(() => import('./components/PlantPage'));
const RanchPage = lazy(() => import('./components/RanchPage'));
const FishPage = lazy(() => import('./components/FishPage'));
const WorkshopPage = lazy(() => import('./components/WorkshopPage'));
const MarketPage = lazy(() => import('./components/MarketPage'));
const ShopPage = lazy(() => import('./components/ShopPage'));
const WarehousePage = lazy(() => import('./components/WarehousePage'));
const TradingPage = lazy(() => import('./components/TradingPage'));
const BankPage = lazy(() => import('./components/BankPage'));
const LevelPage = lazy(() => import('./components/LevelPage'));
const TasksPage = lazy(() => import('./components/TasksPage'));
const AchievementsPage = lazy(() => import('./components/AchievementsPage'));
const EncyclopediaPage = lazy(() => import('./components/EncyclopediaPage'));
const LeaderboardPage = lazy(() => import('./components/LeaderboardPage'));
const ProfilePage = lazy(() => import('./components/ProfilePage'));
const StealPage = lazy(() => import('./components/StealPage'));
const GamesPage = lazy(() => import('./components/GamesPage'));
const DogPage = lazy(() => import('./components/DogPage'));
const AutomationPage = lazy(() => import('./components/AutomationPage'));
const TreeFarmPage = lazy(() => import('./components/TreeFarmPage'));
const PrestigePage = lazy(() => import('./components/PrestigePage'));
const LogsPage = lazy(() => import('./components/LogsPage'));
const EntrustPage = lazy(() => import('./components/EntrustPage'));
const EntrustWorkPage = lazy(() => import('./components/EntrustWorkPage'));
const FriendListPage = lazy(() => import('./components/FriendListPage'));
const VisitFarmPage = lazy(() => import('./components/VisitFarmPage'));

const { Text, Title } = Typography;

/* ── Floating "Join Group" TG-style button ── */
const JoinGroupButton = ({ t }) => {
  const [config, setConfig] = useState(_groupConfigCache || null);
  useEffect(() => {
    if (_groupConfigCache !== undefined) return; // already fetched this session
    (async () => {
      try {
        const { data: res } = await API.get('/api/farm/group-config');
        const cfg = (res.success && res.data && res.data.enabled) ? res.data : null;
        _groupConfigCache = cfg;
        if (cfg) setConfig(cfg);
      } catch (e) { _groupConfigCache = null; }
    })();
  }, []);
  if (!config) return null;
  return (
    <a
      href={config.link}
      target='_blank'
      rel='noopener noreferrer'
      className='farm-join-group-btn'
      title={t('加入官方群组')}
    >
      <span className='tg-icon'>
        <svg viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg'>
          <path d='M9.78 18.65l.28-4.23 7.68-6.92c.34-.31-.07-.46-.52-.19L7.74 13.3 3.64 12c-.88-.25-.89-.86.2-1.3l15.97-6.16c.73-.33 1.43.18 1.15 1.3l-2.72 12.81c-.19.91-.74 1.13-1.5.71L12.6 16.3l-1.99 1.93c-.23.23-.42.42-.83.42z'/>
        </svg>
      </span>
      <span className='tg-label'>{t('加入群组')}</span>
    </a>
  );
};

const LockedPage = ({ feature, userLevel, onGoToLevel, t }) => (
  <div className='farm-locked-wrap'>
    <div className='farm-locked-card'>
      <div className='farm-locked-icon-ring'>🔒</div>
      <div className='farm-locked-emoji'>{feature.emoji}</div>
      <h3 className='farm-locked-title'>{t(feature.name)}</h3>
      <p className='farm-locked-desc'>
        {t('需要等级')} <strong>Lv.{feature.level}</strong> {t('才能解锁')}
      </p>
      <div className='farm-locked-level-badge'>
        {t('当前等级')}: Lv.{userLevel}
      </div>
      <br />
      <button className='farm-locked-btn' onClick={onGoToLevel}>
        {t('查看等级')}
      </button>
    </div>
  </div>
);

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
        <div className='farm-locked-wrap'>
          <div className='farm-locked-card'>
            <div className='farm-locked-icon-ring'>⚠️</div>
            <h3 className='farm-locked-title'>页面加载出错</h3>
            <p className='farm-locked-desc'>{String(this.state.error?.message || '未知错误')}</p>
            <button className='farm-locked-btn' onClick={() => this.setState({ hasError: false, error: null })}>
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

const MobileSheet = ({ activeKey, onNavigate, onClose, navigate, t, userLevel = 1 }) => {
  return (
    <div className='farm-mobile-sheet-overlay' onClick={onClose}>
      <div className='farm-mobile-sheet' onClick={(e) => e.stopPropagation()}>
        <div className='farm-mobile-sheet-handle' />
        {navGroups.map((group) => (
          <div key={group.key} className='farm-mobile-sheet-group'>
            <div className='farm-mobile-sheet-group-title'>{group.emoji} {t(group.label)}</div>
            <div className='farm-mobile-sheet-grid'>
              {group.items.map((item) => {
                const req = FEATURE_LEVEL_MAP[item.key];
                const locked = req && userLevel < req.level;
                return (
                  <div
                    key={item.key}
                    className={`farm-mobile-sheet-item ${activeKey === item.key ? 'active' : ''} ${locked ? 'locked' : ''}`}
                    onClick={locked ? undefined : () => { item.href ? navigate(item.href) : onNavigate(item.key); onClose(); }}
                  >
                    <span className='sheet-emoji'>{locked ? '🔒' : item.emoji}</span>
                    <span>{locked ? `${t(item.label)} (Lv.${req.level})` : t(item.label)}</span>
                  </div>
                );
              })}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// Module-level caches — survive page tab switches within same session
let _cropsCache = null;
let _groupConfigCache = undefined; // undefined = not fetched yet, null = fetched but disabled
const FARM_CACHE_KEY = 'farm_view_v2';

const Farm = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [statusState] = useContext(StatusContext);
  const [loading, setLoading] = useState(true);
  const [farmData, setFarmData] = useState(() => {
    // Stale-while-revalidate: show cached data immediately so there's no blank spinner
    try {
      const raw = sessionStorage.getItem(FARM_CACHE_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch (e) { return null; }
  });
  const [crops, setCrops] = useState(_cropsCache || []);
  const [actionLoading, setActionLoading] = useState(false);
  const [activePage, setActivePage] = useState('overview');
  const [mobileSheetOpen, setMobileSheetOpen] = useState(false);
  const [entrustWorkTaskId, setEntrustWorkTaskId] = useState(null);
  const [betaGate, setBetaGate] = useState(null); // null | 'BETA_NOT_STARTED' | 'BETA_NO_ACCESS' | 'BETA_AGREEMENT_REQUIRED' | 'BETA_EXPIRED'
  const [betaMessage, setBetaMessage] = useState('');
  const [agreementLoading, setAgreementLoading] = useState(false);
  const [agreementChecked, setAgreementChecked] = useState(false);
  const farmDataRef = useRef(farmData);
  const silentLoadFarmPromiseRef = useRef(null);
  const queuedSilentLoadFarmOptionsRef = useRef(null);

  useEffect(() => {
    farmDataRef.current = farmData;
  }, [farmData]);

  // 好友请求数（从 SocialPanel 事件同步，用于侧边栏角标）
  const [friendRequestCount, setFriendRequestCount] = useState(0);
  useEffect(() => {
    const poll = async () => {
      try {
        const { data: res } = await API.get('/api/social/friends/requests', { disableDuplicate: true });
        if (res.success) setFriendRequestCount((res.data || []).length);
      } catch { /* ignore */ }
    };
    const starter = setTimeout(poll, 1200);
    const t2 = setInterval(() => {
      if (!document.hidden) poll();
    }, 20000);
    return () => {
      clearTimeout(starter);
      clearInterval(t2);
    };
  }, []);

  // 打开聊天：派发全局事件给 SocialPanel
  const openChat = useCallback((friendId, friendName) => {
    window.dispatchEvent(new CustomEvent('social:open-chat', { detail: { friendId, friendName } }));
  }, []);

  // 访问好友农场状态
  const [visitFriend, setVisitFriend] = useState(null); // {id, name}

  useEffect(() => {
    const handler = (e) => {
      const { friendId, friendName } = e.detail || {};
      if (friendId) {
        setVisitFriend({ id: friendId, name: friendName });
        setActivePage('visit');
      }
    };
    window.addEventListener('farm:visit-friend', handler);
    return () => window.removeEventListener('farm:visit-friend', handler);
  }, []);

  const navigateTo = useCallback((page) => {
    setActivePage(page);
    if (page !== 'entrust') setEntrustWorkTaskId(null);
    if (page !== 'visit') setVisitFriend(null);
  }, []);

  const persistFarmData = useCallback((nextData) => {
    try { sessionStorage.setItem(FARM_CACHE_KEY, JSON.stringify(nextData)); } catch (e) {}
  }, []);

  const mergeFarmData = useCallback((partialData) => {
    const sanitizedPatch = Object.fromEntries(
      Object.entries(partialData || {}).filter(([, value]) => value !== undefined),
    );
    if (Object.keys(sanitizedPatch).length === 0) {
      return;
    }
    setFarmData((prev) => {
      const nextData = {
        ...(prev || {}),
        ...sanitizedPatch,
      };
      persistFarmData(nextData);
      farmDataRef.current = nextData;
      return nextData;
    });
  }, [persistFarmData]);

  const mergeLoadFarmOptions = useCallback((currentOptions, nextOptions) => ({
    silent: true,
    dynamicOnly: (currentOptions?.dynamicOnly ?? true) && (nextOptions?.dynamicOnly ?? true),
    patchData: {
      ...(currentOptions?.patchData || {}),
      ...(nextOptions?.patchData || {}),
    },
  }), []);

  const loadFarmDynamic = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/view/dynamic', { disableDuplicate: true });
      if (res.success) {
        mergeFarmData(res.data || { task_summary: { done: 0, total: 0, claimed: 0 } });
      }
    } catch (err) { /* ignore */ }
  }, [mergeFarmData]);

  const loadFarm = useCallback(async (options = {}) => {
    const silent = options.silent ?? !!farmDataRef.current;
    const dynamicOnly = options.dynamicOnly ?? false;
    const patchData = options.patchData || null;
    if (silent && silentLoadFarmPromiseRef.current) {
      queuedSilentLoadFarmOptionsRef.current = mergeLoadFarmOptions(queuedSilentLoadFarmOptionsRef.current, {
        silent,
        dynamicOnly,
        patchData,
      });
      if (patchData) {
        mergeFarmData(patchData);
      }
      return silentLoadFarmPromiseRef.current;
    }
    if (patchData) {
      mergeFarmData(patchData);
    }
    const request = (async () => {
      if (dynamicOnly) {
        await loadFarmDynamic();
        return;
      }
      if (!silent) {
        setLoading(true);
      }
      try {
        const [lightResp, dynamicResp] = await Promise.all([
          API.get('/api/farm/view/light'),
          API.get('/api/farm/view/dynamic', { disableDuplicate: true }),
        ]);
        const lightRes = lightResp.data;
        const dynamicRes = dynamicResp.data;
        if (lightRes.success) {
          const mergedData = {
            ...lightRes.data,
            ...(dynamicRes.success
              ? dynamicRes.data
              : { task_summary: farmDataRef.current?.task_summary || { done: 0, total: 0, claimed: 0 } }),
          };
          setFarmData(mergedData);
          farmDataRef.current = mergedData;
          setBetaGate(null);
          persistFarmData(mergedData);
        } else if (lightRes.code === 'BETA_NOT_STARTED' || lightRes.code === 'BETA_NO_ACCESS' || lightRes.code === 'BETA_AGREEMENT_REQUIRED' || lightRes.code === 'BETA_EXPIRED') {
          setBetaGate(lightRes.code);
          setBetaMessage(lightRes.message);
        } else {
          showError(lightRes.message);
        }
      } catch (err) {
        showError(t('加载失败'));
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    })();
    if (!silent) {
      return request;
    }
    silentLoadFarmPromiseRef.current = request.finally(() => {
      silentLoadFarmPromiseRef.current = null;
      const queuedOptions = queuedSilentLoadFarmOptionsRef.current;
      queuedSilentLoadFarmOptionsRef.current = null;
      if (queuedOptions) {
        setTimeout(() => {
          loadFarm(queuedOptions);
        }, 0);
      }
    });
    return silentLoadFarmPromiseRef.current;
  }, [loadFarmDynamic, mergeFarmData, mergeLoadFarmOptions, persistFarmData, t]);

  const loadCrops = useCallback(async () => {
    if (_cropsCache) { setCrops(_cropsCache); return; }
    try {
      const { data: res } = await API.get('/api/farm/crops');
      if (res.success) { _cropsCache = res.data || []; setCrops(_cropsCache); }
    } catch (err) { /* ignore */ }
  }, []);

  useEffect(() => {
    loadFarm({ silent: !!farmDataRef.current });
    if (typeof window.requestIdleCallback === 'function') {
      const idleId = window.requestIdleCallback(() => loadCrops(), { timeout: 1200 });
      return () => window.cancelIdleCallback(idleId);
    }
    const timer = setTimeout(() => { loadCrops(); }, 0);
    return () => clearTimeout(timer);
  }, [loadFarm, loadCrops]);

  useEffect(() => {
    const interval = setInterval(() => {
      if (!document.hidden) loadFarm({ dynamicOnly: true, silent: true });
    }, 30000);
    return () => clearInterval(interval);
  }, [loadFarm]);

  // 映射 API url → 教程事件名
  const urlToEvent = (url) => {
    if (url.includes('/farm/plant')) return 'plant-crop';
    if (url.includes('/farm/water')) return 'water-crop';
    if (url.includes('/farm/fertilize')) return 'fertilize-crop';
    if (url.includes('/farm/harvest/store')) return 'harvest-store';
    if (url.includes('/farm/harvest')) return 'harvest-crop';
    if (url.includes('/farm/warehouse/sellall')) return 'sell-item';
    if (url.includes('/farm/warehouse/sell')) return 'sell-item';
    if (url.includes('/tree/plant')) return 'plant-tree';
    if (url.includes('/tree/water')) return 'water-tree';
    if (url.includes('/tree/harvest')) return 'harvest-tree';
    if (url.includes('/tree/chop')) return 'chop-tree';
    if (url.includes('/farm/tasks/claim')) return 'claim-task';
    return null;
  };

  const doAction = async (url, body) => {
    setActionLoading(true);
    const eventName = urlToEvent(url);
    try {
      const { data: res } = await API.post(url, body);
      if (res.success) {
        showSuccess(res.message || t('操作成功'));
        loadFarm({ silent: true });
        if (eventName) tutorialEvents.emitSuccess(eventName, { ...body, response: res });
        return res;
      } else {
        showError(res.message);
        if (eventName) tutorialEvents.emitFail(eventName, { ...body, message: res.message });
        return null;
      }
    } catch (err) {
      showError(t('操作失败'));
      if (eventName) tutorialEvents.emitFail(eventName, { ...body, error: err.message });
      return null;
    } finally {
      setActionLoading(false);
    }
  };

  if (loading && !farmData && !betaGate) {
    return (
      <Loading size='large' text={t('农场加载中')} />
    );
  }

  const handleAcceptAgreement = async () => {
    if (!agreementChecked) return;
    setAgreementLoading(true);
    try {
      const { data: res } = await API.post('/api/farm/beta/accept-agreement');
      if (res.success) {
        setBetaGate(null);
        await loadFarm({ silent: false });
      } else {
        showError(res.message);
      }
    } catch (err) {
      showError(t('操作失败'));
    } finally {
      setAgreementLoading(false);
    }
  };

  if (betaGate) {
    // 内测协议页面
    if (betaGate === 'BETA_AGREEMENT_REQUIRED') {
      return (
        <div className='farm-agreement-wrap'>
          <div className='farm-agreement-card'>
            <div className='farm-agreement-icon'>
              <ScrollText size={48} strokeWidth={1.5} />
            </div>
            <h2 className='farm-agreement-title'>{t('内测参与协议')}</h2>
            <p className='farm-agreement-subtitle'>{t('欢迎参加农场内测！在开始之前，请仔细阅读以下协议内容。')}</p>

            <div className='farm-agreement-content'>
              <div className='farm-agreement-section'>
                <h3>📋 {t('协议条款')}</h3>
                <ol>
                  <li>
                    <strong>{t('内测周期')}</strong>
                    <p>{t('本次内测持续 2 周（14 天）。内测结束后，农场功能将暂时关闭，等待正式版本上线。')}</p>
                  </li>
                  <li>
                    <strong>{t('数据清除')}</strong>
                    <p>{t('内测期间产生的所有数据（包括但不限于：农场地块、作物、余额、等级、成就、仓库物品、牧场动物等）在内测结束后将全部清除，不会保留到正式上线版本。')}</p>
                  </li>
                  <li>
                    <strong>{t('内测目的')}</strong>
                    <p>{t('本次内测旨在测试游戏功能、平衡性和稳定性。您的参与和反馈将帮助我们优化正式版本。')}</p>
                  </li>
                  <li>
                    <strong>{t('功能变动')}</strong>
                    <p>{t('内测期间，游戏内容、数值、规则等可能随时调整，恕不另行通知。')}</p>
                  </li>
                  <li>
                    <strong>{t('免责声明')}</strong>
                    <p>{t('内测版本可能存在 Bug 或不稳定情况，由此造成的数据丢失或异常，我们将尽力修复但不做赔偿承诺。')}</p>
                  </li>
                </ol>
              </div>

              <div className='farm-agreement-highlight'>
                <strong>⚠️ {t('重要提醒')}</strong>
                <p>{t('内测期间的所有数据（余额、作物、等级等）将在内测结束后全部清零，不会保留到正式版本。请知悉并理解。')}</p>
              </div>
            </div>

            <label className='farm-agreement-checkbox' onClick={() => setAgreementChecked(!agreementChecked)}>
              <div className={`farm-agreement-check ${agreementChecked ? 'checked' : ''}`}>
                {agreementChecked && <CheckCircle size={18} />}
              </div>
              <span>{t('我已阅读并同意以上内测协议内容')}</span>
            </label>

            <button
              className={`farm-agreement-btn ${agreementChecked ? '' : 'disabled'}`}
              onClick={handleAcceptAgreement}
              disabled={!agreementChecked || agreementLoading}
            >
              {agreementLoading ? t('提交中...') : t('同意协议，进入农场')}
            </button>
          </div>
        </div>
      );
    }

    if (betaGate === 'BETA_EXPIRED') {
      return (
        <div className='farm-agreement-wrap'>
          <div className='farm-agreement-card' style={{ textAlign: 'center' }}>
            <div className='farm-agreement-icon'>
              <TimerOff size={48} strokeWidth={1.5} />
            </div>
            <h2 className='farm-agreement-title'>{t('内测已结束')}</h2>
            <p className='farm-agreement-subtitle' style={{ marginBottom: 20 }}>
              {t('感谢您参与农场内测！本次内测周期已结束。')}
            </p>
            <div className='farm-agreement-highlight' style={{ textAlign: 'left', marginTop: 0, marginBottom: 24 }}>
              <strong>📋 {t('清理说明')}</strong>
              <p>{t('根据内测协议，内测期间产生的所有数据（地块、作物、余额、等级、成就等）已全部清除，内测期间获得的额度已回收。')}</p>
            </div>
            <p style={{ color: 'var(--farm-text-2)', fontSize: 13, lineHeight: 1.7, marginBottom: 24 }}>
              {t('正式版本上线时，所有玩家将从零开始。届时欢迎回来体验完整版农场！')}
            </p>
            <Link to='/'>
              <button className='farm-agreement-btn'>
                {t('返回首页')}
              </button>
            </Link>
          </div>
        </div>
      );
    }

    if (betaGate === 'BETA_NO_ACCESS') {
      return <BetaApplicationGate />;
    }

    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#0a0a0a' }}>
        <div style={{
          textAlign: 'center', padding: '48px 32px', maxWidth: 440,
          background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(251,191,36,0.12)',
          borderRadius: 16,
        }}>
          <Clock size={44} style={{ color: 'var(--farm-harvest)', marginBottom: 16 }} />
          <h2 style={{ color: '#fde68a', fontSize: 22, fontWeight: 700, marginBottom: 8 }}>
            {t('农场内测尚未开启')}
          </h2>
          <p style={{ color: '#a8a29e', fontSize: 14, lineHeight: 1.6, marginBottom: 24 }}>
            {t('内测倒计时正在进行中，请返回首页查看倒计时并预约内测资格。')}
          </p>
          <Link to='/'>
            <button style={{
              padding: '10px 28px', borderRadius: 8, border: '1px solid rgba(251,191,36,0.3)',
              background: 'linear-gradient(135deg, var(--farm-harvest), var(--farm-soil))', color: '#fff',
              fontWeight: 700, fontSize: 14, cursor: 'pointer',
            }}>
              {t('返回首页')}
            </button>
          </Link>
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
  const userLevel = farmData.user_level || 1;

  const renderPage = () => {
    // 等级锁定检查
    const req = FEATURE_LEVEL_MAP[activePage];
    if (req && userLevel < req.level) {
      return (
        <LockedPage
          feature={req}
          userLevel={userLevel}
          onGoToLevel={() => navigateTo('level')}
          t={t}
        />
      );
    }

    switch (activePage) {
      case 'overview':
        return <FarmOverview {...commonProps} crops={crops} loading={loading} />;
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
        return <ShopPage {...commonProps} onNavigate={navigateTo} />;
      case 'warehouse':
        return <WarehousePage {...commonProps} />;
      case 'trading':
        return <TradingPage {...commonProps} />;
      case 'entrust':
        if (entrustWorkTaskId) {
          return <EntrustWorkPage taskId={entrustWorkTaskId} onBack={() => setEntrustWorkTaskId(null)} t={t} />;
        }
        return <EntrustPage {...commonProps} onEnterWork={(id) => setEntrustWorkTaskId(id)} />;
      case 'bank':
        return <BankPage {...commonProps} />;
      case 'profile':
        return <ProfilePage farmData={farmData} t={t} />;
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
      case 'treefarm':
        return <TreeFarmPage {...commonProps} />;
      case 'prestige':
        return <PrestigePage loadFarm={loadFarm} t={t} />;
      case 'logs':
        return <LogsPage t={t} />;
      case 'friends':
        return <FriendListPage onChatOpen={openChat} t={t} />;
      case 'visit':
        return visitFriend ? (
          <VisitFarmPage
            friendId={visitFriend.id}
            friendName={visitFriend.name}
            onBack={() => navigateTo('friends')}
            t={t}
          />
        ) : null;
      default:
        return <FarmOverview {...commonProps} crops={crops} loading={loading} />;
    }
  };

  return (
    <TutorialProvider userLevel={userLevel} activePage={activePage} onNavigate={navigateTo} farmData={farmData} loadFarm={loadFarm} t={t}>
      <div className='farm-layout'>
        <Sidebar activeKey={activePage} onNavigate={navigateTo} t={t} farmData={farmData} userLevel={userLevel} friendRequestCount={friendRequestCount} />
        <div className='farm-main' style={{ background: seasonCssVar[currentSeason] || seasonCssVar[0] }}>
          <StatusBar farmData={farmData} t={t} />
          <div className='farm-content'>
            <div key={activePage} className='farm-content-inner app-route-shell'>
              <FarmErrorBoundary resetKey={activePage}>
                <Suspense fallback={<Loading size='large' fullscreen={false} text={t('页面切换中')} />}>
                  {renderPage()}
                </Suspense>
              </FarmErrorBoundary>
            </div>
          </div>
        </div>
        <JoinGroupButton t={t} />
        <MobileBottomNav
          activeKey={activePage}
          onNavigate={navigateTo}
          showSheet={() => setMobileSheetOpen(true)}
          t={t}
        />
        {mobileSheetOpen && (
          <MobileSheet
            activeKey={activePage}
            onNavigate={navigateTo}
            onClose={() => setMobileSheetOpen(false)}
            navigate={navigate}
            t={t}
            userLevel={userLevel}
          />
        )}
      </div>
    </TutorialProvider>
  );
};

export default Farm;
