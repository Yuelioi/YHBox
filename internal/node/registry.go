// internal/node/registry.go
package node

import (
	"fmt"
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

var (
	registryMu     sync.RWMutex
	globalRegistry = map[string]*RegisteredNode{}
	registryFrozen bool
)

// Register 仅在 init() / Wails OnStartup 前调.
func Register(impl Node) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if registryFrozen {
		panic("node.Register after Freeze (must call before Wails OnStartup returns)")
	}
	spec := impl.Spec()
	if _, exists := globalRegistry[spec.Kind]; exists {
		panic(fmt.Sprintf("node kind %q already registered", spec.Kind))
	}
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

	globalRegistry[spec.Kind] = rn
}

// Freeze 在 Wails OnStartup 末尾调一次.
func Freeze() {
	registryMu.Lock()
	registryFrozen = true
	registryMu.Unlock()
}

// Get RPC handler 用. RLock 并发安全.
func Get(kind string) (*RegisteredNode, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	rn, ok := globalRegistry[kind]
	return rn, ok
}

// All 返 snapshot copy.
func All() []*RegisteredNode {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*RegisteredNode, 0, len(globalRegistry))
	for _, rn := range globalRegistry {
		out = append(out, rn)
	}
	return out
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
	registryMu.Lock()
	registryFrozen = false
	for k := range globalRegistry {
		delete(globalRegistry, k)
	}
	registryMu.Unlock()
}
