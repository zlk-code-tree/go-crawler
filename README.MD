# 电商罗盘自动化爬虫系统 (go-crawler)

基于 **Go + GoFrame v2 + go-rod** 构建的多平台电商数据采集与分析管理系统。系统最初以抖音电商罗盘商品榜单爬虫起步，已演进为覆盖 **抖音 / 快手 / 视频号** 三大电商平台、集成 **抖店开放平台 + 巨量千川** 官方 API、并具备组织架构管理、盈亏聚合与飞书推送能力的完整后端服务。

## 📋 功能特性

### 爬虫采集
- 浏览器自动化控制（go-rod），模拟真人操作抖音电商罗盘后台
- 登录状态检测（扫码 / 账号密码）+ Cookie 持久化与自动刷新
- 自动导航商品榜单、点击短视频榜 Tab、分页抓取（20 页 × 10 条 = 200 条）
- 商品详情抓取（首次上架时间、最低到手价）
- 视频分析数据抓取（核心指标 + 短视频列表）
- 断点续爬：任务进度 JSON 保存，异常中断可从断点页恢复
- WebSocket 实时日志推送与爬虫辅助通信

### 多平台订单管理
- 抖音订单：抖店开放平台 API 自动拉取 + Excel 导入
- 快手订单：Excel 批量导入
- 视频号订单：Excel 批量导入
- 精选联盟订单管理（含 buyin_id 回填）
- 达人订单匹配（订单与精选精选联盟数据匹配）
- 订单分析统计（小组 / 店铺维度）与 Excel 导出

### 千川广告数据接入
- 巨量千川 OAuth 授权（`/api/douyin/callback` 回调）
- 千川广告主 ID 自动解析（平台账户 → 千川广告主）
- 千川广告主日结报表拉取（消耗、展示、点击、成交等指标）
- 内部账号 ↔ 千川广告主映射管理
- 实时盈亏查询（今日消耗、佣金、盈亏）

### 财务盈亏聚合
- 每日凌晨自动聚合订单数据到 `finance_daily_stats`
- 盈亏公式：净利润 = 定价总额 − 货款 − 留底 − 退款成本 − 小店服务费 − 佣金扣除 − 广告消耗
- 按投手 / 部门 / 公司多维度统计
- 飞书 Webhook 盈亏日报推送（09:00 早报 / 18:00 晚报，含今日 / 7 日 / 15 日 / 30 日数据）

### 组织架构与权限
- 组织树：公司 → 部门 → 投手（投手 / 运营角色区分）
- 店铺借用账号归属规则管理
- RBAC 权限模型：用户 ↔ 角色 ↔ 权限 ↔ 菜单
- JWT 认证 + 飞书 OAuth2.0 登录
- 操作日志、登录日志自动记录
- 权限白名单机制

### 系统管理
- 用户 / 角色 / 权限 / 菜单 / 部门 CRUD
- 店铺管理（CRUD + Excel 导入导出）
- 商品管理（CRUD + Excel 导入）
- 定时任务管理（内置任务 + 自定义 cron 任务）
- 系统消息通知（WebSocket 实时推送）
- 数据导出（CSV UTF-8 BOM / Excel 带筛选）

## 🏗️ 技术架构

```
┌──────────────────────────────────────────────────────────────┐
│                         用户层                                │
│   Web UI   ·   API 调用   ·   Swagger 文档                    │
└──────────────────────────┬───────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────┐
│              API 网关层 (internal/controller)                 │
│   /api/*  RESTful 接口   ·   /ws/*  WebSocket 实时通信         │
└──────────────────────────┬───────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────┐
│                  中间件层 (internal/middleware)               │
│   CORS  ·  JWT 认证  ·  权限验证  ·  操作日志                  │
└──────────────────────────┬───────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────┐
│                  业务逻辑层 (internal/service)                │
│  订单 · 店铺 · 商品 · 组织架构 · 投手 · 财务聚合 · 爬虫        │
│  调度 · 导出 · 权限 · 飞书认证 · Webhook 推送                 │
│                                                              │
│  平台适配层 (internal/service/platform)                       │
│  ├── douyin       抖店开放平台 SDK 封装                       │
│  └── oceanengine  巨量千川 / 巨量引擎 SDK 封装                │
└──────────────────────────┬───────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────┐
│              数据访问层 (internal/model + g.DB())             │
└──────────────────────────┬───────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────┐
│                        基础设施层                             │
│  MySQL 8.0  ·  文件系统(Cookie/Checkpoint/Export)  ·  go-rod │
└──────────────────────────────────────────────────────────────┘
```

