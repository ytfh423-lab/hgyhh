package model

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
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
	MaturedAt     int64  `json:"matured_at" gorm:"default:0"`     // 成熟时间戳（用于保护期计算）
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

// TgFarmWarehouse 农场仓库
type TgFarmWarehouse struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_farm_wh"`
	CropType   string `json:"crop_type" gorm:"type:varchar(32);uniqueIndex:idx_farm_wh"`
	Quantity   int    `json:"quantity" gorm:"default:0"`
	Category   string `json:"category" gorm:"type:varchar(16);default:'crop'"`  // crop/fish/meat/recipe
	StoredAt   int64  `json:"stored_at" gorm:"default:0"`                      // 存入时间戳
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

// ========== TgFarmWarehouse ==========

// CleanSpoiledWarehouse 清理过期物品（根据仓库等级调整保质期）
func CleanSpoiledWarehouse(telegramId string) {
	now := time.Now().Unix()
	whLevel := GetWarehouseLevel(telegramId)
	multiplier := int64(GetWarehouseExpiryMultiplier(whLevel))
	meatExpiry := int64(common.TgBotFarmWarehouseMeatExpiry) * multiplier / 100
	recipeExpiry := int64(common.TgBotFarmWarehouseRecipeExpiry) * multiplier / 100
	// 删除过期肉类
	DB.Where("telegram_id = ? AND category = ? AND stored_at > 0 AND stored_at + ? < ?", telegramId, "meat", meatExpiry, now).Delete(&TgFarmWarehouse{})
	// 删除过期加工品
	DB.Where("telegram_id = ? AND category = ? AND stored_at > 0 AND stored_at + ? < ?", telegramId, "recipe", recipeExpiry, now).Delete(&TgFarmWarehouse{})
}

// GetWarehouseItems 获取仓库所有物品（自动清理过期物品）
func GetWarehouseItems(telegramId string) ([]*TgFarmWarehouse, error) {
	CleanSpoiledWarehouse(telegramId)
	var items []*TgFarmWarehouse
	err := DB.Where("telegram_id = ? AND quantity > 0", telegramId).Find(&items).Error
	return items, err
}

