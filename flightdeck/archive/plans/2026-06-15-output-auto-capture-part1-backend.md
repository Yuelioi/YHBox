---
status: done
summary: "Spec C 实现计划 Part 1 (后端自动捕获)。① 模板三件套 Found 补成 Bool Data 字段(两出口都 Set); ② config.capture{字段:变量名} 存储 + 路径① fire-time 自动捕获(ContainerRunner.applyCaptures 钩在 routeResult 成功路径 result.OutputData + 失败路径 {Error,Code}, 用 r.bundle.Vars.SetScoped auto); ③ 路径② ctx.CaptureOutput(field,value)(Ctx 接口 + ctxImpl.captureBindings, newCtx 从 config[capture] 解析)+ Loop/ForEach 改用它 + Index/Item 声明为 Body 出口 Data 字段; ④ 删 13 文件 27 个 Semantic:capture 输入 + 11 fire-time 节点 node.Capture 调用 + node.Capture 助手 + capture_test/spec_capture_test, grep+vet 零残留; ⑤ validator 校验 config.capture var-ref + BindableFields 后端单一来源。"
last_updated: 2026-06-15
implements: specs/2026-06-15-output-auto-capture.md
---

# 输出自动捕获 · 后端 Implementation Plan (Spec C · Part 1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把"节点产出存进变量"从"逐节点手声明 `Semantic:"capture"` 输入框 + `Run()` 手调 `node.Capture()`"改成 **config.capture 存储 + 框架两条写路径自动捕获**(① fire-time 出口 Data 字段在 dispatch `routeResult` 自动写; ② region 循环 Index/Item 由 `ctx.CaptureOutput` 每轮写),覆盖所有执行节点产出(含现在漏声明的 PlayClip.Error/Code)。

**Architecture:** 纯 Go 后端。捕获绑定存节点 `config.capture map[string]string`(字段名→变量名)。**路径①**(检测/模板/截图/脚本/秒表等 fire-time 节点):框架 `ContainerRunner.applyCaptures(node, dataMap)` 钩在 `dispatch_v5.go routeResult` —— 成功路径用 `result.OutputData`、失败路径用 `{Error,Code}` map,对 `config.capture` 里绑了且本次 fire 出口实际带的字段 `r.bundle.Vars.SetScoped(var,"auto",值)`(跟旧 `node.Capture` 同写目标同语义)。**路径②**(Loop/ForEach 循环迭代值,不经 routeResult):新增 `ctx.CaptureOutput(field, value)` 方法(`ctxImpl` 持 `captureBindings`,`newCtx` 从 `config["capture"]` 解析),节点 `RunRegion` 每轮调。两路都删旧 capture 输入 + `node.Capture`。validator 把 config.capture 纳入 var-ref 校验。**边界**:不碰前端(Part 2)、不碰画布。

**Tech Stack:** Go 1.x · `internal/node`(framework: spec/ctx/engine/capture/outputs)· `internal/nodes/{detect,control,io,stopwatch,script}`(节点)· `internal/services/container/runtime`(dispatch_v5 / ContainerRunner / varStore)· `internal/services/container`(validator / model)。测试 `go test ./internal/...`。

---

## 验证基线 (本计划通用)

