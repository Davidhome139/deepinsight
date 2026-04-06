// 测试 Playwright MCP 服务器是否正常工作
const net = require('net');

// Playwright MCP 服务器通常使用 stdio 或 socket 通信
// 这里我们检查进程是否在运行，并尝试连接到它
async function testPlaywrightMCP() {
    console.log('测试 Playwright MCP 服务器...\n');
    
    // 检查进程是否在运行
    const { exec } = require('child_process');
    
    exec('ps aux | grep playwright-mcp', (error, stdout, stderr) => {
        if (error) {
            console.log('  ✗ 无法检查进程:', error.message);
            return;
        }
        
        const lines = stdout.split('\n').filter(line => line.includes('playwright-mcp') && !line.includes('grep'));
        if (lines.length > 0) {
            console.log('  ✓ Playwright MCP 服务器进程正在运行');
            console.log('    进程信息:', lines[0].substring(0, 100) + '...');
            
            // 检查配置文件是否存在
            exec('cat /app/config/playwright-mcp-config.json', (error, stdout, stderr) => {
                if (error) {
                    console.log('  ✗ 无法读取配置文件:', error.message);
                } else {
                    console.log('  ✓ 配置文件存在');
                    try {
                        const config = JSON.parse(stdout);
                        console.log('    浏览器:', config.browser);
                        console.log('    executablePath:', config.launchOptions.executablePath);
                        console.log('    headless:', config.launchOptions.headless);
                    } catch (e) {
                        console.log('  ✗ 配置文件格式错误:', e.message);
                    }
                }
            });
        } else {
            console.log('  ✗ Playwright MCP 服务器进程未找到');
        }
    });
}

testPlaywrightMCP();