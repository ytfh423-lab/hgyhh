package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ========== 委托系统常量 ==========

const (
	entrustMinPublishLevel  = 5
	entrustMinAcceptLevel   = 3
	entrustMaxDailyPublish  = 5
	entrustMaxDailyAccept   = 10
	entrustMaxRewardFloat   = 500.0 // 最大报酬 $500
	entrustMinRewardFloat   = 0.01  // 最小报酬 $0.01
	entrustMaxDeadlineHours = 72
	entrustMaxTargetCount   = 50
	entrustMaxSettlePerPair = 5 // 同一对玩家每日结算上限
	entrustQuotaPerUnit     = 500000
)

// 委托任务类型定义
type entrustActionDef struct {
	Action string
	Module string
	Label  string
	Emoji  string
	Unit   string
}

var entrustActions = []entrustActionDef{
	{"water", "farm", "浇水", "💧", "块地"},
	{"fertilize", "farm", "施肥", "🧴", "块地"},
	{"harvest", "farm", "收获", "🌾", "块地"},
	{"treat", "farm", "治疗", "💊", "块地"},
	{"ranch_feed", "ranch", "牧场喂食", "🌾", "只"},
	{"ranch_water", "ranch", "牧场喂水", "💧", "只"},
	{"ranch_clean", "ranch", "牧场清理", "🧹", "次"},
	{"tree_water", "tree", "树场浇水", "💧", "棵"},
	{"tree_harvest", "tree", "树场采收", "🍎", "棵"},
	{"tree_chop", "tree", "树场伐木", "🪓", "棵"},
}

var entrustActionMap map[string]*entrustActionDef

func init() {
	entrustActionMap = make(map[string]*entrustActionDef)
	for i := range entrustActions {
		entrustActionMap[entrustActions[i].Action] = &entrustActions[i]
	}
}

func entrustActionLabel(action string) string {
	if d := entrustActionMap[action]; d != nil {
		return d.Emoji + d.Label
	}
	return action
}

// ========== API: 任务大厅 ==========

func WebEntrustHall(c *gin.Context) {
	_, _, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	module := c.Query("module")
	action := c.Query("action")

	tasks, total, err := model.GetPublishedEntrusts(page, 20, module, action)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "加载失败"})
		return
	}

	var list []gin.H
	for _, t := range tasks {
		list = append(list, entrustTaskToMap(t))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tasks":   list,
			"total":   total,
			"page":    page,
			"actions": entrustActions,
		},
	})
}

// ========== API: 任务详情 ==========

func WebEntrustDetail(c *gin.Context) {
	_, _, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	taskId, _ := strconv.Atoi(c.Query("id"))
	if taskId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	task, err := model.GetEntrustById(taskId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "任务不存在"})
		return
	}
	workers, _ := model.GetEntrustWorkers(taskId)
	var workerList []gin.H
	for _, w := range workers {
		workerList = append(workerList, gin.H{
			"id": w.Id, "worker_id": w.WorkerTelegramId, "status": w.Status,
			"progress": w.ProgressCount, "accepted_at": w.AcceptedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task":    entrustTaskToMap(task),
			"workers": workerList,
		},
	})
}

// ========== API: 创建委托（含托管） ==========

