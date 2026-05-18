// internal/services/container/nodekind/specs/purefunc.go
package specs

import "yhbox/internal/services/container/nodekind"

// pureFunc builds a v4 §6 pure-function node spec. All have:
//   - No exec pins (IsPureData=true)
//   - Static data-in (passed in)
//   - Single data-out named "result"
//   - Defaults.literal echoes data-in zero values
func pureFunc(kind string, dataIn map[string]nodekind.PinType, resultType nodekind.PinType, defaults map[string]any) *nodekind.Spec {
	return &nodekind.Spec{
		Kind:       kind,
		Group:      "purefunc",
		DataIn:     dataIn,
		DataOut:    map[string]nodekind.PinType{"result": resultType},
		Defaults:   map[string]any{"literal": defaults},
		IsPureData: true,
	}
}

func init() {
	num2 := map[string]nodekind.PinType{"a": nodekind.PinNumber, "b": nodekind.PinNumber}
	any2 := map[string]nodekind.PinType{"a": nodekind.PinAny, "b": nodekind.PinAny}
	str2 := map[string]nodekind.PinType{"a": nodekind.PinString, "b": nodekind.PinString}
	zNum := map[string]any{"a": 0.0, "b": 0.0}
	zAny := map[string]any{}
	zStr := map[string]any{"a": "", "b": ""}

	// 算术 (6)
	nodekind.Register(pureFunc("Add", num2, nodekind.PinNumber, zNum))
	nodekind.Register(pureFunc("Sub", num2, nodekind.PinNumber, zNum))
	nodekind.Register(pureFunc("Mul", num2, nodekind.PinNumber, zNum))
	nodekind.Register(pureFunc("Div", num2, nodekind.PinNumber, zNum))
	nodekind.Register(pureFunc("Mod", num2, nodekind.PinNumber, zNum))
	nodekind.Register(pureFunc("Neg",
		map[string]nodekind.PinType{"x": nodekind.PinNumber},
		nodekind.PinNumber,
		map[string]any{"x": 0.0}))

	// 比较 (6) — Lt/LtEq/Gt/GtEq 是 number→bool, Eq/NotEq 是 any→bool
	nodekind.Register(pureFunc("Lt", num2, nodekind.PinBool, zNum))
	nodekind.Register(pureFunc("LtEq", num2, nodekind.PinBool, zNum))
	nodekind.Register(pureFunc("Gt", num2, nodekind.PinBool, zNum))
	nodekind.Register(pureFunc("GtEq", num2, nodekind.PinBool, zNum))
	nodekind.Register(pureFunc("Eq", any2, nodekind.PinBool, zAny))
	nodekind.Register(pureFunc("NotEq", any2, nodekind.PinBool, zAny))

	// 逻辑 (3) — pin type 是 bool 但 v3 expr 历史是 any, 先按 spec §6.3 用 bool
	bool2 := map[string]nodekind.PinType{"a": nodekind.PinBool, "b": nodekind.PinBool}
	zBool := map[string]any{"a": false, "b": false}
	zBoolT := map[string]any{"a": true, "b": true}
	nodekind.Register(pureFunc("And", bool2, nodekind.PinBool, zBoolT))
	nodekind.Register(pureFunc("Or", bool2, nodekind.PinBool, zBool))
	nodekind.Register(pureFunc("Not",
		map[string]nodekind.PinType{"x": nodekind.PinBool},
		nodekind.PinBool,
		map[string]any{"x": false}))

	// 字符串 (3)
	nodekind.Register(pureFunc("Concat", str2, nodekind.PinString, zStr))
	nodekind.Register(pureFunc("Contains",
		map[string]nodekind.PinType{"haystack": nodekind.PinString, "needle": nodekind.PinString},
		nodekind.PinBool,
		map[string]any{"haystack": "", "needle": ""}))
	nodekind.Register(pureFunc("Length",
		map[string]nodekind.PinType{"s": nodekind.PinString},
		nodekind.PinNumber,
		map[string]any{"s": ""}))

	// 转换 (3)
	nodekind.Register(pureFunc("ToString",
		map[string]nodekind.PinType{"x": nodekind.PinAny},
		nodekind.PinString,
		map[string]any{"x": ""}))
	nodekind.Register(pureFunc("ToNumber",
		map[string]nodekind.PinType{"x": nodekind.PinAny},
		nodekind.PinNumber,
		map[string]any{"x": 0.0}))
	nodekind.Register(pureFunc("ToBool",
		map[string]nodekind.PinType{"x": nodekind.PinAny},
		nodekind.PinBool,
		map[string]any{"x": false}))

	// Select 三元 (1)
	nodekind.Register(pureFunc("Select",
		map[string]nodekind.PinType{"cond": nodekind.PinBool, "a": nodekind.PinAny, "b": nodekind.PinAny},
		nodekind.PinAny,
		map[string]any{"cond": true}))
}
