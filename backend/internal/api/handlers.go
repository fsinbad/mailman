package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"mailman/internal/database"
	"mailman/internal/models"
	"mailman/internal/repository"
	"mailman/internal/services"
	"mailman/internal/triggerv2/plugins"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// getUserIDFromContext extracts user ID from request context
func getUserIDFromContext(r *http.Request) *uint {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok || user == nil {
		return nil
	}
	return &user.ID
}

// HealthCheck godoc
// @Summary Show the status of server.
// @Description get the status of server.
// @Tags root
// @Accept */*
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/health [get]
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type APIHandler struct {
	Fetcher             *services.FetcherService
	Parser              *services.ParserService
	EmailAccountRepo    *repository.EmailAccountRepository
	MailProviderRepo    *repository.MailProviderRepository
	EmailRepo           *repository.EmailRepository
	IncrementalSyncRepo *repository.IncrementalSyncRepository
	EmailScheduler      *services.EmailFetchScheduler
	activityLogger      *services.ActivityLogger
	pluginManager       plugins.PluginManager

	// 新增的同步管理器
	optimizedSyncManager  *services.OptimizedIncrementalSyncManager
	perAccountSyncManager *services.PerAccountSyncManager
}

func NewAPIHandler(
	fetcher *services.FetcherService,
	parser *services.ParserService,
	emailAccountRepo *repository.EmailAccountRepository,
	mailProviderRepo *repository.MailProviderRepository,
	emailRepo *repository.EmailRepository,
	incrementalSyncRepo *repository.IncrementalSyncRepository,
	emailScheduler *services.EmailFetchScheduler,
	pluginManager plugins.PluginManager,
	optimizedSyncManager *services.OptimizedIncrementalSyncManager,
	perAccountSyncManager *services.PerAccountSyncManager,
) *APIHandler {
	return &APIHandler{
		Fetcher:               fetcher,
		Parser:                parser,
		EmailAccountRepo:      emailAccountRepo,
		MailProviderRepo:      mailProviderRepo,
		EmailRepo:             emailRepo,
		IncrementalSyncRepo:   incrementalSyncRepo,
		EmailScheduler:        emailScheduler,
		activityLogger:        services.GetActivityLogger(),
		pluginManager:         pluginManager,
		optimizedSyncManager:  optimizedSyncManager,
		perAccountSyncManager: perAccountSyncManager,
	}
}

// FetchEmailsHandler godoc
// @Summary Fetch emails with enhanced filtering and smart email matching
// @Description Fetch emails for a given account with advanced filtering capabilities and intelligent email address matching. Supports Gmail aliases (dots, plus signs, googlemail.com), domain mail forwarding, content filtering, flag-based filtering, size filtering, date ranges, and more. When using email_address parameter, the system will automatically handle Gmail aliases and domain mail scenarios.
// @Tags emails
// @Accept json
// @Produce json
// @Param request body FetchEmailsRequest true "Enhanced email fetch request with multiple filtering options. Supports Gmail aliases (john.doe+work@gmail.com matches johndoe@gmail.com) and domain mail (any@company.com matches domain mail account for company.com)"
// @Success 200 {object} map[string]interface{} "Successful response with emails and metadata"
// @Failure 400 {string} string "Bad Request - Invalid parameters or missing required fields"
// @Failure 404 {string} string "Not Found - Account not found (after trying direct match, Gmail alias normalization, and domain matching)"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/fetch-emails [post]
func (h *APIHandler) FetchEmailsHandler(w http.ResponseWriter, r *http.Request) {
	var request FetchEmailsRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate input
	if request.EmailAddress == "" && request.AccountID == 0 {
		http.Error(w, "Either email_address or account_id must be provided", http.StatusBadRequest)
		return
	}

	// Get account from database
	var account *models.EmailAccount
	var err error

	if request.AccountID != 0 {
		account, err = h.EmailAccountRepo.GetByID(request.AccountID)
	} else {
		account, err = h.EmailAccountRepo.GetByEmail(request.EmailAddress)
	}

	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Parse and validate options
	options, err := h.parseRequestOptions(request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Fetch emails using the account with options
	emails, err := h.Fetcher.FetchEmailsWithOptions(*account, options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"count":  len(emails),
		"emails": emails,
		"options": map[string]interface{}{
			"mailbox":           options.Mailbox,
			"limit":             options.Limit,
			"offset":            options.Offset,
			"fetch_from_server": options.FetchFromServer,
			"include_body":      options.IncludeBody,
			"sort_by":           options.SortBy,
		},
	})
}

// parseRequestOptions converts FetchEmailsRequest to FetchEmailsOptions
func (h *APIHandler) parseRequestOptions(request FetchEmailsRequest) (services.FetchEmailsOptions, error) {
	options := services.FetchEmailsOptions{
		Mailbox:         "INBOX",
		Limit:           10,
		Offset:          0,
		FetchFromServer: false,
		IncludeBody:     false,
		SortBy:          "date_desc",
	}

	// Set mailbox
	if request.Mailbox != "" {
		options.Mailbox = request.Mailbox
	}

	// Set limit with validation
	if request.Limit > 0 {
		if request.Limit > 100 {
			return options, fmt.Errorf("limit cannot exceed 100")
		}
		options.Limit = request.Limit
	}

	// Set offset
	if request.Offset >= 0 {
		options.Offset = request.Offset
	}

	// Parse date range
	if request.StartDate != "" {
		startDate, err := time.Parse(time.RFC3339, request.StartDate)
		if err != nil {
			return options, fmt.Errorf("invalid start_date format, use RFC3339: %v", err)
		}
		options.StartDate = &startDate
	}

	if request.EndDate != "" {
		endDate, err := time.Parse(time.RFC3339, request.EndDate)
		if err != nil {
			return options, fmt.Errorf("invalid end_date format, use RFC3339: %v", err)
		}
		options.EndDate = &endDate
	}

	// Validate date range
	if options.StartDate != nil && options.EndDate != nil && options.StartDate.After(*options.EndDate) {
		return options, fmt.Errorf("start_date cannot be after end_date")
	}

	// Set search query
	if request.SearchQuery != "" {
		options.SearchQuery = request.SearchQuery
	}

	// Set fetch from server flag
	options.FetchFromServer = request.FetchFromServer

	// Set include body flag
	options.IncludeBody = request.IncludeBody

	// Validate and set sort order
	if request.SortBy != "" {
		validSortOptions := map[string]bool{
			"date_desc":    true,
			"date_asc":     true,
			"subject_asc":  true,
			"subject_desc": true,
		}
		if !validSortOptions[request.SortBy] {
			return options, fmt.Errorf("invalid sort_by option, valid options are: date_desc, date_asc, subject_asc, subject_desc")
		}
		options.SortBy = request.SortBy
	}

	return options, nil
}

// CreateAccountHandler creates a new email account
// @Summary Create a new email account
// @Description Create a new email account
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body CreateAccountRequest true "Email Account"
// @Success 201 {object} models.EmailAccount
// @Router /api/accounts [post]
func (h *APIHandler) CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	var request CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert request to model
	account := models.EmailAccount{
		EmailAddress:     request.EmailAddress,
		AuthType:         request.AuthType,
		Password:         request.Password,
		Token:            request.Token,
		MailProviderID:   request.MailProviderID,
		OAuth2ProviderID: request.OAuth2ProviderID,
		Proxy:            request.Proxy,
		IsDomainMail:     request.IsDomainMail,
		Domain:           request.Domain,
		CustomSettings:   request.CustomSettings,
	}

	if err := h.EmailAccountRepo.Create(&account); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log activity
	userID := getUserIDFromContext(r)
	h.activityLogger.LogAccountActivity(models.ActivityAccountAdded, &account, userID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(account)
}

