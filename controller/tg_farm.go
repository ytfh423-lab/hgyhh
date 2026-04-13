package controller

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ========== 农场游戏定义 ==========

type farmCropDef struct {
	Key       string
	Short     string // callback abbreviation
	Name      string
	Emoji     string
	SeedCost  int   // quota units
	GrowSecs  int64 // seconds to grow
	MaxYield  int   // max harvest yield count
	UnitPrice int   // quota per unit harvested
	Season    int   // 0=春 1=夏 2=秋 3=冬
}

// 季节常量
const (
	SeasonSpring = 0
	SeasonSummer = 1
	SeasonAutumn = 2
	SeasonWinter = 3
)

var seasonNames = []string{"春", "夏", "秋", "冬"}
var seasonEmojis = []string{"🌸", "☀️", "🍂", "❄️"}

// 季节起始时间戳（程序启动时初始化，以服务器时间为基准）
var seasonEpoch int64

func init() {
	// 使用 2024-01-01 00:00:00 UTC 作为季节纪元（从春天开始）
	epoch, _ := time.Parse("2006-01-02", "2024-01-01")
	seasonEpoch = epoch.Unix()
}

// getCurrentSeason 获取当前季节 0=春 1=夏 2=秋 3=冬
func getCurrentSeason() int {
	days := common.TgBotFarmSeasonDays
	if days <= 0 {
		days = 7
	}
	elapsed := time.Now().Unix() - seasonEpoch
	seasonIndex := (elapsed / int64(days*86400)) % 4
	return int(seasonIndex)
}

// getSeasonName 获取季节名称
func getSeasonName(s int) string {
	return seasonEmojis[s] + " " + seasonNames[s] + "季"
}

// getSeasonDaysLeft 获取当前季节剩余天数
func getSeasonDaysLeft() int {
	days := common.TgBotFarmSeasonDays
	if days <= 0 {
		days = 7
	}
	elapsed := time.Now().Unix() - seasonEpoch
	cycleSecs := int64(days * 86400)
	secondsIntoSeason := elapsed % cycleSecs
	left := (cycleSecs - secondsIntoSeason) / 86400
	if left < 1 {
		left = 1
	}
	return int(left)
}

// isCropInSeason 判断作物是否应季
func isCropInSeason(crop *farmCropDef) bool {
	return crop.Season == getCurrentSeason()
}

// getSeasonAt 获取指定时间点的季节
func getSeasonAt(timestamp int64) int {
	days := common.TgBotFarmSeasonDays
	if days <= 0 {
		days = 7
	}
	elapsed := timestamp - seasonEpoch
	seasonIndex := (elapsed / int64(days*86400)) % 4
	return int(seasonIndex)
}

// getSeasonGrowthMultiplier 获取作物的季节生长时间倍率%（基于种植时的季节）
func getSeasonGrowthMultiplier(crop *farmCropDef, plantedAt int64) int {
	if crop.Season == getSeasonAt(plantedAt) {
		return common.TgBotFarmSeasonInGrowth
	}
	return common.TgBotFarmSeasonOffGrowth
}

// getSeasonYieldMultiplier 获取作物的季节产量倍率%（基于种植时的季节）
func getSeasonYieldMultiplier(crop *farmCropDef, plantedAt int64) int {
	if crop.Season == getSeasonAt(plantedAt) {
		return common.TgBotFarmSeasonInYield
	}
	return common.TgBotFarmSeasonOffYield
}

// isCropInSeasonAt 判断作物在指定时间是否应季
func isCropInSeasonAt(crop *farmCropDef, plantedAt int64) bool {
	return crop.Season == getSeasonAt(plantedAt)
}

// getSeasonEventChance 计算考虑季节后的事件概率
func getSeasonEventChance(baseChance int, crop *farmCropDef, plantedAt int64) int {
	if isCropInSeasonAt(crop, plantedAt) {
		return baseChance
	}
	// 反季：概率增加
	return baseChance * (100 + common.TgBotFarmSeasonOffEventBonus) / 100
}

// getSeasonWaterInterval 计算考虑季节后的浇水间隔
func getSeasonWaterInterval(baseInterval int64, crop *farmCropDef, plantedAt int64) int64 {
	if isCropInSeasonAt(crop, plantedAt) {
		return baseInterval
	}
	// 反季：浇水间隔缩短
	result := baseInterval * int64(100-common.TgBotFarmSeasonOffWaterPenalty) / 100
	if result < 600 {
		result = 600 // 最尐10分钟
	}
	return result
}

// getSeasonPriceMultiplier 获取季节价格倍率（百分比）
func getSeasonPriceMultiplier(crop *farmCropDef) int {
	if isCropInSeason(crop) {
		return common.TgBotFarmSeasonInBonus // 应季便宜
	}
	return common.TgBotFarmSeasonOffBonus // 反季贵
}

// applySeasonPrice 应用季节价格
func applySeasonPrice(basePrice int, crop *farmCropDef) int {
	m := getSeasonPriceMultiplier(crop)
	return basePrice * m / 100
}

type farmItemDef struct {
	Key   string
	Name  string
	Emoji string
	Cost  int    // quota units
	Cures string // event type it cures
}

// NOTE: all emojis below must be Unicode 6.0-11.0 for wide compatibility
// Season: 0=春 1=夏 2=秋 3=冬
var farmCrops = []farmCropDef{
	// === 作物收益平衡体系 ===
	// 档位设计（按平均利润/小时）:
	//   ⚡极速(≤30分): ~400-480k/h  高频快刷，适合活跃玩家
	//   🔄常规(30分~2h): ~230-330k/h  稳定收益，适合日常
	//   ⚖均衡(2~5h):   ~196-227k/h  中等收益，省心种植
	//   💤离线(5h+):    ~193-207k/h  最高总收益，适合睡前/上班前种
	// 长周期作物每小时收益略低，但单次总收益远高于短周期
	// 原有作物
	{"carrot", "car", "胡萝卜", "🌰", 50000, 1800, 2, 170000, SeasonSpring},
	{"tomato", "tom", "番茄", "🍅", 150000, 3600, 5, 135000, SeasonSummer},
	{"pumpkin", "pum", "南瓜", "🎃", 350000, 7200, 6, 250000, SeasonAutumn},
	{"blueberry", "blu", "蓝莓", "🍇", 200000, 10800, 10, 160000, SeasonSpring},
	{"strawberry", "str", "草莓", "🍓", 750000, 14400, 6, 470000, SeasonSpring},
	{"watermelon", "wat", "西瓜", "🍉", 1250000, 21600, 8, 535000, SeasonSummer},
	{"mango", "man", "芒果", "🍊", 350000, 25200, 8, 400000, SeasonSummer},
	{"corn", "cor", "玉米", "🌽", 500000, 54000, 10, 650000, SeasonAutumn},
	// 蔬菜
	{"potato", "pot", "土豆", "🥔", 40000, 1200, 3, 100000, SeasonSpring},
	{"eggplant", "egg", "茄子", "🍆", 80000, 2400, 4, 120000, SeasonSummer},
	{"pepper", "pep", "辣椒", "🌶️", 120000, 3000, 6, 90000, SeasonSummer},
	{"cucumber", "cuc", "黄瓜", "🥒", 60000, 1500, 5, 80000, SeasonSpring},
	{"broccoli", "bro", "西兰花", "🥦", 200000, 4800, 3, 280000, SeasonAutumn},
	{"garlic", "gar", "大蒜", "🧄", 100000, 5400, 8, 100000, SeasonWinter},
	{"onion", "oni", "洋葱", "🧅", 70000, 3600, 7, 80000, SeasonAutumn},
	{"lettuce", "let", "生菜", "🥬", 30000, 900, 4, 60000, SeasonSpring},
	// 水果
	{"apple", "app", "苹果", "🍎", 300000, 7200, 4, 300000, SeasonAutumn},
	{"peach", "pea", "桃子", "🍑", 450000, 10800, 5, 350000, SeasonSummer},
	{"cherry", "che", "樱桃", "🍒", 500000, 9000, 10, 180000, SeasonSpring},
	{"lemon", "lem", "柠檬", "🍋", 180000, 5400, 6, 150000, SeasonWinter},
	{"pear", "par", "梨子", "🍐", 250000, 7200, 5, 220000, SeasonAutumn},
	{"kiwi", "kiw", "猕猴桃", "🥝", 500000, 14400, 8, 300000, SeasonWinter},
	{"banana", "ban", "香蕉", "🍌", 900000, 18000, 7, 480000, SeasonSummer},
}

// getCropTier 返回作物档位 key 和中文名
func getCropTier(crop *farmCropDef) (string, string) {
	hours := float64(crop.GrowSecs) / 3600.0
	switch {
	case hours <= 0.5:
		return "sprint", "⚡极速"
	case hours <= 2.0:
		return "active", "🔄常规"
	case hours <= 5.0:
		return "balanced", "⚖均衡"
	default:
		return "afk", "💤离线"
	}
}

// getCropTags 返回作物价值标签列表
func getCropTags(crop *farmCropDef) []string {
	var tags []string
	hours := float64(crop.GrowSecs) / 3600.0
	maxProfit := crop.MaxYield*crop.UnitPrice - crop.SeedCost

	if hours >= 8 {
		tags = append(tags, "睡前种植")
	} else if hours >= 5 {
		tags = append(tags, "适合离线")
	}
	if maxProfit >= 4000000 {
		tags = append(tags, "高总收益")
	}
	if hours <= 0.5 {
		tags = append(tags, "快速回本")
	}
	if isCropInSeason(crop) {
		tags = append(tags, "当季作物")
	}
	// 每小时平均利润
	avgYield := float64(1+crop.MaxYield) / 2.0
	avgProfitPerHour := (avgYield*float64(crop.UnitPrice) - float64(crop.SeedCost)) / hours
	if avgProfitPerHour >= 400000 {
		tags = append(tags, "高效快刷")
	}
	return tags
}

var farmItems = []farmItemDef{
	{"pesticide", "杀虫剂", "🧪", 150000, "bugs"},
	{"fertilizer", "化肥", "🧴", 200000, ""},
	{"fertilizer_adv", "高级化肥", "🧪", 500000, ""},
	{"dogfood", "狗粮", "🦴", 500000, ""},
	{"fishbait", "鱼饵", "🪱", common.TgBotFishBaitPrice, ""},
	{"premiumfishbait", "高级鱼饵", "✨", common.TgBotFishPremiumBaitPrice, ""},
}

// ========== 钓鱼定义 ==========

type fishDef struct {
	Key       string
	Name      string
	Emoji     string
	Rarity    string
	Weight    int // probability weight (higher = more common)
	SellPrice int // quota units
}

var fishTypes = []fishDef{
	// ── 普通 ── 总权重418 (~79.6%)
	{"crucian", "鲫鱼", "🐟", "普通", 80, 80000},
	{"sardine", "沙丁鱼", "🐟", "普通", 70, 90000},
	{"carp", "鲤鱼", "🐟", "普通", 60, 120000},
	{"catfish", "鲶鱼", "🐟", "普通", 55, 150000},
	{"tilapia", "罗非鱼", "🐠", "普通", 48, 130000},
	{"perch", "鲈鱼", "🐟", "普通", 40, 200000},
	{"tropical", "热带鱼", "🐠", "普通", 35, 250000},
	{"goldfish", "金鱼", "🐠", "普通", 30, 180000},
	// ── 优良 ── 总权重69 (~13.1%)
	{"shrimp", "虾", "🦐", "优良", 18, 400000},
	{"squid", "鱿鱼", "🦑", "优良", 14, 500000},
	{"crab", "螃蟹", "🦀", "优良", 12, 550000},
	{"puffer", "河豚", "🐡", "优良", 10, 700000},
	{"trout", "鳟鱼", "🐟", "优良", 8, 650000},
	{"eel", "鳗鱼", "🐍", "优良", 7, 800000},
	// ── 稀有 ── 总权重12 (~2.3%) [原10%→2.3%]
	{"lobster", "龙虾", "🦞", "稀有", 4, 1500000},
	{"octopus", "章鱼", "🐙", "稀有", 3, 2000000},
	{"swordfish", "剑鱼", "🐟", "稀有", 2, 3000000},
	{"tuna", "金枪鱼", "🐟", "稀有", 2, 2500000},
	{"seahorse", "海马", "🐠", "稀有", 1, 4000000},
	// ── 史诗 ── 总权重4 (~0.76%) [原2%→0.76%]
	{"shark", "鲨鱼", "🦈", "史诗", 2, 8000000},
	{"manta", "蝠鲼", "🐬", "史诗", 1, 12000000},
	{"marlin", "旗鱼", "🎏", "史诗", 1, 15000000},
	// ── 传说 ── 总权重2 (~0.38%) [原1%→0.38%]
	{"whale", "鲸鱼", "🐋", "传说", 1, 30000000},
	{"goldendragon", "金龙鱼", "🐉", "传说", 1, 50000000},
}

var fishTypeMap map[string]*fishDef
var fishTotalWeight int

// getFishWeight 返回第idx条鱼的可配置权重，落回 fishTypes[idx].Weight
func getFishWeight(idx int) int {
	if idx >= 0 && idx < len(common.TgBotFishWeightsParsed) {
		return common.TgBotFishWeightsParsed[idx]
	}
	if idx >= 0 && idx < len(fishTypes) {
		return fishTypes[idx].Weight
	}
	return 0
}

// ========== 加工坊配方 ==========

type recipeDef struct {
	Key       string
	Name      string
	Emoji     string
	Cost      int   // quota to start
	TimeSecs  int64 // processing time
	SellPrice int   // sell price (before market multiplier)
}

var recipes = []recipeDef{
	// 原有加工品
	{"bread", "面包", "🍞", 500000, 1800, 900000},
	{"juice", "果汁", "🧃", 750000, 2700, 1400000},
	{"butter", "黄油", "🧈", 1000000, 2400, 1750000},
	{"cake", "蛋糕", "🍰", 1500000, 3600, 2750000},
	{"cheese", "奶酪", "🧀", 2000000, 5400, 3750000},
	{"wine", "葡萄酒", "🍷", 2500000, 7200, 5000000},
	{"chocolate", "巧克力", "🍫", 4000000, 10800, 8000000},
	// 加工食品
	{"salad", "沙拉", "🥗", 400000, 600, 700000},
	{"popcorn", "爆米花", "🍿", 300000, 900, 550000},
	{"cookie", "曲奇", "🍪", 650000, 1500, 1200000},
	{"donut", "甜甜圈", "🍩", 800000, 1800, 1500000},
	{"noodles", "面条", "🍜", 600000, 1200, 1050000},
	{"pizza", "披萨", "🍕", 1200000, 2400, 2200000},
	{"pie", "馅饼", "🥧", 1800000, 3000, 3200000},
	{"icecream", "冰淇淋", "🍦", 3000000, 7200, 6000000},
	// 肉制品
	{"dumpling", "饺子", "🥟", 800000, 1800, 1450000},
	{"drumstick", "炸鸡腿", "🍗", 1000000, 2400, 1800000},
	{"sausage", "香肠", "🌭", 1500000, 3600, 2800000},
	{"bacon", "培根", "🥓", 2000000, 4800, 3800000},
	{"burger", "汉堡", "🍔", 2500000, 5400, 4800000},
	{"ham", "火腿", "🍖", 3000000, 7200, 5500000},
	{"steak", "牛排", "🥩", 5000000, 10800, 9500000},
}

var recipeMap map[string]*recipeDef

// ========== 每日任务 & 成就 ==========

type dailyTaskDef struct {
	Action string
	Name   string
	Emoji  string
	Target int
	Reward int
}

var taskPool = []dailyTaskDef{
	{"plant", "种植作物", "🌱", 2, 700000},
	{"harvest", "收获作物", "🌾", 1, 600000},
	{"fish", "钓鱼", "🎣", 3, 1000000},
	{"steal", "偷菜", "🕵️", 1, 600000},
	{"craft", "加工产品", "🏭", 1, 700000},
	{"shop", "购买道具", "🏪", 2, 650000},
	{"ranch_feed", "喂食动物", "🌾", 2, 650000},
	{"ranch_water", "喂水动物", "💧", 2, 650000},
	{"fish_sell", "出售鱼获", "💰", 1, 650000},
	{"craft_sell", "收取加工品", "📥", 1, 700000},
	{"ranch_sell", "出售肉类", "🥩", 1, 750000},
	{"water", "浇水", "💧", 3, 800000},
	{"warehouse_sell", "仓库出售", "📦", 1, 650000},
	{"trade", "交易", "🔄", 1, 700000},
	{"game", "玩小游戏", "🎰", 2, 600000},
}

// ========== 任务元数据（按 action 类型提供描述/提示/自动化信息）==========

type actionMeta struct {
	Verb     string // 动作动词，如 "种植作物"
	Unit     string // 计数单位，如 "次"
	Desc     string // 完成条件模板，{target} 会被替换
	Hint     string // 操作提示
	AutoType string // 关联自动化类型（"irrigation"/"auto_feeder"/""）
	AutoText string // 自动化兼容说明
}

var actionMetaMap = map[string]actionMeta{
	"plant":          {"种植作物", "次", "在空地块上种植 {target} 次作物", "前往农场 → 种植", "", ""},
	"harvest":        {"收获作物", "次", "收获 {target} 次成熟作物", "等作物成熟后点击收获", "", ""},
	"fish":           {"钓鱼", "次", "完成 {target} 次钓鱼", "前往钓鱼页面 → 抛竿", "", ""},
	"steal":          {"偷菜", "次", "偷取他人 {target} 次作物", "前往偷菜 → 选择好友", "", ""},
	"craft":          {"加工产品", "次", "在加工坊启动 {target} 次加工", "前往加工坊 → 选择配方", "", ""},
	"shop":           {"购买道具", "次", "在商店购买 {target} 次道具/种子", "前往商店 → 购买", "", ""},
	"ranch_feed":     {"喂食动物", "次", "给动物喂食 {target} 次", "前往牧场 → 喂食", "auto_feeder", "⚡ 自动喂食器可自动完成"},
	"ranch_water":    {"喂水动物", "次", "给动物喂水 {target} 次", "前往牧场 → 喂水", "auto_feeder", "⚡ 自动喂食器可自动完成"},
	"fish_sell":      {"出售鱼获", "次", "出售 {target} 次鱼获", "钓到鱼后选择出售", "", ""},
	"fish_store":     {"鱼获入仓", "次", "将 {target} 次鱼获存入仓库", "钓到鱼后选择存入仓库（非出售）", "", ""},
	"craft_sell":     {"收取加工品", "次", "收取 {target} 次完成的加工品", "加工完成后点击收取", "", ""},
	"craft_store":    {"加工品入库", "次", "将 {target} 次加工品存入仓库", "加工品完成后选择存入仓库", "", ""},
	"ranch_sell":     {"出售肉类", "次", "屠宰出售 {target} 次动物", "动物成熟后选择屠宰出售", "", ""},
	"ranch_buy":      {"购买动物", "次", "购买 {target} 次牧场动物", "前往牧场 → 购买动物", "", ""},
	"ranch_clean":    {"清理粪便", "次", "清理 {target} 次牧场粪便", "牧场有粪便时点击清理", "", ""},
	"ranch_store":    {"肉品入库", "次", "将 {target} 次肉品存入仓库", "屠宰后选择存入仓库", "", ""},
	"warehouse_sell": {"仓库出售", "次", "从仓库出售 {target} 次物品", "前往仓库 → 选择出售", "", ""},
	"trade":          {"交易", "次", "在交易所完成 {target} 次交易", "前往交易所 → 挂单/购买", "", ""},
	"game":           {"玩小游戏", "次", "完成 {target} 次小游戏", "前往小游戏 → 选择游戏", "", ""},
	"water":          {"浇水", "次", "给作物浇水 {target} 次", "作物需要水时点击浇水", "irrigation", "⚡ 灌溉系统可自动完成"},
	"repay":          {"还款", "次", "偿还 {target} 次贷款", "前往银行 → 还款", "", ""},
	"loan":           {"贷款", "次", "申请 {target} 次贷款", "前往银行 → 贷款", "", ""},
}

// getTaskDesc 根据 action + target 生成完成条件说明
func getTaskDesc(action string, target int) string {
	m, ok := actionMetaMap[action]
	if !ok {
		return fmt.Sprintf("完成 %d 次", target)
	}
	return strings.ReplaceAll(m.Desc, "{target}", strconv.Itoa(target))
}

var dailyTaskCount = 10

type achievementDef struct {
	Key         string
	Name        string
	Emoji       string
	Description string
	Action      string
	Target      int64
	Reward      int
}

