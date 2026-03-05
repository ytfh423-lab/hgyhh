package controller

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
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
	{"fishbait", "鱼饵", "🪱", 250000, ""},
}

// ========== 钓鱼定义 ==========

type fishDef struct {
	Key       string
	Name      string
	Emoji     string
	Rarity    string
	Weight    int // probability weight (higher = more common)
	SellPrice int // quota units
}

var fishTypes = []fishDef{
	{"crucian", "鲫鱼", "🐟", "普通", 30, 100000},
	{"tropical", "热带鱼", "🐠", "普通", 20, 200000},
	{"shrimp", "虾", "🦐", "优良", 13, 400000},
	{"puffer", "河豚", "🐡", "优良", 9, 750000},
	{"lobster", "龙虾", "🦞", "稀有", 7, 1500000},
	{"octopus", "章鱼", "🐙", "稀有", 3, 3000000},
	{"shark", "鲨鱼", "🦈", "史诗", 2, 7500000},
	{"whale", "鲸鱼", "🐋", "传说", 1, 20000000},
}

const fishNothingWeight = 15 // 空军概率权重

var fishTypeMap map[string]*fishDef
var fishTotalWeight int

var farmCropMap map[string]*farmCropDef
var farmCropByShort map[string]*farmCropDef
var farmItemMap map[string]*farmItemDef

const farmMaxSteals = 2

// ========== 市场价格波动 ==========

var marketPrices map[string]int // key -> multiplier percentage (e.g. 150 = 150%)
var marketMu sync.RWMutex
var marketLastUpdate int64
var marketNextUpdate int64

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
	fishTypeMap = make(map[string]*fishDef)
	fishTotalWeight = fishNothingWeight
	for i := range fishTypes {
		fishTypeMap[fishTypes[i].Key] = &fishTypes[i]
		fishTotalWeight += fishTypes[i].Weight
	}
	marketPrices = make(map[string]int)
	refreshMarketPrices()
}

func refreshMarketPrices() {
	marketMu.Lock()
	defer marketMu.Unlock()
	now := time.Now().Unix()
	minM := common.TgBotMarketMinMultiplier
	maxM := common.TgBotMarketMaxMultiplier
	rng := maxM - minM + 1
	// 作物价格
	for _, crop := range farmCrops {
		marketPrices["crop_"+crop.Key] = minM + rand.Intn(rng)
	}
	// 鱼价格
	for _, fish := range fishTypes {
		marketPrices["fish_"+fish.Key] = minM + rand.Intn(rng)
	}
	// 肉价格
	for _, key := range []string{"chicken", "duck", "goose", "pig", "sheep", "cow"} {
		marketPrices["meat_"+key] = minM + rand.Intn(rng)
	}
	marketLastUpdate = now
	marketNextUpdate = now + int64(common.TgBotMarketRefreshHours*3600)
}

func ensureMarketFresh() {
	marketMu.RLock()
	next := marketNextUpdate
	marketMu.RUnlock()
	if time.Now().Unix() >= next {
		refreshMarketPrices()
	}
}

func getMarketMultiplier(key string) int {
	ensureMarketFresh()
	marketMu.RLock()
	defer marketMu.RUnlock()
	if m, ok := marketPrices[key]; ok {
		return m
	}
	return 100 // default 100%
}

