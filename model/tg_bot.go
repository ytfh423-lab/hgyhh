package model

import (
	"math/rand"

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

// TgBotLotteryPrize 抽奖奖品
type TgBotLotteryPrize struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name" gorm:"type:varchar(100);not null"`
	Code      string `json:"code" gorm:"type:varchar(255);not null"`
	Status    int    `json:"status" gorm:"default:1"` // 1=可用 2=已中奖
	WonBy     string `json:"won_by" gorm:"type:varchar(64)"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
}

// TgBotMessageTracker 群组消息追踪（每用户每群组）
type TgBotMessageTracker struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ChatId       int64  `json:"chat_id" gorm:"uniqueIndex:idx_chat_user"`
	TelegramId   string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_chat_user"`
	MessageCount int    `json:"message_count" gorm:"default:0"`
	LotteryUsed  int    `json:"lottery_used" gorm:"default:0"`
	LastBotMsgId int    `json:"last_bot_msg_id"`
	UpdatedAt    int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TgBotLotteryRecord 抽奖记录
type TgBotLotteryRecord struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index"`
	ChatId     int64  `json:"chat_id"`
	PrizeName  string `json:"prize_name" gorm:"type:varchar(100)"`
	PrizeCode  string `json:"prize_code" gorm:"type:varchar(255)"`
	Won        bool   `json:"won"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
}

// ========== TgBotLotteryPrize 奖品管理 ==========

func GetAllTgBotLotteryPrizes() ([]*TgBotLotteryPrize, error) {
	var prizes []*TgBotLotteryPrize
	err := DB.Order("id asc").Find(&prizes).Error
	return prizes, err
}

func GetAvailableTgBotLotteryPrize() (*TgBotLotteryPrize, error) {
	var prize TgBotLotteryPrize
	orderFunc := "RANDOM()"
	if common.UsingMySQL {
		orderFunc = "RAND()"
	}
	err := DB.Where("status = 1").Order(orderFunc).First(&prize).Error
	return &prize, err
}

func MarkTgBotLotteryPrizeWon(id int, telegramId string) error {
	return DB.Model(&TgBotLotteryPrize{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": 2,
		"won_by": telegramId,
	}).Error
}

func CreateTgBotLotteryPrize(prize *TgBotLotteryPrize) error {
	return DB.Create(prize).Error
}

func AddTgBotLotteryPrizes(codes []string, name string) (int, error) {
	added := 0
	for _, code := range codes {
		if code == "" {
			continue
		}
		item := &TgBotLotteryPrize{
			Name:   name,
			Code:   code,
			Status: 1,
		}
		if err := DB.Create(item).Error; err != nil {
			continue
		}
		added++
	}
	return added, nil
}

func DeleteTgBotLotteryPrize(id int) error {
	return DB.Delete(&TgBotLotteryPrize{}, id).Error
}

func CountTgBotLotteryPrizes() (total int64, available int64, err error) {
	err = DB.Model(&TgBotLotteryPrize{}).Count(&total).Error
	if err != nil {
		return
	}
	err = DB.Model(&TgBotLotteryPrize{}).Where("status = 1").Count(&available).Error
	return
}

// ========== TgBotMessageTracker 消息追踪 ==========

func GetOrCreateMessageTracker(chatId int64, telegramId string) (*TgBotMessageTracker, error) {
	var tracker TgBotMessageTracker
	err := DB.Where("chat_id = ? AND telegram_id = ?", chatId, telegramId).First(&tracker).Error
	if err != nil {
		tracker = TgBotMessageTracker{
			ChatId:     chatId,
			TelegramId: telegramId,
		}
		err = DB.Create(&tracker).Error
	}
	return &tracker, err
}

func IncrementMessageCount(id int) error {
	return DB.Model(&TgBotMessageTracker{}).Where("id = ?", id).
		Update("message_count", gorm.Expr("message_count + 1")).Error
}

func UpdateLastBotMsgId(id int, msgId int) error {
	return DB.Model(&TgBotMessageTracker{}).Where("id = ?", id).
		Update("last_bot_msg_id", msgId).Error
}

func IncrementLotteryUsed(id int) error {
	return DB.Model(&TgBotMessageTracker{}).Where("id = ?", id).
		Update("lottery_used", gorm.Expr("lottery_used + 1")).Error
}

// ========== TgBotLotteryRecord 抽奖记录 ==========

func CreateTgBotLotteryRecord(record *TgBotLotteryRecord) error {
	return DB.Create(record).Error
}

func GetTgBotLotteryRecords(telegramId string) ([]*TgBotLotteryRecord, error) {
	var records []*TgBotLotteryRecord
	err := DB.Where("telegram_id = ? AND won = ?", telegramId, true).Order("id desc").Limit(20).Find(&records).Error
	return records, err
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

// DeleteTgBotClaim 删除领取记录（回滚时使用）
func DeleteTgBotClaim(id int) error {
	return DB.Delete(&TgBotClaim{}, id).Error
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

// DispenseRandomCode 随机取一个未使用的库存码并标记已发放，领取记录尽力写入不影响发放
func DispenseRandomCode(categoryId int, telegramId string) (code string, err error) {
	// 1. 随机取一个可用码
	var items []TgBotInventory
	err = DB.Where("category_id = ? AND status = 1", categoryId).Find(&items).Error
	if err != nil || len(items) == 0 {
		return "", gorm.ErrRecordNotFound
	}
	item := items[rand.Intn(len(items))]

	// 2. 标记已发放（乐观锁）
	result := DB.Model(&TgBotInventory{}).Where("id = ? AND status = 1", item.Id).Updates(map[string]interface{}{
		"status":     2,
		"claimed_by": telegramId,
	})
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 {
		// 被并发抢走，再试一次顺序取
		var fallback TgBotInventory
		if e := DB.Where("category_id = ? AND status = 1", categoryId).First(&fallback).Error; e != nil {
			return "", e
		}
		r2 := DB.Model(&TgBotInventory{}).Where("id = ? AND status = 1", fallback.Id).Updates(map[string]interface{}{
			"status":     2,
			"claimed_by": telegramId,
		})
		if r2.Error != nil || r2.RowsAffected == 0 {
			return "", gorm.ErrRecordNotFound
		}
		item = fallback
	}

	code = item.Code

	// 3. 尽力写入领取记录（失败不影响发放）
	claim := &TgBotClaim{
		TelegramId: telegramId,
		CategoryId: categoryId,
		CodeKey:    item.Code,
	}
	if e := DB.Create(claim).Error; e != nil {
		common.SysError("TG Bot: write claim record failed (code already dispensed): " + e.Error())
	}

	return code, nil
}

// DeleteTgBotInventoryByCategory 删除某分类的所有库存
func DeleteTgBotInventoryByCategory(categoryId int) error {
	return DB.Where("category_id = ?", categoryId).Delete(&TgBotInventory{}).Error
}

// RollbackInventoryCode 回滚库存码为可用状态（私聊发送失败时使用）
func RollbackInventoryCode(id int) error {
	return DB.Model(&TgBotInventory{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     1,
		"claimed_by": "",
	}).Error
}

// ClearTgBotInventoryItem 删除单个库存码
func ClearTgBotInventoryItem(id int) error {
	return DB.Delete(&TgBotInventory{}, id).Error
}
