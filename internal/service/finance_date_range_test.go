package service

import "testing"

func TestNormalizeFinanceDateRange(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		endDate   string
		wantStart string
		wantEnd   string
	}{
		{
			name:      "next day midnight is exclusive",
			startDate: "2026-07-11 00:00:00",
			endDate:   "2026-07-12 00:00:00",
			wantStart: "2026-07-11",
			wantEnd:   "2026-07-11",
		},
		{
			name:      "date only remains inclusive",
			startDate: "2026-07-11",
			endDate:   "2026-07-12",
			wantStart: "2026-07-11",
			wantEnd:   "2026-07-12",
		},
		{
			name:      "end of day maps to same day",
			startDate: "2026-07-11 00:00:00",
			endDate:   "2026-07-11 23:59:59",
			wantStart: "2026-07-11",
			wantEnd:   "2026-07-11",
		},
		{
			name:      "missing start uses end",
			startDate: "",
			endDate:   "2026-07-12 00:00:00",
			wantStart: "2026-07-12",
			wantEnd:   "2026-07-12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := NormalizeFinanceDateRange(tt.startDate, tt.endDate)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Fatalf("NormalizeFinanceDateRange() = (%q, %q), want (%q, %q)", gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestParseTimeRangeNormalizesExclusiveMidnightEnd(t *testing.T) {
	service := NewFinanceCommissionService()
	gotStart, gotEnd := service.ParseTimeRange("custom", "2026-07-11 00:00:00", "2026-07-12 00:00:00")
	if gotStart != "2026-07-11" || gotEnd != "2026-07-11" {
		t.Fatalf("ParseTimeRange() = (%q, %q), want (%q, %q)", gotStart, gotEnd, "2026-07-11", "2026-07-11")
	}
}
