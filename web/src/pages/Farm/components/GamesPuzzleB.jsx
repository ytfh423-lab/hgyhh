import React, { useRef, useState } from 'react';
import { clamp, randInt, ReadyPanel, ResultPanel, StatRow, shuffle } from './GamesCommon';

const panelStyle = {
  width: '100%',
  maxWidth: 340,
  margin: '0 auto',
  padding: 12,
  borderRadius: 16,
  border: '2px solid var(--farm-border)',
  background: 'var(--farm-surface-alt)',
};

export const PumpkinGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [round, setRound] = useState(1);
  const [target, setTarget] = useState(18);
  const [cards, setCards] = useState([]);
  const [picked, setPicked] = useState([]);
  const [scores, setScores] = useState([]);
  const pickedRef = useRef([]);
  const cardsRef = useRef([]);
  const totalRounds = 3;

  const finish = (values) => {
    const avg = Math.round(values.reduce((sum, val) => sum + val, 0) / values.length);
    setPhase('done');
    onComplete(avg / 100, avg);
  };

  const beginRound = (nextRound, baseScores = scores) => {
    if (nextRound > totalRounds) {
      finish(baseScores.length ? baseScores : [0]);
      return;
    }
    const targetWeight = randInt(15, 24);
    const values = shuffle([
      randInt(4, 10), randInt(4, 10), randInt(5, 11),
      randInt(5, 11), randInt(6, 12), randInt(3, 9),
    ]);
    cardsRef.current = values;
    pickedRef.current = [];
    setRound(nextRound);
    setTarget(targetWeight);
    setCards(values);
    setPicked([]);
    setPhase('playing');
  };

  const start = () => {
    setScores([]);
    beginRound(1, []);
  };

  const togglePick = (index) => {
    if (phase !== 'playing') return;
    const exists = pickedRef.current.includes(index);
    const nextPicked = exists
      ? pickedRef.current.filter((item) => item !== index)
      : [...pickedRef.current, index];
    if (nextPicked.length > 3) return;
    pickedRef.current = nextPicked;
    setPicked(nextPicked);
  };

  const submit = () => {
    if (phase !== 'playing') return;
    const sum = pickedRef.current.reduce((acc, index) => acc + cardsRef.current[index], 0);
    const diff = Math.abs(sum - target);
    const roundScore = Math.round(clamp(1 - diff / 12, 0, 1) * 100);
    const nextScores = [...scores, roundScore];
    setScores(nextScores);
    setTimeout(() => {
      if (round >= totalRounds) finish(nextScores);
      else beginRound(round + 1, nextScores);
    }, 260);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🎃 ${t('从 6 个南瓜里挑最多 3 个，让总重量最接近目标')}`} hint={`3 ${t('轮')} · ${t('越接近目标分越高')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const avg = Math.round((scores.reduce((sum, val) => sum + val, 0) / scores.length) || 0);
    return <ResultPanel emoji={avg >= 80 ? '🏆' : avg >= 50 ? '🎃' : '🥲'} title={`${t('培育得分')}: ${avg}`} detail={scores.map((score, index) => `R${index + 1}:${score}`).join(' · ')} />;
  }

  const currentSum = picked.reduce((acc, index) => acc + cards[index], 0);

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${t('第')} ${round}/${totalRounds} ${t('轮')}`} right={`${t('目标')} ${target}kg · ${t('当前')} ${currentSum}kg`} />
      <div style={{ ...panelStyle, display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
        {cards.map((value, index) => {
          const active = picked.includes(index);
          return (
            <button
              key={`${round}-${index}`}
              type='button'
              onClick={() => togglePick(index)}
              style={{
                minHeight: 72,
                borderRadius: 16,
                border: `2px solid ${active ? 'var(--farm-harvest)' : 'var(--farm-border)'}`,
                background: active ? 'rgba(200,146,42,0.14)' : 'var(--farm-surface)',
                cursor: 'pointer',
                fontWeight: 700,
              }}
            >
              <div style={{ fontSize: 28 }}>🎃</div>
              <div>{value}kg</div>
            </button>
          );
        })}
      </div>
      <button type='button' onClick={submit} style={{ minWidth: 160, minHeight: 44, borderRadius: 14, border: '2px solid var(--farm-border)', background: 'var(--farm-surface)', fontWeight: 700, cursor: 'pointer' }}>
        {t('提交重量组合')}
      </button>
    </div>
  );
};

const buildProduceRound = () => {
  const rules = [
    { key: 'sweet', label: '甜度', icon: '🍯' },
    { key: 'fresh', label: '新鲜', icon: '💧' },
    { key: 'looks', label: '外观', icon: '✨' },
    { key: 'balance', label: '均衡', icon: '⚖️' },
  ];
  const rule = rules[randInt(0, rules.length - 1)];
  const items = Array.from({ length: 4 }, (_, index) => ({
    id: index,
    emoji: ['🥕', '🍅', '🍆', '🌽'][index],
    sweet: randInt(1, 5),
    fresh: randInt(1, 5),
    looks: randInt(1, 5),
  }));
  const scoreFor = (item) => {
    if (rule.key === 'balance') return item.sweet + item.fresh + item.looks;
    return item.sweet + item.fresh + item.looks + item[rule.key] * 2;
  };
  const values = items.map(scoreFor);
  const bestIndex = values.indexOf(Math.max(...values));
  return { rule, items, bestIndex };
};

export const ProduceGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [round, setRound] = useState(1);
  const [rule, setRule] = useState(null);
  const [items, setItems] = useState([]);
  const [scores, setScores] = useState([]);
  const bestRef = useRef(0);
  const totalRounds = 4;

  const finish = (values) => {
    const avg = Math.round(values.reduce((sum, val) => sum + val, 0) / values.length);
    setPhase('done');
    onComplete(avg / 100, avg);
  };

  const beginRound = (nextRound, baseScores = scores) => {
    if (nextRound > totalRounds) {
      finish(baseScores.length ? baseScores : [0]);
      return;
    }
    const data = buildProduceRound();
    bestRef.current = data.bestIndex;
    setRound(nextRound);
    setRule(data.rule);
    setItems(data.items);
    setPhase('playing');
  };

  const start = () => {
    setScores([]);
    beginRound(1, []);
  };

  const choose = (index) => {
    if (phase !== 'playing') return;
    const roundScore = index === bestRef.current ? 100 : 35;
    const nextScores = [...scores, roundScore];
    setScores(nextScores);
    setTimeout(() => {
      if (round >= totalRounds) finish(nextScores);
      else beginRound(round + 1, nextScores);
    }, 220);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🏆 ${t('根据评委标准，选出本轮最优农产品')}`} hint={`4 ${t('轮')} · ${t('看属性做决策')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const avg = Math.round((scores.reduce((sum, val) => sum + val, 0) / scores.length) || 0);
    return <ResultPanel emoji={avg >= 80 ? '🥇' : avg >= 50 ? '🏅' : '😅'} title={`${t('评比得分')}: ${avg}`} detail={scores.map((score, index) => `R${index + 1}:${score}`).join(' · ')} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${t('第')} ${round}/${totalRounds} ${t('轮')}`} right={rule ? `${rule.icon} ${t(rule.label)}` : ''} />
      {rule && (
        <div style={{ fontSize: 13, color: 'var(--farm-text-2)' }}>
          {rule.key === 'balance' ? t('总分最高且最均衡的产品胜出') : `${t('该项权重更高')}: ${t(rule.label)}`}
        </div>
      )}
      <div style={{ ...panelStyle, display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 10 }}>
        {items.map((item, index) => (
          <button
            key={item.id}
            type='button'
            onClick={() => choose(index)}
            style={{ minHeight: 108, borderRadius: 16, border: '2px solid var(--farm-border)', background: 'var(--farm-surface)', cursor: 'pointer' }}
          >
            <div style={{ fontSize: 28 }}>{item.emoji}</div>
            <div style={{ fontSize: 12, marginTop: 6 }}>🍯 {item.sweet} · 💧 {item.fresh} · ✨ {item.looks}</div>
          </button>
        ))}
      </div>
    </div>
  );
};
