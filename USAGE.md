# 抖音电商罗盘自动化爬虫 - 使用指南

## 📋 目录

1. [快速开始](#快速开始)
2. [环境准备](#环境准备)
3. [安装步骤](#安装步骤)
4. [启动爬虫](#启动爬虫)
5. [查看结果](#查看结果)
6. [常见问题](#常见问题)

---

## 🚀 快速开始

### 方式一：直接运行（推荐新手）

```bash
# 1. 启动 MySQL 数据库
docker-compose up -d mysql

# 2. 初始化数据库
make db-init
# 或者手动执行
mysql -h localhost -u root -p123456 < database/init.sql

# 3. 运行爬虫
make run
```

### 方式二：Docker 部署

```bash
# 一键启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f app
```

---

## 🛠️ 环境准备

### 必需软件

| 软件 | 版本要求 | 用途 |
|------|---------|------|
| Go | 1.21+ | 编译和运行 Go 应用 |
| Docker | 20.10+ | 容器化部署 |
| Docker Compose | 2.0+ | 多容器管理 |
| MySQL | 8.0+ | 数据存储（可用 Docker） |

### 检查环境

```bash
# 检查 Go 版本
go version

# 检查 Docker
docker --version
docker-compose --version

# 检查 MySQL（如果本地安装）
mysql --version
```

---

## 📦 安装步骤

### Step 1: 克隆项目

```bash
git clone <repository-url>
cd go-crawler
```

### Step 2: 安装依赖

```bash
go mod download
```

### Step 3: 配置数据库

**选项 A：使用 Docker（推荐）**

```bash
docker-compose up -d mysql
```

**选项 B：使用本地 MySQL**

1. 创建数据库：
```sql
CREATE DATABASE go_crawler CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

2. 导入表结构：
```bash
mysql -u root -p go_crawler < database/init.sql
```

3. 修改配置文件 `config.yaml`：
```yaml
database:
  default:
    link: "mysql://root:你的密码@tcp(localhost:3306)/go_crawler?charset=utf8mb4&parseTime=True&loc=Local"
```

### Step 4: 编译应用

```bash
# 本地编译
make build

# 或直接运行（无需编译）
make run
```

---

## ▶️ 启动爬虫

### 方式一：命令行启动

```bash
# 开发模式（热加载）
make run-hot

# 普通运行
make run

# 编译后运行
./bin/crawler
```

### 方式二：API 调用

启动服务后，访问 Swagger UI：
```
http://localhost:8000/swagger
```

发送 POST 请求到 `/crawl`：

```json
{
  "login_type": "qrcode",
  "target_url": "https://compass.jinritemai.com/shop/chance/product-rank"
}
```

或使用账号密码登录：

```json
{
  "login_type": "password",
  "username": "你的抖音账号",
  "password": "你的密码",
  "target_url": "https://compass.jinritemai.com/shop/chance/product-rank"
}
```

### 方式三：cURL 命令

```bash
# 扫码登录模式
curl -X POST http://localhost:8000/crawl \
  -H "Content-Type: application/json" \
  -d '{"login_type":"qrcode"}'

# 账号密码模式
curl -X POST http://localhost:8000/crawl \
  -H "Content-Type: application/json" \
  -d '{
    "login_type":"password",
    "username":"your_account",
    "password":"your_password"
  }'
```

---

## 📊 查看结果

### 数据文件位置

```
go-crawler/
├── logs/              # 日志文件
│   ├── crawler-20260331.log    # 爬虫日志
│   └── sql/           # SQL 执行日志
├── cookies/           # Cookie 文件
│   └── cookies.json   # 登录 Cookie
├── checkpoints/       # 断点续爬文件
└── exports/           # 导出数据
    ├── product_ranking_<task_id>.csv
    ├── product_ranking_<task_id>.xlsx
    └── video_analysis_<task_id>.csv
```

### 数据库查询

连接 MySQL：
```bash
mysql -u root -p123456 go_crawler
```

常用查询：

```sql
-- 查看最新的商品榜单数据
SELECT * FROM tiktok_ranking_list 
ORDER BY created_at DESC 
LIMIT 20;

-- 查看首次上榜的商品
SELECT * FROM tiktok_ranking_list 
WHERE first_list = 1 
ORDER BY rank;

-- 统计某个任务的抓取数量
SELECT task_id, COUNT(*) as total 
FROM tiktok_ranking_list 
GROUP BY task_id;

-- 查看视频分析数据
SELECT * FROM video_analysis_data 
ORDER BY created_at DESC 
LIMIT 20;
```

### 导出数据

系统会自动导出以下文件：

1. **CSV 格式**：`exports/product_ranking_<task_id>.csv`
   - 可用 Excel 打开
   - UTF-8 编码（带 BOM）

2. **Excel 格式**：`exports/product_ranking_<task_id>.xlsx`
   - 带格式和自动筛选
   - 更适合数据分析

3. **视频分析 CSV**：`exports/video_analysis_<task_id>.csv`
   - 包含详细的视频维度数据

---

## ⚙️ 配置说明

### config.yaml 关键配置

```yaml
# 服务器配置
server:
  address: ":8000"          # 监听端口

# 数据库配置
database:
  default:
    link: "mysql://..."     # 数据库连接串
    debug: true             # 开启 SQL 调试

# 爬虫配置
crawler:
  browser:
    headless: false         # 是否无头模式（调试建议关闭）
    timeout: 300            # 超时时间（秒）
  
  scrape:
    totalPages: 20          # 爬取页数
    itemsPerPage: 10        # 每页条数

# 日志配置
logger:
  level: "all"              # 日志级别
  rotateBackupLimit: 7      # 保留 7 天日志
```

---

## 🔧 常见问题

### Q1: 浏览器无法启动

**错误信息**：`启动浏览器失败：exec: "chrome": executable file not found`

**解决方案**：
```bash
# macOS
brew install --cask google-chrome

# Windows
# 下载安装：https://www.google.com/chrome/

# Linux
sudo apt-get install google-chrome-stable
```

### Q2: Cookie 失效

**现象**：每次运行都需要重新登录

**解决方案**：
1. 首次登录后，Cookie 会自动保存到 `cookies/cookies.json`
2. 检查文件是否存在且未过期
3. 如仍失效，删除旧 Cookie 重新登录：
```bash
rm cookies/cookies.json
make run
```

### Q3: 数据库连接失败

**错误信息**：`dial tcp 127.0.0.1:3306: connect: connection refused`

**解决方案**：
```bash
# 检查 MySQL 是否运行
docker ps | grep mysql

# 启动 MySQL
docker-compose up -d mysql

# 查看 MySQL 日志
docker-compose logs mysql
```

### Q4: 数据未入库

**现象**：程序运行成功，但数据库中没有数据

**排查步骤**：
1. 检查日志文件：`logs/crawler-*.log`
2. 查看 SQL 日志：`logs/sql/*.log`
3. 确认数据库表已创建：
```sql
SHOW TABLES FROM go_crawler;
```

### Q5: 中文乱码

**现象**：导出的 CSV/Excel 中文显示为乱码

**解决方案**：
- 系统已默认使用 UTF-8 BOM 编码
- 用 Excel 打开时选择"数据" → "从文本/CSV 导入" → 选择 UTF-8 编码

### Q6: 爬取中断

**现象**：网络异常导致爬取中断

**解决方案**：
- 系统支持断点续爬
- 重新启动即可从上次中断的页面继续
- 查看断点文件：`checkpoints/task_*.json`

---

## 📞 技术支持

如遇问题，请查看：

1. **日志文件**：`logs/crawler-*.log`
2. **SQL 日志**：`logs/sql/*.log`
3. **Docker 日志**：`docker-compose logs -f`

---

## 🎯 最佳实践

1. **首次运行建议使用扫码登录**，更安全便捷
2. **定期清理日志**：`make clean`
3. **备份重要数据**：`make db-backup`
4. **生产环境使用 Docker 部署**
5. **避免高频爬取**，建议设置合理的时间间隔

---

祝你使用愉快！🎉
