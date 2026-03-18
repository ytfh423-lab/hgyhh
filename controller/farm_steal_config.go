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
			"steal_enabled":      cfg.StealEnabled,
			"protection_minutes": cfg.OwnerProtectionMinutes,
			"max_steal_per_day":  cfg.MaxStealPerUserPerDay,
			"cooldown_minutes":   cfg.StealCooldownSeconds / 60,
		},
	})
}

// ========== 校验 ==========

func validateStealConfig(cfg *model.FarmStealConfig) error {
	if cfg.OwnerProtectionMinutes < 0 || cfg.OwnerProtectionMinutes > 1440 {
		return fmt.Errorf("保护期必须在 0~1440 分钟之间")
	}
	if cfg.MaxStealPerUserPerDay < 1 || cfg.MaxStealPerUserPerDay > 100 {
		return fmt.Errorf("每人每天最大偷菜次数必须在 1~100 之间")
	}
	if cfg.StealCooldownSeconds < 0 || cfg.StealCooldownSeconds > 86400 {
		return fmt.Errorf("冷却时间必须在 0~86400 秒之间")
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
	addDiff("owner_protection_minutes", old.OwnerProtectionMinutes, new.OwnerProtectionMinutes)
	addDiff("max_steal_per_user_per_day", old.MaxStealPerUserPerDay, new.MaxStealPerUserPerDay)
	addDiff("steal_cooldown_seconds", old.StealCooldownSeconds, new.StealCooldownSeconds)
	addDiff("enable_steal_log", old.EnableStealLog, new.EnableStealLog)
	changes += "}"
	if changes == "{}" {
		return ""
	}
	return changes
}
