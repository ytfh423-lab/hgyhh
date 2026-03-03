package controller

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ========== 农场游戏定义 ==========

type farmCropDef struct {
	Key       string
	Short     string // callback abbreviation
	Name      string
	Emoji     string
	SeedCost  int   // quota units
	GrowSecs  int64 // seconds to grow
	MaxYield  int   // max harvest yield count
	UnitPrice int   // quota per unit harvested
}

type farmItemDef struct {
	Key   string
	Name  string
	Emoji string
	Cost  int    // quota units
	Cures string // event type it cures
}

// NOTE: all emojis below must be Unicode 6.0-11.0 for wide compatibility
var farmCrops = []farmCropDef{
	{"carrot", "car", "胡萝卜", "🌰", 50000, 1800, 2, 170000},
	{"tomato", "tom", "番茄", "🍅", 150000, 3600, 5, 135000},
	{"pumpkin", "pum", "南瓜", "🎃", 350000, 7200, 6, 250000},
	{"blueberry", "blu", "蓝莓", "🍇", 75000, 10800, 25, 10000},
	{"strawberry", "str", "草莓", "🍓", 750000, 14400, 6, 470000},
	{"watermelon", "wat", "西瓜", "🍉", 1250000, 21600, 8, 535000},
	{"mango", "man", "芒果", "🍊", 75000, 25200, 50, 5000},
	{"corn", "cor", "玉米", "🌽", 50000, 54000, 20, 10000},
}

var farmItems = []farmItemDef{
	{"pesticide", "杀虫剂", "🧪", 150000, "bugs"},
	{"fertilizer", "化肥", "🧴", 200000, ""},
	{"dogfood", "狗粮", "🦴", 500000, ""},
}

var farmCropMap map[string]*farmCropDef
var farmCropByShort map[string]*farmCropDef
var farmItemMap map[string]*farmItemDef

const farmMaxSteals = 2

func init() {
	farmCropMap = make(map[string]*farmCropDef)
	farmCropByShort = make(map[string]*farmCropDef)
	for i := range farmCrops {
		farmCropMap[farmCrops[i].Key] = &farmCrops[i]
		farmCropByShort[farmCrops[i].Short] = &farmCrops[i]
	}
	farmItemMap = make(map[string]*farmItemDef)
	for i := range farmItems {
		farmItemMap[farmItems[i].Key] = &farmItems[i]
	}
}

func farmQuotaStr(quota int) string {
	return fmt.Sprintf("$%.2f", float64(quota)/common.QuotaPerUnit)
}

// ========== 状态懒更新 ==========

func updateFarmPlotStatus(plot *model.TgFarmPlot) {
	if plot.Status == 0 || plot.Status == 2 {
		return
	}
	// 状态4(枯萎)检查是否死亡
	if plot.Status == 4 {
		now := time.Now().Unix()
		wiltDuration := int64(common.TgBotFarmWiltDuration)
		if plot.LastWateredAt > 0 {
			waterInterval := int64(common.TgBotFarmWaterInterval)
			wiltStart := plot.LastWateredAt + waterInterval
			if now >= wiltStart+wiltDuration {
				// 死亡：自动清空地块
				_ = model.ClearFarmPlot(plot.Id)
				plot.Status = 0
				plot.CropType = ""
			}
		}
		return
	}
	if plot.Status != 1 && plot.Status != 3 {
		return
	}
	now := time.Now().Unix()
	crop := farmCropMap[plot.CropType]
	if crop == nil {
		return
	}
	changed := false

	// 浇水检查：生长中的作物需要定期浇水
	if plot.Status == 1 && plot.LastWateredAt > 0 {
		waterInterval := int64(common.TgBotFarmWaterInterval)
		if now >= plot.LastWateredAt+waterInterval {
			// 枯萎
			plot.Status = 4
			changed = true
			if changed {
				_ = model.UpdateFarmPlot(plot)
			}
			return
		}
	}

	// 事件触发优先
	if plot.Status == 1 && plot.EventAt > 0 && plot.EventType != "" && now >= plot.EventAt {
		plot.Status = 3
		changed = true
	}
	// 天灾(干旱)死亡检查：status=3 + drought + 超时未处理
	if plot.Status == 3 && plot.EventType == "drought" {
		wiltDuration := int64(common.TgBotFarmWiltDuration)
		if now >= plot.EventAt+wiltDuration {
			_ = model.ClearFarmPlot(plot.Id)
			plot.Status = 0
			plot.CropType = ""
			return
		}
	}
	// 成熟检查（无事件时）
	if plot.Status == 1 {
		matureAt := plot.PlantedAt + crop.GrowSecs
		if now >= matureAt {
			plot.Status = 2
			changed = true
		}
	}
	if changed {
		_ = model.UpdateFarmPlot(plot)
	}
}

// ========== 用户绑定 ==========

func getFarmUser(tgId string) (*model.User, error) {
	user := &model.User{TelegramId: tgId}
	err := user.FillUserByTelegramId()
	return user, err
}

func farmBindingError(chatId int64, editMsgId int, from *TgUser) {
	text := "🔑 你还没有绑定平台账号！\n\n" +
		"请先私聊机器人发送你的 API Key（以 sk- 开头）完成绑定。\n" +
		"绑定后才能使用农场功能。\n\n" +
		"发送 /bindaccount 查看绑定说明。"
	farmSend(chatId, editMsgId, text, nil, from)
}

// ========== 命令入口 ==========

