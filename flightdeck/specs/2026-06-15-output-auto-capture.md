---
status: active
summary: "输出自动捕获 + Inspector 输出组统一绑定 (Spec C)。取消'逐节点手声明 Semantic:capture 输入框 + Run 里手调 node.Capture()'这套, 改成框架在 dispatch routeResult 把 fire 出口的 OutputData[字段] 自动写进用户绑定的变量。前端 Inspector「输出」组合并掉 Part 2 的'速览'+'捕获'两套, 每个可绑产出一行: 翻译名 + 类型 + 「+绑变量」按钮 (绑后显 → \$var ✕ chip), 写 config.capture{字段:变量名}。所有执行节点的数据产出自动可绑 (含现在漏声明捕获框的 PlayClip.Error/Code)。核心不变量: 被捕获值必须是出口 Data 字段 (从 OutputData 读) —— 模板三件套的 Found 布尔补成显式 Data 字段。删 13 文件 27 个 capture 输入 + node.Capture 助手。迁移条件化 (旧 config.literal[Capture<X>] → config.capture[<X>], 没有就跳过)。边界: 不碰 vue-flow 画布/节点/连线/pin, 绑定全在 Inspector。"
last_updated: 2026-06-15
---

# 输出自动捕获 + Inspector 输出组统一绑定 (Spec C)

> 承接 Spec B (容器编辑器 restyle, 已 done 归档)。本 spec 由 2026-06-15 brainstorm 拍定 (用户问"为什么不直接把所有出口都允许设置变量, 统一又好管理", + 嫌现在输出捕获交互不优雅 + 输出速览没翻译)。

## 背景 / 目标

现在"把节点产出存进变量"叫**捕获框**: 节点 Spec 里逐个手声明 `Semantic:"capture"` 的 String 输入(`CaptureCount`/`CaptureCenter`…), `Run()` 里手调 `node.Capture(ctx, in, 框名, 值)` 写进变量(`internal/node/capture.go`, scope 固定 auto)。三个毛病(用户提的):

1. **不统一 / 漏声明**: 捕获框是逐节点手工活 → 有的全(DetectColor 有 CaptureCount/CaptureCenter), 有的漏(PlayClip 的 `Fail` 出口带 `Error`/`Code` Data 字段, 但**没声明捕获框** → 用户根本绑不了)。这违反项目自己的硬约束([node-data-flow](../checklists/2026-06-05-node-data-flow.md): 产出型节点必须给每个产出加捕获框)。
2. **没翻译 / 费解**: Spec B Part 2 我加的「输出速览」取 `pinsFor().dataOut` 的**原始 pin 名**(Count/Center/Error/Code); 捕获框走 `NODE_FIELD_SCHEMAS` 的**翻译 label**(命中像素数/命中中心点)→ 同一输出两种叫法。
3. **交互不优雅**: 捕获框 = 每个"标题 + 文本框 + 一行 hint", 占地大。

**目标**: ① **框架层自动捕获** —— 节点作者不再写捕获框 / `node.Capture()`, 框架自动把出口产出写进用户绑定的变量; ② 前端「输出」组**统一**成一个翻译一致、紧凑的列表, 每个可绑产出一个「+绑变量」按钮(绑后显 chip)。结果: **所有执行节点的数据产出自动可绑**, 无节点作者样板代码。

## 核心机制 (读源码确认, 别脑补)

- 捕获助手: `node.Capture(ctx, in, captureField, value)`(`internal/node/capture.go`)= 读捕获框里的变量名, trim 非空则 `ctx.Vars().SetScoped(name, "auto", value)`。
- 出口 fire 携带数据: `ctx.Out("exit").Set("字段", 值).Fire()` → `Outputs{exitName, data}`(`internal/node/outputs.go`)。
- **dispatch 已有产出数据**: `dispatch_v5.go` 的 `routeResult`(L182) 里, `result.ExitName` + `result.OutputData`(= fire 出口的 data map)现成, 现在用于 `r.edges.nextWithData(node.ID+"."+result.ExitName, ..., result.OutputData)`(L262)喂 exec-data 连线。**自动捕获就钉在这里**: 写连线前/后, 对 `config.capture` 里绑了变量的字段, `varStore.SetScoped(varName, "auto", result.OutputData[字段])`。
- 命名约定(DetectColor 实证): 捕获框 `Capture<Field>` ↔ 出口 Data 字段 `<Field>`(`CaptureCount`↔`Count`, `CaptureCenter`↔`Center`); 被捕获值同时 `node.Capture(...)` 和 `.Set(<Field>, 值)`。
- 前端已能列出可绑产出: `adapter.ts splitOutputs`(L133)把"exec 出口携带的 Data 字段"flatten 进 `dataOut` → `getSpec(kind).dataOut` 含全部 Data 字段(PlayClip.Error/Code 也在内)。

