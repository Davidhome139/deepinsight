// 直接测试 Playwright MCP 服务器是否能启动浏览器
const { spawn } = require('child_process');
const path = require('path');

async function testPlaywrightMCPDirect() {
    console.log('直接测试 Playwright MCP 服务器浏览器启动...\n');
    
    // 启动 Playwright MCP 服务器（与容器中相同的配置）
    const playwrightMCPPath = path.join('/usr/local/lib/node_modules/@playwright/mcp/bin/playwright-mcp.js');
    
    const args = [
        '--browser', 'chromium'
    ];
    
    console.log('启动命令: node', playwrightMCPPath, args.join(' '));
    
    const env = {
        ...process.env,
        PLAYWRIGHT_BROWSER: 'chromium',
        PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH: '/usr/bin/chromium-browser',
        DISPLAY: ':99',
        PLAYWRIGHT_BROWSERS_PATH: '/ms-playwright'
    };
    
    const proc = spawn('node', [playwrightMCPPath, ...args], {
        env,
        stdio: ['pipe', 'pipe', 'pipe']
    });
    
    let stdout = '';
    let stderr = '';
    
    proc.stdout.on('data', (data) => {
        stdout += data.toString();
        console.log('STDOUT:', data.toString());
    });
    
    proc.stderr.on('data', (data) => {
        stderr += data.toString();
        console.log('STDERR:', data.toString());
    });
    
    proc.on('close', (code) => {
        console.log(`\n进程退出，代码: ${code}`);
        console.log('STDOUT 总长度:', stdout.length);
        console.log('STDERR 总长度:', stderr.length);
        
        if (stderr.includes('error') || stderr.includes('Error')) {
            console.log('\n发现错误:');
            console.log(stderr.substring(0, 500));
        }
        
        if (stdout.includes('Server listening')) {
            console.log('\n✓ Playwright MCP 服务器启动成功');
        } else {
            console.log('\n✗ Playwright MCP 服务器可能启动失败');
        }
    });
    
    // 等待 5 秒后杀死进程
    setTimeout(() => {
        if (!proc.killed) {
            console.log('\n5 秒后终止进程...');
            proc.kill('SIGTERM');
        }
    }, 5000);
}

testPlaywrightMCPDirect().catch(console.error);