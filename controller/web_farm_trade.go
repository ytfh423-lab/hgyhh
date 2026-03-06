package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func WebFarmTradeList(c *gin.Context) {
	_, _, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	page := 1
	if p, exists := c.GetQuery("page"); exists {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	pageSize := 20
	offset := (page - 1) * pageSize

	trades, total, err := model.GetOpenTrades(pageSize, offset)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "查询失败"})
		return
	}

	type tradeItem struct {
		Id           int     `json:"id"`
		SellerName   string  `json:"seller_name"`
		Category     string  `json:"category"`
		ItemName     string  `json:"item_name"`
		ItemEmoji    string  `json:"item_emoji"`
		Quantity     int     `json:"quantity"`
		PricePerUnit float64 `json:"price_per_unit"`
		TotalPrice   float64 `json:"total_price"`
		Fee          float64 `json:"fee"`
		CreatedAt    int64   `json:"created_at"`
	}

	feeRate := float64(common.TgBotFarmTradeFee) / 100.0
	var items []tradeItem
	for _, t := range trades {
		unitPrice := webFarmQuotaFloat(t.PricePerUnit)
		tp := unitPrice * float64(t.Quantity)
		items = append(items, tradeItem{
			Id: t.Id, SellerName: t.SellerName, Category: t.Category,
			ItemName: t.ItemName, ItemEmoji: t.ItemEmoji, Quantity: t.Quantity,
			PricePerUnit: unitPrice, TotalPrice: tp, Fee: tp * feeRate, CreatedAt: t.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"trades": items, "total": total, "page": page, "fee_rate": common.TgBotFarmTradeFee}})
}

func WebFarmTradeCreate(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		CropType string  `json:"crop_type"`
		Quantity int     `json:"quantity"`
		Price    float64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Quantity <= 0 || req.Price <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if int(model.CountMyOpenTrades(tgId)) >= common.TgBotFarmTradeMaxListings {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "挂单数量已达上限"})
		return
	}
	err := model.RemoveFromWarehouse(tgId, req.CropType, req.Quantity)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仓库物品不足"})
		return
	}

	itemName, itemEmoji, category := getTradeItemInfo(req.CropType)
	priceQuota := int(req.Price * 500000)

	trade := &model.TgFarmTrade{
		SellerId: tgId, SellerName: user.Username, Category: category,
		ItemKey: req.CropType, ItemName: itemName, ItemEmoji: itemEmoji,
		Quantity: req.Quantity, PricePerUnit: priceQuota, Status: 0,
	}
	if err := model.CreateTrade(trade); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建失败"})
		return
	}
	model.AddFarmLog(tgId, "trade", 0, fmt.Sprintf("📤 挂单: %s%s×%d", itemEmoji, itemName, req.Quantity))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "挂单成功！"})
}

func WebFarmTradeBuy(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		TradeId int `json:"trade_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	trade, err := model.GetTradeById(req.TradeId)
	if err != nil || trade.Status != 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "交易不存在或已完成"})
		return
	}
	if trade.SellerId == tgId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "不能买自己的商品"})
		return
	}

	totalPrice := trade.PricePerUnit * trade.Quantity
	fee := totalPrice * common.TgBotFarmTradeFee / 100
	totalCost := totalPrice + fee
	if user.Quota < totalCost {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "余额不足"})
		return
	}

	model.DecreaseUserQuota(user.Id, totalCost)

	sellerUser := &model.User{TelegramId: trade.SellerId}
	_ = sellerUser.FillUserByTelegramId()
	if sellerUser.Id > 0 {
		model.IncreaseUserQuota(sellerUser.Id, totalPrice)
		model.AddFarmLog(trade.SellerId, "trade", totalPrice, fmt.Sprintf("💰 售出: %s%s×%d", trade.ItemEmoji, trade.ItemName, trade.Quantity))
	}

	_, category := getTradeCategory(trade.ItemKey)
	_ = model.AddToWarehouseWithCategory(tgId, trade.ItemKey, trade.Quantity, category)
	_ = model.UpdateTradeStatus(trade.Id, 1, tgId)
	model.AddFarmLog(tgId, "trade", -totalCost, fmt.Sprintf("📥 购入: %s%s×%d", trade.ItemEmoji, trade.ItemName, trade.Quantity))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "购买成功！"})
}

func WebFarmTradeCancel(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		TradeId int `json:"trade_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	trade, err := model.GetTradeById(req.TradeId)
	if err != nil || trade.Status != 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "交易不存在或已完成"})
		return
	}
	if trade.SellerId != tgId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "只能取消自己的挂单"})
		return
	}

	_, category := getTradeCategory(trade.ItemKey)
	_ = model.AddToWarehouseWithCategory(tgId, trade.ItemKey, trade.Quantity, category)
	_ = model.UpdateTradeStatus(trade.Id, 2, "")
	model.AddFarmLog(tgId, "trade", 0, "❌ 取消挂单: "+trade.ItemEmoji+trade.ItemName)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已取消，物品已退回仓库"})
}

func WebFarmTradeHistory(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	trades, _ := model.GetTradeHistory(tgId, 20)
	type histItem struct {
		Id        int     `json:"id"`
		ItemName  string  `json:"item_name"`
		ItemEmoji string  `json:"item_emoji"`
		Quantity  int     `json:"quantity"`
		Price     float64 `json:"price"`
		Status    int     `json:"status"`
		IsSeller  bool    `json:"is_seller"`
		CreatedAt int64   `json:"created_at"`
	}
	var items []histItem
	for _, t := range trades {
		items = append(items, histItem{
			Id: t.Id, ItemName: t.ItemName, ItemEmoji: t.ItemEmoji,
			Quantity: t.Quantity, Price: webFarmQuotaFloat(t.PricePerUnit),
			Status: t.Status, IsSeller: t.SellerId == tgId, CreatedAt: t.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func getTradeItemInfo(itemKey string) (name, emoji, category string) {
	for _, cr := range farmCrops {
		if cr.Key == itemKey || "crop_"+cr.Key == itemKey {
			return cr.Name, cr.Emoji, "crop"
		}
	}
	for _, f := range fishTypes {
		if f.Key == itemKey || "fish_"+f.Key == itemKey {
			return f.Name, f.Emoji, "fish"
		}
	}
	for _, a := range ranchAnimals {
		if a.Key == itemKey || "meat_"+a.Key == itemKey {
			return a.Name, a.Emoji, "meat"
		}
	}
	for _, r := range recipes {
		if r.Key == itemKey || "recipe_"+r.Key == itemKey {
			return r.Name, r.Emoji, "recipe"
		}
	}
	return itemKey, "📦", "crop"
}

func getTradeCategory(itemKey string) (string, string) {
	for _, cr := range farmCrops {
		if cr.Key == itemKey || "crop_"+cr.Key == itemKey {
			return cr.Key, "crop"
		}
	}
	for _, f := range fishTypes {
		if f.Key == itemKey || "fish_"+f.Key == itemKey {
			return f.Key, "fish"
		}
	}
	for _, a := range ranchAnimals {
		if a.Key == itemKey || "meat_"+a.Key == itemKey {
			return a.Key, "meat"
		}
	}
	for _, r := range recipes {
		if r.Key == itemKey || "recipe_"+r.Key == itemKey {
			return r.Key, "recipe"
		}
	}
	return itemKey, "crop"
}
