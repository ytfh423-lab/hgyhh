package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 土壤肥力系统（A-1）控制器
// 所有接口都基于 model.ApplySoilPatch 写回，不直接写 SQL。
// 四个接口：
//   GET  /api/farm/soil/view                土壤详情 + 肥料价目
//   POST /api/farm/soil/fertilize           施用指定肥料
//   POST /api/farm/soil/fallow              将地块标为休耕 N 小时
//   POST /api/farm/soil/fallow/cancel       提前结束休耕

// fertilizerDef 肥料定义，键为前端 / 后端统一的肥料 code
type fertilizerDef struct {
	Code   string
	Name   string
	Price  int
	Emoji  string
	Effect string
	Patch  model.SoilPatch
}

// 所有肥料的效果表。前端直接吃后端下发的 definitions，保证两端一致。
func allFertilizers() []fertilizerDef {
	return []fertilizerDef{
		{
			Code: "npk", Name: "复合肥", Emoji: "🧪",
			Price:  common.TgBotFarmFertilizerNPKPrice,
			Effect: "N/P/K 各 +15",
			Patch:  model.SoilPatch{DN: 15, DP: 15, DK: 15},
		},
		{
			Code: "urea", Name: "尿素", Emoji: "🟡",
			Price:  common.TgBotFarmFertilizerUreaPrice,
			Effect: "氮 +25，微酸化",
			Patch:  model.SoilPatch{DN: 25, DPH: -1},
		},
		{
			Code: "bone_meal", Name: "骨粉", Emoji: "🦴",
			Price:  common.TgBotFarmFertilizerBoneMealPrice,
			Effect: "磷 +25，OM +3",
			Patch:  model.SoilPatch{DP: 25, DOM: 3},
		},
		{
			Code: "ash", Name: "草木灰", Emoji: "🌫️",
			Price:  common.TgBotFarmFertilizerAshPrice,
			Effect: "钾 +20，PH +2",
			Patch:  model.SoilPatch{DK: 20, DPH: 2},
		},
		{
			Code: "compost", Name: "堆肥", Emoji: "🟤",
			Price:  common.TgBotFarmFertilizerCompostPrice,
			Effect: "OM +20，疲劳 -10，全营养 +5",
			Patch:  model.SoilPatch{DN: 5, DP: 5, DK: 5, DOM: 20, DFatigue: -10},
		},
		{
			Code: "lime", Name: "石灰", Emoji: "⚪",
			Price:  common.TgBotFarmFertilizerLimePrice,
			Effect: "PH +5（酸性土校正）",
			Patch:  model.SoilPatch{DPH: 5},
		},
		{
			Code: "sulfur", Name: "硫磺", Emoji: "🟨",
			Price:  common.TgBotFarmFertilizerSulfurPrice,
			Effect: "PH -5（碱性土校正）",
			Patch:  model.SoilPatch{DPH: -5},
		},
	}
}

// findFertilizer 根据 code 取肥料定义
func findFertilizer(code string) *fertilizerDef {
	for i := range allFertilizers() {
		f := allFertilizers()[i]
		if f.Code == code {
			return &f
		}
	}
	return nil
}

// cropSoilPreset 根据作物 key 猜测它属于哪类需肥偏好
// 未来可以直接把该映射迁移到 farmCropDef，一次全量替换即可。
func cropSoilPreset(cropKey string) string {
	rootKeys := []string{"potato", "radish", "carrot", "sweet_potato", "ginger", "turnip", "yam"}
	fruitKeys := []string{"tomato", "strawberry", "watermelon", "grape", "apple", "corn", "pumpkin", "melon", "pepper", "eggplant"}
	for _, k := range rootKeys {
		if cropKey == k {
			return "root"
		}
	}
	for _, k := range fruitKeys {
		if cropKey == k {
			return "fruit"
		}
	}
	// 其他（青菜、生菜、菠菜、白菜…）归为 leafy
	leafyKeys := []string{"cabbage", "lettuce", "spinach", "bokchoy", "kale"}
	for _, k := range leafyKeys {
		if cropKey == k {
			return "leafy"
		}
	}
	return ""
}

