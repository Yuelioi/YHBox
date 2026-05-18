// internal/services/container/nodekind/specs/control.go
package specs

import (
	"strconv"

	"yhbox/internal/services/container/nodekind"
)

// Parallel / Race dynamic branch count is clamped to [minDynBranches, maxDynBranches].
// Raising maxDynBranches above 99 requires no code change (strconv.Itoa handles any int).
const (
	minDynBranches = 2
	maxDynBranches = 8
)

func init() {
	nodekind.Register(&nodekind.Spec{
		Kind:    "Start",
		Group:   "control",
		ExecIn:  nil, // 入口节点没 exec-in
		ExecOut: []string{"out"},
	})

	nodekind.Register(&nodekind.Spec{
		Kind:    "Stop",
		Group:   "control",
		ExecIn:  []string{"in"},
		ExecOut: nil, // terminal
	})

	nodekind.Register(&nodekind.Spec{
		Kind:     "Sleep",
		Group:    "control",
		ExecIn:   []string{"in"},
		ExecOut:  []string{"out"},
		DataIn:   map[string]nodekind.PinType{"durationMs": nodekind.PinNumber},
		Defaults: map[string]any{"literal": map[string]any{"durationMs": 1000.0}},
		IsYield:  true,
	})

	nodekind.Register(&nodekind.Spec{
		Kind:    "Loop",
		Group:   "control",
		ExecIn:  []string{"in", "loopback"},
		ExecOut: []string{"body", "complete"},
		DataIn: map[string]nodekind.PinType{
			"count":     nodekind.PinNumber,
			"condition": nodekind.PinBool,
		},
		DataOut:  map[string]nodekind.PinType{"iter": nodekind.PinNumber},
		Defaults: map[string]any{"mode": "count", "literal": map[string]any{"count": 10.0, "condition": true}},
	})

	nodekind.Register(&nodekind.Spec{
		Kind:    "If",
		Group:   "control",
		ExecIn:  []string{"in"},
		ExecOut: []string{"then", "else"},
		DataIn: map[string]nodekind.PinType{
			"condition": nodekind.PinBool,
		},
		Defaults: map[string]any{"literal": map[string]any{"condition": true}},
	})

	nodekind.Register(&nodekind.Spec{
		Kind:    "Switch",
		Group:   "control",
		ExecIn:  []string{"in"},
		ExecOut: []string{"default"}, // fallback if ExecOutFn 没匹配
		ExecOutFn: func(cfg map[string]any) []string {
			cases, _ := cfg["cases"].([]any)
			seen := map[string]bool{}
			out := make([]string, 0, len(cases)+1)
			for _, c := range cases {
				s, ok := c.(string)
				if !ok || s == "" || seen[s] {
					continue
				}
				seen[s] = true
				out = append(out, s)
			}
			out = append(out, "default")
			return out
		},
		DataIn:   map[string]nodekind.PinType{"value": nodekind.PinString},
		Defaults: map[string]any{"cases": []any{"a", "b"}, "literal": map[string]any{"value": ""}},
	})

	nodekind.Register(&nodekind.Spec{
		Kind:    "Parallel",
		Group:   "control",
		ExecIn:  []string{"in"},
		ExecOut: []string{"complete"},
		ExecOutFn: func(cfg map[string]any) []string {
			n := dynBranchCount(cfg)
			out := make([]string, 0, n+1)
			for i := range n {
				out = append(out, branchName(i))
			}
			out = append(out, "complete")
			return out
		},
		DataIn:   map[string]nodekind.PinType{"n": nodekind.PinNumber},
		Defaults: map[string]any{"literal": map[string]any{"n": 2.0}},
	})

	nodekind.Register(&nodekind.Spec{
		Kind:    "Race",
		Group:   "control",
		ExecIn:  []string{"in"},
		ExecOut: []string{"complete"},
		ExecOutFn: func(cfg map[string]any) []string {
			n := dynBranchCount(cfg)
			out := make([]string, 0, n+1)
			for i := range n {
				out = append(out, branchName(i))
			}
			out = append(out, "complete")
			return out
		},
		DataIn:   map[string]nodekind.PinType{"n": nodekind.PinNumber},
		DataOut:  map[string]nodekind.PinType{"winnerIdx": nodekind.PinNumber},
		Defaults: map[string]any{"literal": map[string]any{"n": 2.0}},
	})

	nodekind.Register(&nodekind.Spec{
		Kind:    "Break",
		Group:   "control",
		ExecIn:  []string{"in"},
		ExecOut: nil,
	})

	nodekind.Register(&nodekind.Spec{
		Kind:    "Continue",
		Group:   "control",
		ExecIn:  []string{"in"},
		ExecOut: nil,
	})
}

// dynBranchCount reads the branch count from cfg.literal.n (v4 inline literal) with
// fallback to top-level cfg.n (v3 legacy / unit-test shape). Clamps to
// [minDynBranches, maxDynBranches].
func dynBranchCount(cfg map[string]any) int {
	var raw float64
	if v, ok := cfg["literal"].(map[string]any); ok {
		if f, ok := v["n"].(float64); ok {
			raw = f
		}
	}
	if raw == 0 {
		if f, ok := cfg["n"].(float64); ok {
			raw = f
		}
	}
	return max(minDynBranches, min(maxDynBranches, int(raw)))
}

// branchName returns "branch{i}" for Parallel/Race exec-out pins. Safe for any i ≥ 0;
// callers are expected to feed values from dynBranchCount so i ∈ [0, maxDynBranches).
func branchName(i int) string { return "branch" + strconv.Itoa(i) }
