package middleware

import (
	"net/http"

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
	limiter := userRateLimitFactory(
		common.FarmActionRateLimitNum,
		common.FarmActionRateLimitDuration,
		"FARM",
	)
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
