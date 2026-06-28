package node_test

import (
	"encoding/json"
	"testing"
	"unicode"

	nodepkg "yotta/internal/node"

	_ "yotta/internal/nodes/all"
)

// lint — 守护 node-spec 风格约定. 任何节点新加 / 改 Spec 违反这些规则 → fail.
//
// 覆盖:
//  1. 所有 data pin (Type != "Exec") Name 首字母大写 (节点级 whitelist 例外).
//  2. Number/Integer/Duration InputSpec.Default 是 json.Number (节点级 whitelist 例外).
//  3. Exec in pin 名统一 "In" (fire-only 节点 — Start/EventTick/SubgraphInput — 没 exec in, 不约束).
//
// kindMigrationPending — 豁免上述约定的节点 kind whitelist (当前空).
var kindMigrationPending = map[string]struct{}{}

// TestSpecConsistency_DynamicFlagsMutuallyExclusive — DynamicOutputs(出口名动态)与
// DynamicDataFields(Data 字段集由 config 声明)语义正交, 不许同一 kind 并开。
func TestSpecConsistency_DynamicFlagsMutuallyExclusive(t *testing.T) {
	for _, rn := range nodepkg.All() {
		if rn.Spec.DynamicOutputs && rn.Spec.DynamicDataFields {
			t.Errorf("kind %q 同开 DynamicOutputs + DynamicDataFields(互斥)", rn.Spec.Kind)
		}
	}
}

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

func TestSpecConsistency_WidgetPropsShape(t *testing.T) {
	validKinds := map[string]bool{
		"":                true,
		"ai-connection":   true,
		"async-dropdown":  true,
		"checkbox":        true,
		"code":            true,
		"color-preset":    true,
		"dropdown":        true,
		"duration":        true,
		"expr":            true,
		"icon-preset":     true,
		"json":            true,
		"key-capture":     true,
		"number":          true,
		"password":        true,
		"rect-editor":     true,
		"slider":          true,
		"template-picker": true,
		"text":            true,
		"textarea":        true,
	}
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		inputNames := map[string]nodepkg.InputSpec{}
		for _, in := range spec.Inputs {
			inputNames[in.Name] = in
		}
		for _, in := range spec.Inputs {
			w := in.Widget
			if !validKinds[w.Kind] {
				t.Errorf("kind=%s pin=%s unknown widget kind %q", spec.Kind, in.Name, w.Kind)
				continue
			}
			switch w.Kind {
			case "dropdown":
				validateDropdownProps(t, spec.Kind, in.Name, w.Props)
			case "slider":
				validateSliderProps(t, spec.Kind, in.Name, w.Props)
			case "async-dropdown":
				validateAsyncDropdownProps(t, spec.Kind, in.Name, w.Props, inputNames)
			}
		}
	}
}

func validateDropdownProps(t *testing.T, kind, pin string, props map[string]any) {
	t.Helper()
	options, ok := props["options"].([]any)
	if !ok || len(options) == 0 {
		t.Errorf("kind=%s pin=%s dropdown requires non-empty options, got %#v", kind, pin, props["options"])
		return
	}
	for i, raw := range options {
		opt, ok := raw.(map[string]any)
		if !ok {
			t.Errorf("kind=%s pin=%s dropdown option[%d] type %T, want object", kind, pin, i, raw)
			continue
		}
		if _, ok := opt["value"]; !ok {
			t.Errorf("kind=%s pin=%s dropdown option[%d] missing value", kind, pin, i)
		}
	}
}

func validateSliderProps(t *testing.T, kind, pin string, props map[string]any) {
	t.Helper()
	min, minOK := numberProp(props, "min")
	max, maxOK := numberProp(props, "max")
	step, stepOK := numberProp(props, "step")
	if !minOK || !maxOK || !stepOK {
		t.Errorf("kind=%s pin=%s slider requires numeric min/max/step, got %#v", kind, pin, props)
		return
	}
	if min >= max {
		t.Errorf("kind=%s pin=%s slider min >= max (%v >= %v)", kind, pin, min, max)
	}
	if step <= 0 {
		t.Errorf("kind=%s pin=%s slider step must be > 0, got %v", kind, pin, step)
	}
}

func validateAsyncDropdownProps(t *testing.T, kind, pin string, props map[string]any, inputs map[string]nodepkg.InputSpec) {
	t.Helper()
	source, ok := props["asyncSource"].(string)
	if !ok || source == "" {
		t.Errorf("kind=%s pin=%s async-dropdown requires asyncSource, got %#v", kind, pin, props["asyncSource"])
	}
	if raw, ok := props["applyMeta"]; ok {
		applyMeta, ok := raw.(map[string]any)
		if !ok {
			t.Errorf("kind=%s pin=%s applyMeta type %T, want object", kind, pin, raw)
			return
		}
		for metaKey, rawTarget := range applyMeta {
			target, ok := rawTarget.(string)
			if metaKey == "" || !ok || target == "" {
				t.Errorf("kind=%s pin=%s invalid applyMeta entry %q=%#v", kind, pin, metaKey, rawTarget)
				continue
			}
			targetInput, exists := inputs[target]
			if !exists {
				t.Errorf("kind=%s pin=%s applyMeta target %q does not exist", kind, pin, target)
				continue
			}
			if targetInput.Type == nodepkg.TypeExec {
				t.Errorf("kind=%s pin=%s applyMeta target %q is exec input", kind, pin, target)
			}
		}
	}
}

func numberProp(props map[string]any, key string) (float64, bool) {
	switch v := props[key].(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func startsWithUppercase(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsUpper(r)
}
