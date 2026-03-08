package controller

import (
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

	result := gin.H{
		"beta_enabled":   betaEnabled,
		"farm_open":      farmOpen,
		"max_slots":      maxSlots,
		"total_reserved": totalReserved,
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
			t, err := time.Parse(time.RFC3339, countdownDate)
			if err == nil && time.Now().Before(t) {
				// Farm hasn't opened yet
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "农场内测尚未开启，请等待倒计时结束",
					"code":    "BETA_NOT_STARTED",
				})
				c.Abort()
				return
			}
		}

		// Farm is open, check if user has beta access
		_, err := model.GetFarmBetaReservation(userId)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
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
