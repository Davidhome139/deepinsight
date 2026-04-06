// 直接测试 Playwright 是否可以使用系统 Chromium
const path = require('path');
// 从 Playwright MCP 服务器的 node_modules 加载 Playwright
const playwrightPath = path.join('/usr/local/lib/node_modules/@playwright/mcp/node_modules/playwright');
const playwright = require(playwrightPath);

async function testPlaywrightDirect() {
    console.log('直接测试 Playwright 使用系统 Chromium...\n');
    
    try {
        // 测试使用系统 Chromium
        console.log('1. 测试使用系统 Chromium (executablePath)...');
        const browser = await playwright.chromium.launch({
            headless: true,
            executablePath: '/usr/bin/chromium-browser',
            args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
            timeout: 30000
        });
        
        console.log('   ✓ 系统 Chromium 启动成功');
        
        const page = await browser.newPage();
        await page.goto('https://example.com', { waitUntil: 'networkidle', timeout: 30000 });
        const title = await page.title();
        console.log(`   ✓ 页面标题: ${title}`);
        
        await browser.close();
        console.log('   ✓ 浏览器关闭成功\n');
        
        // 测试使用 chrome 浏览器类型（Playwright MCP 服务器可能使用这个）
        console.log('2. 测试使用 chrome 浏览器类型...');
        try {
            // 注意：playwright.chrome 可能不存在，我们使用 chromium 但指定 channel
            const browser2 = await playwright.chromium.launch({
                headless: true,
                channel: 'chrome',  // 尝试使用 chrome channel
                args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
                timeout: 30000
            });
            console.log('   ✓ chrome 浏览器类型启动成功');
            await browser2.close();
            console.log('   ✓ chrome 浏览器关闭成功');
        } catch (error) {
            console.log(`   ✗ chrome 浏览器类型启动失败: ${error.message}`);
            console.log('   注意：这可能是因为系统没有安装 Chrome，只有 Chromium');
        }
        
        console.log('\n结论：');
        console.log('- Playwright 可以使用系统 Chromium (executablePath)');
        console.log('- Playwright MCP 服务器可能需要配置才能使用系统 Chromium');
        console.log('- 如果 Playwright MCP 服务器使用 "chrome" 浏览器类型，可能需要安装 Chrome 或配置 channel');
        
    } catch (error) {
        console.log(`✗ 测试失败: ${error.message}`);
        console.log('错误详情:', error);
    }
}

testPlaywrightDirect().catch(console.error);