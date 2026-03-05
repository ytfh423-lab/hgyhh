package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ========== Web Ranch API ==========

type webAnimalInfo struct {
	Id            int     `json:"id"`
	AnimalType    string  `json:"animal_type"`
	AnimalName    string  `json:"animal_name"`
	AnimalEmoji   string  `json:"animal_emoji"`
	Status        int     `json:"status"`
	StatusLabel   string  `json:"status_label"`
	PurchasedAt   int64   `json:"purchased_at"`
	GrowSecs      int64   `json:"grow_secs"`
	Progress      int     `json:"progress"`
	Remaining     int64   `json:"remaining"`
	MeatPrice     float64 `json:"meat_price"`
	LastFedAt     int64   `json:"last_fed_at"`
	LastWateredAt int64   `json:"last_watered_at"`
	FeedRemaining int64   `json:"feed_remaining"`
	WaterRemaining int64  `json:"water_remaining"`
	NeedsFeed      bool    `json:"needs_feed"`
	NeedsWater     bool    `json:"needs_water"`
	IsDirty        bool    `json:"is_dirty"`
	CleanRemaining int64   `json:"clean_remaining"`
}

func buildAnimalInfo(animal *model.TgRanchAnimal) webAnimalInfo {
	def := ranchAnimalMap[animal.AnimalType]
	info := webAnimalInfo{
		Id:            animal.Id,
		AnimalType:    animal.AnimalType,
		Status:        animal.Status,
		PurchasedAt:   animal.PurchasedAt,
		LastFedAt:     animal.LastFedAt,
		LastWateredAt: animal.LastWateredAt,
	}

	if def != nil {
		info.AnimalName = def.Name
		info.AnimalEmoji = def.Emoji
		info.GrowSecs = *def.GrowSecs
		info.MeatPrice = webFarmQuotaFloat(*def.MeatPrice)
	}

	now := time.Now().Unix()

	switch animal.Status {
	case 1:
		elapsed := now - animal.PurchasedAt
		if def != nil {
			total := *def.GrowSecs
			pct := int(elapsed * 100 / total)
			if pct > 99 {
				pct = 99
			}
			remaining := total - elapsed
			if remaining < 0 {
				remaining = 0
			}
			info.Progress = pct
			info.Remaining = remaining
		}
		info.StatusLabel = "生长中"
	case 2:
		info.Progress = 100
		info.StatusLabel = "已成熟"
	case 3:
		info.StatusLabel = "饥饿"
	case 4:
		info.StatusLabel = "口渴"
	case 5:
		info.StatusLabel = "已死亡"
	}

	// 计算喂食/喂水剩余时间
	feedInterval := int64(common.TgBotRanchFeedInterval)
	nextFeed := animal.LastFedAt + feedInterval
	if now >= nextFeed {
		info.NeedsFeed = true
		info.FeedRemaining = 0
	} else {
		info.FeedRemaining = nextFeed - now
	}

	waterInterval := int64(common.TgBotRanchWaterInterval)
	nextWater := animal.LastWateredAt + waterInterval
	if now >= nextWater {
		info.NeedsWater = true
		info.WaterRemaining = 0
	} else {
		info.WaterRemaining = nextWater - now
	}

	// 粪便清理
	manureInterval := int64(common.TgBotRanchManureInterval)
	if animal.LastCleanedAt > 0 {
		nextClean := animal.LastCleanedAt + manureInterval
		if now >= nextClean {
			info.IsDirty = true
			info.CleanRemaining = 0
		} else {
			info.CleanRemaining = nextClean - now
		}
	}

	return info
}

