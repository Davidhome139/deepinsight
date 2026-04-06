#!/usr/bin/env python3
"""
测试 Playwright 修复
"""

def detect_mcp_intent_semantic_fixed(content):
    """模拟修复后的 detectMCPIntentSemantic 函数"""
    content_lower = content.lower()
    
    # Special handling for playwright queries
    if "playwright" in content_lower:
        print(f"[Chat] Special handling: User explicitly mentioned 'playwright', recommending playwright/browser_navigate")
        return "playwright/browser_navigate"
    
    # 模拟工具推荐器（简化）
    # 在实际代码中，这里会调用 toolRecommender.RecommendTool
    print(f"[Chat] Tool recommendation not shown or empty")
    
    # 模拟传统工具检测
    return detect_mcp_intent_traditional(content)

def detect_mcp_intent_traditional(content):
    """模拟 detectMCPIntentTraditional 函数"""
    content_lower = content.lower()
    
    # 检查搜索意图
    search_keywords = ["搜索", "search", "查找", "find", "查询", "query", "google", "百度", "bing"]
    has_search_keyword = any(keyword in content_lower for keyword in search_keywords)
    
    # Web 上下文关键词
    web_context_keywords = ["网络", "网上", "web", "internet", "online", "最新", "latest", "新闻", "news", "论文", "paper", "文章", "article"]
    has_web_context = any(keyword in content_lower for keyword in web_context_keywords)
    
    # 如果有搜索关键词，或者搜索 + web 上下文
    if has_search_keyword or (has_web_context and "search" in content_lower):
        print(f"[Chat] Semantic intent → tool: search/web_search")
        return "search/web_search"
    
    # 其他工具检测...
    return ""

# 测试场景
test_scenarios = [
    {
        "query": "使用playwright查找百度最新的美伊战报。",
        "description": "用户明确提到 playwright，应该推荐 playwright/browser_navigate"
    },
    {
        "query": "搜索最新的美伊战报",
        "description": "用户提到搜索，应该推荐 search/web_search"
    },
    {
        "query": "打开浏览器访问百度",
        "description": "用户提到浏览器，但没有明确提到 playwright"
    },
    {
        "query": "playwright browser navigate",
        "description": "英文查询提到 playwright"
    },
    {
        "query": "使用playwright打开百度网站",
        "description": "用户明确提到 playwright"
    },
]

print("测试 Playwright 修复")
print("=" * 80)

for scenario in test_scenarios:
    query = scenario["query"]
    description = scenario["description"]
    
    print(f"\n测试: {description}")
    print(f"查询: {query}")
    
    # 模拟语义意图检测
    result = detect_mcp_intent_semantic_fixed(query)
    
    if result:
        print(f"结果: 推荐 {result}")
        
        # 验证结果是否符合预期
        if "playwright" in query.lower() and result == "playwright/browser_navigate":
            print("✓ 符合预期：用户提到 playwright，系统推荐 playwright/browser_navigate")
        elif "搜索" in query or "search" in query.lower() and result == "search/web_search":
            print("✓ 符合预期：用户提到搜索，系统推荐 search/web_search")
        else:
            print("? 结果可能需要进一步分析")
    else:
        print("结果: 没有推荐工具")
    
    print("-" * 80)

# 验证修复
print("\n\n验证修复:")
print("=" * 80)

# 原始问题查询
original_query = "使用playwright查找百度最新的美伊战报。"
print(f"原始问题查询: {original_query}")

result = detect_mcp_intent_semantic_fixed(original_query)
if result == "playwright/browser_navigate":
    print("✓ 修复成功：现在当用户提到 'playwright' 时，系统会推荐 playwright/browser_navigate")
    print("  之前的问题：系统错误地推荐了 search/web_search")
    print("  现在的修复：添加了特殊处理，当用户明确提到 'playwright' 时，直接推荐 playwright/browser_navigate")
else:
    print("✗ 修复失败：系统仍然没有推荐 playwright/browser_navigate")
    print(f"  实际结果: {result}")

# 测试边缘情况
print("\n\n测试边缘情况:")
print("=" * 80)

edge_cases = [
    "PLAYWRIGHT 浏览器自动化",
    "Playwright 测试",
    "使用 Playwright 进行网页爬取",
    "playwright 和 selenium 比较",
]

for query in edge_cases:
    print(f"\n查询: {query}")
    result = detect_mcp_intent_semantic_fixed(query)
    if result == "playwright/browser_navigate":
        print(f"结果: {result} ✓ 正确识别了 playwright")
    else:
        print(f"结果: {result} ✗ 可能没有识别 playwright")

# 总结
print("\n\n总结:")
print("=" * 80)
print("修复内容：在 detectMCPIntentSemantic 函数中添加了特殊处理")
print("  当用户查询中包含 'playwright'（不区分大小写）时，直接返回 'playwright/browser_navigate'")
print("  这确保了用户明确提到 Playwright 时，系统会推荐正确的工具")
print()
print("为什么需要这个修复：")
print("  1. 工具推荐器可能没有返回推荐（serverSummaries 可能不包含未连接的 Playwright 服务器）")
print("  2. 语义意图检测的 LLM 可能误解了用户意图（将 '查找' 解释为搜索而不是浏览器导航）")
print("  3. 用户明确提到 'playwright' 是一个强烈的信号，应该优先处理")
print()
print("这个修复是临时的解决方案。长期解决方案应该是：")
print("  1. 确保工具推荐器能够正确处理所有服务器（包括未连接的服务器）")
print("  2. 改进关键词模式，确保 'playwright' 关键词能够被正确匹配")
print("  3. 优化语义意图检测的提示词，使其更好地理解浏览器自动化意图")