package api

import (
	"mailman/internal/repository"
	"mailman/internal/services"
	"mailman/internal/utils"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouterWithAuth creates a new router with authentication middleware
func NewRouterWithAuth(
	handler *APIHandler,
	openAIHandler *OpenAIHandler,
	authHandler *AuthHandler,
	syncHandlers *SyncHandlers,
	sessionHandler *SessionHandler,
	triggerHandler *TriggerAPIHandler,
	oauth2Handler *OAuth2Handler,
	systemConfigHandler *SystemConfigHandler,
	webSocketHandler *WebSocketHandler,
	authService *services.AuthService,
	emailTriggerService *services.EmailTriggerService,
	emailTriggerV2Repo *repository.EmailTriggerV2Repository,
	triggerExecutionLogV2Repo *repository.TriggerExecutionLogV2Repository,
	pluginManager *services.PluginManager,
	conditionEngine *services.ConditionEngine,
) http.Handler {
	router := mux.NewRouter()

	// Create logger for HTTP logging
	logger := utils.NewLogger("HTTP")

	// Apply logging middleware to all routes
	router.Use(LoggingMiddleware(logger))

	// Create API subrouter with /api prefix
	apiRouter := router.PathPrefix("/api").Subrouter()

	// Public endpoints (no auth required)
	// Health check
	apiRouter.HandleFunc("/health", HealthCheck).Methods("GET")

	// Authentication endpoints (public)
	apiRouter.HandleFunc("/auth/login", authHandler.LoginHandler).Methods("POST")

	// OAuth2 callback endpoints (public - called by external providers)
	apiRouter.HandleFunc("/oauth2/callback/{provider}", oauth2Handler.HandleCallback).Methods("GET")
	apiRouter.HandleFunc("/oauth2/exchange-thunderbird-token", oauth2Handler.ExchangeThunderbirdToken).Methods("POST")

	// Public WebSocket endpoints (no auth required)
	apiRouter.HandleFunc("/ws/wait-email", handler.WaitEmailWebSocketHandler).Methods("GET")
	apiRouter.HandleFunc("/ws/notifications", webSocketHandler.HandleWebSocket).Methods("GET")

	// Create authenticated subrouter
	authRouter := apiRouter.PathPrefix("").Subrouter()
	authRouter.Use(AuthMiddleware(authService))

	// Authentication endpoints (protected)
	authRouter.HandleFunc("/auth/logout", authHandler.LogoutHandler).Methods("POST")
	authRouter.HandleFunc("/auth/me", authHandler.CurrentUserHandler).Methods("GET")
	authRouter.HandleFunc("/auth/update", authHandler.UpdateUserHandler).Methods("PUT")

	// Session management (protected)
	authRouter.HandleFunc("/sessions", sessionHandler.GetUserSessionsHandler).Methods("GET")
	authRouter.HandleFunc("/sessions", sessionHandler.CreateUserSessionHandler).Methods("POST")
	authRouter.HandleFunc("/sessions/{id}", sessionHandler.UpdateUserSessionHandler).Methods("PUT")
	authRouter.HandleFunc("/sessions/{id}", sessionHandler.DeleteUserSessionHandler).Methods("DELETE")

	// Account management (protected)
	authRouter.HandleFunc("/accounts", handler.CreateAccountHandler).Methods("POST")
	authRouter.HandleFunc("/accounts/upsert", handler.UpsertAccountHandler).Methods("POST")
	authRouter.HandleFunc("/accounts", handler.GetAccountsHandler).Methods("GET")
	authRouter.HandleFunc("/accounts/paginated", handler.GetAccountsPaginatedHandler).Methods("GET")
	authRouter.HandleFunc("/accounts/{id}", handler.GetAccountHandler).Methods("GET")
	authRouter.HandleFunc("/accounts/{id}", handler.UpdateAccountHandler).Methods("PUT")
	authRouter.HandleFunc("/accounts/{id}", handler.DeleteAccountHandler).Methods("DELETE")
	authRouter.HandleFunc("/accounts/verify", handler.VerifyAccountHandler).Methods("POST")
	authRouter.HandleFunc("/accounts/batch-verify", handler.BatchVerifyAccountsHandler).Methods("POST")

	// Activity logs (protected)
	authRouter.HandleFunc("/activities/recent", GetRecentActivities).Methods("GET")

	// Account-specific email operations (protected)
	authRouter.HandleFunc("/account-emails/fetch/{id}", handler.FetchAndStoreEmailsHandler).Methods("POST")
	// 新增：获取所有邮件和文件夹的路由（必须放在参数路由之前）
	authRouter.HandleFunc("/account-emails/list/all", handler.GetAllEmailsHandler).Methods("GET")
	authRouter.HandleFunc("/account-emails/folders", handler.GetEmailFoldersHandler).Methods("GET")
	authRouter.HandleFunc("/account-emails/list/{id}", handler.GetEmailsHandler).Methods("GET")
	authRouter.HandleFunc("/account-emails/extract/{id}", handler.ExtractEmailsHandler).Methods("POST")
	authRouter.HandleFunc("/accounts/{id}/sync-records", handler.GetIncrementalSyncRecordsHandler).Methods("GET")
	authRouter.HandleFunc("/accounts/{id}/last-sync-record", handler.GetLastSyncRecordHandler).Methods("GET")
	authRouter.HandleFunc("/accounts/{id}/sync-records", handler.DeleteIncrementalSyncRecordHandler).Methods("DELETE")

	// General email operations (protected)
	authRouter.HandleFunc("/emails/extract", handler.ExtractEmailsHandler).Methods("POST")
	authRouter.HandleFunc("/emails/search", handler.SearchEmailsHandler).Methods("GET") // 添加搜索路由
	authRouter.HandleFunc("/emails/{id}", handler.GetEmailHandler).Methods("GET")

	// Legacy endpoint (protected)
	authRouter.HandleFunc("/fetch-emails", handler.FetchEmailsHandler).Methods("POST")

	// Random email endpoint (protected)
	authRouter.HandleFunc("/random-email", handler.RandomEmailHandler).Methods("GET")

	// Wait email endpoint (protected)
	authRouter.HandleFunc("/wait-email", handler.WaitEmailHandler).Methods("POST")

	// Check email endpoint (simplified for frontend polling) (protected)
	authRouter.HandleFunc("/check-email", handler.CheckEmailHandler).Methods("POST")

	// WebSocket endpoints (protected)
	authRouter.HandleFunc("/ws/wait-email", handler.WaitEmailWebSocketHandler).Methods("GET")
	authRouter.HandleFunc("/ws/subscriptions", handler.SubscriptionWebSocketHandler).Methods("GET")

	// HTTP polling endpoint (fallback for WebSocket) (protected)
	authRouter.HandleFunc("/poll-email", handler.PollEmailHandler).Methods("POST")

	// Email domains endpoint (protected)
	authRouter.HandleFunc("/email-domains", handler.GetEmailDomainsHandler).Methods("GET")

	// Mail providers (protected)
	authRouter.HandleFunc("/providers", handler.GetProvidersHandler).Methods("GET")

	// Extractor templates (protected)
	authRouter.HandleFunc("/extractor-templates", handler.CreateExtractorTemplateHandler).Methods("POST")
	authRouter.HandleFunc("/extractor-templates", handler.GetExtractorTemplatesHandler).Methods("GET")
	authRouter.HandleFunc("/extractor-templates/paginated", handler.GetExtractorTemplatesPaginatedHandler).Methods("GET")
	authRouter.HandleFunc("/extractor-templates/{id}", handler.GetExtractorTemplateHandler).Methods("GET")
	authRouter.HandleFunc("/extractor-templates/{id}", handler.UpdateExtractorTemplateHandler).Methods("PUT")
	authRouter.HandleFunc("/extractor-templates/{id}", handler.DeleteExtractorTemplateHandler).Methods("DELETE")
	authRouter.HandleFunc("/extractor-templates/{id}/test", handler.TestExtractorTemplateHandler).Methods("POST")

	// OpenAI Configuration endpoints (protected)
	authRouter.HandleFunc("/openai/configs", openAIHandler.ListOpenAIConfigs).Methods("GET")
	authRouter.HandleFunc("/openai/configs", openAIHandler.CreateOpenAIConfig).Methods("POST")
	authRouter.HandleFunc("/openai/configs/{id}", openAIHandler.GetOpenAIConfig).Methods("GET")
	authRouter.HandleFunc("/openai/configs/{id}", openAIHandler.UpdateOpenAIConfig).Methods("PUT")
	authRouter.HandleFunc("/openai/configs/{id}", openAIHandler.DeleteOpenAIConfig).Methods("DELETE")

	// AI Prompt Template endpoints (protected)
	authRouter.HandleFunc("/openai/prompt-templates", openAIHandler.ListAIPromptTemplates).Methods("GET")
	authRouter.HandleFunc("/openai/prompt-templates", openAIHandler.CreateAIPromptTemplate).Methods("POST")
	authRouter.HandleFunc("/openai/prompt-templates/{id}", openAIHandler.GetAIPromptTemplate).Methods("GET")
	authRouter.HandleFunc("/openai/prompt-templates/{id}", openAIHandler.UpdateAIPromptTemplate).Methods("PUT")
	authRouter.HandleFunc("/openai/prompt-templates/{id}", openAIHandler.DeleteAIPromptTemplate).Methods("DELETE")

	// AI Generation endpoints (protected)
	authRouter.HandleFunc("/openai/generate-template", openAIHandler.GenerateEmailTemplate).Methods("POST")
	authRouter.HandleFunc("/openai/initialize-templates", openAIHandler.InitializeDefaultPromptTemplates).Methods("POST")
	authRouter.HandleFunc("/openai/call", openAIHandler.CallOpenAI).Methods("POST")
	authRouter.HandleFunc("/openai/test-config", openAIHandler.TestOpenAIConfig).Methods("POST")

	// Email subscription endpoints (protected)
	authRouter.HandleFunc("/subscriptions", handler.CreateSubscriptionHandler).Methods("POST")
	authRouter.HandleFunc("/subscriptions", handler.GetSubscriptionsHandler).Methods("GET")
	authRouter.HandleFunc("/subscriptions/{id}", handler.DeleteSubscriptionHandler).Methods("DELETE")

	// Cache statistics endpoint (protected)
	authRouter.HandleFunc("/cache/stats", handler.GetCacheStatsHandler).Methods("GET")

	// Dashboard statistics endpoint (protected)
	authRouter.HandleFunc("/dashboard/stats", handler.GetEmailStatsHandler).Methods("GET")

	// WebSocket和通知相关端点 (protected)
	authRouter.HandleFunc("/notifications/stats", webSocketHandler.HandleNotificationStats).Methods("GET")
	authRouter.HandleFunc("/notifications/recent", webSocketHandler.HandleRecentNotifications).Methods("GET")

	// 同步监控相关端点 (protected)
	authRouter.HandleFunc("/sync/queue-metrics", handler.GetQueueMetricsHandler).Methods("GET")
	authRouter.HandleFunc("/sync/account-status", handler.GetAccountSyncStatusHandler).Methods("GET")
	authRouter.HandleFunc("/sync/manager-stats", handler.GetSyncManagerStatsHandler).Methods("GET")

	// Immediate email fetch endpoint (protected)
	authRouter.HandleFunc("/emails/fetch-now", handler.FetchNowHandler).Methods("POST")

	// Sync configuration endpoints (protected)
	authRouter.HandleFunc("/accounts/{id}/sync-config", syncHandlers.GetAccountSyncConfig).Methods("GET")
	authRouter.HandleFunc("/accounts/{id}/sync-config", syncHandlers.CreateAccountSyncConfig).Methods("POST")
	authRouter.HandleFunc("/accounts/{id}/sync-config", syncHandlers.UpdateAccountSyncConfig).Methods("PUT")
	authRouter.HandleFunc("/accounts/{id}/sync-config", syncHandlers.DeleteAccountSyncConfig).Methods("DELETE")
	authRouter.HandleFunc("/accounts/{id}/sync-config/effective", syncHandlers.GetEffectiveSyncConfig).Methods("GET")
	authRouter.HandleFunc("/accounts/{id}/sync-config/temporary", syncHandlers.CreateTemporarySyncConfig).Methods("POST")
	authRouter.HandleFunc("/accounts/{id}/sync-now", syncHandlers.SyncNow).Methods("POST")
	authRouter.HandleFunc("/accounts/{id}/sync-statistics", syncHandlers.GetSyncStatistics).Methods("GET")
	authRouter.HandleFunc("/accounts/{id}/mailboxes", syncHandlers.GetAccountMailboxes).Methods("GET")
	authRouter.HandleFunc("/sync/configs", syncHandlers.GetAllSyncConfigs).Methods("GET")
	authRouter.HandleFunc("/sync/global-config", syncHandlers.GetGlobalSyncConfig).Methods("GET")
	authRouter.HandleFunc("/sync/global-config", syncHandlers.UpdateGlobalSyncConfig).Methods("PUT")

	// Batch sync configuration (protected)
	authRouter.HandleFunc("/sync/batch-config", syncHandlers.BatchCreateOrUpdateAccountSyncConfig).Methods("POST")

	// Legacy Trigger endpoints (protected) - 保持向后兼容
	authRouter.HandleFunc("/triggers", triggerHandler.CreateTriggerHandler).Methods("POST")
	authRouter.HandleFunc("/triggers", triggerHandler.GetTriggersHandler).Methods("GET")
	authRouter.HandleFunc("/triggers/{id}", triggerHandler.GetTriggerHandler).Methods("GET")
	authRouter.HandleFunc("/triggers/{id}", triggerHandler.UpdateTriggerHandler).Methods("PUT")
	authRouter.HandleFunc("/triggers/{id}", triggerHandler.DeleteTriggerHandler).Methods("DELETE")
	authRouter.HandleFunc("/triggers/{id}/enable", triggerHandler.EnableTriggerHandler).Methods("POST")
	authRouter.HandleFunc("/triggers/{id}/disable", triggerHandler.DisableTriggerHandler).Methods("POST")
	authRouter.HandleFunc("/triggers/evaluate-expression", triggerHandler.EvaluateExpressionHandler).Methods("POST")
	authRouter.HandleFunc("/triggers/execute-action", triggerHandler.ExecuteActionHandler).Methods("POST")
	authRouter.HandleFunc("/triggers/execute-actions", triggerHandler.ExecuteActionsHandler).Methods("POST")
	authRouter.HandleFunc("/trigger-logs", triggerHandler.GetTriggerExecutionLogsHandler).Methods("GET")
	authRouter.HandleFunc("/trigger-stats", triggerHandler.GetTriggerStatsHandler).Methods("GET")

	// TriggerV2 endpoints (protected) - 新一代触发器API
	v2Router := authRouter.PathPrefix("/v2").Subrouter()
	v2Router.HandleFunc("/triggers", triggerHandler.CreateTriggerV2Handler).Methods("POST")
	v2Router.HandleFunc("/triggers", triggerHandler.GetTriggersV2Handler).Methods("GET")
	v2Router.HandleFunc("/triggers/{id}", triggerHandler.GetTriggerV2Handler).Methods("GET")
	v2Router.HandleFunc("/triggers/{id}", triggerHandler.UpdateTriggerV2Handler).Methods("PUT")
	v2Router.HandleFunc("/triggers/{id}", triggerHandler.DeleteTriggerHandler).Methods("DELETE")                   // 复用Legacy删除逻辑
	v2Router.HandleFunc("/triggers/{id}/enable", triggerHandler.EnableTriggerHandler).Methods("POST")              // 复用Legacy启用逻辑
	v2Router.HandleFunc("/triggers/{id}/disable", triggerHandler.DisableTriggerHandler).Methods("POST")            // 复用Legacy禁用逻辑
	v2Router.HandleFunc("/triggers/evaluate-expression", triggerHandler.EvaluateExpressionHandler).Methods("POST") // 复用Legacy表达式评估
	v2Router.HandleFunc("/triggers/execute-action", triggerHandler.ExecuteActionHandler).Methods("POST")           // 复用Legacy动作执行
	v2Router.HandleFunc("/triggers/execute-actions", triggerHandler.ExecuteActionsHandler).Methods("POST")         // 复用Legacy批量动作执行

	// Create EmailTriggerV2Controller
	emailTriggerV2Controller := NewEmailTriggerV2Controller(
		emailTriggerService,
		emailTriggerV2Repo,
		triggerExecutionLogV2Repo,
		pluginManager,
		conditionEngine,
	)

	// Register all Email Trigger V2 routes
	emailTriggerV2Controller.RegisterRoutes(v2Router)

	// Activity log endpoints (protected)
	authRouter.HandleFunc("/activities/recent", GetRecentActivities).Methods("GET")
	authRouter.HandleFunc("/activities/stats", GetActivityStats).Methods("GET")
	authRouter.HandleFunc("/activities/type/{type}", GetActivitiesByType).Methods("GET")
	authRouter.HandleFunc("/activities/cleanup", DeleteOldActivities).Methods("DELETE")

	// OAuth2 endpoints (protected)
	authRouter.HandleFunc("/oauth2/global-config", oauth2Handler.CreateOrUpdateGlobalConfig).Methods("POST", "PUT")
	authRouter.HandleFunc("/oauth2/global-configs", oauth2Handler.GetGlobalConfigs).Methods("GET")
	authRouter.HandleFunc("/oauth2/global-config/{provider}", oauth2Handler.GetGlobalConfigByProvider).Methods("GET")
	authRouter.HandleFunc("/oauth2/global-configs/{provider}", oauth2Handler.GetGlobalConfigsByProvider).Methods("GET")
	authRouter.HandleFunc("/oauth2/global-config/by-id/{id}", oauth2Handler.GetGlobalConfigByID).Methods("GET")
	authRouter.HandleFunc("/oauth2/global-config/{id}", oauth2Handler.DeleteGlobalConfig).Methods("DELETE")
	authRouter.HandleFunc("/oauth2/auth-url/{provider}", oauth2Handler.GetAuthURL).Methods("GET")
	authRouter.HandleFunc("/oauth2/exchange-token", oauth2Handler.ExchangeToken).Methods("POST")
	authRouter.HandleFunc("/oauth2/refresh-token", oauth2Handler.RefreshTokenHandler).Methods("POST")
	authRouter.HandleFunc("/oauth2/provider/{provider}/enable", oauth2Handler.EnableProvider).Methods("POST")
	authRouter.HandleFunc("/oauth2/provider/{provider}/disable", oauth2Handler.DisableProvider).Methods("POST")

	// OAuth2 授权会话管理端点 (protected)
	authRouter.HandleFunc("/oauth2/session/start/{provider}", oauth2Handler.StartOAuth2Session).Methods("POST")
	authRouter.HandleFunc("/oauth2/session/poll/{state}", oauth2Handler.PollOAuth2SessionStatus).Methods("GET")
	authRouter.HandleFunc("/oauth2/session/cancel/{state}", oauth2Handler.CancelOAuth2Session).Methods("POST")

	// System Configuration endpoints (protected)
	authRouter.HandleFunc("/system-configs", systemConfigHandler.GetAllConfigs).Methods("GET")
	authRouter.HandleFunc("/system-configs/category/{category}", systemConfigHandler.GetConfigsByCategory).Methods("GET")
	authRouter.HandleFunc("/system-config/{key}", systemConfigHandler.GetConfigByKey).Methods("GET")
	authRouter.HandleFunc("/system-config/{key}", systemConfigHandler.UpdateConfigValue).Methods("PUT")
	authRouter.HandleFunc("/system-config/{key}/reset", systemConfigHandler.ResetConfigToDefault).Methods("POST")

	// Plugin management (protected)
	authRouter.HandleFunc("/plugins", handler.ListPluginsHandler).Methods("GET")
	authRouter.HandleFunc("/plugins/ui/schemas", handler.GetPluginUISchemas).Methods("GET")
	authRouter.HandleFunc("/plugins/ui/schema", handler.GetPluginUISchema).Methods("GET")
	authRouter.HandleFunc("/plugins/{pluginID}/callbacks/{callback}", handler.HandlePluginCallback).Methods("POST")

	// Swagger documentation (public)
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Static file server for docs directory (public)
	router.PathPrefix("/docs/").Handler(http.StripPrefix("/docs/", http.FileServer(http.Dir("./docs/"))))

	// Serve the modern interface as the default route (public)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/modern-index.html")
	}).Methods("GET")

	// Add CORS middleware
	return enableCORS(router)
}
