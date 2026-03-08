package aichat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"backend/internal/models"
	"backend/internal/pkg/database"
	"backend/internal/services/agent"
	"backend/internal/services/ai"
	"backend/internal/services/search"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AIChatService AI-AI 聊天服务
type AIChatService struct {
	db                 *gorm.DB
	aiService          ai.AIService
	searchService      search.SearchService
	mcpManager         *agent.MCPManager
	evaluator          *DialogEvaluator
	sessions           map[string]*RunningSession
	pendingSubscribers map[string][]chan StreamEvent
	mu                 sync.RWMutex
}

// RunningSession 运行中的会话
type RunningSession struct {
	Session         *models.AIChatSession
	CancelFunc      context.CancelFunc
	Clients         map[chan StreamEvent]bool
	PendingCommands []*models.DirectorCommand
	LastSearchRound int
	SearchCount     int
	mu              sync.RWMutex
}

// StreamEvent WebSocket 流事件
type StreamEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// NewAIChatService 创建 AI-AI 聊天服务
func NewAIChatService(aiService ai.AIService, searchService search.SearchService, mcpManager *agent.MCPManager) *AIChatService {
	service := &AIChatService{
		db:                 database.DB,
		sessions:           make(map[string]*RunningSession),
		pendingSubscribers: make(map[string][]chan StreamEvent),
		aiService:          aiService,
		searchService:      searchService,
		mcpManager:         mcpManager,
		evaluator:          NewDialogEvaluator(database.DB, aiService),
	}

	// 自动迁移数据库表
	service.db.AutoMigrate(
		&models.AIChatSession{},
		&models.AIChatMessage{},
		&models.DirectorCommand{},
		&models.SessionSnapshot{},
		&models.EvaluationReport{},
		&models.AuditLog{},
		&models.SessionTemplate{},
	)

	// 从 JSON 配置文件加载内置模板
	NewTemplateService()

	return service
}

// CreateSession 创建会话
func (s *AIChatService) CreateSession(config SessionConfig) (*models.AIChatSession, error) {
	// Sync termination max_rounds with top-level max_rounds for fixed_rounds type
	if config.MaxRounds > 0 {
		if config.TerminationConfig.MaxRounds == 0 || config.TerminationConfig.Type == models.TerminationTypeFixedRounds {
			config.TerminationConfig.MaxRounds = config.MaxRounds
		}
	}
	// Default termination type
	if config.TerminationConfig.Type == "" {
		config.TerminationConfig.Type = models.TerminationTypeFixedRounds
	}

	session := &models.AIChatSession{
		ID:                uuid.New().String(),
		Title:             config.Title,
		Status:            models.SessionStatusPending,
		Topic:             config.Topic,
		GlobalConstraint:  config.GlobalConstraint,
		MaxRounds:         config.MaxRounds,
		CurrentRound:      0,
		TerminationConfig: config.TerminationConfig,
	}

	// Multi-agent mode (3+ agents)
	if len(config.Agents) > 2 {
		// Set default models for agents without one
		for i := range config.Agents {
			if config.Agents[i].Model == "" {
				config.Agents[i].Model = "deepseek-chat"
			}
		}
		session.Agents = config.Agents
		session.AgentCount = len(config.Agents)
		session.SpeakingOrder = config.SpeakingOrder
		session.CurrentSpeaker = 0

		// Also set agent_a and agent_b for backward compatibility
		if len(config.Agents) >= 1 {
			session.SetAgentConfig("agent_a", config.Agents[0])
		}
		if len(config.Agents) >= 2 {
			session.SetAgentConfig("agent_b", config.Agents[1])
		}
	} else {
		// Legacy 2-agent mode
		session.AgentCount = 2

		// 设置默认值：如果未指定模型，使用 deepseek-chat
		if config.AgentA.Model == "" {
			config.AgentA.Model = "deepseek-chat"
		}
		if config.AgentB.Model == "" {
			config.AgentB.Model = "deepseek-chat"
		}

		// 设置 Agent A
		session.SetAgentConfig("agent_a", config.AgentA)

		// 设置 Agent B
		session.SetAgentConfig("agent_b", config.AgentB)
	}

	if err := s.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// 记录审计日志
	s.logAudit(session.ID, "session_created", map[string]interface{}{
		"title":      config.Title,
		"topic":      config.Topic,
		"maxRounds":  config.MaxRounds,
		"agentCount": session.AgentCount,
	})

	return session, nil
}

