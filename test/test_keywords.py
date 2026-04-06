#!/usr/bin/env python3
"""
测试关键词匹配
"""

def test_keyword_matching():
    """测试关键词匹配"""
    
    # 关键词模式（从 tool_recommender.go 中提取）
    keyword_patterns = {
        # Context7 patterns
        "documentation": ("context7", "query-docs"),
        "library": ("context7", "resolve-library-id"),
        "api docs": ("context7", "query-docs"),
        "how to use": ("context7", "query-docs"),
        
        # Search patterns
        "search": ("brave-search", "search"),
        "find": ("brave-search", "search"),
        "look up": ("brave-search", "search"),
        "google": ("brave-search", "search"),
        
        # Filesystem patterns
        "file": ("filesystem", "read_file"),
        "read file": ("filesystem", "read_file"),
        "list files": ("filesystem", "list_directory"),
        "directory": ("filesystem", "list_directory"),
        
        # Terminal patterns
        "command": ("terminal", "execute_command"),
        "run": ("terminal", "execute_command"),
        "execute": ("terminal", "execute_command"),
        "shell": ("terminal", "execute_command"),
        
        # Playwright patterns
        "browser": ("playwright", "browser_navigate"),
        "webpage": ("playwright", "browser_navigate"),
        "website": ("playwright", "browser_navigate"),
        "navigate": ("playwright", "browser_navigate"),
        "open browser": ("playwright", "browser_navigate"),
        "open website": ("playwright", "browser_navigate"),
        "click": ("playwright", "browser_click"),
        "hover": ("playwright", "browser_hover"),
        "screenshot": ("playwright", "browser_take_screenshot"),
        "scrape": ("playwright", "browser_navigate"),
        "crawl": ("playwright", "browser_navigate"),
        "automate": ("playwright", "browser_navigate"),
        "playwright": ("playwright", "browser_navigate"),
        
        # Chinese keywords for playwright
        "浏览器": ("playwright", "browser_navigate"),
        "网页": ("playwright", "browser_navigate"),
        "网站": ("playwright", "browser_navigate"),
        "打开浏览器": ("playwright", "browser_navigate"),
        "打开网站": ("playwright", "browser_navigate"),
        "点击": ("playwright", "browser_click"),
        "悬停": ("playwright", "browser_hover"),
        "截图": ("playwright", "browser_take_screenshot"),
        "爬取": ("playwright", "browser_navigate"),
        "爬虫": ("playwright", "browser_navigate"),
        "自动化": ("playwright", "browser_navigate"),
    }
    
    # 测试查询
    test_queries = [
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
        "打开网站并截图",
        "自动化浏览器测试",
    ]
    
    print("测试关键词匹配:")
    print("=" * 60)
    
    for query in test_queries:
        query_lower = query.lower()
        print(f"\n查询: {query}")
        
        matched = False
        for keyword, (server, tool) in keyword_patterns.items():
            if keyword in query_lower:
                print(f"  匹配关键词: '{keyword}' -> {server}/{tool}")
                matched = True
        
        if not matched:
            print("  没有匹配的关键词")
        
        print("-" * 40)

if __name__ == "__main__":
    test_keyword_matching()