package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var betaCleanupOnce sync.Once

// FarmBetaStatus returns the beta status for the current user (or public info if not logged in)
func FarmBetaStatus(c *gin.Context) {
	totalReserved, _ := model.CountFarmBetaReservations()
	maxSlots := common.FarmBetaMaxSlots
	betaEnabled := common.FarmBetaEnabled

	// Parse countdown date to determine if farm is open
	farmOpen := false
	countdownDate := ""
	common.OptionMapRWMutex.RLock()
	countdownDate = common.OptionMap["FarmCountdownDate"]
	common.OptionMapRWMutex.RUnlock()

	if countdownDate != "" {
		t, err := time.Parse(time.RFC3339, countdownDate)
		if err == nil && time.Now().After(t) {
			farmOpen = true
		}
	}

	// Calculate beta end time
	betaExpired := false
	betaEndTime := ""
	if farmOpen && countdownDate != "" {
		start, err := time.Parse(time.RFC3339, countdownDate)
		if err == nil {
			end := start.Add(time.Duration(common.FarmBetaDurationDays) * 24 * time.Hour)
			betaEndTime = end.Format(time.RFC3339)
			if time.Now().After(end) {
				betaExpired = true
			}
		}
	}

	result := gin.H{
		"beta_enabled":    betaEnabled,
		"farm_open":       farmOpen,
		"beta_expired":    betaExpired,
		"beta_end_time":   betaEndTime,
		"beta_duration":   common.FarmBetaDurationDays,
		"max_slots":       maxSlots,
		"total_reserved":  totalReserved,
		"slots_remaining": int64(maxSlots) - totalReserved,
	}

	// If user is logged in, add personal info
	userId := c.GetInt("id")
	if userId > 0 {
		rank := model.GetUserBetaRank(userId)
		result["reserved"] = rank > 0
		result["rank"] = rank
		result["has_access"] = !betaEnabled || rank > 0 && rank <= int64(maxSlots)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// FarmBetaReserve allows a logged-in user to reserve a beta slot
func FarmBetaReserve(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}

	if !common.FarmBetaEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "内测预约未开启"})
		return
	}

	// Check if already reserved
	existing, err := model.GetFarmBetaReservation(userId)
	if err == nil && existing != nil {
		rank := model.GetUserBetaRank(userId)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "你已经预约过了",
			"data": gin.H{
				"rank": rank,
			},
		})
		return
	}

	// Check if slots are full
	count, _ := model.CountFarmBetaReservations()
	if count >= int64(common.FarmBetaMaxSlots) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "预约名额已满"})
		return
	}

	// Create reservation
	err = model.CreateFarmBetaReservation(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "预约失败，请稍后再试"})
		return
	}

	rank := model.GetUserBetaRank(userId)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "预约成功！",
		"data": gin.H{
			"rank": rank,
		},
	})
}

// FarmBetaAcceptAgreement allows a user to accept the beta agreement
func FarmBetaAcceptAgreement(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}

	if !model.HasFarmBetaAccess(userId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你没有内测资格"})
		return
	}

	err := model.AcceptBetaAgreement(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "操作失败，请稍后再试"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已同意内测协议",
	})
}