func WebEntrustCreate(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	// 等级检查
	level := model.GetFarmLevel(tgId)
	if level < entrustMinPublishLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("需要农场等级 Lv.%d 才能发布委托", entrustMinPublishLevel)})
		return
	}

	// 每日发布上限
	if model.CountTodayEntrustPublished(tgId) >= int64(entrustMaxDailyPublish) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("每日最多发布 %d 个委托", entrustMaxDailyPublish)})
		return
	}

	var req struct {
		Title          string  `json:"title"`
		TargetAction   string  `json:"target_action"`
		TargetItemKey  string  `json:"target_item_key"`
		TargetCount    int     `json:"target_count"`
		RewardAmount   float64 `json:"reward_amount"` // 浮点美元值, 如 1.5 = $1.50
		DeadlineHours  int     `json:"deadline_hours"`
		SettlementMode string  `json:"settlement_mode"`
		MaxWorkers     int     `json:"max_workers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	// 验证任务类型
	actionDef := entrustActionMap[req.TargetAction]
	if actionDef == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "不支持的任务类型"})
		return
	}

	if req.TargetCount < 1 || req.TargetCount > entrustMaxTargetCount {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("目标数量 1~%d", entrustMaxTargetCount)})
		return
	}
	if req.RewardAmount < entrustMinRewardFloat || req.RewardAmount > entrustMaxRewardFloat {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("报酬范围 $%.2f~$%.2f", entrustMinRewardFloat, entrustMaxRewardFloat)})
		return
	}
	rewardQuota := int(req.RewardAmount * entrustQuotaPerUnit)
	if req.DeadlineHours < 1 || req.DeadlineHours > entrustMaxDeadlineHours {
		req.DeadlineHours = 24
	}
	if req.Title == "" {
		req.Title = fmt.Sprintf("%s %d%s", actionDef.Label, req.TargetCount, actionDef.Unit)
	}
	if len(req.Title) > 60 {
		req.Title = req.Title[:60]
	}
	if req.SettlementMode != "full" {
		req.SettlementMode = "partial"
	}
	if req.MaxWorkers < 1 {
		req.MaxWorkers = 1
	}
	if req.MaxWorkers > 5 {
		req.MaxWorkers = 5
	}

	// ── 托管扣款（原子操作）──
	if user.Quota < rewardQuota {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("余额不足，需要 $%.2f，当前 $%.2f", req.RewardAmount, float64(user.Quota)/entrustQuotaPerUnit)})
		return
	}
	if err := model.DecreaseUserQuota(user.Id, rewardQuota); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "托管扣款失败"})
		return
	}

	// 创建任务
	now := time.Now().Unix()
	task := &model.TgFarmEntrust{
		OwnerTelegramId: tgId,
		Title:           req.Title,
		TargetAction:    req.TargetAction,
		TargetModule:    actionDef.Module,
		TargetItemKey:   req.TargetItemKey,
		TargetCount:     req.TargetCount,
		RewardAmount:    rewardQuota,
		EscrowStatus:    "escrow_success",
		Status:          "published",
		IsPublic:        1,
		MaxWorkerCount:  req.MaxWorkers,
		SettlementMode:  req.SettlementMode,
		DeadlineAt:      now + int64(req.DeadlineHours)*3600,
	}
	if err := model.CreateEntrust(task); err != nil {
		// 回滚：退回托管款
		_ = model.IncreaseUserQuota(user.Id, rewardQuota, true)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建任务失败，已退款"})
		return
	}

	// 记录托管流水
	_ = model.CreateEntrustEscrow(&model.TgFarmEntrustEscrow{
		TaskId:          task.Id,
		OwnerTelegramId: tgId,
		Amount:          rewardQuota,
		Action:          "escrow",
	})

	model.AddFarmLog(tgId, "entrust_publish", -rewardQuota, fmt.Sprintf("发布委托:%s", req.Title))
	common.SysLog(fmt.Sprintf("Entrust: user %s published task #%d, escrow %d", tgId, task.Id, rewardQuota))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("委托已发布！报酬 $%.2f 已托管到平台", req.RewardAmount),
		"data":    entrustTaskToMap(task),
	})
}

// ========== API: 取消委托 ==========

func WebEntrustCancel(c *gin.Context) {
	user, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		TaskId int `json:"task_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TaskId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	task, err := model.GetEntrustById(req.TaskId)
	if err != nil || task.OwnerTelegramId != tgId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "任务不存在"})
		return
	}
	if task.Status == "completed" || task.Status == "cancelled" || task.Status == "expired" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "任务已结束，无法取消"})
		return
	}

	// 计算已结算金额
	settled := model.GetEntrustSettledAmount(req.TaskId)
	refund := task.RewardAmount - settled
	if refund < 0 {
		refund = 0
	}

	// 已有进度按比例结算给工人
	if task.ProgressCount > 0 && task.Status == "in_progress" {
		workers, _ := model.GetEntrustWorkers(req.TaskId)
		for _, w := range workers {
			if w.Status == "accepted" || w.Status == "working" {
				if w.ProgressCount > 0 {
					workerReward := entrustCalcWorkerReward(task, w.ProgressCount)
					if workerReward > 0 && workerReward <= refund {
						entrustSettleWorker(task, w, workerReward)
						refund -= workerReward
					}
				}
				_ = model.UpdateEntrustWorkerFields(w.Id, map[string]interface{}{"status": "completed", "completed_at": time.Now().Unix()})
			}
		}
	}

	// 退回剩余托管
	if refund > 0 {
		_ = model.IncreaseUserQuota(user.Id, refund, true)
		_ = model.CreateEntrustEscrow(&model.TgFarmEntrustEscrow{
			TaskId:          req.TaskId,
			OwnerTelegramId: tgId,
			Amount:          refund,
			Action:          "refund",
		})
	}

	_ = model.UpdateEntrustFields(req.TaskId, map[string]interface{}{
		"status":        "cancelled",
		"escrow_status": "refunded",
	})

	model.AddFarmLog(tgId, "entrust_cancel", refund, fmt.Sprintf("取消委托#%d，退回$%.2f", req.TaskId, float64(refund)/entrustQuotaPerUnit))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("委托已取消，退回 $%.2f", float64(refund)/entrustQuotaPerUnit),
	})
}

