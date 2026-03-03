package controller

import (
	"fmt"
	"io"
	"math/rand"
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

	// 群组消息：追踪活跃度 + 自动删除上一条bot消息 + 抽奖触发
	if isGroup {
		handleGroupMessage(chatId, msg.From)
	}

	switch {
	case cmd == "/start":
		handleTgStart(chatId, isGroup)
	case cmd == "/claim" || cmd == "/领取":
		handleTgStart(chatId, isGroup)
	case cmd == "/myrecords" || cmd == "/我的记录":
		handleTgMyRecords(chatId, msg.From, isGroup)
	case cmd == "/lottery" || cmd == "/抽奖":
		handleTgLottery(chatId, msg.From, isGroup)
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
	if isGroup && common.TgBotLotteryEnabled {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🎰 抽奖", CallbackData: "lottery_info"},
		})
	}

	welcome := "👋 欢迎使用 " + common.SystemName + " 机器人！\n\n请点击下方按钮领取对应的兑换码/邀请码："
	if isGroup {
		welcome += "\n\n💡 兑换码将通过私聊发送给你，请确保已先私聊机器人一次。"
	}

	sendTgMessageWithKeyboard(chatId, welcome,
		TgInlineKeyboardMarkup{InlineKeyboard: rows})
}

// handleTgLottery 显示抽奖状态和入口
func handleTgLottery(chatId int64, from *TgUser, isGroup bool) {
	if !isGroup {
		sendTgMessage(chatId, "🎰 抽奖功能仅在群组中可用，请在群组里使用此命令。")
		return
	}
	if !common.TgBotLotteryEnabled {
		sendTgMessage(chatId, "🎰 抽奖功能暂未开启。")
		return
	}

	tgId := strconv.FormatInt(from.Id, 10)
	tracker, err := model.GetOrCreateMessageTracker(chatId, tgId)
	if err != nil {
		sendTgMessage(chatId, "❌ 系统错误，请稍后再试。")
		return
	}

	required := common.TgBotLotteryMessagesRequired
	if required <= 0 {
		required = 10
	}
	totalChances := tracker.MessageCount / required
	availableChances := totalChances - tracker.LotteryUsed
	nextAt := (tracker.LotteryUsed + 1) * required
	remaining := nextAt - tracker.MessageCount
	if remaining < 0 {
		remaining = 0
	}

	displayName := from.FirstName
	if from.Username != "" {
		displayName = "@" + from.Username
	}

	text := fmt.Sprintf("🎰 %s 的抽奖信息\n\n"+
		"📊 已发送消息：%d 条\n"+
		"🎫 可用抽奖次数：%d\n"+
		"📨 每 %d 条消息获得一次抽奖机会\n",
		displayName, tracker.MessageCount, availableChances, required)

	if availableChances > 0 {
		text += "\n点击下方按钮立即抽奖！"
		keyboard := TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🎰 点击抽奖", CallbackData: fmt.Sprintf("lottery_%s", tgId)}},
			},
		}
		sentMsgId := sendTgMessageWithKeyboardAndGetId(chatId, text, keyboard)
		if sentMsgId > 0 {
			_ = model.UpdateLastBotMsgId(tracker.Id, sentMsgId)
		}
	} else {
		text += fmt.Sprintf("\n💬 再发送 %d 条消息即可获得下一次抽奖机会！", remaining)
		sendTgMessage(chatId, text)
	}
}

// handleTgHelp 发送帮助信息
func handleTgHelp(chatId int64) {
	helpText := "📖 机器人命令帮助：\n\n" +
		"/start - 显示领取菜单\n" +
		"/claim - 领取兑换码/邀请码\n" +
		"/myrecords - 查看我的领取记录\n" +
		"/lottery - 查看抽奖状态（群组）\n" +
		"/help - 显示此帮助信息\n\n" +
		"💡 在群组中使用时，兑换码会通过私聊发送，请确保已先私聊过机器人。\n" +
		"🎰 在群组中发送消息可累积抽奖次数！"
	sendTgMessage(chatId, helpText)
}

