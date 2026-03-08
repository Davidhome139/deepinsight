#!/bin/bash

echo "=== MCP Server Health Check ==="
echo ""

# 检查 context7
echo "1. Checking Context7..."
timeout 5 npx -y @upstash/context7-mcp 2>&1 | head -1 &
PID=$!
sleep 2
if kill -0 $PID 2>/dev/null; then
    echo "   Context7: Running"
    kill $PID 2>/dev/null
else
    echo "   Context7: Failed to start"
fi
echo ""

# 检查 fetch
echo "2. Checking Fetch..."
timeout 5 npx -y fetch-mcp 2>&1 | head -1 &
PID=$!
sleep 2
if kill -0 $PID 2>/dev/null; then
    echo "   Fetch: Running"
    kill $PID 2>/dev/null
else
    echo "   Fetch: Failed to start"
fi
echo ""

# 检查 filesystem
echo "3. Checking Filesystem..."
timeout 5 npx -y @modelcontextprotocol/server-filesystem /app 2>&1 | head -1 &
PID=$!
sleep 2
if kill -0 $PID 2>/dev/null; then
    echo "   Filesystem: Running"
    kill $PID 2>/dev/null
else
    echo "   Filesystem: Failed to start"
fi
echo ""

# 检查 memory
echo "4. Checking Memory..."
timeout 5 npx -y @modelcontextprotocol/server-memory 2>&1 | head -1 &
PID=$!
sleep 2
if kill -0 $PID 2>/dev/null; then
    echo "   Memory: Running"
    kill $PID 2>/dev/null
else
    echo "   Memory: Failed to start"
fi
echo ""

# 检查 playwright
echo "5. Checking Playwright..."
timeout 5 npx -y playwright-mcp 2>&1 | head -1 &
PID=$!
sleep 2
if kill -0 $PID 2>/dev/null; then
    echo "   Playwright: Running"
    kill $PID 2>/dev/null
else
    echo "   Playwright: Failed to start"
fi
echo ""

# 检查 puppeteer
echo "6. Checking Puppeteer..."
timeout 5 npx -y @modelcontextprotocol/server-puppeteer 2>&1 | head -1 &
PID=$!
sleep 2
if kill -0 $PID 2>/dev/null; then
    echo "   Puppeteer: Running"
    kill $PID 2>/dev/null
else
    echo "   Puppeteer: Failed to start"
fi
echo ""

echo "=== Health Check Complete ==="
