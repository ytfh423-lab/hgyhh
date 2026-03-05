package model

import (
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"
)

// TgFarmPlot 农场地块
type TgFarmPlot struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId  string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_farm_plot"`
	PlotIndex   int    `json:"plot_index" gorm:"uniqueIndex:idx_farm_plot"`
	CropType    string `json:"crop_type" gorm:"type:varchar(32)"`
	PlantedAt   int64  `json:"planted_at"`
	Status      int    `json:"status" gorm:"default:0"` // 0=empty 1=growing 2=mature 3=event
	EventType   string `json:"event_type" gorm:"type:varchar(32)"`
	EventAt     int64  `json:"event_at"`
	StolenCount   int    `json:"stolen_count" gorm:"default:0"`
	Fertilized    int    `json:"fertilized" gorm:"default:0"`    // 0=未施肥 1=已施肥
	LastWateredAt int64  `json:"last_watered_at" gorm:"default:0"` // 上次浇水时间
	SoilLevel     int    `json:"soil_level" gorm:"default:1"`     // 泥土等级 1-5
}

// TgFarmItem 农场道具背包
type TgFarmItem struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_farm_item"`
	ItemType   string `json:"item_type" gorm:"type:varchar(32);uniqueIndex:idx_farm_item"`
	Quantity   int    `json:"quantity" gorm:"default:0"`
}

// TgFarmDog 农场看门狗
type TgFarmDog struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex"`
	Name       string `json:"name" gorm:"type:varchar(32)"`
	Level      int    `json:"level" gorm:"default:1"`      // 1=幼犬 2=成犬
	Hunger     int    `json:"hunger" gorm:"default:100"`   // 0-100 饥饿度
	LastFedAt  int64  `json:"last_fed_at"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
}

// TgFarmStealLog 偷菜记录
type TgFarmStealLog struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ThiefId   string `json:"thief_id" gorm:"type:varchar(64);index"`
	VictimId  string `json:"victim_id" gorm:"type:varchar(64);index"`
	PlotId    int    `json:"plot_id"`
	Amount    int    `json:"amount"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
}

const FarmInitialPlots = 2  // 初始地块数
const FarmMaxPlots = 12     // 最大可购买地块数

// ========== TgFarmPlot ==========

func GetOrCreateFarmPlots(telegramId string) ([]*TgFarmPlot, error) {
	var plots []*TgFarmPlot
	err := DB.Where("telegram_id = ?", telegramId).Order("plot_index asc").Find(&plots).Error
	if err != nil {
		return nil, err
	}
	if len(plots) >= FarmInitialPlots {
		return plots, nil
	}
	existing := make(map[int]bool)
	for _, p := range plots {
		existing[p.PlotIndex] = true
	}
	for i := 0; i < FarmInitialPlots; i++ {
		if !existing[i] {
			plot := &TgFarmPlot{TelegramId: telegramId, PlotIndex: i, Status: 0, SoilLevel: 1}
			if err := DB.Create(plot).Error; err != nil {
				return nil, err
			}
			plots = append(plots, plot)
		}
	}
	sort.Slice(plots, func(i, j int) bool { return plots[i].PlotIndex < plots[j].PlotIndex })
	return plots, nil
}

// GetFarmPlotCount 获取用户当前地块数量
func GetFarmPlotCount(telegramId string) (int64, error) {
	var count int64
	err := DB.Model(&TgFarmPlot{}).Where("telegram_id = ?", telegramId).Count(&count).Error
	return count, err
}

// CreateNewFarmPlot 创建新地块（购买）
func CreateNewFarmPlot(telegramId string, plotIndex int) error {
	plot := &TgFarmPlot{TelegramId: telegramId, PlotIndex: plotIndex, Status: 0, SoilLevel: 1}
	return DB.Create(plot).Error
}

// UpgradeFarmPlotSoil 升级地块泥土等级
func UpgradeFarmPlotSoil(plotId int, newLevel int) error {
	return DB.Model(&TgFarmPlot{}).Where("id = ?", plotId).Update("soil_level", newLevel).Error
}

func UpdateFarmPlot(plot *TgFarmPlot) error {
	return DB.Save(plot).Error
}

