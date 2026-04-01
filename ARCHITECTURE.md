# 抖音电商罗盘自动化爬虫 - 架构设计文档

## 📐 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        用户层                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │  Web UI  │  │  API 调用 │  │  CLI 工具 │                  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘                  │
└───────┼─────────────┼─────────────┼─────────────────────────┘
        │             │             │
┌───────▼─────────────▼─────────────▼─────────────────────────┐
│                    API 网关层 (Controller)                   │
│  ┌─────────────────────────────────────────────────────┐    │
│  │           /crawl  - 爬虫任务接口                      │    │
│  │           /swagger - API 文档                         │    │
│  └─────────────────────────────────────────────────────┘    │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    业务逻辑层 (Logic)                        │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Crawler Logic                          │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │    │
│  │  │ 登录模块     │  │ 数据抓取模块 │  │ 数据处理模块 │ │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘ │    │
│  └─────────────────────────────────────────────────────┘    │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    服务层 (Service)                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ CookieManager│  │CheckpointMgr │  │ ExportManager│      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │ RankingList  │  │ VideoAnalysis│                        │
│  │   Service    │  │   Service    │                        │
│  └──────────────┘  └──────────────┘                        │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    数据访问层 (Model/DAO)                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ TikTokRanking│  │ VideoAnalysis│  │ CrawlerTask  │      │
│  │    List      │  │     Data     │  │     Log      │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    基础设施层                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   MySQL DB   │  │  File System │  │ Browser Auto │      │
│  │              │  │  (Cookie/    │  │  (go-rod)    │      │
│  │              │  │   Checkpoint)│  │              │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

---

## 🏗️ 技术栈

### 后端框架
- **GoFrame v2.10** - 企业级 Go 应用开发框架
- **go-rod v0.116** - 浏览器自动化工具（替代 Selenium）
- **MySQL 8.0** - 关系型数据库
- **Excelize v2.10** - Excel 文件处理库

### 核心组件

| 组件 | 技术选型 | 作用 |
|------|---------|------|
| 浏览器自动化 | go-rod | 模拟真人操作，绕过反爬 |
| Cookie 管理 | 自定义实现 | 持久化登录状态 |
| 断点续爬 | 文件系统 + JSON | 任务进度保存与恢复 |
| 数据导出 | Excelize / encoding/csv | CSV/Excel 导出 |
| 日志系统 | SimpleLogger | 文件轮转、分级记录 |

---

## 📁 目录结构

```
go-crawler/
├── api/                        # API 定义层
│   └── hello/
│       └── v1/
│           ├── hello.go        # Hello 接口定义
│           └── crawl.go        # 爬虫接口定义
│
├── internal/                   # 内部实现层
│   ├── cmd/                    # 命令入口
│   │   └── cmd.go              # 主命令定义
│   │
│   ├── controller/             # 控制器层
│   │   ├── crawler/            # 爬虫控制器
│   │   │   └── crawler.go      # V1 版本控制器
│   │   └── hello/              # Hello 控制器
│   │
│   ├── logic/                  # 业务逻辑层
│   │   └── crawler/            # 爬虫逻辑
│   │       ├── crawler.go      # 核心爬虫逻辑
│   │       └── logger.go       # 日志工具
│   │
│   ├── service/                # 服务层
│   │   ├── cookie_manager.go   # Cookie 管理
│   │   ├── checkpoint_manager.go # 断点管理
│   │   ├── export_manager.go   # 导出管理
│   │   ├── crawler.go          # 爬虫接口定义
│   │   ├── tiktok_ranking_list.go # 榜单服务
│   │   └── video_analysis.go   # 视频分析服务
│   │
│   ├── model/                  # 数据模型层
│   │   ├── tiktok_ranking_list.go # 榜单模型
│   │   ├── entity/             # 数据库实体
│   │   └── do/                 # 数据对象
│   │
│   ├── dao/                    # 数据访问层
│   │   └── .gitkeep
│   │
│   ├── consts/                 # 常量定义
│   │   └── .gitkeep
│   │
│   ├── logger/                 # 日志组件
│   │   └── .gitkeep
│   │
│   └── packed/                 # 打包资源
│       └── .gitkeep
│
├── database/                   # 数据库相关
│   └── init.sql                # 初始化脚本
│
├── logs/                       # 日志目录（运行时生成）
├── cookies/                    # Cookie 存储（运行时生成）
├── checkpoints/                # 断点文件（运行时生成）
├── exports/                    # 导出数据（运行时生成）
│
├── main.go                     # 程序入口
├── go.mod                      # Go 模块定义
├── go.sum                      # 依赖锁定
├── config.yaml                 # 配置文件
├── docker-compose.yaml         # Docker 编排
├── Dockerfile                  # 容器镜像
├── Makefile                    # 构建脚本
├── hack/                       # 构建脚本
│   ├── hack-cli.mk             # CLI 命令
│   └── hack.mk                 # 通用命令
│
├── README.md                   # 项目说明
├── USAGE.md                    # 使用指南
└── ARCHITECTURE.md             # 架构文档（本文件）
```

---

## 🔄 核心流程

### 1. 爬虫执行流程

