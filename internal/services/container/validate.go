package container

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"yhbox/internal/services/container/nodekind"
	_ "yhbox/internal/services/container/nodekind/specs" // 触发所有 specs init() 注册
)

// NOTE(v2 migration)：本文件里的 Validate / Normalize 实现保留过渡，
// 但 v2 上线后所有调用方应该改去用 validator.go 里的 ValidateContainer（返回结构化 []ValidationError）。
// 等 1.7 把 ValidateContainer 实现完，本文件的 Validate 改成薄包装 → 调 ValidateContainer 把第一个 error 转 error。
//
// v4 Phase A.3: KnownNodeKinds / yieldKinds / execInPins / execOutPins / dataOutPins
// 5 张手工表 + dataInPinTypeForKind 100 行 switch 全部删除, 改成 nodekind 注册表 thin wrapper.
// 新加节点 = specs/*.go 里 1 个 Register() 调用, 这里不用改.

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
// data-in 走 static-only 路径 — Expr 等动态 data-in 节点需 cfg, caller 用 dataInPinTypeForNode.
func pinExists(kind, pin string, out bool) bool {
	s, ok := nodekind.Get(kind)
	if !ok {
		return false
	}
	if out {
		// data-out
		if s.DataOutType(pin) != "" {
			return true
		}
		// static exec-out
		for _, p := range s.ExecOut {
			if p == pin {
				return true
			}
		}
		// dynamic branchN for Parallel / Race (Switch 走 nodeHasExecOutPin)
		if (kind == "Parallel" || kind == "Race") && strings.HasPrefix(pin, "branch") {
			return true
		}
		return false
	}
	// data-in (static only — 动态 Expr inputs 用 dataInPinTypeForNode)
	if s.DataInType(pin, nil) != "" {
		return true
	}
	// exec-in
	for _, p := range execInPinsOf(kind) {
		if p == pin {
			return true
		}
	}
	return false
}

// execInPinsOf 返回 kind 的 exec-in pin 集合.
// nil 表示 "没有 exec-in" (Start / OnEvent / SubgraphInput / MouseCalibration / pure-data 节点),
// 跟 v3 旧 execInPins map 里显式赋 nil 的语义一致.
func execInPinsOf(kind string) []string {
	s, ok := nodekind.Get(kind)
	if !ok {
		return []string{"in"} // unknown kind 防御性默认 — 走到这里说明上游 validation 漏了
	}
	return s.ExecIn // nil = 没 exec-in; ["in"] = 标准单 exec-in
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
// 动态节点 (Switch / Parallel / Race) 按 config 派生; 静态节点走 spec.ExecOut.
// 内部自动 dedupe — schema 自洽不依赖 validator 拦截重复.
func execOutPinsForNode(n *GraphNode) map[string]struct{} {
	pins := map[string]struct{}{}
	if n == nil {
		return pins
	}
	s, ok := nodekind.Get(n.Kind)
	if !ok {
		return pins
	}
	for _, p := range s.ExecOutPins(n.Config) {
		pins[p] = struct{}{}
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
	t := dataOutPinTypeForKind(kind, pin)
	if t == "" {
		return false, ""
	}
	return true, t
}

// dataInType 同上 in 方向 (v4: 走 dataInPinTypeForKind).
func dataInType(kind, pin string) (bool, string) {
	t := dataInPinTypeForKind(kind, pin)
	if t == "" {
		return false, ""
	}
	return true, t
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
		if kindIsYield(n.Kind) {
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

// --- Registry-backed helpers (replaces v3 5-table model) ---

// knownKind replaces former KnownNodeKinds[kind] lookup.
func knownKind(kind string) bool {
	return nodekind.IsKnown(kind)
}

// kindIsYield replaces former yieldKinds[kind].
func kindIsYield(kind string) bool {
	s, ok := nodekind.Get(kind)
	return ok && s.IsYield
}

// dataInPinTypeForKind replaces the former 100-line switch.
// Returns "" if (kind, pin) is not a recognized data-in pin.
// Static-only path — Expr's dynamic inputs need cfg, use dataInPinTypeForNode.
func dataInPinTypeForKind(kind, pinName string) string {
	s, ok := nodekind.Get(kind)
	if !ok {
		return ""
	}
	return string(s.DataInType(pinName, nil))
}

// dataInPinTypeForNode is the cfg-aware variant — used by validators that
// need to see Expr.inputs[] dynamic pins (and Subgraph inputParams in future).
func dataInPinTypeForNode(n *GraphNode, pinName string) string {
	if n == nil {
		return ""
	}
	s, ok := nodekind.Get(n.Kind)
	if !ok {
		return ""
	}
	return string(s.DataInType(pinName, n.Config))
}

// dataOutPinTypeForKind returns the type of a data-out pin for a (kind, pinName).
// 走 nodekind 注册表. 动态类型节点 (GetVar / Expr / GetSys / GetParam) spec 里登记 "any",
// 真实类型由 caller 按 config 解析 (e.g. GetVar.varName → Container.Vars[name].Type) —
// validateDataPinTypes 已做.
func dataOutPinTypeForKind(kind, pinName string) string {
	s, ok := nodekind.Get(kind)
	if !ok {
		return ""
	}
	return string(s.DataOutType(pinName))
}
