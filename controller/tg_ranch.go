package controller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ========== 牧场动物定义 ==========

type ranchAnimalDef struct {
	Key      string
	Short    string
	Name     string
	Emoji    string
	BuyPrice *int
	GrowSecs *int64
	MeatPrice *int
}

var ranchAnimals = []ranchAnimalDef{
	{"chicken", "chi", "鸡", "🐔", &common.TgBotRanchChickenPrice, &common.TgBotRanchChickenGrowSecs, &common.TgBotRanchChickenMeatPrice},
	{"duck", "duk", "鸭", "🦆", &common.TgBotRanchDuckPrice, &common.TgBotRanchDuckGrowSecs, &common.TgBotRanchDuckMeatPrice},
	{"goose", "gos", "鹅", "🪿", &common.TgBotRanchGoosePrice, &common.TgBotRanchGooseGrowSecs, &common.TgBotRanchGooseMeatPrice},
	{"pig", "pig", "猪", "🐷", &common.TgBotRanchPigPrice, &common.TgBotRanchPigGrowSecs, &common.TgBotRanchPigMeatPrice},
	{"sheep", "shp", "羊", "🐑", &common.TgBotRanchSheepPrice, &common.TgBotRanchSheepGrowSecs, &common.TgBotRanchSheepMeatPrice},
	{"cow", "cow", "牛", "🐄", &common.TgBotRanchCowPrice, &common.TgBotRanchCowGrowSecs, &common.TgBotRanchCowMeatPrice},
}

var ranchAnimalMap map[string]*ranchAnimalDef
var ranchAnimalByShort map[string]*ranchAnimalDef

func init() {
	ranchAnimalMap = make(map[string]*ranchAnimalDef)
	ranchAnimalByShort = make(map[string]*ranchAnimalDef)
	for i := range ranchAnimals {
		ranchAnimalMap[ranchAnimals[i].Key] = &ranchAnimals[i]
		ranchAnimalByShort[ranchAnimals[i].Short] = &ranchAnimals[i]
	}
}

// ========== 状态更新 ==========

// updateRanchAnimalStatus 懒更新动物状态
func updateRanchAnimalStatus(animal *model.TgRanchAnimal) {
	if animal.Status == 5 { // 已死亡
		return
	}

	now := time.Now().Unix()
	def := ranchAnimalMap[animal.AnimalType]
	if def == nil {
		return
	}

	changed := false

	// 检查是否成熟（脏污时生长减速）
	if animal.Status == 1 {
		elapsed := now - animal.PurchasedAt
		actualGrowSecs := *def.GrowSecs
		if isAnimalDirty(animal, now) {
			penalty := int64(common.TgBotRanchManureGrowPenalty)
			actualGrowSecs = actualGrowSecs * 100 / (100 - penalty)
		}
		if elapsed >= actualGrowSecs {
			animal.Status = 2
			changed = true
		}
	}

	// 检查饥饿（断食死亡）
	if animal.Status != 5 && animal.LastFedAt > 0 {
		feedInterval := int64(common.TgBotRanchFeedInterval)
		hungerStart := animal.LastFedAt + feedInterval
		if now > hungerStart {
			deathThreshold := int64(common.TgBotRanchHungerDeathHours) * 3600
			if now-hungerStart >= deathThreshold {
				animal.Status = 5
				changed = true
			} else if animal.Status != 3 && animal.Status != 5 {
				animal.Status = 3 // hungry
				changed = true
			}
		}
	}

	// 检查口渴（断水死亡）
	if animal.Status != 5 && animal.LastWateredAt > 0 {
		waterInterval := int64(common.TgBotRanchWaterInterval)
		thirstStart := animal.LastWateredAt + waterInterval
		if now > thirstStart {
			deathThreshold := int64(common.TgBotRanchThirstDeathHours) * 3600
			if now-thirstStart >= deathThreshold {
				animal.Status = 5
				changed = true
			} else if animal.Status != 4 && animal.Status != 5 && animal.Status != 3 {
				animal.Status = 4 // thirsty
				changed = true
			}
		}
	}

	if changed {
		_ = model.UpdateRanchAnimal(animal)
	}
}

