package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	_ "go-crawler/internal/packed"
	"go-crawler/internal/service"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
)

func main() {
	filePath := flag.String("file", "", "千川账户Excel路径")
	apply := flag.Bool("apply", false, "确认写入店铺归属；默认只预览")
	flag.Parse()
	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "必须通过 -file 指定千川账户Excel")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result, err := service.NewQianchuanStoreMappingService().ImportExcel(ctx, *filePath, !*apply)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}
