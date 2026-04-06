# 简单Context7服务测试

Write-Host "Testing Context7 Service Status"
Write-Host "================================"

# 测试1: TCP连接测试
Write-Host "`n1. Testing TCP connection to 104.26.4.148:443"
try {
    $tcp = New-Object System.Net.Sockets.TcpClient
    $result = $tcp.BeginConnect("104.26.4.148", 443, $null, $null)
    $success = $result.AsyncWaitHandle.WaitOne(5000, $false)
    
    if ($success) {
        $tcp.EndConnect($result)
        Write-Host "   SUCCESS: Can connect to 104.26.4.148:443"
        $tcp.Close()
    } else {
        Write-Host "   FAILED: Connection timeout"
        $tcp.Close()
    }
} catch {
    Write-Host "   ERROR: $_"
}

# 测试2: 测试域名
Write-Host "`n2. Testing api.context7.com:443"
try {
    $tcp = New-Object System.Net.Sockets.TcpClient
    $result = $tcp.BeginConnect("api.context7.com", 443, $null, $null)
    $success = $result.AsyncWaitHandle.WaitOne(5000, $false)
    
    if ($success) {
        $tcp.EndConnect($result)
        Write-Host "   SUCCESS: Can connect to api.context7.com:443"
        $tcp.Close()
    } else {
        Write-Host "   FAILED: Connection timeout"
        $tcp.Close()
    }
} catch {
    Write-Host "   ERROR: $_"
}

Write-Host "`n================================"
Write-Host "Summary:"
Write-Host "- If both tests fail: Network issue"
Write-Host "- If test1 passes but test2 fails: DNS/hosts issue"
Write-Host "- If both pass: Context7 service should be reachable"