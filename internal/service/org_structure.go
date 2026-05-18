package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// OrgStructureService 组织架构服务
type OrgStructureService struct{}

func NewOrgStructureService() *OrgStructureService {
	return &OrgStructureService{}
}

// GetTree 获取组织架构树
func (s *OrgStructureService) GetTree(ctx context.Context) ([]model.OrgTreeNode, error) {
	// 查询所有组织节点
	var nodes []model.OrgStructure
	err := g.DB().Model("org_structure").
		Where("status", 1).
		Order("sort ASC, id ASC").
		Scan(&nodes)
	if err != nil {
		return nil, fmt.Errorf("查询组织架构失败: %v", err)
	}

	// 构建树形结构
	tree := s.buildTree(nodes, 0)
	return tree, nil
}

// buildTree 递归构建树形结构
func (s *OrgStructureService) buildTree(nodes []model.OrgStructure, parentId int64) []model.OrgTreeNode {
	var tree []model.OrgTreeNode
	for _, node := range nodes {
		if node.ParentId == parentId {
			treeNode := model.OrgTreeNode{
				Id:       node.Id,
				ParentId: node.ParentId,
				Name:     node.Name,
				Type:     node.Type,
				Code:     node.Code,
			}
			// 递归获取子节点
			children := s.buildTree(nodes, node.Id)
			if len(children) > 0 {
				treeNode.Children = children
			}
			tree = append(tree, treeNode)
		}
	}
	return tree
}

// GetList 获取组织节点列表（平铺）
func (s *OrgStructureService) GetList(ctx context.Context, req model.OrgListReq) ([]model.OrgStructure, error) {
	query := g.DB().Model("org_structure").Where("status", 1)

	if req.ParentId > 0 {
		query = query.Where("parent_id", req.ParentId)
	}
	if req.Type != "" {
		query = query.Where("type", req.Type)
	}
	if req.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+req.Keyword+"%")
	}

	var nodes []model.OrgStructure
	err := query.Order("sort ASC, id ASC").Scan(&nodes)
	if err != nil {
		return nil, fmt.Errorf("查询组织节点失败: %v", err)
	}

	return nodes, nil
}

// GetById 根据ID获取组织节点
func (s *OrgStructureService) GetById(ctx context.Context, id int64) (*model.OrgStructure, error) {
	var node model.OrgStructure
	err := g.DB().Model("org_structure").Where("id", id).Scan(&node)
	if err != nil {
		return nil, fmt.Errorf("查询组织节点失败: %v", err)
	}
	return &node, nil
}

// GetAncestors 获取节点的所有祖先节点（从根到当前节点的路径）
func (s *OrgStructureService) GetAncestors(ctx context.Context, id int64) ([]model.OrgStructure, error) {
	var ancestors []model.OrgStructure
	currentId := id

	for currentId > 0 {
		var node model.OrgStructure
		err := g.DB().Model("org_structure").Where("id", currentId).Scan(&node)
		if err != nil {
			return nil, fmt.Errorf("查询组织节点失败: %v", err)
		}
		if node.Id == 0 {
			break
		}
		ancestors = append([]model.OrgStructure{node}, ancestors...)
		currentId = node.ParentId
	}

	return ancestors, nil
}

// GetChildren 获取节点的直接子节点
func (s *OrgStructureService) GetChildren(ctx context.Context, parentId int64) ([]model.OrgStructure, error) {
	var children []model.OrgStructure
	err := g.DB().Model("org_structure").
		Where("parent_id", parentId).
		Where("status", 1).
		Order("sort ASC, id ASC").
		Scan(&children)
	if err != nil {
		return nil, fmt.Errorf("查询子节点失败: %v", err)
	}
	return children, nil
}

// Save 保存组织节点（新增或编辑）
func (s *OrgStructureService) Save(ctx context.Context, req model.OrgSaveReq) (int64, error) {
	if req.Id > 0 {
		// 编辑
		_, err := g.DB().Model("org_structure").
			Where("id", req.Id).
			Data(g.Map{
				"parent_id": req.ParentId,
				"name":      req.Name,
				"type":      req.Type,
				"code":      req.Code,
				"sort":      req.Sort,
				"remark":    req.Remark,
			}).
			Update()
		if err != nil {
			return 0, fmt.Errorf("更新组织节点失败: %v", err)
		}
		return req.Id, nil
	}

	// 新增
	lastInsertId, err := g.DB().Model("org_structure").
		Data(g.Map{
			"parent_id": req.ParentId,
			"name":      req.Name,
			"type":      req.Type,
			"code":      req.Code,
			"sort":      req.Sort,
			"remark":    req.Remark,
		}).
		InsertAndGetId()
	if err != nil {
		return 0, fmt.Errorf("新增组织节点失败: %v", err)
	}
	return lastInsertId, nil
}

