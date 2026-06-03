package container

import (
	"testing"
	"time"
)

func minContainer() *Container {
	return &Container{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "c1",
		Name:          "test",
		Graph: Graph{
			ID:      "g-main",
			Version: GraphSchemaVersion,
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start", CreatedAt: time.Now().UTC()},
			},
		},
	}
}

func hasCode(errs []ValidationError, code string) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}

// B-8 regression: validator 加 4 条原本只声明 code 不产 check 的规则。

func TestValidator_InvalidPin_MainGraph(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "sleep1", Kind: "Sleep", CreatedAt: time.Now().UTC()},
	)
	// "Sleep" 节点只有 in / out pin；这里故意写 "weird-out"
	c.Graph.Edges = []GraphEdge{
		{From: "start.Done", To: "sleep1.In"},
		{From: "sleep1.weird-out", To: "start.In"},
	}
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeInvalidPin) {
		t.Errorf("expected INVALID_PIN, got %+v", errs)
	}
}

func TestValidator_MissingSubgraph_UnknownID(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "call1", Kind: "Subgraph",
			Config: map[string]any{"SubgraphID": "sg-does-not-exist"}, CreatedAt: time.Now().UTC()},
	)
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeMissingSubgraph) {
		t.Errorf("expected MISSING_SUBGRAPH, got %+v", errs)
	}
}

func TestValidator_MissingSubgraph_EmptyID(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "call1", Kind: "Subgraph", Config: map[string]any{}, CreatedAt: time.Now().UTC()},
	)
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeMissingSubgraph) {
		t.Errorf("expected MISSING_SUBGRAPH (empty subgraphId), got %+v", errs)
	}
}

func TestValidator_MissingTemplate_WithContext(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "wait1", Kind: "WaitTemplate",
			Config: map[string]any{"literal": map[string]any{"Templates": []any{"fish/onhook"}}}, CreatedAt: time.Now().UTC()},
	)
	// 不带 context → 跳过 MISSING_TEMPLATE 检查
	plain := ValidateContainer(c)
	if hasCode(plain, CodeMissingTemplate) {
		t.Errorf("nil AvailableTemplateKeys should skip MISSING_TEMPLATE check, got %+v", plain)
	}
	// 带空 keys 集合 → key 不存在，报 MISSING_TEMPLATE
	ctx := ValidateContext{AvailableTemplateKeys: map[string]struct{}{}}
	withCtx := ValidateContainerWithContext(c, ctx)
	if !hasCode(withCtx, CodeMissingTemplate) {
		t.Errorf("expected MISSING_TEMPLATE with empty key set, got %+v", withCtx)
	}
	// 带正确 key → 不报
	ctx2 := ValidateContext{AvailableTemplateKeys: map[string]struct{}{"fish/onhook": {}}}
	clean := ValidateContainerWithContext(c, ctx2)
	if hasCode(clean, CodeMissingTemplate) {
		t.Errorf("key present should not trigger MISSING_TEMPLATE, got %+v", clean)
	}
}

func TestValidator_NoStart(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = []GraphNode{
		{ID: "n1", Kind: "Sleep", CreatedAt: time.Now().UTC()},
	}
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeNoStart) {
		t.Errorf("expected NO_START, got %+v", errs)
	}
}

func TestValidator_MultipleStarts(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes, GraphNode{ID: "start2", Kind: "Start", CreatedAt: time.Now().UTC()})
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeMultipleStarts) {
		t.Errorf("expected MULTIPLE_STARTS, got %+v", errs)
	}
}

func TestValidator_DanglingEdge(t *testing.T) {
	c := minContainer()
	c.Graph.Edges = []GraphEdge{
		{From: "start.Done", To: "ghost.In"},
	}
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeDanglingEdge) {
		t.Errorf("expected DANGLING_EDGE, got %+v", errs)
	}
}

func TestValidator_MissingMouseCalibration(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "m", Kind: "MouseMoveRel", Config: map[string]any{"dx": "100", "dy": "0", "durationMs": "100"}, CreatedAt: time.Now().UTC()},
	)
	c.Graph.Edges = []GraphEdge{{From: "start.Done", To: "m.In"}}
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeMissingMouseCalibration) {
		t.Errorf("expected MISSING_MOUSE_CALIBRATION, got %+v", errs)
	}
}