// CheckFarmBetaAccess is a middleware that gates farm access based on beta status
func CheckFarmBetaAccess() func(c *gin.Context) {
	return func(c *gin.Context) {
		if !common.FarmBetaEnabled {
			// Beta not enabled, allow all access
			c.Next()
			return
		}

		userId := c.GetInt("id")
		userRole := c.GetInt("role")

		// Admin bypass check
		if common.FarmBetaAdminBypass && userRole >= common.RoleAdminUser {
			c.Next()
			return
		}

		// Check if farm has opened (countdown passed)
		common.OptionMapRWMutex.RLock()
		countdownDate := common.OptionMap["FarmCountdownDate"]
		common.OptionMapRWMutex.RUnlock()

		if countdownDate != "" {
			start, err := time.Parse(time.RFC3339, countdownDate)
			if err == nil {
				if time.Now().Before(start) {
					// Farm hasn't opened yet
					c.JSON(http.StatusOK, gin.H{
						"success": false,
						"message": "农场内测尚未开启，请等待倒计时结束",
						"code":    "BETA_NOT_STARTED",
					})
					c.Abort()
					return
				}

				// Check if beta period has expired (start + duration)
				betaEnd := start.Add(time.Duration(common.FarmBetaDurationDays) * 24 * time.Hour)
				if time.Now().After(betaEnd) {
					// Beta expired — trigger one-time cleanup in background
					betaCleanupOnce.Do(func() {
						go func() {
							common.SysLog("Farm beta expired, starting automatic data cleanup...")
							userCount, totalReclaimed, err := model.CleanupAllBetaFarmData()
							if err != nil {
								common.SysError(fmt.Sprintf("Farm beta cleanup error: %v", err))
							} else {
								common.SysLog(fmt.Sprintf("Farm beta cleanup done: %d users, reclaimed %d quota", userCount, totalReclaimed))
							}
						}()
					})
					c.JSON(http.StatusOK, gin.H{
						"success": false,
						"message": "内测已结束，所有内测数据已清除。感谢您的参与！",
						"code":    "BETA_EXPIRED",
					})
					c.Abort()
					return
				}
			}
		}

		// Farm is open, check if user has beta access
		// 路径1: 通过预约获得资格
		_, err := model.GetFarmBetaReservation(userId)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 没有预约记录，检查是否通过申请获得资格
				if model.HasApprovedBetaApplication(userId) {
					// 申请已通过，自动创建预约记录（确保后续流程一致）
					_ = model.GrantBetaAccessViaApplication(userId)
					c.Next()
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "你没有内测资格，无法访问农场",
					"code":    "BETA_NO_ACCESS",
				})
				c.Abort()
				return
			}
		}

		// Check if within max slots
		if !model.HasFarmBetaAccess(userId) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "你的预约排名超出内测名额，暂时无法访问",
				"code":    "BETA_NO_ACCESS",
			})
			c.Abort()
			return
		}

		// Check if user has accepted the beta agreement
		if !model.HasAcceptedBetaAgreement(userId) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "请先阅读并同意内测协议",
				"code":    "BETA_AGREEMENT_REQUIRED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ========== 用户端：内测资格申请 ==========

const (
	betaAppMinReasonLen  = 10
	betaAppMaxReasonLen  = 300
	betaAppMaxRetries    = 3
	betaAppRetryCooldown = 24 * 3600 // 被拒后24小时才可重新申请
)

