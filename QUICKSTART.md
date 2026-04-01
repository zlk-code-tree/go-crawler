# 🚀 抖音电商罗盘爬虫 - 5 分钟快速开始

## 前置要求

- ✅ Go 1.21+ 已安装
- ✅ Docker 和 Docker Compose（可选，用于数据库）
- ✅ 抖音电商罗盘账号

---

## Step 1: 下载项目 (30 秒)

```bash
# 如果已有项目，跳过此步
cd /Users/zhanglikun/WorkSpace/app/goCode/ly-project/go-crawler
```

---

## Step 2: 启动数据库 (1 分钟)

### 方式 A：使用 Docker（推荐）

```bash
# 启动 MySQL 容器
docker-compose up -d mysql

# 等待 10 秒让 MySQL 初始化
sleep 10

# 初始化数据库表结构
docker exec -i crawler_mysql mysql -uroot -p123456 go_crawler < database/init.sql
```

### 方式 B：使用本地 MySQL

```sql
-- 1. 创建数据库
CREATE DATABASE go_crawler CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 2. 导入表结构
mysql -u root -p go_crawler < database/init.sql
```

---

## Step 3: 运行爬虫 (2 分钟)

### 方式 A：一键启动（最简单）

```bash
./start.sh
```

### 方式 B：使用 Makefile

```bash
# 下载依赖
go mod download

# 直接运行
make run

# 或编译后运行
make build
./bin/crawler
```

### 方式 C：手动运行

```bash
# 下载依赖
go mod download

# 编译
go build -o bin/crawler main.go

# 运行
./bin/crawler
```

---

## Step 4: 启动爬虫任务 (1 分钟)

### 方式 A：访问 Swagger UI

打开浏览器访问：
```
http://localhost:8000/swagger
```

点击 `/crawl` 接口，发送请求：
```json
{
  "login_type": "qrcode"
}
```

### 方式 B：使用 cURL

```bash
curl -X POST http://localhost:8000/crawl \
  -H "Content-Type: application/json" \
  -d '{"login_type":"qrcode"}'
```

### 方式 C：扫码登录

1. 浏览器会自动打开抖音电商罗盘登录页面
2. 使用抖音 APP 扫描二维码
3. 登录成功后，爬虫自动开始抓取

---

## Step 5: 查看结果

### 查看日志

```bash
# 实时查看日志
tail -f logs/crawler-*.log

# 查看 SQL 执行日志
tail -f logs/sql/*.log
```

### 查看数据库

```bash
# 连接 MySQL
docker exec -it crawler_mysql mysql -uroot -p123456 go_crawler

# 查询最新数据
SELECT * FROM tiktok_ranking_list ORDER BY created_at DESC LIMIT 10;

# 查看首次上榜商品
SELECT * FROM tiktok_ranking_list WHERE first_list = 1 ORDER BY rank;
```

### 查看导出文件

```bash
# 列出导出的文件
ls -lh exports/

# 用 Excel 打开
open exports/product_ranking_*.xlsx
```

---

## 🎯 完成！

现在你已经成功：
- ✅ 启动了 MySQL 数据库
- ✅ 运行了爬虫服务
- ✅ 启动了爬取任务
- ✅ 查看了抓取结果

---

## ⚡ 常用命令速查

```bash
# 启动服务
make run

# 停止服务（后台运行时）
pkill crawler

# 查看日志
tail -f logs/crawler.log

# 清理缓存
rm cookies/cookies.json checkpoints/*.json

# 备份数据库
make db-backup

# 重启 MySQL
docker-compose restart mysql

# 查看 Docker 日志
docker-compose logs -f app
```

---

## 🔧 遇到问题？

### 问题 1: 端口被占用

**错误**: `bind: address already in use`

**解决**:
```bash
# 查找占用端口的进程
lsof -i :8000

# 杀死进程
kill -9 <PID>
```

### 问题 2: Cookie 失效

**现象**: 每次都需要重新登录

**解决**:
```bash
# 删除旧 Cookie
rm cookies/cookies.json

# 重新运行
make run
```

### 问题 3: 数据库连接失败

**错误**: `dial tcp 127.0.0.1:3306: connect: connection refused`

**解决**:
```bash
# 检查 MySQL 是否运行
docker ps | grep mysql

# 启动 MySQL
docker-compose up -d mysql
```

### 问题 4: 缺少 Chrome 浏览器

**错误**: `exec: "chrome": executable file not found`

**解决**:
```bash
# macOS
brew install --cask google-chrome

# Linux
sudo apt-get install google-chrome-stable
```

---

## 📖 更多帮助

- **详细文档**: [USAGE.md](USAGE.md)
- **架构说明**: [ARCHITECTURE.md](ARCHITECTURE.md)
- **项目总结**: [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md)

---

**祝你使用愉快！** 🎉

如有问题，请查看日志文件或联系技术支持。
