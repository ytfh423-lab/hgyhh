package controller

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ========== 树种定义 ==========

type treeYieldItem struct {
	ItemKey   string `json:"item_key"`
	Name      string `json:"name"`
	Emoji     string `json:"emoji"`
	AmountMin int    `json:"amount_min"`
	AmountMax int    `json:"amount_max"`
	UnitPrice int    `json:"unit_price"` // quota per unit
}

type treeDef struct {
	Key             string          `json:"key"`
	Name            string          `json:"name"`
	Emoji           string          `json:"emoji"`
	SeedCost        int             `json:"seed_cost"`         // quota
	GrowSecs        int64           `json:"grow_secs"`         // seconds to mature
	Repeatable      bool            `json:"repeatable"`        // can harvest fruit repeatedly
	HarvestCooldown int64           `json:"harvest_cooldown"`  // seconds between harvests
	HarvestYield    []treeYieldItem `json:"harvest_yield"`     // fruit harvest yields
	CanChop         bool            `json:"can_chop"`          // can be chopped
	ChopYield       []treeYieldItem `json:"chop_yield"`        // chop yields
	StumpClearSecs  int64           `json:"stump_clear_secs"`  // stump cooldown override (0=use global)
	Description     string          `json:"description"`
}

var treeFarmTrees = []treeDef{
	{
		Key: "pine", Name: "松树", Emoji: "🌲",
		SeedCost: 500000, GrowSecs: 43200, // 12h
		Repeatable: false, CanChop: true,
		ChopYield: []treeYieldItem{
			{"lumber", "木材", "🪵", 3, 6, 300000},
			{"pinecone", "松果", "🌰", 1, 3, 50000},
		},
		Description: "经典针叶树，成熟后可伐木获得大量木材和松果",
	},
	{
		Key: "apple", Name: "苹果树", Emoji: "🍎",
		SeedCost: 800000, GrowSecs: 28800, // 8h
		Repeatable: true, HarvestCooldown: 14400, // 4h
		HarvestYield: []treeYieldItem{
			{"apple", "苹果", "🍎", 2, 5, 200000},
		},
		CanChop: true,
		ChopYield: []treeYieldItem{
			{"lumber", "木材", "🪵", 1, 3, 300000},
		},
		Description: "果树，成熟后可反复采摘苹果，也可伐木获得木材",
	},
	{
		Key: "oak", Name: "橡树", Emoji: "🌳",
		SeedCost: 1500000, GrowSecs: 86400, // 24h
		Repeatable: false, CanChop: true,
		ChopYield: []treeYieldItem{
			{"hardwood", "硬木", "🪓", 4, 8, 500000},
			{"acorn", "橡果", "🌰", 2, 5, 80000},
		},
		Description: "名贵阔叶树，生长缓慢但硬木价值极高",
	},
	{
		Key: "rubber", Name: "橡胶树", Emoji: "🪴",
		SeedCost: 600000, GrowSecs: 21600, // 6h
		Repeatable: true, HarvestCooldown: 10800, // 3h
		HarvestYield: []treeYieldItem{
			{"resin", "树脂", "🧴", 1, 4, 250000},
		},
		CanChop: true,
		ChopYield: []treeYieldItem{
			{"lumber", "木材", "🪵", 1, 2, 300000},
		},
		Description: "可反复采集树脂的经济作物，树脂用途广泛",
	},
	{
		Key: "cherry", Name: "樱桃树", Emoji: "🍒",
		SeedCost: 1000000, GrowSecs: 36000, // 10h
		Repeatable: true, HarvestCooldown: 18000, // 5h
		HarvestYield: []treeYieldItem{
			{"cherry", "樱桃", "🍒", 3, 8, 180000},
		},
		CanChop: true,
		ChopYield: []treeYieldItem{
			{"lumber", "木材", "🪵", 2, 4, 300000},
		},
		Description: "高级果树，樱桃产量高且价格稳定",
	},
	{
		Key: "bamboo", Name: "竹子", Emoji: "🎋",
		SeedCost: 200000, GrowSecs: 14400, // 4h
		Repeatable: true, HarvestCooldown: 7200, // 2h
		HarvestYield: []treeYieldItem{
			{"bamboo_shoot", "竹笋", "🎍", 2, 6, 100000},
		},
		CanChop: true,
		ChopYield: []treeYieldItem{
			{"bamboo_pole", "竹竿", "🎋", 2, 5, 150000},
		},
		Description: "生长最快的树种，竹笋可反复采集",
	},
}

