// 测试 Playwright MCP 服务器使用的浏览器
const path = require('path');
const playwrightPath = path.join('/usr/local/lib/node_modules/@playwright/mcp/node_modules/playwright');
const playwright = require(playwrightPath);

async function testMCPBrowser() {
    console.log('测试 Playwright MCP 服务器浏览器配置...\n');
    
    // 检查环境变量
    console.log('环境变量:');
    console.log('  PLAYWRIGHT_BROWSER:', process.env.PLAYWRIGHT_BROWSER || '未设置');
    console.log('  PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH:', process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH || '未设置');
    console.log('  CHROMIUM_PATH:', process.env.CHROMIUM_PATH || '未设置');
    
    // 测试使用 chrome 浏览器类型
    console.log('\n测试使用 chrome 浏览器类型...');
    try {
        const browser = await playwright.chromium.launch({
            headless: true,
            // 不指定 executablePath，让 Playwright 自己决定
            args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
            timeout: 15000
        });
        console.log('  ✓ chrome 浏览器启动成功（使用 Playwright 默认浏览器）');
        await browser.close();
        console.log('  ✓ chrome 浏览器关闭成功');
    } catch (error) {
        console.log(`  ✗ chrome 浏览器启动失败: ${error.message}`);
    }
    
    // 测试使用系统 Chromium
    console.log('\n测试使用系统 Chromium...');
    try {
        const browser = await playwright.chromium.launch({
            headless: true,
            executablePath: '/usr/bin/chromium-browser',
            args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
            timeout: 15000
        });
        console.log('  ✓ 系统 Chromium 启动成功');
        
        const page = await browser.newPage();
        await page.goto('https://example.com');
        const title = await page.title();
        console.log(`  ✓ 页面标题: ${title}`);
        
        await browser.close();
        console.log('  ✓ 系统 Chromium 关闭成功');
    } catch (error) {
        console.log(`  ✗ 系统 Chromium 启动失败: ${error.message}`);
    }
    
    console.log('\n结论:');
    console.log('1. Playwright 支持 chromium 浏览器类型');
    console.log('2. 可以指定 executablePath 使用系统 Chromium');
    console.log('3. Playwright MCP 服务器可能需要配置才能使用系统 Chromium');
}

testMCPBrowser().catch(console.error);