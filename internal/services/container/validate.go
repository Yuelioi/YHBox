package container

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	nodepkg "yhbox/internal/node"
)

var varNameRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
var edgeFormatRE = regexp.MustCompile(`^[a-zA-Z0-9_-]+\.[a-zA-Z0-9_.]+$`)

// ValidationFailure aggregates all error-severity entries from a single Validate call.
// Implements the error interface — callers can errors.As(&vf) to recover the full list
// for UI display (e.g. ValidationErrorPanel emits one row per Errors[i]).
//
// Warnings are NOT included here — call ValidateContainer directly to access them.
type ValidationFailure struct {
	Errors []ValidationError
}

// Error: B5 ship 后 Code+Params 模型 — i18n 走 FE t(), 后端 log 显示 Code+Params 字面.
// 不漂亮但够 debug; UI 全走 ValidationErrorPanel 不读 Error().
func (f *ValidationFailure) Error() string {
	if len(f.Errors) == 0 {
		return "container: validation passed" // unreachable from Validate(); defensive only
	}
	if len(f.Errors) == 1 {
		e := f.Errors[0]
		return fmt.Sprintf("%s %v", e.Code, e.Params)
	}
	return fmt.Sprintf("%s %v (and %d more)", f.Errors[0].Code, f.Errors[0].Params, len(f.Errors)-1)
}

// Validate 校验 Container 完整性. 任何 SeverityError → 返 *ValidationFailure 含完整错误列表;
// 只有 warning → 返 nil (warning 通过 ValidateContainer 直接获取).
// Save / Load 都调一次; 前端 "检查" 按钮 / 试运行前主动跑走 Service.ValidateContainerByID.
func (c *Container) Validate() error {
	errs := ValidateContainer(c)
	var fatal []ValidationError
	for _, e := range errs {
		if e.Severity == SeverityError {
			fatal = append(fatal, e)
		}
	}
	if len(fatal) == 0 {
		return nil
	}
	return &ValidationFailure{Errors: fatal}
}

func splitRef(ref string) (nodeID, pin string) {
	idx := strings.Index(ref, ".")
	if idx < 0 {
		return ref, ""
	}
	return ref[:idx], ref[idx+1:]
}

// cfgString 读 cfg[k] as string. 缺失 / 非 string → "".
func cfgString(cfg map[string]any, k string) string {
	if v, ok := cfg[k].(string); ok {
		return v
	}
	return ""
}

// pinExists 检查 (kind, pin) 是否合法。out=true 查 out pins / data-out；out=false 查 in pins / data-in。
// data-in 走 static-only 路径 — Expr 等动态 data-in 节点需 cfg, caller 用 dataInPinTypeForNode.
func pinExists(kind, pin string, out bool) bool {
	rn, ok := nodepkg.Get(kind)
	if !ok {
		return false
	}
	if out {
		for _, op := range rn.Spec.Outputs {
			if op.Name == pin {
				return true
			}
		}
		return false
	}
	for _, ip := range rn.Spec.Inputs {
		if ip.Name == pin {
			return true
		}
	}
	return false
}

// execInPinsOf 返回 kind 的 exec-in pin 集合 (Type="Exec" Inputs).
// nil = 没有 exec-in (Start / OnEvent / SubgraphInput / MouseCalibration / pure-data 节点).
// 未注册 kind 防御性返 ["in"].
func execInPinsOf(kind string) []string {
	rn, ok := nodepkg.Get(kind)
	if !ok {
		return []string{"in"}
	}
	var out []string
	for _, ip := range rn.Spec.Inputs {
		if ip.Type == nodepkg.TypeExec {
			out = append(out, ip.Name)
		}
	}
	return out
}

func isExecInPin(kind, pin string) bool {
	for _, p := range execInPinsOf(kind) {
		if p == pin {
			return true
		}
	}
	return false
}

// execOutPinsForNode 返该节点的合法 exec-out pin set. nodepkg Spec.Outputs Type="Exec" 静态查.
// Subgraph / CollapsedNode 的动态 (按 callee OutputPins) 由 validateInvalidPins 单独处理.
func execOutPinsForNode(n *GraphNode) map[string]struct{} {
	pins := map[string]struct{}{}
	if n == nil {
		return pins
	}
	rn, ok := nodepkg.Get(n.Kind)
	if !ok {
		return pins
	}
	for _, op := range rn.Spec.Outputs {
		if op.Type == nodepkg.TypeExec {
			pins[op.Name] = struct{}{}
		}
	}
	return pins
}

// nodeHasExecOutPin O(1) 校验 pin 存在性. 替代旧 pinExists(kind, pin, true) 路径.
func nodeHasExecOutPin(n *GraphNode, pin string) bool {
	_, ok := execOutPinsForNode(n)[pin]
	return ok
}