var achievements = []achievementDef{
	{"first_plant", "初出茅庐", "🌱", "首次种植作物", "plant", 1, 500000},
	{"plant_50", "种植达人", "🌾", "累计种植50次", "plant", 50, 2000000},
	{"plant_200", "农场传说", "👨‍🌾", "累计种植200次", "plant", 200, 7500000},
	{"harvest_30", "丰收使者", "🧺", "累计收获30次", "harvest", 30, 1500000},
	{"harvest_100", "收获之王", "👑", "累计收获100次", "harvest", 100, 4000000},
	{"fish_20", "钓鱼爱好者", "🎣", "累计钓鱼20次", "fish", 20, 1000000},
	{"fish_100", "钓鱼大师", "🐟", "累计钓鱼100次", "fish", 100, 5000000},
	{"steal_10", "小偷小摸", "🕵️", "累计偷菜10次", "steal", 10, 800000},
	{"steal_50", "江洋大盗", "🦹", "累计偷菜50次", "steal", 50, 3000000},
	{"steal_100", "神偷", "🥷", "累计偷菜100次", "steal", 100, 6000000},
	{"craft_20", "加工达人", "🏭", "累计加工20次", "craft", 20, 2000000},
	{"craft_50", "工匠大师", "⚒️", "累计加工50次", "craft", 50, 4000000},
	{"fish_sell_10", "渔商", "💰", "累计卖鱼10次", "fish_sell", 10, 1500000},
	{"ranch_buy_5", "牧场主", "🐄", "累计买动物5次", "ranch_buy", 5, 1000000},
	{"ranch_sell_20", "肉贩", "🥩", "累计出售肉类20次", "ranch_sell", 20, 3000000},
	{"water_50", "浇水达人", "💧", "累计浇水50次", "water", 50, 1000000},
	{"trade_10", "商人", "🔄", "累计交易10次", "trade", 10, 2500000},
	{"game_20", "赌徒", "🎰", "累计玩小游戏20次", "game", 20, 1500000},
	{"levelup_5", "小有名气", "⭐", "升到5级", "levelup", 5, 2500000},
	{"levelup_10", "农场大亨", "🌟", "升到10级", "levelup", 10, 12500000},
	{"levelup_15", "传奇农夫", "💫", "升到满级15级", "levelup", 15, 50000000},
	{"prestige_1", "涅槃重生", "🔄", "完成首次转生", "prestige", 1, 25000000},
}

func getDailyTasks(dateStr string) []dailyTaskDef {
	// Use date string as seed for deterministic daily tasks
	seed := int64(0)
	for _, c := range dateStr {
		seed = seed*31 + int64(c)
	}
	r := rand.New(rand.NewSource(seed))
	perm := r.Perm(len(taskPool))
	var tasks []dailyTaskDef
	for i := 0; i < dailyTaskCount && i < len(perm); i++ {
		tasks = append(tasks, taskPool[perm[i]])
	}
	return tasks
}

func todayDateStr() string {
	return time.Now().Format("20060102")
}

// ========== 等级系统 ==========

type featureUnlock struct {
	Key   string
	Name  string
	Level *int
}

var featureUnlocks = []featureUnlock{
	{"tasks", "每日任务", &common.TgBotFarmUnlockTasks},
	{"achieve", "成就", &common.TgBotFarmUnlockAchieve},
	{"steal", "偷菜", &common.TgBotFarmUnlockSteal},
	{"dog", "狗狗", &common.TgBotFarmUnlockDog},
	{"market", "市场", &common.TgBotFarmUnlockMarket},
	{"encyclopedia", "图鉴", &common.TgBotFarmUnlockEncyclopedia},
	{"ranch", "牧场", &common.TgBotFarmUnlockRanch},
	{"fish", "钓鱼", &common.TgBotFarmUnlockFish},
	{"leaderboard", "排行榜", &common.TgBotFarmUnlockLeaderboard},
	{"workshop", "加工坊", &common.TgBotFarmUnlockWorkshop},
	{"games", "小游戏", &common.TgBotFarmUnlockGames},
	{"trading", "交易所", &common.TgBotFarmUnlockTrading},
	{"automation", "自动化", &common.TgBotFarmUnlockAutomation},
}

func checkFeatureLevel(tgId string, level int, requiredLevel int, featureName string, chatId int64, editMsgId int, from *TgUser) bool {
	if level >= requiredLevel {
		return true
	}
	farmSend(chatId, editMsgId, fmt.Sprintf("🔒 %s需要等级 %d 才能解锁（当前等级 %d）", featureName, requiredLevel, level), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "⬆️ 升级", CallbackData: "farm_levelup"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
	return false
}

func getLevelUpPrice(currentLevel int) int {
	idx := currentLevel - 1 // level 1 -> index 0 = price to reach level 2
	if idx < 0 {
		idx = 0
	}
	if idx >= len(common.TgBotFarmLevelPrices) {
		return common.TgBotFarmLevelPrices[len(common.TgBotFarmLevelPrices)-1]
	}
	return common.TgBotFarmLevelPrices[idx]
}

var farmCropMap map[string]*farmCropDef
var farmCropByShort map[string]*farmCropDef
var farmItemMap map[string]*farmItemDef

// ========== 偷菜辅助函数 ==========

// isPlotInProtection 检查地块是否在基础保护期内
func isPlotInProtection(plot *model.TgFarmPlot, cfg *model.FarmStealConfig) bool {
	if plot.MaturedAt == 0 {
		return false
	}
	protSecs := int64(cfg.OwnerProtectionMinutes) * 60
	return time.Now().Unix() < plot.MaturedAt+protSecs
}

// calcHarvestYield 计算收获产量（扣除被偷数量）
func calcHarvestYield(baseYield int, fertBonus int, stolenCount int) (realYield int, stolenLoss int) {
	totalYield := baseYield + fertBonus
	stolenLoss = stolenCount
	if stolenLoss > totalYield {
		stolenLoss = totalYield
	}
	realYield = totalYield - stolenLoss
	if realYield < 0 {
		realYield = 0
	}
	return
}

// ========== 市场价格波动（桥接新引擎） ==========

// marketSnapshot 保持与旧系统兼容的快照格式
type marketSnapshot struct {
	Timestamp int64          `json:"timestamp"`
	Prices    map[string]int `json:"prices"` // key -> multiplier%
}

func init() {
	farmCropMap = make(map[string]*farmCropDef)
	farmCropByShort = make(map[string]*farmCropDef)
	for i := range farmCrops {
		farmCropMap[farmCrops[i].Key] = &farmCrops[i]
		farmCropByShort[farmCrops[i].Short] = &farmCrops[i]
	}
	farmItemMap = make(map[string]*farmItemDef)
	for i := range farmItems {
		farmItemMap[farmItems[i].Key] = &farmItems[i]
	}
	fishTypeMap = make(map[string]*fishDef)
	fishTotalWeight = common.TgBotFishNothingWeight
	for i := range fishTypes {
		fishTypeMap[fishTypes[i].Key] = &fishTypes[i]
		fishTotalWeight += getFishWeight(i)
	}
	recipeMap = make(map[string]*recipeDef)
	for i := range recipes {
		recipeMap[recipes[i].Key] = &recipes[i]
	}
}

// ensureMarketFresh 确保市场价格是最新的（桥接新引擎）
func ensureMarketFresh() {
	ensureMarketEngine()
}

// getMarketMultiplier 获取商品倍率%（桥接新引擎）
func getMarketMultiplier(key string) int {
	return getMarketMultiplierNew(key)
}

// applyMarket 应用市场倍率到基础价格
func applyMarket(basePrice int, marketKey string) int {
	m := getMarketMultiplier(marketKey)
	return basePrice * m / 100
}

func farmQuotaStr(quota int) string {
	return fmt.Sprintf("$%.2f", float64(quota)/common.QuotaPerUnit)
}

// ========== 状态懒更新 ==========

func updateFarmPlotStatus(plot *model.TgFarmPlot) {
	if plot.Status == 0 || plot.Status == 2 {
		return
	}
	// 状态4(枯萎)检查是否死亡
	if plot.Status == 4 {
		now := time.Now().Unix()
		wiltDuration := int64(common.TgBotFarmWiltDuration)
		if plot.LastWateredAt > 0 {
			waterInterval := int64(common.TgBotFarmWaterInterval)
			wiltStart := plot.LastWateredAt + waterInterval
			if now >= wiltStart+wiltDuration {
				// 死亡：自动清空地块
				_ = model.ClearFarmPlot(plot.Id)
				plot.Status = 0
				plot.CropType = ""
			}
		}
		return
	}
	if plot.Status != 1 && plot.Status != 3 {
		return
	}
	now := time.Now().Unix()
	crop := farmCropMap[plot.CropType]
	if crop == nil {
		return
	}
	changed := false

	// 计算实际生长时间（含泥土加速）
	growSecs := crop.GrowSecs
	soilLevel := plot.SoilLevel
	if soilLevel < 1 {
		soilLevel = 1
	}
	if soilLevel > 1 {
		bonus := int64(common.TgBotFarmSoilSpeedBonus * (soilLevel - 1))
		growSecs = growSecs * (100 - bonus) / 100
		if growSecs < 60 {
			growSecs = 60
		}
	}
	// 季节生长倍率（基于种植时的季节）
	seasonGrowthPct := getSeasonGrowthMultiplier(crop, plot.PlantedAt)
	growSecs = growSecs * int64(seasonGrowthPct) / 100
	if growSecs < 60 {
		growSecs = 60
	}
	// 高级化肥：成熟时间缩短50%
	if plot.Fertilized == 2 {
		growSecs = growSecs * int64(100-AdvFertilizerGrowReduction) / 100
		if growSecs < 60 {
			growSecs = 60
		}
	}

	matureAt := plot.PlantedAt + growSecs

	// 浇水检查：生长中的作物需要定期浇水（反季间隔更短）
	if plot.Status == 1 && plot.LastWateredAt > 0 {
		waterInterval := getSeasonWaterInterval(int64(common.TgBotFarmWaterInterval), crop, plot.PlantedAt)
		waterDeadline := plot.LastWateredAt + waterInterval
		if now >= waterDeadline {
			// 如果作物在水耗尽前（或同时）已成熟，优先判定为成熟
			if matureAt <= waterDeadline {
				plot.Status = 2
				plot.MaturedAt = matureAt
				_ = model.UpdateFarmPlot(plot)
				return
			}
			// 否则枯萎
			plot.Status = 4
			_ = model.UpdateFarmPlot(plot)
			return
		}
	}

	// 事件触发优先
	if plot.Status == 1 && plot.EventAt > 0 && plot.EventType != "" && now >= plot.EventAt {
		// 拥有灌溉自动化时，干旱事件自动消除
		if plot.EventType == "drought" && model.HasAutomation(plot.TelegramId, "irrigation") {
			plot.EventType = ""
			plot.EventAt = 0
			changed = true
		} else {
			plot.Status = 3
			changed = true
		}
	}
	// 事件死亡检查：status=3 + (drought/bugs) + 超时未处理
	if plot.Status == 3 && (plot.EventType == "drought" || plot.EventType == "bugs") {
		wiltDuration := int64(common.TgBotFarmWiltDuration)
		if now >= plot.EventAt+wiltDuration {
			_ = model.ClearFarmPlot(plot.Id)
			plot.Status = 0
			plot.CropType = ""
			return
		}
	}
	// 成熟检查（无事件时）
	if plot.Status == 1 {
		if now >= matureAt {
			plot.Status = 2
			plot.MaturedAt = now
			changed = true
		}
	}
	if changed {
		_ = model.UpdateFarmPlot(plot)
	}
}

// ========== 用户绑定 ==========

func getFarmUser(tgId string) (*model.User, error) {
	user := &model.User{TelegramId: tgId}
	err := user.FillUserByTelegramId()
	return user, err
}

func farmBindingError(chatId int64, editMsgId int, from *TgUser) {
	text := "🔑 你还没有绑定平台账号！\n\n" +
		"请先私聊机器人发送你的 API Key（以 sk- 开头）完成绑定。\n" +
		"绑定后才能使用农场功能。\n\n" +
		"发送 /bindaccount 查看绑定说明。"
	farmSend(chatId, editMsgId, text, nil, from)
}

// ========== 命令入口 ==========

func handleFarmCommand(chatId int64, from *TgUser, isGroup bool) {
	if !isGroup {
		sendTgMessage(chatId, "🌾 农场游戏仅限群组中使用！\n\n请在群组里发送 /farm 开始种菜。\n私聊仅支持绑定账号功能。", from)
		return
	}
	tgId := strconv.FormatInt(from.Id, 10)
	if _, err := getFarmUser(tgId); err != nil {
		farmBindingError(chatId, 0, from)
		return
	}
	showFarmView(chatId, 0, tgId, from)
}

func handleFarmCallback(cb *TgCallbackQuery) {
	chatId := cb.Message.Chat.Id
	msgId := cb.Message.MessageId
	tgId := strconv.FormatInt(cb.From.Id, 10)
	data := cb.Data

	// 统一绑定检查：所有农场操作都需要绑定账号
	from := cb.From
	if _, err := getFarmUser(tgId); err != nil {
		farmBindingError(chatId, msgId, from)
		return
	}

	switch {
	case data == "farm":
		showFarmView(chatId, msgId, tgId, from)
	case data == "farm_plant":
		showFarmPlantCrops(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_p_"):
		cropShort := strings.TrimPrefix(data, "farm_p_")
		showFarmPlotSelection(chatId, msgId, tgId, cropShort, from)
	case strings.HasPrefix(data, "farm_pp_"):
		parts := strings.SplitN(strings.TrimPrefix(data, "farm_pp_"), "_", 2)
		if len(parts) == 2 {
			plotIdx, _ := strconv.Atoi(parts[0])
			doFarmPlant(chatId, msgId, tgId, plotIdx, parts[1], from)
		}
	case data == "farm_harvest":
		doFarmHarvest(chatId, msgId, tgId, from)
	case data == "farm_harvest_sell":
		doFarmHarvestSell(chatId, msgId, tgId, from)
	case data == "farm_harvest_store":
		doFarmHarvestStore(chatId, msgId, tgId, from)
	case data == "farm_warehouse":
		showFarmWarehouse(chatId, msgId, tgId, from)
	case data == "farm_wh_sellall":
		doFarmWarehouseSellAll(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_wh_sell_"):
		cropKey := strings.TrimPrefix(data, "farm_wh_sell_")
		doFarmWarehouseSell(chatId, msgId, tgId, cropKey, from)
	case data == "farm_shop":
		showFarmShop(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_buy_"):
		raw := strings.TrimPrefix(data, "farm_buy_")
		buyQty := 1
		buyKey := raw
		if idx := strings.LastIndex(raw, "_"); idx > 0 {
			if q, err := strconv.Atoi(raw[idx+1:]); err == nil && q > 0 {
				buyQty = q
				buyKey = raw[:idx]
			}
		}
		doFarmBuy(chatId, msgId, tgId, buyKey, buyQty, from)
	case data == "farm_steal":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockSteal, "偷菜", chatId, msgId, from) { return }
		showFarmStealTargets(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_st_"):
		victimId := strings.TrimPrefix(data, "farm_st_")
		doFarmSteal(chatId, msgId, tgId, victimId, from)
	case data == "farm_treat":
		showFarmTreatSelection(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_tr_"):
		plotStr := strings.TrimPrefix(data, "farm_tr_")
		plotIdx, _ := strconv.Atoi(plotStr)
		doFarmTreat(chatId, msgId, tgId, plotIdx, from)
	case data == "farm_fert":
		showFarmFertSelection(chatId, msgId, tgId, from)
	case data == "farm_fertall":
		doFarmFertilizeAll(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_ff_"):
		plotStr := strings.TrimPrefix(data, "farm_ff_")
		plotIdx, _ := strconv.Atoi(plotStr)
		doFarmFertilize(chatId, msgId, tgId, plotIdx, from)
	case data == "farm_buyland":
		doFarmBuyLand(chatId, msgId, tgId, from)
	case data == "farm_water":
		showFarmWaterSelection(chatId, msgId, tgId, from)
	case data == "farm_waterall":
		doFarmWaterAll(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_ww_"):
		plotStr := strings.TrimPrefix(data, "farm_ww_")
		plotIdx, _ := strconv.Atoi(plotStr)
		doFarmWater(chatId, msgId, tgId, plotIdx, from)
	case data == "farm_dog":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockDog, "狗狗", chatId, msgId, from) { return }
		showFarmDog(chatId, msgId, tgId, from)
	case data == "farm_buydog":
		doFarmBuyDog(chatId, msgId, tgId, from)
	case data == "farm_feeddog":
		doFarmFeedDog(chatId, msgId, tgId, from)
	case data == "farm_logs":
		showFarmLogs(chatId, msgId, tgId, from)
	case data == "farm_fish":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockFish, "钓鱼", chatId, msgId, from) { return }
		showFarmFish(chatId, msgId, tgId, from)
	case data == "farm_dofish":
		doFarmFish(chatId, msgId, tgId, from)
	case data == "farm_sellfish":
		doFarmSellFish(chatId, msgId, tgId, from)
	case data == "farm_storefish":
		doFarmStoreFish(chatId, msgId, tgId, from)
	case data == "farm_market":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockMarket, "市场", chatId, msgId, from) { return }
		showFarmMarket(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_chart_"):
		cat := strings.TrimPrefix(data, "farm_chart_")
		doFarmMarketChart(chatId, msgId, tgId, cat, from)
	case data == "farm_levelup":
		showFarmLevelUp(chatId, msgId, tgId, from)
	case data == "farm_dolevelup":
		doFarmLevelUp(chatId, msgId, tgId, from)
	case data == "farm_bank":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmBankUnlockLevel, "银行", chatId, msgId, from) { return }
		showFarmBank(chatId, msgId, tgId, from)
	case data == "farm_doloan":
		doFarmLoan(chatId, msgId, tgId, from)
	case data == "farm_mortgage":
		showFarmMortgage(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_domortgage_"):
		amtStr := strings.TrimPrefix(data, "farm_domortgage_")
		amt, _ := strconv.Atoi(amtStr)
		doFarmMortgage(chatId, msgId, tgId, amt, from)
	case data == "farm_repay":
		doFarmRepay(chatId, msgId, tgId, from)
	case data == "farm_repay_half":
		doFarmRepayPartial(chatId, msgId, tgId, 50, from)
	case data == "farm_tasks":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockTasks, "每日任务", chatId, msgId, from) { return }
		showFarmTasks(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_tclaim_"):
		idxStr := strings.TrimPrefix(data, "farm_tclaim_")
		idx, _ := strconv.Atoi(idxStr)
		doFarmClaimTask(chatId, msgId, tgId, idx, from)
	case data == "farm_achieve":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockAchieve, "成就", chatId, msgId, from) { return }
		showFarmAchievements(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_aclaim_"):
		achKey := strings.TrimPrefix(data, "farm_aclaim_")
		doFarmClaimAchievement(chatId, msgId, tgId, achKey, from)
	case data == "farm_workshop":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockWorkshop, "加工坊", chatId, msgId, from) { return }
		showFarmWorkshop(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_craft_"):
		recipeKey := strings.TrimPrefix(data, "farm_craft_")
		doFarmCraft(chatId, msgId, tgId, recipeKey, from)
	case data == "farm_collect":
		doFarmCollectAll(chatId, msgId, tgId, from)
	case data == "farm_collect_store":
		doFarmCollectStore(chatId, msgId, tgId, from)
	case data == "farm_soil":
		showFarmSoilUpgrade(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_su_"):
		plotStr := strings.TrimPrefix(data, "farm_su_")
		plotIdx, _ := strconv.Atoi(plotStr)
		doFarmSoilUpgrade(chatId, msgId, tgId, plotIdx, from)
	// ===== 新功能回调 =====
	case data == "farm_ency":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockEncyclopedia, "图鉴", chatId, msgId, from) { return }
		showFarmEncyclopedia(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_eclaim_"):
		cat := strings.TrimPrefix(data, "farm_eclaim_")
		doFarmClaimCollection(chatId, msgId, tgId, cat, from)
	case data == "farm_rank":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockLeaderboard, "排行", chatId, msgId, from) { return }
		showFarmLeaderboard(chatId, msgId, tgId, "balance", "global", "all", from)
	case strings.HasPrefix(data, "farm_rank_"):
		payload := strings.TrimPrefix(data, "farm_rank_")
		parts := strings.Split(payload, "_")
		boardType := "balance"
		scope := "global"
		period := "all"
		if len(parts) > 0 && parts[0] != "" {
			boardType = parts[0]
		}
		if len(parts) > 1 && parts[1] != "" {
			scope = parts[1]
		}
		if len(parts) > 2 && parts[2] != "" {
			period = parts[2]
		}
		showFarmLeaderboard(chatId, msgId, tgId, boardType, scope, period, from)
	case data == "farm_trade":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockTrading, "交易", chatId, msgId, from) { return }
		showFarmTradeMarket(chatId, msgId, tgId, from)
	case data == "farm_tsell":
		showFarmTradeSell(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_tlist_"):
		cropType := strings.TrimPrefix(data, "farm_tlist_")
		doFarmTradeList(chatId, msgId, tgId, cropType, from)
	case strings.HasPrefix(data, "farm_tbuy_"):
		tradeIdStr := strings.TrimPrefix(data, "farm_tbuy_")
		tradeId, _ := strconv.Atoi(tradeIdStr)
		doFarmTradeBuy(chatId, msgId, tgId, tradeId, from)
	case strings.HasPrefix(data, "farm_tcancel_"):
		tradeIdStr := strings.TrimPrefix(data, "farm_tcancel_")
		tradeId, _ := strconv.Atoi(tradeIdStr)
		doFarmTradeCancel(chatId, msgId, tgId, tradeId, from)
	case data == "farm_game":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockGames, "游戏", chatId, msgId, from) { return }
		showFarmGamesPage(chatId, msgId, tgId, 1, from)
	case strings.HasPrefix(data, "farm_gp_"):
		page, _ := strconv.Atoi(strings.TrimPrefix(data, "farm_gp_"))
		showFarmGamesPage(chatId, msgId, tgId, page, from)
	case strings.HasPrefix(data, "farm_g_"):
		gameKey := strings.TrimPrefix(data, "farm_g_")
		doMiniGame(chatId, msgId, tgId, gameKey, from)
	case data == "farm_wheel":
		doFarmWheel(chatId, msgId, tgId, from)
	case data == "farm_scratch":
		doFarmScratch(chatId, msgId, tgId, from)
	case data == "farm_auto":
		if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockAutomation, "自动化", chatId, msgId, from) { return }
		showFarmAutomation(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_abuy_"):
		autoType := strings.TrimPrefix(data, "farm_abuy_")
		doFarmBuyAutomation(chatId, msgId, tgId, autoType, from)
	case data == "farm_prestige":
		showFarmPrestige(chatId, msgId, tgId, from)
	case data == "farm_doprestige":
		doFarmPrestige(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "ranch"):
		if data == "ranch" {
			if !checkFeatureLevel(tgId, model.GetFarmLevel(tgId), common.TgBotFarmUnlockRanch, "牧场", chatId, msgId, from) { return }
		}
		handleRanchCallback(cb)
	}
}

// ========== 农场视图 ==========

func showFarmView(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	// 灌溉系统 或 阵雨天气：自动浇水
	wBot := GetCurrentWeather()
	if model.HasAutomation(tgId, "irrigation") || wBot.Type == 1 {
		now := time.Now().Unix()
		waterInterval := int64(common.TgBotFarmWaterInterval)
		for _, plot := range plots {
			if plot.Status == 1 && plot.LastWateredAt > 0 {
				if now-plot.LastWateredAt >= waterInterval/2 {
					_ = model.WaterFarmPlot(plot.Id)
					plot.LastWateredAt = now
					model.AddFarmLog(tgId, "water", 0, "💧自动灌溉")
				}
			}
		}
	}

	userLevel := model.GetFarmLevel(tgId)
	season := getCurrentSeason()
	daysLeft := getSeasonDaysLeft()
	text := fmt.Sprintf("🌾 我的农场  ⭐Lv.%d | %s (剩%d天)\n\n", userLevel, getSeasonName(season), daysLeft)
	hasEvent := false
	hasWiltOrGrowing := false
	for _, plot := range plots {
		text += farmPlotLine(plot) + "\n"
		if plot.Status == 3 && plot.EventType != "drought" {
			hasEvent = true
		}
		if plot.Status == 1 || plot.Status == 4 ||
			(plot.Status == 3 && plot.EventType == "drought") {
			hasWiltOrGrowing = true
		}
	}

	// 狗狗信息
	dog, dogErr := model.GetFarmDog(tgId)
	if dogErr == nil {
		model.UpdateDogHunger(dog)
		dogLevel := "🐶 幼犬"
		if dog.Level == 2 {
			dogLevel = "🐕 成犬"
		}
		guardStatus := ""
		if dog.Level == 2 && dog.Hunger > 0 {
			guardStatus = " ✅看门中"
		} else if dog.Hunger == 0 {
			guardStatus = " ❌饿坏了"
		} else {
			guardStatus = " ⏳成长中"
		}
		text += fmt.Sprintf("\n🐕 %s「%s」 饱食度:%d%%%s\n", dogLevel, dog.Name, dog.Hunger, guardStatus)
	}

	items, _ := model.GetFarmItems(tgId)
	if len(items) > 0 {
		text += "\n📦 背包："
		for _, item := range items {
			def := farmItemMap[item.ItemType]
			if def != nil {
				text += fmt.Sprintf(" %s%s×%d", def.Emoji, def.Name, item.Quantity)
			}
		}
		text += "\n"
	}

	// 天气
	w := GetCurrentWeather()
	text += fmt.Sprintf("\n%s 天气: %s", w.Emoji, w.Name)
	if w.Effects != "" {
		text += " (" + w.Effects + ")"
	}
	text += "\n"

	// 显示地块数量
	text += fmt.Sprintf("\n📊 土地 %d/%d 块", len(plots), model.FarmMaxPlots)
	if len(plots) < model.FarmMaxPlots {
		text += fmt.Sprintf(" | 购买新地 %s", farmQuotaStr(common.TgBotFarmPlotPrice))
	}
	text += "\n"

	var rows [][]TgInlineKeyboardButton
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🌱 种植", CallbackData: "farm_plant"},
		{Text: "🌾 收获", CallbackData: "farm_harvest"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🏪 商店", CallbackData: "farm_shop"},
		{Text: "📦 仓库", CallbackData: "farm_warehouse"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🕵️ 偷菜", CallbackData: "farm_steal"},
	})
	// 浇水按钮
	if hasWiltOrGrowing {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "💧 浇水", CallbackData: "farm_water"},
		})
	}
	// 有生长中作物时显示施肥按钮
	hasGrowing := false
	for _, plot := range plots {
		if plot.Status == 1 && plot.Fertilized == 0 {
			hasGrowing = true
			break
		}
	}
	if hasGrowing {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🧴 施肥", CallbackData: "farm_fert"},
		})
	}
	if hasEvent {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "💊 治疗", CallbackData: "farm_treat"},
		})
	}
	// 泥土升级按钮
	hasUpgradable := false
	for _, plot := range plots {
		sl := plot.SoilLevel
		if sl < 1 {
			sl = 1
		}
		if sl < common.TgBotFarmSoilMaxLevel {
			hasUpgradable = true
			break
		}
	}
	if hasUpgradable {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🌱 泥土升级", CallbackData: "farm_soil"},
		})
	}
	// 功能按钮（带锁标识）
	lockTag := func(name string, lvl int) string {
		if userLevel >= lvl {
			return name
		}
		return fmt.Sprintf("🔒%s(Lv%d)", name, lvl)
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: lockTag("🐕 狗狗", common.TgBotFarmUnlockDog), CallbackData: "farm_dog"},
		{Text: lockTag("🐄 牧场", common.TgBotFarmUnlockRanch), CallbackData: "ranch"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: lockTag("🎣 钓鱼", common.TgBotFarmUnlockFish), CallbackData: "farm_fish"},
		{Text: lockTag("🏭 加工", common.TgBotFarmUnlockWorkshop), CallbackData: "farm_workshop"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: lockTag("📝 任务", common.TgBotFarmUnlockTasks), CallbackData: "farm_tasks"},
		{Text: lockTag("🏆 成就", common.TgBotFarmUnlockAchieve), CallbackData: "farm_achieve"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "📈 市场", CallbackData: "farm_market"},
		{Text: "📋 记录", CallbackData: "farm_logs"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: lockTag("🏦 银行", common.TgBotFarmBankUnlockLevel), CallbackData: "farm_bank"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: lockTag("📖 图鉴", common.TgBotFarmUnlockEncyclopedia), CallbackData: "farm_ency"},
		{Text: lockTag("🏅 排行", common.TgBotFarmUnlockLeaderboard), CallbackData: "farm_rank"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: lockTag("🔄 交易", common.TgBotFarmUnlockTrading), CallbackData: "farm_trade"},
		{Text: lockTag("🎮 游戏", common.TgBotFarmUnlockGames), CallbackData: "farm_game"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: lockTag("⚡ 自动化", common.TgBotFarmUnlockAutomation), CallbackData: "farm_auto"},
	})
	if userLevel < common.TgBotFarmMaxLevel {
		price := getLevelUpPrice(userLevel)
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("⬆️ 升级 Lv.%d→%d (%s)", userLevel, userLevel+1, farmQuotaStr(price)), CallbackData: "farm_levelup"},
		})
	}
	if len(plots) < model.FarmMaxPlots {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🏗️ 购买土地 (%s)", farmQuotaStr(common.TgBotFarmPlotPrice)), CallbackData: "farm_buyland"},
		})
	}
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func farmPlotLine(plot *model.TgFarmPlot) string {
	idx := plot.PlotIndex + 1
	soilTag := ""
	sl := plot.SoilLevel
	if sl < 1 {
		sl = 1
	}
	if sl > 1 {
		soilTag = fmt.Sprintf(" 🌱Lv.%d", sl)
	}

	switch plot.Status {
	case 0:
		return fmt.Sprintf("⬜ %d号地 - 空地%s", idx, soilTag)
	case 1:
		crop := farmCropMap[plot.CropType]
		if crop == nil {
			return fmt.Sprintf("⬜ %d号地 - 空地", idx)
		}
		now := time.Now().Unix()
		elapsed := now - plot.PlantedAt
		total := crop.GrowSecs
		soilLvl := plot.SoilLevel
		if soilLvl < 1 {
			soilLvl = 1
		}
		if soilLvl > 1 {
			bonus := int64(common.TgBotFarmSoilSpeedBonus * (soilLvl - 1))
			total = total * (100 - bonus) / 100
			if total < 60 {
				total = 60
			}
		}
		pct := int(elapsed * 100 / total)
		if pct > 99 {
			pct = 99
		}
		remaining := total - elapsed
		fertTag := ""
		if plot.Fertilized == 1 {
			fertTag = " 🧴"
		}
		// 浇水倒计时
		waterTag := ""
		if plot.LastWateredAt > 0 {
			waterInterval := int64(common.TgBotFarmWaterInterval)
			nextWater := plot.LastWateredAt + waterInterval - now
			if nextWater > 0 {
				waterTag = fmt.Sprintf(" 💧%s", formatDuration(nextWater))
			} else {
				waterTag = " 💧⚠️需浇水"
			}
		}
		return fmt.Sprintf("%s %d号地 - %s 生长中 %d%% 剩余%s%s%s%s", crop.Emoji, idx, crop.Name, pct, formatDuration(remaining), fertTag, waterTag, soilTag)
	case 2:
		crop := farmCropMap[plot.CropType]
		if crop == nil {
			return fmt.Sprintf("✅ %d号地 - 已成熟", idx)
		}
		stolen := ""
		if plot.StolenCount > 0 {
			stolen = fmt.Sprintf(" 🍃被摘%d", plot.StolenCount)
		}
		protTag := ""
		cfg := model.GetStealConfig()
		if isPlotInProtection(plot, cfg) {
			remain := plot.MaturedAt + int64(cfg.OwnerProtectionMinutes)*60 - time.Now().Unix()
			if remain > 0 {
				protTag = fmt.Sprintf(" 🛡️保护%s", formatDuration(remain))
			}
		}
		return fmt.Sprintf("✅ %d号地 - %s%s 已成熟！%s%s%s", crop.Emoji, crop.Name, stolen, protTag, soilTag, "")
	case 3:
		crop := farmCropMap[plot.CropType]
		emoji := "❓"
		name := "未知"
		if crop != nil {
			emoji = crop.Emoji
			name = crop.Name
		}
		if plot.EventType == "drought" {
			now := time.Now().Unix()
			wiltDuration := int64(common.TgBotFarmWiltDuration)
			deathAt := plot.EventAt + wiltDuration
			remaining := deathAt - now
			if remaining < 0 {
				remaining = 0
			}
			return fmt.Sprintf("🏜️ %d号地 - %s%s 天灾干旱！💧快浇水救命！%s后死亡%s", idx, emoji, name, formatDuration(remaining), soilTag)
		}
		eventEmoji := "❌"
		eventLabel := "未知事件"
		switch plot.EventType {
		case "bugs":
			eventEmoji = "🐛"
			eventLabel = "虫害"
		}
		return fmt.Sprintf("%s %d号地 - %s %s%s！需要治疗%s", emoji, idx, name, eventEmoji, eventLabel, soilTag)
	case 4:
		crop := farmCropMap[plot.CropType]
		emoji := "🥀"
		name := "作物"
		if crop != nil {
			emoji = crop.Emoji
			name = crop.Name
		}
		now := time.Now().Unix()
		wiltDuration := int64(common.TgBotFarmWiltDuration)
		waterInterval := int64(common.TgBotFarmWaterInterval)
		deathAt := plot.LastWateredAt + waterInterval + wiltDuration
		remaining := deathAt - now
		if remaining < 0 {
			remaining = 0
		}
		return fmt.Sprintf("🥀 %d号地 - %s%s 枯萎中！💧快浇水！%s后死亡%s", idx, emoji, name, formatDuration(remaining), soilTag)
	}
	return fmt.Sprintf("❓ %d号地", idx)
}