## 📁 项目结构

```
go-crawler/
├── api/                          # API 定义层
│   └── hello/v1/                 # 接口定义（含爬虫接口）
├── internal/
│   ├── cmd/                      # 命令入口（路由注册、定时任务注册）
│   │   └── cmd.go
│   ├── controller/               # 控制器层
│   │   ├── auth/                 # 认证（登录、飞书 OAuth）
│   │   ├── company/              # 公司管理
│   │   ├── crawler/              # 爬虫控制器
│   │   ├── crawler_service/      # 爬虫服务控制
│   │   ├── data/                 # 数据查询
│   │   ├── hello/                # Hello 测试接口
│   │   ├── menu/                 # 菜单管理
│   │   ├── permission/           # 权限管理
│   │   ├── scheduler/            # 调度管理
│   │   ├── system/               # 系统管理（订单/店铺/商品/用户/角色/投手/财务等）
│   │   ├── task/                 # 任务管理
│   │   ├── video/                # 视频分析
│   │   └── ws/                   # WebSocket（日志/辅助/通知）
│   ├── logic/crawler/            # 爬虫核心逻辑（go-rod 自动化）
│   ├── middleware/               # 中间件（CORS/JWT/操作日志）
│   ├── model/                    # 数据模型（含 entity / do）
│   ├── packed/                   # 打包资源
│   ├── pkg/                      # 内嵌第三方 SDK（抖店 SDK / 巨量引擎 SDK）
│   └── service/                  # 服务层
│       ├── platform/             # 平台适配（统一 PlatformAPI 接口）
│       │   ├── douyin/           # 抖店开放平台
│       │   ├── oceanengine/      # 巨量千川 / 巨量引擎
│       │   └── interface.go      # 平台工厂注册
│       ├── webhook/              # 飞书 / 企业微信 Webhook 推送
│       ├── auth.go               # 认证服务
│       ├── order.go              # 抖音订单服务
│       ├── kuaishou_order.go     # 快手订单服务
│       ├── shipinhao_order.go    # 视频号订单服务
│       ├── shop.go               # 店铺服务
│       ├── goods.go              # 商品服务
│       ├── finance_aggregate.go  # 财务日聚合
│       ├── finance_commission.go # 佣金计算
│       ├── finance_realtime.go   # 实时盈亏
│       ├── org_structure.go      # 组织架构
│       ├── trader.go             # 投手管理
│       ├── scheduler.go          # 调度服务
│       ├── cron_manager.go       # 定时任务管理器
│       ├── crawler.go            # 爬虫服务
│       ├── cookie_manager.go     # Cookie 管理
│       ├── checkpoint_manager.go # 断点管理
│       ├── export_manager.go     # 导出管理（CSV/Excel）
│       ├── feishu_auth.go        # 飞书认证
│       └── ...
├── database/                     # 数据库脚本（init.sql + migration_*.sql）
├── docs/                         # 文档（架构、使用、方案设计等）
├── manifest/                     # 部署配置
├── logs/                         # 日志目录（server / sql）
├── cookies/                      # Cookie 存储
├── checkpoints/                  # 断点文件
├── exports/                      # 导出文件
├── config.yaml                   # 应用配置
├── docker-compose.yaml           # Docker 编排
├── Dockerfile                    # 容器镜像
├── Makefile                      # 构建脚本
├── main.go                       # 程序入口
├── go.mod / go.sum
├── start.sh                      # 快速启动脚本
└── build.sh                      # Linux 交叉编译脚本
```

## 🚀 快速开始

### 1. 环境要求

