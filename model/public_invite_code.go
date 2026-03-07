package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type PublicInviteCode struct {
	Id          int            `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int            `json:"user_id" gorm:"index"`
	Username    string         `json:"username" gorm:"type:varchar(64)"`
	Code        string         `json:"code" gorm:"type:varchar(64);uniqueIndex"`
	CodeId      int            `json:"code_id" gorm:"index"`
	Status      int            `json:"status" gorm:"default:1"` // 1=available 2=used 3=expired
	CreatedAt   int64          `json:"created_at" gorm:"autoCreateTime"`
	ExpiredTime int64          `json:"expired_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func GetPublicInviteCodes() ([]*PublicInviteCode, error) {
	var codes []*PublicInviteCode
	err := DB.Order("id desc").Find(&codes).Error
	return codes, err
}

func GetPublicInviteCodesPaginated(page, pageSize int) ([]*PublicInviteCode, int64, error) {
	var total int64
	err := DB.Model(&PublicInviteCode{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	var codes []*PublicInviteCode
	offset := (page - 1) * pageSize
	err = DB.Order("id desc").Offset(offset).Limit(pageSize).Find(&codes).Error
	return codes, total, err
}

func CreatePublicInviteCode(code *PublicInviteCode) error {
	return DB.Create(code).Error
}

func DeletePublicInviteCode(id int, userId int) error {
	result := DB.Where("id = ? AND user_id = ?", id, userId).Delete(&PublicInviteCode{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func CountUserPublicInviteCodes(userId int) (int64, error) {
	var count int64
	err := DB.Model(&PublicInviteCode{}).Where("user_id = ?", userId).Count(&count).Error
	return count, err
}

func IsCodeAlreadyShared(code string) (bool, error) {
	var count int64
	err := DB.Model(&PublicInviteCode{}).Where("code = ?", code).Count(&count).Error
	return count > 0, err
}

// RefreshPublicInviteCodeStatuses checks redemption table and updates statuses
func RefreshPublicInviteCodeStatuses() error {
	var codes []*PublicInviteCode
	err := DB.Where("status = ?", 1).Find(&codes).Error
	if err != nil {
		return err
	}
	now := common.GetTimestamp()
	for _, c := range codes {
		// Check if expired
		if c.ExpiredTime > 0 && now >= c.ExpiredTime {
			DB.Model(c).Update("status", 3)
			continue
		}
		// Check if used in redemption table
		var redemption Redemption
		result := DB.Where("id = ? AND purpose = ?", c.CodeId, common.RedemptionPurposeRegistration).First(&redemption)
		if result.Error == nil {
			if redemption.Status == common.RedemptionCodeStatusUsed {
				DB.Model(c).Update("status", 2)
			} else if redemption.Status == common.RedemptionCodeStatusDisabled {
				DB.Model(c).Update("status", 3)
			} else if redemption.ExpiredTime > 0 && now >= redemption.ExpiredTime {
				DB.Model(c).Update("status", 3)
			}
		} else {
			// Redemption deleted
			DB.Model(c).Update("status", 3)
		}
	}
	return nil
}
