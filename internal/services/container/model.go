// Package container 持有蓝图节点图实体。本包不实现 graph 运行时（在 services/container/runtime/）——
// 这里只负责 JSON schema、持久化、基础 schema 校验。节点 Config 字段以 map[string]any 形式不透明存储。
package container

import "time"

// CurrentSchemaVersion 整个 Container 文件的 schema 版本号（与 Graph.Version 区分；前者控容器外壳布局，
// 后者控 graph 内部 schema）。
const CurrentSchemaVersion = 1

// GraphSchemaVersion 每个 Graph 内部 schema 的版本号。
// 所有 graph 共用同一个值；schema 演进必须递增；启动 load 时不匹配标 Status: Incompatible（不 panic）。
const GraphSchemaVersion = 1

const (
	StatusIncompatible = "incompatible"
)

// VarDecl Container 实例级变量声明。
//
// Type: "number" | "bool" | "string" | "point" | "list" | "any"
// Default: 跟 Type 对应；point 是 map{"x":float, "y":float}；list 是 []any (元素不分类型)
type VarDecl struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
}

// GraphNode 节点。Config 形态由 kind 决定，本包视作不透明。
// CreatedAt 在 v2 新增：debugger / analytics 用。
type GraphNode struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	Label     string         `json:"label,omitempty"`     // 用户可编辑显示名 (optional, 不影响逻辑)
	X         float32        `json:"x"`
	Y         float32        `json:"y"`
	Config    map[string]any `json:"config,omitempty"`
	Disabled    bool           `json:"disabled,omitempty"`    // runtime 跳过该节点 (kind-aware passthrough)
	LogEnabled  bool           `json:"logEnabled,omitempty"`  // 勾选 → 执行时吐通用 dump 日志
	CreatedAt   time.Time      `json:"createdAt"`
}

// GraphEdge 边。From/To 格式："<nodeId>.<pinName>"。
// 对 Subgraph 调用节点，pinName = SubgraphOutputDecl.ID.
//
// Kind 字段已删 — 边类型 (data / exec) 由 (from-node.kind, from-pin) 派生:
// fromPin 在 nodekind.Spec.DataOut 里 → data 边; 否则 exec 边 (含 Subgraph 动态 exec-out).
// 多余字段 (旧 "kind":"data") JSON 反序列化静默 drop.
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Graph 节点图。有 ID (UUID) + Version (= GraphSchemaVersion).
type Graph struct {
	ID      string      `json:"id"`      // UUID；graph 自己有 identity
	Version int         `json:"version"` // = GraphSchemaVersion 写入时
	Nodes   []GraphNode `json:"nodes"`
	Edges   []GraphEdge `json:"edges"`
}

