package controller

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ========== 市场引擎：多因素驱动价格系统 ==========
//
// 价格公式:
//   finalMultiplier = prevMultiplier
//     + seasonDelta      (季节因素推力)
//     + supplyDemandDelta (供需推力)
//     + trendDelta        (趋势惯性)
//     + eventDelta        (事件推力)
//     + meanRevDelta      (均值回归拉力)
//     + noiseDelta        (微扰噪声)
//   clamped to [minMult, maxMult] with maxChangePerTick limit

// ========== 商品市场配置 ==========

// marketItemConfig 每个商品的市场参数
type marketItemConfig struct {
	Key              string
	Name             string
	Emoji            string
	Category         string // crop/fish/meat/recipe/wood
	BasePrice        int    // 基础价格(quota)
	MinMultiplier    int    // 最低倍率% (default 40)
	MaxMultiplier    int    // 最高倍率% (default 220)
	Volatility       int    // 波动等级 1-5 (影响噪声和变化幅度)
	TrendStrength    int    // 趋势惯性 0-100 (默认30，越大趋势越持久)
	MeanRevStrength  int    // 均值回归强度 0-100 (默认15，越大回归越快)
	SupplySensitivity int   // 供需敏感度 0-100 (默认50)
	SeasonProfile    [4]int // 春夏秋冬的季节因子 (100=无影响, 80=降价, 120=涨价)
}

// marketItemState 每个商品的实时状态
type marketItemState struct {
	Multiplier     int   // 当前倍率%
	PrevMultiplier int   // 上次倍率%
	Trend          int   // 趋势方向: 正=涨势, 负=跌势, 0=平
	LastSeasonF    int   // 上次季节因子(百分比偏移)
	LastSupplyF    int   // 上次供需因子
	LastTrendF     int   // 上次趋势因子
	LastEventF     int   // 上次事件因子
	LastNoiseF     int   // 上次噪声因子
	LastMeanRevF   int   // 上次均值回归因子
	TickCount      int   // 已经过的tick次数
}

var (
	mktConfigs   map[string]*marketItemConfig
	mktStates    map[string]*marketItemState
	mktMu        sync.RWMutex
	mktLastTick  int64
	mktNextTick  int64
	mktHistory   []marketSnapshot // 保持与旧系统兼容的快照格式
	mktTips      []marketTip      // 当前市场情报
	mktInitOnce  sync.Once
)

const mktHistoryMaxLen = 72 // 保留最近72次快照

type marketTip struct {
	Icon    string `json:"icon"`
	Text    string `json:"text"`
	ItemKey string `json:"item_key,omitempty"`
	Cat     string `json:"category,omitempty"`
	Type    string `json:"type"` // season/event/trend/supply
}

// ========== 初始化 ==========

func initMarketEngine() {
	mktInitOnce.Do(func() {
		mktConfigs = make(map[string]*marketItemConfig)
		mktStates = make(map[string]*marketItemState)

		// 注册所有商品
		registerCropMarketItems()
		registerFishMarketItems()
		registerMeatMarketItems()
		registerRecipeMarketItems()
		registerWoodMarketItems()

		// 初始化状态 - 从DB加载最近价格或用100%
		loadInitialPrices()

		// 执行首次tick
		doMarketTick()
	})
}

func registerCropMarketItems() {
	for _, c := range farmCrops {
		profile := defaultSeasonProfile(c.Season)
		mktConfigs["crop_"+c.Key] = &marketItemConfig{
			Key: "crop_" + c.Key, Name: c.Name, Emoji: c.Emoji,
			Category: "crop", BasePrice: c.UnitPrice,
			MinMultiplier: 45, MaxMultiplier: 200,
			Volatility: 2, TrendStrength: 25,
			MeanRevStrength: 12, SupplySensitivity: 60,
			SeasonProfile: profile,
		}
	}
}

