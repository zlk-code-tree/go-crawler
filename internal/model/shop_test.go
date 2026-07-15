package model

import (
	"context"
	"testing"

	"github.com/gogf/gf/v2/frame/g"
)

func TestShopSaveReqAllowsEmptyUID(t *testing.T) {
	req := ShopSaveReq{
		Platform:    "douyin",
		AccountName: "测试账号",
		Operator:    "测试运营者",
	}

	if err := g.Validator().Data(req).Run(context.Background()); err != nil {
		t.Fatalf("empty UID should be accepted: %v", err)
	}
}
