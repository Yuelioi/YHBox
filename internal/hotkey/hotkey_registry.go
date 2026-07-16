package hotkey

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// HotkeyRegistry 是所有热键的中央 manifest。
//
// 设计目标：
//   - HotkeyManager 只管 "把 mods/vk 注册到 OS"；Registry 管 "业务语义 + UI 序列化 + 持久化回调"
//   - 用 key（稳定字符串标识，如 "battle.toggle" / "action.<uuid>"）作主键
//   - Source 区分 system/action/schedule/editor/recording，给前端 UI 分组用
//   - 持久化回调 onActionHotkeyChange / onSystemHotkeyChange 由 main.go 注入，
//     registry 自己不知道 ActionService / config 怎么存
//
// Invariants：
//   1. normalized != ""  ⇔  status == active || status == failed
//   2. status == unbound  ⇒  bindingID == 0 && hotkeyStr == "" && normalized == ""

// HotkeySource 标记 entry 归属，给 UI 分组。
type HotkeySource string

const (
	HotkeySourceSystem    HotkeySource = "system"
	HotkeySourceAction    HotkeySource = "action"
	HotkeySourceSchedule  HotkeySource = "schedule"
	HotkeySourceEditor    HotkeySource = "editor"
	HotkeySourceRecording HotkeySource = "recording"
)

// HotkeyStatus runtime 状态。
type HotkeyStatus string

const (
	HotkeyStatusActive  HotkeyStatus = "active"  // 已绑定 + OS 注册成功
	HotkeyStatusUnbound HotkeyStatus = "unbound" // hotkeyStr=""
	HotkeyStatusFailed  HotkeyStatus = "failed"  // OS 注册失败
)

// HotkeyMechanism 标记 entry 用哪种底层机制绑定热键。决定 updateLocked/Resume
// 是否调 OS RegisterHotKey, 也给 FE 文案 (全局 vs 仅应用内)。
type HotkeyMechanism string

const (
	// HotkeyMechanismOSGlobal: Win32 RegisterHotKey 全局热键 (system/schedule)。
	HotkeyMechanismOSGlobal HotkeyMechanism = "os-global"
	// HotkeyMechanismEditorInApp: webview only, 不占 OS (editor in-app key)。
	HotkeyMechanismEditorInApp HotkeyMechanism = "editor-inapp"
	// HotkeyMechanismLLHook: 全局 LL-hook 拦截 (录制热键), 不走 OS RegisterHotKey —
	// 游戏 reserve OS 热键, 必须底层 hook 拦截。registry 只存值 + 做可见性/冲突/rebind,
	// 消费方 (wire_recording adapter) 录制 Start 时读 HotkeyStr。
	HotkeyMechanismLLHook HotkeyMechanism = "ll-hook"
)

// HotkeyEntry 给前端 RPC 序列化用。Key 是稳定标识。
// 注：Normalized 不暴露 — 前端不该依赖 canonicalization 规则。
//
// Label 语义: i18n key string (FE t() 渲染), 不是字面值. 动态部分 (计划名)
// 走 LabelParams 插值, FE 调 t(label, labelParams) 输出最终文案. 未找到 key 时
// vue-i18n fallback 直接返 raw key 字符串 (诊断用).
type HotkeyEntry struct {
	Key            string            `json:"key"`
	Source         HotkeySource      `json:"source"`
	Label          string            `json:"label"`
	LabelParams    map[string]string `json:"labelParams,omitempty"`
	HotkeyStr      string            `json:"hotkeyStr"`
	Status         HotkeyStatus      `json:"status"`
	LastError      string            `json:"lastError"`
	ReadonlyReason string            `json:"readonlyReason"`
	Mechanism      HotkeyMechanism   `json:"mechanism"`
}

// registryEntry registry 内部状态。normalized 字段不暴露。
type registryEntry struct {
	spec       HotkeyEntry
	normalized string
	bindingID  int
	onFire     func()
}

// HotkeyRegistry 所有热键的中央 manifest。线程安全。
type HotkeyRegistry struct {
	mu      sync.RWMutex
	manager *HotkeyManager
	entries map[string]*registryEntry
	paused  bool // Pause 暂停所有 OS hotkey 时为 true

	// 持久化与 presentation callbacks 在构造时固定。
	onActionHotkeyChange func(actionID, newStr string) error
	onSystemHotkeyChange func(key, newStr string) error
	emitChanged          func()
	closed               atomic.Bool
	shutdownOnce         sync.Once
	shutdownDone         chan struct{}
	shutdownErr          error
}