// isAnimalDirty 检查动物是否需要清理粪便
func isAnimalDirty(animal *model.TgRanchAnimal, now int64) bool {
	if animal.LastCleanedAt <= 0 {
		return false
	}
	interval := int64(common.TgBotRanchManureInterval)
	return now >= animal.LastCleanedAt+interval
}

// ========== 回调路由 ==========

func handleRanchCallback(cb *TgCallbackQuery) {
	chatId := cb.Message.Chat.Id
	msgId := cb.Message.MessageId
	tgId := strconv.FormatInt(cb.From.Id, 10)
	from := cb.From
	data := cb.Data

	switch {
	case data == "ranch":
		showRanchView(chatId, msgId, tgId, from)
	case data == "ranch_buy":
		showRanchBuyAnimals(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "ranch_ba_"):
		animalShort := strings.TrimPrefix(data, "ranch_ba_")
		doRanchBuyAnimal(chatId, msgId, tgId, animalShort, from)
	case data == "ranch_feed":
		showRanchFeedSelection(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "ranch_fd_"):
		idStr := strings.TrimPrefix(data, "ranch_fd_")
		animalId, _ := strconv.Atoi(idStr)
		doRanchFeed(chatId, msgId, tgId, animalId, from)
	case data == "ranch_water":
		showRanchWaterSelection(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "ranch_wt_"):
		idStr := strings.TrimPrefix(data, "ranch_wt_")
		animalId, _ := strconv.Atoi(idStr)
		doRanchWater(chatId, msgId, tgId, animalId, from)
	case data == "ranch_slaughter":
		showRanchSlaughterSelection(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "ranch_sl_"):
		idStr := strings.TrimPrefix(data, "ranch_sl_")
		animalId, _ := strconv.Atoi(idStr)
		doRanchSlaughter(chatId, msgId, tgId, animalId, from)
	case data == "ranch_cleanup":
		doRanchCleanup(chatId, msgId, tgId, from)
	case data == "ranch_clean":
		doRanchCleanManure(chatId, msgId, tgId, from)
	}
}

// ========== 牧场视图 ==========

