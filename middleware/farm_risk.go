package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// ═══════════════════════════════════════════════════════════════
//  FarmRiskGuard — 极简硬阈值人机验证中间件
//
//  核心原则：
//    1. 仅在 TurnstileCheckEnabled=true 时启用
//    2. 敏感动作（偷菜/交易/银行/转生/批量）→ 每次验证
//    3. 非敏感写操作 → 突发(45s≥6次)时验证
//    4. 验证通过后 10 分钟内非敏感动作免验
//    5. 连续 5 次验证失败 → 锁定 30 分钟
//    6. GET/HEAD/OPTIONS 永远放行
// ═══════════════════════════════════════════════════════════════

const (
	farmRiskStepUpCode       = "FARM_STEP_UP_REQUIRED"
	farmRiskVerifyFailCode   = "FARM_VERIFICATION_FAILED"
	farmRiskLockedCode       = "FARM_LOCKED"
	farmRiskBurstWindow      = 45 * time.Second
	farmRiskBurstThreshold   = 6
	farmRiskPassTTL          = 10 * time.Minute
	farmRiskLockTTL          = 30 * time.Minute
	farmRiskMaxFail          = 5
	farmRiskFailWindow       = 30 * time.Minute
	farmRiskDefaultMinScore  = 0.50
	farmRiskHighMinScore     = 0.60
)

// ── 动作敏感度表 ──
// sensitive=true 的动作每次都要求验证，不受 pass 豁免。
// 只列入：与他人交互 / 经济决策 / 不可逆操作。
// 一键类（种植/浇水/施肥/治疗/收获）属于正常玩法不列入，由 burst 阈值检测脚本滥用。
var farmRiskSensitiveActions = map[string]bool{
	"farm_steal":             true, // 偷菜：影响他人
	"farm_trade_create":      true, // 交易：经济决策
	"farm_trade_buy":         true,
	"farm_trade_cancel":      true,
	"farm_bank_loan":         true, // 银行：经济决策
	"farm_bank_mortgage":     true,
	"farm_bank_repay":        true,
	"farm_warehouse_sellall": true, // 一键卖仓：大额经济操作
	"farm_prestige":          true, // 转生：不可逆
	"ranch_slaughter":        true, // 屠宰：不可逆
	"ranch_slaughter_store":  true,
	"tree_chop":              true, // 砍树：不可逆
}

// ── 内存兜底（Redis 不可用时） ──
type farmRiskMemEntry struct {
	Timestamps []time.Time
	PassUntil  time.Time
	LockUntil  time.Time
	FailCount  int
	FailFirst  time.Time
}

var (
	farmRiskMem   = map[int]*farmRiskMemEntry{}
	farmRiskMemMu sync.Mutex
)

func farmRiskRedisKey(prefix string, userId int) string {
	return fmt.Sprintf("farm:risk:%s:%d", prefix, userId)
}

// ═══════════════════════════════════════════════════════════════
//  主中间件
// ═══════════════════════════════════════════════════════════════

func FarmRiskGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.TurnstileCheckEnabled {
			c.Next()
			return
		}
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		userId := c.GetInt("id")
		if userId == 0 {
			c.Next()
			return
		}

		// 检查锁定
		if farmRiskIsLocked(userId) {
			respondFarmRiskLocked(c)
			return
		}

		action := normalizeFarmRiskAction(c.FullPath())
		sensitive := farmRiskSensitiveActions[action]

		// 非敏感动作 + 持有 pass → 直接放行
		if !sensitive && farmRiskHasPass(userId) {
			farmRiskRecordBurst(userId)
			c.Next()
			return
		}

		// 判断是否需要验证
		needVerify := sensitive
		if !needVerify {
			burstCount := farmRiskRecordBurst(userId)
			if burstCount >= farmRiskBurstThreshold {
				needVerify = true
			}
		} else {
			farmRiskRecordBurst(userId)
		}

		provider := common.HumanVerificationProvider
		if provider == "" {
			provider = "turnstile"
		}

		// 读取请求中携带的 token，优先 Header，再 Query，最后 PostForm
		token := strings.TrimSpace(c.GetHeader("X-Farm-Captcha-Token"))
		if token == "" {
			token = strings.TrimSpace(c.Query("human_verification_token"))
		}
		if token == "" {
			token = strings.TrimSpace(c.PostForm("human_verification_token"))
		}
		version := strings.TrimSpace(c.GetHeader("X-Farm-Captcha-Version"))
		if version == "" {
			version = strings.TrimSpace(c.Query("human_verification_version"))
		}
		if version == "" {
			version = strings.TrimSpace(c.PostForm("human_verification_version"))
		}

		// 读取 action（v3 校验需要）
		requestAction := strings.TrimSpace(c.GetHeader("X-Farm-Captcha-Action"))
		if requestAction == "" {
			requestAction = strings.TrimSpace(c.Query("human_verification_action"))
		}
		if requestAction == "" {
			requestAction = strings.TrimSpace(c.PostForm("human_verification_action"))
		}

		// ═══════════════════════════════════════════════════════════
		// 新设计：v3 只做评分风控，所有验证都用 v2 复选框
		//   1) 带 v2 token  → v2 校验 → 通过则放行
		//   2) 带 v3 token  → v3 评分 → 分数够放行、分数不够弹 v2
		//   3) 没带 token   → 走 burst / sensitive 检测 → 触发则弹 v2
		// ═══════════════════════════════════════════════════════════

		// 1) v2 token：用户刚勾完 v2 复选框的重试请求
		if version == "v2" && provider == "recaptcha" && token != "" {
			result, err := VerifyHumanVerification(c.ClientIP(), token, HumanVerificationOptions{
				Version: "v2",
			})
			if err != nil {
				farmRiskRecordFail(userId)
				common.SysLog(fmt.Sprintf("[FarmRisk] v2 verify failed: user=%d action=%s err=%s",
					userId, action, err.Error()))
				respondFarmRiskVerifyFail(c, action, provider, err.Error())
				return
			}
			farmRiskGrantPass(userId)
			farmRiskClearFail(userId)
			common.SysLog(fmt.Sprintf("[FarmRisk] v2 verify passed: user=%d action=%s score=%.2f",
				userId, action, farmRiskResultScore(result)))
			c.Next()
			return
		}

		// 2) v3 token：前端请求拦截器自动带上，后端评分
		if provider == "recaptcha" && token != "" {
			minScore := farmRiskMinScore(provider, sensitive)
			result, err := VerifyHumanVerification(c.ClientIP(), token, HumanVerificationOptions{
				ExpectedAction: farmRiskExpectedAction(provider, action, requestAction),
				MinScore:       minScore,
				Version:        "v3",
			})
			if err != nil {
				// v3 评分未通过（分数过低 / 验证失败 / action 不匹配）→ 弹 v2 checkbox
				common.SysLog(fmt.Sprintf("[FarmRisk] v3 score fail → step-up v2: user=%d action=%s err=%s score=%.2f",
					userId, action, err.Error(), farmRiskResultScore(result)))
				if common.IsRecaptchaV2Configured() {
					respondFarmRiskStepUpV2(c, action, err.Error())
					return
				}
				// 没配 v2：视为失败（保守处理）
				farmRiskRecordFail(userId)
				respondFarmRiskVerifyFail(c, action, provider, err.Error())
				return
			}
			// v3 评分通过 → 直接放行
			farmRiskGrantPass(userId)
			farmRiskClearFail(userId)
			common.SysLog(fmt.Sprintf("[FarmRisk] v3 score pass: user=%d action=%s score=%.2f",
				userId, action, farmRiskResultScore(result)))
			c.Next()
			return
		}

		// 3) 没带 token：走 burst / sensitive 兜底
		//    非 sensitive + 持有 pass → 放行（降低正常用户的 burst 误伤）
		if !sensitive && farmRiskHasPass(userId) {
			c.Next()
			return
		}
		if needVerify {
			// sensitive 或 burst 超阈值 → 弹验证（v2 优先）
			respondFarmRiskStepUp(c, action, sensitive, provider)
			return
		}
		// 其他情况：v3 脚本未加载 / 用户未启用 recaptcha → 直接放行
		c.Next()
	}
}

// ═══════════════════════════════════════════════════════════════
//  锁定 / 通行证 / 失败计数 — Redis 优先，内存兜底
// ═══════════════════════════════════════════════════════════════

func farmRiskIsLocked(userId int) bool {
	if common.RedisEnabled {
		val, err := common.RDB.Get(context.Background(), farmRiskRedisKey("lock", userId)).Result()
		if err == nil && val == "1" {
			return true
		}
		return false
	}
	farmRiskMemMu.Lock()
	defer farmRiskMemMu.Unlock()
	e := farmRiskMem[userId]
	return e != nil && time.Now().Before(e.LockUntil)
}

