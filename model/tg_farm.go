package model

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type farmIntCacheEntry struct {
	Value     int
	ExpiresAt int64
}

type farmBoolMapCacheEntry struct {
	Value     map[string]bool
	ExpiresAt int64
}

type farmActionCountCacheEntry struct {
	Value     map[string]int64
	Version   int64
	ExpiresAt int64
}

type farmTaskClaimsCacheEntry struct {
	Value     []int
	ExpiresAt int64
}

type farmLeaderboardCacheEntry struct {
	Value     []FarmLeaderboardEntry
	ExpiresAt int64
}

type farmRankCacheEntry struct {
	Value     int64
	ExpiresAt int64
}

type farmCreditScoreStatsCacheEntry struct {
	Value     FarmCreditScoreStats
	ExpiresAt int64
}

const (
	farmLevelCacheTTLSeconds       int64 = 30
	farmWarehouseCacheTTLSeconds   int64 = 30
	farmAutomationCacheTTLSeconds  int64 = 30
	farmActionCacheTTLSeconds      int64 = 10
	farmTaskClaimsCacheTTLSeconds  int64 = 10
	farmLeaderboardTTLSeconds      int64 = 15
	farmRankTTLSeconds             int64 = 15
	farmCreditScoreTTLSeconds      int64 = 20
)

var farmLevelCache sync.Map
var farmWarehouseLevelCache sync.Map
var farmAutomationCache sync.Map
var farmActionCountCache sync.Map
var farmTaskClaimsCache sync.Map
var farmLeaderboardCache sync.Map
var farmRankCache sync.Map
var farmCreditScoreStatsCache sync.Map
var farmActionVersionLock sync.RWMutex
var farmActionVersions = make(map[string]int64)

func cloneFarmBoolMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return map[string]bool{}
	}
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneFarmActionCountMap(src map[string]int64) map[string]int64 {
	if len(src) == 0 {
		return map[string]int64{}
	}
	dst := make(map[string]int64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func getFarmActionVersion(telegramId string) int64 {
	farmActionVersionLock.RLock()
	defer farmActionVersionLock.RUnlock()
	return farmActionVersions[telegramId]
}

func bumpFarmActionVersion(telegramId string) {
	farmActionVersionLock.Lock()
	farmActionVersions[telegramId]++
	farmActionVersionLock.Unlock()
}

func invalidateFarmLeaderboardCaches() {
	farmLeaderboardCache.Range(func(key, value any) bool {
		farmLeaderboardCache.Delete(key)
		return true
	})
	farmRankCache.Range(func(key, value any) bool {
		farmRankCache.Delete(key)
		return true
	})
}

func invalidateFarmCreditScoreCache(telegramId string) {
	if telegramId == "" {
		return
	}
	farmCreditScoreStatsCache.Delete(telegramId)
}

func shouldInvalidateFarmLeaderboard(action string, amount int) bool {
	if amount != 0 {
		return true
	}
	switch action {
	case "harvest", "steal", "levelup", "prestige":
		return true
	default:
		return false
	}
}

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

	// ===== 土壤肥力系统（A-1）=====
	SoilN        int   `json:"soil_n" gorm:"default:60"`          // 氮 0-100
	SoilP        int   `json:"soil_p" gorm:"default:60"`          // 磷 0-100
	SoilK        int   `json:"soil_k" gorm:"default:60"`          // 钾 0-100
	SoilPH       int   `json:"soil_ph" gorm:"default:65"`         // PH x10，范围 45-85，65=中性
	SoilOM       int   `json:"soil_om" gorm:"default:40"`         // 有机质 0-100
	SoilFatigue  int   `json:"soil_fatigue" gorm:"default:0"`     // 连作疲劳 0-100，值越高越差
	LastCropType string `json:"last_crop_type" gorm:"type:varchar(32);default:''"` // 上一轮作物，用于判定连作
	FallowUntil  int64 `json:"fallow_until" gorm:"default:0"`     // 休耕截止时间（0=未休耕）
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
	err := DB.Create(log).Error
	if err == nil {
		invalidateFarmLeaderboardCaches()
	}
	return err
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

func WaterFarmPlots(plotIds []int, wateredAt int64) error {
	if len(plotIds) == 0 {
		return nil
	}
	if wateredAt <= 0 {
		wateredAt = time.Now().Unix()
	}
	return DB.Model(&TgFarmPlot{}).Where("id IN ?", plotIds).Update("last_watered_at", wateredAt).Error
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
	Quality       int    `json:"quality" gorm:"default:1"`
	Generation    int    `json:"generation" gorm:"default:0"`
	ParentAId     int    `json:"parent_a_id" gorm:"default:0"`
	ParentBId     int    `json:"parent_b_id" gorm:"default:0"`
	BreedCooldownAt int64 `json:"breed_cooldown_at" gorm:"default:0"`
}

const RanchMaxAnimals = 6

func GetRanchAnimals(telegramId string) ([]*TgRanchAnimal, error) {
	var animals []*TgRanchAnimal
	err := DB.Where("telegram_id = ?", telegramId).Order("id asc").Find(&animals).Error
	return animals, err
}

func GetActiveRanchAnimalsByTelegramIds(telegramIds []string) ([]*TgRanchAnimal, error) {
	if len(telegramIds) == 0 {
		return []*TgRanchAnimal{}, nil
	}
	var animals []*TgRanchAnimal
	err := DB.Where("telegram_id IN ? AND status != ?", telegramIds, 5).Order("telegram_id asc, id asc").Find(&animals).Error
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
	now := time.Now().Unix()
	if cached, ok := farmLevelCache.Load(telegramId); ok {
		entry := cached.(farmIntCacheEntry)
		if entry.ExpiresAt >= now {
			return entry.Value
		}
		farmLevelCache.Delete(telegramId)
	}
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_level").First(&item).Error
	if err != nil || item.Quantity < 1 {
		farmLevelCache.Store(telegramId, farmIntCacheEntry{Value: 1, ExpiresAt: now + farmLevelCacheTTLSeconds})
		return 1
	}
	farmLevelCache.Store(telegramId, farmIntCacheEntry{Value: item.Quantity, ExpiresAt: now + farmLevelCacheTTLSeconds})
	return item.Quantity
}

func SetFarmLevel(telegramId string, level int) {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_level").First(&item).Error
	if err != nil {
		item = TgFarmItem{TelegramId: telegramId, ItemType: "_level", Quantity: level}
		_ = DB.Create(&item).Error
		farmLevelCache.Store(telegramId, farmIntCacheEntry{Value: level, ExpiresAt: time.Now().Unix() + farmLevelCacheTTLSeconds})
		invalidateFarmLeaderboardCaches()
		return
	}
	_ = DB.Model(&TgFarmItem{}).Where("id = ?", item.Id).Update("quantity", level).Error
	farmLevelCache.Store(telegramId, farmIntCacheEntry{Value: level, ExpiresAt: time.Now().Unix() + farmLevelCacheTTLSeconds})
	invalidateFarmLeaderboardCaches()
}

// ========== 每日任务 & 成就 ==========

type TgFarmTaskClaim struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index:idx_farm_task_claim,priority:1"`
	TaskDate   string `json:"task_date" gorm:"type:varchar(10);index:idx_farm_task_claim,priority:2"`
	TaskIndex  int    `json:"task_index" gorm:"index:idx_farm_task_claim,priority:3"`
}

type TgFarmAchievement struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId     string `json:"telegram_id" gorm:"type:varchar(64);uniqueIndex:idx_achieve"`
	AchievementKey string `json:"achievement_key" gorm:"type:varchar(32);uniqueIndex:idx_achieve"`
	UnlockedAt     int64  `json:"unlocked_at"`
}

