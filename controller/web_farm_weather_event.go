package controller

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 天气事件层（A-2）— 叠加在基础天气之上的稀有一次性事件。
// 基础天气：常态化，每 8-16 小时一轮，影响全局增长/事件/偷菜系数。
// 本层事件：稀有、剧情感强，直接作用到土壤 N/P/K/OM/Fatigue 或前端横幅。
//
// 触发节流：
//   1) 5 分钟评估一次
//   2) 已有活跃事件时跳过
//   3) 12 小时内已触发 ≥ 2 次时跳过
//   4) 基础触发概率 8%，按季节/基础天气做轻微调整

// ----- 事件定义 -----

type weatherEventDef struct {
	Key      string
	Name     string
	Emoji    string
	Severity int    // 1 轻 2 中 3 重
	MinSecs  int64  // 持续时长下限（秒）
	MaxSecs  int64  // 持续时长上限
	Narrative string

	// 基础权重（每个季节 map 里填具体）
	// 触发瞬间 patch（每块地一次）
	OnStart model.SoilPatch
	// 每小时 tick 一次的 patch（每块地一次）
	OnTick model.SoilPatch
}

var weatherEventCatalog = []weatherEventDef{
	{
		Key: "frost", Name: "霜降", Emoji: "❄️", Severity: 2,
		MinSecs: 2 * 3600, MaxSecs: 4 * 3600,
		Narrative:  "夜间霜冻突袭，未覆盖的作物可能冻伤",
		OnStart:    model.SoilPatch{DOM: -5, DFatigue: 5},
		OnTick:     model.SoilPatch{DN: -2},
	},
	{
		Key: "rainbow", Name: "彩虹日", Emoji: "🌈", Severity: 1,
		MinSecs: 1 * 3600, MaxSecs: 3 * 3600,
		Narrative:  "雨后双彩虹高悬，田野生机勃发",
		OnStart:    model.SoilPatch{DOM: 5},
		OnTick:     model.SoilPatch{DN: 1, DP: 1, DK: 1}, // 持续期内土壤缓慢变好
	},
	{
		Key: "heatwave", Name: "干热风", Emoji: "🔥", Severity: 3,
		MinSecs: 2 * 3600, MaxSecs: 5 * 3600,
		Narrative:  "干热风席卷而来，土壤急速失水",
		OnStart:    model.SoilPatch{DOM: -3},
		OnTick:     model.SoilPatch{DN: -3, DOM: -2},
	},
	{
		Key: "thunderstorm", Name: "雷阵雨", Emoji: "⛈️", Severity: 2,
		MinSecs: 1 * 3600, MaxSecs: 2 * 3600,
		Narrative:  "惊雷倾盆，伴随闪电催肥",
		OnStart:    model.SoilPatch{DN: 8, DOM: 4}, // 雷电固氮 + 雨水带走枯枝
		OnTick:     model.SoilPatch{},
	},
	{
		Key: "spring_fog", Name: "春雾", Emoji: "🌫️", Severity: 1,
		MinSecs: 2 * 3600, MaxSecs: 4 * 3600,
		Narrative:  "薄雾弥漫保温保湿，作物发芽提速",
		OnStart:    model.SoilPatch{DOM: 2},
		OnTick:     model.SoilPatch{DOM: 1},
	},
}

// 各季节 -> 事件 key -> 权重（0 表示不会出现）
var weatherEventSeasonWeight = map[int]map[string]int{
	0: {"frost": 1, "rainbow": 3, "heatwave": 0, "thunderstorm": 2, "spring_fog": 5},
	1: {"frost": 0, "rainbow": 2, "heatwave": 5, "thunderstorm": 4, "spring_fog": 0},
	2: {"frost": 2, "rainbow": 2, "heatwave": 1, "thunderstorm": 2, "spring_fog": 1},
	3: {"frost": 5, "rainbow": 1, "heatwave": 0, "thunderstorm": 0, "spring_fog": 0},
}

func findWeatherEventDef(key string) *weatherEventDef {
	for i := range weatherEventCatalog {
		if weatherEventCatalog[i].Key == key {
			return &weatherEventCatalog[i]
		}
	}
	return nil
}

// ----- 触发调度 -----

var (
	weatherEventTickOnce sync.Once
	lastWeatherEventCheck int64 // 上次评估时间
)

// StartWeatherEventTask 启动天气事件定时评估；挂在 master 节点
func StartWeatherEventTask() {
	weatherEventTickOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			// 启动延迟 30s 让基础天气先初始化
			time.Sleep(30 * time.Second)
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				runWeatherEventTick(time.Now().Unix())
				<-ticker.C
			}
		}()
	})
}

// runWeatherEventTick 评估是否触发新事件，并对活跃事件做每小时 tick
func runWeatherEventTick(now int64) {
	// 1. 对当前活跃事件推进 tick
	active, err := model.GetActiveWeatherEvent()
	if err == nil && active != nil {
		// 每小时 tick 一次
		if active.LastTickAt == 0 || now-active.LastTickAt >= 3600 {
			applyWeatherEventTick(active)
			_ = model.UpdateWeatherEventTickAt(active.Id, now)
		}
		// 到期清理
		if active.EndsAt <= now {
			_ = model.MarkWeatherEventEnded(active.Id)
			active = nil
		}
	}
	// 2. 无活跃事件时评估是否触发新事件
	if active == nil {
		if shouldFireWeatherEvent(now) {
			fireRandomWeatherEvent(now)
		}
	}
	lastWeatherEventCheck = now
}

