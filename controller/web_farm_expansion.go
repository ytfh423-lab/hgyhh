package controller

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ========== Weather System ==========

type weatherState struct {
	Type      int    `json:"type"`
	TypeKey   string `json:"type_key"`
	Name      string `json:"name"`
	Emoji     string `json:"emoji"`
	Effects   string `json:"effects"`
	StartedAt int64  `json:"started_at"`
	EndsAt    int64  `json:"ends_at"`
}

var (
	currentWeather weatherState
	weatherMu      sync.RWMutex
)

type weatherDef struct {
	Type    int
	TypeKey string // frontend 3D view uses this key
	Name    string
	Emoji   string
	Effect  string
}

// 0=春 1=夏 2=秋 3=冬 各季节可用天气及权重
var seasonWeatherPool = map[int][]struct {
	Def    weatherDef
	Weight int
}{
	0: { // 春：多雨温和
		{weatherDef{0, "sunny", "晴天", "☀️", "作物生长加速20%"}, 65},
		{weatherDef{1, "rainy", "春雨", "🌧️", "自动浇水所有地块"}, 15},
		{weatherDef{2, "stormy", "雷阵雨", "⛈️", "事件概率+50%，小心！"}, 5},
		{weatherDef{3, "foggy", "薄雾", "🌫️", "偷菜成功率+30%"}, 8},
		{weatherDef{5, "windy", "微风", "🍃", "作物生长加速10%"}, 7},
	},
	1: { // 夏：炎热多晴，可暴风雨，绝不下雪
		{weatherDef{0, "sunny", "烈日", "☀️", "作物生长加速20%"}, 55},
		{weatherDef{1, "rainy", "阵雨", "🌧️", "自动浇水所有地块"}, 12},
		{weatherDef{2, "stormy", "暴风雨", "⛈️", "事件概率+50%，小心！"}, 8},
		{weatherDef{6, "hot", "酷暑", "🔥", "作物需水量增加，注意浇水"}, 20},
		{weatherDef{5, "windy", "微风", "🍃", "作物生长加速10%"}, 5},
	},
	2: { // 秋：多雾多风，偶尔雨
		{weatherDef{0, "sunny", "晴天", "☀️", "作物生长加速20%"}, 60},
		{weatherDef{1, "rainy", "秋雨", "🌧️", "自动浇水所有地块"}, 10},
		{weatherDef{3, "foggy", "浓雾", "🌫️", "偷菜成功率+30%"}, 10},
		{weatherDef{5, "windy", "秋风", "🍂", "作物生长加速10%"}, 15},
		{weatherDef{2, "stormy", "暴风雨", "⛈️", "事件概率+50%，小心！"}, 5},
	},
	3: { // 冬：下雪为主，寒冷
		{weatherDef{4, "snowy", "大雪", "❄️", "作物生长减速30%"}, 25},
		{weatherDef{7, "snowy", "小雪", "🌨️", "作物生长减速15%"}, 20},
		{weatherDef{0, "sunny", "冬日暖阳", "☀️", "作物生长加速20%"}, 35},
		{weatherDef{3, "foggy", "寒雾", "🌫️", "偷菜成功率+30%"}, 5},
		{weatherDef{8, "cold", "寒潮", "🥶", "事件概率+30%，作物生长减速20%"}, 15},
	},
}

func initWeather() {
	weatherMu.Lock()
	defer weatherMu.Unlock()
	pickNewWeather()
}

func pickNewWeather() {
	now := time.Now().Unix()
	minD := int64(common.TgBotFarmWeatherDurationMin)
	maxD := int64(common.TgBotFarmWeatherDurationMax)
	duration := minD + rand.Int63n(maxD-minD+1)

	season := getCurrentSeasonIndex()
	pool, ok := seasonWeatherPool[season]
	if !ok {
		pool = seasonWeatherPool[0]
	}

	total := 0
	for _, entry := range pool {
		total += entry.Weight
	}
	r := rand.Intn(total)
	acc := 0
	chosen := pool[0].Def
	for _, entry := range pool {
		acc += entry.Weight
		if r < acc {
			chosen = entry.Def
			break
		}
	}

	currentWeather = weatherState{
		Type: chosen.Type, TypeKey: chosen.TypeKey, Name: chosen.Name, Emoji: chosen.Emoji,
		Effects: chosen.Effect, StartedAt: now, EndsAt: now + duration,
	}
}

