// internal/node/types.go
package node

import "sync"

// PointUnit — Point 坐标单位. "" = 比例(0-1), "px" = 客户区像素.
type PointUnit string

const (
	UnitRatio PointUnit = ""   // 百分比/比例 (X/Y 0-1) — 默认
	UnitPx    PointUnit = "px" // 客户区像素 (X/Y 像素值)
)

// Point 域类型 — 坐标 + 单位.
type Point struct {
	X    float64   `json:"x"`
	Y    float64   `json:"y"`
	Unit PointUnit `json:"unit,omitempty"`
}

// Rect 域类型 — ratio 矩形.
type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// PixelRect 像素整数矩形 (几何 override 用; 跟 Rect 的 ratio float 区分).
type PixelRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// Resolution 分辨率键 (具名 + 小写 tag, 不用匿名 struct 防大写 JSON).
type Resolution struct {
	W int `json:"w"`
	H int `json:"h"`
}

// GeoOverride 某分辨率的像素 ROI 覆盖.
type GeoOverride struct {
	Resolution Resolution `json:"resolution"`
	Px         PixelRect  `json:"px"`
}

// Geometry 几何输入值: pct 自适应默认 (ratio 0-1) + 可选每分辨率像素覆盖.
// runtime 解析: override 匹配当前帧分辨率优先, 否则 pct×帧尺寸; pct.W==0||pct.H==0 且无匹配 = 全帧.
type Geometry struct {
	Pct       Rect          `json:"pct"`
	Overrides []GeoOverride `json:"overrides,omitempty"`
}

// Color HSV.
type Color struct {
	H int `json:"h"`
	S int `json:"s"`
	V int `json:"v"`
}

// Image 图像 pin 值 — 编码字节 + 格式。Capture 产出、SaveImage 写盘、LoadImage 读盘、
// AI 节点 vision 多模态消费;在图里作为一等流动值(经 exec-data 直连或捕获到变量)。
//
// Data is immutable; consumers MUST NOT modify the underlying byte slice —
// 值在进程内按引用流动、不深拷, 改了会跨节点串数据。
type Image struct {
	Format string `json:"format"` // "png" | "jpeg"
	Data   []byte `json:"data"`
}

// typeRegistry 加 mutex (跟 globalRegistry 对称). init() 之外的 RegisterType
// 罕见, 但 AllTypes RPC handler 并发读取需安全.
var (
	typeRegistryMu sync.RWMutex
	typeRegistry   = map[string]TypeSpec{}
)

// RegisterType 注册自定义 pin 类型. 内置类型 init() 自动注册.
func RegisterType(ts TypeSpec) {
	typeRegistryMu.Lock()
	defer typeRegistryMu.Unlock()
	if _, exists := typeRegistry[ts.Tag]; exists {
		panic("type tag " + ts.Tag + " already registered")
	}
	typeRegistry[ts.Tag] = ts
}

// AllTypes registry snapshot copy.
func AllTypes() []TypeSpec {
	typeRegistryMu.RLock()
	defer typeRegistryMu.RUnlock()
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
		{Tag: "Geometry", GoType: "node.Geometry", WidgetKind: "geometry", Color: "#a855f7"},
		{Tag: "Color", GoType: "node.Color", WidgetKind: "color-picker", Color: "#ef4444"},
		{Tag: "Image", GoType: "node.Image", WidgetKind: "preview", Color: "#9ca3af"},
		{Tag: "Duration", GoType: "time.Duration", WidgetKind: "duration", Color: "#3b82f6"},
		{Tag: "JSON", GoType: "map[string]any", WidgetKind: "json", Color: "#9ca3af"},
		{Tag: "List", GoType: "[]any", WidgetKind: "list-preview", Color: "#818cf8"},
		{Tag: "Exec", GoType: "(framework)", WidgetKind: "exec-pin", Color: "#ffffff"},
	} {
		typeRegistry[ts.Tag] = ts
	}
}
