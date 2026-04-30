# 订单分析接口优化建议

## 当前代码分析

### 优点
1. ✅ SQL 查询逻辑正确，使用子查询避免 JOIN 重复计算
2. ✅ 代码注释清晰，易于维护
3. ✅ 使用了 GoFrame 框架的数据库抽象

### 存在的性能问题

## 一、数据库层面优化（最重要）

### 1. 缺少索引（Critical）
```sql
-- 必须添加的索引
CREATE INDEX idx_order_pay_finish_time ON order_info(pay_finish_time);
CREATE INDEX idx_order_main_order_no ON order_info(main_order_no);
CREATE INDEX idx_order_talent_id ON order_info(talent_id);
CREATE INDEX idx_order_status ON order_info(after_sale_status, order_status);
CREATE INDEX idx_goods_code_status ON goods(goods_code, status);
CREATE INDEX idx_shop_uid ON shop(uid);

-- 复合索引（覆盖常用查询条件）
CREATE INDEX idx_order_composite ON order_info(
    pay_finish_time, 
    main_order_no, 
    talent_id, 
    after_sale_status, 
    order_status
);
```

**影响：** 没有索引的情况下，每次查询都是全表扫描，数据量大时性能会急剧下降。

### 2. SQL 注入风险（Security）
```go
// ❌ 当前代码：直接拼接 SQL
sql := `WHERE o1.pay_finish_time >= '` + startDate + `'`

// ✅ 应该使用参数化查询
sql := `WHERE o1.pay_finish_time >= ?`
result, err := g.DB().GetAll(ctx, sql, startDate, endDate)
```

### 3. TRIM() 函数影响索引
```sql
-- ❌ 当前：TRIM 会导致索引失效
LEFT JOIN goods g ON TRIM(o1.seller_code) = TRIM(g.goods_code)

-- ✅ 优化：在插入数据时就 TRIM，查询时直接匹配
LEFT JOIN goods g ON o1.seller_code = g.goods_code
```

## 二、Go 代码层面优化

### 1. 使用结构体代替 map[string]interface{}

**当前问题：**
- `map[string]interface{}` 性能差，需要类型断言
- 没有类型安全，容易出错
- 内存分配多，GC 压力大

```go
// ❌ 当前代码
stats := make([]map[string]interface{}, 0, len(result))
for _, row := range result {
    stats = append(stats, map[string]interface{}{
        "groupName":  row["group_name"],
        "orderCount": row["order_count"].Int(),
        // ...
    })
}

// ✅ 优化后
type GroupStat struct {
    GroupName     string  `json:"groupName"`
    OrderCount    int     `json:"orderCount"`
    ProductNum    int     `json:"productNum"`
    PayAmount     float64 `json:"payAmount"`
    CostAmount    float64 `json:"costAmount"`
    ReserveAmount float64 `json:"reserveAmount"`
    Commission    float64 `json:"commission"`
}

stats := make([]GroupStat, 0, len(result))
for _, row := range result {
    productTotal := row["pay_amount"].Float64()
    costAmount := row["cost_amount"].Float64()
    reserveAmount := row["reserve_amount"].Float64()
    
    stats = append(stats, GroupStat{
        GroupName:     row["group_name"].String(),
        OrderCount:    row["order_count"].Int(),
        ProductNum:    row["product_num"].Int(),
        PayAmount:     roundFloat(productTotal, 2),
        CostAmount:    roundFloat(costAmount, 2),
        ReserveAmount: roundFloat(reserveAmount, 2),
        Commission:    roundFloat(productTotal - reserveAmount - costAmount, 2),
    })
}
```

**性能提升：** 约 30-50%，减少内存分配和 GC 压力

### 2. 并发查询优化

**当前问题：** Overview 方法中有 3 个独立查询，串行执行

