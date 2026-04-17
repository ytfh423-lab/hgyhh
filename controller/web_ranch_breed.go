package controller

import (
	crand "crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ranchQualityMeta struct {
	Label        string
	TagColor     string
	PricePercent int
}

type webRanchBreedingInfo struct {
	Id                    int     `json:"id"`
	AnimalType            string  `json:"animal_type"`
	AnimalName            string  `json:"animal_name"`
	AnimalEmoji           string  `json:"animal_emoji"`
	Status                int     `json:"status"`
	StatusLabel           string  `json:"status_label"`
	StartedAt             int64   `json:"started_at"`
	DueAt                 int64   `json:"due_at"`
	Remaining             int64   `json:"remaining"`
	ParentAId             int     `json:"parent_a_id"`
	ParentBId             int     `json:"parent_b_id"`
	ParentAQuality        int     `json:"parent_a_quality"`
	ParentBQuality        int     `json:"parent_b_quality"`
	ParentAQualityLabel   string  `json:"parent_a_quality_label"`
	ParentBQualityLabel   string  `json:"parent_b_quality_label"`
	OffspringQuality      int     `json:"offspring_quality"`
	OffspringQualityLabel string  `json:"offspring_quality_label"`
	OffspringQualityColor string  `json:"offspring_quality_color"`
	OffspringGeneration   int     `json:"offspring_generation"`
	Cost                  float64 `json:"cost"`
}

var ranchQualityMetas = map[int]ranchQualityMeta{
	1: {Label: "普通", TagColor: "grey", PricePercent: 100},
	2: {Label: "优良", TagColor: "green", PricePercent: 150},
	3: {Label: "精英", TagColor: "blue", PricePercent: 250},
	4: {Label: "史诗", TagColor: "purple", PricePercent: 400},
	5: {Label: "传说", TagColor: "orange", PricePercent: 700},
}

func normalizeRanchQuality(quality int) int {
	if quality < 1 {
		return 1
	}
	if quality > 5 {
		return 5
	}
	return quality
}

func getRanchQualityMeta(quality int) ranchQualityMeta {
	quality = normalizeRanchQuality(quality)
	meta, ok := ranchQualityMetas[quality]
	if !ok {
		return ranchQualityMetas[1]
	}
	return meta
}

func getRanchQualityLabel(quality int) string {
	return getRanchQualityMeta(quality).Label
}

func getRanchQualityTagColor(quality int) string {
	return getRanchQualityMeta(quality).TagColor
}

func getRanchQualityPrice(basePrice int, quality int) int {
	percent := getRanchQualityMeta(quality).PricePercent
	return basePrice * percent / 100
}

func getRanchBreedPrice(def *ranchAnimalDef) int {
	if def == nil || def.BuyPrice == nil {
		return 0
	}
	return *def.BuyPrice * common.TgBotRanchBreedCostRate / 100
}

func getRanchBreedDueSeconds(def *ranchAnimalDef) int64 {
	if def == nil || def.GrowSecs == nil {
		return 1
	}
	due := *def.GrowSecs * int64(common.TgBotRanchBreedDueRate) / 100
	if due < 1 {
		return 1
	}
	return due
}

func getRanchBreedCooldownSeconds(def *ranchAnimalDef) int64 {
	if def == nil || def.GrowSecs == nil {
		return 1
	}
	cooldown := *def.GrowSecs * int64(common.TgBotRanchBreedCooldownRate) / 100
	if cooldown < 1 {
		return 1
	}
	return cooldown
}

func getRanchOffspringGeneration(left *model.TgRanchAnimal, right *model.TgRanchAnimal) int {
	gen := left.Generation
	if right.Generation > gen {
		gen = right.Generation
	}
	return gen + 1
}

func ranchRandomInt(max int) int {
	if max <= 1 {
		return 0
	}
	n, err := crand.Int(crand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return int(time.Now().UnixNano() % int64(max))
	}
	return int(n.Int64())
}

func rollRanchOffspringQuality(left int, right int) int {
	qa := normalizeRanchQuality(left)
	qb := normalizeRanchQuality(right)
	base := qa
	if qb > base {
		base = qb
	}
	upChance := 10 + (qa+qb)*4
	if qa == qb {
		upChance += 8
	}
	downChance := 14 - (qa + qb)
	if downChance < 2 {
		downChance = 2
	}
	roll := ranchRandomInt(100)
	if base < 5 && roll < upChance {
		base++
	} else if base > 1 && roll >= 100-downChance {
		base--
	}
	return normalizeRanchQuality(base)
}

func refreshRanchBreedingState(breeding *model.TgRanchBreeding) {
	if breeding == nil || breeding.Status != 1 {
		return
	}
	now := time.Now().Unix()
	if now < breeding.DueAt {
		return
	}
	breeding.Status = 2
	breeding.CompletedAt = now
	_ = model.UpdateRanchBreeding(breeding)
}

