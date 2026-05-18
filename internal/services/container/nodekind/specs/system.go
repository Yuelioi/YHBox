// internal/services/container/nodekind/specs/system.go
package specs

import "yhbox/internal/services/container/nodekind"

func init() {
	nodekind.Register(&nodekind.Spec{
		Kind:   "BringGameForeground", Group: "system",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:     "MouseCalibration",
		Group:    "system",
		ExecIn:   nil,
		ExecOut:  nil, // 声明式 — runtime 直通
		Defaults: map[string]any{"counts360": 0.0},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "WindowTarget", Group: "system",
		ExecIn: nil, ExecOut: nil,
		Defaults: map[string]any{
			"match":   map[string]any{"title": "", "class": "", "processName": "", "titleMatch": "exact"},
			"runtime": map[string]any{"inputBackend": "postmessage", "captureBackend": "auto"},
		},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "CommentBox", Group: "system",
		ExecIn: nil, ExecOut: nil,
		Defaults:     map[string]any{"label": "注释", "color": "#fbbf24", "width": 200.0, "height": 150.0},
		IsVisualOnly: true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "Try", Group: "system",
		ExecIn: []string{"in"}, ExecOut: []string{"done", "timeout", "error"},
		DataIn:   map[string]nodekind.PinType{"timeoutMs": nodekind.PinNumber},
		DataOut:  map[string]nodekind.PinType{"errorMsg": nodekind.PinString},
		Defaults: map[string]any{"subgraphId": "", "literal": map[string]any{"timeoutMs": 0.0}},
		IsYield:  true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "Throw", Group: "system",
		ExecIn: []string{"in"}, ExecOut: nil, // terminal
		DataIn:   map[string]nodekind.PinType{"message": nodekind.PinString},
		Defaults: map[string]any{"literal": map[string]any{"message": ""}},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "StopwatchStart", Group: "system",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		Defaults: map[string]any{"key": "default"},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "StopwatchStop", Group: "system",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		Defaults: map[string]any{"key": "default"},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "StopwatchRead", Group: "system",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		DataOut:  map[string]nodekind.PinType{"elapsedMs": nodekind.PinNumber},
		Defaults: map[string]any{"key": "default"},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "Log", Group: "system",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		DataIn:   map[string]nodekind.PinType{"message": nodekind.PinAny},
		Defaults: map[string]any{"level": "info", "literal": map[string]any{"message": "hello"}},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "Toast", Group: "system",
		ExecIn: []string{"in"}, ExecOut: []string{"out"},
		DataIn:   map[string]nodekind.PinType{"title": nodekind.PinAny, "message": nodekind.PinAny},
		Defaults: map[string]any{"color": "primary", "literal": map[string]any{"title": "提示", "message": ""}},
	})
	// Subgraph 调用节点 — exec-out 来自被调子图的 OutputPins, 不在 Spec 里表达
	// (validator + 前端各自查 subgraph 数据). DataIn 通过 DataInDynamicFn 由 inputParams 派生
	// (Phase B inputParams 实装后 DataInDynamicFn 填入真实查询).
	nodekind.Register(&nodekind.Spec{
		Kind:   "Subgraph", Group: "system",
		ExecIn: []string{"in"},
		// exec-out 跟 data-in 都是动态的, validator/frontend 查 subgraphId → Subgraph.OutputPins/InputParams.
		// 在 Spec 里留空; ExecOutPins(cfg) 永远返 [] 是有意的 — caller 必须特判 kind=="Subgraph".
		Defaults: map[string]any{"subgraphId": ""},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:    "SubgraphInput", Group: "system",
		ExecIn:  nil,
		ExecOut: []string{"out"},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:     "SubgraphOutput", Group: "system",
		ExecIn:   []string{"in"},
		ExecOut:  nil,
		Defaults: map[string]any{"declID": ""},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "CollapsedNode", Group: "system",
		ExecIn: []string{"in"},
		// Same as Subgraph — dynamic from backing isAnonymous Subgraph.
		Defaults: map[string]any{"subgraphId": "", "label": "Collapsed"},
	})
}
