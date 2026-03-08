import React, { useEffect, useState, useCallback, useRef } from 'react';

/**
 * TutorialOverlay — 强制式交互教学覆盖层
 *
 * Props:
 * - step:        当前步骤配置对象
 * - stepIndex:   当前步骤索引
 * - totalSteps:  总步骤数
 * - isForced:    是否强制模式（不可跳过/关闭）
 * - onNext:      下一步回调
 * - onPrev:      上一步回调
 * - onSkip:      跳过回调（仅 replay 模式）
 * - onFinish:    完成回调
 * - onNavigate:  页面导航回调
 * - t:           i18n 翻译函数
 */
const TutorialOverlay = ({ step, stepIndex, totalSteps, isForced, onNext, onPrev, onSkip, onFinish, onNavigate, t }) => {
  const [targetRect, setTargetRect] = useState(null);
  const [tooltipStyle, setTooltipStyle] = useState({});
  const [arrowStyle, setArrowStyle] = useState({});
  const [arrowDir, setArrowDir] = useState('');
  const tooltipRef = useRef(null);
  const rafRef = useRef(null);

  const isLast = stepIndex === totalSteps - 1;
  const isFirst = stepIndex === 0;
  const isCentered = step.placement === 'center' || !step.targetSelector;
  const hasTarget = !!step.targetSelector && !!targetRect; // 遮罩挖孔独立判断
  const isWaitAction = step.actionType === 'wait-action';
  const isNavigate = step.actionType === 'navigate';
  const canManualNext = step.allowManualNext !== false;

  // 定位目标元素
  const updatePosition = useCallback(() => {
    if (!step.targetSelector) {
      setTargetRect(null);
      return;
    }
    const el = document.querySelector(step.targetSelector);
    if (!el) {
      setTargetRect(null);
      return;
    }
    const rect = el.getBoundingClientRect();
    setTargetRect({
      top: rect.top, left: rect.left,
      width: rect.width, height: rect.height,
      bottom: rect.bottom, right: rect.right,
    });
  }, [step.targetSelector]);

  useEffect(() => {
    updatePosition();
    const onResize = () => {
      cancelAnimationFrame(rafRef.current);
      rafRef.current = requestAnimationFrame(updatePosition);
    };
    window.addEventListener('resize', onResize);
    window.addEventListener('scroll', onResize, true);
    const timer = setTimeout(updatePosition, 150);
    return () => {
      window.removeEventListener('resize', onResize);
      window.removeEventListener('scroll', onResize, true);
      cancelAnimationFrame(rafRef.current);
      clearTimeout(timer);
    };
  }, [updatePosition, step.id]);

  // 计算 tooltip 位置
  useEffect(() => {
    if (isCentered || !targetRect) {
      setTooltipStyle({ position: 'fixed', top: '50%', left: '50%', transform: 'translate(-50%, -50%)' });
      setArrowDir('');
      return;
    }

    const pad = 16, arrowSize = 8;
    const tooltipEl = tooltipRef.current;
    const tw = tooltipEl ? tooltipEl.offsetWidth : 340;
    const th = tooltipEl ? tooltipEl.offsetHeight : 200;
    const vw = window.innerWidth, vh = window.innerHeight;

    let placement = step.placement || 'bottom';
    if (placement === 'bottom' && targetRect.bottom + th + pad + arrowSize > vh) placement = 'top';
    if (placement === 'top' && targetRect.top - th - pad - arrowSize < 0) placement = 'bottom';
    if (placement === 'right' && targetRect.right + tw + pad + arrowSize > vw) placement = 'left';
    if (placement === 'left' && targetRect.left - tw - pad - arrowSize < 0) placement = 'right';

    let style = {}, aStyle = {};
    const cx = targetRect.left + targetRect.width / 2;
    const cy = targetRect.top + targetRect.height / 2;

    switch (placement) {
      case 'bottom':
        style = { position: 'fixed', top: targetRect.bottom + pad + arrowSize, left: Math.max(pad, Math.min(cx - tw / 2, vw - tw - pad)) };
        aStyle = { position: 'absolute', top: -arrowSize, left: Math.min(Math.max(20, cx - (style.left || 0)), tw - 20), transform: 'translateX(-50%)' };
        break;
      case 'top':
        style = { position: 'fixed', top: targetRect.top - th - pad - arrowSize, left: Math.max(pad, Math.min(cx - tw / 2, vw - tw - pad)) };
        aStyle = { position: 'absolute', bottom: -arrowSize, left: Math.min(Math.max(20, cx - (style.left || 0)), tw - 20), transform: 'translateX(-50%)' };
        break;
      case 'right':
        style = { position: 'fixed', top: Math.max(pad, Math.min(cy - th / 2, vh - th - pad)), left: targetRect.right + pad + arrowSize };
        aStyle = { position: 'absolute', left: -arrowSize, top: Math.min(Math.max(16, cy - (style.top || 0)), th - 16), transform: 'translateY(-50%)' };
        break;
      case 'left':
        style = { position: 'fixed', top: Math.max(pad, Math.min(cy - th / 2, vh - th - pad)), left: targetRect.left - tw - pad - arrowSize };
        aStyle = { position: 'absolute', right: -arrowSize, top: Math.min(Math.max(16, cy - (style.top || 0)), th - 16), transform: 'translateY(-50%)' };
        break;
      default: break;
    }
    setTooltipStyle(style);
    setArrowStyle(aStyle);
    setArrowDir(placement);
  }, [targetRect, isCentered, step.placement, step.id]);

  // 点击"下一步"或navigate动作
  const handleClickNext = () => {
    if (isNavigate && step.navigateTo && onNavigate) {
      onNavigate(step.navigateTo);
      return; // navigate步骤由TutorialProvider的页面切换监听自动推进
    }
    if (isLast) {
      onFinish();
    } else if (canManualNext) {
      onNext();
    }
    // wait-action 且不允许手动下一步 → 不做任何事
  };

  // 遮罩：clip-path 挖孔 + 高亮边框（clip-path 同时移除渲染和pointer-events，用户可点击挖孔内元素）
  const renderMask = () => {
    if (!hasTarget) {
      return <div className="tutorial-mask tutorial-mask-full" />;
    }
    const p = 6, r = 10;
    const x = targetRect.left - p, y = targetRect.top - p;
    const w = targetRect.width + p * 2, h = targetRect.height + p * 2;

    // clip-path polygon 挖孔：从左上顺时针绕外圈，再逆时针绕内圈
    const clipPath = `polygon(
      0% 0%, 0% 100%, ${x}px 100%, ${x}px ${y}px,
      ${x + w}px ${y}px, ${x + w}px ${y + h}px, ${x}px ${y + h}px,
      ${x}px 100%, 100% 100%, 100% 0%
    )`;

    return (
      <>
        {/* 暗色遮罩（clip-path挖孔，挖孔区域无pointer-events，用户可点击） */}
        <div className="tutorial-mask" style={{
          position: 'fixed', inset: 0, zIndex: 99990,
          background: 'rgba(0,0,0,0.7)',
          pointerEvents: 'auto',
          clipPath,
          WebkitClipPath: clipPath,
        }} />
        {/* 高亮边框（不阻止点击） */}
        <div className="tutorial-highlight-border" style={{
          position: 'fixed', zIndex: 99990,
          top: y, left: x, width: w, height: h,
          borderRadius: r,
          border: '2.5px solid rgba(251,191,36,0.7)',
          pointerEvents: 'none',
        }} />
      </>
    );
  };

  // 操作提示标签
  const getActionHint = () => {
    if (isWaitAction) return `⏳ ${t('请完成上述操作后自动继续')}`;
    if (isNavigate) return `👆 ${t('请点击高亮区域')}`;
    return null;
  };

  const actionHint = getActionHint();
  const arrowClass = arrowDir ? `tutorial-arrow tutorial-arrow-${arrowDir}` : '';

  // 按钮文案
  const getNextLabel = () => {
    if (isLast) return t('完成教学');
    if (isNavigate) return `${t('前往')} →`;
    if (isWaitAction && !canManualNext) return `⏳ ${t('等待操作')}`;
    return `${t('下一步')} →`;
  };

  return (
    <div className="tutorial-overlay" onContextMenu={e => e.preventDefault()}>
      {renderMask()}

      <div className="tutorial-tooltip" ref={tooltipRef} style={tooltipStyle}>
        {arrowDir && <div className={arrowClass} style={arrowStyle} />}

        {/* 强制模式标记 */}
        {isForced && (
          <div className="tutorial-forced-badge">🔒 {t('必修教学')}</div>
        )}

        {/* 步骤指示 */}
        <div className="tutorial-step-indicator">
          <span>{stepIndex + 1} / {totalSteps}</span>
          <div className="tutorial-progress-dots">
            {Array.from({ length: totalSteps }, (_, i) => (
              <span key={i} className={`tutorial-dot ${i === stepIndex ? 'active' : i < stepIndex ? 'done' : ''}`} />
            ))}
          </div>
        </div>

        <h3 className="tutorial-title">{t(step.title)}</h3>
        <p className="tutorial-content">{t(step.content)}</p>

        {/* 操作提示 */}
        {actionHint && (
          <div className="tutorial-action-hint">{actionHint}</div>
        )}

        {/* 操作按钮 */}
        <div className="tutorial-actions">
          <div className="tutorial-actions-left">
            {/* 仅 replay 模式显示退出按钮 */}
            {!isForced && (
              <button className="tutorial-btn-skip" onClick={onSkip}>
                {t('退出教程')}
              </button>
            )}
          </div>
          <div className="tutorial-actions-right">
            {!isFirst && canManualNext && (
              <button className="tutorial-btn-prev" onClick={onPrev}>
                {t('上一步')}
              </button>
            )}
            <button
              className={`tutorial-btn-next ${isWaitAction && !canManualNext ? 'tutorial-btn-disabled' : ''}`}
              onClick={handleClickNext}
              disabled={isWaitAction && !canManualNext}
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
