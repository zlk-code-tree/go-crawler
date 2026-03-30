package main

import (
	_ "go-crawler/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"go-crawler/internal/cmd"

	_ "go-crawler/internal/logic/crawler"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
