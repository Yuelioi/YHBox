package nodekind

import (
	"encoding/json"
	"sort"
)

// ExportedSpec is the JSON-friendly view of Spec — strips function pointers
// (ExecOutFn, DataInDynamicFn) since they can't be compared across languages,
// keeps only static metadata. Both backend and frontend dump this shape to
// JSON; the D2 parity test diffs the two outputs to enforce no drift.
//
// Excludes IsYield (validator-only on backend, no frontend equivalent) — D2
// parity test skips this field.
//
// Includes HasExecOutFn / HasDataInDynamicFn booleans so parity can assert
// "both sides agree this kind has a dynamic-pin function" without comparing
// the function bodies (which would require cross-language semantic analysis).
type ExportedSpec struct {
	Kind               string             `json:"kind"`
	Group              string             `json:"group"`
	ExecIn             []string           `json:"execIn"`
	ExecOut            []string           `json:"execOut"` // static only; dynamic via execOutFn
	HasExecOutFn       bool               `json:"hasExecOutFn"`
	DataIn             map[string]PinType `json:"dataIn"`
	HasDataInDynamicFn bool               `json:"hasDataInDynamicFn"`
	DataOut            map[string]PinType `json:"dataOut"`
	IsPureData         bool               `json:"isPureData"`
	IsVisualOnly       bool               `json:"isVisualOnly"`
}

// ExportJSON returns all registered specs as a sorted JSON array (stable
// output for diffing with frontend export). The slice is sorted by Kind
// ascending so two dumps from semantically-equal registries produce
// byte-identical output.
func ExportJSON() ([]byte, error) {
	specs := make([]ExportedSpec, 0, len(byKind))
	for _, s := range byKind {
		specs = append(specs, ExportedSpec{
			Kind:               s.Kind,
			Group:              s.Group,
			ExecIn:             normalizeStringSlice(s.ExecIn),
			ExecOut:            normalizeStringSlice(s.ExecOut),
			HasExecOutFn:       s.ExecOutFn != nil,
			DataIn:             normalizePinMap(s.DataIn),
			HasDataInDynamicFn: s.DataInDynamicFn != nil,
			DataOut:            normalizePinMap(s.DataOut),
			IsPureData:         s.IsPureData,
			IsVisualOnly:       s.IsVisualOnly,
		})
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Kind < specs[j].Kind })
	return json.MarshalIndent(specs, "", "  ")
}

// normalizeStringSlice returns nil for empty/nil slices so JSON output is
// consistently `null` (not `[]`) — matches frontend's choice of `null` for
// empty arrays, ensuring byte-identical diffs.
func normalizeStringSlice(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

// normalizePinMap returns nil for empty/nil maps for the same reason as
// normalizeStringSlice: byte-identical JSON output for empty maps.
func normalizePinMap(m map[string]PinType) map[string]PinType {
	if len(m) == 0 {
		return nil
	}
	return m
}
