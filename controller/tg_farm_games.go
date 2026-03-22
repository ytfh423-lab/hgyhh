package controller

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// scoreToGameResult 根据前端引擎传来的 score(0~1) 决定倍率和结果文案
func scoreToGameResult(gameKey string, score float64, gameName, emoji string) (string, float64) {
	// 加入少量随机浮动，避免完全可预测
	jitter := (rand.Float64() - 0.5) * 0.1 // ±5%
	adj := score + jitter
	if adj < 0 {
		adj = 0
	}
	if adj > 1 {
		adj = 1
	}

	fourXThreshold := 0.95
	threeXThreshold := 0.8
	twoXThreshold := 0.6
	oneHalfThreshold := 0.4
	halfThreshold := 0.2
	if gameKey == "thresh" {
		fourXThreshold = 0.995
		threeXThreshold = 0.9
		twoXThreshold = 0.72
		oneHalfThreshold = 0.52
		halfThreshold = 0.3
	}

	var label string
	var multi float64
	switch {
	case adj >= fourXThreshold:
		label = "🏆 完美表现！4倍奖励！"
		multi = 4
	case adj >= threeXThreshold:
		label = "🥇 非常出色！3倍奖励！"
		multi = 3
	case adj >= twoXThreshold:
		label = "🥈 表现不错！2倍奖励！"
		multi = 2
	case adj >= oneHalfThreshold:
		label = "👍 还不错！1.5倍！"
		multi = 1.5
	case adj >= halfThreshold:
		label = "😅 差一点点…0.5倍"
		multi = 0.5
	default:
		label = "😢 下次加油..."
		multi = 0
	}
	text := fmt.Sprintf("%s %s\n\n%s", emoji, gameName, label)
	return text, multi
}

// ========== 农场小游戏定义 ==========

type miniGameDef struct {
	Key   string
	Name  string
	Emoji string
	Desc  string
	Price int
}

var miniGames = []miniGameDef{
	{"bugcatch", "捉虫大赛", "🐛", "在田里捉害虫比拼", 300000},
	{"egghunt", "捡蛋比赛", "🥚", "鸡舍里捡鸡蛋", 200000},
	{"milking", "挤奶比赛", "🐄", "比比谁挤奶多", 350000},
	{"sunflower", "猜向日葵", "🌻", "猜向日葵能长多高", 250000},
	{"beekeep", "采蜜任务", "🐝", "采集蜂蜜避开蜂刺", 400000},
	{"fruitpick", "摘果子", "🍎", "爬树摘果看运气", 300000},
	{"sheepcount", "数羊", "🐑", "数对羊群赢奖励", 200000},
	{"cornrace", "掰玉米", "🌽", "限时掰玉米比赛", 350000},
	{"rooster", "斗鸡", "🐓", "公鸡擂台对决", 400000},
	{"sheepdog", "牧羊犬", "🐕", "指挥牧羊犬赶羊", 350000},
	{"seedling", "育苗", "🌱", "培育优质种苗", 300000},
	{"pumpkin", "南瓜大赛", "🎃", "种出最大的南瓜", 500000},
	{"pigchase", "追猪", "🐷", "抓住逃跑的猪", 300000},
	{"duckherd", "赶鸭子", "🦆", "把鸭子赶进池塘", 250000},
	{"thresh", "打谷", "🌾", "打谷脱粒比速度", 400000},
	{"grape", "踩葡萄", "🍇", "踩葡萄酿酒比赛", 350000},
	{"fishcomp", "钓鱼赛", "🎣", "钓鱼大赛比大小", 400000},
	{"weed", "除草", "🌿", "除草速度大比拼", 250000},
	{"woodchop", "劈柴", "🪓", "劈柴比赛看力量", 300000},
	{"lasso", "套牛", "🐮", "套住奔跑的牛", 450000},
	{"pullcarrot", "拔萝卜", "🥕", "拔萝卜看大小", 200000},
	{"mushroom", "采蘑菇", "🍄", "采蘑菇避开毒蘑菇", 350000},
	{"hatchegg", "孵蛋", "🐣", "孵出稀有品种", 500000},
	{"weather", "天气预报", "🌈", "预测明天天气", 250000},
	{"produce", "农产品评比", "🏆", "参加农产品博览会", 600000},
	{"tame", "驯马", "🐴", "驯服野马", 450000},
	{"scarecrow", "扎稻草人", "👒", "扎稻草人赶乌鸦", 300000},
	{"foxhunt", "赶狐狸", "🦊", "保护鸡舍赶走狐狸", 350000},
	{"harvest", "抢收比赛", "👨‍🌾", "暴风雨前抢收庄稼", 400000},
}

var miniGameMap map[string]*miniGameDef

func init() {
	miniGameMap = make(map[string]*miniGameDef, len(miniGames))
	for i := range miniGames {
		miniGameMap[miniGames[i].Key] = &miniGames[i]
	}
}

const mgGamesPerPage = 10

