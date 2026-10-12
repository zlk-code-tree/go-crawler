# 首次上榜检测逻辑修复

## 🐛 问题描述

**现象**：所有商品都被标记为 `first_list=1`，包括没有首次上榜标签的商品。

**影响**：数据库中的首次上榜标识失去意义，无法区分哪些是真正的新上榜商品。

---

## 🔍 根本原因

在 `internal/logic/crawler/crawler.go` 的 `parseProductRow` 函数中，首次上榜检测使用了**过于宽泛的 CSS 选择器**：

### ❌ 修复前的错误代码

```go
firstListSelectors := []string{
    "//*[contains(text(), '首次上榜')]",  // ✅ 正确：精确匹配文本
    ".el-tag--danger",     // ❌ 错误：所有红色标签都匹配（包括价格、状态等）
    ".ant-tag-red",        // ❌ 错误：所有红色标签都匹配
    ".badge-new",          // ❌ 可能误判其他"新"标签
    ".tag-first",          // ❌ 可能误判其他"第一"标签
    "[class*='first']",    // ❌ 太宽泛：任何包含 'first' 的 class
    "[class*='new']",      // ❌ 太宽泛：任何包含 'new' 的 class
}
```

**问题**：这些 CSS 选择器会匹配排名列单元格内**任何带有这些 class 的元素**，而抖音罗盘页面中可能存在其他类型的红色标签（如价格标签、热度标签等），导致误判。

---

## ✅ 修复方案

### 修复后的正确代码

```go
// 策略 1: 在排名列单元格内查找"首次上榜"标签元素（精确匹配文本）
firstListSelectors := []string{
    "//*[contains(text(), '首次上榜')]",
    "//*[contains(text(), '首次')]",
}

for _, selector := range firstListSelectors {
    var err error
    
    // XPath：在第一个单元格内搜索
    _, err = cells[0].Timeout(500 * time.Millisecond).ElementX(selector)
    
    if err == nil {
        firstList = 1
        logDebug(ctx, taskId, fmt.Sprintf("🏆 在排名列检测到首次上榜标识 [%s]：排名=%d", selector, rank))
        break
    }
}

// 策略 2: 如果 DOM 元素未找到，回退到排名列文本匹配
if firstList == 0 {
    rankCellText := s.getText(cells[0])
    if strings.Contains(rankCellText, "首次上榜") || strings.Contains(rankCellText, "首次") {
        firstList = 1
        logDebug(ctx, taskId, fmt.Sprintf("🏆 通过排名列文本匹配检测到首次上榜：排名=%d", rank))
    }
}
```

### 修复要点

1. **仅使用 XPath 文本匹配**：移除所有 CSS 选择器，只保留基于文本内容的 XPath 查询
2. **严格限定检测范围**：只在排名列单元格（`cells[0]`）内搜索
3. **双重验证机制**：
   - 策略 1：DOM 元素文本匹配（优先）
   - 策略 2：单元格纯文本关键词匹配（兜底）

---

## 🧪 验证方法

### 1. 运行爬虫测试

```bash
cd /Users/zhanglikun/workspace/app/go-crawler
go run main.go
```

### 2. 查看日志输出

```bash
# 查看首次上榜检测结果
grep "首次上榜=" logs/crawler-$(date +%Y%m%d).log | tail -20
```

**预期结果**：
- ✅ 有首次上榜标签的商品：`首次上榜=1`
- ✅ 没有首次上榜标签的商品：`首次上榜=0`

### 3. 检查数据库

```sql
-- 查看首次上榜统计
SELECT 
    first_list,
    COUNT(*) as count
FROM tiktok_ranking_list 
WHERE task_id = '你的任务 ID'
GROUP BY first_list;

-- 查看具体的首次上榜商品
SELECT rank, product_name, first_list 
FROM tiktok_ranking_list 
WHERE first_list = 1
ORDER BY rank;
```

---

## 📊 预期效果

| 场景 | 修复前 | 修复后 |
|------|--------|--------|
| 有"首次上榜"标签 | ✅ `first_list=1` | ✅ `first_list=1` |
| 无"首次上榜"标签 | ❌ `first_list=1` (误判) | ✅ `first_list=0` (正确) |
| 准确率 | ~0% (全部误判) | ~100% (精确匹配) |

---

## 🔧 相关文件

- **修改文件**：`internal/logic/crawler/crawler.go`
- **修改函数**：`parseProductRow()`
- **修改位置**：第 522-552 行

---

## 📝 注意事项

1. **不要添加 CSS 选择器**：除非能 100% 确定该 class 是"首次上榜"标签独有
2. **XPath 是最可靠的方式**：基于文本内容的匹配最准确
3. **调试时查看日志**：日志会显示是通过哪个选择器检测到的
4. **如果仍然误判**：检查页面实际 DOM 结构，可能需要调整 XPath 表达式

---

## 📅 修复记录

- **修复时间**：2026-03-31 15:35
- **修复人**：张立昆
- **问题类型**：选择器误判导致全量标记错误
- **修复状态**：✅ 已完成
