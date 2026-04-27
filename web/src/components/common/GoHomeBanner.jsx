import React, { useEffect, useState, useRef } from 'react';

const DETAIL_URL = 'https://www.baobeihuijia.com/';
const SLIDE_INTERVAL = 5000;
const PRELOAD_COUNT = 5;

const fetchOne = async () => {
  try {
    const res = await fetch(`https://grok.hyw.me/api/babygome?_t=${Date.now()}`);
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
      detailUrl: d.detailUrl || DETAIL_URL,
      categoryName: d.categoryName || '宝贝回家',
    };
  } catch (_) {
    return null;
  }
};

const GoHomeBanner = () => {
  const [slides, setSlides] = useState([]);
  const [cur, setCur] = useState(0);
  const timerRef = useRef(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      for (let i = 0; i < PRELOAD_COUNT; i++) {
        if (cancelled) break;
        const item = await fetchOne();
        if (item && item.name) setSlides(prev => [...prev, item]);
        await new Promise(r => setTimeout(r, 500));
      }
    })();
    return () => { cancelled = true; };
  }, []);

  const startTimer = () => {
    clearInterval(timerRef.current);
    timerRef.current = setInterval(() => {
      setCur(prev => (prev + 1) % Math.max(slides.length, 1));
    }, SLIDE_INTERVAL);
  };

  useEffect(() => {
    if (slides.length < 2) return;
    startTimer();
    return () => clearInterval(timerRef.current);
  }, [slides.length]);

  const go = (n) => {
    setCur((n + slides.length) % slides.length);
    startTimer();
  };

  const s = slides[cur];

  return (
    <div className='gohome-carousel'>
      <div className='gohome-track'>
        {slides.length === 0 ? (
          <div className='gohome-loading'>
            <div className='gohome-spinner' />
            <span>加载寻人信息中...</span>
          </div>
        ) : (
          slides.map((item, i) => (
            <div key={i} className={`gohome-slide ${i === cur ? 'gohome-slide-active' : ''}`}>
              {/* 背景虚化 */}
              {item.photoUrl && (
                <div className='gohome-slide-bg' style={{ backgroundImage: `url(${item.photoUrl})` }} />
              )}
              <div className='gohome-slide-overlay' />

              {/* 内容 */}
              <div className='gohome-slide-body'>
                {/* 照片 */}
                <a href={item.detailUrl} target='_blank' rel='noopener noreferrer' className='gohome-photo-wrap'>
                  {item.photoUrl
                    ? <img className='gohome-photo' src={item.photoUrl} alt={item.name} referrerPolicy='no-referrer' />
                    : <div className='gohome-photo gohome-photo-empty'>?</div>
                  }
                </a>

                {/* 文字 */}
                <div className='gohome-info'>
                  <div className='gohome-tag'>{item.categoryName}</div>
                  <div className='gohome-name'>{item.name}</div>
                  <div className='gohome-fields'>
                    {item.sex && <span className='gohome-field'>{item.sex}</span>}
                    {item.age && <span className='gohome-field'>走失时 {item.age} 岁</span>}
                    {item.lostHeight && item.lostHeight !== '未知' && <span className='gohome-field'>身高 {item.lostHeight}</span>}
                    {item.lostDay && <span className='gohome-field'>{item.lostDay} 走失</span>}
                  </div>
                  {item.lostAddress && <div className='gohome-address'>📍 {item.lostAddress}</div>}
                  {item.feature && <div className='gohome-feature'>"{item.feature}"</div>}
                  <div className='gohome-actions'>
                    <a href={item.detailUrl} target='_blank' rel='noopener noreferrer' className='gohome-btn'>查看详情</a>
                    <span className='gohome-hotline'>如有线索请拨打 <strong>110</strong></span>
                  </div>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {slides.length > 1 && (
        <>
          <button className='gohome-arrow gohome-arrow-left' onClick={() => go(cur - 1)}>‹</button>
          <button className='gohome-arrow gohome-arrow-right' onClick={() => go(cur + 1)}>›</button>
        </>
      )}

      <div className='gohome-footer'>
        <span className='gohome-footer-label'>🔍 宝贝回家 · 公益寻人</span>
        <div className='gohome-dots'>
          {slides.map((_, i) => (
            <button key={i} className={`gohome-dot ${i === cur ? 'gohome-dot-active' : ''}`} onClick={() => go(i)} />
          ))}
        </div>
      </div>
    </div>
  );
};

export default GoHomeBanner;