func formatDuration(secs int64) string {
	if secs <= 0 {
		return "0分"
	}
	hours := secs / 3600
	mins := (secs % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%d时%d分", hours, mins)
	}
	return fmt.Sprintf("%d分", mins)
}

// ========== 种植 ==========

func showFarmPlantCrops(chatId int64, editMsgId int, tgId string, from *TgUser) {
	season := getCurrentSeason()
	text := fmt.Sprintf("🌱 选择要种植的作物：\n当前: %s\n\n", getSeasonName(season))
	var rows [][]TgInlineKeyboardButton
	for _, crop := range farmCrops {
		maxValue := crop.MaxYield * crop.UnitPrice
		_, tierName := getCropTier(&crop)
		seasonTag := seasonEmojis[crop.Season] + seasonNames[crop.Season]
		inSeason := ""
		if isCropInSeason(&crop) {
			inSeason = " ✅应季"
		}
		text += fmt.Sprintf("%s %s %s [%s%s] - 种子%s | %s | 产量1~%d | 最高%s\n",
			crop.Emoji, crop.Name, tierName, seasonTag, inSeason, farmQuotaStr(crop.SeedCost),
			formatDuration(crop.GrowSecs), crop.MaxYield, farmQuotaStr(maxValue))
		btnTag := ""
		if isCropInSeason(&crop) {
			btnTag = "✅"
		}
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s%s (%s)", crop.Emoji, crop.Name, btnTag, farmQuotaStr(crop.SeedCost)),
				CallbackData: "farm_p_" + crop.Short},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func showFarmPlotSelection(chatId int64, editMsgId int, tgId string, cropShort string, from *TgUser) {
	crop := farmCropByShort[cropShort]
	if crop == nil {
		farmSend(chatId, editMsgId, "❌ 未知作物", nil, from)
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}
	text := fmt.Sprintf("🌱 种植 %s%s\n选择空地：\n", crop.Emoji, crop.Name)
	var rows [][]TgInlineKeyboardButton
	hasEmpty := false
	for _, plot := range plots {
		if plot.Status == 0 {
			hasEmpty = true
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("⬜ %d号地", plot.PlotIndex+1),
					CallbackData: fmt.Sprintf("farm_pp_%d_%s", plot.PlotIndex, cropShort)},
			})
		}
	}
	if !hasEmpty {
		text += "\n❌ 没有空地了！请先收获或清理。"
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回", CallbackData: "farm_plant"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func doFarmPlant(chatId int64, editMsgId int, tgId string, plotIdx int, cropShort string, from *TgUser) {
	crop := farmCropByShort[cropShort]
	if crop == nil {
		farmSend(chatId, editMsgId, "❌ 未知作物", nil, from)
		return
	}
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}
	if user.Quota < crop.SeedCost {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！种子需要 %s，当前余额 %s",
			farmQuotaStr(crop.SeedCost), farmQuotaStr(user.Quota)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_plant"}},
			},
		}, from)
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	var targetPlot *model.TgFarmPlot
	for _, p := range plots {
		if p.PlotIndex == plotIdx {
			targetPlot = p
			break
		}
	}
	if targetPlot == nil || targetPlot.Status != 0 {
		farmSend(chatId, editMsgId, "❌ 该地块不可用", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_plant"}},
			},
		}, from)
		return
	}
	err = model.DecreaseUserQuota(user.Id, crop.SeedCost)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败，请稍后再试", nil, from)
		return
	}
	model.AddFarmLog(tgId, "plant", -crop.SeedCost, fmt.Sprintf("种植%s%s", crop.Emoji, crop.Name))

	now := time.Now().Unix()
	targetPlot.CropType = crop.Key
	targetPlot.PlantedAt = now
	targetPlot.Status = 1
	targetPlot.EventType = ""
	targetPlot.EventAt = 0
	targetPlot.StolenCount = 0
	targetPlot.LastWateredAt = now

	// 计算实际生长时间（含泥土加速）
	actualGrowSecs := crop.GrowSecs
	plotSoilLvl := targetPlot.SoilLevel
	if plotSoilLvl < 1 {
		plotSoilLvl = 1
	}
	if plotSoilLvl > 1 {
		soilBonus := int64(common.TgBotFarmSoilSpeedBonus * (plotSoilLvl - 1))
		actualGrowSecs = actualGrowSecs * (100 - soilBonus) / 100
		if actualGrowSecs < 60 {
			actualGrowSecs = 60
		}
	}

	// 季节生长倍率
	seasonGrowthPct := getSeasonGrowthMultiplier(crop, now)
	actualGrowSecs = actualGrowSecs * int64(seasonGrowthPct) / 100
	if actualGrowSecs < 60 {
		actualGrowSecs = 60
	}

	// 虫害事件（反季概率更高）
	bugChance := getSeasonEventChance(common.TgBotFarmEventChance, crop, now)
	if rand.Intn(100) < bugChance {
		targetPlot.EventType = "bugs"
		offset := actualGrowSecs * int64(30+rand.Intn(50)) / 100
		targetPlot.EventAt = now + offset
	}
	// 天灾(干旱)：独立概率，不与虫害叠加；拥有灌溉自动化则跳过；反季概率更高
	droughtChance := getSeasonEventChance(common.TgBotFarmDisasterChance, crop, now)
	if targetPlot.EventType == "" && !model.HasAutomation(tgId, "irrigation") && rand.Intn(100) < droughtChance {
		targetPlot.EventType = "drought"
		offset := actualGrowSecs * int64(30+rand.Intn(50)) / 100
		targetPlot.EventAt = now + offset
	}

	_ = model.UpdateFarmPlot(targetPlot)
	common.SysLog(fmt.Sprintf("TG Farm: user %s planted %s on plot %d, cost %d", tgId, crop.Key, plotIdx, crop.SeedCost))
	showFarmView(chatId, editMsgId, tgId, from)
}

// ========== 收获 ==========

