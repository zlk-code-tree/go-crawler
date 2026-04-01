#!/bin/bash

# =====================================================
# 抖音电商罗盘爬虫系统 - 快速启动脚本
# =====================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印横幅
print_banner() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║       抖音电商罗盘自动化爬虫系统                          ║${NC}"
    echo -e "${BLUE}║       TikTok Compass Automated Crawler System             ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# 打印步骤
print_step() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}▶ $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# 检查函数
check_command() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}✗ $2 未安装${NC}"
        return 1
    else
        echo -e "${GREEN}✓ $2 已安装${NC}"
        return 0
    fi
}

# 主函数
main() {
    print_banner
    
    # Step 1: 环境检查
    print_step "Step 1: 环境检查"
    
    check_command "go" "Go" || exit 1
    check_command "docker" "Docker" || echo -e "${YELLOW}! Docker 非必需，可跳过${NC}"
    
    # 检查 Go 版本
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "${GREEN}  Go 版本：$GO_VERSION${NC}"
    
    # Step 2: 创建必要目录
    print_step "Step 2: 创建必要目录"
    
    for dir in logs cookies checkpoints exports; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
            echo -e "${GREEN}✓ 创建目录：$dir/${NC}"
        else
            echo -e "${GREEN}✓ 目录已存在：$dir/${NC}"
        fi
    done
    
    # Step 3: 下载依赖
    print_step "Step 3: 下载 Go 依赖"
    
    echo "正在执行 go mod download..."
    go mod download
    echo -e "${GREEN}✓ 依赖下载完成${NC}"
    
    # Step 4: 数据库设置
    print_step "Step 4: 数据库配置"
    
    if command -v docker &> /dev/null && command -v docker-compose &> /dev/null; then
        echo -e "${GREEN}✓ 检测到 Docker，可以使用容器化部署${NC}"
        echo ""
        read -p "是否使用 Docker 启动 MySQL？(y/n): " -n 1 -r
        echo
        
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            # 检查 MySQL 容器是否已运行
            if docker ps | grep -q crawler_mysql; then
                echo -e "${GREEN}✓ MySQL 容器已在运行${NC}"
            else
                echo "正在启动 MySQL 容器..."
                docker-compose up -d mysql
                
                # 等待 MySQL 就绪
                echo "等待 MySQL 初始化（约 10 秒）..."
                sleep 10
                
                echo -e "${GREEN}✓ MySQL 容器启动成功${NC}"
            fi
            
            # 询问是否初始化数据库
            echo ""
            read -p "是否初始化数据库表结构？(y/n): " -n 1 -r
            echo
            
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                echo "正在导入数据库表结构..."
                docker exec -i crawler_mysql mysql -uroot -p123456 go_crawler < database/init.sql
                echo -e "${GREEN}✓ 数据库初始化完成${NC}"
            fi
        else
            echo -e "${YELLOW}! 请确保本地 MySQL 已启动并修改 config.yaml 配置${NC}"
        fi
    else
        echo -e "${YELLOW}! 未检测到 Docker，请使用本地 MySQL${NC}"
        echo "请确保："
        echo "  1. MySQL 服务已启动"
        echo "  2. 已创建数据库：go_crawler"
        echo "  3. 已导入表结构：mysql -u root -p go_crawler < database/init.sql"
        echo "  4. 已修改 config.yaml 中的数据库连接配置"
    fi
    
    # Step 5: 编译检查
    print_step "Step 5: 编译检查"
    
    echo "正在编译代码..."
    if go build -o bin/crawler main.go; then
        echo -e "${GREEN}✓ 编译成功${NC}"
    else
        echo -e "${RED}✗ 编译失败${NC}"
        exit 1
    fi
    
    # Step 6: 启动应用
    print_step "Step 6: 启动应用"
    
    echo ""
    echo "选择启动模式："
    echo "  1) 直接运行 (make run)"
    echo "  2) 热加载运行 (需要 air)"
    echo "  3) 后台运行"
    echo "  4) 仅编译不运行"
    echo ""
    read -p "请选择 (1-4): " -n 1 -r
    echo
    
    case $REPLY in
        1)
            echo -e "${GREEN}▶ 启动应用...${NC}"
            ./bin/crawler
            ;;
        2)
            echo -e "${GREEN}▶ 热加载启动...${NC}"
            if ! command -v air &> /dev/null; then
                echo "未安装 air，正在安装..."
                go install github.com/cosmtrek/air@latest
            fi
            air -c .air.toml
            ;;
        3)
            echo -e "${GREEN}▶ 后台运行...${NC}"
            nohup ./bin/crawler > logs/app.log 2>&1 &
            PID=$!
            echo -e "${GREEN}✓ 应用已在后台运行，PID: $PID${NC}"
            echo "查看日志：tail -f logs/app.log"
            echo "停止服务：kill $PID"
            ;;
        4)
            echo -e "${GREEN}✓ 编译完成，可手动运行：./bin/crawler${NC}"
            ;;
        *)
            echo -e "${RED}✗ 无效选择${NC}"
            exit 1
            ;;
    esac
    
    # 完成
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                    启动完成！                            ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "📊 服务信息："
    echo "  - API 地址：http://localhost:8000"
    echo "  - Swagger 文档：http://localhost:8000/swagger"
    echo "  - 日志文件：logs/crawler-*.log"
    echo ""
    echo "🚀 下一步操作："
    echo "  1. 访问 Swagger UI 查看 API 文档"
    echo "  2. 调用 /crawl 接口启动爬虫任务"
    echo "  3. 查看 logs/ 目录下的日志文件"
    echo "  4. 查看 exports/ 目录下的导出数据"
    echo ""
    echo "📖 更多帮助请查看：USAGE.md"
    echo ""
}

# 捕获错误
trap 'echo -e "\n${RED}✗ 脚本执行失败${NC}"; exit 1' ERR

# 执行主函数
main "$@"
