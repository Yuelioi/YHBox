// validator.go 容器/子图 graph 校验器.
//
// Backlog: 拆 Validate / Normalize / Compile 三阶段 — 见 backend-backlog.md B3.
package container

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/robfig/cron/v3"
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
	CodeMouseCalibrationForeign    = "MOUSE_CALIBRATION_FOREIGN"
	CodeEmptySubgraphOutput        = "EMPTY_SUBGRAPH_OUTPUT"
	CodeMouseCalibrationInSubgraph = "MOUSE_CALIBRATION_IN_SUBGRAPH"
	CodeMultipleStarts             = "MULTIPLE_STARTS"
	CodeNoStart                    = "NO_START"
	CodeCyclicSubgraphDependency   = "CYCLIC_SUBGRAPH_DEPENDENCY"
	CodePlayClipNoClipID           = "PLAYCLIP_NO_CLIP_ID"
)

const (
	CodeMissingWindowTarget           = "MISSING_WINDOW_TARGET"
	CodeDuplicateWindowTarget         = "DUPLICATE_WINDOW_TARGET"
	CodeWindowTargetInSubgraph        = "WINDOW_TARGET_IN_SUBGRAPH"
	CodeInvalidWindowTargetRegex      = "INVALID_WINDOW_TARGET_REGEX"
	CodeInvalidWindowTargetEmptyMatch = "INVALID_WINDOW_TARGET_EMPTY_MATCH"
)

// Phase C node-kind config validation codes.
const (
	CodeInvalidROI           = "INVALID_ROI"
	CodeInvalidHSVRange      = "INVALID_HSV_RANGE"
	CodeInvalidScanAxis      = "INVALID_SCAN_AXIS"
	CodeInvalidClusterRange  = "INVALID_CLUSTER_RANGE"
	CodeInvalidVK            = "INVALID_VK"
	CodeInvalidMouseButton   = "INVALID_MOUSE_BUTTON"
	CodeUnsafeScreenshotPath = "UNSAFE_SCREENSHOT_PATH"
	CodePollTooFast          = "POLL_TOO_FAST"
	CodeStopwatchEmptyKey    = "STOPWATCH_EMPTY_KEY"
	CodeStopwatchKeyMismatch = "STOPWATCH_KEY_MISMATCH"
	CodeThrowInMainGraph     = "THROW_IN_MAIN_GRAPH"
	CodeInvalidSwitchCases   = "INVALID_SWITCH_CASES"
	CodeInvalidCronExpr      = "INVALID_CRON_EXPR"

	// DualColorBarTrack rois 数组校验 (multi-resolution lookup).
	CodeInvalidDualBarROIs   = "INVALID_DUALBAR_ROIS"
	CodeDuplicateDualBarROI  = "DUPLICATE_DUALBAR_ROI"
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
	CodeExprParseError     = "EXPR_PARSE_ERROR"
	CodeExprUnknownInput   = "EXPR_UNKNOWN_INPUT"
	CodeExprTypeMismatch   = "EXPR_TYPE_MISMATCH"
	CodeExprDuplicateInput = "EXPR_DUPLICATE_INPUT"

	// GetSys / GetParam
	CodeGetSysUnknownPath    = "GETSYS_UNKNOWN_PATH"
	CodeGetParamUnknownParam = "GETPARAM_UNKNOWN_PARAM"

	// CollapsedNode
	CodeCollapsedPinBroken                = "COLLAPSED_PIN_BROKEN"
	CodeCollapsedReferencedBySubgraphCall = "COLLAPSED_REFERENCED_BY_SUBGRAPH_CALL"

	// Container vars
	CodeInvalidVarRef = "INVALID_VAR_REF"

	// Disable Node
	CodeDisabledBranchNodeWarn  = "WARN_DISABLED_BRANCH_NODE"
	CodeInvalidDisabledTerminal = "INVALID_DISABLED_TERMINAL"

	// Sentinel scope (Break/Continue 必须在 Loop body 内; Throw 必须在 Try body 子图内).
	// P1.2: 静态拦截, 避免 sentinel 漏到主 dispatch 顶层只看 generic error.
	CodeBreakOutsideLoop    = "BREAK_OUTSIDE_LOOP"
	CodeContinueOutsideLoop = "CONTINUE_OUTSIDE_LOOP"
	CodeThrowOutsideTry     = "THROW_OUTSIDE_TRY"

	// Template key / dependency codes
	CodeInvalidTemplateKey = "INVALID_TEMPLATE_KEY"
	CodeTemplateNotFound   = "TEMPLATE_NOT_FOUND"
	CodeClipNotFound       = "CLIP_NOT_FOUND"
)

// ValidationError is the i18n-ready error envelope.
//
// Backlog: 删 Message field, 前端走纯 Code+Params 路径 — 见 backend-backlog.md B5.
// 当前 frontend fallback 链 (t(code, params) → backend.message → raw code) 兼容并存. Until then, Message
// remains authoritative for single-line display (ValidationFailure.Error()).
type ValidationError struct {
	Severity  string         `json:"severity"`
	Code      string         `json:"code"`
	GraphPath []string       `json:"graphPath"`
	NodeID    string         `json:"nodeId,omitempty"`
	Message   string         `json:"message,omitempty"` // v4: still authoritative until E4 i18n migration
	Params    map[string]any `json:"params,omitempty"`  // v4: template params for future t(Code, Params)
}