- Go 1.25+
- MySQL 8.0+
- Chrome / Chromium（go-rod 会自动下载）
- （可选）Docker & Docker Compose

### 2. 安装依赖

```bash
cd go-crawler
go mod download
```

### 3. 配置数据库

编辑 `config.yaml`：

```yaml
database:
  default:
    type: "mysql"
    link: "用户名:密码@tcp(127.0.0.1:3306)/数据库名?charset=utf8mb4&parseTime=True&loc=Local"
    debug: true
    maxIdle: 10
    maxOpen: 100
    maxLifetime: "30s"
```

> 服务启动时会自动创建 `qianchuan_token`、`qianchuan_advertiser_daily_report`、`system_notification`、`cron_task`、`shop_account_attribution`、`account_qianchuan_mapping` 等关键表（`CREATE TABLE IF NOT EXISTS`）。其余历史表结构请按需执行 `database/migration_*.sql`。

### 4. 启动服务

```bash
# 方式一：快速启动脚本（含环境检查、依赖下载、数据库引导）
./start.sh

# 方式二：直接运行
go run main.go

# 方式三：编译后运行
go build -o main .
./main

# 方式四：Docker
docker-compose up -d
```

服务默认启动在 `http://localhost:8081`（端口由 `config.yaml` 的 `server.address` 控制）。

### 5. 访问文档

- Swagger UI：`http://localhost:8081/swagger`
- OpenAPI JSON：`http://localhost:8081/api.json`

## ⏰ 内置定时任务

服务启动时由 `CronManager` 统一注册并启动，详见 `internal/cmd/cmd.go`：

| 任务名 | Cron 表达式 | 说明 |
|--------|------------|------|
| `douyin_order_daily_pull` | `0 1 * * *` | 每日 01:00 拉取抖店订单 |
| `alliance_order_daily_pull` | `30 1 * * *` | 每日 01:30 拉取精选联盟订单 |
| `finance_daily_aggregate` | `0 2 * * *` | 每日 02:00 达人佣金日聚合 |
| `order_daily_aggregate` | `0 3 * * *` | 每日 03:00 抖音订单日聚合 |
| `qianchuan_daily_pull` | `37 4 * * *` | 每日 04:37 拉取千川报表（近 7 天） |
| `profit_morning_push` | `0 9 * * *` | 每日 09:00 盈亏早报推送 |
| `profit_evening_push` | `0 18 * * *` | 每日 18:00 盈亏晚报推送 |

## 📖 主要 API 接口

> 所有 `/api/*` 接口默认经过 JWT 认证与操作日志中间件；登录、OAuth 回调等公开接口除外。

### 认证与登录
```
POST /api/login                 账号密码登录
POST /api/feishu/login          飞书登录
GET  /api/feishu/callback       飞书回调
GET  /api/douyin/callback       抖店/千川 OAuth 回调
GET  /api/qianchuan/auth-url    获取千川授权链接
GET  /api/qianchuan/token-status 千川 Token 状态
```

### 爬虫与采集
```
POST /crawl                     启动爬虫任务（兼容旧路由）
GET  /api/ws/log/{taskId}       WebSocket 实时日志
GET  /api/ws/crawl-assist/{taskId} WebSocket 爬虫辅助
GET  /api/ws/notification       WebSocket 系统通知
```

### 订单与商品
```
GET  /api/order/list            抖音订单列表
GET  /api/order-stats           订单统计
POST /api/order-stats/backfill  订单统计回填
GET  /api/order/alliance-list   精选联盟订单
GET  /api/order/alliance-export 精选联盟导出
POST /api/order-match/run       达人订单匹配
GET  /api/order-match/export    匹配结果导出
GET  /api/order-analysis/group-export 订单分析分组导出
```

### 店铺与账号
```
GET  /api/store/list            店铺列表
POST /api/store/save            店铺保存
POST /api/store/import          店铺导入
GET  /api/store/export          店铺导出
GET  /api/shop-account-attribution/*  店铺借用账号归属规则
GET  /api/qianchuan-mapping/*   千川广告主映射管理
```

