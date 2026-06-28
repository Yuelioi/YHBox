package mcpserver

import "encoding/json"

const graphSchemaVersion = 1

func schemaText() string {
	return `Yotta 容器图 schema:
- 顶层: {schemaVersion:1, name, runMode:"background", graph:{version:1, nodes:[], edges:[]}}
- 节点: {id, kind, x, y, config:{literal:{<pinName>:<value>}}}
- pin 值唯一合法写法 = config.literal[<pinName>]。
- edge: {from:"<nodeId>.<PinName>", to:"<nodeId>.<PinName>"}
- 用了 needsWindow 节点 (如 KeyPress/ClickAt) 的图, 必须含一个 Win32WindowTarget 节点 (不连线, config.literal.Title 填窗口标题)。
- Duration 类型 = 毫秒数字 (500 = 500ms)。
- id 由服务端生成, 入参里的 id 会被忽略。
- 节点种类与每个 pin 详见 list_nodes。`
}

// schemaExamples returns ≥2 validated example container JSONs.
// Example 0: minimal Start→Stop (no window nodes).
// Example 1: needsWindow (KeyPress) + Win32WindowTarget + Loop + Sleep with required Duration.
// Both MUST pass ValidateContainer — guarded by TestSchemaExamples_AllValid.
func schemaExamples() [][]byte {
	minimal := []byte(`{
  "schemaVersion": 1,
  "name": "最简",
  "runMode": "background",
  "graph": {
    "version": 1,
    "nodes": [
      { "id": "s", "kind": "Start", "x": 100, "y": 100 },
      { "id": "t", "kind": "Stop",  "x": 400, "y": 100 }
    ],
    "edges": [
      { "from": "s.Done", "to": "t.In" }
    ]
  }
}`)

	rich := []byte(`{
  "schemaVersion": 1,
  "name": "循环按F",
  "runMode": "background",
  "graph": {
    "version": 1,
    "nodes": [
      { "id": "s",  "kind": "Start",        "x": 100, "y": 100 },
      { "id": "w",  "kind": "Win32WindowTarget",  "x": 100, "y": 260,
        "config": { "literal": { "Title": "记事本" } } },
      { "id": "l",  "kind": "Loop",          "x": 320, "y": 100,
        "config": { "literal": { "Mode": "count", "Count": 10 } } },
      { "id": "k",  "kind": "KeyPress",      "x": 540, "y":  60,
        "config": { "literal": { "VK": "F", "DurationMs": 30 } } },
      { "id": "sl", "kind": "Sleep",         "x": 760, "y":  60,
        "config": { "literal": { "Duration": 500 } } },
      { "id": "t",  "kind": "Stop",          "x": 540, "y": 220 }
    ],
    "edges": [
      { "from": "s.Done",  "to": "l.In"   },
      { "from": "l.Body",  "to": "k.In"   },
      { "from": "k.Done",  "to": "sl.In"  },
      { "from": "sl.Done", "to": "l.In"   },
      { "from": "l.Done",  "to": "t.In"   }
    ]
  }
}`)

	return [][]byte{minimal, rich}
}

// graphSchemaJSON builds the get_graph_schema tool response payload.
// Examples are embedded as JSON objects (not strings) via json.RawMessage.
func graphSchemaJSON() []byte {
	exs := schemaExamples()
	raws := make([]json.RawMessage, len(exs))
	for i, e := range exs {
		raws[i] = json.RawMessage(e)
	}
	payload := map[string]any{
		"schemaVersion": graphSchemaVersion,
		"schema":        schemaText(),
		"examples":      raws,
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return b
}
