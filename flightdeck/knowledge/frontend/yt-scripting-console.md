---
kind: note
summary: "历史 3.0 yt/Container JS 批量编辑控制台；3.1 明确不恢复，仅供旧行为取证。"
activation: action
read_when: "仅在审查 3.0 yt 控制台行为、旧快捷键或解释为何 3.1 不恢复 ambient JS 编辑权限时"
recheck_when: "改 yt.* API 形态 / 执行(overlay·归一)或撤销机制 / 补全分流规则 / 控制台入口或 UI 时"
---
# yt 脚本控制台 (JS bulk-edit console)

> 历史知识：3.1 已删除该入口和 ambient `yt.*` API。批量编辑必须使用 `internal/workflow/authoring` typed command/patch；不得从本文恢复 JS/Wails/DOM 旁路。
编辑器内的 **JS 控制台**, 对**当前容器**(主图 + 所有子图)的节点**批量改 config**。命名空间根 `yt` (对标 blender `bpy`)。入口: 工具栏 **⋯ 更多 →「打开 JS 脚本控制台」** 或 **Ctrl+K** 命令面板搜"脚本"。

## yt.* API (注入给脚本的全局)

- **`yt.nodes`** — 当前容器全节点封装数组 (主图 + 所有子图), **运行那刻的冻结只读副本** (`Object.freeze`)。原生 `.filter/.map/.forEach` 等。
- **`yt.selected`** — 开控制台那刻**当前 active graph** 选中节点的快照 (不跨图)。
- **`yt.container`** — `{ id, name }` 冻结只读。
- **`yt.log(...args)`** — 打到输出区。
- 每个节点是 `NodeHandle` (getter-only 真只读):
  - `n.id`(UUID) · `n.kind` · `n.label` · `n.sgID`(`null`=主图 / 否则子图 id)
  - `n.has(pin)` — 该 kind 的 spec (`PIN_SPECS[kind].dataIn`) 有无此输入 pin (查 spec, 非查填没填值)
  - `n.get(pin)` — 有效值: 本次运行已 set 过取 set 的, 否则 literal, 否则 spec 默认 (`KIND_DEFAULTS`), 都没→`undefined`
  - `n.set(pin, value)` — 改值, 写进 pending overlay。**spec 无此 pin / 值归一失败 → 拒绝并记报告**

示例: `yt.nodes.filter(n => n.has('JitterPct')).forEach(n => n.set('JitterPct', 10))`

## 执行模型 (`executor.ts` — 纯函数 `runConsoleScript`)

1. `new Function('yt', '"use strict";\n'+code)` 跑用户 JS。
2. `n.set` 写进 **pending overlay** (`(sgID,nodeId,pin)→value`, 后写覆盖); `n.get` 读序 overlay→literal→默认; 被拒的 set 不进 overlay。
3. **归一** (按 `PIN_SPECS[kind].dataIn[pin]` 的 PinType): `number`(含后端 Number/Integer/**Duration**=毫秒数) 拒非有限值; `bool` 只认真布尔/0|1/"true"|"false" (不裸 `Boolean`); `string`→`String`; `point/list/any` 拒非 JSON-可序列化。
4. **原子**: 只对脚本抛异常原子 (抛了零变更); 被拒 set 不是异常, 是"跳过+报告" → 正常结局是"尽力应用 + 列被拒"。
5. 产出 `{ applied, rejected, logs, error?, nodeCount, pinCount }`; 0 改动如实报告。
- glue (`useYtConsole.ts`): 组装 `NodeModel[]`(draft 主图 + 子图 + `PIN_SPECS`/`KIND_DEFAULTS`) → `runConsoleScript` → 按 sgID 分组 `applied` → 经 `applyBulkMutation` 落地 (主图写 `draft` / 子图写 `editorStore`)。

## 撤销 (`historyEngine.ts` + `useContainerDraft.applyBulkMutation`)

撤销引擎 `historyEngine` 是纯函数 (init/push/replaceTop coalesce/augmentTop/undo/redo/restore); 历史条目可带 `sgState` (被触及子图快照, 加法式: 主图改的条目无此字段, 行为同旧)。
- 控制台批量改 → `applyBulkMutation(touchedSgIDs, mutator)`: 改前 `augmentTop` 存触及子图改前态, 改后 push 带改后态 → **一次 Ctrl+Z 全退** (主图+子图)。
- **手动改子图也可撤销**: `applyDraftMutation` 在子图层编辑时把活动子图也快照 (同机制)。
- 改动进编辑器草稿 (一步撤销), **Ctrl+S 才落盘**; touched 子图由现有 `useEditorSave` 一并存。

## 补全 (`completions.ts` `ytCompletionSource`)

上下文感知 (按光标前文本分流): `yt.`→顶层成员 / `yt.nodes.`·`yt.selected.`→数组方法 / 其它 `X.`→NodeHandle 成员。纯判定 `ytCompletionKind` 有单测。控制台编辑器复用共享 `<CodeEditor>`, 经 `scriptEditorExtensions` 的 `completionSource` 口注入。历史设计材料在 cold archive `2026-06-17-unified-code-editor`;本知识不依赖它。

## 边界 / 非目标

- 控制台**非真沙箱** (`new Function` 跑页面上下文): 本地单人工具, 接受。无硬超时 (死循环卡 UI, UI 有提示)。
- 编辑器 modal `dismissible=false`: 补全浮层挂 body 算"外部", 故禁点外部/Esc 关 (只 ✕/按钮)。
- `yt.ops` (脚本调 UI 动作) / 跨容器批量 / dry-run / 写容器变量 — 均未做, 有需求再加。
