// internal/node/spec.go
// Package node 节点系统核心 — declarative spec + runtime registry + inspector-first.
package node

// TypeExec exec pin 类型 tag. 抽 const 避免 "Exec" magic string 散落.
// 跟 InputSpec.Type / OutputSpec.Type 字段值对齐.
const TypeExec = "Exec"

type TargetCapability string

// RuntimeCapability identifies a ServiceBundle dependency that must be wired
// before a node can execute. It is separate from TargetCapability: the latter
// describes what an active automation controller can do, while this list
// describes which runtime service ports the node itself consumes.
type RuntimeCapability string

type DynamicPortRole string

const (
	DynamicPortInput      DynamicPortRole = "input"
	DynamicPortOutput     DynamicPortRole = "output"
	DynamicPortOutputData DynamicPortRole = "outputData"
)

type DynamicPortShape string

const (
	DynamicPortNames           DynamicPortShape = "names"
	DynamicPortNameTypeRecords DynamicPortShape = "nameTypeRecords"
	DynamicPortGraphInterface  DynamicPortShape = "graphInterface"
)

// DynamicPortSpec declares how a node's config or bound graph interface
// contributes ports that are not present in the static Inputs/Outputs lists.
type DynamicPortSpec struct {
	Role         DynamicPortRole  `json:"role"`
	ConfigKey    string           `json:"configKey"`
	Shape        DynamicPortShape `json:"shape"`
	FixedType    string           `json:"fixedType"`
	ParentOutput string           `json:"parentOutput"`
	MinItems     int              `json:"minItems"`
	MaxItems     int              `json:"maxItems"`
}

const (
	RuntimeCapabilityVision      RuntimeCapability = "vision"
	RuntimeCapabilityLog         RuntimeCapability = "log"
	RuntimeCapabilityInput       RuntimeCapability = "input"
	RuntimeCapabilityVars        RuntimeCapability = "vars"
	RuntimeCapabilityParams      RuntimeCapability = "params"
	RuntimeCapabilityWindow      RuntimeCapability = "window"
	RuntimeCapabilityTarget      RuntimeCapability = "target"
	RuntimeCapabilityApp         RuntimeCapability = "app"
	RuntimeCapabilityCapture     RuntimeCapability = "capture"
	RuntimeCapabilityStopwatches RuntimeCapability = "stopwatches"
	RuntimeCapabilityClip        RuntimeCapability = "clip"
	RuntimeCapabilitySubgraphs   RuntimeCapability = "subgraphs"
	RuntimeCapabilityAI          RuntimeCapability = "ai"
	RuntimeCapabilityRegistry    RuntimeCapability = "registry"
)

const (
	TargetCapabilityScreenshot   TargetCapability = "screenshot"
	TargetCapabilityClick        TargetCapability = "click"
	TargetCapabilityMove         TargetCapability = "move"
	TargetCapabilityScroll       TargetCapability = "scroll"
	TargetCapabilityMouseButton  TargetCapability = "mouse-button"
	TargetCapabilityDrag         TargetCapability = "drag"
	TargetCapabilityMoveRelative TargetCapability = "move-relative"
	TargetCapabilityKeyState     TargetCapability = "key-state"
	TargetCapabilityText         TargetCapability = "text"
	TargetCapabilityStartApp     TargetCapability = "start-app"
	TargetCapabilityStopApp      TargetCapability = "stop-app"
)

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
	// NeedsTarget — 节点 Run 依赖当前自动化目标能力 (Input/Capture/Vision 等
	// target-aware 服务). Win32/Android/Browser 都可经 controller factory 供给;
	// 不代表必须有 Win32 HWND。没有任何 target selection node 时, validator 仍按
	// Windows 默认体验提示补 Win32WindowTarget。
	NeedsTarget bool `json:"needsTarget,omitempty"`
	// TargetCapabilities — NeedsTarget 节点对 active target controller 的具体能力要求.
	// 名称与 automation/controller.Capability 字符串保持一致, validator 用 controller
	// profiles 做静态匹配, runtime adapter 仍保留执行期兜底。
	TargetCapabilities []TargetCapability `json:"targetCapabilities,omitempty"`
	// RuntimeCapabilities declares the ServiceBundle ports used by Run/Evaluate.
	// The engine validates these before calling node code, so a missing adapter is
	// an assembly error instead of a nil-interface panic.
	RuntimeCapabilities []RuntimeCapability `json:"runtimeCapabilities,omitempty"`
	// SupportedTargets — 节点可用的用户可见自动化目标类型, 由 NeedsWindow /
	// NeedsTarget / TargetCapabilities / PlatformTargets 派生后暴露给前端展示平台 badge。
	// 节点实现不要手写此字段; NodeService/Catalog 在导出前填充。
	SupportedTargets []string `json:"supportedTargets,omitempty"`
	// PlatformTargets — 仅用于展示/目录的目标平台标识。给 Windows-only 但不依赖
	// 当前活动窗口的节点使用, 例如 RunProgram / WaitWindow / GetWindow。
	// 不参与 validator/runtime 依赖判断; 需要当前窗口时仍应使用 NeedsWindow。
	PlatformTargets []string `json:"platformTargets,omitempty"`
	// NeedsWindow — legacy Win32 HWND requirement: 节点 Run 依赖 Windows 窗口
	// (调 ctx.Input/Capture/Vision/Window/Clip 等 Win32-backed 服务).
	// validator/runner 据此判定直接 Win32 窗口操作是否需要 Win32WindowTarget;
	// Android/Browser 等非 HWND 自动化对象必须走 NeedsTarget / TargetService /
	// controller capabilities, 不要把 NeedsWindow 当通用 Target requirement.
	NeedsWindow bool `json:"needsWindow,omitempty"`
	// NeedsForeground — 节点向 Windows 窗口注入输入(SendInput 后端需前台焦点)。
	// 派发期 Window 覆盖时, 若后端是 sendinput 且本标志为真, 框架补拉一次前台。
	// 输入类节点(Click/KeyPress...)置真。Browser/Android active target 会忽略此
	// Win32 前台语义, 由对应 controller 决定激活策略。
	NeedsForeground bool `json:"needsForeground,omitempty"`
	// IsGraphMarker — graph 结构标记节点 (SubgraphInput / SubgraphOutput).
	// runtime 在 dispatch_v5 / runRegionBody 里 special-route 跳过 Run.
	// 跟 IsVisualOnly 对称 (一个渲染标, 一个结构标), 跟 IsPureData 区分.
	// Register 允许 IsGraphMarker=true 时 zero capability.
	IsGraphMarker bool              `json:"isGraphMarker,omitempty"`
	DynamicPorts  []DynamicPortSpec `json:"dynamicPorts,omitempty"`
}

