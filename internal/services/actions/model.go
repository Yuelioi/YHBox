// Package actions 持有"动作"实体：录制产物的线性 step 序列。
//
// Container/Action 双层架构 中 Action 是 leaf：
//   - 不可独立运行（不绑热键、无 cron、无 tag）
//   - 只能被 Container 通过 InvokeAction 节点调用
//   - 字段仅 id + name + recordingContext + params + steps
//
// 元数据（hotkey / category / tags / description / loop 等）全部在 Container 层。
package actions

import "time"

const CurrentSchemaVersion = 2

type StepKind string

const (
	StepClickLeft    StepKind = "click_left"
	StepClickMiddle  StepKind = "click_middle"
	StepClickRight   StepKind = "click_right"
	StepKey          StepKind = "key"
	StepKeyDown      StepKind = "key_down"
	StepKeyUp        StepKind = "key_up"
	StepSleep        StepKind = "sleep"
	StepMouseMove    StepKind = "mouse_move"
	StepMouseDrag    StepKind = "mouse_drag"
	StepScroll       StepKind = "scroll"
	StepMouseMoveRel StepKind = "mouse_move_rel"
)

// Step 单条录制指令。XRatio/YRatio 等字段当前是数值类型，runtime 直接读。
type Step struct {
	ID          string   `json:"id"`
	Kind        StepKind `json:"kind"`
	XRatio      float32  `json:"xRatio,omitempty"`
	YRatio      float32  `json:"yRatio,omitempty"`
	DurationMs  int      `json:"durationMs,omitempty"`
	Vk          string   `json:"vk,omitempty"`
	X2Ratio     float32  `json:"x2Ratio,omitempty"`
	Y2Ratio     float32  `json:"y2Ratio,omitempty"`
	MouseButton string   `json:"mouseButton,omitempty"`
	ScrollDelta int      `json:"scrollDelta,omitempty"`
	Dx          int      `json:"dx,omitempty"`
	Dy          int      `json:"dy,omitempty"`
}

// RecordingContext 录制环境快照。供 UI 提示分辨率 + 跨设备回放缩放。
type RecordingContext struct {
	Resolution [2]int  `json:"resolution,omitempty"`
	DPIScale   float32 `json:"dpiScale,omitempty"`
	// MouseCounts360 录制时本机鼠标转 360° 累积 HID counts。
	// 跨设备回放时 dx *= localCounts / sourceCounts（双方均 >0 才缩放）。
	MouseCounts360 int `json:"mouseCounts360,omitempty"`
}

// ParamDecl Action 参数声明。Container 的 InvokeAction 节点根据这个列表渲染
// data input pin / 表单。当前未在 runtime 真正消化（容器变量优先于 action 参数）。
type ParamDecl struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "number" | "bool" | "string" | "point"
	Default any    `json:"default,omitempty"`
}

// Action 录制片段实体。
type Action struct {
	SchemaVersion    int              `json:"schemaVersion"`
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Description      string           `json:"description,omitempty"`
	RecordingContext RecordingContext `json:"recordingContext,omitempty"`
	Params           []ParamDecl      `json:"params,omitempty"`
	Steps            []Step           `json:"steps"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

// IsClickStep 给录制管线判 RequiresForeground 用（保留兼容）。
func IsClickStep(k StepKind) bool {
	return k == StepClickLeft || k == StepClickMiddle || k == StepClickRight
}
