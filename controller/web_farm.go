package controller

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
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
			value := realYield * crop.UnitPrice
			totalQuota += value
			harvestedCount++

			details = append(details, map[string]interface{}{
				"crop_name":  crop.Name,
				"crop_emoji": crop.Emoji,
				"yield":      yield - fertBonus,
				"fert_bonus": fertBonus,
				"stolen":     loss,
				"real_yield": realYield,
				"value":      webFarmQuotaFloat(value),
			})

			_ = model.ClearFarmPlot(plot.Id)
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

// WebFarmBuyItem buys a shop item
func WebFarmBuyItem(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		ItemKey string `json:"item_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	item := farmItemMap[req.ItemKey]
	if item == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知道具"})
		return
	}
	cost := item.Cost
	if req.ItemKey == "dogfood" {
		cost = common.TgBotFarmDogFoodPrice
	}
	if user.Quota < cost {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！需要 $%.2f", webFarmQuotaFloat(cost))})
		return
	}
	err := model.DecreaseUserQuota(user.Id, cost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}
	err = model.IncrementFarmItem(tgId, req.ItemKey, 1)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, cost, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "购买失败"})
		return
	}
	model.AddFarmLog(tgId, "shop", -cost, fmt.Sprintf("购买%s%s", item.Emoji, item.Name))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("购买 %s%s 成功！", item.Emoji, item.Name)})
}

// WebFarmStealTargets returns available steal targets
func WebFarmStealTargets(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
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

// WebFarmDog returns dog info
func WebFarmDog(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
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