func TestValidator_DuplicateMouseCalibration(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "cal1", Kind: "MouseCalibration", Config: map[string]any{"counts360": 4000}, CreatedAt: time.Now().UTC()},
		GraphNode{ID: "cal2", Kind: "MouseCalibration", Config: map[string]any{"counts360": 4000}, CreatedAt: time.Now().UTC()},
	)
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeDuplicateMouseCalibration) {
		t.Errorf("expected DUPLICATE_MOUSE_CALIBRATION, got %+v", errs)
	}
}

func TestValidator_MouseCalibrationNotSet(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "cal", Kind: "MouseCalibration", Config: map[string]any{"counts360": 0}, CreatedAt: time.Now().UTC()},
	)
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeMouseCalibrationNotSet) {
		t.Errorf("expected MOUSE_CALIBRATION_NOT_SET warning, got %+v", errs)
	}
	for _, e := range errs {
		if e.Code == CodeMouseCalibrationNotSet && e.Severity != SeverityWarning {
			t.Errorf("expected warning, got severity=%s", e.Severity)
		}
	}
}

func TestValidator_GraphPathPopulated(t *testing.T) {
	c := minContainer()
	c.Graph.Edges = []GraphEdge{{From: "ghost.Done", To: "start.In"}}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodeDanglingEdge {
			if len(e.GraphPath) == 0 {
				t.Errorf("expected GraphPath populated, got empty")
			}
			if e.GraphPath[0] != "main" {
				t.Errorf("expected GraphPath[0]=main, got %v", e.GraphPath)
			}
			return
		}
	}
	t.Errorf("没找到 DANGLING_EDGE error")
}

func TestValidator_CyclicSelfRecursive(t *testing.T) {
	c := minContainer()
	c.Subgraphs = []Subgraph{
		{
			ID:    "sg-A",
			Label: "A",
			Graph: Graph{
				ID:      "g-A",
				Version: GraphSchemaVersion,
				Nodes: []GraphNode{
					{ID: "in", Kind: "SubgraphInput", CreatedAt: time.Now().UTC()},
					{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "sg-A"}, CreatedAt: time.Now().UTC()},
				},
			},
			OutputPins: []SubgraphOutputDecl{{ID: "d1", Name: "done"}},
		},
	}
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeCyclicSubgraphDependency) {
		t.Errorf("expected CYCLIC_SUBGRAPH_DEPENDENCY for self-recursive, got %+v", errs)
	}
}

func TestValidator_CyclicIndirect(t *testing.T) {
	c := minContainer()
	c.Subgraphs = []Subgraph{
		{
			ID:    "sg-A",
			Label: "A",
			Graph: Graph{
				ID:      "gA",
				Version: GraphSchemaVersion,
				Nodes:   []GraphNode{{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "sg-B"}, CreatedAt: time.Now().UTC()}},
			},
			OutputPins: []SubgraphOutputDecl{{ID: "d", Name: "done"}},
		},
		{
			ID:    "sg-B",
			Label: "B",
			Graph: Graph{
				ID:      "gB",
				Version: GraphSchemaVersion,
				Nodes:   []GraphNode{{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "sg-A"}, CreatedAt: time.Now().UTC()}},
			},
			OutputPins: []SubgraphOutputDecl{{ID: "d", Name: "done"}},
		},
	}
	errs := ValidateContainer(c)
	if !hasCode(errs, CodeCyclicSubgraphDependency) {
		t.Errorf("expected CYCLIC_SUBGRAPH_DEPENDENCY for indirect cycle, got %+v", errs)
	}
}

func TestValidatePlayClip_MissingClipID(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "p", Kind: "PlayClip", Config: map[string]any{}},
				{ID: "p2", Kind: "PlayClip", Config: map[string]any{"ClipID": ""}},
				{ID: "p3", Kind: "PlayClip", Config: map[string]any{"ClipID": "abc"}},
			},
		},
	}
	errs := ValidateContainer(c)
	found := 0
	for _, e := range errs {
		if e.Code == CodePlayClipNoClipID {
			found++
		}
	}
	if found != 2 {
		t.Fatalf("期望 PLAYCLIP_NO_CLIP_ID 报 2 次, 实际 %d (errs: %+v)", found, errs)
	}
}

