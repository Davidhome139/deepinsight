#!/usr/bin/env python3
"""
测试改进后的 MCP 工具推荐算法
基于 playwright_docs.json 和 context7_docs.json 中的工具描述
"""

# 模拟改进后的关键词模式
keyword_patterns = {
    # Context7 patterns - documentation and library queries
    "documentation": ("context7", "query-docs", 0.7),
    "library": ("context7", "resolve-library-id", 0.7),
    "api docs": ("context7", "query-docs", 0.8),
    "how to use": ("context7", "query-docs", 0.8),
    "query docs": ("context7", "query-docs", 0.9),
    "search docs": ("context7", "query-docs", 0.9),
    "code example": ("context7", "query-docs", 0.8),
    "code examples": ("context7", "query-docs", 0.8),
    "latest example": ("context7", "query-docs", 0.8),
    "latest examples": ("context7", "query-docs", 0.8),
    
    # Search patterns
    "search": ("brave-search", "search", 0.6),
    "find": ("brave-search", "search", 0.6),
    "look up": ("brave-search", "search", 0.7),
    "google": ("brave-search", "search", 0.8),
    
    # Playwright patterns - browser automation
    # Navigation and browsing
    "browser": ("playwright", "browser_navigate", 0.7),
    "webpage": ("playwright", "browser_navigate", 0.7),
    "website": ("playwright", "browser_navigate", 0.7),
    "navigate": ("playwright", "browser_navigate", 0.8),
    "open browser": ("playwright", "browser_navigate", 0.9),
    "open website": ("playwright", "browser_navigate", 0.9),
    "visit": ("playwright", "browser_navigate", 0.8),
    "go to": ("playwright", "browser_navigate", 0.8),
    
    # Interaction
    "click": ("playwright", "browser_click", 0.9),
    "hover": ("playwright", "browser_hover", 0.9),
    "drag": ("playwright", "browser_drag", 0.9),
    "drop": ("playwright", "browser_drag", 0.9),
    "drag and drop": ("playwright", "browser_drag", 1.0),
    
    # Screenshot and snapshot
    "screenshot": ("playwright", "browser_take_screenshot", 0.9),
    "take screenshot": ("playwright", "browser_take_screenshot", 1.0),
    "capture screenshot": ("playwright", "browser_take_screenshot", 1.0),
    "snapshot": ("playwright", "browser_snapshot", 0.9),
    "accessibility snapshot": ("playwright", "browser_snapshot", 1.0),
    
    # Form interaction
    "fill form": ("playwright", "browser_fill_form", 1.0),
    "fill out form": ("playwright", "browser_fill_form", 1.0),
    "type": ("playwright", "browser_type", 0.8),
    "type text": ("playwright", "browser_type", 0.9),
    "enter text": ("playwright", "browser_type", 0.9),
    "select option": ("playwright", "browser_select_option", 1.0),
    "choose option": ("playwright", "browser_select_option", 1.0),
    "dropdown": ("playwright", "browser_select_option", 0.8),
    
    # File operations
    "upload file": ("playwright", "browser_file_upload", 1.0),
    "upload files": ("playwright", "browser_file_upload", 1.0),
    
    # JavaScript and code execution
    "javascript": ("playwright", "browser_evaluate", 0.8),
    "execute javascript": ("playwright", "browser_evaluate", 1.0),
    "run javascript": ("playwright", "browser_evaluate", 1.0),
    "run code": ("playwright", "browser_run_code", 1.0),
    "playwright code": ("playwright", "browser_run_code", 1.0),
    
    # Debugging and monitoring
    "console messages": ("playwright", "browser_console_messages", 1.0),
    "console logs": ("playwright", "browser_console_messages", 1.0),
    "network requests": ("playwright", "browser_network_requests", 1.0),
    "network traffic": ("playwright", "browser_network_requests", 1.0),
    
    # Tab management
    "tab": ("playwright", "browser_tabs", 0.7),
    "new tab": ("playwright", "browser_tabs", 0.9),
    "close tab": ("playwright", "browser_tabs", 0.9),
    "switch tab": ("playwright", "browser_tabs", 0.9),
    
    # Waiting
    "wait for": ("playwright", "browser_wait_for", 0.9),
    "wait until": ("playwright", "browser_wait_for", 0.9),
    
    # Keyboard
    "press key": ("playwright", "browser_press_key", 0.9),
    "keyboard": ("playwright", "browser_press_key", 0.7),
    
    # Window management
    "resize": ("playwright", "browser_resize", 0.9),
    "resize window": ("playwright", "browser_resize", 1.0),
    "window size": ("playwright", "browser_resize", 0.8),
    
    # Dialog handling
    "dialog": ("playwright", "browser_handle_dialog", 0.8),
    "alert": ("playwright", "browser_handle_dialog", 0.9),
    "confirm": ("playwright", "browser_handle_dialog", 0.9),
    
    # Navigation
    "go back": ("playwright", "browser_navigate_back", 0.9),
    "back": ("playwright", "browser_navigate_back", 0.8),
    
    # General automation
    "scrape": ("playwright", "browser_navigate", 0.8),
    "crawl": ("playwright", "browser_navigate", 0.8),
    "automate": ("playwright", "browser_navigate", 0.8),
    "playwright": ("playwright", "browser_navigate", 1.0),
    
    # Chinese keywords for playwright
    "浏览器": ("playwright", "browser_navigate", 0.9),
    "网页": ("playwright", "browser_navigate", 0.8),
    "网站": ("playwright", "browser_navigate", 0.8),
    "打开浏览器": ("playwright", "browser_navigate", 1.0),
    "打开网站": ("playwright", "browser_navigate", 1.0),
    "点击": ("playwright", "browser_click", 1.0),
    "悬停": ("playwright", "browser_hover", 1.0),
    "截图": ("playwright", "browser_take_screenshot", 1.0),
    "拖放": ("playwright", "browser_drag", 1.0),
    "填写表单": ("playwright", "browser_fill_form", 1.0),
    "输入文本": ("playwright", "browser_type", 1.0),
    "选择选项": ("playwright", "browser_select_option", 1.0),
    "上传文件": ("playwright", "browser_file_upload", 1.0),
    "执行javascript": ("playwright", "browser_evaluate", 1.0),
    "运行代码": ("playwright", "browser_run_code", 1.0),
    "控制台消息": ("playwright", "browser_console_messages", 1.0),
    "网络请求": ("playwright", "browser_network_requests", 1.0),
    "标签页": ("playwright", "browser_tabs", 0.9),
    "等待": ("playwright", "browser_wait_for", 1.0),
    "按键": ("playwright", "browser_press_key", 1.0),
    "调整窗口大小": ("playwright", "browser_resize", 1.0),
    "对话框": ("playwright", "browser_handle_dialog", 1.0),
    "返回": ("playwright", "browser_navigate_back", 1.0),
    "爬取": ("playwright", "browser_navigate", 0.9),
    "爬虫": ("playwright", "browser_navigate", 0.9),
    "自动化": ("playwright", "browser_navigate", 0.9),
    
    # Chinese keywords for context7
    "查询文档": ("context7", "query-docs", 1.0),
    "搜索文档": ("context7", "query-docs", 1.0),
    "获取文档": ("context7", "query-docs", 1.0),
    "文档查询": ("context7", "query-docs", 1.0),
    "代码示例": ("context7", "query-docs", 1.0),
    "最新示例": ("context7", "query-docs", 1.0),
    "如何使用": ("context7", "query-docs", 1.0),
    "api文档": ("context7", "query-docs", 1.0),
}

