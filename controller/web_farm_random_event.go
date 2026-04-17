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

// 突发事件中心（A-3）—— 玩家个人的叙事性小剧情。
// 对象粒度：每个玩家独立一条；12 小时窗口最多 1 条。
// 玩法：3 选项多分支，每种结果直接结算金币/种子/土壤补剂。
//
// 事件目录以代码内 catalog 形式定义，避免过早抽象到 DB。
// 未来如需运营热更，可把 catalog 移到 setting/ 或 model/。

// randomEventOption 一个选项
// Reward.Kind 枚举：
//   "quota"        -> Amount 正负增减金币
//   "seed"         -> Code=作物 key, Amount=数量
//   "soil_all"     -> 对全体地块施加 Patch
//   "nothing"      -> 纯文本结局，不扣不加
//   "fatigue_all"  -> 对全体地块加/减疲劳
type randomEventReward struct {
	Kind   string
	Code   string
	Amount int
	Patch  model.SoilPatch
}

type randomEventOption struct {
	Label   string              // 按钮文案
	Outcome string              // 结算文案模板，支持 {amount}
	Rewards []randomEventReward // 一个选项可有多项奖励
}

type randomEventDef struct {
	Key       string
	Title     string
	Emoji     string
	Narrative string
	Options   [3]randomEventOption
}

// 事件目录：4 个初版剧情
var randomEventCatalog = []randomEventDef{
	{
		Key: "beggar", Title: "路边乞丐", Emoji: "🥺",
		Narrative: "一位衣衫褴褛的老人拦住你，声称三天没吃东西，恳求你施舍一点。",
		Options: [3]randomEventOption{
			{
				Label: "给他 $10",
				Outcome: "你给了老人 $10，他感激涕零地塞给你一把神秘种子。",
				Rewards: []randomEventReward{
					{Kind: "quota", Amount: -10 * 500000},
					{Kind: "seed", Code: "carrot", Amount: 2},
				},
			},
			{
				Label: "给他 $50",
				Outcome: "老人双手颤抖接过 $50，留下一小瓶陈年肥料就消失在雨中。",
				Rewards: []randomEventReward{
					{Kind: "quota", Amount: -50 * 500000},
					{Kind: "soil_all", Patch: model.SoilPatch{DN: 3, DP: 3, DK: 3, DOM: 2}},
				},
			},
			{
				Label: "视而不见",
				Outcome: "你低头走过，老人低声嘟囔着什么。你隐约觉得今天的运气不太好。",
				Rewards: []randomEventReward{
					{Kind: "fatigue_all", Patch: model.SoilPatch{DFatigue: 3}},
				},
			},
		},
	},
	{
		Key: "merchant", Title: "流浪商人", Emoji: "🧙",
		Narrative: "一位头戴尖帽的神秘商人向你展示手中三件商品，语速极快让你必须立刻决定。",
		Options: [3]randomEventOption{
			{
				Label: "$100 换 10 袋番茄种子",
				Outcome: "交易成功！你多了 10 袋番茄种子。",
				Rewards: []randomEventReward{
					{Kind: "quota", Amount: -100 * 500000},
					{Kind: "seed", Code: "tomato", Amount: 10},
				},
			},
			{
				Label: "$200 换神秘肥料",
				Outcome: "你打开瓶子，浓郁的腐殖气息扑鼻——土壤大为改善。",
				Rewards: []randomEventReward{
					{Kind: "quota", Amount: -200 * 500000},
					{Kind: "soil_all", Patch: model.SoilPatch{DN: 5, DP: 5, DK: 5, DOM: 5, DFatigue: -5}},
				},
			},
			{
				Label: "转身离开",
				Outcome: "你警惕地拒绝了他。商人耸肩消失在拐角，你心里既庆幸又隐隐失落。",
				Rewards: []randomEventReward{
					{Kind: "nothing"},
				},
			},
		},
	},
	{
		Key: "old_farmer", Title: "隔壁老农", Emoji: "👴",
		Narrative: "邻居老张翻过篱笆问你借些东西，说是家里揭不开锅了。",
		Options: [3]randomEventOption{
			{
				Label: "借 $30 给他",
				Outcome: "老张千恩万谢地离开，第二天回赠了你一袋优质堆肥。",
				Rewards: []randomEventReward{
					{Kind: "quota", Amount: -30 * 500000},
					{Kind: "soil_all", Patch: model.SoilPatch{DOM: 4}},
				},
			},
			{
				Label: "送他 $10 做路费",
				Outcome: "老张紧紧握住你的手：\"这情分我记下了！\"次日他回赠了你 5 袋玉米种子。",
				Rewards: []randomEventReward{
					{Kind: "quota", Amount: -10 * 500000},
					{Kind: "seed", Code: "corn", Amount: 5},
				},
			},
			{
				Label: "婉言拒绝",
				Outcome: "老张尴尬地笑了笑转身离开。你继续干自己的活。",
				Rewards: []randomEventReward{
					{Kind: "nothing"},
				},
			},
		},
	},
	{
		Key: "thief", Title: "可疑身影", Emoji: "🕵️",
		Narrative: "半夜里你瞥见一个黑影在田边徘徊，眼看就要动手偷菜。",
		Options: [3]randomEventOption{
			{
				Label: "大声呵斥",
				Outcome: "黑影受惊跑了，你捡起他掉落的 $20。",
				Rewards: []randomEventReward{
					{Kind: "quota", Amount: 20 * 500000},
				},
			},
			{
				Label: "悄悄跟踪",
				Outcome: "你跟踪到他的据点缴获一小袋土壤改良剂！",
				Rewards: []randomEventReward{
					{Kind: "soil_all", Patch: model.SoilPatch{DN: 4, DP: 4, DK: 4}},
				},
			},
			{
				Label: "假装没看见",
				Outcome: "次日清晨你发现几块地被翻得一塌糊涂。",
				Rewards: []randomEventReward{
					{Kind: "fatigue_all", Patch: model.SoilPatch{DFatigue: 8}},
				},
			},
		},
	},
}