// Container 蓝图编排实体。
// v2：删 Category 字段（破坏式）；保留 Tags（多 tag）。
type Container struct {
	SchemaVersion int      `json:"schemaVersion"`
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Hotkey        string   `json:"hotkey,omitempty"`
	// 容器级窗口后端配置 (原在 WindowTarget 节点, v2 挪容器级 — 整容器一套后端).
	// InputBackend "sendinput" 走 OS 全局注入 (需前台焦点 → WindowTarget 解析时自动拉前台);
	// "postmessage" (默认) 按 hwnd 直发, 后台不抢焦点. 激活与否由此字段决定 (原 RunMode 已并入).
	InputBackend   string  `json:"inputBackend,omitempty"`
	CaptureBackend string  `json:"captureBackend,omitempty"`
	ScaleTolerance float64 `json:"scaleTolerance,omitempty"`
	Vars  []VarDecl `json:"vars,omitempty"`
	Graph Graph     `json:"graph"`
	// 子图不再是容器字段 (2026-06-12 全局化): 全局池 data/subgraphs/, 容器图节点按
	// SubgraphID 引用; 校验/运行时按依赖闭包从池解析快照传入.
	// Status 运行时标记（不持久化）。值："" 正常 / "incompatible" graceful load fail
	Status             string    `json:"-"`
	IncompatibleReason string    `json:"-"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// SubgraphOutputDecl 一个命名出口的稳定标识。
// 父图 edge / runtime dispatch / validator 都用 ID 引用；UI 用 Name 显示。
// rename 只改 Name 不动 ID，父图无感。
//
// NodeID/X/Y 是子图内 virtual "出口节点" 的 metadata —
// edges 仍按 `<NodeID>.<pin>` 引用, validator/dispatch 白名单识别. 不再存进 Graph.Nodes.
type SubgraphOutputDecl struct {
	ID     string  `json:"id"`               // UUID，稳定不变
	Name   string  `json:"name"`             // 用户可见显示名（"done"/"found"/"timeout"...）
	NodeID string  `json:"nodeID,omitempty"` // virtual marker ID, edges 引用
	X      float32 `json:"x,omitempty"`      // editor canvas position
	Y      float32 `json:"y,omitempty"`      // editor canvas position
}

// SubgraphMarker Subgraph entry 的 virtual 节点位置 + ID. Edges 引用 NodeID,
// validator/dispatch 白名单识别. 不在 Graph.Nodes 里 — user 无法误删/误改 kind.
type SubgraphMarker struct {
	NodeID string  `json:"nodeID"`
	X      float32 `json:"x,omitempty"`
	Y      float32 `json:"y,omitempty"`
}

// SubgraphInputParam (v4) 子图入参声明.
// Subgraph 调用节点为每个 InputParam 自动加一个 data-in pin (name + type),
// 子图内部用 GetParam 节点 (config.paramName = Name) 读取入参值.
// 入参在子图调用入栈时从父图 pull → 写入 frame.LocalParams.
type SubgraphInputParam struct {
	Name    string `json:"name"`              // identifier (e.g. "hp", "targetTemplate")
	Type    string `json:"type"`              // PinType: "number"/"bool"/"string"/"point"/"any"
	Default any    `json:"default,omitempty"` // 父图 pull 拿到 nil 时用 (literal-style fallback)
}

// RecordingContext 录制自动折叠产生的 subgraph 的元数据。
// 手动组装的 subgraph 该字段为 nil（runtime 不缩放 raw dx/dy，原值回放）。
type RecordingContext struct {
	MouseCounts360 int    `json:"mouseCounts360"` // 录制时源机器 360° HID counts；runtime scale 用作分子
	Resolution     [2]int `json:"resolution"`     // 源机器分辨率（W, H）
	RecordedAt     string `json:"recordedAt"`     // RFC3339 时间戳
}

// SubgraphSchemaVersion 子图文件格式版本。读取契约: 磁盘上版本 > 此值 → 拒载报错;
// 写入时统一盖当前值。
const SubgraphSchemaVersion = 1

// Subgraph 全局子图池里的可执行函数 (2026-06-12 全局化: 容器只引用不拥有)。
// 持久化路径：data/subgraphs/<id>.json
type Subgraph struct {
	SchemaVersion    int                  `json:"schemaVersion"`              // 写盘统一盖 SubgraphSchemaVersion
	ID               string               `json:"id"`                         // sg-<完整uuid>, 铸出后终身不变
	Rev              int64                `json:"rev"`                        // 单调版本号: 每次保存 +1; 乐观锁比对用 (仅单实例并发控制, 不参与跨机真相判定)
	Label            string               `json:"label"`
	Description      string               `json:"description,omitempty"`
	Graph            Graph                `json:"graph"`                      // 内部节点 + 边 (不含 SubgraphInput/Output)
	Entry            SubgraphMarker       `json:"entry"`                      // 子图入口 virtual marker
	OutputPins       []SubgraphOutputDecl `json:"outputPins"`                 // 出口声明 + virtual marker
	InputParams      []SubgraphInputParam `json:"inputParams,omitempty"`      // 子图入参声明 (data-in pin schema on call sites)
	Tags             []string             `json:"tags,omitempty"`
	Category         string               `json:"category,omitempty"`         // 库分组键 (空 = 未分类)
	RequiredGlobals  []string             `json:"requiredGlobals,omitempty"`  // 引用的容器级 var 名字 (保存时派生; 只存名字 — Type/Default 由消费方按目标容器即时现算)
	RecordingContext *RecordingContext    `json:"recordingContext,omitempty"` // 录制自动折叠时写入；手动 nil
	IsAnonymous      bool                 `json:"isAnonymous,omitempty"`      // CollapsedNode 后备子图, 不入库浏览/候选下拉 (实现细节, 非用户资产)
	CreatedAt        time.Time            `json:"createdAt"`
}