func showRanchView(chatId int64, editMsgId int, tgId string, from *TgUser) {
	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	for _, a := range animals {
		updateRanchAnimalStatus(a)
	}

	text := "🐄 我的牧场\n\n"
	if len(animals) == 0 {
		text += "🏚️ 牧场空空如也，去购买动物吧！\n"
	}

	hasHungry := false
	hasMature := false
	hasDead := false
	dirtyCount := 0
	now := time.Now().Unix()
	for _, a := range animals {
		text += ranchAnimalLine(a) + "\n"
		switch a.Status {
		case 2:
			hasMature = true
		case 3, 4:
			hasHungry = true
		case 5:
			hasDead = true
		}
		if a.Status != 5 && isAnimalDirty(a, now) {
			dirtyCount++
		}
	}

	text += fmt.Sprintf("\n📊 动物 %d/%d 只", len(animals), common.TgBotRanchMaxAnimals)
	text += fmt.Sprintf("\n🌾 饲料 %s/次 | 💧 饮水 %s/次", farmQuotaStr(common.TgBotRanchFeedPrice), farmQuotaStr(common.TgBotRanchWaterPrice))
	if dirtyCount > 0 {
		text += fmt.Sprintf("\n💩 %d只动物需要清理粪便（生长减速%d%%）", dirtyCount, common.TgBotRanchManureGrowPenalty)
	}
	text += "\n"

	var rows [][]TgInlineKeyboardButton

	aliveCount := 0
	for _, a := range animals {
		if a.Status != 5 {
			aliveCount++
		}
	}

	if aliveCount < common.TgBotRanchMaxAnimals {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🛒 购买动物", CallbackData: "ranch_buy"},
		})
	}
	if hasHungry || len(animals) > 0 {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🌾 喂食", CallbackData: "ranch_feed"},
			{Text: "💧 喂水", CallbackData: "ranch_water"},
		})
	}
	if hasMature {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🔪 屠宰出售", CallbackData: "ranch_slaughter"},
		})
	}
	if dirtyCount > 0 {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🧹 清理粪便(%s)", farmQuotaStr(common.TgBotRanchManureCleanPrice)), CallbackData: "ranch_clean"},
		})
	}
	if hasDead {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🗑️ 清理死亡动物", CallbackData: "ranch_cleanup"},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func ranchAnimalLine(animal *model.TgRanchAnimal) string {
	def := ranchAnimalMap[animal.AnimalType]
	if def == nil {
		return fmt.Sprintf("❓ #%d - 未知动物", animal.Id)
	}

	now := time.Now().Unix()

	switch animal.Status {
	case 1: // growing
		elapsed := now - animal.PurchasedAt
		total := *def.GrowSecs
		dirtyTag := ""
		if isAnimalDirty(animal, now) {
			penalty := int64(common.TgBotRanchManureGrowPenalty)
			total = total * 100 / (100 - penalty)
			dirtyTag = "💩"
		}
		pct := int(elapsed * 100 / total)
		if pct > 99 {
			pct = 99
		}
		remaining := total - elapsed
		if remaining < 0 {
			remaining = 0
		}
		return fmt.Sprintf("%s %s%s - 生长中 %d%% 剩余%s", def.Emoji, def.Name, dirtyTag, pct, formatDuration(remaining))
	case 2: // mature
		dirtyTag := ""
		if isAnimalDirty(animal, now) {
			dirtyTag = "💩"
		}
		return fmt.Sprintf("✅ %s %s%s - 已成熟！可出售 %s", def.Emoji, def.Name, dirtyTag, farmQuotaStr(*def.MeatPrice))
	case 3: // hungry
		feedInterval := int64(common.TgBotRanchFeedInterval)
		hungerStart := animal.LastFedAt + feedInterval
		deathAt := hungerStart + int64(common.TgBotRanchHungerDeathHours)*3600
		remaining := deathAt - now
		if remaining < 0 {
			remaining = 0
		}
		return fmt.Sprintf("😫 %s %s - 饥饿！🌾快喂食！%s后死亡", def.Emoji, def.Name, formatDuration(remaining))
	case 4: // thirsty
		waterInterval := int64(common.TgBotRanchWaterInterval)
		thirstStart := animal.LastWateredAt + waterInterval
		deathAt := thirstStart + int64(common.TgBotRanchThirstDeathHours)*3600
		remaining := deathAt - now
		if remaining < 0 {
			remaining = 0
		}
		return fmt.Sprintf("🥵 %s %s - 口渴！💧快喂水！%s后死亡", def.Emoji, def.Name, formatDuration(remaining))
	case 5: // dead
		return fmt.Sprintf("💀 %s %s - 已死亡", def.Emoji, def.Name)
	}
	return fmt.Sprintf("%s %s", def.Emoji, def.Name)
}

// ========== 购买动物 ==========