func showFarmGamesPage(chatId int64, editMsgId int, tgId string, page int, from *TgUser) {
	totalPages := (len(miniGames) + mgGamesPerPage - 1) / mgGamesPerPage
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	text := fmt.Sprintf("🎮 农场小游戏 (%d/%d页)\n\n", page, totalPages)

	if page == 1 {
		text += fmt.Sprintf("🎡 幸运转盘 — %s/次\n", farmQuotaStr(common.TgBotFarmWheelPrice))
		text += fmt.Sprintf("🎰 刮刮卡 — %s/次\n\n", farmQuotaStr(common.TgBotFarmScratchPrice))
	}

	startIdx := (page - 1) * mgGamesPerPage
	endIdx := startIdx + mgGamesPerPage
	if endIdx > len(miniGames) {
		endIdx = len(miniGames)
	}

	for i := startIdx; i < endIdx; i++ {
		g := miniGames[i]
		text += fmt.Sprintf("%s %s — %s | %s\n", g.Emoji, g.Name, farmQuotaStr(g.Price), g.Desc)
	}

	logs, _ := model.GetRecentGameLogs(tgId, 3)
	if len(logs) > 0 {
		text += "\n📜 最近:\n"
		for _, log := range logs {
			net := log.WinAmount - log.BetAmount
			netSign := "+"
			if net < 0 {
				netSign = ""
			}
			text += fmt.Sprintf("  %s → %s%s\n", farmQuotaStr(log.BetAmount), netSign, farmQuotaStr(net))
		}
	}

	var rows [][]TgInlineKeyboardButton
	if page == 1 {
		rows = append(rows, []TgInlineKeyboardButton{
			{Text: fmt.Sprintf("🎡 转盘(%s)", farmQuotaStr(common.TgBotFarmWheelPrice)), CallbackData: "farm_wheel"},
			{Text: fmt.Sprintf("🎰 刮刮卡(%s)", farmQuotaStr(common.TgBotFarmScratchPrice)), CallbackData: "farm_scratch"},
		})
	}

	for i := startIdx; i < endIdx; i++ {
		g := miniGames[i]
		if i+1 < endIdx {
			g2 := miniGames[i+1]
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("%s %s", g.Emoji, g.Name), CallbackData: "farm_g_" + g.Key},
				{Text: fmt.Sprintf("%s %s", g2.Emoji, g2.Name), CallbackData: "farm_g_" + g2.Key},
			})
			i++
		} else {
			rows = append(rows, []TgInlineKeyboardButton{
				{Text: fmt.Sprintf("%s %s", g.Emoji, g.Name), CallbackData: "farm_g_" + g.Key},
			})
		}
	}

	var navRow []TgInlineKeyboardButton
	if page > 1 {
		navRow = append(navRow, TgInlineKeyboardButton{Text: "⬅️ 上一页", CallbackData: fmt.Sprintf("farm_gp_%d", page-1)})
	}
	if page < totalPages {
		navRow = append(navRow, TgInlineKeyboardButton{Text: "➡️ 下一页", CallbackData: fmt.Sprintf("farm_gp_%d", page+1)})
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	rows = append(rows, []TgInlineKeyboardButton{
		{Text: "🔙 返回农场", CallbackData: "farm"},
	})
	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{InlineKeyboard: rows}, from)
}

// ========== 通用小游戏调度 ==========

func doMiniGame(chatId int64, editMsgId int, tgId string, gameKey string, from *TgUser) {
	g := miniGameMap[gameKey]
	if g == nil {
		farmSend(chatId, editMsgId, "❌ 未知游戏", nil, from)
		return
	}

	user, err := getFarmUser(tgId)
	if err != nil {
		farmBindingError(chatId, editMsgId, from)
		return
	}

	price := g.Price
	if int64(user.Quota) < int64(price) {
		farmSend(chatId, editMsgId, fmt.Sprintf("❌ 余额不足！需要 %s", farmQuotaStr(price)), nil, from)
		return
	}
	model.DecreaseUserQuota(user.Id, price)

	var resultText string
	var multi float64

	switch gameKey {
	case "bugcatch":
		resultText, multi = playBugCatch()
	case "egghunt":
		resultText, multi = playEggHunt()
	case "milking":
		resultText, multi = playMilking()
	case "sunflower":
		resultText, multi = playSunflower()
	case "beekeep":
		resultText, multi = playBeekeep()
	case "fruitpick":
		resultText, multi = playFruitPick()
	case "sheepcount":
		resultText, multi = playSheepCount()
	case "cornrace":
		resultText, multi = playCornRace()
	case "rooster":
		resultText, multi = playRooster()
	case "horserace":
		resultText, multi = playHorseRace()
	case "sheepdog":
		resultText, multi = playSheepdog()
	case "seedling":
		resultText, multi = playSeedling()
	case "pumpkin":
		resultText, multi = playPumpkinContest()
	case "pigchase":
		resultText, multi = playPigChase()
	case "duckherd":
		resultText, multi = playDuckHerd()
	case "thresh":
		resultText, multi = playThresh()
	case "grape":
		resultText, multi = playGrapeStomp()
	case "fishcomp":
		resultText, multi = playFishComp()
	case "weed":
		resultText, multi = playWeed()
	case "woodchop":
		resultText, multi = playWoodChop()
	case "lasso":
		resultText, multi = playLasso()
	case "pullcarrot":
		resultText, multi = playPullCarrot()
	case "mushroom":
		resultText, multi = playMushroom()
	case "hatchegg":
		resultText, multi = playHatchEgg()
	case "weather":
		resultText, multi = playWeather()
	case "produce":
		resultText, multi = playProduce()
	case "tame":
		resultText, multi = playTame()
	case "scarecrow":
		resultText, multi = playScarecrow()
	case "foxhunt":
		resultText, multi = playFoxHunt()
	case "harvest":
		resultText, multi = playHarvestRace()
	default:
		resultText = "❌ 游戏错误"
		multi = 1
	}

	actualWin := common.ClampQuotaFloat64(float64(price) * multi)
	if actualWin > 0 {
		model.IncreaseUserQuota(user.Id, actualWin, true)
	}

	net := common.SafeQuotaAdd(actualWin, -price)
	model.CreateGameLog(tgId, gameKey, price, actualWin)
	netSign := "+"
	if net < 0 {
		netSign = ""
	}
	model.AddFarmLog(tgId, "game", net, fmt.Sprintf("%s %s", g.Emoji, g.Name))

	text := fmt.Sprintf("%s %s\n\n%s\n\n下注: %s\n中奖: %s\n净收益: %s%s",
		g.Emoji, g.Name, resultText,
		farmQuotaStr(price), farmQuotaStr(actualWin), netSign, farmQuotaStr(net))

	farmSend(chatId, editMsgId, text, &TgInlineKeyboardMarkup{
		InlineKeyboard: [][]TgInlineKeyboardButton{
			{
				{Text: fmt.Sprintf("%s 再来一次", g.Emoji), CallbackData: "farm_g_" + g.Key},
			},
			{{Text: "🎮 返回游戏", CallbackData: "farm_game"}},
			{{Text: "🔙 返回农场", CallbackData: "farm"}},
		},
	}, from)
}