```go
// ❌ 当前：串行执行
periodResult, _ := g.DB().GetAll(ctx, periodSQL)
refundResult, _ := g.DB().GetAll(ctx, refundSQL)
totalResult, _ := g.DB().GetAll(ctx, totalSQL)

// ✅ 优化：并发执行
type queryResult struct {
    data gdb.Result
    err  error
}

var wg sync.WaitGroup
periodCh := make(chan queryResult, 1)
refundCh := make(chan queryResult, 1)
totalCh := make(chan queryResult, 1)

wg.Add(3)

// 并发查询期间统计
go func() {
    defer wg.Done()
    data, err := g.DB().GetAll(ctx, periodSQL)
    periodCh <- queryResult{data, err}
}()

// 并发查询退款统计
go func() {
    defer wg.Done()
    data, err := g.DB().GetAll(ctx, refundSQL)
    refundCh <- queryResult{data, err}
}()

// 并发查询累计统计
go func() {
    defer wg.Done()
    data, err := g.DB().GetAll(ctx, totalSQL)
    totalCh <- queryResult{data, err}
}()

wg.Wait()
close(periodCh)
close(refundCh)
close(totalCh)

// 处理结果
periodRes := <-periodCh
if periodRes.err != nil {
    return nil, nil, periodRes.err
}
// ... 处理其他结果
```

**性能提升：** 3 个查询并发执行，总耗时 = max(query1, query2, query3)，而不是 sum

### 3. 添加缓存层

```go
import (
    "github.com/gogf/gf/v2/os/gcache"
    "time"
)

type OrderAnalysisService struct {
    cache *gcache.Cache
}

func NewOrderAnalysisService() *OrderAnalysisService {
    return &OrderAnalysisService{
        cache: gcache.New(),
    }
}

func (s *OrderAnalysisService) GroupStats(ctx context.Context, startDate, endDate string) ([]GroupStat, error) {
    // 生成缓存 key
    cacheKey := fmt.Sprintf("group_stats:%s:%s", startDate, endDate)
    
    // 尝试从缓存获取
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
        return cached.([]GroupStat), nil
    }
    
    // 查询数据库
    stats, err := s.queryGroupStats(ctx, startDate, endDate)
    if err != nil {
        return nil, err
    }
    
    // 写入缓存（5分钟过期）
    s.cache.Set(ctx, cacheKey, stats, 5*time.Minute)
    
    return stats, nil
}
```

**适用场景：**
- 今日/昨日数据：缓存 5-10 分钟
- 历史数据（last7/last30）：缓存 30 分钟到 1 小时
- 历史固定日期：可以缓存更久

### 4. 数据库连接池优化

```go
// 在初始化时配置连接池
func init() {
    db := g.DB()
    db.SetMaxIdleConns(10)   // 最大空闲连接数
    db.SetMaxOpenConns(100)  // 最大打开连接数
    db.SetConnMaxLifetime(time.Hour) // 连接最大生命周期
}
```

### 5. 预编译 SQL 语句

```go
// 对于频繁执行的查询，使用预编译
type OrderAnalysisService struct {
    groupStatsStmt *gdb.Stmt
}

func NewOrderAnalysisService() *OrderAnalysisService {
    // 预编译 SQL
    stmt, _ := g.DB().Prepare(`
        SELECT 
            COALESCE(s.operation_leader, '未分配小组') AS group_name,
            COUNT(DISTINCT o.main_order_no) AS order_count,
            -- ... 其他字段
        FROM (
            SELECT o1.main_order_no, o1.talent_id, 
                   SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount
            FROM order_info o1
            WHERE o1.pay_finish_time >= ? AND o1.pay_finish_time < ?
            GROUP BY o1.main_order_no, o1.talent_id
        ) o
        LEFT JOIN shop s ON o.talent_id = s.uid
        GROUP BY s.operation_leader
    `)
    
    return &OrderAnalysisService{
        groupStatsStmt: stmt,
    }
}
```

### 6. 批量查询优化

```go
// ❌ 当前：每次都查询所有字段
SELECT o1.main_order_no, o1.shop_name, o1.talent_id, 
       SUM(o1.product_num), SUM(o1.product_price * o1.product_num), ...

// ✅ 优化：只查询需要的字段
SELECT o1.main_order_no, o1.talent_id, 
       SUM(o1.product_price * o1.product_num) AS pay_amount
```

### 7. 使用 sync.Pool 减少内存分配

