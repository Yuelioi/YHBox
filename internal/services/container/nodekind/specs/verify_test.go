// internal/services/container/nodekind/specs/verify_test.go
package specs

import "testing"

func TestVerifyDetectExtractorsRegistered(t *testing.T) {
	if err := VerifyDetectExtractorsRegistered(); err != nil {
		t.Fatalf("detect group has missing DependencyExtractor: %v", err)
	}
}