// handleTgCallback 处理按钮点击
func handleTgCallback(cb *TgCallbackQuery) {
	chatId := cb.Message.Chat.Id
	isGroup := cb.Message.Chat.Type == "group" || cb.Message.Chat.Type == "supergroup"

	// 抽奖信息按钮（从 /start 菜单点击）
	if cb.Data == "lottery_info" {
		answerCallbackQuery(cb.Id)
		handleTgLottery(chatId, cb.From, isGroup)
		return
	}

	// 抽奖回调：不预先应答，由抽奖逻辑用 show_alert 应答
	if strings.HasPrefix(cb.Data, "lottery_") {
		handleTgLotteryCallback(cb)
		return
	}

	// 其他回调：普通应答
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
	if err := model.CreateTgBotClaim(claim); err != nil {
		common.SysError(fmt.Sprintf("TG Bot: failed to create claim record: %s", err.Error()))
	}

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

// ========== 群组消息追踪 + 抽奖逻辑 ==========

// handleGroupMessage 处理群组消息：追踪活跃度、删除上一条bot消息、触发抽奖
func handleGroupMessage(chatId int64, from *TgUser) {
	if !common.TgBotLotteryEnabled {
		return
	}

	tgId := strconv.FormatInt(from.Id, 10)

	// 获取或创建消息追踪器
	tracker, err := model.GetOrCreateMessageTracker(chatId, tgId)
	if err != nil {
		common.SysError("TG Bot: failed to get message tracker: " + err.Error())
		return
	}

	// 删除上一条bot发给该用户的消息
	if tracker.LastBotMsgId > 0 {
		deleteTgMessage(chatId, tracker.LastBotMsgId)
		_ = model.UpdateLastBotMsgId(tracker.Id, 0)
	}

	// 递增消息计数
	_ = model.IncrementMessageCount(tracker.Id)
	newCount := tracker.MessageCount + 1

	// 检查是否达到抽奖条件
	required := common.TgBotLotteryMessagesRequired
	if required <= 0 {
		required = 10
	}
	totalChances := newCount / required
	usedChances := tracker.LotteryUsed
	availableChances := totalChances - usedChances

	if availableChances > 0 {
		// 发送抽奖按钮
		displayName := from.FirstName
		if from.Username != "" {
			displayName = "@" + from.Username
		}
		text := fmt.Sprintf("🎉 %s 你已发送 %d 条消息，获得一次抽奖机会！\n点击下方按钮抽奖：",
			displayName, newCount)

		keyboard := TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🎰 点击抽奖", CallbackData: fmt.Sprintf("lottery_%s", tgId)}},
			},
		}
		sentMsgId := sendTgMessageWithKeyboardAndGetId(chatId, text, keyboard)
		if sentMsgId > 0 {
			_ = model.UpdateLastBotMsgId(tracker.Id, sentMsgId)
		}
	}
}

