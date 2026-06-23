---
status: active
summary: ③ MCP 节点执行实现计划: winutil.EnumTopWindows + ContainerRunner.ExecOutputs 访问器 + internal/services/mcp 包(run_node harness 合成微容器跑+held-output 收割, find_window/list_windows, authoring 工具迁移) + settings arm 开关 + main.go Streamable HTTP server 生命周期 + 设置页 MCP tab + 退役 cmd/yotta-mcp。TDD, mock window/input/capture fixture。
last_updated: 2026-06-23
implements: specs/2026-06-23-mcp-node-exec.md
---

# ③ MCP 节点执行 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** GUI 进程内置一个 Streamable HTTP MCP server，给外部 AI 提供「跑单个动作节点（探测）+ 找窗 + 写图」的工具，闭合「AI 跑节点 → 生成容器」环。

**Architecture:** `run_node` 把单动作节点包成 `{Start→节点}` 微容器，经现有 `ContainerRunner` 跑、从 held-output 缓存（`execOutputs`）收割输出——零新执行机器。MCP server（`internal/services/mcpserver`）在 `main.go` 启动期装配、注入 GUI 已接好的常驻标准件（InputBus/TemplateMatcher/GameProvider/clipSvc/settings/container.Store），后台 goroutine 起 `server.NewStreamableHTTPServer`。现有独立 `cmd/yotta-mcp` 退役、authoring 工具迁入新包。

**Tech Stack:** Go 1.25.1 · `github.com/mark3labs/mcp-go v0.32.0`（`NewMCPServer`/`AddTool`/`NewStreamableHTTPServer`）· 现有 `internal/services/container/{,runtime}` · `pkg/winutil` · Wails3。

## Global Constraints

- **Go 1.25.1 · mcp-go v0.32.0**（已在 go.mod，勿改版本）。
- **不写兼容 shim**：项目未发布，无外部用户；删 `cmd/yotta-mcp` 直接切干净，不留 deprecated。
- **Git**：提交直推当前分支 `feat/v2-foundation`（无 PR），**永不 push 远端**；不跳 hook。每个 task 末尾 commit。
- **判红只信工具退出码**（`go build ./...` / `go test`），不信 IDE 面板；CRLF 警告是 benign 非错误。
- **验证命令以 `flightdeck/checklists/build.md` 为准**（go build / go test / vue-tsc / i18n check）。
- **预存失败基线**照 build.md（runtime 缺 fish fixture、i18n residue 42、pnpm lint 18 错）——这些**不是**本计划引入的红。
- **MCP server 包名 `mcpserver`**（dir `internal/services/mcpserver`），避免与 mcp-go 的 `mcp` 包名冲突。
- **可跑节点闸 = `Spec.NeedsWindow==true` 且非 `IsPureData`**（数据驱动，不写死 kind 名单）。
- **bindings 是 gitignore 产物**，前端类型真机须重启后端重生成。

---

## File Structure

| 文件 | 职责 |
|---|---|
| `pkg/winutil/window.go`（改） | 加 `EnumTopWindows() []WindowHandle` —— 枚举全部顶层可见窗口 |
| `internal/services/container/runtime/runner.go`（改） | 加 `(*ContainerRunner) ExecOutputs() map[string]any` 只读访问器 |
| `internal/services/execution/worker.go`（改） | 加 `(*Worker) IsRunning() bool` —— BUSY 闸用 |
| `internal/services/settings.go`（改） | 加 `MCPSettings{Armed bool}` + `Settings.MCP` + defaultSettings |
| `internal/services/mcpserver/server.go`（新） | `Server` 结构（注入常驻依赖）+ `NewServer` + `Register`（AddTool 装配）|
| `internal/services/mcpserver/authoring.go`（新） | list_nodes / get_graph_schema / validate_container / save_container（从 `cmd/yotta-mcp` 迁入）|
| `internal/services/mcpserver/schema.go`（新） | 图 schema 文本 + 样例（从 `cmd/yotta-mcp/schema.go` 迁入）|
| `internal/services/mcpserver/runnable.go`（新） | `isRunnable(spec) bool` + `execInPin(spec) string` 闸判定 |
| `internal/services/mcpserver/harness.go`（新） | `buildMicroContainer` + `runMicroContainer` + 收割（run_node 核心）|
| `internal/services/mcpserver/tools_exec.go`（新） | find_window / list_windows / run_node 三个 handler |
| `main.go`（改） | 装配 + 启动 StreamableHTTP server + 关停 Shutdown |
| `frontend/src/views/SettingsMCP.vue`（新）+ `SettingsView.vue`（改） | arm 开关 + URL 展示 tab |
| `frontend/src/i18n/{zh,en}.ts`（改） | `settingsTab.mcp` + `settingsMCP.*` |
| `cmd/yotta-mcp/`（删） | 退役独立 stdio 进程 |

---

### Task 1: `winutil.EnumTopWindows` —— 枚举全部顶层可见窗口

**Files:**
- Modify: `pkg/winutil/window.go`（在 `WindowMetadata` 后追加）
- Test: `pkg/winutil/window_test.go`（追加）

**Interfaces:**
- Consumes: 现成 helper `getWindowText`/`getClassName`/`getWindowPID`/`queryProcessName`/`getClientSize`、`procEnumWindows`、`win.IsWindowVisible`。
- Produces: `func EnumTopWindows() []WindowHandle` —— 返全部顶层可见窗口（query 进程名失败的跳过，与 ResolveWindow 同语义）。

- [ ] **Step 1: 写失败测试**

```go
// pkg/winutil/window_test.go
func TestEnumTopWindows_ReturnsVisibleWindows(t *testing.T) {
	got := EnumTopWindows()
	// 测试进程跑在桌面会话里, 至少应枚到若干窗口 (CI headless 可能为空 → 只断言不 panic + 字段完整性).
	for _, w := range got {
		if w.HWND == 0 {
			t.Fatalf("EnumTopWindows 返回了 HWND==0 的条目: %+v", w)
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./pkg/winutil/ -run TestEnumTopWindows -v`
Expected: FAIL —— `undefined: EnumTopWindows`。

- [ ] **Step 3: 实现**

