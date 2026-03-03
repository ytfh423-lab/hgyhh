package controller

import (
	"fmt"
	"io"
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
		common.SysLog(fmt.Sprintf("TG Bot webhook: JSON decode error: %s", err.Error()))
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	// 处理 callback_query（按钮点击）
	if update.CallbackQuery != nil {
		common.SysLog(fmt.Sprintf("TG Bot webhook: callback_query from user %d, data=%s",
			update.CallbackQuery.From.Id, update.CallbackQuery.Data))
		handleTgCallback(update.CallbackQuery)
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	if update.Message == nil || update.Message.From == nil || update.Message.From.IsBot {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	common.SysLog(fmt.Sprintf("TG Bot webhook: message from user %d in chat %d (type=%s): %s",
		update.Message.From.Id, update.Message.Chat.Id, update.Message.Chat.Type, update.Message.Text))

	msg := update.Message
	chatId := msg.Chat.Id
	text := strings.TrimSpace(msg.Text)
	isGroup := msg.Chat.Type == "group" || msg.Chat.Type == "supergroup"

	// 去掉 @botname 后缀，例如 /start@my_bot -> /start
	cmd := text
	if idx := strings.Index(cmd, "@"); idx > 0 {
		cmd = cmd[:idx]
	}

	switch {
	case cmd == "/start":
		handleTgStart(chatId, isGroup)
	case cmd == "/claim" || cmd == "/领取":
		handleTgStart(chatId, isGroup)
	case cmd == "/myrecords" || cmd == "/我的记录":
		handleTgMyRecords(chatId, msg.From, isGroup)
	case cmd == "/help":
		handleTgHelp(chatId)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleTgStart 发送欢迎消息 + 分类按钮菜单
func handleTgStart(chatId int64, isGroup bool) {
	categories, err := model.GetEnabledTgBotCategories()
	if err != nil || len(categories) == 0 {
		sendTgMessage(chatId, "👋 欢迎使用 "+common.SystemName+" 机器人！\n\n暂无可领取的项目，请联系管理员。")
		return
	}

	var rows [][]TgInlineKeyboardButton
	for _, cat := range categories {
		label := cat.Name
		if cat.Description != "" {
			label = cat.Name + " - " + cat.Description
		}
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: label, CallbackData: fmt.Sprintf("claim_%d", cat.Id)},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "📋 我的领取记录", CallbackData: "myrecords"},
	})

	welcome := "👋 欢迎使用 " + common.SystemName + " 机器人！\n\n请点击下方按钮领取对应的兑换码/邀请码："
	if isGroup {
		welcome += "\n\n💡 兑换码将通过私聊发送给你，请确保已先私聊机器人一次。"
	}

	sendTgMessageWithKeyboard(chatId, welcome,
		TgInlineKeyboardMarkup{InlineKeyboard: rows})
}

// handleTgHelp 发送帮助信息
func handleTgHelp(chatId int64) {
	helpText := "📖 机器人命令帮助：\n\n" +
		"/start - 显示领取菜单\n" +
		"/claim - 领取兑换码/邀请码\n" +
		"/myrecords - 查看我的领取记录\n" +
		"/help - 显示此帮助信息\n\n" +
		"💡 在群组中使用时，兑换码会通过私聊发送，请确保已先私聊过机器人。"
	sendTgMessage(chatId, helpText)
}

// handleTgCallback 处理按钮点击
func handleTgCallback(cb *TgCallbackQuery) {
	chatId := cb.Message.Chat.Id
	isGroup := cb.Message.Chat.Type == "group" || cb.Message.Chat.Type == "supergroup"

	// 应答 callback 避免按钮转圈
	answerCallbackQuery(cb.Id)

	if cb.Data == "myrecords" {
		handleTgMyRecords(chatId, cb.From, isGroup)
		return
	}

	if cb.Data == "menu" {
		handleTgStart(chatId, isGroup)
		return
	}

	if strings.HasPrefix(cb.Data, "claim_") {
		catIdStr := strings.TrimPrefix(cb.Data, "claim_")
		catId, err := strconv.Atoi(catIdStr)
		if err != nil {
			sendTgMessage(chatId, "❌ 无效的操作。")
			return
		}
		handleTgClaimCategory(chatId, cb.From, catId, isGroup)
		return
	}
}

// handleTgClaimCategory 处理分类领取（只发放兑换码，不创建平台账户）
func handleTgClaimCategory(chatId int64, from *TgUser, categoryId int, isGroup bool) {
	tgId := strconv.FormatInt(from.Id, 10)
	privateChatId := from.Id // 用于私聊发送敏感信息

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
		sendTgMessage(chatId, fmt.Sprintf("⚠️ 你在「%s」已领取 %d/%d 次，已达上限。",
			category.Name, claimCount, category.MaxClaims))
		return
	}

	// 从分类库存中查找可用码
	invItem, err := model.FindAvailableInventoryCode(categoryId)
	if err != nil {
		sendTgMessage(chatId, fmt.Sprintf("❌ 「%s」暂无库存，请联系管理员补充。", category.Name))
		return
	}

	// 标记库存码为已发放
	if err := model.MarkInventoryCodeDispensed(invItem.Id, tgId); err != nil {
		sendTgMessage(chatId, "❌ 领取失败，请稍后再试。")
		return
	}

	// 记录领取
	claim := &model.TgBotClaim{
		TelegramId: tgId,
		CategoryId: categoryId,
		CodeKey:    invItem.Code,
	}
	_ = model.CreateTgBotClaim(claim)

	// 发送兑换码给用户
	codeType := "兑换码"
	if category.Purpose == common.RedemptionPurposeRegistration {
		codeType = "邀请码"
	}

	codeMsg := fmt.Sprintf(
		"✅ 「%s」领取成功！(%d/%d)\n\n"+
			"🎫 你的%s：\n%s\n\n"+
			"请复制上方%s，前往网站使用。",
		category.Name, claimCount+1, category.MaxClaims,
		codeType, invItem.Code, codeType)

	if isGroup {
		// 群组中：通过私聊发送兑换码，群里只提示
		if sendTgMessageReturnsOk(privateChatId, codeMsg) {
			displayName := from.FirstName
			if from.Username != "" {
				displayName = "@" + from.Username
			}
			sendTgMessage(chatId, fmt.Sprintf("✅ %s 领取「%s」成功！%s已通过私聊发送，请查收。",
				displayName, category.Name, codeType))
		} else {
			// 私聊发送失败，回滚库存码
			_ = model.RollbackInventoryCode(invItem.Id)
			// 同时删除领取记录
			_ = model.DeleteTgBotClaim(claim.Id)
			sendTgMessage(chatId, "❌ 无法私聊发送"+codeType+"，请先私聊机器人发送 /start 后再试。")
		}
	} else {
		// 私聊中：直接发送
		sendTgMessage(chatId, codeMsg)
	}
}

