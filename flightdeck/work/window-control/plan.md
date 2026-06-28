# Window 类型 + 窗口控制节点 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 YHFish 加一等 `Window` 数据类型 + GetWindow 产出节点 + 窗口控制节点(WindowState/MoveResizeWindow/CloseWindow), 并让任一 NeedsWindow 节点可选接 `Window` 输入(不连=当前活动窗口, 连了=派发期 per-node 覆盖)。

**Architecture:** 复用现有「只读产出类型(Image)+ 数据流 + 变量」plumbing, 零新框架机制做出 Window 类型与产出/消费。可选窗口输入靠在 `execNodeViaFramework` 单点 wrap: 读 `dataWire["Window"]` → push 一个 rt 覆盖栈 + defer pop, `ActiveHWND()/WindowHandle()` 优先返栈顶, 所有窗口服务自动作用在覆盖窗口, 粘性 `rt.window` 全程不碰。

**Tech Stack:** Go(后端 + Win32 via syscall LazyDLL / lxn-win), Wails3 bindings, Vue3 + vue-i18n(前端只动 i18n + 分组), Task/vitest/go test。

## Global Constraints

- **不写任何兼容 shim**: 项目未发布、现存仅 1 个容器, 结构变更直接改, 不留 deprecated/fallback。
- pin 名 PascalCase; exec-in 必叫 `In`; Number Default 用 `json.Number("...")`(spec consistency 守卫 `TestSpecConsistency_*` / `TestNoPinNameSplit`)。
- 展示文本全在 FE i18n `frontend/src/i18n/{zh,en}.ts` 的 `node.<Kind>.*`; backend 只出结构。改 i18n 后必跑 `cd frontend && pnpm gen:node-i18n`(catalog drift 守卫)。
- `Default` 与 `Required` 互斥。
- 类型色: `Window` = `#22d3ee`(cyan, 现有未占用)。
- 验证基线: 见 `flightdeck/knowledge/build/build.md`(预存失败判红基线: runtime fish fixture 缺失 / i18n residue, 不算回归)。
- 设计单一来源: `flightdeck/work/window-control/design.md`(本计划是它的执行版, 冲突以 design 为准)。

---

## File Structure

**Create:**
- `internal/nodes/window/get_window.go` — GetWindow 节点 + 共享 `windowSelectorInputs()`/`matchSpecFrom()`。
- `internal/nodes/window/window_state.go` — WindowState 节点。
- `internal/nodes/window/move_resize_window.go` — MoveResizeWindow 节点。
- `internal/nodes/window/close_window.go` — CloseWindow 节点。
- `internal/nodes/window/doc.go` — 包说明 + 全部窗口类别节点位置索引。
- `internal/nodes/window/window_nodes_test.go` — 上述节点 StubWindowService 测试。
- `pkg/winutil/control.go` — 窗口控制 Win32 原语 + IsWindow。

**Modify:**
- `internal/node/types.go` — 加 `node.Window` 域类型 + RegisterType。
- `internal/node/inputs.go` + `interfaces.go` — `Inputs.Window(name)` helper。
- `internal/node/spec.go` — 加 `Spec.NeedsForeground` 标志 + `WindowInputSpec()` helper。
- `internal/node/registry.go` — Register 不变式: NeedsWindow ⟹ 含 Window 输入。
- `internal/node/errorcodes.go` — 加 `CodeWindowInvalid`。
- `internal/node/services.go` + `interfaces.go` — WindowService 加控制方法 + stub。
- `internal/services/container/runtime/runtime_context.go` — 覆盖栈 + 借词 + 借 borderless saved-state + ActiveHWND/WindowHandle 读栈顶。
- `internal/services/container/runtime/node_services.go` — windowAdapter 实现控制方法 + Snapshot。
- `internal/services/container/runtime/dispatch_v5.go` — `execNodeViaFramework` 加覆盖 wrap。
- `internal/nodes/system/window_target.go` — Done 加 Window 字段 + 用共享 selector。
- 全部 22 个 NeedsWindow 节点 Spec.Inputs — spread `node.WindowInputSpec()`(逐个加一行)。
- `internal/services/container/runtime/runner.go` + validator — NeedsWindow 校验放宽。
- `internal/nodes/system/window_target.go` / `wait_window.go` / `wait_window_gone.go` / `internal/nodes/input/bring_foreground.go` — `Category` 改 `"Window"`。
- `main.go` + `internal/services/container/runtime/dispatch_v5_test.go` — blank-import `_ "yotta/internal/nodes/window"`。
- `frontend/src/i18n/zh.ts` + `en.ts` — 新节点 + Window 输入 + nodeGroup 文案。
- `frontend/src/.../adapter.ts`(GROUP_MAP) + `NodePalette.vue`(GROUP_LABEL) + `useNodeGroupColor.ts`(GROUP_I18N_KEY) — Window 组。

---

## Phase A — Window 类型 + 产出节点

### Task A1: `Window` 域类型 + `Inputs.Window` helper

**Files:**
- Modify: `internal/node/types.go`
- Modify: `internal/node/interfaces.go`(Inputs 接口加方法), `internal/node/inputs.go`(实现)
- Test: `internal/node/types_test.go`(新建或追加)

**Interfaces:**
- Produces: `type node.Window struct { HWND uintptr; Title, Class, Process string; PID uint32; ClientW, ClientH int }`; `Inputs.Window(name string) (node.Window, bool)`; 类型 tag `"Window"`。

- [ ] **Step 1: 写失败测试**

```go
// internal/node/types_test.go
package node

import "testing"

func TestWindowTypeRegistered(t *testing.T) {
	found := false
	for _, ts := range AllTypes() {
		if ts.Tag == "Window" {
			found = true
			if ts.Color != "#22d3ee" || ts.WidgetKind != "preview" {
				t.Fatalf("Window TypeSpec 错: %+v", ts)
			}
		}
	}
	if !found {
		t.Fatal("Window 类型未注册")
	}
}

func TestInputsWindow(t *testing.T) {
	w := Window{HWND: 123, Title: "记事本"}
	in := newInputsForTest(map[string]any{"W": w}) // 见 Step 3 注: 复用现有测试构造
	got, ok := in.Window("W")
	if !ok || got.HWND != 123 {
		t.Fatalf("Window 取值失败: %v %v", got, ok)
	}
	if _, ok := in.Window("missing"); ok {
		t.Fatal("缺失 pin 应返 false")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/node/ -run 'TestWindowTypeRegistered|TestInputsWindow' -v`
Expected: FAIL(`Window` 未注册 / `in.Window` 未定义)。
注: 若 `newInputsForTest` 不存在, 看 `inputs_test.go` 现有构造法照搬(用 `&inputsImpl{merged: m}`)。

- [ ] **Step 3: 实现**

`internal/node/types.go` — 加域类型 + 注册:

