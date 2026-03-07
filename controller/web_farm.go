package controller

import (
	"fmt"
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
	if user.TelegramId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先绑定 Telegram 账号后才能使用农场功能"})
		return nil, "", false
	}
	return user, user.TelegramId, true
}

func webFarmQuotaFloat(quota int) float64 {
	return float64(quota) / common.QuotaPerUnit
}

// checkFeatureLevel 检查用户是否达到功能解锁等级，未达到则返回错误并return false
func checkFeatureLevel(c *gin.Context, tgId string, requiredLevel int, featureName string) bool {
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
	DeathRemain   int64   `json:"death_remain"`
	StatusLabel   string  `json:"status_label"`
	SoilLevel     int     `json:"soil_level"`
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
			waterInterval := int64(common.TgBotFarmWaterInterval)
			nextWater := plot.LastWateredAt + waterInterval - now
			info.WaterRemain = nextWater
		}
	case 2:
		info.StatusLabel = "已成熟"
	case 3:
		if plot.EventType == "drought" {
			info.StatusLabel = "天灾干旱"
			wiltDuration := int64(common.TgBotFarmWiltDuration)
			deathAt := plot.EventAt + wiltDuration
			info.DeathRemain = deathAt - now
			if info.DeathRemain < 0 {
				info.DeathRemain = 0
			}
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
			})
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
			}
		}(),
			"prestige_level": model.GetPrestigeLevel(tgId),
			"prestige_bonus": model.GetPrestigeLevel(tgId) * common.TgBotFarmPrestigeBonusPerLevel,
		},
	})
}

