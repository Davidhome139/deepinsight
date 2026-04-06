# Context7网络连接测试脚本

Write-Host "=" * 60
Write-Host "Context7网络连接测试"
Write-Host "=" * 60

# 测试1: DNS解析
Write-Host "`n[测试1] DNS解析测试"
Write-Host "-" * 40

try {
    $result = nslookup api.context7.com 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ nslookup可以解析 api.context7.com"
        Write-Host "解析结果:"
        $result | Select-Object -Last 10
    } else {
        Write-Host "❌ nslookup无法解析 api.context7.com"
        Write-Host "错误信息: $result"
    }
} catch {
    Write-Host "❌ nslookup测试失败: $_"
}

# 测试2: PowerShell DNS解析
Write-Host "`n[测试2] PowerShell DNS解析"
Write-Host "-" * 40

try {
    $ips = [System.Net.Dns]::GetHostAddresses("api.context7.com")
    if ($ips.Count -gt 0) {
        Write-Host "✅ .NET DNS解析成功"
        foreach ($ip in $ips) {
            Write-Host "  IP地址: $($ip.IPAddressToString)"
        }
    } else {
        Write-Host "❌ .NET DNS解析失败，无IP地址返回"
    }
} catch [System.Net.Sockets.SocketException] {
    Write-Host "❌ .NET DNS解析失败: 主机名不存在"
} catch {
    Write-Host "❌ .NET DNS解析失败: $_"
}

# 测试3: 测试连接（不发送实际请求）
Write-Host "`n[测试3] 网络连接测试"
Write-Host "-" * 40

try {
    # 创建TCP客户端测试连接
    $tcpClient = New-Object System.Net.Sockets.TcpClient
    $asyncResult = $tcpClient.BeginConnect("api.context7.com", 443, $null, $null)
    
    # 等待2秒
    $success = $asyncResult.AsyncWaitHandle.WaitOne(2000, $false)
    
    if ($success) {
        $tcpClient.EndConnect($asyncResult)
        Write-Host "✅ 可以连接到 api.context7.com:443"
        $tcpClient.Close()
    } else {
        Write-Host "❌ 连接超时到 api.context7.com:443"
        $tcpClient.Close()
    }
} catch {
    Write-Host "❌ 连接测试失败: $_"
}

# 测试4: 检查其他相关域名
Write-Host "`n[测试4] 检查其他Context7相关域名"
Write-Host "-" * 40

$domains = @(
    "context7.com",
    "www.context7.com",
    "clerk.context7.com",
    "docs.context7.com"
)

foreach ($domain in $domains) {
    try {
        $ips = [System.Net.Dns]::GetHostAddresses($domain) 2>$null
        if ($ips -and $ips.Count -gt 0) {
            Write-Host "✅ $domain 可解析"
        } else {
            Write-Host "❌ $domain 无法解析"
        }
    } catch {
        Write-Host "❌ $domain 解析失败: $_"
    }
}

# 测试5: 检查Cloudflare IP（从nslookup结果）
Write-Host "`n[测试5] 检查Cloudflare IP连接"
Write-Host "-" * 40

# 从之前的nslookup结果中获取IP
$cloudflareIps = @(
    "104.26.4.148",
    "104.26.5.148",
    "172.67.72.218"
)

foreach ($ip in $cloudflareIps) {
    try {
        $tcpClient = New-Object System.Net.Sockets.TcpClient
        $asyncResult = $tcpClient.BeginConnect($ip, 443, $null, $null)
        $success = $asyncResult.AsyncWaitHandle.WaitOne(2000, $false)
        
        if ($success) {
            $tcpClient.EndConnect($asyncResult)
            Write-Host "✅ 可以连接到 Cloudflare IP: $ip:443"
            $tcpClient.Close()
        } else {
            Write-Host "❌ 连接超时到 Cloudflare IP: $ip:443"
            $tcpClient.Close()
        }
    } catch {
        Write-Host "❌ 连接测试失败到 $ip: $_"
    }
}

# 总结
Write-Host "`n" + "=" * 60
Write-Host "测试结果总结"
Write-Host "=" * 60

Write-Host "`n问题分析:"
Write-Host "1. nslookup可以解析 api.context7.com，但其他工具不行"
Write-Host "2. 这可能是因为:"
Write-Host "   - DNS缓存问题"
Write-Host "   - 防火墙/代理设置"
Write-Host "   - 网络策略限制"
Write-Host "   - 域名配置问题"

Write-Host "`n建议解决方案:"
Write-Host "1. 清除DNS缓存: ipconfig /flushdns"
Write-Host "2. 检查防火墙设置"
Write-Host "3. 验证网络代理配置"
Write-Host "4. 尝试使用IP地址直接连接"
Write-Host "5. 检查hosts文件是否有错误条目"

Write-Host "`n对于Context7 MCP服务器:"
Write-Host "1. 如果api.context7.com无法访问，Context7将无法工作"
Write-Host "2. 需要解决网络连接问题"
Write-Host "3. 可以尝试使用代理或VPN"

Write-Host "=" * 60