// doFarmHarvest 显示收获预览，让玩家选择出售或入仓
func doFarmHarvest(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	matureCount := 0
	preview := ""
	for _, plot := range plots {
		if plot.Status == 2 {
			crop := farmCropMap[plot.CropType]
			if crop == nil {
				continue
			}
			matureCount++
			seasonTag := ""
			if isCropInSeason(crop) {
				seasonTag = "🏷️应季"
			} else {
				seasonTag = "📈反季"
			}
			preview += fmt.Sprintf("\n%s %s (%s)", crop.Emoji, crop.Name, seasonTag)
		}
	}

	if matureCount == 0 {
		farmSend(chatId, editMsgId, "🌾 没有可收获的作物。\n\n种植作物并等待成熟后即可收获！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🌱 去种植", CallbackData: "farm_plant"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	season := getCurrentSeason()
	text := fmt.Sprintf("🌾 可收获作物 (%d块地)\n当前: %s | 应季%d%% 反季%d%%\n%s\n\n选择收获方式：",
		matureCount, getSeasonName(season),
		common.TgBotFarmSeasonInBonus, common.TgBotFarmSeasonOffBonus, preview)

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "💰 收获并出售", CallbackData: "farm_harvest_sell"}},
			{{Text: "📦 收获到仓库", CallbackData: "farm_harvest_store"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// doFarmHarvestSell 收获并立即出售（应用市场价+季节价格）
func doFarmHarvestSell(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	totalQuota := 0
	harvestedCount := 0
	details := ""
	for _, plot := range plots {
		if plot.Status == 2 {
			crop := farmCropMap[plot.CropType]
			if crop == nil {
				continue
			}
			rawYield := 1 + rand.Intn(crop.MaxYield)
			// 季节产量修正
			yieldMult := getSeasonYieldMultiplier(crop, plot.PlantedAt)
			baseYield := rawYield * yieldMult / 100
			if baseYield < 1 {
				baseYield = 1
			}
			fertBonus := 0
			if plot.Fertilized == 1 {
				fertBonus = baseYield / 2
				if fertBonus < 1 {
					fertBonus = 1
				}
			}
			realYield, stolenLoss := calcHarvestYield(baseYield, fertBonus, plot.StolenCount)
			// 市场价 × 季节倍率
			marketPrice := applyMarket(crop.UnitPrice, "crop_"+crop.Key)
			seasonPrice := applySeasonPrice(marketPrice, crop)
			value := realYield * seasonPrice
			totalQuota += value
			harvestedCount++

			mPct := getMarketMultiplier("crop_" + crop.Key)
			sPct := getSeasonPriceMultiplier(crop)
			seasonTag := "应季"
			if !isCropInSeason(crop) {
				seasonTag = "反季"
			}
			details += fmt.Sprintf("\n%s %s: 产量%d", crop.Emoji, crop.Name, rawYield)
			if yieldMult != 100 {
				details += fmt.Sprintf("×季%d%%→%d", yieldMult, baseYield)
			}
			if fertBonus > 0 {
				details += fmt.Sprintf(" +化肥%d", fertBonus)
			}
			if stolenLoss > 0 {
				details += fmt.Sprintf(" -被摘%d", stolenLoss)
			}
			details += fmt.Sprintf(" = 实收%d × %s(市场%d%%×%s%d%%) = %s",
				realYield, farmQuotaStr(seasonPrice), mPct, seasonTag, sPct, farmQuotaStr(value))

			_ = model.ClearFarmPlot(plot.Id)
			model.RecordCollection(tgId, "crop", crop.Key, realYield)
		}
	}

	if harvestedCount == 0 {
		farmSend(chatId, editMsgId, "🌾 没有可收获的作物。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	err = model.IncreaseUserQuota(user.Id, totalQuota, true)
	if err != nil {
		common.SysError(fmt.Sprintf("TG Farm: increase quota failed for user %d: %s", user.Id, err.Error()))
	}
	model.AddFarmLog(tgId, "harvest", totalQuota, fmt.Sprintf("收获出售%d种作物", harvestedCount))

	text := fmt.Sprintf("🌾 收获出售完成！\n%s\n\n💰 共获得 %s 额度", details, farmQuotaStr(totalQuota))
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// doFarmHarvestStore 收获并存入仓库
func doFarmHarvestStore(chatId int64, editMsgId int, tgId string, from *TgUser) {
	_, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	currentTotal := model.GetWarehouseTotalCount(tgId)
	harvestedCount := 0
	storedTotal := 0
	details := ""
	for _, plot := range plots {
		if plot.Status == 2 {
			crop := farmCropMap[plot.CropType]
			if crop == nil {
				continue
			}
			rawYield := 1 + rand.Intn(crop.MaxYield)
			// 季节产量修正
			yieldMult := getSeasonYieldMultiplier(crop, plot.PlantedAt)
			baseYield := rawYield * yieldMult / 100
			if baseYield < 1 {
				baseYield = 1
			}
			fertBonus := 0
			if plot.Fertilized == 1 {
				fertBonus = baseYield / 2
				if fertBonus < 1 {
					fertBonus = 1
				}
			}
			realYield, stolenLoss := calcHarvestYield(baseYield, fertBonus, plot.StolenCount)

			whLevel := model.GetWarehouseLevel(tgId)
			whMax := model.GetWarehouseMaxSlots(whLevel)
			if currentTotal+storedTotal+realYield > whMax {
				details += fmt.Sprintf("\n%s %s: ❌ 仓库已满", crop.Emoji, crop.Name)
				continue
			}

			_ = model.AddToWarehouse(tgId, crop.Key, realYield)
			storedTotal += realYield
			harvestedCount++
			model.RecordCollection(tgId, "crop", crop.Key, realYield)

			details += fmt.Sprintf("\n%s %s: 产量%d", crop.Emoji, crop.Name, rawYield)
			if yieldMult != 100 {
				details += fmt.Sprintf("×季%d%%→%d", yieldMult, baseYield)
			}
			if fertBonus > 0 {
				details += fmt.Sprintf(" +化肥%d", fertBonus)
			}
			if stolenLoss > 0 {
				details += fmt.Sprintf(" -被摘%d", stolenLoss)
			}
			details += fmt.Sprintf(" = 入仓%d", realYield)

			_ = model.ClearFarmPlot(plot.Id)
		}
	}

	if harvestedCount == 0 {
		farmSend(chatId, editMsgId, "🌾 没有可收获的作物或仓库已满。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "📦 查看仓库", CallbackData: "farm_warehouse"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	model.AddFarmLog(tgId, "harvest", 0, fmt.Sprintf("收获入仓%d种作物共%d个", harvestedCount, storedTotal))

	text := fmt.Sprintf("📦 收获入仓完成！\n%s\n\n共存入 %d 个作物到仓库\n💡 可在市场价高时出售，注意应季产量高但价低，反季价高但产量低", details, storedTotal)
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "📦 查看仓库", CallbackData: "farm_warehouse"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 商店 ==========

func showFarmShop(chatId int64, editMsgId int, tgId string, from *TgUser) {
	text := "🏪 农场商店\n\n"
	text += "📌 种子（在「种植」中直接购买并种下）：\n"
	for _, crop := range farmCrops {
		_, tierName := getCropTier(&crop)
		maxProfit := crop.MaxYield*crop.UnitPrice - crop.SeedCost
		text += fmt.Sprintf("  %s %s %s - %s | %s | 1~%d个 | 利润%s\n",
			crop.Emoji, crop.Name, tierName, farmQuotaStr(crop.SeedCost),
			formatDuration(crop.GrowSecs), crop.MaxYield, farmQuotaStr(maxProfit))
	}
	text += "\n📌 道具：\n"
	var rows [][]TgInlineKeyboardButton
	for _, item := range farmItems {
		itemCost := item.Cost
		if item.Key == "dogfood" {
			itemCost = common.TgBotFarmDogFoodPrice
		}
		if item.Cures != "" {
			cureLabel := farmEventLabel(item.Cures)
			text += fmt.Sprintf("  %s %s - %s (治疗%s)\n", item.Emoji, item.Name, farmQuotaStr(itemCost), cureLabel)
		} else if item.Key == "dogfood" {
			text += fmt.Sprintf("  %s %s - %s (喂狗)\n", item.Emoji, item.Name, farmQuotaStr(itemCost))
		} else if item.Key == "fertilizer" {
			text += fmt.Sprintf("  %s %s - %s (施肥增产50%%)\n", item.Emoji, item.Name, farmQuotaStr(itemCost))
		} else {
			text += fmt.Sprintf("  %s %s - %s\n", item.Emoji, item.Name, farmQuotaStr(itemCost))
		}
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s ×1 (%s)", item.Emoji, item.Name, farmQuotaStr(itemCost)),
				CallbackData: "farm_buy_" + item.Key + "_1"},
			{Text: "×5 (" + farmQuotaStr(itemCost*5) + ")",
				CallbackData: "farm_buy_" + item.Key + "_5"},
			{Text: "×10 (" + farmQuotaStr(itemCost*10) + ")",
				CallbackData: "farm_buy_" + item.Key + "_10"},
		})
	}
	// 购买狗狗
	_, dogErr := model.GetFarmDog(tgId)
	if dogErr != nil {
		text += fmt.Sprintf("\n🐕 看门狗\n  🐶 小狗 - %s (长大后可拦截偷菜)\n", farmQuotaStr(common.TgBotFarmDogPrice))
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🐶 购买小狗 (%s)", farmQuotaStr(common.TgBotFarmDogPrice)),
				CallbackData: "farm_buydog"},
		})
	}
	text += "\n💡 种子直接在「🌱 种植」中购买"
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🌱 去种植", CallbackData: "farm_plant"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func doFarmBuy(chatId int64, editMsgId int, tgId string, itemKey string, qty int, from *TgUser) {
	item := farmItemMap[itemKey]
	if item == nil {
		farmSend(chatId, editMsgId, "❌ 未知道具", nil, from)
		return
	}
	if qty < 1 {
		qty = 1
	}
	if qty > 99 {
		qty = 99
	}
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}
	unitCost := item.Cost
	if itemKey == "dogfood" {
		unitCost = common.TgBotFarmDogFoodPrice
	}
	totalCost := unitCost * qty
	if user.Quota < totalCost {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！需要 %s（单价 %s × %d）", farmQuotaStr(totalCost), farmQuotaStr(unitCost), qty), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回商店", CallbackData: "farm_shop"}},
			},
		}, from)
		return
	}
	err = model.DecreaseUserQuota(user.Id, totalCost)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败", nil, from)
		return
	}
	err = model.IncrementFarmItem(tgId, itemKey, qty)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, totalCost, true)
		farmSend(chatId, editMsgId, "❌ 购买失败", nil, from)
		return
	}
	model.AddFarmLog(tgId, "shop", -totalCost, fmt.Sprintf("购买%s%s×%d", item.Emoji, item.Name, qty))
	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 购买 %s%s ×%d 成功！已扣除 %s",
		item.Emoji, item.Name, qty, farmQuotaStr(totalCost)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🏪 继续购物", CallbackData: "farm_shop"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 偷菜 ==========

