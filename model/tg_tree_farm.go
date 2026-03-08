package model

import (
	"gorm.io/gorm"
)

// TgTreeSlot 树场树位
type TgTreeSlot struct {
	Id              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId      string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_tree_slot"`
	SlotIndex       int    `json:"slot_index" gorm:"uniqueIndex:idx_tree_slot"`
	TreeType        string `json:"tree_type" gorm:"type:varchar(32)"`
	Status          int    `json:"status" gorm:"default:0"` // 0=empty 1=growing 2=mature 3=stump
	PlantedAt       int64  `json:"planted_at"`
	LastHarvestedAt int64  `json:"last_harvested_at" gorm:"default:0"`
	LastWateredAt   int64  `json:"last_watered_at" gorm:"default:0"`
	Fertilized      int    `json:"fertilized" gorm:"default:0"` // 0=未施肥 1=已施肥
	HarvestCount    int    `json:"harvest_count" gorm:"default:0"`
	StumpAt         int64  `json:"stump_at" gorm:"default:0"` // 伐木后树桩开始时间
}

const TreeFarmInitialSlots = 2
const TreeFarmMaxSlots = 8

// ========== TgTreeSlot CRUD ==========

func GetOrCreateTreeSlots(telegramId string) ([]*TgTreeSlot, error) {
	var slots []*TgTreeSlot
	err := DB.Where("telegram_id = ?", telegramId).Order("slot_index asc").Find(&slots).Error
	if err != nil {
		return nil, err
	}
	if len(slots) >= TreeFarmInitialSlots {
		return slots, nil
	}
	// 初始化树位
	for i := len(slots); i < TreeFarmInitialSlots; i++ {
		slot := &TgTreeSlot{
			TelegramId: telegramId,
			SlotIndex:  i,
			Status:     0,
		}
		if err := DB.Create(slot).Error; err != nil {
			continue
		}
		slots = append(slots, slot)
	}
	return slots, nil
}

func GetTreeSlot(telegramId string, slotIndex int) (*TgTreeSlot, error) {
	var slot TgTreeSlot
	err := DB.Where("telegram_id = ? AND slot_index = ?", telegramId, slotIndex).First(&slot).Error
	return &slot, err
}

func UpdateTreeSlot(slot *TgTreeSlot) error {
	return DB.Save(slot).Error
}

func CreateNewTreeSlot(telegramId string, slotIndex int) error {
	slot := &TgTreeSlot{
		TelegramId: telegramId,
		SlotIndex:  slotIndex,
		Status:     0,
	}
	return DB.Create(slot).Error
}

func GetTreeSlotCount(telegramId string) (int64, error) {
	var count int64
	err := DB.Model(&TgTreeSlot{}).Where("telegram_id = ?", telegramId).Count(&count).Error
	return count, err
}

func ClearTreeSlot(slot *TgTreeSlot) error {
	return DB.Model(&TgTreeSlot{}).Where("id = ?", slot.Id).Updates(map[string]interface{}{
		"tree_type":        "",
		"status":           0,
		"planted_at":       0,
		"last_harvested_at": 0,
		"last_watered_at":  0,
		"fertilized":       0,
		"harvest_count":    0,
		"stump_at":         0,
	}).Error
}

// PlantTree 种植树苗
func PlantTree(slot *TgTreeSlot, treeType string, now int64) error {
	return DB.Model(&TgTreeSlot{}).Where("id = ?", slot.Id).Updates(map[string]interface{}{
		"tree_type":         treeType,
		"status":            1,
		"planted_at":        now,
		"last_harvested_at": 0,
		"last_watered_at":   0,
		"fertilized":        0,
		"harvest_count":     0,
		"stump_at":          0,
	}).Error
}

// WaterTree 浇水
func WaterTree(slotId int, now int64) error {
	return DB.Model(&TgTreeSlot{}).Where("id = ?", slotId).Update("last_watered_at", now).Error
}

// FertilizeTree 施肥
func FertilizeTree(slotId int) error {
	return DB.Model(&TgTreeSlot{}).Where("id = ?", slotId).Update("fertilized", 1).Error
}

// HarvestTree 采收果实（重复采集型树木）
func HarvestTree(slotId int, now int64) error {
	return DB.Model(&TgTreeSlot{}).Where("id = ?", slotId).Updates(map[string]interface{}{
		"last_harvested_at": now,
		"harvest_count":     gorm.Expr("harvest_count + 1"),
	}).Error
}

// ChopTree 伐木，设置为树桩状态
func ChopTree(slotId int, now int64) error {
	return DB.Model(&TgTreeSlot{}).Where("id = ?", slotId).Updates(map[string]interface{}{
		"status":   3,
		"stump_at": now,
	}).Error
}