// handleTgLotteryCallback 处理抽奖按钮点击（结果只有点击者可见）
func handleTgLotteryCallback(cb *TgCallbackQuery) {
	chatId := cb.Message.Chat.Id
	tgId := strconv.FormatInt(cb.From.Id, 10)

	// 验证是否是本人的抽奖按钮
	expectedData := fmt.Sprintf("lottery_%s", tgId)
	if cb.Data != expectedData {
		answerCallbackQueryWithAlert(cb.Id, "❌ 这不是你的抽奖机会哦！")
		return
	}

	// 获取消息追踪器
	tracker, err := model.GetOrCreateMessageTracker(chatId, tgId)
	if err != nil {
		answerCallbackQueryWithAlert(cb.Id, "❌ 系统错误，请稍后再试。")
		return
	}

	// 检查是否有可用抽奖次数
	required := common.TgBotLotteryMessagesRequired
	if required <= 0 {
		required = 10
	}
	totalChances := tracker.MessageCount / required
	if totalChances <= tracker.LotteryUsed {
		answerCallbackQueryWithAlert(cb.Id, "❌ 你没有可用的抽奖次数了。继续聊天获取更多机会！")
		return
	}

	// 消耗一次抽奖次数
	_ = model.IncrementLotteryUsed(tracker.Id)

	// 删除抽奖按钮消息
	deleteTgMessage(chatId, cb.Message.MessageId)
	_ = model.UpdateLastBotMsgId(tracker.Id, 0)

	// 抽奖：判断是否中奖
	winRate := common.TgBotLotteryWinRate
	roll := rand.Intn(100)
	won := roll < winRate

	if !won {
		// 未中奖
		_ = model.CreateTgBotLotteryRecord(&model.TgBotLotteryRecord{
			TelegramId: tgId,
			ChatId:     chatId,
			Won:        false,
		})
		answerCallbackQueryWithAlert(cb.Id, "😢 很遗憾，没有中奖。\n继续在群里聊天，获取更多抽奖机会！")
		return
	}

	// 中奖：从奖品池取一个
	prize, err := model.GetAvailableTgBotLotteryPrize()
	if err != nil {
		// 奖品池空了
		_ = model.CreateTgBotLotteryRecord(&model.TgBotLotteryRecord{
			TelegramId: tgId,
			ChatId:     chatId,
			Won:        false,
		})
		answerCallbackQueryWithAlert(cb.Id, "😢 奖品已被领完，下次再来！")
		return
	}

	// 标记奖品为已中奖
	_ = model.MarkTgBotLotteryPrizeWon(prize.Id, tgId)

	// 记录中奖
	_ = model.CreateTgBotLotteryRecord(&model.TgBotLotteryRecord{
		TelegramId: tgId,
		ChatId:     chatId,
		PrizeName:  prize.Name,
		PrizeCode:  prize.Code,
		Won:        true,
	})

	// 通过 callback alert 弹窗显示中奖信息（只有点击者可见）
	alertText := fmt.Sprintf("🎊 恭喜中奖！\n\n奖品：%s\n兑换码：%s\n\n请复制兑换码前往网站使用。", prize.Name, prize.Code)
	answerCallbackQueryWithAlert(cb.Id, alertText)

	// 在群里发一条通知（不含兑换码）
	displayName := cb.From.FirstName
	if cb.From.Username != "" {
		displayName = "@" + cb.From.Username
	}
	sendTgMessage(chatId, fmt.Sprintf("🎊 恭喜 %s 在抽奖中获得了「%s」！", displayName, prize.Name))
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

// ========== Admin API: Lottery Prize Management ==========

func GetTgBotLotteryPrizes(c *gin.Context) {
	prizes, err := model.GetAllTgBotLotteryPrizes()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	total, available, _ := model.CountTgBotLotteryPrizes()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": prizes, "total": total, "available": available})
}

func AddTgBotLotteryPrizes(c *gin.Context) {
	var req struct {
		Name  string `json:"name"`
		Codes string `json:"codes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Codes == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误：需要奖品名称和兑换码"})
		return
	}
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
	added, err := model.AddTgBotLotteryPrizes(codes, req.Name)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("成功添加 %d 个奖品", added)})
}

func DeleteTgBotLotteryPrize(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}
	if err := model.DeleteTgBotLotteryPrize(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

func GetTgBotLotterySettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":           common.TgBotLotteryEnabled,
			"messages_required": common.TgBotLotteryMessagesRequired,
			"win_rate":          common.TgBotLotteryWinRate,
		},
	})
}

func GetTgBotSettings(c *gin.Context) {
	tokenSet := common.TelegramBotToken != ""
	maskedToken := ""
	if tokenSet {
		token := common.TelegramBotToken
		if len(token) > 8 {
			maskedToken = token[:4] + "****" + token[len(token)-4:]
		} else {
			maskedToken = "****"
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token_set":          tokenSet,
			"masked_token":       maskedToken,
			"bot_name":           common.TelegramBotName,
			"lottery_enabled":    common.TgBotLotteryEnabled,
			"messages_required":  common.TgBotLotteryMessagesRequired,
			"win_rate":           common.TgBotLotteryWinRate,
		},
	})
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
		{"command": "lottery", "description": "查看抽奖状态"},
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

// sendTgMessageWithKeyboardAndGetId 发送带键盘的消息并返回消息ID
func sendTgMessageWithKeyboardAndGetId(chatId int64, text string, keyboard TgInlineKeyboardMarkup) int {
	token := common.TelegramBotToken
	if token == "" {
		return 0
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]interface{}{
		"chat_id":      chatId,
		"text":         text,
		"reply_markup": keyboard,
	}
	bodyBytes, err := common.Marshal(body)
	if err != nil {
		return 0
	}
	resp, err := http.Post(apiUrl, "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := common.Unmarshal(respBody, &result); err != nil {
		return 0
	}
	if ok, _ := result["ok"].(bool); ok {
		if msg, ok := result["result"].(map[string]interface{}); ok {
			if msgId, ok := msg["message_id"].(float64); ok {
				return int(msgId)
			}
		}
	}
	return 0
}

// deleteTgMessage 删除消息
func deleteTgMessage(chatId int64, messageId int) {
	token := common.TelegramBotToken
	if token == "" {
		return
	}
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/deleteMessage", token)
	body := map[string]interface{}{
		"chat_id":    chatId,
		"message_id": messageId,
	}
	tgPost(apiUrl, body)
}

// answerCallbackQueryWithAlert 用弹窗回复 callback（只有点击者可见）
func answerCallbackQueryWithAlert(callbackQueryId string, text string) {
	token := common.TelegramBotToken
	if token == "" {
		return
	}
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", token)
	body := map[string]interface{}{
		"callback_query_id": callbackQueryId,
		"text":              text,
		"show_alert":        true,
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