// WebFarmCrops returns available crops
func WebFarmCrops(c *gin.Context) {
	var crops []map[string]interface{}
	for _, crop := range farmCrops {
		crops = append(crops, map[string]interface{}{
			"key":        crop.Key,
			"short":      crop.Short,
			"name":       crop.Name,
			"emoji":      crop.Emoji,
			"seed_cost":  webFarmQuotaFloat(crop.SeedCost),
			"grow_secs":  crop.GrowSecs,
			"max_yield":  crop.MaxYield,
			"unit_price": webFarmQuotaFloat(crop.UnitPrice),
			"max_value":  webFarmQuotaFloat(crop.MaxYield * crop.UnitPrice),
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
	if user.Quota < crop.SeedCost {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！种子需要 $%.2f", webFarmQuotaFloat(crop.SeedCost))})
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

	err = model.DecreaseUserQuota(user.Id, crop.SeedCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
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

	if rand.Intn(100) < common.TgBotFarmEventChance {
		targetPlot.EventType = "bugs"
		offset := webActualGrowSecs * int64(30+rand.Intn(50)) / 100
		targetPlot.EventAt = now + offset
	}
	if targetPlot.EventType == "" && rand.Intn(100) < common.TgBotFarmDisasterChance {
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

	totalQuota := 0
	harvestedCount := 0
	var details []map[string]interface{}

	for _, plot := range plots {
		if plot.Status == 2 {
			crop := farmCropMap[plot.CropType]
			if crop == nil {
				continue
			}
			yield := 1 + rand.Intn(crop.MaxYield)
			fertBonus := 0
			if plot.Fertilized == 1 {
				fertBonus = yield / 2
				if fertBonus < 1 {
					fertBonus = 1
				}
				yield += fertBonus
			}
			loss := plot.StolenCount
			realYield := yield - loss
			if realYield < 0 {
				realYield = 0
			}
			marketPrice := applyMarket(crop.UnitPrice, "crop_"+crop.Key)
			seasonPrice := applySeasonPrice(marketPrice, crop)
			value := realYield * seasonPrice
			totalQuota += value
			harvestedCount++

			details = append(details, map[string]interface{}{
				"crop_name":    crop.Name,
				"crop_emoji":   crop.Emoji,
				"yield":        yield - fertBonus,
				"fert_bonus":   fertBonus,
				"stolen":       loss,
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
	item := farmItemMap[req.ItemKey]
	if item == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知道具"})
		return
	}
	unitCost := item.Cost
	if req.ItemKey == "dogfood" {
		unitCost = common.TgBotFarmDogFoodPrice
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
	err = model.IncrementFarmItem(tgId, req.ItemKey, req.Quantity)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, totalCost, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "购买失败"})
		return
	}
	model.AddFarmLog(tgId, "shop", -totalCost, fmt.Sprintf("购买%s%s×%d", item.Emoji, item.Name, req.Quantity))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("购买 %s%s ×%d 成功！", item.Emoji, item.Name, req.Quantity),
		"data": gin.H{
			"item":       req.ItemKey,
			"quantity":   req.Quantity,
			"total_cost": webFarmQuotaFloat(totalCost),
		},
	})
}

// WebFarmStealTargets returns available steal targets
func WebFarmStealTargets(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockSteal, "偷菜") {
		return
	}
	targets, err := model.GetMatureFarmTargets(tgId)
	if err != nil || len(targets) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
		return
	}
	var result []map[string]interface{}
	for _, t := range targets {
		result = append(result, map[string]interface{}{
			"id":    t.TelegramId,
			"label": maskTgId(t.TelegramId),
			"count": t.Count,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// WebFarmSteal steals from another player
func WebFarmSteal(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockSteal, "偷菜") {
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

	now := time.Now().Unix()
	recentSteals, _ := model.CountRecentSteals(tgId, req.VictimId, now-int64(common.TgBotFarmStealCooldown))
	if recentSteals > 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("冷却中！%d分钟内只能偷同一人一次", common.TgBotFarmStealCooldown/60)})
		return
	}

	// Check victim's dog
	victimDog, dogErr := model.GetFarmDog(req.VictimId)
	if dogErr == nil {
		model.UpdateDogHunger(victimDog)
		if victimDog.Level == 2 && victimDog.Hunger > 0 {
			if rand.Intn(100) < common.TgBotFarmDogGuardRate {
				c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("对方的看门狗「%s」发现了你，偷菜失败！", victimDog.Name)})
				return
			}
		}
	}

	plots, err := model.GetStealablePlots(req.VictimId)
	if err != nil || len(plots) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该玩家没有可偷的成熟作物"})
		return
	}

	target := plots[rand.Intn(len(plots))]
	crop := farmCropMap[target.CropType]
	cropName := "作物"
	cropEmoji := "🌿"
	unitPrice := 10000
	if crop != nil {
		cropName = crop.Name
		cropEmoji = crop.Emoji
		unitPrice = crop.UnitPrice
	}

	stealUnits := 1 + rand.Intn(3)
	stealValue := stealUnits * unitPrice

	_ = model.IncrementPlotStolenCount(target.Id)
	_ = model.CreateFarmStealLog(&model.TgFarmStealLog{
		ThiefId:  tgId,
		VictimId: req.VictimId,
		PlotId:   target.Id,
		Amount:   stealValue,
	})
	_ = model.IncreaseUserQuota(user.Id, stealValue, true)
	model.AddFarmLog(tgId, "steal", stealValue, fmt.Sprintf("偷取%s%s×%d", cropEmoji, cropName, stealUnits))

	common.SysLog(fmt.Sprintf("Web Farm: user %s stole %s from %s, +%d quota", tgId, cropName, req.VictimId, stealValue))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("偷了 %d个%s%s，获得 $%.2f", stealUnits, cropEmoji, cropName, webFarmQuotaFloat(stealValue)),
		"data": gin.H{
			"victim":     maskTgId(req.VictimId),
			"crop_name":  cropName,
			"crop_emoji": cropEmoji,
			"units":      stealUnits,
			"value":      webFarmQuotaFloat(stealValue),
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockDog, "狗狗") {
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockDog, "狗狗") {
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockDog, "狗狗") {
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

// WebFarmMarket returns current market prices
func WebFarmMarket(c *gin.Context) {
	ensureMarketFresh()

	type priceInfo struct {
		Key        string  `json:"key"`
		Name       string  `json:"name"`
		Emoji      string  `json:"emoji"`
		Category   string  `json:"category"`
		BasePrice  float64 `json:"base_price"`
		Multiplier int     `json:"multiplier"`
		CurPrice   float64 `json:"cur_price"`
	}

	var prices []priceInfo

	// 作物
	for _, crop := range farmCrops {
		m := getMarketMultiplier("crop_" + crop.Key)
		prices = append(prices, priceInfo{
			Key:        "crop_" + crop.Key,
			Name:       crop.Name,
			Emoji:      crop.Emoji,
			Category:   "crop",
			BasePrice:  webFarmQuotaFloat(crop.UnitPrice),
			Multiplier: m,
			CurPrice:   webFarmQuotaFloat(applyMarket(crop.UnitPrice, "crop_"+crop.Key)),
		})
	}

	// 鱼
	for _, fish := range fishTypes {
		m := getMarketMultiplier("fish_" + fish.Key)
		prices = append(prices, priceInfo{
			Key:        "fish_" + fish.Key,
			Name:       fish.Name,
			Emoji:      fish.Emoji,
			Category:   "fish",
			BasePrice:  webFarmQuotaFloat(fish.SellPrice),
			Multiplier: m,
			CurPrice:   webFarmQuotaFloat(applyMarket(fish.SellPrice, "fish_"+fish.Key)),
		})
	}

	// 肉类
	for _, a := range ranchAnimals {
		m := getMarketMultiplier("meat_" + a.Key)
		prices = append(prices, priceInfo{
			Key:        "meat_" + a.Key,
			Name:       a.Name + "肉",
			Emoji:      a.Emoji,
			Category:   "meat",
			BasePrice:  webFarmQuotaFloat(*a.MeatPrice),
			Multiplier: m,
			CurPrice:   webFarmQuotaFloat(applyMarket(*a.MeatPrice, "meat_"+a.Key)),
		})
	}

	// 加工品
	for _, r := range recipes {
		m := getMarketMultiplier("recipe_" + r.Key)
		prices = append(prices, priceInfo{
			Key:        "recipe_" + r.Key,
			Name:       r.Name,
			Emoji:      r.Emoji,
			Category:   "recipe",
			BasePrice:  webFarmQuotaFloat(r.SellPrice),
			Multiplier: m,
			CurPrice:   webFarmQuotaFloat(applyMarket(r.SellPrice, "recipe_"+r.Key)),
		})
	}

	marketMu.RLock()
	nextRefresh := marketNextUpdate - time.Now().Unix()
	marketMu.RUnlock()
	if nextRefresh < 0 {
		nextRefresh = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"prices":       prices,
			"next_refresh": nextRefresh,
			"refresh_hours": common.TgBotMarketRefreshHours,
		},
	})
}

// WebFarmMarketHistory returns market price history for chart rendering
func WebFarmMarketHistory(c *gin.Context) {
	ensureMarketFresh()

	marketMu.RLock()
	history := make([]marketSnapshot, len(marketHistory))
	copy(history, marketHistory)
	marketMu.RUnlock()

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
	for _, crop := range farmCrops {
		items = append(items, itemMeta{"crop_" + crop.Key, crop.Name, crop.Emoji, "crop"})
	}
	for _, fish := range fishTypes {
		items = append(items, itemMeta{"fish_" + fish.Key, fish.Name, fish.Emoji, "fish"})
	}
	for _, a := range ranchAnimals {
		items = append(items, itemMeta{"meat_" + a.Key, a.Name + "肉", a.Emoji, "meat"})
	}
	for _, r := range recipes {
		items = append(items, itemMeta{"recipe_" + r.Key, r.Name, r.Emoji, "recipe"})
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

	// 有未还贷款时禁止升级
	activeLoan, loanErr := model.GetActiveLoan(tgId)
	if loanErr == nil && activeLoan != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你有未还清的贷款，还清后才能升级！贷款资金不能用于升级。"})
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
		Index    int     `json:"index"`
		Action   string  `json:"action"`
		Name     string  `json:"name"`
		Emoji    string  `json:"emoji"`
		Target   int     `json:"target"`
		Progress int64   `json:"progress"`
		Done     bool    `json:"done"`
		Claimed  bool    `json:"claimed"`
		Reward   float64 `json:"reward"`
	}
	var taskList []taskInfo
	for i, task := range tasks {
		progress := model.CountTodayActions(tgId, task.Action)
		taskList = append(taskList, taskInfo{
			Index:    i,
			Action:   task.Action,
			Name:     task.Name,
			Emoji:    task.Emoji,
			Target:   task.Target,
			Progress: progress,
			Done:     progress >= int64(task.Target),
			Claimed:  claimedSet[i],
			Reward:   webFarmQuotaFloat(task.Reward),
		})
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
	for _, ach := range achievements {
		progress := model.CountTotalActions(tgId, ach.Action)
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

	progress := model.CountTotalActions(tgId, ach.Action)
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockWorkshop, "加工坊") {
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockWorkshop, "加工坊") {
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockWorkshop, "加工坊") {
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockFish, "钱鱼") {
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

	// 冷却
	lastFish := model.GetLastFishTime(tgId)
	now := time.Now().Unix()
	cd := int64(common.TgBotFishCooldown)
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

	// 鱼种列表
	type fishTypeInfo struct {
		Key       string  `json:"key"`
		Name      string  `json:"name"`
		Emoji     string  `json:"emoji"`
		Rarity    string  `json:"rarity"`
		Chance    int     `json:"chance"`
		SellPrice float64 `json:"sell_price"`
	}
	var types []fishTypeInfo
	for _, ft := range fishTypes {
		types = append(types, fishTypeInfo{
			Key:       ft.Key,
			Name:      ft.Name,
			Emoji:     ft.Emoji,
			Rarity:    ft.Rarity,
			Chance:    ft.Weight * 100 / fishTotalWeight,
			SellPrice: webFarmQuotaFloat(ft.SellPrice),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"bait_count":    baitCount,
			"cooldown":      cdRemain,
			"inventory":     inventory,
			"total_value":   webFarmQuotaFloat(totalValue),
			"fish_types":    types,
			"nothing_chance": fishNothingWeight * 100 / fishTotalWeight,
			"bait_price":    webFarmQuotaFloat(common.TgBotFishBaitPrice),
		},
	})
}

// WebFarmFishDo performs a fishing action
func WebFarmFishDo(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockFish, "钱鱼") {
		return
	}

	// 冷却检查
	lastFish := model.GetLastFishTime(tgId)
	now := time.Now().Unix()
	cd := int64(common.TgBotFishCooldown)
	if now < lastFish+cd {
		remain := lastFish + cd - now
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("冷却中，还需等待 %d 秒", remain)})
		return
	}

	// 扣鱼饵
	err := model.DecrementFarmItem(tgId, "fishbait")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有鱼饵！请先到商店购买"})
		return
	}

	// 记录冷却
	model.SetLastFishTime(tgId, now)

	// 随机钓鱼
	fish := randomFish()
	if fish == nil {
		model.AddFarmLog(tgId, "fish", -common.TgBotFishBaitPrice, "钓鱼空军")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "空军！什么都没钓到...",
			"data": gin.H{
				"caught": false,
			},
		})
		return
	}

	_ = model.IncrementFarmItem(tgId, "fish_"+fish.Key, 1)
	model.RecordCollection(tgId, "fish", fish.Key, 1)
	model.AddFarmLog(tgId, "fish", 0, fmt.Sprintf("钓到%s%s[%s]", fish.Emoji, fish.Name, fish.Rarity))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("钓到了 %s %s！", fish.Emoji, fish.Name),
		"data": gin.H{
			"caught":     true,
			"fish_key":   fish.Key,
			"fish_name":  fish.Name,
			"fish_emoji": fish.Emoji,
			"rarity":     fish.Rarity,
			"sell_price": webFarmQuotaFloat(fish.SellPrice),
		},
	})
}

