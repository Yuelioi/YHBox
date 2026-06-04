package node_test

import (
	"encoding/json"
	"testing"
	"unicode"

	nodepkg "yotta/internal/node"

	// 触发所有节点 init().
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)

// lint — 守护 node-spec 风格约定. 任何节点新加 / 改 Spec 违反这些规则 → fail.
//
// 覆盖:
//   1. 所有 data pin (Type != "Exec") Name 首字母大写 (节点级 whitelist 例外).
//   2. Number/Integer/Duration InputSpec.Default 是 json.Number (节点级 whitelist 例外).
//   3. Exec in pin 名统一 "In" (fire-only 节点 — Start/EventTick/SubgraphInput — 没 exec in, 不约束).
//
// kindMigrationPending — 豁免上述约定的节点 kind whitelist (当前空).
var kindMigrationPending = map[string]struct{}{}

func TestSpecConsistency_DataPinNamingConvention(t *testing.T) {
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		if _, skip := kindMigrationPending[spec.Kind]; skip {
			continue
		}
		for _, in := range spec.Inputs {
			if in.Type == nodepkg.TypeExec {
				continue
			}
			if !startsWithUppercase(in.Name) {
				t.Errorf("data InputSpec.Name = %q, want PascalCase (kind=%s)", in.Name, spec.Kind)
			}
		}
		for _, out := range spec.Outputs {
			if out.Type == nodepkg.TypeExec {
				continue
			}
			if !startsWithUppercase(out.Name) {
				t.Errorf("data OutputSpec.Name = %q, want PascalCase (kind=%s)", out.Name, spec.Kind)
			}
		}
	}
}

// TestSpecConsistency_ExecInPinNamedIn 守护 exec in pin 命名约定 — 必须叫 "In".
// fire-only 节点 (Start/EventTick/SubgraphInput) 没 exec in, 不约束.
func TestSpecConsistency_ExecInPinNamedIn(t *testing.T) {
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		for _, in := range spec.Inputs {
			if in.Type != nodepkg.TypeExec {
				continue
			}
			if in.Name != "In" {
				t.Errorf("Exec InputSpec.Name = %q, want \"In\" (kind=%s)", in.Name, spec.Kind)
			}
		}
	}
}

func TestSpecConsistency_NumberDefaultsAreJSONNumber(t *testing.T) {
	numTypes := map[string]struct{}{
		"Number":   {},
		"Integer":  {},
		"Duration": {},
	}
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		if _, skip := kindMigrationPending[spec.Kind]; skip {
			continue
		}
		for _, in := range spec.Inputs {
			if _, ok := numTypes[in.Type]; !ok {
				continue
			}
			if in.Default == nil {
				continue
			}
			if _, ok := in.Default.(json.Number); !ok {
				t.Errorf("kind=%s pin=%s Type=%s Default 类型 %T, 应是 json.Number (精度安全)",
					spec.Kind, in.Name, in.Type, in.Default)
			}
		}
	}
}

func startsWithUppercase(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsUpper(r)
}