type Callbacks struct {
	OnActionChange func(actionID, newStr string) error
	OnSystemChange func(key, newStr string) error
	EmitChanged    func()
}

// NewHotkeyRegistry 构造。
func NewHotkeyRegistry(mgr *HotkeyManager) *HotkeyRegistry {
	return NewHotkeyRegistryWithCallbacks(mgr, Callbacks{})
}

func NewHotkeyRegistryWithCallbacks(mgr *HotkeyManager, callbacks Callbacks) *HotkeyRegistry {
	return &HotkeyRegistry{
		manager: mgr, entries: map[string]*registryEntry{}, shutdownDone: make(chan struct{}),
		onActionHotkeyChange: callbacks.OnActionChange, onSystemHotkeyChange: callbacks.OnSystemChange,
		emitChanged: callbacks.EmitChanged,
	}
}

var ErrRegistryClosed = errors.New("hotkey registry is closed")

// Register 注册新 entry。
//   - duplicate key → ErrDuplicateKey
//   - hotkeyStr != "" 时，invalid/reserved/conflict 前置错误 → 删除 entry 返 err
//   - OS 注册失败：entry.Status=failed，LastError 填，return nil（startup-friendly）
//
// startup 期的语义：用户从 config 加载一堆 hotkeys，某个被别的进程占用不该让整个 startup 挂掉。
//
// label: i18n key string (FE t() 渲染), 见 HotkeyEntry doc.
// labelParams: 给 label 的 vue-i18n named interpolation 用 (nil 表示无动态部分).
func (r *HotkeyRegistry) Register(key string, source HotkeySource, label string, labelParams map[string]string, hotkeyStr, readonlyReason string, onFire func()) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed.Load() {
		return ErrRegistryClosed
	}
	if _, exists := r.entries[key]; exists {
		return ErrDuplicateKey
	}
	entry := &registryEntry{
		spec: HotkeyEntry{
			Key:            key,
			Source:         source,
			Label:          label,
			LabelParams:    labelParams,
			HotkeyStr:      "",
			Status:         HotkeyStatusUnbound,
			ReadonlyReason: readonlyReason,
			Mechanism:      HotkeyMechanismOSGlobal,
		},
		onFire: onFire,
	}
	r.entries[key] = entry
	if hotkeyStr != "" {
		err := r.updateLocked(key, hotkeyStr)
		var invErr *HotkeyInvalidError
		var resErr *HotkeyReservedError
		var conErr *HotkeyConflictError
		if errors.As(err, &invErr) || errors.As(err, &resErr) || errors.As(err, &conErr) {
			// 前置错误：删除 entry 返 err
			delete(r.entries, key)
			return err
		}
		// OS 失败已写入 entry.Status=failed / LastError，return nil（registry-level success）
	}
	if r.emitChanged != nil {
		r.emitChanged()
	}
	return nil
}

// Get 按 key 查 entry，返 spec 副本。
func (r *HotkeyRegistry) Get(key string) (HotkeyEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[key]
	if !ok {
		return HotkeyEntry{}, false
	}
	return e.spec, true
}

// List 当前所有 entry 的快照。顺序不保证。
func (r *HotkeyRegistry) List() []HotkeyEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]HotkeyEntry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e.spec)
	}
	return out
}

