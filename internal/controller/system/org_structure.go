package system

import (
	"context"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
)

type OrgStructureController struct{}

func NewOrgStructureController() *OrgStructureController {
	return &OrgStructureController{}
}

// GetTree 获取组织架构树 GET /api/org/tree
func (c *OrgStructureController) GetTree(ctx context.Context, req *v1.OrgTreeReq) (res *v1.OrgTreeRes, err error) {
	orgService := service.NewOrgStructureService()

	tree, err := orgService.GetTree(ctx)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.OrgTreeRes{
		Tree: tree,
	}
	return
}

// GetList 获取组织节点列表 GET /api/org/list
func (c *OrgStructureController) GetList(ctx context.Context, req *v1.OrgListReq) (res *v1.OrgListRes, err error) {
	orgService := service.NewOrgStructureService()

	listReq := model.OrgListReq{
		ParentId: req.ParentId,
		Type:     req.Type,
		Keyword:  req.Keyword,
	}

	nodes, err := orgService.GetList(ctx, listReq)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.OrgListRes{
		List: nodes,
	}
	return
}

// GetDetail 获取组织节点详情 GET /api/org/detail
func (c *OrgStructureController) GetDetail(ctx context.Context, req *v1.OrgDetailReq) (res *v1.OrgDetailRes, err error) {
	orgService := service.NewOrgStructureService()

	node, err := orgService.GetById(ctx, req.Id)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}
	if node == nil || node.Id == 0 {
		return nil, gerror.New("组织节点不存在")
	}

	// 获取完整路径
	pathName, _ := orgService.GetFullPathName(ctx, req.Id)

	res = &v1.OrgDetailRes{
		Data:     node,
		PathName: pathName,
	}
	return
}

// Save 保存组织节点 POST /api/org/save
func (c *OrgStructureController) Save(ctx context.Context, req *v1.OrgSaveReq) (res *v1.OrgSaveRes, err error) {
	orgService := service.NewOrgStructureService()

	saveReq := model.OrgSaveReq{
		Id:       req.Id,
		ParentId: req.ParentId,
		Name:     req.Name,
		Type:     req.Type,
		Code:     req.Code,
		Sort:     req.Sort,
		Remark:   req.Remark,
	}

	id, err := orgService.Save(ctx, saveReq)
	if err != nil {
		return nil, gerror.New("保存失败: " + err.Error())
	}

	res = &v1.OrgSaveRes{
		Id: id,
	}
	return
}

// Delete 删除组织节点 POST /api/org/delete
func (c *OrgStructureController) Delete(ctx context.Context, req *v1.OrgDeleteReq) (res *v1.OrgDeleteRes, err error) {
	orgService := service.NewOrgStructureService()

	err = orgService.Delete(ctx, req.Id)
	if err != nil {
		return nil, gerror.New("删除失败: " + err.Error())
	}

	res = &v1.OrgDeleteRes{}
	return
}

// GetChildren 获取子节点列表 GET /api/org/children
func (c *OrgStructureController) GetChildren(ctx context.Context, req *v1.OrgChildrenReq) (res *v1.OrgChildrenRes, err error) {
	orgService := service.NewOrgStructureService()

	children, err := orgService.GetChildren(ctx, req.ParentId)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.OrgChildrenRes{
		List: children,
	}
	return
}

// GetAncestors 获取祖先节点路径 GET /api/org/ancestors
func (c *OrgStructureController) GetAncestors(ctx context.Context, req *v1.OrgAncestorsReq) (res *v1.OrgAncestorsRes, err error) {
	orgService := service.NewOrgStructureService()

	ancestors, err := orgService.GetAncestors(ctx, req.Id)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.OrgAncestorsRes{
		List: ancestors,
	}
	return
}

// BindShop 绑定账号到组织节点 POST /api/org/bind-shop
func (c *OrgStructureController) BindShop(ctx context.Context, req *v1.OrgBindShopReq) (res *v1.OrgBindShopRes, err error) {
	orgService := service.NewOrgStructureService()

	err = orgService.BindShop(ctx, req.AccountId, req.OrgId)
	if err != nil {
		return nil, gerror.New("绑定失败: " + err.Error())
	}

	res = &v1.OrgBindShopRes{}
	return
}

// UnbindShop 解绑账号与组织节点 POST /api/org/unbind-shop
func (c *OrgStructureController) UnbindShop(ctx context.Context, req *v1.OrgUnbindShopReq) (res *v1.OrgUnbindShopRes, err error) {
	orgService := service.NewOrgStructureService()

	err = orgService.UnbindShop(ctx, req.AccountId, req.OrgId)
	if err != nil {
		return nil, gerror.New("解绑失败: " + err.Error())
	}

	res = &v1.OrgUnbindShopRes{}
	return
}

// GetShopOrgPath 获取账号的组织路径 GET /api/org/shop-path
func (c *OrgStructureController) GetShopOrgPath(ctx context.Context, req *v1.OrgShopPathReq) (res *v1.OrgShopPathRes, err error) {
	orgService := service.NewOrgStructureService()

	path, err := orgService.GetShopOrgPath(ctx, req.AccountId)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	// 构建路径名称
	pathName := ""
	for i, node := range path {
		if i > 0 {
			pathName += " > "
		}
		pathName += node.Name
	}

	res = &v1.OrgShopPathRes{
		Path:     path,
		PathName: pathName,
	}
	return
}

// Stats 组织节点佣金统计 GET /api/org/stats
func (c *OrgStructureController) Stats(ctx context.Context, req *v1.OrgStatsReq) (res *v1.OrgStatsRes, err error) {
	orgService := service.NewOrgStructureService()

	// 解析时间范围（复用OrderAnalysisService的方法）
	orderService := service.NewOrderAnalysisService()
	startDate, endDate := orderService.ParseTimeRange(req.Preset, req.StartDate, req.EndDate)

	// 获取统计
	statsNode, err := orgService.GetOrgStats(ctx, req.OrgId, startDate, endDate)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.OrgStatsRes{
		OrgId:     statsNode.OrgId,
		OrgName:   statsNode.OrgName,
		OrgType:   statsNode.OrgType,
		StartDate: startDate,
		EndDate:   endDate,
		Summary:   statsNode.Summary,
		Children:  statsNode.Children,
	}
	return
}

// CommissionStats 达人佣金统计 GET /api/org/commission-stats
func (c *OrgStructureController) CommissionStats(ctx context.Context, req *v1.OrgCommissionStatsReq) (res *v1.OrgCommissionStatsRes, err error) {
	orgService := service.NewOrgStructureService()

	// 解析时间范围
	orderService := service.NewOrderAnalysisService()
	startDate, endDate := orderService.ParseTimeRange(req.Preset, req.StartDate, req.EndDate)

	stats, err := orgService.GetCommissionStats(ctx, req.OrgId, startDate, endDate, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.OrgCommissionStatsRes{
		OrgId:     stats.OrgId,
		OrgName:   stats.OrgName,
		Summary:   stats.Summary,
		Items:     stats.Items,
		Total:     stats.Total,
		StartDate: stats.StartDate,
		EndDate:   stats.EndDate,
	}
	return
}
