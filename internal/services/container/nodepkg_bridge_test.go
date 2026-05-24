// nodepkg_bridge_test.go — Phase 6+ partial (atomic #2).
//
// 验证 validator helper 通过 nodepkg union 能识别新 framework Spec (PascalCase pin name + PascalCase Type tag).
// 用本地 Register fake kind 避免依赖 internal/nodes/* anonymous import — 这样不影响现有
// TestExecOutPinsForNode_Switch_Dynamic 等 test (它们假设 nodepkg 未注册).
package container

import (
	"testing"

	nodepkg "yhbox/internal/node"
)

// nbCheckTemplate 测试用 fake — 模拟新 framework CheckTemplate Spec (PascalCase pin + Type).
// Kind 用 "NBCheckTemplate" 避免跟真 CheckTemplate (老 nodekind 已注册) 冲突.
type nbCheckTemplate struct{}

func (nbCheckTemplate) Spec() nodepkg.Spec {
	return nodepkg.Spec{
		Kind: "NBCheckTemplate",
		Inputs: []nodepkg.InputSpec{
			{Name: "in", Type: "Exec"},
			{Name: "Template", Type: "String", Required: true},
			{Name: "Threshold", Type: "Number", Default: 0.8},
		},
		Outputs: []nodepkg.OutputSpec{
			{Name: "Found", Type: "Exec"},
			{Name: "NotFound", Type: "Exec"},
			{Name: "Point", Type: "Point"},
		},
	}
}
func (nbCheckTemplate) Run(_ nodepkg.Ctx, _ nodepkg.Inputs) (nodepkg.Outputs, error) {
	return nil, nil
}

// nbPureAdd fake pure-data kind (复刻 Add) — 验证 IsPureData + dataOut Type=Number 路径.
type nbPureAdd struct{}

func (nbPureAdd) Spec() nodepkg.Spec {
	return nodepkg.Spec{
		Kind: "NBPureAdd",
		Inputs: []nodepkg.InputSpec{
			{Name: "a", Type: "Number"},
			{Name: "b", Type: "Number"},
		},
		Outputs:    []nodepkg.OutputSpec{{Name: "result", Type: "Number"}},
		IsPureData: true,
	}
}
func (nbPureAdd) Run(_ nodepkg.Ctx, _ nodepkg.Inputs) (nodepkg.Outputs, error) { return nil, nil }

// registerNBFakes Register fake kinds into nodepkg. Tests run serial; ResetRegistryForTest
// 会清掉 nodekind 包的真节点不影响 (nodekind 是另一个 registry).
func registerNBFakes(t *testing.T) {
	t.Helper()
	nodepkg.ResetRegistryForTest()
	nodepkg.Register(&nbCheckTemplate{})
	nodepkg.Register(&nbPureAdd{})
}

func TestNodepkgBridge_PinExistsPascalCase(t *testing.T) {
	registerNBFakes(t)
	// out: Found / NotFound / Point 应都存在 (新 PascalCase).
	for _, p := range []string{"Found", "NotFound", "Point"} {
		if !pinExists("NBCheckTemplate", p, true) {
			t.Errorf("pinExists out %q should be true via nodepkg union", p)
		}
	}
	// in: Template / Threshold / in (Exec).
	for _, p := range []string{"Template", "Threshold", "in"} {
		if !pinExists("NBCheckTemplate", p, false) {
			t.Errorf("pinExists in %q should be true via nodepkg union", p)
		}
	}
	// 不存在 pin — 应返 false.
	if pinExists("NBCheckTemplate", "BogusPin", true) {
		t.Error("BogusPin should not exist")
	}
}

func TestNodepkgBridge_IsDataOutPinUsesNodepkg(t *testing.T) {
	registerNBFakes(t)
	// Point (Type=Point) → data out.
	if !IsDataOutPin("NBCheckTemplate", "Point") {
		t.Error("Point should be data-out via nodepkg union")
	}
	// Found (Type=Exec) → 不是 data out.
	if IsDataOutPin("NBCheckTemplate", "Found") {
		t.Error("Found (Exec) should NOT be data-out")
	}
}

func TestNodepkgBridge_DataInPinTypeCanonicalized(t *testing.T) {
	registerNBFakes(t)
	// nodepkg "Number" → canonicalize → "number".
	got := dataInPinTypeForKind("NBCheckTemplate", "Threshold")
	if got != "number" {
		t.Errorf("Threshold type = %q, want \"number\" (canonicalized from nodepkg PascalCase \"Number\")", got)
	}
	// "String" → "string"
	got = dataInPinTypeForKind("NBCheckTemplate", "Template")
	if got != "string" {
		t.Errorf("Template type = %q, want \"string\"", got)
	}
	// Exec pin returns "" (not data type)
	got = dataInPinTypeForKind("NBCheckTemplate", "in")
	if got != "" {
		t.Errorf("Exec pin 'in' should return \"\" (not data type), got %q", got)
	}
}

func TestNodepkgBridge_DataOutPinTypeCanonicalized(t *testing.T) {
	registerNBFakes(t)
	got := dataOutPinTypeForKind("NBPureAdd", "result")
	if got != "number" {
		t.Errorf("result type = %q, want \"number\"", got)
	}
	got = dataOutPinTypeForKind("NBCheckTemplate", "Point")
	if got != "point" {
		t.Errorf("Point type = %q, want \"point\"", got)
	}
}

func TestNodepkgBridge_KnownKindUnion(t *testing.T) {
	registerNBFakes(t)
	// nodepkg-only kind → known (老 nodekind 没注册 "NBCheckTemplate")
	if !knownKind("NBCheckTemplate") {
		t.Error("NBCheckTemplate should be knownKind via nodepkg union")
	}
	// 老 nodekind kind 仍 known
	if !knownKind("Sleep") {
		t.Error("Sleep (老 nodekind 注册) should still be knownKind")
	}
	if knownKind("BogusUnregistered") {
		t.Error("BogusUnregistered should NOT be knownKind")
	}
}

func TestNodepkgBridge_ExecOutPinsNodepkgOnly(t *testing.T) {
	registerNBFakes(t)
	n := &GraphNode{Kind: "NBCheckTemplate"}
	pins := execOutPinsForNode(n)
	for _, want := range []string{"Found", "NotFound"} {
		if _, ok := pins[want]; !ok {
			t.Errorf("execOutPinsForNode should include %q via nodepkg, got %v", want, pins)
		}
	}
	// Point 是 data-out, 不算 exec-out
	if _, ok := pins["Point"]; ok {
		t.Errorf("Point is data-out, should not be in exec-out set: %v", pins)
	}
}

func TestNodepkgBridge_ExecInPinsNodepkgOnly(t *testing.T) {
	registerNBFakes(t)
	pins := execInPinsOf("NBCheckTemplate")
	found := false
	for _, p := range pins {
		if p == "in" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("execInPinsOf should include \"in\" via nodepkg, got %v", pins)
	}
}

func TestNodepkgBridge_OldLowercasePinFallback(t *testing.T) {
	// 不 reset nodepkg — 让既不 register fake, nodepkg.Get(\"Sleep\") 返 false → fallback 老 nodekind.
	// 验证 Sleep (nodekind 老 lowercase \"in\") 还能查到.
	nodepkg.ResetRegistryForTest()
	if !pinExists("Sleep", "in", false) {
		t.Error("Sleep 'in' pin should exist via nodekind fallback")
	}
	if !pinExists("Sleep", "out", true) {
		t.Error("Sleep 'out' pin should exist via nodekind fallback")
	}
}