### 平台数据拉取控制
```
POST /api/douyin-poller/start        启动抖店轮询
POST /api/douyin-poller/stop         停止抖店轮询
GET  /api/douyin-poller/status       抖店轮询状态
POST /api/douyin-poller/trigger-order    手动触发订单拉取
POST /api/douyin-poller/trigger-alliance 手动触发联盟拉取
POST /api/qianchuan-poller/start     启动千川拉取
POST /api/qianchuan-poller/stop      停止千川拉取
GET  /api/qianchuan-poller/status    千川拉取状态
POST /api/qianchuan-poller/trigger   手动触发千川拉取
POST /api/qianchuan-poller/sync-advertisers 同步广告主
GET  /api/qianchuan-poller/daily-report 千川日报查询
```

### 财务盈亏
```
GET  /api/finance/today-cost     今日消耗/佣金/盈亏
GET  /api/finance/cost-ranking   消耗排行（trader/department/company）
GET  /api/finance/trader-export  投手盈亏导出
GET  /api/finance/account-export 账号盈亏导出
```

### 系统管理
```
/api/user/*      用户管理
/api/role/*      角色管理
/api/menu/*      菜单管理
/api/dept/*      部门管理
/api/profile/*   个人中心
/api/operation-log/*  操作日志
/api/login-log/*      登录日志
/api/notifications/*  系统消息
```

完整接口定义可在 Swagger UI 中查看。

## 📊 爬虫数据流程

```
1. 启动浏览器（go-rod，自动隐藏自动化特征）
   ↓
2. 加载 Cookie / 检测登录状态 → 未登录则扫码或账号密码登录
   ↓
3. 导航到商品榜单页面 → 点击短视频榜 Tab
   ↓
4. 分页抓取 20 页（200 条商品），每页抓完立即落库 + 更新断点
   ↓
5. 获取商品详情（首次上架时间、最低到手价）
   ↓
6. 获取视频分析数据（核心指标 + 短视频列表）
   ↓
7. 数据批量入库（MySQL，事务保证）
   ↓
8. 支持 CSV / Excel 导出
```

## 📝 抓取数据字段

### 商品榜单数据

| 字段 | 说明 |
|------|------|
| rank | 排名（优先解析真实值，回退公式计算） |
| product_name | 商品名称（跨页去重） |
| shop_name | 店铺名称 |
| payment_amount | 短视频用户支付金额 |
| click_count | 短视频点击次数 |
| sales_volume | 短视频成交件数 |
| delivery_video_num | 带货视频数 |
| first_shelf_time | 首次上架时间 |
| min_price | 最低到手价 |
| first_list | 首次上榜标识（DOM + 文本兜底检测） |

### 视频分析数据

| 字段 | 说明 |
|------|------|
| video_id / video_desc | 短视频信息 |
| payment_amount | 用户支付金额 |
| expose_user_count | 商品曝光人数 |
| click_user_count | 商品点击人数 |
| convert_user_count | 商品成交人数 |
| click_rate | 商品点击率（人数） |
| click_convert_rate | 点击-成交率（人数） |
| expose_convert_rate | 曝光-成交率（人数） |
| order_count | 商品成交订单数 |
| sales_volume | 商品成交件数 |

## 🔧 配置说明

`config.yaml` 主要配置段：

```yaml
server:                       # 服务端口、Swagger/OpenAPI 路径
  address: ":8081"

jwt:                          # JWT 密钥与过期时间
  secret: "xxx"
  expire: 86400

database:                     # MySQL 连接与连接池
  default: { type, link, maxIdle, maxOpen, maxLifetime }

logger:                       # 日志路径、轮转策略（100M / 保留7份 / 30天）

crawler:                      # 浏览器、抓取、Cookie、断点续爬配置
  browser: { headless, timeout, slowMotion, userAgent }
  scrape:  { totalPages, itemsPerPage, retryTimes, retryDelay }
  cookie:  { enabled, expireCheck }
  checkpoint: { enabled, autoSave, saveInterval }

export:                       # CSV / Excel 导出配置

permission:                   # 权限白名单（跳过权限验证的用户）

platform:                     # 第三方平台对接
  douyin:    { appKey, appSecret, baseUrl, timeout }   # 抖店开放平台
  kuaishou:  { appKey, appSecret, baseUrl }
  shipinhao: { appKey, appSecret, baseUrl }
  qianchuan: { appId, appSecret, pullDays, redirectUri, rid, materialAuth }  # 巨量千川

feishu:                       # 飞书 OAuth 登录配置
  { appId, appSecret, redirectUri }

task:                         # 任务调度监控
  feishuWebhook: ""           # 任务失败告警 webhook
  concurrency: 5              # 全局并发数

webhook:                      # 盈亏报告推送
  feishu: { enabled, webhook_url }
  push_schedule: { morning: "0 9 * * *", evening: "0 18 * * *" }
```

