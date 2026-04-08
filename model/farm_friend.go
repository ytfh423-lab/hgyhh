package model

import (
	"time"

	"gorm.io/gorm"
)

// FarmFriend 好友关系（单向，accepted 代表双方互为好友）
type FarmFriend struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId    int    `json:"user_id" gorm:"uniqueIndex:idx_farm_friend_pair"`
	FriendId  int    `json:"friend_id" gorm:"uniqueIndex:idx_farm_friend_pair"`
	Status    string `json:"status" gorm:"type:varchar(16);default:'pending'"` // pending/accepted/rejected
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

// FarmMessage 好友间站内消息
type FarmMessage struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	FromUserId int    `json:"from_user_id" gorm:"index:idx_farm_msg_from"`
	ToUserId   int    `json:"to_user_id" gorm:"index:idx_farm_msg_to"`
	Content    string `json:"content" gorm:"type:text"`
	IsRead     bool   `json:"is_read" gorm:"default:false"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
}

/* ────────── FarmFriend 操作 ────────── */

// SendFriendRequest 发送好友申请（不存在时创建，否则忽略）
func SendFriendRequest(fromUserId, toUserId int) error {
	req := &FarmFriend{
		UserId:   fromUserId,
		FriendId: toUserId,
		Status:   "pending",
	}
	return DB.Where(FarmFriend{UserId: fromUserId, FriendId: toUserId}).
		FirstOrCreate(req).Error
}

// AcceptFriendRequest 接受好友申请
func AcceptFriendRequest(requestId, toUserId int) error {
	var req FarmFriend
	if err := DB.Where("id = ? AND friend_id = ? AND status = ?", requestId, toUserId, "pending").First(&req).Error; err != nil {
		return err
	}
	// 更新申请状态为 accepted
	if err := DB.Model(&req).Update("status", "accepted").Error; err != nil {
		return err
	}
	// 建立反向关系（双向好友）
	reverse := &FarmFriend{
		UserId:   toUserId,
		FriendId: req.UserId,
		Status:   "accepted",
	}
	return DB.Where(FarmFriend{UserId: toUserId, FriendId: req.UserId}).
		Assign(FarmFriend{Status: "accepted"}).
		FirstOrCreate(reverse).Error
}

// RejectFriendRequest 拒绝好友申请
func RejectFriendRequest(requestId, toUserId int) error {
	return DB.Model(&FarmFriend{}).
		Where("id = ? AND friend_id = ? AND status = ?", requestId, toUserId, "pending").
		Update("status", "rejected").Error
}

// RemoveFriend 删除好友（双向）
func RemoveFriend(userId, friendId int) error {
	return DB.Where(
		"(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userId, friendId, friendId, userId,
	).Delete(&FarmFriend{}).Error
}

// GetFriendList 获取已接受的好友列表，返回对方的用户 ID 切片
func GetFriendList(userId int) ([]int, error) {
	var friends []FarmFriend
	if err := DB.Where("user_id = ? AND status = ?", userId, "accepted").Find(&friends).Error; err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(friends))
	for _, f := range friends {
		ids = append(ids, f.FriendId)
	}
	return ids, nil
}

// GetPendingRequests 获取发给 userId 的待处理好友申请
func GetPendingFriendRequests(userId int) ([]FarmFriend, error) {
	var reqs []FarmFriend
	err := DB.Where("friend_id = ? AND status = ?", userId, "pending").
		Order("created_at DESC").Limit(50).Find(&reqs).Error
	return reqs, err
}

// IsFriend 检查双方是否已是好友
func IsFriend(userId, friendId int) bool {
	var count int64
	DB.Model(&FarmFriend{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", userId, friendId, "accepted").
		Count(&count)
	return count > 0
}

// GetFriendRequestByUsers 获取两人之间的好友申请
func GetFriendRequestByUsers(fromUserId, toUserId int) (*FarmFriend, error) {
	var req FarmFriend
	err := DB.Where("user_id = ? AND friend_id = ?", fromUserId, toUserId).First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

/* ────────── FarmMessage 操作 ────────── */

// SaveFarmMessage 保存一条消息
func SaveFarmMessage(fromUserId, toUserId int, content string) (*FarmMessage, error) {
	msg := &FarmMessage{
		FromUserId: fromUserId,
		ToUserId:   toUserId,
		Content:    content,
	}
	if err := DB.Create(msg).Error; err != nil {
		return nil, err
	}
	return msg, nil
}

// GetFarmMessages 获取两人之间最近 N 条消息
func GetFarmMessages(userId, friendId int, limit int) ([]FarmMessage, error) {
	var msgs []FarmMessage
	err := DB.Where(
		"(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)",
		userId, friendId, friendId, userId,
	).Order("created_at DESC").Limit(limit).Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	// 反转为正序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	// 标记为已读
	DB.Model(&FarmMessage{}).
		Where("from_user_id = ? AND to_user_id = ? AND is_read = ?", friendId, userId, false).
		Update("is_read", true)
	return msgs, nil
}

// GetUnreadCount 获取 userId 未读消息数
func GetFarmUnreadCount(userId int) int64 {
	var count int64
	DB.Model(&FarmMessage{}).
		Where("to_user_id = ? AND is_read = ?", userId, false).
		Count(&count)
	return count
}

// CleanOldMessages 清理 30 天前的旧消息（可定期调用）
func CleanOldFarmMessages() {
	cutoff := time.Now().AddDate(0, 0, -30).Unix()
	DB.Where("created_at < ?", cutoff).Delete(&FarmMessage{})
}

/* ────────── 用于展示的联合查询 ────────── */

type FriendInfo struct {
	UserId      int    `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Online      bool   `json:"online"`
	UnreadCount int64  `json:"unread_count"`
	RequestId   int    `json:"request_id,omitempty"` // 用于好友申请列表
}

