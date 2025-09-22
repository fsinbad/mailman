package services

import (
	"fmt"
	"sync"
	"time"

	"mailman/internal/models"
	"github.com/patrickmn/go-cache"
)

// ResultCache 提供触发器执行结果的缓存
type ResultCache struct {
	cache      *cache.Cache
	mutex      sync.RWMutex
	enabled    bool
	hitCounter int64
	missCounter int64
}

// NewResultCache 创建一个新的结果缓存
func NewResultCache(defaultExpiration, cleanupInterval time.Duration) *ResultCache {
	return &ResultCache{
		cache:   cache.New(defaultExpiration, cleanupInterval),
		enabled: true,
	}
}

// SetEnabled 设置缓存是否启用
func (c *ResultCache) SetEnabled(enabled bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.enabled = enabled
}

// IsEnabled 检查缓存是否启用
func (c *ResultCache) IsEnabled() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.enabled
}

// GetHitRate 获取缓存命中率
func (c *ResultCache) GetHitRate() float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	total := c.hitCounter + c.missCounter
	if total == 0 {
		return 0
	}
	
	return float64(c.hitCounter) / float64(total)
}

// GetStats 获取缓存统计信息
func (c *ResultCache) GetStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	total := c.hitCounter + c.missCounter
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hitCounter) / float64(total)
	}
	
	items := c.cache.ItemCount()
	
	return map[string]interface{}{
		"enabled":     c.enabled,
		"items":       items,
		"hits":        c.hitCounter,
		"misses":      c.missCounter,
		"total":       total,
		"hit_rate":    hitRate,
	}
}

// Clear 清除缓存
func (c *ResultCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache.Flush()
	c.hitCounter = 0
	c.missCounter = 0
}

// Get 从缓存获取条件评估结果
func (c *ResultCache) Get(triggerID uint, emailID uint) (bool, models.JSONMap, bool) {
	if !c.IsEnabled() {
		return false, nil, false
	}
	
	key := c.generateKey(triggerID, emailID)
	
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	if data, found := c.cache.Get(key); found {
		c.hitCounter++
		result := data.(struct {
			Result  bool
			Details models.JSONMap
		})
		return result.Result, result.Details, true
	}
	
	c.missCounter++
	return false, nil, false
}

// Set 将条件评估结果存入缓存
func (c *ResultCache) Set(triggerID uint, emailID uint, result bool, details models.JSONMap, duration time.Duration) {
	if !c.IsEnabled() {
		return
	}
	
	key := c.generateKey(triggerID, emailID)
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache.Set(key, struct {
		Result  bool
		Details models.JSONMap
	}{
		Result:  result,
		Details: details,
	}, duration)
}

// Delete 从缓存中删除条件评估结果
func (c *ResultCache) Delete(triggerID uint, emailID uint) {
	key := c.generateKey(triggerID, emailID)
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache.Delete(key)
}

// generateKey 生成缓存键
func (c *ResultCache) generateKey(triggerID uint, emailID uint) string {
	return fmt.Sprintf("trigger:%d:email:%d", triggerID, emailID)
}