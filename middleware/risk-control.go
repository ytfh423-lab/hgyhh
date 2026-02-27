package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

// concurrentTracker tracks the number of in-flight requests per key (in-memory mode)
type concurrentTracker struct {
	mu       sync.Mutex
	counters map[string]*int64
}

var globalConcurrentTracker = &concurrentTracker{
	counters: make(map[string]*int64),
}

func (ct *concurrentTracker) getCounter(key string) *int64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if c, ok := ct.counters[key]; ok {
		return c
	}
	var counter int64
	ct.counters[key] = &counter
	return ct.counters[key]
}

func (ct *concurrentTracker) acquire(key string, limit int) bool {
	counter := ct.getCounter(key)
	for {
		current := atomic.LoadInt64(counter)
		if current >= int64(limit) {
			return false
		}
		if atomic.CompareAndSwapInt64(counter, current, current+1) {
			return true
		}
	}
}

func (ct *concurrentTracker) release(key string) {
	counter := ct.getCounter(key)
	atomic.AddInt64(counter, -1)
}

// redisConcurrentAcquire uses Redis INCR with TTL for distributed concurrent tracking
func redisConcurrentAcquire(key string, limit int) (bool, error) {
	ctx := context.Background()
	rdb := common.RDB

	current, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	// Set expiry on first creation to auto-cleanup (safety net)
	if current == 1 {
		rdb.Expire(ctx, key, 5*time.Minute)
	}

	if current > int64(limit) {
		rdb.Decr(ctx, key)
		return false, nil
	}
	return true, nil
}

func redisConcurrentRelease(key string) {
	ctx := context.Background()
	rdb := common.RDB
	rdb.Decr(ctx, key)
}

// redisBurstCheck implements burst rate limiting using Redis LLen-based sliding window
// (consistent with existing rate-limit.go pattern in this codebase)
func redisBurstCheck(c *gin.Context, key string, limit int, windowSeconds int) bool {
	ctx := context.Background()
	rdb := common.RDB

	listLength, err := rdb.LLen(ctx, key).Result()
	if err != nil {
		fmt.Println("risk control burst check LLen error:", err.Error())
		return true // fail open
	}

	expiration := time.Duration(windowSeconds+10) * time.Second

	if listLength < int64(limit) {
		rdb.LPush(ctx, key, time.Now().Format(timeFormat))
		rdb.Expire(ctx, key, expiration)
		return true
	}

	// Check if oldest entry is outside the window
	oldTimeStr, _ := rdb.LIndex(ctx, key, -1).Result()
	oldTime, err := time.Parse(timeFormat, oldTimeStr)
	if err != nil {
		fmt.Println("risk control burst check parse error:", err.Error())
		return true // fail open
	}

	nowStr := time.Now().Format(timeFormat)
	nowTime, err := time.Parse(timeFormat, nowStr)
	if err != nil {
		return true
	}

	if int64(nowTime.Sub(oldTime).Seconds()) < int64(windowSeconds) {
		rdb.Expire(ctx, key, expiration)
		return false // rate limited
	}

	rdb.LPush(ctx, key, nowStr)
	rdb.LTrim(ctx, key, 0, int64(limit-1))
	rdb.Expire(ctx, key, expiration)
	return true
}

// RequestRiskControl is the main risk control middleware.
// It enforces:
//  1. Burst rate limiting: max N requests per short time window (per token key)
//  2. Concurrent request limiting: max N simultaneous in-flight requests (per token key)
//
// This effectively blocks high-frequency burst requests and translation service
// abuse patterns (e.g., Immersive Translate sending 10+ parallel requests).
func RequestRiskControl() func(c *gin.Context) {
	return func(c *gin.Context) {
		if !setting.RequestRiskControlEnabled {
			c.Next()
			return
		}

		tokenId := c.GetInt("token_id")
		userId := c.GetInt("id")
		if tokenId == 0 && userId == 0 {
			c.Next()
			return
		}

		burstLimit := setting.RequestRiskControlBurstLimit
		burstWindow := setting.RequestRiskControlBurstWindow
		concurrentLimit := setting.RequestRiskControlTokenThreshold

		// Use token ID as primary key for more granular control
		keyId := fmt.Sprintf("t:%d", tokenId)
		if tokenId == 0 {
			keyId = fmt.Sprintf("u:%d", userId)
		}

		if common.RedisEnabled {
			riskControlRedis(c, keyId, burstLimit, burstWindow, concurrentLimit)
		} else {
			riskControlMemory(c, keyId, burstLimit, burstWindow, concurrentLimit)
		}
	}
}

func riskControlRedis(c *gin.Context, keyId string, burstLimit, burstWindow, concurrentLimit int) {
	burstKey := fmt.Sprintf("rc:burst:%s", keyId)
	concurrentKey := fmt.Sprintf("rc:conc:%s", keyId)

	// 1. Check burst rate limit
	if burstLimit > 0 && burstWindow > 0 {
		if !redisBurstCheck(c, burstKey, burstLimit, burstWindow) {
			abortWithOpenAiMessage(c, http.StatusTooManyRequests,
				fmt.Sprintf("请求频率过高，%d秒内最多允许%d次请求，请降低请求频率",
					burstWindow, burstLimit))
			return
		}
	}

	// 2. Check concurrent request limit
	if concurrentLimit > 0 {
		allowed, err := redisConcurrentAcquire(concurrentKey, concurrentLimit)
		if err != nil {
			fmt.Println("risk control concurrent check error:", err.Error())
		} else if !allowed {
			abortWithOpenAiMessage(c, http.StatusTooManyRequests,
				fmt.Sprintf("并发请求数超限，最多允许%d个并发请求，请减少同时发送的请求数量",
					concurrentLimit))
			return
		}
		defer redisConcurrentRelease(concurrentKey)
	}

	c.Next()
}

func riskControlMemory(c *gin.Context, keyId string, burstLimit, burstWindow, concurrentLimit int) {
	burstKey := "rc:burst:" + keyId
	concurrentKey := "rc:conc:" + keyId

	// 1. Check burst rate limit
	if burstLimit > 0 && burstWindow > 0 {
		inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
		if !inMemoryRateLimiter.Request(burstKey, burstLimit, int64(burstWindow)) {
			abortWithOpenAiMessage(c, http.StatusTooManyRequests,
				fmt.Sprintf("请求频率过高，%d秒内最多允许%d次请求，请降低请求频率",
					burstWindow, burstLimit))
			return
		}
	}

	// 2. Check concurrent request limit
	if concurrentLimit > 0 {
		if !globalConcurrentTracker.acquire(concurrentKey, concurrentLimit) {
			abortWithOpenAiMessage(c, http.StatusTooManyRequests,
				fmt.Sprintf("并发请求数超限，最多允许%d个并发请求，请减少同时发送的请求数量",
					concurrentLimit))
			return
		}
		defer globalConcurrentTracker.release(concurrentKey)
	}

	c.Next()
}
