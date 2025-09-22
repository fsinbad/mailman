package services

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mailman/internal/models"
	"mailman/internal/repository"
	"mailman/internal/utils"
)

// PerAccountSyncManager 每账户独立goroutine的同步管理器
type PerAccountSyncManager struct {
	// 账户同步器映射
	accountSyncers map[uint]*AccountSyncer
	mu             sync.RWMutex

	// 配置监控
	configMonitor *FastConfigMonitor

	// 控制和生命周期
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 并发控制
	semaphore chan struct{} // 控制并发网络请求数量

	// 服务依赖
	syncConfigRepo   *repository.SyncConfigRepository
	emailRepo        *repository.EmailRepository
	mailboxRepo      *repository.MailboxRepository
	emailAccountRepo *repository.EmailAccountRepository
	fetcherService   *FetcherService
	activityLogger   *ActivityLogger
	logger           *utils.Logger

	// 通知系统
	notificationService *EmailNotificationService

	// 监控统计
	stats PerAccountSyncStats
}

// AccountSyncer 单个账户的同步器
type AccountSyncer struct {
	AccountID    uint
	Config       models.EmailAccountSyncConfig
	LastSyncTime time.Time

	// 定时器
	timer   *time.Timer
	timerMu sync.Mutex

	// 控制
	ctx    context.Context
	cancel context.CancelFunc

	// 管理器引用
	manager *PerAccountSyncManager
	logger  *utils.Logger

	// 状态
	isRunning bool
	mu        sync.RWMutex

	// 统计
	syncCount     int64
	errorCount    int64
	lastError     error
	lastErrorTime time.Time
}

// PerAccountSyncStats 管理器统计信息
type PerAccountSyncStats struct {
	ActiveSyncers     int64     `json:"active_syncers"`
	TotalSyncers      int64     `json:"total_syncers"`
	TotalSyncs        int64     `json:"total_syncs"`
	TotalErrors       int64     `json:"total_errors"`
	ConcurrentLimit   int       `json:"concurrent_limit"`
	CurrentConcurrent int64     `json:"current_concurrent"`
	StartTime         time.Time `json:"start_time"`
}

// NewPerAccountSyncManager 创建每账户同步管理器
func NewPerAccountSyncManager(
	syncConfigRepo *repository.SyncConfigRepository,
	emailRepo *repository.EmailRepository,
	mailboxRepo *repository.MailboxRepository,
	emailAccountRepo *repository.EmailAccountRepository,
	fetcherService *FetcherService,
	notificationService *EmailNotificationService,
) *PerAccountSyncManager {
	ctx, cancel := context.WithCancel(context.Background())

	// 根据系统资源计算并发限制
	concurrentLimit := calculateConcurrentLimit()

	manager := &PerAccountSyncManager{
		accountSyncers:      make(map[uint]*AccountSyncer),
		ctx:                 ctx,
		cancel:              cancel,
		semaphore:           make(chan struct{}, concurrentLimit),
		syncConfigRepo:      syncConfigRepo,
		emailRepo:           emailRepo,
		mailboxRepo:         mailboxRepo,
		emailAccountRepo:    emailAccountRepo,
		fetcherService:      fetcherService,
		activityLogger:      GetActivityLogger(),
		logger:              utils.NewLogger("PerAccountSyncManager"),
		notificationService: notificationService,
		stats: PerAccountSyncStats{
			StartTime:       time.Now(),
			ConcurrentLimit: concurrentLimit,
		},
	}

	// 创建配置监控器
	manager.configMonitor = NewFastConfigMonitor(syncConfigRepo, manager)

	return manager
}

// calculateConcurrentLimit 根据系统资源计算并发限制
func calculateConcurrentLimit() int {
	cpuCount := runtime.NumCPU()

	// 基础策略：每个CPU核心允许10个并发请求
	limit := cpuCount * 10

	// 最小20，最大200
	if limit < 20 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	log.Printf("[PerAccountSyncManager] Calculated concurrent limit: %d (CPU cores: %d)", limit, cpuCount)
	return limit
}

// Start 启动管理器
func (m *PerAccountSyncManager) Start() error {
	m.logger.Info("Starting per-account sync manager")

	// 启动配置监控器
	if err := m.configMonitor.Start(); err != nil {
		return fmt.Errorf("failed to start config monitor: %w", err)
	}

	// 启动配置变更处理器
	m.wg.Add(1)
	go m.handleConfigChanges()

	// 启动统计更新器
	m.wg.Add(1)
	go m.updateStatsRoutine()

	// 启动清理例程
	m.wg.Add(1)
	go m.cleanupRoutine()

	// 加载现有配置并启动AccountSyncer
	if err := m.loadExistingConfigs(); err != nil {
		m.logger.Error("Failed to load existing configs: %v", err)
		return err
	}

	m.logger.Info("Per-account sync manager started with %d active syncers", len(m.accountSyncers))
	return nil
}