// handleTgMyRecords 查看领取记录
func handleTgMyRecords(chatId int64, from *TgUser, isGroup bool) {
	tgId := strconv.FormatInt(from.Id, 10)
	privateChatId := from.Id

	claims, _ := model.GetTgBotClaimsByTelegramId(tgId)
	if len(claims) == 0 {
		sendTgMessageWithKeyboard(chatId,
			"📋 你还没有领取过任何兑换码。\n\n点击下方按钮开始领取：",
			TgInlineKeyboardMarkup{InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回菜单", CallbackData: "menu"}},
			}})
		return
	}

	msg := "📋 你的领取记录：\n"
	for _, c := range claims {
		cat, err := model.GetTgBotCategoryById(c.CategoryId)
		catName := "未知分类"
		if err == nil {
			catName = cat.Name
		}
		msg += fmt.Sprintf("\n· %s\n  %s", catName, c.CodeKey)
	}

	if isGroup {
		// 群组中：通过私聊发送记录（含敏感码）
		if sendTgMessageReturnsOk(privateChatId, msg) {
			sendTgMessage(chatId, "📋 你的领取记录已通过私聊发送，请查收。")
		} else {
			sendTgMessage(chatId, "❌ 无法私聊发送记录，请先私聊机器人发送 /start 后再试。")
		}
	} else {
		sendTgMessageWithKeyboard(chatId, msg,
			TgInlineKeyboardMarkup{InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回菜单", CallbackData: "menu"}},
			}})
	}
}

// ========== Admin API for TG Bot Categories ==========

func GetTgBotCategories(c *gin.Context) {
	categories, err := model.GetAllTgBotCategories()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	// 附带库存统计
	stockMap, _ := model.CountAllTgBotInventory()
	type categoryWithStock struct {
		model.TgBotCategory
		StockTotal     int64 `json:"stock_total"`
		StockAvailable int64 `json:"stock_available"`
	}
	var result []categoryWithStock
	for _, cat := range categories {
		item := categoryWithStock{TgBotCategory: *cat}
		if s, ok := stockMap[cat.Id]; ok {
			item.StockTotal = s["total"]
			item.StockAvailable = s["available"]
		}
		result = append(result, item)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
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

// ========== Admin API: Inventory Management ==========

func AddTgBotInventory(c *gin.Context) {
	var req struct {
		CategoryId int    `json:"category_id"`
		Codes      string `json:"codes"` // 换行分隔的多个码
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.CategoryId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	// 按换行分割
	lines := strings.Split(req.Codes, "\n")
	var codes []string
	for _, line := range lines {
		code := strings.TrimSpace(line)
		if code != "" {
			codes = append(codes, code)
		}
	}
	if len(codes) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请输入至少一个兑换码"})
		return
	}

	added, err := model.AddTgBotInventoryCodes(req.CategoryId, codes)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("成功添加 %d 个兑换码", added), "data": gin.H{"added": added}})
}

