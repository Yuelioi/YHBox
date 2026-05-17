// transform_test.go: event 流 → Subgraph 转换的表驱动测试.
package recording

import (
	"testing"

	"yhbox/internal/services/inputclip"
)

const (
	tw = 1920
	th = 1080
)

func ev(typ inputclip.EventType, tUs uint64, a, b, c int32) inputclip.Event {
	return inputclip.Event{TUs: tUs, Type: typ, A: a, B: b, C: c}
}

func TestCompactToSteps_Empty(t *testing.T) {
	got := compactToSteps(nil, tw, th)
	if len(got) != 0 {
		t.Fatalf("空事件应返 0 step, got %d", len(got))
	}
}

func TestCompactToSteps_SingleKeyPress(t *testing.T) {
	events := []inputclip.Event{
		ev(inputclip.EventTypeKeyDown, 0, 'A', 0, 0),
		ev(inputclip.EventTypeKeyUp, 100_000, 'A', 0, 0), // 100ms
	}
	steps := compactToSteps(events, tw, th)
	if len(steps) != 1 {
		t.Fatalf("want 1 step, got %d", len(steps))
	}
	if steps[0].kind != "KeyPress" {
		t.Errorf("want KeyPress, got %s", steps[0].kind)
	}
	if vk, _ := steps[0].config["vk"].(string); vk != "A" {
		t.Errorf("want vk=A, got %v", steps[0].config["vk"])
	}
	if dur, _ := steps[0].config["durationMs"].(float64); dur != 100 {
		t.Errorf("want durationMs=100, got %v", steps[0].config["durationMs"])
	}
}

func TestCompactToSteps_OrphanUpIgnored(t *testing.T) {
	events := []inputclip.Event{
		ev(inputclip.EventTypeKeyUp, 0, 'A', 0, 0), // 无对应 Down
	}
	got := compactToSteps(events, tw, th)
	if len(got) != 0 {
		t.Fatalf("orphan up 应丢弃, got %d step", len(got))
	}
}

func TestCompactToSteps_AutoRepeatCollapsed(t *testing.T) {
	// Windows 长按: Down Down Down Up. 应只输出 1 个 KeyPress, 时长 = 首 Down → Up.
	events := []inputclip.Event{
		ev(inputclip.EventTypeKeyDown, 0, 'W', 0, 0),
		ev(inputclip.EventTypeKeyDown, 50_000, 'W', 0, 0),
		ev(inputclip.EventTypeKeyDown, 100_000, 'W', 0, 0),
		ev(inputclip.EventTypeKeyUp, 500_000, 'W', 0, 0),
	}
	steps := compactToSteps(events, tw, th)
	if len(steps) != 1 || steps[0].kind != "KeyPress" {
		t.Fatalf("auto-repeat 应折叠成 1 个 KeyPress, got %+v", steps)
	}
	if dur := steps[0].config["durationMs"].(float64); dur != 500 {
		t.Errorf("durationMs 应从首 Down 算, want 500, got %v", dur)
	}
}

func TestCompactToSteps_Click(t *testing.T) {
	events := []inputclip.Event{
		ev(inputclip.EventTypeMouseBtnDown, 0, int32(HookBtnLeft), 960, 540), // 中心
		ev(inputclip.EventTypeMouseBtnUp, 50_000, int32(HookBtnLeft), 970, 545),
	}
	steps := compactToSteps(events, tw, th)
	if len(steps) != 1 || steps[0].kind != "ClickAt" {
		t.Fatalf("want 1 ClickAt, got %+v", steps)
	}
	xr := steps[0].config["xRatio"].(float64)
	yr := steps[0].config["yRatio"].(float64)
	if xr < 0.499 || xr > 0.501 {
		t.Errorf("xRatio 应 ≈ 0.5 (用 Down 坐标), got %v", xr)
	}
	if yr < 0.499 || yr > 0.501 {
		t.Errorf("yRatio 应 ≈ 0.5 (用 Down 坐标), got %v", yr)
	}
	if btn := steps[0].config["button"].(string); btn != "left" {
		t.Errorf("button want left, got %s", btn)
	}
}

