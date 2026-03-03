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
	Key      string
	Short    string // callback abbreviation
	Name     string
	Emoji    string
	SeedCost int   // quota units
	GrowSecs int64 // seconds to grow
}

type farmItemDef struct {
	Key   string
	Name  string
	Emoji string
	Cost  int    // quota units
	Cures string // event type it cures
}

var farmCrops = []farmCropDef{
	{"cabbage", "cab", "白菜", "🥬", 250000, 1800},
	{"tomato", "tom", "番茄", "🍅", 500000, 3600},
	{"carrot", "car", "胡萝卜", "🥕", 750000, 7200},
	{"strawberry", "str", "草莓", "🍓", 1000000, 10800},
	{"corn", "cor", "玉米", "🌽", 1250000, 14400},
	{"watermelon", "wat", "西瓜", "🍉", 1500000, 21600},
}

var farmItems = []farmItemDef{
	{"water", "水壶", "💧", 150000, "drought"},
	{"pesticide", "杀虫剂", "🧪", 150000, "bugs"},
}

var farmCropMap map[string]*farmCropDef
var farmCropByShort map[string]*farmCropDef
var farmItemMap map[string]*farmItemDef

const farmHarvestValue = 2500000 // $5
const farmStealValue = 500000    // $1
const farmMaxSteals = 2
const farmStealCooldownSecs = 1800 // 30 min
const farmEventChance = 30         // 30%

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
	if plot.Status != 1 {
		return
	}
	now := time.Now().Unix()
	crop := farmCropMap[plot.CropType]
	if crop == nil {
		return
	}
	changed := false
	// 事件触发优先
	if plot.EventAt > 0 && plot.EventType != "" && now >= plot.EventAt {
		plot.Status = 3
		changed = true
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

func farmBindingError(chatId int64, editMsgId int) {
	text := "🔑 你还没有绑定平台账号！\n\n" +
		"请先私聊机器人发送你的 API Key（以 sk- 开头）完成绑定。\n" +
		"绑定后才能使用农场功能。\n\n" +
		"发送 /bindaccount 查看绑定说明。"
	farmSend(chatId, editMsgId, text, nil)
}

// ========== 命令入口 ==========

func handleFarmCommand(chatId int64, from *TgUser, isGroup bool) {
	tgId := strconv.FormatInt(from.Id, 10)
	if _, err := getFarmUser(tgId); err != nil {
		farmBindingError(chatId, 0)
		return
	}
	showFarmView(chatId, 0, tgId)
}

func handleFarmCallback(cb *TgCallbackQuery) {
	chatId := cb.Message.Chat.Id
	msgId := cb.Message.MessageId
	tgId := strconv.FormatInt(cb.From.Id, 10)
	data := cb.Data

	// 统一绑定检查：所有农场操作都需要绑定账号
	if _, err := getFarmUser(tgId); err != nil {
		farmBindingError(chatId, msgId)
		return
	}

	switch {
	case data == "farm":
		showFarmView(chatId, msgId, tgId)
	case data == "farm_plant":
		showFarmPlantCrops(chatId, msgId, tgId)
	case strings.HasPrefix(data, "farm_p_"):
		cropShort := strings.TrimPrefix(data, "farm_p_")
		showFarmPlotSelection(chatId, msgId, tgId, cropShort)
	case strings.HasPrefix(data, "farm_pp_"):
		parts := strings.SplitN(strings.TrimPrefix(data, "farm_pp_"), "_", 2)
		if len(parts) == 2 {
			plotIdx, _ := strconv.Atoi(parts[0])
			doFarmPlant(chatId, msgId, tgId, plotIdx, parts[1])
		}
	case data == "farm_harvest":
		doFarmHarvest(chatId, msgId, tgId)
	case data == "farm_shop":
		showFarmShop(chatId, msgId, tgId)
	case strings.HasPrefix(data, "farm_buy_"):
		itemKey := strings.TrimPrefix(data, "farm_buy_")
		doFarmBuy(chatId, msgId, tgId, itemKey)
	case data == "farm_steal":
		showFarmStealTargets(chatId, msgId, tgId)
	case strings.HasPrefix(data, "farm_st_"):
		victimId := strings.TrimPrefix(data, "farm_st_")
		doFarmSteal(chatId, msgId, tgId, victimId)
	case data == "farm_treat":
		showFarmTreatSelection(chatId, msgId, tgId)
	case strings.HasPrefix(data, "farm_tr_"):
		plotStr := strings.TrimPrefix(data, "farm_tr_")
		plotIdx, _ := strconv.Atoi(plotStr)
		doFarmTreat(chatId, msgId, tgId, plotIdx)
	}
}

// ========== 农场视图 ==========

func showFarmView(chatId int64, editMsgId int, tgId string) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	text := "🌾 我的农场\n\n"
	hasEvent := false
	for _, plot := range plots {
		text += farmPlotLine(plot) + "\n"
		if plot.Status == 3 {
			hasEvent = true
		}
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

	var rows [][]TgInlineKeyboardButton
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🌱 种植", CallbackData: "farm_plant"},
		{Text: "🌾 收获", CallbackData: "farm_harvest"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🏪 商店", CallbackData: "farm_shop"},
		{Text: "🕵️ 偷菜", CallbackData: "farm_steal"},
	})
	if hasEvent {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "💊 治疗", CallbackData: "farm_treat"},
		})
	}
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard)
}

