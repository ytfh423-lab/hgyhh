package controller

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ========== helpers ==========

func getWebFarmUser(c *gin.Context) (*model.User, string, bool) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return nil, "", false
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return nil, "", false
	}
	// 优先用 TelegramId，未绑定则用 u_{userId} 作为农场标识
	farmId := user.TelegramId
	if farmId == "" {
		farmId = fmt.Sprintf("u_%d", user.Id)
	}
	return user, farmId, true
}

func webFarmQuotaFloat(quota int) float64 {
	return float64(quota) / common.QuotaPerUnit
}

// checkFeatureLevel 检查用户是否达到功能解锁等级，未达到则返回错误并return false
func webCheckFeatureLevel(c *gin.Context, tgId string, requiredLevel int, featureName string) bool {
	userLevel := model.GetFarmLevel(tgId)
	if userLevel < requiredLevel {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("%s需要等级 %d 才能解锁（当前等级 %d）", featureName, requiredLevel, userLevel),
		})
		return false
	}
	return true
}

type webPlotInfo struct {
	PlotIndex     int     `json:"plot_index"`
	Status        int     `json:"status"`
	CropType      string  `json:"crop_type"`
	CropName      string  `json:"crop_name"`
	CropEmoji     string  `json:"crop_emoji"`
	PlantedAt     int64   `json:"planted_at"`
	GrowSecs      int64   `json:"grow_secs"`
	Progress      int     `json:"progress"`
	Remaining     int64   `json:"remaining"`
	EventType     string  `json:"event_type"`
	EventAt       int64   `json:"event_at"`
	StolenCount   int     `json:"stolen_count"`
	Fertilized    int     `json:"fertilized"`
	LastWateredAt int64   `json:"last_watered_at"`
	WaterRemain   int64   `json:"water_remain"`
	DeathRemain      int64   `json:"death_remain"`
	StatusLabel      string  `json:"status_label"`
	SoilLevel        int     `json:"soil_level"`
	ProtectionRemain int64   `json:"protection_remain"` // 保护期剩余秒数
	OwnerKeepPct     int     `json:"owner_keep_pct"`    // 主人保底比例%
}

func buildPlotInfo(plot *model.TgFarmPlot) webPlotInfo {
	updateFarmPlotStatus(plot)
	now := time.Now().Unix()
	soilLevel := plot.SoilLevel
	if soilLevel < 1 {
		soilLevel = 1
	}
	info := webPlotInfo{
		PlotIndex:     plot.PlotIndex,
		Status:        plot.Status,
		CropType:      plot.CropType,
		EventType:     plot.EventType,
		EventAt:       plot.EventAt,
		StolenCount:   plot.StolenCount,
		Fertilized:    plot.Fertilized,
		LastWateredAt: plot.LastWateredAt,
		PlantedAt:     plot.PlantedAt,
		SoilLevel:     soilLevel,
	}

	crop := farmCropMap[plot.CropType]
	if crop != nil {
		info.CropName = crop.Name
		info.CropEmoji = crop.Emoji
		info.GrowSecs = crop.GrowSecs
	}

	switch plot.Status {
	case 0:
		info.StatusLabel = "空地"
	case 1:
		info.StatusLabel = "生长中"
		if crop != nil {
			growSecs := crop.GrowSecs
			if soilLevel > 1 {
				bonus := int64(common.TgBotFarmSoilSpeedBonus * (soilLevel - 1))
				growSecs = growSecs * (100 - bonus) / 100
				if growSecs < 60 {
					growSecs = 60
				}
			}
			// 季节生长时间修正
			seasonGrowPct := getSeasonGrowthMultiplier(crop, plot.PlantedAt)
			growSecs = growSecs * int64(seasonGrowPct) / 100
			if growSecs < 60 {
				growSecs = 60
			}
			info.GrowSecs = growSecs
			elapsed := now - plot.PlantedAt
			pct := int(elapsed * 100 / growSecs)
			if pct > 99 {
				pct = 99
			}
			info.Progress = pct
			info.Remaining = growSecs - elapsed
			if info.Remaining < 0 {
				info.Remaining = 0
			}
		}
		if plot.LastWateredAt > 0 {
			waterInterval := getSeasonWaterInterval(int64(common.TgBotFarmWaterInterval), crop, plot.PlantedAt)
			nextWater := plot.LastWateredAt + waterInterval - now
			info.WaterRemain = nextWater
		}
	case 2:
		info.StatusLabel = "已成熟"
		if crop != nil {
			stealCfg := model.GetStealConfig()
			info.OwnerKeepPct = int(getStealKeepRatio(crop, stealCfg) * 100)
			if plot.MaturedAt > 0 {
				protEnd := plot.MaturedAt + getStealProtectionSeconds(crop, stealCfg)
				if now < protEnd {
					info.ProtectionRemain = protEnd - now
				}
			}
		}
	case 3:
		wiltDuration3 := int64(common.TgBotFarmWiltDuration)
		deathAt3 := plot.EventAt + wiltDuration3
		info.DeathRemain = deathAt3 - now
		if info.DeathRemain < 0 {
			info.DeathRemain = 0
		}
		if plot.EventType == "drought" {
			info.StatusLabel = "天灾干旱"
		} else {
			info.StatusLabel = "虫害"
		}
	case 4:
		info.StatusLabel = "枯萎"
		wiltDuration := int64(common.TgBotFarmWiltDuration)
		waterInterval := int64(common.TgBotFarmWaterInterval)
		deathAt := plot.LastWateredAt + waterInterval + wiltDuration
		info.DeathRemain = deathAt - now
		if info.DeathRemain < 0 {
			info.DeathRemain = 0
		}
	}
	return info
}

// ========== API handlers ==========

