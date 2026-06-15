---
status: done
summary: "Spec C 实现计划 Part 2 (前端输出组 + 消费者审计)。① FE 单一来源 bindableFields(kind,config)=isPureData?[]:pinsFor().dataOut (与后端 BindableFields parity) + 单测; ② useVarMutations 5 处 (rename/count/listUsageNodeIDs/deleteVar-cascade/listUsageRefs) 从 config.literal[capture框] 改读 config.capture[字段], cascade 删 key 非空串 (落地精度#2), 可绑字段来源从 semantic==='capture' 改 bindableFields; ③ NodeInspector「输出」组方案 A (按钮绑+chip, 写 config.capture) 替掉旧 captureLiterals 折叠组; ④ i18n: 加 node.<kind>.output.<字段>.label (从旧 input.Capture* 迁) + 删旧 capture 输入键 + pnpm gen:node-i18n 重生成; ⑤ 清 FieldSchema.captureType/adapter 透传/semantic==='capture' 残留 + bindings 重生成。收尾: typecheck/test/build:dev/禁用色扫描 + 迁移扫描 + 真机 smoke (P1+P2 一个发布单元, 落地精度#7)。"
last_updated: 2026-06-15
implements: specs/2026-06-15-output-auto-capture.md
---

# 输出自动捕获 · 前端 + 消费者审计 Implementation Plan (Spec C · Part 2)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development 或 executing-plans 逐 task 实施。Steps 用 checkbox 跟踪。

**Goal:** 后端 Part 1 已把捕获绑定从「手声明 Semantic:capture 输入框」改成 `config.capture{字段→变量名}` 自动捕获 (两条写路径已落地)。Part 2 把前端补齐到同一约定: Inspector「输出」组统一一行式绑定 (方案 A 按钮绑+chip, 写 `config.capture`) + 消费者审计 (useVarMutations 5 处改读 config.capture, 否则删变量/重命名漏改捕获绑定 = 悬空) + 翻译 + 清旧 captureType 残留。

**当前破损态 (实证):** 后端 Part 1 删光 `Semantic:"capture"` 输入 → 前端 `isCaptureLit`(`semantic==='capture'`) 恒 false → captureLiterals 恒空 → **编辑器输出捕获 UI 暂不可用**; `useVarMutations` 仍读 `config.literal[capture框]` (已不存在的字段) → 删变量/改名不再清捕获绑定。这正是 spec 落地精度#7「P1+P2 一个发布单元」要补的中间态。

