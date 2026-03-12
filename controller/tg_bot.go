package controller

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
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
		handleTgStart(chatId, isGroup, msg.From)
	case cmd == "/claim" || cmd == "/领取":
		handleTgStart(chatId, isGroup, msg.From)
	case cmd == "/myrecords" || cmd == "/我的记录":
		handleTgMyRecords(chatId, msg.From, isGroup)
	case cmd == "/lottery" || cmd == "/抽奖":
		handleTgLottery(chatId, msg.From, isGroup)
	case cmd == "/farm" || cmd == "/农场":
		handleFarmCommand(chatId, msg.From, isGroup)
	case cmd == "/bindaccount" || cmd == "/绑定账号":
		handleTgBindAccount(chatId, msg.From, isGroup, "")
	case cmd == "/help":
		handleTgHelp(chatId, msg.From)
	default:
		// 私聊中发送非命令文本，尝试作为 API Key 绑定
		if !isGroup && strings.HasPrefix(text, "sk-") {
			handleTgBindAccount(chatId, msg.From, isGroup, text)
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleTgStart 发送欢迎消息 + 分类按钮菜单
func handleTgStart(chatId int64, isGroup bool, from *TgUser) {
	categories, err := model.GetEnabledTgBotCategories()
	if err != nil || len(categories) == 0 {
		sendTgMessage(chatId, "👋 欢迎使用 "+common.SystemName+" 机器人！\n\n暂无可领取的项目，请联系管理员。", from)
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
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🌾 农场小游戏", CallbackData: "farm"},
	})

	welcome := "👋 欢迎使用 " + common.SystemName + " 机器人！\n\n请点击下方按钮领取对应的兑换码/邀请码："
	if isGroup {
		welcome += "\n\n💡 兑换码将通过私聊发送给你，请确保已先私聊机器人一次。"
	}

	sendTgMessageWithKeyboard(chatId, welcome,
		TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

// handleTgLottery 显示抽奖状态和入口
func handleTgLottery(chatId int64, from *TgUser, isGroup bool) {
	if !isGroup {
		sendTgMessage(chatId, "🎰 抽奖功能仅在群组中可用，请在群组里使用此命令。", from)
		return
	}
	if !common.TgBotLotteryEnabled {
		sendTgMessage(chatId, "🎰 抽奖功能暂未开启。", from)
		return
	}

	tgId := strconv.FormatInt(from.Id, 10)
	tracker, err := model.GetOrCreateMessageTracker(chatId, tgId)
	if err != nil {
		sendTgMessage(chatId, "❌ 系统错误，请稍后再试。", from)
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
		sendTgMessageWithKeyboard(chatId, text, keyboard, from)
	} else {
		text += fmt.Sprintf("\n💬 再发送 %d 条消息即可获得下一次抽奖机会！", remaining)
		sendTgMessage(chatId, text, from)
	}
}

// handleTgHelp 发送帮助信息
func handleTgHelp(chatId int64, from *TgUser) {
	helpText := "📖 机器人命令帮助：\n\n" +
		"/start - 显示领取菜单\n" +
		"/claim - 领取兑换码/邀请码\n" +
		"/myrecords - 查看我的领取记录\n" +
		"/lottery - 查看抽奖状态（群组）\n" +
		"/farm - 🌾 农场小游戏\n" +
		"/bindaccount - 绑定平台账号\n" +
		"/help - 显示此帮助信息\n\n" +
		"💡 绑定账号：私聊发送你的 API Key（sk-xxx）即可自动绑定\n" +
		"ℹ️ 在群组中使用时，兑换码会通过私聊发送，请确保已先私聊过机器人。\n" +
		"🎰 在群组中发送消息可累积抽奖次数！\n" +
		"🌾 种菜、收菜、偷菜，收获直接变成账户额度！"
	sendTgMessage(chatId, helpText, from)
}

// handleTgBindAccount 通过 API Key 绑定平台账号
func handleTgBindAccount(chatId int64, from *TgUser, isGroup bool, apiKey string) {
	tgId := strconv.FormatInt(from.Id, 10)

	if isGroup {
		sendTgMessage(chatId, "🔑 请私聊机器人发送你的 API Key 进行绑定，不要在群组中发送！", from)
		return
	}

	// 检查是否已绑定
	existingUser := &model.User{TelegramId: tgId}
	if err := existingUser.FillUserByTelegramId(); err == nil {
		sendTgMessage(chatId, fmt.Sprintf("✅ 你的 Telegram 账号已绑定到平台用户「%s」。\n\n如需更换绑定，请联系管理员。", existingUser.Username), from)
		return
	}

	if apiKey == "" {
		sendTgMessage(chatId, "🔑 账号绑定说明：\n\n"+
			"请直接发送你在平台上的任意一个 API Key（以 sk- 开头）即可完成绑定。\n\n"+
			"绑定后可使用农场游戏、收获额度等功能。\n\n"+
			"⚠️ API Key 仅用于验证身份，不会被存储或泄露。", from)
		return
	}

	// 通过 API Key 查找用户（数据库中存储的 key 不含 sk- 前缀）
	lookupKey := strings.TrimPrefix(apiKey, "sk-")
	// 如果 key 中包含 -，只取第一段（与 auth 中间件一致）
	if parts := strings.Split(lookupKey, "-"); len(parts) > 0 {
		lookupKey = parts[0]
	}
	token, err := model.GetTokenByKey(lookupKey, true)
	if err != nil || token == nil {
		sendTgMessage(chatId, "❌ API Key 无效，请检查后重试。\n\n请发送正确的 API Key（以 sk- 开头）。", from)
		return
	}

	// 检查该平台账号是否已被其他 Telegram 绑定
	if model.IsTelegramIdAlreadyTaken(tgId) {
		sendTgMessage(chatId, "❌ 该 Telegram 账号已被绑定，如需更换请联系管理员。", from)
		return
	}

	// 获取用户信息
	user, err := model.GetUserById(token.UserId, false)
	if err != nil || user == nil {
		sendTgMessage(chatId, "❌ 系统错误，无法查找用户信息。", from)
		return
	}

	// 检查该用户是否已绑定其他 Telegram
	if user.TelegramId != "" {
		sendTgMessage(chatId, "❌ 该平台账号已绑定了另一个 Telegram 账号。如需更换请联系管理员。", from)
		return
	}

	// 执行绑定
	err = model.DB.Model(user).Update("telegram_id", tgId).Error
	if err != nil {
		common.SysError(fmt.Sprintf("TG Bot: bind account failed for tgId=%s userId=%d: %s", tgId, user.Id, err.Error()))
		sendTgMessage(chatId, "❌ 绑定失败，请稍后再试。", from)
		return
	}

	common.SysLog(fmt.Sprintf("TG Bot: user %s bound telegram %s to platform user %s (id=%d)", tgId, tgId, user.Username, user.Id))

	// 删除用户发送的 API Key 消息（安全）
	if from.Id > 0 {
		// 无法删除用户消息（bot只能删除自己的消息或群组中的消息），但在私聊中是安全的
	}

	sendTgMessage(chatId, fmt.Sprintf("✅ 绑定成功！\n\n"+
		"🔗 平台用户：%s\n"+
		"💰 当前余额：%s\n\n"+
		"现在可以使用农场游戏等功能了！发送 /farm 开始种菜吧 🌾",
		user.Username, fmt.Sprintf("$%.2f", float64(user.Quota)/common.QuotaPerUnit)), from)
}

// handleTgCallback 处理按钮点击
func handleTgCallback(cb *TgCallbackQuery) {
	chatId := cb.Message.Chat.Id
	isGroup := cb.Message.Chat.Type == "group" || cb.Message.Chat.Type == "supergroup"

	// 农场游戏回调（仅群组可用）
	if cb.Data == "farm" || strings.HasPrefix(cb.Data, "farm_") || strings.HasPrefix(cb.Data, "ranch") {
		answerCallbackQuery(cb.Id)
		if !isGroup {
			sendTgMessage(chatId, "🌾 农场游戏仅限群组中使用！\n\n请在群组里发送 /farm 开始种菜。\n私聊仅支持绑定账号功能。", cb.From)
			return
		}
		handleFarmCallback(cb)
		return
	}

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
		handleTgStart(chatId, isGroup, cb.From)
		return
	}

	if strings.HasPrefix(cb.Data, "claim_") {
		catIdStr := strings.TrimPrefix(cb.Data, "claim_")
		catId, err := strconv.Atoi(catIdStr)
		if err != nil {
			sendTgMessage(chatId, "❌ 无效的操作。", cb.From)
			return
		}
		handleTgClaimCategory(chatId, cb.From, catId, isGroup)
		return
	}
}

// handleTgClaimCategory 处理分类领取（事务性：查码+标记+写记录一步完成）
func handleTgClaimCategory(chatId int64, from *TgUser, categoryId int, isGroup bool) {
	tgId := strconv.FormatInt(from.Id, 10)
	privateChatId := from.Id

	// 获取分类
	category, err := model.GetTgBotCategoryById(categoryId)
	if err != nil {
		sendTgMessage(chatId, "❌ 该分类不存在或已被删除。", from)
		return
	}
	if category.Status != 1 {
		sendTgMessage(chatId, "❌ 该分类已被禁用。", from)
		return
	}

	// 随机取一个未使用的库存码直接发放（无限制）
	code, err := model.DispenseRandomCode(categoryId, tgId)
	if err != nil {
		common.SysError(fmt.Sprintf("TG Bot: DispenseRandomCode failed for tgId=%s cat=%d: %s", tgId, categoryId, err.Error()))
		sendTgMessage(chatId, fmt.Sprintf("❌ 「%s」暂无库存或领取失败，请联系管理员。", category.Name), from)
		return
	}

	common.SysLog(fmt.Sprintf("TG Bot: user %s claimed code from cat %d, code=%s", tgId, categoryId, code))

	// 发送兑换码给用户
	codeType := "兑换码"
	if category.Purpose == common.RedemptionPurposeRegistration {
		codeType = "邀请码"
	}

	codeMsg := fmt.Sprintf(
		"✅ 「%s」领取成功！\n\n"+
			"🎫 你的%s：\n%s\n\n"+
			"请复制上方%s，前往网站使用。",
		category.Name,
		codeType, code, codeType)

	displayName := from.FirstName
	if from.Username != "" {
		displayName = "@" + from.Username
	}

	if isGroup {
		if sendTgMessageReturnsOk(privateChatId, codeMsg, from) {
			sendTgMessage(chatId, fmt.Sprintf("✅ %s 领取「%s」成功！%s已通过私聊发送，请查收。",
				displayName, category.Name, codeType), from)
		} else {
			sendTgMessage(chatId, fmt.Sprintf("✅ %s 领取「%s」成功！\n\n⚠️ 无法通过私聊发送%s，请先私聊机器人发送 /start，然后发送 /myrecords 查看你的%s。",
				displayName, category.Name, codeType, codeType), from)
		}
	} else {
		sendTgMessage(chatId, codeMsg, from)
	}
}

// handleTgMyRecords 查看领取记录（含抽奖中奖记录）
func handleTgMyRecords(chatId int64, from *TgUser, isGroup bool) {
	tgId := strconv.FormatInt(from.Id, 10)
	privateChatId := from.Id

	claims, _ := model.GetTgBotClaimsByTelegramId(tgId)
	lotteryWins, _ := model.GetTgBotLotteryRecords(tgId)

	if len(claims) == 0 && len(lotteryWins) == 0 {
		sendTgMessageWithKeyboard(chatId,
			"📋 你还没有任何领取或中奖记录。\n\n点击下方按钮开始领取：",
			TgInlineKeyboardMarkup{InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回菜单", CallbackData: "menu"}},
			}}, from)
		return
	}

	msg := ""
	if len(claims) > 0 {
		msg += "📋 你的领取记录：\n"
		for _, c := range claims {
			cat, err := model.GetTgBotCategoryById(c.CategoryId)
			catName := "未知分类"
			if err == nil {
				catName = cat.Name
			}
			msg += fmt.Sprintf("\n· %s\n  %s", catName, c.CodeKey)
		}
	}

	if len(lotteryWins) > 0 {
		if msg != "" {
			msg += "\n\n"
		}
		msg += "🎉 你的中奖记录：\n"
		for _, r := range lotteryWins {
			msg += fmt.Sprintf("\n· %s\n  %s", r.PrizeName, r.PrizeCode)
		}
	}

	if isGroup {
		// 群组中：通过私聊发送记录（含敏感码）
		if sendTgMessageReturnsOk(privateChatId, msg, from) {
			sendTgMessage(chatId, "📋 你的记录已通过私聊发送，请查收。", from)
		} else {
			sendTgMessage(chatId, "❌ 无法私聊发送记录，请先私聊机器人发送 /start 后再试。", from)
		}
	} else {
		sendTgMessageWithKeyboard(chatId, msg,
			TgInlineKeyboardMarkup{InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回菜单", CallbackData: "menu"}},
			}}, from)
	}
}

