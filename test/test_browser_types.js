const playwright = require('playwright');

async function testBrowserTypes() {
    console.log('测试 Playwright 支持的浏览器类型...\n');
    
    // 检查 Playwright 模块导出的浏览器类型
    console.log('Playwright 模块导出:');
    console.log('  chromium:', typeof playwright.chromium !== 'undefined' ? '✓ 支持' : '✗ 不支持');
    console.log('  firefox:', typeof playwright.firefox !== 'undefined' ? '✓ 支持' : '✗ 不支持');
    console.log('  webkit:', typeof playwright.webkit !== 'undefined' ? '✓ 支持' : '✗ 不支持');
    
    // 尝试启动不同浏览器
    const browsers = [
        { name: 'chromium', launcher: playwright.chromium },
        { name: 'firefox', launcher: playwright.firefox },
        { name: 'webkit', launcher: playwright.webkit }
    ];
    
    for (const browser of browsers) {
        console.log(`\n测试 ${browser.name}...`);
        try {
            const instance = await browser.launcher.launch({
                headless: true,
                timeout: 10000
            });
            console.log(`  ✓ ${browser.name} 启动成功`);
            await instance.close();
            console.log(`  ✓ ${browser.name} 关闭成功`);
        } catch (error) {
            console.log(`  ✗ ${browser.name} 启动失败: ${error.message}`);
        }
    }
    
    console.log('\n\n测试使用系统 Chromium...');
    try {
        const browser = await playwright.chromium.launch({
            headless: true,
            executablePath: '/usr/bin/chromium-browser',
            args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
            timeout: 10000
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
}

testBrowserTypes().catch(console.error);