var treeFarmTreeMap map[string]*treeDef

// ========== 树场产品定义（供市场/仓库/交易使用） ==========

type treeProductDef struct {
	Key       string
	Name      string
	Emoji     string
	BasePrice int // quota per unit
}

var treeProducts = []treeProductDef{
	{"lumber", "木材", "🪵", 300000},
	{"pinecone", "松果", "🌰", 50000},
	{"hardwood", "硬木", "🪓", 500000},
	{"acorn", "橡果", "🌰", 80000},
	{"resin", "树脂", "🧴", 250000},
	{"apple", "苹果", "🍎", 200000},
	{"cherry", "樱桃", "🍒", 180000},
	{"bamboo_shoot", "竹笋", "🎍", 100000},
	{"bamboo_pole", "竹竿", "🎋", 150000},
}

var treeProductMap map[string]*treeProductDef

func init() {
	treeFarmTreeMap = make(map[string]*treeDef)
	for i := range treeFarmTrees {
		treeFarmTreeMap[treeFarmTrees[i].Key] = &treeFarmTrees[i]
	}
	treeProductMap = make(map[string]*treeProductDef)
	for i := range treeProducts {
		treeProductMap[treeProducts[i].Key] = &treeProducts[i]
	}
}

// ========== helpers ===========

func getTreeGrowSecs(tree *treeDef, slot *model.TgTreeSlot) int64 {
	growSecs := tree.GrowSecs
	// 施肥加速
	if slot.Fertilized == 1 {
		bonus := int64(common.TgBotTreeFarmFertilizerBonus)
		growSecs = growSecs * (100 - bonus) / 100
	}
	// 浇水加速
	if slot.LastWateredAt > 0 {
		waterInterval := int64(common.TgBotTreeFarmWaterInterval)
		now := time.Now().Unix()
		if now-slot.LastWateredAt < waterInterval {
			bonus := int64(common.TgBotTreeFarmWaterBonus)
			growSecs = growSecs * (100 - bonus) / 100
		}
	}
	if growSecs < 60 {
		growSecs = 60
	}
	return growSecs
}

func getTreeStumpClearSecs(tree *treeDef) int64 {
	if tree.StumpClearSecs > 0 {
		return tree.StumpClearSecs
	}
	return int64(common.TgBotTreeFarmStumpClearSecs)
}

type yieldResult struct {
	ItemKey string
	Name    string
	Emoji   string
	Amount  int
}

func calcTreeYieldItems(yields []treeYieldItem) []yieldResult {
	var results []yieldResult
	for _, y := range yields {
		amount := y.AmountMin
		if y.AmountMax > y.AmountMin {
			amount = y.AmountMin + rand.Intn(y.AmountMax-y.AmountMin+1)
		}
		results = append(results, yieldResult{
			ItemKey: y.ItemKey,
			Name:    y.Name,
			Emoji:   y.Emoji,
			Amount:  amount,
		})
	}
	return results
}

// ========== 树场查看 ==========

type webTreeSlotInfo struct {
	SlotIndex       int    `json:"slot_index"`
	Status          int    `json:"status"`          // 0=empty 1=growing 2=mature 3=stump
	TreeType        string `json:"tree_type"`
	TreeName        string `json:"tree_name"`
	TreeEmoji       string `json:"tree_emoji"`
	PlantedAt       int64  `json:"planted_at"`
	GrowSecs        int64  `json:"grow_secs"`
	Progress        int    `json:"progress"`
	Remaining       int64  `json:"remaining"`
	ReadyAt         int64  `json:"ready_at"`
	CanHarvest      bool   `json:"can_harvest"`
	CanChop         bool   `json:"can_chop"`
	HarvestCooldown int64  `json:"harvest_cooldown"`  // seconds until next harvest
	HarvestCount    int    `json:"harvest_count"`
	Fertilized      int    `json:"fertilized"`
	LastWateredAt   int64  `json:"last_watered_at"`
	WaterRemain     int64  `json:"water_remain"`
	StumpRemain     int64  `json:"stump_remain"`      // seconds until stump can be cleared
	StatusLabel     string `json:"status_label"`
	Repeatable      bool   `json:"repeatable"`
}

