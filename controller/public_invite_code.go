package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetPublicInviteCodes returns all publicly shared invitation codes (no auth required)
func GetPublicInviteCodes(c *gin.Context) {
	// Refresh statuses before returning
	_ = model.RefreshPublicInviteCodeStatuses()

	codes, err := model.GetPublicInviteCodes()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	// Mask the code key for non-used codes to prevent scraping — only show first 8 chars
	// Actually, the purpose is for users to copy and use, so show full code
	c.JSON(http.StatusOK, gin.H{"success": true, "data": codes})
}

// SharePublicInviteCode allows a logged-in user to share their invitation code publicly
func SharePublicInviteCode(c *gin.Context) {
	userId := c.GetInt("id")
	username := c.GetString("username")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}

	type ShareRequest struct {
		CodeId int `json:"code_id"`
	}
	var req ShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	// Find the redemption code - must belong to this user and be a registration code
	var redemption model.Redemption
	result := model.DB.Where("id = ? AND user_id = ? AND purpose = ?",
		req.CodeId, userId, common.RedemptionPurposeRegistration).First(&redemption)
	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未找到该邀请码或非本人所有"})
		return
	}

	// Check if code is already used or disabled
	if redemption.Status != common.RedemptionCodeStatusEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该邀请码已被使用或已失效"})
		return
	}

	// Check if expired
	now := common.GetTimestamp()
	if redemption.ExpiredTime > 0 && now >= redemption.ExpiredTime {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该邀请码已过期"})
		return
	}

	// Check if already shared
	already, err := model.IsCodeAlreadyShared(redemption.Key)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	if already {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该邀请码已被分享"})
		return
	}

	// Limit per user: max 10 active shared codes
	count, err := model.CountUserPublicInviteCodes(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	if count >= 10 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "最多分享10个邀请码"})
		return
	}

	pub := &model.PublicInviteCode{
		UserId:      userId,
		Username:    username,
		Code:        redemption.Key,
		CodeId:      redemption.Id,
		Status:      1,
		ExpiredTime: redemption.ExpiredTime,
	}
	if err := model.CreatePublicInviteCode(pub); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "分享失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "分享成功"})
}

// DeletePublicInviteCode allows a user to remove their shared code
func DeletePublicInviteCode(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.DeletePublicInviteCode(id, userId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已删除"})
}

// GetMyShareableInviteCodes returns the user's invitation codes that can be shared
func GetMyShareableInviteCodes(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}

	var codes []*model.Redemption
	now := common.GetTimestamp()
	err := model.DB.Where("user_id = ? AND purpose = ? AND status = ? AND (expired_time = 0 OR expired_time > ?)",
		userId, common.RedemptionPurposeRegistration, common.RedemptionCodeStatusEnabled, now).
		Order("id desc").Find(&codes).Error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": codes})
}
