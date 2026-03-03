package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TG Bot Webhook 数据结构
type TgUpdate struct {
	UpdateId int       `json:"update_id"`
	Message  *TgMsg    `json:"message"`
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

// TgBotWebhook 处理 Telegram Bot Webhook 请求
func TgBotWebhook(c *gin.Context) {
	var update TgUpdate
	if err := common.DecodeJson(c.Request.Body, &update); err != nil {
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
		sendTgMessage(chatId, "👋 欢迎使用 "+common.SystemName+" 机器人！\n\n"+
			"可用命令：\n"+
			"/claim - 领取账户和额度\n"+
			"/myinfo - 查看我的账户信息\n\n"+
			"每人仅限领取一次。")
	case text == "/claim" || text == "/领取":
		handleTgClaim(msg)
	case text == "/myinfo" || text == "/我的信息":
		handleTgMyInfo(msg)
	case strings.HasPrefix(text, "/redeem ") || strings.HasPrefix(text, "/兑换 "):
		parts := strings.SplitN(text, " ", 2)
		if len(parts) == 2 {
			handleTgRedeem(msg, strings.TrimSpace(parts[1]))
		} else {
			sendTgMessage(chatId, "❌ 请提供兑换码，例如：/redeem your_code")
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleTgClaim 处理领取命令
func handleTgClaim(msg *TgMsg) {
	chatId := msg.Chat.Id
	tgId := strconv.FormatInt(msg.From.Id, 10)
	tgUsername := msg.From.Username
	if tgUsername == "" {
		tgUsername = msg.From.FirstName
	}

	// 检查用户是否已存在（已领取过）
	var existingUser model.User
	err := model.DB.Where("telegram_id = ?", tgId).First(&existingUser).Error
	if err == nil {
		sendTgMessage(chatId, "⚠️ 你已经领取过了！\n\n"+
			"🔑 用户名：`"+existingUser.Username+"`\n"+
			"请使用之前领取时的密码登录。\n\n"+
			"如需兑换额度，请使用：/redeem 兑换码")
		return
	}

	// 查找可用的邀请码（注册码）
	var regCode model.Redemption
	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}
	_ = keyCol

	err = model.DB.Where("purpose = ? AND status = ?",
		common.RedemptionPurposeRegistration, common.RedemptionCodeStatusEnabled).
		Order("id asc").First(&regCode).Error
	if err != nil {
		sendTgMessage(chatId, "❌ 暂无可用的邀请码，请联系管理员添加。")
		return
	}

	// 生成用户名和密码
	username := fmt.Sprintf("tg_%s", tgId)
	password := common.GetRandomString(12)
	displayName := tgUsername
	if len(displayName) > 20 {
		displayName = displayName[:20]
	}

	// 创建用户
	user := &model.User{
		Username:    username,
		DisplayName: displayName,
		Password:    password,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		TelegramId:  tgId,
	}
	if err := user.Insert(0); err != nil {
		// 用户名可能冲突，尝试加后缀
		username = fmt.Sprintf("tg_%s_%s", tgId, common.GetRandomString(4))
		user.Username = username
		if err := user.Insert(0); err != nil {
			sendTgMessage(chatId, "❌ 创建账户失败，请稍后再试。")
			return
		}
	}

	// 消费邀请码
	_, consumeErr := model.ConsumeRedemptionCodeForRegistration(regCode.Key, user.Id)
	if consumeErr != nil {
		common.SysError(fmt.Sprintf("TG Bot: consume registration code failed for user %s: %v", username, consumeErr))
	}

	// 查找可用的余额兑换码并自动兑换
	var redeemCode model.Redemption
	err = model.DB.Where("purpose IN ? AND status = ?",
		[]int{common.RedemptionPurposeLegacy, common.RedemptionPurposeTopUp},
		common.RedemptionCodeStatusEnabled).
		Order("id asc").First(&redeemCode).Error

	quotaMsg := ""
	if err == nil {
		quota, redeemErr := model.Redeem(redeemCode.Key, user.Id)
		if redeemErr == nil {
			quotaMsg = fmt.Sprintf("\n💰 已自动充值额度：%s", formatQuota(quota))
		}
	}

	// 发送成功消息
	sendTgMessage(chatId, "✅ 账户创建成功！\n\n"+
		"🔑 用户名：`"+username+"`\n"+
		"🔒 密码：`"+password+"`\n"+
		quotaMsg+"\n\n"+
		"⚠️ 请立即保存以上信息！密码不会再次显示。\n"+
		"如需兑换更多额度，请使用：/redeem 兑换码")
}

// handleTgMyInfo 处理查看信息命令
func handleTgMyInfo(msg *TgMsg) {
	chatId := msg.Chat.Id
	tgId := strconv.FormatInt(msg.From.Id, 10)

	var user model.User
	err := model.DB.Where("telegram_id = ?", tgId).First(&user).Error
	if err != nil {
		sendTgMessage(chatId, "❌ 你还没有领取账户，请先使用 /claim 领取。")
		return
	}

	sendTgMessage(chatId, fmt.Sprintf("📊 你的账户信息：\n\n"+
		"👤 用户名：`%s`\n"+
		"💰 剩余额度：%s\n"+
		"📈 已用额度：%s\n"+
		"🔢 请求次数：%d",
		user.Username,
		formatQuota(user.Quota),
		formatQuota(user.UsedQuota),
		user.RequestCount))
}

// handleTgRedeem 处理兑换码命令
func handleTgRedeem(msg *TgMsg, code string) {
	chatId := msg.Chat.Id
	tgId := strconv.FormatInt(msg.From.Id, 10)

	var user model.User
	err := model.DB.Where("telegram_id = ?", tgId).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sendTgMessage(chatId, "❌ 你还没有领取账户，请先使用 /claim 领取。")
		} else {
			sendTgMessage(chatId, "❌ 查询账户失败，请稍后再试。")
		}
		return
	}

	quota, redeemErr := model.Redeem(code, user.Id)
	if redeemErr != nil {
		sendTgMessage(chatId, "❌ 兑换失败：兑换码无效或已被使用。")
		return
	}

	sendTgMessage(chatId, fmt.Sprintf("✅ 兑换成功！\n\n💰 充值额度：%s\n💰 当前余额：%s",
		formatQuota(quota), formatQuota(user.Quota+quota)))
}

// formatQuota 格式化额度显示
func formatQuota(quota int) string {
	if common.DisplayInCurrencyEnabled {
		return fmt.Sprintf("$%.4f", float64(quota)/common.QuotaPerUnit)
	}
	return fmt.Sprintf("%d", quota)
}

// sendTgMessage 通过 Telegram Bot API 发送消息
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

	bodyBytes, err := common.Marshal(body)
	if err != nil {
		common.SysError("TG Bot: marshal message failed: " + err.Error())
		return
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		common.SysError("TG Bot: send message failed: " + err.Error())
		return
	}
	defer resp.Body.Close()
}
