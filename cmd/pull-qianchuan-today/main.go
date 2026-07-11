package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	_ "go-crawler/internal/packed"
	"go-crawler/internal/service/platform/oceanengine"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
)

func main() {
	today := time.Now().Format("2006-01-02")
	startDate := flag.String("start", today, "开始日期，格式 2006-01-02")
	endDate := flag.String("end", today, "结束日期，格式 2006-01-02")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	result, err := oceanengine.GetPoller().ManualPullByDate(ctx, *startDate, *endDate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}
