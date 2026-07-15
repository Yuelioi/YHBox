package compiler

import "testing"

func TestBuildDigestIsStableExecutableIdentity(t *testing.T) {
	first, err := BuildDigest()
	if err != nil || !first.Valid() {
		t.Fatalf("BuildDigest = %s, %v", first, err)
	}
	second, err := BuildDigest()
	if err != nil || second != first {
		t.Fatalf("second BuildDigest = %s, %v; want %s", second, err, first)
	}
}