// DebugDump 给 debug 面板/日志用的人类可读 dump。
func (r *HotkeyRegistry) DebugDump() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var b strings.Builder
	for k, e := range r.entries {
		fmt.Fprintf(&b, "[%s] source=%s label=%s hotkey=%q status=%s bindingID=%d",
			k, e.spec.Source, e.spec.Label, e.spec.HotkeyStr, e.spec.Status, e.bindingID)
		if e.spec.LastError != "" {
			fmt.Fprintf(&b, " lastError=%q", e.spec.LastError)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Update hot reload：清空 / 改 mods+vk。
//
// 流程：
//  1. Invalid/Reserved/Conflict 三种前置错误返 typed error，**不持久化**
//  2. OS Register 失败：entry.Status=failed, LastError 填, 返 nil (不持久化)
//  3. 成功：entry.Status=active/unbound, 调持久化 callback, emit
func (r *HotkeyRegistry) Update(key, newHotkeyStr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed.Load() {
		return ErrRegistryClosed
	}
	entry, ok := r.entries[key]
	if !ok {
		return fmt.Errorf("hotkey registry: key %q not found", key)
	}
	if entry.spec.ReadonlyReason != "" {
		return fmt.Errorf("hotkey %q 锁定不可改: %s", key, entry.spec.ReadonlyReason)
	}
	previousHotkey := entry.spec.HotkeyStr
	if err := r.updateLocked(key, newHotkeyStr); err != nil {
		return err
	}
	// 持久化（只在 entry 改成 active/unbound 时调；failed 不持久化）
	if entry.spec.Status == HotkeyStatusActive || entry.spec.Status == HotkeyStatusUnbound {
		var persistErr error
		switch entry.spec.Source {
		case HotkeySourceAction:
			actionID := strings.TrimPrefix(key, "action.")
			if r.onActionHotkeyChange != nil {
				persistErr = r.onActionHotkeyChange(actionID, entry.spec.HotkeyStr)
			}
		case HotkeySourceSystem, HotkeySourceRecording:
			if r.onSystemHotkeyChange != nil {
				persistErr = r.onSystemHotkeyChange(key, entry.spec.HotkeyStr)
			}
		}
		if persistErr != nil {
			rollbackErr := r.updateLocked(key, previousHotkey)
			if rollbackErr == nil && entry.spec.Status == HotkeyStatusFailed {
				rollbackErr = errors.New(entry.spec.LastError)
			}
			if rollbackErr != nil {
				entry.spec.Status = HotkeyStatusFailed
				entry.spec.LastError = rollbackErr.Error()
				if r.emitChanged != nil {
					r.emitChanged()
				}
			}
			return errors.Join(
				fmt.Errorf("persist hotkey %q: %w", key, persistErr),
				wrapOptionalError("rollback hotkey", rollbackErr),
			)
		}
	}
	if r.emitChanged != nil {
		r.emitChanged()
	}
	return nil
}

func wrapOptionalError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// updateLocked 调用方持锁。
// Invalid/Reserved/Conflict → 返 typed error 不改 entry
// OS 失败 → entry.Status=failed, LastError 填, 返 nil
// 成功 → entry 全部字段更新到 active
// hotkeyStr="" → 解绑 + unbound
func (r *HotkeyRegistry) updateLocked(key, newHotkeyStr string) error {
	entry, ok := r.entries[key]
	if !ok {
		return fmt.Errorf("hotkey registry: key %q not found", key)
	}

	// 清空：Unregister OS binding，置 unbound
	if newHotkeyStr == "" {
		if entry.bindingID != 0 {
			if err := r.manager.Unregister(entry.bindingID); err != nil {
				return fmt.Errorf("unregister hotkey %q: %w", key, err)
			}
			entry.bindingID = 0
		}
		entry.spec.HotkeyStr = ""
		entry.normalized = ""
		entry.spec.Status = HotkeyStatusUnbound
		entry.spec.LastError = ""
		return nil
	}

	// 解析
	mods, vk, err := parseHotkey(newHotkeyStr)
	if err != nil {
		return &HotkeyInvalidError{Hotkey: newHotkeyStr, Reason: err.Error()}
	}
	// Normalize
	normalized, err := NormalizeHotkey(newHotkeyStr)
	if err != nil {
		return &HotkeyInvalidError{Hotkey: newHotkeyStr, Reason: err.Error()}
	}
	// noop
	if normalized == entry.normalized {
		return nil
	}
	// Reserved 检查（system source 走特权通道：reserved 黑名单的初衷是拦用户绑定，
	// 系统自己注册 Ctrl+Shift+R 这种 reserved 键属于预期行为）
	if entry.spec.Source != HotkeySourceSystem {
		if reserved, reason := IsReservedHotkey(newHotkeyStr); reserved {
			return &HotkeyReservedError{Hotkey: newHotkeyStr, Reason: reason}
		}
	}
	// 冲突检查（跨所有 entry，但排除自己）
	for k, other := range r.entries {
		if k == key {
			continue
		}
		if other.normalized == normalized {
			return &HotkeyConflictError{Key: key, ConflictKey: k, Label: other.spec.Label}
		}
	}
	// OS 注册仅 os-global 机制走 — ll-hook (录制) / editor-inapp 不占 OS,
	// 由各自消费方处理 (LL-hook 拦截 / webview keydown)。
	if entry.spec.Mechanism == HotkeyMechanismOSGlobal {
		// 先 unregister 旧的，再 register 新的
		if entry.bindingID != 0 {
			if err := r.manager.Unregister(entry.bindingID); err != nil {
				return fmt.Errorf("unregister previous hotkey %q: %w", key, err)
			}
			entry.bindingID = 0
		}
		newID, err := r.manager.Register(HotkeySpec{
			Mods: mods | winMOD_NOREPEAT,
			VK:   vk,
			Name: newHotkeyStr,
		}, OwnerAction, entry.onFire)
		if err != nil {
			// OS 失败：entry 写入新值（用户意图），Status=failed，**不**调持久化
			entry.spec.HotkeyStr = newHotkeyStr
			entry.normalized = normalized
			entry.spec.Status = HotkeyStatusFailed
			entry.spec.LastError = err.Error()
			return nil // registry-level success
		}
		entry.bindingID = newID
	}
	entry.spec.HotkeyStr = newHotkeyStr
	entry.normalized = normalized
	entry.spec.Status = HotkeyStatusActive
	entry.spec.LastError = ""
	return nil
}

// Unregister 完全删除 entry（per-action 删时调）。
// 不存在的 key → no-op 返 nil。
func (r *HotkeyRegistry) Unregister(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.entries[key]
	if !ok {
		return nil
	}
	if entry.bindingID != 0 {
		if err := r.manager.Unregister(entry.bindingID); err != nil {
			return fmt.Errorf("unregister hotkey %q: %w", key, err)
		}
	}
	delete(r.entries, key)
	if r.emitChanged != nil {
		r.emitChanged()
	}
	return nil
}

// Pause 暂时反注册所有 OS hotkey binding，让 webview 能收到正常 KEYDOWN。
// 用途：前端 HotkeyCaptureInput 进入捕获模式时调 — 不暂停的话用户按已注册的
// 组合（如 Ctrl+Shift+1）会被 Win32 RegisterHotKey 直接派发给原 action，
// webview 根本收不到 keystroke。
//
// entries map 状态保留（spec.Status / Normalized / HotkeyStr 不变），
// 只清 bindingID。Resume 时按 spec.HotkeyStr 重新 register。
// 幂等：已 paused → noop。
func (r *HotkeyRegistry) Pause() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed.Load() {
		return ErrRegistryClosed
	}
	return r.pauseLocked()
}

func (r *HotkeyRegistry) pauseLocked() error {
	var errs []error
	for key, e := range r.entries {
		if e.bindingID != 0 {
			if err := r.manager.Unregister(e.bindingID); err != nil {
				errs = append(errs, fmt.Errorf("pause hotkey %q: %w", key, err))
				continue
			}
			e.bindingID = 0
		}
	}
	// Even a partial pause must enter the paused state so Resume can restore
	// entries whose unregister succeeded. pauseLocked still scans non-zero
	// bindings on every call, allowing Shutdown to retry failures.
	r.paused = true
	return errors.Join(errs...)
}

// Resume Pause 的反操作：按 entries.HotkeyStr 重新 register OS binding。
// 只重新注册原本是 active 的 entry；unbound/failed 不动。
// 重新注册失败的 entry 标 status=failed + lastError 填（不影响其它）。
// 幂等：未 paused → noop。
func (r *HotkeyRegistry) Resume() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed.Load() {
		return ErrRegistryClosed
	}
	if !r.paused {
		return nil
	}
	r.paused = false
	var hasFailures bool
	for _, e := range r.entries {
		// 只 os-global 机制原本占 OS binding, 才需要 resume 重注册。
		// ll-hook (录制) / editor-inapp 不占 OS, bindingID 恒 0, 跳过 — 否则会被误 OS 抢键。
		if e.spec.Mechanism != HotkeyMechanismOSGlobal {
			continue
		}
		if e.bindingID != 0 {
			continue
		}
		if e.spec.Status != HotkeyStatusActive || e.spec.HotkeyStr == "" {
			continue
		}
		mods, vk, err := parseHotkey(e.spec.HotkeyStr)
		if err != nil {
			e.spec.Status = HotkeyStatusFailed
			e.spec.LastError = err.Error()
			hasFailures = true
			continue
		}
		newID, err := r.manager.Register(HotkeySpec{
			Mods: mods | winMOD_NOREPEAT,
			VK:   vk,
			Name: e.spec.HotkeyStr,
		}, OwnerAction, e.onFire)
		if err != nil {
			e.spec.Status = HotkeyStatusFailed
			e.spec.LastError = err.Error()
			hasFailures = true
			continue
		}
		e.bindingID = newID
	}
	// 只在有失败时 emit changed — 正常路径前端无感
	if hasFailures && r.emitChanged != nil {
		r.emitChanged()
	}
	return nil
}

