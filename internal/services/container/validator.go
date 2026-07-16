// validator.go 容器/子图 graph 校验器 (Validate 阶段入口).
//
// 三阶段协作:
//
//	Validate  — 这里, 纯检查不 mutate. 入口 ValidateContainer / (c).Validate().
//	             内部分 3 sub-phase (Structural/Reference/Type-Semantic), 见 ValidateContainer.
//	Normalize — validate.go::Container.Normalize, self-heal 默认 + 子图 normalizeSubgraph 一次跑完.
//	Compile   — runtime/compile.go::CompileContainer, 编译 main+全部 subgraphs 的 edge/node 索引.
//
// 调用顺序:
//
//	Save 流程: Normalize → Validate → 写盘.
//	Run 流程:  (Store.Get 已 normalize/validate) → CompileContainer → NewContainerRunner → Run.
package container

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/robfig/cron/v3"

	nodepkg "github.com/yottaapp/yotta/internal/node"
)

const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

const (
	CodeDanglingEdge               = "DANGLING_EDGE"
	CodeInvalidPin                 = "INVALID_PIN"
	CodeDuplicateOutputPin         = "DUPLICATE_OUTPUT_PIN"
	CodeMissingTemplate            = "MISSING_TEMPLATE"
	CodeMissingSubgraph            = "MISSING_SUBGRAPH"
	CodeMissingMouseCalibration    = "MISSING_MOUSE_CALIBRATION"
	CodeDuplicateMouseCalibration  = "DUPLICATE_MOUSE_CALIBRATION"
	CodeMouseCalibrationNotSet     = "MOUSE_CALIBRATION_NOT_SET"
	CodeEmptySubgraphOutput        = "EMPTY_SUBGRAPH_OUTPUT"
	CodeMouseCalibrationInSubgraph = "MOUSE_CALIBRATION_IN_SUBGRAPH"
	CodeMultipleStarts             = "MULTIPLE_STARTS"
	CodeNoStart                    = "NO_START"
	CodeCyclicSubgraphDependency   = "CYCLIC_SUBGRAPH_DEPENDENCY"
	CodeMissingRequiredPin         = "MISSING_REQUIRED_PIN"
	CodeUnknownLiteralPin          = "UNKNOWN_LITERAL_PIN"
)

const (
	CodeMissingWin32WindowTarget           = "MISSING_WIN32_WINDOW_TARGET"
	CodeInvalidWin32WindowTargetRegex      = "INVALID_WIN32_WINDOW_TARGET_REGEX"
	CodeInvalidWin32WindowTargetEmptyMatch = "INVALID_WIN32_WINDOW_TARGET_EMPTY_MATCH"
	CodeUnsupportedTargetCapability        = "UNSUPPORTED_TARGET_CAPABILITY"
	// CodeNoActiveWindow は runtime 用 (ErrNoActiveWindow); validator 不主动发,
	// 此常量供错码集合 / 前端 i18n 对齐.
	CodeNoActiveWindow = "NO_ACTIVE_WINDOW"
)

// node-kind config validation codes.
const (
	CodeInvalidVK            = "INVALID_VK"
	CodeInvalidMouseButton   = "INVALID_MOUSE_BUTTON"
	CodeUnsafeScreenshotPath = "UNSAFE_SCREENSHOT_PATH"
	CodeStopwatchEmptyKey    = "STOPWATCH_EMPTY_KEY"
	CodeStopwatchKeyMismatch = "STOPWATCH_KEY_MISMATCH"
	CodeThrowInMainGraph     = "THROW_IN_MAIN_GRAPH"
	CodeInvalidSwitchCases   = "INVALID_SWITCH_CASES"
	CodeInvalidCronExpr      = "INVALID_CRON_EXPR"
	CodeInvalidRegexPattern  = "INVALID_REGEX_PATTERN"
)

