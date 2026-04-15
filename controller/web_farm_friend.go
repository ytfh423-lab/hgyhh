package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	redisv8 "github.com/go-redis/redis/v8"
)

/* ═══════════════════════════════════════════════════════════════
   全站在线状态追踪（心跳 30s，3 分钟内算在线）
   ═══════════════════════════════════════════════════════════════ */

const (
	siteOnlineKey    = "site:online"
	siteOnlineTTLSec = 180 // 3 分钟
	siteHeartbeatMinIntervalSec = 15
	siteCleanupMinIntervalSec = 30
)

var siteOnlineMemory sync.Map // userId(int) -> int64(timestamp)
var siteOnlineWriteMemory sync.Map
var siteOnlineLastCleanupUnix int64

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
	if last, ok := siteOnlineWriteMemory.Load(userId); ok {
		if lastTs, ok := last.(int64); ok && now-lastTs < siteHeartbeatMinIntervalSec {
			return
		}
	}
	siteOnlineWriteMemory.Store(userId, now)
	member := strconv.Itoa(userId)
	if common.RedisEnabled {
		ctx := context.Background()
		common.RDB.ZAdd(ctx, siteOnlineKey, &redisv8.Z{
			Score:  float64(now),
			Member: member,
		})
		lastCleanup := atomic.LoadInt64(&siteOnlineLastCleanupUnix)
		if now-lastCleanup >= siteCleanupMinIntervalSec && atomic.CompareAndSwapInt64(&siteOnlineLastCleanupUnix, lastCleanup, now) {
			cutoff := float64(now - siteOnlineTTLSec)
			common.RDB.ZRemRangeByScore(ctx, siteOnlineKey, "-inf",
				fmt.Sprintf("%f", cutoff))
		}
	} else {
		siteOnlineMemory.Store(userId, now)
	}
}

// getOnlineUserIds 返回当前在线的所有用户 ID（去除自己）
func getOnlineUserIds(excludeId int) []int {
	now := time.Now().Unix()
	cutoff := now - siteOnlineTTLSec
	var ids []int
	if common.RedisEnabled {
		ctx := context.Background()
		members, err := common.RDB.ZRangeByScore(ctx, siteOnlineKey, &redisv8.ZRangeBy{
			Min: strconv.FormatInt(cutoff, 10),
			Max: "+inf",
		}).Result()
		if err != nil {
			return ids
		}
		for _, m := range members {
			id, err := strconv.Atoi(m)
			if err == nil && id != excludeId {
				ids = append(ids, id)
			}
		}
	} else {
		siteOnlineMemory.Range(func(key, value interface{}) bool {
			ts, ok := value.(int64)
			if !ok || ts < cutoff {
				siteOnlineMemory.Delete(key)
				return true
			}
			id, ok := key.(int)
			if ok && id != excludeId {
				ids = append(ids, id)
			}
			return true
		})
	}
	return ids
}

