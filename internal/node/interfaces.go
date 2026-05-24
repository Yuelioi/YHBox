// internal/node/interfaces.go
// Package node interfaces — Node + extension interfaces + Inputs/Outputs/Ctx contracts.
package node

import (
	"context"
	"time"
)

// Node minimal interface — Spec + Run.
type Node interface {
	Spec() Spec
	Run(ctx Ctx, in Inputs) (Outputs, error)
}

// 以下是扩展接口, 节点选择性实现. Framework type assertion 探测.

// Displayer — 节点想自定义日志面板显示就实现. 不实现 → 节点不出现在 node-log.
type Displayer interface {
	Display(in Inputs, exitName string, out OutputData) string
}

// Validator — 节点自身静态校验.
type Validator interface {
	Validate(in Inputs) []ValidationError
}

// Dependencer — 子图分享 / library import 时 BFS 抽外部资产引用.
type Dependencer interface {
	Dependencies(in Inputs) []Dependency
}

// RegionRunner — 控制流节点 (Loop / Parallel / Try) 实现. 第一版不做.
type RegionRunner interface {
	Node
	RunRegion(ctx Ctx, in Inputs, body func(Ctx) error) (Outputs, error)
}

// Inputs — 节点 Run 收的输入. 取值优先级:
//   1. data-wire (纯数据 pin 上游) [最高]
//   2. Inspector config
//   3. exec-data wire (上游 exec 出口 Data 字段同名注入)
//   4. InputSpec.Default
//   5. Required + 缺 → framework 返 ValidationError (NOT panic, GPT r4 #8)
//   6. Optional + 缺 → 零值, Has() = false
type Inputs interface {
	String(name string) string
	Float64(name string) float64
	Int(name string) int
	Bool(name string) bool
	Point(name string) Point
	Rect(name string) Rect
	Color(name string) Color
	Duration(name string) time.Duration
	JSON(name string) map[string]any
	Raw(name string) any
	Has(name string) bool
}

// Outputs — Run 返回值. 节点不该直接构造, 通过 ctx.Out(...).Fire() 拿.
type Outputs interface {
	exit() (name string, data map[string]any)
}

// OutputData — Display 接收的出口数据 (immutable snapshot).
type OutputData interface {
	String(field string) string
	Float64(field string) float64
	Int(field string) int
	Bool(field string) bool
	Point(field string) Point
	Raw(field string) any
	Has(field string) bool
}

// Ctx — Run 期间框架注入. 最小集合.
type Ctx interface {
	Context() context.Context
	Vision() VisionService
	Log() LogService
	Now() time.Time
	Out(exitName string) OutBuilder
}

// OutBuilder — fluent builder.
//   - exitName 不在 Spec.Outputs → Fire panic
//   - Run 内调多次 Fire → 第 2 次 panic
type OutBuilder interface {
	Set(field string, value any) OutBuilder
	Fire() Outputs
}

// VisionService — Phase 1 stub; Phase 4 真接 wire_container.go::templateMatcherAdapter.
type VisionService interface {
	Match(key string, threshold float64) (pt *Point, conf float64, err error)
}

// LogService — 节点日志. Phase 1 stub stdout, Phase 4 接 zerolog.
type LogService interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
}

type ValidationError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Field   string         `json:"field,omitempty"`
	Params  map[string]any `json:"params,omitempty"`
}

type Dependency struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}