// GetWarehouseItem 获取仓库中某种作物数量
func GetWarehouseItem(telegramId, cropType string) (*TgFarmWarehouse, error) {
	var item TgFarmWarehouse
	err := DB.Where("telegram_id = ? AND crop_type = ?", telegramId, cropType).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// AddToWarehouse 添加物品到仓库（category: crop/fish/meat/recipe）
func AddToWarehouse(telegramId, cropType string, quantity int) error {
	return AddToWarehouseWithCategory(telegramId, cropType, quantity, "crop")
}

// AddToWarehouseWithCategory 添加物品到仓库（指定分类）
func AddToWarehouseWithCategory(telegramId, cropType string, quantity int, category string) error {
	var item TgFarmWarehouse
	err := DB.Where("telegram_id = ? AND crop_type = ?", telegramId, cropType).First(&item).Error
	if err != nil {
		// 不存在则创建
		item = TgFarmWarehouse{TelegramId: telegramId, CropType: cropType, Quantity: quantity, Category: category, StoredAt: time.Now().Unix()}
		return DB.Create(&item).Error
	}
	// 已存在则增加数量，不更新StoredAt（保留最早存入时间）
	return DB.Model(&TgFarmWarehouse{}).Where("id = ?", item.Id).Update("quantity", item.Quantity+quantity).Error
}

// RemoveFromWarehouse 从仓库取出作物
func RemoveFromWarehouse(telegramId, cropType string, quantity int) error {
	var item TgFarmWarehouse
	err := DB.Where("telegram_id = ? AND crop_type = ?", telegramId, cropType).First(&item).Error
	if err != nil {
		return err
	}
	if item.Quantity < quantity {
		return fmt.Errorf("仓库数量不足")
	}
	newQty := item.Quantity - quantity
	if newQty <= 0 {
		return DB.Delete(&item).Error
	}
	return DB.Model(&TgFarmWarehouse{}).Where("id = ?", item.Id).Update("quantity", newQty).Error
}

// GetWarehouseTotalCount 获取仓库总存储数量
func GetWarehouseTotalCount(telegramId string) int {
	var total int64
	DB.Model(&TgFarmWarehouse{}).Where("telegram_id = ?", telegramId).Select("COALESCE(SUM(quantity),0)").Scan(&total)
	return int(total)
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

func GetFarmItemQuantity(telegramId string, itemType string) (int, error) {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, itemType).First(&item).Error
	if err != nil {
		return 0, err
	}
	return item.Quantity, nil
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

// ========== 等级 ==========

func GetFarmLevel(telegramId string) int {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_level").First(&item).Error
	if err != nil || item.Quantity < 1 {
		return 1
	}
	return item.Quantity
}

func SetFarmLevel(telegramId string, level int) {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_level").First(&item).Error
	if err != nil {
		item = TgFarmItem{TelegramId: telegramId, ItemType: "_level", Quantity: level}
		_ = DB.Create(&item).Error
		return
	}
	_ = DB.Model(&TgFarmItem{}).Where("id = ?", item.Id).Update("quantity", level).Error
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

func DeleteFarmProcess(id int) error {
	return DB.Delete(&TgFarmProcess{}, id).Error
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

// 钓鱼体力系统：惰性恢复模型

func getFarmItemInt(telegramId, key string) int {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, key).First(&item).Error
	if err != nil {
		return 0
	}
	return item.Quantity
}

func setFarmItemInt(telegramId, key string, val int) {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, key).First(&item).Error
	if err != nil {
		item = TgFarmItem{TelegramId: telegramId, ItemType: key, Quantity: val}
		_ = DB.Create(&item).Error
		return
	}
	_ = DB.Model(&TgFarmItem{}).Where("id = ?", item.Id).Update("quantity", val).Error
}

// GetFishStamina 获取当前体力（惰性恢复），返回当前体力和下次恢复剩余秒数
func GetFishStamina(telegramId string) (current int, recoverIn int64) {
	saved := getFarmItemInt(telegramId, "_fish_stamina")
	lastTs := int64(getFarmItemInt(telegramId, "_fish_stamina_ts"))
	now := time.Now().Unix()
	max := common.TgBotFishStaminaMax
	interval := int64(common.TgBotFishStaminaRecoverInterval)
	amount := common.TgBotFishStaminaRecoverAmount

	// 新用户（无记录）给满体力
	if lastTs == 0 {
		return max, 0
	}

	elapsed := now - lastTs
	if interval > 0 && amount > 0 {
		recovered := int(elapsed/interval) * amount
		current = saved + recovered
	} else {
		current = saved
	}
	if current >= max {
		current = max
		recoverIn = 0
	} else if interval > 0 {
		// 下次恢复剩余秒数
		recoverIn = interval - (elapsed % interval)
	}
	return current, recoverIn
}

// SetFishStamina 设置体力和时间戳
func SetFishStamina(telegramId string, stamina int) {
	setFarmItemInt(telegramId, "_fish_stamina", stamina)
	setFarmItemInt(telegramId, "_fish_stamina_ts", int(time.Now().Unix()))
}

// 钓鱼每日统计（自动跨天重置）

func fishTodayStr() string {
	return time.Now().Format("20060102")
}

// ResetFishDailyIfNeeded 检查是否跨天，跨天则重置每日计数
func ResetFishDailyIfNeeded(telegramId string) {
	savedDate := getFarmItemInt(telegramId, "_fish_daily_date")
	today, _ := strconv.Atoi(fishTodayStr())
	if savedDate != today {
		setFarmItemInt(telegramId, "_fish_daily_date", today)
		setFarmItemInt(telegramId, "_fish_daily_count", 0)
		setFarmItemInt(telegramId, "_fish_daily_income", 0)
	}
}

func GetFishDailyCount(telegramId string) int {
	ResetFishDailyIfNeeded(telegramId)
	return getFarmItemInt(telegramId, "_fish_daily_count")
}

func IncrFishDailyCount(telegramId string) {
	ResetFishDailyIfNeeded(telegramId)
	cur := getFarmItemInt(telegramId, "_fish_daily_count")
	setFarmItemInt(telegramId, "_fish_daily_count", cur+1)
}

func GetFishDailyIncome(telegramId string) int {
	ResetFishDailyIfNeeded(telegramId)
	return getFarmItemInt(telegramId, "_fish_daily_income")
}

func IncrFishDailyIncome(telegramId string, amount int) {
	ResetFishDailyIfNeeded(telegramId)
	cur := getFarmItemInt(telegramId, "_fish_daily_income")
	setFarmItemInt(telegramId, "_fish_daily_income", cur+amount)
}

func GetFishItems(telegramId string) ([]*TgFarmItem, error) {
	var items []*TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type LIKE ? AND item_type != ? AND quantity > 0", telegramId, "fish_%", "fishbait").Find(&items).Error
	return items, err
}

func SellAllFish(telegramId string) (int, error) {
	result := DB.Model(&TgFarmItem{}).
		Where("telegram_id = ? AND item_type LIKE ? AND item_type != ? AND quantity > 0", telegramId, "fish_%", "fishbait").
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

// ========== 银行贷款 ==========

type TgFarmLoan struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId   string `json:"telegram_id" gorm:"type:varchar(64);index"`
	Principal    int    `json:"principal"`                    // 本金(quota)
	Interest     int    `json:"interest"`                     // 利息(quota)
	TotalDue     int    `json:"total_due"`                    // 应还总额
	Repaid       int    `json:"repaid" gorm:"default:0"`      // 已还金额
	Status       int    `json:"status" gorm:"default:0"`      // 0=未还清 1=已还清 2=违约
	LoanType     int    `json:"loan_type" gorm:"default:0"`   // 0=普通贷款 1=抵押贷款
	CreditScore  int    `json:"credit_score"`                 // 贷款时的信用评分
	DueAt        int64  `json:"due_at"`                       // 到期时间
	CreatedAt    int64  `json:"created_at"`
}

// GetActiveLoan 获取用户当前未还清贷款
func GetActiveLoan(telegramId string) (*TgFarmLoan, error) {
	var loan TgFarmLoan
	err := DB.Where("telegram_id = ? AND status = 0", telegramId).First(&loan).Error
	if err != nil {
		return nil, err
	}
	return &loan, nil
}

// GetCreditScore 根据消费记录计算信用评分(1~maxMultiplier)
func GetCreditScore(telegramId string) int {
	// 统计最近30天的正向收入记录数量和总额
	thirtyDaysAgo := time.Now().Unix() - 30*86400
	var count int64
	var totalIncome int64

	// 正向操作次数
	DB.Model(&TgFarmLog{}).Where("telegram_id = ? AND created_at > ? AND amount > 0", telegramId, thirtyDaysAgo).Count(&count)

	// 正向收入总额
	type sumResult struct {
		Total int64
	}
	var sr sumResult
	DB.Model(&TgFarmLog{}).Select("COALESCE(SUM(amount),0) as total").Where("telegram_id = ? AND created_at > ? AND amount > 0", telegramId, thirtyDaysAgo).Scan(&sr)
	totalIncome = sr.Total

	// 检查历史贷款记录：是否有逾期未还
	var overdueLoanCount int64
	now := time.Now().Unix()
	DB.Model(&TgFarmLoan{}).Where("telegram_id = ? AND status = 0 AND due_at < ?", telegramId, now).Count(&overdueLoanCount)

	// 已还清的贷款数量加分
	var repaidCount int64
	DB.Model(&TgFarmLoan{}).Where("telegram_id = ? AND status = 1", telegramId).Count(&repaidCount)

	// 评分算法: 基础1分 + 活跃度 + 收入 + 信用历史 - 逾期扣分
	score := 1
	// 活跃度: 每10次操作+1分, 最多+3
	activityBonus := int(count / 10)
	if activityBonus > 3 {
		activityBonus = 3
	}
	score += activityBonus

	// 收入: 每5000000(=$10)+1分, 最多+3
	incomeBonus := int(totalIncome / 5000000)
	if incomeBonus > 3 {
		incomeBonus = 3
	}
	score += incomeBonus

	// 信用历史: 每次还清+1, 最多+2
	historyBonus := int(repaidCount)
	if historyBonus > 2 {
		historyBonus = 2
	}
	score += historyBonus

	// 等级加分
	level := GetFarmLevel(telegramId)
	levelBonus := (level - 1) / 3 // 每3级+1
	if levelBonus > 2 {
		levelBonus = 2
	}
	score += levelBonus

	// 逾期扣分
	score -= int(overdueLoanCount) * 3

	if score < 1 {
		score = 1
	}
	maxMul := 10
	if score > maxMul {
		score = maxMul
	}
	return score
}

// CreateLoan 创建贷款 (loanType: 0=普通, 1=抵押)
func CreateLoan(telegramId string, principal, interest, totalDue int, creditScore int, dueDays int) (*TgFarmLoan, error) {
	return CreateLoanWithType(telegramId, principal, interest, totalDue, creditScore, dueDays, 0)
}

// CreateLoanWithType 创建指定类型贷款
func CreateLoanWithType(telegramId string, principal, interest, totalDue int, creditScore int, dueDays int, loanType int) (*TgFarmLoan, error) {
	now := time.Now().Unix()
	loan := &TgFarmLoan{
		TelegramId:  telegramId,
		Principal:   principal,
		Interest:    interest,
		TotalDue:    totalDue,
		Repaid:      0,
		Status:      0,
		LoanType:    loanType,
		CreditScore: creditScore,
		DueAt:       now + int64(dueDays)*86400,
		CreatedAt:   now,
	}
	err := DB.Create(loan).Error
	return loan, err
}

// RepayLoan 还款（部分或全部）
func RepayLoan(loanId int, amount int) (*TgFarmLoan, error) {
	var loan TgFarmLoan
	err := DB.Where("id = ? AND status = 0", loanId).First(&loan).Error
	if err != nil {
		return nil, errors.New("未找到待还贷款")
	}
	remaining := loan.TotalDue - loan.Repaid
	if amount > remaining {
		amount = remaining
	}
	loan.Repaid += amount
	if loan.Repaid >= loan.TotalDue {
		loan.Status = 1
	}
	err = DB.Model(&TgFarmLoan{}).Where("id = ?", loanId).Updates(map[string]interface{}{
		"repaid": loan.Repaid,
		"status": loan.Status,
	}).Error
	return &loan, err
}

// GetLoanHistory 获取贷款历史
func GetLoanHistory(telegramId string, limit int) ([]*TgFarmLoan, error) {
	var loans []*TgFarmLoan
	err := DB.Where("telegram_id = ?", telegramId).Order("id desc").Limit(limit).Find(&loans).Error
	return loans, err
}

// HasActiveMortgageLoan 检查是否有未还清的抵押贷款
func HasActiveMortgageLoan(telegramId string) bool {
	var count int64
	DB.Model(&TgFarmLoan{}).Where("telegram_id = ? AND loan_type = 1 AND status = 0", telegramId).Count(&count)
	return count > 0
}

// HasMortgageBlocked 检查是否被永久禁止升级到10级+
func HasMortgageBlocked(telegramId string) bool {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_mortgage_blocked").First(&item).Error
	if err != nil {
		return false
	}
	return item.Quantity > 0
}

// SetMortgageBlocked 设置永久禁止升级到10级+
func SetMortgageBlocked(telegramId string) {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_mortgage_blocked").First(&item).Error
	if err != nil {
		DB.Create(&TgFarmItem{TelegramId: telegramId, ItemType: "_mortgage_blocked", Quantity: 1})
	} else {
		DB.Model(&TgFarmItem{}).Where("telegram_id = ? AND item_type = ?", telegramId, "_mortgage_blocked").Update("quantity", 1)
	}
}

// BanUserByTelegramId 通过tgId封禁用户平台账号
func BanUserByTelegramId(telegramId string) error {
	var user User
	err := DB.Where("telegram_id = ?", telegramId).First(&user).Error
	if err != nil {
		return err
	}
	return DB.Model(&User{}).Where("id = ?", user.Id).Update("status", common.UserStatusDisabled).Error
}

// CheckMortgageDefault 检查抵押贷款是否违约，执行惩罚
// 返回: defaulted bool, penalty string
func CheckMortgageDefault(telegramId string) (bool, string) {
	var loans []TgFarmLoan
	now := time.Now().Unix()
	// 找到所有逾期的抵押贷款
	DB.Where("telegram_id = ? AND loan_type = 1 AND status = 0 AND due_at < ?", telegramId, now).Find(&loans)
	if len(loans) == 0 {
		return false, ""
	}

	level := GetFarmLevel(telegramId)
	for _, loan := range loans {
		// 标记为违约
		DB.Model(&TgFarmLoan{}).Where("id = ?", loan.Id).Update("status", 2)

		if level >= 10 {
			// 10级以上：封禁平台账号
			_ = BanUserByTelegramId(telegramId)
			AddFarmLog(telegramId, "mortgage_default", 0, "抵押贷款违约-账号封禁")
			return true, "ban"
		} else {
			// 10级以下：永久禁止升级到10级+
			SetMortgageBlocked(telegramId)
			AddFarmLog(telegramId, "mortgage_default", 0, "抵押贷款违约-永久禁止10级")
			return true, "block_level"
		}
	}
	return false, ""
}

// CheckCreditLoanDefault 检查信用贷款是否逾期，执行封禁
// 返回: defaulted bool, penalty string
func CheckCreditLoanDefault(telegramId string) (bool, string) {
	var loans []TgFarmLoan
	now := time.Now().Unix()
	// 找到所有逾期的信用贷款（type=0）
	DB.Where("telegram_id = ? AND loan_type = 0 AND status = 0 AND due_at < ?", telegramId, now).Find(&loans)
	if len(loans) == 0 {
		return false, ""
	}

	for _, loan := range loans {
		// 标记为违约
		DB.Model(&TgFarmLoan{}).Where("id = ?", loan.Id).Update("status", 2)
	}

	// 信用贷款逾期直接封禁账号
	_ = BanUserByTelegramId(telegramId)
	AddFarmLog(telegramId, "credit_default", 0, "信用贷款逾期违约-账号封禁")
	return true, "ban"
}

// ========== 仓库等级系统 ==========

// GetWarehouseLevel 获取用户仓库等级（最低1）
func GetWarehouseLevel(telegramId string) int {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_warehouse_level").First(&item).Error
	if err != nil || item.Quantity < 1 {
		return 1
	}
	return item.Quantity
}

// SetWarehouseLevel 设置用户仓库等级
func SetWarehouseLevel(telegramId string, level int) error {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_warehouse_level").First(&item).Error
	if err != nil {
		return DB.Create(&TgFarmItem{TelegramId: telegramId, ItemType: "_warehouse_level", Quantity: level}).Error
	}
	return DB.Model(&TgFarmItem{}).Where("telegram_id = ? AND item_type = ?", telegramId, "_warehouse_level").Update("quantity", level).Error
}

// GetWarehouseMaxSlots 根据等级计算仓库最大容量
func GetWarehouseMaxSlots(level int) int {
	base := common.TgBotFarmWarehouseMaxSlots
	perLevel := common.TgBotFarmWarehouseCapacityPerLevel
	return base + (level-1)*perLevel
}

// GetWarehouseExpiryMultiplier 根据等级计算保质期倍率（百分比，100=不变，150=1.5倍）
func GetWarehouseExpiryMultiplier(level int) int {
	bonus := common.TgBotFarmWarehouseExpiryBonusPerLevel
	return 100 + (level-1)*bonus
}

// ========== 管理员功能 ==========

// ActiveFarmUser 活跃农场用户信息
type ActiveFarmUser struct {
	FarmId      string  `json:"farm_id"`
	UserId      int     `json:"user_id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	TotalPlots  int     `json:"total_plots"`
	ActivePlots int     `json:"active_plots"`
	MaturePlots int     `json:"mature_plots"`
	FarmLevel   int     `json:"farm_level"`
	Balance     float64 `json:"balance"`
}

// GetActiveFarmUsers 获取所有真正在玩农场的用户（有非空地块）
func GetActiveFarmUsers() ([]ActiveFarmUser, error) {
	// 1. 获取所有有地块的 distinct telegram_id
	type plotStat struct {
		TelegramId  string
		Total       int
		Active      int
		Mature      int
	}
	var stats []plotStat
	err := DB.Model(&TgFarmPlot{}).
		Select("telegram_id, COUNT(*) as total, SUM(CASE WHEN status > 0 THEN 1 ELSE 0 END) as active, SUM(CASE WHEN status = 2 THEN 1 ELSE 0 END) as mature").
		Group("telegram_id").
		Having("SUM(CASE WHEN status > 0 THEN 1 ELSE 0 END) > 0").
		Find(&stats).Error
	if err != nil {
		return nil, err
	}

	var result []ActiveFarmUser
	for _, s := range stats {
		u := ActiveFarmUser{
			FarmId:      s.TelegramId,
			TotalPlots:  s.Total,
			ActivePlots: s.Active,
			MaturePlots: s.Mature,
			FarmLevel:   GetFarmLevel(s.TelegramId),
		}
		// 尝试关联 User 表
		var user User
		if strings.HasPrefix(s.TelegramId, "u_") {
			idStr := strings.TrimPrefix(s.TelegramId, "u_")
			uid, _ := strconv.Atoi(idStr)
			if uid > 0 {
				if e := DB.Select("id, username, display_name, quota").Where("id = ?", uid).First(&user).Error; e == nil {
					u.UserId = user.Id
					u.Username = user.Username
					u.DisplayName = user.DisplayName
					u.Balance = float64(user.Quota) / 500000.0
				}
			}
		} else {
			if e := DB.Select("id, username, display_name, quota").Where("telegram_id = ?", s.TelegramId).First(&user).Error; e == nil {
				u.UserId = user.Id
				u.Username = user.Username
				u.DisplayName = user.DisplayName
				u.Balance = float64(user.Quota) / 500000.0
			}
		}
		result = append(result, u)
	}
	return result, nil
}

// ResetNegativeBalanceUsers 将所有余额为负数的用户重置为0
func ResetNegativeBalanceUsers() (int64, error) {
	result := DB.Model(&User{}).Where("quota < 0").Update("quota", 0)
	return result.RowsAffected, result.Error
}

// ResetAllFarmLevels 将所有用户的农场等级重置为指定等级
func ResetAllFarmLevels(level int) (int64, error) {
	result := DB.Model(&TgFarmItem{}).Where("item_type = ?", "_level").Update("quantity", level)
	return result.RowsAffected, result.Error
}

func migrateFarmIDColumnTx(tx *gorm.DB, table any, column, fromFarmID, toFarmID string) error {
	return tx.Model(table).Where(column+" = ?", fromFarmID).Update(column, toFarmID).Error
}

// BindTelegramAndMigrateFarmData binds telegram_id to user and migrates legacy web farm id data.
// Legacy web farm id is usually in the form u_{userId}.
func BindTelegramAndMigrateFarmData(userId int, oldFarmID, newTelegramID string) error {
	if newTelegramID == "" {
		return fmt.Errorf("telegram id is empty")
	}
	if oldFarmID == "" || oldFarmID == newTelegramID {
		return DB.Model(&User{}).Where("id = ?", userId).Update("telegram_id", newTelegramID).Error
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&User{}).Where("id = ?", userId).Update("telegram_id", newTelegramID).Error; err != nil {
			return err
		}

		// Core farm data (telegram_id).
		coreTables := []any{
			&TgFarmPlot{}, &TgFarmItem{}, &TgFarmDog{}, &TgFarmWarehouse{},
			&TgRanchAnimal{}, &TgFarmTaskClaim{}, &TgFarmAchievement{}, &TgFarmProcess{},
			&TgFarmLog{}, &TgFarmLoan{}, &TgFarmCollection{}, &TgFarmPrestige{},
			&TgFarmGameLog{}, &TgFarmAutomation{}, &TgTreeSlot{},
		}
		for _, table := range coreTables {
			if err := migrateFarmIDColumnTx(tx, table, "telegram_id", oldFarmID, newTelegramID); err != nil {
				return err
			}
		}

		// Farm related references not named telegram_id.
		if err := migrateFarmIDColumnTx(tx, &TgFarmStealLog{}, "thief_id", oldFarmID, newTelegramID); err != nil {
			return err
		}
		if err := migrateFarmIDColumnTx(tx, &TgFarmStealLog{}, "victim_id", oldFarmID, newTelegramID); err != nil {
			return err
		}
		if err := migrateFarmIDColumnTx(tx, &TgFarmTrade{}, "seller_id", oldFarmID, newTelegramID); err != nil {
			return err
		}
		if err := migrateFarmIDColumnTx(tx, &TgFarmTrade{}, "buyer_id", oldFarmID, newTelegramID); err != nil {
			return err
		}
		if err := migrateFarmIDColumnTx(tx, &TgFarmEntrust{}, "owner_telegram_id", oldFarmID, newTelegramID); err != nil {
			return err
		}
		if err := migrateFarmIDColumnTx(tx, &TgFarmEntrustWorker{}, "worker_telegram_id", oldFarmID, newTelegramID); err != nil {
			return err
		}
		if err := migrateFarmIDColumnTx(tx, &TgFarmEntrustLog{}, "worker_telegram_id", oldFarmID, newTelegramID); err != nil {
			return err
		}
		if err := migrateFarmIDColumnTx(tx, &TgFarmEntrustEscrow{}, "owner_telegram_id", oldFarmID, newTelegramID); err != nil {
			return err
		}
		if err := migrateFarmIDColumnTx(tx, &TgFarmEntrustEscrow{}, "worker_telegram_id", oldFarmID, newTelegramID); err != nil {
			return err
		}
		return nil
	})
}

// ========== 图鉴收藏 ==========

type TgFarmCollection struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_farm_coll"`
	Category   string `json:"category" gorm:"type:varchar(16);uniqueIndex:idx_farm_coll"`
	ItemKey    string `json:"item_key" gorm:"type:varchar(32);uniqueIndex:idx_farm_coll"`
	Quantity   int    `json:"quantity" gorm:"default:0"`
	FirstAt    int64  `json:"first_at"`
}

func RecordCollection(telegramId, category, itemKey string, qty int) {
	var item TgFarmCollection
	err := DB.Where("telegram_id = ? AND category = ? AND item_key = ?", telegramId, category, itemKey).First(&item).Error
	if err != nil {
		DB.Create(&TgFarmCollection{TelegramId: telegramId, Category: category, ItemKey: itemKey, Quantity: qty, FirstAt: time.Now().Unix()})
		return
	}
	DB.Model(&TgFarmCollection{}).Where("id = ?", item.Id).Update("quantity", item.Quantity+qty)
}

func GetCollections(telegramId string) ([]*TgFarmCollection, error) {
	var items []*TgFarmCollection
	err := DB.Where("telegram_id = ?", telegramId).Find(&items).Error
	return items, err
}

func HasCollectionReward(telegramId, category string) bool {
	return HasAchievement(telegramId, "_coll_"+category)
}

func ClaimCollectionReward(telegramId, category string) error {
	return UnlockAchievement(telegramId, "_coll_"+category)
}

// ========== 玩家交易 ==========

type TgFarmTrade struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	SellerId     string `json:"seller_id" gorm:"type:varchar(64);index"`
	SellerName   string `json:"seller_name" gorm:"type:varchar(64)"`
	Category     string `json:"category" gorm:"type:varchar(16)"`
	ItemKey      string `json:"item_key" gorm:"type:varchar(32)"`
	ItemName     string `json:"item_name" gorm:"type:varchar(32)"`
	ItemEmoji    string `json:"item_emoji" gorm:"type:varchar(16)"`
	Quantity     int    `json:"quantity"`
	PricePerUnit int    `json:"price_per_unit"`
	Status       int    `json:"status" gorm:"default:0"`
	BuyerId      string `json:"buyer_id" gorm:"type:varchar(64)"`
	CreatedAt    int64  `json:"created_at"`
}

func GetOpenTrades(limit, offset int) ([]*TgFarmTrade, int64, error) {
	var trades []*TgFarmTrade
	var total int64
	DB.Model(&TgFarmTrade{}).Where("status = 0").Count(&total)
	err := DB.Where("status = 0").Order("id desc").Limit(limit).Offset(offset).Find(&trades).Error
	return trades, total, err
}

func CountMyOpenTrades(telegramId string) int64 {
	var count int64
	DB.Model(&TgFarmTrade{}).Where("seller_id = ? AND status = 0", telegramId).Count(&count)
	return count
}

func CreateTrade(trade *TgFarmTrade) error {
	trade.CreatedAt = time.Now().Unix()
	return DB.Create(trade).Error
}

func GetTradeById(id int) (*TgFarmTrade, error) {
	var trade TgFarmTrade
	err := DB.Where("id = ?", id).First(&trade).Error
	return &trade, err
}

func UpdateTradeStatus(id int, status int, buyerId string) error {
	updates := map[string]interface{}{"status": status}
	if buyerId != "" {
		updates["buyer_id"] = buyerId
	}
	return DB.Model(&TgFarmTrade{}).Where("id = ?", id).Updates(updates).Error
}

func GetTradeHistory(telegramId string, limit int) ([]*TgFarmTrade, error) {
	var trades []*TgFarmTrade
	err := DB.Where("(seller_id = ? OR buyer_id = ?) AND status != 0", telegramId, telegramId).
		Order("id desc").Limit(limit).Find(&trades).Error
	return trades, err
}

// ========== 转生系统 ==========

type TgFarmPrestige struct {
	Id            int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId    string `json:"telegram_id" gorm:"type:varchar(64);index"`
	PrestigeLevel int    `json:"prestige_level"`
	PrestigedAt   int64  `json:"prestiged_at"`
}

func GetPrestigeLevel(telegramId string) int {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_prestige").First(&item).Error
	if err != nil || item.Quantity < 1 {
		return 0
	}
	return item.Quantity
}

func SetPrestigeLevel(telegramId string, level int) {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_prestige").First(&item).Error
	if err != nil {
		DB.Create(&TgFarmItem{TelegramId: telegramId, ItemType: "_prestige", Quantity: level})
		return
	}
	DB.Model(&TgFarmItem{}).Where("id = ?", item.Id).Update("quantity", level)
}

func CreatePrestigeRecord(telegramId string, level int) {
	DB.Create(&TgFarmPrestige{TelegramId: telegramId, PrestigeLevel: level, PrestigedAt: time.Now().Unix()})
}

func ResetFarmForPrestige(userId int, telegramId string) {
	DB.Where("telegram_id = ?", telegramId).Delete(&TgFarmPlot{})
	DB.Where("telegram_id = ? AND item_type NOT IN ('_level','_prestige','_mortgage_blocked','_last_fish')", telegramId).Delete(&TgFarmItem{})
	DB.Where("telegram_id = ?", telegramId).Delete(&TgFarmWarehouse{})
	DB.Where("telegram_id = ?", telegramId).Delete(&TgFarmDog{})
	DB.Where("telegram_id = ? AND status IN (1,2)", telegramId).Delete(&TgFarmProcess{})
	DB.Where("telegram_id = ?", telegramId).Delete(&TgRanchAnimal{})
	DB.Where("telegram_id = ?", telegramId).Delete(&TgFarmAutomation{})
	DB.Where("seller_id = ? OR buyer_id = ?", telegramId, telegramId).Delete(&TgFarmTrade{})
	if userId > 0 {
		DB.Model(&User{}).Where("id = ?", userId).Update("quota", 5000000)
	}
	SetFarmLevel(telegramId, 1)
}

func GetPrestigePrice(nextPrestigeLevel int) int {
	if nextPrestigeLevel < 1 {
		nextPrestigeLevel = 1
	}
	idx := nextPrestigeLevel - 1
	if idx >= 0 && idx < len(common.TgBotFarmPrestigePrices) {
		return common.TgBotFarmPrestigePrices[idx]
	}
	if len(common.TgBotFarmPrestigePrices) == 0 {
		return 0
	}
	return common.TgBotFarmPrestigePrices[len(common.TgBotFarmPrestigePrices)-1]
}

// ========== 小游戏记录 ==========

type TgFarmGameLog struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index"`
	GameType   string `json:"game_type" gorm:"type:varchar(16)"`
	BetAmount  int    `json:"bet_amount"`
	WinAmount  int    `json:"win_amount"`
	CreatedAt  int64  `json:"created_at"`
}