func handleFarmCommand(chatId int64, from *TgUser, isGroup bool) {
	if !isGroup {
		sendTgMessage(chatId, "🌾 农场游戏仅限群组中使用！\n\n请在群组里发送 /farm 开始种菜。\n私聊仅支持绑定账号功能。", from)
		return
	}
	tgId := strconv.FormatInt(from.Id, 10)
	if _, err := getFarmUser(tgId); err != nil {
		farmBindingError(chatId, 0, from)
		return
	}
	showFarmView(chatId, 0, tgId, from)
}

func handleFarmCallback(cb *TgCallbackQuery) {
	chatId := cb.Message.Chat.Id
	msgId := cb.Message.MessageId
	tgId := strconv.FormatInt(cb.From.Id, 10)
	data := cb.Data

	// 统一绑定检查：所有农场操作都需要绑定账号
	from := cb.From
	if _, err := getFarmUser(tgId); err != nil {
		farmBindingError(chatId, msgId, from)
		return
	}

	switch {
	case data == "farm":
		showFarmView(chatId, msgId, tgId, from)
	case data == "farm_plant":
		showFarmPlantCrops(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_p_"):
		cropShort := strings.TrimPrefix(data, "farm_p_")
		showFarmPlotSelection(chatId, msgId, tgId, cropShort, from)
	case strings.HasPrefix(data, "farm_pp_"):
		parts := strings.SplitN(strings.TrimPrefix(data, "farm_pp_"), "_", 2)
		if len(parts) == 2 {
			plotIdx, _ := strconv.Atoi(parts[0])
			doFarmPlant(chatId, msgId, tgId, plotIdx, parts[1], from)
		}
	case data == "farm_harvest":
		doFarmHarvest(chatId, msgId, tgId, from)
	case data == "farm_shop":
		showFarmShop(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_buy_"):
		itemKey := strings.TrimPrefix(data, "farm_buy_")
		doFarmBuy(chatId, msgId, tgId, itemKey, from)
	case data == "farm_steal":
		showFarmStealTargets(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_st_"):
		victimId := strings.TrimPrefix(data, "farm_st_")
		doFarmSteal(chatId, msgId, tgId, victimId, from)
	case data == "farm_treat":
		showFarmTreatSelection(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_tr_"):
		plotStr := strings.TrimPrefix(data, "farm_tr_")
		plotIdx, _ := strconv.Atoi(plotStr)
		doFarmTreat(chatId, msgId, tgId, plotIdx, from)
	case data == "farm_fert":
		showFarmFertSelection(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_ff_"):
		plotStr := strings.TrimPrefix(data, "farm_ff_")
		plotIdx, _ := strconv.Atoi(plotStr)
		doFarmFertilize(chatId, msgId, tgId, plotIdx, from)
	case data == "farm_buyland":
		doFarmBuyLand(chatId, msgId, tgId, from)
	case data == "farm_water":
		showFarmWaterSelection(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_ww_"):
		plotStr := strings.TrimPrefix(data, "farm_ww_")
		plotIdx, _ := strconv.Atoi(plotStr)
		doFarmWater(chatId, msgId, tgId, plotIdx, from)
	case data == "farm_dog":
		showFarmDog(chatId, msgId, tgId, from)
	case data == "farm_buydog":
		doFarmBuyDog(chatId, msgId, tgId, from)
	case data == "farm_feeddog":
		doFarmFeedDog(chatId, msgId, tgId, from)
	}
}

// ========== 农场视图 ==========

func showFarmView(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	text := "🌾 我的农场\n\n"
	hasEvent := false
	hasWiltOrGrowing := false
	for _, plot := range plots {
		text += farmPlotLine(plot) + "\n"
		if plot.Status == 3 && plot.EventType != "drought" {
			hasEvent = true
		}
		if plot.Status == 1 || plot.Status == 4 ||
			(plot.Status == 3 && plot.EventType == "drought") {
			hasWiltOrGrowing = true
		}
	}

	// 狗狗信息
	dog, dogErr := model.GetFarmDog(tgId)
	if dogErr == nil {
		model.UpdateDogHunger(dog)
		dogLevel := "🐶 幼犬"
		if dog.Level == 2 {
			dogLevel = "🐕 成犬"
		}
		guardStatus := ""
		if dog.Level == 2 && dog.Hunger > 0 {
			guardStatus = " ✅看门中"
		} else if dog.Hunger == 0 {
			guardStatus = " ❌饿坏了"
		} else {
			guardStatus = " ⏳成长中"
		}
		text += fmt.Sprintf("\n🐕 %s「%s」 饱食度:%d%%%s\n", dogLevel, dog.Name, dog.Hunger, guardStatus)
	}

	items, _ := model.GetFarmItems(tgId)
	if len(items) > 0 {
		text += "\n📦 背包："
		for _, item := range items {
			def := farmItemMap[item.ItemType]
			if def != nil {
				text += fmt.Sprintf(" %s%s×%d", def.Emoji, def.Name, item.Quantity)
			}
		}
		text += "\n"
	}

	// 显示地块数量
	text += fmt.Sprintf("\n📊 土地 %d/%d 块", len(plots), model.FarmMaxPlots)
	if len(plots) < model.FarmMaxPlots {
		text += fmt.Sprintf(" | 购买新地 %s", farmQuotaStr(common.TgBotFarmPlotPrice))
	}
	text += "\n"

	var rows [][]TgInlineKeyboardButton
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🌱 种植", CallbackData: "farm_plant"},
		{Text: "🌾 收获", CallbackData: "farm_harvest"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🏪 商店", CallbackData: "farm_shop"},
		{Text: "🕵️ 偷菜", CallbackData: "farm_steal"},
	})
	// 浇水按钮
	if hasWiltOrGrowing {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "💧 浇水", CallbackData: "farm_water"},
		})
	}
	// 有生长中作物时显示施肥按钮
	hasGrowing := false
	for _, plot := range plots {
		if plot.Status == 1 && plot.Fertilized == 0 {
			hasGrowing = true
			break
		}
	}
	if hasGrowing {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🧴 施肥", CallbackData: "farm_fert"},
		})
	}
	if hasEvent {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "💊 治疗", CallbackData: "farm_treat"},
		})
	}
	// 狗狗按钮
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🐕 狗狗", CallbackData: "farm_dog"},
	})
	if len(plots) < model.FarmMaxPlots {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🏗️ 购买土地 (%s)", farmQuotaStr(common.TgBotFarmPlotPrice)), CallbackData: "farm_buyland"},
		})
	}
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func farmPlotLine(plot *model.TgFarmPlot) string {
	idx := plot.PlotIndex + 1
	switch plot.Status {
	case 0:
		return fmt.Sprintf("⬜ %d号地 - 空地", idx)
	case 1:
		crop := farmCropMap[plot.CropType]
		if crop == nil {
			return fmt.Sprintf("⬜ %d号地 - 空地", idx)
		}
		now := time.Now().Unix()
		elapsed := now - plot.PlantedAt
		total := crop.GrowSecs
		pct := int(elapsed * 100 / total)
		if pct > 99 {
			pct = 99
		}
		remaining := total - elapsed
		fertTag := ""
		if plot.Fertilized == 1 {
			fertTag = " 🧴"
		}
		// 浇水倒计时
		waterTag := ""
		if plot.LastWateredAt > 0 {
			waterInterval := int64(common.TgBotFarmWaterInterval)
			nextWater := plot.LastWateredAt + waterInterval - now
			if nextWater > 0 {
				waterTag = fmt.Sprintf(" 💧%s", formatDuration(nextWater))
			} else {
				waterTag = " 💧⚠️需浇水"
			}
		}
		return fmt.Sprintf("%s %d号地 - %s 生长中 %d%% 剩余%s%s%s", crop.Emoji, idx, crop.Name, pct, formatDuration(remaining), fertTag, waterTag)
	case 2:
		crop := farmCropMap[plot.CropType]
		if crop == nil {
			return fmt.Sprintf("✅ %d号地 - 已成熟", idx)
		}
		stolen := ""
		if plot.StolenCount > 0 {
			stolen = fmt.Sprintf(" ⚠️被偷%d次", plot.StolenCount)
		}
		return fmt.Sprintf("✅ %d号地 - %s%s 已成熟！%s", crop.Emoji, crop.Name, stolen, "")
	case 3:
		crop := farmCropMap[plot.CropType]
		emoji := "❓"
		name := "未知"
		if crop != nil {
			emoji = crop.Emoji
			name = crop.Name
		}
		if plot.EventType == "drought" {
			now := time.Now().Unix()
			wiltDuration := int64(common.TgBotFarmWiltDuration)
			deathAt := plot.EventAt + wiltDuration
			remaining := deathAt - now
			if remaining < 0 {
				remaining = 0
			}
			return fmt.Sprintf("🏜️ %d号地 - %s%s 天灾干旱！💧快浇水救命！%s后死亡", idx, emoji, name, formatDuration(remaining))
		}
		eventEmoji := "❌"
		eventLabel := "未知事件"
		switch plot.EventType {
		case "bugs":
			eventEmoji = "🐛"
			eventLabel = "虫害"
		}
		return fmt.Sprintf("%s %d号地 - %s %s%s！需要治疗", emoji, idx, name, eventEmoji, eventLabel)
	case 4:
		crop := farmCropMap[plot.CropType]
		emoji := "🥀"
		name := "作物"
		if crop != nil {
			emoji = crop.Emoji
			name = crop.Name
		}
		now := time.Now().Unix()
		wiltDuration := int64(common.TgBotFarmWiltDuration)
		waterInterval := int64(common.TgBotFarmWaterInterval)
		deathAt := plot.LastWateredAt + waterInterval + wiltDuration
		remaining := deathAt - now
		if remaining < 0 {
			remaining = 0
		}
		return fmt.Sprintf("🥀 %d号地 - %s%s 枯萎中！💧快浇水！%s后死亡", idx, emoji, name, formatDuration(remaining))
	}
	return fmt.Sprintf("❓ %d号地", idx)
}