func findRandomEventDef(key string) *randomEventDef {
	for i := range randomEventCatalog {
		if randomEventCatalog[i].Key == key {
			return &randomEventCatalog[i]
		}
	}
	return nil
}

// ----- 触发 -----

// TriggerRandomEventIfEligible 若节流允许则给指定玩家推一条事件
// 返回新建事件或 nil。调用方不关心错误，因为这是后台 tick。
func TriggerRandomEventIfEligible(tgId string) *model.TgFarmRandomEvent {
	// 节流：12 小时内最多 1 条
	cnt, _ := model.CountRandomEventsSince(tgId, 12*3600)
	if cnt >= 1 {
		return nil
	}
	// 已有未结算？跳过
	pending, _ := model.GetPendingRandomEvent(tgId)
	if pending != nil {
		return nil
	}
	if len(randomEventCatalog) == 0 {
		return nil
	}
	def := &randomEventCatalog[rand.Intn(len(randomEventCatalog))]
	// 序列化 options 以便稳定展示（不做真正的 JSON，足够简单）
	ev := &model.TgFarmRandomEvent{
		TgId:       tgId,
		EventKey:   def.Key,
		Title:      def.Title,
		Emoji:      def.Emoji,
		Narrative:  def.Narrative,
		OptionsRaw: serializeOptions(def),
		ChosenIdx:  -1,
		ExpiresAt:  time.Now().Unix() + 24*3600,
	}
	if err := model.CreateRandomEvent(ev); err != nil {
		common.SysError(fmt.Sprintf("random event create failed: %v", err))
		return nil
	}
	return ev
}

func serializeOptions(def *randomEventDef) string {
	// 纯展示用；选项按钮文案即可。用 `|` 分隔三个 label。
	return def.Options[0].Label + "|" + def.Options[1].Label + "|" + def.Options[2].Label
}

// applyReward 结算一份奖励
func applyReward(user *model.User, tgId string, r randomEventReward) {
	switch r.Kind {
	case "quota":
		if r.Amount > 0 {
			_ = model.IncreaseUserQuota(user.Id, r.Amount, true)
		} else if r.Amount < 0 {
			_ = model.DecreaseUserQuota(user.Id, -r.Amount)
		}
		model.AddFarmLog(tgId, "random_event", r.Amount, "突发事件奖励")
	case "seed":
		seedKey := "seed_" + r.Code
		_ = model.IncrementFarmItem(tgId, seedKey, r.Amount)
		model.AddFarmLog(tgId, "random_event", 0,
			fmt.Sprintf("突发事件: %s 种子 %+d", r.Code, r.Amount))
	case "soil_all":
		applyPatchAllPlots(tgId, r.Patch)
		model.AddFarmLog(tgId, "random_event", 0, "突发事件: 土壤变化")
	case "fatigue_all":
		applyPatchAllPlots(tgId, r.Patch)
		model.AddFarmLog(tgId, "random_event", 0, "突发事件: 疲劳累积")
	case "nothing":
		// 无结算
	}
}

