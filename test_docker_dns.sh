#!/bin/bash
# Docker容器内DNS测试脚本

echo "=========================================="
echo "Docker容器内DNS解析测试"
echo "=========================================="

# 测试1: 检查容器内是否可以解析api.context7.com
echo -e "\n1. 测试api.context7.com解析:"
docker exec backend-1 nslookup api.context7.com 2>&1 | grep -A5 "Name:"

# 测试2: 检查容器内的hosts文件
echo -e "\n2. 检查容器内hosts文件:"
docker exec backend-1 cat /etc/hosts | grep -i context7 || echo "未找到context7相关条目"

# 测试3: 测试ping（如果支持）
echo -e "\n3. 测试ping api.context7.com:"
docker exec backend-1 ping -c 2 api.context7.com 2>&1 | head -5

# 测试4: 测试curl连接（如果可用）
echo -e "\n4. 测试curl连接:"
docker exec backend-1 curl -I https://api.context7.com 2>&1 | head -5

# 测试5: 检查DNS配置
echo -e "\n5. 检查容器DNS配置:"
docker exec backend-1 cat /etc/resolv.conf

echo -e "\n=========================================="
echo "如果api.context7.com无法解析，需要:"
echo "1. 重启Docker容器: docker-compose down && docker-compose up -d"
echo "2. 验证extra_hosts配置是否正确应用"
echo "3. 检查防火墙/网络设置"
echo "=========================================="