## ⛔ 核心不变量 (本设计成立的前提 — 实现第一步逐节点核对)

自动捕获从 `result.OutputData` 读, 所以**每个"被捕获的值"必须作为出口 Data 字段存在并被 `Set()`**。逐节点核对 `grep -rn 'Semantic: "capture"' internal/nodes`(13 文件 27 个), 对每个捕获点确认"被捕获值已 `.Set(<Field>, 值)`"; 缺的补 Data 字段。**已知要补**:

- **模板三件套 `Found` 布尔**(`wait_template.go` / `check_template.go` / `click_template.go`): `Found` 捕获的是布尔, 但它不是 Data 字段 —— "命中与否"靠**走哪个出口**(命中 vs Timeout/NotFound)编码。**补法**(brainstorm 拍定): 给每个加 `Found`(Type `Bool`)Data 字段, 命中出口 `Set(Found, true)`、未命中/超时出口 `Set(Found, false)`, 两出口都带 → 自动捕获可读。
- **待核**: `control/loop.go`(CaptureIndex)、`control/foreach.go`(2 个, Item/Index?)—— 确认循环序号/元素是否已作为出口 Data 字段 `Set`; 不是就补(同 Found 处理)。其余检测/截图/脚本/秒表节点的捕获多是真数据值(Count/Center/Point/Conf/Ratio/Clusters/Path/Elapsed/Result/Inner*X/Outer*X/Px…), 大概率已 `Set`, 仍须逐个核。

## 设计决策

### 1. 后端 · 自动捕获

- **dispatch hook**: `routeResult` 里, 拿到 `result.ExitName` + `result.OutputData` 后, 读本节点 `config.capture`(见下), 对每个 `字段→变量名` 且 `字段 ∈ result.OutputData`(本次 fire 出口实际带该字段)→ `varStore.SetScoped(变量名, "auto", result.OutputData[字段])`。语义跟现在一致: 只写该出口实际带的字段(未命中出口不带 Center → 不写 Center, 变量留旧值); 节点返 error → 不 route、不写(routeResult 不处理 error 分支的捕获)。
- **存储**: 节点 `config.capture: map[string]string`(`字段名 → 变量名`)。顶层 config 键, 跟 `config.literal` 平级。空 map / 缺省 = 无绑定。
- **删干净**(二号铁律): 移除 13 文件全部 `Semantic:"capture"` 输入 pin 声明 + 全部 `node.Capture(...)` 调用 + `internal/node/capture.go`(助手)+ `capture_test.go` + `spec_capture_test.go`(CaptureType 校验, 框没了就没意义)。`InputSpec.CaptureType` 字段: 若再无消费者一并删(核对 `grep CaptureType`)。
- **校验**: `config.capture` 的变量名走变量引用校验(新校验或并入 `validator_var_refs.go`, 码 `INVALID_VAR_REF`): 变量名非空且不在 `Container.Vars[].Name` / 子图 `RequiredGlobals` → 红错(scope 固定 auto, 同捕获旧语义)。字段名必须是该 kind 的合法 Data 字段(否则 `INVALID_PIN` 类)。

### 2. 前端 · Inspector「输出」组统一 (方案 A: 按钮绑 + chip)

合并掉 Part 2 的「输出速览(只读)」+「输出捕获(VarNameInput)」两套, 成一个列表(`NodeInspector.vue`「输出」组):

- **exec 出口**(完成/失败/命中…): 顶部一行只读参考(`↗ 完成  ↗ 失败`), 不可绑。
- **可绑产出**(`非 IsPureData` 节点的 dataOut 字段 = 出口 Data 字段): 每个一行 `翻译名  类型  [+ 绑变量]`; 绑了显 `→ $变量名 ✕`(✕ 解绑)。点「+绑变量」走现有 `VarNameInput`(选现有变量 / 现场新建; 新建类型取该 Data 字段的类型)。写回 `config.capture[字段]`。
- **纯数据节点**(`IsPureData`: GetVar/Now/VarLastChange/Expr/PureFunc): 输出是连线源, 显示为只读参考行, **不给绑定**(绑了没意义)。
- 删 `NodeInspector` 现有 `captureLiterals` / `captureOpen` / 折叠头那套(9d66558 移进输出组的那段)→ 换成上述统一列表; `outPins` 速览并入。