// GetSession 获取会话
func (s *AIChatService) GetSession(id string) (*models.AIChatSession, error) {
	var session models.AIChatSession
	if err := s.db.Preload("Messages").First(&session, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// ListSessions 列会话
func (s *AIChatService) ListSessions(filter SessionFilter) ([]*models.AIChatSession, error) {
	var sessions []*models.AIChatSession
	query := s.db.Model(&models.AIChatSession{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ParentID != nil {
		query = query.Where("parent_id = ?", *filter.ParentID)
	}

	if err := query.Order("created_at DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}

	return sessions, nil
}

// DeleteSession 删除会话
func (s *AIChatService) DeleteSession(id string) error {
	// 如果会话正在运行，先停止
	s.mu.Lock()
	if running, ok := s.sessions[id]; ok {
		if running.CancelFunc != nil {
			running.CancelFunc()
		}
		delete(s.sessions, id)
	}
	s.mu.Unlock()

	// 删除数据库记录
	return s.db.Delete(&models.AIChatSession{}, "id = ?", id).Error
}

// StartSession 开始会话
func (s *AIChatService) StartSession(id string) error {
	session, err := s.GetSession(id)
	if err != nil {
		return err
	}

	if session.Status != models.SessionStatusPending && session.Status != models.SessionStatusPaused {
		return fmt.Errorf("session cannot be started from status: %s", session.Status)
	}

	// 更新状态
	now := time.Now()
	session.Status = models.SessionStatusRunning
	if session.StartedAt == nil {
		session.StartedAt = &now
	}

	if err := s.db.Save(session).Error; err != nil {
		return err
	}

	// 创建运行时会话
	ctx, cancel := context.WithCancel(context.Background())
	running := &RunningSession{
		Session:         session,
		CancelFunc:      cancel,
		Clients:         make(map[chan StreamEvent]bool),
		PendingCommands: make([]*models.DirectorCommand, 0),
		LastSearchRound: -1,
		SearchCount:     0,
	}

	s.mu.Lock()
	s.sessions[id] = running
	// Flush any pending subscribers that connected before session started
	if pending, ok := s.pendingSubscribers[id]; ok {
		running.mu.Lock()
		for _, client := range pending {
			running.Clients[client] = true
		}
		running.mu.Unlock()
		delete(s.pendingSubscribers, id)
	}
	s.mu.Unlock()

	// 启动对话协程
	go s.runSession(ctx, running)

	// 记录审计日志
	s.logAudit(id, "session_started", nil)

	return nil
}

// PauseSession 暂停会话
func (s *AIChatService) PauseSession(id string) error {
	s.mu.Lock()
	running, ok := s.sessions[id]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("session not running")
	}

	if running.CancelFunc != nil {
		running.CancelFunc()
	}

	// 更新状态
	if err := s.db.Model(&models.AIChatSession{}).Where("id = ?", id).Update("status", models.SessionStatusPaused).Error; err != nil {
		return err
	}

	// 广播暂停事件
	s.broadcastToSession(id, StreamEvent{
		Type: "status",
		Data: map[string]interface{}{
			"status": "paused",
		},
	})

	s.logAudit(id, "session_paused", nil)

	return nil
}

// StopSession 停止会话
func (s *AIChatService) StopSession(id string) error {
	s.mu.Lock()
	running, ok := s.sessions[id]
	if ok {
		if running.CancelFunc != nil {
			running.CancelFunc()
		}
		delete(s.sessions, id)
	}
	s.mu.Unlock()

	// 更新状态
	now := time.Now()
	if err := s.db.Model(&models.AIChatSession{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       models.SessionStatusCompleted,
		"completed_at": now,
	}).Error; err != nil {
		return err
	}

	// 广播停止事件
	s.broadcastToSession(id, StreamEvent{
		Type: "termination",
		Data: map[string]interface{}{
			"reason":  "manual_stop",
			"message": "会话已被手动停止",
		},
	})

	s.logAudit(id, "session_stopped", nil)

	return nil
}

// InjectDirectorCommand 注入导演指令
func (s *AIChatService) InjectDirectorCommand(sessionID string, cmd *models.DirectorCommand) error {
	cmd.ID = uuid.New().String()
	cmd.SessionID = sessionID
	cmd.Executed = false
	cmd.Timestamp = time.Now()

	if err := s.db.Create(cmd).Error; err != nil {
		return err
	}

	// 添加到运行时会话的待执行队列
	s.mu.RLock()
	running, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if ok {
		running.mu.Lock()
		running.PendingCommands = append(running.PendingCommands, cmd)
		running.mu.Unlock()
	}

	// 广播指令已接收
	s.broadcastToSession(sessionID, StreamEvent{
		Type: "director_command_received",
		Data: map[string]interface{}{
			"commandId":   cmd.ID,
			"targetAgent": cmd.TargetAgent,
			"command":     cmd.Command,
		},
	})

	s.logAudit(sessionID, "director_command_injected", map[string]interface{}{
		"targetAgent": cmd.TargetAgent,
		"command":     cmd.Command,
	})

	return nil
}

// CreateBranch 创建分支
func (s *AIChatService) CreateBranch(parentID string, round int, title string) (*models.AIChatSession, error) {
	parent, err := s.GetSession(parentID)
	if err != nil {
		return nil, err
	}

	// 创建新会话
	branch := &models.AIChatSession{
		ID:                uuid.New().String(),
		Title:             title,
		Status:            models.SessionStatusPending,
		Topic:             parent.Topic,
		GlobalConstraint:  parent.GlobalConstraint,
		MaxRounds:         parent.MaxRounds,
		CurrentRound:      round,
		TerminationConfig: parent.TerminationConfig,
		ParentID:          &parentID,
		BranchPoint:       &round,
		TokenUsage:        parent.TokenUsage,
	}

	// 复制 Agent 配置
	branch.SetAgentConfig("agent_a", parent.GetAgentConfig("agent_a"))
	branch.SetAgentConfig("agent_b", parent.GetAgentConfig("agent_b"))

	if err := s.db.Create(branch).Error; err != nil {
		return nil, err
	}

	// 复制指定轮数之前的历史消息
	var messages []models.AIChatMessage
	if err := s.db.Where("session_id = ? AND round <= ?", parentID, round).Find(&messages).Error; err != nil {
		return nil, err
	}

	// 插入到新会话
	for _, msg := range messages {
		msg.ID = 0 // 重置ID
		msg.SessionID = branch.ID
		if err := s.db.Create(&msg).Error; err != nil {
			return nil, err
		}
	}

	s.logAudit(branch.ID, "branch_created", map[string]interface{}{
		"parentId": parentID,
		"round":    round,
	})

	return branch, nil
}

// CreateSnapshot 创建快照
func (s *AIChatService) CreateSnapshot(sessionID string, title string) (*models.SessionSnapshot, error) {
	session, err := s.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	// 序列化会话数据
	snapshotData := map[string]interface{}{
		"session":  session,
		"messages": session.Messages,
	}

	snapshot := &models.SessionSnapshot{
		ID:           uuid.New().String(),
		SessionID:    sessionID,
		Title:        title,
		Round:        session.CurrentRound,
		SnapshotData: snapshotData,
	}

	if err := s.db.Create(snapshot).Error; err != nil {
		return nil, err
	}

	return snapshot, nil
}

// SubscribeToSession 订阅会话事件
func (s *AIChatService) SubscribeToSession(sessionID string, client chan StreamEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	running, ok := s.sessions[sessionID]
	if ok {
		// Session already running — register immediately
		running.mu.Lock()
		running.Clients[client] = true
		running.mu.Unlock()
	} else {
		// Session not started yet — queue as pending subscriber
		s.pendingSubscribers[sessionID] = append(s.pendingSubscribers[sessionID], client)
	}
}

// UnsubscribeFromSession 取消订阅
func (s *AIChatService) UnsubscribeFromSession(sessionID string, client chan StreamEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	running, ok := s.sessions[sessionID]
	if ok {
		running.mu.Lock()
		delete(running.Clients, client)
		running.mu.Unlock()
	}

	// Also remove from pending subscribers
	if pending, ok := s.pendingSubscribers[sessionID]; ok {
		filtered := pending[:0]
		for _, c := range pending {
			if c != client {
				filtered = append(filtered, c)
			}
		}
		s.pendingSubscribers[sessionID] = filtered
	}
}

// broadcastToSession 广播事件到所有客户端
func (s *AIChatService) broadcastToSession(sessionID string, event StreamEvent) {
	s.mu.RLock()
	running, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	running.mu.RLock()
	clients := make([]chan StreamEvent, 0, len(running.Clients))
	for client := range running.Clients {
		clients = append(clients, client)
	}
	running.mu.RUnlock()

	// 异步发送，避免阻塞
	go func() {
		for _, client := range clients {
			select {
			case client <- event:
			default:
				// 客户端通道已满，跳过
			}
		}
	}()
}

// runSession 运行会话主循环
func (s *AIChatService) runSession(ctx context.Context, running *RunningSession) {
	defer func() {
		// 清理
		s.mu.Lock()
		delete(s.sessions, running.Session.ID)
		s.mu.Unlock()
	}()

	session := running.Session

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 检查终止条件
		if shouldTerminate, reason := s.checkTermination(session); shouldTerminate {
			s.terminateSession(session.ID, reason)
			return
		}

		// 确定当前发言方
		agentID := s.getCurrentAgentID(session)
		agentConfig := session.GetAgentConfig(agentID)

		// 执行一轮对话
		_, err := s.executeTurn(ctx, session, agentID, agentConfig, running)
		if err != nil {
			fmt.Printf("[AIChat] Error executing turn: %v\n", err)
			s.handleError(session.ID, err)
			return
		}

		// 更新状态（消息已通过 message_complete 事件广播）
		session.CurrentRound = s.calculateRound(session)
		if err := s.db.Model(&models.AIChatSession{}).Where("id = ?", session.ID).Update("current_round", session.CurrentRound).Error; err != nil {
			fmt.Printf("[AIChat] Error updating round: %v\n", err)
		}

		// 应用延迟
		time.Sleep(time.Second) // 可配置
	}
}

// executeTurn 执行一轮对话
func (s *AIChatService) executeTurn(ctx context.Context, session *models.AIChatSession, agentID string, agentConfig models.AgentConfig, running *RunningSession) (*models.AIChatMessage, error) {
	startTime := time.Now()

	// 构建上下文
	contextMessages, err := s.buildContext(session, agentID)
	if err != nil {
		return nil, err
	}

	// === PROACTIVE SEARCH: 在第一轮对话时，检查主题是否需要搜索 ===
	var searchContext string
	fmt.Printf("[AI-Chat] Proactive search check - Round: %d, AllowedTools: %v, LastSearchRound: %d, SearchCount: %d\n",
		session.CurrentRound, agentConfig.AllowedTools, running.LastSearchRound, running.SearchCount)

	// 限制：每个会话最多主动搜索 2 次，且相邻搜索至少间隔 2 轮
	maxProactiveSearches := 2
	minRoundsBetweenSearch := 2

	canProactivelySearch := running.SearchCount < maxProactiveSearches &&
		(session.CurrentRound-running.LastSearchRound >= minRoundsBetweenSearch || running.LastSearchRound == -1)

	if canProactivelySearch && session.CurrentRound == 0 && len(agentConfig.AllowedTools) > 0 {
		// 检查是否有搜索工具权限
		hasSearchTool := false
		for _, tool := range agentConfig.AllowedTools {
			if strings.Contains(strings.ToLower(tool), "search") {
				hasSearchTool = true
				break
			}
		}

		fmt.Printf("[AI-Chat] Has search tool: %v, searchService nil: %v\n", hasSearchTool, s.searchService == nil)

		if hasSearchTool && s.searchService != nil {
			// 分析主题是否需要最新信息（语义检测，关键词匹配作为兜底）
			needsSearch := s.semanticNeedsSearch(ctx, session.Topic, agentConfig.Model)

			if needsSearch {
				fmt.Printf("[AI-Chat] Proactive search triggered for topic: %s\n", session.Topic)

				// 执行搜索
				searchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				results, err := s.searchService.Search(searchCtx, session.Topic, 0, "tencent")
				if err != nil || len(results) == 0 {
					results, err = s.searchService.Search(searchCtx, session.Topic, 0, "brightdata")
					if err != nil || len(results) == 0 {
						results, err = s.searchService.Search(searchCtx, session.Topic, 0, "serper")
					}
				}

				if err == nil && len(results) > 0 {
					var searchBuilder strings.Builder
					searchBuilder.WriteString("\n\n=== 最新搜索结果 (Latest Search Results) ===\n")
					searchBuilder.WriteString(fmt.Sprintf("搜索主题: %s\n", session.Topic))
					searchBuilder.WriteString(fmt.Sprintf("结果数量: %d\n\n", len(results)))

					for i, result := range results {
						if i >= 5 {
							break
						}
						// 清理搜索结果中的无效 UTF-8
						title := sanitizeUTF8(result.Title)
						url := sanitizeUTF8(result.URL)
						snippet := sanitizeUTF8(result.Snippet)

						searchBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, title))
						if url != "" {
							searchBuilder.WriteString(fmt.Sprintf("   链接: %s\n", url))
						}
						if snippet != "" {
							if len(snippet) > 200 {
								snippet = snippet[:200] + "..."
							}
							searchBuilder.WriteString(fmt.Sprintf("   摘要: %s\n", snippet))
						}
						searchBuilder.WriteString("\n")
					}
					searchBuilder.WriteString("=== 搜索结果结束 ===\n\n")
					// 弱化对固定话术的引导：让模型自然使用这些信息，而不是每次都说“基于搜索结果”
					searchBuilder.WriteString("以上是最新的网络搜索结果，仅供参考。回答时可以自由综合这些信息，无需重复强调‘根据搜索结果’。\n")

					searchContext = searchBuilder.String()
					// 更新搜索计数
					running.SearchCount++
					running.LastSearchRound = session.CurrentRound
					fmt.Printf("[AI-Chat] Search results injected, length: %d, SearchCount: %d\n", len(searchContext), running.SearchCount)
				} else {
					fmt.Printf("[AI-Chat] Search failed or no results: %v\n", err)
				}
			}
		}
	}

	// 如果有搜索结果，注入到上下文的最后一条消息
	if searchContext != "" {
		contextMessages = append(contextMessages, map[string]string{
			"role":    "system",
			"content": searchContext,
		})
	}

	// 检查并应用导演指令
	running.mu.Lock()
	var pendingCmd *models.DirectorCommand
	for i, cmd := range running.PendingCommands {
		if !cmd.Executed && (cmd.TargetAgent == agentID || cmd.TargetAgent == "both") {
			if cmd.InsertAfterRound == nil || *cmd.InsertAfterRound <= session.CurrentRound {
				pendingCmd = cmd
				running.PendingCommands = append(running.PendingCommands[:i], running.PendingCommands[i+1:]...)
				break
			}
		}
	}
	running.mu.Unlock()

	if pendingCmd != nil {
		// 注入导演指令到上下文
		directorMessage := fmt.Sprintf("[导演指令] %s", pendingCmd.Command)
		contextMessages = append(contextMessages, map[string]string{
			"role":    "system",
			"content": directorMessage,
		})

		// 标记为已执行
		s.db.Model(&models.DirectorCommand{}).Where("id = ?", pendingCmd.ID).Update("executed", true)

		// 广播指令已应用
		s.broadcastToSession(session.ID, StreamEvent{
			Type: "director_command_applied",
			Data: map[string]interface{}{
				"commandId":   pendingCmd.ID,
				"targetAgent": pendingCmd.TargetAgent,
			},
		})
	}

	// 调用 AI（流式）
	var responseBuilder strings.Builder
	var tokens int

	// 广播消息开始
	s.broadcastToSession(session.ID, StreamEvent{
		Type: "message_start",
		Data: map[string]interface{}{
			"agentId":   agentID,
			"agentName": agentConfig.Name,
			"round":     s.calculateRound(session),
		},
	})

	response, tokens, err := s.callAI(ctx, session.ID, agentConfig, contextMessages, agentID, func(chunk string) {
		responseBuilder.WriteString(chunk)
		// 广播消息 chunk
		s.broadcastToSession(session.ID, StreamEvent{
			Type: "message_chunk",
			Data: map[string]interface{}{
				"agentId":   agentID,
				"agentName": agentConfig.Name,
				"chunk":     chunk,
			},
		})
	})
	if err != nil {
		return nil, err
	}

	// 处理工具调用
	messageType := models.MessageTypeText
	toolCalls := make(models.JSONMap)
	toolResults := make(models.JSONMap)

	// 检测并执行工具调用
	// 获取历史消息用于上下文分析
	var historyMessages []models.AIChatMessage
	s.db.Where("session_id = ?", session.ID).Order("round, id").Find(&historyMessages)

	toolCall := s.detectToolCall(response, agentConfig, historyMessages)
	if toolCall != nil && s.mcpManager != nil {
		// 限制：每轮最多调用一次工具
		messageType = models.MessageTypeToolCall
		toolCalls[toolCall.Name] = toolCall.Arguments

		// 广播工具调用
		s.broadcastToSession(session.ID, StreamEvent{
			Type: "tool_call",
			Data: map[string]interface{}{
				"agentId":   agentID,
				"agentName": agentConfig.Name,
				"toolName":  toolCall.Name,
				"arguments": toolCall.Arguments,
			},
		})

		// 执行工具
		result := s.executeTool(toolCall, agentConfig)
		toolResults[toolCall.Name] = result

		// 广播工具结果
		s.broadcastToSession(session.ID, StreamEvent{
			Type: "tool_result",
			Data: map[string]interface{}{
				"agentId":   agentID,
				"agentName": agentConfig.Name,
				"toolName":  toolCall.Name,
				"result":    result,
			},
		})

		// 将工具结果追加到响应
		response += fmt.Sprintf("\n\n[Tool Result - %s]\n%s", toolCall.Name, result)
	}

	// 清理和验证 UTF-8 字符串
	response = sanitizeUTF8(response)

	// 创建消息
	message := &models.AIChatMessage{
		SessionID:   session.ID,
		Round:       s.calculateRound(session),
		AgentID:     agentID,
		AgentName:   agentConfig.Name,
		Content:     response,
		MessageType: messageType,
		ToolCalls:   toolCalls,
		ToolResults: toolResults,
		Tokens:      tokens,
		Latency:     time.Since(startTime).Milliseconds(),
	}

	if err := s.db.Create(message).Error; err != nil {
		return nil, err
	}

	// 广播消息完成
	s.broadcastToSession(session.ID, StreamEvent{
		Type: "message_complete",
		Data: map[string]interface{}{
			"agentId":   agentID,
			"agentName": agentConfig.Name,
			"round":     message.Round,
			"content":   response,
			"tokens":    tokens,
		},
	})

	// 更新 Token 使用统计
	if agentID == "agent_a" {
		session.TokenUsage.AgentAOutput += tokens
	} else {
		session.TokenUsage.AgentBOutput += tokens
	}
	session.TokenUsage.Total = session.TokenUsage.AgentAInput + session.TokenUsage.AgentAOutput +
		session.TokenUsage.AgentBInput + session.TokenUsage.AgentBOutput

	s.db.Model(&models.AIChatSession{}).Where("id = ?", session.ID).Updates(map[string]interface{}{
		"token_agent_a_output": session.TokenUsage.AgentAOutput,
		"token_agent_b_output": session.TokenUsage.AgentBOutput,
		"token_total":          session.TokenUsage.Total,
	})

	return message, nil
}

// buildContext 构建对话上下文
func (s *AIChatService) buildContext(session *models.AIChatSession, agentID string) ([]map[string]string, error) {
	var messages []map[string]string

	// 系统提示词
	agentConfig := session.GetAgentConfig(agentID)

	// Use multi-agent aware system prompt
	var systemPrompt string
	if session.IsMultiAgent() {
		systemPrompt = s.buildSystemPromptWithSession(agentConfig, session.Topic, session.GlobalConstraint, session, agentID)
	} else {
		systemPrompt = s.buildSystemPrompt(agentConfig, session.Topic, session.GlobalConstraint)
	}

	messages = append(messages, map[string]string{
		"role":    "system",
		"content": systemPrompt,
	})

	// 获取历史消息
	var history []models.AIChatMessage
	if err := s.db.Where("session_id = ?", session.ID).Order("round, id").Find(&history).Error; err != nil {
		return nil, err
	}

	// 构建对话历史
	for _, msg := range history {
		role := "assistant"
		if msg.AgentID != agentID {
			role = "user" // 对方的消息作为 user 输入
		}

		// For multi-agent, prefix messages with speaker name
		content := msg.Content
		if session.IsMultiAgent() && msg.AgentID != agentID {
			content = fmt.Sprintf("[%s]: %s", msg.AgentName, msg.Content)
		}

		messages = append(messages, map[string]string{
			"role":    role,
			"content": content,
		})
	}

	return messages, nil
}

// buildSystemPrompt 构建系统提示词
func (s *AIChatService) buildSystemPrompt(agent models.AgentConfig, topic, constraint string) string {
	return s.buildSystemPromptWithSession(agent, topic, constraint, nil, "")
}

// buildSystemPromptWithSession 构建系统提示词（带会话信息用于多Agent场景）
func (s *AIChatService) buildSystemPromptWithSession(agent models.AgentConfig, topic, constraint string, session *models.AIChatSession, currentAgentID string) string {
	var prompt strings.Builder

	// 角色设定
	prompt.WriteString(fmt.Sprintf("你是%s。%s\n\n", agent.Name, agent.Role))

	// 风格设定
	prompt.WriteString(fmt.Sprintf("语言风格：%s\n", agent.Style.LanguageStyle))
	prompt.WriteString(fmt.Sprintf("知识水平：%s\n", agent.Style.KnowledgeLevel))
	prompt.WriteString(fmt.Sprintf("语气：%s\n\n", agent.Style.Tone))

	// Multi-agent: identify other participants
	if session != nil && session.IsMultiAgent() {
		prompt.WriteString("对话参与者：\n")
		for i, otherAgent := range session.GetAllAgents() {
			agentID := session.GetAgentID(i)
			if agentID != currentAgentID {
				prompt.WriteString(fmt.Sprintf("- %s: %s\n", otherAgent.Name, truncateString(otherAgent.Role, 100)))
			}
		}
		prompt.WriteString("\n")
	}

	// 主题
	prompt.WriteString(fmt.Sprintf("讨论主题：%s\n\n", topic))

	// 全局限定
	if constraint != "" {
		prompt.WriteString(fmt.Sprintf("重要限定：%s\n\n", constraint))
	}

	// 输出长度指导（通过prompt而非硬截断）
	if agent.MaxTokens > 0 {
		prompt.WriteString(fmt.Sprintf("输出长度指导：请尽量在%d个token以内完成回答。如果内容较长，可以自然结束，不要生硬截断。\n\n", agent.MaxTokens))
	}

	// 工具说明
	if len(agent.AllowedTools) > 0 {
		prompt.WriteString("你可以使用以下工具：\n")
		for _, tool := range agent.AllowedTools {
			prompt.WriteString(fmt.Sprintf("- %s\n", tool))
		}
	}

	return prompt.String()
}

// truncateString truncates a string to max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// callAI 调用 AI 服务（流式版本，通过回调实时发送 chunk）
func (s *AIChatService) callAI(ctx context.Context, sessionID string, agent models.AgentConfig, messages []map[string]string, agentID string, onChunk func(chunk string)) (string, int, error) {
	// 转换消息格式
	aiMessages := make([]models.Message, 0, len(messages))
	for _, m := range messages {
		aiMessages = append(aiMessages, models.Message{
			Role:    m["role"],
			Content: m["content"],
		})
	}

	// 构建请求
	// MaxTokens 设为 0 表示不限制，避免 API 硬截断输出
	// 长度控制已通过 system prompt 中的"输出长度指导"实现
	request := ai.ChatRequest{
		Model:       agent.Model,
		Messages:    aiMessages,
		Temperature: agent.Temperature,
		MaxTokens:   0,
		Stream:      true, // 启用流式
	}

	// 调用 AI 服务
	stream, err := s.aiService.ChatStream(ctx, &request)
	if err != nil {
		return "", 0, err
	}

	// 收集响应，同时通过回调发送 chunk
	var content strings.Builder
	tokens := 0
	for chunk := range stream {
		if chunk.Done {
			break
		}
		content.WriteString(chunk.Content)
		tokens += len(chunk.Content) / 4 // 粗略估算
		if onChunk != nil {
			onChunk(chunk.Content)
		}
	}

	return content.String(), tokens, nil
}

// checkTermination 检查终止条件
func (s *AIChatService) checkTermination(session *models.AIChatSession) (bool, string) {
	config := session.TerminationConfig

	// 1. 固定轮数
	if config.Type == models.TerminationTypeFixedRounds && session.CurrentRound >= config.MaxRounds {
		return true, "max_rounds_reached"
	}

	// 2. 关键词检测
	if config.Type == models.TerminationTypeKeyword && len(config.Keywords) > 0 {
		// 获取最后一条消息
		var lastMsg models.AIChatMessage
		if err := s.db.Where("session_id = ?", session.ID).Order("id DESC").First(&lastMsg).Error; err == nil {
			content := strings.ToLower(lastMsg.Content)
			for _, keyword := range config.Keywords {
				if strings.Contains(content, strings.ToLower(keyword)) {
					return true, "keyword_triggered"
				}
			}
		}
	}

	// 3. 相似度检测（循环检测）
	if config.SimilarityThreshold > 0 && config.ConsecutiveSimilarRounds > 0 {
		if s.checkSimilarityLoop(session, config.SimilarityThreshold, config.ConsecutiveSimilarRounds) {
			return true, "similarity_threshold"
		}
	}

	return false, ""
}

// checkSimilarityLoop 检测相似度循环
func (s *AIChatService) checkSimilarityLoop(session *models.AIChatSession, threshold float64, consecutive int) bool {
	// 获取最近的消息
	var messages []models.AIChatMessage
	if err := s.db.Where("session_id = ?", session.ID).Order("id DESC").Limit(consecutive * 2).Find(&messages).Error; err != nil {
		return false
	}

	if len(messages) < consecutive*2 {
		return false
	}

	// 检查连续几轮的相似度
	for i := 0; i < consecutive-1; i++ {
		similarity := calculateSimilarity(messages[i].Content, messages[i+1].Content)
		if similarity < threshold {
			return false
		}
	}

	return true
}

// calculateSimilarity 计算文本相似度（简化版）
func calculateSimilarity(text1, text2 string) float64 {
	// 这里应该使用更复杂的算法，如余弦相似度
	// 简化版：使用简单的字符重叠率
	if text1 == "" || text2 == "" {
		return 0
	}

	// 转换为小写并分词
	words1 := strings.Fields(strings.ToLower(text1))
	words2 := strings.Fields(strings.ToLower(text2))

	// 计算重叠词数
	overlap := 0
	wordCount := make(map[string]int)
	for _, w := range words1 {
		wordCount[w]++
	}
	for _, w := range words2 {
		if wordCount[w] > 0 {
			overlap++
			wordCount[w]--
		}
	}

	// Jaccard 相似度
	union := len(words1) + len(words2) - overlap
	if union == 0 {
		return 1.0
	}

	return float64(overlap) / float64(union)
}

// getCurrentAgentID 获取当前应该发言的 Agent ID
func (s *AIChatService) getCurrentAgentID(session *models.AIChatSession) string {
	// 根据消息数量判断轮到谁
	var count int64
	s.db.Model(&models.AIChatMessage{}).Where("session_id = ?", session.ID).Count(&count)

	// Multi-agent mode
	if session.IsMultiAgent() {
		speakerIndex := session.GetNextSpeakerIndex(int(count))
		return session.GetAgentID(speakerIndex)
	}

	// Legacy 2-agent mode
	if count%2 == 0 {
		return "agent_a"
	}
	return "agent_b"
}

// calculateRound 计算当前轮数
func (s *AIChatService) calculateRound(session *models.AIChatSession) int {
	var count int64
	s.db.Model(&models.AIChatMessage{}).Where("session_id = ?", session.ID).Count(&count)

	return int(math.Ceil(float64(count) / 2))
}

// terminateSession 终止会话
func (s *AIChatService) terminateSession(sessionID string, reason string) {
	// 更新会话状态到 DB
	now := time.Now()
	statusVal := models.SessionStatusCompleted
	if reason == "manual_stop" {
		statusVal = models.SessionStatusTerminated
	}
	s.db.Model(&models.AIChatSession{}).Where("id = ?", sessionID).Updates(map[string]interface{}{
		"status":       statusVal,
		"completed_at": now,
	})

	// 广播终止事件
	s.broadcastToSession(sessionID, StreamEvent{
		Type: "termination",
		Data: map[string]interface{}{
			"reason":     reason,
			"message":    getTerminationMessage(reason),
			"sessionId":  sessionID,
			"session_id": sessionID,
		},
	})

	// 非手动停止时，异步生成评估报告（LLM 调用耗时，放到 goroutine 中）
	if reason != "manual_stop" && s.evaluator != nil {
		go func(sessionID string) {
			session, err := s.GetSession(sessionID)
			if err != nil {
				fmt.Printf("[AI-Chat] Failed to load session for evaluation: %v\n", err)
				return
			}
			fmt.Printf("[AI-Chat] Starting evaluation for session: %s\n", sessionID)
			if _, err := s.evaluator.Evaluate(session); err != nil {
				fmt.Printf("[AI-Chat] Failed to evaluate session: %v\n", err)
			} else {
				fmt.Printf("[AI-Chat] Evaluation completed for session: %s\n", sessionID)
			}
		}(sessionID)
	}

	s.logAudit(sessionID, "session_terminated", map[string]interface{}{
		"reason": reason,
	})
}

// handleError 处理错误
func (s *AIChatService) handleError(sessionID string, err error) {
	s.db.Model(&models.AIChatSession{}).Where("id = ?", sessionID).Update("status", models.SessionStatusError)

	s.broadcastToSession(sessionID, StreamEvent{
		Type: "error",
		Data: map[string]interface{}{
			"message": err.Error(),
		},
	})
}

// logAudit 记录审计日志
func (s *AIChatService) logAudit(sessionID string, eventType string, eventData map[string]interface{}) {
	log := &models.AuditLog{
		SessionID: sessionID,
		EventType: eventType,
		EventData: eventData,
	}
	s.db.Create(log)
}

// getTerminationMessage 获取终止消息
func getTerminationMessage(reason string) string {
	switch reason {
	case "max_rounds_reached":
		return "已达到设定的最大对话轮数"
	case "keyword_triggered":
		return "检测到终止关键词"
	case "similarity_threshold":
		return "检测到对话循环，内容重复度过高"
	case "manual_stop":
		return "会话已被手动停止"
	default:
		return "会话已结束"
	}
}

// SessionConfig 会话配置
type SessionConfig struct {
	Title             string                   `json:"title"`
	Topic             string                   `json:"topic"`
	GlobalConstraint  string                   `json:"global_constraint"`
	MaxRounds         int                      `json:"max_rounds"`
	TerminationConfig models.TerminationConfig `json:"termination_config"`
	AgentA            models.AgentConfig       `json:"agent_a"`
	AgentB            models.AgentConfig       `json:"agent_b"`
	// Multi-agent support (3+ agents)
	Agents        []models.AgentConfig `json:"agents,omitempty"`
	SpeakingOrder []string             `json:"speaking_order,omitempty"`
}

// SessionFilter 会话过滤器
type SessionFilter struct {
	Status   models.SessionStatus `json:"status,omitempty"`
	ParentID *string              `json:"parent_id,omitempty"`
}

// GetTemplates 获取模板列表
func (s *AIChatService) GetTemplates() ([]models.SessionTemplate, error) {
	var templates []models.SessionTemplate
	if err := s.db.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// GetTemplate 获取单个模板
func (s *AIChatService) GetTemplate(id string) (*models.SessionTemplate, error) {
	var template models.SessionTemplate
	if err := s.db.First(&template, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// semanticNeedsSearch uses an LLM to determine whether the given discussion topic
// requires real-time web information. Falls back to keywordNeedsSearch on error.
func (s *AIChatService) semanticNeedsSearch(ctx context.Context, topic string, _ string) bool {
	if s.aiService == nil {
		return keywordNeedsSearch(topic)
	}

	systemPrompt := `You are a search necessity classifier for AI debate sessions. Determine whether the given topic requires real-time or recently updated web information to be discussed with accurate, current facts.

Return {"search": true} if the topic involves:
- Current events, ongoing news, or recent political/economic developments
- Live market data, cryptocurrency prices, or financial indicators
- Recent scientific discoveries, tech releases, or product launches
- Sports results, election outcomes, or recent statistics
- Any subject where information from the past 6 months significantly matters

Return {"search": false} if the topic involves:
- Timeless philosophical, ethical, or social debates
- Historical events (before 2024)
- General scientific principles or established theory
- Cultural, literary, or artistic discussions
- Math, logic, or abstract reasoning

Respond with ONLY valid JSON: {"search": true} or {"search": false}`

	detectCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	ch, err := s.aiService.ChatStream(detectCtx, &ai.ChatRequest{
		Messages: []models.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: topic},
		},
		Model:     "deepseek-chat", // always use the fast cheap model for classification
		MaxTokens: 20,
		Stream:    true,
	})
	if err != nil {
		fmt.Printf("[AI-Chat] Semantic search classification error, using keyword fallback: %v\n", err)
		return keywordNeedsSearch(topic)
	}

	var sb strings.Builder
	collecting := true
	for collecting {
		select {
		case chunk, ok := <-ch:
			if !ok {
				collecting = false
			} else {
				sb.WriteString(chunk.Content)
				if chunk.Done {
					collecting = false
				}
			}
		case <-detectCtx.Done():
			fmt.Println("[AI-Chat] Semantic search classification timed out, using keyword fallback")
			return keywordNeedsSearch(topic)
		}
	}

	raw := strings.TrimSpace(sb.String())
	if idx := strings.Index(raw, "{"); idx >= 0 {
		if end := strings.LastIndex(raw, "}"); end >= idx {
			raw = raw[idx : end+1]
		}
	}

	var decision struct {
		Search bool `json:"search"`
	}
	if err := json.Unmarshal([]byte(raw), &decision); err != nil {
		fmt.Printf("[AI-Chat] Semantic search parse error (%v), raw=%q — using keyword fallback\n", err, raw)
		return keywordNeedsSearch(topic)
	}

	fmt.Printf("[AI-Chat] Semantic search classification: topic=%q → search=%v\n", topic, decision.Search)
	return decision.Search
}

// keywordNeedsSearch is the legacy keyword-based fallback for proactive search detection.
func keywordNeedsSearch(topic string) bool {
	topicLower := strings.ToLower(topic)
	searchKeywords := []string{
		"最新", "latest", "新闻", "news", "实时", "real-time",
		"当前", "current", "现在", "now", "今天", "today",
		"昨天", "yesterday", "本周", "this week",
		"热点", "trending", "热门", "popular",
	}
	for _, keyword := range searchKeywords {
		if strings.Contains(topicLower, keyword) {
			return true
		}
	}
	return false
}

// CreateTemplate 创建自定义模板
func (s *AIChatService) CreateTemplate(template *models.SessionTemplate) error {
	template.ID = uuid.New().String()
	template.IsBuiltin = false
	return s.db.Create(template).Error
}

// UpdateTemplate 更新模板（仅允许非内置模板）
func (s *AIChatService) UpdateTemplate(id string, req *models.SessionTemplate) (*models.SessionTemplate, error) {
	var template models.SessionTemplate
	if err := s.db.First(&template, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if template.IsBuiltin {
		return nil, errors.New("cannot modify builtin templates")
	}
	template.Name = req.Name
	template.Description = req.Description
	template.Icon = req.Icon
	template.Category = req.Category
	template.Config = req.Config
	if err := s.db.Save(&template).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// DeleteTemplate 删除模板（仅允许非内置模板）
func (s *AIChatService) DeleteTemplate(id string) error {
	var template models.SessionTemplate
	if err := s.db.First(&template, "id = ?", id).Error; err != nil {
		return err
	}
	if template.IsBuiltin {
		return errors.New("cannot delete builtin templates")
	}
	return s.db.Delete(&template).Error
}

// CloneTemplate 克隆模板（内置或自定义均可）
func (s *AIChatService) CloneTemplate(id string) (*models.SessionTemplate, error) {
	original, err := s.GetTemplate(id)
	if err != nil {
		return nil, err
	}
	clone := models.SessionTemplate{
		ID:          uuid.New().String(),
		Name:        original.Name + " (副本)",
		Description: original.Description,
		Icon:        original.Icon,
		Category:    original.Category,
		Config:      original.Config,
		IsBuiltin:   false,
	}
	if err := s.db.Create(&clone).Error; err != nil {
		return nil, err
	}
	return &clone, nil
}

// GetExportService 获取导出服务
func (s *AIChatService) GetExportService() *ExportService {
	return NewExportService(s.db)
}

// GetMCPManager 获取MCP管理器
func (s *AIChatService) GetMCPManager() *agent.MCPManager {
	return s.mcpManager
}

// ToolCall 工具调用结构
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// detectToolCall 检测响应中是否包含工具调用意图（基于AI响应内容和上下文消息）
func (s *AIChatService) detectToolCall(response string, agentConfig models.AgentConfig, contextMessages []models.AIChatMessage) *ToolCall {
	// 检查该Agent是否有允许的工具
	if len(agentConfig.AllowedTools) == 0 {
		return nil
	}

	// 首先检查当前对话上下文，提取最近的对话主题
	var recentContext string
	if len(contextMessages) > 0 {
		// 获取最后几条消息作为上下文
		startIdx := len(contextMessages) - 3
		if startIdx < 0 {
			startIdx = 0
		}
		for i := startIdx; i < len(contextMessages); i++ {
			recentContext += contextMessages[i].Content + " "
		}
	}
	recentContext += response
	contextLower := strings.ToLower(recentContext)

	// 检测工具调用意图
	for _, tool := range agentConfig.AllowedTools {
		toolLower := strings.ToLower(tool)

		// 检测搜索操作 - 优先检测，因为这是最常用的
		if strings.Contains(toolLower, "search") || strings.Contains(toolLower, "搜索") {
			// 检查是否需要搜索最新信息/新闻
			searchKeywords := []string{
				"搜索", "search", "查找", "find", "查询", "query",
				"最新", "latest", "新闻", "news", "实时", "real-time",
				"当前", "current", "现在", "now", "今天", "today",
				"网上", "web", "互联网", "internet", "在线", "online",
				"论文", "paper", "文章", "article", "资料", "information",
			}

			hasSearchIntent := false
			for _, keyword := range searchKeywords {
				if strings.Contains(contextLower, keyword) {
					hasSearchIntent = true
					break
				}
			}

			if hasSearchIntent {
				query := extractSearchQuery(recentContext)
				if query == "" {
					// 如果无法提取查询，使用最后一条消息
					if len(contextMessages) > 0 {
						query = contextMessages[len(contextMessages)-1].Content
					}
				}
				fmt.Printf("[AI-Chat] Auto-detected search intent, query: %s\n", query)
				return &ToolCall{
					Name: "search/web_search",
					Arguments: map[string]interface{}{
						"query": query,
					},
				}
			}
		}

		// 检测文件系统操作
		if strings.Contains(toolLower, "filesystem") || strings.Contains(toolLower, "file") {
			if strings.Contains(contextLower, "read_file") || strings.Contains(contextLower, "读取文件") ||
				strings.Contains(contextLower, "read") || strings.Contains(contextLower, "读取") {
				filePath := extractFilePath(recentContext)
				if filePath != "." && filePath != "" {
					return &ToolCall{
						Name: "filesystem-local/read_file",
						Arguments: map[string]interface{}{
							"path": filePath,
						},
					}
				}
			}
			if strings.Contains(contextLower, "list_directory") || strings.Contains(contextLower, "列出目录") ||
				strings.Contains(contextLower, "list") || strings.Contains(contextLower, "列出") {
				return &ToolCall{
					Name: "filesystem-local/list_directory",
					Arguments: map[string]interface{}{
						"path": ".",
					},
				}
			}
		}

		// 检测终端操作
		if strings.Contains(toolLower, "terminal") || strings.Contains(toolLower, "command") {
			if strings.Contains(contextLower, "execute_command") || strings.Contains(contextLower, "执行命令") ||
				strings.Contains(contextLower, "execute") || strings.Contains(contextLower, "执行") {
				command := extractCommand(recentContext)
				if command != "" {
					return &ToolCall{
						Name: "terminal/execute_command",
						Arguments: map[string]interface{}{
							"command": command,
						},
					}
				}
			}
		}
	}

	return nil
}

// executeTool 执行工具调用
func (s *AIChatService) executeTool(toolCall *ToolCall, agentConfig models.AgentConfig) string {
	if s.mcpManager == nil {
		return "[Error: MCP manager not initialized]"
	}

	// 解析工具名称
	parts := strings.SplitN(toolCall.Name, "/", 2)
	if len(parts) != 2 {
		return fmt.Sprintf("[Error: Invalid tool format: %s]", toolCall.Name)
	}
	serverName := parts[0]
	toolName := parts[1]

	// 检查工具是否在允许列表中
	allowed := false
	for _, tool := range agentConfig.AllowedTools {
		if strings.Contains(tool, serverName) || strings.Contains(tool, toolName) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Sprintf("[Error: Tool %s not in allowed tools list]", toolCall.Name)
	}

	// 检查工具是否在禁止列表中
	for _, tool := range agentConfig.BlockedTools {
		if strings.Contains(tool, toolCall.Name) {
			return fmt.Sprintf("[Error: Tool %s is blocked]", toolCall.Name)
		}
	}

	// 确保MCP服务器已发现
	s.mcpManager.Discover()

	// 获取服务器
	server, ok := s.mcpManager.GetServer(serverName)
	if !ok {
		return fmt.Sprintf("[Error: MCP server '%s' not found]", serverName)
	}

	// 对于内置服务器，直接处理
	switch serverName {
	case "filesystem-local":
		return s.handleFilesystemTool(toolName, toolCall.Arguments)
	case "terminal":
		return s.handleTerminalTool(toolName, toolCall.Arguments)
	case "search":
		return s.handleSearchTool(toolName, toolCall.Arguments)
	}

	// 对于外部MCP服务器，检查连接状态
	if !server.Connected && server.Client == nil {
		return fmt.Sprintf("[Error: MCP server '%s' not connected]", serverName)
	}

	// 调用工具
	result, err := s.mcpManager.CallTool(serverName, toolName, toolCall.Arguments)
	if err != nil {
		return fmt.Sprintf("[Error: %v]", err)
	}

	return result
}

// handleFilesystemTool 处理文件系统工具
func (s *AIChatService) handleFilesystemTool(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "read_file":
		path, _ := args["path"].(string)
		if path == "" {
			path = "."
		}
		return s.executeLocalCommand("cat", path)
	case "list_directory":
		path, _ := args["path"].(string)
		if path == "" {
			path = "."
		}
		return s.executeLocalCommand("ls", "-la", path)
	default:
		return fmt.Sprintf("[Error: Unknown filesystem tool: %s]", toolName)
	}
}

// handleTerminalTool 处理终端工具
func (s *AIChatService) handleTerminalTool(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "execute_command":
		command, _ := args["command"].(string)
		if command == "" {
			return "[Error: No command specified]"
		}
		// 安全命令白名单
		safeCommands := []string{"ls", "pwd", "whoami", "cat", "echo", "grep", "find", "head", "tail", "wc"}
		isSafe := false
		for _, safe := range safeCommands {
			if strings.HasPrefix(command, safe) {
				isSafe = true
				break
			}
		}
		if !isSafe {
			return fmt.Sprintf("[Error: Command '%s' is not in safe command list]", command)
		}
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return "[Error: Empty command]"
		}
		return s.executeLocalCommand(parts[0], parts[1:]...)
	default:
		return fmt.Sprintf("[Error: Unknown terminal tool: %s]", toolName)
	}
}

// handleSearchTool 处理搜索工具
func (s *AIChatService) handleSearchTool(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "web_search":
		query, _ := args["query"].(string)
		if query == "" {
			return "[Error: No search query specified]"
		}

		// 检查搜索服务是否可用
		if s.searchService == nil {
			return "[Error: Search service not available]"
		}

		// 使用实际的搜索服务
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 调用搜索服务（优先使用 Tencent Hunyuan Search，然后尝试其他提供商）
		results, err := s.searchService.Search(ctx, query, 0, "tencent")
		if err != nil || len(results) == 0 {
			// 尝试 BrightData 作为备用
			results, err = s.searchService.Search(ctx, query, 0, "brightdata")
			if err != nil || len(results) == 0 {
				// 尝试 Serper 作为备用
				results, err = s.searchService.Search(ctx, query, 0, "serper")
				if err != nil {
					return fmt.Sprintf("[Search Error: %v]", err)
				}
			}
		}

		// 格式化搜索结果
		var resultStr strings.Builder
		resultStr.WriteString("=== WEB SEARCH RESULTS ===")
		resultStr.WriteString(fmt.Sprintf("\nQuery: %s", query))
		resultStr.WriteString(fmt.Sprintf("\nResults Found: %d\n\n", len(results)))

		for i, result := range results {
			if i >= 5 { // 限制显示前5条结果
				break
			}
			// 清理搜索结果中的无效 UTF-8
			title := sanitizeUTF8(result.Title)
			url := sanitizeUTF8(result.URL)
			snippet := sanitizeUTF8(result.Snippet)

			resultStr.WriteString(fmt.Sprintf("%d. %s\n", i+1, title))
			if url != "" {
				resultStr.WriteString(fmt.Sprintf("   URL: %s\n", url))
			}
			if snippet != "" {
				// 限制内容长度
				if len(snippet) > 300 {
					snippet = snippet[:300] + "..."
				}
				resultStr.WriteString(fmt.Sprintf("   %s\n", snippet))
			}
			resultStr.WriteString("\n")
		}

		resultStr.WriteString("=== END OF SEARCH RESULTS ===\n")
		return resultStr.String()
	default:
		return fmt.Sprintf("[Error: Unknown search tool: %s]", toolName)
	}
}