// WebFarmFishSell sells all fish in inventory
func WebFarmFishSell(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockFish, "钱鱼") {
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmBankUnlockLevel, "银行") {
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmBankUnlockLevel, "银行") {
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
	var crops []gin.H
	for _, crop := range farmCrops {
		marketPrice := applyMarket(crop.UnitPrice, "crop_"+crop.Key)
		seasonPrice := applySeasonPrice(marketPrice, &crop)
		crops = append(crops, gin.H{
			"key":          crop.Key,
			"name":         crop.Name,
			"emoji":        crop.Emoji,
			"season":       crop.Season,
			"season_name":  seasonNames[crop.Season],
			"in_season":    isCropInSeason(&crop),
			"unit_price":   webFarmQuotaFloat(crop.UnitPrice),
			"market_price": webFarmQuotaFloat(marketPrice),
			"season_price": webFarmQuotaFloat(seasonPrice),
			"season_pct":   getSeasonPriceMultiplier(&crop),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"season":       season,
			"season_name":  seasonNames[season],
			"season_emoji": seasonEmojis[season],
			"days_left":    daysLeft,
			"season_days":  common.TgBotFarmSeasonDays,
			"in_bonus":     common.TgBotFarmSeasonInBonus,
			"off_bonus":    common.TgBotFarmSeasonOffBonus,
			"crops":        crops,
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockFish, "钓鱼") {
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
	if !checkFeatureLevel(c, tgId, common.TgBotFarmUnlockWorkshop, "加工坊") {
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
