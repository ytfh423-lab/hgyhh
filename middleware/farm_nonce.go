package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// ═══════════════════════════════════════════════════════════════
//  Farm Nonce Guard — 隐形计数器 + Nonce 防重放 + HMAC 签名绑定
//
//  核心原理（大白话版）：
//  ┌─────────────────────────────────────────────────────────┐
//  │ 1. 每个用户在服务端有一个「隐形计数器」，每次写操作+1     │
//  │ 2. 浏览器 Cookie 里有一个签名过的计数器副本               │
//  │ 3. 每次 POST 必须带一个「一次性随机数」(Nonce)            │
//  │ 4. 服务端校验：Nonce 没用过 + Cookie 计数器 = 服务端计数器 │
//  │ 5. 脚本偷了 Cookie → 用一次后计数器前进 → 旧 Cookie 作废  │
//  │ 6. 用户刷新页面（GET）→ 自动拿到最新计数器 → 正常使用     │
//  └─────────────────────────────────────────────────────────┘
// ═══════════════════════════════════════════════════════════════

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// 根密钥 K — 从 SESSION_SECRET 派生，跨重启稳定，绝不对外传输
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

var (
	farmRootKey     []byte
	farmRootKeyOnce sync.Once
)

func getFarmRootKey() []byte {
	farmRootKeyOnce.Do(func() {
		h := sha256.Sum256([]byte("farm_nonce_guard_v1:" + common.SessionSecret))
		farmRootKey = h[:]
		common.SysLog("Farm nonce guard: root key derived from SESSION_SECRET")
	})
	return farmRootKey
}

