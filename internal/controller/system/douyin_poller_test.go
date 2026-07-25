package system

import "testing"

func TestValidateOrderAllianceRangeMonthly(t *testing.T) {
	windowCount, err := validateOrderAllianceRange("2026-07-01 00:00:00", "2026-07-31 23:59:59")
	if err != nil {
		t.Fatalf("validateOrderAllianceRange() error = %v", err)
	}
	if windowCount != 31 {
		t.Fatalf("window count = %d, want 31", windowCount)
	}
}

func TestValidateOrderAllianceRangeRejectsOver31Days(t *testing.T) {
	_, err := validateOrderAllianceRange("2026-06-01 00:00:00", "2026-07-31 23:59:59")
	if err == nil {
		t.Fatal("validateOrderAllianceRange() error = nil, want range limit error")
	}
}
