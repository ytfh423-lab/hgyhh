package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	redisv8 "github.com/go-redis/redis/v8"
)

/* ═══════════════════════════════════════════════════════════════
   全站在线状态追踪（心跳 30s，3 分钟内算在线）
   ═══════════════════════════════════════════════════════════════ */

const (
	siteOnlineKey    = "site:online"
	siteOnlineTTLSec = 180 // 3 分钟
)

var siteOnlineMemory sync.Map // userId(int) -> int64(timestamp)

// SiteHeartbeat POST /api/heartbeat — 更新在线状态
func SiteHeartbeat(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false})
		return
	}
	siteOnlineHeartbeat(userId)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func siteOnlineHeartbeat(userId int) {
	now := time.Now().Unix()
	member := strconv.Itoa(userId)
	if common.RedisEnabled {
		ctx := context.Background()
		common.RDB.ZAdd(ctx, siteOnlineKey, &redisv8.Z{
			Score:  float64(now),
			Member: member,
		})
		cutoff := float64(now - siteOnlineTTLSec)
		common.RDB.ZRemRangeByScore(ctx, siteOnlineKey, "-inf",
			fmt.Sprintf("%f", cutoff))
	} else {
		siteOnlineMemory.Store(userId, now)
	}
}

func isSiteOnline(userId int) bool {
	now := time.Now().Unix()
	cutoff := now - siteOnlineTTLSec
	if common.RedisEnabled {
		ctx := context.Background()
		score, err := common.RDB.ZScore(ctx, siteOnlineKey, strconv.Itoa(userId)).Result()
		if err != nil {
			return false
		}
		return int64(score) >= cutoff
	}
	val, ok := siteOnlineMemory.Load(userId)
	if !ok {
		return false
	}
	ts, ok := val.(int64)
	return ok && ts >= cutoff
}

/* ═══════════════════════════════════════════════════════════════
   事件队列（用于实时通知：好友申请 / 农场邀请 / 聊天消息）
   ═══════════════════════════════════════════════════════════════ */

const (
	eventKeyPrefix = "farm:events:"
	eventMaxLen    = 50
	eventTTL       = 2 * time.Hour
)

type FarmEvent struct {
	Type      string                 `json:"type"` // friend_request / farm_invite / chat_message
	FromId    int                    `json:"from_id"`
	FromName  string                 `json:"from_name"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

var eventMemory sync.Map // userId(int) -> *[]FarmEvent (mutex protected via eventMemMu)
var eventMemMu sync.Mutex

func pushEvent(toUserId int, ev FarmEvent) {
	ev.Timestamp = time.Now().Unix()
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	if common.RedisEnabled {
		key := eventKeyPrefix + strconv.Itoa(toUserId)
		ctx := context.Background()
		common.RDB.LPush(ctx, key, string(data))
		common.RDB.LTrim(ctx, key, 0, int64(eventMaxLen-1))
		common.RDB.Expire(ctx, key, eventTTL)
	} else {
		eventMemMu.Lock()
		defer eventMemMu.Unlock()
		var list []FarmEvent
		if v, ok := eventMemory.Load(toUserId); ok {
			list = v.([]FarmEvent)
		}
		list = append([]FarmEvent{ev}, list...)
		if len(list) > eventMaxLen {
			list = list[:eventMaxLen]
		}
		eventMemory.Store(toUserId, list)
	}
}

func popEvents(userId int) []FarmEvent {
	if common.RedisEnabled {
		key := eventKeyPrefix + strconv.Itoa(userId)
		ctx := context.Background()
		strs, err := common.RDB.LRange(ctx, key, 0, int64(eventMaxLen-1)).Result()
		if err != nil || len(strs) == 0 {
			return nil
		}
		common.RDB.Del(ctx, key)
		events := make([]FarmEvent, 0, len(strs))
		for i := len(strs) - 1; i >= 0; i-- { // 正序（旧→新）
			var ev FarmEvent
			if json.Unmarshal([]byte(strs[i]), &ev) == nil {
				events = append(events, ev)
			}
		}
		return events
	}
	eventMemMu.Lock()
	defer eventMemMu.Unlock()
	v, ok := eventMemory.Load(userId)
	if !ok {
		return nil
	}
	list := v.([]FarmEvent)
	eventMemory.Delete(userId)
	// 正序
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list
}

/* ═══════════════════════════════════════════════════════════════
   GET /api/farm/events/poll — 轮询待处理事件
   ═══════════════════════════════════════════════════════════════ */

func WebFarmEventsPoll(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	// 顺便刷新在线心跳
	siteOnlineHeartbeat(userId)
	events := popEvents(userId)
	if events == nil {
		events = []FarmEvent{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"events": events}})
}

/* ═══════════════════════════════════════════════════════════════
   好友系统
   ═══════════════════════════════════════════════════════════════ */

// GET /api/farm/friends
func WebFarmFriendList(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	list, err := model.GetFriendInfoList(userId, isSiteOnline)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list})
}

// GET /api/farm/friends/requests
func WebFarmFriendRequests(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	list, err := model.GetPendingRequestInfoList(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list})
}

// POST /api/farm/friends/request  body: {friend_id: int}
func WebFarmFriendRequest(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	var req struct {
		FriendId int `json:"friend_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FriendId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.FriendId == userId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "不能添加自己为好友"})
		return
	}
	// 检查是否已是好友
	if model.IsFriend(userId, req.FriendId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "已经是好友了"})
		return
	}
	if err := model.SendFriendRequest(userId, req.FriendId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "发送失败"})
		return
	}
	// 推送事件通知对方（带申请 ID，方便前端直接接受）
	me, _ := model.GetUserById(userId, false)
	fromName := nameOf(me)
	friendReq, _ := model.GetFriendRequestByUsers(userId, req.FriendId)
	reqId := 0
	if friendReq != nil {
		reqId = friendReq.Id
	}
	pushEvent(req.FriendId, FarmEvent{
		Type:     "friend_request",
		FromId:   userId,
		FromName: fromName,
		Payload:  map[string]interface{}{"request_id": reqId},
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "好友申请已发送"})
}