// Stop 停止管理器
func (m *PerAccountSyncManager) Stop() {
	m.logger.Info("Stopping per-account sync manager")

	// 停止配置监控器
	m.configMonitor.Stop()

	// 取消上下文
	m.cancel()

	// 停止所有AccountSyncer
	m.mu.Lock()
	for accountID, syncer := range m.accountSyncers {
		m.logger.Debug("Stopping syncer for account %d", accountID)
		syncer.Stop()
	}
	m.accountSyncers = make(map[uint]*AccountSyncer) // 清空映射
	m.mu.Unlock()

	// 等待所有goroutine退出
	m.wg.Wait()

	m.logger.Info("Per-account sync manager stopped")
}

// loadExistingConfigs 加载现有配置
func (m *PerAccountSyncManager) loadExistingConfigs() error {
	m.logger.Debug("Starting to load existing sync configurations from database")
	configs, err := m.syncConfigRepo.GetEnabledConfigsWithAccounts()
	if err != nil {
		m.logger.Error("Failed to query enabled configs from database: %v", err)
		return fmt.Errorf("failed to get enabled configs: %w", err)
	}

	m.logger.Debug("Found %d enabled sync configurations in database", len(configs))

	successCount := 0
	for _, config := range configs {
		m.logger.Debug("Processing config for account %d: email=%s, interval=%ds, enabled=%v",
			config.AccountID, config.Account.EmailAddress, config.SyncInterval, config.EnableAutoSync)

		if err := m.startAccountSyncer(&config); err != nil {
			m.logger.Error("Failed to start syncer for account %d (%s): %v",
				config.AccountID, config.Account.EmailAddress, err)
			// 继续处理其他账户
		} else {
			successCount++
			m.logger.Debug("Successfully started syncer for account %d (%s)",
				config.AccountID, config.Account.EmailAddress)
		}
	}

	m.logger.Info("Completed loading configs: %d/%d syncers started successfully", successCount, len(configs))
	return nil
}

// handleConfigChanges 处理配置变更
func (m *PerAccountSyncManager) handleConfigChanges() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return

		case event := <-m.configMonitor.changes:
			m.processConfigChange(event)
		}
	}
}

// processConfigChange 处理单个配置变更事件
func (m *PerAccountSyncManager) processConfigChange(event ConfigChangeEvent) {
	m.logger.Info("Processing config change: %s for account %d", event.Type, event.AccountID)

	switch event.Type {
	case ConfigAdded, ConfigEnabled:
		if event.NewConfig != nil {
			m.startAccountSyncer(event.NewConfig)
		}

	case ConfigDeleted, ConfigDisabled:
		m.stopAccountSyncer(event.AccountID)

	case ConfigUpdated:
		if event.NewConfig != nil {
			m.updateAccountSyncer(event.AccountID, event.NewConfig)
		}
	}
}