// shouldFireWeatherEvent 综合节流 + 概率
func shouldFireWeatherEvent(now int64) bool {
	// 节流：12 小时内已触发 ≥ 2 次
	cnt, _ := model.CountWeatherEventsSince(12 * 3600)
	if cnt >= 2 {
		return false
	}
	// 基础概率 8%
	chance := 8
	// 基础天气处于 stormy / hot / cold 时事件更活跃
	base := GetCurrentWeather()
	switch base.Type {
	case 2, 6, 8:
		chance += 5
	}
	return rand.Intn(100) < chance
}

// fireRandomWeatherEvent 按季节权重摇一个事件
func fireRandomWeatherEvent(now int64) {
	season := getCurrentSeasonIndex()
	weights := weatherEventSeasonWeight[season]
	if len(weights) == 0 {
		return
	}
	total := 0
	for _, w := range weights {
		total += w
	}
	if total <= 0 {
		return
	}
	r := rand.Intn(total)
	cursor := 0
	picked := ""
	for key, w := range weights {
		if w <= 0 {
			continue
		}
		cursor += w
		if r < cursor {
			picked = key
			break
		}
	}
	def := findWeatherEventDef(picked)
	if def == nil {
		return
	}
	duration := def.MinSecs
	if def.MaxSecs > def.MinSecs {
		duration += rand.Int63n(def.MaxSecs - def.MinSecs + 1)
	}
	ev := &model.TgFarmWeatherEvent{
		EventKey:  def.Key,
		Name:      def.Name,
		Emoji:     def.Emoji,
		Severity:  def.Severity,
		StartedAt: now,
		EndsAt:    now + duration,
		Narrative: def.Narrative,
	}
	if err := model.CreateWeatherEvent(ev); err != nil {
		common.SysError(fmt.Sprintf("weather event create failed: %v", err))
		return
	}
	// 触发瞬间对所有地块打一次 OnStart patch
	applyWeatherEventStart(ev, def)
	common.SysLog(fmt.Sprintf("Weather event fired: %s %s (%d secs)", def.Emoji, def.Name, duration))
}

// applyWeatherEventStart 对所有地块施加一次开幕 patch
// 注意：空地块（status=0）也施加，这样休耕地也受事件影响；避免歧视。
func applyWeatherEventStart(ev *model.TgFarmWeatherEvent, def *weatherEventDef) {
	if isEmptyPatch(def.OnStart) {
		return
	}
	var plots []*model.TgFarmPlot
	if err := model.DB.Find(&plots).Error; err != nil {
		return
	}
	for _, p := range plots {
		_ = model.ApplySoilPatch(p, def.OnStart)
	}
}

// applyWeatherEventTick 每小时一次对所有地块施加 tick patch
func applyWeatherEventTick(ev *model.TgFarmWeatherEvent) {
	def := findWeatherEventDef(ev.EventKey)
	if def == nil || isEmptyPatch(def.OnTick) {
		return
	}
	var plots []*model.TgFarmPlot
	if err := model.DB.Find(&plots).Error; err != nil {
		return
	}
	for _, p := range plots {
		_ = model.ApplySoilPatch(p, def.OnTick)
	}
}

func isEmptyPatch(p model.SoilPatch) bool {
	return p.DN == 0 && p.DP == 0 && p.DK == 0 && p.DPH == 0 && p.DOM == 0 && p.DFatigue == 0
}

// ----- 对外接口 -----

// WebFarmWeatherEvent 前端 / 总览拉取活跃事件 + 最近 10 条历史
func WebFarmWeatherEvent(c *gin.Context) {
	_, _, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	active, _ := model.GetActiveWeatherEvent()
	recent, _ := model.GetRecentWeatherEvents(10)
	out := gin.H{
		"success": true,
		"data": gin.H{
			"active": nil,
			"recent": mapWeatherEventList(recent),
		},
	}
	if active != nil {
		out["data"].(gin.H)["active"] = mapWeatherEvent(active)
	}
	c.JSON(http.StatusOK, out)
}

func mapWeatherEvent(ev *model.TgFarmWeatherEvent) gin.H {
	now := time.Now().Unix()
	remain := ev.EndsAt - now
	if remain < 0 {
		remain = 0
	}
	return gin.H{
		"id":         ev.Id,
		"event_key":  ev.EventKey,
		"name":       ev.Name,
		"emoji":      ev.Emoji,
		"severity":   ev.Severity,
		"started_at": ev.StartedAt,
		"ends_at":    ev.EndsAt,
		"remain":     remain,
		"narrative":  ev.Narrative,
		"ended":      ev.Ended == 1,
	}
}

func mapWeatherEventList(list []*model.TgFarmWeatherEvent) []gin.H {
	out := make([]gin.H, 0, len(list))
	for _, ev := range list {
		out = append(out, mapWeatherEvent(ev))
	}
	return out
}
