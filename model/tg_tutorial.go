package model

import "time"

// TgFarmTutorial 用户总教程状态（保留向后兼容）
type TgFarmTutorial struct {
	Id            int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId    string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex"`
	CurrentStep   int    `json:"current_step" gorm:"default:0"`
	Completed     int    `json:"completed" gorm:"default:0"`
	Skipped       int    `json:"skipped" gorm:"default:0"`
	Version       int    `json:"version" gorm:"default:1"`
	CompletedAt   int64  `json:"completed_at" gorm:"default:0"`
	LastUpdatedAt int64  `json:"last_updated_at" gorm:"default:0"`
}

// TgFarmTutorialState 功能级教程状态表
type TgFarmTutorialState struct {
	Id                 int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId         string `json:"telegram_id" gorm:"type:varchar(64);index:idx_tut_tg_feat,unique"`
	FeatureKey         string `json:"feature_key" gorm:"type:varchar(64);index:idx_tut_tg_feat,unique"`
	TutorialVersion    int    `json:"tutorial_version" gorm:"default:1"`
	TutorialRequired   int    `json:"tutorial_required" gorm:"default:1"`
	TutorialStarted    int    `json:"tutorial_started" gorm:"default:0"`
	TutorialCompleted  int    `json:"tutorial_completed" gorm:"default:0"`
	CurrentStep        int    `json:"current_step" gorm:"default:0"`
	TutorialMode       string `json:"tutorial_mode" gorm:"type:varchar(16);default:'forced'"` // forced / replay
	UnlockTime         int64  `json:"unlock_time" gorm:"default:0"`
	CompletedAt        int64  `json:"completed_at" gorm:"default:0"`
	LastUpdatedAt      int64  `json:"last_updated_at" gorm:"default:0"`
}

// ───── TgFarmTutorialState CRUD ─────

func GetFeatureTutorialState(tgId, featureKey string) (*TgFarmTutorialState, error) {
	var s TgFarmTutorialState
	err := DB.Where("telegram_id = ? AND feature_key = ?", tgId, featureKey).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func GetAllFeatureTutorialStates(tgId string) ([]TgFarmTutorialState, error) {
	var list []TgFarmTutorialState
	err := DB.Where("telegram_id = ?", tgId).Order("unlock_time ASC").Find(&list).Error
	return list, err
}

func EnsureFeatureTutorialState(tgId, featureKey string, version int) (*TgFarmTutorialState, error) {
	s, err := GetFeatureTutorialState(tgId, featureKey)
	if err == nil {
		return s, nil
	}
	now := time.Now().Unix()
	s = &TgFarmTutorialState{
		TelegramId:       tgId,
		FeatureKey:       featureKey,
		TutorialVersion:  version,
		TutorialRequired: 1,
		TutorialMode:     "forced",
		UnlockTime:       now,
		LastUpdatedAt:    now,
	}
	err = DB.Create(s).Error
	return s, err
}

func UpdateFeatureTutorialStep(tgId, featureKey string, step int) error {
	return DB.Model(&TgFarmTutorialState{}).
		Where("telegram_id = ? AND feature_key = ?", tgId, featureKey).
		Updates(map[string]interface{}{
			"current_step":     step,
			"tutorial_started": 1,
			"last_updated_at":  time.Now().Unix(),
		}).Error
}

func CompleteFeatureTutorial(tgId, featureKey string) error {
	now := time.Now().Unix()
	return DB.Model(&TgFarmTutorialState{}).
		Where("telegram_id = ? AND feature_key = ?", tgId, featureKey).
		Updates(map[string]interface{}{
			"tutorial_completed": 1,
			"tutorial_required":  0,
			"completed_at":      now,
			"last_updated_at":   now,
		}).Error
}

func RestartFeatureTutorial(tgId, featureKey string) error {
	return DB.Model(&TgFarmTutorialState{}).
		Where("telegram_id = ? AND feature_key = ?", tgId, featureKey).
		Updates(map[string]interface{}{
			"current_step":      0,
			"tutorial_started":  0,
			"tutorial_completed": 0,
			"tutorial_required": 0,
			"tutorial_mode":     "replay",
			"last_updated_at":   time.Now().Unix(),
		}).Error
}

// GetPendingForcedTutorial 获取第一个未完成的强制教程
func GetPendingForcedTutorial(tgId string) (*TgFarmTutorialState, error) {
	var s TgFarmTutorialState
	err := DB.Where("telegram_id = ? AND tutorial_required = 1 AND tutorial_completed = 0 AND tutorial_mode = 'forced'",
		tgId).Order("unlock_time ASC").First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ───── 旧 TgFarmTutorial 兼容函数（保留，但不再主力使用）─────

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