// startAccountSyncer 启动账户同步器
func (m *PerAccountSyncManager) startAccountSyncer(config *models.EmailAccountSyncConfig) error {
	m.logger.Debug("Starting AccountSyncer for account %d", config.AccountID)

	if !config.EnableAutoSync {
		m.logger.Debug("Auto-sync disabled for account %d, skipping", config.AccountID)
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if _, exists := m.accountSyncers[config.AccountID]; exists {
		m.logger.Debug("AccountSyncer already exists for account %d", config.AccountID)
		return nil
	}

	// 创建新的AccountSyncer
	ctx, cancel := context.WithCancel(m.ctx)
	syncer := &AccountSyncer{
		AccountID: config.AccountID,
		Config:    *config,
		ctx:       ctx,
		cancel:    cancel,
		manager:   m,
		logger:    utils.NewLogger(fmt.Sprintf("AccountSyncer-%d", config.AccountID)),
	}

	// 如果有上次同步时间，使用它；否则设置为较早时间触发立即同步
	if config.LastSyncTime != nil {
		syncer.LastSyncTime = *config.LastSyncTime
		m.logger.Debug("Using existing LastSyncTime for account %d: %v", config.AccountID, *config.LastSyncTime)
	} else {
		syncer.LastSyncTime = time.Now().Add(-24 * time.Hour)
		m.logger.Debug("No LastSyncTime for account %d, setting to 24 hours ago for immediate sync", config.AccountID)
	}

	m.accountSyncers[config.AccountID] = syncer

	// 启动goroutine
	m.wg.Add(1)
	go syncer.Run()

	atomic.AddInt64(&m.stats.TotalSyncers, 1)
	atomic.AddInt64(&m.stats.ActiveSyncers, 1)

	m.logger.Debug("Started AccountSyncer for account %d (email: %s, interval: %ds)",
		config.AccountID, config.Account.EmailAddress, config.SyncInterval)

	return nil
}

// stopAccountSyncer 停止账户同步器
func (m *PerAccountSyncManager) stopAccountSyncer(accountID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	syncer, exists := m.accountSyncers[accountID]
	if !exists {
		return
	}

	// 停止同步器
	syncer.Stop()
	delete(m.accountSyncers, accountID)

	atomic.AddInt64(&m.stats.ActiveSyncers, -1)

	m.logger.Info("Stopped AccountSyncer for account %d", accountID)
}

// updateAccountSyncer 更新账户同步器
func (m *PerAccountSyncManager) updateAccountSyncer(accountID uint, newConfig *models.EmailAccountSyncConfig) {
	m.mu.RLock()
	syncer, exists := m.accountSyncers[accountID]
	m.mu.RUnlock()

	if !exists {
		// 如果不存在但配置启用了，创建新的
		if newConfig.EnableAutoSync {
			m.startAccountSyncer(newConfig)
		}
		return
	}

	// 更新现有同步器的配置
	syncer.UpdateConfig(*newConfig)
}

// updateStatsRoutine 更新统计信息
func (m *PerAccountSyncManager) updateStatsRoutine() {
	defer m.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return

		case <-ticker.C:
			m.updateStats()
		}
	}
}

// updateStats 更新统计信息
func (m *PerAccountSyncManager) updateStats() {
	atomic.StoreInt64(&m.stats.CurrentConcurrent, int64(len(m.semaphore)))

	// 计算总同步次数和错误次数
	var totalSyncs, totalErrors int64
	m.mu.RLock()
	for _, syncer := range m.accountSyncers {
		syncer.mu.RLock()
		totalSyncs += atomic.LoadInt64(&syncer.syncCount)
		totalErrors += atomic.LoadInt64(&syncer.errorCount)
		syncer.mu.RUnlock()
	}
	m.mu.RUnlock()

	atomic.StoreInt64(&m.stats.TotalSyncs, totalSyncs)
	atomic.StoreInt64(&m.stats.TotalErrors, totalErrors)
}

// cleanupRoutine 清理例程
func (m *PerAccountSyncManager) cleanupRoutine() {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return

		case <-ticker.C:
			m.cleanupInactiveSyncers()
		}
	}
}

// cleanupInactiveSyncers 清理不活跃的同步器
func (m *PerAccountSyncManager) cleanupInactiveSyncers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for accountID, syncer := range m.accountSyncers {
		syncer.mu.RLock()
		lastSync := syncer.LastSyncTime
		isRunning := syncer.isRunning
		syncer.mu.RUnlock()

		// 如果同步器超过1小时没有活动且不在运行，检查配置是否仍然有效
		if !isRunning && now.Sub(lastSync) > time.Hour {
			// 检查数据库中的配置是否仍然启用
			config, err := m.syncConfigRepo.GetByAccountID(accountID)
			if err != nil || config == nil || !config.EnableAutoSync {
				m.logger.Info("Cleaning up inactive syncer for account %d", accountID)
				syncer.Stop()
				delete(m.accountSyncers, accountID)
				atomic.AddInt64(&m.stats.ActiveSyncers, -1)
			}
		}
	}
}

// GetStats 获取统计信息
func (m *PerAccountSyncManager) GetStats() PerAccountSyncStats {
	m.updateStats() // 强制更新一次
	return PerAccountSyncStats{
		ActiveSyncers:     atomic.LoadInt64(&m.stats.ActiveSyncers),
		TotalSyncers:      atomic.LoadInt64(&m.stats.TotalSyncers),
		TotalSyncs:        atomic.LoadInt64(&m.stats.TotalSyncs),
		TotalErrors:       atomic.LoadInt64(&m.stats.TotalErrors),
		ConcurrentLimit:   m.stats.ConcurrentLimit,
		CurrentConcurrent: atomic.LoadInt64(&m.stats.CurrentConcurrent),
		StartTime:         m.stats.StartTime,
	}
}

