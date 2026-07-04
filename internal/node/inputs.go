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
		if k == "literal" {
			continue // 单独 overlay (下方), 不把 literal 这个 meta key 自身当 input
		}
		merged[k] = v
		present[k] = true
	}
	// pin 字面量正源 = config.literal[name]; 优先级镜像 dispatch newInputs (literal > 顶层 config)。
	// 让 scanner / dependency extractor (PlayClip ClipID / Subgraph SubgraphID 等) 也读到 literal 值。
	if lit, ok := cfg["literal"].(map[string]any); ok {
		for k, v := range lit {
			merged[k] = v
			present[k] = true
		}
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

// StringList 读 string 列表型 pin (e.g. 模板节点 Templates). JSON 反序列化成 []any,
// 也容忍 []string 和裸 string (单值当一元列表). 空串元素跳过.
func (i *inputsImpl) StringList(name string) []string {
	switch v := i.merged[name].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if v != "" {
			return []string{v}
		}
	}
	return nil
}

// List 读 List 型 pin. 容忍 []any (原样) / []string (转 []any, 一次性分配不缓存);
// nil 及其它任意类型 → nil (不 panic). 不把裸 string 当一元列表 (与 StringList 区别).
func (i *inputsImpl) List(name string) []any {
	switch v := i.merged[name].(type) {
	case []any:
		return v
	case []string:
		out := make([]any, len(v))
		for idx, s := range v {
			out[idx] = s
		}
		return out
	}
	return nil
}

func (i *inputsImpl) File(name string) (File, bool) {
	if v, ok := i.merged[name].(File); ok {
		return v, true
	}
	return File{}, false
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

func (i *inputsImpl) Window(name string) (Window, bool) {
	if v, ok := i.merged[name].(Window); ok {
		return v, true
	}
	return Window{}, false
}

func (i *inputsImpl) Rect(name string) Rect {
	if v, ok := i.merged[name].(Rect); ok {
		return v
	}
	return Rect{}
}

func (i *inputsImpl) Geometry(name string) Geometry {
	if v, ok := i.merged[name].(Geometry); ok {
		return v
	}
	return Geometry{}
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

func (i *inputsImpl) JSONValue(name string) any { return i.merged[name] }

func (i *inputsImpl) Raw(name string) any { return i.merged[name] }

func (i *inputsImpl) Keys() []string {
	out := make([]string, 0, len(i.merged))
	for k := range i.merged {
		out = append(out, k)
	}
	return out
}