func getOnlineUserIdsLimited(excludeId int, limit int) []int {
	if limit <= 0 {
		return getOnlineUserIds(excludeId)
	}
	now := time.Now().Unix()
	cutoff := now - siteOnlineTTLSec
	ids := make([]int, 0, limit)
	if common.RedisEnabled {
		ctx := context.Background()
		members, err := common.RDB.ZRevRangeByScore(ctx, siteOnlineKey, &redisv8.ZRangeBy{
			Min:   strconv.FormatInt(cutoff, 10),
			Max:   "+inf",
			Count: int64(limit + 1),
		}).Result()
		if err != nil {
			return ids
		}
		for _, m := range members {
			id, err := strconv.Atoi(m)
			if err == nil && id != excludeId {
				ids = append(ids, id)
				if len(ids) >= limit {
					break
				}
			}
		}
		return ids
	}
	siteOnlineMemory.Range(func(key, value interface{}) bool {
		ts, ok := value.(int64)
		if !ok || ts < cutoff {
			siteOnlineMemory.Delete(key)
			return true
		}
		id, ok := key.(int)
		if ok && id != excludeId {
			ids = append(ids, id)
			if len(ids) >= limit {
				return false
			}
		}
		return true
	})
	return ids
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

func getSiteOnlineStatusMap(userIds []int) map[int]bool {
	result := make(map[int]bool, len(userIds))
	if len(userIds) == 0 {
		return result
	}
	now := time.Now().Unix()
	cutoff := now - siteOnlineTTLSec
	if common.RedisEnabled {
		ctx := context.Background()
		pipe := common.RDB.Pipeline()
		cmds := make(map[int]*redisv8.FloatCmd, len(userIds))
		seen := make(map[int]struct{}, len(userIds))
		for _, userId := range userIds {
			if _, ok := seen[userId]; ok {
				continue
			}
			seen[userId] = struct{}{}
			cmds[userId] = pipe.ZScore(ctx, siteOnlineKey, strconv.Itoa(userId))
		}
		_, _ = pipe.Exec(ctx)
		for userId, cmd := range cmds {
			if score, err := cmd.Result(); err == nil && int64(score) >= cutoff {
				result[userId] = true
			}
		}
		return result
	}
	for _, userId := range userIds {
		if result[userId] {
			continue
		}
		val, ok := siteOnlineMemory.Load(userId)
		if !ok {
			continue
		}
		ts, ok := val.(int64)
		if ok && ts >= cutoff {
			result[userId] = true
		}
	}
	return result
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
	friendIds, err := model.GetFriendList(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	onlineMap := getSiteOnlineStatusMap(friendIds)
	list, err := model.GetFriendInfoListByIds(userId, friendIds, func(friendId int) bool {
		return onlineMap[friendId]
	})
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
	targetIds := make([]int, 0, len(users))
	for _, u := range users {
		targetIds = append(targetIds, u.Id)
	}
	onlineMap := getSiteOnlineStatusMap(targetIds)
	relationMap, err := model.GetOutgoingFriendStatusMap(userId, targetIds)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "搜索失败"})
		return
	}
	result := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		relation := relationMap[u.Id]
		result = append(result, map[string]interface{}{
			"user_id":      u.Id,
			"username":     u.Username,
			"display_name": u.DisplayName,
			"is_friend":    relation.IsFriend,
			"req_status":   relation.ReqStatus,
			"online":       onlineMap[u.Id],
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// GET /api/social/online-users — 返回当前所有在线用户（含好友关系状态）
func WebSocialOnlineUsers(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先登录"})
		return
	}
	onlineIds := getOnlineUserIdsLimited(userId, 100)
	if len(onlineIds) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
		return
	}
	// 批量查用户信息
	var users []model.User
	if err := model.DB.Select("id, username, display_name").
		Where("id IN ? AND deleted_at IS NULL", onlineIds).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	targetIds := make([]int, 0, len(users))
	for _, u := range users {
		targetIds = append(targetIds, u.Id)
	}
	relationMap, err := model.GetOutgoingFriendStatusMap(userId, targetIds)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "系统错误"})
		return
	}
	result := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		relation := relationMap[u.Id]
		result = append(result, map[string]interface{}{
			"user_id":      u.Id,
			"username":     u.Username,
			"display_name": u.DisplayName,
			"is_friend":    relation.IsFriend,
			"req_status":   relation.ReqStatus,
			"online":       true,
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

// GET /api/farm/chat/:friend_id  (also used by /api/social/chat/:friend_id)
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
	// 通知好友：我已读了他发的消息
	me, _ := model.GetUserById(userId, false)
	pushEvent(friendId, FarmEvent{
		Type:     "messages_read",
		FromId:   userId,
		FromName: nameOf(me),
		Payload:  map[string]interface{}{"reader_id": userId},
	})
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
	if !isSiteOnline(friendId) {
		friendUser, friendErr := model.GetUserById(friendId, false)
		if friendErr == nil && friendUser != nil {
			preview := content
			if utf8.RuneCountInString(preview) > 60 {
				runes := []rune(preview)
				preview = string(runes[:60]) + "..."
			}
			_ = service.TryNotifyUserBoundEmailWithWindow(friendUser, dto.NewNotify(
				dto.NotifyTypeSocialOfflineMessage,
				"好友消息提醒",
				"你的好友 {{value}} 给你发来一条新消息：\n{{value}}",
				[]interface{}{nameOf(me), preview},
			), fmt.Sprintf("offline_message_%d", userId), 90*time.Minute)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": msg})
}

/* ═══════════════════════════════════════════════════════════════
   Typing 指示器
   ═══════════════════════════════════════════════════════════════ */

// POST /api/social/chat/typing  body: {friend_id: int}
func WebFarmChatTyping(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false})
		return
	}
	var req struct {
		FriendId int `json:"friend_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FriendId == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false})
		return
	}
	if !model.IsFriend(userId, req.FriendId) {
		c.JSON(http.StatusOK, gin.H{"success": false})
		return
	}
	me, _ := model.GetUserById(userId, false)
	pushEvent(req.FriendId, FarmEvent{
		Type:     "typing",
		FromId:   userId,
		FromName: nameOf(me),
	})
	c.JSON(http.StatusOK, gin.H{"success": true})
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