```go
// EnumTopWindows 枚举全部顶层可见窗口, 返完整元数据列表 (MCP list_windows 用).
// 与 ResolveWindow 同枚举/同跳过语义: 不可见窗口跳过; 进程名 query 失败 (admin/僵尸) 跳过.
func EnumTopWindows() []WindowHandle {
	var out []WindowHandle
	callback := syscall.NewCallback(func(hwnd win.HWND, _ uintptr) uintptr {
		if !win.IsWindowVisible(hwnd) {
			return 1 // continue
		}
		pid := getWindowPID(hwnd)
		procName, err := queryProcessName(pid)
		if err != nil {
			return 1
		}
		cw, ch := getClientSize(hwnd)
		out = append(out, WindowHandle{
			HWND:        uintptr(hwnd),
			Title:       getWindowText(hwnd),
			Class:       getClassName(hwnd),
			ProcessName: strings.ToLower(procName),
			PID:         pid,
			ClientW:     cw,
			ClientH:     ch,
		})
		return 1 // continue 枚下一个
	})
	procEnumWindows.Call(callback, 0)
	return out
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./pkg/winutil/ -run TestEnumTopWindows -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add pkg/winutil/window.go pkg/winutil/window_test.go
git commit -m "feat(winutil): EnumTopWindows 枚举全部顶层可见窗口 (MCP list_windows)"
```

---

### Task 2: `ContainerRunner.ExecOutputs()` 只读访问器

**Files:**
- Modify: `internal/services/container/runtime/runner.go`
- Test: `internal/services/container/runtime/runner_test.go`（追加；若无此文件则建）

**Interfaces:**
- Produces: `func (r *ContainerRunner) ExecOutputs() map[string]any` —— 返 held-output 缓存的浅拷贝（键 `"<nodeID>.<field>"`）。run_node harness 收割用。

- [ ] **Step 1: 写失败测试**

```go
func TestContainerRunner_ExecOutputs_ReturnsCacheCopy(t *testing.T) {
	r := &ContainerRunner{execOutputs: map[string]any{"n.Code": "boom"}}
	got := r.ExecOutputs()
	if got["n.Code"] != "boom" {
		t.Fatalf("ExecOutputs 没返回缓存内容: %+v", got)
	}
	got["n.Code"] = "mutated" // 改返回值不该影响内部缓存
	if r.execOutputs["n.Code"] != "boom" {
		t.Fatalf("ExecOutputs 返回的不是拷贝, 内部缓存被改了")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/services/container/runtime/ -run TestContainerRunner_ExecOutputs -v`
Expected: FAIL —— `r.ExecOutputs undefined`。

- [ ] **Step 3: 实现**

```go
// ExecOutputs 返 held-output 缓存的浅拷贝 (键 "<nodeID>.<field>").
// MCP run_node 跑完单节点后据此收割节点输出 (见 docs/held-exec-outputs).
func (r *ContainerRunner) ExecOutputs() map[string]any {
	out := make(map[string]any, len(r.execOutputs))
	for k, v := range r.execOutputs {
		out[k] = v
	}
	return out
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/services/container/runtime/ -run TestContainerRunner_ExecOutputs -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/services/container/runtime/runner.go internal/services/container/runtime/runner_test.go
git commit -m "feat(runtime): ContainerRunner.ExecOutputs 只读访问器 (MCP 收割 held-output)"
```

---

### Task 3: `Worker.IsRunning()` + `MCPSettings{Armed}`

两个独立小改动（BUSY 闸 + arm 持久化），一个 task 内做掉。

**Files:**
- Modify: `internal/services/execution/worker.go`
- Modify: `internal/services/settings.go`
- Test: `internal/services/execution/worker_test.go`（追加）

**Interfaces:**
- Produces: `func (w *Worker) IsRunning() bool`（`active != nil`，持 `w.mu`）。
- Produces: `MCPSettings struct{ Armed bool }` + `Settings.MCP MCPSettings`（defaultSettings 给零值 `{Armed:false}`）。

- [ ] **Step 1: 写 Worker.IsRunning 失败测试**

```go
func TestWorker_IsRunning_FalseWhenIdle(t *testing.T) {
	q := NewExecutionQueue()
	w := NewWorker(q, func(ctx context.Context, _ execution.TargetRef) error { return nil }, nil, nil)
	if w.IsRunning() {
		t.Fatal("空闲 Worker 不该 IsRunning")
	}
}
```
> 构造参数以 `worker.go` 现有 `NewWorker` 签名为准（实现前先核对入参个数/类型，按实际补齐 stub）。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/services/execution/ -run TestWorker_IsRunning -v`
Expected: FAIL —— `w.IsRunning undefined`。

- [ ] **Step 3: 实现 Worker.IsRunning**

```go
// IsRunning 当前是否有 run 在执行 (MCP BUSY 闸用, 防 AI 驱动输入与 GUI run 交错).
func (w *Worker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.active != nil
}
```

- [ ] **Step 4: 加 MCPSettings**

`internal/services/settings.go`：`Settings` 结构在 `AI AISettings` 字段旁加：
```go
	MCP     MCPSettings     `json:"mcp"`     // MCP 对外暴露开关
```
并加类型（放 `AISettings` 定义附近）：
```go
// MCPSettings 控制对外 MCP server 的「武装」状态。Armed=false (默认) 时
// 会改变世界的工具 (run_node/save_container) 拒绝执行, 只读工具不受闸。
type MCPSettings struct {
	Armed bool `json:"armed"`
}
```
`defaultSettings()` 里**无需显式赋值**（`MCPSettings{}` 零值 `Armed:false` 即默认关）；确认 `defaultSettings` 返回的 `Settings{}` 字面量不会遗漏该字段即可（结构零值天然带上）。老 `settings.json` 无 `mcp` 键 → 反序列化天然得零值，无迁移代码。

- [ ] **Step 5: 跑测试 + build 确认通过**

Run: `go test ./internal/services/execution/ -run TestWorker_IsRunning -v && go build ./...`
Expected: PASS + build 绿。

- [ ] **Step 6: 提交**

```bash
git add internal/services/execution/worker.go internal/services/execution/worker_test.go internal/services/settings.go
git commit -m "feat(mcp): Worker.IsRunning BUSY 闸 + MCPSettings.Armed 持久化"
```

---

### Task 4: `mcpserver` 包骨架 + authoring 工具迁入

把 `cmd/yotta-mcp` 的 catalog/schema/validate/save 逻辑迁进新包（纯函数层），不接 HTTP（Task 8 才接）。

**Files:**
- Create: `internal/services/mcpserver/server.go`
- Create: `internal/services/mcpserver/authoring.go`
- Create: `internal/services/mcpserver/schema.go`
- Test: `internal/services/mcpserver/authoring_test.go`

**Interfaces:**
- Produces: `type Server struct{...}`（注入常驻依赖，见下）+ `func NewServer(deps Deps) *Server`。
- Produces: `listNodesJSON() []byte` / `graphSchemaJSON() []byte` / `validateContainerJSON([]byte) ([]byte, bool)` / `saveContainer(*container.Store, []byte) (SaveResult, []byte)`（从 spike 平移，逻辑不变）。

- [ ] **Step 1: 建 Server 结构 + Deps**

```go
// internal/services/mcpserver/server.go
package mcpserver

