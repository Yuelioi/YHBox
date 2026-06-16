---
status: active
graduate: true
summary: 编辑器内 JS 脚本控制台 (命名空间根 yt, 对标 blender bpy): 对当前容器主图+所有子图全节点批量改 config。v1: yt.nodes/selected/container/log + 预留 yt.ops。new Function 执行, set 收集后一次性 applyDraftMutation(一步撤销)+标脏, 抛错零变更, Ctrl+S 落盘。复用 CodeInput(CodeMirror)/walkAllGraphs/PIN_SPECS。
last_updated: 2026-06-17
---

# yt 脚本控制台 (JS bulk-edit console)

## 背景 & 目标

手动逐个节点改值很烦。典型诉求: "把当前容器里所有节点的抖动值 (`JitterPct`) 设成 10" / "所有 Sleep 的抖动在原值上 +10%" 之类的**批量改**。

参考 UE/Unity/Blender: 给编辑器一个**脚本控制台**, 用户写一段 **JS** 对当前容器的全节点做批量操作。API 仿 Blender `bpy` —— 一个命名空间根 (`yt`, = yotta, 对标 `bpy` = blender python) 挂所有能力, 可发现 (CodeMirror 补全) + 不污染全局 + 可成长。

> 注: Blender 用 `import bpy` 是因为 Python 有模块系统; 本控制台是**注入式 JS**(无模块系统), 所以 `yt` 是**预注入全局**, 不写 import 行。

## 范围

**v1 (本 spec)**:
- 作用域 = **当前正在编辑的容器**, 含其**主图 + 所有子图** (经 `walkAllGraphs`)。不跨容器。
- 操作 = 读/改节点 **config.literal** 的 pin 值 (批量 set)。
- 前端实现; 改动走编辑器草稿 → **一步可撤销** (含子图), Ctrl+S 才落盘。

**非目标 (明确不做, 见 §以后)**: 跨容器批量; `yt.ops` 填充 (只预留命名空间); 强制 dry-run 预演; Web Worker 真沙箱; 增删节点/连边 (v1 只改 pin 值)。

## ⚠ 分解 (本 spec = 两块, 有先后)

读源码发现: 编辑器撤销栈 (`useContainerDraft.ts` 的 `history`/`undo`/`redo`) **只快照主图 `draft: Container`**; 子图存在独立的 `editorStore` 池里, **压根不在撤销栈内** —— 连平时手动改子图都不可撤销。要兑现"批量改 (含子图) 一步 Ctrl+Z 全退", 必须先补这块。故拆:

