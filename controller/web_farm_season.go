package controller

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ═══════════════════════════════════════════════════════════
//  赛季系统 — Controller
// ═══════════════════════════════════════════════════════════

// 高级化肥属性常量
const (
	AdvFertilizerGrowReduction = 50 // 成熟时间缩短 50%
	AdvFertilizerYieldBoost    = 30 // 产量提升 30%
	NormalFertilizerGrowReduction = 20 // 普通化肥：成熟时间缩短 20%
	NormalFertilizerYieldBoost    = 10 // 普通化肥：产量提升 10%
)

// ────── 管理员：赛季 CRUD ──────

func AdminGetAllSeasons(c *gin.Context) {
	seasons, err := model.GetAllSeasons()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取赛季列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": seasons})
}

func AdminCreateSeason(c *gin.Context) {
	var req model.TgFarmSeason
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误：需要 code"})
		return
	}
	if req.WeeksPerSeason <= 0 {
		req.WeeksPerSeason = 4
	}
	if req.PointsMultiplier <= 0 {
		req.PointsMultiplier = 100
	}
	if req.Status == 0 && req.StartAt == 0 {
		req.Status = model.SeasonStatusPending
	}
	if err := model.CreateSeason(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建赛季失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "赛季创建成功", "data": req})
}

func AdminUpdateSeason(c *gin.Context) {
	var req model.TgFarmSeason
	if err := c.ShouldBindJSON(&req); err != nil || req.Id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	existing, err := model.GetSeasonById(req.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "赛季不存在"})
		return
	}
	existing.Code = req.Code
	existing.WeeksPerSeason = req.WeeksPerSeason
	existing.RushDays = req.RushDays
	existing.RestDays = req.RestDays
	existing.Status = req.Status
	existing.StartAt = req.StartAt
	existing.EndAt = req.EndAt
	existing.PointsMultiplier = req.PointsMultiplier
	if err := model.UpdateSeason(existing); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新赛季失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "赛季更新成功", "data": existing})
}

func AdminDeleteSeason(c *gin.Context) {
	var req struct {
		Id int `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.DeleteSeason(req.Id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "赛季已删除"})
}

// AdminStartSeason 管理员手动启动赛季
func AdminStartSeason(c *gin.Context) {
	var req struct {
		Id int `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	season, err := model.GetSeasonById(req.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "赛季不存在"})
		return
	}
	if season.Status != model.SeasonStatusPending {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "赛季状态不是「未开始」，无法启动"})
		return
	}
	now := time.Now().Unix()
	totalDays := season.WeeksPerSeason * 7
	season.StartAt = now
	season.EndAt = now + int64(totalDays)*86400
	season.Status = model.SeasonStatusRush
	if err := model.UpdateSeason(season); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "启动失败"})
		return
	}
	common.SysLog(fmt.Sprintf("Admin started season %d (%s)", season.Id, season.Code))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "赛季已启动", "data": season})
}

// AdminEndSeason 管理员手动结束赛季
func AdminEndSeason(c *gin.Context) {
	var req struct {
		Id int `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	season, err := model.GetSeasonById(req.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "赛季不存在"})
		return
	}
	if season.Status == model.SeasonStatusFinished {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "赛季已结束"})
		return
	}
	executeSeasonEnd(season)
	common.SysLog(fmt.Sprintf("Admin ended season %d (%s)", season.Id, season.Code))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "赛季已结束"})
}

// ────── 管理员：段位 CRUD ──────

func AdminGetAllTiers(c *gin.Context) {
	tiers, err := model.GetAllSeasonTiers()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取段位列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tiers})
}

func AdminCreateTier(c *gin.Context) {
	var req model.TgFarmSeasonTier
	if err := c.ShouldBindJSON(&req); err != nil || req.TierKey == "" || req.TierName == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误：需要 tier_key 和 tier_name"})
		return
	}
	if err := model.CreateSeasonTier(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建段位失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "段位创建成功", "data": req})
}

func AdminUpdateTier(c *gin.Context) {
	var req model.TgFarmSeasonTier
	if err := c.ShouldBindJSON(&req); err != nil || req.Id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.UpdateSeasonTier(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新段位失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "段位更新成功", "data": req})
}

func AdminDeleteTier(c *gin.Context) {
	var req struct {
		Id int `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.DeleteSeasonTier(req.Id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "段位已删除"})
}

