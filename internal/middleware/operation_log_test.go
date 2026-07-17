package middleware

import (
	"strings"
	"testing"
)

func TestSanitizeOperationParams(t *testing.T) {
	got := sanitizeOperationParams([]byte(`{"username":"tester","password":"secret","nested":{"access_token":"token-value"}}`))
	if strings.Contains(got, "secret") || strings.Contains(got, "token-value") {
		t.Fatalf("敏感字段未脱敏: %s", got)
	}
	if !strings.Contains(got, "tester") || !strings.Contains(got, `"password":"***"`) {
		t.Fatalf("脱敏结果不正确: %s", got)
	}
}

func TestSanitizeOperationFormParams(t *testing.T) {
	got := sanitizeOperationParams([]byte(`username=tester&password=secret&access_token=token-value`))
	if strings.Contains(got, "secret") || strings.Contains(got, "token-value") {
		t.Fatalf("表单敏感字段未脱敏: %s", got)
	}
}
