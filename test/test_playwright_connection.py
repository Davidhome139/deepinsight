#!/usr/bin/env python3
"""
测试 Playwright 连接问题
"""

def simulate_playwright_connection():
    """模拟 Playwright 连接逻辑"""
    
    print("模拟 Playwright 连接逻辑")
    print("=" * 80)
    
    # 模拟 connectServer 方法中的逻辑
    server_name = "playwright"
    server_connected = False
    server_client = None
    
    print(f"1. 在 connectServer 方法中:")
    print(f"   服务器名称: {server_name}")
    
    # 模拟特殊处理
    if server_name == "playwright":
        print(f"   ✓ 检测到 Playwright 服务器，跳过预连接")
        print(f"   ✓ 标记服务器为已连接 (server.Connected = True)")
        print(f"   ✓ 客户端为 nil (server.Client = nil)")
        server_connected = True
        server_client = None
    
    print()
    print(f"2. 在 executeMCPTool 方法中:")
    print(f"   检查连接状态: !server.Connected && server.Client == nil")
    print(f"   server.Connected = {server_connected}")
    print(f"   server.Client = {server_client}")
    
    # 模拟检查逻辑
    if not server_connected and server_client is None:
        print(f"   ✗ 条件成立: !{server_connected} && {server_client} == None")
        print(f"   ✗ 返回错误: MCP server not connected: {server_name}")
    else:
        print(f"   ✓ 条件不成立: !{server_connected} && {server_client} == None")
        print(f"   ✓ 继续执行工具调用")
    
    print()
    print(f"3. 在 CallTool 方法中:")
    print(f"   检查服务器名称: {server_name}")
    
    if server_name == "playwright":
        print(f"   ✓ 检测到 Playwright 服务器，调用 callPlaywrightToolOnDemand")
        print(f"   ✓ 按需启动 Playwright MCP 服务器")
        print(f"   ✓ 执行工具调用")
    else:
        print(f"   ✗ 不是 Playwright 服务器，走正常流程")
    
    print()
    print("问题分析:")
    print("-" * 80)
    print("从终端日志看:")
    print("  [Chat] MCP server not connected: playwright")
    print()
    print("这意味着在 executeMCPTool 中，条件 !server.Connected && server.Client == nil 成立了")
    print("也就是说: server.Connected = False 或者 server.Client != nil")
    print()
    print("可能的原因:")
    print("1. server.Connected 是 False (没有被标记为已连接)")
    print("2. server.Client 不是 nil (但这是不可能的，因为 Playwright 是按需连接的)")
    print()
    print("最可能的原因是: server.Connected 是 False")
    print("这意味着 connectServer 方法没有正确标记 Playwright 服务器为已连接")
    print()
    print("检查点:")
    print("1. connectServer 方法是否被调用?")
    print("2. 在 connectServer 中，Playwright 的特殊处理是否执行?")
    print("3. 服务器是否被正确添加到 m.servers 映射中?")
    print("4. GetServer 方法是否返回正确的服务器对象?")

def simulate_get_server_logic():
    """模拟 GetServer 方法逻辑"""
    
    print()
    print("模拟 GetServer 方法逻辑")
    print("=" * 80)
    
    # 模拟服务器映射
    servers = {}
    
    print(f"1. 初始状态: servers 映射中有 {len(servers)} 个服务器")
    
    # 模拟 Discover 方法
    print(f"2. 调用 Discover 方法")
    print(f"   - 从配置加载服务器")
    print(f"   - 对于每个启用的服务器，调用 connectServer")
    
    # 模拟添加 Playwright 服务器
    playwright_server = {
        "name": "playwright",
        "connected": False,  # 初始状态
        "client": None
    }
    
    # 模拟 connectServer 中的特殊处理
    if playwright_server["name"] == "playwright":
        print(f"   - 检测到 Playwright 服务器，执行特殊处理")
        print(f"   - 标记为已连接: playwright_server['connected'] = True")
        print(f"   - 客户端为 nil: playwright_server['client'] = None")
        playwright_server["connected"] = True
        playwright_server["client"] = None
    
    # 添加到映射
    servers["playwright"] = playwright_server
    print(f"   - 服务器添加到映射: servers['playwright'] = {playwright_server}")
    
    print()
    print(f"3. GetServer 方法被调用")
    print(f"   - 查找服务器: servers.get('playwright')")
    
    server = servers.get("playwright")
    if server:
        print(f"   ✓ 找到服务器: {server}")
        print(f"   - server['connected'] = {server['connected']}")
        print(f"   - server['client'] = {server['client']}")
    else:
        print(f"   ✗ 未找到服务器")
    
    print()
    print("关键问题:")
    print("-" * 80)
    print("如果 GetServer 返回的服务器对象中 connected = False，那么:")
    print("在 executeMCPTool 中: !server.Connected && server.Client == nil")
    print("会变成: !False && None == None")
    print("也就是: True && True = True")
    print("所以会返回错误: MCP server not connected: playwright")
    print()
    print("解决方案:")
    print("1. 确保 connectServer 方法正确标记 Playwright 服务器为已连接")
    print("2. 确保服务器对象被正确更新到 m.servers 映射中")
    print("3. 确保 GetServer 方法返回的是最新的服务器对象")