// ========== API: 接单 ==========

func WebEntrustAccept(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	level := model.GetFarmLevel(tgId)
	if level < entrustMinAcceptLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("需要农场等级 Lv.%d 才能接单", entrustMinAcceptLevel)})
		return
	}
	if model.CountTodayEntrustAccepted(tgId) >= int64(entrustMaxDailyAccept) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("每日最多接 %d 单", entrustMaxDailyAccept)})
		return
	}

	var req struct {
		TaskId int `json:"task_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TaskId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	task, err := model.GetEntrustById(req.TaskId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "任务不存在"})
		return
	}

	// 禁止自己接自己的任务
	if task.OwnerTelegramId == tgId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "不能接自己发布的委托"})
		return
	}

	if task.Status != "published" && task.Status != "in_progress" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该任务当前无法接单"})
		return
	}
	if task.DeadlineAt <= time.Now().Unix() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该任务已过期"})
		return
	}

	// 检查是否已接过此任务
	if _, err := model.GetEntrustWorker(req.TaskId, tgId); err == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你已经接过这个任务了"})
		return
	}

	// 检查接单人数上限
	activeCount := model.CountActiveEntrustWorkers(req.TaskId)
	if activeCount >= int64(task.MaxWorkerCount) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该任务已满员"})
		return
	}

	// 同一对玩家每日结算上限
	if model.CountTodaySettleBetween(tgId, task.OwnerTelegramId) >= int64(entrustMaxSettlePerPair) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "与该玩家今日协作已达上限"})
		return
	}

	worker := &model.TgFarmEntrustWorker{
		TaskId:           req.TaskId,
		WorkerTelegramId: tgId,
		Status:           "accepted",
	}
	if err := model.CreateEntrustWorker(worker); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "接单失败"})
		return
	}

	// 更新任务状态
	if task.Status == "published" {
		_ = model.UpdateEntrustFields(req.TaskId, map[string]interface{}{"status": "in_progress"})
	}

	model.AddFarmLog(tgId, "entrust_accept", 0, fmt.Sprintf("接受委托#%d:%s", req.TaskId, task.Title))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("接单成功！报酬 $%.2f", float64(task.RewardAmount)/entrustQuotaPerUnit),
	})
}

// ========== API: 放弃接单 ==========

func WebEntrustAbandon(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	var req struct {
		TaskId int `json:"task_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TaskId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	worker, err := model.GetEntrustWorker(req.TaskId, tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你没有接这个任务"})
		return
	}

	_ = model.UpdateEntrustWorkerFields(worker.Id, map[string]interface{}{"status": "abandoned"})

	// 如果没有其他活跃工人，任务回到 published
	if model.CountActiveEntrustWorkers(req.TaskId) == 0 {
		_ = model.UpdateEntrustFields(req.TaskId, map[string]interface{}{"status": "published"})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已放弃该委托"})
}

// ========== API: 我发布的委托 ==========

func WebEntrustMyPublished(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	tasks, _ := model.GetEntrustsByOwner(tgId)
	var list []gin.H
	for _, t := range tasks {
		m := entrustTaskToMap(t)
		settled := model.GetEntrustSettledAmount(t.Id)
		m["settled_amount"] = float64(settled) / entrustQuotaPerUnit
		m["refundable"] = float64(t.RewardAmount-settled) / entrustQuotaPerUnit
		list = append(list, m)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list})
}

// ========== API: 我接的委托 ==========

func WebEntrustMyAccepted(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	workers, _ := model.GetAllMyEntrustWorkers(tgId)
	var list []gin.H
	for _, w := range workers {
		task, err := model.GetEntrustById(w.TaskId)
		if err != nil {
			continue
		}
		list = append(list, gin.H{
			"worker_id":     w.Id,
			"task":          entrustTaskToMap(task),
			"status":        w.Status,
			"my_progress":   w.ProgressCount,
			"reward_earned": float64(w.RewardAmount) / entrustQuotaPerUnit,
			"accepted_at":   w.AcceptedAt,
			"completed_at":  w.CompletedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list})
}

// ========== API: 工作模式视图 ==========

func WebEntrustWorkView(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}
	taskId, _ := strconv.Atoi(c.Query("id"))
	if taskId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	task, err := model.GetEntrustById(taskId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "任务不存在"})
		return
	}

	worker, err := model.GetEntrustWorker(taskId, tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你没有接这个任务"})
		return
	}

	if task.Status != "in_progress" && task.Status != "published" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "任务已结束"})
		return
	}

	// 获取雇主农场相关数据
	ownerTgId := task.OwnerTelegramId
	entities := entrustGetWorkEntities(task, ownerTgId, tgId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task":        entrustTaskToMap(task),
			"my_progress": worker.ProgressCount,
			"entities":    entities,
		},
	})
}

