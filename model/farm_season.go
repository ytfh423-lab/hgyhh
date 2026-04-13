package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════
//  赛季系统 — 数据模型
// ═══════════════════════════════════════════════════════════

// ────── 赛季配置 ──────

const (
	SeasonStatusPending  = 0 // 未开始
	SeasonStatusRush     = 1 // 冲榜期
	SeasonStatusRest     = 2 // 休赛期
	SeasonStatusFinished = 3 // 已结束
)

type TgFarmSeason struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Code             string `json:"code" gorm:"type:varchar(64);uniqueIndex"`  // 赛季代号 e.g. "S01春日开荒季"
	WeeksPerSeason   int    `json:"weeks_per_season" gorm:"default:4"`         // X 周/赛季
	RushDays         int    `json:"rush_days" gorm:"default:0"`                // 冲榜期天数（赛季末尾）
	RestDays         int    `json:"rest_days" gorm:"default:1"`                // 休赛期天数（赛季结束后）
	Status           int    `json:"status" gorm:"default:0"`                   // 0=未开始 1=冲榜期 2=休赛期 3=已结束
	StartAt          int64  `json:"start_at"`                                  // 赛季开始时间戳
	EndAt            int64  `json:"end_at"`                                    // 赛季结束时间戳（含冲榜期，不含休赛期）
	PointsMultiplier int    `json:"points_multiplier" gorm:"default:100"`      // 积分倍率百分比（100=1x）
	CreatedAt        int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TgFarmSeason) TableName() string { return "tg_farm_seasons" }

// ────── 段位配置 ──────

type TgFarmSeasonTier struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TierKey        string `json:"tier_key" gorm:"type:varchar(32);uniqueIndex"` // bronze/silver/gold/platinum/diamond/rich
	TierName       string `json:"tier_name" gorm:"type:varchar(64)"`            // 显示名：青铜/白银/黄金/铂金/钻石/富可敌国
	TierLevel      int    `json:"tier_level" gorm:"default:0"`                  // 段位序号 0-5（越大越高）
	MinPoints      int    `json:"min_points" gorm:"default:0"`                  // 达到该段位所需最低积分
	InitialBalance int    `json:"initial_balance" gorm:"default:0"`             // 继承初始资金（quota 单位）
	GiftItems      string `json:"gift_items" gorm:"type:text"`                  // 继承礼包 JSON [{"item":"fertilizer_adv","qty":10}, ...]
	Emoji          string `json:"emoji" gorm:"type:varchar(16)"`                // 段位图标
	Color          string `json:"color" gorm:"type:varchar(16)"`                // 段位颜色
	CreatedAt      int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TgFarmSeasonTier) TableName() string { return "tg_farm_season_tiers" }

// ────── 玩家赛季数据 ──────

type TgFarmSeasonPlayer struct {
	Id              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId      string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_season_player"`
	SeasonId        int    `json:"season_id" gorm:"uniqueIndex:idx_season_player"`
	Points          int    `json:"points" gorm:"default:0"`                        // 赛季积分
	HighestTierKey  string `json:"highest_tier_key" gorm:"type:varchar(32)"`       // 本赛季最高段位 key
	CurrentTierKey  string `json:"current_tier_key" gorm:"type:varchar(32)"`       // 当前段位 key
	InheritedFrom   string `json:"inherited_from" gorm:"type:varchar(32)"`         // 从哪个段位继承进来的
	InheritanceApplied bool `json:"inheritance_applied" gorm:"default:false"`      // 继承奖励是否已发放
	CreatedAt       int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TgFarmSeasonPlayer) TableName() string { return "tg_farm_season_players" }

// ────── 积分流水日志 ──────

type TgFarmSeasonPointsLog struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index:idx_sp_log_tg"`
	SeasonId   int    `json:"season_id" gorm:"index:idx_sp_log_season"`
	Action     string `json:"action" gorm:"type:varchar(32)"`  // water/harvest/plant/challenge/bonus
	Points     int    `json:"points"`                           // 获得积分（可负）
	Detail     string `json:"detail" gorm:"type:varchar(255)"` // 描述
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
}

