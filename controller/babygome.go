package controller

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var babyGomeClient = &http.Client{Timeout: 8 * time.Second}

// GetBabyGome proxies the xunjinlu babygome API to avoid browser CORS issues.
func GetBabyGome(c *gin.Context) {
	resp, err := babyGomeClient.Get("https://api.xunjinlu.fun/api/babygome/index.php?type=json")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "upstream request failed"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "failed to read upstream response"})
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}