- 后端每 Task 收尾:`go test ./internal/...` 绿(改的包测试 + 新增测试)。`go vet ./...` 干净。
- 全计划收尾:`grep -rn 'node\.Capture(' internal/` = **零**(旧助手彻底删、无残留无死代码,落地精度#12);`grep -rn 'Semantic: "capture"' internal/nodes` = **零**。
- 关键不变量(spec §核心不变量):**被 fire-time 捕获的值必须是出口 Data 字段**;实现按 Task 顺序逐节点核对,缺的补。
- 真机 smoke 在 Part 3 收尾(本 Part 只后端单测 + dispatch 集成测试)。

## 受影响文件总览

**框架 (`internal/node/`)**:
- `spec.go` — `OutputSpec.Data []DataField` **已支持**(L63),无 struct 改动;Task 4 末视情况删 `InputSpec.CaptureType`。
- `interfaces.go` — `Ctx` 接口加 `CaptureOutput(field string, value any)`。
- `ctx.go` — `ctxImpl` 加 `captureBindings map[string]string`;`newCtx` 改签名收 config;实现 `CaptureOutput` + `parseCaptureBindings`。
- `engine.go` — `prepareExec` 调 `newCtx` 传 config(签名联动)。
- `capture.go` — **删**(Task 4,`node.Capture` 助手);`capture_test.go` **删**。
- `spec_capture_test.go` — **删**(Task 4,capture 输入没了)。

**dispatch (`internal/services/container/runtime/`)**:
- `dispatch_v5.go` — 加 `applyCaptures` 方法 + `routeResult` 成功/失败路径各钩一次。
- `dispatch_v5_test.go` — 加路径① dispatch 测试 + 集成测试。

**节点 (`internal/nodes/`)**:
- `detect/wait_template.go` · `check_template.go` · `click_template.go` — Task 1 加 `Found` Bool Data 字段(两出口 Set);Task 4 删 capture 输入 + node.Capture。
- `control/loop.go` · `foreach.go` — Task 3 改 `ctx.CaptureOutput` + Body Data 字段声明 + 删 capture 输入。
- `detect/{detect_color,detect_color_hsv,detect_color_blobs,roi_color_scan,dual_color_bar_track,screenshot}.go` · `stopwatch/read.go` · `script/script.go` — Task 4 删 capture 输入 + node.Capture(.Set Data 字段保留)。
- 各 `*_test.go` — Task 4 改写捕获断言。

**校验 (`internal/services/container/`)**:
- `validator_var_refs.go`(或新文件)— Task 5 config.capture var-ref 校验 + `BindableFields` 后端单一来源。

---

## Progress

current: **done**。后端自动捕获全落地, go build/test/vet 绿(仅余 10 个预存「runtime 缺 fish fixture」失败 = apply_direction/watchdog json 缺文件, 实测与 Spec C 前同款、子集、零新增回归)。**无真机 smoke**(Part 1 无前端入口; 留 Part 2)。

- T1 done — 816d4cb (模板三件套 Matched bool Data 字段, 两出口 Set; 命名 Matched 避与 Found 出口冲突)
- T2 done — 5107d7f (路径① applyCaptures 钩 routeResult 成功+失败路径; 4 dispatch 测试)
- T3 done — cf7e2d3 (路径② ctx.CaptureOutput + ctxImpl.captureBindings; Loop/ForEach 改用 + Body Data 字段)
- T4 done — 33fa43f (删 13 文件 27 capture 输入 + 11 node.Capture + 助手 + CaptureType 字段; 测试改断 OutputData; grep 零)
- T5 done — (commit 后) (BindableFields 单一来源 + validateCaptureRefs var-ref/字段校验)
- fix — 89e16c1 (state_FISHING fixture 迁 config.literal[Capture*]→config.capture; 修 T4 引入的 fixture 回归)

**遗留(Part 2 处理)**: ① `internal/catalog/node-i18n.json` 仍有旧 capture 字段 label (stale, 无消费者不破坏测试; Part 2 改前端 i18n 时一并重生成); ② 编辑器输出捕获 UI 暂不可用(旧捕获框随后端 capture 输入删而消失, 新「输出」组绑定 UI 是 Part 2)—— **落地精度#7: P1+P2 是一个发布单元, 别只发 P1**。

---

## Task 1: 模板三件套 Found 补成 Bool Data 字段

spec §核心不变量: `Found`(命中与否)捕获的是布尔, 但它不是 Data 字段(靠走哪个出口编码)→ 路径① 读不到。补成两出口都带的 `Found` Bool Data 字段。**additive**: 此 Task 不删旧 capture(老 path 仍工作), 只加 Data 字段 + Set, 让 Found 将来可被路径① 捕获。

**Files:** `internal/nodes/detect/wait_template.go` · `check_template.go` · `click_template.go`(+ 各 `*_test.go`)

- [ ] **Step 1: wait_template.go — 加 Found const + 两出口 Data 字段 + Set**

`const` 块加 `wtDataFound = "Found"`(跟 `wtDataPoint`/`wtDataConf` 同处)。Spec Outputs 改:

```go
Outputs: []node.OutputSpec{
    {Name: wtOutFound, Type: "Exec",
        Data: []node.DataField{
            {Name: wtDataPoint, Type: "Point"},
            {Name: wtDataConf, Type: "Number"},
            {Name: wtDataFound, Type: "Bool"},
        }},
    {Name: wtOutTimeout, Type: "Exec",
        Data: []node.DataField{
            {Name: wtDataConf, Type: "Number", Optional: true},
            {Name: wtDataFound, Type: "Bool"},
        }},
},
```

Run 里两出口都 Set Found(命中 true / 超时 false),保留现有 `node.Capture(wtCapFound,...)`(Task 4 删):

```go
// 命中分支 (现 L90 附近):
return ctx.Out(wtOutFound).Set(wtDataPoint, *pt).Set(wtDataConf, conf).Set(wtDataFound, true).Fire(), nil
// 超时分支 (现 L93 附近):
return ctx.Out(wtOutTimeout).Set(wtDataConf, conf).Set(wtDataFound, false).Fire(), nil
```

- [ ] **Step 2: check_template.go / click_template.go — 同样补 Found**

二者 const 已有 `ctCapFound`/`clkCapFound` + 对应 Data 字段常量(读文件确认 `*DataFound` 是否已存在; 没有则加 `ctDataFound="Found"` / `clkDataFound="Found"`)。在各自命中出口 `.Set(*DataFound, true)`、未命中出口 `.Set(*DataFound, false)`, 两出口 Data 列表都加 `{Name:*DataFound, Type:"Bool"}`。(check/click 的 Found 现状是 `node.Capture(*CapFound, true/false)`, 保留。)

- [ ] **Step 3: 测试 — 断言 OutputData 带 Found**

各节点 `*_test.go` 加(或扩现有命中/未命中用例): 命中 → `result.OutputData["Found"] == true`; 未命中/超时 → `result.OutputData["Found"] == false`。命中用例同时断言 Point/Conf 仍在(没回归)。

- [ ] **Step 4: 验证** `go test ./internal/nodes/detect/...` 绿; `go vet ./internal/nodes/detect/...` 干净。

- [ ] **Step 5: commit**

```bash
git add internal/nodes/detect/wait_template.go internal/nodes/detect/check_template.go internal/nodes/detect/click_template.go internal/nodes/detect/*_test.go
git commit -m "feat(nodes): template trio Found -> explicit Bool Data field on both exits (Spec C prep)"
```

---

## Task 2: config.capture + 路径① fire-time 自动捕获 (routeResult hook)

新增框架在 dispatch 成功/失败路由时, 把出口 Data 字段按 `config.capture` 绑定写进变量。**additive**: 不动任何节点; config.capture 此时为空(无绑定)→ 实测无副作用, 旧 `node.Capture` 仍各管各。

**Files:** `internal/services/container/runtime/dispatch_v5.go`(+ `dispatch_v5_test.go`)

- [ ] **Step 1: 写 `applyCaptures` 方法**

`dispatch_v5.go` 加(import 已有 `strings`? 没有则加;`container` 已 import):

```go
// applyCaptures 路径① fire-time 自动捕获 (Spec C)。对 node.config.capture 里绑了变量、
// 且本次 fire 出口实际带该字段 (field ∈ data, 稀疏: 只含节点 .Set() 过的字段) 的字段,
// 写进变量 (scope auto, 跟旧 node.Capture 同写目标同语义)。未带字段不写 → 变量留旧值。
func (r *ContainerRunner) applyCaptures(node *container.GraphNode, data map[string]any) {
    if len(data) == 0 || r.bundle.Vars == nil {
        return
    }
    capRaw, ok := node.Config["capture"].(map[string]any)
    if !ok || len(capRaw) == 0 {
        return
    }
    for field, varNameRaw := range capRaw {
        v, present := data[field]
        if !present {
            continue // 本次 fire 出口没带该字段 → 不写, 留旧值
        }
        varName, _ := varNameRaw.(string)
        varName = strings.TrimSpace(varName)
        if varName == "" {
            continue
        }
        r.bundle.Vars.SetScoped(varName, "auto", v)
    }
}
```

- [ ] **Step 2: routeResult — 成功路径钩 applyCaptures**

`routeResult` 成功路径(现 L259-263)改:

```go
    if result.ExitName == "" {
        return nil, nil
    }
    r.applyCaptures(node, result.OutputData) // 路径①: 出口 Data 字段 → 绑定变量
    tokens := r.edges.nextWithData(node.ID+"."+result.ExitName, tok.LoopStack, result.OutputData)
    return tokens, nil
```

- [ ] **Step 3: routeResult — 失败路径钩 applyCaptures**

失败路由(现 L247-252 `if handled` 块)改 —— 失败出口的 Data = `{Error, Code}`, 提取成 map 复用给 capture + nextWithData:

```go
    if handled {
        failData := map[string]any{
            "Error": result.Error.Error(),
            "Code":  string(coded.ErrCode()),
        }
        r.applyCaptures(node, failData) // 路径①: Fail 出口 Error/Code → 绑定变量 (PlayClip 等)
        return r.edges.nextWithData(node.ID+".Fail", tok.LoopStack, failData), nil
    }
```

- [ ] **Step 4: dispatch 测试 — 路径① 写入 / 不写 / error 不写 / 留旧值**

`dispatch_v5_test.go` 加测试(仿现有 dispatch 测试搭一个带 Data 字段出口的测试节点 + config.capture 绑定, 跑 dispatch, 断言 VarStore):
- 绑定字段 fire 后变量被写成出口值。
- 出口未带某字段(稀疏)→ 该绑定变量**保留旧值**(先 set 旧值, 跑未带该字段的出口, 断言没变)—— spec/reviewer 关注的负面用例。
- 节点返 error 且无 Fail 接线(冒泡)→ 不写。
- Fail 出口接线 + 绑 Error/Code → 写入 error message / code。
- config.capture 为空 / 变量名空白 → 不写(no-op)。

> 测试节点可用 `dispatch_v5_test.go` 现有 helper 风格(参 `newRegionTestLoop` 等); 出口 Data 字段在测试节点 Spec 里声明 + Run 里 `.Set()`。

- [ ] **Step 5: 验证** `go test ./internal/services/container/runtime/...` 绿; `go vet` 干净。

- [ ] **Step 6: commit**

```bash
git add internal/services/container/runtime/dispatch_v5.go internal/services/container/runtime/dispatch_v5_test.go
git commit -m "feat(runtime): path-1 fire-time auto-capture (config.capture -> var at routeResult)"
```

---

## Task 3: 路径② `ctx.CaptureOutput` + Loop/ForEach 改用 + Body Data 字段

Loop/ForEach 的 Index/Item 在 `RunRegion` 每轮捕获、不经 routeResult → 走新 ctx 方法。

**Files:** `internal/node/interfaces.go` · `ctx.go` · `engine.go` · `internal/nodes/control/loop.go` · `foreach.go`(+ 测试)

- [ ] **Step 1: Ctx 接口加 CaptureOutput**

`interfaces.go` `Ctx interface`(现 L88)加(放 `Out(...)` 附近):

```go
    // CaptureOutput 路径② (Spec C): region 节点每轮把迭代产出写进用户绑定变量。
    // field = 该节点 Body 出口声明的 Data 字段名; 框架查 config.capture[field], 非空则 SetScoped auto。
    CaptureOutput(field string, value any)
```

- [ ] **Step 2: ctxImpl 加 captureBindings + 实现 CaptureOutput + parse helper**

`ctx.go`: `ctxImpl` struct 加字段 `captureBindings map[string]string`。改 `newCtx` 签名收 config + 解析:

```go
func newCtx(ctx context.Context, services ServiceBundle, spec *Spec, config map[string]any) *ctxImpl {
    return &ctxImpl{ctx: ctx, services: services, spec: spec, captureBindings: parseCaptureBindings(config)}
}

// parseCaptureBindings 从 node config 取 capture 绑定 (字段名→变量名)。
func parseCaptureBindings(config map[string]any) map[string]string {
    raw, ok := config["capture"].(map[string]any)
    if !ok {
        return nil
    }
    m := make(map[string]string, len(raw))
    for k, v := range raw {
        if s, ok := v.(string); ok {
            m[k] = s
        }
    }
    return m
}

func (c *ctxImpl) CaptureOutput(field string, value any) {
    name := strings.TrimSpace(c.captureBindings[field])
    if name == "" {
        return
    }
    if c.services.Vars == nil {
        return
    }
    c.services.Vars.SetScoped(name, "auto", value)
}
```

(`ctx.go` 加 import `"strings"`。)

- [ ] **Step 3: 更新 newCtx 全部调用点**

`grep -rn 'newCtx(' internal/node` → 改全部为 4 参签名。已知:`engine.go prepareExec`(L43)`c := newCtx(ctx, services, &rn.Spec, config)`。其余命中(测试 helper 等)逐个补 `config` 实参(无 config 的测试传 `nil`)。

- [ ] **Step 4: loop.go — 改 CaptureOutput + Body Data 字段 + 删 capture 输入**

`loop.go`: 删 `loopCapIndex` const + 删 Inputs 里的 `{Name: loopCapIndex, ...Semantic:"capture"...}`(现 L48-49)。`loopOutBody` 出口加 Data 声明(供前端列举 Index 为可绑):

```go
{Name: loopOutBody, Type: "Exec", Data: []node.DataField{{Name: "Index", Type: "Number"}}},
```

RunRegion 两处 `node.Capture(ctx, in, loopCapIndex, i)`(现 L71/L85)→ `ctx.CaptureOutput("Index", i)`。

- [ ] **Step 5: foreach.go — 同样**

删 `feCapItem`/`feCapIndex` const + Inputs 里两个 capture 声明。`feOutBody` 加 Data:

```go
{Name: feOutBody, Type: "Exec", Data: []node.DataField{{Name: "Item", Type: "Any"}, {Name: "Index", Type: "Number"}}},
```

RunRegion(现 L57-58)`node.Capture(ctx,in,feCapItem,el)` / `node.Capture(ctx,in,feCapIndex,i)` → `ctx.CaptureOutput("Item", el)` / `ctx.CaptureOutput("Index", i)`。

> `"Any"` 是 DataField.Type 的 any 类型 tag;若该 tag 名在类型系统里不同(`grep '"Any"' internal/node` / 看 types 注册),用注册的 any tag。前端 `backendTypeToPinType` 对未知 tag 有 fallback,不阻断。

- [ ] **Step 6: 区域测试 — 功能即一致性断言 (落地精度#3)**

`loop` / `foreach` 测试(`control` 包,或 dispatch 层 region 测试)绑 `config.capture`:
- Loop: config.capture `{"Index":"i"}`, mode count N → 跑完变量 `i` = N-1(末轮); 或逐轮断言(若测试能 hook body)。**字段名写错(如 Run 里 `CaptureOutput("Idx",...)`)→ 绑定 `Index` 永不被写 → 此测试失败** = 自然捕获 Spec↔代码名不一致。
- ForEach: list `[a,b,c]`, config.capture `{"Item":"x","Index":"j"}` → 跑完 `x`=c、`j`=2。
> region 写经 `ctx.services.Vars`(`SetScoped` auto),测试用真 VarStore(参现有 region 测试的 ctx 搭建)。

- [ ] **Step 7: 验证** `go test ./internal/node/... ./internal/nodes/control/... ./internal/services/container/runtime/...` 绿;`go vet` 干净。

- [ ] **Step 8: commit**

```bash
git add internal/node/interfaces.go internal/node/ctx.go internal/node/engine.go internal/nodes/control/loop.go internal/nodes/control/foreach.go internal/nodes/control/*_test.go internal/node/*_test.go
git commit -m "feat(node): path-2 ctx.CaptureOutput + Loop/ForEach iteration capture via config.capture"
```

---

## Task 4: 删旧 capture (11 fire-time 节点 + node.Capture 助手)

路径① 已接管 fire-time 捕获 → 删 11 节点的 capture 输入 + node.Capture 调用 + 助手。**这 Task 后旧 capture 机制彻底没了**。

**Files:**(节点 `*.go` + `*_test.go`)`detect/detect_color.go` · `detect_color_hsv.go` · `detect_color_blobs.go` · `roi_color_scan.go` · `dual_color_bar_track.go` · `screenshot.go` · `wait_template.go` · `check_template.go` · `click_template.go` · `stopwatch/read.go` · `script/script.go`;删 `internal/node/capture.go` · `capture_test.go` · `spec_capture_test.go`。

- [ ] **Step 1: 逐节点删 capture 输入 + node.Capture 调用**

每个文件统一两动作(**保留 `.Set(<Data字段>,值)` —— Data 字段是路径① 的来源,不动**):
1. Inputs 里删 `{Name: <xxCap*>, ...Semantic:"capture"...}` 行 + 删对应 `const` 名。
2. Run 里删 `node.Capture(ctx, in, <xxCap*>, 值)` 行。

逐文件 capture const(grep 实证,删这些 + 其 node.Capture 调用):

| 文件 | capture 输入 const → (对应 Data 字段) |
|---|---|
| `detect/detect_color.go` | `dcCapCount`→Count · `dcCapCenter`→Center |
| `detect/detect_color_hsv.go` | `dchCapCount`→Count · `dchCapRatio`→Ratio |
| `detect/detect_color_blobs.go` | `dcbCapPrimaryCenter` · `dcbCapPrimaryArea` · `dcbCapBlobCount` |
| `detect/roi_color_scan.go` | `rcsCapCount` · `rcsCapClusters` |
| `detect/dual_color_bar_track.go` | `dcbtCapInnerX·OuterX·OuterW·Conf·InnerPx·OuterPx`(6) |
| `detect/screenshot.go` | `ssCapPath`→Path |
| `detect/wait_template.go` | `wtCapFound`(Found 已 Task1 补 Data)· `wtCapPoint`→Point |
| `detect/check_template.go` | `ctCapFound` · `ctCapPoint` |
| `detect/click_template.go` | `clkCapFound` · `clkCapPoint` |
| `stopwatch/read.go` | `swReadCapElapsed`→Elapsed |
| `script/script.go` | `inCaptureResult`→(脚本输出 Data 字段) |

> **逐节点核对不变量(spec §核心不变量)**: 删 node.Capture 前确认被捕获值已 `.Set(<Data字段>,值)` 在 fire 出口。若发现某捕获值没对应 Data 字段(如 screenshot Path / script Result 没 Set),**先补 Data 字段 + Set**(同 Task1 Found 做法)再删 capture。这是本 Task 的硬前置,逐文件确认。

- [ ] **Step 2: 删 node.Capture 助手 + 其测试**

```bash
git rm internal/node/capture.go internal/node/capture_test.go internal/node/spec_capture_test.go
```

- [ ] **Step 3: 改写节点测试**

各节点 `*_test.go` 里"断言捕获写入变量"的用例 → 改成断言 **fire 出口 OutputData 带该字段 + 值正确**(节点的责任 = Set 对;框架写变量的责任已由 Task2 dispatch 测试覆盖)。例:detect_color_test 原断言 `vars["c"]==count` → 改 `result.OutputData["Count"]==count` + `result.OutputData["Center"]==Point{...}`。

- [ ] **Step 4: CaptureType 字段清理(条件)**

`grep -rn 'CaptureType' internal/` —— 若 Go 侧只剩 `spec.go` 的 `InputSpec.CaptureType` 声明(无赋值/读取消费者,capture 输入已删)→ 删该字段(二号铁律)。前端的 captureType 消费 Part 2 处理,跟本 Part 无关。

- [ ] **Step 5: 全后端零残留验证**

```bash
grep -rn 'node\.Capture(' internal/        # 期望: 零
grep -rn 'Semantic: "capture"' internal/nodes   # 期望: 零
go vet ./...                                # 干净 (无未用 import/函数)
go test ./internal/...                      # 绿
```

- [ ] **Step 6: commit**

```bash
git add internal/nodes/ && git rm internal/node/capture.go internal/node/capture_test.go internal/node/spec_capture_test.go
git add -A internal/node/spec.go
git commit -m "refactor(nodes): remove all Semantic:capture inputs + node.Capture (framework auto-capture takes over)"
```

---

## Task 5: validator 把 config.capture 纳入 var-ref 校验 + BindableFields 单一来源

config.capture 引用变量名 → 编辑期校验(漏绑变量名 / 字段名非法报错)。后端 `BindableFields` 当"哪些字段可绑"的单一来源(落地精度#1,Part 2 前端另有对应实现,二者派生规则一致)。

**Files:** `internal/services/container/validator_var_refs.go`(扩,或新 `validator_capture.go`)·(BindableFields 放 `internal/node/` 或 validator 处)+ 测试。

- [ ] **Step 1: BindableFields 后端单一来源**

加(`internal/node/spec.go` 或同包 helper)—— 某 kind 可绑字段 = 非 IsPureData 节点所有 exec 出口的 Data 字段名(含 Task3 给 Loop/ForEach Body 声明的 Index/Item):

```go
// BindableFields 返回某节点 spec 的可捕获字段名 (exec 出口的 Data 字段)。
// 纯数据节点 (IsPureData) 无可绑字段 (其输出是连线源, 非捕获)。单一来源, validator/前端派生一致。
func BindableFields(spec *Spec) []string {
    if spec.IsPureData {
        return nil
    }
    seen := map[string]bool{}
    var out []string
    for _, o := range spec.Outputs {
        if o.Type != TypeExec {
            continue
        }
        for _, f := range o.Data {
            if !seen[f.Name] {
                seen[f.Name] = true
                out = append(out, f.Name)
            }
        }
    }
    return out
}
```

- [ ] **Step 2: validator — config.capture var-ref + 字段合法性校验**

仿 `validator_var_refs.go` 的 GetVar/SetVar VarName 校验,对每个节点的 `config.capture`:
- 变量名非空且不在 `Container.Vars[].Name` / 子图 `RequiredGlobals[].Name` → `INVALID_VAR_REF`(scope auto,同捕获语义)。
- 字段名 ∉ `BindableFields(spec)` → `INVALID_PIN`(或 `INVALID_CAPTURE_FIELD`)。
读 config.capture: `node.Config["capture"].(map[string]any)`,nil-safe。

- [ ] **Step 3: 测试**

validator 测试:① 绑了不存在的变量 → INVALID_VAR_REF;② 绑了合法变量 → 无错;③ 绑了该 kind 不存在的字段 → 报错;④ `BindableFields` 单测(DetectColor → [Count,Center];Loop → [Index,Error,Code];GetVar(IsPureData)→ nil)。

- [ ] **Step 4: 验证** `go test ./internal/services/container/...` 绿;`go vet` 干净。

- [ ] **Step 5: commit**

```bash
git add internal/services/container/ internal/node/spec.go
git commit -m "feat(validator): config.capture var-ref validation + BindableFields single source"
```

---

## 收尾

- [ ] 全后端 `go test ./internal/...` 绿 + `go vet ./...` 干净 + 两条 grep 零残留(见验证基线)。
- [ ] **不真机 smoke**(本 Part 无前端入口;真机验在 Part 2 出 UI 后)。后端能力就绪 = dispatch 测试(路径①)+ region 测试(路径②)+ validator 测试全绿。
- [ ] 更新 plan `status`、cockpit;Part 1 done 后接 Part 2(前端输出组 + 消费者审计)。**注意 spec 落地精度#7**:Part 1(config.capture 后端)与 Part 2(前端消费者审计)是一个发布单元,别在 P1↔P2 之间留过夜半成品(中间态删变量/重命名漏改捕获绑定)。

## Self-Review (spec 覆盖核对)

- spec §两条写路径 → T2(路径①)+ T3(路径②)。✅
- spec §核心不变量(被捕获值须是 Data 字段;Found 补 Data)→ T1(Found)+ T4 Step1 逐节点核对(screenshot/script 等缺 Set 则补)。✅
- spec §1 存储 config.capture + routeResult hook + 写/注入独立 channel → T2(成功+失败两钩点)。✅
- spec §1 删干净(13 文件 capture 输入 + node.Capture + 助手)→ T3(region 2)+ T4(fire-time 11 + 助手 + 测试)。✅
- spec §2 消费者审计 → **Part 2**(前端 useVarMutations)+ 本 Part T5(后端 validator/referrers 的 config.capture var-ref)。本 Part 覆盖后端侧。✅
- spec 落地精度#1 BindableFields 单一来源 → T5 Step1(后端);前端 Part 2 对应。✅
- 落地精度#8 OutputSpec.Data 复用零 struct 改 → T3 Body Data 字段直接用现有结构。✅
- 落地精度#12 grep+vet 零残留 → T4 Step5 + 收尾。✅
- 落地精度#13 spec_capture_test 删 + region 一致性测试独立 → T4 Step2 删 / T3 Step6 功能测试即一致性。✅
- **占位扫描**: 各 Step 给确切文件/const/行号区域/代码/命令;T4 表格逐文件列 const,非"similar to";唯 screenshot/script 的 Data 字段补全在 T4 Step1 标为"核对后补"(确定性核对步,非 TBD)。✅
- **类型一致**: `applyCaptures(node, data)` / `ctx.CaptureOutput(field, value)` / `parseCaptureBindings` / `BindableFields(spec)` / `captureBindings map[string]string` 前后一致;`config.capture` 形态 `map[string]any`(值 string)贯穿 T2/T3/T5。✅
