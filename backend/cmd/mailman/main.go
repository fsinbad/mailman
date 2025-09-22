package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "mailman/docs" // This is required for swag to find your docs
	"mailman/internal/api"
	"mailman/internal/config"
	"mailman/internal/database"
	"mailman/internal/repository"
	"mailman/internal/services"
	"mailman/internal/triggerv2/plugins"
	"mailman/internal/triggerv2/plugins/builtin"
	"mailman/internal/utils"
)

// @title Mailman API
// @version 1.0
// @description This is a sample server for a mailman service.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
func main() {
	// Initialize logger with configured log level
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO" // Default log level
	}

	mainLogger := utils.NewLogger("Main")
	mainLogger.Info("Starting Mailman Service with log level: %s", logLevel)

	// Load configuration
	cfg := config.Load()

	// Initialize database
	dbConfig := database.Config{
		Driver:   cfg.Database.Driver,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	if err := database.Initialize(dbConfig); err != nil {
		mainLogger.Error("Failed to initialize database: %v", err)
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	db := database.GetDB()

	// Initialize repositories
	mailProviderRepo := repository.NewMailProviderRepository(db)
	emailAccountRepo := repository.NewEmailAccountRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	incrementalSyncRepo := repository.NewIncrementalSyncRepository(db)
	extractorTemplateRepo := repository.NewExtractorTemplateRepository(db)
	openAIConfigRepo := repository.NewOpenAIConfigRepository(db)
	aiPromptTemplateRepo := repository.NewAIPromptTemplateRepository(db)
	userRepo := repository.NewUserRepository(db)
	userSessionRepo := repository.NewUserSessionRepository(db)
	syncConfigRepo := repository.NewSyncConfigRepository(db)
	mailboxRepo := repository.NewMailboxRepository(db)
	triggerRepo := repository.NewTriggerRepository(db)
	triggerLogRepo := repository.NewTriggerExecutionLogRepository(db)
	emailTriggerV2Repo := repository.NewEmailTriggerV2Repository(db)
	triggerExecutionLogV2Repo := repository.NewTriggerExecutionLogV2Repository(db)
	oauth2GlobalConfigRepo := repository.NewOAuth2GlobalConfigRepository(db)
	oauth2AuthSessionRepo := repository.NewOAuth2AuthSessionRepository(db)
	systemConfigRepo := repository.NewSystemConfigRepository(db)

	// Seed default mail providers
	if err := mailProviderRepo.SeedDefaultProviders(); err != nil {
		mainLogger.Warn("Failed to seed default providers: %v", err)
	}

	// Seed default OAuth2 configurations
	if err := oauth2GlobalConfigRepo.SeedDefaultConfigs(); err != nil {
		mainLogger.Warn("Failed to seed default OAuth2 configs: %v", err)
	}

	// Initialize services with repositories
	fetcherService := services.NewFetcherService(emailAccountRepo, emailRepo, db)
	parserService := services.NewParserService()
	authService := services.NewAuthService(userRepo, userSessionRepo)
	oauth2Service := services.NewOAuth2Service(db)
	oauth2ConfigService := services.NewOAuth2GlobalConfigService(oauth2GlobalConfigRepo)
	oauth2AuthSessionService := services.NewOAuth2AuthSessionService(oauth2AuthSessionRepo)

	// Initialize activity logger service (singleton)
	activityLogger := services.GetActivityLogger()
	mainLogger.Info("Activity logger service initialized")

	// Initialize email fetch scheduler
	schedulerConfig := services.DefaultSchedulerConfig()
	emailFetchScheduler := services.NewEmailFetchScheduler(fetcherService, emailAccountRepo, schedulerConfig)

	// Start the scheduler
	if err := emailFetchScheduler.Start(); err != nil {
		mainLogger.Error("Failed to start email fetch scheduler: %v", err)
		log.Fatalf("Failed to start email fetch scheduler: %v", err)
	}

	// 旧的队列式同步管理器已被替换为每账户独立goroutine方案
	// Initialize incremental sync manager (使用优化版实现) - 已禁用
	mainLogger.Info("正在跳过旧版队列式同步管理器（已被每账户独立goroutine方案替代）...")
	incrementalSyncManager := services.NewOptimizedIncrementalSyncManager(emailFetchScheduler, syncConfigRepo, emailRepo, mailboxRepo, fetcherService)
	// 不再启动旧系统: incrementalSyncManager.Start()
	mainLogger.Info("旧版同步管理器已禁用，新版每账户独立同步正在使用中")

	// Initialize subscription manager (needed for trigger service)
	subscriptionManager := services.NewSubscriptionManager()

	// Initialize trigger service
	mainLogger.Info("正在初始化触发器服务...")
	triggerService := services.NewTriggerService(triggerRepo, triggerLogRepo, emailRepo, subscriptionManager)
	if err := triggerService.Start(); err != nil {
		mainLogger.Error("Failed to start trigger service: %v", err)
		log.Fatalf("Failed to start trigger service: %v", err)
	}

	// Initialize Plugin Manager
	mainLogger.Info("Initializing plugin manager...")
	pluginManager := plugins.NewTriggerV2PluginManager(plugins.DefaultPluginManagerConfig())

	// 注册所有内置插件
	mainLogger.Info("Registering builtin plugins...")
	if err := builtin.RegisterBuiltinPlugins(pluginManager); err != nil {
		mainLogger.Error("Failed to register builtin plugins: %v", err)
	} else {
		mainLogger.Info("All builtin plugins registered successfully")
	}

	// Initialize EventBus and ConditionEngine
	eventBus := services.NewEventBus()
	conditionEngine := services.NewConditionEngine()

	// Initialize services.PluginManager for EmailTriggerService
	servicesPluginManager := services.NewPluginManager()

	// Initialize EmailTriggerService for V2
	emailTriggerService := services.NewEmailTriggerService(
		emailTriggerV2Repo,
		triggerExecutionLogV2Repo,
		subscriptionManager,
		eventBus,
		conditionEngine,
		servicesPluginManager,
	)

	// Initialize Email Notification Service
	mainLogger.Info("正在初始化邮件通知服务...")
	emailNotificationService := services.NewEmailNotificationService()

	// Initialize Per Account Sync Manager
	mainLogger.Info("正在初始化每账户同步管理器...")
	perAccountSyncManager := services.NewPerAccountSyncManager(
		syncConfigRepo,
		emailRepo,
		mailboxRepo,
		emailAccountRepo,
		fetcherService,
		emailNotificationService,
	)
	if err := perAccountSyncManager.Start(); err != nil {
		mainLogger.Error("Failed to start per-account sync manager: %v", err)
		return
	}

	// Initialize Account Recovery Service
	mainLogger.Info("正在初始化账户恢复服务...")
	accountRecoveryService := services.NewAccountRecoveryService(
		syncConfigRepo,
		emailAccountRepo,
		oauth2Service,
		perAccountSyncManager,
	)
	if err := accountRecoveryService.Start(); err != nil {
		mainLogger.Error("Failed to start account recovery service: %v", err)
		return
	}

	// Initialize API handler
	apiHandler := api.NewAPIHandler(fetcherService, parserService, emailAccountRepo, mailProviderRepo, emailRepo, incrementalSyncRepo, emailFetchScheduler, pluginManager, incrementalSyncManager, perAccountSyncManager)

	// Initialize OpenAI handler
	openAIHandler := api.NewOpenAIHandler(openAIConfigRepo, aiPromptTemplateRepo, extractorTemplateRepo)

	// Initialize Auth handler
	authHandler := api.NewAuthHandler(authService, userRepo)

	// Initialize Sync handlers（使用新的PerAccountSyncManager）
	syncHandlers := api.NewSyncHandlers(syncConfigRepo, perAccountSyncManager, mailboxRepo, fetcherService, emailAccountRepo, db)

	// Initialize Session handler
	sessionHandler := api.NewSessionHandler(authService)

	// Initialize Trigger handler
	triggerHandler := api.NewTriggerAPIHandler(triggerService, triggerRepo, triggerLogRepo, pluginManager)

	// Initialize OAuth2 handler
	oauth2Handler := api.NewOAuth2Handler(oauth2ConfigService, oauth2Service, oauth2AuthSessionService)

	// Initialize System Config service and handler
	systemConfigService := services.NewSystemConfigService(systemConfigRepo)
	systemConfigHandler := api.NewSystemConfigHandler(systemConfigService)

	// Initialize default system configurations
	if err := systemConfigService.InitializeDefaults(); err != nil {
		mainLogger.Warn("Failed to initialize default system configurations: %v", err)
	}

	// Initialize WebSocket handler
	mainLogger.Info("正在初始化WebSocket处理器...")
	webSocketHandler := api.NewWebSocketHandler(emailNotificationService)

	// Initialize default AI prompt templates
	if err := aiPromptTemplateRepo.InitializeDefaultTemplates(); err != nil {
		mainLogger.Warn("Failed to initialize default AI prompt templates: %v", err)
	}

	// Create router with authentication
	router := api.NewRouterWithAuth(
		apiHandler,
		openAIHandler,
		authHandler,
		syncHandlers,
		sessionHandler,
		triggerHandler,
		oauth2Handler,
		systemConfigHandler,
		webSocketHandler,
		authService,
		emailTriggerService,
		emailTriggerV2Repo,
		triggerExecutionLogV2Repo,
		servicesPluginManager,
		conditionEngine,
	)

	// Create HTTP server
	srv := &http.Server{
		Addr:    cfg.ServerAddress(),
		Handler: router,
	}

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		mainLogger.Info("Server is running on http://%s", cfg.ServerAddress())
		fmt.Printf("Server is running on http://%s\n", cfg.ServerAddress())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mainLogger.Error("Server failed to start: %v", err)
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	mainLogger.Info("Shutting down server...")
	fmt.Println("\nShutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop activity logger
	mainLogger.Info("Stopping activity logger...")
	activityLogger.Stop()

	// Stop trigger service first
	mainLogger.Info("Stopping trigger service...")
	triggerService.Stop()

	// Stop incremental sync manager
	mainLogger.Info("Stopping incremental sync manager...")
	incrementalSyncManager.Stop()

	// Stop email fetch scheduler
	mainLogger.Info("Stopping email fetch scheduler...")
	emailFetchScheduler.Stop()

	// Gracefully shutdown the HTTP server
	mainLogger.Info("Shutting down HTTP server...")
	if err := srv.Shutdown(ctx); err != nil {
		mainLogger.Error("Server forced to shutdown: %v", err)
	}

	mainLogger.Info("Server shutdown complete")
}