// SyncNow 立即同步指定账户
func (m *PerAccountSyncManager) SyncNow(accountID uint) (*SyncResult, error) {
	m.mu.RLock()
	syncer, exists := m.accountSyncers[accountID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no active syncer for account %d", accountID)
	}

	return syncer.SyncNow()
}

// GetAccountSyncerStatus 获取账户同步器状态
func (m *PerAccountSyncManager) GetAccountSyncerStatus(accountID uint) (*AccountSyncerStatus, error) {
	m.mu.RLock()
	syncer, exists := m.accountSyncers[accountID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no active syncer for account %d", accountID)
	}

	return syncer.GetStatus(), nil
}

// GetAllAccountSyncerStatuses 获取所有账户同步器状态
func (m *PerAccountSyncManager) GetAllAccountSyncerStatuses() []AccountSyncerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]AccountSyncerStatus, 0, len(m.accountSyncers))
	for _, syncer := range m.accountSyncers {
		statuses = append(statuses, *syncer.GetStatus())
	}

	return statuses
}

// AccountSyncerStatus 账户同步器状态
type AccountSyncerStatus struct {
	AccountID     uint      `json:"account_id"`
	AccountEmail  string    `json:"account_email"`
	IsRunning     bool      `json:"is_running"`
	LastSyncTime  time.Time `json:"last_sync_time"`
	NextSyncTime  time.Time `json:"next_sync_time"`
	SyncInterval  int       `json:"sync_interval"`
	SyncCount     int64     `json:"sync_count"`
	ErrorCount    int64     `json:"error_count"`
	LastError     string    `json:"last_error,omitempty"`
	LastErrorTime time.Time `json:"last_error_time,omitempty"`
}

// ============= AccountSyncer 实现 =============

// Run 运行账户同步器
func (as *AccountSyncer) Run() {
	defer as.manager.wg.Done()

	as.mu.Lock()
	as.isRunning = true
	as.mu.Unlock()

	defer func() {
		as.mu.Lock()
		as.isRunning = false
		as.mu.Unlock()
		as.logger.Info("AccountSyncer stopped")
	}()

	as.logger.Debug("AccountSyncer started, calculating first sync time")

	// 计算首次同步时间
	nextSyncTime := as.calculateNextSyncTime()
	as.resetTimer(nextSyncTime)
	as.logger.Debug("First sync scheduled for %v", nextSyncTime)

	for {
		select {
		case <-as.ctx.Done():
			as.logger.Info("Received shutdown signal")
			return

		case <-as.timer.C:
			as.logger.Debug("Timer triggered, starting sync")
			// 执行同步
			as.performSync()

			// 重新计算下次同步时间
			nextSyncTime := as.calculateNextSyncTime()
			as.resetTimer(nextSyncTime)
			as.logger.Debug("Next sync scheduled for %v", nextSyncTime)
		}
	}
}

// performSync 执行同步
func (as *AccountSyncer) performSync() {
	as.logger.Debug("Preparing to perform sync")
	start := time.Now()

	// 获取并发许可
	select {
	case as.manager.semaphore <- struct{}{}:
		as.logger.Debug("Semaphore acquired")
		defer func() {
			<-as.manager.semaphore
			as.logger.Debug("Semaphore released")
		}()
	case <-time.After(5 * time.Second):
		// 并发许可获取超时，跳过本次同步
		as.logger.Warn("Failed to acquire semaphore for sync, skipping this cycle")
		return
	}

	atomic.AddInt64(&as.syncCount, 1)
	atomic.AddInt64(&as.manager.stats.TotalSyncs, 1)

	as.logger.Debug("Starting sync cycle")

	// 执行实际同步
	err := as.doSync(start)

	as.mu.Lock()
	as.LastSyncTime = time.Now()
	if err != nil {
		atomic.AddInt64(&as.errorCount, 1)
		atomic.AddInt64(&as.manager.stats.TotalErrors, 1)
		as.lastError = err
		as.lastErrorTime = time.Now()
		as.logger.Error("Sync cycle failed: %v", err)

		// 智能错误处理：分析错误类型并决定是否自动禁用
		as.handleSyncError(err)
	} else {
		as.lastError = nil
		// 同步成功，重置错误计数
		as.resetErrorStatus()
		// 日志已在doSync中输出
	}
	as.mu.Unlock()
}

