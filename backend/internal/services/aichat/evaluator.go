package aichat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/services/ai"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DialogEvaluator 对话评估器
type DialogEvaluator struct {
	db        *gorm.DB
	aiService ai.AIService
}

// NewDialogEvaluator 创建评估器
func NewDialogEvaluator(db *gorm.DB, aiService ai.AIService) *DialogEvaluator {
	return &DialogEvaluator{
		db:        db,
		aiService: aiService,
	}
}

// EvaluationReport 评估报告
type EvaluationReport struct {
	ID               string                `json:"id"`
	SessionID        string                `json:"session_id"`
	CreatedAt        time.Time             `json:"created_at"`
	OverallScore     float64               `json:"overall_score"`
	TopicAdherence   TopicAdherenceScore   `json:"topic_adherence"`
	RoleConsistency  RoleConsistencyScore  `json:"role_consistency"`
	LogicalCoherence LogicalCoherenceScore `json:"logical_coherence"`
	Engagement       EngagementScore       `json:"engagement"`
	Summary          string                `json:"summary"`
	Highlights       []DialogHighlight     `json:"highlights"`
	Suggestions      []string              `json:"suggestions"`
}

// TopicAdherenceScore 主题紧扣度评分
type TopicAdherenceScore struct {
	Score       float64 `json:"score"`
	Explanation string  `json:"explanation"`
}

// RoleConsistencyScore 角色一致性评分
type RoleConsistencyScore struct {
	AgentA      RoleScore `json:"agent_a"`
	AgentB      RoleScore `json:"agent_b"`
	Overall     float64   `json:"overall"`
	Explanation string    `json:"explanation"`
}

// RoleScore 角色评分
type RoleScore struct {
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	Explanation string  `json:"explanation"`
}

// LogicalCoherenceScore 逻辑连贯性评分
type LogicalCoherenceScore struct {
	Score          float64 `json:"score"`
	FlowQuality    float64 `json:"flow_quality"`
	Contradictions int     `json:"contradictions"`
	Explanation    string  `json:"explanation"`
}

// EngagementScore 精彩程度评分
type EngagementScore struct {
	Score       float64 `json:"score"`
	Creativity  float64 `json:"creativity"`
	Depth       float64 `json:"depth"`
	Explanation string  `json:"explanation"`
}

// DialogHighlight 对话亮点
type DialogHighlight struct {
	Round     int    `json:"round"`
	AgentName string `json:"agent_name"`
	Content   string `json:"content"`
	Reason    string `json:"reason"`
}

