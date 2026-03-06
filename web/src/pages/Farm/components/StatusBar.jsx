import React from 'react';
import { formatBalance } from './utils';

const StatusBar = ({ farmData, t }) => {
  if (!farmData) return null;

  return (
    <div className='farm-statusbar'>
      <div className='farm-pill farm-pill-green'>
        <span>💰</span>
        <span>{formatBalance(farmData.balance)}</span>
      </div>
      <div className='farm-pill farm-pill-blue'>
        <span>⭐</span>
        <span>Lv.{farmData.user_level || 1}</span>
      </div>
      <div className='farm-pill'>
        <span>🌾</span>
        <span>{farmData.plot_count}/{farmData.max_plots} {t('块地')}</span>
      </div>
      {farmData.weather && (
        <div className='farm-pill farm-pill-cyan'>
          <span>{farmData.weather.emoji}</span>
          <span>{farmData.weather.name}</span>
          {farmData.weather.season_name && (
            <span style={{ opacity: 0.7, fontSize: 11 }}>· {farmData.weather.season_name}</span>
          )}
        </div>
      )}
      {farmData.prestige_level > 0 && (
        <div className='farm-pill farm-pill-purple'>
          <span>🔄</span>
          <span>P{farmData.prestige_level} (+{farmData.prestige_bonus}%)</span>
        </div>
      )}
      {farmData.dog && (
        <div className={`farm-pill ${farmData.dog.hunger > 0 ? 'farm-pill-green' : 'farm-pill-red'}`}>
          <span>{farmData.dog.level === 2 ? '🐕' : '🐶'}</span>
          <span>{farmData.dog.name || ''} {farmData.dog.hunger}%</span>
        </div>
      )}
      {farmData.items && farmData.items.length > 0 && (
        <div className='farm-statusbar-items'>
          {farmData.items.map((item) => (
            <div key={item.key} className='farm-pill farm-pill-amber'>
              <span>{item.emoji}</span>
              <span>{item.name}×{item.quantity}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default StatusBar;