func showFarmStealTargets(chatId int64, editMsgId int, tgId string, from *TgUser) {
	cfg := model.GetStealConfig()
	backBtn := &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}

	if !cfg.StealEnabled {
		farmSend(chatId, editMsgId, "🚫 偷菜功能当前已关闭。", backBtn, from)
		return
	}

	// 检查今日次数
	thiefToday := model.CountThiefStealsToday(tgId)
	if thiefToday >= int64(cfg.MaxStealPerUserPerDay) {
		farmSend(chatId, editMsgId, fmt.Sprintf("⏳ 今日偷菜次数已达上限（%d/%d次）。明天再来！",
			thiefToday, cfg.MaxStealPerUserPerDay), backBtn, from)
		return
	}

	targets, err := model.GetMatureFarmTargetsV2(tgId)
	if err != nil || len(targets) == 0 {
		farmSend(chatId, editMsgId, "🕵️ 暂时没有可偷的菜地。\n\n等其他玩家的作物成熟后再来！", backBtn, from)
		return
	}
	text := fmt.Sprintf("🕵️ 可偷取的农场：\n\n📋 规则: 成熟%d分钟后可自由偷取\n📊 今日: %d/%d次\n\n",
		cfg.OwnerProtectionMinutes, thiefToday, cfg.MaxStealPerUserPerDay)
	var rows [][]TgInlineKeyboardButton
	for _, t := range targets {
		masked := maskTgId(t.TelegramId)
		text += fmt.Sprintf("👤 %s - %d块可摘\n", masked, t.Count)
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🕵️ 摘 %s 的额外收益", masked),
				CallbackData: "farm_st_" + t.TelegramId},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func doFarmSteal(chatId int64, editMsgId int, tgId string, victimId string, from *TgUser) {
	backBtn := &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🕵️ 看看别人", CallbackData: "farm_steal"}, {Text: "🔙 返回", CallbackData: "farm"}},
		},
	}

	if tgId == victimId {
		farmSend(chatId, editMsgId, "❌ 不能偷自己的菜！", nil, from)
		return
	}

	cfg := model.GetStealConfig()
	if !cfg.StealEnabled {
		farmSend(chatId, editMsgId, "🚫 偷菜功能当前已关闭。", backBtn, from)
		return
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	// 每日偷菜次数限制（偷方）
	thiefToday := model.CountThiefStealsToday(tgId)
	if thiefToday >= int64(cfg.MaxStealPerUserPerDay) {
		farmSend(chatId, editMsgId, fmt.Sprintf("⏳ 今日偷菜次数已达上限（%d次）。明天再来！", cfg.MaxStealPerUserPerDay), backBtn, from)
		return
	}

	// 冷却
	now := time.Now().Unix()
	recentSteals, _ := model.CountRecentSteals(tgId, victimId, now-int64(cfg.StealCooldownSeconds))
	if recentSteals > 0 {
		cooldownMin := cfg.StealCooldownSeconds / 60
		farmSend(chatId, editMsgId, fmt.Sprintf("⏳ 冷却中！%d分钟内只能偷同一人一次。", cooldownMin), backBtn, from)
		return
	}

	// 获取可偷地块
	plots, err := model.GetStealablePlotsV2(victimId)
	if err != nil || len(plots) == 0 {
		farmSend(chatId, editMsgId, "❌ 该玩家没有可偷的成熟作物了。", backBtn, from)
		return
	}

	// 过滤保护期内的地块
	var stealable []*model.TgFarmPlot
	for _, p := range plots {
		if isPlotInProtection(p, cfg) {
			continue
		}
		stealable = append(stealable, p)
	}
	if len(stealable) == 0 {
		farmSend(chatId, editMsgId, "🛡️ 该玩家的作物仍在保护期内，暂时不能偷。", backBtn, from)
		return
	}

	target := stealable[rand.Intn(len(stealable))]
	crop := farmCropMap[target.CropType]
	cropName := "作物"
	cropEmoji := "🌿"
	unitPrice := 10000
	if crop != nil {
		cropName = crop.Name
		cropEmoji = crop.Emoji
		unitPrice = crop.UnitPrice
	}

	// 偷取 1 个单位
	stealUnits := 1
	marketPrice := applyMarket(unitPrice, "crop_"+crop.Key)
	stealValue := stealUnits * marketPrice

	_ = model.IncrementPlotStolenBy(target.Id, stealUnits)
	if cfg.EnableStealLog {
		_ = model.CreateFarmStealLog(&model.TgFarmStealLog{
			ThiefId:  tgId,
			VictimId: victimId,
			PlotId:   target.Id,
			Amount:   stealValue,
		})
	}
	_ = model.IncreaseUserQuota(user.Id, stealValue, true)
	model.AddFarmLog(tgId, "steal", stealValue, fmt.Sprintf("摘取%s%s额外收益×%d", cropEmoji, cropName, stealUnits))

	common.SysLog(fmt.Sprintf("TG Farm: user %s stole %s bonus from %s, +%d quota", tgId, cropName, victimId, stealValue))

	text := fmt.Sprintf("🕵️ 偷菜成功！\n\n你从 %s 的农场偷取了 %d个%s%s\n💰 获得 %s",
		maskTgId(victimId), stealUnits, cropEmoji, cropName, farmQuotaStr(stealValue))
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🕵️ 继续偷菜", CallbackData: "farm_steal"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 治疗 ==========

func showFarmTreatSelection(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	text := "💊 选择要治疗的地块：\n\n"
	var rows [][]TgInlineKeyboardButton
	hasEvent := false
	hasDrought := false
	for _, plot := range plots {
		if plot.Status == 3 {
			crop := farmCropMap[plot.CropType]
			cropName := "作物"
			cropEmoji := "🌿"
			if crop != nil {
				cropName = crop.Name
				cropEmoji = crop.Emoji
			}
			if plot.EventType == "drought" {
				hasDrought = true
				text += fmt.Sprintf("🏜️ %d号地 - %s 天灾干旱！（💧请去浇水救命）\n",
					plot.PlotIndex+1, cropName)
			} else {
				hasEvent = true
				evtLabel := farmEventLabel(plot.EventType)
				var needItem string
				for _, item := range farmItems {
					if item.Cures == plot.EventType {
						needItem = item.Emoji + item.Name
						break
					}
				}
				text += fmt.Sprintf("%s %d号地 - %s %s (需要%s)\n",
					cropEmoji, plot.PlotIndex+1, cropName, evtLabel, needItem)
				rows = append(rows, []TgInlineKeyboardButton{
					{Text: fmt.Sprintf("💊 治疗 %d号地", plot.PlotIndex+1),
						CallbackData: fmt.Sprintf("farm_tr_%d", plot.PlotIndex)},
				})
			}
		}
	}
	if !hasEvent && !hasDrought {
		text = "💊 没有需要治疗的地块。"
	}
	if hasDrought {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "💧 去浇水", CallbackData: "farm_water"},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🏪 去商店", CallbackData: "farm_shop"},
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func doFarmTreat(chatId int64, editMsgId int, tgId string, plotIdx int, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	var targetPlot *model.TgFarmPlot
	for _, p := range plots {
		if p.PlotIndex == plotIdx {
			targetPlot = p
			break
		}
	}
	if targetPlot == nil || targetPlot.Status != 3 {
		farmSend(chatId, editMsgId, "❌ 该地块不需要治疗", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	var cureItem *farmItemDef
	for i := range farmItems {
		if farmItems[i].Cures == targetPlot.EventType {
			cureItem = &farmItems[i]
			break
		}
	}
	if cureItem == nil {
		farmSend(chatId, editMsgId, "❌ 无法治疗此事件", nil, from)
		return
	}

	err = model.DecrementFarmItem(tgId, cureItem.Key)
	if err != nil {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 你没有 %s%s！请先到商店购买。",
			cureItem.Emoji, cureItem.Name), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	now := time.Now().Unix()
	downtime := now - targetPlot.EventAt
	targetPlot.PlantedAt += downtime
	targetPlot.Status = 1
	targetPlot.EventType = ""
	targetPlot.EventAt = 0
	_ = model.UpdateFarmPlot(targetPlot)

	crop := farmCropMap[targetPlot.CropType]
	cropName := "作物"
	if crop != nil {
		cropName = crop.Name
	}
	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 使用 %s%s 治疗成功！\n%s 恢复生长中。",
		cureItem.Emoji, cureItem.Name, cropName), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 施肥 ==========

func showFarmFertSelection(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	// 检查背包化肥
	items, _ := model.GetFarmItems(tgId)
	hasFert := false
	for _, item := range items {
		if item.ItemType == "fertilizer" && item.Quantity > 0 {
			hasFert = true
			break
		}
	}
	if !hasFert {
		farmSend(chatId, editMsgId, "❌ 你没有化肥！请先到商店购买。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	text := "🧴 选择要施肥的地块（生长中且未施肥）：\n\n"
	var rows [][]TgInlineKeyboardButton
	hasTarget := false
	for _, plot := range plots {
		if plot.Status == 1 && plot.Fertilized == 0 {
			crop := farmCropMap[plot.CropType]
			if crop == nil {
				continue
			}
			hasTarget = true
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("%s %d号地 - %s", crop.Emoji, plot.PlotIndex+1, crop.Name),
					CallbackData: fmt.Sprintf("farm_ff_%d", plot.PlotIndex)},
			})
		}
	}
	if !hasTarget {
		text += "没有可施肥的地块（需要生长中且未施肥）。"
	} else {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🧴 一键全部施肥", CallbackData: "farm_fertall"},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmFertilizeAll(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	// 统计需要施肥的地块数
	var targets []*model.TgFarmPlot
	for _, plot := range plots {
		if plot.Status == 1 && plot.Fertilized == 0 {
			targets = append(targets, plot)
		}
	}
	if len(targets) == 0 {
		farmSend(chatId, editMsgId, "🧴 没有可施肥的地块。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 检查化肥数量
	items, _ := model.GetFarmItems(tgId)
	fertQty := 0
	for _, item := range items {
		if item.ItemType == "fertilizer" {
			fertQty = item.Quantity
			break
		}
	}
	if fertQty <= 0 {
		farmSend(chatId, editMsgId, "❌ 你没有化肥！请先到商店购买。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	fertilizedCount := 0
	details := ""
	for _, plot := range targets {
		if fertQty <= 0 {
			details += fmt.Sprintf("\n  ❌ 化肥不足，剩余地块未施肥")
			break
		}
		crop := farmCropMap[plot.CropType]
		if crop == nil {
			continue
		}
		if err := model.DecrementFarmItem(tgId, "fertilizer"); err != nil {
			break
		}
		plot.Fertilized = 1
		_ = model.UpdateFarmPlot(plot)
		fertilizedCount++
		fertQty--
		details += fmt.Sprintf("\n  %s %d号地 %s ✅", crop.Emoji, plot.PlotIndex+1, crop.Name)
	}

	text := fmt.Sprintf("🧴 一键施肥完成！共施肥 %d 块地（收获产量+50%%）\n%s", fertilizedCount, details)
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func doFarmFertilize(chatId int64, editMsgId int, tgId string, plotIdx int, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	var target *model.TgFarmPlot
	for _, plot := range plots {
		if plot.PlotIndex == plotIdx {
			target = plot
			break
		}
	}
	if target == nil || target.Status != 1 || target.Fertilized == 1 {
		farmSend(chatId, editMsgId, "❌ 该地块不可施肥。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 消耗化肥
	if err := model.DecrementFarmItem(tgId, "fertilizer"); err != nil {
		farmSend(chatId, editMsgId, "❌ 化肥不足！请先到商店购买。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 标记已施肥
	target.Fertilized = 1
	_ = model.UpdateFarmPlot(target)

	crop := farmCropMap[target.CropType]
	cropName := "作物"
	if crop != nil {
		cropName = crop.Emoji + crop.Name
	}

	common.SysLog(fmt.Sprintf("TG Farm: user %s fertilized plot %d (%s)", tgId, plotIdx, cropName))

	farmSend(chatId, editMsgId, fmt.Sprintf("🧴 施肥成功！\n\n%d号地 %s 已施肥，收获时产量+50%%！", plotIdx+1, cropName), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🧴 继续施肥", CallbackData: "farm_fert"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 购买土地 ==========

func doFarmBuyLand(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	plotCount, err := model.GetFarmPlotCount(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	if int(plotCount) >= model.FarmMaxPlots {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 你已拥有 %d 块土地，已达上限！", model.FarmMaxPlots), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	price := common.TgBotFarmPlotPrice
	if user.Quota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！\n\n土地价格：%s\n你的余额：%s",
			farmQuotaStr(price), farmQuotaStr(user.Quota)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 扣费
	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败，请稍后再试。", nil, from)
		return
	}

	// 创建新地块
	newIdx := int(plotCount)
	err = model.CreateNewFarmPlot(tgId, newIdx)
	if err != nil {
		// 回滚扣费
		_ = model.IncreaseUserQuota(user.Id, price, true)
		farmSend(chatId, editMsgId, "❌ 创建地块失败，已退款。", nil, from)
		return
	}

	model.AddFarmLog(tgId, "buy_plot", -price, fmt.Sprintf("购买%d号地", newIdx+1))
	common.SysLog(fmt.Sprintf("TG Farm: user %s bought plot %d for %d quota", tgId, newIdx+1, price))

	farmSend(chatId, editMsgId, fmt.Sprintf("🏗️ 购买成功！\n\n你获得了 %d号地！\n💰 花费 %s\n📊 当前土地 %d/%d 块",
		newIdx+1, farmQuotaStr(price), newIdx+1, model.FarmMaxPlots), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 浇水 ==========

func showFarmWaterSelection(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	text := "💧 选择要浇水的地块：\n\n"
	var rows [][]TgInlineKeyboardButton
	hasTarget := false
	for _, plot := range plots {
		needsWater := plot.Status == 1 || plot.Status == 4 ||
			(plot.Status == 3 && plot.EventType == "drought")
		if needsWater {
			crop := farmCropMap[plot.CropType]
			if crop == nil {
				continue
			}
			hasTarget = true
			statusLabel := "生长中"
			if plot.Status == 4 {
				statusLabel = "🥀枯萎中"
			} else if plot.Status == 3 && plot.EventType == "drought" {
				statusLabel = "🏜️天灾干旱"
			}
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("%s %d号地 - %s (%s)", crop.Emoji, plot.PlotIndex+1, crop.Name, statusLabel),
					CallbackData: fmt.Sprintf("farm_ww_%d", plot.PlotIndex)},
			})
		}
	}
	if !hasTarget {
		text += "没有需要浇水的地块。"
	} else {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "💧 一键全部浇水", CallbackData: "farm_waterall"},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmWaterAll(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	wateredCount := 0
	details := ""
	for _, plot := range plots {
		needsWater := plot.Status == 1 || plot.Status == 4 ||
			(plot.Status == 3 && plot.EventType == "drought")
		if !needsWater {
			continue
		}
		crop := farmCropMap[plot.CropType]
		if crop == nil {
			continue
		}

		wasWilting := plot.Status == 4
		wasDrought := plot.Status == 3 && plot.EventType == "drought"

		if wasWilting {
			now := time.Now().Unix()
			waterInterval := int64(common.TgBotFarmWaterInterval)
			wiltStart := plot.LastWateredAt + waterInterval
			downtime := now - wiltStart
			plot.PlantedAt += downtime
			plot.Status = 1
			_ = model.UpdateFarmPlot(plot)
		}
		if wasDrought {
			now := time.Now().Unix()
			downtime := now - plot.EventAt
			plot.PlantedAt += downtime
			plot.Status = 1
			plot.EventType = ""
			plot.EventAt = 0
			_ = model.UpdateFarmPlot(plot)
		}

		_ = model.WaterFarmPlot(plot.Id)
		wateredCount++

		tag := ""
		if wasDrought {
			tag = " (干旱已解除)"
		} else if wasWilting {
			tag = " (枯萎已恢复)"
		}
		details += fmt.Sprintf("\n  %s %d号地 %s%s", crop.Emoji, plot.PlotIndex+1, crop.Name, tag)
	}

	if wateredCount == 0 {
		farmSend(chatId, editMsgId, "💧 没有需要浇水的地块。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	model.AddFarmLog(tgId, "water", 0, fmt.Sprintf("一键浇水%d块地", wateredCount))

	text := fmt.Sprintf("💧 一键浇水完成！共浇水 %d 块地\n%s", wateredCount, details)
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func doFarmWater(chatId int64, editMsgId int, tgId string, plotIdx int, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	var target *model.TgFarmPlot
	for _, plot := range plots {
		if plot.PlotIndex == plotIdx {
			target = plot
			break
		}
	}
	if target == nil {
		farmSend(chatId, editMsgId, "❌ 该地块不需要浇水。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}
	canWater := target.Status == 1 || target.Status == 4 ||
		(target.Status == 3 && target.EventType == "drought")
	if !canWater {
		farmSend(chatId, editMsgId, "❌ 该地块不需要浇水。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	wasWilting := target.Status == 4
	wasDrought := target.Status == 3 && target.EventType == "drought"

	// 如果是枯萎状态，恢复为生长中，补偿枯萎期间的时间
	if wasWilting {
		now := time.Now().Unix()
		waterInterval := int64(common.TgBotFarmWaterInterval)
		wiltStart := target.LastWateredAt + waterInterval
		downtime := now - wiltStart
		target.PlantedAt += downtime
		target.Status = 1
		_ = model.UpdateFarmPlot(target)
	}

	// 如果是天灾干旱，恢复为生长中，补偿干旱期间的时间
	if wasDrought {
		now := time.Now().Unix()
		downtime := now - target.EventAt
		target.PlantedAt += downtime
		target.Status = 1
		target.EventType = ""
		target.EventAt = 0
		_ = model.UpdateFarmPlot(target)
	}

	_ = model.WaterFarmPlot(target.Id)

	crop := farmCropMap[target.CropType]
	cropName := "作物"
	if crop != nil {
		cropName = crop.Emoji + crop.Name
	}

	model.AddFarmLog(tgId, "water", 0, fmt.Sprintf("浇水%d号地%s", plotIdx+1, cropName))

	msg := fmt.Sprintf("💧 浇水成功！\n\n%d号地 %s", plotIdx+1, cropName)
	if wasDrought {
		msg += " 天灾干旱已解除，恢复生长！"
	} else if wasWilting {
		msg += " 已从枯萎中恢复生长！"
	} else {
		msg += " 已浇水。"
	}

	farmSend(chatId, editMsgId, msg, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "💧 继续浇水", CallbackData: "farm_water"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 狗狗系统 ==========

func showFarmDog(chatId int64, editMsgId int, tgId string, from *TgUser) {
	dog, err := model.GetFarmDog(tgId)
	if err != nil {
		// 没有狗
		text := "🐕 你还没有狗狗！\n\n" +
			fmt.Sprintf("在商店购买一只小狗（%s），养大后可以帮你看门拦截偷菜者！\n\n", farmQuotaStr(common.TgBotFarmDogPrice)) +
			fmt.Sprintf("🐶 幼犬需要 %d 小时长大为成犬\n", common.TgBotFarmDogGrowHours) +
			"🦴 记得定期喂狗粮，饿坏了就不看门了\n" +
			fmt.Sprintf("🛡️ 成犬看门拦截率：%d%%", common.TgBotFarmDogGuardRate)
		farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: fmt.Sprintf("🐶 购买小狗 (%s)", farmQuotaStr(common.TgBotFarmDogPrice)), CallbackData: "farm_buydog"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	model.UpdateDogHunger(dog)

	levelStr := "🐶 幼犬"
	statusStr := "成长中"
	if dog.Level == 2 {
		levelStr = "🐕 成犬"
		if dog.Hunger > 0 {
			statusStr = "✅ 看门中"
		} else {
			statusStr = "❌ 饿坏了，无法看门"
		}
	} else {
		if dog.Hunger == 0 {
			statusStr = "❌ 饿坏了"
		} else {
			now := time.Now().Unix()
			hoursLeft := int64(common.TgBotFarmDogGrowHours) - (now-dog.CreatedAt)/3600
			if hoursLeft < 0 {
				hoursLeft = 0
			}
			statusStr = fmt.Sprintf("⏳ 还需 %d 小时长大", hoursLeft)
		}
	}

	text := fmt.Sprintf("🐕 我的狗狗\n\n"+
		"名字：%s\n"+
		"等级：%s\n"+
		"状态：%s\n"+
		"饱食度：%d%%\n\n"+
		"🛡️ 看门拦截率：%d%%\n"+
		"🦴 狗粮价格：%s",
		dog.Name, levelStr, statusStr, dog.Hunger,
		common.TgBotFarmDogGuardRate, farmQuotaStr(common.TgBotFarmDogFoodPrice))

	var rows [][]TgInlineKeyboardButton
	if dog.Hunger < 100 {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🦴 喂狗粮", CallbackData: "farm_feeddog"},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🏪 商店买狗粮", CallbackData: "farm_shop"},
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmBuyDog(chatId int64, editMsgId int, tgId string, from *TgUser) {
	// 检查是否已有狗
	_, err := model.GetFarmDog(tgId)
	if err == nil {
		farmSend(chatId, editMsgId, "❌ 你已经有一只狗了！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🐕 查看狗狗", CallbackData: "farm_dog"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	price := common.TgBotFarmDogPrice
	if user.Quota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！需要 %s", farmQuotaStr(price)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回商店", CallbackData: "farm_shop"}},
			},
		}, from)
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败", nil, from)
		return
	}

	// 生成随机狗名
	dogNames := []string{"旺财", "小黑", "大黄", "豆豆", "球球", "毛毛", "Lucky", "小白", "花花", "阿福"}
	dogName := dogNames[rand.Intn(len(dogNames))]

	now := time.Now().Unix()
	dog := &model.TgFarmDog{
		TelegramId: tgId,
		Name:       dogName,
		Level:      1,
		Hunger:     100,
		LastFedAt:  now,
	}
	err = model.CreateFarmDog(dog)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		farmSend(chatId, editMsgId, "❌ 购买失败，已退款。", nil, from)
		return
	}

	model.AddFarmLog(tgId, "buy_dog", -price, fmt.Sprintf("购买看门狗「%s」", dogName))
	common.SysLog(fmt.Sprintf("TG Farm: user %s bought dog '%s' for %d quota", tgId, dogName, price))

	farmSend(chatId, editMsgId, fmt.Sprintf("🐶 恭喜！你获得了一只小狗「%s」！\n\n"+
		"花费：%s\n"+
		"等级：幼犬\n"+
		"⏳ %d 小时后长大为成犬，即可看门拦截偷菜者\n"+
		"🦴 记得定期喂狗粮哦！",
		dogName, farmQuotaStr(price), common.TgBotFarmDogGrowHours), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🐕 查看狗狗", CallbackData: "farm_dog"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func doFarmFeedDog(chatId int64, editMsgId int, tgId string, from *TgUser) {
	dog, err := model.GetFarmDog(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 你还没有狗狗！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	model.UpdateDogHunger(dog)

	if dog.Hunger >= 100 {
		farmSend(chatId, editMsgId, "❌ 狗狗现在不饿，不需要喂食！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🐕 查看狗狗", CallbackData: "farm_dog"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 消耗狗粮
	err = model.DecrementFarmItem(tgId, "dogfood")
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 你没有狗粮！请先到商店购买。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	_ = model.FeedFarmDog(dog.Id)

	farmSend(chatId, editMsgId, fmt.Sprintf("🦴 喂食成功！「%s」吃饱了，饱食度恢复到 100%%！", dog.Name), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🐕 查看狗狗", CallbackData: "farm_dog"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 泥土升级 ==========

func showFarmSoilUpgrade(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	text := "🌱 泥土升级\n\n"
	text += fmt.Sprintf("📌 每级加速生长 %d%%\n", common.TgBotFarmSoilSpeedBonus)
	text += fmt.Sprintf("📌 最高等级 Lv.%d\n\n", common.TgBotFarmSoilMaxLevel)
	text += "升级价格：\n"
	prices := map[int]int{
		2: common.TgBotFarmSoilUpgradePrice2,
		3: common.TgBotFarmSoilUpgradePrice3,
		4: common.TgBotFarmSoilUpgradePrice4,
		5: common.TgBotFarmSoilUpgradePrice5,
	}
	for lvl := 2; lvl <= common.TgBotFarmSoilMaxLevel && lvl <= 5; lvl++ {
		text += fmt.Sprintf("  Lv.%d → %s (加速 %d%%)\n", lvl, farmQuotaStr(prices[lvl]), common.TgBotFarmSoilSpeedBonus*(lvl-1))
	}
	text += "\n选择要升级的地块：\n"

	var rows [][]TgInlineKeyboardButton
	hasUpgradable := false
	for _, plot := range plots {
		sl := plot.SoilLevel
		if sl < 1 {
			sl = 1
		}
		if sl >= common.TgBotFarmSoilMaxLevel {
			continue
		}
		hasUpgradable = true
		nextLvl := sl + 1
		price := 0
		if p, ok := prices[nextLvl]; ok {
			price = p
		}
		label := fmt.Sprintf("%d号地 Lv.%d→%d (%s)", plot.PlotIndex+1, sl, nextLvl, farmQuotaStr(price))
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: label, CallbackData: fmt.Sprintf("farm_su_%d", plot.PlotIndex)},
		})
	}
	if !hasUpgradable {
		text += "所有地块已达最高等级！🎉\n"
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmSoilUpgrade(chatId int64, editMsgId int, tgId string, plotIdx int, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	var target *model.TgFarmPlot
	for _, plot := range plots {
		if plot.PlotIndex == plotIdx {
			target = plot
			break
		}
	}
	if target == nil {
		farmSend(chatId, editMsgId, "❌ 地块不存在", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_soil"}},
			},
		}, from)
		return
	}

	currentLevel := target.SoilLevel
	if currentLevel < 1 {
		currentLevel = 1
	}
	nextLevel := currentLevel + 1
	if nextLevel > common.TgBotFarmSoilMaxLevel {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ %d号地泥土已达最高等级 Lv.%d！", plotIdx+1, common.TgBotFarmSoilMaxLevel), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_soil"}},
			},
		}, from)
		return
	}

	var price int
	switch nextLevel {
	case 2:
		price = common.TgBotFarmSoilUpgradePrice2
	case 3:
		price = common.TgBotFarmSoilUpgradePrice3
	case 4:
		price = common.TgBotFarmSoilUpgradePrice4
	case 5:
		price = common.TgBotFarmSoilUpgradePrice5
	default:
		farmSend(chatId, editMsgId, "❌ 不支持的升级等级", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_soil"}},
			},
		}, from)
		return
	}

	if user.Quota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！\n\n升级到 Lv.%d 需要：%s\n你的余额：%s",
			nextLevel, farmQuotaStr(price), farmQuotaStr(user.Quota)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_soil"}},
			},
		}, from)
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败，请稍后再试。", nil, from)
		return
	}

	err = model.UpgradeFarmPlotSoil(target.Id, nextLevel)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		farmSend(chatId, editMsgId, "❌ 升级失败，已退款。", nil, from)
		return
	}

	speedBonus := common.TgBotFarmSoilSpeedBonus * (nextLevel - 1)
	model.AddFarmLog(tgId, "upgrade_soil", -price, fmt.Sprintf("%d号地泥土升级Lv.%d", plotIdx+1, nextLevel))
	common.SysLog(fmt.Sprintf("TG Farm: user %s upgraded plot %d soil to Lv.%d for %d quota", tgId, plotIdx+1, nextLevel, price))

	farmSend(chatId, editMsgId, fmt.Sprintf("🌱 升级成功！\n\n%d号地泥土升级到 Lv.%d\n⚡ 生长加速 %d%%\n💰 花费 %s",
		plotIdx+1, nextLevel, speedBonus, farmQuotaStr(price)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🌱 继续升级", CallbackData: "farm_soil"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 辅助函数 ==========

func farmEventLabel(eventType string) string {
	switch eventType {
	case "bugs":
		return "虫害🐛"
	case "drought":
		return "天灾干旱🏜️"
	}
	return "未知"
}

func maskTgId(tgId string) string {
	if len(tgId) > 6 {
		return tgId[:3] + "***" + tgId[len(tgId)-3:]
	}
	return "***"
}

// ========== 消费记录 ==========

func showFarmLogs(chatId int64, editMsgId int, tgId string, from *TgUser) {
	logs, total, err := model.GetFarmLogs(tgId, 15, 0)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 获取记录失败", nil, from)
		return
	}

	actionLabels := map[string]string{
		"plant": "种植", "harvest": "收获", "shop": "商店", "steal": "偷菜",
		"buy_plot": "购地", "buy_dog": "买狗", "upgrade_soil": "升级",
		"ranch_buy": "买动物", "ranch_feed": "喂食", "ranch_water": "喂水",
		"ranch_sell": "出售", "ranch_clean": "清粪",
		"fish": "钓鱼", "fish_sell": "卖鱼",
		"craft": "加工", "craft_sell": "收取",
		"task": "任务", "achieve": "成就",
		"levelup": "升级",
		"loan": "贷款", "repay": "还款",
		"mortgage_default": "抵押违约",
		"warehouse_sell": "仓库出售",
	}

	text := fmt.Sprintf("📋 消费记录（最近15条，共%d条）\n\n", total)
	if len(logs) == 0 {
		text += "暂无记录\n"
	}
	for _, l := range logs {
		label := actionLabels[l.Action]
		if label == "" {
			label = l.Action
		}
		sign := "+"
		if l.Amount < 0 {
			sign = ""
		}
		ts := time.Unix(l.CreatedAt, 0)
		text += fmt.Sprintf("%s %s%s %s · %s\n",
			label, sign, farmQuotaStr(l.Amount), l.Detail,
			ts.Format("01-02 15:04"))
	}

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 钓鱼 ==========

// --- 风控：内存滑动窗口 ---
var fishRiskTimestamps sync.Map // map[string][]int64

func recordFishTimestamp(tgId string) {
	now := time.Now().Unix()
	cutoff := now - 300 // 5分钟窗口
	val, _ := fishRiskTimestamps.Load(tgId)
	var timestamps []int64
	if val != nil {
		timestamps = val.([]int64)
	}
	var filtered []int64
	for _, ts := range timestamps {
		if ts > cutoff {
			filtered = append(filtered, ts)
		}
	}
	filtered = append(filtered, now)
	fishRiskTimestamps.Store(tgId, filtered)
}

func checkFishRisk(tgId string) bool {
	if !common.TgBotFishRiskEnabled {
		return false
	}
	val, ok := fishRiskTimestamps.Load(tgId)
	if !ok {
		return false
	}
	timestamps := val.([]int64)
	if len(timestamps) < 10 {
		return false
	}
	recent := timestamps[len(timestamps)-10:]
	var intervals []float64
	for i := 1; i < len(recent); i++ {
		intervals = append(intervals, float64(recent[i]-recent[i-1]))
	}
	var sum float64
	for _, v := range intervals {
		sum += v
	}
	mean := sum / float64(len(intervals))
	var sqDiffSum float64
	for _, v := range intervals {
		diff := v - mean
		sqDiffSum += diff * diff
	}
	std := math.Sqrt(sqDiffSum / float64(len(intervals)))
	return std < 0.5 && mean <= float64(common.TgBotFishActionCD)+1
}

// --- 带疲劳衰减的随机钓鱼 ---
func randomFishFromRarityPool(tgId string, rarities map[string]bool) *fishDef {
	dailyCount := model.GetFishDailyCount(tgId)
	fatigueActive := common.TgBotFishFatigueEnabled && dailyCount >= common.TgBotFishFatigueThreshold

	adjustedTotal := 0
	type aw struct {
		fish   *fishDef
		weight int
	}
	var adjusted []aw
	for i := range fishTypes {
		if !rarities[fishTypes[i].Rarity] {
			continue
		}
		w := getFishWeight(i)
		if fatigueActive && (fishTypes[i].Rarity == "稀有" || fishTypes[i].Rarity == "史诗" || fishTypes[i].Rarity == "传说") {
			w = w * (100 - common.TgBotFishFatigueDecay) / 100
			if w < 0 {
				w = 0
			}
		}
		adjusted = append(adjusted, aw{&fishTypes[i], w})
		adjustedTotal += w
	}
	if adjustedTotal <= 0 {
		return nil
	}
	r := rand.Intn(adjustedTotal)
	cumulative := 0
	for _, a := range adjusted {
		cumulative += a.weight
		if r < cumulative {
			return a.fish
		}
	}
	return adjusted[len(adjusted)-1].fish
}

func randomFishWithFatigue(tgId string, premiumBait bool) *fishDef {
	dailyCount := model.GetFishDailyCount(tgId)
	fatigueActive := common.TgBotFishFatigueEnabled && dailyCount >= common.TgBotFishFatigueThreshold

	adjustedTotal := common.TgBotFishNothingWeight
	type aw struct {
		fish   *fishDef
		weight int
	}
	if premiumBait && rand.Intn(100) < 5 {
		if rareFish := randomFishFromRarityPool(tgId, map[string]bool{"史诗": true, "传说": true}); rareFish != nil {
			return rareFish
		}
	}
	var adjusted []aw
	for i := range fishTypes {
		w := getFishWeight(i)
		if fatigueActive && (fishTypes[i].Rarity == "稀有" || fishTypes[i].Rarity == "史诗" || fishTypes[i].Rarity == "传说") {
			w = w * (100 - common.TgBotFishFatigueDecay) / 100
			if w < 0 {
				w = 0
			}
		}
		adjusted = append(adjusted, aw{&fishTypes[i], w})
		adjustedTotal += w
	}
	if adjustedTotal <= 0 {
		return nil
	}
	r := rand.Intn(adjustedTotal)
	cumulative := common.TgBotFishNothingWeight
	if r < cumulative {
		return nil
	}
	for _, a := range adjusted {
		cumulative += a.weight
		if r < cumulative {
			return a.fish
		}
	}
	return adjusted[len(adjusted)-1].fish
}

// fishAdjustedTotal 计算疲劳调整后的总权重
func fishAdjustedTotal(fatigueActive bool) int {
	total := common.TgBotFishNothingWeight
	for i, ft := range fishTypes {
		w := getFishWeight(i)
		if fatigueActive && (ft.Rarity == "稀有" || ft.Rarity == "史诗" || ft.Rarity == "传说") {
			w = w * (100 - common.TgBotFishFatigueDecay) / 100
			if w < 0 {
				w = 0
			}
		}
		total += w
	}
	return total
}

func showFarmFish(chatId int64, editMsgId int, tgId string, from *TgUser) {
	// 鱼饵数量
	items, _ := model.GetFarmItems(tgId)
	baitCount := 0
	premiumBaitCount := 0
	for _, item := range items {
		if item.ItemType == "fishbait" {
			baitCount = item.Quantity
		} else if item.ItemType == "premiumfishbait" {
			premiumBaitCount = item.Quantity
		}
	}

	// 体力
	stamina, recoverIn := model.GetFishStamina(tgId)
	staminaMax := common.TgBotFishStaminaMax

	// 每日统计
	dailyCount := model.GetFishDailyCount(tgId)
	dailyIncome := model.GetFishDailyIncome(tgId)
	dailyMaxIncome := common.TgBotFishDailyMaxIncome

	// 疲劳状态
	fatigueActive := common.TgBotFishFatigueEnabled && dailyCount >= common.TgBotFishFatigueThreshold

	// 鱼仓库
	fishItems, _ := model.GetFishItems(tgId)
	totalValue := 0

	text := "🎣 钓鱼\n\n"

	// 体力条
	barLen := 10
	filled := 0
	if staminaMax > 0 {
		filled = stamina * barLen / staminaMax
	}
	if filled > barLen {
		filled = barLen
	}
	bar := ""
	for i := 0; i < barLen; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	text += fmt.Sprintf("⚡ 体力: %d/%d [%s]\n", stamina, staminaMax, bar)
	if stamina < staminaMax && recoverIn > 0 {
		text += fmt.Sprintf("🔄 下次恢复: %d秒后 (+%d)\n", recoverIn, common.TgBotFishStaminaRecoverAmount)
	}

	// 疲劳
	if fatigueActive {
		text += fmt.Sprintf("😰 疲劳中！稀有鱼概率 -%d%%\n", common.TgBotFishFatigueDecay)
	} else if common.TgBotFishFatigueEnabled {
		text += fmt.Sprintf("😊 精力充沛（%d/%d次后疲劳）\n", dailyCount, common.TgBotFishFatigueThreshold)
	}

	// 每日进度
	if common.TgBotFishIncomeCapEnabled {
		incomeCap := common.TgBotFishDailyIncomeCap
		overCap := dailyIncome >= incomeCap
		text += fmt.Sprintf("💰 今日收益: %s / %s\n", farmQuotaStr(dailyIncome), farmQuotaStr(incomeCap))
		if overCap {
			text += "🚫 今日收益已达上限\n"
		} else {
			remaining := incomeCap - dailyIncome
			text += fmt.Sprintf("📊 距离上限还差: %s\n", farmQuotaStr(remaining))
		}
		text += fmt.Sprintf("📊 今日次数: %d\n", dailyCount)
	} else {
		text += fmt.Sprintf("📊 今日次数: %d 💰 %s/%s\n", dailyCount, farmQuotaStr(dailyIncome), farmQuotaStr(dailyMaxIncome))
	}
	text += fmt.Sprintf("🪱 普通鱼饵: %d个\n", baitCount)
	text += fmt.Sprintf("✨ 高级鱼饵: %d个\n", premiumBaitCount)
	if premiumBaitCount > 0 {
		text += "✨ 钓鱼时会优先消耗高级鱼饵，史诗/传说额外概率+5%\n"
	}
	totalBaitCount := baitCount + premiumBaitCount

	// 短CD
	lastFish := model.GetLastFishTime(tgId)
	now := time.Now().Unix()
	cd := int64(common.TgBotFishActionCD)
	if cd < 5 {
		cd = 5
	}
	cdRemain := lastFish + cd - now
	if cdRemain > 0 {
		text += fmt.Sprintf("⏱️ 操作冷却: %d秒\n", cdRemain)
	}

	text += "\n📦 鱼仓库:\n"
	if len(fishItems) == 0 {
		text += "  (空)\n"
	} else {
		for _, fi := range fishItems {
			fishKey := fi.ItemType[5:]
			fd := fishTypeMap[fishKey]
			if fd != nil {
				mPrice := applyMarket(fd.SellPrice, "fish_"+fishKey)
				val := mPrice * fi.Quantity
				totalValue += val
				mPct := getMarketMultiplier("fish_" + fishKey)
				text += fmt.Sprintf("  %s %s ×%d [%s] %s(%d%%)\n", fd.Emoji, fd.Name, fi.Quantity, fd.Rarity, farmQuotaStr(val), mPct)
			}
		}
		text += fmt.Sprintf("\n💰 总价值: %s\n", farmQuotaStr(totalValue))
	}

	// 市场倒计时
	ensureMarketFresh()
	nextRefresh := getMarketNextRefresh()
	if nextRefresh > 0 {
		text += fmt.Sprintf("\n📈 市场%dh后刷新\n", nextRefresh/3600+1)
	}

	// 鱼种概率（疲劳调整后）
	text += "\n📊 鱼种概率:\n"
	adjTotal := fishAdjustedTotal(fatigueActive)
	for i, ft := range fishTypes {
		w := getFishWeight(i)
		if fatigueActive && (ft.Rarity == "稀有" || ft.Rarity == "史诗" || ft.Rarity == "传说") {
			w = w * (100 - common.TgBotFishFatigueDecay) / 100
		}
		mPrice := applyMarket(ft.SellPrice, "fish_"+ft.Key)
		mPct := getMarketMultiplier("fish_" + ft.Key)
		pct := 0
		if adjTotal > 0 {
			pct = w * 100 / adjTotal
		}
		tag := ""
		if fatigueActive && (ft.Rarity == "稀有" || ft.Rarity == "史诗" || ft.Rarity == "传说") {
			tag = " ⬇"
		}
		text += fmt.Sprintf("  %s %s [%s] %d%%%s %s(%d%%)\n", ft.Emoji, ft.Name, ft.Rarity, pct, tag, farmQuotaStr(mPrice), mPct)
	}
	nothingPct := 0
	if adjTotal > 0 {
		nothingPct = common.TgBotFishNothingWeight * 100 / adjTotal
	}
	text += fmt.Sprintf("  🗑️ 空军 %d%%\n", nothingPct)

	var rows [][]TgInlineKeyboardButton
	// 按钮文案优先级
	btnText := "🎣 开始钓鱼"
	if common.TgBotFishIncomeCapEnabled {
		overCap := dailyIncome >= common.TgBotFishDailyIncomeCap
		if overCap {
			btnText = "🚫 今日收益已达上限"
		} else if cdRemain > 0 {
			btnText = fmt.Sprintf("⏱️ 冷却中(%ds)", cdRemain)
		} else if totalBaitCount <= 0 {
			btnText = "🪱 缺少鱼饵"
		}
	} else {
		if dailyIncome >= common.TgBotFishDailyMaxIncome {
			btnText = "🚫 今日收益已达上限"
		} else if cdRemain > 0 {
			btnText = fmt.Sprintf("⏱️ 冷却中(%ds)", cdRemain)
		} else if totalBaitCount <= 0 {
			btnText = "🪱 缺少鱼饵"
		}
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: btnText, CallbackData: "farm_dofish"},
	})
	if totalValue > 0 {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("💰 出售全部 (%s)", farmQuotaStr(totalValue)), CallbackData: "farm_sellfish"},
		})
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "📦 存入仓库", CallbackData: "farm_storefish"},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmFish(chatId int64, editMsgId int, tgId string, from *TgUser) {
	now := time.Now().Unix()

	// 1. 每日限制检查
	dailyIncome := model.GetFishDailyIncome(tgId)

	if common.TgBotFishIncomeCapEnabled {
		// 收益CAP模型：保留每日收益硬限制
		if dailyIncome >= common.TgBotFishDailyIncomeCap {
			farmSend(chatId, editMsgId, "🚫 今日钓鱼收益已达上限，明天再来吧！", &TgInlineKeyboardMarkup{
				InlineKeyboard: [][]TgInlineKeyboardButton{
					{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
				},
			}, from)
			return
		}
	} else {
		// 旧模型兼容：仅保留每日收益限制
		if dailyIncome >= common.TgBotFishDailyMaxIncome {
			farmSend(chatId, editMsgId, "🚫 今日钓鱼收入已达上限，明天再来吧！", &TgInlineKeyboardMarkup{
				InlineKeyboard: [][]TgInlineKeyboardButton{
					{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
				},
			}, from)
			return
		}
	}

	// 2. 短CD检查
	lastFish := model.GetLastFishTime(tgId)
	cd := int64(common.TgBotFishActionCD)
	if cd < 5 {
		cd = 5
	}
	if now < lastFish+cd {
		remain := lastFish + cd - now
		farmSend(chatId, editMsgId, fmt.Sprintf("⏱️ 操作太快，请等待 %d 秒", remain), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
			},
		}, from)
		return
	}

	// 3. 鱼饵检查，优先消耗高级鱼饵
	baitKey := "fishbait"
	baitCost := common.TgBotFishBaitPrice
	baitLabel := "鱼饵"
	if qty, premiumErr := model.GetFarmItemQuantity(tgId, "premiumfishbait"); premiumErr == nil && qty > 0 {
		baitKey = "premiumfishbait"
		baitCost = common.TgBotFishPremiumBaitPrice
		baitLabel = "高级鱼饵"
	}
	err := model.DecrementFarmItem(tgId, baitKey)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 没有鱼饵！请先到商店购买🪱鱼饵", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"}},
				{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
			},
		}, from)
		return
	}

	// 4. 记录CD
	model.SetLastFishTime(tgId, now)

	// 5. 风控检测
	recordFishTimestamp(tgId)
	if checkFishRisk(tgId) {
		model.AddFarmLog(tgId, "fish_risk", 0, "钓鱼行为异常：操作间隔高度一致")
	}

	// 6. 随机钓鱼（含疲劳衰减，兼容旧模型）
	fish := randomFishWithFatigue(tgId, baitKey == "premiumfishbait")

	// 7. 增加每日计数（仅用于统计/任务/疲劳）
	model.IncrFishDailyCount(tgId)

	if fish == nil {
		// 空军
		model.AddFarmLog(tgId, "fish", -baitCost, fmt.Sprintf("钓鱼空军[%s]", baitLabel))
		farmSend(chatId, editMsgId, fmt.Sprintf("🎣 甩竿...\n\n🗑️ 空军！什么都没钓到...\n消耗了1个%s", baitLabel), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🎣 再钓一次", CallbackData: "farm_dofish"}},
				{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
			},
		}, from)
		return
	}

	// 8. 计算本次收益。
	// If the user is still allowed to cast, the current catch always counts in full.
	fishValue := applyMarket(fish.SellPrice, "fish_"+fish.Key)
	effectiveValue := fishValue

	_ = model.IncrementFarmItem(tgId, "fish_"+fish.Key, 1)
	model.RecordCollection(tgId, "fish", fish.Key, 1)
	model.IncrFishDailyIncome(tgId, effectiveValue)
	model.AddFarmLog(tgId, "fish", 0, fmt.Sprintf("钓到%s%s[%s]", fish.Emoji, fish.Name, fish.Rarity))

	rarityMsg := ""
	if fish.Rarity == "稀有" {
		rarityMsg = "🎉 不错！"
	} else if fish.Rarity == "史诗" {
		rarityMsg = "🎊 太棒了！！"
	} else if fish.Rarity == "传说" {
		rarityMsg = "🏆🎊 传说级！！！"
	}

	capReachedAfterCatch := common.TgBotFishIncomeCapEnabled && dailyIncome < common.TgBotFishDailyIncomeCap &&
		model.GetFishDailyIncome(tgId) >= common.TgBotFishDailyIncomeCap

	baitNotice := ""
	if baitKey == "premiumfishbait" {
		baitNotice = "\n✨ 使用了高级鱼饵：史诗/传说额外概率+5%"
	}
	text := fmt.Sprintf("🎣 甩竿...\n\n%s 钓到了 %s %s！\n品质: [%s]\n价值: %s\n%s%s",
		rarityMsg, fish.Emoji, fish.Name, fish.Rarity, farmQuotaStr(effectiveValue), rarityMsg, baitNotice)
	if capReachedAfterCatch {
		text += "\n\n⛔ 今日钓鱼收益已满，明天再来吧"
	}

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🎣 再钓一次", CallbackData: "farm_dofish"}},
			{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
		},
	}, from)
}

func doFarmSellFish(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 用户不存在", nil, from)
		return
	}

	fishItems, _ := model.GetFishItems(tgId)
	if len(fishItems) == 0 {
		farmSend(chatId, editMsgId, "❌ 鱼仓库为空", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
			},
		}, from)
		return
	}

	totalValue := 0
	totalCount := 0
	for _, fi := range fishItems {
		fishKey := fi.ItemType[5:]
		fd := fishTypeMap[fishKey]
		if fd != nil {
			totalValue += applyMarket(fd.SellPrice, "fish_"+fishKey) * fi.Quantity
			totalCount += fi.Quantity
		}
	}

	_, _ = model.SellAllFish(tgId)
	_ = model.IncreaseUserQuota(user.Id, totalValue, true)
	model.AddFarmLog(tgId, "fish_sell", totalValue, fmt.Sprintf("出售%d条鱼", totalCount))

	farmSend(chatId, editMsgId, fmt.Sprintf("💰 出售成功！\n\n卖出 %d 条鱼\n收入 %s（含市场波动）", totalCount, farmQuotaStr(totalValue)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🎣 继续钓鱼", CallbackData: "farm_fish"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func doFarmStoreFish(chatId int64, editMsgId int, tgId string, from *TgUser) {
	fishItems, _ := model.GetFishItems(tgId)
	if len(fishItems) == 0 {
		farmSend(chatId, editMsgId, "❌ 鱼仓库为空", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
			},
		}, from)
		return
	}

	currentTotal := model.GetWarehouseTotalCount(tgId)
	storedCount := 0
	details := ""
	for _, fi := range fishItems {
		fishKey := fi.ItemType[5:] // remove "fish_" prefix from item_type
		fd := fishTypeMap[fishKey]
		if fd == nil {
			continue
		}
		whLevel := model.GetWarehouseLevel(tgId)
		whMax := model.GetWarehouseMaxSlots(whLevel)
		if currentTotal+storedCount+fi.Quantity > whMax {
			details += fmt.Sprintf("\n%s %s: ❌ 仓库已满", fd.Emoji, fd.Name)
			continue
		}
		_ = model.AddToWarehouseWithCategory(tgId, "fish_"+fishKey, fi.Quantity, "fish")
		storedCount += fi.Quantity
		details += fmt.Sprintf("\n%s %s × %d → 📦仓库", fd.Emoji, fd.Name, fi.Quantity)
	}
	// 清空鱼背包
	if storedCount > 0 {
		_, _ = model.SellAllFish(tgId)
	}

	if storedCount == 0 {
		farmSend(chatId, editMsgId, "❌ 仓库已满，无法存入", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "📦 查看仓库", CallbackData: "farm_warehouse"}},
				{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
			},
		}, from)
		return
	}

	model.AddFarmLog(tgId, "fish_store", 0, fmt.Sprintf("鱼存入仓库%d条", storedCount))
	farmSend(chatId, editMsgId, fmt.Sprintf("📦 存入仓库成功！\n%s\n\n共存入 %d 条鱼", details, storedCount), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "📦 查看仓库", CallbackData: "farm_warehouse"}},
			{{Text: "🎣 继续钓鱼", CallbackData: "farm_fish"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 等级升级 ==========

func showFarmLevelUp(chatId int64, editMsgId int, tgId string, from *TgUser) {
	level := model.GetFarmLevel(tgId)
	if level >= common.TgBotFarmMaxLevel {
		farmSend(chatId, editMsgId, fmt.Sprintf("⭐ 已达最高等级 Lv.%d！", level), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	price := getLevelUpPrice(level)
	text := fmt.Sprintf("⬆️ 等级升级\n\n当前等级: ⭐Lv.%d\n升级费用: %s\n升级后: ⭐Lv.%d\n\n", level, farmQuotaStr(price), level+1)

	text += "📋 功能解锁一览:\n"
	for _, fu := range featureUnlocks {
		req := *fu.Level
		icon := "✅"
		if level < req {
			icon = "🔒"
		}
		text += fmt.Sprintf("  %s %s - Lv.%d\n", icon, fu.Name, req)
	}

	text += fmt.Sprintf("\n📊 等级价格表:\n")
	for i, p := range common.TgBotFarmLevelPrices {
		lv := i + 2
		if lv > common.TgBotFarmMaxLevel {
			break
		}
		marker := "  "
		if lv == level+1 {
			marker = "👉"
		}
		text += fmt.Sprintf("%s Lv.%d: %s\n", marker, lv, farmQuotaStr(p))
	}

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: fmt.Sprintf("💰 升级到 Lv.%d (%s)", level+1, farmQuotaStr(price)), CallbackData: "farm_dolevelup"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func doFarmLevelUp(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	level := model.GetFarmLevel(tgId)
	if level >= common.TgBotFarmMaxLevel {
		farmSend(chatId, editMsgId, "❌ 已达最高等级", nil, from)
		return
	}

	// 有未还贷款时禁止升级
	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr == nil && activeLoan != nil {
		farmSend(chatId, editMsgId, "❌ 你有未还清的贷款，还清后才能升级！\n\n贷款资金不能用于升级。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏦 去银行还款", CallbackData: "farm_bank"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 抵押违约永久禁止10级+
	newLevel := level + 1
	if newLevel >= 10 && model.HasMortgageBlocked(tgId) {
		farmSend(chatId, editMsgId, "🚫 由于抵押贷款违约，你已被永久禁止升级到10级及以上等级。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	price := getLevelUpPrice(level)
	userQuota, _ := model.GetUserQuota(user.Id, false)
	if userQuota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足\n\n需要 %s，当前余额 %s", farmQuotaStr(price), farmQuotaStr(userQuota)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}
	_ = model.DecreaseUserQuota(user.Id, price)

	model.SetFarmLevel(tgId, newLevel)
	model.AddFarmLog(tgId, "levelup", -price, fmt.Sprintf("升级到Lv.%d", newLevel))

	// 检查新解锁的功能
	unlocked := ""
	for _, fu := range featureUnlocks {
		if *fu.Level == newLevel {
			unlocked += fmt.Sprintf("\n🔓 解锁: %s", fu.Name)
		}
	}

	farmSend(chatId, editMsgId, fmt.Sprintf("🎉 升级成功！\n\n⭐ Lv.%d → Lv.%d\n💰 花费 %s%s",
		level, newLevel, farmQuotaStr(price), unlocked), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 每日任务 ==========

func showFarmTasks(chatId int64, editMsgId int, tgId string, from *TgUser) {
	dateStr := todayDateStr()
	tasks := getDailyTasks(dateStr)
	claimed, _ := model.GetTaskClaims(tgId, dateStr)
	claimedSet := make(map[int]bool)
	for _, idx := range claimed {
		claimedSet[idx] = true
	}

	text := fmt.Sprintf("📝 每日任务（%s）\n\n", dateStr[:4]+"-"+dateStr[4:6]+"-"+dateStr[6:])

	var rows [][]TgInlineKeyboardButton
	allDone := true
	for i, task := range tasks {
		progress := model.CountTodayActions(tgId, task.Action)
		done := progress >= int64(task.Target)
		isClaimed := claimedSet[i]

		statusIcon := "⬜"
		if isClaimed {
			statusIcon = "✅"
		} else if done {
			statusIcon = "🟢"
			allDone = false
		} else {
			allDone = false
		}

		text += fmt.Sprintf("%s %s %s %d/%d  奖励%s\n",
			statusIcon, task.Emoji, task.Name, progress, task.Target, farmQuotaStr(task.Reward))

		// 显示完成条件说明
		desc := getTaskDesc(task.Action, task.Target)
		text += fmt.Sprintf("   📋 %s\n", desc)

		// 显示操作提示（未完成时）
		if !done {
			if meta, ok := actionMetaMap[task.Action]; ok {
				text += fmt.Sprintf("   💡 %s\n", meta.Hint)
				// 显示自动化兼容状态
				if meta.AutoType != "" {
					if model.HasAutomation(tgId, meta.AutoType) {
						text += fmt.Sprintf("   %s（✅ 已安装）\n", meta.AutoText)
					} else {
						text += fmt.Sprintf("   %s（❌ 未安装）\n", meta.AutoText)
					}
				}
			}
		}

		if done && !isClaimed {
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("📥 领取: %s %s", task.Emoji, task.Name),
					CallbackData: fmt.Sprintf("farm_tclaim_%d", i)},
			})
		}
	}

	if allDone && len(claimed) == len(tasks) {
		text += "\n🎉 今日任务全部完成！\n"
	}

	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🏆 查看成就", CallbackData: "farm_achieve"},
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmClaimTask(chatId int64, editMsgId int, tgId string, taskIdx int, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	dateStr := todayDateStr()
	tasks := getDailyTasks(dateStr)
	if taskIdx < 0 || taskIdx >= len(tasks) {
		farmSend(chatId, editMsgId, "❌ 无效任务", nil, from)
		return
	}

	// Check already claimed
	claimed, _ := model.GetTaskClaims(tgId, dateStr)
	for _, idx := range claimed {
		if idx == taskIdx {
			farmSend(chatId, editMsgId, "❌ 该任务奖励已领取", &TgInlineKeyboardMarkup{
				InlineKeyboard: [][]TgInlineKeyboardButton{
					{{Text: "🔙 返回任务", CallbackData: "farm_tasks"}},
				},
			}, from)
			return
		}
	}

	task := tasks[taskIdx]
	progress := model.CountTodayActions(tgId, task.Action)
	if progress < int64(task.Target) {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 任务未完成（%d/%d）", progress, task.Target), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回任务", CallbackData: "farm_tasks"}},
			},
		}, from)
		return
	}

	_ = model.ClaimTask(tgId, dateStr, taskIdx)
	_ = model.IncreaseUserQuota(user.Id, task.Reward, true)
	model.AddFarmLog(tgId, "task", task.Reward, fmt.Sprintf("完成任务:%s", task.Name))

	farmSend(chatId, editMsgId, fmt.Sprintf("🎉 任务完成！\n\n%s %s\n💰 奖励 %s",
		task.Emoji, task.Name, farmQuotaStr(task.Reward)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "📝 返回任务", CallbackData: "farm_tasks"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 成就 ==========

func showFarmAchievements(chatId int64, editMsgId int, tgId string, from *TgUser) {
	unlocked, _ := model.GetAchievements(tgId)
	unlockedSet := make(map[string]bool)
	for _, a := range unlocked {
		unlockedSet[a.AchievementKey] = true
	}

	text := "🏆 成就\n\n"
	farmLevel := model.GetFarmLevel(tgId)
	var rows [][]TgInlineKeyboardButton
	for _, ach := range achievements {
		isUnlocked := unlockedSet[ach.Key]
		var progress int64
		if ach.Action == "levelup" {
			progress = int64(farmLevel)
		} else {
			progress = model.CountTotalActions(tgId, ach.Action)
		}
		done := progress >= ach.Target

		statusIcon := "⬜"
		if isUnlocked {
			statusIcon = "✅"
		} else if done {
			statusIcon = "🟢"
		}

		text += fmt.Sprintf("%s %s %s - %s %d/%d  奖励%s\n",
			statusIcon, ach.Emoji, ach.Name, ach.Description, progress, ach.Target, farmQuotaStr(ach.Reward))

		if done && !isUnlocked {
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("📥 领取: %s %s", ach.Emoji, ach.Name),
					CallbackData: "farm_aclaim_" + ach.Key},
			})
		}
	}

	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "📝 每日任务", CallbackData: "farm_tasks"},
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmClaimAchievement(chatId int64, editMsgId int, tgId string, achKey string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	// Find achievement
	var ach *achievementDef
	for i := range achievements {
		if achievements[i].Key == achKey {
			ach = &achievements[i]
			break
		}
	}
	if ach == nil {
		farmSend(chatId, editMsgId, "❌ 未知成就", nil, from)
		return
	}

	if model.HasAchievement(tgId, achKey) {
		farmSend(chatId, editMsgId, "❌ 成就已领取", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回成就", CallbackData: "farm_achieve"}},
			},
		}, from)
		return
	}

	var progress int64
	if ach.Action == "levelup" {
		progress = int64(model.GetFarmLevel(tgId))
	} else {
		progress = model.CountTotalActions(tgId, ach.Action)
	}
	if progress < ach.Target {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 成就未达成（%d/%d）", progress, ach.Target), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回成就", CallbackData: "farm_achieve"}},
			},
		}, from)
		return
	}

	_ = model.UnlockAchievement(tgId, achKey)
	_ = model.IncreaseUserQuota(user.Id, ach.Reward, true)
	model.AddFarmLog(tgId, "achieve", ach.Reward, fmt.Sprintf("解锁成就:%s", ach.Name))

	farmSend(chatId, editMsgId, fmt.Sprintf("🏆 成就解锁！\n\n%s %s\n%s\n💰 奖励 %s",
		ach.Emoji, ach.Name, ach.Description, farmQuotaStr(ach.Reward)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🏆 返回成就", CallbackData: "farm_achieve"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 加工坊 ==========

func showFarmWorkshop(chatId int64, editMsgId int, tgId string, from *TgUser) {
	procs, _ := model.GetFarmProcesses(tgId)
	now := time.Now().Unix()

	// 更新状态
	for _, p := range procs {
		if p.Status == 1 && now >= p.FinishAt {
			p.Status = 2
		}
	}

	activeCount := int64(len(procs))
	text := fmt.Sprintf("🏭 加工坊（%d/%d 槽位）\n\n", activeCount, model.FarmMaxProcessSlots)

	// 当前加工
	hasCollectable := false
	if len(procs) == 0 {
		text += "📭 暂无加工任务\n"
	} else {
		for _, p := range procs {
			r := recipeMap[p.RecipeKey]
			if r == nil {
				continue
			}
			if p.Status == 2 {
				sellPrice := applyMarket(r.SellPrice, "recipe_"+r.Key)
				mPct := getMarketMultiplier("recipe_" + r.Key)
				text += fmt.Sprintf("✅ %s %s - 已完成！可收取 %s(%d%%)\n", r.Emoji, r.Name, farmQuotaStr(sellPrice), mPct)
				hasCollectable = true
			} else {
				remain := p.FinishAt - now
				if remain < 0 {
					remain = 0
				}
				pct := int((now - p.StartedAt) * 100 / (p.FinishAt - p.StartedAt))
				if pct > 99 {
					pct = 99
				}
				text += fmt.Sprintf("⏳ %s %s - 加工中 %d%% 剩余%s\n", r.Emoji, r.Name, pct, formatDuration(remain))
			}
		}
	}

	text += "\n📋 配方列表:\n"
	for _, r := range recipes {
		sellPrice := applyMarket(r.SellPrice, "recipe_"+r.Key)
		mPct := getMarketMultiplier("recipe_" + r.Key)
		profit := sellPrice - r.Cost
		profitSign := "+"
		if profit < 0 {
			profitSign = ""
		}
		text += fmt.Sprintf("  %s %s 成本%s → 售价%s(%d%%) %s%s 耗时%s\n",
			r.Emoji, r.Name, farmQuotaStr(r.Cost), farmQuotaStr(sellPrice), mPct,
			profitSign, farmQuotaStr(profit), formatDuration(r.TimeSecs))
	}

	var rows [][]TgInlineKeyboardButton
	if hasCollectable {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "📥 收取并出售", CallbackData: "farm_collect"},
		})
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "📦 收取存入仓库", CallbackData: "farm_collect_store"},
		})
	}
	if activeCount < int64(model.FarmMaxProcessSlots) {
		for _, r := range recipes {
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("%s %s (%s)", r.Emoji, r.Name, farmQuotaStr(r.Cost)),
					CallbackData: "farm_craft_" + r.Key},
			})
		}
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmCraft(chatId int64, editMsgId int, tgId string, recipeKey string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	r := recipeMap[recipeKey]
	if r == nil {
		farmSend(chatId, editMsgId, "❌ 未知配方", nil, from)
		return
	}

	// 检查槽位
	count := model.CountActiveProcesses(tgId)
	if count >= int64(model.FarmMaxProcessSlots) {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 加工槽已满（%d/%d）", count, model.FarmMaxProcessSlots), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回加工坊", CallbackData: "farm_workshop"}},
			},
		}, from)
		return
	}

	// 扣费
	err = model.DecreaseUserQuota(user.Id, r.Cost)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 余额不足", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回加工坊", CallbackData: "farm_workshop"}},
			},
		}, from)
		return
	}

	now := time.Now().Unix()
	proc := &model.TgFarmProcess{
		TelegramId: tgId,
		RecipeKey:  recipeKey,
		StartedAt:  now,
		FinishAt:   now + r.TimeSecs,
		Status:     1,
	}
	_ = model.CreateFarmProcess(proc)
	model.AddFarmLog(tgId, "craft", -r.Cost, fmt.Sprintf("加工%s%s", r.Emoji, r.Name))

	farmSend(chatId, editMsgId, fmt.Sprintf("🏭 开始加工 %s %s！\n\n成本: %s\n耗时: %s\n预计产出: %s",
		r.Emoji, r.Name, farmQuotaStr(r.Cost), formatDuration(r.TimeSecs),
		farmQuotaStr(applyMarket(r.SellPrice, "recipe_"+r.Key))),
		&TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏭 返回加工坊", CallbackData: "farm_workshop"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
}

