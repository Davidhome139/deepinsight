package aichat

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
)

// ExportService 导出服务
type ExportService struct {
	db *gorm.DB
}

// NewExportService 创建导出服务
func NewExportService(db *gorm.DB) *ExportService {
	return &ExportService{db: db}
}

// ExportFormat 导出格式
type ExportFormat string

const (
	ExportFormatMarkdown ExportFormat = "markdown"
	ExportFormatJSON     ExportFormat = "json"
	ExportFormatText     ExportFormat = "text"
)

// ExportSession 导出会话
func (s *ExportService) ExportSession(sessionID string, format ExportFormat) (string, error) {
	// 获取会话
	var session models.AIChatSession
	if err := s.db.Preload("Messages").First(&session, "id = ?", sessionID).Error; err != nil {
		return "", err
	}

	switch format {
	case ExportFormatMarkdown:
		return s.exportToMarkdown(&session)
	case ExportFormatJSON:
		return s.exportToJSON(&session)
	case ExportFormatText:
		return s.exportToText(&session)
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportToMarkdown 导出为 Markdown
func (s *ExportService) exportToMarkdown(session *models.AIChatSession) (string, error) {
	var sb strings.Builder

	// 标题
	sb.WriteString(fmt.Sprintf("# %s\n\n", session.Title))

	// 元信息
	sb.WriteString("## 会话信息\n\n")
	sb.WriteString(fmt.Sprintf("- **主题**: %s\n", session.Topic))
	sb.WriteString(fmt.Sprintf("- **全局限定**: %s\n", session.GlobalConstraint))
	sb.WriteString(fmt.Sprintf("- **状态**: %s\n", session.Status))
	sb.WriteString(fmt.Sprintf("- **总轮数**: %d/%d\n", session.CurrentRound, session.MaxRounds))
	sb.WriteString(fmt.Sprintf("- **创建时间**: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05")))
	if session.CompletedAt != nil {
		sb.WriteString(fmt.Sprintf("- **完成时间**: %s\n", session.CompletedAt.Format("2006-01-02 15:04:05")))
	}
	sb.WriteString("\n")

	// AI 配置
	sb.WriteString("## AI 配置\n\n")
	sb.WriteString(fmt.Sprintf("### %s (AI-A)\n\n", session.AgentAName))
	sb.WriteString(fmt.Sprintf("- **角色**: %s\n", session.AgentARole))
	sb.WriteString(fmt.Sprintf("- **风格**: %s / %s\n", session.AgentAStyle.LanguageStyle, session.AgentAStyle.KnowledgeLevel))
	sb.WriteString(fmt.Sprintf("- **语气**: %s\n", session.AgentAStyle.Tone))
	sb.WriteString(fmt.Sprintf("- **模型**: %s\n", session.AgentAModel))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("### %s (AI-B)\n\n", session.AgentBName))
	sb.WriteString(fmt.Sprintf("- **角色**: %s\n", session.AgentBRole))
	sb.WriteString(fmt.Sprintf("- **风格**: %s / %s\n", session.AgentBStyle.LanguageStyle, session.AgentBStyle.KnowledgeLevel))
	sb.WriteString(fmt.Sprintf("- **语气**: %s\n", session.AgentBStyle.Tone))
	sb.WriteString(fmt.Sprintf("- **模型**: %s\n", session.AgentBModel))
	sb.WriteString("\n")

	// Token 使用
	sb.WriteString("## Token 使用统计\n\n")
	sb.WriteString(fmt.Sprintf("- **%s**: 输入 %d / 输出 %d\n", session.AgentAName, session.TokenUsage.AgentAInput, session.TokenUsage.AgentAOutput))
	sb.WriteString(fmt.Sprintf("- **%s**: 输入 %d / 输出 %d\n", session.AgentBName, session.TokenUsage.AgentBInput, session.TokenUsage.AgentBOutput))
	sb.WriteString(fmt.Sprintf("- **总计**: %d\n", session.TokenUsage.Total))
	sb.WriteString("\n")

	// 对话内容
	sb.WriteString("## 对话内容\n\n")

	currentRound := 0
	for _, msg := range session.Messages {
		if msg.Round != currentRound {
			currentRound = msg.Round
			sb.WriteString(fmt.Sprintf("### 第 %d 轮\n\n", currentRound))
		}

		// 消息头部
		icon := "🤖"
		if msg.AgentID == "agent_a" {
			icon = "🅰️"
		} else if msg.AgentID == "agent_b" {
			icon = "🅱️"
		}

		sb.WriteString(fmt.Sprintf("**%s %s** *(%s)*\n\n", icon, msg.AgentName, msg.Timestamp.Format("15:04:05")))

		// 消息内容
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")

		// 工具调用
		if msg.ToolCalls != nil && len(msg.ToolCalls) > 0 {
			sb.WriteString("*工具调用:*\n")
			for name, args := range msg.ToolCalls {
				sb.WriteString(fmt.Sprintf("- `%s`: %v\n", name, args))
			}
			sb.WriteString("\n")
		}

		// 工具结果
		if msg.ToolResults != nil && len(msg.ToolResults) > 0 {
			sb.WriteString("*工具结果:*\n")
			for name, result := range msg.ToolResults {
				sb.WriteString(fmt.Sprintf("- `%s`: %v\n", name, result))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("---\n\n")
	}

	// 页脚
	sb.WriteString("\n---\n\n")
	sb.WriteString(fmt.Sprintf("*导出时间: %s*\n", time.Now().Format("2006-01-02 15:04:05")))

	return sb.String(), nil
}

// exportToJSON 导出为 JSON
func (s *ExportService) exportToJSON(session *models.AIChatSession) (string, error) {
	exportData := struct {
		Session  *models.AIChatSession `json:"session"`
		Exported time.Time             `json:"exported_at"`
		Version  string                `json:"version"`
	}{
		Session:  session,
		Exported: time.Now(),
		Version:  "1.0",
	}

	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// exportToText 导出为纯文本
func (s *ExportService) exportToText(session *models.AIChatSession) (string, error) {
	var sb strings.Builder

	// 标题
	sb.WriteString(strings.Repeat("=", 50))
	sb.WriteString(fmt.Sprintf("\n%s\n", session.Title))
	sb.WriteString(strings.Repeat("=", 50))
	sb.WriteString("\n\n")

	// 元信息
	sb.WriteString("【会话信息】\n")
	sb.WriteString(fmt.Sprintf("主题: %s\n", session.Topic))
	sb.WriteString(fmt.Sprintf("限定: %s\n", session.GlobalConstraint))
	sb.WriteString(fmt.Sprintf("状态: %s\n", session.Status))
	sb.WriteString(fmt.Sprintf("轮数: %d/%d\n", session.CurrentRound, session.MaxRounds))
	sb.WriteString("\n")

	// AI 配置
	sb.WriteString("【AI配置】\n")
	sb.WriteString(fmt.Sprintf("AI-A: %s (%s)\n", session.AgentAName, session.AgentAModel))
	sb.WriteString(fmt.Sprintf("AI-B: %s (%s)\n", session.AgentBName, session.AgentBModel))
	sb.WriteString("\n")

	// 对话内容
	sb.WriteString("【对话内容】\n")
	sb.WriteString(strings.Repeat("-", 50))
	sb.WriteString("\n\n")

	currentRound := 0
	for _, msg := range session.Messages {
		if msg.Round != currentRound {
			currentRound = msg.Round
			sb.WriteString(fmt.Sprintf("--- 第 %d 轮 ---\n\n", currentRound))
		}

		sb.WriteString(fmt.Sprintf("[%s] %s:\n", msg.Timestamp.Format("15:04:05"), msg.AgentName))
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")
	}

	// 页脚
	sb.WriteString(strings.Repeat("-", 50))
	sb.WriteString(fmt.Sprintf("\n导出时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))

	return sb.String(), nil
}