func applyMarket(basePrice int, marketKey string) int {
	m := getMarketMultiplier(marketKey)
	return basePrice * m / 100
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
		growSecs := crop.GrowSecs
		soilLevel := plot.SoilLevel
		if soilLevel < 1 {
			soilLevel = 1
		}
		if soilLevel > 1 {
			bonus := int64(common.TgBotFarmSoilSpeedBonus * (soilLevel - 1))
			growSecs = growSecs * (100 - bonus) / 100
			if growSecs < 60 {
				growSecs = 60
			}
		}
		matureAt := plot.PlantedAt + growSecs
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
	case data == "farm_logs":
		showFarmLogs(chatId, msgId, tgId, from)
	case data == "farm_fish":
		showFarmFish(chatId, msgId, tgId, from)
	case data == "farm_dofish":
		doFarmFish(chatId, msgId, tgId, from)
	case data == "farm_sellfish":
		doFarmSellFish(chatId, msgId, tgId, from)
	case data == "farm_market":
		showFarmMarket(chatId, msgId, tgId, from)
	case data == "farm_soil":
		showFarmSoilUpgrade(chatId, msgId, tgId, from)
	case strings.HasPrefix(data, "farm_su_"):
		plotStr := strings.TrimPrefix(data, "farm_su_")
		plotIdx, _ := strconv.Atoi(plotStr)
		doFarmSoilUpgrade(chatId, msgId, tgId, plotIdx, from)
	case strings.HasPrefix(data, "ranch"):
		handleRanchCallback(cb)
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
	// 泥土升级按钮
	hasUpgradable := false
	for _, plot := range plots {
		sl := plot.SoilLevel
		if sl < 1 {
			sl = 1
		}
		if sl < common.TgBotFarmSoilMaxLevel {
			hasUpgradable = true
			break
		}
	}
	if hasUpgradable {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: "🌱 泥土升级", CallbackData: "farm_soil"},
		})
	}
	// 狗狗 & 牧场 & 钓鱼 & 记录按钮
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🐕 狗狗", CallbackData: "farm_dog"},
		{Text: "🐄 牧场", CallbackData: "ranch"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🎣 钓鱼", CallbackData: "farm_fish"},
		{Text: "� 市场", CallbackData: "farm_market"},
		{Text: "� 记录", CallbackData: "farm_logs"},
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
	soilTag := ""
	sl := plot.SoilLevel
	if sl < 1 {
		sl = 1
	}
	if sl > 1 {
		soilTag = fmt.Sprintf(" 🌱Lv.%d", sl)
	}

	switch plot.Status {
	case 0:
		return fmt.Sprintf("⬜ %d号地 - 空地%s", idx, soilTag)
	case 1:
		crop := farmCropMap[plot.CropType]
		if crop == nil {
			return fmt.Sprintf("⬜ %d号地 - 空地", idx)
		}
		now := time.Now().Unix()
		elapsed := now - plot.PlantedAt
		total := crop.GrowSecs
		soilLvl := plot.SoilLevel
		if soilLvl < 1 {
			soilLvl = 1
		}
		if soilLvl > 1 {
			bonus := int64(common.TgBotFarmSoilSpeedBonus * (soilLvl - 1))
			total = total * (100 - bonus) / 100
			if total < 60 {
				total = 60
			}
		}
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
		return fmt.Sprintf("%s %d号地 - %s 生长中 %d%% 剩余%s%s%s%s", crop.Emoji, idx, crop.Name, pct, formatDuration(remaining), fertTag, waterTag, soilTag)
	case 2:
		crop := farmCropMap[plot.CropType]
		if crop == nil {
			return fmt.Sprintf("✅ %d号地 - 已成熟", idx)
		}
		stolen := ""
		if plot.StolenCount > 0 {
			stolen = fmt.Sprintf(" ⚠️被偷%d次", plot.StolenCount)
		}
		return fmt.Sprintf("✅ %d号地 - %s%s 已成熟！%s%s", crop.Emoji, crop.Name, stolen, soilTag, "")
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
			return fmt.Sprintf("🏜️ %d号地 - %s%s 天灾干旱！💧快浇水救命！%s后死亡%s", idx, emoji, name, formatDuration(remaining), soilTag)
		}
		eventEmoji := "❌"
		eventLabel := "未知事件"
		switch plot.EventType {
		case "bugs":
			eventEmoji = "🐛"
			eventLabel = "虫害"
		}
		return fmt.Sprintf("%s %d号地 - %s %s%s！需要治疗%s", emoji, idx, name, eventEmoji, eventLabel, soilTag)
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
		return fmt.Sprintf("🥀 %d号地 - %s%s 枯萎中！💧快浇水！%s后死亡%s", idx, emoji, name, formatDuration(remaining), soilTag)
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
	model.AddFarmLog(tgId, "plant", -crop.SeedCost, fmt.Sprintf("种植%s%s", crop.Emoji, crop.Name))

	now := time.Now().Unix()
	targetPlot.CropType = crop.Key
	targetPlot.PlantedAt = now
	targetPlot.Status = 1
	targetPlot.EventType = ""
	targetPlot.EventAt = 0
	targetPlot.StolenCount = 0
	targetPlot.LastWateredAt = now

	// 计算实际生长时间（含泥土加速）
	actualGrowSecs := crop.GrowSecs
	plotSoilLvl := targetPlot.SoilLevel
	if plotSoilLvl < 1 {
		plotSoilLvl = 1
	}
	if plotSoilLvl > 1 {
		soilBonus := int64(common.TgBotFarmSoilSpeedBonus * (plotSoilLvl - 1))
		actualGrowSecs = actualGrowSecs * (100 - soilBonus) / 100
		if actualGrowSecs < 60 {
			actualGrowSecs = 60
		}
	}

	// 虫害事件
	if rand.Intn(100) < common.TgBotFarmEventChance {
		targetPlot.EventType = "bugs"
		offset := actualGrowSecs * int64(30+rand.Intn(50)) / 100
		targetPlot.EventAt = now + offset
	}
	// 天灾(干旱)：独立概率，不与虫害叠加
	if targetPlot.EventType == "" && rand.Intn(100) < common.TgBotFarmDisasterChance {
		targetPlot.EventType = "drought"
		offset := actualGrowSecs * int64(30+rand.Intn(50)) / 100
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
			marketPrice := applyMarket(crop.UnitPrice, "crop_"+crop.Key)
			value := realYield * marketPrice
			totalQuota += value
			harvestedCount++

			mPct := getMarketMultiplier("crop_" + crop.Key)
			details += fmt.Sprintf("\n%s %s: 产量%d", crop.Emoji, crop.Name, yield-fertBonus)
			if fertBonus > 0 {
				details += fmt.Sprintf(" +化肥%d", fertBonus)
			}
			if loss > 0 {
				details += fmt.Sprintf(" -被偷%d", loss)
			}
			details += fmt.Sprintf(" = 实收%d × %s(%d%%) = %s",
				realYield, farmQuotaStr(marketPrice), mPct, farmQuotaStr(value))

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
	model.AddFarmLog(tgId, "harvest", totalQuota, fmt.Sprintf("收获%d种作物", harvestedCount))
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
	model.AddFarmLog(tgId, "shop", -cost, fmt.Sprintf("购买%s%s", item.Emoji, item.Name))
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
	model.AddFarmLog(tgId, "steal", stealValue, fmt.Sprintf("偷取%s%s×%d", cropEmoji, cropName, stealUnits))

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

	model.AddFarmLog(tgId, "buy_plot", -price, fmt.Sprintf("购买%d号地", newIdx+1))
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

	model.AddFarmLog(tgId, "buy_dog", -price, fmt.Sprintf("购买看门狗「%s」", dogName))
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

// ========== 泥土升级 ==========

func showFarmSoilUpgrade(chatId int64, editMsgId int, tgId string, from *TgUser) {
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 系统错误", nil, from)
		return
	}

	text := "🌱 泥土升级\n\n"
	text += fmt.Sprintf("📌 每级加速生长 %d%%\n", common.TgBotFarmSoilSpeedBonus)
	text += fmt.Sprintf("📌 最高等级 Lv.%d\n\n", common.TgBotFarmSoilMaxLevel)
	text += "升级价格：\n"
	prices := map[int]int{
		2: common.TgBotFarmSoilUpgradePrice2,
		3: common.TgBotFarmSoilUpgradePrice3,
		4: common.TgBotFarmSoilUpgradePrice4,
		5: common.TgBotFarmSoilUpgradePrice5,
	}
	for lvl := 2; lvl <= common.TgBotFarmSoilMaxLevel && lvl <= 5; lvl++ {
		text += fmt.Sprintf("  Lv.%d → %s (加速 %d%%)\n", lvl, farmQuotaStr(prices[lvl]), common.TgBotFarmSoilSpeedBonus*(lvl-1))
	}
	text += "\n选择要升级的地块：\n"

	var rows [][]TgInlineKeyboardButton
	hasUpgradable := false
	for _, plot := range plots {
		sl := plot.SoilLevel
		if sl < 1 {
			sl = 1
		}
		if sl >= common.TgBotFarmSoilMaxLevel {
			continue
		}
		hasUpgradable = true
		nextLvl := sl + 1
		price := 0
		if p, ok := prices[nextLvl]; ok {
			price = p
		}
		label := fmt.Sprintf("%d号地 Lv.%d→%d (%s)", plot.PlotIndex+1, sl, nextLvl, farmQuotaStr(price))
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: label, CallbackData: fmt.Sprintf("farm_su_%d", plot.PlotIndex)},
		})
	}
	if !hasUpgradable {
		text += "所有地块已达最高等级！🎉\n"
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmSoilUpgrade(chatId int64, editMsgId int, tgId string, plotIdx int, from *TgUser) {
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

	var target *model.TgFarmPlot
	for _, plot := range plots {
		if plot.PlotIndex == plotIdx {
			target = plot
			break
		}
	}
	if target == nil {
		farmSend(chatId, editMsgId, "❌ 地块不存在", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_soil"}},
			},
		}, from)
		return
	}

	currentLevel := target.SoilLevel
	if currentLevel < 1 {
		currentLevel = 1
	}
	nextLevel := currentLevel + 1
	if nextLevel > common.TgBotFarmSoilMaxLevel {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ %d号地泥土已达最高等级 Lv.%d！", plotIdx+1, common.TgBotFarmSoilMaxLevel), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_soil"}},
			},
		}, from)
		return
	}

	var price int
	switch nextLevel {
	case 2:
		price = common.TgBotFarmSoilUpgradePrice2
	case 3:
		price = common.TgBotFarmSoilUpgradePrice3
	case 4:
		price = common.TgBotFarmSoilUpgradePrice4
	case 5:
		price = common.TgBotFarmSoilUpgradePrice5
	default:
		farmSend(chatId, editMsgId, "❌ 不支持的升级等级", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_soil"}},
			},
		}, from)
		return
	}

	if user.Quota < price {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！\n\n升级到 Lv.%d 需要：%s\n你的余额：%s",
			nextLevel, farmQuotaStr(price), farmQuotaStr(user.Quota)), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回", CallbackData: "farm_soil"}},
			},
		}, from)
		return
	}

	err = model.DecreaseUserQuota(user.Id, price)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 扣费失败，请稍后再试。", nil, from)
		return
	}

	err = model.UpgradeFarmPlotSoil(target.Id, nextLevel)
	if err != nil {
		_ = model.IncreaseUserQuota(user.Id, price, true)
		farmSend(chatId, editMsgId, "❌ 升级失败，已退款。", nil, from)
		return
	}

	speedBonus := common.TgBotFarmSoilSpeedBonus * (nextLevel - 1)
	model.AddFarmLog(tgId, "upgrade_soil", -price, fmt.Sprintf("%d号地泥土升级Lv.%d", plotIdx+1, nextLevel))
	common.SysLog(fmt.Sprintf("TG Farm: user %s upgraded plot %d soil to Lv.%d for %d quota", tgId, plotIdx+1, nextLevel, price))

	farmSend(chatId, editMsgId, fmt.Sprintf("🌱 升级成功！\n\n%d号地泥土升级到 Lv.%d\n⚡ 生长加速 %d%%\n💰 花费 %s",
		plotIdx+1, nextLevel, speedBonus, farmQuotaStr(price)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🌱 继续升级", CallbackData: "farm_soil"},
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

// ========== 消费记录 ==========

func showFarmLogs(chatId int64, editMsgId int, tgId string, from *TgUser) {
	logs, total, err := model.GetFarmLogs(tgId, 15, 0)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 获取记录失败", nil, from)
		return
	}

	actionLabels := map[string]string{
		"plant": "种植", "harvest": "收获", "shop": "商店", "steal": "偷菜",
		"buy_plot": "购地", "buy_dog": "买狗", "upgrade_soil": "升级",
		"ranch_buy": "买动物", "ranch_feed": "喂食", "ranch_water": "喂水",
		"ranch_sell": "出售", "ranch_clean": "清粪",
		"fish": "钓鱼", "fish_sell": "卖鱼",
	}

	text := fmt.Sprintf("📋 消费记录（最近15条，共%d条）\n\n", total)
	if len(logs) == 0 {
		text += "暂无记录\n"
	}
	for _, l := range logs {
		label := actionLabels[l.Action]
		if label == "" {
			label = l.Action
		}
		sign := "+"
		if l.Amount < 0 {
			sign = ""
		}
		ts := time.Unix(l.CreatedAt, 0)
		text += fmt.Sprintf("%s %s%s %s · %s\n",
			label, sign, farmQuotaStr(l.Amount), l.Detail,
			ts.Format("01-02 15:04"))
	}

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 钓鱼 ==========

func randomFish() *fishDef {
	r := rand.Intn(fishTotalWeight)
	cumulative := fishNothingWeight
	if r < cumulative {
		return nil // 空军
	}
	for i := range fishTypes {
		cumulative += fishTypes[i].Weight
		if r < cumulative {
			return &fishTypes[i]
		}
	}
	return &fishTypes[len(fishTypes)-1]
}

func showFarmFish(chatId int64, editMsgId int, tgId string, from *TgUser) {
	// 鱼饵数量
	items, _ := model.GetFarmItems(tgId)
	baitCount := 0
	for _, item := range items {
		if item.ItemType == "fishbait" {
			baitCount = item.Quantity
			break
		}
	}

	// 鱼仓库
	fishItems, _ := model.GetFishItems(tgId)
	totalValue := 0

	text := "🎣 钓鱼\n\n"
	text += fmt.Sprintf("🪱 鱼饵: %d个\n", baitCount)

	// 冷却时间
	lastFish := model.GetLastFishTime(tgId)
	now := time.Now().Unix()
	cd := int64(common.TgBotFishCooldown)
	cdRemain := lastFish + cd - now
	if cdRemain > 0 {
		text += fmt.Sprintf("⏱️ 冷却中: %d秒\n", cdRemain)
	} else {
		text += "✅ 可以钓鱼\n"
	}

	text += "\n📦 鱼仓库:\n"
	if len(fishItems) == 0 {
		text += "  (空)\n"
	} else {
		for _, fi := range fishItems {
			fishKey := fi.ItemType[5:] // remove "fish_" prefix
			fd := fishTypeMap[fishKey]
			if fd != nil {
				mPrice := applyMarket(fd.SellPrice, "fish_"+fishKey)
				val := mPrice * fi.Quantity
				totalValue += val
				mPct := getMarketMultiplier("fish_" + fishKey)
				text += fmt.Sprintf("  %s %s ×%d [%s] %s(%d%%)\n", fd.Emoji, fd.Name, fi.Quantity, fd.Rarity, farmQuotaStr(val), mPct)
			}
		}
		text += fmt.Sprintf("\n💰 总价值: %s\n", farmQuotaStr(totalValue))
	}

	// 市场倒计时
	ensureMarketFresh()
	marketMu.RLock()
	nextRefresh := marketNextUpdate - now
	marketMu.RUnlock()
	if nextRefresh > 0 {
		text += fmt.Sprintf("\n📈 市场%dh后刷新\n", nextRefresh/3600+1)
	}

	text += "\n📊 鱼种概率:\n"
	for _, ft := range fishTypes {
		mPrice := applyMarket(ft.SellPrice, "fish_"+ft.Key)
		mPct := getMarketMultiplier("fish_" + ft.Key)
		text += fmt.Sprintf("  %s %s [%s] %d%% %s(%d%%)\n", ft.Emoji, ft.Name, ft.Rarity, ft.Weight*100/fishTotalWeight, farmQuotaStr(mPrice), mPct)
	}
	text += fmt.Sprintf("  🗑️ 空军 %d%%\n", fishNothingWeight*100/fishTotalWeight)

	var rows [][]TgInlineKeyboardButton
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🎣 开始钓鱼", CallbackData: "farm_dofish"},
	})
	if totalValue > 0 {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("💰 出售全部 (%s)", farmQuotaStr(totalValue)), CallbackData: "farm_sellfish"},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmFish(chatId int64, editMsgId int, tgId string, from *TgUser) {
	// 冷却检查
	lastFish := model.GetLastFishTime(tgId)
	now := time.Now().Unix()
	cd := int64(common.TgBotFishCooldown)
	if now < lastFish+cd {
		remain := lastFish + cd - now
		farmSend(chatId, editMsgId, fmt.Sprintf("⏱️ 钓鱼冷却中，还需等待 %d 秒", remain), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
			},
		}, from)
		return
	}

	// 扣鱼饵
	err := model.DecrementFarmItem(tgId, "fishbait")
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 没有鱼饵！请先到商店购买🪱鱼饵", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🏪 去商店", CallbackData: "farm_shop"}},
				{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
			},
		}, from)
		return
	}

	// 记录冷却
	model.SetLastFishTime(tgId, now)

	// 随机钓鱼
	fish := randomFish()
	if fish == nil {
		// 空军
		model.AddFarmLog(tgId, "fish", -common.TgBotFishBaitPrice, "钓鱼空军")
		farmSend(chatId, editMsgId, "🎣 甩竿...\n\n🗑️ 空军！什么都没钓到...\n\n消耗了1个鱼饵", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🎣 再钓一次", CallbackData: "farm_dofish"}},
				{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
			},
		}, from)
		return
	}

	// 钓到鱼了
	_ = model.IncrementFarmItem(tgId, "fish_"+fish.Key, 1)
	model.AddFarmLog(tgId, "fish", 0, fmt.Sprintf("钓到%s%s[%s]", fish.Emoji, fish.Name, fish.Rarity))

	rarityMsg := ""
	if fish.Rarity == "稀有" {
		rarityMsg = "🎉 不错！"
	} else if fish.Rarity == "史诗" {
		rarityMsg = "🎊 太棒了！！"
	} else if fish.Rarity == "传说" {
		rarityMsg = "🏆🎊 传说级！！！"
	}

	text := fmt.Sprintf("🎣 甩竿...\n\n%s 钓到了 %s %s！\n品质: [%s]\n价值: %s\n%s",
		rarityMsg, fish.Emoji, fish.Name, fish.Rarity, farmQuotaStr(fish.SellPrice), rarityMsg)

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🎣 再钓一次", CallbackData: "farm_dofish"}},
			{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
		},
	}, from)
}

