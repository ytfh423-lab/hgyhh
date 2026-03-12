package model

import (
	"sync"
	"time"

	"gorm.io/gorm"
)

// FarmStealConfig 偷菜机制配置（单行表，id=1）
type FarmStealConfig struct {
	Id int `json:"id" gorm:"primaryKey"`

	// === 基础开关 ===
	StealEnabled              bool `json:"steal_enabled" gorm:"default:true"`
	StealBonusOnlyEnabled     bool `json:"steal_bonus_only_enabled" gorm:"default:false"`
	LongCropProtectionEnabled bool `json:"long_crop_protection_enabled" gorm:"default:true"`

	// === 收益拆分 ===
	OwnerBaseKeepRatio float64 `json:"owner_base_keep_ratio" gorm:"default:0.80"` // 主人保底收益比例 0.70~0.95
	StealableRatio     float64 `json:"stealable_ratio" gorm:"default:0.20"`       // 可偷收益比例 0.05~0.30

	// === 成熟保护期 ===
	OwnerProtectionMinutes int `json:"owner_protection_minutes" gorm:"default:60"` // 成熟后主人优先收获分钟数

	// === 偷取次数限制 ===
	MaxStealPerPlot       int `json:"max_steal_per_plot" gorm:"default:1"`          // 每块地最多被偷次数
	MaxStealPerUserPerDay int `json:"max_steal_per_user_per_day" gorm:"default:10"` // 每人每天最多偷菜次数
	MaxStealPerFarmPerDay int `json:"max_steal_per_farm_per_day" gorm:"default:5"`  // 每个农场每天最多被偷次数
	StealCooldownSeconds  int `json:"steal_cooldown_seconds" gorm:"default:1800"`   // 同一人偷同一人冷却秒数

	// === 损失上限 ===
	MaxDailyLossRatioPerFarm float64 `json:"max_daily_loss_ratio_per_farm" gorm:"default:0.20"` // 单农场每日最大损失比例

	// === 概率 ===
	StealSuccessRate   int `json:"steal_success_rate" gorm:"default:100"`   // 偷取成功率% (0~100)
	ScarecrowBlockRate int `json:"scarecrow_block_rate" gorm:"default:30"`  // 稻草人拦截率% (0~100)
	DogGuardRate       int `json:"dog_guard_rate" gorm:"default:50"`        // 看门狗拦截率% (0~100)

	// === 长周期作物保护 ===
	LongCropHoursThreshold      int     `json:"long_crop_hours_threshold" gorm:"default:8"`        // 长周期判定阈值(小时)
	SuperLongCropHoursThreshold int     `json:"super_long_crop_hours_threshold" gorm:"default:12"` // 超长周期判定阈值(小时)
	LongCropOwnerKeepRatio      float64 `json:"long_crop_owner_keep_ratio" gorm:"default:0.90"`    // 长周期保底比例
	SuperLongCropBonusOnly      bool    `json:"super_long_crop_bonus_only" gorm:"default:true"`    // 超长周期仅可偷bonus
	LongCropProtectionExtraMin  int     `json:"long_crop_protection_extra_min" gorm:"default:60"`  // 长周期额外保护分钟

	// === 补偿与日志 ===
	EnableStealLog        bool    `json:"enable_steal_log" gorm:"default:true"`
	NotifyOwnerWhenStolen bool    `json:"notify_owner_when_stolen" gorm:"default:true"`
	CompensationRatio     float64 `json:"compensation_ratio" gorm:"default:0"` // 被偷补偿比例 0~0.5

	// === 审计 ===
	UpdatedBy int   `json:"updated_by" gorm:"default:0"`
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime"`
}

// FarmStealConfigLog 偷菜配置修改日志
type FarmStealConfigLog struct {
	Id            int    `json:"id" gorm:"primaryKey;autoIncrement"`
	OperatorId    int    `json:"operator_id"`
	ChangedFields string `json:"changed_fields" gorm:"type:text"` // JSON: {"field": {"old": x, "new": y}}
	CreatedAt     int64  `json:"created_at" gorm:"autoCreateTime"`
}

// ========== 缓存 ==========

var (
	stealConfigCache     *FarmStealConfig
	stealConfigCacheMu   sync.RWMutex
	stealConfigCacheTime int64
)

const stealConfigCacheTTL = 30 // 缓存30秒

// DefaultStealConfig 返回默认配置
func DefaultStealConfig() *FarmStealConfig {
	return &FarmStealConfig{
		Id:                          1,
		StealEnabled:                true,
		StealBonusOnlyEnabled:       false,
		LongCropProtectionEnabled:   true,
		OwnerBaseKeepRatio:          0.80,
		StealableRatio:              0.20,
		OwnerProtectionMinutes:      60,
		MaxStealPerPlot:             1,
		MaxStealPerUserPerDay:       10,
		MaxStealPerFarmPerDay:       5,
		StealCooldownSeconds:        1800,
		MaxDailyLossRatioPerFarm:    0.20,
		StealSuccessRate:            100,
		ScarecrowBlockRate:          30,
		DogGuardRate:                50,
		LongCropHoursThreshold:      8,
		SuperLongCropHoursThreshold: 12,
		LongCropOwnerKeepRatio:      0.90,
		SuperLongCropBonusOnly:      true,
		LongCropProtectionExtraMin:  60,
		EnableStealLog:              true,
		NotifyOwnerWhenStolen:       true,
		CompensationRatio:           0,
	}
}

