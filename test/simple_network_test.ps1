# 简单网络测试脚本

Write-Host "Context7网络连接测试"
Write-Host "===================="

# 测试DNS解析
Write-Host "`n1. 测试DNS解析:"
try {
    $result = nslookup api.context7.com 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "   ✅ nslookup成功"
        $result | Select-Object -Last 6
    } else {
        Write-Host "   ❌ nslookup失败"
    }
} catch {
    Write-Host "   ❌ nslookup异常: $_"
}

# 测试.NET DNS
Write-Host "`n2. 测试.NET DNS解析:"
try {
    $ips = [System.Net.Dns]::GetHostAddresses("api.context7.com")
    if ($ips.Count -gt 0) {
        Write-Host "   ✅ .NET DNS成功"
        foreach ($ip in $ips) {
            Write-Host "      IP: $($ip.IPAddressToString)"
        }
    } else {
        Write-Host "   ❌ .NET DNS失败"
    }
} catch {
    Write-Host "   ❌ .NET DNS异常: $_"
}

# 测试TCP连接
Write-Host "`n3. 测试TCP连接 (端口443):"
try {
    $tcp = New-Object System.Net.Sockets.TcpClient
    $result = $tcp.BeginConnect("api.context7.com", 443, $null, $null)
    $success = $result.AsyncWaitHandle.WaitOne(3000, $false)
    
    if ($success) {
        $tcp.EndConnect($result)
        Write-Host "   ✅ TCP连接成功"
        $tcp.Close()
    } else {
        Write-Host "   ❌ TCP连接超时"
        $tcp.Close()
    }
} catch {
    Write-Host "   ❌ TCP连接异常: $_"
}

Write-Host "`n===================="
Write-Host "总结:"
Write-Host "如果DNS解析失败，Context7无法工作"
Write-Host "需要检查网络连接和DNS设置"