// WebRanchView returns the ranch state
func WebRanchView(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取牧场数据失败"})
		return
	}

	for _, a := range animals {
		updateRanchAnimalStatus(a)
	}

	var animalInfos []webAnimalInfo
	for _, a := range animals {
		animalInfos = append(animalInfos, buildAnimalInfo(a))
	}

	// 可购买动物列表
	var animalDefs []map[string]interface{}
	for _, a := range ranchAnimals {
		animalDefs = append(animalDefs, map[string]interface{}{
			"key":        a.Key,
			"name":       a.Name,
			"emoji":      a.Emoji,
			"buy_price":  webFarmQuotaFloat(*a.BuyPrice),
			"grow_secs":  *a.GrowSecs,
			"meat_price": webFarmQuotaFloat(*a.MeatPrice),
		})
	}

	aliveCount := 0
	for _, a := range animals {
		if a.Status != 5 {
			aliveCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"animals":       animalInfos,
			"animal_types":  animalDefs,
			"alive_count":   aliveCount,
			"max_animals":   common.TgBotRanchMaxAnimals,
			"feed_price":         webFarmQuotaFloat(common.TgBotRanchFeedPrice),
			"water_price":        webFarmQuotaFloat(common.TgBotRanchWaterPrice),
			"manure_clean_price": webFarmQuotaFloat(common.TgBotRanchManureCleanPrice),
			"manure_penalty":     common.TgBotRanchManureGrowPenalty,
			"balance":            webFarmQuotaFloat(user.Quota),
		},
	})
}

// WebRanchBuy purchases an animal
func WebRanchBuy(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		AnimalType string `json:"animal_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	def := ranchAnimalMap[req.AnimalType]
	if def == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知动物类型"})
		return
	}

	// 检查存活数量
	animals, _ := model.GetRanchAnimals(tgId)
	aliveCount := 0
	for _, a := range animals {
		updateRanchAnimalStatus(a)
		if a.Status != 5 {
			aliveCount++
		}
	}
	if aliveCount >= common.TgBotRanchMaxAnimals {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("牧场已满，最多养 %d 只动物", common.TgBotRanchMaxAnimals)})
		return
	}

	price := *def.BuyPrice
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "余额不足"})
		return
	}

	err := model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

	now := time.Now().Unix()
	animal := &model.TgRanchAnimal{
		TelegramId:    tgId,
		AnimalType:    def.Key,
		Status:        1,
		PurchasedAt:   now,
		LastFedAt:     now,
		LastWateredAt: now,
		LastCleanedAt: now,
	}
	err = model.CreateRanchAnimal(animal)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建失败，已退款"})
		return
	}

	model.AddFarmLog(tgId, "ranch_buy", -price, fmt.Sprintf("购买%s%s", def.Emoji, def.Name))
	common.SysLog(fmt.Sprintf("Web Ranch: user %s bought %s for %d quota", tgId, def.Key, price))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("购买 %s%s 成功！", def.Emoji, def.Name)})
}

// WebRanchFeed feeds an animal
func WebRanchFeed(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		AnimalId int `json:"animal_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	animals, _ := model.GetRanchAnimals(tgId)
	var target *model.TgRanchAnimal
	for _, a := range animals {
		if a.Id == req.AnimalId {
			target = a
			break
		}
	}
	if target == nil || target.Status == 5 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该动物不存在或已死亡"})
		return
	}

	price := common.TgBotRanchFeedPrice
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "余额不足"})
		return
	}

	err := model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

	err = model.FeedRanchAnimal(target.Id)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "喂食失败，已退款"})
		return
	}

	// 恢复状态
	if target.Status == 3 {
		now := time.Now().Unix()
		waterInterval := int64(common.TgBotRanchWaterInterval)
		if now > target.LastWateredAt+waterInterval {
			target.Status = 4
		} else {
			def := ranchAnimalMap[target.AnimalType]
			if def != nil && now-target.PurchasedAt >= *def.GrowSecs {
				target.Status = 2
			} else {
				target.Status = 1
			}
		}
		_ = model.UpdateRanchAnimal(target)
	}

	def := ranchAnimalMap[target.AnimalType]
	detail := "喂食动物"
	if def != nil {
		detail = fmt.Sprintf("喂食%s%s", def.Emoji, def.Name)
	}
	model.AddFarmLog(tgId, "ranch_feed", -price, detail)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "喂食成功！"})
}

