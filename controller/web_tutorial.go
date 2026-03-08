package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// 功能教程版本号——升级后可强制重新触发教学
var featureTutorialVersions = map[string]int{
	"farm_basic":  1,
	"market":      1,
	"warehouse":   1,
	"treefarm":    1,
	"tasks":       1,
}

// WebFarmTutorialState 获取当前用户全部教程状态
func WebFarmTutorialState(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	states, _ := model.GetAllFeatureTutorialStates(tgId)

	// 构造 feature -> state map
	stateMap := make(map[string]gin.H)
	for _, s := range states {
		stateMap[s.FeatureKey] = gin.H{
			"feature_key":       s.FeatureKey,
			"current_step":      s.CurrentStep,
			"tutorial_required": s.TutorialRequired == 1,
			"tutorial_started":  s.TutorialStarted == 1,
			"tutorial_completed": s.TutorialCompleted == 1,
			"tutorial_mode":     s.TutorialMode,
			"tutorial_version":  s.TutorialVersion,
			"completed_at":      s.CompletedAt,
		}
	}

	// 检查基础教程是否存在
	_, hasBasic := stateMap["farm_basic"]
	needsBasicTutorial := !hasBasic

	// 找第一个待完成的强制教程
	var pendingForced gin.H
	pending, err := model.GetPendingForcedTutorial(tgId)
	if err == nil && pending != nil {
		pendingForced = gin.H{
			"feature_key":  pending.FeatureKey,
			"current_step": pending.CurrentStep,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"features":             stateMap,
			"needs_basic_tutorial": needsBasicTutorial,
			"pending_forced":       pendingForced,
		},
	})
}

// WebFarmTutorialUpdate 更新指定功能的教程进度
func WebFarmTutorialUpdate(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		FeatureKey string `json:"feature_key"`
		Step       int    `json:"step"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FeatureKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	ver := featureTutorialVersions[req.FeatureKey]
	if ver == 0 {
		ver = 1
	}
	if _, err := model.EnsureFeatureTutorialState(tgId, req.FeatureKey, ver); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建教程状态失败"})
		return
	}

	if err := model.UpdateFeatureTutorialStep(tgId, req.FeatureKey, req.Step); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"feature_key":  req.FeatureKey,
		"current_step": req.Step,
	}})
}

// WebFarmTutorialComplete 标记指定功能教程完成
func WebFarmTutorialComplete(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		FeatureKey string `json:"feature_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FeatureKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	ver := featureTutorialVersions[req.FeatureKey]
	if ver == 0 {
		ver = 1
	}
	if _, err := model.EnsureFeatureTutorialState(tgId, req.FeatureKey, ver); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建教程状态失败"})
		return
	}

	if err := model.CompleteFeatureTutorial(tgId, req.FeatureKey); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "教程已完成"})
}

// WebFarmTutorialRestart 以replay模式重启指定功能教程
func WebFarmTutorialRestart(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		FeatureKey string `json:"feature_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FeatureKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	ver := featureTutorialVersions[req.FeatureKey]
	if ver == 0 {
		ver = 1
	}
	if _, err := model.EnsureFeatureTutorialState(tgId, req.FeatureKey, ver); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建教程状态失败"})
		return
	}

	if err := model.RestartFeatureTutorial(tgId, req.FeatureKey); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "重启失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "教程已重置", "data": gin.H{
		"feature_key":  req.FeatureKey,
		"current_step": 0,
		"tutorial_mode": "replay",
	}})
}

// WebFarmTutorialUnlock 功能解锁时自动创建强制教程状态（由前端在检测到新功能解锁时调用）
func WebFarmTutorialUnlock(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		FeatureKey string `json:"feature_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FeatureKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	ver := featureTutorialVersions[req.FeatureKey]
	if ver == 0 {
		ver = 1
	}

	state, err := model.EnsureFeatureTutorialState(tgId, req.FeatureKey, ver)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"feature_key":       state.FeatureKey,
			"tutorial_required": state.TutorialRequired == 1,
			"tutorial_completed": state.TutorialCompleted == 1,
			"current_step":      state.CurrentStep,
		},
	})
}

// WebFarmTutorialSkip 跳过教程（仅replay模式允许）
func WebFarmTutorialSkip(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		FeatureKey string `json:"feature_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FeatureKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	state, err := model.GetFeatureTutorialState(tgId, req.FeatureKey)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "教程不存在"})
		return
	}

	// 强制模式不允许跳过
	if state.TutorialMode == "forced" && state.TutorialRequired == 1 && state.TutorialCompleted == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "强制教程不可跳过"})
		return
	}

	if err := model.CompleteFeatureTutorial(tgId, req.FeatureKey); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已退出教程"})
}
