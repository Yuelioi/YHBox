package hotkey

import (
	"errors"
	"strings"
	"testing"
)

func TestRegistry_RegisterBasicEntry(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)

	err := r.Register("test.foo", HotkeySourceSystem, "测试 foo", nil, "", "", func() {})
	if err != nil {
		t.Fatalf("Register err=%v", err)
	}
	got, ok := r.Get("test.foo")
	if !ok {
		t.Fatal("Get 应找到 test.foo")
	}
	if got.Status != HotkeyStatusUnbound {
		t.Errorf("空 hotkeyStr 时 Status=%q, want unbound", got.Status)
	}
}

func TestRegistry_RegisterRejectsDuplicateKey(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("dup", HotkeySourceSystem, "first", nil, "", "", func() {})
	err := r.Register("dup", HotkeySourceSystem, "second", nil, "", "", func() {})
	if !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("第二次 Register 应返 ErrDuplicateKey, got %v", err)
	}
}

func TestRegistry_ListReturnsAll(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceSystem, "A", nil, "", "", func() {})
	_ = r.Register("b", HotkeySourceAction, "B", nil, "", "", func() {})
	list := r.List()
	if len(list) != 2 {
		t.Errorf("len=%d want 2", len(list))
	}
}

func TestRegistry_DebugDumpFormat(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("x", HotkeySourceSystem, "测试 X", nil, "", "", func() {})
	dump := r.DebugDump()
	if !strings.Contains(dump, "x") || !strings.Contains(dump, "unbound") {
		t.Errorf("DebugDump 应含 key 和 status, got: %s", dump)
	}
}

func TestRegistry_UpdateRejectsReserved(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("x", HotkeySourceAction, "X", nil, "", "", func() {})
	err := r.Update("x", "Ctrl+C")
	var rerr *HotkeyReservedError
	if !errors.As(err, &rerr) {
		t.Errorf("Ctrl+C 应返 ReservedError, got %v", err)
	}
}

func TestRegistry_UpdateRejectsConflict(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	// 用 F1 这种不冲突真 OS 注册的（mock manager 太复杂，跑真 OS register）
	// 实际上 HotkeyManager.Register 会真注册 OS hotkey，单测可能影响系统。
	// 但 plan 接受这风险（已存在的 hotkey_test.go 也是真 OS 注册）。
	// 给两个 entry 同个值看是否撞内部 normalized
	_ = r.Register("a", HotkeySourceAction, "A", nil, "Ctrl+Shift+Alt+F1", "", func() {})
	_ = r.Register("b", HotkeySourceAction, "B", nil, "", "", func() {})
	err := r.Update("b", "Ctrl+Shift+Alt+F1")
	var cerr *HotkeyConflictError
	if !errors.As(err, &cerr) {
		t.Errorf("撞 a 应返 ConflictError, got %v", err)
	}
	// cleanup
	_ = r.Unregister("a")
	_ = r.Unregister("b")
}

func TestRegistry_UpdateSelfNoConflict(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceAction, "A", nil, "Ctrl+Shift+Alt+F2", "", func() {})
	// 改自己当前值 noop
	if err := r.Update("a", "Ctrl+Shift+Alt+F2"); err != nil {
		t.Errorf("改自己当前值应返 nil, got %v", err)
	}
	_ = r.Unregister("a")
}

func TestRegistry_UpdateNormalizationDetectsConflict(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceAction, "A", nil, "Ctrl+Shift+Alt+F3", "", func() {})
	_ = r.Register("b", HotkeySourceAction, "B", nil, "", "", func() {})
	err := r.Update("b", "Shift+Ctrl+Alt+F3") // 顺序不同同义
	var cerr *HotkeyConflictError
	if !errors.As(err, &cerr) {
		t.Errorf("Shift+Ctrl+Alt+F3 撞 Ctrl+Shift+Alt+F3 应识别为冲突, got %v", err)
	}
	_ = r.Unregister("a")
	_ = r.Unregister("b")
}

func TestRegistry_ClearViaEmptyString(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceAction, "A", nil, "Ctrl+Shift+Alt+F4", "", func() {})
	if err := r.Update("a", ""); err != nil {
		t.Fatal(err)
	}
	got, _ := r.Get("a")
	if got.Status != HotkeyStatusUnbound {
		t.Errorf("Clear 后 Status=%q want unbound", got.Status)
	}
	if got.HotkeyStr != "" {
		t.Errorf("Clear 后 HotkeyStr=%q want ''", got.HotkeyStr)
	}
}

func TestRegistry_UnregisterRemovesEntry(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceAction, "A", nil, "Ctrl+Shift+Alt+F5", "", func() {})
	if err := r.Unregister("a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Get("a"); ok {
		t.Error("Unregister 后 Get 应返 false")
	}
}

func TestRegistry_OnActionHotkeyChangeCallback(t *testing.T) {
	mgr := NewHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	var gotID, gotStr string
	r.SetCallbacks(
		func(id, str string) error { gotID, gotStr = id, str; return nil },
		nil, nil,
	)
	_ = r.Register("action.abc", HotkeySourceAction, "A", nil, "", "", func() {})
	if err := r.Update("action.abc", "Ctrl+Shift+Alt+F6"); err != nil {
		t.Fatal(err)
	}
	if gotID != "abc" || gotStr != "Ctrl+Shift+Alt+F6" {
		t.Errorf("callback got (%q, %q), want (abc, Ctrl+Shift+Alt+F6)", gotID, gotStr)
	}
	_ = r.Unregister("action.abc")
}
