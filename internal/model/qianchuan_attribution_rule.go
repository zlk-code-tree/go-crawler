package model

import "time"

// QianchuanAttributionRule 千川消耗临时归属规则
type QianchuanAttributionRule struct {
	Id                 int64      `json:"id"`
	AdvertiserId       int64      `json:"advertiserId"`
	AdvertiserName     string     `json:"advertiserName"`
	SourceAccountNames string     `json:"sourceAccountNames"`
	SourceTraderId     int64      `json:"sourceTraderId"`
	SourceTraderName   string     `json:"sourceTraderName"`
	TargetTraderId     int64      `json:"targetTraderId"`
	TargetTraderName   string     `json:"targetTraderName"`
	EffectiveStartDate string     `json:"effectiveStartDate"`
	EffectiveEndDate   string     `json:"effectiveEndDate"`
	IsPermanent        bool       `json:"isPermanent"`
	Reason             string     `json:"reason"`
	Status             int        `json:"status"`
	CreatedAt          *time.Time `json:"createdAt"`
	UpdatedAt          *time.Time `json:"updatedAt"`
}

// QianchuanAttributionRuleListReq 列表请求
type QianchuanAttributionRuleListReq struct {
	Page         int    `json:"page" v:"required|min:1#页码不能为空|页码最小为1"`
	PageSize     int    `json:"pageSize" v:"required|min:1|max:100#每页数量不能为空|每页数量最小为1|每页数量最大为100"`
	Keyword      string `json:"keyword"`
	AdvertiserId int64  `json:"advertiserId"`
	Status       int    `json:"status"`
}

// QianchuanAttributionRuleSaveReq 保存请求
type QianchuanAttributionRuleSaveReq struct {
	Id                 int64  `json:"id"`
	AdvertiserId       int64  `json:"advertiserId" v:"required#千川户不能为空"`
	AdvertiserName     string `json:"advertiserName"`
	SourceAccountNames string `json:"sourceAccountNames" v:"required#命中达人不能为空"`
	SourceTraderId     int64  `json:"sourceTraderId" v:"required#原归属组不能为空"`
	TargetTraderId     int64  `json:"targetTraderId" v:"required#目标归属组不能为空"`
	EffectiveStartDate string `json:"effectiveStartDate" v:"required#生效开始日期不能为空"`
	EffectiveEndDate   string `json:"effectiveEndDate"`
	IsPermanent        bool   `json:"isPermanent"`
	Reason             string `json:"reason"`
	Status             int    `json:"status" d:"1"`
}
