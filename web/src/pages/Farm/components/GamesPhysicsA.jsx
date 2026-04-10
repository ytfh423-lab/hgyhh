import React, { useEffect, useRef, useState } from 'react';
import { clamp, ChoiceButton, ReadyPanel, ResultPanel, StatRow, randInt } from './GamesCommon';

const panelStyle = {
  width: '100%',
  maxWidth: 340,
  margin: '0 auto',
  padding: 12,
  borderRadius: 16,
  border: '2px solid var(--farm-border)',
  background: 'var(--farm-surface-alt)',
};

export const RoosterGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [lane, setLane] = useState(1);
  const [warningLane, setWarningLane] = useState(null);
  const [stats, setStats] = useState({ dodged: 0, total: 10, current: 0 });
  const laneRef = useRef(1);
  const stateRef = useRef(null);
  const warningRef = useRef(null);
  const nextRef = useRef(null);

  const clearAll = () => {
    if (warningRef.current) clearTimeout(warningRef.current);
    if (nextRef.current) clearTimeout(nextRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = () => {
    clearAll();
    const s = stateRef.current;
    const score = Math.round((s.dodged / s.total) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const launchAttack = () => {
    const s = stateRef.current;
    if (s.current >= s.total) return finish();
    const nextLane = randInt(0, 2);
    setWarningLane(nextLane);
    warningRef.current = setTimeout(() => {
      if (laneRef.current !== nextLane) s.dodged += 1;
      s.current += 1;
      setWarningLane(null);
      setStats({ dodged: s.dodged, total: s.total, current: s.current });
      nextRef.current = setTimeout(() => {
        if (s.current >= s.total) finish();
        else launchAttack();
      }, 240);
    }, 520);
  };

  const start = () => {
    stateRef.current = { dodged: 0, total: 10, current: 0 };
    laneRef.current = 1;
    setLane(1);
    setWarningLane(null);
    setStats({ dodged: 0, total: 10, current: 0 });
    setPhase('playing');
    launchAttack();
  };

  const pickLane = (nextLane) => {
    if (phase !== 'playing') return;
    laneRef.current = nextLane;
    setLane(nextLane);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🐓 ${t('看准对手冲撞路线，及时闪到别的斗圈')}`} hint={`10 ${t('次冲撞')} · ${t('提前换道躲避')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    const score = Math.round((stats.dodged / stats.total) * 100);
    return <ResultPanel emoji={score >= 80 ? '🏟️' : score >= 50 ? '🐓' : '💫'} title={`${t('闪避成功')} ${stats.dodged}/${stats.total}`} detail={`${t('得分')} ${score}`} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`🛡️ ${stats.dodged}`} right={`${t('进度')} ${stats.current}/${stats.total}`} />
      <div style={{ ...panelStyle, display: 'flex', flexDirection: 'column', gap: 8 }}>
        {[0, 1, 2].map((row) => (
          <div key={row} style={{ position: 'relative', height: 56, borderRadius: 14, background: row === lane ? 'rgba(200,146,42,0.12)' : 'var(--farm-surface)' }}>
            <div style={{ position: 'absolute', left: 12, top: 10, fontSize: 28 }}>🐓</div>
            <div style={{ position: 'absolute', right: 12, top: 10, fontSize: 28 }}>{warningLane === row ? '💥' : '🐓'}</div>
            <div style={{ position: 'absolute', inset: '0 62px 0 62px', display: 'flex', alignItems: 'center' }}>
              <div style={{ width: '100%', borderTop: `2px dashed ${warningLane === row ? 'var(--farm-danger)' : 'var(--farm-border)'}` }} />
            </div>
          </div>
        ))}
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        {[0, 1, 2].map((row) => (
          <ChoiceButton key={row} active={lane === row} onClick={() => pickLane(row)}>{t('斗圈')} {row + 1}</ChoiceButton>
        ))}
      </div>
    </div>
  );
};

export const SunflowerGame = ({ game, onComplete, t }) => {
  const [phase, setPhase] = useState('ready');
  const [timeLeft, setTimeLeft] = useState(9);
  const [sun, setSun] = useState(50);
  const [flower, setFlower] = useState(50);
  const [lightRate, setLightRate] = useState(0);
  const stateRef = useRef(null);
  const loopRef = useRef(null);
  const timerRef = useRef(null);

  const clearAll = () => {
    if (loopRef.current) clearInterval(loopRef.current);
    if (timerRef.current) clearInterval(timerRef.current);
  };

  useEffect(() => clearAll, []);

  const finish = () => {
    clearAll();
    const s = stateRef.current;
    const score = Math.round((s.lightFrames / Math.max(1, s.totalFrames)) * 100);
    setPhase('done');
    onComplete(score / 100, score);
  };

  const start = () => {
    stateRef.current = { sun: 50, sunV: 1.6, flower: 50, lightFrames: 0, totalFrames: 0 };
    setSun(50);
    setFlower(50);
    setLightRate(0);
    setTimeLeft(9);
    setPhase('playing');
    loopRef.current = setInterval(() => {
      const s = stateRef.current;
      s.sun += s.sunV;
      if (s.sun <= 8 || s.sun >= 92) {
        s.sunV = -s.sunV;
        s.sun = clamp(s.sun, 8, 92);
      }
      s.totalFrames += 1;
      if (Math.abs(s.sun - s.flower) <= 10) s.lightFrames += 1;
      setSun(s.sun);
      setFlower(s.flower);
      setLightRate(Math.round((s.lightFrames / s.totalFrames) * 100));
    }, 90);
    timerRef.current = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          finish();
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  const turnFlower = (dir) => {
    if (phase !== 'playing') return;
    const s = stateRef.current;
    s.flower = clamp(s.flower + dir * 7, 0, 100);
    setFlower(s.flower);
  };

  if (phase === 'ready') {
    return <ReadyPanel game={game} desc={`🌻 ${t('让向日葵始终朝向太阳，积累更多日照')}`} hint={`9s · ${t('太阳会来回移动')} · ${t('左右转向')}`} onStart={start} t={t} />;
  }

  if (phase === 'done') {
    return <ResultPanel emoji={lightRate >= 80 ? '🌞' : lightRate >= 50 ? '🌻' : '🌥️'} title={`${t('采光率')}: ${lightRate}%`} detail={t('越长时间对准太阳，得分越高')} />;
  }

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
      <StatRow left={`${timeLeft}s`} right={`☀️ ${lightRate}%`} />
      <div style={{ ...panelStyle, position: 'relative', height: 150 }}>
        <div style={{ position: 'absolute', left: `${sun}%`, top: 14, transform: 'translateX(-50%)', fontSize: 28 }}>☀️</div>
        <div style={{ position: 'absolute', left: `${flower}%`, bottom: 18, transform: 'translateX(-50%)', textAlign: 'center' }}>
          <div style={{ fontSize: 34 }}>🌻</div>
          <div style={{ width: 2, height: 40, background: 'var(--farm-leaf)', margin: '0 auto' }} />
        </div>
        <div style={{ position: 'absolute', left: `${sun}%`, top: 44, width: 2, height: 56, background: 'rgba(242,201,76,0.4)', transform: 'translateX(-50%)' }} />
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        <ChoiceButton onClick={() => turnFlower(-1)}>⬅️</ChoiceButton>
        <ChoiceButton onClick={() => turnFlower(1)}>➡️</ChoiceButton>
      </div>
    </div>
  );
};
