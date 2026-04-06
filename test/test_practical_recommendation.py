#!/usr/bin/env python3
"""
实用的 MCP 工具推荐测试 - 基于实际工具描述和关键词
"""

# 基于实际工具描述的关键词映射
tool_keywords = {
    # Playwright 工具
    "playwright/browser_navigate": {
        "keywords": ["browser", "webpage", "website", "navigate", "open browser", "open website", "visit", "go to", "浏览器", "网页", "网站", "打开浏览器", "打开网站", "访问", "前往"],
        "description": "Navigate to a URL",
        "weight": 0.9
    },
    "playwright/browser_click": {
        "keywords": ["click", "tap", "press button", "button", "点击", "按钮", "按下"],
        "description": "Perform click on a web page",
        "weight": 0.9
    },
    "playwright/browser_take_screenshot": {
        "keywords": ["screenshot", "take screenshot", "capture screenshot", "picture", "image", "截图", "屏幕截图", "捕捉画面"],
        "description": "Take a screenshot of the current page",
        "weight": 0.9
    },
    "playwright/browser_hover": {
        "keywords": ["hover", "mouse over", "悬停", "鼠标悬停"],
        "description": "Hover over element on page",
        "weight": 0.9
    },
    "playwright/browser_drag": {
        "keywords": ["drag", "drop", "drag and drop", "拖放", "拖动"],
        "description": "Perform drag and drop between two elements",
        "weight": 0.9
    },
    "playwright/browser_fill_form": {
        "keywords": ["fill form", "fill out form", "form", "input", "field", "填写表单", "表单", "输入"],
        "description": "Fill multiple form fields",
        "weight": 0.9
    },
    "playwright/browser_type": {
        "keywords": ["type", "type text", "enter text", "input text", "输入文本", "打字", "输入"],
        "description": "Type text into editable element",
        "weight": 0.8
    },
    "playwright/browser_select_option": {
        "keywords": ["select option", "choose option", "dropdown", "select", "选择选项", "下拉菜单", "选择"],
        "description": "Select an option in a dropdown",
        "weight": 0.9
    },
    "playwright/browser_file_upload": {
        "keywords": ["upload file", "upload files", "file upload", "上传文件", "文件上传"],
        "description": "Upload one or multiple files",
        "weight": 0.9
    },
    "playwright/browser_evaluate": {
        "keywords": ["javascript", "execute javascript", "run javascript", "js", "执行javascript", "运行javascript", "执行代码"],
        "description": "Evaluate JavaScript expression on page or element",
        "weight": 0.8
    },
    "playwright/browser_run_code": {
        "keywords": ["run code", "playwright code", "code snippet", "运行代码", "代码片段"],
        "description": "Run Playwright code snippet",
        "weight": 0.8
    },
    "playwright/browser_console_messages": {
        "keywords": ["console messages", "console logs", "console", "log", "控制台消息", "控制台日志", "日志"],
        "description": "Returns all console messages",
        "weight": 0.8
    },
    "playwright/browser_network_requests": {
        "keywords": ["network requests", "network traffic", "monitor network", "网络请求", "网络流量", "监控网络"],
        "description": "Returns all network requests since loading the page",
        "weight": 0.8
    },
    "playwright/browser_snapshot": {
        "keywords": ["snapshot", "accessibility snapshot", "快照", "可访问性快照"],
        "description": "Capture accessibility snapshot of the current page",
        "weight": 0.7
    },
    "playwright/browser_tabs": {
        "keywords": ["tab", "new tab", "close tab", "switch tab", "标签页", "新建标签页", "关闭标签页", "切换标签页"],
        "description": "List, create, close, or select a browser tab",
        "weight": 0.8
    },
    "playwright/browser_wait_for": {
        "keywords": ["wait for", "wait until", "wait", "等待", "等待直到"],
        "description": "Wait for text to appear or disappear or a specified time to pass",
        "weight": 0.8
    },
    "playwright/browser_press_key": {
        "keywords": ["press key", "keyboard", "key", "按键", "键盘", "按下键"],
        "description": "Press a key on the keyboard",
        "weight": 0.8
    },
    "playwright/browser_resize": {
        "keywords": ["resize", "resize window", "window size", "调整窗口大小", "调整大小", "窗口尺寸"],
        "description": "Resize the browser window",
        "weight": 0.8
    },
    "playwright/browser_handle_dialog": {
        "keywords": ["dialog", "alert", "confirm", "popup", "对话框", "弹窗", "确认框"],
        "description": "Handle a dialog",
        "weight": 0.8
    },
    "playwright/browser_navigate_back": {
        "keywords": ["go back", "back", "previous", "返回", "后退", "上一页"],
        "description": "Go back to the previous page in the history",
        "weight": 0.8
    },
    "playwright/browser_close": {
        "keywords": ["close", "exit", "quit", "关闭", "退出"],
        "description": "Close the page",
        "weight": 0.7
    },
    
    # Context7 工具
    "context7/query-docs": {
        "keywords": ["documentation", "docs", "how to use", "query docs", "search docs", "code example", "code examples", "latest example", "latest examples", "查询文档", "搜索文档", "获取文档", "文档查询", "代码示例", "最新示例", "如何使用", "api文档", "教程", "示例"],
        "description": "Retrieves and queries up-to-date documentation and code examples",
        "weight": 0.9
    },
    "context7/resolve-library-id": {
        "keywords": ["library", "package", "framework", "库", "包", "框架"],
        "description": "Resolves a package/product name to a Context7-compatible library ID",
        "weight": 0.7
    },
    
    # Brave Search 工具
    "brave-search/search": {
        "keywords": ["search", "find", "look up", "google", "搜索", "查找", "查询"],
        "description": "Search the web using Brave Search",
        "weight": 0.6
    },
    
    # Filesystem 工具
    "filesystem/read_file": {
        "keywords": ["file", "read file", "open file", "文件", "读取文件", "打开文件"],
        "description": "Read a file from the filesystem",
        "weight": 0.7
    },
    "filesystem/list_directory": {
        "keywords": ["list files", "directory", "folder", "list directory", "列出文件", "目录", "文件夹"],
        "description": "List files in a directory",
        "weight": 0.7
    },
    
    # Terminal 工具
    "terminal/execute_command": {
        "keywords": ["command", "run", "execute", "shell", "命令", "运行", "执行", "终端"],
        "description": "Execute a command in the terminal",
        "weight": 0.7
    }
}

