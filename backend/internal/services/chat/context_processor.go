package chat

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"backend/internal/models"
)

// ContextProcessor 上下文处理器
type ContextProcessor struct{}

// NewContextProcessor 创建上下文处理器
func NewContextProcessor() *ContextProcessor {
	return &ContextProcessor{}
}

// ExtractKeyEntities 从对话历史中提取关键实体（仅基于上下文模式，无硬编码词表）
func (cp *ContextProcessor) ExtractKeyEntities(history []models.Message) []string {
	var entities []string
	locations := cp.extractLocationsFromContext(history)
	times := cp.extractTimeFromContext(history)
	entities = append(entities, locations...)
	entities = append(entities, times...)
	return cp.deduplicateAndFilter(entities)
}

// BuildContextualQuery 构建包含上下文的搜索查询
func (cp *ContextProcessor) BuildContextualQuery(currentQuery string, history []models.Message) string {
	// 提取关键实体
	entities := cp.ExtractKeyEntities(history)

	// 如果没有提取到实体，直接返回原查询
	if len(entities) == 0 {
		return currentQuery
	}

	// 智能组合查询
	var queryBuilder strings.Builder

	// 添加关键实体作为前缀
	for _, entity := range entities {
		queryBuilder.WriteString(entity)
		queryBuilder.WriteString(" ")
	}

	// 添加当前查询
	queryBuilder.WriteString(currentQuery)

	optimizedQuery := strings.TrimSpace(queryBuilder.String())
	fmt.Printf("[Context] Original query: '%s' -> Contextual query: '%s'\n", currentQuery, optimizedQuery)

	return optimizedQuery
}

// BuildEnhancedQuery 构建增强搜索查询（含对话上下文），供 web_search 使用
func (cp *ContextProcessor) BuildEnhancedQuery(currentQuery string, history []models.Message) string {
	return cp.BuildContextualQuery(currentQuery, history)
}

// extractLocationsFromContext 仅从上下文中按句式/模式提取地点，无硬编码词表
func (cp *ContextProcessor) extractLocationsFromContext(history []models.Message) []string {
	var locations []string
	seen := make(map[string]bool)
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || len(s) < 2 || len(s) > 20 || seen[s] {
			return
		}
		seen[s] = true
		locations = append(locations, s)
	}

	for _, msg := range history {
		if msg.Role != "user" {
			continue
		}
		content := msg.Content

		// “X的Y”：西雅图的天气、北京的酒店 -> X
		for _, sub := range regexp.MustCompile(`([^\s]{2,15})的`).FindAllStringSubmatch(content, -1) {
			if len(sub) >= 2 && isLikelyName(sub[1]) {
				add(sub[1])
			}
		}
		// “在X”“去X”“到X”：在东京、去北京
		for _, re := range []*regexp.Regexp{
			regexp.MustCompile(`[在去到]([^\s，。？！]{2,15})`),
		} {
			for _, sub := range re.FindAllStringSubmatch(content, -1) {
				if len(sub) >= 2 && isLikelyName(sub[1]) {
					add(sub[1])
				}
			}
		}
		// “X+天气/酒店/景点/推荐/攻略/美食/交通/住宿”：西雅图天气、北京酒店 -> X
		reSuffix := regexp.MustCompile(`([^\s]{2,10})(天气|酒店|景点|推荐|攻略|美食|交通|住宿)`)
		for _, sub := range reSuffix.FindAllStringSubmatch(content, -1) {
			if len(sub) >= 2 && isLikelyName(sub[1]) {
				add(sub[1])
			}
		}
	}
	return locations
}

func isLikelyName(s string) bool {
	if len(s) < 2 {
		return false
	}
	for _, r := range s {
		if !unicode.Is(unicode.Han, r) && !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// extractTimeFromContext 仅从上下文中按模式提取时间表述，无硬编码词表
func (cp *ContextProcessor) extractTimeFromContext(history []models.Message) []string {
	var times []string
	seen := make(map[string]bool)
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		times = append(times, s)
	}

	for _, msg := range history {
		if msg.Role != "user" {
			continue
		}
		content := msg.Content
		// 数字+月、中文数字+月
		for _, sub := range regexp.MustCompile(`\d{1,2}月|[一二三四五六七八九十百]+月`).FindAllString(content, -1) {
			add(sub)
		}
		// 最近/近期/现在/当前/今天/明天/本周/下周/春季/夏季/秋季/冬季 等
		for _, sub := range regexp.MustCompile(`最近|近期|现在|当前|目前|今天|明天|本周|下周|春季|夏季|秋季|冬季|年初|年中|年末`).FindAllString(content, -1) {
			add(sub)
		}
	}
	return times
}

// deduplicateAndFilter 去重和过滤
func (cp *ContextProcessor) deduplicateAndFilter(entities []string) []string {
	if len(entities) == 0 {
		return entities
	}

	// 去重
	seen := make(map[string]bool)
	var result []string

	for _, entity := range entities {
		if !seen[entity] && len(entity) > 1 {
			seen[entity] = true
			result = append(result, entity)
		}
	}

	// 限制数量，避免查询过长
	if len(result) > 5 {
		result = result[:5]
	}

	return result
}

// GenerateContextSummary 生成上下文摘要（用于AI模型）
func (cp *ContextProcessor) GenerateContextSummary(history []models.Message) string {
	if len(history) == 0 {
		return ""
	}

	var summary strings.Builder
	summary.WriteString("基于以下对话历史：\n")

	// 只保留最近几条用户消息作为上下文
	maxHistory := 3
	startIndex := len(history) - maxHistory
	if startIndex < 0 {
		startIndex = 0
	}

	for i := startIndex; i < len(history); i++ {
		msg := history[i]
		if msg.Role == "user" {
			summary.WriteString(fmt.Sprintf("- 用户询问：%s\n", msg.Content))
		}
	}

	summary.WriteString("请结合上述历史信息回答当前问题。")
	return summary.String()
}