func farmPlotLine(plot *model.TgFarmPlot) string {
	idx := plot.PlotIndex + 1
	switch plot.Status {
	case 0:
		return fmt.Sprintf("🟫 %d号地 - 空地", idx)
	case 1:
		crop := farmCropMap[plot.CropType]
		if crop == nil {
			return fmt.Sprintf("🟫 %d号地 - 空地", idx)
		}
		now := time.Now().Unix()
		elapsed := now - plot.PlantedAt
		total := crop.GrowSecs
		pct := int(elapsed * 100 / total)
		if pct > 99 {
			pct = 99
		}
		remaining := total - elapsed
		return fmt.Sprintf("%s %d号地 - %s 生长中 %d%% 剩余%s", crop.Emoji, idx, crop.Name, pct, formatDuration(remaining))
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
		eventEmoji := "❌"
		eventLabel := "未知事件"
		switch plot.EventType {
		case "bugs":
			eventEmoji = "🐛"
			eventLabel = "虫害"
		case "drought":
			eventEmoji = "🏜️"
			eventLabel = "干旱"
		}
		return fmt.Sprintf("%s %d号地 - %s %s%s！需要治疗", emoji, idx, name, eventEmoji, eventLabel)
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

func showFarmPlantCrops(chatId int64, editMsgId int, tgId string) {
	text := "🌱 选择要种植的作物：\n\n"
	var rows [][]TgInlineKeyboardButton
	for _, crop := range farmCrops {
		text += fmt.Sprintf("%s %s - 种子%s 生长%s 收获$5\n",
			crop.Emoji, crop.Name, farmQuotaStr(crop.SeedCost), formatDuration(crop.GrowSecs))
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s (%s)", crop.Emoji, crop.Name, farmQuotaStr(crop.SeedCost)),
				CallbackData: "farm_p_" + crop.Short},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard)
}

func showFarmPlotSelection(chatId int64, editMsgId int, tgId string, cropShort string) {
	crop := farmCropByShort[cropShort]
	if crop == nil {
		farmSend(chatId, editMsgId, "❌ 未知作物", nil)
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil)
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
				{Text: fmt.Sprintf("🟫 %d号地", plot.PlotIndex+1),
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
	farmSend(chatId, editMsgId, text, &keyboard)
}

func doFarmPlant(chatId int64, editMsgId int, tgId string, plotIdx int, cropShort string) {
	crop := farmCropByShort[cropShort]
	if crop == nil {
		farmSend(chatId, editMsgId, "❌ 未知作物", nil)
		return
	}
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId)
		return
	}
	if user.Quota < crop.SeedCost {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！种子需要 %s，当前余额 %s",
			farmQuotaStr(crop.SeedCost), farmQuotaStr(user.Quota)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_plant"}},
			},
		})
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil)
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
		})
		return
	}
	err = model.DecreaseUserQuota(user.Id, crop.SeedCost)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败，请稍后再试", nil)
		return
	}

	now := time.Now().Unix()
	targetPlot.CropType = crop.Key
	targetPlot.PlantedAt = now
	targetPlot.Status = 1
	targetPlot.EventType = ""
	targetPlot.EventAt = 0
	targetPlot.StolenCount = 0

	if rand.Intn(100) < farmEventChance {
		eventTypes := []string{"bugs", "drought"}
		targetPlot.EventType = eventTypes[rand.Intn(2)]
		offset := crop.GrowSecs * int64(30+rand.Intn(50)) / 100
		targetPlot.EventAt = now + offset
	}

	_ = model.UpdateFarmPlot(targetPlot)
	common.SysLog(fmt.Sprintf("TG Farm: user %s planted %s on plot %d, cost %d", tgId, crop.Key, plotIdx, crop.SeedCost))
	showFarmView(chatId, editMsgId, tgId)
}

