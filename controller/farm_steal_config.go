package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// AdminGetStealConfig 获取偷菜配置
func AdminGetStealConfig(c *gin.Context) {
	cfg := model.GetStealConfig()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cfg})
}

// AdminUpdateStealConfig 更新偷菜配置
func AdminUpdateStealConfig(c *gin.Context) {
	var req model.FarmStealConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数格式错误: " + err.Error()})
		return
	}

	// 校验
	if err := validateStealConfig(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	// 记录变更日志
	oldCfg := model.GetStealConfig()
	changedFields := diffStealConfig(oldCfg, &req)

	userId := c.GetInt("id")
	req.UpdatedBy = userId

	if err := model.UpdateStealConfig(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "保存失败: " + err.Error()})
		return
	}

	if changedFields != "" {
		_ = model.CreateStealConfigLog(&model.FarmStealConfigLog{
			OperatorId:    userId,
			ChangedFields: changedFields,
		})
	}

	common.SysLog(fmt.Sprintf("Admin %d updated steal config: %s", userId, changedFields))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "保存成功"})
}

// AdminGetStealConfigLogs 获取偷菜配置修改日志
func AdminGetStealConfigLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	logs, total, err := model.GetStealConfigLogs(page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": logs, "total": total})
}

// AdminResetStealConfig 恢复默认配置
func AdminResetStealConfig(c *gin.Context) {
	userId := c.GetInt("id")
	def := model.DefaultStealConfig()
	def.UpdatedBy = userId
	if err := model.UpdateStealConfig(def); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "重置失败"})
		return
	}
	_ = model.CreateStealConfigLog(&model.FarmStealConfigLog{
		OperatorId:    userId,
		ChangedFields: `{"action":"reset_to_default"}`,
	})
	common.SysLog(fmt.Sprintf("Admin %d reset steal config to default", userId))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已恢复默认配置"})
}

// WebFarmStealRules 玩家端获取偷菜规则摘要
func WebFarmStealRules(c *gin.Context) {
	cfg := model.GetStealConfig()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"steal_enabled":            cfg.StealEnabled,
			"owner_keep_pct":           int(cfg.OwnerBaseKeepRatio * 100),
			"stealable_pct":            int(cfg.StealableRatio * 100),
			"protection_minutes":       cfg.OwnerProtectionMinutes,
			"max_steal_per_plot":       cfg.MaxStealPerPlot,
			"max_steal_per_day":        cfg.MaxStealPerUserPerDay,
			"max_farm_stolen_per_day":  cfg.MaxStealPerFarmPerDay,
			"cooldown_minutes":         cfg.StealCooldownSeconds / 60,
			"long_crop_hours":          cfg.LongCropHoursThreshold,
			"long_crop_keep_pct":       int(cfg.LongCropOwnerKeepRatio * 100),
			"super_long_crop_hours":    cfg.SuperLongCropHoursThreshold,
			"super_long_bonus_only":    cfg.SuperLongCropBonusOnly,
			"scarecrow_block_pct":      cfg.ScarecrowBlockRate,
			"dog_guard_pct":            cfg.DogGuardRate,
		},
	})
}

// ========== 校验 ==========