// ValidateContext 给 ValidateContainerWithContext 用，传入文件系统 / 设置态等
// "纯 graph 结构本身得不到" 的信息。各字段空 zero 值意味着"该项检查跳过"。
type ValidateContext struct {
	// AvailableTemplateKeys 容器 templates/ 目录下所有可用 key（含库下载夹带过来的）。
	// nil = 跳过 MISSING_TEMPLATE 检查（向后兼容老调用方）。
	AvailableTemplateKeys map[string]struct{}
	// SettingsMouseCounts360 用户本机 settings.ui.mouseCounts360。0 = 跳过 FOREIGN 警告。
	SettingsMouseCounts360 int
}

// ValidateContainer 无 context 短版：只跑结构级校验（dangling edge / cyclic / pin / etc.）
// MISSING_TEMPLATE 和 MOUSE_CALIBRATION_FOREIGN 需要外部信息，调 ValidateContainerWithContext。
func ValidateContainer(c *Container) []ValidationError {
	return ValidateContainerWithContext(c, ValidateContext{})
}

// ValidateContainerWithContext 全量校验。
// 业务层（service.go ValidateContainerByID）应该用这个版本传入完整上下文。
//
// E5: 3-phase ORDERING (documentation-driven, no short-circuit). Validators
// are grouped + commented by dependency layer so readers can map errors back
// to their concern. We do NOT short-circuit between phases — partial test
// fixtures (no WindowTarget, partial graph) would otherwise lose downstream
// coverage. The benefit is purely readability/maintenance, not perf:
//
//  1. Structural   — Start uniqueness, dangling edges, subgraph cycles,
//                    MouseCalibration/WindowTarget rules.
//  2. Reference    — pin existence, missing subgraphs/templates, GetSys/Param
//                    path validity, CollapsedNode integrity.
//  3. Type/Semantic— pin types, literal types, Expr parse, data-graph DAG.
//
// Future hard-mode (short-circuit on fatal) can be opted-in via a flag without
// touching this signature. Tests would need fuller fixtures first.
func ValidateContainerWithContext(c *Container, vctx ValidateContext) []ValidationError {
	if c == nil {
		return nil
	}
	var errs []ValidationError

	// Phase 1: Structural
	errs = append(errs, validateMainGraph(c)...)
	errs = append(errs, validateWindowTarget(c)...)
	errs = append(errs, validateMouseCalibration(c, vctx)...)
	for i := range c.Subgraphs {
		errs = append(errs, validateSubgraph(c, &c.Subgraphs[i])...)
	}
	errs = append(errs, validateCyclicSubgraphs(c)...)

	// Phase 2: Reference
	errs = append(errs, validateInvalidPins(c)...)
	errs = append(errs, validateMissingSubgraph(c)...)
	errs = append(errs, validateMissingTemplate(c, vctx)...)
	errs = append(errs, validatePlayClip(c)...)
	errs = append(errs, validateDualColorBarTrack(c)...)
	errs = append(errs, validateTemplateKeyNodes(c)...)
	errs = append(errs, validateGetSysNodes(c)...)
	errs = append(errs, validateGetParamNodes(c)...)
	errs = append(errs, validateCollapsedReferences(c)...)
	errs = append(errs, validateVarRefs(c)...)
	errs = append(errs, validateDisabledNodes(c)...)
	errs = append(errs, validateSentinelScope(c)...)

	// Phase 3: Type / Semantic
	errs = append(errs, validatePhaseCNodeKinds(c)...)
	errs = append(errs, validateDataPinTypes(c)...)
	errs = append(errs, validateLiteralTypes(c)...)
	errs = append(errs, validateExprNodes(c)...)
	errs = append(errs, validateDataGraphAcyclic(c)...)

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
				Message:   "主图没有 Start 节点",
			})
		} else if startCount > 1 {
			errs = append(errs, ValidationError{
				Severity: SeverityError, Code: CodeMultipleStarts,
				GraphPath: []string{"main"},
				Message:   fmt.Sprintf("主图有 %d 个 Start 节点（应恰好 1 个）", startCount),
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
					Message:   fmt.Sprintf("边 %q → %q：源节点 %q 不存在", e.From, e.To, from),
					Params:    map[string]any{"from": e.From, "to": e.To, "missing": from, "side": "source"},
				})
			}
		}
		if to != "" {
			if _, ok := nodeIDs[to]; !ok {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeDanglingEdge,
					GraphPath: []string{"main"},
					Message:   fmt.Sprintf("边 %q → %q：目标节点 %q 不存在", e.From, e.To, to),
					Params:    map[string]any{"from": e.From, "to": e.To, "missing": to, "side": "target"},
				})
			}
		}
	}
	return errs
}