// POST /api/farm/friends/respond  body: {request_id: int, action: "accept"|"reject"}
func WebFarmFriendRespond(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	var req struct {
		RequestId int    `json:"request_id"`
		Action    string `json:"action"` // accept / reject
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.RequestId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	switch req.Action {
	case "accept":
		if err := model.AcceptFriendRequest(req.RequestId, userId); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "操作失败"})
			return
		}
		// 通知对方申请被接受
		me, _ := model.GetUserById(userId, false)
		// 查出申请的 from user id
		var fr model.FarmFriend
		if model.DB.Where("id = ?", req.RequestId).First(&fr).Error == nil {
			pushEvent(fr.UserId, FarmEvent{
				Type:     "friend_accepted",
				FromId:   userId,
				FromName: nameOf(me),
			})
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "已接受好友申请"})
	case "reject":
		if err := model.RejectFriendRequest(req.RequestId, userId); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "操作失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "已拒绝好友申请"})
	default:
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "action 参数错误"})
	}
}

// DELETE /api/farm/friends/:friend_id
func WebFarmFriendRemove(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	friendIdStr := c.Param("friend_id")
	friendId, err := strconv.Atoi(friendIdStr)
	if err != nil || friendId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := model.RemoveFriend(userId, friendId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已删除好友"})
}

// GET /api/farm/friends/search?q=xxx
func WebFarmFriendSearch(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	q := strings.TrimSpace(c.Query("q"))
	if utf8.RuneCountInString(q) < 1 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请输入搜索关键词"})
		return
	}
	users, err := model.SearchFarmUsers(q, userId, 20)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "搜索失败"})
		return
	}
	result := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		// 检查是否已是好友或已有申请
		isFr := model.IsFriend(userId, u.Id)
		var reqStatus string
		if !isFr {
			existing, err2 := model.GetFriendRequestByUsers(userId, u.Id)
			if err2 == nil {
				reqStatus = existing.Status
			}
		}
		result = append(result, map[string]interface{}{
			"user_id":      u.Id,
			"username":     u.Username,
			"display_name": u.DisplayName,
			"is_friend":    isFr,
			"req_status":   reqStatus,
			"online":       isSiteOnline(u.Id),
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

/* ═══════════════════════════════════════════════════════════════
   农场邀请
   ═══════════════════════════════════════════════════════════════ */

// POST /api/farm/invite  body: {friend_id: int}
func WebFarmInviteFriend(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	var req struct {
		FriendId int `json:"friend_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FriendId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if !model.IsFriend(userId, req.FriendId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "对方不是你的好友"})
		return
	}
	if !isSiteOnline(req.FriendId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "好友当前不在线"})
		return
	}
	me, _ := model.GetUserById(userId, false)
	pushEvent(req.FriendId, FarmEvent{
		Type:     "farm_invite",
		FromId:   userId,
		FromName: nameOf(me),
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "邀请已发送"})
}

/* ═══════════════════════════════════════════════════════════════
   站内聊天
   ═══════════════════════════════════════════════════════════════ */

// GET /api/farm/chat/:friend_id
func WebFarmChatHistory(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	friendIdStr := c.Param("friend_id")
	friendId, err := strconv.Atoi(friendIdStr)
	if err != nil || friendId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if !model.IsFriend(userId, friendId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "只能和好友聊天"})
		return
	}
	msgs, err := model.GetFarmMessages(userId, friendId, 50)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": msgs})
}

// POST /api/farm/chat/:friend_id  body: {content: string}
func WebFarmChatSend(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	friendIdStr := c.Param("friend_id")
	friendId, err := strconv.Atoi(friendIdStr)
	if err != nil || friendId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if !model.IsFriend(userId, friendId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "只能和好友聊天"})
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	content := strings.TrimSpace(req.Content)
	if utf8.RuneCountInString(content) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "消息不能为空"})
		return
	}
	if utf8.RuneCountInString(content) > 300 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "消息不能超过 300 字"})
		return
	}
	msg, err := model.SaveFarmMessage(userId, friendId, content)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "发送失败"})
		return
	}
	me, _ := model.GetUserById(userId, false)
	// 推送实时事件给对方
	pushEvent(friendId, FarmEvent{
		Type:     "chat_message",
		FromId:   userId,
		FromName: nameOf(me),
		Payload: map[string]interface{}{
			"msg_id":     msg.Id,
			"content":    content,
			"created_at": msg.CreatedAt,
		},
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "data": msg})
}

/* ═══════════════════════════════════════════════════════════════
   辅助
   ═══════════════════════════════════════════════════════════════ */

func nameOf(u *model.User) string {
	if u == nil {
		return "未知用户"
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Username
}