func (TgFarmSeasonPointsLog) TableName() string { return "tg_farm_season_points_logs" }

// ────── 玩家历史最高段位（跨赛季保留） ──────

type TgFarmSeasonHistory struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId     string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_season_hist"`
	SeasonId       int    `json:"season_id" gorm:"uniqueIndex:idx_season_hist"`
	FinalTierKey   string `json:"final_tier_key" gorm:"type:varchar(32)"`
	FinalPoints    int    `json:"final_points"`
	FinalRank      int    `json:"final_rank"`
	CreatedAt      int64  `json:"created_at" gorm:"autoCreateTime"`
}

func (TgFarmSeasonHistory) TableName() string { return "tg_farm_season_histories" }

// ────── 防作弊日志 ──────

type TgFarmSeasonAntiCheat struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index"`
	SeasonId   int    `json:"season_id"`
	RuleKey    string `json:"rule_key" gorm:"type:varchar(32)"`  // points_spike / item_anomaly / ...
	Detail     string `json:"detail" gorm:"type:varchar(512)"`
	Severity   int    `json:"severity" gorm:"default:1"`         // 1=警告 2=临时封禁 3=永久封禁
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
}

func (TgFarmSeasonAntiCheat) TableName() string { return "tg_farm_season_anti_cheats" }

// ────── 防作弊规则配置 ──────

type TgFarmSeasonAntiCheatRule struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RuleKey     string `json:"rule_key" gorm:"type:varchar(32);uniqueIndex"` // points_spike / item_anomaly
	RuleName    string `json:"rule_name" gorm:"type:varchar(64)"`
	Enabled     bool   `json:"enabled" gorm:"default:true"`
	Threshold   int    `json:"threshold" gorm:"default:0"`                    // 触发阈值
	WindowSecs  int    `json:"window_secs" gorm:"default:3600"`              // 检测窗口（秒）
	Action      int    `json:"action" gorm:"default:1"`                       // 1=警告 2=临时封禁 3=永久封禁
	BanDuration int    `json:"ban_duration" gorm:"default:3600"`             // 临时封禁时长（秒）
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TgFarmSeasonAntiCheatRule) TableName() string { return "tg_farm_season_anti_cheat_rules" }

// ────── 赛季积分任务配置（后台可配） ──────