func CreateGameLog(telegramId, gameType string, bet, win int) {
	DB.Create(&TgFarmGameLog{
		TelegramId: telegramId, GameType: gameType,
		BetAmount: bet, WinAmount: win, CreatedAt: time.Now().Unix(),
	})
}

func GetRecentGameLogs(telegramId string, limit int) ([]*TgFarmGameLog, error) {
	var logs []*TgFarmGameLog
	err := DB.Where("telegram_id = ?", telegramId).Order("id desc").Limit(limit).Find(&logs).Error
	return logs, err
}

// ========== 自动化设施 ==========

type TgFarmAutomation struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId  string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_farm_auto"`
	Type        string `json:"type" gorm:"type:varchar(32);uniqueIndex:idx_farm_auto"`
	Level       int    `json:"level" gorm:"default:1"`
	InstalledAt int64  `json:"installed_at"`
}

func GetAutomations(telegramId string) ([]*TgFarmAutomation, error) {
	var items []*TgFarmAutomation
	err := DB.Where("telegram_id = ?", telegramId).Find(&items).Error
	return items, err
}

func HasAutomation(telegramId, autoType string) bool {
	var count int64
	DB.Model(&TgFarmAutomation{}).Where("telegram_id = ? AND type = ?", telegramId, autoType).Count(&count)
	return count > 0
}

