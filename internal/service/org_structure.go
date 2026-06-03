package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// OrgStructureService 组织架构服务
// 数据来源：sys_company（公司）、sys_department（部门/组）、trader（投手/员工）
// ID 编码：Company(原ID) / Department(1000000+原ID) / Trader(2000000+原ID)
type OrgStructureService struct{}

func NewOrgStructureService() *OrgStructureService {
	return &OrgStructureService{}
}

// ==================== 树结构查询 ====================

// GetTree 获取组织架构树（公司→部门→投手）
func (s *OrgStructureService) GetTree(ctx context.Context) ([]model.OrgTreeNode, error) {
	// 1. 查询所有启用的公司
	var companies []model.SysCompany
	err := g.DB().Model("sys_company").
		Where("status", 1).
		Order("sort ASC, id ASC").
		Scan(&companies)
	if err != nil {
		return nil, fmt.Errorf("查询公司列表失败: %v", err)
	}

	// 2. 查询所有启用的部门
	var departments []model.SysDept
	err = g.DB().Model("sys_department").
		Where("status", 1).
		Order("sort ASC, id ASC").
		Scan(&departments)
	if err != nil {
		return nil, fmt.Errorf("查询部门列表失败: %v", err)
	}

	// 3. 查询所有启用的投手
	var traders []model.Trader
	err = g.DB().Model("trader").
		Where("status", 1).
		Order("id ASC").
		Scan(&traders)
	if err != nil {
		return nil, fmt.Errorf("查询投手列表失败: %v", err)
	}

	// 4. 组装树：公司（递归 parent_id） → 部门（递归 parent_id） → 投手
	tree := s.buildCompanyTree(0, companies, departments, traders)

	return tree, nil
}

// buildCompanyTree 递归构建公司层级树
// parentCompanyId: 上级公司的原始ID（0=顶级公司）
// companies: 所有公司
// depts: 所有部门
// traders: 所有投手
func (s *OrgStructureService) buildCompanyTree(parentCompanyId int64, companies []model.SysCompany, depts []model.SysDept, traders []model.Trader) []model.OrgTreeNode {
	var nodes []model.OrgTreeNode

	for _, company := range companies {
		if company.ParentId != parentCompanyId {
			continue
		}

		// 公司节点的 parentId：顶级→0，子级→父公司原始ID
		treeParentId := company.ParentId

		// 该公司下所有部门
		var companyDepts []model.SysDept
		for _, dept := range depts {
			if dept.CompanyId == company.Id {
				companyDepts = append(companyDepts, dept)
			}
		}

		companyNode := model.OrgTreeNode{
			Id:       company.Id,
			ParentId: treeParentId,
			Name:     company.Name,
			Type:     model.OrgTypeCompany,
			Code:     company.Code,
		}

		// 递归构建部门层级树
		deptChildren := s.buildDeptTree(company.Id, 0, companyDepts, traders)
		if len(deptChildren) > 0 {
			companyNode.Children = deptChildren
		}

		// 递归构建子公司
		subCompanies := s.buildCompanyTree(company.Id, companies, depts, traders)
		if len(subCompanies) > 0 {
			if companyNode.Children == nil {
				companyNode.Children = subCompanies
			} else {
				companyNode.Children = append(companyNode.Children, subCompanies...)
			}
		}

		nodes = append(nodes, companyNode)
	}

	return nodes
}

// buildDeptTree 递归构建部门层级树
// companyId: 所属公司原始ID
// parentDeptId: 上级部门的原始ID（0=顶级部门）
// depts: 该公司下所有部门
// traders: 所有投手
func (s *OrgStructureService) buildDeptTree(companyId, parentDeptId int64, depts []model.SysDept, traders []model.Trader) []model.OrgTreeNode {
	var nodes []model.OrgTreeNode

	for _, dept := range depts {
		if dept.ParentId != parentDeptId {
			continue
		}

		// 当前部门的 parentId：顶级部门→公司ID，子级→编码后的父部门ID
		var treeParentId int64
		if parentDeptId == 0 {
			treeParentId = companyId
		} else {
			treeParentId = model.EncodeDeptID(parentDeptId)
		}

		deptNode := model.OrgTreeNode{
			Id:       model.EncodeDeptID(dept.Id),
			ParentId: treeParentId,
			Name:     dept.Name,
			Type:     model.OrgTypeGroup,
		}

		// 挂载该部门下的投手
		var traderChildren []model.OrgTreeNode
		for _, trader := range traders {
			if trader.GroupId == dept.Id {
				traderChildren = append(traderChildren, model.OrgTreeNode{
					Id:       model.EncodeTraderID(trader.Id),
					ParentId: model.EncodeDeptID(dept.Id),
					Name:     trader.Name,
					Type:     model.OrgTypeEmployee,
				})
			}
		}
		if len(traderChildren) > 0 {
			deptNode.Children = traderChildren
		}

		// 递归构建当前部门下的子部门
		subDepts := s.buildDeptTree(companyId, dept.Id, depts, traders)
		if len(subDepts) > 0 {
			if deptNode.Children == nil {
				deptNode.Children = subDepts
			} else {
				deptNode.Children = append(deptNode.Children, subDepts...)
			}
		}

		nodes = append(nodes, deptNode)
	}

	return nodes
}

// ==================== 列表查询 ====================