// ────── 管理员：积分规则 CRUD ──────

func AdminGetPointsRules(c *gin.Context) {
	rules, err := model.GetAllSeasonPointsRules()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取积分规则失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rules})
}

func AdminSavePointsRule(c *gin.Context) {
	var req model.TgFarmSeasonPointsRule
	if err := c.ShouldBindJSON(&req); err != nil || req.Action == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.Id > 0 {
		if err := model.UpdateSeasonPointsRule(&req); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
			return
		}
	} else {
		if err := model.CreateSeasonPointsRule(&req); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建失败: " + err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "保存成功", "data": req})
}

func AdminDeletePointsRule(c *gin.Context) {
	var req struct {
		Id int `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.DeleteSeasonPointsRule(req.Id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已删除"})
}

// ────── 管理员：防作弊规则 CRUD ──────

func AdminGetAntiCheatRules(c *gin.Context) {
	rules, err := model.GetAllAntiCheatRules()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取防作弊规则失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rules})
}

func AdminSaveAntiCheatRule(c *gin.Context) {
	var req model.TgFarmSeasonAntiCheatRule
	if err := c.ShouldBindJSON(&req); err != nil || req.RuleKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.Id > 0 {
		if err := model.UpdateAntiCheatRule(&req); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
			return
		}
	} else {
		if err := model.CreateAntiCheatRule(&req); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建失败"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "保存成功", "data": req})
}

func AdminDeleteAntiCheatRule(c *gin.Context) {
	var req struct {
		Id int `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.DeleteAntiCheatRule(req.Id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已删除"})
}

func AdminGetAntiCheatLogs(c *gin.Context) {
	seasonId := 0
	if v, ok := c.GetQuery("season_id"); ok {
		fmt.Sscanf(v, "%d", &seasonId)
	}
	logs, err := model.GetAntiCheatLogs(seasonId, 200)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取防作弊日志失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": logs})
}

// ────── 玩家接口 ──────

// WebFarmSeasonOverview 获取当前赛季总览
func WebFarmSeasonOverview(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	season, err := model.GetActiveSeason()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"active": false,
				"message": "当前没有进行中的赛季",
			},
		})
		return
	}

	player, _ := model.GetOrCreateSeasonPlayer(tgId, season.Id)
	rank, _ := model.GetSeasonPlayerRank(tgId, season.Id)
	tiers, _ := model.GetAllSeasonTiers()
	playerCount := model.GetSeasonAllPlayerCount(season.Id)

	// 当前段位详情
	var currentTier *model.TgFarmSeasonTier
	var nextTier *model.TgFarmSeasonTier
	for i := range tiers {
		if tiers[i].TierKey == player.CurrentTierKey {
			currentTier = &tiers[i]
		}
		if currentTier != nil && tiers[i].TierLevel == currentTier.TierLevel+1 {
			nextTier = &tiers[i]
		}
	}

	tierInfo := gin.H{}
	if currentTier != nil {
		tierInfo["key"] = currentTier.TierKey
		tierInfo["name"] = currentTier.TierName
		tierInfo["emoji"] = currentTier.Emoji
		tierInfo["color"] = currentTier.Color
		tierInfo["level"] = currentTier.TierLevel
	}

	nextTierInfo := gin.H{}
	if nextTier != nil {
		nextTierInfo["key"] = nextTier.TierKey
		nextTierInfo["name"] = nextTier.TierName
		nextTierInfo["min_points"] = nextTier.MinPoints
		nextTierInfo["points_needed"] = nextTier.MinPoints - player.Points
	}

	now := time.Now().Unix()
	daysLeft := 0
	if season.EndAt > now {
		daysLeft = int((season.EndAt - now) / 86400)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"active":        true,
			"season_id":     season.Id,
			"season_code":   season.Code,
			"status":        season.Status,
			"start_at":      season.StartAt,
			"end_at":        season.EndAt,
			"days_left":     daysLeft,
			"multiplier":    season.PointsMultiplier,
			"points":        player.Points,
			"rank":          rank,
			"player_count":  playerCount,
			"current_tier":  tierInfo,
			"next_tier":     nextTierInfo,
			"highest_tier":  player.HighestTierKey,
			"inherited_from": player.InheritedFrom,
		},
	})
}