func validateMouseCalibration(c *Container, vctx ValidateContext) []ValidationError {
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
			Message:   fmt.Sprintf("主图有 %d 个 MouseCalibration 节点（应最多 1 个）", calCount),
			Params:    map[string]any{"count": calCount},
		})
	}

	for _, sg := range c.Subgraphs {
		for _, n := range sg.Graph.Nodes {
			if n.Kind == "MouseCalibration" {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeMouseCalibrationInSubgraph,
					GraphPath: []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)},
					NodeID:    n.ID,
					Message:   "MouseCalibration 只能放在主图（容器范围唯一）",
				})
			}
		}
	}

	hasMouseRel := containsMouseMoveRel(c)
	if hasMouseRel && calCount == 0 {
		errs = append(errs, ValidationError{
			Severity: SeverityError, Code: CodeMissingMouseCalibration,
			GraphPath: []string{"main"},
			Message:   "容器使用了相对鼠标移动，主图必须有 1 个 MouseCalibration 节点",
		})
	}

	if calNode != nil {
		counts, _ := intFromConfig(calNode.Config, "counts360")
		if counts == 0 {
			errs = append(errs, ValidationError{
				Severity: SeverityWarning, Code: CodeMouseCalibrationNotSet,
				GraphPath: []string{"main"},
				NodeID:    calNode.ID,
				Message:   "MouseCalibration 节点未校准（counts360 = 0），运行时鼠标转向不准",
			})
		}
		// MOUSE_CALIBRATION_FOREIGN: 节点值 != 本机 settings 当前值 且 settings > 0
		// → 容器疑似从别人机器来。Settings 0 / 节点 0 都不触发（避免噪音）。
		if counts > 0 && vctx.SettingsMouseCounts360 > 0 && counts != vctx.SettingsMouseCounts360 {
			errs = append(errs, ValidationError{
				Severity: SeverityWarning, Code: CodeMouseCalibrationForeign,
				GraphPath: []string{"main"},
				NodeID:    calNode.ID,
				Message: fmt.Sprintf(
					"MouseCalibration 节点值 %d 跟你本机 (%d) 不一致，疑似从别人机器下载，建议用本机值覆盖此节点",
					counts, vctx.SettingsMouseCounts360,
				),
				Params: map[string]any{"nodeValue": counts, "settingsValue": vctx.SettingsMouseCounts360},
			})
		}
	}
	return errs
}

// validateInvalidPins 扫所有 edges，确认 from/to 引用的 pin 名在 kind 的 pin 集合里。
// Subgraph 调用节点的 exec-out pin 是动态的 = 绑定子图 OutputPins 的 decl ID, 不能走静态
// pinExists. 这里查 subgraphById[node.config.subgraphId].OutputPins.
func validateInvalidPins(c *Container) []ValidationError {
	var errs []ValidationError

	// 容器全部子图 id → outputPin decl id 集合 (Subgraph 调用节点 out pin 动态 check 用)
	subgraphOutputIDsByID := map[string]map[string]struct{}{}
	for _, sg := range c.Subgraphs {
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
				isDataOut := IsDataOutPin(node.Kind, fromPin)
				isExecOut := nodeHasExecOutPin(node, fromPin)
				// CollapsedNode 跟 Subgraph 一样 dynamic exec-out — pin 名 = 后备子图 OutputPins.id.
				if !isDataOut && !isExecOut && (node.Kind == "Subgraph" || node.Kind == "CollapsedNode") {
					if sgID := cfgString(node.Config, "SubgraphID"); sgID != "" {
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
						Message: fmt.Sprintf("节点 %s (kind=%s) 不存在 out pin %q", fromID, node.Kind, fromPin),
						Params:  map[string]any{"nodeID": fromID, "kind": node.Kind, "pin": fromPin, "side": "out"},
					})
				}
			}
			// In-pin: 走 node-aware 路径 — pinExists 只看静态 schema, Expr.inputs[] 动态 pin
			// 会被误报 INVALID_PIN. 先用 dataInPinTypeForNode 探一下, 失败再 fallback static pinExists.
			if toNode, ok := nodeByID[toID]; ok && toPin != "" {
				valid := false
				if dataInPinTypeForNode(toNode, toPin) != "" {
					valid = true
				} else if pinExists(toNode.Kind, toPin, false) {
					valid = true
				}
				if !valid {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeInvalidPin,
						GraphPath: graphPath, NodeID: toID,
						Message: fmt.Sprintf("节点 %s (kind=%s) 不存在 in pin %q", toID, toNode.Kind, toPin),
						Params:  map[string]any{"nodeID": toID, "kind": toNode.Kind, "pin": toPin, "side": "in"},
					})
				}
			}
		}
	}

	checkEdges(c.Graph.Nodes, c.Graph.Edges, []string{"main"})
	for _, sg := range c.Subgraphs {
		sgPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}
		checkEdges(sg.Graph.Nodes, sg.Graph.Edges, sgPath)
	}
	return errs
}