> ⚠️ **安全提醒**：`config.yaml` 中的数据库密码、JWT Secret、第三方 AppSecret 等敏感信息请通过环境变量或密钥管理服务注入，生产环境切勿直接提交到版本控制。

## 🐛 常见问题

### 1. 浏览器启动失败
- 确保系统内存充足（建议 4GB+）
- 检查 Chrome / Chromium 是否安装，或让 go-rod 自动下载
- 检查防火墙设置

### 2. 登录超时 / Cookie 失效
- 删除 `cookies/cookies.json` 重新登录
- 检查网络连接与账号状态

### 3. 数据抓取不完整
- 检查页面是否完全加载，适当增加 `time.Sleep`
- 检查 CSS 选择器是否需要更新（抖音罗盘页面改版）

### 4. 数据库连接失败
- 检查 MySQL 服务是否启动
- 验证 `config.yaml` 中账号密码与白名单
- 确认数据库已创建且字符集为 `utf8mb4`

### 5. 千川 Token 失效
- 访问 `/api/qianchuan/auth-url` 获取新授权链接重新授权
- 调用 `/api/qianchuan-poller/debug` 排查 Token 状态

### 6. 定时任务未执行
- 检查 `cron_task` 表状态
- 查看日志 `logs/server/{Y-m-d}.log` 中的 cron 注册信息

## 📝 日志查看

```bash
# 实时查看服务日志
tail -f logs/server/$(date +%Y-%m-%d).log

# 查看 SQL 执行日志
tail -f logs/sql/$(date +%Y-%m-%d).log

# 搜索错误日志
grep -i "error" logs/server/*.log
```

## 🔐 注意事项

1. **合规使用**：请确保爬取行为符合抖音电商罗盘的使用条款
2. **频率控制**：避免高频请求，设置合理的抓取间隔与并发数（`task.concurrency`）
3. **账号安全**：妥善保管登录账号与第三方 AppSecret，避免被封禁
4. **数据使用**：抓取的数据仅限内部经营分析使用，不得用于其他商业用途
5. **敏感信息**：生产环境务必通过环境变量覆盖 `config.yaml` 中的密钥

## 📦 部署

### Docker 部署

```bash
# 构建并启动（含 MySQL / app / web）
docker-compose up -d

# 仅启动 MySQL
docker-compose up -d mysql
```

> 注意：浏览器自动化（go-rod）需要在宿主机或带显示环境运行，不建议容器化运行爬虫模块。

### Linux 交叉编译

```bash
# 使用 build.sh
./build.sh
# 产物：myapp-v1.0.1-linux-amd64
```

## 📚 相关文档

更多设计细节请参阅 `docs/` 目录：

- `ARCHITECTURE.md` — 架构设计文档
- `PROJECT_STATE.md` — 项目状态与技术债务
- `QUICKSTART.md` — 5 分钟快速开始
- `千川消耗接入及盈亏推送方案.md` — 千川接入与 Webhook 推送方案
- `抖店订单与精选联盟及定时任务梳理.md` — 订单拉取与定时任务梳理
- `组织架构佣金统计方案.md` — 组织架构佣金统计方案
- `API接口文档-店铺管理和订单管理.md` — 店铺与订单接口文档

## 📄 License

MIT License

## 👥 联系方式

如有问题，请提交 Issue 或联系开发者。