import (
	"sync"

	"yotta/internal/services/container"
	"yotta/internal/services/container/runtime"
	"yotta/internal/services/execution"
)

// Deps 是 main.go 装配时注入的 GUI 常驻标准件 (与 runFunc 用的同一批).
type Deps struct {
	Store       *container.Store
	InputBus    *execution.InputBus
	Matcher     runtime.TemplateMatcher
	Game        runtime.GameProvider
	Clip        runtime.ClipResolver
	MouseCounts func() int      // 取 settings.ActiveMouseCounts360, live
	Armed       func() bool     // 取 settings.MCP.Armed, live
	Busy        func() bool     // worker.IsRunning
}

type Server struct {
	deps   Deps
	runMu  sync.Mutex // 串行化 run_node, 防 AI 并行调用交错输入
}

func NewServer(deps Deps) *Server { return &Server{deps: deps} }
```

- [ ] **Step 2: 迁 schema.go**

把 `cmd/yotta-mcp/schema.go` 整文件复制到 `internal/services/mcpserver/schema.go`，仅改 `package main` → `package mcpserver`。内容（`schemaText`/`schemaExamples`/`graphSchemaJSON`/`graphSchemaVersion`）逐字不动。

- [ ] **Step 3: 迁 authoring.go**

把 `cmd/yotta-mcp/tools.go` 的 `listNodesJSON`/`SaveResult`/`saveContainer`/`validateContainerJSON` 复制到 `internal/services/mcpserver/authoring.go`（`package mcpserver`，import 不变：`yotta/internal/catalog`、`yotta/internal/services/container`、`github.com/google/uuid`）。逻辑逐字不动。

- [ ] **Step 4: 迁 schema 守卫测试**

把 `cmd/yotta-mcp/tools_test.go` 里 `TestSchemaExamples_AllValid`（断言两个样例 `ValidateContainer` 全 clean）复制到 `authoring_test.go`（`package mcpserver`）。**注意**：测试需匿名 import 全 `internal/nodes/*` 触发注册——把 spike `main.go` 那串 `_ "yotta/internal/nodes/..."` 匿名 import 搬进 `authoring_test.go` 的 import 块（test-only 注册，不污染包正常依赖）。

- [ ] **Step 5: 跑测试确认通过**

Run: `go test ./internal/services/mcpserver/ -v`
Expected: PASS（schema 样例校验通过）。

- [ ] **Step 6: 提交**

```bash
git add internal/services/mcpserver/
git commit -m "feat(mcp): mcpserver 包骨架 + authoring 工具迁入 (从 cmd/yotta-mcp 平移)"
```

---

### Task 5: 可跑节点闸 `isRunnable` + `execInPin`

**Files:**
- Create: `internal/services/mcpserver/runnable.go`
- Test: `internal/services/mcpserver/runnable_test.go`

**Interfaces:**
- Consumes: `node.Get(kind) (Registration, bool)`、`node.Spec{ Kind, NeedsWindow, IsPureData, Inputs []InputSpec, Outputs []OutputSpec }`、`node.TypeExec`。
- Produces: `func isRunnable(spec node.Spec) bool`、`func execInPin(spec node.Spec) string`（无 exec 输入返 ""）。

- [ ] **Step 1: 写失败测试**

```go
// 用真实注册表 (匿名 import 全节点; 复用 authoring_test.go 的 import 即可同包生效).
func TestIsRunnable_GatesByNeedsWindowNotPureData(t *testing.T) {
	clickAt, ok := node.Get("ClickAt")
	if !ok { t.Skip("ClickAt 未注册") }
	if !isRunnable(clickAt.Spec) {
		t.Error("ClickAt (NeedsWindow 动作节点) 应可跑")
	}
	if execInPin(clickAt.Spec) == "" {
		t.Error("ClickAt 应有 exec 输入 pin")
	}
	loop, ok := node.Get("Loop")
	if ok && isRunnable(loop.Spec) {
		t.Error("Loop (结构节点) 不该可跑")
	}
	getVar, ok := node.Get("GetVar")
	if ok && isRunnable(getVar.Spec) {
		t.Error("GetVar (IsPureData) 不该可跑")
	}
}
```
> 实现前先 `grep "func Get" internal/node/*.go` 核对 `node.Get` 返回类型字段名（`.Spec`），按实际微调断言取值路径。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/services/mcpserver/ -run TestIsRunnable -v`
Expected: FAIL —— `undefined: isRunnable`。

- [ ] **Step 3: 实现**

```go
// internal/services/mcpserver/runnable.go
package mcpserver

import "yotta/internal/node"

// isRunnable: run_node 只接「对窗口做一件事」的动作节点。闸 = NeedsWindow 且非纯数据。
// 数据驱动 (读 Spec 能力位), 不写死 kind 名单 —— 节点增删自动跟随。
// WindowTarget 例外: 它职责被 find_window 取代, 显式排除。
func isRunnable(spec node.Spec) bool {
	if spec.Kind == "WindowTarget" {
		return false
	}
	return spec.NeedsWindow && !spec.IsPureData
}

// execInPin 返该节点的 exec 输入 pin 名 (Type==Exec 的首个输入); 无则 "".
func execInPin(spec node.Spec) string {
	for _, in := range spec.Inputs {
		if in.Type == node.TypeExec {
			return in.Name
		}
	}
	return ""
}
```
> 若 `node.Spec` 的字段名与此处不符（`Inputs`/`Type`/`Kind`/`NeedsWindow`/`IsPureData`），以 `internal/node/spec.go` 实际为准修正——catalog.go 已确认这些名字成立。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/services/mcpserver/ -run TestIsRunnable -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/services/mcpserver/runnable.go internal/services/mcpserver/runnable_test.go
git commit -m "feat(mcp): isRunnable 可跑节点闸 (NeedsWindow 且非 PureData) + execInPin"
```

---

### Task 6: run_node harness —— 合成微容器 + 跑 + 收割

核心 task。分两层：`buildMicroContainer`（纯函数、易测）+ `runMicroContainer`（跑 + 收割，用预注入 mock 的 rt 测）。

**Files:**
- Create: `internal/services/mcpserver/harness.go`
- Test: `internal/services/mcpserver/harness_test.go`

**Interfaces:**
- Produces:
  - `type RunNodeResult struct{ Ok bool; FiredOutput string; Data map[string]any; Error *RunNodeError }`
  - `type RunNodeError struct{ Code, Message string }`
  - `func buildMicroContainer(kind string, params map[string]any) (*container.Container, string, error)` —— 返微容器 + 目标节点 ID + err（kind 不可跑/无 exec-in → err）。
  - `func runMicroContainer(ctx context.Context, rt *runtime.RuntimeContext, c *container.Container, nodeID string) (RunNodeResult, *node.Image)` —— 跑 + 收割；image 非 nil 表示有图像输出。

- [ ] **Step 1: 写 buildMicroContainer 失败测试**

```go
func TestBuildMicroContainer_WiresStartToExecIn(t *testing.T) {
	c, nodeID, err := buildMicroContainer("ClickAt", map[string]any{"X": 10, "Y": 20})
	if err != nil { t.Fatalf("意外 err: %v", err) }
	if len(c.Graph.Nodes) != 2 { t.Fatalf("应是 Start+目标 两节点, got %d", len(c.Graph.Nodes)) }
	// 边: start.Done → <nodeID>.<execIn>
	if len(c.Graph.Edges) != 1 || c.Graph.Edges[0].From != "start.Done" {
		t.Fatalf("边接线错: %+v", c.Graph.Edges)
	}
	// 目标节点 config.literal 带上了 params
	var target *container.GraphNode
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].ID == nodeID { target = &c.Graph.Nodes[i] }
	}
	lit := target.Config["literal"].(map[string]any)
	if lit["X"] != 10 { t.Fatalf("params 没进 literal: %+v", lit) }

	if _, _, err := buildMicroContainer("Loop", nil); err == nil {
		t.Error("Loop 不可跑, 应返 err")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/services/mcpserver/ -run TestBuildMicroContainer -v`
Expected: FAIL —— `undefined: buildMicroContainer`。

- [ ] **Step 3: 实现 buildMicroContainer**

```go
// internal/services/mcpserver/harness.go
package mcpserver

import (
	"context"
	"errors"

	"yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/container/runtime"
)

type RunNodeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RunNodeResult struct {
	Ok          bool           `json:"ok"`
	FiredOutput string         `json:"firedOutput,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
	Error       *RunNodeError  `json:"error,omitempty"`
}

const microNodeID = "n"

// buildMicroContainer 把单动作节点包成 {Start → 节点} 微容器, 节点参数进 config.literal.
func buildMicroContainer(kind string, params map[string]any) (*container.Container, string, error) {
	rn, ok := node.Get(kind)
	if !ok {
		return nil, "", errors.New("unknown kind")
	}
	if !isRunnable(rn.Spec) {
		return nil, "", errors.New("kind not runnable")
	}
	execIn := execInPin(rn.Spec)
	if execIn == "" {
		return nil, "", errors.New("kind has no exec input")
	}
	if params == nil {
		params = map[string]any{}
	}
	c := &container.Container{
		SchemaVersion: container.CurrentSchemaVersion,
		Name:          "mcp-run-node",
		Graph: container.Graph{
			Version: container.GraphSchemaVersion,
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: microNodeID, Kind: kind, Config: map[string]any{"literal": params}},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: microNodeID + "." + execIn},
			},
		},
	}
	c.Normalize()
	return c, microNodeID, nil
}
```
> `node.Get` 返回值字段（`rn.Spec`）+ Start 节点 exec 出口名（"Done"）：实现前 `grep -r "Kind:.*\"Start\"" internal/nodes/system/` 核对 Start 的 Done 出口名，按实际改。schema.go 样例里 `s.Done` 已佐证 "Done" 成立。

- [ ] **Step 4: 跑 buildMicroContainer 测试确认通过**

Run: `go test ./internal/services/mcpserver/ -run TestBuildMicroContainer -v`
Expected: PASS。

- [ ] **Step 5: 写 runMicroContainer 收割测试（用预注入 mock 的 rt）**

> **关键**：`runMicroContainer` 跑 `NewContainerRunner.Run`，会触发 `setupRuntime`。`setupRuntime` 的幂等逃生口是「`rt.WindowHandle().HWND!=0 && rt.Input!=nil` → 整段跳过」（`runner.go:339`）。所以测试要**预置 window + Input + Capture 三者**，让它跳过真实 backend 创建、用 mock。**实现前先 `grep -rn "rt.Input =\|\.Capture =\|SetActiveWindow\|fakeInput\|mockCapture" internal/services/container/runtime/*_test.go`** 找到 runtime 包测试现成的 mock window/input/capture 注入范式，**复用那套 fixture**（勿手搓 backend mock——接口方法多、易写错）。

```go
func TestRunMicroContainer_HarvestsDetectColorData(t *testing.T) {
	// 复用 runtime 包测试 fixture: 构造一个预注入 mock window+input+capture 的 rt,
	// 让 DetectColor 命中并 .Set("Center", pt). (具体 fixture 名按 Step 前 grep 结果填)
	c, nodeID, _ := buildMicroContainer("DetectColor", map[string]any{/* 命中所需 literal */})
	rt := newTestRTWithMocks(t, c) // ← 复用现成 helper / 仿其写法
	res, img := runMicroContainer(context.Background(), rt, c, nodeID)
	if !res.Ok { t.Fatalf("应成功: %+v", res.Error) }
	if _, ok := res.Data["Center"]; !ok {
		t.Errorf("没收割到 Center: %+v", res.Data)
	}
	_ = img
}
```
> 若 DetectColor 命中所需 mock 太重，退而用更简单的可跑节点（如一个 NeedsWindow 但只 `.Set` 一个标量字段的节点）验证收割路径；核心断言不变：跑完 `res.Data` 里有目标节点 `.Set` 过的字段。

- [ ] **Step 6: 实现 runMicroContainer + 收割**

```go
// runMicroContainer 跑微容器并从 held-output 缓存收割目标节点输出.
// rt 由调用方备好 (生产: NewRuntimeContext + SetActiveWindow; 测试: 预注入 mock).
func runMicroContainer(ctx context.Context, rt *runtime.RuntimeContext, c *container.Container, nodeID string) (RunNodeResult, *node.Image) {
	r := runtime.NewContainerRunner(rt)
	runErr := r.Run(ctx)

	// 收割: execOutputs 里属于本节点的字段.
	prefix := nodeID + "."
	data := map[string]any{}
	var img *node.Image
	for k, v := range r.ExecOutputs() {
		if len(k) <= len(prefix) || k[:len(prefix)] != prefix {
			continue
		}
		field := k[len(prefix):]
		if im, ok := v.(node.Image); ok {
			cp := im
			img = &cp
			continue
		}
		data[field] = v
	}

	if runErr != nil {
		code := "error"
		if errors.Is(runErr, context.DeadlineExceeded) {
			code = "TIMEOUT"
		} else {
			var coded node.Coded
			if errors.As(runErr, &coded) {
				code = string(coded.ErrCode())
			}
		}
		return RunNodeResult{Ok: false, Data: data, Error: &RunNodeError{Code: code, Message: runErr.Error()}}, img
	}
	return RunNodeResult{Ok: true, FiredOutput: firedOutput(c, nodeID, data), Data: data}, img
}

