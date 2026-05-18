// internal/services/container/nodekind/spec.go
//
// Package nodekind owns the single source of truth for every node Kind in
// the v4 container graph. Both validator and runtime derive their per-kind
// tables from here. Adding a new Kind = adding one Register(&Spec{...}) call
// in specs/*.go — nothing else.
//
// Design notes:
//   - Spec does NOT carry executor function pointers. That would force the
//     nodekind pkg to import runtime, which imports container, which would
//     loop. Instead runtime keeps its execNode switch, and a CI test asserts
//     the switch's case set matches the registry's kind set (Phase D).
//   - ExecOutFn is the only callable in Spec — it's a pure function over
//     config, no runtime state required (used by Switch/Parallel/Race for
//     dynamic out pins).
package nodekind

// PinType is the v4 5-type data pin system. Mirrors frontend pinSpec.PinDataType.
type PinType string

const (
	PinNumber PinType = "number"
	PinBool   PinType = "bool"
	PinString PinType = "string"
	PinPoint  PinType = "point"
	PinAny    PinType = "any"
)

// Spec is the canonical description of a node Kind.
type Spec struct {
	// Kind is the unique identifier used in GraphNode.Kind and edge refs.
	Kind string

	// Group is the palette grouping key (e.g. "control", "data", "input").
	// Drives NodePalette.KINDS_BY_GROUP on the frontend (derived).
	Group string

	// ExecIn is the list of legal exec-in pin names. Empty = no exec in
	// (pure data node like GetVar, or entry node like Start / OnEvent).
	// Default for callers that omit: ["in"].
	ExecIn []string

	// ExecOut is the list of legal static exec-out pin names. Used when
	// ExecOutFn is nil. Empty = terminal node (Stop / Break / Continue / Throw).
	ExecOut []string

	// ExecOutFn, if set, computes exec-out pins from node config. Takes
	// precedence over ExecOut. Used by Switch (cases + default), Parallel /
	// Race (branchN + complete).
	ExecOutFn func(cfg map[string]any) []string

	// DataIn maps data-in pin name → type. Empty = no data inputs.
	// Dynamic data-in (Expr.inputs[], Subgraph call's inputParams) is not
	// represented here — callers ask the spec's DataInDynamicFn.
	DataIn map[string]PinType

	// DataInDynamicFn, if set, returns extra data-in pins computed from
	// config or external state (Subgraph inputParams). Returned map is
	// merged with DataIn at lookup time. Nil = static only.
	DataInDynamicFn func(cfg map[string]any) map[string]PinType

	// DataOut maps data-out pin name → type. Empty = no data outputs.
	DataOut map[string]PinType

	// Defaults is the initial node.Config when user creates a new node.
	// May contain "literal": {pinName: value, ...} for inline pin defaults.
	Defaults map[string]any

	// IsYield = true means a Loop body containing this kind is allowed to be
	// infinite (the kind yields CPU). Replaces validate.go yieldKinds map.
	IsYield bool

	// IsPureData = true means this kind exposes only data-out pins (no exec
	// pins) and is evaluated on-demand by data_pull.go's evalDataSource.
	// Used by validator to enforce "no exec edge from a pure-data node".
	IsPureData bool

	// IsVisualOnly = true means the kind is render-only (CommentBox).
	// Runtime no-op; validator skips pin/edge checks.
	IsVisualOnly bool
}

// ExecOutPins returns the legal exec-out pin set for the given config.
// Used by validator and frontend exec-out rendering.
func (s *Spec) ExecOutPins(cfg map[string]any) []string {
	if s.ExecOutFn != nil {
		return s.ExecOutFn(cfg)
	}
	return s.ExecOut
}

// DataInType returns the type of a data-in pin, considering static + dynamic.
// Empty string = pin does not exist on this kind.
func (s *Spec) DataInType(pinName string, cfg map[string]any) PinType {
	if t, ok := s.DataIn[pinName]; ok {
		return t
	}
	if s.DataInDynamicFn != nil {
		if t, ok := s.DataInDynamicFn(cfg)[pinName]; ok {
			return t
		}
	}
	return ""
}

// DataOutType returns the type of a data-out pin (static only — no kind
// currently has dynamic data-out except via Subgraph OutputPins, handled
// at call site, not in spec).
func (s *Spec) DataOutType(pinName string) PinType {
	return s.DataOut[pinName]
}
