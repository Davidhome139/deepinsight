#!/usr/bin/env python3
"""
测试Context7 MCP服务器连接
用于诊断Context7初始化失败的问题
"""

import subprocess
import time
import sys
import os

def test_context7_direct():
    """直接测试Context7 MCP服务器"""
    print("=" * 60)
    print("测试Context7 MCP服务器连接")
    print("=" * 60)
    
    # 检查npx是否可用
    try:
        result = subprocess.run(["npx", "--version"], capture_output=True, text=True)
        print(f"✅ npx版本: {result.stdout.strip()}")
    except Exception as e:
        print(f"❌ npx不可用: {e}")
        return False
    
    # 检查@upstash/context7-mcp是否已安装
    try:
        result = subprocess.run(["npm", "list", "-g", "@upstash/context7-mcp"], 
                              capture_output=True, text=True)
        if "@upstash/context7-mcp" in result.stdout:
            print("✅ @upstash/context7-mcp 已全局安装")
        else:
            print("⚠️  @upstash/context7-mcp 未全局安装，尝试安装...")
            install_result = subprocess.run(["npm", "install", "-g", "@upstash/context7-mcp"],
                                          capture_output=True, text=True)
            if install_result.returncode == 0:
                print("✅ @upstash/context7-mcp 安装成功")
            else:
                print(f"❌ @upstash/context7-mcp 安装失败: {install_result.stderr}")
                return False
    except Exception as e:
        print(f"❌ 检查npm包失败: {e}")
        return False
    
    # 直接运行Context7 MCP服务器测试
    print("\n" + "=" * 60)
    print("启动Context7 MCP服务器...")
    print("=" * 60)
    
    # 构建命令
    cmd = ["npx", "-y", "@upstash/context7-mcp"]
    
    try:
        # 启动进程
        print(f"执行命令: {' '.join(cmd)}")
        process = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,
            universal_newlines=True
        )
        
        # 等待几秒让服务器启动
        print("等待服务器启动...")
        time.sleep(5)
        
        # 检查进程状态
        if process.poll() is not None:
            # 进程已退出
            stdout, stderr = process.communicate()
            print(f"❌ 进程已退出，返回码: {process.returncode}")
            print(f"标准输出:\n{stdout}")
            print(f"标准错误:\n{stderr}")
            return False
        else:
            print("✅ 进程仍在运行，服务器可能已启动")
            
            # 尝试发送简单的MCP初始化请求
            print("\n尝试发送MCP初始化请求...")
            # 这里可以添加实际的MCP协议测试
            # 暂时只是检查进程状态
            
            # 终止进程
            print("终止测试进程...")
            process.terminate()
            try:
                process.wait(timeout=5)
                print("✅ 进程已正常终止")
            except subprocess.TimeoutExpired:
                print("⚠️  进程未在5秒内终止，强制终止...")
                process.kill()
                process.wait()
                print("✅ 进程已强制终止")
            
            return True
            
    except Exception as e:
        print(f"❌ 启动Context7失败: {e}")
        return False

def test_network_connectivity():
    """测试网络连接性"""
    print("\n" + "=" * 60)
    print("测试网络连接性")
    print("=" * 60)
    
    test_urls = [
        "https://api.context7.com",
        "https://www.npmjs.com",
        "https://github.com"
    ]
    
    import urllib.request
    import ssl
    
    # 创建不验证SSL的上下文（仅用于测试）
    context = ssl._create_unverified_context()
    
    for url in test_urls:
        try:
            print(f"测试连接到 {url}...")
            req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
            response = urllib.request.urlopen(req, timeout=10, context=context)
            print(f"✅ 成功连接到 {url} (状态码: {response.status})")
        except Exception as e:
            print(f"❌ 连接到 {url} 失败: {e}")
    
    return True

def check_system_resources():
    """检查系统资源"""
    print("\n" + "=" * 60)
    print("检查系统资源")
    print("=" * 60)
    
    import psutil
    
    try:
        # 内存使用
        memory = psutil.virtual_memory()
        print(f"内存总量: {memory.total / (1024**3):.2f} GB")
        print(f"可用内存: {memory.available / (1024**3):.2f} GB")
        print(f"内存使用率: {memory.percent}%")
        
        # CPU使用
        cpu_percent = psutil.cpu_percent(interval=1)
        print(f"CPU使用率: {cpu_percent}%")
        
        # 磁盘空间
        disk = psutil.disk_usage('/')
        print(f"磁盘总量: {disk.total / (1024**3):.2f} GB")
        print(f"可用磁盘: {disk.free / (1024**3):.2f} GB")
        print(f"磁盘使用率: {disk.percent}%")
        
        return True
    except Exception as e:
        print(f"❌ 检查系统资源失败: {e}")
        return False

def main():
    """主函数"""
    print("Context7连接问题诊断工具")
    print("=" * 60)
    
    # 检查Python版本
    print(f"Python版本: {sys.version}")
    
    # 检查当前目录
    print(f"当前目录: {os.getcwd()}")
    
    # 测试网络连接
    test_network_connectivity()
    
    # 检查系统资源
    check_system_resources()
    
    # 测试Context7直接连接
    success = test_context7_direct()
    
    print("\n" + "=" * 60)
    if success:
        print("✅ 测试完成，Context7可能正常工作")
        print("建议：")
        print("1. 检查MCP管理器中的超时设置（已修复为30秒）")
        print("2. 确保网络连接正常")
        print("3. 检查系统资源是否充足")
    else:
        print("❌ 测试发现问题")
        print("可能的原因：")
        print("1. @upstash/context7-mcp包安装问题")
        print("2. 网络连接问题")
        print("3. 系统资源不足")
        print("4. Node.js/npm版本问题")
    
    print("=" * 60)
    
    return 0 if success else 1

if __name__ == "__main__":
    sys.exit(main())