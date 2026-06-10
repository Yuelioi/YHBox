---
status: active
when_to_read: 写/改 Script 节点或绑定层前; 想知道脚本里能调什么函数、怎么传参取值接错误; 加新节点想确认脚本侧是否自动可见; 改 Expr/Script 动态输入机制前; 撞 SCRIPT_* 校验码或脚本取消/行号问题
applies_to: [script, script-node, goja, node-as-function, ScriptBindable, binding, sugar, vars, sleep, dynamic-inputs, DynamicInputs, code-widget, CodeInput, internal/services/script, internal/nodes/script, internal/services/container/validator_script.go, frontend/src/components/expressions, frontend/src/lib/scriptCompletions.ts, frontend/src/components/containers/inspector/DynamicInputsEditor.vue]
when_to_update: 改绑定调用/返回/错误约定、糖函数集合、ScriptBindable 排除规则、IIFE 包裹/行号修正、code widget 组件链路或 DynamicInputs 机制时
last_updated: 2026-06-11
---

# Script 系统 — 内嵌 JS 脚本节点与节点即函数绑定

一句话: **Script 节点让用户用 JavaScript (goja 引擎) 写一段逻辑, 每个已注册的 Runnable/Evaluator 节点自动是脚本里的同名全局函数** — 加新节点零绑定维护, 脚本侧自动多一个函数 + 补全自动多一项。

## 节点形态 (`internal/nodes/script/script.go`)

- Kind `Script`, Category `Control`, **Runnable**。
- 输入: `In` (Exec) / `Code` (String, widget `code`) / `CaptureResult` (捕获框, 脚本返回值→变量)。
- 输出: `Done` (Data 字段 `Result` `*` = 脚本 `return` 的值) / `Fail` (error 语义, Error+Code)。
- `NeedsWindow: true` — 含 Script 的容器必须配 WindowTarget (保守设计: 脚本可能调输入/视觉节点, 宁可编辑期报也不跑一半炸)。
- `DynamicInputs: true` — 动态输入 pin, 见下。
- 返回值消费: 同 exec 节点惯例, `CaptureResult` 填变量名 + GetVar 读; Result 也作为 Done 出口 exec-data 下发。

## 脚本怎么写 (用户面约定)

- **顶层 `return` 返回结果** (实现: 源码包 IIFE `(function(){…})()`, 行号偏移已在报错侧减回, `internal/services/script/compile.go`)。
- **同步模型**: 无 async/await; 调阻塞节点 (WaitTemplate 等) 就是阻塞等, 和图里跑一样。
- ES2017 级语法 (let/const/箭头/解构/模板字符串)。

## 节点即函数 (绑定层 `internal/services/script/binding.go`)

- **哪些节点可调**: `node.ScriptBindable(rn)` (`internal/node/registry.go`) 是唯一判定源 — Runnable 或 Evaluator, 排除 RegionRunner (Loop/Subgraph/CollapsedNode — 脚本有原生循环, 调子图未实现)、IsGraphMarker、IsVisualOnly、Script 自身 (防递归)。前端补全 RPC `GetScriptBindableKinds` 走同一函数。
- **调用约定**: 单对象参数, 键 = pin 名 (PascalCase); 省略键用 Spec 默认值; 缺必填抛异常。`ClickAt({XRatio: 0.5, YRatio: 0.5})`。
- **返回约定**: exec 节点 → `{exit: "出口名", ...出口Data字段}` (如 `r.exit === "Found"`, `r.Point`); PureFunc (Evaluator) → 直接返回求值结果。
- **错误约定**: 节点失败 = JS 异常, 异常对象带 `code` / `message` / `kind` 字段, `try/catch` 可接 (超时重试等); 未接住 → Script 走 `Fail` 出口, **透传原节点错误码**; 脚本自己 `throw` → 码 `thrown`。
- 实现走框架公开入口 `node.RunNode` / `node.EvaluatePureData` (自带 Required/Validate 门 + 正确 Spec 的子 ctx + recover); RunResult.Panic 原样 re-panic, 不许被脚本 catch。
- 数字归一: goja Export 整数是 int64, 绑定层 `NormalizeJS` 递归转 float64 (Inputs coercion 只认 float64/json.Number/string)。

## 糖函数 (仅四组, 高频到值得短名字)

`vars.get/set/inc(name, [scope])` (scope 缺省 "auto") · `params.get(name)` · `sleep(ms)` (可取消) · `log.debug/info/warn(...)`。变量读的是 **live 值** (Snapshot 不包, 脚本是 exec 语境)。

## 取消

watchdog goroutine 监听 `ctx.Context().Done()` → `vm.Interrupt()`: 停容器时哪怕脚本在纯 JS 死循环也立即打断, Run 返 `ctx.Err()` (graceful halt, 同 Sleep/KeyPress)。被调的阻塞节点用同一 ctx, 原生取消。

## 编辑期校验 (`internal/services/container/validator_script.go`)

- `SCRIPT_PARSE_ERROR` — goja 编译查语法, 报错带行号 (已修正 IIFE 偏移)。
- `SCRIPT_DUPLICATE_INPUT` — 动态输入重名。
- 运行时语法错 = 裸 error 冒泡中断 (配置错, 同 Expr parse 模式), 不走 Fail。

## 动态输入机制 (Expr/Script 共用)

- `Spec.DynamicInputs: true` 节点的额外 data-in pin 由 `config.Inputs[]` 声明 (PascalCase Name/Type), 后端统一走 `container.ParseDynamicInputDecls`; dispatch (`buildDataWireFor`) / validator (`dataInPinTypeForNode` / unknown-literal 跳过) / FE (`adapter.ts parseDynamicInputsCfg` / `pinLiterals.ts`) 全部标志驱动, **无 kind 字符串特判**。
- 声明编辑 UI: Inspector「输入口」区 (`DynamicInputsEditor.vue`, 按 `spec.dynamicInputs` 显示) — 加/删行 + 名字合法性/重名标红; 此前 Expr 没有任何添加 UI (只能 fusion/手改 JSON), 这是 2026-06-11 一并补上的。
- 连进来的值在脚本里是**同名只读全局变量** (Expr 里是表达式标识符)。

## 前端编辑器链路

- widget kind `code` → `PinInput.vue` 分支 → `CodeInput.vue` (CodeMirror 6 + `@codemirror/lang-javascript`, 骨架同 ExprInput) + 右上「放大编辑」按钮 → `CodeEditorModal.vue` (BaseModal 大编辑器, 确认才回写)。
- 补全 = 节点函数 (registry store 的 Spec 推导签名, `frontend/src/lib/scriptCompletions.ts`) + 四组糖 + 本节点动态输入名; **无前端 lint** (语法错归后端 validator)。
- 画布内联框排除 code widget (8 行代码只在 Inspector/modal 编辑)。

## 加新节点时脚本侧要做什么

**什么都不用做。** 注册即绑定 (ScriptBindable), 补全经 RPC 自动出现; 只要按 add-node checklist 把 i18n label 写了, 补全 detail 自动带中文名。

## 设计取舍记录 (简)

- 语言选 JS/goja 而非 Lua/starlark: 纯 Go + Interrupt 可打断 + 语法与 expr 同体系 + CodeMirror 官方包 + 默认无 IO (沙箱)。
- 绑定走「节点即函数自动绑定」而非手写精选 API (用户 2026-06-10 拍板): 零维护、覆盖面随注册表增长; 高频操作用糖弥补人体工学。
- 编辑器自动建 GetVar / 恢复 $vars 语法等方案的淘汰理由见变量系统议题 (cockpit 在册)。
