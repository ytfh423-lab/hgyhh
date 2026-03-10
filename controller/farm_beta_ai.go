package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ========== 管理员：AI 审核配置 ==========

// AdminGetBetaAIConfig 获取 AI 审核配置（不返回完整 API Key）
func AdminGetBetaAIConfig(c *gin.Context) {
	config, err := model.GetFarmBetaAIConfig()
	if err != nil {
		// 返回默认配置
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"enabled":                 false,
				"api_base_url":            "https://codex.hyw.me",
				"model_name":              "gpt5.2",
				"api_key_configured":      false,
				"system_prompt":           model.DefaultBetaAISystemPrompt,
				"auto_approve_confidence": 85,
				"auto_reject_confidence":  80,
				"allow_auto_apply_result": true,
				"log_raw_response":        true,
				"timeout_ms":              30000,
				"json_mode":               true,
				"prompt_version":          1,
				"daily_quota":             0,
				"updated_by":              0,
				"created_at":              0,
				"updated_at":              0,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":                 config.Enabled,
			"api_base_url":            config.ApiBaseUrl,
			"model_name":              config.ModelName,
			"api_key_configured":      config.ApiKeyEncrypted != "",
			"system_prompt":           config.SystemPrompt,
			"auto_approve_confidence": config.AutoApproveConfidence,
			"auto_reject_confidence":  config.AutoRejectConfidence,
			"allow_auto_apply_result": config.AllowAutoApplyResult,
			"log_raw_response":        config.LogRawResponse,
			"timeout_ms":              config.TimeoutMs,
			"json_mode":               config.JsonMode,
			"prompt_version":          config.PromptVersion,
			"daily_quota":             config.DailyQuota,
			"updated_by":              config.UpdatedBy,
			"created_at":              config.CreatedAt,
			"updated_at":              config.UpdatedAt,
		},
	})
}