// WebFarmView returns the complete farm state
func WebFarmView(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	// 检查信用贷款违约
	creditDefaulted, _ := model.CheckCreditLoanDefault(tgId)
	if creditDefaulted {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你的信用贷款已逾期违约！你的平台账号已被封禁。"})
		return
	}
	// 检查抵押贷款违约
	mortgageDefaulted, mortgagePenalty := model.CheckMortgageDefault(tgId)
	if mortgageDefaulted && mortgagePenalty == "ban" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你的抵押贷款已逾期违约！你的平台账号已被封禁。"})
		return
	}

	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	// 自动浇水：灌溉系统已安装 或 阵雨天气
	w := GetCurrentWeather()
	hasIrrigation := model.HasAutomation(tgId, "irrigation")
	if hasIrrigation || w.Type == 1 {
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

	var plotInfos []webPlotInfo
	for _, plot := range plots {
		plotInfos = append(plotInfos, buildPlotInfo(plot))
	}

	// Dog info
	var dogInfo map[string]interface{}
	dog, dogErr := model.GetFarmDog(tgId)
	if dogErr == nil {
		model.UpdateDogHunger(dog)
		levelStr := "幼犬"
		statusStr := "成长中"
		if dog.Level == 2 {
			levelStr = "成犬"
			if dog.Hunger > 0 {
				statusStr = "看门中"
			} else {
				statusStr = "饿坏了"
			}
		} else {
			if dog.Hunger == 0 {
				statusStr = "饿坏了"
			} else {
				now := time.Now().Unix()
				hoursLeft := int64(common.TgBotFarmDogGrowHours) - (now-dog.CreatedAt)/3600
				if hoursLeft < 0 {
					hoursLeft = 0
				}
				statusStr = fmt.Sprintf("还需%d小时长大", hoursLeft)
			}
		}
		dogInfo = map[string]interface{}{
			"name":       dog.Name,
			"level":      dog.Level,
			"level_name": levelStr,
			"hunger":     dog.Hunger,
			"status":     statusStr,
			"guard_rate": common.TgBotFarmDogGuardRate,
		}
	}

	// Items
	items, _ := model.GetFarmItems(tgId)
	var itemInfos []map[string]interface{}
	for _, item := range items {
		def := farmItemMap[item.ItemType]
		if def != nil {
			itemInfos = append(itemInfos, map[string]interface{}{
				"key":      item.ItemType,
				"name":     def.Name,
				"emoji":    def.Emoji,
				"quantity": item.Quantity,
				"category": "item",
			})
		} else if strings.HasPrefix(item.ItemType, "seed_") {
			cropKey := strings.TrimPrefix(item.ItemType, "seed_")
			crop := farmCropMap[cropKey]
			if crop != nil {
				itemInfos = append(itemInfos, map[string]interface{}{
					"key":       item.ItemType,
					"name":      crop.Name + "种子",
					"emoji":     crop.Emoji,
					"quantity":  item.Quantity,
					"category":  "seed",
					"crop_key":  cropKey,
					"seed_cost": webFarmQuotaFloat(crop.SeedCost),
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"plots":      plotInfos,
			"dog":        dogInfo,
			"items":      itemInfos,
			"plot_count": len(plots),
			"max_plots":  model.FarmMaxPlots,
			"plot_price":            webFarmQuotaFloat(common.TgBotFarmPlotPrice),
			"balance":               webFarmQuotaFloat(user.Quota),
			"soil_max_level":        common.TgBotFarmSoilMaxLevel,
			"soil_speed_bonus":      common.TgBotFarmSoilSpeedBonus,
			"soil_upgrade_prices": map[string]interface{}{
				"2": webFarmQuotaFloat(common.TgBotFarmSoilUpgradePrice2),
				"3": webFarmQuotaFloat(common.TgBotFarmSoilUpgradePrice3),
				"4": webFarmQuotaFloat(common.TgBotFarmSoilUpgradePrice4),
				"5": webFarmQuotaFloat(common.TgBotFarmSoilUpgradePrice5),
			},
			"user_level": model.GetFarmLevel(tgId),
		"weather": func() gin.H {
			w := GetCurrentWeather()
			return gin.H{
				"type": w.Type, "type_key": w.TypeKey, "name": w.Name, "emoji": w.Emoji,
				"effects": w.Effects, "ends_in": w.EndsAt - time.Now().Unix(),
				"season": getCurrentSeason(),
			}
		}(),
			"prestige_level": model.GetPrestigeLevel(tgId),
			"prestige_bonus": model.GetPrestigeLevel(tgId) * common.TgBotFarmPrestigeBonusPerLevel,
			"task_summary": func() gin.H {
				dateStr := todayDateStr()
				tasks := getDailyTasks(dateStr)
				claimed, _ := model.GetTaskClaims(tgId, dateStr)
				done := 0
				for i, task := range tasks {
					progress := model.CountTodayActions(tgId, task.Action)
					isClaimed := false
					for _, idx := range claimed {
						if idx == i {
							isClaimed = true
							break
						}
					}
					if progress >= int64(task.Target) || isClaimed {
						done++
					}
				}
				return gin.H{"done": done, "total": len(tasks), "claimed": len(claimed)}
			}(),
		},
	})
}

// WebFarmCrops returns available crops
func WebFarmCrops(c *gin.Context) {
	var crops []map[string]interface{}
	for _, crop := range farmCrops {
		tierKey, tierName := getCropTier(&crop)
		tags := getCropTags(&crop)
		maxProfit := crop.MaxYield*crop.UnitPrice - crop.SeedCost
		hours := float64(crop.GrowSecs) / 3600.0
		avgYield := float64(1+crop.MaxYield) / 2.0
		avgProfit := avgYield*float64(crop.UnitPrice) - float64(crop.SeedCost)
		avgProfitPerHour := avgProfit / hours
		nowTs := time.Now().Unix()
		inSeason := isCropInSeason(&crop)
		seasonGrowPct := getSeasonGrowthMultiplier(&crop, nowTs)
		seasonYieldPct := getSeasonYieldMultiplier(&crop, nowTs)
		var seasonEventInfo string
		if inSeason {
			seasonEventInfo = "正常"
		} else {
			seasonEventInfo = fmt.Sprintf("+%d%%", common.TgBotFarmSeasonOffEventBonus)
		}
		crops = append(crops, map[string]interface{}{
			"key":                  crop.Key,
			"short":                crop.Short,
			"name":                 crop.Name,
			"emoji":                crop.Emoji,
			"seed_cost":            webFarmQuotaFloat(crop.SeedCost),
			"grow_secs":            crop.GrowSecs,
			"max_yield":            crop.MaxYield,
			"unit_price":           webFarmQuotaFloat(crop.UnitPrice),
			"max_value":            webFarmQuotaFloat(crop.MaxYield * crop.UnitPrice),
			"max_profit":           webFarmQuotaFloat(maxProfit),
			"avg_profit_per_hour":  webFarmQuotaFloat(int(avgProfitPerHour)),
			"tier":                 tierKey,
			"tier_name":            tierName,
			"tags":                 tags,
			"season":               crop.Season,
			"in_season":            inSeason,
			"season_grow_pct":      seasonGrowPct,
			"season_yield_pct":     seasonYieldPct,
			"season_event_info":    seasonEventInfo,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": crops})
}

// WebFarmShop returns shop items
func WebFarmShop(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var items []map[string]interface{}
	for _, item := range farmItems {
		cost := item.Cost
		if item.Key == "dogfood" {
			cost = common.TgBotFarmDogFoodPrice
		}
		desc := ""
		if item.Cures != "" {
			desc = "治疗" + farmEventLabel(item.Cures)
		} else if item.Key == "fertilizer" {
			desc = "施肥增产50%"
		} else if item.Key == "dogfood" {
			desc = "喂狗"
		}
		items = append(items, map[string]interface{}{
			"key":   item.Key,
			"name":  item.Name,
			"emoji": item.Emoji,
			"cost":  webFarmQuotaFloat(cost),
			"desc":  desc,
		})
	}

	// Dog purchase option
	hasDog := false
	_, dogErr := model.GetFarmDog(tgId)
	if dogErr == nil {
		hasDog = true
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     items,
			"has_dog":   hasDog,
			"dog_price": webFarmQuotaFloat(common.TgBotFarmDogPrice),
		},
	})
}

// WebFarmPlant plants a crop
func WebFarmPlant(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		CropKey   string `json:"crop_key"`
		PlotIndex int    `json:"plot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	crop := farmCropMap[req.CropKey]
	if crop == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知作物"})
		return
	}

	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	var targetPlot *model.TgFarmPlot
	for _, p := range plots {
		if p.PlotIndex == req.PlotIndex {
			targetPlot = p
			break
		}
	}
	if targetPlot == nil || targetPlot.Status != 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该地块不可用"})
		return
	}

	// 优先消耗库存种子，库存不足则从余额扣费
	seedKey := "seed_" + crop.Key
	usedInventory := false
	if errSeed := model.DecrementFarmItem(tgId, seedKey); errSeed == nil {
		usedInventory = true
		model.AddFarmLog(tgId, "plant", 0, fmt.Sprintf("使用库存种子种植%s%s", crop.Emoji, crop.Name))
	} else {
		if user.Quota < crop.SeedCost {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！种子需要 $%.2f（也可在商店提前购买种子）", webFarmQuotaFloat(crop.SeedCost))})
			return
		}
		err = model.DecreaseUserQuota(user.Id, crop.SeedCost)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
			return
		}
		model.AddFarmLog(tgId, "plant", -crop.SeedCost, fmt.Sprintf("种植%s%s", crop.Emoji, crop.Name))
	}
	_ = usedInventory

	now := time.Now().Unix()
	targetPlot.CropType = crop.Key
	targetPlot.PlantedAt = now
	targetPlot.Status = 1
	targetPlot.EventType = ""
	targetPlot.EventAt = 0
	targetPlot.StolenCount = 0
	targetPlot.LastWateredAt = now

	// 计算实际生长时间（含泥土加速）
	webActualGrowSecs := crop.GrowSecs
	webSoilLvl := targetPlot.SoilLevel
	if webSoilLvl < 1 {
		webSoilLvl = 1
	}
	if webSoilLvl > 1 {
		sBonus := int64(common.TgBotFarmSoilSpeedBonus * (webSoilLvl - 1))
		webActualGrowSecs = webActualGrowSecs * (100 - sBonus) / 100
		if webActualGrowSecs < 60 {
			webActualGrowSecs = 60
		}
	}

	// 季节生长倍率
	webSeasonGrowthPct := getSeasonGrowthMultiplier(crop, now)
	webActualGrowSecs = webActualGrowSecs * int64(webSeasonGrowthPct) / 100
	if webActualGrowSecs < 60 {
		webActualGrowSecs = 60
	}

	// 教程期间不触发随机事件
	inTutorial := model.IsFarmTutorialActive(tgId)
	bugChance := getSeasonEventChance(common.TgBotFarmEventChance, crop, now)
	if !inTutorial && rand.Intn(100) < bugChance {
		targetPlot.EventType = "bugs"
		offset := webActualGrowSecs * int64(30+rand.Intn(50)) / 100
		targetPlot.EventAt = now + offset
	}
	// 拥有灌溉自动化则跳过干旱；反季概率更高
	droughtChance := getSeasonEventChance(common.TgBotFarmDisasterChance, crop, now)
	if !inTutorial && targetPlot.EventType == "" && !model.HasAutomation(tgId, "irrigation") && rand.Intn(100) < droughtChance {
		targetPlot.EventType = "drought"
		offset := webActualGrowSecs * int64(30+rand.Intn(50)) / 100
		targetPlot.EventAt = now + offset
	}

	_ = model.UpdateFarmPlot(targetPlot)
	common.SysLog(fmt.Sprintf("Web Farm: user %s planted %s on plot %d", tgId, crop.Key, req.PlotIndex))

	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("种植 %s%s 成功！", crop.Emoji, crop.Name)})
}

// WebFarmHarvest harvests all mature crops
func WebFarmHarvest(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	stealCfg := model.GetStealConfig()
	totalQuota := 0
	harvestedCount := 0
	var details []map[string]interface{}

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
			realYield, guaranteed, stolenLoss := calcHarvestYield(baseYield, fertBonus, plot.StolenCount, crop, stealCfg)
			marketPrice := applyMarket(crop.UnitPrice, "crop_"+crop.Key)
			seasonPrice := applySeasonPrice(marketPrice, crop)
			value := realYield * seasonPrice
			totalQuota += value
			harvestedCount++

			details = append(details, map[string]interface{}{
				"crop_name":    crop.Name,
				"crop_emoji":   crop.Emoji,
				"raw_yield":    rawYield,
				"yield":        baseYield,
				"yield_mult":   yieldMult,
				"fert_bonus":   fertBonus,
				"stolen":       stolenLoss,
				"guaranteed":   guaranteed,
				"real_yield":   realYield,
				"value":        webFarmQuotaFloat(value),
				"in_season":    isCropInSeason(crop),
				"season_pct":   getSeasonPriceMultiplier(crop),
			})

			_ = model.ClearFarmPlot(plot.Id)
			model.RecordCollection(tgId, "crop", crop.Key, realYield)
		}
	}

	if harvestedCount == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有可收获的作物"})
		return
	}

	_ = model.IncreaseUserQuota(user.Id, totalQuota, true)
	model.AddFarmLog(tgId, "harvest", totalQuota, fmt.Sprintf("收获%d种作物", harvestedCount))
	common.SysLog(fmt.Sprintf("Web Farm: user %s harvested %d crops, total %d quota", tgId, harvestedCount, totalQuota))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("收获 %d 块作物，获得 $%.2f", harvestedCount, webFarmQuotaFloat(totalQuota)),
		"data": gin.H{
			"count":   harvestedCount,
			"total":   webFarmQuotaFloat(totalQuota),
			"details": details,
		},
	})
}

// WebFarmBuyItem buys a shop item (supports quantity for batch purchase)
func WebFarmBuyItem(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		ItemKey  string `json:"item_key"`
		Quantity int    `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}
	if req.Quantity > 99 {
		req.Quantity = 99
	}
	var unitCost int
	var itemEmoji, itemName, inventoryKey string

	item := farmItemMap[req.ItemKey]
	if item != nil {
		unitCost = item.Cost
		if req.ItemKey == "dogfood" {
			unitCost = common.TgBotFarmDogFoodPrice
		}
		itemEmoji = item.Emoji
		itemName = item.Name
		inventoryKey = req.ItemKey
	} else {
		crop := farmCropMap[req.ItemKey]
		if crop == nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知道具"})
			return
		}
		unitCost = crop.SeedCost
		itemEmoji = crop.Emoji
		itemName = crop.Name + "种子"
		inventoryKey = "seed_" + crop.Key
	}

	totalCost := unitCost * req.Quantity
	if user.Quota < totalCost {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！需要 $%.2f（单价 $%.2f × %d）", webFarmQuotaFloat(totalCost), webFarmQuotaFloat(unitCost), req.Quantity)})
		return
	}
	err := model.DecreaseUserQuota(user.Id, totalCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}
	err = model.IncrementFarmItem(tgId, inventoryKey, req.Quantity)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, totalCost, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "购买失败"})
		return
	}
	model.AddFarmLog(tgId, "shop", -totalCost, fmt.Sprintf("购买%s%s×%d", itemEmoji, itemName, req.Quantity))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("购买 %s%s ×%d 成功！", itemEmoji, itemName, req.Quantity),
		"data": gin.H{
			"item":       req.ItemKey,
			"quantity":   req.Quantity,
			"total_cost": webFarmQuotaFloat(totalCost),
		},
	})
}

