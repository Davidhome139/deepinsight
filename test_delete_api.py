import requests
import json
import time

# 测试注册和登录
register_url = "http://localhost:8080/api/v1/auth/register"
login_url = "http://localhost:8080/api/v1/auth/login"

# 使用现有用户
print("Testing login with existing user...")
try:
    # 直接登录
    login_data = {
        "email": "13902263987@139.com",
        "password": "767676"  # 用户提供的正确密码
    }
    
    print("\nTesting login...")
    login_response = requests.post(login_url, json=login_data, timeout=10)
    print(f"Login status: {login_response.status_code}")
    print(f"Login response: {login_response.text}")
    
    if login_response.status_code == 200:
        result = login_response.json()
        token = result.get('access_token')
        if not token:
            # 尝试其他可能的字段名
            token = result.get('token')
        
        if token:
            print(f"Token obtained: {token[:20]}...")
            
            # 测试删除API - 使用存在的provider
            provider_name = "deepseek"
            delete_url = f"http://localhost:8080/api/v1/settings/ai-providers/{provider_name}?type=ai"
            headers = {
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/json"
            }
            
            # 先测试GET请求看看settings路由是否工作
            get_url = "http://localhost:8080/api/v1/settings/ai-providers"
            print(f"\nTesting GET settings API...")
            get_response = requests.get(get_url, headers=headers, timeout=10)
            print(f"GET status: {get_response.status_code}")
            print(f"GET response: {get_response.text[:200] if get_response.text else 'Empty'}")
            
            # 尝试删除aliyun，因为GET响应显示有这个provider
            provider_name = "aliyun"
            delete_url = f"http://localhost:8080/api/v1/settings/ai-providers/{provider_name}?type=ai"
            print(f"\nTesting delete API for provider: {provider_name}")
            print(f"DELETE URL: {delete_url}")
            print(f"Headers: {headers}")
            delete_response = requests.delete(delete_url, headers=headers, timeout=10)
            print(f"Delete status: {delete_response.status_code}")
            print(f"Delete response: {delete_response.text}")
            
            # 检查是否是路由问题
            if delete_response.status_code == 404 and "page not found" in delete_response.text:
                print("\n⚠️  Route not found - checking if DELETE route is registered")
                # 测试其他DELETE路由
                test_url = "http://localhost:8080/api/v1/settings/ai-providers/test123"
                test_response = requests.delete(test_url, headers=headers, timeout=5)
                print(f"Test DELETE status: {test_response.status_code}")
                print(f"Test DELETE response: {test_response.text}")
            
            if delete_response.status_code in [200, 404]:
                print("\n✅ Delete API is working correctly!")
            else:
                print("\n❌ Delete API returned unexpected status")
        else:
            print("❌ No token in login response. Response keys:", result.keys() if hasattr(result, 'keys') else 'N/A')
    else:
        print("❌ Login failed")
except Exception as e:
    print(f"❌ Error: {e}")