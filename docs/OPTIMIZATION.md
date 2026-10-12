# 抖音电商罗盘爬虫 - 优化功能说明

## ✅ 已完成的优化功能

### 1. Cookie 保持登录态

**功能说明：**
- 首次登录后自动保存 Cookie 到 `cookies/cookies.json`
- 下次启动时自动加载 Cookie，无需重新登录
- Cookie 过期自动检测并清除

**文件位置：**
- `internal/service/cookie_manager.go`

**使用方式：**
```bash
# 首次运行需要手动登录
./main

# 下次运行会自动使用保存的 Cookie
./main
```

### 2. 断点续爬功能

**功能说明：**
- 爬虫任务意外中断后可以从断点处继续
- 自动保存进度到 `checkpoints/task_<taskId>.json`
- 任务完成后自动清理断点文件

**文件位置：**
- `internal/service/checkpoint_manager.go`

**使用场景：**
- 网络中断
- 程序崩溃
- 手动停止

### 3. 数据导出功能（CSV/Excel）

**功能说明：**
- 自动导出商品数据到 CSV 和 Excel 格式
- 导出文件保存到 `exports/` 目录
- 支持 UTF-8 BOM 编码，Excel 打开不乱码
- 自动设置表头样式、列宽、筛选

**文件位置：**
- `internal/service/export_manager.go`

**导出文件命名：**
- `product_ranking_<taskId>.csv` - 商品榜单 CSV
- `product_ranking_<taskId>.xlsx` - 商品榜单 Excel
- `video_analysis_<taskId>.csv` - 视频分析 CSV

## 📊 数据目录结构

```
go-crawler/
├── cookies/              # Cookie 存储目录
│   └── cookies.json
├── checkpoints/          # 断点文件目录
│   └── task_<id>.json
├── exports/              # 导出文件目录
│   ├── product_ranking_<id>.csv
│   ├── product_ranking_<id>.xlsx
│   └── video_analysis_<id>.csv
├── logs/                 # 日志文件目录
│   └── crawler-20260330.log
└── main                  # 可执行文件
```

## 🚀 快速开始

### 1. 启动服务

```bash
cd /Users/zhanglikun/workspace/app/go-crawler
./main
```

### 2. 访问 Swagger 文档

打开浏览器访问：http://localhost:8000/swagger

### 3. 调用 API

```bash
curl -X POST http://localhost:8000/crawler \
  -H "Content-Type: application/json" \
  -d '{
    "targetUrl": "https://compass.jinritemai.com/shop",
    "loginType": "qrcode"
  }'
```

### 4. 查看导出文件

```bash
# 查看最新导出的文件
ls -lt exports/ | head

# 打开 Excel 文件
open exports/product_ranking_*.xlsx
```

## 🔧 高级配置

### 清理旧数据

```bash
# 清理 7 天前的断点文件
rm -rf checkpoints/task_*.json

# 清理 30 天前的导出文件
find exports/ -mtime +30 -delete

# 清理过期的 Cookie
rm cookies/cookies.json
```

### 查看日志

```bash
# 实时查看日志
tail -f logs/crawler-$(date +%Y%m%d).log

# 查看错误日志
grep ERROR logs/crawler-*.log
```

## 📈 性能优化建议

1. **并发控制**：目前顺序抓取，可考虑增加并发限制
2. **代理池**：大规模爬取时建议添加代理 IP
3. **限流**：添加请求间隔，避免被封禁
4. **增量更新**：只抓取新增或变化的数据

## ⚠️ 注意事项

1. **合法合规**：请确保爬取行为符合目标网站的使用条款
2. **频率控制**：避免高频请求导致 IP 被封
3. **数据安全**：妥善保管 Cookie 和导出的数据
4. **资源占用**：爬虫运行时会占用较多内存和 CPU

## 📞 技术支持

如有问题，请查看日志文件：
```bash
# 最新日志
tail -100 logs/crawler-$(date +%Y%m%d).log
```
