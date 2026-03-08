import React, { useEffect, useState, useCallback } from 'react';
import {
  Banner,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Modal,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';

const PURPOSE_OPTIONS = [
  { value: 1, label: '余额兑换码' },
  { value: 2, label: '注册邀请码' },
];

const STATUS_OPTIONS = [
  { value: 1, label: '启用' },
  { value: 2, label: '禁用' },
];

const TgBotPage = () => {
  const { t } = useTranslation();

  // ===== Bot 设置 =====
  const [botToken, setBotToken] = useState('');
  const [botName, setBotName] = useState('');
  const [tokenSet, setTokenSet] = useState(false);
  const [maskedToken, setMaskedToken] = useState('');
  const [settingsLoading, setSettingsLoading] = useState(false);
  const [savingSettings, setSavingSettings] = useState(false);
  const [webhookInfo, setWebhookInfo] = useState(null);
  const [settingWebhook, setSettingWebhook] = useState(false);
  const [registeringCommands, setRegisteringCommands] = useState(false);

  // ===== 分类管理 =====
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingCategory, setEditingCategory] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  // ===== 库存管理 =====
  const [inventoryModalVisible, setInventoryModalVisible] = useState(false);
  const [inventoryCategory, setInventoryCategory] = useState(null);
  const [inventoryItems, setInventoryItems] = useState([]);
  const [inventoryLoading, setInventoryLoading] = useState(false);
  const [addCodesText, setAddCodesText] = useState('');
  const [addingCodes, setAddingCodes] = useState(false);

  // ===== 抽奖管理 =====
  const [lotteryEnabled, setLotteryEnabled] = useState(false);
  const [lotteryMessagesRequired, setLotteryMessagesRequired] = useState(10);
  const [lotteryWinRate, setLotteryWinRate] = useState(30);
  const [savingLottery, setSavingLottery] = useState(false);
  const [farmPlotPrice, setFarmPlotPrice] = useState(2000000);
  const [farmDogPrice, setFarmDogPrice] = useState(5000000);
  const [farmDogFoodPrice, setFarmDogFoodPrice] = useState(500000);
  const [farmDogGrowHours, setFarmDogGrowHours] = useState(24);
  const [farmDogGuardRate, setFarmDogGuardRate] = useState(50);
  const [farmWaterInterval, setFarmWaterInterval] = useState(7200);
  const [farmWiltDuration, setFarmWiltDuration] = useState(3600);
  const [farmEventChance, setFarmEventChance] = useState(30);
  const [farmDisasterChance, setFarmDisasterChance] = useState(15);
  const [farmStealCooldown, setFarmStealCooldown] = useState(1800);
  const [farmSoilMaxLevel, setFarmSoilMaxLevel] = useState(5);
  const [farmSoilUpgradePrice2, setFarmSoilUpgradePrice2] = useState(1000000);
  const [farmSoilUpgradePrice3, setFarmSoilUpgradePrice3] = useState(3000000);
  const [farmSoilUpgradePrice4, setFarmSoilUpgradePrice4] = useState(6000000);
  const [farmSoilUpgradePrice5, setFarmSoilUpgradePrice5] = useState(10000000);
  const [farmSoilSpeedBonus, setFarmSoilSpeedBonus] = useState(10);
  // 牧场
  const [ranchMaxAnimals, setRanchMaxAnimals] = useState(6);
  const [ranchFeedPrice, setRanchFeedPrice] = useState(200000);
  const [ranchWaterPrice, setRanchWaterPrice] = useState(100000);
  const [ranchFeedInterval, setRanchFeedInterval] = useState(14400);
  const [ranchWaterInterval, setRanchWaterInterval] = useState(10800);
  const [ranchHungerDeathHours, setRanchHungerDeathHours] = useState(24);
  const [ranchThirstDeathHours, setRanchThirstDeathHours] = useState(18);
  const [ranchChickenPrice, setRanchChickenPrice] = useState(500000);
  const [ranchDuckPrice, setRanchDuckPrice] = useState(800000);
  const [ranchGoosePrice, setRanchGoosePrice] = useState(1200000);
  const [ranchPigPrice, setRanchPigPrice] = useState(3000000);
  const [ranchSheepPrice, setRanchSheepPrice] = useState(4000000);
  const [ranchCowPrice, setRanchCowPrice] = useState(8000000);
  const [ranchChickenMeatPrice, setRanchChickenMeatPrice] = useState(1500000);
  const [ranchDuckMeatPrice, setRanchDuckMeatPrice] = useState(2500000);
  const [ranchGooseMeatPrice, setRanchGooseMeatPrice] = useState(4000000);
  const [ranchPigMeatPrice, setRanchPigMeatPrice] = useState(10000000);
  const [ranchSheepMeatPrice, setRanchSheepMeatPrice] = useState(14000000);
  const [ranchCowMeatPrice, setRanchCowMeatPrice] = useState(28000000);
  const [ranchManureInterval, setRanchManureInterval] = useState(21600);
  const [ranchManureCleanPrice, setRanchManureCleanPrice] = useState(150000);
  const [ranchManureGrowPenalty, setRanchManureGrowPenalty] = useState(30);
  // 等级系统
  const [farmUnlockSteal, setFarmUnlockSteal] = useState(2);
  const [farmUnlockDog, setFarmUnlockDog] = useState(2);
  const [farmUnlockRanch, setFarmUnlockRanch] = useState(3);
  const [farmUnlockFish, setFarmUnlockFish] = useState(3);
  const [farmUnlockWorkshop, setFarmUnlockWorkshop] = useState(4);
  const [farmUnlockMarket, setFarmUnlockMarket] = useState(2);
  const [farmUnlockTasks, setFarmUnlockTasks] = useState(1);
  const [farmUnlockAchieve, setFarmUnlockAchieve] = useState(1);
  const [farmLevelPrices, setFarmLevelPrices] = useState('500000,1000000,2000000,3000000,5000000,8000000,12000000,18000000,25000000,35000000,50000000,70000000,100000000,150000000');
  // 银行贷款
  const [farmBankAdminId, setFarmBankAdminId] = useState(1);
  const [farmBankInterestRate, setFarmBankInterestRate] = useState(10);
  const [farmBankMaxLoanDays, setFarmBankMaxLoanDays] = useState(7);
  const [farmBankBaseAmount, setFarmBankBaseAmount] = useState(50000000);
  const [farmBankMaxMultiplier, setFarmBankMaxMultiplier] = useState(10);
  const [farmBankUnlockLevel, setFarmBankUnlockLevel] = useState(3);
  const [farmMortgageMaxAmount, setFarmMortgageMaxAmount] = useState(500000000);
  const [farmMortgageInterestRate, setFarmMortgageInterestRate] = useState(15);
  // 季节系统
  const [farmSeasonDays, setFarmSeasonDays] = useState(7);
  const [farmSeasonInBonus, setFarmSeasonInBonus] = useState(70);
  const [farmSeasonOffBonus, setFarmSeasonOffBonus] = useState(140);
  const [farmWarehouseMaxSlots, setFarmWarehouseMaxSlots] = useState(100);
  // 农场公告
  const [farmAnnEnabled, setFarmAnnEnabled] = useState(false);
  const [farmAnnText, setFarmAnnText] = useState('');
  const [farmAnnType, setFarmAnnType] = useState('info');
  const [savingFarmAnn, setSavingFarmAnn] = useState(false);
  const [savingFarm, setSavingFarm] = useState(false);
  const [resetLevel, setResetLevel] = useState(1);
  // ===== 农场用户列表 =====
  const [farmUsers, setFarmUsers] = useState([]);
  const [farmUsersLoading, setFarmUsersLoading] = useState(false);
  const [lotteryPrizes, setLotteryPrizes] = useState([]);
  const [lotteryPrizesLoading, setLotteryPrizesLoading] = useState(false);
  const [lotteryPrizeTotal, setLotteryPrizeTotal] = useState(0);
  const [lotteryPrizeAvailable, setLotteryPrizeAvailable] = useState(0);
  const [addPrizeName, setAddPrizeName] = useState('');
  const [addPrizeCodes, setAddPrizeCodes] = useState('');
  const [addingPrizes, setAddingPrizes] = useState(false);

  // 加载系统设置
  const loadSettings = useCallback(async () => {
    setSettingsLoading(true);
    try {
      const res = await API.get('/api/tgbot/settings');
      if (res.data.success) {
        const data = res.data.data;
        setTokenSet(data.token_set || false);
        setMaskedToken(data.masked_token || '');
        setBotName(data.bot_name || '');
        setLotteryEnabled(data.lottery_enabled || false);
        setLotteryMessagesRequired(data.messages_required || 10);
        setLotteryWinRate(data.win_rate || 30);
        setFarmPlotPrice(data.farm_plot_price || 2000000);
        setFarmDogPrice(data.farm_dog_price || 5000000);
        setFarmDogFoodPrice(data.farm_dog_food_price || 500000);
        setFarmDogGrowHours(data.farm_dog_grow_hours || 24);
        setFarmDogGuardRate(data.farm_dog_guard_rate || 50);
        setFarmWaterInterval(data.farm_water_interval || 7200);
        setFarmWiltDuration(data.farm_wilt_duration || 3600);
        setFarmEventChance(data.farm_event_chance || 30);
        setFarmDisasterChance(data.farm_disaster_chance || 15);
        setFarmStealCooldown(data.farm_steal_cooldown || 1800);
        setFarmSoilMaxLevel(data.farm_soil_max_level || 5);
        setFarmSoilUpgradePrice2(data.farm_soil_upgrade_price_2 || 1000000);
        setFarmSoilUpgradePrice3(data.farm_soil_upgrade_price_3 || 3000000);
        setFarmSoilUpgradePrice4(data.farm_soil_upgrade_price_4 || 6000000);
        setFarmSoilUpgradePrice5(data.farm_soil_upgrade_price_5 || 10000000);
        setFarmSoilSpeedBonus(data.farm_soil_speed_bonus || 10);
        // 牧场
        setRanchMaxAnimals(data.ranch_max_animals || 6);
        setRanchFeedPrice(data.ranch_feed_price || 200000);
        setRanchWaterPrice(data.ranch_water_price || 100000);
        setRanchFeedInterval(data.ranch_feed_interval || 14400);
        setRanchWaterInterval(data.ranch_water_interval || 10800);
        setRanchHungerDeathHours(data.ranch_hunger_death_hours || 24);
        setRanchThirstDeathHours(data.ranch_thirst_death_hours || 18);
        setRanchChickenPrice(data.ranch_chicken_price || 500000);
        setRanchDuckPrice(data.ranch_duck_price || 800000);
        setRanchGoosePrice(data.ranch_goose_price || 1200000);
        setRanchPigPrice(data.ranch_pig_price || 3000000);
        setRanchSheepPrice(data.ranch_sheep_price || 4000000);
        setRanchCowPrice(data.ranch_cow_price || 8000000);
        setRanchChickenMeatPrice(data.ranch_chicken_meat_price || 1500000);
        setRanchDuckMeatPrice(data.ranch_duck_meat_price || 2500000);
        setRanchGooseMeatPrice(data.ranch_goose_meat_price || 4000000);
        setRanchPigMeatPrice(data.ranch_pig_meat_price || 10000000);
        setRanchSheepMeatPrice(data.ranch_sheep_meat_price || 14000000);
        setRanchCowMeatPrice(data.ranch_cow_meat_price || 28000000);
        setRanchManureInterval(data.ranch_manure_interval || 21600);
        setRanchManureCleanPrice(data.ranch_manure_clean_price || 150000);
        setRanchManureGrowPenalty(data.ranch_manure_grow_penalty || 30);
        // 等级系统
        setFarmUnlockSteal(data.farm_unlock_steal ?? 2);
        setFarmUnlockDog(data.farm_unlock_dog ?? 2);
        setFarmUnlockRanch(data.farm_unlock_ranch ?? 3);
        setFarmUnlockFish(data.farm_unlock_fish ?? 3);
        setFarmUnlockWorkshop(data.farm_unlock_workshop ?? 4);
        setFarmUnlockMarket(data.farm_unlock_market ?? 2);
        setFarmUnlockTasks(data.farm_unlock_tasks ?? 1);
        setFarmUnlockAchieve(data.farm_unlock_achieve ?? 1);
        if (data.farm_level_prices) setFarmLevelPrices(data.farm_level_prices);
        // 银行贷款
        setFarmBankAdminId(data.farm_bank_admin_id ?? 1);
        setFarmBankInterestRate(data.farm_bank_interest_rate ?? 10);
        setFarmBankMaxLoanDays(data.farm_bank_max_loan_days ?? 7);
        setFarmBankBaseAmount(data.farm_bank_base_amount ?? 50000000);
        setFarmBankMaxMultiplier(data.farm_bank_max_multiplier ?? 10);
        setFarmBankUnlockLevel(data.farm_bank_unlock_level ?? 3);
        setFarmMortgageMaxAmount(data.farm_mortgage_max_amount ?? 500000000);
        setFarmMortgageInterestRate(data.farm_mortgage_interest_rate ?? 15);
        // 季节系统
        setFarmSeasonDays(data.farm_season_days ?? 7);
        setFarmSeasonInBonus(data.farm_season_in_bonus ?? 70);
        setFarmSeasonOffBonus(data.farm_season_off_bonus ?? 140);
        setFarmWarehouseMaxSlots(data.farm_warehouse_max_slots ?? 100);
        // 农场公告
        setFarmAnnEnabled(data.farm_announcement_enabled === true || data.farm_announcement_enabled === 'true');
        setFarmAnnText(data.farm_announcement_text || '');
        setFarmAnnType(data.farm_announcement_type || 'info');
        setBotToken('');
      }
    } catch (err) {
      showError(t('加载设置失败'));
    } finally {
      setSettingsLoading(false);
    }
  }, [t]);

  // 保存 Bot 设置
  const saveSettings = async () => {
    setSavingSettings(true);
    try {
      const options = [
        { key: 'TelegramBotName', value: botName },
      ];
      if (botToken.trim()) {
        options.unshift({ key: 'TelegramBotToken', value: botToken });
      }
      for (const opt of options) {
        const res = await API.put('/api/option/', opt);
        if (!res.data.success) {
          showError(res.data.message);
          setSavingSettings(false);
          return;
        }
      }
      showSuccess(t('保存成功'));
      await loadSettings();
    } catch (err) {
      showError(t('保存失败'));
    } finally {
      setSavingSettings(false);
    }
  };

  // 注册命令菜单
  const registerCommands = async () => {
    if (!botToken && !tokenSet) {
      showError(t('请先填写并保存 Bot Token'));
      return;
    }
    setRegisteringCommands(true);
    try {
      const res = await API.post('/api/tgbot/register-commands');
      if (res.data.success) {
        showSuccess(res.data.message || t('命令菜单注册成功'));
      } else {
        showError(res.data.message || t('注册失败'));
      }
    } catch (err) {
      showError(err.response?.data?.message || t('注册失败'));
    } finally {
      setRegisteringCommands(false);
    }
  };

  // 设置 Webhook
  const setupWebhook = async () => {
    if (!botToken && !tokenSet) {
      showError(t('请先填写并保存 Bot Token'));
      return;
    }
    setSettingWebhook(true);
    try {
      const res = await API.post('/api/tgbot/setup-webhook');
      if (res.data.success) {
        showSuccess(t('Webhook 设置成功'));
        loadWebhookInfo();
      } else {
        showError(res.data.message || t('设置失败'));
      }
    } catch (err) {
      showError(err.response?.data?.message || t('设置失败'));
    } finally {
      setSettingWebhook(false);
    }
  };

  // 获取 Webhook 信息
  const loadWebhookInfo = useCallback(async () => {
    try {
      const res = await API.get('/api/tgbot/webhook-info');
      if (res.data.success) {
        setWebhookInfo(res.data.data);
      }
    } catch {
      // ignore
    }
  }, []);

  // 加载分类
  const loadCategories = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/tgbot/category/');
      if (res.data.success) {
        setCategories(res.data.data || []);
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('加载失败'));
    } finally {
      setLoading(false);
    }
  };

  // ===== 抽奖管理 API =====
  const loadLotteryPrizes = async () => {
    setLotteryPrizesLoading(true);
    try {
      const res = await API.get('/api/tgbot/lottery/prizes');
      if (res.data.success) {
        setLotteryPrizes(res.data.data || []);
        setLotteryPrizeTotal(res.data.total || 0);
        setLotteryPrizeAvailable(res.data.available || 0);
      }
    } catch {
      // ignore
    } finally {
      setLotteryPrizesLoading(false);
    }
  };

  const saveLotterySettings = async () => {
    setSavingLottery(true);
    try {
      const options = [
        { key: 'TgBotLotteryEnabled', value: String(lotteryEnabled) },
        { key: 'TgBotLotteryMessagesRequired', value: String(lotteryMessagesRequired) },
        { key: 'TgBotLotteryWinRate', value: String(lotteryWinRate) },
      ];
      for (const opt of options) {
        const res = await API.put('/api/option/', opt);
        if (!res.data.success) {
          showError(res.data.message);
          setSavingLottery(false);
          return;
        }
      }
      showSuccess(t('抽奖设置保存成功'));
      await loadSettings();
    } catch (err) {
      showError(t('保存失败'));
    } finally {
      setSavingLottery(false);
    }
  };

  const handleAddPrizes = async () => {
    if (!addPrizeName.trim() || !addPrizeCodes.trim()) return;
    setAddingPrizes(true);
    try {
      const res = await API.post('/api/tgbot/lottery/prizes', {
        name: addPrizeName,
        codes: addPrizeCodes,
      });
      if (res.data.success) {
        showSuccess(res.data.message || t('添加成功'));
        setAddPrizeName('');
        setAddPrizeCodes('');
        await loadLotteryPrizes();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('添加失败'));
    } finally {
      setAddingPrizes(false);
    }
  };

  const handleDeletePrize = async (prizeId) => {
    Modal.confirm({
      title: t('确认删除'),
      content: t('确定要删除该奖品吗？'),
      onOk: async () => {
        try {
          const res = await API.delete(`/api/tgbot/lottery/prizes/${prizeId}`);
          if (res.data.success) {
            showSuccess(t('删除成功'));
            await loadLotteryPrizes();
          } else {
            showError(res.data.message);
          }
        } catch (err) {
          showError(err.response?.data?.message || t('删除失败'));
        }
      },
    });
  };

  const lotteryPrizeColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: t('奖品名称'), dataIndex: 'name', width: 120 },
    {
      title: t('兑换码'),
      dataIndex: 'code',
      width: 200,
      render: (text) => (
        <Typography.Text copyable style={{ fontFamily: 'monospace' }}>
          {text}
        </Typography.Text>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 80,
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'grey'}>
          {status === 1 ? t('可用') : t('已中奖')}
        </Tag>
      ),
    },
    {
      title: t('中奖者'),
      dataIndex: 'won_by',
      width: 120,
      render: (text) => text || '-',
    },
    {
      title: t('操作'),
      width: 80,
      render: (_, record) => (
        <Button
          size='small'
          type='danger'
          onClick={() => handleDeletePrize(record.id)}
        >
          {t('删除')}
        </Button>
      ),
    },
  ];

  const loadFarmUsers = useCallback(async () => {
    setFarmUsersLoading(true);
    try {
      const { data: res } = await API.get('/api/tgbot/farm/users');
      if (res.success) setFarmUsers(res.data || []);
    } catch (err) { /* ignore */ }
    finally { setFarmUsersLoading(false); }
  }, []);

  useEffect(() => {
    loadSettings();
    loadCategories();
    loadWebhookInfo();
    loadLotteryPrizes();
    loadFarmUsers();
  }, [loadSettings, loadWebhookInfo, loadFarmUsers]);

  // ===== 分类 CRUD =====
  const openCreateModal = () => {
    setEditingCategory(null);
    setModalVisible(true);
  };

  const openEditModal = (record) => {
    setEditingCategory(record);
    setModalVisible(true);
  };

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      const payload = {
        ...values,
        max_claims: Number(values.max_claims) || 1,
      };
      if (editingCategory) {
        payload.id = editingCategory.id;
        const res = await API.put('/api/tgbot/category/', payload);
        if (res.data.success) {
          showSuccess(t('更新成功'));
        } else {
          showError(res.data.message);
          return;
        }
      } else {
        const res = await API.post('/api/tgbot/category/', payload);
        if (res.data.success) {
          showSuccess(t('创建成功'));
        } else {
          showError(res.data.message);
          return;
        }
      }
      setModalVisible(false);
      loadCategories();
    } catch (err) {
      showError(err.response?.data?.message || t('操作失败'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id) => {
    Modal.confirm({
      title: t('确认删除'),
      content: t('删除后不可恢复，确定要删除该分类吗？'),
      onOk: async () => {
        try {
          const res = await API.delete(`/api/tgbot/category/${id}`);
          if (res.data.success) {
            showSuccess(t('删除成功'));
            loadCategories();
          } else {
            showError(res.data.message);
          }
        } catch (err) {
          showError(err.response?.data?.message || t('删除失败'));
        }
      },
    });
  };

  const handleToggleStatus = async (record) => {
    const newStatus = record.status === 1 ? 2 : 1;
    try {
      const res = await API.put('/api/tgbot/category/', {
        id: record.id,
        name: record.name,
        description: record.description,
        max_claims: record.max_claims,
        purpose: record.purpose,
        status: newStatus,
      });
      if (res.data.success) {
        showSuccess(newStatus === 1 ? t('已启用') : t('已禁用'));
        loadCategories();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('操作失败'));
    }
  };

  // ===== 库存管理 API =====
  const openInventoryModal = async (category) => {
    setInventoryCategory(category);
    setInventoryModalVisible(true);
    setAddCodesText('');
    await loadInventory(category.id);
  };

  const loadInventory = async (categoryId) => {
    setInventoryLoading(true);
    try {
      const res = await API.get(`/api/tgbot/inventory/?category_id=${categoryId}`);
      if (res.data.success) {
        setInventoryItems(res.data.data || []);
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('加载库存失败'));
    } finally {
      setInventoryLoading(false);
    }
  };

  const handleAddCodes = async () => {
    if (!addCodesText.trim() || !inventoryCategory) return;
    setAddingCodes(true);
    try {
      const res = await API.post('/api/tgbot/inventory/', {
        category_id: inventoryCategory.id,
        codes: addCodesText,
      });
      if (res.data.success) {
        showSuccess(res.data.message || t('添加成功'));
        setAddCodesText('');
        await loadInventory(inventoryCategory.id);
        loadCategories();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.response?.data?.message || t('添加失败'));
    } finally {
      setAddingCodes(false);
    }
  };

  const handleDeleteInventoryItem = async (itemId) => {
    Modal.confirm({
      title: t('确认删除'),
      content: t('确定要删除该兑换码吗？'),
      onOk: async () => {
        try {
          const res = await API.delete(`/api/tgbot/inventory/${itemId}`);
          if (res.data.success) {
            showSuccess(t('删除成功'));
            if (inventoryCategory) {
              await loadInventory(inventoryCategory.id);
            }
            loadCategories();
          } else {
            showError(res.data.message);
          }
        } catch (err) {
          showError(err.response?.data?.message || t('删除失败'));
        }
      },
    });
  };

  const inventoryColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    {
      title: t('兑换码/邀请码'),
      dataIndex: 'code',
      width: 250,
      render: (text) => (
        <Typography.Text copyable style={{ fontFamily: 'monospace' }}>
          {text}
        </Typography.Text>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 100,
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'grey'}>
          {status === 1 ? t('可用') : t('已发放')}
        </Tag>
      ),
    },
    {
      title: t('领取者'),
      dataIndex: 'claimed_by',
      width: 150,
      render: (text) => text || '-',
    },
    {
      title: t('操作'),
      width: 80,
      render: (_, record) => (
        <Button
          size='small'
          type='danger'
          onClick={() => handleDeleteInventoryItem(record.id)}
        >
          {t('删除')}
        </Button>
      ),
    },
  ];

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: t('分类名称'), dataIndex: 'name', width: 150 },
    {
      title: t('描述'),
      dataIndex: 'description',
      width: 200,
      render: (text) => text || '-',
    },
    {
      title: t('兑换码类型'),
      dataIndex: 'purpose',
      width: 120,
      render: (purpose) => {
        const opt = PURPOSE_OPTIONS.find((o) => o.value === purpose);
        return (
          <Tag color={purpose === 2 ? 'blue' : 'green'}>
            {opt ? t(opt.label) : t('未知')}
          </Tag>
        );
      },
    },
    {
      title: t('库存(可用/总数)'),
      width: 130,
      render: (_, record) => {
        const available = record.stock_available ?? 0;
        const total = record.stock_total ?? 0;
        const color = available === 0 ? 'red' : available <= 5 ? 'orange' : 'green';
        return (
          <Tag color={color}>
            {available} / {total}
          </Tag>
        );
      },
    },
    { title: t('每人可领取次数'), dataIndex: 'max_claims', width: 130 },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 80,
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'grey'}>
          {status === 1 ? t('启用') : t('禁用')}
        </Tag>
      ),
    },
    {
      title: t('操作'),
      width: 300,
      fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button
            size='small'
            theme='light'
            type='primary'
            onClick={() => openInventoryModal(record)}
          >
            {t('管理库存')}
          </Button>
          <Button size='small' onClick={() => openEditModal(record)}>
            {t('编辑')}
          </Button>
          <Button
            size='small'
            type={record.status === 1 ? 'warning' : 'primary'}
            onClick={() => handleToggleStatus(record)}
          >
            {record.status === 1 ? t('禁用') : t('启用')}
          </Button>
          <Button
            size='small'
            type='danger'
            onClick={() => handleDelete(record.id)}
          >
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  const cardStyle = {
    boxShadow: '0 1px 3px rgba(0,0,0,0.04), 0 4px 16px rgba(0,0,0,0.02)',
    border: '1px solid var(--semi-color-border)',
  };

  return (
    <div className='mt-[60px] px-2 space-y-4'>
      {/* ===== Bot 基本设置 ===== */}
      <Card className='!rounded-2xl' style={cardStyle}>
        <Typography.Title heading={5} style={{ marginBottom: 16 }}>
          {t('TG 机器人设置')}
        </Typography.Title>

        <Spin spinning={settingsLoading}>
          <Form labelPosition='left' labelWidth={140}>
            <Form.Input
              field='TelegramBotToken'
              label='Bot Token'
              placeholder={tokenSet ? maskedToken : t('从 @BotFather 获取的 Bot Token')}
              type='password'
              mode='password'
              value={botToken}
              onChange={setBotToken}
              extraText={tokenSet ? t('Token 已配置，留空则保持不变，输入新值可更新') : t('在 Telegram 中找 @BotFather 创建机器人获取 Token')}
            />
            <Form.Input
              field='TelegramBotName'
              label={t('Bot 用户名')}
              placeholder={t('如：my_cool_bot')}
              value={botName}
              onChange={setBotName}
              extraText={t('机器人的用户名（不含 @）')}
            />
          </Form>

          <div className='flex items-center gap-3 mt-4'>
            <Button
              theme='solid'
              type='primary'
              loading={savingSettings}
              onClick={saveSettings}
            >
              {t('保存设置')}
            </Button>
            <Button
              theme='light'
              type='tertiary'
              loading={settingWebhook}
              onClick={setupWebhook}
            >
              {t('设置 Webhook')}
            </Button>
            <Button
              theme='light'
              type='tertiary'
              loading={registeringCommands}
              onClick={registerCommands}
            >
              {t('注册命令菜单')}
            </Button>
          </div>

          {webhookInfo && (
            <div className='mt-4'>
              <Descriptions
                size='small'
                row
                data={[
                  {
                    key: 'Webhook URL',
                    value: webhookInfo.url || t('未设置'),
                  },
                  {
                    key: t('状态'),
                    value: webhookInfo.url ? (
                      <Tag color='green'>{t('已设置')}</Tag>
                    ) : (
                      <Tag color='red'>{t('未设置')}</Tag>
                    ),
                  },
                  {
                    key: t('待处理更新'),
                    value: webhookInfo.pending_update_count ?? '-',
                  },
                ]}
              />
            </div>
          )}

          <Banner
            type='info'
            className='!rounded-lg mt-4'
            closeIcon={null}
            description={t(
              '用户在 Telegram 群组中与机器人交互时，会自动创建系统账户。管理员需要在下方添加分类，每个分类对应一个按钮，用户点击按钮即可领取对应的兑换码。',
            )}
          />
        </Spin>
      </Card>

      {/* ===== 分类管理 ===== */}
      <Card className='!rounded-2xl' style={cardStyle}>
        <div className='flex items-center justify-between mb-4 flex-wrap gap-2'>
          <Typography.Title heading={5} style={{ marginBottom: 0 }}>
            {t('领取分类管理')}
          </Typography.Title>
          <Button theme='solid' type='primary' onClick={openCreateModal}>
            {t('添加分类')}
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={categories}
          loading={loading}
          rowKey='id'
          pagination={false}
          scroll={{ x: 1100 }}
          empty={
            <div className='py-8 text-center text-gray-400'>
              {t('暂无分类，请添加')}
            </div>
          }
        />
      </Card>

      {/* ===== 农场游戏设置 ===== */}
      <Card
        title={
          <span>{t('农场游戏设置')}</span>
        }
        className='mt-4'
      >
        <div className='mb-4'>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('土地价格(额度)')}</Typography.Text>
            <InputNumber value={farmPlotPrice} onChange={setFarmPlotPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (farmPlotPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('买狗价格(额度)')}</Typography.Text>
            <InputNumber value={farmDogPrice} onChange={setFarmDogPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (farmDogPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('狗粮价格(额度)')}</Typography.Text>
            <InputNumber value={farmDogFoodPrice} onChange={setFarmDogFoodPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (farmDogFoodPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('小狗成长时间(小时)')}</Typography.Text>
            <InputNumber value={farmDogGrowHours} onChange={setFarmDogGrowHours} min={1} max={720} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('看门狗拦截率(%)')}</Typography.Text>
            <InputNumber value={farmDogGuardRate} onChange={setFarmDogGuardRate} min={0} max={100} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('浇水间隔(秒)')}</Typography.Text>
            <InputNumber value={farmWaterInterval} onChange={setFarmWaterInterval} min={60} step={600} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{(farmWaterInterval / 3600).toFixed(1) + t('小时')}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('枯萎到死亡时间(秒)')}</Typography.Text>
            <InputNumber value={farmWiltDuration} onChange={setFarmWiltDuration} min={60} step={600} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{(farmWiltDuration / 3600).toFixed(1) + t('小时')}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('虫害概率(%)')}</Typography.Text>
            <InputNumber value={farmEventChance} onChange={setFarmEventChance} min={0} max={100} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('天灾干旱概率(%)')}</Typography.Text>
            <InputNumber value={farmDisasterChance} onChange={setFarmDisasterChance} min={0} max={100} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('偷菜冷却时间(秒)')}</Typography.Text>
            <InputNumber value={farmStealCooldown} onChange={setFarmStealCooldown} min={0} step={60} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{(farmStealCooldown / 60).toFixed(0) + t('分钟')}</Typography.Text>
          </div>

          <div style={{ borderTop: '1px solid var(--semi-color-border)', paddingTop: 16, marginTop: 16, marginBottom: 12 }}>
            <Typography.Title heading={6} style={{ marginBottom: 12 }}>{t('泥土升级设置')}</Typography.Title>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('泥土最高等级')}</Typography.Text>
            <InputNumber value={farmSoilMaxLevel} onChange={setFarmSoilMaxLevel} min={1} max={10} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('每级加速百分比(%)')}</Typography.Text>
            <InputNumber value={farmSoilSpeedBonus} onChange={setFarmSoilSpeedBonus} min={0} max={50} style={{ width: 120 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{t('满级加速') + ': ' + (farmSoilSpeedBonus * (farmSoilMaxLevel - 1)) + '%'}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('升级到2级价格(额度)')}</Typography.Text>
            <InputNumber value={farmSoilUpgradePrice2} onChange={setFarmSoilUpgradePrice2} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (farmSoilUpgradePrice2 / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('升级到3级价格(额度)')}</Typography.Text>
            <InputNumber value={farmSoilUpgradePrice3} onChange={setFarmSoilUpgradePrice3} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (farmSoilUpgradePrice3 / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('升级到4级价格(额度)')}</Typography.Text>
            <InputNumber value={farmSoilUpgradePrice4} onChange={setFarmSoilUpgradePrice4} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (farmSoilUpgradePrice4 / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('升级到5级价格(额度)')}</Typography.Text>
            <InputNumber value={farmSoilUpgradePrice5} onChange={setFarmSoilUpgradePrice5} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (farmSoilUpgradePrice5 / 500000).toFixed(2)}</Typography.Text>
          </div>
          <Typography.Title heading={6} style={{ marginTop: 16, marginBottom: 12 }}>🐄 {t('牧场设置')}</Typography.Title>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('最大养殖数量')}</Typography.Text>
            <InputNumber value={ranchMaxAnimals} onChange={setRanchMaxAnimals} min={1} max={20} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('饲料价格(额度)')}</Typography.Text>
            <InputNumber value={ranchFeedPrice} onChange={setRanchFeedPrice} min={0} step={10000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchFeedPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('饮水价格(额度)')}</Typography.Text>
            <InputNumber value={ranchWaterPrice} onChange={setRanchWaterPrice} min={0} step={10000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchWaterPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('喂食间隔(秒)')}</Typography.Text>
            <InputNumber value={ranchFeedInterval} onChange={setRanchFeedInterval} min={60} step={3600} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{(ranchFeedInterval / 3600).toFixed(1) + 'h'}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('喂水间隔(秒)')}</Typography.Text>
            <InputNumber value={ranchWaterInterval} onChange={setRanchWaterInterval} min={60} step={3600} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{(ranchWaterInterval / 3600).toFixed(1) + 'h'}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('断食死亡(小时)')}</Typography.Text>
            <InputNumber value={ranchHungerDeathHours} onChange={setRanchHungerDeathHours} min={1} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 200 }}>{t('断水死亡(小时)')}</Typography.Text>
            <InputNumber value={ranchThirstDeathHours} onChange={setRanchThirstDeathHours} min={1} style={{ width: 120 }} />
          </div>
          <Typography.Title heading={6} style={{ marginTop: 8, marginBottom: 8 }}>{t('动物购买价格')}</Typography.Title>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🐔 {t('鸡(额度)')}</Typography.Text>
            <InputNumber value={ranchChickenPrice} onChange={setRanchChickenPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchChickenPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🦆 {t('鸭(额度)')}</Typography.Text>
            <InputNumber value={ranchDuckPrice} onChange={setRanchDuckPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchDuckPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🪿 {t('鹅(额度)')}</Typography.Text>
            <InputNumber value={ranchGoosePrice} onChange={setRanchGoosePrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchGoosePrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🐷 {t('猪(额度)')}</Typography.Text>
            <InputNumber value={ranchPigPrice} onChange={setRanchPigPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchPigPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🐑 {t('羊(额度)')}</Typography.Text>
            <InputNumber value={ranchSheepPrice} onChange={setRanchSheepPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchSheepPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🐄 {t('牛(额度)')}</Typography.Text>
            <InputNumber value={ranchCowPrice} onChange={setRanchCowPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchCowPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <Typography.Title heading={6} style={{ marginTop: 8, marginBottom: 8 }}>{t('肉类出售价格')}</Typography.Title>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🐔 {t('鸡肉(额度)')}</Typography.Text>
            <InputNumber value={ranchChickenMeatPrice} onChange={setRanchChickenMeatPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchChickenMeatPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🦆 {t('鸭肉(额度)')}</Typography.Text>
            <InputNumber value={ranchDuckMeatPrice} onChange={setRanchDuckMeatPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchDuckMeatPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🪿 {t('鹅肉(额度)')}</Typography.Text>
            <InputNumber value={ranchGooseMeatPrice} onChange={setRanchGooseMeatPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchGooseMeatPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🐷 {t('猪肉(额度)')}</Typography.Text>
            <InputNumber value={ranchPigMeatPrice} onChange={setRanchPigMeatPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchPigMeatPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🐑 {t('羊肉(额度)')}</Typography.Text>
            <InputNumber value={ranchSheepMeatPrice} onChange={setRanchSheepMeatPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchSheepMeatPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🐄 {t('牛肉(额度)')}</Typography.Text>
            <InputNumber value={ranchCowMeatPrice} onChange={setRanchCowMeatPrice} min={0} step={100000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchCowMeatPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <Typography.Title heading={6} style={{ marginTop: 8, marginBottom: 8 }}>💩 {t('粪便清理')}</Typography.Title>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>{t('清理间隔(秒)')}</Typography.Text>
            <InputNumber value={ranchManureInterval} onChange={setRanchManureInterval} min={60} step={3600} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{(ranchManureInterval / 3600).toFixed(1) + 'h'}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>{t('清理费用(额度)')}</Typography.Text>
            <InputNumber value={ranchManureCleanPrice} onChange={setRanchManureCleanPrice} min={0} step={10000} style={{ width: 180 }} />
            <Typography.Text type='tertiary' style={{ marginLeft: 8 }}>{'$' + (ranchManureCleanPrice / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>{t('脏污生长减速(%)')}</Typography.Text>
            <InputNumber value={ranchManureGrowPenalty} onChange={setRanchManureGrowPenalty} min={0} max={90} style={{ width: 120 }} />
          </div>
          <Typography.Title heading={6} style={{ marginTop: 16, marginBottom: 12 }}>⭐ {t('等级系统')}</Typography.Title>
          <Typography.Title heading={6} style={{ marginTop: 8, marginBottom: 8 }}>{t('功能解锁等级')}</Typography.Title>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🕵️ {t('偷菜解锁等级')}</Typography.Text>
            <InputNumber value={farmUnlockSteal} onChange={setFarmUnlockSteal} min={1} max={15} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🐕 {t('狗狗解锁等级')}</Typography.Text>
            <InputNumber value={farmUnlockDog} onChange={setFarmUnlockDog} min={1} max={15} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🐄 {t('牧场解锁等级')}</Typography.Text>
            <InputNumber value={farmUnlockRanch} onChange={setFarmUnlockRanch} min={1} max={15} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🎣 {t('钓鱼解锁等级')}</Typography.Text>
            <InputNumber value={farmUnlockFish} onChange={setFarmUnlockFish} min={1} max={15} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🏭 {t('加工坊解锁等级')}</Typography.Text>
            <InputNumber value={farmUnlockWorkshop} onChange={setFarmUnlockWorkshop} min={1} max={15} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>📈 {t('市场解锁等级')}</Typography.Text>
            <InputNumber value={farmUnlockMarket} onChange={setFarmUnlockMarket} min={1} max={15} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>📝 {t('任务解锁等级')}</Typography.Text>
            <InputNumber value={farmUnlockTasks} onChange={setFarmUnlockTasks} min={1} max={15} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🏆 {t('成就解锁等级')}</Typography.Text>
            <InputNumber value={farmUnlockAchieve} onChange={setFarmUnlockAchieve} min={1} max={15} style={{ width: 120 }} />
          </div>
          <Typography.Title heading={6} style={{ marginTop: 8, marginBottom: 8 }}>{t('升级价格(额度,逗号分隔,Lv2~Lv15)')}</Typography.Title>
          <div style={{ marginBottom: 12 }}>
            <Input value={farmLevelPrices} onChange={setFarmLevelPrices} placeholder='500000,1000000,...' style={{ width: '100%' }} />
            <Typography.Text type='tertiary' size='small'>
              {farmLevelPrices.split(',').map((p, i) => `Lv${i+2}=$${(parseInt(p.trim()) / 500000 || 0).toFixed(2)}`).join(' | ')}
            </Typography.Text>
          </div>
          <Typography.Title heading={6} style={{ marginTop: 16, marginBottom: 8 }}>🏦 {t('银行贷款设置')}</Typography.Title>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>👤 {t('银行管理员ID')}</Typography.Text>
            <InputNumber value={farmBankAdminId} onChange={setFarmBankAdminId} min={1} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>💰 {t('信用贷最低额度(quota)')}</Typography.Text>
            <InputNumber value={farmBankBaseAmount} onChange={setFarmBankBaseAmount} min={1} style={{ width: 180 }} />
            <Typography.Text type='tertiary' size='small' style={{ marginLeft: 8 }}>= ${(farmBankBaseAmount / 500000).toFixed(2)} ({t('评分1时可贷额度')})</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>📈 {t('信用贷利率(%)')}</Typography.Text>
            <InputNumber value={farmBankInterestRate} onChange={setFarmBankInterestRate} min={1} max={100} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>📅 {t('最长还款天数')}</Typography.Text>
            <InputNumber value={farmBankMaxLoanDays} onChange={setFarmBankMaxLoanDays} min={1} max={30} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>⭐ {t('信用评分最高倍率')}</Typography.Text>
            <InputNumber value={farmBankMaxMultiplier} onChange={setFarmBankMaxMultiplier} min={1} max={20} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🔓 {t('银行解锁等级')}</Typography.Text>
            <InputNumber value={farmBankUnlockLevel} onChange={setFarmBankUnlockLevel} min={1} max={15} style={{ width: 120 }} />
          </div>
          <Typography.Title heading={6} style={{ marginTop: 16, marginBottom: 8 }}>🏠 {t('抵押贷款设置')}</Typography.Title>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>💰 {t('抵押贷最高额度(quota)')}</Typography.Text>
            <InputNumber value={farmMortgageMaxAmount} onChange={setFarmMortgageMaxAmount} min={1} style={{ width: 180 }} />
            <Typography.Text type='tertiary' size='small' style={{ marginLeft: 8 }}>= ${(farmMortgageMaxAmount / 500000).toFixed(2)}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>📈 {t('抵押贷利率(%)')}</Typography.Text>
            <InputNumber value={farmMortgageInterestRate} onChange={setFarmMortgageInterestRate} min={1} max={100} style={{ width: 120 }} />
          </div>
          <Typography.Title heading={6} style={{ marginTop: 16, marginBottom: 8 }}>🌸 {t('季节系统设置')}</Typography.Title>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>📅 {t('每季天数')}</Typography.Text>
            <InputNumber value={farmSeasonDays} onChange={setFarmSeasonDays} min={1} max={90} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>🏷️ {t('应季价格倍率(%)')}</Typography.Text>
            <InputNumber value={farmSeasonInBonus} onChange={setFarmSeasonInBonus} min={10} max={100} style={{ width: 120 }} />
            <Typography.Text type='tertiary' size='small' style={{ marginLeft: 8 }}>{t('越低越便宜')}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>📈 {t('反季价格倍率(%)')}</Typography.Text>
            <InputNumber value={farmSeasonOffBonus} onChange={setFarmSeasonOffBonus} min={100} max={300} style={{ width: 120 }} />
            <Typography.Text type='tertiary' size='small' style={{ marginLeft: 8 }}>{t('越高越贵')}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
            <Typography.Text style={{ width: 200 }}>📦 {t('仓库最大容量')}</Typography.Text>
            <InputNumber value={farmWarehouseMaxSlots} onChange={setFarmWarehouseMaxSlots} min={10} max={1000} style={{ width: 120 }} />
          </div>

          <Button
            theme='solid'
            type='primary'
            loading={savingFarm}
            onClick={async () => {
              setSavingFarm(true);
              try {
                const farmOptions = [
                  { key: 'TgBotFarmPlotPrice', value: String(farmPlotPrice) },
                  { key: 'TgBotFarmDogPrice', value: String(farmDogPrice) },
                  { key: 'TgBotFarmDogFoodPrice', value: String(farmDogFoodPrice) },
                  { key: 'TgBotFarmDogGrowHours', value: String(farmDogGrowHours) },
                  { key: 'TgBotFarmDogGuardRate', value: String(farmDogGuardRate) },
                  { key: 'TgBotFarmWaterInterval', value: String(farmWaterInterval) },
                  { key: 'TgBotFarmWiltDuration', value: String(farmWiltDuration) },
                  { key: 'TgBotFarmEventChance', value: String(farmEventChance) },
                  { key: 'TgBotFarmDisasterChance', value: String(farmDisasterChance) },
                  { key: 'TgBotFarmStealCooldown', value: String(farmStealCooldown) },
                  { key: 'TgBotFarmSoilMaxLevel', value: String(farmSoilMaxLevel) },
                  { key: 'TgBotFarmSoilUpgradePrice2', value: String(farmSoilUpgradePrice2) },
                  { key: 'TgBotFarmSoilUpgradePrice3', value: String(farmSoilUpgradePrice3) },
                  { key: 'TgBotFarmSoilUpgradePrice4', value: String(farmSoilUpgradePrice4) },
                  { key: 'TgBotFarmSoilUpgradePrice5', value: String(farmSoilUpgradePrice5) },
                  { key: 'TgBotFarmSoilSpeedBonus', value: String(farmSoilSpeedBonus) },
                  // 牧场
                  { key: 'TgBotRanchMaxAnimals', value: String(ranchMaxAnimals) },
                  { key: 'TgBotRanchFeedPrice', value: String(ranchFeedPrice) },
                  { key: 'TgBotRanchWaterPrice', value: String(ranchWaterPrice) },
                  { key: 'TgBotRanchFeedInterval', value: String(ranchFeedInterval) },
                  { key: 'TgBotRanchWaterInterval', value: String(ranchWaterInterval) },
                  { key: 'TgBotRanchHungerDeathHours', value: String(ranchHungerDeathHours) },
                  { key: 'TgBotRanchThirstDeathHours', value: String(ranchThirstDeathHours) },
                  { key: 'TgBotRanchChickenPrice', value: String(ranchChickenPrice) },
                  { key: 'TgBotRanchDuckPrice', value: String(ranchDuckPrice) },
                  { key: 'TgBotRanchGoosePrice', value: String(ranchGoosePrice) },
                  { key: 'TgBotRanchPigPrice', value: String(ranchPigPrice) },
                  { key: 'TgBotRanchSheepPrice', value: String(ranchSheepPrice) },
                  { key: 'TgBotRanchCowPrice', value: String(ranchCowPrice) },
                  { key: 'TgBotRanchChickenMeatPrice', value: String(ranchChickenMeatPrice) },
                  { key: 'TgBotRanchDuckMeatPrice', value: String(ranchDuckMeatPrice) },
                  { key: 'TgBotRanchGooseMeatPrice', value: String(ranchGooseMeatPrice) },
                  { key: 'TgBotRanchPigMeatPrice', value: String(ranchPigMeatPrice) },
                  { key: 'TgBotRanchSheepMeatPrice', value: String(ranchSheepMeatPrice) },
                  { key: 'TgBotRanchCowMeatPrice', value: String(ranchCowMeatPrice) },
                  { key: 'TgBotRanchManureInterval', value: String(ranchManureInterval) },
                  { key: 'TgBotRanchManureCleanPrice', value: String(ranchManureCleanPrice) },
                  { key: 'TgBotRanchManureGrowPenalty', value: String(ranchManureGrowPenalty) },
                  // 等级系统
                  { key: 'TgBotFarmUnlockSteal', value: String(farmUnlockSteal) },
                  { key: 'TgBotFarmUnlockDog', value: String(farmUnlockDog) },
                  { key: 'TgBotFarmUnlockRanch', value: String(farmUnlockRanch) },
                  { key: 'TgBotFarmUnlockFish', value: String(farmUnlockFish) },
                  { key: 'TgBotFarmUnlockWorkshop', value: String(farmUnlockWorkshop) },
                  { key: 'TgBotFarmUnlockMarket', value: String(farmUnlockMarket) },
                  { key: 'TgBotFarmUnlockTasks', value: String(farmUnlockTasks) },
                  { key: 'TgBotFarmUnlockAchieve', value: String(farmUnlockAchieve) },
                  { key: 'TgBotFarmLevelPrices', value: farmLevelPrices },
                  // 银行贷款
                  { key: 'TgBotFarmBankAdminId', value: String(farmBankAdminId) },
                  { key: 'TgBotFarmBankInterestRate', value: String(farmBankInterestRate) },
                  { key: 'TgBotFarmBankMaxLoanDays', value: String(farmBankMaxLoanDays) },
                  { key: 'TgBotFarmBankBaseAmount', value: String(farmBankBaseAmount) },
                  { key: 'TgBotFarmBankMaxMultiplier', value: String(farmBankMaxMultiplier) },
                  { key: 'TgBotFarmBankUnlockLevel', value: String(farmBankUnlockLevel) },
                  { key: 'TgBotFarmMortgageMaxAmount', value: String(farmMortgageMaxAmount) },
                  { key: 'TgBotFarmMortgageInterestRate', value: String(farmMortgageInterestRate) },
                  // 季节系统
                  { key: 'TgBotFarmSeasonDays', value: String(farmSeasonDays) },
                  { key: 'TgBotFarmSeasonInBonus', value: String(farmSeasonInBonus) },
                  { key: 'TgBotFarmSeasonOffBonus', value: String(farmSeasonOffBonus) },
                  { key: 'TgBotFarmWarehouseMaxSlots', value: String(farmWarehouseMaxSlots) },
                ];
                for (const opt of farmOptions) {
                  const res = await API.put('/api/option/', opt);
                  if (!res.data.success) {
                    showError(res.data.message);
                    setSavingFarm(false);
                    return;
                  }
                }
                showSuccess(t('农场设置保存成功'));
                await loadSettings();
              } catch (err) {
                showError(t('保存失败'));
              } finally {
                setSavingFarm(false);
              }
            }}
            style={{ marginTop: 8 }}
          >
            {t('保存农场设置')}
          </Button>
        </div>
      </Card>

      {/* ===== 农场公告设置 ===== */}
      <Card
        title={<span>📢 {t('农场全站公告')}</span>}
        className='mt-4'
      >
        <div className='mb-4'>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
            <Typography.Text style={{ width: 200 }}>{t('启用公告')}</Typography.Text>
            <Switch checked={farmAnnEnabled} onChange={setFarmAnnEnabled} />
            <Typography.Text type='tertiary' style={{ marginLeft: 12 }}>
              {farmAnnEnabled ? t('已开启') : t('已关闭')}
            </Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
            <Typography.Text style={{ width: 200 }}>{t('公告类型')}</Typography.Text>
            <Form.Select
              value={farmAnnType}
              onChange={setFarmAnnType}
              style={{ width: 200 }}
              optionList={[
                { value: 'info', label: '📢 ' + t('普通通知') + ' (蓝色)' },
                { value: 'urgent', label: '🚨 ' + t('紧急维护') + ' (红色)' },
                { value: 'event', label: '🎉 ' + t('活动喜报') + ' (金色)' },
              ]}
            />
          </div>
          <div style={{ marginBottom: 16 }}>
            <Typography.Text style={{ display: 'block', marginBottom: 8 }}>{t('公告内容')}</Typography.Text>
            <TextArea
              value={farmAnnText}
              onChange={setFarmAnnText}
              placeholder={t('输入公告文字，长文本将自动跑马灯滚动')}
              rows={2}
              maxCount={200}
              style={{ width: '100%' }}
            />
          </div>
          {farmAnnText && (
            <div style={{
              marginBottom: 16, padding: '10px 16px', borderRadius: 9999,
              background: 'rgba(15,10,30,0.82)', border: '1.5px solid ' +
                (farmAnnType === 'urgent' ? 'rgba(239,68,68,0.7)' : farmAnnType === 'event' ? 'rgba(234,179,8,0.65)' : 'rgba(99,102,241,0.6)'),
              color: '#f1f5f9', fontSize: 13, display: 'flex', alignItems: 'center', gap: 10,
            }}>
              <span>{farmAnnType === 'urgent' ? '🚨' : farmAnnType === 'event' ? '🎉' : '📢'}</span>
              <span style={{
                padding: '2px 8px', borderRadius: 9999, fontSize: 11, fontWeight: 700,
                background: farmAnnType === 'urgent' ? 'rgba(239,68,68,0.2)' : farmAnnType === 'event' ? 'rgba(234,179,8,0.18)' : 'rgba(99,102,241,0.25)',
                color: farmAnnType === 'urgent' ? '#fca5a5' : farmAnnType === 'event' ? '#fde68a' : '#a5b4fc',
              }}>
                {farmAnnType === 'urgent' ? t('维护') : farmAnnType === 'event' ? t('活动') : t('通知')}
              </span>
              <span>{farmAnnText}</span>
            </div>
          )}
          <Button
            theme='solid'
            type='primary'
            loading={savingFarmAnn}
            onClick={async () => {
              setSavingFarmAnn(true);
              try {
                const opts = [
                  { key: 'FarmAnnouncementEnabled', value: farmAnnEnabled ? 'true' : 'false' },
                  { key: 'FarmAnnouncementText', value: farmAnnText },
                  { key: 'FarmAnnouncementType', value: farmAnnType },
                ];
                for (const opt of opts) {
                  const res = await API.put('/api/option/', opt);
                  if (!res.data.success) { showError(res.data.message); return; }
                }
                showSuccess(t('公告设置已保存'));
                await loadSettings();
              } catch (err) {
                showError(t('保存失败'));
              } finally {
                setSavingFarmAnn(false);
              }
            }}
          >
            {t('保存公告设置')}
          </Button>
        </div>
      </Card>

      {/* ===== 农场活跃用户 ===== */}
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
            <span>🌾 {t('农场活跃用户')} ({farmUsers.length})</span>
            <Button size='small' theme='light' onClick={loadFarmUsers} loading={farmUsersLoading}>{t('刷新')}</Button>
          </div>
        }
        className='mt-4'
      >
        <Table
          dataSource={farmUsers}
          loading={farmUsersLoading}
          pagination={{ pageSize: 20 }}
          size='small'
          empty={t('暂无活跃农场用户')}
          columns={[
            {
              title: t('用户'),
              dataIndex: 'username',
              render: (text, record) => (
                <span>
                  {record.display_name || record.username || record.farm_id}
                  {record.user_id > 0 && <Tag size='small' color='blue' style={{ marginLeft: 6 }}>ID:{record.user_id}</Tag>}
                </span>
              ),
            },
            { title: t('农场ID'), dataIndex: 'farm_id', width: 120 },
            { title: t('等级'), dataIndex: 'farm_level', width: 70, render: v => `Lv.${v}`, sorter: (a, b) => a.farm_level - b.farm_level, defaultSortOrder: 'descend' },
            { title: t('总地块'), dataIndex: 'total_plots', width: 80 },
            { title: t('种植中'), dataIndex: 'active_plots', width: 80, render: v => <span style={{ color: 'var(--semi-color-success)' }}>{v}</span> },
            { title: t('已成熟'), dataIndex: 'mature_plots', width: 80, render: v => v > 0 ? <span style={{ color: 'var(--semi-color-warning)' }}>{v}</span> : '0' },
            { title: t('余额'), dataIndex: 'balance', width: 100, render: v => `$${v?.toFixed(2) || '0.00'}`, sorter: (a, b) => a.balance - b.balance },
          ]}
          rowKey='farm_id'
        />
      </Card>

      {/* ===== 管理操作 ===== */}
      <Card
        title={t('农场管理操作')}
        className='mt-4'
      >
        <div className='mb-4'>
          <Banner type='warning' description={t('以下操作不可撤销，请谨慎使用')} style={{ marginBottom: 16, borderRadius: 8 }} />

          <div style={{ borderBottom: '1px solid var(--semi-color-border)', paddingBottom: 16, marginBottom: 16 }}>
            <Typography.Title heading={6} style={{ marginBottom: 8 }}>💰 {t('重置负余额用户')}</Typography.Title>
            <Typography.Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 12 }}>
              {t('将所有余额为负数的用户重置为0，不影响正常用户')}
            </Typography.Text>
            <Button
              theme='solid'
              type='danger'
              onClick={async () => {
                Modal.confirm({
                  title: t('确认重置'),
                  content: t('确定要将所有负余额用户重置为0吗？此操作不可撤销。'),
                  onOk: async () => {
                    try {
                      const { data: res } = await API.post('/api/tgbot/farm/reset-negative-balances');
                      if (res.success) {
                        showSuccess(res.message);
                      } else {
                        showError(res.message);
                      }
                    } catch (err) {
                      showError(t('操作失败'));
                    }
                  },
                });
              }}
            >
              {t('重置所有负余额为0')}
            </Button>
          </div>

          <div>
            <Typography.Title heading={6} style={{ marginBottom: 8 }}>⭐ {t('重置所有用户等级')}</Typography.Title>
            <Typography.Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 12 }}>
              {t('将所有用户的农场等级重置到指定等级')}
            </Typography.Text>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <InputNumber
                value={resetLevel}
                onChange={setResetLevel}
                min={1}
                max={15}
                style={{ width: 120 }}
                placeholder={t('目标等级')}
              />
              <Button
                theme='solid'
                type='danger'
                onClick={async () => {
                  const level = resetLevel;
                  if (!level || level < 1 || level > 15) {
                    showError(t('请输入有效等级 (1-15)'));
                    return;
                  }
                  Modal.confirm({
                    title: t('确认重置'),
                    content: t('确定要将所有用户等级重置为') + ` Lv.${level} ` + t('吗？此操作不可撤销。'),
                    onOk: async () => {
                      try {
                        const { data: res } = await API.post('/api/tgbot/farm/reset-all-levels', { level });
                        if (res.success) {
                          showSuccess(res.message);
                        } else {
                          showError(res.message);
                        }
                      } catch (err) {
                        showError(t('操作失败'));
                      }
                    },
                  });
                }}
              >
                {t('重置所有用户等级')}
              </Button>
            </div>
          </div>
        </div>
      </Card>

      {/* ===== 抽奖设置 ===== */}
      <Card
        title={
          <div className='flex items-center justify-between'>
            <span>{t('抽奖管理')}</span>
            <Tag color={lotteryEnabled ? 'green' : 'grey'}>
              {lotteryEnabled ? t('已开启') : t('已关闭')}
            </Tag>
          </div>
        }
        className='mt-4'
      >
        <div className='mb-4'>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 160 }}>{t('开启群聊抽奖')}</Typography.Text>
            <Switch checked={lotteryEnabled} onChange={setLotteryEnabled} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 160 }}>{t('每多少条消息抽一次')}</Typography.Text>
            <InputNumber value={lotteryMessagesRequired} onChange={setLotteryMessagesRequired} min={1} max={1000} style={{ width: 120 }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
            <Typography.Text style={{ width: 160 }}>{t('中奖概率(%)')}</Typography.Text>
            <InputNumber value={lotteryWinRate} onChange={setLotteryWinRate} min={0} max={100} style={{ width: 120 }} />
          </div>
          <Button
            theme='solid'
            type='primary'
            loading={savingLottery}
            onClick={saveLotterySettings}
            style={{ marginTop: 8 }}
          >
            {t('保存抽奖设置')}
          </Button>
        </div>

        <div className='mb-4' style={{ borderTop: '1px solid var(--semi-color-border)', paddingTop: 16 }}>
          <Typography.Text strong className='mb-2 block'>
            {t('添加奖品')}
          </Typography.Text>
          <Input
            placeholder={t('奖品名称，如：VIP会员码、新手礼包')}
            value={addPrizeName}
            onChange={setAddPrizeName}
            style={{ marginBottom: 8 }}
          />
          <TextArea
            value={addPrizeCodes}
            onChange={setAddPrizeCodes}
            placeholder={t('每行一个兑换码，支持批量添加')}
            rows={3}
            style={{ marginBottom: 8 }}
          />
          <Button
            theme='solid'
            type='primary'
            loading={addingPrizes}
            onClick={handleAddPrizes}
            disabled={!addPrizeName.trim() || !addPrizeCodes.trim()}
          >
            {t('批量添加奖品')}
          </Button>
        </div>

        <div className='mb-2'>
          <Typography.Text strong>
            {t('奖品库存')}
            <Tag color='blue' style={{ marginLeft: 8 }}>
              {t('可用')} {lotteryPrizeAvailable} / {t('总数')} {lotteryPrizeTotal}
            </Tag>
          </Typography.Text>
        </div>
        <Table
          columns={lotteryPrizeColumns}
          dataSource={lotteryPrizes}
          loading={lotteryPrizesLoading}
          rowKey='id'
          pagination={{ pageSize: 10 }}
          size='small'
          scroll={{ y: 300 }}
          empty={
            <div className='py-4 text-center text-gray-400'>
              {t('暂无奖品，请添加')}
            </div>
          }
        />
      </Card>

      {/* ===== 添加/编辑分类弹窗 ===== */}
      <Modal
        title={editingCategory ? t('编辑分类') : t('添加分类')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        centered
        size='small'
      >
        <Form
          onSubmit={handleSubmit}
          initValues={
            editingCategory || {
              name: '',
              description: '',
              max_claims: 1,
              purpose: 1,
              status: 1,
            }
          }
          labelPosition='top'
        >
          <Form.Input
            field='name'
            label={t('分类名称')}
            placeholder={t('如：新手福利、每日签到奖励')}
            rules={[{ required: true, message: t('请输入分类名称') }]}
          />
          <Form.Input
            field='description'
            label={t('描述')}
            placeholder={t('可选，分类的简要描述')}
          />
          <Form.Select
            field='purpose'
            label={t('兑换码类型')}
            optionList={PURPOSE_OPTIONS.map((o) => ({
              ...o,
              label: t(o.label),
            }))}
            rules={[{ required: true, message: t('请选择兑换码类型') }]}
          />
          <Form.InputNumber
            field='max_claims'
            label={t('每人可领取次数')}
            min={1}
            max={9999}
            rules={[{ required: true, message: t('请输入领取次数') }]}
          />
          <Form.Select
            field='status'
            label={t('状态')}
            optionList={STATUS_OPTIONS.map((o) => ({
              ...o,
              label: t(o.label),
            }))}
          />
          <div className='flex justify-end gap-2 mt-4'>
            <Button onClick={() => setModalVisible(false)}>{t('取消')}</Button>
            <Button
              theme='solid'
              type='primary'
              htmlType='submit'
              loading={submitting}
            >
              {editingCategory ? t('更新') : t('创建')}
            </Button>
          </div>
        </Form>
      </Modal>

      {/* ===== 库存管理弹窗 ===== */}
      <Modal
        title={
          inventoryCategory
            ? `${t('库存管理')} - ${inventoryCategory.name}（${inventoryCategory.purpose === 2 ? t('邀请码') : t('兑换码')}）`
            : t('库存管理')
        }
        visible={inventoryModalVisible}
        onCancel={() => setInventoryModalVisible(false)}
        footer={null}
        centered
        width={700}
      >
        {/* 添加兑换码/邀请码 */}
        <div className='mb-4'>
          <Typography.Text strong className='mb-2 block'>
            {inventoryCategory?.purpose === 2
              ? t('添加邀请码')
              : t('添加兑换码')}
          </Typography.Text>
          <TextArea
            value={addCodesText}
            onChange={setAddCodesText}
            placeholder={t('每行一个兑换码/邀请码，支持批量添加')}
            rows={4}
            style={{ marginBottom: 8 }}
          />
          <Button
            theme='solid'
            type='primary'
            loading={addingCodes}
            onClick={handleAddCodes}
            disabled={!addCodesText.trim()}
          >
            {t('批量添加')}
          </Button>
        </div>

        {/* 库存列表 */}
        <div className='mb-2'>
          <Typography.Text strong>
            {t('当前库存')}
            {inventoryItems.length > 0 && (
              <Tag color='blue' style={{ marginLeft: 8 }}>
                {t('可用')} {inventoryItems.filter((i) => i.status === 1).length} / {t('总数')} {inventoryItems.length}
              </Tag>
            )}
          </Typography.Text>
        </div>
        <Table
          columns={inventoryColumns}
          dataSource={inventoryItems}
          loading={inventoryLoading}
          rowKey='id'
          pagination={{ pageSize: 10 }}
          size='small'
          scroll={{ y: 300 }}
          empty={
            <div className='py-4 text-center text-gray-400'>
              {t('暂无库存，请添加')}
            </div>
          }
        />
      </Modal>
    </div>
  );
};

export default TgBotPage;
