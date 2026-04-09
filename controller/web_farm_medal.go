package controller

import (
	"fmt"
	"math/rand"
	"net/http"
	"sort"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type farmMedalDef struct {
	Key         string
	Name        string
	Emoji       string
	Description string
	Rarity      string
	RarityLabel string
	Animation   string
	ColorFrom   string
	ColorTo     string
	GlowColor   string
}

type farmMedalChoice struct {
	Key    string
	Weight int
}

type farmMedalPool struct {
	Chance  int
	Choices []farmMedalChoice
}

var farmMedalDefs = map[string]farmMedalDef{
	"sprout": {Key: "sprout", Name: "新芽之歌", Emoji: "🌱", Description: "土地回应了你的照料，新的生机在指尖发芽。", Rarity: "common", RarityLabel: "常见勋章", Animation: "bloom", ColorFrom: "#22c55e", ColorTo: "#86efac", GlowColor: "rgba(34, 197, 94, 0.45)"},
	"dew": {Key: "dew", Name: "晨露微光", Emoji: "💧", Description: "清晨的露珠在叶尖折光，记录下你耐心的浇灌。", Rarity: "common", RarityLabel: "常见勋章", Animation: "ripple", ColorFrom: "#38bdf8", ColorTo: "#a5f3fc", GlowColor: "rgba(56, 189, 248, 0.4)"},
	"sunharvest": {Key: "sunharvest", Name: "晴穗荣光", Emoji: "🌞", Description: "当丰收落进仓里，阳光会为勤劳的人镀上一层金边。", Rarity: "uncommon", RarityLabel: "稀有勋章", Animation: "flare", ColorFrom: "#f59e0b", ColorTo: "#fde68a", GlowColor: "rgba(245, 158, 11, 0.45)"},
	"ember": {Key: "ember", Name: "工坊火印", Emoji: "⚒️", Description: "锻造台上跃动的火花，为你的巧手留下了专属印记。", Rarity: "rare", RarityLabel: "珍贵勋章", Animation: "forge", ColorFrom: "#fb7185", ColorTo: "#f97316", GlowColor: "rgba(249, 115, 22, 0.46)"},
	"clover": {Key: "clover", Name: "幸运四叶章", Emoji: "🍀", Description: "运气在这一刻悄悄偏向了你，连风都替你庆祝。", Rarity: "rare", RarityLabel: "珍贵勋章", Animation: "orbit", ColorFrom: "#10b981", ColorTo: "#22c55e", GlowColor: "rgba(16, 185, 129, 0.46)"},
	"angler": {Key: "angler", Name: "星鳞渔章", Emoji: "🎣", Description: "你甩出的钓线划过水面，像把夜色里的星光也一并钓起。", Rarity: "epic", RarityLabel: "史诗勋章", Animation: "comet", ColorFrom: "#60a5fa", ColorTo: "#a78bfa", GlowColor: "rgba(96, 165, 250, 0.5)"},
	"forest": {Key: "forest", Name: "林语徽记", Emoji: "🌲", Description: "林地把枝叶低语编成纹章，只送给真正懂得照料森林的人。", Rarity: "rare", RarityLabel: "珍贵勋章", Animation: "leaf", ColorFrom: "#16a34a", ColorTo: "#84cc16", GlowColor: "rgba(34, 197, 94, 0.5)"},
	"shadow": {Key: "shadow", Name: "夜行采影", Emoji: "🦝", Description: "在夜色与机敏之间，你悄悄带走了一缕不属于白昼的战利品。", Rarity: "rare", RarityLabel: "珍贵勋章", Animation: "shadow", ColorFrom: "#8b5cf6", ColorTo: "#c084fc", GlowColor: "rgba(139, 92, 246, 0.48)"},
	"pasture": {Key: "pasture", Name: "牧歌之心", Emoji: "🐄", Description: "饲草、清水与耐心，让牧场把温柔也镌进了你的徽章。", Rarity: "uncommon", RarityLabel: "稀有勋章", Animation: "pulse", ColorFrom: "#f97316", ColorTo: "#fdba74", GlowColor: "rgba(249, 115, 22, 0.42)"},
}

var farmMedalPools = map[string]farmMedalPool{
	"plant":     {Chance: 12, Choices: []farmMedalChoice{{Key: "sprout", Weight: 70}, {Key: "dew", Weight: 20}, {Key: "sunharvest", Weight: 10}}},
	"water":     {Chance: 10, Choices: []farmMedalChoice{{Key: "dew", Weight: 60}, {Key: "sprout", Weight: 25}, {Key: "sunharvest", Weight: 15}}},
	"fertilize": {Chance: 10, Choices: []farmMedalChoice{{Key: "dew", Weight: 45}, {Key: "sprout", Weight: 20}, {Key: "sunharvest", Weight: 35}}},
	"treat":     {Chance: 9, Choices: []farmMedalChoice{{Key: "dew", Weight: 35}, {Key: "sunharvest", Weight: 30}, {Key: "sprout", Weight: 35}}},
	"harvest":   {Chance: 14, Choices: []farmMedalChoice{{Key: "sunharvest", Weight: 60}, {Key: "sprout", Weight: 20}, {Key: "clover", Weight: 20}}},
	"workshop":  {Chance: 12, Choices: []farmMedalChoice{{Key: "ember", Weight: 68}, {Key: "sunharvest", Weight: 12}, {Key: "clover", Weight: 20}}},
	"game":      {Chance: 18, Choices: []farmMedalChoice{{Key: "clover", Weight: 72}, {Key: "ember", Weight: 14}, {Key: "sunharvest", Weight: 14}}},
	"fish":      {Chance: 14, Choices: []farmMedalChoice{{Key: "angler", Weight: 72}, {Key: "clover", Weight: 18}, {Key: "dew", Weight: 10}}},
	"tree":      {Chance: 12, Choices: []farmMedalChoice{{Key: "forest", Weight: 72}, {Key: "sprout", Weight: 14}, {Key: "dew", Weight: 14}}},
	"steal":     {Chance: 10, Choices: []farmMedalChoice{{Key: "shadow", Weight: 72}, {Key: "clover", Weight: 28}}},
	"ranch":     {Chance: 11, Choices: []farmMedalChoice{{Key: "pasture", Weight: 70}, {Key: "sunharvest", Weight: 15}, {Key: "dew", Weight: 15}}},
}

var farmMedalSourceLabels = map[string]string{
	"plant": "种植", "water": "浇灌", "fertilize": "施肥", "treat": "治疗", "harvest": "收获",
	"workshop": "加工坊", "game": "小游戏", "fish": "钓鱼", "tree": "树场", "steal": "偷菜", "ranch": "牧场",
}

func buildFarmMedalData(def farmMedalDef, quantity int, firstAt int64, isNew bool, source string) gin.H {
	return gin.H{
		"key":          def.Key,
		"name":         def.Name,
		"emoji":        def.Emoji,
		"description":  def.Description,
		"rarity":       def.Rarity,
		"rarity_label": def.RarityLabel,
		"animation":    def.Animation,
		"color_from":   def.ColorFrom,
		"color_to":     def.ColorTo,
		"glow_color":   def.GlowColor,
		"quantity":     quantity,
		"first_at":     firstAt,
		"is_new":       isNew,
		"source":       source,
		"source_label": farmMedalSourceLabels[source],
	}
}

func maybeFarmMedalDrop(tgId, source string) gin.H {
	pool, ok := farmMedalPools[source]
	if !ok || pool.Chance <= 0 || len(pool.Choices) == 0 {
		return nil
	}
	if rand.Intn(100) >= pool.Chance {
		return nil
	}
	totalWeight := 0
	for _, choice := range pool.Choices {
		totalWeight += choice.Weight
	}
	if totalWeight <= 0 {
		return nil
	}
	roll := rand.Intn(totalWeight)
	pickedKey := pool.Choices[0].Key
	acc := 0
	for _, choice := range pool.Choices {
		acc += choice.Weight
		if roll < acc {
			pickedKey = choice.Key
			break
		}
	}
	def, ok := farmMedalDefs[pickedKey]
	if !ok {
		return nil
	}
	isNew, quantity, firstAt, err := model.RecordCollectionWithStatus(tgId, "medal", def.Key, 1)
	if err != nil {
		return nil
	}
	model.AddFarmLog(tgId, "medal_drop", 0, fmt.Sprintf("掉落勋章%s%s", def.Emoji, def.Name))
	return buildFarmMedalData(def, quantity, firstAt, isNew, source)
}

func respondFarmSuccessWithMedal(c *gin.Context, tgId, source, message string, data gin.H) {
	payload := gin.H{"success": true, "message": message}
	if data != nil {
		payload["data"] = data
	}
	if medalDrop := maybeFarmMedalDrop(tgId, source); medalDrop != nil {
		payload["medal_drop"] = medalDrop
	}
	c.JSON(http.StatusOK, payload)
}

func buildFarmMedalCollectionData(tgId string) []gin.H {
	collections, err := model.GetCollections(tgId)
	if err != nil || len(collections) == 0 {
		return []gin.H{}
	}
	items := make([]gin.H, 0)
	for _, item := range collections {
		if item.Category != "medal" {
			continue
		}
		def, ok := farmMedalDefs[item.ItemKey]
		if !ok {
			continue
		}
		items = append(items, buildFarmMedalData(def, item.Quantity, item.FirstAt, false, ""))
	}
	sort.Slice(items, func(i, j int) bool {
		left, _ := items[i]["first_at"].(int64)
		right, _ := items[j]["first_at"].(int64)
		if left == right {
			leftQty, _ := items[i]["quantity"].(int)
			rightQty, _ := items[j]["quantity"].(int)
			return leftQty > rightQty
		}
		return left > right
	})
	return items
}
