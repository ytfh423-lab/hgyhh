package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func Cache() func(c *gin.Context) {
	return func(c *gin.Context) {
		uri := c.Request.RequestURI
		// Don't cache API routes or root path
		if uri == "/" || strings.HasPrefix(uri, "/api/") || strings.HasPrefix(uri, "/v1/") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		} else if strings.HasPrefix(uri, "/assets/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "max-age=604800") // one week
		}
		c.Header("Cache-Version", "b688f2fb5be447c25e5aa3bd063087a83db32a288bf6a4f35f2d8db310e40b14")
		c.Next()
	}
}
