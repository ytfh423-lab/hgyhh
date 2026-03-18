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
import { Skeleton, Typography } from '@douyinfe/semi-ui';
import { useMinimumLoadingTime } from '../../../hooks/common/useMinimumLoadingTime';
import { Palette } from 'lucide-react';
import CompactModeToggle from '../../common/ui/CompactModeToggle';

const { Text } = Typography;

const MjLogsActions = ({
  loading,
  showBanner,
  isAdminUser,
  compactMode,
  setCompactMode,
  t,
}) => {
  const showSkeleton = useMinimumLoadingTime(loading);

  const placeholder = (
    <div className='flex items-center mb-2 md:mb-0'>
      <IconEyeOpened className='mr-2' />
      <Skeleton.Title style={{ width: 300, height: 21, borderRadius: 6 }} />
    </div>
  );

  return (
    <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full'>
      <Skeleton loading={showSkeleton} active placeholder={placeholder}>
        <div className='flex items-center gap-2.5 mb-2 md:mb-0'>
          <div
            style={{
              width: '30px',
              height: '30px',
              borderRadius: '9px',
              background: 'linear-gradient(135deg, #ec4899, #f43f5e)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              boxShadow: '0 2px 8px rgba(244, 63, 94, 0.25)',
              flexShrink: 0,
            }}
          >
            <Palette size={14} style={{ color: 'white' }} />
          </div>
          <Text style={{ fontWeight: 600, fontSize: '15px' }}>
            {isAdminUser && showBanner
              ? t(
                  '当前未开启Midjourney回调，部分项目可能无法获得绘图结果，可在运营设置中开启。',
                )
              : t('Midjourney 任务记录')}
          </Text>
        </div>
      </Skeleton>

      <CompactModeToggle
        compactMode={compactMode}
        setCompactMode={setCompactMode}
        t={t}
      />
    </div>
  );
};

export default MjLogsActions;