func GetTaskClaims(telegramId, taskDate string) ([]int, error) {
	cacheKey := telegramId + "|" + taskDate
	now := time.Now().Unix()
	if cached, ok := farmTaskClaimsCache.Load(cacheKey); ok {
		entry := cached.(farmTaskClaimsCacheEntry)
		if entry.ExpiresAt >= now {
			result := make([]int, len(entry.Value))
			copy(result, entry.Value)
			return result, nil
		}
		farmTaskClaimsCache.Delete(cacheKey)
	}
	var claims []*TgFarmTaskClaim
	err := DB.Model(&TgFarmTaskClaim{}).
		Select("task_index").
		Where("telegram_id = ? AND task_date = ?", telegramId, taskDate).
		Find(&claims).Error
	if err != nil {
		return nil, err
	}
	var indices []int
	for _, c := range claims {
		indices = append(indices, c.TaskIndex)
	}
	cachedValue := make([]int, len(indices))
	copy(cachedValue, indices)
	farmTaskClaimsCache.Store(cacheKey, farmTaskClaimsCacheEntry{Value: cachedValue, ExpiresAt: now + farmTaskClaimsCacheTTLSeconds})
	return indices, nil
}

func ClaimTask(telegramId, taskDate string, taskIndex int) error {
	err := DB.Create(&TgFarmTaskClaim{
		TelegramId: telegramId,
		TaskDate:   taskDate,
		TaskIndex:  taskIndex,
	}).Error
	if err == nil {
		farmTaskClaimsCache.Delete(telegramId + "|" + taskDate)
	}
	return err
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

type farmActionCountRow struct {
	Action string
	Count  int64
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func GetActionCountsSince(telegramId string, actions []string, since int64) (map[string]int64, error) {
	result := make(map[string]int64)
	uniqueActions := uniqueStrings(actions)
	if len(uniqueActions) == 0 {
		return result, nil
	}
	sort.Strings(uniqueActions)
	cacheKey := telegramId + "|" + strconv.FormatInt(since, 10) + "|" + strings.Join(uniqueActions, ",")
	now := time.Now().Unix()
	version := getFarmActionVersion(telegramId)
	if cached, ok := farmActionCountCache.Load(cacheKey); ok {
		entry := cached.(farmActionCountCacheEntry)
		if entry.ExpiresAt >= now && entry.Version == version {
			return cloneFarmActionCountMap(entry.Value), nil
		}
		farmActionCountCache.Delete(cacheKey)
	}

	var rows []farmActionCountRow
	query := DB.Model(&TgFarmLog{}).
		Select("action, COUNT(*) as count").
		Where("telegram_id = ? AND action IN ?", telegramId, uniqueActions)
	if since > 0 {
		query = query.Where("created_at >= ?", since)
	}
	if err := query.Group("action").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.Action] = row.Count
	}
	farmActionCountCache.Store(cacheKey, farmActionCountCacheEntry{
		Value:     cloneFarmActionCountMap(result),
		Version:   version,
		ExpiresAt: now + farmActionCacheTTLSeconds,
	})
	return result, nil
}

func GetActionCountsTotal(telegramId string, actions []string) (map[string]int64, error) {
	return GetActionCountsSince(telegramId, actions, 0)
}

func GetFarmItemQuantities(telegramId string, itemTypes []string) (map[string]int, error) {
	result := make(map[string]int)
	uniqueTypes := uniqueStrings(itemTypes)
	if len(uniqueTypes) == 0 {
		return result, nil
	}

	var items []*TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type IN ?", telegramId, uniqueTypes).Find(&items).Error
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.ItemType] = item.Quantity
	}
	return result, nil
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
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index:idx_farm_log_tg_created,priority:1;index:idx_farm_log_tg_action_created,priority:1"`
	Action     string `json:"action" gorm:"type:varchar(32);index:idx_farm_log_tg_action_created,priority:2"`
	Amount     int    `json:"amount" gorm:"type:bigint;default:0"`
	Detail     string `json:"detail" gorm:"type:varchar(255)"`
	CreatedAt  int64  `json:"created_at" gorm:"index:idx_farm_log_tg_created,priority:2;index:idx_farm_log_tg_action_created,priority:3"`
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
	if shouldInvalidateFarmLeaderboard(action, amount) {
		invalidateFarmLeaderboardCaches()
	}
	invalidateFarmCreditScoreCache(telegramId)
	bumpFarmActionVersion(telegramId)
}

func AddFarmLogs(telegramId, action string, amount int, detail string, count int) {
	if count <= 1 {
		AddFarmLog(telegramId, action, amount, detail)
		return
	}
	now := time.Now().Unix()
	logs := make([]TgFarmLog, 0, count)
	for i := 0; i < count; i++ {
		logs = append(logs, TgFarmLog{
			TelegramId: telegramId,
			Action:     action,
			Amount:     amount,
			Detail:     detail,
			CreatedAt:  now,
		})
	}
	_ = DB.Create(&logs).Error
	if shouldInvalidateFarmLeaderboard(action, amount) {
		invalidateFarmLeaderboardCaches()
	}
	invalidateFarmCreditScoreCache(telegramId)
	bumpFarmActionVersion(telegramId)
}

// ========== 银行贷款 ==========

type TgFarmLoan struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId   string `json:"telegram_id" gorm:"type:varchar(64);index:idx_farm_loan_tg_status_due,priority:1;index:idx_farm_loan_tg_type_status_due,priority:1"`
	Principal    int    `json:"principal" gorm:"type:bigint;default:0"`               // 本金(quota)
	Interest     int    `json:"interest" gorm:"type:bigint;default:0"`                // 利息(quota)
	TotalDue     int    `json:"total_due" gorm:"type:bigint;default:0"`               // 应还总额
	Repaid       int    `json:"repaid" gorm:"type:bigint;default:0"`                  // 已还金额
	Status       int    `json:"status" gorm:"default:0;index:idx_farm_loan_tg_status_due,priority:2;index:idx_farm_loan_tg_type_status_due,priority:3"`      // 0=未还清 1=已还清 2=违约
	LoanType     int    `json:"loan_type" gorm:"default:0;index:idx_farm_loan_tg_type_status_due,priority:2"`   // 0=普通贷款 1=抵押贷款
	CreditScore  int    `json:"credit_score"`                 // 贷款时的信用评分
	DueAt        int64  `json:"due_at" gorm:"index:idx_farm_loan_tg_status_due,priority:3;index:idx_farm_loan_tg_type_status_due,priority:4"`                       // 到期时间
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

// GetCreditScoreStats aggregates the data needed for credit scoring.
type FarmCreditScoreStats struct {
	PositiveCount    int64
	TotalIncome      int64
	OverdueLoanCount int64
	RepaidCount      int64
	Level            int
}

