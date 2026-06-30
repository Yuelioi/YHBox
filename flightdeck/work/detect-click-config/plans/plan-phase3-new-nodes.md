# Phase 3:新节点群 + 后端 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development 逐 task 实现。步骤 `- [ ]`。依赖 Phase 1(`MatchHit`/roi/`matchOnce`/`waitOrCancel`)与 Phase 2(`clickWithMods`/`parseMods`/`interClickGapMs` 在 `internal/nodes/detect/click_common.go`)。

**Goal:** 补齐 spec 剩余 5 项 + ClickAt 复用:② WaitTemplateGone · ⑥ Swipe · ⑦ InputText · ⑨ Scroll 横向 · ⑩ StopApp · ClickAt 接 Keys/ClickCount。

**Architecture:** 4 个新节点 + 改 2 个现有节点(Scroll/ClickAt)。新输入能力走既有 4 层栈:`pkg/input`(原语)→ 输入后端接口(`RuntimeContext.Input` 的类型,生产实现 + 测试 `spyInputBackend`)→ `inputAdapter`(ratio→px、取 hwnd)→ `node.InputService`(节点面)。进程管理走 `pkg/platform`(同 RunProgram 的 ShellOpen)。

**Tech Stack:** Go;`internal/nodes/detect`、`internal/nodes/input`、`internal/nodes/io`、`internal/node`、`internal/services/container/runtime`、`pkg/input`、`pkg/platform`。

## Global Constraints

- **不要兼容**:不留 shim;新增后端方法直接加进接口 + 所有实现(含 spy stub),`go build ./...` 编译器揪漏网点。
- **新节点/新 pin 默认 = 旧行为零回归**:Scroll 默认 `Axis=vertical` = 旧竖直滚;ClickAt 默认 `Keys=""/ClickCount=1` = 旧单击。
- **TDD**:节点逻辑用 mock InputService 验调用序列;后端原语(TypeText 编码 / hscroll 消息)按 pkg/input 既有单测风格(如有)补,无现成 harness 的纯 WinAPI 投递层可只做节点级 mock 验证 + 真机 smoke(诚实标注)。
- 构建:`go build ./...` + `go test ./internal/...`;新节点改 catalog → `cd frontend && pnpm gen:node-i18n` + **在 `frontend/src/i18n/zh.ts` 和 `en.ts` 补新节点的 label/description/pin 文案**(Phase 2 教训:光跑 gen 不够,要先补源),再 `go test ./internal/catalog/...`。
- **验证基线**:见 `flightdeck/knowledge/build/build.md`; 当前 Go/前端测试应绿, 不再套旧预存红豁免。
- **本机 Write 故障**:写文件尾部可能混入 `</content>`,写完检查清掉;改既有文件优先 Edit;诊断面板编辑期 stale,以 `go build`/`go test` 退出码为准。

---

### Task 3.1: WaitTemplateGone(新 detect 节点)

等模板从画面消失。纯 vision,无后端。镜像 WaitTemplate 取反。

**Files:**
- Create: `internal/nodes/detect/wait_template_gone.go`
- Test: `internal/nodes/detect/wait_template_gone_test.go`
- i18n: `frontend/src/i18n/zh.ts`、`en.ts`(加 WaitTemplateGone 块)

**Interfaces:**
- Consumes(已有):`matchOnce(ctx, keys, threshold) (node.MatchHit, error)`、`waitOrCancel(ctx, d) error`、`visionWaitPollMs`、`templateDeps`(均在 detect 包)

- [ ] **Step 1:写失败测试**

