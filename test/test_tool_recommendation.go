package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"backend/internal/config"
	"backend/internal/services/agent"
)

func main() {
	// 创建模拟的 MCP 管理器
	mcpManager := &agent.MCPManager{
		Servers: make(map[string]*config.MCPServer),
	}

	// 创建工具推荐器
	recommender := agent.NewToolRecommender(mcpManager, nil)

	// 测试查询
	testQueries := []string{
		"打开浏览器访问百度网站",
		"navigate to google.com",
		"take a screenshot of the page",
		"点击页面上的按钮",
		"获取网页内容",
		"how to use react hooks",
		"查询文档",
		"浏览器自动化",
		"playwright 截图",
		"爬取网站数据",
	}

	// 模拟服务器摘要
	serverSummaries := map[string]string{
		"context7":      "文档查询和库信息检索",
		"playwright":    "浏览器自动化和网页交互",
		"brave-search":  "网页搜索和信息检索",
		"filesystem":    "文件系统操作和文件管理",
		"terminal":      "命令执行和系统操作",
	}

	fmt.Println("测试工具推荐系统:")
	fmt.Println("=" * 60)

	for _, query := range testQueries {
		fmt.Printf("\n查询: %s\n", query)
		
		// 使用关键词推荐
		result := recommender.KeywordBasedRecommendation(query, serverSummaries)
		
		if result != nil && result.RecommendedTool != "" {
			fmt.Printf("推荐工具: %s\n", result.RecommendedTool)
			fmt.Printf("置信度: %.2f\n", result.Confidence)
			fmt.Printf("推理: %s\n", result.Reasoning)
			if result.ServerSummary != "" {
				fmt.Printf("服务器摘要: %s\n", result.ServerSummary)
			}
		} else {
			fmt.Println("没有推荐工具")
		}
		
		fmt.Println("-" * 40)
	}

	// 测试 detectMCPIntentSemantic 中的特殊处理
	fmt.Println("\n\n测试 detectMCPIntentSemantic 中的特殊处理:")
	fmt.Println("=" * 60)

	// 测试包含文档关键词的查询
	docQueries := []string{
		"获取 react 的最新示例",
		"查询 next.js 文档",
		"how to use vue.js with examples",
		"获取 mark3labs 的最新示例",
	}

	for _, query := range docQueries {
		fmt.Printf("\n查询: %s\n", query)
		
		// 检查是否包含文档关键词
		contentLower := strings.ToLower(query)
		chineseKeywords := []string{"示例", "文档", "帮助", "查询", "搜索", "获取", "最新"}
		englishKeywords := []string{"example", "examples", "documentation", "docs", "help", "query", "search", "get", "latest"}

		hasDocIntent := false
		for _, keyword := range chineseKeywords {
			if strings.Contains(query, keyword) {
				hasDocIntent = true
				break
			}
		}
		if !hasDocIntent {
			for _, keyword := range englishKeywords {
				if strings.Contains(contentLower, keyword) {
					hasDocIntent = true
					break
				}
			}
		}

		if hasDocIntent {
			fmt.Println("检测到文档意图: 应该推荐 query-docs")
		} else {
			fmt.Println("没有检测到文档意图")
		}
	}
}