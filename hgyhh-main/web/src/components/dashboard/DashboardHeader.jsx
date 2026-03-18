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
import { Button } from '@douyinfe/semi-ui';
import { RefreshCw, Search } from 'lucide-react';

const DashboardHeader = ({
  getGreeting,
  greetingVisible,
  showSearchModal,
  refresh,
  loading,
  t,
}) => {
  return (
    <div className='flex items-center justify-between mb-5'>
      <h2
        className='text-2xl font-semibold transition-opacity duration-1000 ease-in-out'
        style={{
          opacity: greetingVisible ? 1 : 0,
          color: 'var(--semi-color-text-0)',
        }}
      >
        {getGreeting}
      </h2>
      <div className='flex gap-2'>
        <Button
          type='tertiary'
          icon={<Search size={15} style={{ color: 'white' }} />}
          onClick={showSearchModal}
          style={{
            background: 'linear-gradient(135deg, #10b981, #059669)',
            border: 'none',
            borderRadius: '12px',
            width: '36px',
            height: '36px',
            boxShadow: '0 2px 8px rgba(16, 185, 129, 0.3)',
          }}
        />
        <Button
          type='tertiary'
          icon={<RefreshCw size={15} style={{ color: 'white' }} />}
          onClick={refresh}
          loading={loading}
          style={{
            background: 'linear-gradient(135deg, #3b82f6, #6366f1)',
            border: 'none',
            borderRadius: '12px',
            width: '36px',
            height: '36px',
            boxShadow: '0 2px 8px rgba(99, 102, 241, 0.3)',
          }}
        />
      </div>
    </div>
  );
};

export default DashboardHeader;
