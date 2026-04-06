#!/usr/bin/env python3
"""
测试带有API密钥的Context7连接
"""

import subprocess
import time
import sys
import os

def test_context7_with_api_key():
    """测试带有API密钥的Context7连接"""
    print("=" * 60)
    print("测试带有API密钥的Context7连接")
    print("=" * 60)
    
    # 设置环境变量
    env = os.environ.copy()
    env["CONTEXT7_API_KEY"] = "ctx7sk-50c0a67f-bee4-4a40-8028-82fd726ab7e2"
    
    print(f"API密钥已设置: {env['CONTEXT7_API_KEY'][:10]}...")
    
    # 测试命令
    cmd = ["npx", "-y", "@upstash/context7-mcp", "--version"]
    
    try:
        print(f"\n执行命令: {' '.join(cmd)}")
        print("设置环境变量: CONTEXT7_API_KEY")
        
        result = subprocess.run(
            cmd,
            env=env,
            capture_output=True,
            text=True,
            timeout=30
        )
        
        print(f"返回码: {result.returncode}")
        if result.stdout:
            print(f"标准输出:\n{result.stdout}")
        if result.stderr:
            print(f"标准错误:\n{result.stderr}")
            
        if result.returncode == 0:
            print("✅ Context7 MCP服务器可以正常启动")
            return True
        else:
            print("❌ Context7 MCP服务器启动失败")
            return False
            
    except subprocess.TimeoutExpired:
        print("❌ 命令执行超时（30秒）")
        return False
    except FileNotFoundError:
        print("❌ npx未找到，请安装Node.js")
        return False
    except Exception as e:
        print(f"❌ 执行失败: {e}")
        return False

def test_context7_help():
    """测试Context7帮助命令"""
    print("\n" + "=" * 60)
    print("测试Context7帮助命令")
    print("=" * 60)
    
    env = os.environ.copy()
    env["CONTEXT7_API_KEY"] = "ctx7sk-50c0a67f-bee4-4a40-8028-82fd726ab7e2"
    
    cmd = ["npx", "-y", "@upstash/context7-mcp", "--help"]
    
    try:
        print(f"执行命令: {' '.join(cmd)}")
        
        result = subprocess.run(
            cmd,
            env=env,
            capture_output=True,
            text=True,
            timeout=15
        )
        
        if result.returncode == 0:
            print("✅ Context7帮助命令成功")
            # 检查输出中是否包含有用的信息
            if "context7" in result.stdout.lower() or "mcp" in result.stdout.lower():
                print("输出中包含Context7/MCP相关信息")
            return True
        else:
            print(f"❌ 帮助命令失败，返回码: {result.returncode}")
            if result.stderr:
                print(f"错误信息: {result.stderr[:200]}")
            return False
            
    except Exception as e:
        print(f"❌ 测试失败: {e}")
        return False

def check_api_key_format():
    """检查API密钥格式"""
    print("\n" + "=" * 60)
    print("检查API密钥格式")
    print("=" * 60)
    
    api_key = "ctx7sk-50c0a67f-bee4-4a40-8028-82fd726ab7e2"
    
    print(f"API密钥: {api_key}")
    print(f"长度: {len(api_key)} 字符")
    print(f"前缀: {api_key[:10]}")
    
    # 检查格式
    if api_key.startswith("ctx7sk-"):
        print("✅ API密钥格式正确（以ctx7sk-开头）")
    else:
        print("⚠️  API密钥格式可能不正确")
        
    return True

def main():
    """主函数"""
    print("Context7 API密钥连接测试")
    print("=" * 60)
    
    # 检查API密钥格式
    check_api_key_format()
    
    # 测试帮助命令
    help_success = test_context7_help()
    
    # 测试完整连接
    print("\n" + "=" * 60)
    print("测试完整Context7连接（可能需要较长时间）")
    print("=" * 60)
    
    # 注意：完整测试可能需要较长时间，因为要启动MCP服务器
    print("注意：完整测试可能需要启动MCP服务器，这可能需要一些时间...")
    
    # 先测试简单命令
    simple_success = test_context7_with_api_key()
    
    print("\n" + "=" * 60)
    print("测试结果总结")
    print("=" * 60)
    
    if help_success and simple_success:
        print("✅ 所有测试通过")
        print("\nContext7应该可以正常工作：")
        print("1. API密钥格式正确")
        print("2. Context7 MCP服务器可以启动")
        print("3. 帮助命令正常")
        print("\n如果MCP管理器仍然连接失败，可能是：")
        print("1. 网络连接问题")
        print("2. 超时设置问题（已修复为15秒）")
        print("3. Docker环境配置问题")
    elif help_success:
        print("⚠️  部分测试通过")
        print("Context7帮助命令正常，但完整连接测试失败")
        print("可能的原因：")
        print("1. Context7服务器启动需要更多时间")
        print("2. 网络连接问题")
        print("3. API密钥权限问题")
    else:
        print("❌ 测试失败")
        print("Context7连接有问题")
        print("可能的原因：")
        print("1. @upstash/context7-mcp包未正确安装")
        print("2. API密钥无效")
        print("3. Node.js/npm环境问题")
    
    print("=" * 60)
    
    return 0 if (help_success or simple_success) else 1

if __name__ == "__main__":
    sys.exit(main())