type TgFarmSeasonPointsRule struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Action     string `json:"action" gorm:"type:varchar(32);uniqueIndex"` // water / harvest / plant / steal / ...
	ActionName string `json:"action_name" gorm:"type:varchar(64)"`
	Points     int    `json:"points" gorm:"default:1"`                     // 每次获得积分
	DailyCap   int    `json:"daily_cap" gorm:"default:0"`                  // 每日上限（0=无限）
	Enabled    bool   `json:"enabled" gorm:"default:true"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TgFarmSeasonPointsRule) TableName() string { return "tg_farm_season_points_rules" }

// ═══════════════════════════════════════════════════════════
//  CRUD 操作
// ═══════════════════════════════════════════════════════════

// ────── 赛季 CRUD ──────

func GetAllSeasons() ([]TgFarmSeason, error) {
	var seasons []TgFarmSeason
	err := DB.Order("id desc").Find(&seasons).Error
	return seasons, err
}

func GetSeasonById(id int) (*TgFarmSeason, error) {
	var season TgFarmSeason
	err := DB.Where("id = ?", id).First(&season).Error
	return &season, err
}

func GetActiveSeason() (*TgFarmSeason, error) {
	var season TgFarmSeason
	err := DB.Where("status IN (?, ?)", SeasonStatusRush, SeasonStatusRest).Order("id desc").First(&season).Error
	if err != nil {
		return nil, err
	}
	return &season, nil
}

func GetCurrentOrNextSeason() (*TgFarmSeason, error) {
	var season TgFarmSeason
	err := DB.Where("status IN (?, ?, ?)", SeasonStatusPending, SeasonStatusRush, SeasonStatusRest).
		Order("start_at asc").First(&season).Error
	if err != nil {
		return nil, err
	}
	return &season, nil
}

func CreateSeason(season *TgFarmSeason) error {
	return DB.Create(season).Error
}

func UpdateSeason(season *TgFarmSeason) error {
	return DB.Save(season).Error
}

func DeleteSeason(id int) error {
	return DB.Where("id = ?", id).Delete(&TgFarmSeason{}).Error
}

// ────── 段位 CRUD ──────

func GetAllSeasonTiers() ([]TgFarmSeasonTier, error) {
	var tiers []TgFarmSeasonTier
	err := DB.Order("tier_level asc").Find(&tiers).Error
	return tiers, err
}

func GetSeasonTierByKey(key string) (*TgFarmSeasonTier, error) {
	var tier TgFarmSeasonTier
	err := DB.Where("tier_key = ?", key).First(&tier).Error
	return &tier, err
}

func CreateSeasonTier(tier *TgFarmSeasonTier) error {
	return DB.Create(tier).Error
}

func UpdateSeasonTier(tier *TgFarmSeasonTier) error {
	return DB.Save(tier).Error
}

func DeleteSeasonTier(id int) error {
	return DB.Where("id = ?", id).Delete(&TgFarmSeasonTier{}).Error
}

// GetTierForPoints 根据积分返回对应段位
func GetTierForPoints(points int) (*TgFarmSeasonTier, error) {
	var tier TgFarmSeasonTier
	err := DB.Where("min_points <= ?", points).Order("min_points desc").First(&tier).Error
	if err != nil {
		return nil, err
	}
	return &tier, nil
}

// ────── 玩家赛季数据 ──────

func GetSeasonPlayer(telegramId string, seasonId int) (*TgFarmSeasonPlayer, error) {
	var player TgFarmSeasonPlayer
	err := DB.Where("telegram_id = ? AND season_id = ?", telegramId, seasonId).First(&player).Error
	return &player, err
}

func GetOrCreateSeasonPlayer(telegramId string, seasonId int) (*TgFarmSeasonPlayer, error) {
	player, err := GetSeasonPlayer(telegramId, seasonId)
	if err == nil {
		return player, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	// 查找历史最高段位用于继承
	inheritTierKey := GetPlayerLifetimeHighestTier(telegramId)
	player = &TgFarmSeasonPlayer{
		TelegramId:     telegramId,
		SeasonId:       seasonId,
		Points:         0,
		CurrentTierKey:  "bronze",
		HighestTierKey:  "bronze",
		InheritedFrom:   inheritTierKey,
		InheritanceApplied: false,
	}
	err = DB.Create(player).Error
	return player, err
}

func UpdateSeasonPlayer(player *TgFarmSeasonPlayer) error {
	return DB.Save(player).Error
}

// AddSeasonPoints 给玩家加积分，自动更新段位
func AddSeasonPoints(telegramId string, seasonId int, action string, points int, detail string) error {
	player, err := GetOrCreateSeasonPlayer(telegramId, seasonId)
	if err != nil {
		return err
	}
	player.Points += points
	if player.Points < 0 {
		player.Points = 0
	}
	// 更新段位
	tier, tierErr := GetTierForPoints(player.Points)
	if tierErr == nil && tier != nil {
		player.CurrentTierKey = tier.TierKey
		// 更新最高段位
		if tier.TierLevel > getTierLevelFromKey(player.HighestTierKey) {
			player.HighestTierKey = tier.TierKey
		}
	}
	if err := UpdateSeasonPlayer(player); err != nil {
		return err
	}
	// 记录积分流水
	log := &TgFarmSeasonPointsLog{
		TelegramId: telegramId,
		SeasonId:   seasonId,
		Action:     action,
		Points:     points,
		Detail:     detail,
	}
	return DB.Create(log).Error
}

// getTierLevelFromKey 从内存缓存获取 tier level（避免循环查询）
func getTierLevelFromKey(tierKey string) int {
	if tierKey == "" {
		return -1
	}
	var tier TgFarmSeasonTier
	if err := DB.Where("tier_key = ?", tierKey).First(&tier).Error; err != nil {
		return -1
	}
	return tier.TierLevel
}

// GetSeasonLeaderboard 获取赛季积分排行榜
func GetSeasonLeaderboard(seasonId int, limit int) ([]TgFarmSeasonPlayer, error) {
	var players []TgFarmSeasonPlayer
	err := DB.Where("season_id = ?", seasonId).Order("points desc").Limit(limit).Find(&players).Error
	return players, err
}

// GetSeasonPlayerRank 获取玩家在赛季中的排名
func GetSeasonPlayerRank(telegramId string, seasonId int) (int64, error) {
	player, err := GetSeasonPlayer(telegramId, seasonId)
	if err != nil {
		return 0, err
	}
	var rank int64
	err = DB.Model(&TgFarmSeasonPlayer{}).
		Where("season_id = ? AND points > ?", seasonId, player.Points).
		Count(&rank).Error
	return rank + 1, err
}

// GetSeasonPointsLogs 获取玩家积分流水
func GetSeasonPointsLogs(telegramId string, seasonId int, limit int) ([]TgFarmSeasonPointsLog, error) {
	var logs []TgFarmSeasonPointsLog
	err := DB.Where("telegram_id = ? AND season_id = ?", telegramId, seasonId).
		Order("id desc").Limit(limit).Find(&logs).Error
	return logs, err
}

// ────── 历史记录 ──────

func CreateSeasonHistory(history *TgFarmSeasonHistory) error {
	return DB.Create(history).Error
}

func GetPlayerSeasonHistories(telegramId string) ([]TgFarmSeasonHistory, error) {
	var histories []TgFarmSeasonHistory
	err := DB.Where("telegram_id = ?", telegramId).Order("season_id desc").Find(&histories).Error
	return histories, err
}

// GetPlayerLifetimeHighestTier 获取玩家历史最高段位
func GetPlayerLifetimeHighestTier(telegramId string) string {
	var histories []TgFarmSeasonHistory
	if err := DB.Where("telegram_id = ?", telegramId).Find(&histories).Error; err != nil {
		return ""
	}
	highestTierKey := ""
	highestLevel := -1
	for _, history := range histories {
		level := getTierLevelFromKey(history.FinalTierKey)
		if level > highestLevel {
			highestLevel = level
			highestTierKey = history.FinalTierKey
		}
	}
	return highestTierKey
}

// ────── 赛季重置 ──────

// ResetAllPlayersForNewSeason 赛季结算：保存历史 → 清除赛季数据
func ResetAllPlayersForNewSeason(endingSeasonId int) error {
	// 1. 保存所有玩家历史记录
	var players []TgFarmSeasonPlayer
	if err := DB.Where("season_id = ?", endingSeasonId).Order("points desc, id asc").Find(&players).Error; err != nil {
		return err
	}
	for i, p := range players {
		rank := int64(i + 1)
		history := &TgFarmSeasonHistory{
			TelegramId:   p.TelegramId,
			SeasonId:     endingSeasonId,
			FinalTierKey: p.HighestTierKey,
			FinalPoints:  p.Points,
			FinalRank:    int(rank),
		}
		_ = CreateSeasonHistory(history)
		if userId, err := resolveSeasonUserIdByFarmID(p.TelegramId); err == nil && userId > 0 {
			ResetFarmForNewSeason(userId, p.TelegramId)
		}
	}
	// 2. 清除赛季积分日志（可选：保留 N 天）
	cutoff := time.Now().AddDate(0, 0, -7).Unix()
	DB.Where("season_id = ? AND created_at < ?", endingSeasonId, cutoff).Delete(&TgFarmSeasonPointsLog{})
	return nil
}

func resolveSeasonUserIdByFarmID(farmID string) (int, error) {
	if farmID == "" {
		return 0, fmt.Errorf("farm id is empty")
	}
	if strings.HasPrefix(farmID, "u_") {
		userId, err := strconv.Atoi(strings.TrimPrefix(farmID, "u_"))
		if err != nil {
			return 0, err
		}
		return userId, nil
	}
	user := User{TelegramId: farmID}
	if err := user.FillUserByTelegramId(); err != nil {
		return 0, err
	}
	return user.Id, nil
}

// ResetFarmForNewSeason 赛季全重置（对单个玩家）
// 重置内容：作物、道具（保留 _level, _prestige）、仓库、狗、加工坊、牧场、自动化、交易
// 保留：历史最高段位
func ResetFarmForNewSeason(userId int, telegramId string) {
	DB.Where("telegram_id = ?", telegramId).Delete(&TgFarmPlot{})
	DB.Where("telegram_id = ? AND item_type NOT IN ('_level','_prestige')", telegramId).Delete(&TgFarmItem{})
	DB.Where("telegram_id = ?", telegramId).Delete(&TgFarmWarehouse{})
	DB.Where("telegram_id = ?", telegramId).Delete(&TgFarmDog{})
	DB.Where("telegram_id = ? AND status IN (1,2)", telegramId).Delete(&TgFarmProcess{})
	DB.Where("telegram_id = ?", telegramId).Delete(&TgRanchAnimal{})
	DB.Where("telegram_id = ?", telegramId).Delete(&TgFarmAutomation{})
	DB.Where("seller_id = ? OR buyer_id = ?", telegramId, telegramId).Delete(&TgFarmTrade{})
	// 重置等级到 1
	SetFarmLevel(telegramId, 1)
	// 重置转生等级到 0
	SetPrestigeLevel(telegramId, 0)
	// 清除任务进度
	DB.Where("telegram_id = ?", telegramId).Delete(&TgFarmTaskClaim{})
	// 清除成就
	DB.Where("telegram_id = ?", telegramId).Delete(&TgFarmAchievement{})
	common.SysLog(fmt.Sprintf("Farm season reset: user %d (%s) farm data cleared", userId, telegramId))
}

// ApplySeasonInheritance 应用赛季继承福利
func ApplySeasonInheritance(userId int, telegramId string, tierKey string) error {
	if tierKey == "" {
		return nil
	}
	tier, err := GetSeasonTierByKey(tierKey)
	if err != nil {
		return err
	}
	// 1. 发放初始资金
	if tier.InitialBalance > 0 {
		_ = IncreaseUserQuota(userId, tier.InitialBalance, true)
		AddFarmLog(telegramId, "season_inherit", tier.InitialBalance,
			fmt.Sprintf("赛季继承[%s]初始资金", tier.TierName))
	}
	// 2. 发放礼包道具
	if tier.GiftItems != "" {
		var gifts []struct {
			Item string `json:"item"`
			Qty  int    `json:"qty"`
		}
		if err := common.Unmarshal([]byte(tier.GiftItems), &gifts); err == nil {
			for _, gift := range gifts {
				if gift.Item != "" && gift.Qty > 0 {
					_ = IncrementFarmItem(telegramId, gift.Item, gift.Qty)
					AddFarmLog(telegramId, "season_gift", 0,
						fmt.Sprintf("赛季礼包[%s]: %s ×%d", tier.TierName, gift.Item, gift.Qty))
				}
			}
		}
	}
	return nil
}

func EnsureSeasonInheritanceApplied(userId int, telegramId string, seasonId int) error {
	player, err := GetOrCreateSeasonPlayer(telegramId, seasonId)
	if err != nil {
		return err
	}
	if player.InheritanceApplied || player.InheritedFrom == "" {
		return nil
	}
	if err := ApplySeasonInheritance(userId, telegramId, player.InheritedFrom); err != nil {
		return err
	}
	player.InheritanceApplied = true
	return UpdateSeasonPlayer(player)
}

// ────── 积分规则 CRUD ──────

func GetAllSeasonPointsRules() ([]TgFarmSeasonPointsRule, error) {
	var rules []TgFarmSeasonPointsRule
	err := DB.Order("id asc").Find(&rules).Error
	return rules, err
}

func GetSeasonPointsRule(action string) (*TgFarmSeasonPointsRule, error) {
	var rule TgFarmSeasonPointsRule
	err := DB.Where("action = ?", action).First(&rule).Error
	return &rule, err
}

func CreateSeasonPointsRule(rule *TgFarmSeasonPointsRule) error {
	return DB.Create(rule).Error
}

func UpdateSeasonPointsRule(rule *TgFarmSeasonPointsRule) error {
	return DB.Save(rule).Error
}

func DeleteSeasonPointsRule(id int) error {
	return DB.Where("id = ?", id).Delete(&TgFarmSeasonPointsRule{}).Error
}

// ────── 防作弊规则 CRUD ──────

func GetAllAntiCheatRules() ([]TgFarmSeasonAntiCheatRule, error) {
	var rules []TgFarmSeasonAntiCheatRule
	err := DB.Order("id asc").Find(&rules).Error
	return rules, err
}

func CreateAntiCheatRule(rule *TgFarmSeasonAntiCheatRule) error {
	return DB.Create(rule).Error
}

func UpdateAntiCheatRule(rule *TgFarmSeasonAntiCheatRule) error {
	return DB.Save(rule).Error
}

func DeleteAntiCheatRule(id int) error {
	return DB.Where("id = ?", id).Delete(&TgFarmSeasonAntiCheatRule{}).Error
}

// CreateAntiCheatLog 记录防作弊触发
func CreateAntiCheatLog(log *TgFarmSeasonAntiCheat) error {
	return DB.Create(log).Error
}

func GetAntiCheatLogs(seasonId int, limit int) ([]TgFarmSeasonAntiCheat, error) {
	var logs []TgFarmSeasonAntiCheat
	q := DB.Order("id desc").Limit(limit)
	if seasonId > 0 {
		q = q.Where("season_id = ?", seasonId)
	}
	err := q.Find(&logs).Error
	return logs, err
}

// CountRecentSeasonPoints 统计指定时间窗口内的积分获取总量（防作弊用）
func CountRecentSeasonPoints(telegramId string, seasonId int, sinceSecs int64) (int, error) {
	cutoff := time.Now().Unix() - sinceSecs
	var total int
	err := DB.Model(&TgFarmSeasonPointsLog{}).
		Where("telegram_id = ? AND season_id = ? AND created_at > ? AND points > 0", telegramId, seasonId, cutoff).
		Select("COALESCE(SUM(points),0)").Scan(&total).Error
	return total, err
}

// CountRecentAdvancedFertilizerPurchases 统计指定时间窗口内高级化肥购买数量（防作弊用）
func CountRecentAdvancedFertilizerPurchases(telegramId string, sinceSecs int64) (int, error) {
	cutoff := time.Now().Unix() - sinceSecs
	var totalCost int
	err := DB.Model(&TgFarmLog{}).
		Where("telegram_id = ? AND action = ? AND created_at > ? AND detail LIKE ? AND amount < 0",
			telegramId, "shop", cutoff, "%高级化肥%").
		Select("COALESCE(SUM(-amount),0)").Scan(&totalCost).Error
	if err != nil {
		return 0, err
	}
	unitCost := 500000
	return totalCost / unitCost, nil
}

// GetSeasonAllPlayerCount 获取赛季参与人数
func GetSeasonAllPlayerCount(seasonId int) int64 {
	var count int64
	DB.Model(&TgFarmSeasonPlayer{}).Where("season_id = ?", seasonId).Count(&count)
	return count
}
