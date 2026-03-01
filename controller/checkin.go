package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

// GetCheckinStatus 获取用户签到状态和历史记录
func GetCheckinStatus(c *gin.Context) {
	setting := operation_setting.GetCheckinSetting()
	if !setting.Enabled {
		common.ApiErrorMsg(c, "签到功能未启用")
		return
	}
	userId := c.GetInt("id")
	// 获取月份参数，默认为当前月份
	month := c.DefaultQuery("month", time.Now().Format("2006-01"))

	stats, err := model.GetUserCheckinStats(userId, month)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":   setting.Enabled,
			"min_quota": setting.MinQuota,
			"max_quota": setting.MaxQuota,
			"stats":     stats,
		},
	})
}

// GetCheckinLeaderboard 获取签到排行榜
func GetCheckinLeaderboard(c *gin.Context) {
	// 从 OptionMap 读取管理员配置的排行榜上限
	limit := 100
	if val, ok := common.OptionMap["CheckinLeaderboardLimit"]; ok && val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			limit = n
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 50

	leaderboard, total, err := model.GetCheckinLeaderboard(limit, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "获取排行榜失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    leaderboard,
		"total":   total,
		"limit":   limit,
	})
}

// DoCheckin 执行用户签到
func DoCheckin(c *gin.Context) {
	setting := operation_setting.GetCheckinSetting()
	if !setting.Enabled {
		common.ApiErrorMsg(c, "签到功能未启用")
		return
	}

	userId := c.GetInt("id")

	checkin, err := model.UserCheckin(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	model.RecordLog(userId, model.LogTypeSystem, fmt.Sprintf("用户签到，获得额度 %s", logger.LogQuota(checkin.QuotaAwarded)))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "签到成功",
		"data": gin.H{
			"quota_awarded": checkin.QuotaAwarded,
			"checkin_date":  checkin.CheckinDate},
	})
}