// Data-pin + variable + literal validation codes.
const (
	CodePinTypeMismatch        = "PIN_TYPE_MISMATCH"
	CodePinTypeCoercionWarning = "PIN_TYPE_COERCION_WARNING"
	CodeGetVarUnknownVar       = "GETVAR_UNKNOWN_VAR"
	CodeGetVarTypeMismatch     = "GETVAR_TYPE_MISMATCH"
	CodeLiteralTypeMismatch    = "LITERAL_TYPE_MISMATCH"
	CodeDataPinDangling        = "DATA_PIN_DANGLING"
	CodeDataGraphCycle         = "DATA_GRAPH_CYCLE"

	// Expr node
	CodeExprParseError      = "EXPR_PARSE_ERROR"
	CodeExprUnknownInput    = "EXPR_UNKNOWN_INPUT"
	CodeExprUnknownVar      = "EXPR_UNKNOWN_VAR"
	CodeExprTypeMismatch    = "EXPR_TYPE_MISMATCH"
	CodeExprDuplicateInput  = "EXPR_DUPLICATE_INPUT"
	CodeExprUnknownFunction = "EXPR_UNKNOWN_FUNCTION"
	CodeExprFnArity         = "EXPR_FN_ARITY"

	// Script node
	CodeScriptParseError     = "SCRIPT_PARSE_ERROR"
	CodeScriptDuplicateInput = "SCRIPT_DUPLICATE_INPUT"

	// GetParam
	CodeGetParamUnknownParam = "GETPARAM_UNKNOWN_PARAM"

	// CollapsedNode
	CodeCollapsedPinBroken                = "COLLAPSED_PIN_BROKEN"
	CodeCollapsedReferencedBySubgraphCall = "COLLAPSED_REFERENCED_BY_SUBGRAPH_CALL"

	// Container vars
	CodeInvalidVarRef = "INVALID_VAR_REF"
	// 引用的子图需要的容器级 var 未声明 (全局化后插入引用即检, 编辑器一键补全)
	CodeSubgraphVarUndeclared = "SUBGRAPH_VAR_UNDECLARED"

	// Disable Node
	CodeDisabledBranchNodeWarn  = "WARN_DISABLED_BRANCH_NODE"
	CodeInvalidDisabledTerminal = "INVALID_DISABLED_TERMINAL"

	// Sentinel scope (Break/Continue 必须在 Loop body 内).
	// 静态拦截, 避免 sentinel 漏到主 dispatch 顶层只看 generic error.
	// (Throw 不再受 scope 限制 — 由 region 的 Fail 出口截获或冒泡到顶层.)
	CodeBreakOutsideLoop    = "BREAK_OUTSIDE_LOOP"
	CodeContinueOutsideLoop = "CONTINUE_OUTSIDE_LOOP"

	// Template / dependency codes (GUID 存在性校验, 无格式校验)
)

// ValidationError is the i18n-ready error envelope.
// B5 ship 后纯 Code+Params 模型, 前端 t(`error.<Code>`, Params) 走 vue-i18n.
type ValidationError struct {
	Severity  string         `json:"severity"`
	Code      string         `json:"code"`
	GraphPath []string       `json:"graphPath"`
	NodeID    string         `json:"nodeId,omitempty"`
	Params    map[string]any `json:"params,omitempty"` // 占位符 keys 跟 i18n string 的 {name} 对齐
}

// ValidateContainer 全量校验。
//
// E5: 3-phase ORDERING (documentation-driven, no short-circuit). Validators
// are grouped + commented by dependency layer so readers can map errors back
// to their concern. We do NOT short-circuit between phases — partial test
// fixtures (no Win32WindowTarget, partial graph) would otherwise lose downstream
// coverage. The benefit is purely readability/maintenance, not perf:
//
//  1. Structural   — Start uniqueness, dangling edges, subgraph cycles,
//     MouseCalibration/Win32WindowTarget rules.
//  2. Reference    — pin existence, missing subgraphs/templates, GetParam
//     param validity, CollapsedNode integrity.
//  3. Type/Semantic— pin types, literal types, Expr parse, data-graph DAG.
//
// Future hard-mode (short-circuit on fatal) can be opted-in via a flag without
// touching this signature. Tests would need fuller fixtures first.
// ValidateContainer 全量校验。sgs = 该容器引用闭包解析出的子图集 (全局池来源,
// 容器不再拥有子图 — 2026-06-12 全局化); nil = 容器不引用任何子图。
func ValidateContainer(c *Container, sgs []Subgraph) []ValidationError {
	return ValidateContainerWithRegistry(c, sgs, nodepkg.DefaultRegistrySnapshot())
}