// firedOutput 按节点 spec 反推走了哪个 exec 出口: 收割到的字段属于哪个出口声明的 Data 集.
// best-effort: 无数据字段的出口 (如 NotFound) 推不出 → 返 "" (消费方从空 data 自行判断).
func firedOutput(c *container.Container, nodeID string, data map[string]any) string {
	var kind string
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].ID == nodeID {
			kind = c.Graph.Nodes[i].Kind
		}
	}
	rn, ok := node.Get(kind)
	if !ok {
		return ""
	}
	for _, o := range rn.Spec.Outputs {
		for _, d := range o.Data {
			if _, present := data[d.Name]; present {
				return o.Name
			}
		}
	}
	return ""
}
```
> `node.Coded` / `ErrCode()`：见 `internal/node/errorcodes.go`（error-model.md 已述）。`node.Image{Format,Data}`：见 `internal/node/types.go`。`Spec.Outputs[].Data[].Name`：catalog.go 已确认。

- [ ] **Step 7: 跑收割测试确认通过 + 全包测试**

Run: `go test ./internal/services/mcpserver/ -v`
Expected: PASS。

- [ ] **Step 8: 提交**

```bash
git add internal/services/mcpserver/harness.go internal/services/mcpserver/harness_test.go
git commit -m "feat(mcp): run_node harness — 合成微容器跑 + held-output 收割 (含 Image/firedOutput)"
```

---

### Task 7: MCP 工具 handler —— find_window / list_windows / run_node + 注册

把 §Task 1/4/5/6 串成 MCP 工具，含 arm/busy 闸 + 句柄重校验 + 微容器 validate。

**Files:**
- Create: `internal/services/mcpserver/tools_exec.go`
- Modify: `internal/services/mcpserver/server.go`（加 `Register(*server.MCPServer)`）
- Test: `internal/services/mcpserver/tools_exec_test.go`

**Interfaces:**
- Consumes: `s.deps`（Armed/Busy/Store/InputBus/Matcher/Game/Clip/MouseCounts）、`buildMicroContainer`、`runMicroContainer`、`winutil.{ResolveWindow,WindowMetadata,EnumTopWindows}`、`container.ValidateContainer`。
- Produces: `func (s *Server) runNode(ctx, kind, params, hwnd) (RunNodeResult, *node.Image)` + 三个 mcp handler + `func (s *Server) Register(m *server.MCPServer)`。

- [ ] **Step 1: 写 arm 闸失败测试**

```go
func TestRunNode_NotArmed_Rejects(t *testing.T) {
	s := NewServer(Deps{Armed: func() bool { return false }, Busy: func() bool { return false }})
	res, _ := s.runNode(context.Background(), "ClickAt", map[string]any{"X": 1, "Y": 1}, 0)
	if res.Ok || res.Error == nil || res.Error.Code != "NOT_ARMED" {
		t.Fatalf("未武装应返 NOT_ARMED, got %+v", res)
	}
}

