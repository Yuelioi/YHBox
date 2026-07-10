package node_test

import (
	"encoding/json"
	"testing"
	"unicode"

	"github.com/yottaapp/yotta/internal/automation/controller"
	nodepkg "github.com/yottaapp/yotta/internal/node"

	_ "github.com/yottaapp/yotta/internal/nodes/all"
)

// lint — 守护 node-spec 风格约定. 任何节点新加 / 改 Spec 违反这些规则 → fail.
//
// 覆盖:
//  1. 所有 data pin (Type != "Exec") Name 首字母大写 (节点级 whitelist 例外).
//  2. Number/Integer/Duration InputSpec.Default 是 json.Number (节点级 whitelist 例外).
//  3. Exec in pin 名统一 "In" (fire-only 节点 — Start/EventTick/SubgraphInput — 没 exec in, 不约束).
//  4. Exec out pin 名统一 PascalCase; Switch 的 "default" 是唯一小写例外.
//  5. String InputSpec.Default 若设置, 必须是 string; nil 表示无默认/必填.
//  6. Bool InputSpec.Default 若设置, 必须是 bool; nil 表示无默认/必填.
//  7. Point/Rect InputSpec.Default 若设置, 必须是 node.Point/node.Rect.
//  8. JSON InputSpec.Default 若设置, 必须是 map[string]any.
//  9. FieldSchema 递归结构必须 well-formed.
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

func TestSpecConsistency_TargetCapabilitiesKnownByController(t *testing.T) {
	allControllerCaps := controller.CapabilitySet{
		Screenshot:   true,
		Click:        true,
		Move:         true,
		Scroll:       true,
		MouseButton:  true,
		Drag:         true,
		MoveRelative: true,
		KeyChord:     true,
		KeyState:     true,
		Text:         true,
		StartApp:     true,
		StopApp:      true,
	}
	for _, rn := range nodepkg.All() {
		for _, cap := range rn.Spec.TargetCapabilities {
			if !allControllerCaps.Has(controller.Capability(cap)) {
				t.Errorf("kind=%s target capability %q is not recognized by controller.CapabilitySet", rn.Spec.Kind, cap)
			}
		}
	}
}

func TestSpecConsistency_NoDuplicatePinsWithinNode(t *testing.T) {
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		inputTypes := map[string]string{}
		for _, in := range spec.Inputs {
			if prev, ok := inputTypes[in.Name]; ok {
				if prev == in.Type {
					t.Errorf("kind=%s duplicate input pin %q type=%s", spec.Kind, in.Name, in.Type)
				} else {
					t.Errorf("kind=%s input pin %q declared with multiple types: %s / %s", spec.Kind, in.Name, prev, in.Type)
				}
				continue
			}
			inputTypes[in.Name] = in.Type
		}

		outputTypes := map[string]string{}
		for _, out := range spec.Outputs {
			if prev, ok := outputTypes[out.Name]; ok {
				if prev == out.Type {
					t.Errorf("kind=%s duplicate output pin %q type=%s", spec.Kind, out.Name, out.Type)
				} else {
					t.Errorf("kind=%s output pin %q declared with multiple types: %s / %s", spec.Kind, out.Name, prev, out.Type)
				}
				continue
			}
			outputTypes[out.Name] = out.Type

			dataTypes := map[string]string{}
			for _, data := range out.Data {
				if prev, ok := dataTypes[data.Name]; ok {
					if prev == data.Type {
						t.Errorf("kind=%s output=%s duplicate data field %q type=%s", spec.Kind, out.Name, data.Name, data.Type)
					} else {
						t.Errorf("kind=%s output=%s data field %q declared with multiple types: %s / %s", spec.Kind, out.Name, data.Name, prev, data.Type)
					}
					continue
				}
				dataTypes[data.Name] = data.Type
			}
		}
	}
}

