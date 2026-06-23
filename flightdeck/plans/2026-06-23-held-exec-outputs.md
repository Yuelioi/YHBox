---
status: done
summary: TDD 实现 held exec output 缓存: ContainerRunner.execOutputs(per-run, 键 nodeID.field) 在 routeResult 两处 applyCaptures 旁写; pullDataPin 对 exec 出口字段改读缓存(替原 nil); 删 applyExecDataEdges 收编单跳并给 buildDataWireFor 动态分支补 coerceToType; 删 validateExecDataAdjacency + EXEC_DATA_NOT_ADJACENT i18n; 反转跨跳警告测试为端到端直连; 子图键唯一性 + fan-out + loop + 稀疏 + 回归(Fail.Code/vision) 全测。
last_updated: 2026-06-23
implements: specs/2026-06-23-held-exec-outputs.md
---

# held exec outputs (免 GetVar 任意距离直连) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline, this session — context is hot) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 exec 节点出口的 Data 字段获得 UE 式 "held output" 语义 —— fire 时自动存进本次运行缓存，下游数据线任意距离直连读，去掉 GetVar 与紧邻约束。

**Architecture:** `ContainerRunner` 持一张 per-run `execOutputs map[string]any`（键 `"<nodeID>.<field>"`）。`routeResult` fire 出口时（成功 `OutputData` + 失败 `failData`）在 `applyCaptures` 旁写缓存；`pullDataPin` 解析数据线源为 exec 出口字段时改读缓存（替代原先返 `nil` 等单跳）。缓存是单跳 `applyExecDataEdges` 的超集 → 删单跳机制 + 删 `EXEC_DATA_NOT_ADJACENT` 编辑期警告。

**Tech Stack:** Go (`internal/services/container/runtime` + `internal/services/container`)；Vue/TS i18n（`frontend/src/i18n`）。测试 = Go `testing` + 前端 `vitest`/`vue-tsc`/i18n parity 脚本。

## Progress

current: done — ready to land。Task 1-6 全落 (6 commits)，全量验证仅余预存基线红，FE 类型/i18n parity 绿。实测结论回填 spec §11。

## Global Constraints

