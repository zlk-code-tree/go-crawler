package oceanengine

import (
	"testing"
	"time"
)

func TestParseQianchuanTokenTimeUsesChinaTimezone(t *testing.T) {
	parsed, err := parseQianchuanTokenTime("2026-07-10 14:56:14")
	if err != nil {
		t.Fatalf("parseQianchuanTokenTime() error = %v", err)
	}

	want := time.Date(2026, 7, 10, 14, 56, 14, 0, time.FixedZone("CST", 8*60*60))
	if !parsed.Equal(want) {
		t.Fatalf("parseQianchuanTokenTime() = %s, want %s", parsed, want)
	}
}
