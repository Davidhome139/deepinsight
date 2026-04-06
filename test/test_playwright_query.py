#!/usr/bin/env python3
"""
测试 Playwright 工具推荐
"""

import requests
import json
import sys

def test_tool_recommendation(query):
    """测试工具推荐系统"""
    url = "http://localhost:8080/api/v1/chat/stream"
    
    headers = {
        "Content-Type": "application/json",
        "Authorization": "Bearer test_token"  # 简化测试，实际需要有效token
    }
    
    payload = {
        "conversation_id": 1,
        "content": query,
        "model": "deepseek-chat",
        "web_search": False,
        "search_provider": "brave",
        "mcp_tool": "",  # 留空，让系统自动推荐
        "system_prompt": "",
        "promptEngineeringConfig": None
    }
    
    try:
        response = requests.post(url, json=payload, headers=headers, stream=True)
        
        if response.status_code != 200:
            print(f"错误: HTTP {response.status_code}")
            print(f"响应: {response.text}")
            return
        
        print(f"查询: {query}")
        print("响应流:")
        
        for line in response.iter_lines():
            if line:
                line_str = line.decode('utf-8')
                if line_str.startswith('data: '):
                    data_str = line_str[6:]  # 移除 'data: ' 前缀
                    if data_str:
                        try:
                            data = json.loads(data_str)
                            if 'content' in data:
                                print(data['content'], end='', flush=True)
                        except json.JSONDecodeError:
                            print(f"无法解析JSON: {data_str}")
        
        print("\n" + "="*50)
        
    except requests.exceptions.RequestException as e:
        print(f"请求错误: {e}")

if __name__ == "__main__":
    # 测试不同的查询
    test_queries = [
        "打开浏览器访问百度网站",  # 中文，包含"浏览器"关键词
        "navigate to google.com",  # 英文，包含"navigate"关键词
        "take a screenshot of the page",  # 英文，包含"screenshot"关键词
        "点击页面上的按钮",  # 中文，包含"点击"关键词
        "获取网页内容",  # 中文，包含"网页"关键词
        "how to use react hooks",  # 英文，应该推荐context7
        "查询文档",  # 中文，应该推荐context7
    ]
    
    for query in test_queries:
        test_tool_recommendation(query)
        input("按Enter继续下一个测试...")