func showRanchBuyAnimals(chatId int64, editMsgId int, tgId string, from *TgUser) {
	count, err := model.GetRanchAnimalCount(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	// 计算存活数量
	animals, _ := model.GetRanchAnimals(tgId)
	aliveCount := 0
	for _, a := range animals {
		updateRanchAnimalStatus(a)
		if a.Status != 5 {
			aliveCount++
		}
	}

	if aliveCount >= common.TgBotRanchMaxAnimals {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 牧场已满！最多养 %d 只动物。", common.TgBotRanchMaxAnimals), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	_ = count
	text := "🛒 选择要购买的动物：\n\n"
	var rows [][]TgInlineKeyboardButton
	for _, a := range ranchAnimals {
		growHours := *a.GrowSecs / 3600
		text += fmt.Sprintf("%s %s - %s | 生长%d小时 | 肉价%s\n",
			a.Emoji, a.Name, farmQuotaStr(*a.BuyPrice), growHours, farmQuotaStr(*a.MeatPrice))
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s (%s)", a.Emoji, a.Name, farmQuotaStr(*a.BuyPrice)),
				CallbackData: "ranch_ba_" + a.Short},
		})
	}
	text += fmt.Sprintf("\n📊 当前 %d/%d 只", aliveCount, common.TgBotRanchMaxAnimals)
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回牧场", CallbackData: "ranch"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doRanchBuyAnimal(chatId int64, editMsgId int, tgId string, animalShort string, from *TgUser) {
	def := ranchAnimalByShort[animalShort]
	if def == nil {
		farmSend(chatId, editMsgId, "❌ 未知动物类型", nil, from)
		return
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	// 检查存活数量
	animals, _ := model.GetRanchAnimals(tgId)
	aliveCount := 0
	for _, a := range animals {
		updateRanchAnimalStatus(a)
		if a.Status != 5 {
			aliveCount++
		}
	}
	if aliveCount >= common.TgBotRanchMaxAnimals {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 牧场已满！最多养 %d 只动物。", common.TgBotRanchMaxAnimals), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	price := *def.BuyPrice
	if user.Quota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！\n\n%s%s 价格：%s\n你的余额：%s",
			def.Emoji, def.Name, farmQuotaStr(price), farmQuotaStr(user.Quota)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "ranch_buy"}},
			},
		}, from)
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败，请稍后再试。", nil, from)
		return
	}

	now := time.Now().Unix()
	animal := &model.TgRanchAnimal{
		TelegramId:    tgId,
		AnimalType:    def.Key,
		Status:        1,
		PurchasedAt:   now,
		LastFedAt:     now,
		LastWateredAt: now,
		LastCleanedAt: now,
	}
	err = model.CreateRanchAnimal(animal)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		farmSend(chatId, editMsgId, "❌ 创建失败，已退款。", nil, from)
		return
	}

	growHours := *def.GrowSecs / 3600
	common.SysLog(fmt.Sprintf("TG Ranch: user %s bought %s for %d quota", tgId, def.Key, price))
	farmSend(chatId, editMsgId, fmt.Sprintf("🎉 购买成功！\n\n%s %s 已入栏！\n⏱️ 预计 %d 小时后成熟\n💰 花费 %s\n\n⚠️ 记得定时喂食和喂水！",
		def.Emoji, def.Name, growHours, farmQuotaStr(price)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🛒 继续购买", CallbackData: "ranch_buy"},
				{Text: "🐄 返回牧场", CallbackData: "ranch"}},
		},
	}, from)
}

// ========== 喂食 ==========

func showRanchFeedSelection(chatId int64, editMsgId int, tgId string, from *TgUser) {
	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	text := fmt.Sprintf("🌾 喂食 (%s/次)\n\n选择要喂食的动物：\n", farmQuotaStr(common.TgBotRanchFeedPrice))
	var rows [][]TgInlineKeyboardButton
	hasTarget := false
	for _, a := range animals {
		updateRanchAnimalStatus(a)
		if a.Status == 5 {
			continue
		}
		def := ranchAnimalMap[a.AnimalType]
		if def == nil {
			continue
		}
		hasTarget = true
		// 计算下次需要喂食时间
		feedInterval := int64(common.TgBotRanchFeedInterval)
		nextFeed := a.LastFedAt + feedInterval
		now := time.Now().Unix()
		label := ""
		if now >= nextFeed {
			label = fmt.Sprintf("%s %s ⚠️需喂食", def.Emoji, def.Name)
		} else {
			label = fmt.Sprintf("%s %s (剩余%s)", def.Emoji, def.Name, formatDuration(nextFeed-now))
		}
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: label, CallbackData: fmt.Sprintf("ranch_fd_%d", a.Id)},
		})
	}
	if !hasTarget {
		text += "没有需要喂食的动物。\n"
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回牧场", CallbackData: "ranch"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doRanchFeed(chatId int64, editMsgId int, tgId string, animalId int, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	var target *model.TgRanchAnimal
	for _, a := range animals {
		if a.Id == animalId {
			target = a
			break
		}
	}
	if target == nil || target.Status == 5 {
		farmSend(chatId, editMsgId, "❌ 该动物不存在或已死亡。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	price := common.TgBotRanchFeedPrice
	if user.Quota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！饲料需要 %s", farmQuotaStr(price)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败", nil, from)
		return
	}

	err = model.FeedRanchAnimal(target.Id)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		farmSend(chatId, editMsgId, "❌ 喂食失败，已退款。", nil, from)
		return
	}

	now := time.Now().Unix()
	target.LastFedAt = now
	_ = model.UpdateRanchAnimal(target)

	// 如果动物因饥饿状态需要恢复
	if target.Status == 3 {
		waterInterval := int64(common.TgBotRanchWaterInterval)
		if now > target.LastWateredAt+waterInterval {
			target.Status = 4 // 还口渴
		} else {
			// 恢复到之前状态
			def := ranchAnimalMap[target.AnimalType]
			if def != nil && now-target.PurchasedAt >= *def.GrowSecs {
				target.Status = 2
			} else {
				target.Status = 1
			}
		}
		_ = model.UpdateRanchAnimal(target)
	}

	def := ranchAnimalMap[target.AnimalType]
	name := "动物"
	emoji := "🐾"
	if def != nil {
		name = def.Name
		emoji = def.Emoji
	}
	farmSend(chatId, editMsgId, fmt.Sprintf("🌾 喂食成功！\n\n%s %s 吃饱了！\n💰 花费 %s", emoji, name, farmQuotaStr(price)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🌾 继续喂食", CallbackData: "ranch_feed"},
				{Text: "🐄 返回牧场", CallbackData: "ranch"}},
		},
	}, from)
	showRanchView(chatId, editMsgId, tgId, from)
}