func getCurrentSeasonIndex() int {
	return getCurrentSeason()
}

func RefreshWeatherIfNeeded() {
	weatherMu.Lock()
	defer weatherMu.Unlock()
	if time.Now().Unix() >= currentWeather.EndsAt {
		pickNewWeather()
	}
}

func GetCurrentWeather() weatherState {
	RefreshWeatherIfNeeded()
	weatherMu.RLock()
	defer weatherMu.RUnlock()
	return currentWeather
}

func GetWeatherGrowthMultiplier() float64 {
	w := GetCurrentWeather()
	switch w.Type {
	case 0:
		return 1.0 - float64(common.TgBotFarmWeatherSunnyGrowthBonus)/100.0
	case 4:
		if getCurrentSeasonIndex() == 3 {
			return 1.3
		}
	}
	return 1.0
}

func GetWeatherEventBonus() int {
	w := GetCurrentWeather()
	if w.Type == 2 {
		return common.TgBotFarmWeatherStormyEventBonus
	}
	return 0
}

func GetWeatherStealBonus() int {
	w := GetCurrentWeather()
	if w.Type == 3 {
		return common.TgBotFarmWeatherFoggyStealBonus
	}
	return 0
}

// ========== Lucky Events ==========

type luckyEventResult struct {
	Triggered  bool    `json:"triggered"`
	EventType  string  `json:"event_type"`
	EventName  string  `json:"event_name"`
	EventEmoji string  `json:"event_emoji"`
	Message    string  `json:"message"`
	BonusAmount float64 `json:"bonus_amount,omitempty"`
}

func RollLuckyEvent() luckyEventResult {
	r := rand.Intn(100)
	if r < 5 {
		return luckyEventResult{Triggered: true, EventType: "golden_harvest", EventName: "黄金丰收", EventEmoji: "✨", Message: "黄金丰收！产量翻倍！"}
	} else if r < 8 {
		return luckyEventResult{Triggered: true, EventType: "lucky_rain", EventName: "幸运雨", EventEmoji: "🌈", Message: "幸运雨降临！所有地块自动浇水！"}
	} else if r < 10 {
		return luckyEventResult{Triggered: true, EventType: "merchant", EventName: "神秘商人", EventEmoji: "🧙", Message: "神秘商人来访！下次购物半价！"}
	} else if r < 14 {
		return luckyEventResult{Triggered: true, EventType: "seed_gift", EventName: "种子礼物", EventEmoji: "🎁", Message: "收到种子礼物！获得$1奖励！"}
	}
	return luckyEventResult{Triggered: false}
}

func ApplyLuckyEvent(tgId string, event luckyEventResult) {
	switch event.EventType {
	case "lucky_rain":
		now := time.Now().Unix()
		model.DB.Model(&model.TgFarmPlot{}).Where("telegram_id = ? AND status = 1", tgId).Update("last_watered_at", now)
	case "merchant":
		_ = model.IncrementFarmItem(tgId, "_merchant_discount", 1)
	case "seed_gift":
		giftAmount := 500000
		model.DB.Model(&model.User{}).Where("telegram_id = ?", tgId).Update("quota", model.DB.Raw("quota + ?", giftAmount))
		model.AddFarmLog(tgId, "lucky_event", giftAmount, "🎁 种子礼物奖励")
	}
}

// ========== Encyclopedia ==========

