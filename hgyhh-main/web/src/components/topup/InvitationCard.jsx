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
import {
  Avatar,
  Typography,
  Card,
  Button,
  Input,
  Badge,
  Space,
} from '@douyinfe/semi-ui';
import { Copy, Users, BarChart2, TrendingUp, Gift, Zap, Sparkles } from 'lucide-react';

const { Text } = Typography;

const InvitationCard = ({
  t,
  userState,
  renderQuota,
  setOpenTransfer,
  affLink,
  handleAffLinkClick,
}) => {
  return (
    <Card
      className='!rounded-2xl'
      style={{
        boxShadow: '0 1px 3px rgba(0,0,0,0.05), 0 10px 30px rgba(0,0,0,0.03)',
        border: '1px solid var(--semi-color-border)',
      }}
    >
      {/* 卡片头部 */}
      <div className='flex items-center mb-5'>
        <div
          className='mr-3 flex items-center justify-center'
          style={{
            width: '40px',
            height: '40px',
            borderRadius: '12px',
            background: 'linear-gradient(135deg, #10b981, #059669)',
            boxShadow: '0 4px 12px rgba(16, 185, 129, 0.25)',
          }}
        >
          <Gift size={18} style={{ color: 'white' }} />
        </div>
        <div>
          <Typography.Text style={{ fontSize: '17px', fontWeight: 600 }}>
            {t('邀请奖励')}
          </Typography.Text>
          <div style={{ fontSize: '12px', color: 'var(--semi-color-text-2)', marginTop: '2px' }}>
            {t('邀请好友获得额外奖励')}
          </div>
        </div>
      </div>

      {/* 收益展示区域 */}
      <Space vertical style={{ width: '100%' }}>
        {/* 统计数据统一卡片 */}
        <Card
          className='!rounded-2xl w-full !overflow-hidden'
          style={{ border: 'none' }}
          cover={
            <div
              className='relative'
              style={{
                background: 'linear-gradient(135deg, #065f46 0%, #059669 50%, #0d9488 100%)',
                padding: '28px 24px',
              }}
            >
              <div
                style={{
                  position: 'absolute',
                  top: '-30px',
                  right: '-30px',
                  width: '140px',
                  height: '140px',
                  borderRadius: '50%',
                  background: 'rgba(255,255,255,0.06)',
                }}
              />
              <div
                style={{
                  position: 'absolute',
                  bottom: '-20px',
                  left: '30%',
                  width: '80px',
                  height: '80px',
                  borderRadius: '50%',
                  background: 'rgba(255,255,255,0.04)',
                }}
              />
              <div className='relative z-10 flex flex-col'>
                <div className='flex justify-between items-center mb-5'>
                  <Text strong style={{ color: 'white', fontSize: '17px', letterSpacing: '0.5px' }}>
                    {t('收益统计')}
                  </Text>
                  <Button
                    type='primary'
                    theme='solid'
                    size='small'
                    disabled={
                      !userState?.user?.aff_quota ||
                      userState?.user?.aff_quota <= 0
                    }
                    onClick={() => setOpenTransfer(true)}
                    style={{ borderRadius: '10px', fontWeight: 500 }}
                  >
                    <Zap size={12} className='mr-1' />
                    {t('划转到余额')}
                  </Button>
                </div>

                <div className='grid grid-cols-3 gap-4'>
                  {[
                    { value: renderQuota(userState?.user?.aff_quota || 0), label: t('待使用收益'), icon: TrendingUp },
                    { value: renderQuota(userState?.user?.aff_history_quota || 0), label: t('总收益'), icon: BarChart2 },
                    { value: userState?.user?.aff_count || 0, label: t('邀请人数'), icon: Users },
                  ].map((stat, i) => (
                    <div
                      key={i}
                      className='text-center'
                      style={{
                        background: 'rgba(255,255,255,0.1)',
                        backdropFilter: 'blur(10px)',
                        borderRadius: '14px',
                        padding: '16px 8px',
                        border: '1px solid rgba(255,255,255,0.15)',
                        transition: 'transform 0.2s ease, background 0.2s ease',
                      }}
                      onMouseEnter={(e) => {
                        e.currentTarget.style.transform = 'translateY(-2px)';
                        e.currentTarget.style.background = 'rgba(255,255,255,0.15)';
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.transform = 'translateY(0)';
                        e.currentTarget.style.background = 'rgba(255,255,255,0.1)';
                      }}
                    >
                      <div
                        className='text-base sm:text-xl font-bold mb-1.5'
                        style={{ color: 'white' }}
                      >
                        {stat.value}
                      </div>
                      <div className='flex items-center justify-center gap-1'>
                        <stat.icon size={13} style={{ color: 'rgba(255,255,255,0.7)' }} />
                        <Text style={{ color: 'rgba(255,255,255,0.7)', fontSize: '11px' }}>
                          {stat.label}
                        </Text>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          }
        >
          {/* 邀请链接部分 */}
          <Input
            value={affLink}
            readonly
            style={{ borderRadius: '12px' }}
            prefix={
              <Text style={{ fontWeight: 500, whiteSpace: 'nowrap' }}>
                {t('邀请链接')}
              </Text>
            }
            suffix={
              <Button
                type='primary'
                theme='solid'
                onClick={handleAffLinkClick}
                icon={<Copy size={14} />}
                style={{ borderRadius: '10px', fontWeight: 500 }}
              >
                {t('复制')}
              </Button>
            }
          />
        </Card>

        {/* 奖励说明 */}
        <Card
          className='!rounded-2xl w-full'
          style={{ border: '1px solid var(--semi-color-border)' }}
          title={
            <div className='flex items-center gap-2'>
              <Sparkles size={16} style={{ color: 'var(--semi-color-warning)' }} />
              <Text type='tertiary' style={{ fontWeight: 600 }}>{t('奖励说明')}</Text>
            </div>
          }
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
            {[
              t('邀请好友注册，好友充值后您可获得相应奖励'),
              t('通过划转功能将奖励额度转入到您的账户余额中'),
              t('邀请的好友越多，获得的奖励越多'),
            ].map((text, i) => (
              <div key={i} className='flex items-start gap-3'>
                <div style={{
                  width: '6px',
                  height: '6px',
                  borderRadius: '50%',
                  background: 'linear-gradient(135deg, #10b981, #059669)',
                  marginTop: '7px',
                  flexShrink: 0,
                }} />
                <Text type='tertiary' style={{ fontSize: '13px', lineHeight: '1.6' }}>
                  {text}
                </Text>
              </div>
            ))}
          </div>
        </Card>
      </Space>
    </Card>
  );
};

export default InvitationCard;
