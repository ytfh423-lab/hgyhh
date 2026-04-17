package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// 突发事件中心（A-3）— 每个玩家独立的叙事性小剧情。
// 每 12 小时最多推一条；玩家做出选择后事件立即结算并归档。
//
// 与 A-2（TgFarmWeatherEvent 全服共享）的区别：
//   - A-2：全服一条，被动承受土壤 patch
//   - A-3：每个玩家一条，主动做选择，结果直接结算到玩家资产

type TgFarmRandomEvent struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TgId       string `json:"tg_id" gorm:"type:varchar(64);index"`
	EventKey   string `json:"event_key" gorm:"type:varchar(32);index"` // beggar / merchant / old_farmer / thief ...
	Title      string `json:"title" gorm:"type:varchar(64)"`
	Emoji      string `json:"emoji" gorm:"type:varchar(16)"`
	Narrative  string `json:"narrative" gorm:"type:varchar(512)"`
	OptionsRaw string `json:"options_raw" gorm:"type:text"`             // JSON 序列化的 []OptionDef
	ChosenIdx  int    `json:"chosen_idx" gorm:"default:-1"`             // -1=未选 0/1/2=已选
	Outcome    string `json:"outcome" gorm:"type:varchar(255)"`         // 结果文案
	StartedAt  int64  `json:"started_at" gorm:"autoCreateTime;index"`
	ExpiresAt  int64  `json:"expires_at" gorm:"index"`                  // 过期未选视作放弃
	ResolvedAt int64  `json:"resolved_at" gorm:"default:0"`
}

// GetPendingRandomEvent 返回玩家当前未结算的事件；无返回 nil
func GetPendingRandomEvent(tgId string) (*TgFarmRandomEvent, error) {
	var ev TgFarmRandomEvent
	err := DB.Where("tg_id = ? AND chosen_idx = -1 AND expires_at > ?",
		tgId, time.Now().Unix()).
		Order("started_at desc").First(&ev).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ev, nil
}

// GetRecentRandomEvents 返回玩家最近 N 条事件历史（已结算）
func GetRecentRandomEvents(tgId string, limit int) ([]*TgFarmRandomEvent, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var list []*TgFarmRandomEvent
	err := DB.Where("tg_id = ? AND chosen_idx >= 0", tgId).
		Order("resolved_at desc").Limit(limit).Find(&list).Error
	return list, err
}

// GetRecentRandomEventsAll 管理员视图：返回最近 N 条事件（所有玩家，含未结算）
func GetRecentRandomEventsAll(limit int) ([]*TgFarmRandomEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var list []*TgFarmRandomEvent
	err := DB.Order("started_at desc").Limit(limit).Find(&list).Error
	return list, err
}

// CreateRandomEvent 落库一条新事件
func CreateRandomEvent(ev *TgFarmRandomEvent) error {
	return DB.Create(ev).Error
}

// ResolveRandomEvent 结算：写入选择、结果文案、结算时间
func ResolveRandomEvent(id int, chosenIdx int, outcome string) error {
	return DB.Model(&TgFarmRandomEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"chosen_idx":  chosenIdx,
			"outcome":     outcome,
			"resolved_at": time.Now().Unix(),
		}).Error
}

// CountRandomEventsSince 给节流判定用：玩家最近 since 秒内触发过几次
func CountRandomEventsSince(tgId string, since int64) (int64, error) {
	var cnt int64
	err := DB.Model(&TgFarmRandomEvent{}).
		Where("tg_id = ? AND started_at >= ?", tgId, time.Now().Unix()-since).
		Count(&cnt).Error
	return cnt, err
}
