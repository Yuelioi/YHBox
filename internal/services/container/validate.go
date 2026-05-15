package container

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"yhbox/internal/services/expr"
)

// KnownNodeKinds 节点 kind 白名单。
var KnownNodeKinds = map[string]bool{
	"Start": true, "Sleep": true, "Loop": true, "If": true,
	"Parallel": true, "Race": true, "Stop": true, "Break": true, "Continue": true,
	"SetVar": true, "IncVar": true,
	"WaitTemplate": true, "CheckTemplate": true, "ClickTemplate": true,
	"DetectColor": true,
	"InvokeAction": true,
	"ClickAt": true, "KeyPress": true, "MouseMoveRel": true, "Scroll": true,
	"OnEvent": true,
	"Log": true, "Toast": true,
}

// yieldKinds Loop body 至少含一个，避免 forever loop CPU 100%。
// DetectColor 走截屏 + 像素扫，单次 ~3-10ms，算 yield。
var yieldKinds = map[string]bool{
	"Sleep": true, "WaitTemplate": true, "CheckTemplate": true,
	"ClickTemplate": true, "DetectColor": true, "InvokeAction": true, "OnEvent": true,
}

// execInPins kind → 该 kind 接受的 exec-in pin 名集合。
// 默认 = {"in"}；OnEvent / Start 没 exec-in。
var execInPins = map[string][]string{
	"Start":   nil,
	"OnEvent": nil,
	"Loop":    {"in", "loopback"}, // body 末尾接回 loopback 是单独 pin
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
	"InvokeAction":  {"out"},
	"ClickAt":       {"out"},
	"KeyPress":      {"out"},
	"MouseMoveRel":  {"out"},
	"Scroll":        {"out"},
	"OnEvent":       {"out"},
	"Log":           {"out"},
	"Toast":         {"out"},
}

