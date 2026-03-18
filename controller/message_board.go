package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

var validCategories = map[string]bool{
	"bug": true, "suggestion": true, "feedback": true, "other": true,
}

var validStatuses = map[string]bool{
	"pending": true, "viewed": true, "processing": true, "resolved": true, "rejected": true,
}

// ========== 用户端接口 ==========

// SubmitFeedback 提交留言
func SubmitFeedback(c *gin.Context) {
	userId := c.GetInt("id")
	username, _ := c.Get("username")

	// 防刷：1分钟内最多1条
	if model.CountRecentMessageBoardPosts(userId, 60) > 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "提交过于频繁，请1分钟后再试"})
		return
	}

	var req struct {
		Title       string `json:"title"`
		Content     string `json:"content"`
		Category    string `json:"category"`
		ContactInfo string `json:"contact_info"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	// 校验
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "标题不能为空"})
		return
	}
	if len([]rune(req.Title)) > 128 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "标题最多128个字符"})
		return
	}
	if req.Content == "" || len([]rune(req.Content)) < 10 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "内容不能为空，且至少10个字符"})
		return
	}
	if !validCategories[req.Category] {
		req.Category = "other"
	}

	usernameStr := ""
	if username != nil {
		usernameStr = username.(string)
	}

	post := &model.MessageBoardPost{
		UserId:      userId,
		Username:    usernameStr,
		Title:       req.Title,
		Content:     req.Content,
		Category:    req.Category,
		ContactInfo: strings.TrimSpace(req.ContactInfo),
		IsPublic:    req.IsPublic,
		Status:      "pending",
	}
	if err := model.CreateMessageBoardPost(post); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "提交失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "提交成功"})
}

// GetMyFeedbacks 获取我的留言列表
func GetMyFeedbacks(c *gin.Context) {
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	posts, total, err := model.GetMessageBoardPostsByUserId(userId, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": posts, "total": total})
}

// GetMyFeedbackDetail 获取我的留言详情
func GetMyFeedbackDetail(c *gin.Context) {
	userId := c.GetInt("id")
	id, _ := strconv.Atoi(c.Param("id"))
	post, err := model.GetMessageBoardPostById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "留言不存在"})
		return
	}
	if post.UserId != userId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权查看"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": post})
}

// GetPublicFeedbacks 获取公开留言列表
func GetPublicFeedbacks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := c.Query("category")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	posts, total, err := model.GetPublicMessageBoardPosts(category, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取失败"})
		return
	}
	// 隐藏敏感字段
	type publicPost struct {
		Id         int    `json:"id"`
		Title      string `json:"title"`
		Content    string `json:"content"`
		Category   string `json:"category"`
		Status     string `json:"status"`
		AdminReply string `json:"admin_reply"`
		CreatedAt  int64  `json:"created_at"`
	}
	var result []publicPost
	for _, p := range posts {
		result = append(result, publicPost{
			Id: p.Id, Title: p.Title, Content: p.Content,
			Category: p.Category, Status: p.Status,
			AdminReply: p.AdminReply, CreatedAt: p.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result, "total": total})
}

// ========== 管理员接口 ==========

// AdminGetAllFeedbacks 管理员获取所有留言
func AdminGetAllFeedbacks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	category := c.Query("category")
	keyword := c.Query("keyword")
	userIdFilter, _ := strconv.Atoi(c.Query("user_id"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	posts, total, err := model.GetAllMessageBoardPosts(status, category, keyword, userIdFilter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": posts, "total": total})
}

// AdminGetFeedbackDetail 管理员获取留言详情
func AdminGetFeedbackDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	post, err := model.GetMessageBoardPostById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "留言不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": post})
}

// AdminUpdateFeedbackStatus 管理员更新留言状态
func AdminUpdateFeedbackStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	adminId := c.GetInt("id")

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !validStatuses[req.Status] {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效状态"})
		return
	}
	if err := model.UpdateMessageBoardPostStatus(id, req.Status, adminId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "状态已更新"})
}

// AdminReplyFeedback 管理员回复留言
func AdminReplyFeedback(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Reply string `json:"reply"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.UpdateMessageBoardPostAdminReply(id, strings.TrimSpace(req.Reply)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "回复成功"})
}

// AdminNoteFeedback 管理员添加备注
func AdminNoteFeedback(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.UpdateMessageBoardPostAdminNote(id, strings.TrimSpace(req.Note)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "备注已更新"})
}

// AdminSetFeedbackPublic 管理员设置留言是否公开
func AdminSetFeedbackPublic(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		IsPublic bool `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.UpdateMessageBoardPostIsPublic(id, req.IsPublic); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已更新"})
}
