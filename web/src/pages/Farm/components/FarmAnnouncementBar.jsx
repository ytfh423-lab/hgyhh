import React, { useEffect, useState, useRef, useCallback } from 'react';
import { API } from './utils';

const TYPE_THEMES = {
  info:   { border: 'var(--farm-ann-border-info)',   glow: 'var(--farm-ann-glow-info)',   badge: '📢', badgeLabel: '通知', badgeClass: 'info' },
  urgent: { border: 'var(--farm-ann-border-urgent)', glow: 'var(--farm-ann-glow-urgent)', badge: '🚨', badgeLabel: '维护', badgeClass: 'urgent' },
  event:  { border: 'var(--farm-ann-border-event)',  glow: 'var(--farm-ann-glow-event)',  badge: '🎉', badgeLabel: '活动', badgeClass: 'event' },
};

const hashText = (str) => {
  let h = 0;
  for (let i = 0; i < str.length; i++) {
    h = ((h << 5) - h + str.charCodeAt(i)) | 0;
  }
  return 'farm_ann_' + (h >>> 0).toString(36);
};

const FarmAnnouncementBar = ({ t }) => {
  const [announcement, setAnnouncement] = useState(null);
  const [dismissed, setDismissed] = useState(false);
  const [visible, setVisible] = useState(false);
  const [needsMarquee, setNeedsMarquee] = useState(false);
  const textRef = useRef(null);
  const wrapRef = useRef(null);

  const load = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/announcement');
      if (res.success && res.data && res.data.enabled) {
        const key = hashText(res.data.text);
        if (sessionStorage.getItem(key)) return;
        setAnnouncement(res.data);
        setTimeout(() => setVisible(true), 100);
      }
    } catch (e) { /* ignore */ }
  }, []);

  useEffect(() => { load(); }, [load]);

  useEffect(() => {
    if (!announcement || !textRef.current || !wrapRef.current) return;
    const tw = textRef.current.scrollWidth;
    const ww = wrapRef.current.clientWidth;
    setNeedsMarquee(tw > ww - 20);
  }, [announcement]);

  const dismiss = () => {
    setVisible(false);
    setTimeout(() => setDismissed(true), 350);
    if (announcement) {
      sessionStorage.setItem(hashText(announcement.text), '1');
    }
  };

  if (dismissed || !announcement) return null;

  const theme = TYPE_THEMES[announcement.type] || TYPE_THEMES.info;

  return (
    <div className={`farm-ann-bar ${visible ? 'visible' : ''} ${theme.badgeClass}`}>
      <div className='farm-ann-inner'>
        <span className={`farm-ann-icon ${theme.badgeClass}`}>{theme.badge}</span>
        <span className={`farm-ann-badge ${theme.badgeClass}`}>{t ? t(theme.badgeLabel) : theme.badgeLabel}</span>
        <div className='farm-ann-text-wrap' ref={wrapRef}>
          <span
            ref={textRef}
            className={`farm-ann-text ${needsMarquee ? 'marquee' : ''}`}
          >
            {announcement.text}
          </span>
          {needsMarquee && (
            <span className='farm-ann-text marquee' aria-hidden='true'>
              {announcement.text}
            </span>
          )}
        </div>
        <button className='farm-ann-close' onClick={dismiss} aria-label='Close'>✕</button>
      </div>
    </div>
  );
};

export default FarmAnnouncementBar;