func ClearFarmPlot(id int) error {
	return DB.Model(&TgFarmPlot{}).Where("id = ?", id).Updates(map[string]interface{}{
		"crop_type": "", "planted_at": 0, "status": 0,
		"event_type": "", "event_at": 0, "stolen_count": 0, "fertilized": 0,
		"last_watered_at": 0,
	}).Error
}

// FarmStealTarget 偷菜目标
type FarmStealTarget struct {
	TelegramId string
	Count      int64
}

func GetMatureFarmTargets(excludeId string) ([]FarmStealTarget, error) {
	var results []FarmStealTarget
	err := DB.Model(&TgFarmPlot{}).
		Select("telegram_id, count(*) as count").
		Where("telegram_id != ? AND status = 2 AND stolen_count < ?", excludeId, 2).
		Group("telegram_id").
		Scan(&results).Error
	return results, err
}

func GetStealablePlots(victimId string) ([]*TgFarmPlot, error) {
	var plots []*TgFarmPlot
	err := DB.Where("telegram_id = ? AND status = 2 AND stolen_count < ?", victimId, 2).
		Find(&plots).Error
	return plots, err
}

func IncrementPlotStolenCount(plotId int) error {
	return DB.Model(&TgFarmPlot{}).Where("id = ?", plotId).
		Update("stolen_count", gorm.Expr("stolen_count + 1")).Error
}

// ========== TgFarmItem ==========

func GetFarmItems(telegramId string) ([]*TgFarmItem, error) {
	var items []*TgFarmItem
	err := DB.Where("telegram_id = ? AND quantity > 0", telegramId).Find(&items).Error
	return items, err
}

func IncrementFarmItem(telegramId string, itemType string, qty int) error {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, itemType).First(&item).Error
	if err != nil {
		item = TgFarmItem{TelegramId: telegramId, ItemType: itemType, Quantity: qty}
		return DB.Create(&item).Error
	}
	return DB.Model(&TgFarmItem{}).Where("id = ?", item.Id).
		Update("quantity", gorm.Expr("quantity + ?", qty)).Error
}

