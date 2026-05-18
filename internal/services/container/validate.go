package container

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// NOTE(v2 migration)：本文件里的 Validate / Normalize 实现保留过渡，
// 但 v2 上线后所有调用方应该改去用 validator.go 里的 ValidateContainer（返回结构化 []ValidationError）。
// 等 1.7 把 ValidateContainer 实现完，本文件的 Validate 改成薄包装 → 调 ValidateContainer 把第一个 error 转 error。

// KnownNodeKinds 节点 kind 白名单。
var KnownNodeKinds = map[string]bool{
	"Start": true, "Sleep": true, "Loop": true, "If": true,
	"Parallel": true, "Race": true, "Stop": true, "Break": true, "Continue": true,
	"SetVar": true, "IncVar": true,
	"WaitTemplate": true, "CheckTemplate": true, "ClickTemplate": true,
	"DetectColor": true,
	"ClickAt": true, "KeyPress": true, "MouseMoveRel": true, "Scroll": true,
	"OnEvent": true,
	"Log": true, "Toast": true,

	// v2 新增：删 InvokeAction，加这 5 个
	"Subgraph": true, "SubgraphInput": true, "SubgraphOutput": true,
	"BringGameForeground": true,
	"MouseCalibration":    true,
	"PlayClip":            true,

	// v3 Phase C 新增
	"DetectColorHSV":  true,
	"ROIColorScan":    true,
	"Screenshot":      true,
	"KeyHoldStart":    true,
	"KeyHoldStop":     true,
	"MouseHoldStart":  true,
	"MouseHoldStop":   true,
	"Try":             true,
	"Throw":           true,
	"StopwatchStart":  true,
	"StopwatchStop":   true,
	"StopwatchRead":   true,

	// v4 (data-flow / variables / expressions)
	"GetVar":   true,
	"Expr":     true,
	"GetSys":   true,
	"GetParam": true,
	// v4 Phase B (B6-B8) will add: Add/Sub/.../Select
	// v4 Phase C/D adds: CommentBox, CollapsedNode
}

// yieldKinds Loop body 至少含一个，避免 forever loop CPU 100%。
// DetectColor 走截屏 + 像素扫，单次 ~3-10ms，算 yield。
var yieldKinds = map[string]bool{
	"Sleep": true, "WaitTemplate": true, "CheckTemplate": true,
	"ClickTemplate": true, "DetectColor": true, "OnEvent": true,
	// v3 Phase C 新增
	"DetectColorHSV": true,
	"ROIColorScan":   true,
	"Try":            true, // Try 内必含 subgraph, 子图调度本身就 yield
}

// execInPins kind → 该 kind 接受的 exec-in pin 名集合。
// 默认 = {"in"}；OnEvent / Start / SubgraphInput / MouseCalibration 没 exec-in。
var execInPins = map[string][]string{
	"Start":            nil,
	"OnEvent":          nil,
	"SubgraphInput":    nil, // 子图入口节点是被外部跳进, 没 exec-in pin
	"MouseCalibration": nil, // 声明式节点, 不参与执行流
	"Loop":             {"in", "loopback"}, // body 末尾接回 loopback 是单独 pin
}

// execOutPins kind → exec-out pin 名集合。默认 = {"out"}。
var execOutPins = map[string][]string{
	"Start":         {"out"},
	"Sleep":         {"out"},
	"Loop":          {"body", "complete"},
	"If":            {"then", "else"},
	"Parallel":      {"complete"}, // branch0..N-1 动态生成
	"Race":          {"complete"}, // branch0..N-1 动态生成
	"Stop":          nil,
	"Break":         nil,
	"Continue":      nil,
	"SetVar":        {"out"},
	"IncVar":        {"out"},
	"WaitTemplate":  {"found", "timeout"},
	"CheckTemplate": {"yes", "no"},
	"ClickTemplate": {"done", "timeout"},
	"DetectColor":   {"yes", "no"},
	"ClickAt":       {"out"},
	"KeyPress":      {"out"},
	"MouseMoveRel":  {"out"},
	"Scroll":        {"out"},
	"OnEvent":       {"out"},
	"Log":           {"out"},
	"Toast":         {"out"},
	// v2 新增 kind
	"BringGameForeground": {"out"},
	"SubgraphInput":       {"out"}, // 子图入口对内部节点 dispatch
	"SubgraphOutput":      nil,      // 子图终点, 退栈不再向下
	"MouseCalibration":    nil,      // 声明式
	"Subgraph":            nil,      // 动态 out pin = 绑定子图的 OutputPins decl ID, validateInvalidPins 单独处理
	"PlayClip":            {"out"},  // InputClip 回放, 阻塞到完成 / cancel

	// v3 Phase C 新增
	"DetectColorHSV": {"yes", "no", "timeout"},
	"ROIColorScan":   {"found", "notFound", "timeout"},
	"Screenshot":     {"done"},
	"KeyHoldStart":   {"out"},
	"KeyHoldStop":    {"out"},
	"MouseHoldStart": {"out"},
	"MouseHoldStop":  {"out"},
	"Try":            {"done", "timeout", "error"},
	"Throw":          nil, // terminal, 同 Stop/Break
	"StopwatchStart": {"out"},
	"StopwatchStop":  {"out"},
	"StopwatchRead":  {"out"},
	"Switch":         nil, // 动态 — execOutPinsForNode 派生 cases + default. nil 防 default fallback 返 ["out"]
}

