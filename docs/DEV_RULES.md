# 电商罗盘自动化爬虫系统 - 开发规范

## 1. 代码规范

### 1.1 命名规范

#### 包名
- 使用小写单词，避免下划线
- 简短且有意义
- 示例: `service`, `model`, `controller`

#### 结构体
- 使用大驼峰命名法 (PascalCase)
- 示例: `OrderService`, `ShopModel`, `UserController`

#### 函数/方法
- 使用大驼峰命名法 (PascalCase)
- 动词开头，语义清晰
- 示例: `GetList`, `SaveOrder`, `DeleteById`

#### 变量
- 使用小驼峰命名法 (camelCase)
- 避免缩写，除非是约定俗成的
- 示例: `orderList`, `pageSize`, `totalCount`

#### 常量
- 使用大写下划线命名法 (UPPER_SNAKE_CASE)
- 示例: `MAX_PAGE_SIZE`, `DEFAULT_TIMEOUT`

### 1.2 文件组织

#### 目录结构
```
internal/
├── controller/     # 控制器层（处理请求、返回响应）
├── service/        # 服务层（业务逻辑）
├── model/          # 模型层（数据结构定义）
├── logic/          # 逻辑层（复杂业务逻辑，如爬虫）
├── middleware/     # 中间件
├── dao/            # 数据访问层（待完善）
├── consts/         # 常量定义
└── logger/         # 日志组件
```

#### 文件命名
- 使用小写下划线命名法 (snake_case.go)
- 示例: `order_service.go`, `shop_model.go`

### 1.3 注释规范

#### 包注释
```go
// Package service 提供业务逻辑层实现
// 包含订单、店铺、商品等核心业务服务
package service
```

#### 结构体注释
```go
// OrderService 订单服务
// 提供订单的增删改查、统计分析等功能
type OrderService struct{}
```

#### 函数注释
```go
// GetList 获取订单列表
// 支持分页、关键词搜索、时间范围筛选
// 返回订单列表、总数、错误信息
func (s *OrderService) GetList(ctx context.Context, req *model.OrderListReq) ([]model.Order, int, error) {
    // ...
}
```

---

## 2. 架构规范

### 2.1 分层原则

#### Controller 层
- **职责**: 接收请求、参数校验、调用 Service、返回响应
- **禁止**: 直接操作数据库、包含业务逻辑
- **示例**:
```go
func (c *OrderController) List(ctx context.Context, req *v1.OrderListReq) (res *v1.OrderListRes, err error) {
    orderService := service.NewOrderService()
    list, total, err := orderService.List(ctx, req.Page, req.PageSize, req.Keyword)
    if err != nil {
        return nil, gerror.New("查询失败: " + err.Error())
    }
    return &v1.OrderListRes{List: list, Total: total}, nil
}
```

#### Service 层
- **职责**: 业务逻辑处理、数据组装、事务管理
- **禁止**: 直接处理 HTTP 请求/响应
- **示例**:
```go
func (s *OrderService) List(ctx context.Context, page, pageSize int, keyword string) ([]g.Map, int64, error) {
    // 业务逻辑处理
    query := "SELECT * FROM order_info WHERE 1=1"
    // ...
    return orders, total, nil
}
```

#### Model 层
- **职责**: 数据结构定义、数据验证规则
- **禁止**: 包含业务逻辑
- **示例**:
```go
type OrderInfo struct {
    Id          int64     `json:"id"`
    ShopName    string    `json:"shopName"`
    OrderNo     string    `json:"orderNo"`
    // ...
}
```

### 2.2 依赖注入

#### Service 实例化
```go
// 推荐：每次调用创建新实例
orderService := service.NewOrderService()

// 避免：全局单例（除非有特殊需求）
var orderService = service.NewOrderService()
```

#### 依赖传递
```go
// 推荐：通过 Context 传递必要信息
func (s *OrderService) List(ctx context.Context, ...) {
    userId := gconv.Int(ctx.Value("userId"))
    // ...
}

// 避免：直接传递 Request 对象
func (s *OrderService) List(req *ghttp.Request) {
    // ...
}
```

---

## 3. 数据库规范

### 3.1 SQL 规范

#### 参数化查询
```go
// 推荐：使用参数化查询
query := "SELECT * FROM order_info WHERE shop_name = ? AND status = ?"
result, err := g.DB().GetAll(ctx, query, shopName, status)

// 避免：字符串拼接
query := fmt.Sprintf("SELECT * FROM order_info WHERE shop_name = '%s'", shopName)
```

#### 分页查询
```go
// 推荐：使用 LIMIT 和 OFFSET
query := "SELECT * FROM order_info ORDER BY id DESC LIMIT ? OFFSET ?"
params := []interface{}{pageSize, (page - 1) * pageSize}
result, err := g.DB().GetAll(ctx, query, params...)
```