func buildWebRanchBreedingInfo(breeding *model.TgRanchBreeding) webRanchBreedingInfo {
	def := ranchAnimalMap[breeding.AnimalType]
	info := webRanchBreedingInfo{
		Id:                  breeding.Id,
		AnimalType:          breeding.AnimalType,
		Status:              breeding.Status,
		StartedAt:           breeding.StartedAt,
		DueAt:               breeding.DueAt,
		ParentAId:           breeding.ParentAId,
		ParentBId:           breeding.ParentBId,
		ParentAQuality:      breeding.ParentAQuality,
		ParentBQuality:      breeding.ParentBQuality,
		ParentAQualityLabel: getRanchQualityLabel(breeding.ParentAQuality),
		ParentBQualityLabel: getRanchQualityLabel(breeding.ParentBQuality),
		Cost:                webFarmQuotaFloat(breeding.Cost),
	}
	if def != nil {
		info.AnimalName = def.Name
		info.AnimalEmoji = def.Emoji
	}
	now := time.Now().Unix()
	if breeding.DueAt > now {
		info.Remaining = breeding.DueAt - now
	}
	switch breeding.Status {
	case 1:
		info.StatusLabel = "育种中"
	case 2:
		info.StatusLabel = "可领取"
		info.OffspringQuality = breeding.OffspringQuality
		info.OffspringQualityLabel = getRanchQualityLabel(breeding.OffspringQuality)
		info.OffspringQualityColor = getRanchQualityTagColor(breeding.OffspringQuality)
		info.OffspringGeneration = breeding.OffspringGeneration
	case 3:
		info.StatusLabel = "已领取"
		info.OffspringQuality = breeding.OffspringQuality
		info.OffspringQualityLabel = getRanchQualityLabel(breeding.OffspringQuality)
		info.OffspringQualityColor = getRanchQualityTagColor(breeding.OffspringQuality)
		info.OffspringGeneration = breeding.OffspringGeneration
	default:
		info.StatusLabel = "未知"
	}
	return info
}

func WebRanchBreedView(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockBreeding, "育种") {
		return
	}
	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取牧场数据失败"})
		return
	}
	breedings, err := model.GetRanchBreedings(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取育种数据失败"})
		return
	}
	aliveCount := 0
	animalInfos := make([]webAnimalInfo, 0, len(animals))
	for _, animal := range animals {
		updateRanchAnimalStatus(animal)
		if animal.Status != 5 {
			aliveCount++
		}
		animalInfos = append(animalInfos, buildAnimalInfo(animal))
	}
	breedingInfos := make([]webRanchBreedingInfo, 0, len(breedings))
	activeCount := 0
	for _, breeding := range breedings {
		refreshRanchBreedingState(breeding)
		if breeding.Status == 1 || breeding.Status == 2 {
			activeCount++
		}
		breedingInfos = append(breedingInfos, buildWebRanchBreedingInfo(breeding))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"animals":      animalInfos,
			"breedings":    breedingInfos,
			"alive_count":  aliveCount,
			"max_animals":  common.TgBotRanchMaxAnimals,
			"active_count": activeCount,
			"max_active":   common.TgBotRanchBreedMaxActive,
			"balance":      webFarmQuotaFloat(user.Quota),
		},
	})
}

