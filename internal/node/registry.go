// internal/node/registry.go
package node

import (
	"fmt"
	"sync"
	"testing"
)

// RegisteredNode framework 注册条目, 含 Spec + 探测出的扩展 interface fn 指针.
type RegisteredNode struct {
	Impl Node
	Spec Spec
	// Optional extensions — nil if node doesn't implement.
	Display      func(in Inputs, exitName string, out OutputData) string
	Validate     func(in Inputs) []ValidationError
	Dependencies func(in Inputs) []Dependency
	// RunRegion — Phase 5 control flow nodes 实现.
	RunRegion func(ctx Ctx, in Inputs, body func(Ctx) error) (Outputs, error)
	// Evaluate — pure-data 节点 (IsPureData=true) 实现, EvaluatePureData 调.
	Evaluate func(ctx Ctx, in Inputs) (any, error)
	// Defaults — Spec.Inputs 中非 nil Default 值的 name→value 缓存. P1.9: per-token 不再
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
	if d, ok := impl.(Displayer); ok {
		rn.Display = d.Display
	}
	if v, ok := impl.(Validator); ok {
		rn.Validate = v.Validate
	}
	if d, ok := impl.(Dependencer); ok {
		rn.Dependencies = d.Dependencies
	}
	if r, ok := impl.(RegionRunner); ok {
		rn.RunRegion = r.RunRegion
	}
	if e, ok := impl.(Evaluator); ok {
		rn.Evaluate = e.Evaluate
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
