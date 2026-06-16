package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
)

func main() {
	ctx := gctx.New()

	fmt.Println("开始分批执行回填操作...")
	start := time.Now()

	batchSize := 500
	totalAffected := int64(0)
	batchNum := 0

	for {
		batchNum++

		// MySQL 多表 UPDATE 不支持 LIMIT，改用子查询先获取要更新的 ID
		// Step 1: 获取本批次需要更新的订单 ID
		idSQL := `
			SELECT o.id
			FROM order_info o
			INNER JOIN alliance_orders ao ON o.sub_order_no COLLATE utf8mb4_general_ci = ao.order_id COLLATE utf8mb4_general_ci AND ao.platform = 'douyin'
			INNER JOIN account a ON ao.author_account COLLATE utf8mb4_unicode_ci = a.account_name COLLATE utf8mb4_unicode_ci
			WHERE (o.talent_id IS NULL OR o.talent_id = '' OR o.talent_id = '0')
			  AND ao.author_account IS NOT NULL AND ao.author_account != ''
			  AND a.uid IS NOT NULL AND a.uid != ''
			LIMIT ?
		`
		rows, err := g.DB().GetAll(ctx, idSQL, batchSize)
		if err != nil {
			log.Printf("批次 %d 查询ID失败: %v", batchNum, err)
			time.Sleep(2 * time.Second)
			continue
		}

		if len(rows) == 0 {
			fmt.Println("没有更多需要回填的数据")
			break
		}

		// 收集 ID 列表
		var ids []interface{}
		for _, row := range rows {
			ids = append(ids, row["id"].Int64())
		}

		// Step 2: 使用 ID 列表进行更新
		updateSQL := `
			UPDATE order_info o
			INNER JOIN alliance_orders ao ON o.sub_order_no COLLATE utf8mb4_general_ci = ao.order_id COLLATE utf8mb4_general_ci AND ao.platform = 'douyin'
			INNER JOIN account a ON ao.author_account COLLATE utf8mb4_unicode_ci = a.account_name COLLATE utf8mb4_unicode_ci
			SET o.talent_id = a.uid,
			    o.talent_name = a.account_name
			WHERE o.id IN(?)
			  AND ao.author_account IS NOT NULL AND ao.author_account != ''
			  AND a.uid IS NOT NULL AND a.uid != ''
		`
		result, err := g.DB().Exec(ctx, updateSQL, ids)
		if err != nil {
			log.Printf("批次 %d 更新失败: %v", batchNum, err)
			time.Sleep(2 * time.Second)
			continue
		}

		affected, _ := result.RowsAffected()
		totalAffected += affected

		fmt.Printf("批次 %d: 回填 %d 条, 累计 %d 条\n", batchNum, affected, totalAffected)

		// 短暂休息，避免锁冲突
		time.Sleep(200 * time.Millisecond)
	}

	duration := time.Since(start)
	fmt.Printf("\n✅ 回填完成！\n")
	fmt.Printf("  总回填记录数: %d\n", totalAffected)
	fmt.Printf("  总耗时: %v\n", duration)

	// 验证订单
	fmt.Println("\n验证订单 6953666518178797210:")
	sql2 := `SELECT main_order_no, talent_id, talent_name FROM order_info WHERE main_order_no = '6953666518178797210'`
	result2, _ := g.DB().GetOne(ctx, sql2)
	fmt.Printf("  talent_id: %s\n", result2["talent_id"].String())
	fmt.Printf("  talent_name: %s\n", result2["talent_name"].String())

	if result2["talent_id"].String() != "0" && result2["talent_id"].String() != "" {
		fmt.Println("  ✅ 回填成功！")
	} else {
		fmt.Println("  ❌ 仍未回填")
	}
}
