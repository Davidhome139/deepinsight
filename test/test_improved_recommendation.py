#!/usr/bin/env python3
"""
测试改进后的 MCP 工具推荐算法 - 基于实际工具描述
"""

import json

# 模拟从文档中提取的工具描述
def load_tool_descriptions():
    """模拟从 MCP 文档中加载工具描述"""
    # 基于 playwright_docs.json 和 context7_docs.json 的工具描述
    tools = {
        "playwright": {
            "browser_navigate": "Navigate to a URL",
            "browser_click": "Perform click on a web page",
            "browser_take_screenshot": "Take a screenshot of the current page. You can't perform actions based on the screenshot, use browser_snapshot for actions.",
            "browser_hover": "Hover over element on page",
            "browser_drag": "Perform drag and drop between two elements",
            "browser_fill_form": "Fill multiple form fields",
            "browser_type": "Type text into editable element",
            "browser_select_option": "Select an option in a dropdown",
            "browser_file_upload": "Upload one or multiple files",
            "browser_evaluate": "Evaluate JavaScript expression on page or element",
            "browser_run_code": "Run Playwright code snippet",
            "browser_console_messages": "Returns all console messages",
            "browser_network_requests": "Returns all network requests since loading the page",
            "browser_snapshot": "Capture accessibility snapshot of the current page, this is better than screenshot",
            "browser_tabs": "List, create, close, or select a browser tab.",
            "browser_wait_for": "Wait for text to appear or disappear or a specified time to pass",
            "browser_press_key": "Press a key on the keyboard",
            "browser_resize": "Resize the browser window",
            "browser_handle_dialog": "Handle a dialog",
            "browser_navigate_back": "Go back to the previous page in the history",
            "browser_close": "Close the page",
        },
        "context7": {
            "resolve-library-id": "Resolves a package/product name to a Context7-compatible library ID and returns matching libraries.",
            "query-docs": "Retrieves and queries up-to-date documentation and code examples from Context7 for any programming library or framework.",
        },
        "brave-search": {
            "search": "Search the web using Brave Search",
        },
        "filesystem": {
            "read_file": "Read a file from the filesystem",
            "list_directory": "List files in a directory",
        },
        "terminal": {
            "execute_command": "Execute a command in the terminal",
        }
    }
    return tools

def semantic_match_score(query, tool_description):
    """计算查询和工具描述的语义匹配分数"""
    query_lower = query.lower()
    desc_lower = tool_description.lower()
    
    # 简单的关键词匹配算法
    score = 0.0
    
    # 检查工具描述中的关键词是否出现在查询中
    desc_words = set(desc_lower.split())
    query_words = set(query_lower.split())
    
    # 共同词汇数量
    common_words = desc_words.intersection(query_words)
    if common_words:
        score += len(common_words) * 0.1
    
    # 检查特定的关键词组合
    keyword_patterns = {
        # 导航相关
        "navigate": ["go to", "visit", "open", "browse", "website", "webpage"],
        "click": ["click", "tap", "press", "button"],
        "screenshot": ["screenshot", "capture", "picture", "image"],
        "form": ["form", "fill", "input", "field", "submit"],
        "upload": ["upload", "file", "attachment"],
        "javascript": ["javascript", "js", "execute", "code"],
        "console": ["console", "log", "message", "error"],
        "network": ["network", "request", "traffic", "monitor"],
        "tab": ["tab", "window", "browser"],
        "wait": ["wait", "load", "timeout"],
        "keyboard": ["key", "press", "enter", "type"],
        "resize": ["resize", "window", "size", "dimension"],
        "dialog": ["dialog", "alert", "confirm", "popup"],
        "back": ["back", "previous", "history"],
        "close": ["close", "exit", "quit"],
        
        # 文档相关
        "documentation": ["documentation", "docs", "manual", "guide"],
        "example": ["example", "sample", "code", "snippet"],
        "library": ["library", "package", "framework", "tool"],
        "query": ["query", "search", "find", "look up"],
    }
    
    for tool_keyword, query_keywords in keyword_patterns.items():
        if tool_keyword in desc_lower:
            for q_keyword in query_keywords:
                if q_keyword in query_lower:
                    score += 0.2
                    break
    
    return min(score, 1.0)

