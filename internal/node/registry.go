// internal/node/registry.go
package node

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
)

// RegisteredNode framework 注册条目, 含 Spec + 探测出的扩展 interface fn 指针.
type RegisteredNode struct {
	// Impl 仅作 identity / debug / reflect 用. engine dispatch 必须走下面 cached
	// capability funcs, 不许 impl.(Runnable) / impl.(Evaluator) 之类 runtime type assert
	// (Register 已探测一次, 缓存到 function pointer, 重复 assert 是污染).
	Impl Node
	Spec Spec

	// exactly-one capability — Register 时验证. nil = 节点未实现该 capability.
	Run       func(Ctx, Inputs) (Outputs, error)
	RunRegion func(Ctx, Inputs, func(Ctx) (string, error)) (Outputs, error)
	Evaluate  func(Ctx, Inputs) (any, error)

	// Optional extensions (unchanged)
	Validate     func(in Inputs) []ValidationError
	Dependencies func(in Inputs) []Dependency

	// Defaults — Spec.Inputs 中非 nil Default 值的 name→value 缓存. per-token 不再
	// 重算 defaultsFromSpec — Register 时 build 一次, 引擎 newInputs 复用.
	Defaults map[string]any
}

// Registry owns a set of node implementations. Instances are independent and
// may be assembled in parallel by tests or embedders.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]*RegisteredNode
	frozen  bool
}

// RegistrySnapshot is an immutable point-in-time registry view. Readers receive
// defensive copies, so callers cannot mutate the stored metadata generation.
type RegistrySnapshot struct{ entries map[string]*RegisteredNode }

// RegistryReader is the read surface consumed by catalog and runtime assembly.
type RegistryReader interface {
	Get(kind string) (*RegisteredNode, bool)
	All() []*RegisteredNode
}

func NewRegistry() *Registry { return &Registry{entries: map[string]*RegisteredNode{}} }

var defaultRegistry = NewRegistry()

// Register 仅在 init() / Wails OnStartup 前调.
func Register(impl Node) {
	defaultRegistry.Register(impl)
}

