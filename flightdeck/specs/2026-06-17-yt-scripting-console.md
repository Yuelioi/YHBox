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
- **Part 2 — yt 控制台 (建在 Part 1 之上)**: 下面所有 API/执行/UI。批量 set 的"一次性应用"落进 Part 1 加固后的撤销条目 → 主图+子图改动一步退。

> 两块各自一个 plan; Part 1 先落地验绿, 再做 Part 2。

## 架构 & 复用 (已核实的现有机制)

| 复用点 | 位置 | 用途 |
|---|---|---|
| `applyDraftMutation(mutator)` | `composables/containerEditor/useContainerDraft.ts` | 唯一改图入口; 自带 50 步撤销栈 + dirty + 渲染同步。批量改**一次**调它 = 一步撤销。 |
| `walkAllGraphs(container, subgraphs, visit)` | `composables/containerEditor/graphWalk.ts` | 遍历主图 + 所有子图节点 (含 `location` 上下文)。建 `yt.nodes` 用。 |
| `PIN_SPECS[kind]` / `KIND_DEFAULTS[kind]` | `components/containers/pinSpec.ts` | 查 kind 有哪些 input pin (`has`) + 默认值 (`get` 回退)。 |
| `getSelectedNodes` | `useVueFlow()` | 建 `yt.selected`。 |
| `CodeInput.vue` + editorTheme/CodeMirror | `components/expressions/CodeInput.vue` | 控制台代码编辑器 (JS 高亮)。`SubgraphScriptPreviewModal.vue` 是 CodeMirror 弹窗先例。 |
| `CommandPalette` + `useCommandPalette.ts` | `components/containers/` | 加一条「脚本控制台」入口命令。 |
| 节点对象形状 | `lib/backend.ts` `GraphNode` | `{ id, kind, label?, x, y, config: { literal: {pin: value} }, ... }` |

后端**无需改动**: 作用域是前端草稿, 落盘走现有 `containers.updateSilent` (保存即 `ValidateContainer` 校验)。goja/expr 是运行期引擎, 设计期不用 (Explore 已确认)。

## yt API 表面 (v1)

预注入全局 `yt` (也可 `const { nodes } = yt` 解构, 口味自选):

- **`yt.nodes`** — 当前容器全节点封装数组 (主图 + 所有子图)。每个元素 `NodeHandle`:
  - `n.id` `n.kind` `n.label` `n.location` — 只读 (`location` = "主图" / "子图: xxx")
  - `n.has(pin)` — 该 kind 的 spec 是否有此 input pin (查 `PIN_SPECS`, **不是**查有没有填值)
  - `n.get(pin)` — 当前有效值: literal 填了取填的, 没填取 spec 默认 (`KIND_DEFAULTS`); pin 不存在返 `undefined`
  - `n.set(pin, value)` — 改值。**spec 没此 pin → 拒绝并记入报告** (防 typo); 按 pin 类型轻度归一 (见 §归一)
- **`yt.selected`** — 仅当前画布选中的节点 (`yt.nodes` 的子集, 同 `NodeHandle`)
- **`yt.container`** — `{ id, name, vars }` (v1: name/vars 只读; 写变量留后)
- **`yt.log(...args)`** — 打到控制台输出区
- **`yt.ops`** — **预留命名空间** (v1 为空对象 / 不挂动作; 见 §以后)

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

1. 用 `new Function('yt', userCode)` 编译用户代码 (前端原生 JS, 全语法), 传入注入的 `yt`。
2. `n.set(...)` **不直接写草稿**, 先收集进一个 pending 变更表 `[{nodeId, sgID, pin, value}]`; `n.get()` 反映同一次运行内已 set 的值 (读自己刚改的)。
3. 脚本跑完**无异常** → 把 pending 变更**一次性**应用: 主图节点写 `draft`, 子图节点写 `editorStore.subgraphById(sgID).graph` + `touchSubgraph`; 整个应用包成**一条 Part-1 加固后的撤销条目** (主图+子图状态同进一个 snapshot) → 一步 Ctrl+Z 全退 + 标脏 + 重渲染。脚本**抛异常** → 丢弃全部 pending, **零变更** (原子)。
4. **不自动存盘**: 同现状 Ctrl+S 才落盘。安全网 = 跑完看画布/报告 → 不对 Ctrl+Z 一步退 → 对了 Ctrl+S。
5. 输出区报告: `改了 N 个节点的 M 个 pin` + `yt.log` 内容 + 被拒清单 (`节点 <id>(<kind>) 没有 pin <X>`) + 异常 (堆栈)。

