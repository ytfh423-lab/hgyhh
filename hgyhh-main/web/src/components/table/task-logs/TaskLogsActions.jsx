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
import { Typography } from '@douyinfe/semi-ui';
import { ListChecks } from 'lucide-react';
import CompactModeToggle from '../../common/ui/CompactModeToggle';

const { Text } = Typography;

const TaskLogsActions = ({ compactMode, setCompactMode, t }) => {
  return (
    <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full'>
      <div className='flex items-center gap-2.5 mb-2 md:mb-0'>
        <div
          style={{
            width: '30px',
            height: '30px',
            borderRadius: '9px',
            background: 'linear-gradient(135deg, #f59e0b, #f97316)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            boxShadow: '0 2px 8px rgba(249, 115, 22, 0.25)',
            flexShrink: 0,
          }}
        >
          <ListChecks size={14} style={{ color: 'white' }} />
        </div>
        <Text style={{ fontWeight: 600, fontSize: '15px' }}>{t('任务记录')}</Text>
      </div>
      <CompactModeToggle
        compactMode={compactMode}
        setCompactMode={setCompactMode}
        t={t}
      />
    </div>
  );
};

export default TaskLogsActions;
