// Package container 持有蓝图节点图实体。本包不实现 graph 运行时（在 services/execution
// 与 services/actions/runtime/ 后续 plan 添加）—— 这里只负责 JSON schema、持久化、
// 基础 schema 校验。节点 Config 字段以 map[string]any 形式不透明存储，节点 kind 为字符串。
package container

import "time"

const CurrentSchemaVersion = 1

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
type GraphNode struct {
	ID     string         `json:"id"`
	Kind   string         `json:"kind"`
	X      float32        `json:"x"`
	Y      float32        `json:"y"`
	Config map[string]any `json:"config,omitempty"`
}

// GraphEdge 边。From/To 格式："<nodeId>.<pinName>"。
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// Container 蓝图编排实体。
type Container struct {
	SchemaVersion int       `json:"schemaVersion"`
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	Category      string    `json:"category,omitempty"`
	Tags          []string  `json:"tags,omitempty"`
	Hotkey        string    `json:"hotkey,omitempty"`
	// RunMode "foreground" 启动前激活游戏窗口（用 SendInput 类必须前台的脚本）；
	// "background" 默认，PostMessage 直发 hwnd 不抢焦点。
	RunMode   string    `json:"runMode,omitempty"`
	Vars      []VarDecl `json:"vars,omitempty"`
	Graph     Graph     `json:"graph"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
