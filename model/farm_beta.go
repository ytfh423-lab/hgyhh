package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
)

// FarmBetaReservation 农场内测预约记录
type FarmBetaReservation struct {
	Id                  int   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId              int   `json:"user_id" gorm:"uniqueIndex"`
	ReservedAt          int64 `json:"reserved_at" gorm:"autoCreateTime"`
	AgreementAcceptedAt int64 `json:"agreement_accepted_at" gorm:"default:0"`
}

func (FarmBetaReservation) TableName() string {
	return "farm_beta_reservations"
}

// CreateFarmBetaReservation creates a reservation for a user
func CreateFarmBetaReservation(userId int) error {
	reservation := FarmBetaReservation{
		UserId:     userId,
		ReservedAt: time.Now().Unix(),
	}
	return DB.Create(&reservation).Error
}

// GetFarmBetaReservation checks if a user has reserved
func GetFarmBetaReservation(userId int) (*FarmBetaReservation, error) {
	var reservation FarmBetaReservation
	err := DB.Where("user_id = ?", userId).First(&reservation).Error
	if err != nil {
		return nil, err
	}
	return &reservation, nil
}

// CountFarmBetaReservations returns total number of reservations
func CountFarmBetaReservations() (int64, error) {
	var count int64
	err := DB.Model(&FarmBetaReservation{}).Count(&count).Error
	return count, err
}

// HasFarmBetaAccess checks if a user has beta access (reserved within max slots)
func HasFarmBetaAccess(userId int) bool {
	maxSlots := common.FarmBetaMaxSlots
	if maxSlots <= 0 {
		return false
	}
	var rank int64
	// Count how many reservations were made before or at the same time as this user's
	err := DB.Model(&FarmBetaReservation{}).
		Where("id <= (SELECT id FROM farm_beta_reservations WHERE user_id = ?)", userId).
		Count(&rank).Error
	if err != nil {
		return false
	}
	return rank > 0 && rank <= int64(maxSlots)
}

// GetUserBetaRank returns the user's reservation rank (0 = not reserved)
func GetUserBetaRank(userId int) int64 {
	var reservation FarmBetaReservation
	err := DB.Where("user_id = ?", userId).First(&reservation).Error
	if err != nil {
		return 0
	}
	var rank int64
	DB.Model(&FarmBetaReservation{}).Where("id <= ?", reservation.Id).Count(&rank)
	return rank
}

// HasAcceptedBetaAgreement checks if the user has accepted the beta agreement
func HasAcceptedBetaAgreement(userId int) bool {
	var reservation FarmBetaReservation
	err := DB.Where("user_id = ?", userId).First(&reservation).Error
	if err != nil {
		return false
	}
	return reservation.AgreementAcceptedAt > 0
}

// AcceptBetaAgreement marks the user as having accepted the beta agreement
func AcceptBetaAgreement(userId int) error {
	return DB.Model(&FarmBetaReservation{}).Where("user_id = ?", userId).
		Update("agreement_accepted_at", time.Now().Unix()).Error
}

// ========== 内测资格申请 ==========

// FarmBetaApplication 农场内测资格申请记录
type FarmBetaApplication struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId           int    `json:"user_id" gorm:"index"`
	Reason           string `json:"reason" gorm:"type:text"`
	LinuxdoProfile   string `json:"linuxdo_profile" gorm:"type:varchar(512);default:''"`
	Status           string `json:"status" gorm:"type:varchar(20);default:'pending';index"` // pending / approved / rejected
	SubmittedAt      int64  `json:"submitted_at"`
	ReviewedAt       int64  `json:"reviewed_at" gorm:"default:0"`
	ReviewedBy       int    `json:"reviewed_by" gorm:"default:0"`
	ReviewNote       string `json:"review_note" gorm:"type:text"`
	ApplicationRound int     `json:"application_round" gorm:"default:1"`
	AiDecision       string  `json:"ai_decision" gorm:"type:varchar(20)"`
	AiConfidence     float64 `json:"ai_confidence" gorm:"default:0"`
	AiSummary        string  `json:"ai_summary" gorm:"type:text"`
	AiReviewLogId    int     `json:"ai_review_log_id" gorm:"default:0"`
}

func (FarmBetaApplication) TableName() string {
	return "farm_beta_applications"
}

// CreateBetaApplication 创建申请
func CreateBetaApplication(app *FarmBetaApplication) error {
	app.SubmittedAt = time.Now().Unix()
	return DB.Create(app).Error
}

// GetLatestBetaApplication 获取用户最新一条申请
func GetLatestBetaApplication(userId int) (*FarmBetaApplication, error) {
	var app FarmBetaApplication
	err := DB.Where("user_id = ?", userId).Order("id desc").First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetBetaApplicationById 按 ID 获取申请
func GetBetaApplicationById(id int) (*FarmBetaApplication, error) {
	var app FarmBetaApplication
	err := DB.Where("id = ?", id).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// CountUserBetaApplications 统计用户申请次数
func CountUserBetaApplications(userId int) int64 {
	var count int64
	DB.Model(&FarmBetaApplication{}).Where("user_id = ?", userId).Count(&count)
	return count
}

// UpdateBetaApplicationFields 更新申请字段
func UpdateBetaApplicationFields(id int, fields map[string]interface{}) error {
	return DB.Model(&FarmBetaApplication{}).Where("id = ?", id).Updates(fields).Error
}

// GetBetaApplicationList 管理员获取申请列表（分页+筛选）
func GetBetaApplicationList(page, pageSize int, status string) ([]*FarmBetaApplication, int64, error) {
	var apps []*FarmBetaApplication
	var total int64
	query := DB.Model(&FarmBetaApplication{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)
	err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&apps).Error
	return apps, total, err
}

// GetUserBetaApplicationHistory 获取用户所有申请记录
func GetUserBetaApplicationHistory(userId int) ([]*FarmBetaApplication, error) {
	var apps []*FarmBetaApplication
	err := DB.Where("user_id = ?", userId).Order("id desc").Find(&apps).Error
	return apps, err
}

// HasApprovedBetaApplication 检查用户是否有已通过的申请
func HasApprovedBetaApplication(userId int) bool {
	var count int64
	DB.Model(&FarmBetaApplication{}).Where("user_id = ? AND status = 'approved'", userId).Count(&count)
	return count > 0
}

// GrantBetaAccessViaApplication 通过申请发放资格（创建预约记录）
func GrantBetaAccessViaApplication(userId int) error {
	// 检查是否已有预约记录
	existing, err := GetFarmBetaReservation(userId)
	if err == nil && existing != nil {
		// 已有预约，确保 agreement 已接受
		if existing.AgreementAcceptedAt == 0 {
			return DB.Model(&FarmBetaReservation{}).Where("user_id = ?", userId).
				Update("agreement_accepted_at", time.Now().Unix()).Error
		}
		return nil // 幂等
	}
	// 创建新预约记录并自动接受协议
	reservation := FarmBetaReservation{
		UserId:              userId,
		ReservedAt:          time.Now().Unix(),
		AgreementAcceptedAt: time.Now().Unix(),
	}
	return DB.Create(&reservation).Error
}
