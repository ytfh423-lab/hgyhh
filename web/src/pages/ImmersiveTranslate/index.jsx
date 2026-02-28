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

import React, { useContext, useState, useEffect } from 'react';
import { Typography, Button, Form, Toast, Card, Steps, Banner } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  Languages,
  Zap,
  Shield,
  BookOpen,
  Copy,
  CheckCircle,
  ArrowRight,
  Settings,
  Key,
  Globe,
  AlertTriangle,
} from 'lucide-react';
import { StatusContext } from '../../context/Status';
import { UserContext } from '../../context/User';
import { API, showError, showSuccess, copy } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';

const { Title, Text, Paragraph } = Typography;

const ImmersiveTranslate = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [statusState] = useContext(StatusContext);
  const [userState] = useContext(UserContext);
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [tokenKey, setTokenKey] = useState('');
  const [tokenCreated, setTokenCreated] = useState(false);
  const serverAddress = statusState?.status?.server_address || `${window.location.origin}`;

  const isLoggedIn = !!userState?.user;

  const handleCreateToken = async () => {
    if (!isLoggedIn) {
      navigate('/login');
      return;
    }

    setLoading(true);
    try {
      const res = await API.post('/api/token/', {
        name: t('沉浸式翻译专用'),
        remain_quota: 0,
        expired_time: -1,
        unlimited_quota: true,
        model_limits_enabled: true,
        model_limits: 'gpt-4o-mini,gpt-3.5-turbo,deepseek-chat,deepseek-v3,glm-4-flash',
      });

      const { success, message, data } = res.data;
      if (success) {
        setTokenKey(data.key);
        setTokenCreated(true);
        showSuccess(t('令牌创建成功'));
      } else {
        showError(message);
      }
    } catch (err) {
      showError(t('创建令牌失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  const handleCopyKey = () => {
    if (tokenKey) {
      copy(tokenKey, t('令牌'));
    }
  };

  const handleCopyAddress = () => {
    copy(`${serverAddress}/v1`, 'API');
  };

  const features = [
    {
      icon: <Zap size={28} style={{ color: '#6366f1' }} />,
      title: t('极速翻译'),
      desc: t('基于大模型的高质量翻译，支持上下文理解，翻译更准确自然'),
      bg: 'rgba(99, 102, 241, 0.08)',
    },
    {
      icon: <Shield size={28} style={{ color: '#10b981' }} />,
      title: t('免费使用'),
      desc: t('公益接口完全免费，无需付费即可享受 AI 翻译服务'),
      bg: 'rgba(16, 185, 129, 0.08)',
    },
    {
      icon: <Globe size={28} style={{ color: '#f59e0b' }} />,
      title: t('多模型支持'),
      desc: t('支持 GPT-4o-mini、DeepSeek、GLM 等多种模型，自由切换'),
      bg: 'rgba(245, 158, 11, 0.08)',
    },
  ];

  const steps = [
    {
      title: t('注册账号'),
      desc: t('注册并登录 NPC-API 账号'),
      icon: <Key size={20} />,
    },
    {
      title: t('获取令牌'),
      desc: t('点击下方按钮一键生成专属翻译令牌'),
      icon: <Settings size={20} />,
    },
    {
      title: t('配置插件'),
      desc: t('在沉浸式翻译插件中填入接口地址和令牌'),
      icon: <Languages size={20} />,
    },
  ];

  return (
    <div className='w-full overflow-x-hidden'>
      <div className='w-full max-w-5xl mx-auto px-4 pt-20 pb-16 md:pt-28 md:pb-24'>
        {/* Hero */}
        <div className='text-center mb-12 md:mb-16 npc-animate npc-delay-1'>
          <div
            className='inline-flex items-center gap-2 px-4 py-2 rounded-full mb-6'
            style={{
              background: 'linear-gradient(135deg, rgba(99,102,241,0.1), rgba(168,85,247,0.1))',
              border: '1px solid rgba(99,102,241,0.15)',
            }}
          >
            <Languages size={18} style={{ color: '#6366f1' }} />
            <Text style={{ color: '#6366f1', fontWeight: 600, fontSize: '14px' }}>
              {t('沉浸式翻译专属接口')}
            </Text>
          </div>

          <Title
            heading={2}
            style={{
              marginBottom: '16px',
              background: 'linear-gradient(135deg, #6366f1, #a855f7, #ec4899)',
              WebkitBackgroundClip: 'text',
              WebkitTextFillColor: 'transparent',
              fontSize: isMobile ? '28px' : '40px',
              fontWeight: 800,
              lineHeight: 1.2,
            }}
          >
            {t('为沉浸式翻译量身打造')}
          </Title>

          <Paragraph
            style={{
              color: 'var(--semi-color-text-2)',
              fontSize: isMobile ? '15px' : '17px',
              lineHeight: 1.7,
              maxWidth: '600px',
              margin: '0 auto',
            }}
          >
            {t('一键获取专属 API 令牌，即刻在沉浸式翻译中享受高质量 AI 翻译体验')}
          </Paragraph>
        </div>

        {/* Feature Cards */}
        <div className={`grid gap-5 mb-12 md:mb-16 npc-animate npc-delay-2 ${isMobile ? 'grid-cols-1' : 'grid-cols-3'}`}>
          {features.map((f, i) => (
            <div
              key={i}
              className='npc-feature-card'
              style={{ textAlign: 'center', padding: '32px 24px' }}
            >
              <div
                className='npc-feature-icon'
                style={{
                  background: f.bg,
                  margin: '0 auto 16px',
                }}
              >
                {f.icon}
              </div>
              <Title heading={5} style={{ marginBottom: '8px' }}>
                {f.title}
              </Title>
              <Text style={{ color: 'var(--semi-color-text-2)', fontSize: '14px', lineHeight: 1.6 }}>
                {f.desc}
              </Text>
            </div>
          ))}
        </div>

        {/* Steps */}
        <div className='mb-12 md:mb-16 npc-animate npc-delay-3'>
          <Title heading={4} style={{ textAlign: 'center', marginBottom: '32px' }}>
            {t('三步配置，即刻使用')}
          </Title>

          <div className={`grid gap-6 ${isMobile ? 'grid-cols-1' : 'grid-cols-3'}`}>
            {steps.map((step, i) => (
              <div
                key={i}
                className='flex flex-col items-center text-center p-6 rounded-2xl'
                style={{
                  background: 'var(--semi-color-bg-1)',
                  border: '1px solid var(--semi-color-border)',
                }}
              >
                <div
                  className='flex items-center justify-center w-12 h-12 rounded-full mb-4'
                  style={{
                    background: 'linear-gradient(135deg, #6366f1, #a855f7)',
                    color: '#fff',
                    fontSize: '18px',
                    fontWeight: 700,
                  }}
                >
                  {i + 1}
                </div>
                <Title heading={5} style={{ marginBottom: '8px' }}>
                  {step.title}
                </Title>
                <Text style={{ color: 'var(--semi-color-text-2)', fontSize: '14px' }}>
                  {step.desc}
                </Text>
              </div>
            ))}
          </div>
        </div>

        {/* Token Generation */}
        <div className='npc-animate npc-delay-4'>
          <div
            className='rounded-2xl p-6 md:p-10'
            style={{
              background: 'var(--semi-color-bg-1)',
              border: '1px solid var(--semi-color-border)',
              boxShadow: '0 4px 24px rgba(0,0,0,0.04)',
            }}
          >
            <Title heading={4} style={{ textAlign: 'center', marginBottom: '8px' }}>
              {t('获取专属令牌')}
            </Title>
            <Paragraph
              style={{
                textAlign: 'center',
                color: 'var(--semi-color-text-2)',
                marginBottom: '24px',
                fontSize: '14px',
              }}
            >
              {t('点击下方按钮一键生成沉浸式翻译专属令牌，已限制仅可使用翻译相关模型')}
            </Paragraph>

            {!isLoggedIn && (
              <Banner
                type='warning'
                description={t('请先登录后再申请令牌')}
                style={{ marginBottom: '24px', borderRadius: '12px' }}
                icon={<AlertTriangle size={16} />}
              />
            )}

            {tokenCreated ? (
              <div className='space-y-4'>
                <Banner
                  type='success'
                  description={t('令牌创建成功！请妥善保管，关闭页面后将无法再次查看。')}
                  style={{ borderRadius: '12px', marginBottom: '16px' }}
                  icon={<CheckCircle size={16} />}
                />

                {/* API Address */}
                <div>
                  <Text strong style={{ display: 'block', marginBottom: '8px' }}>
                    {t('API 接口地址')}
                  </Text>
                  <div
                    className='flex items-center gap-2 p-3 rounded-xl'
                    style={{
                      background: 'var(--semi-color-fill-0)',
                      border: '1px solid var(--semi-color-border)',
                      fontFamily: 'monospace',
                      fontSize: '14px',
                      wordBreak: 'break-all',
                    }}
                  >
                    <Text style={{ flex: 1 }}>{serverAddress}/v1</Text>
                    <Button
                      icon={<Copy size={14} />}
                      size='small'
                      theme='borderless'
                      onClick={handleCopyAddress}
                    />
                  </div>
                </div>

                {/* Token Key */}
                <div>
                  <Text strong style={{ display: 'block', marginBottom: '8px' }}>
                    {t('API 密钥（令牌）')}
                  </Text>
                  <div
                    className='flex items-center gap-2 p-3 rounded-xl'
                    style={{
                      background: 'var(--semi-color-fill-0)',
                      border: '1px solid var(--semi-color-border)',
                      fontFamily: 'monospace',
                      fontSize: '14px',
                      wordBreak: 'break-all',
                    }}
                  >
                    <Text style={{ flex: 1 }}>{tokenKey}</Text>
                    <Button
                      icon={<Copy size={14} />}
                      size='small'
                      theme='borderless'
                      onClick={handleCopyKey}
                    />
                  </div>
                </div>

                {/* Usage Instructions */}
                <div
                  className='p-4 rounded-xl mt-4'
                  style={{
                    background: 'linear-gradient(135deg, rgba(99,102,241,0.05), rgba(168,85,247,0.05))',
                    border: '1px solid rgba(99,102,241,0.1)',
                  }}
                >
                  <Title heading={6} style={{ marginBottom: '12px' }}>
                    <Settings size={16} style={{ display: 'inline', marginRight: '6px', verticalAlign: 'text-bottom' }} />
                    {t('配置说明')}
                  </Title>
                  <ol style={{ paddingLeft: '20px', color: 'var(--semi-color-text-2)', fontSize: '14px', lineHeight: 2 }}>
                    <li>{t('打开沉浸式翻译插件设置')}</li>
                    <li>{t('在「翻译服务」中选择 OpenAI 或自定义 API')}</li>
                    <li>{t('填入上方的 API 接口地址')}</li>
                    <li>{t('填入上方的 API 密钥')}</li>
                    <li>{t('模型选择 gpt-4o-mini 或 deepseek-chat')}</li>
                    <li>{t('保存设置，即可使用')}</li>
                  </ol>
                </div>
              </div>
            ) : (
              <div className='text-center'>
                <Button
                  theme='solid'
                  size='large'
                  loading={loading}
                  onClick={handleCreateToken}
                  style={{
                    background: 'linear-gradient(135deg, #6366f1, #a855f7)',
                    border: 'none',
                    borderRadius: '14px',
                    padding: '12px 40px',
                    fontSize: '16px',
                    fontWeight: 600,
                    height: 'auto',
                  }}
                  icon={isLoggedIn ? <Key size={18} /> : <ArrowRight size={18} />}
                >
                  {isLoggedIn ? t('一键生成专属令牌') : t('登录后申请')}
                </Button>
              </div>
            )}
          </div>
        </div>

        {/* Warning */}
        <div className='mt-8 npc-animate npc-delay-5'>
          <Banner
            type='info'
            description={t('本接口为公益服务，请合理使用。严禁批量爬取、高频滥用等行为，违规将被封禁。')}
            style={{ borderRadius: '12px' }}
            icon={<AlertTriangle size={16} />}
          />
        </div>
      </div>
    </div>
  );
};

export default ImmersiveTranslate;
