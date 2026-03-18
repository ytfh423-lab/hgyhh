package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ========== 管理员市场API ==========

// WebMarketAdminGetEvents 获取所有市场事件
func WebMarketAdminGetEvents(c *gin.Context) {
	events, err := model.GetAllMarketEvents()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取事件失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": events})
}

// WebMarketAdminCreateEvent 创建市场事件
func WebMarketAdminCreateEvent(c *gin.Context) {
	var event model.TgMarketEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if event.Name == "" || event.EndTime <= event.StartTime {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "事件名称和时间范围不能为空"})
		return
	}
	if err := model.CreateMarketEvent(&event); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "事件已创建"})
}

// WebMarketAdminUpdateEvent 更新市场事件
func WebMarketAdminUpdateEvent(c *gin.Context) {
	var event model.TgMarketEvent
	if err := c.ShouldBindJSON(&event); err != nil || event.Id == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.UpdateMarketEvent(&event); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "事件已更新"})
}

// WebMarketAdminDeleteEvent 删除市场事件
func WebMarketAdminDeleteEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效ID"})
		return
	}
	if err := model.DeleteMarketEvent(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "事件已删除"})
}

// WebMarketAdminGetConfig 获取市场引擎全局配置和商品参数
func WebMarketAdminGetConfig(c *gin.Context) {
	configs := getAllMarketConfigs()

	type itemCfgView struct {
		Key               string `json:"key"`
		Name              string `json:"name"`
		Emoji             string `json:"emoji"`
		Category          string `json:"category"`
		BasePrice         int    `json:"base_price"`
		MinMultiplier     int    `json:"min_multiplier"`
		MaxMultiplier     int    `json:"max_multiplier"`
		Volatility        int    `json:"volatility"`
		TrendStrength     int    `json:"trend_strength"`
		MeanRevStrength   int    `json:"mean_rev_strength"`
		SupplySensitivity int    `json:"supply_sensitivity"`
		SeasonProfile     [4]int `json:"season_profile"`
		CurMultiplier     int    `json:"cur_multiplier"`
	}

	var items []itemCfgView
	for _, cfg := range configs {
		state := getMarketItemState(cfg.Key)
		curMult := 100
		if state != nil {
			curMult = state.Multiplier
		}
		items = append(items, itemCfgView{
			Key:               cfg.Key,
			Name:              cfg.Name,
			Emoji:             cfg.Emoji,
			Category:          cfg.Category,
			BasePrice:         cfg.BasePrice,
			MinMultiplier:     cfg.MinMultiplier,
			MaxMultiplier:     cfg.MaxMultiplier,
			Volatility:        cfg.Volatility,
			TrendStrength:     cfg.TrendStrength,
			MeanRevStrength:   cfg.MeanRevStrength,
			SupplySensitivity: cfg.SupplySensitivity,
			SeasonProfile:     cfg.SeasonProfile,
			CurMultiplier:     curMult,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":          items,
			"refresh_hours":  common.TgBotMarketRefreshHours,
			"next_refresh":   getMarketNextRefresh(),
		},
	})
}

