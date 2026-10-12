# go-crawler 代码审查与优化建议

> 更新说明（2026-10-08）：以下正文保留 10 月 4 日的审查快照，行号与“尚未修复”判断只对应当时版本。本轮已拆分爬虫文件，并修复详情缓存死锁、视频写入与断点顺序、视频池暂停续跑、榜单流水线重入、浏览器停止状态竞争，以及榜单重放重复入库和详情跨任务覆盖。新增离线回归测试；未进行真实平台与数据库联调。本文其余权限、订单、构建等问题仍属后续工作，不应把旧行号当作当前定位。

审查日期：2026-10-04。基于当前工作区，保留已有未提交改动。本次未修改业务代码，未启动服务、访问业务数据库或执行真实平台采集。

**总体判断**

项目已是一个电商数据后端：GoFrame HTTP API、Rod 浏览器采集、多平台 API 同步、MySQL 数据处理、财务统计、权限与通知。排除 `internal/pkg` 内嵌 SDK 后，共 179 个 Go 文件、约 63,384 行，其中 19 个测试文件。

已有值得保留的设计：平台适配层、按页持久化、断点对象、订单批量幂等写入、财务日汇总事务替换，以及日期、归属和利润计算的单元测试。当前最值得投入的是修复任务正确性与后端权限，其次是订单查询的内存开销和构建可复现性。

下文 P1 表示应优先修复的功能、安全或数据完整性问题；P2 表示随后安排的可靠性、性能和工程改进。影响由代码路径推导；只有明确注明复现的条目做了运行验证，不代表已经在生产环境观察到故障。

**1. [P1] 商品详情缓存更新存在确定性死锁，已本地复现**

位置：[product_detail_cache.go:133](/Users/zhanglikun/workspace/app/go-crawler/internal/service/product_detail_cache.go:133)、[product_detail_cache.go:98](/Users/zhanglikun/workspace/app/go-crawler/internal/service/product_detail_cache.go:98)。

`UpsertRecord` 获取写锁并通过 defer 延后释放，再调用 `saveRecords` 获取同一个 RWMutex 的读锁。后者永远无法获得锁，前者也无法返回释放锁。一次非空商品 ID 的缓存更新即可触发，不需要并发。

详情缓存未命中或过期且成功抓到价格/上架时间时，[crawler.go:1781](/Users/zhanglikun/workspace/app/go-crawler/internal/logic/crawler/crawler.go:1781) 会调用该方法，详情任务最终卡在 worker 等待中；Context 取消不能解除锁等待。当前配置是否开启详情抓取决定业务触发时机。

建议：明确锁的归属，锁内更新并获取快照，释放锁后交由串行持久化流程写文件；持久化使用临时文件加原子替换，并保证旧快照不会覆盖新快照。增加缓存更新返回及并发持久化测试。

验证：通过临时 Go overlay 注入纯内存/临时目录测试，200ms 内无法返回，稳定复现。仓库源码未改。

**2. [P1] 手动触发内置定时任务会返回成功，但 handler 实际不执行，已本地复现**

位置：[cron_manager.go:225](/Users/zhanglikun/workspace/app/go-crawler/internal/service/cron_manager.go:225)、[cron_manager.go:134](/Users/zhanglikun/workspace/app/go-crawler/internal/service/cron_manager.go:134)。

`triggerByName` 先设置 `running[name]=true`，然后调用 `runWithLock`；后者看到这个标记后直接返回。外层仍更新时间并返回成功。手动触发内置任务以及任务链触发内置任务都会受影响；正常 cron 直接调用 `runWithLock` 的路径不受此问题影响。

建议：将运行资格的检查与占用集中到一个入口，手动和定时调度共用；任务接收、开始、完成三种状态分开表示。补充 handler 恰好执行一次、重复触发被拒绝的测试。

验证：临时 overlay 中注册纯内存计数 handler，等待任务收尾后计数为 0，而 Trigger 返回 nil；没有数据库/API 调用。

**3. [P1] 管理接口缺少实际的角色权限校验**

位置：[cmd.go:1226](/Users/zhanglikun/workspace/app/go-crawler/internal/cmd/cmd.go:1226)、[middleware.go:49](/Users/zhanglikun/workspace/app/go-crawler/internal/middleware/middleware.go:49)、[role.go:40](/Users/zhanglikun/workspace/app/go-crawler/internal/controller/system/role.go:40)。

