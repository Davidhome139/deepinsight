# Context7服务状态测试

Write-Host "========================================"
Write-Host "Context7服务状态测试"
Write-Host "========================================"

# 测试1: 测试IP地址连接
Write-Host "`n1. 测试IP地址连接 (104.26.4.148:443):"
try {
    $tcp = New-Object System.Net.Sockets.TcpClient
    $result = $tcp.BeginConnect("104.26.4.148", 443, $null, $null)
    $success = $result.AsyncWaitHandle.WaitOne(5000, $false)
    
    if ($success) {
        $tcp.EndConnect($result)
        Write-Host "   ✅ 可以连接到 104.26.4.148:443"
        $tcp.Close()
        
        # 获取SSL证书信息
        try {
            $tcp = New-Object System.Net.Sockets.TcpClient("104.26.4.148", 443)
            $ssl = New-Object System.Net.Security.SslStream($tcp.GetStream(), $false)
            $ssl.AuthenticateAsClient("api.context7.com")
            Write-Host "   ✅ SSL证书验证成功"
            Write-Host "      证书主题: $($ssl.RemoteCertificate.Subject)"
            Write-Host "      证书有效期: $($ssl.RemoteCertificate.GetExpirationDateString())"
            $ssl.Close()
            $tcp.Close()
        } catch {
            Write-Host "   ⚠️ SSL证书验证失败: $_"
        }
    } else {
        Write-Host "   ❌ 连接超时到 104.26.4.148:443"
        $tcp.Close()
    }
} catch {
    Write-Host "   ❌ 连接测试失败: $_"
}

# 测试2: 测试域名连接（使用hosts映射）
Write-Host "`n2. 测试域名连接 (api.context7.com:443):"
try {
    $tcp = New-Object System.Net.Sockets.TcpClient
    $result = $tcp.BeginConnect("api.context7.com", 443, $null, $null)
    $success = $result.AsyncWaitHandle.WaitOne(5000, $false)
    
    if ($success) {
        $tcp.EndConnect($result)
        Write-Host "   ✅ 可以连接到 api.context7.com:443"
        $tcp.Close()
    } else {
        Write-Host "   ❌ 连接超时到 api.context7.com:443"
        $tcp.Close()
    }
} catch {
    Write-Host "   ❌ 连接测试失败: $_"
}

# 测试3: 测试HTTP请求
Write-Host "`n3. 测试HTTP请求到Context7 API:"
try {
    # 创建HTTP请求
    $request = [System.Net.HttpWebRequest]::Create("https://api.context7.com/")
    $request.Method = "GET"
    $request.Timeout = 10000  # 10秒超时
    $request.UserAgent = "Context7-Test/1.0"
    
    # 添加API密钥头（如果测试需要）
    # $request.Headers.Add("Authorization", "Bearer ctx7sk-50c0a67f-bee4-4a40-8028-82fd726ab7e2")
    
    $response = $request.GetResponse()
    Write-Host "   ✅ HTTP请求成功"
    Write-Host "      状态码: $($response.StatusCode)"
    Write-Host "      状态描述: $($response.StatusDescription)"
    Write-Host "      内容类型: $($response.ContentType)"
    $response.Close()
} catch [System.Net.WebException] {
    Write-Host "   ⚠️ HTTP请求失败: $($_.Exception.Message)"
    if ($_.Exception.Response) {
        Write-Host "      状态码: $($_.Exception.Response.StatusCode)"
        Write-Host "      状态描述: $($_.Exception.Response.StatusDescription)"
    }
} catch {
    Write-Host "   ❌ HTTP请求异常: $_"
}

# 测试4: 测试Context7网站
Write-Host "`n4. 测试Context7主网站 (context7.com):"
try {
    $request = [System.Net.HttpWebRequest]::Create("https://context7.com/")
    $request.Method = "GET"
    $request.Timeout = 5000
    
    $response = $request.GetResponse()
    Write-Host "   ✅ Context7网站可访问"
    Write-Host "      状态码: $($response.StatusCode)"
    $response.Close()
} catch {
    Write-Host "   ❌ Context7网站不可访问: $_"
}

# 测试5: 检查Cloudflare状态
Write-Host "`n5. 检查Cloudflare状态:"
$cloudflareIPs = @(
    @{IP="104.26.4.148"; Name="Primary"},
    @{IP="104.26.5.148"; Name="Secondary"},
    @{IP="172.67.72.218"; Name="Tertiary"}
)

foreach ($cf in $cloudflareIPs) {
    try {
        $tcp = New-Object System.Net.Sockets.TcpClient
        $result = $tcp.BeginConnect($cf.IP, 443, $null, $null)
        $success = $result.AsyncWaitHandle.WaitOne(2000, $false)
        
        if ($success) {
            $tcp.EndConnect($result)
            Write-Host "   ✅ $($cf.Name) Cloudflare节点 ($($cf.IP)): 可连接"
            $tcp.Close()
        } else {
            Write-Host "   ❌ $($cf.Name) Cloudflare节点 ($($cf.IP)): 连接超时"
            $tcp.Close()
        }
    } catch {
        Write-Host "   ❌ $($cf.Name) Cloudflare节点 ($($cf.IP)): 连接失败"
    }
}

Write-Host "`n========================================"
Write-Host "测试结果分析"
Write-Host "========================================"

Write-Host "`n如果测试1和2成功，但Context7仍然失败:"
Write-Host "1. Context7 API服务可能有问题"
Write-Host "2. API密钥可能无效"
Write-Host "3. 需要特定端口或协议"

Write-Host "`n如果测试1和2失败:"
Write-Host "1. 网络连接有问题"
Write-Host "2. 防火墙阻止了连接"
Write-Host "3. Cloudflare节点有问题"

Write-Host "`n建议下一步:"
Write-Host "1. 检查Context7官方状态页面"
Write-Host "2. 验证API密钥有效性"
Write-Host "3. 测试其他Context7端点"
Write-Host "`n========================================"