```go
// Window 域类型 — 窗口对象。HWND 是 live 句柄；其余元数据是解析时快照(可能过期，别当实时)。
// 运行期瞬时值(同 Image)，不序列化进 workflow JSON。
type Window struct {
	HWND    uintptr `json:"hwnd"`
	Title   string  `json:"title"`
	Class   string  `json:"class"`
	Process string  `json:"process"`
	PID     uint32  `json:"pid"`
	ClientW int     `json:"clientW"`
	ClientH int     `json:"clientH"`
}
```

在 `init()` 的内置类型 slice 末尾加一行:

```go
		{Tag: "Window", GoType: "node.Window", WidgetKind: "preview", Color: "#22d3ee"},
```

`internal/node/interfaces.go` — Inputs 接口加(挨着 `Point`):

```go
	Window(name string) (Window, bool)
```

`internal/node/inputs.go` — 实现(挨着 `Point`):

```go
func (i *inputsImpl) Window(name string) (Window, bool) {
	if v, ok := i.merged[name].(Window); ok {
		return v, true
	}
	return Window{}, false
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/node/ -run 'TestWindowTypeRegistered|TestInputsWindow' -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/node/types.go internal/node/interfaces.go internal/node/inputs.go internal/node/types_test.go
git commit -m "feat(node): add Window pin type + Inputs.Window helper"
```

---

### Task A2: 共享窗口选择器 + `GetWindow` 节点

**Files:**
- Create: `internal/nodes/window/get_window.go`, `internal/nodes/window/window_nodes_test.go`
- Modify: `main.go`, `internal/services/container/runtime/dispatch_v5_test.go`(blank-import)
- Test: `internal/nodes/window/window_nodes_test.go`

**Interfaces:**
- Consumes: `node.Window`(A1), `winutil.ResolveWindow`/`winutil.MatchSpec`(现有)。
- Produces: `windowSelectorInputs() []node.InputSpec`, `matchSpecFrom(in node.Inputs) winutil.MatchSpec`(window 包内); GetWindow kind, Done 出口带 Data `Window`(Type "Window"), Fail 出口带 `Error`/`Code`。

- [ ] **Step 1: 写失败测试**

```go
// internal/nodes/window/window_nodes_test.go
package window

import (
	"context"
	"testing"

	"yotta/internal/node"
	"yotta/pkg/winutil"
)

func TestGetWindow_ResolvesToDoneWindow(t *testing.T) {
	// 替身: 不连真 Win32, 注入 resolve 结果。
	orig := resolveWindowFn
	defer func() { resolveWindowFn = orig }()
	resolveWindowFn = func(_ context.Context, _ winutil.MatchSpec, _, _ any) (winutil.WindowHandle, error) {
		return winutil.WindowHandle{HWND: 42, Title: "记事本", ClientW: 800, ClientH: 600}, nil
	}
	ctx := node.NewTestCtx() // 见 Step 2 注
	out, err := GetWindow{}.Run(ctx, node.NewInputsForTest(nil))
	if err != nil {
		t.Fatal(err)
	}
	exit, data := node.ExitOf(out)
	if exit != "Done" {
		t.Fatalf("应走 Done, got %s", exit)
	}
	w, _ := data["Window"].(node.Window)
	if w.HWND != 42 || w.Title != "记事本" {
		t.Fatalf("Done.Window 错: %+v", w)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/window/ -run TestGetWindow -v`
Expected: FAIL(包/类型不存在)。
注: `NewTestCtx`/`NewInputsForTest`/`ExitOf` 用现有测试 helper —— 看 `internal/nodes/system/window_target_test.go` 怎么构造 ctx/inputs/读出口, 照搬其模式(若名字不同按其实际名用)。`resolveWindowFn` 的签名要与 `winutil.ResolveWindow` 一致(`func(context.Context, winutil.MatchSpec, time.Duration, time.Duration) (winutil.WindowHandle, error)`), 上面 `any` 占位实际写 `time.Duration`。

- [ ] **Step 3: 实现**

```go
// internal/nodes/window/get_window.go
package window

import (
	"context"
	"errors"
	"time"

	"yotta/internal/node"
	"yotta/pkg/winutil"
)

func init() { node.Register(&GetWindow{}) }

// resolveWindowFn 测试可替换; 默认真 Win32 解析。
var resolveWindowFn = winutil.ResolveWindow

const (
	selInTitle      = "Title"
	selInClass      = "Class"
	selInProcess    = "ProcessName"
	selInTitleMatch = "TitleMatch"
)

// windowSelectorInputs — GetWindow / Win32WindowTarget 共用的窗口匹配输入(防漂移)。
func windowSelectorInputs() []node.InputSpec {
	return []node.InputSpec{
		{Name: selInTitle, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		{Name: selInClass, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		{Name: selInProcess, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		{Name: selInTitleMatch, Type: "String", Default: "exact",
			Widget: node.WidgetSpec{Kind: "dropdown", Props: node.MarshalProps(node.DropdownProps{
				Options: []node.EnumOption{{Value: "exact"}, {Value: "contains"}, {Value: "prefix"}, {Value: "suffix"}, {Value: "regex"}},
			})}},
	}
}

func matchSpecFrom(in node.Inputs) winutil.MatchSpec {
	return winutil.MatchSpec{
		Title: in.String(selInTitle), Class: in.String(selInClass),
		ProcessName: in.String(selInProcess), TitleMatch: in.String(selInTitleMatch),
	}
}

func windowFromHandle(wh winutil.WindowHandle) node.Window {
	return node.Window{HWND: wh.HWND, Title: wh.Title, Class: wh.Class,
		Process: wh.ProcessName, PID: wh.PID, ClientW: wh.ClientW, ClientH: wh.ClientH}
}

// GetWindow 解析匹配窗口为 Window 对象, 不改活动窗口。
type GetWindow struct{}

const (
	gwInExec = "In"
	gwDone   = "Done"
	gwFail   = "Fail"
)

func (GetWindow) Spec() node.Spec {
	return node.Spec{
		Kind:     "GetWindow",
		Category: "Window",
		Inputs:   append([]node.InputSpec{{Name: gwInExec, Type: "Exec"}}, windowSelectorInputs()...),
		Outputs: []node.OutputSpec{
			{Name: gwDone, Type: "Exec", Data: []node.DataField{{Name: "Window", Type: "Window"}}},
			{Name: gwFail, Type: "Exec", Semantic: "error", Data: []node.DataField{
				{Name: "Error", Type: "String"}, {Name: "Code", Type: "String"}}},
		},
	}
}

func (GetWindow) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	wh, err := resolveWindowFn(ctx.Context(), matchSpecFrom(in), 3*time.Second, 500*time.Millisecond)
	if err != nil {
		if errors.Is(err, winutil.ErrWindowNotFound) {
			return nil, node.Failf(node.CodeNotFound, err, "GetWindow: %v", err)
		}
		return nil, err
	}
	return ctx.Out(gwDone).Set("Window", windowFromHandle(wh)).Fire(), nil
}
```

`internal/nodes/window/doc.go`:

