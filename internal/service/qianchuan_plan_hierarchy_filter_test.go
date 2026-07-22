package service

import (
	"strings"
	"testing"
)

func TestQianchuanAwemeKeywordMatchesIDAndNameOnly(t *testing.T) {
	clause, args := qianchuanAwemeKeyword("daily", "万众读书")
	if clause == "" {
		t.Fatal("expected aweme keyword clause")
	}
	for _, unexpected := range []string{"ad_name", "product_name", "advertiser_name"} {
		if strings.Contains(clause, unexpected) {
			t.Fatalf("aweme keyword clause should not match %s: %s", unexpected, clause)
		}
	}
	for _, expected := range []string{"daily.aweme_show_id = ?", "CAST(daily.aweme_id AS CHAR) = ?", "daily.aweme_name LIKE ?"} {
		if !strings.Contains(clause, expected) {
			t.Fatalf("expected clause to contain %q: %s", expected, clause)
		}
	}
	if len(args) != 3 || args[0] != "万众读书" || args[1] != "万众读书" || args[2] != "%万众读书%" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestQianchuanAwemeKeywordIsFilteredRequest(t *testing.T) {
	if isUnfilteredQianchuanProfitRequest(QianchuanPlanHierarchyRequest{AwemeKeyword: "万众读书"}) {
		t.Fatal("aweme keyword requests must not use the unfiltered douyin profit snapshot")
	}
}
