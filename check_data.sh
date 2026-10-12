#!/bin/bash

# 抖音电商罗盘爬虫 - 数据检查工具

echo "=============================================="
echo "  📊 抖音电商罗盘爬虫 - 数据检查工具"
echo "=============================================="
echo ""

PROJECT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)" || exit 1
cd "$PROJECT_DIR" || exit 1

echo "📁 项目目录：$PROJECT_DIR"
echo ""

# 1. 检查数据目录
echo "=============================================="
echo "  1️⃣  检查数据目录"
echo "=============================================="
for dir in logs exports cookies checkpoints database; do
    if [ -d "$dir" ]; then
        count=$(find "$dir" -type f 2>/dev/null | wc -l)
        echo "✅ $dir/ - $count 个文件"
        if [ "$dir" = "logs" ] && [ $count -gt 0 ]; then
            latest=$(ls -t "$dir"/*.log 2>/dev/null | head -1)
            if [ -n "$latest" ]; then
                echo "   └─ 最新：$(basename "$latest") ($(du -h "$latest" | cut -f1))"
            fi
        fi
    else
        echo "❌ $dir/ - 目录不存在"
        mkdir -p "$dir"
        echo "   └─ 已创建目录"
    fi
done
echo ""

# 2. 检查导出文件
echo "=============================================="
echo "  2️⃣  检查导出文件"
echo "=============================================="
csv_count=$(find exports/ -name "*.csv" 2>/dev/null | wc -l)
xlsx_count=$(find exports/ -name "*.xlsx" 2>/dev/null | wc -l)
echo "CSV 文件：$csv_count 个"
echo "Excel 文件：$xlsx_count 个"

if [ $csv_count -gt 0 ] || [ $xlsx_count -gt 0 ]; then
    echo ""
    echo "最新导出文件："
    ls -lht exports/*.{csv,xlsx} 2>/dev/null | head -5
else
    echo ""
    echo "⚠️  暂无导出文件（需要先运行爬虫任务）"
fi
echo ""

# 3. 检查 Cookie 文件
echo "=============================================="
echo "  3️⃣  检查 Cookie 文件"
echo "=============================================="
if [ -f "cookies/cookies.json" ]; then
    size=$(du -h cookies/cookies.json | cut -f1)
    cookie_count=$(grep -c '"name"' cookies/cookies.json 2>/dev/null || echo 0)
    echo "✅ Cookie 文件存在"
    echo "   大小：$size"
    echo "   Cookie 数量：$cookie_count 个"
    echo "   修改时间：$(stat -f '%Sm' cookies/cookies.json 2>/dev/null || stat -c '%y' cookies/cookies.json 2>/dev/null | cut -d'.' -f1)"
else
    echo "⚠️  Cookie 文件不存在"
    echo "   首次运行爬虫后会自动保存"
fi
echo ""

# 4. 检查断点文件
echo "=============================================="
echo "  4️⃣  检查断点文件"
echo "=============================================="
checkpoint_count=$(find checkpoints/ -name "*.json" 2>/dev/null | wc -l)
if [ $checkpoint_count -gt 0 ]; then
    echo "⚠️  发现 $checkpoint_count 个未完成的断点文件："
    ls -lht checkpoints/*.json 2>/dev/null | head -5
else
    echo "✅ 没有未完成的断点任务"
fi
echo ""

# 5. 检查数据库配置
echo "=============================================="
echo "  5️⃣  检查数据库配置"
echo "=============================================="
if [ -f "manifest/config/config.yaml" ]; then
    echo "✅ 配置文件存在"
    echo ""
    echo "数据库配置："
    grep -A5 "database:" manifest/config/config.yaml | grep -E "type:|link:" | sed 's/^/   /'
    echo ""
    
    # 提取数据库信息
    db_link=$(grep "link:" manifest/config/config.yaml | head -1 | sed 's/.*link: "//' | sed 's/".*//')
    db_user=$(echo "$db_link" | cut -d':' -f1)
    db_host=$(echo "$db_link" | grep -oP 'tcp\(\K[^:]+' || echo "127.0.0.1")
    db_port=$(echo "$db_link" | grep -oP ':\K\d+(?=/)' || echo "3306")
    db_name=$(echo "$db_link" | grep -oP '/\K[^?]+' || echo "easyadmin8")
    
    echo "   数据库名：$db_name"
    echo "   主机：$db_host:$db_port"
    echo "   用户：$db_user"
    echo ""
    
    # 尝试检查数据库连接
    echo "测试数据库连接..."
    if command -v mysql &> /dev/null; then
        if mysql -u"$db_user" -p"$db_user" -h"$db_host" -P"$db_port" -e "SELECT 1;" &> /dev/null; then
            echo "✅ 数据库连接成功"
            
            # 检查表是否存在
            echo ""
            echo "检查数据表："
            tables=$(mysql -u"$db_user" -p"$db_user" -h"$db_host" -P"$db_port" -D"$db_name" -e "SHOW TABLES;" 2>/dev/null | grep -E "tiktok|video|crawl")
            if [ -n "$tables" ]; then
                echo "$tables" | sed 's/^/   ✅ /'
            else
                echo "   ⚠️  表不存在，需要运行初始化脚本"
            fi
        else
            echo "❌ 数据库连接失败"
            echo "   请检查 MySQL 服务是否启动"
            echo "   或账号密码是否正确"
        fi
    else
        echo "⚠️  未安装 mysql 客户端，跳过连接测试"
        echo "   可以手动安装：brew install mysql"
    fi
else
    echo "❌ 配置文件不存在"
fi
echo ""

# 6. 检查服务状态
echo "=============================================="
echo "  6️⃣  检查服务状态"
echo "=============================================="
if lsof -i :8000 &> /dev/null; then
    pid=$(lsof -ti :8000)
    echo "✅ 服务运行中 (PID: $pid)"
    echo ""
    echo "访问地址："
    echo "   Swagger: http://localhost:8000/swagger"
    echo "   API: http://localhost:8000/crawler"
else
    echo "❌ 服务未运行"
    echo ""
    echo "启动命令："
    echo "   cd $PROJECT_DIR"
    echo "   ./main"
fi
echo ""

# 7. 数据统计
echo "=============================================="
echo "  7️⃣  数据统计"
echo "=============================================="
if [ -d "logs" ] && [ "$(ls -A logs 2>/dev/null)" ]; then
    today_log="logs/crawler-$(date +%Y%m%d).log"
    if [ -f "$today_log" ]; then
        total_tasks=$(grep -c "爬虫任务开始" "$today_log" 2>/dev/null || echo 0)
        success_tasks=$(grep -c "爬虫任务全部完成" "$today_log" 2>/dev/null || echo 0)
        failed_tasks=$(grep -c "异常终止\|执行失败" "$today_log" 2>/dev/null || echo 0)
        
        echo "今日任务统计："
        echo "   总任务数：$total_tasks"
        echo "   成功：$success_tasks"
        echo "   失败：$failed_tasks"
        
        if [ $success_tasks -gt 0 ]; then
            echo ""
            echo "数据抓取："
            products=$(grep -oP "共获取 \K\d+(?= 条商品数据)" "$today_log" 2>/dev/null | awk '{s+=$1} END {print s+0}')
            videos=$(grep -oP "视频分析抓取完成，共获取 \K\d+(?= 条)" "$today_log" 2>/dev/null | awk '{s+=$1} END {print s+0}')
            echo "   商品数据：$products 条"
            echo "   视频数据：$videos 条"
        fi
    else
        echo "⚠️  今天暂无日志"
    fi
else
    echo "⚠️  暂无日志文件"
fi
echo ""

echo "=============================================="
echo "  💡 使用提示"
echo "=============================================="
echo "1. 查看实时日志：tail -f logs/crawler-$(date +%Y%m%d).log"
echo "2. 查看导出文件：ls -lh exports/"
echo "3. 查看 Cookie:   cat cookies/cookies.json | jq ."
echo "4. 清理旧日志：   find logs/ -mtime +7 -delete"
echo "5. 初始化数据库：mysql -u 用户 -p < database/init.sql"
echo "=============================================="
echo ""
