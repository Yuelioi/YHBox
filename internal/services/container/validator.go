// validator.go 容器/子图 graph 校验器.
//
// FUTURE-WORK (spec §11 架构演进): 当前所有校验都堆在 Validate 里 (含语义错误 + 兼容性
// normalize + dispatch 编译). 增长到 30+ 规则后会变难维护. 拆成三阶段会更清晰:
//
//   Validate   → 只报错: dangling edge, cyclic dep, EMPTY_SUBGRAPH_OUTPUT 等结构性 bug
//   Normalize  → 修能修的: subgraph_store.normalizeSubgraph 这种 self-heal 单独成阶段
//   Compile    → 把高层图编译成 runtime 友好结构 (edge index, pin map, frame layout)
//
// 现状这三件事混在 Validate + ContainerRunner 构造时. 拆分时机: 规则数 ≥ 25 或
// runtime 启动时间 (Validate + edgeIndex) 超过用户感知阈值.
package container

import (
	"fmt"
	"regexp"
	"strings"
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
	CodeInvalidROI            = "INVALID_ROI"
	CodeInvalidHSVRange       = "INVALID_HSV_RANGE"
	CodeInvalidScanAxis       = "INVALID_SCAN_AXIS"
	CodeInvalidClusterRange   = "INVALID_CLUSTER_RANGE"
	CodeInvalidVK             = "INVALID_VK"
	CodeInvalidMouseButton    = "INVALID_MOUSE_BUTTON"
	CodeUnsafeScreenshotPath  = "UNSAFE_SCREENSHOT_PATH"
	CodePollTooFast           = "POLL_TOO_FAST"
	CodeStopwatchEmptyKey     = "STOPWATCH_EMPTY_KEY"
	CodeStopwatchKeyMismatch  = "STOPWATCH_KEY_MISMATCH"
	CodeThrowInMainGraph      = "THROW_IN_MAIN_GRAPH"
	CodeInvalidSwitchCases    = "INVALID_SWITCH_CASES"
)

// v4 data-pin + variable + literal validation codes (spec §11).
const (
	CodePinTypeMismatch        = "PIN_TYPE_MISMATCH"
	CodePinTypeCoercionWarning = "PIN_TYPE_COERCION_WARNING"
	CodeGetVarUnknownVar       = "GETVAR_UNKNOWN_VAR"
	CodeGetVarTypeMismatch     = "GETVAR_TYPE_MISMATCH"
	CodeLiteralTypeMismatch    = "LITERAL_TYPE_MISMATCH"
	CodeDataPinDangling        = "DATA_PIN_DANGLING"

	// v4 Expr node (spec §5.4)
	CodeExprParseError     = "EXPR_PARSE_ERROR"
	CodeExprUnknownInput   = "EXPR_UNKNOWN_INPUT"
	CodeExprTypeMismatch   = "EXPR_TYPE_MISMATCH"
	CodeExprDuplicateInput = "EXPR_DUPLICATE_INPUT"
)

