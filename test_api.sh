#!/bin/bash

# 测试爬虫 API
echo "=========================================="
echo "测试 go-crawler API"
echo "=========================================="

# 测试 1: Hello API
echo ""
echo "📝 测试 1: Hello API"
echo "------------------------------------------"
curl -s http://localhost:8000/hello | jq .

# 测试 2: Crawler API (模拟数据模式)
echo ""
echo "🕷️ 测试 2: Crawler API"
echo "------------------------------------------"
curl -X POST http://localhost:8000/crawler \
  -H "Content-Type: application/json" \
  -d '{"targetUrl":"https://compass.jinritemai.com/shop"}' | jq .

# 测试 3: Crawler API (空参数)
echo ""
echo "❌ 测试 3: Crawler API (空参数 - 应该失败)"
echo "------------------------------------------"
curl -X POST http://localhost:8000/crawler \
  -H "Content-Type: application/json" \
  -d '{}' | jq .

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