// validateMissingSubgraph 扫 Subgraph 调用节点，确认 config.subgraphId 在 c.Subgraphs 列表里。
func validateMissingSubgraph(c *Container) []ValidationError {
	var errs []ValidationError
	known := map[string]bool{}
	for _, sg := range c.Subgraphs {
		known[sg.ID] = true
	}
	check := func(nodes []GraphNode, graphPath []string) {
		for _, n := range nodes {
			if n.Kind != "Subgraph" {
				continue
			}
			id := cfgString(n.Config, "SubgraphID")
			if id == "" {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeMissingSubgraph,
					GraphPath: graphPath, NodeID: n.ID,
					Message: "Subgraph 调用节点未设 subgraphId",
				})
				continue
			}
			if !known[id] {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeMissingSubgraph,
					GraphPath: graphPath, NodeID: n.ID,
					Message: fmt.Sprintf("Subgraph 调用节点引用了不存在的子图 %q", id),
					Params:  map[string]any{"subgraphId": id},
				})
			}
		}
	}
	check(c.Graph.Nodes, []string{"main"})
	for _, sg := range c.Subgraphs {
		sgPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}
		check(sg.Graph.Nodes, sgPath)
	}
	return errs
}

// validateMissingTemplate 扫 template-引用节点的 config.template，确认 key 存在于容器的
// templates/ 目录。AvailableTemplateKeys = nil → 跳过此项（向后兼容老调用方 / 单测）。
func validateMissingTemplate(c *Container, vctx ValidateContext) []ValidationError {
	if vctx.AvailableTemplateKeys == nil {
		return nil
	}
	var errs []ValidationError
	check := func(nodes []GraphNode, graphPath []string) {
		for _, n := range nodes {
			switch n.Kind {
			case "WaitTemplate", "CheckTemplate", "ClickTemplate", "OnEvent":
			default:
				continue
			}
			key, _ := n.Config["template"].(string)
			if key == "" {
				continue // 未配 template 由其它规则报（节点功能性 validation 不在 v1）
			}
			if _, ok := vctx.AvailableTemplateKeys[key]; !ok {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeMissingTemplate,
					GraphPath: graphPath, NodeID: n.ID,
					Message: fmt.Sprintf("节点 %s 引用的模板 %q 在容器 templates/ 里找不到", n.ID, key),
					Params:  map[string]any{"nodeID": n.ID, "template": key},
				})
			}
		}
	}
	check(c.Graph.Nodes, []string{"main"})
	for _, sg := range c.Subgraphs {
		sgPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}
		check(sg.Graph.Nodes, sgPath)
	}
	return errs
}

// validatePlayClip 校验 PlayClip 节点必须设 clipID (空 = 没绑录制片段, 运行时报错).
// Phase 4: 不校验 clipID 在 clip store 是否真实存在 (clip 可能在另台机器 / 还没同步过来),
// 仅静态校验 config.clipID != "". 库管理后续阶段可加 PLAYCLIP_MISSING_CLIP rule.
func validatePlayClip(c *Container) []ValidationError {
	var errs []ValidationError
	check := func(nodes []GraphNode, graphPath []string) {
		for _, n := range nodes {
			if n.Kind != "PlayClip" {
				continue
			}
			id, _ := n.Config["clipID"].(string)
			if id == "" {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodePlayClipNoClipID,
					GraphPath: graphPath, NodeID: n.ID,
					Message: "PlayClip 节点未设 clipID (没绑定录制片段)",
				})
			}
		}
	}
	check(c.Graph.Nodes, []string{"main"})
	for _, sg := range c.Subgraphs {
		sgPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}
		check(sg.Graph.Nodes, sgPath)
	}
	return errs
}

func containsMouseMoveRel(c *Container) bool {
	for _, n := range c.Graph.Nodes {
		if n.Kind == "MouseMoveRel" {
			return true
		}
	}
	for _, sg := range c.Subgraphs {
		for _, n := range sg.Graph.Nodes {
			if n.Kind == "MouseMoveRel" {
				return true
			}
		}
	}
	return false
}

func validateSubgraph(_ *Container, sg *Subgraph) []ValidationError {
	var errs []ValidationError
	graphPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}

	outCount := 0
	for _, n := range sg.Graph.Nodes {
		if n.Kind == "SubgraphOutput" {
			outCount++
		}
	}
	if outCount == 0 {
		errs = append(errs, ValidationError{
			Severity: SeverityError, Code: CodeEmptySubgraphOutput,
			GraphPath: graphPath,
			Message:   "子图至少要有一个 SubgraphOutput 节点",
		})
	}

	seen := map[string]bool{}
	for _, p := range sg.OutputPins {
		if seen[p.Name] {
			errs = append(errs, ValidationError{
				Severity: SeverityError, Code: CodeDuplicateOutputPin,
				GraphPath: graphPath,
				Message:   fmt.Sprintf("OutputPin Name %q 重复", p.Name),
				Params:    map[string]any{"name": p.Name},
			})
		}
		seen[p.Name] = true
	}
	return errs
}

