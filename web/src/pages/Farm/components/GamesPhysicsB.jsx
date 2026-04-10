import React, { useEffect, useRef, useState } from 'react';
import { ChoiceButton, Pill, ReadyPanel, ResultPanel, StatRow, pickOne, randInt, shuffle } from './GamesCommon';

const panelStyle = {
  width: '100%',
  maxWidth: 340,
  margin: '0 auto',
  padding: 12,
  borderRadius: 16,
  border: '2px solid var(--farm-border)',
  background: 'var(--farm-surface-alt)',
};

export const TameGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [round, setRound] = useState(1);
  const [cue, setCue] = useState('left');
  const [countdown, setCountdown] = useState(2);
  const [correct, setCorrect] = useState(0);
  const cueRef = useRef('left');
  const correctRef = useRef(0);
  const timeoutRef = useRef(null);
  const tickRef = useRef(null);
  const totalRounds = 10;

  const clearAll = () => {
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    if (tickRef.current) clearInterval(tickRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = () => {
    clearAll();
    const score = Math.round((correctRef.current / totalRounds) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const nextCue = (nextRound) => {
    clearAll();
    if (nextRound > totalRounds) return finish();
    const next = pickOne(['left', 'right', 'hold']);
    cueRef.current = next;
    setRound(nextRound);
    setCue(next);
    setCountdown(2);
    setPhase('playing');
    tickRef.current = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          clearAll();
          timeoutRef.current = setTimeout(() => nextCue(nextRound + 1), 160);
          return 0;
        }
        return prev - 1;
      });
    }, 700);
  };

  const start = () => {
    correctRef.current = 0;
    setCorrect(0);
    nextCue(1);
  };

  const choose = (action) => {
    if (phase !== 'playing') return;
    if (action === cueRef.current) {
      correctRef.current += 1;
      setCorrect(correctRef.current);
    }
    clearAll();
    timeoutRef.current = setTimeout(() => nextCue(round + 1), 150);
  };

  const cueText = cue === 'left' ? t('压左侧') : cue === 'right' ? t('压右侧') : t('拉紧缰绳');
  const cueEmoji = cue === 'left' ? '↙️' : cue === 'right' ? '↘️' : '⬆️';

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🐴 ${t('看野马动作，立刻做出正确驯服动作')}`} hint={`10 ${t('轮')} · ${t('压左 / 压右 / 拉缰')} · ${t('稳定越高分越高')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const score = Math.round((correct / totalRounds) * 100);
    return <ResultPanel emoji={score >= 80 ? '🏇' : score >= 50 ? '🐴' : '🤕'} title={`${t('驯服成功')} ${correct}/${totalRounds}`} detail={`${t('得分')} ${score}`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${t('第')} ${round}/${totalRounds} ${t('轮')}`} right={`${countdown}s · ✅ ${correct}`} />
      <div style={{ ...panelStyle, textAlign: 'center' }}>
        <div style={{ fontSize: 18, color: 'var(--farm-text-2)', marginBottom: 8 }}>{t('当前动作')}</div>
        <div style={{ fontSize: 40, marginBottom: 8 }}>{cueEmoji}</div>
        <div style={{ fontSize: 22, fontWeight: 800 }}>{cueText}</div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 8, width: '100%', maxWidth: 340 }}>
        <ChoiceButton onClick={() => choose('left')}>↙️ {t('左压')}</ChoiceButton>
        <ChoiceButton onClick={() => choose('hold')}>⬆️ {t('拉缰')}</ChoiceButton>
        <ChoiceButton onClick={() => choose('right')}>↘️ {t('右压')}</ChoiceButton>
      </div>
    </div>
  );
};

const weatherCards = [
  { key: 'sunny', emoji: '☀️', label: '晴朗', ideal: ['wheat', 'grape'] },
  { key: 'cloudy', emoji: '⛅', label: '多云', ideal: ['pumpkin', 'corn'] },
  { key: 'rainy', emoji: '🌧️', label: '降雨', ideal: ['rice', 'mushroom'] },
  { key: 'storm', emoji: '⛈️', label: '雷暴', ideal: ['windmill'] },
  { key: 'rainbow', emoji: '🌈', label: '彩虹', ideal: ['tour'] },
];

const cropCards = [
  { key: 'wheat', emoji: '🌾', label: '麦田' },
  { key: 'grape', emoji: '🍇', label: '葡萄园' },
  { key: 'pumpkin', emoji: '🎃', label: '南瓜地' },
  { key: 'corn', emoji: '🌽', label: '玉米田' },
  { key: 'rice', emoji: '🍚', label: '水田' },
  { key: 'mushroom', emoji: '🍄', label: '菌棚' },
  { key: 'windmill', emoji: '🌬️', label: '风车区' },
  { key: 'tour', emoji: '🎡', label: '观光园' },
];

export const WeatherGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [roundIndex, setRoundIndex] = useState(0);
  const [question, setQuestion] = useState(null);
  const [correct, setCorrect] = useState(0);
  const questionsRef = useRef([]);
  const correctRef = useRef(0);
  const totalRounds = 5;

  const finish = () => {
    const score = Math.round((correctRef.current / totalRounds) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const pushRound = (index) => {
    if (index >= totalRounds) return finish();
    setRoundIndex(index);
    setQuestion(questionsRef.current[index]);
    setPhase('playing');
  };

  const start = () => {
    const rounds = shuffle(weatherCards)
      .slice(0, totalRounds)
      .map((weather) => {
        const options = shuffle(
          cropCards.filter((item) => weather.ideal.includes(item.key)).concat(
            shuffle(cropCards.filter((item) => !weather.ideal.includes(item.key))).slice(0, 2),
          ),
        ).slice(0, 4);
        return { weather, options };
      });
    questionsRef.current = rounds;
    correctRef.current = 0;
    setCorrect(0);
    pushRound(0);
  };

  const choose = (key) => {
    if (!question) return;
    if (question.weather.ideal.includes(key)) {
      correctRef.current += 1;
      setCorrect(correctRef.current);
    }
    setTimeout(() => pushRound(roundIndex + 1), 180);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🌈 ${t('判断天气最适合哪块区域，做出更合理安排')}`} hint={`5 ${t('轮')} · ${t('选对天气对应区域')} · ${t('策略预测玩法')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const score = Math.round((correct / totalRounds) * 100);
    return <ResultPanel emoji={score >= 80 ? '🛰️' : score >= 50 ? '🌦️' : '🌫️'} title={`${t('预测准确')} ${correct}/${totalRounds}`} detail={`${t('得分')} ${score}`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${t('第')} ${roundIndex + 1}/${totalRounds} ${t('轮')}`} right={`✅ ${correct}`} />
      <div style={{ ...panelStyle, textAlign: 'center' }}>
        <div style={{ fontSize: 42, marginBottom: 8 }}>{question.weather.emoji}</div>
        <div style={{ fontSize: 22, fontWeight: 800 }}>{t(question.weather.label)}</div>
        <div style={{ marginTop: 8 }}><Pill tone='blue'>{t('请选择最适合安排作业的区域')}</Pill></div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 8, width: '100%', maxWidth: 340 }}>
        {question.options.map((option) => (
          <ChoiceButton key={option.key} onClick={() => choose(option.key)} style={{ minHeight: 64 }}>
            {option.emoji} {t(option.label)}
          </ChoiceButton>
        ))}
      </div>
    </div>
  );
};