// ========== 收获 ==========

func doFarmHarvest(chatId int64, editMsgId int, tgId string) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId)
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil)
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
			value := farmHarvestValue - (plot.StolenCount * farmStealValue)
			if value < 0 {
				value = 0
			}
			totalQuota += value
			harvestedCount++
			details += fmt.Sprintf("\n%s %s → %s", crop.Emoji, crop.Name, farmQuotaStr(value))
			if plot.StolenCount > 0 {
				details += fmt.Sprintf(" (被偷%d次)", plot.StolenCount)
			}
			_ = model.ClearFarmPlot(plot.Id)
		}
	}

	if harvestedCount == 0 {
		farmSend(chatId, editMsgId, "🌾 没有可收获的作物。\n\n种植作物并等待成熟后即可收获！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🌱 去种植", CallbackData: "farm_plant"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		})
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
	})
}

// ========== 商店 ==========

func showFarmShop(chatId int64, editMsgId int, tgId string) {
	text := "🏪 农场商店\n\n"
	text += "📌 种子（在「种植」中直接购买并种下）：\n"
	for _, crop := range farmCrops {
		text += fmt.Sprintf("  %s %s - %s (生长%s)\n",
			crop.Emoji, crop.Name, farmQuotaStr(crop.SeedCost), formatDuration(crop.GrowSecs))
	}
	text += "\n📌 道具：\n"
	var rows [][]TgInlineKeyboardButton
	for _, item := range farmItems {
		cureLabel := farmEventLabel(item.Cures)
		text += fmt.Sprintf("  %s %s - %s (治疗%s)\n", item.Emoji, item.Name, farmQuotaStr(item.Cost), cureLabel)
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("%s 购买%s (%s)", item.Emoji, item.Name, farmQuotaStr(item.Cost)),
				CallbackData: "farm_buy_" + item.Key},
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
	farmSend(chatId, editMsgId, text, &keyboard)
}

func doFarmBuy(chatId int64, editMsgId int, tgId string, itemKey string) {
	item := farmItemMap[itemKey]
	if item == nil {
		farmSend(chatId, editMsgId, "❌ 未知道具", nil)
		return
	}
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId)
		return
	}
	if user.Quota < item.Cost {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！需要 %s", farmQuotaStr(item.Cost)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回商店", CallbackData: "farm_shop"}},
			},
		})
		return
	}
	err = model.DecreaseUserQuota(user.Id, item.Cost)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败", nil)
		return
	}
	err = model.IncrementFarmItem(tgId, itemKey, 1)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, item.Cost, true)
		farmSend(chatId, editMsgId, "❌ 购买失败", nil)
		return
	}
	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 购买 %s%s 成功！已扣除 %s",
		item.Emoji, item.Name, farmQuotaStr(item.Cost)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🏪 继续购物", CallbackData: "farm_shop"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	})
}

// ========== 偷菜 ==========

func showFarmStealTargets(chatId int64, editMsgId int, tgId string) {
	targets, err := model.GetMatureFarmTargets(tgId)
	if err != nil || len(targets) == 0 {
		farmSend(chatId, editMsgId, "🕵️ 暂时没有可偷的菜地。\n\n等其他玩家的作物成熟后再来！", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		})
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
	farmSend(chatId, editMsgId, text, &keyboard)
}

