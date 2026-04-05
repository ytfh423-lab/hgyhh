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

/* ═══════════════════════════════════════════════════════════════
   好友农场访问系统
   规则：
     - 必须是互相好友
     - 浇水/治疗：免费，访客无消耗
     - 施肥：消耗访客自己的化肥
     - 种植：消耗访客自己的种子/余额
     - 收获：所有收益归农场主，任务进度计入农场主
     - 访客行为记录在农场主日志中（注明来自谁的帮助）
   ═══════════════════════════════════════════════════════════════ */

// getVisitParams 解析访问参数，返回（访客user, 访客tgId, 农场主tgId, 农场主friendUserId, ok）
func getVisitParams(c *gin.Context) (visitor *model.User, visitorTgId string, ownerTgId string, ownerUserId int, ok bool) {
	visitor, visitorTgId, ok = getWebFarmUser(c)
	if !ok {
		return
	}
	friendIdStr := c.Param("friend_id")
	friendId, err := strconv.Atoi(friendIdStr)
	if err != nil || friendId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		ok = false
		return
	}
	if !model.IsFriend(visitor.Id, friendId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "只能访问好友的农场"})
		ok = false
		return
	}
	ownerUser, err2 := model.GetUserById(friendId, false)
	if err2 != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "好友不存在"})
		ok = false
		return
	}
	ownerUserId = ownerUser.Id
	ownerTgId = ownerUser.TelegramId
	if ownerTgId == "" {
		ownerTgId = fmt.Sprintf("u_%d", ownerUser.Id)
	}
	ok = true
	return
}

/* ── GET /api/farm/visit/:friend_id ── */
func WebFarmVisitView(c *gin.Context) {
	_, _, ownerTgId, _, ok := getVisitParams(c)
	if !ok {
		return
	}

	plots, err := model.GetOrCreateFarmPlots(ownerTgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	now := time.Now().Unix()
	plotInfos := make([]webPlotInfo, 0, len(plots))
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
		plotInfos = append(plotInfos, buildPlotInfo(plot))
	}

	// 农场主等级
	items, _ := model.GetFarmItems(ownerTgId)
	ownerLevel := 1
	for _, item := range items {
		if item.ItemType == "_level" && item.Quantity > 0 {
			ownerLevel = item.Quantity
		}
	}

	weather := GetCurrentWeather()
	weatherData := gin.H{
		"type":     weather.Type,
		"type_key": weather.TypeKey,
		"name":     weather.Name,
		"emoji":    weather.Emoji,
		"effects":  weather.Effects,
		"ends_in":  weather.EndsAt - now,
		"season":   getCurrentSeason(),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"plots":       plotInfos,
			"plot_count":  len(plots),
			"user_level":  ownerLevel,
			"weather":     weatherData,
			"max_plots":   model.FarmMaxPlots,
			// 隐藏农场主余额、背包、任务等私有信息
		},
	})
}

/* ── POST /api/farm/visit/:friend_id/harvest ── */
func WebFarmVisitHarvest(c *gin.Context) {
	visitor, visitorTgId, ownerTgId, ownerUserId, ok := getVisitParams(c)
	if !ok {
		return
	}

	ownerUser, err := model.GetUserById(ownerUserId, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	plots, err := model.GetOrCreateFarmPlots(ownerTgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	totalQuota := 0
	harvestedCount := 0
	for _, plot := range plots {
		if plot.Status != 2 {
			continue
		}
		crop := farmCropMap[plot.CropType]
		if crop == nil {
			continue
		}
		rawYield := 1 + rand.Intn(crop.MaxYield)
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
		realYield, _ := calcHarvestYield(baseYield, fertBonus, plot.StolenCount)
		marketPrice := applyMarket(crop.UnitPrice, "crop_"+crop.Key)
		seasonPrice := applySeasonPrice(marketPrice, crop)
		value := realYield * seasonPrice
		totalQuota += value
		harvestedCount++
		_ = model.ClearFarmPlot(plot.Id)
		model.RecordCollection(ownerTgId, "crop", crop.Key, realYield)
	}

	if harvestedCount == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有可收获的作物"})
		return
	}

	// 收益全部归农场主
	sellResult := farmSellToAdmin(ownerUser, ownerTgId, "crop", "visit_harvest", harvestedCount, totalQuota,
		fmt.Sprintf("好友%s帮助收获%d种", nameOf(visitor), harvestedCount))
	// 额外记录 harvest action，让农场主的任务进度正常推进
	model.AddFarmLog(ownerTgId, "harvest", sellResult.FinalValue,
		fmt.Sprintf("好友[%s]帮助收获%d块地", nameOf(visitor), harvestedCount))
	// 访客也记录一笔
	model.AddFarmLog(visitorTgId, "visit_help", 0,
		fmt.Sprintf("帮助好友收获%d块地", harvestedCount))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("帮助收获 %d 块地，农场主获得 $%.2f ✨",
			harvestedCount, webFarmQuotaFloat(sellResult.FinalValue)),
	})
}

