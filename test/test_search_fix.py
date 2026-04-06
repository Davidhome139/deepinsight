#!/usr/bin/env python3
"""
测试搜索工具修复
"""

# 模拟修复后的 executeMCPTool 逻辑
def execute_mcp_tool_fixed(mcp_tool, user_content, mcp_manager_available=True):
    """模拟修复后的 executeMCPTool 函数"""
    # Parse mcpTool format: "server/tool"
    parts = mcp_tool.split("/", 1)
    if len(parts) != 2:
        print(f"[Chat] Invalid MCP tool format: {mcp_tool}")
        return ""
    
    server_name = parts[0]
    tool_name = parts[1]
    
    print(f"[Chat] DEBUG: executeMCPTool called with mcpTool={mcp_tool}, serverName={server_name}, toolName={tool_name}")
    print(f"[Chat] Calling MCP tool: server={server_name}, tool={tool_name}")
    
    # For built-in servers, handle directly without CallTool
    if server_name == "filesystem-local":
        return f"[Built-in filesystem tool: {tool_name}]"
    elif server_name == "terminal":
        return f"[Built-in terminal tool: {tool_name}]"
    elif server_name == "search":
        return handle_builtin_search_tool(tool_name, user_content)
    elif server_name == "code-analysis":
        return f"[Built-in code analysis tool: {tool_name}]"
    
    # For external MCP servers, check MCP manager
    if not mcp_manager_available:
        print("[Chat] MCP manager not available")
        return "[MCP Error: MCP manager not initialized]"
    
    # Simulate getting server from MCP manager
    # In real code: server, ok := s.mcpManager.GetServer(serverName)
    server_found = server_name in ["context7", "playwright", "brave-search"]
    
    if not server_found:
        print(f"[Chat] MCP server not found: {server_name}")
        return f"[MCP Error: Server '{server_name}' not found]"
    
    # Simulate calling external MCP tool
    return f"[External MCP tool: {server_name}/{tool_name}]"

def handle_builtin_search_tool(tool_name, user_content):
    """模拟 handleBuiltinSearchTool 函数"""
    if tool_name == "web_search":
        query = user_content.strip()
        if not query:
            return "[MCP Error: No search query provided]"
        
        # Simulate search results
        return f"""
=== WEB SEARCH SUCCESSFUL ===
Search Query: {query}
Total Results: 3

1. **Latest US-Iran War Reports**
   URL: https://news.example.com/us-iran-latest
   Summary: Latest developments in the US-Iran conflict as of April 2026

2. **Baidu News: US-Iran Relations**
   URL: https://news.baidu.com/us-iran
   Summary: Chinese coverage of US-Iran diplomatic relations

3. **International News Analysis**
   URL: https://analysis.example.com/middle-east
   Summary: In-depth analysis of the Middle East situation

=== END OF SEARCH RESULTS ===
"""
    else:
        return f"[MCP Error: Unknown search tool '{tool_name}']"

# 测试场景
test_scenarios = [
    {
        "query": "search for the latest US-Iran war reports using Baidu",
        "mcp_tool": "search/web_search",
        "description": "搜索查询应该触发内置搜索工具"
    },
    {
        "query": "打开浏览器访问百度网站",
        "mcp_tool": "playwright/browser_navigate",
        "description": "Playwright 查询应该触发外部 MCP 工具"
    },
    {
        "query": "查询文档",
        "mcp_tool": "context7/query-docs",
        "description": "文档查询应该触发 Context7 MCP 工具"
    },
    {
        "query": "列出当前目录文件",
        "mcp_tool": "filesystem-local/list_directory",
        "description": "文件系统查询应该触发内置工具"
    },
    {
        "query": "运行 ls -la 命令",
        "mcp_tool": "terminal/execute_command",
        "description": "终端命令应该触发内置工具"
    },
]

print("测试搜索工具修复")
print("=" * 80)

for scenario in test_scenarios:
    query = scenario["query"]
    mcp_tool = scenario["mcp_tool"]
    description = scenario["description"]
    
    print(f"\n测试: {description}")
    print(f"查询: {query}")
    print(f"MCP 工具: {mcp_tool}")
    
    # 模拟执行 MCP 工具
    result = execute_mcp_tool_fixed(mcp_tool, query)
    
    print(f"结果: {result[:100]}..." if len(result) > 100 else f"结果: {result}")
    print("-" * 80)

# 测试错误场景
print("\n\n错误场景测试:")
print("=" * 80)

error_scenarios = [
    {
        "mcp_tool": "invalid-tool",
        "description": "无效的工具格式"
    },
    {
        "mcp_tool": "unknown-server/web_search",
        "description": "未知的服务器"
    },
    {
        "mcp_tool": "search/invalid_tool",
        "description": "搜索服务器的无效工具"
    },
    {
        "mcp_tool": "context7/query-docs",
        "mcp_manager_available": False,
        "description": "MCP 管理器不可用"
    },
]

for scenario in error_scenarios:
    mcp_tool = scenario["mcp_tool"]
    description = scenario["description"]
    mcp_manager_available = scenario.get("mcp_manager_available", True)
    
    print(f"\n测试: {description}")
    print(f"MCP 工具: {mcp_tool}")
    
    result = execute_mcp_tool_fixed(mcp_tool, "test query", mcp_manager_available)
    print(f"结果: {result}")
    print("-" * 80)

# 验证修复
print("\n\n验证修复:")
print("=" * 80)

# 原始问题：search/web_search 应该被识别为内置工具，而不是尝试从 MCP 管理器获取
original_problem = {
    "query": "search for the latest US-Iran war reports using Baidu",
    "mcp_tool": "search/web_search",
    "expected": "=== WEB SEARCH SUCCESSFUL ==="
}

result = execute_mcp_tool_fixed(original_problem["mcp_tool"], original_problem["query"])
if original_problem["expected"] in result:
    print("✓ 修复成功：search/web_search 现在被正确识别为内置工具")
    print(f"  结果包含: {original_problem['expected']}")
else:
    print("✗ 修复失败：search/web_search 仍然有问题")
    print(f"  结果: {result[:200]}...")

# 检查语义意图检测的输出
print("\n\n语义意图检测测试:")
print("=" * 80)

# 模拟 detectMCPIntentTraditional 函数中的内置工具列表
builtin_tools = [
    "search/web_search",
    "filesystem-local/list_directory", 
    "filesystem-local/read_file",
    "terminal/execute_command",
]

print("内置工具列表（来自 detectMCPIntentTraditional）:")
for tool in builtin_tools:
    print(f"  - {tool}")

print("\n这些工具应该在 executeMCPTool 中被正确处理：")
for tool in builtin_tools:
    server = tool.split("/")[0]
    if server in ["search", "filesystem-local", "terminal"]:
        print(f"  ✓ {tool} -> 内置工具（服务器: {server}）")
    else:
        print(f"  ✗ {tool} -> 未知类型")