// doSync 执行实际的邮件同步
func (as *AccountSyncer) doSync(startTime time.Time) error {
	as.logger.Debug("Executing doSync")
	// 创建超时上下文（暂时不使用，但保留用于后续扩展）
	_, cancel := context.WithTimeout(as.ctx, 60*time.Second)
	defer cancel()

	// 获取账户信息
	as.logger.Debug("Getting account details")
	account, err := as.getAccount()
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	as.logger.Debug("Account details obtained for: %s", account.EmailAddress)

	// 计算同步时间窗口
	var startDate *time.Time
	endDate := time.Now()

	if as.Config.LastSyncEndTime != nil {
		// 使用上次同步结束时间减5分钟作为开始时间
		bufferTime := as.Config.LastSyncEndTime.Add(-5 * time.Minute)
		startDate = &bufferTime
		as.logger.Debug("Sync window started from LastSyncEndTime with buffer: %v", startDate)
	} else if as.LastSyncTime.After(time.Time{}) {
		// 使用上次同步时间减5分钟
		bufferTime := as.LastSyncTime.Add(-5 * time.Minute)
		startDate = &bufferTime
		as.logger.Debug("Sync window started from LastSyncTime with buffer: %v", startDate)
	} else {
		// 首次同步，获取最近24小时的邮件
		bufferTime := time.Now().Add(-24 * time.Hour)
		startDate = &bufferTime
		as.logger.Debug("First sync, window started from 24 hours ago: %v", startDate)
	}
	as.logger.Debug("Sync window ends now: %v", endDate)

	// 创建获取选项
	options := FetchEmailsOptions{
		Folders:         as.Config.SyncFolders,
		StartDate:       startDate,
		EndDate:         &endDate,
		FetchFromServer: true,
		IncludeBody:     true,
	}
	as.logger.Debug("Fetching emails with options: Folders=%v, StartDate=%v, EndDate=%v", options.Folders, options.StartDate, options.EndDate)

	// 获取邮件
	emails, err := as.manager.fetcherService.FetchEmailsFromMultipleMailboxes(*account, options)
	if err != nil {
		as.logger.Error("Failed to fetch emails: %v", err)
		return fmt.Errorf("failed to fetch emails: %w", err)
	}
	as.logger.Debug("Fetched %d emails from server", len(emails))

	// 处理邮件
	newEmailCount, err := as.processEmails(emails)
	if err != nil {
		as.logger.Error("Failed to process emails: %v", err)
		return fmt.Errorf("failed to process emails: %w", err)
	}
	as.logger.Debug("Processed %d new emails", newEmailCount)

	// 更新同步配置
	if err := as.updateSyncConfig(endDate, newEmailCount > 0); err != nil {
		as.logger.Warn("Failed to update sync config: %v", err)
	}

	// 如果有新邮件，发送通知
	if newEmailCount > 0 && as.manager.notificationService != nil {
		as.notifyNewEmails(newEmailCount, account.EmailAddress)
	}

	// 输出合并后的同步完成日志
	historyId := "unknown"
	if as.Config.LastHistoryID != "" {
		historyId = as.Config.LastHistoryID
	}
	duration := time.Since(startTime)
	as.logger.Info("email: %s, historyId: %s, newEmails: %d, time: %v",
		account.EmailAddress, historyId, newEmailCount, duration)

	return nil
}

// getAccount 获取账户信息
func (as *AccountSyncer) getAccount() (*models.EmailAccount, error) {
	// 通过邮件仓库获取账户信息
	account, err := as.manager.syncConfigRepo.GetByAccountIDWithAccount(as.AccountID)
	if err != nil {
		return nil, err
	}
	return &account.Account, nil
}

// processEmails 处理邮件
func (as *AccountSyncer) processEmails(emails []models.Email) (int, error) {
	newEmailCount := 0

	for _, email := range emails {
		// 检查邮件是否已存在
		exists, err := as.manager.emailRepo.CheckDuplicate(email.MessageID, email.AccountID)
		if err != nil {
			as.manager.logger.Error("Error checking duplicate for %s: %v", email.MessageID, err)
			continue
		}

		if exists {
			continue
		}

		// 保存新邮件
		if err := as.manager.emailRepo.Create(&email); err != nil {
			as.manager.logger.Error("Failed to save email %s: %v", email.MessageID, err)
			continue
		}

		newEmailCount++
	}

	return newEmailCount, nil
}

