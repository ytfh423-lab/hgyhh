package model

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/performance_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

type Option struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}

func AllOption() ([]*Option, error) {
	var options []*Option
	var err error
	err = DB.Find(&options).Error
	return options, err
}

func InitOptionMap() {
	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)

	// 添加原有的系统配置
	common.OptionMap["FileUploadPermission"] = strconv.Itoa(common.FileUploadPermission)
	common.OptionMap["FileDownloadPermission"] = strconv.Itoa(common.FileDownloadPermission)
	common.OptionMap["ImageUploadPermission"] = strconv.Itoa(common.ImageUploadPermission)
	common.OptionMap["ImageDownloadPermission"] = strconv.Itoa(common.ImageDownloadPermission)
	common.OptionMap["PasswordLoginEnabled"] = strconv.FormatBool(common.PasswordLoginEnabled)
	common.OptionMap["PasswordRegisterEnabled"] = strconv.FormatBool(common.PasswordRegisterEnabled)
	common.OptionMap["EmailVerificationEnabled"] = strconv.FormatBool(common.EmailVerificationEnabled)
	common.OptionMap["GitHubOAuthEnabled"] = strconv.FormatBool(common.GitHubOAuthEnabled)
	common.OptionMap["LinuxDOOAuthEnabled"] = strconv.FormatBool(common.LinuxDOOAuthEnabled)
	common.OptionMap["TelegramOAuthEnabled"] = strconv.FormatBool(common.TelegramOAuthEnabled)
	common.OptionMap["WeChatAuthEnabled"] = strconv.FormatBool(common.WeChatAuthEnabled)
	common.OptionMap["TurnstileCheckEnabled"] = strconv.FormatBool(common.TurnstileCheckEnabled)
	common.OptionMap["RegisterEnabled"] = strconv.FormatBool(common.RegisterEnabled)
	common.OptionMap["AutomaticDisableChannelEnabled"] = strconv.FormatBool(common.AutomaticDisableChannelEnabled)
	common.OptionMap["AutomaticEnableChannelEnabled"] = strconv.FormatBool(common.AutomaticEnableChannelEnabled)
	common.OptionMap["LogConsumeEnabled"] = strconv.FormatBool(common.LogConsumeEnabled)
	common.OptionMap["DisplayInCurrencyEnabled"] = strconv.FormatBool(common.DisplayInCurrencyEnabled)
	common.OptionMap["DisplayTokenStatEnabled"] = strconv.FormatBool(common.DisplayTokenStatEnabled)
	common.OptionMap["DrawingEnabled"] = strconv.FormatBool(common.DrawingEnabled)
	common.OptionMap["TaskEnabled"] = strconv.FormatBool(common.TaskEnabled)
	common.OptionMap["DataExportEnabled"] = strconv.FormatBool(common.DataExportEnabled)
	common.OptionMap["ChannelDisableThreshold"] = strconv.FormatFloat(common.ChannelDisableThreshold, 'f', -1, 64)
	common.OptionMap["EmailDomainRestrictionEnabled"] = strconv.FormatBool(common.EmailDomainRestrictionEnabled)
	common.OptionMap["EmailAliasRestrictionEnabled"] = strconv.FormatBool(common.EmailAliasRestrictionEnabled)
	common.OptionMap["EmailDomainWhitelist"] = strings.Join(common.EmailDomainWhitelist, ",")
	common.OptionMap["SMTPServer"] = ""
	common.OptionMap["SMTPFrom"] = ""
	common.OptionMap["SMTPPort"] = strconv.Itoa(common.SMTPPort)
	common.OptionMap["SMTPAccount"] = ""
	common.OptionMap["SMTPToken"] = ""
	common.OptionMap["SMTPSSLEnabled"] = strconv.FormatBool(common.SMTPSSLEnabled)
	common.OptionMap["Notice"] = ""
	common.OptionMap["About"] = ""
	common.OptionMap["HomePageContent"] = ""
	common.OptionMap["Footer"] = common.Footer
	common.OptionMap["SystemName"] = common.SystemName
	common.OptionMap["Logo"] = common.Logo
	common.OptionMap["ServerAddress"] = ""
	common.OptionMap["WorkerUrl"] = system_setting.WorkerUrl
	common.OptionMap["WorkerValidKey"] = system_setting.WorkerValidKey
	common.OptionMap["WorkerAllowHttpImageRequestEnabled"] = strconv.FormatBool(system_setting.WorkerAllowHttpImageRequestEnabled)
	common.OptionMap["PayAddress"] = ""
	common.OptionMap["CustomCallbackAddress"] = ""
	common.OptionMap["EpayId"] = ""
	common.OptionMap["EpayKey"] = ""
	common.OptionMap["Price"] = strconv.FormatFloat(operation_setting.Price, 'f', -1, 64)
	common.OptionMap["USDExchangeRate"] = strconv.FormatFloat(operation_setting.USDExchangeRate, 'f', -1, 64)
	common.OptionMap["MinTopUp"] = strconv.Itoa(operation_setting.MinTopUp)
	common.OptionMap["StripeMinTopUp"] = strconv.Itoa(setting.StripeMinTopUp)
	common.OptionMap["StripeApiSecret"] = setting.StripeApiSecret
	common.OptionMap["StripeWebhookSecret"] = setting.StripeWebhookSecret
	common.OptionMap["StripePriceId"] = setting.StripePriceId
	common.OptionMap["StripeUnitPrice"] = strconv.FormatFloat(setting.StripeUnitPrice, 'f', -1, 64)
	common.OptionMap["StripePromotionCodesEnabled"] = strconv.FormatBool(setting.StripePromotionCodesEnabled)
	common.OptionMap["CreemApiKey"] = setting.CreemApiKey
	common.OptionMap["CreemProducts"] = setting.CreemProducts
	common.OptionMap["CreemTestMode"] = strconv.FormatBool(setting.CreemTestMode)
	common.OptionMap["CreemWebhookSecret"] = setting.CreemWebhookSecret
	common.OptionMap["TopupGroupRatio"] = common.TopupGroupRatio2JSONString()
	common.OptionMap["Chats"] = setting.Chats2JsonString()
	common.OptionMap["AutoGroups"] = setting.AutoGroups2JsonString()
	common.OptionMap["DefaultUseAutoGroup"] = strconv.FormatBool(setting.DefaultUseAutoGroup)
	common.OptionMap["PayMethods"] = operation_setting.PayMethods2JsonString()
	common.OptionMap["GitHubClientId"] = ""
	common.OptionMap["GitHubClientSecret"] = ""
	common.OptionMap["TelegramBotToken"] = ""
	common.OptionMap["TelegramBotName"] = ""
	common.OptionMap["TgBotLotteryEnabled"] = "false"
	common.OptionMap["TgBotLotteryMessagesRequired"] = "10"
	common.OptionMap["TgBotLotteryWinRate"] = "30"
	common.OptionMap["TgBotFarmPlotPrice"] = strconv.Itoa(common.TgBotFarmPlotPrice)
	common.OptionMap["TgBotFarmDogPrice"] = strconv.Itoa(common.TgBotFarmDogPrice)
	common.OptionMap["TgBotFarmDogFoodPrice"] = strconv.Itoa(common.TgBotFarmDogFoodPrice)
	common.OptionMap["TgBotFarmDogGrowHours"] = strconv.Itoa(common.TgBotFarmDogGrowHours)
	common.OptionMap["TgBotFarmDogGuardRate"] = strconv.Itoa(common.TgBotFarmDogGuardRate)
	common.OptionMap["TgBotFarmWaterInterval"] = strconv.Itoa(common.TgBotFarmWaterInterval)
	common.OptionMap["TgBotFarmWiltDuration"] = strconv.Itoa(common.TgBotFarmWiltDuration)
	common.OptionMap["TgBotFarmEventChance"] = strconv.Itoa(common.TgBotFarmEventChance)
	common.OptionMap["TgBotFarmDisasterChance"] = strconv.Itoa(common.TgBotFarmDisasterChance)
	common.OptionMap["TgBotFarmStealCooldown"] = strconv.Itoa(common.TgBotFarmStealCooldown)
	common.OptionMap["TgBotFarmSoilMaxLevel"] = strconv.Itoa(common.TgBotFarmSoilMaxLevel)
	common.OptionMap["TgBotFarmSoilUpgradePrice2"] = strconv.Itoa(common.TgBotFarmSoilUpgradePrice2)
	common.OptionMap["TgBotFarmSoilUpgradePrice3"] = strconv.Itoa(common.TgBotFarmSoilUpgradePrice3)
	common.OptionMap["TgBotFarmSoilUpgradePrice4"] = strconv.Itoa(common.TgBotFarmSoilUpgradePrice4)
	common.OptionMap["TgBotFarmSoilUpgradePrice5"] = strconv.Itoa(common.TgBotFarmSoilUpgradePrice5)
	common.OptionMap["TgBotFarmSoilSpeedBonus"] = strconv.Itoa(common.TgBotFarmSoilSpeedBonus)
	// 牧场
	common.OptionMap["TgBotRanchMaxAnimals"] = strconv.Itoa(common.TgBotRanchMaxAnimals)
	common.OptionMap["TgBotRanchFeedPrice"] = strconv.Itoa(common.TgBotRanchFeedPrice)
	common.OptionMap["TgBotRanchWaterPrice"] = strconv.Itoa(common.TgBotRanchWaterPrice)
	common.OptionMap["TgBotRanchFeedInterval"] = strconv.Itoa(common.TgBotRanchFeedInterval)
	common.OptionMap["TgBotRanchWaterInterval"] = strconv.Itoa(common.TgBotRanchWaterInterval)
	common.OptionMap["TgBotRanchHungerDeathHours"] = strconv.Itoa(common.TgBotRanchHungerDeathHours)
	common.OptionMap["TgBotRanchThirstDeathHours"] = strconv.Itoa(common.TgBotRanchThirstDeathHours)
	common.OptionMap["TgBotRanchChickenPrice"] = strconv.Itoa(common.TgBotRanchChickenPrice)
	common.OptionMap["TgBotRanchDuckPrice"] = strconv.Itoa(common.TgBotRanchDuckPrice)
	common.OptionMap["TgBotRanchGoosePrice"] = strconv.Itoa(common.TgBotRanchGoosePrice)
	common.OptionMap["TgBotRanchPigPrice"] = strconv.Itoa(common.TgBotRanchPigPrice)
	common.OptionMap["TgBotRanchSheepPrice"] = strconv.Itoa(common.TgBotRanchSheepPrice)
	common.OptionMap["TgBotRanchCowPrice"] = strconv.Itoa(common.TgBotRanchCowPrice)
	common.OptionMap["TgBotRanchChickenGrowSecs"] = strconv.FormatInt(common.TgBotRanchChickenGrowSecs, 10)
	common.OptionMap["TgBotRanchDuckGrowSecs"] = strconv.FormatInt(common.TgBotRanchDuckGrowSecs, 10)
	common.OptionMap["TgBotRanchGooseGrowSecs"] = strconv.FormatInt(common.TgBotRanchGooseGrowSecs, 10)
	common.OptionMap["TgBotRanchPigGrowSecs"] = strconv.FormatInt(common.TgBotRanchPigGrowSecs, 10)
	common.OptionMap["TgBotRanchSheepGrowSecs"] = strconv.FormatInt(common.TgBotRanchSheepGrowSecs, 10)
	common.OptionMap["TgBotRanchCowGrowSecs"] = strconv.FormatInt(common.TgBotRanchCowGrowSecs, 10)
	common.OptionMap["TgBotRanchChickenMeatPrice"] = strconv.Itoa(common.TgBotRanchChickenMeatPrice)
	common.OptionMap["TgBotRanchDuckMeatPrice"] = strconv.Itoa(common.TgBotRanchDuckMeatPrice)
	common.OptionMap["TgBotRanchGooseMeatPrice"] = strconv.Itoa(common.TgBotRanchGooseMeatPrice)
	common.OptionMap["TgBotRanchPigMeatPrice"] = strconv.Itoa(common.TgBotRanchPigMeatPrice)
	common.OptionMap["TgBotRanchSheepMeatPrice"] = strconv.Itoa(common.TgBotRanchSheepMeatPrice)
	common.OptionMap["TgBotRanchCowMeatPrice"] = strconv.Itoa(common.TgBotRanchCowMeatPrice)
	common.OptionMap["TgBotRanchManureInterval"] = strconv.Itoa(common.TgBotRanchManureInterval)
	common.OptionMap["TgBotRanchManureCleanPrice"] = strconv.Itoa(common.TgBotRanchManureCleanPrice)
	common.OptionMap["TgBotRanchManureGrowPenalty"] = strconv.Itoa(common.TgBotRanchManureGrowPenalty)
	common.OptionMap["WeChatServerAddress"] = ""
	common.OptionMap["WeChatServerToken"] = ""
	common.OptionMap["WeChatAccountQRCodeImageURL"] = ""
	common.OptionMap["TurnstileSiteKey"] = ""
	common.OptionMap["TurnstileSecretKey"] = ""
	common.OptionMap["QuotaForNewUser"] = strconv.Itoa(common.QuotaForNewUser)
	common.OptionMap["QuotaForInviter"] = strconv.Itoa(common.QuotaForInviter)
	common.OptionMap["QuotaForInvitee"] = strconv.Itoa(common.QuotaForInvitee)
	common.OptionMap["QuotaRemindThreshold"] = strconv.Itoa(common.QuotaRemindThreshold)
	common.OptionMap["PreConsumedQuota"] = strconv.Itoa(common.PreConsumedQuota)
	common.OptionMap["ModelRequestRateLimitCount"] = strconv.Itoa(setting.ModelRequestRateLimitCount)
	common.OptionMap["ModelRequestRateLimitDurationMinutes"] = strconv.Itoa(setting.ModelRequestRateLimitDurationMinutes)
	common.OptionMap["ModelRequestRateLimitSuccessCount"] = strconv.Itoa(setting.ModelRequestRateLimitSuccessCount)
	common.OptionMap["ModelRequestRateLimitGroup"] = setting.ModelRequestRateLimitGroup2JSONString()
	common.OptionMap["ModelRatio"] = ratio_setting.ModelRatio2JSONString()
	common.OptionMap["ModelPrice"] = ratio_setting.ModelPrice2JSONString()
	common.OptionMap["CacheRatio"] = ratio_setting.CacheRatio2JSONString()
	common.OptionMap["CreateCacheRatio"] = ratio_setting.CreateCacheRatio2JSONString()
	common.OptionMap["GroupRatio"] = ratio_setting.GroupRatio2JSONString()
	common.OptionMap["GroupGroupRatio"] = ratio_setting.GroupGroupRatio2JSONString()
	common.OptionMap["UserUsableGroups"] = setting.UserUsableGroups2JSONString()
	common.OptionMap["CompletionRatio"] = ratio_setting.CompletionRatio2JSONString()
	common.OptionMap["ImageRatio"] = ratio_setting.ImageRatio2JSONString()
	common.OptionMap["AudioRatio"] = ratio_setting.AudioRatio2JSONString()
	common.OptionMap["AudioCompletionRatio"] = ratio_setting.AudioCompletionRatio2JSONString()
	common.OptionMap["TopUpLink"] = common.TopUpLink
	//common.OptionMap["ChatLink"] = common.ChatLink
	//common.OptionMap["ChatLink2"] = common.ChatLink2
	common.OptionMap["QuotaPerUnit"] = strconv.FormatFloat(common.QuotaPerUnit, 'f', -1, 64)
	common.OptionMap["RetryTimes"] = strconv.Itoa(common.RetryTimes)
	common.OptionMap["DataExportInterval"] = strconv.Itoa(common.DataExportInterval)
	common.OptionMap["DataExportDefaultTime"] = common.DataExportDefaultTime
	common.OptionMap["DefaultCollapseSidebar"] = strconv.FormatBool(common.DefaultCollapseSidebar)
	common.OptionMap["MjNotifyEnabled"] = strconv.FormatBool(setting.MjNotifyEnabled)
	common.OptionMap["MjAccountFilterEnabled"] = strconv.FormatBool(setting.MjAccountFilterEnabled)
	common.OptionMap["MjModeClearEnabled"] = strconv.FormatBool(setting.MjModeClearEnabled)
	common.OptionMap["MjForwardUrlEnabled"] = strconv.FormatBool(setting.MjForwardUrlEnabled)
	common.OptionMap["MjActionCheckSuccessEnabled"] = strconv.FormatBool(setting.MjActionCheckSuccessEnabled)
	common.OptionMap["CheckSensitiveEnabled"] = strconv.FormatBool(setting.CheckSensitiveEnabled)
	common.OptionMap["DemoSiteEnabled"] = strconv.FormatBool(operation_setting.DemoSiteEnabled)
	common.OptionMap["SelfUseModeEnabled"] = strconv.FormatBool(operation_setting.SelfUseModeEnabled)
	common.OptionMap["ModelRequestRateLimitEnabled"] = strconv.FormatBool(setting.ModelRequestRateLimitEnabled)
	common.OptionMap["RequestRiskControlEnabled"] = strconv.FormatBool(setting.RequestRiskControlEnabled)
	common.OptionMap["RequestRiskControlBurstLimit"] = strconv.Itoa(setting.RequestRiskControlBurstLimit)
	common.OptionMap["RequestRiskControlBurstWindow"] = strconv.Itoa(setting.RequestRiskControlBurstWindow)
	common.OptionMap["RequestRiskControlTokenThreshold"] = strconv.Itoa(setting.RequestRiskControlTokenThreshold)
	common.OptionMap["RequestRiskControlTokenWindow"] = strconv.Itoa(setting.RequestRiskControlTokenWindow)
	common.OptionMap["CheckSensitiveOnPromptEnabled"] = strconv.FormatBool(setting.CheckSensitiveOnPromptEnabled)
	common.OptionMap["StopOnSensitiveEnabled"] = strconv.FormatBool(setting.StopOnSensitiveEnabled)
	common.OptionMap["SensitiveWords"] = setting.SensitiveWordsToString()
	common.OptionMap["StreamCacheQueueLength"] = strconv.Itoa(setting.StreamCacheQueueLength)
	common.OptionMap["AutomaticDisableKeywords"] = operation_setting.AutomaticDisableKeywordsToString()
	common.OptionMap["AutomaticDisableStatusCodes"] = operation_setting.AutomaticDisableStatusCodesToString()
	common.OptionMap["AutomaticRetryStatusCodes"] = operation_setting.AutomaticRetryStatusCodesToString()
	common.OptionMap["ExposeRatioEnabled"] = strconv.FormatBool(ratio_setting.IsExposeRatioEnabled())

	// 自动添加所有注册的模型配置
	modelConfigs := config.GlobalConfig.ExportAllConfigs()
	for k, v := range modelConfigs {
		common.OptionMap[k] = v
	}

	common.OptionMapRWMutex.Unlock()
	loadOptionsFromDatabase()
}