func registerFishMarketItems() {
	// 鱼类波动大，供需敏感度低（随机性来自稀有度）
	for _, f := range fishTypes {
		vol := 2
		if f.Rarity == "稀有" || f.Rarity == "史诗" {
			vol = 3
		}
		if f.Rarity == "传说" {
			vol = 4
		}
		mktConfigs["fish_"+f.Key] = &marketItemConfig{
			Key: "fish_" + f.Key, Name: f.Name, Emoji: f.Emoji,
			Category: "fish", BasePrice: f.SellPrice,
			MinMultiplier: 50, MaxMultiplier: 210,
			Volatility: vol, TrendStrength: 20,
			MeanRevStrength: 18, SupplySensitivity: 30,
			SeasonProfile: [4]int{100, 105, 95, 100},
		}
	}
}

func registerMeatMarketItems() {
	for _, a := range ranchAnimals {
		mktConfigs["meat_"+a.Key] = &marketItemConfig{
			Key: "meat_" + a.Key, Name: a.Name + "肉", Emoji: a.Emoji,
			Category: "meat", BasePrice: *a.MeatPrice,
			MinMultiplier: 55, MaxMultiplier: 190,
			Volatility: 2, TrendStrength: 30,
			MeanRevStrength: 15, SupplySensitivity: 55,
			SeasonProfile: [4]int{100, 95, 105, 110},
		}
	}
}

func registerRecipeMarketItems() {
	for _, r := range recipes {
		mktConfigs["recipe_"+r.Key] = &marketItemConfig{
			Key: "recipe_" + r.Key, Name: r.Name, Emoji: r.Emoji,
			Category: "recipe", BasePrice: r.SellPrice,
			MinMultiplier: 60, MaxMultiplier: 180,
			Volatility: 1, TrendStrength: 35,
			MeanRevStrength: 10, SupplySensitivity: 45,
			SeasonProfile: [4]int{100, 100, 100, 100},
		}
	}
}

func registerWoodMarketItems() {
	for _, tp := range treeProducts {
		// 木材类受季节影响: 春秋建造季需求高
		profile := [4]int{110, 95, 110, 90}
		// 水果类树产品用水果季节逻辑
		if tp.Key == "apple" || tp.Key == "cherry" {
			profile = [4]int{95, 80, 110, 120}
		}
		if tp.Key == "bamboo_shoot" {
			profile = [4]int{80, 95, 100, 115}
		}
		mktConfigs["wood_"+tp.Key] = &marketItemConfig{
			Key: "wood_" + tp.Key, Name: tp.Name, Emoji: tp.Emoji,
			Category: "wood", BasePrice: tp.BasePrice,
			MinMultiplier: 50, MaxMultiplier: 200,
			Volatility: 2, TrendStrength: 30,
			MeanRevStrength: 12, SupplySensitivity: 50,
			SeasonProfile: profile,
		}
	}
}

// defaultSeasonProfile 根据作物当季生成季节因子
// 当季供应充足价格低，非当季稀缺价格高
func defaultSeasonProfile(cropSeason int) [4]int {
	profile := [4]int{100, 100, 100, 100}
	profile[cropSeason] = 78           // 当季降价（供应多）
	profile[(cropSeason+1)%4] = 95     // 邻季微降
	profile[(cropSeason+2)%4] = 118    // 对季涨价（稀缺）
	profile[(cropSeason+3)%4] = 108    // 邻季微涨
	return profile
}

func loadInitialPrices() {
	// 尝试从DB加载最近价格
	records, err := model.GetLatestMarketPrices()
	latestMap := make(map[string]int)
	if err == nil {
		for _, r := range records {
			latestMap[r.ItemKey] = r.Multiplier
		}
	}

	for key, cfg := range mktConfigs {
		mult := 100
		if m, ok := latestMap[key]; ok {
			mult = m
		}
		// clamp
		if mult < cfg.MinMultiplier {
			mult = cfg.MinMultiplier
		}
		if mult > cfg.MaxMultiplier {
			mult = cfg.MaxMultiplier
		}
		mktStates[key] = &marketItemState{
			Multiplier:     mult,
			PrevMultiplier: mult,
			Trend:          0,
		}
	}
}