def recommend_tool_practical(query):
    """实用的工具推荐算法"""
    query_lower = query.lower()
    matches = []
    
    # 查找所有匹配的工具
    for tool_name, tool_info in tool_keywords.items():
        for keyword in tool_info["keywords"]:
            if keyword in query_lower:
                # 计算匹配分数
                score = tool_info["weight"]
                
                # 如果关键词是短语（包含空格），增加分数
                if " " in keyword:
                    score += 0.1
                
                # 如果关键词是中文，稍微调整分数
                if any('\u4e00' <= char <= '\u9fff' for char in keyword):
                    score += 0.05
                
                matches.append({
                    'tool': tool_name,
                    'keyword': keyword,
                    'score': min(score, 1.0),  # 确保不超过1.0
                    'description': tool_info["description"]
                })
                break  # 每个工具只匹配一个关键词
    
    # 选择最佳匹配
    if matches:
        # 按分数降序排序
        matches.sort(key=lambda x: x['score'], reverse=True)
        best_match = matches[0]
        
        return {
            'recommended_tool': best_match['tool'],
            'confidence': best_match['score'],
            'reasoning': f"匹配关键词: '{best_match['keyword']}' - {best_match['description']}",
            'all_matches': matches[:3]  # 显示前3个匹配
        }
    else:
        return {
            'recommended_tool': None,
            'confidence': 0.0,
            'reasoning': "未找到匹配的工具",
            'all_matches': []
        }

