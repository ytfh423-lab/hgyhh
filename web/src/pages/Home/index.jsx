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

import React, { useContext, useEffect, useState } from 'react';
import {
  Button,
  Typography,
  Input,
  ScrollList,
  ScrollItem,
} from '@douyinfe/semi-ui';
import { API, showError, copy, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import {
  IconGithubLogo,
  IconPlay,
  IconFile,
  IconCopy,
} from '@douyinfe/semi-icons';
import { Zap, Shield, Globe, Rocket } from 'lucide-react';
import { Link } from 'react-router-dom';
import NoticeModal from '../../components/layout/NoticeModal';
import {
  Moonshot,
  OpenAI,
  XAI,
  Zhipu,
  Volcengine,
  Cohere,
  Claude,
  Gemini,
  Suno,
  Minimax,
  Wenxin,
  Spark,
  Qingyan,
  DeepSeek,
  Qwen,
  Midjourney,
  Grok,
  AzureAI,
  Hunyuan,
  Xinference,
} from '@lobehub/icons';

const { Text } = Typography;

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const docsLink = statusState?.status?.docs_link || '';
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const endpointItems = API_ENDPOINTS.map((e) => ({ value: e }));
  const [endpointIndex, setEndpointIndex] = useState(0);
  const isChinese = i18n.language.startsWith('zh');

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    const res = await API.get('/api/home_page_content');
    const { success, message, data } = res.data;
    if (success) {
      let content = data;
      if (!data.startsWith('https://')) {
        content = marked.parse(data);
      }
      setHomePageContent(content);
      localStorage.setItem('home_page_content', content);

      // 如果内容是 URL，则发送主题模式
      if (data.startsWith('https://')) {
        const iframe = document.querySelector('iframe');
        if (iframe) {
          iframe.onload = () => {
            iframe.contentWindow.postMessage({ themeMode: actualTheme }, '*');
            iframe.contentWindow.postMessage({ lang: i18n.language }, '*');
          };
        }
      }
    } else {
      showError(message);
      setHomePageContent('加载首页内容失败...');
    }
    setHomePageContentLoaded(true);
  };

  const handleCopyBaseURL = async () => {
    const ok = await copy(serverAddress);
    if (ok) {
      showSuccess(t('已复制到剪切板'));
    }
  };

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice');
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            setNoticeVisible(true);
          }
        } catch (error) {
          console.error('获取公告失败:', error);
        }
      }
    };

    checkNoticeAndShow();
  }, []);

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  useEffect(() => {
    const timer = setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % endpointItems.length);
    }, 3000);
    return () => clearInterval(timer);
  }, [endpointItems.length]);

  return (
    <div className='w-full overflow-x-hidden'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      {homePageContentLoaded && homePageContent === '' ? (
        <div className='w-full overflow-x-hidden'>
          {/* ===== Hero 区域 ===== */}
          <div className='w-full relative overflow-hidden'>
            {/* Mesh 渐变背景 */}
            <div className='npc-hero-bg'>
              <div className='npc-mesh-blob npc-mesh-1' />
              <div className='npc-mesh-blob npc-mesh-2' />
              <div className='npc-mesh-blob npc-mesh-3' />
            </div>

            <div className='flex flex-col items-center justify-center text-center px-4 pt-24 pb-16 md:pt-32 md:pb-20 lg:pt-40 lg:pb-24 relative z-10'>
              {/* 状态徽标 */}
              <div className='npc-animate npc-delay-1 mb-8'>
                <div className='npc-status-badge'>
                  <div className='npc-status-dot' />
                  <span>{t('所有服务运行正常')}</span>
                </div>
              </div>

              {/* 品牌大标题 */}
              <div className='npc-animate npc-delay-2 mb-6'>
                <h1 className='npc-brand-title'>NPC-API</h1>
              </div>

              {/* 副标题 */}
              <div className='npc-animate npc-delay-3 mb-10'>
                <p className='npc-subtitle' style={{ margin: '0 auto' }}>
                  {t('一站式 AI 模型接口聚合平台，更快、更稳、更省')}
                </p>
              </div>

              {/* URL 区域 */}
              <div className='npc-animate npc-delay-3 w-full max-w-lg mb-10'>
                <div className='npc-url-container'>
                  <div className='npc-url-inner'>
                    <Input
                      readonly
                      value={serverAddress}
                      className='flex-1 !rounded-full'
                      size={isMobile ? 'default' : 'large'}
                      suffix={
                        <div className='flex items-center gap-2'>
                          <ScrollList
                            bodyHeight={32}
                            style={{ border: 'unset', boxShadow: 'unset' }}
                          >
                            <ScrollItem
                              mode='wheel'
                              cycled={true}
                              list={endpointItems}
                              selectedIndex={endpointIndex}
                              onSelect={({ index }) => setEndpointIndex(index)}
                            />
                          </ScrollList>
                          <Button
                            type='primary'
                            onClick={handleCopyBaseURL}
                            icon={<IconCopy />}
                            className='!rounded-full'
                          />
                        </div>
                      }
                    />
                  </div>
                </div>
              </div>

              {/* 操作按钮 */}
              <div className='npc-animate npc-delay-4 flex flex-row gap-4 justify-center items-center'>
                <Link to='/console'>
                  <Button
                    theme='solid'
                    type='primary'
                    size={isMobile ? 'default' : 'large'}
                    className='npc-btn-primary'
                    icon={<Rocket size={18} />}
                  >
                    {t('开始使用')}
                  </Button>
                </Link>
                {isDemoSiteMode && statusState?.status?.version ? (
                  <Button
                    size={isMobile ? 'default' : 'large'}
                    className='npc-btn-secondary'
                    icon={<IconGithubLogo />}
                    onClick={() =>
                      window.open(
                        'https://github.com/QuantumNous/new-api',
                        '_blank',
                      )
                    }
                  >
                    {statusState.status.version}
                  </Button>
                ) : (
                  docsLink && (
                    <Button
                      size={isMobile ? 'default' : 'large'}
                      className='npc-btn-secondary'
                      icon={<IconFile />}
                      onClick={() => window.open(docsLink, '_blank')}
                    >
                      {t('查看文档')}
                    </Button>
                  )
                )}
              </div>
            </div>
          </div>

          {/* ===== 特性卡片区 ===== */}
          <div className='w-full px-4 py-16 md:py-20 relative z-10'>
            <div className='npc-animate npc-delay-5 max-w-4xl mx-auto'>
              <div className='grid grid-cols-1 md:grid-cols-3 gap-5'>
                <div className='npc-feature-card'>
                  <div className='npc-feature-icon' style={{ background: 'rgba(99, 102, 241, 0.1)' }}>
                    <Zap size={24} style={{ color: '#6366f1' }} />
                  </div>
                  <Typography.Title heading={5} style={{ marginBottom: '8px' }}>
                    {t('极速响应')}
                  </Typography.Title>
                  <Text style={{ color: 'var(--semi-color-text-2)', fontSize: '14px', lineHeight: 1.6 }}>
                    {t('全球节点智能路由，毫秒级转发，让每一次调用都快人一步')}
                  </Text>
                </div>
                <div className='npc-feature-card'>
                  <div className='npc-feature-icon' style={{ background: 'rgba(168, 85, 247, 0.1)' }}>
                    <Shield size={24} style={{ color: '#a855f7' }} />
                  </div>
                  <Typography.Title heading={5} style={{ marginBottom: '8px' }}>
                    {t('稳定可靠')}
                  </Typography.Title>
                  <Text style={{ color: 'var(--semi-color-text-2)', fontSize: '14px', lineHeight: 1.6 }}>
                    {t('多通道自动故障切换，99.9%+ 可用性保障')}
                  </Text>
                </div>
                <div className='npc-feature-card'>
                  <div className='npc-feature-icon' style={{ background: 'rgba(6, 182, 212, 0.1)' }}>
                    <Globe size={24} style={{ color: '#06b6d4' }} />
                  </div>
                  <Typography.Title heading={5} style={{ marginBottom: '8px' }}>
                    {t('全模型覆盖')}
                  </Typography.Title>
                  <Text style={{ color: 'var(--semi-color-text-2)', fontSize: '14px', lineHeight: 1.6 }}>
                    {t('聚合 30+ 主流 AI 供应商，统一接口一键切换')}
                  </Text>
                </div>
              </div>
            </div>
          </div>

          {/* ===== 供应商展示区 ===== */}
          <div className='w-full px-4 pb-20 md:pb-28 relative z-10'>
            <div className='npc-animate npc-delay-6 max-w-4xl mx-auto'>
              <div className='text-center mb-10'>
                <div className='npc-divider mb-5' />
                <Text style={{
                  fontWeight: 500,
                  fontSize: '15px',
                  letterSpacing: '1px',
                  color: 'var(--semi-color-text-2)',
                }}>
                  {t('支持众多的大模型供应商')}
                </Text>
              </div>
              <div className='npc-provider-grid'>
                <div className='npc-provider-item'><OpenAI size={26} /></div>
                <div className='npc-provider-item'><Claude.Color size={26} /></div>
                <div className='npc-provider-item'><Gemini.Color size={26} /></div>
                <div className='npc-provider-item'><DeepSeek.Color size={26} /></div>
                <div className='npc-provider-item'><Qwen.Color size={26} /></div>
                <div className='npc-provider-item'><XAI size={26} /></div>
                <div className='npc-provider-item'><Grok size={26} /></div>
                <div className='npc-provider-item'><Zhipu.Color size={26} /></div>
                <div className='npc-provider-item'><Moonshot size={26} /></div>
                <div className='npc-provider-item'><Volcengine.Color size={26} /></div>
                <div className='npc-provider-item'><Cohere.Color size={26} /></div>
                <div className='npc-provider-item'><Minimax.Color size={26} /></div>
                <div className='npc-provider-item'><Wenxin.Color size={26} /></div>
                <div className='npc-provider-item'><Spark.Color size={26} /></div>
                <div className='npc-provider-item'><Qingyan.Color size={26} /></div>
                <div className='npc-provider-item'><Suno size={26} /></div>
                <div className='npc-provider-item'><Midjourney size={26} /></div>
                <div className='npc-provider-item'><AzureAI.Color size={26} /></div>
                <div className='npc-provider-item'><Hunyuan.Color size={26} /></div>
                <div className='npc-provider-item'><Xinference.Color size={26} /></div>
                <div className='npc-provider-item'>
                  <Typography.Text style={{
                    fontSize: '16px',
                    fontWeight: 800,
                    background: 'linear-gradient(135deg, #6366f1, #a855f7)',
                    WebkitBackgroundClip: 'text',
                    WebkitTextFillColor: 'transparent',
                  }}>
                    30+
                  </Typography.Text>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className='overflow-x-hidden w-full'>
          {homePageContent.startsWith('https://') ? (
            <iframe
              src={homePageContent}
              className='w-full h-screen border-none'
            />
          ) : (
            <div
              className='mt-[60px]'
              dangerouslySetInnerHTML={{ __html: homePageContent }}
            />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;