func TestCompactToSteps_ClickRightMiddle(t *testing.T) {
	events := []inputclip.Event{
		ev(inputclip.EventTypeMouseBtnDown, 0, int32(HookBtnRight), 100, 200),
		ev(inputclip.EventTypeMouseBtnUp, 10_000, int32(HookBtnRight), 100, 200),
		ev(inputclip.EventTypeMouseBtnDown, 20_000, int32(HookBtnMiddle), 300, 400),
		ev(inputclip.EventTypeMouseBtnUp, 30_000, int32(HookBtnMiddle), 300, 400),
	}
	steps := compactToSteps(events, tw, th)
	if len(steps) != 2 {
		t.Fatalf("want 2 click, got %d", len(steps))
	}
	if b := steps[0].config["button"].(string); b != "right" {
		t.Errorf("[0] want right, got %s", b)
	}
	if b := steps[1].config["button"].(string); b != "middle" {
		t.Errorf("[1] want middle, got %s", b)
	}
}

func TestCompactToSteps_Scroll(t *testing.T) {
	events := []inputclip.Event{
		ev(inputclip.EventTypeScroll, 0, 3, 960, 540),
	}
	steps := compactToSteps(events, tw, th)
	if len(steps) != 1 || steps[0].kind != "Scroll" {
		t.Fatalf("want 1 Scroll, got %+v", steps)
	}
	if d := steps[0].config["delta"].(float64); d != 3 {
		t.Errorf("delta want 3, got %v", d)
	}
}

func TestCompactToSteps_SleepInsertion(t *testing.T) {
	// 两个 KeyPress, 间隔 1s — 应插一个 Sleep ~1000ms.
	events := []inputclip.Event{
		ev(inputclip.EventTypeKeyDown, 0, 'A', 0, 0),
		ev(inputclip.EventTypeKeyUp, 50_000, 'A', 0, 0), // A 在 50ms 结束
		ev(inputclip.EventTypeKeyDown, 1_050_000, 'B', 0, 0), // 1000ms 后 B Down
		ev(inputclip.EventTypeKeyUp, 1_100_000, 'B', 0, 0),
	}
	steps := compactToSteps(events, tw, th)
	if len(steps) != 3 {
		t.Fatalf("want 3 steps (KeyPress, Sleep, KeyPress), got %d: %+v", len(steps), steps)
	}
	if steps[0].kind != "KeyPress" || steps[1].kind != "Sleep" || steps[2].kind != "KeyPress" {
		t.Errorf("顺序错, got [%s, %s, %s]", steps[0].kind, steps[1].kind, steps[2].kind)
	}
	gap := steps[1].config["durationMs"].(float64)
	if gap < 999 || gap > 1001 {
		t.Errorf("Sleep duration 应 ≈ 1000ms, got %v", gap)
	}
}

func TestCompactToSteps_NoSleepBelowThreshold(t *testing.T) {
	// 间隔 100ms < 200ms 阈值 — 不插 Sleep.
	events := []inputclip.Event{
		ev(inputclip.EventTypeKeyDown, 0, 'A', 0, 0),
		ev(inputclip.EventTypeKeyUp, 50_000, 'A', 0, 0),
		ev(inputclip.EventTypeKeyDown, 150_000, 'B', 0, 0), // 100ms gap
		ev(inputclip.EventTypeKeyUp, 200_000, 'B', 0, 0),
	}
	steps := compactToSteps(events, tw, th)
	if len(steps) != 2 {
		t.Fatalf("阈值下不插 Sleep, want 2, got %d", len(steps))
	}
	if steps[0].kind != "KeyPress" || steps[1].kind != "KeyPress" {
		t.Errorf("want 2 KeyPress, got [%s, %s]", steps[0].kind, steps[1].kind)
	}
}

func TestCompactToSteps_UnsupportedVKDropped(t *testing.T) {
	// 0xFF 在 vkName 返空 — 该事件跳过.
	events := []inputclip.Event{
		ev(inputclip.EventTypeKeyDown, 0, 0xFF, 0, 0),
		ev(inputclip.EventTypeKeyUp, 50_000, 0xFF, 0, 0),
	}
	steps := compactToSteps(events, tw, th)
	if len(steps) != 0 {
		t.Fatalf("不支持的 vk 应丢, got %+v", steps)
	}
}

