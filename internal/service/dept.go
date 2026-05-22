package service

import (
	"context"
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
)

// DeptService 部门相关业务逻辑
type DeptService struct{}

func NewDeptService() *DeptService {
	return &DeptService{}
}

// Tree 查询部门树（用于父级部门下拉选择）
func (s *DeptService) Tree(ctx context.Context, companyId int64) ([]g.Map, error) {
	// 查询该公司下所有部门（状态正常的）
	query := `
		SELECT id, name, parent_id, level, sort
		FROM sys_department
		WHERE company_id = ? AND status = 1
		ORDER BY sort ASC, id ASC
	`
	result, err := g.DB().GetAll(ctx, query, companyId)
	if err != nil {
		return nil, err
	}

	// 转换为 map 列表
	allDepts := make([]g.Map, len(result))
	for i, row := range result {
		allDepts[i] = g.Map{
			"id":       row["id"],
			"name":     row["name"],
			"parentId": row["parent_id"],
			"level":    row["level"],
			"sort":     row["sort"],
		}
	}

	// 构建树形结构
	return buildDeptTree(allDepts, 0), nil
}

// buildDeptTree 递归构建部门树
func buildDeptTree(depts []g.Map, parentId int64) []g.Map {
	var tree []g.Map
	for _, dept := range depts {
		pid := gconv.Int64(dept["parentId"])
		if pid == parentId {
			did := gconv.Int64(dept["id"])
			children := buildDeptTree(depts, did)
			node := g.Map{
				"id":       dept["id"],
				"name":     dept["name"],
				"parentId": dept["parentId"],
				"level":    dept["level"],
				"sort":     dept["sort"],
			}
			if len(children) > 0 {
				node["children"] = children
			}
			tree = append(tree, node)
		}
	}
	return tree
}

// List 查询部门列表
func (s *DeptService) List(ctx context.Context, page, pageSize int, keyword string, companyId int64) ([]g.Map, int64, error) {
	// 直接执行SQL查询，添加关联查询
	query := `
		SELECT 
			d.id, d.name, d.parent_id, d.level, d.status, d.sort, d.leader_id, d.created_at,
			c.name as company_name,
			p.name as parent_name,
			u.real_name as leader_name
		FROM sys_department d
		LEFT JOIN sys_company c ON d.company_id = c.id
		LEFT JOIN sys_department p ON d.parent_id = p.id
		LEFT JOIN sys_user u ON d.leader_id = u.id
		WHERE d.company_id = ?
	`
	if keyword != "" {
		query += " AND d.name LIKE ?"
	}
	query += " ORDER BY d.sort ASC LIMIT ? OFFSET ?"

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建参数
	params := []interface{}{companyId}
	if keyword != "" {
		params = append(params, "%"+keyword+"%")
	}
	params = append(params, pageSize, offset)

	// 执行查询
	result, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return nil, 0, err
	}

	// 获取总数
	totalQuery := "SELECT COUNT(*) as total FROM sys_department WHERE company_id = ?"
	if keyword != "" {
		totalQuery += " AND name LIKE ?"
	}
	totalParams := []interface{}{companyId}
	if keyword != "" {
		totalParams = append(totalParams, "%"+keyword+"%")
	}
	resultTotal, err := g.DB().GetOne(ctx, totalQuery, totalParams...)
	if err != nil {
		return nil, 0, err
	}
	total := resultTotal["total"].Int()

	// 转换为g.Map
	depts := make([]g.Map, len(result))
	for i, row := range result {
		depts[i] = g.Map{
			"id":          row["id"],
			"name":        row["name"],
			"parentId":    row["parent_id"],
			"parentName":  row["parent_name"],
			"level":       row["level"],
			"status":      row["status"],
			"sort":        row["sort"],
			"leaderId":    row["leader_id"],
			"leaderName":  row["leader_name"],
			"companyId":   companyId,
			"companyName": row["company_name"],
			"createdAt":   row["created_at"],
		}
	}

	return depts, int64(total), nil
}

// Save 保存部门
func (s *DeptService) Save(ctx context.Context, dept *model.SysDept, companyId int64) (int64, error) {
	data := g.Map{
		"name":       dept.Name,
		"parent_id":  dept.ParentId,
		"level":      dept.Level,
		"status":     dept.Status,
		"sort":       dept.Sort,
		"remark":     dept.Remark,
		"company_id": companyId,
	}

	if dept.Id > 0 {
		// 更新部门
		_, err := g.DB().Model("sys_department").Where("id", dept.Id).Data(data).Update()
		return dept.Id, err
	} else {
		// 新增部门
		id, err := g.DB().Model("sys_department").Data(data).InsertAndGetId()
		return id, err
	}
}

// Detail 获取部门详情
func (s *DeptService) Detail(ctx context.Context, deptId int64) (g.Map, error) {
	if deptId <= 0 {
		return nil, gerror.New("部门ID不能为空")
	}

	// 查询部门详情，包含关联信息
	query := `
		SELECT 
			d.id, d.name, d.parent_id, d.level, d.status, d.sort,
			d.company_id, d.created_at, d.updated_at,
			c.name as company_name,
			p.name as parent_name,
			u.real_name as leader_name
		FROM sys_department d
		LEFT JOIN sys_company c ON d.company_id = c.id
		LEFT JOIN sys_department p ON d.parent_id = p.id
		LEFT JOIN sys_user u ON d.leader_id = u.id
		WHERE d.id = ?
	`

	result, err := g.DB().GetOne(ctx, query, deptId)
	if err != nil {
		return nil, err
	}

	if result.IsEmpty() {
		return nil, nil
	}

	// 转换为前端需要的格式
	dept := g.Map{
		"id":          result["id"],
		"name":        result["name"],
		"parentId":    result["parent_id"],
		"parentName":  result["parent_name"],
		"level":       result["level"],
		"status":      result["status"],
		"sort":        result["sort"],
		"companyId":   result["company_id"],
		"companyName": result["company_name"],
		"leaderId":    result["leader_id"],
		"leaderName":  result["leader_name"],
		"createdAt":   result["created_at"],
		"updatedAt":   result["updated_at"],
	}

	return dept, nil
}

// Delete 删除部门
func (s *DeptService) Delete(ctx context.Context, deptId int64) error {
	// 检查是否有子部门
	count, err := g.DB().Model("sys_department").Where("parent_id", deptId).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return gerror.New("该部门下有子部门，无法删除")
	}

	// 检查是否有用户属于该部门
	count, err = g.DB().Model("sys_user").Where("dept_id", deptId).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return gerror.New("该部门下有用户，无法删除")
	}

	// 删除部门
	_, err = g.DB().Model("sys_department").Where("id", deptId).Delete()
	return err
}