// ========== API: 执行委托操作 ==========

func WebEntrustWorkExecute(c *gin.Context) {
	_, tgId, ok := getWebFarmUser(c)
	if !ok {
		return
	}

	var req struct {
		TaskId   int `json:"task_id"`
		EntityId int `json:"entity_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TaskId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	task, err := model.GetEntrustById(req.TaskId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "任务不存在"})
		return
	}
	if task.Status != "in_progress" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "任务不在进行中"})
		return
	}
	if task.DeadlineAt <= time.Now().Unix() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "任务已过期"})
		return
	}
	if task.ProgressCount >= task.TargetCount {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "任务目标已完成"})
		return
	}

	worker, err := model.GetEntrustWorker(req.TaskId, tgId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "你没有接这个任务"})
		return
	}

	// 防重复：检查该实体是否已被此worker操作过
	if model.HasEntrustLogForEntity(req.TaskId, tgId, req.EntityId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该目标已处理过"})
		return
	}

	// 执行实际操作
	ownerTgId := task.OwnerTelegramId
	msg, execErr := entrustExecuteAction(task, ownerTgId, req.EntityId)
	if execErr != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": execErr.Error()})
		return
	}

	// 记录操作日志（防重复）
	_ = model.CreateEntrustLog(&model.TgFarmEntrustLog{
		TaskId:           req.TaskId,
		WorkerTelegramId: tgId,
		ActionType:       task.TargetAction,
		TargetEntityId:   req.EntityId,
		ProgressDelta:    1,
	})

	// 递增进度
	_ = model.IncrementWorkerProgress(worker.Id, 1)
	newProgress, targetCount, _ := model.IncrementEntrustProgress(req.TaskId, 1)

	// 更新 worker 状态
	if worker.Status == "accepted" {
		_ = model.UpdateEntrustWorkerFields(worker.Id, map[string]interface{}{"status": "working"})
	}

	// 检查是否完成
	completed := newProgress >= targetCount
	if completed {
		entrustCompleteTask(task)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   msg,
		"completed": completed,
		"progress":  newProgress,
		"target":    targetCount,
	})
}

// ========== 内部：执行具体操作 ==========

func entrustExecuteAction(task *model.TgFarmEntrust, ownerTgId string, entityId int) (string, error) {
	switch task.TargetAction {
	case "water":
		return entrustDoWater(ownerTgId, entityId)
	case "fertilize":
		return entrustDoFertilize(ownerTgId, entityId)
	case "harvest":
		return entrustDoHarvest(ownerTgId, entityId)
	case "treat":
		return entrustDoTreat(ownerTgId, entityId)
	case "ranch_feed":
		return entrustDoRanchFeed(ownerTgId, entityId)
	case "ranch_water":
		return entrustDoRanchWater(ownerTgId, entityId)
	case "ranch_clean":
		return entrustDoRanchClean(ownerTgId)
	case "tree_water":
		return entrustDoTreeWater(ownerTgId, entityId)
	case "tree_harvest":
		return entrustDoTreeHarvest(ownerTgId, entityId)
	case "tree_chop":
		return entrustDoTreeChop(ownerTgId, entityId)
	default:
		return "", fmt.Errorf("不支持的操作类型")
	}
}

func entrustDoWater(ownerTgId string, plotId int) (string, error) {
	plots, err := model.GetOrCreateFarmPlots(ownerTgId)
	if err != nil {
		return "", fmt.Errorf("系统错误")
	}
	var target *model.TgFarmPlot
	for _, p := range plots {
		if p.Id == plotId {
			target = p
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("地块不存在")
	}
	updateFarmPlotStatus(target)
	canWater := target.Status == 1 || target.Status == 4 || (target.Status == 3 && target.EventType == "drought")
	if !canWater {
		return "", fmt.Errorf("该地块不需要浇水")
	}
	if target.Status == 4 {
		now := time.Now().Unix()
		target.Status = 1
		target.LastWateredAt = now
		_ = model.UpdateFarmPlot(target)
	} else if target.Status == 3 && target.EventType == "drought" {
		now := time.Now().Unix()
		downtime := now - target.EventAt
		target.PlantedAt += downtime
		target.Status = 1
		target.EventType = ""
		target.EventAt = 0
		_ = model.UpdateFarmPlot(target)
	}
	_ = model.WaterFarmPlot(target.Id)
	return "💧 浇水成功", nil
}

func entrustDoFertilize(ownerTgId string, plotId int) (string, error) {
	plots, _ := model.GetOrCreateFarmPlots(ownerTgId)
	var target *model.TgFarmPlot
	for _, p := range plots {
		if p.Id == plotId {
			target = p
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("地块不存在")
	}
	updateFarmPlotStatus(target)
	if target.Status != 1 {
		return "", fmt.Errorf("该地块不在生长中")
	}
	if target.Fertilized == 1 {
		return "", fmt.Errorf("已经施过肥了")
	}
	// 检查雇主是否有肥料
	fertCount, _ := model.GetFarmItemQuantity(ownerTgId, "fertilizer")
	if fertCount <= 0 {
		return "", fmt.Errorf("雇主没有肥料")
	}
	_ = model.DecrementFarmItem(ownerTgId, "fertilizer")
	target.Fertilized = 1
	_ = model.UpdateFarmPlot(target)
	return "🧴 施肥成功", nil
}

func entrustDoHarvest(ownerTgId string, plotId int) (string, error) {
	plots, _ := model.GetOrCreateFarmPlots(ownerTgId)
	var target *model.TgFarmPlot
	for _, p := range plots {
		if p.Id == plotId {
			target = p
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("地块不存在")
	}
	updateFarmPlotStatus(target)
	if target.Status != 2 {
		return "", fmt.Errorf("该地块尚未成熟")
	}
	crop := farmCropMap[target.CropType]
	if crop == nil {
		return "", fmt.Errorf("未知作物")
	}
	// 收获产物归雇主
	_ = model.AddToWarehouseWithCategory(ownerTgId, target.CropType, 1, "crop")
	model.RecordCollection(ownerTgId, "crop", target.CropType, 1)
	_ = model.ClearFarmPlot(target.Id)
	return fmt.Sprintf("🌾 收获 %s%s 入雇主仓库", crop.Emoji, crop.Name), nil
}

func entrustDoTreat(ownerTgId string, plotId int) (string, error) {
	plots, _ := model.GetOrCreateFarmPlots(ownerTgId)
	var target *model.TgFarmPlot
	for _, p := range plots {
		if p.Id == plotId {
			target = p
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("地块不存在")
	}
	updateFarmPlotStatus(target)
	if target.Status != 3 || target.EventType == "drought" {
		return "", fmt.Errorf("该地块没有需要治疗的事件")
	}
	// 检查雇主是否有对应药品
	var cureItemKey, cureItemEmoji, cureItemName string
	for _, fi := range farmItems {
		if fi.Cures == target.EventType {
			cureItemKey = fi.Key
			cureItemEmoji = fi.Emoji
			cureItemName = fi.Name
			break
		}
	}
	if cureItemKey == "" {
		return "", fmt.Errorf("无法治疗该事件")
	}
	count, _ := model.GetFarmItemQuantity(ownerTgId, cureItemKey)
	if count <= 0 {
		return "", fmt.Errorf("雇主缺少 %s%s", cureItemEmoji, cureItemName)
	}
	_ = model.DecrementFarmItem(ownerTgId, cureItemKey)
	now := time.Now().Unix()
	downtime := now - target.EventAt
	target.PlantedAt += downtime
	target.Status = 1
	target.EventType = ""
	target.EventAt = 0
	_ = model.UpdateFarmPlot(target)
	return "💊 治疗成功", nil
}

func entrustDoRanchFeed(ownerTgId string, animalId int) (string, error) {
	animals, _ := model.GetRanchAnimals(ownerTgId)
	var target *model.TgRanchAnimal
	for _, a := range animals {
		if a.Id == animalId {
			target = a
			break
		}
	}
	if target == nil || target.Status == 5 {
		return "", fmt.Errorf("动物不存在或已死亡")
	}
	price := common.TgBotRanchFeedPrice
	ownerUser, err := model.GetUserByFarmId(ownerTgId)
	if err != nil {
		return "", fmt.Errorf("雇主账户异常")
	}
	if ownerUser.Quota < price {
		return "", fmt.Errorf("雇主余额不足支付饲料费")
	}
	_ = model.DecreaseUserQuota(ownerUser.Id, price)
	_ = model.FeedRanchAnimal(target.Id)
	now := time.Now().Unix()
	target.LastFedAt = now
	if target.Status == 3 {
		waterInterval := int64(common.TgBotRanchWaterInterval)
		if now > target.LastWateredAt+waterInterval {
			target.Status = 4
		} else {
			def := ranchAnimalMap[target.AnimalType]
			if def != nil && now-target.PurchasedAt >= *def.GrowSecs {
				target.Status = 2
			} else {
				target.Status = 1
			}
		}
		_ = model.UpdateRanchAnimal(target)
	}
	model.AddFarmLog(ownerTgId, "ranch_feed", -price, "委托喂食")
	return "🌾 喂食成功", nil
}

func entrustDoRanchWater(ownerTgId string, animalId int) (string, error) {
	animals, _ := model.GetRanchAnimals(ownerTgId)
	var target *model.TgRanchAnimal
	for _, a := range animals {
		if a.Id == animalId {
			target = a
			break
		}
	}
	if target == nil || target.Status == 5 {
		return "", fmt.Errorf("动物不存在或已死亡")
	}
	price := common.TgBotRanchWaterPrice
	ownerUser, err := model.GetUserByFarmId(ownerTgId)
	if err != nil {
		return "", fmt.Errorf("雇主账户异常")
	}
	if ownerUser.Quota < price {
		return "", fmt.Errorf("雇主余额不足支付喂水费")
	}
	_ = model.DecreaseUserQuota(ownerUser.Id, price)
	_ = model.WaterRanchAnimal(target.Id)
	now := time.Now().Unix()
	target.LastWateredAt = now
	if target.Status == 4 {
		feedInterval := int64(common.TgBotRanchFeedInterval)
		if now > target.LastFedAt+feedInterval {
			target.Status = 3
		} else {
			def := ranchAnimalMap[target.AnimalType]
			if def != nil && now-target.PurchasedAt >= *def.GrowSecs {
				target.Status = 2
			} else {
				target.Status = 1
			}
		}
		_ = model.UpdateRanchAnimal(target)
	}
	model.AddFarmLog(ownerTgId, "ranch_water", -price, "委托喂水")
	return "💧 喂水成功", nil
}

func entrustDoRanchClean(ownerTgId string) (string, error) {
	ownerUser, err := model.GetUserByFarmId(ownerTgId)
	if err != nil {
		return "", fmt.Errorf("雇主账户异常")
	}
	price := common.TgBotRanchManureCleanPrice
	if ownerUser.Quota < price {
		return "", fmt.Errorf("雇主余额不足支付清理费")
	}
	_ = model.DecreaseUserQuota(ownerUser.Id, price)
	_ = model.CleanRanchAnimals(ownerTgId)
	model.AddFarmLog(ownerTgId, "ranch_clean", -price, "委托清理粪便")
	return "🧹 清理成功", nil
}

func entrustDoTreeWater(ownerTgId string, slotId int) (string, error) {
	slot, err := model.GetTreeSlotById(slotId)
	if err != nil || slot.TelegramId != ownerTgId {
		return "", fmt.Errorf("树位不存在")
	}
	if slot.Status != 1 && slot.Status != 2 {
		return "", fmt.Errorf("该树位不需要浇水")
	}
	now := time.Now().Unix()
	_ = model.WaterTree(slot.Id, now)
	return "💧 树场浇水成功", nil
}

func entrustDoTreeHarvest(ownerTgId string, slotId int) (string, error) {
	slot, err := model.GetTreeSlotById(slotId)
	if err != nil || slot.TelegramId != ownerTgId {
		return "", fmt.Errorf("树位不存在")
	}
	if slot.Status != 2 {
		return "", fmt.Errorf("该树尚未成熟")
	}
	tree := treeFarmTreeMap[slot.TreeType]
	if tree == nil || !tree.Repeatable || len(tree.HarvestYield) == 0 {
		return "", fmt.Errorf("该树种不支持采收")
	}
	now := time.Now().Unix()
	_ = model.HarvestTree(slot.Id, now)
	// 按树木定义产物存入雇主仓库
	items := calcTreeYieldItems(tree.HarvestYield)
	msg := fmt.Sprintf("🍎 树场采收%s%s：", tree.Emoji, tree.Name)
	for _, item := range items {
		_ = model.AddToWarehouseWithCategory(ownerTgId, "wood_"+item.ItemKey, item.Amount, "wood")
		msg += fmt.Sprintf("%s%s×%d ", item.Emoji, item.Name, item.Amount)
	}
	return msg, nil
}

func entrustDoTreeChop(ownerTgId string, slotId int) (string, error) {
	slot, err := model.GetTreeSlotById(slotId)
	if err != nil || slot.TelegramId != ownerTgId {
		return "", fmt.Errorf("树位不存在")
	}
	if slot.Status != 2 {
		return "", fmt.Errorf("该树尚未成熟，无法伐木")
	}
	tree := treeFarmTreeMap[slot.TreeType]
	if tree == nil || !tree.CanChop || len(tree.ChopYield) == 0 {
		return "", fmt.Errorf("该树种不支持伐木")
	}
	now := time.Now().Unix()
	_ = model.ChopTree(slot.Id, now)
	// 按树木定义产物存入雇主仓库
	items := calcTreeYieldItems(tree.ChopYield)
	msg := fmt.Sprintf("🪓 伐木%s%s：", tree.Emoji, tree.Name)
	for _, item := range items {
		_ = model.AddToWarehouseWithCategory(ownerTgId, "wood_"+item.ItemKey, item.Amount, "wood")
		msg += fmt.Sprintf("%s%s×%d ", item.Emoji, item.Name, item.Amount)
	}
	return msg, nil
}

// ========== 内部：获取工作视图实体 ==========

func entrustGetWorkEntities(task *model.TgFarmEntrust, ownerTgId, workerTgId string) []gin.H {
	switch task.TargetAction {
	case "water", "fertilize", "harvest", "treat":
		return entrustGetFarmEntities(task, ownerTgId, workerTgId)
	case "ranch_feed", "ranch_water":
		return entrustGetRanchEntities(task, ownerTgId, workerTgId)
	case "ranch_clean":
		done := model.HasEntrustLogForEntity(task.Id, workerTgId, 0)
		return []gin.H{{"id": 0, "label": "🧹 清理牧场", "actionable": !done, "done": done}}
	case "tree_water", "tree_harvest", "tree_chop":
		return entrustGetTreeEntities(task, ownerTgId, workerTgId)
	}
	return nil
}

func entrustGetFarmEntities(task *model.TgFarmEntrust, ownerTgId, workerTgId string) []gin.H {
	plots, _ := model.GetOrCreateFarmPlots(ownerTgId)
	var entities []gin.H
	for _, p := range plots {
		updateFarmPlotStatus(p)
		actionable := false
		label := ""
		crop := farmCropMap[p.CropType]
		cropName := p.CropType
		cropEmoji := "🌱"
		if crop != nil {
			cropName = crop.Name
			cropEmoji = crop.Emoji
		}

		switch task.TargetAction {
		case "water":
			actionable = p.Status == 1 || p.Status == 4 || (p.Status == 3 && p.EventType == "drought")
			if p.Status == 0 {
				continue
			}
			label = fmt.Sprintf("%s %s #%d", cropEmoji, cropName, p.PlotIndex+1)
		case "fertilize":
			actionable = p.Status == 1 && p.Fertilized == 0
			if p.Status != 1 {
				continue
			}
			label = fmt.Sprintf("%s %s #%d", cropEmoji, cropName, p.PlotIndex+1)
		case "harvest":
			if p.Status != 2 {
				continue
			}
			actionable = true
			label = fmt.Sprintf("%s %s #%d ✅", cropEmoji, cropName, p.PlotIndex+1)
		case "treat":
			if p.Status != 3 || p.EventType == "drought" {
				continue
			}
			actionable = true
			label = fmt.Sprintf("%s %s #%d 🐛", cropEmoji, cropName, p.PlotIndex+1)
		}

		done := model.HasEntrustLogForEntity(task.Id, workerTgId, p.Id)
		if done {
			actionable = false
		}
		entities = append(entities, gin.H{
			"id": p.Id, "label": label, "actionable": actionable, "done": done,
		})
	}
	return entities
}

func entrustGetRanchEntities(task *model.TgFarmEntrust, ownerTgId, workerTgId string) []gin.H {
	animals, _ := model.GetRanchAnimals(ownerTgId)
	var entities []gin.H
	now := time.Now().Unix()
	for _, a := range animals {
		if a.Status == 5 {
			continue
		}
		def := ranchAnimalMap[a.AnimalType]
		name := a.AnimalType
		emoji := "🐾"
		if def != nil {
			name = def.Name
			emoji = def.Emoji
		}
		actionable := false
		feedInterval := int64(common.TgBotRanchFeedInterval)
		waterInterval := int64(common.TgBotRanchWaterInterval)
		switch task.TargetAction {
		case "ranch_feed":
			actionable = a.LastFedAt == 0 || now >= a.LastFedAt+feedInterval
		case "ranch_water":
			actionable = a.LastWateredAt == 0 || now >= a.LastWateredAt+waterInterval
		}
		done := model.HasEntrustLogForEntity(task.Id, workerTgId, a.Id)
		if done {
			actionable = false
		}
		entities = append(entities, gin.H{
			"id": a.Id, "label": fmt.Sprintf("%s %s", emoji, name), "actionable": actionable, "done": done,
		})
	}
	return entities
}

func entrustGetTreeEntities(task *model.TgFarmEntrust, ownerTgId, workerTgId string) []gin.H {
	slots, _ := model.GetOrCreateTreeSlots(ownerTgId)
	var entities []gin.H
	for _, s := range slots {
		if s.Status == 0 {
			continue
		}
		actionable := false
		label := fmt.Sprintf("🌲 %s #%d", s.TreeType, s.SlotIndex+1)
		switch task.TargetAction {
		case "tree_water":
			actionable = s.Status == 1 || s.Status == 2
		case "tree_harvest":
			actionable = s.Status == 2
		case "tree_chop":
			actionable = s.Status == 2
		}
		done := model.HasEntrustLogForEntity(task.Id, workerTgId, s.Id)
		if done {
			actionable = false
		}
		entities = append(entities, gin.H{
			"id": s.Id, "label": label, "actionable": actionable, "done": done,
		})
	}
	return entities
}

// ========== 内部：任务完成与结算 ==========

func entrustCompleteTask(task *model.TgFarmEntrust) {
	workers, _ := model.GetEntrustWorkers(task.Id)
	settled := model.GetEntrustSettledAmount(task.Id)
	remaining := task.RewardAmount - settled

	totalWorkerProgress := 0
	for _, w := range workers {
		if w.Status == "working" || w.Status == "accepted" {
			totalWorkerProgress += w.ProgressCount
		}
	}

	for _, w := range workers {
		if (w.Status == "working" || w.Status == "accepted") && w.ProgressCount > 0 {
			var reward int
			if totalWorkerProgress > 0 {
				reward = remaining * w.ProgressCount / totalWorkerProgress
			}
			if reward > remaining {
				reward = remaining
			}
			if reward > 0 {
				entrustSettleWorker(task, w, reward)
				remaining -= reward
			}
			_ = model.UpdateEntrustWorkerFields(w.Id, map[string]interface{}{
				"status":       "completed",
				"completed_at": time.Now().Unix(),
			})
		}
	}

	// 如果还有剩余，退回雇主
	if remaining > 0 {
		ownerUser, err := model.GetUserByFarmId(task.OwnerTelegramId)
		if err == nil {
			_ = model.IncreaseUserQuota(ownerUser.Id, remaining, true)
			_ = model.CreateEntrustEscrow(&model.TgFarmEntrustEscrow{
				TaskId:          task.Id,
				OwnerTelegramId: task.OwnerTelegramId,
				Amount:          remaining,
				Action:          "refund",
			})
		}
	}

	_ = model.UpdateEntrustFields(task.Id, map[string]interface{}{
		"status":        "completed",
		"escrow_status": "settled",
	})
}

func entrustSettleWorker(task *model.TgFarmEntrust, worker *model.TgFarmEntrustWorker, reward int) {
	workerUser, err := model.GetUserByFarmId(worker.WorkerTelegramId)
	if err != nil {
		return
	}
	_ = model.IncreaseUserQuota(workerUser.Id, reward, true)
	_ = model.UpdateEntrustWorkerFields(worker.Id, map[string]interface{}{"reward_amount": reward})
	_ = model.CreateEntrustEscrow(&model.TgFarmEntrustEscrow{
		TaskId:           task.Id,
		OwnerTelegramId:  task.OwnerTelegramId,
		WorkerTelegramId: worker.WorkerTelegramId,
		Amount:           reward,
		Action:           "settle",
	})
	model.AddFarmLog(worker.WorkerTelegramId, "entrust_reward", reward,
		fmt.Sprintf("委托报酬#%d: $%.2f", task.Id, float64(reward)/entrustQuotaPerUnit))
}

func entrustCalcWorkerReward(task *model.TgFarmEntrust, workerProgress int) int {
	if task.TargetCount <= 0 {
		return 0
	}
	if task.SettlementMode == "full" {
		return 0
	}
	return task.RewardAmount * workerProgress / task.TargetCount
}

// ========== 辅助函数 ==========

func entrustTaskToMap(t *model.TgFarmEntrust) gin.H {
	actionLabel := t.TargetAction
	if d := entrustActionMap[t.TargetAction]; d != nil {
		actionLabel = d.Emoji + d.Label
	}
	return gin.H{
		"id":              t.Id,
		"owner_id":        t.OwnerTelegramId,
		"title":           t.Title,
		"target_action":   t.TargetAction,
		"target_module":   t.TargetModule,
		"target_item_key": t.TargetItemKey,
		"target_count":    t.TargetCount,
		"progress_count":  t.ProgressCount,
		"reward_amount":   t.RewardAmount,
		"reward_display":  fmt.Sprintf("$%.2f", float64(t.RewardAmount)/entrustQuotaPerUnit),
		"escrow_status":   t.EscrowStatus,
		"status":          t.Status,
		"action_label":    actionLabel,
		"max_workers":     t.MaxWorkerCount,
		"settlement_mode": t.SettlementMode,
		"deadline_at":     t.DeadlineAt,
		"created_at":      t.CreatedAt,
		"remaining_secs":  t.DeadlineAt - time.Now().Unix(),
	}
}
