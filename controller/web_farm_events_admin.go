package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 事件后端日志面板（A-4）—— admin 只读查询 A-2 天气事件 + A-3 突发事件。
// 挂在 /api/tgbot/farm/events*；tgBotRoute 已应用 AdminAuth 中间件。

// AdminGetFarmEvents GET /api/tgbot/farm/events?limit=50
// 返回两段数据合并：天气事件 + 随机事件
func AdminGetFarmEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	weatherList, _ := model.GetRecentWeatherEvents(limit)
	randomList, _ := model.GetRecentRandomEventsAll(limit)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"weather": mapAdminWeatherEventList(weatherList),
			"random":  mapAdminRandomEventList(randomList),
		},
	})
}

// AdminTriggerRandomEvent POST /api/tgbot/farm/events/trigger-random
// body: { "tg_id": "123" }  — admin 为指定玩家强制触发一条突发事件（忽略 12h 节流）
func AdminTriggerRandomEvent(c *gin.Context) {
	var req struct {
		TgId string `json:"tg_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TgId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	// admin 强制：跳过节流，直接起事件
	ev := adminForceRandomEvent(req.TgId)
	if ev == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "触发失败（玩家已有未结算事件或目录为空）"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已为玩家 %s 推送 %s %s", req.TgId, ev.Emoji, ev.Title),
		"data":    mapRandomEvent(ev, false),
	})
}

// ----- 辅助：管理员强制触发（绕过 12h 节流） -----

func adminForceRandomEvent(tgId string) *model.TgFarmRandomEvent {
	pending, _ := model.GetPendingRandomEvent(tgId)
	if pending != nil {
		return nil
	}
	if len(randomEventCatalog) == 0 {
		return nil
	}
	// admin 强制就不做随机了，按 catalog 第一个推；避免 admin 测试结果不稳定
	def := &randomEventCatalog[0]
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
		return nil
	}
	return ev
}

// ----- 辅助：映射 -----

func mapAdminWeatherEventList(list []*model.TgFarmWeatherEvent) []gin.H {
	out := make([]gin.H, 0, len(list))
	for _, ev := range list {
		out = append(out, gin.H{
			"id":           ev.Id,
			"event_key":    ev.EventKey,
			"name":         ev.Name,
			"emoji":        ev.Emoji,
			"severity":     ev.Severity,
			"started_at":   ev.StartedAt,
			"ends_at":      ev.EndsAt,
			"ended":        ev.Ended == 1,
			"last_tick_at": ev.LastTickAt,
			"narrative":    ev.Narrative,
		})
	}
	return out
}

func mapAdminRandomEventList(list []*model.TgFarmRandomEvent) []gin.H {
	out := make([]gin.H, 0, len(list))
	for _, ev := range list {
		out = append(out, gin.H{
			"id":          ev.Id,
			"tg_id":       ev.TgId,
			"event_key":   ev.EventKey,
			"title":       ev.Title,
			"emoji":       ev.Emoji,
			"narrative":   ev.Narrative,
			"chosen_idx":  ev.ChosenIdx,
			"outcome":     ev.Outcome,
			"started_at":  ev.StartedAt,
			"expires_at":  ev.ExpiresAt,
			"resolved_at": ev.ResolvedAt,
			"resolved":    ev.ChosenIdx >= 0,
		})
	}
	return out
}
