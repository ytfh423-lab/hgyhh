package common

import (
	"crypto/tls"
	//"os"
	//"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

var StartTime = time.Now().Unix() // unit: second
var Version = "v0.0.0"            // this hard coding will be replaced automatically when building, no need to manually change
var SystemName = "NPC-API"
var Footer = ""
var Logo = ""
var TopUpLink = ""

// var ChatLink = ""
// var ChatLink2 = ""
var QuotaPerUnit = 500 * 1000.0 // $0.002 / 1K tokens
// 保留旧变量以兼容历史逻辑，实际展示由 general_setting.quota_display_type 控制
var DisplayInCurrencyEnabled = true
var DisplayTokenStatEnabled = true
var DrawingEnabled = true
var TaskEnabled = true
var DataExportEnabled = true
var DataExportInterval = 5         // unit: minute
var DataExportDefaultTime = "hour" // unit: minute
var DefaultCollapseSidebar = false // default value of collapse sidebar

// Any options with "Secret", "Token" in its key won't be return by GetOptions

var SessionSecret = uuid.New().String()
var CryptoSecret = uuid.New().String()

var OptionMap map[string]string
var OptionMapRWMutex sync.RWMutex

var ItemsPerPage = 10
var MaxRecentItems = 1000

var PasswordLoginEnabled = true
var PasswordRegisterEnabled = true
var EmailVerificationEnabled = false
var GitHubOAuthEnabled = false
var LinuxDOOAuthEnabled = false
var WeChatAuthEnabled = false
var TelegramOAuthEnabled = false
var TurnstileCheckEnabled = false
var RegisterEnabled = true

var EmailDomainRestrictionEnabled = false // 是否启用邮箱域名限制
var EmailAliasRestrictionEnabled = false  // 是否启用邮箱别名限制
var EmailDomainWhitelist = []string{
	"gmail.com",
	"163.com",
	"126.com",
	"qq.com",
	"outlook.com",
	"hotmail.com",
	"icloud.com",
	"yahoo.com",
	"foxmail.com",
}
var EmailLoginAuthServerList = []string{
	"smtp.sendcloud.net",
	"smtp.azurecomm.net",
}

var DebugEnabled bool
var MemoryCacheEnabled bool

var LogConsumeEnabled = true

var TLSInsecureSkipVerify bool
var InsecureTLSConfig = &tls.Config{InsecureSkipVerify: true}

var SMTPServer = ""
var SMTPPort = 587
var SMTPSSLEnabled = false
var SMTPAccount = ""
var SMTPFrom = ""
var SMTPToken = ""

var GitHubClientId = ""
var GitHubClientSecret = ""
var LinuxDOClientId = ""
var LinuxDOClientSecret = ""
var LinuxDOMinimumTrustLevel = 0

var WeChatServerAddress = ""
var WeChatServerToken = ""
var WeChatAccountQRCodeImageURL = ""

var TurnstileSiteKey = ""
var TurnstileSecretKey = ""

var TelegramBotToken = ""
var TelegramBotName = ""
var TgBotLotteryEnabled = false
var TgBotLotteryMessagesRequired = 10
var TgBotLotteryWinRate = 30 // 中奖概率百分比
var TgBotFarmPlotPrice = 2000000      // 购买土地价格（quota单位，默认$4）
var TgBotFarmDogPrice = 5000000       // 买狗价格（quota单位，默认$10）
var TgBotFarmDogFoodPrice = 500000    // 狗粮价格（quota单位，默认$1）
var TgBotFarmDogGrowHours = 24        // 小狗长大所需小时
var TgBotFarmDogGuardRate = 50        // 看门狗拦截偷菜概率%
var TgBotFarmWaterInterval = 7200     // 浇水间隔秒数（默认2小时）
var TgBotFarmWiltDuration = 3600      // 枯萎到死亡秒数（默认1小时）
var TgBotFarmEventChance = 30         // 随机事件(虫害)概率%
var TgBotFarmDisasterChance = 15      // 天灾(干旱)概率%，不处理会死亡
var TgBotFarmStealCooldown = 1800     // 偷菜冷却秒数
var TgBotFarmSoilMaxLevel = 5         // 泥土最高等级
var TgBotFarmSoilUpgradePrice2 = 1000000  // 升级到2级价格（quota单位，默认$2）
var TgBotFarmSoilUpgradePrice3 = 3000000  // 升级到3级价格（quota单位，默认$6）
var TgBotFarmSoilUpgradePrice4 = 6000000  // 升级到4级价格（quota单位，默认$12）
var TgBotFarmSoilUpgradePrice5 = 10000000 // 升级到5级价格（quota单位，默认$20）
var TgBotFarmSoilSpeedBonus = 10      // 每级泥土加速百分比（默认10%）

// 牧场相关
var TgBotRanchMaxAnimals = 6              // 最大养殖数量
var TgBotRanchFeedPrice = 200000          // 饲料价格（quota单位，默认$0.4）
var TgBotRanchWaterPrice = 100000         // 饮水价格（quota单位，默认$0.2）
var TgBotRanchFeedInterval = 14400        // 喂食间隔秒数（默认4小时）
var TgBotRanchWaterInterval = 10800       // 喂水间隔秒数（默认3小时）
var TgBotRanchHungerDeathHours = 24       // 断食多少小时后死亡
var TgBotRanchThirstDeathHours = 18       // 断水多少小时后死亡
// 动物购买价格
var TgBotRanchChickenPrice = 500000       // 鸡价格（$1）
var TgBotRanchDuckPrice = 800000          // 鸭价格（$1.6）
var TgBotRanchGoosePrice = 1200000        // 鹅价格（$2.4）
var TgBotRanchPigPrice = 3000000          // 猪价格（$6）
var TgBotRanchSheepPrice = 4000000        // 羊价格（$8）
var TgBotRanchCowPrice = 8000000          // 牛价格（$16）
// 动物生长时间（秒）
var TgBotRanchChickenGrowSecs int64 = 28800   // 鸡 8小时
var TgBotRanchDuckGrowSecs int64 = 43200      // 鸭 12小时
var TgBotRanchGooseGrowSecs int64 = 57600     // 鹅 16小时
var TgBotRanchPigGrowSecs int64 = 86400       // 猪 24小时
var TgBotRanchSheepGrowSecs int64 = 115200    // 羊 32小时
var TgBotRanchCowGrowSecs int64 = 172800      // 牛 48小时
// 肉类出售价格（管理员可配置）
var TgBotRanchChickenMeatPrice = 1500000      // 鸡肉 $3
var TgBotRanchDuckMeatPrice = 2500000         // 鸭肉 $5
var TgBotRanchGooseMeatPrice = 4000000        // 鹅肉 $8
var TgBotRanchPigMeatPrice = 10000000         // 猪肉 $20
var TgBotRanchSheepMeatPrice = 14000000       // 羊肉 $28
var TgBotRanchCowMeatPrice = 28000000         // 牛肉 $56
// 粪便清理
var TgBotRanchManureInterval = 21600          // 粪便清理间隔秒数（默认6小时）
var TgBotRanchManureCleanPrice = 150000       // 清理费用（quota单位，默认$0.3）
var TgBotRanchManureGrowPenalty = 30          // 脏污时生长减速百分比（默认30%）

// 钓鱼相关
var TgBotFishBaitPrice = 250000                   // 鱼饵价格（quota单位，默认$0.5）
var TgBotFishActionCD = 5                         // 单次钓鱼操作CD秒（防连点，默认5秒）
var TgBotFishStaminaMax = 20                      // 钓鱼体力上限
var TgBotFishStaminaCost = 1                      // 每次钓鱼消耗体力
var TgBotFishStaminaRecoverInterval = 300          // 体力恢复间隔秒（默认5分钟）
var TgBotFishStaminaRecoverAmount = 1              // 每次恢复体力量
var TgBotFishFatigueEnabled = true                 // 是否开启疲劳衰减
var TgBotFishFatigueThreshold = 30                 // 每日钓鱼N次后进入疲劳
var TgBotFishFatigueDecay = 50                     // 疲劳时稀有鱼概率衰减%
var TgBotFishDailyMaxActions = 60                  // 每日最大有效钓鱼次数
var TgBotFishDailyMaxIncome = 100000000            // 每日最大钓鱼收入（quota，默认$200）
var TgBotFishRiskEnabled = true                    // 是否开启钓鱼风控检测

// 市场价格波动
var TgBotMarketRefreshHours = 4              // 市场刷新间隔（小时）
var TgBotMarketMinMultiplier = 50            // 最低价格倍率%（默认50%）
var TgBotMarketMaxMultiplier = 200           // 最高价格倍率%（默认200%）

// 等级系统
var TgBotFarmMaxLevel = 15
// 每级升级价格（quota），索引0=升到2级，索引13=升到15级
var TgBotFarmLevelPrices = []int{
	500000, 1000000, 2000000, 3000000, 5000000,
	8000000, 12000000, 18000000, 25000000, 35000000,
	50000000, 70000000, 100000000, 150000000,
}
// 功能解锁等级
var TgBotFarmUnlockSteal = 2      // 偷菜
var TgBotFarmUnlockDog = 2        // 狗狗
var TgBotFarmUnlockRanch = 3      // 牧场
var TgBotFarmUnlockFish = 3       // 钓鱼
var TgBotFarmUnlockWorkshop = 4   // 加工坊
var TgBotFarmUnlockMarket = 2     // 市场
var TgBotFarmUnlockTasks = 1      // 每日任务
var TgBotFarmUnlockAchieve = 1    // 成就

// 银行贷款系统
var TgBotFarmBankAdminId = 1              // 管理员账户ID（放款从此扣，还款加回）
var TgBotFarmBankInterestRate = 10        // 利率百分比（如10=10%）
var TgBotFarmBankMaxLoanDays = 7          // 最长还款天数
var TgBotFarmBankBaseAmount = 50000000    // 基础贷款额度(quota) = $100
var TgBotFarmBankMaxMultiplier = 10       // 信用评分最高倍率
var TgBotFarmBankUnlockLevel = 3          // 解锁银行功能的等级
var TgBotFarmMortgageMaxAmount = 500000000 // 抵押贷款最高额度(quota) = $1000
var TgBotFarmMortgageInterestRate = 15    // 抵押贷款利率百分比

// 季节系统
var TgBotFarmSeasonDays = 7               // 每个季节持续天数（默认7天）
var TgBotFarmSeasonInBonus = 70           // 应季价格倍率%（默认70%=打7折）
var TgBotFarmSeasonOffBonus = 140         // 反季价格倍率%（默认140%=涨40%）
var TgBotFarmSeasonInGrowth = 80          // 应季生长时间倍率%（默认80%=加速20%）
var TgBotFarmSeasonOffGrowth = 130        // 反季生长时间倍率%（默认130%=减速30%）
var TgBotFarmSeasonInYield = 115          // 应季产量倍率%（默认115%=增产15%）
var TgBotFarmSeasonOffYield = 80          // 反季产量倍率%（默认80%=减产20%）
var TgBotFarmSeasonOffEventBonus = 50     // 反季事件概率增幅%（默认+50%）
var TgBotFarmSeasonOffWaterPenalty = 25   // 反季浇水间隔缩短%（默认缩25%）
var TgBotFarmWarehouseMaxSlots = 100          // 仓库基础存储格数（1级）
var TgBotFarmWarehouseMeatExpiry = 259200     // 肉类基础保质期秒数（默认3天）
var TgBotFarmWarehouseRecipeExpiry = 432000   // 加工食品基础保质期秒数（默认5天）
var TgBotFarmWarehouseMaxLevel = 10           // 仓库最大等级
var TgBotFarmWarehouseUpgradePrice = 2000000  // 仓库每级升级价格（quota）
var TgBotFarmWarehouseCapacityPerLevel = 50   // 每级增加的容量
var TgBotFarmWarehouseExpiryBonusPerLevel = 20 // 每级增加的保质期百分比

// 天气系统
var TgBotFarmWeatherDurationMin = 14400
var TgBotFarmWeatherDurationMax = 28800
var TgBotFarmWeatherSunnyGrowthBonus = 20
var TgBotFarmWeatherStormyEventBonus = 50
var TgBotFarmWeatherFoggyStealBonus = 30

// 转生系统
var TgBotFarmPrestigeMinLevel = 15
var TgBotFarmPrestigeBonusPerLevel = 5

// 自动化设施
var TgBotFarmIrrigationPrice = 5000000
var TgBotFarmAutoFeederPrice = 8000000
var TgBotFarmScarecrowPrice = 3000000
var TgBotFarmScarecrowDefenseRate = 30

// 树场系统
var TgBotFarmUnlockTreeFarm = 5          // 树场解锁等级
var TgBotTreeFarmSlotPrice = 3000000     // 扩展树位价格 (quota)
var TgBotTreeFarmWaterInterval = 14400   // 浇水间隔秒数（4小时）
var TgBotTreeFarmWaterBonus = 15         // 浇水加速百分比
var TgBotTreeFarmFertilizerBonus = 20    // 施肥加速百分比
var TgBotTreeFarmStumpClearSecs = 7200   // 树桩清理冷却秒数（默认2小时）

// 交易系统
var TgBotFarmTradeFee = 5
var TgBotFarmTradeMaxListings = 10

// 小游戏
var TgBotFarmWheelPrice = 500000
var TgBotFarmScratchPrice = 250000

// 新功能解锁等级
var TgBotFarmUnlockLeaderboard = 3
var TgBotFarmUnlockTrading = 5
var TgBotFarmUnlockGames = 4
var TgBotFarmUnlockEncyclopedia = 2
var TgBotFarmUnlockAutomation = 6

// 农场内测系统
var FarmBetaEnabled = false    // 内测开关（开启后农场受内测控制）
var FarmBetaMaxSlots = 100     // 最大预约名额
var FarmBetaAdminBypass = true  // 管理员是否绕过内测限制
var FarmBetaDurationDays = 14   // 内测持续天数（从开始时间算起）

var QuotaForNewUser = 0
var QuotaForInviter = 0
var QuotaForInvitee = 0
var ChannelDisableThreshold = 5.0
var AutomaticDisableChannelEnabled = false
var AutomaticEnableChannelEnabled = false
var QuotaRemindThreshold = 1000
var PreConsumedQuota = 500

var RetryTimes = 0

//var RootUserEmail = ""

var IsMasterNode bool

var requestInterval int
var RequestInterval time.Duration

var SyncFrequency int // unit is second

var BatchUpdateEnabled = false
var BatchUpdateInterval int

var RelayTimeout int // unit is second

var RelayMaxIdleConns int
var RelayMaxIdleConnsPerHost int

var GeminiSafetySetting string

// https://docs.cohere.com/docs/safety-modes Type; NONE/CONTEXTUAL/STRICT
var CohereSafetySetting string

const (
	RequestIdKey = "X-Oneapi-Request-Id"
)

const (
	RoleGuestUser  = 0
	RoleCommonUser = 1
	RoleAdminUser  = 10
	RoleRootUser   = 100
)

func IsValidateRole(role int) bool {
	return role == RoleGuestUser || role == RoleCommonUser || role == RoleAdminUser || role == RoleRootUser
}

var (
	FileUploadPermission    = RoleGuestUser
	FileDownloadPermission  = RoleGuestUser
	ImageUploadPermission   = RoleGuestUser
	ImageDownloadPermission = RoleGuestUser
)

// All duration's unit is seconds
// Shouldn't larger then RateLimitKeyExpirationDuration
var (
	GlobalApiRateLimitEnable   bool
	GlobalApiRateLimitNum      int
	GlobalApiRateLimitDuration int64

	GlobalWebRateLimitEnable   bool
	GlobalWebRateLimitNum      int
	GlobalWebRateLimitDuration int64

	CriticalRateLimitEnable   bool
	CriticalRateLimitNum            = 20
	CriticalRateLimitDuration int64 = 20 * 60

	UploadRateLimitNum            = 10
	UploadRateLimitDuration int64 = 60

	DownloadRateLimitNum            = 10
	DownloadRateLimitDuration int64 = 60

	// Per-user search rate limit (applies after authentication, keyed by user ID)
	SearchRateLimitNum            = 10
	SearchRateLimitDuration int64 = 60
)

var RateLimitKeyExpirationDuration = 20 * time.Minute

const (
	UserStatusEnabled  = 1 // don't use 0, 0 is the default value!
	UserStatusDisabled = 2 // also don't use 0
)

const (
	TokenStatusEnabled   = 1 // don't use 0, 0 is the default value!
	TokenStatusDisabled  = 2 // also don't use 0
	TokenStatusExpired   = 3
	TokenStatusExhausted = 4
)

const (
	RedemptionCodeStatusEnabled  = 1 // don't use 0, 0 is the default value!
	RedemptionCodeStatusDisabled = 2 // also don't use 0
	RedemptionCodeStatusUsed     = 3 // also don't use 0
)

const (
	RedemptionPurposeLegacy       = 0
	RedemptionPurposeTopUp        = 1
	RedemptionPurposeRegistration = 2
)

const (
	ChannelStatusUnknown          = 0
	ChannelStatusEnabled          = 1 // don't use 0, 0 is the default value!
	ChannelStatusManuallyDisabled = 2 // also don't use 0
	ChannelStatusAutoDisabled     = 3
)

const (
	TopUpStatusPending = "pending"
	TopUpStatusSuccess = "success"
	TopUpStatusExpired = "expired"
	TopUpStatusRefunded = "refunded"
)