func TestSpecConsistency_SupportedTargetsAreDerived(t *testing.T) {
	publicTargets := map[string]bool{
		nodepkg.SupportedTargetWin32Window: true,
		nodepkg.SupportedTargetAndroidADB:  true,
	}
	for _, rn := range nodepkg.All() {
		if len(rn.Spec.SupportedTargets) > 0 {
			t.Errorf("kind=%s sets SupportedTargets directly; use NeedsTarget/NeedsWindow/TargetCapabilities and let exporters derive it", rn.Spec.Kind)
		}
		for _, target := range rn.Spec.PlatformTargets {
			if !publicTargets[target] {
				t.Errorf("kind=%s PlatformTargets contains unknown public target %q", rn.Spec.Kind, target)
			}
		}
		derived := nodepkg.SupportedTargetsForSpec(rn.Spec)
		if rn.Spec.NeedsTarget || rn.Spec.NeedsWindow || rn.Spec.Category == "Target" || len(rn.Spec.PlatformTargets) > 0 {
			if len(derived) == 0 {
				t.Errorf("kind=%s has target/window semantics but no derived supported targets", rn.Spec.Kind)
			}
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

func TestSpecConsistency_ExecOutPinNamingConvention(t *testing.T) {
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		for _, out := range spec.Outputs {
			if out.Type != nodepkg.TypeExec {
				continue
			}
			if out.Name == "default" {
				continue
			}
			if !startsWithUppercase(out.Name) {
				t.Errorf("Exec OutputSpec.Name = %q, want PascalCase or reserved \"default\" (kind=%s)", out.Name, spec.Kind)
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

func TestSpecConsistency_StringDefaultsAreStringWhenSet(t *testing.T) {
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		if _, skip := kindMigrationPending[spec.Kind]; skip {
			continue
		}
		for _, in := range spec.Inputs {
			if in.Type != "String" {
				continue
			}
			if in.Default == nil {
				continue
			}
			if _, ok := in.Default.(string); !ok {
				t.Errorf("kind=%s pin=%s Type=String Default 类型 %T, 应是 string", spec.Kind, in.Name, in.Default)
			}
		}
	}
}

func TestSpecConsistency_BoolDefaultsAreBoolWhenSet(t *testing.T) {
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		if _, skip := kindMigrationPending[spec.Kind]; skip {
			continue
		}
		for _, in := range spec.Inputs {
			if in.Type != "Bool" {
				continue
			}
			if in.Default == nil {
				continue
			}
			if _, ok := in.Default.(bool); !ok {
				t.Errorf("kind=%s pin=%s Type=Bool Default 类型 %T, 应是 bool", spec.Kind, in.Name, in.Default)
			}
		}
	}
}

func TestSpecConsistency_GeometryDefaultsUseNodeTypesWhenSet(t *testing.T) {
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		if _, skip := kindMigrationPending[spec.Kind]; skip {
			continue
		}
		for _, in := range spec.Inputs {
			if in.Default == nil {
				continue
			}
			switch in.Type {
			case "Point":
				if _, ok := in.Default.(nodepkg.Point); !ok {
					t.Errorf("kind=%s pin=%s Type=Point Default 类型 %T, 应是 node.Point", spec.Kind, in.Name, in.Default)
				}
			case "Rect":
				if _, ok := in.Default.(nodepkg.Rect); !ok {
					t.Errorf("kind=%s pin=%s Type=Rect Default 类型 %T, 应是 node.Rect", spec.Kind, in.Name, in.Default)
				}
			}
		}
	}
}

func TestSpecConsistency_JSONDefaultsAreObjectWhenSet(t *testing.T) {
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		if _, skip := kindMigrationPending[spec.Kind]; skip {
			continue
		}
		for _, in := range spec.Inputs {
			if in.Type != "JSON" {
				continue
			}
			if in.Default == nil {
				continue
			}
			if _, ok := in.Default.(map[string]any); !ok {
				t.Errorf("kind=%s pin=%s Type=JSON Default 类型 %T, 应是 map[string]any", spec.Kind, in.Name, in.Default)
			}
		}
	}
}

func TestSpecConsistency_FieldSchemasAreWellFormed(t *testing.T) {
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		for _, in := range spec.Inputs {
			if in.Schema == nil {
				continue
			}
			validateFieldSchema(t, spec.Kind, in.Name, in.Schema)
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

func validateFieldSchema(t *testing.T, kind, path string, schema *nodepkg.FieldSchema) {
	t.Helper()
	switch schema.Type {
	case "object", "tuple":
		for i, field := range schema.Fields {
			if field.Key == "" {
				t.Errorf("kind=%s schema=%s field[%d] missing key", kind, path, i)
			}
			if field.Schema == nil {
				t.Errorf("kind=%s schema=%s.%s missing nested schema", kind, path, field.Key)
				continue
			}
			validateFieldSchema(t, kind, path+"."+field.Key, field.Schema)
		}
	case "array":
		if schema.Items == nil {
			t.Errorf("kind=%s schema=%s array missing items schema", kind, path)
			return
		}
		validateFieldSchema(t, kind, path+"[]", schema.Items)
	case "enum":
		if len(schema.Options) == 0 {
			t.Errorf("kind=%s schema=%s enum missing options", kind, path)
		}
		for i, opt := range schema.Options {
			if opt.Value == nil {
				t.Errorf("kind=%s schema=%s enum option[%d] missing value", kind, path, i)
			}
		}
	case "number", "string", "bool":
		if schema.Items != nil || len(schema.Fields) > 0 || len(schema.Options) > 0 {
			t.Errorf("kind=%s schema=%s scalar type %q has nested fields/items/options", kind, path, schema.Type)
		}
	default:
		t.Errorf("kind=%s schema=%s unknown FieldSchema.Type %q", kind, path, schema.Type)
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
