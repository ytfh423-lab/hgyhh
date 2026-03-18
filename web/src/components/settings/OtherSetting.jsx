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

import React, { useContext, useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  DatePicker,
  Form,
  Row,
  Modal,
  Space,
  Card,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess, timestamp2string } from '../../helpers';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../context/Status';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';

const LEGAL_USER_AGREEMENT_KEY = 'legal.user_agreement';
const LEGAL_PRIVACY_POLICY_KEY = 'legal.privacy_policy';

const OtherSetting = () => {
  const { t } = useTranslation();
  let [inputs, setInputs] = useState({
    Notice: '',
    [LEGAL_USER_AGREEMENT_KEY]: '',
    [LEGAL_PRIVACY_POLICY_KEY]: '',
    SystemName: '',
    Logo: '',
    Footer: '',
    About: '',
    HomePageContent: '',
    HomeAdHtml: '',
    FarmCountdownDate: '',
    FarmBetaEnabled: 'false',
    FarmBetaMaxSlots: '100',
    FarmBetaAdminBypass: 'true',
    FarmBetaEndTime: '',
    FarmAdminUserId: '0',
    TgBotFarmWarehouseMaxLevel: '10',
    TgBotFarmWarehouseUpgradePrice: '2000000',
    TgBotFarmWarehouseCapacityPerLevel: '50',
    TgBotFarmWarehouseExpiryBonusPerLevel: '20',
    CheckinLeaderboardLimit: '',
  });
  let [loading, setLoading] = useState(false);
  const [showUpdateModal, setShowUpdateModal] = useState(false);
  const [statusState, statusDispatch] = useContext(StatusContext);
  const [updateData, setUpdateData] = useState({
    tag_name: '',
    content: '',
  });

  const updateOption = async (key, value) => {
    setLoading(true);
    const res = await API.put('/api/option/', {
      key,
      value,
    });
    const { success, message } = res.data;
    if (success) {
      setInputs((inputs) => ({ ...inputs, [key]: value }));
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const [loadingInput, setLoadingInput] = useState({
    Notice: false,
    [LEGAL_USER_AGREEMENT_KEY]: false,
    [LEGAL_PRIVACY_POLICY_KEY]: false,
    SystemName: false,
    Logo: false,
    HomePageContent: false,
    HomeAdHtml: false,
    FarmCountdownDate: false,
    FarmBetaEnabled: false,
    FarmBetaMaxSlots: false,
    FarmBetaAdminBypass: false,
    FarmBetaEndTime: false,
    FarmAdminUserId: false,
    TgBotFarmWarehouseUpgrade: false,
    CheckinLeaderboardLimit: false,
    About: false,
    Footer: false,
    CheckUpdate: false,
  });
  const handleInputChange = async (value, e) => {
    const name = e.target.id;
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  // 通用设置
  const formAPISettingGeneral = useRef();
  // 通用设置 - Notice
  const submitNotice = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Notice: true }));
      await updateOption('Notice', inputs.Notice);
      showSuccess(t('公告已更新'));
    } catch (error) {
      console.error(t('公告更新失败'), error);
      showError(t('公告更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Notice: false }));
    }
  };
  // 通用设置 - UserAgreement
  const submitUserAgreement = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_USER_AGREEMENT_KEY]: true,
      }));
      await updateOption(
        LEGAL_USER_AGREEMENT_KEY,
        inputs[LEGAL_USER_AGREEMENT_KEY],
      );
      showSuccess(t('用户协议已更新'));
    } catch (error) {
      console.error(t('用户协议更新失败'), error);
      showError(t('用户协议更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_USER_AGREEMENT_KEY]: false,
      }));
    }
  };
  // 通用设置 - PrivacyPolicy
  const submitPrivacyPolicy = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_PRIVACY_POLICY_KEY]: true,
      }));
      await updateOption(
        LEGAL_PRIVACY_POLICY_KEY,
        inputs[LEGAL_PRIVACY_POLICY_KEY],
      );
      showSuccess(t('隐私政策已更新'));
    } catch (error) {
      console.error(t('隐私政策更新失败'), error);
      showError(t('隐私政策更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_PRIVACY_POLICY_KEY]: false,
      }));
    }
  };
  // 个性化设置
  const formAPIPersonalization = useRef();
  //  个性化设置 - SystemName
  const submitSystemName = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        SystemName: true,
      }));
      await updateOption('SystemName', inputs.SystemName);
      showSuccess(t('系统名称已更新'));
    } catch (error) {
      console.error(t('系统名称更新失败'), error);
      showError(t('系统名称更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        SystemName: false,
      }));
    }
  };

  // 个性化设置 - Logo
  const submitLogo = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Logo: true }));
      await updateOption('Logo', inputs.Logo);
      showSuccess('Logo 已更新');
    } catch (error) {
      console.error('Logo 更新失败', error);
      showError('Logo 更新失败');
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Logo: false }));
    }
  };
  // 个性化设置 - 首页内容
  const submitOption = async (key) => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [key]: true,
      }));
      await updateOption(key, inputs[key]);
      showSuccess(t('设置已更新'));
    } catch (error) {
      console.error(t('设置更新失败'), error);
      showError(t('设置更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [key]: false,
      }));
    }
  };
  // 个性化设置 - 关于
  const submitAbout = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, About: true }));
      await updateOption('About', inputs.About);
      showSuccess('关于内容已更新');
    } catch (error) {
      console.error('关于内容更新失败', error);
      showError('关于内容更新失败');
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, About: false }));
    }
  };
  // 个性化设置 - 页脚
  const submitFooter = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Footer: true }));
      await updateOption('Footer', inputs.Footer);
      showSuccess('页脚内容已更新');
    } catch (error) {
      console.error('页脚内容更新失败', error);
      showError('页脚内容更新失败');
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Footer: false }));
    }
  };

  const checkUpdate = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        CheckUpdate: true,
      }));
      // Use a CORS proxy to avoid direct cross-origin requests to GitHub API
      // Option 1: Use a public CORS proxy service
      // const proxyUrl = 'https://cors-anywhere.herokuapp.com/';
      // const res = await API.get(
      //   `${proxyUrl}https://api.github.com/repos/Calcium-Ion/new-api/releases/latest`,
      // );

      // Option 2: Use the JSON proxy approach which often works better with GitHub API
      const res = await fetch(
        'https://api.github.com/repos/Calcium-Ion/new-api/releases/latest',
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
            // Adding User-Agent which is often required by GitHub API
            'User-Agent': 'new-api-update-checker',
          },
        },
      ).then((response) => response.json());

      // Option 3: Use a local proxy endpoint
      // Create a cached version of the response to avoid frequent GitHub API calls
      // const res = await API.get('/api/status/github-latest-release');

      const { tag_name, body } = res;
      if (tag_name === statusState?.status?.version) {
        showSuccess(`已是最新版本：${tag_name}`);
      } else {
        setUpdateData({
          tag_name: tag_name,
          content: marked.parse(body),
        });
        setShowUpdateModal(true);
      }
    } catch (error) {
      console.error('Failed to check for updates:', error);
      showError('检查更新失败，请稍后再试');
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        CheckUpdate: false,
      }));
    }
  };
  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.key in inputs) {
          newInputs[item.key] = item.value;
        }
      });
      setInputs(newInputs);
      formAPISettingGeneral.current.setValues(newInputs);
      formAPIPersonalization.current.setValues(newInputs);
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    getOptions();
  }, []);

  // Function to open GitHub release page
  const openGitHubRelease = () => {
    window.open(
      `https://github.com/Calcium-Ion/new-api/releases/tag/${updateData.tag_name}`,
      '_blank',
    );
  };

  const getStartTimeString = () => {
    const timestamp = statusState?.status?.start_time;
    return statusState.status ? timestamp2string(timestamp) : '';
  };

  return (
    <Row>
      <Col
        span={24}
        style={{
          marginTop: '10px',
          display: 'flex',
          flexDirection: 'column',
          gap: '10px',
        }}
      >
        {/* 版本信息 */}
        <Form>
          <Card>
            <Form.Section text={t('系统信息')}>
              <Row>
                <Col span={16}>
                  <Space>
                    <Text>
                      {t('当前版本')}：
                      {statusState?.status?.version || t('未知')}
                    </Text>
                    <Button
                      type='primary'
                      onClick={checkUpdate}
                      loading={loadingInput['CheckUpdate']}
                    >
                      {t('检查更新')}
                    </Button>
                  </Space>
                </Col>
              </Row>
              <Row>
                <Col span={16}>
                  <Text>
                    {t('启动时间')}：{getStartTimeString()}
                  </Text>
                </Col>
              </Row>
            </Form.Section>
          </Card>
        </Form>
        {/* 通用设置 */}
        <Form
          values={inputs}
          getFormApi={(formAPI) => (formAPISettingGeneral.current = formAPI)}
        >
          <Card>
            <Form.Section text={t('通用设置')}>
              <Form.TextArea
                label={t('公告')}
                placeholder={t(
                  '在此输入新的公告内容，支持 Markdown & HTML 代码',
                )}
                field={'Notice'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button onClick={submitNotice} loading={loadingInput['Notice']}>
                {t('设置公告')}
              </Button>
              <Form.TextArea
                label={t('用户协议')}
                placeholder={t(
                  '在此输入用户协议内容，支持 Markdown & HTML 代码',
                )}
                field={LEGAL_USER_AGREEMENT_KEY}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
                helpText={t(
                  '填写用户协议内容后，用户注册时将被要求勾选已阅读用户协议',
                )}
              />
              <Button
                onClick={submitUserAgreement}
                loading={loadingInput[LEGAL_USER_AGREEMENT_KEY]}
              >
                {t('设置用户协议')}
              </Button>
              <Form.TextArea
                label={t('隐私政策')}
                placeholder={t(
                  '在此输入隐私政策内容，支持 Markdown & HTML 代码',
                )}
                field={LEGAL_PRIVACY_POLICY_KEY}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
                helpText={t(
                  '填写隐私政策内容后，用户注册时将被要求勾选已阅读隐私政策',
                )}
              />
              <Button
                onClick={submitPrivacyPolicy}
                loading={loadingInput[LEGAL_PRIVACY_POLICY_KEY]}
              >
                {t('设置隐私政策')}
              </Button>
            </Form.Section>
          </Card>
        </Form>
        {/* 个性化设置 */}
        <Form
          values={inputs}
          getFormApi={(formAPI) => (formAPIPersonalization.current = formAPI)}
        >
          <Card>
            <Form.Section text={t('个性化设置')}>
              <Form.Input
                label={t('系统名称')}
                placeholder={t('在此输入系统名称')}
                field={'SystemName'}
                onChange={handleInputChange}
              />
              <Button
                onClick={submitSystemName}
                loading={loadingInput['SystemName']}
              >
                {t('设置系统名称')}
              </Button>
              <Form.Input
                label={t('Logo 图片地址')}
                placeholder={t('在此输入 Logo 图片地址')}
                field={'Logo'}
                onChange={handleInputChange}
              />
              <Button onClick={submitLogo} loading={loadingInput['Logo']}>
                {t('设置 Logo')}
              </Button>
              <Form.TextArea
                label={t('首页内容')}
                placeholder={t(
                  '在此输入首页内容，支持 Markdown & HTML 代码，设置后首页的状态信息将不再显示。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性，这允许你设置任意网页作为首页',
                )}
                field={'HomePageContent'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button
                onClick={() => submitOption('HomePageContent')}
                loading={loadingInput['HomePageContent']}
              >
                {t('设置首页内容')}
              </Button>
              <Form.TextArea
                label={t('首页广告 HTML')}
                placeholder={t(
                  '在此输入首页广告区域的 HTML 代码，将显示在首页按钮下方，留空则不显示',
                )}
                field={'HomeAdHtml'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button
                onClick={() => submitOption('HomeAdHtml')}
                loading={loadingInput['HomeAdHtml']}
              >
                {t('设置首页广告')}
              </Button>
              <Form.Slot label={t('农场内测倒计时目标日期')}>
                <DatePicker
                  type='dateTime'
                  value={inputs.FarmCountdownDate ? new Date(inputs.FarmCountdownDate) : null}
                  onChange={(date) => {
                    const iso = date ? date.toISOString() : '';
                    setInputs((prev) => ({ ...prev, FarmCountdownDate: iso }));
                  }}
                  placeholder={t('选择倒计时目标日期，留空则默认 30 天后')}
                  style={{ width: '100%' }}
                />
                <div style={{ marginTop: 8, display: 'flex', gap: 8 }}>
                  <Button
                    onClick={() => submitOption('FarmCountdownDate')}
                    loading={loadingInput['FarmCountdownDate']}
                  >
                    {t('设置倒计时日期')}
                  </Button>
                  <Button
                    type='warning'
                    onClick={async () => {
                      setInputs((prev) => ({ ...prev, FarmCountdownDate: '' }));
                      await updateOption('FarmCountdownDate', '');
                      showSuccess(t('设置已更新'));
                    }}
                  >
                    {t('清除')}
                  </Button>
                </div>
              </Form.Slot>
              <Form.Slot label={t('农场内测系统')}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
                    <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
                      <input
                        type='checkbox'
                        checked={inputs.FarmBetaEnabled === 'true'}
                        onChange={(e) => setInputs((prev) => ({ ...prev, FarmBetaEnabled: e.target.checked ? 'true' : 'false' }))}
                      />
                      {t('启用内测模式')}
                    </label>
                    <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
                      <input
                        type='checkbox'
                        checked={inputs.FarmBetaAdminBypass === 'true'}
                        onChange={(e) => setInputs((prev) => ({ ...prev, FarmBetaAdminBypass: e.target.checked ? 'true' : 'false' }))}
                      />
                      {t('管理员绕过内测限制')}
                    </label>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span>{t('最大预约名额')}:</span>
                    <input
                      type='number'
                      value={inputs.FarmBetaMaxSlots}
                      onChange={(e) => setInputs((prev) => ({ ...prev, FarmBetaMaxSlots: e.target.value }))}
                      style={{ width: 100, padding: '4px 8px', borderRadius: 4, border: '1px solid var(--semi-color-border)', background: 'var(--semi-color-bg-1)', color: 'var(--semi-color-text-0)' }}
                      min={0}
                    />
                  </div>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Button
                      onClick={async () => {
                        setLoadingInput((prev) => ({ ...prev, FarmBetaEnabled: true }));
                        try {
                          await updateOption('FarmBetaEnabled', inputs.FarmBetaEnabled);
                          await updateOption('FarmBetaMaxSlots', inputs.FarmBetaMaxSlots);
                          await updateOption('FarmBetaAdminBypass', inputs.FarmBetaAdminBypass);
                          showSuccess(t('设置已更新'));
                        } catch (e) {
                          showError(t('设置更新失败'));
                        } finally {
                          setLoadingInput((prev) => ({ ...prev, FarmBetaEnabled: false }));
                        }
                      }}
                      loading={loadingInput['FarmBetaEnabled']}
                    >
                      {t('保存内测设置')}
                    </Button>
                  </div>
                  <Text type='tertiary' size='small'>
                    {t('开启内测模式后，农场将在倒计时结束前关闭。到达目标时间后，仅有预约且排名在名额内的用户可以访问农场。管理员绕过选项可让管理员在内测期间自由测试。')}
                  </Text>
                </div>
              </Form.Slot>
              <Form.Slot label={t('内测结束时间')}>
                <DatePicker
                  type='dateTime'
                  value={inputs.FarmBetaEndTime ? new Date(inputs.FarmBetaEndTime) : null}
                  onChange={(date) => {
                    const iso = date ? date.toISOString() : '';
                    setInputs((prev) => ({ ...prev, FarmBetaEndTime: iso }));
                  }}
                  placeholder={t('选择内测结束时间，到期后自动关闭农场并清空数据')}
                  style={{ width: '100%' }}
                />
                {inputs.FarmBetaEndTime && (
                  <div style={{ marginTop: 4, fontSize: 12, color: 'var(--semi-color-warning)' }}>
                    {(() => {
                      const end = new Date(inputs.FarmBetaEndTime);
                      const now = new Date();
                      if (end <= now) return t('内测已到期');
                      const diff = end - now;
                      const days = Math.floor(diff / 86400000);
                      const hours = Math.floor((diff % 86400000) / 3600000);
                      return t('距结束还有') + ` ${days} ` + t('天') + ` ${hours} ` + t('小时');
                    })()}
                  </div>
                )}
                <div style={{ marginTop: 8, display: 'flex', gap: 8 }}>
                  <Button
                    onClick={() => submitOption('FarmBetaEndTime')}
                    loading={loadingInput['FarmBetaEndTime']}
                  >
                    {t('设置结束时间')}
                  </Button>
                  <Button
                    type='warning'
                    onClick={async () => {
                      setInputs((prev) => ({ ...prev, FarmBetaEndTime: '' }));
                      await updateOption('FarmBetaEndTime', '');
                      showSuccess(t('设置已更新'));
                    }}
                  >
                    {t('清除')}
                  </Button>
                </div>
                <Text type='tertiary' size='small' style={{ marginTop: 4 }}>
                  {t('设置内测结束时间后，到达该时间将自动关闭农场并清空所有内测数据。用户访问时会看到内测已结束提示。')}
                </Text>
              </Form.Slot>
              <Form.Slot label={t('农场系统管理员账户')}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span>{t('管理员用户ID')}:</span>
                    <input
                      type='number'
                      value={inputs.FarmAdminUserId}
                      onChange={(e) => setInputs((prev) => ({ ...prev, FarmAdminUserId: e.target.value }))}
                      style={{ width: 120, padding: '4px 8px', borderRadius: 4, border: '1px solid var(--semi-color-border)', background: 'var(--semi-color-bg-1)', color: 'var(--semi-color-text-0)' }}
                      min={0}
                    />
                    <Button
                      onClick={() => submitOption('FarmAdminUserId')}
                      loading={loadingInput['FarmAdminUserId']}
                    >
                      {t('保存')}
                    </Button>
                  </div>
                  <Text type='tertiary' size='small'>
                    {t('设置后，用户出售物品时额度将从管理员余额划转，物品进入管理员仓库。设为0则关闭此功能（传统模式）。转生加成对所有出售操作生效。')}
                  </Text>
                </div>
              </Form.Slot>
              <Form.Slot label={t('仓库升级系统')}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <span>{t('最高等级')}:</span>
                      <input
                        type='number'
                        value={inputs.TgBotFarmWarehouseMaxLevel}
                        onChange={(e) => setInputs((prev) => ({ ...prev, TgBotFarmWarehouseMaxLevel: e.target.value }))}
                        style={{ width: 70, padding: '4px 8px', borderRadius: 4, border: '1px solid var(--semi-color-border)', background: 'var(--semi-color-bg-1)', color: 'var(--semi-color-text-0)' }}
                        min={1}
                      />
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <span>{t('升级基础价格(quota)')}:</span>
                      <input
                        type='number'
                        value={inputs.TgBotFarmWarehouseUpgradePrice}
                        onChange={(e) => setInputs((prev) => ({ ...prev, TgBotFarmWarehouseUpgradePrice: e.target.value }))}
                        style={{ width: 120, padding: '4px 8px', borderRadius: 4, border: '1px solid var(--semi-color-border)', background: 'var(--semi-color-bg-1)', color: 'var(--semi-color-text-0)' }}
                        min={0}
                      />
                    </div>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <span>{t('每级增加容量')}:</span>
                      <input
                        type='number'
                        value={inputs.TgBotFarmWarehouseCapacityPerLevel}
                        onChange={(e) => setInputs((prev) => ({ ...prev, TgBotFarmWarehouseCapacityPerLevel: e.target.value }))}
                        style={{ width: 80, padding: '4px 8px', borderRadius: 4, border: '1px solid var(--semi-color-border)', background: 'var(--semi-color-bg-1)', color: 'var(--semi-color-text-0)' }}
                        min={0}
                      />
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <span>{t('每级增加保质期%')}:</span>
                      <input
                        type='number'
                        value={inputs.TgBotFarmWarehouseExpiryBonusPerLevel}
                        onChange={(e) => setInputs((prev) => ({ ...prev, TgBotFarmWarehouseExpiryBonusPerLevel: e.target.value }))}
                        style={{ width: 80, padding: '4px 8px', borderRadius: 4, border: '1px solid var(--semi-color-border)', background: 'var(--semi-color-bg-1)', color: 'var(--semi-color-text-0)' }}
                        min={0}
                      />
                    </div>
                  </div>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Button
                      onClick={async () => {
                        setLoadingInput((prev) => ({ ...prev, TgBotFarmWarehouseUpgrade: true }));
                        try {
                          await updateOption('TgBotFarmWarehouseMaxLevel', inputs.TgBotFarmWarehouseMaxLevel);
                          await updateOption('TgBotFarmWarehouseUpgradePrice', inputs.TgBotFarmWarehouseUpgradePrice);
                          await updateOption('TgBotFarmWarehouseCapacityPerLevel', inputs.TgBotFarmWarehouseCapacityPerLevel);
                          await updateOption('TgBotFarmWarehouseExpiryBonusPerLevel', inputs.TgBotFarmWarehouseExpiryBonusPerLevel);
                          showSuccess(t('设置已更新'));
                        } catch (e) {
                          showError(t('设置更新失败'));
                        } finally {
                          setLoadingInput((prev) => ({ ...prev, TgBotFarmWarehouseUpgrade: false }));
                        }
                      }}
                      loading={loadingInput['TgBotFarmWarehouseUpgrade']}
                    >
                      {t('保存仓库升级设置')}
                    </Button>
                  </div>
                  <Text type='tertiary' size='small'>
                    {t('升级价格 = 基础价格 × 当前等级。每级提升容量和保质期。例如：基础价格200万quota，2级升3级花费400万quota。')}
                  </Text>
                </div>
              </Form.Slot>
              <Form.Input
                label={t('排行榜显示人数')}
                placeholder={t('设置签到排行榜显示前多少名，默认 100')}
                field={'CheckinLeaderboardLimit'}
                onChange={handleInputChange}
                type='number'
              />
              <Button
                onClick={() => submitOption('CheckinLeaderboardLimit')}
                loading={loadingInput['CheckinLeaderboardLimit']}
              >
                {t('设置排行榜人数')}
              </Button>
              <Form.TextArea
                label={t('关于')}
                placeholder={t(
                  '在此输入新的关于内容，支持 Markdown & HTML 代码。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性，这允许你设置任意网页作为关于页面',
                )}
                field={'About'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button onClick={submitAbout} loading={loadingInput['About']}>
                {t('设置关于')}
              </Button>
              {/*  */}
              <Banner
                fullMode={false}
                type='info'
                description={t(
                  '移除 One API 的版权标识必须首先获得授权，项目维护需要花费大量精力，如果本项目对你有意义，请主动支持本项目',
                )}
                closeIcon={null}
                style={{ marginTop: 15 }}
              />
              <Form.Input
                label={t('页脚')}
                placeholder={t(
                  '在此输入新的页脚，留空则使用默认页脚，支持 HTML 代码',
                )}
                field={'Footer'}
                onChange={handleInputChange}
              />
              <Button onClick={submitFooter} loading={loadingInput['Footer']}>
                {t('设置页脚')}
              </Button>
            </Form.Section>
          </Card>
        </Form>
      </Col>
      <Modal
        title={t('新版本') + '：' + updateData.tag_name}
        visible={showUpdateModal}
        onCancel={() => setShowUpdateModal(false)}
        footer={[
          <Button
            key='details'
            type='primary'
            onClick={() => {
              setShowUpdateModal(false);
              openGitHubRelease();
            }}
          >
            {t('详情')}
          </Button>,
        ]}
      >
        <div dangerouslySetInnerHTML={{ __html: updateData.content }}></div>
      </Modal>
    </Row>
  );
};

export default OtherSetting;
