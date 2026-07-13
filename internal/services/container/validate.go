package container

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	nodepkg "github.com/yottaapp/yotta/internal/node"
)

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

// Validate 校验 Container 完整性. sgs = 引用闭包解析出的子图集 (见 ValidateContainer)。
// 任何 SeverityError → 返 *ValidationFailure 含完整错误列表;
// 只有 warning → 返 nil (warning 通过 ValidateContainer 直接获取).
// Save / Load 都调一次; 前端 "检查" 按钮 / 试运行前主动跑走 Service.ValidateContainerByID.
func (c *Container) Validate(sgs []Subgraph) error {
	return c.ValidateWithRegistry(sgs, nodepkg.DefaultRegistrySnapshot())
}

func (c *Container) ValidateWithRegistry(sgs []Subgraph, registry nodepkg.RegistryReader) error {
	errs := ValidateContainerWithRegistry(c, sgs, registry)
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

// pinExists 检查 (kind, pin) 是否合法。out=true 查 out pins / data-out；out=false 查 in pins / data-in。
// data-in 走 static-only 路径 — Expr 等动态 data-in 节点需 cfg, caller 用 dataInPinTypeForNode.
func pinExists(registry nodepkg.RegistryReader, kind, pin string, out bool) bool {
	rn, ok := registry.Get(kind)
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

// execOutPinsForNode 返该节点的合法 exec-out pin set. nodepkg Spec.Outputs Type="Exec" 静态查.
// Subgraph / CollapsedNode 的动态 (按 callee OutputPins) 由 validateInvalidPins 单独处理.
func execOutPinsForNode(n *GraphNode) map[string]struct{} {
	return execOutPinsForNodeWithRegistry(nodepkg.DefaultRegistrySnapshot(), n)
}

func execOutPinsForNodeWithRegistry(registry nodepkg.RegistryReader, n *GraphNode) map[string]struct{} {
	pins := map[string]struct{}{}
	if n == nil {
		return pins
	}
	// Switch: 出口动态 = config.cases 里每个 case 值 + 'default' (named-by-value).
	// 镜像 nodeRegistry/adapter.ts DYNAMIC_EXEC_OUT.Switch + switch.go Run.
	if n.Kind == "Switch" {
		cfg, _ := ParseSwitchConfig(n)
		for _, c := range cfg.Cases {
			if c != "" {
				pins[c] = struct{}{}
			}
		}
		pins["default"] = struct{}{}
		return pins
	}
	rn, ok := registry.Get(n.Kind)
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
	return nodeHasExecOutPinWithRegistry(nodepkg.DefaultRegistrySnapshot(), n, pin)
}

func nodeHasExecOutPinWithRegistry(registry nodepkg.RegistryReader, n *GraphNode, pin string) bool {
	_, ok := execOutPinsForNodeWithRegistry(registry, n)[pin]
	return ok
}

// canonPinType 把 nodepkg Spec 的 Type tag (PascalCase, e.g. "Number") 转 validator/data-graph
// 的 lowercase 风格 ("number"). Exec pin 返 "" (Exec 不算 data type, 调用方靠 != "" 判断).
//
// 新框架补充类型 (Integer / Duration / JSON / Color / Rect / Time) 映射回 validator 已有的 5 类
// (number/string/bool/point/any), 让 literal 校验 + data-graph 类型比较仍走单一坐标系.
// Integer / Duration JSON 值都是数字 (Integer=ms; Duration ms 整数); JSON/Color/Rect/Time 暂归 any.
func canonPinType(t string) string {
	return nodepkg.CanonicalPinType(t)
}

// Normalize self-heal — 填默认值 (SchemaVersion/Graph.ID/Graph.SchemaVersion) + 子图补缺
// SubgraphOutput. self-heal 的唯一入口.
func (c *Container) Normalize() {
	if c.SchemaVersion == 0 {
		c.SchemaVersion = CurrentSchemaVersion
	}
	c.Hotkey = strings.TrimSpace(c.Hotkey)
	if c.Graph.Nodes == nil {
		c.Graph.Nodes = []GraphNode{}
	}
	if c.Graph.Edges == nil {
		c.Graph.Edges = []GraphEdge{}
	}
	// v2 兜底：写盘前自动填 Graph.ID + Graph.SchemaVersion
	if c.Graph.ID == "" {
		c.Graph.ID = uuid.NewString()
	}
	if c.Graph.SchemaVersion == 0 {
		c.Graph.SchemaVersion = GraphSchemaVersion
	}
	// 子图已全局化 — self-heal 与 RequiredGlobals 派生在全局 SubgraphStore 的保存路径跑,
	// 不再挂在容器 Normalize 上.
}

// --- Registry-backed helpers (nodepkg-only) ---

// knownKind kind 是否在 nodepkg 注册.
func knownKind(registry nodepkg.RegistryReader, kind string) bool {
	_, ok := registry.Get(kind)
	return ok
}

// dataInPinTypeForKind 返该 data-in pin 的 canonical lowercase type (number/string/bool/point/any).
// "" = 不是已注册的 data-in pin (e.g. Exec pin / kind 未注册 / pin 不存在).
// Static-only — Expr's dynamic inputs need cfg, use dataInPinTypeForNode.
func dataInPinTypeForKind(registry nodepkg.RegistryReader, kind, pinName string) string {
	rn, ok := registry.Get(kind)
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

// dataInPinSemanticForKind 返该 data-in pin 的 Semantic ("TemplateKey" 等), 无则 "".
// literal 校验用它识别 list 型 pin (e.g. TemplateKey = 字符串列表).
func dataInPinSemanticForKind(registry nodepkg.RegistryReader, kind, pinName string) string {
	rn, ok := registry.Get(kind)
	if !ok {
		return ""
	}
	for _, ip := range rn.Spec.Inputs {
		if ip.Name == pinName {
			return ip.Semantic
		}
	}
	return ""
}

// dataInPinTypeForNode cfg-aware 变种 — DynamicInputs 节点 config.Inputs[] 动态声明的
// pin 走 ParseDynamicInputDecls 查.
func dataInPinTypeForNode(registry nodepkg.RegistryReader, n *GraphNode, pinName string) string {
	if n == nil {
		return ""
	}
	if t := dataInPinTypeForKind(registry, n.Kind, pinName); t != "" {
		return t
	}
	if rn, ok := registry.Get(n.Kind); ok && rn.Spec.DynamicInputs {
		for _, in := range ParseDynamicInputDecls(n) {
			if in.Name == pinName && in.Type != "" {
				return strings.ToLower(in.Type)
			}
		}
	}
	return ""
}

// dataOutPinTypeForKind 同上 outputs.
// 动态类型节点 (GetVar / Expr / GetParam) Spec 里登记 "any" / "*",
// 真实类型由 caller 按 config 解析 — validateDataPinTypes 已做.
func dataOutPinTypeForKind(registry nodepkg.RegistryReader, kind, pinName string) string {
	rn, ok := registry.Get(kind)
	if !ok {
		return ""
	}
	for _, op := range rn.Spec.Outputs {
		if op.Type == nodepkg.TypeExec {
			// exec 出口本身是 exec-out (非 data); 但它的 Data 字段 (Fail 出口的 Error/Code)
			// 算 data-out —— 可被 data 边消费, 值由 exec-data 沿该 exec 边带下来 (runtime bridge 取).
			for _, f := range op.Data {
				if f.Name == pinName {
					return canonPinType(f.Type)
				}
			}
			continue
		}
		if op.Name == pinName {
			return canonPinType(op.Type)
		}
	}
	return ""
}

// IsExecOutputDataField reports whether (kind, pin) names a Data field nested under
// an exec output (e.g. RunProgram.Fail 的 Error/Code). 这类 pin 连线/校验上算 data-out
// (IsDataOutPin 真), 但值不靠 pure-data pull —— 源 fire 时存进 per-run held output 缓存,
// runtime 经 pullDataPin 任意距离直连读 (见 ContainerRunner.captureExecOutputs / pullDataPin).
func IsExecOutputDataField(kind, pin string) bool {
	return IsExecOutputDataFieldWithRegistry(nodepkg.DefaultRegistrySnapshot(), kind, pin)
}

func IsExecOutputDataFieldWithRegistry(registry nodepkg.RegistryReader, kind, pin string) bool {
	rn, ok := registry.Get(kind)
	if !ok {
		return false
	}
	for _, op := range rn.Spec.Outputs {
		if op.Type != nodepkg.TypeExec {
			continue
		}
		for _, f := range op.Data {
			if f.Name == pin {
				return true
			}
		}
	}
	return false
}

// IsDataOutPin reports whether (kind, pin) is a registered data-out pin.
// Centralizes the "is this a data edge?" predicate — validator + runtime
// must agree on this rule to keep edge-type derivation consistent.
func IsDataOutPin(kind, pin string) bool {
	return IsDataOutPinWithRegistry(nodepkg.DefaultRegistrySnapshot(), kind, pin)
}

func IsDataOutPinWithRegistry(registry nodepkg.RegistryReader, kind, pin string) bool {
	return dataOutPinTypeForKind(registry, kind, pin) != ""
}

// dataOutPinTypeForNode — config-aware 变种: 静态 + DynamicDataFields 节点 config.Outputs[]
// 动态声明的 Data 字段 (AI 结构化输出)。让 AI.red 这类动态输出字段也算 data-out, 可直连。
func dataOutPinTypeForNode(registry nodepkg.RegistryReader, n *GraphNode, pinName string) string {
	if n == nil {
		return ""
	}
	if t := dataOutPinTypeForKind(registry, n.Kind, pinName); t != "" {
		return t
	}
	if rn, ok := registry.Get(n.Kind); ok && rn.Spec.DynamicDataFields {
		for _, o := range ParseDynamicOutputDecls(n) {
			if o.Name == pinName && o.Type != "" {
				return canonPinType(o.Type)
			}
		}
	}
	return ""
}

// IsDataOutPinNode — config-aware IsDataOutPin (含动态输出字段)。
func IsDataOutPinNode(n *GraphNode, pin string) bool {
	return IsDataOutPinNodeWithRegistry(nodepkg.DefaultRegistrySnapshot(), n, pin)
}

func IsDataOutPinNodeWithRegistry(registry nodepkg.RegistryReader, n *GraphNode, pin string) bool {
	return dataOutPinTypeForNode(registry, n, pin) != ""
}

// IsExecOutputDataFieldNode — config-aware IsExecOutputDataField: 静态 + DynamicDataFields
// 的 config.Outputs[] 字段。值同样经 per-run held output 缓存 (captureExecOutputs 写 / pullDataPin 读) 直连下游。
func IsExecOutputDataFieldNode(n *GraphNode, pin string) bool {
	return IsExecOutputDataFieldNodeWithRegistry(nodepkg.DefaultRegistrySnapshot(), n, pin)
}

func IsExecOutputDataFieldNodeWithRegistry(registry nodepkg.RegistryReader, n *GraphNode, pin string) bool {
	if n == nil {
		return false
	}
	if IsExecOutputDataFieldWithRegistry(registry, n.Kind, pin) {
		return true
	}
	if rn, ok := registry.Get(n.Kind); ok && rn.Spec.DynamicDataFields {
		for _, o := range ParseDynamicOutputDecls(n) {
			if o.Name == pin {
				return true
			}
		}
	}
	return false
}

// hasUnwiredNeedsWindowNode — 是否存在「Window 输入未连」的 NeedsWindow 节点(主图或子图)。
// 连了 Window 的节点派发期自带覆盖窗口, 不需要 Win32WindowTarget; 没连的会回落活动窗口, 故仍要求 Win32WindowTarget。
func hasUnwiredNeedsWindowNode(registry nodepkg.RegistryReader, c *Container, sgs []Subgraph) bool {
	if graphHasUnwiredWindowNode(registry, c.Graph) {
		return true
	}
	for i := range sgs {
		if graphHasUnwiredWindowNode(registry, sgs[i].Graph) {
			return true
		}
	}
	return false
}

func graphHasUnwiredWindowNode(registry nodepkg.RegistryReader, g Graph) bool {
	for i := range g.Nodes {
		rn, ok := registry.Get(g.Nodes[i].Kind)
		if !ok || !rn.Spec.NeedsWindow {
			continue
		}
		if !windowPinWired(g, g.Nodes[i].ID) {
			return true
		}
	}
	return false
}

// hasUnwiredNeedsTargetNode — 是否存在「需要自动化目标」且未显式连 Window override 的节点。
// 连了 Window 的 target-aware 节点可用 Win32 Window data edge 作为本节点覆盖目标;
// 未连时必须依赖图中的 target selection node 或 Windows 默认 Win32WindowTarget。
func hasUnwiredNeedsTargetNode(registry nodepkg.RegistryReader, c *Container, sgs []Subgraph) bool {
	if graphHasUnwiredTargetNode(registry, c.Graph) {
		return true
	}
	for i := range sgs {
		if graphHasUnwiredTargetNode(registry, sgs[i].Graph) {
			return true
		}
	}
	return false
}

func graphHasUnwiredTargetNode(registry nodepkg.RegistryReader, g Graph) bool {
	for i := range g.Nodes {
		rn, ok := registry.Get(g.Nodes[i].Kind)
		if !ok || !rn.Spec.NeedsTarget {
			continue
		}
		if !windowPinWired(g, g.Nodes[i].ID) {
			return true
		}
	}
	return false
}

func windowPinWired(g Graph, nodeID string) bool {
	target := nodeID + ".Window"
	for _, e := range g.Edges {
		if e.To == target {
			return true
		}
	}
	return false
}