func buildTreeSlotInfo(slot *model.TgTreeSlot) webTreeSlotInfo {
	now := time.Now().Unix()
	info := webTreeSlotInfo{
		SlotIndex:    slot.SlotIndex,
		Status:       slot.Status,
		TreeType:     slot.TreeType,
		Fertilized:   slot.Fertilized,
		LastWateredAt: slot.LastWateredAt,
		HarvestCount: slot.HarvestCount,
		PlantedAt:    slot.PlantedAt,
	}

	tree := treeFarmTreeMap[slot.TreeType]
	if tree != nil {
		info.TreeName = tree.Name
		info.TreeEmoji = tree.Emoji
		info.Repeatable = tree.Repeatable
	}

	switch slot.Status {
	case 0:
		info.StatusLabel = "空地"
	case 1:
		info.StatusLabel = "生长中"
		if tree != nil {
			growSecs := getTreeGrowSecs(tree, slot)
			info.GrowSecs = growSecs
			elapsed := now - slot.PlantedAt
			if elapsed >= growSecs {
				// 实际已成熟，更新状态
				slot.Status = 2
				info.Status = 2
				info.StatusLabel = "已成熟"
				info.Progress = 100
				info.Remaining = 0
				_ = model.UpdateTreeSlot(slot)
			} else {
				pct := int(elapsed * 100 / growSecs)
				if pct > 99 {
					pct = 99
				}
				info.Progress = pct
				info.Remaining = growSecs - elapsed
				info.ReadyAt = slot.PlantedAt + growSecs
			}
		}
	case 2:
		info.StatusLabel = "已成熟"
		info.Progress = 100
		if tree != nil {
			info.CanChop = tree.CanChop
			if tree.Repeatable {
				if slot.LastHarvestedAt == 0 {
					info.CanHarvest = true
				} else {
					nextHarvest := slot.LastHarvestedAt + tree.HarvestCooldown
					if now >= nextHarvest {
						info.CanHarvest = true
					} else {
						info.HarvestCooldown = nextHarvest - now
					}
				}
			}
		}
	case 3:
		info.StatusLabel = "树桩"
		if tree != nil {
			clearSecs := getTreeStumpClearSecs(tree)
			remain := slot.StumpAt + clearSecs - now
			if remain < 0 {
				remain = 0
			}
			info.StumpRemain = remain
		}
	}

	// 浇水状态
	if slot.LastWateredAt > 0 && slot.Status == 1 {
		waterInterval := int64(common.TgBotTreeFarmWaterInterval)
		remain := slot.LastWateredAt + waterInterval - now
		if remain > 0 {
			info.WaterRemain = remain
		}
	}

	return info
}

// WebTreeFarmView 获取树场信息
func WebTreeFarmView(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockTreeFarm, "树场") {
		return
	}

	slots, err := model.GetOrCreateTreeSlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "加载树场失败"})
		return
	}

	var slotInfos []webTreeSlotInfo
	for _, slot := range slots {
		slotInfos = append(slotInfos, buildTreeSlotInfo(slot))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"slots":          slotInfos,
			"slot_count":     len(slots),
			"max_slots":      model.TreeFarmMaxSlots,
			"slot_price":     webFarmQuotaFloat(common.TgBotTreeFarmSlotPrice),
			"balance":        webFarmQuotaFloat(user.Quota),
			"user_level":     model.GetFarmLevel(tgId),
			"unlock_level":   common.TgBotFarmUnlockTreeFarm,
			"water_interval": common.TgBotTreeFarmWaterInterval,
			"water_bonus":    common.TgBotTreeFarmWaterBonus,
			"fert_bonus":     common.TgBotTreeFarmFertilizerBonus,
		},
	})
}