func doFarmSellFish(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 用户不存在", nil, from)
		return
	}

	fishItems, _ := model.GetFishItems(tgId)
	if len(fishItems) == 0 {
		farmSend(chatId, editMsgId, "❌ 鱼仓库为空", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔙 返回钓鱼", CallbackData: "farm_fish"}},
			},
		}, from)
		return
	}

	totalValue := 0
	totalCount := 0
	for _, fi := range fishItems {
		fishKey := fi.ItemType[5:]
		fd := fishTypeMap[fishKey]
		if fd != nil {
			totalValue += applyMarket(fd.SellPrice, "fish_"+fishKey) * fi.Quantity
			totalCount += fi.Quantity
		}
	}

	_, _ = model.SellAllFish(tgId)
	_ = model.IncreaseUserQuota(user.Id, totalValue, true)
	model.AddFarmLog(tgId, "fish_sell", totalValue, fmt.Sprintf("出售%d条鱼", totalCount))

	farmSend(chatId, editMsgId, fmt.Sprintf("💰 出售成功！\n\n卖出 %d 条鱼\n收入 %s（含市场波动）", totalCount, farmQuotaStr(totalValue)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🎣 继续钓鱼", CallbackData: "farm_fish"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 市场行情 ==========

func showFarmMarket(chatId int64, editMsgId int, tgId string, from *TgUser) {
	ensureMarketFresh()

	now := time.Now().Unix()
	marketMu.RLock()
	nextRefresh := marketNextUpdate - now
	marketMu.RUnlock()
	if nextRefresh < 0 {
		nextRefresh = 0
	}

	text := fmt.Sprintf("📈 市场行情（%dh刷新一次，%dh后刷新）\n\n", common.TgBotMarketRefreshHours, nextRefresh/3600+1)

	text += "🌾 作物:\n"
	for _, crop := range farmCrops {
		m := getMarketMultiplier("crop_" + crop.Key)
		tag := marketTag(m)
		text += fmt.Sprintf("  %s %s %d%% %s %s\n", crop.Emoji, crop.Name, m, tag, farmQuotaStr(applyMarket(crop.UnitPrice, "crop_"+crop.Key)))
	}

	text += "\n🐟 鱼类:\n"
	for _, fish := range fishTypes {
		m := getMarketMultiplier("fish_" + fish.Key)
		tag := marketTag(m)
		text += fmt.Sprintf("  %s %s %d%% %s %s\n", fish.Emoji, fish.Name, m, tag, farmQuotaStr(applyMarket(fish.SellPrice, "fish_"+fish.Key)))
	}

	text += "\n🥩 肉类:\n"
	for _, a := range ranchAnimals {
		m := getMarketMultiplier("meat_" + a.Key)
		tag := marketTag(m)
		text += fmt.Sprintf("  %s %s肉 %d%% %s %s\n", a.Emoji, a.Name, m, tag, farmQuotaStr(applyMarket(*a.MeatPrice, "meat_"+a.Key)))
	}

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func marketTag(m int) string {
	if m >= 180 {
		return "🔥暴涨"
	} else if m >= 140 {
		return "📈大涨"
	} else if m >= 110 {
		return "📈涨"
	} else if m >= 90 {
		return "➡️稳"
	} else if m >= 60 {
		return "📉跌"
	}
	return "📉暴跌"
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
