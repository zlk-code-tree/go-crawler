package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "go-crawler/internal/packed"
	"go-crawler/internal/service/platform/oceanengine"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	today := time.Now().Format("2006-01-02")
	result, err := oceanengine.GetPoller().ManualPullByDate(ctx, today, today)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}
