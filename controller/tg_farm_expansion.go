package controller

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ========== 交易辅助 ==========

func getWarehouseItemMarketPrice(cropType string) int {
	for _, cr := range farmCrops {
		if cr.Key == cropType {
			return applyMarket(cr.UnitPrice, "crop_"+cr.Key)
		}
	}
	for _, f := range fishTypes {
		if "fish_"+f.Key == cropType || f.Key == cropType {
			return applyMarket(f.SellPrice, "fish_"+f.Key)
		}
	}
	for _, a := range ranchAnimals {
		if "meat_"+a.Key == cropType || a.Key == cropType {
			return applyMarket(*a.MeatPrice, "meat_"+a.Key)
		}
	}
	for _, r := range recipes {
		if "recipe_"+r.Key == cropType || r.Key == cropType {
			return applyMarket(r.SellPrice, "recipe_"+r.Key)
		}
	}
	return 500000
}

// ========== 图鉴 ==========

func backfillCollections(tgId string) {
	existing, _ := model.GetCollections(tgId)
	have := make(map[string]bool)
	for _, c := range existing {
		have[c.Category+"_"+c.ItemKey] = true
	}
	record := func(cat, key string) {
		if !have[cat+"_"+key] {
			model.RecordCollection(tgId, cat, key, 1)
			have[cat+"_"+key] = true
		}
	}

	// 1) Warehouse items
	whItems, _ := model.GetWarehouseItems(tgId)
	for _, wh := range whItems {
		switch wh.Category {
		case "crop":
			record("crop", wh.CropType)
		case "fish":
			key := wh.CropType
			if len(key) > 5 && key[:5] == "fish_" {
				key = key[5:]
			}
			record("fish", key)
		case "meat":
			key := wh.CropType
			if len(key) > 5 && key[:5] == "meat_" {
				key = key[5:]
			}
			record("animal", key)
		case "recipe":
			key := wh.CropType
			if len(key) > 7 && key[:7] == "recipe_" {
				key = key[7:]
			}
			record("recipe", key)
		}
	}

	// 2) Fish in backpack
	fishItems, _ := model.GetFishItems(tgId)
	for _, fi := range fishItems {
		if len(fi.ItemType) > 5 {
			record("fish", fi.ItemType[5:])
		}
	}

	// 3) Current plots with crops
	plots, _ := model.GetOrCreateFarmPlots(tgId)
	for _, plot := range plots {
		if plot.CropType != "" && plot.Status > 0 {
			record("crop", plot.CropType)
		}
	}

	// 4) Ranch animals
	animals, _ := model.GetRanchAnimals(tgId)
	for _, a := range animals {
		if a.AnimalType != "" {
			record("animal", a.AnimalType)
		}
	}

	// 5) Farm logs — parse known patterns
	logs, _, _ := model.GetFarmLogs(tgId, 200, 0)
	for _, log := range logs {
		detail := log.Detail
		switch log.Action {
		case "plant":
			for _, c := range farmCrops {
				if strings.Contains(detail, c.Name) {
					record("crop", c.Key)
				}
			}
		case "fish":
			for _, f := range fishTypes {
				if strings.Contains(detail, f.Name) {
					record("fish", f.Key)
				}
			}
		case "ranch_sell", "ranch_store":
			for _, a := range ranchAnimals {
				if strings.Contains(detail, a.Name) {
					record("animal", a.Key)
				}
			}
		case "craft_sell", "craft_store":
			for _, r := range recipes {
				if strings.Contains(detail, r.Name) {
					record("recipe", r.Key)
				}
			}
		case "ranch_buy":
			for _, a := range ranchAnimals {
				if strings.Contains(detail, a.Name) {
					record("animal", a.Key)
				}
			}
		}
	}
}