- **Part 1 — 子图纳入撤销 (核心编辑器, 先做)**: 扩展 `useContainerDraft` 的快照/undo/redo, 让历史条目同时携带**本容器的子图**状态, undo/redo 时一并写回 `editorStore`。**独立有价值** (顺带修好"子图编辑不可撤销"这一既有缺口), 可单独验收。
  - **快照规模**: 只快照**本容器** (= 本编辑器实例的 `containerID`) 关联的子图 (非全局池), 且优先只快照**动过的** (touched) → 控制 50 条历史 × 子图体积; 深拷贝 vs 结构共享策略 plan 定。**历史条目须连 touched 集一起快照/还原** —— 否则 undo 回旧版本后 dirty/touched 跟内容脱节。
  - **dirty 隔离 (守复发#5)**: 现有 `touchSubgraph(containerID,…)` / `touchedFor(containerID)` / `clearTouched(containerID)` **已按容器键隔离** (签名带 containerID, 已核实, 不用改它); Part 1 只需保证写回 + 快照都限本实例 `containerID`, watch 仍只盯本实例 activeGraph。
  - **对 Part 2 的契约**: Part 1 须提供"一次批量改主图+本容器子图、落成**单条**可撤销条目"的能力 (扩 `applyDraftMutation` 或新增 `applyBulkMutation`); 确切签名 = Part 1 plan 定 (动笔前精读 `useContainerDraft`/`editorStore` 内部)。
- **Part 2 — yt 控制台 (建在 Part 1 之上)**: 下面所有 API/执行/UI。批量 set 的"一次性应用"落进 Part 1 加固后的撤销条目 → 主图+子图改动一步退。

> 两块各自一个 plan; Part 1 先落地验绿, 再做 Part 2。

## 架构 & 复用 (已核实的现有机制)

| 复用点 | 位置 | 用途 |
|---|---|---|
| `applyDraftMutation(mutator)` | `composables/containerEditor/useContainerDraft.ts` | 唯一改图入口; 自带 50 步撤销栈 + dirty + 渲染同步。批量改**一次**调它 = 一步撤销。 |
| `walkAllGraphs(container, subgraphs, visit)` | `composables/containerEditor/graphWalk.ts` | 遍历主图 + 所有子图节点 (含 `location` 上下文)。建 `yt.nodes` 用。 |
| `PIN_SPECS[kind]` / `KIND_DEFAULTS[kind]` | `components/containers/pinSpec.ts` | 查 kind 有哪些 input pin (`has`) + 默认值 (`get` 回退)。 |
| `getSelectedNodes` | `useVueFlow()` | 建 `yt.selected` (开控制台那刻快照)。 |
| `editorStore.subgraphById` / `touchSubgraph` / `replaceSubgraph` | `stores/containerEditor.ts` | 子图读 / 标脏(按容器隔离) / 替换 —— Part 1 写回 + Part 2 改子图节点用 (已核实存在)。 |
| `CodeInput.vue` + editorTheme/CodeMirror | `components/expressions/CodeInput.vue` | 控制台代码编辑器 (JS 高亮)。`SubgraphScriptPreviewModal.vue` 是 CodeMirror 弹窗先例。 |
| `CommandPalette` + `useCommandPalette.ts` | `components/containers/` | 加一条「脚本控制台」入口命令。 |
| 节点对象形状 | `lib/backend.ts` `GraphNode` | `{ id, kind, label?, x, y, config: { literal: {pin: value} }, ... }` |

后端**无需改动**: 作用域是前端草稿, 落盘走现有 `containers.updateSilent` (保存即 `ValidateContainer` 校验)。goja/expr 是运行期引擎, 设计期不用 (Explore 已确认)。

## yt API 表面 (v1)

预注入全局 `yt` (也可 `const { nodes } = yt` 解构, 口味自选):

- **`yt.nodes`** — 当前容器全节点封装 (主图 + 所有子图), **点「运行」那刻基于草稿的只读副本构建** —— 数组 + 每个 handle 都 `Object.freeze`(用户 `yt.nodes.pop()` / 改字段都无效)。视图稳定**不靠"模态锁画布"假设**, 而靠**副本隔离**: 应用时按 `id` 回原图定位, 运行后被删的节点 → 跳过+报告。每个元素 `NodeHandle`:
  - `n.id` (UUID, 全局唯一) · `n.kind` · `n.label` · **`n.sgID`** (机器字段: `null`=主图 / 否则子图 id; filter 层级用 `n.sgID===null`(主图) / `!==null`(子图)) — **全部 getter-only 真只读** (`n.kind='x'` 改不动)。**不暴露 location 展示串** (易被误当 filter 判据 + 随改名变; 要显示用 `n.sgID??'主图'` 或交报告层算)
  - `n.has(pin)` — 该 kind 的 spec (`PIN_SPECS[kind].dataIn`) 有没有此 input pin (**查 spec, 不是查填没填值**)
  - `n.get(pin)` — 有效值: 本次运行已 set 过取 set 的, 否则 literal 填了取 literal, 都没有取该 kind spec 默认 (`KIND_DEFAULTS`, = 拖出来预填那个值); 无 literal 又无默认 (如 Required-无默认 pin) → `undefined`
  - `n.set(pin, value)` — 改值, 写进 pending overlay (见 §执行模型)。**spec 没此 pin / 值归一失败 → 拒绝并记报告** (不写 overlay, 后续 get 读不到这次); 按 pin 类型归一 (见 §归一)
- **`yt.selected`** — 开控制台那刻 **当前 active graph (主图或所在子图) 的选中**节点快照, **不跨图收集** (按 id 取 `yt.nodes` 子集)
- **`yt.container`** — `{ id, name }`, **冻结只读** (写它在 strict 下抛错; 写容器变量留后, 见 §以后; v1 不暴露 `vars` 半成品)
- **`yt.log(...args)`** — 打到控制台输出区
- **`yt.ops` v1 不注入** —— 只在设计层预留这个名给"脚本调 UI 动作"; **不挂空对象**(免得用户看见就调、撞 undefined)。见 §以后

数组方法 (filter/map/forEach/...) 是原生 JS Array, 不另造。

示例 (均可跑):
```js
// 所有抖动设 10
yt.nodes.filter(n => n.has('JitterPct')).forEach(n => n.set('JitterPct', 10))

// 所有 Sleep 的抖动在原值上 +10%
yt.nodes.filter(n => n.kind === 'Sleep')
        .forEach(n => n.set('JitterPct', n.get('JitterPct') * 1.1))

// 选中节点的超时统一 3s
yt.selected.filter(n => n.has('TimeoutMs')).forEach(n => n.set('TimeoutMs', 3000))
```

## 执行模型

1. 用 `new Function('yt', '"use strict";\n' + userCode)` 编译运行 (前端原生 JS 全语法, **strict mode** —— 少踩 JS 老坑、让对冻结对象的写直接抛错), 传入注入的 `yt`。执行器契约 (纯函数签名 + `NodeModel` 入参) 见 §测试。
2. `n.set(...)` **不直接写草稿**, 写进 **pending overlay** —— 一张 `(sgID, nodeId, pin) → value` 的 Map (**同 pin 多次 set = 后写覆盖, 只留最后值**)。`n.get()` 先读 overlay (读到自己刚 set 的), 再 literal, 再默认。被拒的 set (无 pin / 归一失败) **不进 overlay** → 后续 get 读到的是改之前的值。
3. 脚本跑完**无异常** → 把 overlay **一次性**应用 (全在内存): 先按 `id` 回原图定位每个目标 (运行后被删的 → 跳过+报告), 主图节点写 `draft`、子图节点写 `editorStore.subgraphById(sgID).graph` 对应节点 + `editorStore.touchSubgraph(containerID, sgID)`; 算完整批新状态后包成**一条 Part-1 加固后的撤销条目** (主图 + 本容器子图状态同进一个 snapshot) 提交 → 一步 Ctrl+Z 全退 + 标脏 + 重渲染。脚本**抛异常** → 丢弃整个 overlay, **零变更**。**应用阶段自身抛错** (replaceSubgraph / store 写入异常等) → 视为执行失败, **不提交历史条目**、报错 (不留半应用)。
   > **原子的精确含义**: 只对"脚本抛异常"原子 (抛了 = 一个都不改)。**被拒的 set 不是异常** —— 是有意的"跳过 + 报告"。所以正常结局是"**尽力应用 + 列出被拒**"(可部分应用), 不是"全有/全无"。
4. **不自动存盘**: 同现状 Ctrl+S 才落盘 (现有 `useEditorSave` 两段式保存**已**会把 touched 子图一并写盘 → 第 3 步 `touchSubgraph` 后保存链路**无需改**)。安全网 = 跑完看画布/报告 → 不对 Ctrl+Z 一步退 → 对了 Ctrl+S。
5. 输出区报告分三块: **已应用** `改了 N 个节点的 M 个 pin` (N=不同节点数, M=overlay 里不同 `(节点,pin)` 对数, 后写覆盖只算一次; **0 改动也明说「0 个节点 / 0 个 pin」**) · **被拒** 清单 (`节点 <id>(<kind>): 无 pin X / 值非法`) · `yt.log` 内容; 抛错另附**异常堆栈**。

## set 归一 & 拒绝规则

- `set(pin, v)`: 若 `!has(pin)` → 拒绝并记报告 (不写 overlay)。
- 类型归一 (按 `PIN_SPECS[kind].dataIn[pin]` 的 `PinType`, 防 `set('TimeoutMs','3000')` 存成字符串):
  - `number` (含后端 Number/Integer/**Duration** —— 本项目 **Duration 字面值就是毫秒数** (`inputs.go` `in.Duration`: json.Number→ms, **无 "100ms" 单位串**, 故无单位丢失问题)) → `Number(v)`; **非有限值 (NaN / Infinity) → 拒绝并报告**; **Integer 遇非整数 → 拒绝并报告** (不静默取整 —— 免得跟编辑器别处取整约定不一致; 真要取整 plan 里查清约定再说)
  - `bool` → 真布尔 / 数字 `0|1` / 字符串 `"true"|"false"` 才映射, 其它 (含数字 2、其它串) → 拒绝 (**不用裸 `Boolean(v)`** —— `Boolean("false")` 会错成 `true`)
  - `string` → `String(v)`
  - `point` / `list` / `any` (Geometry/JSON/List 等) → 原样写, 但**拒绝非 JSON-可序列化值** (函数/DOM 等, 否则要到落盘才炸); 深校验仍交 `ValidateContainer`

## 安全 & 错误

- **定位 = 本地单人自动化工具** (整个产品就是给用户在自己机器跑自己写的自动化), 非多租户/不可信用户场景 → 安全等级低。
- `new Function` 跑在页面上下文, **非真沙箱**: 脚本能碰 window/fetch/localStorage、甚至绕过 `yt` 直接改 `editorStore` (绕过撤销/校验)。鉴于上面的定位 → **接受**, 换实现简单 + 全 JS 能力; v1 不做沙箱/网络限制 (真要隔离上 Web Worker、API 改消息传递, 留后)。
- **无硬超时**: 同步 JS 死循环会卡 UI 线程 (没法对同步代码硬超时)。v1 **约定**用户自保脚本会终止, 并在控制台 UI **标一行提示**: "脚本同步执行, 死循环会卡界面, 请确保能结束"。
- 语法/运行异常一律 catch → 打到输出区, **不崩编辑器、不改图**。

## UI / 入口

- 模态弹窗: 上半 `CodeInput`(CodeMirror, JS 高亮) + 下半输出区 + 「运行」按钮 (Ctrl+Enter 跑) + 一行死循环提示 (见 §安全)。
- 入口: **主入口 = Ctrl+K 命令面板加一条「脚本控制台」** (命令入口 i18n `editor.palette.cmd.jsConsole`; 模态内文案统一前缀 `editor.jsConsole.*`)。专用快捷键可选, 须选**不撞 WebView/浏览器 DevTools** 的组合 (**避开 `Ctrl+Shift+J`/`I`/`C`** —— Chromium 拿它们开 DevTools), 具体 plan 定。
- **`yt.*` 自动补全**: "可发现"得靠 CodeMirror 补全认识 `yt.*`。v1 挂一份补全表, 范围 = `yt.{nodes,selected,container,log}` + `NodeHandle.{id,kind,label,sgID,has,get,set}` (kind 值补全如 'Sleep' = nice-to-have)。**补全表与 yt API 定义同一份常量** (别维护两份真相 → 加 `yt.x` 不漏同步)。现有 `scriptCompletions.ts` 可挂, Part 2 plan 落实。
- (记 localStorage 复用最近脚本 = nice-to-have, **v1 不做**, 留后。)

## 测试

**执行器抽成纯函数** (不碰 Vue/草稿) —— 签名:
`runConsoleScript(code: string, model: NodeModel[]) → { applied: Change[], rejected: Reject[], logs: string[], error?: string }`
`NodeModel = { id, kind, sgID: string|null, label, literal: Record<string,unknown>, specPins: Record<string,PinType> }` (从 `GraphNode` + `PIN_SPECS`/`KIND_DEFAULTS` 组装)。执行器据此造 `NodeHandle` → 跑代码 → 产出 overlay → `applied`/`rejected`。Vue 层只: 造 model → 调执行器 → `applied` 经 Part-1 应用 + 渲染报告。

单测 (Part 2 执行器):
- filter + set 命中 → applied 正确
- set spec 没有的 pin → rejected, 不进 applied
- **同 pin 多次 set → 只留最后值, 算 1 个改动**
- **set 被拒后 get → 读到改前的值 (不是被拒值)**
- 脚本抛错 → applied 为空 (零变更), error 带信息
- 归一: number pin set 字符串数字 → number; set NaN/Infinity → rejected; **bool set `"false"` → `false` (不是 true)**
- get 回退: literal 没填取 KIND_DEFAULTS; 无默认 → undefined
- get 读到同次运行内先前的 set

单测 (Part 1 子图撤销):
- 改子图节点 → 历史 snapshot 带上子图状态; undo/redo 正确还原子图
- 主图改 + 子图改交错 → undo 多步顺序正确
- undo 回初始 → dirty 标记正确; 不污染别的容器实例 (复发#5)

模态 UI + 应用接线轻量手验 (项目惯例 = 单测 + 真机 smoke; 不上 e2e)。

## 以后 (非 v1, 预留不堵死)

- **`yt.ops.*`** — 把 `useCommandPalette` 的 `Command` 注册表暴露成可调函数 (autoLayout/fold/save/...), 即 `bpy.ops` 平行。注册表现成, 加进来便宜, 有需求再填。
- **跨容器批量** — 后端已有范式 `SyncLocalMouseCalibration` / `ClearAllHotkeys`; 真要"改所有容器"再做后端服务版。
- **dry-run 预演** / **Web Worker 沙箱** / **增删节点连边** / **写容器变量** — 均待需求。

## 验收

**Part 1 (子图撤销)**:
- 进子图手动改一个节点的值 → Ctrl+Z 能退回改之前的状态 (当前实现退不回)。
- undo/redo 跨主图↔子图改动都正确; 不误触发别的容器编辑器 dirty (守住复发#5)。

**Part 2 (yt 控制台)**:
- 控制台能开 (命令面板 + 快捷键), 写 JS 跑通示例 (抖动批量设值 / 按原值算)。
- 批量改**主图+子图**节点 → **一步 Ctrl+Z 全退**, 不存盘直到 Ctrl+S。
- **联动核心路径** (Part1+Part2): 一次批量改**主图 + 多个子图**节点 → **一次 Ctrl+Z 全退、一次 Ctrl+Shift+Z 全恢复** (单列, 两块联动最关键路径)。
- set 不存在的 pin 被拒并在报告里; 脚本抛错零变更不崩。
- 执行器单测全绿; typecheck / i18n:check / `task build` 全绿。

## 评审纪要 (三方 AI 审核, 2026-06-17)

原文 `tmp/{ds,gpt,claude}.txt`。AI 不了解项目全貌, 逐条对源码评估后:

**采纳, 已改进本 spec**:
- 原子性精确化: "原子"只对异常; 被拒 set = 跳过+报告 (尽力应用, 可部分) → §执行模型 加注。
- pending overlay 数据结构 `(sgID,nodeId,pin)→value` 后写覆盖; get 读序 overlay→literal→默认; 被拒不进 overlay。
- `n.sgID` 机器字段 (判层级用), `n.location` 降为纯展示 (别 filter)。
- 归一收紧: number 拒非有限值 (NaN/Infinity)、Integer 取整; bool 不用裸 `Boolean(v)` (`Boolean("false")`=true 的坑)。
- get 默认值语义 = literal ?? KIND_DEFAULTS, 无默认→undefined, 说清。
- `yt.selected` / `yt.nodes` = 运行/开窗那刻**快照** (模态期画布不可动, 无并发漂移)。
- `yt.container` 裁到 `{id,name}` (砍半成品 vars); `yt.ops` v1 **不注入空对象** (免误导)。
- 快捷键避开 `Ctrl+Shift+J/I/C` (撞 DevTools); 主入口走命令面板 + i18n key `editor.palette.cmd.jsConsole`。
- 死循环: UI 标提示 + 文档化"无超时, 用户自保终止"。
- 自动补全: 明确 v1 要挂 `yt.*` 静态补全表 (否则"可发现"落空)。
- 执行器签名 + `NodeModel` 入参形状定死; 测试补 (同 pin 多 set / 被拒后 get / bool 特例 / Part 1 子图撤销单测)。
- 报告计数规则 (不同节点 / 不同 (节点,pin) 对); 保存链路确认**无需改** (现 `useEditorSave` 已存 touched 子图); Part 1 快照规模 + dirty 隔离 + 对 Part 2 的契约写进 §分解。
- 文案: "今天"→"当前实现"。

**驳回 (AI 不懂项目 / 已决策)**:
- "`last_updated: 2026-06-17` 日期有误" → **就是今天**, 无误。
- "Duration `Number(v)` 丢单位 (如 100ms)" → 本项目 Duration 字面值就是**毫秒数** (`in.Duration`), 无单位串, 无此问题。
- "必须加沙箱 / 限制 fetch/window / 必须 e2e" → 本地单人工具, 非沙箱已两次拍板接受 (YAGNI); 项目测试惯例 = 单测 + 真机 smoke, 不引 e2e。
- "`const {nodes}=yt` 被重新赋值会坏" → 用户自己的 JS, 非框架职责。

**推迟 (YAGNI / 留后)**: localStorage 记最近脚本; `yt.ops` 填充; 容器变量读写; dry-run; Web Worker 沙箱; 按具体子图 id 取节点的 API (主图/子图判别用 `sgID` null-check 已够)。

**第二轮 (gpt 判"接近可实施", ds 复审)采纳**: 快照改**只读冻结副本** + 按 id 回查 (不再依赖"模态锁画布"前提); NodeHandle/`yt.nodes`/`yt.container` 真冻结 (getter-only); `new Function` 加 `"use strict"`; **应用阶段抛错 = 不提交历史条目、报错不留半应用**; `yt.selected` = 当前 active graph 选择不跨图; **去掉 `n.location`** (改名陷阱) 只留 `n.sgID`; Integer 非整数**拒绝**(不静默取整, 免撞未知约定); point/list/any 拒非 JSON-可序列化值; 报告含 0 改动场景; 补全表与 API **同一份常量**; i18n 前缀 `editor.jsConsole.*`; touchSubgraph **已按容器隔离**(核实, 不改它) + 历史须连 touched 集快照; 加 Part1+Part2 **联动验收** (主图+多子图 → 1×Ctrl+Z 全退 / 1×Ctrl+Shift+Z 全恢复)。
驳回/推迟: "文档太长" (是 spec, 细节喂 plan; gpt 已认可接近可实施); 运行前预测死循环 (检测不了, 静态提示足够); 具体子图 id 列表 API (YAGNI)。
