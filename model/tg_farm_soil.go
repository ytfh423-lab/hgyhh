package model

// 土壤肥力系统（A-1）— 所有土壤相关的数值变更入口，集中在此文件。
// 设计原则：
//   1. 所有对 soil_* 字段的修改必须走这里的函数，不允许在 controller 里直接写 SQL。
//   2. 字段范围约束在写入前 clamp，避免脏数据。
//   3. 读取使用已有的 GetOrCreateFarmPlots / DB.First，不重复封装。

// SoilParamRange 返回每个土壤字段允许的区间
func SoilParamRange() (nMin, nMax, phMin, phMax int) {
	return 0, 100, 45, 85
}

// clampInt 夹紧整数到 [min, max]
func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// SoilPatch 一次土壤修改的增量参数，负值代表扣减，0 表示不变
type SoilPatch struct {
	DN        int // 氮增量
	DP        int // 磷增量
	DK        int // 钾增量
	DPH       int // PH x10 增量，+5 即 PH +0.5
	DOM       int // 有机质增量
	DFatigue  int // 连作疲劳增量
}

// ApplySoilPatch 将增量落库；需要传入完整 plot 以便在内存里先 clamp
func ApplySoilPatch(plot *TgFarmPlot, patch SoilPatch) error {
	if plot == nil {
		return nil
	}
	plot.SoilN = clampInt(plot.SoilN+patch.DN, 0, 100)
	plot.SoilP = clampInt(plot.SoilP+patch.DP, 0, 100)
	plot.SoilK = clampInt(plot.SoilK+patch.DK, 0, 100)
	plot.SoilPH = clampInt(plot.SoilPH+patch.DPH, 45, 85)
	plot.SoilOM = clampInt(plot.SoilOM+patch.DOM, 0, 100)
	plot.SoilFatigue = clampInt(plot.SoilFatigue+patch.DFatigue, 0, 100)
	return DB.Model(&TgFarmPlot{}).Where("id = ?", plot.Id).Updates(map[string]interface{}{
		"soil_n":      plot.SoilN,
		"soil_p":      plot.SoilP,
		"soil_k":      plot.SoilK,
		"soil_ph":     plot.SoilPH,
		"soil_om":     plot.SoilOM,
		"soil_fatigue": plot.SoilFatigue,
	}).Error
}

// SetFallowUntil 标记地块休耕到某个时间戳
func SetFallowUntil(plotId int, until int64) error {
	return DB.Model(&TgFarmPlot{}).Where("id = ?", plotId).Update("fallow_until", until).Error
}

// SetLastCropType 记录上一次种植作物（用于连作疲劳判定）
func SetLastCropType(plotId int, cropType string) error {
	return DB.Model(&TgFarmPlot{}).Where("id = ?", plotId).Update("last_crop_type", cropType).Error
}

// SoilScore 给土壤打个综合评分（0-100），用于前端展示与"沃土加成"判定
// 评分 = 平均NPK * 0.5 + OM * 0.25 + PH偏中性程度 * 0.15 + (100-疲劳) * 0.1
func SoilScore(plot *TgFarmPlot) int {
	if plot == nil {
		return 0
	}
	npk := float64(plot.SoilN+plot.SoilP+plot.SoilK) / 3.0
	om := float64(plot.SoilOM)
	// PH 偏离 65（中性）越远得分越低
	phDiff := plot.SoilPH - 65
	if phDiff < 0 {
		phDiff = -phDiff
	}
	phScore := 100.0 - float64(phDiff)*5.0
	if phScore < 0 {
		phScore = 0
	}
	fatigue := 100.0 - float64(plot.SoilFatigue)
	score := npk*0.5 + om*0.25 + phScore*0.15 + fatigue*0.1
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return int(score)
}

// SoilYieldFactor 返回土壤对产量的乘数（0.5 ~ 1.3）
// 用于种植/收获时对产量进行微调；独立于现有 SoilLevel 加速。
func SoilYieldFactor(plot *TgFarmPlot) float64 {
	if plot == nil {
		return 1.0
	}
	score := SoilScore(plot)
	// 0 -> 0.5；50 -> 1.0；100 -> 1.3
	switch {
	case score >= 85:
		return 1.30
	case score >= 70:
		return 1.15
	case score >= 50:
		return 1.00
	case score >= 30:
		return 0.85
	default:
		return 0.70
	}
}

// SoilConsumeOnPlant 种植时按作物消耗 N/P/K 并累加疲劳
// consumePreset: "leafy"(叶菜偏氮) / "fruit"(果实偏磷钾) / "root"(根茎偏钾) / ""(通用)
func SoilConsumeOnPlant(plot *TgFarmPlot, cropType string, consumePreset string) error {
	if plot == nil {
		return nil
	}
	patch := SoilPatch{}
	switch consumePreset {
	case "leafy":
		patch.DN, patch.DP, patch.DK = -8, -3, -3
	case "fruit":
		patch.DN, patch.DP, patch.DK = -4, -8, -6
	case "root":
		patch.DN, patch.DP, patch.DK = -3, -4, -9
	default:
		patch.DN, patch.DP, patch.DK = -5, -5, -5
	}
	// 连作判定：同一作物种两次，疲劳累加更多
	if plot.LastCropType != "" && plot.LastCropType == cropType {
		patch.DFatigue = 12
	} else {
		patch.DFatigue = 4
	}
	if err := ApplySoilPatch(plot, patch); err != nil {
		return err
	}
	return SetLastCropType(plot.Id, cropType)
}

// SoilRecoverOnHarvest 收获时少量回补（残余根系腐解回补有机质）
func SoilRecoverOnHarvest(plot *TgFarmPlot) error {
	return ApplySoilPatch(plot, SoilPatch{DOM: 2})
}

// SoilFallowTick 休耕期间每小时被调用一次，缓慢恢复地力
func SoilFallowTick(plot *TgFarmPlot) error {
	return ApplySoilPatch(plot, SoilPatch{
		DN: 2, DP: 2, DK: 2,
		DOM: 1, DFatigue: -3,
	})
}
