# Go 参数
GO      ?= go
GOPATH  := $(shell go env GOPATH)
GOOS    := $(shell go env GOOS)
GOARCH  := $(shell go env GOARCH)

# 编译输出目录
BINDIR  := $(ROOT_DIR)/bin
BUILDDIR := $(ROOT_DIR)/build

# 应用信息
VERSION ?= 1.0.0
APP_NAME := crawler
MAIN_FILE := main.go

# 默认目标
.PHONY: default
default: build

# ==================== 编译相关 ====================

# 本地编译
.PHONY: build
build:
	@echo "🔨 正在编译本地版本..."
	@$(GO) build -o $(BINDIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "✅ 编译完成：$(BINDIR)/$(APP_NAME)"

# Linux 版本（用于 Docker）
.PHONY: build-linux
build-linux:
	@echo "🔨 正在编译 Linux 版本..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o $(BINDIR)/linux_amd64/$(APP_NAME) $(MAIN_FILE)
	@echo "✅ Linux 版本编译完成：$(BINDIR)/linux_amd64/$(APP_NAME)"

# Windows 版本
.PHONY: build-windows
build-windows:
	@echo "🔨 正在编译 Windows 版本..."
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -o $(BINDIR)/windows_amd64/$(APP_NAME).exe $(MAIN_FILE)
	@echo "✅ Windows 版本编译完成：$(BINDIR)/windows_amd64/$(APP_NAME).exe"

# macOS 版本
.PHONY: build-darwin
build-darwin:
	@echo "🔨 正在编译 macOS 版本..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -o $(BINDIR)/darwin_amd64/$(APP_NAME) $(MAIN_FILE)
	@echo "✅ macOS 版本编译完成：$(BINDIR)/darwin_amd64/$(APP_NAME)"

# 编译所有平台
.PHONY: build-all
build-all: build build-linux build-windows build-darwin
	@echo "✅ 所有平台编译完成"

# ==================== 运行相关 ====================

# 运行应用
.PHONY: run
run:
	@echo "🚀 启动应用..."
	@$(GO) run $(MAIN_FILE)

# 热加载运行（需要 air）
.PHONY: run-hot
run-hot:
	@echo "🔥 热加载启动..."
	@if ! command -v air > /dev/null; then \
		echo "⚠️  未安装 air，正在安装..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	@air -c .air.toml

# ==================== 测试相关 ====================

# 运行测试
.PHONY: test
test:
	@echo "🧪 运行测试..."
	@$(GO) test -v ./...

# 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage:
	@echo "🧪 运行测试并生成覆盖率报告..."
	@$(GO) test -v -coverprofile=$(BUILDDIR)/coverage.out ./...
	@$(GO) tool cover -html=$(BUILDDIR)/coverage.out -o $(BUILDDIR)/coverage.html
	@echo "✅ 覆盖率报告已生成：$(BUILDDIR)/coverage.html"

# ==================== 代码质量 ====================

# 代码格式化
.PHONY: fmt
fmt:
	@echo "📝 格式化代码..."
	@$(GO) fmt ./...
	@gofmt -w -s .
	@echo "✅ 代码格式化完成"

# 代码检查
.PHONY: lint
lint:
	@echo "🔍 代码检查..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "⚠️  未安装 golangci-lint，正在安装..."; \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin latest; \
	fi
	@$(GOPATH)/bin/golangci-lint run ./...
	@echo "✅ 代码检查完成"

# 清理生成的文件
.PHONY: clean
clean:
	@echo "🗑️  清理生成的文件..."
	@rm -rf $(BINDIR) $(BUILDDIR)
	@rm -rf logs/* cookies/* checkpoints/* exports/*
	@echo "✅ 清理完成"

# ==================== Docker 相关 ====================

# 构建 Docker 镜像
.PHONY: docker-build
docker-build:
	@echo "🐳 构建 Docker 镜像..."
	@docker build -t $(DOCKER_NAME):$(VERSION) .
	@echo "✅ Docker 镜像构建完成：$(DOCKER_NAME):$(VERSION)"

# 启动 Docker Compose
.PHONY: docker-up
docker-up:
	@echo "🚀 启动 Docker Compose..."
	@docker-compose up -d
	@echo "✅ 服务已启动"

# 停止 Docker Compose
.PHONY: docker-down
docker-down:
	@echo "🛑 停止 Docker Compose..."
	@docker-compose down
	@echo "✅ 服务已停止"

# 查看 Docker 日志
.PHONY: docker-logs
docker-logs:
	@docker-compose logs -f

# ==================== 前端相关（Docker） ====================

# 启动前端开发服务器
.PHONY: web-dev
web-dev:
	@echo "🌐 启动前端开发服务器 (Docker)..."
	@docker-compose up -d web
	@echo "✅ 前端已启动: http://localhost:5555"

# 停止前端开发服务器
.PHONY: web-stop
web-stop:
	@echo "🛑 停止前端开发服务器..."
	@docker-compose stop web
	@echo "✅ 前端已停止"

# 查看前端日志
.PHONY: web-logs
web-logs:
	@docker-compose logs -f web

# 前端安装依赖
.PHONY: web-install
web-install:
	@echo "📦 安装前端依赖..."
	@docker-compose run --rm web sh -c "corepack enable && corepack prepare pnpm@latest --activate && pnpm install --ignore-scripts && pnpm run -r stub"
	@echo "✅ 前端依赖安装完成"

# 前端构建
.PHONY: web-build
web-build:
	@echo "📦 构建前端..."
	@docker-compose run --rm web sh -c "corepack enable && corepack prepare pnpm@latest --activate && pnpm install --ignore-scripts && pnpm run -r stub && pnpm build:ele"
	@echo "✅ 前端构建完成"

# 前端 shell（进入容器）
.PHONY: web-shell
web-shell:
	@docker-compose run --rm web sh

# ==================== 数据库相关 ====================

# 初始化数据库
.PHONY: db-init
db-init:
	@echo "🗄️  初始化数据库..."
	@mysql -h localhost -u root -p123456 < ./database/init.sql
	@echo "✅ 数据库初始化完成"

# 备份数据库
.PHONY: db-backup
db-backup:
	@echo "💾 备份数据库..."
	@mkdir -p $(BUILDDIR)/backup
	@mysqldump -h localhost -u root -p123456 go_crawler > $(BUILDDIR)/backup/go_crawler_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "✅ 数据库备份完成"

# ==================== 帮助信息 ====================

.PHONY: help
help:
	@echo ""
	@echo "📖 抖音电商罗盘爬虫系统 - Makefile 命令列表"
	@echo "================================================"
	@echo ""
	@echo "🔨 编译命令:"
	@echo "  make build          - 编译本地版本"
	@echo "  make build-linux    - 编译 Linux 版本"
	@echo "  make build-windows  - 编译 Windows 版本"
	@echo "  make build-darwin   - 编译 macOS 版本"
	@echo "  make build-all      - 编译所有平台"
	@echo ""
	@echo "🚀 运行命令:"
	@echo "  make run            - 运行应用"
	@echo "  make run-hot        - 热加载运行"
	@echo ""
	@echo "🧪 测试命令:"
	@echo "  make test           - 运行测试"
	@echo "  make test-coverage  - 生成测试覆盖率报告"
	@echo ""
	@echo "📝 代码质量:"
	@echo "  make fmt            - 格式化代码"
	@echo "  make lint           - 代码检查"
	@echo "  make clean          - 清理生成的文件"
	@echo ""
	@echo "🐳 Docker 命令:"
	@echo "  make docker-build   - 构建 Docker 镜像"
	@echo "  make docker-up      - 启动 Docker Compose"
	@echo "  make docker-down    - 停止 Docker Compose"
	@echo "  make docker-logs    - 查看 Docker 日志"
	@echo ""
	@echo "🌐 前端命令 (Docker):"
	@echo "  make web-dev        - 启动前端开发服务器"
	@echo "  make web-stop       - 停止前端开发服务器"
	@echo "  make web-logs       - 查看前端日志"
	@echo "  make web-install    - 安装前端依赖"
	@echo "  make web-build      - 构建前端"
	@echo "  make web-shell      - 进入前端容器"
	@echo ""
	@echo "🗄️  数据库命令:"
	@echo "  make db-init        - 初始化数据库"
	@echo "  make db-backup      - 备份数据库"
	@echo ""
	@echo "📖 其他:"
	@echo "  make help           - 显示帮助信息"
	@echo ""