func showFarmEncyclopedia(chatId int64, editMsgId int, tgId string, from *TgUser) {
	backfillCollections(tgId)

	collections, _ := model.GetCollections(tgId)
	collectedMap := make(map[string]bool)
	for _, c := range collections {
		collectedMap[c.Category+"_"+c.ItemKey] = true
	}

	type catInfo struct {
		name    string
		emoji   string
		key     string
		items   []struct{ key, name, emoji string }
		total   int
		found   int
		claimed bool
	}

	categories := []catInfo{
		{name: "农作物", emoji: "🌾", key: "crop"},
		{name: "鱼类", emoji: "🐟", key: "fish"},
		{name: "动物", emoji: "🐄", key: "animal"},
		{name: "加工品", emoji: "🏭", key: "recipe"},
	}

	for i := range categories {
		cat := &categories[i]
		switch cat.key {
		case "crop":
			for _, c := range farmCrops {
				cat.items = append(cat.items, struct{ key, name, emoji string }{c.Key, c.Name, c.Emoji})
			}
		case "fish":
			for _, f := range fishTypes {
				cat.items = append(cat.items, struct{ key, name, emoji string }{f.Key, f.Name, f.Emoji})
			}
		case "animal":
			for _, a := range ranchAnimals {
				cat.items = append(cat.items, struct{ key, name, emoji string }{a.Key, a.Name, a.Emoji})
			}
		case "recipe":
			for _, r := range recipes {
				cat.items = append(cat.items, struct{ key, name, emoji string }{r.Key, r.Name, r.Emoji})
			}
		}
		cat.total = len(cat.items)
		for _, it := range cat.items {
			if collectedMap[cat.key+"_"+it.key] {
				cat.found++
			}
		}
		cat.claimed = model.HasCollectionReward(tgId, cat.key)
	}

	totalItems := 0
	totalFound := 0
	for _, cat := range categories {
		totalItems += cat.total
		totalFound += cat.found
	}

	text := fmt.Sprintf("📖 图鉴 (%d/%d 已发现)\n\n", totalFound, totalItems)

	for _, cat := range categories {
		text += fmt.Sprintf("%s %s (%d/%d)", cat.emoji, cat.name, cat.found, cat.total)
		if cat.found == cat.total && !cat.claimed {
			text += " ✨可领奖"
		} else if cat.claimed {
			text += " ✅"
		}
		text += "\n"
		for _, it := range cat.items {
			if collectedMap[cat.key+"_"+it.key] {
				text += fmt.Sprintf("  %s %s\n", it.emoji, it.name)
			} else {
				text += "  ❓ ???\n"
			}
		}
		text += "\n"
	}

	var rows [][]TgInlineKeyboardButton
	for _, cat := range categories {
		if cat.found == cat.total && !cat.claimed {
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("🎁 领取%s奖励", cat.name), CallbackData: "farm_eclaim_" + cat.key},
			})
		}
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmClaimCollection(chatId int64, editMsgId int, tgId string, category string, from *TgUser) {
	if model.HasCollectionReward(tgId, category) {
		farmSend(chatId, editMsgId, "❌ 已经领取过了", nil, from)
		return
	}

	catNames := map[string]string{"crop": "农作物", "fish": "鱼类", "animal": "动物", "recipe": "加工品"}
	catName, ok := catNames[category]
	if !ok {
		farmSend(chatId, editMsgId, "❌ 未知类别", nil, from)
		return
	}

	reward := 500000
	prestige := model.GetPrestigeLevel(tgId)
	if prestige > 0 {
		reward = reward + reward*prestige*common.TgBotFarmPrestigeBonusPerLevel/100
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	_ = model.ClaimCollectionReward(tgId, category)
	model.IncreaseUserQuota(user.Id, reward, true)
	model.AddFarmLog(tgId, "encyclopedia", reward, "📖 图鉴奖励: "+catName)

	farmSend(chatId, editMsgId, fmt.Sprintf("🎉 恭喜！领取%s图鉴奖励 %s！", catName, farmQuotaStr(reward)), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "📖 返回图鉴", CallbackData: "farm_ency"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 排行 ==========

func showFarmLeaderboard(chatId int64, editMsgId int, tgId string, boardType string, from *TgUser) {
	boardNames := map[string]string{
		"balance": "💰 资产", "level": "⭐ 等级",
		"harvest": "🌾 收获", "prestige": "🔄 转生",
	}
	boardName := boardNames[boardType]
	if boardName == "" {
		boardType = "balance"
		boardName = "💰 资产"
	}

	entries, _ := model.GetFarmLeaderboard(boardType, 10)
	myRank := model.GetFarmRank(tgId, boardType)

	text := fmt.Sprintf("🏅 排行榜 — %s\n\n", boardName)

	medals := []string{"🥇", "🥈", "🥉"}
	for i, e := range entries {
		rank := i + 1
		prefix := fmt.Sprintf("#%d", rank)
		if rank <= 3 {
			prefix = medals[rank-1]
		}
		me := ""
		if e.TelegramId == tgId {
			me = " ← 你"
		}
		name := e.Username
		if name == "" {
			name = e.TelegramId
		}
		valStr := fmt.Sprintf("%d", e.Value)
		if boardType == "balance" {
			valStr = farmQuotaStr(int(e.Value))
		}
		text += fmt.Sprintf("%s %s: %s%s\n", prefix, name, valStr, me)
	}

	if myRank > 0 {
		text += fmt.Sprintf("\n📊 我的排名: #%d\n", myRank)
	}

	var rows [][]TgInlineKeyboardButton
	typeOrder := []struct{ key, label string }{
		{"balance", "💰资产"}, {"level", "⭐等级"},
		{"harvest", "🌾收获"}, {"prestige", "🔄转生"},
	}
	var btnRow []TgInlineKeyboardButton
	for _, tp := range typeOrder {
		label := tp.label
		if tp.key == boardType {
			label = "✅" + label
		}
		btnRow = append(btnRow, TgInlineKeyboardButton{Text: label, CallbackData: "farm_rank_" + tp.key})
	}
	rows = append(rows, btnRow)
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

// ========== 交易 ==========

func showFarmTradeMarket(chatId int64, editMsgId int, tgId string, from *TgUser) {
	trades, _, _ := model.GetOpenTrades(20, 0)

	text := "🔄 玩家交易市场\n\n"
	if len(trades) == 0 {
		text += "📭 暂无挂单\n"
	} else {
		text += "📋 当前挂单:\n"
		for _, t := range trades {
			price := float64(t.PricePerUnit) / 500000.0
			total := price * float64(t.Quantity)
			fee := total * float64(common.TgBotFarmTradeFee) / 100
			text += fmt.Sprintf("  %s %s ×%d — $%.2f/个 (共$%.2f+$%.2f手续费)\n    卖家: %s\n",
				t.ItemEmoji, t.ItemName, t.Quantity, price, total, fee, t.SellerName)
		}
	}

	myOpenCount := model.CountMyOpenTrades(tgId)
	if myOpenCount > 0 {
		text += fmt.Sprintf("\n📤 我有 %d 个挂单\n", myOpenCount)
	}

	var rows [][]TgInlineKeyboardButton
	for _, t := range trades {
		if t.SellerId != tgId {
			price := float64(t.PricePerUnit) / 500000.0
			total := price * float64(t.Quantity)
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("🛒 买 %s%s×%d ($%.2f)", t.ItemEmoji, t.ItemName, t.Quantity, total),
					CallbackData: fmt.Sprintf("farm_tbuy_%d", t.Id)},
			})
		} else {
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("❌ 取消 %s%s×%d", t.ItemEmoji, t.ItemName, t.Quantity),
					CallbackData: fmt.Sprintf("farm_tcancel_%d", t.Id)},
			})
		}
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "📤 挂单出售", CallbackData: "farm_tsell"},
	})
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func showFarmTradeSell(chatId int64, editMsgId int, tgId string, from *TgUser) {
	if int(model.CountMyOpenTrades(tgId)) >= common.TgBotFarmTradeMaxListings {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 挂单数量已达上限（%d个）", common.TgBotFarmTradeMaxListings), &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔄 返回交易", CallbackData: "farm_trade"}},
			},
		}, from)
		return
	}

	items, _ := model.GetWarehouseItems(tgId)
	if len(items) == 0 {
		farmSend(chatId, editMsgId, "📭 仓库为空，没有可出售的物品！\n\n请先收获作物存入仓库。", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔄 返回交易", CallbackData: "farm_trade"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
		return
	}

	text := "📤 选择要出售的物品\n\n📦 仓库物品:\n"
	var rows [][]TgInlineKeyboardButton
	for _, item := range items {
		if item.Quantity <= 0 {
			continue
		}
		itemName, itemEmoji, _ := getTradeItemInfo(item.CropType)
		marketPrice := getWarehouseItemMarketPrice(item.CropType)
		priceStr := fmt.Sprintf("$%.2f", float64(marketPrice)/500000.0)
		text += fmt.Sprintf("  %s %s ×%d (市价%s/个)\n", itemEmoji, itemName, item.Quantity, priceStr)
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("📤 %s%s ×%d (%s/个)", itemEmoji, itemName, item.Quantity, priceStr),
				CallbackData: fmt.Sprintf("farm_tlist_%s", item.CropType)},
		})
	}
	text += fmt.Sprintf("\n💡 将以当前市场价挂单出售\n💰 买家需额外支付 %d%% 手续费", common.TgBotFarmTradeFee)

	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔄 返回交易", CallbackData: "farm_trade"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmTradeList(chatId int64, editMsgId int, tgId string, cropType string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	if int(model.CountMyOpenTrades(tgId)) >= common.TgBotFarmTradeMaxListings {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 挂单数量已达上限（%d个）", common.TgBotFarmTradeMaxListings), nil, from)
		return
	}

	whItem, err := model.GetWarehouseItem(tgId, cropType)
	if err != nil || whItem.Quantity <= 0 {
		farmSend(chatId, editMsgId, "❌ 仓库中没有该物品", &TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "📤 返回挂单", CallbackData: "farm_tsell"}},
			},
		}, from)
		return
	}

	quantity := whItem.Quantity
	itemName, itemEmoji, category := getTradeItemInfo(cropType)
	marketPrice := getWarehouseItemMarketPrice(cropType)

	err = model.RemoveFromWarehouse(tgId, cropType, quantity)
	if err != nil {
		farmSend(chatId, editMsgId, "❌ 仓库物品不足", nil, from)
		return
	}

	trade := &model.TgFarmTrade{
		SellerId: tgId, SellerName: user.Username, Category: category,
		ItemKey: cropType, ItemName: itemName, ItemEmoji: itemEmoji,
		Quantity: quantity, PricePerUnit: marketPrice, Status: 0,
	}
	if err := model.CreateTrade(trade); err != nil {
		_ = model.AddToWarehouseWithCategory(tgId, cropType, quantity, category)
		farmSend(chatId, editMsgId, "❌ 创建挂单失败", nil, from)
		return
	}

	model.AddFarmLog(tgId, "trade", 0, fmt.Sprintf("📤 挂单: %s%s×%d", itemEmoji, itemName, quantity))
	totalPrice := float64(marketPrice*quantity) / 500000.0

	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 挂单成功！\n\n%s %s ×%d\n💰 单价: $%.2f\n💰 总价: $%.2f\n\n等待其他玩家购买...",
		itemEmoji, itemName, quantity, float64(marketPrice)/500000.0, totalPrice),
		&TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "📤 继续挂单", CallbackData: "farm_tsell"}},
				{{Text: "🔄 返回交易", CallbackData: "farm_trade"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
}

func doFarmTradeBuy(chatId int64, editMsgId int, tgId string, tradeId int, from *TgUser) {
	trade, err := model.GetTradeById(tradeId)
	if err != nil || trade == nil || trade.Status != 0 {
		farmSend(chatId, editMsgId, "❌ 该交易不存在或已完成", nil, from)
		return
	}
	if trade.SellerId == tgId {
		farmSend(chatId, editMsgId, "❌ 不能购买自己的挂单", nil, from)
		return
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	totalPrice := trade.PricePerUnit * trade.Quantity
	feeAmount := totalPrice * common.TgBotFarmTradeFee / 100
	totalCost := totalPrice + feeAmount

	if int64(user.Quota) < int64(totalCost) {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！需要 %s（含手续费）", farmQuotaStr(totalCost)), nil, from)
		return
	}

	model.DecreaseUserQuota(user.Id, totalCost)
	model.UpdateTradeStatus(tradeId, 1, tgId)

	_ = model.AddToWarehouseWithCategory(tgId, trade.ItemKey, trade.Quantity, trade.Category)
	model.AddFarmLog(tgId, "trade", -totalCost, fmt.Sprintf("🛒 购买: %s%s×%d", trade.ItemEmoji, trade.ItemName, trade.Quantity))

	var sellerUser model.User
	model.DB.Where("telegram_id = ?", trade.SellerId).First(&sellerUser)
	if sellerUser.Id > 0 {
		model.IncreaseUserQuota(sellerUser.Id, totalPrice, true)
		model.AddFarmLog(trade.SellerId, "trade", totalPrice, fmt.Sprintf("💰 售出: %s%s×%d", trade.ItemEmoji, trade.ItemName, trade.Quantity))
	}

	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 成功购买 %s%s ×%d！花费 %s",
		trade.ItemEmoji, trade.ItemName, trade.Quantity, farmQuotaStr(totalCost)),
		&TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔄 返回交易", CallbackData: "farm_trade"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
}

func doFarmTradeCancel(chatId int64, editMsgId int, tgId string, tradeId int, from *TgUser) {
	trade, err := model.GetTradeById(tradeId)
	if err != nil || trade == nil || trade.Status != 0 {
		farmSend(chatId, editMsgId, "❌ 该交易不存在或已完成", nil, from)
		return
	}
	if trade.SellerId != tgId {
		farmSend(chatId, editMsgId, "❌ 只能取消自己的挂单", nil, from)
		return
	}

	model.UpdateTradeStatus(tradeId, 2, "")
	_ = model.AddToWarehouseWithCategory(tgId, trade.ItemKey, trade.Quantity, trade.Category)
	model.AddFarmLog(tgId, "trade", 0, fmt.Sprintf("↩️ 取消挂单: %s%s×%d", trade.ItemEmoji, trade.ItemName, trade.Quantity))

	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 已取消挂单，%s%s ×%d 已退回仓库",
		trade.ItemEmoji, trade.ItemName, trade.Quantity),
		&TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🔄 返回交易", CallbackData: "farm_trade"}},
				{{Text: "🔙 返回农场", CallbackData: "farm"}},
			},
		}, from)
}