func WebFarmEncyclopedia(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockEncyclopedia, "图鉴") {
		return
	}

	backfillCollections(tgId)

	collections, _ := model.GetCollections(tgId)
	collMap := make(map[string]map[string]int64)
	for _, col := range collections {
		if collMap[col.Category] == nil {
			collMap[col.Category] = make(map[string]int64)
		}
		collMap[col.Category][col.ItemKey] = col.FirstAt
	}

	type itemInfo struct {
		Key      string `json:"key"`
		Name     string `json:"name"`
		Emoji    string `json:"emoji"`
		Unlocked bool   `json:"unlocked"`
		FirstAt  int64  `json:"first_at,omitempty"`
	}
	type categoryInfo struct {
		Key      string     `json:"key"`
		Name     string     `json:"name"`
		Items    []itemInfo `json:"items"`
		Unlocked int        `json:"unlocked"`
		Total    int        `json:"total"`
		Complete bool       `json:"complete"`
		Claimed  bool       `json:"claimed"`
		Reward   float64    `json:"reward"`
	}

	rewards := map[string]float64{"crop": 20, "fish": 30, "animal": 15, "recipe": 50}
	var categories []categoryInfo

	buildCat := func(key, name string, keys []struct{ k, n, e string }) categoryInfo {
		var items []itemInfo
		unlocked := 0
		for _, it := range keys {
			firstAt, found := collMap[key][it.k]
			items = append(items, itemInfo{Key: it.k, Name: it.n, Emoji: it.e, Unlocked: found, FirstAt: firstAt})
			if found {
				unlocked++
			}
		}
		return categoryInfo{
			Key: key, Name: name, Items: items, Unlocked: unlocked, Total: len(items),
			Complete: unlocked == len(items), Claimed: model.HasCollectionReward(tgId, key), Reward: rewards[key],
		}
	}

	var cropKeys []struct{ k, n, e string }
	for _, cr := range farmCrops {
		cropKeys = append(cropKeys, struct{ k, n, e string }{cr.Key, cr.Name, cr.Emoji})
	}
	categories = append(categories, buildCat("crop", "作物", cropKeys))

	var fishKeys []struct{ k, n, e string }
	for _, f := range fishTypes {
		fishKeys = append(fishKeys, struct{ k, n, e string }{f.Key, f.Name, f.Emoji})
	}
	categories = append(categories, buildCat("fish", "鱼类", fishKeys))

	var meatKeys []struct{ k, n, e string }
	for _, a := range ranchAnimals {
		meatKeys = append(meatKeys, struct{ k, n, e string }{a.Key, a.Name, a.Emoji})
	}
	categories = append(categories, buildCat("animal", "肉类", meatKeys))

	var recipeKeys []struct{ k, n, e string }
	for _, r := range recipes {
		recipeKeys = append(recipeKeys, struct{ k, n, e string }{r.Key, r.Name, r.Emoji})
	}
	categories = append(categories, buildCat("recipe", "加工品", recipeKeys))

	totalUnlocked, totalItems := 0, 0
	for _, cat := range categories {
		totalUnlocked += cat.Unlocked
		totalItems += cat.Total
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"categories":     categories,
			"total_unlocked": totalUnlocked,
			"total_items":    totalItems,
			"all_complete":   totalUnlocked == totalItems,
			"grand_claimed":  model.HasCollectionReward(tgId, "grand"),
			"grand_reward":   100.0,
		},
	})
}

func WebFarmClaimCollection(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockEncyclopedia, "图鉴") {
		return
	}
	var req struct {
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	rewards := map[string]int{"crop": 10000000, "fish": 15000000, "animal": 7500000, "recipe": 25000000, "grand": 50000000}
	reward, exists := rewards[req.Category]
	if !exists {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效分类"})
		return
	}
	if model.HasCollectionReward(tgId, req.Category) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "已领取过该奖励"})
		return
	}

	prestige := model.GetPrestigeLevel(tgId)
	if prestige > 0 {
		reward = reward + reward*prestige*common.TgBotFarmPrestigeBonusPerLevel/100
	}

	_ = model.ClaimCollectionReward(tgId, req.Category)
	model.IncreaseUserQuota(user.Id, reward, true)
	model.AddFarmLog(tgId, "encyclopedia", reward, "📖 图鉴奖励: "+req.Category)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "领取成功！"})
}

