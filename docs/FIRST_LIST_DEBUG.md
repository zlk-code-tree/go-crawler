# 首次上榜标签检测优化指南

## 📸 实际页面结构分析（基于截图）

### 视觉特征
从截图中可以看到：

1. **排名列布局**：
   - 前 3 名有 `TOP 1/2/3` 的金色徽章
   - 第 4 名开始只有纯数字排名
   - "首次上榜"标签位于排名数字**下方**，是独立的浅绿色背景标签

2. **"首次上榜"标签样式**：
   - ✅ 浅绿色背景（可能是 ElementUI 的 `el-tag--success`）
   - ✅ 圆角边框
   - ✅ 小号字体
   - ✅ 独立于排名数字的元素

### 推测的 DOM 结构

```html
<td class="ranking-cell">
  <div class="rank-info">
    <span class="rank-number">4</span>
    <span class="el-tag el-tag--success el-tag--mini">首次上榜</span>
  </div>
</td>
```

或者：

```html
<td class="ranking-cell">
  <div>
    <span>4</span>
  </div>
  <div class="first-list-badge">
    <span class="el-tag el-tag--success">首次上榜</span>
  </div>
</td>
```

---

## 🔍 当前选择器策略的问题

### ❌ 问题 1：XPath `//*[text()='首次上榜']` 过于严格

这个选择器要求元素的**文本节点完全等于**"首次上榜"，但如果 HTML 结构是：

```html
<span class="el-tag el-tag--success">
  <i class="el-icon-success"></i>  <!-- 图标 -->
  首次上榜                          <!-- 文本 -->
</span>
```

那么 `text()` 会返回多个文本节点，导致匹配失败。

### ❌ 问题 2：CSS `:contains()` 伪类不是标准 CSS

`:contains('首次上榜')` 是 jQuery 扩展，**原生 CSS 不支持**。go-rod 的 `Element()` 方法可能无法正确解析。

### ❌ 问题 3：没有利用页面的实际 class 名

从截图看，标签很可能是标准的 ElementUI 组件，应该优先使用：
- `.el-tag--success`（绿色成功标签）
- `.el-tag--mini`（小尺寸标签）

---

## ✅ 优化方案

### 方案 A：调整 XPath 表达式（推荐）

```go
xpathSelectors := []string{
    // 使用 normalize-space() 处理空白字符
    "//*[normalize-space()='首次上榜']",
    
    // 或者检查文本内容（不要求是唯一文本节点）
    "//span[contains(., '首次上榜')]",
    
    // 针对 ElementUI 标签结构
    "//span[contains(@class, 'el-tag') and contains(., '首次')]",
}
```

**关键改进**：
- `normalize-space()` - 去除首尾空白并压缩中间空白
- `contains(., '首次上榜')` - 检查元素及其子元素的文本内容，而不是单个文本节点

### 方案 B：使用纯 CSS 选择器 + 文本验证

```go
cssSelectors := []string{
    ".el-tag.el-tag--success",      // ElementUI 成功标签
    ".el-tag.el-tag--mini",         // 小尺寸标签
    "[class*='first-list']",        // 自定义 first-list 类名
    "[class*='first-time']",        // 自定义 first-time 类名
}

// 找到元素后，再验证文本内容
for _, selector := range cssSelectors {
    elem, err := cells[0].Timeout(500 * time.Millisecond).Element(selector)
    if err == nil {
        // 获取文本验证
        if text, _ := elem.Text(); strings.Contains(text, "首次") {
            firstList = 1
            break
        }
    }
}
```

### 方案 C：JavaScript 注入检测（最可靠）

```go
// 直接在排名列单元格内执行 JS 检测
jsCode := `function() {
    const tags = this.querySelectorAll('.el-tag');
    for (let tag of tags) {
        if (tag.textContent.includes('首次上榜')) {
            return true;
        }
    }
    return false;
}`

isFirstList, err := cells[0].Eval(jsCode)
if err == nil && isFirstList.(bool) {
    firstList = 1
}
```

---

## 🛠️ 实施建议

### 优先级排序

1. **第一优先级**：使用 `//*[contains(., '首次上榜')]` 
   - 兼容性最好，能处理复杂的 DOM 结构
   
2. **第二优先级**：CSS `.el-tag.el-tag--success` + 文本验证
   - 利用 ElementUI 的标准样式类
   
3. **第三优先级**：JavaScript 注入检测
   - 作为最终回退方案，几乎不会失败

