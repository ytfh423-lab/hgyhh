import React, { useEffect, useState, useCallback, useRef } from 'react';

/**
 * TutorialOverlay v2 — 教程覆盖层
 *
 * 核心改进：
 * 1. clip-path 挖孔保证目标区域可点击（pointer-events 穿透）
 * 2. MutationObserver + 轮询 等待目标 DOM 出现
 * 3. 自动滚动目标到可视区域
 * 4. tooltip 位置自适应，不遮挡目标
 * 5. 根据 stepPhase 显示不同状态（waiting_target / active / validating）
 */
const TutorialOverlay = ({
  step, stepIndex, totalSteps,
  isForced, stepPhase,
  onNext, onSkip, onFinish, onNavigate, t,
}) => {
  const [targetRect, setTargetRect] = useState(null);
  const [tooltipPos, setTooltipPos] = useState({});
  const [arrowStyle, setArrowStyle] = useState({});
  const [arrowDir, setArrowDir] = useState('');
  const tooltipRef = useRef(null);
  const rafRef = useRef(null);
  const observerRef = useRef(null);
  const pollRef = useRef(null);

  const isLast = stepIndex === totalSteps - 1;
  const isInfo = step.actionType === 'info';
  const isNav = step.actionType === 'navigate';
  const isWait = step.actionType === 'wait-action';
  const hasSelector = !!step.targetSelector;
  const hasTarget = hasSelector && !!targetRect;

  // ─── 查找 & 测量目标元素 ───
  const measureTarget = useCallback(() => {
    if (!step.targetSelector) { setTargetRect(null); return null; }
    const el = document.querySelector(step.targetSelector);
    if (!el || el.offsetWidth === 0) { setTargetRect(null); return null; }
    const r = el.getBoundingClientRect();
    const rect = { top: r.top, left: r.left, width: r.width, height: r.height, bottom: r.bottom, right: r.right };
    setTargetRect(rect);
    return el;
  }, [step.targetSelector]);

  // ─── 自动滚动目标到可视区域 ───
  const scrollToTarget = useCallback((el) => {
    if (!el) return;
    const r = el.getBoundingClientRect();
    const inView = r.top >= 0 && r.bottom <= window.innerHeight;
    if (!inView) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      // 滚动后重新测量
      setTimeout(() => measureTarget(), 350);
    }
  }, [measureTarget]);

  // ─── 目标等待：Observer + 轮询 ───
  useEffect(() => {
    // 清理旧的
    if (observerRef.current) { observerRef.current.disconnect(); observerRef.current = null; }
    if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null; }

    if (!step.targetSelector) { setTargetRect(null); return; }

    // 立即尝试
    const el = measureTarget();
    if (el) scrollToTarget(el);

    // 轮询兜底（200ms 间隔，最多 15 秒）
    let elapsed = 0;
    pollRef.current = setInterval(() => {
      elapsed += 200;
      const found = measureTarget();
      if (found) {
        scrollToTarget(found);
        clearInterval(pollRef.current);
        pollRef.current = null;
      }
      if (elapsed > 15000) {
        clearInterval(pollRef.current);
        pollRef.current = null;
      }
    }, 200);

    // MutationObserver 快速检测
    observerRef.current = new MutationObserver(() => {
      const found = measureTarget();
      if (found) {
        scrollToTarget(found);
        if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null; }
      }
    });
    observerRef.current.observe(document.body, { childList: true, subtree: true, attributes: true });

    return () => {
      if (observerRef.current) { observerRef.current.disconnect(); observerRef.current = null; }
      if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null; }
    };
  }, [step.id, step.targetSelector]); // eslint-disable-line react-hooks/exhaustive-deps

  // ─── resize / scroll 时更新位置 ───
  useEffect(() => {
    const update = () => {
      cancelAnimationFrame(rafRef.current);
      rafRef.current = requestAnimationFrame(measureTarget);
    };
    window.addEventListener('resize', update);
    window.addEventListener('scroll', update, true);
    return () => {
      window.removeEventListener('resize', update);
      window.removeEventListener('scroll', update, true);
      cancelAnimationFrame(rafRef.current);
    };
  }, [measureTarget]);

  // ─── tooltip 位置计算 ───
  useEffect(() => {
    const centered = step.placement === 'center' || !hasTarget;
    if (centered) {
      setTooltipPos({ position: 'fixed', top: '50%', left: '50%', transform: 'translate(-50%, -50%)' });
      setArrowDir('');
      return;
    }

    const pad = 16, arrSz = 8;
    const el = tooltipRef.current;
    const tw = el ? el.offsetWidth : 340;
    const th = el ? el.offsetHeight : 200;
    const vw = window.innerWidth, vh = window.innerHeight;

    let pl = step.placement || 'bottom';
    if (pl === 'bottom' && targetRect.bottom + th + pad + arrSz > vh) pl = 'top';
    if (pl === 'top' && targetRect.top - th - pad - arrSz < 0) pl = 'bottom';
    if (pl === 'right' && targetRect.right + tw + pad + arrSz > vw) pl = 'left';
    if (pl === 'left' && targetRect.left - tw - pad - arrSz < 0) pl = 'right';

    const cx = targetRect.left + targetRect.width / 2;
    const cy = targetRect.top + targetRect.height / 2;
    let s = {}, a = {};

    switch (pl) {
      case 'bottom':
        s = { position: 'fixed', top: Math.min(targetRect.bottom + pad + arrSz, vh - th - pad), left: Math.max(pad, Math.min(cx - tw / 2, vw - tw - pad)) };
        a = { position: 'absolute', top: -arrSz, left: Math.min(Math.max(20, cx - (s.left || 0)), tw - 20), transform: 'translateX(-50%)' };
        break;
      case 'top':
        s = { position: 'fixed', top: Math.max(pad, targetRect.top - th - pad - arrSz), left: Math.max(pad, Math.min(cx - tw / 2, vw - tw - pad)) };
        a = { position: 'absolute', bottom: -arrSz, left: Math.min(Math.max(20, cx - (s.left || 0)), tw - 20), transform: 'translateX(-50%)' };
        break;
      case 'right':
        s = { position: 'fixed', top: Math.max(pad, Math.min(cy - th / 2, vh - th - pad)), left: targetRect.right + pad + arrSz };
        a = { position: 'absolute', left: -arrSz, top: Math.min(Math.max(16, cy - (s.top || 0)), th - 16), transform: 'translateY(-50%)' };
        break;
      case 'left':
        s = { position: 'fixed', top: Math.max(pad, Math.min(cy - th / 2, vh - th - pad)), left: targetRect.left - tw - pad - arrSz };
        a = { position: 'absolute', right: -arrSz, top: Math.min(Math.max(16, cy - (s.top || 0)), th - 16), transform: 'translateY(-50%)' };
        break;
      default: break;
    }
    setTooltipPos(s);
    setArrowStyle(a);
    setArrowDir(pl);
  }, [targetRect, hasTarget, step.placement, step.id]);

  // ─── 点击处理 ───
  const handleClickNext = () => {
    if (isNav && step.navigateTo && onNavigate) {
      onNavigate(step.navigateTo);
      return;
    }
    if (isLast) { onFinish(); return; }
    if (isInfo) { onNext(); return; }
    // wait-action: 按钮被 disabled，不会走到这
  };

  // ─── 遮罩渲染：clip-path 挖孔 ───
  const renderMask = () => {
    if (!hasTarget) {
      return (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 99990,
          background: 'rgba(0,0,0,0.65)',
          pointerEvents: 'auto',
        }} />
      );
    }
    const pad = 6;
    const x = targetRect.left - pad, y = targetRect.top - pad;
    const w = targetRect.width + pad * 2, h = targetRect.height + pad * 2;

    const clipPath = `polygon(
      0% 0%, 0% 100%, ${x}px 100%, ${x}px ${y}px,
      ${x + w}px ${y}px, ${x + w}px ${y + h}px, ${x}px ${y + h}px,
      ${x}px 100%, 100% 100%, 100% 0%
    )`;

    return (
      <>
        <div style={{
          position: 'fixed', inset: 0, zIndex: 99990,
          background: 'rgba(0,0,0,0.65)',
          pointerEvents: 'auto',
          clipPath, WebkitClipPath: clipPath,
        }} />
        <div style={{
          position: 'fixed', zIndex: 99990, pointerEvents: 'none',
          top: y, left: x, width: w, height: h,
          borderRadius: 10,
          border: '2.5px solid rgba(251,191,36,0.7)',
          boxShadow: '0 0 0 4px rgba(251,191,36,0.15), 0 0 20px rgba(251,191,36,0.2)',
          transition: 'all 0.3s ease',
        }} />
      </>
    );
  };

  // ─── 按钮文案 ───
  const getNextLabel = () => {
    if (isLast) return t('完成教学');
    if (isNav) return `${t('前往')} →`;
    if (isWait) return `⏳ ${t('等待操作')}`;
    return `${t('下一步')} →`;
  };

  const isPending = stepPhase === 'pending';
  const displayContent = isPending && step.pendingContent ? step.pendingContent : step.content;

  const getHint = () => {
    if (isPending) return `⏳ ${t('等待条件满足...')}`;
    if (stepPhase === 'waiting_target') return `⏳ ${t('等待页面加载...')}`;
    if (stepPhase === 'validating') return `⏳ ${t('正在验证...')}`;
    if (isWait) return `👆 ${t('请完成上方高亮区域的操作')}`;
    if (isNav) return `👆 ${t('请点击高亮区域')}`;
    return null;
  };

  const hint = getHint();
  const arrowClass = arrowDir ? `tutorial-arrow tutorial-arrow-${arrowDir}` : '';
  const nextDisabled = isWait || isPending; // wait-action 和 pending 步骤不能手动跳过

  return (
    <div className="tutorial-overlay" style={{ position: 'fixed', inset: 0, zIndex: 99989 }}
      onContextMenu={e => e.preventDefault()}>
      {renderMask()}

      <div className="tutorial-tooltip" ref={tooltipRef}
        style={{ ...tooltipPos, zIndex: 99995, pointerEvents: 'auto' }}>
        {arrowDir && <div className={arrowClass} style={arrowStyle} />}

        {isForced && <div className="tutorial-forced-badge">🔒 {t('必修教学')}</div>}

        <div className="tutorial-step-indicator">
          <span>{stepIndex + 1} / {totalSteps}</span>
          <div className="tutorial-progress-dots">
            {Array.from({ length: totalSteps }, (_, i) => (
              <span key={i} className={`tutorial-dot ${i === stepIndex ? 'active' : i < stepIndex ? 'done' : ''}`} />
            ))}
          </div>
        </div>

        <h3 className="tutorial-title">{t(step.title)}</h3>
        <p className="tutorial-content">{t(displayContent)}</p>

        {hint && <div className="tutorial-action-hint">{hint}</div>}

        <div className="tutorial-actions">
          <div className="tutorial-actions-left">
            {!isForced && (
              <button className="tutorial-btn-skip" onClick={onSkip}>{t('退出教程')}</button>
            )}
          </div>
          <div className="tutorial-actions-right">
            <button
              className={`tutorial-btn-next ${nextDisabled ? 'tutorial-btn-disabled' : ''}`}
              onClick={handleClickNext}
              disabled={nextDisabled}
            >
              {getNextLabel()}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TutorialOverlay;