// ========== 游戏 ==========

func showFarmGames(chatId int64, editMsgId int, tgId string, from *TgUser) {
	logs, _ := model.GetRecentGameLogs(tgId, 5)

	text := "🎮 农场小游戏\n\n"
	text += fmt.Sprintf("🎡 幸运转盘 — %s/次\n", farmQuotaStr(common.TgBotFarmWheelPrice))
	text += "  转动转盘赢取奖励！奖池从 0.1x 到 10x\n\n"
	text += fmt.Sprintf("🎰 刮刮卡 — %s/次\n", farmQuotaStr(common.TgBotFarmScratchPrice))
	text += "  刮出3个相同符号赢取奖励！\n"

	if len(logs) > 0 {
		text += "\n📜 最近记录:\n"
		for _, log := range logs {
			gameEmoji := "🎡"
			if log.GameType == "scratch" {
				gameEmoji = "🎰"
			}
			net := log.WinAmount - log.BetAmount
			netSign := "+"
			if net < 0 {
				netSign = ""
			}
			text += fmt.Sprintf("  %s 下注%s → %s (%s%s)\n",
				gameEmoji, farmQuotaStr(log.BetAmount), farmQuotaStr(log.WinAmount),
				netSign, farmQuotaStr(net))
		}
	}

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{
				{Text: fmt.Sprintf("🎡 转盘 (%s)", farmQuotaStr(common.TgBotFarmWheelPrice)), CallbackData: "farm_wheel"},
				{Text: fmt.Sprintf("🎰 刮刮卡 (%s)", farmQuotaStr(common.TgBotFarmScratchPrice)), CallbackData: "farm_scratch"},
			},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func doFarmWheel(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}
	price := common.TgBotFarmWheelPrice
	if int64(user.Quota) < int64(price) {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！需要 %s", farmQuotaStr(price)), nil, from)
		return
	}
	model.DecreaseUserQuota(user.Id, price)

	type sector struct {
		Label string
		Multi float64 // multiplier of price
	}
	sectors := []sector{
		{"💀 0x", 0}, {"🎯 0.5x", 0.5}, {"✨ 1x", 1}, {"💎 1.5x", 1.5},
		{"🌟 2x", 2}, {"🔥 3x", 3}, {"💰 5x", 5}, {"🏆 10x", 10},
	}
	weights := []int{25, 25, 20, 12, 8, 5, 3, 2}
	totalW := 0
	for _, w := range weights {
		totalW += w
	}
	r := rand.Intn(totalW)
	winIdx := 0
	cum := 0
	for i, w := range weights {
		cum += w
		if r < cum {
			winIdx = i
			break
		}
	}
	win := sectors[winIdx]
	actualWin := int(float64(price) * win.Multi)
	prestige := model.GetPrestigeLevel(tgId)
	if prestige > 0 && actualWin > 0 {
		actualWin = actualWin + actualWin*prestige*common.TgBotFarmPrestigeBonusPerLevel/100
	}
	if actualWin > 0 {
		model.IncreaseUserQuota(user.Id, actualWin, true)
	}
	net := actualWin - price
	model.CreateGameLog(tgId, "wheel", price, actualWin)
	netSign := "+"
	if net < 0 {
		netSign = ""
	}
	model.AddFarmLog(tgId, "game", net, "🎡 转盘: "+win.Label)

	text := fmt.Sprintf("🎡 转盘结果: %s\n\n下注: %s\n中奖: %s\n净收益: %s%s",
		win.Label, farmQuotaStr(price), farmQuotaStr(actualWin), netSign, farmQuotaStr(net))

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{
				{Text: "🎡 再来一次", CallbackData: "farm_wheel"},
				{Text: "🎰 刮刮卡", CallbackData: "farm_scratch"},
			},
			{{Text: "🎮 返回游戏", CallbackData: "farm_game"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

func doFarmScratch(chatId int64, editMsgId int, tgId string, from *TgUser) {
	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}
	price := common.TgBotFarmScratchPrice
	if int64(user.Quota) < int64(price) {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！需要 %s", farmQuotaStr(price)), nil, from)
		return
	}
	model.DecreaseUserQuota(user.Id, price)

	type scratchPrize struct {
		Symbol string
		Label  string
		Multi  float64
	}
	prizes := []scratchPrize{
		{"🍒", "樱桃", 1}, {"🍋", "柠檬", 1.5}, {"🍊", "橘子", 2},
		{"🍇", "葡萄", 3}, {"💎", "钻石", 5}, {"👑", "皇冠", 10},
	}
	symbols := []string{"🍒", "🍋", "🍊", "🍇", "💎", "👑"}

	grid := [3][3]string{}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			grid[i][j] = symbols[rand.Intn(len(symbols))]
		}
	}

	winChance := rand.Intn(100)
	var winPrize *scratchPrize
	if winChance < 30 {
		var idx int
		r := rand.Intn(100)
		if r < 40 {
			idx = 0
		} else if r < 65 {
			idx = 1
		} else if r < 82 {
			idx = 2
		} else if r < 93 {
			idx = 3
		} else if r < 98 {
			idx = 4
		} else {
			idx = 5
		}
		winPrize = &prizes[idx]
		row := rand.Intn(3)
		for j := 0; j < 3; j++ {
			grid[row][j] = winPrize.Symbol
		}
	}

	text := "🎰 刮刮卡\n\n"
	for i := 0; i < 3; i++ {
		text += fmt.Sprintf("  %s %s %s\n", grid[i][0], grid[i][1], grid[i][2])
	}

	actualWin := 0
	if winPrize != nil {
		actualWin = int(float64(price) * winPrize.Multi)
		prestige := model.GetPrestigeLevel(tgId)
		if prestige > 0 && actualWin > 0 {
			actualWin = actualWin + actualWin*prestige*common.TgBotFarmPrestigeBonusPerLevel/100
		}
	}
	if actualWin > 0 {
		model.IncreaseUserQuota(user.Id, actualWin, true)
	}
	net := actualWin - price
	model.CreateGameLog(tgId, "scratch", price, actualWin)

	netSign := "+"
	if net < 0 {
		netSign = ""
	}
	if winPrize != nil {
		text += fmt.Sprintf("\n🎉 %s×3 中奖: %s (%s%s)\n", winPrize.Symbol, farmQuotaStr(actualWin), netSign, farmQuotaStr(net))
		model.AddFarmLog(tgId, "game", net, "🎰 刮刮卡: "+winPrize.Label+"×3")
	} else {
		text += fmt.Sprintf("\n😢 未中奖 (%s)\n", farmQuotaStr(net))
		model.AddFarmLog(tgId, "game", net, "🎰 刮刮卡: 未中奖")
	}

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{
				{Text: "🎰 再来一张", CallbackData: "farm_scratch"},
				{Text: "🎡 转盘", CallbackData: "farm_wheel"},
			},
			{{Text: "🎮 返回游戏", CallbackData: "farm_game"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 自动化 ==========

func showFarmAutomation(chatId int64, editMsgId int, tgId string, from *TgUser) {
	type autoItem struct {
		Type  string
		Name  string
		Emoji string
		Desc  string
		Price int
	}
	items := []autoItem{
		{"irrigation", "自动灌溉", "💧", "自动浇水，防止枯萎", common.TgBotFarmIrrigationPrice},
		{"auto_feeder", "自动喂食器", "🍖", "自动喂养牧场动物", common.TgBotFarmAutoFeederPrice},
		{"scarecrow", "稻草人", "🎃", fmt.Sprintf("降低%d%%偷菜成功率", common.TgBotFarmScarecrowDefenseRate), common.TgBotFarmScarecrowPrice},
	}

	automations, _ := model.GetAutomations(tgId)
	installedMap := make(map[string]bool)
	for _, a := range automations {
		installedMap[a.Type] = true
	}

	text := "⚡ 自动化设施\n\n"
	for _, it := range items {
		status := "❌ 未安装"
		if installedMap[it.Type] {
			status = "✅ 已安装"
		}
		text += fmt.Sprintf("%s %s — %s\n  %s | %s %s\n\n", it.Emoji, it.Name, status, it.Desc, "价格:", farmQuotaStr(it.Price))
	}

	var rows [][]TgInlineKeyboardButton
	for _, it := range items {
		if !installedMap[it.Type] {
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("%s 购买%s (%s)", it.Emoji, it.Name, farmQuotaStr(it.Price)),
					CallbackData: "farm_abuy_" + it.Type},
			})
		}
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmBuyAutomation(chatId int64, editMsgId int, tgId string, autoType string, from *TgUser) {
	prices := map[string]int{
		"irrigation":  common.TgBotFarmIrrigationPrice,
		"auto_feeder": common.TgBotFarmAutoFeederPrice,
		"scarecrow":   common.TgBotFarmScarecrowPrice,
	}
	names := map[string]string{
		"irrigation": "💧 自动灌溉", "auto_feeder": "🍖 自动喂食器", "scarecrow": "🎃 稻草人",
	}

	price, ok := prices[autoType]
	if !ok {
		farmSend(chatId, editMsgId, "❌ 未知设施类型", nil, from)
		return
	}
	if model.HasAutomation(tgId, autoType) {
		farmSend(chatId, editMsgId, "❌ 已安装该设施", nil, from)
		return
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}
	if int64(user.Quota) < int64(price) {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！需要 %s", farmQuotaStr(price)), nil, from)
		return
	}

	model.DecreaseUserQuota(user.Id, price)
	model.CreateAutomation(tgId, autoType)
	model.AddFarmLog(tgId, "automation", -price, "⚡ 安装: "+names[autoType])

	farmSend(chatId, editMsgId, fmt.Sprintf("✅ 成功安装 %s！", names[autoType]), &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{{Text: "⚡ 返回自动化", CallbackData: "farm_auto"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 转生 ==========

func showFarmPrestige(chatId int64, editMsgId int, tgId string, from *TgUser) {
	level := model.GetFarmLevel(tgId)
	prestige := model.GetPrestigeLevel(tgId)
	bonus := prestige * common.TgBotFarmPrestigeBonusPerLevel
	nextBonus := (prestige + 1) * common.TgBotFarmPrestigeBonusPerLevel

	text := "🔄 转生系统\n\n"
	text += "满级后重置进度，获得永久收入加成。\n\n"
	text += fmt.Sprintf("📊 当前等级: Lv.%d\n", level)
	text += fmt.Sprintf("🔄 转生次数: %d\n", prestige)
	text += fmt.Sprintf("💰 当前加成: +%d%%\n", bonus)
	text += fmt.Sprintf("📈 转生后加成: +%d%%\n", nextBonus)
	text += fmt.Sprintf("🎯 需要等级: Lv.%d\n", common.TgBotFarmPrestigeMinLevel)
	text += "\n⚠️ 转生将重置:\n等级、地块、仓库、狗、牧场、加工\n保留: 成就、图鉴\n获得: 永久收入加成"

	canPrestige := level >= common.TgBotFarmPrestigeMinLevel

	var rows [][]TgInlineKeyboardButton
	if canPrestige {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🔄 确认转生 (+%d%%)", common.TgBotFarmPrestigeBonusPerLevel), CallbackData: "farm_doprestige"},
		})
	} else {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🔒 需要 Lv.%d (当前 Lv.%d)", common.TgBotFarmPrestigeMinLevel, level), CallbackData: "farm_prestige"},
		})
	}
	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

func doFarmPrestige(chatId int64, editMsgId int, tgId string, from *TgUser) {
	level := model.GetFarmLevel(tgId)
	if level < common.TgBotFarmPrestigeMinLevel {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 需要 Lv.%d 才能转生（当前 Lv.%d）", common.TgBotFarmPrestigeMinLevel, level), nil, from)
		return
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	currentPrestige := model.GetPrestigeLevel(tgId)
	newPrestige := currentPrestige + 1

	model.ResetFarmForPrestige(tgId)
	model.SetPrestigeLevel(tgId, newPrestige)
	model.CreatePrestigeRecord(tgId, newPrestige)

	prestigeReward := 25000000
	model.IncreaseUserQuota(user.Id, prestigeReward, true)
	model.AddFarmLog(tgId, "prestige", prestigeReward, fmt.Sprintf("🔄 转生到第%d世", newPrestige))

	newBonus := newPrestige * common.TgBotFarmPrestigeBonusPerLevel

	farmSend(chatId, editMsgId, fmt.Sprintf("🎉 恭喜转生成功！\n\n🔄 转生次数: %d\n💰 收入加成: +%d%%\n🎁 转生奖励: %s\n\n所有进度已重置，开始新的旅程吧！",
		newPrestige, newBonus, farmQuotaStr(prestigeReward)),
		&TgInlineKeyboardMarkup{
			InlineKeyboard: [][]TgInlineKeyboardButton{
				{{Text: "🌾 开始新旅程", CallbackData: "farm"}},
			},
		}, from)
}