// AdminSaveBetaAIConfig 保存 AI 审核配置
func AdminSaveBetaAIConfig(c *gin.Context) {
	adminId := c.GetInt("id")

	var req struct {
		Enabled               bool   `json:"enabled"`
		ApiBaseUrl            string `json:"api_base_url"`
		ModelName             string `json:"model_name"`
		ApiKey                string `json:"api_key"`
		SystemPrompt          string `json:"system_prompt"`
		AutoApproveConfidence int    `json:"auto_approve_confidence"`
		AutoRejectConfidence  int    `json:"auto_reject_confidence"`
		AllowAutoApplyResult  bool   `json:"allow_auto_apply_result"`
		LogRawResponse        bool   `json:"log_raw_response"`
		TimeoutMs             int    `json:"timeout_ms"`
		JsonMode              bool   `json:"json_mode"`
		DailyQuota            int    `json:"daily_quota"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	req.ApiBaseUrl = strings.TrimSpace(req.ApiBaseUrl)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.SystemPrompt = strings.TrimSpace(req.SystemPrompt)

	if req.Enabled {
		if req.ApiBaseUrl == "" || req.ModelName == "" {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "开启 AI 审核时必须配置 API 地址和模型名称"})
			return
		}
		if req.SystemPrompt == "" {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "开启 AI 审核时必须配置前置提示词"})
			return
		}
	}

	if req.AutoApproveConfidence < 0 || req.AutoApproveConfidence > 100 {
		req.AutoApproveConfidence = 85
	}
	if req.AutoRejectConfidence < 0 || req.AutoRejectConfidence > 100 {
		req.AutoRejectConfidence = 80
	}
	if req.TimeoutMs < 5000 {
		req.TimeoutMs = 5000
	}
	if req.TimeoutMs > 120000 {
		req.TimeoutMs = 120000
	}
	if req.DailyQuota < 0 {
		req.DailyQuota = 0
	}

	// 获取现有配置
	existing, _ := model.GetFarmBetaAIConfig()

	config := &model.FarmBetaAIConfig{
		Enabled:               req.Enabled,
		ApiBaseUrl:            req.ApiBaseUrl,
		ModelName:             req.ModelName,
		SystemPrompt:          req.SystemPrompt,
		AutoApproveConfidence: req.AutoApproveConfidence,
		AutoRejectConfidence:  req.AutoRejectConfidence,
		AllowAutoApplyResult:  req.AllowAutoApplyResult,
		LogRawResponse:        req.LogRawResponse,
		TimeoutMs:             req.TimeoutMs,
		JsonMode:              req.JsonMode,
		DailyQuota:            req.DailyQuota,
		UpdatedBy:             adminId,
	}

	// 处理 API Key
	if req.ApiKey != "" {
		encrypted, err := model.EncryptAPIKey(req.ApiKey)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "API Key 加密失败"})
			return
		}
		config.ApiKeyEncrypted = encrypted
	} else if existing != nil {
		config.ApiKeyEncrypted = existing.ApiKeyEncrypted
	}

	// 检查是否需要检查 API Key
	if req.Enabled && config.ApiKeyEncrypted == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "开启 AI 审核时必须配置 API Key"})
		return
	}

	// 处理 prompt 版本号
	if existing != nil {
		config.PromptVersion = existing.PromptVersion
		if existing.SystemPrompt != req.SystemPrompt {
			config.PromptVersion = existing.PromptVersion + 1
		}
	} else {
		config.PromptVersion = 1
	}

	if err := model.SaveFarmBetaAIConfig(config); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "保存失败"})
		return
	}

	common.SysLog(fmt.Sprintf("Admin %d updated beta AI review config, enabled=%v, prompt_version=%d", adminId, req.Enabled, config.PromptVersion))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置保存成功",
		"data": gin.H{
			"prompt_version": config.PromptVersion,
		},
	})
}

// AdminTestBetaAIConnection 测试 AI 连接
func AdminTestBetaAIConnection(c *gin.Context) {
	var req struct {
		ApiBaseUrl string `json:"api_base_url"`
		ModelName  string `json:"model_name"`
		ApiKey     string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	apiKey := strings.TrimSpace(req.ApiKey)
	apiBase := strings.TrimSpace(req.ApiBaseUrl)
	modelName := strings.TrimSpace(req.ModelName)

	// 如果没传 key，尝试用已保存的
	if apiKey == "" {
		config, err := model.GetFarmBetaAIConfig()
		if err != nil || config.ApiKeyEncrypted == "" {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "未配置 API Key"})
			return
		}
		apiKey, err = model.DecryptAPIKey(config.ApiKeyEncrypted)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "API Key 解密失败"})
			return
		}
		if apiBase == "" {
			apiBase = config.ApiBaseUrl
		}
		if modelName == "" {
			modelName = config.ModelName
		}
	}

	if apiBase == "" || modelName == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "API 地址和模型名不能为空"})
		return
	}

	// 发送简单测试请求
	payload := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"Hi, respond with ok"}],"max_tokens":10}`, modelName)
	url := strings.TrimRight(apiBase, "/") + "/v1/chat/completions"

	client := &http.Client{Timeout: 15 * time.Second}
	httpReq, _ := http.NewRequest("POST", url, bytes.NewBufferString(payload))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	start := time.Now()
	resp, err := client.Do(httpReq)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("连接失败: %v", err), "latency_ms": latency})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		c.JSON(http.StatusOK, gin.H{
			"success":    false,
			"message":    fmt.Sprintf("API 返回错误状态码 %d", resp.StatusCode),
			"latency_ms": latency,
			"detail":     string(body),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    fmt.Sprintf("连接成功，延迟 %dms", latency),
		"latency_ms": latency,
	})
}