// ========== 30个农场小游戏实现 ==========

// 1. 捉虫大赛
func playBugCatch() (string, float64) {
	bugs := []struct{ emoji, name string; pts int }{
		{"🐛", "毛毛虫", 1}, {"🐜", "蚂蚁", 1}, {"🦗", "蟋蟀", 2}, {"🐞", "瓢虫", 3}, {"🦋", "蝴蝶", 5},
	}
	text := "🐛 捉虫大赛！翻开菜叶找害虫！\n\n"
	totalPts := 0
	for round := 1; round <= 4; round++ {
		if rand.Intn(100) < 15 {
			text += fmt.Sprintf("第%d片叶子: 空的...\n", round)
		} else {
			b := bugs[rand.Intn(len(bugs))]
			totalPts += b.pts
			text += fmt.Sprintf("第%d片叶子: %s %s +%d分\n", round, b.emoji, b.name, b.pts)
		}
	}
	text += fmt.Sprintf("\n总分: %d\n\n", totalPts)
	switch {
	case totalPts >= 14: return text + "🏆 捉虫高手！5倍！", 5
	case totalPts >= 10: return text + "🎉 大丰收！3倍！", 3
	case totalPts >= 6: return text + "✨ 不错！1.5倍！", 1.5
	case totalPts >= 3: return text + "👍 还行", 0.8
	default: return text + "😢 田里没什么虫...", 0
	}
}

// 2. 捡蛋比赛
func playEggHunt() (string, float64) {
	text := "🥚 鸡舍捡蛋！翻开鸡窝...\n\n"
	total := 0
	for i := 1; i <= 5; i++ {
		r := rand.Intn(100)
		switch {
		case r < 10:
			text += fmt.Sprintf("窝%d: 🐔 母鸡啄你！-1\n", i); total--
		case r < 30:
			text += fmt.Sprintf("窝%d: 空窝\n", i)
		case r < 70:
			text += fmt.Sprintf("窝%d: 🥚×1\n", i); total++
		case r < 90:
			text += fmt.Sprintf("窝%d: 🥚🥚×2\n", i); total += 2
		default:
			text += fmt.Sprintf("窝%d: 🌟金蛋！×3\n", i); total += 3
		}
	}
	if total < 0 { total = 0 }
	text += fmt.Sprintf("\n捡到: %d个蛋\n\n", total)
	switch {
	case total >= 10: return text + "🏆 蛋王！5倍！", 5
	case total >= 7: return text + "🎉 满篮子！3倍！", 3
	case total >= 4: return text + "✨ 不少！1.5倍！", 1.5
	case total >= 2: return text + "👍 凑合", 0.8
	default: return text + "😢 空手而归...", 0
	}
}

// 3. 挤奶比赛
func playMilking() (string, float64) {
	text := "🐄 挤奶比赛开始！\n\n"
	totalMilk := 0
	for round := 1; round <= 3; round++ {
		milk := rand.Intn(40) + 5
		event := ""
		if rand.Intn(10) == 0 { event = " 🐄牛踢了你！"; milk = 0 } else if rand.Intn(8) == 0 { event = " ⭐手感极佳！"; milk *= 2 }
		totalMilk += milk
		text += fmt.Sprintf("第%d轮: %d升%s\n", round, milk, event)
	}
	text += fmt.Sprintf("\n总产量: %d升\n\n", totalMilk)
	switch {
	case totalMilk >= 100: return text + "🏆 挤奶冠军！5倍！", 5
	case totalMilk >= 70: return text + "🎉 产量不错！3倍！", 3
	case totalMilk >= 40: return text + "✨ 够喝了！1.5倍！", 1.5
	case totalMilk >= 20: return text + "👍 少了点", 0.8
	default: return text + "😢 牛不配合...", 0
	}
}

// 4. 猜向日葵
func playSunflower() (string, float64) {
	height := rand.Intn(300) + 50
	guess := rand.Intn(300) + 50
	diff := height - guess
	if diff < 0 { diff = -diff }
	text := fmt.Sprintf("🌻 猜向日葵有多高？\n\n你猜: %dcm\n实际: %dcm\n误差: %dcm\n\n", guess, height, diff)
	switch {
	case diff <= 5: return text + "🏆 精确命中！5倍！", 5
	case diff <= 20: return text + "🎉 很接近！3倍！", 3
	case diff <= 50: return text + "✨ 还可以！1.5倍！", 1.5
	case diff <= 100: return text + "👍 差了点", 0.5
	default: return text + "😢 差太远了...", 0
	}
}

// 5. 采蜜任务
func playBeekeep() (string, float64) {
	text := "🐝 到蜂箱采蜜！小心蜜蜂...\n\n"
	honey := 0
	for step := 1; step <= 5; step++ {
		if rand.Intn(100) < 20 {
			text += fmt.Sprintf("第%d次: 🐝 被蜇了！结束！\n", step)
			break
		}
		h := rand.Intn(3) + 1
		honey += h
		text += fmt.Sprintf("第%d次: 🍯 采到%d罐\n", step, h)
	}
	text += fmt.Sprintf("\n采集: %d罐蜂蜜\n\n", honey)
	switch {
	case honey >= 12: return text + "🏆 采蜜大师！5倍！", 5
	case honey >= 8: return text + "🎉 甜蜜丰收！3倍！", 3
	case honey >= 5: return text + "✨ 够甜了！1.5倍！", 1.5
	case honey >= 1: return text + "👍 聊胜于无", 0.8
	default: return text + "😢 被蜇跑了...", 0
	}
}

