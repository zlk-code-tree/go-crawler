#!/bin/bash

# 抖音电商罗盘爬虫服务 - 启动脚本

cd "$(dirname "$0")"

echo "=============================================="
echo "  抖音电商罗盘爬虫服务"
echo "=============================================="
echo ""

# 检查是否已运行
if curl -s http://localhost:8000/hello > /dev/null 2>&1; then
    echo "⚠️  服务已在运行（PID: $(lsof -t -i:8000))"
    echo ""
    echo "访问地址:"
    echo "  - Swagger: http://localhost:8000/swagger"
    echo "  - API 文档：http://localhost:8000/api.json"
    echo ""
    read -p "是否重启服务？(y/n): " confirm
    if [ "$confirm" != "y" ]; then
        exit 0
    fi
    
    # 停止现有服务
    echo "🛑 停止现有服务..."
    kill $(lsof -t -i:8000) 2>/dev/null
    sleep 2
fi

# 检查编译
if [ ! -f "./main" ]; then
    echo "🔨 首次运行，正在编译..."
    go build -o main .
    if [ $? -ne 0 ]; then
        echo "❌ 编译失败"
        exit 1
    fi
    echo "✅ 编译完成"
fi

# 启动服务
echo "🚀 启动服务..."
./main &
sleep 3

# 验证启动
if curl -s http://localhost:8000/hello > /dev/null 2>&1; then
    echo "✅ 服务启动成功！"
    echo ""
    echo "访问地址:"
    echo "  🌐 Swagger:   http://localhost:8000/swagger"
    echo "  📄 API 文档：http://localhost:8000/api.json"
    echo "  📝 日志目录：./logs"
    echo ""
    echo "按 Ctrl+C 停止服务"
else
    echo "❌ 服务启动失败，请检查日志"
    exit 1
fi
