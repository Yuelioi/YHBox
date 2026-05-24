// internal/node/registry.go
package node

import (
	"fmt"
	"sync"
	"testing"
)

// Node 节点必须实现的最小接口.
// Spec() 返 metadata, framework 用来注册 + 给 FE inspector.
// Run() 在 interfaces.go (Task 1.1) 加 — Task 0.2 还不接 Run.
type Node interface {
	Spec() Spec
}

// RegisteredNode framework 注册条目, 含 Spec + 探测出的扩展 interface (Task 1.1+ 加).
type RegisteredNode struct {
	Impl Node
	Spec Spec
	// Phase 1 加: Run / Display / Validate / Dependencies / RunRegion 函数指针 (type assertion 探测)
}

var (
	registryMu     sync.RWMutex
	globalRegistry = map[string]*RegisteredNode{}
	registryFrozen bool
)

// Register 仅在 init() / Wails OnStartup 前调.
// Frozen 后调或 dup Kind → panic (fail fast, author bug).
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
	globalRegistry[spec.Kind] = &RegisteredNode{
		Impl: impl,
		Spec: spec,
	}
}

// Freeze 在 Wails OnStartup hook 末尾调一次. 之后 Register panic.
// main.go (Task 0.6+) 负责调用时机.
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

// All 返 snapshot copy (slice header 安全, 内部 *RegisteredNode 引用同对象但 Spec 不可变).
func All() []*RegisteredNode {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*RegisteredNode, 0, len(globalRegistry))
	for _, rn := range globalRegistry {
		out = append(out, rn)
	}
	return out
}

// ResetRegistryForTest 测试用. 生产代码调 panic (testing.Testing() = false).
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