func ValidateContainerWithRegistry(c *Container, sgs []Subgraph, registry nodepkg.RegistryReader) []ValidationError {
	if c == nil {
		return nil
	}
	var errs []ValidationError

	// 结构检查
	errs = append(errs, validateMainGraph(c)...)
	errs = append(errs, validateWin32WindowTargetWithRegistry(registry, c, sgs)...)
	errs = append(errs, validateTargetCapabilities(registry, c, sgs)...)
	errs = append(errs, validateMouseCalibration(c, sgs)...)
	for i := range sgs {
		errs = append(errs, validateSubgraph(c, &sgs[i])...)
	}
	errs = append(errs, validateCyclicSubgraphs(sgs)...)

	// 引用检查
	errs = append(errs, validateInvalidPins(registry, c, sgs)...)
	errs = append(errs, validateMissingSubgraph(c, sgs)...)
	errs = append(errs, validateGetParamNodes(c, sgs)...)
	errs = append(errs, validateCollapsedReferences(c, sgs)...)
	errs = append(errs, validateVarRefs(c, sgs)...)
	errs = append(errs, validateCaptureRefsWithRegistry(registry, c, sgs)...)
	errs = append(errs, validateRequiredGlobalsDeclared(c, sgs)...)
	errs = append(errs, validateDisabledNodes(c, sgs)...)
	errs = append(errs, validateSentinelScopeWithRegistry(registry, c, sgs)...)

	// 类型 / 语义检查
	errs = append(errs, validatePerKindConfig(c, sgs)...)
	errs = append(errs, validateDataPinTypes(registry, c, sgs)...)
	errs = append(errs, validateLiteralTypes(registry, c, sgs)...)
	errs = append(errs, validateExprNodes(c, sgs)...)
	errs = append(errs, validateScriptNodes(c, sgs)...)
	errs = append(errs, validateDataGraphAcyclic(registry, c, sgs)...)
	errs = append(errs, validateRequiredPinsWithRegistry(registry, c, sgs)...)
	errs = append(errs, validateUnknownLiteralPinsWithRegistry(registry, c, sgs)...)

	return errs
}

func validateMainGraph(c *Container) []ValidationError {
	var errs []ValidationError
	startCount := 0
	nodeIDs := map[string]string{}
	for _, n := range c.Graph.Nodes {
		if n.Kind == "Start" {
			startCount++
		}
		nodeIDs[n.ID] = n.Kind
	}
	if len(c.Graph.Nodes) > 0 {
		if startCount == 0 {
			errs = append(errs, ValidationError{
				Severity: SeverityError, Code: CodeNoStart,
				GraphPath: []string{"main"},
			})
		} else if startCount > 1 {
			errs = append(errs, ValidationError{
				Severity: SeverityError, Code: CodeMultipleStarts,
				GraphPath: []string{"main"},
				Params:    map[string]any{"count": startCount},
			})
		}
	}
	for _, e := range c.Graph.Edges {
		from := edgeNodeID(e.From)
		to := edgeNodeID(e.To)
		if from != "" {
			if _, ok := nodeIDs[from]; !ok {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeDanglingEdge,
					GraphPath: []string{"main"},
					Params:    map[string]any{"from": e.From, "to": e.To, "missing": from, "side": "source"},
				})
			}
		}
		if to != "" {
			if _, ok := nodeIDs[to]; !ok {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeDanglingEdge,
					GraphPath: []string{"main"},
					Params:    map[string]any{"from": e.From, "to": e.To, "missing": to, "side": "target"},
				})
			}
		}
	}
	return errs
}

