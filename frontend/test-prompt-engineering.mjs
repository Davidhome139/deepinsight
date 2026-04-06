import { chromium } from 'playwright';

(async () => {
  const browser = await chromium.launch({ 
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });
  const page = await browser.newPage({
    viewport: { width: 1280, height: 720 }
  });

  console.log('正在打开页面...');
  await page.goto('http://localhost/login', { waitUntil: 'networkidle' });
  console.log('页面已打开');

  // 等待页面加载
  await page.waitForTimeout(2000);
  
  // 检查是否需要登录
  const isLoginPage = await page.url().includes('/login');
  if (isLoginPage) {
    console.log('检测到登录页面，尝试自动登录...');
    // 查找用户名输入框
    const usernameInput = page.locator('input[type="text"], input[name="username"]').first();
    const usernameExists = await usernameInput.count() > 0;
    
    if (usernameExists) {
      console.log('找到用户名输入框，输入 admin...');
      await usernameInput.fill('admin');
      
      // 查找密码输入框
      const passwordInput = page.locator('input[type="password"], input[name="password"]').first();
      const passwordExists = await passwordInput.count() > 0;
      
      if (passwordExists) {
        console.log('找到密码输入框，输入 password...');
        await passwordInput.fill('password');
        
        // 查找登录按钮
        const loginButton = page.locator('button:has-text("登录"), button[type="submit"]').first();
        const loginButtonExists = await loginButton.count() > 0;
        
        if (loginButtonExists) {
          console.log('点击登录按钮...');
          await loginButton.click();
          await page.waitForTimeout(2000);
          console.log('登录完成');
        }
      }
    }
    
    // 导航到聊天页面
    console.log('导航到聊天页面...');
    await page.goto('http://localhost');
    await page.waitForTimeout(2000);
  }

  // 查找并点击提示词工程按钮（魔术棒图标）
  console.log('查找提示词工程按钮...');
  // 先尝试通过文本查找
  let buttonLocator = page.locator('button:has-text("提示词工程")').first();
  let buttonExists = await buttonLocator.count() > 0;
  
  if (!buttonExists) {
    // 尝试通过类名和图标查找
    buttonLocator = page.locator('.toolbar-btn:has(.el-icon)').filter({ hasText: /提示词工程 |Prompt/ }).first();
    buttonExists = await buttonLocator.count() > 0;
  }
  
  if (!buttonExists) {
    // 最后尝试：查找所有 toolbar-btn，找到包含 MagicStick 图标的
    const allToolbarButtons = page.locator('.toolbar-btn');
    const count = await allToolbarButtons.count();
    console.log('找到工具栏按钮数量:', count);
    
    for (let i = 0; i < count; i++) {
      const btnText = await allToolbarButtons.nth(i).textContent();
      console.log('按钮', i, '文本:', btnText);
      if (btnText && (btnText.includes('提示词工程') || btnText.includes('Prompt'))) {
        buttonLocator = allToolbarButtons.nth(i);
        buttonExists = true;
        break;
      }
    }
  }
  
  console.log('按钮存在:', buttonExists);

  if (!buttonExists) {
    console.log('❌ 未找到提示词工程按钮');
    await page.screenshot({ path: 'test-no-button-found.png' });
    await browser.close();
    return;
  }
  
  await buttonLocator.click();

  console.log('已点击提示词工程按钮');
  
  // 等待弹出层出现
  await page.waitForTimeout(1000);
  
  // 截图查看当前状态
  await page.screenshot({ path: 'test-after-open-panel.png', fullPage: false });
  console.log('已截图：test-after-open-panel.png');

  // 检查弹出层是否存在
  const popoverExists = await page.locator('.el-popover, [role="dialog"], .prompt-engineering-panel').count() > 0;
  console.log('弹出层存在:', popoverExists);

  if (!popoverExists) {
    console.log('❌ 测试失败：弹出层未打开');
    await browser.close();
    return;
  }

  // 查找角色选择下拉框
  console.log('查找角色选择下拉框...');
  const roleSelect = page.locator('.role-selector .el-select').first();
  const roleSelectExists = await roleSelect.count() > 0;
  console.log('角色选择框存在:', roleSelectExists);

  let popoverStillOpenAfterRole = true;

  if (roleSelectExists) {
    console.log('点击角色选择框...');
    await roleSelect.click();
    await page.waitForTimeout(500);

    // 检查弹出层是否仍然打开
    popoverStillOpenAfterRole = await page.locator('.el-popover, .prompt-engineering-panel').count() > 0;
    console.log('选择角色后弹出层是否仍打开:', popoverStillOpenAfterRole);

    if (!popoverStillOpenAfterRole) {
      console.log('❌ 测试失败：选择角色后弹出层关闭了');
      await page.screenshot({ path: 'test-role-select-failed.png' });
    } else {
      console.log('✅ 测试通过：选择角色后弹出层仍打开');
    }
  }

  // 重新打开弹出层（如果关闭了）
  if (!popoverStillOpenAfterRole) {
    console.log('重新打开弹出层...');
    await buttonLocator.click();
    await page.waitForTimeout(1000);
  }

  // 查找输出格式选择框
  console.log('查找输出格式选择框...');
  const formatSelect = page.locator('.output-format-selector .el-select').first();
  const formatSelectExists = await formatSelect.count() > 0;
  console.log('输出格式选择框存在:', formatSelectExists);

  if (formatSelectExists) {
    console.log('点击输出格式选择框...');
    await formatSelect.click();
    await page.waitForTimeout(500);

    // 检查弹出层是否仍然打开
    const popoverStillOpenAfterFormat = await page.locator('.el-popover, .prompt-engineering-panel').count() > 0;
    console.log('选择输出格式后弹出层是否仍打开:', popoverStillOpenAfterFormat);

    if (!popoverStillOpenAfterFormat) {
      console.log('❌ 测试失败：选择输出格式后弹出层关闭了');
      await page.screenshot({ path: 'test-format-select-failed.png' });
    } else {
      console.log('✅ 测试通过：选择输出格式后弹出层仍打开');
      
      // 选择一个格式
      console.log('选择 JSON 格式...');
      const jsonOption = page.locator('.el-select-dropdown__item:has-text("JSON")').first();
      await jsonOption.click();
      await page.waitForTimeout(500);

      const popoverStillOpenAfterJson = await page.locator('.el-popover, .prompt-engineering-panel').count() > 0;
      console.log('选择 JSON 后弹出层是否仍打开:', popoverStillOpenAfterJson);

      if (!popoverStillOpenAfterJson) {
        console.log('❌ 测试失败：选择 JSON 后弹出层关闭了');
        await page.screenshot({ path: 'test-json-select-failed.png' });
      } else {
        console.log('✅ 测试通过：选择 JSON 后弹出层仍打开');
      }
    }
  }

  // 截图最终状态
  await page.screenshot({ path: 'test-final-state.png' });
  console.log('已截图：test-final-state.png');

  await browser.close();
  console.log('测试完成');
})();
