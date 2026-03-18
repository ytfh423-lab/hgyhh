package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetPublicInviteCodes returns publicly shared invitation codes with pagination (no auth required)
func GetPublicInviteCodes(c *gin.Context) {
	// Refresh statuses before returning
	_ = model.RefreshPublicInviteCodeStatuses()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	codes, total, err := model.GetPublicInviteCodesPaginated(page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      codes,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
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

// GetMyShareableInviteCodes returns all user's invitation codes with status and shared info
func GetMyShareableInviteCodes(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}

	var codes []*model.Redemption
	err := model.DB.Where("user_id = ? AND purpose = ?",
		userId, common.RedemptionPurposeRegistration).
		Order("id desc").Find(&codes).Error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	now := common.GetTimestamp()

	type CodeInfo struct {
		Id          int    `json:"id"`
		Key         string `json:"key"`
		Status      int    `json:"status"`
		ExpiredTime int64  `json:"expired_time"`
		UsedUserId  int    `json:"used_user_id"`
		StatusLabel string `json:"status_label"` // 可用/已使用/已过期/已分享
		Shareable   bool   `json:"shareable"`
	}

	var result []CodeInfo
	for _, code := range codes {
		info := CodeInfo{
			Id:          code.Id,
			Key:         code.Key,
			Status:      code.Status,
			ExpiredTime: code.ExpiredTime,
			UsedUserId:  code.UsedUserId,
		}

		if code.Status == common.RedemptionCodeStatusUsed {
			info.StatusLabel = "已使用"
			info.Shareable = false
		} else if code.Status == common.RedemptionCodeStatusDisabled {
			info.StatusLabel = "已禁用"
			info.Shareable = false
		} else if code.ExpiredTime > 0 && now >= code.ExpiredTime {
			info.StatusLabel = "已过期"
			info.Shareable = false
		} else {
			// Check if already shared
			shared, _ := model.IsCodeAlreadyShared(code.Key)
			if shared {
				info.StatusLabel = "已分享"
				info.Shareable = false
			} else {
				info.StatusLabel = "可用"
				info.Shareable = true
			}
		}
		result = append(result, info)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