```
开始
  │
  ▼
启动浏览器
  │
  ▼
加载 Cookie ────→ Cookie 有效？───是──→ 跳过登录
  │                    │
  否                   │
  │                    │
  ▼                    │
执行登录 ←────────────┘
(扫码/密码)
  │
  ▼
保存新 Cookie
  │
  ▼
导航到商品榜单页
  │
  ▼
点击"短视频榜"Tab
  │
  ▼
分页抓取 (20 页×10 条) ──→ 每页保存 ──→ 数据库
  │                           │
  │                           ▼
  │                    保存断点信息
  │
  ▼
去重处理
  │
  ▼
获取商品详情
(首次上架时间、最低价)
  │
  ▼
获取视频分析数据
  │
  ▼
保存视频数据到数据库
  │
  ▼
导出数据文件
(CSV/Excel)
  │
  ▼
标记任务完成
  │
  ▼
结束
```

### 2. 断点续爬流程

```
任务启动
  │
  ▼
检查断点文件
  │
  ├─ 无断点 ──→ 从第 1 页开始
  │
  ├─ 有断点 ──→ 读取进度
  │               │
  │               ▼
  │           状态检查
  │               │
  │               ├─ completed ──→ 直接返回
  │               │
  │               ├─ failed ──→ 询问是否恢复
  │               │
  │               └─ running ──→ 从断点页继续
  │
  ▼
正常爬取
  │
  ▼
每页保存后更新断点
  │
  ▼
完成后删除断点
```

### 3. 数据流转

```
网页 HTML
  │
  ▼ (go-rod 解析)
DOM 元素
  │
  ▼ (选择器提取)
原始文本
  │
  ▼ (清洗转换)
ProductInfo 结构体
  │
  ├─→ TikTokRankingListInput ──→ MySQL
  │
  ├─→ ProductDetail ───────────→ 合并到主数据
  │
  └─→ VideoAnalysis ───────────→ MySQL
                                      │
                                      ▼
                              CSV/Excel 导出
```

---

## 🔐 安全机制

### 1. Cookie 安全
- Cookie 仅保存在本地文件系统
- 每次会话自动刷新过期时间
- 支持手动清除 (`rm cookies/cookies.json`)

### 2. 防反爬策略
```go
// User-Agent 伪装
userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ..."

// 禁用自动化特征
l.Set("disable-blink-features", "AutomationControlled")

// 添加随机延迟
SlowMotion(100 * time.Millisecond)

// 多选择器容错
selectors := []string{".el-table__row", ".ant-table-row", ...}
```

### 3. 数据一致性
- 数据库事务保证批量插入的原子性
- 断点续爬防止数据丢失
- 每页保存模式降低单次失败影响

---

## 📊 数据模型

### 1. 商品榜单表 (tiktok_ranking_list)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| task_id | VARCHAR(64) | 任务 ID |
| rank | INT | 排名 |
| product_name | VARCHAR(255) | 商品名称 |
| shop_name | VARCHAR(255) | 店铺名称 |
| payment_amount | VARCHAR(32) | 支付金额 |
| click_count | VARCHAR(32) | 点击次数 |
| sales_volume | VARCHAR(32) | 成交件数 |
| delivery_video_num | VARCHAR(32) | 带货视频数 |
| first_shelf_time | VARCHAR(32) | 首次上架时间 |
| min_price | VARCHAR(32) | 最低到手价 |
| first_list | TINYINT | 是否首次上榜 |
| created_at | TIMESTAMP | 创建时间 |

### 2. 视频分析表 (video_analysis_data)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| task_id | VARCHAR(64) | 任务 ID |
| product_id | VARCHAR(64) | 商品 ID |
| product_name | VARCHAR(255) | 商品名称 |
| video_id | VARCHAR(64) | 视频 ID |
| video_desc | VARCHAR(512) | 视频描述 |
| payment_amount | VARCHAR(32) | 支付金额 |
| expose_user_count | VARCHAR(32) | 曝光人数 |
| click_user_count | VARCHAR(32) | 点击人数 |
| convert_user_count | VARCHAR(32) | 成交人数 |
| click_rate | VARCHAR(16) | 点击率 |
| click_convert_rate | VARCHAR(16) | 点击转化率 |
| expose_convert_rate | VARCHAR(16) | 曝光转化率 |
| order_count | VARCHAR(32) | 订单数 |
| sales_volume | VARCHAR(32) | 成交量 |
| created_at | TIMESTAMP | 创建时间 |

### 3. 任务日志表 (crawler_task_log)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| task_id | VARCHAR(64) | 任务 ID（唯一） |
| status | VARCHAR(32) | 状态 |
| current_step | INT | 当前步骤 |
| current_page | INT | 当前页码 |
| total_pages | INT | 总页数 |
| scraped_count | INT | 已抓取数 |
| start_time | TIMESTAMP | 开始时间 |
| end_time | TIMESTAMP | 结束时间 |
| error_message | TEXT | 错误信息 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

---

## ⚡ 性能优化

### 1. 并发控制
- 单浏览器实例，避免资源竞争
- 请求间隔控制（SlowMotion）
- 数据库连接池管理

### 2. 内存管理
- 流式数据处理，避免一次性加载
- 及时关闭页面和浏览器
- 日志文件轮转（保留 7 天）

### 3. 容错机制
- 多选择器兜底策略
- 超时重试（3 次）
- 异常捕获和日志记录

---

## 🔮 扩展方向

### 短期优化
1. 增加代理 IP 池支持
2. 实现分布式爬虫集群
3. 添加数据质量校验
4. 支持更多榜单类型（直播榜、达人榜）

### 长期规划
1. 混合架构：浏览器自动化 + HTTP 请求模拟
2. 智能调度系统：任务队列 + 优先级管理
3. 数据可视化 Dashboard
4. API 限流和权限控制

---

## 📞 联系方式

如有技术问题，请查阅：
- 使用指南：USAGE.md
- 日志文件：logs/crawler-*.log
- SQL 日志：logs/sql/*.log