// ========== 群组消息追踪 + 抽奖逻辑 ==========

// handleGroupMessage 处理群组消息：追踪活跃度、触发抽奖
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
		// 先删除旧的抽奖按钮消息（只在发送新按钮时才删除）
		if tracker.LastBotMsgId > 0 {
			deleteTgMessage(chatId, tracker.LastBotMsgId)
			_ = model.UpdateLastBotMsgId(tracker.Id, 0)
		}

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
		sentMsgId := sendTgMessageWithKeyboardAndGetId(chatId, text, keyboard, from)
		if sentMsgId > 0 {
			_ = model.UpdateLastBotMsgId(tracker.Id, sentMsgId)
		}
	}
}

// handleTgLotteryCallback 处理抽奖按钮点击
func handleTgLotteryCallback(cb *TgCallbackQuery) {
	chatId := cb.Message.Chat.Id
	tgId := strconv.FormatInt(cb.From.Id, 10)
	privateChatId := cb.From.Id

	displayName := cb.From.FirstName
	if cb.From.Username != "" {
		displayName = "@" + cb.From.Username
	}

	// 验证是否是本人的抽奖按钮
	expectedData := fmt.Sprintf("lottery_%s", tgId)
	if cb.Data != expectedData {
		answerCallbackQuery(cb.Id)
		sendTgMessage(chatId, "❌ 这不是你的抽奖机会哦！", cb.From)
		return
	}

	// 获取消息追踪器
	tracker, err := model.GetOrCreateMessageTracker(chatId, tgId)
	if err != nil {
		answerCallbackQuery(cb.Id)
		sendTgMessage(chatId, "❌ 系统错误，请稍后再试。", cb.From)
		return
	}

	// 检查是否有可用抽奖次数
	required := common.TgBotLotteryMessagesRequired
	if required <= 0 {
		required = 10
	}
	totalChances := tracker.MessageCount / required
	if totalChances <= tracker.LotteryUsed {
		answerCallbackQuery(cb.Id)
		sendTgMessage(chatId, fmt.Sprintf("❌ %s 你没有可用的抽奖次数了。继续聊天获取更多机会！", displayName), cb.From)
		return
	}

	// 消耗一次抽奖次数
	_ = model.IncrementLotteryUsed(tracker.Id)

	// 删除抽奖按钮消息
	deleteTgMessage(chatId, cb.Message.MessageId)
	_ = model.UpdateLastBotMsgId(tracker.Id, 0)

	// 先应答 callback query（防止 Telegram 显示加载状态）
	answerCallbackQuery(cb.Id)

	// 抽奖：判断是否中奖
	winRate := common.TgBotLotteryWinRate
	roll := rand.Intn(100)
	won := roll < winRate

	// 计算剩余次数信息
	newUsed := tracker.LotteryUsed + 1
	remainChances := totalChances - newUsed
	nextAt := (newUsed + 1) * required
	needMore := nextAt - tracker.MessageCount
	if needMore < 0 {
		needMore = 0
	}

	if !won {
		// 未中奖 - 发送可见消息
		_ = model.CreateTgBotLotteryRecord(&model.TgBotLotteryRecord{
			TelegramId: tgId,
			ChatId:     chatId,
			Won:        false,
		})
		loseMsg := fmt.Sprintf("😢 %s 很遗憾，没有中奖。", displayName)
		if remainChances > 0 {
			loseMsg += fmt.Sprintf("\n\n🎫 你还有 %d 次抽奖机会，发送 /lottery 继续抽奖！", remainChances)
		} else {
			loseMsg += fmt.Sprintf("\n\n💬 再发送 %d 条消息即可获得下一次抽奖机会！", needMore)
		}
		sendTgMessage(chatId, loseMsg, cb.From)
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
		sendTgMessage(chatId, fmt.Sprintf("😢 %s 奖品已被领完，下次再来！", displayName), cb.From)
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

	// 通过私聊发送中奖兑换码（安全）
	prizeMsg := fmt.Sprintf("🎊 恭喜中奖！\n\n奖品：%s\n兑换码：%s\n\n请复制兑换码前往网站使用。", prize.Name, prize.Code)
	if sendTgMessageReturnsOk(privateChatId, prizeMsg, cb.From) {
		// 在群里发一条通知（不含兑换码）
		sendTgMessage(chatId, fmt.Sprintf("🎊 恭喜 %s 在抽奖中获得了「%s」！兑换码已通过私聊发送，请查收。", displayName, prize.Name), cb.From)
	} else {
		// 私聊失败，直接在群里显示（用 alert 弹窗作为备选）
		sendTgMessage(chatId, fmt.Sprintf("🎊 恭喜 %s 在抽奖中获得了「%s」！\n\n⚠️ 无法私聊发送兑换码，请私聊机器人发送 /start 后使用 /myrecords 查看。", displayName, prize.Name), cb.From)
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
			"token_set":            tokenSet,
			"masked_token":         maskedToken,
			"bot_name":             common.TelegramBotName,
			"lottery_enabled":      common.TgBotLotteryEnabled,
			"messages_required":    common.TgBotLotteryMessagesRequired,
			"win_rate":             common.TgBotLotteryWinRate,
			"farm_plot_price":      common.TgBotFarmPlotPrice,
			"farm_dog_price":       common.TgBotFarmDogPrice,
			"farm_dog_food_price":  common.TgBotFarmDogFoodPrice,
			"farm_dog_grow_hours":  common.TgBotFarmDogGrowHours,
			"farm_dog_guard_rate":  common.TgBotFarmDogGuardRate,
			"farm_water_interval":  common.TgBotFarmWaterInterval,
			"farm_wilt_duration":   common.TgBotFarmWiltDuration,
			"farm_event_chance":    common.TgBotFarmEventChance,
			"farm_disaster_chance": common.TgBotFarmDisasterChance,
			"farm_steal_cooldown":        common.TgBotFarmStealCooldown,
			"farm_soil_max_level":        common.TgBotFarmSoilMaxLevel,
			"farm_soil_upgrade_price_2":  common.TgBotFarmSoilUpgradePrice2,
			"farm_soil_upgrade_price_3":  common.TgBotFarmSoilUpgradePrice3,
			"farm_soil_upgrade_price_4":  common.TgBotFarmSoilUpgradePrice4,
			"farm_soil_upgrade_price_5":  common.TgBotFarmSoilUpgradePrice5,
			"farm_soil_speed_bonus":      common.TgBotFarmSoilSpeedBonus,
			// 牧场
			"ranch_max_animals":          common.TgBotRanchMaxAnimals,
			"ranch_feed_price":           common.TgBotRanchFeedPrice,
			"ranch_water_price":          common.TgBotRanchWaterPrice,
			"ranch_feed_interval":        common.TgBotRanchFeedInterval,
			"ranch_water_interval":       common.TgBotRanchWaterInterval,
			"ranch_hunger_death_hours":   common.TgBotRanchHungerDeathHours,
			"ranch_thirst_death_hours":   common.TgBotRanchThirstDeathHours,
			"ranch_chicken_price":        common.TgBotRanchChickenPrice,
			"ranch_duck_price":           common.TgBotRanchDuckPrice,
			"ranch_goose_price":          common.TgBotRanchGoosePrice,
			"ranch_pig_price":            common.TgBotRanchPigPrice,
			"ranch_sheep_price":          common.TgBotRanchSheepPrice,
			"ranch_cow_price":            common.TgBotRanchCowPrice,
			"ranch_chicken_grow_secs":    common.TgBotRanchChickenGrowSecs,
			"ranch_duck_grow_secs":       common.TgBotRanchDuckGrowSecs,
			"ranch_goose_grow_secs":      common.TgBotRanchGooseGrowSecs,
			"ranch_pig_grow_secs":        common.TgBotRanchPigGrowSecs,
			"ranch_sheep_grow_secs":      common.TgBotRanchSheepGrowSecs,
			"ranch_cow_grow_secs":        common.TgBotRanchCowGrowSecs,
			"ranch_chicken_meat_price":   common.TgBotRanchChickenMeatPrice,
			"ranch_duck_meat_price":      common.TgBotRanchDuckMeatPrice,
			"ranch_goose_meat_price":     common.TgBotRanchGooseMeatPrice,
			"ranch_pig_meat_price":       common.TgBotRanchPigMeatPrice,
			"ranch_sheep_meat_price":     common.TgBotRanchSheepMeatPrice,
			"ranch_cow_meat_price":       common.TgBotRanchCowMeatPrice,
			"ranch_manure_interval":      common.TgBotRanchManureInterval,
			"ranch_manure_clean_price":   common.TgBotRanchManureCleanPrice,
			"ranch_manure_grow_penalty":  common.TgBotRanchManureGrowPenalty,
			// 等级系统
			"farm_unlock_steal":         common.TgBotFarmUnlockSteal,
			"farm_unlock_dog":           common.TgBotFarmUnlockDog,
			"farm_unlock_ranch":         common.TgBotFarmUnlockRanch,
			"farm_unlock_fish":          common.TgBotFarmUnlockFish,
			"farm_unlock_workshop":      common.TgBotFarmUnlockWorkshop,
			"farm_unlock_market":        common.TgBotFarmUnlockMarket,
			"farm_unlock_tasks":         common.TgBotFarmUnlockTasks,
			"farm_unlock_achieve":       common.TgBotFarmUnlockAchieve,
			"farm_unlock_leaderboard":   common.TgBotFarmUnlockLeaderboard,
			"farm_unlock_trading":       common.TgBotFarmUnlockTrading,
			"farm_unlock_games":         common.TgBotFarmUnlockGames,
			"farm_unlock_encyclopedia":  common.TgBotFarmUnlockEncyclopedia,
			"farm_unlock_automation":    common.TgBotFarmUnlockAutomation,
			"farm_unlock_tree_farm":     common.TgBotFarmUnlockTreeFarm,
			// 树场系统
			"tree_farm_slot_price":      common.TgBotTreeFarmSlotPrice,
			"tree_farm_water_interval":  common.TgBotTreeFarmWaterInterval,
			"tree_farm_water_bonus":     common.TgBotTreeFarmWaterBonus,
			"tree_farm_fert_bonus":      common.TgBotTreeFarmFertilizerBonus,
			"tree_farm_stump_clear_secs": common.TgBotTreeFarmStumpClearSecs,
			"farm_level_prices":         common.OptionMap["TgBotFarmLevelPrices"],
			// 银行贷款
			"farm_bank_admin_id":        common.TgBotFarmBankAdminId,
			"farm_bank_interest_rate":   common.TgBotFarmBankInterestRate,
			"farm_bank_max_loan_days":   common.TgBotFarmBankMaxLoanDays,
			"farm_bank_base_amount":     common.TgBotFarmBankBaseAmount,
			"farm_bank_max_multiplier":  common.TgBotFarmBankMaxMultiplier,
			"farm_bank_unlock_level":    common.TgBotFarmBankUnlockLevel,
			"farm_mortgage_max_amount":  common.TgBotFarmMortgageMaxAmount,
			"farm_mortgage_interest_rate": common.TgBotFarmMortgageInterestRate,
			"farm_season_days":          common.TgBotFarmSeasonDays,
			"farm_season_in_bonus":      common.TgBotFarmSeasonInBonus,
			"farm_season_off_bonus":     common.TgBotFarmSeasonOffBonus,
			"farm_season_in_growth":     common.TgBotFarmSeasonInGrowth,
			"farm_season_off_growth":    common.TgBotFarmSeasonOffGrowth,
			"farm_warehouse_base_slots":          common.TgBotFarmWarehouseMaxSlots,
			"farm_warehouse_max_level":            common.TgBotFarmWarehouseMaxLevel,
			"farm_warehouse_upgrade_price":        common.TgBotFarmWarehouseUpgradePrice,
			"farm_warehouse_capacity_per_level":   common.TgBotFarmWarehouseCapacityPerLevel,
			"farm_warehouse_expiry_bonus_per_level": common.TgBotFarmWarehouseExpiryBonusPerLevel,
			// 农场公告
			"farm_announcement_enabled": common.OptionMap["FarmAnnouncementEnabled"],
			"farm_announcement_text":    common.OptionMap["FarmAnnouncementText"],
			"farm_announcement_type":    common.OptionMap["FarmAnnouncementType"],
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
		{"command": "farm", "description": "🌾 农场小游戏"},
		{"command": "bindaccount", "description": "🔑 绑定平台账号"},
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

// tgHTMLEscape 转义 HTML 特殊字符
func tgHTMLEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// tgMentionHTML 生成 HTML 格式的 @提及
func tgMentionHTML(from *TgUser) string {
	if from == nil {
		return ""
	}
	name := from.FirstName
	if name == "" {
		name = from.Username
	}
	if name == "" {
		name = fmt.Sprintf("%d", from.Id)
	}
	return fmt.Sprintf(`<a href="tg://user?id=%d">@%s</a>`, from.Id, tgHTMLEscape(name))
}

// tgFormatMsg 格式化消息文本（HTML转义 + @提及前缀）
func tgFormatMsg(text string, from ...*TgUser) string {
	escaped := tgHTMLEscape(text)
	if len(from) > 0 && from[0] != nil {
		mention := tgMentionHTML(from[0])
		return mention + "\n" + escaped
	}
	return escaped
}

func sendTgMessage(chatId int64, text string, from ...*TgUser) {
	token := common.TelegramBotToken
	if token == "" {
		common.SysError("TG Bot: token not configured")
		return
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]interface{}{
		"chat_id":    chatId,
		"text":       tgFormatMsg(text, from...),
		"parse_mode": "HTML",
	}
	tgPost(apiUrl, body)
}

// sendTgMessageReturnsOk 发送消息并返回是否成功（用于私聊尝试）
func sendTgMessageReturnsOk(chatId int64, text string, from ...*TgUser) bool {
	token := common.TelegramBotToken
	if token == "" {
		return false
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]interface{}{
		"chat_id":    chatId,
		"text":       tgFormatMsg(text, from...),
		"parse_mode": "HTML",
	}
	return tgPostReturnsOk(apiUrl, body)
}

func sendTgMessageWithKeyboard(chatId int64, text string, keyboard TgInlineKeyboardMarkup, from ...*TgUser) {
	token := common.TelegramBotToken
	if token == "" {
		common.SysError("TG Bot: token not configured")
		return
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]interface{}{
		"chat_id":      chatId,
		"text":         tgFormatMsg(text, from...),
		"reply_markup": keyboard,
		"parse_mode":   "HTML",
	}
	tgPost(apiUrl, body)
}

// sendTgMessageWithKeyboardAndGetId 发送带键盘的消息并返回消息ID
func sendTgMessageWithKeyboardAndGetId(chatId int64, text string, keyboard TgInlineKeyboardMarkup, from ...*TgUser) int {
	token := common.TelegramBotToken
	if token == "" {
		return 0
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]interface{}{
		"chat_id":      chatId,
		"text":         tgFormatMsg(text, from...),
		"reply_markup": keyboard,
		"parse_mode":   "HTML",
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

// editTgMessage 编辑已有消息
func editTgMessage(chatId int64, messageId int, text string, keyboard *TgInlineKeyboardMarkup, from ...*TgUser) {
	token := common.TelegramBotToken
	if token == "" {
		return
	}
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", token)
	body := map[string]interface{}{
		"chat_id":    chatId,
		"message_id": messageId,
		"text":       tgFormatMsg(text, from...),
		"parse_mode": "HTML",
	}
	if keyboard != nil {
		body["reply_markup"] = *keyboard
	}
	tgPost(apiUrl, body)
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

// sendTgPhoto 发送图片到 Telegram（multipart upload）
func sendTgPhoto(chatId int64, pngData []byte, caption string, keyboard *TgInlineKeyboardMarkup) {
	token := common.TelegramBotToken
	if token == "" {
		return
	}
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", token)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", fmt.Sprintf("%d", chatId))
	if caption != "" {
		_ = w.WriteField("caption", caption)
	}
	if keyboard != nil {
		kbBytes, _ := common.Marshal(keyboard)
		_ = w.WriteField("reply_markup", string(kbBytes))
	}
	part, _ := w.CreateFormFile("photo", "chart.png")
	_, _ = part.Write(pngData)
	_ = w.Close()

	resp, err := http.Post(apiUrl, w.FormDataContentType(), &buf)
	if err != nil {
		common.SysError("TG Bot: sendPhoto failed: " + err.Error())
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := common.Unmarshal(respBody, &result); err == nil {
		if ok, _ := result["ok"].(bool); !ok {
			common.SysError(fmt.Sprintf("TG Bot: sendPhoto API error: %s", string(respBody)))
		}
	}
}

// ========== Admin API: Farm Management ==========

// AdminResetNegativeBalances resets all users with negative quota to 0
func AdminResetNegativeBalances(c *gin.Context) {
	affected, err := model.ResetNegativeBalanceUsers()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "重置失败: " + err.Error()})
		return
	}
	common.SysLog(fmt.Sprintf("Admin: reset %d negative-balance users to 0", affected))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已将 %d 个负余额用户重置为 0", affected),
		"data":    gin.H{"affected": affected},
	})
}

// AdminBetaCleanup manually triggers beta farm data cleanup and quota reclamation
func AdminBetaCleanup(c *gin.Context) {
	common.SysLog("Admin: manually triggered farm beta data cleanup")
	userCount, totalReclaimed, err := model.CleanupAllBetaFarmData()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "清理失败: " + err.Error()})
		return
	}
	common.SysLog(fmt.Sprintf("Admin: farm beta cleanup done: %d users, reclaimed %d quota", userCount, totalReclaimed))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("内测数据清理完成：%d 个用户的数据已清除，回收额度 %d", userCount, totalReclaimed),
		"data": gin.H{
			"user_count":      userCount,
			"total_reclaimed": totalReclaimed,
		},
	})
}

// AdminResetAllFarmLevels resets all users' farm level to a specified value
func AdminResetAllFarmLevels(c *gin.Context) {
	var req struct {
		Level int `json:"level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Level < 1 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请提供有效的等级（>=1）"})
		return
	}
	if req.Level > common.TgBotFarmMaxLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("等级不能超过最大值 %d", common.TgBotFarmMaxLevel)})
		return
	}

	affected, err := model.ResetAllFarmLevels(req.Level)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "重置失败: " + err.Error()})
		return
	}
	common.SysLog(fmt.Sprintf("Admin: reset %d users farm level to %d", affected, req.Level))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已将 %d 个用户的农场等级重置为 Lv.%d", affected, req.Level),
		"data":    gin.H{"affected": affected, "level": req.Level},
	})
}

// AdminGetFarmUsers 获取当前真正在玩农场的用户列表
func AdminGetFarmUsers(c *gin.Context) {
	users, err := model.GetActiveFarmUsers()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "查询失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
		"total":   len(users),
	})
}