// GetStealConfig 获取偷菜配置（带缓存）
func GetStealConfig() *FarmStealConfig {
	stealConfigCacheMu.RLock()
	if stealConfigCache != nil && time.Now().Unix()-stealConfigCacheTime < stealConfigCacheTTL {
		cfg := *stealConfigCache
		stealConfigCacheMu.RUnlock()
		return &cfg
	}
	stealConfigCacheMu.RUnlock()

	stealConfigCacheMu.Lock()
	defer stealConfigCacheMu.Unlock()

	// double check
	if stealConfigCache != nil && time.Now().Unix()-stealConfigCacheTime < stealConfigCacheTTL {
		cfg := *stealConfigCache
		return &cfg
	}

	var cfg FarmStealConfig
	err := DB.First(&cfg, 1).Error
	if err != nil {
		// 不存在则创建默认
		cfg = *DefaultStealConfig()
		DB.Create(&cfg)
	}
	stealConfigCache = &cfg
	stealConfigCacheTime = time.Now().Unix()
	result := cfg
	return &result
}

// UpdateStealConfig 更新偷菜配置
func UpdateStealConfig(cfg *FarmStealConfig) error {
	cfg.Id = 1
	cfg.UpdatedAt = time.Now().Unix()
	err := DB.Save(cfg).Error
	if err != nil {
		return err
	}
	// 清除缓存
	stealConfigCacheMu.Lock()
	stealConfigCache = nil
	stealConfigCacheMu.Unlock()
	return nil
}

// InvalidateStealConfigCache 清除缓存
func InvalidateStealConfigCache() {
	stealConfigCacheMu.Lock()
	stealConfigCache = nil
	stealConfigCacheMu.Unlock()
}

// CreateStealConfigLog 创建配置修改日志
func CreateStealConfigLog(log *FarmStealConfigLog) error {
	return DB.Create(log).Error
}

// GetStealConfigLogs 获取配置修改日志
func GetStealConfigLogs(page, pageSize int) ([]FarmStealConfigLog, int64, error) {
	var logs []FarmStealConfigLog
	var total int64
	DB.Model(&FarmStealConfigLog{}).Count(&total)
	err := DB.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

// ========== 偷菜统计查询 ==========

// CountThiefStealsToday 统计某玩家今日偷菜次数
func CountThiefStealsToday(thiefId string) int64 {
	todayStart := todayStartUnix()
	var count int64
	DB.Model(&TgFarmStealLog{}).
		Where("thief_id = ? AND created_at >= ?", thiefId, todayStart).
		Count(&count)
	return count
}

// CountFarmStolenToday 统计某农场今日被偷次数
func CountFarmStolenToday(victimId string) int64 {
	todayStart := todayStartUnix()
	var count int64
	DB.Model(&TgFarmStealLog{}).
		Where("victim_id = ? AND created_at >= ?", victimId, todayStart).
		Count(&count)
	return count
}

// SumFarmStolenValueToday 统计某农场今日被偷总金额
func SumFarmStolenValueToday(victimId string) int64 {
	todayStart := todayStartUnix()
	var total int64
	DB.Model(&TgFarmStealLog{}).
		Where("victim_id = ? AND created_at >= ?", victimId, todayStart).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total)
	return total
}

func todayStartUnix() int64 {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location()).Unix()
}

// GetStealablePlotsV2 获取可偷地块（新版：考虑保护期和单块偷取上限）
func GetStealablePlotsV2(victimId string, maxStealPerPlot int) ([]*TgFarmPlot, error) {
	var plots []*TgFarmPlot
	err := DB.Where("telegram_id = ? AND status = 2 AND stolen_count < ?", victimId, maxStealPerPlot).
		Find(&plots).Error
	return plots, err
}

// GetMatureFarmTargetsV2 获取偷菜目标（新版）
func GetMatureFarmTargetsV2(excludeId string, maxStealPerPlot int) ([]FarmStealTarget, error) {
	var results []FarmStealTarget
	err := DB.Model(&TgFarmPlot{}).
		Select("telegram_id, count(*) as count").
		Where("telegram_id != ? AND status = 2 AND stolen_count < ?", excludeId, maxStealPerPlot).
		Group("telegram_id").
		Scan(&results).Error
	return results, err
}

// IncrementPlotStolenBy 增加地块被偷单位数
func IncrementPlotStolenBy(plotId int, units int) error {
	return DB.Model(&TgFarmPlot{}).Where("id = ?", plotId).
		Update("stolen_count", gorm.Expr("stolen_count + ?", units)).Error
}