func formatDuration(secs int64) string {
	if secs <= 0 {
		return "0分"
	}
	hours := secs / 3600
	mins := (secs % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%d时%d分", hours, mins)
	}
	return fmt.Sprintf("%d分", mins)
}

// ========== 种植 ==========

func showFarmPlantCrops(chatId int64, editMsgId int, tgId string, from *TgUser) {
	text := "🌱 选择要种植的作物：\n\n"
	var rows [][]TgInlineKeyboardButton
	for _, crop := range farmCrops {
		maxValue := crop.MaxYield * crop.UnitPrice
		text += fmt.Sprintf("%s %s - 种子%s | %s | 产量1~%d×%s | 最高%s\n",
			crop.Emoji, crop.Name, farmQuotaStr(crop.SeedCost),
			formatDuration(crop.GrowSecs), crop.MaxYield,
			farmQuotaStr(crop.UnitPrice), farmQuotaStr(maxValue))
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s (%s)", crop.Emoji, crop.Name, farmQuotaStr(crop.SeedCost)),
				CallbackData: "farm_p_" + crop.Short},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func showFarmPlotSelection(chatId int64, editMsgId int, tgId string, cropShort string, from *TgUser) {
	crop := farmCropByShort[cropShort]
	if crop == nil {
		farmSend(chatId, editMsgId, "❌ 未知作物", nil, from)
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}
	text := fmt.Sprintf("🌱 种植 %s%s\n选择空地：\n", crop.Emoji, crop.Name)
	var rows [][]TgInlineKeyboardButton
	hasEmpty := false
	for _, plot := range plots {
		if plot.Status == 0 {
			hasEmpty = true
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("⬜ %d号地", plot.PlotIndex+1),
					CallbackData: fmt.Sprintf("farm_pp_%d_%s", plot.PlotIndex, cropShort)},
			})
		}
	}
	if !hasEmpty {
		text += "\n❌ 没有空地了！请先收获或清理。"
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回", CallbackData: "farm_plant"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func doFarmPlant(chatId int64, editMsgId int, tgId string, plotIdx int, cropShort string, from *TgUser) {
	crop := farmCropByShort[cropShort]
	if crop == nil {
		farmSend(chatId, editMsgId, "❌ 未知作物", nil, from)
		return
	}
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}
	if user.Quota < crop.SeedCost {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！种子需要 %s，当前余额 %s",
			farmQuotaStr(crop.SeedCost), farmQuotaStr(user.Quota)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_plant"}},
			},
		}, from)
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	var targetPlot *model.TgFarmPlot
	for _, p := range plots {
		if p.PlotIndex == plotIdx {
			targetPlot = p
			break
		}
	}
	if targetPlot == nil || targetPlot.Status != 0 {
		farmSend(chatId, editMsgId, "❌ 该地块不可用", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_plant"}},
			},
		}, from)
		return
	}
	err = model.DecreaseUserQuota(user.Id, crop.SeedCost)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败，请稍后再试", nil, from)
		return
	}

	now := time.Now().Unix()
	targetPlot.CropType = crop.Key
	targetPlot.PlantedAt = now
	targetPlot.Status = 1
	targetPlot.EventType = ""
	targetPlot.EventAt = 0
	targetPlot.StolenCount = 0
	targetPlot.LastWateredAt = now

	// 虫害事件
	if rand.Intn(100) < common.TgBotFarmEventChance {
		targetPlot.EventType = "bugs"
		offset := crop.GrowSecs * int64(30+rand.Intn(50)) / 100
		targetPlot.EventAt = now + offset
	}
	// 天灾(干旱)：独立概率，不与虫害叠加
	if targetPlot.EventType == "" && rand.Intn(100) < common.TgBotFarmDisasterChance {
		targetPlot.EventType = "drought"
		offset := crop.GrowSecs * int64(30+rand.Intn(50)) / 100
		targetPlot.EventAt = now + offset
	}

	_ = model.UpdateFarmPlot(targetPlot)
	common.SysLog(fmt.Sprintf("TG Farm: user %s planted %s on plot %d, cost %d", tgId, crop.Key, plotIdx, crop.SeedCost))
	showFarmView(chatId, editMsgId, tgId, from)
}