认证分组仅挂 JWT、操作日志和响应中间件；`PermissionAuth` 在仓库中有定义但未接入路由。角色保存、角色菜单授权等控制器及服务也没有校验操作人权限。因此普通用户的有效 JWT 可以通过这些敏感管理接口的后端校验，前端隐藏按钮不能提供保护。

建议：接入后端 RBAC，并校对每个实际路由的方法与权限编码。不能仅添加一次中间件调用：现有映射对未知接口默认放行，部分路径/方法与当前接口不同。对敏感操作采用显式声明权限，并增加普通用户被拒绝的接口测试。

**4. [P1] 兼容爬虫入口允许匿名调用**

位置：[cmd.go:1435](/Users/zhanglikun/workspace/app/go-crawler/internal/cmd/cmd.go:1435)、[crawler.go:9](/Users/zhanglikun/workspace/app/go-crawler/api/hello/v1/crawler.go:9)。

根路由注册的 `POST /crawl` 没有 JWT 中间件，控制器直接执行浏览器采集流程。该流程会使用全局已保存 Cookie；已有有效登录状态时，匿名调用可借用该业务身份采集。即使无有效 Cookie，仍可能启动浏览器等待登录、消耗资源。

建议：移除兼容入口，或统一放进认证、授权和有并发上限的任务队列。代码层面缺口明确；未验证部署端反向代理是否额外拦截该路径。

**5. [P1] 数据写入失败与同步进度脱节，可能漏单、漏视频**

位置：[douyin/poller.go:280](/Users/zhanglikun/workspace/app/go-crawler/internal/service/platform/douyin/poller.go:280)、[douyin/poller.go:294](/Users/zhanglikun/workspace/app/go-crawler/internal/service/platform/douyin/poller.go:294)、[crawler.go:983](/Users/zhanglikun/workspace/app/go-crawler/internal/logic/crawler/crawler.go:983)。

订单保存接口只返回成功数量与 ID，单条失败只记日志；轮询调用未依据保存结果设置失败状态，仍可能推进 `last_poll_time`。如果部分数据写失败、后续更新断点成功，这些订单就落在过去的增量区间，不能依赖后续轮询补回，需回补或订单再次发生更新。

视频保存也在 INSERT 失败后继续并返回 nil；回调不返回 error，而 [crawler.go:3202](/Users/zhanglikun/workspace/app/go-crawler/internal/logic/crawler/crawler.go:3202) 继续推进商品断点。

建议：保存结果明确返回失败记录/错误；只有该批数据可靠入库后才提交进度。保留唯一键幂等写入，失败可重试；无法入库的异常数据应进入可查询的失败队列，不能仅写日志后跳过。

**6. [P1] 视频池任务丢弃暂停与断点信息**

位置：[crawler.go:788](/Users/zhanglikun/workspace/app/go-crawler/internal/logic/crawler/crawler.go:788)、[crawler.go:3118](/Users/zhanglikun/workspace/app/go-crawler/internal/logic/crawler/crawler.go:3118)、[crawler.go:3131](/Users/zhanglikun/workspace/app/go-crawler/internal/logic/crawler/crawler.go:3131)。

独立视频池任务通过包装器每次传入 `startIndex=0`、`checkpointManager=nil`，只取返回值的 Videos，丢弃 Paused、NextProductIndex 和 StopReason。达到默认 10 次详情查看上限或连续空结果阈值时，上层仍记录完成并返回 nil。商品集合顺序相同时，后续周期会重复处理开头的商品，无法推进到后面的商品。

建议：保留访问配额保护，贯穿传递完整结果；保存下一商品位置，将暂停与完成区分，并让下一轮从断点继续。

**7. [P2] 榜单到视频池的流水线交接可能再次触发榜单，已用 stub 复现**

位置：[scheduler.go:240](/Users/zhanglikun/workspace/app/go-crawler/internal/service/scheduler.go:240)、[scheduler.go:185](/Users/zhanglikun/workspace/app/go-crawler/internal/service/scheduler.go:185)。

榜单完成后立即设为 Idle，触发视频任务并返回，但直到视频结束才更新榜单 NextRunAt。视频执行期间 tick 仍可能认为榜单到期，再次执行它。真实浏览器流程中的在线状态检查通常会拒绝这次执行，但会制造无效重试，交接时也存在竞争。

建议：对整条流水线建立互斥或明确的阶段状态，视频结束后再释放下一轮资格。纯本地 stub 测试中，视频 handler 尚未结束时，榜单 handler 执行次数已变为 2。

