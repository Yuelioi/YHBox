package expr

import (
	"strings"
	"testing"
)

// TestMaxDepthRejected guards against stack-overflow on pathological Expr
// configs (E3 security cap). A user — accidentally or maliciously — could
// paste 1000+ nested parens; without the depth cap the recursive parser
// would blow the goroutine stack. With the cap, we get a clean error string.
func TestMaxDepthRejected(t *testing.T) {
	// 300 nested parens — well above MaxExprDepth (256).
	deep := strings.Repeat("(", 300) + "1" + strings.Repeat(")", 300)
	_, err := Parse(deep)
	if err == nil {
		t.Fatal("expected depth error on deeply nested expression")
	}
	if !strings.Contains(err.Error(), "max nesting depth") {
		t.Errorf("wrong error: %v", err)
	}
}

// TestModerateDepthAccepted ensures the cap is high enough for real expressions.
// 200 nested parens (under MaxExprDepth=256) must still parse.
func TestModerateDepthAccepted(t *testing.T) {
	// 100 nested parens — sane real-world Expr would never exceed this.
	moderate := strings.Repeat("(", 100) + "1" + strings.Repeat(")", 100)
	if _, err := Parse(moderate); err != nil {
		t.Errorf("100-deep parse should succeed, got: %v", err)
	}
}
