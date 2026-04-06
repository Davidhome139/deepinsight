// 直接测试 Playwright MCP 服务器是否能启动浏览器
const { spawn } = require('child_process');
const net = require('net');

async function testPlaywrightMCPDirect() {
    console.log('直接测试 Playwright MCP 服务器浏览器启动...\n');
    
    // 启动 Playwright MCP 服务器
    const proc = spawn('npx', ['@playwright/mcp@latest'], {
        env: {
            ...process.env,
            PLAYWRIGHT_BROWSERS_PATH: '/ms-playwright',
            DISPLAY: ':99',
            NODE_OPTIONS: '--inspect=0.0.0.0:9229'
        },
        stdio: ['pipe', 'pipe', 'pipe']
    });
    
    let stdout = '';
    let stderr = '';
    
    proc.stdout.on('data', (data) => {
        const text = data.toString();
        stdout += text;
        console.log('STDOUT:', text);
        
        // 检查是否有错误信息
        if (text.includes('error') || text.includes('Error') || text.includes('failed')) {
            console.log('⚠️ 发现可能的错误:', text.substring(0, 200));
        }
        
        // 检查是否成功启动
        if (text.includes('Server listening')) {
            console.log('✅ Playwright MCP 服务器启动成功');
        }
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
        
        // 分析输出
        if (stdout.includes('Server listening')) {
            console.log('\n✅ Playwright MCP 服务器启动成功');
        } else {
            console.log('\n❌ Playwright MCP 服务器可能启动失败');
            
            // 检查常见错误
            if (stdout.includes('browser') && stdout.includes('not found')) {
                console.log('🔍 错误: 浏览器未找到');
            }
            if (stdout.includes('executable')) {
                console.log('🔍 错误: 可执行文件问题');
            }
            if (stdout.includes('permission')) {
                console.log('🔍 错误: 权限问题');
            }
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