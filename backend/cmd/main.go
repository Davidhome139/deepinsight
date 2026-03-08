package main

import (
	"context"
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
	// 1. Load all configs
	configManager := config.NewConfigManager("config")
	if err := configManager.LoadAll(); err != nil {
		log.Fatalf("Failed to load configs: %v", err)
	}

	// Load main config for other services
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load main config: %v", err)
	}

	// 2. Initialize DB & Cache (non-fatal if fails)
	_, err = database.InitDB(&cfg.Database.Postgres)
	if err != nil {
		log.Printf("Warning: Failed to initialize database: %v", err)
		log.Println("Continuing without database connection...")
	} else {
		log.Println("Database connection established")
	}

	_, err = cache.InitRedis(&cfg.Database.Redis)
	if err != nil {
		log.Printf("Warning: Failed to initialize redis: %v", err)
		log.Println("Continuing without Redis connection...")
	} else {
		log.Println("Redis connection established")
	}

	// Get JWT secret from config or use default
	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" {
		jwtSecret = "3wKknT3f4H7KUrjn4W3Grn80yWxh2TUaF84DLstvlv4="
	}

	// 3. Initialize Services
	searchService := search.NewSearchManagerFromConfig()
	aiService := ai.NewAIManager()
	authService := auth.NewAuthService(jwtSecret)

	// Initialize Agent System first (creates MCPManager)
	llmClient := llm.NewClientWithService(aiService) // Uses DeepSeek reasoning by default
	mcpManager := agent.NewMCPManager()
	skillRegistry := agent.NewSkillRegistry()
	orchestrator := agent.NewOrchestrator(llmClient, mcpManager, skillRegistry)

	// Initialize Chat Service with MCPManager
	chatService := chat.NewChatService(aiService, searchService, mcpManager)

	// Initialize AI-AI Chat Service
	aiChatService := aichat.NewAIChatService(aiService, searchService, mcpManager)

	// Get Aliyun DashScope API key from models.yaml config (for RAG, TTS, Image services)
	aliyunAPIKey := ""
	modelsConfig := config.GetModelsConfig()
	if modelsConfig != nil {
		if aliyunConfig, ok := modelsConfig.Providers["aliyun"]; ok && aliyunConfig.Enabled {
			aliyunAPIKey = aliyunConfig.APIKey
		}
	}

	// Initialize new services (Analytics, TTS, Image, RAG) - using Aliyun DashScope
	analyticsService := analytics.NewAnalyticsService(database.DB)
	analyticsCollector := analytics.NewCollector(database.DB)
	ttsService := tts.NewTTSService(aliyunAPIKey)
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

	// 4. Initialize Handlers
	authHandler := handlers.NewAuthHandler(authService)
	chatHandler := handlers.NewChatHandler(chatService)
	aiChatHandler := handlers.NewAIChatHandler(aiChatService)
	aiChatWS := handlers.NewAIChatWebSocket(aiChatService, jwtSecret)
	settingHandler := handlers.NewSettingHandler()
	videoHandler := handlers.NewVideoHandler(database.DB)
	agentHandler := handlers.NewAgentHandler(orchestrator, jwtSecret)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	ttsHandler := handlers.NewTTSHandler(ttsService)
	imageHandler := handlers.NewImageHandler(imageService)
	ragHandler := handlers.NewRAGHandler(ragService)
	promptTemplateHandler := handlers.NewPromptTemplateHandler()
	branchHandler := handlers.NewBranchHandler(branchService)
	agentSystemHandler := handlers.NewAgentSystemHandler(customAgentService, workflowEngine, permissionService, marketplaceService, abTestService)
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
	routes.SetupRoutes(r, authHandler, chatHandler, settingHandler, videoHandler, agentHandler, aiChatHandler, aiChatWS, analyticsHandler, ttsHandler, imageHandler, ragHandler, promptTemplateHandler, branchHandler, agentSystemHandler, jwtSecret)

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
		log.Println("Services stopped")
	}()

	// 8. Start Server
	log.Printf("Starting server on :%d", cfg.Server.Port)
	r.Run(":8080")
}
