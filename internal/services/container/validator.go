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

import "fmt"

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

type ValidationError struct {
	Severity  string   `json:"severity"`
	Code      string   `json:"code"`
	GraphPath []string `json:"graphPath"`
	NodeID    string   `json:"nodeId,omitempty"`
	Message   string   `json:"message"`
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
	errs = append(errs, validateMouseCalibration(c, vctx)...)
	errs = append(errs, validateInvalidPins(c)...)
	errs = append(errs, validateMissingSubgraph(c)...)
	errs = append(errs, validateMissingTemplate(c, vctx)...)
	errs = append(errs, validatePlayClip(c)...)
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
			if kind, ok := kindByID[fromID]; ok && fromPin != "" {
				validOut := pinExists(kind, fromPin, true)
				// Subgraph 调用节点: 出 pin 是子图 OutputPins decl ID
				if !validOut && kind == "Subgraph" {
					if n := nodeByID[fromID]; n != nil {
						if sgID, _ := n.Config["subgraphId"].(string); sgID != "" {
							if set, ok := subgraphOutputIDsByID[sgID]; ok {
								if _, has := set[fromPin]; has {
									validOut = true
								}
							}
						}
					}
				}
				if !validOut {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeInvalidPin,
						GraphPath: graphPath, NodeID: fromID,
						Message: fmt.Sprintf("节点 %s (kind=%s) 不存在 out pin %q", fromID, kind, fromPin),
					})
				}
			}
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
