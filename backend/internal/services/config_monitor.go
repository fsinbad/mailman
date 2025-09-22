package services

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"mailman/internal/models"
	"mailman/internal/repository"
)

// ConfigChangeType 配置变更类型
type ConfigChangeType string

const (
	ConfigAdded    ConfigChangeType = "added"
	ConfigUpdated  ConfigChangeType = "updated"
	ConfigDeleted  ConfigChangeType = "deleted"
	ConfigEnabled  ConfigChangeType = "enabled"
	ConfigDisabled ConfigChangeType = "disabled"
)

// ConfigChangeEvent 配置变更事件
type ConfigChangeEvent struct {
	Type      ConfigChangeType               `json:"type"`
	AccountID uint                           `json:"account_id"`
	OldConfig *models.EmailAccountSyncConfig `json:"old_config,omitempty"`
	NewConfig *models.EmailAccountSyncConfig `json:"new_config,omitempty"`
	Timestamp time.Time                      `json:"timestamp"`
}

// MinimalSyncConfig 最小化配置缓存
type MinimalSyncConfig struct {
	AccountID      uint      `json:"account_id"`
	EnableAutoSync bool      `json:"enable_auto_sync"`
	SyncInterval   int       `json:"sync_interval"`
	LastModified   time.Time `json:"last_modified"`
	ConfigHash     uint64    `json:"config_hash"`
}

// ConfigCache 线程安全的配置缓存
type ConfigCache struct {
	configs     map[uint]*MinimalSyncConfig
	mu          sync.RWMutex
	lastUpdate  time.Time
	updateCount int64
}

// ConfigMonitorStats 配置监控统计
type ConfigMonitorStats struct {
	TotalChecks     int64         `json:"total_checks"`
	ChangesDetected int64         `json:"changes_detected"`
	LastCheckTime   time.Time     `json:"last_check_time"`
	CheckDuration   time.Duration `json:"avg_check_duration"`
	CacheSize       int           `json:"cache_size"`
}

// FastConfigMonitor 快速配置监控器
type FastConfigMonitor struct {
	cache          *ConfigCache
	syncConfigRepo *repository.SyncConfigRepository
	manager        *PerAccountSyncManager

	// 配置变更通道
	changes chan ConfigChangeEvent

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 统计
	stats ConfigMonitorStats
}

// NewFastConfigMonitor 创建快速配置监控器
func NewFastConfigMonitor(repo *repository.SyncConfigRepository, manager *PerAccountSyncManager) *FastConfigMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &FastConfigMonitor{
		cache: &ConfigCache{
			configs: make(map[uint]*MinimalSyncConfig),
		},
		syncConfigRepo: repo,
		manager:        manager,
		changes:        make(chan ConfigChangeEvent), // 无缓冲channel
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start 启动配置监控器
func (m *FastConfigMonitor) Start() error {
	// 初始化缓存
	if err := m.initializeCache(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// 启动配置变更处理器
	m.wg.Add(1)
	go m.handleConfigChanges()

	// 启动每秒检测器
	m.wg.Add(1)
	go m.highFrequencyMonitor()

	log.Printf("[FastConfigMonitor] Started with %d cached configs", len(m.cache.configs))
	return nil
}

// Stop 停止配置监控器
func (m *FastConfigMonitor) Stop() {
	log.Printf("[FastConfigMonitor] Stopping...")
	m.cancel()
	close(m.changes)
	m.wg.Wait()
	log.Printf("[FastConfigMonitor] Stopped")
}

// initializeCache 初始化缓存
func (m *FastConfigMonitor) initializeCache() error {
	configs, err := m.syncConfigRepo.GetAllConfigsWithAccounts()
	if err != nil {
		return err
	}

	m.cache.mu.Lock()
	defer m.cache.mu.Unlock()

	for _, config := range configs {
		minimalConfig := &MinimalSyncConfig{
			AccountID:      config.AccountID,
			EnableAutoSync: config.EnableAutoSync,
			SyncInterval:   config.SyncInterval,
			LastModified:   config.UpdatedAt,
			ConfigHash:     m.calculateConfigHash(config),
		}
		m.cache.configs[config.AccountID] = minimalConfig
	}

	m.cache.lastUpdate = time.Now()
	log.Printf("[FastConfigMonitor] Initialized cache with %d configs", len(m.cache.configs))

	return nil
}

// highFrequencyMonitor 每秒检测配置变更
func (m *FastConfigMonitor) highFrequencyMonitor() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Printf("[FastConfigMonitor] High frequency monitor started (1s interval)")

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			start := time.Now()
			m.checkConfigChanges()

			// 更新统计信息
			atomic.AddInt64(&m.stats.TotalChecks, 1)
			m.stats.CheckDuration = time.Since(start)
			m.stats.LastCheckTime = time.Now()
		}
	}
}

