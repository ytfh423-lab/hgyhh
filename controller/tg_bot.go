package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// ========== Telegram Webhook 数据结构 ==========

type TgUpdate struct {
	UpdateId      int              `json:"update_id"`
	Message       *TgMsg           `json:"message"`
	CallbackQuery *TgCallbackQuery `json:"callback_query"`
}

type TgMsg struct {
	MessageId int     `json:"message_id"`
	From      *TgUser `json:"from"`
	Chat      *TgChat `json:"chat"`
	Text      string  `json:"text"`
}

type TgUser struct {
	Id        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type TgChat struct {
	Id   int64  `json:"id"`
	Type string `json:"type"`
}

type TgCallbackQuery struct {
	Id      string  `json:"id"`
	From    *TgUser `json:"from"`
	Message *TgMsg  `json:"message"`
	Data    string  `json:"data"`
}

type TgInlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type TgInlineKeyboardMarkup struct {
	InlineKeyboard [][]TgInlineKeyboardButton `json:"inline_keyboard"`
}

// ========== Webhook Handler ==========

func TgBotWebhook(c *gin.Context) {
	var update TgUpdate
	if err := common.DecodeJson(c.Request.Body, &update); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	// 处理 callback_query（按钮点击）
	if update.CallbackQuery != nil {
		handleTgCallback(update.CallbackQuery)
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	if update.Message == nil || update.Message.From == nil || update.Message.From.IsBot {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	msg := update.Message
	chatId := msg.Chat.Id
	text := strings.TrimSpace(msg.Text)

	switch {
	case text == "/start":
		handleTgStart(chatId)
	case text == "/claim" || text == "/领取":
		handleTgShowMenu(msg)
	case text == "/myinfo" || text == "/我的信息":
		handleTgMyInfo(msg)
	case strings.HasPrefix(text, "/redeem ") || strings.HasPrefix(text, "/兑换 "):
		parts := strings.SplitN(text, " ", 2)
		if len(parts) == 2 {
			handleTgRedeem(msg, strings.TrimSpace(parts[1]))
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleTgStart 发送欢迎消息 + 分类按钮
func handleTgStart(chatId int64) {
	categories, err := model.GetEnabledTgBotCategories()
	if err != nil || len(categories) == 0 {
		sendTgMessage(chatId, "👋 欢迎使用 "+common.SystemName+" 机器人！\n\n暂无可领取的项目，请联系管理员。")
		return
	}

	var rows [][]TgInlineKeyboardButton
	for _, cat := range categories {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: cat.Name, CallbackData: fmt.Sprintf("claim_%d", cat.Id)},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "📊 我的信息", CallbackData: "myinfo"},
	})

	sendTgMessageWithKeyboard(chatId,
		"👋 欢迎使用 "+common.SystemName+" 机器人！\n\n请点击下方按钮领取：",
		TgInlineKeyboardMarkup{InlineKeyboard: rows})
}

// handleTgShowMenu 显示领取菜单
func handleTgShowMenu(msg *TgMsg) {
	handleTgStart(msg.Chat.Id)
}

// handleTgCallback 处理按钮点击
func handleTgCallback(cb *TgCallbackQuery) {
	chatId := cb.Message.Chat.Id

	// 应答 callback 避免按钮转圈
	answerCallbackQuery(cb.Id)

	if cb.Data == "myinfo" {
		handleTgMyInfoByUser(chatId, cb.From)
		return
	}

	if strings.HasPrefix(cb.Data, "claim_") {
		catIdStr := strings.TrimPrefix(cb.Data, "claim_")
		catId, err := strconv.Atoi(catIdStr)
		if err != nil {
			sendTgMessage(chatId, "❌ 无效的操作。")
			return
		}
		handleTgClaimCategory(chatId, cb.From, catId)
		return
	}
}

// handleTgClaimCategory 处理分类领取
func handleTgClaimCategory(chatId int64, from *TgUser, categoryId int) {
	tgId := strconv.FormatInt(from.Id, 10)
	tgUsername := from.Username
	if tgUsername == "" {
		tgUsername = from.FirstName
	}

	// 获取分类
	category, err := model.GetTgBotCategoryById(categoryId)
	if err != nil {
		sendTgMessage(chatId, "❌ 该分类不存在或已被删除。")
		return
	}
	if category.Status != 1 {
		sendTgMessage(chatId, "❌ 该分类已被禁用。")
		return
	}

	// 检查领取次数
	claimCount, err := model.CountTgBotClaims(tgId, categoryId)
	if err != nil {
		sendTgMessage(chatId, "❌ 系统错误，请稍后再试。")
		return
	}
	if claimCount >= int64(category.MaxClaims) {
		sendTgMessage(chatId, fmt.Sprintf("⚠️ 你在「%s」分类下已领取 %d/%d 次，已达上限。",
			category.Name, claimCount, category.MaxClaims))
		return
	}

	// 确保用户存在，不存在则自动创建
	user, created, password := ensureTgUser(tgId, tgUsername)
	if user == nil {
		sendTgMessage(chatId, "❌ 创建账户失败，请稍后再试。")
		return
	}

	// 查找可用兑换码
	code, err := model.FindAvailableRedemptionCode(category.Purpose)
	if err != nil {
		sendTgMessage(chatId, fmt.Sprintf("❌ 「%s」暂无可用兑换码，请联系管理员添加。", category.Name))
		return
	}

	// 兑换
	var quota int
	if category.Purpose == common.RedemptionPurposeRegistration {
		_, err = model.ConsumeRedemptionCodeForRegistration(code.Key, user.Id)
	} else {
		quota, err = model.Redeem(code.Key, user.Id)
	}
	if err != nil {
		sendTgMessage(chatId, "❌ 领取失败，请稍后再试。")
		return
	}

	// 记录领取
	claim := &model.TgBotClaim{
		TelegramId: tgId,
		CategoryId: categoryId,
		UserId:     user.Id,
		Quota:      quota,
	}
	_ = model.CreateTgBotClaim(claim)

	// 构建消息
	var msgParts []string
	msgParts = append(msgParts, fmt.Sprintf("✅ 「%s」领取成功！(%d/%d)", category.Name, claimCount+1, category.MaxClaims))

	if created {
		msgParts = append(msgParts, fmt.Sprintf("\n🆕 已为你创建账户：\n🔑 用户名：`%s`\n🔒 密码：`%s`\n⚠️ 请立即保存密码！", user.Username, password))
	}

	if quota > 0 {
		msgParts = append(msgParts, fmt.Sprintf("\n💰 充值额度：%s", tgFormatQuota(quota)))
	}

	if category.Purpose == common.RedemptionPurposeRegistration {
		msgParts = append(msgParts, fmt.Sprintf("\n🎫 邀请码：`%s`", code.Key))
	}

	sendTgMessage(chatId, strings.Join(msgParts, ""))
}

// ensureTgUser 确保TG用户在系统中存在，返回 (user, isNewlyCreated, plainPassword)
func ensureTgUser(tgId string, tgUsername string) (*model.User, bool, string) {
	var existingUser model.User
	err := model.DB.Where("telegram_id = ?", tgId).First(&existingUser).Error
	if err == nil {
		return &existingUser, false, ""
	}

	displayName := tgUsername
	if len(displayName) > 20 {
		displayName = displayName[:20]
	}

	username := fmt.Sprintf("tg_%s", tgId)
	password := common.GetRandomString(12)

	user := &model.User{
		Username:    username,
		DisplayName: displayName,
		Password:    password,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		TelegramId:  tgId,
	}
	if err := user.Insert(0); err != nil {
		username = fmt.Sprintf("tg_%s_%s", tgId, common.GetRandomString(4))
		user.Username = username
		if err := user.Insert(0); err != nil {
			return nil, false, ""
		}
	}
	return user, true, password
}

// handleTgMyInfo 处理 /myinfo 命令
func handleTgMyInfo(msg *TgMsg) {
	handleTgMyInfoByUser(msg.Chat.Id, msg.From)
}

func handleTgMyInfoByUser(chatId int64, from *TgUser) {
	tgId := strconv.FormatInt(from.Id, 10)

	var user model.User
	err := model.DB.Where("telegram_id = ?", tgId).First(&user).Error
	if err != nil {
		sendTgMessage(chatId, "❌ 你还没有领取过，请先点击分类按钮领取。")
		return
	}

	// 获取领取记录
	claims, _ := model.GetTgBotClaimsByTelegramId(tgId)
	claimInfo := ""
	if len(claims) > 0 {
		claimInfo = "\n\n📋 领取记录："
		for _, c := range claims {
			cat, err := model.GetTgBotCategoryById(c.CategoryId)
			catName := "未知分类"
			if err == nil {
				catName = cat.Name
			}
			claimInfo += fmt.Sprintf("\n  · %s", catName)
			if c.Quota > 0 {
				claimInfo += fmt.Sprintf("（%s）", tgFormatQuota(c.Quota))
			}
		}
	}

	sendTgMessage(chatId, fmt.Sprintf("📊 你的账户信息：\n\n"+
		"👤 用户名：`%s`\n"+
		"💰 剩余额度：%s\n"+
		"📈 已用额度：%s\n"+
		"🔢 请求次数：%d%s",
		user.Username,
		tgFormatQuota(user.Quota),
		tgFormatQuota(user.UsedQuota),
		user.RequestCount,
		claimInfo))
}

// handleTgRedeem 处理 /redeem 命令
func handleTgRedeem(msg *TgMsg, code string) {
	chatId := msg.Chat.Id
	tgId := strconv.FormatInt(msg.From.Id, 10)

	var user model.User
	err := model.DB.Where("telegram_id = ?", tgId).First(&user).Error
	if err != nil {
		sendTgMessage(chatId, "❌ 你还没有账户，请先领取。")
		return
	}

	quota, redeemErr := model.Redeem(code, user.Id)
	if redeemErr != nil {
		sendTgMessage(chatId, "❌ 兑换失败：兑换码无效或已被使用。")
		return
	}

	sendTgMessage(chatId, fmt.Sprintf("✅ 兑换成功！\n\n💰 充值额度：%s\n💰 当前余额：%s",
		tgFormatQuota(quota), tgFormatQuota(user.Quota+quota)))
}

// ========== Admin API for TG Bot Categories ==========

func GetTgBotCategories(c *gin.Context) {
	categories, err := model.GetAllTgBotCategories()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": categories})
}

func CreateTgBotCategory(c *gin.Context) {
	var category model.TgBotCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if category.Name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "分类名称不能为空"})
		return
	}
	if category.MaxClaims <= 0 {
		category.MaxClaims = 1
	}
	if category.Status == 0 {
		category.Status = 1
	}
	if err := model.CreateTgBotCategory(&category); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "创建成功", "data": category})
}