// Delete 删除组织节点（软删除，同时删除子节点）
func (s *OrgStructureService) Delete(ctx context.Context, id int64) error {
	// 检查是否有子节点
	children, err := s.GetChildren(ctx, id)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return fmt.Errorf("该节点下有子节点，无法删除")
	}

	// 软删除
	_, err = g.DB().Model("org_structure").
		Where("id", id).
		Data(g.Map{"status": 0}).
		Update()
	if err != nil {
		return fmt.Errorf("删除组织节点失败: %v", err)
	}
	return nil
}

// BindShop 绑定店铺到组织节点（账号级别）
func (s *OrgStructureService) BindShop(ctx context.Context, shopId int, orgId int64) error {
	// 验证组织节点是否存在且为 account 类型
	node, err := s.GetById(ctx, orgId)
	if err != nil {
		return err
	}
	if node == nil || node.Id == 0 {
		return fmt.Errorf("组织节点不存在")
	}
	if node.Type != "account" {
		return fmt.Errorf("只能绑定到账号(account)级别的组织节点")
	}

	// 检查是否已绑定
	count, err := g.DB().Model("shop_org_relation").
		Where("shop_id", shopId).
		Where("org_id", orgId).
		Count()
	if err != nil {
		return fmt.Errorf("查询绑定关系失败: %v", err)
	}
	if count > 0 {
		return nil // 已绑定，直接返回成功
	}

	// 绑定
	_, err = g.DB().Model("shop_org_relation").
		Data(g.Map{
			"shop_id": shopId,
			"org_id":  orgId,
		}).
		Insert()
	if err != nil {
		return fmt.Errorf("绑定店铺失败: %v", err)
	}
	return nil
}

// UnbindShop 解绑店铺与组织节点
func (s *OrgStructureService) UnbindShop(ctx context.Context, shopId int, orgId int64) error {
	_, err := g.DB().Model("shop_org_relation").
		Where("shop_id", shopId).
		Where("org_id", orgId).
		Delete()
	if err != nil {
		return fmt.Errorf("解绑店铺失败: %v", err)
	}
	return nil
}

// GetShopOrgPath 获取店铺的组织路径（从根到账号的完整路径）
func (s *OrgStructureService) GetShopOrgPath(ctx context.Context, shopId int) ([]model.OrgStructure, error) {
	// 查询店铺关联的组织节点
	var relation model.ShopOrgRelation
	err := g.DB().Model("shop_org_relation").
		Where("shop_id", shopId).
		Scan(&relation)
	if err != nil {
		return nil, fmt.Errorf("查询店铺组织关联失败: %v", err)
	}
	if relation.Id == 0 {
		return nil, nil // 未绑定组织
	}

	// 获取组织路径
	return s.GetAncestors(ctx, relation.OrgId)
}

// GetShopsByOrgId 获取组织节点下的所有店铺ID
func (s *OrgStructureService) GetShopsByOrgId(ctx context.Context, orgId int64) ([]int, error) {
	// 获取该节点及其所有子节点的ID
	orgIds, err := s.getOrgAndDescendantIds(ctx, orgId)
	if err != nil {
		return nil, err
	}

	// 查询关联的店铺
	var shopIds []int
	err = g.DB().Model("shop_org_relation").
		Where("org_id IN(?)", orgIds).
		Fields("shop_id").
		Scan(&shopIds)
	if err != nil {
		return nil, fmt.Errorf("查询店铺失败: %v", err)
	}

	return shopIds, nil
}

// getOrgAndDescendantIds 获取节点及其所有后代节点的ID
func (s *OrgStructureService) getOrgAndDescendantIds(ctx context.Context, orgId int64) ([]int64, error) {
	var ids []int64
	ids = append(ids, orgId)

	// 获取直接子节点
	children, err := s.GetChildren(ctx, orgId)
	if err != nil {
		return nil, err
	}

	// 递归获取所有后代节点
	for _, child := range children {
		childIds, err := s.getOrgAndDescendantIds(ctx, child.Id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, childIds...)
	}

	return ids, nil
}

// GetOrgByType 获取指定类型的组织节点列表
func (s *OrgStructureService) GetOrgByType(ctx context.Context, orgType string) ([]model.OrgStructure, error) {
	var nodes []model.OrgStructure
	err := g.DB().Model("org_structure").
		Where("type", orgType).
		Where("status", 1).
		Order("sort ASC, id ASC").
		Scan(&nodes)
	if err != nil {
		return nil, fmt.Errorf("查询组织节点失败: %v", err)
	}
	return nodes, nil
}

// GetFullPathName 获取节点的完整路径名称（如：总公司 > 济宁 > 李艳雨 > 投放一组 > 李百川 > 抖音 > 账号1）
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