func validateCyclicSubgraphs(c *Container) []ValidationError {
	adj := map[string][]string{}
	for _, sg := range c.Subgraphs {
		var calls []string
		for _, n := range sg.Graph.Nodes {
			if n.Kind == "Subgraph" {
				if calledID := cfgString(n.Config, "SubgraphID"); calledID != "" {
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
			Message:   "子图存在环形引用（A→A 自递归 或 A→B→A 间接环）",
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

// validateWindowTarget v3 Phase B: 主图必须 1 个 WindowTarget, 子图禁,
// match 不能全空 / 不能万能 regex / regex 编译必须过.
// 空图 (len(Nodes)==0) 跳过 — 跟 Start 检查同模式, 刚创建的 container 不报噪音.
func validateWindowTarget(c *Container) []ValidationError {
	if len(c.Graph.Nodes) == 0 {
		return nil
	}
	var errs []ValidationError
	mainCount := 0
	var mainNode *GraphNode
	for i := range c.Graph.Nodes {
		n := &c.Graph.Nodes[i]
		if n.Kind == "WindowTarget" {
			mainCount++
			mainNode = n
		}
	}
	if mainCount == 0 {
		errs = append(errs, ValidationError{
			Severity:  SeverityError,
			Code:      CodeMissingWindowTarget,
			GraphPath: []string{"main"},
			Message:   "主图缺 WindowTarget 节点 — v3 container 必须声明目标窗口",
		})
	} else if mainCount > 1 {
		errs = append(errs, ValidationError{
			Severity:  SeverityError,
			Code:      CodeDuplicateWindowTarget,
			GraphPath: []string{"main"},
			Message:   "主图有多个 WindowTarget 节点, 只能 1 个",
		})
	}

	// 子图不能有 WindowTarget
	for _, sg := range c.Subgraphs {
		for _, n := range sg.Graph.Nodes {
			if n.Kind == "WindowTarget" {
				errs = append(errs, ValidationError{
					Severity:  SeverityError,
					Code:      CodeWindowTargetInSubgraph,
					GraphPath: []string{"subgraph:" + sg.ID},
					NodeID:    n.ID,
					Message:   "WindowTarget 不能放在子图里 (子图要复用跨 container 跨窗口)",
				})
			}
		}
	}

	// 主图唯一 WindowTarget config 合法性
	if mainNode != nil {
		spec := readWindowTargetMatchSpec(mainNode)
		// regex 编译
		if spec.TitleMatch == "regex" && spec.Title != "" {
			if _, err := regexp.Compile(spec.Title); err != nil {
				errs = append(errs, ValidationError{
					Severity:  SeverityError,
					Code:      CodeInvalidWindowTargetRegex,
					GraphPath: []string{"main"},
					NodeID:    mainNode.ID,
					Message:   "WindowTarget title regex 编译失败: " + err.Error(),
					Params:    map[string]any{"error": err.Error()},
				})
			}
		}
		// empty match (含 .* / .+ 万能)
		if windowTargetIsEmptyMatch(spec) {
			errs = append(errs, ValidationError{
				Severity:  SeverityError,
				Code:      CodeInvalidWindowTargetEmptyMatch,
				GraphPath: []string{"main"},
				NodeID:    mainNode.ID,
				Message:   "WindowTarget 匹配条件全空或万能 (会匹配任意窗口, 极易闯祸). 至少填一个实质性 title/class/processName",
			})
		}
	}

	return errs
}

// matchSpec local 副本 — 避免 validator 包 import pkg/winutil 引入循环依赖.
type windowTargetMatchSpec struct {
	Title, Class, ProcessName, TitleMatch string
}

func readWindowTargetMatchSpec(n *GraphNode) windowTargetMatchSpec {
	if n.Config == nil {
		return windowTargetMatchSpec{}
	}
	getStr := func(k string) string {
		v, _ := n.Config[k].(string)
		return v
	}
	return windowTargetMatchSpec{
		Title:       getStr("Title"),
		Class:       getStr("Class"),
		ProcessName: getStr("ProcessName"),
		TitleMatch:  getStr("TitleMatch"),
	}
}

func windowTargetIsEmptyMatch(spec windowTargetMatchSpec) bool {
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
// Phase C node-kind config validators
// ---------------------------------------------------------------------------

// validatePhaseCNodeKinds runs per-kind config checks over every graph
// (main + all subgraphs) and emits Phase C ValidationErrors.
// It also performs the cross-node Stopwatch key coherence check per graph.
func validatePhaseCNodeKinds(c *Container) []ValidationError {
	var errs []ValidationError

	// main graph — graphPath ["main"], isMain=true
	errs = append(errs, checkPhaseCGraph(c.Graph.Nodes, []string{"main"}, true)...)

	// subgraphs — isMain=false
	for _, sg := range c.Subgraphs {
		sgPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}
		errs = append(errs, checkPhaseCGraph(sg.Graph.Nodes, sgPath, false)...)
	}
	return errs
}

// checkPhaseCGraph runs per-node dispatch + cross-node Stopwatch key check
// for a single graph (main or subgraph).
func checkPhaseCGraph(nodes []GraphNode, graphPath []string, isMain bool) []ValidationError {
	var errs []ValidationError

	// Collect StopwatchStart keys for cross-node coherence check.
	startKeys := map[string]struct{}{}

	for i := range nodes {
		n := &nodes[i]
		var nodeErrs []ValidationError

		switch n.Kind {
		case "DetectColorHSV":
			nodeErrs = validateDetectColorHSV(n)
		case "ROIColorScan":
			nodeErrs = validateROIColorScan(n)
		case "Screenshot":
			nodeErrs = validateScreenshot(n)
		case "KeyHoldStart", "KeyHoldStop":
			nodeErrs = validateKeyHold(n)
		case "MouseHoldStart", "MouseHoldStop":
			nodeErrs = validateMouseHold(n)
		case "StopwatchStart":
			nodeErrs = validateStopwatch(n)
			if key, _ := n.Config["key"].(string); key != "" {
				startKeys[key] = struct{}{}
			}
		case "StopwatchStop", "StopwatchRead":
			nodeErrs = validateStopwatch(n)
		case "Switch":
			nodeErrs = validateSwitchConfig(n)
		case "Cron":
			nodeErrs = validateCronConfig(n)
		case "Throw":
			if isMain {
				nodeErrs = []ValidationError{{
					Severity:  SeverityWarning,
					NodeID:    n.ID,
					Code:      CodeThrowInMainGraph,
					GraphPath: graphPath,
					Message:   "Throw 节点在主图里会终止整个 runner；通常应放在子图的错误路径中",
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
		key, _ := n.Config["key"].(string)
		if key == "" {
			continue // already reported by validateStopwatch
		}
		if _, ok := startKeys[key]; !ok {
			errs = append(errs, ValidationError{
				Severity:  SeverityWarning,
				Code:      CodeStopwatchKeyMismatch,
				GraphPath: graphPath,
				NodeID:    n.ID,
				Message:   fmt.Sprintf("%s 使用 key %q 但同图中无对应的 StopwatchStart (运行时会读零值)", n.Kind, key),
				Params:    map[string]any{"kind": n.Kind, "key": key},
			})
		}
	}

	return errs
}

// validateDetectColorHSV checks ROI presence/size, HSV range ordering,
// and poll interval sanity.
func validateDetectColorHSV(n *GraphNode) []ValidationError {
	var errs []ValidationError

	roi, _ := n.Config["roi"].(map[string]any)
	if roi == nil {
		errs = append(errs, ValidationError{
			Severity: SeverityError,
			NodeID:   n.ID,
			Code:     CodeInvalidROI,
			Message:  "missing roi",
		})
	} else {
		w, _ := roi["w"].(float64)
		h, _ := roi["h"].(float64)
		if w < 1 || h < 1 {
			errs = append(errs, ValidationError{
				Severity: SeverityError,
				NodeID:   n.ID,
				Code:     CodeInvalidROI,
				Message:  fmt.Sprintf("roi w/h must be >=1, got %vx%v", w, h),
				Params:   map[string]any{"w": w, "h": h},
			})
		}
	}

	if hsv, _ := n.Config["hsv"].(map[string]any); hsv != nil {
		get := func(k string) float64 { v, _ := hsv[k].(float64); return v }
		if get("hMin") > get("hMax") || get("sMin") > get("sMax") || get("vMin") > get("vMax") {
			errs = append(errs, ValidationError{
				Severity: SeverityError,
				NodeID:   n.ID,
				Code:     CodeInvalidHSVRange,
				Message:  "HSV min > max",
			})
		}
	}

	// v4: numeric thresholds live at config.literal.<pin>, not config root.
	if poll := literalFloat(n, "pollIntervalMs"); poll > 0 && poll < 30 {
		errs = append(errs, ValidationError{
			Severity: SeverityWarning,
			NodeID:   n.ID,
			Code:     CodePollTooFast,
			Message:  fmt.Sprintf("pollIntervalMs=%v < 30, will hammer CPU; runtime clamps <10", poll),
			Params:   map[string]any{"actual": poll, "minMs": 30},
		})
	}

	return errs
}

// validateDualColorBarTrack 校验 DualColorBarTrack 节点 config.rois 必须非空 + 每项格式正确.
// 多分辨率支持: rois 每项含 resolution=[W,H] + ROI (x,y,w,h, 像素坐标 in window client area).
// 出错 code:
//   - INVALID_DUALBAR_ROIS (Error): rois 空 / 非数组 / 项非对象 / 缺 resolution / w/h <= 0 / 坐标越界
//   - DUPLICATE_DUALBAR_ROI (Warning): 同分辨率多条 entry (后者覆盖, 但通常是 import bug)
func validateDualColorBarTrack(c *Container) []ValidationError {
	var errs []ValidationError
	check := func(nodes []GraphNode, graphPath []string) {
		for _, n := range nodes {
			if n.Kind != "DualColorBarTrack" {
				continue
			}
			rois, ok := n.Config["Rois"].([]any)
			if !ok || len(rois) == 0 {
				errs = append(errs, ValidationError{
					Severity:  SeverityError,
					Code:      CodeInvalidDualBarROIs,
					GraphPath: graphPath,
					NodeID:    n.ID,
					Message:   fmt.Sprintf("DualColorBarTrack 节点 %s config.rois 必须是非空数组", n.ID),
					Params:    map[string]any{"nodeID": n.ID},
				})
				continue
			}
			seen := map[[2]int]bool{}
			for i, item := range rois {
				m, ok := item.(map[string]any)
				if !ok {
					errs = append(errs, ValidationError{
						Severity:  SeverityError,
						Code:      CodeInvalidDualBarROIs,
						GraphPath: graphPath,
						NodeID:    n.ID,
						Message:   fmt.Sprintf("DualColorBarTrack %s rois[%d] 不是对象", n.ID, i),
						Params:    map[string]any{"nodeID": n.ID, "index": i},
					})
					continue
				}
				res, ok := m["resolution"].([]any)
				if !ok || len(res) != 2 {
					errs = append(errs, ValidationError{
						Severity:  SeverityError,
						Code:      CodeInvalidDualBarROIs,
						GraphPath: graphPath,
						NodeID:    n.ID,
						Message:   fmt.Sprintf("DualColorBarTrack %s rois[%d].resolution 必须是 [W,H]", n.ID, i),
						Params:    map[string]any{"nodeID": n.ID, "index": i},
					})
					continue
				}
				rwf, _ := res[0].(float64)
				rhf, _ := res[1].(float64)
				resW := int(rwf)
				resH := int(rhf)
				if resW <= 0 || resH <= 0 {
					errs = append(errs, ValidationError{
						Severity:  SeverityError,
						Code:      CodeInvalidDualBarROIs,
						GraphPath: graphPath,
						NodeID:    n.ID,
						Message:   fmt.Sprintf("DualColorBarTrack %s rois[%d].resolution 必须正数 (得到 %dx%d)", n.ID, i, resW, resH),
						Params:    map[string]any{"nodeID": n.ID, "index": i, "w": resW, "h": resH},
					})
					continue
				}
				xf, _ := m["x"].(float64)
				yf, _ := m["y"].(float64)
				wf, _ := m["w"].(float64)
				hf, _ := m["h"].(float64)
				x := int(xf)
				y := int(yf)
				rw := int(wf)
				rh := int(hf)
				if rw < 1 || rh < 1 {
					errs = append(errs, ValidationError{
						Severity:  SeverityError,
						Code:      CodeInvalidDualBarROIs,
						GraphPath: graphPath,
						NodeID:    n.ID,
						Message:   fmt.Sprintf("DualColorBarTrack %s rois[%d] w/h 必须 >= 1 (得到 %dx%d)", n.ID, i, rw, rh),
						Params:    map[string]any{"nodeID": n.ID, "index": i, "w": rw, "h": rh},
					})
					continue
				}
				if x+rw > resW || y+rh > resH {
					errs = append(errs, ValidationError{
						Severity:  SeverityError,
						Code:      CodeInvalidDualBarROIs,
						GraphPath: graphPath,
						NodeID:    n.ID,
						Message:   fmt.Sprintf("DualColorBarTrack %s rois[%d] ROI (%d,%d)+(%dx%d) 超出 resolution %dx%d", n.ID, i, x, y, rw, rh, resW, resH),
						Params:    map[string]any{"nodeID": n.ID, "index": i},
					})
					continue
				}
				key := [2]int{resW, resH}
				if seen[key] {
					errs = append(errs, ValidationError{
						Severity:  SeverityWarning,
						Code:      CodeDuplicateDualBarROI,
						GraphPath: graphPath,
						NodeID:    n.ID,
						Message:   fmt.Sprintf("DualColorBarTrack %s rois[%d] resolution %dx%d 重复声明", n.ID, i, resW, resH),
						Params:    map[string]any{"nodeID": n.ID, "index": i, "w": resW, "h": resH},
					})
				}
				seen[key] = true
			}
		}
	}
	check(c.Graph.Nodes, []string{"main"})
	for _, sg := range c.Subgraphs {
		sgPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}
		check(sg.Graph.Nodes, sgPath)
	}
	return errs
}

// literalFloat reads n.Config["literal"][pinName] as float64 (v4 inline pin literal).
// Returns 0 if missing or wrong type. Validator-only helper — runtime uses r.pullNumber.
func literalFloat(n *GraphNode, pinName string) float64 {
	if n == nil || n.Config == nil {
		return 0
	}
	lit, _ := n.Config["literal"].(map[string]any)
	if lit == nil {
		return 0
	}
	switch v := lit[pinName].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

// validateROIColorScan extends validateDetectColorHSV with axis + cluster checks.
func validateROIColorScan(n *GraphNode) []ValidationError {
	errs := validateDetectColorHSV(n)

	axis, _ := n.Config["scanAxis"].(string)
	if axis != "x" && axis != "y" {
		errs = append(errs, ValidationError{
			Severity: SeverityError,
			NodeID:   n.ID,
			Code:     CodeInvalidScanAxis,
			Message:  fmt.Sprintf("scanAxis must be 'x' or 'y', got %q", axis),
			Params:   map[string]any{"got": axis},
		})
	}

	// v4: cluster bounds live at config.literal.<pin>, not config root.
	minC := literalFloat(n, "minClusterPx")
	maxC := literalFloat(n, "maxClusterPx")
	if maxC > 0 && minC > maxC {
		errs = append(errs, ValidationError{
			Severity: SeverityError,
			NodeID:   n.ID,
			Code:     CodeInvalidClusterRange,
			Message:  fmt.Sprintf("minClusterPx %v > maxClusterPx %v", minC, maxC),
			Params:   map[string]any{"min": minC, "max": maxC},
		})
	}

	return errs
}

// validateScreenshot checks that pathTemplate is relative and contains no "..".
func validateScreenshot(n *GraphNode) []ValidationError {
	tpl, _ := n.Config["pathTemplate"].(string)
	if tpl == "" {
		return nil // no template set: no path safety concern
	}
	unsafe := strings.Contains(tpl, "..") ||
		strings.HasPrefix(tpl, "/") ||
		strings.HasPrefix(tpl, "\\") ||
		(len(tpl) >= 3 && tpl[1] == ':') // e.g. C:\...
	if unsafe {
		return []ValidationError{{
			Severity: SeverityError,
			NodeID:   n.ID,
			Code:     CodeUnsafeScreenshotPath,
			Message:  "pathTemplate must be relative with no '..' traversal",
		}}
	}
	return nil
}

// validateKeyHold checks that vk is a non-empty string.
// Runtime calls pkginput.VK(name); we do not pre-validate the name here
// (Phase C precedent: let runtime fail loudly for unknown VK names).
//
func validateKeyHold(n *GraphNode) []ValidationError {
	vk := cfgString(n.Config, "VK")
	if vk == "" {
		return []ValidationError{{
			Severity: SeverityError,
			NodeID:   n.ID,
			Code:     CodeInvalidVK,
			Message:  "vk required (string, e.g. 'A' / 'Space' / 'F9')",
		}}
	}
	return nil
}

// validateMouseHold checks that button is one of left/right/middle.
func validateMouseHold(n *GraphNode) []ValidationError {
	btn, _ := n.Config["button"].(string)
	if btn != "left" && btn != "right" && btn != "middle" {
		return []ValidationError{{
			Severity: SeverityError,
			NodeID:   n.ID,
			Code:     CodeInvalidMouseButton,
			Message:  fmt.Sprintf("button %q not in left/right/middle", btn),
			Params:   map[string]any{"button": btn},
		}}
	}
	return nil
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
			Message: "Switch cases 数组必须非空",
		})
		return errs
	}
	seen := map[string]bool{}
	for i, cs := range cfg.Cases {
		if cs == "" {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Message: fmt.Sprintf("Switch cases[%d] 是空字符串", i),
				Params:  map[string]any{"index": i, "reason": "empty"},
			})
			continue
		}
		if strings.Contains(cs, ".") {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Message: fmt.Sprintf("Switch cases[%d]=%q 含 '.' (pin 分隔符, 禁用)", i, cs),
				Params:  map[string]any{"index": i, "case": cs, "reason": "contains_dot"},
			})
			continue
		}
		if cs == "default" {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Message: fmt.Sprintf("Switch cases[%d]='default' 跟保留 default pin 冲突", i),
				Params:  map[string]any{"index": i, "case": cs, "reason": "reserved_default"},
			})
			continue
		}
		if trimmed := strings.TrimSpace(cs); trimmed != cs {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Message: fmt.Sprintf("Switch cases[%d]=%q 含前导/尾部空格", i, cs),
				Params:  map[string]any{"index": i, "case": cs, "reason": "whitespace"},
			})
			continue
		}
		if seen[cs] {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Message: fmt.Sprintf("Switch cases[%d]=%q 重复", i, cs),
				Params:  map[string]any{"index": i, "case": cs, "reason": "duplicate"},
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
// 动态来源 (上游 data edge 推 expr) 解析失败由 runtime 报同款 err — 见
// debug/docs/superpowers/specs/2026-05-19-cron-node-design.md §3.1.
func validateCronConfig(n *GraphNode) []ValidationError {
	var errs []ValidationError
	lit, _ := n.Config["literal"].(map[string]any)
	s, _ := lit["Expression"].(string)
	if s == "" {
		return nil // 空 = 用户准备连上游 / 还没填 (dangling pin validator 别处报)
	}
	if _, err := cronParser.Parse(s); err != nil {
		errs = append(errs, ValidationError{
			Severity: SeverityError,
			Code:     CodeInvalidCronExpr,
			NodeID:   n.ID,
			// E4 i18n 迁移前中文 fallback (跟现有 ~30 个 validator 同款). 切 i18n 后前端 t() 覆盖.
			Message: fmt.Sprintf("Cron 节点表达式无效: %q (%v)", s, err),
			Params:  map[string]any{"expr": s, "parseErr": err.Error()},
		})
	}
	return errs
}

// validateStopwatch checks that key is non-empty.
func validateStopwatch(n *GraphNode) []ValidationError {
	key, _ := n.Config["key"].(string)
	if key == "" {
		return []ValidationError{{
			Severity: SeverityError,
			NodeID:   n.ID,
			Code:     CodeStopwatchEmptyKey,
			Message:  "key required",
		}}
	}
	return nil
}

// ---------------------------------------------------------------------------

func intFromConfig(cfg map[string]any, key string) (int, bool) {
	if cfg == nil {
		return 0, false
	}
	v, ok := cfg[key]
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		var n int
		_, err := fmt.Sscanf(x, "%d", &n)
		return n, err == nil
	}
	return 0, false
}