// dataOutPins kind → data-out pin 名 → pin 类型 ("point" | "number" | "string" | "any")。
var dataOutPins = map[string]map[string]string{
	"WaitTemplate":  {"point": "point"},
	"CheckTemplate": {"point": "point"},
	"ClickTemplate": {"point": "point"},
	"Loop":          {"iter": "number"},
	"Race":          {"winnerIdx": "number"},

	// v3 Phase C 新增
	"DetectColorHSV": {"pixelCount": "number", "pixelRatio": "number"},
	"ROIColorScan":   {"clusters": "any", "clusterCount": "number"},
	"Screenshot":     {"path": "string"},
	"Try":            {"errorMsg": "string"},
	"StopwatchRead":  {"elapsedMs": "number"},
}

// dataInPins kind → data-in pin 名 → pin 类型。
// InvokeAction 的 params.* 动态由 Action.params 决定，这里仅处理静态 pin。
var dataInPins = map[string]map[string]string{
	"ClickAt": {"pos": "point"},
}

// exprConfigKeys kind → config 键名集合（值应为表达式，Validate 时 expr.Parse 测过）。
var exprConfigKeys = map[string][]string{
	"Sleep":         {"durationMs"},
	"Loop":          {"count", "condition"},
	"If":            {"condition"},
	"Parallel":      {"n"},
	"Race":          {"n"},
	"SetVar":        {"value"},
	"IncVar":        {"delta"},
	"WaitTemplate":  {"timeoutMs", "threshold"},
	"CheckTemplate": {"threshold"},
	"ClickTemplate": {"timeoutMs", "threshold"},
	"DetectColor":   {"minPixels"},
	"ClickAt":       {"xRatio", "yRatio", "durationMs"},
	"KeyPress":      {"durationMs"},
	"MouseMoveRel":  {"dx", "dy", "durationMs"},
	"Scroll":        {"xRatio", "yRatio", "delta"},
	"OnEvent":       {"pollIntervalMs", "maxConcurrent", "cooldownMs"},
	"Log":           {"message"},
	"Toast":         {"title", "message"},
}

var varNameRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
var edgeFormatRE = regexp.MustCompile(`^[a-zA-Z0-9_-]+\.[a-zA-Z0-9_.]+$`)

// Validate 校验 Container 完整性。Save / Load 都调一次。
// v2：薄包装 ValidateContainer，返回第一个 error 级别的错误。
func (c *Container) Validate() error {
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Severity == SeverityError {
			return fmt.Errorf("%s: %s", e.Code, e.Message)
		}
	}
	return nil
}

func splitRef(ref string) (nodeID, pin string) {
	idx := strings.Index(ref, ".")
	if idx < 0 {
		return ref, ""
	}
	return ref[:idx], ref[idx+1:]
}

// pinExists 检查 (kind, pin) 是否合法。out=true 查 out pins / data-out；out=false 查 in pins / data-in。
// 对 Parallel/Race 的 branch0..N-1 动态 pin 允许任意 "branchN"。
// InvokeAction 的 params.<X> 动态 data-in 允许任意 "params.X"。
func pinExists(kind, pin string, out bool) bool {
	if out {
		// exec-out
		for _, p := range execOutPins[kind] {
			if p == pin {
				return true
			}
		}
		// dynamic branchN for Parallel / Race
		if (kind == "Parallel" || kind == "Race") && strings.HasPrefix(pin, "branch") {
			return true
		}
		// data-out
		if m := dataOutPins[kind]; m != nil {
			if _, ok := m[pin]; ok {
				return true
			}
		}
		return false
	}
	// in：常规 exec-in
	for _, p := range execInPinsOf(kind) {
		if p == pin {
			return true
		}
	}
	// data-in
	if m := dataInPins[kind]; m != nil {
		if _, ok := m[pin]; ok {
			return true
		}
	}
	return false
}

func execInPinsOf(kind string) []string {
	if pins, ok := execInPins[kind]; ok {
		return pins
	}
	return []string{"in"} // default
}

func isExecInPin(kind, pin string) bool {
	for _, p := range execInPinsOf(kind) {
		if p == pin {
			return true
		}
	}
	return false
}