// WebRanchWater waters an animal
func WebRanchWater(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		AnimalId int `json:"animal_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	animals, _ := model.GetRanchAnimals(tgId)
	var target *model.TgRanchAnimal
	for _, a := range animals {
		if a.Id == req.AnimalId {
			target = a
			break
		}
	}
	if target == nil || target.Status == 5 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该动物不存在或已死亡"})
		return
	}

	price := common.TgBotRanchWaterPrice
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "余额不足"})
		return
	}

	err := model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

	err = model.WaterRanchAnimal(target.Id)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "喂水失败，已退款"})
		return
	}

	// 恢复状态
	if target.Status == 4 {
		now := time.Now().Unix()
		feedInterval := int64(common.TgBotRanchFeedInterval)
		if now > target.LastFedAt+feedInterval {
			target.Status = 3
		} else {
			def := ranchAnimalMap[target.AnimalType]
			if def != nil && now-target.PurchasedAt >= *def.GrowSecs {
				target.Status = 2
			} else {
				target.Status = 1
			}
		}
		_ = model.UpdateRanchAnimal(target)
	}

	def := ranchAnimalMap[target.AnimalType]
	detail := "喂水动物"
	if def != nil {
		detail = fmt.Sprintf("喂水%s%s", def.Emoji, def.Name)
	}
	model.AddFarmLog(tgId, "ranch_water", -price, detail)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "喂水成功！"})
}

// WebRanchSlaughter slaughters a mature animal and sells the meat
func WebRanchSlaughter(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		AnimalId int `json:"animal_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	animals, _ := model.GetRanchAnimals(tgId)
	var target *model.TgRanchAnimal
	for _, a := range animals {
		if a.Id == req.AnimalId {
			target = a
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该动物不存在"})
		return
	}

	updateRanchAnimalStatus(target)
	if target.Status != 2 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该动物尚未成熟，无法屠宰"})
		return
	}

	def := ranchAnimalMap[target.AnimalType]
	if def == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知动物类型"})
		return
	}

	meatPrice := *def.MeatPrice

	err := model.DeleteRanchAnimal(target.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "操作失败"})
		return
	}

	err = model.IncreaseUserQuota(user.Id, meatPrice, true)
	if err != nil {
		_ = model.CreateRanchAnimal(target)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "收入到账失败，已恢复动物"})
		return
	}

	model.AddFarmLog(tgId, "ranch_sell", meatPrice, fmt.Sprintf("出售%s%s", def.Emoji, def.Name))
	common.SysLog(fmt.Sprintf("Web Ranch: user %s slaughtered %s for %d quota", tgId, def.Key, meatPrice))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("出售 %s%s 成功！收入 $%.2f", def.Emoji, def.Name, webFarmQuotaFloat(meatPrice)),
	})
}

// WebRanchCleanManure cleans manure for all alive animals
func WebRanchCleanManure(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	now := time.Now().Unix()
	dirtyCount := 0
	for _, a := range animals {
		if a.Status != 5 && isAnimalDirty(a, now) {
			dirtyCount++
		}
	}

	if dirtyCount == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "牧场很干净，不需要清理"})
		return
	}

	price := common.TgBotRanchManureCleanPrice
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "余额不足"})
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

	err = model.CleanRanchAnimals(tgId)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "清理失败，已退款"})
		return
	}

	model.AddFarmLog(tgId, "ranch_clean", -price, fmt.Sprintf("清理粪便%d只", dirtyCount))
	common.SysLog(fmt.Sprintf("Web Ranch: user %s cleaned manure for %d animals, cost %d", tgId, dirtyCount, price))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已清理 %d 只动物的粪便，生长速度恢复正常！", dirtyCount),
	})
}

// WebRanchCleanup removes dead animals
func WebRanchCleanup(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}

	cleaned := 0
	for _, a := range animals {
		updateRanchAnimalStatus(a)
		if a.Status == 5 {
			_ = model.DeleteRanchAnimal(a.Id)
			cleaned++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已清理 %d 只死亡动物", cleaned),
	})
}