func HasDynamicPortRole(spec *Spec, role DynamicPortRole) bool {
	_, ok := DynamicPortForRole(spec, role)
	return ok
}

func DynamicPortForRole(spec *Spec, role DynamicPortRole) (DynamicPortSpec, bool) {
	if spec == nil {
		return DynamicPortSpec{}, false
	}
	for _, dynamic := range spec.DynamicPorts {
		if dynamic.Role == role {
			return dynamic, true
		}
	}
	return DynamicPortSpec{}, false
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
	Schema      *FieldSchema `json:"schema,omitempty"` // 结构化输入的数据 schema; 非 nil → FE StructuredInput
}

// WindowInputSpec — NeedsWindow 节点统一 spread 的可选窗口输入。连了→派发期作用在该窗口;
// 不连→当前活动窗口。框架在 execNodeViaFramework 解释此 pin(节点 Run 无需读它)。
func WindowInputSpec() InputSpec {
	return InputSpec{Name: "Window", Type: "Window"}
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

// BindableFields 某节点 spec 可被「输出捕获」绑定到变量的字段名 (Spec C 单一来源)。
// = 非纯数据节点 (IsPureData=false) 所有 exec 出口携带的 Data 字段名 (去重)。
// 纯数据节点 (GetVar/Now/Expr/PureFunc) 无可绑字段 — 其输出是连线源, 非捕获。
// 前端 Inspector「输出」组 / validator / useVarMutations 共用此派生规则 (派生一致, 不各维护白名单)。
func BindableFields(spec *Spec) []string {
	if spec == nil || spec.IsPureData {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, o := range spec.Outputs {
		if o.Type != TypeExec {
			continue
		}
		for _, f := range o.Data {
			if !seen[f.Name] {
				seen[f.Name] = true
				out = append(out, f.Name)
			}
		}
	}
	return out
}

// BindableFieldsForNode = 静态字段加 outputData descriptor 声明的 config.Outputs[] 名称。
// 让 config 声明的动态输出 Data 字段也可被捕获绑定; FE 输出组 / capture 校验共用此派生。
func BindableFieldsForNode(spec *Spec, config map[string]any) []string {
	fields := BindableFields(spec)
	dynamic, ok := DynamicPortForRole(spec, DynamicPortOutputData)
	if !ok || dynamic.Shape != DynamicPortNameTypeRecords {
		return fields
	}
	seen := map[string]bool{}
	for _, f := range fields {
		seen[f] = true
	}
	raw, _ := config[dynamic.ConfigKey].([]any)
	for _, it := range raw {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["Name"].(string)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		fields = append(fields, name)
	}
	return fields
}

// EnumOption — dropdown 选项.
//   - 静态 dropdown (Spec 里 DropdownProps.Options): 只填 Value, label 由 FE i18n 持有
//     (node.<kind>.input.<name>.option.<value>).
//   - async source (RegisterAsyncSource 运行时返回, e.g. 模板键 / clip / subgraph):
//     Label 是动态数据 (非 UI 文案, 不可 i18n), 必须填.
type EnumOption struct {
	Value any            `json:"value"`           // json.Number for 精度
	Label string         `json:"label,omitempty"` // 仅 async 源填; 静态 dropdown 留空走 i18n
	Meta  map[string]any `json:"meta,omitempty"`  // 仅 async 源填; 供 inspector 应用到 sibling inputs
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
