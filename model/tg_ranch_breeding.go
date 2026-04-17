package model

type TgRanchBreeding struct {
	Id                 int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId         string `json:"telegram_id" gorm:"type:varchar(64);index"`
	ParentAId          int    `json:"parent_a_id" gorm:"index"`
	ParentBId          int    `json:"parent_b_id" gorm:"index"`
	AnimalType         string `json:"animal_type" gorm:"type:varchar(32);index"`
	Status             int    `json:"status" gorm:"default:1;index"`
	StartedAt          int64  `json:"started_at"`
	DueAt              int64  `json:"due_at" gorm:"index"`
	CompletedAt        int64  `json:"completed_at"`
	ClaimedAt          int64  `json:"claimed_at"`
	ParentAQuality     int    `json:"parent_a_quality" gorm:"default:1"`
	ParentBQuality     int    `json:"parent_b_quality" gorm:"default:1"`
	OffspringQuality   int    `json:"offspring_quality" gorm:"default:1"`
	OffspringGeneration int   `json:"offspring_generation" gorm:"default:1"`
	Cost               int    `json:"cost" gorm:"default:0"`
}

func GetRanchBreedings(telegramId string) ([]*TgRanchBreeding, error) {
	var breedings []*TgRanchBreeding
	err := DB.Where("telegram_id = ?", telegramId).
		Order("status asc, due_at asc, id desc").
		Find(&breedings).Error
	return breedings, err
}

func CountUnclaimedRanchBreedings(telegramId string) (int64, error) {
	var count int64
	err := DB.Model(&TgRanchBreeding{}).
		Where("telegram_id = ? AND status IN ?", telegramId, []int{1, 2}).
		Count(&count).Error
	return count, err
}

func GetRanchBreedingByIdAndTelegramId(id int, telegramId string) (*TgRanchBreeding, error) {
	var breeding TgRanchBreeding
	err := DB.Where("id = ? AND telegram_id = ?", id, telegramId).First(&breeding).Error
	if err != nil {
		return nil, err
	}
	return &breeding, nil
}

func CreateRanchBreeding(breeding *TgRanchBreeding) error {
	return DB.Create(breeding).Error
}

func UpdateRanchBreeding(breeding *TgRanchBreeding) error {
	return DB.Save(breeding).Error
}

func IsRecordNotFound(err error) bool {
	return err != nil && err == gorm.ErrRecordNotFound
}