func farmRiskHasPass(userId int) bool {
	if common.RedisEnabled {
		val, err := common.RDB.Get(context.Background(), farmRiskRedisKey("pass", userId)).Result()
		if err == nil && val == "1" {
			return true
		}
		return false
	}
	farmRiskMemMu.Lock()
	defer farmRiskMemMu.Unlock()
	e := farmRiskMem[userId]
	return e != nil && time.Now().Before(e.PassUntil)
}

func farmRiskGrantPass(userId int) {
	if common.RedisEnabled {
		_ = common.RDB.Set(context.Background(), farmRiskRedisKey("pass", userId), "1", farmRiskPassTTL).Err()
		return
	}
	farmRiskMemMu.Lock()
	defer farmRiskMemMu.Unlock()
	e := farmRiskGetOrCreate(userId)
	e.PassUntil = time.Now().Add(farmRiskPassTTL)
}

func farmRiskRecordFail(userId int) {
	if common.RedisEnabled {
		ctx := context.Background()
		key := farmRiskRedisKey("fail", userId)
		val, _ := common.RDB.Incr(ctx, key).Result()
		if val == 1 {
			_ = common.RDB.Expire(ctx, key, farmRiskFailWindow).Err()
		}
		if val >= int64(farmRiskMaxFail) {
			_ = common.RDB.Set(ctx, farmRiskRedisKey("lock", userId), "1", farmRiskLockTTL).Err()
			_ = common.RDB.Del(ctx, key).Err()
			_ = common.RDB.Del(ctx, farmRiskRedisKey("pass", userId)).Err()
			common.SysLog(fmt.Sprintf("[FarmRisk] user=%d locked for %v due to %d consecutive failures", userId, farmRiskLockTTL, farmRiskMaxFail))
		}
		return
	}
	farmRiskMemMu.Lock()
	defer farmRiskMemMu.Unlock()
	e := farmRiskGetOrCreate(userId)
	now := time.Now()
	if now.Sub(e.FailFirst) > farmRiskFailWindow {
		e.FailCount = 0
		e.FailFirst = now
	}
	if e.FailCount == 0 {
		e.FailFirst = now
	}
	e.FailCount++
	if e.FailCount >= farmRiskMaxFail {
		e.LockUntil = now.Add(farmRiskLockTTL)
		e.FailCount = 0
		e.PassUntil = time.Time{}
		common.SysLog(fmt.Sprintf("[FarmRisk] user=%d locked (in-memory) for %v due to %d consecutive failures", userId, farmRiskLockTTL, farmRiskMaxFail))
	}
}

func farmRiskClearFail(userId int) {
	if common.RedisEnabled {
		_ = common.RDB.Del(context.Background(), farmRiskRedisKey("fail", userId)).Err()
		return
	}
	farmRiskMemMu.Lock()
	defer farmRiskMemMu.Unlock()
	if e := farmRiskMem[userId]; e != nil {
		e.FailCount = 0
	}
}

// ── 突发计数 ──

func farmRiskRecordBurst(userId int) int {
	if common.RedisEnabled {
		return farmRiskRecordBurstRedis(userId)
	}
	return farmRiskRecordBurstMem(userId)
}

func farmRiskRecordBurstRedis(userId int) int {
	ctx := context.Background()
	key := farmRiskRedisKey("burst", userId)
	now := time.Now().UnixMilli()
	cutoff := time.Now().Add(-farmRiskBurstWindow).UnixMilli()
	pipe := common.RDB.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(cutoff, 10))
	pipe.ZAdd(ctx, key, &redis.Z{Score: float64(now), Member: strconv.FormatInt(now, 10)})
	cardCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, farmRiskBurstWindow+5*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return 1
	}
	return int(cardCmd.Val())
}

func farmRiskRecordBurstMem(userId int) int {
	now := time.Now()
	cutoff := now.Add(-farmRiskBurstWindow)
	farmRiskMemMu.Lock()
	defer farmRiskMemMu.Unlock()
	e := farmRiskGetOrCreate(userId)
	filtered := e.Timestamps[:0]
	for _, ts := range e.Timestamps {
		if ts.After(cutoff) {
			filtered = append(filtered, ts)
		}
	}
	e.Timestamps = append(filtered, now)
	return len(e.Timestamps)
}

func farmRiskGetOrCreate(userId int) *farmRiskMemEntry {
	e := farmRiskMem[userId]
	if e == nil {
		e = &farmRiskMemEntry{}
		farmRiskMem[userId] = e
	}
	return e
}

