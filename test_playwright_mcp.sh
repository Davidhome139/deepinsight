#!/bin/bash
# 测试Playwright MCP服务器

echo "测试Playwright MCP服务器..."
echo "=============================="

# 1. 检查playwright-mcp命令
echo "1. 检查playwright-mcp命令..."
which playwright-mcp
if [ $? -eq 0 ]; then
    echo "✅ playwright-mcp命令存在"
    playwright-mcp --version
else
    echo "❌ playwright-mcp命令不存在"
    exit 1
fi

echo ""
echo "2. 测试简单运行..."
timeout 3 playwright-mcp --help
if [ $? -eq 124 ]; then
    echo "✅ playwright-mcp可以启动（超时退出是正常的）"
else
    echo "⚠️ playwright-mcp可能有问题，退出码: $?"
fi

echo ""
echo "3. 检查浏览器..."
echo "检查Chrome符号链接:"
ls -la /opt/google/chrome/chrome
if [ -f "/opt/google/chrome/chrome" ]; then
    echo "✅ Chrome符号链接存在"
else
    echo "❌ Chrome符号链接不存在"
fi

echo ""
echo "4. 检查Playwright浏览器缓存..."
ls -la /home/pwuser/.cache/ms-playwright/
echo ""

echo "5. 环境变量检查:"
echo "PLAYWRIGHT_BROWSERS_PATH: $PLAYWRIGHT_BROWSERS_PATH"
echo "PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH: $PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH"
echo "DISPLAY: $DISPLAY"
echo "PLAYWRIGHT_BROWSER: $PLAYWRIGHT_BROWSER"

echo ""
echo "测试完成。"