def recommend_tool_improved(query, tools):
    """改进的工具推荐算法"""
    query_lower = query.lower()
    recommendations = []
    
    # 第一阶段：关键词匹配（快速筛选）
    keyword_matches = []
    
    # 简化的关键词模式（用于快速筛选）
    quick_keywords = {
        # Context7
        "documentation": ("context7", "query-docs"),
        "docs": ("context7", "query-docs"),
        "example": ("context7", "query-docs"),
        "library": ("context7", "resolve-library-id"),
        "how to use": ("context7", "query-docs"),
        
        # Playwright - 导航
        "browser": ("playwright", "browser_navigate"),
        "website": ("playwright", "browser_navigate"),
        "navigate": ("playwright", "browser_navigate"),
        "open browser": ("playwright", "browser_navigate"),
        
        # Playwright - 交互
        "click": ("playwright", "browser_click"),
        "screenshot": ("playwright", "browser_take_screenshot"),
        "hover": ("playwright", "browser_hover"),
        "form": ("playwright", "browser_fill_form"),
        "upload": ("playwright", "browser_file_upload"),
        
        # 搜索
        "search": ("brave-search", "search"),
        "find": ("brave-search", "search"),
        
        # 文件系统
        "file": ("filesystem", "read_file"),
        "directory": ("filesystem", "list_directory"),
        
        # 终端
        "command": ("terminal", "execute_command"),
        "run": ("terminal", "execute_command"),
    }
    
    for keyword, (server, tool) in quick_keywords.items():
        if keyword in query_lower:
            keyword_matches.append((server, tool, 0.6))  # 基础置信度
    
    # 第二阶段：语义匹配（更精确）
    for server, server_tools in tools.items():
        for tool_name, tool_description in server_tools.items():
            # 计算语义匹配分数
            semantic_score = semantic_match_score(query, tool_description)
            
            if semantic_score > 0.3:  # 阈值
                # 结合关键词匹配结果
                base_score = semantic_score
                
                # 检查是否有快速关键词匹配
                for kw_server, kw_tool, kw_score in keyword_matches:
                    if server == kw_server and tool_name == kw_tool:
                        base_score = max(base_score, kw_score + 0.1)
                        break
                
                recommendations.append({
                    'server': server,
                    'tool': tool_name,
                    'score': base_score,
                    'description': tool_description[:100] + "..." if len(tool_description) > 100 else tool_description
                })
    
    # 排序并返回最佳推荐
    if recommendations:
        recommendations.sort(key=lambda x: x['score'], reverse=True)
        best = recommendations[0]
        
        return {
            'recommended_server': best['server'],
            'recommended_tool': f"{best['server']}/{best['tool']}",
            'confidence': best['score'],
            'reasoning': f"语义匹配分数: {best['score']:.2f} - {best['description']}",
            'all_recommendations': recommendations[:3]
        }
    elif keyword_matches:
        # 如果没有语义匹配，使用关键词匹配
        keyword_matches.sort(key=lambda x: x[2], reverse=True)
        best = keyword_matches[0]
        
        return {
            'recommended_server': best[0],
            'recommended_tool': f"{best[0]}/{best[1]}",
            'confidence': best[2],
            'reasoning': f"关键词匹配",
            'all_recommendations': []
        }
    else:
        return {
            'recommended_server': None,
            'recommended_tool': None,
            'confidence': 0.0,
            'reasoning': "未找到匹配的工具",
            'all_recommendations': []
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
]

# 加载工具描述
tools = load_tool_descriptions()

print("改进的 MCP 工具推荐算法测试")
print("=" * 100)

for query in test_queries:
    result = recommend_tool_improved(query, tools)
    print(f"\n查询: {query}")
    print(f"推荐工具: {result['recommended_tool'] or '无'}")
    print(f"置信度: {result['confidence']:.2f}")
    print(f"推理: {result['reasoning']}")
    
    if result['all_recommendations']:
        print("其他推荐:")
        for rec in result['all_recommendations'][:2]:  # 只显示前2个
            print(f"  - {rec['server']}/{rec['tool']} (分数: {rec['score']:.2f})")
    
    print("-" * 100)

# 分析推荐结果
print("\n\n推荐结果分析:")
print("=" * 100)

# 统计推荐分布
recommendation_counts = {}
for query in test_queries:
    result = recommend_tool_improved(query, tools)
    if result['recommended_server']:
        server = result['recommended_server']
        recommendation_counts[server] = recommendation_counts.get(server, 0) + 1

print("推荐分布:")
for server, count in sorted(recommendation_counts.items(), key=lambda x: x[1], reverse=True):
    percentage = (count / len(test_queries)) * 100
    print(f"  {server}: {count} 次 ({percentage:.1f}%)")

# 测试冲突场景
print("\n\n冲突场景测试:")
print("=" * 100)

conflict_scenarios = [
    ("我想搜索playwright的文档", "应该推荐 context7/query-docs"),
    ("打开浏览器搜索资料", "应该推荐 playwright/browser_navigate"),
    ("查询如何截图网页", "应该推荐 playwright/browser_take_screenshot"),
    ("获取最新代码示例", "应该推荐 context7/query-docs"),
    ("浏览器自动化测试", "应该推荐 playwright/browser_navigate"),
]

for query, expected in conflict_scenarios:
    result = recommend_tool_improved(query, tools)
    print(f"\n查询: {query}")
    print(f"期望: {expected}")
    print(f"实际: {result['recommended_tool'] or '无'} (置信度: {result['confidence']:.2f})")
    print(f"匹配: {'✓' if result['confidence'] > 0.5 else '✗'}")