package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// 天气事件层（A-2）— 叠加在基础天气之上的稀有一次性事件。
// 基础天气在 controller/web_farm_expansion.go 里按季节节流刷新，
// 本层事件独立：每 5 分钟评估一次是否触发，全服共享同一事件。

// TgFarmWeatherEvent 全服共享的天气事件记录
// 每条为"一次事件的生命周期"，通过 ended 标记是否已结束。
type TgFarmWeatherEvent struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	EventKey  string `json:"event_key" gorm:"type:varchar(32);index"`   // frost / rainbow / heatwave / thunderstorm / spring_fog
	Name      string `json:"name" gorm:"type:varchar(32)"`
	Emoji     string `json:"emoji" gorm:"type:varchar(16)"`
	Severity  int    `json:"severity" gorm:"default:1"`                 // 1 轻 / 2 中 / 3 重
	StartedAt int64  `json:"started_at" gorm:"index"`
	EndsAt    int64  `json:"ends_at" gorm:"index"`
	Ended     int    `json:"ended" gorm:"default:0;index"`               // 0=进行中 1=已结束
	LastTickAt int64 `json:"last_tick_at" gorm:"default:0"`             // 上次每小时 tick 时间
	Narrative string `json:"narrative" gorm:"type:varchar(255)"`        // 给玩家看的一行描述
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime;index"`
}

// GetActiveWeatherEvent 返回当前仍在持续的天气事件；无则返回 nil
func GetActiveWeatherEvent() (*TgFarmWeatherEvent, error) {
	var ev TgFarmWeatherEvent
	now := time.Now().Unix()
	err := DB.Where("ended = 0 AND ends_at > ?", now).
		Order("started_at desc").First(&ev).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ev, nil
}

// GetRecentWeatherEvents 返回最近 N 条事件（含已结束），给前端日志用
func GetRecentWeatherEvents(limit int) ([]*TgFarmWeatherEvent, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var list []*TgFarmWeatherEvent
	err := DB.Order("started_at desc").Limit(limit).Find(&list).Error
	return list, err
}

// CreateWeatherEvent 创建并落库一条事件
func CreateWeatherEvent(ev *TgFarmWeatherEvent) error {
	return DB.Create(ev).Error
}

// MarkWeatherEventEnded 标记事件结束
func MarkWeatherEventEnded(id int) error {
	return DB.Model(&TgFarmWeatherEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"ended":   1,
			"ends_at": time.Now().Unix(),
		}).Error
}

// UpdateWeatherEventTickAt 更新上次 tick 时间
func UpdateWeatherEventTickAt(id int, ts int64) error {
	return DB.Model(&TgFarmWeatherEvent{}).
		Where("id = ?", id).
		Update("last_tick_at", ts).Error
}

// CountWeatherEventsSince 返回最近 since 秒内的事件数，用于节流
func CountWeatherEventsSince(since int64) (int64, error) {
	var cnt int64
	err := DB.Model(&TgFarmWeatherEvent{}).
		Where("started_at >= ?", time.Now().Unix()-since).
		Count(&cnt).Error
	return cnt, err
}
