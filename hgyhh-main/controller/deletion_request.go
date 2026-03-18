package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetAllDeletionRequests(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	statusStr := c.Query("status")

	var statusPtr *int
	if statusStr != "" {
		s, err := strconv.Atoi(statusStr)
		if err == nil {
			statusPtr = &s
		}
	}

	requests, total, err := model.GetDeletionRequests(page, pageSize, statusPtr)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    requests,
		"total":   total,
	})
}

func ApproveDeletionRequest(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}
	adminId := c.GetInt("id")

	var req struct {
		AdminRemark string `json:"admin_remark"`
	}
	_ = common.DecodeJson(c.Request.Body, &req)

	err = model.ApproveDeletionRequest(id, adminId, req.AdminRemark)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "注销申请已通过，用户已删除",
	})
}

func RejectDeletionRequest(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}
	adminId := c.GetInt("id")

	var req struct {
		AdminRemark string `json:"admin_remark"`
	}
	_ = common.DecodeJson(c.Request.Body, &req)

	err = model.RejectDeletionRequest(id, adminId, req.AdminRemark)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "注销申请已拒绝",
	})
}
