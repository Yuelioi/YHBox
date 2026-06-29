// Package execution 持单 worker / 全局 FIFO 队列 / 键鼠互斥 bus。
//
// 关键约束：
//   - 全局只 1 个 Worker 在跑（键鼠独占的保证根）
//   - InputBus 只锁"真正发送 OS input 的瞬间"，不锁 sleep/wait
//   - 取消统一走 context.Context
package execution

import "sync"

// InputBus 包住 OS 键鼠发送。所有"会真敲键鼠"的节点（ClickAt/KeyPress/...）调
// Lock() → 单次操作 → Unlock()。Sleep / WaitTemplate / SetVar 不走 bus。
//
// 单实例 process-wide：v1 single-worker 模型下永远只一个 ContainerRunner 在跑，
// 但 EventTick 子图 + ActionRunner per-step 输入会跟主图交错；bus 保证任意时刻
// 至多 1 个 input call 在飞。
type InputBus struct {
	mu sync.Mutex
}

func NewInputBus() *InputBus { return &InputBus{} }

func (b *InputBus) Lock()   { b.mu.Lock() }
func (b *InputBus) Unlock() { b.mu.Unlock() }
