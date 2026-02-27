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
          {/* Hero 区域 */}
          <div className='w-full min-h-[600px] md:min-h-[700px] lg:min-h-[800px] relative overflow-hidden'>
            {/* 动态浮动光球背景 */}
            <div className='hero-orb hero-orb-1' />
            <div className='hero-orb hero-orb-2' />
            <div className='hero-orb hero-orb-3' />
            <div className='hero-orb hero-orb-4' />
            {/* 脉冲环装饰 */}
            <div className='hero-pulse-ring' />
            <div className='hero-pulse-ring hero-pulse-ring-2' />

            <div className='flex items-center justify-center h-full px-4 py-20 md:py-24 lg:py-32 mt-10 relative z-10'>
              <div className='flex flex-col items-center justify-center text-center max-w-4xl mx-auto'>
                {/* 品牌徽标 */}
                <div className='hero-animate-in hero-animate-in-delay-1 mb-6'>
                  <div
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: '8px',
                      padding: '6px 18px',
                      borderRadius: '100px',
                      background: 'rgba(99, 102, 241, 0.08)',
                      border: '1px solid rgba(99, 102, 241, 0.15)',
                      backdropFilter: 'blur(10px)',
                    }}
                  >
                    <div
                      style={{
                        width: '8px',
                        height: '8px',
                        borderRadius: '50%',
                        background: 'linear-gradient(135deg, #6366f1, #a855f7)',
                        boxShadow: '0 0 8px rgba(99, 102, 241, 0.5)',
                      }}
                    />
                    <span
                      style={{
                        fontSize: '13px',
                        fontWeight: 600,
                        color: 'var(--semi-color-text-0)',
                        letterSpacing: '1px',
                      }}
                    >
                      NPC-API
                    </span>
                  </div>
                </div>

                {/* 主标题 */}
                <div className='hero-animate-in hero-animate-in-delay-2 mb-4 md:mb-6'>
                  <h1
                    className={`text-5xl md:text-6xl lg:text-7xl xl:text-8xl font-extrabold leading-tight ${isChinese ? 'tracking-wide md:tracking-wider' : ''}`}
                    style={{ lineHeight: 1.1 }}
                  >
                    {t('统一的')}
                    <br />
                    <span className='hero-gradient-title'>
                      {t('大模型接口网关')}
                    </span>
                  </h1>
                </div>

                {/* 副标题 */}
                <div className='hero-animate-in hero-animate-in-delay-2'>
                  <p
                    className='text-base md:text-lg lg:text-xl max-w-lg mx-auto'
                    style={{
                      color: 'var(--semi-color-text-2)',
                      lineHeight: 1.7,
                    }}
                  >
                    {t('更好的价格，更好的稳定性，只需要将模型基址替换为：')}
                  </p>
                </div>

                {/* 毛玻璃 URL 卡片 */}
                <div className='hero-animate-in hero-animate-in-delay-3 mt-8 md:mt-10 w-full max-w-lg'>
                  <div className='hero-url-card'>
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

                {/* 操作按钮 */}
                <div className='hero-animate-in hero-animate-in-delay-4 flex flex-row gap-4 justify-center items-center mt-8'>
                  <Link to='/console'>
                    <Button
                      theme='solid'
                      type='primary'
                      size={isMobile ? 'default' : 'large'}
                      className='!rounded-3xl'
                      icon={<IconPlay />}
                      style={{
                        padding: '10px 32px',
                        fontWeight: 600,
                        fontSize: '15px',
                        background: 'linear-gradient(135deg, #6366f1 0%, #8b5cf6 50%, #a855f7 100%)',
                        border: 'none',
                        boxShadow: '0 6px 24px rgba(99, 102, 241, 0.35), 0 0 0 1px rgba(99, 102, 241, 0.1)',
                        transition: 'all 0.3s ease',
                      }}
                    >
                      {t('获取密钥')}
                    </Button>
                  </Link>
                  {isDemoSiteMode && statusState?.status?.version ? (
                    <Button
                      size={isMobile ? 'default' : 'large'}
                      className='flex items-center !rounded-3xl'
                      icon={<IconGithubLogo />}
                      onClick={() =>
                        window.open(
                          'https://github.com/QuantumNous/new-api',
                          '_blank',
                        )
                      }
                      style={{
                        padding: '10px 24px',
                        fontWeight: 500,
                        borderRadius: '24px',
                        border: '1px solid var(--semi-color-border)',
                        background: 'var(--semi-color-bg-1)',
                        transition: 'all 0.3s ease',
                      }}
                    >
                      {statusState.status.version}
                    </Button>
                  ) : (
                    docsLink && (
                      <Button
                        size={isMobile ? 'default' : 'large'}
                        className='flex items-center !rounded-3xl'
                        icon={<IconFile />}
                        onClick={() => window.open(docsLink, '_blank')}
                        style={{
                          padding: '10px 24px',
                          fontWeight: 500,
                          borderRadius: '24px',
                          border: '1px solid var(--semi-color-border)',
                          background: 'var(--semi-color-bg-1)',
                          transition: 'all 0.3s ease',
                        }}
                      >
                        {t('文档')}
                      </Button>
                    )
                  )}
                </div>

                {/* 供应商图标区 */}
                <div className='hero-animate-in hero-animate-in-delay-5 mt-16 md:mt-20 lg:mt-24 w-full'>
                  <div className='flex items-center mb-8 md:mb-10 justify-center'>
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '12px',
                      }}
                    >
                      <div
                        style={{
                          width: '40px',
                          height: '1px',
                          background: 'linear-gradient(to right, transparent, var(--semi-color-border))',
                        }}
                      />
                      <Text
                        style={{
                          fontWeight: 400,
                          letterSpacing: '2px',
                          fontSize: '14px',
                          textTransform: 'uppercase',
                          color: 'var(--semi-color-text-2)',
                        }}
                      >
                        {t('支持众多的大模型供应商')}
                      </Text>
                      <div
                        style={{
                          width: '40px',
                          height: '1px',
                          background: 'linear-gradient(to left, transparent, var(--semi-color-border))',
                        }}
                      />
                    </div>
                  </div>
                  <div className='flex flex-wrap items-center justify-center gap-3 sm:gap-4 max-w-4xl mx-auto px-4'>
                    <div className='provider-icon-wrapper'><Moonshot size={28} /></div>
                    <div className='provider-icon-wrapper'><OpenAI size={28} /></div>
                    <div className='provider-icon-wrapper'><XAI size={28} /></div>
                    <div className='provider-icon-wrapper'><Zhipu.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Volcengine.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Cohere.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Claude.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Gemini.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Suno size={28} /></div>
                    <div className='provider-icon-wrapper'><Minimax.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Wenxin.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Spark.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Qingyan.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><DeepSeek.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Qwen.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Midjourney size={28} /></div>
                    <div className='provider-icon-wrapper'><Grok size={28} /></div>
                    <div className='provider-icon-wrapper'><AzureAI.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Hunyuan.Color size={28} /></div>
                    <div className='provider-icon-wrapper'><Xinference.Color size={28} /></div>
                    <div className='provider-icon-wrapper'>
                      <Typography.Text
                        style={{
                          fontSize: '18px',
                          fontWeight: 800,
                          background: 'linear-gradient(135deg, #6366f1, #a855f7)',
                          WebkitBackgroundClip: 'text',
                          WebkitTextFillColor: 'transparent',
                        }}
                      >
                        30+
                      </Typography.Text>
                    </div>
                  </div>
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
