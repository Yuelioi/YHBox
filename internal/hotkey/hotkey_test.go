package hotkey

import (
	"testing"
)

// hotkey 测试使用不常见组合（Ctrl+Shift+Alt+F12 ~ F9 范围），尽量避开实用快捷键。
// 真注册到 Win32 —— 在 CI/无桌面环境跑会失败；本地 windows 桌面 OK。

const (
	winVK_F9  = 0x78
	winVK_F10 = 0x79
	winVK_F11 = 0x7A
	winVK_F12 = 0x7B
)

func testSpec(vk uint32, name string) HotkeySpec {
	return HotkeySpec{
		Mods: winMOD_CONTROL | winMOD_SHIFT | winMOD_ALT | winMOD_NOREPEAT,
		VK:   vk,
		Name: name,
	}
}

func TestHotkeyManager_RegisterUnregister(t *testing.T) {
	m := NewHotkeyManager()
	id, err := m.Register(testSpec(winVK_F12, "Ctrl+Shift+Alt+F12"), OwnerAction, func() {})
	if err != nil {
		t.Fatalf("Register 失败: %v", err)
	}
	if id == 0 {
		t.Fatalf("Register 返回 id=0，期望 manager 分配的非零 id")
	}
	if !m.HasOwner(OwnerAction) {
		t.Fatalf("HasOwner(OwnerAction) 应为 true")
	}
	if got := m.Bindings(); len(got) != 1 || got[0].ID != id || got[0].Owner != OwnerAction {
		t.Fatalf("Bindings 不符: %+v", got)
	}

	if err := m.Unregister(id); err != nil {
		t.Fatalf("Unregister 失败: %v", err)
	}
	if got := m.Bindings(); len(got) != 0 {
		t.Fatalf("Unregister 后 Bindings 应为空，got: %+v", got)
	}
	// 不存在 id → no-op
	if err := m.Unregister(id); err != nil {
		t.Fatalf("Unregister 不存在 id 应为 no-op: %v", err)
	}
}

func TestHotkeyManager_Replace(t *testing.T) {
	m := NewHotkeyManager()
	hits := make(chan string, 4)
	id, err := m.Register(testSpec(winVK_F12, "F12"), OwnerAction, func() { hits <- "old" })
	if err != nil {
		t.Fatalf("Register 失败: %v", err)
	}
	if err := m.Replace(id, testSpec(winVK_F11, "F11"), func() { hits <- "new" }); err != nil {
		t.Fatalf("Replace 失败: %v", err)
	}
	binds := m.Bindings()
	if len(binds) != 1 || binds[0].ID != id || binds[0].Spec.VK != winVK_F11 {
		t.Fatalf("Replace 后 binding 不符: %+v", binds)
	}
	// 替换不存在 id → error
	if err := m.Replace(99999, testSpec(winVK_F10, "F10"), func() {}); err == nil {
		t.Fatalf("Replace 不存在 id 应返回 error")
	}
	_ = m.Unregister(id)
}

func TestHotkeyManager_StartCoexistWithRegister(t *testing.T) {
	// 验证 Start (OwnerBattle) 和 Register (OwnerAction) 能共存，
	// Stop 后 OwnerAction binding 仍保留。
	m := NewHotkeyManager()

	actionID, err := m.Register(testSpec(winVK_F12, "F12"), OwnerAction, func() {})
	if err != nil {
		t.Fatalf("Register action 失败: %v", err)
	}

	battleSpecs := []HotkeySpec{
		{ID: 1, Mods: winMOD_CONTROL | winMOD_SHIFT | winMOD_ALT | winMOD_NOREPEAT, VK: winVK_F9, Name: "battle1"},
		{ID: 2, Mods: winMOD_CONTROL | winMOD_SHIFT | winMOD_ALT | winMOD_NOREPEAT, VK: winVK_F10, Name: "battle2"},
	}
	if err := m.Start(battleSpecs, func(id int) {}); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	if !m.IsRunning() {
		t.Fatalf("Start 后 IsRunning 应为 true")
	}
	if !m.HasOwner(OwnerBattle) || !m.HasOwner(OwnerAction) {
		t.Fatalf("两个 owner 都应有 binding")
	}
	if got := len(m.Bindings()); got != 3 {
		t.Fatalf("应有 3 个 binding，got %d", got)
	}

	// Stop 只清 OwnerBattle
	m.Stop()
	if m.IsRunning() {
		t.Fatalf("Stop 后 IsRunning(=OwnerBattle exists) 应为 false")
	}
	if !m.HasOwner(OwnerAction) {
		t.Fatalf("Stop 后 OwnerAction binding 应保留")
	}
	if got := len(m.Bindings()); got != 1 {
		t.Fatalf("Stop 后应剩 1 个 action binding，got %d", got)
	}

	_ = m.Unregister(actionID)
}

func TestHotkeyManager_RegisterFailureKeepsExistingLoop(t *testing.T) {
	// 验证 rollback 修复：新 Register 失败时，bindings 回滚 + loop 重启 snapshot specs，
	// 旧 binding 仍可用。
	//
	// 旧 bug：rebuildLocked 失败只回滚 bindings 但不重启 loop —— hotkey 线程死掉，
	// 原本工作的热键全失效。复现路径：用户加了 F9（被 OBS 占用）→ Ctrl+Shift+R 也失效。
	m := NewHotkeyManager()
	id1, err := m.Register(testSpec(winVK_F12, "F12-1"), OwnerAction, func() {})
	if err != nil {
		t.Fatalf("第一次 Register 失败: %v", err)
	}
	if !m.HasOwner(OwnerAction) {
		t.Fatalf("第一次 Register 后应有 OwnerAction binding")
	}

	// 同 mods+vk 再注册 → OS RegisterHotKey 返 ALREADY_REGISTERED → rebuildLocked 失败
	_, err = m.Register(testSpec(winVK_F12, "F12-2"), OwnerAction, func() {})
	if err == nil {
		t.Fatalf("第二次 Register 同 mods+vk 应失败")
	}

	// 失败后：bindings 回滚（只剩 id1），loop 应该被重启
	bindings := m.Bindings()
	if len(bindings) != 1 || bindings[0].ID != id1 {
		t.Fatalf("rollback 后 bindings 应只剩 id1=%d，got %+v", id1, bindings)
	}

	// 核心断言：loop 仍在跑（这是 fix 的核心 —— 之前是 false）
	m.mu.Lock()
	running := m.running
	m.mu.Unlock()
	if !running {
		t.Fatalf("rollback 后 hotkey loop 应仍在跑（snapshot 重启），但 running=false（旧 bug）")
	}

	_ = m.Unregister(id1)
}