// ========== 收获 ==========

func doFarmHarvest(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	totalQuota := 0
	harvestedCount := 0
	details := ""
	for _, plot := range plots {
		if plot.Status == 2 {
			crop := farmCropMap[plot.CropType]
			if crop == nil {
				continue
			}
			// 随机产量：1 ~ MaxYield
			yield := 1 + rand.Intn(crop.MaxYield)
			// 化肥加成：+50%
			fertBonus := 0
			if plot.Fertilized == 1 {
				fertBonus = yield / 2
				if fertBonus < 1 {
					fertBonus = 1
				}
				yield += fertBonus
			}
			// 被偷损失
			loss := plot.StolenCount
			realYield := yield - loss
			if realYield < 0 {
				realYield = 0
			}
			value := realYield * crop.UnitPrice
			totalQuota += value
			harvestedCount++

			details += fmt.Sprintf("\n%s %s: 产量%d", crop.Emoji, crop.Name, yield-fertBonus)
			if fertBonus > 0 {
				details += fmt.Sprintf(" +化肥%d", fertBonus)
			}
			if loss > 0 {
				details += fmt.Sprintf(" -被偷%d", loss)
			}
			details += fmt.Sprintf(" = 实收%d × %s = %s",
				realYield, farmQuotaStr(crop.UnitPrice), farmQuotaStr(value))

			_ = model.ClearFarmPlot(plot.Id)
		}
	}

	if harvestedCount == 0 {
		farmSend(chatId, editMsgId, "🌾 没有可收获的作物。\n\n种植作物并等待成熟后即可收获！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🌱 去种植", CallbackData: "farm_plant"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	err = model.IncreaseUserQuota(user.Id, totalQuota, true)
	if err != nil {
		common.SysError(fmt.Sprintf("TG Farm: increase quota failed for user %d: %s", user.Id, err.Error()))
	}
	common.SysLog(fmt.Sprintf("TG Farm: user %s harvested %d crops, total %d quota", tgId, harvestedCount, totalQuota))

	text := fmt.Sprintf("🌾 收获完成！\n%s\n\n💰 共获得 %s 额度", details, farmQuotaStr(totalQuota))
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 商店 ==========

func showFarmShop(chatId int64, editMsgId int, tgId string, from *TgUser) {
	text := "🏪 农场商店\n\n"
	text += "📌 种子（在「种植」中直接购买并种下）：\n"
	for _, crop := range farmCrops {
		text += fmt.Sprintf("  %s %s - %s | %s | 产量1~%d×%s\n",
			crop.Emoji, crop.Name, farmQuotaStr(crop.SeedCost),
			formatDuration(crop.GrowSecs), crop.MaxYield, farmQuotaStr(crop.UnitPrice))
	}
	text += "\n📌 道具：\n"
	var rows [][]TgInlineKeyboardButton
	for _, item := range farmItems {
		itemCost := item.Cost
		if item.Key == "dogfood" {
			itemCost = common.TgBotFarmDogFoodPrice
		}
		if item.Cures != "" {
			cureLabel := farmEventLabel(item.Cures)
			text += fmt.Sprintf("  %s %s - %s (治疗%s)\n", item.Emoji, item.Name, farmQuotaStr(itemCost), cureLabel)
		} else if item.Key == "dogfood" {
			text += fmt.Sprintf("  %s %s - %s (喂狗)\n", item.Emoji, item.Name, farmQuotaStr(itemCost))
		} else if item.Key == "fertilizer" {
			text += fmt.Sprintf("  %s %s - %s (施肥增产50%%)\n", item.Emoji, item.Name, farmQuotaStr(itemCost))
		} else {
			text += fmt.Sprintf("  %s %s - %s\n", item.Emoji, item.Name, farmQuotaStr(itemCost))
		}
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("%s 购买%s (%s)", item.Emoji, item.Name, farmQuotaStr(itemCost)),
				CallbackData: "farm_buy_" + item.Key},
		})
	}
	// 购买狗狗
	_, dogErr := model.GetFarmDog(tgId)
	if dogErr != nil {
		text += fmt.Sprintf("\n🐕 看门狗\n  🐶 小狗 - %s (长大后可拦截偷菜)\n", farmQuotaStr(common.TgBotFarmDogPrice))
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🐶 购买小狗 (%s)", farmQuotaStr(common.TgBotFarmDogPrice)),
				CallbackData: "farm_buydog"},
		})
	}
	text += "\n💡 种子直接在「🌱 种植」中购买"
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🌱 去种植", CallbackData: "farm_plant"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func doFarmBuy(chatId int64, editMsgId int, tgId string, itemKey string, from *TgUser) {
	item := farmItemMap[itemKey]
	if item == nil {
		farmSend(chatId, editMsgId, "❌ 未知道具", nil, from)
		return
	}
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}
	cost := item.Cost
	if itemKey == "dogfood" {
		cost = common.TgBotFarmDogFoodPrice
	}
	if user.Quota < cost {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！需要 %s", farmQuotaStr(cost)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回商店", CallbackData: "farm_shop"}},
			},
		}, from)
		return
	}
	err = model.DecreaseUserQuota(user.Id, cost)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败", nil, from)
		return
	}
	err = model.IncrementFarmItem(tgId, itemKey, 1)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, cost, true)
		farmSend(chatId, editMsgId, "❌ 购买失败", nil, from)
		return
	}
	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 购买 %s%s 成功！已扣除 %s",
		item.Emoji, item.Name, farmQuotaStr(cost)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🏪 继续购物", CallbackData: "farm_shop"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 偷菜 ==========