// ========== Tick：核心价格计算 ==========

func doMarketTick() {
	mktMu.Lock()
	defer mktMu.Unlock()

	now := time.Now().Unix()
	season := getCurrentSeason()

	// 加载供需数据（最近7天）
	sdMap := loadSupplyDemandMap()

	// 加载活跃事件
	events, _ := model.GetActiveMarketEvents()
	eventMap := buildEventEffectMap(events)

	// 准备历史记录
	var histRecords []*model.TgMarketPriceHistory
	dateStr := time.Now().Format("20060102")
	snapshot := marketSnapshot{
		Timestamp: now,
		Prices:    make(map[string]int),
	}

	for key, cfg := range mktConfigs {
		state := mktStates[key]
		if state == nil {
			state = &marketItemState{Multiplier: 100, PrevMultiplier: 100}
			mktStates[key] = state
		}

		prev := state.Multiplier

		// 1) 季节因子推力
		seasonTarget := cfg.SeasonProfile[season]
		seasonDelta := (seasonTarget - 100) * 15 / 100 // 缓慢推向季节目标

		// 2) 供需因子推力
		supplyDelta := calcSupplyDemandDelta(key, sdMap, cfg)

		// 3) 趋势惯性
		trendDelta := state.Trend * cfg.TrendStrength / 100

		// 4) 事件因子推力
		eventDelta := 0
		if ef, ok := eventMap[key]; ok {
			eventDelta = ef
		}
		// 类别事件
		if ef, ok := eventMap["cat:"+cfg.Category]; ok {
			eventDelta += ef
		}

		// 5) 均值回归拉力
		deviation := prev - 100
		meanRevDelta := -deviation * cfg.MeanRevStrength / 100

		// 6) 噪声微扰
		noiseRange := cfg.Volatility * 2 // ±(volatility*2)%
		noiseDelta := 0
		if noiseRange > 0 {
			noiseDelta = rand.Intn(noiseRange*2+1) - noiseRange
		}

		// 汇总
		totalDelta := seasonDelta + supplyDelta + trendDelta + eventDelta + meanRevDelta + noiseDelta

		// 限制单tick最大变化
		maxChange := getMaxTickChange(cfg)
		if totalDelta > maxChange {
			totalDelta = maxChange
		}
		if totalDelta < -maxChange {
			totalDelta = -maxChange
		}

		newMult := prev + totalDelta

		// clamp
		if newMult < cfg.MinMultiplier {
			newMult = cfg.MinMultiplier
		}
		if newMult > cfg.MaxMultiplier {
			newMult = cfg.MaxMultiplier
		}

		// 更新趋势（EMA）
		change := newMult - prev
		state.Trend = (state.Trend*60 + change*40) / 100

		state.PrevMultiplier = prev
		state.Multiplier = newMult
		state.LastSeasonF = seasonDelta
		state.LastSupplyF = supplyDelta
		state.LastTrendF = trendDelta
		state.LastEventF = eventDelta
		state.LastNoiseF = noiseDelta
		state.LastMeanRevF = meanRevDelta
		state.TickCount++

		snapshot.Prices[key] = newMult

		histRecords = append(histRecords, &model.TgMarketPriceHistory{
			ItemKey:        key,
			Category:       cfg.Category,
			DateStr:        dateStr,
			TickIndex:      state.TickCount,
			Multiplier:     newMult,
			PrevMultiplier: prev,
			SeasonFactor:   seasonDelta,
			SupplyFactor:   supplyDelta,
			TrendFactor:    trendDelta,
			EventFactor:    eventDelta,
			NoiseFactor:    noiseDelta,
			MeanRevFactor:  meanRevDelta,
			Timestamp:      now,
		})
	}

	// 保存快照
	mktHistory = append(mktHistory, snapshot)
	if len(mktHistory) > mktHistoryMaxLen {
		mktHistory = mktHistory[len(mktHistory)-mktHistoryMaxLen:]
	}

	mktLastTick = now
	tickInterval := common.TgBotMarketRefreshHours
	if tickInterval <= 0 {
		tickInterval = 4
	}
	mktNextTick = now + int64(tickInterval*3600)

	// 异步写DB
	go func() {
		_ = model.CreateMarketPriceHistory(histRecords)
		// 定期清理旧数据
		if rand.Intn(10) == 0 {
			_ = model.CleanOldMarketHistory(60)
			_ = model.CleanOldSupplyDemand(30)
		}
	}()

	// 生成市场情报
	mktTips = generateMarketTips(season, events)
}