func doFarmCollectAll(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	procs, _ := model.GetFarmProcesses(tgId)
	now := time.Now().Unix()

	totalValue := 0
	collected := 0
	for _, p := range procs {
		if p.Status == 1 && now >= p.FinishAt {
			p.Status = 2
		}
		if p.Status == 2 {
			r := recipeMap[p.RecipeKey]
			if r == nil {
				continue
			}
			sellPrice := applyMarket(r.SellPrice, "recipe_"+r.Key)
			totalValue += sellPrice
			collected++
			_ = model.CollectFarmProcess(p.Id)
			model.RecordCollection(tgId, "recipe", r.Key, 1)
		}
	}

	if collected == 0 {
		farmSend(chatId, editMsgId, "❌ 没有可收取的成品", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回加工坊", CallbackData: "farm_workshop"}},
			},
		}, from)
		return
	}

	// 应用转生加成
	prestige := model.GetPrestigeLevel(tgId)
	bonus := prestige * common.TgBotFarmPrestigeBonusPerLevel
	prestigeAdd := 0
	if bonus > 0 {
		prestigeAdd = common.SafeQuotaMulDiv(totalValue, bonus, 100)
	}
	finalValue := common.SafeQuotaAdd(totalValue, prestigeAdd)

	adminId := common.FarmAdminUserId
	if adminId > 0 && adminId != user.Id {
		_ = model.DecreaseUserQuota(adminId, finalValue)
		_ = model.IncreaseUserQuota(user.Id, finalValue, true)
		model.AddFarmLog(tgId, "craft_sell", finalValue, fmt.Sprintf("收取%d件加工品(加成+%d%%)", collected, bonus))
	} else {
		_ = model.IncreaseUserQuota(user.Id, finalValue, true)
		model.AddFarmLog(tgId, "craft_sell", finalValue, fmt.Sprintf("收取%d件加工品(加成+%d%%)", collected, bonus))
	}

	farmSend(chatId, editMsgId, fmt.Sprintf("📥 收取成功！\n\n收取 %d 件成品\n💰 收入 %s（转生加成+%d%%）", collected, farmQuotaStr(finalValue), bonus),
		&TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏭 返回加工坊", CallbackData: "farm_workshop"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
}

