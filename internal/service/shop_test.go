package service

import "testing"

func TestNullableAccountUID(t *testing.T) {
	if got := nullableAccountUID("   "); got != nil {
		t.Fatalf("blank uid should map to nil, got %#v", got)
	}

	got := nullableAccountUID("  123456  ")
	uid, ok := got.(string)
	if !ok {
		t.Fatalf("non-empty uid should remain string, got %#v", got)
	}
	if uid != "123456" {
		t.Fatalf("uid = %q, want %q", uid, "123456")
	}
}
