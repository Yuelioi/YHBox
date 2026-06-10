// internal/node/spec.go
// Package node 节点系统核心 — declarative spec + runtime registry + inspector-first.
package node

// TypeExec exec pin 类型 tag. 抽 const 避免 "Exec" magic string 散落.
// 跟 InputSpec.Type / OutputSpec.Type 字段值对齐.
const TypeExec = "Exec"

// Spec 节点 metadata. 节点作者实现 Spec() 方法返这个.
// 展示文本 (节点名 / 描述 / pin label / hint / enum option label) 全由 FE i18n 单源持有
// (frontend/src/i18n/zh.ts node.<kind>.*), backend 只出结构 (kind / pin name / type / widget / enum value).
type Spec struct {
	Kind     string `json:"kind"`
	Category string `json:"category"` // FE palette 分组

	Inputs  []InputSpec  `json:"inputs"`
	Outputs []OutputSpec `json:"outputs"`

	IsPureData bool `json:"isPureData,omitempty"`
	// IsNonDeterministic — 节点 Evaluate 非确定 (如随机). 框架在 per-dispatch eval cache
	// 里记忆化其成功结果, 让同一 dispatch 内多路径引用拿同值 (守住 Determinism contract).
	// 仅在 IsPureData=true 节点上有意义 — 记忆化发生在 pure-data 拉取路径; exec 节点不读此字段.
	IsNonDeterministic bool `json:"isNonDeterministic,omitempty"`
	IsVisualOnly       bool `json:"isVisualOnly,omitempty"`
	// NeedsWindow — 节点 Run 依赖目标窗口 hwnd (调 ctx.Input/Capture/Vision/Window/Clip).
	// validator/runner 据此判定: 图里有 NeedsWindow 节点才要求 WindowTarget; 纯窗口无关
	// 容器 (Sleep/Log/Cron/Expr...) 免 WindowTarget. 新节点用了上述 ctx 服务务必置真
	// (漏置 → 该节点在无窗口容器里 SafeBackend 静默 no-op).
	NeedsWindow bool `json:"needsWindow,omitempty"`
	// IsGraphMarker — graph 结构标记节点 (SubgraphInput / SubgraphOutput).
	// runtime 在 dispatch_v5 / runRegionBody 里 special-route 跳过 Run.
	// 跟 IsVisualOnly 对称 (一个渲染标, 一个结构标), 跟 IsPureData 区分.
	// Register 允许 IsGraphMarker=true 时 zero capability.
	IsGraphMarker bool `json:"isGraphMarker,omitempty"`
	// DynamicOutputs — 出口名运行时按 config 推导, 不在静态 Outputs 里枚举 (e.g. Switch
	// 的 named-by-value case 出口). 置真时 ctx.Out(name) 放行任意 name (跳过 Outputs 成员检查);
	// 出口名合法性由节点自身 + validator 静态保证. Outputs 仍可列固定兜底出口 (Switch 的 default).
	DynamicOutputs bool `json:"dynamicOutputs,omitempty"`
	// DynamicInputs — 动态 data-in pin 由 config.Inputs[] 声明, 不在静态 Inputs 里枚举
	// (Expr / Script). dispatch / validator / FE 按此标志走 ParseDynamicInputDecls
	// 解析声明列表; Inputs 仍可列固定静态 pin (Expr 的 Expression).
	DynamicInputs bool `json:"dynamicInputs,omitempty"`
}

type InputSpec struct {
	Name     string `json:"name"`
	Type     string `json:"type"`               // runtime 类型 tag
	Semantic string `json:"semantic,omitempty"` // UI 语义提示
	// CaptureType 仅 Semantic=="capture" 框有效，被捕获值的 VarType（number/string/bool/point/any）。
	CaptureType string       `json:"captureType,omitempty"`
	Required    bool         `json:"required,omitempty"`
	Advanced    bool         `json:"advanced,omitempty"`
	Default     any          `json:"default,omitempty"` // JSON 序列化用 json.Number
	Widget      WidgetSpec   `json:"widget,omitempty"`
	VisibleWhen *VisibleRule `json:"visibleWhen,omitempty"`
	Schema      *FieldSchema `json:"schema,omitempty"` // 结构化输入的数据 schema; 非 nil → FE StructuredInput
}

type OutputSpec struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`               // "Exec" / 其他数据类型
	Semantic string      `json:"semantic,omitempty"` // UI/dispatch 语义: "error" = 失败出口
	Data     []DataField `json:"data,omitempty"`     // 仅 Type=Exec 出口有
}

type DataField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional,omitempty"`
}

// EnumOption — dropdown 选项.
//   - 静态 dropdown (Spec 里 DropdownProps.Options): 只填 Value, label 由 FE i18n 持有
//     (node.<kind>.input.<name>.option.<value>).
//   - async source (RegisterAsyncSource 运行时返回, e.g. 模板键 / clip / subgraph):
//     Label 是动态数据 (非 UI 文案, 不可 i18n), 必须填.
type EnumOption struct {
	Value any    `json:"value"`           // json.Number for 精度
	Label string `json:"label,omitempty"` // 仅 async 源填; 静态 dropdown 留空走 i18n
}

type VisibleRule struct {
	Field     string         `json:"field,omitempty"`
	Equals    any            `json:"equals,omitempty"`
	NotEquals any            `json:"notEquals,omitempty"`
	In        []any          `json:"in,omitempty"`
	And       []*VisibleRule `json:"and,omitempty"`
	Or        []*VisibleRule `json:"or,omitempty"`
}

type TypeSpec struct {
	Tag        string `json:"tag"`
	GoType     string `json:"goType"`
	WidgetKind string `json:"widgetKind"`
	Color      string `json:"color"`
	Doc        string `json:"doc,omitempty"`
}