- **未发布，不要兼容**：删 `applyExecDataEdges` / `validateExecDataAdjacency` / `EXEC_DATA_NOT_ADJACENT` 一次切净，不留 deprecated / shim / fallback；该改的测试直接改。
- **键唯一性已证**：production node ID = `kind_<6位base36随机>`（`frontend/src/composables/containerEditor/ids.ts`），容器内全局唯一，**flat 共享 map 键 `nodeID.field` 无碰撞** —— 不做 per-frame 命名空间（子图测试是安全网，不是前置条件）。
- **保留 `execData` → `RunNode` 参数**：本 spec 只删 `applyExecDataEdges`（数据线桥），不删 token.ExecData / `buildExecDataFor` / `RunNode` 的 `execData` 入参（那是 `in.X("k")` 原始 key-match 路径，正交）。
- **`exec 节点永不因 pull 重跑**：只读已存值，不触发副作用重放。
- **预存失败基线**（跑 `go test` 判红按此，见 `checklists/build.md`）：`TestApplyDirection_*` · `TestWatchdog_*` · `TestScanSubgraphDependencies_*` · `TestFishingV2Main_StateCycleSmoke` 本机恒红、非回归。
- **命令**：Go 语法 `go build ./...`；Go 测试 `go test ./internal/services/container/...`；FE 类型 `cd frontend && ./node_modules/.bin/vue-tsc --noEmit`；i18n parity `cd frontend && node src/i18n/check.cjs`。
- **TDD + 频繁本地 commit；永不 push。**

## File Structure

| 文件 | 职责 | 改动 |
|---|---|---|
| `internal/services/container/runtime/runner.go` | ContainerRunner 结构 + 构造 | 加 `execOutputs` 字段 + 构造初始化 |
| `internal/services/container/runtime/dispatch_v5.go` | dispatch / fire 路由 | 加 `captureExecOutputs`；routeResult 两处接入；**删 `applyExecDataEdges` + 2 调用**；buildDataWireFor 动态分支补 coerce |
| `internal/services/container/runtime/data_pull.go` | 数据线求值 | pullDataPin exec 出口分支改读 `execOutputs` |
| `internal/services/container/validator.go` | 编辑期校验 | **删 `validateExecDataAdjacency` + 调用(line 170)** |
| `frontend/src/i18n/zh.ts` · `en.ts` | 校验码文案 | **删 `EXEC_DATA_NOT_ADJACENT`** |
| `internal/services/container/ai_capture_repro_test.go` | 跨跳场景测试 | 反转：跨跳**不再**报警告 |
| `internal/services/container/runtime/held_output_test.go` | 新建 | held output 语义全覆盖（fan-out / 稀疏 / 未fire / loop / subgraph） |

---

### Task 1: per-run 缓存字段 + fire 写钩子

**Files:**
- Modify: `internal/services/container/runtime/runner.go:92-109`（struct）+ `:125-132`（构造）
- Modify: `internal/services/container/runtime/dispatch_v5.go:184-207`（captureExecOutputs 新方法 + routeResult 两处接入 `:289`、`:301`）
- Test: `internal/services/container/runtime/dispatch_v5_test.go`

**Interfaces:**
- Produces: `ContainerRunner.execOutputs map[string]any`（键 `"<nodeID>.<field>"`）；`(r *ContainerRunner) captureExecOutputs(node *container.GraphNode, data map[string]any)`。Task 2 的 pullDataPin 读这张 map。

- [ ] **Step 1: 写失败测试**（fire 出口字段进缓存）

加到 `dispatch_v5_test.go`：

```go
// TestCaptureExecOutputs_WritesPerRunCache — exec 节点 fire 出口时, OutputData 各字段进 execOutputs.
func TestCaptureExecOutputs_WritesPerRunCache(t *testing.T) {
	resetTdEcho()
	c := &container.Container{
		SchemaVersion: 1, ID: "test-execoutputs-write",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "n1", Kind: tkFailf},
				{ID: "sink", Kind: tkEcho},
			},
			Edges: []container.GraphEdge{
				{From: "n1.Fail", To: "sink.in"}, // 接线 → Fail 路由 handled → 写 failData 缓存
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	if _, err := r.execNodeViaFramework(context.Background(), r.nodesByID["n1"], ExecToken{NodeID: "n1", InPin: "in"}); err != nil {
		t.Fatalf("n1 fail-route: %v", err)
	}
	if got := r.execOutputs["n1.Code"]; got != "capture_failed" {
		t.Errorf("execOutputs[n1.Code] = %v, want capture_failed", got)
	}
	if _, ok := r.execOutputs["n1.Error"]; !ok {
		t.Errorf("execOutputs 缺 n1.Error")
	}
}
```

- [ ] **Step 2: 跑测试确认 FAIL**

Run: `go test ./internal/services/container/runtime/ -run TestCaptureExecOutputs_WritesPerRunCache -v`
Expected: 编译失败 `r.execOutputs undefined` / `r.captureExecOutputs undefined`。

- [ ] **Step 3: 加字段 + 构造初始化**

`runner.go` struct（`stopwatches` 字段后加）：

```go
	stopwatches *stopwatchTable

	// execOutputs per-run held output 缓存: exec 节点 fire 某出口时, 该出口 OutputData 每个字段
	// 写进 execOutputs["<nodeID>.<field>"]; 下游数据线经 pullDataPin 任意距离直连读 (免 GetVar、
	// 免紧邻). 键全局唯一 (node ID 含随机后缀), 主图/子图/listener 共用一张. per-run 生命周期 =
	// runner 实例 (NewContainerRunner 起一张新的, 一次 Run 用完即随实例释放).
	execOutputs map[string]any
