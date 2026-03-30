#!/bin/bash

# 抖音电商罗盘爬虫测试脚本

echo "=============================================="
echo "  抖音电商罗盘爬虫服务 - 测试脚本"
echo "=============================================="
echo ""

# 检查服务是否运行
echo "📡 检查服务状态..."
if curl -s http://localhost:8000/hello > /dev/null 2>&1; then
    echo "✅ 服务运行正常"
else
    echo "❌ 服务未运行，请先启动：./main"
    exit 1
fi

echo ""
echo "=============================================="
echo "  测试 1: Hello API"
echo "=============================================="
curl -s http://localhost:8000/hello
echo ""

echo ""
echo "=============================================="
echo "  测试 2: Swagger 文档"
echo "=============================================="
echo "访问地址：http://localhost:8000/swagger"
echo ""

echo ""
echo "=============================================="
echo "  测试 3: 爬虫 API (需要登录)"
echo "=============================================="
echo "注意：此测试会启动浏览器并等待登录，耗时较长"
echo ""
read -p "是否执行爬虫测试？(y/n): " confirm
if [ "$confirm" = "y" ]; then
    echo "🕷️  启动爬虫任务..."
    curl -X POST http://localhost:8000/crawler \
      -H "Content-Type: application/json" \
      -d '{
        "targetUrl": "https://compass.jinritemai.com/shop",
        "loginType": "qrcode"
      }' | jq .
else
    echo "⏭️  跳过爬虫测试"
fi

echo ""
echo "=============================================="
echo "  测试完成"
echo "=============================================="
