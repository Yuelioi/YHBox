// internal/services/container/nodekind/specs/input.go
package specs

import "yhbox/internal/services/container/nodekind"

func init() {
	nodekind.Register(&nodekind.Spec{
		Kind:   "ClickAt", Group: "input",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		DataIn: map[string]nodekind.PinType{
			"xRatio": nodekind.PinNumber, "yRatio": nodekind.PinNumber, "durationMs": nodekind.PinNumber,
		},
		Defaults: map[string]any{"button": "left", "literal": map[string]any{"xRatio": 0.5, "yRatio": 0.5, "durationMs": 50.0}},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "KeyPress", Group: "input",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		DataIn:   map[string]nodekind.PinType{"durationMs": nodekind.PinNumber},
		Defaults: map[string]any{"vk": "W", "literal": map[string]any{"durationMs": 50.0}},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "MouseMoveRel", Group: "input",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		DataIn: map[string]nodekind.PinType{
			"dx": nodekind.PinNumber, "dy": nodekind.PinNumber, "durationMs": nodekind.PinNumber,
		},
		Defaults: map[string]any{"literal": map[string]any{"dx": 0.0, "dy": 0.0, "durationMs": 200.0}},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "Scroll", Group: "input",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		DataIn: map[string]nodekind.PinType{
			"xRatio": nodekind.PinNumber, "yRatio": nodekind.PinNumber, "delta": nodekind.PinNumber,
		},
		Defaults: map[string]any{"literal": map[string]any{"xRatio": 0.5, "yRatio": 0.5, "delta": 3.0}},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "KeyHoldStart", Group: "input",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		Defaults: map[string]any{"vk": "A"},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "KeyHoldStop", Group: "input",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		Defaults: map[string]any{"vk": "A"},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "MouseHoldStart", Group: "input",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		DataIn:   map[string]nodekind.PinType{"xRatio": nodekind.PinNumber, "yRatio": nodekind.PinNumber},
		Defaults: map[string]any{"button": "left", "literal": map[string]any{"xRatio": 0.5, "yRatio": 0.5}},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "MouseHoldStop", Group: "input",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		Defaults: map[string]any{"button": "left"},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:    "OnEvent",
		Group:   "input",
		ExecIn:  nil, // listener 节点, 没 exec-in
		ExecOut: []string{"out"},
		DataIn: map[string]nodekind.PinType{
			"threshold": nodekind.PinNumber, "pollIntervalMs": nodekind.PinNumber,
			"maxConcurrent": nodekind.PinNumber, "cooldownMs": nodekind.PinNumber,
		},
		Defaults: map[string]any{
			"kind": "template_appeared", "template": "", "retriggerPolicy": "drop",
			"literal": map[string]any{"pollIntervalMs": 100.0, "maxConcurrent": 1.0, "cooldownMs": 0.0},
		},
		IsYield: true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "PlayClip", Group: "input",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		Defaults: map[string]any{"clipID": "", "keepRanges": []any{}},
	})
}