/* ── POST /api/farm/visit/:friend_id/water ── 一键浇水（免费） */
func WebFarmVisitWaterAll(c *gin.Context) {
	visitor, visitorTgId, ownerTgId, _, ok := getVisitParams(c)
	if !ok {
		return
	}

	plots, err := model.GetOrCreateFarmPlots(ownerTgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	now := time.Now().Unix()
	watered := 0
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
		canWater := plot.Status == 1 || plot.Status == 4 ||
			(plot.Status == 3 && plot.EventType == "drought")
		if !canWater {
			continue
		}
		if plot.Status == 4 {
			waterInterval := int64(common.TgBotFarmWaterInterval)
			wiltStart := plot.LastWateredAt + waterInterval
			downtime := now - wiltStart
			plot.PlantedAt += downtime
			plot.Status = 1
			_ = model.UpdateFarmPlot(plot)
		} else if plot.Status == 3 && plot.EventType == "drought" {
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

	model.AddFarmLog(ownerTgId, "water", 0,
		fmt.Sprintf("好友[%s]帮助浇水%d块地", nameOf(visitor), watered))
	model.AddFarmLog(visitorTgId, "visit_help", 0,
		fmt.Sprintf("帮助好友浇水%d块地", watered))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("帮助浇水 %d 块地 💧", watered),
	})
}

/* ── POST /api/farm/visit/:friend_id/fertilize ── 施肥（消耗访客化肥） */
func WebFarmVisitFertilizeAll(c *gin.Context) {
	visitor, visitorTgId, ownerTgId, _, ok := getVisitParams(c)
	if !ok {
		return
	}

	plots, err := model.GetOrCreateFarmPlots(ownerTgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	fertilized := 0
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
		if plot.Status != 1 || plot.Fertilized == 1 {
			continue
		}
		// 消耗访客自己的化肥
		if err := model.DecrementFarmItem(visitorTgId, "fertilizer"); err != nil {
			break // 化肥用完了
		}
		plot.Fertilized = 1
		_ = model.UpdateFarmPlot(plot)
		fertilized++
	}

	if fertilized == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有可施肥的地块，或你的化肥不足"})
		return
	}

	model.AddFarmLog(ownerTgId, "visit_fertilize", 0,
		fmt.Sprintf("好友[%s]帮助施肥%d块地", nameOf(visitor), fertilized))
	model.AddFarmLog(visitorTgId, "visit_help", 0,
		fmt.Sprintf("帮助好友施肥%d块地，消耗化肥×%d", fertilized, fertilized))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("帮助施肥 %d 块地（消耗你的化肥×%d）🧴", fertilized, fertilized),
	})
}