```

`NewContainerRunner` 的 `r := &ContainerRunner{...}` 里加：

```go
	r := &ContainerRunner{
		rt:          rt,
		compiled:    cc,
		nodesByID:   cc.Main.NodesByID,
		edges:       cc.Main.Edges,
		dataEdges:   cc.Main.DataEdges,
		stopwatches: newStopwatchTable(),
		execOutputs: map[string]any{},
	}
```

- [ ] **Step 4: 加 captureExecOutputs + 接入 routeResult**

`dispatch_v5.go`，在 `applyCaptures` 方法后加：

```go
// captureExecOutputs 路径②(held output): fire 时把本次出口 OutputData 的每个字段写进 per-run
// 缓存 execOutputs["<nodeID>.<field>"], 供下游数据线任意距离直连读 (pullDataPin). 稀疏: 只写
// 本次 fire 实际带的字段, 未带的保留上次值 (同 applyCaptures 语义). 与 applyCaptures 并列、互不依赖.
func (r *ContainerRunner) captureExecOutputs(node *container.GraphNode, data map[string]any) {
	for field, v := range data {
		r.execOutputs[node.ID+"."+field] = v
	}
}
```

`routeResult` 失败分支 `r.applyCaptures(node, failData)` 后加一行：

```go
			r.applyCaptures(node, failData) // 路径①: Fail 出口 Error/Code → 绑定变量 (如 PlayClip)
			r.captureExecOutputs(node, failData) // 路径②: Fail 出口 → held 缓存
			return r.edges.nextWithData(node.ID+".Fail", tok.LoopStack, failData), nil
```

`routeResult` 成功分支 `r.applyCaptures(node, result.OutputData)` 后加一行：

```go
	r.applyCaptures(node, result.OutputData) // 路径①: 出口 Data 字段 → 绑定变量
	r.captureExecOutputs(node, result.OutputData) // 路径②: 出口 Data 字段 → held 缓存
	tokens := r.edges.nextWithData(node.ID+"."+result.ExitName, tok.LoopStack, result.OutputData)
```

- [ ] **Step 5: 跑测试确认 PASS**

Run: `go test ./internal/services/container/runtime/ -run TestCaptureExecOutputs_WritesPerRunCache -v`
Expected: PASS。

- [ ] **Step 6: commit**

```bash
git add internal/services/container/runtime/runner.go internal/services/container/runtime/dispatch_v5.go internal/services/container/runtime/dispatch_v5_test.go
git commit -m "feat(runtime): per-run held-output cache, written on exec fire"
```

---

### Task 2: pullDataPin 读缓存（跨跳直连）

**Files:**
- Modify: `internal/services/container/runtime/data_pull.go:73-96`（pullDataPin exec 出口分支）
- Test: `internal/services/container/runtime/dispatch_v5_test.go`

**Interfaces:**
- Consumes: Task 1 的 `r.execOutputs`。
- Produces: pullDataPin 对 `IsExecOutputDataFieldNode` 源返缓存值（命中）或 `nil`（未命中）。

- [ ] **Step 1: 写失败测试**（无 execData 的 token 也能读到值 = 跨跳/非紧邻语义）

加到 `dispatch_v5_test.go`：

```go
// TestHeldOutput_CrossHopReadsCache — 源 fire 写缓存后, 一个 token 不带 ExecData 的消费者
// (模拟非紧邻 / 跨跳: applyExecDataEdges 无可注入) 仍经缓存读到 n1.Code → Value.
func TestHeldOutput_CrossHopReadsCache(t *testing.T) {
	resetTdEcho()
	c := &container.Container{
		SchemaVersion: 1, ID: "test-heldoutput-crosshop",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "n1", Kind: tkFailf},
				{ID: "sink", Kind: tkEcho},
			},
			Edges: []container.GraphEdge{
				{From: "n1.Fail", To: "sink.in"},    // exec 接线 (n1 Fail 路由 handled)
				{From: "n1.Code", To: "sink.Value"}, // data 边: n1.Code → sink.Value
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	// 1) 跑 n1 → 写 execOutputs[n1.Code].
	if _, err := r.execNodeViaFramework(context.Background(), r.nodesByID["n1"], ExecToken{NodeID: "n1", InPin: "in"}); err != nil {
		t.Fatalf("n1: %v", err)
	}
	// 2) 跑 sink, token 不带 ExecData → 仅缓存能交付.
	if _, err := r.execNodeViaFramework(context.Background(), r.nodesByID["sink"], ExecToken{NodeID: "sink", InPin: "in"}); err != nil {
		t.Fatalf("sink: %v", err)
	}
	if got := getTdEchoLast(); got != "capture_failed" {
		t.Errorf("sink.Value = %v, want capture_failed (held 缓存交付, token 无 execData)", got)
	}
}
```

- [ ] **Step 2: 跑测试确认 FAIL**

Run: `go test ./internal/services/container/runtime/ -run TestHeldOutput_CrossHopReadsCache -v`
Expected: FAIL — `sink.Value = <nil>`（旧行为：exec 出口分支返 nil，token 无 execData → 无注入）。

- [ ] **Step 3: pullDataPin 改读缓存**

`data_pull.go`，把 line 75-83 的 exec 出口分支改为：

```go
	// 1. Data edge lookup
	if srcID, srcPin := r.dataEdges.Source(nodeID, pinName); srcID != "" {
		// 源是 exec 出口的 Data 字段 (Fail.Code / AI red·white / Capture.Image): 不走 pure-data
		// 重算, 读 per-run held output 缓存 (源 fire 时 captureExecOutputs 写). 任意距离直连、
		// 免 GetVar、免紧邻; 未命中 (源还没 fire) → nil, 消费方走默认.
		if n := r.nodesByID[srcID]; n != nil && container.IsExecOutputDataFieldNode(n, srcPin) {
			if v, ok := r.execOutputs[srcID+"."+srcPin]; ok {
				return toExprValue(v), nil
			}
			return nil, nil
		}
		return r.evalDataSource(ctx, srcID, srcPin)
	}