// AdminTestBetaAIPrompt 测试前置提示词
func AdminTestBetaAIPrompt(c *gin.Context) {
	var req struct {
		SystemPrompt string `json:"system_prompt"`
		TestData     struct {
			Reason         string `json:"reason"`
			LinuxdoProfile string `json:"linuxdo_profile"`
		} `json:"test_data"`
		ApiBaseUrl string `json:"api_base_url"`
		ModelName  string `json:"model_name"`
		ApiKey     string `json:"api_key"`
		JsonMode   bool   `json:"json_mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	if systemPrompt == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "前置提示词不能为空"})
		return
	}

	apiKey := strings.TrimSpace(req.ApiKey)
	apiBase := strings.TrimSpace(req.ApiBaseUrl)
	modelName := strings.TrimSpace(req.ModelName)

	// 如果没传完整信息，用已保存的
	config, _ := model.GetFarmBetaAIConfig()
	if apiKey == "" && config != nil {
		var err error
		apiKey, err = model.DecryptAPIKey(config.ApiKeyEncrypted)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "API Key 解密失败"})
			return
		}
	}
	if apiBase == "" && config != nil {
		apiBase = config.ApiBaseUrl
	}
	if modelName == "" && config != nil {
		modelName = config.ModelName
	}

	if apiBase == "" || modelName == "" || apiKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "API 配置不完整"})
		return
	}

	// 构造模拟申请数据
	testReason := req.TestData.Reason
	if testReason == "" {
		testReason = "我非常喜欢这个农场玩法，希望能参与内测，帮助发现问题和提供改进建议。"
	}
	testProfile := req.TestData.LinuxdoProfile
	userData := buildUserDataJSON(0, "test_user", "测试用户", testReason, testProfile, false, 1, "")

	result, rawResp, err := callAIReview(apiBase, modelName, apiKey, systemPrompt, userData, req.JsonMode, 30000)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("AI 调用失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "测试完成",
		"data": gin.H{
			"ai_result":    result,
			"raw_response": rawResp,
		},
	})
}

// ========== AI 审核日志查询 ==========

// AdminGetBetaAIReviewLogs 获取 AI 审核日志列表
func AdminGetBetaAIReviewLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	logs, total, err := model.GetAIReviewLogList(page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "加载失败"})
		return
	}

	var list []gin.H
	for _, log := range logs {
		item := gin.H{
			"id":             log.Id,
			"application_id": log.ApplicationId,
			"user_id":        log.UserId,
			"model_name":     log.ModelName,
			"ai_decision":    log.AiDecision,
			"ai_confidence":  log.AiConfidence,
			"ai_score":       log.AiScore,
			"ai_summary":     log.AiSummary,
			"final_action":   log.FinalAction,
			"status":         log.Status,
			"error_message":  log.ErrorMessage,
			"prompt_version": log.PromptVersion,
			"created_at":     log.CreatedAt,
		}
		list = append(list, item)
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

// AdminGetBetaAIReviewLogDetail 获取 AI 审核日志详情
func AdminGetBetaAIReviewLogDetail(c *gin.Context) {
	logId, _ := strconv.Atoi(c.Query("id"))
	if logId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	log, err := model.GetAIReviewLogById(logId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "日志不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":                    log.Id,
			"application_id":       log.ApplicationId,
			"user_id":              log.UserId,
			"system_prompt_snapshot": log.SystemPromptSnapshot,
			"model_name":           log.ModelName,
			"api_base_url":         log.ApiBaseUrl,
			"request_payload":      log.RequestPayload,
			"ai_decision":          log.AiDecision,
			"ai_confidence":        log.AiConfidence,
			"ai_score":             log.AiScore,
			"ai_summary":           log.AiSummary,
			"ai_reasons":           log.AiReasons,
			"ai_raw_response":      log.AiRawResponse,
			"final_action":         log.FinalAction,
			"status":               log.Status,
			"error_message":        log.ErrorMessage,
			"prompt_version":       log.PromptVersion,
			"created_at":           log.CreatedAt,
		},
	})
}

// ========== AI 审核核心逻辑 ==========

// AIReviewResult AI 审核返回结构
type AIReviewResult struct {
	Decision           string   `json:"decision"`
	Confidence         float64  `json:"confidence"`
	Score              float64  `json:"score"`
	Summary            string   `json:"summary"`
	Reasons            []string `json:"reasons"`
	RiskFlags          []string `json:"risk_flags"`
	SuggestedReviewNote string  `json:"suggested_review_note"`
}

// buildUserDataJSON 构造发送给 AI 的用户申请数据
func buildUserDataJSON(userId int, username, displayName, reason, linuxdoProfile string, hasReservation bool, applicationRound int, historyStatus string) string {
	hasLinuxdo := "否"
	if linuxdoProfile != "" {
		hasLinuxdo = "是"
	}
	hasReservationStr := "否"
	if hasReservation {
		hasReservationStr = "是"
	}

	data := fmt.Sprintf(`{
  "user_id": %d,
  "username": "%s",
  "display_name": "%s",
  "reason": "%s",
  "linuxdo_profile": "%s",
  "has_linuxdo_link": "%s",
  "has_reservation": "%s",
  "application_round": %d,
  "history_status": "%s"
}`,
		userId,
		escapeJSON(username),
		escapeJSON(displayName),
		escapeJSON(reason),
		escapeJSON(linuxdoProfile),
		hasLinuxdo,
		hasReservationStr,
		applicationRound,
		historyStatus,
	)
	return data
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// callAIReview 调用 AI 审核接口
func callAIReview(apiBase, modelName, apiKey, systemPrompt, userData string, jsonMode bool, timeoutMs int) (*AIReviewResult, string, error) {
	url := strings.TrimRight(apiBase, "/") + "/v1/chat/completions"

	// 构造 messages
	var payloadBuilder strings.Builder
	payloadBuilder.WriteString(`{"model":"`)
	payloadBuilder.WriteString(escapeJSON(modelName))
	payloadBuilder.WriteString(`","messages":[{"role":"system","content":"`)
	payloadBuilder.WriteString(escapeJSON(systemPrompt))
	payloadBuilder.WriteString(`"},{"role":"user","content":"以下是待审核的申请信息：\n`)
	payloadBuilder.WriteString(escapeJSON(userData))
	payloadBuilder.WriteString(`"}],"temperature":0.3`)
	if jsonMode {
		payloadBuilder.WriteString(`,"response_format":{"type":"json_object"}`)
	}
	payloadBuilder.WriteString(`}`)

	payload := payloadBuilder.String()

	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeout < 5*time.Second {
		timeout = 5 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBufferString(payload))
	if err != nil {
		return nil, "", fmt.Errorf("构造请求失败: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取响应失败: %v", err)
	}

	rawResp := string(body)

	if resp.StatusCode != 200 {
		return nil, rawResp, fmt.Errorf("API 返回状态码 %d", resp.StatusCode)
	}

	// 解析 OpenAI 兼容响应
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := common.Unmarshal(body, &chatResp); err != nil {
		return nil, rawResp, fmt.Errorf("解析响应 JSON 失败: %v", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, rawResp, fmt.Errorf("AI 未返回有效内容")
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)

	// 尝试提取 JSON（可能被包裹在 markdown code block 中）
	jsonContent := content
	if idx := strings.Index(content, "```json"); idx >= 0 {
		start := idx + 7
		if end := strings.Index(content[start:], "```"); end >= 0 {
			jsonContent = strings.TrimSpace(content[start : start+end])
		}
	} else if idx := strings.Index(content, "```"); idx >= 0 {
		start := idx + 3
		if end := strings.Index(content[start:], "```"); end >= 0 {
			jsonContent = strings.TrimSpace(content[start : start+end])
		}
	}

	// 解析 AI 审核结果
	var result AIReviewResult
	if err := common.Unmarshal([]byte(jsonContent), &result); err != nil {
		return nil, rawResp, fmt.Errorf("解析 AI 结果 JSON 失败: %v (content: %s)", err, content[:min(len(content), 200)])
	}

	// 验证 decision 值
	switch result.Decision {
	case "approve", "reject", "manual_review":
		// valid
	default:
		result.Decision = "manual_review"
	}

	// 确保置信度在范围内
	if result.Confidence < 0 {
		result.Confidence = 0
	}
	if result.Confidence > 1 {
		result.Confidence = 1
	}

	return &result, rawResp, nil
}

// ExecuteAIReview 对单条申请执行 AI 审核
func ExecuteAIReview(app *model.FarmBetaApplication) {
	config, err := model.GetFarmBetaAIConfig()
	if err != nil || !config.Enabled {
		return
	}

	if config.ApiKeyEncrypted == "" || config.SystemPrompt == "" || config.ApiBaseUrl == "" || config.ModelName == "" {
		common.SysLog("AI review config incomplete, skipping")
		return
	}

	apiKey, err := model.DecryptAPIKey(config.ApiKeyEncrypted)
	if err != nil {
		common.SysError(fmt.Sprintf("AI review: decrypt API key failed: %v", err))
		return
	}

	// 获取用户信息
	user, _ := model.GetUserById(app.UserId, false)
	username := ""
	displayName := ""
	if user != nil {
		username = user.Username
		displayName = user.DisplayName
	}

	// 检查是否有预约记录
	hasReservation := model.HasFarmBetaAccess(app.UserId)

	// 获取历史状态
	historyStatus := ""
	if app.ApplicationRound > 1 {
		historyStatus = fmt.Sprintf("第%d次申请，之前被拒绝", app.ApplicationRound)
	}

	userData := buildUserDataJSON(app.UserId, username, displayName, app.Reason, app.LinuxdoProfile, hasReservation, app.ApplicationRound, historyStatus)

	// 创建日志记录
	reviewLog := &model.FarmBetaAIReviewLog{
		ApplicationId:        app.Id,
		UserId:               app.UserId,
		SystemPromptSnapshot: config.SystemPrompt,
		ModelName:            config.ModelName,
		ApiBaseUrl:           config.ApiBaseUrl,
		RequestPayload:       userData,
		PromptVersion:        config.PromptVersion,
		Status:               "processing",
	}

	result, rawResp, err := callAIReview(config.ApiBaseUrl, config.ModelName, apiKey, config.SystemPrompt, userData, config.JsonMode, config.TimeoutMs)

	if err != nil {
		// AI 调用失败，转人工审核
		reviewLog.Status = "error"
		reviewLog.ErrorMessage = err.Error()
		reviewLog.FinalAction = "error_manual_review"
		if config.LogRawResponse && rawResp != "" {
			reviewLog.AiRawResponse = rawResp
		}
		_ = model.CreateAIReviewLog(reviewLog)

		// 更新申请记录的 AI 字段
		_ = model.UpdateBetaApplicationFields(app.Id, map[string]interface{}{
			"ai_decision":      "error",
			"ai_review_log_id": reviewLog.Id,
		})

		common.SysError(fmt.Sprintf("AI review failed for app #%d: %v", app.Id, err))
		return
	}

	// 记录结果
	reviewLog.AiDecision = result.Decision
	reviewLog.AiConfidence = result.Confidence
	reviewLog.AiScore = result.Score
	reviewLog.AiSummary = result.Summary
	reviewLog.Status = "completed"

	// 序列化 reasons
	if len(result.Reasons) > 0 {
		reasonsBytes, _ := common.Marshal(result.Reasons)
		reviewLog.AiReasons = string(reasonsBytes)
	}

	if config.LogRawResponse {
		reviewLog.AiRawResponse = rawResp
	}

	// 决定最终动作
	confidencePercent := int(result.Confidence * 100)
	finalAction := "manual_review"

	if config.AllowAutoApplyResult {
		if result.Decision == "approve" && confidencePercent >= config.AutoApproveConfidence {
			finalAction = "auto_approved"
		} else if result.Decision == "reject" && confidencePercent >= config.AutoRejectConfidence {
			finalAction = "auto_rejected"
		}
	}

	reviewLog.FinalAction = finalAction
	_ = model.CreateAIReviewLog(reviewLog)

	// 更新申请记录的 AI 字段
	updateFields := map[string]interface{}{
		"ai_decision":      result.Decision,
		"ai_confidence":    result.Confidence,
		"ai_summary":       result.Summary,
		"ai_review_log_id": reviewLog.Id,
	}

	// 执行自动审核动作
	switch finalAction {
	case "auto_approved":
		updateFields["status"] = "approved"
		updateFields["reviewed_at"] = time.Now().Unix()
		updateFields["reviewed_by"] = 0 // AI 审核标记为 0
		updateFields["review_note"] = fmt.Sprintf("[AI自动通过] %s", result.SuggestedReviewNote)
		_ = model.UpdateBetaApplicationFields(app.Id, updateFields)

		// 发放资格
		if err := model.GrantBetaAccessViaApplication(app.UserId); err != nil {
			common.SysError(fmt.Sprintf("AI auto-approve: grant access failed for user %d: %v", app.UserId, err))
		} else {
			common.SysLog(fmt.Sprintf("AI auto-approved beta app #%d for user %d (confidence=%.2f, score=%.0f)", app.Id, app.UserId, result.Confidence, result.Score))
		}

	case "auto_rejected":
		updateFields["status"] = "rejected"
		updateFields["reviewed_at"] = time.Now().Unix()
		updateFields["reviewed_by"] = 0
		updateFields["review_note"] = fmt.Sprintf("[AI自动拒绝] %s", result.SuggestedReviewNote)
		_ = model.UpdateBetaApplicationFields(app.Id, updateFields)
		common.SysLog(fmt.Sprintf("AI auto-rejected beta app #%d for user %d (confidence=%.2f, score=%.0f)", app.Id, app.UserId, result.Confidence, result.Score))

	default:
		// 转人工审核，只更新 AI 字段，不改变 status
		_ = model.UpdateBetaApplicationFields(app.Id, updateFields)
		common.SysLog(fmt.Sprintf("AI review -> manual_review for beta app #%d (decision=%s, confidence=%.2f, score=%.0f)", app.Id, result.Decision, result.Confidence, result.Score))
	}
}
