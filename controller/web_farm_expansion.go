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

var weatherTypes = []struct {
	Type   int
	Name   string
	Emoji  string
	Effect string
}{
	{0, "晴天", "☀️", "作物生长加速20%"},
	{1, "雨天", "🌧️", "自动浇水所有地块"},
	{2, "暴风雨", "⛈️", "事件概率+50%，小心！"},
	{3, "大雾", "🌫️", "偷菜成功率+30%"},
	{4, "下雪", "❄️", "作物生长减速30%（仅冬季）"},
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
	weights := []int{30, 25, 15, 15, 15}
	if season == 3 {
		weights = []int{10, 15, 10, 25, 40}
	} else if season == 1 {
		weights = []int{45, 15, 15, 15, 10}
	}

	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	idx := 0
	acc := 0
	for i, w := range weights {
		acc += w
		if r < acc {
			idx = i
			break
		}
	}

	wt := weatherTypes[idx]
	currentWeather = weatherState{
		Type: wt.Type, Name: wt.Name, Emoji: wt.Emoji,
		Effects: wt.Effect, StartedAt: now, EndsAt: now + duration,
	}
}

func getCurrentSeasonIndex() int {
	daysSinceEpoch := int(time.Now().Unix() / 86400)
	seasonLen := common.TgBotFarmSeasonDays
	if seasonLen <= 0 {
		seasonLen = 7
	}
	return (daysSinceEpoch / seasonLen) % 4
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

	rewards := map[string]float64{"crop": 20, "fish": 30, "meat": 15, "recipe": 50}
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
		cropKeys = append(cropKeys, struct{ k, n, e string }{"crop_" + cr.Key, cr.Name, cr.Emoji})
	}
	categories = append(categories, buildCat("crop", "作物", cropKeys))

	var fishKeys []struct{ k, n, e string }
	for _, f := range fishTypes {
		fishKeys = append(fishKeys, struct{ k, n, e string }{"fish_" + f.Key, f.Name, f.Emoji})
	}
	categories = append(categories, buildCat("fish", "鱼类", fishKeys))

	var meatKeys []struct{ k, n, e string }
	for _, a := range ranchAnimals {
		meatKeys = append(meatKeys, struct{ k, n, e string }{"meat_" + a.Key, a.Name, a.Emoji})
	}
	categories = append(categories, buildCat("meat", "肉类", meatKeys))

	var recipeKeys []struct{ k, n, e string }
	for _, r := range recipes {
		recipeKeys = append(recipeKeys, struct{ k, n, e string }{"recipe_" + r.Key, r.Name, r.Emoji})
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
	var req struct {
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	rewards := map[string]int{"crop": 10000000, "fish": 15000000, "meat": 7500000, "recipe": 25000000, "grand": 50000000}
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"current_level":   level,
			"prestige_level":  prestige,
			"min_level":       common.TgBotFarmPrestigeMinLevel,
			"can_prestige":    level >= common.TgBotFarmPrestigeMinLevel,
			"bonus_per_level": common.TgBotFarmPrestigeBonusPerLevel,
			"current_bonus":   bonus,
			"next_bonus":      bonus + common.TgBotFarmPrestigeBonusPerLevel,
		},
	})
}

func WebFarmPrestige(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	level := model.GetFarmLevel(tgId)
	if level < common.TgBotFarmPrestigeMinLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "等级不足，需要满级才能转生"})
		return
	}
	loan, _ := model.GetActiveLoan(tgId)
	if loan != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先还清贷款"})
		return
	}

	oldPrestige := model.GetPrestigeLevel(tgId)
	newPrestige := oldPrestige + 1
	model.ResetFarmForPrestige(tgId)
	model.SetPrestigeLevel(tgId, newPrestige)
	model.CreatePrestigeRecord(tgId, newPrestige)

	prestigeReward := 25000000
	model.IncreaseUserQuota(user.Id, prestigeReward, true)
	model.AddFarmLog(tgId, "prestige", prestigeReward, fmt.Sprintf("🔄 转生到第%d世", newPrestige))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("转生成功！永久收入加成+%d%%", newPrestige*common.TgBotFarmPrestigeBonusPerLevel),
	})
}

// ========== Automation ==========

func WebFarmAutomationView(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
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
		{"autofeeder", "自动喂食器", "🌾", "自动喂食牧场动物", autoMap["autofeeder"], webFarmQuotaFloat(common.TgBotFarmAutoFeederPrice)},
		{"scarecrow", "稻草人", "🧑‍🌾", "阻挡30%偷菜尝试", autoMap["scarecrow"], webFarmQuotaFloat(common.TgBotFarmScarecrowPrice)},
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func WebFarmAutomationBuy(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	prices := map[string]int{"irrigation": common.TgBotFarmIrrigationPrice, "autofeeder": common.TgBotFarmAutoFeederPrice, "scarecrow": common.TgBotFarmScarecrowPrice}
	names := map[string]string{"irrigation": "灌溉系统", "autofeeder": "自动喂食器", "scarecrow": "稻草人"}

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
		{0, 20, "$0"}, {250000, 25, "$0.50"}, {500000, 20, "$1"}, {1000000, 15, "$2"},
		{2500000, 10, "$5"}, {5000000, 5, "$10"}, {10000000, 3, "$20"}, {25000000, 2, "$50"},
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
	model.CreateGameLog(tgId, "wheel", price, actualWin)
	model.AddFarmLog(tgId, "game", net, "🎡 转盘: "+win.Label)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"sector_index": winIdx, "prize_label": win.Label,
		"prize_amount": webFarmQuotaFloat(actualWin), "net": webFarmQuotaFloat(net),
	}})
}

func WebFarmGameScratch(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
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
		{125000, 30, "🍒", "$0.25"}, {500000, 25, "🍋", "$1"}, {1250000, 20, "🍊", "$2.50"},
		{2500000, 15, "🍇", "$5"}, {12500000, 8, "💎", "$25"}, {50000000, 2, "👑", "$100"},
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
	grid[0][0], grid[0][1], grid[0][2] = win.Symbol, win.Symbol, win.Symbol

	actualWin := win.Amount
	prestige := model.GetPrestigeLevel(tgId)
	if prestige > 0 && actualWin > 0 {
		actualWin = actualWin + actualWin*prestige*common.TgBotFarmPrestigeBonusPerLevel/100
	}
	if actualWin > 0 {
		model.IncreaseUserQuota(user.Id, actualWin, true)
	}
	net := actualWin - price
	model.CreateGameLog(tgId, "scratch", price, actualWin)
	model.AddFarmLog(tgId, "game", net, "🎰 刮刮卡: "+win.Label)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"grid": grid, "win_symbol": win.Symbol, "prize_label": win.Label,
		"prize_amount": webFarmQuotaFloat(actualWin), "net": webFarmQuotaFloat(net),
	}})
}

func WebFarmGameHistory(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
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

func init() {
	initWeather()
}
