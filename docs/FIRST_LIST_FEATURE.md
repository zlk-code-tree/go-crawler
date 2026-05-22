# 首次上榜标识功能实现文档

## 📋 功能说明

在爬取短视频榜商品数据时，自动检测商品是否包含"首次上榜"标识，并在数据库中进行标记。

- **字段名称**: `first_list`
- **字段类型**: `TINYINT`
- **默认值**: `0` (否)
- **标记值**: `1` (是首次上榜)

---

## 🔧 修改内容

### 1. 数据库层修改

#### 1.1 表结构更新 (`database/schema.sql`)

在 `tiktok_ranking_list` 表中新增字段：

```sql
`first_list` TINYINT NOT NULL DEFAULT 0 COMMENT '是否首次上榜：0-否 1-是'
```

**位置**: 在 `min_price` 字段之后，`created_at` 字段之前

#### 1.2 迁移脚本 (`database/migration_add_first_list.sql`)

为现有数据库提供迁移脚本：

```sql
ALTER TABLE `tiktok_ranking_list` 
ADD COLUMN `first_list` TINYINT NOT NULL DEFAULT 0 COMMENT '是否首次上榜：0-否 1-是' AFTER `min_price`;
```

**执行方式**:
```bash
mysql -u root -p easyadmin8 < database/migration_add_first_list.sql
```

---

### 2. Model 层修改

#### 2.1 输入模型 (`internal/model/tiktok_ranking_list.go`)

在 `TikTokRankingListInput` 结构体中添加字段：

```go
type TikTokRankingListInput struct {
    // ... 其他字段
    FirstList int `json:"first_list" db:"first_list"`  // 是否首次上榜：0-否 1-是
    // ... 其他字段
}
```

在 `TiktokRankingList` 实体结构体中添加字段：

```go
type TiktokRankingList struct {
    // ... 其他字段
    FirstList int `json:"first_list" db:"first_list"`  // 是否首次上榜：0-否 1-是
    // ... 其他字段
}
```

---

### 3. Logic 层修改

#### 3.1 数据结构 (`internal/logic/crawler/crawler.go`)

在 `ProductInfo` 结构体中添加字段：

```go
type ProductInfo struct {
    Rank             int    // 排名
    ProductName      string // 商品名称
    ShopName         string // 店铺名称
    // ... 其他字段
    FirstList        int    // 是否首次上榜：0-否 1-是
}
```

#### 3.2 解析逻辑 (`parseProductRow` 函数)

新增"首次上榜"标识检测逻辑：

```go
// 检测是否包含"首次上榜"标识
firstList := 0
rowText, _ := row.Text()
if strings.Contains(rowText, "首次上榜") || strings.Contains(rowText, "首次") {
    firstList = 1
    logDebug(ctx, taskId, fmt.Sprintf("🏆 检测到首次上榜商品：排名=%d", rank))
}

product := &ProductInfo{
    // ... 其他字段
    FirstList: firstList,
}
```

**检测策略**:
- 检测整行文本是否包含"首次上榜"或"首次"关键词
- 支持多种 DOM 结构和标识展示方式
- 记录调试日志，方便追踪检测结果

#### 3.3 数据转换 (`savePageData` 函数)

在保存到数据库时包含 `FirstList` 字段：

```go
record := &model.TikTokRankingListInput{
    TaskId:           taskId,
    Rank:             p.Rank,
    ProductName:      p.ProductName,
    // ... 其他字段
    FirstList:        p.FirstList,  // 首次上榜标识
    CreatedAt:        time.Now(),
}
```

#### 3.4 数据转换 (`convertToModelProducts` 函数)

在转换函数中也包含 `FirstList` 字段：

```go
record := model.TikTokRankingListInput{
    Rank:             p.Rank,
    ProductName:      p.ProductName,
    // ... 其他字段
    FirstList:        p.FirstList,
    CreatedAt:        time.Now(),
}
```

---

## 📊 数据查询示例

### 查询所有首次上榜商品

```sql
SELECT 
    id,
    task_id,
    rank,
    product_name,
    shop_name,
    payment_amount,
    first_list,
    created_at
FROM tiktok_ranking_list 
WHERE first_list = 1 
ORDER BY created_at DESC;
```

### 统计首次上榜商品数量

```sql
SELECT 
    COUNT(*) as total_count,
    SUM(CASE WHEN first_list = 1 THEN 1 ELSE 0 END) as first_list_count
FROM tiktok_ranking_list;
```

### 按任务统计首次上榜情况

```sql
SELECT 
    task_id,
    COUNT(*) as total_products,
    SUM(CASE WHEN first_list = 1 THEN 1 ELSE 0 END) as first_list_count,
    ROUND(SUM(CASE WHEN first_list = 1 THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) as first_list_percentage
FROM tiktok_ranking_list 
GROUP BY task_id
ORDER BY created_at DESC;
```

---

## ✅ 测试验证

### 1. 数据库验证

执行迁移脚本后，检查字段是否添加成功：

```sql
DESCRIBE tiktok_ranking_list;
```

应该看到 `first_list` 字段。

### 2. 爬虫日志验证

启动爬虫后，在日志中查找：

```
🏆 检测到首次上榜商品：排名=X
✅ 解析商品行 X: 排名=Y, 名称=XXX, 首次上榜=1
```

### 3. 数据库数据验证

爬取完成后，查询数据：

```sql
SELECT rank, product_name, first_list 
FROM tiktok_ranking_list 
WHERE first_list = 1 
LIMIT 10;
```

---

## 📝 注意事项

1. **默认值**: 所有非首次上榜商品的 `first_list` 字段默认为 `0`
2. **兼容性**: 新增字段不影响现有功能和数据结构
3. **性能**: 字段使用 `TINYINT` 类型，占用空间小，查询效率高
4. **扩展性**: 如需更详细的标识（如"新上榜"、"重回榜单"等），可在此基础上扩展

---

## 🔄 版本信息

- **功能版本**: v1.0
- **实现日期**: 2026-03-30
- **修改文件**:
  - `database/schema.sql`
  - `database/migration_add_first_list.sql` (新增)
  - `internal/model/tiktok_ranking_list.go`
  - `internal/logic/crawler/crawler.go`
  - `FIRST_LIST_FEATURE.md` (新增)

---

## 📞 技术支持

如有问题，请查看：
- 爬虫日志：`logs/` 目录
- SQL 日志：`logs/sql/` 目录
- 数据库文档：`database/schema.sql`
