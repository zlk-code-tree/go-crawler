package main

import (
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	ctx := gctx.New()
	
	// 查询所有外链菜单
	result, err := g.DB().GetAll(ctx, 
		`SELECT id, name, code, type, parent_id, path, component, ext_url, hide, status, sort 
		 FROM sys_permission 
		 WHERE ext_url != '' AND ext_url IS NOT NULL
		 ORDER BY sort ASC`)
	
	if err != nil {
		fmt.Println("查询失败:", err)
		return
	}
	
	fmt.Println("外链菜单列表：")
	fmt.Println("ID\t名称\t\tCode\t\t\tPath\t\t\tExtUrl\t\t\tHide\tStatus")
	fmt.Println("------------------------------------------------------------------------------------")
	for _, row := range result {
		fmt.Printf("%d\t%s\t\t%s\t\t%s\t\t%s\t\t%d\t%d\n",
			row["id"].Int(),
			row["name"].String(),
			row["code"].String(),
			row["path"].String(),
			row["ext_url"].String(),
			row["hide"].Int(),
			row["status"].Int(),
		)
	}
}
