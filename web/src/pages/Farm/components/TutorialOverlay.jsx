import React, { useEffect, useState, useCallback, useRef } from 'react';

/**
 * TutorialOverlay — 聚焦式分步引导覆盖层
 *
 * Props:
 * - step:        当前步骤配置对象 (from tutorialSteps)
 * - stepIndex:   当前步骤索引 (0-based, 在过滤后的列表中)
 * - totalSteps:  总步骤数
 * - onNext:      下一步回调
 * - onPrev:      上一步回调
 * - onSkip:      跳过引导回调
 * - onFinish:    完成引导回调
 * - onNavigate:  页面导航回调 (pageKey) => void
 * - t:           i18n 翻译函数
 */
const TutorialOverlay = ({ step, stepIndex, totalSteps, onNext, onPrev, onSkip, onFinish, onNavigate, t }) => {
  const [targetRect, setTargetRect] = useState(null);
  const [tooltipStyle, setTooltipStyle] = useState({});
  const [arrowStyle, setArrowStyle] = useState({});
  const [arrowDir, setArrowDir] = useState('');
  const tooltipRef = useRef(null);
  const rafRef = useRef(null);

  const isLast = stepIndex === totalSteps - 1;
  const isFirst = stepIndex === 0;
  const isCentered = step.placement === 'center' || !step.targetSelector;

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
      top: rect.top,
      left: rect.left,
      width: rect.width,
      height: rect.height,
      bottom: rect.bottom,
      right: rect.right,
    });
  }, [step.targetSelector]);

  useEffect(() => {
    updatePosition();
    // 监听 resize / scroll 重新定位
    const onResize = () => {
      cancelAnimationFrame(rafRef.current);
      rafRef.current = requestAnimationFrame(updatePosition);
    };
    window.addEventListener('resize', onResize);
    window.addEventListener('scroll', onResize, true);
    // 延迟再定位一次（等 DOM 渲染完）
    const timer = setTimeout(updatePosition, 100);
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
      setTooltipStyle({
        position: 'fixed',
        top: '50%',
        left: '50%',
        transform: 'translate(-50%, -50%)',
      });
      setArrowDir('');
      return;
    }

    const pad = 16;
    const arrowSize = 8;
    const tooltipEl = tooltipRef.current;
    const tw = tooltipEl ? tooltipEl.offsetWidth : 340;
    const th = tooltipEl ? tooltipEl.offsetHeight : 200;
    const vw = window.innerWidth;
    const vh = window.innerHeight;

    let placement = step.placement || 'bottom';
    // 自适应：如果空间不足，翻转
    if (placement === 'bottom' && targetRect.bottom + th + pad + arrowSize > vh) {
      placement = 'top';
    }
    if (placement === 'top' && targetRect.top - th - pad - arrowSize < 0) {
      placement = 'bottom';
    }
    if (placement === 'right' && targetRect.right + tw + pad + arrowSize > vw) {
      placement = 'left';
    }
    if (placement === 'left' && targetRect.left - tw - pad - arrowSize < 0) {
      placement = 'right';
    }

    let style = {};
    let aStyle = {};
    const cx = targetRect.left + targetRect.width / 2;
    const cy = targetRect.top + targetRect.height / 2;

    switch (placement) {
      case 'bottom':
        style = {
          position: 'fixed',
          top: targetRect.bottom + pad + arrowSize,
          left: Math.max(pad, Math.min(cx - tw / 2, vw - tw - pad)),
        };
        aStyle = {
          position: 'absolute',
          top: -arrowSize,
          left: Math.min(Math.max(20, cx - (style.left || 0)), tw - 20),
          transform: 'translateX(-50%)',
        };
        break;
      case 'top':
        style = {
          position: 'fixed',
          top: targetRect.top - th - pad - arrowSize,
          left: Math.max(pad, Math.min(cx - tw / 2, vw - tw - pad)),
        };
        aStyle = {
          position: 'absolute',
          bottom: -arrowSize,
          left: Math.min(Math.max(20, cx - (style.left || 0)), tw - 20),
          transform: 'translateX(-50%)',
        };
        break;
      case 'right':
        style = {
          position: 'fixed',
          top: Math.max(pad, Math.min(cy - th / 2, vh - th - pad)),
          left: targetRect.right + pad + arrowSize,
        };
        aStyle = {
          position: 'absolute',
          left: -arrowSize,
          top: Math.min(Math.max(16, cy - (style.top || 0)), th - 16),
          transform: 'translateY(-50%)',
        };
        break;
      case 'left':
        style = {
          position: 'fixed',
          top: Math.max(pad, Math.min(cy - th / 2, vh - th - pad)),
          left: targetRect.left - tw - pad - arrowSize,
        };
        aStyle = {
          position: 'absolute',
          right: -arrowSize,
          top: Math.min(Math.max(16, cy - (style.top || 0)), th - 16),
          transform: 'translateY(-50%)',
        };
        break;
      default:
        break;
    }

    setTooltipStyle(style);
    setArrowStyle(aStyle);
    setArrowDir(placement);
  }, [targetRect, isCentered, step.placement, step.id]);

  const handleNext = () => {
    if (step.actionType === 'navigate' && step.navigateTo && onNavigate) {
      onNavigate(step.navigateTo);
      // 导航后延迟触发下一步
      setTimeout(() => onNext(), 300);
    } else if (isLast) {
      onFinish();
    } else {
      onNext();
    }
  };

  // 遮罩挖孔 SVG
  const renderMask = () => {
    if (isCentered || !targetRect) {
      return <div className="tutorial-mask tutorial-mask-full" />;
    }

    const p = 6; // 高亮区域的 padding
    const r = 10; // 圆角
    const x = targetRect.left - p;
    const y = targetRect.top - p;
    const w = targetRect.width + p * 2;
    const h = targetRect.height + p * 2;

    return (
      <svg className="tutorial-mask" width="100%" height="100%"
        style={{ position: 'fixed', inset: 0, zIndex: 99990 }}>
        <defs>
          <mask id="tutorial-hole">
            <rect x="0" y="0" width="100%" height="100%" fill="white" />
            <rect x={x} y={y} width={w} height={h} rx={r} ry={r} fill="black" />
          </mask>
        </defs>
        <rect x="0" y="0" width="100%" height="100%" fill="rgba(0,0,0,0.65)"
          mask="url(#tutorial-hole)" />
        {/* 高亮边框 */}
        <rect x={x} y={y} width={w} height={h} rx={r} ry={r}
          fill="none" stroke="rgba(251,191,36,0.6)" strokeWidth="2" />
      </svg>
    );
  };

  const arrowClass = arrowDir ? `tutorial-arrow tutorial-arrow-${arrowDir}` : '';

  return (
    <div className="tutorial-overlay">
      {renderMask()}

      {/* 提示框 */}
      <div className="tutorial-tooltip" ref={tooltipRef} style={tooltipStyle}>
        {arrowDir && <div className={arrowClass} style={arrowStyle} />}

        {/* 步骤指示 */}
        <div className="tutorial-step-indicator">
          <span>{stepIndex + 1} / {totalSteps}</span>
          <div className="tutorial-progress-dots">
            {Array.from({ length: totalSteps }, (_, i) => (
              <span key={i} className={`tutorial-dot ${i === stepIndex ? 'active' : i < stepIndex ? 'done' : ''}`} />
            ))}
          </div>
        </div>

        {/* 标题 */}
        <h3 className="tutorial-title">{t(step.title)}</h3>

        {/* 内容 */}
        <p className="tutorial-content">{t(step.content)}</p>

        {/* 操作按钮 */}
        <div className="tutorial-actions">
          <button className="tutorial-btn-skip" onClick={onSkip}>
            {t('跳过引导')}
          </button>
          <div className="tutorial-actions-right">
            {!isFirst && (
              <button className="tutorial-btn-prev" onClick={onPrev}>
                {t('上一步')}
              </button>
            )}
            <button className="tutorial-btn-next" onClick={handleNext}>
              {isLast ? t('完成引导') : step.actionType === 'navigate' ? `${t('前往')} →` : `${t('下一步')} →`}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TutorialOverlay;