func TestValidator_HappyPath(t *testing.T) {
	c := minContainer()
	errs := ValidateContainer(c)
	if hasCode(errs, CodeNoStart) || hasCode(errs, CodeMultipleStarts) || hasCode(errs, CodeDanglingEdge) {
		t.Errorf("min container should be clean, got %+v", errs)
	}
}

func TestValidateWindowTarget_Missing(t *testing.T) {
	// 含窗口类节点 (ClickAt, NeedsWindow) 但无 WindowTarget → 触发 MISSING (validate-on-use).
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "s", Kind: "Start"},
		{ID: "c", Kind: "ClickAt"},
	}}}
	errs := validateWindowTarget(c)
	if !hasCode(errs, CodeMissingWindowTarget) {
		t.Errorf("want MISSING_WINDOW_TARGET, got %+v", errs)
	}
}

func TestValidateWindowTarget_WindowlessSkipped(t *testing.T) {
	// 纯窗口无关容器 (Start 无 NeedsWindow) 无 WindowTarget → 不报 MISSING (validate-on-use).
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "s", Kind: "Start"},
		{ID: "lg", Kind: "Log"},
		{ID: "sl", Kind: "Sleep"},
	}}}
	errs := validateWindowTarget(c)
	if hasCode(errs, CodeMissingWindowTarget) {
		t.Errorf("窗口无关容器不该报 MISSING_WINDOW_TARGET, got %+v", errs)
	}
}

func TestValidateWindowTarget_SubgraphWindowNodeRequires(t *testing.T) {
	// 主图窗口无关, 但子图含 ClickAt → 仍要求 WindowTarget (子图跟主图共用 hwnd).
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "s", Kind: "Start"},
			{ID: "sg", Kind: "Subgraph"},
		}},
		Subgraphs: []Subgraph{
			{ID: "sg1", Graph: Graph{Nodes: []GraphNode{
				{ID: "c", Kind: "ClickAt"},
			}}},
		},
	}
	errs := validateWindowTarget(c)
	if !hasCode(errs, CodeMissingWindowTarget) {
		t.Errorf("子图含窗口节点应要求 WindowTarget, got %+v", errs)
	}
}

func TestValidateWindowTarget_EmptyGraphSkipped(t *testing.T) {
	c := &Container{Graph: Graph{Nodes: []GraphNode{}}}
	errs := validateWindowTarget(c)
	if len(errs) != 0 {
		t.Errorf("empty graph should skip WindowTarget check, got %+v", errs)
	}
}

func TestValidate_MultipleWindowTargetsAllowed(t *testing.T) {
	// 主图两个合法 WindowTarget (各有 Title) → 不再报 DUPLICATE_WINDOW_TARGET
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "w1", Kind: "WindowTarget", Config: map[string]any{
			"Title": "异环",
		}},
		{ID: "w2", Kind: "WindowTarget", Config: map[string]any{
			"Title": "原神",
		}},
	}}}
	errs := validateWindowTarget(c)
	if hasCode(errs, "DUPLICATE_WINDOW_TARGET") {
		t.Errorf("多个 WindowTarget 不应再报 DUPLICATE_WINDOW_TARGET, got %+v", errs)
	}
	if len(errs) != 0 {
		t.Errorf("两个合法 WindowTarget 应无错误, got %+v", errs)
	}
}

func TestValidate_SubgraphWindowTargetAllowed(t *testing.T) {
	// 子图含 WindowTarget → 不再报 WINDOW_TARGET_IN_SUBGRAPH
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "w1", Kind: "WindowTarget", Config: map[string]any{
				"Title": "异环",
			}},
		}},
		Subgraphs: []Subgraph{
			{ID: "sg1", Graph: Graph{Nodes: []GraphNode{
				{ID: "w2", Kind: "WindowTarget", Config: map[string]any{
					"Title": "原神",
				}},
			}}},
		},
	}
	errs := validateWindowTarget(c)
	if hasCode(errs, "WINDOW_TARGET_IN_SUBGRAPH") {
		t.Errorf("子图 WindowTarget 不应再报 WINDOW_TARGET_IN_SUBGRAPH, got %+v", errs)
	}
	if len(errs) != 0 {
		t.Errorf("子图合法 WindowTarget 应无错误, got %+v", errs)
	}
}