// checkConfigChanges 检查配置变更
func (m *FastConfigMonitor) checkConfigChanges() {
	// 获取最近更新的配置
	configs, err := m.syncConfigRepo.GetRecentlyModifiedConfigs(m.cache.lastUpdate)
	if err != nil {
		log.Printf("[FastConfigMonitor] Failed to check configs: %v", err)
		return
	}

	if len(configs) == 0 {
		return // 没有变更
	}

	// 处理变更
	changes := m.processConfigUpdates(configs)

	// 发送变更事件
	for _, change := range changes {
		select {
		case m.changes <- change:
			atomic.AddInt64(&m.stats.ChangesDetected, 1)
		case <-m.ctx.Done():
			return
		default:
			log.Printf("[FastConfigMonitor] Change channel full, dropping event for account %d", change.AccountID)
		}
	}
}

// processConfigUpdates 处理配置更新
func (m *FastConfigMonitor) processConfigUpdates(configs []models.EmailAccountSyncConfig) []ConfigChangeEvent {
	var changes []ConfigChangeEvent

	m.cache.mu.Lock()
	defer m.cache.mu.Unlock()

	for _, config := range configs {
		newMinimalConfig := &MinimalSyncConfig{
			AccountID:      config.AccountID,
			EnableAutoSync: config.EnableAutoSync,
			SyncInterval:   config.SyncInterval,
			LastModified:   config.UpdatedAt,
			ConfigHash:     m.calculateConfigHash(config),
		}

		oldConfig, exists := m.cache.configs[config.AccountID]

		if !exists {
			// 新增配置
			m.cache.configs[config.AccountID] = newMinimalConfig
			changes = append(changes, ConfigChangeEvent{
				Type:      ConfigAdded,
				AccountID: config.AccountID,
				NewConfig: &config,
				Timestamp: time.Now(),
			})

		} else if oldConfig.ConfigHash != newMinimalConfig.ConfigHash {
			// 配置发生变更
			changeType := m.detectChangeType(oldConfig, newMinimalConfig)

			// 更新缓存
			m.cache.configs[config.AccountID] = newMinimalConfig

			changes = append(changes, ConfigChangeEvent{
				Type:      changeType,
				AccountID: config.AccountID,
				OldConfig: m.convertToFullConfig(oldConfig),
				NewConfig: &config,
				Timestamp: time.Now(),
			})
		}

		// 更新最后检查时间
		if config.UpdatedAt.After(m.cache.lastUpdate) {
			m.cache.lastUpdate = config.UpdatedAt
		}
	}

	atomic.AddInt64(&m.cache.updateCount, 1)
	return changes
}

// calculateConfigHash 计算配置哈希
func (m *FastConfigMonitor) calculateConfigHash(config models.EmailAccountSyncConfig) uint64 {
	h := fnv.New64a()
	hashStr := fmt.Sprintf("%t:%d:%v",
		config.EnableAutoSync,
		config.SyncInterval,
		config.SyncFolders,
	)
	h.Write([]byte(hashStr))
	return h.Sum64()
}

// detectChangeType 检测变更类型
func (m *FastConfigMonitor) detectChangeType(old, new *MinimalSyncConfig) ConfigChangeType {
	if old.EnableAutoSync != new.EnableAutoSync {
		if new.EnableAutoSync {
			return ConfigEnabled
		} else {
			return ConfigDisabled
		}
	}
	return ConfigUpdated
}

// convertToFullConfig 转换为完整配置（占位实现）
func (m *FastConfigMonitor) convertToFullConfig(minimal *MinimalSyncConfig) *models.EmailAccountSyncConfig {
	return &models.EmailAccountSyncConfig{
		AccountID:      minimal.AccountID,
		EnableAutoSync: minimal.EnableAutoSync,
		SyncInterval:   minimal.SyncInterval,
	}
}

// handleConfigChanges 处理配置变更（空实现，由manager处理）
func (m *FastConfigMonitor) handleConfigChanges() {
	defer m.wg.Done()
	// 这个方法由PerAccountSyncManager处理
}

// GetStats 获取统计信息
func (m *FastConfigMonitor) GetStats() ConfigMonitorStats {
	m.cache.mu.RLock()
	cacheSize := len(m.cache.configs)
	m.cache.mu.RUnlock()

	return ConfigMonitorStats{
		TotalChecks:     atomic.LoadInt64(&m.stats.TotalChecks),
		ChangesDetected: atomic.LoadInt64(&m.stats.ChangesDetected),
		LastCheckTime:   m.stats.LastCheckTime,
		CheckDuration:   m.stats.CheckDuration,
		CacheSize:       cacheSize,
	}
}

// GetConfig 获取配置
func (m *FastConfigMonitor) GetConfig(accountID uint) (*MinimalSyncConfig, bool) {
	m.cache.mu.RLock()
	defer m.cache.mu.RUnlock()

	config, exists := m.cache.configs[accountID]
	if !exists {
		return nil, false
	}

	// 返回副本
	return &MinimalSyncConfig{
		AccountID:      config.AccountID,
		EnableAutoSync: config.EnableAutoSync,
		SyncInterval:   config.SyncInterval,
		LastModified:   config.LastModified,
		ConfigHash:     config.ConfigHash,
	}, true
}