// 6. 摘果子
func playFruitPick() (string, float64) {
	fruits := []struct{ emoji, name string; pts int }{
		{"🍎", "苹果", 2}, {"🍐", "梨子", 2}, {"🍑", "桃子", 3}, {"🍒", "樱桃", 4}, {"🌟", "金苹果", 10},
	}
	weights := []int{30, 25, 20, 15, 3}
	totalW := 93
	text := "🍎 爬上果树摘果子！\n\n"
	totalPts := 0
	for round := 1; round <= 4; round++ {
		if rand.Intn(100) < 12 {
			text += fmt.Sprintf("第%d次: 🌿 树枝断了！\n", round); break
		}
		r := rand.Intn(totalW); cum := 0; idx := 0
		for i, w := range weights { cum += w; if r < cum { idx = i; break } }
		f := fruits[idx]; totalPts += f.pts
		text += fmt.Sprintf("第%d次: %s %s +%d\n", round, f.emoji, f.name, f.pts)
	}
	text += fmt.Sprintf("\n总分: %d\n\n", totalPts)
	switch {
	case totalPts >= 18: return text + "🏆 摘果达人！5倍！", 5
	case totalPts >= 12: return text + "🎉 满筐了！3倍！", 3
	case totalPts >= 7: return text + "✨ 不错！1.5倍！", 1.5
	case totalPts >= 3: return text + "👍 还行", 0.8
	default: return text + "😢 没摘到...", 0
	}
}

// 7. 数羊
func playSheepCount() (string, float64) {
	actual := rand.Intn(30) + 10
	guess := actual + rand.Intn(11) - 5
	sheepRow := strings.Repeat("🐑", actual/3)
	text := fmt.Sprintf("🐑 数羊！羊群跑过围栏...\n\n%s\n\n你数到: %d只\n实际: %d只\n\n", sheepRow, guess, actual)
	diff := guess - actual; if diff < 0 { diff = -diff }
	switch {
	case diff == 0: return text + "🏆 一只不差！5倍！", 5
	case diff <= 1: return text + "🎉 几乎对了！3倍！", 3
	case diff <= 3: return text + "✨ 差不多！1.5倍！", 1.5
	default: return text + "😢 数错了...", 0
	}
}

// 8. 掰玉米
func playCornRace() (string, float64) {
	text := "🌽 掰玉米比赛！限时抢收！\n\n"
	total := 0
	for round := 1; round <= 5; round++ {
		corn := rand.Intn(8) + 1
		event := ""
		if rand.Intn(8) == 0 { event = " ⚡手速加倍！"; corn *= 2 }
		total += corn
		text += fmt.Sprintf("第%d趟: 🌽×%d%s\n", round, corn, event)
	}
	text += fmt.Sprintf("\n总共: %d根\n\n", total)
	switch {
	case total >= 35: return text + "🏆 玉米王！5倍！", 5
	case total >= 25: return text + "🎉 大丰收！3倍！", 3
	case total >= 18: return text + "✨ 不错！1.5倍！", 1.5
	case total >= 10: return text + "👍 还行", 0.8
	default: return text + "😢 手太慢了...", 0
	}
}

// 9. 斗鸡
func playRooster() (string, float64) {
	names := []string{"红冠", "铁爪", "金翼", "霸王"}
	myR := rand.Intn(4); enemy := (myR + 1 + rand.Intn(3)) % 4
	text := fmt.Sprintf("🐓 斗鸡擂台！\n你的鸡: 🐓%s  对手: 🐓%s\n\n", names[myR], names[enemy])
	myHP, eHP := 100, 100
	for round := 1; round <= 5 && myHP > 0 && eHP > 0; round++ {
		myDmg := rand.Intn(30) + 10; eDmg := rand.Intn(30) + 10
		eHP -= myDmg; myHP -= eDmg
		text += fmt.Sprintf("R%d: 攻击-%d 受伤-%d | %d vs %d\n", round, myDmg, eDmg, myHP, eHP)
	}
	text += "\n"
	if eHP <= 0 && myHP > 0 { return text + "🏆 你的鸡赢了！3倍！", 3 }
	if myHP <= 0 && eHP <= 0 { return text + "🤝 打平了！", 1 }
	if myHP > eHP { return text + "🎉 判定获胜！2倍！", 2 }
	return text + "😢 你的鸡输了...", 0
}

// 10. 赛马
func playHorseRace() (string, float64) {
	horses := []struct{ emoji, name string }{{"🏇", "烈焰"}, {"🏇", "疾风"}, {"🐴", "闪电"}, {"🐴", "雷鸣"}}
	speeds := make([]int, 4); for i := range speeds { speeds[i] = rand.Intn(100) }
	myHorse := rand.Intn(4)
	type entry struct{ idx, spd int }
	entries := make([]entry, 4); for i := range entries { entries[i] = entry{i, speeds[i]} }
	sort.Slice(entries, func(i, j int) bool { return entries[i].spd > entries[j].spd })
	text := "🏇 农场赛马！\n\n"
	for rank, e := range entries {
		me := ""; if e.idx == myHorse { me = " ← 你" }
		medal := fmt.Sprintf("#%d", rank+1); if rank == 0 { medal = "🥇" } else if rank == 1 { medal = "🥈" }
		text += fmt.Sprintf("%s %s%s %s%s\n", medal, horses[e.idx].emoji, horses[e.idx].name, strings.Repeat("▓", e.spd/10+1), me)
	}
	text += "\n"
	myRank := 0; for i, e := range entries { if e.idx == myHorse { myRank = i; break } }
	switch myRank {
	case 0: return text + "🏆 你的马赢了！4倍！", 4
	case 1: return text + "🥈 第二名！1.5倍！", 1.5
	default: return text + "😢 你的马没赢...", 0
	}
}

