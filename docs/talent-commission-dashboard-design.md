# 达人佣金数据看板 - 设计方案

> 状态：方案评审中
> 决策口径：主数据源 `order_info` + JOIN `external_talent_registry` 用佣金率算**应计佣金**
> 阶段：先做阶段 1（KPI 总览 + 达人佣金排行表）

## 1. 背景

现有系统有"达人佣金"数据但分散在多处，缺少一个以**达人(talent)为第一维度**、用**佣金率算应计佣金**的对账看板。

- `CommissionStats`（`finance_commission.go`）：投手(trader)维度，非达人维度，且佣金是预聚合值不是佣金率算的。
- `GetShopTalentStats`（`order_analysis.go:848`）：达人维度，但**只支持单店铺**，佣金是 `pay-reserve-cost` 算出的"实际佣金"，不是佣金率算的应计佣金。
- `external_talent_registry`：存每个达人的 `online_commission_rate`（线上佣金率），按达人+商品粒度，是应计佣金计算的关键，目前没被看板利用。

目标：补一个"每个达人的应计佣金"看板，用对账和达人合作决策。

## 2. 关键概念澄清

| 概念 | 含义 | 来源 |
|---|---|---|
| trader（投手） | 公司内部员工，负责运营 | `trader` / `finance_daily_stats.trader_id` |
| talent（达人） | 外部带货者，按佣金率结算 | `order_info.talent_id` / `external_talent_registry.uid` |
| account（账号） | 一个达人在某平台的账号 | `account.uid` 即达人 UID |

本看板"达人"= talent。`account.uid` 与 `external_talent_registry.uid` 同源（达人 UID），可作为兜底关联。

## 3. 数据口径

### 3.1 应计佣金（本看板主口径）

```
应计佣金 = SUM(order_info.order_pay_amount × external_talent_registry.online_commission_rate)
```

JOIN 路径（优先达人+商品级，兜底达人级）：

```sql
order_info o
LEFT JOIN external_talent_registry r
  ON o.talent_id = r.uid
 AND o.product_id = r.product_id        -- 达人+商品级佣金率（最准）
 AND r.status = 1
```

- 命中 `r`：用 `online_commission_rate` 算应计佣金
- 未命中（该达人/商品没登记佣金率）：该订单佣金记为"未配置"，单列统计，**不按 0 也不按默认率猜**

### 3.2 与现有"实际佣金"并存

看板排行表同时展示两列：
- **应计佣金**：佣金率算的（本看板主口径）
- **实际佣金**：`pay_amount - reserve_amount - cost_amount`（沿用 `GetShopTalentStats` 算法，cost 经 `goods.base_price` join）

差额 = 应计 - 实际，便于发现佣金对账偏差。

> 你选的是"只展示应计佣金"。考虑对账场景，我建议保留实际佣金做对照列，**但不影响主排序**（默认按应计佣金排）。若你坚持只一列，去掉实际佣金列即可。

### 3.3 有效订单口径

沿用 `financeValidOrderCondition("o1")`（现有有效订单过滤逻辑，`order_analysis.go`），保证与现有统计口径一致。

## 4. 阶段 1 交付：KPI + 达人排行

### 4.1 页面布局

```
┌─────────────────────────────────────────────────────────┐
│ 筛选：平台 | 时间区间(默认近30天) | 团队/小组 | 店铺(可选)   │
├─────────────────────────────────────────────────────────┤
│ ① KPI 卡片（5 个）                                        │
│   达人数 | 总应计佣金 | 总销售额 | 平均佣金率 | 人均应计佣金  │
├────────────────────────────────────────────────────────┤
│ ② 达人佣金排行表（分页、可排序）                              │
│  达人 | 所属小组 | 平台 | 销售额 | 应计佣金 | 实际佣金 |    │
│       | 订单数 | 商品数 | 佣金率 | 环比 | [趋势]            │
└─────────────────────────────────────────────────────────┘
```

### 4.2 后端接口（阶段 1 仅需 1 个）

`GET /api/talent-commission/dashboard`

**入参**
| 参数 | 说明 |
|---|---|
| platform | douyin/kuaishou/shipinhao，空=全部 |
| startDate / endDate | 日期范围，默认近 30 天 |
| groupName | 团队/小组筛选（可选） |
| shopName | 店铺筛选（可选） |
| page / pageSize | 分页，默认 1/20 |

**出参**
```json
{
  "summary": {
    "talentCount": 120,          // 达人数
    "totalAccruedCommission": 0, // 总应计佣金
    "totalPayAmount": 0,         // 总销售额
    "avgCommissionRate": 0,      // 平均佣金率
    "perTalentCommission": 0     // 人均应计佣金
  },
  "list": [
    {
      "talentId": "123", "talentName": "达人A", "groupName": "一组",
      "platform": "douyin", "shopName": "xx旗舰店",
      "payAmount": 0, "accruedCommission": 0, "actualCommission": 0,
      "orderCount": 0, "goodsCount": 0,
      "commissionRate": 0.03, "rateConfigured": true,
      "prevAccruedCommission": 0, "commissionDelta": 0  // 环比
    }
  ],
  "total": 120, "page": 1, "pageSize": 20
}
```

### 4.3 核心 SQL（达人聚合 + 环比）

达人维度聚合（当期）：