```go
// Package window 提供窗口类别节点: GetWindow(解析→Window 对象) + WindowState/MoveResizeWindow/CloseWindow(窗口控制)。
//
// 注: Category=="Window" 的节点不止本包 —— Win32WindowTarget/WaitWindow/WaitWindowGone 在 internal/nodes/system/,
// BringWindowForeground 在 internal/nodes/input/(只是 Category 标 "Window")。找全部窗口节点按 Category 而非包。
package window
```

`main.go` 与 `dispatch_v5_test.go` 的 blank-import 区加:

```go
	_ "yotta/internal/nodes/window"
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/window/ -run TestGetWindow -v && go build ./...`
Expected: PASS + build 绿。

- [ ] **Step 5: 提交**

```bash
git add internal/nodes/window/ main.go internal/services/container/runtime/dispatch_v5_test.go
git commit -m "feat(window): add GetWindow node + shared window selector"
```

---

### Task A3: `Win32WindowTarget` 产出 Window + 用共享 selector

**Files:**
- Modify: `internal/nodes/system/window_target.go`
- Test: `internal/nodes/system/window_target_test.go`

**Interfaces:**
- Consumes: `window.windowSelectorInputs`? —— **不可**(system 不能 import window, 会环)。改: Win32WindowTarget 自己保留现有 selector 输入, **共享口径靠 winutil.MatchSpec**(本就共用)。Done 出口加 Data `Window`。

- [ ] **Step 1: 写失败测试**(在 `window_target_test.go` 追加)

```go
func TestWin32WindowTarget_EmitsWindowOnDone(t *testing.T) {
	orig := resolveWindowFn // window_target 所在包的注入点(见现有测试)
	defer func() { resolveWindowFn = orig }()
	// ... 复用现有 fixture 让 SetActive 成功, 断言 Done.Window.HWND 非 0、Title 对。
}
```
(按 `window_target_test.go` 现有 stub 模式补全; 关键断言: `data["Window"].(node.Window).HWND == 预期`。)

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/system/ -run TestWin32WindowTarget_EmitsWindowOnDone -v`
Expected: FAIL(Done 无 Window 字段)。

- [ ] **Step 3: 实现**(`window_target.go`)

Done 出口加 Data; Run 里 SetActive 成功后把解析到的窗口塞进 Done。`WindowService.SetActive` 当前不回传窗口 —— 加一个读法: SetActive 后 `wh := ctx.Window().Snapshot()`(见 Task B2 加的 Snapshot) 拿当前活动窗口(即刚 set 的)。

Spec Outputs 改:
```go
		Outputs: []node.OutputSpec{
			{Name: wtOutDone, Type: "Exec", Data: []node.DataField{{Name: "Window", Type: "Window"}}},
			{Name: "Fail", Type: "Exec", Semantic: "error", Data: []node.DataField{
				{Name: "Error", Type: "String"}, {Name: "Code", Type: "String"}}},
		},
```
Run 成功分支改:
```go
	w, err := ctx.Window().Snapshot()
	if err != nil {
		return nil, err
	}
	return ctx.Out(wtOutDone).Set("Window", w).Fire(), nil
```
(`ctx.Window().Snapshot()` 返 `node.Window` —— Task B2 加该方法; 本任务依赖 B2 先做, 或先在 stub 返零值让测试用 fixture。**实施顺序: 先 B2 再 A3**, 故把 A3 移到 Phase B 之后执行; 计划编号保留 A3, 执行序见末尾「执行顺序」。)

- [ ] **Step 4 / 5**: 测试通过 → 提交 `feat(window): Win32WindowTarget emits Window on Done`。

---

## Phase B — winutil 原语 + WindowService 控制 + 控制节点

### Task B1: winutil 窗口控制原语

**Files:**
- Create: `pkg/winutil/control.go`
- Test: 无单测(纯 Win32, 靠真窗口; 同 BringToFront 现状)。建一个 `pkg/winutil/control_compile_test.go` 仅保证编译 + 签名稳定(调用一遍传 0 句柄不崩, 不验效果)。

**Interfaces:**
- Produces(全 `pkg/winutil`):
  - `func IsWindow(hwnd uintptr) bool`
  - `func Maximize(hwnd uintptr) error` / `Minimize` / `Restore`
  - `func MoveResize(hwnd uintptr, x, y, w, h int) error`
  - `func CloseWindow(hwnd uintptr) error`
  - `func EnterBorderless(hwnd uintptr) (SavedWindow, error)` —— 返进入前 placement+style 快照
  - `func ExitBorderless(hwnd uintptr, saved SavedWindow) error`
  - `type SavedWindow struct { Style, ExStyle uintptr; Placement []byte; PID uint32 }`(Placement 用 win.WINDOWPLACEMENT 序列化或直接存结构体)

- [ ] **Step 1: 先核 lxn/win 暴露面(不脑补签名)**

Run: `grep -rEn "func (ShowWindow|SetWindowPos|GetWindowLongPtr|SetWindowLongPtr|MonitorFromWindow|GetMonitorInfo|GetWindowPlacement|SetWindowPlacement|IsWindow|PostMessage)\b" $(go env GOMODCACHE)/github.com/lxn/win*/`
按结果决定: 有的用 `win.*`; 缺的按 `window.go` 的 `syscall.NewLazyDLL("user32.dll").NewProc(...)` 模式补 proc。常量 `win.SW_MAXIMIZE/SW_MINIMIZE/SW_RESTORE/WS_CAPTION/WS_THICKFRAME/WS_OVERLAPPEDWINDOW/GWL_STYLE/SWP_*/WM_CLOSE/MONITOR_DEFAULTTONEAREST` 一并核。

- [ ] **Step 2: 实现 control.go**

```go
package winutil

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

var (
	procIsWindow         = user32.NewProc("IsWindow")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procGetWindowLongPtr = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr = user32.NewProc("SetWindowLongPtrW")
	procMonitorFromWindow= user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo   = user32.NewProc("GetMonitorInfoW")
	procGetWindowPlacement = user32.NewProc("GetWindowPlacement")
	procSetWindowPlacement = user32.NewProc("SetWindowPlacement")
	procPostMessage      = user32.NewProc("PostMessageW")
)

const (
	swMaximize = 3
	swMinimize = 6
	gwlStyle   = -16
	wsCaption     = 0x00C00000
	wsThickFrame  = 0x00040000
	wsOverlapped  = 0x00CF0000 // WS_OVERLAPPEDWINDOW
	swpFrameChanged = 0x0020
	swpNoZOrder     = 0x0004
	swpShowWindow   = 0x0040
	wmClose         = 0x0010
	monitorNearest  = 0x00000002
)

func IsWindow(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	r, _, _ := procIsWindow.Call(hwnd)
	return r != 0
}

func Maximize(hwnd uintptr) error { return showErr(hwnd, swMaximize) }
func Minimize(hwnd uintptr) error { return showErr(hwnd, swMinimize) }
func Restore(hwnd uintptr) error  { return showErr(hwnd, swRestore) } // swRestore=9 已在 window.go

func showErr(hwnd uintptr, cmd int) error {
	if hwnd == 0 {
		return fmt.Errorf("hwnd 0")
	}
	procShowWindow.Call(hwnd, uintptr(cmd)) // ShowWindow 返回值是「之前是否可见」, 非成功标志
	return nil
}