// 11. 牧羊犬
func playSheepdog() (string, float64) {
	total := rand.Intn(10) + 5
	herded := 0
	text := fmt.Sprintf("🐕 指挥牧羊犬赶%d只羊入栏！\n\n", total)
	for i := 1; i <= total; i++ {
		r := rand.Intn(100)
		if r < 65 {
			herded++
			text += fmt.Sprintf("🐑%d: ✅入栏 ", i)
		} else if r < 85 {
			text += fmt.Sprintf("🐑%d: ❌跑了 ", i)
		} else {
			text += fmt.Sprintf("🐑%d: 🐕追丢 ", i)
		}
		if i%3 == 0 { text += "\n" }
	}
	pct := herded * 100 / total
	text += fmt.Sprintf("\n\n入栏: %d/%d (%d%%)\n\n", herded, total, pct)
	switch {
	case pct >= 90: return text + "🏆 牧羊高手！5倍！", 5
	case pct >= 70: return text + "🎉 不错！3倍！", 3
	case pct >= 50: return text + "✨ 还行！1.5倍！", 1.5
	case pct >= 30: return text + "👍 努力了", 0.8
	default: return text + "😢 羊全跑了...", 0
	}
}

// 12. 育苗
func playSeedling() (string, float64) {
	steps := []struct{ name, emoji string }{{"选种", "🌰"}, {"播种", "🌱"}, {"浇水", "💧"}, {"施肥", "🧴"}}
	totalScore := 0
	text := "🌱 育苗比赛！培育优质种苗\n\n"
	for _, s := range steps {
		score := rand.Intn(30) + 1; totalScore += score
		stars := "⭐"; if score >= 25 { stars = "⭐⭐⭐" } else if score >= 15 { stars = "⭐⭐" }
		text += fmt.Sprintf("%s %s: %d分 %s\n", s.emoji, s.name, score, stars)
	}
	text += fmt.Sprintf("\n总分: %d/120\n\n", totalScore)
	switch {
	case totalScore >= 100: return text + "🏆 育苗大师！5倍！", 5
	case totalScore >= 80: return text + "🎉 优质种苗！3倍！", 3
	case totalScore >= 60: return text + "✨ 还不错！1.5倍！", 1.5
	case totalScore >= 40: return text + "👍 一般般", 0.8
	default: return text + "😢 种苗没活...", 0
	}
}

// 13. 南瓜大赛
func playPumpkinContest() (string, float64) {
	weight := rand.Intn(500) + 10
	names := []string{"老王", "老李", "老张"}
	rivals := make([]int, 3)
	for i := range rivals { rivals[i] = rand.Intn(500) + 10 }
	text := fmt.Sprintf("🎃 南瓜种植大赛！\n\n浇水... 施肥... 等待成长...\n\n你的南瓜: %d斤\n", weight)
	myRank := 1
	for i, r := range rivals {
		if r > weight { myRank++ }
		text += fmt.Sprintf("  %s的南瓜: %d斤\n", names[i], r)
	}
	text += fmt.Sprintf("\n排名: 第%d名\n\n", myRank)
	switch myRank {
	case 1: return text + "🏆 南瓜冠军！5倍！", 5
	case 2: return text + "🥈 亚军！2倍！", 2
	case 3: return text + "🥉 季军！1倍", 1
	default: return text + "😢 末名...", 0
	}
}

// 14. 追猪
func playPigChase() (string, float64) {
	text := "🐷 猪从猪圈跑了！快追！\n\n"
	caught := false
	for step := 1; step <= 5; step++ {
		r := rand.Intn(100)
		if r < 15+step*8 {
			text += fmt.Sprintf("第%d步: 🎉 抓住了！\n", step); caught = true; break
		} else if r < 50 {
			text += fmt.Sprintf("第%d步: 🐷💨 猪溜走了！\n", step)
		} else {
			text += fmt.Sprintf("第%d步: 🌿 你被绊倒了！\n", step)
		}
	}
	text += "\n"
	if !caught { return text + "😢 猪跑太快了...", 0 }
	return text + "🎉 成功抓回！3倍！", 3
}

// 15. 赶鸭子
func playDuckHerd() (string, float64) {
	total := 8; inPond := 0
	text := fmt.Sprintf("🦆 把%d只鸭子赶进池塘！\n\n", total)
	for i := 1; i <= total; i++ {
		r := rand.Intn(100)
		if r < 60 { inPond++; text += fmt.Sprintf("鸭%d: 🦆→💧入水 ", i)
		} else if r < 85 { text += fmt.Sprintf("鸭%d: 🦆💨跑了 ", i)
		} else { text += fmt.Sprintf("鸭%d: 🦆😤反追你 ", i) }
		if i%2 == 0 { text += "\n" }
	}
	text += fmt.Sprintf("\n\n入水: %d/%d\n\n", inPond, total)
	switch {
	case inPond >= 7: return text + "🏆 赶鸭高手！5倍！", 5
	case inPond >= 5: return text + "🎉 不错！3倍！", 3
	case inPond >= 3: return text + "✨ 还行！1.5倍！", 1.5
	default: return text + "😢 鸭子太皮了...", 0
	}
}

