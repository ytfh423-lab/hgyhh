package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────
// 1. FarmSessionOnly — 禁止 access token 调用农场接口
//    农场是纯网页功能，外部脚本不应通过 API token 直调。
// ─────────────────────────────────────────────────────────

func FarmSessionOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("use_access_token") {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "农场功能仅支持网页访问，不支持 API 调用",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ─────────────────────────────────────────────────────────
// 2. FarmActionRateLimit — 按用户 ID 限速农场写操作
//    每用户每 60 秒最多 10 次 POST，防止脚本高频自动化。
//    GET 请求（页面自动刷新/轮询）不计入。
// ─────────────────────────────────────────────────────────

func FarmActionRateLimit() gin.HandlerFunc {
	limiter := userRateLimitFactory(10, 60, "FARM")
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" {
			limiter(c)
			if c.IsAborted() {
				return
			}
		}
		c.Next()
	}
}

// ─────────────────────────────────────────────────────────
// 3. FarmDailyActionCap — 每用户每天农场写操作总量上限
//    超过上限直接拒绝，防止长时间低频脚本（如每 10 分钟一轮）
//    正常玩家一天很难超过 500 次写操作。
// ─────────────────────────────────────────────────────────

const farmDailyActionCap = 500

var (
	farmDailyCounters   = make(map[string]*farmDayCounter)
	farmDailyCountersMu sync.Mutex
)

type farmDayCounter struct {
	Day   string
	Count int
}

func FarmDailyActionCap() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "POST" && c.Request.Method != "PUT" && c.Request.Method != "DELETE" {
			c.Next()
			return
		}
		userId := c.GetInt("id")
		if userId == 0 {
			c.Next()
			return
		}

		today := time.Now().Format("2006-01-02")
		key := fmt.Sprintf("%d", userId)

		farmDailyCountersMu.Lock()
		counter, ok := farmDailyCounters[key]
		if !ok || counter.Day != today {
			counter = &farmDayCounter{Day: today, Count: 0}
			farmDailyCounters[key] = counter
		}
		counter.Count++
		current := counter.Count
		// 顺便清理过期条目（低频操作，不影响性能）
		if len(farmDailyCounters) > 10000 {
			for k, v := range farmDailyCounters {
				if v.Day != today {
					delete(farmDailyCounters, k)
				}
			}
		}
		farmDailyCountersMu.Unlock()

		if current > farmDailyActionCap {
			common.SysLog(fmt.Sprintf("Farm anti-script: user %d exceeded daily action cap (%d/%d)", userId, current, farmDailyActionCap))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "今日农场操作次数已达上限，请明天再来",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