// WebFarmSellSeed 出售库存种子（半价退回）
func WebFarmSellSeed(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		SeedKey  string `json:"seed_key"`
		Quantity int    `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}

	// 校验种子类型
	cropKey := req.SeedKey
	if !strings.HasPrefix(cropKey, "seed_") {
		cropKey = "seed_" + cropKey
	}
	realCropKey := strings.TrimPrefix(cropKey, "seed_")
	crop := farmCropMap[realCropKey]
	if crop == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知种子"})
		return
	}

	// 检查库存
	qty, _ := model.GetFarmItemQuantity(tgId, cropKey)
	if qty < req.Quantity {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("库存不足！当前仅有 %d 个", qty)})
		return
	}

	// 扣库存
	for i := 0; i < req.Quantity; i++ {
		if err := model.DecrementFarmItem(tgId, cropKey); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣除库存失败"})
			return
		}
	}

	// 半价退回
	refund := crop.SeedCost * req.Quantity / 2
	_ = model.IncreaseUserQuota(user.Id, refund, true)
	model.AddFarmLog(tgId, "shop", refund, fmt.Sprintf("出售%s%s种子×%d", crop.Emoji, crop.Name, req.Quantity))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("出售 %s%s种子 ×%d，获得 $%.2f", crop.Emoji, crop.Name, req.Quantity, webFarmQuotaFloat(refund)),
	})
}

// WebFarmStealTargets returns available steal targets
func WebFarmStealTargets(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockSteal, "偷菜") {
		return
	}
	cfg := model.GetStealConfig()
	if !cfg.StealEnabled {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}, "steal_disabled": true})
		return
	}
	targets, err := model.GetMatureFarmTargetsV2(tgId, cfg.MaxStealPerPlot)
	if err != nil || len(targets) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
		return
	}
	thiefToday := model.CountThiefStealsToday(tgId)
	var result []map[string]interface{}
	for _, t := range targets {
		result = append(result, map[string]interface{}{
			"id":    t.TelegramId,
			"label": maskTgId(t.TelegramId),
			"count": t.Count,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
		"steal_info": gin.H{
			"today_count":   thiefToday,
			"daily_limit":   cfg.MaxStealPerUserPerDay,
			"cooldown_secs": cfg.StealCooldownSeconds,
			"keep_ratio":    int(cfg.OwnerBaseKeepRatio * 100),
		},
	})
}

// WebFarmSteal steals from another player
func WebFarmSteal(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockSteal, "偷菜") {
		return
	}

	cfg := model.GetStealConfig()
	if !cfg.StealEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "偷菜功能当前已关闭"})
		return
	}

	var req struct {
		VictimId string `json:"victim_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if tgId == req.VictimId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "不能偷自己的菜！"})
		return
	}

	// 每日偷菜次数限制
	thiefToday := model.CountThiefStealsToday(tgId)
	if thiefToday >= int64(cfg.MaxStealPerUserPerDay) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("今日偷菜次数已达上限（%d次）", cfg.MaxStealPerUserPerDay)})
		return
	}

	// 目标农场每日被偷限制
	farmStolenToday := model.CountFarmStolenToday(req.VictimId)
	if farmStolenToday >= int64(cfg.MaxStealPerFarmPerDay) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该农场今日已受保护"})
		return
	}

	// 冷却
	now := time.Now().Unix()
	recentSteals, _ := model.CountRecentSteals(tgId, req.VictimId, now-int64(cfg.StealCooldownSeconds))
	if recentSteals > 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("冷却中！%d分钟内只能偷同一人一次", cfg.StealCooldownSeconds/60)})
		return
	}

	// 稻草人
	if model.HasAutomation(req.VictimId, "scarecrow") {
		if rand.Intn(100) < cfg.ScarecrowBlockRate {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "对方的稻草人吓跑了你，偷菜失败！🧑‍🌾"})
			return
		}
	}

	// 看门狗
	victimDog, dogErr := model.GetFarmDog(req.VictimId)
	if dogErr == nil {
		model.UpdateDogHunger(victimDog)
		if victimDog.Level == 2 && victimDog.Hunger > 0 {
			if rand.Intn(100) < cfg.DogGuardRate {
				c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("对方的看门狗「%s」发现了你，偷菜失败！", victimDog.Name)})
				return
			}
		}
	}

	// 偷取成功率
	if cfg.StealSuccessRate < 100 && rand.Intn(100) >= cfg.StealSuccessRate {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "手一滑，偷菜失败了！"})
		return
	}

	// 获取可偷地块
	plots, err := model.GetStealablePlotsV2(req.VictimId, cfg.MaxStealPerPlot)
	if err != nil || len(plots) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该玩家没有可偷的成熟作物"})
		return
	}

	// 过滤保护期和超长周期
	var stealable []*model.TgFarmPlot
	for _, p := range plots {
		crop := farmCropMap[p.CropType]
		if crop == nil {
			continue
		}
		if isPlotInProtection(p, crop, cfg) {
			continue
		}
		if cfg.LongCropProtectionEnabled && cfg.SuperLongCropBonusOnly {
			hours := float64(crop.GrowSecs) / 3600.0
			if hours >= float64(cfg.SuperLongCropHoursThreshold) {
				continue
			}
		}
		stealable = append(stealable, p)
	}
	if len(stealable) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该玩家的作物仍在保护期内"})
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

	// 计算可偷单位
	keepRatio := getStealKeepRatio(crop, cfg)
	maxYield := 1
	if crop != nil {
		maxYield = crop.MaxYield
	}
	maxStealableUnits := int(float64(maxYield) * (1.0 - keepRatio))
	if maxStealableUnits < 1 {
		maxStealableUnits = 1
	}
	remainStealable := maxStealableUnits - target.StolenCount
	if remainStealable <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "这块地的可偷收益已被摘完"})
		return
	}

	stealUnits := 1
	if remainStealable < stealUnits {
		stealUnits = remainStealable
	}
	marketPrice := applyMarket(unitPrice, "crop_"+crop.Key)
	stealValue := stealUnits * marketPrice

	_ = model.IncrementPlotStolenBy(target.Id, stealUnits)
	if cfg.EnableStealLog {
		_ = model.CreateFarmStealLog(&model.TgFarmStealLog{
			ThiefId:  tgId,
			VictimId: req.VictimId,
			PlotId:   target.Id,
			Amount:   stealValue,
		})
	}
	_ = model.IncreaseUserQuota(user.Id, stealValue, true)
	model.AddFarmLog(tgId, "steal", stealValue, fmt.Sprintf("摘取%s%s额外收益×%d", cropEmoji, cropName, stealUnits))

	common.SysLog(fmt.Sprintf("Web Farm: user %s stole %s bonus from %s, +%d quota", tgId, cropName, req.VictimId, stealValue))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("摘取了 %d个%s%s 的额外收益，获得 $%.2f（对方基础收益不受影响）", stealUnits, cropEmoji, cropName, webFarmQuotaFloat(stealValue)),
		"data": gin.H{
			"victim":     maskTgId(req.VictimId),
			"crop_name":  cropName,
			"crop_emoji": cropEmoji,
			"units":      stealUnits,
			"value":      webFarmQuotaFloat(stealValue),
			"bonus_only": true,
		},
	})
}

// WebFarmTreat treats a plot event
func WebFarmTreat(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		PlotIndex int `json:"plot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	var targetPlot *model.TgFarmPlot
	for _, p := range plots {
		if p.PlotIndex == req.PlotIndex {
			targetPlot = p
			break
		}
	}
	if targetPlot == nil || targetPlot.Status != 3 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该地块不需要治疗"})
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
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无法治疗此事件（干旱请用浇水）"})
		return
	}

	err = model.DecrementFarmItem(tgId, cureItem.Key)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("没有 %s%s！请先购买", cureItem.Emoji, cureItem.Name)})
		return
	}

	now := time.Now().Unix()
	downtime := now - targetPlot.EventAt
	targetPlot.PlantedAt += downtime
	targetPlot.Status = 1
	targetPlot.EventType = ""
	targetPlot.EventAt = 0
	_ = model.UpdateFarmPlot(targetPlot)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("使用 %s%s 治疗成功！", cureItem.Emoji, cureItem.Name)})
}

// WebFarmFertilize fertilizes a plot
func WebFarmFertilize(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		PlotIndex int `json:"plot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	var target *model.TgFarmPlot
	for _, plot := range plots {
		if plot.PlotIndex == req.PlotIndex {
			target = plot
			break
		}
	}
	if target == nil || target.Status != 1 || target.Fertilized == 1 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该地块不可施肥"})
		return
	}

	if err := model.DecrementFarmItem(tgId, "fertilizer"); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "化肥不足！请先购买"})
		return
	}

	target.Fertilized = 1
	_ = model.UpdateFarmPlot(target)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("%d号地施肥成功！收获时产量+50%%", req.PlotIndex+1)})
}

// WebFarmBuyLand buys a new plot
func WebFarmBuyLand(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	plotCount, err := model.GetFarmPlotCount(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	if int(plotCount) >= model.FarmMaxPlots {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("已达上限 %d 块土地！", model.FarmMaxPlots)})
		return
	}

	price := common.TgBotFarmPlotPrice
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！土地价格 $%.2f", webFarmQuotaFloat(price))})
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

	newIdx := int(plotCount)
	err = model.CreateNewFarmPlot(tgId, newIdx)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建地块失败，已退款"})
		return
	}
	model.AddFarmLog(tgId, "buy_plot", -price, fmt.Sprintf("购买%d号地", newIdx+1))

	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("购买 %d号地 成功！", newIdx+1)})
}

// WebFarmUpgradeSoil upgrades the soil level of a plot
func WebFarmUpgradeSoil(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		PlotIndex int `json:"plot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	var target *model.TgFarmPlot
	for _, p := range plots {
		if p.PlotIndex == req.PlotIndex {
			target = p
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "地块不存在"})
		return
	}

	currentLevel := target.SoilLevel
	if currentLevel < 1 {
		currentLevel = 1
	}
	nextLevel := currentLevel + 1
	if nextLevel > common.TgBotFarmSoilMaxLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("泥土已达最高等级 %d", common.TgBotFarmSoilMaxLevel)})
		return
	}

	// Get upgrade price based on target level
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
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "不支持的升级等级"})
		return
	}

	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！升级到%d级需要 $%.2f", nextLevel, webFarmQuotaFloat(price))})
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

	err = model.UpgradeFarmPlotSoil(target.Id, nextLevel)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "升级失败，已退款"})
		return
	}

	speedBonus := common.TgBotFarmSoilSpeedBonus * (nextLevel - 1)
	model.AddFarmLog(tgId, "upgrade_soil", -price, fmt.Sprintf("%d号地泥土升级Lv.%d", req.PlotIndex+1, nextLevel))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("泥土升级到 %d 级成功！生长加速 %d%%", nextLevel, speedBonus),
	})
}

// WebFarmWater waters a plot
func WebFarmWater(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		PlotIndex int `json:"plot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	var target *model.TgFarmPlot
	for _, plot := range plots {
		if plot.PlotIndex == req.PlotIndex {
			target = plot
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该地块不存在"})
		return
	}

	updateFarmPlotStatus(target)

	canWater := target.Status == 1 || target.Status == 4 ||
		(target.Status == 3 && target.EventType == "drought")
	if !canWater {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该地块不需要浇水"})
		return
	}

	wasWilting := target.Status == 4
	wasDrought := target.Status == 3 && target.EventType == "drought"

	if wasWilting {
		now := time.Now().Unix()
		waterInterval := int64(common.TgBotFarmWaterInterval)
		wiltStart := target.LastWateredAt + waterInterval
		downtime := now - wiltStart
		target.PlantedAt += downtime
		target.Status = 1
		_ = model.UpdateFarmPlot(target)
	}

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
	model.AddFarmLog(tgId, "water", 0, fmt.Sprintf("浇水%d号地", req.PlotIndex+1))

	msg := "浇水成功！"
	if wasDrought {
		msg = "天灾干旱已解除，恢复生长！"
	} else if wasWilting {
		msg = "已从枯萎中恢复生长！"
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": msg})
}

// WebFarmWaterAll waters all plots that need watering
func WebFarmWaterAll(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	watered := 0
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
		canWater := plot.Status == 1 || plot.Status == 4 ||
			(plot.Status == 3 && plot.EventType == "drought")
		if !canWater {
			continue
		}
		if plot.Status == 4 {
			now := time.Now().Unix()
			waterInterval := int64(common.TgBotFarmWaterInterval)
			wiltStart := plot.LastWateredAt + waterInterval
			downtime := now - wiltStart
			plot.PlantedAt += downtime
			plot.Status = 1
			_ = model.UpdateFarmPlot(plot)
		}
		if plot.Status == 3 && plot.EventType == "drought" {
			now := time.Now().Unix()
			downtime := now - plot.EventAt
			plot.PlantedAt += downtime
			plot.Status = 1
			plot.EventType = ""
			plot.EventAt = 0
			_ = model.UpdateFarmPlot(plot)
		}
		_ = model.WaterFarmPlot(plot.Id)
		watered++
	}
	if watered == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有需要浇水的地块"})
		return
	}
	model.AddFarmLog(tgId, "water", 0, fmt.Sprintf("一键浇水%d块地", watered))

	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("成功浇水 %d 块地！", watered)})
}