// ========== Prestige ==========

func WebFarmPrestigeInfo(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	level := model.GetFarmLevel(tgId)
	prestige := model.GetPrestigeLevel(tgId)
	bonus := prestige * common.TgBotFarmPrestigeBonusPerLevel
	nextBonus := bonus + common.TgBotFarmPrestigeBonusPerLevel
	maxBonus := common.TgBotFarmPrestigeMaxTimes * common.TgBotFarmPrestigeBonusPerLevel
	if nextBonus > maxBonus {
		nextBonus = maxBonus
	}
	nextPrice := model.GetPrestigePrice(prestige + 1)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"current_level":   level,
			"prestige_level":  prestige,
			"min_level":       common.TgBotFarmPrestigeMinLevel,
			"max_times":       common.TgBotFarmPrestigeMaxTimes,
			"can_prestige":    level >= common.TgBotFarmPrestigeMinLevel && prestige < common.TgBotFarmPrestigeMaxTimes,
			"bonus_per_level": common.TgBotFarmPrestigeBonusPerLevel,
			"current_bonus":   bonus,
			"next_bonus":      nextBonus,
			"next_price":      webFarmQuotaFloat(nextPrice),
			"reset_balance":   10.0,
		},
	})
}

func WebFarmPrestige(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	level := model.GetFarmLevel(tgId)
	currentPrestige := model.GetPrestigeLevel(tgId)
	if level < common.TgBotFarmPrestigeMinLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "等级不足，需要满级才能转生"})
		return
	}
	if currentPrestige >= common.TgBotFarmPrestigeMaxTimes {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("最多只能转生%d次", common.TgBotFarmPrestigeMaxTimes)})
		return
	}
	loan, _ := model.GetActiveLoan(tgId)
	if loan != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先还清贷款"})
		return
	}
	price := model.GetPrestigePrice(currentPrestige + 1)
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足，本次转生需要%.2f", webFarmQuotaFloat(price))})
		return
	}

	oldPrestige := currentPrestige
	newPrestige := oldPrestige + 1
	model.DecreaseUserQuota(user.Id, price)
	model.ResetFarmForPrestige(user.Id, tgId)
	model.SetPrestigeLevel(tgId, newPrestige)
	model.CreatePrestigeRecord(tgId, newPrestige)
	model.AddFarmLog(tgId, "prestige", -price, fmt.Sprintf("🔄 转生到第%d世，支付%s，余额重置为10", newPrestige, farmQuotaStr(price)))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("转生成功：已支付%.2f，余额已重置为10，仅保留成就和图鉴，永久收入加成+%d%%", webFarmQuotaFloat(price), newPrestige*common.TgBotFarmPrestigeBonusPerLevel),
	})
}

// ========== Automation ==========

func WebFarmAutomationView(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockAutomation, "自动化") {
		return
	}
	autos, _ := model.GetAutomations(tgId)
	autoMap := make(map[string]bool)
	for _, a := range autos {
		autoMap[a.Type] = true
	}

	type autoInfo struct {
		Type      string  `json:"type"`
		Name      string  `json:"name"`
		Emoji     string  `json:"emoji"`
		Desc      string  `json:"desc"`
		Installed bool    `json:"installed"`
		Price     float64 `json:"price"`
	}

	items := []autoInfo{
		{"irrigation", "灌溉系统", "💧", "自动浇水所有地块", autoMap["irrigation"], webFarmQuotaFloat(common.TgBotFarmIrrigationPrice)},
		{"auto_feeder", "自动喂食器", "🌾", "自动喂食牧场动物", autoMap["auto_feeder"], webFarmQuotaFloat(common.TgBotFarmAutoFeederPrice)},
		{"scarecrow", "稻草人", "🧑‍🌾", "阻挡30%偷菜尝试", autoMap["scarecrow"], webFarmQuotaFloat(common.TgBotFarmScarecrowPrice)},
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func WebFarmAutomationBuy(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockAutomation, "自动化") {
		return
	}
	var req struct {
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	prices := map[string]int{"irrigation": common.TgBotFarmIrrigationPrice, "auto_feeder": common.TgBotFarmAutoFeederPrice, "scarecrow": common.TgBotFarmScarecrowPrice}
	names := map[string]string{"irrigation": "灌溉系统", "auto_feeder": "自动喂食器", "scarecrow": "稻草人"}

	price, exists := prices[req.Type]
	if !exists {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效设施类型"})
		return
	}
	if model.HasAutomation(tgId, req.Type) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "已安装该设施"})
		return
	}
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "余额不足"})
		return
	}

	model.DecreaseUserQuota(user.Id, price)
	_ = model.CreateAutomation(tgId, req.Type)
	model.AddFarmLog(tgId, "automation", -price, "🔧 安装"+names[req.Type])
	c.JSON(http.StatusOK, gin.H{"success": true, "message": names[req.Type] + "安装成功！"})
}