```sql
SELECT
  o.talent_id,
  COALESCE(MAX(r.talent_name), MAX(o.talent_name), '未知达人') AS talent_name,
  COALESCE(MAX(a.operator), '未分配小组') AS group_name,
  MAX(o.shop_name) AS shop_name,
  MAX(o.platform) AS platform,
  COUNT(DISTINCT o.main_order_no) AS order_count,
  COALESCE(SUM(o.product_num), 0) AS goods_count,
  COALESCE(SUM(o.order_pay_amount), 0) AS pay_amount,
  -- 应计佣金：用佣金率算
  COALESCE(SUM(o.order_pay_amount * r.online_commission_rate), 0) AS accrued_commission,
  -- 实际佣金：pay - reserve - cost（cost 经 goods.base_price join）
  COALESCE(SUM(o.order_pay_amount - o.reserve_amount - g.base_price * o.product_num), 0) AS actual_commission,
  -- 是否配置了佣金率
  CASE WHEN MAX(r.online_commission_rate) IS NOT NULL THEN 1 ELSE 0 END AS rate_configured,
  -- 平均佣金率（加权）
  CASE WHEN SUM(o.order_pay_amount) > 0
       THEN SUM(o.order_pay_amount * r.online_commission_rate) / SUM(o.order_pay_amount)
       ELSE 0 END AS commission_rate
FROM order_info o
LEFT JOIN external_talent_registry r
  ON o.talent_id = r.uid AND o.product_id = r.product_id AND r.status = 1
LEFT JOIN account a ON o.talent_id = a.uid
LEFT JOIN goods g ON o.seller_code = g.goods_code AND g.status = 1
WHERE o.pay_finish_time >= ? AND o.pay_finish_time < ?
  AND (o.shop_name = ? OR ? = '')      -- shopName 可空
  AND (a.operator = ? OR ? = '')         -- groupName 可空
  AND <有效订单条件 financeValidOrderCondition("o")>
GROUP BY o.talent_id
ORDER BY accrued_commission DESC
LIMIT ? OFFSET ?
```

环比：用同样 SQL 跑上一周期（`startDate - 周期长度` 到 `startDate`），按 `talent_id` 在后端 join 出 `commissionDelta`。

> 注意：`external_talent_registry` 按 `(uid, product_id)` 可能多条（同一达人不同商品）。`SUM(o.order_pay_amount * r.online_commission_rate)` 在逐行 join 后求和是正确的——每条订单只匹配它自己的商品佣金率。

### 4.4 数据质量处理

1. **talent_id 为空**：单独统计"未归属达人"的订单金额，单独一行展示，不混入达人排行。
2. **佣金率未配置**：`rate_configured=false` 的订单，应计佣金记 0，行内高亮提示"未配置佣金率"。
3. **达人名归一**：优先 `external_talent_registry.talent_name`，兜底 `order_info.talent_name`，再兜底 `account.account_name`。
4. **平均佣金率**：用销售额加权平均，非简单算术平均，避免小额高佣金率达人被放大。

## 5. 实施清单（阶段 1）

### 后端
- [ ] 新建 `api/hello/v1/talent_commission.go`：定义 `TalentCommissionDashboardReq/Res`
- [ ] 新建 `internal/controller/talent_commission/talent_commission.go`：路由处理
- [ ] 在 `internal/service/order_analysis.go` 新增 `GetTalentCommissionDashboard(ctx, req)`：跨店铺达人聚合 + 环比
  - 复用 `financeValidOrderCondition`、`roundFloat`、时间区间解析（现有 `getTimeRangeByType`）
  - 关键改造：相对 `GetShopTalentStats`，去掉 `shop_name` 强制过滤、加 `external_talent_registry` join、加应计佣金
- [ ] 在 `internal/cmd/cmd.go` 注册路由（若手动注册）或依赖 GoFrame 规范路由

### 前端
- [ ] 新建 `apps/web-ele/src/views/douyin/talent-commission/index.vue`
- [ ] KPI 片组件 + 排行表（Element Plus Table，可排序分页）
- [ ] 顶部筛选（平台下拉、日期范围、团队、店铺）
- [ ] 新建 `apps/web-ele/src/api/finance/talent-commission.ts`：调 dashboard 接口
- [ ] 菜单注册（`migration_douyin_monitor_menu.sql` 同款方式）

### 数据准备（前置）
- [ ] 确认 `external_talent_registry` 的 `(uid, product_id)` 是否覆盖了主要达人+商品；未覆盖的看板会显示"未配置"，需运营补录佣金率
- [ ] 确认 `order_info.platform` 字段是否填充（看板按平台筛选用）

## 6. 风险与取舍

| 风险 | 影响 | 对策 |
|---|---|---|
| `external_talent_registry` 覆盖率低 | 大量达人显示"未配置佣金率"，应计佣金失真 | 上线前导出"未配置达人+商品"清单给运营补录 |
| order_info 数据量大，实时聚合慢 | 接口超时 | 已有 `idx_talent_paytime` 索引；先测近 30 天耗时，>2s 考虑加 `talent_id, pay_finish_time` 复合索引或预聚合 |
| 达人名多源不一致 | 排行表达人名混乱 | 统一归一规则（external_talent_registry 优先），文档记录 |
| 一达人多账号/多平台 | 同一达人被拆成多行 | 阶段1按 talent_id 聚合（跨平台合并）；若需按平台拆，阶段2加分组 |

## 7. 阶段 2 预告（本次不做，先确认阶段1）

- 单达人趋势折线（日/周应计佣金曲线）
- 达人明细抽屉：Top 商品佣金贡献、平台分布
- ROI 标签（高产/低产/待激活）
- 导出 Excel、飞书告警

## 8. 待你确认

1. 是否保留"实际佣金"对照列（建议保留，便于对账）？
2. 排行默认按"应计佣金"降序，OK？
3. 时间默认范围近 30 天，OK？
4. `external_talent_registry` 的佣金率覆盖率大概多少？若很低，应计佣金会大面积为 0，需要先补数据再看板才有效。