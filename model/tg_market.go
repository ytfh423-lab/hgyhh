package model

import (
	"time"

	"gorm.io/gorm"
)

// ========== 市场价格历史 ==========

type TgMarketPriceHistory struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ItemKey          string `json:"item_key" gorm:"type:varchar(64);index:idx_market_ph_item_date"`
	Category         string `json:"category" gorm:"type:varchar(32)"`
	DateStr          string `json:"date_str" gorm:"type:varchar(16);index:idx_market_ph_item_date"`
	TickIndex        int    `json:"tick_index"`
	Multiplier       int    `json:"multiplier"`
	PrevMultiplier   int    `json:"prev_multiplier"`
	SeasonFactor     int    `json:"season_factor"`
	SupplyFactor     int    `json:"supply_factor"`
	TrendFactor      int    `json:"trend_factor"`
	EventFactor      int    `json:"event_factor"`
	NoiseFactor      int    `json:"noise_factor"`
	MeanRevFactor    int    `json:"mean_rev_factor"`
	Timestamp        int64  `json:"timestamp"`
}

func CreateMarketPriceHistory(records []*TgMarketPriceHistory) error {
	if len(records) == 0 {
		return nil
	}
	return DB.CreateInBatches(records, 100).Error
}

func GetMarketPriceHistory(itemKey string, limit int) ([]*TgMarketPriceHistory, error) {
	var records []*TgMarketPriceHistory
	err := DB.Where("item_key = ?", itemKey).Order("timestamp DESC").Limit(limit).Find(&records).Error
	return records, err
}

func GetMarketPriceHistoryAll(limit int) ([]*TgMarketPriceHistory, error) {
	var records []*TgMarketPriceHistory
	err := DB.Order("timestamp DESC").Limit(limit).Find(&records).Error
	return records, err
}

func GetMarketPriceHistoryByDate(dateStr string) ([]*TgMarketPriceHistory, error) {
	var records []*TgMarketPriceHistory
	err := DB.Where("date_str = ?", dateStr).Find(&records).Error
	return records, err
}

// GetLatestMarketPrices 获取每个商品最新一条价格记录
func GetLatestMarketPrices() ([]*TgMarketPriceHistory, error) {
	var records []*TgMarketPriceHistory
	// 使用子查询获取每个 item_key 的最大 timestamp
	subQuery := DB.Model(&TgMarketPriceHistory{}).Select("item_key, MAX(timestamp) as max_ts").Group("item_key")
	err := DB.Where("(item_key, timestamp) IN (?)", subQuery).Find(&records).Error
	if err != nil {
		// fallback: 简单方式
		err = DB.Order("timestamp DESC").Limit(500).Find(&records).Error
	}
	return records, err
}

// CleanOldMarketHistory 清理超过 days 天的旧记录
func CleanOldMarketHistory(days int) error {
	cutoff := time.Now().Unix() - int64(days*86400)
	return DB.Where("timestamp < ?", cutoff).Delete(&TgMarketPriceHistory{}).Error
}

// ========== 市场事件 ==========

type TgMarketEvent struct {
	Id              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name            string `json:"name" gorm:"type:varchar(128)"`
	EventType       string `json:"event_type" gorm:"type:varchar(32)"`
	AffectedItems   string `json:"affected_items" gorm:"type:text"`
	AffectedCats    string `json:"affected_cats" gorm:"type:varchar(256)"`
	EffectDirection int    `json:"effect_direction"`
	EffectValue     int    `json:"effect_value"`
	StartTime       int64  `json:"start_time"`
	EndTime         int64  `json:"end_time"`
	IsActive        int    `json:"is_active" gorm:"default:1"`
	IsPublic        int    `json:"is_public" gorm:"default:1"`
	Description     string `json:"description" gorm:"type:text"`
	CreatedAt       int64  `json:"created_at"`
}

func GetActiveMarketEvents() ([]*TgMarketEvent, error) {
	now := time.Now().Unix()
	var events []*TgMarketEvent
	err := DB.Where("is_active = 1 AND start_time <= ? AND end_time > ?", now, now).Find(&events).Error
	return events, err
}