// GetAccountsHandler retrieves all email accounts
// @Summary Get all email accounts
// @Description Get all email accounts
// @Tags accounts
// @Accept json
// @Produce json
// @Success 200 {array} models.EmailAccount
// @Router /api/accounts [get]
func (h *APIHandler) GetAccountsHandler(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.EmailAccountRepo.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

// GetAccountsPaginatedHandler retrieves email accounts with pagination
// @Summary Get email accounts with pagination
// @Description Get email accounts with pagination support
// @Tags accounts
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort_by query string false "Sort field (default: created_at)"
// @Param sort_order query string false "Sort order: asc or desc (default: desc)"
// @Param search query string false "Search term for email address (default: ”)"
// @Success 200 {object} PaginatedAccountsResponse
// @Router /api/accounts/paginated [get]
func (h *APIHandler) GetAccountsPaginatedHandler(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	limit := 10
	sortBy := "created_at"
	sortOrder := "desc"
	search := ""

	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	if s := r.URL.Query().Get("sort_by"); s != "" {
		sortBy = s
	}

	if o := r.URL.Query().Get("sort_order"); o == "asc" || o == "desc" {
		sortOrder = o
	}

	// 获取搜索参数
	search = r.URL.Query().Get("search")

	// 输出日志，便于调试
	log.Printf("搜索参数: '%s'", search)

	// 获取分页数据
	accounts, total, err := h.EmailAccountRepo.GetAllPaginated(page, limit, sortBy, sortOrder, search)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 计算总页数
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	// 构建响应
	response := PaginatedAccountsResponse{
		Data:       accounts,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// BatchVerifyAccountsRequest represents the request for batch account verification
type BatchVerifyAccountsRequest struct {
	AccountIDs []uint `json:"account_ids"`
}

// BatchVerifyAccountsResponse represents the response for batch account verification
type BatchVerifyAccountsResponse struct {
	SuccessCount int                        `json:"success_count"`
	ErrorCount   int                        `json:"error_count"`
	Results      []BatchVerifyAccountResult `json:"results"`
}

// BatchVerifyAccountResult represents the result for a single account verification
type BatchVerifyAccountResult struct {
	AccountID    uint   `json:"account_id"`
	EmailAddress string `json:"email_address"`
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
	Error        string `json:"error,omitempty"`
}

// BatchVerifyAccountsHandler handles batch account verification
// @Summary Batch verify account connectivity
// @Description Verify connectivity for multiple email accounts in batch
// @Tags accounts
// @Accept json
// @Produce json
// @Param request body BatchVerifyAccountsRequest true "Batch account verification request"
// @Success 200 {object} BatchVerifyAccountsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts/batch-verify [post]
func (h *APIHandler) BatchVerifyAccountsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req BatchVerifyAccountsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.AccountIDs) == 0 {
		http.Error(w, "At least one account ID is required", http.StatusBadRequest)
		return
	}

	// Limit batch size to prevent timeout
	const maxBatchSize = 20
	if len(req.AccountIDs) > maxBatchSize {
		http.Error(w, fmt.Sprintf("Batch size cannot exceed %d accounts", maxBatchSize), http.StatusBadRequest)
		return
	}

	fmt.Printf("[DEBUG] Starting batch verification for %d accounts\n", len(req.AccountIDs))

	var response BatchVerifyAccountsResponse
	var results []BatchVerifyAccountResult

	// Process each account
	for _, accountID := range req.AccountIDs {
		result := h.verifyAccountByID(accountID)
		results = append(results, result)

		if result.Success {
			response.SuccessCount++
		} else {
			response.ErrorCount++
		}

		fmt.Printf("[DEBUG] Verified account %d (%s): success=%t\n",
			accountID, result.EmailAddress, result.Success)
	}

	response.Results = results

	fmt.Printf("[DEBUG] Batch verification completed: %d success, %d errors in %v\n",
		response.SuccessCount, response.ErrorCount, time.Since(start))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// verifyAccountByID verifies a single account by ID and updates verification status
func (h *APIHandler) verifyAccountByID(accountID uint) BatchVerifyAccountResult {
	// Get account from database
	account, err := h.EmailAccountRepo.GetByID(accountID)
	if err != nil {
		return BatchVerifyAccountResult{
			AccountID:    accountID,
			EmailAddress: fmt.Sprintf("account_%d", accountID),
			Success:      false,
			Error:        "Account not found: " + err.Error(),
		}
	}

	// Verify connection
	err = h.Fetcher.VerifyConnection(*account)

	result := BatchVerifyAccountResult{
		AccountID:    accountID,
		EmailAddress: account.EmailAddress,
		Success:      err == nil,
	}

	if err != nil {
		result.Message = "Connection verification failed"
		result.Error = err.Error()
		fmt.Printf("[DEBUG] Verification failed for account %d (%s): %v\n",
			accountID, account.EmailAddress, err)
	} else {
		// Update account verification status in database
		account.IsVerified = true
		account.VerifiedAt = timePtr(time.Now())

		if updateErr := h.EmailAccountRepo.Update(account); updateErr != nil {
			fmt.Printf("[DEBUG] Failed to update verification status for account %d: %v\n",
				accountID, updateErr)
			result.Message = "Connection verified but failed to update database"
			result.Error = updateErr.Error()
		} else {
			result.Message = "Connection verified successfully"
			fmt.Printf("[DEBUG] Successfully verified and updated account %d (%s)\n",
				accountID, account.EmailAddress)
		}
	}

	return result
}

// timePtr returns a pointer to the given time
func timePtr(t time.Time) *time.Time {
	return &t
}

// syncEmailsForToQuery 根据to_query参数同步对应账户的邮件
func (h *APIHandler) syncEmailsForToQuery(toQuery string) error {
	// 使用EmailAccountRepository的GetByEmailOrAlias方法，它已经处理了别名和域名邮箱
	account, err := h.EmailAccountRepo.GetByEmailOrAlias(toQuery)
	if err != nil {
		return fmt.Errorf("failed to find account for email %s: %w", toQuery, err)
	}
	if account == nil {
		return fmt.Errorf("no account found for email %s", toQuery)
	}

	// 使用EmailScheduler触发立即同步，类似FetchNowHandler的实现
	if h.EmailScheduler != nil {
		// 获取该账户的所有订阅
		subscriptions := h.EmailScheduler.GetAccountSubscriptions(account.ID)
		if len(subscriptions) == 0 {
			// 如果没有订阅，记录日志但不报错，因为可能数据库中已有邮件
			return nil
		}

		// 对每个订阅触发立即同步
		var errors []string
		for _, sub := range subscriptions {
			_, err := h.EmailScheduler.FetchNow(sub.ID, false) // 不强制刷新
			if err != nil {
				errors = append(errors, fmt.Sprintf("Subscription %s: %v", sub.ID, err))
			}
		}

		// 如果有错误，返回合并的错误信息
		if len(errors) > 0 {
			return fmt.Errorf("sync errors for account %d: %s", account.ID, strings.Join(errors, "; "))
		}
	}

	return nil
}

// CreateSubscriptionHandler creates a new email subscription
// @Summary Create email subscription
// @Description Create a new email subscription for real-time monitoring
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param request body CreateSubscriptionRequest true "Subscription configuration"
// @Success 201 {object} SubscriptionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/subscriptions [post]
func (h *APIHandler) CreateSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate account exists
	account, err := h.EmailAccountRepo.GetByID(req.AccountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Set defaults
	if req.Mailbox == "" {
		req.Mailbox = "INBOX"
	}
	if req.PollingInterval <= 0 {
		req.PollingInterval = 60
	} else if req.PollingInterval < 30 {
		req.PollingInterval = 30 // Minimum 30 seconds
	}

	// Create subscription
	subscriptionID, err := h.EmailScheduler.SubscribeSimple(
		account.ID,
		account.EmailAddress, // 传递真实的邮箱地址
		req.Mailbox,
		time.Duration(req.PollingInterval)*time.Second,
		req.IncludeBody,
		req.Filters,
	)
	if err != nil {
		http.Error(w, "Failed to create subscription: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get subscription details
	subscription := h.EmailScheduler.GetSubscription(subscriptionID)
	if subscription == nil {
		http.Error(w, "Failed to retrieve created subscription", http.StatusInternalServerError)
		return
	}

	// Log activity
	userID := getUserIDFromContext(r)
	h.activityLogger.LogActivity(
		models.ActivitySubscribed,
		"创建邮件订阅",
		fmt.Sprintf("为账户 %s 创建了邮件订阅，邮箱: %s，轮询间隔: %d秒", account.EmailAddress, req.Mailbox, req.PollingInterval),
		userID,
		map[string]interface{}{
			"subscription_id":  subscriptionID,
			"account_id":       account.ID,
			"mailbox":          req.Mailbox,
			"polling_interval": req.PollingInterval,
			"include_body":     req.IncludeBody,
		},
	)

	// Build response
	response := SubscriptionResponse{
		ID:              subscriptionID,
		AccountID:       account.ID,
		EmailAddress:    account.EmailAddress,
		Mailbox:         req.Mailbox,
		PollingInterval: req.PollingInterval,
		IncludeBody:     req.IncludeBody,
		Filters:         req.Filters,
		Status:          "active",
		CreatedAt:       time.Now(),
		NextCheckAt:     subscription.NextRunAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetSubscriptionsHandler retrieves all active subscriptions
// @Summary List email subscriptions
// @Description Get all active email subscriptions
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param account_id query int false "Filter by account ID"
// @Success 200 {object} SubscriptionListResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/subscriptions [get]
func (h *APIHandler) GetSubscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse optional account ID filter
	var accountID uint
	if accountIDStr := r.URL.Query().Get("account_id"); accountIDStr != "" {
		if id, err := strconv.ParseUint(accountIDStr, 10, 32); err == nil {
			accountID = uint(id)
		}
	}

	// Get all subscriptions
	allSubscriptions := h.EmailScheduler.GetAllSubscriptions()

	// Build response
	subscriptions := []SubscriptionResponse{} // Initialize as empty slice instead of nil
	for _, sub := range allSubscriptions {
		// Extract account ID from metadata
		var subAccountID uint
		if accountIDMeta, ok := sub.Metadata["accountID"].(uint); ok {
			subAccountID = accountIDMeta
		} else if accountIDMeta, ok := sub.Metadata["accountID"].(float64); ok {
			subAccountID = uint(accountIDMeta)
		}

		// Apply account filter if specified
		if accountID != 0 && subAccountID != accountID {
			continue
		}

		// Get account details
		account, err := h.EmailAccountRepo.GetByID(subAccountID)
		if err != nil {
			continue // Skip if account not found
		}

		// Extract other metadata
		mailbox := "INBOX"
		if len(sub.Filter.Folders) > 0 {
			mailbox = sub.Filter.Folders[0]
		}

		interval := 60
		if intervalMeta, ok := sub.Metadata["interval"].(time.Duration); ok {
			interval = int(intervalMeta.Seconds())
		}

		includeBody := false
		if includeBodyMeta, ok := sub.Metadata["includeBody"].(bool); ok {
			includeBody = includeBodyMeta
		}

		var filters *SubscriptionFilters
		if filtersMeta, ok := sub.Metadata["filters"].(*SubscriptionFilters); ok {
			filters = filtersMeta
		}

		status := "active"
		if sub.Context.Err() != nil {
			status = "cancelled"
		} else if sub.ExpiresAt != nil && time.Now().After(*sub.ExpiresAt) {
			status = "expired"
		}

		subscriptions = append(subscriptions, SubscriptionResponse{
			ID:              sub.ID,
			AccountID:       subAccountID,
			EmailAddress:    account.EmailAddress,
			Mailbox:         mailbox,
			PollingInterval: interval,
			IncludeBody:     includeBody,
			Filters:         filters,
			Status:          status,
			CreatedAt:       sub.CreatedAt,
			LastCheckedAt:   sub.LastEmailAt,
			NextCheckAt:     sub.NextRunAt,
		})
	}

	response := SubscriptionListResponse{
		Subscriptions: subscriptions,
		Total:         len(subscriptions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteSubscriptionHandler cancels an email subscription
// @Summary Cancel email subscription
// @Description Cancel an active email subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "Subscription ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/subscriptions/{id} [delete]
func (h *APIHandler) DeleteSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subscriptionID := vars["id"]

	if subscriptionID == "" {
		http.Error(w, "Subscription ID is required", http.StatusBadRequest)
		return
	}

	// Check if subscription exists
	subscription := h.EmailScheduler.GetSubscription(subscriptionID)
	if subscription == nil {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		return
	}

	// Get subscription metadata for logging
	var accountEmail string
	if accountIDMeta, ok := subscription.Metadata["accountID"].(uint); ok {
		if account, err := h.EmailAccountRepo.GetByID(accountIDMeta); err == nil {
			accountEmail = account.EmailAddress
		}
	}

	// Unsubscribe
	if err := h.EmailScheduler.Unsubscribe(subscriptionID); err != nil {
		http.Error(w, "Failed to cancel subscription: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Log activity
	userID := getUserIDFromContext(r)
	h.activityLogger.LogActivity(
		models.ActivityUnsubscribed,
		"取消邮件订阅",
		fmt.Sprintf("取消了订阅 %s，账户: %s", subscriptionID, accountEmail),
		userID,
		map[string]interface{}{
			"subscription_id": subscriptionID,
			"account_email":   accountEmail,
		},
	)

	w.WriteHeader(http.StatusNoContent)
}

// GetCacheStatsHandler retrieves email cache statistics
// @Summary Get cache statistics
// @Description Get email cache statistics and performance metrics
// @Tags cache
// @Accept json
// @Produce json
// @Success 200 {object} CacheStatsResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/cache/stats [get]
func (h *APIHandler) GetCacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Get cache stats from EmailScheduler
	stats := h.EmailScheduler.GetCacheStats()

	// Get account-specific stats
	var accountStats []AccountCacheStats
	accounts, err := h.EmailAccountRepo.GetAll()
	if err == nil {
		for _, account := range accounts {
			accountStat := h.EmailScheduler.GetAccountCacheStats(account.ID)
			if accountStat != nil {
				accountStats = append(accountStats, AccountCacheStats{
					AccountID:    account.ID,
					EmailAddress: account.EmailAddress,
					EmailCount:   accountStat.EmailCount,
					CacheSize:    accountStat.Size,
					OldestEmail:  accountStat.OldestEmail,
					NewestEmail:  accountStat.NewestEmail,
				})
			}
		}
	}

	response := CacheStatsResponse{
		TotalEmails:  stats.TotalEmails,
		TotalSize:    0, // TODO: Calculate actual size
		AccountStats: accountStats,
		HitRate:      stats.HitRate,
		LastCleanup:  nil, // TODO: Track cleanup time
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// FetchNowHandler triggers immediate email fetch
// @Summary Fetch emails immediately
// @Description Trigger immediate email fetch for subscriptions
// @Tags emails
// @Accept json
// @Produce json
// @Param request body FetchNowRequest true "Fetch configuration"
// @Success 200 {object} FetchNowResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/emails/fetch-now [post]
func (h *APIHandler) FetchNowHandler(w http.ResponseWriter, r *http.Request) {
	var req FetchNowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	startTime := time.Now()
	var totalNew, totalProcessed int
	var errors []string

	// If subscription ID is provided, fetch for that subscription
	if req.SubscriptionID != "" {
		subscription := h.EmailScheduler.GetSubscription(req.SubscriptionID)
		if subscription == nil {
			http.Error(w, "Subscription not found", http.StatusNotFound)
			return
		}

		result, err := h.EmailScheduler.FetchNow(req.SubscriptionID, req.ForceRefresh)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			totalNew += result.NewEmails
			totalProcessed += result.ProcessedEmails
		}
	} else if req.AccountID != 0 {
		// Fetch for all subscriptions of an account
		subscriptions := h.EmailScheduler.GetAccountSubscriptions(req.AccountID)
		for _, sub := range subscriptions {
			result, err := h.EmailScheduler.FetchNow(sub.ID, req.ForceRefresh)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Subscription %s: %v", sub.ID, err))
			} else {
				totalNew += result.NewEmails
				totalProcessed += result.ProcessedEmails
			}
		}
	} else {
		// Fetch for all subscriptions
		allSubscriptions := h.EmailScheduler.GetAllSubscriptions()
		for _, sub := range allSubscriptions {
			result, err := h.EmailScheduler.FetchNow(sub.ID, req.ForceRefresh)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Subscription %s: %v", sub.ID, err))
			} else {
				totalNew += result.NewEmails
				totalProcessed += result.ProcessedEmails
			}
		}
	}

	processingTime := time.Since(startTime)

	response := FetchNowResponse{
		Status:           "success",
		NewEmails:        totalNew,
		TotalProcessed:   totalProcessed,
		ProcessingTimeMs: processingTime.Milliseconds(),
		Errors:           errors,
	}

	if len(errors) > 0 && totalProcessed == 0 {
		response.Status = "failed"
	} else if len(errors) > 0 {
		response.Status = "partial"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// VerifyAccountRequest represents the request for verifying account connectivity
type VerifyAccountRequest struct {
	AccountID      *uint             `json:"account_id,omitempty"`
	EmailAddress   string            `json:"email_address,omitempty"`
	Password       string            `json:"password,omitempty"`
	AuthType       string            `json:"auth_type,omitempty"`
	MailProviderID uint              `json:"mail_provider_id,omitempty"`
	CustomSettings map[string]string `json:"custom_settings,omitempty"`
	Proxy          string            `json:"proxy,omitempty"`
}

// VerifyAccountResponse represents the response for account verification
type VerifyAccountResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// VerifyAccountHandler handles account connectivity verification
// @Summary Verify account connectivity
// @Description Verify if an email account can connect successfully using IMAP or OAuth2
// @Tags accounts
// @Accept json
// @Produce json
// @Param request body VerifyAccountRequest true "Account verification request"
// @Success 200 {object} VerifyAccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts/verify [post]
func (h *APIHandler) VerifyAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req VerifyAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var account models.EmailAccount
	var existingAccount *models.EmailAccount
	var err error

	// If account ID is provided, fetch the account from database
	if req.AccountID != nil {
		existingAccount, err = h.EmailAccountRepo.GetByID(*req.AccountID)
		if err != nil {
			response := VerifyAccountResponse{
				Success: false,
				Message: "Failed to fetch account",
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		account = *existingAccount

		// Debug: Log the loaded CustomSettings
		fmt.Printf("[DEBUG] Loaded account CustomSettings: %+v\n", account.CustomSettings)
		if len(account.CustomSettings) == 0 {
			fmt.Printf("[DEBUG] CustomSettings is empty for account %d\n", account.ID)
		}
	} else {
		// Create a temporary account object from the provided details
		if req.EmailAddress == "" || req.MailProviderID == 0 {
			response := VerifyAccountResponse{
				Success: false,
				Message: "Email address and mail provider ID are required",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Get mail provider
		provider, err := h.MailProviderRepo.GetByID(req.MailProviderID)
		if err != nil {
			response := VerifyAccountResponse{
				Success: false,
				Message: "Invalid mail provider",
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		var mailProviderID *uint
		if req.MailProviderID != 0 {
			mailProviderID = &req.MailProviderID
		}

		account = models.EmailAccount{
			EmailAddress:   req.EmailAddress,
			Password:       req.Password,
			AuthType:       models.AuthType(req.AuthType),
			MailProviderID: mailProviderID,
			MailProvider:   provider,
			CustomSettings: models.JSONMap(req.CustomSettings),
			Proxy:          req.Proxy,
		}

		// Set default auth type if not provided
		if account.AuthType == "" {
			account.AuthType = models.AuthTypePassword
		}
	}

	// Verify the connection
	err = h.Fetcher.VerifyConnection(account)

	response := VerifyAccountResponse{
		Success: err == nil,
	}

	if err != nil {
		response.Message = "Connection verification failed"
		response.Error = err.Error()
	} else {
		response.Message = "Connection verified successfully"

		// If account ID is provided and verification successful, update the account's verification status
		if req.AccountID != nil && err == nil {
			now := time.Now()
			existingAccount.IsVerified = true
			existingAccount.VerifiedAt = &now

			if updateErr := h.EmailAccountRepo.Update(existingAccount); updateErr != nil {
				log.Printf("Failed to update account verification status: %v", updateErr)
				// Don't fail the response, just log the error
			} else {
				// Log activity for successful verification
				userID := getUserIDFromContext(r)
				h.activityLogger.LogAccountActivity(models.ActivityAccountVerified, existingAccount, userID)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TestExtractorTemplateHandler tests an extractor template with a specific email or custom email content
// @Summary Test extractor template
// @Description Test an extractor template with a specific email or custom email content
// @Tags extractor-templates
// @Accept json
// @Produce json
// @Param id path int true "Template ID"
// @Param request body TestExtractorTemplateRequest true "Test request"
// @Success 200 {array} TestExtractorResult "Test results"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 404 {object} ErrorResponse "Template not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/extractor-templates/{id}/test [post]
func (h *APIHandler) TestExtractorTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	var req TestExtractorTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get the template
	templateRepo := repository.NewExtractorTemplateRepository(database.GetDB())
	template, err := templateRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// Prepare extractors
	extractors := req.Extractors
	if len(extractors) == 0 {
		// Use template extractors if none provided
		for _, ext := range template.Extractors {
			extractors = append(extractors, ExtractorConfig{
				Field: ext.Field,
				Type:  ext.Type,
				Match: ext.Match,

				Extract: ext.Extract,
			})
		}
	}

	// Get email content
	var emailContent map[string]string
	if req.EmailID != nil {
		// Fetch email from database
		emailRepo := repository.NewEmailRepository(database.GetDB())
		email, err := emailRepo.GetByID(*req.EmailID)
		if err != nil {
			http.Error(w, "Email not found", http.StatusNotFound)
			return
		}
		emailContent = map[string]string{
			"from":      strings.Join(email.From, ", "),
			"to":        strings.Join(email.To, ", "),
			"cc":        strings.Join(email.Cc, ", "),
			"subject":   email.Subject,
			"body":      email.Body,
			"html_body": email.HTMLBody,
		}
	} else if req.CustomEmail != nil {
		// Use custom email content
		emailContent = map[string]string{
			"from":      req.CustomEmail.From,
			"to":        req.CustomEmail.To,
			"cc":        req.CustomEmail.Cc,
			"subject":   req.CustomEmail.Subject,
			"body":      req.CustomEmail.Body,
			"html_body": req.CustomEmail.HTMLBody,
		}
	} else {
		http.Error(w, "Either email_id or custom_email must be provided", http.StatusBadRequest)
		return
	}

	// Test each extractor
	results := []TestExtractorResult{}
	extractorService := services.NewExtractorService()

	for _, extractor := range extractors {
		result := TestExtractorResult{
			Field: extractor.Field,
			Type:  extractor.Type,
		}

		// Get the content to extract from
		var content string
		if extractor.Field == "ALL" {
			// Combine all fields
			content = fmt.Sprintf("From: %s\nTo: %s\nCc: %s\nSubject: %s\n\n%s",
				emailContent["from"],
				emailContent["to"],
				emailContent["cc"],
				emailContent["subject"],
				emailContent["body"])
		} else {
			content = emailContent[extractor.Field]
		}

		// Create a temporary email for extraction
		tempEmail := models.Email{
			From:     models.StringSlice{content},
			To:       models.StringSlice{},
			Cc:       models.StringSlice{},
			Subject:  "",
			Body:     "",
			HTMLBody: "",
		}

		// Set the appropriate field based on extractor field
		switch extractor.Field {
		case "from":
			tempEmail.From = models.StringSlice{content}
		case "to":
			tempEmail.To = models.StringSlice{content}
		case "cc":
			tempEmail.Cc = models.StringSlice{content}
		case "subject":
			tempEmail.Subject = content
		case "body":
			tempEmail.Body = content
		case "html_body":
			tempEmail.HTMLBody = content
		}

		// Extract using the service
		extractResult, err := extractorService.ExtractFromEmail(tempEmail, []services.ExtractorConfig{
			{
				Field: services.ExtractorField(extractor.Field),
				Type:  services.ExtractorType(extractor.Type),
				Match: extractor.Match,

				Extract: extractor.Extract,
			},
		})
		if err != nil {
			result.Error = err.Error()
		} else if extractResult != nil && len(extractResult.Matches) > 0 {
			result.Result = &extractResult.Matches[0]
		}

		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// CreateExtractorTemplateHandler creates a new extractor template
// @Summary Create a new extractor template
// @Description Create a new extractor template for reusable email extraction patterns
// @Tags extractor-templates
// @Accept json
// @Produce json
// @Param request body CreateExtractorTemplateRequest true "Create extractor template request"
// @Success 201 {object} ExtractorTemplateResponse
// @Failure 400 {string} string "Invalid request body"
// @Failure 500 {string} string "Failed to create extractor template"
// @Router /api/extractor-templates [post]
func (h *APIHandler) CreateExtractorTemplateHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateExtractorTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate extractors
	for i, extractor := range req.Extractors {
		if extractor.Field == "" || extractor.Type == "" || extractor.Extract == "" {
			http.Error(w, fmt.Sprintf("Extractor %d is missing required fields", i), http.StatusBadRequest)
			return
		}

		// Validate field values
		validFields := map[string]bool{
			"ALL": true, "from": true, "to": true, "cc": true,
			"subject": true, "body": true, "html_body": true, "headers": true,
		}
		if !validFields[extractor.Field] {
			http.Error(w, fmt.Sprintf("Invalid field '%s' in extractor %d", extractor.Field, i), http.StatusBadRequest)
			return
		}

		// Validate type values
		validTypes := map[string]bool{"regex": true, "js": true, "gotemplate": true}
		if !validTypes[extractor.Type] {
			http.Error(w, fmt.Sprintf("Invalid type '%s' in extractor %d", extractor.Type, i), http.StatusBadRequest)
			return
		}
	}

	// Convert API extractors to model extractors
	var modelExtractors models.ExtractorTemplateConfigs
	for _, apiExtractor := range req.Extractors {
		modelExtractors = append(modelExtractors, models.ExtractorTemplateConfig{
			Field: apiExtractor.Field,
			Type:  apiExtractor.Type,
			Match: apiExtractor.Match,

			Extract: apiExtractor.Extract,
		})
	}

	// Create template
	template := &models.ExtractorTemplate{
		Name:        req.Name,
		Description: req.Description,
		Extractors:  modelExtractors,
	}

	templateRepo := repository.NewExtractorTemplateRepository(database.GetDB())
	if err := templateRepo.Create(template); err != nil {
		http.Error(w, "Failed to create extractor template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response
	response := ExtractorTemplateResponse{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Extractors:  req.Extractors,
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetExtractorTemplatesHandler retrieves all extractor templates
// @Summary Get all extractor templates
// @Description Get all extractor templates
// @Tags extractor-templates
// @Accept json
// @Produce json
// @Success 200 {array} ExtractorTemplateResponse
// @Failure 500 {string} string "Failed to retrieve extractor templates"
// @Router /api/extractor-templates [get]
func (h *APIHandler) GetExtractorTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templateRepo := repository.NewExtractorTemplateRepository(database.GetDB())
	templates, err := templateRepo.GetAll()
	if err != nil {
		http.Error(w, "Failed to retrieve extractor templates: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response
	var response []ExtractorTemplateResponse
	for _, template := range templates {
		var extractors []ExtractorConfig
		for _, extractor := range template.Extractors {
			extractors = append(extractors, ExtractorConfig{
				Field: extractor.Field,
				Type:  extractor.Type,
				Match: extractor.Match,

				Extract: extractor.Extract,
			})
		}

		response = append(response, ExtractorTemplateResponse{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			Extractors:  extractors,
			CreatedAt:   template.CreatedAt,
			UpdatedAt:   template.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetExtractorTemplateHandler retrieves a specific extractor template
// @Summary Get an extractor template by ID
// @Description Get an extractor template by ID
// @Tags extractor-templates
// @Accept json
// @Produce json
// @Param id path int true "Extractor Template ID"
// @Success 200 {object} ExtractorTemplateResponse
// @Failure 400 {string} string "Invalid template ID"
// @Failure 404 {string} string "Extractor template not found"
// @Router /api/extractor-templates/{id} [get]
func (h *APIHandler) GetExtractorTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	templateRepo := repository.NewExtractorTemplateRepository(database.GetDB())
	template, err := templateRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Convert to response
	var extractors []ExtractorConfig
	for _, extractor := range template.Extractors {
		extractors = append(extractors, ExtractorConfig{
			Field: extractor.Field,
			Type:  extractor.Type,
			Match: extractor.Match,

			Extract: extractor.Extract,
		})
	}

	response := ExtractorTemplateResponse{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Extractors:  extractors,
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateExtractorTemplateHandler updates an existing extractor template
// @Summary Update an extractor template
// @Description Update an existing extractor template
// @Tags extractor-templates
// @Accept json
// @Produce json
// @Param id path int true "Extractor Template ID"
// @Param request body UpdateExtractorTemplateRequest true "Update extractor template request"
// @Success 200 {object} ExtractorTemplateResponse
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Extractor template not found"
// @Failure 500 {string} string "Failed to update extractor template"
// @Router /api/extractor-templates/{id} [put]
func (h *APIHandler) UpdateExtractorTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	var req UpdateExtractorTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	templateRepo := repository.NewExtractorTemplateRepository(database.GetDB())
	template, err := templateRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Update fields if provided
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if len(req.Extractors) > 0 {
		// Validate extractors
		for i, extractor := range req.Extractors {
			if extractor.Field == "" || extractor.Type == "" || extractor.Extract == "" {
				http.Error(w, fmt.Sprintf("Extractor %d is missing required fields", i), http.StatusBadRequest)
				return
			}

			// Validate field values
			validFields := map[string]bool{
				"ALL": true, "from": true, "to": true, "cc": true,
				"subject": true, "body": true, "html_body": true, "headers": true,
			}
			if !validFields[extractor.Field] {
				http.Error(w, fmt.Sprintf("Invalid field '%s' in extractor %d", extractor.Field, i), http.StatusBadRequest)
				return
			}

			// Validate type values
			validTypes := map[string]bool{"regex": true, "js": true, "gotemplate": true}
			if !validTypes[extractor.Type] {
				http.Error(w, fmt.Sprintf("Invalid type '%s' in extractor %d", extractor.Type, i), http.StatusBadRequest)
				return
			}
		}

		// Convert API extractors to model extractors
		var modelExtractors models.ExtractorTemplateConfigs
		for _, apiExtractor := range req.Extractors {
			modelExtractors = append(modelExtractors, models.ExtractorTemplateConfig{
				Field: apiExtractor.Field,
				Type:  apiExtractor.Type,
				Match: apiExtractor.Match,

				Extract: apiExtractor.Extract,
			})
		}
		template.Extractors = modelExtractors
	}

	if err := templateRepo.Update(template); err != nil {
		http.Error(w, "Failed to update extractor template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response
	var extractors []ExtractorConfig
	for _, extractor := range template.Extractors {
		extractors = append(extractors, ExtractorConfig{
			Field: extractor.Field,
			Type:  extractor.Type,
			Match: extractor.Match,

			Extract: extractor.Extract,
		})
	}

	response := ExtractorTemplateResponse{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Extractors:  extractors,
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteExtractorTemplateHandler deletes an extractor template
// @Summary Delete an extractor template
// @Description Delete an extractor template by ID
// @Tags extractor-templates
// @Accept json
// @Produce json
// @Param id path int true "Extractor Template ID"
// @Success 204 {string} string "No Content"
// @Failure 400 {string} string "Invalid template ID"
// @Failure 500 {string} string "Failed to delete extractor template"
// @Router /api/extractor-templates/{id} [delete]
func (h *APIHandler) DeleteExtractorTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	templateRepo := repository.NewExtractorTemplateRepository(database.GetDB())
	if err := templateRepo.Delete(uint(id)); err != nil {
		http.Error(w, "Failed to delete extractor template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetExtractorTemplatesPaginatedHandler retrieves extractor templates with pagination and search
// @Summary Get extractor templates with pagination
// @Description Get extractor templates with pagination support and name search
// @Tags extractor-templates
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort_by query string false "Sort field (default: created_at)"
// @Param sort_order query string false "Sort order: asc or desc (default: desc)"
// @Param search query string false "Search term for template name"
// @Success 200 {object} PaginatedExtractorTemplatesResponse
// @Failure 500 {string} string "Failed to retrieve extractor templates"
// @Router /api/extractor-templates/paginated [get]
func (h *APIHandler) GetExtractorTemplatesPaginatedHandler(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	limit := 10
	sortBy := "created_at"
	sortOrder := "desc"
	search := ""

	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	if s := r.URL.Query().Get("sort_by"); s != "" {
		sortBy = s
	}

	if o := r.URL.Query().Get("sort_order"); o == "asc" || o == "desc" {
		sortOrder = o
	}

	// 获取搜索参数
	search = r.URL.Query().Get("search")

	// 获取分页数据
	templateRepo := repository.NewExtractorTemplateRepository(database.GetDB())
	templates, total, err := templateRepo.GetAllPaginated(page, limit, sortBy, sortOrder, search)
	if err != nil {
		http.Error(w, "Failed to retrieve extractor templates: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 计算总页数
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	// 转换为响应格式
	var responseTemplates []ExtractorTemplateResponse
	for _, template := range templates {
		// 转换 ExtractorTemplateConfigs 到 []ExtractorConfig
		var extractors []ExtractorConfig
		for _, ec := range template.Extractors {
			extractors = append(extractors, ExtractorConfig{
				Field: ec.Field,
				Type:  ec.Type,
				Match: ec.Match,

				Extract: ec.Extract,
			})
		}

		responseTemplates = append(responseTemplates, ExtractorTemplateResponse{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			Extractors:  extractors,
			CreatedAt:   template.CreatedAt,
			UpdatedAt:   template.UpdatedAt,
		})
	}

	// 构建响应
	response := PaginatedExtractorTemplatesResponse{
		Data:       responseTemplates,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAccountHandler retrieves a specific email account
// @Summary Get an email account by ID
// @Description Get an email account by ID
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Success 200 {object} models.EmailAccount
// @Router /api/accounts/{id} [get]
func (h *APIHandler) GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	account, err := h.EmailAccountRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// UpdateAccountHandler updates an email account
// @Summary Update an email account
// @Description Update an email account (supports partial updates)
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Param account body UpdateAccountRequest true "Email Account Update"
// @Success 200 {object} models.EmailAccount
// @Router /api/accounts/{id} [put]
func (h *APIHandler) UpdateAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	// Get existing account
	existingAccount, err := h.EmailAccountRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	var request UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Apply partial updates only for provided fields
	if request.EmailAddress != nil {
		existingAccount.EmailAddress = *request.EmailAddress
	}
	if request.AuthType != nil {
		existingAccount.AuthType = *request.AuthType
	}
	if request.Password != nil {
		existingAccount.Password = *request.Password
	}
	if request.Token != nil {
		existingAccount.Token = *request.Token
	}
	if request.MailProviderID != nil {
		existingAccount.MailProviderID = request.MailProviderID
	}
	if request.Proxy != nil {
		existingAccount.Proxy = *request.Proxy
	}
	if request.IsDomainMail != nil {
		existingAccount.IsDomainMail = *request.IsDomainMail
	}
	if request.Domain != nil {
		existingAccount.Domain = *request.Domain
	}
	if request.CustomSettings != nil {
		existingAccount.CustomSettings = *request.CustomSettings
	}
	if request.LastSyncAt != nil {
		existingAccount.LastSyncAt = request.LastSyncAt
	}

	if err := h.EmailAccountRepo.Update(existingAccount); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log activity
	userID := getUserIDFromContext(r)
	h.activityLogger.LogAccountActivity(models.ActivityAccountUpdated, existingAccount, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingAccount)
}

// UpsertAccountHandler creates or updates an email account (Outlook Token flow)
// @Summary Create or update an email account
// @Description Create a new email account or update existing one based on email address
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body CreateAccountRequest true "Email Account"
// @Success 200 {object} models.EmailAccount
// @Success 201 {object} models.EmailAccount
// @Failure 400 {object} ErrorResponse
// @Router /api/accounts/upsert [post]
func (h *APIHandler) UpsertAccountHandler(w http.ResponseWriter, r *http.Request) {
	var request CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// First, try to find existing account by email address
	existingAccount, err := h.EmailAccountRepo.GetByEmail(request.EmailAddress)

	var account *models.EmailAccount
	var activityType models.ActivityType

	if err != nil {
		// Account doesn't exist, create new one
		activityType = models.ActivityAccountAdded

		account = &models.EmailAccount{
			EmailAddress:     request.EmailAddress,
			AuthType:         request.AuthType,
			Password:         request.Password,
			Token:            request.Token,
			MailProviderID:   request.MailProviderID,
			OAuth2ProviderID: request.OAuth2ProviderID,
			Proxy:            request.Proxy,
			IsDomainMail:     request.IsDomainMail,
			Domain:           request.Domain,
			CustomSettings:   request.CustomSettings,
		}

		if err := h.EmailAccountRepo.Create(account); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	} else {
		// Account exists, update it
		activityType = models.ActivityAccountUpdated
		account = existingAccount

		// Update fields if they are provided
		if request.AuthType != "" {
			account.AuthType = request.AuthType
		}
		if request.Password != "" {
			account.Password = request.Password
		}
		if request.Token != "" {
			account.Token = request.Token
		}
		if request.MailProviderID != nil {
			account.MailProviderID = request.MailProviderID
		}
		if request.OAuth2ProviderID != nil {
			account.OAuth2ProviderID = request.OAuth2ProviderID
		}
		if request.Proxy != "" {
			account.Proxy = request.Proxy
		}
		if request.CustomSettings != nil {
			account.CustomSettings = request.CustomSettings
		}

		if err := h.EmailAccountRepo.Update(account); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}

	// Log activity
	userID := getUserIDFromContext(r)
	h.activityLogger.LogAccountActivity(activityType, account, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// DeleteAccountHandler deletes an email account
// @Summary Delete an email account
// @Description Delete an email account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Success 204
// @Router /api/accounts/{id} [delete]
func (h *APIHandler) DeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	// Get account info before deletion for logging
	account, err := h.EmailAccountRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if err := h.EmailAccountRepo.Delete(uint(id)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log activity
	userID := getUserIDFromContext(r)
	h.activityLogger.LogAccountActivity(models.ActivityAccountDeleted, account, userID)

	w.WriteHeader(http.StatusNoContent)
}

// GetProvidersHandler retrieves all mail providers
// @Summary Get all mail providers
// @Description Get all mail providers
// @Tags providers
// @Accept json
// @Produce json
// @Success 200 {array} models.MailProvider
// @Router /api/providers [get]
func (h *APIHandler) GetProvidersHandler(w http.ResponseWriter, r *http.Request) {
	providers, err := h.MailProviderRepo.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

// FetchAndStoreEmailsHandler fetches and stores emails for an account with sync options
// @Summary Fetch and store emails for an account with incremental/full sync support
// @Description Fetch and store emails for an account with support for incremental sync, custom mailboxes, and date ranges. Supports both full sync and incremental sync modes. For incremental sync, maintains sync records to track last sync times per mailbox.
// @Tags account-emails
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Param request body FetchAndStoreRequest false "Sync options (optional - defaults to incremental sync of INBOX)"
// @Success 200 {object} FetchAndStoreResponse "Successful sync operation with detailed results"
// @Failure 400 {string} string "Bad Request - Invalid account ID or request parameters"
// @Failure 404 {string} string "Not Found - Account not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/account-emails/fetch/{id} [post]
func (h *APIHandler) FetchAndStoreEmailsHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}
	accountID := uint(id)

	// Parse request body (optional)
	var request FetchAndStoreRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			// If body is not valid JSON, use defaults
			log.Printf("Failed to parse request body, using defaults: %v", err)
		}
	}

	// Set defaults
	if request.SyncMode == "" {
		request.SyncMode = "incremental"
	}
	if len(request.Mailboxes) == 0 {
		request.Mailboxes = []string{"INBOX"}
	}
	if request.MaxEmailsPerMailbox <= 0 {
		request.MaxEmailsPerMailbox = 1000
	}
	// IncludeBody defaults to true if not specified
	if request.IncludeBody == false && r.Body != nil {
		// Only set to true if body was provided but IncludeBody wasn't explicitly set
		request.IncludeBody = true
	} else if r.Body == nil {
		request.IncludeBody = true
	}

	// Validate account exists
	account, err := h.EmailAccountRepo.GetByID(accountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Parse date parameters
	var defaultStartDate *time.Time
	var endDate *time.Time

	if request.DefaultStartDate != nil {
		if parsed, err := time.Parse(time.RFC3339, *request.DefaultStartDate); err == nil {
			defaultStartDate = &parsed
		} else {
			http.Error(w, fmt.Sprintf("Invalid default_start_date format: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// Default to 1 month ago
		oneMonthAgo := time.Now().AddDate(0, -1, 0)
		defaultStartDate = &oneMonthAgo
	}

	if request.EndDate != nil {
		if parsed, err := time.Parse(time.RFC3339, *request.EndDate); err == nil {
			endDate = &parsed
		} else {
			http.Error(w, fmt.Sprintf("Invalid end_date format: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// Default to now
		now := time.Now()
		endDate = &now
	}

	// Process each mailbox
	var mailboxResults []MailboxSyncResult
	var totalEmailsProcessed int
	var totalNewEmails int
	var messages []string

	for _, mailboxName := range request.Mailboxes {
		result := h.processSingleMailbox(
			*account,
			mailboxName,
			request.SyncMode,
			defaultStartDate,
			endDate,
			request.MaxEmailsPerMailbox,
			request.IncludeBody,
		)

		mailboxResults = append(mailboxResults, result)
		totalEmailsProcessed += result.EmailsProcessed
		totalNewEmails += result.NewEmails

		if result.Error != "" {
			messages = append(messages, fmt.Sprintf("Error in mailbox %s: %s", mailboxName, result.Error))
		}
	}

	processingTime := time.Since(startTime)

	// Log activity
	userID := getUserIDFromContext(r)
	if totalNewEmails > 0 {
		h.activityLogger.LogActivity(
			models.ActivityEmailReceived,
			fmt.Sprintf("收到 %d 封新邮件", totalNewEmails),
			fmt.Sprintf("账户 %s 同步了 %d 封新邮件", account.EmailAddress, totalNewEmails),
			userID,
			map[string]interface{}{
				"sync_mode":       request.SyncMode,
				"mailboxes":       request.Mailboxes,
				"total_processed": totalEmailsProcessed,
				"new_emails":      totalNewEmails,
				"processing_ms":   processingTime.Nanoseconds() / 1000000,
			},
		)
	}

	response := FetchAndStoreResponse{
		Status:               "success",
		SyncMode:             request.SyncMode,
		MailboxResults:       mailboxResults,
		TotalEmailsProcessed: totalEmailsProcessed,
		TotalNewEmails:       totalNewEmails,
		ProcessingTimeMs:     processingTime.Nanoseconds() / 1000000,
		Messages:             messages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// WaitEmailHandler waits for emails to arrive with optional filtering and extraction
// @Summary Wait for emails to arrive with filtering and extraction
// @Description Wait for new emails to arrive for a specific account or email address. Supports timeout, interval checking, and content extraction using the same extractors as the extract-emails endpoint. Only one of accountId or email parameters must be provided.
// @Tags emails
// @Accept json
// @Produce json
// @Param request body WaitEmailRequest true "Request body with account identification and optional extractors"
// @Success 200 {object} WaitEmailResponse "Email found or timeout reached"
// @Failure 400 {string} string "Bad Request - Invalid parameters"
// @Failure 404 {string} string "Not Found - Account not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/wait-email [post]
func (h *APIHandler) WaitEmailHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Parse request body
	var request WaitEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Set default values if not provided
	if request.Timeout <= 0 {
		request.Timeout = 30 // Default 30 seconds
	}
	if request.Interval <= 0 {
		request.Interval = 5 // Default 5 seconds
	}

	// Validate input - exactly one of accountId or email must be provided
	if (request.AccountID == nil && request.Email == nil) || (request.AccountID != nil && request.Email != nil) {
		http.Error(w, "Exactly one of accountId or email must be provided", http.StatusBadRequest)
		return
	}

	// Get account from database
	var account *models.EmailAccount
	var err error

	if request.AccountID != nil {
		account, err = h.EmailAccountRepo.GetByID(*request.AccountID)
	} else {
		// Use GetByEmailOrAlias to handle aliases and domain emails
		account, err = h.EmailAccountRepo.GetByEmailOrAlias(*request.Email)
	}

	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Parse start time
	var filterStartTime time.Time
	if request.StartTime != nil {
		if parsed, err := time.Parse(time.RFC3339, *request.StartTime); err == nil {
			filterStartTime = parsed
		} else {
			http.Error(w, fmt.Sprintf("Invalid start_time format: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		filterStartTime = time.Now() // Default to current time
	}

	// Validate extractors if provided
	var extractorService *services.ExtractorService
	var serviceExtractors []services.ExtractorConfig
	if len(request.Extract) > 0 {
		extractorService = services.NewExtractorService()
		for i, extractor := range request.Extract {
			if extractor.Field == "" || extractor.Type == "" || extractor.Extract == "" {
				http.Error(w, fmt.Sprintf("Extractor %d is missing required fields", i), http.StatusBadRequest)
				return
			}

			// Validate field values
			validFields := map[string]bool{
				"ALL": true, "from": true, "to": true, "cc": true,
				"subject": true, "body": true, "html_body": true, "headers": true,
			}
			if !validFields[extractor.Field] {
				http.Error(w, fmt.Sprintf("Invalid field '%s' in extractor %d", extractor.Field, i), http.StatusBadRequest)
				return
			}

			// Validate type values
			validTypes := map[string]bool{"regex": true, "js": true, "gotemplate": true}
			if !validTypes[extractor.Type] {
				http.Error(w, fmt.Sprintf("Invalid type '%s' in extractor %d", extractor.Type, i), http.StatusBadRequest)
				return
			}

			serviceExtractors = append(serviceExtractors, services.ExtractorConfig{
				Field: services.ExtractorField(extractor.Field),
				Type:  services.ExtractorType(extractor.Type),
				Match: extractor.Match,

				Extract: extractor.Extract,
			})
		}
	}

	// Start waiting for emails
	timeout := time.Duration(request.Timeout) * time.Second
	interval := time.Duration(request.Interval) * time.Second
	deadline := time.Now().Add(timeout)
	checksPerformed := 0

	for time.Now().Before(deadline) {
		checksPerformed++

		// Fetch recent emails from database (not directly from server)
		// This follows the subscription pattern where only the subscription system fetches from server
		options := services.FetchEmailsOptions{
			Mailbox:         "INBOX",
			Limit:           50, // Check recent emails
			Offset:          0,
			StartDate:       &filterStartTime,
			FetchFromServer: false,                    // Use database, not direct server access
			IncludeBody:     len(request.Extract) > 0, // Include body only if extractors are provided
			SortBy:          "date_desc",
		}

		emails, err := h.Fetcher.FetchEmailsWithOptions(*account, options)
		if err != nil {
			log.Printf("Error fetching emails during wait: %v", err)
		} else {
			// Check if any email matches the criteria
			for _, email := range emails {
				// Check if email is newer than start time
				if email.Date.After(filterStartTime) {
					// If no extractors, return the first new email
					if len(serviceExtractors) == 0 {
						elapsed := time.Since(startTime).Seconds()
						response := WaitEmailResponse{
							Status:          "success",
							Found:           true,
							Email:           &email,
							ElapsedTime:     elapsed,
							ChecksPerformed: checksPerformed,
							Message:         "Email found",
						}
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(response)
						return
					}

					// If extractors are provided, check if email matches
					result, err := extractorService.ExtractFromEmail(email, serviceExtractors)
					if err != nil {
						log.Printf("Error extracting from email ID %d: %v", email.ID, err)
						continue
					}

					if result != nil && len(result.Matches) > 0 {
						elapsed := time.Since(startTime).Seconds()
						response := WaitEmailResponse{
							Status:          "success",
							Found:           true,
							Email:           &email,
							Matches:         result.Matches,
							ElapsedTime:     elapsed,
							ChecksPerformed: checksPerformed,
							Message:         "Email found matching extraction criteria",
						}
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(response)
						return
					}
				}
			}
		}

		// Wait for the next check
		if time.Now().Add(interval).Before(deadline) {
			time.Sleep(interval)
		} else {
			break
		}
	}

	// Timeout reached
	elapsed := time.Since(startTime).Seconds()
	response := WaitEmailResponse{
		Status:          "timeout",
		Found:           false,
		ElapsedTime:     elapsed,
		ChecksPerformed: checksPerformed,
		Message:         "Timeout reached, no matching email found",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PollEmailHandler godoc
// @Summary Poll for new emails with optional filtering and extraction (fallback for WebSocket)
// @Description Poll for new emails for a specific account. This is a fallback mechanism when WebSocket is not available.
// It checks for new emails using polling and supports filtering by start time and content extraction.
// @Tags emails
// @Accept json
// @Produce json
// @Param request body WaitEmailWebSocketRequest true "Request body with account identification and optional extractors"
// @Success 200 {object} PollEmailResponse "Response with email status and data"
// @Failure 400 {string} string "Bad Request - Invalid parameters"
// @Failure 404 {string} string "Not Found - Account not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/poll-email [post]
func (h *APIHandler) PollEmailHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Parse request body
	var request WaitEmailWebSocketRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate input - exactly one of accountId or email must be provided
	if (request.AccountID == nil && request.Email == nil) || (request.AccountID != nil && request.Email != nil) {
		http.Error(w, "Exactly one of accountId or email must be provided", http.StatusBadRequest)
		return
	}

	// Get account from database
	var account *models.EmailAccount
	var err error

	if request.AccountID != nil {
		account, err = h.EmailAccountRepo.GetByID(*request.AccountID)
	} else {
		// Use GetByEmailOrAlias to handle aliases and domain emails
		account, err = h.EmailAccountRepo.GetByEmailOrAlias(*request.Email)
	}

	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Set default values if not provided
	if request.Interval <= 0 {
		request.Interval = 5 // Default 5 seconds
	}

	// Parse start time - support both RFC3339 string and Unix timestamp (milliseconds)
	var filterStartTime time.Time
	if request.StartTime != nil {
		// First try to parse as RFC3339
		if parsed, err := time.Parse(time.RFC3339, *request.StartTime); err == nil {
			filterStartTime = parsed.UTC()
		} else {
			// Try to parse as Unix timestamp in milliseconds
			if timestamp, err := time.Parse("2006-01-02T15:04:05.999Z07:00", *request.StartTime); err == nil {
				filterStartTime = timestamp.UTC()
			} else {
				// Try to parse as Unix milliseconds (e.g., "1703980800000")
				var unixMs int64
				if _, err := fmt.Sscanf(*request.StartTime, "%d", &unixMs); err == nil && unixMs > 0 {
					filterStartTime = time.Unix(unixMs/1000, (unixMs%1000)*1e6).UTC()
				} else {
					// Default to current time if parsing fails
					log.Printf("[PollEmail] Failed to parse start time '%s', using current time", *request.StartTime)
					filterStartTime = time.Now().UTC()
				}
			}
		}
	} else {
		filterStartTime = time.Now().UTC()
	}

	// Setup extractor service if needed
	var extractorService *services.ExtractorService
	var serviceExtractors []services.ExtractorConfig
	if len(request.Extract) > 0 {
		extractorService = services.NewExtractorService()
		for _, extractor := range request.Extract {
			serviceExtractors = append(serviceExtractors, services.ExtractorConfig{
				Field:   services.ExtractorField(extractor.Field),
				Type:    services.ExtractorType(extractor.Type),
				Match:   extractor.Match,
				Extract: extractor.Extract,
			})
		}
	}

	// Get "processedIds" from request header or cookie to maintain state between requests
	// This is used to avoid returning the same email multiple times in different poll requests
	processedIDsStr := r.Header.Get("X-Processed-Ids")
	processedMessageIDs := make(map[string]bool)

	if processedIDsStr != "" {
		// Parse comma-separated list of message IDs
		for _, id := range strings.Split(processedIDsStr, ",") {
			if trimmedID := strings.TrimSpace(id); trimmedID != "" {
				processedMessageIDs[trimmedID] = true
			}
		}
	}

	// Fetch emails from multiple mailboxes including spam
	options := services.FetchEmailsOptions{
		Mailbox:         "INBOX",
		Limit:           100,
		Offset:          0,
		StartDate:       &filterStartTime,
		FetchFromServer: true,
		IncludeBody:     true,
		SortBy:          "date_desc",
	}

	log.Printf("[PollEmail] Fetching emails for %s from %s", account.EmailAddress, filterStartTime.Format(time.RFC3339))

	// Use the method that fetches from multiple mailboxes including spam
	emails, err := h.Fetcher.FetchEmailsFromMultipleMailboxes(*account, options)
	if err != nil {
		log.Printf("[PollEmail] Error fetching emails: %v", err)
		http.Error(w, fmt.Sprintf("Error fetching emails: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[PollEmail] Fetched %d total emails for %s from server", len(emails), account.EmailAddress)

	// Response structure
	type PollEmailResponse struct {
		Status       string        `json:"status"`
		Found        bool          `json:"found"`
		Email        *models.Email `json:"email,omitempty"`
		Matches      interface{}   `json:"matches,omitempty"`
		ProcessedIds []string      `json:"processedIds"`
		ElapsedTime  float64       `json:"elapsedTime"`
		Message      string        `json:"message"`
	}

	// Track new processed IDs for this request
	var newProcessedIDs []string
	currentTime := time.Now().UTC()

	// Process emails
	for _, email := range emails {
		// Skip if already processed
		messageKey := email.MessageID
		if messageKey == "" {
			// Use combination of properties as unique identifier
			messageKey = fmt.Sprintf("%s_%s_%s_%d",
				email.Subject,
				email.From,
				email.Date.Format(time.RFC3339Nano),
				email.Size)
		}

		if processedMessageIDs[messageKey] {
			continue
		}

		// Check email date
		emailDateUTC := email.Date.UTC()
		if emailDateUTC.Before(filterStartTime) || emailDateUTC.After(currentTime) {
			continue
		}

		// Check if the email is addressed to the monitored account
		if !isEmailAddressedToAccount(&email, account) {
			continue
		}

		// Add to processed IDs
		newProcessedIDs = append(newProcessedIDs, messageKey)
		processedMessageIDs[messageKey] = true

		// Found a matching email
		if len(serviceExtractors) == 0 {
			// No extractors, return the email
			response := PollEmailResponse{
				Status:       "success",
				Found:        true,
				Email:        &email,
				ProcessedIds: append(newProcessedIDs, getMapKeys(processedMessageIDs)...),
				ElapsedTime:  time.Since(startTime).Seconds(),
				Message:      "Email found",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Check extractors
		result, err := extractorService.ExtractFromEmail(email, serviceExtractors)
		if err != nil {
			log.Printf("[PollEmail] Error extracting from email ID %d: %v", email.ID, err)
			continue
		}

		if result != nil && len(result.Matches) > 0 {
			response := PollEmailResponse{
				Status:       "success",
				Found:        true,
				Email:        &email,
				Matches:      result.Matches,
				ProcessedIds: append(newProcessedIDs, getMapKeys(processedMessageIDs)...),
				ElapsedTime:  time.Since(startTime).Seconds(),
				Message:      "Email found matching extraction criteria",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// No matching email found
	response := PollEmailResponse{
		Status:       "not_found",
		Found:        false,
		ProcessedIds: append(newProcessedIDs, getMapKeys(processedMessageIDs)...),
		ElapsedTime:  time.Since(startTime).Seconds(),
		Message:      "No matching email found in this poll. Continue polling.",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to get map keys as slice
func getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// RandomEmailHandler godoc
// @Summary Get a random email account
// @Description Get a random email account from existing accounts. Supports generating random Gmail aliases and selecting domain email accounts based on parameters.
// @Tags emails
// @Accept json
// @Produce json
// @Param alias query bool false "Allow random alias emails (Gmail aliases)"
// @Param domain query bool false "Allow domain emails"
// @Success 200 {object} RandomEmailResponse "Successful response with random email account"
// @Failure 404 {string} string "Not Found - No email accounts available"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/random-email [get]
func (h *APIHandler) RandomEmailHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	aliasParam := r.URL.Query().Get("alias")
	domainParam := r.URL.Query().Get("domain")

	allowAlias := aliasParam == "true"
	allowDomain := domainParam == "true"

	if allowAlias {
		// Check availability of different account types
		var err error
		allowAlias, err = h.EmailAccountRepo.HasGmailAccounts()
		if err != nil {
			http.Error(w, "Error checking Gmail accounts", http.StatusInternalServerError)
			return
		}
	}

	if allowDomain {
		var err error
		allowDomain, err = h.EmailAccountRepo.HasDomainAccounts()
		if err != nil {
			http.Error(w, "Error checking domain accounts", http.StatusInternalServerError)
			return
		}
	}

	// If neither alias nor domain is specified, return a random account
	if !allowAlias && !allowDomain {
		account, err := h.EmailAccountRepo.GetRandomAccount()
		if err != nil {
			http.Error(w, "No email accounts found", http.StatusNotFound)
			return
		}

		response := RandomEmailResponse{
			Status:    "success",
			EmailType: "regular",
			RawEmail:  account.EmailAddress,
			Email:     account.EmailAddress,
			Message:   "Random email account selected",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Determine what types are available and requested
	// Randomly choose between available options
	var useAlias bool
	if allowAlias && allowDomain {
		// Both options available, randomly choose
		choice, _ := rand.Int(rand.Reader, big.NewInt(2))
		useAlias = choice.Int64() == 0
	} else {
		// Only one option available
		useAlias = allowAlias
	}

	if useAlias {
		// Generate Gmail alias
		account, err := h.EmailAccountRepo.GetRandomGmailAccount()
		if err != nil {
			http.Error(w, "No Gmail accounts found", http.StatusNotFound)
			return
		}

		// Generate random alias
		generatedEmail, err := h.generateGmailAlias(account.EmailAddress)
		if err != nil {
			http.Error(w, "Error generating Gmail alias", http.StatusInternalServerError)
			return
		}

		response := RandomEmailResponse{
			Status:    "success",
			EmailType: "alias",
			RawEmail:  account.EmailAddress,
			Email:     generatedEmail,
			Message:   "Generated random Gmail alias",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	} else {
		// Use domain email
		account, err := h.EmailAccountRepo.GetRandomDomainAccount()
		if err != nil {
			http.Error(w, "No domain email accounts found", http.StatusNotFound)
			return
		}

		// Generate random domain email
		generatedEmail, err := h.generateDomainEmail(account.Domain)
		if err != nil {
			http.Error(w, "Error generating domain email", http.StatusInternalServerError)
			return
		}

		response := RandomEmailResponse{
			Status:    "success",
			EmailType: "domain",
			Email:     generatedEmail,
			RawEmail:  account.EmailAddress,
			Message:   "Generated random domain email",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
}

// generateGmailAlias generates a random Gmail alias
func (h *APIHandler) generateGmailAlias(originalEmail string) (string, error) {
	// Extract the local part and domain
	parts := strings.Split(originalEmail, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid email format")
	}

	localPart := parts[0]
	domain := parts[1]

	// Generate random suffix
	randomNum, err := rand.Int(rand.Reader, big.NewInt(999999))
	if err != nil {
		return "", err
	}

	// Create alias with + notation
	alias := fmt.Sprintf("%s+random%06d@%s", localPart, randomNum.Int64(), domain)
	return alias, nil
}

// generateDomainEmail generates a random email for a domain
func (h *APIHandler) generateDomainEmail(domain string) (string, error) {
	// Generate random local part
	randomNum, err := rand.Int(rand.Reader, big.NewInt(999999))
	if err != nil {
		return "", err
	}

	// Generate random string prefix
	prefixes := []string{"user", "mail", "temp", "random", "test", "inbox"}
	prefixIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(prefixes))))
	if err != nil {
		return "", err
	}

	localPart := fmt.Sprintf("%s%06d", prefixes[prefixIndex.Int64()], randomNum.Int64())
	return fmt.Sprintf("%s@%s", localPart, domain), nil
}

// GetEmailDomainsHandler returns all unique email domains from accounts
// @Summary Get all email domains
// @Description Get all unique email domains from registered email accounts
// @Tags emails
// @Accept json
// @Produce json
// @Success 200 {object} EmailDomainsResponse "List of email domains"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/email-domains [get]
func (h *APIHandler) GetEmailDomainsHandler(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.EmailAccountRepo.GetAll()
	if err != nil {
		http.Error(w, "Error fetching accounts", http.StatusInternalServerError)
		return
	}

	// Extract unique domains
	domainMap := make(map[string]bool)
	for _, account := range accounts {
		if account.Domain != "" {
			domainMap[account.Domain] = true
		} else {
			// Extract domain from email address
			parts := strings.Split(account.EmailAddress, "@")
			if len(parts) == 2 {
				domainMap[parts[1]] = true
			}
		}
	}

	// Convert map to slice
	domains := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		domains = append(domains, domain)
	}

	// Sort domains
	sort.Strings(domains)

	response := EmailDomainsResponse{
		Status:  "success",
		Domains: domains,
		Count:   len(domains),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetIncrementalSyncRecordsHandler retrieves incremental sync records for an account
// @Summary Get incremental sync records for an account
// @Description Get all incremental sync records for a specific account, showing last sync times per mailbox
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Success 200 {array} models.IncrementalSyncRecord "List of incremental sync records"
// @Failure 400 {string} string "Bad Request - Invalid account ID"
// @Failure 404 {string} string "Not Found - Account not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/accounts/{id}/sync-records [get]
func (h *APIHandler) GetIncrementalSyncRecordsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}
	accountID := uint(id)

	// Verify account exists
	_, err = h.EmailAccountRepo.GetByID(accountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Get sync records
	records, err := h.IncrementalSyncRepo.GetAllByAccount(accountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// GetLastSyncRecordHandler retrieves the last sync record for an account
// @Summary Get the last sync record for an account
// @Description Get the most recent sync record for a specific account across all mailboxes
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Success 200 {object} models.IncrementalSyncRecord "Last sync record"
// @Failure 400 {string} string "Bad Request - Invalid account ID"
// @Failure 404 {string} string "Not Found - Account not found or no sync records"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/accounts/{id}/last-sync-record [get]
func (h *APIHandler) GetLastSyncRecordHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}
	accountID := uint(id)

	// Verify account exists
	_, err = h.EmailAccountRepo.GetByID(accountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Get all sync records for the account
	records, err := h.IncrementalSyncRepo.GetAllByAccount(accountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(records) == 0 {
		http.Error(w, "No sync records found", http.StatusNotFound)
		return
	}

	// Find the most recent sync record
	var lastRecord *models.IncrementalSyncRecord
	for i := range records {
		if lastRecord == nil || records[i].LastSyncEndTime.After(lastRecord.LastSyncEndTime) {
			lastRecord = &records[i]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lastRecord)
}

// DeleteIncrementalSyncRecordHandler deletes an incremental sync record
// @Summary Delete an incremental sync record
// @Description Delete an incremental sync record for a specific account and mailbox (forces full sync on next fetch)
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Param mailbox query string true "Mailbox name"
// @Success 204 "No Content - Record deleted successfully"
// @Failure 400 {string} string "Bad Request - Invalid account ID or missing mailbox parameter"
// @Failure 404 {string} string "Not Found - Account or sync record not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/accounts/{id}/sync-records [delete]
func (h *APIHandler) DeleteIncrementalSyncRecordHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}
	accountID := uint(id)

	mailboxName := r.URL.Query().Get("mailbox")
	if mailboxName == "" {
		http.Error(w, "Mailbox parameter is required", http.StatusBadRequest)
		return
	}

	// Verify account exists
	_, err = h.EmailAccountRepo.GetByID(accountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Delete sync record
	err = h.IncrementalSyncRepo.Delete(accountID, mailboxName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// processSingleMailbox handles the sync process for a single mailbox
func (h *APIHandler) processSingleMailbox(
	account models.EmailAccount,
	mailboxName string,
	syncMode string,
	defaultStartDate *time.Time,
	endDate *time.Time,
	maxEmails int,
	includeBody bool,
) MailboxSyncResult {
	syncStartTime := time.Now()

	result := MailboxSyncResult{
		MailboxName:   mailboxName,
		SyncStartTime: syncStartTime,
		SyncEndTime:   syncStartTime, // Will be updated at the end
	}

	// Determine sync date range
	var startDate *time.Time

	if syncMode == "incremental" {
		// Try to get previous sync record
		syncRecord, err := h.IncrementalSyncRepo.GetByAccountAndMailbox(account.ID, mailboxName)
		if err == nil {
			// Use previous sync end time as start time
			startDate = &syncRecord.LastSyncEndTime
			result.PreviousSyncEndTime = &syncRecord.LastSyncEndTime
		} else {
			// No previous sync record, use default start date
			startDate = defaultStartDate
			log.Printf("No previous sync record found for account %d mailbox %s, using default start date", account.ID, mailboxName)
		}
	} else {
		// Full sync mode - use default start date or nil for all emails
		startDate = defaultStartDate
	}

	// Prepare fetch options
	options := services.FetchEmailsOptions{
		Mailbox:         mailboxName,
		Limit:           maxEmails,
		Offset:          0,
		StartDate:       startDate,
		EndDate:         endDate,
		FetchFromServer: true,
		IncludeBody:     includeBody,
		SortBy:          "date_desc",
	}

	// Fetch emails from server
	emails, err := h.Fetcher.FetchEmailsWithOptions(account, options)
	if err != nil {
		result.Error = err.Error()
		result.SyncEndTime = time.Now()
		return result
	}

	result.EmailsProcessed = len(emails)

	// Check for duplicates and store new emails
	var newEmails []models.Email
	for _, email := range emails {
		if email.MessageID != "" {
			exists, err := h.EmailRepo.CheckDuplicate(email.MessageID, account.ID)
			if err != nil {
				log.Printf("Error checking duplicate for message %s: %v", email.MessageID, err)
				continue
			}
			if exists {
				continue
			}
		}
		newEmails = append(newEmails, email)
	}

	result.NewEmails = len(newEmails)

	// Store new emails
	if len(newEmails) > 0 {
		if err := h.EmailRepo.CreateBatch(newEmails); err != nil {
			result.Error = fmt.Sprintf("Failed to store emails: %v", err)
			result.SyncEndTime = time.Now()
			return result
		}
		log.Printf("Stored %d new emails for account %d mailbox %s", len(newEmails), account.ID, mailboxName)
	}

	result.SyncEndTime = time.Now()

	// Update or create incremental sync record
	if syncMode == "incremental" {
		syncRecord := &models.IncrementalSyncRecord{
			AccountID:         account.ID,
			MailboxName:       mailboxName,
			LastSyncStartTime: syncStartTime,
			LastSyncEndTime:   result.SyncEndTime,
			EmailsProcessed:   result.EmailsProcessed,
		}

		if err := h.IncrementalSyncRepo.CreateOrUpdate(syncRecord); err != nil {
			log.Printf("Failed to update incremental sync record: %v", err)
			// Don't fail the entire operation for this
		}
	}

	return result
}

// GetEmailsHandler retrieves emails for an account with advanced search capabilities
// @Summary Get emails for an account with advanced search
// @Description Get emails for an account with support for date range, text search, and keyword filtering
// @Tags account-emails
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Param limit query int false "Limit (default: 50, max: 100)"
// @Param offset query int false "Offset for pagination (default: 0)"
// @Param sort_by query string false "Sort order: date_desc, date_asc, subject_asc, subject_desc (default: date_desc)"
// @Param start_date query string false "Start date for filtering (RFC3339 format)"
// @Param end_date query string false "End date for filtering (RFC3339 format)"
// @Param from_query query string false "Search in From field (fuzzy match)"
// @Param to_query query string false "Search in To field (fuzzy match)"
// @Param cc_query query string false "Search in CC field (fuzzy match)"
// @Param subject_query query string false "Search in Subject field (fuzzy match)"
// @Param body_query query string false "Search in email body (fuzzy match)"
// @Param html_query query string false "Search in HTML body (fuzzy match)"
// @Param keyword query string false "Global keyword search across all text fields"
// @Param mailbox query string false "Filter by mailbox name"
// @Success 200 {object} map[string]interface{} "Response with emails array and pagination info"
// @Router /api/account-emails/list/{id} [get]
func (h *APIHandler) GetEmailsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	options := repository.EmailSearchOptions{
		AccountID: uint(id),
		Limit:     50,
		Offset:    0,
		SortBy:    "date DESC",
	}

	// Parse limit
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			if parsedLimit > 100 {
				parsedLimit = 100 // Cap at 100
			}
			options.Limit = parsedLimit
		}
	}

	// Parse offset
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			options.Offset = parsedOffset
		}
	}

	// Parse sort order
	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		validSortOptions := map[string]string{
			"date_desc":    "date DESC",
			"date_asc":     "date ASC",
			"subject_asc":  "subject ASC",
			"subject_desc": "subject DESC",
		}
		if validSort, exists := validSortOptions[sortBy]; exists {
			options.SortBy = validSort
		}
	}

	// Parse date range
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if parsed, err := time.Parse(time.RFC3339, startDate); err == nil {
			options.StartDate = &parsed
		}
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if parsed, err := time.Parse(time.RFC3339, endDate); err == nil {
			options.EndDate = &parsed
		}
	}

	// Parse text search parameters
	options.FromQuery = r.URL.Query().Get("from_query")
	options.ToQuery = r.URL.Query().Get("to_query")
	options.CcQuery = r.URL.Query().Get("cc_query")
	options.SubjectQuery = r.URL.Query().Get("subject_query")
	options.BodyQuery = r.URL.Query().Get("body_query")
	options.HTMLQuery = r.URL.Query().Get("html_query")
	options.Keyword = r.URL.Query().Get("keyword")
	options.MailboxName = r.URL.Query().Get("mailbox")

	// 如果有to_query参数，先尝试立即同步对应账户的邮件
	if options.ToQuery != "" {
		if err := h.syncEmailsForToQuery(options.ToQuery); err != nil {
			// 记录错误但不阻止搜索，因为可能数据库中已有部分邮件
			// 使用标准log包记录错误
			log.Printf("Failed to sync emails for to_query %s: %v", options.ToQuery, err)
		}
	}

	// Perform search
	emails, totalCount, err := h.EmailRepo.SearchEmails(options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate pagination info
	totalPages := int((totalCount + int64(options.Limit) - 1) / int64(options.Limit))
	currentPage := (options.Offset / options.Limit) + 1
	hasNext := options.Offset+options.Limit < int(totalCount)
	hasPrev := options.Offset > 0

	response := map[string]interface{}{
		"emails": emails,
		"pagination": map[string]interface{}{
			"total":        totalCount,
			"total_pages":  totalPages,
			"current_page": currentPage,
			"limit":        options.Limit,
			"offset":       options.Offset,
			"has_next":     hasNext,
			"has_prev":     hasPrev,
		},
		"search_criteria": map[string]interface{}{
			"account_id":    options.AccountID,
			"start_date":    options.StartDate,
			"end_date":      options.EndDate,
			"from_query":    options.FromQuery,
			"to_query":      options.ToQuery,
			"cc_query":      options.CcQuery,
			"subject_query": options.SubjectQuery,
			"body_query":    options.BodyQuery,
			"html_query":    options.HTMLQuery,
			"keyword":       options.Keyword,
			"mailbox":       options.MailboxName,
			"sort_by":       options.SortBy,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SearchEmailsHandler searches emails with optional account filtering
// @Summary Search emails with optional account ID
// @Description Search emails with optional account filtering and to_query parameter
// @Tags emails
// @Accept json
// @Produce json
// @Param account_id query int false "Account ID (optional)"
// @Param to_query query string false "Filter by recipient email"
// @Param from_query query string false "Filter by sender email"
// @Param limit query int false "Limit results (default 50, max 100)"
// @Param offset query int false "Offset for pagination"
// @Param sort_by query string false "Sort order (date_desc, date_asc, etc.)"
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param subject_query query string false "Filter by subject"
// @Param body_query query string false "Filter by email body"
// @Param html_query query string false "Filter by HTML body"
// @Param keyword query string false "Global search across all fields"
// @Param mailbox query string false "Filter by mailbox name"
// @Success 200 {object} map[string]interface{} "Response with emails array and pagination info"
// @Router /api/emails/search [get]
func (h *APIHandler) SearchEmailsHandler(w http.ResponseWriter, r *http.Request) {
	// Create search options
	options := repository.EmailSearchOptions{
		Limit:  50,
		Offset: 0,
		SortBy: "date DESC",
	}

	// Try to get optional account ID from query parameters
	if accountID := r.URL.Query().Get("account_id"); accountID != "" {
		if id, err := strconv.ParseUint(accountID, 10, 32); err == nil {
			options.AccountID = uint(id)
		}
	}

	// Parse limit
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			if parsedLimit > 100 {
				parsedLimit = 100 // Cap at 100
			}
			options.Limit = parsedLimit
		}
	}

	// Parse offset
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			options.Offset = parsedOffset
		}
	}

	// Parse sort order
	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		validSortOptions := map[string]string{
			"date_desc":    "date DESC",
			"date_asc":     "date ASC",
			"subject_asc":  "subject ASC",
			"subject_desc": "subject DESC",
		}
		if validSort, exists := validSortOptions[sortBy]; exists {
			options.SortBy = validSort
		}
	}

	// Parse date range
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if parsed, err := time.Parse(time.RFC3339, startDate); err == nil {
			options.StartDate = &parsed
		}
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if parsed, err := time.Parse(time.RFC3339, endDate); err == nil {
			options.EndDate = &parsed
		}
	}

	// Parse text search parameters
	options.FromQuery = r.URL.Query().Get("from_query")
	options.ToQuery = r.URL.Query().Get("to_query")
	options.CcQuery = r.URL.Query().Get("cc_query")
	options.SubjectQuery = r.URL.Query().Get("subject_query")
	options.BodyQuery = r.URL.Query().Get("body_query")
	options.HTMLQuery = r.URL.Query().Get("html_query")
	options.Keyword = r.URL.Query().Get("keyword")
	options.MailboxName = r.URL.Query().Get("mailbox")

	// Perform search
	emails, totalCount, err := h.EmailRepo.SearchEmails(options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate pagination info
	totalPages := int((totalCount + int64(options.Limit) - 1) / int64(options.Limit))
	currentPage := (options.Offset / options.Limit) + 1
	hasNext := options.Offset+options.Limit < int(totalCount)
	hasPrev := options.Offset > 0

	response := map[string]interface{}{
		"emails": emails,
		"pagination": map[string]interface{}{
			"total":        totalCount,
			"total_pages":  totalPages,
			"current_page": currentPage,
			"limit":        options.Limit,
			"offset":       options.Offset,
			"has_next":     hasNext,
			"has_prev":     hasPrev,
		},
		"search_criteria": map[string]interface{}{
			"account_id":    options.AccountID,
			"start_date":    options.StartDate,
			"end_date":      options.EndDate,
			"from_query":    options.FromQuery,
			"to_query":      options.ToQuery,
			"cc_query":      options.CcQuery,
			"subject_query": options.SubjectQuery,
			"body_query":    options.BodyQuery,
			"html_query":    options.HTMLQuery,
			"keyword":       options.Keyword,
			"mailbox":       options.MailboxName,
			"sort_by":       options.SortBy,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEmailHandler retrieves a specific email
// @Summary Get an email by ID
// @Description Get an email by ID
// @Tags emails
// @Accept json
// @Produce json
// @Param id path int true "Email ID"
// @Success 200 {object} models.Email
// @Router /api/emails/{id} [get]
func (h *APIHandler) GetEmailHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	email, err := h.EmailRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(email)
}

// ExtractEmailsHandler handles email content extraction with advanced filtering
// @Summary Extract content from emails with advanced filtering and processing
// @Description Extract specific content from emails using regex, JavaScript, or Go templates with comprehensive search and filtering capabilities. Supports both account-specific and global extraction.
// @Tags account-emails,emails
// @Accept json
// @Produce json
// @Param id path int false "Account ID (only for /account-emails/extract/{id} endpoint)"
// @Param request body ExtractEmailsRequest true "Extraction request with search criteria and extractors"
// @Success 200 {object} ExtractEmailsResponse "Extraction results with matches and statistics"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/account-emails/extract/{id} [post]
// @Router /api/emails/extract [post]
func (h *APIHandler) ExtractEmailsHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Parse account ID from URL (optional for global endpoint)
	vars := mux.Vars(r)
	var accountID uint
	if idStr, exists := vars["id"]; exists && idStr != "" {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			http.Error(w, "Invalid account ID", http.StatusBadRequest)
			return
		}
		accountID = uint(id)
	}

	// Parse request body
	var req ExtractEmailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get extractors from template if ExtractorID is provided
	var templateExtractors []ExtractorConfig
	if req.ExtractorID != nil {
		templateRepo := repository.NewExtractorTemplateRepository(database.GetDB())
		template, err := templateRepo.GetByID(*req.ExtractorID)
		if err != nil {
			http.Error(w, "Invalid extractor template ID: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Convert template extractors to API extractors
		for _, extractor := range template.Extractors {
			templateExtractors = append(templateExtractors, ExtractorConfig{
				Field: extractor.Field,
				Type:  extractor.Type,
				Match: extractor.Match,

				Extract: extractor.Extract,
			})
		}
	}

	// Merge template extractors with request extractors
	allExtractors := append(templateExtractors, req.Extractors...)

	// Validate extractors
	if len(allExtractors) == 0 {
		http.Error(w, "At least one extractor must be provided (either directly or via template)", http.StatusBadRequest)
		return
	}

	// Validate extractor configurations
	for i, extractor := range allExtractors {
		if extractor.Field == "" || extractor.Type == "" || extractor.Extract == "" {
			http.Error(w, fmt.Sprintf("Extractor %d is missing required fields", i), http.StatusBadRequest)
			return
		}

		// Validate field values
		validFields := map[string]bool{
			"ALL": true, "from": true, "to": true, "cc": true,
			"subject": true, "body": true, "html_body": true, "headers": true,
		}
		if !validFields[extractor.Field] {
			http.Error(w, fmt.Sprintf("Invalid field '%s' in extractor %d", extractor.Field, i), http.StatusBadRequest)
			return
		}

		// Validate type values
		validTypes := map[string]bool{"regex": true, "js": true, "gotemplate": true}
		if !validTypes[extractor.Type] {
			http.Error(w, fmt.Sprintf("Invalid type '%s' in extractor %d", extractor.Type, i), http.StatusBadRequest)
			return
		}
	}

	// Convert API request to repository search options
	options := repository.EmailSearchOptions{
		AccountID:    accountID, // Use the parsed account ID from URL path (0 for global search)
		Limit:        req.Limit,
		Offset:       req.Offset,
		SortBy:       req.SortBy,
		FromQuery:    req.FromQuery,
		ToQuery:      req.ToQuery,
		CcQuery:      req.CcQuery,
		SubjectQuery: req.SubjectQuery,
		BodyQuery:    req.BodyQuery,
		HTMLQuery:    req.HTMLQuery,
		Keyword:      req.Keyword,
		MailboxName:  req.MailboxName,
	}

	// Parse date filters
	if req.StartDate != nil {
		if parsed, err := time.Parse(time.RFC3339, *req.StartDate); err == nil {
			options.StartDate = &parsed
		}
	}
	if req.EndDate != nil {
		if parsed, err := time.Parse(time.RFC3339, *req.EndDate); err == nil {
			options.EndDate = &parsed
		}
	}

	// Set default batch size
	batchSize := req.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	// Create extractor service
	extractorService := services.NewExtractorService()

	// Convert API extractors to service extractors
	var serviceExtractors []services.ExtractorConfig
	for _, apiExtractor := range allExtractors {
		serviceExtractors = append(serviceExtractors, services.ExtractorConfig{
			Field: services.ExtractorField(apiExtractor.Field),
			Type:  services.ExtractorType(apiExtractor.Type),
			Match: apiExtractor.Match,

			Extract: apiExtractor.Extract,
		})
	}

	// Create cursor for streaming processing
	cursor := h.EmailRepo.NewEmailCursor(options, batchSize)
	defer cursor.Close()

	// Process emails in batches
	var results []ExtractorResult
	var totalProcessed int
	var totalMatched int
	var batchesProcessed int
	var extractorStats []ExtractorStats

	// Initialize extractor statistics
	for _, extractor := range allExtractors {
		extractorStats = append(extractorStats, ExtractorStats{
			Config:             extractor,
			MatchCount:         0,
			TotalMatches:       0,
			AvgMatchesPerEmail: 0,
		})
	}

	// Process emails in batches using cursor
	for {
		emails, err := cursor.Next()
		if err != nil {
			http.Error(w, "Error processing emails: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(emails) == 0 {
			break // No more emails
		}

		batchesProcessed++
		totalProcessed += len(emails)

		// Process each email in the batch
		for _, email := range emails {
			result, err := extractorService.ExtractFromEmail(email, serviceExtractors)
			if err != nil {
				log.Printf("Error extracting from email ID %d: %v", email.ID, err)
				continue
			}

			if result != nil {
				results = append(results, ExtractorResult{
					Email:   result.Email,
					Matches: result.Matches,
				})
				totalMatched++

				// Update extractor statistics
				for i, extractor := range serviceExtractors {
					extractorResult, err := extractorService.ExtractFromEmail(email, []services.ExtractorConfig{extractor})
					if err == nil && extractorResult != nil && len(extractorResult.Matches) > 0 {
						extractorStats[i].MatchCount++
						extractorStats[i].TotalMatches += len(extractorResult.Matches)
					}
				}
			}
		}
	}

	// Calculate final statistics
	processingTime := time.Since(startTime)
	avgTimePerEmail := float64(processingTime.Nanoseconds()) / float64(totalProcessed) / 1000000 // Convert to milliseconds

	for i := range extractorStats {
		if extractorStats[i].MatchCount > 0 {
			extractorStats[i].AvgMatchesPerEmail = float64(extractorStats[i].TotalMatches) / float64(extractorStats[i].MatchCount)
		}
	}

	// Build response
	response := ExtractEmailsResponse{
		Results:        results,
		TotalProcessed: totalProcessed,
		TotalMatched:   totalMatched,
		Summary: ExtractSummary{
			ProcessingTimeMs: processingTime.Nanoseconds() / 1000000,
			BatchesProcessed: batchesProcessed,
			AvgTimePerEmail:  avgTimePerEmail,
			ExtractorStats:   extractorStats,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckEmailHandler checks for new emails once (for frontend polling)
// @Summary Check for new emails once
// @Description Check for new emails for a specific email address. Uses intelligent email resolution to handle Gmail aliases, domain emails, and real email addresses. Returns immediately without polling.
// @Tags emails
// @Accept json
// @Produce json
// @Param request body CheckEmailRequest true "Request body with email and optional extractors"
// @Success 200 {object} CheckEmailResponse "Email check result"
// @Failure 400 {string} string "Bad Request - Invalid parameters"
// @Failure 404 {string} string "Not Found - Email address cannot be resolved to any account"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/check-email [post]
func (h *APIHandler) CheckEmailHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var request CheckEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate input
	if request.Email == "" {
		http.Error(w, "Email address is required", http.StatusBadRequest)
		return
	}

	// Try to resolve email to account using intelligent resolution
	account, err := h.EmailAccountRepo.GetByEmailOrAlias(request.Email)
	if err != nil {
		// Return detailed error information
		response := CheckEmailResponse{
			Status:  "error",
			Found:   false,
			Error:   fmt.Sprintf("无法解析邮箱 %s 到任何已配置的账户。可能原因：1) 邮箱不存在于系统中 2) 需要配置域名邮箱 3) Gmail别名未正确配置", request.Email),
			Message: "Email address resolution failed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Parse start time
	var filterStartTime time.Time
	if request.StartTime != nil {
		if parsed, err := time.Parse(time.RFC3339, *request.StartTime); err == nil {
			filterStartTime = parsed
		} else {
			response := CheckEmailResponse{
				Status:  "error",
				Found:   false,
				Error:   fmt.Sprintf("Invalid start_time format: %v", err),
				Message: "Invalid start_time format",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
	} else {
		// Default to 1 hour ago to catch recent emails
		filterStartTime = time.Now().Add(-1 * time.Hour)
	}

	// Validate extractors if provided
	var extractorService *services.ExtractorService
	var serviceExtractors []services.ExtractorConfig
	if len(request.Extract) > 0 {
		extractorService = services.NewExtractorService()
		for i, extractor := range request.Extract {
			if extractor.Field == "" || extractor.Type == "" || extractor.Extract == "" {
				response := CheckEmailResponse{
					Status:  "error",
					Found:   false,
					Error:   fmt.Sprintf("Extractor %d is missing required fields", i),
					Message: "Invalid extractor configuration",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(response)
				return
			}

			// Validate field values
			validFields := map[string]bool{
				"ALL": true, "from": true, "to": true, "cc": true,
				"subject": true, "body": true, "html_body": true, "headers": true,
			}
			if !validFields[extractor.Field] {
				response := CheckEmailResponse{
					Status:  "error",
					Found:   false,
					Error:   fmt.Sprintf("Invalid field '%s' in extractor %d", extractor.Field, i),
					Message: "Invalid extractor field",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(response)
				return
			}

			// Validate type values
			validTypes := map[string]bool{"regex": true, "js": true, "gotemplate": true}
			if !validTypes[extractor.Type] {
				response := CheckEmailResponse{
					Status:  "error",
					Found:   false,
					Error:   fmt.Sprintf("Invalid type '%s' in extractor %d", extractor.Type, i),
					Message: "Invalid extractor type",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(response)
				return
			}

			serviceExtractors = append(serviceExtractors, services.ExtractorConfig{
				Field:   services.ExtractorField(extractor.Field),
				Type:    services.ExtractorType(extractor.Type),
				Match:   extractor.Match,
				Extract: extractor.Extract,
			})
		}
	}

	// Check for new emails from database (single check, no polling)
	// This follows the subscription pattern where only the subscription system fetches from server
	options := services.FetchEmailsOptions{
		Mailbox:         "INBOX",
		Limit:           50, // Check recent emails
		Offset:          0,
		StartDate:       &filterStartTime,
		FetchFromServer: false,                    // Use database, not direct server access
		IncludeBody:     len(request.Extract) > 0, // Include body only if extractors are provided
		SortBy:          "date_desc",
	}

	emails, err := h.Fetcher.FetchEmailsWithOptions(*account, options)
	if err != nil {
		log.Printf("Error fetching emails during check: %v", err)
		response := CheckEmailResponse{
			Status:  "error",
			Found:   false,
			Error:   fmt.Sprintf("Failed to fetch emails: %v", err),
			Message: "Email fetch failed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if any email matches the criteria
	for _, email := range emails {
		// If no extractors, return the first email (already filtered by FetchEmailsWithOptions)
		if len(serviceExtractors) == 0 {
			response := CheckEmailResponse{
				Status:  "success",
				Found:   true,
				Email:   &email,
				Message: "Email found",
				ResolvedAccount: &AccountInfo{
					ID:           account.ID,
					EmailAddress: account.EmailAddress,
					IsDomainMail: account.IsDomainMail,
					Domain:       account.Domain,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// If extractors are provided, check if email matches
		result, err := extractorService.ExtractFromEmail(email, serviceExtractors)
		if err != nil {
			log.Printf("Error extracting from email ID %d: %v", email.ID, err)
			continue
		}

		if result != nil && len(result.Matches) > 0 {
			response := CheckEmailResponse{
				Status:  "success",
				Found:   true,
				Email:   &email,
				Matches: result.Matches,
				Message: "Email found with matching extractors",
				ResolvedAccount: &AccountInfo{
					ID:           account.ID,
					EmailAddress: account.EmailAddress,
					IsDomainMail: account.IsDomainMail,
					Domain:       account.Domain,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// No matching email found
	response := CheckEmailResponse{
		Status:  "success",
		Found:   false,
		Message: "No new emails found",
		ResolvedAccount: &AccountInfo{
			ID:           account.ID,
			EmailAddress: account.EmailAddress,
			IsDomainMail: account.IsDomainMail,
			Domain:       account.Domain,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// EmailStatsResponse represents the response structure for email statistics
type EmailStatsResponse struct {
	TotalEmails     int64   `json:"totalEmails"`
	UnreadEmails    int64   `json:"unreadEmails"`
	TodayEmails     int64   `json:"todayEmails"`
	TotalGrowthRate float64 `json:"totalGrowthRate"` // 总邮件增长率 (今日vs昨日24:00)
	TodayGrowthRate float64 `json:"todayGrowthRate"` // 今日邮件增长率 (今日vs昨日)
}

// GetEmailStatsHandler godoc
// @Summary Get email statistics for dashboard
// @Description Get comprehensive email statistics including total, unread, today's emails and growth rates
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} EmailStatsResponse
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/dashboard/stats [get]
func (h *APIHandler) GetEmailStatsHandler(w http.ResponseWriter, r *http.Request) {
	// 获取总邮件数
	totalEmails, err := h.EmailRepo.GetTotalCount()
	if err != nil {
		totalEmails = 0
	}

	// 获取未读邮件数
	unreadEmails, err := h.EmailRepo.GetTotalUnreadCount()
	if err != nil {
		unreadEmails = 0
	}

	// 获取今日邮件数
	todayEmails, err := h.EmailRepo.GetTodayEmailCount()
	if err != nil {
		todayEmails = 0
	}

	// 获取昨日邮件数（用于计算今日增长率）
	yesterdayEmails, err := h.EmailRepo.GetYesterdayEmailCount()
	if err != nil {
		yesterdayEmails = 0
	}

	// 获取截至昨日24:00的总邮件数（用于计算总增长率）
	emailsUntilYesterday, err := h.EmailRepo.GetEmailCountUntilYesterday()
	if err != nil {
		emailsUntilYesterday = 0
	}

	// 计算总邮件增长率：(今日总数 - 昨日24:00总数) / 昨日24:00总数 * 100
	var totalGrowthRate float64 = 0
	if emailsUntilYesterday > 0 {
		growth := totalEmails - emailsUntilYesterday
		totalGrowthRate = float64(growth) / float64(emailsUntilYesterday) * 100
	} else if totalEmails > 0 {
		totalGrowthRate = 100 // 如果昨日没有邮件，今日有邮件，增长率为100%
	}

	// 计算今日邮件增长率：(今日邮件数 - 昨日邮件数) / 昨日邮件数 * 100
	var todayGrowthRate float64 = 0
	if yesterdayEmails > 0 {
		growth := todayEmails - yesterdayEmails
		todayGrowthRate = float64(growth) / float64(yesterdayEmails) * 100
	} else if todayEmails > 0 {
		todayGrowthRate = 100 // 如果昨日没有邮件，今日有邮件，增长率为100%
	}

	response := EmailStatsResponse{
		TotalEmails:     totalEmails,
		UnreadEmails:    unreadEmails,
		TodayEmails:     todayEmails,
		TotalGrowthRate: totalGrowthRate,
		TodayGrowthRate: todayGrowthRate,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAllEmailsHandler retrieves all emails from all accounts with advanced search capabilities
// @Summary Get all emails with advanced search
// @Description Get all emails across all accounts with support for date range, text search, and keyword filtering
// @Tags account-emails
// @Accept json
// @Produce json
// @Param limit query int false "Limit (default: 50, max: 100)"
// @Param offset query int false "Offset for pagination (default: 0)"
// @Param sort_by query string false "Sort order: date_desc, date_asc, subject_asc, subject_desc (default: date_desc)"
// @Param start_date query string false "Start date for filtering (RFC3339 format)"
// @Param end_date query string false "End date for filtering (RFC3339 format)"
// @Param from_query query string false "Search in From field (fuzzy match)"
// @Param to_query query string false "Search in To field (fuzzy match)"
// @Param cc_query query string false "Search in CC field (fuzzy match)"
// @Param subject_query query string false "Search in Subject field (fuzzy match)"
// @Param body_query query string false "Search in email body (fuzzy match)"
// @Param html_query query string false "Search in HTML body (fuzzy match)"
// @Param keyword query string false "Global keyword search across all text fields"
// @Param mailbox query string false "Filter by mailbox name (comma-separated for multiple)"
// @Success 200 {object} map[string]interface{} "Response with emails array and pagination info"
// @Router /api/account-emails/list/all [get]
func (h *APIHandler) GetAllEmailsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	options := repository.EmailSearchOptions{
		AccountID: 0, // 0 means all accounts
		Limit:     50,
		Offset:    0,
		SortBy:    "date DESC",
	}

	// Parse limit
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			if parsedLimit > 100 {
				parsedLimit = 100 // Cap at 100
			}
			options.Limit = parsedLimit
		}
	}

	// Parse offset
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			options.Offset = parsedOffset
		}
	}

	// Parse sort order
	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		validSortOptions := map[string]string{
			"date_desc":    "date DESC",
			"date_asc":     "date ASC",
			"subject_asc":  "subject ASC",
			"subject_desc": "subject DESC",
		}
		if validSort, exists := validSortOptions[sortBy]; exists {
			options.SortBy = validSort
		}
	}

	// Parse date range
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if parsed, err := time.Parse(time.RFC3339, startDate); err == nil {
			options.StartDate = &parsed
		}
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if parsed, err := time.Parse(time.RFC3339, endDate); err == nil {
			options.EndDate = &parsed
		}
	}

	// Parse text search parameters
	options.FromQuery = r.URL.Query().Get("from_query")
	options.ToQuery = r.URL.Query().Get("to_query")
	options.CcQuery = r.URL.Query().Get("cc_query")
	options.SubjectQuery = r.URL.Query().Get("subject_query")
	options.BodyQuery = r.URL.Query().Get("body_query")
	options.HTMLQuery = r.URL.Query().Get("html_query")
	options.Keyword = r.URL.Query().Get("keyword")
	options.MailboxName = r.URL.Query().Get("mailbox")

	// Perform search
	emails, totalCount, err := h.EmailRepo.SearchEmails(options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate pagination info
	totalPages := int((totalCount + int64(options.Limit) - 1) / int64(options.Limit))
	currentPage := (options.Offset / options.Limit) + 1
	hasNext := options.Offset+options.Limit < int(totalCount)
	hasPrev := options.Offset > 0

	response := map[string]interface{}{
		"emails": emails,
		"pagination": map[string]interface{}{
			"total":        totalCount,
			"total_pages":  totalPages,
			"current_page": currentPage,
			"limit":        options.Limit,
			"offset":       options.Offset,
			"has_next":     hasNext,
			"has_prev":     hasPrev,
		},
		"search_criteria": map[string]interface{}{
			"account_id":    options.AccountID,
			"start_date":    options.StartDate,
			"end_date":      options.EndDate,
			"from_query":    options.FromQuery,
			"to_query":      options.ToQuery,
			"cc_query":      options.CcQuery,
			"subject_query": options.SubjectQuery,
			"body_query":    options.BodyQuery,
			"html_query":    options.HTMLQuery,
			"keyword":       options.Keyword,
			"mailbox":       options.MailboxName,
			"sort_by":       options.SortBy,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEmailFoldersHandler retrieves all unique mailbox folders across all accounts
// @Summary Get all unique email folders
// @Description Get all unique mailbox folders (like INBOX, Sent, Drafts, etc.) across all accounts
// @Tags account-emails
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Response with folders array"
// @Router /api/account-emails/folders [get]
func (h *APIHandler) GetEmailFoldersHandler(w http.ResponseWriter, r *http.Request) {
	folders, err := h.EmailRepo.GetAllMailboxFolders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"folders": folders,
		"count":   len(folders),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ================ 同步监控相关处理器 ================

// GetQueueMetricsHandler 获取队列监控指标
func (h *APIHandler) GetQueueMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if h.optimizedSyncManager == nil {
		http.Error(w, "Optimized sync manager not available", http.StatusServiceUnavailable)
		return
	}

	metrics := h.optimizedSyncManager.GetQueueMetrics()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success": true,
		"data":    metrics,
	}

	json.NewEncoder(w).Encode(response)
}

// GetAccountSyncStatusHandler 获取账户同步状态
func (h *APIHandler) GetAccountSyncStatusHandler(w http.ResponseWriter, r *http.Request) {
	if h.perAccountSyncManager == nil {
		http.Error(w, "Per-account sync manager not available", http.StatusServiceUnavailable)
		return
	}

	// 获取查询参数
	accountIDStr := r.URL.Query().Get("account_id")

	if accountIDStr != "" {
		// 获取单个账户状态
		accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
		if err != nil {
			http.Error(w, "Invalid account ID", http.StatusBadRequest)
			return
		}

		status, err := h.perAccountSyncManager.GetAccountSyncerStatus(uint(accountID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"success": true,
			"data":    status,
		}
		json.NewEncoder(w).Encode(response)
	} else {
		// 获取所有账户状态
		statuses := h.perAccountSyncManager.GetAllAccountSyncerStatuses()

		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"success": true,
			"data":    statuses,
		}
		json.NewEncoder(w).Encode(response)
	}
}

// GetSyncManagerStatsHandler 获取同步管理器统计
func (h *APIHandler) GetSyncManagerStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})

	// 获取优化同步管理器统计
	if h.optimizedSyncManager != nil {
		stats["optimized_manager"] = h.optimizedSyncManager.GetQueueMetrics()
	}

	// 获取每账户同步管理器统计
	if h.perAccountSyncManager != nil {
		stats["per_account_manager"] = h.perAccountSyncManager.GetStats()
	}

	// 获取邮件调度器统计
	if h.EmailScheduler != nil {
		stats["email_scheduler"] = h.EmailScheduler.GetMetrics()
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success": true,
		"data":    stats,
	}

	json.NewEncoder(w).Encode(response)
}
