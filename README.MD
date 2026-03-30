# 抖音电商罗盘爬虫服务

基于 Go + GoFrame + go-rod 开发的自动化爬虫服务，模拟真人操作抖音电商罗盘后台，抓取商品榜单和视频分析数据。

## 📋 功能特性

- ✅ 浏览器自动化控制（go-rod）
- ✅ 登录状态检测（支持扫码/账号密码登录）
- ✅ 自动导航到商品榜单页面
- ✅ 点击短视频榜 Tab
- ✅ 分页抓取 200 条商品数据（20 页×10 条）
- ✅ 获取商品详情（首次上架时间、最低到手价）
- ✅ 获取视频分析数据（核心指标 + 短视频列表）
- ✅ 数据入库存储（MySQL）
- ✅ 详细日志记录

## 🏗️ 技术架构

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  API 层      │ ──→ │  Controller  │ ──→ │   Logic     │
│ (api/hello) │     │ (controller) │     │ (crawler.go)│
└─────────────┘     └──────────────┘     └─────────────┘
                                              │
                     ┌──────────────┐         │
                     │   Database   │ ←───────┘
                     │   (MySQL)    │
                     └──────────────┘
```

## 📁 项目结构

```
go-crawler/
├── api/                      # API 定义
│   └── hello/
│       └── v1/
│           ├── crawler.go    # 爬虫 API 定义
│           └── hello.go      # Hello API 定义
├── internal/
│   ├── cmd/                  # 命令行入口
│   │   └── cmd.go
│   ├── controller/           # 控制器层
│   │   ├── crawler/
│   │   │   └── crawler.go    # 爬虫控制器
│   │   └── hello/
│   ├── logic/                # 业务逻辑层
│   │   └── crawler/
│   │       └── crawler.go    # 爬虫核心逻辑
│   ├── model/                # 数据模型
│   │   └── tiktok_ranking_list.go
│   ├── service/              # 服务层（数据访问）
│   │   ├── crawler.go
│   │   ├── tiktok_ranking_list.go
│   │   └── video_analysis.go
│   └── consts/               # 常量定义
├── manifest/
│   └── config/
│       └── config.yaml       # 配置文件
├── database/
│   └── schema.sql            # 数据库表结构
├── logs/                     # 日志目录
├── main.go                   # 程序入口
├── go.mod
└── README.md
```

## 🚀 快速开始

### 1. 环境要求

- Go 1.23+
- MySQL 5.7+
- Chrome/Chromium（go-rod 会自动下载）

### 2. 安装依赖

```bash
cd /Users/zhanglikun/WorkSpace/app/goCode/ly-project/go-crawler
go mod download
```

### 3. 配置数据库

编辑 `manifest/config/config.yaml`：

```yaml
database:
  default:
    type: "mysql"
    link: "admin:admin@tcp(127.0.0.1:3306)/easyadmin8?charset=utf8mb4&parseTime=True&loc=Local"
    debug: true
```

执行数据库初始化：

```bash
mysql -u admin -p < database/schema.sql
```

### 4. 启动服务

```bash
# 编译
go build -o main .

# 运行
./main
```

服务默认启动在 `http://localhost:8000`

### 5. 访问文档

- Swagger: http://localhost:8000/swagger
- API 文档：http://localhost:8000/api.json

## 📖 API 接口

### 1. Hello API（测试接口）

```bash
curl http://localhost:8000/hello
```

### 2. 爬虫 API

```bash
curl -X POST http://localhost:8000/crawler \
  -H "Content-Type: application/json" \
  -d '{
    "targetUrl": "https://compass.jinritemai.com/shop",
    "loginType": "qrcode",
    "username": "",
    "password": ""
  }'
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| targetUrl | string | 是 | 目标网站 URL |
| loginType | string | 否 | 登录方式：qrcode/password，默认 qrcode |
| username | string | 否 | 账号密码登录时的用户名 |
| password | string | 否 | 账号密码登录时的密码 |

**响应示例：**

```json
{
  "success": true,
  "message": "抓取成功",
  "data": [
    {
      "url": "https://compass.jinritemai.com/shop/chance/product-rank",
      "title": "抖音商品短视频榜_200 条商品_500 条视频",
      "content": "成功抓取 200 条商品数据，200 条详情数据，500 条视频分析数据",
      "product_count": 200,
      "video_count": 500,
      "products": [...],
      "product_details": [...],
      "video_analysis": [...]
    }
  ]
}
```

## 📊 数据流程

```
1. 启动浏览器
   ↓
