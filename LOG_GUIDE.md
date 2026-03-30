# 📝 日志系统使用指南

## 🎯 日志特性

✅ **双路输出**：同时记录到文件和控制台  
✅ **按天分割**：每天自动生成新的日志文件  
✅ **自动轮转**：单个文件超过 10MB 自动分割  
✅ **定期清理**：自动保留最近 7 天的日志  
✅ **详细记录**：记录爬虫全流程的关键节点  

## 📁 日志文件位置

```
go-crawler/
└── logs/
    ├── crawler-20260330.log    # 今天的日志
    ├── crawler-20260329.log    # 昨天的日志
    ├── crawler-20260328.log    # 前天的日志
    └── ...                     # 最近 7 天的日志
```

## 🔍 查看日志的方法

### 方法 1: 使用日志查看工具（推荐）

```bash
# 赋予执行权限
chmod +x view_logs.sh

# 运行交互式查看工具
./view_logs.sh
```

**功能菜单：**
- 实时查看最新日志
- 查看最新 100 行
- 查看错误/警告日志
- 爬虫任务统计
- 清理旧日志
- 导出日志到文本

### 方法 2: 使用命令行工具

```bash
# 实时查看最新日志
tail -f logs/crawler-$(date +%Y%m%d).log

# 查看最新 100 行
tail -100 logs/crawler-$(date +%Y%m%d).log

# 搜索错误日志
grep "ERROR\|❌" logs/crawler-$(date +%Y%m%d).log

# 搜索警告日志
grep "WARNING\|⚠️" logs/crawler-$(date +%Y%m%d).log

# 搜索特定任务
grep "任务 ID" logs/crawler-$(date +%Y%m%d).log

# 统计今天任务数
grep -c "爬虫任务开始" logs/crawler-$(date +%Y%m%d).log
```

### 方法 3: 使用日志分析命令

```bash
# 查看今天的爬虫任务统计
echo "=== 今日爬虫统计 ==="
echo "总任务数：$(grep -c '爬虫任务开始' logs/crawler-$(date +%Y%m%d).log)"
echo "成功：$(grep -c '爬虫任务全部完成' logs/crawler-$(date +%Y%m%d).log)"
echo "失败：$(grep -c '异常终止\|执行失败' logs/crawler-$(date +%Y%m%d).log)"

# 查看抓取的数据总量
echo "=== 数据抓取统计 ==="
grep "共获取.*条商品数据" logs/crawler-$(date +%Y%m%d).log | tail -5
```

## 📊 日志格式说明

### 日志级别

- **INFO** - 普通信息（蓝色📝）
- **WARNING** - 警告信息（黄色⚠️）
- **ERROR** - 错误信息（红色❌）
- **DEBUG** - 调试信息（绿色🔍）
- **SUCCESS** - 成功信息（绿色✅）

### 日志格式

```
[时间戳] [级别] [任务 ID] 消息内容
```

**示例：**
```
[2026-03-30 10:18:19.000] [INFO] 📌 [abc123] 步骤 1: 启动浏览器
[2026-03-30 10:18:20.500] [INFO] 🌐 浏览器启动成功 | 任务 ID：abc123
[2026-03-30 10:18:25.000] [INFO] ✅ 登录成功 | 任务 ID：abc123
[2026-03-30 10:18:30.000] [WARNING] ⚠️ [abc123] 页面滚动异常
[2026-03-30 10:19:00.000] [ERROR] ❌ [abc123] 数据入库失败
```

## 🎯 关键日志节点

### 爬虫启动
```
📌 [任务 ID] 步骤 1: 启动浏览器
🚀 初始化浏览器 | 任务 ID：xxx
🌐 浏览器启动成功
```

### 登录检测
```
🔐 步骤 2: 打开首页并检测登录状态
✅ 已登录，跳过登录步骤
# 或
📝 步骤 3: 未登录，执行登录流程
✅ 登录成功
💾 保存 Cookie 到文件
```

### 数据抓取
```
📈 步骤 6: 分页抓取商品榜单数据
📄 正在抓取第 1/20 页
✅ 第 1 页抓取完成：获取 10 条数据
📊 分页抓取完成，共获取 200 条商品
```

### 数据导出
```
📥 步骤 10: 导出数据文件
✅ 导出 CSV 成功：exports/xxx.csv (200 条)
✅ 导出 Excel 成功：exports/xxx.xlsx (200 条)
```

### 任务完成
```
🎉 爬虫任务全部完成！商品：200 条，详情：200 条，视频分析：500 条
✅ 任务完成，共抓取 200 条数据
```

## 🔧 日志配置

日志配置在 `internal/logic/crawler/crawler.go` 的 `initLogConfig()` 函数中：

```go
glog.SetConfigWithMap(map[string]interface{}{
    "path":    "logs",                    // 日志目录
    "file":    "crawler-{date}.log",      // 文件名格式
    "level":   "all",                     // 日志级别
    "stdout":  true,                      // 输出到控制台
    "rotate":  true,                      // 开启轮转
    "daily":   true,                      // 按天分割
    "maxDays": 7,                         // 保留天数
    "maxSize": 10 * 1024 * 1024,         // 单文件最大 10MB
})
```

## 📈 日志分析示例

### 1. 查看任务成功率
```bash
total=$(grep -c "爬虫任务开始" logs/crawler-$(date +%Y%m%d).log)
success=$(grep -c "爬虫任务全部完成" logs/crawler-$(date +%Y%m%d).log)
echo "成功率：$(echo "scale=2; $success * 100 / $total" | bc)%"
```

### 2. 查看平均抓取数据量
```bash
grep "共获取.*条商品数据" logs/crawler-*.log | \
  grep -oP "\d+(?= 条商品数据)" | \
  awk '{s+=$1; n++} END {print "平均每次抓取：" s/n " 条"}'
```

### 3. 查找失败原因
```bash
# 查看最近的错误
grep -B5 "ERROR\|❌" logs/crawler-$(date +%Y%m%d).log | tail -30
```

### 4. 查看性能瓶颈
```bash
# 查看各步骤耗时
grep "步骤 [0-9]" logs/crawler-$(date +%Y%m%d).log | head -20
```

## ⚠️ 常见问题

### Q: 日志文件太大怎么办？
A: 系统会自动轮转，单文件超过 10MB 会分割。也可以手动清理：
```bash
# 清理 7 天前的日志
find logs/ -name "*.log" -mtime +7 -delete
```

### Q: 如何只记录错误日志？
A: 修改配置中的 `"level": "all"` 为 `"level": "error"`

### Q: 如何关闭控制台输出？
A: 修改配置中的 `"stdout": true` 为 `"stdout": false`

### Q: 日志文件在哪里？
A: 在项目根目录的 `logs/` 文件夹下

## 📞 技术支持

查看日志时如遇到问题：
1. 检查 `logs/` 目录是否有写权限
2. 确认磁盘空间是否充足
3. 查看最新的错误日志定位问题

```bash
# 快速诊断
ls -lh logs/
df -h .
tail -50 logs/crawler-$(date +%Y%m%d).log
```