// 16. 打谷
func playThresh() (string, float64) {
	text := "🌾 打谷比赛！用力脱粒！\n\n"
	totalGrain := 0
	for round := 1; round <= 4; round++ {
		grain := rand.Intn(30) + 5; event := ""
		if rand.Intn(6) == 0 { event = " 💪力量爆发！"; grain *= 2 }
		totalGrain += grain
		text += fmt.Sprintf("第%d轮: 🌾 %d斤%s\n", round, grain, event)
	}
	text += fmt.Sprintf("\n总产量: %d斤\n\n", totalGrain)
	switch {
	case totalGrain >= 120: return text + "🏆 打谷王！5倍！", 5
	case totalGrain >= 80: return text + "🎉 大丰收！3倍！", 3
	case totalGrain >= 50: return text + "✨ 不错！1.5倍！", 1.5
	case totalGrain >= 30: return text + "👍 还行", 0.8
	default: return text + "😢 产量太低...", 0
	}
}

// 17. 踩葡萄
func playGrapeStomp() (string, float64) {
	text := "🍇 踩葡萄酿酒比赛！\n\n"
	totalJuice := 0
	for round := 1; round <= 4; round++ {
		juice := rand.Intn(25) + 5; event := ""
		if rand.Intn(8) == 0 { event = " 🤸脚下打滑！"; juice = 0
		} else if rand.Intn(6) == 0 { event = " ⭐完美节奏！"; juice = juice * 3 / 2 }
		totalJuice += juice
		text += fmt.Sprintf("第%d轮: 🍷 %d毫升%s\n", round, juice, event)
	}
	text += fmt.Sprintf("\n总量: %d毫升\n\n", totalJuice)
	switch {
	case totalJuice >= 90: return text + "🏆 酿酒大师！5倍！", 5
	case totalJuice >= 65: return text + "🎉 佳酿！3倍！", 3
	case totalJuice >= 40: return text + "✨ 还行！1.5倍！", 1.5
	case totalJuice >= 20: return text + "👍 少了点", 0.8
	default: return text + "😢 踩不出汁...", 0
	}
}

// 18. 钓鱼赛
func playFishComp() (string, float64) {
	fishes := []struct{ emoji, name string; pts int }{
		{"🐟", "鲫鱼", 1}, {"🐟", "鲤鱼", 2}, {"🐠", "鲶鱼", 3}, {"🦐", "大虾", 4}, {"🐡", "大鱼王", 8},
	}
	weights := []int{30, 25, 20, 15, 5}; totalW := 95
	text := "🎣 农场池塘钓鱼赛！\n\n"
	totalPts := 0
	for round := 1; round <= 3; round++ {
		if rand.Intn(100) < 15 { text += fmt.Sprintf("第%d竿: 🌊 空军...\n", round); continue }
		r := rand.Intn(totalW); cum := 0; idx := 0
		for i, w := range weights { cum += w; if r < cum { idx = i; break } }
		f := fishes[idx]; totalPts += f.pts
		text += fmt.Sprintf("第%d竿: %s %s +%d\n", round, f.emoji, f.name, f.pts)
	}
	text += fmt.Sprintf("\n总分: %d\n\n", totalPts)
	switch {
	case totalPts >= 15: return text + "🏆 钓神！5倍！", 5
	case totalPts >= 10: return text + "🎉 大丰收！3倍！", 3
	case totalPts >= 5: return text + "✨ 不错！1.5倍！", 1.5
	case totalPts >= 2: return text + "👍 勉强", 0.8
	default: return text + "😢 空军...", 0
	}
}

// 19. 除草
func playWeed() (string, float64) {
	total := 10; weeded := 0
	text := fmt.Sprintf("🌿 田地里有%d棵杂草！\n\n", total)
	for i := 1; i <= total; i++ {
		r := rand.Intn(100)
		if r < 60 { weeded++
		} else if r >= 80 { weeded--; text += fmt.Sprintf("  第%d棵: ❌拔到庄稼了！\n", i) }
	}
	if weeded < 0 { weeded = 0 }
	text += fmt.Sprintf("\n除掉: %d/%d\n\n", weeded, total)
	switch {
	case weeded >= 9: return text + "🏆 除草达人！5倍！", 5
	case weeded >= 7: return text + "🎉 干净！3倍！", 3
	case weeded >= 5: return text + "✨ 还行！1.5倍！", 1.5
	case weeded >= 3: return text + "👍 凑合", 0.8
	default: return text + "😢 草太多了...", 0
	}
}

// 20. 劈柴
func playWoodChop() (string, float64) {
	text := "🌲 劈柴比赛！给农场准备柴火！\n\n"
	totalLogs := 0
	for round := 1; round <= 4; round++ {
		logs := rand.Intn(8) + 1; event := ""
		if rand.Intn(7) == 0 { event = " 💪怒劈！"; logs *= 2
		} else if rand.Intn(10) == 0 { event = " ⚠️斧头卡住了！"; logs = 0 }
		totalLogs += logs
		text += fmt.Sprintf("第%d轮: 🪵×%d%s\n", round, logs, event)
	}
	text += fmt.Sprintf("\n总共: %d根\n\n", totalLogs)
	switch {
	case totalLogs >= 30: return text + "🏆 伐木冠军！5倍！", 5
	case totalLogs >= 20: return text + "🎉 好多柴！3倍！", 3
	case totalLogs >= 12: return text + "✨ 够用了！1.5倍！", 1.5
	case totalLogs >= 6: return text + "👍 少了点", 0.8
	default: return text + "😢 力气不够...", 0
	}
}