func loadOptionsFromDatabase() {
	options, _ := AllOption()
	for _, option := range options {
		err := updateOptionMap(option.Key, option.Value)
		if err != nil {
			common.SysLog("failed to update option map: " + err.Error())
		}
	}
}

func SyncOptions(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		common.SysLog("syncing options from database")
		loadOptionsFromDatabase()
	}
}

func UpdateOption(key string, value string) error {
	// Save to database first
	option := Option{
		Key: key,
	}
	// https://gorm.io/docs/update.html#Save-All-Fields
	DB.FirstOrCreate(&option, Option{Key: key})
	option.Value = value
	// Save is a combination function.
	// If save value does not contain primary key, it will execute Create,
	// otherwise it will execute Update (with all fields).
	DB.Save(&option)
	// Update OptionMap
	return updateOptionMap(key, value)
}

func updateOptionMap(key string, value string) (err error) {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	common.OptionMap[key] = value

	// 检查是否是模型配置 - 使用更规范的方式处理
	if handleConfigUpdate(key, value) {
		return nil // 已由配置系统处理
	}

	// 处理传统配置项...
	if strings.HasSuffix(key, "Permission") {
		intValue, _ := strconv.Atoi(value)
		switch key {
		case "FileUploadPermission":
			common.FileUploadPermission = intValue
		case "FileDownloadPermission":
			common.FileDownloadPermission = intValue
		case "ImageUploadPermission":
			common.ImageUploadPermission = intValue
		case "ImageDownloadPermission":
			common.ImageDownloadPermission = intValue
		}
	}
	if strings.HasSuffix(key, "Enabled") || key == "DefaultCollapseSidebar" || key == "DefaultUseAutoGroup" {
		boolValue := value == "true"
		switch key {
		case "PasswordRegisterEnabled":
			common.PasswordRegisterEnabled = boolValue
		case "PasswordLoginEnabled":
			common.PasswordLoginEnabled = boolValue
		case "EmailVerificationEnabled":
			common.EmailVerificationEnabled = boolValue
		case "GitHubOAuthEnabled":
			common.GitHubOAuthEnabled = boolValue
		case "LinuxDOOAuthEnabled":
			common.LinuxDOOAuthEnabled = boolValue
		case "WeChatAuthEnabled":
			common.WeChatAuthEnabled = boolValue
		case "TelegramOAuthEnabled":
			common.TelegramOAuthEnabled = boolValue
		case "TurnstileCheckEnabled":
			common.TurnstileCheckEnabled = boolValue
		case "RegisterEnabled":
			common.RegisterEnabled = boolValue
		case "EmailDomainRestrictionEnabled":
			common.EmailDomainRestrictionEnabled = boolValue
		case "EmailAliasRestrictionEnabled":
			common.EmailAliasRestrictionEnabled = boolValue
		case "AutomaticDisableChannelEnabled":
			common.AutomaticDisableChannelEnabled = boolValue
		case "AutomaticEnableChannelEnabled":
			common.AutomaticEnableChannelEnabled = boolValue
		case "LogConsumeEnabled":
			common.LogConsumeEnabled = boolValue
		case "DisplayInCurrencyEnabled":
			// 兼容旧字段：同步到新配置 general_setting.quota_display_type（运行时生效）
			// true -> USD, false -> TOKENS
			newVal := "USD"
			if !boolValue {
				newVal = "TOKENS"
			}
			if cfg := config.GlobalConfig.Get("general_setting"); cfg != nil {
				_ = config.UpdateConfigFromMap(cfg, map[string]string{"quota_display_type": newVal})
			}
		case "DisplayTokenStatEnabled":
			common.DisplayTokenStatEnabled = boolValue
		case "DrawingEnabled":
			common.DrawingEnabled = boolValue
		case "TaskEnabled":
			common.TaskEnabled = boolValue
		case "DataExportEnabled":
			common.DataExportEnabled = boolValue
		case "DefaultCollapseSidebar":
			common.DefaultCollapseSidebar = boolValue
		case "MjNotifyEnabled":
			setting.MjNotifyEnabled = boolValue
		case "MjAccountFilterEnabled":
			setting.MjAccountFilterEnabled = boolValue
		case "MjModeClearEnabled":
			setting.MjModeClearEnabled = boolValue
		case "MjForwardUrlEnabled":
			setting.MjForwardUrlEnabled = boolValue
		case "MjActionCheckSuccessEnabled":
			setting.MjActionCheckSuccessEnabled = boolValue
		case "CheckSensitiveEnabled":
			setting.CheckSensitiveEnabled = boolValue
		case "DemoSiteEnabled":
			operation_setting.DemoSiteEnabled = boolValue
		case "SelfUseModeEnabled":
			operation_setting.SelfUseModeEnabled = boolValue
		case "CheckSensitiveOnPromptEnabled":
			setting.CheckSensitiveOnPromptEnabled = boolValue
		case "ModelRequestRateLimitEnabled":
			setting.ModelRequestRateLimitEnabled = boolValue
		case "RequestRiskControlEnabled":
			setting.RequestRiskControlEnabled = boolValue
		case "StopOnSensitiveEnabled":
			setting.StopOnSensitiveEnabled = boolValue
		case "SMTPSSLEnabled":
			common.SMTPSSLEnabled = boolValue
		case "WorkerAllowHttpImageRequestEnabled":
			system_setting.WorkerAllowHttpImageRequestEnabled = boolValue
		case "DefaultUseAutoGroup":
			setting.DefaultUseAutoGroup = boolValue
		case "ExposeRatioEnabled":
			ratio_setting.SetExposeRatioEnabled(boolValue)
		}
	}
	switch key {
	case "EmailDomainWhitelist":
		common.EmailDomainWhitelist = strings.Split(value, ",")
	case "SMTPServer":
		common.SMTPServer = value
	case "SMTPPort":
		intValue, _ := strconv.Atoi(value)
		common.SMTPPort = intValue
	case "SMTPAccount":
		common.SMTPAccount = value
	case "SMTPFrom":
		common.SMTPFrom = value
	case "SMTPToken":
		common.SMTPToken = value
	case "ServerAddress":
		system_setting.ServerAddress = value
	case "WorkerUrl":
		system_setting.WorkerUrl = value
	case "WorkerValidKey":
		system_setting.WorkerValidKey = value
	case "PayAddress":
		operation_setting.PayAddress = value
	case "Chats":
		err = setting.UpdateChatsByJsonString(value)
	case "AutoGroups":
		err = setting.UpdateAutoGroupsByJsonString(value)
	case "CustomCallbackAddress":
		operation_setting.CustomCallbackAddress = value
	case "EpayId":
		operation_setting.EpayId = value
	case "EpayKey":
		operation_setting.EpayKey = value
	case "Price":
		operation_setting.Price, _ = strconv.ParseFloat(value, 64)
	case "USDExchangeRate":
		operation_setting.USDExchangeRate, _ = strconv.ParseFloat(value, 64)
	case "MinTopUp":
		operation_setting.MinTopUp, _ = strconv.Atoi(value)
	case "StripeApiSecret":
		setting.StripeApiSecret = value
	case "StripeWebhookSecret":
		setting.StripeWebhookSecret = value
	case "StripePriceId":
		setting.StripePriceId = value
	case "StripeUnitPrice":
		setting.StripeUnitPrice, _ = strconv.ParseFloat(value, 64)
	case "StripeMinTopUp":
		setting.StripeMinTopUp, _ = strconv.Atoi(value)
	case "StripePromotionCodesEnabled":
		setting.StripePromotionCodesEnabled = value == "true"
	case "CreemApiKey":
		setting.CreemApiKey = value
	case "CreemProducts":
		setting.CreemProducts = value
	case "CreemTestMode":
		setting.CreemTestMode = value == "true"
	case "CreemWebhookSecret":
		setting.CreemWebhookSecret = value
	case "TopupGroupRatio":
		err = common.UpdateTopupGroupRatioByJSONString(value)
	case "GitHubClientId":
		common.GitHubClientId = value
	case "GitHubClientSecret":
		common.GitHubClientSecret = value
	case "LinuxDOClientId":
		common.LinuxDOClientId = value
	case "LinuxDOClientSecret":
		common.LinuxDOClientSecret = value
	case "LinuxDOMinimumTrustLevel":
		common.LinuxDOMinimumTrustLevel, _ = strconv.Atoi(value)
	case "Footer":
		common.Footer = value
	case "SystemName":
		common.SystemName = value
	case "Logo":
		common.Logo = value
	case "WeChatServerAddress":
		common.WeChatServerAddress = value
	case "WeChatServerToken":
		common.WeChatServerToken = value
	case "WeChatAccountQRCodeImageURL":
		common.WeChatAccountQRCodeImageURL = value
	case "TelegramBotToken":
		common.TelegramBotToken = value
	case "TelegramBotName":
		common.TelegramBotName = value
	case "TgBotLotteryEnabled":
		common.TgBotLotteryEnabled = value == "true"
	case "TgBotLotteryMessagesRequired":
		common.TgBotLotteryMessagesRequired, _ = strconv.Atoi(value)
		if common.TgBotLotteryMessagesRequired <= 0 {
			common.TgBotLotteryMessagesRequired = 10
		}
	case "TgBotLotteryWinRate":
		common.TgBotLotteryWinRate, _ = strconv.Atoi(value)
		if common.TgBotLotteryWinRate < 0 {
			common.TgBotLotteryWinRate = 0
		}
		if common.TgBotLotteryWinRate > 100 {
			common.TgBotLotteryWinRate = 100
		}
	case "TgBotFarmPlotPrice":
		common.TgBotFarmPlotPrice, _ = strconv.Atoi(value)
		if common.TgBotFarmPlotPrice <= 0 {
			common.TgBotFarmPlotPrice = 2000000
		}
	case "TgBotFarmDogPrice":
		common.TgBotFarmDogPrice, _ = strconv.Atoi(value)
		if common.TgBotFarmDogPrice <= 0 {
			common.TgBotFarmDogPrice = 5000000
		}
	case "TgBotFarmDogFoodPrice":
		common.TgBotFarmDogFoodPrice, _ = strconv.Atoi(value)
		if common.TgBotFarmDogFoodPrice <= 0 {
			common.TgBotFarmDogFoodPrice = 500000
		}
	case "TgBotFarmDogGrowHours":
		common.TgBotFarmDogGrowHours, _ = strconv.Atoi(value)
		if common.TgBotFarmDogGrowHours <= 0 {
			common.TgBotFarmDogGrowHours = 24
		}
	case "TgBotFarmDogGuardRate":
		common.TgBotFarmDogGuardRate, _ = strconv.Atoi(value)
		if common.TgBotFarmDogGuardRate < 0 {
			common.TgBotFarmDogGuardRate = 0
		}
		if common.TgBotFarmDogGuardRate > 100 {
			common.TgBotFarmDogGuardRate = 100
		}
	case "TgBotFarmWaterInterval":
		common.TgBotFarmWaterInterval, _ = strconv.Atoi(value)
		if common.TgBotFarmWaterInterval <= 0 {
			common.TgBotFarmWaterInterval = 7200
		}
	case "TgBotFarmWiltDuration":
		common.TgBotFarmWiltDuration, _ = strconv.Atoi(value)
		if common.TgBotFarmWiltDuration <= 0 {
			common.TgBotFarmWiltDuration = 3600
		}
	case "TgBotFarmEventChance":
		common.TgBotFarmEventChance, _ = strconv.Atoi(value)
		if common.TgBotFarmEventChance < 0 {
			common.TgBotFarmEventChance = 0
		}
		if common.TgBotFarmEventChance > 100 {
			common.TgBotFarmEventChance = 100
		}
	case "TgBotFarmDisasterChance":
		common.TgBotFarmDisasterChance, _ = strconv.Atoi(value)
		if common.TgBotFarmDisasterChance < 0 {
			common.TgBotFarmDisasterChance = 0
		}
		if common.TgBotFarmDisasterChance > 100 {
			common.TgBotFarmDisasterChance = 100
		}
	case "TgBotFarmStealCooldown":
		common.TgBotFarmStealCooldown, _ = strconv.Atoi(value)
		if common.TgBotFarmStealCooldown < 0 {
			common.TgBotFarmStealCooldown = 0
		}
	case "TgBotFarmSoilMaxLevel":
		common.TgBotFarmSoilMaxLevel, _ = strconv.Atoi(value)
		if common.TgBotFarmSoilMaxLevel < 1 {
			common.TgBotFarmSoilMaxLevel = 1
		}
		if common.TgBotFarmSoilMaxLevel > 10 {
			common.TgBotFarmSoilMaxLevel = 10
		}
	case "TgBotFarmSoilUpgradePrice2":
		common.TgBotFarmSoilUpgradePrice2, _ = strconv.Atoi(value)
		if common.TgBotFarmSoilUpgradePrice2 <= 0 {
			common.TgBotFarmSoilUpgradePrice2 = 1000000
		}
	case "TgBotFarmSoilUpgradePrice3":
		common.TgBotFarmSoilUpgradePrice3, _ = strconv.Atoi(value)
		if common.TgBotFarmSoilUpgradePrice3 <= 0 {
			common.TgBotFarmSoilUpgradePrice3 = 3000000
		}
	case "TgBotFarmSoilUpgradePrice4":
		common.TgBotFarmSoilUpgradePrice4, _ = strconv.Atoi(value)
		if common.TgBotFarmSoilUpgradePrice4 <= 0 {
			common.TgBotFarmSoilUpgradePrice4 = 6000000
		}
	case "TgBotFarmSoilUpgradePrice5":
		common.TgBotFarmSoilUpgradePrice5, _ = strconv.Atoi(value)
		if common.TgBotFarmSoilUpgradePrice5 <= 0 {
			common.TgBotFarmSoilUpgradePrice5 = 10000000
		}
	case "TgBotFarmSoilSpeedBonus":
		common.TgBotFarmSoilSpeedBonus, _ = strconv.Atoi(value)
		if common.TgBotFarmSoilSpeedBonus < 0 {
			common.TgBotFarmSoilSpeedBonus = 0
		}
		if common.TgBotFarmSoilSpeedBonus > 50 {
			common.TgBotFarmSoilSpeedBonus = 50
		}
	// 牧场
	case "TgBotRanchMaxAnimals":
		common.TgBotRanchMaxAnimals, _ = strconv.Atoi(value)
		if common.TgBotRanchMaxAnimals < 1 {
			common.TgBotRanchMaxAnimals = 1
		}
		if common.TgBotRanchMaxAnimals > 20 {
			common.TgBotRanchMaxAnimals = 20
		}
	case "TgBotRanchFeedPrice":
		common.TgBotRanchFeedPrice, _ = strconv.Atoi(value)
	case "TgBotRanchWaterPrice":
		common.TgBotRanchWaterPrice, _ = strconv.Atoi(value)
	case "TgBotRanchFeedInterval":
		common.TgBotRanchFeedInterval, _ = strconv.Atoi(value)
		if common.TgBotRanchFeedInterval < 60 {
			common.TgBotRanchFeedInterval = 60
		}
	case "TgBotRanchWaterInterval":
		common.TgBotRanchWaterInterval, _ = strconv.Atoi(value)
		if common.TgBotRanchWaterInterval < 60 {
			common.TgBotRanchWaterInterval = 60
		}
	case "TgBotRanchHungerDeathHours":
		common.TgBotRanchHungerDeathHours, _ = strconv.Atoi(value)
		if common.TgBotRanchHungerDeathHours < 1 {
			common.TgBotRanchHungerDeathHours = 1
		}
	case "TgBotRanchThirstDeathHours":
		common.TgBotRanchThirstDeathHours, _ = strconv.Atoi(value)
		if common.TgBotRanchThirstDeathHours < 1 {
			common.TgBotRanchThirstDeathHours = 1
		}
	case "TgBotRanchChickenPrice":
		common.TgBotRanchChickenPrice, _ = strconv.Atoi(value)
	case "TgBotRanchDuckPrice":
		common.TgBotRanchDuckPrice, _ = strconv.Atoi(value)
	case "TgBotRanchGoosePrice":
		common.TgBotRanchGoosePrice, _ = strconv.Atoi(value)
	case "TgBotRanchPigPrice":
		common.TgBotRanchPigPrice, _ = strconv.Atoi(value)
	case "TgBotRanchSheepPrice":
		common.TgBotRanchSheepPrice, _ = strconv.Atoi(value)
	case "TgBotRanchCowPrice":
		common.TgBotRanchCowPrice, _ = strconv.Atoi(value)
	case "TgBotRanchChickenGrowSecs":
		common.TgBotRanchChickenGrowSecs, _ = strconv.ParseInt(value, 10, 64)
	case "TgBotRanchDuckGrowSecs":
		common.TgBotRanchDuckGrowSecs, _ = strconv.ParseInt(value, 10, 64)
	case "TgBotRanchGooseGrowSecs":
		common.TgBotRanchGooseGrowSecs, _ = strconv.ParseInt(value, 10, 64)
	case "TgBotRanchPigGrowSecs":
		common.TgBotRanchPigGrowSecs, _ = strconv.ParseInt(value, 10, 64)
	case "TgBotRanchSheepGrowSecs":
		common.TgBotRanchSheepGrowSecs, _ = strconv.ParseInt(value, 10, 64)
	case "TgBotRanchCowGrowSecs":
		common.TgBotRanchCowGrowSecs, _ = strconv.ParseInt(value, 10, 64)
	case "TgBotRanchChickenMeatPrice":
		common.TgBotRanchChickenMeatPrice, _ = strconv.Atoi(value)
	case "TgBotRanchDuckMeatPrice":
		common.TgBotRanchDuckMeatPrice, _ = strconv.Atoi(value)
	case "TgBotRanchGooseMeatPrice":
		common.TgBotRanchGooseMeatPrice, _ = strconv.Atoi(value)
	case "TgBotRanchPigMeatPrice":
		common.TgBotRanchPigMeatPrice, _ = strconv.Atoi(value)
	case "TgBotRanchSheepMeatPrice":
		common.TgBotRanchSheepMeatPrice, _ = strconv.Atoi(value)
	case "TgBotRanchCowMeatPrice":
		common.TgBotRanchCowMeatPrice, _ = strconv.Atoi(value)
	case "TgBotRanchManureInterval":
		common.TgBotRanchManureInterval, _ = strconv.Atoi(value)
		if common.TgBotRanchManureInterval < 60 {
			common.TgBotRanchManureInterval = 60
		}
	case "TgBotRanchManureCleanPrice":
		common.TgBotRanchManureCleanPrice, _ = strconv.Atoi(value)
	case "TgBotRanchManureGrowPenalty":
		common.TgBotRanchManureGrowPenalty, _ = strconv.Atoi(value)
		if common.TgBotRanchManureGrowPenalty < 0 {
			common.TgBotRanchManureGrowPenalty = 0
		}
		if common.TgBotRanchManureGrowPenalty > 90 {
			common.TgBotRanchManureGrowPenalty = 90
		}
	case "TurnstileSiteKey":
		common.TurnstileSiteKey = value
	case "TurnstileSecretKey":
		common.TurnstileSecretKey = value
	case "QuotaForNewUser":
		common.QuotaForNewUser, _ = strconv.Atoi(value)
	case "QuotaForInviter":
		common.QuotaForInviter, _ = strconv.Atoi(value)
	case "QuotaForInvitee":
		common.QuotaForInvitee, _ = strconv.Atoi(value)
	case "QuotaRemindThreshold":
		common.QuotaRemindThreshold, _ = strconv.Atoi(value)
	case "PreConsumedQuota":
		common.PreConsumedQuota, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitCount":
		setting.ModelRequestRateLimitCount, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitDurationMinutes":
		setting.ModelRequestRateLimitDurationMinutes, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitSuccessCount":
		setting.ModelRequestRateLimitSuccessCount, _ = strconv.Atoi(value)
	case "RequestRiskControlBurstLimit":
		setting.RequestRiskControlBurstLimit, _ = strconv.Atoi(value)
	case "RequestRiskControlBurstWindow":
		setting.RequestRiskControlBurstWindow, _ = strconv.Atoi(value)
	case "RequestRiskControlTokenThreshold":
		setting.RequestRiskControlTokenThreshold, _ = strconv.Atoi(value)
	case "RequestRiskControlTokenWindow":
		setting.RequestRiskControlTokenWindow, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitGroup":
		err = setting.UpdateModelRequestRateLimitGroupByJSONString(value)
	case "RetryTimes":
		common.RetryTimes, _ = strconv.Atoi(value)
	case "DataExportInterval":
		common.DataExportInterval, _ = strconv.Atoi(value)
	case "DataExportDefaultTime":
		common.DataExportDefaultTime = value
	case "ModelRatio":
		err = ratio_setting.UpdateModelRatioByJSONString(value)
	case "GroupRatio":
		err = ratio_setting.UpdateGroupRatioByJSONString(value)
	case "GroupGroupRatio":
		err = ratio_setting.UpdateGroupGroupRatioByJSONString(value)
	case "UserUsableGroups":
		err = setting.UpdateUserUsableGroupsByJSONString(value)
	case "CompletionRatio":
		err = ratio_setting.UpdateCompletionRatioByJSONString(value)
	case "ModelPrice":
		err = ratio_setting.UpdateModelPriceByJSONString(value)
	case "CacheRatio":
		err = ratio_setting.UpdateCacheRatioByJSONString(value)
	case "CreateCacheRatio":
		err = ratio_setting.UpdateCreateCacheRatioByJSONString(value)
	case "ImageRatio":
		err = ratio_setting.UpdateImageRatioByJSONString(value)
	case "AudioRatio":
		err = ratio_setting.UpdateAudioRatioByJSONString(value)
	case "AudioCompletionRatio":
		err = ratio_setting.UpdateAudioCompletionRatioByJSONString(value)
	case "TopUpLink":
		common.TopUpLink = value
	//case "ChatLink":
	//	common.ChatLink = value
	//case "ChatLink2":
	//	common.ChatLink2 = value
	case "ChannelDisableThreshold":
		common.ChannelDisableThreshold, _ = strconv.ParseFloat(value, 64)
	case "QuotaPerUnit":
		common.QuotaPerUnit, _ = strconv.ParseFloat(value, 64)
	case "SensitiveWords":
		setting.SensitiveWordsFromString(value)
	case "AutomaticDisableKeywords":
		operation_setting.AutomaticDisableKeywordsFromString(value)
	case "AutomaticDisableStatusCodes":
		err = operation_setting.AutomaticDisableStatusCodesFromString(value)
	case "AutomaticRetryStatusCodes":
		err = operation_setting.AutomaticRetryStatusCodesFromString(value)
	case "StreamCacheQueueLength":
		setting.StreamCacheQueueLength, _ = strconv.Atoi(value)
	case "PayMethods":
		err = operation_setting.UpdatePayMethodsByJsonString(value)
	}
	return err
}

// handleConfigUpdate 处理分层配置更新，返回是否已处理
func handleConfigUpdate(key, value string) bool {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return false // 不是分层配置
	}

	configName := parts[0]
	configKey := parts[1]

	// 获取配置对象
	cfg := config.GlobalConfig.Get(configName)
	if cfg == nil {
		return false // 未注册的配置
	}

	// 更新配置
	configMap := map[string]string{
		configKey: value,
	}
	config.UpdateConfigFromMap(cfg, configMap)

	// 特定配置的后处理
	if configName == "performance_setting" {
		// 同步磁盘缓存配置到 common 包
		performance_setting.UpdateAndSync()
	}

	return true // 已处理
}
