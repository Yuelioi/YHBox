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
	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeInvalidPin) {
		t.Errorf("expected INVALID_PIN, got %+v", errs)
	}
}

// 失败出口的 Code 数据字段接到下游 data-in (Switch.Value 按错误码分流) 是合法的 data 边 —
// 不应再报 INVALID_PIN / PIN_TYPE_MISMATCH. (回归: error-model 早期 Error/Code 只在前端暴露成
// data 引脚, validator 不认 → 保存即 INVALID_PIN「out pin Code 不存在」.)
func TestValidator_ExecOutputDataField_WiredAsData(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "rp", Kind: "RunProgram", CreatedAt: time.Now().UTC(),
			Config: map[string]any{"literal": map[string]any{"Target": "notepad.exe"}}},
		GraphNode{ID: "sw", Kind: "Switch", CreatedAt: time.Now().UTC(),
			Config: map[string]any{"cases": []any{"launch_failed"}}},
	)
	c.Graph.Edges = []GraphEdge{
		{From: "start.Done", To: "rp.In"},
		{From: "rp.Fail", To: "sw.In"},    // exec 边: 失败分支
		{From: "rp.Code", To: "sw.Value"}, // data 边: Code (Fail 出口的数据字段) → Switch 分流值
	}
	errs := ValidateContainer(c, nil)
	if hasCode(errs, CodeInvalidPin) {
		t.Errorf("RunProgram.Code → Switch.Value 应是合法 data 边, 不该报 INVALID_PIN: %+v", errs)
	}
	if hasCode(errs, CodePinTypeMismatch) {
		t.Errorf("Code(string) → Value(string) 类型相容, 不该报 PIN_TYPE_MISMATCH: %+v", errs)
	}
}

func TestValidator_MissingSubgraph_UnknownID(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "call1", Kind: "Subgraph",
			Config: map[string]any{"SubgraphID": "sg-does-not-exist"}, CreatedAt: time.Now().UTC()},
	)
	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeMissingSubgraph) {
		t.Errorf("expected MISSING_SUBGRAPH, got %+v", errs)
	}
}

func TestValidator_MissingSubgraph_EmptyID(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "call1", Kind: "Subgraph", Config: map[string]any{}, CreatedAt: time.Now().UTC()},
	)
	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeMissingSubgraph) {
		t.Errorf("expected MISSING_SUBGRAPH (empty subgraphId), got %+v", errs)
	}
}

func TestValidator_NoStart(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = []GraphNode{
		{ID: "n1", Kind: "Sleep", CreatedAt: time.Now().UTC()},
	}
	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeNoStart) {
		t.Errorf("expected NO_START, got %+v", errs)
	}
}

func TestValidator_MultipleStarts(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes, GraphNode{ID: "start2", Kind: "Start", CreatedAt: time.Now().UTC()})
	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeMultipleStarts) {
		t.Errorf("expected MULTIPLE_STARTS, got %+v", errs)
	}
}

func TestValidator_DanglingEdge(t *testing.T) {
	c := minContainer()
	c.Graph.Edges = []GraphEdge{
		{From: "start.Done", To: "ghost.In"},
	}
	errs := ValidateContainer(c, nil)
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
	errs := ValidateContainer(c, nil)
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
	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeDuplicateMouseCalibration) {
		t.Errorf("expected DUPLICATE_MOUSE_CALIBRATION, got %+v", errs)
	}
}

func TestValidator_MouseCalibrationNotSet(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "cal", Kind: "MouseCalibration", Config: map[string]any{"counts360": 0}, CreatedAt: time.Now().UTC()},
	)
	errs := ValidateContainer(c, nil)
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
	errs := ValidateContainer(c, nil)
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
	sgs := []Subgraph{
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
	errs := ValidateContainer(c, sgs)
	if !hasCode(errs, CodeCyclicSubgraphDependency) {
		t.Errorf("expected CYCLIC_SUBGRAPH_DEPENDENCY for self-recursive, got %+v", errs)
	}
}