// canonPinType 把 nodepkg Spec 的 Type tag (PascalCase, e.g. "Number") 转 validator/data-graph
// 的 lowercase 风格 ("number"). Exec pin 返 "" (Exec 不算 data type, 调用方靠 != "" 判断).
//
// 新框架补充类型 (Integer / Duration / JSON / Color / Rect / Time) 映射回 validator 已有的 5 类
// (number/string/bool/point/any), 让 literal 校验 + data-graph 类型比较仍走单一坐标系.
// Integer / Duration JSON 值都是数字 (Integer=ms; Duration ms 整数); JSON/Color/Rect/Time 暂归 any.
func canonPinType(t string) string {
	switch t {
	case "Number", "Integer", "Duration":
		return "number"
	case "String":
		return "string"
	case "Bool":
		return "bool"
	case "Point":
		return "point"
	case "*", "JSON", "Color", "Rect", "Time":
		return "any"
	case "Exec":
		return ""
	}
	return strings.ToLower(t)
}

// Normalize self-heal — 填默认值 (SchemaVersion/RunMode/Graph.ID/Graph.Version) + 子图补缺
// SubgraphOutput. B3 三阶段中 Normalize 阶段的唯一入口.
func (c *Container) Normalize() {
	if c.SchemaVersion == 0 {
		c.SchemaVersion = CurrentSchemaVersion
	}
	c.Hotkey = strings.TrimSpace(c.Hotkey)
	c.RunMode = strings.TrimSpace(c.RunMode)
	if c.RunMode != "foreground" && c.RunMode != "background" {
		c.RunMode = "background"
	}
	if c.Graph.Nodes == nil {
		c.Graph.Nodes = []GraphNode{}
	}
	if c.Graph.Edges == nil {
		c.Graph.Edges = []GraphEdge{}
	}
	// v2 兜底：写盘前自动填 Graph.ID + Graph.Version
	if c.Graph.ID == "" {
		c.Graph.ID = uuid.NewString()
	}
	if c.Graph.Version == 0 {
		c.Graph.Version = GraphSchemaVersion
	}
	// B3: subgraph self-heal 归这里统一入口 — Save 路径之前漏 sg.normalize, 现在 Container.Normalize 一次跑完.
	// B11: RequiredGlobals 一同派生, 跟 c.Vars 同步. validator + library import 都依赖此字段.
	for i := range c.Subgraphs {
		normalizeSubgraph(&c.Subgraphs[i])
		c.Subgraphs[i].RequiredGlobals = computeRequiredGlobals(&c.Subgraphs[i], c.Vars)
	}
}

// --- Registry-backed helpers (nodepkg-only) ---

// knownKind kind 是否在 nodepkg 注册.
func knownKind(kind string) bool {
	_, ok := nodepkg.Get(kind)
	return ok
}

// dataInPinTypeForKind 返该 data-in pin 的 canonical lowercase type (number/string/bool/point/any).
// "" = 不是已注册的 data-in pin (e.g. Exec pin / kind 未注册 / pin 不存在).
// Static-only — Expr's dynamic inputs need cfg, use dataInPinTypeForNode.
func dataInPinTypeForKind(kind, pinName string) string {
	rn, ok := nodepkg.Get(kind)
	if !ok {
		return ""
	}
	for _, ip := range rn.Spec.Inputs {
		if ip.Name != pinName {
			continue
		}
		if ip.Type == nodepkg.TypeExec {
			return ""
		}
		return canonPinType(ip.Type)
	}
	return ""
}

// dataInPinTypeForNode cfg-aware 变种 — Expr.inputs[] 动态声明的 pin 走 ParseExprConfig 查.
func dataInPinTypeForNode(n *GraphNode, pinName string) string {
	if n == nil {
		return ""
	}
	if t := dataInPinTypeForKind(n.Kind, pinName); t != "" {
		return t
	}
	if n.Kind == "Expr" {
		cfg, _ := ParseExprConfig(n)
		for _, in := range cfg.Inputs {
			if in.Name == pinName && in.Type != "" {
				return strings.ToLower(in.Type)
			}
		}
	}
	return ""
}

// dataOutPinTypeForKind 同上 outputs.
// 动态类型节点 (GetVar / Expr / GetSys / GetParam) Spec 里登记 "any" / "*",
// 真实类型由 caller 按 config 解析 — validateDataPinTypes 已做.
func dataOutPinTypeForKind(kind, pinName string) string {
	rn, ok := nodepkg.Get(kind)
	if !ok {
		return ""
	}
	for _, op := range rn.Spec.Outputs {
		if op.Name != pinName {
			continue
		}
		if op.Type == nodepkg.TypeExec {
			return ""
		}
		return canonPinType(op.Type)
	}
	return ""
}

// IsDataOutPin reports whether (kind, pin) is a registered data-out pin.
// Centralizes the "is this a data edge?" predicate — validator + runtime
// must agree on this rule to keep edge-type derivation consistent.
func IsDataOutPin(kind, pin string) bool {
	return dataOutPinTypeForKind(kind, pin) != ""
}