// 21. 套牛
func playLasso() (string, float64) {
	text := "🐮 套牛比赛！甩出绳套！\n\n"
	caught := 0
	for i := 1; i <= 3; i++ {
		r := rand.Intn(100)
		if r < 35 { caught++; text += fmt.Sprintf("第%d次: 🎯 套中了！\n", i)
		} else if r < 70 { text += fmt.Sprintf("第%d次: ❌ 没套到...\n", i)
		} else { text += fmt.Sprintf("第%d次: 🐮💨 牛跑了！\n", i) }
	}
	text += fmt.Sprintf("\n套中: %d/3\n\n", caught)
	switch caught {
	case 3: return text + "🏆 套牛高手！全中！5倍！", 5
	case 2: return text + "🎉 不错！3倍！", 3
	case 1: return text + "✨ 套到一头！1.5倍！", 1.5
	default: return text + "😢 全部没套到...", 0
	}
}

// 22. 拔萝卜
func playPullCarrot() (string, float64) {
	sizes := []struct{ emoji, name string; multi float64 }{
		{"🥕", "迷你萝卜", 0.5}, {"🥕", "普通萝卜", 1}, {"🥕", "大萝卜", 2}, {"🥕", "巨型萝卜", 4}, {"🌟", "金萝卜", 10},
	}
	weights := []int{25, 35, 20, 12, 3}; totalW := 95
	r := rand.Intn(totalW); cum := 0; idx := 0
	for i, w := range weights { cum += w; if r < cum { idx = i; break } }
	s := sizes[idx]
	text := fmt.Sprintf("🥕 拔萝卜！使劲拔...\n\n嘿哟嘿哟拔萝卜...\n\n拔出来了：%s %s！\n\n", s.emoji, s.name)
	if s.multi >= 4 { text += fmt.Sprintf("🏆 %.0f倍奖励！", s.multi)
	} else if s.multi >= 2 { text += fmt.Sprintf("🎉 %.0f倍！", s.multi)
	} else if s.multi >= 1 { text += "✨ 普通大小，保本！"
	} else { text += "😢 太小了..." }
	return text, s.multi
}

// 23. 采蘑菇
func playMushroom() (string, float64) {
	mushrooms := []struct{ emoji, name string; pts int }{
		{"🍄", "香菇", 2}, {"🍄", "松茸", 4}, {"🍄", "鸡枞菌", 6},
	}
	text := "🍄 进山采蘑菇！小心毒蘑菇！\n\n"
	totalPts := 0; poisoned := false
	for round := 1; round <= 4; round++ {
		r := rand.Intn(100)
		if r < 15 { text += fmt.Sprintf("第%d丛: ☠️ 毒蘑菇！中毒了！\n", round); poisoned = true; break
		} else if r < 20 { text += fmt.Sprintf("第%d丛: 🌿 空的\n", round)
		} else { m := mushrooms[rand.Intn(3)]; totalPts += m.pts; text += fmt.Sprintf("第%d丛: %s %s +%d\n", round, m.emoji, m.name, m.pts) }
	}
	text += "\n"
	if poisoned { return text + "☠️ 中毒了！全部作废...", 0 }
	text += fmt.Sprintf("总分: %d\n\n", totalPts)
	switch {
	case totalPts >= 16: return text + "🏆 采菇大师！5倍！", 5
	case totalPts >= 10: return text + "🎉 满筐了！3倍！", 3
	case totalPts >= 5: return text + "✨ 不少！1.5倍！", 1.5
	default: return text + "👍 聊胜于无", 0.8
	}
}

// 24. 孵蛋
func playHatchEgg() (string, float64) {
	breeds := []struct{ emoji, name, rarity string; multi float64 }{
		{"🐤", "小黄鸡", "普通", 0.5}, {"🐔", "芦花鸡", "良品", 1}, {"🦆", "小鸭子", "优良", 1.5},
		{"🦢", "天鹅", "稀有", 3}, {"🦚", "孔雀", "史诗", 6}, {"🐦", "凤凰", "传说", 12},
	}
	weights := []int{30, 25, 20, 13, 8, 2}; totalW := 98
	r := rand.Intn(totalW); cum := 0; idx := 0
	for i, w := range weights { cum += w; if r < cum { idx = i; break } }
	b := breeds[idx]
	text := "🐣 孵蛋中...\n\n🥚 裂开了... 裂开了...\n\n"
	text += fmt.Sprintf("孵出了: %s %s [%s]\n\n", b.emoji, b.name, b.rarity)
	if b.multi >= 6 { text += fmt.Sprintf("🏆 传说品种！%.0f倍！", b.multi)
	} else if b.multi >= 3 { text += fmt.Sprintf("🎉 稀有品种！%.0f倍！", b.multi)
	} else if b.multi >= 1.5 { text += fmt.Sprintf("✨ 优良！%.1f倍！", b.multi)
	} else if b.multi >= 1 { text += "👍 普通品种，保本"
	} else { text += "😢 太普通了..." }
	return text, b.multi
}

// 25. 天气预报
func playWeather() (string, float64) {
	weathers := []struct{ emoji, name string }{
		{"☀️", "晴天"}, {"🌤", "多云"}, {"🌧", "下雨"}, {"⛈", "雷暴"}, {"🌈", "彩虹"},
	}
	actual := rand.Intn(5)
	guess := rand.Intn(5)
	text := fmt.Sprintf("🌈 预测明天天气！\n\n你猜: %s %s\n实际: %s %s\n\n", weathers[guess].emoji, weathers[guess].name, weathers[actual].emoji, weathers[actual].name)
	if guess == actual {
		if actual == 4 { return text + "🏆 猜中彩虹！5倍！", 5 }
		return text + "🎉 猜对了！3倍！", 3
	}
	diff := guess - actual; if diff < 0 { diff = -diff }
	if diff == 1 { return text + "✨ 接近了！1.5倍！", 1.5 }
	return text + "😢 猜错了...", 0
}