// execOutPinsForNode 按节点实例计算所有合法 exec out pin (返 set 让 caller O(1) 查).
// 动态节点 (Switch / Parallel / Race) 按 config 派生; 静态节点 fallback execOutPins 表.
// 内部自动 dedupe — schema 自洽不依赖 validator 拦截重复.
func execOutPinsForNode(n *GraphNode) map[string]struct{} {
	pins := map[string]struct{}{}
	if n == nil {
		return pins
	}
	switch n.Kind {
	case "Switch":
		cfg, _ := ParseSwitchConfig(n)
		for _, cs := range cfg.Cases {
			if cs == "" {
				continue
			}
			pins[cs] = struct{}{}
		}
		pins["default"] = struct{}{}
	case "Parallel", "Race":
		cfg, _ := ParseParallelConfig(n)
		for i := 0; i < cfg.N; i++ {
			pins[fmt.Sprintf("branch%d", i)] = struct{}{}
		}
		pins["complete"] = struct{}{}
	default:
		for _, p := range execOutPins[n.Kind] {
			pins[p] = struct{}{}
		}
	}
	return pins
}

// nodeHasExecOutPin O(1) 校验 pin 存在性. 替代旧 pinExists(kind, pin, true) 路径.
func nodeHasExecOutPin(n *GraphNode, pin string) bool {
	_, ok := execOutPinsForNode(n)[pin]
	return ok
}

// dataOutType 返该 pin 是否为 data-out + 类型。
func dataOutType(kind, pin string) (bool, string) {
	if m := dataOutPins[kind]; m != nil {
		if t, ok := m[pin]; ok {
			return true, t
		}
	}
	return false, ""
}

// dataInType 同上 in 方向。
func dataInType(kind, pin string) (bool, string) {
	if m := dataInPins[kind]; m != nil {
		if t, ok := m[pin]; ok {
			return true, t
		}
	}
	return false, ""
}

// isInsideLoop 沿 exec edge 反向 BFS，看能否到达任何 Loop 节点的 body out。
func isInsideLoop(nodeID string, g Graph, nodesByID map[string]*GraphNode) bool {
	// 建反向索引：to.nodeId → []fromRefs
	revIn := map[string][]string{}
	for _, e := range g.Edges {
		toN, _ := splitRef(e.To)
		revIn[toN] = append(revIn[toN], e.From)
	}
	visited := map[string]bool{nodeID: true}
	queue := []string{nodeID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, fromRef := range revIn[cur] {
			fromN, fromPin := splitRef(fromRef)
			n := nodesByID[fromN]
			if n != nil && n.Kind == "Loop" && fromPin == "body" {
				return true
			}
			if !visited[fromN] {
				visited[fromN] = true
				queue = append(queue, fromN)
			}
		}
	}
	return false
}

// loopBodyHasYield 从 loop.body 向下 BFS 检测 yield kind。遇到 loop 自身（loopback）则停。
func loopBodyHasYield(loopID string, g Graph, nodesByID map[string]*GraphNode) bool {
	// outIdx：fromNodeID → []toRefs（含 pin），过滤只走 exec edges。
	type edgeInfo struct {
		toNode, fromPin string
	}
	outIdx := map[string][]edgeInfo{}
	for _, e := range g.Edges {
		fromN, fromPin := splitRef(e.From)
		toN, _ := splitRef(e.To)
		outIdx[fromN] = append(outIdx[fromN], edgeInfo{toNode: toN, fromPin: fromPin})
	}
	// seed：loop.body → 下游节点
	visited := map[string]bool{}
	var queue []string
	for _, e := range outIdx[loopID] {
		if e.fromPin == "body" {
			queue = append(queue, e.toNode)
			visited[e.toNode] = true
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		n := nodesByID[cur]
		if n == nil {
			continue
		}
		if yieldKinds[n.Kind] {
			return true
		}
		if cur == loopID {
			// 走到 loop 自己（loopback），不再深入
			continue
		}
		for _, e := range outIdx[cur] {
			if !visited[e.toNode] {
				visited[e.toNode] = true
				queue = append(queue, e.toNode)
			}
		}
	}
	return false
}

// Normalize 补默认值。
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
}

// --- v4: data pin type schema ---

// dataInPinTypeForKind returns the type of a data-in pin for a node kind,
// or "" if the kind has no such pin. Mirrors the frontend pinSpec.dataInFn.
// Expanded incrementally as Phase A-C migrate exec nodes to data-in pins.
func dataInPinTypeForKind(kind, pinName string) string {
	switch kind {
	case "SetVar":
		if pinName == "value" {
			return "any" // SetVar accepts any; runtime coerces per declared var type
		}
	case "IncVar":
		if pinName == "delta" {
			return "number"
		}
	}
	// Phase C migrations append cases here (Sleep, If, Switch, Loop, Parallel, Race, etc.)
	return ""
}

// dataOutPinTypeForKind returns the type of a data-out pin for a node kind.
// For dynamic-typed nodes (GetVar / Expr / GetSys / GetParam) returns "" — caller
// must resolve via node config (e.g. GetVar.varName → Container.Vars[name].Type).
// pinName parameter reserved for nodes with multiple data-out pins (Phase B+).
func dataOutPinTypeForKind(kind, pinName string) string {
	_ = pinName // reserved for future multi-pin nodes
	switch kind {
	case "GetVar":
		// Type is dynamic — resolved by caller via Container.Vars lookup.
		return ""
	}
	// Phase B adds pure-func data-out types here.
	return ""
}