func validateMouseCalibration(c *Container, sgs []Subgraph) []ValidationError {
	var errs []ValidationError
	calCount := 0
	var calNode *GraphNode
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].Kind == "MouseCalibration" {
			calCount++
			calNode = &c.Graph.Nodes[i]
		}
	}
	if calCount > 1 {
		errs = append(errs, ValidationError{
			Severity: SeverityError, Code: CodeDuplicateMouseCalibration,
			GraphPath: []string{"main"},
			Params:    map[string]any{"count": calCount},
		})
	}

	for _, sg := range sgs {
		for _, n := range sg.Graph.Nodes {
			if n.Kind == "MouseCalibration" {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeMouseCalibrationInSubgraph,
					GraphPath: []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)},
					NodeID:    n.ID,
				})
			}
		}
	}

	if calNode != nil {
		counts, _ := PinInt(calNode, "Counts360")
		if counts == 0 {
			errs = append(errs, ValidationError{
				Severity: SeverityWarning, Code: CodeMouseCalibrationNotSet,
				GraphPath: []string{"main"},
				NodeID:    calNode.ID,
			})
		}
		// 注: 多 profile 后不再有"本机单一全局值", 节点值跟某个 profile 不一致是常态
		// (各游戏 counts 本就不同), 故删 MOUSE_CALIBRATION_FOREIGN 校验。
	}
	return errs
}

// validateInvalidPins 扫所有 edges，确认 from/to 引用的 pin 名在 kind 的 pin 集合里。
// Subgraph 调用节点的 exec-out pin 是动态的 = 绑定子图 OutputPins 的 decl ID, 不能走静态
// pinExists. 这里查 subgraphById[node.config.subgraphId].OutputPins.
func validateInvalidPins(registry nodepkg.RegistryReader, c *Container, sgs []Subgraph) []ValidationError {
	var errs []ValidationError

	// 解析闭包内子图 id → outputPin decl id 集合 (Subgraph 调用节点 out pin 动态 check 用)
	subgraphOutputIDsByID := map[string]map[string]struct{}{}
	for _, sg := range sgs {
		set := map[string]struct{}{}
		for _, p := range sg.OutputPins {
			set[p.ID] = struct{}{}
		}
		subgraphOutputIDsByID[sg.ID] = set
	}

	// nodeByID: 同图节点 id → *GraphNode (用于查 Subgraph.config.subgraphId)
	checkEdges := func(nodes []GraphNode, edges []GraphEdge, graphPath []string) {
		nodeByID := map[string]*GraphNode{}
		for i := range nodes {
			n := &nodes[i]
			nodeByID[n.ID] = n
		}
		for _, e := range edges {
			fromID, fromPin := splitRef(e.From)
			toID, toPin := splitRef(e.To)
			if node, ok := nodeByID[fromID]; ok && fromPin != "" {
				// 边类型从 (kind, fromPin) 派生: spec.DataOut → data; 否则查 exec-out
				// (含 Subgraph 动态 exec-out via OutputPins).
				isDataOut := IsDataOutPinNodeWithRegistry(registry, node, fromPin)
				isExecOut := nodeHasExecOutPinWithRegistry(registry, node, fromPin)
				// CollapsedNode 跟 Subgraph 一样 dynamic exec-out — pin 名 = 后备子图 OutputPins.id.
				if !isDataOut && !isExecOut && (node.Kind == "Subgraph" || node.Kind == "CollapsedNode") {
					if sgID := PinString(node, "SubgraphID"); sgID != "" {
						if set, ok := subgraphOutputIDsByID[sgID]; ok {
							if _, has := set[fromPin]; has {
								isExecOut = true
							}
						}
					}
				}
				validOut := isDataOut || isExecOut
				if !validOut {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeInvalidPin,
						GraphPath: graphPath, NodeID: fromID,
						Params: map[string]any{"nodeID": fromID, "kind": node.Kind, "pin": fromPin, "side": "out"},
					})
				}
			}
			// In-pin: 走 node-aware 路径 — pinExists 只看静态 schema, Expr.inputs[] 动态 pin
			// 会被误报 INVALID_PIN. 先用 dataInPinTypeForNode 探一下, 失败再 fallback static pinExists.
			if toNode, ok := nodeByID[toID]; ok && toPin != "" {
				valid := false
				if dataInPinTypeForNode(registry, toNode, toPin) != "" {
					valid = true
				} else if pinExists(registry, toNode.Kind, toPin, false) {
					valid = true
				}
				if !valid {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeInvalidPin,
						GraphPath: graphPath, NodeID: toID,
						Params: map[string]any{"nodeID": toID, "kind": toNode.Kind, "pin": toPin, "side": "in"},
					})
				}
			}
		}
	}

	checkEdges(c.Graph.Nodes, c.Graph.Edges, []string{"main"})
	for _, sg := range sgs {
		sgPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}
		checkEdges(sg.Graph.Nodes, sg.Graph.Edges, sgPath)
	}
	return errs
}