func TestValidator_CyclicIndirect(t *testing.T) {
	c := minContainer()
	sgs := []Subgraph{
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
	errs := ValidateContainer(c, sgs)
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
	errs := ValidateContainer(c, nil)
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
	errs := ValidateContainer(c, nil)
	if hasCode(errs, CodeNoStart) || hasCode(errs, CodeMultipleStarts) || hasCode(errs, CodeDanglingEdge) {
		t.Errorf("min container should be clean, got %+v", errs)
	}
}

func TestValidateWin32WindowTarget_Missing(t *testing.T) {
	// 含窗口类节点 (ClickAt, NeedsWindow) 但无 Win32WindowTarget → 触发 MISSING (validate-on-use).
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "s", Kind: "Start"},
		{ID: "c", Kind: "ClickAt"},
	}}}
	errs := validateWin32WindowTarget(c, nil)
	if !hasCode(errs, CodeMissingWin32WindowTarget) {
		t.Errorf("want MISSING_WIN32_WINDOW_TARGET, got %+v", errs)
	}
}

func TestValidateWin32WindowTarget_WindowlessSkipped(t *testing.T) {
	// 纯窗口无关容器 (Start 无 NeedsWindow) 无 Win32WindowTarget → 不报 MISSING (validate-on-use).
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "s", Kind: "Start"},
		{ID: "lg", Kind: "Log"},
		{ID: "sl", Kind: "Sleep"},
	}}}
	errs := validateWin32WindowTarget(c, nil)
	if hasCode(errs, CodeMissingWin32WindowTarget) {
		t.Errorf("窗口无关容器不该报 MISSING_WIN32_WINDOW_TARGET, got %+v", errs)
	}
}

func TestValidateWin32WindowTarget_SubgraphWindowNodeRequires(t *testing.T) {
	// 主图窗口无关, 但子图含 ClickAt → 仍要求 Win32WindowTarget (子图跟主图共用 hwnd).
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "s", Kind: "Start"},
			{ID: "sg", Kind: "Subgraph"},
		}},
	}
	sgs := []Subgraph{
		{ID: "sg1", Graph: Graph{Nodes: []GraphNode{
			{ID: "c", Kind: "ClickAt"},
		}}},
	}
	errs := validateWin32WindowTarget(c, sgs)
	if !hasCode(errs, CodeMissingWin32WindowTarget) {
		t.Errorf("子图含窗口节点应要求 Win32WindowTarget, got %+v", errs)
	}
}

func TestValidateWin32WindowTarget_EmptyGraphSkipped(t *testing.T) {
	c := &Container{Graph: Graph{Nodes: []GraphNode{}}}
	errs := validateWin32WindowTarget(c, nil)
	if len(errs) != 0 {
		t.Errorf("empty graph should skip Win32WindowTarget check, got %+v", errs)
	}
}

func TestValidate_MultipleWin32WindowTargetsAllowed(t *testing.T) {
	// 主图两个合法 Win32WindowTarget (各有 Title) → 不再报 DUPLICATE_WIN32_WINDOW_TARGET
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "w1", Kind: "Win32WindowTarget", Config: map[string]any{
			"Title": "异环",
		}},
		{ID: "w2", Kind: "Win32WindowTarget", Config: map[string]any{
			"Title": "原神",
		}},
	}}}
	errs := validateWin32WindowTarget(c, nil)
	if hasCode(errs, "DUPLICATE_WIN32_WINDOW_TARGET") {
		t.Errorf("多个 Win32WindowTarget 不应再报 DUPLICATE_WIN32_WINDOW_TARGET, got %+v", errs)
	}
	if len(errs) != 0 {
		t.Errorf("两个合法 Win32WindowTarget 应无错误, got %+v", errs)
	}
}

