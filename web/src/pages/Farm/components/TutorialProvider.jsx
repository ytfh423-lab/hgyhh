import React, { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';
import { API } from '../../../helpers';
import { tutorialFlows, getFlowSteps, getUnlockableFlows } from './tutorialSteps';
import tutorialEvents from './tutorialEvents';
import TutorialOverlay from './TutorialOverlay';

const TutorialContext = createContext(null);

export const useTutorial = () => useContext(TutorialContext);

/**
 * TutorialProvider — 强制式交互教学状态机
 *
 * 核心逻辑：
 * 1. 首次进入 → 自动触发 farm_basic 强制教程（不可跳过/关闭）
 * 2. 每解锁一个新功能 → 自动触发对应功能教程（不可跳过/关闭）
 * 3. 教程步骤中 actionType=wait-action → 监听 tutorialEvents 真实操作完成
 * 4. 教程步骤中 actionType=navigate → 监听页面切换
 * 5. 手动重播教程 → replay 模式（允许退出）
 */
const TutorialProvider = ({ userLevel, activePage, onNavigate, t, children }) => {
  const [activeFlowKey, setActiveFlowKey] = useState(null);  // 当前活跃的教程流程 key
  const [currentStep, setCurrentStep] = useState(0);          // 当前步骤索引
  const [tutorialMode, setTutorialMode] = useState('forced'); // forced / replay
  const [loaded, setLoaded] = useState(false);
  const [featuresState, setFeaturesState] = useState({});      // 后端功能教程状态
  const initRef = useRef(false);
  const prevLevelRef = useRef(userLevel);

  // 当前流程的步骤列表
  const steps = activeFlowKey ? getFlowSteps(activeFlowKey) : [];
  const step = steps[currentStep] || null;
  const isActive = !!activeFlowKey && steps.length > 0;
  const isForced = tutorialMode === 'forced';

  // ────── 加载服务端状态 ──────
  const loadState = useCallback(async () => {
    try {
      const { data: res } = await API.get('/api/farm/tutorial');
      if (res.success) {
        setFeaturesState(res.data.features || {});
        return res.data;
      }
    } catch (e) { /* ignore */ }
    return null;
  }, []);

  // ────── 首次加载 + 自动触发 ──────
  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;

    loadState().then((state) => {
      setLoaded(true);
      if (!state) return;

      // 1. 检查基础教程
      if (state.needs_basic_tutorial) {
        startFlow('farm_basic', 'forced', 0);
        return;
      }

      // 2. 检查是否有未完成的强制教程
      if (state.pending_forced) {
        const pk = state.pending_forced.feature_key;
        const ps = state.pending_forced.current_step || 0;
        startFlow(pk, 'forced', ps);
        return;
      }

      // 3. 检查基础教程进行中但未完成
      const basicState = state.features?.farm_basic;
      if (basicState && !basicState.tutorial_completed && basicState.tutorial_started) {
        startFlow('farm_basic', 'forced', basicState.current_step || 0);
      }
    });
  }, [loadState]);

  // ────── 等级变化 → 检测新功能解锁教程 ──────
  useEffect(() => {
    if (!loaded || !initRef.current) return;
    const prevLevel = prevLevelRef.current;
    prevLevelRef.current = userLevel;
    if (userLevel <= prevLevel) return;
    if (isActive) return; // 正在进行教程中，不打断

    // 找到新解锁的功能教程
    const unlockable = getUnlockableFlows(userLevel);
    for (const flowKey of unlockable) {
      const fs = featuresState[flowKey];
      if (!fs || (!fs.tutorial_completed && !fs.tutorial_started)) {
        // 该功能教程还没开始或不存在 → 触发
        triggerFeatureUnlock(flowKey);
        break;
      }
    }
  }, [userLevel, loaded, isActive, featuresState]);

  // ────── 触发功能解锁教程 ──────
  const triggerFeatureUnlock = useCallback(async (flowKey) => {
    try {
      await API.post('/api/farm/tutorial/unlock', { feature_key: flowKey });
    } catch (e) { /* ignore */ }
    startFlow(flowKey, 'forced', 0);
  }, []);

  // ────── 启动教程流程 ──────
  const startFlow = useCallback((flowKey, mode, startStep = 0) => {
    const flowSteps = getFlowSteps(flowKey);
    if (flowSteps.length === 0) return;

    const clamped = Math.min(startStep, flowSteps.length - 1);
    setActiveFlowKey(flowKey);
    setTutorialMode(mode);
    setCurrentStep(clamped);

    // 确保导航到第一步所在页面
    const firstStep = flowSteps[clamped];
    if (firstStep.page && firstStep.page !== activePage && onNavigate) {
      // 延迟导航，等状态设置完
      setTimeout(() => onNavigate(firstStep.page), 50);
    }
  }, [activePage, onNavigate]);

  // ────── 同步进度到服务端 ──────
  const syncStep = useCallback(async (flowKey, stepIdx) => {
    try {
      await API.post('/api/farm/tutorial/update', { feature_key: flowKey, step: stepIdx });
    } catch (e) { /* ignore */ }
  }, []);

  // ────── 下一步 ──────
  const handleNext = useCallback(() => {
    if (!activeFlowKey) return;

    const nextIdx = currentStep + 1;
    if (nextIdx >= steps.length) {
      handleFinish();
      return;
    }

    const nextStep = steps[nextIdx];
    // 切换页面
    if (nextStep.page && nextStep.page !== activePage && onNavigate) {
      onNavigate(nextStep.page);
    }

    setCurrentStep(nextIdx);
    syncStep(activeFlowKey, nextIdx);
  }, [activeFlowKey, currentStep, steps, activePage, onNavigate, syncStep]);

  // ────── 上一步 ──────
  const handlePrev = useCallback(() => {
    if (currentStep <= 0 || !activeFlowKey) return;
    const prevIdx = currentStep - 1;
    const prevStep = steps[prevIdx];
    if (prevStep.page && prevStep.page !== activePage && onNavigate) {
      onNavigate(prevStep.page);
    }
    setCurrentStep(prevIdx);
    syncStep(activeFlowKey, prevIdx);
  }, [activeFlowKey, currentStep, steps, activePage, onNavigate, syncStep]);

  // ────── 完成教程 ──────
  const handleFinish = useCallback(async () => {
    const flowKey = activeFlowKey;
    setActiveFlowKey(null);
    setCurrentStep(0);
    try {
      await API.post('/api/farm/tutorial/complete', { feature_key: flowKey });
    } catch (e) { /* ignore */ }

    // 更新本地状态
    setFeaturesState(prev => ({
      ...prev,
      [flowKey]: { ...prev[flowKey], tutorial_completed: true, tutorial_required: false },
    }));

    // 完成后导航回总览
    if (onNavigate) onNavigate('overview');

    // 重新加载状态，检查是否有下一个待完成教程
    const state = await loadState();
    if (state?.pending_forced) {
      setTimeout(() => {
        startFlow(state.pending_forced.feature_key, 'forced', state.pending_forced.current_step || 0);
      }, 500);
    }
  }, [activeFlowKey, onNavigate, loadState, startFlow]);

  // ────── 跳过/退出教程（仅 replay 模式允许）──────
  const handleSkip = useCallback(async () => {
    if (isForced) return; // 强制模式不允许跳过

    const flowKey = activeFlowKey;
    setActiveFlowKey(null);
    setCurrentStep(0);
    try {
      await API.post('/api/farm/tutorial/skip', { feature_key: flowKey });
    } catch (e) { /* ignore */ }
  }, [activeFlowKey, isForced]);

  // ────── 手动重播教程 ──────
  const restartTutorial = useCallback(async (flowKey = 'farm_basic') => {
    if (onNavigate && activePage !== 'overview') {
      onNavigate('overview');
    }
    try {
      await API.post('/api/farm/tutorial/restart', { feature_key: flowKey });
    } catch (e) { /* ignore */ }
    // 延迟启动等页面切换完
    setTimeout(() => startFlow(flowKey, 'replay', 0), 100);
  }, [onNavigate, activePage, startFlow]);

  // ────── 监听 tutorialEvents 推进 wait-action 步骤 ──────
  useEffect(() => {
    if (!isActive || !step || step.actionType !== 'wait-action') return;

    const eventName = step.actionEvent;
    if (!eventName) return;

    const unsubscribe = tutorialEvents.on(`action:${eventName}`, () => {
      // 动作完成，自动推进下一步
      setTimeout(() => handleNext(), 300);
    });

    return unsubscribe;
  }, [isActive, step, handleNext]);

  // ────── 监听页面切换推进 navigate 步骤 ──────
  useEffect(() => {
    if (!isActive || !step || step.actionType !== 'navigate') return;
    if (step.navigateTo && activePage === step.navigateTo) {
      // 已经到达目标页面
      setTimeout(() => handleNext(), 200);
    }
  }, [isActive, step, activePage, handleNext]);

  // ────── 当前步骤页面匹配检查 ──────
  const shouldRender = isActive && step && (step.page === activePage || step.actionType === 'navigate');

  const contextValue = {
    isActive,
    activeFlowKey,
    currentStep,
    totalSteps: steps.length,
    tutorialMode,
    isForced,
    step,
    restartTutorial,
    loaded,
    featuresState,
    tutorialFlows,
  };

  return (
    <TutorialContext.Provider value={contextValue}>
      {children}
      {shouldRender && (
        <TutorialOverlay
          step={step}
          stepIndex={currentStep}
          totalSteps={steps.length}
          isForced={isForced}
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
