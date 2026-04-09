/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';

const Loading = ({ size = 'small', fullscreen = true, text = '加载中' }) => {
  const className = [
    'app-loading',
    fullscreen ? 'app-loading--fullscreen' : 'app-loading--inline',
    size === 'large' ? 'app-loading--large' : '',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <div className={className} role='status' aria-live='polite' aria-busy='true'>
      <div className='app-loading-shell'>
        <div className='app-loading-orbit'>
          <span className='app-loading-orbit-dot' />
          <span className='app-loading-orbit-dot app-loading-orbit-dot--delay' />
          <span className='app-loading-ring' />
        </div>
        <div className='app-loading-text'>{text}</div>
        <div className='app-loading-dots'>
          <span />
          <span />
          <span />
        </div>
      </div>
    </div>
  );
};

export default Loading;