// WebFarmFertilizeAll fertilizes all growing plots
func WebFarmFertilizeAll(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	fertilized := 0
	for _, plot := range plots {
		if plot.Status != 1 || plot.Fertilized == 1 {
			continue
		}
		if err := model.DecrementFarmItem(tgId, "fertilizer"); err != nil {
			break
		}
		plot.Fertilized = 1
		_ = model.UpdateFarmPlot(plot)
		fertilized++
	}
	if fertilized == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有可施肥的地块（或化肥不足）"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("成功施肥 %d 块地！", fertilized)})
}

// WebFarmDog returns dog info
func WebFarmDog(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockDog, "狗狗") {
		return
	}

	dog, err := model.GetFarmDog(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"has_dog":    false,
				"dog_price":  webFarmQuotaFloat(common.TgBotFarmDogPrice),
				"grow_hours": common.TgBotFarmDogGrowHours,
				"guard_rate": common.TgBotFarmDogGuardRate,
				"food_price": webFarmQuotaFloat(common.TgBotFarmDogFoodPrice),
			},
		})
		return
	}

	model.UpdateDogHunger(dog)
	levelStr := "幼犬"
	statusStr := "成长中"
	hoursLeft := int64(0)
	if dog.Level == 2 {
		levelStr = "成犬"
		if dog.Hunger > 0 {
			statusStr = "看门中"
		} else {
			statusStr = "饿坏了"
		}
	} else {
		now := time.Now().Unix()
		hoursLeft = int64(common.TgBotFarmDogGrowHours) - (now-dog.CreatedAt)/3600
		if hoursLeft < 0 {
			hoursLeft = 0
		}
		if dog.Hunger == 0 {
			statusStr = "饿坏了"
		} else {
			statusStr = fmt.Sprintf("还需%d小时长大", hoursLeft)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"has_dog":    true,
			"name":       dog.Name,
			"level":      dog.Level,
			"level_name": levelStr,
			"hunger":     dog.Hunger,
			"status":     statusStr,
			"hours_left": hoursLeft,
			"guard_rate": common.TgBotFarmDogGuardRate,
			"food_price": webFarmQuotaFloat(common.TgBotFarmDogFoodPrice),
			"grow_hours": common.TgBotFarmDogGrowHours,
			"dog_price":  webFarmQuotaFloat(common.TgBotFarmDogPrice),
		},
	})
}

// WebFarmBuyDog buys a dog
func WebFarmBuyDog(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockDog, "狗狗") {
		return
	}

	_, err := model.GetFarmDog(tgId)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你已经有一只狗了！"})
		return
	}

	price := common.TgBotFarmDogPrice
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！需要 $%.2f", webFarmQuotaFloat(price))})
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

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
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "购买失败，已退款"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("恭喜！获得小狗「%s」！%d小时后长大可看门", dogName, common.TgBotFarmDogGrowHours),
	})
}

// WebFarmFeedDog feeds the dog
func WebFarmFeedDog(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockDog, "狗狗") {
		return
	}

	dog, err := model.GetFarmDog(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你还没有狗狗！"})
		return
	}

	model.UpdateDogHunger(dog)

	if dog.Hunger >= 100 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "狗狗不饿，不需要喂食！"})
		return
	}

	err = model.DecrementFarmItem(tgId, "dogfood")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有狗粮！请先到商店购买"})
		return
	}

	_ = model.FeedFarmDog(dog.Id)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("喂食成功！「%s」饱食度恢复到100%%", dog.Name)})
}

// WebFarmLogs returns spending/income logs
func WebFarmLogs(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	offset := (page - 1) * pageSize
	logs, total, err := model.GetFarmLogs(tgId, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取记录失败"})
		return
	}

	type logItem struct {
		Id        int     `json:"id"`
		Action    string  `json:"action"`
		ActionLabel string `json:"action_label"`
		Amount    float64 `json:"amount"`
		Detail    string  `json:"detail"`
		CreatedAt int64   `json:"created_at"`
	}

	actionLabels := map[string]string{
		"plant":        "种植",
		"harvest":      "收获",
		"shop":         "商店",
		"steal":        "偷菜",
		"buy_plot":     "购地",
		"buy_dog":      "买狗",
		"upgrade_soil": "升级",
		"ranch_buy":    "买动物",
		"ranch_feed":   "喂食",
		"ranch_water":  "喂水",
		"ranch_sell":   "出售",
		"ranch_clean":  "清粪",
		"fish":         "钓鱼",
		"fish_sell":    "卖鱼",
		"craft":        "加工",
		"craft_sell":   "收取",
		"task":         "任务",
		"achieve":      "成就",
		"levelup":      "升级",
	}

	var items []logItem
	for _, l := range logs {
		label := actionLabels[l.Action]
		if label == "" {
			label = l.Action
		}
		items = append(items, logItem{
			Id:          l.Id,
			Action:      l.Action,
			ActionLabel: label,
			Amount:      webFarmQuotaFloat(l.Amount),
			Detail:      l.Detail,
			CreatedAt:   l.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs":      items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// ========== 市场 ==========

// WebFarmMarket returns current market prices with trend info and tips
func WebFarmMarket(c *gin.Context) {
	ensureMarketFresh()

	type priceInfo struct {
		Key        string  `json:"key"`
		Name       string  `json:"name"`
		Emoji      string  `json:"emoji"`
		Category   string  `json:"category"`
		BasePrice  float64 `json:"base_price"`
		Multiplier int     `json:"multiplier"`
		PrevMult   int     `json:"prev_multiplier"`
		CurPrice   float64 `json:"cur_price"`
		Change     int     `json:"change"`
		TrendTag   string  `json:"trend_tag"`
		TrendArrow string  `json:"trend_arrow"`
		TrendColor string  `json:"trend_color"`
	}

	configs := getAllMarketConfigs()
	var prices []priceInfo

	for _, cfg := range configs {
		state := getMarketItemState(cfg.Key)
		mult := 100
		prevMult := 100
		if state != nil {
			mult = state.Multiplier
			prevMult = state.PrevMultiplier
		}
		tag, arrow, clr := getMarketPriceTrend(cfg.Key)
		prices = append(prices, priceInfo{
			Key:        cfg.Key,
			Name:       cfg.Name,
			Emoji:      cfg.Emoji,
			Category:   cfg.Category,
			BasePrice:  webFarmQuotaFloat(cfg.BasePrice),
			Multiplier: mult,
			PrevMult:   prevMult,
			CurPrice:   webFarmQuotaFloat(cfg.BasePrice * mult / 100),
			Change:     mult - prevMult,
			TrendTag:   tag,
			TrendArrow: arrow,
			TrendColor: clr,
		})
	}

	nextRefresh := getMarketNextRefresh()
	tips := getMarketTips()
	season := getCurrentSeason()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"prices":        prices,
			"next_refresh":  nextRefresh,
			"refresh_hours": common.TgBotMarketRefreshHours,
			"tips":          tips,
			"season":        season,
			"season_name":   getSeasonName(season),
			"season_days_left": getSeasonDaysLeft(),
		},
	})
}

// WebFarmMarketHistory returns market price history for chart rendering
func WebFarmMarketHistory(c *gin.Context) {
	ensureMarketFresh()

	history := getMarketHistorySnapshots()

	type historyPoint struct {
		Timestamp int64          `json:"timestamp"`
		Prices    map[string]int `json:"prices"`
	}

	points := make([]historyPoint, len(history))
	for i, snap := range history {
		points[i] = historyPoint{
			Timestamp: snap.Timestamp,
			Prices:    snap.Prices,
		}
	}

	// 构建物品元信息
	type itemMeta struct {
		Key      string `json:"key"`
		Name     string `json:"name"`
		Emoji    string `json:"emoji"`
		Category string `json:"category"`
	}
	var items []itemMeta
	configs := getAllMarketConfigs()
	for _, cfg := range configs {
		items = append(items, itemMeta{cfg.Key, cfg.Name, cfg.Emoji, cfg.Category})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"history": points,
			"items":   items,
		},
	})
}

// ========== 等级系统 ==========

// WebFarmLevelInfo returns current level, prices, feature unlock info
func WebFarmLevelInfo(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	level := model.GetFarmLevel(tgId)

	type unlockInfo struct {
		Key      string `json:"key"`
		Name     string `json:"name"`
		Level    int    `json:"level"`
		Unlocked bool   `json:"unlocked"`
	}
	var unlocks []unlockInfo
	for _, fu := range featureUnlocks {
		unlocks = append(unlocks, unlockInfo{
			Key:      fu.Key,
			Name:     fu.Name,
			Level:    *fu.Level,
			Unlocked: level >= *fu.Level,
		})
	}

	type priceItem struct {
		Level int     `json:"level"`
		Price float64 `json:"price"`
	}
	var prices []priceItem
	for i, p := range common.TgBotFarmLevelPrices {
		lv := i + 2
		if lv > common.TgBotFarmMaxLevel {
			break
		}
		prices = append(prices, priceItem{Level: lv, Price: webFarmQuotaFloat(p)})
	}

	nextPrice := float64(0)
	if level < common.TgBotFarmMaxLevel {
		nextPrice = webFarmQuotaFloat(getLevelUpPrice(level))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"level":      level,
			"max_level":  common.TgBotFarmMaxLevel,
			"next_price": nextPrice,
			"prices":     prices,
			"unlocks":    unlocks,
		},
	})
}