func getMaxTickChange(cfg *marketItemConfig) int {
	// 基础最大变化 8%，根据波动等级调整
	base := 8
	return base + cfg.Volatility*2
}

// ========== 供需因子计算 ==========

type sdData struct {
	TotalSell int
	TotalBuy  int
}

func loadSupplyDemandMap() map[string]*sdData {
	result := make(map[string]*sdData)
	records, err := model.GetRecentSupplyDemandAll(7)
	if err != nil {
		return result
	}
	for _, r := range records {
		sd, ok := result[r.ItemKey]
		if !ok {
			sd = &sdData{}
			result[r.ItemKey] = sd
		}
		sd.TotalSell += r.SellVolume
		sd.TotalBuy += r.BuyVolume
	}
	return result
}

func calcSupplyDemandDelta(key string, sdMap map[string]*sdData, cfg *marketItemConfig) int {
	sd, ok := sdMap[key]
	if !ok {
		return 0
	}
	// 只根据购买/消耗量推动价格：买得多 → 需求旺 → 涨价
	// 出售量不再压低价格（避免玩家卖货越多亏越多的"火耗"体验）
	if sd.TotalBuy == 0 {
		return 0
	}
	// 购买量越大，推力越强，但有上限
	buyPressure := float64(sd.TotalBuy) / 100.0
	if buyPressure > 1 {
		buyPressure = 1
	}
	delta := int(buyPressure * float64(cfg.SupplySensitivity) / 10.0)
	return delta
}

// ========== 事件效果计算 ==========

func buildEventEffectMap(events []*model.TgMarketEvent) map[string]int {
	result := make(map[string]int)
	for _, e := range events {
		effect := e.EffectValue
		if e.EffectDirection < 0 {
			effect = -effect
		}

		// 影响特定商品
		if e.AffectedItems != "" {
			items := strings.Split(e.AffectedItems, ",")
			for _, itemKey := range items {
				itemKey = strings.TrimSpace(itemKey)
				if itemKey != "" {
					result[itemKey] += effect
				}
			}
		}
		// 影响特定类别
		if e.AffectedCats != "" {
			cats := strings.Split(e.AffectedCats, ",")
			for _, cat := range cats {
				cat = strings.TrimSpace(cat)
				if cat != "" {
					result["cat:"+cat] += effect
				}
			}
		}
	}
	return result
}

// ========== 市场情报生成 ==========

func generateMarketTips(season int, events []*model.TgMarketEvent) []marketTip {
	var tips []marketTip

	// 季节提示
	seasonTips := getSeasonTips(season)
	tips = append(tips, seasonTips...)

	// 事件提示
	for _, e := range events {
		if e.IsPublic == 1 {
			icon := "📰"
			if e.EffectDirection > 0 {
				icon = "📈"
			} else {
				icon = "📉"
			}
			tips = append(tips, marketTip{
				Icon: icon,
				Text: e.Description,
				Cat:  e.AffectedCats,
				Type: "event",
			})
		}
	}

	// 趋势提示 (找出涨/跌幅度最大的几个商品)
	trendTips := getTrendTips()
	tips = append(tips, trendTips...)

	// 供需提示
	sdTips := getSupplyDemandTips()
	tips = append(tips, sdTips...)

	return tips
}

