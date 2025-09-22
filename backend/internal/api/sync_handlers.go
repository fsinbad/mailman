package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"mailman/internal/models"
	"mailman/internal/repository"
	"mailman/internal/services"
	"mailman/internal/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// SyncHandlers handles sync configuration related requests
type SyncHandlers struct {
	syncConfigRepo *repository.SyncConfigRepository
	syncManager    services.SyncManager // 使用接口类型代替具体实现
	mailboxRepo    *repository.MailboxRepository
	fetcher        *services.FetcherService
	accountRepo    *repository.EmailAccountRepository
	logger         *utils.Logger
	activityLogger *services.ActivityLogger
	db             *gorm.DB // Add database connection for transactions

	// 账户级别的锁，防止同步配置并发冲突
	accountLocks map[uint]*sync.Mutex
	locksMutex   sync.RWMutex
}

// NewSyncHandlers creates a new sync handlers instance
func NewSyncHandlers(
	syncConfigRepo *repository.SyncConfigRepository,
	syncManager services.SyncManager, // 使用接口类型代替具体实现
	mailboxRepo *repository.MailboxRepository,
	fetcher *services.FetcherService,
	accountRepo *repository.EmailAccountRepository,
	db *gorm.DB, // Add database connection parameter
) *SyncHandlers {
	return &SyncHandlers{
		syncConfigRepo: syncConfigRepo,
		syncManager:    syncManager,
		mailboxRepo:    mailboxRepo,
		fetcher:        fetcher,
		accountRepo:    accountRepo,
		logger:         utils.NewLogger("SyncHandlers"),
		activityLogger: services.GetActivityLogger(),
		db:             db,
		accountLocks:   make(map[uint]*sync.Mutex),
		locksMutex:     sync.RWMutex{},
	}
}

// getAccountLock 获取指定账户的互斥锁，如果不存在则创建
func (h *SyncHandlers) getAccountLock(accountID uint) *sync.Mutex {
	h.locksMutex.RLock()
	if lock, exists := h.accountLocks[accountID]; exists {
		h.locksMutex.RUnlock()
		return lock
	}
	h.locksMutex.RUnlock()

	// 需要写锁来创建新锁
	h.locksMutex.Lock()
	defer h.locksMutex.Unlock()

	// 双重检查，防止并发创建
	if lock, exists := h.accountLocks[accountID]; exists {
		return lock
	}

	// 创建新锁
	lock := &sync.Mutex{}
	h.accountLocks[accountID] = lock
	return lock
}