func validateStealConfig(cfg *model.FarmStealConfig) error {
	if cfg.OwnerBaseKeepRatio < 0.5 || cfg.OwnerBaseKeepRatio > 1.0 {
		return fmt.Errorf("主人保底比例必须在 50%%~100%% 之间")
	}
	if cfg.StealableRatio < 0 || cfg.StealableRatio > 0.5 {
		return fmt.Errorf("可偷比例必须在 0%%~50%% 之间")
	}
	if cfg.OwnerBaseKeepRatio+cfg.StealableRatio > 1.0 {
		return fmt.Errorf("保底比例 + 可偷比例不能超过 100%%")
	}
	if cfg.OwnerProtectionMinutes < 0 || cfg.OwnerProtectionMinutes > 1440 {
		return fmt.Errorf("保护期必须在 0~1440 分钟之间")
	}
	if cfg.MaxStealPerPlot < 1 || cfg.MaxStealPerPlot > 10 {
		return fmt.Errorf("每块地最大偷取次数必须在 1~10 之间")
	}
	if cfg.MaxStealPerUserPerDay < 1 || cfg.MaxStealPerUserPerDay > 100 {
		return fmt.Errorf("每人每天最大偷菜次数必须在 1~100 之间")
	}
	if cfg.MaxStealPerFarmPerDay < 1 || cfg.MaxStealPerFarmPerDay > 100 {
		return fmt.Errorf("每农场每天最大被偷次数必须在 1~100 之间")
	}
	if cfg.StealCooldownSeconds < 0 || cfg.StealCooldownSeconds > 86400 {
		return fmt.Errorf("冷却时间必须在 0~86400 秒之间")
	}
	if cfg.MaxDailyLossRatioPerFarm < 0 || cfg.MaxDailyLossRatioPerFarm > 1.0 {
		return fmt.Errorf("每日最大损失比例必须在 0%%~100%% 之间")
	}
	if cfg.StealSuccessRate < 0 || cfg.StealSuccessRate > 100 {
		return fmt.Errorf("偷取成功率必须在 0~100 之间")
	}
	if cfg.ScarecrowBlockRate < 0 || cfg.ScarecrowBlockRate > 100 {
		return fmt.Errorf("稻草人拦截率必须在 0~100 之间")
	}
	if cfg.DogGuardRate < 0 || cfg.DogGuardRate > 100 {
		return fmt.Errorf("看门狗拦截率必须在 0~100 之间")
	}
	if cfg.LongCropHoursThreshold < 1 || cfg.LongCropHoursThreshold > 48 {
		return fmt.Errorf("长周期阈值必须在 1~48 小时之间")
	}
	if cfg.SuperLongCropHoursThreshold < cfg.LongCropHoursThreshold {
		return fmt.Errorf("超长周期阈值不能小于长周期阈值")
	}
	if cfg.LongCropOwnerKeepRatio < cfg.OwnerBaseKeepRatio || cfg.LongCropOwnerKeepRatio > 1.0 {
		return fmt.Errorf("长周期保底比例必须 >= 基础保底比例且 <= 100%%")
	}
	if cfg.CompensationRatio < 0 || cfg.CompensationRatio > 0.5 {
		return fmt.Errorf("补偿比例必须在 0%%~50%% 之间")
	}
	return nil
}

// ========== 变更对比 ==========

func diffStealConfig(old, new *model.FarmStealConfig) string {
	changes := "{"
	sep := ""
	addDiff := func(field string, oldVal, newVal interface{}) {
		if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
			changes += fmt.Sprintf(`%s"%s":{"old":%v,"new":%v}`, sep, field, oldVal, newVal)
			sep = ","
		}
	}
	addDiff("steal_enabled", old.StealEnabled, new.StealEnabled)
	addDiff("steal_bonus_only_enabled", old.StealBonusOnlyEnabled, new.StealBonusOnlyEnabled)
	addDiff("long_crop_protection_enabled", old.LongCropProtectionEnabled, new.LongCropProtectionEnabled)
	addDiff("owner_base_keep_ratio", old.OwnerBaseKeepRatio, new.OwnerBaseKeepRatio)
	addDiff("stealable_ratio", old.StealableRatio, new.StealableRatio)
	addDiff("owner_protection_minutes", old.OwnerProtectionMinutes, new.OwnerProtectionMinutes)
	addDiff("max_steal_per_plot", old.MaxStealPerPlot, new.MaxStealPerPlot)
	addDiff("max_steal_per_user_per_day", old.MaxStealPerUserPerDay, new.MaxStealPerUserPerDay)
	addDiff("max_steal_per_farm_per_day", old.MaxStealPerFarmPerDay, new.MaxStealPerFarmPerDay)
	addDiff("steal_cooldown_seconds", old.StealCooldownSeconds, new.StealCooldownSeconds)
	addDiff("max_daily_loss_ratio_per_farm", old.MaxDailyLossRatioPerFarm, new.MaxDailyLossRatioPerFarm)
	addDiff("steal_success_rate", old.StealSuccessRate, new.StealSuccessRate)
	addDiff("scarecrow_block_rate", old.ScarecrowBlockRate, new.ScarecrowBlockRate)
	addDiff("dog_guard_rate", old.DogGuardRate, new.DogGuardRate)
	addDiff("long_crop_hours_threshold", old.LongCropHoursThreshold, new.LongCropHoursThreshold)
	addDiff("super_long_crop_hours_threshold", old.SuperLongCropHoursThreshold, new.SuperLongCropHoursThreshold)
	addDiff("long_crop_owner_keep_ratio", old.LongCropOwnerKeepRatio, new.LongCropOwnerKeepRatio)
	addDiff("super_long_crop_bonus_only", old.SuperLongCropBonusOnly, new.SuperLongCropBonusOnly)
	addDiff("long_crop_protection_extra_min", old.LongCropProtectionExtraMin, new.LongCropProtectionExtraMin)
	addDiff("enable_steal_log", old.EnableStealLog, new.EnableStealLog)
	addDiff("notify_owner_when_stolen", old.NotifyOwnerWhenStolen, new.NotifyOwnerWhenStolen)
	addDiff("compensation_ratio", old.CompensationRatio, new.CompensationRatio)
	changes += "}"
	if changes == "{}" {
		return ""
	}
	return changes
}
