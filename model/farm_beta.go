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
