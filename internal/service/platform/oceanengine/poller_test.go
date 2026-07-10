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

func TestShouldRefreshQianchuanToken(t *testing.T) {
	now := time.Date(2026, 7, 10, 10, 0, 0, 0, qianchuanTZ)
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{name: "outside refresh window", expiresAt: now.Add(9 * time.Hour), want: false},
		{name: "at refresh boundary", expiresAt: now.Add(8 * time.Hour), want: true},
		{name: "inside refresh window", expiresAt: now.Add(2 * time.Hour), want: true},
		{name: "already expired", expiresAt: now.Add(-time.Minute), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRefreshQianchuanToken(now, tt.expiresAt); got != tt.want {
				t.Fatalf("shouldRefreshQianchuanToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