## set 归一 & 拒绝规则

- `set(pin, v)`: 若 `!has(pin)` → 不写, 记入"被拒"报告。
- 类型归一 (按 `PIN_SPECS` 该 pin 的类型, 防 `set('TimeoutMs','3000')` 存成字符串):
  - Number / Integer / Duration → `Number(v)` (NaN 则拒并报告)
  - Bool → `Boolean(v)`
  - String → `String(v)`
  - 其它 (Geometry/JSON/List/...) → 原样写 (v1 不深校验, 落盘时 `ValidateContainer` 兜)

## 安全 & 错误

- `new Function` 跑在页面上下文, **非真沙箱**: 脚本理论上能碰全局、写死循环会卡 UI 线程 (同步 JS 无法硬超时)。**本地单人工具 + 用户跑自己的脚本** → 可接受, 换实现简单 + 全 JS 能力。真隔离需 Web Worker (API 要改消息传递), v1 不做。
- 语法/运行异常一律 catch → 打到输出区, **不崩编辑器、不改图**。

## UI / 入口

- 模态弹窗: 上半 `CodeInput`(CodeMirror, JS 模式) + 下半输出区 + 「运行」按钮 (Ctrl+Enter 跑)。
- 入口: ① Ctrl+K 命令面板加一条「脚本控制台」; ② 快捷键 (建议 Ctrl+Shift+J, 实施时查 `useEditorHotkeys.ts` 避免撞键)。
- 最近一次脚本可记进 localStorage 方便复用 (轻量, 可选)。

## 测试

把**执行器**抽成纯函数: `runConsoleScript(code, nodesModel) → { changes: [...], rejected: [...], logs: [...], error? }` (不碰 Vue/草稿)。单测:
- filter + set 命中节点 → changes 正确
- set spec 没有的 pin → 进 rejected, 不进 changes
- 脚本抛错 → changes 为空 (零变更), error 带信息
- 类型归一: Number pin set 字符串数字 → 存为 number; set 非数字 → rejected
- `get` 回退默认值 (literal 没填时取 KIND_DEFAULTS)
- `get` 读到同次运行内先前的 set

模态 UI + applyDraftMutation 接线轻量手验 (跑示例脚本 → 画布变 → Ctrl+Z 一步退 → Ctrl+S 落盘)。

## 以后 (非 v1, 预留不堵死)

- **`yt.ops.*`** — 把 `useCommandPalette` 的 `Command` 注册表暴露成可调函数 (autoLayout/fold/save/...), 即 `bpy.ops` 平行。注册表现成, 加进来便宜, 有需求再填。
- **跨容器批量** — 后端已有范式 `SyncLocalMouseCalibration` / `ClearAllHotkeys`; 真要"改所有容器"再做后端服务版。
- **dry-run 预演** / **Web Worker 沙箱** / **增删节点连边** / **写容器变量** — 均待需求。

## 验收

**Part 1 (子图撤销)**:
- 进子图手动改一个节点的值 → Ctrl+Z 能退回 (今天退不回)。
- undo/redo 跨主图↔子图改动都正确; 不误触发别的容器编辑器 dirty (守住复发#5)。

**Part 2 (yt 控制台)**:
- 控制台能开 (命令面板 + 快捷键), 写 JS 跑通示例 (抖动批量设值 / 按原值算)。
- 批量改**主图+子图**节点 → **一步 Ctrl+Z 全退**, 不存盘直到 Ctrl+S。
- set 不存在的 pin 被拒并在报告里; 脚本抛错零变更不崩。
- 执行器单测全绿; typecheck / i18n:check / `task build` 全绿。