func DecrementFarmItem(telegramId string, itemType string) error {
	result := DB.Model(&TgFarmItem{}).
		Where("telegram_id = ? AND item_type = ? AND quantity > 0", telegramId, itemType).
		Update("quantity", gorm.Expr("quantity - 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("道具不足")
	}
	return nil
}

// ========== TgFarmStealLog ==========

func CreateFarmStealLog(log *TgFarmStealLog) error {
	return DB.Create(log).Error
}

func CountRecentSteals(thiefId, victimId string, sinceUnix int64) (int64, error) {
	var count int64
	err := DB.Model(&TgFarmStealLog{}).
		Where("thief_id = ? AND victim_id = ? AND created_at > ?", thiefId, victimId, sinceUnix).
		Count(&count).Error
	return count, err
}

// ========== TgFarmDog ==========

func GetFarmDog(telegramId string) (*TgFarmDog, error) {
	var dog TgFarmDog
	err := DB.Where("telegram_id = ?", telegramId).First(&dog).Error
	return &dog, err
}

func CreateFarmDog(dog *TgFarmDog) error {
	return DB.Create(dog).Error
}

func UpdateFarmDog(dog *TgFarmDog) error {
	return DB.Save(dog).Error
}

// UpdateDogHunger 懒更新狗的饥饿度（每小时-1）并检查是否升级
func UpdateDogHunger(dog *TgFarmDog) bool {
	now := time.Now().Unix()
	changed := false

	// 计算自上次喂食以来过了多少小时
	if dog.LastFedAt > 0 {
		hoursPassed := int((now - dog.LastFedAt) / 3600)
		newHunger := 100 - hoursPassed
		if newHunger < 0 {
			newHunger = 0
		}
		if newHunger != dog.Hunger {
			dog.Hunger = newHunger
			changed = true
		}
	}

	// 幼犬升级为成犬
	if dog.Level == 1 && dog.Hunger > 0 {
		hoursSinceCreation := (now - dog.CreatedAt) / 3600
		if int(hoursSinceCreation) >= getDogGrowHours() {
			dog.Level = 2
			changed = true
		}
	}

	if changed {
		_ = UpdateFarmDog(dog)
	}
	return changed
}

func getDogGrowHours() int {
	// 从 common 包读取，避免循环导入用延迟求值
	return 24 // 会被 controller 层覆盖调用
}

// FeedFarmDog 喂狗，重置饥饿度
func FeedFarmDog(dogId int) error {
	now := time.Now().Unix()
	return DB.Model(&TgFarmDog{}).Where("id = ?", dogId).Updates(map[string]interface{}{
		"hunger":      100,
		"last_fed_at": now,
	}).Error
}

// WaterFarmPlot 浇水
func WaterFarmPlot(plotId int) error {
	now := time.Now().Unix()
	return DB.Model(&TgFarmPlot{}).Where("id = ?", plotId).Update("last_watered_at", now).Error
}

// ========== TgRanchAnimal 牧场动物 ==========

type TgRanchAnimal struct {
	Id            int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId    string `json:"telegram_id" gorm:"type:varchar(64);index"`
	AnimalType    string `json:"animal_type" gorm:"type:varchar(32)"`
	Status        int    `json:"status" gorm:"default:1"` // 1=growing 2=mature 3=hungry 4=thirsty 5=dead
	PurchasedAt   int64  `json:"purchased_at"`
	LastFedAt     int64  `json:"last_fed_at"`
	LastWateredAt int64  `json:"last_watered_at"`
	LastCleanedAt int64  `json:"last_cleaned_at"`
}

const RanchMaxAnimals = 6

func GetRanchAnimals(telegramId string) ([]*TgRanchAnimal, error) {
	var animals []*TgRanchAnimal
	err := DB.Where("telegram_id = ?", telegramId).Order("id asc").Find(&animals).Error
	return animals, err
}

func GetRanchAnimalCount(telegramId string) (int64, error) {
	var count int64
	err := DB.Model(&TgRanchAnimal{}).Where("telegram_id = ?", telegramId).Count(&count).Error
	return count, err
}

func CreateRanchAnimal(animal *TgRanchAnimal) error {
	return DB.Create(animal).Error
}

func UpdateRanchAnimal(animal *TgRanchAnimal) error {
	return DB.Save(animal).Error
}

func DeleteRanchAnimal(animalId int) error {
	return DB.Delete(&TgRanchAnimal{}, animalId).Error
}

func FeedRanchAnimal(animalId int) error {
	now := time.Now().Unix()
	return DB.Model(&TgRanchAnimal{}).Where("id = ?", animalId).Update("last_fed_at", now).Error
}

func WaterRanchAnimal(animalId int) error {
	now := time.Now().Unix()
	return DB.Model(&TgRanchAnimal{}).Where("id = ?", animalId).Update("last_watered_at", now).Error
}

func CleanRanchAnimals(telegramId string) error {
	now := time.Now().Unix()
	return DB.Model(&TgRanchAnimal{}).Where("telegram_id = ? AND status != 5", telegramId).Update("last_cleaned_at", now).Error
}

// ========== 每日任务 & 成就 ==========

type TgFarmTaskClaim struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index"`
	TaskDate   string `json:"task_date" gorm:"type:varchar(10)"`
	TaskIndex  int    `json:"task_index"`
}

type TgFarmAchievement struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId     string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_achieve"`
	AchievementKey string `json:"achievement_key" gorm:"type:varchar(32);uniqueIndex:idx_achieve"`
	UnlockedAt     int64  `json:"unlocked_at"`
}

func GetTaskClaims(telegramId, taskDate string) ([]int, error) {
	var claims []*TgFarmTaskClaim
	err := DB.Where("telegram_id = ? AND task_date = ?", telegramId, taskDate).Find(&claims).Error
	if err != nil {
		return nil, err
	}
	var indices []int
	for _, c := range claims {
		indices = append(indices, c.TaskIndex)
	}
	return indices, nil
}

func ClaimTask(telegramId, taskDate string, taskIndex int) error {
	return DB.Create(&TgFarmTaskClaim{
		TelegramId: telegramId,
		TaskDate:   taskDate,
		TaskIndex:  taskIndex,
	}).Error
}

func CountTodayActions(telegramId, action string) int64 {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	var count int64
	DB.Model(&TgFarmLog{}).Where("telegram_id = ? AND action = ? AND created_at >= ?", telegramId, action, startOfDay).Count(&count)
	return count
}