// ValidationError is the i18n-ready error envelope (spec §12).
// Message is deprecated (v3 carried Chinese; v4 frontend uses t(Code, Params) instead).
type ValidationError struct {
	Severity  string         `json:"severity"`
	Code      string         `json:"code"`
	GraphPath []string       `json:"graphPath"`
	NodeID    string         `json:"nodeId,omitempty"`
	Message   string         `json:"message,omitempty"` // deprecated: kept for v3 callers; v4 frontend ignores
	Params    map[string]any `json:"params,omitempty"`  // v4: template params for t(Code, Params)
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
func ValidateContainerWithContext(c *Container, vctx ValidateContext) []ValidationError {
	if c == nil {
		return nil
	}
	var errs []ValidationError
	errs = append(errs, validateMainGraph(c)...)
	errs = append(errs, validateWindowTarget(c)...)
	errs = append(errs, validateMouseCalibration(c, vctx)...)
	errs = append(errs, validateInvalidPins(c)...)
	errs = append(errs, validateMissingSubgraph(c)...)
	errs = append(errs, validateMissingTemplate(c, vctx)...)
	errs = append(errs, validatePlayClip(c)...)
	errs = append(errs, validatePhaseCNodeKinds(c)...)
	errs = append(errs, validateDataPinTypes(c)...)
	errs = append(errs, validateExprNodes(c)...)
	for i := range c.Subgraphs {
		errs = append(errs, validateSubgraph(c, &c.Subgraphs[i])...)
	}
	errs = append(errs, validateCyclicSubgraphs(c)...)
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
				})
			}
		}
		if to != "" {
			if _, ok := nodeIDs[to]; !ok {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeDanglingEdge,
					GraphPath: []string{"main"},
					Message:   fmt.Sprintf("边 %q → %q：目标节点 %q 不存在", e.From, e.To, to),
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
		kindByID := map[string]string{}
		nodeByID := map[string]*GraphNode{}
		for i := range nodes {
			n := &nodes[i]
			kindByID[n.ID] = n.Kind
			nodeByID[n.ID] = n
		}
		for _, e := range edges {
			fromID, fromPin := splitRef(e.From)
			toID, toPin := splitRef(e.To)
			if node, ok := nodeByID[fromID]; ok && fromPin != "" {
				validOut := nodeHasExecOutPin(node, fromPin)
				// Subgraph 调用节点: 出 pin 是子图 OutputPins decl ID
				if !validOut && node.Kind == "Subgraph" {
					if sgID, _ := node.Config["subgraphId"].(string); sgID != "" {
						if set, ok := subgraphOutputIDsByID[sgID]; ok {
							if _, has := set[fromPin]; has {
								validOut = true
							}
						}
					}
				}
				if !validOut {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeInvalidPin,
						GraphPath: graphPath, NodeID: fromID,
						Message: fmt.Sprintf("节点 %s (kind=%s) 不存在 out pin %q", fromID, node.Kind, fromPin),
					})
				}
			}
			// In-pin 仍走 kind-based pinExists — 当前所有节点 in pin 都静态.
			// 未来动态 in pin 节点出现再加 nodeHasExecInPin.
			if kind, ok := kindByID[toID]; ok && toPin != "" {
				if !pinExists(kind, toPin, false) {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeInvalidPin,
						GraphPath: graphPath, NodeID: toID,
						Message: fmt.Sprintf("节点 %s (kind=%s) 不存在 in pin %q", toID, kind, toPin),
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
			id, _ := n.Config["subgraphId"].(string)
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
				if calledID, _ := n.Config["subgraphId"].(string); calledID != "" {
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
	matchAny, _ := n.Config["match"].(map[string]any)
	getStr := func(k string) string {
		v, _ := matchAny[k].(string)
		return v
	}
	return windowTargetMatchSpec{
		Title:       getStr("title"),
		Class:       getStr("class"),
		ProcessName: getStr("processName"),
		TitleMatch:  getStr("titleMatch"),
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

	if poll, _ := n.Config["pollIntervalMs"].(float64); poll > 0 && poll < 30 {
		errs = append(errs, ValidationError{
			Severity: SeverityWarning,
			NodeID:   n.ID,
			Code:     CodePollTooFast,
			Message:  fmt.Sprintf("pollIntervalMs=%v < 30, will hammer CPU; runtime clamps <10", poll),
		})
	}

	return errs
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
		})
	}

	minC, _ := n.Config["minClusterPx"].(float64)
	maxC, _ := n.Config["maxClusterPx"].(float64)
	if maxC > 0 && minC > maxC {
		errs = append(errs, ValidationError{
			Severity: SeverityError,
			NodeID:   n.ID,
			Code:     CodeInvalidClusterRange,
			Message:  fmt.Sprintf("minClusterPx %v > maxClusterPx %v", minC, maxC),
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
func validateKeyHold(n *GraphNode) []ValidationError {
	vk, _ := n.Config["vk"].(string)
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
		}}
	}
	return nil
}

// validateSwitchConfig 校验 Switch 节点 config:
// - value 表达式必填
// - cases 数组必须非空
// - 每个 case: 非空 / 不含 '.' / 非 'default' / 无前后空格 / 不重复
// 错误 code 全用 INVALID_SWITCH_CASES (粗粒度, value 错也归这里方便用户找).
func validateSwitchConfig(n *GraphNode) []ValidationError {
	var errs []ValidationError
	cfg, _ := ParseSwitchConfig(n)

	if cfg.Value == "" {
		errs = append(errs, ValidationError{
			NodeID: n.ID, Code: CodeInvalidSwitchCases,
			Message: "Switch value 表达式必填",
		})
	}
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
			})
			continue
		}
		if strings.Contains(cs, ".") {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Message: fmt.Sprintf("Switch cases[%d]=%q 含 '.' (pin 分隔符, 禁用)", i, cs),
			})
			continue
		}
		if cs == "default" {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Message: fmt.Sprintf("Switch cases[%d]='default' 跟保留 default pin 冲突", i),
			})
			continue
		}
		if trimmed := strings.TrimSpace(cs); trimmed != cs {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Message: fmt.Sprintf("Switch cases[%d]=%q 含前导/尾部空格", i, cs),
			})
			continue
		}
		if seen[cs] {
			errs = append(errs, ValidationError{
				NodeID: n.ID, Code: CodeInvalidSwitchCases,
				Message: fmt.Sprintf("Switch cases[%d]=%q 重复", i, cs),
			})
			continue
		}
		seen[cs] = true
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