// 26. 农产品评比
func playProduce() (string, float64) {
	categories := []struct{ emoji, name string }{
		{"🍎", "水果外观"}, {"🌾", "谷物品质"}, {"🥕", "蔬菜新鲜度"}, {"🍯", "加工品口感"}, {"🌸", "综合印象"},
	}
	text := "🏆 农产品博览会评比！\n\n"
	totalScore := 0
	for _, c := range categories {
		score := rand.Intn(20) + 1; totalScore += score
		stars := "⭐"; if score >= 17 { stars = "⭐⭐⭐" } else if score >= 12 { stars = "⭐⭐" }
		text += fmt.Sprintf("%s %s: %d分 %s\n", c.emoji, c.name, score, stars)
	}
	text += fmt.Sprintf("\n总分: %d/100\n\n", totalScore)
	switch {
	case totalScore >= 85: return text + "🏆 金奖！5倍！", 5
	case totalScore >= 70: return text + "🥈 银奖！3倍！", 3
	case totalScore >= 55: return text + "🥉 铜奖！1.5倍！", 1.5
	case totalScore >= 40: return text + "👍 参与奖", 0.8
	default: return text + "😢 没获奖...", 0
	}
}

// 27. 驯马
func playTame() (string, float64) {
	text := "🐴 野马出现了！尝试驯服！\n\n"
	stayed := 0
	for round := 1; round <= 6; round++ {
		chance := 70 - round*8
		if rand.Intn(100) < chance {
			stayed = round
			text += fmt.Sprintf("第%d秒: 🐴 还在马背上！\n", round)
		} else {
			text += fmt.Sprintf("第%d秒: 🐴💨 被甩下来了！\n", round)
			break
		}
	}
	if stayed == 6 { return text + "\n🏆 成功驯服野马！8倍！", 8 }
	text += fmt.Sprintf("\n坚持了%d秒\n\n", stayed)
	switch {
	case stayed >= 5: return text + "🎉 差一点！3倍！", 3
	case stayed >= 3: return text + "✨ 还行！1.5倍！", 1.5
	case stayed >= 1: return text + "👍 努力了", 0.8
	default: return text + "😢 马上就摔了...", 0
	}
}

// 28. 扎稻草人
func playScarecrow() (string, float64) {
	parts := []struct{ emoji, name string }{
		{"👒", "帽子"}, {"👔", "衣服"}, {"🧤", "手套"}, {"👢", "靴子"},
	}
	text := "👒 扎稻草人赶乌鸦！\n\n"
	totalScore := 0
	for _, p := range parts {
		score := rand.Intn(30) + 1; totalScore += score
		quality := "普通"; if score >= 25 { quality = "完美" } else if score >= 15 { quality = "良好" }
		text += fmt.Sprintf("%s %s: %d分 (%s)\n", p.emoji, p.name, score, quality)
	}
	crows := rand.Intn(10) + 1
	scared := crows * totalScore / 120
	if scared > crows { scared = crows }
	text += fmt.Sprintf("\n稻草人质量: %d/120\n🐦 乌鸦%d只，吓跑%d只\n\n", totalScore, crows, scared)
	switch {
	case totalScore >= 100: return text + "🏆 完美稻草人！5倍！", 5
	case totalScore >= 80: return text + "🎉 不错！3倍！", 3
	case totalScore >= 60: return text + "✨ 还行！1.5倍！", 1.5
	case totalScore >= 40: return text + "👍 凑合用", 0.8
	default: return text + "😢 乌鸦不怕...", 0
	}
}

// 29. 赶狐狸
func playFoxHunt() (string, float64) {
	text := "🦊 狐狸来偷鸡了！保护鸡舍！\n\n"
	chickens := 6; saved := 0
	for i := 1; i <= chickens; i++ {
		r := rand.Intn(100)
		if r < 55 { saved++; text += fmt.Sprintf("🐔%d: ✅ 保住了！\n", i)
		} else if r < 80 { text += fmt.Sprintf("🐔%d: 🦊 被叼走了！\n", i)
		} else { text += fmt.Sprintf("🐔%d: 🦊💨 狐狸太快了！\n", i) }
	}
	text += fmt.Sprintf("\n保住: %d/%d只鸡\n\n", saved, chickens)
	switch {
	case saved >= 6: return text + "🏆 全部保住！5倍！", 5
	case saved >= 4: return text + "🎉 不错！3倍！", 3
	case saved >= 3: return text + "✨ 还行！1.5倍！", 1.5
	case saved >= 1: return text + "👍 至少保住几只", 0.8
	default: return text + "😢 鸡都被叼走了...", 0
	}
}

// 30. 抢收比赛
func playHarvestRace() (string, float64) {
	text := "👨‍🌾 暴风雨要来了！抢收庄稼！\n\n"
	plots := 8; harvested := 0
	for i := 1; i <= plots; i++ {
		r := rand.Intn(100)
		if r < 55 { harvested++; text += fmt.Sprintf("田%d: 🌾✅收了 ", i)
		} else if r < 80 { text += fmt.Sprintf("田%d: 🌧️淋了 ", i)
		} else { text += fmt.Sprintf("田%d: ⚡毁了 ", i) }
		if i%2 == 0 { text += "\n" }
	}
	text += fmt.Sprintf("\n\n抢收: %d/%d块田\n\n", harvested, plots)
	switch {
	case harvested >= 7: return text + "🏆 抢收大师！5倍！", 5
	case harvested >= 5: return text + "🎉 不错！3倍！", 3
	case harvested >= 3: return text + "✨ 还行！1.5倍！", 1.5
	case harvested >= 1: return text + "👍 保住一点", 0.8
	default: return text + "😢 全毁了...", 0
	}
}
