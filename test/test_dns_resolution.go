package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	fmt.Println("测试DNS解析和网络连接")
	fmt.Println("======================")

	// 测试1: DNS解析
	fmt.Println("\n1. 测试api.context7.com DNS解析:")
	start := time.Now()
	addrs, err := net.LookupHost("api.context7.com")
	elapsed := time.Since(start)
	
	if err != nil {
		fmt.Printf("   ❌ DNS解析失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ DNS解析成功 (耗时: %v)\n", elapsed)
		for i, addr := range addrs {
			fmt.Printf("      IP地址 %d: %s\n", i+1, addr)
		}
	}

	// 测试2: TCP连接测试
	fmt.Println("\n2. 测试TCP连接到api.context7.com:443:")
	start = time.Now()
	conn, err := net.DialTimeout("tcp", "api.context7.com:443", 5*time.Second)
	elapsed = time.Since(start)
	
	if err != nil {
		fmt.Printf("   ❌ TCP连接失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ TCP连接成功 (耗时: %v)\n", elapsed)
		conn.Close()
		
		// 获取连接详细信息
		addr := conn.RemoteAddr()
		fmt.Printf("      远程地址: %s\n", addr.String())
	}

	// 测试3: 检查hosts文件条目
	fmt.Println("\n3. 检查系统解析:")
	ips, err := net.LookupIP("api.context7.com")
	if err != nil {
		fmt.Printf("   ❌ IP查找失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ 找到 %d 个IP地址:\n", len(ips))
		for i, ip := range ips {
			fmt.Printf("      IP %d: %s\n", i+1, ip.String())
		}
	}

	fmt.Println("\n======================")
	fmt.Println("总结:")
	if len(addrs) > 0 {
		fmt.Println("✅ api.context7.com 可以解析")
		fmt.Println("✅ 网络连接应该正常")
		fmt.Println("✅ Context7 MCP服务器应该能连接")
	} else {
		fmt.Println("❌ api.context7.com 无法解析")
		fmt.Println("❌ 需要检查网络/DNS配置")
	}
}