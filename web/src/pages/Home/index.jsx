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

import React, { useContext, useEffect, useState, useRef, useMemo } from 'react';
import { API, showError } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import NoticeModal from '../../components/layout/NoticeModal';
import GoHomeBanner from '../../components/common/GoHomeBanner';
import {
  Moonshot, OpenAI, XAI, Zhipu, Volcengine, Cohere, Claude, Gemini,
  Suno, Minimax, Wenxin, Spark, Qingyan, DeepSeek, Qwen, Midjourney,
  Grok, AzureAI, Hunyuan, Xinference,
} from '@lobehub/icons';

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const isMobile = useIsMobile();
  const canvasRef = useRef(null);
  const [visibleSections, setVisibleSections] = useState(new Set());
  const [betaInfo, setBetaInfo] = useState(null);
  const [reserving, setReserving] = useState(false);
  const betaEnabled = statusState?.status?.farm_beta_enabled || false;
  const betaMaxSlots = statusState?.status?.farm_beta_max_slots || 0;

  const targetDate = useMemo(() => {
    const raw = statusState?.status?.farm_countdown_date;
    if (raw) {
      const parsed = new Date(raw);
      if (!isNaN(parsed.getTime())) return parsed;
    }
    const d = new Date();
    d.setDate(d.getDate() + 30);
    return d;
  }, [statusState?.status?.farm_countdown_date]);
  const [countdown, setCountdown] = useState({ d: 0, h: 0, m: 0, s: 0 });
  const countdownExpired = targetDate <= Date.now();

  useEffect(() => {
    const tick = () => {
      const diff = Math.max(0, targetDate - Date.now());
      setCountdown({
        d: Math.floor(diff / 86400000),
        h: Math.floor((diff / 3600000) % 24),
        m: Math.floor((diff / 60000) % 60),
        s: Math.floor((diff / 1000) % 60),
      });
    };
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [targetDate]);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((e) => {
          if (e.isIntersecting) {
            setVisibleSections((prev) => new Set([...prev, e.target.dataset.section]));
          }
        });
      },
      { threshold: 0.12 },
    );
    document.querySelectorAll('[data-section]').forEach((el) => observer.observe(el));
    return () => observer.disconnect();
  }, [homePageContentLoaded, homePageContent]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    let animId;
    let particles = [];
    const resize = () => {
      canvas.width = canvas.offsetWidth * (window.devicePixelRatio || 1);
      canvas.height = canvas.offsetHeight * (window.devicePixelRatio || 1);
      ctx.scale(window.devicePixelRatio || 1, window.devicePixelRatio || 1);
    };
    resize();
    const N = isMobile ? 30 : 70;
    for (let i = 0; i < N; i++) {
      particles.push({
        x: Math.random() * canvas.offsetWidth,
        y: Math.random() * canvas.offsetHeight,
        r: Math.random() * 1.5 + 0.3,
        dx: (Math.random() - 0.5) * 0.25,
        dy: -Math.random() * 0.15 - 0.03,
        o: Math.random() * 0.5 + 0.1,
        gold: Math.random() > 0.35,
      });
    }
    const draw = () => {
      ctx.clearRect(0, 0, canvas.offsetWidth, canvas.offsetHeight);
      particles.forEach((p) => {
        ctx.beginPath();
        ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2);
        ctx.fillStyle = p.gold
          ? `rgba(251, 191, 36, ${p.o})`
          : `rgba(161, 161, 170, ${p.o * 0.4})`;
        ctx.fill();
        p.x += p.dx;
        p.y += p.dy;
        if (p.x < 0 || p.x > canvas.offsetWidth) p.dx *= -1;
        if (p.y < -10) {
          p.y = canvas.offsetHeight + 10;
          p.x = Math.random() * canvas.offsetWidth;
        }
      });
      animId = requestAnimationFrame(draw);
    };
    draw();
    window.addEventListener('resize', resize);
    return () => {
      cancelAnimationFrame(animId);
      window.removeEventListener('resize', resize);
    };
  }, [isMobile]);

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    const res = await API.get('/api/home_page_content');
    const { success, message, data } = res.data;
    if (success) {
      let content = data;
      if (!data.startsWith('https://')) content = marked.parse(data);
      setHomePageContent(content);
      localStorage.setItem('home_page_content', content);
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
      setHomePageContent('');
    }
    setHomePageContentLoaded(true);
  };

  useEffect(() => {
    const check = async () => {
      const last = localStorage.getItem('notice_close_date');
      if (last !== new Date().toDateString()) {
        try {
          const res = await API.get('/api/notice');
          if (res.data.success && res.data.data?.trim()) setNoticeVisible(true);
        } catch (_) { /* ignore */ }
      }
    };
    check();
  }, []);

  useEffect(() => {
    displayHomePageContent();
  }, []);

  const fetchBetaStatus = async () => {
    try {
      const res = await API.get('/api/farm/beta/status');
      if (res.data.success) setBetaInfo(res.data.data);
    } catch (_) { /* ignore if not logged in */ }
  };

  useEffect(() => {
    if (betaEnabled) fetchBetaStatus();
  }, [betaEnabled]);

  const handleReserve = async () => {
    setReserving(true);
    try {
      const res = await API.post('/api/farm/beta/reserve');
      if (res.data.success) {
        await fetchBetaStatus();
      } else {
        showError(res.data.message);
      }
    } catch (_) {
      showError(t('预约失败'));
    }
    setReserving(false);
  };

  const farmFeatures = [
    { icon: '\u{1F33E}', title: '\u667A\u80FD\u79CD\u690D', desc: 'AI \u9A71\u52A8\u519C\u4F5C\u7269\u751F\u957F\u7CFB\u7EDF\uFF0C\u6A21\u62DF\u771F\u5B9E\u56DB\u5B63\u53D8\u5316', tag: 'CORE' },
    { icon: '\u{1F4CA}', title: '\u52A8\u6001\u5E02\u573A', desc: '\u4F9B\u9700\u9A71\u52A8\u5B9E\u65F6\u7ECF\u6D4E\uFF0C\u6BCF\u7B14\u4EA4\u6613\u5F71\u54CD\u5168\u5C40\u4EF7\u683C', tag: 'ECONOMY' },
    { icon: '\u{1F91D}', title: '\u793E\u4EA4\u751F\u6001', desc: '\u62DC\u8BBF\u597D\u53CB\u519C\u573A\u3001\u5077\u83DC\u4E92\u52A8\u3001\u7EC4\u5EFA\u516C\u4F1A\u534F\u4F5C', tag: 'SOCIAL' },
    { icon: '\u{1F3C6}', title: '\u7ADE\u6280\u7CFB\u7EDF', desc: '\u5168\u7403\u6392\u884C\u699C\u3001\u8D5B\u5B63\u5956\u52B1\u3001\u7A00\u6709\u6210\u5C31\u89E3\u9501', tag: 'COMPETE' },
  ];

  const providerIcons = [
    <OpenAI size={22} key='openai' />, <Claude.Color size={22} key='claude' />,
    <Gemini.Color size={22} key='gemini' />, <DeepSeek.Color size={22} key='ds' />,
    <Qwen.Color size={22} key='qwen' />, <XAI size={22} key='xai' />,
    <Grok size={22} key='grok' />, <Zhipu.Color size={22} key='zhipu' />,
    <Moonshot size={22} key='moon' />, <Volcengine.Color size={22} key='volc' />,
    <Cohere.Color size={22} key='cohere' />, <Minimax.Color size={22} key='mm' />,
    <Wenxin.Color size={22} key='wx' />, <Spark.Color size={22} key='spark' />,
    <Qingyan.Color size={22} key='qy' />, <Suno size={22} key='suno' />,
    <Midjourney size={22} key='mj' />, <AzureAI.Color size={22} key='azure' />,
    <Hunyuan.Color size={22} key='hy' />, <Xinference.Color size={22} key='xf' />,
  ];

  const pad = (n) => String(n).padStart(2, '0');
  const sc = (name) => `cy-section cy-reveal ${visibleSections.has(name) ? 'cy-visible' : ''}`;

  return (
    <div className='cy-landing'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      {homePageContentLoaded && homePageContent === '' ? (
        <>
          {/* ═══ 宝贝回家大图轮播（最顶部） ═══ */}
          <div style={{ position: 'relative', zIndex: 3 }}>
            <GoHomeBanner />
          </div>

          {/* Animated grid background */}
          <div className='cy-grid-bg' />
          {/* Gold particles */}
          <canvas ref={canvasRef} className='cy-particles' />

          {/* Side floating labels */}
          {!isMobile && (
            <>
              <div className='cy-side cy-side-l'>
                <span>NPC</span><span>{t('农场')}</span><span>BETA</span>
              </div>
              <div className='cy-side cy-side-r'>
                <span>2025</span><span>{t('公益')}</span><span>AI</span>
              </div>
            </>
          )}

          {/* ═══ HERO ═══ */}
          <section className='cy-hero'>
            <div className='cy-hero-glow' />
            <div className='cy-hero-ring cy-ring-1' />
            <div className='cy-hero-ring cy-ring-2' />
            <div className='cy-hero-ring cy-ring-3' />

            <div className='cy-hero-inner'>
              <div className='cy-hero-badge'>NON-PROFIT · OPEN SOURCE · AI FOR ALL</div>
              <h1 className='cy-hero-title'>
                <span className='cy-glow-text'>NPC</span>
                <span className='cy-title-sep' />
                <span className='cy-title-sub'>{t('公益')}</span>
              </h1>
              <p className='cy-hero-tagline'>{t('非营利 · 纯粹 · 开放')}</p>
              <p className='cy-hero-desc'>
                {t('聚合 30+ 主流 AI 模型，打造零门槛的公益 AI 接口平台')}
              </p>
              <div className='cy-hero-actions'>
                <Link to='/console'>
                  <button className='cy-btn cy-btn-gold'>{t('进入控制台')}</button>
                </Link>
                <Link to='/farm'>
                  <button className='cy-btn cy-btn-outline'>🌾 {t('探索农场')}</button>
                </Link>
              </div>

              {/* Countdown + Reservation — directly in hero */}
              <div className='cy-countdown cy-countdown-hero'>
                <div className='cy-countdown-label'>
                  {countdownExpired ? t('内测已开启') : t('NPC 农场 · 内测倒计时')}
                </div>
                {!countdownExpired ? (
                  <>
                    <div className='cy-countdown-row'>
                      {[
                        { v: pad(countdown.d), l: t('天') },
                        { v: pad(countdown.h), l: t('时') },
                        { v: pad(countdown.m), l: t('分') },
                        { v: pad(countdown.s), l: t('秒') },
                      ].map((c, i) => (
                        <div key={i} className='cy-cd-cell'>
                          <span className='cy-cd-num'>{c.v}</span>
                          <span className='cy-cd-unit'>{c.l}</span>
                        </div>
                      ))}
                    </div>
                    {betaEnabled && (
                      <div className='cy-beta-reserve'>
                        <div className='cy-beta-slots'>
                          <span className='cy-beta-slots-num'>{betaInfo?.total_reserved ?? 0}</span>
                          <span className='cy-beta-slots-sep'>/</span>
                          <span className='cy-beta-slots-max'>{betaMaxSlots}</span>
                          <span className='cy-beta-slots-label'>{t('已预约')}</span>
                        </div>
                        {betaInfo?.reserved ? (
                          <div className='cy-beta-done'>
                            <span className='cy-beta-check'>✓</span>
                            {t('已预约')} · {t('排名')} #{betaInfo.rank}
                          </div>
                        ) : (
                          <button
                            className='cy-btn cy-btn-gold'
                            onClick={handleReserve}
                            disabled={reserving || (betaInfo?.slots_remaining !== undefined && betaInfo.slots_remaining <= 0)}
                          >
                            {reserving ? t('预约中...') : t('🔥 预约内测资格')}
                          </button>
                        )}
                        {betaInfo?.slots_remaining !== undefined && betaInfo.slots_remaining <= 0 && !betaInfo?.reserved && (
                          <div className='cy-beta-full'>{t('名额已满')}</div>
                        )}
                      </div>
                    )}
                  </>
                ) : (
                  <div style={{ textAlign: 'center' }}>
                    <Link to='/farm'>
                      <button className='cy-btn cy-btn-gold cy-btn-lg'>🌾 {t('立即进入农场')}</button>
                    </Link>
                  </div>
                )}
              </div>
            </div>
          </section>

          {/* ═══ TICKER ═══ */}
          <div className='cy-ticker'>
            <div className='cy-ticker-track'>
              {Array(10).fill(null).map((_, i) => (
                <span key={i} className='cy-ticker-item'>
                  ◆ NPC {t('农场内测即将开启')} ◆ {t('全新数字公益生态')} ◆ REDEFINE PUBLIC WELFARE ◆ AI-POWERED FARM ◆&nbsp;
                </span>
              ))}
            </div>
          </div>

          {/* ═══ FARM PROMO ═══ */}
          <section className={sc('farm')} data-section='farm'>
            <div className='cy-section-head'>
              <div className='cy-gold-line' />
              <h2 className='cy-section-title'>NPC {t('农场')}</h2>
              <p className='cy-section-sub'>{t('重新定义公益与数字农场的结合')}</p>
            </div>

            {/* Glass cards */}
            <div className='cy-cards'>
              {farmFeatures.map((f, i) => (
                <Link to='/farm' key={i} style={{ textDecoration: 'none' }}>
                  <div className='cy-glass-card' style={{ animationDelay: `${i * 0.12}s` }}>
                    <div className='cy-card-tag'>{f.tag}</div>
                    <div className='cy-card-icon'>{f.icon}</div>
                    <h3 className='cy-card-title'>{t(f.title)}</h3>
                    <p className='cy-card-desc'>{t(f.desc)}</p>
                  </div>
                </Link>
              ))}
            </div>

            {/* CTA */}
            <div className='cy-cta'>
              <p className='cy-cta-copy'>
                {t('全新的数字生态，重新定义公益与农场的结合。')}
                <br />
                {t('你准备好了吗？')}
              </p>
              <Link to='/farm'>
                <button className='cy-btn cy-btn-gold cy-btn-lg'>🌾 {t('预约内测')}</button>
              </Link>
            </div>
          </section>

          {/* ═══ MANIFESTO ═══ */}
          <section className={sc('manifesto')} data-section='manifesto'>
            <blockquote className='cy-quote'>
              <span className='cy-quote-mark'>&ldquo;</span>
              {t('我们相信，AI 不应该被少数人垄断。')}<br />
              {t('NPC 公益，让每个人都能平等地使用最前沿的 AI 技术。')}<br />
              {t('NPC 农场，让公益变得有趣、可持续、充满想象力。')}
              <span className='cy-quote-mark'>&rdquo;</span>
            </blockquote>
          </section>

          {/* ═══ PROVIDERS ═══ */}
          <section className={sc('providers')} data-section='providers'>
            <div className='cy-section-head'>
              <div className='cy-gold-line' />
              <h2 className='cy-section-title'>{t('底层技术支持')}</h2>
              <p className='cy-section-sub'>{t('聚合全球 30+ 主流 AI 供应商')}</p>
            </div>
            <div className='cy-providers'>
              {providerIcons.map((icon, i) => (
                <div key={i} className='cy-prov-item'>{icon}</div>
              ))}
              <div className='cy-prov-item cy-prov-count'>30+</div>
            </div>
          </section>

          {/* ═══ FOOTER ═══ */}
          <footer className='cy-footer'>
            <div className='cy-footer-brand'>NPC {t('公益')}</div>
            <div className='cy-footer-text'>{t('非营利 · 开源 · 让 AI 属于每个人')}</div>
          </footer>

          {/* Ad HTML */}
          {statusState?.status?.home_ad_html && (
            <div
              className='cy-ad'
              dangerouslySetInnerHTML={{ __html: statusState.status.home_ad_html }}
            />
          )}
        </>
      ) : (
        <div className='w-full overflow-x-hidden'>
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