// ========== 喂水 ==========

func showRanchWaterSelection(chatId int64, editMsgId int, tgId string, from *TgUser) {
	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	text := fmt.Sprintf("💧 喂水 (%s/次)\n\n选择要喂水的动物：\n", farmQuotaStr(common.TgBotRanchWaterPrice))
	var rows [][]TgInlineKeyboardButton
	hasTarget := false
	for _, a := range animals {
		updateRanchAnimalStatus(a)
		if a.Status == 5 {
			continue
		}
		def := ranchAnimalMap[a.AnimalType]
		if def == nil {
			continue
		}
		hasTarget = true
		waterInterval := int64(common.TgBotRanchWaterInterval)
		nextWater := a.LastWateredAt + waterInterval
		now := time.Now().Unix()
		label := ""
		if now >= nextWater {
			label = fmt.Sprintf("%s %s ⚠️需喂水", def.Emoji, def.Name)
		} else {
			label = fmt.Sprintf("%s %s (剩余%s)", def.Emoji, def.Name, formatDuration(nextWater-now))
		}
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: label, CallbackData: fmt.Sprintf("ranch_wt_%d", a.Id)},
		})
	}
	if !hasTarget {
		text += "没有需要喂水的动物。\n"
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回牧场", CallbackData: "ranch"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doRanchWater(chatId int64, editMsgId int, tgId string, animalId int, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	var target *model.TgRanchAnimal
	for _, a := range animals {
		if a.Id == animalId {
			target = a
			break
		}
	}
	if target == nil || target.Status == 5 {
		farmSend(chatId, editMsgId, "❌ 该动物不存在或已死亡。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	price := common.TgBotRanchWaterPrice
	if user.Quota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！饮水需要 %s", farmQuotaStr(price)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败", nil, from)
		return
	}

	err = model.WaterRanchAnimal(target.Id)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		farmSend(chatId, editMsgId, "❌ 喂水失败，已退款。", nil, from)
		return
	}

	now := time.Now().Unix()
	target.LastWateredAt = now
	_ = model.UpdateRanchAnimal(target)

	// 如果动物因口渴状态需要恢复
	if target.Status == 4 {
		feedInterval := int64(common.TgBotRanchFeedInterval)
		if now > target.LastFedAt+feedInterval {
			target.Status = 3 // 还饥饿
		} else {
			def := ranchAnimalMap[target.AnimalType]
			if def != nil && now-target.PurchasedAt >= *def.GrowSecs {
				target.Status = 2
			} else {
				target.Status = 1
			}
		}
		_ = model.UpdateRanchAnimal(target)
	}

	def := ranchAnimalMap[target.AnimalType]
	name := "动物"
	emoji := "🐾"
	if def != nil {
		name = def.Name
		emoji = def.Emoji
	}
	farmSend(chatId, editMsgId, fmt.Sprintf("💧 喂水成功！\n\n%s %s 喝饱了！\n💰 花费 %s", emoji, name, farmQuotaStr(price)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "💧 继续喂水", CallbackData: "ranch_water"},
				{Text: "🐄 返回牧场", CallbackData: "ranch"}},
		},
	}, from)
	showRanchView(chatId, editMsgId, tgId, from)
}

// ========== 屠宰出售 ==========

