#!/bin/bash

# 抖音电商罗盘爬虫 - 完整测试流程

echo "=============================================="
echo "  🕷️  抖音电商罗盘爬虫 - 完整测试流程"
echo "=============================================="
echo ""

PROJECT_DIR="/Users/zhanglikun/WorkSpace/app/goCode/ly-project/go-crawler"
cd "$PROJECT_DIR"

# 1. 检查服务状态
echo "=============================================="
echo "  1️⃣  检查服务状态"
echo "=============================================="
if lsof -ti :8000 > /dev/null 2>&1; then
    echo "✅ 服务运行中 (PID: $(lsof -ti :8000))"
else
    echo "❌ 服务未运行，正在启动..."
    ./main > /dev/null 2>&1 &
    sleep 3
    if lsof -ti :8000 > /dev/null 2>&1; then
        echo "✅ 服务已启动"
    else
        echo "❌ 服务启动失败"
        exit 1
    fi
fi
echo ""

# 2. 检查数据库连接
echo "=============================================="
echo "  2️⃣  检查数据库连接"
echo "=============================================="
if docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ 数据库连接成功"
    
    # 检查表
    tables=$(docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SHOW TABLES;" 2>/dev/null | grep -E "tiktok|video|crawl")
    if [ -n "$tables" ]; then
        echo "✅ 数据表已创建:"
        echo "$tables" | sed 's/^/   - /'
    else
        echo "❌ 数据表未创建"
    fi
else
    echo "❌ 数据库连接失败"
fi
echo ""

# 3. 测试 Hello API
echo "=============================================="
echo "  3️⃣  测试 Hello API"
echo "=============================================="
response=$(curl -s http://localhost:8000/hello)
if [ "$response" = "Hello World!" ]; then
    echo "✅ Hello API 正常"
else
    echo "❌ Hello API 异常：$response"
fi
echo ""

# 4. 检查数据目录
echo "=============================================="
echo "  4️⃣  检查数据目录"
echo "=============================================="
for dir in logs exports cookies checkpoints; do
    if [ -d "$dir" ]; then
        count=$(find "$dir" -type f 2>/dev/null | wc -l | tr -d ' ')
        echo "✅ $dir/ - $count 个文件"
    else
        echo "❌ $dir/ - 目录不存在"
        mkdir -p "$dir"
    fi
done
echo ""

# 5. 准备爬虫测试
echo "=============================================="
echo "  5️⃣  准备爬虫测试"
echo "=============================================="
echo ""
echo "⚠️  注意：爬虫测试会启动浏览器并访问抖音电商罗盘"
echo "   需要手动完成登录操作（扫码或账号密码）"
echo ""
read -p "是否继续执行爬虫测试？(y/n): " confirm

if [ "$confirm" != "y" ]; then
    echo ""
    echo "⏭️  跳过爬虫测试"
    echo ""
    echo "你可以手动测试："
    echo "  curl -X POST http://localhost:8000/crawler \\"
    echo "    -H 'Content-Type: application/json' \\"
    echo "    -d '{\"targetUrl\":\"https://compass.jinritemai.com/shop\"}'"
    echo ""
    exit 0
fi

# 6. 执行爬虫测试
echo ""
echo "=============================================="
echo "  6️⃣  执行爬虫测试"
echo "=============================================="
echo ""
echo "🚀 启动爬虫任务..."
echo "   目标URL: https://compass.jinritemai.com/shop"
echo "   登录方式：扫码登录"
echo ""
echo "⏳ 请在打开的浏览器中完成登录..."
echo ""

# 发送爬虫请求
start_time=$(date +%s)
response=$(curl -s -X POST http://localhost:8000/crawler \
  -H "Content-Type: application/json" \
  -d '{
    "targetUrl": "https://compass.jinritemai.com/shop",
    "loginType": "qrcode"
  }')

end_time=$(date +%s)
duration=$((end_time - start_time))

# 解析响应
success=$(echo "$response" | grep -o '"success":true' || echo "")
message=$(echo "$response" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)

if [ -n "$success" ]; then
    echo "✅ 爬虫任务成功！"
    echo ""
    echo "📊 响应信息："
    echo "   状态：成功"
    echo "   消息：$message"
    echo "   耗时：${duration}秒"
    echo ""
    
    # 查看导出的文件
    echo "📥 导出的文件："
    if [ -d "exports" ] && [ "$(ls -A exports 2>/dev/null)" ]; then
        ls -lh exports/ | tail -5
    else
        echo "   暂无导出文件"
    fi
    echo ""
    
    # 查看数据库
    echo "🗄️  数据库记录："
    count=$(docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SELECT COUNT(*) FROM tiktok_ranking_list;" 2>/dev/null | tail -1)
    echo "   商品榜单数据：$count 条"
    
    video_count=$(docker exec mysql8 mysql -uroot -proot easyadmin8 -e "SELECT COUNT(*) FROM video_analysis;" 2>/dev/null | tail -1)
    echo "   视频分析数据：$video_count 条"
    echo ""
    
    # 查看最新日志
    echo "📝 最新日志："
    today_log="logs/crawler-$(date +%Y%m%d).log"
    if [ -f "$today_log" ]; then
        echo "   文件：$today_log"
        echo "   大小：$(du -h "$today_log" | cut -f1)"
        echo "   最后 10 行:"
        tail -10 "$today_log" | sed 's/^/   /'
    else
        echo "   暂无日志文件"
    fi
    
else
    echo "❌ 爬虫任务失败"
    echo ""
    echo "错误信息：$message"
    echo ""
    echo "请查看日志获取详细信息："
    tail -20 "logs/crawler-$(date +%Y%m%d).log" 2>/dev/null | sed 's/^/   /'
fi

echo ""
echo "=============================================="
echo "  ✅ 测试完成"
echo "=============================================="
echo ""
echo "💡 后续操作："
echo "   1. 查看导出数据：ls -lh exports/"
echo "   2. 查看数据库：docker exec -it mysql8 mysql -uroot -proot easyadmin8"
echo "   3. 查看实时日志：tail -f logs/crawler-$(date +%Y%m%d).log"
echo "   4. 运行检查工具：./check_data.sh"
echo ""
