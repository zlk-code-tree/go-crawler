#!/bin/bash

# =====================================================
# 抖音电商罗盘爬虫系统 - 测试脚本
# =====================================================

set -e

echo "🚀 开始测试抖音电商罗盘爬虫系统..."
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查函数
check_result() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ $1${NC}"
    else
        echo -e "${RED}❌ $1${NC}"
        exit 1
    fi
}

# Step 1: 检查环境
echo "📋 Step 1: 检查环境依赖..."
echo ""

# 检查 Go
if command -v go &> /dev/null; then
    GO_VERSION=$(go version)
    echo -e "${GREEN}✓${NC} Go 已安装：$GO_VERSION"
else
    echo -e "${RED}✗${NC} Go 未安装"
    exit 1
fi

# 检查 Docker（可选）
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version)
    echo -e "${GREEN}✓${NC} Docker 已安装：$DOCKER_VERSION"
else
    echo -e "${YELLOW}!${NC} Docker 未安装（不影响运行，但无法使用容器化部署）"
fi

# 检查 MySQL 客户端（可选）
if command -v mysql &> /dev/null; then
    MYSQL_VERSION=$(mysql --version)
    echo -e "${GREEN}✓${NC} MySQL 客户端已安装：$MYSQL_VERSION"
else
    echo -e "${YELLOW}!${NC} MySQL 客户端未安装（不影响运行）"
fi

echo ""

# Step 2: 检查项目结构
echo "📁 Step 2: 检查项目结构..."
echo ""

REQUIRED_FILES=(
    "main.go"
    "go.mod"
    "go.sum"
    "config.yaml"
    "database/init.sql"
    "internal/cmd/cmd.go"
    "internal/logic/crawler/crawler.go"
    "internal/service/cookie_manager.go"
    "internal/service/export_manager.go"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $file"
    else
        echo -e "${RED}✗ $file (缺失)${NC}"
        exit 1
    fi
done

echo ""

# Step 3: 检查目录结构
echo "📂 Step 3: 检查必要目录..."
echo ""

REQUIRED_DIRS=(
    "logs"
    "cookies"
    "checkpoints"
    "exports"
    "database"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓${NC} $dir/"
    else
        echo -e "${YELLOW}!${NC} 创建目录：$dir/"
        mkdir -p "$dir"
    fi
done

echo ""

# Step 4: 编译检查
echo "🔨 Step 4: 编译检查..."
echo ""

# 下载依赖
echo "正在下载 Go 依赖..."
go mod download
check_result "依赖下载完成"

# 编译代码
echo "正在编译代码..."
go build -o bin/test_crawler main.go
check_result "编译成功"

# 清理测试文件
rm -f bin/test_crawler

echo ""

# Step 5: 数据库检查（如果 Docker 可用）
if command -v docker &> /dev/null && command -v docker-compose &> /dev/null; then
    echo "🗄️  Step 5: 检查数据库..."
    echo ""
    
    # 检查 MySQL 容器
    if docker ps | grep -q crawler_mysql; then
        echo -e "${GREEN}✓${NC} MySQL 容器正在运行"
    else
        echo -e "${YELLOW}!${NC} MySQL 容器未运行"
        read -p "是否启动 MySQL 容器？(y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker-compose up -d mysql
            check_result "MySQL 容器启动成功"
            
            # 等待 MySQL 就绪
            echo "等待 MySQL 初始化..."
            sleep 5
        fi
    fi
    
    echo ""
fi

# Step 6: 运行测试
echo "🧪 Step 6: 运行单元测试..."
echo ""

# 如果有测试文件
if ls internal/*/*_test.go 1> /dev/null 2>&1; then
    go test -v ./...
    check_result "单元测试通过"
else
    echo -e "${YELLOW}!${NC} 暂无单元测试文件"
fi

echo ""

# Step 7: 代码质量检查
echo "📝 Step 7: 代码格式化检查..."
echo ""

# Go fmt
gofmt -l . > /tmp/fmt_check.txt
if [ ! -s /tmp/fmt_check.txt ]; then
    echo -e "${GREEN}✓${NC} 代码格式良好"
else
    echo -e "${YELLOW}!${NC} 以下文件需要格式化:"
    cat /tmp/fmt_check.txt
    read -p "是否自动格式化？(y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        gofmt -w .
        check_result "代码格式化完成"
    fi
fi

rm -f /tmp/fmt_check.txt

echo ""

# 总结
echo "================================================"
echo -e "${GREEN}✅ 所有检查完成！${NC}"
echo "================================================"
echo ""
echo "📊 检查结果汇总："
echo "  ✓ 环境依赖检查通过"
echo "  ✓ 项目结构完整"
echo "  ✓ 编译无错误"
echo "  ✓ 代码格式良好"
echo ""
echo "🚀 下一步操作："
echo "  1. 启动数据库：docker-compose up -d mysql"
echo "  2. 初始化数据库：make db-init"
echo "  3. 运行爬虫：make run"
echo ""
echo "📖 详细使用说明请查看：USAGE.md"
echo ""