// WebFarmSeasonLeaderboard 赛季排行榜
func WebFarmSeasonLeaderboard(c *gin.Context) {
	season, err := model.GetActiveSeason()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"entries": []interface{}{}}})
		return
	}

	players, err := model.GetSeasonLeaderboard(season.Id, 50)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取排行榜失败"})
		return
	}

	entries := make([]gin.H, 0, len(players))
	for i, p := range players {
		// 查找用户名
		username := p.TelegramId
		u := &model.User{TelegramId: p.TelegramId}
		if p.TelegramId != "" && u.FillUserByTelegramId() == nil {
			if u.DisplayName != "" {
				username = u.DisplayName
			} else if u.Username != "" {
				username = u.Username
			}
		}

		entries = append(entries, gin.H{
			"rank":       i + 1,
			"telegram_id": p.TelegramId,
			"username":   username,
			"points":     p.Points,
			"tier_key":   p.CurrentTierKey,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"season_id":   season.Id,
			"season_code": season.Code,
			"entries":     entries,
		},
	})
}

// WebFarmSeasonPointsLogs 积分流水
func WebFarmSeasonPointsLogs(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	season, err := model.GetActiveSeason()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
		return
	}
	logs, _ := model.GetSeasonPointsLogs(tgId, season.Id, 100)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": logs})
}

// WebFarmSeasonTiers 获取所有段位信息
func WebFarmSeasonTiers(c *gin.Context) {
	tiers, err := model.GetAllSeasonTiers()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取段位列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tiers})
}

// WebFarmSeasonHistory 获取玩家赛季历史
func WebFarmSeasonPlayerHistory(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	histories, _ := model.GetPlayerSeasonHistories(tgId)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": histories})
}

// ═══════════════════════════════════════════════════════════
//  赛季自动任务 — 定时检查赛季状态
// ═══════════════════════════════════════════════════════════

var seasonTaskOnce sync.Once

func StartSeasonAutoTask() {
	seasonTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			// 首次运行
			runSeasonAutoTick()
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				runSeasonAutoTick()
			}
		}()
	})
}

func runSeasonAutoTick() {
	now := time.Now().Unix()

	// 检查是否有赛季需要启动
	var pendingSeasons []model.TgFarmSeason
	model.DB.Where("status = ? AND start_at > 0 AND start_at <= ?", model.SeasonStatusPending, now).Find(&pendingSeasons)
	for _, season := range pendingSeasons {
		season.Status = model.SeasonStatusRush
		_ = model.UpdateSeason(&season)
		common.SysLog(fmt.Sprintf("Season auto-started: %d (%s)", season.Id, season.Code))
	}

	// 检查是否有赛季需要进入休赛期
	var rushSeasons []model.TgFarmSeason
	model.DB.Where("status = ? AND end_at > 0 AND end_at <= ?", model.SeasonStatusRush, now).Find(&rushSeasons)
	for _, season := range rushSeasons {
		if season.RestDays > 0 {
			season.Status = model.SeasonStatusRest
			_ = model.UpdateSeason(&season)
			common.SysLog(fmt.Sprintf("Season entered rest period: %d (%s)", season.Id, season.Code))
		} else {
			executeSeasonEnd(&season)
			common.SysLog(fmt.Sprintf("Season auto-ended: %d (%s)", season.Id, season.Code))
		}
	}

	// 检查是否有赛季休赛期结束
	var restSeasons []model.TgFarmSeason
	model.DB.Where("status = ?", model.SeasonStatusRest).Find(&restSeasons)
	for _, season := range restSeasons {
		restEnd := season.EndAt + int64(season.RestDays)*86400
		if now >= restEnd {
			executeSeasonEnd(&season)
			common.SysLog(fmt.Sprintf("Season rest ended, season finished: %d (%s)", season.Id, season.Code))
		}
	}
}

// executeSeasonEnd 执行赛季结算
func executeSeasonEnd(season *model.TgFarmSeason) {
	// 1. 按积分排名保存历史
	_ = model.ResetAllPlayersForNewSeason(season.Id)

	// 2. 标记赛季为已结束
	season.Status = model.SeasonStatusFinished
	_ = model.UpdateSeason(season)

	common.SysLog(fmt.Sprintf("Season %d (%s) fully ended. Histories saved.", season.Id, season.Code))
}