// Register adds an implementation and validates its complete node contract.
func (r *Registry) Register(impl Node) {
	// Spec is extension code. Evaluate and validate it without holding the
	// registry lock so a slow or re-entrant implementation cannot block readers.
	spec := cloneSpec(impl.Spec())
	rn := &RegisteredNode{
		Impl:     impl,
		Spec:     spec,
		Defaults: defaultsFromSpec(&spec),
	}
	// 扩展 interface 探测
	if v, ok := impl.(Validator); ok {
		rn.Validate = v.Validate
	}
	if d, ok := impl.(Dependencer); ok {
		rn.Dependencies = d.Dependencies
	}
	// Capability 探测 (C5b 激活 strict validation).
	if r, ok := impl.(Runnable); ok {
		rn.Run = r.Run
	}
	if r, ok := impl.(RegionRunner); ok {
		rn.RunRegion = r.RunRegion
	}
	if e, ok := impl.(Evaluator); ok {
		rn.Evaluate = e.Evaluate
	}

	// Strict capability invariant:
	//   - 非 IsGraphMarker / IsVisualOnly 节点必须 exactly one capability
	//   - IsPureData 节点必须 Evaluator
	// 违反 → Register panic, init-time 失败立刻可见.
	capCount := 0
	if rn.Run != nil {
		capCount++
	}
	if rn.RunRegion != nil {
		capCount++
	}
	if rn.Evaluate != nil {
		capCount++
	}
	isException := spec.IsGraphMarker || spec.IsVisualOnly
	if !isException && capCount == 0 {
		panic(fmt.Sprintf("node %q: zero capabilities — must implement one of Runnable/RegionRunner/Evaluator (or set IsGraphMarker/IsVisualOnly)", spec.Kind))
	}
	if capCount > 1 {
		panic(fmt.Sprintf("node %q: multiple capabilities (%d) — exactly one of Runnable/RegionRunner/Evaluator allowed", spec.Kind, capCount))
	}
	if spec.IsPureData && rn.Evaluate == nil {
		panic(fmt.Sprintf("node %q: IsPureData=true but missing Evaluator", spec.Kind))
	}
	if spec.NeedsWindow {
		hasWin := false
		for _, ip := range spec.Inputs {
			if ip.Name == "Window" && ip.Type == "Window" {
				hasWin = true
				break
			}
		}
		if !hasWin {
			panic(fmt.Sprintf("node %q: NeedsWindow=true but missing Window input — spread node.WindowInputSpec() into Inputs", spec.Kind))
		}
	}
	seenRuntimeCapabilities := make(map[RuntimeCapability]struct{}, len(spec.RuntimeCapabilities))
	for _, capability := range spec.RuntimeCapabilities {
		if _, known := (ServiceBundle{}).runtimeCapabilityAvailable(capability); !known {
			panic(fmt.Sprintf("node %q: unknown runtime capability %q", spec.Kind, capability))
		}
		if _, duplicate := seenRuntimeCapabilities[capability]; duplicate {
			panic(fmt.Sprintf("node %q: duplicate runtime capability %q", spec.Kind, capability))
		}
		seenRuntimeCapabilities[capability] = struct{}{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.entries == nil {
		r.entries = map[string]*RegisteredNode{}
	}
	if r.frozen {
		panic("node.Register after Freeze (must call before Wails OnStartup returns)")
	}
	if _, exists := r.entries[spec.Kind]; exists {
		panic(fmt.Sprintf("node kind %q already registered", spec.Kind))
	}
	r.entries[spec.Kind] = rn
}

// Freeze 在 Wails OnStartup 末尾调一次.
func Freeze() {
	defaultRegistry.Freeze()
}

func (r *Registry) Freeze() {
	r.mu.Lock()
	r.frozen = true
	r.mu.Unlock()
}

// Get RPC handler 用. RLock 并发安全.
func Get(kind string) (*RegisteredNode, bool) {
	return defaultRegistry.Get(kind)
}

func (r *Registry) Get(kind string) (*RegisteredNode, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rn, ok := r.entries[kind]
	if !ok {
		return nil, false
	}
	return cloneRegisteredNode(rn), true
}

// All 返 snapshot copy.
func All() []*RegisteredNode {
	return defaultRegistry.All()
}

func (r *Registry) All() []*RegisteredNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return allRegisteredNodes(r.entries)
}

func (r *Registry) Snapshot() RegistrySnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries := make(map[string]*RegisteredNode, len(r.entries))
	for kind, registered := range r.entries {
		entries[kind] = registered
	}
	return RegistrySnapshot{entries: entries}
}

func DefaultRegistrySnapshot() RegistrySnapshot { return defaultRegistry.Snapshot() }

// SnapshotRegistry freezes any RegistryReader into an independent generation.
func SnapshotRegistry(reader RegistryReader) RegistrySnapshot {
	if reader == nil {
		panic("node: registry is required")
	}
	entries := map[string]*RegisteredNode{}
	for _, registered := range reader.All() {
		entries[registered.Spec.Kind] = cloneRegisteredNode(registered)
	}
	return RegistrySnapshot{entries: entries}
}

func (s RegistrySnapshot) Get(kind string) (*RegisteredNode, bool) {
	rn, ok := s.entries[kind]
	if !ok {
		return nil, false
	}
	return cloneRegisteredNode(rn), true
}

func (s RegistrySnapshot) All() []*RegisteredNode { return allRegisteredNodes(s.entries) }

func allRegisteredNodes(entries map[string]*RegisteredNode) []*RegisteredNode {
	out := make([]*RegisteredNode, 0, len(entries))
	for _, rn := range entries {
		out = append(out, cloneRegisteredNode(rn))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Spec.Kind < out[j].Spec.Kind })
	return out
}

func cloneRegisteredNode(source *RegisteredNode) *RegisteredNode {
	clone := *source
	clone.Spec = cloneSpec(source.Spec)
	clone.Defaults = cloneMap(source.Defaults)
	return &clone
}

