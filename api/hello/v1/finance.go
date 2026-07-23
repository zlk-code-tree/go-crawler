package v1

import (
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// ==================== 佣金统计 ====================

// FinanceStatsReq 佣金统计请求
type FinanceStatsReq struct {
	g.Meta    `path:"/finance/commission-stats" method:"get" tags:"Finance" summary:"获取佣金统计"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型：today/yesterday/week/month/custom"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"开始日期（custom时必填，格式：2006-01-02）"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"结束日期（custom时必填，格式：2006-01-02）"`
	NodeId    int64  `json:"nodeId" in:"query" d:"0" dc:"组织节点ID（0=全部）"`
	NodeType  string `json:"nodeType" in:"query" d:"company" v:"required|in:company,department,trader#节点类型不能为空|节点类型非法" dc:"节点类型：company/department/trader"`
	Platform  string `json:"platform" in:"query" d:"" dc:"平台筛选：douyin/kuaishou/shipinhao（空=全部）"`
}

// FinanceStatsRes 佣金统计响应
type FinanceStatsRes = model.FinanceStatsRes

// ==================== 下级节点查询 ====================

// FinanceChildrenReq 获取下级节点请求
type FinanceChildrenReq struct {
	g.Meta    `path:"/finance/commission-children" method:"get" tags:"Finance" summary:"获取下级节点统计"`
	NodeId    int64  `json:"nodeId" in:"query" v:"required#节点ID不能为空" dc:"节点ID"`
	NodeType  string `json:"nodeType" in:"query" v:"required|in:company,department,trader#节点类型不能为空|节点类型非法" dc:"节点类型：company/department/trader"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"结束日期"`
	Platform  string `json:"platform" in:"query" d:"" dc:"平台筛选：douyin/kuaishou/shipinhao（空=全部）"`
}

// FinanceChildrenRes 获取下级节点响应
type FinanceChildrenRes = model.FinanceChildrenRes

// ==================== 投手详情 ====================

// FinanceTraderDetailReq 投手详情统计请求
type FinanceTraderDetailReq struct {
	g.Meta    `path:"/finance/trader-detail" method:"get" tags:"Finance" summary:"获取投手详情统计"`
	TraderId  int64  `json:"traderId" in:"query" v:"required#投手ID不能为空" dc:"投手ID"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"结束日期"`
}

// FinanceTraderDetailRes 投手详情统计响应
type FinanceTraderDetailRes = model.FinanceTraderDetailRes

// ==================== 店铺明细 ====================

// FinanceShopDetailReq 账号明细统计请求
type FinanceShopDetailReq struct {
	g.Meta    `path:"/finance/shop-detail" method:"get" tags:"Finance" summary:"获取账号明细统计"`
	AccountId int    `json:"accountId" in:"query" v:"required#账号ID不能为空" dc:"账号ID"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"结束日期"`
}

// FinanceShopDetailRes 账号明细统计响应
type FinanceShopDetailRes = model.FinanceShopDetailRes

// ==================== 手动触发聚合 ====================

// FinanceAggregateReq 手动触发聚合请求
type FinanceAggregateReq struct {
	g.Meta `path:"/finance/aggregate" method:"post" tags:"Finance" summary:"手动触发日聚合任务"`
	Date   string `json:"date" v:"required#日期不能为空" dc:"统计日期（格式：2006-01-02）"`
}

// FinanceAggregateRes 手动触发聚合响应
type FinanceAggregateRes struct {
	g.Meta  `mime:"application/json"`
	Success bool   `json:"success" dc:"是否成功"`
	Message string `json:"message" dc:"提示信息"`
}

// ==================== 财务归属口径设置 ====================

// FinanceQianchuanPlanInferenceSettingReq 获取千川计划反推空白单开关
type FinanceQianchuanPlanInferenceSettingReq struct {
	g.Meta `path:"/finance/qianchuan-plan-inference-setting" method:"get" tags:"Finance" summary:"获取千川计划反推空白单开关"`
}

// FinanceQianchuanPlanInferenceSettingRes 千川计划反推空白单开关响应
type FinanceQianchuanPlanInferenceSettingRes struct {
	g.Meta  `mime:"application/json"`
	Enabled bool   `json:"enabled" dc:"是否开启"`
	Message string `json:"message" dc:"提示信息"`
}

// FinanceSetQianchuanPlanInferenceSettingReq 设置千川计划反推空白单开关
type FinanceSetQianchuanPlanInferenceSettingReq struct {
	g.Meta  `path:"/finance/qianchuan-plan-inference-setting" method:"post" tags:"Finance" summary:"设置千川计划反推空白单开关"`
	Enabled bool `json:"enabled" dc:"是否开启"`
}

// FinanceSetQianchuanPlanInferenceSettingRes 设置千川计划反推空白单开关响应
type FinanceSetQianchuanPlanInferenceSettingRes struct {
	g.Meta  `mime:"application/json"`
	Success bool   `json:"success" dc:"是否成功"`
	Enabled bool   `json:"enabled" dc:"是否开启"`
	Message string `json:"message" dc:"提示信息"`
}

// ==================== 历史数据回填 ====================

// FinanceBackfillReq 历史数据回填请求
type FinanceBackfillReq struct {
	g.Meta    `path:"/finance/backfill" method:"post" tags:"Finance" summary:"历史数据回填"`
	StartDate string `json:"startDate" v:"required#开始日期不能为空" dc:"开始日期（格式：2006-01-02）"`
	EndDate   string `json:"endDate" v:"required#结束日期不能为空" dc:"结束日期（格式：2006-01-02）"`
}

// FinanceBackfillRes 历史数据回填响应
type FinanceBackfillRes struct {
	g.Meta  `mime:"application/json"`
	Success bool   `json:"success" dc:"是否成功"`
	Message string `json:"message" dc:"提示信息"`
}

// ==================== 投手订单列表 ====================

// FinanceTraderOrdersReq 投手订单列表请求
type FinanceTraderOrdersReq struct {
	g.Meta    `path:"/finance/trader-orders" method:"get" tags:"Finance" summary:"获取投手订单列表"`
	TraderId  int64  `json:"traderId" in:"query" v:"required#投手ID不能为空" dc:"投手ID"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"结束日期"`
	Page      int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize  int    `json:"pageSize" in:"query" d:"20" dc:"每页数量"`
}

// FinanceTraderOrdersRes 投手订单列表响应
type FinanceTraderOrdersRes = model.FinanceTraderOrdersRes

// FinanceTraderPlatformStatsReq 投手平台统计明细请求
type FinanceTraderPlatformStatsReq struct {
	g.Meta    `path:"/finance/trader-platform-stats" method:"get" tags:"Finance" summary:"获取投手平台统计明细"`
	TraderId  int64  `json:"traderId" in:"query" v:"required#投手ID不能为空" dc:"投手ID"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"结束日期"`
}

// FinanceTraderPlatformStatsRes 投手平台统计明细响应
type FinanceTraderPlatformStatsRes = model.FinanceTraderPlatformStatsRes

// ==================== 投手订单导出 ====================

// FinanceTraderExportReq 导出投手订单请求
type FinanceTraderExportReq struct {
	g.Meta    `path:"/finance/trader-export" method:"get" tags:"Finance" summary:"导出投手订单明细"`
	TraderId  int64  `json:"traderId" in:"query" v:"required#投手ID不能为空" dc:"投手ID"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" dc:"结束日期"`
}

// ==================== 账号订单列表 ====================

// FinanceAccountOrdersReq 账号订单列表请求
type FinanceAccountOrdersReq struct {
	g.Meta    `path:"/finance/account-orders" method:"get" tags:"Finance" summary:"获取账号订单列表"`
	AccountId int    `json:"accountId" in:"query" v:"required#账号ID不能为空" dc:"账号ID"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" dc:"结束日期"`
	Page      int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize  int    `json:"pageSize" in:"query" d:"20" dc:"每页数量"`
}

// FinanceAccountOrdersRes 账号订单列表响应
type FinanceAccountOrdersRes = model.FinanceAccountOrdersRes

// ==================== 空白单归属明细 ====================

// FinanceBlankOrderAttributionReq 空白单归属明细请求
type FinanceBlankOrderAttributionReq struct {
	g.Meta    `path:"/finance/blank-order-attributions" method:"get" tags:"Finance" summary:"获取空白单归属明细"`
	NodeId    int64  `json:"nodeId" in:"query" v:"required#节点ID不能为空" dc:"节点ID"`
	NodeType  string `json:"nodeType" in:"query" v:"required#节点类型不能为空" dc:"节点类型"`
	AccountId int    `json:"accountId" in:"query" dc:"账号ID，账号行点击时传入"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" dc:"结束日期"`
	Page      int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize  int    `json:"pageSize" in:"query" d:"20" dc:"每页数量"`
}

// FinanceBlankOrderAttributionRes 空白单归属明细响应
type FinanceBlankOrderAttributionRes = model.FinanceBlankOrderAttributionRes

// FinanceBlankOrderAttributionExportReq 导出空白单归属明细请求
type FinanceBlankOrderAttributionExportReq struct {
	g.Meta    `path:"/finance/blank-order-attributions-export" method:"get" tags:"Finance" summary:"导出空白单归属明细"`
	NodeId    int64  `json:"nodeId" in:"query" v:"required#节点ID不能为空" dc:"节点ID"`
	NodeType  string `json:"nodeType" in:"query" v:"required#节点类型不能为空" dc:"节点类型"`
	AccountId int    `json:"accountId" in:"query" dc:"账号ID，账号行导出时传入"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" dc:"结束日期"`
}

// ==================== 有效订单归属核验明细 ====================

// FinanceValidOrderAttributionReq 有效订单归属核验请求
type FinanceValidOrderAttributionReq struct {
	g.Meta    `path:"/finance/valid-order-attributions" method:"get" tags:"Finance" summary:"获取有效订单归属核验明细"`
	NodeId    int64  `json:"nodeId" in:"query" v:"required#节点ID不能为空" dc:"节点ID"`
	NodeType  string `json:"nodeType" in:"query" v:"required#节点类型不能为空" dc:"节点类型"`
	AccountId int    `json:"accountId" in:"query" dc:"账号ID，账号行点击时传入"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" dc:"结束日期"`
	Page      int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize  int    `json:"pageSize" in:"query" d:"20" dc:"每页数量"`
}

// FinanceValidOrderAttributionRes 有效订单归属核验响应
type FinanceValidOrderAttributionRes = model.FinanceValidOrderAttributionRes

// FinanceValidOrderAttributionExportReq 导出有效订单归属核验请求
type FinanceValidOrderAttributionExportReq struct {
	g.Meta    `path:"/finance/valid-order-attributions-export" method:"get" tags:"Finance" summary:"导出有效订单归属核验明细"`
	NodeId    int64  `json:"nodeId" in:"query" v:"required#节点ID不能为空" dc:"节点ID"`
	NodeType  string `json:"nodeType" in:"query" v:"required#节点类型不能为空" dc:"节点类型"`
	AccountId int    `json:"accountId" in:"query" dc:"账号ID，账号行导出时传入"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型"`
	StartDate string `json:"startDate" in:"query" dc:"开始日期"`
	EndDate   string `json:"endDate" in:"query" dc:"结束日期"`
}