func TestBuildSimpleSubgraph_LinearLayout(t *testing.T) {
	events := []inputclip.Event{
		ev(inputclip.EventTypeKeyDown, 0, 'W', 0, 0),
		ev(inputclip.EventTypeKeyUp, 100_000, 'W', 0, 0),
		ev(inputclip.EventTypeKeyDown, 200_000, 'A', 0, 0),
		ev(inputclip.EventTypeKeyUp, 250_000, 'A', 0, 0),
	}
	meta := inputclip.ClipMeta{BaseResolution: [2]int{tw, th}, MouseCounts360: 9000}
	sg := BuildSimpleSubgraph(events, meta, tw, th, "测试录制")

	if sg.ID == "" || sg.Graph.ID == "" {
		t.Error("ID / Graph.ID 应非空")
	}
	if sg.Label != "测试录制" {
		t.Errorf("label want '测试录制', got %s", sg.Label)
	}
	if len(sg.OutputPins) != 1 || sg.OutputPins[0].Name != "done" {
		t.Errorf("应 1 个 OutputPin name=done, got %+v", sg.OutputPins)
	}
	if sg.RecordingContext == nil || sg.RecordingContext.MouseCounts360 != 9000 {
		t.Errorf("RecordingContext 应带 MouseCounts360=9000, got %+v", sg.RecordingContext)
	}

	// 节点拓扑: [SubgraphInput, KeyPress, KeyPress, SubgraphOutput]
	wantKinds := []string{"SubgraphInput", "KeyPress", "KeyPress", "SubgraphOutput"}
	if len(sg.Graph.Nodes) != len(wantKinds) {
		t.Fatalf("节点数 want %d, got %d", len(wantKinds), len(sg.Graph.Nodes))
	}
	for i, w := range wantKinds {
		if sg.Graph.Nodes[i].Kind != w {
			t.Errorf("nodes[%d] want %s, got %s", i, w, sg.Graph.Nodes[i].Kind)
		}
	}
	// 边数 = N-1 (4 节点 = 3 条边)
	if len(sg.Graph.Edges) != 3 {
		t.Fatalf("3 条边, got %d", len(sg.Graph.Edges))
	}
	// SubgraphOutput.config.declID 必须等于 OutputPins[0].ID (validator 要)
	outNode := sg.Graph.Nodes[len(sg.Graph.Nodes)-1]
	if outNode.Kind != "SubgraphOutput" {
		t.Fatalf("最后节点应是 SubgraphOutput")
	}
	if declID, _ := outNode.Config["declID"].(string); declID != sg.OutputPins[0].ID {
		t.Errorf("SubgraphOutput.declID 应=OutputPins[0].ID, want %s, got %s", sg.OutputPins[0].ID, declID)
	}
}

func TestBuildSimpleSubgraph_EmptyEvents(t *testing.T) {
	meta := inputclip.ClipMeta{BaseResolution: [2]int{tw, th}}
	sg := BuildSimpleSubgraph(nil, meta, tw, th, "空录制")
	// 0 step → SubgraphInput → SubgraphOutput, 1 边
	if len(sg.Graph.Nodes) != 2 {
		t.Errorf("空录制应 2 节点 (input+output), got %d", len(sg.Graph.Nodes))
	}
	if len(sg.Graph.Edges) != 1 {
		t.Errorf("空录制应 1 边 (input→output), got %d", len(sg.Graph.Edges))
	}
}

func TestBuildPreciseSubgraph_SinglePlayClip(t *testing.T) {
	meta := inputclip.ClipMeta{BaseResolution: [2]int{tw, th}, MouseCounts360: 9000}
	sg := BuildPreciseSubgraph("clip-xyz", meta, "精准录制")
	wantKinds := []string{"SubgraphInput", "PlayClip", "SubgraphOutput"}
	if len(sg.Graph.Nodes) != 3 {
		t.Fatalf("want 3 nodes, got %d", len(sg.Graph.Nodes))
	}
	for i, w := range wantKinds {
		if sg.Graph.Nodes[i].Kind != w {
			t.Errorf("nodes[%d] want %s, got %s", i, w, sg.Graph.Nodes[i].Kind)
		}
	}
	pc := sg.Graph.Nodes[1]
	if cid, _ := pc.Config["clipID"].(string); cid != "clip-xyz" {
		t.Errorf("PlayClip.clipID want clip-xyz, got %v", pc.Config["clipID"])
	}
	if sg.RecordingContext == nil || sg.RecordingContext.MouseCounts360 != 9000 {
		t.Errorf("precise 也带 RecordingContext")
	}
}

func TestToRatios_ZeroClient(t *testing.T) {
	xr, yr := toRatios(100, 200, 0, 0)
	if xr != 0.5 || yr != 0.5 {
		t.Errorf("client 维度 0 应退化中心, got (%v,%v)", xr, yr)
	}
}
