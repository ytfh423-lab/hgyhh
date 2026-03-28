import React, { useEffect, useState, useRef } from 'react';

const API_IMG = 'https://api.zjb522.cn/api?type=img';
const DETAIL_URL = 'https://www.baobeihuijia.com/';
const SLIDE_INTERVAL = 5000;
const SLIDE_COUNT = 5;

// 预生成若干张不同的随机海报 URL（时间戳错开，确保返回不同图片）
const makeSlides = () =>
  Array.from({ length: SLIDE_COUNT }, (_, i) => ({
    src: `${API_IMG}&_t=${Date.now() + i * 1000}`,
    key: i,
  }));

const GoHomeBanner = () => {
  const [slides] = useState(makeSlides);
  const [cur, setCur] = useState(0);
  const timerRef = useRef(null);

  const startTimer = (from = cur) => {
    clearInterval(timerRef.current);
    timerRef.current = setInterval(() => {
      setCur(prev => (prev + 1) % SLIDE_COUNT);
    }, SLIDE_INTERVAL);
  };

  useEffect(() => {
    startTimer();
    return () => clearInterval(timerRef.current);
  }, []);

  const go = (next) => {
    setCur((next + SLIDE_COUNT) % SLIDE_COUNT);
    startTimer();
  };

  return (
    <div className='gohome-carousel'>
      <div className='gohome-track'>
        {slides.map((s, i) => (
          <div key={s.key} className={`gohome-slide ${i === cur ? 'gohome-slide-active' : ''}`}>
            <a href={DETAIL_URL} target='_blank' rel='noopener noreferrer' className='gohome-slide-img-link'>
              <img
                className='gohome-slide-img'
                src={s.src}
                alt='宝贝回家寻人海报'
              />
            </a>
          </div>
        ))}
      </div>

      {/* 左右箭头 */}
      <button className='gohome-arrow gohome-arrow-left' onClick={() => go(cur - 1)}>‹</button>
      <button className='gohome-arrow gohome-arrow-right' onClick={() => go(cur + 1)}>›</button>

      {/* 底部圆点 */}
      <div className='gohome-footer'>
        <span className='gohome-footer-label'>🔍 宝贝回家 · 公益寻人 · 如有线索请拨打 <strong>110</strong></span>
        <div className='gohome-dots'>
          {slides.map((_, i) => (
            <button
              key={i}
              className={`gohome-dot ${i === cur ? 'gohome-dot-active' : ''}`}
              onClick={() => go(i)}
            />
          ))}
        </div>
      </div>
    </div>
  );
};

export default GoHomeBanner;