func applyPatchAllPlots(tgId string, patch model.SoilPatch) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		return
	}
	for _, p := range plots {
		_ = model.ApplySoilPatch(p, patch)
	}
}

// ----- HTTP 接口 -----

// WebFarmRandomEventView 拉取当前 pending 事件 + 最近 10 条历史
func WebFarmRandomEventView(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	pending, _ := model.GetPendingRandomEvent(tgId)
	recent, _ := model.GetRecentRandomEvents(tgId, 10)
	out := gin.H{
		"success": true,
		"data": gin.H{
			"pending": nil,
			"recent":  mapRandomEventList(recent),
		},
	}
	if pending != nil {
		out["data"].(gin.H)["pending"] = mapRandomEvent(pending, true)
	}
	c.JSON(http.StatusOK, out)
}

// WebFarmRandomEventChoose 玩家做出选择
func WebFarmRandomEventChoose(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		EventId     int `json:"event_id"`
		OptionIndex int `json:"option_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.OptionIndex < 0 || req.OptionIndex > 2 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效选项"})
		return
	}
	pending, _ := model.GetPendingRandomEvent(tgId)
	if pending == nil || pending.Id != req.EventId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "事件已过期或不存在"})
		return
	}
	def := findRandomEventDef(pending.EventKey)
	if def == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "事件定义缺失"})
		return
	}
	opt := def.Options[req.OptionIndex]
	// 前置检查：若选项会扣钱，先验证余额
	quotaCost := 0
	for _, r := range opt.Rewards {
		if r.Kind == "quota" && r.Amount < 0 {
			quotaCost += -r.Amount
		}
	}
	if quotaCost > 0 && user.Quota < quotaCost {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "余额不足"})
		return
	}
	// 结算所有奖励
	for _, r := range opt.Rewards {
		applyReward(user, tgId, r)
	}
	_ = model.ResolveRandomEvent(pending.Id, req.OptionIndex, opt.Outcome)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": opt.Outcome,
		"data": gin.H{
			"outcome":  opt.Outcome,
			"chosen":   req.OptionIndex,
			"event_id": pending.Id,
		},
	})
}

// WebFarmRandomEventTrigger 调试/测试用接口：立即给自己推一条事件
// 生产环境由 tick 自动触发，这里保留手动入口便于 QA。
func WebFarmRandomEventTrigger(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	ev := TriggerRandomEventIfEligible(tgId)
	if ev == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前无法触发（冷却中或已有事件）"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "事件已生成",
		"data":    mapRandomEvent(ev, true),
	})
}

func mapRandomEvent(ev *model.TgFarmRandomEvent, includeOptions bool) gin.H {
	out := gin.H{
		"id":          ev.Id,
		"event_key":   ev.EventKey,
		"title":       ev.Title,
		"emoji":       ev.Emoji,
		"narrative":   ev.Narrative,
		"chosen_idx":  ev.ChosenIdx,
		"outcome":     ev.Outcome,
		"started_at":  ev.StartedAt,
		"expires_at":  ev.ExpiresAt,
		"resolved_at": ev.ResolvedAt,
	}
	if includeOptions {
		// 从 catalog 现取展示，不信任 DB 中序列化的 label（catalog 可能升级）
		def := findRandomEventDef(ev.EventKey)
		if def != nil {
			labels := make([]string, 0, 3)
			for _, o := range def.Options {
				labels = append(labels, o.Label)
			}
			out["options"] = labels
		} else {
			// 兼容旧事件 key：从 DB 序列化字段拆
			out["options"] = splitPipe(ev.OptionsRaw)
		}
	}
	return out
}

func mapRandomEventList(list []*model.TgFarmRandomEvent) []gin.H {
	out := make([]gin.H, 0, len(list))
	for _, ev := range list {
		out = append(out, mapRandomEvent(ev, false))
	}
	return out
}

func splitPipe(s string) []string {
	// 手写拆，不引 strings 以缩减 import
	parts := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