// ═══════════════════════════════════════════════════════════
//  积分获取逻辑 — 嵌入现有操作
// ═══════════════════════════════════════════════════════════

// TryAwardSeasonPoints 在操作成功后调用，尝试给玩家发放赛季积分
// action: water/harvest/plant/steal/fish/workshop/challenge_rare_crop 等
func TryAwardSeasonPoints(telegramId string, action string, detail string) {
	season, err := model.GetActiveSeason()
	if err != nil || season.Status != model.SeasonStatusRush {
		return // 不在冲榜期，不发积分
	}

	rule, err := model.GetSeasonPointsRule(action)
	if err != nil || !rule.Enabled || rule.Points <= 0 {
		return
	}

	// 检查每日上限
	if rule.DailyCap > 0 {
		todayStr := time.Now().Format("2006-01-02")
		var todayTotal int
		model.DB.Model(&model.TgFarmSeasonPointsLog{}).
			Where("telegram_id = ? AND season_id = ? AND action = ? AND created_at > ?",
				telegramId, season.Id, action,
				todayStartUnix(todayStr)).
			Select("COALESCE(SUM(points),0)").Scan(&todayTotal)
		if todayTotal >= rule.DailyCap {
			return
		}
	}

	// 应用赛季倍率
	points := rule.Points * season.PointsMultiplier / 100
	if points <= 0 {
		points = 1
	}

	_ = model.AddSeasonPoints(telegramId, season.Id, action, points, detail)

	// 防作弊检测
	go checkSeasonAntiCheat(telegramId, season.Id)
}

func todayStartUnix(dateStr string) int64 {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Now().Unix() - 86400
	}
	return t.Unix()
}

// ═══════════════════════════════════════════════════════════
//  防作弊检测
// ═══════════════════════════════════════════════════════════

func checkSeasonAntiCheat(telegramId string, seasonId int) {
	rules, err := model.GetAllAntiCheatRules()
	if err != nil {
		return
	}
	for _, rule := range rules {
		if !rule.Enabled || rule.Threshold <= 0 || rule.WindowSecs <= 0 {
			continue
		}
		switch rule.RuleKey {
		case "points_spike":
			total, err := model.CountRecentSeasonPoints(telegramId, seasonId, int64(rule.WindowSecs))
			if err != nil {
				continue
			}
			if total >= rule.Threshold {
				_ = model.CreateAntiCheatLog(&model.TgFarmSeasonAntiCheat{
					TelegramId: telegramId,
					SeasonId:   seasonId,
					RuleKey:    rule.RuleKey,
					Detail:     fmt.Sprintf("积分暴涨: %d秒内获得%d积分(阈值%d)", rule.WindowSecs, total, rule.Threshold),
					Severity:   rule.Action,
				})
				common.SysLog(fmt.Sprintf("Anti-cheat triggered [%s] for %s: %d points in %ds (threshold %d)",
					rule.RuleKey, telegramId, total, rule.WindowSecs, rule.Threshold))
			}
		case "item_anomaly":
			totalQty, err := model.CountRecentAdvancedFertilizerPurchases(telegramId, int64(rule.WindowSecs))
			if err != nil {
				continue
			}
			if totalQty >= rule.Threshold {
				_ = model.CreateAntiCheatLog(&model.TgFarmSeasonAntiCheat{
					TelegramId: telegramId,
					SeasonId:   seasonId,
					RuleKey:    rule.RuleKey,
					Detail:     fmt.Sprintf("高级化肥异常购买: %d秒内购买%d个(阈值%d)", rule.WindowSecs, totalQty, rule.Threshold),
					Severity:   rule.Action,
				})
				common.SysLog(fmt.Sprintf("Anti-cheat triggered [%s] for %s: %d advanced fertilizers in %ds (threshold %d)",
					rule.RuleKey, telegramId, totalQty, rule.WindowSecs, rule.Threshold))
			}
		}
	}
}

// ═══════════════════════════════════════════════════════════
//  初始化默认段位与规则（首次运行时种子数据）
// ═══════════════════════════════════════════════════════════

var seedOnce sync.Once