// FarmBetaApplicationStatus 获取当前用户的申请状态
func FarmBetaApplicationStatus(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}

	hasAccess := model.HasFarmBetaAccess(userId) || model.HasApprovedBetaApplication(userId)
	if hasAccess {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"has_access": true,
				"app_status": "approved",
				"can_apply":  false,
			},
		})
		return
	}

	app, err := model.GetLatestBetaApplication(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"has_access": false,
				"app_status": "not_applied",
				"can_apply":  true,
			},
		})
		return
	}

	result := gin.H{
		"has_access":        false,
		"app_status":        app.Status,
		"submitted_at":      app.SubmittedAt,
		"review_note":       app.ReviewNote,
		"reviewed_at":       app.ReviewedAt,
		"linuxdo_profile":   app.LinuxdoProfile,
		"application_round": app.ApplicationRound,
	}

	switch app.Status {
	case "pending":
		result["can_apply"] = false
	case "rejected":
		canRetry := app.ApplicationRound < betaAppMaxRetries
		if canRetry && app.ReviewedAt > 0 {
			elapsed := time.Now().Unix() - app.ReviewedAt
			if elapsed < betaAppRetryCooldown {
				canRetry = false
				result["retry_after"] = app.ReviewedAt + betaAppRetryCooldown
			}
		}
		result["can_apply"] = canRetry
	case "approved":
		result["has_access"] = true
		result["can_apply"] = false
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// FarmBetaApply 提交内测资格申请
func FarmBetaApply(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}

	if model.HasFarmBetaAccess(userId) || model.HasApprovedBetaApplication(userId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你已拥有内测资格"})
		return
	}

	var req struct {
		Reason         string `json:"reason"`
		LinuxdoProfile string `json:"linuxdo_profile"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	req.Reason = strings.TrimSpace(req.Reason)
	req.LinuxdoProfile = strings.TrimSpace(req.LinuxdoProfile)

	// 校验理由长度
	reasonRunes := []rune(req.Reason)
	if len(reasonRunes) < betaAppMinReasonLen {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("申请理由不能少于%d个字", betaAppMinReasonLen)})
		return
	}
	if len(reasonRunes) > betaAppMaxReasonLen {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("申请理由不能超过%d个字", betaAppMaxReasonLen)})
		return
	}

	// 校验 LinuxDo 链接格式
	if req.LinuxdoProfile != "" {
		if !strings.HasPrefix(req.LinuxdoProfile, "https://linux.do/") && !strings.HasPrefix(req.LinuxdoProfile, "https://www.linux.do/") {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "LinuxDo 链接格式不正确，请填写完整的个人主页链接"})
			return
		}
	}

	// 检查每日名额
	dailyQuota := model.GetBetaAIDailyQuota()
	if dailyQuota > 0 {
		todayApproved := model.CountTodayApprovedBetaApplications()
		if todayApproved >= int64(dailyQuota) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "今日内测名额已满，请明天再来申请"})
			return
		}
	}

	// 检查是否有未处理的申请
	existing, err := model.GetLatestBetaApplication(userId)
	applicationRound := 1
	if err == nil {
		if existing.Status == "pending" {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "你已有一份申请正在审核中，请耐心等待"})
			return
		}
		if existing.Status == "rejected" {
			if existing.ApplicationRound >= betaAppMaxRetries {
				c.JSON(http.StatusOK, gin.H{"success": false, "message": "已达到最大申请次数"})
				return
			}
			if existing.ReviewedAt > 0 {
				elapsed := time.Now().Unix() - existing.ReviewedAt
				if elapsed < betaAppRetryCooldown {
					c.JSON(http.StatusOK, gin.H{"success": false, "message": "申请被拒绝后需等待24小时才能重新申请"})
					return
				}
			}
			applicationRound = existing.ApplicationRound + 1
		}
	}

	app := &model.FarmBetaApplication{
		UserId:           userId,
		Reason:           req.Reason,
		LinuxdoProfile:   req.LinuxdoProfile,
		Status:           "pending",
		ApplicationRound: applicationRound,
	}
	if err := model.CreateBetaApplication(app); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "提交失败，请稍后重试"})
		return
	}

	// 异步触发 AI 审核（如果已开启）
	if model.IsBetaAIReviewEnabled() {
		go ExecuteAIReview(app)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "申请已提交，请等待审核结果",
	})
}

// ========== 管理员端：内测资格申请审核 ==========

// AdminBetaApplicationList 管理员获取申请列表
func AdminBetaApplicationList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	status := c.Query("status") // pending / approved / rejected / ""

	apps, total, err := model.GetBetaApplicationList(page, pageSize, status)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "加载失败"})
		return
	}

	var list []gin.H
	for _, app := range apps {
		// 获取用户信息
		user, _ := model.GetUserById(app.UserId, false)
		username := ""
		displayName := ""
		if user != nil {
			username = user.Username
			displayName = user.DisplayName
		}

		notifyStatus := "unavailable"
		if app.LinuxdoProfile != "" {
			notifyStatus = "available"
		}

		list = append(list, gin.H{
			"id":                app.Id,
			"user_id":           app.UserId,
			"username":          username,
			"display_name":      displayName,
			"reason":            app.Reason,
			"linuxdo_profile":   app.LinuxdoProfile,
			"notify_status":     notifyStatus,
			"status":            app.Status,
			"submitted_at":      app.SubmittedAt,
			"reviewed_at":       app.ReviewedAt,
			"reviewed_by":       app.ReviewedBy,
			"review_note":       app.ReviewNote,
			"application_round": app.ApplicationRound,
			"ai_decision":       app.AiDecision,
			"ai_confidence":     app.AiConfidence,
			"ai_summary":        app.AiSummary,
			"ai_review_log_id":  app.AiReviewLogId,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"list":  list,
			"total": total,
			"page":  page,
		},
	})
}

// AdminBetaApplicationDetail 管理员获取申请详情
func AdminBetaApplicationDetail(c *gin.Context) {
	appId, _ := strconv.Atoi(c.Query("id"))
	if appId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	app, err := model.GetBetaApplicationById(appId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "申请不存在"})
		return
	}

	user, _ := model.GetUserById(app.UserId, false)
	username := ""
	displayName := ""
	email := ""
	if user != nil {
		username = user.Username
		displayName = user.DisplayName
		email = user.Email
	}

	hasExistingAccess := model.HasFarmBetaAccess(app.UserId)

	// 获取历史申请
	history, _ := model.GetUserBetaApplicationHistory(app.UserId)
	var historyList []gin.H
	for _, h := range history {
		historyList = append(historyList, gin.H{
			"id":           h.Id,
			"reason":       h.Reason,
			"status":       h.Status,
			"submitted_at": h.SubmittedAt,
			"reviewed_at":  h.ReviewedAt,
			"review_note":  h.ReviewNote,
			"round":        h.ApplicationRound,
		})
	}

	notifyStatus := "unavailable"
	notifyMsg := "未填写 LinuxDo 链接，不做通知"
	if app.LinuxdoProfile != "" {
		notifyStatus = "available"
		notifyMsg = "可手动私信通知"
	}

	// AI 审核日志
	var aiLogs []gin.H
	if app.AiReviewLogId > 0 {
		logs, _ := model.GetAIReviewLogsByApplicationId(app.Id)
		for _, l := range logs {
			aiLogs = append(aiLogs, gin.H{
				"id":            l.Id,
				"ai_decision":   l.AiDecision,
				"ai_confidence": l.AiConfidence,
				"ai_score":      l.AiScore,
				"ai_summary":    l.AiSummary,
				"ai_reasons":    l.AiReasons,
				"final_action":  l.FinalAction,
				"model_name":    l.ModelName,
				"status":        l.Status,
				"error_message": l.ErrorMessage,
				"prompt_version": l.PromptVersion,
				"created_at":    l.CreatedAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":                  app.Id,
			"user_id":             app.UserId,
			"username":            username,
			"display_name":        displayName,
			"email":               email,
			"reason":              app.Reason,
			"linuxdo_profile":     app.LinuxdoProfile,
			"notify_status":       notifyStatus,
			"notify_message":      notifyMsg,
			"status":              app.Status,
			"submitted_at":        app.SubmittedAt,
			"reviewed_at":         app.ReviewedAt,
			"reviewed_by":         app.ReviewedBy,
			"review_note":         app.ReviewNote,
			"application_round":   app.ApplicationRound,
			"has_existing_access": hasExistingAccess,
			"history":             historyList,
			"ai_decision":         app.AiDecision,
			"ai_confidence":       app.AiConfidence,
			"ai_summary":          app.AiSummary,
			"ai_review_log_id":    app.AiReviewLogId,
			"ai_logs":             aiLogs,
		},
	})
}

// AdminBetaApplicationApprove 管理员审核通过
func AdminBetaApplicationApprove(c *gin.Context) {
	adminId := c.GetInt("id")
	var req struct {
		AppId      int    `json:"app_id"`
		ReviewNote string `json:"review_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.AppId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	app, err := model.GetBetaApplicationById(req.AppId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "申请不存在"})
		return
	}

	if app.Status == "approved" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该申请已通过，无需重复操作"})
		return
	}

	// 更新申请状态
	now := time.Now().Unix()
	_ = model.UpdateBetaApplicationFields(req.AppId, map[string]interface{}{
		"status":      "approved",
		"reviewed_at": now,
		"reviewed_by": adminId,
		"review_note": strings.TrimSpace(req.ReviewNote),
	})

	// 真正发放资格（幂等）
	if err := model.GrantBetaAccessViaApplication(app.UserId); err != nil {
		common.SysError(fmt.Sprintf("GrantBetaAccess failed for user %d: %v", app.UserId, err))
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "资格发放失败，请重试"})
		return
	}

	common.SysLog(fmt.Sprintf("Admin %d approved beta application #%d for user %d", adminId, req.AppId, app.UserId))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "审核通过，已发放内测资格",
	})
}

// AdminBetaApplicationReject 管理员审核拒绝
func AdminBetaApplicationReject(c *gin.Context) {
	adminId := c.GetInt("id")
	var req struct {
		AppId      int    `json:"app_id"`
		ReviewNote string `json:"review_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.AppId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	app, err := model.GetBetaApplicationById(req.AppId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "申请不存在"})
		return
	}

	if app.Status != "pending" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该申请已处理，无法重复操作"})
		return
	}

	now := time.Now().Unix()
	_ = model.UpdateBetaApplicationFields(req.AppId, map[string]interface{}{
		"status":      "rejected",
		"reviewed_at": now,
		"reviewed_by": adminId,
		"review_note": strings.TrimSpace(req.ReviewNote),
	})

	common.SysLog(fmt.Sprintf("Admin %d rejected beta application #%d for user %d", adminId, req.AppId, app.UserId))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已拒绝该申请",
	})
}