func GetCreditScoreStats(telegramId string) (FarmCreditScoreStats, error) {
	stats := FarmCreditScoreStats{Level: 1}
	now := time.Now().Unix()
	if cached, ok := farmCreditScoreStatsCache.Load(telegramId); ok {
		entry := cached.(farmCreditScoreStatsCacheEntry)
		if entry.ExpiresAt >= now {
			return entry.Value, nil
		}
		farmCreditScoreStatsCache.Delete(telegramId)
	}
	thirtyDaysAgo := time.Now().Unix() - 30*86400

	type incomeResult struct {
		PositiveCount int64
		TotalIncome   int64
	}
	var income incomeResult
	if err := DB.Model(&TgFarmLog{}).
		Select("COUNT(*) as positive_count, COALESCE(SUM(amount),0) as total_income").
		Where("telegram_id = ? AND created_at > ? AND amount > 0", telegramId, thirtyDaysAgo).
		Scan(&income).Error; err != nil {
		return stats, err
	}
	stats.PositiveCount = income.PositiveCount
	stats.TotalIncome = income.TotalIncome

	type loanResult struct {
		OverdueLoanCount int64
		RepaidCount      int64
	}
	var loan loanResult
	if err := DB.Model(&TgFarmLoan{}).
		Select("COALESCE(SUM(CASE WHEN status = 0 AND due_at < ? THEN 1 ELSE 0 END),0) as overdue_loan_count, COALESCE(SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END),0) as repaid_count", now).
		Where("telegram_id = ?", telegramId).
		Scan(&loan).Error; err != nil {
		return stats, err
	}
	stats.OverdueLoanCount = loan.OverdueLoanCount
	stats.RepaidCount = loan.RepaidCount

	itemMap, err := GetFarmItemQuantities(telegramId, []string{"_level"})
	if err != nil {
		return stats, err
	}
	if level, ok := itemMap["_level"]; ok && level > 0 {
		stats.Level = level
	}
	farmCreditScoreStatsCache.Store(telegramId, farmCreditScoreStatsCacheEntry{Value: stats, ExpiresAt: now + farmCreditScoreTTLSeconds})

	return stats, nil
}

func CalculateCreditScore(stats FarmCreditScoreStats) int {
	score := 1

	activityBonus := int(stats.PositiveCount / 10)
	if activityBonus > 3 {
		activityBonus = 3
	}
	score += activityBonus

	incomeBonus := int(stats.TotalIncome / 5000000)
	if incomeBonus > 3 {
		incomeBonus = 3
	}
	score += incomeBonus

	historyBonus := int(stats.RepaidCount)
	if historyBonus > 2 {
		historyBonus = 2
	}
	score += historyBonus

	levelBonus := (stats.Level - 1) / 3
	if levelBonus > 2 {
		levelBonus = 2
	}
	score += levelBonus

	score -= int(stats.OverdueLoanCount) * 3

	if score < 1 {
		score = 1
	}
	maxMul := common.TgBotFarmBankMaxMultiplier
	if maxMul < 1 {
		maxMul = 10
	}
	if score > maxMul {
		score = maxMul
	}
	return score
}

// GetCreditScore 鏍规嵁娑堣垂璁板綍璁＄畻淇＄敤璇勫垎(1~maxMultiplier)
func GetCreditScore(telegramId string) int {
	stats, err := GetCreditScoreStats(telegramId)
	if err != nil {
		stats = FarmCreditScoreStats{Level: 1}
	}
	return CalculateCreditScore(stats)
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
	if err == nil {
		invalidateFarmCreditScoreCache(telegramId)
	}
	return loan, err
}

// RepayLoan 还款（部分或全部）
func RepayLoan(loanId int, amount int) (*TgFarmLoan, error) {
	return RepayLoanWithExtend(loanId, amount, 0)
}

// RepayLoanWithExtend 还款并可选延长期限
func RepayLoanWithExtend(loanId int, amount int, extendDays int) (*TgFarmLoan, error) {
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
	updates := map[string]interface{}{
		"repaid": loan.Repaid,
		"status": loan.Status,
	}
	if extendDays > 0 && loan.Status != 1 {
		loan.DueAt += int64(extendDays) * 86400
		updates["due_at"] = loan.DueAt
	}
	err = DB.Model(&TgFarmLoan{}).Where("id = ?", loanId).Updates(updates).Error
	if err == nil {
		invalidateFarmCreditScoreCache(loan.TelegramId)
	}
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
	now := time.Now().Unix()
	if cached, ok := farmWarehouseLevelCache.Load(telegramId); ok {
		entry := cached.(farmIntCacheEntry)
		if entry.ExpiresAt >= now {
			return entry.Value
		}
		farmWarehouseLevelCache.Delete(telegramId)
	}
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_warehouse_level").First(&item).Error
	if err != nil || item.Quantity < 1 {
		farmWarehouseLevelCache.Store(telegramId, farmIntCacheEntry{Value: 1, ExpiresAt: now + farmWarehouseCacheTTLSeconds})
		return 1
	}
	farmWarehouseLevelCache.Store(telegramId, farmIntCacheEntry{Value: item.Quantity, ExpiresAt: now + farmWarehouseCacheTTLSeconds})
	return item.Quantity
}

// SetWarehouseLevel 设置用户仓库等级
func SetWarehouseLevel(telegramId string, level int) error {
	var item TgFarmItem
	err := DB.Where("telegram_id = ? AND item_type = ?", telegramId, "_warehouse_level").First(&item).Error
	if err != nil {
		err = DB.Create(&TgFarmItem{TelegramId: telegramId, ItemType: "_warehouse_level", Quantity: level}).Error
		if err == nil {
			farmWarehouseLevelCache.Store(telegramId, farmIntCacheEntry{Value: level, ExpiresAt: time.Now().Unix() + farmWarehouseCacheTTLSeconds})
		}
		return err
	}
	err = DB.Model(&TgFarmItem{}).Where("telegram_id = ? AND item_type = ?", telegramId, "_warehouse_level").Update("quantity", level).Error
	if err == nil {
		farmWarehouseLevelCache.Store(telegramId, farmIntCacheEntry{Value: level, ExpiresAt: time.Now().Unix() + farmWarehouseCacheTTLSeconds})
	}
	return err
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

func RecordCollectionWithStatus(telegramId, category, itemKey string, qty int) (bool, int, int64, error) {
	var item TgFarmCollection
	err := DB.Where("telegram_id = ? AND category = ? AND item_key = ?", telegramId, category, itemKey).First(&item).Error
	if err == nil {
		nextQty := item.Quantity + qty
		if updateErr := DB.Model(&TgFarmCollection{}).Where("id = ?", item.Id).Update("quantity", nextQty).Error; updateErr != nil {
			return false, 0, 0, updateErr
		}
		return false, nextQty, item.FirstAt, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, 0, 0, err
	}
	firstAt := time.Now().Unix()
	createItem := &TgFarmCollection{TelegramId: telegramId, Category: category, ItemKey: itemKey, Quantity: qty, FirstAt: firstAt}
	if createErr := DB.Create(createItem).Error; createErr == nil {
		return true, qty, firstAt, nil
	}
	if retryErr := DB.Where("telegram_id = ? AND category = ? AND item_key = ?", telegramId, category, itemKey).First(&item).Error; retryErr != nil {
		return false, 0, 0, retryErr
	}
	nextQty := item.Quantity + qty
	if updateErr := DB.Model(&TgFarmCollection{}).Where("id = ?", item.Id).Update("quantity", nextQty).Error; updateErr != nil {
		return false, 0, 0, updateErr
	}
	return false, nextQty, item.FirstAt, nil
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
	PricePerUnit int    `json:"price_per_unit" gorm:"type:bigint;default:0"`
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
		invalidateFarmLeaderboardCaches()
		return
	}
	DB.Model(&TgFarmItem{}).Where("id = ?", item.Id).Update("quantity", level)
	invalidateFarmLeaderboardCaches()
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
	SetFarmLevel(telegramId, 1)
}

