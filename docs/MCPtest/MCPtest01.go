package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	fmt.Println("程序启动...")
	// 1. 创建基于stdio的MCP客户端
	fmt.Println("正在创建客户端...")
	// 在Docker容器中运行时，使用全局安装的playwright-mcp
	stdio := transport.NewStdio("playwright-mcp", nil)
	mcpClient := client.NewClient(stdio)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 启动客户端
	if err := mcpClient.Start(ctx); err != nil {
		log.Fatalf("启动客户端失败: %v", err)
	}
	defer mcpClient.Close()
	fmt.Println("客户端创建成功")

	// 2. 初始化连接
	fmt.Println("初始化MCP客户端...")
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "playwright-go-client",
				Version: "1.0.0",
			},
		},
	}

	initResult, err := mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	fmt.Printf("✅ 初始化成功，服务器信息: %s %s\n\n",
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version)

	// 3. 列出所有工具
	toolsResult, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("获取工具列表失败: %v", err)
	}

	fmt.Println("📋 可用工具:")
	for _, tool := range toolsResult.Tools {
		fmt.Printf("  • %s - %s\n", tool.Name, tool.Description)
	}

	// 4. 先导航到网页
	fmt.Println("\n🌐 导航到网页...")
	navigateResult, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "browser_navigate",
			Arguments: map[string]interface{}{
				"url": "https://www.baidu.com",
			},
		},
	})
	if err != nil {
		log.Fatalf("导航失败: %v", err)
	}
	if navigateResult.IsError {
		if len(navigateResult.Content) > 0 {
			log.Printf("导航错误: %v", navigateResult.Content[0])
		}
	} else {
		fmt.Println("✅ 导航成功!")
	}

	// 5. 获取页面快照以查找元素引用
	fmt.Println("\n📸 获取页面快照...")
	snapshotResult, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "browser_snapshot",
			Arguments: map[string]interface{}{},
		},
	})
	if err != nil {
		log.Fatalf("获取快照失败: %v", err)
	}

	textboxRef := "" // 存储找到的文本框引用ID

	if snapshotResult.IsError {
		if len(snapshotResult.Content) > 0 {
			log.Printf("快照错误: %v", snapshotResult.Content[0])
		}
	} else {
		fmt.Println("✅ 快照获取成功!")
		// 打印快照的部分内容以便调试
		if len(snapshotResult.Content) > 0 {
			snapshotContent := snapshotResult.Content[0]
			// 打印更多字符以查找搜索框
			snapshotText := fmt.Sprintf("%v", snapshotContent)
			// 查找搜索输入框的引用
			if len(snapshotText) > 5000 {
				fmt.Printf("快照预览 (前5000字符):\n%s...\n", snapshotText[:5000])
			} else {
				fmt.Printf("快照内容:\n%s\n", snapshotText)
			}
			// 尝试在快照中搜索输入框
			if strings.Contains(snapshotText, "input") || strings.Contains(snapshotText, "search") {
				fmt.Println("✅ 快照中包含输入框或搜索框")
			}
			// 查找所有textbox引用
			if strings.Contains(snapshotText, "textbox") {
				fmt.Println("✅ 快照中包含文本框")
				// 提取textbox的引用ID
				lines := strings.Split(snapshotText, "\n")
				for _, line := range lines {
					if strings.Contains(line, "textbox") && strings.Contains(line, "ref=") {
						fmt.Printf("找到文本框行: %s\n", line)
						// 提取ref值
						if refStart := strings.Index(line, "ref="); refStart != -1 {
							refStart += 4
							if refEnd := strings.Index(line[refStart:], "]"); refEnd != -1 {
								ref := line[refStart : refStart+refEnd]
								fmt.Printf("文本框引用ID: %s\n", ref)
								// 使用找到的第一个文本框引用
								if textboxRef == "" {
									textboxRef = ref
									fmt.Printf("✅ 将使用引用ID: %s\n", textboxRef)
								}
							}
						}
					}
				}
			}
		}
	}

	// 如果没有找到引用，使用默认值
	if textboxRef == "" {
		textboxRef = "e36" // 默认引用ID
		fmt.Printf("⚠️ 未找到文本框引用，使用默认值: %s\n", textboxRef)
	}

	// 6. 点击搜索框使其获得焦点
	fmt.Println("\n🖱️ 点击搜索框...")
	clickResult, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "browser_click",
			Arguments: map[string]interface{}{
				"ref":      textboxRef, // 使用找到的引用ID
				"selector": "#kw",      // 百度搜索框的选择器
			},
		},
	})
	if err != nil {
		log.Fatalf("点击失败: %v", err)
	}
	if clickResult.IsError {
		if len(clickResult.Content) > 0 {
			log.Printf("点击错误: %v", clickResult.Content[0])
		}
	} else {
		fmt.Println("✅ 点击成功!")
	}

	// 7. 在搜索框中输入文本
	fmt.Println("\n🖋️ 在搜索框中输入文本...")
	typeResult, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "browser_type",
			Arguments: map[string]interface{}{
				"ref":      textboxRef, // 使用找到的引用ID
				"selector": "#kw",      // 百度搜索框的选择器
				"text":     "Playwright自动化测试",
			},
		},
	})
	if err != nil {
		log.Fatalf("调用工具失败: %v", err)
	}

	if typeResult.IsError {
		if len(typeResult.Content) > 0 {
			log.Printf("工具执行错误: %v", typeResult.Content[0])
		} else {
			log.Printf("工具执行错误: 未知错误")
		}
	} else {
		fmt.Println("✅ 文本输入成功!")
	}
}
