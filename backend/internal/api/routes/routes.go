package routes

import (
	"backend/internal/api/handlers"
	"backend/internal/api/middleware"
	"log"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, authHandler *handlers.AuthHandler, chatHandler *handlers.ChatHandler, settingHandler *handlers.SettingHandler, videoHandler *handlers.VideoHandler, agentHandler *handlers.AgentHandler, aiChatHandler *handlers.AIChatHandler, aiChatWS *handlers.AIChatWebSocket, analyticsHandler *handlers.AnalyticsHandler, ttsHandler *handlers.TTSHandler, imageHandler *handlers.ImageHandler, ragHandler *handlers.RAGHandler, promptTemplateHandler *handlers.PromptTemplateHandler, branchHandler *handlers.BranchHandler, agentSystemHandler *handlers.AgentSystemHandler, jwtSecret string) {
	log.Println("Setting up routes...")
	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// WebSocket endpoint for agent (outside auth middleware - handles auth internally)
		v1.GET("/ws/agent", agentHandler.HandleWebSocket)

		// AI-AI Chat WebSocket (outside auth middleware - handles auth via query param)
		v1.GET("/ws/ai-chat/:id", aiChatWS.HandleWebSocket)

		// Protected routes
		v1.Use(middleware.AuthMiddleware(jwtSecret))

		chat := v1.Group("/chat")
		{
			chat.GET("/conversations", chatHandler.GetConversations)
			chat.POST("/conversations", chatHandler.CreateConversation)
			chat.GET("/conversations/:id/messages", chatHandler.GetMessages)
			chat.POST("/stream", chatHandler.StreamChat)
			chat.POST("/stream/rag", chatHandler.StreamChatWithRAG)
			chat.POST("/conversations/:id/summary", chatHandler.GenerateConversationSummary)
			chat.GET("/models", chatHandler.GetChatModels)
			chat.GET("/search-providers", chatHandler.GetSearchProviders)
			chat.GET("/mcp-tools", chatHandler.GetMCPTools)

			// Branch routes
			chat.GET("/conversations/:id/branches", branchHandler.GetBranches)
			chat.POST("/conversations/:id/branches", branchHandler.CreateBranch)
			chat.GET("/conversations/:id/branches/tree", branchHandler.GetBranchTree)
			chat.POST("/conversations/:id/branches/:branchId/switch", branchHandler.SwitchBranch)
			chat.DELETE("/conversations/:id/branches/:branchId", branchHandler.DeleteBranch)
			chat.GET("/branches/:branchId/messages", branchHandler.GetBranchMessages)
			chat.GET("/conversations/:id/message-tree", branchHandler.GetMessageTree)

			// Message editing routes
			chat.PUT("/messages/:msgId", branchHandler.EditMessage)
			chat.POST("/messages/:msgId/branch", branchHandler.EditAndBranch)
			chat.GET("/messages/:msgId/versions", branchHandler.GetMessageVersions)
			chat.POST("/messages/:msgId/revert", branchHandler.RevertMessage)
			chat.POST("/messages/:msgId/regenerate", branchHandler.RegenerateResponse)
			chat.POST("/messages/:msgId/regenerate/stream", branchHandler.RegenerateResponseStream)
			chat.POST("/messages/:msgId/multi-regenerate", branchHandler.MultiRegenerate)

			// Parallel exploration routes
			chat.POST("/conversations/:id/parallel-explore", branchHandler.StartParallelExploration)
			chat.GET("/conversations/:id/parallel-explorations", branchHandler.GetParallelExplorations)
			chat.GET("/parallel-explorations/:explorationId", branchHandler.GetParallelExploration)
			chat.POST("/conversations/:id/parallel-explorations/:explorationId/select", branchHandler.SelectExplorationBranch)
		}

		settings := v1.Group("/settings")
		{
			log.Println("Setting up settings routes...")

			// Get all settings
			settings.GET("", settingHandler.GetAllSettings)
			log.Println("Registered GET /settings")

			// Get settings by type
			settings.GET("/:type", settingHandler.GetSettingsByType)
			log.Println("Registered GET /settings/:type")

			// Models
			settings.PUT("/models/:key", settingHandler.UpdateModelProvider)
			settings.DELETE("/models/:key", settingHandler.DeleteModelProvider)
			settings.POST("/models/:key/test", settingHandler.TestModelProviderConnectivity)
			log.Println("Registered models routes")

			// Searchs
			settings.PUT("/searchs/:key", settingHandler.UpdateSearchProvider)
			settings.DELETE("/searchs/:key", settingHandler.DeleteSearchProvider)
			settings.POST("/searchs/:key/test", settingHandler.TestSearchProviderConnectivity)
			log.Println("Registered searchs routes")

			// MCP Servers
			settings.PUT("/mcpservers/:key", settingHandler.UpdateMCPServer)
			settings.DELETE("/mcpservers/:key", settingHandler.DeleteMCPServer)
			settings.POST("/mcpservers/:key/test", settingHandler.TestMCPServerConnectivity)
			log.Println("Registered mcpservers routes")

			// Skills
			settings.PUT("/skills/:key", settingHandler.UpdateSkill)
			settings.DELETE("/skills/:key", settingHandler.DeleteSkill)
			log.Println("Registered skills routes")

			// Agents
			settings.PUT("/agents/:key", settingHandler.UpdateAgent)
			settings.DELETE("/agents/:key", settingHandler.DeleteAgent)
			log.Println("Registered agents routes")

			// Legacy routes for backward compatibility
			settings.GET("/ai-providers", settingHandler.GetProviderSettingsLegacy)
			settings.POST("/ai-providers", settingHandler.SaveProviderSettingLegacy)
			settings.DELETE("/ai-providers/:provider", settingHandler.DeleteProviderSettingLegacy)
			log.Println("Registered legacy routes")
		}

		// Simple test route directly under v1
		v1.GET("/ping", func(c *gin.Context) {
			log.Println("Ping route called")
			c.JSON(200, gin.H{"message": "pong"})
		})

		video := v1.Group("/video")
		{
			video.POST("/generate", videoHandler.Generate)
			video.GET("/tasks", videoHandler.ListTasks)
			video.GET("/tasks/:id", videoHandler.GetTask)
			video.DELETE("/tasks/:id", videoHandler.DeleteTask)
			video.GET("/models", videoHandler.GetModels)
			video.GET("/providers", videoHandler.GetProviders)
		}

		// Agent routes
		v1.GET("/agents", agentHandler.ListAgents)
		v1.GET("/agents/:name", agentHandler.GetAgent)
		v1.PUT("/agents/:name", agentHandler.UpdateAgent)
		v1.GET("/mcp/tools", agentHandler.ListMCPTools)
		v1.GET("/skills", agentHandler.ListSkills)
		v1.POST("/skills/:name/execute", agentHandler.ExecuteSkill)

		// Enhanced Agent Routes (Priority 1-3 Features)
		enhanced := v1.Group("/agent-enhanced")
		{
			// Codebase context (P1)
			enhanced.GET("/codebase/context", agentHandler.GetCodebaseContext)
			enhanced.POST("/codebase/reindex", agentHandler.ReindexCodebase)
			enhanced.GET("/codebase/affected", agentHandler.GetAffectedFiles)

			// Human-in-loop approvals (P1)
			enhanced.GET("/approvals", agentHandler.GetPendingApprovals)
			enhanced.POST("/approvals/submit", agentHandler.SubmitApproval)
			enhanced.POST("/approvals/settings", agentHandler.SetApprovalRequired)

			// Multi-file changes (P2)
			enhanced.GET("/changes", agentHandler.GetProposedChanges)
			enhanced.GET("/changes/diff", agentHandler.GetUnifiedDiff)
			enhanced.POST("/changes/approve-all", agentHandler.ApproveAllChanges)

			// Git operations (P3)
			enhanced.GET("/git/status", agentHandler.GetGitStatus)
			enhanced.POST("/git/branch", agentHandler.CreateTaskBranch)
			enhanced.POST("/git/commit", agentHandler.CommitChanges)
			enhanced.POST("/git/prepare-pr", agentHandler.PreparePullRequest)

			// LSP/Syntax checking (P3)
			enhanced.GET("/syntax/check", agentHandler.CheckSyntax)
			enhanced.GET("/syntax/check-all", agentHandler.CheckAllSyntax)
		}

		// AI-AI Chat routes
		aichat := v1.Group("/ai-chat")
		{
			aichat.GET("/sessions", aiChatHandler.ListSessions)
			aichat.POST("/sessions", aiChatHandler.CreateSession)
			aichat.GET("/sessions/:id", aiChatHandler.GetSession)
			aichat.DELETE("/sessions/:id", aiChatHandler.DeleteSession)
			aichat.GET("/sessions/:id/status", aiChatHandler.GetSessionStatus)
			aichat.POST("/sessions/:id/start", aiChatHandler.StartSession)
			aichat.POST("/sessions/:id/pause", aiChatHandler.PauseSession)
			aichat.POST("/sessions/:id/stop", aiChatHandler.StopSession)
			aichat.POST("/sessions/:id/director-command", aiChatHandler.InjectDirectorCommand)
			aichat.POST("/sessions/:id/branch", aiChatHandler.CreateBranch)
			aichat.POST("/sessions/:id/snapshot", aiChatHandler.CreateSnapshot)
			aichat.GET("/sessions/:id/export", aiChatHandler.ExportSession)
			aichat.GET("/sessions/:id/evaluation", aiChatHandler.GetEvaluation)
			aichat.GET("/templates", aiChatHandler.GetTemplates)
			aichat.GET("/templates/:id", aiChatHandler.GetTemplate)
			aichat.POST("/templates", aiChatHandler.CreateTemplate)
			aichat.PUT("/templates/:id", aiChatHandler.UpdateTemplate)
			aichat.DELETE("/templates/:id", aiChatHandler.DeleteTemplate)
			aichat.POST("/templates/:id/clone", aiChatHandler.CloneTemplate)
			aichat.GET("/models", aiChatHandler.GetModels)
			aichat.GET("/mcp-tools", aiChatHandler.GetMCPTools)
		}

		// Analytics routes
		analytics := v1.Group("/analytics")
		{
			analytics.GET("/usage", analyticsHandler.GetUsageSummary)
			analytics.GET("/daily", analyticsHandler.GetDailyUsage)
			analytics.GET("/costs", analyticsHandler.GetCostBreakdown)
			analytics.GET("/stats", analyticsHandler.GetTotalStats)
			analytics.GET("/recent", analyticsHandler.GetRecentUsage)
		}

		// TTS routes
		ttsGroup := v1.Group("/tts")
		{
			ttsGroup.POST("/speak", ttsHandler.Speak)
			ttsGroup.GET("/voices", ttsHandler.GetVoices)
			ttsGroup.GET("/models", ttsHandler.GetModels)
		}

		// Image Generation routes
		imageGroup := v1.Group("/image")
		{
			imageGroup.POST("/generate", imageHandler.Generate)
			imageGroup.GET("/history", imageHandler.GetHistory)
			imageGroup.GET("/:id", imageHandler.GetImage)
			imageGroup.DELETE("/:id", imageHandler.DeleteImage)
			imageGroup.GET("/models", imageHandler.GetModels)
			imageGroup.GET("/providers", imageHandler.GetProviders)
			imageGroup.POST("/variations", imageHandler.CreateVariation)
			imageGroup.POST("/edit", imageHandler.EditImage)
		}

		// RAG/Knowledge Base routes
		ragGroup := v1.Group("/rag")
		{
			ragGroup.POST("/documents", ragHandler.UploadDocument)
			ragGroup.GET("/documents", ragHandler.ListDocuments)
			ragGroup.GET("/documents/:id", ragHandler.GetDocument)
			ragGroup.DELETE("/documents/:id", ragHandler.DeleteDocument)
			ragGroup.GET("/documents/:id/chunks", ragHandler.GetDocumentChunks)
			ragGroup.POST("/query", ragHandler.Query)
		}

		// Prompt Template routes
		promptGroup := v1.Group("/prompt-templates")
		{
			promptGroup.GET("", promptTemplateHandler.GetAllTemplates)
			promptGroup.GET("/categories", promptTemplateHandler.GetCategories)
			promptGroup.GET("/categories/:id", promptTemplateHandler.GetCategory)
			promptGroup.GET("/templates/:id", promptTemplateHandler.GetTemplate)
			promptGroup.POST("/render", promptTemplateHandler.RenderTemplate)
			promptGroup.GET("/search", promptTemplateHandler.SearchTemplates)
		}

		// Agent System routes - Custom Agents, Workflows, Permissions, Marketplace
		agentSystem := v1.Group("/agent-system")
		{
			// Custom Agents
			agents := agentSystem.Group("/agents")
			{
				agents.GET("/templates", agentSystemHandler.GetAgentTemplates)
				agents.GET("", agentSystemHandler.ListAgents)
				agents.POST("", agentSystemHandler.CreateAgent)
				agents.GET("/:id", agentSystemHandler.GetAgent)
				agents.PUT("/:id", agentSystemHandler.UpdateAgent)
				agents.DELETE("/:id", agentSystemHandler.DeleteAgent)
				agents.POST("/:id/duplicate", agentSystemHandler.DuplicateAgent)
				agents.POST("/:id/execute", agentSystemHandler.ExecuteAgent)
				agents.GET("/:id/executions", agentSystemHandler.GetAgentExecutionHistory)
				agents.POST("/executions/:executionId/feedback", agentSystemHandler.ProvideFeedback)
				agents.GET("/:id/export", agentSystemHandler.ExportAgent)
				agents.POST("/import", agentSystemHandler.ImportAgent)
				agents.POST("/:id/publish", agentSystemHandler.PublishAgentToMarketplace)
			}

			// Workflows
			workflows := agentSystem.Group("/workflows")
			{
				workflows.GET("", agentSystemHandler.ListWorkflows)
				workflows.POST("", agentSystemHandler.CreateWorkflow)
				workflows.GET("/:id", agentSystemHandler.GetWorkflow)
				workflows.PUT("/:id", agentSystemHandler.UpdateWorkflow)
				workflows.DELETE("/:id", agentSystemHandler.DeleteWorkflow)
				workflows.POST("/:id/steps", agentSystemHandler.AddWorkflowStep)
				workflows.PUT("/steps/:stepId", agentSystemHandler.UpdateWorkflowStep)
				workflows.DELETE("/steps/:stepId", agentSystemHandler.DeleteWorkflowStep)
				workflows.POST("/:id/edges", agentSystemHandler.AddWorkflowEdge)
				workflows.DELETE("/edges/:edgeId", agentSystemHandler.DeleteWorkflowEdge)
				workflows.POST("/:id/start", agentSystemHandler.StartWorkflow)
				workflows.GET("/runs/:runId", agentSystemHandler.GetWorkflowRun)
				workflows.GET("/:id/runs", agentSystemHandler.GetWorkflowRunHistory)
				workflows.POST("/runs/:runId/resume", agentSystemHandler.ResumeWorkflow)
				workflows.POST("/runs/:runId/cancel", agentSystemHandler.CancelWorkflow)
				workflows.GET("/:id/export", agentSystemHandler.ExportWorkflow)
				workflows.POST("/import", agentSystemHandler.ImportWorkflow)
				workflows.POST("/:id/publish", agentSystemHandler.PublishWorkflowToMarketplace)
			}

			// Permissions
			permissions := agentSystem.Group("/permissions")
			{
				permissions.GET("", agentSystemHandler.ListPermissions)
				permissions.POST("", agentSystemHandler.CreatePermission)
				permissions.GET("/:id", agentSystemHandler.GetPermission)
				permissions.PUT("/:id", agentSystemHandler.UpdatePermission)
				permissions.DELETE("/:id", agentSystemHandler.DeletePermission)
				permissions.POST("/check", agentSystemHandler.CheckPermission)
				permissions.GET("/logs", agentSystemHandler.GetInvocationLogs)
				permissions.GET("/stats", agentSystemHandler.GetUsageStats)
				permissions.POST("/defaults", agentSystemHandler.CreateDefaultPermissions)
			}

			// Marketplace
			marketplace := agentSystem.Group("/marketplace")
			{
				marketplace.GET("/search", agentSystemHandler.SearchMarketplace)
				marketplace.GET("/categories", agentSystemHandler.GetMarketplaceCategories)
				marketplace.GET("/featured", agentSystemHandler.GetFeaturedItems)
				marketplace.GET("/trending", agentSystemHandler.GetTrendingItems)
				marketplace.GET("/items/:id", agentSystemHandler.GetMarketplaceItem)
				marketplace.POST("/items/:id/download", agentSystemHandler.DownloadMarketplaceItem)
				marketplace.POST("/items/:id/star", agentSystemHandler.StarMarketplaceItem)
				marketplace.DELETE("/items/:id/star", agentSystemHandler.UnstarMarketplaceItem)
				marketplace.POST("/items/:id/reviews", agentSystemHandler.AddMarketplaceReview)
				marketplace.GET("/items/:id/reviews", agentSystemHandler.GetMarketplaceReviews)
				marketplace.POST("/items/:id/fork", agentSystemHandler.ForkMarketplaceItem)
				marketplace.GET("/my-items", agentSystemHandler.GetMyMarketplaceItems)
			}

			// A/B Testing
			abtests := agentSystem.Group("/ab-tests")
			{
				abtests.GET("", agentSystemHandler.ListABTests)
				abtests.POST("", agentSystemHandler.CreateABTest)
				abtests.GET("/:id", agentSystemHandler.GetABTest)
				abtests.POST("/:id/start", agentSystemHandler.StartABTest)
			}
		}
	}
}