// executeLocalCommand 执行本地命令
func (s *AIChatService) executeLocalCommand(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("[Error: %v]\nOutput: %s", err, string(output))
	}
	return string(output)
}

// extractFilePath 从文本中提取文件路径
// sanitizeUTF8 清理字符串中的无效 UTF-8 字符
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	// 如果字符串包含无效 UTF-8，逐字节检查并重建
	var buf strings.Builder
	buf.Grow(len(s))

	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			// 跳过无效字节
			s = s[1:]
			continue
		}
		buf.WriteRune(r)
		s = s[size:]
	}

	return buf.String()
}

func extractFilePath(text string) string {
	// 简单的路径提取逻辑
	re := regexp.MustCompile(`['"\s]([\w\-\./]+)['"\s]`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return "."
}

// extractSearchQuery 从文本中提取搜索查询
func extractSearchQuery(text string) string {
	// 简单的查询提取逻辑
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "```") {
			return line
		}
	}
	return text
}

// GetEvaluation 获取会话评估结果
func (s *AIChatService) GetEvaluation(sessionID string) (*EvaluationReport, error) {
	if s.evaluator == nil {
		return nil, fmt.Errorf("evaluator not initialized")
	}
	return s.evaluator.GetReport(sessionID)
}

// extractCommand 从文本中提取命令
func extractCommand(text string) string {
	// 简单的命令提取逻辑
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "```") {
			return line
		}
	}
	return text
}