```

- [ ] **Step 4: 跑测试确认 PASS**

Run: `go test ./internal/services/container/runtime/ -run TestHeldOutput_CrossHopReadsCache -v`
Expected: PASS。

- [ ] **Step 5: 回归 — 原单跳测试仍绿**（此刻 applyExecDataEdges 仍在；缓存是超集）

Run: `go test ./internal/services/container/runtime/ -run TestExecDataEdge_FailCodeIntoDataPin -v`
Expected: PASS。

- [ ] **Step 6: commit**

```bash
git add internal/services/container/runtime/data_pull.go internal/services/container/runtime/dispatch_v5_test.go
git commit -m "feat(runtime): pullDataPin reads held-output cache (any-distance direct connect)"
```

---

### Task 3: 收编单跳 — 删 applyExecDataEdges + 动态输入补 coerce

**Files:**
- Modify: `internal/services/container/runtime/dispatch_v5.go`（删 `applyExecDataEdges` 函数 `:89-126` + 2 调用 `:39`、`:350`；buildDataWireFor 动态分支 `:83` 补 coerce）
- Test: 复用 `TestExecDataEdge_FailCodeIntoDataPin`（静态）+ `ai_capture_repro` 系列（vision 动态 Image）

**Interfaces:**
- Consumes: Task 2 的缓存读路径（删单跳后唯一交付路径）。
- Produces: 单跳 exec-data 数据线桥彻底由缓存承接；动态输入保留 `coerceToType`。

- [ ] **Step 1: 删 applyExecDataEdges 的 2 个调用**

`dispatch_v5.go` `execNodeViaFramework`（约 :36-39）删掉 applyExecDataEdges 行：

```go
	dataWire := r.buildDataWireFor(ctx, node, rn)
	config := r.buildConfigFor(node)
	execData := r.buildExecDataFor(tok)

	result := nodepkg.RunNode(ctx, rn, dataWire, config, execData, r.bundle, node.LogEnabled)
```

`execNodeAsRegionViaFramework`（约 :347-352）同样删 applyExecDataEdges 行：

```go
	dataWire := r.buildDataWireFor(ctx, node, rn)
	config := r.buildConfigFor(node)
	execData := r.buildExecDataFor(tok)

	result := nodepkg.RunNodeAsRegion(ctx, rn, dataWire, config, execData, r.bundle, node.LogEnabled, body)