// validateMissingSubgraph 扫 Subgraph 调用节点，确认 config.subgraphId 能在解析闭包里找到
// (闭包从全局池解析而来 — 不在 sgs 里 = 池里不存在或引用悬空)。
func validateMissingSubgraph(c *Container, sgs []Subgraph) []ValidationError {
	var errs []ValidationError
	known := map[string]bool{}
	for _, sg := range sgs {
		known[sg.ID] = true
	}
	check := func(nodes []GraphNode, graphPath []string) {
		for _, n := range nodes {
			if n.Kind != "Subgraph" {
				continue
			}
			id := PinString(&n, "SubgraphID")
			if id == "" {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeMissingSubgraph,
					GraphPath: graphPath, NodeID: n.ID,
				})
				continue
			}
			if !known[id] {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeMissingSubgraph,
					GraphPath: graphPath, NodeID: n.ID,
					Params: map[string]any{"subgraphId": id},
				})
			}
		}
	}
	check(c.Graph.Nodes, []string{"main"})
	for _, sg := range sgs {
		sgPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}
		check(sg.Graph.Nodes, sgPath)
	}
	return errs
}

func validateSubgraph(_ *Container, sg *Subgraph) []ValidationError {
	var errs []ValidationError
	graphPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}

	// 用 OutputPins 数检 EMPTY. normalize 兜底非空,
	// 这里 defensive 检 — load 后 normalize 漏跑或 raw mutate sg 才可能 0.
	if len(sg.OutputPins) == 0 {
		errs = append(errs, ValidationError{
			Severity: SeverityError, Code: CodeEmptySubgraphOutput,
			GraphPath: graphPath,
		})
	}

	seen := map[string]bool{}
	for _, p := range sg.OutputPins {
		if seen[p.Name] {
			errs = append(errs, ValidationError{
				Severity: SeverityError, Code: CodeDuplicateOutputPin,
				GraphPath: graphPath,
				Params:    map[string]any{"name": p.Name},
			})
		}
		seen[p.Name] = true
	}
	return errs
}

