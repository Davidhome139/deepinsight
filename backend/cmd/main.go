package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"backend/internal/api/handlers"
	"backend/internal/api/routes"
	"backend/internal/config"
	"backend/internal/pkg/cache"
	"backend/internal/pkg/database"
	"backend/internal/pkg/llm"
	"backend/internal/services/agent"
	"backend/internal/services/agentsystem"
	"backend/internal/services/ai"
	"backend/internal/services/aichat"
	"backend/internal/services/analytics"
	"backend/internal/services/auth"
	"backend/internal/services/branch"
	"backend/internal/services/chat"
	"backend/internal/services/image"
	"backend/internal/services/rag"
	"backend/internal/services/scheduler"
	"backend/internal/services/search"
	"backend/internal/services/tts"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("==========================================")
	log.Println("[Main] Starting application...")
	log.Println("==========================================")
	var err error
	// 1. Load all configs
	log.Println("[Main] Loading configs...")
	configManager := config.NewConfigManager("config")
	if err := configManager.LoadAll(); err != nil {
		log.Fatalf("Failed to load configs: %v", err)
	}
	log.Println("[Main] Configs loaded successfully")

	// Debug: Check if GlobalConfig is loaded
	if config.GlobalConfig == nil {
		log.Println("[DEBUG] GlobalConfig is nil after loading!")
	} else {
		log.Printf("[DEBUG] GlobalConfig loaded: Server.Port=%d, Server.HTTPS.Enabled=%v",
			config.GlobalConfig.Server.Port, config.GlobalConfig.Server.HTTPS.Enabled)
	}

	// 2. Initialize DB & Cache (non-fatal if fails)
	dbConfig := config.GetPostgresConfig()
	if dbConfig == nil {
		log.Println("Warning: Failed to get database config")
		log.Println("Continuing without database connection...")
	} else {
		_, err = database.InitDB(dbConfig)
		if err != nil {
			log.Printf("Warning: Failed to initialize database: %v", err)
			log.Println("Continuing without database connection...")
		} else {
			log.Println("Database connection established")
		}
	}

	// Get Redis config from GlobalConfig
	if config.GlobalConfig != nil {
		_, err = cache.InitRedis(&config.GlobalConfig.Database.Redis)
		if err != nil {
			log.Printf("Warning: Failed to initialize redis: %v", err)
			log.Println("Continuing without Redis connection...")
		} else {
			log.Println("Redis connection established")
		}
	} else {
		log.Println("Warning: GlobalConfig is nil, skipping Redis initialization")
	}

	// Get JWT secret from config
	jwtSecret := ""
	if config.GlobalConfig != nil {
		jwtSecret = config.GlobalConfig.JWT.Secret
	}
	if jwtSecret == "" {
		log.Fatalf("JWT secret is required in config or via environment variable APP_JWT_SECRET")
	}

	// 3. Initialize Services
	log.Println("[Main] Initializing search service...")
	searchService := search.NewSearchManagerFromConfig()
	log.Println("[Main] Initializing AI service...")
	aiService := ai.NewAIManager()
	log.Println("[Main] Initializing auth service...")
	authService := auth.NewAuthService(jwtSecret)

	// Initialize Agent System first (creates MCPManager)
	log.Println("[Main] Initializing LLM client...")
	llmClient := llm.NewClientWithService(aiService) // Uses DeepSeek reasoning by default
	log.Println("==========================================")
	log.Println("[Main] Creating MCPManager...")

	// Force MCP manager creation with error handling
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Main] PANIC during MCPManager creation: %v", r)
		}
	}()

	// 使用log.Println确保输出被捕获到Docker日志中
	log.Println("==========================================")
	log.Println("[Main] Creating MCPManager...")
	mcpManager := agent.NewMCPManager()
	if mcpManager == nil {
		log.Println("[Main] ERROR: MCPManager is nil!")
		log.Fatal("[Main] Failed to create MCPManager - returned nil")
	}
	log.Println("[Main] MCPManager created successfully")

	// Set the MCP manager in config package for immediate connection on enable
	config.SetMCPManager(mcpManager)
	log.Println("[Main] MCPManager set in config package")

	// Immediately discover and connect to all enabled MCP servers
	log.Println("[Main] Starting immediate MCP discovery...")
	mcpManager.Discover()
	log.Println("[Main] MCP discovery started")

	log.Println("==========================================")
	log.Println("[Main] Creating skill registry...")
	skillRegistry := agent.NewSkillRegistry()
	log.Println("[Main] Creating orchestrator...")
	orchestrator := agent.NewOrchestrator(llmClient, mcpManager, skillRegistry)

	// Initialize Chat Service with MCPManager
	chatService := chat.NewChatService(aiService, searchService, mcpManager)

	// Initialize AI-AI Chat Service
	aiChatService := aichat.NewAIChatService(aiService, searchService, mcpManager)

	// Get Aliyun DashScope API key from models.yaml config (for RAG, TTS, Image services)
	aliyunAPIKey := ""
	aliyunVoiceToken := ""
	aliyunVoiceAppKey := ""
	modelsConfig := config.GetModelsConfig()
	if modelsConfig != nil {
		if aliyunConfig, ok := modelsConfig.Providers["aliyun"]; ok && aliyunConfig.Enabled {
			aliyunAPIKey = aliyunConfig.APIKey
		}
	}

	// Get voice token and appkey from environment variables
	aliyunVoiceToken = config.GetEnvWithDefault("MODELS_PROVIDERS_ALIYUN_VOICE_TOKEN", "")
	aliyunVoiceAppKey = config.GetEnvWithDefault("MODELS_PROVIDERS_ALIYUN_VOICE_APPKEY", "")

	// Initialize new services (Analytics, TTS, Image, RAG) - using Aliyun DashScope
	analyticsService := analytics.NewAnalyticsService(database.DB)
	analyticsCollector := analytics.NewCollector(database.DB)
	ttsService := tts.NewTTSService(aliyunAPIKey, aliyunVoiceToken, aliyunVoiceAppKey)
	imageService := image.NewImageService(database.DB, aliyunAPIKey)
	ragService := rag.NewRAGService(database.DB, aliyunAPIKey)

	// Initialize Scheduler and Maintenance Tasks
	schedulerService := scheduler.New(nil)
	maintenanceTasks := scheduler.NewMaintenanceTasks(database.DB, ".")
	if err := maintenanceTasks.RegisterAll(schedulerService); err != nil {
		log.Printf("Warning: Failed to register maintenance tasks: %v", err)
	}

	// Initialize Branch Service for conversation branching and parallel exploration
	branchService := branch.NewBranchService(aiService)

	// Initialize Agent System Services (Custom Agents, Workflows, Permissions, Marketplace)
	customAgentService := agentsystem.NewCustomAgentService(database.DB, llmClient)
	workflowEngine := agentsystem.NewWorkflowEngine(database.DB, customAgentService, mcpManager)
	permissionService := agentsystem.NewPermissionService(database.DB)
	marketplaceService := agentsystem.NewMarketplaceService(database.DB, customAgentService, workflowEngine)
	abTestService := agentsystem.NewABTestService(database.DB, customAgentService)

	// Initialize MCP Automation Coordinator (disabled for now to fix server startup)
	log.Println("[Main] Initializing MCP automation coordinator...")

	// Create dependency manager factory and adapter (disabled for now)
	// depManagerFactory := mcp.NewDependencyManagerFactory()
	// depManager := mcp.NewFactoryDependencyManager(depManagerFactory)

	// Create documentation fetcher factory and adapter (disabled for now)
	// docFetcherFactory := mcp.NewDocumentationFetcherFactory()
	// docFetcher := mcp.NewFactoryDocumentationFetcher(docFetcherFactory)

	// Create config generator (disabled for now)
	// configGenerator := mcp.NewDefaultConfigGenerator()

	// Create hot reload manager (disabled for now)
	// configDir := config.GetConfigDir()
	// hotReloadManager, err := mcp.NewDefaultHotReloadManager(mcpManager, configDir+"/mcpservers.json")
	// if err != nil {
	// 	log.Printf("Warning: Failed to create hot reload manager: %v", err)
	// 	log.Println("Continuing without hot reload...")
	// 	hotReloadManager = nil
	// }

	// Create automation coordinator (disabled for now to fix server startup)
	// coordinator := mcp.NewDefaultAutomationCoordinator(
	// 	depManager,
	// 	docFetcher,
	// 	configGenerator,
	// 	hotReloadManager,
	// 	mcpManager,
	// 	configDir+"/mcpservers.json",
	// 	configDir+"/mcp_docs",
	// )

	// Start automation coordinator (disabled for now to fix server startup)
	ctx := context.Background()
	log.Println("[Main] MCP automation coordinator initialization skipped for now")
	// Temporarily disable coordinator to fix server startup
	// if err := coordinator.Start(ctx); err != nil {
	// 	log.Printf("Warning: Failed to start MCP automation coordinator: %v", err)
	// 	log.Println("Continuing without MCP automation...")
	// } else {
	// 	log.Println("[Main] MCP automation coordinator started successfully")
	// }

	// 4. Initialize Handlers
	authHandler := handlers.NewAuthHandler(authService)
	chatHandler := handlers.NewChatHandler(chatService)
	// Set MCP manager for chat handler
	chatHandler.SetMCPManager(mcpManager)
	aiChatHandler := handlers.NewAIChatHandler(aiChatService)
	aiChatWS := handlers.NewAIChatWebSocket(aiChatService, jwtSecret)
	// Pass nil coordinator for now since it's disabled
	settingHandler := handlers.NewSettingHandler(nil)
	videoHandler := handlers.NewVideoHandler(database.DB)
	agentHandler := handlers.NewAgentHandler(orchestrator, jwtSecret)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	ttsHandler := handlers.NewTTSHandler(ttsService)
	imageHandler := handlers.NewImageHandler(imageService)
	ragHandler := handlers.NewRAGHandler(ragService)
	promptTemplateHandler := handlers.NewPromptTemplateHandler()
	branchHandler := handlers.NewBranchHandler(branchService)
	agentSystemHandler := handlers.NewAgentSystemHandler(customAgentService, workflowEngine, permissionService, marketplaceService, abTestService)
	// Pass nil coordinator for now since it's disabled
	mcpAutomationHandler := handlers.NewMCPAutomationHandler(nil)
	healthHandler := handlers.NewHealthHandler(database.DB, cache.RedisClient)

	// 5. Setup Gin
	r := gin.Default()
	// Set trusted proxies (empty = don't trust any, or specify IP ranges)
	r.SetTrustedProxies(nil)
	// Debug middleware to log routes
	r.Use(func(c *gin.Context) {
		log.Printf("Request: %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
	})
	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
	routes.SetupRoutes(r, authHandler, chatHandler, settingHandler, videoHandler, agentHandler, aiChatHandler, aiChatWS, analyticsHandler, ttsHandler, imageHandler, ragHandler, promptTemplateHandler, branchHandler, agentSystemHandler, mcpAutomationHandler, jwtSecret)

	// Register Health endpoints (outside auth middleware)
	healthGroup := r.Group("/api/v1/health")
	{
		healthGroup.GET("/live", healthHandler.LivenessProbe)
		healthGroup.GET("/ready", healthHandler.ReadinessProbe)
		healthGroup.GET("", healthHandler.HealthCheck)
		healthGroup.GET("/metrics", healthHandler.GetMetrics())
		healthGroup.GET("/system", healthHandler.GetSystemInfo)
	}

	// Apply metrics middleware
	r.Use(healthHandler.MetricsMiddleware())

	// Print all registered routes
	log.Println("=== Registered Routes ===")
	for _, route := range r.Routes() {
		log.Printf("%s %s", route.Method, route.Path)
	}
	log.Println("=== End Routes ===")

	// Add NoRoute handler for debugging
	r.NoRoute(func(c *gin.Context) {
		log.Printf("NoRoute: %s %s", c.Request.Method, c.Request.URL.Path)
		c.JSON(404, gin.H{"error": "Not found", "method": c.Request.Method, "path": c.Request.URL.Path})
	})

	// 6. Start Background Services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start analytics collector
	analyticsCollector.Start(ctx)
	log.Println("Analytics collector started")

	// Start scheduler
	schedulerService.Start()
	log.Println("Scheduler started")

	// 7. Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		cancel()
		analyticsCollector.Stop()
		schedulerService.Stop()
		// Stop MCP automation coordinator (disabled for now)
		// if err := coordinator.Stop(); err != nil {
		// 	log.Printf("Warning: Failed to stop MCP automation coordinator: %v", err)
		// } else {
		// 	log.Println("MCP automation coordinator stopped")
		// }
		log.Println("Services stopped")
	}()

	// 8. Start Server
	port := 8080
	httpsEnabled := false
	httpsPort := 8443
	httpsCertFile := ""
	httpsKeyFile := ""

	if config.GlobalConfig != nil {
		port = config.GlobalConfig.Server.Port
		httpsEnabled = config.GlobalConfig.Server.HTTPS.Enabled
		httpsPort = config.GlobalConfig.Server.HTTPS.Port
		httpsCertFile = config.GlobalConfig.Server.HTTPS.CertFile
		httpsKeyFile = config.GlobalConfig.Server.HTTPS.KeyFile

		// Debug logging
		log.Printf("[DEBUG] Server config - Port: %d, HTTPS Enabled: %v, HTTPS Port: %d", port, httpsEnabled, httpsPort)
		log.Printf("[DEBUG] HTTPS Cert File: %s, Key File: %s", httpsCertFile, httpsKeyFile)
	} else {
		log.Printf("[DEBUG] GlobalConfig is nil, using default values")
	}

	if httpsEnabled {
		log.Printf("Starting HTTPS server on :%d", httpsPort)
		r.RunTLS(fmt.Sprintf(":%d", httpsPort), httpsCertFile, httpsKeyFile)
	} else {
		log.Printf("Starting HTTP server on :%d", port)
		r.Run(fmt.Sprintf(":%d", port))
	}
}