```

- [ ] **Step 2: 删 applyExecDataEdges 函数定义**（`:89-126` 整段移除）

- [ ] **Step 3: buildDataWireFor 动态分支补 coerce**

`buildDataWireFor` 动态输入分支（`:79-84`）把 `dw[in.Name] = v` 改为带 coerce（原由 applyExecDataEdges 承担）：

```go
		for _, in := range container.ParseDynamicInputDecls(node) {
			if in.Name == "" || static[in.Name] {
				continue
			}
			if _, exists := dw[in.Name]; exists {
				continue
			}
			v, err := r.resolveDataPinV5(ctx, node.ID, in.Name)
			if err != nil || v == nil {
				continue
			}
			dw[in.Name] = coerceToType(v, in.Type) // 删 applyExecDataEdges 后, 动态输入的 coerce 移到此处
		}
```

- [ ] **Step 4: 语法 + 回归测试**

Run: `go build ./...`
Expected: 通过（无 `applyExecDataEdges` 残留引用）。

Run: `go test ./internal/services/container/... -run 'TestExecDataEdge_FailCodeIntoDataPin|AICapture|AINode_ExecData' -v`
Expected: `TestExecDataEdge_FailCodeIntoDataPin` PASS（现经缓存）；AI capture / vision 相关 PASS（动态 Image 经缓存 + coerce 不变）。

> 若 `ParseDynamicInputDecls` 返回的 decl 字段名不是 `.Type`（核对 applyExecDataEdges 原 `d.Type` 用法已确认是 `.Type`），按实际字段名调整 —— 这是 Task 3 内唯一需对源核实的符号。

- [ ] **Step 5: commit**

```bash
git add internal/services/container/runtime/dispatch_v5.go
git commit -m "refactor(runtime): drop single-hop applyExecDataEdges, subsumed by held-output cache"
```

---

### Task 4: 删紧邻警告 — validateExecDataAdjacency + EXEC_DATA_NOT_ADJACENT

**Files:**
- Modify: `internal/services/container/validator.go`（删 `validateExecDataAdjacency` `:375-423` + 调用 `:170`）
- Modify: `frontend/src/i18n/zh.ts:1817` · `frontend/src/i18n/en.ts:1797`（删 `EXEC_DATA_NOT_ADJACENT`）
- Test: `internal/services/container/ai_capture_repro_test.go`（反转）

**Interfaces:**
- Consumes: 无（纯删除 + 测试反转）。
- Produces: 跨跳 exec-data 数据线编辑期不再报警告。

- [ ] **Step 1: 反转测试**（先改测试 → 红 → 删代码 → 绿）

`ai_capture_repro_test.go` 把 `TestAINode_ExecDataNonAdjacentWarns` 整体替换为（保留原 fixture 的 Nodes/Edges 不动，只改断言与函数名）：

```go
// 跨跳 exec-data 数据线 (ai.white→log2) 现在合法 —— held output 缓存任意距离直连,
// 不再报 EXEC_DATA_NOT_ADJACENT (约束消失).
func TestAINode_ExecDataNonAdjacentNoWarn(t *testing.T) {
	c := &Container{
		// ... 保留原 fixture 的 Nodes / Edges 原样 ...
	}
	for _, e := range ValidateContainer(c, nil) {
		if e.Code == "EXEC_DATA_NOT_ADJACENT" {
			t.Fatalf("跨跳 exec-data 已合法化, 不该再报 EXEC_DATA_NOT_ADJACENT, got %+v", e)
		}
	}
}
```

- [ ] **Step 2: 跑测试确认 FAIL**

Run: `go test ./internal/services/container/ -run TestAINode_ExecDataNonAdjacent -v`
Expected: FAIL（validateExecDataAdjacency 仍在 → 跨跳仍报 EXEC_DATA_NOT_ADJACENT → Fatalf 触发）。

- [ ] **Step 3: 删 validateExecDataAdjacency + 调用**

`validator.go` 删 `:170` 调用行 `errs = append(errs, validateExecDataAdjacency(c, sgs)...)`，并删整个 `validateExecDataAdjacency` 函数（`:375` 起整段，含 `check` 闭包到收尾）。

- [ ] **Step 4: 删 i18n key（zh + en 同删，保 parity）**

删 `frontend/src/i18n/zh.ts:1817` 的 `EXEC_DATA_NOT_ADJACENT: '...'` 整行；删 `frontend/src/i18n/en.ts:1797` 的 `EXEC_DATA_NOT_ADJACENT: '...'` 整行。

- [ ] **Step 5: 跑测试确认 PASS + 无残留引用**

Run: `go test ./internal/services/container/ -run TestAINode_ExecDataNonAdjacent -v`
Expected: PASS。

Run: `git grep -n EXEC_DATA_NOT_ADJACENT -- ':!flightdeck'`
Expected: 无输出（代码/前端无残留；flightdeck 里 spec/plan 提及不算）。

Run: `cd frontend && node src/i18n/check.cjs`
Expected: parity 通过（zh/en 对称删除）。

- [ ] **Step 6: commit**

```bash
git add internal/services/container/validator.go internal/services/container/ai_capture_repro_test.go frontend/src/i18n/zh.ts frontend/src/i18n/en.ts
git commit -m "refactor: remove EXEC_DATA_NOT_ADJACENT (cross-hop now legal via held output)"
```

---

### Task 5: held output 语义全覆盖（fan-out / 稀疏 / 未fire / loop / subgraph）

**Files:**
- Create: `internal/services/container/runtime/held_output_test.go`

**Interfaces:**
- Consumes: Task 1-3 的写/读路径。证明缓存语义在各拓扑下正确。

- [ ] **Step 1: 写 fan-out + 稀疏 + 未fire 测试**（直接 dispatch 模式，参照 `TestHeldOutput_CrossHopReadsCache`）

新建 `held_output_test.go`，三个用例：

- `TestHeldOutput_FanOut`：`n1(Failf)` 后两个并联 echo（`a`、`b`）的 `Value` 都数据线接 `n1.Code`；跑 `n1` 后分别跑 `a`、`b`，两者都读到 `capture_failed`（一次 fire、多消费读同一缓存值）。
- `TestHeldOutput_NotFired`：消费者 `Value` 接 `n1.Code` 但**不跑 `n1`** → 缓存无键 → echo 读到默认（空串 / nil），不 panic。
- `TestHeldOutput_Sparse`：先跑一次写 `n1.Code=v1`，再让同 `n1` 走一条不带 `Code` 的出口（或第二次 fire 不带该字段）→ 缓存留旧值 `v1`（验证稀疏「未带字段保留上次值」）。

> 三者全用 `tkFailf` / `tkEcho` / `resetTdEcho` / `getTdEchoLast` + `execNodeViaFramework` 直接派发，结构同 Task 2 的测试，不需要 Run()/window。`getTdEchoLast` 记录最近一次 echo，fan-out 取「分别跑 a/b 各自断言」。

- [ ] **Step 2: 写 loop + subgraph 端到端测试**（区域/子图 harness，参照 `dispatch_v5_test.go` 的 Loop 用例 + `subgraph_caller_test.go`）

- `TestHeldOutput_LoopLastIteration`：循环体内 exec 节点每轮产出递增字段，循环外消费者读最后一轮值（覆盖写语义）。构造参照 `dispatch_v5_test.go` 里现有 Loop body 测试的 `makeBodyForLoop`/`runRegionBody` 调用法。
- `TestHeldOutput_SubgraphScopedKeyUnique`：子图内 source 产出 exec 出口字段、子图内 consumer 跨跳读；用**不同 node ID**（production 必然不同）验证 flat 共享 map 键无碰撞、值正确。构造参照 `subgraph_caller_test.go` 的 `runSubgraphCall` 调用法。

> 这两个用例的 harness 样板（容器+子图装配、Loop seed）**照搬上面括号里点名的现有测试文件**的写法 —— 它们是本仓 region/subgraph 测试的既有范式来源，不是「类似 Task N」的占位，而是指向具体可抄的源。

- [ ] **Step 3: 跑全部新测试确认 PASS**

Run: `go test ./internal/services/container/runtime/ -run TestHeldOutput -v`
Expected: 全部 PASS。

- [ ] **Step 4: commit**

```bash
git add internal/services/container/runtime/held_output_test.go
git commit -m "test(runtime): held-output semantics across fan-out/sparse/not-fired/loop/subgraph"
```

---

### Task 6: 全量验证 + 收尾

**Files:**
- Modify: `flightdeck/specs/2026-06-23-held-exec-outputs.md`（标记落地，若有 plan-实测发现回填 §10）
- Modify: `flightdeck/cockpit.md`（经 stage/landing 流程，非手改 AUTO）

- [ ] **Step 1: Go 全量**

Run: `go build ./... && go test ./internal/services/container/...`
Expected: 仅预存基线红（`TestApplyDirection_*` / `TestWatchdog_*` / `TestScanSubgraphDependencies_*` / `TestFishingV2Main_StateCycleSmoke`），无新增回归。

- [ ] **Step 2: 前端类型 + i18n parity**

Run: `cd frontend && ./node_modules/.bin/vue-tsc --noEmit`
Expected: 无新增类型错（删 i18n key 不引入类型问题）。

Run: `cd frontend && node src/i18n/check.cjs`
Expected: parity 通过。

- [ ] **Step 3: 回填 spec 的 §10「plan 待验」实测结论**

把 §10 各风险点（单跳收编 / 子图键唯一 / per-tick / 缓存生命周期）改成「✅ 已验：<结论>」或在 spec 末加 `## Review notes`，记录 flat 共享 map 键唯一已证 + 子图测试结论。

