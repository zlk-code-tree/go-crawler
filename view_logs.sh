#!/bin/bash

# 抖音电商罗盘爬虫 - 日志查看工具

LOG_DIR="./logs"
TODAY_LOG="$LOG_DIR/crawler-$(date +%Y%m%d).log"

echo "=============================================="
echo "  📊 抖音电商罗盘爬虫 - 日志查看工具"
echo "=============================================="
echo ""

# 检查日志目录
if [ ! -d "$LOG_DIR" ]; then
    echo "❌ 日志目录不存在：$LOG_DIR"
    exit 1
fi

# 显示日志文件列表
echo "📁 日志文件列表："
echo "----------------------------------------------"
ls -lht "$LOG_DIR"/*.log 2>/dev/null | head -10
echo ""

# 显示今天的日志文件
if [ -f "$TODAY_LOG" ]; then
    echo "✅ 今天的日志文件：$TODAY_LOG"
    echo "   大小：$(du -h "$TODAY_LOG" | cut -f1)"
    echo "   行数：$(wc -l < "$TODAY_LOG")"
else
    echo "⚠️  今天的日志文件不存在"
fi
echo ""

# 菜单选择
echo "=============================================="
echo "  请选择操作："
echo "=============================================="
echo "  1) 实时查看最新日志 (tail -f)"
echo "  2) 查看最新 100 行日志"
echo "  3) 查看错误日志 (ERROR)"
echo "  4) 查看警告日志 (WARNING)"
echo "  5) 查看爬虫任务统计"
echo "  6) 清理 7 天前的旧日志"
echo "  7) 导出今天日志到文本"
echo "  0) 退出"
echo "=============================================="
echo ""

read -p "请输入选项 [0-7]: " choice

case $choice in
    1)
        echo ""
        echo "🔍 实时查看最新日志 (Ctrl+C 退出)..."
        echo "----------------------------------------------"
        tail -f "$TODAY_LOG"
        ;;
    2)
        echo ""
        echo "📄 最新 100 行日志："
        echo "----------------------------------------------"
        tail -100 "$TODAY_LOG"
        ;;
    3)
        echo ""
        echo "❌ 错误日志："
        echo "----------------------------------------------"
        grep "ERROR\|❌" "$TODAY_LOG" | tail -50
        ;;
    4)
        echo ""
        echo "⚠️  警告日志："
        echo "----------------------------------------------"
        grep "WARNING\|⚠️" "$TODAY_LOG" | tail -50
        ;;
    5)
        echo ""
        echo "📊 爬虫任务统计："
        echo "----------------------------------------------"
        echo "总任务数：$(grep -c "爬虫任务开始" "$TODAY_LOG" 2>/dev/null || echo 0)"
        echo "成功任务：$(grep -c "爬虫任务全部完成" "$TODAY_LOG" 2>/dev/null || echo 0)"
        echo "失败任务：$(grep -c "爬虫任务异常终止\|爬虫任务执行失败" "$TODAY_LOG" 2>/dev/null || echo 0)"
        echo ""
        echo "数据抓取统计："
        echo "  - 商品数据：$(grep -oP "共获取 \K\d+(?= 条商品数据)" "$TODAY_LOG" | awk '{s+=$1} END {print s}' 2>/dev/null || echo 0) 条"
        echo "  - 详情数据：$(grep -oP "详情抓取完成，共获取 \K\d+(?= 条)" "$TODAY_LOG" | awk '{s+=$1} END {print s}' 2>/dev/null || echo 0) 条"
        echo "  - 视频数据：$(grep -oP "视频分析抓取完成，共获取 \K\d+(?= 条)" "$TODAY_LOG" | awk '{s+=$1} END {print s}' 2>/dev/null || echo 0) 条"
        echo ""
        echo "导出统计："
        echo "  - CSV 文件：$(ls -1 "$LOG_DIR"/../exports/*.csv 2>/dev/null | wc -l) 个"
        echo "  - Excel 文件：$(ls -1 "$LOG_DIR"/../exports/*.xlsx 2>/dev/null | wc -l) 个"
        ;;
    6)
        echo ""
        echo "🗑️  清理 7 天前的旧日志..."
        find "$LOG_DIR" -name "*.log" -mtime +7 -delete
        echo "✅ 清理完成"
        ;;
    7)
        EXPORT_FILE="$LOG_DIR/../logs_export_$(date +%Y%m%d_%H%M%S).txt"
        cp "$TODAY_LOG" "$EXPORT_FILE"
        echo ""
        echo "✅ 日志已导出到：$EXPORT_FILE"
        ;;
    0)
        echo "👋 退出"
        exit 0
        ;;
    *)
        echo "❌ 无效选项"
        exit 1
        ;;
esac

echo ""
echo "=============================================="