// ═══════════════════════════════════════════════════════════════
//  辅助函数
// ═══════════════════════════════════════════════════════════════

func normalizeFarmRiskAction(fullPath string) string {
	fullPath = strings.TrimSpace(fullPath)
	if fullPath == "" {
		return ""
	}
	action := strings.TrimPrefix(fullPath, "/api/")
	action = strings.ReplaceAll(action, "/visit/:friend_id/", "/visit_")
	action = strings.ReplaceAll(action, "/chat/:friend_id", "/chat")
	action = strings.ReplaceAll(action, "/friends/:friend_id", "/friends_remove")
	action = strings.ReplaceAll(action, "/", "_")
	action = strings.ReplaceAll(action, ":friend_id", "friend")
	return action
}

func farmRiskMinScore(provider string, sensitive bool) float64 {
	if provider != "recaptcha" {
		return 0
	}
	if sensitive {
		return farmRiskHighMinScore
	}
	return farmRiskDefaultMinScore
}

func farmRiskExpectedAction(provider, assessmentAction, requestAction string) string {
	if provider != "recaptcha" {
		return ""
	}
	if requestAction != "" {
		return requestAction
	}
	return assessmentAction
}

func farmRiskResultScore(result *HumanVerificationResult) float64 {
	if result == nil {
		return 0
	}
	return result.Score
}

// ── 响应构造 ──

func respondFarmRiskStepUp(c *gin.Context, action string, sensitive bool, provider string) {
	reason := "burst"
	if sensitive {
		reason = "sensitive_action"
	}
	// reCAPTCHA：如果配置了 v2，step-up 直接发 v2 checkbox（用户体验更直观、更快）
	// 只有在没配 v2 时才 fallback 到 v3 静默
	if provider == "recaptcha" && common.IsRecaptchaV2Configured() {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"code":    farmRiskStepUpCode,
			"message": "当前操作需要人机验证，请完成验证后重试",
			"data": gin.H{
				"action":   action,
				"reason":   reason,
				"provider": "recaptcha",
				"version":  "v2",
				"site_key": common.RecaptchaV2SiteKey,
			},
		})
		c.Abort()
		return
	}
	version := ""
	siteKey := common.GetHumanVerificationSiteKey()
	if provider == "recaptcha" {
		version = "v3"
	}
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"code":    farmRiskStepUpCode,
		"message": "当前操作需要人机验证，请完成验证后重试",
		"data": gin.H{
			"action":   action,
			"reason":   reason,
			"provider": provider,
			"version":  version,
			"site_key": siteKey,
		},
	})
	c.Abort()
}

// respondFarmRiskStepUpV2：v3 风控失败后要求用户完成 v2 checkbox
func respondFarmRiskStepUpV2(c *gin.Context, action, v3Reason string) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"code":    farmRiskStepUpCode,
		"message": "当前操作触发风控，请完成人机验证后重试",
		"data": gin.H{
			"action":    action,
			"reason":    "v3_fallback",
			"v3_reason": v3Reason,
			"provider":  "recaptcha",
			"version":   "v2",
			"site_key":  common.RecaptchaV2SiteKey,
		},
	})
	c.Abort()
}

func respondFarmRiskVerifyFail(c *gin.Context, action, provider, reason string) {
	// reCAPTCHA：如果配了 v2，失败重试也继续走 v2（用户体验一致）
	if provider == "recaptcha" && common.IsRecaptchaV2Configured() {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"code":    farmRiskVerifyFailCode,
			"message": "人机验证未通过，请重试",
			"data": gin.H{
				"action":   action,
				"reason":   reason,
				"provider": "recaptcha",
				"version":  "v2",
				"site_key": common.RecaptchaV2SiteKey,
			},
		})
		c.Abort()
		return
	}
	version := ""
	siteKey := common.GetHumanVerificationSiteKey()
	if provider == "recaptcha" {
		version = "v3"
	}
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"code":    farmRiskVerifyFailCode,
		"message": "人机验证未通过，请重试",
		"data": gin.H{
			"action":   action,
			"reason":   reason,
			"provider": provider,
			"version":  version,
			"site_key": siteKey,
		},
	})
	c.Abort()
}

func respondFarmRiskLocked(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"code":    farmRiskLockedCode,
		"message": fmt.Sprintf("由于多次验证失败，操作已被临时锁定 %d 分钟", int(farmRiskLockTTL.Minutes())),
	})
	c.Abort()
}
