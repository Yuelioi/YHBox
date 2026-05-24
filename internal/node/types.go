// internal/node/types.go
package node

// Point 域类型 — ratio 坐标 (0.0-1.0).
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Rect 域类型 — ratio 矩形.
type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// Color HSV.
type Color struct {
	H int `json:"h"`
	S int `json:"s"`
	V int `json:"v"`
}

var typeRegistry = map[string]TypeSpec{}

// RegisterType 注册自定义 pin 类型. 内置类型 init() 自动注册.
func RegisterType(ts TypeSpec) {
	if _, exists := typeRegistry[ts.Tag]; exists {
		panic("type tag " + ts.Tag + " already registered")
	}
	typeRegistry[ts.Tag] = ts
}

// AllTypes registry snapshot copy.
func AllTypes() []TypeSpec {
	out := make([]TypeSpec, 0, len(typeRegistry))
	for _, ts := range typeRegistry {
		out = append(out, ts)
	}
	return out
}

func init() {
	// 内置类型 — FE 启动时通过 GetAllTypes RPC 拉颜色/widget 映射.
	for _, ts := range []TypeSpec{
		{Tag: "String", GoType: "string", WidgetKind: "text", Color: "#4ade80"},
		{Tag: "Number", GoType: "float64", WidgetKind: "number", Color: "#3b82f6"},
		{Tag: "Integer", GoType: "int", WidgetKind: "number", Color: "#3b82f6"},
		{Tag: "Bool", GoType: "bool", WidgetKind: "checkbox", Color: "#facc15"},
		{Tag: "Point", GoType: "node.Point", WidgetKind: "point-editor", Color: "#a855f7"},
		{Tag: "Rect", GoType: "node.Rect", WidgetKind: "rect-editor", Color: "#a855f7"},
		{Tag: "Color", GoType: "node.Color", WidgetKind: "color-picker", Color: "#ef4444"},
		{Tag: "Image", GoType: "*image.RGBA", WidgetKind: "preview", Color: "#9ca3af"},
		{Tag: "Duration", GoType: "time.Duration", WidgetKind: "duration", Color: "#3b82f6"},
		{Tag: "JSON", GoType: "map[string]any", WidgetKind: "json", Color: "#9ca3af"},
		{Tag: "Exec", GoType: "(framework)", WidgetKind: "exec-pin", Color: "#ffffff"},
	} {
		typeRegistry[ts.Tag] = ts
	}
}
