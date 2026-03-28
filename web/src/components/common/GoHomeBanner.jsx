import React, { useEffect, useState, useRef, useCallback } from 'react';

const API_JSON = 'https://api.zjb522.cn/api?type=json&order=random';
const GOTO_URL = 'https://www.baobeihuijia.com/';
const INTERVAL = 6000; // 每条展示 6 秒
const PRELOAD = 5;     // 预加载几条

const fetchOne = async () => {
  try {
    const res = await fetch(`${API_JSON}&_t=${Date.now()}`);
    const d = await res.json();
    return {
      name: d.name || d.xing_ming || '',
      age: d.age || d.nian_ling || '',
      date: d.missing_time || d.shi_zong_shi_jian || d.time || '',
      place: d.missing_place || d.shi_zong_di_dian || d.place || '',
      url: d.baobei_url || d.url || d.link || GOTO_URL,
    };
  } catch (_) {
    return null;
  }
};

const GoHomeBanner = () => {
  const [items, setItems] = useState([]);
  const [idx, setIdx] = useState(0);
  const [fade, setFade] = useState(true);
  const timerRef = useRef(null);

  // 预加载多条数据
  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      for (let i = 0; i < PRELOAD; i++) {
        if (cancelled) break;
        const item = await fetchOne();
        if (item && (item.name || item.place)) {
          setItems(prev => [...prev, item]);
        }
        // 错开请求，避免 API 返回同一条
        await new Promise(r => setTimeout(r, 300));
      }
    };
    load();
    return () => { cancelled = true; };
  }, []);

  // 轮播定时器
  useEffect(() => {
    if (items.length < 2) return;
    timerRef.current = setInterval(() => {
      setFade(false);
      setTimeout(() => {
        setIdx(prev => (prev + 1) % items.length);
        setFade(true);
      }, 350);
    }, INTERVAL);
    return () => clearInterval(timerRef.current);
  }, [items.length]);

  const cur = items[idx];

  return (
    <div className='gohome-topbar'>
      <span className='gohome-topbar-badge'>🔍 宝贝回家</span>
      <span className='gohome-topbar-sep'>|</span>

      <span className={`gohome-topbar-content ${fade ? 'gohome-fade-in' : 'gohome-fade-out'}`}>
        {cur ? (
          <>
            {cur.name && <span className='gohome-topbar-name'>{cur.name}</span>}
            {cur.age && <span>· {cur.age}岁</span>}
            {cur.date && <span>· 失踪于 {cur.date}</span>}
            {cur.place && <span>· {cur.place}</span>}
          </>
        ) : (
          <span>加载中...</span>
        )}
      </span>

      {cur?.url && (
        <a
          href={cur.url}
          target='_blank'
          rel='noopener noreferrer'
          className='gohome-topbar-link'
        >
          查看详情 →
        </a>
      )}

      <span className='gohome-topbar-right'>
        如有线索请拨打&nbsp;<strong>110</strong>
      </span>
    </div>
  );
};

export default GoHomeBanner;