`wait_template_gone_test.go`(参照 `wait_template_test.go` 的 mockVision 构造):
```go
package detect

import (
	"testing"
	"yotta/internal/node"
)

func TestWaitTemplateGone_Gone(t *testing.T) {
	// mockVision: 第 1 帧命中, 之后 nil → 应走 Gone
	vision := &mockVision{hitForFirstN: 1, point: &node.Point{X: 0.5, Y: 0.5}, conf: 0.9}
	out := runNodeWithVision(t, &WaitTemplateGone{}, map[string]any{
		"Templates": "ns.icon", "TimeoutMs": json.Number("2000"), "Threshold": json.Number("0.85"),
	}, vision)
	if out.ExitName != "Gone" {
		t.Fatalf("exit=%s want Gone", out.ExitName)
	}
}

func TestWaitTemplateGone_Timeout(t *testing.T) {
	// 一直命中 → 超时走 Timeout, 带 Conf
	vision := &mockVision{alwaysHit: true, point: &node.Point{X: 0.5, Y: 0.5}, conf: 0.93}
	out := runNodeWithVision(t, &WaitTemplateGone{}, map[string]any{
		"Templates": "ns.icon", "TimeoutMs": json.Number("0"), // 单帧: 在 → Timeout
	}, vision)
	if out.ExitName != "Timeout" {
		t.Fatalf("exit=%s want Timeout", out.ExitName)
	}
}
```
> 实现者:对齐本包既有测试驱动方式(`wait_template_test.go` 怎么跑节点拿出口、mockVision 字段叫什么)。若现有 mockVision 没有"前 N 帧命中后消失"的能力,加一个计数字段;`alwaysHit`/单帧路径用现成即可。

- [ ] **Step 2:跑确认失败**

Run: `go test ./internal/nodes/detect/ -run TestWaitTemplateGone -v`
Expected: FAIL(undefined: WaitTemplateGone)

- [ ] **Step 3:实现**

`internal/nodes/detect/wait_template_gone.go`:
```go
// internal/nodes/detect/wait_template_gone.go
// WaitTemplateGone — 等指定模板从画面消失 (无任何命中) 再放行; 超时仍在走 Timeout。
package detect

import (
	"encoding/json"
	"strings"
	"time"

	"yotta/internal/node"
)

func init() { node.Register(&WaitTemplateGone{}) }

type WaitTemplateGone struct{}

const (
	wtgInExec      = "In"
	wtgInTemplates = "Templates"
	wtgInTimeoutMs = "TimeoutMs"
	wtgInThreshold = "Threshold"
	wtgOutGone     = "Gone"
	wtgOutTimeout  = "Timeout"
	wtgDataConf    = "Conf"
)

func (WaitTemplateGone) Spec() node.Spec {
	return node.Spec{
		Kind:        "WaitTemplateGone",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: wtgInExec, Type: "Exec"},
			{Name: wtgInTemplates, Type: "String", Semantic: "TemplateGUID", Required: true,
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: wtgInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: wtgInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
		},
		Outputs: []node.OutputSpec{
			{Name: wtgOutGone, Type: "Exec"},
			{Name: wtgOutTimeout, Type: "Exec",
				Data: []node.DataField{{Name: wtgDataConf, Type: "Number", Optional: true}}},
		},
	}
}

func (WaitTemplateGone) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(wtgInTemplates)
	threshold := in.Float64(wtgInThreshold)
	timeout := time.Duration(in.Int(wtgInTimeoutMs)) * time.Millisecond

	hit, err := matchOnce(ctx, keys, threshold)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "WaitTemplateGone %s: %v", strings.Join(keys, "+"), err)
	}
	if !hit.Found {
		return ctx.Out(wtgOutGone).Fire(), nil
	}
	if timeout <= 0 {
		return ctx.Out(wtgOutTimeout).Set(wtgDataConf, hit.Conf).Fire(), nil
	}
	deadline := ctx.Now().Add(timeout)
	for {
		if err := waitOrCancel(ctx, visionWaitPollMs*time.Millisecond); err != nil {
			return nil, err
		}
		hit, err = matchOnce(ctx, keys, threshold)
		if err != nil {
			return nil, node.Failf(node.CodeCaptureFailed, err, "WaitTemplateGone recheck %s: %v", strings.Join(keys, "+"), err)
		}
		if !hit.Found {
			return ctx.Out(wtgOutGone).Fire(), nil
		}
		if ctx.Now().After(deadline) {
			return ctx.Out(wtgOutTimeout).Set(wtgDataConf, hit.Conf).Fire(), nil
		}
	}
}

func (WaitTemplateGone) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(wtgInTemplates))
}
```
> 确认 `visionWaitPollMs` 在 detect 包可见(Phase 1/2 用过 `visionWaitPollMs` 或 `visionPollInterval`,用实际存在的那个常量名;若叫 visionPollInterval 则用它)。