**Architecture (读源码确认):**
- **单一来源 (落地精度#1):** 后端 `node.BindableFields(spec)` = 非 IsPureData 节点所有 exec 出口的 Data 字段名 (去重)。前端等价 = `isPureData ? [] : pinsFor(kind,config).dataOut` —— 已核 `splitOutputs` 把 exec 出口 Data flatten 进 `s.dataOut`, 且非纯数据节点无独立 data 输出 pin (8 个 IsPureData 节点之外的 OutputSpec 数据全走 exec 出口 Data 字段, `DataField` 注释「仅 Type=Exec 出口有」), 故二者集合一致。region 节点 (Loop/ForEach) 的 Index/Item 已由 Part 1 声明为 Body 出口 Data 字段 → 自动进 dataOut → 自动可绑。
- **存储:** `config.capture: map[字段名→变量名]` (顶层, 跟 `config.literal` 平级)。空/缺=无绑定。验证由 Part 1 `validateCaptureRefs` 负责 (字段∉BindableFields→INVALID_PIN; 变量名未声明→INVALID_VAR_REF; 空绑定跳过)。
- **i18n:** zh.ts/en.ts 是单一来源 (`node.<kind>.output.<字段>.label`)。`node-i18n.json` 由 `pnpm gen:node-i18n` (esbuild 打包 zh.ts → extract-node-i18n.mjs 抽 node.* 块) 生成、go:embed 进 catalog 供 MCP。output 块只抽 label (无 hint)。
- **bindings:** `frontend/bindings/` 是 wails gitignore 生成物, `task dev`/`task build` 自动重生成。Part 1 删了 `InputSpec.CaptureType` 但 bindings 未重生成 (models.js 仍有 captureType) → 现 FE typecheck 仍绿 (读 stale 字段)。T5 清读取点 + 重生成后 captureType 全消失。
- **边界:** 不碰 vue-flow 画布/`ContainerFlowNode`/连线/pin/inline (沿用 Spec B 边界)。绑定全在 Inspector「输出」组。

**Tech Stack:** Vue 3 + NuxtUI v4 · TS · pnpm。`frontend/src/components/containers/{NodeInspector.vue, nodeRegistry/, inline/VarNameInput.vue}` · `frontend/src/composables/containerEditor/{useVarMutations.ts, bindableFields.ts(新)}` · `frontend/src/i18n/{zh,en}.ts` · `frontend/scripts/extract-node-i18n.mjs`。

---

## 验证基线 (本计划通用)

- 每 Task 收尾: `pnpm typecheck` 绿 (改的文件无 TS 错); 改了纯函数/composable 的跑对应 `pnpm test <file>` 绿。
- 全计划收尾: `pnpm typecheck` + `pnpm test` + `pnpm build:dev` 全绿; **禁用色扫描零行** (ui.md 硬约束); grep 残留零 (见 T5)。
- 预存非回归 (build.md / cockpit): `pnpm lint` 预存 ~18 错 (oxlint 1.64, 全在 HEAD 同款代码) —— 不算本次回归, 但**别新增**。i18n residue 扫描预存 39 (28 misc + 11 editorTheme zh 文案映射, 误报类)。
- **真机 smoke 在收尾** (P1+P2 一个发布单元, 落地精度#7): 给 ①DetectColor(Count/Center, 路径①) ②PlayClip(Error/Code, 旧绑不了的) ③Loop(Index, 路径②) 各绑变量 → 跑 → GetVar/日志验写入; 未命中出口不覆盖旧值; 删一个 Loop.Index 绑的变量 → 验 config.capture 键被删 (不悬空); 翻译名一致。

## 受影响文件总览

- `composables/containerEditor/bindableFields.ts` — **新增** (T1, 单一来源纯函数) + `bindableFields.spec.ts`。
- `composables/containerEditor/useVarMutations.ts` — T2 (5 处改读 config.capture) + `useVarMutations.spec.ts` 改写。
- `components/containers/NodeInspector.vue` — T3 (删 captureLiterals 折叠组 + 新「输出」组方案 A) + script 清理。
- `components/containers/inline/VarNameInput.vue` — T3 复用 (保留 captureType prop = 字段类型, 新建变量类型推断; 无改动或小改)。
- `i18n/zh.ts` · `i18n/en.ts` — T4 (加 output.<字段>.label, 删 input.Capture* 键, UI 新键)。
- `scripts/extract-node-i18n.mjs` 跑 `pnpm gen:node-i18n` → `internal/catalog/node-i18n.json` 重生成 (T4)。
- `components/containers/nodeRegistry/{index.ts, adapter.ts, adapter.test.ts}` — T5 (删 FieldSchema.captureType + adapter 透传 + 测试)。

---

## Progress

current: **done (代码全落地, 待真机 smoke)**。typecheck + 344 测试 + build:dev 全绿; 禁用色零新增; T5 三条 grep 零残留 (captureType 仅 VarNameInput 自有 prop)。bindings 重生成 (InputSpec 去 captureType)。迁移已跑 (真机数据)。

- T1 done — 8f.. bindableFields 单一来源 (isPureData?[]:pinsFor().dataOut, 与后端 BindableFields parity) + 4 单测
- T2 done — useVarMutations 5 处改读 config.capture; cascade 删 key 非空串; spec.ts 改写 (注册假 spec)
- T3 done — NodeInspector「输出」组方案 A (按钮绑+chip, 写 config.capture) 替 captureLiterals 折叠组
- T4 done — i18n: 共享 `inspector.output.field` 字典 (output 字段名→译名, DRY) + outLabel 共享 fallback + 删全部 input.Capture* 键 (zh+en) + 重生成 node-i18n.json (106 节点, catalog drift 绿)
- T5 done — 删 FieldSchema.captureType + adapter 透传 + adapter.test.ts 用例; `task common:generate:bindings` 重生成
- 迁移 done — `tmp/migrate-capture.mjs` (per-node 映射) 跑 19 文件: DualColorBarTrack 3 真绑定 (InnerX/OuterWidth/OuterX→_bar*) 搬进 config.capture, 1 空 CaptureResult 删; 备份 `bin/data.bak-spec-c`; 残留零、_bar* 变量已声明 → validator 清
- 收尾 — **待真机 smoke** (用户跑 task dev): DetectColor(Count/Center) / PlayClip(Error/Code) / Loop(Index) 绑变量写入 + 未命中留旧值 + 删变量清绑定 + 翻译名; DualColorBarTrack 迁移后绑定仍生效

> **T4 与 spec §4 的偏差 (有意)**: spec 写 `node.<kind>.output.<字段>` 逐 kind label; 实现改用**共享 `inspector.output.field` 字典** (字段名近乎唯一, 译名仍精确) + outLabel fallback —— 同样达成"编辑器显中文、不显英文 pin 名", 但 DRY (一处字典 vs 13 kind × 2 locale 重复)、少维护面。代价: node-i18n.json (MCP catalog) 不带这些字段译名 (仅结构序列化, 非用户面)。exec 出口仍走既有 per-kind `output.<exit>.label`。

---

## Task 1: 前端单一来源 `bindableFields(kind, config)` + 单测

落地精度#1: 前端 Inspector / useVarMutations 对「某 kind 哪些字段可绑」走**同一份派生**, 与后端 `BindableFields` 集合一致。

**Files:** `frontend/src/composables/containerEditor/bindableFields.ts` (新) + `bindableFields.spec.ts` (新)

- [ ] **Step 1: 写 bindableFields.ts**

```ts
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import { pinsFor } from '@/components/containers/pinSpec'

// 某节点可被「输出捕获」绑定到变量的字段名 (Spec C 前端单一来源)。
// = 非纯数据节点 (isPureData=false) 所有 exec 出口的 Data 字段 (pinsFor().dataOut 已 flatten)。
// 纯数据节点 (GetVar/Now/Expr/PureFunc/...) 无可绑字段 —— 其输出是连线源, 非捕获。
// 与后端 node.BindableFields 集合一致; Inspector「输出」组 + useVarMutations 共用。
export function bindableFields(
  kind: string,
  config?: Record<string, unknown> | null,
): string[] {
  const s = getSpec(kind)
  if (!s || s.isPureData) return []
  return pinsFor(kind, config).dataOut
}
```

- [ ] **Step 2: 单测** `bindableFields.spec.ts` —— 用 registry `register`/`__resetForTests` 注册最小假 spec (仿 adapter.test.ts 风格), 断言:
  - 非纯数据 + exec 出口带 Data 字段 (DetectColor 风格: dataOut={Count,Center}) → `['Count','Center']`。
  - region 风格 (Loop: dataOut={Index}) → `['Index']`。
  - 纯数据 (isPureData=true, dataOut={value}) → `[]`。
  - 未注册 kind → `[]` (getSpec undefined)。
  > register 需要完整 NodeKindSpec —— 抽个 `makeSpec(partial)` helper 填默认 (execIn/execOut/dataIn/dataOut/fields/defaults/visual/labelZh/...)。

- [ ] **Step 3: 验证** `pnpm test bindableFields` 绿。`pnpm typecheck` 绿。

- [ ] **Step 4: commit** `feat(editor): bindableFields single source (Spec C T1, FE/backend parity)`

---

## Task 2: useVarMutations 5 处改读 config.capture (消费者审计)

spec §2 + 落地精度#2/#9: config.capture 是新 var-ref 站。旧实现把捕获绑定当 `config.literal[capture框]` (semantic==='capture' 派生字段) 消费, 被 5 处用。改存储后全要改, 漏一处 = 删变量不清绑定 (悬空) / 重命名漏改 / 查引用漏列 = incident [[2026-05-29-storage-convention-consumer-audit-gap]] 的坑。

**Files:** `frontend/src/composables/containerEditor/useVarMutations.ts` + `useVarMutations.spec.ts`

- [ ] **Step 1: 换数据源 helper**

删 `captureFieldsOf`(NODE_FIELD_SCHEMAS semantic==='capture' 派生) + `captureLiteralOf` 对捕获字段的用法。改为读 `config.capture`:

```ts
import { bindableFields } from './bindableFields'

// 读 node.config.capture[field] 的绑定变量名 (string), 不存在返 ''.
function captureVarOf(node: GraphNode, field: string): string {
  const cap = node.config?.capture as Record<string, unknown> | undefined
  return typeof cap?.[field] === 'string' ? (cap[field] as string) : ''
}
```
> `varNodeName`/`isContainerScope` 仍读 `config.literal.VarName`/`config.literal.Scope` (VAR_NODE_KINDS 不变, 那是 pin 字面量不是捕获)。只有捕获分支改。`captureLiteralOf` 若仅剩 VAR_NODE_KINDS 用 (VarName/Scope) → 保留改名为它原义; 否则按引用情况收敛。

- [ ] **Step 2: 5 处逐个改 (捕获分支)** —— 「可绑字段」从 `captureFieldsOf(kind)` 改 `bindableFields(node.kind, node.config)`; 「绑定值」从 `config.literal[field]` 改 `captureVarOf(node, field)`:
  - **renameVar** (现 L100-109): `for f of bindableFields(...)`: `if (cap[f] === oldName) cap[f] = newName` (写回 `node.config.capture`)。
  - **countUsage** (现 L125-128): `if (captureVarOf(node,f) === name) count++`。
  - **listUsageNodeIDs** (现 L144-149): 命中即 matched。
  - **deleteVar cascade** (现 L174-183, **落地精度#2 删 key 非空串**): `for f of bindableFields(...)`: `if (cap[f] === name) delete cap[f]` (delete 键, 不是 `=''`)。
  - **listUsageRefs** (现 L197-204): 命中 push `{nodeID, access:'write', kind}`, 同节点多字段命中只计一条 (break)。
  > 写 config.capture 前 nil-safe: `const cap = (node.config!.capture ??= {}) as Record<string,string>` (rename/cascade 写时)。读时 nil-safe 走 captureVarOf。

- [ ] **Step 3: 改写 `useVarMutations.spec.ts` 捕获用例** (现 §Task12/13, 4 个用例) —— 不再注 `NODE_FIELD_SCHEMAS['CheckTemplate']`, 改:
  1. `beforeEach`: `setActivePinia` + 注册假 spec `register({kind:'CheckTemplate', isPureData:false, execOut:['Found'], dataOut:{Found:'bool'}, ...makeSpec})`; `afterEach: __resetForTests()`。
  2. 节点 config 从 `{literal:{CaptureFound:'hp'}}` 改 `{capture:{Found:'hp'}}`。
  3. countUsage / renameVar (`cap.Found` 改名) / listUsageRefs (write) / **deleteVar cascade → 断言 `'Found' in cap === false`** (键被删, 非空串)。
  > 复用 T1 的 `makeSpec` helper (抽到共享 test util 或各文件本地)。

- [ ] **Step 4: 验证** `pnpm test useVarMutations` 绿。`pnpm typecheck` 绿。

- [ ] **Step 5: commit** `refactor(editor): useVarMutations consumes config.capture not literal (Spec C T2 consumer audit)`

---

## Task 3: NodeInspector「输出」组方案 A (按钮绑 + chip)

spec §3: 合并旧「输出捕获 (VarNameInput 折叠组)」+「出口 pin 速览 (只读)」→ 统一一行式。删 9d66558 那段 captureLiterals/captureOpen 折叠组。

**Files:** `frontend/src/components/containers/NodeInspector.vue`

- [ ] **Step 1: 删旧捕获机制 (script + template)**
  - script: 删 `isCaptureLit` / `captureLiterals` / `captureFilledCount` / `captureOpen` (现 L726-737)。`normalLiterals` (现 L730) → 直接用 `dataInLiterals` (无 capture 输入了, 二者等价) —— 把「数据输入」section (现 L498) 的 `normalLiterals` 引用改 `dataInLiterals`, 删 normalLiterals computed。
  - template: 删整个「输出捕获」folding `<section>` (现 L563-602, 含 captureType 读取)。
  - 修 no_config 占位 (现 L556): 去掉 `&& captureLiterals.length === 0` 条件。

- [ ] **Step 2: 新「输出」组列表 (替现 L604-625 的只读速览)**

`SectionHeader group_outputs` 之下, 渲染:
  - **可绑产出行** (`bindableFields(kind, config)` 每个字段): 一行 `翻译名 (类型) [绑定控件]`。翻译名 = `t('node.<kind>.output.<字段>.label')` (T4 加), fallback 字段名; 类型 = `PIN_SPECS[kind].dataOut[字段]`。绑定控件 (方案 A):
    - 未绑且非编辑态 → 「+ 绑定变量」`UButton` (xs ghost), 点 → 加该字段进 `editing` Set。
    - 已绑或编辑态 → `VarNameInput` (`:model-value="getCapture(字段)"`, `:declared-vars`, `:capture-type="字段类型"`, `scope="auto"`, `@update:model-value` 写 `setCapture(字段,v)`, `@declare-var` 透传)。已绑显示在 input 旁一个 ✕ `UButton` → `clearCapture(字段)` (删 config.capture[字段] 键 + 移出 editing)。
    - 每行底部一句 hint `t('inspector.output.stale_hint')` ("该变量仅在此出口触发时更新, 未触发保留上次值") (落地精度#6); Found 字段额外/替换 `t('inspector.output.found_hint')` ("Found = 本次是否命中, 始终更新")。
  - **exec 出口行** (只读, `outPins.exec`): 沿用现样式, 名走 `t('node.<kind>.output.<exec>.label')` 有则用、无则原名。
  - **纯数据节点** (bindableFields=[]): 渲染 `outPins.data` 只读行 (现样式), 不可绑 (理由 spec 落地精度#5: Evaluator 无 fire/routeResult hook; 存别的变量用 SetVar)。
  - 全空 → `outputs_none` 占位。

- [ ] **Step 3: 新 script helpers**

```ts
const editing = ref(new Set<string>())  // 正在绑/改的字段
const bindable = computed(() => props.node ? bindableFields(props.node.kind, props.node.config) : [])
const dataTypeOf = (field: string) => String(PIN_SPECS[props.node!.kind]?.dataOut?.[field] ?? '')
function getCapture(field: string): string {
  const cap = props.node?.config?.capture as Record<string, unknown> | undefined
  return typeof cap?.[field] === 'string' ? (cap[field] as string) : ''
}
function setCapture(field: string, varName: string) {
  if (!props.node) return
  const cfg = { ...props.node.config }
  const cap = { ...(cfg.capture as Record<string, string> | undefined) }
  if (varName.trim() === '') delete cap[field]   // 空 = 解绑, 删 key (落地精度#2)
  else cap[field] = varName
  cfg.capture = cap
  emit('update', cfg)
}
function clearCapture(field: string) { setCapture(field, ''); editing.value.delete(field) }
```
> 写 config.capture 经现有 `emit('update', cfg)` 通道 (跟 setLiteral 同, 父图 deep watch 标 dirty)。`isPureData` 判定走 `getSpec(kind)?.isPureData` —— bindable=[] 即纯数据, 不另判。

- [ ] **Step 4: 验证** `pnpm typecheck` 绿。手核: DetectColor 显 Count/Center 可绑、Loop 显 Index 可绑、GetVar 显 value 只读、PlayClip 显 Error/Code 可绑 (旧绑不了的)。(渲染真机在收尾 smoke。)

- [ ] **Step 5: commit** `feat(editor): Inspector outputs group unified binding (Spec C T3, scheme A buttons+chips, config.capture)`

---

## Task 4: i18n — output 字段 label + 删旧 capture 键 + 重生成 node-i18n.json

spec §4: Data 字段加 `node.<kind>.output.<字段>.label`; 旧捕获框中文 label 搬过来复用 (去「→变量」后缀), 删旧捕获 i18n 键。

**Files:** `frontend/src/i18n/zh.ts` · `frontend/src/i18n/en.ts` (+ `pnpm gen:node-i18n` → `internal/catalog/node-i18n.json`)

- [ ] **Step 1: 逐 kind 加 output 字段 label (zh + en)** —— 对每个有可绑 Data 字段的 kind, 在其 `output:{}` 块补 Data 字段 label (沿用旧 input.Capture* 的中文/英文语义, 去 "→变量"):

| kind | output Data 字段 label (zh) ← 迁自 input.Capture* |
|---|---|
| DetectColor | Count='命中像素数' Center='命中中心点' |
| DetectColorHSV | Count='命中像素数' Ratio='命中比例' (核实际字段名 CapturePixelCount→? 读 spec) |
| DetectColorBlobs | PrimaryCenter='首块中心' PrimaryArea='首块面积' BlobCount='块数' (核字段名) |
| RoiColorScan | ClusterCount='段数' Clusters='各段信息' (+Count?) |
| DualColorBarTrack | InnerX/OuterX/OuterWidth/Confidence/InnerPx/OuterPx (6, 核字段名) |
| Screenshot | Path='截图路径' |
| WaitTemplate | Point='命中点' Conf='置信度' Found='是否命中' |
| CheckTemplate | Point='命中点' Found='是否命中' |
| ClickTemplate | Point='点击点' Found='是否命中' |
| StopwatchRead | Elapsed='已用时' (核字段/kind 名) |
| Script | Result='结果' (核字段名) |
| Loop | Index='循环序号' |
| ForEach | Item='当前元素' Index='序号' |

> **逐 kind 读 backend spec 文件确认精确 Data 字段名** (头号铁律, 不照搬本表猜名): `grep -n 'DataField\|wtData\|dcData...' internal/nodes/<file>.go` 或读 OutputSpec.Data。字段名写错 = label 不显 (走 fallback 原名)。exec 出口已有 label (Found/Done/Fail) 保留; Found 既是 exec 出口又是 Data 字段, 同名共用一条 label (语义一致)。

- [ ] **Step 2: 删旧 capture 输入键 (zh + en)** —— 删所有 `node.<kind>.input.Capture*` 键 (CaptureCount/CaptureCenter/CaptureFound/CapturePoint/CaptureResult/CapturePath/CaptureIndex/... 见 Step1 grep 命中 + zh.ts L647/689/704.../951 区域)。二号铁律: 删干净不留。

- [ ] **Step 3: UI 新键 (zh + en, `inspector` 块)** —— 加: `output.bind`='绑定变量' / `output.stale_hint`='该变量仅在此出口触发时更新, 未触发保留上次值' / `output.found_hint`='Found = 本次是否命中, 每次都会更新' / `output.unbind_tooltip`='解除绑定'。删不再用的 `inspector.capture_section` (旧折叠组头, T3 已删消费者)。
  > vue-i18n message-compiler traps (checklist [[vue-i18n-message-compiler-traps]]): label 是纯中文/英文短语, 无 `{}/@/|/$` 特殊字符, 低风险; 不派 subagent 批量写。

- [ ] **Step 4: 重生成 node-i18n.json** `pnpm gen:node-i18n` (catalog 漏 kind 时 Go `TestBuildWithI18n_AllKindsLabeled` 会 fail) → diff 确认旧 Capture* 字段消失、output Data 字段进入。

- [ ] **Step 5: 验证** `pnpm typecheck` 绿; `go test ./internal/catalog/...` 绿 (i18n drift guard); UI 缺 label 不报错 (fallback 字段名)。

- [ ] **Step 6: commit** `feat(i18n): node output field labels + drop capture input keys + regen node-i18n (Spec C T4)`

---

## Task 5: 清 captureType/semantic-capture 残留 + bindings 重生成

二号铁律: Part 1 删了 `InputSpec.CaptureType` + 所有 `Semantic:"capture"` 输入 → 前端对应透传/字段是死代码, 删。bindings 重生成后 `i.captureType` 会成 TS 错, 必须同批清。

**Files:** `frontend/src/components/containers/nodeRegistry/{index.ts, adapter.ts, adapter.test.ts}` (+ bindings 重生成)

- [ ] **Step 1: 删 FieldSchema.captureType** —— `nodeRegistry/index.ts` L77-80 删 `captureType?: string` 字段 + 注释 (semantic 字段保留 —— 'varname' 仍用)。
- [ ] **Step 2: 删 adapter 透传** —— `adapter.ts` L166 删 `if (i.captureType) f.captureType = i.captureType`; L163 注释去掉 "capture" 措辞 (semantic 透传仍在, 为 'varname')。
- [ ] **Step 3: 删 adapter.test.ts captureType 用例** —— L80-91 「captureType 透传」整个 it 删 (字段没了)。
- [ ] **Step 4: VarNameInput 保留** —— 它的 `captureType` **prop** (L47) 是组件自有入参 (新建变量类型推断 + 类型冲突警告), 非 FieldSchema 字段, **不动**; T3 已改成传 Data 字段类型。
- [ ] **Step 5: bindings 重生成** —— `task dev` 或 `task build` (自动 regen `frontend/bindings/`); 或确认其重生成机制后跑。重生成后 `InputSpec` 无 captureType。
- [ ] **Step 6: grep 零残留**
```bash
grep -rn "captureType" frontend/src/        # 期望: 仅 VarNameInput (prop + helpers + spec) 命中, 无 FieldSchema/adapter
grep -rn "semantic *=== *'capture'\|semantic *=== *\"capture\"" frontend/src/   # 期望: 零
grep -rn "captureLiterals\|isCaptureLit\|captureFieldsOf" frontend/src/         # 期望: 零
```
- [ ] **Step 7: 验证** `pnpm typecheck` + `pnpm test` 绿 (bindings 重生成后)。
- [ ] **Step 8: commit** `refactor(editor): drop dead captureType/semantic-capture plumbing (Spec C T5)`

---

## 收尾

- [ ] 全绿: `pnpm typecheck` + `pnpm test` + `pnpm build:dev` + 禁用色扫描零行。T5 三条 grep 零残留。
- [ ] **迁移扫描 (Part 3, 条件化)**: `grep -rn 'Capture' bin/data/containers/ 2>/dev/null` (dev 机真实容器) + 已知 fixture (Part 1 89e16c1 已迁 state_FISHING)。有非空 `config.literal["Capture<X>"]` → 写一次性脚本 per-node 映射搬到 `config.capture` (映射 = T4 Step1 表 + spec §5); 没有 → 跳过 + log 说明。
- [ ] **真机 smoke** (P1+P2 一个发布单元, 落地精度#7; 见验证基线 smoke 步): 用户跑 `task dev` 验 ①DetectColor ②PlayClip ③Loop 绑变量写入 + 未命中留旧值 + 删变量清绑定 + 翻译名。
- [ ] 更新 plan status=done + cockpit; spec graduate (docs/ 已有 variable-system / node-data-flow, 视情况补「输出捕获」常驻知识) + 归档 spec/plans (landing)。

## Self-Review (spec 覆盖核对)

- spec §2 消费者审计 (useVarMutations 5 处 + cascade 删 key) → T2 (落地精度#2/#9)。✅
- spec §3 方案 A 输出组 (按钮绑+chip, 纯数据不可绑, tooltip) → T3 (落地精度#5/#6)。✅
- spec §4 翻译 (output.<字段>.label 迁自旧 capture + 删旧键 + 重生成) → T4。✅
- spec 落地精度#1 单一来源 → T1 (FE bindableFields, 与后端 BindableFields parity, 已核 dataOut==exec出口Data)。✅
- spec §5 迁移 (条件化 per-node 映射) → 收尾 (Part 3)。✅
- 二号铁律 (删 captureType 死代码) → T5。✅
- **类型一致**: `bindableFields(kind,config):string[]` / `config.capture: map[字段→变量名]` (FE 读写一致) / `captureVarOf`/`getCapture`/`setCapture` 贯穿 T2/T3。✅
- **占位扫描**: T4 Step1 表标注「核实际字段名」为确定性核对步 (读 spec, 非 TBD); 各 Step 给确切文件/行号区域/代码/命令。✅
