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
	apply := flag.Bool("apply", false, "写入官方店铺归属；默认仅预览")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	accessToken, err := oceanengine.GetLatestAccessToken(ctx)
	if err != nil {
		fail(err)
	}
	result, err := oceanengine.SyncQianchuanStoreMappings(ctx, accessToken, !*apply)
	if err != nil {
		fail(err)
	}
	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