func UpdateTgBotCategory(c *gin.Context) {
	var category model.TgBotCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if category.Id == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "ID不能为空"})
		return
	}
	if err := model.UpdateTgBotCategory(&category); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "更新成功"})
}

func DeleteTgBotCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}
	if err := model.DeleteTgBotCategory(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

// ========== Admin API: Webhook Management ==========

func SetupTgBotWebhook(c *gin.Context) {
	token := common.TelegramBotToken
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先保存 Bot Token"})
		return
	}

	// 从请求头或 OptionMap 获取 ServerAddress
	serverAddress := ""
	if val, ok := common.OptionMap["ServerAddress"]; ok {
		serverAddress = val
	}
	if serverAddress == "" {
		scheme := "https"
		serverAddress = scheme + "://" + c.Request.Host
	}

	webhookUrl := serverAddress + "/api/tgbot/webhook"
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", token)

	body := map[string]interface{}{
		"url": webhookUrl,
	}
	bodyBytes, err := common.Marshal(body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "序列化失败"})
		return
	}

	resp, err := http.Post(apiUrl, "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请求 Telegram API 失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := common.DecodeJson(resp.Body, &result); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解析响应失败"})
		return
	}

	if ok, _ := result["ok"].(bool); ok {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Webhook 设置成功", "data": gin.H{"url": webhookUrl}})
	} else {
		desc, _ := result["description"].(string)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Telegram 返回错误: " + desc})
	}
}

