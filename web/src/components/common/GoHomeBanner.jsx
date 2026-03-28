import React, { useEffect, useState, useRef, useCallback } from 'react';

const API_BASE = 'https://api.xunjinlu.fun/api/babygome/index.php';
const SLIDE_INTERVAL = 5000;
const PRELOAD_COUNT = 5;

const fetchOne = async () => {
  try {
    const res = await fetch(`${API_BASE}?type=json&_t=${Date.now()}`);
    const json = await res.json();
    if (json.code !== 200 || !json.data) return null;
    const d = json.data;
    let age = '';
    if (d.birthDay && d.lostDay) {
      const birth = new Date(d.birthDay);
      const lost = new Date(d.lostDay);
      const y = lost.getFullYear() - birth.getFullYear();
      age = String(lost.getMonth() < birth.getMonth() ? y - 1 : y);
    }
    return {
      name: d.name || '',
      sex: d.sex || '',
      age,
      lostDay: d.lostDay || '',
      lostAddress: (d.lostAddress || '').replace(/,/g, ' '),
      lostHeight: d.lostHeight || '',
      feature: d.feature || '',
      photoUrl: d.photoUrl || '',
      detailUrl: d.detailUrl || 'https://www.baobeihuijia.com/',
      categoryName: d.categoryName || '',
    };
  } catch (_) {
    return null;
  }
};

const GoHomeBanner = () => {
  const [slides, setSlides] = useState([]);
  const [cur, setCur] = useState(0);
  const [animating, setAnimating] = useState(false);
  const timerRef = useRef(null);
  const loadedRef = useRef(0);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      for (let i = 0; i < PRELOAD_COUNT; i++) {
        if (cancelled) break;
        const item = await fetchOne();
        if (item && item.name) {
          setSlides(prev => [...prev, item]);
          loadedRef.current += 1;
        }
        await new Promise(r => setTimeout(r, 500));
      }
    })();
    return () => { cancelled = true; };
  }, []);

  const goTo = useCallback((next) => {
    if (animating) return;
    setAnimating(true);
    setTimeout(() => {
      setCur(next);
      setAnimating(false);
    }, 450);
  }, [animating]);

  useEffect(() => {
    if (slides.length < 2) return;
    timerRef.current = setInterval(() => {
      setCur(prev => {
        const next = (prev + 1) % slides.length;
        return next;
      });
    }, SLIDE_INTERVAL);
    return () => clearInterval(timerRef.current);
  }, [slides.length]);

  const handleDot = (i) => {
    clearInterval(timerRef.current);
    setCur(i);
    timerRef.current = setInterval(() => {
      setCur(prev => (prev + 1) % slides.length);
    }, SLIDE_INTERVAL);
  };

  const handlePrev = () => {
    clearInterval(timerRef.current);
    setCur(prev => (prev - 1 + slides.length) % slides.length);
    timerRef.current = setInterval(() => {
      setCur(prev => (prev + 1) % slides.length);
    }, SLIDE_INTERVAL);
  };

  const handleNext = () => {
    clearInterval(timerRef.current);
    setCur(prev => (prev + 1) % slides.length);
    timerRef.current = setInterval(() => {
      setCur(prev => (prev + 1) % slides.length);
    }, SLIDE_INTERVAL);
  };

  const slide = slides[cur];

  return (
    <div className='gohome-carousel'>
      {/* 滑动轨道 */}
      <div className='gohome-track'>
        {slides.length === 0 ? (
          <div className='gohome-loading'>
            <div className='gohome-loading-spinner' />
            <span>加载寻人信息中...</span>
          </div>
        ) : (
          slides.map((s, i) => (
            <div
              key={i}
              className={`gohome-slide ${i === cur ? 'gohome-slide-active' : ''}`}
            >
              {/* 背景模糊图 */}
              {s.photoUrl && (
                <div
                  className='gohome-slide-bg'
                  style={{ backgroundImage: `url(${s.photoUrl})` }}
                />
              )}
              <div className='gohome-slide-overlay' />

              {/* 内容 */}
              <div className='gohome-slide-body'>
                {/* 照片 */}
                <a href={s.detailUrl} target='_blank' rel='noopener noreferrer' className='gohome-slide-photo-link'>
                  {s.photoUrl ? (
                    <img
                      className='gohome-slide-photo'
                      src={s.photoUrl}
                      alt={s.name}
                      referrerPolicy='no-referrer'
                    />
                  ) : (
                    <div className='gohome-slide-photo gohome-slide-photo-empty'>?</div>
                  )}
                </a>

                {/* 文字区 */}
                <div className='gohome-slide-info'>
                  <div className='gohome-slide-tag'>
                    {s.categoryName || '宝贝回家 · 寻人公益'}
                  </div>
                  <div className='gohome-slide-name'>{s.name}</div>
                  <div className='gohome-slide-fields'>
                    {s.sex && <span className='gohome-slide-field'>{s.sex}</span>}
                    {s.age && <span className='gohome-slide-field'>走失时 {s.age} 岁</span>}
                    {s.lostHeight && s.lostHeight !== '未知' && (
                      <span className='gohome-slide-field'>身高 {s.lostHeight}</span>
                    )}
                    {s.lostDay && <span className='gohome-slide-field'>走失 {s.lostDay}</span>}
                  </div>
                  {s.lostAddress && (
                    <div className='gohome-slide-address'>📍 {s.lostAddress}</div>
                  )}
                  {s.feature && (
                    <div className='gohome-slide-feature'>"{s.feature}"</div>
                  )}
                  <div className='gohome-slide-actions'>
                    <a
                      href={s.detailUrl}
                      target='_blank'
                      rel='noopener noreferrer'
                      className='gohome-slide-btn'
                    >
                      查看详情
                    </a>
                    <span className='gohome-slide-hotline'>如有线索请拨打 <strong>110</strong></span>
                  </div>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* 左右箭头 */}
      {slides.length > 1 && (
        <>
          <button className='gohome-arrow gohome-arrow-left' onClick={handlePrev}>‹</button>
          <button className='gohome-arrow gohome-arrow-right' onClick={handleNext}>›</button>
        </>
      )}

      {/* 底部圆点 + 标题 */}
      <div className='gohome-footer'>
        <span className='gohome-footer-label'>🔍 宝贝回家公益寻人</span>
        <div className='gohome-dots'>
          {slides.map((_, i) => (
            <button
              key={i}
              className={`gohome-dot ${i === cur ? 'gohome-dot-active' : ''}`}
              onClick={() => handleDot(i)}
            />
          ))}
        </div>
      </div>
    </div>
  );
};

export default GoHomeBanner;