func GetPrestigePrice(nextPrestigeLevel int) int {
	if nextPrestigeLevel < 1 {
		nextPrestigeLevel = 1
	}
	price := big.NewRat(int64(common.TgBotFarmPrestigeBasePrice), 1)
	maxQuota := big.NewInt(common.MaxSafeQuota)
	mul110 := big.NewRat(110, 100)
	mul150 := big.NewRat(150, 100)
	for i := 2; i <= nextPrestigeLevel; i++ {
		if i <= 100 {
			price.Mul(price, mul110)
		} else {
			price.Mul(price, mul150)
		}
		threshold := new(big.Int).Mul(maxQuota, price.Denom())
		if price.Num().Cmp(threshold) >= 0 {
			return int(common.MaxSafeQuota)
		}
	}
	result := new(big.Int).Quo(price.Num(), price.Denom())
	return common.ClampQuotaBigInt(result)
}

// ========== 小游戏记录 ==========

type TgFarmGameLog struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramId string `json:"telegram_id" gorm:"type:varchar(64);index"`
	GameType   string `json:"game_type" gorm:"type:varchar(16)"`
	BetAmount  int    `json:"bet_amount" gorm:"type:bigint;default:0"`
	WinAmount  int    `json:"win_amount" gorm:"type:bigint;default:0"`
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

func GetInstalledAutomations(telegramId string) (map[string]bool, error) {
	now := time.Now().Unix()
	if cached, ok := farmAutomationCache.Load(telegramId); ok {
		entry := cached.(farmBoolMapCacheEntry)
		if entry.ExpiresAt >= now {
			return cloneFarmBoolMap(entry.Value), nil
		}
		farmAutomationCache.Delete(telegramId)
	}
	result := make(map[string]bool)
	items, err := GetAutomations(telegramId)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.Type] = true
	}
	farmAutomationCache.Store(telegramId, farmBoolMapCacheEntry{Value: cloneFarmBoolMap(result), ExpiresAt: now + farmAutomationCacheTTLSeconds})
	return result, nil
}

func GetAutomationOwnerMap(autoTypes []string) (map[string][]string, error) {
	result := make(map[string][]string)
	uniqueTypes := uniqueStrings(autoTypes)
	if len(uniqueTypes) == 0 {
		return result, nil
	}
	for _, autoType := range uniqueTypes {
		result[autoType] = []string{}
	}
	var items []*TgFarmAutomation
	err := DB.Select("telegram_id, type").Where("type IN ?", uniqueTypes).Find(&items).Error
	if err != nil {
		return nil, err
	}
	seen := make(map[string]map[string]struct{}, len(uniqueTypes))
	for _, item := range items {
		if item.TelegramId == "" {
			continue
		}
		if _, ok := seen[item.Type]; !ok {
			seen[item.Type] = make(map[string]struct{})
		}
		if _, ok := seen[item.Type][item.TelegramId]; ok {
			continue
		}
		seen[item.Type][item.TelegramId] = struct{}{}
		result[item.Type] = append(result[item.Type], item.TelegramId)
	}
	return result, nil
}

func GetDueAutoWaterPlots(cutoff int64, telegramIds []string) ([]*TgFarmPlot, error) {
	var plots []*TgFarmPlot
	query := DB.Where("status = ? AND last_watered_at > 0 AND last_watered_at <= ?", 1, cutoff)
	if len(telegramIds) > 0 {
		query = query.Where("telegram_id IN ?", telegramIds)
	}
	err := query.Order("telegram_id asc, id asc").Find(&plots).Error
	return plots, err
}

func GetTriggeredDroughtPlots(telegramIds []string, now int64) ([]*TgFarmPlot, error) {
	if len(telegramIds) == 0 {
		return []*TgFarmPlot{}, nil
	}
	var plots []*TgFarmPlot
	err := DB.Where("telegram_id IN ? AND status = ? AND event_type = ? AND event_at > 0 AND event_at <= ?", telegramIds, 1, "drought", now).
		Order("telegram_id asc, id asc").
		Find(&plots).Error
	return plots, err
}

func HasAutomation(telegramId, autoType string) bool {
	installed, err := GetInstalledAutomations(telegramId)
	if err != nil {
		var count int64
		DB.Model(&TgFarmAutomation{}).Where("telegram_id = ? AND type = ?", telegramId, autoType).Count(&count)
		return count > 0
	}
	return installed[autoType]
}

func CreateAutomation(telegramId, autoType string) error {
	err := DB.Create(&TgFarmAutomation{
		TelegramId: telegramId, Type: autoType, Level: 1, InstalledAt: time.Now().Unix(),
	}).Error
	if err == nil {
		farmAutomationCache.Delete(telegramId)
	}
	return err
}

// ========== 排行榜 ==========

type FarmLeaderboardEntry struct {
	TelegramId string
	Username   string
	Value      int64
}

type FarmLeaderboardRankedEntry struct {
	Rank int64
	FarmLeaderboardEntry
}

type FarmLeaderboardOptions struct {
	Scope  string
	Period string
	Group  string
	// Cohort 新老玩家分组："old" / "new" / "all" / ""
	//   - "old"：首次进入农场时间 < FarmLeaderboardCohortCutoff 的玩家
	//   - "new"：首次进入农场时间 >= FarmLeaderboardCohortCutoff 的玩家
	//   - "all"/""：不做 cohort 过滤（管理员全量视角 / 历史兼容行为）
	Cohort string
	UserId int
}

type farmLeaderboardValueRow struct {
	TelegramId string
	Value      int64
}

const (
	farmLeaderboardScopeGlobal  = "global"
	farmLeaderboardScopeFriends = "friends"
	farmLeaderboardPeriodAll    = "all"
	farmLeaderboardPeriodWeekly = "weekly"

	FarmLeaderboardCohortOld = "old"
	FarmLeaderboardCohortNew = "new"
	FarmLeaderboardCohortAll = "all"
)

// FarmLeaderboardCohortCutoff 新老玩家分水岭（Unix 秒）
//
// 玩家"首次进入农场"的时间定义为：该玩家在 tg_farm_logs 中最早一条
// 记录的 created_at（只要有任何农场操作就会产生 log）。
//   - first_at >= cutoff → new cohort（新玩家榜）
//   - first_at <  cutoff → old cohort（老玩家榜）
//   - 没有任何 log（纯新账号没操作过农场）→ 不会出现在排行榜里，无需分组
//
// 默认值：2026-04-19 00:00:00 +08（即北京时间当天零点）。
// 如需调整分水岭，修改此变量后重启服务即可。
var FarmLeaderboardCohortCutoff = time.Date(2026, 4, 19, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)).Unix()

func normalizeFarmLeaderboardCohort(cohort string) string {
	switch cohort {
	case FarmLeaderboardCohortOld, FarmLeaderboardCohortNew, FarmLeaderboardCohortAll:
		return cohort
	}
	return ""
}