func doFarmCollectStore(chatId int64, editMsgId int, tgId string, from *TgUser) {
	procs, _ := model.GetFarmProcesses(tgId)
	now := time.Now().Unix()

	currentTotal := model.GetWarehouseTotalCount(tgId)
	stored := 0
	details := ""
	for _, p := range procs {
		if p.Status == 1 && now >= p.FinishAt {
			p.Status = 2
		}
		if p.Status == 2 {
			r := recipeMap[p.RecipeKey]
			if r == nil {
				continue
			}
			whLevel := model.GetWarehouseLevel(tgId)
			whMax := model.GetWarehouseMaxSlots(whLevel)
			if currentTotal+stored+1 > whMax {
				details += fmt.Sprintf("\n%s %s: ❌ 仓库已满", r.Emoji, r.Name)
				continue
			}
			_ = model.AddToWarehouseWithCategory(tgId, "recipe_"+r.Key, 1, "recipe")
			stored++
			_ = model.CollectFarmProcess(p.Id)
			model.RecordCollection(tgId, "recipe", r.Key, 1)
			details += fmt.Sprintf("\n%s %s → 📦仓库", r.Emoji, r.Name)
		}
	}

	if stored == 0 {
		farmSend(chatId, editMsgId, "❌ 没有可收取的成品或仓库已满", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回加工坊", CallbackData: "farm_workshop"}},
			},
		}, from)
		return
	}

	model.AddFarmLog(tgId, "craft_store", 0, fmt.Sprintf("加工品存入仓库%d件", stored))
	farmSend(chatId, editMsgId, fmt.Sprintf("📦 存入仓库成功！\n%s\n\n共存入 %d 件加工品\n⚠️ 加工食品5天后发霉", details, stored),
		&TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "📦 查看仓库", CallbackData: "farm_warehouse"}},
				{{Text: "🏭 返回加工坊", CallbackData: "farm_workshop"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
}

// ========== 市场行情 ==========

func showFarmMarket(chatId int64, editMsgId int, tgId string, from *TgUser) {
	ensureMarketFresh()

	nextRefresh := getMarketNextRefresh()
	text := fmt.Sprintf("📈 市场行情（%dh刷新一次，%dh后刷新）\n\n", common.TgBotMarketRefreshHours, nextRefresh/3600+1)

	season := getCurrentSeason()
	text += fmt.Sprintf("当前: %s\n\n", getSeasonName(season))

	// 市场情报
	tips := getMarketTips()
	if len(tips) > 0 {
		text += "📋 市场情报:\n"
		maxTips := 5
		if len(tips) < maxTips {
			maxTips = len(tips)
		}
		for _, tip := range tips[:maxTips] {
			text += fmt.Sprintf("  %s %s\n", tip.Icon, tip.Text)
		}
		text += "\n"
	}

	text += "🌾 作物:\n"
	for _, crop := range farmCrops {
		m := getMarketMultiplier("crop_" + crop.Key)
		tag, arrow, _ := getMarketPriceTrend("crop_" + crop.Key)
		marketPrice := applyMarket(crop.UnitPrice, "crop_"+crop.Key)
		seasonPrice := applySeasonPrice(marketPrice, &crop)
		sTag := "应季"
		if !isCropInSeason(&crop) {
			sTag = "反季"
		}
		text += fmt.Sprintf("  %s %s %d%% %s%s ×%s%d%% = %s\n",
			crop.Emoji, crop.Name, m, arrow, tag, sTag, getSeasonPriceMultiplier(&crop), farmQuotaStr(seasonPrice))
	}

	text += "\n🐟 鱼类:\n"
	for _, fish := range fishTypes {
		m := getMarketMultiplier("fish_" + fish.Key)
		tag, arrow, _ := getMarketPriceTrend("fish_" + fish.Key)
		text += fmt.Sprintf("  %s %s %d%% %s%s %s\n", fish.Emoji, fish.Name, m, arrow, tag, farmQuotaStr(applyMarket(fish.SellPrice, "fish_"+fish.Key)))
	}

	text += "\n🥩 肉类:\n"
	for _, a := range ranchAnimals {
		m := getMarketMultiplier("meat_" + a.Key)
		tag, arrow, _ := getMarketPriceTrend("meat_" + a.Key)
		text += fmt.Sprintf("  %s %s肉 %d%% %s%s %s\n", a.Emoji, a.Name, m, arrow, tag, farmQuotaStr(applyMarket(*a.MeatPrice, "meat_"+a.Key)))
	}

	text += "\n🏭 加工品:\n"
	for _, r := range recipes {
		m := getMarketMultiplier("recipe_" + r.Key)
		tag, arrow, _ := getMarketPriceTrend("recipe_" + r.Key)
		text += fmt.Sprintf("  %s %s %d%% %s%s %s\n", r.Emoji, r.Name, m, arrow, tag, farmQuotaStr(applyMarket(r.SellPrice, "recipe_"+r.Key)))
	}

	text += "\n🪵 木材:\n"
	for _, tp := range treeProducts {
		m := getMarketMultiplier("wood_" + tp.Key)
		tag, arrow, _ := getMarketPriceTrend("wood_" + tp.Key)
		text += fmt.Sprintf("  %s %s %d%% %s%s %s\n", tp.Emoji, tp.Name, m, arrow, tag, farmQuotaStr(applyMarket(tp.BasePrice, "wood_"+tp.Key)))
	}

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{
				{Text: "📊 作物波动图", CallbackData: "farm_chart_crop"},
				{Text: "📊 鱼类波动图", CallbackData: "farm_chart_fish"},
			},
			{
				{Text: "📊 肉类波动图", CallbackData: "farm_chart_meat"},
				{Text: "📊 加工品波动图", CallbackData: "farm_chart_recipe"},
			},
			{
				{Text: "📊 木材波动图", CallbackData: "farm_chart_wood"},
			},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func marketTag(m int) string {
	if m >= 180 {
		return "🔥暴涨"
	} else if m >= 140 {
		return "📈大涨"
	} else if m >= 110 {
		return "📈涨"
	} else if m >= 90 {
		return "➡️稳"
	} else if m >= 60 {
		return "📉跌"
	}
	return "📉暴跌"
}

// doFarmMarketChart 生成并发送市场波动图
func doFarmMarketChart(chatId int64, editMsgId int, tgId string, category string, from *TgUser) {
	ensureMarketFresh()

	pngData, err := generateMarketChartPNG(category)
	if err != nil {
		farmSend(chatId, editMsgId, "📊 "+err.Error()+"\n\n市场需要至少刷新2次才能生成波动图。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回市场", CallbackData: "farm_market"}},
			},
		}, from)
		return
	}

	sendTgPhoto(chatId, pngData, getCategoryTitle(category), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{
				{Text: "📊 作物", CallbackData: "farm_chart_crop"},
				{Text: "📊 鱼类", CallbackData: "farm_chart_fish"},
			},
			{
				{Text: "📊 肉类", CallbackData: "farm_chart_meat"},
				{Text: "📊 加工品", CallbackData: "farm_chart_recipe"},
			},
			{
				{Text: "📊 木材", CallbackData: "farm_chart_wood"},
			},
			{{Text: "📈 返回市场", CallbackData: "farm_market"}},
		},
	})
}

// ========== 银行贷款 ==========

