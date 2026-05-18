// internal/services/container/nodekind/specs/control.go
package specs

import (
	"yhbox/internal/services/container/nodekind"
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
			for i := 0; i < n; i++ {
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
			for i := 0; i < n; i++ {
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

// dynBranchCount clamps Parallel/Race's n config to [2, 8].
func dynBranchCount(cfg map[string]any) int {
	v, _ := cfg["literal"].(map[string]any)
	raw, _ := v["n"].(float64)
	n := int(raw)
	if n < 2 {
		n = 2
	}
	if n > 8 {
		n = 8
	}
	return n
}

func branchName(i int) string { return "branch" + itoa(i) }

func itoa(i int) string {
	if i < 10 {
		return string('0' + byte(i))
	}
	// only used for 0..7, never >= 10 by clamp above; fallback for safety
	return ""
}