// dataOutPins kind → data-out pin 名 → pin 类型（v1 只 "point"）。
var dataOutPins = map[string]map[string]string{
	"WaitTemplate":  {"point": "point"},
	"CheckTemplate": {"point": "point"},
	"ClickTemplate": {"point": "point"},
	"Loop":          {"iter": "number"},
	"Race":          {"winnerIdx": "number"},
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
//
// 规则：Start 唯一 / 边 pin 存在 / exec-in 入边唯一 / Break&Continue 必须在 Loop 后裔 /
// data edge 类型匹配 / Loop body 必须含 yield 节点 / Var 名合法+类型合法 / 表达式可 parse +
// 变量引用必须声明 / SetVar&IncVar 的 varName 必须声明。
func (c *Container) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("container name 不能为空")
	}
	if c.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("schemaVersion %d 不支持（当前 %d）", c.SchemaVersion, CurrentSchemaVersion)
	}

	// rule 7：var name 唯一 + regex + type 合法
	seenVar := map[string]bool{}
	for _, v := range c.Vars {
		if !varNameRE.MatchString(v.Name) {
			return fmt.Errorf("var name %q 不合法（必须匹配 [a-zA-Z_][a-zA-Z0-9_]*）", v.Name)
		}
		if seenVar[v.Name] {
			return fmt.Errorf("duplicate var name: %q", v.Name)
		}
		seenVar[v.Name] = true
		switch v.Type {
		case "number", "bool", "string", "point":
		default:
			return fmt.Errorf("var %q type %q 不支持（必须 number|bool|string|point）", v.Name, v.Type)
		}
	}

	// node id 唯一 + kind 在白名单
	seenNode := map[string]bool{}
	nodesByID := map[string]*GraphNode{}
	startCount := 0
	for i := range c.Graph.Nodes {
		n := &c.Graph.Nodes[i]
		if n.ID == "" {
			return errors.New("node id 不能为空")
		}
		if seenNode[n.ID] {
			return fmt.Errorf("duplicate node id: %q", n.ID)
		}
		seenNode[n.ID] = true
		if !KnownNodeKinds[n.Kind] {
			return fmt.Errorf("unknown node kind: %q（节点 %s）", n.Kind, n.ID)
		}
		nodesByID[n.ID] = n
		if n.Kind == "Start" {
			startCount++
		}
	}

	// rule 1：必须有且仅有 1 个 Start 节点（允许空图——首次创建容器为空）
	if len(c.Graph.Nodes) > 0 && startCount != 1 {
		return fmt.Errorf("Start 节点必须恰好 1 个，当前 %d 个", startCount)
	}

	// rule 2 + edge 格式校验
	for i, e := range c.Graph.Edges {
		if !edgeFormatRE.MatchString(e.From) {
			return fmt.Errorf("edge[%d] from %q 格式必须 <nodeId>.<pinName>", i, e.From)
		}
		if !edgeFormatRE.MatchString(e.To) {
			return fmt.Errorf("edge[%d] to %q 格式必须 <nodeId>.<pinName>", i, e.To)
		}
		fromNode, fromPin := splitRef(e.From)
		toNode, toPin := splitRef(e.To)
		if nodesByID[fromNode] == nil {
			return fmt.Errorf("edge[%d] from references unknown node %q", i, fromNode)
		}
		if nodesByID[toNode] == nil {
			return fmt.Errorf("edge[%d] to references unknown node %q", i, toNode)
		}
		// rule 2：pin 必须存在
		if !pinExists(nodesByID[fromNode].Kind, fromPin, true /* out */) {
			return fmt.Errorf("edge[%d] from %q: node kind %q 无输出 pin %q",
				i, e.From, nodesByID[fromNode].Kind, fromPin)
		}
		if !pinExists(nodesByID[toNode].Kind, toPin, false /* in */) {
			return fmt.Errorf("edge[%d] to %q: node kind %q 无输入 pin %q",
				i, e.To, nodesByID[toNode].Kind, toPin)
		}
	}

	// rule 3：同一 exec-in pin 最多 1 个入边（除 Loop.body 外允许扇入也不允许；
	// spec 说"除 Loop.body 之外"——但 Loop.body 是 out pin，从 Loop 出去；
	// 实际可能多入的是 loopback：from(各分支末尾).out → to(loop.loopback)）。
	// 这里实现：execInPin 入边唯一；data-in pin 不强制（多 setter 写同 var 是用户责任）。
	inEdgeCount := map[string]int{} // "<nodeId>.<pin>" → count
	for _, e := range c.Graph.Edges {
		toNode, toPin := splitRef(e.To)
		n := nodesByID[toNode]
		if n == nil {
			continue
		}
		if isExecInPin(n.Kind, toPin) && !(n.Kind == "Loop" && toPin == "loopback") {
			inEdgeCount[e.To]++
		}
	}
	for ref, count := range inEdgeCount {
		if count > 1 {
			return fmt.Errorf("exec-in pin %q 有 %d 条入边（最多 1 条）", ref, count)
		}
	}

	// rule 5：data edge 类型匹配（v1 仅 point）
	for i, e := range c.Graph.Edges {
		fromNode, fromPin := splitRef(e.From)
		toNode, toPin := splitRef(e.To)
		fromKind := nodesByID[fromNode].Kind
		toKind := nodesByID[toNode].Kind
		fromIsData, fromT := dataOutType(fromKind, fromPin)
		toIsData, toT := dataInType(toKind, toPin)
		if fromIsData != toIsData {
			return fmt.Errorf("edge[%d] data/exec 类型混接：%s.%s → %s.%s", i, fromNode, fromPin, toNode, toPin)
		}
		if fromIsData && fromT != toT {
			return fmt.Errorf("edge[%d] data type mismatch：%s.%s (%s) → %s.%s (%s)",
				i, fromNode, fromPin, fromT, toNode, toPin, toT)
		}
	}

	// rule 4：Break/Continue 节点必须在 Loop 后裔子图中
	for _, n := range c.Graph.Nodes {
		if n.Kind != "Break" && n.Kind != "Continue" {
			continue
		}
		if !isInsideLoop(n.ID, c.Graph, nodesByID) {
			return fmt.Errorf("%s 节点 %s 不在任何 Loop 后裔子图中", n.Kind, n.ID)
		}
	}

	// rule 6：Loop body 必须含至少一个 yield 节点
	for _, n := range c.Graph.Nodes {
		if n.Kind != "Loop" {
			continue
		}
		if !loopBodyHasYield(n.ID, c.Graph, nodesByID) {
			return fmt.Errorf("Loop %s 的 body 必须含 yield 节点（Sleep/WaitTemplate/CheckTemplate/ClickTemplate/InvokeAction/OnEvent）", n.ID)
		}
	}

	// rule 8a：表达式 parse + 8b：$vars.X 必须在 c.Vars 已声明
	declaredVars := map[string]bool{}
	for _, v := range c.Vars {
		declaredVars[v.Name] = true
	}
	for _, n := range c.Graph.Nodes {
		keys := exprConfigKeys[n.Kind]
		for _, k := range keys {
			raw, ok := n.Config[k]
			if !ok {
				continue
			}
			s, ok := raw.(string)
			if !ok || s == "" {
				continue
			}
			ast, err := expr.Parse(s)
			if err != nil {
				return fmt.Errorf("node %s.%s 表达式 parse 失败: %w", n.ID, k, err)
			}
			for _, p := range expr.VarRefs(ast) {
				if !strings.HasPrefix(p, "$vars.") {
					// $params / $sys 由 runtime env 校验，这里不挡。
					continue
				}
				name := strings.TrimPrefix(p, "$vars.")
				// 只取首段；$vars.point.x 用首段 "point" 比对声明。
				if dot := strings.Index(name, "."); dot >= 0 {
					name = name[:dot]
				}
				if name == "" || !declaredVars[name] {
					return fmt.Errorf("node %s.%s 引用未声明的变量 %q（请先在 Vars 里声明）", n.ID, k, p)
				}
			}
		}
	}

	// SetVar / IncVar 的 varName 字段必须指向已声明变量
	for _, n := range c.Graph.Nodes {
		if n.Kind != "SetVar" && n.Kind != "IncVar" {
			continue
		}
		raw, ok := n.Config["varName"]
		if !ok {
			return fmt.Errorf("%s 节点 %s 缺少 varName 字段", n.Kind, n.ID)
		}
		name, ok := raw.(string)
		if !ok || strings.TrimSpace(name) == "" {
			return fmt.Errorf("%s 节点 %s varName 必须是非空字符串", n.Kind, n.ID)
		}
		if !declaredVars[name] {
			return fmt.Errorf("%s 节点 %s 引用未声明的变量 %q", n.Kind, n.ID, name)
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
	// InvokeAction params.<X>
	if kind == "InvokeAction" && strings.HasPrefix(pin, "params.") {
		return true
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

// dataOutType 返该 pin 是否为 data-out + 类型。
func dataOutType(kind, pin string) (bool, string) {
	if m := dataOutPins[kind]; m != nil {
		if t, ok := m[pin]; ok {
			return true, t
		}
	}
	return false, ""
}

// dataInType 同上 in 方向。InvokeAction params.* 默认 "any"（不参与 type check）。
func dataInType(kind, pin string) (bool, string) {
	if m := dataInPins[kind]; m != nil {
		if t, ok := m[pin]; ok {
			return true, t
		}
	}
	if kind == "InvokeAction" && strings.HasPrefix(pin, "params.") {
		return true, "any"
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
}
