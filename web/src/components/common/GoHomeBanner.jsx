import React, { useEffect, useState, useRef } from 'react';

const API_BASE = 'https://api.xunjinlu.fun/api/babygome/index.php';
const INTERVAL = 7000;
const PRELOAD = 5;

const fetchOne = async () => {
  try {
    const res = await fetch(`${API_BASE}?type=json&_t=${Date.now()}`);
    const json = await res.json();
    if (json.code !== 200 || !json.data) return null;
    const d = json.data;
    // 计算走失时年龄
    let age = '';
    if (d.birthDay && d.lostDay) {
      const birth = new Date(d.birthDay);
      const lost = new Date(d.lostDay);
      const y = lost.getFullYear() - birth.getFullYear();
      const m = lost.getMonth() - birth.getMonth();
      age = String(m < 0 ? y - 1 : y);
    }
    return {
      name: d.name || '',
      sex: d.sex || '',
      age,
      lostDay: d.lostDay || '',
      lostAddress: d.lostAddress || '',
      photoUrl: d.photoUrl || '',
      detailUrl: d.detailUrl || 'https://www.baobeihuijia.com/',
    };
  } catch (_) {
    return null;
  }
};

const GoHomeBanner = () => {
  const [items, setItems] = useState([]);
  const [idx, setIdx] = useState(0);
  const [visible, setVisible] = useState(true);
  const timerRef = useRef(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      for (let i = 0; i < PRELOAD; i++) {
        if (cancelled) break;
        const item = await fetchOne();
        if (item && item.name) setItems(prev => [...prev, item]);
        await new Promise(r => setTimeout(r, 400));
      }
    })();
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    if (items.length < 2) return;
    timerRef.current = setInterval(() => {
      setVisible(false);
      setTimeout(() => {
        setIdx(prev => (prev + 1) % items.length);
        setVisible(true);
      }, 400);
    }, INTERVAL);
    return () => clearInterval(timerRef.current);
  }, [items.length]);

  const cur = items[idx];

  return (
    <div className='gohome-topbar'>
      {/* 左侧标签 */}
      <span className='gohome-topbar-badge'>🔍 宝贝回家</span>
      <span className='gohome-topbar-divider' />

      {/* 轮播内容 */}
      <div className={`gohome-topbar-content ${visible ? 'gohome-visible' : 'gohome-hidden'}`}>
        {cur ? (
          <>
            {cur.photoUrl && (
              <a href={cur.detailUrl} target='_blank' rel='noopener noreferrer' className='gohome-topbar-img-wrap'>
                <img
                  className='gohome-topbar-photo'
                  src={cur.photoUrl}
                  alt={cur.name}
                  referrerPolicy='no-referrer'
                />
              </a>
            )}
            <span className='gohome-topbar-name'>{cur.name}</span>
            {cur.sex && <span className='gohome-topbar-meta'>{cur.sex}</span>}
            {cur.age && <span className='gohome-topbar-meta'>走失时 {cur.age} 岁</span>}
            {cur.lostDay && <span className='gohome-topbar-meta'>· {cur.lostDay} 走失</span>}
            {cur.lostAddress && <span className='gohome-topbar-meta'>· {cur.lostAddress}</span>}
          </>
        ) : (
          <span className='gohome-topbar-meta'>数据加载中...</span>
        )}
      </div>

      {/* 查看详情 */}
      {cur?.detailUrl && (
        <a
          href={cur.detailUrl}
          target='_blank'
          rel='noopener noreferrer'
          className='gohome-topbar-link'
        >
          查看详情 →
        </a>
      )}

      {/* 右侧提示 */}
      <span className='gohome-topbar-hotline'>
        线索请拨 <strong>110</strong>
      </span>
    </div>
  );
};

export default GoHomeBanner;