// soilPlotInfo 构造土壤详情返回体
func soilPlotInfo(plot *model.TgFarmPlot) map[string]interface{} {
	now := time.Now().Unix()
	fallowRemain := int64(0)
	if plot.FallowUntil > now {
		fallowRemain = plot.FallowUntil - now
	}
	return map[string]interface{}{
		"plot_index":     plot.PlotIndex,
		"crop_type":      plot.CropType,
		"last_crop_type": plot.LastCropType,
		"status":         plot.Status,
		"soil_level":     plot.SoilLevel,
		"soil": map[string]interface{}{
			"n":       plot.SoilN,
			"p":       plot.SoilP,
			"k":       plot.SoilK,
			"ph":      float64(plot.SoilPH) / 10.0,
			"om":      plot.SoilOM,
			"fatigue": plot.SoilFatigue,
			"score":   model.SoilScore(plot),
			"yield":   model.SoilYieldFactor(plot),
		},
		"fallow_until":  plot.FallowUntil,
		"fallow_remain": fallowRemain,
	}
}

// WebFarmSoilView 返回所有地块土壤详情 + 肥料价目
func WebFarmSoilView(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	plotInfos := make([]map[string]interface{}, 0, len(plots))
	for _, p := range plots {
		plotInfos = append(plotInfos, soilPlotInfo(p))
	}
	defs := allFertilizers()
	fertDefs := make([]map[string]interface{}, 0, len(defs))
	for _, f := range defs {
		fertDefs = append(fertDefs, map[string]interface{}{
			"code":   f.Code,
			"name":   f.Name,
			"emoji":  f.Emoji,
			"price":  f.Price,
			"effect": f.Effect,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"plots":       plotInfos,
			"fertilizers": fertDefs,
			"fallow": gin.H{
				"min_hours": common.TgBotFarmFallowMinHours,
				"max_hours": common.TgBotFarmFallowMaxHours,
			},
		},
	})
}

// WebFarmSoilFertilize 施用一种肥料到指定地块
func WebFarmSoilFertilize(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		PlotIndex int    `json:"plot_index"`
		Code      string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	fer := findFertilizer(req.Code)
	if fer == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知肥料类型"})
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	var target *model.TgFarmPlot
	for _, p := range plots {
		if p.PlotIndex == req.PlotIndex {
			target = p
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "地块不存在"})
		return
	}
	if user.Quota < fer.Price {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("余额不足！%s价格 $%.2f", fer.Name, webFarmQuotaFloat(fer.Price)),
		})
		return
	}
	if err := model.DecreaseUserQuota(user.Id, fer.Price); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "扣费失败"})
		return
	}
	if err := model.ApplySoilPatch(target, fer.Patch); err != nil {
		_ = model.IncreaseUserQuota(user.Id, fer.Price, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "土壤更新失败，已退款"})
		return
	}
	model.AddFarmLog(tgId, "soil_fertilize", -fer.Price, fmt.Sprintf("%d号地使用%s%s", req.PlotIndex+1, fer.Emoji, fer.Name))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("%d号地使用%s成功！%s", req.PlotIndex+1, fer.Name, fer.Effect),
		"data":    soilPlotInfo(target),
	})
}

// WebFarmSoilFallow 将地块标为休耕
func WebFarmSoilFallow(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		PlotIndex int `json:"plot_index"`
		Hours     int `json:"hours"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.Hours < common.TgBotFarmFallowMinHours || req.Hours > common.TgBotFarmFallowMaxHours {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("休耕时长需在 %d-%d 小时之间",
				common.TgBotFarmFallowMinHours, common.TgBotFarmFallowMaxHours),
		})
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	var target *model.TgFarmPlot
	for _, p := range plots {
		if p.PlotIndex == req.PlotIndex {
			target = p
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "地块不存在"})
		return
	}
	if target.Status != 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "地块有作物，无法休耕"})
		return
	}
	until := time.Now().Unix() + int64(req.Hours)*3600
	if err := model.SetFallowUntil(target.Id, until); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "休耕失败"})
		return
	}
	target.FallowUntil = until
	model.AddFarmLog(tgId, "soil_fallow", 0, fmt.Sprintf("%d号地开始休耕 %d 小时", req.PlotIndex+1, req.Hours))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("%d号地开始休耕 %d 小时", req.PlotIndex+1, req.Hours),
		"data":    soilPlotInfo(target),
	})
}

// WebFarmSoilFallowCancel 提前结束休耕
func WebFarmSoilFallowCancel(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		PlotIndex int `json:"plot_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	plots, err := model.GetOrCreateFarmPlots(tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	var target *model.TgFarmPlot
	for _, p := range plots {
		if p.PlotIndex == req.PlotIndex {
			target = p
			break
		}
	}
	if target == nil || target.FallowUntil == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "地块未在休耕"})
		return
	}
	if err := model.SetFallowUntil(target.Id, 0); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "操作失败"})
		return
	}
	target.FallowUntil = 0
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("%d号地已结束休耕", req.PlotIndex+1),
		"data":    soilPlotInfo(target),
	})
}