// WebTreeFarmTypes 获取可种植的树种列表
func WebTreeFarmTypes(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockTreeFarm, "树场") {
		return
	}

	var types []map[string]interface{}
	for _, t := range treeFarmTrees {
		types = append(types, map[string]interface{}{
			"key":              t.Key,
			"name":             t.Name,
			"emoji":            t.Emoji,
			"seed_cost":        webFarmQuotaFloat(t.SeedCost),
			"grow_secs":        t.GrowSecs,
			"repeatable":       t.Repeatable,
			"harvest_cooldown": t.HarvestCooldown,
			"can_chop":         t.CanChop,
			"description":      t.Description,
			"harvest_yield":    t.HarvestYield,
			"chop_yield":       t.ChopYield,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": types})
}

// WebTreeFarmPlant 种植树苗
func WebTreeFarmPlant(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockTreeFarm, "树场") {
		return
	}

	var req struct {
		TreeKey   string `json:"tree_key"`
		SlotIndex int    `json:"slot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	tree := treeFarmTreeMap[req.TreeKey]
	if tree == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知树种"})
		return
	}

	if user.Quota < tree.SeedCost {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！树苗需要 $%.2f", webFarmQuotaFloat(tree.SeedCost))})
		return
	}

	slot, err := model.GetTreeSlot(tgId, req.SlotIndex)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "树位不存在"})
		return
	}
	if slot.Status != 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该树位不是空地"})
		return
	}

	err = model.DecreaseUserQuota(user.Id, tree.SeedCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

	now := time.Now().Unix()
	err = model.PlantTree(slot, tree.Key, now)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, tree.SeedCost, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "种植失败，已退款"})
		return
	}

	model.AddFarmLog(tgId, "tree_plant", -tree.SeedCost, fmt.Sprintf("种植%s%s", tree.Emoji, tree.Name))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功种植 %s%s！", tree.Emoji, tree.Name),
	})
}

// WebTreeFarmWater 浇水
func WebTreeFarmWater(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockTreeFarm, "树场") {
		return
	}

	var req struct {
		SlotIndex int `json:"slot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	slot, err := model.GetTreeSlot(tgId, req.SlotIndex)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "树位不存在"})
		return
	}
	if slot.Status != 1 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "只有生长中的树木才能浇水"})
		return
	}

	now := time.Now().Unix()
	waterInterval := int64(common.TgBotTreeFarmWaterInterval)
	if slot.LastWateredAt > 0 && now-slot.LastWateredAt < waterInterval {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "浇水还在生效中，无需重复浇水"})
		return
	}

	err = model.WaterTree(slot.Id, now)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "浇水失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("浇水成功！生长速度提升 %d%%", common.TgBotTreeFarmWaterBonus),
	})
}