func WebRanchBreedStart(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockBreeding, "育种") {
		return
	}
	var req struct {
		ParentAId int `json:"parent_a_id"`
		ParentBId int `json:"parent_b_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ParentAId <= 0 || req.ParentBId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.ParentAId == req.ParentBId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请选择两只不同的动物"})
		return
	}
	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取牧场数据失败"})
		return
	}
	aliveCount := 0
	animalMap := make(map[int]*model.TgRanchAnimal, len(animals))
	for _, animal := range animals {
		updateRanchAnimalStatus(animal)
		animalMap[animal.Id] = animal
		if animal.Status != 5 {
			aliveCount++
		}
	}
	left := animalMap[req.ParentAId]
	right := animalMap[req.ParentBId]
	if left == nil || right == nil || left.Status == 5 || right.Status == 5 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "育种对象不存在或已死亡"})
		return
	}
	if left.AnimalType != right.AnimalType {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "只能同种动物育种"})
		return
	}
	if left.Status != 2 || right.Status != 2 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "只有成熟动物才能育种"})
		return
	}
	now := time.Now().Unix()
	if left.BreedCooldownAt > now || right.BreedCooldownAt > now {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "有动物仍在育种冷却中"})
		return
	}
	if aliveCount >= common.TgBotRanchMaxAnimals {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "牧场没有空余栏位，请先释放一个栏位再育种"})
		return
	}
	activeCount, err := model.CountUnclaimedRanchBreedings(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取育种状态失败"})
		return
	}
	if activeCount >= int64(common.TgBotRanchBreedMaxActive) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("最多同时进行 %d 条育种", common.TgBotRanchBreedMaxActive)})
		return
	}
	def := ranchAnimalMap[left.AnimalType]
	if def == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知动物类型"})
		return
	}
	price := getRanchBreedPrice(def)
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "余额不足"})
		return
	}
	if err = model.DecreaseUserQuota(user.Id, price); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}
	breeding := &model.TgRanchBreeding{
		TelegramId:          tgId,
		ParentAId:           left.Id,
		ParentBId:           right.Id,
		AnimalType:          left.AnimalType,
		Status:              1,
		StartedAt:           now,
		DueAt:               now + getRanchBreedDueSeconds(def),
		ParentAQuality:      normalizeRanchQuality(left.Quality),
		ParentBQuality:      normalizeRanchQuality(right.Quality),
		OffspringQuality:    rollRanchOffspringQuality(left.Quality, right.Quality),
		OffspringGeneration: getRanchOffspringGeneration(left, right),
		Cost:                price,
	}
	left.Status = 6
	right.Status = 6
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(left).Error; err != nil {
			return err
		}
		if err := tx.Save(right).Error; err != nil {
			return err
		}
		if err := tx.Create(breeding).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		left.Status = 2
		right.Status = 2
		_ = model.IncreaseUserQuota(user.Id, price, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "发起育种失败，已退款"})
		return
	}
	model.AddFarmLog(tgId, "ranch_breed_start", -price, fmt.Sprintf("发起%s%s育种", def.Emoji, def.Name))
	respondFarmSuccessWithMedal(c, tgId, "ranch_breed", fmt.Sprintf("已开始 %s%s 育种，预计 %s 后可领取后代", def.Emoji, def.Name, formatDuration(getRanchBreedDueSeconds(def))), nil)
}

func WebRanchBreedClaim(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockBreeding, "育种") {
		return
	}
	var req struct {
		BreedingId int `json:"breeding_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.BreedingId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	breeding, err := model.GetRanchBreedingByIdAndTelegramId(req.BreedingId, tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "育种记录不存在"})
		return
	}
	refreshRanchBreedingState(breeding)
	if breeding.Status != 2 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该育种尚未完成"})
		return
	}
	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取牧场数据失败"})
		return
	}
	aliveCount := 0
	animalMap := make(map[int]*model.TgRanchAnimal, len(animals))
	for _, animal := range animals {
		updateRanchAnimalStatus(animal)
		animalMap[animal.Id] = animal
		if animal.Status != 5 {
			aliveCount++
		}
	}
	if aliveCount >= common.TgBotRanchMaxAnimals {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "牧场已满，请先腾出一个栏位再领取后代"})
		return
	}
	left := animalMap[breeding.ParentAId]
	right := animalMap[breeding.ParentBId]
	if left == nil || right == nil || left.Status == 5 || right.Status == 5 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "亲本数据异常，无法领取后代"})
		return
	}
	def := ranchAnimalMap[breeding.AnimalType]
	if def == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知动物类型"})
		return
	}
	now := time.Now().Unix()
	child := &model.TgRanchAnimal{
		TelegramId:       tgId,
		AnimalType:       breeding.AnimalType,
		Status:           1,
		PurchasedAt:      now,
		LastFedAt:        now,
		LastWateredAt:    now,
		LastCleanedAt:    now,
		Quality:          normalizeRanchQuality(breeding.OffspringQuality),
		Generation:       breeding.OffspringGeneration,
		ParentAId:        breeding.ParentAId,
		ParentBId:        breeding.ParentBId,
		BreedCooldownAt:  0,
	}
	cooldownAt := now + getRanchBreedCooldownSeconds(def)
	left.Status = 2
	left.LastFedAt = now
	left.LastWateredAt = now
	left.LastCleanedAt = now
	left.BreedCooldownAt = cooldownAt
	right.Status = 2
	right.LastFedAt = now
	right.LastWateredAt = now
	right.LastCleanedAt = now
	right.BreedCooldownAt = cooldownAt
	breeding.Status = 3
	breeding.ClaimedAt = now
	if breeding.CompletedAt == 0 {
		breeding.CompletedAt = now
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(left).Error; err != nil {
			return err
		}
		if err := tx.Save(right).Error; err != nil {
			return err
		}
		if err := tx.Create(child).Error; err != nil {
			return err
		}
		if err := tx.Save(breeding).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "领取后代失败"})
		return
	}
	model.AddFarmLog(tgId, "ranch_breed_claim", 0, fmt.Sprintf("领取%s%s后代[%s]", def.Emoji, def.Name, getRanchQualityLabel(child.Quality)))
	respondFarmSuccessWithMedal(c, tgId, "ranch_breed", fmt.Sprintf("成功领取 %s%s 后代！品质：%s", def.Emoji, def.Name, getRanchQualityLabel(child.Quality)), gin.H{
		"offspring": buildAnimalInfo(child),
	})
}