// WebFarmLevelUp handles level upgrade
func WebFarmLevelUp(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	level := model.GetFarmLevel(tgId)
	if level >= common.TgBotFarmMaxLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "已达最高等级"})
		return
	}

	// 有未还抵押贷款时禁止升级（信用贷款允许升级）
	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr == nil && activeLoan != nil && activeLoan.LoanType == 1 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你有未还清的抵押贷款，还清后才能升级！抵押贷款资金不能用于升级。"})
		return
	}

	// 抵押违约永久禁止10级+
	newLevel := level + 1
	if newLevel >= 10 && model.HasMortgageBlocked(tgId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "由于抵押贷款违约，你已被永久禁止升级到10级及以上等级。"})
		return
	}

	price := getLevelUpPrice(level)
	userQuota, _ := model.GetUserQuota(user.Id, false)
	if userQuota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足，需要$%.2f，当前$%.2f", float64(price)/500000.0, float64(userQuota)/500000.0)})
		return
	}
	_ = model.DecreaseUserQuota(user.Id, price)

	model.SetFarmLevel(tgId, newLevel)
	model.AddFarmLog(tgId, "levelup", -price, fmt.Sprintf("升级到Lv.%d", newLevel))

	var newUnlocks []string
	for _, fu := range featureUnlocks {
		if *fu.Level == newLevel {
			newUnlocks = append(newUnlocks, fu.Name)
		}
	}

	msg := fmt.Sprintf("升级到 Lv.%d", newLevel)
	if len(newUnlocks) > 0 {
		msg += "，解锁: " + strings.Join(newUnlocks, ", ")
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": msg,
		"data": gin.H{
			"new_level": newLevel,
			"unlocks":   newUnlocks,
		},
	})
}

// ========== 每日任务 & 成就 ==========

// WebFarmTasks returns daily tasks with progress
func WebFarmTasks(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	dateStr := todayDateStr()
	tasks := getDailyTasks(dateStr)
	claimed, _ := model.GetTaskClaims(tgId, dateStr)
	claimedSet := make(map[int]bool)
	for _, idx := range claimed {
		claimedSet[idx] = true
	}

	type taskInfo struct {
		Index        int     `json:"index"`
		Action       string  `json:"action"`
		Name         string  `json:"name"`
		Emoji        string  `json:"emoji"`
		Target       int     `json:"target"`
		Progress     int64   `json:"progress"`
		Done         bool    `json:"done"`
		Claimed      bool    `json:"claimed"`
		Reward       float64 `json:"reward"`
		Description  string  `json:"description"`
		Hint         string  `json:"hint"`
		AutoType     string  `json:"auto_type"`
		AutoInstalled bool   `json:"auto_installed"`
		AutoText     string  `json:"auto_text"`
	}
	var taskList []taskInfo
	for i, task := range tasks {
		progress := model.CountTodayActions(tgId, task.Action)
		info := taskInfo{
			Index:       i,
			Action:      task.Action,
			Name:        task.Name,
			Emoji:       task.Emoji,
			Target:      task.Target,
			Progress:    progress,
			Done:        progress >= int64(task.Target),
			Claimed:     claimedSet[i],
			Reward:      webFarmQuotaFloat(task.Reward),
			Description: getTaskDesc(task.Action, task.Target),
		}
		if meta, ok := actionMetaMap[task.Action]; ok {
			info.Hint = meta.Hint
			info.AutoType = meta.AutoType
			info.AutoText = meta.AutoText
			if meta.AutoType != "" {
				info.AutoInstalled = model.HasAutomation(tgId, meta.AutoType)
			}
		}
		taskList = append(taskList, info)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"date":  dateStr,
			"tasks": taskList,
		},
	})
}

// WebFarmClaimTask claims a daily task reward
func WebFarmClaimTask(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		Index int `json:"index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	dateStr := todayDateStr()
	tasks := getDailyTasks(dateStr)
	if req.Index < 0 || req.Index >= len(tasks) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效任务"})
		return
	}

	claimed, _ := model.GetTaskClaims(tgId, dateStr)
	for _, idx := range claimed {
		if idx == req.Index {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "奖励已领取"})
			return
		}
	}

	task := tasks[req.Index]
	progress := model.CountTodayActions(tgId, task.Action)
	if progress < int64(task.Target) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("任务未完成（%d/%d）", progress, task.Target)})
		return
	}

	_ = model.ClaimTask(tgId, dateStr, req.Index)
	_ = model.IncreaseUserQuota(user.Id, task.Reward, true)
	model.AddFarmLog(tgId, "task", task.Reward, fmt.Sprintf("完成任务:%s", task.Name))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("领取奖励 $%.2f", webFarmQuotaFloat(task.Reward)),
	})
}

// WebFarmAchievements returns all achievements with progress
func WebFarmAchievements(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	unlocked, _ := model.GetAchievements(tgId)
	unlockedSet := make(map[string]bool)
	for _, a := range unlocked {
		unlockedSet[a.AchievementKey] = true
	}

	type achInfo struct {
		Key         string  `json:"key"`
		Name        string  `json:"name"`
		Emoji       string  `json:"emoji"`
		Description string  `json:"description"`
		Target      int64   `json:"target"`
		Progress    int64   `json:"progress"`
		Done        bool    `json:"done"`
		Unlocked    bool    `json:"unlocked"`
		Reward      float64 `json:"reward"`
	}
	var achList []achInfo
	farmLevel := model.GetFarmLevel(tgId)
	for _, ach := range achievements {
		var progress int64
		switch ach.Action {
		case "levelup":
			progress = int64(farmLevel)
		default:
			progress = model.CountTotalActions(tgId, ach.Action)
		}
		achList = append(achList, achInfo{
			Key:         ach.Key,
			Name:        ach.Name,
			Emoji:       ach.Emoji,
			Description: ach.Description,
			Target:      ach.Target,
			Progress:    progress,
			Done:        progress >= ach.Target,
			Unlocked:    unlockedSet[ach.Key],
			Reward:      webFarmQuotaFloat(ach.Reward),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"achievements": achList,
		},
	})
}

// WebFarmClaimAchievement claims an achievement reward
func WebFarmClaimAchievement(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	var ach *achievementDef
	for i := range achievements {
		if achievements[i].Key == req.Key {
			ach = &achievements[i]
			break
		}
	}
	if ach == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知成就"})
		return
	}

	if model.HasAchievement(tgId, req.Key) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "成就已领取"})
		return
	}

	var progress int64
	if ach.Action == "levelup" {
		progress = int64(model.GetFarmLevel(tgId))
	} else {
		progress = model.CountTotalActions(tgId, ach.Action)
	}
	if progress < ach.Target {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("成就未达成（%d/%d）", progress, ach.Target)})
		return
	}

	_ = model.UnlockAchievement(tgId, req.Key)
	_ = model.IncreaseUserQuota(user.Id, ach.Reward, true)
	model.AddFarmLog(tgId, "achieve", ach.Reward, fmt.Sprintf("解锁成就:%s", ach.Name))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("解锁 %s %s，奖励 $%.2f", ach.Emoji, ach.Name, webFarmQuotaFloat(ach.Reward)),
	})
}

// ========== 加工坊 ==========

// WebFarmWorkshopView returns workshop status and recipes
func WebFarmWorkshopView(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockWorkshop, "加工坊") {
		return
	}

	procs, _ := model.GetFarmProcesses(tgId)
	now := time.Now().Unix()

	type procInfo struct {
		Id        int     `json:"id"`
		RecipeKey string  `json:"recipe_key"`
		Name      string  `json:"name"`
		Emoji     string  `json:"emoji"`
		Status    int     `json:"status"` // 1=processing, 2=done
		Progress  int     `json:"progress"`
		Remaining int64   `json:"remaining"`
		SellPrice float64 `json:"sell_price"`
	}
	var active []procInfo
	for _, p := range procs {
		status := p.Status
		if status == 1 && now >= p.FinishAt {
			status = 2
		}
		r := recipeMap[p.RecipeKey]
		if r == nil {
			continue
		}
		pi := procInfo{
			Id:        p.Id,
			RecipeKey: p.RecipeKey,
			Name:      r.Name,
			Emoji:     r.Emoji,
			Status:    status,
			SellPrice: webFarmQuotaFloat(applyMarket(r.SellPrice, "recipe_"+r.Key)),
		}
		if status == 1 {
			remain := p.FinishAt - now
			if remain < 0 {
				remain = 0
			}
			total := p.FinishAt - p.StartedAt
			if total > 0 {
				pi.Progress = int((now - p.StartedAt) * 100 / total)
			}
			pi.Remaining = remain
		} else {
			pi.Progress = 100
		}
		active = append(active, pi)
	}

	type recipeInfo struct {
		Key        string  `json:"key"`
		Name       string  `json:"name"`
		Emoji      string  `json:"emoji"`
		Cost       float64 `json:"cost"`
		TimeSecs   int64   `json:"time_secs"`
		SellPrice  float64 `json:"sell_price"`
		Multiplier int     `json:"multiplier"`
		Profit     float64 `json:"profit"`
	}
	var recipeList []recipeInfo
	for _, r := range recipes {
		sellPrice := applyMarket(r.SellPrice, "recipe_"+r.Key)
		m := getMarketMultiplier("recipe_" + r.Key)
		recipeList = append(recipeList, recipeInfo{
			Key:        r.Key,
			Name:       r.Name,
			Emoji:      r.Emoji,
			Cost:       webFarmQuotaFloat(r.Cost),
			TimeSecs:   r.TimeSecs,
			SellPrice:  webFarmQuotaFloat(sellPrice),
			Multiplier: m,
			Profit:     webFarmQuotaFloat(sellPrice - r.Cost),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"active":    active,
			"recipes":   recipeList,
			"max_slots": model.FarmMaxProcessSlots,
			"used_slots": len(procs),
		},
	})
}

// WebFarmWorkshopCraft starts a crafting process
func WebFarmWorkshopCraft(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockWorkshop, "加工坊") {
		return
	}

	var req struct {
		RecipeKey string `json:"recipe_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	r := recipeMap[req.RecipeKey]
	if r == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知配方"})
		return
	}

	count := model.CountActiveProcesses(tgId)
	if count >= int64(model.FarmMaxProcessSlots) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("加工槽已满（%d/%d）", count, model.FarmMaxProcessSlots)})
		return
	}

	err := model.DecreaseUserQuota(user.Id, r.Cost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "余额不足"})
		return
	}

	now := time.Now().Unix()
	proc := &model.TgFarmProcess{
		TelegramId: tgId,
		RecipeKey:  req.RecipeKey,
		StartedAt:  now,
		FinishAt:   now + r.TimeSecs,
		Status:     1,
	}
	_ = model.CreateFarmProcess(proc)
	model.AddFarmLog(tgId, "craft", -r.Cost, fmt.Sprintf("加工%s%s", r.Emoji, r.Name))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("开始加工 %s %s", r.Emoji, r.Name),
	})
}

// WebFarmWorkshopCollect collects all finished products
func WebFarmWorkshopCollect(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockWorkshop, "加工坊") {
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
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有可收取的成品"})
		return
	}

	_ = model.IncreaseUserQuota(user.Id, totalValue, true)
	model.AddFarmLog(tgId, "craft_sell", totalValue, fmt.Sprintf("收取%d件加工品", collected))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("收取 %d 件成品，收入 $%.2f", collected, webFarmQuotaFloat(totalValue)),
		"data": gin.H{
			"count": collected,
			"total": webFarmQuotaFloat(totalValue),
		},
	})
}

