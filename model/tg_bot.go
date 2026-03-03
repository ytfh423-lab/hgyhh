package model

import (
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

// TgBotInventory 机器人分类库存（每个分类独立的兑换码池）
type TgBotInventory struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	CategoryId int    `json:"category_id" gorm:"index;not null"`
	Code       string `json:"code" gorm:"type:varchar(255);not null"`
	Status     int    `json:"status" gorm:"default:1"` // 1=可用 2=已发放
	ClaimedBy  string `json:"claimed_by" gorm:"type:varchar(64)"` // telegram_id
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
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

// ========== TgBotInventory 库存管理 ==========

// AddTgBotInventoryCodes 批量添加库存码
func AddTgBotInventoryCodes(categoryId int, codes []string) (int, error) {
	added := 0
	for _, code := range codes {
		if code == "" {
			continue
		}
		item := &TgBotInventory{
			CategoryId: categoryId,
			Code:       code,
			Status:     1,
		}
		if err := DB.Create(item).Error; err != nil {
			continue
		}
		added++
	}
	return added, nil
}

// GetTgBotInventoryByCategoryId 获取某分类的所有库存
func GetTgBotInventoryByCategoryId(categoryId int) ([]*TgBotInventory, error) {
	var items []*TgBotInventory
	err := DB.Where("category_id = ?", categoryId).Order("id asc").Find(&items).Error
	return items, err
}

// CountTgBotInventory 统计某分类的库存（总数和可用数）
func CountTgBotInventory(categoryId int) (total int64, available int64, err error) {
	err = DB.Model(&TgBotInventory{}).Where("category_id = ?", categoryId).Count(&total).Error
	if err != nil {
		return
	}
	err = DB.Model(&TgBotInventory{}).Where("category_id = ? AND status = 1", categoryId).Count(&available).Error
	return
}

// CountAllTgBotInventory 批量统计所有分类的库存
func CountAllTgBotInventory() (map[int]map[string]int64, error) {
	type result struct {
		CategoryId int
		Status     int
		Cnt        int64
	}
	var results []result
	err := DB.Model(&TgBotInventory{}).Select("category_id, status, count(*) as cnt").Group("category_id, status").Find(&results).Error
	if err != nil {
		return nil, err
	}
	m := make(map[int]map[string]int64)
	for _, r := range results {
		if m[r.CategoryId] == nil {
			m[r.CategoryId] = map[string]int64{"total": 0, "available": 0}
		}
		m[r.CategoryId]["total"] += r.Cnt
		if r.Status == 1 {
			m[r.CategoryId]["available"] = r.Cnt
		}
	}
	return m, nil
}

// FindAvailableInventoryCode 从库存中查找可用的兑换码
func FindAvailableInventoryCode(categoryId int) (*TgBotInventory, error) {
	var item TgBotInventory
	err := DB.Where("category_id = ? AND status = 1", categoryId).Order("id asc").First(&item).Error
	return &item, err
}

// MarkInventoryCodeDispensed 标记库存码为已发放
func MarkInventoryCodeDispensed(id int, telegramId string) error {
	return DB.Model(&TgBotInventory{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     2,
		"claimed_by": telegramId,
	}).Error
}

// DeleteTgBotInventoryByCategory 删除某分类的所有库存
func DeleteTgBotInventoryByCategory(categoryId int) error {
	return DB.Where("category_id = ?", categoryId).Delete(&TgBotInventory{}).Error
}

// ClearTgBotInventoryItem 删除单个库存码
func ClearTgBotInventoryItem(id int) error {
	return DB.Delete(&TgBotInventory{}, id).Error
}