func GetTgBotWebhookInfo(c *gin.Context) {
	token := common.TelegramBotToken
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Bot Token 未配置"})
		return
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/getWebhookInfo", token)
	resp, err := http.Get(apiUrl)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请求失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := common.DecodeJson(resp.Body, &result); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解析响应失败"})
		return
	}

	if ok, _ := result["ok"].(bool); ok {
		data, _ := result["result"].(map[string]interface{})
		c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
	} else {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取失败"})
	}
}

// ========== Telegram API Helpers ==========

func tgFormatQuota(quota int) string {
	if common.DisplayInCurrencyEnabled {
		return fmt.Sprintf("$%.4f", float64(quota)/common.QuotaPerUnit)
	}
	return fmt.Sprintf("%d", quota)
}

func sendTgMessage(chatId int64, text string) {
	token := common.TelegramBotToken
	if token == "" {
		common.SysError("TG Bot: token not configured")
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]interface{}{
		"chat_id":    chatId,
		"text":       text,
		"parse_mode": "Markdown",
	}
	tgPost(url, body)
}

func sendTgMessageWithKeyboard(chatId int64, text string, keyboard TgInlineKeyboardMarkup) {
	token := common.TelegramBotToken
	if token == "" {
		common.SysError("TG Bot: token not configured")
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]interface{}{
		"chat_id":      chatId,
		"text":         text,
		"parse_mode":   "Markdown",
		"reply_markup": keyboard,
	}
	tgPost(url, body)
}

func answerCallbackQuery(callbackQueryId string) {
	token := common.TelegramBotToken
	if token == "" {
		return
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", token)
	body := map[string]interface{}{
		"callback_query_id": callbackQueryId,
	}
	tgPost(url, body)
}

func tgPost(url string, body map[string]interface{}) {
	bodyBytes, err := common.Marshal(body)
	if err != nil {
		common.SysError("TG Bot: marshal failed: " + err.Error())
		return
	}
	resp, err := http.Post(url, "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		common.SysError("TG Bot: request failed: " + err.Error())
		return
	}
	defer resp.Body.Close()
}