#### 总数查询
```go
// 推荐：使用 COUNT 查询总数
totalQuery := "SELECT COUNT(*) as total FROM order_info WHERE 1=1"
resultTotal, err := g.DB().GetOne(ctx, totalQuery, params...)
total := resultTotal["total"].Int64()
```

### 3.2 事务处理

```go
// 推荐：使用 GoFrame 事务
err := g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
    // 事务内的操作
    _, err := tx.Insert("order_info", orderData)
    if err != nil {
        return err
    }
    // ...
    return nil
})
```

### 3.3 字段映射

```go
// 推荐：使用结构体标签映射
type OrderInfo struct {
    Id       int64  `json:"id"`
    ShopName string `json:"shopName" orm:"shop_name"`
}

// 或使用手动映射
orders[i] = g.Map{
    "id":       row["id"],
    "shopName": row["shop_name"],
}
```

---

## 4. API 规范

### 4.1 请求格式

#### RESTful 风格
```
GET    /api/resource          # 查询列表
GET    /api/resource/{id}     # 查询详情
POST   /api/resource          # 创建
PUT    /api/resource/{id}     # 更新
DELETE /api/resource/{id}     # 删除
```

#### 请求参数
```go
type OrderListReq struct {
    Page     int    `json:"page" v:"required|min:1#页码不能为空|页码最小为1"`
    PageSize int    `json:"pageSize" v:"required|min:1|max:100#每页数量不能为空|每页数量最小为1|每页数量最大为100"`
    Keyword  string `json:"keyword"`
}
```

### 4.2 响应格式

#### 成功响应
```go
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}
```

#### 错误响应
```go
// 推荐：使用 GoFrame 错误
return nil, gerror.New("查询失败")

// 或使用自定义错误码
return nil, gerror.NewCode(gcode.New(10001, "订单不存在", nil))
```

### 4.3 分页响应
```go
type PageResponse struct {
    List  interface{} `json:"list"`
    Total int64       `json:"total"`
    Page  int         `json:"page"`
    PageSize int      `json:"pageSize"`
}
```

---

## 5. 错误处理规范

### 5.1 错误类型

#### 业务错误
```go
// 推荐：使用 gerror
return nil, gerror.New("订单不存在")

// 带错误码
return nil, gerror.NewCode(gcode.New(10001, "订单不存在", nil))
```

#### 系统错误
```go
// 推荐：包装原始错误
if err != nil {
    return nil, gerror.Wrap(err, "查询订单失败")
}
```

### 5.2 错误日志

```go
// 推荐：记录错误上下文
if err != nil {
    g.Log().Errorf(ctx, "查询订单失败: %v, req: %+v", err, req)
    return nil, gerror.New("查询失败")
}

// 避免：只记录错误，不记录上下文
if err != nil {
    g.Log().Error(err)
    return nil, err
}
```

---

## 6. 日志规范

### 6.1 日志级别

- **DEBUG**: 调试信息，开发环境使用
- **INFO**: 一般信息，记录关键操作
- **WARNING**: 警告信息，不影响运行但需关注
- **ERROR**: 错误信息，需要修复

### 6.2 日志格式

```go
// 推荐：结构化日志
g.Log().Infof(ctx, "订单查询成功, userId: %d, count: %d", userId, count)

// 推荐：错误日志带上下文
g.Log().Errorf(ctx, "查询订单失败, req: %+v, err: %v", req, err)

// 避免：无意义的日志
g.Log().Info("查询成功")
```

### 6.3 日志分类

| 日志类型 | 路径 | 说明 |
|---------|------|------|
| 应用日志 | logs/server/ | 服务运行日志 |
| SQL 日志 | logs/sql/ | 数据库查询日志 |
| 爬虫日志 | logs/ | 爬虫运行日志 |

---

## 7. 安全规范

### 7.1 敏感信息处理

#### 配置文件
- 敏感信息使用环境变量
- 配置文件不提交到版本控制
- 使用 `.gitignore` 忽略敏感文件

#### 数据库
- 使用参数化查询，避免 SQL 注入
- 敏感字段加密存储
- 定期备份数据

### 7.2 认证授权

#### JWT Token
- 设置合理的过期时间
- 使用强随机密钥
- 支持 Token 刷新

#### 权限控制
- 遵循最小权限原则
- 细粒度权限控制
- 记录操作日志

### 7.3 输入验证

```go
// 推荐：使用 GoFrame 验证器
type OrderSaveReq struct {
    ShopName string `json:"shopName" v:"required#店铺名称不能为空"`
    OrderNo  string `json:"orderNo" v:"required|length:10,20#订单号不能为空|订单号长度10-20位"`
}

// 推荐：手动验证
if req.Page < 1 {
    return nil, gerror.New("页码不能小于1")
}
```

---

## 8. 测试规范

### 8.1 单元测试

#### 文件命名
```
order_service_test.go
shop_model_test.go
```