func CreateAutomation(telegramId, autoType string) error {
	return DB.Create(&TgFarmAutomation{
		TelegramId: telegramId, Type: autoType, Level: 1, InstalledAt: time.Now().Unix(),
	}).Error
}

// ========== 排行榜 ==========

type FarmLeaderboardEntry struct {
	TelegramId string
	Username   string
	Value      int64
}

func GetFarmLeaderboard(boardType string, limit int) ([]FarmLeaderboardEntry, error) {
	var entries []FarmLeaderboardEntry
	switch boardType {
	case "balance":
		err := DB.Raw("SELECT telegram_id, username, quota as value FROM users WHERE telegram_id != '' AND status = 1 ORDER BY quota DESC LIMIT ?", limit).Scan(&entries).Error
		return entries, err
	case "level":
		err := DB.Raw("SELECT fi.telegram_id, u.username, fi.quantity as value FROM tg_farm_items fi JOIN users u ON fi.telegram_id = u.telegram_id WHERE fi.item_type = '_level' ORDER BY fi.quantity DESC LIMIT ?", limit).Scan(&entries).Error
		return entries, err
	case "harvest":
		err := DB.Raw("SELECT fl.telegram_id, u.username, COUNT(*) as value FROM tg_farm_logs fl JOIN users u ON fl.telegram_id = u.telegram_id WHERE fl.action = 'harvest' GROUP BY fl.telegram_id, u.username ORDER BY value DESC LIMIT ?", limit).Scan(&entries).Error
		return entries, err
	case "prestige":
		err := DB.Raw("SELECT fi.telegram_id, u.username, fi.quantity as value FROM tg_farm_items fi JOIN users u ON fi.telegram_id = u.telegram_id WHERE fi.item_type = '_prestige' AND fi.quantity > 0 ORDER BY fi.quantity DESC LIMIT ?", limit).Scan(&entries).Error
		return entries, err
	default:
		return entries, nil
	}
}