// farmHMACSign 用根密钥对数据做 HMAC-SHA256 签名
func farmHMACSign(data string) string {
	mac := hmac.New(sha256.New, getFarmRootKey())
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Nonce 存储 — 内存，仅存 60 秒，自动清理，永不爆盘
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

var (
	nonceStore       = make(map[string]time.Time) // nonce → 过期时间
	nonceMu          sync.Mutex
	nonceCleanerOnce sync.Once
)

const nonceTTL = 60 * time.Second

// initNonceCleaner 启动后台 goroutine，每 30 秒物理删除过期 Nonce
func initNonceCleaner() {
	nonceCleanerOnce.Do(func() {
		go func() {
			for {
				time.Sleep(30 * time.Second)
				now := time.Now()
				nonceMu.Lock()
				before := len(nonceStore)
				for k, expAt := range nonceStore {
					if now.After(expAt) {
						delete(nonceStore, k)
					}
				}
				after := len(nonceStore)
				nonceMu.Unlock()
				if before-after > 0 {
					common.SysLog(fmt.Sprintf("Farm nonce cleaner: purged %d expired nonces, %d remaining", before-after, after))
				}
			}
		}()
	})
}

// checkAndAddNonce 检查 nonce 是否已使用过（60秒内）
// 返回 true = 新 nonce（合法），false = 重复（重放攻击）
func checkAndAddNonce(nonce string) bool {
	initNonceCleaner()
	nonceMu.Lock()
	defer nonceMu.Unlock()
	if expAt, exists := nonceStore[nonce]; exists && time.Now().Before(expAt) {
		return false // 重复！
	}
	nonceStore[nonce] = time.Now().Add(nonceTTL)
	return true
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// 用户计数器存储 — 内存，每用户 1 条，自动清理不活跃条目
//
// 为什么不存数据库？
// → 计数器每次请求都要读写，放内存零延迟
// → 重启后丢失不影响安全（自动重新初始化）
// → 一人一条，10 万用户仅占 ~2MB
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

type farmUserCounter struct {
	Counter   int64
	UpdatedAt time.Time
}

var (
	farmCounterStore = make(map[int]*farmUserCounter)
	farmCounterMu    sync.RWMutex
)

func getFarmCounter(userId int) (int64, bool) {
	farmCounterMu.RLock()
	defer farmCounterMu.RUnlock()
	if c, ok := farmCounterStore[userId]; ok {
		return c.Counter, true
	}
	return 0, false
}

func setFarmCounter(userId int, counter int64) {
	farmCounterMu.Lock()
	defer farmCounterMu.Unlock()
	farmCounterStore[userId] = &farmUserCounter{Counter: counter, UpdatedAt: time.Now()}
	// 超过 5 万条时，清理 24 小时无活动的条目
	if len(farmCounterStore) > 50000 {
		cutoff := time.Now().Add(-24 * time.Hour)
		for k, v := range farmCounterStore {
			if v.UpdatedAt.Before(cutoff) {
				delete(farmCounterStore, k)
			}
		}
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Farm Token Cookie (_ft)
//
// 格式: HMAC前16字符.计数器
// 例如: a1b2c3d4e5f67890.42
//
// HMAC = HMAC-SHA256(根密钥K, "userId:counter")
// → 客户端无法伪造（不知道根密钥）
// → 服务端通过重算 HMAC 验证完整性
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func makeFarmToken(userId int, counter int64) string {
	data := fmt.Sprintf("%d:%d", userId, counter)
	sig := farmHMACSign(data)[:16] // 前 16 字符，节省 cookie 空间
	return fmt.Sprintf("%s.%d", sig, counter)
}

func parseFarmToken(token string, userId int) (int64, bool) {
	dot := strings.LastIndex(token, ".")
	if dot < 0 || dot == 0 {
		return 0, false
	}
	sigPart := token[:dot]
	counterStr := token[dot+1:]
	counter, err := strconv.ParseInt(counterStr, 10, 64)
	if err != nil {
		return 0, false
	}
	expected := farmHMACSign(fmt.Sprintf("%d:%d", userId, counter))[:16]
	if !hmac.Equal([]byte(sigPart), []byte(expected)) {
		return 0, false // HMAC 不匹配 → 伪造的
	}
	return counter, true
}

func setFarmTokenCookie(c *gin.Context, userId int, counter int64) {
	token := makeFarmToken(userId, counter)
	// HttpOnly=true → 前端 JS 读不到
	// Secure=false → 与 session cookie 保持一致（生产环境应改为 true）
	// SameSite 由全局 cookie 配置控制
	c.SetCookie("_ft", token, 7*86400, "/", "", false, true)
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// 多标签页容忍度
//
// 场景：用户开了两个标签页，几乎同时点了两个操作
// → 第一个请求成功后 counter+1，第二个请求的 cookie 会落后 1 步
// → 允许落后最多 2 步，正常使用完全不受影响
// → 脚本无法利用（即使落后也只多 2 次机会）
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

const farmCounterTolerance int64 = 2

// ═══════════════════════════════════════════════════════════════
//  FarmNonceGuard — 核心中间件
//
//  挂载位置：farmRoute / ranchRoute / treeFarmRoute
//  执行顺序：UserAuth → FarmSessionOnly → FarmNonceGuard → ...
// ═══════════════════════════════════════════════════════════════

func FarmNonceGuard() gin.HandlerFunc {
	initNonceCleaner()
	return func(c *gin.Context) {
		userId := c.GetInt("id")
		if userId == 0 {
			c.Next()
			return
		}

		// ─── GET/HEAD/OPTIONS: 同步计数器 → 设置 Cookie ───
		// 用户打开农场页面或轮询时，自动拿到最新计数器
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			counter, exists := getFarmCounter(userId)
			if !exists {
				counter = 0
				setFarmCounter(userId, counter)
			}
			setFarmTokenCookie(c, userId, counter)
			c.Next()
			return
		}

		// ─── POST/PUT/DELETE: 完整三重校验 ───

		// 【校验 1】Nonce 唯一性（防重放）
		nonce := c.GetHeader("X-Farm-Nonce")
		if nonce == "" || len(nonce) < 16 || len(nonce) > 64 {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "请求缺少安全凭证，请刷新页面重试",
			})
			c.Abort()
			return
		}
		if !checkAndAddNonce(nonce) {
			common.SysLog(fmt.Sprintf("Farm nonce replay: user=%d nonce=%.8s...", userId, nonce))
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "重复请求，请稍后重试",
			})
			c.Abort()
			return
		}

		// 【校验 2】Farm Token Cookie 签名验证（防伪造）
		tokenStr, err := c.Cookie("_ft")
		if err != nil || tokenStr == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "缺少农场安全凭证，请刷新页面",
			})
			c.Abort()
			return
		}
		cookieCounter, valid := parseFarmToken(tokenStr, userId)
		if !valid {
			common.SysLog(fmt.Sprintf("Farm token forged: user=%d", userId))
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "农场凭证无效，请刷新页面",
			})
			c.Abort()
			return
		}

		// 【校验 3】计数器比对（核心防盗机制）
		serverCounter, exists := getFarmCounter(userId)
		if !exists {
			// 服务重启后首次请求 → 以 cookie counter 为准重建
			setFarmCounter(userId, cookieCounter+1)
			setFarmTokenCookie(c, userId, cookieCounter+1)
			c.Next()
			return
		}

		// cookie counter 必须在 [serverCounter - tolerance, serverCounter] 范围内
		// 落后太多 = 被别的客户端推进了（脚本/其他设备）
		// 超前 = 伪造
		if cookieCounter < serverCounter-farmCounterTolerance || cookieCounter > serverCounter {
			common.SysLog(fmt.Sprintf(
				"Farm counter mismatch: user=%d cookie=%d server=%d → possible script or session theft",
				userId, cookieCounter, serverCounter))
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "农场会话已过期，请刷新页面重试",
			})
			c.Abort()
			return
		}

		// ─── 全部通过 → 推进计数器 + 下发新 Cookie ───
		newCounter := serverCounter + 1
		setFarmCounter(userId, newCounter)
		setFarmTokenCookie(c, userId, newCounter)

		c.Next()
	}
}