// GetAccountMailboxes retrieves all mailboxes for an account
func (h *SyncHandlers) GetAccountMailboxes(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	accountIDStr := vars["id"]

	h.logger.Debug("GetAccountMailboxes called for account: %s", accountIDStr)

	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		h.logger.Error("Invalid account ID: %s, error: %v", accountIDStr, err)
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	// First, try to fetch mailboxes from IMAP server
	// Get the account details first
	h.logger.Debug("Fetching account details for ID: %d", accountID)
	account, err := h.accountRepo.GetByID(uint(accountID))
	if err != nil {
		h.logger.Error("Failed to get account %d: %v", accountID, err)
		http.Error(w, "Failed to get account", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Fetching mailboxes from IMAP server for account: %s", account.EmailAddress)
	fetchedMailboxes, err := h.fetcher.GetMailboxes(*account)
	if err == nil && len(fetchedMailboxes) > 0 {
		h.logger.Info("Fetched %d mailboxes from IMAP server", len(fetchedMailboxes))
		// Sync the fetched mailboxes to database
		if syncErr := h.mailboxRepo.SyncMailboxes(uint(accountID), fetchedMailboxes); syncErr != nil {
			h.logger.Warn("Failed to sync mailboxes to database: %v", syncErr)
		} else {
			h.logger.Debug("Successfully synced mailboxes to database")
		}
	} else if err != nil {
		h.logger.Warn("Failed to fetch mailboxes from IMAP: %v", err)
	}

	// Get mailboxes from database (includes both active and deleted)
	h.logger.Debug("Retrieving mailboxes from database")
	mailboxes, err := h.mailboxRepo.GetByAccountID(uint(accountID))
	if err != nil {
		h.logger.Error("Failed to get mailboxes from database: %v", err)
		http.Error(w, "Failed to get mailboxes", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Found %d mailboxes in database", len(mailboxes))

	// Format response with mailbox status
	type MailboxResponse struct {
		ID        uint     `json:"id"`
		Name      string   `json:"name"`
		Delimiter string   `json:"delimiter"`
		Flags     []string `json:"flags"`
		IsDeleted bool     `json:"is_deleted"`
	}

	var response []MailboxResponse
	for _, mailbox := range mailboxes {
		isDeleted := false
		for _, flag := range mailbox.Flags {
			if flag == "\\Deleted" {
				isDeleted = true
				break
			}
		}
		response = append(response, MailboxResponse{
			ID:        mailbox.ID,
			Name:      mailbox.Name,
			Delimiter: mailbox.Delimiter,
			Flags:     mailbox.Flags,
			IsDeleted: isDeleted,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}

	h.logger.Info("GetAccountMailboxes completed in %v", time.Since(start))
}

// GetAccountSyncConfig retrieves sync configuration for an account
func (h *SyncHandlers) GetAccountSyncConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	accountIDStr := vars["id"]

	h.logger.Debug("GetAccountSyncConfig called for account: %s", accountIDStr)

	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		h.logger.Error("Invalid account ID: %s, error: %v", accountIDStr, err)
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	config, err := h.syncConfigRepo.GetByAccountID(uint(accountID))
	if err != nil {
		// If config not found, create default config
		if err.Error() == "record not found" {
			h.logger.Info("No sync config found for account %d, creating default", accountID)
			if createErr := h.syncConfigRepo.CreateDefaultConfigForAccount(uint(accountID)); createErr != nil {
				h.logger.Error("Failed to create default config: %v", createErr)
				http.Error(w, "Failed to create default config", http.StatusInternalServerError)
				return
			}

			// Retrieve the newly created config
			config, err = h.syncConfigRepo.GetByAccountID(uint(accountID))
			if err != nil {
				h.logger.Error("Failed to retrieve newly created config: %v", err)
				http.Error(w, "Failed to retrieve config", http.StatusInternalServerError)
				return
			}
		} else {
			h.logger.Error("Failed to get config: %v", err)
			http.Error(w, "Failed to get config", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}

	h.logger.Info("GetAccountSyncConfig completed in %v", time.Since(start))
}

// CreateAccountSyncConfig creates a new sync configuration for an account
func (h *SyncHandlers) CreateAccountSyncConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	accountIDStr := vars["id"]

	h.logger.Debug("CreateAccountSyncConfig called for account: %s", accountIDStr)
	h.logger.LogHTTPRequest(r, true)

	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		h.logger.Error("Invalid account ID: %s, error: %v", accountIDStr, err)
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	// 获取账户级别的锁，防止与同步进程冲突
	accountLock := h.getAccountLock(uint(accountID))
	accountLock.Lock()
	defer accountLock.Unlock()

	h.logger.Debug("Acquired lock for account %d", accountID)

	// 关键修复：暂时停止该账户的AccountSyncer，避免长时间同步阻塞API
	h.logger.Debug("Temporarily stopping AccountSyncer for account %d to prevent sync conflicts", accountID)
	h.syncManager.UpdateSubscription(uint(accountID), nil) // 停止同步器

	var req UpdateSyncConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Debug("Request data: EnableAutoSync=%v, SyncInterval=%v, SyncFolders=%v",
		req.EnableAutoSync, req.SyncInterval, req.SyncFolders)

	// Validate sync interval - 允许更小的间隔值
	if req.SyncInterval != nil && *req.SyncInterval < 1 {
		h.logger.Warn("Invalid sync interval: %d", *req.SyncInterval)
		http.Error(w, "Sync interval must be at least 1 second", http.StatusBadRequest)
		return
	}

	// 不再验证文件夹，由系统自动处理

	// Check if config already exists
	_, err = h.syncConfigRepo.GetByAccountID(uint(accountID))
	if err == nil {
		h.logger.Warn("Sync config already exists for account %d", accountID)
		http.Error(w, "Sync config already exists for this account", http.StatusConflict)
		return
	}

	// Create new config
	config := &models.EmailAccountSyncConfig{
		AccountID:      uint(accountID),
		EnableAutoSync: true,
		SyncInterval:   5,                           // 默认5秒间隔
		SyncFolders:    models.StringSlice{"INBOX"}, // 系统默认同步所有重要文件夹
		SyncStatus:     "idle",
	}

	// Apply request values
	if req.EnableAutoSync != nil {
		config.EnableAutoSync = *req.EnableAutoSync
	}
	if req.SyncInterval != nil {
		config.SyncInterval = *req.SyncInterval
	}
	// 不再允许用户指定文件夹，系统自动处理

	h.logger.Info("Creating sync config for account %d: AutoSync=%v, Interval=%d, Folders=%v",
		accountID, config.EnableAutoSync, config.SyncInterval, config.SyncFolders)

	// Save config
	if err := h.syncConfigRepo.CreateOrUpdate(config); err != nil {
		h.logger.ErrorWithStack(err, "Failed to create config")
		http.Error(w, "Failed to create config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}

	h.logger.Info("CreateAccountSyncConfig completed in %v", time.Since(start))
}

// UpdateAccountSyncConfig updates sync configuration for an account
func (h *SyncHandlers) UpdateAccountSyncConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	accountIDStr := vars["id"]

	h.logger.Debug("UpdateAccountSyncConfig called for account: %s", accountIDStr)
	h.logger.LogHTTPRequest(r, true)

	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		h.logger.Error("Invalid account ID: %s, error: %v", accountIDStr, err)
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	// 获取账户级别的锁，防止与同步进程冲突
	accountLock := h.getAccountLock(uint(accountID))
	accountLock.Lock()
	defer accountLock.Unlock()

	h.logger.Debug("Acquired lock for account %d", accountID)

	var req UpdateSyncConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Debug("Update request: EnableAutoSync=%v, SyncInterval=%v, SyncFolders=%v",
		req.EnableAutoSync, req.SyncInterval, req.SyncFolders)

	// Validate sync interval - 允许更小的间隔值
	if req.SyncInterval != nil && *req.SyncInterval < 1 {
		h.logger.Warn("Invalid sync interval: %d", *req.SyncInterval)
		http.Error(w, "Sync interval must be at least 1 second", http.StatusBadRequest)
		return
	}

	// 不再验证文件夹，由系统自动处理

	// Get existing config
	config, err := h.syncConfigRepo.GetByAccountID(uint(accountID))
	if err != nil {
		h.logger.Error("Config not found for account %d: %v", accountID, err)
		http.Error(w, "Config not found", http.StatusNotFound)
		return
	}

	h.logger.Debug("Current config: AutoSync=%v, Interval=%d, Folders=%v",
		config.EnableAutoSync, config.SyncInterval, config.SyncFolders)

	// Update fields
	if req.EnableAutoSync != nil {
		config.EnableAutoSync = *req.EnableAutoSync
	}
	if req.SyncInterval != nil {
		config.SyncInterval = *req.SyncInterval
	}
	// 不再允许用户指定文件夹，系统自动处理

	h.logger.Info("Updating sync config for account %d: AutoSync=%v, Interval=%d, Folders=%v",
		accountID, config.EnableAutoSync, config.SyncInterval, config.SyncFolders)

	// Save updated config
	if err := h.syncConfigRepo.CreateOrUpdate(config); err != nil {
		h.logger.ErrorWithStack(err, "Failed to update config")
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	// Update subscription in sync manager
	if err := h.syncManager.UpdateSubscription(uint(accountID), config); err != nil {
		h.logger.Warn("Failed to update subscription in sync manager: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}

	h.logger.Info("UpdateAccountSyncConfig completed in %v", time.Since(start))
}

// DeleteAccountSyncConfig deletes sync configuration for an account
func (h *SyncHandlers) DeleteAccountSyncConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	accountIDStr := vars["id"]

	h.logger.Debug("DeleteAccountSyncConfig called for account: %s", accountIDStr)

	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		h.logger.Error("Invalid account ID: %s, error: %v", accountIDStr, err)
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	// Get the config first to get its ID
	config, err := h.syncConfigRepo.GetByAccountID(uint(accountID))
	if err != nil {
		h.logger.Error("Config not found for account %d: %v", accountID, err)
		http.Error(w, "Config not found", http.StatusNotFound)
		return
	}

	h.logger.Info("Deleting sync config ID %d for account %d", config.ID, accountID)

	// Delete the config by ID
	if err := h.syncConfigRepo.Delete(config.ID); err != nil {
		h.logger.ErrorWithStack(err, "Failed to delete config")
		http.Error(w, "Failed to delete config", http.StatusInternalServerError)
		return
	}

	// Remove from sync manager (if the method exists)
	// For now, we'll just ignore this as the sync manager will handle missing configs

	w.WriteHeader(http.StatusNoContent)
	h.logger.Info("DeleteAccountSyncConfig completed in %v", time.Since(start))
}

// SyncNow triggers immediate sync for an account
func (h *SyncHandlers) SyncNow(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	accountIDStr := vars["id"]

	h.logger.Debug("SyncNow called for account: %s", accountIDStr)

	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		h.logger.Error("Invalid account ID: %s, error: %v", accountIDStr, err)
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	h.logger.Info("Triggering immediate sync for account %d", accountID)

	syncStart := time.Now()
	result, err := h.syncManager.SyncNow(uint(accountID))
	syncDuration := time.Since(syncStart)

	if err != nil {
		h.logger.ErrorWithStack(err, "Sync failed for account %d", accountID)
		http.Error(w, "Sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("Sync completed for account %d: %d emails synced in %v",
		accountID, result.EmailsSynced, syncDuration)

	// Log activity
	userID := getUserIDFromContext(r)
	if result.EmailsSynced > 0 {
		h.activityLogger.LogSyncActivity(
			models.ActivitySyncCompleted,
			fmt.Sprintf("账户 %d", accountID),
			userID,
			map[string]interface{}{
				"emails_synced": result.EmailsSynced,
				"duration_ms":   syncDuration.Milliseconds(),
			},
		)
	}

	response := SyncNowResponse{
		Success:      true,
		EmailsSynced: result.EmailsSynced,
		Duration:     result.Duration.String(),
	}

	if result.Error != nil {
		response.Success = false
		response.Error = result.Error.Error()
		h.logger.Warn("Sync had errors: %v", result.Error)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}

	h.logger.Info("SyncNow handler completed in %v", time.Since(start))
}

// GetGlobalSyncConfig retrieves global sync configuration
func (h *SyncHandlers) GetGlobalSyncConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.logger.Debug("GetGlobalSyncConfig called")

	config, err := h.syncConfigRepo.GetGlobalConfig()
	if err != nil {
		h.logger.Error("Failed to get global config: %v", err)
		http.Error(w, "Failed to get global config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}

	h.logger.Info("GetGlobalSyncConfig completed in %v", time.Since(start))
}

// UpdateGlobalSyncConfig updates global sync configuration
func (h *SyncHandlers) UpdateGlobalSyncConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.logger.Debug("UpdateGlobalSyncConfig called")
	h.logger.LogHTTPRequest(r, true)

	var req UpdateGlobalSyncConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate sync interval - 允许更小的间隔值
	if req.DefaultSyncInterval != nil && *req.DefaultSyncInterval < 1 {
		h.logger.Warn("Invalid sync interval: %d", *req.DefaultSyncInterval)
		http.Error(w, "Sync interval must be at least 1 second", http.StatusBadRequest)
		return
	}

	// 不再验证文件夹，由系统自动处理

	// Get current global config
	globalConfig, err := h.syncConfigRepo.GetGlobalConfig()
	if err != nil {
		h.logger.Error("Failed to get global config: %v", err)
		http.Error(w, "Failed to get global config", http.StatusInternalServerError)
		return
	}

	// Since GetGlobalConfig returns map[string]interface{}, we need to update it
	if req.DefaultEnableSync != nil {
		globalConfig["default_enable_sync"] = *req.DefaultEnableSync
	}
	if req.DefaultSyncInterval != nil {
		globalConfig["default_sync_interval"] = *req.DefaultSyncInterval
	}
	if req.DefaultSyncFolders != nil {
		globalConfig["default_sync_folders"] = req.DefaultSyncFolders
	}
	if req.MaxSyncWorkers != nil && *req.MaxSyncWorkers > 0 {
		globalConfig["max_sync_workers"] = *req.MaxSyncWorkers
	}
	if req.MaxEmailsPerSync != nil && *req.MaxEmailsPerSync > 0 {
		globalConfig["max_emails_per_sync"] = *req.MaxEmailsPerSync
	}

	h.logger.Info("Updating global sync config: %+v", globalConfig)

	// Save updated config
	if err := h.syncConfigRepo.UpdateGlobalConfig(globalConfig); err != nil {
		h.logger.ErrorWithStack(err, "Failed to update global config")
		http.Error(w, "Failed to update global config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(globalConfig); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}

	h.logger.Info("UpdateGlobalSyncConfig completed in %v", time.Since(start))
}

// GetSyncStatistics retrieves sync statistics for an account
func (h *SyncHandlers) GetSyncStatistics(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("GetSyncStatistics called - not implemented yet")

	// TODO: Implement this properly
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Not implemented yet",
	})
}

// GetAllSyncConfigs retrieves all sync configurations with pagination
func (h *SyncHandlers) GetAllSyncConfigs(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.logger.Debug("GetAllSyncConfigs called")

	// Parse query parameters
	page := 1
	limit := 20
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	h.logger.Debug("Query params: page=%d, limit=%d, search=%s, status=%s", page, limit, search, status)

	// Get total count and configs
	totalCount, configs, err := h.syncConfigRepo.GetAllWithPagination(page, limit, search)
	if err != nil {
		h.logger.Error("Failed to get sync configs: %v", err)
		http.Error(w, "Failed to get sync configs", http.StatusInternalServerError)
		return
	}

	// Filter by status if provided
	if status != "" && status != "all" {
		var filteredConfigs []models.EmailAccountSyncConfig
		for _, config := range configs {
			if config.SyncStatus == status {
				filteredConfigs = append(filteredConfigs, config)
			}
		}
		configs = filteredConfigs
		h.logger.Debug("Filtered configs by status '%s': %d results", status, len(configs))
	}

	// Calculate pagination info
	totalPages := (totalCount + limit - 1) / limit

	response := GetAllSyncConfigsResponse{
		Configs:     configs,
		TotalCount:  totalCount,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}

	h.logger.Info("GetAllSyncConfigs completed in %v: returned %d configs", time.Since(start), len(configs))
}

// CreateTemporarySyncConfig creates a temporary sync configuration for an account
func (h *SyncHandlers) CreateTemporarySyncConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	accountIDStr := vars["id"]

	h.logger.Debug("CreateTemporarySyncConfig called for account: %s", accountIDStr)
	h.logger.LogHTTPRequest(r, true)

	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		h.logger.Error("Invalid account ID: %s, error: %v", accountIDStr, err)
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	var req CreateTemporarySyncConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate sync interval
	if req.SyncInterval < 1 {
		req.SyncInterval = 5 // Default to 5 seconds
	}

	// 不再验证文件夹，直接使用默认的同步所有重要文件夹
	req.SyncFolders = []string{"INBOX"} // 系统默认同步所有重要文件夹

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute)
	if req.DurationMinutes <= 0 {
		expiresAt = time.Now().Add(30 * time.Minute) // Default to 30 minutes
	}

	// Create temporary config
	tempConfig := &models.TemporarySyncConfig{
		AccountID:    uint(accountID),
		SyncInterval: req.SyncInterval,
		SyncFolders:  models.StringSlice(req.SyncFolders),
		ExpiresAt:    expiresAt,
	}

	h.logger.Info("Creating temporary sync config for account %d: Interval=%d, Folders=%v, ExpiresAt=%v",
		accountID, tempConfig.SyncInterval, tempConfig.SyncFolders, tempConfig.ExpiresAt)

	// Save temporary config
	if err := h.syncConfigRepo.CreateTemporaryConfig(tempConfig); err != nil {
		h.logger.ErrorWithStack(err, "Failed to create temporary config")
		http.Error(w, "Failed to create temporary config", http.StatusInternalServerError)
		return
	}

	// Update subscription in sync manager with the effective config
	effectiveConfig, err := h.syncConfigRepo.GetEffectiveSyncConfig(uint(accountID))
	if err == nil && effectiveConfig != nil {
		if err := h.syncManager.UpdateSubscription(uint(accountID), effectiveConfig); err != nil {
			h.logger.Warn("Failed to update subscription in sync manager: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tempConfig); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}

	h.logger.Info("CreateTemporarySyncConfig completed in %v", time.Since(start))
}

// GetEffectiveSyncConfig retrieves the effective sync configuration for an account
func (h *SyncHandlers) GetEffectiveSyncConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	accountIDStr := vars["id"]

	h.logger.Debug("GetEffectiveSyncConfig called for account: %s", accountIDStr)

	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		h.logger.Error("Invalid account ID: %s, error: %v", accountIDStr, err)
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	config, err := h.syncConfigRepo.GetEffectiveSyncConfig(uint(accountID))
	if err != nil {
		h.logger.Error("Failed to get effective config: %v", err)
		http.Error(w, "Failed to get effective config", http.StatusInternalServerError)
		return
	}

	// Add a flag to indicate if this is a temporary config
	response := map[string]interface{}{
		"config":       config,
		"is_temporary": false,
	}

	// Check if there's an active temporary config
	tempConfig, err := h.syncConfigRepo.GetTemporaryConfigByAccountID(uint(accountID))
	if err == nil && tempConfig != nil && !tempConfig.IsExpired() {
		response["is_temporary"] = true
		response["expires_at"] = tempConfig.ExpiresAt
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}

	h.logger.Info("GetEffectiveSyncConfig completed in %v", time.Since(start))
}

// BatchCreateOrUpdateAccountSyncConfig handles batch sync configuration creation/update with performance optimizations
func (h *SyncHandlers) BatchCreateOrUpdateAccountSyncConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.logger.Debug("BatchCreateOrUpdateAccountSyncConfig called")
	h.logger.LogHTTPRequest(r, true)

	// Set request timeout context
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	var req BatchSyncConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Debug("Batch request: AccountIds=%v, EnableAutoSync=%v, SyncInterval=%v, SyncFolders=%v",
		req.AccountIds, req.EnableAutoSync, req.SyncInterval, req.SyncFolders)

	// Validate sync interval
	if req.SyncInterval < 1 {
		h.logger.Warn("Invalid sync interval: %d", req.SyncInterval)
		http.Error(w, "Sync interval must be at least 1 second", http.StatusBadRequest)
		return
	}

	// Validate account IDs
	if len(req.AccountIds) == 0 {
		h.logger.Warn("No account IDs provided")
		http.Error(w, "At least one account ID is required", http.StatusBadRequest)
		return
	}

	// Limit batch size to prevent timeout
	const maxBatchSize = 50
	if len(req.AccountIds) > maxBatchSize {
		h.logger.Warn("Batch size %d exceeds maximum %d", len(req.AccountIds), maxBatchSize)
		http.Error(w, fmt.Sprintf("Batch size cannot exceed %d accounts", maxBatchSize), http.StatusBadRequest)
		return
	}

	h.logger.Info("Processing optimized batch sync config for %d accounts", len(req.AccountIds))

	// Execute batch processing with transaction
	response, err := h.processBatchSyncConfigOptimized(ctx, req)
	if err != nil {
		h.logger.Error("Batch processing failed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Optimized batch sync config completed: %d success, %d errors in %v",
		response.SuccessCount, response.ErrorCount, time.Since(start))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
	}
}

// processBatchSyncConfigOptimized performs optimized batch processing with transaction
func (h *SyncHandlers) processBatchSyncConfigOptimized(ctx context.Context, req BatchSyncConfigRequest) (*BatchSyncConfigResponse, error) {
	// Begin transaction for data consistency
	tx := h.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer tx.Rollback() // Will be no-op if Commit() succeeds

	var response BatchSyncConfigResponse
	var errors []BatchSyncError
	var successfulConfigs []*models.EmailAccountSyncConfig

	// Batch load all accounts to reduce database queries
	h.logger.Debug("Batch loading accounts")
	accountMap, err := h.batchLoadAccounts(tx, req.AccountIds)
	if err != nil {
		return nil, fmt.Errorf("failed to batch load accounts: %w", err)
	}

	// Batch load existing configs
	h.logger.Debug("Batch loading existing sync configs")
	existingConfigMap, err := h.batchLoadConfigs(tx, req.AccountIds)
	if err != nil {
		return nil, fmt.Errorf("failed to batch load configs: %w", err)
	}

	// Process each account with individual locks to prevent conflicts
	for _, accountID := range req.AccountIds {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("batch processing timeout")
		default:
		}

		// 获取账户级别的锁，防止与同步进程冲突
		accountLock := h.getAccountLock(accountID)
		accountLock.Lock()

		// Check if account exists
		account, exists := accountMap[accountID]
		if !exists {
			h.logger.Warn("Account %d not found", accountID)
			errors = append(errors, BatchSyncError{
				AccountID:    accountID,
				EmailAddress: fmt.Sprintf("account_%d", accountID),
				Error:        "Account not found",
			})
			accountLock.Unlock()
			continue
		}

		// Create or update config
		var config *models.EmailAccountSyncConfig
		if existingConfig, hasConfig := existingConfigMap[accountID]; hasConfig {
			// Update existing config
			config = existingConfig
			config.EnableAutoSync = req.EnableAutoSync
			config.SyncInterval = req.SyncInterval
			// Keep existing sync folders or use system default
			if len(config.SyncFolders) == 0 {
				config.SyncFolders = models.StringSlice{"INBOX"}
			}
		} else {
			// Create new config
			config = &models.EmailAccountSyncConfig{
				AccountID:      accountID,
				EnableAutoSync: req.EnableAutoSync,
				SyncInterval:   req.SyncInterval,
				SyncFolders:    models.StringSlice{"INBOX"}, // System default
				SyncStatus:     "idle",
			}
		}

		// Save config within transaction
		if err := tx.Save(config).Error; err != nil {
			h.logger.Error("Failed to save config for account %d: %v", accountID, err)
			errors = append(errors, BatchSyncError{
				AccountID:    accountID,
				EmailAddress: account.EmailAddress,
				Error:        "Failed to save config: " + err.Error(),
			})
			accountLock.Unlock()
			continue
		}

		successfulConfigs = append(successfulConfigs, config)
		response.SuccessCount++
		h.logger.Debug("Successfully processed config for account %d (%s)", accountID, account.EmailAddress)

		// 释放账户锁
		accountLock.Unlock()
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	h.logger.Info("Transaction committed successfully for %d configs", len(successfulConfigs))

	// Update sync manager subscriptions after successful database commit
	// This is done outside transaction to avoid blocking
	go h.updateSyncManagerBatch(successfulConfigs)

	response.ErrorCount = len(errors)
	response.Errors = errors

	return &response, nil
}

// batchLoadAccounts loads all accounts in a single query
func (h *SyncHandlers) batchLoadAccounts(tx *gorm.DB, accountIDs []uint) (map[uint]*models.EmailAccount, error) {
	var accounts []models.EmailAccount
	if err := tx.Where("id IN ?", accountIDs).Find(&accounts).Error; err != nil {
		return nil, err
	}

	accountMap := make(map[uint]*models.EmailAccount)
	for i := range accounts {
		accountMap[accounts[i].ID] = &accounts[i]
	}

	return accountMap, nil
}

// batchLoadConfigs loads all existing sync configs in a single query
func (h *SyncHandlers) batchLoadConfigs(tx *gorm.DB, accountIDs []uint) (map[uint]*models.EmailAccountSyncConfig, error) {
	var configs []models.EmailAccountSyncConfig
	if err := tx.Where("account_id IN ?", accountIDs).Find(&configs).Error; err != nil {
		return nil, err
	}

	configMap := make(map[uint]*models.EmailAccountSyncConfig)
	for i := range configs {
		configMap[configs[i].AccountID] = &configs[i]
	}

	return configMap, nil
}

// updateSyncManagerBatch updates sync manager subscriptions in background
func (h *SyncHandlers) updateSyncManagerBatch(configs []*models.EmailAccountSyncConfig) {
	h.logger.Debug("Starting background sync manager updates for %d configs", len(configs))

	// Process in smaller batches to avoid overwhelming the sync manager
	const batchSize = 10
	for i := 0; i < len(configs); i += batchSize {
		end := i + batchSize
		if end > len(configs) {
			end = len(configs)
		}

		batch := configs[i:end]
		for _, config := range batch {
			if err := h.syncManager.UpdateSubscription(config.AccountID, config); err != nil {
				h.logger.Warn("Failed to update subscription for account %d: %v", config.AccountID, err)
			} else {
				h.logger.Debug("Updated sync manager subscription for account %d", config.AccountID)
			}
		}

		// Small delay between batches to prevent overwhelming
		if end < len(configs) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	h.logger.Info("Completed background sync manager updates for %d configs", len(configs))
}

// Request/Response types

type UpdateSyncConfigRequest struct {
	EnableAutoSync *bool    `json:"enable_auto_sync,omitempty"`
	SyncInterval   *int     `json:"sync_interval,omitempty"`
	SyncFolders    []string `json:"sync_folders,omitempty"` // 保留但不再处理
}

type CreateTemporarySyncConfigRequest struct {
	SyncInterval    int      `json:"sync_interval"`
	SyncFolders     []string `json:"sync_folders,omitempty"` // 可选，系统自动处理
	DurationMinutes int      `json:"duration_minutes"`
}

type UpdateGlobalSyncConfigRequest struct {
	DefaultEnableSync   *bool    `json:"default_enable_sync,omitempty"`
	DefaultSyncInterval *int     `json:"default_sync_interval,omitempty"`
	DefaultSyncFolders  []string `json:"default_sync_folders,omitempty"`
	MaxSyncWorkers      *int     `json:"max_sync_workers,omitempty"`
	MaxEmailsPerSync    *int     `json:"max_emails_per_sync,omitempty"`
}

type SyncNowResponse struct {
	Success      bool   `json:"success"`
	EmailsSynced int    `json:"emails_synced"`
	Duration     string `json:"duration"`
	Error        string `json:"error,omitempty"`
}

type SyncStatisticsResponse struct {
	AccountID         uint                    `json:"account_id"`
	Days              int                     `json:"days"`
	TotalEmailsSynced int                     `json:"total_emails_synced"`
	TotalErrors       int                     `json:"total_errors"`
	AverageDurationMs int                     `json:"average_duration_ms"`
	DailyStats        []models.SyncStatistics `json:"daily_stats"`
}

type GetAllSyncConfigsResponse struct {
	Configs     []models.EmailAccountSyncConfig `json:"configs"`
	TotalCount  int                             `json:"total_count"`
	Page        int                             `json:"page"`
	Limit       int                             `json:"limit"`
	TotalPages  int                             `json:"total_pages"`
	HasNext     bool                            `json:"has_next"`
	HasPrevious bool                            `json:"has_previous"`
}

type BatchSyncConfigRequest struct {
	AccountIds     []uint   `json:"account_ids"`
	EnableAutoSync bool     `json:"enable_auto_sync"`
	SyncInterval   int      `json:"sync_interval"`
	SyncFolders    []string `json:"sync_folders,omitempty"` // Optional, system will handle
}

type BatchSyncConfigResponse struct {
	SuccessCount int              `json:"success_count"`
	ErrorCount   int              `json:"error_count"`
	Errors       []BatchSyncError `json:"errors"`
}

type BatchSyncError struct {
	AccountID    uint   `json:"account_id"`
	EmailAddress string `json:"email_address"`
	Error        string `json:"error"`
}