// GetFarmPlayerCohort 返回玩家所属 cohort（"old"/"new"）。
// 判定依据：tg_farm_logs 中该玩家最早一条记录的 created_at 与 cutoff 比较。
// 未在 tg_farm_logs 留下痕迹的账号（从未操作过农场）按 old 处理，因为
// 他们本来就不会出现在排行榜上，归哪里都无实际影响。
func GetFarmPlayerCohort(telegramId string) string {
	if telegramId == "" {
		return FarmLeaderboardCohortOld
	}
	var firstAt int64
	err := DB.Model(&TgFarmLog{}).
		Where("telegram_id = ?", telegramId).
		Select("COALESCE(MIN(created_at), 0)").
		Row().Scan(&firstAt)
	if err != nil || firstAt == 0 {
		return FarmLeaderboardCohortOld
	}
	if firstAt >= FarmLeaderboardCohortCutoff {
		return FarmLeaderboardCohortNew
	}
	return FarmLeaderboardCohortOld
}

// getFarmLeaderboardCohortIds 预先计算属于指定 cohort 的玩家 telegram_id 集合。
//
// 返回值语义：
//   - (nil, nil)     —— 不过滤（cohort 为空或 "all"）
//   - ([], nil)      —— 指定 cohort 明确没有人（调用方应直接返回空结果）
//   - (ids, nil)     —— 只保留 telegram_id ∈ ids 的玩家
//
// 性能：在 tg_farm_logs(telegram_id, created_at) 联合索引上做 GROUP BY + MIN，
// 对几千-几万级玩家量完全够用。外层 30 秒缓存进一步降低 QPS 压力。
func getFarmLeaderboardCohortIds(cohort string) ([]string, error) {
	normalized := normalizeFarmLeaderboardCohort(cohort)
	if normalized == "" || normalized == FarmLeaderboardCohortAll {
		return nil, nil
	}
	var ids []string
	q := DB.Model(&TgFarmLog{}).
		Select("telegram_id").
		Where("telegram_id != ''").
		Group("telegram_id")
	if normalized == FarmLeaderboardCohortNew {
		q = q.Having("MIN(created_at) >= ?", FarmLeaderboardCohortCutoff)
	} else {
		q = q.Having("MIN(created_at) < ?", FarmLeaderboardCohortCutoff)
	}
	if err := q.Pluck("telegram_id", &ids).Error; err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

type FarmLeaderboardGroupMeta struct {
	Key        string
	Label      string
	RangeLabel string
	MinLevel   int
	MaxLevel   int
}

type FarmLeaderboardRewardTier struct {
	Key        string
	Label      string
	Title      string
	ShortTitle string
	Emoji      string
}

type FarmLeaderboardRewardBand struct {
	Key        string
	Label      string
	Title      string
	ShortTitle string
	Emoji      string
	StartRank  int
	EndRank    int
	Count      int
}

var farmLeaderboardGroupMetas = map[string]FarmLeaderboardGroupMeta{
	"newbie":   {Key: "newbie", Label: "新手组", RangeLabel: "Lv.1-30", MinLevel: 1, MaxLevel: 30},
	"advanced": {Key: "advanced", Label: "进阶组", RangeLabel: "Lv.31-60", MinLevel: 31, MaxLevel: 60},
	"elite":    {Key: "elite", Label: "大佬组", RangeLabel: "Lv.61+", MinLevel: 61, MaxLevel: 0},
}

var farmLeaderboardRewardTiers = []FarmLeaderboardRewardTier{
	{Key: "diamond", Label: "钻石段位", Title: "钻石荣誉", ShortTitle: "钻石", Emoji: "💎"},
	{Key: "gold", Label: "黄金段位", Title: "黄金荣誉", ShortTitle: "黄金", Emoji: "🏆"},
	{Key: "silver", Label: "白银段位", Title: "白银荣誉", ShortTitle: "白银", Emoji: "🥈"},
	{Key: "sprout", Label: "上榜荣誉", Title: "上榜荣誉", ShortTitle: "上榜", Emoji: "🌱"},
}

func normalizeFarmLeaderboardScope(scope string) string {
	if scope == farmLeaderboardScopeFriends {
		return scope
	}
	return farmLeaderboardScopeGlobal
}

func normalizeFarmLeaderboardGroup(group string) string {
	if _, ok := farmLeaderboardGroupMetas[group]; ok {
		return group
	}
	return ""
}

func ResolveFarmLeaderboardGroupByLevel(level int) string {
	if level <= 30 {
		return "newbie"
	}
	if level <= 60 {
		return "advanced"
	}
	return "elite"
}

func GetFarmLeaderboardGroupMeta(group string) FarmLeaderboardGroupMeta {
	normalized := normalizeFarmLeaderboardGroup(group)
	if meta, ok := farmLeaderboardGroupMetas[normalized]; ok {
		return meta
	}
	return FarmLeaderboardGroupMeta{Key: "all", Label: "全部玩家", RangeLabel: "Lv.1+", MinLevel: 1, MaxLevel: 0}
}

func farmLeaderboardLevelInGroup(level int, group string) bool {
	normalized := normalizeFarmLeaderboardGroup(group)
	if normalized == "" {
		return true
	}
	meta := GetFarmLeaderboardGroupMeta(normalized)
	if level < meta.MinLevel {
		return false
	}
	if meta.MaxLevel > 0 && level > meta.MaxLevel {
		return false
	}
	return true
}

func buildFarmLeaderboardLevelFilter(group, expr string) (string, []any) {
	normalized := normalizeFarmLeaderboardGroup(group)
	if normalized == "" {
		return "", nil
	}
	meta := GetFarmLeaderboardGroupMeta(normalized)
	if meta.MaxLevel > 0 {
		return fmt.Sprintf(" AND %s BETWEEN ? AND ?", expr), []any{meta.MinLevel, meta.MaxLevel}
	}
	return fmt.Sprintf(" AND %s >= ?", expr), []any{meta.MinLevel}
}

func farmLeaderboardRewardCutoffs(total int) (int, int, int) {
	if total <= 0 {
		return 0, 0, 0
	}
	diamond := int(math.Ceil(float64(total) * 0.10))
	gold := int(math.Ceil(float64(total) * 0.20))
	silver := int(math.Ceil(float64(total) * 0.50))
	if diamond < 1 {
		diamond = 1
	}
	if gold < diamond {
		gold = diamond
	}
	if silver < gold {
		silver = gold
	}
	if diamond > total {
		diamond = total
	}
	if gold > total {
		gold = total
	}
	if silver > total {
		silver = total
	}
	return diamond, gold, silver
}

func GetFarmLeaderboardRewardTier(rank int64, total int) *FarmLeaderboardRewardTier {
	if rank <= 0 || total <= 0 {
		return nil
	}
	diamond, gold, silver := farmLeaderboardRewardCutoffs(total)
	var tier FarmLeaderboardRewardTier
	switch {
	case int(rank) <= diamond:
		tier = farmLeaderboardRewardTiers[0]
	case int(rank) <= gold:
		tier = farmLeaderboardRewardTiers[1]
	case int(rank) <= silver:
		tier = farmLeaderboardRewardTiers[2]
	default:
		tier = farmLeaderboardRewardTiers[3]
	}
	return &tier
}

func GetFarmLeaderboardRewardBands(total int) []FarmLeaderboardRewardBand {
	diamond, gold, silver := farmLeaderboardRewardCutoffs(total)
	bands := make([]FarmLeaderboardRewardBand, 0, 3)
	if diamond > 0 {
		tier := farmLeaderboardRewardTiers[0]
		bands = append(bands, FarmLeaderboardRewardBand{Key: tier.Key, Label: tier.Label, Title: tier.Title, ShortTitle: tier.ShortTitle, Emoji: tier.Emoji, StartRank: 1, EndRank: diamond, Count: diamond})
	}
	if gold > diamond {
		tier := farmLeaderboardRewardTiers[1]
		bands = append(bands, FarmLeaderboardRewardBand{Key: tier.Key, Label: tier.Label, Title: tier.Title, ShortTitle: tier.ShortTitle, Emoji: tier.Emoji, StartRank: diamond + 1, EndRank: gold, Count: gold - diamond})
	}
	if silver > gold {
		tier := farmLeaderboardRewardTiers[2]
		bands = append(bands, FarmLeaderboardRewardBand{Key: tier.Key, Label: tier.Label, Title: tier.Title, ShortTitle: tier.ShortTitle, Emoji: tier.Emoji, StartRank: gold + 1, EndRank: silver, Count: silver - gold})
	}
	return bands
}

func normalizeFarmLeaderboardPeriod(period string) string {
	if period == farmLeaderboardPeriodWeekly {
		return period
	}
	return farmLeaderboardPeriodAll
}

func getFarmLeaderboardWeekStart() int64 {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start = start.AddDate(0, 0, -(weekday - 1))
	return start.Unix()
}

func sortFarmLeaderboardEntries(entries []FarmLeaderboardEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Value == entries[j].Value {
			return entries[i].TelegramId < entries[j].TelegramId
		}
		return entries[i].Value > entries[j].Value
	})
}