// Evaluate 评估对话
func (e *DialogEvaluator) Evaluate(session *models.AIChatSession) (*EvaluationReport, error) {
	// 获取所有消息
	var messages []models.AIChatMessage
	if err := e.db.Where("session_id = ?", session.ID).Order("round, id").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	if len(messages) < 2 {
		return nil, fmt.Errorf("not enough messages to evaluate")
	}

	// 构建评估提示词
	evalPrompt := e.buildEvaluationPrompt(session, messages)

	// 调用 AI 进行评估
	evalResult, err := e.callEvaluationAI(evalPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call evaluation AI: %w", err)
	}

	// 解析评估结果
	report, err := e.parseEvaluationResult(session.ID, evalResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse evaluation result: %w", err)
	}

	// 保存评估报告
	if err := e.saveReport(report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	return report, nil
}

// buildEvaluationPrompt 构建评估提示词
func (e *DialogEvaluator) buildEvaluationPrompt(session *models.AIChatSession, messages []models.AIChatMessage) string {
	var prompt strings.Builder

	prompt.WriteString("你是一位专业的对话质量评估专家。请对以下AI-AI对话进行全面评估。\n\n")

	// 基本信息
	prompt.WriteString("=== 对话基本信息 ===\n")
	prompt.WriteString(fmt.Sprintf("主题：%s\n", session.Topic))
	prompt.WriteString(fmt.Sprintf("全局限定：%s\n", session.GlobalConstraint))
	prompt.WriteString(fmt.Sprintf("总轮数：%d\n", session.CurrentRound))
	prompt.WriteString(fmt.Sprintf("AI-A（%s）：%s\n", session.AgentAName, session.AgentARole))
	prompt.WriteString(fmt.Sprintf("AI-B（%s）：%s\n\n", session.AgentBName, session.AgentBRole))

	// 对话内容
	prompt.WriteString("=== 对话内容 ===\n")
	for _, msg := range messages {
		if msg.MessageType == models.MessageTypeText {
			prompt.WriteString(fmt.Sprintf("[%s - 第%d轮] %s\n", msg.AgentName, msg.Round, msg.Content))
		}
	}

	// 评估要求
	prompt.WriteString("\n=== 评估要求 ===\n")
	prompt.WriteString("请从以下几个维度进行评估（每项满分10分）：\n\n")

	prompt.WriteString("1. 主题紧扣度 (Topic Adherence)\n")
	prompt.WriteString("   - 对话是否始终围绕主题展开\n")
	prompt.WriteString("   - 是否有偏离主题的讨论\n\n")

	prompt.WriteString("2. 角色扮演一致性 (Role Consistency)\n")
	prompt.WriteString("   - 每个AI是否始终保持其角色设定\n")
	prompt.WriteString("   - 语言风格是否符合角色特点\n\n")

	prompt.WriteString("3. 逻辑连贯性 (Logical Coherence)\n")
	prompt.WriteString("   - 对话流程是否自然流畅\n")
	prompt.WriteString("   - 是否存在逻辑矛盾\n\n")

	prompt.WriteString("4. 精彩程度 (Engagement)\n")
	prompt.WriteString("   - 对话是否有深度和见解\n")
	prompt.WriteString("   - 是否有创造性的观点\n")
	prompt.WriteString("   - 是否引人入胜\n\n")

	prompt.WriteString("请按以下JSON格式输出评估结果：\n")
	prompt.WriteString(`{
  "overall_score": 8.5,
  "topic_adherence": {
    "score": 9.0,
    "explanation": "说明"
  },
  "role_consistency": {
    "agent_a": {"name": "名字", "score": 8.5, "explanation": "说明"},
    "agent_b": {"name": "名字", "score": 8.0, "explanation": "说明"},
    "overall": 8.25,
    "explanation": "总体说明"
  },
  "logical_coherence": {
    "score": 8.0,
    "flow_quality": 8.5,
    "contradictions": 0,
    "explanation": "说明"
  },
  "engagement": {
    "score": 8.5,
    "creativity": 9.0,
    "depth": 8.0,
    "explanation": "说明"
  },
  "summary": "总体评价摘要",
  "highlights": [
    {"round": 3, "agent_name": "名字", "content": "亮点内容", "reason": "亮点原因"}
  ],
  "suggestions": ["改进建议1", "改进建议2"]
}`)

	return prompt.String()
}

// callEvaluationAI 调用 AI 进行评估
func (e *DialogEvaluator) callEvaluationAI(prompt string) (string, error) {
	messages := []models.Message{
		{Role: "system", Content: "你是一位专业的对话质量评估专家。请客观、公正地评估对话质量。"},
		{Role: "user", Content: prompt},
	}

	request := ai.ChatRequest{
		Model:       "deepseek-chat",
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   2000,
		Stream:      false,
	}

	stream, err := e.aiService.ChatStream(context.Background(), &request)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	for chunk := range stream {
		if chunk.Done {
			break
		}
		result.WriteString(chunk.Content)
	}

	return result.String(), nil
}

// parseEvaluationResult 解析评估结果
func (e *DialogEvaluator) parseEvaluationResult(sessionID string, result string) (*EvaluationReport, error) {
	// 提取 JSON 部分
	jsonStart := strings.Index(result, "{")
	jsonEnd := strings.LastIndex(result, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("invalid evaluation result format")
	}

	jsonStr := result[jsonStart : jsonEnd+1]

	// 解析 JSON
	var rawReport struct {
		OverallScore   float64 `json:"overall_score"`
		TopicAdherence struct {
			Score       float64 `json:"score"`
			Explanation string  `json:"explanation"`
		} `json:"topic_adherence"`
		RoleConsistency struct {
			AgentA struct {
				Name        string  `json:"name"`
				Score       float64 `json:"score"`
				Explanation string  `json:"explanation"`
			} `json:"agent_a"`
			AgentB struct {
				Name        string  `json:"name"`
				Score       float64 `json:"score"`
				Explanation string  `json:"explanation"`
			} `json:"agent_b"`
			Overall     float64 `json:"overall"`
			Explanation string  `json:"explanation"`
		} `json:"role_consistency"`
		LogicalCoherence struct {
			Score          float64 `json:"score"`
			FlowQuality    float64 `json:"flow_quality"`
			Contradictions int     `json:"contradictions"`
			Explanation    string  `json:"explanation"`
		} `json:"logical_coherence"`
		Engagement struct {
			Score       float64 `json:"score"`
			Creativity  float64 `json:"creativity"`
			Depth       float64 `json:"depth"`
			Explanation string  `json:"explanation"`
		} `json:"engagement"`
		Summary    string `json:"summary"`
		Highlights []struct {
			Round     int    `json:"round"`
			AgentName string `json:"agent_name"`
			Content   string `json:"content"`
			Reason    string `json:"reason"`
		} `json:"highlights"`
		Suggestions []string `json:"suggestions"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawReport); err != nil {
		return nil, fmt.Errorf("failed to unmarshal evaluation result: %w", err)
	}

	// 转换为报告结构
	highlights := make([]DialogHighlight, len(rawReport.Highlights))
	for i, h := range rawReport.Highlights {
		highlights[i] = DialogHighlight{
			Round:     h.Round,
			AgentName: h.AgentName,
			Content:   h.Content,
			Reason:    h.Reason,
		}
	}

	report := &EvaluationReport{
		ID:           uuid.New().String(),
		SessionID:    sessionID,
		CreatedAt:    time.Now(),
		OverallScore: rawReport.OverallScore,
		TopicAdherence: TopicAdherenceScore{
			Score:       rawReport.TopicAdherence.Score,
			Explanation: rawReport.TopicAdherence.Explanation,
		},
		RoleConsistency: RoleConsistencyScore{
			AgentA: RoleScore{
				Name:        rawReport.RoleConsistency.AgentA.Name,
				Score:       rawReport.RoleConsistency.AgentA.Score,
				Explanation: rawReport.RoleConsistency.AgentA.Explanation,
			},
			AgentB: RoleScore{
				Name:        rawReport.RoleConsistency.AgentB.Name,
				Score:       rawReport.RoleConsistency.AgentB.Score,
				Explanation: rawReport.RoleConsistency.AgentB.Explanation,
			},
			Overall:     rawReport.RoleConsistency.Overall,
			Explanation: rawReport.RoleConsistency.Explanation,
		},
		LogicalCoherence: LogicalCoherenceScore{
			Score:          rawReport.LogicalCoherence.Score,
			FlowQuality:    rawReport.LogicalCoherence.FlowQuality,
			Contradictions: rawReport.LogicalCoherence.Contradictions,
			Explanation:    rawReport.LogicalCoherence.Explanation,
		},
		Engagement: EngagementScore{
			Score:       rawReport.Engagement.Score,
			Creativity:  rawReport.Engagement.Creativity,
			Depth:       rawReport.Engagement.Depth,
			Explanation: rawReport.Engagement.Explanation,
		},
		Summary:     rawReport.Summary,
		Highlights:  highlights,
		Suggestions: rawReport.Suggestions,
	}

	return report, nil
}

// saveReport 保存评估报告
func (e *DialogEvaluator) saveReport(report *EvaluationReport) error {
	reportData := models.JSONMap{
		"overall_score":     report.OverallScore,
		"topic_adherence":   report.TopicAdherence,
		"role_consistency":  report.RoleConsistency,
		"logical_coherence": report.LogicalCoherence,
		"engagement":        report.Engagement,
		"summary":           report.Summary,
		"highlights":        report.Highlights,
		"suggestions":       report.Suggestions,
	}

	dbReport := &models.EvaluationReport{
		ID:        report.ID,
		SessionID: report.SessionID,
		Report:    reportData,
	}

	return e.db.Create(dbReport).Error
}

// GetReport 获取评估报告
func (e *DialogEvaluator) GetReport(sessionID string) (*EvaluationReport, error) {
	var dbReport models.EvaluationReport
	if err := e.db.Where("session_id = ?", sessionID).First(&dbReport).Error; err != nil {
		return nil, err
	}

	// 解析报告数据
	reportData, _ := json.Marshal(dbReport.Report)
	var report EvaluationReport
	if err := json.Unmarshal(reportData, &report); err != nil {
		return nil, err
	}

	report.ID = dbReport.ID
	report.SessionID = dbReport.SessionID
	report.CreatedAt = dbReport.CreatedAt

	return &report, nil
}
