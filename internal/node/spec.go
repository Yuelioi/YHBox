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

	IsPureData   bool `json:"isPureData,omitempty"`
	IsVisualOnly bool `json:"isVisualOnly,omitempty"`
	// IsGraphMarker — graph 结构标记节点 (SubgraphInput / SubgraphOutput).
	// runtime 在 dispatch_v5 / runRegionBody 里 special-route 跳过 Run.
	// 跟 IsVisualOnly 对称 (一个渲染标, 一个结构标), 跟 IsPureData 区分.
	// Register 允许 IsGraphMarker=true 时 zero capability.
	IsGraphMarker bool `json:"isGraphMarker,omitempty"`
}

type InputSpec struct {
	Name        string       `json:"name"`
	Type        string       `json:"type"`               // runtime 类型 tag
	Semantic    string       `json:"semantic,omitempty"` // UI 语义提示
	Required    bool         `json:"required,omitempty"`
	Advanced    bool         `json:"advanced,omitempty"`
	Default     any          `json:"default,omitempty"` // JSON 序列化用 json.Number
	Widget      WidgetSpec   `json:"widget,omitempty"`
	VisibleWhen *VisibleRule `json:"visibleWhen,omitempty"`
}

type OutputSpec struct {
	Name string      `json:"name"`
	Type string      `json:"type"`           // "Exec" / 其他数据类型
	Data []DataField `json:"data,omitempty"` // 仅 Type=Exec 出口有
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
