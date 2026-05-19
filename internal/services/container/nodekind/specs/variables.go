// internal/services/container/nodekind/specs/variables.go
package specs

import "yhbox/internal/services/container/nodekind"

func init() {
	nodekind.Register(&nodekind.Spec{
		Kind:     "SetVar",
		Group:    "variables",
		ExecIn:   []string{"in"},
		ExecOut:  []string{"out"},
		DataIn:   map[string]nodekind.PinType{"value": nodekind.PinAny},
		Defaults: map[string]any{"varName": "", "scope": "auto", "literal": map[string]any{"value": ""}},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:     "IncVar",
		Group:    "variables",
		ExecIn:   []string{"in"},
		ExecOut:  []string{"out"},
		DataIn:   map[string]nodekind.PinType{"delta": nodekind.PinNumber},
		Defaults: map[string]any{"varName": "", "scope": "auto", "literal": map[string]any{"delta": 1.0}},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:       "GetVar",
		Group:      "variables",
		ExecIn:     nil,
		ExecOut:    nil,
		DataOut:    map[string]nodekind.PinType{"value": nodekind.PinAny},
		Defaults:   map[string]any{"varName": "", "scope": "auto"},
		IsPureData: true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:       "GetSys",
		Group:      "variables",
		ExecIn:     nil,
		ExecOut:    nil,
		DataOut:    map[string]nodekind.PinType{"value": nodekind.PinAny},
		Defaults:   map[string]any{"path": ""},
		IsPureData: true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:       "GetParam",
		Group:      "variables",
		ExecIn:     nil,
		ExecOut:    nil,
		DataOut:    map[string]nodekind.PinType{"value": nodekind.PinAny},
		Defaults:   map[string]any{"paramName": ""},
		IsPureData: true,
	})
}
