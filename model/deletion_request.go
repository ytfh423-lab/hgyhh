package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
)

const (
	DeletionRequestStatusPending  = 0
	DeletionRequestStatusApproved = 1
	DeletionRequestStatusRejected = 2
)

type DeletionRequest struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"type:int;index;not null"`
	Username    string `json:"username" gorm:"type:varchar(64)"`
	Reason      string `json:"reason" gorm:"type:text;not null"`
	Status      int    `json:"status" gorm:"type:int;default:0;index"` // 0=pending, 1=approved, 2=rejected
	AdminId     int    `json:"admin_id" gorm:"type:int;default:0"`
	AdminRemark string `json:"admin_remark" gorm:"type:text"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func CreateDeletionRequest(userId int, username string, reason string) (*DeletionRequest, error) {
	// 检查是否已有待审核的注销申请
	var count int64
	DB.Model(&DeletionRequest{}).Where("user_id = ? AND status = ?", userId, DeletionRequestStatusPending).Count(&count)
	if count > 0 {
		return nil, errors.New("您已有待审核的注销申请，请勿重复提交")
	}

	req := &DeletionRequest{
		UserId:   userId,
		Username: username,
		Reason:   reason,
		Status:   DeletionRequestStatusPending,
	}
	err := DB.Create(req).Error
	return req, err
}

func GetDeletionRequests(page, pageSize int, status *int) ([]DeletionRequest, int64, error) {
	var requests []DeletionRequest
	var total int64

	query := DB.Model(&DeletionRequest{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	query.Count(&total)
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&requests).Error
	return requests, total, err
}

func GetPendingDeletionRequestByUserId(userId int) (*DeletionRequest, error) {
	var req DeletionRequest
	err := DB.Where("user_id = ? AND status = ?", userId, DeletionRequestStatusPending).First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func ApproveDeletionRequest(id int, adminId int, adminRemark string) error {
	var req DeletionRequest
	if err := DB.First(&req, id).Error; err != nil {
		return errors.New("注销申请不存在")
	}
	if req.Status != DeletionRequestStatusPending {
		return errors.New("该申请已被处理")
	}

	// 更新申请状态
	req.Status = DeletionRequestStatusApproved
	req.AdminId = adminId
	req.AdminRemark = adminRemark
	req.UpdatedAt = common.GetTimestamp()
	if err := DB.Save(&req).Error; err != nil {
		return err
	}

	// 硬删除用户
	return HardDeleteUserById(req.UserId)
}

func RejectDeletionRequest(id int, adminId int, adminRemark string) error {
	var req DeletionRequest
	if err := DB.First(&req, id).Error; err != nil {
		return errors.New("注销申请不存在")
	}
	if req.Status != DeletionRequestStatusPending {
		return errors.New("该申请已被处理")
	}

	req.Status = DeletionRequestStatusRejected
	req.AdminId = adminId
	req.AdminRemark = adminRemark
	req.UpdatedAt = common.GetTimestamp()
	return DB.Save(&req).Error
}
