import React, { useContext, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Card, Form } from '@douyinfe/semi-ui';
import Title from '@douyinfe/semi-ui/lib/es/typography/title';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import { IconKey } from '@douyinfe/semi-icons';
import {
  API,
  showError,
  showSuccess,
  updateAPI,
  setUserData,
  getLogo,
  getSystemName,
} from '../../helpers';
import { UserContext } from '../../context/User';

const OAuthRegistrationForm = () => {
  const { provider } = useParams();
  const { t } = useTranslation();
  const [, userDispatch] = useContext(UserContext);
  const navigate = useNavigate();
  const [registrationCode, setRegistrationCode] = useState('');
  const [loading, setLoading] = useState(false);

  const logo = getLogo();
  const systemName = getSystemName();

  const handleSubmit = async () => {
    if (!registrationCode) {
      showError(t('请输入管理员发放的注册码'));
      return;
    }
    setLoading(true);
    try {
      const { data: resData } = await API.post(`/api/oauth/${provider}/register`, {
        registration_code: registrationCode,
      });
      const { success, message, data } = resData;
      if (!success) {
        showError(message || t('注册失败，请重试'));
        return;
      }
      userDispatch({ type: 'login', payload: data });
      localStorage.setItem('user', JSON.stringify(data));
      setUserData(data);
      updateAPI();
      showSuccess(t('注册并登录成功！'));
      navigate('/console/token');
    } catch (error) {
      showError(error.message || t('注册失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className='flex flex-col items-center mt-16 px-4'>
      <div className='w-full max-w-md'>
        <div className='flex items-center justify-center mb-6 gap-2'>
          <img src={logo} alt='Logo' className='h-10 rounded-full' />
          <Title heading={3} className='!text-gray-800'>
            {systemName}
          </Title>
        </div>
        <Card className='border-0 !rounded-2xl overflow-hidden'>
          <div className='flex justify-center pt-6 pb-2'>
            <Title heading={3} className='text-gray-800 dark:text-gray-200'>
              {t('输入注册码完成注册')}
            </Title>
          </div>
          <div className='px-2 py-8'>
            <Form>
              <Form.Input
                field='registration_code'
                label={t('注册码')}
                placeholder={t('请输入管理员发放的注册码')}
                value={registrationCode}
                onChange={(value) => setRegistrationCode(value)}
                prefix={<IconKey />}
              />
              <Text type='secondary' size='small'>
                {t('LinuxDO 授权成功后，首次注册需要管理员注册码。')}
              </Text>
              <div className='pt-4'>
                <Button
                  theme='solid'
                  type='primary'
                  className='w-full !rounded-full'
                  loading={loading}
                  onClick={handleSubmit}
                >
                  {t('提交并完成注册')}
                </Button>
              </div>
            </Form>
          </div>
        </Card>
      </div>
    </div>
  );
};

export default OAuthRegistrationForm;