func MoveResize(hwnd uintptr, x, y, w, h int) error {
	if hwnd == 0 {
		return fmt.Errorf("hwnd 0")
	}
	r, _, err := procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoZOrder)
	if r == 0 {
		return fmt.Errorf("SetWindowPos: %v", err)
	}
	return nil
}

func CloseWindow(hwnd uintptr) error {
	if hwnd == 0 {
		return fmt.Errorf("hwnd 0")
	}
	procPostMessage.Call(hwnd, wmClose, 0, 0) // 发送即返, 不等关闭
	return nil
}

// SavedWindow — borderless 进入前快照, 供 ExitBorderless 还原。
type SavedWindow struct {
	Style     uintptr
	Placement win.WINDOWPLACEMENT
	PID       uint32
}

func EnterBorderless(hwnd uintptr) (SavedWindow, error) {
	if hwnd == 0 {
		return SavedWindow{}, fmt.Errorf("hwnd 0")
	}
	style, _, _ := procGetWindowLongPtr.Call(hwnd, uintptr(int32(gwlStyle)))
	var wp win.WINDOWPLACEMENT
	wp.Length = uint32(unsafe.Sizeof(wp))
	procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&wp)))
	saved := SavedWindow{Style: style, Placement: wp, PID: getWindowPID(win.HWND(hwnd))}

	// 去标题/边框
	procSetWindowLongPtr.Call(hwnd, uintptr(int32(gwlStyle)), style&^(wsCaption|wsThickFrame))
	// 铺满所在显示器
	mon, _, _ := procMonitorFromWindow.Call(hwnd, monitorNearest)
	var mi win.MONITORINFO
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi)))
	r := mi.RcMonitor
	procSetWindowPos.Call(hwnd, 0, uintptr(r.Left), uintptr(r.Top),
		uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), swpFrameChanged|swpNoZOrder|swpShowWindow)
	return saved, nil
}

func ExitBorderless(hwnd uintptr, saved SavedWindow) error {
	if hwnd == 0 {
		return fmt.Errorf("hwnd 0")
	}
	style := saved.Style
	if style == 0 {
		style = wsOverlapped // 无记录退化
	}
	procSetWindowLongPtr.Call(hwnd, uintptr(int32(gwlStyle)), style)
	if saved.Placement.Length != 0 {
		procSetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&saved.Placement)))
	} else {
		procShowWindow.Call(hwnd, swRestore)
	}
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpFrameChanged|swpNoZOrder|0x0001|0x0002) // SWP_NOMOVE|NOSIZE
	return nil
}

// WindowPID 暴露 saved.PID 校验用(restoreBorders 防 HWND 复用)。
func WindowPID(hwnd uintptr) uint32 { return getWindowPID(win.HWND(hwnd)) }
```

注: `getWindowPID`/`procShowWindow`/`swRestore`/`user32` 已在 `window.go` 定义, 复用; 若 `win.MONITORINFO`/`win.WINDOWPLACEMENT` 字段名与上不符, 按 Step 1 grep 结果改(字段名以 lxn/win 实际为准)。

- [ ] **Step 3: 编译稳定测试**

```go
// pkg/winutil/control_compile_test.go
package winutil

import "testing"

