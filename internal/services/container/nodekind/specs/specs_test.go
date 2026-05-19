// internal/services/container/nodekind/specs/specs_test.go
package specs

import (
	"sort"
	"testing"

	"yhbox/internal/services/container/nodekind"
)

// expectedKinds 是当前支持的 kind 全集合 — Registry 的真理源 (drift detection).
var expectedKinds = []string{
	// control (11)
	"Start", "Stop", "Sleep", "Loop", "If", "Switch", "Parallel", "Race", "Break", "Continue", "Cron",
	// variables (6 — includes Expr)
	"SetVar", "IncVar", "GetVar", "GetSys", "GetParam", "Expr",
	// purefunc (22)
	"Add", "Sub", "Mul", "Div", "Mod", "Neg",
	"Lt", "LtEq", "Gt", "GtEq", "Eq", "NotEq",
	"And", "Or", "Not",
	"Concat", "Contains", "Length",
	"ToString", "ToNumber", "ToBool",
	"Select",
	// detect (8)
	"WaitTemplate", "CheckTemplate", "ClickTemplate", "DetectColor",
	"DetectColorHSV", "ROIColorScan", "Screenshot", "ColorBarTrack",
	// input (10)
	"ClickAt", "KeyPress", "MouseMoveRel", "Scroll",
	"KeyHoldStart", "KeyHoldStop", "MouseHoldStart", "MouseHoldStop",
	"OnEvent", "PlayClip",
	// system (15)
	"BringGameForeground", "MouseCalibration", "WindowTarget", "CommentBox",
	"Try", "Throw",
	"StopwatchStart", "StopwatchStop", "StopwatchRead",
	"Log", "Toast",
	"Subgraph", "SubgraphInput", "SubgraphOutput", "CollapsedNode",
}

func TestAllKindsRegistered(t *testing.T) {
	got := nodekind.Kinds()
	gotSet := map[string]bool{}
	for _, k := range got {
		gotSet[k] = true
	}
	for _, want := range expectedKinds {
		if !gotSet[want] {
			t.Errorf("kind %q missing from registry", want)
		}
	}
	expSet := map[string]bool{}
	for _, k := range expectedKinds {
		expSet[k] = true
	}
	for _, k := range got {
		if !expSet[k] {
			t.Errorf("kind %q in registry but not in expectedKinds (update test or remove spec)", k)
		}
	}
	t.Logf("registered %d kinds (expected %d)", len(got), len(expectedKinds))
}

func TestSpecConsistency(t *testing.T) {
	for _, kind := range nodekind.Kinds() {
		spec, _ := nodekind.Get(kind)
		// IsPureData implies no exec pins.
		if spec.IsPureData && (len(spec.ExecIn) > 0 || len(spec.ExecOut) > 0 || spec.ExecOutFn != nil) {
			t.Errorf("kind %q: IsPureData=true but has exec pins", kind)
		}
		// IsVisualOnly implies no exec, no data pins.
		if spec.IsVisualOnly {
			if len(spec.ExecIn) > 0 || len(spec.ExecOut) > 0 || len(spec.DataIn) > 0 || len(spec.DataOut) > 0 {
				t.Errorf("kind %q: IsVisualOnly=true but has pins", kind)
			}
		}
	}
}

// Stable order helps debugging: print sorted kind list on test failure.
func TestKindsAreLexicallyAccessible(t *testing.T) {
	ks := nodekind.Kinds()
	sort.Strings(ks)
	t.Logf("registered kinds (sorted): %v", ks)
}