// ========== 钓鱼 ==========

// WebFarmFishView returns fish inventory and status
func WebFarmFishView(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockFish, "钱鱼") {
		return
	}

	// 鱼饵数量
	allItems, _ := model.GetFarmItems(tgId)
	baitCount := 0
	for _, item := range allItems {
		if item.ItemType == "fishbait" {
			baitCount = item.Quantity
			break
		}
	}

	// 体力
	stamina, recoverIn := model.GetFishStamina(tgId)

	// 每日统计
	dailyCount := model.GetFishDailyCount(tgId)
	dailyIncome := model.GetFishDailyIncome(tgId)

	// 疲劳
	fatigueActive := common.TgBotFishFatigueEnabled && dailyCount >= common.TgBotFishFatigueThreshold

	// 短CD
	lastFish := model.GetLastFishTime(tgId)
	now := time.Now().Unix()
	cd := int64(common.TgBotFishActionCD)
	if cd < 5 {
		cd = 5
	}
	cdRemain := lastFish + cd - now
	if cdRemain < 0 {
		cdRemain = 0
	}

	// 鱼仓库
	fishItems, _ := model.GetFishItems(tgId)
	type fishInfo struct {
		Key       string  `json:"key"`
		Name      string  `json:"name"`
		Emoji     string  `json:"emoji"`
		Rarity    string  `json:"rarity"`
		Quantity  int     `json:"quantity"`
		UnitPrice float64 `json:"unit_price"`
		TotalVal  float64 `json:"total_value"`
	}
	var inventory []fishInfo
	totalValue := 0
	for _, fi := range fishItems {
		fishKey := fi.ItemType[5:]
		fd := fishTypeMap[fishKey]
		if fd != nil {
			val := fd.SellPrice * fi.Quantity
			totalValue += val
			inventory = append(inventory, fishInfo{
				Key:       fd.Key,
				Name:      fd.Name,
				Emoji:     fd.Emoji,
				Rarity:    fd.Rarity,
				Quantity:  fi.Quantity,
				UnitPrice: webFarmQuotaFloat(fd.SellPrice),
				TotalVal:  webFarmQuotaFloat(val),
			})
		}
	}

	// 鱼种列表（含疲劳调整后概率）
	adjTotal := fishAdjustedTotal(fatigueActive)
	type fishTypeInfo struct {
		Key       string  `json:"key"`
		Name      string  `json:"name"`
		Emoji     string  `json:"emoji"`
		Rarity    string  `json:"rarity"`
		Chance    float64 `json:"chance"`
		SellPrice float64 `json:"sell_price"`
	}
	var types []fishTypeInfo
	for i, ft := range fishTypes {
		w := getFishWeight(i)
		if fatigueActive && (ft.Rarity == "稀有" || ft.Rarity == "史诗" || ft.Rarity == "传说") {
			w = w * (100 - common.TgBotFishFatigueDecay) / 100
		}
		pct := 0.0
		if adjTotal > 0 {
			pct = math.Round(float64(w)*1000.0/float64(adjTotal)) / 10.0
		}
		types = append(types, fishTypeInfo{
			Key:       ft.Key,
			Name:      ft.Name,
			Emoji:     ft.Emoji,
			Rarity:    ft.Rarity,
			Chance:    pct,
			SellPrice: webFarmQuotaFloat(ft.SellPrice),
		})
	}
	nothingPct := 0.0
	if adjTotal > 0 {
		nothingPct = math.Round(float64(common.TgBotFishNothingWeight)*1000.0/float64(adjTotal)) / 10.0
	}

	// 收益CAP状态
	capEnabled := common.TgBotFishIncomeCapEnabled
	dailyIncomeCap := common.TgBotFishDailyIncomeCap
	overCap := capEnabled && dailyIncome >= dailyIncomeCap
	overCapEnabled := common.TgBotFishOverCapEnabled
	overCapRatio := common.TgBotFishOverCapRatio

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"bait_count":        baitCount,
			"cooldown":          cdRemain,
			"stamina":           stamina,
			"stamina_max":       common.TgBotFishStaminaMax,
			"stamina_cost":      common.TgBotFishStaminaCost,
			"recover_in":        recoverIn,
			"recover_amount":    common.TgBotFishStaminaRecoverAmount,
			"daily_count":       dailyCount,
			"daily_max":         common.TgBotFishDailyMaxActions,
			"daily_income":      webFarmQuotaFloat(dailyIncome),
			"daily_max_income":  webFarmQuotaFloat(common.TgBotFishDailyMaxIncome),
			"fatigue_active":    fatigueActive,
			"fatigue_threshold": common.TgBotFishFatigueThreshold,
			"fatigue_decay":     common.TgBotFishFatigueDecay,
			"inventory":         inventory,
			"total_value":       webFarmQuotaFloat(totalValue),
			"fish_types":        types,
			"nothing_chance":    nothingPct,
			"bait_price":        webFarmQuotaFloat(common.TgBotFishBaitPrice),
			// 收益CAP模型
			"cap_enabled":       capEnabled,
			"daily_income_cap":  webFarmQuotaFloat(dailyIncomeCap),
			"over_cap":          overCap,
			"over_cap_enabled":  overCapEnabled,
			"over_cap_ratio":    overCapRatio,
		},
	})
}

// WebFarmFishDo performs a fishing action
func WebFarmFishDo(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockFish, "钱鱼") {
		return
	}

	now := time.Now().Unix()

	// 1. 每日限制检查
	dailyIncome := model.GetFishDailyIncome(tgId)

	// 收益CAP模型：主限制是收益CAP
	if common.TgBotFishIncomeCapEnabled {
		if dailyIncome >= common.TgBotFishDailyIncomeCap {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "今日钓鱼收益已达上限，明天再来吧"})
			return
		}
	} else {
		// 旧模型：仅保留收益上限
		if dailyIncome >= common.TgBotFishDailyMaxIncome {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "今日钓鱼收入已达上限"})
			return
		}
	}

	// 2. 短CD检查（最少5秒）
	lastFish := model.GetLastFishTime(tgId)
	cd := int64(common.TgBotFishActionCD)
	if cd < 5 {
		cd = 5
	}
	if now < lastFish+cd {
		remain := lastFish + cd - now
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("操作太快，请等待 %d 秒", remain)})
		return
	}

	// 3. 扣鱼饵
	err := model.DecrementFarmItem(tgId, "fishbait")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有鱼饵！请先到商店购买"})
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
	fish := randomFishWithFatigue(tgId)

	// 7. 增加每日计数（仅用于统计/任务/疲劳）
	model.IncrFishDailyCount(tgId)

	// 重新读取每日收入（用于判断是否超CAP）
	currentDailyIncome := model.GetFishDailyIncome(tgId)
	overCap := common.TgBotFishIncomeCapEnabled && currentDailyIncome >= common.TgBotFishDailyIncomeCap

	if fish == nil {
		model.AddFarmLog(tgId, "fish", -common.TgBotFishBaitPrice, "钓鱼空军")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "空军！什么都没钓到...",
			"data": gin.H{
				"caught":   false,
				"over_cap": overCap,
			},
		})
		return
	}

	// 8. 计算实际收益（确保不突破当日收益上限）
	fishValue := applyMarket(fish.SellPrice, "fish_"+fish.Key)
	effectiveValue := fishValue
	if common.TgBotFishIncomeCapEnabled {
		remaining := common.TgBotFishDailyIncomeCap - dailyIncome
		if remaining < effectiveValue {
			effectiveValue = remaining
		}
	} else {
		remaining := common.TgBotFishDailyMaxIncome - dailyIncome
		if remaining < effectiveValue {
			effectiveValue = remaining
		}
	}
	if effectiveValue < 0 {
		effectiveValue = 0
	}
	isOverCapCatch := effectiveValue < fishValue

	_ = model.IncrementFarmItem(tgId, "fish_"+fish.Key, 1)
	model.RecordCollection(tgId, "fish", fish.Key, 1)
	model.IncrFishDailyIncome(tgId, effectiveValue)
	model.AddFarmLog(tgId, "fish", 0, fmt.Sprintf("钓到%s%s[%s]", fish.Emoji, fish.Name, fish.Rarity))

	// 钓完后重新检查是否超CAP
	newDailyIncome := model.GetFishDailyIncome(tgId)
	newOverCap := common.TgBotFishIncomeCapEnabled && newDailyIncome >= common.TgBotFishDailyIncomeCap

	msg := fmt.Sprintf("钓到了 %s %s！", fish.Emoji, fish.Name)
	if isOverCapCatch {
		msg = fmt.Sprintf("钓到了 %s %s！本次收益按当日上限封顶结算", fish.Emoji, fish.Name)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": msg,
		"data": gin.H{
			"caught":          true,
			"fish_key":        fish.Key,
			"fish_name":       fish.Name,
			"fish_emoji":      fish.Emoji,
			"rarity":          fish.Rarity,
			"sell_price":      webFarmQuotaFloat(fish.SellPrice),
			"effective_price": webFarmQuotaFloat(effectiveValue),
			"over_cap":        newOverCap,
			"over_cap_catch":  isOverCapCatch,
		},
	})
}

// WebFarmFishSell sells all fish in inventory
func WebFarmFishSell(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockFish, "钱鱼") {
		return
	}

	fishItems, _ := model.GetFishItems(tgId)
	if len(fishItems) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "鱼仓库为空"})
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("卖出 %d 条鱼，收入 $%.2f（含市场波动）", totalCount, webFarmQuotaFloat(totalValue)),
		"data": gin.H{
			"count": totalCount,
			"total": webFarmQuotaFloat(totalValue),
		},
	})
}

// ========== 银行贷款 Web API ==========