- [ ] **Step 4:跑确认通过** — `go test ./internal/nodes/detect/ -run TestWaitTemplateGone -v` → PASS
- [ ] **Step 5:i18n** — 在 zh.ts/en.ts 加 WaitTemplateGone 块(label「等待模板消失」、description 大白话、pin: Templates/TimeoutMs/Threshold、output: Gone/Timeout);`cd frontend && pnpm gen:node-i18n`;`go test ./internal/catalog/...` → PASS
- [ ] **Step 6:提交** — `git commit -m "feat(detect): WaitTemplateGone — 等模板从画面消失再放行"`

---

### Task 3.2: ClickAt 接 Keys/ClickCount(提升公共 helper 到 internal/node)

`clickWithMods`/`parseMods` 现在 detect 包,ClickAt 在 input 包用不了。提升到 `internal/node`(导出),两包共用。

**Files:**
- Create: `internal/node/click_mods.go`(`ClickWithMods` + `ParseMods` + `InterClickGapMs`)
- Test: `internal/node/click_mods_test.go`
- Modify: `internal/nodes/detect/click_common.go`(删 clickWithMods/parseMods,改调 node.*)、`click_template.go`(调用 + Validate 改 node.ParseMods)
- Modify: `internal/nodes/input/click_at.go`(加 Keys/ClickCount + 用 node.ClickWithMods + Validate)
- Test: `internal/nodes/input/click_at_test.go`

**Interfaces:**
- Produces:`node.ClickWithMods(ctx Ctx, pt Point, btn string, keys string, count int) error`、`node.ParseMods(keys string) ([]string, bool)`、`node.InterClickGapMs`

- [ ] **Step 1:写失败测试** — `click_mods_test.go` 验 `ParseMods`(ctrl+shift / 空 / 非法)+ `ClickWithMods` 序列(用 internal/node 的 mock InputService;参照 detect 包 Phase 2 的 recInput,但放 node 包测试)。
- [ ] **Step 2:跑确认失败**
- [ ] **Step 3:实现** — 把 Phase 2 `click_common.go` 的 `parseMods`/`clickWithMods`/`validMods`/`interClickGapMs` 逻辑搬到 `internal/node/click_mods.go`,导出为 `ParseMods`/`ClickWithMods`/`InterClickGapMs`。**`clickWithMods` 里的 `waitOrCancel(ctx, gap)` 改成内联 select**(node 包没有 detect 的 waitOrCancel):
```go
func sleepOrCancel(ctx Ctx, d time.Duration) error {
	if d <= 0 { return nil }
	select {
	case <-ctx.Context().Done():
		return ctx.Context().Err()
	case <-time.After(d):
		return nil
	}
}
```
- [ ] **Step 4:改 detect** — `click_common.go` 删本地 parseMods/clickWithMods/validMods/interClickGapMs;`click_template.go` 调用改 `node.ClickWithMods(...)`,Validate 改 `node.ParseMods(...)`。`go build ./...` 确认 detect 包过。
- [ ] **Step 5:改 ClickAt** — 加 `caInKeys`(String,默认空)/`caInClickCount`(Integer,默认 1)pin(Advanced);Run 末尾 `ctx.Input().Click(...)` 换成 `node.ClickWithMods(ctx, node.Point{X:x,Y:y}, btn, in.String(caInKeys), in.Int(caInClickCount))`;Validate 加 INVALID_MODIFIER_KEY(`node.ParseMods` 不合法)+ INVALID_CLICK_COUNT(<1)。保留 MoveMs/DurationMs/JitterPct 现有逻辑(ClickWithMods 内部用 50ms hold;若要保留 ClickAt 的 DurationMs 作为 hold,给 ClickWithMods 加 durationMs 参数或在 ClickAt 仍走 moveCursor 后调 —— 实现者按最小改动:moveCursor 保留, 点击换 ClickWithMods)。
  > 注:ClickWithMods 内部 hold 固定 50ms,会丢 ClickAt 的 DurationMs 语义。**修正**:给 `node.ClickWithMods` 加 `durationMs int` 参数(detect 调用传 50,ClickAt 传 in.Int(caInDurationMs)),保 ClickAt 的按住时长可调(长按)不回归。