func getSeasonTips(season int) []marketTip {
	var tips []marketTip
	seasonName := seasonNames[season]
	daysLeft := getSeasonDaysLeft()

	tips = append(tips, marketTip{
		Icon: seasonEmojis[season],
		Text: fmt.Sprintf("当前%s季，剩余%d天", seasonName, daysLeft),
		Type: "season",
	})

	// 找出当季降价和反季涨价的典型商品
	var cheapItems, expensiveItems []string
	for _, cfg := range mktConfigs {
		if cfg.Category != "crop" {
			continue
		}
		sf := cfg.SeasonProfile[season]
		if sf <= 82 {
			cheapItems = append(cheapItems, cfg.Emoji+cfg.Name)
		}
		if sf >= 115 {
			expensiveItems = append(expensiveItems, cfg.Emoji+cfg.Name)
		}
	}

	if len(cheapItems) > 0 {
		names := cheapItems
		if len(names) > 4 {
			names = names[:4]
		}
		tips = append(tips, marketTip{
			Icon: "🔻",
			Text: fmt.Sprintf("%s季当季丰收，%s供应充足，价格偏低", seasonName, strings.Join(names, "、")),
			Type: "season",
		})
	}
	if len(expensiveItems) > 0 {
		names := expensiveItems
		if len(names) > 4 {
			names = names[:4]
		}
		tips = append(tips, marketTip{
			Icon: "🔺",
			Text: fmt.Sprintf("%s等反季稀缺，市场价格走高", strings.Join(names, "、")),
			Type: "season",
		})
	}

	// 换季提示
	if daysLeft <= 2 {
		nextSeason := (season + 1) % 4
		tips = append(tips, marketTip{
			Icon: "⚠️",
			Text: fmt.Sprintf("即将进入%s季，部分商品价格将发生变化", seasonNames[nextSeason]),
			Type: "season",
		})
	}

	return tips
}

type trendItem struct {
	name   string
	emoji  string
	change int
}

func getTrendTips() []marketTip {
	var tips []marketTip
	var risers, fallers []trendItem

	for key, state := range mktStates {
		cfg := mktConfigs[key]
		if cfg == nil {
			continue
		}
		change := state.Multiplier - state.PrevMultiplier
		if change >= 5 {
			risers = append(risers, trendItem{cfg.Name, cfg.Emoji, change})
		}
		if change <= -5 {
			fallers = append(fallers, trendItem{cfg.Name, cfg.Emoji, change})
		}
	}

	if len(risers) > 0 {
		sortTrendItemsDesc(risers)
		n := len(risers)
		if n > 3 {
			n = 3
		}
		var names []string
		for _, r := range risers[:n] {
			names = append(names, fmt.Sprintf("%s%s(+%d%%)", r.emoji, r.name, r.change))
		}
		tips = append(tips, marketTip{
			Icon: "📈",
			Text: fmt.Sprintf("近期涨势明显: %s", strings.Join(names, "、")),
			Type: "trend",
		})
	}
	if len(fallers) > 0 {
		sortTrendItemsAsc(fallers)
		n := len(fallers)
		if n > 3 {
			n = 3
		}
		var names []string
		for _, f := range fallers[:n] {
			names = append(names, fmt.Sprintf("%s%s(%d%%)", f.emoji, f.name, f.change))
		}
		tips = append(tips, marketTip{
			Icon: "📉",
			Text: fmt.Sprintf("近期跌势明显: %s", strings.Join(names, "、")),
			Type: "trend",
		})
	}

	return tips
}