// GetList 获取组织节点平铺列表
// - parentId=0：返回顶级公司列表（parent_id=0）
// - parentId=公司ID：返回该公司的子公司 + 顶级部门
// - parentId=部门编码ID：返回该部门下的子部门 + 投手列表
func (s *OrgStructureService) GetList(ctx context.Context, req model.OrgListReq) ([]model.OrgStructure, error) {
	switch {
	case req.ParentId == 0:
		return s.listCompanies(ctx, 0, req.Keyword, req.Type)
	case model.IsCompanyID(req.ParentId):
		return s.listCompanyChildren(ctx, req.ParentId, req.Keyword, req.Type)
	case model.IsDeptID(req.ParentId):
		return s.listDeptChildren(ctx, model.DecodeDeptID(req.ParentId), req.Keyword, req.Type)
	default:
		return nil, fmt.Errorf("无效的父节点ID: %d", req.ParentId)
	}
}

func (s *OrgStructureService) listCompanies(ctx context.Context, parentId int64, keyword, typ string) ([]model.OrgStructure, error) {
	if typ != "" && typ != model.OrgTypeCompany {
		return []model.OrgStructure{}, nil
	}
	query := g.DB().Model("sys_company").Where("status", 1)
	if parentId >= 0 {
		query = query.Where("parent_id", parentId)
	}
	if keyword != "" {
		query = query.WhereLike("name", "%"+keyword+"%")
	}
	var companies []model.SysCompany
	if err := query.Order("sort ASC, id ASC").Scan(&companies); err != nil {
		return nil, fmt.Errorf("查询公司列表失败: %v", err)
	}
	result := make([]model.OrgStructure, 0, len(companies))
	for _, c := range companies {
		result = append(result, model.OrgStructure{
			Id:        c.Id,
			ParentId:  c.ParentId,
			Name:      c.Name,
			Type:      model.OrgTypeCompany,
			Code:      c.Code,
			Sort:      c.Sort,
			Status:    c.Status,
			Remark:    c.Remark,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}
	return result, nil
}

// listCompanyChildren 获取公司下的子节点（子公司 + 顶级部门）
func (s *OrgStructureService) listCompanyChildren(ctx context.Context, companyId int64, keyword, typ string) ([]model.OrgStructure, error) {
	var result []model.OrgStructure

	// 1. 子公司（仅当 typ 为空或为 company）
	if typ == "" || typ == model.OrgTypeCompany {
		subs, err := s.listCompanies(ctx, companyId, keyword, typ)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}

	// 2. 顶级部门（仅当 typ 为空或为 group）
	if typ == "" || typ == model.OrgTypeGroup {
		depts, err := s.listDepartments(ctx, companyId, 0, keyword, typ)
		if err != nil {
			return nil, err
		}
		result = append(result, depts...)
	}

	return result, nil
}

func (s *OrgStructureService) listDepartments(ctx context.Context, companyId, parentDeptId int64, keyword, typ string) ([]model.OrgStructure, error) {
	if typ != "" && typ != model.OrgTypeGroup {
		return []model.OrgStructure{}, nil
	}
	query := g.DB().Model("sys_department").Where("status", 1).Where("company_id", companyId).Where("parent_id", parentDeptId)
	if keyword != "" {
		query = query.WhereLike("name", "%"+keyword+"%")
	}
	var depts []model.SysDept
	if err := query.Order("sort ASC, id ASC").Scan(&depts); err != nil {
		return nil, fmt.Errorf("查询部门列表失败: %v", err)
	}
	result := make([]model.OrgStructure, 0, len(depts))
	for _, d := range depts {
		// 部门 ParentId：顶级部门→公司ID，子级部门→编码后的父部门ID
		treeParentId := companyId
		if d.ParentId != 0 {
			treeParentId = model.EncodeDeptID(d.ParentId)
		}
		result = append(result, model.OrgStructure{
			Id:        model.EncodeDeptID(d.Id),
			ParentId:  treeParentId,
			Name:      d.Name,
			Type:      model.OrgTypeGroup,
			Sort:      d.Sort,
			Status:    d.Status,
			Remark:    d.Remark,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		})
	}
	return result, nil
}

// listDeptChildren 获取部门下的直接子节点（子部门 + 投手）
func (s *OrgStructureService) listDeptChildren(ctx context.Context, deptId int64, keyword, typ string) ([]model.OrgStructure, error) {
	var result []model.OrgStructure

	// 1. 查询子部门（仅当 typ 为空或为 group）
	if typ == "" || typ == model.OrgTypeGroup {
		var childDepts []model.SysDept
		err := g.DB().Model("sys_department").
			Where("status", 1).
			Where("parent_id", deptId).
			Order("sort ASC, id ASC").
			Scan(&childDepts)
		if err != nil {
			return nil, fmt.Errorf("查询子部门失败: %v", err)
		}
		for _, d := range childDepts {
			if keyword != "" && !containsKeyword(d.Name, keyword) {
				continue
			}
			result = append(result, model.OrgStructure{
				Id:        model.EncodeDeptID(d.Id),
				ParentId:  model.EncodeDeptID(deptId),
				Name:      d.Name,
				Type:      model.OrgTypeGroup,
				Sort:      d.Sort,
				Status:    d.Status,
				Remark:    d.Remark,
				CreatedAt: d.CreatedAt,
				UpdatedAt: d.UpdatedAt,
			})
		}
	}

	// 2. 查询投手（仅当 typ 为空或为 employee）
	if typ == "" || typ == model.OrgTypeEmployee {
		var traders []model.Trader
		err := g.DB().Model("trader").
			Where("status", 1).
			Where("group_id", deptId).
			Order("id ASC").
			Scan(&traders)
		if err != nil {
			return nil, fmt.Errorf("查询投手失败: %v", err)
		}
		for _, t := range traders {
			if keyword != "" && !containsKeyword(t.Name, keyword) {
				continue
			}
			result = append(result, model.OrgStructure{
				Id:        model.EncodeTraderID(t.Id),
				ParentId:  model.EncodeDeptID(deptId),
				Name:      t.Name,
				Type:      model.OrgTypeEmployee,
				Status:    t.Status,
				Remark:    t.Remark,
				CreatedAt: t.CreatedAt,
				UpdatedAt: t.UpdatedAt,
			})
		}
	}

	return result, nil
}

// containsKeyword 简单关键词匹配
func containsKeyword(s, keyword string) bool {
	return len(keyword) == 0 || len(s) >= len(keyword) && stringContains(s, keyword)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (s *OrgStructureService) listTraders(ctx context.Context, groupId int64, keyword, typ string) ([]model.OrgStructure, error) {
	if typ != "" && typ != model.OrgTypeEmployee {
		return []model.OrgStructure{}, nil
	}
	query := g.DB().Model("trader").Where("status", 1).Where("group_id", groupId)
	if keyword != "" {
		query = query.WhereLike("name", "%"+keyword+"%")
	}
	var traders []model.Trader
	if err := query.Order("id ASC").Scan(&traders); err != nil {
		return nil, fmt.Errorf("查询投手列表失败: %v", err)
	}
	result := make([]model.OrgStructure, 0, len(traders))
	for _, t := range traders {
		result = append(result, model.OrgStructure{
			Id:        model.EncodeTraderID(t.Id),
			ParentId:  model.EncodeDeptID(t.GroupId),
			Name:      t.Name,
			Type:      model.OrgTypeEmployee,
			Status:    t.Status,
			Remark:    t.Remark,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		})
	}
	return result, nil
}

// ==================== 单节点查询 ====================

// GetById 根据全局 ID 获取组织节点（跨表查询）
func (s *OrgStructureService) GetById(ctx context.Context, id int64) (*model.OrgStructure, error) {
	switch {
	case model.IsCompanyID(id):
		return s.getCompany(ctx, id)
	case model.IsDeptID(id):
		return s.getDepartment(ctx, model.DecodeDeptID(id))
	case model.IsTraderID(id):
		return s.getTrader(ctx, model.DecodeTraderID(id))
	default:
		return nil, fmt.Errorf("无效的节点ID: %d", id)
	}
}

func (s *OrgStructureService) getCompany(ctx context.Context, id int64) (*model.OrgStructure, error) {
	var c model.SysCompany
	if err := g.DB().Model("sys_company").Where("id", id).Scan(&c); err != nil {
		return nil, fmt.Errorf("查询公司失败: %v", err)
	}
	if c.Id == 0 {
		return &model.OrgStructure{}, nil
	}
	return &model.OrgStructure{
		Id:        c.Id,
		ParentId:  c.ParentId,
		Name:      c.Name,
		Type:      model.OrgTypeCompany,
		Code:      c.Code,
		Sort:      c.Sort,
		Status:    c.Status,
		Remark:    c.Remark,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}, nil
}

func (s *OrgStructureService) getDepartment(ctx context.Context, deptId int64) (*model.OrgStructure, error) {
	var d model.SysDept
	if err := g.DB().Model("sys_department").Where("id", deptId).Scan(&d); err != nil {
		return nil, fmt.Errorf("查询部门失败: %v", err)
	}
	if d.Id == 0 {
		return &model.OrgStructure{}, nil
	}
	// 部门 ParentId：顶级部门→公司ID，子级部门→编码后的父部门ID
	treeParentId := d.CompanyId
	if d.ParentId != 0 {
		treeParentId = model.EncodeDeptID(d.ParentId)
	}
	return &model.OrgStructure{
		Id:        model.EncodeDeptID(d.Id),
		ParentId:  treeParentId,
		Name:      d.Name,
		Type:      model.OrgTypeGroup,
		Sort:      d.Sort,
		Status:    d.Status,
		Remark:    d.Remark,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}, nil
}

// buildCompanyAncestors 从公司向上追溯父公司链（不包含当前公司本身）
// 返回从顶级公司开始的祖先列表
func (s *OrgStructureService) buildCompanyAncestors(ctx context.Context, companyId int64) []model.OrgStructure {
	var ancestors []model.OrgStructure
	var companyChain []model.SysCompany
	currentId := companyId

	for {
		var c model.SysCompany
		err := g.DB().Model("sys_company").Where("id", currentId).Scan(&c)
		if err != nil || c.Id == 0 || c.ParentId == 0 {
			break
		}
		var parent model.SysCompany
		err = g.DB().Model("sys_company").Where("id", c.ParentId).Scan(&parent)
		if err != nil || parent.Id == 0 {
			break
		}
		companyChain = append([]model.SysCompany{parent}, companyChain...)
		currentId = parent.Id
	}

	for _, c := range companyChain {
		ancestors = append(ancestors, model.OrgStructure{
			Id:        c.Id,
			ParentId:  c.ParentId,
			Name:      c.Name,
			Type:      model.OrgTypeCompany,
			Code:      c.Code,
			Sort:      c.Sort,
			Status:    c.Status,
			Remark:    c.Remark,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}

	return ancestors
}

// buildDeptAncestors 从部门向上追溯祖先路径：公司 → 父部门(们)
// 返回从公司开始的祖先列表（不包含当前部门本身）
func (s *OrgStructureService) buildDeptAncestors(ctx context.Context, deptId int64) []model.OrgStructure {
	var ancestors []model.OrgStructure

	// 1. 追溯所有父部门
	var parentChain []model.SysDept
	currentId := deptId
	for {
		var d model.SysDept
		err := g.DB().Model("sys_department").Where("id", currentId).Scan(&d)
		if err != nil || d.Id == 0 {
			break
		}
		if d.ParentId == 0 {
			// 达到顶级部门，停止
			break
		}
		// 查询父部门
		var parent model.SysDept
		err = g.DB().Model("sys_department").Where("id", d.ParentId).Scan(&parent)
		if err != nil || parent.Id == 0 {
			break
		}
		parentChain = append([]model.SysDept{parent}, parentChain...)
		currentId = parent.Id
	}

	// 2. 获取公司（从最后一个父部门或当前部门的 company_id）
	var companyId int64
	if len(parentChain) > 0 {
		companyId = parentChain[0].CompanyId
	} else {
		var d model.SysDept
		if err := g.DB().Model("sys_department").Where("id", deptId).Scan(&d); err == nil {
			companyId = d.CompanyId
		}
	}

	if companyId > 0 {
		companyNode, err := s.getCompany(ctx, companyId)
		if err == nil && companyNode != nil && companyNode.Id > 0 {
			ancestors = append(ancestors, *companyNode)
		}
	}

	// 3. 添加父部门链
	for _, p := range parentChain {
		treeParentId := p.CompanyId
		if p.ParentId != 0 {
			treeParentId = model.EncodeDeptID(p.ParentId)
		}
		ancestors = append(ancestors, model.OrgStructure{
			Id:        model.EncodeDeptID(p.Id),
			ParentId:  treeParentId,
			Name:      p.Name,
			Type:      model.OrgTypeGroup,
			Sort:      p.Sort,
			Status:    p.Status,
			Remark:    p.Remark,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}

	return ancestors
}

func (s *OrgStructureService) getTrader(ctx context.Context, traderId int64) (*model.OrgStructure, error) {
	var t model.Trader
	if err := g.DB().Model("trader").Where("id", traderId).Scan(&t); err != nil {
		return nil, fmt.Errorf("查询投手失败: %v", err)
	}
	if t.Id == 0 {
		return &model.OrgStructure{}, nil
	}
	return &model.OrgStructure{
		Id:        model.EncodeTraderID(t.Id),
		ParentId:  model.EncodeDeptID(t.GroupId),
		Name:      t.Name,
		Type:      model.OrgTypeEmployee,
		Status:    t.Status,
		Remark:    t.Remark,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}, nil
}

// ==================== 子节点查询 ====================

// GetChildren 获取节点直接子节点
func (s *OrgStructureService) GetChildren(ctx context.Context, parentId int64) ([]model.OrgStructure, error) {
	// 统一走 GetList（带上 parentId 即可）
	req := model.OrgListReq{ParentId: parentId}
	return s.GetList(ctx, req)
}

// ==================== 祖先路径 ====================

// GetAncestors 获取节点祖先路径（跨表查询，从根到当前）
func (s *OrgStructureService) GetAncestors(ctx context.Context, id int64) ([]model.OrgStructure, error) {
	switch {
	case model.IsCompanyID(id):
		node, err := s.getCompany(ctx, id)
		if err != nil {
			return nil, err
		}
		if node.Id == 0 {
			return nil, nil
		}
		// 向上追溯父公司链
		ancestors := s.buildCompanyAncestors(ctx, id)
		ancestors = append(ancestors, *node)
		return ancestors, nil

	case model.IsDeptID(id):
		deptNode, err := s.getDepartment(ctx, model.DecodeDeptID(id))
		if err != nil {
			return nil, err
		}
		if deptNode.Id == 0 {
			return nil, nil
		}
		// 从当前部门向上遍历：部门 → 父部门 → ... → 公司
		ancestors := s.buildDeptAncestors(ctx, model.DecodeDeptID(id))
		// ancestors 是倒序的（公司在前），追加当前部门
		ancestors = append(ancestors, *deptNode)
		return ancestors, nil

	case model.IsTraderID(id):
		traderNode, err := s.getTrader(ctx, model.DecodeTraderID(id))
		if err != nil {
			return nil, err
		}
		if traderNode.Id == 0 {
			return nil, nil
		}
		// 通过 ParentId（部门编码ID）拿到部门
		deptNode, err := s.GetById(ctx, traderNode.ParentId)
		if err != nil {
			return nil, err
		}

		ancestors := []model.OrgStructure{}
		if deptNode != nil && deptNode.Id > 0 {
			companyNode, err := s.getCompany(ctx, deptNode.ParentId)
			if err != nil {
				return nil, err
			}
			if companyNode != nil && companyNode.Id > 0 {
				ancestors = append(ancestors, *companyNode)
			}
			ancestors = append(ancestors, *deptNode)
		}
		ancestors = append(ancestors, *traderNode)
		return ancestors, nil

	default:
		return nil, fmt.Errorf("无效的节点ID: %d", id)
	}
}

// GetFullPathName 获取节点完整路径名称
func (s *OrgStructureService) GetFullPathName(ctx context.Context, id int64) (string, error) {
	ancestors, err := s.GetAncestors(ctx, id)
	if err != nil {
		return "", err
	}
	path := ""
	for i, node := range ancestors {
		if i > 0 {
			path += " > "
		}
		path += node.Name
	}
	return path, nil
}

// GetOrgByType 获取指定类型的组织节点列表（全量）
func (s *OrgStructureService) GetOrgByType(ctx context.Context, orgType string) ([]model.OrgStructure, error) {
	switch orgType {
	case model.OrgTypeCompany:
		return s.listCompanies(ctx, -1, "", orgType)
	case model.OrgTypeGroup:
		// 查全部部门
		var depts []model.SysDept
		if err := g.DB().Model("sys_department").Where("status", 1).Order("sort ASC, id ASC").Scan(&depts); err != nil {
			return nil, fmt.Errorf("查询部门列表失败: %v", err)
		}
		result := make([]model.OrgStructure, 0, len(depts))
		for _, d := range depts {
			treeParentId := d.CompanyId
			if d.ParentId != 0 {
				treeParentId = model.EncodeDeptID(d.ParentId)
			}
			result = append(result, model.OrgStructure{
				Id:        model.EncodeDeptID(d.Id),
				ParentId:  treeParentId,
				Name:      d.Name,
				Type:      model.OrgTypeGroup,
				Sort:      d.Sort,
				Status:    d.Status,
				Remark:    d.Remark,
				CreatedAt: d.CreatedAt,
				UpdatedAt: d.UpdatedAt,
			})
		}
		return result, nil
	case model.OrgTypeEmployee:
		var traders []model.Trader
		if err := g.DB().Model("trader").Where("status", 1).Order("id ASC").Scan(&traders); err != nil {
			return nil, fmt.Errorf("查询投手列表失败: %v", err)
		}
		result := make([]model.OrgStructure, 0, len(traders))
		for _, t := range traders {
			result = append(result, model.OrgStructure{
				Id:        model.EncodeTraderID(t.Id),
				ParentId:  model.EncodeDeptID(t.GroupId),
				Name:      t.Name,
				Type:      model.OrgTypeEmployee,
				Status:    t.Status,
				Remark:    t.Remark,
				CreatedAt: t.CreatedAt,
				UpdatedAt: t.UpdatedAt,
			})
		}
		return result, nil
	default:
		return nil, fmt.Errorf("不支持的节点类型: %s", orgType)
	}
}

// ==================== 增/删/改 ====================

// Save 保存组织节点（根据 type 路由到不同表）
func (s *OrgStructureService) Save(ctx context.Context, req model.OrgSaveReq) (int64, error) {
	switch req.Type {
	case model.OrgTypeCompany:
		return s.saveCompany(ctx, req)
	case model.OrgTypeGroup:
		return s.saveDepartment(ctx, req)
	case model.OrgTypeEmployee:
		return s.saveTrader(ctx, req)
	default:
		return 0, fmt.Errorf("不支持的节点类型: %s（仅支持 company/group/employee）", req.Type)
	}
}

func (s *OrgStructureService) saveCompany(ctx context.Context, req model.OrgSaveReq) (int64, error) {
	if req.Id > 0 {
		_, err := g.DB().Model("sys_company").
			Where("id", req.Id).
			Data(g.Map{
				"name":   req.Name,
				"code":   req.Code,
				"sort":   req.Sort,
				"remark": req.Remark,
			}).
			Update()
		if err != nil {
			return 0, fmt.Errorf("更新公司失败: %v", err)
		}
		return req.Id, nil
	}
	id, err := g.DB().Model("sys_company").
		Data(g.Map{
			"name":   req.Name,
			"code":   req.Code,
			"sort":   req.Sort,
			"remark": req.Remark,
			"status": 1,
		}).
		InsertAndGetId()
	if err != nil {
		return 0, fmt.Errorf("新增公司失败: %v", err)
	}
	return id, nil
}

func (s *OrgStructureService) saveDepartment(ctx context.Context, req model.OrgSaveReq) (int64, error) {
	companyId := req.ParentId // 部门的 parentId 即公司 ID
	if companyId <= 0 || !model.IsCompanyID(companyId) {
		return 0, fmt.Errorf("部门必须指定有效的所属公司ID")
	}
	if req.Id > 0 {
		realId := model.DecodeDeptID(req.Id)
		_, err := g.DB().Model("sys_department").
			Where("id", realId).
			Data(g.Map{
				"company_id": companyId,
				"name":       req.Name,
				"sort":       req.Sort,
				"remark":     req.Remark,
			}).
			Update()
		if err != nil {
			return 0, fmt.Errorf("更新部门失败: %v", err)
		}
		return req.Id, nil
	}
	realId, err := g.DB().Model("sys_department").
		Data(g.Map{
			"company_id": companyId,
			"name":       req.Name,
			"sort":       req.Sort,
			"remark":     req.Remark,
			"status":     1,
		}).
		InsertAndGetId()
	if err != nil {
		return 0, fmt.Errorf("新增部门失败: %v", err)
	}
	return model.EncodeDeptID(realId), nil
}

func (s *OrgStructureService) saveTrader(ctx context.Context, req model.OrgSaveReq) (int64, error) {
	// parentId 是部门的编码 ID
	if !model.IsDeptID(req.ParentId) {
		return 0, fmt.Errorf("投手的父节点必须是部门")
	}
	realGroupId := model.DecodeDeptID(req.ParentId)

	// 从部门获取 company_id
	var dept model.SysDept
	if err := g.DB().Model("sys_department").Where("id", realGroupId).Scan(&dept); err != nil || dept.Id == 0 {
		return 0, fmt.Errorf("所属部门不存在")
	}

	if req.Id > 0 {
		realId := model.DecodeTraderID(req.Id)
		_, err := g.DB().Model("trader").
			Where("id", realId).
			Data(g.Map{
				"name":       req.Name,
				"company_id": dept.CompanyId,
				"group_id":   realGroupId,
				"remark":     req.Remark,
			}).
			Update()
		if err != nil {
			return 0, fmt.Errorf("更新投手失败: %v", err)
		}
		return req.Id, nil
	}
	realId, err := g.DB().Model("trader").
		Data(g.Map{
			"name":       req.Name,
			"company_id": dept.CompanyId,
			"group_id":   realGroupId,
			"role":       1,
			"remark":     req.Remark,
			"status":     1,
		}).
		InsertAndGetId()
	if err != nil {
		return 0, fmt.Errorf("新增投手失败: %v", err)
	}
	return model.EncodeTraderID(realId), nil
}

// Delete 软删除组织节点
func (s *OrgStructureService) Delete(ctx context.Context, id int64) error {
	// 检查是否有子节点
	children, err := s.GetChildren(ctx, id)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return fmt.Errorf("该节点下有子节点，无法删除")
	}

	switch {
	case model.IsCompanyID(id):
		_, err = g.DB().Model("sys_company").Where("id", id).Data(g.Map{"status": 0}).Update()
		if err != nil {
			return fmt.Errorf("删除公司失败: %v", err)
		}
	case model.IsDeptID(id):
		_, err = g.DB().Model("sys_department").Where("id", model.DecodeDeptID(id)).Data(g.Map{"status": 0}).Update()
		if err != nil {
			return fmt.Errorf("删除部门失败: %v", err)
		}
	case model.IsTraderID(id):
		realId := model.DecodeTraderID(id)
		// 删除关联的店铺关系
		_, _ = g.DB().Model("trader_account_relation").Where("trader_id", realId).Delete()
		_, err = g.DB().Model("trader").Where("id", realId).Data(g.Map{"status": 0, "updated_at": gtime.Now()}).Update()
		if err != nil {
			return fmt.Errorf("删除投手失败: %v", err)
		}
	default:
		return fmt.Errorf("无效的节点ID: %d", id)
	}
	return nil
}

// ==================== 账号绑定 ====================

// BindShop 绑定账号到组织节点（投手级别）
func (s *OrgStructureService) BindShop(ctx context.Context, accountId int, orgId int64) error {
	// 验证 orgId 必须是投手 (employee) 类型
	if !model.IsTraderID(orgId) {
		return fmt.Errorf("只能将账号绑定到投手(employee)级别的组织节点")
	}
	realTraderId := model.DecodeTraderID(orgId)

	// 验证投手是否存在
	var trader model.Trader
	if err := g.DB().Model("trader").Where("id", realTraderId).Where("status", 1).Scan(&trader); err != nil || trader.Id == 0 {
		return fmt.Errorf("投手不存在或已禁用")
	}

	// 检查是否已绑定
	count, err := g.DB().Model("trader_account_relation").
		Where("trader_id", realTraderId).
		Where("account_id", accountId).
		Count()
	if err != nil {
		return fmt.Errorf("查询绑定关系失败: %v", err)
	}
	if count > 0 {
		return nil // 已绑定
	}

	// 写入关联
	_, err = g.DB().Model("trader_account_relation").
		Data(g.Map{
			"trader_id": realTraderId,
			"account_id":   accountId,
		}).
		Insert()
	if err != nil {
		return fmt.Errorf("绑定账号失败: %v", err)
	}

	// 更新账号绑定状态
	_, _ = g.DB().Model("account").Where("id", accountId).Data(g.Map{"bind_status": 1}).Update()
	return nil
}

// UnbindShop 解绑账号
func (s *OrgStructureService) UnbindShop(ctx context.Context, accountId int, orgId int64) error {
	if !model.IsTraderID(orgId) {
		return fmt.Errorf("只能从投手(employee)节点解绑账号")
	}
	realTraderId := model.DecodeTraderID(orgId)

	result, err := g.DB().Model("trader_account_relation").
		Where("trader_id", realTraderId).
		Where("account_id", accountId).
		Delete()
	if err != nil {
		return fmt.Errorf("解绑账号失败: %v", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("绑定关系不存在")
	}

	// 检查店铺是否还被其他投手绑定
	otherCount, _ := g.DB().Model("trader_account_relation").Where("account_id", accountId).Count()
	if otherCount == 0 {
		_, _ = g.DB().Model("account").Where("id", accountId).Data(g.Map{"bind_status": 0}).Update()
	}
	return nil
}

// GetShopOrgPath 获取账号的组织路径
func (s *OrgStructureService) GetShopOrgPath(ctx context.Context, accountId int) ([]model.OrgStructure, error) {
	// 通过 trader_account_relation 找到投手
	var rel model.TraderShopRelation
	err := g.DB().Model("trader_account_relation").
		Where("account_id", accountId).
		Scan(&rel)
	if err != nil {
		return nil, fmt.Errorf("查询账号关联失败: %v", err)
	}
	if rel.Id == 0 {
		return nil, nil // 未绑定
	}

	// 通过投手 ID 获取组织路径
	return s.GetAncestors(ctx, model.EncodeTraderID(rel.TraderId))
}

// ==================== 账号 ID 获取（统计基础） ====================

// GetShopsByOrgId 获取组织节点下的所有账号ID（递归下级）
func (s *OrgStructureService) GetShopsByOrgId(ctx context.Context, orgId int64) ([]int, error) {
	switch {
	case model.IsCompanyID(orgId):
		return s.getCompanyShops(ctx, orgId)
	case model.IsDeptID(orgId):
		return s.getDepartmentShops(ctx, model.DecodeDeptID(orgId))
	case model.IsTraderID(orgId):
		return s.getTraderShops(ctx, model.DecodeTraderID(orgId))
	default:
		return nil, fmt.Errorf("无效的节点ID: %d", orgId)
	}
}

func (s *OrgStructureService) getCompanyShops(ctx context.Context, companyId int64) ([]int, error) {
	// 公司 → 所有部门 → 所有投手 → 店铺
	var deptIds []int64
	if err := g.DB().Model("sys_department").
		Where("company_id", companyId).Where("status", 1).
		Fields("id").Scan(&deptIds); err != nil {
		return nil, fmt.Errorf("查询公司部门失败: %v", err)
	}
	if len(deptIds) == 0 {
		return nil, nil
	}

	var traderIds []int64
	if err := g.DB().Model("trader").
		Where("group_id IN(?)", deptIds).Where("status", 1).
		Fields("id").Scan(&traderIds); err != nil {
		return nil, fmt.Errorf("查询投手失败: %v", err)
	}
	if len(traderIds) == 0 {
		return nil, nil
	}

	return s.getAccountIdsByTraderIds(ctx, traderIds)
}

func (s *OrgStructureService) getDepartmentShops(ctx context.Context, deptId int64) ([]int, error) {
	// 部门 → 所有投手 → 店铺
	var traderIds []int64
	if err := g.DB().Model("trader").
		Where("group_id", deptId).Where("status", 1).
		Fields("id").Scan(&traderIds); err != nil {
		return nil, fmt.Errorf("查询部门投手失败: %v", err)
	}
	if len(traderIds) == 0 {
		return nil, nil
	}
	return s.getAccountIdsByTraderIds(ctx, traderIds)
}

func (s *OrgStructureService) getTraderShops(ctx context.Context, traderId int64) ([]int, error) {
	var accountIds []int
	err := g.DB().Model("trader_account_relation").
		Where("trader_id", traderId).
		Fields("account_id").
		Scan(&accountIds)
	if err != nil {
		return nil, fmt.Errorf("查询投手账号失败: %v", err)
	}
	return accountIds, nil
}

func (s *OrgStructureService) getAccountIdsByTraderIds(ctx context.Context, traderIds []int64) ([]int, error) {
	var accountIds []int
	err := g.DB().Model("trader_account_relation").
		Where("trader_id IN(?)", traderIds).
		Fields("account_id").
		Scan(&accountIds)
	if err != nil {
		return nil, fmt.Errorf("查询账号关联失败: %v", err)
	}
	return accountIds, nil
}

// ==================== 佣金统计 ====================

// GetOrgStats 获取组织节点佣金统计（含直接子节点）
func (s *OrgStructureService) GetOrgStats(ctx context.Context, orgId int64, startDate, endDate string) (*model.OrgStatsNode, error) {
	// 1. 获取当前节点信息
	node, err := s.GetById(ctx, orgId)
	if err != nil {
		return nil, fmt.Errorf("查询组织节点失败: %v", err)
	}
	if node == nil || node.Id == 0 {
		return nil, fmt.Errorf("组织节点不存在")
	}

	// 2. 获取该节点下所有账号ID
	accountIds, err := s.GetShopsByOrgId(ctx, orgId)
	if err != nil {
		return nil, fmt.Errorf("获取账号列表失败: %v", err)
	}

	// 3. 计算汇总
	summary, err := s.calcShopStats(ctx, accountIds, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("计算统计失败: %v", err)
	}

	// 4. 获取直接子节点
	childNodes, err := s.GetChildren(ctx, orgId)
	if err != nil {
		return nil, fmt.Errorf("获取子节点失败: %v", err)
	}

	childStats := make([]model.OrgStatsNode, 0, len(childNodes))
	for _, child := range childNodes {
		childAccountIds, err := s.GetShopsByOrgId(ctx, child.Id)
		if err != nil {
			continue
		}
		childSummary, err := s.calcShopStats(ctx, childAccountIds, startDate, endDate)
		if err != nil {
			continue
		}
		childStats = append(childStats, model.OrgStatsNode{
			OrgId:   child.Id,
			OrgName: child.Name,
			OrgType: child.Type,
			Summary: *childSummary,
		})
	}

	return &model.OrgStatsNode{
		OrgId:    node.Id,
		OrgName:  node.Name,
		OrgType:  node.Type,
		Summary:  *summary,
		Children: childStats,
	}, nil
}

// calcShopStats 计算账号列表的佣金统计
func (s *OrgStructureService) calcShopStats(ctx context.Context, accountIds []int, startDate, endDate string) (*model.StatsSummary, error) {
	if len(accountIds) == 0 {
		return &model.StatsSummary{}, nil
	}

	sql := `
		SELECT
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			SELECT
				o1.main_order_no,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON o1.seller_code = g.goods_code AND g.status = 1
			LEFT JOIN account a ON o1.talent_id = a.uid
			WHERE s.id IN (?)
			  AND o1.pay_finish_time >= ?
			  AND o1.pay_finish_time < ?
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  AND o1.after_sale_status NOT IN ('退款成功', '已全额退款', '待退货')
			  AND o1.order_status != '已关闭'
			GROUP BY o1.main_order_no
		) o
	`

	result, err := g.DB().GetOne(ctx, sql, accountIds, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("查询统计失败: %v", err)
	}

	payAmount := result["pay_amount"].Float64()
	costAmount := result["cost_amount"].Float64()
	reserveAmount := result["reserve_amount"].Float64()

	return &model.StatsSummary{
		OrderCount:    result["order_count"].Int(),
		PayAmount:     roundFloat2(payAmount, 2),
		CostAmount:    roundFloat2(costAmount, 2),
		ReserveAmount: roundFloat2(reserveAmount, 2),
		Commission:    roundFloat2(payAmount-reserveAmount-costAmount, 2),
	}, nil
}

// GetCommissionStats 获取达人佣金统计（按投手分组）
func (s *OrgStructureService) GetCommissionStats(ctx context.Context, orgId int64, startDate, endDate, keyword string, page, pageSize int) (*model.CommissionStatsRes, error) {
	// 1. 获取节点信息
	node, err := s.GetById(ctx, orgId)
	if err != nil {
		return nil, fmt.Errorf("查询组织节点失败: %v", err)
	}
	if node == nil || node.Id == 0 {
		return nil, fmt.Errorf("组织节点不存在")
	}

	// 2. 获取该节点下所有账号ID
	accountIds, err := s.GetShopsByOrgId(ctx, orgId)
	if err != nil {
		return nil, fmt.Errorf("获取账号列表失败: %v", err)
	}

	// 3. 计算汇总
	summary, err := s.calcShopStats(ctx, accountIds, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("计算汇总失败: %v", err)
	}

	// 4. 查询达人明细
	var items []model.CommissionStatItem
	total := 0

	if len(accountIds) > 0 {
		// 先查总数
		countSQL := `
			SELECT COUNT(DISTINCT t.id) AS cnt
			FROM (
				SELECT o1.talent_id
				FROM order_info o1
				LEFT JOIN account a ON o1.talent_id = a.uid
				WHERE s.id IN (?)
				  AND o1.pay_finish_time >= ?
				  AND o1.pay_finish_time < ?
				  AND o1.main_order_no IS NOT NULL
				  AND o1.main_order_no != ''
				  AND o1.after_sale_status NOT IN ('退款成功', '已全额退款', '待退货')
				  AND o1.order_status != '已关闭'
				GROUP BY o1.talent_id
			) o
			LEFT JOIN account a ON o.talent_id = a.uid
			LEFT JOIN trader_account_relation tsr ON tsr.account_id = a.id
			LEFT JOIN trader t ON tsr.trader_id = t.id AND t.status = 1
			WHERE t.id IS NOT NULL
		`
		if keyword != "" {
			countSQL += " AND t.name LIKE '%" + keyword + "%'"
		}

		countResult, err := g.DB().GetOne(ctx, countSQL, accountIds, startDate, endDate)
		if err == nil {
			total = countResult["cnt"].Int()
		}

		// 查询明细
		sql := `
			SELECT
				t.id AS trader_id,
				t.name AS trader_name,
				COUNT(DISTINCT o.main_order_no) AS order_count,
				COALESCE(SUM(o.product_num), 0) AS product_num,
				COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
				COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
				COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
			FROM (
				SELECT
					o1.main_order_no,
					o1.talent_id,
					SUM(COALESCE(o1.product_num, 0)) AS product_num,
					SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
					SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
					SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
				FROM order_info o1
				LEFT JOIN goods g ON o1.seller_code = g.goods_code AND g.status = 1
				LEFT JOIN account a ON o1.talent_id = a.uid
				WHERE s.id IN (?)
				  AND o1.pay_finish_time >= ?
				  AND o1.pay_finish_time < ?
				  AND o1.main_order_no IS NOT NULL
				  AND o1.main_order_no != ''
				  AND o1.after_sale_status NOT IN ('退款成功', '已全额退款', '待退货')
				  AND o1.order_status != '已关闭'
				GROUP BY o1.main_order_no, o1.talent_id
			) o
			LEFT JOIN account a ON o.talent_id = a.uid
			LEFT JOIN trader_account_relation tsr ON tsr.account_id = a.id
			LEFT JOIN trader t ON tsr.trader_id = t.id AND t.status = 1
			WHERE t.id IS NOT NULL
		`
		if keyword != "" {
			sql += " AND t.name LIKE '%" + keyword + "%'"
		}
		sql += " GROUP BY t.id, t.name ORDER BY order_count DESC"
		sql += fmt.Sprintf(" LIMIT %d, %d", (page-1)*pageSize, pageSize)

		result, err := g.DB().GetAll(ctx, sql, accountIds, startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf("查询达人佣金明细失败: %v", err)
		}

		items = make([]model.CommissionStatItem, 0, len(result))
		for _, row := range result {
			payAmount := row["pay_amount"].Float64()
			costAmount := row["cost_amount"].Float64()
			reserveAmount := row["reserve_amount"].Float64()
			commission := roundFloat2(payAmount-reserveAmount-costAmount, 2)
			items = append(items, model.CommissionStatItem{
				TraderId:   row["trader_id"].Int64(),
				TraderName: row["trader_name"].String(),
				OrderCount: row["order_count"].Int(),
				ProductNum: row["product_num"].Int(),
				PayAmount:  roundFloat2(payAmount, 2),
				Commission: commission,
			})
		}
	}

	return &model.CommissionStatsRes{
		OrgId:     node.Id,
		OrgName:   node.Name,
		Summary:   *summary,
		Items:     items,
		Total:     total,
		StartDate: startDate,
		EndDate:   endDate,
	}, nil
}

// roundFloat2 四舍五入保留指定小数位
func roundFloat2(val float64, precision int) float64 {
	ratio := 1.0
	for i := 0; i < precision; i++ {
		ratio *= 10
	}
	return float64(int(val*ratio+0.5)) / ratio
}