2. 检测登录状态 → 未登录 → 等待用户登录（扫码/账号密码）
   ↓
3. 导航到商品榜单页面
   ↓
4. 点击短视频榜 Tab
   ↓
5. 分页抓取 20 页数据（200 条商品）
   ↓
6. 获取商品详情（首次上架时间、最低到手价）
   ↓
7. 获取视频分析数据
   ↓
8. 保存数据到数据库
   ↓
9. 返回结果
```

## 📝 抓取数据字段

### 商品榜单数据

| 字段 | 说明 |
|------|------|
| 排名 | 商品在榜单中的排名 |
| 商品名称 | 商品标题 |
| 店铺名称 | 销售商品的店铺 |
| 短视频用户支付金额 | 通过短视频带来的支付金额 |
| 短视频点击次数 | 短视频的总点击次数 |
| 短视频成交件数 | 通过短视频成交的商品件数 |
| 带货视频数 | 推广该商品的视频数量 |
| 首次上架时间 | 商品首次上架时间 |
| 最低到手价 | 商品最低价格 |

### 视频分析数据

| 字段 | 说明 |
|------|------|
| 短视频信息 | 视频标题/描述 |
| 用户支付金额 | 视频带来的支付金额 |
| 商品曝光人数 | 看到商品的用户数 |
| 商品点击人数 | 点击商品的用户数 |
| 商品成交人数 | 购买商品的用户数 |
| 商品点击率 (人数) | 点击人数/曝光人数 |
| 商品点击 - 成交率 (人数) | 成交人数/点击人数 |
| 商品曝光 - 成交率 (人数) | 成交人数/曝光人数 |
| 商品成交订单数 | 成交的订单数量 |
| 商品成交件数 | 成交的商品件数 |

## 🔧 配置说明

### manifest/config/config.yaml

```yaml
server:
  address:     ":8000"        # 服务端口
  openapiPath: "/api.json"    # API 文档路径
  swaggerPath: "/swagger"     # Swagger UI 路径

logger:
  path:    "./logs"           # 日志目录
  file:    "{Y-m-d}.log"      # 日志文件名
  level:   "all"              # 日志级别
  stdout:  true               # 输出到控制台

database:
  default:
    type:  "mysql"
    link:  "账号：密码@tcp(127.0.0.1:3306)/数据库名？charset=utf8mb4&parseTime=True&loc=Local"
    debug: true
```

## 🐛 常见问题

### 1. 浏览器启动失败

- 确保系统有足够的内存（建议 4GB+）
- 关闭其他占用内存的程序
- 检查防火墙设置

### 2. 登录超时

- 检查网络连接
- 尝试刷新页面重新登录
- 确保账号状态正常

### 3. 数据抓取不完整

- 检查页面是否完全加载
- 增加页面等待时间（修改代码中的 `time.Sleep`）
- 检查 CSS 选择器是否需要更新

### 4. 数据库连接失败

- 检查 MySQL 服务是否启动
- 验证账号密码是否正确
- 确认数据库已创建

## 📝 日志查看

日志文件位于 `logs/` 目录，按日期分割：

```bash
# 查看最新日志
tail -f logs/crawler-$(date +%Y%m%d).log

# 搜索错误日志
grep "ERROR" logs/crawler-*.log
```

## 🔐 注意事项

1. **合规使用**：请确保爬取行为符合抖音电商罗盘的使用条款
2. **频率控制**：避免高频请求，建议设置合理的抓取间隔
3. **账号安全**：妥善保管登录账号，避免被封禁
4. **数据使用**：抓取的数据仅限个人分析使用，不得用于商业用途

## 📄 License

MIT License

## 👥 联系方式

如有问题，请提交 Issue 或联系开发者。