func GetTgBotInventory(c *gin.Context) {
	categoryId, err := strconv.Atoi(c.Query("category_id"))
	if err != nil || categoryId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	items, err := model.GetTgBotInventoryByCategoryId(categoryId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func DeleteTgBotInventoryItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}
	if err := model.ClearTgBotInventoryItem(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

// ========== Admin API: Bot Commands Registration ==========

func RegisterTgBotCommands(c *gin.Context) {
	token := common.TelegramBotToken
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先保存 Bot Token"})
		return
	}
	registerTgBotCommands(token)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "命令菜单注册成功（私聊+群组）"})
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
		"url":             webhookUrl,
		"allowed_updates": []string{"message", "callback_query"},
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
		// 注册机器人命令菜单（私聊+群组）
		registerTgBotCommands(token)
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Webhook 设置成功", "data": gin.H{"url": webhookUrl}})
	} else {
		desc, _ := result["description"].(string)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Telegram 返回错误: " + desc})
	}
}

// registerTgBotCommands 向 Telegram 注册机器人命令菜单
func registerTgBotCommands(token string) {
	commands := []map[string]string{
		{"command": "start", "description": "显示领取菜单"},
		{"command": "claim", "description": "领取兑换码/邀请码"},
		{"command": "myrecords", "description": "查看我的领取记录"},
		{"command": "help", "description": "显示帮助信息"},
	}

	// 注册默认命令（私聊）
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/setMyCommands", token)
	body := map[string]interface{}{
		"commands": commands,
	}
	tgPost(apiUrl, body)

	// 注册群组命令
	groupBody := map[string]interface{}{
		"commands": commands,
		"scope": map[string]string{
			"type": "all_group_chats",
		},
	}
	tgPost(apiUrl, groupBody)

	common.SysLog("TG Bot: commands registered for private and group chats")
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

func sendTgMessage(chatId int64, text string) {
	token := common.TelegramBotToken
	if token == "" {
		common.SysError("TG Bot: token not configured")
		return
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]interface{}{
		"chat_id": chatId,
		"text":    text,
	}
	tgPost(apiUrl, body)
}

// sendTgMessageReturnsOk 发送消息并返回是否成功（用于私聊尝试）
func sendTgMessageReturnsOk(chatId int64, text string) bool {
	token := common.TelegramBotToken
	if token == "" {
		return false
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]interface{}{
		"chat_id": chatId,
		"text":    text,
	}
	return tgPostReturnsOk(apiUrl, body)
}

func sendTgMessageWithKeyboard(chatId int64, text string, keyboard TgInlineKeyboardMarkup) {
	token := common.TelegramBotToken
	if token == "" {
		common.SysError("TG Bot: token not configured")
		return
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]interface{}{
		"chat_id":      chatId,
		"text":         text,
		"reply_markup": keyboard,
	}
	tgPost(apiUrl, body)
}

func answerCallbackQuery(callbackQueryId string) {
	token := common.TelegramBotToken
	if token == "" {
		return
	}
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", token)
	body := map[string]interface{}{
		"callback_query_id": callbackQueryId,
	}
	tgPost(apiUrl, body)
}

func tgPost(apiUrl string, body map[string]interface{}) {
	bodyBytes, err := common.Marshal(body)
	if err != nil {
		common.SysError("TG Bot: marshal failed: " + err.Error())
		return
	}
	resp, err := http.Post(apiUrl, "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		common.SysError("TG Bot: request failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	// 读取并记录错误响应
	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := common.Unmarshal(respBody, &result); err == nil {
		if ok, _ := result["ok"].(bool); !ok {
			common.SysError(fmt.Sprintf("TG Bot: API error: %s", string(respBody)))
		}
	}
	if resp.StatusCode != http.StatusOK {
		common.SysError(fmt.Sprintf("TG Bot: API returned HTTP %d: %s", resp.StatusCode, string(respBody)))
	}
}

// tgPostReturnsOk 发送请求并返回 Telegram API 是否返回 ok=true
func tgPostReturnsOk(apiUrl string, body map[string]interface{}) bool {
	bodyBytes, err := common.Marshal(body)
	if err != nil {
		common.SysError("TG Bot: marshal failed: " + err.Error())
		return false
	}
	resp, err := http.Post(apiUrl, "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		common.SysError("TG Bot: request failed: " + err.Error())
		return false
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := common.Unmarshal(respBody, &result); err != nil {
		return false
	}
	ok, _ := result["ok"].(bool)
	if !ok {
		common.SysError(fmt.Sprintf("TG Bot: API error: %s", string(respBody)))
	}
	return ok
}
