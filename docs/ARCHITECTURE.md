# 电商罗盘自动化爬虫系统 - 架构设计文档

## 系统架构

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
│  │           /api/*  - RESTful API 接口                 │    │
│  │           /ws/*   - WebSocket 实时通信                │    │
│  └─────────────────────────────────────────────────────┘    │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    中间件层 (Middleware)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   CORS   │  │  JWT认证  │  │ 权限验证  │  │ 操作日志  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    业务逻辑层 (Service)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  订单服务     │  │  店铺服务     │  │  商品服务     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  组织架构服务  │  │  投手服务     │  │  用户权限服务  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  爬虫服务     │  │  调度服务     │  │  导出服务     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    数据访问层 (Model/DAO)                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  OrderModel  │  │  ShopModel   │  │  GoodsModel  │      │
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

## 技术栈

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
| 日志系统 | GoFrame glog | 文件轮转、分级记录 |
| 认证授权 | JWT + RBAC | 用户认证和权限控制 |
| 实时通信 | WebSocket | 爬虫状态实时推送 |

---

## 目录结构

```
go-crawler/
├── api/                        # API 定义层
│   └── hello/
│       └── v1/
│           ├── hello.go        # Hello 接口定义
│           ├── crawl.go        # 爬虫接口定义
│           └── ...             # 其他接口定义
│
├── internal/                   # 内部实现层
│   ├── cmd/                    # 命令入口
│   │   └── cmd.go              # 主命令定义（路由注册）
│   │
│   ├── controller/             # 控制器层
│   │   ├── auth/               # 认证控制器
│   │   ├── system/             # 系统管理控制器
│   │   │   ├── order.go        # 订单管理
│   │   │   ├── shop.go         # 店铺管理
│   │   │   ├── goods.go        # 商品管理
│   │   │   ├── user.go         # 用户管理
│   │   │   ├── role.go         # 角色管理
│   │   │   ├── dept.go         # 部门管理
│   │   │   ├── order_analysis.go # 订单分析
│   │   │   ├── org_structure.go  # 组织架构
│   │   │   ├── trader.go       # 投手管理
│   │   │   └── ...
│   │   ├── crawler/            # 爬虫控制器
│   │   ├── scheduler/          # 调度控制器
│   │   ├── ws/                 # WebSocket 控制器
│   │   └── ...
│   │
│   ├── service/                # 服务层（业务逻辑）
│   │   ├── auth.go             # 认证服务
│   │   ├── order.go            # 订单服务
│   │   ├── shop.go             # 店铺服务
│   │   ├── goods.go            # 商品服务
│   │   ├── org_structure.go    # 组织架构服务
│   │   ├── trader.go           # 投手服务
│   │   ├── scheduler.go        # 调度服务
│   │   ├── crawler.go          # 爬虫服务
│   │   └── ...
│   │
│   ├── model/                  # 数据模型层
│   │   ├── order.go            # 订单模型
│   │   ├── shop.go             # 店铺模型
│   │   ├── goods.go            # 商品模型
│   │   ├── org_structure.go    # 组织架构模型
│   │   ├── trader.go           # 投手模型
│   │   └── ...
│   │
│   ├── logic/                  # 业务逻辑层（爬虫）
│   │   └── crawler/            # 爬虫逻辑
│   │       ├── crawler.go      # 核心爬虫逻辑
│   │       └── logger.go       # 日志工具
│   │
│   ├── middleware/             # 中间件
│   │   ├── middleware.go       # 认证、权限中间件
│   │   └── operation_log.go   # 操作日志中间件
│   │
│   ├── dao/                    # 数据访问层（待完善）
│   ├── consts/                 # 常量定义
│   ├── logger/                 # 日志组件
│   └── packed/                 # 打包资源
│
├── database/                   # 数据库相关
│   ├── init.sql                # 初始化脚本
│   ├── rbac_schema.sql         # RBAC 权限表结构
│   └── migration_*.sql         # 数据库迁移脚本
│
├── docs/                       # 文档目录
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
└── Makefile                    # 构建脚本
```

---

## 核心模块说明

### 1. 爬虫模块 (Crawler)

**职责**: 自动化抓取电商平台数据

**核心组件**:
- `internal/logic/crawler/` - 爬虫核心逻辑
- `internal/service/crawler.go` - 爬虫服务接口
- `internal/service/cookie_manager.go` - Cookie 管理
- `internal/service/checkpoint_manager.go` - 断点管理

**工作流程**:
1. 启动浏览器实例（go-rod）
2. 加载已保存的 Cookie 或执行登录
3. 拦截 API 请求获取数据
4. 解析并存储到数据库
5. 支持断点续爬

### 2. 订单管理模块 (Order)

**职责**: 管理多平台订单数据

**核心组件**:
- `internal/service/order.go` - 抖音订单服务
- `internal/service/kuaishou_order.go` - 快手订单服务
- `internal/service/shipinhao_order.go` - 视频号订单服务
- `internal/service/order_analysis.go` - 订单分析服务

**支持平台**:
- 抖音电商
- 快手电商
- 视频号电商

### 3. 组织架构模块 (OrgStructure)

**职责**: 管理公司、部门、投手的层级关系

**核心组件**:
- `internal/service/org_structure.go` - 组织架构服务
- `internal/model/org_structure.go` - 组织架构模型

**层级结构**:
```
公司 (Company)
  └── 部门 (Department)
        └── 投手 (Trader)
```

**ID 编码规则**:
- Company: 1 ~ 999,999
- Department: 1,000,000 + 原 ID
- Trader: 2,000,000 + 原 ID

### 4. 权限管理模块 (Permission)

**职责**: RBAC 权限控制

**核心组件**:
- `internal/service/auth.go` - 认证服务
- `internal/service/role.go` - 角色服务
- `internal/service/permission.go` - 权限服务
- `internal/middleware/middleware.go` - 权限中间件

**权限模型**:
- 用户 (User)
- 角色 (Role)
- 权限 (Permission)
- 菜单 (Menu)

### 5. 调度模块 (Scheduler)

**职责**: 定时任务调度

**核心组件**:
- `internal/service/scheduler.go` - 调度服务

**任务类型**:
- 商品榜单抓取
- 视频池抓取

---

## 数据流

### 1. 爬虫数据流

```
电商平台 → go-rod 拦截 → 数据解析 → MySQL 存储
                    ↓
              Cookie 持久化
                    ↓
              断点保存
```

### 2. API 数据流

```
前端请求 → Controller → Service → Model → MySQL
                ↓
           Middleware (JWT/权限/日志)
```

### 3. 导出数据流

```
MySQL 查询 → 数据处理 → Excelize 生成 → 文件下载
```

---

## API 结构

### 认证接口

```
POST /api/login              # 用户登录
POST /api/feishu/login       # 飞书登录
GET  /api/feishu/callback    # 飞书回调
```

### 系统管理接口

```
# 用户管理
GET    /api/system/user/list
POST   /api/system/user/save
PUT    /api/system/user/save
POST   /api/system/user/delete

# 角色管理
GET    /api/system/role/list
POST   /api/system/role/save
POST   /api/system/role/delete

# 部门管理
GET    /api/system/dept/list
POST   /api/system/dept/save
PUT    /api/system/dept/update
DELETE /api/system/dept/delete

# 店铺管理
GET    /api/system/shop/list
POST   /api/system/shop/save
POST   /api/system/shop/delete
POST   /api/system/shop/import

# 订单管理
GET    /api/system/order/list
GET    /api/system/order/detail
GET    /api/system/order/export

# 商品管理
GET    /api/system/goods/list
POST   /api/system/goods/save
POST   /api/system/goods/import

# 订单分析
GET    /api/order-analysis/group-stats
GET    /api/order-analysis/shop-stats
GET    /api/order-analysis/overview
GET    /api/order-analysis/group-export

# 组织架构
GET    /api/system/org/tree
GET    /api/system/org/list
POST   /api/system/org/save
DELETE /api/system/org/delete

# 投手管理
GET    /api/system/trader/list
POST   /api/system/trader/save
POST   /api/system/trader/bind-shop
POST   /api/system/trader/unbind-shop
```

### 爬虫接口

```
POST /api/crawl/start        # 启动爬虫任务
POST /api/crawl/stop         # 停止爬虫任务
GET  /api/crawl/status       # 获取爬虫状态
```

### WebSocket 接口

```
WS /api/ws/log/{taskId}          # 爬虫日志实时推送
WS /api/ws/crawl-assist/{taskId} # 爬虫辅助通信
```

---

## 数据库设计

### 核心表结构

| 表名 | 说明 | 主要字段 |
|------|------|---------|
| order_info | 抖音订单 | main_order_no, shop_name, product_name |
| kuaishou_orders | 快手订单 | order_no, shop_name, product_name |
| shipinhao_orders | 视频号订单 | order_no, shop_name, product_name |
| shop | 店铺/账号 | platform, uid, account_name, operator |
| goods | 商品 | goods_code, goods_name, base_price |
| trader | 投手 | name, company_id, group_id, role |
| sys_user | 用户 | username, password, status |
| sys_role | 角色 | name, code, status |
| sys_permission | 权限 | name, code, status |
| sys_menu | 菜单 | name, path, component, parent_id |
| sys_company | 公司 | name, code, parent_id |
| sys_department | 部门 | name, company_id, parent_id |

### 关联关系

- 用户 ↔ 角色: 多对多 (sys_user_role)
- 角色 ↔ 权限: 多对多 (sys_role_permission)
- 角色 ↔ 菜单: 多对多 (sys_role_menu)
- 投手 ↔ 店铺: 多对多 (trader_shop_relation)
- 店铺 ↔ 组织: 多对一 (shop.company_id, shop.department_id)

---

## 部署架构

### 开发环境

```
本地开发 → config.yaml → MySQL (远程)
                    ↓
              go-rod (本地浏览器)
```

### 生产环境

```
Docker 容器 → docker-compose.yaml
                    ↓
              MySQL (阿里云 RDS)
                    ↓
              文件存储 (Cookie/Checkpoint/Export)
```

---

## 性能优化

### 1. 数据库优化

- 连接池配置: maxIdle=10, maxOpen=100
- 索引优化: 关键查询字段添加索引
- 分页查询: 避免全表扫描

### 2. 爬虫优化

- 并发控制: 避免过度请求
- 断点续爬: 避免重复抓取
- Cookie 复用: 减少登录次数

### 3. 导出优化

- 流式写入: 避免内存溢出
- 分批查询: 大数据量分批处理

---

## 安全设计

### 1. 认证安全

- JWT Token: 24小时过期
- 密码加密: bcrypt 哈希
- 飞书 OAuth2.0 集成

### 2. 权限安全

- RBAC 模型: 细粒度权限控制
- 白名单机制: 特定用户跳过权限验证
- 操作日志: 记录关键操作

### 3. 数据安全

- SQL 参数化: 防止 SQL 注入
- CORS 配置: 跨域访问控制
- 敏感信息: 配置文件不提交到版本控制

---

## 监控与日志

### 日志分类

| 日志类型 | 路径 | 说明 |
|---------|------|------|
| 应用日志 | logs/server/ | 服务运行日志 |
| SQL 日志 | logs/sql/ | 数据库查询日志 |
| 爬虫日志 | logs/ | 爬虫运行日志 |

### 日志配置

```yaml
logger:
  path: "./logs/server"
  file: "{Y-m-d}.log"
  level: "all"
  rotateSize: "100M"
  rotateBackupLimit: 7
  rotateBackupExpire: "30d"
```
