package main

import (
	_ "go-crawler/internal/packed"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"

	"github.com/gogf/gf/v2/os/gctx"

	"go-crawler/internal/cmd"

	_ "go-crawler/internal/logic/crawler"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
