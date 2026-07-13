package artifact

import (
	"bytes"
	"testing"
)

func TestCanonicalizeUsesRFC8785ValueSemantics(t *testing.T) {
	left, err := Canonicalize([]byte(` { "b": 1.0, "a": "x" } `))
	if err != nil {
		t.Fatal(err)
	}
	right, err := Canonicalize([]byte(`{"a":"x","b":1e0}`))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) || string(left) != `{"a":"x","b":1}` {
		t.Fatalf("left=%s right=%s", left, right)
	}
}

func TestDigestIsDomainSeparatedAndStrict(t *testing.T) {
	a, err := Sum("yotta/test/a/v1", []byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Sum("yotta/test/b/v1", []byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	if a == b || !a.Valid() || !b.Valid() {
		t.Fatalf("a=%q b=%q", a, b)
	}
	for _, invalid := range []string{"", string(a)[7:], "SHA256:" + string(a)[7:]} {
		if _, err := ParseDigest(invalid); err == nil {
			t.Fatalf("accepted invalid digest %q", invalid)
		}
	}
	for _, domain := range []string{"", "unversioned", "yotta/test/v0", "yotta/test\x00/v1"} {
		if _, err := Sum(domain, nil); err == nil {
			t.Fatalf("accepted invalid domain %q", domain)
		}
	}
}

func TestCanonicalizeRejectsNumbersThatWouldLoseIdentity(t *testing.T) {
	for _, raw := range []string{`9007199254740992`, `-9007199254740992`, `1e400`, `1e-400`} {
		if _, err := Canonicalize([]byte(`{"n":` + raw + `}`)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	for _, raw := range []string{`9007199254740991`, `-9007199254740991`, `0.1`} {
		if _, err := Canonicalize([]byte(`{"n":` + raw + `}`)); err != nil {
			t.Fatalf("rejected %s: %v", raw, err)
		}
	}
}

func TestCanonicalizeRejectsInvalidUTF8(t *testing.T) {
	if _, err := Canonicalize([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}); err == nil {
		t.Fatal("accepted invalid UTF-8")
	}
}
