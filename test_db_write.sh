#!/bin/bash

# 测试数据库写入功能

echo "=============================================="
echo "  🗄️  数据库写入测试"
echo "=============================================="
echo ""

# 1. 检查数据库连接
echo "1️⃣  检查数据库连接..."
if docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ 数据库连接成功"
else
    echo "❌ 数据库连接失败"
    exit 1
fi
echo ""

# 2. 检查表结构
echo "2️⃣  检查数据表..."
tables=$(docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SHOW TABLES;" 2>/dev/null | grep -E "tiktok|video|crawl")
if [ -n "$tables" ]; then
    echo "✅ 数据表存在:"
    echo "$tables" | sed 's/^/   - /'
else
    echo "❌ 数据表不存在"
    exit 1
fi
echo ""

# 3. 检查当前数据量
echo "3️⃣  检查当前数据量..."
count=$(docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SELECT COUNT(*) as count FROM tiktok_ranking_list;" 2>/dev/null | tail -1)
echo "   商品榜单数据：$count 条"

video_count=$(docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SELECT COUNT(*) as count FROM video_analysis;" 2>/dev/null | tail -1)
echo "   视频分析数据：$video_count 条"

task_count=$(docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SELECT COUNT(*) as count FROM crawl_tasks;" 2>/dev/null | tail -1)
echo "   任务记录数据：$task_count 条"
echo ""

# 4. 测试爬虫 API
echo "4️⃣  测试爬虫 API..."
echo "   目标：https://compass.jinritemai.com/shop"
echo ""
read -p "是否执行爬虫测试？(y/n): " confirm

if [ "$confirm" = "y" ]; then
    echo ""
    echo "🚀 启动爬虫任务..."
    start_time=$(date +%s)
    
    response=$(curl -s -X POST http://localhost:8000/crawler \
      -H "Content-Type: application/json" \
      -d '{
        "targetUrl": "https://compass.jinritemai.com/shop",
        "loginType": "qrcode"
      }')
    
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    echo ""
    echo "⏱️  耗时：${duration}秒"
    echo ""
    
    # 检查响应
    success=$(echo "$response" | grep -o '"success":true' || echo "")
    if [ -n "$success" ]; then
        echo "✅ 爬虫任务成功"
    else
        echo "❌ 爬虫任务失败"
        echo "   响应：$response"
    fi
    echo ""
    
    # 等待 5 秒让数据入库
    echo "⏳ 等待数据入库..."
    sleep 5
    echo ""
    
    # 5. 检查数据库数据
    echo "5️⃣  检查数据库数据..."
    new_count=$(docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SELECT COUNT(*) as count FROM tiktok_ranking_list;" 2>/dev/null | tail -1)
    echo "   商品榜单数据：$new_count 条"
    
    new_video_count=$(docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SELECT COUNT(*) as count FROM video_analysis;" 2>/dev/null | tail -1)
    echo "   视频分析数据：$new_video_count 条"
    
    new_task_count=$(docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SELECT COUNT(*) as count FROM crawl_tasks;" 2>/dev/null | tail -1)
    echo "   任务记录数据：$new_task_count 条"
    echo ""
    
    # 6. 查看最新数据
    if [ "$new_count" -gt "$count" ]; then
        echo "6️⃣  查看最新商品数据（前 5 条）..."
        docker exec mysql8 mysql -uroot -proot easyadmin8 -e \
          "SELECT rank, product_name, shop_name, payment_amount FROM tiktok_ranking_list ORDER BY id DESC LIMIT 5;" 2>/dev/null
        echo ""
    fi
    
    # 7. 查看任务记录
    if [ "$new_task_count" -gt "$task_count" ]; then
        echo "7️⃣  查看最新任务记录..."
        docker exec mysql8 mysql -uroot -proot easyadmin8 -e \
          "SELECT task_id, status, product_count, video_count, created_at FROM crawl_tasks ORDER BY id DESC LIMIT 1;" 2>/dev/null
        echo ""
    fi
else
    echo "⏭️  跳过爬虫测试"
fi

echo "=============================================="
echo "  测试完成"
echo "=============================================="
echo ""
echo "💡 手动查看数据库："
echo "   docker exec -it mysql8 mysql -uroot -proot easyadmin8"
echo ""
echo "📝 查看日志："
echo "   tail -f logs/crawler-$(date +%Y%m%d).log"
echo ""
