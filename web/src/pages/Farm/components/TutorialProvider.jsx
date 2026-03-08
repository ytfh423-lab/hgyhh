import React, { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';
import { API } from '../../../helpers';
import { tutorialFlows, getFlowSteps, getUnlockableFlows } from './tutorialSteps';
import tutorialEvents from './tutorialEvents';
import TutorialOverlay from './TutorialOverlay';

const TutorialContext = createContext(null);

export const useTutorial = () => useContext(TutorialContext);

/**
 * TutorialProvider v2 — 状态机驱动的教程系统
 *
 * stepPhase 生命周期:
 *   idle → active → (validating → completed) → idle
 *   对 navigate 步骤：active → 等待 activePage 匹配 → completed
 *   对 wait-action 步骤：active → 等待 tutorialEvents 成功事件 → completed
 *   对 info 步骤：active → 用户点下一步 → completed
 */
const TutorialProvider = ({ userLevel, activePage, onNavigate, t, children }) => {
  const [activeFlowKey, setActiveFlowKey] = useState(null);
  const [currentStep, setCurrentStep] = useState(0);
  const [tutorialMode, setTutorialMode] = useState('forced'); // forced / replay
  const [stepPhase, setStepPhase] = useState('idle'); // idle / active / validating
  const [loaded, setLoaded] = useState(false);
  const [featuresState, setFeaturesState] = useState({});
  const initRef = useRef(false);
  const prevLevelRef = useRef(userLevel);
  const advanceTimerRef = useRef(null);

  const steps = activeFlowKey ? getFlowSteps(activeFlowKey) : [];
  const step = steps[currentStep] || null;
  const isActive = !!activeFlowKey && steps.length > 0;
  const isForced = tutorialMode === 'forced';

  // ────── 辅助：清理定时器 ──────
  const clearAdvanceTimer = () => {
    if (advanceTimerRef.current) { clearTimeout(advanceTimerRef.current); advanceTimerRef.current = null; }
  };

  // ────── 加载服务端教程状态 ──────
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

  // ────── 同步进度到服务端 ──────
  const syncStep = useCallback(async (flowKey, stepIdx) => {
    try {
      await API.post('/api/farm/tutorial/update', { feature_key: flowKey, step: stepIdx });
    } catch (e) { /* ignore */ }
  }, []);

  // ────── 启动教程流程 ──────
  const startFlow = useCallback((flowKey, mode, startStep = 0) => {
    const flowSteps = getFlowSteps(flowKey);
    if (flowSteps.length === 0) return;

    const idx = Math.min(startStep, flowSteps.length - 1);
    setActiveFlowKey(flowKey);
    setTutorialMode(mode);
    setCurrentStep(idx);
    setStepPhase('active');

    // 确保导航到第一步所在页面
    const first = flowSteps[idx];
    if (first.page && first.page !== activePage && onNavigate) {
      setTimeout(() => onNavigate(first.page), 50);
    }
  }, [activePage, onNavigate]);

  // ────── 完成教程 ──────
  const handleFinish = useCallback(async () => {
    const flowKey = activeFlowKey;
    setActiveFlowKey(null);
    setCurrentStep(0);
    setStepPhase('idle');
    clearAdvanceTimer();

    try {
      await API.post('/api/farm/tutorial/complete', { feature_key: flowKey });
    } catch (e) { /* ignore */ }

    setFeaturesState(prev => ({
      ...prev,
      [flowKey]: { ...prev[flowKey], tutorial_completed: true, tutorial_required: false },
    }));

    if (onNavigate) onNavigate('overview');

    // 检查是否有下一个待完成教程
    const state = await loadState();
    if (state?.pending_forced) {
      setTimeout(() => {
        startFlow(state.pending_forced.feature_key, 'forced', 0);
      }, 500);
    }
  }, [activeFlowKey, onNavigate, loadState, startFlow]);

  // ────── 推进到下一步 ──────
  const advanceStep = useCallback(() => {
    clearAdvanceTimer();
    if (!activeFlowKey) return;

    const nextIdx = currentStep + 1;
    if (nextIdx >= steps.length) {
      handleFinish();
      return;
    }

    const nextStep = steps[nextIdx];
    if (nextStep.page && nextStep.page !== activePage && onNavigate) {
      onNavigate(nextStep.page);
    }

    setCurrentStep(nextIdx);
    setStepPhase('active');
    syncStep(activeFlowKey, nextIdx);
  }, [activeFlowKey, currentStep, steps, activePage, onNavigate, syncStep, handleFinish]);

  // ────── 跳过/退出（仅 replay 允许）──────
  const handleSkip = useCallback(async () => {
    if (isForced) return;
    const flowKey = activeFlowKey;
    setActiveFlowKey(null);
    setCurrentStep(0);
    setStepPhase('idle');
    clearAdvanceTimer();
    try {
      await API.post('/api/farm/tutorial/skip', { feature_key: flowKey });
    } catch (e) { /* ignore */ }
  }, [activeFlowKey, isForced]);

  // ────── 手动重播教程 ──────
  const restartTutorial = useCallback(async (flowKey = 'farm_basic') => {
    if (onNavigate && activePage !== 'overview') onNavigate('overview');
    try {
      await API.post('/api/farm/tutorial/restart', { feature_key: flowKey });
    } catch (e) { /* ignore */ }
    setTimeout(() => startFlow(flowKey, 'replay', 0), 100);
  }, [onNavigate, activePage, startFlow]);

  // ────── 首次加载 + 自动触发 ──────
  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;

    loadState().then((state) => {
      setLoaded(true);
      if (!state) return;

      if (state.needs_basic_tutorial) {
        startFlow('farm_basic', 'forced', 0);
        return;
      }
      if (state.pending_forced) {
        startFlow(state.pending_forced.feature_key, 'forced', 0);
        return;
      }
      const basicState = state.features?.farm_basic;
      if (basicState && !basicState.tutorial_completed && basicState.tutorial_started) {
        startFlow('farm_basic', 'forced', 0);
      }
    });
  }, [loadState]); // eslint-disable-line react-hooks/exhaustive-deps

  // ────── 等级变化 → 检测新功能解锁教程 ──────
  useEffect(() => {
    if (!loaded || !initRef.current) return;
    const prevLevel = prevLevelRef.current;
    prevLevelRef.current = userLevel;
    if (userLevel <= prevLevel || isActive) return;

    const unlockable = getUnlockableFlows(userLevel);
    for (const flowKey of unlockable) {
      const fs = featuresState[flowKey];
      if (!fs || (!fs.tutorial_completed && !fs.tutorial_started)) {
        (async () => {
          try { await API.post('/api/farm/tutorial/unlock', { feature_key: flowKey }); } catch (e) { /* */ }
          const flow = tutorialFlows[flowKey];
          startFlow(flowKey, flow?.skippable ? 'replay' : 'forced', 0);
        })();
        break;
      }
    }
  }, [userLevel, loaded, isActive, featuresState]); // eslint-disable-line react-hooks/exhaustive-deps

  // ────── 核心：监听 wait-action 事件（含 altEvents）──────
  useEffect(() => {
    if (!isActive || !step || step.actionType !== 'wait-action' || stepPhase !== 'active') return;

    const events = [step.actionEvent, ...(step.altEvents || [])].filter(Boolean);
    if (events.length === 0) return;

    const handler = (payload) => {
      // 只在业务成功时推进
      if (payload && payload.success === false) return;
      setStepPhase('validating');
      clearAdvanceTimer();
      advanceTimerRef.current = setTimeout(() => advanceStep(), 400);
    };

    const unsubs = events.map(ev => tutorialEvents.on(`action:${ev}`, handler));
    return () => unsubs.forEach(fn => fn());
  }, [isActive, step, stepPhase, advanceStep]);

  // ────── 核心：navigate 步骤自动推进 ──────
  useEffect(() => {
    if (!isActive || !step || step.actionType !== 'navigate' || stepPhase !== 'active') return;
    if (step.navigateTo && activePage === step.navigateTo) {
      clearAdvanceTimer();
      advanceTimerRef.current = setTimeout(() => advanceStep(), 250);
    }
    return () => clearAdvanceTimer();
  }, [isActive, step, stepPhase, activePage, advanceStep]);

  // ────── 页面匹配：决定是否渲染 overlay ──────
  const shouldRender = isActive && step && (
    step.page === activePage ||
    (step.actionType === 'navigate' && step.page === activePage)
  );

  const contextValue = {
    isActive,
    activeFlowKey,
    currentStep,
    totalSteps: steps.length,
    tutorialMode,
    isForced,
    step,
    stepPhase,
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
          stepPhase={stepPhase}
          onNext={advanceStep}
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
