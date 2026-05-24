// internal/node/inputs.go
package node

import "time"

// inputsImpl 实现 Inputs 接口. framework 在 Run 入口前构造.
// 优先级 data-wire > config > exec-data > default 已在 merged 物化好.
type inputsImpl struct {
	merged  map[string]any
	present map[string]bool
}

func newInputs(dataWire, config, execData, defaults map[string]any) *inputsImpl {
	merged := map[string]any{}
	present := map[string]bool{}

	// 倒序填: 低优先级先, 高的覆盖
	for k, v := range defaults {
		merged[k] = v
		present[k] = true
	}
	for k, v := range execData {
		merged[k] = v
		present[k] = true
	}
	for k, v := range config {
		merged[k] = v
		present[k] = true
	}
	for k, v := range dataWire {
		merged[k] = v
		present[k] = true
	}
	return &inputsImpl{merged: merged, present: present}
}

func (i *inputsImpl) Has(name string) bool { return i.present[name] }

func (i *inputsImpl) String(name string) string {
	if v, ok := i.merged[name].(string); ok {
		return v
	}
	return ""
}

func (i *inputsImpl) Float64(name string) float64 {
	switch v := i.merged[name].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}

func (i *inputsImpl) Int(name string) int {
	switch v := i.merged[name].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func (i *inputsImpl) Bool(name string) bool {
	if v, ok := i.merged[name].(bool); ok {
		return v
	}
	return false
}

func (i *inputsImpl) Point(name string) Point {
	if v, ok := i.merged[name].(Point); ok {
		return v
	}
	return Point{}
}

func (i *inputsImpl) Rect(name string) Rect {
	if v, ok := i.merged[name].(Rect); ok {
		return v
	}
	return Rect{}
}

func (i *inputsImpl) Color(name string) Color {
	if v, ok := i.merged[name].(Color); ok {
		return v
	}
	return Color{}
}

func (i *inputsImpl) Duration(name string) time.Duration {
	if v, ok := i.merged[name].(time.Duration); ok {
		return v
	}
	return 0
}

func (i *inputsImpl) JSON(name string) map[string]any {
	if v, ok := i.merged[name].(map[string]any); ok {
		return v
	}
	return nil
}

func (i *inputsImpl) Raw(name string) any { return i.merged[name] }
