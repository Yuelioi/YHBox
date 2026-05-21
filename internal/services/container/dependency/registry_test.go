package dependency

import "testing"

func TestRegisterExtractor_Basic(t *testing.T) {
	const k = "_test_register_kind"
	RegisterExtractor(k, fakeExtractor{})
	defer delete(extractorByKind, k)
	if got := GetExtractor(k); got == nil {
		t.Errorf("GetExtractor(%q) = nil", k)
	}
	// 第二次注册同 kind → panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on duplicate")
		}
	}()
	RegisterExtractor(k, fakeExtractor{})
}

func TestRegisterExtractor_EmptyKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on empty kind")
		}
	}()
	RegisterExtractor("", fakeExtractor{})
}

func TestRegisterExtractor_NilExtractorPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on nil extractor")
		}
	}()
	RegisterExtractor("_test_nil_ext_kind", nil)
}

func TestVerifyDetectGroupHasExtractors(t *testing.T) {
	const knownK = "_test_known"
	RegisterExtractor(knownK, fakeExtractor{})
	defer delete(extractorByKind, knownK)

	if err := VerifyDetectGroupHasExtractors([]string{knownK}); err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	err := VerifyDetectGroupHasExtractors([]string{knownK, "_test_missing"})
	if err == nil {
		t.Fatal("expected error for missing kind")
	}
	if !containsStr(err.Error(), "_test_missing") {
		t.Errorf("error doesn't mention missing kind: %v", err)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