// WebFarmBankView returns bank info and active loan
func WebFarmBankView(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	userLevel := model.GetFarmLevel(tgId)
	if userLevel < common.TgBotFarmBankUnlockLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("银行功能需要等级 %d 才能解锁（当前等级 %d）", common.TgBotFarmBankUnlockLevel, userLevel)})
		return
	}

	// 检查信用贷款违约
	creditDefaulted, creditPenalty := model.CheckCreditLoanDefault(tgId)
	if creditDefaulted && creditPenalty == "ban" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你的信用贷款已逾期违约！你的平台账号已被封禁。"})
		return
	}
	// 检查抵押贷款违约
	defaulted, penalty := model.CheckMortgageDefault(tgId)
	if defaulted && penalty == "ban" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你的抵押贷款已逾期违约！由于你等级≥10级，你的平台账号已被封禁。"})
		return
	}

	creditScore := model.GetCreditScore(tgId)
	baseAmount := common.TgBotFarmBankBaseAmount
	maxLoan := baseAmount * creditScore
	interestRate := common.TgBotFarmBankInterestRate
	interest := maxLoan * interestRate / 100
	totalDue := maxLoan + interest
	loanDays := common.TgBotFarmBankMaxLoanDays

	data := gin.H{
		"balance":                webFarmQuotaFloat(user.Quota),
		"credit_score":           creditScore,
		"max_score":              common.TgBotFarmBankMaxMultiplier,
		"max_loan":               webFarmQuotaFloat(maxLoan),
		"interest_rate":          interestRate,
		"interest":               webFarmQuotaFloat(interest),
		"total_due":              webFarmQuotaFloat(totalDue),
		"loan_days":              loanDays,
		"unlock_level":           common.TgBotFarmBankUnlockLevel,
		"has_active_loan":        false,
		"mortgage_blocked":       model.HasMortgageBlocked(tgId),
		"mortgage_max":           webFarmQuotaFloat(common.TgBotFarmMortgageMaxAmount),
		"mortgage_interest_rate": common.TgBotFarmMortgageInterestRate,
	}

	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr == nil && activeLoan != nil {
		remaining := activeLoan.TotalDue - activeLoan.Repaid
		now := time.Now().Unix()
		daysLeft := (activeLoan.DueAt - now) / 86400
		if daysLeft < 0 {
			daysLeft = 0
		}
		data["has_active_loan"] = true
		data["active_loan"] = gin.H{
			"id":        activeLoan.Id,
			"principal": webFarmQuotaFloat(activeLoan.Principal),
			"interest":  webFarmQuotaFloat(activeLoan.Interest),
			"total_due": webFarmQuotaFloat(activeLoan.TotalDue),
			"repaid":    webFarmQuotaFloat(activeLoan.Repaid),
			"remaining": webFarmQuotaFloat(remaining),
			"due_at":    activeLoan.DueAt,
			"days_left": daysLeft,
			"overdue":   now > activeLoan.DueAt,
			"loan_type": activeLoan.LoanType,
		}
	}

	history, _ := model.GetLoanHistory(tgId, 10)
	var historyList []gin.H
	for _, loan := range history {
		historyList = append(historyList, gin.H{
			"id":           loan.Id,
			"principal":    webFarmQuotaFloat(loan.Principal),
			"interest":     webFarmQuotaFloat(loan.Interest),
			"total_due":    webFarmQuotaFloat(loan.TotalDue),
			"repaid":       webFarmQuotaFloat(loan.Repaid),
			"status":       loan.Status,
			"loan_type":    loan.LoanType,
			"credit_score": loan.CreditScore,
			"created_at":   loan.CreatedAt,
			"due_at":       loan.DueAt,
		})
	}
	data["history"] = historyList

	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// WebFarmMortgageLoan applies for a mortgage loan
func WebFarmMortgageLoan(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmBankUnlockLevel, "银行") {
		return
	}

	var req struct {
		Amount int `json:"amount"`
	}
	maxDollar := common.TgBotFarmMortgageMaxAmount / 500000
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount < 1 || req.Amount > maxDollar {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("金额必须在 $1 ~ $%d 之间", maxDollar)})
		return
	}

	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr == nil && activeLoan != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你还有未还清的贷款！请先还清再申请。"})
		return
	}

	principal := req.Amount * 500000
	if principal > common.TgBotFarmMortgageMaxAmount {
		principal = common.TgBotFarmMortgageMaxAmount
	}
	interestRate := common.TgBotFarmMortgageInterestRate
	interest := principal * interestRate / 100
	totalDue := principal + interest
	loanDays := common.TgBotFarmBankMaxLoanDays
	creditScore := model.GetCreditScore(tgId)

	loan, err := model.CreateLoanWithType(tgId, principal, interest, totalDue, creditScore, loanDays, 1)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "抵押贷款申请失败"})
		return
	}

	_ = model.IncreaseUserQuota(user.Id, principal, true)
	model.AddFarmLog(tgId, "loan", principal, fmt.Sprintf("抵押贷款$%d", req.Amount))
	common.SysLog(fmt.Sprintf("TG Farm Mortgage: user %s loan $%d", tgId, req.Amount))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("抵押贷款成功！获得 $%d", req.Amount),
		"data": gin.H{
			"loan_id":      loan.Id,
			"principal":    webFarmQuotaFloat(principal),
			"interest":     webFarmQuotaFloat(interest),
			"total_due":    webFarmQuotaFloat(totalDue),
			"credit_score": creditScore,
			"due_at":       loan.DueAt,
			"loan_type":    1,
		},
	})
}

// WebFarmBankLoan applies for a new loan
func WebFarmBankLoan(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	userLevel := model.GetFarmLevel(tgId)
	if userLevel < common.TgBotFarmBankUnlockLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "银行功能未解锁"})
		return
	}

	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr == nil && activeLoan != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你还有未还清的贷款！请先还清再申请。"})
		return
	}

	creditScore := model.GetCreditScore(tgId)
	baseAmount := common.TgBotFarmBankBaseAmount
	principal := baseAmount * creditScore
	interestRate := common.TgBotFarmBankInterestRate
	interest := principal * interestRate / 100
	totalDue := principal + interest
	loanDays := common.TgBotFarmBankMaxLoanDays

	loan, err := model.CreateLoan(tgId, principal, interest, totalDue, creditScore, loanDays)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "贷款申请失败"})
		return
	}

	_ = model.IncreaseUserQuota(user.Id, principal, true)
	model.AddFarmLog(tgId, "loan", principal, fmt.Sprintf("银行贷款 评分%d", creditScore))
	common.SysLog(fmt.Sprintf("TG Farm Bank: user %s loan %d quota, score %d", tgId, principal, creditScore))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("贷款成功！获得 $%.2f", webFarmQuotaFloat(principal)),
		"data": gin.H{
			"loan_id":      loan.Id,
			"principal":    webFarmQuotaFloat(principal),
			"interest":     webFarmQuotaFloat(interest),
			"total_due":    webFarmQuotaFloat(totalDue),
			"credit_score": creditScore,
			"due_at":       loan.DueAt,
		},
	})
}

// WebFarmBankRepay repays a loan (percent: 50 or 100)
func WebFarmBankRepay(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmBankUnlockLevel, "银行") {
		return
	}

	var req struct {
		Percent int `json:"percent"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Percent = 100
	}
	if req.Percent <= 0 || req.Percent > 100 {
		req.Percent = 100
	}

	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr != nil || activeLoan == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有待还贷款"})
		return
	}

	remaining := activeLoan.TotalDue - activeLoan.Repaid
	repayAmount := remaining
	if req.Percent < 100 {
		repayAmount = remaining * req.Percent / 100
		if repayAmount < 1 {
			repayAmount = 1
		}
	}

	if user.Quota < repayAmount {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！需要 $%.2f，余额 $%.2f", webFarmQuotaFloat(repayAmount), webFarmQuotaFloat(user.Quota))})
		return
	}

	err := model.DecreaseUserQuota(user.Id, repayAmount)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

	loan, err := model.RepayLoan(activeLoan.Id, repayAmount)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, repayAmount, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "还款失败，已退款"})
		return
	}

	model.AddFarmLog(tgId, "repay", -repayAmount, fmt.Sprintf("还款%d%%", req.Percent))
	common.SysLog(fmt.Sprintf("TG Farm Bank: user %s repaid %d quota", tgId, repayAmount))

	newRemaining := loan.TotalDue - loan.Repaid
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("还款成功！还款 $%.2f", webFarmQuotaFloat(repayAmount)),
		"data": gin.H{
			"repaid":    webFarmQuotaFloat(repayAmount),
			"remaining": webFarmQuotaFloat(newRemaining),
			"cleared":   loan.Status == 1,
		},
	})
}

// ========== 季节 & 仓库 ==========

// WebFarmSeasonInfo 获取当前季节信息
func WebFarmSeasonInfo(c *gin.Context) {
	season := getCurrentSeason()
	daysLeft := getSeasonDaysLeft()
	// 各作物季节和当前售价
	nowTs := time.Now().Unix()
	var crops []gin.H
	for _, crop := range farmCrops {
		marketPrice := applyMarket(crop.UnitPrice, "crop_"+crop.Key)
		seasonPrice := applySeasonPrice(marketPrice, &crop)
		tierKey, tierName := getCropTier(&crop)
		crops = append(crops, gin.H{
			"key":              crop.Key,
			"name":             crop.Name,
			"emoji":            crop.Emoji,
			"season":           crop.Season,
			"season_name":      seasonNames[crop.Season],
			"in_season":        isCropInSeason(&crop),
			"unit_price":       webFarmQuotaFloat(crop.UnitPrice),
			"market_price":     webFarmQuotaFloat(marketPrice),
			"season_price":     webFarmQuotaFloat(seasonPrice),
			"season_pct":       getSeasonPriceMultiplier(&crop),
			"season_grow_pct":  getSeasonGrowthMultiplier(&crop, nowTs),
			"season_yield_pct": getSeasonYieldMultiplier(&crop, nowTs),
			"tier":             tierKey,
			"tier_name":        tierName,
			"tags":             getCropTags(&crop),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"season":            season,
			"season_name":       seasonNames[season],
			"season_emoji":      seasonEmojis[season],
			"days_left":         daysLeft,
			"season_days":       common.TgBotFarmSeasonDays,
			"in_bonus":          common.TgBotFarmSeasonInBonus,
			"off_bonus":         common.TgBotFarmSeasonOffBonus,
			"in_growth":         common.TgBotFarmSeasonInGrowth,
			"off_growth":        common.TgBotFarmSeasonOffGrowth,
			"in_yield":          common.TgBotFarmSeasonInYield,
			"off_yield":         common.TgBotFarmSeasonOffYield,
			"off_event_bonus":   common.TgBotFarmSeasonOffEventBonus,
			"off_water_penalty": common.TgBotFarmSeasonOffWaterPenalty,
			"crops":             crops,
		},
	})
}

// WebFarmWarehouseView 查看仓库
func WebFarmWarehouseView(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	whLevel := model.GetWarehouseLevel(tgId)
	maxSlots := model.GetWarehouseMaxSlots(whLevel)
	expiryMultiplier := model.GetWarehouseExpiryMultiplier(whLevel)

	items, _ := model.GetWarehouseItems(tgId)
	totalCount := model.GetWarehouseTotalCount(tgId)
	season := getCurrentSeason()
	now := time.Now().Unix()

	var itemList []gin.H
	for _, item := range items {
		emoji, name := warehouseItemName(item)
		unitPrice := warehouseItemSellPrice(item)

		entry := gin.H{
			"item_key":    item.CropType,
			"category":    item.Category,
			"name":        name,
			"emoji":       emoji,
			"quantity":    item.Quantity,
			"unit_price":  webFarmQuotaFloat(unitPrice),
			"total_value": webFarmQuotaFloat(item.Quantity * unitPrice),
			"stored_at":   item.StoredAt,
		}

		if item.Category == "crop" {
			crop := farmCropMap[item.CropType]
			if crop != nil {
				entry["in_season"] = isCropInSeason(crop)
				entry["season_pct"] = getSeasonPriceMultiplier(crop)
			}
		} else if item.Category == "meat" {
			expiry := int64(common.TgBotFarmWarehouseMeatExpiry) * int64(expiryMultiplier) / 100
			entry["expire_at"] = item.StoredAt + expiry
			entry["expire_remain"] = item.StoredAt + expiry - now
		} else if item.Category == "recipe" {
			expiry := int64(common.TgBotFarmWarehouseRecipeExpiry) * int64(expiryMultiplier) / 100
			entry["expire_at"] = item.StoredAt + expiry
			entry["expire_remain"] = item.StoredAt + expiry - now
		}

		itemList = append(itemList, entry)
	}

	// 计算下一级升级信息
	nextLevel := whLevel + 1
	canUpgrade := whLevel < common.TgBotFarmWarehouseMaxLevel
	upgradePrice := common.TgBotFarmWarehouseUpgradePrice * whLevel // 每级递增

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":            itemList,
			"total":            totalCount,
			"max_slots":        maxSlots,
			"warehouse_level":  whLevel,
			"max_level":        common.TgBotFarmWarehouseMaxLevel,
			"can_upgrade":      canUpgrade,
			"upgrade_price":    webFarmQuotaFloat(upgradePrice),
			"next_capacity":    model.GetWarehouseMaxSlots(nextLevel),
			"next_expiry_pct":  model.GetWarehouseExpiryMultiplier(nextLevel),
			"expiry_pct":       expiryMultiplier,
			"season":           season,
			"season_name":      seasonNames[season],
			"days_left":        getSeasonDaysLeft(),
			"meat_expiry":      int64(common.TgBotFarmWarehouseMeatExpiry) * int64(expiryMultiplier) / 100,
			"recipe_expiry":    int64(common.TgBotFarmWarehouseRecipeExpiry) * int64(expiryMultiplier) / 100,
		},
	})
}

// WebFarmWarehouseSell 从仓库出售指定物品
func WebFarmWarehouseSell(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		ItemKey string `json:"item_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ItemKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请指定物品类型"})
		return
	}

	item, err := model.GetWarehouseItem(tgId, req.ItemKey)
	if err != nil || item.Quantity <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仓库中没有该物品"})
		return
	}

	unitPrice := warehouseItemSellPrice(item)
	totalValue := item.Quantity * unitPrice
	_, name := warehouseItemName(item)

	_ = model.RemoveFromWarehouse(tgId, req.ItemKey, item.Quantity)
	_ = model.IncreaseUserQuota(user.Id, totalValue, true)
	model.RecordMarketSell(req.ItemKey, item.Quantity)
	model.AddFarmLog(tgId, "warehouse_sell", totalValue, fmt.Sprintf("仓库出售%s×%d", name, item.Quantity))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("出售成功！获得 $%.2f", webFarmQuotaFloat(totalValue)),
		"data": gin.H{
			"name":     name,
			"quantity": item.Quantity,
			"earned":   webFarmQuotaFloat(totalValue),
		},
	})
}