// updateSyncConfig 更新同步配置
func (as *AccountSyncer) updateSyncConfig(endTime time.Time, hasNewEmails bool) error {
	// CRITICAL FIX: 重新获取config确保拿到最新的Gmail History ID
	config, err := as.manager.syncConfigRepo.GetByAccountID(as.AccountID)
	if err != nil {
		return err
	}

	now := time.Now()
	// 只更新同步时间和状态，不要覆盖LastHistoryID
	config.LastSyncTime = &now
	config.LastSyncEndTime = &endTime
	config.SyncStatus = models.SyncStatusIdle
	// 注意：故意不修改config.LastHistoryID，保持Gmail API的更新

	as.logger.Debug("Updating sync config - preserving History ID: %s", config.LastHistoryID)
	return as.manager.syncConfigRepo.CreateOrUpdate(config)
}

// notifyNewEmails 发送新邮件通知
func (as *AccountSyncer) notifyNewEmails(count int, emailAddress string) {
	notification := EmailNotification{
		Type:         "new_email",
		AccountID:    as.AccountID,
		AccountEmail: emailAddress,
		EmailCount:   count,
		Timestamp:    time.Now(),
	}

	as.manager.notificationService.BroadcastNotification(notification)
}

// calculateNextSyncTime 计算下次同步时间
func (as *AccountSyncer) calculateNextSyncTime() time.Time {
	as.mu.RLock()
	lastSync := as.LastSyncTime
	intervalSeconds := as.Config.SyncInterval
	as.mu.RUnlock()

	if intervalSeconds <= 0 {
		as.logger.Warn("Sync interval is %d, defaulting to 300 seconds", intervalSeconds)
		intervalSeconds = 300
	}
	interval := time.Duration(intervalSeconds) * time.Second

	// 添加随机抖动，避免雷群效应 (10% of interval)
	jitter := time.Duration(rand.Intn(int(interval / 10)))
	if jitter <= 0 {
		jitter = 1 * time.Second
	}

	nextSync := lastSync.Add(interval + jitter)
	as.logger.Debug("Calculated next sync: LastSync=%v, Interval=%v, Jitter=%v, NextSync=%v", lastSync, interval, jitter, nextSync)

	// 如果已经过期，稍后同步 (1-10秒内)
	if nextSync.Before(time.Now()) {
		immediateNext := time.Now().Add(time.Duration(rand.Intn(10)+1) * time.Second)
		as.logger.Debug("Scheduled next sync was in the past, rescheduling for immediate execution at %v", immediateNext)
		return immediateNext
	}

	return nextSync
}

// resetTimer 重置定时器
func (as *AccountSyncer) resetTimer(nextTime time.Time) {
	as.timerMu.Lock()
	defer as.timerMu.Unlock()

	if as.timer != nil {
		as.timer.Stop()
	}

	duration := time.Until(nextTime)
	if duration < 0 {
		as.logger.Warn("Calculated sync duration is negative (%v), defaulting to 1 second", duration)
		duration = 1 * time.Second
	}

	as.logger.Debug("Resetting sync timer to trigger in %v (at %v)", duration, nextTime)
	as.timer = time.NewTimer(duration)
}

// UpdateConfig 更新配置
func (as *AccountSyncer) UpdateConfig(newConfig models.EmailAccountSyncConfig) {
	as.mu.Lock()
	defer as.mu.Unlock()

	oldInterval := as.Config.SyncInterval
	as.Config = newConfig

	// 如果同步间隔改变，重置定时器
	if oldInterval != newConfig.SyncInterval {
		nextSync := as.calculateNextSyncTime()
		as.resetTimer(nextSync)

		as.manager.logger.Info("Updated sync interval for account %d from %ds to %ds",
			as.AccountID, oldInterval, newConfig.SyncInterval)
	}
}

// Stop 停止同步器
func (as *AccountSyncer) Stop() {
	as.cancel()

	as.timerMu.Lock()
	if as.timer != nil {
		as.timer.Stop()
	}
	as.timerMu.Unlock()
}

// SyncNow 立即同步
func (as *AccountSyncer) SyncNow() (*SyncResult, error) {
	start := time.Now()

	err := as.doSync(start)

	result := &SyncResult{
		EmailsSynced: 0, // 这里应该从doSync返回实际数量
		Duration:     time.Since(start),
		Error:        err,
	}

	return result, nil
}