def recommend_tool(query):
    """模拟改进后的推荐算法"""
    query_lower = query.lower()
    matches = []
    
    # 查找所有匹配的关键词
    for keyword, (server, tool, weight) in keyword_patterns.items():
        if keyword in query_lower:
            matches.append({
                'keyword': keyword,
                'server': server,
                'tool': tool,
                'weight': weight
            })
    
    # 选择最佳匹配（权重最高）
    if matches:
        # 按权重降序排序
        matches.sort(key=lambda x: (-x['weight'], -len(x['keyword'])))
        best_match = matches[0]
        
        return {
            'recommended_server': best_match['server'],
            'recommended_tool': f"{best_match['server']}/{best_match['tool']}",
            'confidence': best_match['weight'],
            'reasoning': f"匹配关键词: '{best_match['keyword']}' (权重: {best_match['weight']:.2f})",
            'all_matches': matches[:3]  # 显示前3个匹配
        }
    else:
        return {
            'recommended_server': None,
            'recommended_tool': None,
            'confidence': 0.0,
            'reasoning': "未找到匹配的工具",
            'all_matches': []
        }

# 测试查询
test_queries = [
    # Context7 相关查询
    "how to use react hooks",
    "query docs for next.js",
    "search docs for python",
    "get latest examples of mark3labs",
    "code examples for express.js",
    "查询文档",
    "搜索文档",
    "获取文档",
    "文档查询",
    "代码示例",
    "最新示例",
    "如何使用",
    "api文档",
    
    # Playwright 相关查询
    "open browser and navigate to google.com",
    "take a screenshot of the page",
    "click on the button",
    "hover over the element",
    "drag and drop the file",
    "fill out the form with my information",
    "type text into the input field",
    "select option from dropdown",
    "upload file to the website",
    "execute javascript on the page",
    "run playwright code",
    "get console messages",
    "monitor network requests",
    "create new tab",
    "wait for the page to load",
    "press enter key",
    "resize browser window",
    "handle alert dialog",
    "go back to previous page",
    "scrape website data",
    "automate browser testing",
    
    # 中文 Playwright 查询
    "打开浏览器访问百度网站",
    "截图页面",
    "点击按钮",
    "悬停在元素上",
    "拖放文件",
    "填写表单",
    "输入文本",
    "选择选项",
    "上传文件",
    "执行javascript代码",
    "运行代码",
    "查看控制台消息",
    "监控网络请求",
    "新建标签页",
    "等待页面加载",
    "按键",
    "调整窗口大小",
    "处理对话框",
    "返回上一页",
    "爬取网站数据",
    "浏览器自动化",
    
    # 混合查询（测试冲突解决）
    "search for browser automation documentation",
    "query docs for playwright",
    "how to use playwright for web scraping",
    "获取playwright的文档",
    "搜索浏览器自动化文档",
]