func GetFarmRank(telegramId, boardType string) int64 {
	var rank int64
	switch boardType {
	case "balance":
		var userQuota int64
		DB.Model(&User{}).Select("quota").Where("telegram_id = ?", telegramId).Scan(&userQuota)
		DB.Model(&User{}).Where("telegram_id != '' AND status = 1 AND quota > ?", userQuota).Count(&rank)
	case "level":
		level := GetFarmLevel(telegramId)
		DB.Model(&TgFarmItem{}).Where("item_type = '_level' AND quantity > ?", level).Count(&rank)
	}
	return rank + 1
}

func GetFarmLogDetails(telegramId string, actions []string) []string {
	var details []string
	DB.Model(&TgFarmLog{}).Select("DISTINCT detail").
		Where("telegram_id = ? AND action IN ?", telegramId, actions).
		Pluck("detail", &details)
	return details
}

func GetFarmLogs(telegramId string, limit, offset int) ([]*TgFarmLog, int64, error) {
	var logs []*TgFarmLog
	var total int64
	DB.Model(&TgFarmLog{}).Where("telegram_id = ?", telegramId).Count(&total)
	err := DB.Where("telegram_id = ?", telegramId).Order("id desc").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}

// ========== 内测数据清理 ==========

// CleanupAllBetaFarmData 清理所有内测农场数据并回收用户净收益额度
// 返回: 清理的用户数, 回收的总额度, error
func CleanupAllBetaFarmData() (int, int64, error) {
	// 1. 获取所有参与农场的用户（从 farm_logs 汇总净收益）
	type userEarning struct {
		TelegramId string
		NetEarning int64
	}
	var earnings []userEarning
	err := DB.Model(&TgFarmLog{}).
		Select("telegram_id, COALESCE(SUM(amount), 0) as net_earning").
		Group("telegram_id").
		Having("COALESCE(SUM(amount), 0) > 0").
		Scan(&earnings).Error
	if err != nil {
		return 0, 0, fmt.Errorf("查询用户收益失败: %w", err)
	}

	// 2. 回收每个用户的净收益额度
	var totalReclaimed int64
	for _, e := range earnings {
		if e.NetEarning <= 0 {
			continue
		}
		// 通过 telegram_id 找到用户并扣减额度
		var user User
		findErr := DB.Where("telegram_id = ?", e.TelegramId).First(&user).Error
		if findErr != nil {
			continue
		}
		// 扣减额度，最多扣到0
		reclaimAmount := e.NetEarning
		if int64(user.Quota) < reclaimAmount {
			reclaimAmount = int64(user.Quota)
		}
		if reclaimAmount > 0 {
			DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", gorm.Expr("CASE WHEN quota >= ? THEN quota - ? ELSE 0 END", reclaimAmount, reclaimAmount))
			totalReclaimed += reclaimAmount
		}
	}

	// 3. 删除所有农场相关数据（按表逐个清空）
	DB.Where("1 = 1").Delete(&TgFarmPlot{})
	DB.Where("1 = 1").Delete(&TgFarmItem{})
	DB.Where("1 = 1").Delete(&TgFarmStealLog{})
	DB.Where("1 = 1").Delete(&TgFarmDog{})
	DB.Where("1 = 1").Delete(&TgRanchAnimal{})
	DB.Where("1 = 1").Delete(&TgFarmLog{})
	DB.Where("1 = 1").Delete(&TgFarmProcess{})
	DB.Where("1 = 1").Delete(&TgFarmTaskClaim{})
	DB.Where("1 = 1").Delete(&TgFarmAchievement{})
	DB.Where("1 = 1").Delete(&TgFarmLoan{})
	DB.Where("1 = 1").Delete(&TgFarmWarehouse{})
	DB.Where("1 = 1").Delete(&TgFarmCollection{})
	DB.Where("1 = 1").Delete(&TgFarmTrade{})
	DB.Where("1 = 1").Delete(&TgFarmPrestige{})
	DB.Where("1 = 1").Delete(&TgFarmGameLog{})
	DB.Where("1 = 1").Delete(&TgFarmAutomation{})
	DB.Where("1 = 1").Delete(&TgTreeSlot{})

	// 4. 重置所有内测预约的协议接受状态
	DB.Model(&FarmBetaReservation{}).Where("agreement_accepted_at > 0").Update("agreement_accepted_at", 0)

	return len(earnings), totalReclaimed, nil
}