// GetStatus 获取状态
func (as *AccountSyncer) GetStatus() *AccountSyncerStatus {
	as.mu.RLock()
	defer as.mu.RUnlock()

	var lastErrorStr string
	if as.lastError != nil {
		lastErrorStr = as.lastError.Error()
	}

	nextSyncTime := as.calculateNextSyncTime()

	return &AccountSyncerStatus{
		AccountID:     as.AccountID,
		AccountEmail:  as.Config.Account.EmailAddress,
		IsRunning:     as.isRunning,
		LastSyncTime:  as.LastSyncTime,
		NextSyncTime:  nextSyncTime,
		SyncInterval:  as.Config.SyncInterval,
		SyncCount:     atomic.LoadInt64(&as.syncCount),
		ErrorCount:    atomic.LoadInt64(&as.errorCount),
		LastError:     lastErrorStr,
		LastErrorTime: as.lastErrorTime,
	}
}

// handleSyncError 处理同步错误，智能分析并决定是否自动禁用
func (as *AccountSyncer) handleSyncError(err error) {
	errorMsg := err.Error()

	// 分析错误类型
	errorStatus := as.analyzeErrorType(errorMsg)

	// 更新同步配置的错误计数
	config, getErr := as.manager.syncConfigRepo.GetByAccountID(as.AccountID)
	if getErr != nil {
		as.manager.logger.Error("Failed to get sync config for error handling: %v", getErr)
		return
	}

	// 增加连续错误计数
	config.ConsecutiveErrors++
	now := time.Now()
	config.LastErrorTime = &now

	// 根据错误类型决定是否自动禁用
	shouldDisable, reason := as.shouldAutoDisable(errorStatus, config.ConsecutiveErrors)

	if shouldDisable {
		// 自动禁用同步
		config.EnableAutoSync = false
		config.AutoDisabled = true
		config.DisableReason = reason

		as.manager.logger.Warn("Auto-disabling sync for account %d: %s (consecutive errors: %d)",
			as.AccountID, reason, config.ConsecutiveErrors)

		// 更新账户错误状态
		as.updateAccountErrorStatus(errorStatus, errorMsg)

		// 发送禁用通知
		as.sendDisableNotification(reason)

		// 停止当前AccountSyncer
		go func() {
			time.Sleep(1 * time.Second) // 给日志记录时间
			as.Stop()
		}()
	}

	// 更新同步配置
	if updateErr := as.manager.syncConfigRepo.CreateOrUpdate(config); updateErr != nil {
		as.manager.logger.Error("Failed to update sync config after error: %v", updateErr)
	}
}

// analyzeErrorType 分析错误类型
func (as *AccountSyncer) analyzeErrorType(errorMsg string) models.AccountErrorStatus {
	errorMsg = strings.ToLower(errorMsg)

	// OAuth2认证相关错误
	if strings.Contains(errorMsg, "401") ||
		strings.Contains(errorMsg, "invalid credentials") ||
		strings.Contains(errorMsg, "unauthorized") {
		if strings.Contains(errorMsg, "token") {
			return models.ErrorStatusOAuthExpired
		}
		return models.ErrorStatusAuthRevoked
	}

	// API配额或权限问题
	if strings.Contains(errorMsg, "403") ||
		strings.Contains(errorMsg, "quota") ||
		strings.Contains(errorMsg, "rate limit") {
		return models.ErrorStatusQuotaExceeded
	}

	// API服务禁用
	if strings.Contains(errorMsg, "api disabled") ||
		strings.Contains(errorMsg, "service disabled") {
		return models.ErrorStatusAPIDisabled
	}

	// 网络相关错误
	if strings.Contains(errorMsg, "timeout") ||
		strings.Contains(errorMsg, "connection") ||
		strings.Contains(errorMsg, "network") {
		return models.ErrorStatusNetworkError
	}

	// 服务器错误
	if strings.Contains(errorMsg, "500") ||
		strings.Contains(errorMsg, "502") ||
		strings.Contains(errorMsg, "503") {
		return models.ErrorStatusServerError
	}

	return models.ErrorStatusServerError // 默认为服务器错误
}

