package oceanengine

import (
	"context"
	"errors"
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

func TestAuthorRateLimitBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: -1, want: 2 * time.Second},
		{attempt: 0, want: 2 * time.Second},
		{attempt: 1, want: 4 * time.Second},
		{attempt: 2, want: 8 * time.Second},
		{attempt: 9, want: 8 * time.Second},
	}
	for _, tt := range tests {
		if got := authorRateLimitBackoff(tt.attempt); got != tt.want {
			t.Fatalf("authorRateLimitBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
	if !isQianchuanRateLimitCode(40100) || isQianchuanRateLimitCode(0) {
		t.Fatal("isQianchuanRateLimitCode() did not recognize only code 40100")
	}
}

func TestRequestIntervalLimiterHonorsContext(t *testing.T) {
	limiter := newRequestIntervalLimiter(time.Hour)
	if err := limiter.Wait(context.Background()); err != nil {
		t.Fatalf("first Wait() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := limiter.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("second Wait() error = %v, want context.Canceled", err)
	}
}

func TestAuthorMatchIndexMatchPlanReport(t *testing.T) {
	uidAccount := authorOrgInfo{AccountID: 1, AwemeID: 4308320006710169, AccountName: "麓酒讲堂", PlatformNickname: "75346375612"}
	showIDAccount := authorOrgInfo{AccountID: 2, AwemeID: 2673613480925396, AccountName: "南山思维", PlatformNickname: "Banxia1122"}
	nameAccount := authorOrgInfo{AccountID: 3, AwemeID: 2049916467414779, AccountName: "Rika黄老师科学育儿", PlatformNickname: "73917566259"}
	index := &authorMatchIndex{
		uidMap: map[int64][]authorOrgInfo{
			uidAccount.AwemeID: {uidAccount},
		},
		showIDMap: map[string][]authorOrgInfo{
			normalizeAuthorMatchValue(showIDAccount.PlatformNickname): {showIDAccount},
			normalizeAuthorMatchValue(nameAccount.PlatformNickname):   {nameAccount},
		},
		accountNameMap: map[string][]authorOrgInfo{
			normalizeAuthorMatchValue(nameAccount.AccountName): {nameAccount},
		},
	}

	tests := []struct {
		name       string
		report     AdPlanDailyReport
		wantID     int
		wantSource string
	}{
		{name: "uid", report: AdPlanDailyReport{AwemeID: "4308320006710169"}, wantID: 1, wantSource: "uid"},
		{name: "alphanumeric show id", report: AdPlanDailyReport{AwemeShowID: " banxia1122 "}, wantID: 2, wantSource: "show_id"},
		{name: "numeric show id stored in aweme id", report: AdPlanDailyReport{AwemeID: "73917566259"}, wantID: 3, wantSource: "show_id"},
		{name: "exact account name", report: AdPlanDailyReport{AwemeName: " Rika黄老师科学育儿 "}, wantID: 3, wantSource: "account_name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, source, ok := index.matchPlanReport(tt.report)
			if !ok || got.AccountID != tt.wantID || source != tt.wantSource {
				t.Fatalf("matchPlanReport() = (accountID=%d, source=%q, ok=%v), want (%d, %q, true)", got.AccountID, source, ok, tt.wantID, tt.wantSource)
			}
		})
	}
}

func TestAuthorMatchIndexMatchPlanReportRejectsAmbiguousValue(t *testing.T) {
	index := &authorMatchIndex{
		uidMap: map[int64][]authorOrgInfo{},
		showIDMap: map[string][]authorOrgInfo{
			"duplicate": {{AccountID: 1}, {AccountID: 2}},
		},
		accountNameMap: map[string][]authorOrgInfo{},
	}
	if got, source, ok := index.matchPlanReport(AdPlanDailyReport{AwemeShowID: "duplicate"}); ok {
		t.Fatalf("matchPlanReport() = (accountID=%d, source=%q, ok=true), want ambiguous match rejected", got.AccountID, source)
	}
}

func TestOverallProductAwemeAuthorFallbackUsesPureAdCost(t *testing.T) {
	org := authorOrgInfo{AwemeID: 4308320006710169, AccountName: "麓酒讲堂"}
	report, ok := overallProductAwemeAuthorFallback(1741941292252167, "2026-07-14 00:00:00", OverallProductAwemeReportItem{
		AnchorName:                      "麓酒讲堂",
		StatCostForOverallRoi2:          3093.41,
		StatCostForRoi2:                 2985.05,
		TotalOrderSettleAmountForRoi21h: 4047.70,
		TotalOrderSettleCountForRoi21h:  206,
	}, org)
	if !ok {
		t.Fatal("overallProductAwemeAuthorFallback() rejected positive ad cost")
	}
	if report.StatCost != 2985.05 {
		t.Fatalf("StatCost = %.2f, want pure ad cost 2985.05", report.StatCost)
	}
	if report.AwemeID != org.AwemeID || report.StatDatetime != "2026-07-14" {
		t.Fatalf("fallback identity/date = (%d, %q), want (%d, %q)", report.AwemeID, report.StatDatetime, org.AwemeID, "2026-07-14")
	}
}

func TestOverallProductAwemeAuthorFallbackRejectsZeroAdCost(t *testing.T) {
	_, ok := overallProductAwemeAuthorFallback(1, "2026-07-14", OverallProductAwemeReportItem{
		StatCostForOverallRoi2: 100,
		StatCostForRoi2:        0,
	}, authorOrgInfo{AwemeID: 123})
	if ok {
		t.Fatal("overallProductAwemeAuthorFallback() accepted zero pure ad cost")
	}
}