func doFarmSteal(chatId int64, editMsgId int, tgId string, victimId string) {
	if tgId == victimId {
		farmSend(chatId, editMsgId, "❌ 不能偷自己的菜！", nil)
		return
	}
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId)
		return
	}

	now := time.Now().Unix()
	recentSteals, _ := model.CountRecentSteals(tgId, victimId, now-int64(farmStealCooldownSecs))
	if recentSteals > 0 {
		farmSend(chatId, editMsgId, "⏳ 冷却中！30分钟内只能偷同一人一次。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🕵️ 看看别人", CallbackData: "farm_steal"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		})
		return
	}

	plots, err := model.GetStealablePlots(victimId)
	if err != nil || len(plots) == 0 {
		farmSend(chatId, editMsgId, "❌ 该玩家没有可偷的成熟作物了。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🕵️ 看看别人", CallbackData: "farm_steal"},
					{Text: "🔙 返回", CallbackData: "farm"}},
			},
		})
		return
	}

	target := plots[rand.Intn(len(plots))]
	crop := farmCropMap[target.CropType]
	cropName := "作物"
	cropEmoji := "🌿"
	if crop != nil {
		cropName = crop.Name
		cropEmoji = crop.Emoji
	}

	_ = model.IncrementPlotStolenCount(target.Id)
	_ = model.CreateFarmStealLog(&model.TgFarmStealLog{
		ThiefId:  tgId,
		VictimId: victimId,
		PlotId:   target.Id,
		Amount:   farmStealValue,
	})
	_ = model.IncreaseUserQuota(user.Id, farmStealValue, true)

	common.SysLog(fmt.Sprintf("TG Farm: user %s stole %s from %s, +%d quota", tgId, cropName, victimId, farmStealValue))

	text := fmt.Sprintf("🕵️ 偷菜成功！\n\n你从 %s 的农场偷了 %s%s\n💰 获得 %s 额度",
		maskTgId(victimId), cropEmoji, cropName, farmQuotaStr(farmStealValue))
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🕵️ 继续偷菜", CallbackData: "farm_steal"},
				{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	})
}

// ========== 治疗 ==========

func showFarmTreatSelection(chatId int64, editMsgId int, tgId string) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil)
		return
	}
	for _, plot := range plots {
		updateFarmPlotStatus(plot)
	}

	text := "💊 选择要治疗的地块：\n\n"
	var rows [][]TgInlineKeyboardButton
	hasEvent := false
	for _, plot := range plots {
		if plot.Status == 3 {
			hasEvent = true
			crop := farmCropMap[plot.CropType]
			cropName := "作物"
			cropEmoji := "🌿"
			if crop != nil {
				cropName = crop.Name
				cropEmoji = crop.Emoji
			}
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
	if !hasEvent {
		text = "💊 没有需要治疗的地块。"
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🏪 去商店", CallbackData: "farm_shop"},
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	keyboard := TgInlineKeyboardMarkup{InlineKeyboard: rows}
	farmSend(chatId, editMsgId, text, &keyboard)
}

func doFarmTreat(chatId int64, editMsgId int, tgId string, plotIdx int) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil)
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
		})
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
		farmSend(chatId, editMsgId, "❌ 无法治疗此事件", nil)
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
		})
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
	})
}

// ========== 辅助函数 ==========

func farmEventLabel(eventType string) string {
	switch eventType {
	case "bugs":
		return "虫害🐛"
	case "drought":
		return "干旱🏜️"
	}
	return "未知"
}

func maskTgId(tgId string) string {
	if len(tgId) > 6 {
		return tgId[:3] + "***" + tgId[len(tgId)-3:]
	}
	return "***"
}

func farmSend(chatId int64, editMsgId int, text string, keyboard *TgInlineKeyboardMarkup) {
	if editMsgId > 0 {
		editTgMessage(chatId, editMsgId, text, keyboard)
	} else if keyboard != nil {
		sendTgMessageWithKeyboard(chatId, text, *keyboard)
	} else {
		sendTgMessage(chatId, text)
	}
}