// RegisterEditor 注册 editor in-app key (webview only, 不占 OS).
//
// 跟 Register 区别:
//   - source 固定 HotkeySourceEditor, Mechanism=editor-inapp (跳 r.manager.Register)
//   - 不接 onFire — editor key 派发由 FE keydown 处理, registry 只挂可见性 + 冲突检查
//   - conflict / parse / reserved 失败时不 delete entry: 保留 + Status=failed + LastError 填.
//     FE 表里能看见 (SettingsHotkeys.vue lastError 渲染). Register 路径 delete + return err
//     不适合 editor 自动注册场景 (用户视角 "Ctrl+K 撞了" 没线索).
func (r *HotkeyRegistry) RegisterEditor(key, label, hotkeyStr, readonlyReason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed.Load() {
		return ErrRegistryClosed
	}
	if _, exists := r.entries[key]; exists {
		return ErrDuplicateKey
	}
	entry := &registryEntry{
		spec: HotkeyEntry{
			Key:            key,
			Source:         HotkeySourceEditor,
			Label:          label,
			HotkeyStr:      hotkeyStr,
			Status:         HotkeyStatusActive,
			ReadonlyReason: readonlyReason,
			Mechanism:      HotkeyMechanismEditorInApp,
		},
	}
	failWith := func(reason string) {
		entry.spec.Status = HotkeyStatusFailed
		entry.spec.LastError = reason
		r.entries[key] = entry
		if r.emitChanged != nil {
			r.emitChanged()
		}
	}
	if _, _, err := parseHotkey(hotkeyStr); err != nil {
		failWith(err.Error())
		return nil
	}
	normalized, err := NormalizeHotkey(hotkeyStr)
	if err != nil {
		failWith(err.Error())
		return nil
	}
	if reserved, reason := IsReservedHotkey(hotkeyStr); reserved {
		failWith("reserved: " + reason)
		return nil
	}
	for k, other := range r.entries {
		if other.normalized == normalized {
			failWith(fmt.Sprintf("跟 %q 冲突", k))
			return nil
		}
	}
	entry.normalized = normalized
	r.entries[key] = entry
	if r.emitChanged != nil {
		r.emitChanged()
	}
	return nil
}