### 3. 翻译

- 给 Data 字段加 i18n label: `node.<kind>.output.<字段>`(如 `node.DetectColor.output.Count` = "命中像素数")。旧捕获框的中文 label(`node.<kind>.input.Capture<X>.*` / `NODE_FIELD_SCHEMAS`)直接搬过来复用, 然后删旧捕获框 i18n 键。前端输出行 label 走这套, 不再显原始英文 pin 名。

### 4. 迁移 (条件化)

- 实现时 `grep` 现有容器(`internal/.../*.json` 测试 fixture + 用户 cook TOML 不涉及节点 config; 主要是 dev 机上的容器存档)有没有 `config.literal["Capture<X>"]` 非空绑定。
- **有** → 写一次性迁移(`config.literal["Capture<X>"]` 的变量名 → `config.capture["<X>"]`, 键名去 `Capture` 前缀; 跑完删旧键)。**没有**(用户印象里没绑过)→ 跳过, 不写迁移。
- 这不是兼容 shim, 是一次性切干净(二号铁律允许; 服务用户存档)。

## 边界与风险

- **边界**: 不碰 vue-flow 画布 / `ContainerFlowNode` / 连线 / pin / 右键 / inline 输入件(沿用 Spec B 边界)。绑定入口全在 Inspector「输出」组。Data 字段在画布上仍是 exec 出口携带(渲染不变, 仍是 [node-data-flow](../checklists/2026-06-05-node-data-flow.md) 说的"画得出连不上"footgun, 本 spec 不解决画布侧, 只给 Inspector 绑定路径)。
- **风险 1 — dispatch 是状态机核心**: 改 `routeResult` 要确保 ① 只在 fire 出口后写 ② 跟 exec-data 连线注入**并存**(同一 OutputData 既喂连线又喂捕获)③ error 分支不误写。配 dispatch 单测(`dispatch_v5_test.go`)。
- **风险 2 — 不变量漏核**: 见上 §核心不变量。漏一个 = 该产出绑了变量但运行时不写。**实现第一步必须逐节点核对**, 不是写完再补。
- **风险 3 — capture 节点测试**: 13 文件改 Spec(删捕获输入)→ 各自 `*_test.go` 里断言捕获写入的用例要改成"绑 config.capture + 验 dispatch 后变量被写"或移到 dispatch 层统一测。
- **风险 4 — 变量类型提示**: 旧捕获框带 `CaptureType`(新建变量时定类型)。新方案从 Data 字段 `Type`(Number/Point/Bool/String…)派生 → `VarNameInput` 新建变量的类型用它。确认 Data 字段类型→VarType 映射齐(point/bool/number/string/any; list 不在捕获白名单, 保持)。

## 验证

- **后端**: `go test ./internal/...` —— 新增 auto-capture dispatch 单测(绑定字段 fire 后变量被写 / 未带字段不写 / error 不写 / 多字段); 13 capture 节点测试改写; 删 capture 助手测试。
- **前端**: `pnpm typecheck` / `pnpm test`(NodeInspector 输出组若抽纯逻辑配 .helpers 单测, 如"哪些 dataOut 可绑")/ `pnpm build:dev`; 禁用色扫描零行。
- **真机 smoke**: 进编辑器, 给 ① 检测节点(DetectColor 命中像素数/中心点)② 回放录像(Error/Code, 旧本绑不了的)各绑一个变量 → 跑 → 用 GetVar / 日志确认变量被写入正确值; 未命中出口不覆盖旧值; 翻译名一致。

## 实现分期 (plan 切分建议, 写 plan 时定)

- **Part 1 · 后端自动捕获**: dispatch hook + `config.capture` + 不变量核对/补 Data 字段(Found bool 等)+ 删 13 文件捕获框/助手 + 校验 + 后端测试。(结构大头、风险集中, 先做先验。)
- **Part 2 · 前端输出组统一**: 方案 A UI + 翻译 i18n + 删旧 captureLiterals 那套 + 前端测试。
- **Part 3 · 迁移**(条件): 查存档 → 有则脚本、无则跳过 + 收尾真机 smoke。

(顺序可调; Part 1 必须先于 Part 2 真机验。)