func TestControlPrimitives_ZeroHandleSafe(t *testing.T) {
	if IsWindow(0) { t.Fatal("0 句柄应非窗口") }
	if err := Maximize(0); err == nil { t.Fatal("0 句柄应报错") }
	if err := MoveResize(0, 0, 0, 0, 0); err == nil { t.Fatal("0 句柄应报错") }
	if err := CloseWindow(0); err == nil { t.Fatal("0 句柄应报错") }
	if _, err := EnterBorderless(0); err == nil { t.Fatal("0 句柄应报错") }
}
```

- [ ] **Step 4: 运行**

Run: `go test ./pkg/winutil/ -run TestControlPrimitives -v && go build ./...`
Expected: PASS + build 绿。

- [ ] **Step 5: 提交** `feat(winutil): window control primitives (max/min/restore/move/close/borderless)`

---

### Task B2: WindowService 控制方法 + adapter + stub + Snapshot

**Files:**
- Modify: `internal/node/interfaces.go`(WindowService 接口), `internal/node/services.go`(stub)
- Modify: `internal/services/container/runtime/node_services.go`(windowAdapter)
- Modify: `internal/services/container/runtime/runtime_context.go`(borderless saved-state map)
- Test: `internal/services/container/runtime/node_services_test.go`(追加)

**Interfaces:**
- Produces(WindowService 加):
  ```go
  Maximize() error
  Minimize() error
  Restore() error
  BorderlessFullscreen() error
  RestoreBorders() error
  MoveResize(x, y, w, h int) error
  Close() error
  Snapshot() (Window, error) // 当前活动窗口(含覆盖)的元数据快照, 给 Done.Window / Win32WindowTarget 用
  ```

- [ ] **Step 1: 写失败测试**(node_services_test.go)

```go
func TestWindowAdapter_Snapshot(t *testing.T) {
	rt := newTestRuntimeContext(t) // 复用现有 helper
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 7, Title: "X", ClientW: 100, ClientH: 50})
	a := NewWindowAdapter(rt)
	w, err := a.Snapshot()
	if err != nil || w.HWND != 7 || w.ClientW != 100 {
		t.Fatalf("Snapshot 错: %+v %v", w, err)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/services/container/runtime/ -run TestWindowAdapter_Snapshot -v`
Expected: FAIL(Snapshot 未定义)。

- [ ] **Step 3: 实现**

`runtime_context.go` — rt 加 borderless saved-state(挨着 frameCache):
```go
	borderlessMu    sync.Mutex
	borderlessSaved map[uintptr]winutil.SavedWindow
```
加方法:
```go
func (rt *RuntimeContext) saveBorderless(hwnd uintptr, s winutil.SavedWindow) {
	rt.borderlessMu.Lock(); defer rt.borderlessMu.Unlock()
	if rt.borderlessSaved == nil { rt.borderlessSaved = map[uintptr]winutil.SavedWindow{} }
	rt.borderlessSaved[hwnd] = s
}
func (rt *RuntimeContext) takeBorderless(hwnd uintptr) (winutil.SavedWindow, bool) {
	rt.borderlessMu.Lock(); defer rt.borderlessMu.Unlock()
	s, ok := rt.borderlessSaved[hwnd]
	if ok { delete(rt.borderlessSaved, hwnd) }
	return s, ok
}
```

`interfaces.go` WindowService + `services.go` stub 加全部新方法(stub 全返 nil / zero)。

`node_services.go` windowAdapter:
```go
func (a *windowAdapter) Snapshot() (node.Window, error) {
	wh := a.rt.WindowHandle()
	if wh.HWND == 0 {
		return node.Window{}, ErrNoActiveWindow
	}
	return node.Window{HWND: wh.HWND, Title: wh.Title, Class: wh.Class,
		Process: wh.ProcessName, PID: wh.PID, ClientW: wh.ClientW, ClientH: wh.ClientH}, nil
}
func (a *windowAdapter) Maximize() error { h, err := a.rt.ActiveHWND(); if err != nil { return err }; return winutil.Maximize(h) }
func (a *windowAdapter) Minimize() error { h, err := a.rt.ActiveHWND(); if err != nil { return err }; return winutil.Minimize(h) }
func (a *windowAdapter) Restore() error  { h, err := a.rt.ActiveHWND(); if err != nil { return err }; return winutil.Restore(h) }
func (a *windowAdapter) MoveResize(x, y, w, h int) error {
	hwnd, err := a.rt.ActiveHWND(); if err != nil { return err }
	return winutil.MoveResize(hwnd, x, y, w, h)
}
func (a *windowAdapter) Close() error { h, err := a.rt.ActiveHWND(); if err != nil { return err }; return winutil.CloseWindow(h) }
func (a *windowAdapter) BorderlessFullscreen() error {
	h, err := a.rt.ActiveHWND(); if err != nil { return err }
	saved, err := winutil.EnterBorderless(h); if err != nil { return err }
	a.rt.saveBorderless(h, saved)
	return nil
}
func (a *windowAdapter) RestoreBorders() error {
	h, err := a.rt.ActiveHWND(); if err != nil { return err }
	saved, ok := a.rt.takeBorderless(h)
	if ok && winutil.WindowPID(h) != saved.PID {
		ok = false // HWND 复用了别的进程, 退化普通 restore
	}
	return winutil.ExitBorderless(h, pick(ok, saved, winutil.SavedWindow{}))
}
```
(`pick` 一个 2 行三元 helper 或直接 if; ExitBorderless 对零值 SavedWindow 退化处理已在 B1。)

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/services/container/runtime/ -run TestWindowAdapter_Snapshot -v && go build ./...`
Expected: PASS + build 绿(stub 也实现了新方法)。

- [ ] **Step 5: 提交** `feat(runtime): WindowService control methods + Snapshot + borderless saved-state`

---

### Task B3: `WindowState` 节点

**Files:**
- Create: `internal/nodes/window/window_state.go`
- Test: `internal/nodes/window/window_nodes_test.go`(追加)

**Interfaces:**
- Consumes: `ctx.Window().Maximize/Minimize/Restore/BorderlessFullscreen/RestoreBorders + Snapshot`(B2)。
- Produces: WindowState kind, `State` dropdown, Done 带 `Window`(透传重读)。

- [ ] **Step 1: 写失败测试**

```go
func TestWindowState_Maximize_FiresDoneWithFreshWindow(t *testing.T) {
	rec := &recordingWindowService{snap: node.Window{HWND: 9, ClientW: 1920, ClientH: 1080}}
	ctx := node.NewTestCtxWithWindow(rec) // 注入自定义 WindowService(看现有 test helper)
	out, err := WindowState{}.Run(ctx, node.NewInputsForTest(map[string]any{"State": "maximize"}))
	if err != nil { t.Fatal(err) }
	if !rec.maximized { t.Fatal("应调 Maximize") }
	exit, data := node.ExitOf(out)
	if exit != "Done" || data["Window"].(node.Window).ClientW != 1920 {
		t.Fatalf("Done.Window 应是操作后重读: %v %+v", exit, data["Window"])
	}
}
```
(`recordingWindowService` 实现 WindowService, 记录调了哪个方法、Snapshot 返预设值。)

- [ ] **Step 2: 运行确认失败** → `go test ./internal/nodes/window/ -run TestWindowState -v` → FAIL。

- [ ] **Step 3: 实现**

```go
// internal/nodes/window/window_state.go
package window

import "yotta/internal/node"

func init() { node.Register(&WindowState{}) }

type WindowState struct{}

const (
	wsInExec  = "In"
	wsInState = "State"
	wsDone    = "Done"
)

func (WindowState) Spec() node.Spec {
	return node.Spec{
		Kind: "WindowState", Category: "Window", NeedsWindow: true,
		Inputs: append([]node.InputSpec{
			{Name: wsInExec, Type: "Exec"},
			{Name: wsInState, Type: "String", Default: "maximize",
				Widget: node.WidgetSpec{Kind: "dropdown", Props: node.MarshalProps(node.DropdownProps{
					Options: []node.EnumOption{
						{Value: "maximize"}, {Value: "minimize"}, {Value: "restore"},
						{Value: "borderlessFullscreen"}, {Value: "restoreBorders"},
					}})}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: wsDone, Type: "Exec", Data: []node.DataField{{Name: "Window", Type: "Window"}}},
		},
	}
}

func (WindowState) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	var err error
	switch in.String(wsInState) {
	case "maximize":
		err = ctx.Window().Maximize()
	case "minimize":
		err = ctx.Window().Minimize()
	case "restore":
		err = ctx.Window().Restore()
	case "borderlessFullscreen":
		err = ctx.Window().BorderlessFullscreen()
	case "restoreBorders":
		err = ctx.Window().RestoreBorders()
	default:
		return nil, node.Failf(node.CodeError, nil, "WindowState: 未知 State %q", in.String(wsInState))
	}
	if err != nil {
		return nil, err
	}
	w, err := ctx.Window().Snapshot()
	if err != nil {
		return nil, err
	}
	return ctx.Out(wsDone).Set("Window", w).Fire(), nil
}
```

- [ ] **Step 4: 运行确认通过** → PASS + `go build ./...`。

- [ ] **Step 5: 提交** `feat(window): WindowState node`

---

### Task B4: `MoveResizeWindow` 节点

**Files:** Create `internal/nodes/window/move_resize_window.go`; Test 追加。

- [ ] **Step 1: 失败测试**: 注入 recordingWindowService, 断言 `MoveResize(100,200,800,600)` 被调 + Done.Window 重读。
- [ ] **Step 2: 运行确认失败**。
- [ ] **Step 3: 实现**

```go
// internal/nodes/window/move_resize_window.go
package window

import (
	"encoding/json"
	"yotta/internal/node"
)

func init() { node.Register(&MoveResizeWindow{}) }

type MoveResizeWindow struct{}

const (
	mrInExec = "In"
	mrInX    = "X"
	mrInY    = "Y"
	mrInW    = "Width"
	mrInH    = "Height"
	mrDone   = "Done"
)

func (MoveResizeWindow) Spec() node.Spec {
	num := func(name string) node.InputSpec {
		return node.InputSpec{Name: name, Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}}
	}
	return node.Spec{
		Kind: "MoveResizeWindow", Category: "Window", NeedsWindow: true,
		Inputs: append([]node.InputSpec{
			{Name: mrInExec, Type: "Exec"}, num(mrInX), num(mrInY), num(mrInW), num(mrInH),
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: mrDone, Type: "Exec", Data: []node.DataField{{Name: "Window", Type: "Window"}}},
		},
	}
}

func (MoveResizeWindow) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	if err := ctx.Window().MoveResize(in.Int(mrInX), in.Int(mrInY), in.Int(mrInW), in.Int(mrInH)); err != nil {
		return nil, err
	}
	w, err := ctx.Window().Snapshot()
	if err != nil {
		return nil, err
	}
	return ctx.Out(mrDone).Set("Window", w).Fire(), nil
}
```

- [ ] **Step 4/5**: 通过 → 提交 `feat(window): MoveResizeWindow node`。

---

### Task B5: `CloseWindow` 节点

**Files:** Create `internal/nodes/window/close_window.go`; Test 追加。

- [ ] **Step 1: 失败测试**: 注入 recordingWindowService, 断言 `Close()` 被调 + 走 Done(无 Window 透传)。
- [ ] **Step 2: 运行确认失败**。
- [ ] **Step 3: 实现**

```go
// internal/nodes/window/close_window.go
package window

import "yotta/internal/node"

func init() { node.Register(&CloseWindow{}) }

type CloseWindow struct{}

const (
	cwInExec = "In"
	cwDone   = "Done"
)

func (CloseWindow) Spec() node.Spec {
	return node.Spec{
		Kind: "CloseWindow", Category: "Window", NeedsWindow: true,
		Inputs:  append([]node.InputSpec{{Name: cwInExec, Type: "Exec"}}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{{Name: cwDone, Type: "Exec"}},
	}
}

func (CloseWindow) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	if err := ctx.Window().Close(); err != nil {
		return nil, err
	}
	return ctx.Out(cwDone).Fire(), nil
}
```

- [ ] **Step 4/5**: 通过 → 提交 `feat(window): CloseWindow node`。

---

## Phase C — 可选 Window 输入 + 派发覆盖

### Task C1: `Spec.NeedsForeground` + `WindowInputSpec()` + Register 不变式 + 守卫

**Files:**
- Modify: `internal/node/spec.go`(标志 + helper), `internal/node/registry.go`(不变式)
- Test: `internal/node/registry_test.go` 或 `spec_consistency_test.go`(追加守卫)

**Interfaces:**
- Produces: `Spec.NeedsForeground bool`; `func node.WindowInputSpec() node.InputSpec`(返 `{Name:"Window", Type:"Window"}`); Register: `NeedsWindow ⟹ Inputs 含 Name=="Window"`, 否则 panic。

- [ ] **Step 1: 写失败测试**

```go
// internal/node/registry_test.go (追加)
func TestRegister_NeedsWindowRequiresWindowInput(t *testing.T) {
	ResetRegistryForTest()
	defer ResetRegistryForTest()
	defer func() {
		if recover() == nil { t.Fatal("NeedsWindow 无 Window 输入应 panic") }
	}()
	Register(&badNeedsWindowNode{}) // Spec.NeedsWindow=true 但 Inputs 无 Window
}
```
(`badNeedsWindowNode` 一个最小 Runnable, Spec.NeedsWindow=true, Inputs 只有 In。)

并加全量守卫:
```go
func TestAllNeedsWindowNodesHaveWindowInput(t *testing.T) {
	for _, rn := range All() {
		if !rn.Spec.NeedsWindow { continue }
		has := false
		for _, ip := range rn.Spec.Inputs { if ip.Name == "Window" && ip.Type == "Window" { has = true } }
		if !has { t.Errorf("%s NeedsWindow 但缺 Window 输入", rn.Spec.Kind) }
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/node/ -run 'TestRegister_NeedsWindowRequiresWindowInput|TestAllNeedsWindowNodesHaveWindowInput' -v`
Expected: FAIL(不变式未加 / 现有节点还没 spread WindowInputSpec)。

- [ ] **Step 3: 实现**

`spec.go` Spec 结构加(挨着 NeedsWindow):
```go
	// NeedsForeground — 节点向目标窗口注入输入(SendInput 后端需前台焦点)。派发期 Window 覆盖
	// 时, 若后端是 sendinput 且本标志为真, 框架补拉一次前台。输入类节点(Click/KeyPress...)置真。
	NeedsForeground bool `json:"needsForeground,omitempty"`
```
加 helper:
```go
// WindowInputSpec — NeedsWindow 节点统一 spread 的可选窗口输入。连了→派发期作用在该窗口;
// 不连→当前活动窗口。框架在 execNodeViaFramework 解释此 pin(节点 Run 无需读它)。
func WindowInputSpec() InputSpec {
	return InputSpec{Name: "Window", Type: "Window"}
}
```
`registry.go` Register 在现有 capability 校验后加:
```go
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
```

- [ ] **Step 4: 运行**: 此时 `TestRegister_...` PASS, 但 `TestAllNeedsWindowNodesHaveWindowInput` 仍 FAIL(现有 22 节点没加)——**预期**, Task C4 补齐后转绿。本步先只验 `TestRegister_NeedsWindowRequiresWindowInput`:
Run: `go test ./internal/node/ -run TestRegister_NeedsWindowRequiresWindowInput -v` → PASS。

- [ ] **Step 5: 提交** `feat(node): NeedsForeground flag + WindowInputSpec + Register invariant`

---

### Task C2: rt 覆盖栈 + ActiveHWND/WindowHandle 读栈顶

**Files:**
- Modify: `internal/services/container/runtime/runtime_context.go`
- Test: `internal/services/container/runtime/runtime_context_test.go`(追加)

**Interfaces:**
- Produces: `rt.PushWindowOverride(wh winutil.WindowHandle)`, `rt.PopWindowOverride()`; `ActiveHWND()/WindowHandle()` 栈非空返栈顶。

- [ ] **Step 1: 写失败测试**

```go
func TestRuntimeContext_WindowOverrideStack(t *testing.T) {
	rt := &RuntimeContext{}
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 1})
	rt.PushWindowOverride(winutil.WindowHandle{HWND: 2})
	if h, _ := rt.ActiveHWND(); h != 2 { t.Fatal("栈顶应为 2") }
	rt.PushWindowOverride(winutil.WindowHandle{HWND: 3})
	if h, _ := rt.ActiveHWND(); h != 3 { t.Fatal("嵌套栈顶应为 3") }
	rt.PopWindowOverride()
	if h, _ := rt.ActiveHWND(); h != 2 { t.Fatal("pop 后应回 2") }
	rt.PopWindowOverride()
	if h, _ := rt.ActiveHWND(); h != 1 { t.Fatal("清空后回粘性 1") }
}
```

- [ ] **Step 2: 运行确认失败** → FAIL(Push/Pop 未定义)。

- [ ] **Step 3: 实现**(runtime_context.go)

rt 结构加(windowMu 同段):
```go
	windowOverride []winutil.WindowHandle // 覆盖栈; 栈顶为当前有效窗口; windowMu 守护
```
方法:
```go
func (rt *RuntimeContext) PushWindowOverride(wh winutil.WindowHandle) {
	rt.windowMu.Lock(); defer rt.windowMu.Unlock()
	rt.windowOverride = append(rt.windowOverride, wh)
}
func (rt *RuntimeContext) PopWindowOverride() {
	rt.windowMu.Lock(); defer rt.windowMu.Unlock()
	if n := len(rt.windowOverride); n > 0 { rt.windowOverride = rt.windowOverride[:n-1] }
}
```
`WindowHandle()` 改:
```go
func (rt *RuntimeContext) WindowHandle() winutil.WindowHandle {
	rt.windowMu.RLock(); defer rt.windowMu.RUnlock()
	if n := len(rt.windowOverride); n > 0 { return rt.windowOverride[n-1] }
	return rt.window
}
```
`ActiveHWND()` 改(同样先读栈顶):
```go
func (rt *RuntimeContext) ActiveHWND() (uintptr, error) {
	rt.windowMu.RLock(); defer rt.windowMu.RUnlock()
	wh := rt.window
	if n := len(rt.windowOverride); n > 0 { wh = rt.windowOverride[n-1] }
	if wh.HWND == 0 { return 0, ErrNoActiveWindow }
	return wh.HWND, nil
}
```

- [ ] **Step 4: 运行确认通过** → PASS + `go build ./...`。

- [ ] **Step 5: 提交** `feat(runtime): per-node window override stack`

---

### Task C3: `execNodeViaFramework` 覆盖 wrap

**Files:**
- Modify: `internal/node/errorcodes.go`(CodeWindowInvalid), `internal/services/container/runtime/dispatch_v5.go`
- Test: `internal/services/container/runtime/dispatch_v5_test.go` 或新 `window_override_dispatch_test.go`

**Interfaces:**
- Consumes: `dataWire["Window"]`(node.Window), `rn.Spec.NeedsForeground`, `r.rt.Container.InputBackend`, `r.rt.Game.BringToForeground`, `winutil.IsWindow`, `rt.PushWindowOverride/PopWindowOverride`。

- [ ] **Step 1: 写失败测试**(fixture 注两窗口, 仿 `win32windowtarget_dispatch_test.go`)

```go
// 关键断言:
// ① 一个 ClickAt 节点的 dataWire 带 Window{HWND:B} → 跑它期间 rt.ActiveHWND()==B;
// ② 跑完 rt.ActiveHWND() 回粘性 A;
// ③ Window{HWND:0 或 IsWindow=false} → execNodeViaFramework 返 CodeWindowInvalid 错。
```
(用一个假的 NeedsWindow 节点记录它 Run 时 `ctx` 看到的 hwnd; 或断言覆盖栈在 Run 前后变化。具体构造照 `win32windowtarget_dispatch_test.go` 的 runner fixture。)

- [ ] **Step 2: 运行确认失败** → FAIL。

- [ ] **Step 3: 实现**

`errorcodes.go` 加码:
```go
	CodeWindowInvalid  ErrCode = "window_invalid"  // Window 输入指向的句柄已失效
```
并加进 `ErrorCodes` map。

`dispatch_v5.go` `execNodeViaFramework` —— 在 `buildDataWireFor` 之后、`RunNode` 之前插覆盖逻辑:
```go
	dataWire := r.buildDataWireFor(ctx, node, rn)
	config := r.buildConfigFor(node)
	execData := r.buildExecDataFor(tok)

	// 可选 Window 输入: 连了则派发期把活动窗口覆盖成它(作用域限本节点)。
	if raw, ok := dataWire["Window"]; ok {
		w, ok := raw.(nodepkg.Window)
		if !ok || !winutil.IsWindow(w.HWND) {
			return nil, nodepkg.Failf(nodepkg.CodeWindowInvalid, nil,
				"%s: Window 输入无效或句柄已失效", node.Kind)
		}
		wh := winutil.WindowHandle{HWND: w.HWND, Title: w.Title, Class: w.Class,
			ProcessName: w.Process, PID: w.PID, ClientW: w.ClientW, ClientH: w.ClientH}
		r.rt.PushWindowOverride(wh)
		defer r.rt.PopWindowOverride()
		// sendinput 后端 + 需前台的输入节点: 补拉一次前台(不在前台 SendInput 打错窗)。
		if r.rt.Container != nil && r.rt.Container.InputBackend == "sendinput" &&
			rn.Spec.NeedsForeground && r.rt.Game != nil {
			r.rt.Game.BringToForeground(w.HWND)
			time.Sleep(150 * time.Millisecond)
		}
	}

	result := nodepkg.RunNode(ctx, rn, dataWire, config, execData, r.bundle, node.LogEnabled)
	return r.routeResult(node, tok, result)
```
(确认 `dispatch_v5.go` 已 import `winutil` + `time`; 没有则加。`winutil` 路径 `yotta/pkg/winutil`。)

- [ ] **Step 4: 运行确认通过** → `go test ./internal/services/container/runtime/ -run Override -v` PASS + `go build ./...`。

- [ ] **Step 5: 提交** `feat(runtime): per-node Window input override in dispatch + WINDOW_INVALID`

---

### Task C4: 给 22 个 NeedsWindow 节点加 Window 输入 + 标 NeedsForeground

**Files:** Modify 全部 `Spec.NeedsWindow==true` 节点(Detect 11 + Input 10 + PlayClip)。

**Interfaces:** Consumes `node.WindowInputSpec()`(C1)。

- [ ] **Step 1: 先列出全部 NeedsWindow 节点**

Run: `go run ./cmd/node-catalog export | grep -i needsWindow` 或直接 `grep -rln "NeedsWindow: true" internal/nodes/`。
Expected: 拿到完整清单(对照 reference §6: CheckTemplate/ClickTemplate/DetectColor/DetectColorBlobs/DetectColorHSV/DualColorBarTrack/ROIColorScan/Screenshot/WaitChange/WaitStable/WaitTemplate/BringWindowForeground/ClickAt/KeyHoldStart/KeyHoldStop/KeyPress/MouseHoldStart/MouseHoldStop/MouseMoveRel/MouseMoveTo/Scroll/PlayClip + 本计划新增的 WindowState/MoveResizeWindow/CloseWindow 已自带)。

- [ ] **Step 2: 逐节点把 `node.WindowInputSpec()` append 进 Inputs**

每个节点的 `Spec().Inputs` 末尾改成 `append([]node.InputSpec{...原有...}, node.WindowInputSpec())`(若原是字面 slice, 包一层 append)。**输入类节点**(ClickAt/KeyPress/KeyHoldStart/KeyHoldStop/MouseHoldStart/MouseHoldStop/MouseMoveRel/MouseMoveTo/Scroll + BringWindowForeground)同时加 `NeedsForeground: true`。截图/检测/PlayClip **不**加 NeedsForeground(后台抓帧不需前台)。
示例(ClickAt):
```go
		NeedsWindow: true,
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: caInExec, Type: "Exec"},
			// ... 原有输入 ...
		}, node.WindowInputSpec()),
```

- [ ] **Step 3: 运行全量守卫**

Run: `go test ./internal/node/ -run TestAllNeedsWindowNodesHaveWindowInput -v`
Expected: PASS(全部 NeedsWindow 节点现都含 Window 输入)。

- [ ] **Step 4: 跑节点 + catalog 测试**

Run: `go test ./internal/nodes/... ./internal/catalog/... -count=1`
Expected: PASS(注意 catalog drift —— 加了 Window 输入会改 catalog 结构, 若有 golden 需 i18n 补在 Task D2 后一并过; 此处若 catalog drift 因缺 i18n 报红, 记下、D2 后复跑)。

- [ ] **Step 5: 提交** `feat(nodes): optional Window input on all NeedsWindow nodes + NeedsForeground`

---

### Task C5: NeedsWindow 校验放宽(图级)

**Files:** Modify `internal/services/container/runtime/runner.go`(`containerNeedsWindow` 调用方) + `internal/services/container/validator.go`(缺 Win32WindowTarget 校验)。

**Interfaces:** 判定: 仅当存在「无 Window 输入连线」的 NeedsWindow 节点 **且** 全图无 Win32WindowTarget 时才要求/报缺。

- [ ] **Step 1: 写失败测试**(validator_test 追加)

```go
// 图: GetWindow→绑变量→GetVar→Screenshot(Window 连线), 无 Win32WindowTarget → 不应报「缺 Win32WindowTarget」。
// 图: ClickAt(Window 未连)无 Win32WindowTarget → 应报缺。
```

- [ ] **Step 2: 运行确认失败** → 现状两图都报缺(校验未放宽)→ 第一图 FAIL。

- [ ] **Step 3: 实现**: 在「缺 Win32WindowTarget」判定里, 跳过「该 NeedsWindow 节点的 `Window` 输入 pin 在图里有入边」的节点。判定「有入边」用图的 data edges(`Window` pin 作为 target)。仅当**剩余**(无 Window 入边的)NeedsWindow 节点非空且全图无 Win32WindowTarget → 报缺。

- [ ] **Step 4: 运行确认通过** → 两图分别 不报 / 报。`go test ./internal/services/container/... -count=1` 绿。

- [ ] **Step 5: 提交** `feat(validator): relax Win32WindowTarget requirement when Window input wired`

---

## Phase D — 类别 + i18n + 收尾验证

### Task D1: "Window" palette 类别

**Files:** Modify `window_target.go`/`wait_window.go`/`wait_window_gone.go`(Category→"Window") + `bring_foreground.go`(Category→"Window") + `adapter.ts`(GROUP_MAP) + `NodePalette.vue`(GROUP_LABEL) + `useNodeGroupColor.ts`(GROUP_I18N_KEY)。

- [ ] **Step 1**: 4 个现有节点 `Category: "System"/"Input"` 改 `"Window"`。
- [ ] **Step 2**: 前端 `adapter.ts` GROUP_MAP 加 `Window` → group key; `NodePalette.vue` GROUP_LABEL + `useNodeGroupColor.ts` GROUP_I18N_KEY 都指向 `nodeGroup.window`(zh/en 要建该键, D2)。
- [ ] **Step 3**: Run `go test ./internal/catalog/... -count=1`(Category 变更不破结构) + `cd frontend && pnpm typecheck`。
- [ ] **Step 4**: 提交 `refactor(window): gather window nodes into Window palette category`。

### Task D2: i18n

**Files:** Modify `frontend/src/i18n/zh.ts` + `en.ts`; Run `pnpm gen:node-i18n`。

- [ ] **Step 1**: 加 `node.GetWindow/WindowState/MoveResizeWindow/CloseWindow` 块(label/description/各 input label/Done+Window 出口 label/State 5 个 option/Title-Class-ProcessName-TitleMatch label)。zh/en 对称。
- [ ] **Step 2**: 加共享 `input.Window.label`(= "窗口"/"Window") 到每个 NeedsWindow 节点的 i18n 块(gen 按 node.<Kind>.input.Window 抽; 22+ 处, 文案统一)。加 `nodeGroup.window`("窗口"/"Window")。
- [ ] **Step 3**: Run `cd frontend && pnpm gen:node-i18n && pnpm i18n:check && pnpm typecheck`。
- [ ] **Step 4**: Run `go test ./internal/catalog/... -count=1`(drift 守卫现应绿)。
- [ ] **Step 5**: 提交 `feat(i18n): window nodes + Window input + Window group labels`。

### Task D3: 全绿门 + 真机 smoke 清单

**Files:** 无(验证)。

- [ ] **Step 1**: Run `go build ./... && go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... ./internal/services/container/... -count=1`。Expected: 绿(除 build.md 预存失败基线)。
- [ ] **Step 2**: Run `cd frontend && pnpm typecheck && pnpm i18n:check`。
- [ ] **Step 3**: Run `task build`(按 `flightdeck/knowledge/build/build.md`)。
- [ ] **Step 4**: 真机 smoke(人工, 记录结果): ① 侧边面板/右键菜单/explorer 三处能找到 Window 组的 GetWindow/WindowState/MoveResizeWindow/CloseWindow; ② 单窗口图(Win32WindowTarget→WindowState 最大化)不连 Window 输入照常作用当前窗口; ③ 多窗口图(GetWindow 主/子各绑变量 → 两个 WindowState 各连 GetVar)分别作用对窗口; ④ 无边框全屏→MoveResize→退无边框 回到全屏前布局; ⑤ CloseWindow 后接 WaitWindowGone 确认关闭; ⑥ sendinput 后端下, ClickAt 连子窗口 Window 输入能打到子窗口(覆盖期补前台生效)。
- [ ] **Step 5**: smoke 全过 → 把 `flightdeck/work/window-control/` 移出 work(归档到 `~/.flightdeck/projects/<slug>/archive/`), cockpit 收尾。

---

## 执行顺序(依赖)

A1 → B1 → B2 → **A3**(依赖 B2 的 Snapshot) → A2 → B3 → B4 → B5 → C1 → C2 → C3 → C4 → C5 → D1 → D2 → D3。
(A2 GetWindow 不依赖 B2, 可与 B3 前任意时刻做; 这里排在 A3 后只为 commit 线性。)

## Self-Review 记录

- **Spec 覆盖**: design §4.1→A1; §4.2→A2/A3; §4.3 输入声明→C1, 覆盖栈→C2, wrap→C3, 22 节点→C4, 校验放宽→C5; §4.4 控制节点→B3/B4/B5(含 Done.Window 重读、borderless saved-state、CloseWindow 语义); §4.5 分层→B1/B2; §4.6 类别→D1; §7 测试→各任务 Step1 + D3; i18n→D2。**全覆盖, 无遗漏**。
- **类型一致**: `node.Window` 字段(HWND/Title/Class/Process/PID/ClientW/ClientH)在 A1 定义, A2/A3/B2/B3/C3 一致引用; `winutil.WindowHandle.ProcessName`↔`node.Window.Process` 映射在 windowFromHandle/Snapshot/C3 三处一致。WindowService 新方法签名在 B2 定义, B3/B4/B5 一致调用。
- **占位符**: winutil 原语的 lxn/win 签名核实下沉为 B1-Step1 的真实 grep 步骤(非占位, 是「先验库再写」纪律); 其余均含完整代码。
