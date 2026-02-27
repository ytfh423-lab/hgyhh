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
import { Empty, Button } from '@douyinfe/semi-ui';
import {
  IllustrationNotFound,
  IllustrationNotFoundDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

const NotFound = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  return (
    <div className='flex justify-center items-center h-screen p-8'>
      <Empty
        image={<IllustrationNotFound style={{ width: 250, height: 250 }} />}
        darkModeImage={
          <IllustrationNotFoundDark style={{ width: 250, height: 250 }} />
        }
        description={
          <span style={{ color: 'var(--semi-color-text-2)', fontSize: '15px' }}>
            {t('页面未找到，请检查您的浏览器地址是否正确')}
          </span>
        }
      >
        <Button
          theme='solid'
          type='primary'
          onClick={() => navigate('/')}
          style={{
            borderRadius: '12px',
            padding: '8px 24px',
            marginTop: '8px',
            fontWeight: 500,
          }}
        >
          {t('返回首页')}
        </Button>
      </Empty>
    </div>
  );
};

export default NotFound;