export const SheepdogGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [round, setRound] = useState(1);
  const [targetLane, setTargetLane] = useState(1);
  const [dogLane, setDogLane] = useState(1);
  const [saved, setSaved] = useState(0);
  const [sheepIcons, setSheepIcons] = useState(['🐑', '🐑', '🐑']);
  const totalRounds = 8;
  const dogLaneRef = useRef(1);
  const savedRef = useRef(0);
  const timerRef = useRef(null);

  const clearAll = () => {
    if (timerRef.current) clearTimeout(timerRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = () => {
    clearAll();
    const score = Math.round((savedRef.current / totalRounds) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const loadRound = (nextRound) => {
    clearAll();
    if (nextRound > totalRounds) return finish();
    const lane = randInt(0, 2);
    setRound(nextRound);
    setTargetLane(lane);
    dogLaneRef.current = 1;
    setDogLane(1);
    setSheepIcons([0, 1, 2].map((idx) => (idx === lane ? pickOne(['🐑', '🐏']) : '🌿')));
    setPhase('playing');
    timerRef.current = setTimeout(() => {
      if (dogLaneRef.current === lane) {
        savedRef.current += 1;
        setSaved(savedRef.current);
      }
      loadRound(nextRound + 1);
    }, 900);
  };

  const start = () => {
    savedRef.current = 0;
    setSaved(0);
    loadRound(1);
  };

  const moveDog = (lane) => {
    if (phase !== 'playing') return;
    dogLaneRef.current = lane;
    setDogLane(lane);
    if (lane === targetLane) {
      savedRef.current += 1;
      setSaved(savedRef.current);
      loadRound(round + 1);
    }
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🐕 ${t('把牧羊犬派到跑偏羊所在的通道，及时赶回羊群')}`} hint={`8 ${t('轮')} · ${t('三路选位')} · ${t('反应越快越好')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const score = Math.round((saved / totalRounds) * 100);
    return <ResultPanel emoji={score >= 80 ? '🏡' : score >= 50 ? '🐕' : '🐾'} title={`${t('赶回羊群')} ${saved}/${totalRounds}`} detail={`${t('得分')} ${score}`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${t('第')} ${round}/${totalRounds} ${t('轮')}`} right={`🐑 ${saved}`} />
      <div style={{ ...panelStyle, display: 'flex', flexDirection: 'column', gap: 8 }}>
        {[0, 1, 2].map((lane) => (
          <div key={lane} style={{ position: 'relative', height: 52, borderRadius: 12, background: lane === targetLane ? 'rgba(74,124,63,0.08)' : 'var(--farm-surface)' }}>
            <div style={{ position: 'absolute', left: 12, top: 10, fontSize: 28 }}>{dogLane === lane ? '🐕' : '🏁'}</div>
            <div style={{ position: 'absolute', right: 12, top: 10, fontSize: 28 }}>{sheepIcons[lane]}</div>
            <div style={{ position: 'absolute', inset: '0 62px 0 62px', display: 'flex', alignItems: 'center' }}>
              <div style={{ width: '100%', borderTop: '2px dashed var(--farm-border)' }} />
            </div>
          </div>
        ))}
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        {[0, 1, 2].map((lane) => (
          <ChoiceButton key={lane} active={dogLane === lane} onClick={() => moveDog(lane)}>{t('通道')} {lane + 1}</ChoiceButton>
        ))}
      </div>
    </div>
  );
};
