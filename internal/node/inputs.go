// internal/node/inputs.go
package node

import (
	"encoding/json"
	"strconv"
	"time"
)

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

// NewInputsFromConfig 公开 helper — 从节点 Config map 构造 Inputs 实现. 给静态分析
// 路径 (library scanner / dependency extractor / lint) 用, 不走 dispatch wire/exec-data/
// defaults 路径. 让 nodepkg.Dependencies(in) 接口可以在 scanner 里以非 runtime 方式调用.
func NewInputsFromConfig(cfg map[string]any) Inputs {
	merged := map[string]any{}
	present := map[string]bool{}
	for k, v := range cfg {
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
	case json.Number:
		// InputSpec.Default 用 json.Number 保精度 (DS r2 #9a)
		f, _ := v.Float64()
		return f
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

func (i *inputsImpl) Int(name string) int {
	switch v := i.merged[name].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
		if f, err := v.Float64(); err == nil {
			return int(f)
		}
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
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
	switch v := i.merged[name].(type) {
	case time.Duration:
		return v
	case float64:
		return time.Duration(v) * time.Millisecond
	case int:
		return time.Duration(v) * time.Millisecond
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return time.Duration(n) * time.Millisecond
		}
		if f, err := v.Float64(); err == nil {
			return time.Duration(f) * time.Millisecond
		}
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Duration(n) * time.Millisecond
		}
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

func (i *inputsImpl) Keys() []string {
	out := make([]string, 0, len(i.merged))
	for k := range i.merged {
		out = append(out, k)
	}
	return out
}