// ========== Leaderboard ==========

func WebFarmLeaderboard(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockLeaderboard, "排行榜") {
		return
	}
	boardType := c.DefaultQuery("type", "balance")
	entries, err := model.GetFarmLeaderboard(boardType, 20)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "查询失败"})
		return
	}
	myRank := model.GetFarmRank(tgId, boardType)

	type rankItem struct {
		Rank  int     `json:"rank"`
		Name  string  `json:"name"`
		Value float64 `json:"value"`
		IsMe  bool    `json:"is_me"`
	}
	var items []rankItem
	for i, e := range entries {
		val := float64(e.Value)
		if boardType == "balance" {
			val = float64(e.Value) / 500000.0
		}
		name := e.Username
		if name == "" && len(e.TelegramId) > 6 {
			name = e.TelegramId[:6] + "..."
		}
		items = append(items, rankItem{Rank: i + 1, Name: name, Value: val, IsMe: e.TelegramId == tgId})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"type": boardType, "items": items, "my_rank": myRank}})
}

// ========== Mini-Games ==========

func WebFarmGameWheel(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockGames, "小游戏") {
		return
	}
	price := common.TgBotFarmWheelPrice
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足，需要$%.2f", float64(price)/500000.0)})
		return
	}
	model.DecreaseUserQuota(user.Id, price)

	type sector struct {
		Prize  int
		Weight int
		Label  string
	}
	sectors := []sector{
		{0, 40, "$0"}, {250000, 25, "$0.50"}, {500000, 16, "$1"}, {750000, 9, "$1.50"},
		{1000000, 5, "$2"}, {1500000, 3, "$3"}, {2500000, 1, "$5"}, {5000000, 1, "$10"},
	}
	totalW := 0
	for _, s := range sectors {
		totalW += s.Weight
	}
	r := rand.Intn(totalW)
	acc, winIdx := 0, 0
	for i, s := range sectors {
		acc += s.Weight
		if r < acc {
			winIdx = i
			break
		}
	}
	win := sectors[winIdx]
	actualWin := win.Prize
	prestige := model.GetPrestigeLevel(tgId)
	if prestige > 0 && actualWin > 0 {
		actualWin = actualWin + actualWin*prestige*common.TgBotFarmPrestigeBonusPerLevel/100
	}
	if actualWin > 0 {
		model.IncreaseUserQuota(user.Id, actualWin, true)
	}
	net := actualWin - price
	prizeLabel := win.Label
	if prestige > 0 && actualWin > 0 {
		prizeLabel = fmt.Sprintf("$%.2f", webFarmQuotaFloat(actualWin))
	}
	model.CreateGameLog(tgId, "wheel", price, actualWin)
	model.AddFarmLog(tgId, "game", net, "🎡 转盘: "+prizeLabel)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"sector_index": winIdx, "prize_label": prizeLabel,
		"prize_amount": webFarmQuotaFloat(actualWin), "net": webFarmQuotaFloat(net),
	}})
}