func cloneSpec(source Spec) Spec {
	return cloneValue(reflect.ValueOf(source), map[cloneVisit]bool{}).Interface().(Spec)
}

func cloneMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	return cloneValue(reflect.ValueOf(source), map[cloneVisit]bool{}).Interface().(map[string]any)
}

type cloneVisit struct {
	typ reflect.Type
	ptr uintptr
}

// cloneValue preserves named Go types in defaults and widget metadata. JSON
// round-tripping is deliberately unsuitable here because it erases types such
// as Point and concrete widget property structs.
func cloneValue(source reflect.Value, active map[cloneVisit]bool) reflect.Value {
	if !source.IsValid() {
		return source
	}
	switch source.Kind() {
	case reflect.Interface:
		if source.IsNil() {
			return reflect.Zero(source.Type())
		}
		clone := reflect.New(source.Type()).Elem()
		clone.Set(cloneValue(source.Elem(), active))
		return clone
	case reflect.Pointer:
		if source.IsNil() {
			return reflect.Zero(source.Type())
		}
		visit := cloneVisit{typ: source.Type(), ptr: source.Pointer()}
		if active[visit] {
			panic("node metadata contains a reference cycle")
		}
		active[visit] = true
		defer delete(active, visit)
		clone := reflect.New(source.Type().Elem())
		clone.Elem().Set(cloneValue(source.Elem(), active))
		return clone
	case reflect.Slice:
		if source.IsNil() {
			return reflect.Zero(source.Type())
		}
		visit := cloneVisit{typ: source.Type(), ptr: source.Pointer()}
		if active[visit] {
			panic("node metadata contains a reference cycle")
		}
		active[visit] = true
		defer delete(active, visit)
		clone := reflect.MakeSlice(source.Type(), source.Len(), source.Len())
		for i := 0; i < source.Len(); i++ {
			clone.Index(i).Set(cloneValue(source.Index(i), active))
		}
		return clone
	case reflect.Array:
		clone := reflect.New(source.Type()).Elem()
		for i := 0; i < source.Len(); i++ {
			clone.Index(i).Set(cloneValue(source.Index(i), active))
		}
		return clone
	case reflect.Map:
		if source.IsNil() {
			return reflect.Zero(source.Type())
		}
		visit := cloneVisit{typ: source.Type(), ptr: source.Pointer()}
		if active[visit] {
			panic("node metadata contains a reference cycle")
		}
		active[visit] = true
		defer delete(active, visit)
		clone := reflect.MakeMapWithSize(source.Type(), source.Len())
		iter := source.MapRange()
		for iter.Next() {
			clone.SetMapIndex(cloneValue(iter.Key(), active), cloneValue(iter.Value(), active))
		}
		return clone
	case reflect.Struct:
		clone := reflect.New(source.Type()).Elem()
		clone.Set(source)
		for i := 0; i < source.NumField(); i++ {
			if source.Type().Field(i).PkgPath == "" {
				clone.Field(i).Set(cloneValue(source.Field(i), active))
			}
		}
		return clone
	default:
		return source
	}
}

// ScriptBindable — Script 节点把注册节点暴露成脚本函数的判定单一源.
// 绑定层 (services/script.Install) 与补全 RPC (GetScriptBindableKinds) 共用.
// RegionRunner 排除 (脚本有原生循环, 调子图另立项); marker/visual 无执行语义; Script 自身防递归.
func ScriptBindable(rn *RegisteredNode) bool {
	if rn.Spec.IsGraphMarker || rn.Spec.IsVisualOnly || rn.Spec.Kind == "Script" {
		return false
	}
	return rn.Run != nil || rn.Evaluate != nil
}

// ResetRegistryForTest 测试用.
func ResetRegistryForTest() {
	if !testing.Testing() {
		panic("ResetRegistryForTest only callable from tests")
	}
	defaultRegistry.mu.Lock()
	defaultRegistry.frozen = false
	for k := range defaultRegistry.entries {
		delete(defaultRegistry.entries, k)
	}
	defaultRegistry.mu.Unlock()
}
