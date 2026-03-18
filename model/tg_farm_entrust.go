package model

import (
	"time"

	"gorm.io/gorm"
)

// ========== 委托任务系统 ==========

// TgFarmEntrust 委托任务主表
type TgFarmEntrust struct {
	Id              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	OwnerTelegramId string `json:"owner_telegram_id" gorm:"type:varchar(64);index"`
	Title           string `json:"title" gorm:"type:varchar(128)"`
	TargetAction    string `json:"target_action" gorm:"type:varchar(32)"`
	TargetModule    string `json:"target_module" gorm:"type:varchar(16)"`
	TargetItemKey   string `json:"target_item_key" gorm:"type:varchar(64)"`
	TargetCount     int    `json:"target_count"`
	ProgressCount   int    `json:"progress_count" gorm:"default:0"`
	RewardAmount    int    `json:"reward_amount"`
	EscrowStatus    string `json:"escrow_status" gorm:"type:varchar(24);default:'unpaid'"`
	Status          string `json:"status" gorm:"type:varchar(24);default:'published'"`
	IsPublic        int    `json:"is_public" gorm:"default:1"`
	MaxWorkerCount  int    `json:"max_worker_count" gorm:"default:1"`
	SettlementMode  string `json:"settlement_mode" gorm:"type:varchar(16);default:'partial'"`
	DeadlineAt      int64  `json:"deadline_at"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

// TgFarmEntrustWorker 接单记录
type TgFarmEntrustWorker struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskId           int    `json:"task_id" gorm:"index"`
	WorkerTelegramId string `json:"worker_telegram_id" gorm:"type:varchar(64);index"`
	Status           string `json:"status" gorm:"type:varchar(24);default:'accepted'"`
	ProgressCount    int    `json:"progress_count" gorm:"default:0"`
	RewardAmount     int    `json:"reward_amount" gorm:"default:0"`
	AcceptedAt       int64  `json:"accepted_at"`
	CompletedAt      int64  `json:"completed_at" gorm:"default:0"`
}

// TgFarmEntrustLog 委托操作日志（防重复计数）
type TgFarmEntrustLog struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskId           int    `json:"task_id" gorm:"index"`
	WorkerTelegramId string `json:"worker_telegram_id" gorm:"type:varchar(64)"`
	ActionType       string `json:"action_type" gorm:"type:varchar(32)"`
	TargetEntityId   int    `json:"target_entity_id"`
	ProgressDelta    int    `json:"progress_delta" gorm:"default:1"`
	CreatedAt        int64  `json:"created_at"`
}

// TgFarmEntrustEscrow 托管流水
type TgFarmEntrustEscrow struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskId           int    `json:"task_id" gorm:"index"`
	OwnerTelegramId  string `json:"owner_telegram_id" gorm:"type:varchar(64)"`
	WorkerTelegramId string `json:"worker_telegram_id" gorm:"type:varchar(64)"`
	Amount           int    `json:"amount"`
	Action           string `json:"action" gorm:"type:varchar(24)"`
	CreatedAt        int64  `json:"created_at"`
}

// ========== Entrust CRUD ==========

func CreateEntrust(e *TgFarmEntrust) error {
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = e.CreatedAt
	return DB.Create(e).Error
}

func GetEntrustById(id int) (*TgFarmEntrust, error) {
	var e TgFarmEntrust
	err := DB.Where("id = ?", id).First(&e).Error
	return &e, err
}

func GetPublishedEntrusts(page, pageSize int, module, action string) ([]*TgFarmEntrust, int64, error) {
	var tasks []*TgFarmEntrust
	var total int64
	now := time.Now().Unix()
	query := DB.Model(&TgFarmEntrust{}).Where("status IN ('published','in_progress') AND is_public = 1 AND deadline_at > ? AND escrow_status = 'escrow_success'", now)
	if module != "" {
		query = query.Where("target_module = ?", module)
	}
	if action != "" {
		query = query.Where("target_action = ?", action)
	}
	query.Count(&total)
	err := query.Order("reward_amount desc, created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error
	return tasks, total, err
}

func GetEntrustsByOwner(tgId string) ([]*TgFarmEntrust, error) {
	var tasks []*TgFarmEntrust
	err := DB.Where("owner_telegram_id = ?", tgId).Order("created_at desc").Limit(50).Find(&tasks).Error
	return tasks, err
}

func UpdateEntrustFields(id int, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now().Unix()
	return DB.Model(&TgFarmEntrust{}).Where("id = ?", id).Updates(fields).Error
}

func IncrementEntrustProgress(id int, delta int) (int, int, error) {
	err := DB.Model(&TgFarmEntrust{}).Where("id = ?", id).Updates(map[string]interface{}{
		"progress_count": gorm.Expr("progress_count + ?", delta),
		"updated_at":     time.Now().Unix(),
	}).Error
	if err != nil {
		return 0, 0, err
	}
	var task TgFarmEntrust
	DB.Where("id = ?", id).Select("progress_count, target_count").First(&task)
	return task.ProgressCount, task.TargetCount, nil
}

// ========== Worker CRUD ==========

func CreateEntrustWorker(w *TgFarmEntrustWorker) error {
	w.AcceptedAt = time.Now().Unix()
	return DB.Create(w).Error
}

func GetEntrustWorker(taskId int, workerTgId string) (*TgFarmEntrustWorker, error) {
	var w TgFarmEntrustWorker
	err := DB.Where("task_id = ? AND worker_telegram_id = ? AND status IN ('accepted','working')", taskId, workerTgId).First(&w).Error
	return &w, err
}

func GetEntrustWorkers(taskId int) ([]*TgFarmEntrustWorker, error) {
	var workers []*TgFarmEntrustWorker
	err := DB.Where("task_id = ?", taskId).Order("accepted_at desc").Find(&workers).Error
	return workers, err
}

func CountActiveEntrustWorkers(taskId int) int64 {
	var count int64
	DB.Model(&TgFarmEntrustWorker{}).Where("task_id = ? AND status IN ('accepted','working')", taskId).Count(&count)
	return count
}

func GetMyAcceptedEntrusts(tgId string) ([]*TgFarmEntrustWorker, error) {
	var workers []*TgFarmEntrustWorker
	err := DB.Where("worker_telegram_id = ? AND status IN ('accepted','working')", tgId).Order("accepted_at desc").Limit(50).Find(&workers).Error
	return workers, err
}

func GetAllMyEntrustWorkers(tgId string) ([]*TgFarmEntrustWorker, error) {
	var workers []*TgFarmEntrustWorker
	err := DB.Where("worker_telegram_id = ?", tgId).Order("accepted_at desc").Limit(50).Find(&workers).Error
	return workers, err
}

func UpdateEntrustWorkerFields(id int, fields map[string]interface{}) error {
	return DB.Model(&TgFarmEntrustWorker{}).Where("id = ?", id).Updates(fields).Error
}

func IncrementWorkerProgress(id int, delta int) error {
	return DB.Model(&TgFarmEntrustWorker{}).Where("id = ?", id).Updates(map[string]interface{}{
		"progress_count": gorm.Expr("progress_count + ?", delta),
	}).Error
}

// ========== Log (防重复计数) ==========

func CreateEntrustLog(log *TgFarmEntrustLog) error {
	log.CreatedAt = time.Now().Unix()
	return DB.Create(log).Error
}

func HasEntrustLogForEntity(taskId int, workerTgId string, entityId int) bool {
	var count int64
	DB.Model(&TgFarmEntrustLog{}).Where("task_id = ? AND worker_telegram_id = ? AND target_entity_id = ?", taskId, workerTgId, entityId).Count(&count)
	return count > 0
}

// ========== Escrow 托管流水 ==========

func CreateEntrustEscrow(e *TgFarmEntrustEscrow) error {
	e.CreatedAt = time.Now().Unix()
	return DB.Create(e).Error
}

func GetEntrustEscrowLogs(taskId int) ([]*TgFarmEntrustEscrow, error) {
	var logs []*TgFarmEntrustEscrow
	err := DB.Where("task_id = ?", taskId).Order("created_at desc").Find(&logs).Error
	return logs, err
}

func GetEntrustSettledAmount(taskId int) int {
	var total struct{ Sum int }
	DB.Model(&TgFarmEntrustEscrow{}).Where("task_id = ? AND action = 'settle'", taskId).Select("COALESCE(SUM(amount),0) as sum").Scan(&total)
	return total.Sum
}

// ========== 每日限制计数 ==========

func CountTodayEntrustPublished(tgId string) int64 {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	var count int64
	DB.Model(&TgFarmEntrust{}).Where("owner_telegram_id = ? AND created_at >= ?", tgId, startOfDay).Count(&count)
	return count
}

func CountTodayEntrustAccepted(tgId string) int64 {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	var count int64
	DB.Model(&TgFarmEntrustWorker{}).Where("worker_telegram_id = ? AND accepted_at >= ?", tgId, startOfDay).Count(&count)
	return count
}

func CountTodaySettleBetween(tgId1, tgId2 string) int64 {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	var count int64
	DB.Model(&TgFarmEntrustEscrow{}).Where(
		"((owner_telegram_id = ? AND worker_telegram_id = ?) OR (owner_telegram_id = ? AND worker_telegram_id = ?)) AND action = 'settle' AND created_at >= ?",
		tgId1, tgId2, tgId2, tgId1, startOfDay,
	).Count(&count)
	return count
}

// ========== 过期清理 ==========

func CleanupExpiredEntrusts() int64 {
	now := time.Now().Unix()
	result := DB.Model(&TgFarmEntrust{}).
		Where("status IN ('published','in_progress') AND deadline_at <= ?", now).
		Updates(map[string]interface{}{"status": "expired", "updated_at": now})
	return result.RowsAffected
}

func GetExpiredUnrefundedEntrusts() ([]*TgFarmEntrust, error) {
	var tasks []*TgFarmEntrust
	err := DB.Where("status = 'expired' AND escrow_status = 'escrow_success'").Find(&tasks).Error
	return tasks, err
}

func MarkEntrustEscrowRefunded(id int) error {
	return DB.Model(&TgFarmEntrust{}).Where("id = ?", id).
		Updates(map[string]interface{}{"escrow_status": "refunded", "updated_at": time.Now().Unix()}).Error
}

// ========== 辅助：通过 tgId 找 User ==========

func GetUserByFarmId(farmId string) (*User, error) {
	var user User
	if len(farmId) > 2 && farmId[:2] == "u_" {
		// u_{userId} 格式
		err := DB.Where("id = ?", farmId[2:]).First(&user).Error
		return &user, err
	}
	// telegram_id 格式
	err := DB.Where("telegram_id = ?", farmId).First(&user).Error
	return &user, err
}