# 测试查询
test_queries = [
    # 实际场景测试
    "打开浏览器访问百度网站并截图",
    "如何获取mark3labs的最新示例代码",
    "查询playwright的文档和示例",
    "搜索关于浏览器自动化的资料",
    "在网页上点击登录按钮",
    "填写注册表单并提交",
    "上传文件到网站",
    "执行JavaScript代码获取页面标题",
    "监控网页的网络请求",
    "创建新的浏览器标签页",
    "等待页面加载完成",
    "按回车键提交表单",
    "调整浏览器窗口大小为1024x768",
    "处理网页弹出的确认对话框",
    "返回上一页",
    "关闭浏览器",
    
    # 中文查询
    "截图当前页面",
    "悬停在菜单上查看子菜单",
    "拖放文件到上传区域",
    "选择下拉菜单中的选项",
    "查看控制台错误消息",
    "获取网络请求的详细信息",
    "切换到第二个标签页",
    "等待元素出现",
    "按下Ctrl+C复制",
    "调整窗口大小适应屏幕",
    "点击确认对话框的确定按钮",
    "后退到上一页面",
    
    # 文档查询
    "获取react的最新文档",
    "查询express.js的使用示例",
    "搜索python flask的教程",
    "查找node.js的API文档",
    "获取最新版本的示例代码",
    
    # 混合查询
    "我想搜索playwright的文档",
    "打开浏览器搜索资料",
    "查询如何截图网页",
    "浏览器自动化测试",
]

print("实用的 MCP 工具推荐测试")
print("=" * 100)

for query in test_queries:
    result = recommend_tool_practical(query)
    print(f"\n查询: {query}")
    print(f"推荐工具: {result['recommended_tool'] or '无'}")
    print(f"置信度: {result['confidence']:.2f}")
    print(f"推理: {result['reasoning']}")
    
    if result['all_matches'] and len(result['all_matches']) > 1:
        print("其他匹配:")
        for match in result['all_matches'][1:]:
            print(f"  - {match['tool']} (关键词: '{match['keyword']}', 分数: {match['score']:.2f})")
    
    print("-" * 100)

# 分析推荐结果
print("\n\n推荐结果分析:")
print("=" * 100)

# 统计推荐分布
recommendation_counts = {}
for query in test_queries:
    result = recommend_tool_practical(query)
    if result['recommended_tool']:
        # 提取服务器名称
        if "/" in result['recommended_tool']:
            server = result['recommended_tool'].split("/")[0]
            recommendation_counts[server] = recommendation_counts.get(server, 0) + 1

print("推荐分布:")
total_queries = len(test_queries)
for server, count in sorted(recommendation_counts.items(), key=lambda x: x[1], reverse=True):
    percentage = (count / total_queries) * 100
    print(f"  {server}: {count} 次 ({percentage:.1f}%)")

# 未匹配的查询
unmatched = [q for q in test_queries if recommend_tool_practical(q)['confidence'] == 0.0]
if unmatched:
    print(f"\n未匹配的查询 ({len(unmatched)} 个):")
    for q in unmatched[:5]:  # 只显示前5个
        print(f"  - {q}")

# 测试特定场景
print("\n\n特定场景测试:")
print("=" * 100)

specific_scenarios = [
    ("打开浏览器访问百度网站", "playwright/browser_navigate"),
    ("截图页面", "playwright/browser_take_screenshot"),
    ("点击按钮", "playwright/browser_click"),
    ("查询文档", "context7/query-docs"),
    ("搜索资料", "brave-search/search"),
    ("读取文件", "filesystem/read_file"),
    ("运行命令", "terminal/execute_command"),
]

print("场景测试结果:")
for query, expected in specific_scenarios:
    result = recommend_tool_practical(query)
    match = result['recommended_tool'] == expected
    print(f"  {query:30} -> {result['recommended_tool'] or '无':40} {'✓' if match else '✗'} (置信度: {result['confidence']:.2f})")