### 调试日志增强

```go
// 记录每个策略的检测结果
logDebug(ctx, taskId, fmt.Sprintf("🔍 [XPath 精确] 排名=%d | 结果=%v", rank, err1 == nil))
logDebug(ctx, taskId, fmt.Sprintf("🔍 [XPath 模糊] 排名=%d | 结果=%v", rank, err2 == nil))
logDebug(ctx, taskId, fmt.Sprintf("🔍 [CSS 类名] 排名=%d | 结果=%v", rank, err3 == nil))
logDebug(ctx, taskId, fmt.Sprintf("📊 最终结果 | 排名=%d | first_list=%d | 单元格文本=%q", rank, firstList, cellText))
```

### 验证步骤

1. **运行爬虫**，抓取包含"首次上榜"商品的页面
2. **查看日志**，确认哪些商品被标记为 `first_list=1`
3. **对比数据库**，检查标记是否准确
4. **如有误判**，在浏览器中手动检查该商品的 DOM 结构

---

## 📝 代码示例

```go
// 检测是否包含"首次上榜"标识（仅在排名列单元格内查找）
firstList := 0

// ===== 策略 1: XPath contains() - 最兼容 =====
xpathSelectors := []string{
    "//*[contains(., '首次上榜')]",                              // 包含文本（推荐）
    "//span[contains(@class, 'el-tag') and contains(., '首次')]", // ElementUI 标签
}

for _, selector := range xpathSelectors {
    _, err := cells[0].Timeout(500 * time.Millisecond).ElementX(selector)
    if err == nil {
        firstList = 1
        logDebug(ctx, taskId, fmt.Sprintf("🏆 [XPath] 检测到首次上榜 [%s]：排名=%d", selector, rank))
        break
    }
}

// ===== 策略 2: CSS 类名 + 文本验证 =====
if firstList == 0 {
    cssSelectors := []string{
        ".el-tag.el-tag--success",  // ElementUI 成功标签
        ".el-tag.el-tag--mini",     // 小尺寸标签
    }
    
    for _, selector := range cssSelectors {
        elem, err := cells[0].Timeout(500 * time.Millisecond).Element(selector)
        if err == nil {
            if text, _ := elem.Text(); strings.Contains(text, "首次") {
                firstList = 1
                logDebug(ctx, taskId, fmt.Sprintf("🏆 [CSS] 检测到首次上榜 [%s]：排名=%d", selector, rank))
                break
            }
        }
    }
}

// ===== 策略 3: JavaScript 注入（最终回退）=====
if firstList == 0 {
    jsCode := `function() {
        const tags = this.querySelectorAll('.el-tag, [class*="first"]');
        for (let tag of tags) {
            if (tag.textContent && tag.textContent.includes('首次')) {
                return true;
            }
        }
        return false;
    }`
    
    result, err := cells[0].Eval(jsCode)
    if err == nil && result != nil {
        if isFirst, ok := result.(bool); ok && isFirst {
            firstList = 1
            logDebug(ctx, taskId, fmt.Sprintf("🏆 [JS] 检测到首次上榜：排名=%d", rank))
        }
    }
}

// 输出调试日志
cellText := s.getText(cells[0])
logDebug(ctx, taskId, fmt.Sprintf("📊 首次上榜检测完成 | 排名=%d | first_list=%d | 单元格文本=%q", 
    rank, firstList, cellText))
```

---

## 🧪 测试验证

运行爬虫后，使用以下命令验证：

```bash
# 查看前 50 条商品的检测结果
grep "首次上榜检测完成" logs/crawler-$(date +%Y%m%d).log | head -50

# 只查看被标记为首次上榜的商品
grep "first_list=1" logs/crawler-*.log | wc -l

# 查看具体的选择器命中情况
grep "\[XPath\]\|\[CSS\]\|\[JS\]" logs/crawler-*.log | sort | uniq -c
```

**预期结果**：
- ✅ 有"首次上榜"标签的商品 → `first_list=1`
- ✅ 没有标签的商品 → `first_list=0`
- ✅ 日志显示是通过哪个选择器命中的

---

## 📚 参考资料

- [XPath contains() vs text()](https://stackoverflow.com/questions/1297468/xpath-text-function-vs-xpath-text)
- [ElementUI Tag 组件文档](https://element.eleme.io/#/zh-CN/component/tag)
- [go-rod ElementX 文档](https://pkg.go.dev/github.com/go-rod/rod#Element.ElementX)