// shouldAutoDisable 判断是否应该自动禁用
func (as *AccountSyncer) shouldAutoDisable(errorStatus models.AccountErrorStatus, consecutiveErrors int) (bool, string) {
	switch errorStatus {
	case models.ErrorStatusOAuthExpired:
		// OAuth过期：3次错误后禁用
		if consecutiveErrors >= 3 {
			return true, "OAuth2 Token已过期，需要重新授权"
		}

	case models.ErrorStatusAuthRevoked:
		// 授权撤销：立即禁用
		return true, "账户授权已被撤销，需要重新授权"

	case models.ErrorStatusAPIDisabled:
		// API禁用：立即禁用
		return true, "Gmail API服务已被禁用，请检查配置"

	case models.ErrorStatusQuotaExceeded:
		// 配额超限：5次错误后禁用
		if consecutiveErrors >= 5 {
			return true, "API配额已超限，请检查使用情况"
		}

	case models.ErrorStatusNetworkError:
		// 网络错误：10次错误后禁用
		if consecutiveErrors >= 10 {
			return true, "网络连接持续失败，请检查网络状况"
		}

	case models.ErrorStatusServerError:
		// 服务器错误：15次错误后禁用
		if consecutiveErrors >= 15 {
			return true, "邮件服务器持续异常，请联系服务提供商"
		}
	}

	return false, ""
}

// updateAccountErrorStatus 更新账户错误状态
func (as *AccountSyncer) updateAccountErrorStatus(errorStatus models.AccountErrorStatus, errorMsg string) {
	// 获取账户信息
	account, err := as.manager.emailAccountRepo.GetByID(as.AccountID)
	if err != nil {
		as.manager.logger.Error("Failed to get account for error status update: %v", err)
		return
	}

	// 更新错误状态
	now := time.Now()
	account.ErrorStatus = string(errorStatus)
	account.ErrorMessage = errorMsg
	account.ErrorTimestamp = &now
	account.ErrorCount++
	account.AutoDisabledAt = &now

	// 保存到数据库
	if err := as.manager.emailAccountRepo.Update(account); err != nil {
		as.manager.logger.Error("Failed to update account error status: %v", err)
	}
}

// resetErrorStatus 重置错误状态（同步成功时调用）
func (as *AccountSyncer) resetErrorStatus() {
	// 获取同步配置
	config, err := as.manager.syncConfigRepo.GetByAccountID(as.AccountID)
	if err != nil {
		return
	}

	// 如果之前有连续错误，现在重置
	if config.ConsecutiveErrors > 0 {
		config.ConsecutiveErrors = 0
		config.LastErrorTime = nil

		if updateErr := as.manager.syncConfigRepo.CreateOrUpdate(config); updateErr != nil {
			as.manager.logger.Error("Failed to reset error status: %v", updateErr)
		}

		// 如果账户之前有错误状态，也重置
		as.resetAccountErrorStatus()
	}
}

// resetAccountErrorStatus 重置账户错误状态
func (as *AccountSyncer) resetAccountErrorStatus() {
	account, err := as.manager.emailAccountRepo.GetByID(as.AccountID)
	if err != nil {
		return
	}

	// 如果账户状态异常，重置为正常
	if account.ErrorStatus != string(models.ErrorStatusNormal) {
		account.ErrorStatus = string(models.ErrorStatusNormal)
		account.ErrorMessage = ""
		account.ErrorTimestamp = nil
		// 不重置ErrorCount，保留历史统计

		if err := as.manager.emailAccountRepo.Update(account); err != nil {
			as.manager.logger.Error("Failed to reset account error status: %v", err)
		} else {
			as.manager.logger.Info("Account %d error status reset to normal", as.AccountID)
		}
	}
}

// sendDisableNotification 发送禁用通知
func (as *AccountSyncer) sendDisableNotification(reason string) {
	if as.manager.notificationService == nil {
		return
	}

	account, err := as.getAccount()
	if err != nil {
		return
	}

	notification := EmailNotification{
		Type:         "account_disabled",
		AccountID:    as.AccountID,
		AccountEmail: account.EmailAddress,
		EmailCount:   0,
		Subject:      "同步已自动禁用",
		From:         reason,
		Timestamp:    time.Now(),
	}

	as.manager.notificationService.BroadcastNotification(notification)
}

// UpdateSubscription 更新账户同步订阅配置（实现SyncManager接口）
func (m *PerAccountSyncManager) UpdateSubscription(accountID uint, config *models.EmailAccountSyncConfig) error {
	m.logger.Info("Updating subscription for account %d", accountID)

	if config == nil {
		// 移除账户同步器
		m.stopAccountSyncer(accountID)
		return nil
	}

	// 更新或创建账户同步器
	m.mu.RLock()
	syncer, exists := m.accountSyncers[accountID]
	m.mu.RUnlock()

	if exists {
		// 更新现有同步器
		syncer.UpdateConfig(*config)
	} else if config.EnableAutoSync {
		// 创建新的同步器
		m.startAccountSyncer(config)
	}

	return nil
}