/* ── POST /api/farm/visit/:friend_id/plant ── 种植（使用访客种子/余额） */
func WebFarmVisitPlant(c *gin.Context) {
	visitor, visitorTgId, ownerTgId, _, ok := getVisitParams(c)
	if !ok {
		return
	}

	var req struct {
		CropKey string `json:"crop_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.CropKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请选择要种植的作物"})
		return
	}

	crop := farmCropMap[req.CropKey]
	if crop == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知作物"})
		return
	}

	plots, err := model.GetOrCreateFarmPlots(ownerTgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	now := time.Now().Unix()
	planted := 0
	seedKey := "seed_" + crop.Key

	for _, plot := range plots {
		if plot.Status != 0 {
			continue
		}
		// 优先消耗访客库存种子，没有则从余额扣费
		usedInventory := false
		if errSeed := model.DecrementFarmItem(visitorTgId, seedKey); errSeed == nil {
			usedInventory = true
		} else {
			if visitor.Quota < crop.SeedCost {
				break // 余额不足，停止
			}
			if errQ := model.DecreaseUserQuota(visitor.Id, crop.SeedCost); errQ != nil {
				break
			}
			// 刷新访客余额
			fresh, _ := model.GetUserById(visitor.Id, false)
			if fresh != nil {
				visitor.Quota = fresh.Quota
			}
		}
		_ = usedInventory

		growSecs := int64(crop.GrowSecs)
		soilBonus := float64(plot.SoilLevel-1) * float64(common.TgBotFarmSoilSpeedBonus) / 100.0
		if soilBonus > 0 {
			growSecs = int64(float64(growSecs) * (1 - soilBonus))
		}
		seasonMult := getSeasonGrowthMultiplier(crop, now)
		if seasonMult != 100 {
			growSecs = growSecs * 100 / int64(seasonMult)
		}

		plot.CropType = crop.Key
		plot.PlantedAt = now - (int64(crop.GrowSecs) - growSecs)
		plot.Status = 1
		plot.EventType = ""
		plot.EventAt = 0
		plot.StolenCount = 0
		plot.Fertilized = 0
		plot.LastWateredAt = 0
		_ = model.UpdateFarmPlot(plot)
		planted++
	}

	if planted == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有空地块，或你的种子/余额不足"})
		return
	}

	model.AddFarmLog(ownerTgId, "plant", 0,
		fmt.Sprintf("好友[%s]帮助种植%s×%d", nameOf(visitor), crop.Name, planted))
	model.AddFarmLog(visitorTgId, "visit_help", 0,
		fmt.Sprintf("帮助好友种植%s%s×%d", crop.Emoji, crop.Name, planted))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("帮助种植 %s%s ×%d 块 🌱", crop.Emoji, crop.Name, planted),
	})
}

/* ── POST /api/farm/visit/:friend_id/treat ── 治疗（消耗访客药品） */
func WebFarmVisitTreatAll(c *gin.Context) {
	visitor, visitorTgId, ownerTgId, _, ok := getVisitParams(c)
	if !ok {
		return
	}

	plots, err := model.GetOrCreateFarmPlots(ownerTgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	treated := 0
	noMedicine := false
	now := time.Now().Unix()

	for _, plot := range plots {
		updateFarmPlotStatus(plot)
		if plot.Status != 3 {
			continue
		}
		if strings.EqualFold(plot.EventType, "drought") {
			continue // 干旱用浇水处理
		}
		var cureItem *farmItemDef
		for i := range farmItems {
			if farmItems[i].Cures == plot.EventType {
				cureItem = &farmItems[i]
				break
			}
		}
		if cureItem == nil {
			continue
		}
		if err := model.DecrementFarmItem(visitorTgId, cureItem.Key); err != nil {
			noMedicine = true
			break
		}
		downtime := now - plot.EventAt
		plot.PlantedAt += downtime
		plot.Status = 1
		plot.EventType = ""
		plot.EventAt = 0
		_ = model.UpdateFarmPlot(plot)
		treated++
	}

	if treated == 0 {
		msg := "没有需要治疗的病害地块"
		if noMedicine {
			msg = "你的药品不足，无法治疗"
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}

	model.AddFarmLog(ownerTgId, "visit_treat", 0,
		fmt.Sprintf("好友[%s]帮助治疗%d块地", nameOf(visitor), treated))
	model.AddFarmLog(visitorTgId, "visit_help", 0,
		fmt.Sprintf("帮助好友治疗%d块地", treated))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("帮助治疗 %d 块地 💊", treated),
	})
}

/* ── GET /api/farm/visit/:friend_id/inventory ── 访客自己的道具（种子/化肥/药品） */
func WebFarmVisitMyInventory(c *gin.Context) {
	_, visitorTgId, _, _, ok := getVisitParams(c)
	if !ok {
		return
	}

	items, _ := model.GetFarmItems(visitorTgId)
	var seeds []map[string]interface{}
	hasFertilizer := false
	hasMedicine := false

	for _, item := range items {
		if strings.HasPrefix(item.ItemType, "seed_") {
			cropKey := strings.TrimPrefix(item.ItemType, "seed_")
			crop := farmCropMap[cropKey]
			if crop != nil && item.Quantity > 0 {
				seeds = append(seeds, map[string]interface{}{
					"key":      cropKey,
					"name":     crop.Name,
					"emoji":    crop.Emoji,
					"quantity": item.Quantity,
				})
			}
		}
		if item.ItemType == "fertilizer" && item.Quantity > 0 {
			hasFertilizer = true
		}
		for _, fi := range farmItems {
			if fi.Key == item.ItemType && fi.Cures != "" && item.Quantity > 0 {
				hasMedicine = true
			}
		}
	}

	if seeds == nil {
		seeds = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"seeds":          seeds,
			"has_fertilizer": hasFertilizer,
			"has_medicine":   hasMedicine,
		},
	})
}
