#!/usr/bin/env python3
"""
简单测试Context7 MCP服务器连接
"""

import subprocess
import time
import sys
import os

def test_npx_context7():
    """测试直接运行npx context7"""
    print("测试npx运行@upstash/context7-mcp...")
    
    try:
        # 直接运行npx命令，设置超时
        cmd = ["npx", "-y", "@upstash/context7-mcp", "--help"]
        
        print(f"执行命令: {' '.join(cmd)}")
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30  # 30秒超时
        )
        
        print(f"返回码: {result.returncode}")
        if result.stdout:
            print(f"标准输出 (前500字符):\n{result.stdout[:500]}")
        if result.stderr:
            print(f"标准错误 (前500字符):\n{result.stderr[:500]}")
            
        if result.returncode == 0:
            print("✅ npx可以成功运行@upstash/context7-mcp")
            return True
        else:
            print("❌ npx运行@upstash/context7-mcp失败")
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

def test_npm_installation():
    """测试npm包安装状态"""
    print("\n检查npm包安装状态...")
    
    try:
        # 检查是否已安装
        result = subprocess.run(
            ["npm", "list", "-g", "@upstash/context7-mcp"],
            capture_output=True,
            text=True
        )
        
        if "@upstash/context7-mcp" in result.stdout:
            print("✅ @upstash/context7-mcp 已全局安装")
            return True
        else:
            print("❌ @upstash/context7-mcp 未全局安装")
            
            # 尝试安装
            print("尝试安装@upstash/context7-mcp...")
            install_result = subprocess.run(
                ["npm", "install", "-g", "@upstash/context7-mcp"],
                capture_output=True,
                text=True,
                timeout=120  # 2分钟超时
            )
            
            if install_result.returncode == 0:
                print("✅ 安装成功")
                return True
            else:
                print(f"❌ 安装失败: {install_result.stderr[:500]}")
                return False
                
    except Exception as e:
        print(f"❌ 检查失败: {e}")
        return False

def main():
    """主函数"""
    print("Context7简单连接测试")
    print("=" * 60)
    
    # 检查Node.js和npm
    try:
        node_result = subprocess.run(["node", "--version"], capture_output=True, text=True)
        print(f"Node.js版本: {node_result.stdout.strip()}")
        
        npm_result = subprocess.run(["npm", "--version"], capture_output=True, text=True)
        print(f"npm版本: {npm_result.stdout.strip()}")
    except Exception as e:
        print(f"❌ Node.js/npm检查失败: {e}")
        return 1
    
    # 测试npm包安装
    if not test_npm_installation():
        print("\n⚠️ npm包安装有问题，可能影响Context7运行")
    
    # 测试直接运行
    print("\n" + "=" * 60)
    print("测试直接运行Context7 MCP服务器...")
    print("=" * 60)
    
    success = test_npx_context7()
    
    print("\n" + "=" * 60)
    if success:
        print("✅ 测试通过")
        print("Context7 MCP服务器应该可以正常工作")
        print("如果仍然失败，可能是：")
        print("1. 网络连接问题（Context7需要访问API）")
        print("2. 系统资源不足")
        print("3. 防火墙/代理设置")
    else:
        print("❌ 测试失败")
        print("可能的原因：")
        print("1. @upstash/context7-mcp包有问题")
        print("2. Node.js版本不兼容")
        print("3. 系统权限问题")
    
    print("=" * 60)
    
    return 0 if success else 1

if __name__ == "__main__":
    sys.exit(main())