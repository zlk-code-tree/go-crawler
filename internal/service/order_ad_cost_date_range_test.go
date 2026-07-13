package service

import "testing"

func TestOrderAdCostDateRange(t *testing.T) {
	tests := []struct {
		name      string
		startTime string
		endTime   string
		wantStart string
		wantEnd   string
	}{
		{
			name:      "next day midnight is exclusive",
			startTime: "2026-07-11 00:00:00",
			endTime:   "2026-07-12 00:00:00",
			wantStart: "2026-07-11",
			wantEnd:   "2026-07-11",
		},
		{
			name:      "end of day remains inclusive",
			startTime: "2026-07-11 00:00:00",
			endTime:   "2026-07-11 23:59:59",
			wantStart: "2026-07-11",
			wantEnd:   "2026-07-11",
		},
		{
			name:      "same midnight remains a single day",
			startTime: "2026-07-11 00:00:00",
			endTime:   "2026-07-11 00:00:00",
			wantStart: "2026-07-11",
			wantEnd:   "2026-07-11",
		},
		{
			name:      "date-only range remains inclusive",
			startTime: "2026-07-11",
			endTime:   "2026-07-12",
			wantStart: "2026-07-11",
			wantEnd:   "2026-07-12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := orderAdCostDateRange(tt.startTime, tt.endTime)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Fatalf("orderAdCostDateRange() = (%q, %q), want (%q, %q)", gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