func CountTotalActions(telegramId, action string) int64 {
	var count int64
	DB.Model(&TgFarmLog{}).Where("telegram_id = ? AND action = ?", telegramId, action).Count(&count)
	return count
}

func GetAchievements(telegramId string) ([]*TgFarmAchievement, error) {
	var achs []*TgFarmAchievement
	err := DB.Where("telegram_id = ?", telegramId).Find(&achs).Error
	return achs, err
}

func HasAchievement(telegramId, key string) bool {
	var count int64
	DB.Model(&TgFarmAchievement{}).Where("telegram_id = ? AND achievement_key = ?", telegramId, key).Count(&count)
	return count > 0
}

func UnlockAchievement(telegramId, key string) error {
	return DB.Create(&TgFarmAchievement{
		TelegramId:     telegramId,
		AchievementKey: key,
		UnlockedAt:     time.Now().Unix(),
	}).Error
}

// ========== 加工坊 ==========

type TgFarmProcess struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index"`
	RecipeKey  string `json:"recipe_key" gorm:"type:varchar(32)"`
	StartedAt  int64  `json:"started_at"`
	FinishAt   int64  `json:"finish_at"`
	Status     int    `json:"status" gorm:"default:1"` // 1=processing, 2=done, 3=collected
}

const FarmMaxProcessSlots = 3

func GetFarmProcesses(telegramId string) ([]*TgFarmProcess, error) {
	var procs []*TgFarmProcess
	err := DB.Where("telegram_id = ? AND status IN (1,2)", telegramId).Order("id asc").Find(&procs).Error
	return procs, err
}

func CreateFarmProcess(p *TgFarmProcess) error {
	return DB.Create(p).Error
}

func CollectFarmProcess(id int) error {
	return DB.Model(&TgFarmProcess{}).Where("id = ?", id).Update("status", 3).Error
}

func CountActiveProcesses(telegramId string) int64 {
	var count int64
	DB.Model(&TgFarmProcess{}).Where("telegram_id = ? AND status IN (1,2)", telegramId).Count(&count)
	return count
}

// ========== 钓鱼相关 ==========

func GetLastFishTime(telegramId string) int64 {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_last_fish").First(&item).Error
	if err != nil {
		return 0
	}
	return int64(item.Quantity)
}

func SetLastFishTime(telegramId string, ts int64) {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_last_fish").First(&item).Error
	if err != nil {
		item = TgFarmItem{TelegramId: telegramId, ItemType: "_last_fish", Quantity: int(ts)}
		_ = DB.Create(&item).Error
		return
	}
	_ = DB.Model(&TgFarmItem{}).Where("id = ?", item.Id).Update("quantity", int(ts)).Error
}

func GetFishItems(telegramId string) ([]*TgFarmItem, error) {
	var items []*TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type LIKE ? AND quantity > 0", telegramId, "fish_%").Find(&items).Error
	return items, err
}

func SellAllFish(telegramId string) (int, error) {
	result := DB.Model(&TgFarmItem{}).
		Where("telegram_id = ? AND item_type LIKE ? AND quantity > 0", telegramId, "fish_%").
		Update("quantity", 0)
	return int(result.RowsAffected), result.Error
}

// ========== TgFarmLog 消费记录 ==========

type TgFarmLog struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index"`
	Action     string `json:"action" gorm:"type:varchar(32)"`
	Amount     int    `json:"amount"`
	Detail     string `json:"detail" gorm:"type:varchar(255)"`
	CreatedAt  int64  `json:"created_at"`
}

func AddFarmLog(telegramId, action string, amount int, detail string) {
	log := &TgFarmLog{
		TelegramId: telegramId,
		Action:     action,
		Amount:     amount,
		Detail:     detail,
		CreatedAt:  time.Now().Unix(),
	}
	_ = DB.Create(log).Error
}

func GetFarmLogs(telegramId string, limit, offset int) ([]*TgFarmLog, int64, error) {
	var logs []*TgFarmLog
	var total int64
	DB.Model(&TgFarmLog{}).Where("telegram_id = ?", telegramId).Count(&total)
	err := DB.Where("telegram_id = ?", telegramId).Order("id desc").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}