func sortTrendItemsDesc(items []trendItem) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].change > items[i].change {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func sortTrendItemsAsc(items []trendItem) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].change < items[i].change {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func getSupplyDemandTips() []marketTip {
	var tips []marketTip

	for key, state := range mktStates {
		cfg := mktConfigs[key]
		if cfg == nil {
			continue
		}
		if state.LastSupplyF >= 3 {
			tips = append(tips, marketTip{
				Icon:    "🛒",
				Text:    fmt.Sprintf("%s%s需求旺盛，价格受到支撑", cfg.Emoji, cfg.Name),
				ItemKey: key,
				Type:    "supply",
			})
		}
		if state.LastSupplyF <= -3 {
			tips = append(tips, marketTip{
				Icon:    "📦",
				Text:    fmt.Sprintf("%s%s市场供应充足，价格承压", cfg.Emoji, cfg.Name),
				ItemKey: key,
				Type:    "supply",
			})
		}
	}

	// 限制数量
	if len(tips) > 4 {
		tips = tips[:4]
	}
	return tips
}

// ========== 公共API ==========

// ensureMarketEngine 确保引擎已初始化并且价格是最新的
func ensureMarketEngine() {
	initMarketEngine()
	mktMu.RLock()
	next := mktNextTick
	mktMu.RUnlock()
	if time.Now().Unix() >= next {
		doMarketTick()
	}
}

// getMarketMultiplierNew 获取商品当前倍率 (替代旧的 getMarketMultiplier)
func getMarketMultiplierNew(key string) int {
	ensureMarketEngine()
	mktMu.RLock()
	defer mktMu.RUnlock()
	if state, ok := mktStates[key]; ok {
		return state.Multiplier
	}
	return 100
}

// getMarketItemState 获取商品完整状态
func getMarketItemState(key string) *marketItemState {
	ensureMarketEngine()
	mktMu.RLock()
	defer mktMu.RUnlock()
	if state, ok := mktStates[key]; ok {
		cp := *state
		return &cp
	}
	return nil
}

// getMarketTips 获取当前市场情报
func getMarketTips() []marketTip {
	ensureMarketEngine()
	mktMu.RLock()
	defer mktMu.RUnlock()
	tips := make([]marketTip, len(mktTips))
	copy(tips, mktTips)
	return tips
}

// getMarketNextRefresh 获取下次刷新倒计时
func getMarketNextRefresh() int64 {
	ensureMarketEngine()
	mktMu.RLock()
	defer mktMu.RUnlock()
	remain := mktNextTick - time.Now().Unix()
	if remain < 0 {
		remain = 0
	}
	return remain
}

// getMarketHistorySnapshots 获取历史快照
func getMarketHistorySnapshots() []marketSnapshot {
	ensureMarketEngine()
	mktMu.RLock()
	defer mktMu.RUnlock()
	result := make([]marketSnapshot, len(mktHistory))
	copy(result, mktHistory)
	return result
}

// getAllMarketConfigs 获取所有商品配置
func getAllMarketConfigs() map[string]*marketItemConfig {
	ensureMarketEngine()
	mktMu.RLock()
	defer mktMu.RUnlock()
	result := make(map[string]*marketItemConfig, len(mktConfigs))
	for k, v := range mktConfigs {
		cp := *v
		result[k] = &cp
	}
	return result
}

// getMarketPriceTrend 获取商品价格趋势标签
func getMarketPriceTrend(key string) (tag string, arrow string, color string) {
	state := getMarketItemState(key)
	if state == nil {
		return "平稳", "→", "grey"
	}
	change := state.Multiplier - state.PrevMultiplier
	trend := state.Trend

	if change >= 10 || trend >= 8 {
		return "暴涨", "⬆️", "red"
	}
	if change >= 5 || trend >= 4 {
		return "上涨", "↗️", "orange"
	}
	if change >= 2 || trend >= 2 {
		return "微涨", "↗", "yellow"
	}
	if change <= -10 || trend <= -8 {
		return "暴跌", "⬇️", "green"
	}
	if change <= -5 || trend <= -4 {
		return "下跌", "↘️", "cyan"
	}
	if change <= -2 || trend <= -2 {
		return "微跌", "↘", "blue"
	}
	return "平稳", "→", "grey"
}

// ========== 辅助函数 ==========
