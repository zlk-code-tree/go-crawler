package service

import (
	"context"
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

// DeptService 部门相关业务逻辑
type DeptService struct{}

func NewDeptService() *DeptService {
	return &DeptService{}
}

// Tree 查询部门树
func (s *DeptService) Tree(ctx context.Context) ([]g.Map, error) {
	// TODO: 实现真实的部门树查询
	data := []g.Map{{"id": 1, "name": "总部", "parentId": 0, "children": nil}}
	return data, nil
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
