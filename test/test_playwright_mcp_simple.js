// 简单测试 Playwright MCP 服务器
const { spawn } = require('child_process');
const readline = require('readline');

async function testPlaywrightMCP() {
    console.log('测试 Playwright MCP 服务器...\n');
    
    // 启动 Playwright MCP 服务器（与容器中相同的配置）
    const proc = spawn('npx', ['@playwright/mcp@latest'], {
        env: {
            ...process.env,
            PLAYWRIGHT_BROWSERS_PATH: '/ms-playwright',
            PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH: '/usr/bin/chromium-browser',
            DISPLAY: ':99',
            PLAYWRIGHT_BROWSER: 'chromium'
        },
        stdio: ['pipe', 'pipe', 'pipe']
    });
    
    let stdout = '';
    let stderr = '';
    
    proc.stdout.on('data', (data) => {
        const text = data.toString();
        stdout += text;
        console.log('STDOUT:', text);
    });
    
    proc.stderr.on('data', (data) => {
        const text = data.toString();
        stderr += text;
        console.log('STDERR:', text);
    });
    
    proc.on('close', (code) => {
        console.log(`\n进程退出，代码: ${code}`);
        console.log('STDOUT 总长度:', stdout.length);
        console.log('STDERR 总长度:', stderr.length);
        
        if (stderr) {
            console.log('\n错误输出:');
            console.log(stderr.substring(0, 1000));
        }
        
        if (stdout.includes('Server listening')) {
            console.log('\n✓ Playwright MCP 服务器启动成功');
        } else {
            console.log('\n✗ Playwright MCP 服务器可能启动失败');
        }
    });
    
    // 等待 3 秒后杀死进程
    setTimeout(() => {
        if (!proc.killed) {
            console.log('\n3 秒后终止进程...');
            proc.kill('SIGTERM');
        }
    }, 3000);
}

testPlaywrightMCP().catch(console.error);