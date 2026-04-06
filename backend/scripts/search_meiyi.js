// Playwright脚本：搜索美伊战报
async function searchMeiYiBattleReport(page) {
    console.log('开始搜索美伊战报...');
    
    // 1. 导航到百度
    await page.goto('https://www.baidu.com');
    console.log('已导航到百度首页');
    
    // 2. 在搜索框中输入关键词
    const searchInput = '#kw';
    await page.fill(searchInput, '美伊战报 最新 美国伊朗冲突 2024');
    console.log('已输入搜索关键词');
    
    // 3. 点击搜索按钮
    const searchButton = '#su';
    await page.click(searchButton);
    console.log('已点击搜索按钮');
    
    // 4. 等待搜索结果加载
    await page.waitForSelector('.result.c-container', { timeout: 10000 });
    console.log('搜索结果已加载');
    
    // 5. 获取搜索结果
    const searchResults = await page.$$eval('.result.c-container', (elements) => {
        return elements.slice(0, 10).map(el => {
            const titleEl = el.querySelector('h3 a');
            const title = titleEl?.textContent?.trim() || '无标题';
            const link = titleEl?.href || '无链接';
            
            // 尝试不同的摘要选择器
            const snippet = el.querySelector('.content-right_8Zs40')?.textContent?.trim() || 
                           el.querySelector('.c-abstract')?.textContent?.trim() || 
                           el.querySelector('.c-span-last')?.textContent?.trim() || 
                           '无摘要';
            
            return {
                title,
                link,
                snippet: snippet.substring(0, 200),
                time: el.querySelector('.c-color-gray2')?.textContent?.trim() || '时间未知'
            };
        });
    });
    
    console.log(`美伊战报搜索结果：`);
    console.log(`共找到 ${searchResults.length} 条相关结果\n`);
    
    // 格式化结果
    let formattedResults = `美伊战报搜索结果（共 ${searchResults.length} 条）：\n\n`;
    searchResults.forEach((result, index) => {
        formattedResults += `${index + 1}. ${result.title}\n`;
        formattedResults += `   链接: ${result.link}\n`;
        formattedResults += `   时间: ${result.time}\n`;
        formattedResults += `   摘要: ${result.snippet}\n`;
        formattedResults += `   ---\n`;
    });
    
    // 6. 保存搜索结果截图
    await page.screenshot({ 
        path: '/tmp/美伊战报搜索结果.png', 
        fullPage: true 
    });
    console.log('已保存搜索结果截图');
    
    return {
        summary: `成功搜索到 ${searchResults.length} 条美伊战报相关信息`,
        results: searchResults,
        formatted: formattedResults,
        screenshotPath: '/tmp/美伊战报搜索结果.png'
    };
}

// 导出函数供MCP调用
module.exports = { searchMeiYiBattleReport };