func resolveFarmLeaderboardUserName(user User) string {
	if user.DisplayName != "" {
		return user.DisplayName
	}
	if user.Username != "" {
		return user.Username
	}
	if user.TelegramId != "" {
		return user.TelegramId
	}
	return strconv.Itoa(user.Id)
}

func resolveFarmLeaderboardFarmId(user User) string {
	if user.TelegramId != "" {
		return user.TelegramId
	}
	return strconv.Itoa(user.Id)
}

func copyFarmLeaderboardEntries(entries []FarmLeaderboardEntry) []FarmLeaderboardEntry {
	copied := make([]FarmLeaderboardEntry, len(entries))
	copy(copied, entries)
	return copied
}

func getFarmLeaderboardFriendUsers(userId int, group string) ([]User, map[string]string, []string, error) {
	if userId == 0 {
		return []User{}, map[string]string{}, []string{}, nil
	}
	friendIds, err := GetFriendList(userId)
	if err != nil {
		return nil, nil, nil, err
	}
	ids := make([]int, 0, len(friendIds)+1)
	seen := make(map[int]struct{}, len(friendIds)+1)
	ids = append(ids, userId)
	seen[userId] = struct{}{}
	for _, friendId := range friendIds {
		if _, ok := seen[friendId]; ok {
			continue
		}
		seen[friendId] = struct{}{}
		ids = append(ids, friendId)
	}
	var users []User
	if err := DB.Select("id, username, display_name, telegram_id, quota").Where("id IN ? AND status = ? AND role < ?", ids, 1, 10).Find(&users).Error; err != nil {
		return nil, nil, nil, err
	}
	filteredUsers := make([]User, 0, len(users))
	nameMap := make(map[string]string, len(users))
	farmIds := make([]string, 0, len(users))
	for _, user := range users {
		level := 1
		if user.TelegramId != "" {
			level = GetFarmLevel(user.TelegramId)
		}
		if !farmLeaderboardLevelInGroup(level, group) {
			continue
		}
		farmId := resolveFarmLeaderboardFarmId(user)
		filteredUsers = append(filteredUsers, user)
		nameMap[farmId] = resolveFarmLeaderboardUserName(user)
		farmIds = append(farmIds, farmId)
	}
	return filteredUsers, nameMap, farmIds, nil
}

func buildFarmLeaderboardEntriesFromRows(rows []farmLeaderboardValueRow, nameMap map[string]string) []FarmLeaderboardEntry {
	entries := make([]FarmLeaderboardEntry, 0, len(rows))
	for _, row := range rows {
		if row.Value <= 0 {
			continue
		}
		name := nameMap[row.TelegramId]
		if name == "" {
			continue
		}
		entries = append(entries, FarmLeaderboardEntry{
			TelegramId: row.TelegramId,
			Username:   name,
			Value:      row.Value,
		})
	}
	return entries
}

