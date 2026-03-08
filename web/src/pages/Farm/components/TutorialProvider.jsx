import React, { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';
import { API } from '../../../helpers';
import tutorialSteps from './tutorialSteps';
import TutorialOverlay from './TutorialOverlay';

const TutorialContext = createContext(null);

export const useTutorial = () => useContext(TutorialContext);

/**
 * TutorialProvider — 新手引导状态管理 + 渲染
 *
 * Props:
 * - userLevel:   玩家等级（用于过滤需要解锁的步骤）
 * - activePage:  当前页面 key
 * - onNavigate:  页面切换回调
 * - t:           i18n 翻译函数
 * - children
 */
const TutorialProvider = ({ userLevel, activePage, onNavigate, t, children }) => {
  const [active, setActive] = useState(false);        // 引导是否进行中
  const [currentStep, setCurrentStep] = useState(0);   // 当前步骤索引
  const [loaded, setLoaded] = useState(false);         // 是否已加载状态
  const [serverState, setServerState] = useState(null); // 服务端状态
  const initRef = useRef(false);

  // 根据玩家等级 + 功能解锁过滤步骤
  const featureLevelMap = {
    treefarm: 5,
    market: 2,
    ranch: 3,
    fish: 3,
    workshop: 4,
  };

  const filteredSteps = tutorialSteps.filter((step) => {
    if (!step.requiredFeature) return true;
    const reqLevel = featureLevelMap[step.requiredFeature];
    if (!reqLevel) return true;
    return userLevel >= reqLevel;
  });

  // 加载服务端教程状态
  const loadState = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/tutorial');
      if (res.success) {
        setServerState(res.data);
        return res.data;
      }
    } catch (e) {
      // 忽略
    }
    return null;
  }, []);

  // 首次加载 + 自动触发
  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;

    loadState().then((state) => {
      setLoaded(true);
      if (!state) return;

      if (state.needs_tutorial) {
        // 首次用户 -> 自动触发引导
        setActive(true);
        setCurrentStep(0);
      } else if (state.needs_upgrade) {
        // 教程版本升级 -> 重新触发
        setActive(true);
        setCurrentStep(0);
      } else if (!state.completed && !state.skipped && state.has_seen) {
        // 中途刷新回来 -> 恢复进度
        const savedStep = state.current_step || 0;
        const clampedStep = Math.min(savedStep, filteredSteps.length - 1);
        setActive(true);
        setCurrentStep(clampedStep);
      }
    });
  }, [loadState, filteredSteps.length]);

  // 同步进度到服务端
  const syncStep = useCallback(async (step) => {
    try {
      await API.post('/api/farm/tutorial/update', { step });
    } catch (e) {
      // 忽略
    }
  }, []);

  // 下一步
  const handleNext = useCallback(() => {
    const next = currentStep + 1;
    if (next >= filteredSteps.length) {
      // 完成
      handleFinish();
      return;
    }

    // 检查下一步所在页面是否需要切换
    const nextStep = filteredSteps[next];
    if (nextStep.page && nextStep.page !== activePage && onNavigate) {
      onNavigate(nextStep.page);
    }

    setCurrentStep(next);
    syncStep(next);
  }, [currentStep, filteredSteps, activePage, onNavigate, syncStep]);

  // 上一步
  const handlePrev = useCallback(() => {
    if (currentStep <= 0) return;
    const prev = currentStep - 1;
    const prevStep = filteredSteps[prev];
    if (prevStep.page && prevStep.page !== activePage && onNavigate) {
      onNavigate(prevStep.page);
    }
    setCurrentStep(prev);
    syncStep(prev);
  }, [currentStep, filteredSteps, activePage, onNavigate, syncStep]);

  // 跳过
  const handleSkip = useCallback(async () => {
    setActive(false);
    setCurrentStep(0);
    try {
      await API.post('/api/farm/tutorial/skip');
    } catch (e) {
      // 忽略
    }
  }, []);

  // 完成
  const handleFinish = useCallback(async () => {
    setActive(false);
    try {
      await API.post('/api/farm/tutorial/complete');
    } catch (e) {
      // 忽略
    }
  }, []);

  // 重启引导
  const restartTutorial = useCallback(async () => {
    // 回到总览页
    if (onNavigate && activePage !== 'overview') {
      onNavigate('overview');
    }
    setCurrentStep(0);
    setActive(true);
    try {
      await API.post('/api/farm/tutorial/restart');
    } catch (e) {
      // 忽略
    }
  }, [onNavigate, activePage]);

  // 当前步骤对象
  const step = filteredSteps[currentStep] || null;

  // 如果当前步骤所在页面与 activePage 不匹配，且不是 navigate 类型，暂时不渲染覆盖层
  const shouldRender = active && step && step.page === activePage;

  const contextValue = {
    active,
    currentStep,
    totalSteps: filteredSteps.length,
    restartTutorial,
    loaded,
  };

  return (
    <TutorialContext.Provider value={contextValue}>
      {children}
      {shouldRender && (
        <TutorialOverlay
          step={step}
          stepIndex={currentStep}
          totalSteps={filteredSteps.length}
          onNext={handleNext}
          onPrev={handlePrev}
          onSkip={handleSkip}
          onFinish={handleFinish}
          onNavigate={onNavigate}
          t={t}
        />
      )}
    </TutorialContext.Provider>
  );
};

export default TutorialProvider;
