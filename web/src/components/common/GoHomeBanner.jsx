import React, { useEffect, useState, useCallback } from 'react';

const API_URL = 'https://api.zjb522.cn/api?type=json&order=latest';

const GoHomeBanner = ({ variant = 'bar' }) => {
  const [child, setChild] = useState(null);
  const [loading, setLoading] = useState(true);
  const [imgErr, setImgErr] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setImgErr(false);
      const res = await fetch(API_URL);
      const data = await res.json();
      setChild(data);
    } catch (_) {
      setChild(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  if (loading || !child) return null;

  const name = child.name || child.xing_ming || '';
  const age = child.age || child.nian_ling || child.shi_zong_shi_nian_ling || '';
  const missingDate = child.missing_time || child.shi_zong_shi_jian || child.time || '';
  const missingPlace = child.missing_place || child.shi_zong_di_dian || child.place || '';
  const detailUrl = child.baobei_url || child.url || child.link || 'https://www.baobeihuijia.com/';
  const imgUrl = child.img || child.image || child.pic || '';

  // 横向通知条（首页顶部和控制台顶部共用）
  if (variant === 'bar') {
    return (
      <a
        href={detailUrl}
        target='_blank'
        rel='noopener noreferrer'
        className='gohome-bar'
        title='点击查看详情 · 帮助失踪儿童回家'
      >
        <span className='gohome-bar-icon'>🔍</span>
        <span className='gohome-bar-label'>宝贝回家公益寻人</span>
        <span className='gohome-bar-sep'>|</span>
        {imgUrl && !imgErr && (
          <img
            className='gohome-bar-photo'
            src={imgUrl}
            alt={name}
            onError={() => setImgErr(true)}
          />
        )}
        {name && <span className='gohome-bar-name'>{name}</span>}
        {age && <span className='gohome-bar-field'>{age}岁</span>}
        {missingDate && <span className='gohome-bar-field'>失踪于 {missingDate}</span>}
        {missingPlace && <span className='gohome-bar-field'>{missingPlace}</span>}
        <span className='gohome-bar-cta'>查看详情 →</span>
        <span className='gohome-bar-refresh' onClick={(e) => { e.preventDefault(); fetchData(); }} title='换一条'>
          ↺
        </span>
      </a>
    );
  }

  // 卡片模式（控制台内嵌卡片）
  return (
    <div className='gohome-card'>
      <div className='gohome-card-header'>
        <span className='gohome-card-icon'>🔍</span>
        <span className='gohome-card-title'>宝贝回家 · 公益寻人</span>
        <button
          className='gohome-card-refresh'
          onClick={fetchData}
          title='换一条'
        >
          ↺
        </button>
      </div>
      <div className='gohome-card-body'>
        {imgUrl && !imgErr && (
          <img
            className='gohome-card-photo'
            src={imgUrl}
            alt={name}
            onError={() => setImgErr(true)}
          />
        )}
        <div className='gohome-card-info'>
          {name && <div className='gohome-card-name'>{name}</div>}
          {age && <div className='gohome-card-field'>年龄：{age}岁</div>}
          {missingDate && <div className='gohome-card-field'>失踪时间：{missingDate}</div>}
          {missingPlace && <div className='gohome-card-field'>失踪地点：{missingPlace}</div>}
        </div>
      </div>
      <a
        href={detailUrl}
        target='_blank'
        rel='noopener noreferrer'
        className='gohome-card-link'
      >
        查看详情，帮助 {name || '孩子'} 回家 →
      </a>
    </div>
  );
};

export default GoHomeBanner;