// WebTreeFarmFertilize 施肥
func WebTreeFarmFertilize(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockTreeFarm, "树场") {
		return
	}

	var req struct {
		SlotIndex int `json:"slot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	slot, err := model.GetTreeSlot(tgId, req.SlotIndex)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "树位不存在"})
		return
	}
	if slot.Status != 1 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "只有生长中的树木才能施肥"})
		return
	}
	if slot.Fertilized == 1 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "已经施过肥了"})
		return
	}

	// 消耗化肥道具
	fertCount, _ := model.GetFarmItemQuantity(tgId, "fertilizer")
	if fertCount <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有化肥了，请先去商店购买"})
		return
	}
	_ = model.DecrementFarmItem(tgId, "fertilizer")

	err = model.FertilizeTree(slot.Id)
	if err != nil {
		_ = model.IncrementFarmItem(tgId, "fertilizer", 1)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "施肥失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("施肥成功！生长速度提升 %d%%", common.TgBotTreeFarmFertilizerBonus),
	})
}

// WebTreeFarmHarvest 采收果实（重复采集型树木），产物存入仓库
func WebTreeFarmHarvest(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockTreeFarm, "树场") {
		return
	}

	var req struct {
		SlotIndex int `json:"slot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	slot, err := model.GetTreeSlot(tgId, req.SlotIndex)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "树位不存在"})
		return
	}
	if slot.Status != 2 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "树木尚未成熟"})
		return
	}

	tree := treeFarmTreeMap[slot.TreeType]
	if tree == nil || !tree.Repeatable {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该树种不支持采收果实"})
		return
	}

	now := time.Now().Unix()
	if slot.LastHarvestedAt > 0 {
		nextHarvest := slot.LastHarvestedAt + tree.HarvestCooldown
		if now < nextHarvest {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "果实还没成熟，请稍后再来"})
			return
		}
	}

	// 检查仓库容量
	currentTotal := model.GetWarehouseTotalCount(tgId)
	whLevel := model.GetWarehouseLevel(tgId)
	whMax := model.GetWarehouseMaxSlots(whLevel)

	items := calcTreeYieldItems(tree.HarvestYield)
	totalItems := 0
	for _, item := range items {
		totalItems += item.Amount
	}
	if currentTotal+totalItems > whMax {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仓库空间不足！请先出售仓库物品"})
		return
	}

	// 存入仓库
	var details []map[string]interface{}
	for _, item := range items {
		_ = model.AddToWarehouseWithCategory(tgId, "wood_"+item.ItemKey, item.Amount, "wood")
		details = append(details, map[string]interface{}{
			"item":   item.Name,
			"emoji":  item.Emoji,
			"amount": item.Amount,
		})
	}
	_ = model.HarvestTree(slot.Id, now)

	msg := fmt.Sprintf("采收%s%s：", tree.Emoji, tree.Name)
	for _, item := range items {
		msg += fmt.Sprintf("%s%s×%d ", item.Emoji, item.Name, item.Amount)
	}
	model.AddFarmLog(tgId, "tree_harvest", 0, msg)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("采收成功！产物已存入仓库"),
		"data": gin.H{
			"details": details,
		},
	})
}

// WebTreeFarmHarvestAll 一键采收所有可采收的果树
func WebTreeFarmHarvestAll(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockTreeFarm, "树场") {
		return
	}

	slots, _ := model.GetOrCreateTreeSlots(tgId)
	now := time.Now().Unix()

	// 预取仓库容量
	currentTotal := model.GetWarehouseTotalCount(tgId)
	whLevel := model.GetWarehouseLevel(tgId)
	whMax := model.GetWarehouseMaxSlots(whLevel)

	collected := 0
	totalItems := 0
	var allDetails []map[string]interface{}

	for i := range slots {
		slot := &slots[i]

		// 自动成熟判定
		if slot.Status == 1 && now >= slot.PlantedAt+treeFarmTreeMap[slot.TreeType].GrowSecs {
			slot.Status = 2
		}
		if slot.Status != 2 {
			continue
		}

		tree := treeFarmTreeMap[slot.TreeType]
		if tree == nil || !tree.Repeatable {
			continue
		}

		// 检查采收冷却
		if slot.LastHarvestedAt > 0 && now < slot.LastHarvestedAt+tree.HarvestCooldown {
			continue
		}

		// 计算产出
		items := calcTreeYieldItems(tree.HarvestYield)
		itemCount := 0
		for _, item := range items {
			itemCount += item.Amount
		}

		// 仓库容量检查，放不下就跳过
		if currentTotal+itemCount > whMax {
			continue
		}

		// 存入仓库
		for _, item := range items {
			_ = model.AddToWarehouseWithCategory(tgId, "wood_"+item.ItemKey, item.Amount, "wood")
			allDetails = append(allDetails, map[string]interface{}{
				"tree":   tree.Name,
				"item":   item.Name,
				"emoji":  item.Emoji,
				"amount": item.Amount,
			})
		}
		_ = model.HarvestTree(slot.Id, now)
		currentTotal += itemCount
		totalItems += itemCount
		collected++

		msg := fmt.Sprintf("采收%s%s：", tree.Emoji, tree.Name)
		for _, item := range items {
			msg += fmt.Sprintf("%s%s×%d ", item.Emoji, item.Name, item.Amount)
		}
		model.AddFarmLog(tgId, "tree_harvest", 0, msg)
	}

	if collected == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "没有可采收的果树"})
		return
	}

	model.AddFarmLog(tgId, "tree_harvest_all", 0, fmt.Sprintf("一键采收%d棵果树，共%d件产物", collected, totalItems))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("一键采收 %d 棵果树，共 %d 件产物已存入仓库", collected, totalItems),
		"data": gin.H{
			"count":      collected,
			"totalItems": totalItems,
			"details":    allDetails,
		},
	})
}