func showFarmStealTargets(chatId int64, editMsgId int, tgId string, from *TgUser) {
	targets, err := model.GetMatureFarmTargets(tgId)
	if err != nil || len(targets) == 0 {
		farmSend(chatId, editMsgId, "🕵️ 暂时没有可偷的菜地。\n\n等其他玩家的作物成熟后再来！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}
	text := "🕵️ 可偷菜的农场：\n\n"
	var rows [][]TgInlineKeyboardButton
	for _, t := range targets {
		masked := maskTgId(t.TelegramId)
		text += fmt.Sprintf("👤 %s - %d块成熟的地\n", masked, t.Count)
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🕵️ 偷 %s 的菜", masked),
				CallbackData: "farm_st_" + t.TelegramId},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func doFarmSteal(chatId int64, editMsgId int, tgId string, victimId string, from *TgUser) {
	if tgId == victimId {
		farmSend(chatId, editMsgId, "❌ 不能偷自己的菜！", nil, from)
		return
	}
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	now := time.Now().Unix()
	recentSteals, _ := model.CountRecentSteals(tgId, victimId, now-int64(common.TgBotFarmStealCooldown))
	if recentSteals > 0 {
		cooldownMin := common.TgBotFarmStealCooldown / 60
		farmSend(chatId, editMsgId, fmt.Sprintf("⏳ 冷却中！%d分钟内只能偷同一人一次。", cooldownMin), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🕵️ 看看别人", CallbackData: "farm_steal"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 检查对方是否有看门狗
	victimDog, dogErr := model.GetFarmDog(victimId)
	if dogErr == nil {
		model.UpdateDogHunger(victimDog)
		if victimDog.Level == 2 && victimDog.Hunger > 0 {
			// 成犬且未饿坏：有概率拦截
			if rand.Intn(100) < common.TgBotFarmDogGuardRate {
				farmSend(chatId, editMsgId, fmt.Sprintf("🐕 %s 的看门狗「%s」发现了你，偷菜失败！",
					maskTgId(victimId), victimDog.Name), &TgInlineKeyboardMarkup{
					InlineKeyboard: [][]TgInlineKeyboardButton{
						{{Text: "🕵️ 看看别人", CallbackData: "farm_steal"},
							{Text: "🔙 返回", CallbackData: "farm"}},
					},
				}, from)
				return
			}
		}
	}

	plots, err := model.GetStealablePlots(victimId)
	if err != nil || len(plots) == 0 {
		farmSend(chatId, editMsgId, "❌ 该玩家没有可偷的成熟作物了。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🕵️ 看看别人", CallbackData: "farm_steal"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	target := plots[rand.Intn(len(plots))]
	crop := farmCropMap[target.CropType]
	cropName := "作物"
	cropEmoji := "🌿"
	unitPrice := 10000 // fallback
	if crop != nil {
		cropName = crop.Name
		cropEmoji = crop.Emoji
		unitPrice = crop.UnitPrice
	}

	// 偷取随机 1~3 个单位
	stealUnits := 1 + rand.Intn(3)
	stealValue := stealUnits * unitPrice

	_ = model.IncrementPlotStolenCount(target.Id)
	_ = model.CreateFarmStealLog(&model.TgFarmStealLog{
		ThiefId:  tgId,
		VictimId: victimId,
		PlotId:   target.Id,
		Amount:   stealValue,
	})
	_ = model.IncreaseUserQuota(user.Id, stealValue, true)

	common.SysLog(fmt.Sprintf("TG Farm: user %s stole %s from %s, +%d quota", tgId, cropName, victimId, stealValue))

	text := fmt.Sprintf("🕵️ 偷菜成功！\n\n你从 %s 的农场偷了 %d个%s%s\n💰 获得 %s 额度",
		maskTgId(victimId), stealUnits, cropEmoji, cropName, farmQuotaStr(stealValue))
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🕵️ 继续偷菜", CallbackData: "farm_steal"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 治疗 ==========

func showFarmTreatSelection(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	text := "💊 选择要治疗的地块：\n\n"
	var rows [][]TgInlineKeyboardButton
	hasEvent := false
	hasDrought := false
	for _, plot := range plots {
		if plot.Status == 3 {
			crop := farmCropMap[plot.CropType]
			cropName := "作物"
			cropEmoji := "🌿"
			if crop != nil {
				cropName = crop.Name
				cropEmoji = crop.Emoji
			}
			if plot.EventType == "drought" {
				hasDrought = true
				text += fmt.Sprintf("🏜️ %d号地 - %s 天灾干旱！（💧请去浇水救命）\n",
					plot.PlotIndex+1, cropName)
			} else {
				hasEvent = true
				evtLabel := farmEventLabel(plot.EventType)
				var needItem string
				for _, item := range farmItems {
					if item.Cures == plot.EventType {
						needItem = item.Emoji + item.Name
						break
					}
				}
				text += fmt.Sprintf("%s %d号地 - %s %s (需要%s)\n",
					cropEmoji, plot.PlotIndex+1, cropName, evtLabel, needItem)
				rows = append(rows, []TgInlineKeyboardButton{
					{Text: fmt.Sprintf("💊 治疗 %d号地", plot.PlotIndex+1),
						CallbackData: fmt.Sprintf("farm_tr_%d", plot.PlotIndex)},
				})
			}
		}
	}
	if !hasEvent && !hasDrought {
		text = "💊 没有需要治疗的地块。"
	}
	if hasDrought {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "💧 去浇水", CallbackData: "farm_water"},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🏪 去商店", CallbackData: "farm_shop"},
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard, from)
}

func doFarmTreat(chatId int64, editMsgId int, tgId string, plotIdx int, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	var targetPlot *model.TgFarmPlot
	for _, p := range plots {
		if p.PlotIndex == plotIdx {
			targetPlot = p
			break
		}
	}
	if targetPlot == nil || targetPlot.Status != 3 {
		farmSend(chatId, editMsgId, "❌ 该地块不需要治疗", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	var cureItem *farmItemDef
	for i := range farmItems {
		if farmItems[i].Cures == targetPlot.EventType {
			cureItem = &farmItems[i]
			break
		}
	}
	if cureItem == nil {
		farmSend(chatId, editMsgId, "❌ 无法治疗此事件", nil, from)
		return
	}

	err = model.DecrementFarmItem(tgId, cureItem.Key)
	if err != nil {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 你没有 %s%s！请先到商店购买。",
			cureItem.Emoji, cureItem.Name), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	now := time.Now().Unix()
	downtime := now - targetPlot.EventAt
	targetPlot.PlantedAt += downtime
	targetPlot.Status = 1
	targetPlot.EventType = ""
	targetPlot.EventAt = 0
	_ = model.UpdateFarmPlot(targetPlot)

	crop := farmCropMap[targetPlot.CropType]
	cropName := "作物"
	if crop != nil {
		cropName = crop.Name
	}
	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 使用 %s%s 治疗成功！\n%s 恢复生长中。",
		cureItem.Emoji, cureItem.Name, cropName), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 施肥 ==========

func showFarmFertSelection(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	// 检查背包化肥
	items, _ := model.GetFarmItems(tgId)
	hasFert := false
	for _, item := range items {
		if item.ItemType == "fertilizer" && item.Quantity > 0 {
			hasFert = true
			break
		}
	}
	if !hasFert {
		farmSend(chatId, editMsgId, "❌ 你没有化肥！请先到商店购买。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	text := "🧴 选择要施肥的地块（生长中且未施肥）：\n\n"
	var rows [][]TgInlineKeyboardButton
	hasTarget := false
	for _, plot := range plots {
		if plot.Status == 1 && plot.Fertilized == 0 {
			crop := farmCropMap[plot.CropType]
			if crop == nil {
				continue
			}
			hasTarget = true
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("%s %d号地 - %s", crop.Emoji, plot.PlotIndex+1, crop.Name),
					CallbackData: fmt.Sprintf("farm_ff_%d", plot.PlotIndex)},
			})
		}
	}
	if !hasTarget {
		text += "没有可施肥的地块（需要生长中且未施肥）。"
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmFertilize(chatId int64, editMsgId int, tgId string, plotIdx int, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	var target *model.TgFarmPlot
	for _, plot := range plots {
		if plot.PlotIndex == plotIdx {
			target = plot
			break
		}
	}
	if target == nil || target.Status != 1 || target.Fertilized == 1 {
		farmSend(chatId, editMsgId, "❌ 该地块不可施肥。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 消耗化肥
	if err := model.DecrementFarmItem(tgId, "fertilizer"); err != nil {
		farmSend(chatId, editMsgId, "❌ 化肥不足！请先到商店购买。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 标记已施肥
	target.Fertilized = 1
	_ = model.UpdateFarmPlot(target)

	crop := farmCropMap[target.CropType]
	cropName := "作物"
	if crop != nil {
		cropName = crop.Emoji + crop.Name
	}

	common.SysLog(fmt.Sprintf("TG Farm: user %s fertilized plot %d (%s)", tgId, plotIdx, cropName))

	farmSend(chatId, editMsgId, fmt.Sprintf("🧴 施肥成功！\n\n%d号地 %s 已施肥，收获时产量+50%%！", plotIdx+1, cropName), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🧴 继续施肥", CallbackData: "farm_fert"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 购买土地 ==========

func doFarmBuyLand(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	plotCount, err := model.GetFarmPlotCount(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	if int(plotCount) >= model.FarmMaxPlots {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 你已拥有 %d 块土地，已达上限！", model.FarmMaxPlots), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	price := common.TgBotFarmPlotPrice
	if user.Quota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！\n\n土地价格：%s\n你的余额：%s",
			farmQuotaStr(price), farmQuotaStr(user.Quota)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 扣费
	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败，请稍后再试。", nil, from)
		return
	}

	// 创建新地块
	newIdx := int(plotCount)
	err = model.CreateNewFarmPlot(tgId, newIdx)
	if err != nil {
		// 回滚扣费
		_ = model.IncreaseUserQuota(user.Id, price, true)
		farmSend(chatId, editMsgId, "❌ 创建地块失败，已退款。", nil, from)
		return
	}

	common.SysLog(fmt.Sprintf("TG Farm: user %s bought plot %d for %d quota", tgId, newIdx+1, price))

	farmSend(chatId, editMsgId, fmt.Sprintf("🏗️ 购买成功！\n\n你获得了 %d号地！\n💰 花费 %s\n📊 当前土地 %d/%d 块",
		newIdx+1, farmQuotaStr(price), newIdx+1, model.FarmMaxPlots), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 浇水 ==========

func showFarmWaterSelection(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	text := "💧 选择要浇水的地块：\n\n"
	var rows [][]TgInlineKeyboardButton
	hasTarget := false
	for _, plot := range plots {
		needsWater := plot.Status == 1 || plot.Status == 4 ||
			(plot.Status == 3 && plot.EventType == "drought")
		if needsWater {
			crop := farmCropMap[plot.CropType]
			if crop == nil {
				continue
			}
			hasTarget = true
			statusLabel := "生长中"
			if plot.Status == 4 {
				statusLabel = "🥀枯萎中"
			} else if plot.Status == 3 && plot.EventType == "drought" {
				statusLabel = "🏜️天灾干旱"
			}
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("%s %d号地 - %s (%s)", crop.Emoji, plot.PlotIndex+1, crop.Name, statusLabel),
					CallbackData: fmt.Sprintf("farm_ww_%d", plot.PlotIndex)},
			})
		}
	}
	if !hasTarget {
		text += "没有需要浇水的地块。"
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmWater(chatId int64, editMsgId int, tgId string, plotIdx int, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	var target *model.TgFarmPlot
	for _, plot := range plots {
		if plot.PlotIndex == plotIdx {
			target = plot
			break
		}
	}
	if target == nil {
		farmSend(chatId, editMsgId, "❌ 该地块不需要浇水。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}
	canWater := target.Status == 1 || target.Status == 4 ||
		(target.Status == 3 && target.EventType == "drought")
	if !canWater {
		farmSend(chatId, editMsgId, "❌ 该地块不需要浇水。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	wasWilting := target.Status == 4
	wasDrought := target.Status == 3 && target.EventType == "drought"

	// 如果是枯萎状态，恢复为生长中，补偿枯萎期间的时间
	if wasWilting {
		now := time.Now().Unix()
		waterInterval := int64(common.TgBotFarmWaterInterval)
		wiltStart := target.LastWateredAt + waterInterval
		downtime := now - wiltStart
		target.PlantedAt += downtime
		target.Status = 1
		_ = model.UpdateFarmPlot(target)
	}

	// 如果是天灾干旱，恢复为生长中，补偿干旱期间的时间
	if wasDrought {
		now := time.Now().Unix()
		downtime := now - target.EventAt
		target.PlantedAt += downtime
		target.Status = 1
		target.EventType = ""
		target.EventAt = 0
		_ = model.UpdateFarmPlot(target)
	}

	_ = model.WaterFarmPlot(target.Id)

	crop := farmCropMap[target.CropType]
	cropName := "作物"
	if crop != nil {
		cropName = crop.Emoji + crop.Name
	}

	msg := fmt.Sprintf("💧 浇水成功！\n\n%d号地 %s", plotIdx+1, cropName)
	if wasDrought {
		msg += " 天灾干旱已解除，恢复生长！"
	} else if wasWilting {
		msg += " 已从枯萎中恢复生长！"
	} else {
		msg += " 已浇水。"
	}

	farmSend(chatId, editMsgId, msg, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "💧 继续浇水", CallbackData: "farm_water"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 狗狗系统 ==========

func showFarmDog(chatId int64, editMsgId int, tgId string, from *TgUser) {
	dog, err := model.GetFarmDog(tgId)
	if err != nil {
		// 没有狗
		text := "🐕 你还没有狗狗！\n\n" +
			fmt.Sprintf("在商店购买一只小狗（%s），养大后可以帮你看门拦截偷菜者！\n\n", farmQuotaStr(common.TgBotFarmDogPrice)) +
			fmt.Sprintf("🐶 幼犬需要 %d 小时长大为成犬\n", common.TgBotFarmDogGrowHours) +
			"🦴 记得定期喂狗粮，饿坏了就不看门了\n" +
			fmt.Sprintf("🛡️ 成犬看门拦截率：%d%%", common.TgBotFarmDogGuardRate)
		farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: fmt.Sprintf("🐶 购买小狗 (%s)", farmQuotaStr(common.TgBotFarmDogPrice)), CallbackData: "farm_buydog"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	model.UpdateDogHunger(dog)

	levelStr := "🐶 幼犬"
	statusStr := "成长中"
	if dog.Level == 2 {
		levelStr = "🐕 成犬"
		if dog.Hunger > 0 {
			statusStr = "✅ 看门中"
		} else {
			statusStr = "❌ 饿坏了，无法看门"
		}
	} else {
		if dog.Hunger == 0 {
			statusStr = "❌ 饿坏了"
		} else {
			now := time.Now().Unix()
			hoursLeft := int64(common.TgBotFarmDogGrowHours) - (now-dog.CreatedAt)/3600
			if hoursLeft < 0 {
				hoursLeft = 0
			}
			statusStr = fmt.Sprintf("⏳ 还需 %d 小时长大", hoursLeft)
		}
	}

	text := fmt.Sprintf("🐕 我的狗狗\n\n"+
		"名字：%s\n"+
		"等级：%s\n"+
		"状态：%s\n"+
		"饱食度：%d%%\n\n"+
		"🛡️ 看门拦截率：%d%%\n"+
		"🦴 狗粮价格：%s",
		dog.Name, levelStr, statusStr, dog.Hunger,
		common.TgBotFarmDogGuardRate, farmQuotaStr(common.TgBotFarmDogFoodPrice))

	var rows [][]TgInlineKeyboardButton
	if dog.Hunger < 100 {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🦴 喂狗粮", CallbackData: "farm_feeddog"},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🏪 商店买狗粮", CallbackData: "farm_shop"},
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmBuyDog(chatId int64, editMsgId int, tgId string, from *TgUser) {
	// 检查是否已有狗
	_, err := model.GetFarmDog(tgId)
	if err == nil {
		farmSend(chatId, editMsgId, "❌ 你已经有一只狗了！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🐕 查看狗狗", CallbackData: "farm_dog"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	price := common.TgBotFarmDogPrice
	if user.Quota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！需要 %s", farmQuotaStr(price)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回商店", CallbackData: "farm_shop"}},
			},
		}, from)
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败", nil, from)
		return
	}

	// 生成随机狗名
	dogNames := []string{"旺财", "小黑", "大黄", "豆豆", "球球", "毛毛", "Lucky", "小白", "花花", "阿福"}
	dogName := dogNames[rand.Intn(len(dogNames))]

	now := time.Now().Unix()
	dog := &model.TgFarmDog{
		TelegramId: tgId,
		Name:       dogName,
		Level:      1,
		Hunger:     100,
		LastFedAt:  now,
	}
	err = model.CreateFarmDog(dog)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		farmSend(chatId, editMsgId, "❌ 购买失败，已退款。", nil, from)
		return
	}

	common.SysLog(fmt.Sprintf("TG Farm: user %s bought dog '%s' for %d quota", tgId, dogName, price))

	farmSend(chatId, editMsgId, fmt.Sprintf("🐶 恭喜！你获得了一只小狗「%s」！\n\n"+
		"花费：%s\n"+
		"等级：幼犬\n"+
		"⏳ %d 小时后长大为成犬，即可看门拦截偷菜者\n"+
		"🦴 记得定期喂狗粮哦！",
		dogName, farmQuotaStr(price), common.TgBotFarmDogGrowHours), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🐕 查看狗狗", CallbackData: "farm_dog"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func doFarmFeedDog(chatId int64, editMsgId int, tgId string, from *TgUser) {
	dog, err := model.GetFarmDog(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 你还没有狗狗！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	model.UpdateDogHunger(dog)

	if dog.Hunger >= 100 {
		farmSend(chatId, editMsgId, "❌ 狗狗现在不饿，不需要喂食！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🐕 查看狗狗", CallbackData: "farm_dog"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	// 消耗狗粮
	err = model.DecrementFarmItem(tgId, "dogfood")
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 你没有狗粮！请先到商店购买。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	_ = model.FeedFarmDog(dog.Id)

	farmSend(chatId, editMsgId, fmt.Sprintf("🦴 喂食成功！「%s」吃饱了，饱食度恢复到 100%%！", dog.Name), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🐕 查看狗狗", CallbackData: "farm_dog"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 辅助函数 ==========

func farmEventLabel(eventType string) string {
	switch eventType {
	case "bugs":
		return "虫害🐛"
	case "drought":
		return "天灾干旱🏜️"
	}
	return "未知"
}

func maskTgId(tgId string) string {
	if len(tgId) > 6 {
		return tgId[:3] + "***" + tgId[len(tgId)-3:]
	}
	return "***"
}

func farmSend(chatId int64, editMsgId int, text string, keyboard *TgInlineKeyboardMarkup, from *TgUser) {
	if editMsgId > 0 {
		editTgMessage(chatId, editMsgId, text, keyboard, from)
	} else if keyboard != nil {
		sendTgMessageWithKeyboard(chatId, text, *keyboard, from)
	} else {
		sendTgMessage(chatId, text, from)
	}
}