// WebFarmWarehouseSellAll 从仓库出售全部
func WebFarmWarehouseSellAll(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	items, err := model.GetWarehouseItems(tgId)
	if err != nil || len(items) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仓库为空"})
		return
	}

	totalValue := 0
	for _, item := range items {
		unitPrice := warehouseItemSellPrice(item)
		totalValue += item.Quantity * unitPrice
		model.RecordMarketSell(item.CropType, item.Quantity)
		_ = model.RemoveFromWarehouse(tgId, item.CropType, item.Quantity)
	}

	_ = model.IncreaseUserQuota(user.Id, totalValue, true)
	model.AddFarmLog(tgId, "warehouse_sell", totalValue, "仓库全部出售")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("全部出售成功！获得 $%.2f", webFarmQuotaFloat(totalValue)),
		"data": gin.H{
			"earned": webFarmQuotaFloat(totalValue),
		},
	})
}

// WebFarmFishStore 鱼存入仓库
func WebFarmFishStore(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockFish, "钓鱼") {
		return
	}

	fishItems, _ := model.GetFishItems(tgId)
	if len(fishItems) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有可存入的鱼"})
		return
	}

	currentTotal := model.GetWarehouseTotalCount(tgId)
	storedCount := 0
	var stored []gin.H
	for _, fi := range fishItems {
		if len(fi.ItemType) <= 5 {
			continue
		}
		fishKey := fi.ItemType[5:]
		fd := fishTypeMap[fishKey]
		if fd == nil {
			continue
		}
		whLevel := model.GetWarehouseLevel(tgId)
		whMax := model.GetWarehouseMaxSlots(whLevel)
		if currentTotal+storedCount+fi.Quantity > whMax {
			continue
		}
		_ = model.AddToWarehouseWithCategory(tgId, "fish_"+fishKey, fi.Quantity, "fish")
		storedCount += fi.Quantity
		stored = append(stored, gin.H{"name": fd.Name, "quantity": fi.Quantity})
	}
	if storedCount > 0 {
		_, _ = model.SellAllFish(tgId)
	}

	if storedCount == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仓库已满，无法存入"})
		return
	}

	model.AddFarmLog(tgId, "fish_store", 0, fmt.Sprintf("鱼存入仓库%d条", storedCount))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("存入仓库成功！共%d条鱼", storedCount),
		"data":    gin.H{"stored": stored, "total": storedCount},
	})
}

// WebFarmWorkshopCollectStore 加工品存入仓库
func WebFarmWorkshopCollectStore(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockWorkshop, "加工坊") {
		return
	}

	procs, _ := model.GetFarmProcesses(tgId)
	now := time.Now().Unix()
	currentTotal := model.GetWarehouseTotalCount(tgId)
	stored := 0
	var details []gin.H
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
				continue
			}
			_ = model.AddToWarehouseWithCategory(tgId, "recipe_"+r.Key, 1, "recipe")
			stored++
			_ = model.CollectFarmProcess(p.Id)
			model.RecordCollection(tgId, "recipe", r.Key, 1)
			details = append(details, gin.H{"name": r.Name, "quantity": 1})
		}
	}

	if stored == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有可收取的成品或仓库已满"})
		return
	}

	model.AddFarmLog(tgId, "craft_store", 0, fmt.Sprintf("加工品存入仓库%d件", stored))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("存入仓库成功！共%d件加工品（5天后发霉）", stored),
		"data":    gin.H{"stored": details, "total": stored},
	})
}

// WebFarmHarvestStore 收获到仓库
func WebFarmHarvestStore(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	currentTotal := model.GetWarehouseTotalCount(tgId)
	harvestedCount := 0
	storedTotal := 0
	var stored []gin.H
	for _, plot := range plots {
		if plot.Status == 2 {
			crop := farmCropMap[plot.CropType]
			if crop == nil {
				continue
			}
			yield := 1 + rand.Intn(crop.MaxYield)
			if plot.Fertilized == 1 {
				bonus := yield / 2
				if bonus < 1 {
					bonus = 1
				}
				yield += bonus
			}
			realYield := yield - plot.StolenCount
			if realYield < 0 {
				realYield = 0
			}
			whLevel := model.GetWarehouseLevel(tgId)
			whMax := model.GetWarehouseMaxSlots(whLevel)
			if currentTotal+storedTotal+realYield > whMax {
				continue
			}
			_ = model.AddToWarehouse(tgId, crop.Key, realYield)
			_ = model.ClearFarmPlot(plot.Id)
			storedTotal += realYield
			harvestedCount++
			model.RecordCollection(tgId, "crop", crop.Key, realYield)
			stored = append(stored, gin.H{"crop": crop.Name, "quantity": realYield})
		}
	}

	if harvestedCount == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有可收获的作物或仓库已满"})
		return
	}

	model.AddFarmLog(tgId, "harvest", 0, fmt.Sprintf("收获入仓%d种共%d个", harvestedCount, storedTotal))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("收获入仓完成！存入%d个作物", storedTotal),
		"data": gin.H{
			"stored": stored,
			"total":  storedTotal,
		},
	})
}

// WebFarmWarehouseUpgrade 升级仓库等级
func WebFarmWarehouseUpgrade(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	whLevel := model.GetWarehouseLevel(tgId)
	maxLevel := common.TgBotFarmWarehouseMaxLevel
	if whLevel >= maxLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("仓库已达最高等级 %d", maxLevel)})
		return
	}

	upgradePrice := common.TgBotFarmWarehouseUpgradePrice * whLevel
	if user.Quota < upgradePrice {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！升级需要 $%.2f，当前余额 $%.2f", webFarmQuotaFloat(upgradePrice), webFarmQuotaFloat(user.Quota))})
		return
	}

	err := model.DecreaseUserQuota(user.Id, upgradePrice)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

	newLevel := whLevel + 1
	err = model.SetWarehouseLevel(tgId, newLevel)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, upgradePrice, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "升级失败，已退款"})
		return
	}

	newCapacity := model.GetWarehouseMaxSlots(newLevel)
	newExpiryPct := model.GetWarehouseExpiryMultiplier(newLevel)
	model.AddFarmLog(tgId, "warehouse_upgrade", -upgradePrice, fmt.Sprintf("仓库升级至%d级", newLevel))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("仓库升级成功！等级 %d → %d", whLevel, newLevel),
		"data": gin.H{
			"level":      newLevel,
			"capacity":   newCapacity,
			"expiry_pct": newExpiryPct,
			"cost":       webFarmQuotaFloat(upgradePrice),
		},
	})
}

// WebFarmClearPlot 铲除地块上的作物
func WebFarmClearPlot(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		PlotIndex int `json:"plot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	var target *model.TgFarmPlot
	for _, plot := range plots {
		if plot.PlotIndex == req.PlotIndex {
			target = plot
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该地块不存在"})
		return
	}
	if target.Status == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该地块已经是空的"})
		return
	}
	cropName := target.CropType
	crop := farmCropMap[target.CropType]
	if crop != nil {
		cropName = crop.Emoji + crop.Name
	}
	_ = model.ClearFarmPlot(target.Id)
	model.AddFarmLog(tgId, "clear_plot", 0, fmt.Sprintf("铲除%d号地%s", req.PlotIndex+1, cropName))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("已铲除 %d号地 的 %s", req.PlotIndex+1, cropName)})
}

// WebRanchRelease 放生牧场动物（清空槽位）
func WebRanchRelease(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		AnimalId int `json:"animal_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.AnimalId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	var found bool
	var animalName string
	for _, a := range animals {
		if a.Id == req.AnimalId {
			found = true
			def := ranchAnimalMap[a.AnimalType]
			if def != nil {
				animalName = def.Emoji + def.Name
			} else {
				animalName = a.AnimalType
			}
			break
		}
	}
	if !found {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未找到该动物"})
		return
	}
	_ = model.DeleteRanchAnimal(req.AnimalId)
	model.AddFarmLog(tgId, "ranch_release", 0, fmt.Sprintf("放生%s", animalName))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("已放生 %s", animalName)})
}

// WebWorkshopCancel 取消加工坊槽位
func WebWorkshopCancel(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		ProcessId int `json:"process_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ProcessId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	procs, err := model.GetFarmProcesses(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	var found bool
	var recipeName string
	for _, p := range procs {
		if p.Id == req.ProcessId {
			found = true
			recipeName = p.RecipeKey
			break
		}
	}
	if !found {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未找到该加工任务"})
		return
	}
	_ = model.DeleteFarmProcess(req.ProcessId)
	model.AddFarmLog(tgId, "workshop_cancel", 0, fmt.Sprintf("取消加工%s", recipeName))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已取消该加工任务"})
}