// validateCyclicSubgraphs 子图互调静态防环 (闭包内 DFS 三色)。运行时另有 32 层深度兜底
// (脚本动态调用拦不住静态环检, 见 runtime/subgraph_call.go)。
func validateCyclicSubgraphs(sgs []Subgraph) []ValidationError {
	adj := map[string][]string{}
	for _, sg := range sgs {
		var calls []string
		for _, n := range sg.Graph.Nodes {
			if n.Kind == "Subgraph" {
				if calledID := PinString(&n, "SubgraphID"); calledID != "" {
					calls = append(calls, calledID)
				}
			}
		}
		adj[sg.ID] = calls
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var cyclic bool

	var dfs func(node string)
	dfs = func(node string) {
		if cyclic {
			return
		}
		color[node] = gray
		for _, next := range adj[node] {
			if color[next] == gray {
				cyclic = true
				return
			}
			if color[next] == white {
				dfs(next)
			}
		}
		color[node] = black
	}
	for id := range adj {
		if color[id] == white {
			dfs(id)
		}
		if cyclic {
			break
		}
	}
	if cyclic {
		return []ValidationError{{
			Severity:  SeverityError,
			Code:      CodeCyclicSubgraphDependency,
			GraphPath: []string{"main"},
		}}
	}
	return nil
}

func edgeNodeID(ref string) string {
	for i := 0; i < len(ref); i++ {
		if ref[i] == '.' {
			return ref[:i]
		}
	}
	return ""
}

// validateWin32WindowTarget: 收集主图 + 子图所有 Win32WindowTarget, 逐个校验 MatchSpec.
// 若容器需要 Win32 窗口但一个都没有, 报 MISSING_WIN32_WINDOW_TARGET.
// 若容器只有 target-aware 节点且没有任何 target selection, 也报同码: Windows 是默认自动修复目标。
// 空图 (len(Nodes)==0) 跳过 — 跟 Start 检查同模式, 刚创建的 container 不报噪音.
func validateWin32WindowTarget(c *Container, sgs []Subgraph) []ValidationError {
	return validateWin32WindowTargetWithRegistry(nodepkg.DefaultRegistrySnapshot(), c, sgs)
}

func validateWin32WindowTargetWithRegistry(registry nodepkg.RegistryReader, c *Container, sgs []Subgraph) []ValidationError {
	if len(c.Graph.Nodes) == 0 {
		return nil
	}
	var errs []ValidationError

	// 收集所有 Win32WindowTarget (主图 + 子图), 逐个校验 MatchSpec.
	type wt struct {
		node      *GraphNode
		graphPath []string
	}
	var all []wt
	hasAnyTarget := false
	for i := range c.Graph.Nodes {
		if isTargetSelectionKind(c.Graph.Nodes[i].Kind) {
			hasAnyTarget = true
		}
		if c.Graph.Nodes[i].Kind == "Win32WindowTarget" {
			all = append(all, wt{node: &c.Graph.Nodes[i], graphPath: []string{"main"}})
		}
	}
	for si := range sgs {
		sg := &sgs[si]
		for ni := range sg.Graph.Nodes {
			if isTargetSelectionKind(sg.Graph.Nodes[ni].Kind) {
				hasAnyTarget = true
			}
			if sg.Graph.Nodes[ni].Kind == "Win32WindowTarget" {
				all = append(all, wt{node: &sg.Graph.Nodes[ni], graphPath: []string{"subgraph:" + sg.ID}})
			}
		}
	}

	if len(all) == 0 {
		if hasUnwiredNeedsWindowNode(registry, c, sgs) || (!hasAnyTarget && hasUnwiredNeedsTargetNode(registry, c, sgs)) {
			errs = append(errs, ValidationError{
				Severity:  SeverityError,
				Code:      CodeMissingWin32WindowTarget,
				GraphPath: []string{"main"},
			})
		}
		return errs
	}

	for _, w := range all {
		spec := readWin32WindowTargetMatchSpec(w.node)
		if spec.TitleMatch == "regex" && spec.Title != "" {
			if _, err := regexp.Compile(spec.Title); err != nil {
				errs = append(errs, ValidationError{
					Severity:  SeverityError,
					Code:      CodeInvalidWin32WindowTargetRegex,
					GraphPath: w.graphPath,
					NodeID:    w.node.ID,
					Params:    map[string]any{"error": err.Error()},
				})
			}
		}
		if win32WindowTargetIsEmptyMatch(spec) {
			errs = append(errs, ValidationError{
				Severity:  SeverityError,
				Code:      CodeInvalidWin32WindowTargetEmptyMatch,
				GraphPath: w.graphPath,
				NodeID:    w.node.ID,
			})
		}
	}
	return errs
}

func isTargetSelectionKind(kind string) bool {
	switch kind {
	case "Win32WindowTarget", "AndroidTarget":
		return true
	default:
		return false
	}
}

// matchSpec local 副本 — 避免 validator 包 import pkg/winutil 引入循环依赖.
type win32WindowTargetMatchSpec struct {
	Title, Class, ProcessName, TitleMatch string
}

func readWin32WindowTargetMatchSpec(n *GraphNode) win32WindowTargetMatchSpec {
	if n.Config == nil {
		return win32WindowTargetMatchSpec{}
	}
	return win32WindowTargetMatchSpec{
		Title:       PinString(n, "Title"),
		Class:       PinString(n, "Class"),
		ProcessName: PinString(n, "ProcessName"),
		TitleMatch:  PinString(n, "TitleMatch"),
	}
}

func win32WindowTargetIsEmptyMatch(spec win32WindowTargetMatchSpec) bool {
	hasAny := spec.Title != "" || spec.Class != "" || spec.ProcessName != ""
	if !hasAny {
		return true
	}
	if spec.TitleMatch == "regex" && spec.Class == "" && spec.ProcessName == "" {
		t := strings.TrimSpace(spec.Title)
		if t == ".*" || t == ".+" || t == "^.*$" || t == "^.+$" {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// node-kind config validators
// ---------------------------------------------------------------------------

// validatePerKindConfig runs per-kind config checks over every graph
// (main + all subgraphs) and emits ValidationErrors.
// It also performs the cross-node Stopwatch key coherence check per graph.
func validatePerKindConfig(c *Container, sgs []Subgraph) []ValidationError {
	var errs []ValidationError

	// main graph — graphPath ["main"], isMain=true
	errs = append(errs, checkGraphPerKind(c.Graph.Nodes, []string{"main"}, true)...)

	// subgraphs — isMain=false
	for _, sg := range sgs {
		sgPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}
		errs = append(errs, checkGraphPerKind(sg.Graph.Nodes, sgPath, false)...)
	}
	return errs
}

// checkGraphPerKind runs per-node dispatch + cross-node Stopwatch key check
// for a single graph (main or subgraph).
func checkGraphPerKind(nodes []GraphNode, graphPath []string, isMain bool) []ValidationError {
	var errs []ValidationError

	// Collect StopwatchStart keys for cross-node coherence check.
	startKeys := map[string]struct{}{}

	for i := range nodes {
		n := &nodes[i]
		var nodeErrs []ValidationError

		switch n.Kind {
		case "AI":
			nodeErrs = validateAI(n)
		case "StopwatchStart":
			nodeErrs = validateStopwatch(n)
			if key := PinString(n, "Key"); key != "" {
				startKeys[key] = struct{}{}
			}
		case "StopwatchStop", "StopwatchRead":
			nodeErrs = validateStopwatch(n)
		case "Switch":
			nodeErrs = validateSwitchConfig(n)
		case "Cron":
			nodeErrs = validateCronConfig(n)
		case "RegexMatch", "RegexExtract":
			nodeErrs = validateRegexPattern(n)
		case "Throw":
			if isMain {
				nodeErrs = []ValidationError{{
					Severity:  SeverityWarning,
					NodeID:    n.ID,
					Code:      CodeThrowInMainGraph,
					GraphPath: graphPath,
				}}
			}
		}

		for j := range nodeErrs {
			nodeErrs[j].GraphPath = graphPath
		}
		errs = append(errs, nodeErrs...)
	}

	// Cross-node: StopwatchStop/Read keys must have a matching StopwatchStart in same graph.
	for i := range nodes {
		n := &nodes[i]
		if n.Kind != "StopwatchStop" && n.Kind != "StopwatchRead" {
			continue
		}
		key := PinString(n, "Key")
		if key == "" {
			continue // already reported by validateStopwatch
		}
		if _, ok := startKeys[key]; !ok {
			errs = append(errs, ValidationError{
				Severity:  SeverityWarning,
				Code:      CodeStopwatchKeyMismatch,
				GraphPath: graphPath,
				NodeID:    n.ID,
				Params:    map[string]any{"kind": n.Kind, "key": key},
			})
		}
	}

	return errs
}

// validateSwitchConfig 校验 Switch 节点 config:
// - cases 数组必须非空
// - 每个 case: 非空 / 不含 '.' / 非 'default' / 无前后空格 / 不重复
// 错误 code 全用 INVALID_SWITCH_CASES.
//
// v4: Switch.value 现在是 data-in pin (走 r.pullValue), 不再是 config 字符串 → 不在此校验
// (validator_pin_types 跟 validateDataPinTypes 负责 data 边类型 / 是否有 literal).
func validateSwitchConfig(n *GraphNode) []ValidationError {
	var errs []ValidationError
	cfg, _ := ParseSwitchConfig(n)

	if len(cfg.Cases) == 0 {
		errs = append(errs, ValidationError{
			NodeID: n.ID, Code: CodeInvalidSwitchCases,
		})
		return errs
	}
	seen := map[string]bool{}
	for i, cs := range cfg.Cases {
		if cs == "" {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Params: map[string]any{"index": i, "reason": "empty"},
			})
			continue
		}
		if strings.Contains(cs, ".") {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Params: map[string]any{"index": i, "case": cs, "reason": "contains_dot"},
			})
			continue
		}
		if cs == "default" {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Params: map[string]any{"index": i, "case": cs, "reason": "reserved_default"},
			})
			continue
		}
		if trimmed := strings.TrimSpace(cs); trimmed != cs {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Params: map[string]any{"index": i, "case": cs, "reason": "whitespace"},
			})
			continue
		}
		if seen[cs] {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Params: map[string]any{"index": i, "case": cs, "reason": "duplicate"},
			})
			continue
		}
		seen[cs] = true
	}
	return errs
}

