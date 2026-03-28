import React, { useEffect, useState, useCallback, useRef } from 'react';

const IMG_URL = 'https://api.zjb522.cn/api?type=img&order=latest';
const JSON_URL = 'https://api.zjb522.cn/api?type=json&order=latest';

// 给图片加时间戳，每次点刷新时强制重新加载
const makeImgUrl = (seed) =>
  `${IMG_URL}&_t=${seed}`;

const GoHomeBanner = () => {
  const [info, setInfo] = useState(null);
  const [seed, setSeed] = useState(() => Date.now());
  const [imgLoaded, setImgLoaded] = useState(false);
  const imgRef = useRef(null);

  const fetchInfo = useCallback(async () => {
    try {
      const res = await fetch(JSON_URL + `&_t=${Date.now()}`);
      const data = await res.json();
      setInfo(data);
    } catch (_) {
      // CORS 或网络问题时只展示图片，忽略文字信息
      setInfo(null);
    }
  }, []);

  const refresh = useCallback(() => {
    setSeed(Date.now());
    setImgLoaded(false);
    fetchInfo();
  }, [fetchInfo]);

  useEffect(() => {
    fetchInfo();
  }, [fetchInfo]);

  const name = info?.name || info?.xing_ming || '';
  const age = info?.age || info?.nian_ling || '';
  const missingDate = info?.missing_time || info?.shi_zong_shi_jian || info?.time || '';
  const missingPlace = info?.missing_place || info?.shi_zong_di_dian || info?.place || '';
  const detailUrl = info?.baobei_url || info?.url || info?.link || 'https://www.baobeihuijia.com/';

  return (
    <div className='gohome-wrap'>
      {/* 图片区 */}
      <a
        href={detailUrl}
        target='_blank'
        rel='noopener noreferrer'
        className='gohome-img-link'
        title='点击查看详情，帮助孩子回家'
      >
        <img
          ref={imgRef}
          key={seed}
          src={makeImgUrl(seed)}
          alt='宝贝回家寻人'
          className={`gohome-photo ${imgLoaded ? 'gohome-photo-loaded' : ''}`}
          onLoad={() => setImgLoaded(true)}
          onError={() => setImgLoaded(true)}
        />
      </a>

      {/* 文字信息区 */}
      <div className='gohome-info'>
        <div className='gohome-tag'>🔍 宝贝回家 · 公益寻人</div>
        {name && <div className='gohome-name'>{name}</div>}
        <div className='gohome-fields'>
          {age && <span className='gohome-field'>年龄 {age}岁</span>}
          {missingDate && <span className='gohome-field'>失踪 {missingDate}</span>}
          {missingPlace && <span className='gohome-field'>{missingPlace}</span>}
        </div>
        <div className='gohome-actions'>
          <a
            href={detailUrl}
            target='_blank'
            rel='noopener noreferrer'
            className='gohome-detail-btn'
          >
            查看详情 →
          </a>
          <button className='gohome-next-btn' onClick={refresh} title='换一个'>
            换一个 ↺
          </button>
        </div>
        <div className='gohome-hint'>
          如有线索请拨打 <strong>110</strong> 或联系宝贝回家志愿者协会
        </div>
      </div>
    </div>
  );
};

export default GoHomeBanner;