func showFarmBank(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	// 检查抵押贷款违约
	defaulted, penalty := model.CheckMortgageDefault(tgId)
	if defaulted {
		if penalty == "ban" {
			farmSend(chatId, editMsgId, "🚫 你的抵押贷款已逾期违约！\n\n由于你等级≥10级，你的平台账号已被封禁。", nil, from)
			return
		} else if penalty == "block_level" {
			farmSend(chatId, editMsgId, "⚠️ 你的抵押贷款已逾期违约！\n\n你已被永久禁止升级到10级及以上等级。", &TgInlineKeyboardMarkup{
				InlineKeyboard: [][]TgInlineKeyboardButton{
					{{Text: "🏦 返回银行", CallbackData: "farm_bank"}},
				},
			}, from)
			return
		}
	}

	creditScore := model.GetCreditScore(tgId)
	baseAmount := common.TgBotFarmBankBaseAmount
	maxLoan := common.SafeQuotaMulDiv(baseAmount, creditScore, 1)
	interestRate := common.TgBotFarmBankInterestRate
	interest := common.SafeQuotaMulDiv(maxLoan, interestRate, 100)
	totalDue := common.SafeQuotaAdd(maxLoan, interest)
	loanDays := common.TgBotFarmBankMaxLoanDays

	text := fmt.Sprintf("🏦 银行\n\n"+
		"💰 余额: %s\n"+
		"📊 信用评分: %d/%d\n"+
		"💵 可贷额度: %s\n"+
		"📈 利率: %d%%\n"+
		"💸 应还总额: %s（本金+利息）\n"+
		"📅 还款期限: %d天\n",
		farmQuotaStr(user.Quota),
		creditScore, common.TgBotFarmBankMaxMultiplier,
		farmQuotaStr(maxLoan),
		interestRate,
		farmQuotaStr(totalDue),
		loanDays)

	// 抵押贷款信息
	mortgageBlocked := model.HasMortgageBlocked(tgId)
	if mortgageBlocked {
		text += "\n🚫 你已被永久禁止升级到10级及以上（抵押违约）\n"
	}
	text += fmt.Sprintf("\n🏠 抵押贷款: 最高 %s（利率%d%%）\n  以10级升级权为抵押，还不上将永久失去10级资格\n  10级以上违约将封禁账号\n  ⚠️ 抵押贷款不能用于升级\n",
		farmQuotaStr(common.TgBotFarmMortgageMaxAmount), common.TgBotFarmMortgageInterestRate)

	// 检查是否有未还贷款
	activeLoan, loanErr := model.GetActiveLoan(tgId)
	var rows [][]TgInlineKeyboardButton

	if loanErr == nil && activeLoan != nil {
		remaining := activeLoan.TotalDue - activeLoan.Repaid
		now := time.Now().Unix()
		daysLeft := (activeLoan.DueAt - now) / 86400
		if daysLeft < 0 {
			daysLeft = 0
		}
		overdue := ""
		if now > activeLoan.DueAt {
			overdue = " ⚠️已逾期！"
		}

		loanTypeTag := "普通"
		if activeLoan.LoanType == 1 {
			loanTypeTag = "🏠抵押"
		}

		text += fmt.Sprintf("\n📋 当前贷款（%s）:\n"+
			"  本金: %s\n"+
			"  利息: %s\n"+
			"  已还: %s\n"+
			"  剩余: %s\n"+
			"  剩余天数: %d天%s\n",
			loanTypeTag,
			farmQuotaStr(activeLoan.Principal),
			farmQuotaStr(activeLoan.Interest),
			farmQuotaStr(activeLoan.Repaid),
			farmQuotaStr(remaining),
			daysLeft, overdue)

		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("💰 全额还款 (%s)", farmQuotaStr(remaining)), CallbackData: "farm_repay"},
		})
		if remaining > 1 {
			halfAmount := remaining / 2
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("💰 还一半 (%s)", farmQuotaStr(halfAmount)), CallbackData: "farm_repay_half"},
			})
		}
	} else {
		text += "\n✅ 当前无贷款\n"
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("💵 信用贷款 (%s)", farmQuotaStr(maxLoan)), CallbackData: "farm_doloan"},
		})
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🏠 抵押贷款（$1~$%d）", common.TgBotFarmMortgageMaxAmount/500000), CallbackData: "farm_mortgage"},
		})
	}

	// 贷款历史
	history, _ := model.GetLoanHistory(tgId, 5)
	if len(history) > 0 {
		text += "\n📜 贷款历史（最近5条）:\n"
		for _, loan := range history {
			statusTag := "⏳还款中"
			if loan.Status == 1 {
				statusTag = "✅已还清"
			} else if loan.Status == 2 {
				statusTag = "❌违约"
			}
			typeTag := ""
			if loan.LoanType == 1 {
				typeTag = "[抵押]"
			}
			ts := time.Unix(loan.CreatedAt, 0)
			text += fmt.Sprintf("  %s %s本金%s 评分%d %s\n",
				ts.Format("01-02"), typeTag, farmQuotaStr(loan.Principal), loan.CreditScore, statusTag)
		}
	}

	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmLoan(chatId int64, editMsgId int, tgId string, from *TgUser) {
	// 检查是否已有未还贷款
	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr == nil && activeLoan != nil {
		farmSend(chatId, editMsgId, "❌ 你还有未还清的贷款！请先还清再申请。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏦 返回银行", CallbackData: "farm_bank"}},
			},
		}, from)
		return
	}

	creditScore := model.GetCreditScore(tgId)
	baseAmount := common.TgBotFarmBankBaseAmount
	principal := common.SafeQuotaMulDiv(baseAmount, creditScore, 1)
	interestRate := common.TgBotFarmBankInterestRate
	interest := common.SafeQuotaMulDiv(principal, interestRate, 100)
	totalDue := common.SafeQuotaAdd(principal, interest)
	loanDays := common.TgBotFarmBankMaxLoanDays

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	// 创建贷款
	loan, err := model.CreateLoan(tgId, principal, interest, totalDue, creditScore, loanDays)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 贷款申请失败，请稍后再试。", nil, from)
		return
	}

	// 放款到用户账户
	_ = model.IncreaseUserQuota(user.Id, principal, true)
	model.AddFarmLog(tgId, "loan", principal, fmt.Sprintf("银行贷款 评分%d", creditScore))

	common.SysLog(fmt.Sprintf("TG Farm Bank: user %s loan %d quota, score %d, due %d", tgId, principal, creditScore, totalDue))

	dueTime := time.Unix(loan.DueAt, 0)
	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 贷款成功！\n\n"+
		"💵 贷款金额: %s\n"+
		"📈 利息: %s (%d%%)\n"+
		"💸 应还总额: %s\n"+
		"📅 还款期限: %s\n"+
		"📊 信用评分: %d\n\n"+
		"贷款已发放到你的账户！",
		farmQuotaStr(principal),
		farmQuotaStr(interest), interestRate,
		farmQuotaStr(totalDue),
		dueTime.Format("2006-01-02"),
		creditScore), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🏦 返回银行", CallbackData: "farm_bank"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func doFarmRepay(chatId int64, editMsgId int, tgId string, from *TgUser) {
	doFarmRepayPartial(chatId, editMsgId, tgId, 100, from)
}

func doFarmRepayPartial(chatId int64, editMsgId int, tgId string, percent int, from *TgUser) {
	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr != nil || activeLoan == nil {
		farmSend(chatId, editMsgId, "❌ 你没有待还贷款。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏦 返回银行", CallbackData: "farm_bank"}},
			},
		}, from)
		return
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	remaining := activeLoan.TotalDue - activeLoan.Repaid
	repayAmount := remaining
	if percent < 100 {
		repayAmount = remaining * percent / 100
		if repayAmount < 1 {
			repayAmount = 1
		}
	}

	if user.Quota < repayAmount {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！\n\n需要: %s\n余额: %s",
			farmQuotaStr(repayAmount), farmQuotaStr(user.Quota)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏦 返回银行", CallbackData: "farm_bank"}},
			},
		}, from)
		return
	}

	// 扣款
	err = model.DecreaseUserQuota(user.Id, repayAmount)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败，请稍后再试。", nil, from)
		return
	}

	extendDays := 0
	if percent == 50 {
		extendDays = 2
	}
	loan, err := model.RepayLoanWithExtend(activeLoan.Id, repayAmount, extendDays)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, repayAmount, true)
		farmSend(chatId, editMsgId, "❌ 还款失败，已退款。", nil, from)
		return
	}

	logMsg := fmt.Sprintf("还款%d%%", percent)
	if extendDays > 0 && loan.Status != 1 {
		logMsg += fmt.Sprintf("，期限+%d天", extendDays)
	}
	model.AddFarmLog(tgId, "repay", -repayAmount, logMsg)
	common.SysLog(fmt.Sprintf("TG Farm Bank: user %s repaid %d quota, loan status %d", tgId, repayAmount, loan.Status))

	statusMsg := ""
	if loan.Status == 1 {
		statusMsg = "\n\n🎉 贷款已全部还清！"
	} else {
		newRemaining := loan.TotalDue - loan.Repaid
		extendMsg := ""
		if extendDays > 0 {
			dueTime := time.Unix(loan.DueAt, 0)
			extendMsg = fmt.Sprintf("\n📅 期限延长%d天，新截止: %s", extendDays, dueTime.Format("2006-01-02"))
		}
		statusMsg = fmt.Sprintf("\n\n剩余待还: %s%s", farmQuotaStr(newRemaining), extendMsg)
	}

	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 还款成功！\n\n💰 还款金额: %s%s",
		farmQuotaStr(repayAmount), statusMsg), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🏦 返回银行", CallbackData: "farm_bank"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 仓库 ==========

func warehouseItemName(item *model.TgFarmWarehouse) (string, string) {
	switch item.Category {
	case "fish":
		fishKey := item.CropType
		if len(fishKey) > 5 && fishKey[:5] == "fish_" {
			fishKey = fishKey[5:]
		}
		if fd := fishTypeMap[fishKey]; fd != nil {
			return fd.Emoji, fd.Name
		}
		return "🐟", item.CropType
	case "meat":
		meatKey := item.CropType
		if len(meatKey) > 5 && meatKey[:5] == "meat_" {
			meatKey = meatKey[5:]
		}
		if ad := ranchAnimalMap[meatKey]; ad != nil {
			return ad.Emoji, ad.Name + "肉"
		}
		return "🥩", item.CropType
	case "recipe":
		recipeKey := item.CropType
		if len(recipeKey) > 7 && recipeKey[:7] == "recipe_" {
			recipeKey = recipeKey[7:]
		}
		if rd := recipeMap[recipeKey]; rd != nil {
			return rd.Emoji, rd.Name
		}
		return "🍽️", item.CropType
	case "fruit":
		// 向后兼容：旧数据存储为 fruit_<treeType>，通过树种定义找到第一个采收产物
		treeKey := item.CropType
		if len(treeKey) > 6 && treeKey[:6] == "fruit_" {
			treeKey = treeKey[6:]
		}
		if tree := treeFarmTreeMap[treeKey]; tree != nil && len(tree.HarvestYield) > 0 {
			y := tree.HarvestYield[0]
			if tp := treeProductMap[y.ItemKey]; tp != nil {
				return tp.Emoji, tp.Name
			}
			return y.Emoji, y.Name
		}
		return "🍎", item.CropType
	case "wood":
		woodKey := item.CropType
		if len(woodKey) > 5 && woodKey[:5] == "wood_" {
			woodKey = woodKey[5:]
		}
		if tp := treeProductMap[woodKey]; tp != nil {
			return tp.Emoji, tp.Name
		}
		// 向后兼容：旧数据存储为 wood_<treeType>，通过树种找伐木第一产物
		if tree := treeFarmTreeMap[woodKey]; tree != nil && len(tree.ChopYield) > 0 {
			y := tree.ChopYield[0]
			if tp := treeProductMap[y.ItemKey]; tp != nil {
				return tp.Emoji, tp.Name
			}
			return y.Emoji, y.Name
		}
		return "🪵", item.CropType
	default:
		if crop := farmCropMap[item.CropType]; crop != nil {
			return crop.Emoji, crop.Name
		}
		return "🌿", item.CropType
	}
}

func warehouseItemSellPrice(item *model.TgFarmWarehouse) int {
	switch item.Category {
	case "fish":
		fishKey := item.CropType
		if len(fishKey) > 5 && fishKey[:5] == "fish_" {
			fishKey = fishKey[5:]
		}
		if fd := fishTypeMap[fishKey]; fd != nil {
			return applyMarket(fd.SellPrice, "fish_"+fishKey)
		}
		return 0
	case "meat":
		meatKey := item.CropType
		if len(meatKey) > 5 && meatKey[:5] == "meat_" {
			meatKey = meatKey[5:]
		}
		if ad := ranchAnimalMap[meatKey]; ad != nil {
			return applyMarket(*ad.MeatPrice, "meat_"+meatKey)
		}
		return 0
	case "recipe":
		recipeKey := item.CropType
		if len(recipeKey) > 7 && recipeKey[:7] == "recipe_" {
			recipeKey = recipeKey[7:]
		}
		if rd := recipeMap[recipeKey]; rd != nil {
			return applyMarket(rd.SellPrice, "recipe_"+recipeKey)
		}
		return 0
	case "fruit":
		// 向后兼容：旧数据存储为 fruit_<treeType>
		treeKey := item.CropType
		if len(treeKey) > 6 && treeKey[:6] == "fruit_" {
			treeKey = treeKey[6:]
		}
		if tree := treeFarmTreeMap[treeKey]; tree != nil && len(tree.HarvestYield) > 0 {
			y := tree.HarvestYield[0]
			if tp := treeProductMap[y.ItemKey]; tp != nil {
				return applyMarket(tp.BasePrice, "wood_"+y.ItemKey)
			}
		}
		return 0
	case "wood":
		woodKey := item.CropType
		if len(woodKey) > 5 && woodKey[:5] == "wood_" {
			woodKey = woodKey[5:]
		}
		if tp := treeProductMap[woodKey]; tp != nil {
			return applyMarket(tp.BasePrice, "wood_"+woodKey)
		}
		// 向后兼容：旧数据存储为 wood_<treeType>，通过树种找伐木第一产物
		if tree := treeFarmTreeMap[woodKey]; tree != nil && len(tree.ChopYield) > 0 {
			y := tree.ChopYield[0]
			if tp := treeProductMap[y.ItemKey]; tp != nil {
				return applyMarket(tp.BasePrice, "wood_"+y.ItemKey)
			}
		}
		return 0
	default:
		if crop := farmCropMap[item.CropType]; crop != nil {
			marketPrice := applyMarket(crop.UnitPrice, "crop_"+crop.Key)
			return applySeasonPrice(marketPrice, crop)
		}
		return 0
	}
}

func showFarmWarehouse(chatId int64, editMsgId int, tgId string, from *TgUser) {
	items, err := model.GetWarehouseItems(tgId)
	if err != nil || len(items) == 0 {
		farmSend(chatId, editMsgId, "📦 仓库空空如也\n\n收获时选择「收获到仓库」可以把作物存起来。\n钓鱼、屠宰、加工品也可存入仓库！\n⚠️ 肉类3天变质，加工品5天发霉", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	season := getCurrentSeason()
	daysLeft := getSeasonDaysLeft()
	totalCount := model.GetWarehouseTotalCount(tgId)
	whLevel := model.GetWarehouseLevel(tgId)
	whMax := model.GetWarehouseMaxSlots(whLevel)
	text := fmt.Sprintf("📦 仓库 Lv.%d (%d/%d)\n当前: %s (还剩%d天)\n\n",
		whLevel, totalCount, whMax, getSeasonName(season), daysLeft)

	now := time.Now().Unix()
	var rows [][]TgInlineKeyboardButton
	for _, item := range items {
		emoji, name := warehouseItemName(item)
		unitPrice := warehouseItemSellPrice(item)
		totalValue := common.SafeQuotaMulDiv(item.Quantity, unitPrice, 1)

		extra := ""
		if item.Category == "crop" {
			crop := farmCropMap[item.CropType]
			if crop != nil {
				if isCropInSeason(crop) {
					extra = " 🏷️应季"
				} else {
					extra = " 📈反季"
				}
			}
		} else if item.Category == "meat" {
			expiry := int64(common.TgBotFarmWarehouseMeatExpiry)
			remain := item.StoredAt + expiry - now
			if remain > 0 {
				extra = fmt.Sprintf(" ⏳%s后变质", formatDuration(remain))
			} else {
				extra = " ❌已变质"
			}
		} else if item.Category == "recipe" {
			expiry := int64(common.TgBotFarmWarehouseRecipeExpiry)
			remain := item.StoredAt + expiry - now
			if remain > 0 {
				extra = fmt.Sprintf(" ⏳%s后发霉", formatDuration(remain))
			} else {
				extra = " ❌已发霉"
			}
		}

		text += fmt.Sprintf("%s %s × %d | 单价%s%s | 总值%s\n",
			emoji, name, item.Quantity,
			farmQuotaStr(unitPrice), extra, farmQuotaStr(totalValue))
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("💰 出售 %s (%d个)", name, item.Quantity),
				CallbackData: fmt.Sprintf("farm_wh_sell_%s", item.CropType)},
		})
	}

	text += "\n💡 应季作物价格低，反季价格高\n⚠️ 肉类3天变质，加工品5天发霉"
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "💰 全部出售", CallbackData: "farm_wh_sellall"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmWarehouseSell(chatId int64, editMsgId int, tgId string, itemKey string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	item, err := model.GetWarehouseItem(tgId, itemKey)
	if err != nil || item.Quantity <= 0 {
		farmSend(chatId, editMsgId, "❌ 仓库中没有该物品", nil, from)
		return
	}

	unitPrice := warehouseItemSellPrice(item)
	totalValue := common.SafeQuotaMulDiv(item.Quantity, unitPrice, 1)
	emoji, name := warehouseItemName(item)

	_ = model.RemoveFromWarehouse(tgId, itemKey, item.Quantity)
	_ = model.IncreaseUserQuota(user.Id, totalValue, true)
	model.AddFarmLog(tgId, "warehouse_sell", totalValue, fmt.Sprintf("仓库出售%s×%d", name, item.Quantity))

	text := fmt.Sprintf("💰 仓库出售成功！\n\n%s %s × %d\n单价 %s\n\n💰 获得 %s 额度",
		emoji, name, item.Quantity,
		farmQuotaStr(unitPrice), farmQuotaStr(totalValue))

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "📦 返回仓库", CallbackData: "farm_warehouse"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func doFarmWarehouseSellAll(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	items, err := model.GetWarehouseItems(tgId)
	if err != nil || len(items) == 0 {
		farmSend(chatId, editMsgId, "❌ 仓库为空", nil, from)
		return
	}

	totalValue := 0
	details := ""
	for _, item := range items {
		emoji, name := warehouseItemName(item)
		unitPrice := warehouseItemSellPrice(item)
		value := common.SafeQuotaMulDiv(item.Quantity, unitPrice, 1)
		totalValue = common.SafeQuotaAdd(totalValue, value)
		_ = model.RemoveFromWarehouse(tgId, item.CropType, item.Quantity)
		details += fmt.Sprintf("\n%s %s × %d = %s", emoji, name, item.Quantity, farmQuotaStr(value))
	}

	_ = model.IncreaseUserQuota(user.Id, totalValue, true)
	model.AddFarmLog(tgId, "warehouse_sell", totalValue, "仓库全部出售")

	text := fmt.Sprintf("💰 仓库全部出售完成！\n%s\n\n💰 共获得 %s 额度", details, farmQuotaStr(totalValue))
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 抵押贷款 ==========

func showFarmMortgage(chatId int64, editMsgId int, tgId string, from *TgUser) {
	// 检查是否有未还贷款
	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr == nil && activeLoan != nil {
		farmSend(chatId, editMsgId, "❌ 你还有未还清的贷款！请先还清再申请。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏦 返回银行", CallbackData: "farm_bank"}},
			},
		}, from)
		return
	}

	interestRate := common.TgBotFarmMortgageInterestRate
	loanDays := common.TgBotFarmBankMaxLoanDays
	level := model.GetFarmLevel(tgId)

	maxDollar := common.TgBotFarmMortgageMaxAmount / 500000
	text := fmt.Sprintf("🏠 抵押贷款\n\n"+
		"以你的10级升级权力为抵押物\n"+
		"📈 利率: %d%%\n"+
		"📅 还款期限: %d天\n"+
		"💰 可贷金额: $1 ~ $%d\n\n"+
		"⚠️ 注意事项:\n"+
		"• 抵押贷款金额不能用于升级\n"+
		"• 逾期未还: \n",
		interestRate, loanDays, maxDollar)

	if level >= 10 {
		text += "  🚫 你等级≥10，违约将直接封禁平台账号！\n"
	} else {
		text += "  🚫 将永久失去升级到10级的资格\n"
	}

	text += "\n请选择贷款金额："

	var rows [][]TgInlineKeyboardButton
	// 常用金额快捷按钮
	amounts := []int{}
	for _, a := range []int{100, 200, 500, 1000} {
		if a <= maxDollar {
			amounts = append(amounts, a)
		}
	}
	if len(amounts) == 0 || amounts[len(amounts)-1] != maxDollar {
		amounts = append(amounts, maxDollar)
	}
	for _, amt := range amounts {
		principal := common.SafeQuotaMulDiv(amt, 500000, 1) // $1 = 500000 quota
		interest := common.SafeQuotaMulDiv(principal, interestRate, 100)
		total := common.SafeQuotaAdd(principal, interest)
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("$%d（还%s）", amt, farmQuotaStr(total)), CallbackData: fmt.Sprintf("farm_domortgage_%d", amt)},
		})
	}

	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🏦 返回银行", CallbackData: "farm_bank"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmMortgage(chatId int64, editMsgId int, tgId string, amountDollar int, from *TgUser) {
	maxDollar := common.TgBotFarmMortgageMaxAmount / 500000
	if amountDollar < 1 || amountDollar > maxDollar {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 金额必须在 $1 ~ $%d 之间", maxDollar), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏠 返回抵押贷款", CallbackData: "farm_mortgage"}},
			},
		}, from)
		return
	}

	// 检查是否有未还贷款
	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr == nil && activeLoan != nil {
		farmSend(chatId, editMsgId, "❌ 你还有未还清的贷款！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏦 返回银行", CallbackData: "farm_bank"}},
			},
		}, from)
		return
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	principal := common.SafeQuotaMulDiv(amountDollar, 500000, 1) // $1 = 500000 quota
	if principal > common.TgBotFarmMortgageMaxAmount {
		principal = common.TgBotFarmMortgageMaxAmount
	}
	interestRate := common.TgBotFarmMortgageInterestRate
	interest := common.SafeQuotaMulDiv(principal, interestRate, 100)
	totalDue := common.SafeQuotaAdd(principal, interest)
	loanDays := common.TgBotFarmBankMaxLoanDays
	creditScore := model.GetCreditScore(tgId)

	// 创建抵押贷款 (loanType=1)
	loan, err := model.CreateLoanWithType(tgId, principal, interest, totalDue, creditScore, loanDays, 1)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 抵押贷款申请失败", nil, from)
		return
	}

	// 放款
	_ = model.IncreaseUserQuota(user.Id, principal, true)
	model.AddFarmLog(tgId, "loan", principal, fmt.Sprintf("抵押贷款$%d", amountDollar))
	common.SysLog(fmt.Sprintf("TG Farm Mortgage: user %s loan $%d, due $%.2f", tgId, amountDollar, float64(totalDue)/500000.0))

	dueTime := time.Unix(loan.DueAt, 0)
	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 抵押贷款成功！\n\n"+
		"💵 贷款金额: %s\n"+
		"📈 利息: %s (%d%%)\n"+
		"💸 应还总额: %s\n"+
		"📅 还款期限: %s\n\n"+
		"⚠️ 此贷款不能用于升级\n"+
		"⚠️ 逾期未还将执行抵押惩罚",
		farmQuotaStr(principal),
		farmQuotaStr(interest), interestRate,
		farmQuotaStr(totalDue),
		dueTime.Format("2006-01-02")), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🏦 返回银行", CallbackData: "farm_bank"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func farmSend(chatId int64, editMsgId int, text string, keyboard *TgInlineKeyboardMarkup, from *TgUser) {
	if editMsgId > 0 {
		editTgMessage(chatId, editMsgId, text, keyboard, from)
	} else if keyboard != nil {
		sendTgMessageWithKeyboard(chatId, text, *keyboard, from)
	} else {
		sendTgMessage(chatId, text, from)
	}
}