func GetPublicMarketEvents() ([]*TgMarketEvent, error) {
	now := time.Now().Unix()
	var events []*TgMarketEvent
	err := DB.Where("is_active = 1 AND is_public = 1 AND start_time <= ? AND end_time > ?", now, now).Find(&events).Error
	return events, err
}

func GetAllMarketEvents() ([]*TgMarketEvent, error) {
	var events []*TgMarketEvent
	err := DB.Order("created_at DESC").Limit(100).Find(&events).Error
	return events, err
}

func CreateMarketEvent(event *TgMarketEvent) error {
	event.CreatedAt = time.Now().Unix()
	return DB.Create(event).Error
}

func UpdateMarketEvent(event *TgMarketEvent) error {
	return DB.Save(event).Error
}

func DeleteMarketEvent(id int) error {
	return DB.Delete(&TgMarketEvent{}, id).Error
}

// ========== 供需统计 ==========

type TgMarketSupplyDemand struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ItemKey    string `json:"item_key" gorm:"type:varchar(64);index:idx_market_sd_item_date"`
	DateStr    string `json:"date_str" gorm:"type:varchar(16);index:idx_market_sd_item_date"`
	SellVolume int    `json:"sell_volume" gorm:"default:0"`
	BuyVolume  int    `json:"buy_volume" gorm:"default:0"`
	Timestamp  int64  `json:"timestamp"`
}

// RecordMarketSell 记录出售量
func RecordMarketSell(itemKey string, quantity int) {
	dateStr := time.Now().Format("20060102")
	var sd TgMarketSupplyDemand
	err := DB.Where("item_key = ? AND date_str = ?", itemKey, dateStr).First(&sd).Error
	if err != nil {
		sd = TgMarketSupplyDemand{
			ItemKey:    itemKey,
			DateStr:    dateStr,
			SellVolume: quantity,
			Timestamp:  time.Now().Unix(),
		}
		DB.Create(&sd)
		return
	}
	DB.Model(&TgMarketSupplyDemand{}).Where("id = ?", sd.Id).
		Update("sell_volume", gorm.Expr("sell_volume + ?", quantity))
}

// RecordMarketBuy 记录购买/消耗量
func RecordMarketBuy(itemKey string, quantity int) {
	dateStr := time.Now().Format("20060102")
	var sd TgMarketSupplyDemand
	err := DB.Where("item_key = ? AND date_str = ?", itemKey, dateStr).First(&sd).Error
	if err != nil {
		sd = TgMarketSupplyDemand{
			ItemKey:   itemKey,
			DateStr:   dateStr,
			BuyVolume: quantity,
			Timestamp: time.Now().Unix(),
		}
		DB.Create(&sd)
		return
	}
	DB.Model(&TgMarketSupplyDemand{}).Where("id = ?", sd.Id).
		Update("buy_volume", gorm.Expr("buy_volume + ?", quantity))
}

// GetRecentSupplyDemand 获取最近 n 天的供需数据
func GetRecentSupplyDemand(itemKey string, days int) ([]*TgMarketSupplyDemand, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days).Format("20060102")
	var records []*TgMarketSupplyDemand
	err := DB.Where("item_key = ? AND date_str >= ?", itemKey, cutoffDate).
		Order("date_str DESC").Find(&records).Error
	return records, err
}

// GetRecentSupplyDemandAll 获取所有商品最近 n 天的供需数据
func GetRecentSupplyDemandAll(days int) ([]*TgMarketSupplyDemand, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days).Format("20060102")
	var records []*TgMarketSupplyDemand
	err := DB.Where("date_str >= ?", cutoffDate).Find(&records).Error
	return records, err
}

// CleanOldSupplyDemand 清理旧供需数据
func CleanOldSupplyDemand(days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days).Format("20060102")
	return DB.Where("date_str < ?", cutoffDate).Delete(&TgMarketSupplyDemand{}).Error
}