func TestRunNode_Busy_Rejects(t *testing.T) {
	s := NewServer(Deps{Armed: func() bool { return true }, Busy: func() bool { return true }})
	res, _ := s.runNode(context.Background(), "ClickAt", map[string]any{"X": 1, "Y": 1}, 0)
	if res.Error == nil || res.Error.Code != "BUSY" {
		t.Fatalf("GUI 在跑应返 BUSY, got %+v", res)
	}
}

func TestRunNode_UnrunnableKind_Rejects(t *testing.T) {
	s := NewServer(Deps{Armed: func() bool { return true }, Busy: func() bool { return false }})
	res, _ := s.runNode(context.Background(), "Loop", nil, 0)
	if res.Error == nil || res.Error.Code != "UNRUNNABLE_KIND" {
		t.Fatalf("Loop 应返 UNRUNNABLE_KIND, got %+v", res)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/services/mcpserver/ -run TestRunNode_ -v`
Expected: FAIL —— `s.runNode undefined`。

- [ ] **Step 3: 实现 runNode（编排闸 + harness）**

```go
// internal/services/mcpserver/tools_exec.go
package mcpserver

import (
	"context"

	"yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/container/runtime"
	"yotta/pkg/winutil"
)

func errResult(code, msg string) RunNodeResult {
	return RunNodeResult{Ok: false, Error: &RunNodeError{Code: code, Message: msg}}
}

// runNode 编排: arm/busy 闸 → 闸节点 → 句柄重校验 → 合成微容器 → validate → 跑 → 收割.
func (s *Server) runNode(ctx context.Context, kind string, params map[string]any, hwnd uintptr) (RunNodeResult, *node.Image) {
	if s.deps.Armed == nil || !s.deps.Armed() {
		return errResult("NOT_ARMED", "MCP 未武装, 去设置页打开 arm 开关"), nil
	}
	if s.deps.Busy != nil && s.deps.Busy() {
		return errResult("BUSY", "GUI 正在跑容器, 稍后重试"), nil
	}
	c, nodeID, err := buildMicroContainer(kind, params)
	if err != nil {
		return errResult("UNRUNNABLE_KIND", err.Error()), nil
	}
	// 句柄重校验 (HWND 可能被 OS 复用 / 窗口已关).
	wh, err := winutil.WindowMetadata(hwnd)
	if err != nil {
		return errResult("WINDOW_GONE", err.Error()), nil
	}
	// 参数校验 (缺必填 / 类型非法). 豁免 MISSING_WINDOW_TARGET — 微容器无 WindowTarget
	// 节点, NeedsWindow 节点必触发它; 但窗口经 hwnd 带外提供, 这条结构校验不适用.
	if hasBlockingValidationError(c) {
		return errResult("INVALID_PARAMS", "params 校验未过 (详见节点 spec)"), nil
	}
	// 串行化 run_node, 防 AI 并行交错输入.
	s.runMu.Lock()
	defer s.runMu.Unlock()

	rt := runtime.NewRuntimeContext(
		c, s.deps.InputBus, s.deps.Matcher, s.deps.Game,
		func(string, any) {}, // no-op emit: 收割走 execOutputs, 不靠事件
		s.deps.Clip, s.deps.MouseCounts(),
	)
	rt.SetActiveWindow(wh)
	return runMicroContainer(ctx, rt, c, nodeID)
}
```
> `winutil.WindowMetadata(0)` 对 hwnd==0 返 err（已确认 `window.go:305`）→ 测试里 hwnd=0 走 WINDOW_GONE，但 arm/busy/unrunnable 闸在它**之前**，故那三个测试 hwnd=0 也能命中各自 code（闸先短路）。`container.SeverityError`：见 spike `tools.go` 已用。

- [ ] **Step 4: 跑闸测试确认通过**

Run: `go test ./internal/services/mcpserver/ -run TestRunNode_ -v`
Expected: PASS（三个 reject 分支）。

- [ ] **Step 5: 加三个 mcp handler + Register**

`server.go` 加（import `github.com/mark3labs/mcp-go/mcp`、`github.com/mark3labs/mcp-go/server`、`encoding/json`、`fmt`）:
```go
// Register 把全部工具挂到 MCPServer (authoring 复用迁入的纯函数; execution 新增).
func (s *Server) Register(m *server.MCPServer) {
	// --- authoring (只读/写图, 不受 arm 闸; save_container 受 arm 闸) ---
	m.AddTool(mcp.NewTool("list_nodes", mcp.WithDescription("List all Yotta node kinds with pins/types/required/defaults/category/capability flags. The building blocks; run_node executes one of these against a window.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(string(listNodesJSON())), nil
		})
	m.AddTool(mcp.NewTool("get_graph_schema", mcp.WithDescription("Yotta container-graph JSON schema + validated examples. Read before generating a container.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(string(graphSchemaJSON())), nil
		})
	m.AddTool(mcp.NewTool("validate_container", mcp.WithDescription("Validate a container graph JSON. Returns []ValidationError (empty=clean)."),
		mcp.WithString("container", mcp.Required(), mcp.Description("Container graph JSON."))),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			raw, err := req.RequireString("container")
			if err != nil || raw == "" {
				return mcp.NewToolResultError("missing 'container'"), nil
			}
			out, _ := validateContainerJSON([]byte(raw))
			return mcp.NewToolResultText(string(out)), nil
		})
	m.AddTool(mcp.NewTool("save_container", mcp.WithDescription("Validate + persist a container graph (rejects on error-level issues). Server assigns id. Requires MCP armed."),
		mcp.WithString("container", mcp.Required(), mcp.Description("Container graph JSON."))),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if s.deps.Armed == nil || !s.deps.Armed() {
				return mcp.NewToolResultError("NOT_ARMED: 去设置页打开 arm 开关"), nil
			}
			raw, err := req.RequireString("container")
			if err != nil {
				return mcp.NewToolResultError("missing 'container'"), nil
			}
			res, saveErrs := saveContainer(s.deps.Store, []byte(raw))
			if saveErrs != nil {
				return mcp.NewToolResultError(string(saveErrs)), nil
			}
			b, _ := json.MarshalIndent(res, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		})

	// --- window (只读, 不受 arm 闸) ---
	m.AddTool(mcp.NewTool("list_windows", mcp.WithDescription("List all top-level visible windows: {hwnd,title,class,processName,pid,clientW,clientH}. Pick a target, pass its hwnd to run_node.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			b, _ := json.MarshalIndent(winutil.EnumTopWindows(), "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		})
	m.AddTool(mcp.NewTool("find_window", mcp.WithDescription("Resolve the first top-level window matching title/class/processName. Returns its handle (hwnd + metadata)."),
		mcp.WithString("title", mcp.Description("Window title (exact unless titleMatch=regex).")),
		mcp.WithString("class", mcp.Description("Window class (exact).")),
		mcp.WithString("processName", mcp.Description("Process exe basename (case-insensitive).")),
		mcp.WithString("titleMatch", mcp.Description("'exact' (default) or 'regex'."))),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			spec := winutil.MatchSpec{
				Title:       req.GetString("title", ""),
				Class:       req.GetString("class", ""),
				ProcessName: req.GetString("processName", ""),
				TitleMatch:  req.GetString("titleMatch", "exact"),
			}
			wh, err := winutil.ResolveWindow(ctx, spec, 3*time.Second, 200*time.Millisecond)
			if err != nil {
				return mcp.NewToolResultError("WINDOW_NOT_FOUND: " + err.Error()), nil
			}
			b, _ := json.MarshalIndent(wh, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		})

	// --- run_node (受 arm + busy 闸) ---
	m.AddTool(mcp.NewTool("run_node", mcp.WithDescription("Execute ONE action node (kind from list_nodes, NeedsWindow only) against a window once. params = {pinName: literal}. Returns {ok,firedOutput,data,error}; image outputs returned as an image block. Requires MCP armed. Use this to probe (Capture to see, DetectColor for coords, ClickAt to test), then bake findings into save_container."),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Node kind, e.g. ClickAt / Capture / DetectColor.")),
		mcp.WithString("window", mcp.Required(), mcp.Description("Target window hwnd (decimal uintptr from find_window/list_windows).")),
		mcp.WithObject("params", mcp.Description("Input pin literals {pinName: value}."))),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			kind, err := req.RequireString("kind")
			if err != nil {
				return mcp.NewToolResultError("missing 'kind'"), nil
			}
			hwndStr, _ := req.RequireString("window")
			hwnd64, perr := strconv.ParseUint(hwndStr, 10, 64)
			if perr != nil {
				return mcp.NewToolResultError("invalid 'window' (expect decimal uintptr)"), nil
			}
			params := req.GetArguments()["params"]
			pm, _ := params.(map[string]any)
			res, img := s.runNode(ctx, kind, pm, uintptr(hwnd64))
			b, _ := json.MarshalIndent(res, "", "  ")
			if img != nil {
				mime := "image/png"
				if img.Format == "jpeg" {
					mime = "image/jpeg"
				}
				return mcp.NewToolResultImage(string(b), base64.StdEncoding.EncodeToString(img.Data), mime), nil
			}
			return mcp.NewToolResultText(string(b)), nil
		})
}
```
> **实现前核对 mcp-go v0.32.0 的 request 取参 API**：`req.RequireString`/`req.GetString`/`req.GetArguments`/`mcp.WithObject` 的确切名字（spike 已用 `RequireString`；`GetString`/`WithObject` 需 `grep` 库 `mcp/*.go` 核对，不符则用 `req.GetArguments()` map 手取）。imports 补 `time`/`strconv`/`encoding/base64`。

- [ ] **Step 6: 跑全包测试 + build**

Run: `go test ./internal/services/mcpserver/ -v && go build ./...`
Expected: PASS + build 绿。

- [ ] **Step 7: 提交**

```bash
git add internal/services/mcpserver/tools_exec.go internal/services/mcpserver/server.go internal/services/mcpserver/tools_exec_test.go
git commit -m "feat(mcp): find_window/list_windows/run_node handler + arm/busy 闸 + Register"
```

---

### Task 8: `main.go` —— 装配并启动 Streamable HTTP server

**Files:**
- Modify: `main.go`（在 worker 装配后、约 line 298 附近）

**Interfaces:**
- Consumes: `mcpserver.NewServer`/`Server.Register`、`server.NewMCPServer`/`server.NewStreamableHTTPServer`、worker/containerStore/inputBus/templateMatcher/newGameProviderAdapter/clipSvc/app。

- [ ] **Step 1: 装配 + 启动（worker.Start() 之后插入）**

```go
	// MCP 对外暴露 server (③): 复用执行标准件, 后台起 Streamable HTTP.
	mcpSrv := mcpserver.NewServer(mcpserver.Deps{
		Store:       containerStore,
		InputBus:    inputBus,
		Matcher:     templateMatcher,
		Game:        newGameProviderAdapter(),
		Clip:        clipSvc,
		MouseCounts: func() int { return app.Settings().ActiveMouseCounts360() },
		Armed:       func() bool { return app.Settings().MCP.Armed },
		Busy:        worker.IsRunning,
	})
	mcpCore := server.NewMCPServer("yotta-mcp", "0.1.0")
	mcpSrv.Register(mcpCore)
	mcpHTTP := server.NewStreamableHTTPServer(mcpCore)
	go func() {
		if err := mcpHTTP.Start("127.0.0.1:8765"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			rootLog.Warn().Err(err).Str("tag", "MCP").Msg("MCP HTTP server 退出")
		}
	}()
	rootLog.Info().Str("tag", "MCP").Msg("MCP server: http://127.0.0.1:8765/mcp")
```
> imports 补 `"net/http"`、`mcpserver "yotta/internal/services/mcpserver"`、mcp-go 的 `server "github.com/mark3labs/mcp-go/server"`。`app.Settings()` 返回值类型确认有 `.MCP`（Task 3 已加）+ `.ActiveMouseCounts360()`（runFunc 已用）。端口 `8765` 固定（v1 不做可配，YAGNI；若占用冲突再议）。

- [ ] **Step 2: 优雅关停（找到 app 退出/shutdown 钩子处，挂 Shutdown）**

> `grep -n "OnShutdown\|Shutdown\|defer.*Close\|app.Run" main.go` 找到 Wails app 生命周期关停点；在那调：
```go
	// app 退出时关停 MCP server (5s 超时).
	ctxSd, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = mcpHTTP.Shutdown(ctxSd)
```
> 若 main.go 无显式 shutdown 钩子（Wails 进程随窗口关直接退），则 server goroutine 随进程退即可——此 step **可省**，但要在计划执行时确认 Wails 关停语义后决定（不强加一个不存在的钩子）。

- [ ] **Step 3: build + 真机起服务确认**

Run: `go build ./...`
Expected: build 绿。

Run（真机，按 build.md 起应用）：启动 GUI，日志应出现 `MCP server: http://127.0.0.1:8765/mcp`；`curl -s http://127.0.0.1:8765/mcp`（或 MCP inspector）能握手列出工具。
Expected: server 起得来、`list_nodes` 可调。（headless CI 不验此 step，标人工 smoke。）

- [ ] **Step 4: 提交**

```bash
git add main.go
git commit -m "feat(mcp): main.go 装配并启动 Streamable HTTP MCP server (注入执行标准件)"
```

---

### Task 9: 前端设置页 MCP tab（arm 开关 + URL 展示）

**Files:**
- Create: `frontend/src/views/SettingsMCP.vue`
- Modify: `frontend/src/views/SettingsView.vue`（加 tab）
- Modify: `frontend/src/stores/settings.ts`（`Settings` interface 加 `mcp`）
- Modify: `frontend/src/i18n/{zh,en}.ts`

**Interfaces:**
- Consumes: `backend.settings.update(patch)`（复用，无新 RPC）、`stores/settings.ts` 的 settings store。

- [ ] **Step 1: settings store 加 mcp 段**

`frontend/src/stores/settings.ts` 的 `Settings` interface（手镜像）加：
```ts
  mcp: { armed: boolean }
```
（与现有 `ai` 段手镜像同风格；字段名 Go `Armed`→JSON/TS `armed`。）

- [ ] **Step 2: SettingsMCP.vue（照 settings-page-style 范式）**

> 照 `flightdeck/checklists/2026-06-06-settings-page-style.md` + 现有 `SettingsAI.vue`/`SettingsGeneral.vue` 范式写。内容最小：
> - 一个 **arm 开关**（toggle），`@change` → `backend.settings.update({mcp:{armed:<v>}})`（不 v-model 双绑只读 store）。
> - 武装时显眼**红/橙警示**文案：「AI 可驱动你的鼠标键盘与写入容器」。
> - 一行**只读 URL 展示** `http://127.0.0.1:8765/mcp` + 「把它配进 AI 客户端（Claude Desktop / Cline）」说明，带复制按钮。
> - 文案全走 i18n（见 Step 4），URL 是字面常量。

- [ ] **Step 3: SettingsView.vue 接 tab**

`tabs` 加 `{ key:'mcp', label:t('settingsTab.mcp') }` + `<SettingsMCP v-if="activeTab==='mcp'"/>` + import + `TabKey` 加 `'mcp'`。

- [ ] **Step 4: i18n（zh/en 对称）**

`zh.ts`/`en.ts` 加 `settingsTab.mcp` + `settingsMCP.{title,armLabel,armWarning,urlLabel,urlHint,copy}`。**文案值禁 `{}` `@` `|`**（i18n 保留字符）。zh/en 键集必须对称（`node src/i18n/check.cjs` 兜）。

- [ ] **Step 5: 校验**

Run: `cd frontend && ./node_modules/.bin/vue-tsc --noEmit && node src/i18n/check.cjs`
Expected: vue-tsc 绿（除预存基线）；i18n check 不新增 residue（保持基线 42）。

- [ ] **Step 6: 提交**

```bash
git add frontend/src/views/SettingsMCP.vue frontend/src/views/SettingsView.vue frontend/src/stores/settings.ts frontend/src/i18n/zh.ts frontend/src/i18n/en.ts
git commit -m "feat(mcp): 设置页 MCP tab — arm 开关 + server URL 展示"
```

---

### Task 10: 退役 `cmd/yotta-mcp` 独立进程

逻辑已迁入 `internal/services/mcpserver`（Task 4），独立 stdio 进程不再需要。

**Files:**
- Delete: `cmd/yotta-mcp/`（整目录：main.go / tools.go / schema.go / tools_test.go）

- [ ] **Step 1: 确认无其他引用**

Run: `grep -rn "cmd/yotta-mcp\|yotta-mcp" --include=*.go . ; grep -rn "yotta-mcp" Taskfile* *.md 2>/dev/null`
Expected: 除文档外无源码 import（cmd 是独立 main，不会被 import；确认构建脚本/Taskfile 无单独 build 它的 target，有则一并删）。

- [ ] **Step 2: 删目录**

```bash
git rm -r cmd/yotta-mcp/
```

- [ ] **Step 3: build 确认无断**

Run: `go build ./... && go test ./internal/services/mcpserver/ -v`
Expected: build 绿、mcpserver 测试仍 PASS（守卫测试已随迁入存活）。

- [ ] **Step 4: 提交**

```bash
git add -A cmd/
git commit -m "chore(mcp): 退役 cmd/yotta-mcp 独立进程 (逻辑已迁入 internal/services/mcpserver)"
```

---

## 收尾验证（全部 task 后）

- [ ] **全量构建 + 测试**：`go build ./... && go test ./internal/services/mcpserver/ ./internal/services/container/... ./pkg/winutil/ ./internal/services/execution/`
      —— 绿（runtime 缺 fish fixture 等预存基线除外，照 build.md 判）。
- [ ] **前端**：`cd frontend && ./node_modules/.bin/vue-tsc --noEmit && node src/i18n/check.cjs`（除基线外绿）。
- [ ] **真机 smoke（人工，按 build.md 起应用，须重启后端重生成 bindings）**：
  1. 设置页出现 MCP tab，arm 开关默认关。
  2. AI 客户端连 `http://127.0.0.1:8765/mcp`，`list_nodes`/`list_windows`/`find_window` 不武装也能调。
  3. 未武装时 `run_node`/`save_container` 返 `NOT_ARMED`。
  4. 武装后：`find_window` 拿记事本句柄 → `run_node` 跑 `Capture` 拿到截图（图像块）→ 跑 `ClickAt` 在记事本里点一下（可见生效）→ `save_container` 存一张图，GUI 列表里出现。
  5. GUI ▶ 跑一个容器期间调 `run_node` → 返 `BUSY`。

> 真机 smoke 项进 cockpit `## Pending Review`（与 ①余项 smoke 并列），单测背书、待用户顺手点。

## Self-Review 记录

- **Spec 覆盖**:§3 架构→Task 4/8;§4 工具面→Task 4(authoring)/7(window+run_node);§5 run_node 机制→Task 6;§6 闸→Task 5;§7 窗口工具→Task 1/7;§8 transport→Task 8;§9 安全 arm→Task 3/7;§10 错误模型→Task 6/7;§11 触点清单逐条对上;§12 测试策略→各 task TDD + 收尾 smoke。**无遗漏**。
- **类型一致**:`Deps`/`Server`/`RunNodeResult`/`RunNodeError`/`runNode`/`buildMicroContainer`/`runMicroContainer` 跨 task 签名一致;`mcpserver` 包名贯穿。
- **已标注「实现前核对」点**(非占位,是铁律防脑补):`NewWorker` 入参、`node.Get`/`node.Spec` 字段、Start 的 Done 出口名、mcp-go request 取参 API、Wails 关停钩子——均给了 grep 锚点,实证后再落码。
