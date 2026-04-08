package controller

import (
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/samber/hot"
)

const userGroupsCacheNamespace = "new-api:user_groups:v1"

var (
	userGroupsCacheOnce sync.Once
	userGroupsCache     *cachex.HybridCache[map[string]map[string]interface{}]
)

func userGroupsCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("USER_GROUPS_CACHE_TTL", 30)
	if ttlSeconds <= 0 {
		ttlSeconds = 30
	}
	return time.Duration(ttlSeconds) * time.Second
}

func userGroupsCacheCapacity() int {
	capacity := common.GetEnvOrDefault("USER_GROUPS_CACHE_CAP", 256)
	if capacity <= 0 {
		capacity = 256
	}
	return capacity
}

func getUserGroupsCache() *cachex.HybridCache[map[string]map[string]interface{}] {
	userGroupsCacheOnce.Do(func() {
		ttl := userGroupsCacheTTL()
		userGroupsCache = cachex.NewHybridCache[map[string]map[string]interface{}](cachex.HybridCacheConfig[map[string]map[string]interface{}]{
			Namespace: cachex.Namespace(userGroupsCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[map[string]map[string]interface{}]{},
			Memory: func() *hot.HotCache[string, map[string]map[string]interface{}] {
				return hot.NewHotCache[string, map[string]map[string]interface{}](hot.LRU, userGroupsCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return userGroupsCache
}

func getUserGroupsCacheKey(group string) string {
	group = strings.TrimSpace(group)
	if group == "" {
		return "guest"
	}
	return group
}

func buildUserGroupsData(userGroup string) map[string]map[string]interface{} {
	usableGroups := make(map[string]map[string]interface{})
	userUsableGroups := service.GetUserUsableGroups(userGroup)
	groupNames := make([]string, 0, len(ratio_setting.GetGroupRatioCopy()))
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupNames = append(groupNames, groupName)
	}
	sort.Strings(groupNames)
	for _, groupName := range groupNames {
		if desc, ok := userUsableGroups[groupName]; ok {
			usableGroups[groupName] = map[string]interface{}{
				"ratio": service.GetUserGroupRatio(userGroup, groupName),
				"desc":  desc,
			}
		}
	}
	if _, ok := userUsableGroups["auto"]; ok {
		usableGroups["auto"] = map[string]interface{}{
			"ratio": "自动",
			"desc":  setting.GetUsableGroupDescription("auto"),
		}
	}
	return usableGroups
}

func GetGroups(c *gin.Context) {
	groupNames := make([]string, 0)
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupNames = append(groupNames, groupName)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    groupNames,
	})
}

func GetUserGroups(c *gin.Context) {
	userId := c.GetInt("id")
	userGroup, _ := model.GetUserGroup(userId, false)
	cacheKey := getUserGroupsCacheKey(userGroup)
	cache := getUserGroupsCache()
	usableGroups, found, cacheErr := cache.Get(cacheKey)
	if cacheErr != nil {
		common.SysLog("GetUserGroups cache get failed: " + cacheErr.Error())
	}
	if !found {
		usableGroups = buildUserGroupsData(userGroup)
		if err := cache.SetWithTTL(cacheKey, usableGroups, userGroupsCacheTTL()); err != nil {
			common.SysLog("GetUserGroups cache set failed: " + err.Error())
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}