// WebMarketAdminUpdateItemConfig 更新单个商品市场参数
func WebMarketAdminUpdateItemConfig(c *gin.Context) {
	var req struct {
		Key               string `json:"key"`
		MinMultiplier     *int   `json:"min_multiplier"`
		MaxMultiplier     *int   `json:"max_multiplier"`
		Volatility        *int   `json:"volatility"`
		TrendStrength     *int   `json:"trend_strength"`
		MeanRevStrength   *int   `json:"mean_rev_strength"`
		SupplySensitivity *int   `json:"supply_sensitivity"`
		SeasonProfile     *[4]int `json:"season_profile"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	mktMu.Lock()
	cfg, ok := mktConfigs[req.Key]
	if !ok {
		mktMu.Unlock()
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "商品不存在"})
		return
	}
	if req.MinMultiplier != nil {
		cfg.MinMultiplier = *req.MinMultiplier
	}
	if req.MaxMultiplier != nil {
		cfg.MaxMultiplier = *req.MaxMultiplier
	}
	if req.Volatility != nil {
		cfg.Volatility = *req.Volatility
	}
	if req.TrendStrength != nil {
		cfg.TrendStrength = *req.TrendStrength
	}
	if req.MeanRevStrength != nil {
		cfg.MeanRevStrength = *req.MeanRevStrength
	}
	if req.SupplySensitivity != nil {
		cfg.SupplySensitivity = *req.SupplySensitivity
	}
	if req.SeasonProfile != nil {
		cfg.SeasonProfile = *req.SeasonProfile
	}
	mktMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "参数已更新"})
}

// WebMarketAdminForceRefresh 强制刷新市场价格
func WebMarketAdminForceRefresh(c *gin.Context) {
	doMarketTick()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "市场已强制刷新"})
}

// ========== 玩家市场详情API ==========

// WebMarketItemDetail 获取单个商品详情（含影响因素）
func WebMarketItemDetail(c *gin.Context) {
	itemKey := c.Query("key")
	if itemKey == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请指定商品"})
		return
	}

	ensureMarketEngine()

	mktMu.RLock()
	cfg, cfgOk := mktConfigs[itemKey]
	state, stateOk := mktStates[itemKey]
	mktMu.RUnlock()

	if !cfgOk {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "商品不存在"})
		return
	}

	// 获取价格历史
	history, _ := model.GetMarketPriceHistory(itemKey, 48)

	// 构建历史数据点
	type historyPoint struct {
		Timestamp int64 `json:"timestamp"`
		Mult      int   `json:"multiplier"`
	}
	var points []historyPoint
	for i := len(history) - 1; i >= 0; i-- {
		points = append(points, historyPoint{
			Timestamp: history[i].Timestamp,
			Mult:      history[i].Multiplier,
		})
	}

	// 构建影响因素说明
	type factorInfo struct {
		Name  string `json:"name"`
		Icon  string `json:"icon"`
		Value int    `json:"value"`
		Desc  string `json:"desc"`
	}
	var factors []factorInfo

	if stateOk {
		season := getCurrentSeason()
		seasonTarget := cfg.SeasonProfile[season]
		seasonDesc := "无明显影响"
		if seasonTarget <= 85 {
			seasonDesc = "当季丰收，供应充足"
		} else if seasonTarget >= 115 {
			seasonDesc = "反季稀缺，价格上涨"
		} else if seasonTarget <= 95 {
			seasonDesc = "当季产量偏高"
		} else if seasonTarget >= 105 {
			seasonDesc = "非当季，需求偏强"
		}
		factors = append(factors, factorInfo{"季节", seasonEmojis[season], state.LastSeasonF, seasonDesc})

		supplyDesc := "供需平衡"
		if state.LastSupplyF > 2 {
			supplyDesc = "需求旺盛，供不应求"
		} else if state.LastSupplyF < -2 {
			supplyDesc = "供应充足，供过于求"
		}
		factors = append(factors, factorInfo{"供需", "⚖️", state.LastSupplyF, supplyDesc})

		trendDesc := "走势平稳"
		if state.LastTrendF > 2 {
			trendDesc = "上涨趋势延续"
		} else if state.LastTrendF < -2 {
			trendDesc = "下跌趋势延续"
		}
		factors = append(factors, factorInfo{"趋势", "📊", state.LastTrendF, trendDesc})

		if state.LastEventF != 0 {
			eventDesc := "受市场事件影响"
			if state.LastEventF > 0 {
				eventDesc = "事件推动价格上涨"
			} else {
				eventDesc = "事件导致价格下跌"
			}
			factors = append(factors, factorInfo{"事件", "📰", state.LastEventF, eventDesc})
		}

		revDesc := "价格在合理区间"
		if state.LastMeanRevF > 2 {
			revDesc = "价格偏低，有回升压力"
		} else if state.LastMeanRevF < -2 {
			revDesc = "价格偏高，有回调压力"
		}
		factors = append(factors, factorInfo{"回归", "🎯", state.LastMeanRevF, revDesc})
	}

	tag, arrow, clr := getMarketPriceTrend(itemKey)

	mult := 100
	prevMult := 100
	trend := 0
	if stateOk {
		mult = state.Multiplier
		prevMult = state.PrevMultiplier
		trend = state.Trend
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"key":            itemKey,
			"name":           cfg.Name,
			"emoji":          cfg.Emoji,
			"category":       cfg.Category,
			"base_price":     webFarmQuotaFloat(cfg.BasePrice),
			"cur_price":      webFarmQuotaFloat(cfg.BasePrice * mult / 100),
			"multiplier":     mult,
			"prev_multiplier": prevMult,
			"change":         mult - prevMult,
			"trend":          trend,
			"trend_tag":      tag,
			"trend_arrow":    arrow,
			"trend_color":    clr,
			"min_multiplier": cfg.MinMultiplier,
			"max_multiplier": cfg.MaxMultiplier,
			"season_profile": cfg.SeasonProfile,
			"history":        points,
			"factors":        factors,
		},
	})
}