func TestValidate_EachWindowTargetMatchValidated(t *testing.T) {
	// 两个 WindowTarget: 第一个合法, 第二个 Title/Class/ProcessName 全空 → 报 CodeInvalidWindowTargetEmptyMatch
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "w1", Kind: "WindowTarget", Config: map[string]any{
			"Title": "异环",
		}},
		{ID: "w2", Kind: "WindowTarget", Config: map[string]any{}},
	}}}
	errs := validateWindowTarget(c)
	if !hasCode(errs, CodeInvalidWindowTargetEmptyMatch) {
		t.Errorf("第二个空匹配 WindowTarget 应报 INVALID_WINDOW_TARGET_EMPTY_MATCH, got %+v", errs)
	}
	// 只有 w2 报错, w1 不报
	count := 0
	for _, e := range errs {
		if e.Code == CodeInvalidWindowTargetEmptyMatch {
			count++
			if e.NodeID != "w2" {
				t.Errorf("期望错误指向 w2, 实际 NodeID=%s", e.NodeID)
			}
		}
	}
	if count != 1 {
		t.Errorf("期望 EMPTY_MATCH 报 1 次, 实际 %d", count)
	}
}

func TestValidateWindowTarget_InvalidRegex(t *testing.T) {
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "w1", Kind: "WindowTarget", Config: map[string]any{
			"Title": "[invalid", "TitleMatch": "regex",
		}},
	}}}
	errs := validateWindowTarget(c)
	if !hasCode(errs, CodeInvalidWindowTargetRegex) {
		t.Errorf("want INVALID_REGEX, got %+v", errs)
	}
}

func TestValidateWindowTarget_EmptyMatch(t *testing.T) {
	cases := []struct {
		name   string
		config map[string]any
	}{
		{"all blank", map[string]any{}},
		{"regex dot star", map[string]any{"Title": ".*", "TitleMatch": "regex"}},
		{"regex dot plus", map[string]any{"Title": ".+", "TitleMatch": "regex"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Container{Graph: Graph{Nodes: []GraphNode{
				{ID: "w1", Kind: "WindowTarget", Config: tc.config},
			}}}
			errs := validateWindowTarget(c)
			if !hasCode(errs, CodeInvalidWindowTargetEmptyMatch) {
				t.Errorf("%s: want EMPTY_MATCH, got %+v", tc.name, errs)
			}
		})
	}
}

func TestValidateWindowTarget_Valid(t *testing.T) {
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "w1", Kind: "WindowTarget", Config: map[string]any{
			"Title": "异环", "Class": "Unreal",
		}},
	}}}
	errs := validateWindowTarget(c)
	if len(errs) != 0 {
		t.Errorf("valid WindowTarget should have no errors, got %+v", errs)
	}
}

// TestValidate_DataEdgeNoLongerRaisesInvalidPin: a data edge must not raise INVALID_PIN.
// Edge type is derived from (from-node.kind, from-pin), so a missing edge.Kind can't make
// the validator run an exec-pin check on a data-out pin (e.g. GetVar.value) — structurally impossible.
func TestValidate_DataEdgeNoLongerRaisesInvalidPin(t *testing.T) {
	c := &Container{
		SchemaVersion: CurrentSchemaVersion,
		Graph: Graph{
			ID: "g", Version: GraphSchemaVersion,
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "wt", Kind: "WindowTarget", Config: map[string]any{
					"Title": "异环",
				}},
				{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "x", "Scope": "local"}},
				{ID: "log", Kind: "Log", Config: map[string]any{"literal": map[string]any{"Message": "", "Level": "info"}}},
			},
			Edges: []GraphEdge{
				{From: "start.Done", To: "log.In"},     // exec edge
				{From: "gv.Value", To: "log.Message"}, // data edge — no Kind field, must still validate
			},
		},
		Vars: []VarDecl{{Name: "x", Type: "any"}},
	}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodeInvalidPin {
			t.Errorf("regression: data edge without Kind field raised INVALID_PIN: %+v", e)
		}
	}
}