```go
var statPool = sync.Pool{
    New: func() interface{} {
        return make([]GroupStat, 0, 100)
    },
}

func (s *OrderAnalysisService) GroupStats(ctx context.Context, startDate, endDate string) ([]GroupStat, error) {
    // 从池中获取切片
    stats := statPool.Get().([]GroupStat)
    defer func() {
        // 清空并放回池中
        stats = stats[:0]
        statPool.Put(stats)
    }()
    
    // ... 查询和填充数据
    
    // 返回副本
    result := make([]GroupStat, len(stats))
    copy(result, stats)
    return result, nil
}
```

## 三、架构层面优化

### 1. 读写分离
```go
// 统计查询走从库
func (s *OrderAnalysisService) GroupStats(ctx context.Context, startDate, endDate string) ([]GroupStat, error) {
    // 使用只读从库
    result, err := g.DB("slave").GetAll(ctx, sql, startDate, endDate)
    // ...
}
```

### 2. 分库分表
如果订单数据量超过千万级别，考虑按时间分表：
- `order_info_202601`
- `order_info_202602`
- ...

### 3. 使用物化视图或汇总表
```sql
-- 创建每日汇总表
CREATE TABLE order_daily_summary (
    stat_date DATE PRIMARY KEY,
    group_name VARCHAR(100),
    order_count INT,
    pay_amount DECIMAL(15,2),
    cost_amount DECIMAL(15,2),
    commission DECIMAL(15,2),
    updated_at TIMESTAMP
);

-- 定时任务每天凌晨汇总前一天数据
-- 查询时直接读汇总表，速度极快
```

## 四、监控和分析

### 1. 添加慢查询日志
```go
func (s *OrderAnalysisService) GroupStats(ctx context.Context, startDate, endDate string) ([]GroupStat, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        if duration > 1*time.Second {
            g.Log().Warningf(ctx, "Slow query detected: GroupStats took %v", duration)
        }
    }()
    
    // ... 查询逻辑
}
```

### 2. 使用 EXPLAIN 分析查询计划
```sql
EXPLAIN SELECT ... FROM order_info WHERE pay_finish_time >= '2024-01-01';
```

## 五、优先级排序

### 🔴 高优先级（立即实施）
1. **添加数据库索引** - 性能提升 10-100 倍
2. **修复 SQL 注入风险** - 安全问题
3. **使用结构体代替 map** - 性能提升 30-50%

### 🟡 中优先级（近期实施）
4. **添加缓存层** - 性能提升 5-10 倍（对重复查询）
5. **并发查询优化** - 性能提升 2-3 倍
6. **移除 TRIM() 函数** - 性能提升 20-30%

### 🟢 低优先级（长期优化）
7. **读写分离** - 需要架构调整
8. **物化视图/汇总表** - 需要定时任务支持
9. **分库分表** - 数据量达到瓶颈时考虑

## 六、预期性能提升

假设当前查询耗时 5 秒：

| 优化项 | 预期耗时 | 提升比例 |
|--------|---------|---------|
| 添加索引 | 500ms | 90% ↓ |
| + 使用结构体 | 350ms | 30% ↓ |
| + 添加缓存（命中） | 10ms | 97% ↓ |
| + 并发查询 | 150ms | 57% ↓ |
| + 移除 TRIM | 100ms | 33% ↓ |

**综合优化后：5000ms → 100ms（98% 性能提升）**

## 七、实施建议

### 第一阶段（本周）
1. 添加数据库索引
2. 修复 SQL 注入问题
3. 定义结构体类型

### 第二阶段（下周）
4. 实现缓存层
5. 优化并发查询
6. 清理数据中的空格，移除 TRIM

### 第三阶段（下月）
7. 监控慢查询
8. 考虑读写分离
9. 评估是否需要汇总表

## 八、注意事项

1. **索引不是越多越好**：每个索引都会占用空间，并影响写入性能
2. **缓存一致性**：数据更新时需要清除相关缓存
3. **并发安全**：使用 goroutine 时注意数据竞争
4. **测试覆盖**：每次优化后都要进行性能测试和功能测试