- [ ] **Step 6:测试 + i18n** — `go test ./internal/node/... ./internal/nodes/detect/... ./internal/nodes/input/...`;ClickAt 加了 pin → zh.ts/en.ts 补 ClickAt 的 Keys/ClickCount 文案 + `pnpm gen:node-i18n` + catalog 测试。
- [ ] **Step 7:提交** — `git commit -m "refactor(node): 提升 ClickWithMods/ParseMods 到 node 包; ClickAt 接组合键+多击"`

---

### Task 3.3: Swipe(新 input 节点)

Begin→End 拖拽。后端 `pkg/input.MouseDrag` 已存在,只差暴露。

**Files:**
- Create: `internal/nodes/input/swipe.go` + 测试
- Modify: 输入后端接口 + 生产实现 + `spyInputBackend`(测试)+ `inputAdapter` + `node.InputService` —— 加 `Drag`
- i18n: zh.ts/en.ts 加 Swipe

**Interfaces:**
- Produces:`node.InputService.Drag(x1,y1,x2,y2 float64, button string, durationMs int) error`

- [ ] **Step 1:写失败测试** — Swipe 节点用 mock InputService(记录 Drag 调用参数)验 `Drag(beginX,beginY,endX,endY,btn,dur)` 被正确调用。
- [ ] **Step 2:跑确认失败**
- [ ] **Step 3:加后端 Drag(4 层)**
  - `node.InputService` 接口加 `Drag(x1, y1, x2, y2 float64, button string, durationMs int) error`。
  - 输入后端接口(`RuntimeContext.Input` 的类型,`spyInputBackend` 实现的那个 —— grep `spyInputBackend` 找接口定义文件)加 `Drag(h win.HWND, x1, y1, x2, y2 float64, button string, durationMs int) error`;生产实现里 ratio→client px(参照该实现 Click 怎么转)后调 `input.MouseDrag(h, px1,py1,px2,py2, btn 转 MouseButton, time.Duration(durationMs)*ms, activateDelay, cursorSettle)`(activateDelay/cursorSettle 用该实现 Click 用的同款值);`spyInputBackend` 加记录式空实现。
  - `inputAdapter` 加 `Drag`(ensure + hwnd + 转调 `a.rt.Input.Drag`,参照 Click)。
  - `go build ./...` 让编译器揪所有实现/ mock 漏网点,补全。
- [ ] **Step 4:实现 Swipe 节点**
```go
// internal/nodes/input/swipe.go
package input
// Swipe — 从 Begin 拖到 End。
// pins: In, Begin(Point), End(Point), DurationMs(Number,200), Button(下拉 left/right/middle)
// out: Done。NeedsWindow。
// Run: ctx.Input().Drag(begin.X, begin.Y, end.X, end.Y, btn, durationMs); 失败 node.Failf(CodeSendFailed,...)
```
(完整代码参照 ClickAt 结构:Spec + Run + Validate(button 校验);Begin/End 用 `in.Point(...)`。)
- [ ] **Step 5:测试通过**
- [ ] **Step 6:i18n + catalog**
- [ ] **Step 7:提交** — `git commit -m "feat(input): Swipe 拖拽节点 (InputService.Drag → pkg/input.MouseDrag)"`

