package model

import "time"

// TgFarmTutorial 新手引导状态
type TgFarmTutorial struct {
	Id              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId      string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex"`
	CurrentStep     int    `json:"current_step" gorm:"default:0"`
	Completed       int    `json:"completed" gorm:"default:0"`
	Skipped         int    `json:"skipped" gorm:"default:0"`
	Version         int    `json:"version" gorm:"default:1"`
	CompletedAt     int64  `json:"completed_at" gorm:"default:0"`
	LastUpdatedAt   int64  `json:"last_updated_at" gorm:"default:0"`
}

func GetTutorialState(tgId string) (*TgFarmTutorial, error) {
	var t TgFarmTutorial
	err := DB.Where("telegram_id = ?", tgId).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func CreateTutorialState(tgId string) (*TgFarmTutorial, error) {
	t := &TgFarmTutorial{
		TelegramId:    tgId,
		CurrentStep:   0,
		Version:       1,
		LastUpdatedAt: time.Now().Unix(),
	}
	err := DB.Create(t).Error
	return t, err
}

func UpdateTutorialStep(tgId string, step int) error {
	return DB.Model(&TgFarmTutorial{}).Where("telegram_id = ?", tgId).
		Updates(map[string]interface{}{
			"current_step":    step,
			"last_updated_at": time.Now().Unix(),
		}).Error
}

func CompleteTutorial(tgId string) error {
	now := time.Now().Unix()
	return DB.Model(&TgFarmTutorial{}).Where("telegram_id = ?", tgId).
		Updates(map[string]interface{}{
			"completed":       1,
			"skipped":         0,
			"completed_at":    now,
			"last_updated_at": now,
		}).Error
}

func SkipTutorial(tgId string) error {
	now := time.Now().Unix()
	return DB.Model(&TgFarmTutorial{}).Where("telegram_id = ?", tgId).
		Updates(map[string]interface{}{
			"completed":       1,
			"skipped":         1,
			"completed_at":    now,
			"last_updated_at": now,
		}).Error
}

func RestartTutorial(tgId string, version int) error {
	return DB.Model(&TgFarmTutorial{}).Where("telegram_id = ?", tgId).
		Updates(map[string]interface{}{
			"current_step":    0,
			"completed":       0,
			"skipped":         0,
			"completed_at":    0,
			"version":         version,
			"last_updated_at": time.Now().Unix(),
		}).Error
}
