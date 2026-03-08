package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"backend/internal/models"
	"backend/internal/services/aichat"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// AIChatWebSocket AI-AI 聊天 WebSocket 处理器
type AIChatWebSocket struct {
	service   *aichat.AIChatService
	jwtSecret string
}

// NewAIChatWebSocket 创建 WebSocket 处理器
func NewAIChatWebSocket(service *aichat.AIChatService, jwtSecret string) *AIChatWebSocket {
	return &AIChatWebSocket{
		service:   service,
		jwtSecret: jwtSecret,
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (h *AIChatWebSocket) HandleWebSocket(c *gin.Context) {
	sessionID := c.Param("id")

	// Try to get userID from context (set by auth middleware)
	_, exists := c.Get("userID")

	// If not in context, try to validate token from query string
	if !exists {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte(h.jwtSecret), nil
		})
		if err != nil || !parsedToken.Valid {
			fmt.Printf("[AIChatWS] JWT invalid: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
	}

	// 升级连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 创建事件通道
	eventChan := make(chan aichat.StreamEvent, 100)

	// 订阅会话事件
	h.service.SubscribeToSession(sessionID, eventChan)
	defer h.service.UnsubscribeFromSession(sessionID, eventChan)

	// 启动 goroutine 发送事件到客户端
	go func() {
		for event := range eventChan {
			data, _ := json.Marshal(event)
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}()

	// 处理客户端消息
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// 解析客户端消息
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			continue
		}

		// 处理不同类型的消息
		switch clientMsg.Type {
		case "director_command":
			h.handleDirectorCommand(sessionID, clientMsg.Data)
		case "control":
			h.handleControlCommand(sessionID, clientMsg.Data)
		}
	}
}

// ClientMessage 客户端消息
type ClientMessage struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// handleDirectorCommand 处理导演指令
func (h *AIChatWebSocket) handleDirectorCommand(sessionID string, data map[string]interface{}) {
	targetAgent, _ := data["targetAgent"].(string)
	command, _ := data["command"].(string)

	if targetAgent == "" || command == "" {
		return
	}

	var insertAfterRound *int
	if round, ok := data["insertAfterRound"].(float64); ok {
		r := int(round)
		insertAfterRound = &r
	}

	cmd := &models.DirectorCommand{
		TargetAgent:      targetAgent,
		Command:          command,
		InsertAfterRound: insertAfterRound,
	}

	h.service.InjectDirectorCommand(sessionID, cmd)
}

// handleControlCommand 处理控制命令
func (h *AIChatWebSocket) handleControlCommand(sessionID string, data map[string]interface{}) {
	action, _ := data["action"].(string)

	switch action {
	case "pause":
		h.service.PauseSession(sessionID)
	case "resume":
		h.service.StartSession(sessionID)
	case "stop":
		h.service.StopSession(sessionID)
	}
}

// AIChatStatusResponse 会话状态响应
type AIChatStatusResponse struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	CurrentRound  int       `json:"current_round"`
	MaxRounds     int       `json:"max_rounds"`
	TokenUsage    TokenInfo `json:"token_usage"`
	LastMessageAt time.Time `json:"last_message_at,omitempty"`
}

type TokenInfo struct {
	AgentA int `json:"agent_a"`
	AgentB int `json:"agent_b"`
	Total  int `json:"total"`
}

// GetSessionStatus 获取会话实时状态
func (h *AIChatHandler) GetSessionStatus(c *gin.Context) {
	id := c.Param("id")
	session, err := h.service.GetSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// 获取最后一条消息的时间
	var lastMessageAt time.Time
	if len(session.Messages) > 0 {
		lastMessageAt = session.Messages[len(session.Messages)-1].Timestamp
	}

	response := AIChatStatusResponse{
		ID:            session.ID,
		Status:        string(session.Status),
		CurrentRound:  session.CurrentRound,
		MaxRounds:     session.MaxRounds,
		LastMessageAt: lastMessageAt,
		TokenUsage: TokenInfo{
			AgentA: session.TokenUsage.AgentAInput + session.TokenUsage.AgentAOutput,
			AgentB: session.TokenUsage.AgentBInput + session.TokenUsage.AgentBOutput,
			Total:  session.TokenUsage.Total,
		},
	}

	c.JSON(http.StatusOK, response)
}
