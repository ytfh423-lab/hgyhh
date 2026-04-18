/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import HeaderBar from './headerbar';
import { Layout } from '@douyinfe/semi-ui';
import SiderBar from './SiderBar';
import App from '../../App';
import FooterBar from './Footer';
import { ToastContainer } from 'react-toastify';
import React, { Suspense, lazy, useContext, useEffect, useState } from 'react';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useSidebarCollapsed } from '../../hooks/common/useSidebarCollapsed';
import { useTranslation } from 'react-i18next';
import {
  API,
  getLogo,
  getSystemName,
  showError,
  setStatusData,
} from '../../helpers';
import { loadRecaptchaV3Script } from '../../helpers/recaptcha';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { useLocation } from 'react-router-dom';
import { FarmConfirmProvider } from '../../pages/Farm/components/farmConfirm';
const { Sider, Content, Header } = Layout;
const SocialPanel = lazy(() => import('../social/SocialPanel'));

const PageLayout = () => {
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState, statusDispatch] = useContext(StatusContext);
  const isMobile = useIsMobile();
  const [collapsed, , setCollapsed] = useSidebarCollapsed();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [shouldLoadSocialPanel, setShouldLoadSocialPanel] = useState(false);
  const { i18n, t } = useTranslation();
  const location = useLocation();

  // reCAPTCHA 合规声明：启用 recaptcha 时固定在右下角显示（badge 已被 CSS 隐藏）
  const showRecaptchaNotice = (() => {
    const st = statusState?.status;
    if (!st) return false;
    const enabled = st.human_verification_enabled ?? st.turnstile_check ?? false;
    const provider = st.human_verification_provider || 'turnstile';
    return enabled && provider === 'recaptcha';
  })();

  const cardProPages = [
    '/console/channel',
    '/console/log',
    '/console/redemption',
    '/console/user',
    '/console/token',
    '/console/midjourney',
    '/console/task',
    '/console/models',
    '/pricing',
  ];

  const isFarmPage = location.pathname === '/farm';
  const shouldHideFooter = cardProPages.includes(location.pathname) || isFarmPage;

  const shouldInnerPadding =
    location.pathname.includes('/console') &&
    !location.pathname.startsWith('/console/chat') &&
    location.pathname !== '/console/playground';

  const isConsoleRoute = location.pathname.startsWith('/console');
  const showSider = isConsoleRoute && (!isMobile || drawerOpen);

  useEffect(() => {
    if (isMobile && drawerOpen && collapsed) {
      setCollapsed(false);
    }
  }, [isMobile, drawerOpen, collapsed, setCollapsed]);

  useEffect(() => {
    if (!userState?.user?.id) {
      setShouldLoadSocialPanel(false);
      return undefined;
    }
    let cancelled = false;
    const activate = () => {
      if (!cancelled) {
        setShouldLoadSocialPanel(true);
      }
    };
    if (typeof window !== 'undefined' && typeof window.requestIdleCallback === 'function') {
      const idleId = window.requestIdleCallback(activate, { timeout: 1500 });
      return () => {
        cancelled = true;
        window.cancelIdleCallback(idleId);
      };
    }
    const timer = window.setTimeout(activate, 600);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [userState?.user?.id]);

  const loadUser = () => {
    let user = localStorage.getItem('user');
    if (user) {
      let data = JSON.parse(user);
      userDispatch({ type: 'login', payload: data });
    }
  };

  const loadStatus = async () => {
    try {
      const res = await API.get('/api/status');
      const { success, data } = res.data;
      if (success) {
        statusDispatch({ type: 'set', payload: data });
        setStatusData(data);
        // 启用了 reCAPTCHA v3 时预加载脚本，让徽章提前出现 + 后续 step-up 秒开
        const enabled =
          data?.human_verification_enabled ?? data?.turnstile_check ?? false;
        const provider = data?.human_verification_provider || 'turnstile';
        const siteKey = data?.human_verification_site_key || '';
        if (enabled && provider === 'recaptcha' && siteKey) {
          loadRecaptchaV3Script(siteKey).catch(() => {});
        }
      } else {
        showError('Unable to connect to server');
      }
    } catch (error) {
      showError('Failed to load status');
    }
  };

  useEffect(() => {
    loadUser();
    loadStatus().catch(console.error);
    let systemName = getSystemName();
    if (systemName) {
      document.title = systemName;
    }
    let logo = getLogo();
    if (logo) {
      let linkElement = document.querySelector("link[rel~='icon']");
      if (linkElement) {
        linkElement.href = logo;
      }
    }
    const savedLang = localStorage.getItem('i18nextLng');
    if (savedLang) {
      i18n.changeLanguage(savedLang);
    }
  }, [i18n]);

  // 全站在线心跳（30秒一次，登录后才发）
  useEffect(() => {
    const sendHeartbeat = () => {
      try {
        const u = JSON.parse(localStorage.getItem('user') || '{}');
        if (!u.id) return;
        API.post('/api/heartbeat').catch(() => {});
      } catch { /* ignore */ }
    };
    sendHeartbeat();
    const timer = setInterval(sendHeartbeat, 30000);
    return () => clearInterval(timer);
  }, []);

  return (
    <Layout
      className='app-layout'
      style={{
        display: 'flex',
        flexDirection: 'column',
        overflow: isMobile ? 'visible' : 'hidden',
      }}
    >
      {!isFarmPage && (
        <Header
          style={{
            padding: 0,
            height: 'auto',
            lineHeight: 'normal',
            position: 'fixed',
            width: '100%',
            top: 0,
            zIndex: 100,
          }}
        >
          <HeaderBar
            onMobileMenuToggle={() => setDrawerOpen((prev) => !prev)}
            drawerOpen={drawerOpen}
          />
        </Header>
      )}
      <Layout
        style={{
          overflow: isMobile ? 'visible' : 'auto',
          display: 'flex',
          flexDirection: 'column',
          marginTop: isFarmPage ? '0' : '64px',
        }}
      >
        {showSider && (
          <Sider
            className='app-sider'
            style={{
              position: 'fixed',
              left: 0,
              top: '64px',
              zIndex: 99,
              border: 'none',
              paddingRight: '0',
              width: 'var(--sidebar-current-width)',
            }}
          >
            <SiderBar
              onNavigate={() => {
                if (isMobile) setDrawerOpen(false);
              }}
            />
          </Sider>
        )}
        <Layout
          style={{
            marginLeft: isMobile
              ? '0'
              : showSider
                ? 'var(--sidebar-current-width)'
                : '0',
            flex: '1 1 auto',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <Content
            style={{
              flex: '1 0 auto',
              overflowY: isMobile ? 'visible' : 'hidden',
              WebkitOverflowScrolling: 'touch',
              padding: shouldInnerPadding ? (isMobile ? '5px' : '24px') : '0',
              position: 'relative',
            }}
          >
            <div key={location.pathname + location.search} className='app-route-shell'>
              <App />
            </div>
          </Content>
          {!shouldHideFooter && (
            <Layout.Footer
              style={{
                flex: '0 0 auto',
                width: '100%',
              }}
            >
              <FooterBar />
            </Layout.Footer>
          )}
        </Layout>
      </Layout>
      <ToastContainer />
      {shouldLoadSocialPanel && (
        <Suspense fallback={null}>
          <SocialPanel />
        </Suspense>
      )}
      <FarmConfirmProvider />
      {showRecaptchaNotice && (
        <div className='recaptcha-compliance-bar'>
          {t('本站受')}{' '}
          <a
            href='https://policies.google.com/privacy'
            target='_blank'
            rel='noopener noreferrer'
          >
            reCAPTCHA
          </a>{' '}
          {t('保护')}（
          <a
            href='https://policies.google.com/privacy'
            target='_blank'
            rel='noopener noreferrer'
          >
            {t('隐私政策')}
          </a>
          {' · '}
          <a
            href='https://policies.google.com/terms'
            target='_blank'
            rel='noopener noreferrer'
          >
            {t('服务条款')}
          </a>
          ）
        </div>
      )}
    </Layout>
  );
};

export default PageLayout;