**8. [P1] 通知 WebSocket 存在并发写入路径**

位置：[ws.go:89](/Users/zhanglikun/workspace/app/go-crawler/internal/controller/ws/ws.go:89)、[ws.go:225](/Users/zhanglikun/workspace/app/go-crawler/internal/controller/ws/ws.go:225)。

广播持 hub 锁写 TextMessage，而连接处理器不持同一锁，每 30 秒用 WriteMessage 写 PingMessage。两个路径可能同时写同一连接，触发 gorilla/websocket 的并发写 panic。广播还在持锁时进行无写超时的网络操作，慢连接会影响通知与调用方任务收尾。

建议：每个连接单一 writer goroutine，使用有界队列与写超时；心跳统一串行发送，或使用支持并发的 WriteControl。写失败后及时移除连接。

**9. [P1] 账号禁用与会话有效性没有统一检查**

位置：[middleware.go:34](/Users/zhanglikun/workspace/app/go-crawler/internal/middleware/middleware.go:34)、[feishu_auth.go:155](/Users/zhanglikun/workspace/app/go-crawler/internal/service/feishu_auth.go:155)。

请求鉴权只检查 JWT 签名与过期时间，禁用/删除用户不会使现有 Token 失效。飞书登录查找已绑定用户时不检查 status，后续仍可签发新 Token，因而仅在后台禁用账号不能可靠阻止访问。

建议：密码登录、飞书登录和请求鉴权复用有效用户检查，加入会话撤销或用户 Token 版本；账户禁用时立即使相关会话失效。

**10. [P2] 带订单类型筛选的查询全量加载，缓存没有容量边界**

位置：[order.go:143](/Users/zhanglikun/workspace/app/go-crawler/internal/service/order.go:143)、[order.go:3338](/Users/zhanglikun/workspace/app/go-crawler/internal/service/order.go:3338)、[order.go:100](/Users/zhanglikun/workspace/app/go-crawler/internal/service/order.go:100)。

类型筛选路径先查询日期范围内的全部订单，完成归属计算后再在内存过滤和分页。因此即使只请求一页，冷缓存也要处理整个区间。缓存使用按日期区间区分的 sync.Map；一分钟 TTL 仅在再次访问同一个 key 时触发删除，另有显式全局清理，但没有定期清理或容量限制。历史日期回填及不断变化的日期组合可能长期持有多份完整订单集合。

建议：先记录真实请求的耗时、订单数、堆内存和慢查询；对可下推的条件做 SQL 过滤，复杂归属可落成可索引的结果表，再进行数据库分页。给缓存增加容量/字节上限与主动过期清理，历史任务可禁用缓存。未做生产压测，不能据此量化优化收益。

**11. [P1/P2] 凭证与会话文件进入 Git 管理范围**

位置：[config.yaml:26](/Users/zhanglikun/workspace/app/go-crawler/config.yaml:26)、[PROJECT_STATE.md:103](/Users/zhanglikun/workspace/app/go-crawler/docs/PROJECT_STATE.md:103)、[.gitignore](/Users/zhanglikun/workspace/app/go-crawler/.gitignore)。

已跟踪配置与说明文档包含凭证值，Git 还跟踪 `cookies/cookies.json` 和一个任务断点。报告不复述任何凭证，也未验证它们目前是否有效。

建议：配置使用无秘密的示例文件与运行时注入；Cookie、断点、缓存退出版本管理并设置合适文件权限。已经进入仓库、仍有效的凭证需要轮换；仅添加 .gitignore 不会移除已跟踪内容或历史记录。

**12. [P2] 构建与 SDK 引用不能在新环境可靠复现**

位置：[Makefile](/Users/zhanglikun/workspace/app/go-crawler/Makefile)、[hack.mk:10](/Users/zhanglikun/workspace/app/go-crawler/hack/hack.mk:10)、[Dockerfile:18](/Users/zhanglikun/workspace/app/go-crawler/Dockerfile:18)、[Dockerfile:36](/Users/zhanglikun/workspace/app/go-crawler/Dockerfile:36)、[go.mod:57](/Users/zhanglikun/workspace/app/go-crawler/go.mod:57)。