func WebFarmGameScratch(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockGames, "小游戏") {
		return
	}
	price := common.TgBotFarmScratchPrice
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "余额不足"})
		return
	}
	model.DecreaseUserQuota(user.Id, price)

	type prize struct {
		Amount int
		Weight int
		Symbol string
		Label  string
	}
	prizes := []prize{
		{250000, 45, "🍒", "$0.50"}, {375000, 27, "🍋", "$0.75"}, {500000, 15, "🍊", "$1"},
		{750000, 9, "🍇", "$1.50"}, {1250000, 3, "💎", "$2.50"}, {2500000, 1, "👑", "$5"},
	}
	totalW := 0
	for _, p := range prizes {
		totalW += p.Weight
	}
	r := rand.Intn(totalW)
	acc, winIdx := 0, 0
	for i, p := range prizes {
		acc += p.Weight
		if r < acc {
			winIdx = i
			break
		}
	}
	win := prizes[winIdx]

	grid := make([][]string, 3)
	for row := 0; row < 3; row++ {
		grid[row] = make([]string, 3)
		for col := 0; col < 3; col++ {
			grid[row][col] = prizes[rand.Intn(len(prizes))].Symbol
		}
	}

	// 20% win chance
	actualWin := 0
	if rand.Intn(100) < 20 {
		grid[0][0], grid[0][1], grid[0][2] = win.Symbol, win.Symbol, win.Symbol
		actualWin = win.Amount
	} else {
		win = prizes[0] // for label display
	}
	prestige := model.GetPrestigeLevel(tgId)
	if prestige > 0 && actualWin > 0 {
		actualWin = actualWin + actualWin*prestige*common.TgBotFarmPrestigeBonusPerLevel/100
	}
	if actualWin > 0 {
		model.IncreaseUserQuota(user.Id, actualWin, true)
	}
	net := actualWin - price
	model.CreateGameLog(tgId, "scratch", price, actualWin)
	if actualWin > 0 {
		model.AddFarmLog(tgId, "game", net, "🎰 刮刮卡: "+win.Label)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
			"grid": grid, "win_symbol": win.Symbol, "prize_label": win.Label,
			"prize_amount": webFarmQuotaFloat(actualWin), "net": webFarmQuotaFloat(net),
		}})
	} else {
		model.AddFarmLog(tgId, "game", net, "🎰 刮刮卡: 未中奖")
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
			"grid": grid, "win_symbol": "😢", "prize_label": "未中奖",
			"prize_amount": 0, "net": webFarmQuotaFloat(net),
		}})
	}
}