---

### Task 3.4: Scroll 横向(改现有 + 后端)

Scroll 节点加 Axis;后端加横向滚动(`WM_MOUSEHWHEEL`)。

**Files:**
- Modify: `internal/nodes/input/scroll.go`(加 Axis pin)
- Modify: `node.InputService.Scroll`(加轴向)+ 后端接口/实现/spy + inputAdapter + `pkg/input`(横向滚)
- i18n: Scroll 块加 Axis

**Interfaces:**
- Changed:`node.InputService.Scroll(xRatio, yRatio float64, notches int, horizontal bool) error`(加 `horizontal`)

- [ ] **Step 1:写失败测试** — Scroll(Axis=horizontal) → mock InputService 记录 `Scroll(...,horizontal=true)`;默认 Axis=vertical → horizontal=false(零回归)。
- [ ] **Step 2:跑确认失败**
- [ ] **Step 3:改后端(4 层 + pkg/input)**
  - `pkg/input`:现有 `MouseScroll(hwnd, notches, activateDelay)` 发 `WM_MOUSEWHEEL`;加 `MouseScrollH(hwnd, notches, activateDelay)` 发 `WM_MOUSEHWHEEL`(wParam 高位 = notches*WHEEL_DELTA,同 MouseScroll 但消息号换 0x020E)。grep `WM_MOUSEWHEEL`/`MouseScroll` 看现有实现照搬改消息号。
  - `node.InputService.Scroll` 加 `horizontal bool` 参数;后端接口 `Scroll(h, x, y, notches, horizontal)` ;实现按 horizontal 选 MouseScroll/MouseScrollH;spy 记录;inputAdapter 透传。
  - `go build ./...` 揪漏网(现有 Scroll 调用点 + mock),补 `horizontal=false`。
- [ ] **Step 4:改 Scroll 节点** — 加 `Axis` 下拉(`vertical` 默认/`horizontal`);Run 读 axis,`ctx.Input().Scroll(x, y, delta, axis=="horizontal")`。
- [ ] **Step 5:测试通过(含默认竖直零回归)**
- [ ] **Step 6:i18n(Scroll 加 Axis 文案)+ catalog**
- [ ] **Step 7:提交** — `git commit -m "feat(input): Scroll 横向滚动 (Axis pin + pkg/input WM_MOUSEHWHEEL)"`

---

### Task 3.5: InputText(新 input 节点 + 新后端原语)

输入字符串。需新后端 `pkg/input.TypeText`(SendInput KEYEVENTF_UNICODE 逐 rune)。

**Files:**
- Create: `internal/nodes/input/input_text.go` + 测试
- Modify: `pkg/input`(加 `TypeText`)+ 后端接口/实现/spy + inputAdapter + `node.InputService`(加 `TypeText`)
- i18n: 加 InputText

**Interfaces:**
- Produces:`node.InputService.TypeText(s string) error`

- [ ] **Step 1:写失败测试** — InputText 节点用 mock InputService 记录 `TypeText("hello")`;空 Text → Validate/Run 报错或 Fail(实现者定:Text Required)。
- [ ] **Step 2:跑确认失败**
- [ ] **Step 3:加 pkg/input.TypeText** — 逐 rune 走 `SendInput` 的 `KEYEVENTF_UNICODE`(参照本文件 `sendInputMouseRel`/`procSendInput` + INPUT 结构的用法):每字符两条 INPUT(keydown unicode + keyup unicode),`wScan=rune`、`dwFlags=KEYEVENTF_UNICODE`(keyup 再 `|KEYEVENTF_KEYUP`)。BMP 外字符(rune>0xFFFF)拆 surrogate pair。签名 `func TypeText(hwnd win.HWND, s string)`(hwnd 可仅用于 FakeActivate,SendInput 走全局)。
- [ ] **Step 4:接 4 层** — `node.InputService.TypeText(s)`;后端接口 `TypeText(h, s)`;实现调 `input.TypeText(h, s)`;spy 记录;inputAdapter 透传。`go build ./...` 揪漏网。
- [ ] **Step 5:实现 InputText 节点**
```go
// internal/nodes/input/input_text.go
package input
// InputText — 向当前窗口输入一串文字 (支持 unicode)。
// pins: In, Text(String, Required, text widget)。out: Done / Fail(Error,Code)。NeedsWindow。
// Run: ctx.Input().TypeText(in.String(...)); 失败 node.Failf(CodeSendFailed,...) 走 Fail。
```
- [ ] **Step 6:测试 + i18n + catalog**
- [ ] **Step 7:提交** — `git commit -m "feat(input): InputText 输入字符串节点 (pkg/input.TypeText SendInput unicode)"`