func SeedSeasonDefaults() {
	seedOnce.Do(func() {
 		defaultTiers := []model.TgFarmSeasonTier{
 			{TierKey: "bronze", TierName: "青铜", TierLevel: 0, MinPoints: 0, InitialBalance: 20 * int(common.QuotaPerUnit), GiftItems: "[]", Emoji: "🥉", Color: "#cd7f32"},
 			{TierKey: "silver", TierName: "白银", TierLevel: 1, MinPoints: 500, InitialBalance: 50 * int(common.QuotaPerUnit), GiftItems: `[{"item":"fertilizer","qty":10},{"item":"fishbait","qty":10}]`, Emoji: "🥈", Color: "#c0c0c0"},
 			{TierKey: "gold", TierName: "黄金", TierLevel: 2, MinPoints: 1500, InitialBalance: 100 * int(common.QuotaPerUnit), GiftItems: `[{"item":"fertilizer_adv","qty":10},{"item":"fertilizer","qty":20}]`, Emoji: "🥇", Color: "#ffd700"},
 			{TierKey: "platinum", TierName: "铂金", TierLevel: 3, MinPoints: 3000, InitialBalance: 130 * int(common.QuotaPerUnit), GiftItems: "[]", Emoji: "💎", Color: "#e5e4e2"},
			{TierKey: "diamond", TierName: "钻石", TierLevel: 4, MinPoints: 6000, InitialBalance: 200 * int(common.QuotaPerUnit), GiftItems: `[{"item":"fertilizer_adv","qty":20},{"item":"premiumfishbait","qty":20}]`, Emoji: "💠", Color: "#b9f2ff"},
			{TierKey: "rich", TierName: "富可敌国", TierLevel: 5, MinPoints: 12000, InitialBalance: 500 * int(common.QuotaPerUnit), GiftItems: `[{"item":"fertilizer_adv","qty":50},{"item":"premiumfishbait","qty":50},{"item":"fertilizer","qty":50}]`, Emoji: "👑", Color: "#ff4500"},
 		}
		for _, tier := range defaultTiers {
			if _, err := model.GetSeasonTierByKey(tier.TierKey); err != nil {
				_ = model.CreateSeasonTier(&tier)
			}
		}
		common.SysLog("Season: seeded default tier configuration")

		// 默认积分规则
		defaultRules := []model.TgFarmSeasonPointsRule{
			{Action: "water", ActionName: "浇水", Points: 1, DailyCap: 50, Enabled: true},
			{Action: "fertilize", ActionName: "施肥", Points: 2, DailyCap: 20, Enabled: true},
			{Action: "harvest", ActionName: "收割", Points: 3, DailyCap: 30, Enabled: true},
			{Action: "plant", ActionName: "种植", Points: 2, DailyCap: 30, Enabled: true},
			{Action: "steal", ActionName: "偷菜", Points: 2, DailyCap: 10, Enabled: true},
			{Action: "fish", ActionName: "钓鱼", Points: 2, DailyCap: 20, Enabled: true},
			{Action: "workshop", ActionName: "加工坊制作", Points: 3, DailyCap: 10, Enabled: true},
			{Action: "challenge_rare_crop", ActionName: "挑战·种植稀有作物", Points: 10, DailyCap: 5, Enabled: true},
			{Action: "levelup", ActionName: "升级", Points: 20, DailyCap: 0, Enabled: true},
			{Action: "prestige", ActionName: "转生", Points: 50, DailyCap: 0, Enabled: true},
		}
		for _, rule := range defaultRules {
			if _, err := model.GetSeasonPointsRule(rule.Action); err != nil {
				_ = model.CreateSeasonPointsRule(&rule)
			}
		}
		common.SysLog("Season: seeded default points rules")

		// 默认防作弊规则
		defaultAntiCheat := []model.TgFarmSeasonAntiCheatRule{
			{RuleKey: "points_spike", RuleName: "积分暴涨检测", Enabled: true, Threshold: 200, WindowSecs: 600, Action: 1, BanDuration: 3600},
			{RuleKey: "item_anomaly", RuleName: "高级化肥异常购买", Enabled: true, Threshold: 20, WindowSecs: 600, Action: 1, BanDuration: 3600},
		}
		for _, rule := range defaultAntiCheat {
			var existing model.TgFarmSeasonAntiCheatRule
			if err := model.DB.Where("rule_key = ?", rule.RuleKey).First(&existing).Error; err != nil {
				_ = model.CreateAntiCheatRule(&rule)
			}
		}
		common.SysLog("Season: seeded default anti-cheat rules")
	})
}