// filterFarmLeaderboardEntriesByCohort 按 cohortIds 在内存里过滤条目。
//   - cohortIds == nil：不过滤，原样返回（保留历史行为）
//   - cohortIds == []：调用方应已短路，这里兜底返回空
//   - cohortIds 非空：只保留 TelegramId ∈ cohortIds 的条目
//
// 之所以不在 SQL 层做 IN 过滤：
//  1. 避免 DB.Raw 与超长 IN 列表的参数展开复杂度与 DB 驱动差异
//  2. 全量结果最多几千-几万行，内存 O(n) 过滤 + 30 秒缓存，成本可忽略
func filterFarmLeaderboardEntriesByCohort(entries []FarmLeaderboardEntry, cohortIds []string) []FarmLeaderboardEntry {
	if cohortIds == nil {
		return entries
	}
	allow := make(map[string]struct{}, len(cohortIds))
	for _, id := range cohortIds {
		allow[id] = struct{}{}
	}
	filtered := entries[:0]
	for _, e := range entries {
		if _, ok := allow[e.TelegramId]; ok {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func getGlobalFarmLeaderboardEntries(boardType, period, group string, cohortIds []string) ([]FarmLeaderboardEntry, error) {
	var entries []FarmLeaderboardEntry
	var err error
	labelExpr := "COALESCE(NULLIF(u.display_name, ''), u.username)"
	since := getFarmLeaderboardWeekStart()
	levelJoin := " LEFT JOIN tg_farm_items level_item ON level_item.telegram_id = u.telegram_id AND level_item.item_type = '_level' "
	levelWhere, levelArgs := buildFarmLeaderboardLevelFilter(group, "COALESCE(level_item.quantity, 1)")
	switch boardType {
	case "balance":
		if period == farmLeaderboardPeriodWeekly {
			query := "SELECT fl.telegram_id, " + labelExpr + " as username, COALESCE(SUM(CASE WHEN fl.amount > 0 THEN fl.amount ELSE 0 END), 0) as value FROM tg_farm_logs fl JOIN users u ON fl.telegram_id = u.telegram_id" + levelJoin + "WHERE fl.created_at >= ? AND u.telegram_id != '' AND u.status = 1 AND u.role < 10" + levelWhere + " GROUP BY fl.telegram_id, " + labelExpr + " HAVING COALESCE(SUM(CASE WHEN fl.amount > 0 THEN fl.amount ELSE 0 END), 0) > 0 ORDER BY value DESC, fl.telegram_id ASC"
			args := append([]any{since}, levelArgs...)
			err = DB.Raw(query, args...).Scan(&entries).Error
		} else {
			query := "SELECT u.telegram_id, " + labelExpr + " as username, u.quota as value FROM users u" + levelJoin + "WHERE u.telegram_id != '' AND u.status = 1 AND u.role < 10 AND u.quota > 0" + levelWhere + " ORDER BY value DESC, u.telegram_id ASC"
			err = DB.Raw(query, levelArgs...).Scan(&entries).Error
		}
	case "level":
		if period == farmLeaderboardPeriodWeekly {
			query := "SELECT fl.telegram_id, " + labelExpr + " as username, COUNT(*) as value FROM tg_farm_logs fl JOIN users u ON fl.telegram_id = u.telegram_id" + levelJoin + "WHERE fl.action = 'levelup' AND fl.created_at >= ? AND u.telegram_id != '' AND u.status = 1 AND u.role < 10" + levelWhere + " GROUP BY fl.telegram_id, " + labelExpr + " ORDER BY value DESC, fl.telegram_id ASC"
			args := append([]any{since}, levelArgs...)
			err = DB.Raw(query, args...).Scan(&entries).Error
		} else {
			groupWhere, groupArgs := buildFarmLeaderboardLevelFilter(group, "COALESCE(fi.quantity, 1)")
			query := "SELECT fi.telegram_id, " + labelExpr + " as username, fi.quantity as value FROM tg_farm_items fi JOIN users u ON fi.telegram_id = u.telegram_id WHERE fi.item_type = '_level' AND fi.quantity > 1 AND u.telegram_id != '' AND u.status = 1 AND u.role < 10" + groupWhere + " ORDER BY fi.quantity DESC, fi.telegram_id ASC"
			err = DB.Raw(query, groupArgs...).Scan(&entries).Error
		}
	case "harvest":
		query := "SELECT fl.telegram_id, " + labelExpr + " as username, MAX(fl.amount) as value FROM tg_farm_logs fl JOIN users u ON fl.telegram_id = u.telegram_id" + levelJoin + "WHERE fl.action = 'harvest' AND fl.amount > 0 AND u.telegram_id != '' AND u.status = 1 AND u.role < 10"
		args := make([]any, 0, len(levelArgs)+1)
		if period == farmLeaderboardPeriodWeekly {
			query += " AND fl.created_at >= ?"
			args = append(args, since)
		}
		query += levelWhere + " GROUP BY fl.telegram_id, " + labelExpr + " ORDER BY value DESC, fl.telegram_id ASC"
		args = append(args, levelArgs...)
		err = DB.Raw(query, args...).Scan(&entries).Error
	case "prestige":
		if period == farmLeaderboardPeriodWeekly {
			query := "SELECT fl.telegram_id, " + labelExpr + " as username, COUNT(*) as value FROM tg_farm_logs fl JOIN users u ON fl.telegram_id = u.telegram_id" + levelJoin + "WHERE fl.action = 'prestige' AND fl.created_at >= ? AND u.telegram_id != '' AND u.status = 1 AND u.role < 10" + levelWhere + " GROUP BY fl.telegram_id, " + labelExpr + " ORDER BY value DESC, fl.telegram_id ASC"
			args := append([]any{since}, levelArgs...)
			err = DB.Raw(query, args...).Scan(&entries).Error
		} else {
			query := "SELECT fi.telegram_id, " + labelExpr + " as username, fi.quantity as value FROM tg_farm_items fi JOIN users u ON fi.telegram_id = u.telegram_id" + levelJoin + "WHERE fi.item_type = '_prestige' AND fi.quantity > 0 AND u.telegram_id != '' AND u.status = 1 AND u.role < 10" + levelWhere + " ORDER BY fi.quantity DESC, fi.telegram_id ASC"
			err = DB.Raw(query, levelArgs...).Scan(&entries).Error
		}
	case "steal":
		query := "SELECT fl.telegram_id, " + labelExpr + " as username, MAX(fl.amount) as value FROM tg_farm_logs fl JOIN users u ON fl.telegram_id = u.telegram_id" + levelJoin + "WHERE fl.action = 'steal' AND fl.amount > 0 AND u.telegram_id != '' AND u.status = 1 AND u.role < 10"
		args := make([]any, 0, len(levelArgs)+1)
		if period == farmLeaderboardPeriodWeekly {
			query += " AND fl.created_at >= ?"
			args = append(args, since)
		}
		query += levelWhere + " GROUP BY fl.telegram_id, " + labelExpr + " ORDER BY value DESC, fl.telegram_id ASC"
		args = append(args, levelArgs...)
		err = DB.Raw(query, args...).Scan(&entries).Error
	default:
		return []FarmLeaderboardEntry{}, nil
	}
	if err != nil {
		return entries, err
	}
	return filterFarmLeaderboardEntriesByCohort(entries, cohortIds), nil
}

func getFriendFarmLeaderboardEntries(boardType, period string, userId int, group string, cohortIds []string) ([]FarmLeaderboardEntry, error) {
	users, nameMap, farmIds, err := getFarmLeaderboardFriendUsers(userId, group)
	if err != nil {
		return nil, err
	}
	// 先按 cohortIds 裁剪好友列表，减少后续 SQL 压力
	if cohortIds != nil {
		allow := make(map[string]struct{}, len(cohortIds))
		for _, id := range cohortIds {
			allow[id] = struct{}{}
		}
		filteredUsers := users[:0]
		filteredFarmIds := farmIds[:0]
		filteredNameMap := make(map[string]string, len(nameMap))
		for i, user := range users {
			farmId := farmIds[i]
			if _, ok := allow[farmId]; !ok {
				continue
			}
			filteredUsers = append(filteredUsers, user)
			filteredFarmIds = append(filteredFarmIds, farmId)
			filteredNameMap[farmId] = nameMap[farmId]
		}
		users, farmIds, nameMap = filteredUsers, filteredFarmIds, filteredNameMap
	}
	if len(users) == 0 || len(farmIds) == 0 {
		return []FarmLeaderboardEntry{}, nil
	}
	if boardType == "balance" && period == farmLeaderboardPeriodAll {
		entries := make([]FarmLeaderboardEntry, 0, len(users))
		for _, user := range users {
			if user.Quota <= 0 {
				continue
			}
			entries = append(entries, FarmLeaderboardEntry{
				TelegramId: resolveFarmLeaderboardFarmId(user),
				Username:   resolveFarmLeaderboardUserName(user),
				Value:      int64(user.Quota),
			})
		}
		sortFarmLeaderboardEntries(entries)
		return entries, nil
	}
	var rows []farmLeaderboardValueRow
	since := getFarmLeaderboardWeekStart()
	switch boardType {
	case "balance":
		err = DB.Model(&TgFarmLog{}).
			Select("telegram_id, COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) as value").
			Where("telegram_id IN ? AND created_at >= ?", farmIds, since).
			Group("telegram_id").
			Having("COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) > 0").
			Scan(&rows).Error
	case "level":
		if period == farmLeaderboardPeriodWeekly {
			err = DB.Model(&TgFarmLog{}).
				Select("telegram_id, COUNT(*) as value").
				Where("action = ? AND telegram_id IN ? AND created_at >= ?", "levelup", farmIds, since).
				Group("telegram_id").
				Scan(&rows).Error
		} else {
			err = DB.Model(&TgFarmItem{}).
				Select("telegram_id, quantity as value").
				Where("item_type = ? AND quantity > 1 AND telegram_id IN ?", "_level", farmIds).
				Scan(&rows).Error
		}
	case "harvest":
		query := DB.Model(&TgFarmLog{}).
			Select("telegram_id, MAX(amount) as value").
			Where("action = ? AND amount > 0 AND telegram_id IN ?", "harvest", farmIds)
		if period == farmLeaderboardPeriodWeekly {
			query = query.Where("created_at >= ?", since)
		}
		err = query.Group("telegram_id").Scan(&rows).Error
	case "prestige":
		if period == farmLeaderboardPeriodWeekly {
			err = DB.Model(&TgFarmLog{}).
				Select("telegram_id, COUNT(*) as value").
				Where("action = ? AND telegram_id IN ? AND created_at >= ?", "prestige", farmIds, since).
				Group("telegram_id").
				Scan(&rows).Error
		} else {
			err = DB.Model(&TgFarmItem{}).
				Select("telegram_id, quantity as value").
				Where("item_type = ? AND quantity > 0 AND telegram_id IN ?", "_prestige", farmIds).
				Scan(&rows).Error
		}
	case "steal":
		query := DB.Model(&TgFarmLog{}).
			Select("telegram_id, MAX(amount) as value").
			Where("action = ? AND amount > 0 AND telegram_id IN ?", "steal", farmIds)
		if period == farmLeaderboardPeriodWeekly {
			query = query.Where("created_at >= ?", since)
		}
		err = query.Group("telegram_id").Scan(&rows).Error
	default:
		return []FarmLeaderboardEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	entries := buildFarmLeaderboardEntriesFromRows(rows, nameMap)
	sortFarmLeaderboardEntries(entries)
	return entries, nil
}

func getFarmLeaderboardEntriesWithOptions(boardType string, options FarmLeaderboardOptions) ([]FarmLeaderboardEntry, error) {
	options.Scope = normalizeFarmLeaderboardScope(options.Scope)
	options.Period = normalizeFarmLeaderboardPeriod(options.Period)
	options.Group = normalizeFarmLeaderboardGroup(options.Group)
	options.Cohort = normalizeFarmLeaderboardCohort(options.Cohort)
	if options.Scope == farmLeaderboardScopeFriends && options.UserId == 0 {
		options.Scope = farmLeaderboardScopeGlobal
	}
	cacheKey := boardType + "|" + options.Scope + "|" + options.Period + "|" + options.Group + "|" + options.Cohort
	if options.Scope == farmLeaderboardScopeFriends {
		cacheKey += "|" + strconv.Itoa(options.UserId)
	}
	now := time.Now().Unix()
	if cached, ok := farmLeaderboardCache.Load(cacheKey); ok {
		entry := cached.(farmLeaderboardCacheEntry)
		if entry.ExpiresAt >= now {
			return copyFarmLeaderboardEntries(entry.Value), nil
		}
		farmLeaderboardCache.Delete(cacheKey)
	}
	// cohort 过滤：预取属于指定 cohort 的 telegram_id 列表。
	// cohortIds == nil → 不过滤；len == 0 → 指定 cohort 无人 → 直接返回空。
	cohortIds, err := getFarmLeaderboardCohortIds(options.Cohort)
	if err != nil {
		return nil, err
	}
	if cohortIds != nil && len(cohortIds) == 0 {
		// 该 cohort 明确没有玩家（比如分水岭刚设置、新玩家还没出现）
		farmLeaderboardCache.Store(cacheKey, farmLeaderboardCacheEntry{Value: []FarmLeaderboardEntry{}, ExpiresAt: now + farmLeaderboardTTLSeconds})
		return []FarmLeaderboardEntry{}, nil
	}
	var entries []FarmLeaderboardEntry
	if options.Scope == farmLeaderboardScopeFriends {
		entries, err = getFriendFarmLeaderboardEntries(boardType, options.Period, options.UserId, options.Group, cohortIds)
	} else {
		entries, err = getGlobalFarmLeaderboardEntries(boardType, options.Period, options.Group, cohortIds)
	}
	if err != nil {
		return nil, err
	}
	copyForCache := copyFarmLeaderboardEntries(entries)
	farmLeaderboardCache.Store(cacheKey, farmLeaderboardCacheEntry{Value: copyForCache, ExpiresAt: now + farmLeaderboardTTLSeconds})
	return entries, nil
}

func GetFarmLeaderboardWithOptions(boardType string, limit int, options FarmLeaderboardOptions) ([]FarmLeaderboardEntry, error) {
	entries, err := getFarmLeaderboardEntriesWithOptions(boardType, options)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return copyFarmLeaderboardEntries(entries), nil
}

func GetFarmLeaderboard(boardType string, limit int) ([]FarmLeaderboardEntry, error) {
	return GetFarmLeaderboardWithOptions(boardType, limit, FarmLeaderboardOptions{})
}

func GetFarmRankWithOptions(telegramId, boardType string, options FarmLeaderboardOptions) int64 {
	options.Scope = normalizeFarmLeaderboardScope(options.Scope)
	options.Period = normalizeFarmLeaderboardPeriod(options.Period)
	options.Group = normalizeFarmLeaderboardGroup(options.Group)
	options.Cohort = normalizeFarmLeaderboardCohort(options.Cohort)
	if options.Scope == farmLeaderboardScopeFriends && options.UserId == 0 {
		options.Scope = farmLeaderboardScopeGlobal
	}
	cacheKey := telegramId + "|" + boardType + "|" + options.Scope + "|" + options.Period + "|" + options.Group + "|" + options.Cohort
	if options.Scope == farmLeaderboardScopeFriends {
		cacheKey += "|" + strconv.Itoa(options.UserId)
	}
	now := time.Now().Unix()
	if cached, ok := farmRankCache.Load(cacheKey); ok {
		entry := cached.(farmRankCacheEntry)
		if entry.ExpiresAt >= now {
			return entry.Value
		}
		farmRankCache.Delete(cacheKey)
	}
	entries, err := getFarmLeaderboardEntriesWithOptions(boardType, options)
	if err != nil {
		return 0
	}
	var rank int64
	for i, entry := range entries {
		if entry.TelegramId == telegramId {
			rank = int64(i + 1)
			break
		}
	}
	farmRankCache.Store(cacheKey, farmRankCacheEntry{Value: rank, ExpiresAt: now + farmRankTTLSeconds})
	return rank
	}

func GetFarmRank(telegramId, boardType string) int64 {
	return GetFarmRankWithOptions(telegramId, boardType, FarmLeaderboardOptions{})
	}

func GetFarmLeaderboardContextWithOptions(telegramId, boardType string, radius int, options FarmLeaderboardOptions) ([]FarmLeaderboardRankedEntry, int64, error) {
	if radius < 0 {
		radius = 0
	}
	entries, err := getFarmLeaderboardEntriesWithOptions(boardType, options)
	if err != nil {
		return nil, 0, err
	}
	myIndex := -1
	for i, entry := range entries {
		if entry.TelegramId == telegramId {
			myIndex = i
			break
		}
	}
	if myIndex < 0 {
		return []FarmLeaderboardRankedEntry{}, 0, nil
	}
	start := myIndex - radius
	if start < 0 {
		start = 0
	}
	end := myIndex + radius + 1
	if end > len(entries) {
		end = len(entries)
	}
	nearby := make([]FarmLeaderboardRankedEntry, 0, end-start)
	for i := start; i < end; i++ {
		nearby = append(nearby, FarmLeaderboardRankedEntry{
			Rank:                 int64(i + 1),
			FarmLeaderboardEntry: entries[i],
		})
	}
	return nearby, int64(myIndex + 1), nil
	}

func GetFarmLeaderboardContext(telegramId, boardType string, radius int) ([]FarmLeaderboardRankedEntry, int64, error) {
	return GetFarmLeaderboardContextWithOptions(telegramId, boardType, radius, FarmLeaderboardOptions{})
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