// cronParser 6-field cron 解析器 (sec min hour dom month dow). 跟 runtime/cron.go 同款.
// `cron.ParseStandard` 是 5-field 不够 — 必须用 NewParser 显式开 Second.
var cronParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// validateCronConfig 静态校验 Cron 节点的 inline literal expr.
// 动态来源 (上游 data edge 推 expr) 解析失败由 runtime 报同款 err.
func validateCronConfig(n *GraphNode) []ValidationError {
	var errs []ValidationError
	s := PinString(n, "Expression")
	if s == "" {
		return nil // 空 = 用户准备连上游 / 还没填 (dangling pin validator 别处报)
	}
	if _, err := cronParser.Parse(s); err != nil {
		errs = append(errs, ValidationError{
			Severity: SeverityError,
			Code:     CodeInvalidCronExpr,
			NodeID:   n.ID,
			// E4 i18n 迁移前中文 fallback (跟现有 ~30 个 validator 同款). 切 i18n 后前端 t() 覆盖.
			Params: map[string]any{"expr": s, "parseErr": err.Error()},
		})
	}
	return errs
}

// validateRegexPattern 静态校验 RegexMatch/RegexExtract 的 inline literal Pattern.
// 空 = 用户准备连上游 / 还没填 → 跳过 (同 validateCronConfig 惯例);
// 动态来源 (上游 data edge) 编辑期不可知, 运行时节点自身 Log.Warn + 安全值兜.
func validateRegexPattern(n *GraphNode) []ValidationError {
	s := PinString(n, "Pattern")
	if s == "" {
		return nil
	}
	if _, err := regexp.Compile(s); err != nil {
		return []ValidationError{{
			Severity: SeverityError,
			Code:     CodeInvalidRegexPattern,
			NodeID:   n.ID,
			Params:   map[string]any{"pattern": s, "parseErr": err.Error()},
		}}
	}
	return nil
}

// validateStopwatch checks that key is non-empty.
func validateStopwatch(n *GraphNode) []ValidationError {
	key := PinString(n, "Key")
	if key == "" {
		return []ValidationError{{
			Severity: SeverityError,
			NodeID:   n.ID,
			Code:     CodeStopwatchEmptyKey,
		}}
	}
	return nil
}

// ---------------------------------------------------------------------------