func showRanchSlaughterSelection(chatId int64, editMsgId int, tgId string, from *TgUser) {
	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	text := "🔪 屠宰出售\n\n选择要出售的成熟动物：\n"
	var rows [][]TgInlineKeyboardButton
	hasTarget := false
	for _, a := range animals {
		updateRanchAnimalStatus(a)
		if a.Status != 2 {
			continue
		}
		def := ranchAnimalMap[a.AnimalType]
		if def == nil {
			continue
		}
		hasTarget = true
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s → %s", def.Emoji, def.Name, farmQuotaStr(*def.MeatPrice)),
				CallbackData: fmt.Sprintf("ranch_sl_%d", a.Id)},
		})
	}
	if !hasTarget {
		text += "没有可出售的成熟动物。\n"
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回牧场", CallbackData: "ranch"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doRanchSlaughter(chatId int64, editMsgId int, tgId string, animalId int, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	var target *model.TgRanchAnimal
	for _, a := range animals {
		if a.Id == animalId {
			target = a
			break
		}
	}
	if target == nil {
		farmSend(chatId, editMsgId, "❌ 该动物不存在。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	updateRanchAnimalStatus(target)
	if target.Status != 2 {
		farmSend(chatId, editMsgId, "❌ 该动物尚未成熟，无法屠宰。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	def := ranchAnimalMap[target.AnimalType]
	if def == nil {
		farmSend(chatId, editMsgId, "❌ 未知动物类型", nil, from)
		return
	}

	meatPrice := *def.MeatPrice

	// 删除动物
	err = model.DeleteRanchAnimal(target.Id)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 操作失败", nil, from)
		return
	}

	// 增加收入
	err = model.IncreaseUserQuota(user.Id, meatPrice, true)
	if err != nil {
		// 尝试恢复动物
		_ = model.CreateRanchAnimal(target)
		farmSend(chatId, editMsgId, "❌ 收入到账失败，已恢复动物。", nil, from)
		return
	}

	common.SysLog(fmt.Sprintf("TG Ranch: user %s slaughtered %s for %d quota", tgId, def.Key, meatPrice))
	farmSend(chatId, editMsgId, fmt.Sprintf("🔪 屠宰成功！\n\n%s %s 已出售！\n💰 收入 %s",
		def.Emoji, def.Name, farmQuotaStr(meatPrice)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔪 继续出售", CallbackData: "ranch_slaughter"},
				{Text: "🐄 返回牧场", CallbackData: "ranch"}},
		},
	}, from)
}

// ========== 清理死亡动物 ==========

func doRanchCleanup(chatId int64, editMsgId int, tgId string, from *TgUser) {
	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	cleaned := 0
	for _, a := range animals {
		updateRanchAnimalStatus(a)
		if a.Status == 5 {
			_ = model.DeleteRanchAnimal(a.Id)
			cleaned++
		}
	}

	if cleaned == 0 {
		farmSend(chatId, editMsgId, "没有需要清理的死亡动物。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	farmSend(chatId, editMsgId, fmt.Sprintf("🗑️ 已清理 %d 只死亡动物。", cleaned), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🐄 返回牧场", CallbackData: "ranch"}},
		},
	}, from)
}

// ========== 清理粪便 ==========

func doRanchCleanManure(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	animals, err := model.GetRanchAnimals(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	now := time.Now().Unix()
	dirtyCount := 0
	for _, a := range animals {
		if a.Status != 5 && isAnimalDirty(a, now) {
			dirtyCount++
		}
	}

	if dirtyCount == 0 {
		farmSend(chatId, editMsgId, "✨ 牧场很干净，不需要清理！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🐄 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	price := common.TgBotRanchManureCleanPrice
	if user.Quota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！清理粪便需要 %s", farmQuotaStr(price)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回牧场", CallbackData: "ranch"}},
			},
		}, from)
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败", nil, from)
		return
	}

	err = model.CleanRanchAnimals(tgId)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		farmSend(chatId, editMsgId, "❌ 清理失败，已退款。", nil, from)
		return
	}

	common.SysLog(fmt.Sprintf("TG Ranch: user %s cleaned manure for %d animals, cost %d", tgId, dirtyCount, price))
	farmSend(chatId, editMsgId, fmt.Sprintf("🧹 清理完成！\n\n✨ 为 %d 只动物清理了粪便\n💰 花费 %s\n\n动物们生长速度恢复正常！",
		dirtyCount, farmQuotaStr(price)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🐄 返回牧场", CallbackData: "ranch"}},
		},
	}, from)
}