- [ ] **Step 4: commit + 走 stage/landing**

```bash
git add internal/services/container/ flightdeck/specs/2026-06-23-held-exec-outputs.md
git commit -m "docs(spec): held-exec-outputs landed, §10 risks verified"
```

完成后 spec/plan 可标 `done`，走 `/flightdeck:landing` 归档 + graduate spec→docs。

---

## Self-Review

**Spec coverage**（逐节点对）：
- §3 存（routeResult 写）→ Task 1 ✓；读（pullDataPin）→ Task 2 ✓。
- §5.1 execOutputs 字段 + 重置 → Task 1（per-run = 实例生命周期）✓。
- §5.2 fire 钩子 applyCaptures 旁 → Task 1（两处）✓。
- §5.3 pullDataPin 读缓存 → Task 2 ✓。
- §5.4 删 applyExecDataEdges + 动态走缓存 → Task 3 ✓（含动态 coerce 缺口修补）。
- §5.5 删 validateExecDataAdjacency + i18n → Task 4 ✓。
- §5.6 复核 resolveDataPinV5 不误入 pure-data 闸 → Task 2 已确认（短路在 pullDataPin，resolveDataPinV5 对 exec 出口源走 pullDataPin 分支）✓。
- §6 语义（未fire/稀疏/loop/fan-out/顺序）→ Task 5 ✓。
- §9 测试策略（缓存读写/跨跳/fan-out/loop/回归/稀疏/未fire/校验）→ Task 1-5 ✓。
- §10 风险（单跳收编/子图/per-tick/生命周期/内存/写序）→ Task 3+5 实测 + Task 6 回填 ✓。

**额外修补（spec 未点透、源码核实得出）**：① buildDataWireFor 动态分支 coerce 缺口（Task 3 Step 3）；② 保留 `execData→RunNode` 原始 key-match 路径（Global Constraints，不误删）。

**Type 一致性**：`execOutputs map[string]any` / `captureExecOutputs(node, data)` 在 Task 1 定义、Task 2 消费，签名一致；键格式 `nodeID+"."+field` 写读两侧一致。
