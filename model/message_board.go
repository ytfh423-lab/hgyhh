package model

import (
	"time"
)

// MessageBoardPost 留言板帖子
type MessageBoardPost struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"index"`
	Username    string `json:"username" gorm:"type:varchar(64)"`
	Title       string `json:"title" gorm:"type:varchar(128);not null"`
	Content     string `json:"content" gorm:"type:text;not null"`
	Category    string `json:"category" gorm:"type:varchar(16);index;default:'other'"` // bug / suggestion / feedback / other
	ContactInfo string `json:"contact_info" gorm:"type:varchar(128)"`
	Status      string `json:"status" gorm:"type:varchar(16);index;default:'pending'"` // pending / viewed / processing / resolved / rejected
	IsPublic    bool   `json:"is_public" gorm:"default:false"`
	AdminReply  string `json:"admin_reply" gorm:"type:text"`
	AdminNote   string `json:"admin_note" gorm:"type:text"`
	ResolvedBy  int    `json:"resolved_by" gorm:"default:0"`
	ResolvedAt  int64  `json:"resolved_at" gorm:"default:0"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (MessageBoardPost) TableName() string {
	return "message_board_posts"
}

// ========== 创建 ==========

func CreateMessageBoardPost(post *MessageBoardPost) error {
	return DB.Create(post).Error
}

// ========== 查询 ==========

func GetMessageBoardPostById(id int) (*MessageBoardPost, error) {
	var post MessageBoardPost
	err := DB.Where("id = ?", id).First(&post).Error
	return &post, err
}

// GetMessageBoardPostsByUserId 获取用户自己的留言 (分页)
func GetMessageBoardPostsByUserId(userId int, page, pageSize int) ([]*MessageBoardPost, int64, error) {
	var posts []*MessageBoardPost
	var total int64
	tx := DB.Model(&MessageBoardPost{}).Where("user_id = ?", userId)
	tx.Count(&total)
	err := tx.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, total, err
}

// GetPublicMessageBoardPosts 获取公开留言 (分页, 可按 category 筛选)
func GetPublicMessageBoardPosts(category string, page, pageSize int) ([]*MessageBoardPost, int64, error) {
	var posts []*MessageBoardPost
	var total int64
	tx := DB.Model(&MessageBoardPost{}).Where("is_public = ?", true)
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	tx.Count(&total)
	err := tx.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, total, err
}

// GetAllMessageBoardPosts 管理员获取所有留言 (分页 + 筛选)
func GetAllMessageBoardPosts(status, category, keyword string, userId, page, pageSize int) ([]*MessageBoardPost, int64, error) {
	var posts []*MessageBoardPost
	var total int64
	tx := DB.Model(&MessageBoardPost{})
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	if keyword != "" {
		kw := "%" + keyword + "%"
		tx = tx.Where("title LIKE ? OR content LIKE ?", kw, kw)
	}
	if userId > 0 {
		tx = tx.Where("user_id = ?", userId)
	}
	tx.Count(&total)
	err := tx.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, total, err
}

// ========== 更新 ==========

func UpdateMessageBoardPostStatus(id int, status string, adminId int) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if status == "resolved" || status == "rejected" {
		updates["resolved_by"] = adminId
		updates["resolved_at"] = time.Now().Unix()
	}
	return DB.Model(&MessageBoardPost{}).Where("id = ?", id).Updates(updates).Error
}

func UpdateMessageBoardPostAdminReply(id int, reply string) error {
	return DB.Model(&MessageBoardPost{}).Where("id = ?", id).Update("admin_reply", reply).Error
}

func UpdateMessageBoardPostAdminNote(id int, note string) error {
	return DB.Model(&MessageBoardPost{}).Where("id = ?", id).Update("admin_note", note).Error
}

func UpdateMessageBoardPostIsPublic(id int, isPublic bool) error {
	return DB.Model(&MessageBoardPost{}).Where("id = ?", id).Update("is_public", isPublic).Error
}

// ========== 防刷 ==========

// CountRecentPosts 统计用户最近N秒内的发帖数
func CountRecentMessageBoardPosts(userId int, seconds int64) int64 {
	var count int64
	since := time.Now().Unix() - seconds
	DB.Model(&MessageBoardPost{}).Where("user_id = ? AND created_at > ?", userId, since).Count(&count)
	return count
}
