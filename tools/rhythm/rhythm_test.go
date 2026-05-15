package rhythm

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// Mock keyer 替代真实 pkg/input
type mockKeyer struct {
	downs atomic.Int64
	ups   atomic.Int64
}

func (m *mockKeyer) KeyDown(key string) bool { m.downs.Add(1); return true }
func (m *mockKeyer) KeyUp(key string) bool   { m.ups.Add(1); return true }

// 触发 100 次：worker channel cap=4 + non-blocking send → 大量 drop。
// sleep 短到 worker 来不及消费大部分 backlog，验证 trigger 不阻塞主循环。
func TestPressWorker_NonBlockingDropsBacklog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &mockKeyer{}
	w := newPressWorker("d", m)
	w.start(ctx)

	for i := 0; i < 100; i++ {
		w.trigger()
	}
	// 10ms 内 worker 最多消费 ~2 次 (KeyHoldMs=5ms 一次)
	time.Sleep(10 * time.Millisecond)

	// cap=4 + 可能正在处理 1 = 最多 5 次落地按键。100 - 5 = 95 个被 drop。
	if got := m.downs.Load(); got > 6 {
		t.Errorf("100 triggers should drop most (≤6 actual KeyDowns in 10ms), got %d", got)
	}
}

// ctx cancel 在 KeyDown 后 / KeyUp 前 → 必须 KeyUp，防卡键
func TestPressWorker_KeyUpOnCtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := &mockKeyer{}
	w := newPressWorker("d", m)
	w.start(ctx)
	w.trigger()

	// 等 KeyDown 真发出再 cancel；否则 cancel 太早会跳过 KeyDown 路径，测试退化成 0==0 平凡通过
	deadline := time.Now().Add(200 * time.Millisecond)
	for m.downs.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(1 * time.Millisecond)
	}
	if m.downs.Load() == 0 {
		t.Fatalf("worker never called KeyDown within 200ms — scheduler too slow or worker broken")
	}

	cancel()
	time.Sleep(50 * time.Millisecond) // 让 worker 退出

	if d, u := m.downs.Load(), m.ups.Load(); d != u {
		t.Errorf("KeyDown=%d KeyUp=%d should match (no stuck key on ctx cancel)", d, u)
	}
}