print("测试改进后的 MCP 工具推荐算法")
print("=" * 80)

for query in test_queries:
    result = recommend_tool(query)
    print(f"\n查询: {query}")
    print(f"推荐工具: {result['recommended_tool'] or '无'}")
    print(f"置信度: {result['confidence']:.2f}")
    print(f"推理: {result['reasoning']}")
    
    if result['all_matches']:
        print("所有匹配:")
        for match in result['all_matches']:
            print(f"  - '{match['keyword']}' -> {match['server']}/{match['tool']} (权重: {match['weight']:.2f})")
    
    print("-" * 80)

# 测试冲突解决
print("\n\n冲突解决测试:")
print("=" * 80)

conflict_queries = [
    "search docs for playwright",  # 应该匹配 context7/query-docs (权重 0.9) 而不是 brave-search/search (权重 0.6)
    "query docs for browser automation",  # 应该匹配 context7/query-docs
    "how to use playwright for web scraping",  # 应该匹配 context7/query-docs
    "获取playwright的文档",  # 应该匹配 context7/query-docs
    "搜索浏览器自动化文档",  # 应该匹配 context7/query-docs
]

for query in conflict_queries:
    result = recommend_tool(query)
    print(f"\n查询: {query}")
    print(f"推荐工具: {result['recommended_tool'] or '无'}")
    print(f"置信度: {result['confidence']:.2f}")
    print(f"推理: {result['reasoning']}")
    
    if result['all_matches']:
        print("所有匹配:")
        for match in result['all_matches']:
            print(f"  - '{match['keyword']}' -> {match['server']}/{match['tool']} (权重: {match['weight']:.2f})")