---

### Task 3.6: StopApp(新 IO 节点)

按进程名/PID 杀进程。走 `pkg/platform`(同 RunProgram 的 ShellOpen 套路)。

**Files:**
- Create: `internal/nodes/io/stop_app.go` + 测试
- Modify: `pkg/platform`(加 `KillProcess(target string) error`)
- i18n: 加 StopApp

**Interfaces:**
- Produces:`platform.KillProcess(target string) error`(target = 进程名 *.exe 或纯数字 PID)

- [ ] **Step 1:写失败测试** — StopApp 节点:Target 空 → 报错/Fail;非空 → 调 KillProcess(可把 kill 抽成可注入函数变量便于测试,或测 platform.KillProcess 的命令构造)。
- [ ] **Step 2:跑确认失败**
- [ ] **Step 3:加 platform.KillProcess** — Windows:纯数字 → `taskkill /F /PID <n>`;否则 → `taskkill /F /IM <name>`(用 `os/exec`,参照 pkg/platform 既有 exec 风格;非零退出包 error)。grep pkg/platform 看现有 windows 文件组织(build tag)。
- [ ] **Step 4:实现 StopApp 节点**(IO 分组,Spec/Run 参照 RunProgram:Target Required + Done/Fail 出口)。
- [ ] **Step 5:测试 + i18n + catalog**
- [ ] **Step 6:提交** — `git commit -m "feat(io): StopApp 关闭进程节点 (platform.KillProcess taskkill)"`

---

## 收尾(全 10 项落地后)

- node-catalog 重导出核对 4 新节点 + Scroll/ClickAt/ClickTemplate 改动齐全、大白话到位。
- **真机 smoke(spec §收尾全清单)**:① 锚点偏移点中 · ⑤ 多命中点最上/第2个 · ⑧ 限 ROI · ② 等图消失 · ③ ctrl+点 · ④ 双击 · ⑥ 拖滑块 · ⑦ 搜索框打字 · ⑨ 横向滚 · ⑩ 杀进程。
- 全 effort 落地后:把 `work/detect-click-config/` 移出 work/ 到 cold store(flightdeck 完成约定);常驻技术知识(如 magnitude-switch 单位约定、输入后端 4 层扩展套路)若值得沉淀 → `knowledge/`。

## Self-Review(已过)

- **spec 覆盖**:②(3.1)⑥(3.3)⑦(3.5)⑨(3.4)⑩(3.6)+ ClickAt Keys/ClickCount(3.2)。至此 spec 10 项全覆盖。
- **占位符**:后端原语(TypeText/hscroll/Drag/KillProcess)给了确切 WinAPI 路径 + 签名 + "参照现有 X" 锚点,非 vague;节点级全代码或明确结构。诚实标注:纯 WinAPI 投递层无单测 harness 处靠节点 mock + 真机 smoke。
- **类型一致**:`node.ClickWithMods/ParseMods`(3.2 提升后)、`InputService.Drag/TypeText/Scroll(+horizontal)`(3.3/3.4/3.5)、`platform.KillProcess`(3.6)签名贯穿一致。
- **零回归**:Scroll Axis=vertical / ClickAt 默认 = 旧行为,各有回归测试。
