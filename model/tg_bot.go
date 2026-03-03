package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// TgBotCategory 机器人领取分类
type TgBotCategory struct {
	Id          int            `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null"`
	Description string         `json:"description" gorm:"type:varchar(255)"`
	MaxClaims   int            `json:"max_claims" gorm:"default:1"`
	Purpose     int            `json:"purpose" gorm:"default:1"` // 1=余额兑换码 2=注册邀请码
	Status      int            `json:"status" gorm:"default:1"`  // 1=启用 2=禁用
	CreatedAt   int64          `json:"created_at" gorm:"autoCreateTime"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TgBotClaim 机器人领取记录
type TgBotClaim struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index"`
	CategoryId int    `json:"category_id" gorm:"index"`
	CodeKey    string `json:"code_key" gorm:"type:varchar(255);index"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
}

// GetAllTgBotCategories 获取所有分类
func GetAllTgBotCategories() ([]*TgBotCategory, error) {
	var categories []*TgBotCategory
	err := DB.Order("id asc").Find(&categories).Error
	return categories, err
}

// GetEnabledTgBotCategories 获取所有启用的分类
func GetEnabledTgBotCategories() ([]*TgBotCategory, error) {
	var categories []*TgBotCategory
	err := DB.Where("status = ?", 1).Order("id asc").Find(&categories).Error
	return categories, err
}

// GetTgBotCategoryById 根据ID获取分类
func GetTgBotCategoryById(id int) (*TgBotCategory, error) {
	var category TgBotCategory
	err := DB.First(&category, id).Error
	return &category, err
}

// CreateTgBotCategory 创建分类
func CreateTgBotCategory(category *TgBotCategory) error {
	return DB.Create(category).Error
}

// UpdateTgBotCategory 更新分类
func UpdateTgBotCategory(category *TgBotCategory) error {
	return DB.Model(category).Select("name", "description", "max_claims", "purpose", "status").Updates(category).Error
}

// DeleteTgBotCategory 删除分类
func DeleteTgBotCategory(id int) error {
	return DB.Delete(&TgBotCategory{}, id).Error
}

// CountTgBotClaims 统计某用户在某分类下的领取次数
func CountTgBotClaims(telegramId string, categoryId int) (int64, error) {
	var count int64
	err := DB.Model(&TgBotClaim{}).Where("telegram_id = ? AND category_id = ?", telegramId, categoryId).Count(&count).Error
	return count, err
}

// CreateTgBotClaim 创建领取记录
func CreateTgBotClaim(claim *TgBotClaim) error {
	return DB.Create(claim).Error
}

// GetTgBotClaimsByTelegramId 获取用户的所有领取记录
func GetTgBotClaimsByTelegramId(telegramId string) ([]*TgBotClaim, error) {
	var claims []*TgBotClaim
	err := DB.Where("telegram_id = ?", telegramId).Order("id desc").Find(&claims).Error
	return claims, err
}

// FindAvailableRedemptionCode 查找可用的兑换码（排除已通过机器人发放的）
func FindAvailableRedemptionCode(purpose int) (*Redemption, error) {
	var code Redemption
	now := common.GetTimestamp()

	// 获取已发放的 code keys
	var dispensedKeys []string
	DB.Model(&TgBotClaim{}).Where("code_key != ''").Pluck("code_key", &dispensedKeys)

	query := DB.Where("purpose = ? AND status = ? AND (expired_time = 0 OR expired_time > ?)",
		purpose, common.RedemptionCodeStatusEnabled, now)
	if len(dispensedKeys) > 0 {
		keyCol := "`key`"
		if common.UsingPostgreSQL {
			keyCol = `"key"`
		}
		query = query.Where(keyCol+" NOT IN ?", dispensedKeys)
	}
	err := query.Order("id asc").First(&code).Error
	return &code, err
}
