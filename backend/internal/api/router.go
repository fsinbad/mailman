package api

import (
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter creates a new router with all the necessary routes.
func NewRouter(handler *APIHandler, openAIHandler *OpenAIHandler, wsHandler *WebSocketHandler, syncHandlers *SyncHandlers) http.Handler {
	router := mux.NewRouter()

	// Create API subrouter with /api prefix
	apiRouter := router.PathPrefix("/api").Subrouter()

	// Plugin management
	apiRouter.HandleFunc("/plugins", handler.ListPluginsHandler).Methods("GET")
	apiRouter.HandleFunc("/plugins/ui/schemas", handler.GetPluginUISchemas).Methods("GET")
	apiRouter.HandleFunc("/plugins/ui/schema", handler.GetPluginUISchema).Methods("GET")
	apiRouter.HandleFunc("/plugins/{pluginID}/callbacks/{callback}", handler.HandlePluginCallback).Methods("POST")

	// Health check
	apiRouter.HandleFunc("/health", HealthCheck).Methods("GET")

	// Account management
	apiRouter.HandleFunc("/accounts", handler.CreateAccountHandler).Methods("POST")
	apiRouter.HandleFunc("/accounts", handler.GetAccountsHandler).Methods("GET")
	apiRouter.HandleFunc("/accounts/paginated", handler.GetAccountsPaginatedHandler).Methods("GET")
	apiRouter.HandleFunc("/accounts/{id}", handler.GetAccountHandler).Methods("GET")
	apiRouter.HandleFunc("/accounts/{id}", handler.UpdateAccountHandler).Methods("PUT")
	apiRouter.HandleFunc("/accounts/{id}", handler.DeleteAccountHandler).Methods("DELETE")
	apiRouter.HandleFunc("/accounts/verify", handler.VerifyAccountHandler).Methods("POST")
	apiRouter.HandleFunc("/accounts/batch-verify", handler.BatchVerifyAccountsHandler).Methods("POST")

	// Account-specific email operations (moved to account-emails category)
	apiRouter.HandleFunc("/account-emails/fetch/{id}", handler.FetchAndStoreEmailsHandler).Methods("POST")
	// 新增：获取所有邮件的路由（必须放在参数路由之前）
	apiRouter.HandleFunc("/account-emails/list/all", handler.GetAllEmailsHandler).Methods("GET")
	apiRouter.HandleFunc("/account-emails/folders", handler.GetEmailFoldersHandler).Methods("GET")
	apiRouter.HandleFunc("/account-emails/list/{id}", handler.GetEmailsHandler).Methods("GET")
	apiRouter.HandleFunc("/account-emails/extract/{id}", handler.ExtractEmailsHandler).Methods("POST")
	apiRouter.HandleFunc("/accounts/{id}/sync-records", handler.GetIncrementalSyncRecordsHandler).Methods("GET")
	apiRouter.HandleFunc("/accounts/{id}/last-sync-record", handler.GetLastSyncRecordHandler).Methods("GET")
	apiRouter.HandleFunc("/accounts/{id}/sync-records", handler.DeleteIncrementalSyncRecordHandler).Methods("DELETE")

	// General email operations
	apiRouter.HandleFunc("/emails/extract", handler.ExtractEmailsHandler).Methods("POST") // Global extract without account ID
	apiRouter.HandleFunc("/emails/search", handler.SearchEmailsHandler).Methods("GET")    // New search endpoint with optional account ID
	apiRouter.HandleFunc("/emails/{id}", handler.GetEmailHandler).Methods("GET")

	// Legacy endpoint
	apiRouter.HandleFunc("/fetch-emails", handler.FetchEmailsHandler).Methods("POST")

	// Random email endpoint
	apiRouter.HandleFunc("/random-email", handler.RandomEmailHandler).Methods("GET")

	// Wait email endpoint
	apiRouter.HandleFunc("/wait-email", handler.WaitEmailHandler).Methods("POST")

	// Check email endpoint (simplified for frontend polling)
	apiRouter.HandleFunc("/check-email", handler.CheckEmailHandler).Methods("POST")

	// WebSocket endpoint for waiting emails
	apiRouter.HandleFunc("/ws/wait-email", handler.WaitEmailWebSocketHandler).Methods("GET")

	// HTTP polling endpoint (fallback for WebSocket)
	apiRouter.HandleFunc("/poll-email", handler.PollEmailHandler).Methods("POST")

	// WebSocket endpoint for subscription-based monitoring
	apiRouter.HandleFunc("/ws/subscriptions", handler.SubscriptionWebSocketHandler).Methods("GET")

	// Email domains endpoint
	apiRouter.HandleFunc("/email-domains", handler.GetEmailDomainsHandler).Methods("GET")

	// Mail providers
	apiRouter.HandleFunc("/providers", handler.GetProvidersHandler).Methods("GET")

	// Extractor templates
	apiRouter.HandleFunc("/extractor-templates", handler.CreateExtractorTemplateHandler).Methods("POST")
	apiRouter.HandleFunc("/extractor-templates", handler.GetExtractorTemplatesHandler).Methods("GET")
	apiRouter.HandleFunc("/extractor-templates/paginated", handler.GetExtractorTemplatesPaginatedHandler).Methods("GET")
	apiRouter.HandleFunc("/extractor-templates/{id}", handler.GetExtractorTemplateHandler).Methods("GET")
	apiRouter.HandleFunc("/extractor-templates/{id}", handler.UpdateExtractorTemplateHandler).Methods("PUT")
	apiRouter.HandleFunc("/extractor-templates/{id}", handler.DeleteExtractorTemplateHandler).Methods("DELETE")
	apiRouter.HandleFunc("/extractor-templates/{id}/test", handler.TestExtractorTemplateHandler).Methods("POST")

	// OpenAI Configuration endpoints
	apiRouter.HandleFunc("/openai/configs", openAIHandler.ListOpenAIConfigs).Methods("GET")
	apiRouter.HandleFunc("/openai/configs", openAIHandler.CreateOpenAIConfig).Methods("POST")
	apiRouter.HandleFunc("/openai/configs/{id}", openAIHandler.GetOpenAIConfig).Methods("GET")
	apiRouter.HandleFunc("/openai/configs/{id}", openAIHandler.UpdateOpenAIConfig).Methods("PUT")
	apiRouter.HandleFunc("/openai/configs/{id}", openAIHandler.DeleteOpenAIConfig).Methods("DELETE")

	// AI Prompt Template endpoints
	apiRouter.HandleFunc("/openai/prompt-templates", openAIHandler.ListAIPromptTemplates).Methods("GET")
	apiRouter.HandleFunc("/openai/prompt-templates", openAIHandler.CreateAIPromptTemplate).Methods("POST")
	apiRouter.HandleFunc("/openai/prompt-templates/{id}", openAIHandler.GetAIPromptTemplate).Methods("GET")
	apiRouter.HandleFunc("/openai/prompt-templates/{id}", openAIHandler.UpdateAIPromptTemplate).Methods("PUT")
	apiRouter.HandleFunc("/openai/prompt-templates/{id}", openAIHandler.DeleteAIPromptTemplate).Methods("DELETE")

	// AI Generation endpoints
	apiRouter.HandleFunc("/openai/generate-template", openAIHandler.GenerateEmailTemplate).Methods("POST")
	apiRouter.HandleFunc("/openai/initialize-templates", openAIHandler.InitializeDefaultPromptTemplates).Methods("POST")
	apiRouter.HandleFunc("/openai/call", openAIHandler.CallOpenAI).Methods("POST")
	apiRouter.HandleFunc("/openai/test-config", openAIHandler.TestOpenAIConfig).Methods("POST")

	// Email subscription endpoints
	apiRouter.HandleFunc("/subscriptions", handler.CreateSubscriptionHandler).Methods("POST")
	apiRouter.HandleFunc("/subscriptions", handler.GetSubscriptionsHandler).Methods("GET")
	apiRouter.HandleFunc("/subscriptions/{id}", handler.DeleteSubscriptionHandler).Methods("DELETE")

	// Cache statistics endpoint
	apiRouter.HandleFunc("/cache/stats", handler.GetCacheStatsHandler).Methods("GET")

	// Dashboard statistics endpoint
	apiRouter.HandleFunc("/dashboard/stats", handler.GetEmailStatsHandler).Methods("GET")

	// Immediate email fetch endpoint
	apiRouter.HandleFunc("/emails/fetch-now", handler.FetchNowHandler).Methods("POST")

	// WebSocket和通知相关端点
	apiRouter.HandleFunc("/ws/notifications", wsHandler.HandleWebSocket)
	apiRouter.HandleFunc("/notifications/stats", wsHandler.HandleNotificationStats).Methods("GET")
	apiRouter.HandleFunc("/notifications/recent", wsHandler.HandleRecentNotifications).Methods("GET")

	// 同步监控相关端点
	apiRouter.HandleFunc("/sync/queue-metrics", handler.GetQueueMetricsHandler).Methods("GET")
	apiRouter.HandleFunc("/sync/account-status", handler.GetAccountSyncStatusHandler).Methods("GET")
	apiRouter.HandleFunc("/sync/manager-stats", handler.GetSyncManagerStatsHandler).Methods("GET")

	// Sync configuration endpoints
	apiRouter.HandleFunc("/accounts/{id}/sync-config", syncHandlers.GetAccountSyncConfig).Methods("GET")
	apiRouter.HandleFunc("/accounts/{id}/sync-config", syncHandlers.CreateAccountSyncConfig).Methods("POST")
	apiRouter.HandleFunc("/accounts/{id}/sync-config", syncHandlers.UpdateAccountSyncConfig).Methods("PUT")
	apiRouter.HandleFunc("/accounts/{id}/sync-config", syncHandlers.DeleteAccountSyncConfig).Methods("DELETE")
	apiRouter.HandleFunc("/accounts/{id}/sync-config/effective", syncHandlers.GetEffectiveSyncConfig).Methods("GET")
	apiRouter.HandleFunc("/accounts/{id}/sync-config/temporary", syncHandlers.CreateTemporarySyncConfig).Methods("POST")
	apiRouter.HandleFunc("/accounts/{id}/sync-now", syncHandlers.SyncNow).Methods("POST")
	apiRouter.HandleFunc("/accounts/{id}/mailboxes", syncHandlers.GetAccountMailboxes).Methods("GET")

	// Sync global configuration
	apiRouter.HandleFunc("/sync/global-config", syncHandlers.GetGlobalSyncConfig).Methods("GET")
	apiRouter.HandleFunc("/sync/global-config", syncHandlers.UpdateGlobalSyncConfig).Methods("PUT")
	apiRouter.HandleFunc("/sync/configs", syncHandlers.GetAllSyncConfigs).Methods("GET")

	// Batch sync configuration
	apiRouter.HandleFunc("/sync/batch-config", syncHandlers.BatchCreateOrUpdateAccountSyncConfig).Methods("POST")

	// Swagger documentation
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Static file server for docs directory
	router.PathPrefix("/docs/").Handler(http.StripPrefix("/docs/", http.FileServer(http.Dir("./docs/"))))

	// Serve the modern interface as the default route
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/modern-index.html")
	}).Methods("GET")

	// Add CORS middleware
	return enableCORS(router)
}

// enableCORS adds CORS headers to responses
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