// RegisterLLHook 注册 LL-hook 全局拦截热键 (录制 stop/pause)。
//
// 跟 Register 区别:
//   - Mechanism = ll-hook: updateLocked 不调 r.manager.Register (不占 OS)。
//     游戏 reserve OS 热键, 录制必须底层 hook 拦截; registry 只存值 + 可见性 + 冲突 + rebind。
//   - 无 onFire: LL-hook 派发不走 registry callback, 而是 wire_recording adapter
//     在录制 Start 时读 entry.HotkeyStr 解析成 VK 喂给 hook。
//   - 可 rebind (readonlyReason="" 时): Update 走 updateLocked, 机制 guard 跳 OS。
//
// startup-friendly: parse/reserved/conflict 前置错误 → 删 entry 返 err (同 Register)。
func (r *HotkeyRegistry) RegisterLLHook(key string, source HotkeySource, label, hotkeyStr, readonlyReason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed.Load() {
		return ErrRegistryClosed
	}
	if _, exists := r.entries[key]; exists {
		return ErrDuplicateKey
	}
	entry := &registryEntry{
		spec: HotkeyEntry{
			Key:            key,
			Source:         source,
			Label:          label,
			HotkeyStr:      "",
			Status:         HotkeyStatusUnbound,
			ReadonlyReason: readonlyReason,
			Mechanism:      HotkeyMechanismLLHook,
		},
	}
	r.entries[key] = entry
	if hotkeyStr != "" {
		err := r.updateLocked(key, hotkeyStr)
		var invErr *HotkeyInvalidError
		var resErr *HotkeyReservedError
		var conErr *HotkeyConflictError
		if errors.As(err, &invErr) || errors.As(err, &resErr) || errors.As(err, &conErr) {
			delete(r.entries, key)
			return err
		}
	}
	if r.emitChanged != nil {
		r.emitChanged()
	}
	return nil
}

// Shutdown permanently rejects new registrations and removes every OS-global
// binding. Callers may stop waiting on ctx while cleanup continues once.
func (r *HotkeyRegistry) Shutdown(ctx context.Context) error {
	r.shutdownOnce.Do(func() {
		r.closed.Store(true)
		go func() {
			r.mu.Lock()
			err := r.pauseLocked()
			r.shutdownErr = err
			r.mu.Unlock()
			close(r.shutdownDone)
		}()
	})
	select {
	case <-r.shutdownDone:
		r.mu.RLock()
		defer r.mu.RUnlock()
		return r.shutdownErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SyncLabel 修改某 entry 的展示 label（ActionService rename 时调）。
// 公开方法 — actions 包的 RegistryAdapter 需要调。
func (r *HotkeyRegistry) SyncLabel(key, newLabel string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[key]; ok {
		e.spec.Label = newLabel
	}
	if r.emitChanged != nil {
		r.emitChanged()
	}
}