// GetFriendInfoList 获取好友信息列表（含展示名、在线状态）
func GetFriendInfoList(userId int, onlineChecker func(int) bool) ([]FriendInfo, error) {
	friendIds, err := GetFriendList(userId)
	if err != nil {
		return nil, err
	}
	return GetFriendInfoListByIds(userId, friendIds, onlineChecker)
}

func GetFriendInfoListByIds(userId int, friendIds []int, onlineChecker func(int) bool) ([]FriendInfo, error) {
	if len(friendIds) == 0 {
		return []FriendInfo{}, nil
	}
	var users []User
	if err := DB.Select("id, username, display_name").Where("id IN ?", friendIds).Find(&users).Error; err != nil {
		return nil, err
	}
	userMap := make(map[int]User, len(users))
	for _, u := range users {
		userMap[u.Id] = u
	}
	var unreadRows []struct {
		FromUserId int
		Count      int64
	}
	if err := DB.Model(&FarmMessage{}).
		Select("from_user_id, COUNT(*) as count").
		Where("to_user_id = ? AND is_read = ? AND from_user_id IN ?", userId, false, friendIds).
		Group("from_user_id").
		Scan(&unreadRows).Error; err != nil {
		return nil, err
	}
	unreadMap := make(map[int]int64, len(unreadRows))
	for _, row := range unreadRows {
		unreadMap[row.FromUserId] = row.Count
	}
	result := make([]FriendInfo, 0, len(friendIds))
	for _, fid := range friendIds {
		u := userMap[fid]
		result = append(result, FriendInfo{
			UserId:      fid,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Online:      onlineChecker(fid),
			UnreadCount: unreadMap[fid],
		})
	}
	return result, nil
}

// GetPendingRequestInfoList 获取待处理好友申请（含发起者信息）
func GetPendingRequestInfoList(userId int) ([]FriendInfo, error) {
	reqs, err := GetPendingFriendRequests(userId)
	if err != nil {
		return nil, err
	}
	if len(reqs) == 0 {
		return []FriendInfo{}, nil
	}
	fromIds := make([]int, 0, len(reqs))
	reqMap := make(map[int]FarmFriend)
	for _, r := range reqs {
		fromIds = append(fromIds, r.UserId)
		reqMap[r.UserId] = r
	}
	var users []User
	if err := DB.Select("id, username, display_name").Where("id IN ?", fromIds).Find(&users).Error; err != nil {
		return nil, err
	}
	result := make([]FriendInfo, 0, len(users))
	for _, u := range users {
		req := reqMap[u.Id]
		result = append(result, FriendInfo{
			UserId:      u.Id,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			RequestId:   req.Id,
		})
	}
	return result, nil
}

type FriendRelationState struct {
	IsFriend  bool
	ReqStatus string
}

func GetOutgoingFriendStatusMap(userId int, targetIds []int) (map[int]FriendRelationState, error) {
	result := make(map[int]FriendRelationState)
	if len(targetIds) == 0 {
		return result, nil
	}
	uniqueIds := make([]int, 0, len(targetIds))
	seen := make(map[int]struct{}, len(targetIds))
	for _, id := range targetIds {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqueIds = append(uniqueIds, id)
	}
	var rows []FarmFriend
	if err := DB.Select("friend_id, status").Where("user_id = ? AND friend_id IN ?", userId, uniqueIds).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		state := FriendRelationState{ReqStatus: row.Status}
		if row.Status == "accepted" {
			state.IsFriend = true
			state.ReqStatus = ""
		}
		result[row.FriendId] = state
	}
	return result, nil
}

// SearchUsers 按用户名搜索用户（排除自己和已有关系的用户）
func SearchFarmUsers(keyword string, excludeUserId int, limit int) ([]User, error) {
	var users []User
	err := DB.Select("id, username, display_name").
		Where("(username LIKE ? OR display_name LIKE ?) AND id != ? AND deleted_at IS NULL",
			"%"+keyword+"%", "%"+keyword+"%", excludeUserId).
		Limit(limit).Find(&users).Error
	return users, err
}

// GetFarmFriendDeleted 兼容软删除查询
func init() {
	// 注册软删除忽略（FarmFriend 不用软删除，直接物理删除）
	_ = gorm.ErrRecordNotFound // 保持 import
}
