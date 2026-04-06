#!/usr/bin/env python3
"""
测试用户查询："使用playwright查找百度最新的美伊战报。"
分析工具推荐和语义意图
"""

# 模拟改进后的关键词模式
keyword_patterns = {
    # Playwright 关键词
    "playwright": ("playwright", "browser_navigate", 1.0),
    "浏览器": ("playwright", "browser_navigate", 0.9),
    "查找": ("playwright", "browser_navigate", 0.7),
    "百度": ("playwright", "browser_navigate", 0.6),
    
    # 搜索关键词
    "搜索": ("brave-search", "search", 0.6),
    "查找": ("brave-search", "search", 0.6),  # 注意：这个与上面的冲突
    "查询": ("brave-search", "search", 0.6),
    
    # 内置搜索工具关键词
    "web_search": ("search", "web_search", 0.8),
}

def analyze_query(query):
    """分析查询并推荐工具"""
    query_lower = query.lower()
    print(f"查询: {query}")
    print(f"小写: {query_lower}")
    print()
    
    # 查找匹配的关键词
    matches = []
    for keyword, (server, tool, weight) in keyword_patterns.items():
        if keyword in query_lower:
            matches.append({
                'keyword': keyword,
                'server': server,
                'tool': tool,
                'weight': weight
            })
    
    print("匹配的关键词:")
    for match in matches:
        print(f"  - '{match['keyword']}' -> {match['server']}/{match['tool']} (权重: {match['weight']:.2f})")
    
    # 选择最佳匹配
    if matches:
        matches.sort(key=lambda x: (-x['weight'], -len(x['keyword'])))
        best = matches[0]
        print(f"\n最佳匹配: '{best['keyword']}' -> {best['server']}/{best['tool']} (权重: {best['weight']:.2f})")
        
        # 检查置信度是否足够高
        confidence = best['weight']
        if confidence >= 0.7:
            print(f"置信度: {confidence:.2f} >= 0.7 ✓ 工具推荐器会推荐这个工具")
        else:
            print(f"置信度: {confidence:.2f} < 0.7 ✗ 工具推荐器可能不会推荐（置信度太低）")
    else:
        print("没有匹配的关键词")
    
    return matches

# 测试用户查询
user_query = "使用playwright查找百度最新的美伊战报。"
print("测试用户查询分析")
print("=" * 80)
matches = analyze_query(user_query)

# 分析语义意图
print("\n\n语义意图分析:")
print("=" * 80)

# 模拟语义意图检测的思考过程
print("语义意图检测可能会考虑:")
print("1. 用户提到了 'playwright' → 浏览器自动化工具")
print("2. 用户提到了 '查找' → 搜索意图")
print("3. 用户提到了 '百度' → 特定的网站")
print("4. 用户提到了 '最新的美伊战报' → 实时信息需求")
print()
print("可能的冲突:")
print("- 如果强调 'playwright' → 应该推荐 playwright/browser_navigate")
print("- 如果强调 '查找' → 可能推荐 search/web_search")
print("- 如果强调 '百度' → 可能需要浏览器导航到百度")

# 检查查询中的关键词权重
print("\n查询中的关键词权重分析:")
query_keywords = ["playwright", "查找", "百度", "最新", "美伊战报"]
for kw in query_keywords:
    if kw in user_query:
        # 查找这个关键词的权重
        weight = 0
        for pattern in keyword_patterns:
            if kw in pattern or pattern in kw:
                weight = max(weight, keyword_patterns[pattern][2])
        print(f"  '{kw}': 权重 ≈ {weight:.2f}")

# 模拟工具推荐器的决策
print("\n\n模拟工具推荐器决策:")
print("=" * 80)

# 根据改进后的算法，playwright 关键词权重最高（1.0）
if matches:
    print("工具推荐器会看到:")
    print("1. 'playwright' 匹配 → playwright/browser_navigate (权重: 1.0)")
    print("2. '查找' 匹配 → brave-search/search (权重: 0.6) 或 playwright/browser_navigate (权重: 0.7)")
    print()
    print("决策: 选择权重最高的匹配")
    print("结果: 推荐 playwright/browser_navigate (权重: 1.0)")
    print("置信度: 1.0 >= 0.7 ✓ 会推荐这个工具")
else:
    print("没有匹配的关键词，工具推荐器返回 None/None")

# 检查为什么语义意图检测可能选择 search/web_search
print("\n\n为什么语义意图检测可能选择 search/web_search?")
print("=" * 80)
print("可能的原因:")
print("1. 语义意图检测的 LLM 可能误解了用户意图")
print("2. LLM 可能认为 '查找' 比 'playwright' 更重要")
print("3. 内置工具列表中有 search/web_search，但没有 playwright/browser_navigate")
print("4. LLM 的训练数据可能偏向于将 '查找' 解释为搜索")

# 检查 detectMCPIntentTraditional 中的内置工具
print("\n\ndetectMCPIntentTraditional 中的内置工具列表:")
builtin_tools = [
    "search/web_search",
    "filesystem-local/list_directory", 
    "filesystem-local/read_file",
    "terminal/execute_command",
]
for tool in builtin_tools:
    print(f"  - {tool}")
print("注意: playwright/browser_navigate 不在内置工具列表中，它是外部 MCP 工具")

# 建议的解决方案
print("\n\n建议的解决方案:")
print("=" * 80)
print("1. 确保工具推荐器的置信度阈值适当（当前是 0.7）")
print("2. 在语义意图检测中，优先考虑工具推荐器的建议")
print("3. 为 'playwright' 添加更高的权重或特殊处理")
print("4. 确保语义意图检测的 LLM 理解 '使用playwright查找' 意味着浏览器自动化")

# 实际测试
print("\n\n实际测试改进后的算法:")
print("=" * 80)

# 使用改进后的关键词模式（来自之前的改进）
enhanced_patterns = {
    "playwright": ("playwright", "browser_navigate", 1.0),
    "使用playwright": ("playwright", "browser_navigate", 1.0),
    "浏览器": ("playwright", "browser_navigate", 0.9),
    "查找": ("playwright", "browser_navigate", 0.7),
    "百度": ("playwright", "browser_navigate", 0.6),
    "搜索": ("brave-search", "search", 0.6),
    "web_search": ("search", "web_search", 0.8),
}

def recommend_with_enhanced_patterns(query):
    query_lower = query.lower()
    matches = []
    
    for keyword, (server, tool, weight) in enhanced_patterns.items():
        if keyword in query_lower:
            matches.append({
                'keyword': keyword,
                'server': server,
                'tool': tool,
                'weight': weight
            })
    
    if matches:
        matches.sort(key=lambda x: (-x['weight'], -len(x['keyword'])))
        return matches[0]
    return None

result = recommend_with_enhanced_patterns(user_query)
if result:
    print(f"改进后的算法推荐: {result['server']}/{result['tool']}")
    print(f"匹配关键词: '{result['keyword']}'")
    print(f"权重: {result['weight']:.2f}")
    print(f"置信度: {'足够高 (>= 0.7)' if result['weight'] >= 0.7 else '太低 (< 0.7)'}")
else:
    print("改进后的算法没有找到匹配")