func TestValidate_SubgraphWin32WindowTargetAllowed(t *testing.T) {
	// 子图含 Win32WindowTarget → 不再报 WIN32_WINDOW_TARGET_IN_SUBGRAPH
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "w1", Kind: "Win32WindowTarget", Config: map[string]any{
				"Title": "异环",
			}},
		}},
	}
	sgs := []Subgraph{
		{ID: "sg1", Graph: Graph{Nodes: []GraphNode{
			{ID: "w2", Kind: "Win32WindowTarget", Config: map[string]any{
				"Title": "原神",
			}},
		}}},
	}
	errs := validateWin32WindowTarget(c, sgs)
	if hasCode(errs, "WIN32_WINDOW_TARGET_IN_SUBGRAPH") {
		t.Errorf("子图 Win32WindowTarget 不应再报 WIN32_WINDOW_TARGET_IN_SUBGRAPH, got %+v", errs)
	}
	if len(errs) != 0 {
		t.Errorf("子图合法 Win32WindowTarget 应无错误, got %+v", errs)
	}
}

func TestValidate_EachWin32WindowTargetMatchValidated(t *testing.T) {
	// 两个 Win32WindowTarget: 第一个合法, 第二个 Title/Class/ProcessName 全空 → 报 CodeInvalidWin32WindowTargetEmptyMatch
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "w1", Kind: "Win32WindowTarget", Config: map[string]any{
			"Title": "异环",
		}},
		{ID: "w2", Kind: "Win32WindowTarget", Config: map[string]any{}},
	}}}
	errs := validateWin32WindowTarget(c, nil)
	if !hasCode(errs, CodeInvalidWin32WindowTargetEmptyMatch) {
		t.Errorf("第二个空匹配 Win32WindowTarget 应报 INVALID_WIN32_WINDOW_TARGET_EMPTY_MATCH, got %+v", errs)
	}
	// 只有 w2 报错, w1 不报
	count := 0
	for _, e := range errs {
		if e.Code == CodeInvalidWin32WindowTargetEmptyMatch {
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

func TestValidateWin32WindowTarget_InvalidRegex(t *testing.T) {
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "w1", Kind: "Win32WindowTarget", Config: map[string]any{
			"Title": "[invalid", "TitleMatch": "regex",
		}},
	}}}
	errs := validateWin32WindowTarget(c, nil)
	if !hasCode(errs, CodeInvalidWin32WindowTargetRegex) {
		t.Errorf("want INVALID_REGEX, got %+v", errs)
	}
}

func TestValidateWin32WindowTarget_EmptyMatch(t *testing.T) {
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
				{ID: "w1", Kind: "Win32WindowTarget", Config: tc.config},
			}}}
			errs := validateWin32WindowTarget(c, nil)
			if !hasCode(errs, CodeInvalidWin32WindowTargetEmptyMatch) {
				t.Errorf("%s: want EMPTY_MATCH, got %+v", tc.name, errs)
			}
		})
	}
}

func TestValidateWin32WindowTarget_Valid(t *testing.T) {
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "w1", Kind: "Win32WindowTarget", Config: map[string]any{
			"Title": "异环", "Class": "Unreal",
		}},
	}}}
	errs := validateWin32WindowTarget(c, nil)
	if len(errs) != 0 {
		t.Errorf("valid Win32WindowTarget should have no errors, got %+v", errs)
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
				{ID: "wt", Kind: "Win32WindowTarget", Config: map[string]any{
					"Title": "异环",
				}},
				{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "x", "Scope": "local"}},
				{ID: "log", Kind: "Log", Config: map[string]any{"literal": map[string]any{"Message": "", "Level": "info"}}},
			},
			Edges: []GraphEdge{
				{From: "start.Done", To: "log.In"},    // exec edge
				{From: "gv.Value", To: "log.Message"}, // data edge — no Kind field, must still validate
			},
		},
		Vars: []VarDecl{{Name: "x", Type: "any"}},
	}
	errs := ValidateContainer(c, nil)
	for _, e := range errs {
		if e.Code == CodeInvalidPin {
			t.Errorf("regression: data edge without Kind field raised INVALID_PIN: %+v", e)
		}
	}
}