def check_actual_code():
    """检查实际代码中的问题"""
    
    print()
    print("检查实际代码")
    print("=" * 80)
    
    print("1. connectServer 方法中的 Playwright 特殊处理:")
    print("   ```go")
    print("   // Special handling for playwright - we don't pre-connect")
    print("   if server.Name == \"playwright\" || strings.Contains(strings.ToLower(server.Name), \"playwright\") {")
    print("       fmt.Println(\"[MCP] ========== SKIPPING playwright connection (on-demand mode) ==========\")")
    print("       fmt.Println(\"[MCP] Server config: name=\", server.Name, \"type=\", server.Type, \"command=\", server.Command)")
    print("       // Mark as connected but without actual client")
    print("       server.Connected = true")
    print("       server.Client = nil")
    print("       m.mu.Lock()")
    print("       m.servers[server.Name] = &server")
    print("       m.mu.Unlock()")
    print("       fmt.Println(\"[MCP] ========== playwright marked as connected (no actual client) ==========\")")
    print("       return")
    print("   }")
    print("   ```")
    
    print()
    print("2. 问题可能出现在:")
    print("   a) server 变量是值传递，修改后没有正确更新")
    print("   b) m.servers[server.Name] 存储的是指针，但 server 是局部变量")
    print("   c) 并发问题: 多个 goroutine 同时修改服务器状态")
    
    print()
    print("3. 建议的修复:")
    print("   ```go")
    print("   // Special handling for playwright - we don't pre-connect")
    print("   if server.Name == \"playwright\" || strings.Contains(strings.ToLower(server.Name), \"playwright\") {")
    print("       fmt.Println(\"[MCP] ========== SKIPPING playwright connection (on-demand mode) ==========\")")
    print("       fmt.Println(\"[MCP] Server config: name=\", server.Name, \"type=\", server.Type, \"command=\", server.Command)")
    print("       // Mark as connected but without actual client")
    print("       server.Connected = true")
    print("       server.Client = nil")
    print("       // 确保我们存储的是正确的指针")
    print("       m.mu.Lock()")
    print("       m.servers[server.Name] = &server")
    print("       m.mu.Unlock()")
    print("       fmt.Println(\"[MCP] ========== playwright marked as connected (no actual client) ==========\")")
    print("       return")
    print("   }")
    print("   ```")
    
    print()
    print("4. 或者，更好的修复是在 executeMCPTool 中:")
    print("   ```go")
    print("   // Check connection status for external MCP servers")
    print("   if !server.Connected && server.Client == nil {")
    print("       // 对于 Playwright 服务器，即使没有客户端也允许继续")
    print("       if serverName != \"playwright\" {")
    print("           fmt.Printf(\"[Chat] MCP server not connected: %s\\n\", serverName)")
    print("           return fmt.Sprintf(\"[MCP Error: Server '%s' not connected]\\n\\nPlease check MCP server configuration in Settings.\", serverName)")
    print("       }")
    print("   }")
    print("   ```")

if __name__ == "__main__":
    simulate_playwright_connection()
    simulate_get_server_logic()
    check_actual_code()