func WebFarmGameHistory(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockGames, "小游戏") {
		return
	}
	logs, _ := model.GetRecentGameLogs(tgId, 20)
	type logItem struct {
		GameType string  `json:"game_type"`
		Bet      float64 `json:"bet"`
		Win      float64 `json:"win"`
		Net      float64 `json:"net"`
		Time     int64   `json:"time"`
	}
	var items []logItem
	for _, l := range logs {
		items = append(items, logItem{
			GameType: l.GameType, Bet: webFarmQuotaFloat(l.BetAmount),
			Win: webFarmQuotaFloat(l.WinAmount), Net: webFarmQuotaFloat(l.WinAmount - l.BetAmount), Time: l.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func WebFarmGameList(c *gin.Context) {
	type gameItem struct {
		Key   string  `json:"key"`
		Name  string  `json:"name"`
		Emoji string  `json:"emoji"`
		Desc  string  `json:"desc"`
		Price float64 `json:"price"`
	}
	var items []gameItem
	for _, g := range miniGames {
		items = append(items, gameItem{
			Key: g.Key, Name: g.Name, Emoji: g.Emoji,
			Desc: g.Desc, Price: webFarmQuotaFloat(g.Price),
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func WebFarmGamePlay(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockGames, "小游戏") {
		return
	}

	var req struct {
		GameKey string  `json:"game_key"`
		Score   float64 `json:"score"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.GameKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "缺少 game_key"})
		return
	}
	// clamp score to [0,1]
	if req.Score < 0 {
		req.Score = 0
	}
	if req.Score > 1 {
		req.Score = 1
	}

	g := miniGameMap[req.GameKey]
	if g == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未知游戏"})
		return
	}

	price := g.Price
	if int64(user.Quota) < int64(price) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足，需要 %s", farmQuotaStr(price))})
		return
	}
	model.DecreaseUserQuota(user.Id, price)

	var resultText string
	var multi float64

	// 有前端引擎的游戏：用前端传来的 score 决定倍率
	engineGames := map[string]bool{
		"horserace": true, "woodchop": true, "weed": true, "milking": true, "thresh": true,
		"fishcomp": true, "harvest": true, "lasso": true, "pullcarrot": true, "seedling": true,
	}

	if engineGames[req.GameKey] {
		resultText, multi = scoreToGameResult(req.GameKey, req.Score, g.Name, g.Emoji)
	} else {
		switch req.GameKey {
		case "bugcatch":
			resultText, multi = playBugCatch()
		case "egghunt":
			resultText, multi = playEggHunt()
		case "sunflower":
			resultText, multi = playSunflower()
		case "beekeep":
			resultText, multi = playBeekeep()
		case "fruitpick":
			resultText, multi = playFruitPick()
		case "sheepcount":
			resultText, multi = playSheepCount()
		case "cornrace":
			resultText, multi = playCornRace()
		case "rooster":
			resultText, multi = playRooster()
		case "sheepdog":
			resultText, multi = playSheepdog()
		case "pumpkin":
			resultText, multi = playPumpkinContest()
		case "pigchase":
			resultText, multi = playPigChase()
		case "duckherd":
			resultText, multi = playDuckHerd()
		case "grape":
			resultText, multi = playGrapeStomp()
		case "mushroom":
			resultText, multi = playMushroom()
		case "hatchegg":
			resultText, multi = playHatchEgg()
		case "weather":
			resultText, multi = playWeather()
		case "produce":
			resultText, multi = playProduce()
		case "tame":
			resultText, multi = playTame()
		case "scarecrow":
			resultText, multi = playScarecrow()
		case "foxhunt":
			resultText, multi = playFoxHunt()
		default:
			model.IncreaseUserQuota(user.Id, price, true)
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未知游戏"})
			return
		}
	}

	actualWin := int(float64(price) * multi)
	prestige := model.GetPrestigeLevel(tgId)
	if prestige > 0 && actualWin > 0 {
		actualWin = actualWin + actualWin*prestige*common.TgBotFarmPrestigeBonusPerLevel/100
	}
	if actualWin > 0 {
		model.IncreaseUserQuota(user.Id, actualWin, true)
	}

	net := actualWin - price
	model.CreateGameLog(tgId, req.GameKey, price, actualWin)
	model.AddFarmLog(tgId, "game", net, fmt.Sprintf("%s %s", g.Emoji, g.Name))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"game_key":    req.GameKey,
			"game_name":   g.Name,
			"game_emoji":  g.Emoji,
			"result_text": resultText,
			"bet":         webFarmQuotaFloat(price),
			"win":         webFarmQuotaFloat(actualWin),
			"net":         webFarmQuotaFloat(net),
			"multi":       multi,
		},
	})
}

// ========== Farm Announcement ==========

func WebFarmAnnouncement(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	enabled := common.OptionMap["FarmAnnouncementEnabled"]
	text := common.OptionMap["FarmAnnouncementText"]
	aType := common.OptionMap["FarmAnnouncementType"]
	common.OptionMapRWMutex.RUnlock()

	if enabled != "true" || text == "" {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": nil})
		return
	}
	if aType == "" {
		aType = "info"
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"text":    text,
		"type":    aType,
		"enabled": true,
	}})
}

// ========== Farm Group Config ==========

func WebFarmGroupConfig(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	enabled := common.OptionMap["FarmGroupEnabled"]
	link := common.OptionMap["FarmGroupLink"]
	common.OptionMapRWMutex.RUnlock()

	if enabled != "true" || link == "" {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"enabled": true,
		"link":    link,
	}})
}

func init() {
	initWeather()
}