// WebTreeFarmChop 伐木，产物存入仓库
func WebTreeFarmChop(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockTreeFarm, "树场") {
		return
	}

	var req struct {
		SlotIndex int `json:"slot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	slot, err := model.GetTreeSlot(tgId, req.SlotIndex)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "树位不存在"})
		return
	}
	if slot.Status != 2 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "树木尚未成熟，无法伐木"})
		return
	}

	tree := treeFarmTreeMap[slot.TreeType]
	if tree == nil || !tree.CanChop {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该树种不支持伐木"})
		return
	}

	// 检查仓库容量
	currentTotal := model.GetWarehouseTotalCount(tgId)
	whLevel := model.GetWarehouseLevel(tgId)
	whMax := model.GetWarehouseMaxSlots(whLevel)

	items := calcTreeYieldItems(tree.ChopYield)
	totalItems := 0
	for _, item := range items {
		totalItems += item.Amount
	}
	if currentTotal+totalItems > whMax {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仓库空间不足！请先出售仓库物品"})
		return
	}

	now := time.Now().Unix()

	// 存入仓库
	var details []map[string]interface{}
	for _, item := range items {
		_ = model.AddToWarehouseWithCategory(tgId, "wood_"+item.ItemKey, item.Amount, "wood")
		details = append(details, map[string]interface{}{
			"item":   item.Name,
			"emoji":  item.Emoji,
			"amount": item.Amount,
		})
	}
	_ = model.ChopTree(slot.Id, now)

	msg := fmt.Sprintf("伐木%s%s：", tree.Emoji, tree.Name)
	for _, item := range items {
		msg += fmt.Sprintf("%s%s×%d ", item.Emoji, item.Name, item.Amount)
	}
	model.AddFarmLog(tgId, "tree_chop", 0, msg)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "伐木成功！产物已存入仓库",
		"data": gin.H{
			"details": details,
		},
	})
}

// WebTreeFarmClearStump 清理树桩
func WebTreeFarmClearStump(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockTreeFarm, "树场") {
		return
	}

	var req struct {
		SlotIndex int `json:"slot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	slot, err := model.GetTreeSlot(tgId, req.SlotIndex)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "树位不存在"})
		return
	}
	if slot.Status != 3 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该树位没有树桩"})
		return
	}

	tree := treeFarmTreeMap[slot.TreeType]
	now := time.Now().Unix()
	clearSecs := int64(common.TgBotTreeFarmStumpClearSecs)
	if tree != nil && tree.StumpClearSecs > 0 {
		clearSecs = tree.StumpClearSecs
	}
	if now < slot.StumpAt+clearSecs {
		remain := slot.StumpAt + clearSecs - now
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("树桩还需要 %d 分钟才能清理", remain/60+1)})
		return
	}

	err = model.ClearTreeSlot(slot)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "清理失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "树桩已清理，可以重新种植",
	})
}

// WebTreeFarmBuySlot 购买新树位
func WebTreeFarmBuySlot(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	if !webCheckFeatureLevel(c, tgId, common.TgBotFarmUnlockTreeFarm, "树场") {
		return
	}

	slotCount, _ := model.GetTreeSlotCount(tgId)
	if slotCount >= int64(model.TreeFarmMaxSlots) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("树位已达上限 %d 个", model.TreeFarmMaxSlots)})
		return
	}

	price := common.TgBotTreeFarmSlotPrice
	if user.Quota < price {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足！扩展树位需要 $%.2f", webFarmQuotaFloat(price))})
		return
	}

	err := model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}

	newIdx := int(slotCount)
	err = model.CreateNewTreeSlot(tgId, newIdx)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扩展失败，已退款"})
		return
	}

	model.AddFarmLog(tgId, "tree_buyslot", -price, fmt.Sprintf("购买第%d个树位", newIdx+1))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功开垦第 %d 号树位！", newIdx+1),
	})
}
