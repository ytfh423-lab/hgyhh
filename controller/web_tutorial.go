package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const tutorialCurrentVersion = 1

// WebFarmTutorialState 获取当前用户教程状态
func WebFarmTutorialState(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	state, err := model.GetTutorialState(tgId)
	if err != nil {
		// 首次用户，没有记录
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"has_seen":      false,
				"current_step":  0,
				"completed":     false,
				"skipped":       false,
				"version":       tutorialCurrentVersion,
				"needs_tutorial": true,
			},
		})
		return
	}

	// 检查版本升级：如果服务端版本更高，提示需要重新引导
	needsTutorial := state.Completed == 0 && state.Skipped == 0
	needsUpgrade := state.Version < tutorialCurrentVersion && state.Completed == 1

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"has_seen":       true,
			"current_step":   state.CurrentStep,
			"completed":      state.Completed == 1,
			"skipped":        state.Skipped == 1,
			"version":        state.Version,
			"completed_at":   state.CompletedAt,
			"needs_tutorial":  needsTutorial,
			"needs_upgrade":   needsUpgrade,
			"latest_version":  tutorialCurrentVersion,
		},
	})
}

// WebFarmTutorialUpdate 更新教程进度
func WebFarmTutorialUpdate(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		Step int `json:"step"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	// 确保记录存在
	state, err := model.GetTutorialState(tgId)
	if err != nil {
		state, err = model.CreateTutorialState(tgId)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建教程状态失败"})
			return
		}
	}
	_ = state

	if err := model.UpdateTutorialStep(tgId, req.Step); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"current_step": req.Step}})
}

// WebFarmTutorialComplete 标记教程完成
func WebFarmTutorialComplete(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	// 确保记录存在
	if _, err := model.GetTutorialState(tgId); err != nil {
		if _, err := model.CreateTutorialState(tgId); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建教程状态失败"})
			return
		}
	}

	if err := model.CompleteTutorial(tgId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "教程已完成"})
}

// WebFarmTutorialSkip 跳过教程
func WebFarmTutorialSkip(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	if _, err := model.GetTutorialState(tgId); err != nil {
		if _, err := model.CreateTutorialState(tgId); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建教程状态失败"})
			return
		}
	}

	if err := model.SkipTutorial(tgId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已跳过教程"})
}

// WebFarmTutorialRestart 重启教程
func WebFarmTutorialRestart(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	if _, err := model.GetTutorialState(tgId); err != nil {
		if _, err := model.CreateTutorialState(tgId); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建教程状态失败"})
			return
		}
	}

	if err := model.RestartTutorial(tgId, tutorialCurrentVersion); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "重启失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "教程已重置", "data": gin.H{"current_step": 0}})
}
