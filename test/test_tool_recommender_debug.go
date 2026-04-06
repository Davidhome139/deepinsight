package main

import (
	"fmt"
	"strings"
)

// 模拟工具推荐器的逻辑
func simulateToolRecommender(query string) {
	fmt.Printf("测试查询: %s\n", query)
	fmt.Println(strings.Repeat("=", 80))

	// 模拟关键词模式
	keywordPatterns := map[string]struct {
		Server string
		Tool   string
		Weight float64
	}{
		// Playwright 关键词
		"playwright": {"playwright", "browser_navigate", 1.0},
		"浏览器":        {"playwright", "browser_navigate", 0.9},
		"查找":         {"playwright", "browser_navigate", 0.7},
		"百度":         {"playwright", "browser_navigate", 0.6},

		// 搜索关键词
		"搜索": {"brave-search", "search", 0.6},
		// 注意：查找有两个权重，一个是 playwright 的 0.7，一个是 brave-search 的 0.6
		// 但在 map 中不能有重复键，所以这里只保留一个
		"查询": {"brave-search", "search", 0.6},

		// 内置搜索工具关键词
		"web_search": {"search", "web_search", 0.8},
	}

	queryLower := strings.ToLower(query)

	// 查找匹配的关键词
	var matches []struct {
		Keyword string
		Server  string
		Tool    string
		Weight  float64
	}

	for keyword, toolInfo := range keywordPatterns {
		if strings.Contains(queryLower, keyword) {
			matches = append(matches, struct {
				Keyword string
				Server  string
				Tool    string
				Weight  float64
			}{
				Keyword: keyword,
				Server:  toolInfo.Server,
				Tool:    toolInfo.Tool,
				Weight:  toolInfo.Weight,
			})
		}
	}

	fmt.Printf("匹配的关键词数量: %d\n", len(matches))
	for i, match := range matches {
		fmt.Printf("  匹配 %d: '%s' -> %s/%s (权重: %.2f)\n",
			i+1, match.Keyword, match.Server, match.Tool, match.Weight)
	}

	// 选择最佳匹配
	if len(matches) > 0 {
		// 按权重排序（降序）
		for i := 0; i < len(matches); i++ {
			for j := i + 1; j < len(matches); j++ {
				if matches[i].Weight < matches[j].Weight {
					matches[i], matches[j] = matches[j], matches[i]
				} else if matches[i].Weight == matches[j].Weight {
					// 如果权重相等，选择更长的关键词（更具体）
					if len(matches[i].Keyword) < len(matches[j].Keyword) {
						matches[i], matches[j] = matches[j], matches[i]
					}
				}
			}
		}

		bestMatch := matches[0]
		fmt.Printf("\n最佳匹配: '%s' -> %s/%s (权重: %.2f)\n",
			bestMatch.Keyword, bestMatch.Server, bestMatch.Tool, bestMatch.Weight)

		// 检查置信度是否足够高
		confidence := bestMatch.Weight
		if confidence >= 0.7 {
			fmt.Printf("置信度: %.2f >= 0.7 ✓ 工具推荐器会推荐这个工具\n", confidence)
			fmt.Printf("推荐的工具: %s/%s\n", bestMatch.Server, bestMatch.Tool)
		} else {
			fmt.Printf("置信度: %.2f < 0.7 ✗ 工具推荐器可能不会推荐（置信度太低）\n", confidence)
		}
	} else {
		fmt.Println("没有匹配的关键词")
	}

	fmt.Println()
}

func main() {
	// 测试用户查询
	queries := []string{
		"使用playwright查找百度最新的美伊战报。",
		"搜索最新的美伊战报",
		"打开浏览器访问百度",
		"使用playwright打开百度",
		"查找百度上的新闻",
		"playwright browser navigate",
	}

	for _, query := range queries {
		simulateToolRecommender(query)
	}

	// 分析冲突
	fmt.Println("\n\n分析关键词冲突:")
	fmt.Println(strings.Repeat("=", 80))

	// 检查 "查找" 关键词的冲突
	query := "使用playwright查找百度最新的美伊战报。"
	queryLower := strings.ToLower(query)

	fmt.Printf("查询: %s\n", query)
	fmt.Printf("包含 'playwright': %v\n", strings.Contains(queryLower, "playwright"))
	fmt.Printf("包含 '查找': %v\n", strings.Contains(queryLower, "查找"))
	fmt.Printf("包含 '百度': %v\n", strings.Contains(queryLower, "百度"))

	// 权重比较
	playwrightWeight := 1.0
	// findWeight := 0.6 // 注意：查找有两个权重，一个是 playwright 的 0.7，一个是 brave-search 的 0.6

	fmt.Printf("\n权重比较:\n")
	fmt.Printf("  'playwright' 权重: %.2f\n", playwrightWeight)
	fmt.Printf("  '查找' 权重 (playwright): 0.7\n")
	fmt.Printf("  '查找' 权重 (brave-search): 0.6\n")
	fmt.Printf("  '百度' 权重: 0.6\n")

	fmt.Printf("\n结论: 'playwright' 权重最高 (1.0)，应该推荐 playwright/browser_navigate\n")
}
