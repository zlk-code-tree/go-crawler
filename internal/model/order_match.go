package model

// ============================================
// 达人订单匹配 相关模型
// ============================================

// OrderMatchResult 匹配结果
type OrderMatchResult struct {
	Total     int      `json:"total"`     // 精选联盟订单总数
	Matched   int      `json:"matched"`   // 成功匹配回填数
	Skipped   int      `json:"skipped"`   // 原订单已有达人跳过数
	Unmatched int      `json:"unmatched"` // 未匹配数
	Errors    []string `json:"errors"`
}
