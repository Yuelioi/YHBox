// Package container 持有蓝图节点图实体。本包不实现 graph 运行时（在 services/container/runtime/）——
// 这里只负责 JSON schema、持久化、基础 schema 校验。节点 Config 字段以 map[string]any 形式不透明存储。
package container

import "time"

// CurrentSchemaVersion 整个 Container 文件的 schema 版本号（与 Graph.Version 区分；前者控容器外壳布局，
// 后者控 graph 内部 schema）。
const CurrentSchemaVersion = 1

// GraphSchemaVersion 每个 Graph 内部 schema 的版本号。v2 spec §1.2：
// 所有 graph 共用同一个值；schema 演进必须递增；启动 load 时不匹配标 Status: Incompatible（不 panic）。
const GraphSchemaVersion = 1

const (
	StatusIncompatible = "incompatible"
)

// VarDecl Container 实例级变量声明。
//
// Type: "number" | "bool" | "string" | "point"
// Default: 跟 Type 对应；point 是 map{"x":float, "y":float}
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
	X         float32        `json:"x"`
	Y         float32        `json:"y"`
	Config    map[string]any `json:"config,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

// GraphEdge 边。From/To 格式："<nodeId>.<pinName>"。
// 对 Subgraph 调用节点，pinName = SubgraphOutputDecl.ID（v2 spec §1.4）。
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Graph 节点图。v2 在 v1 基础上加 ID（UUID）+ Version。
//
// FUTURE-WORK (spec §11): 缺 Revision 字段. 协作/撤回/历史回溯/HMR 都需要单调递增的
// 修订号. 当前 onSave 整个 graph 全量写盘, 没法做 "撤回到上 3 步" 这种交互. 加 Revision
// (uint64, save 时 +1) + 保留最近 N 个 snapshot 到 containers/<id>/history/, 就能解锁.
// 也是分布式协作 (CRDT / OT) 的前置. 加的时机: 用户开始抱怨 "改坏了想撤回" 或者真做协作.
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
	// RunMode "foreground" 启动前激活游戏窗口；"background" 默认 PostMessage 不抢焦点。
	RunMode   string     `json:"runMode,omitempty"`
	Vars      []VarDecl  `json:"vars,omitempty"`
	Graph     Graph      `json:"graph"`
	Subgraphs []Subgraph `json:"-"` // 运行时从 subgraphs/*.json 读入；不存进 container.json
	// Status 运行时标记（不持久化）。值："" 正常 / "incompatible" graceful load fail
	Status             string    `json:"-"`
	IncompatibleReason string    `json:"-"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// SubgraphOutputDecl 一个命名出口的稳定标识（v2 spec §1.4，GPT 第三轮关键加固）。
// 父图 edge / runtime dispatch / validator 都用 ID 引用；UI 用 Name 显示。
// rename 只改 Name 不动 ID，父图无感。
type SubgraphOutputDecl struct {
	ID   string `json:"id"`   // UUID，稳定不变
	Name string `json:"name"` // 用户可见显示名（"done"/"found"/"timeout"...）
}

// RecordingContext 录制自动折叠产生的 subgraph 的元数据。
// 手动组装的 subgraph 该字段为 nil（runtime 不缩放 raw dx/dy，原值回放）。
type RecordingContext struct {
	MouseCounts360 int    `json:"mouseCounts360"` // 录制时源机器 360° HID counts；runtime scale 用作分子
	Resolution     [2]int `json:"resolution"`     // 源机器分辨率（W, H）
	RecordedAt     string `json:"recordedAt"`     // RFC3339 时间戳
}

// Subgraph 容器内的可执行函数（v2 spec §1.4）。
// 持久化路径：bin/data/containers/<container-id>/subgraphs/<id>.json
// 库 subgraph 路径：bin/data/library/subgraphs/<id>.json（多带 requiredTemplates 信息，见 library 包）
type Subgraph struct {
	ID               string               `json:"id"`
	Label            string               `json:"label"`
	Description      string               `json:"description,omitempty"`
	Graph            Graph                `json:"graph"`                      // 内部节点 + 边
	OutputPins       []SubgraphOutputDecl `json:"outputPins"`                 // 由内部 SubgraphOutput 节点派生缓存
	Tags             []string             `json:"tags,omitempty"`             // 容器内 + 库 都用
	RecordingContext *RecordingContext    `json:"recordingContext,omitempty"` // 录制自动折叠时写入；手动 nil
	CreatedAt        time.Time            `json:"createdAt"`
}