#### 测试函数
```go
func TestOrderService_GetList(t *testing.T) {
    // Arrange
    service := NewOrderService()
    ctx := context.Background()

    // Act
    list, total, err := service.GetList(ctx, 1, 10, "")

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, list)
    assert.GreaterOrEqual(t, total, int64(0))
}
```

### 8.2 接口测试

```go
func TestOrderController_List(t *testing.T) {
    // 使用 httptest 测试接口
    // ...
}
```

### 8.3 测试覆盖率

- 目标覆盖率 > 60%
- 核心业务逻辑覆盖率 > 80%

---

## 9. Git 规范

### 9.1 分支管理

- **master**: 主分支，稳定版本
- **develop**: 开发分支
- **feature/**: 功能分支
- **bugfix/**: 修复分支
- **release/**: 发布分支

### 9.2 提交规范

#### 提交格式
```
<type>(<scope>): <subject>

<body>

<footer>
```

#### Type 类型
- **feat**: 新功能
- **fix**: 修复 Bug
- **docs**: 文档更新
- **style**: 代码格式调整
- **refactor**: 重构
- **test**: 测试相关
- **chore**: 构建/工具相关

#### 示例
```
feat(order): 新增订单导出功能

- 支持 Excel 格式导出
- 支持按时间范围筛选
- 支持按店铺筛选

Closes #123
```

### 9.3 代码审查

- 所有代码必须经过 Code Review
- 至少一人审核通过
- 关注代码质量、安全性、性能

---

## 10. 部署规范

### 10.1 环境配置

#### 开发环境
- 使用本地数据库
- 开启调试模式
- 详细日志输出

#### 生产环境
- 使用云数据库
- 关闭调试模式
- 精简日志输出

### 10.2 部署流程

1. 代码合并到 master
2. 运行测试套件
3. 构建 Docker 镜像
4. 部署到生产环境
5. 验证功能正常

### 10.3 监控告警

- 应用健康检查
- 错误率监控
- 性能指标监控
- 日志告警

---

## 11. 文档规范

### 11.1 文档类型

| 文档 | 位置 | 说明 |
|------|------|------|
| 架构设计 | docs/ARCHITECTURE.md | 系统架构说明 |
| 项目状态 | docs/PROJECT_STATE.md | 当前进度和问题 |
| 开发规范 | docs/DEV_RULES.md | 本文档 |
| API 文档 | docs/API*.md | 接口说明 |
| 数据库文档 | database/*.sql | 表结构说明 |

### 11.2 文档更新

- 代码变更时同步更新文档
- 重大功能变更必须更新架构文档
- 定期审查文档准确性

---

## 12. 性能规范

### 12.1 数据库优化

- 合理使用索引
- 避免 N+1 查询
- 大数据量分批处理

### 12.2 缓存策略

- 热点数据缓存
- 合理设置过期时间
- 缓存一致性保证

### 12.3 并发控制

- 避免过度并发
- 使用连接池
- 限流保护

---

## 13. 代码审查清单

### 13.1 功能检查

- [ ] 功能是否完整
- [ ] 边界条件处理
- [ ] 错误处理完善
- [ ] 日志记录充分

### 13.2 代码质量

- [ ] 命名规范
- [ ] 注释充分
- [ ] 代码简洁
- [ ] 无重复代码

### 13.3 安全检查

- [ ] 输入验证
- [ ] SQL 注入防护
- [ ] XSS 防护
- [ ] 权限控制

### 13.4 性能检查

- [ ] 数据库查询优化
- [ ] 无内存泄漏
- [ ] 并发安全
- [ ] 资源释放

---

## 14. 常见问题

### 14.1 如何添加新功能？

1. 在 `api/hello/v1/` 定义请求/响应结构
2. 在 `internal/controller/` 创建控制器
3. 在 `internal/service/` 实现业务逻辑
4. 在 `internal/model/` 定义数据模型
5. 在 `internal/cmd/cmd.go` 注册路由
6. 编写单元测试
7. 更新文档

### 14.2 如何处理数据库变更？

1. 创建迁移脚本 `database/migration_*.sql`
2. 更新 Model 定义
3. 更新 Service 逻辑
4. 测试验证
5. 执行迁移脚本

### 14.3 如何调试爬虫？

1. 设置 `headless: false` 查看浏览器
2. 查看 `logs/` 目录下的日志
3. 使用 WebSocket 实时日志
4. 检查 `checkpoints/` 断点文件

---

## 15. 工具推荐

### 15.1 开发工具

- **IDE**: GoLand / VSCode
- **数据库**: Navicat / DBeaver
- **API 测试**: Postman / Apifox
- **版本控制**: Git

### 15.2 常用命令

```bash
# 运行项目
go run main.go

# 构建项目
go build -o server main.go

# 运行测试
go test ./...

# 格式化代码
gofmt -w .

# 代码检查
golangci-lint run
```

---

## 16. 更新记录

| 日期 | 版本 | 说明 |
|------|------|------|
| 2026-05-23 | v1.0 | 初始版本 |
