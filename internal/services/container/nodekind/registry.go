// internal/services/container/nodekind/registry.go
package nodekind

import "fmt"

// byKind is the package-level registry. Populated by specs/*.go init() calls.
// Once registered a Spec is immutable from the outside (we hand back the
// pointer but no setter — treat as read-only).
var byKind = map[string]*Spec{}

// Register adds a Spec to the registry. Panics on duplicate Kind — that's
// a programmer error, not a runtime condition. Must be called from init()
// only; doing it later is racy.
func Register(s *Spec) {
	if s == nil {
		panic("nodekind.Register: nil spec")
	}
	if s.Kind == "" {
		panic("nodekind.Register: empty Kind")
	}
	if _, dup := byKind[s.Kind]; dup {
		panic(fmt.Sprintf("nodekind.Register: duplicate Kind %q", s.Kind))
	}
	byKind[s.Kind] = s
}

// Get returns the Spec for the given kind. Second return is false for
// unknown kinds. Callers MUST check ok before using the spec.
func Get(kind string) (*Spec, bool) {
	s, ok := byKind[kind]
	return s, ok
}

// MustGet panics if the kind is unknown. Use in places where prior validation
// already confirmed the kind is known (e.g. runtime after ValidateContainer).
func MustGet(kind string) *Spec {
	s, ok := byKind[kind]
	if !ok {
		panic(fmt.Sprintf("nodekind.MustGet: unknown Kind %q", kind))
	}
	return s
}

// Kinds returns all registered kind names in unspecified order. Used by
// validator (KnownNodeKinds check) and cross-check tests.
func Kinds() []string {
	out := make([]string, 0, len(byKind))
	for k := range byKind {
		out = append(out, k)
	}
	return out
}

// IsKnown is the v4 replacement for validate.KnownNodeKinds[kind].
func IsKnown(kind string) bool {
	_, ok := byKind[kind]
	return ok
}