- `make -n build` 实际报错：两个 include 定义同名 build，最终目标依赖不存在的 `cli.install`。
- `make build-linux` 输出 `bin/linux_amd64/crawler`，Dockerfile 却复制 `bin/linux_amd64/main`。
- Dockerfile 在复制两个本地 replace 模块之前执行 `go mod download`；构建基础镜像标注 Go 1.21，而项目声明 Go 1.25，需统一并避免依赖隐式工具链下载。
- 巨量 SDK 在 Git 索引中为 gitlink，但没有 `.gitmodules` 映射。实际 `git submodule status` 报错；当前机器有源码不代表新 clone 能获得它。

建议：统一为一套可执行构建入口和产物名称；明确选择模块版本、正规子模块或完整 vendor 策略，并补充干净环境编译与容器构建 CI。未实际构建 Docker 镜像。

**13. [P1] 强制停止没有贯穿浏览器生命周期**

位置：[crawler.go:1068](/Users/zhanglikun/workspace/app/go-crawler/internal/logic/crawler/crawler.go:1068)、[crawler.go:1117](/Users/zhanglikun/workspace/app/go-crawler/internal/logic/crawler/crawler.go:1117)、[crawler_status.go:80](/Users/zhanglikun/workspace/app/go-crawler/internal/service/crawler_status.go:80)。

创建 Rod 浏览器时未绑定传入的任务 Context；本地依赖源码显示默认使用 Background。登录循环最长等待 8 分钟，也没有检查取消。强制下线会先清除在线状态并允许新任务启动，而旧任务可能仍在浏览器操作中；旧任务最终无条件 SetOffline，又可能清掉新任务状态及取消函数。

建议：浏览器和页面操作绑定任务 Context，并设置截止时间；取消后等待旧任务实际退出，再释放执行权。状态更新必须校验任务 ID，避免旧任务影响新任务。该项为代码路径核查，未启动浏览器复现。

**可维护性与测试的后续改进**

- [crawler.go](/Users/zhanglikun/workspace/app/go-crawler/internal/logic/crawler/crawler.go) 约 5,412 行，[finance_commission.go](/Users/zhanglikun/workspace/app/go-crawler/internal/service/finance_commission.go) 约 4,927 行，[order.go](/Users/zhanglikun/workspace/app/go-crawler/internal/service/order.go) 约 4,110 行。建议围绕浏览器会话、页面解析、调度、持久化、归属计算和查询分别拆分；先给稳定边界增加接口，不需要一次性重写。
- [cmd.go](/Users/zhanglikun/workspace/app/go-crawler/internal/cmd/cmd.go) 混合大量建表、改字段、业务数据修正、路由和 cron 注册。应拆出有版本记录的数据库迁移；当前启动时部分 UPDATE 会重复覆盖业务配置。逐步让启动入口只负责装配与启动。
- [crawler_test.go:16](/Users/zhanglikun/workspace/app/go-crawler/internal/logic/crawler/crawler_test.go:16) 默认会启动真实浏览器并访问平台；[zz_codex_li_commission_export_test.go:16](/Users/zhanglikun/workspace/app/go-crawler/internal/service/zz_codex_li_commission_export_test.go:16) 默认连接配置中的数据库并导出特定业务数据。应将这类人工诊断/集成任务移到显式命令或 build tag 后，日常单测默认离线。
- 补测试优先覆盖本次发现的死锁、手动调度、流水线状态迁移、暂停续跑、写入失败不推进断点、普通用户权限拒绝，以及 WebSocket 连接生命周期。现有解析与财务公式测试继续保留。

**本次验证结果**

- `go test -vet=off ./... -run '^$'`：通过，当前工作区所有主模块包与测试均可编译。
- `go vet ./...`：通过。
- `go test -vet=off ./... -skip 'TestProductDetailIntercept|TestCodexExportLiCommissionCurrent|Integration|Diagnostics' -count=1 -timeout=90s`：通过。
- 缓存死锁、内置任务手动触发、流水线重入的隔离测试：均复现相应问题；复现程序位于临时目录，没有加入业务代码。
- `make -n build`：失败；`git submodule status`：失败；`make -n build-linux`：揭示 Docker 产物路径不一致。
- 未进行线上接口测试、数据库集成测试、真实浏览器采集、压力测试或完整安全审计。现有测试通过不代表上述异常路径已被覆盖。

建议实施顺序：先修权限入口、缓存死锁与手动调度，再确保入库与断点一致、视频暂停可恢复；随后补齐构建 CI，依据实际指标优化订单查询与缓存，最后逐步拆分大文件和启动迁移逻辑。
