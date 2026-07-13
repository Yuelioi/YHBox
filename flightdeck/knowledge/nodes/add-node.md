---
kind: checklist
summary: "加一个节点 kind 的全链路机械步骤 —— backend Spec、blank-import、i18n、前端渲染、3 个面板、校验、验证"
activation: action
read_when: "新增 / 改一个节点 kind 前 (backend Spec → 前端渲染 → 面板 → i18n 全链路)"
recheck_when: "改节点新增链路任一环 (nodepkg.Spec 结构 / registry 注册 / palette 面板 / 前端渲染映射 / i18n 注入流程) 时"
---
# 加一个节点 kind — 全链路 checklist
新增节点是**跨 Go + 前端多处**的事，漏一处就"代码在、能渲染、但用户加不进去 / 没翻译 / 没默认值"。按这份走，别凭记忆。

> 配套：pin 命名规范看 [node-spec-style.md](node-spec-style.md)；校验该写哪条管线看 [node-validation-pipeline-bifurcation.md](node-validation-pipeline-bifurcation.md)；Geometry pin 值形状看 [geometry-pin-value-pct-shape.md](geometry-pin-value-pct-shape.md)。

## 1. 后端 — 节点实现 (`internal/nodes/<category>/<name>.go`)

- [ ] `func init() { node.Register(&X{}) }`。
- [ ] `Spec()`：`Kind`（PascalCase）、`Category`、`Inputs`/`Outputs`。exec-in pin 必须叫 `"In"`；pin 名 PascalCase（守卫测试 `TestSpecConsistency_*`）。Number Default 用 `json.Number("...")`。
- [ ] **同概念必须复用既有 pin 名** — 加 pin 前先查 [node-spec-style §9 Canonical pin 词汇表](node-spec-style.md)（屏幕区域=`ROI`、超时=`TimeoutMs`、命中分支=`Found`/`NotFound`、捕获=`Capture<字段>` 等）。"同概念异名"（语义层，如区域口叫 `Zone`）lint 测不出，全靠这张表 + review；但**机械分裂**（拼写撞名 / 同角色同名不同类型）已有 guard `TestNoPinNameSplit`，`go test ./internal/catalog/`（下方验证步骤本就含）会卡，也可 `task nodes:pins` 看明细。
- [ ] **恰好一种 capability**：实现 `Runnable`(Run) / `RegionRunner`(RunRegion) / `Evaluator`(Evaluate) **之一**；或纯展示设 `IsVisualOnly: true`、图标记设 `IsGraphMarker: true`（这两类可零 capability）。注册时 `registry.go` 会校验。
- [ ] 字段默认值：`InputSpec.Default` —— 前端建节点时会经 `deriveDefaults` 收成 `{ literal: {...} }` 自动填进 `config.literal`。要有默认值就在这写。Number/Integer/Duration 用 `json.Number("...")`；**Duration 的数字按毫秒解析**（`in.Duration`：json.Number → ms），所以 1s 写 `json.Number("1000")`。
- [ ] ⛔ **`Default` 与 `Required` 互斥 —— 加了 `Default` 就别再留 `Required: true`**。`validateRequired` 查的是 `in.Has(name)`，而 inputs 已 merge 进 `rn.Defaults`（`engine.go` `newInputs`）→ 有默认的 pin 永远 `Has`==true → `Required` 成**死标**（永不报 `REQUIRED_FIELD_MISSING`），留着是误导性 cruft。想"拖出来即可用又不许清空"只能靠默认值 + Run 里的运行期校验（如 Sleep `if d<=0`），Required 起不到护栏作用。
- [ ] ⚠ **结构化 / 变长输入要配 `Schema`，别留裸 JSON**（颜色签名 Signature 踩过，2026-06-18 补）：输入是对象 / 定长向量 / 同质变长列表 / 颜色范围 / 几何这类**结构化数据**时，给 `InputSpec.Schema` 配 `node.ObjSchema` / `TupleSchema` / `ArraySchema` / `GeometrySchema`（见 [node-spec-style §9](node-spec-style.md)）→ 前端 `StructuredInput` 自动渲染**双段**（逐字段/逐项表单 + 「添加/删除」+ 可切整组 JSON）。**只给 `Type:"JSON" + Widget:json`** 而不配 Schema → Inspector 显示成**裸 JSON 文本框**，用户看不出该填什么结构。子字段 label 走 `input.<Pin>.<fieldKey>.label`（变长列表各项共用、不带下标）。（注：引脚类型徽章显 `any` 是另一回事 —— Geometry/JSON 在 `PinType` union 里都归 `any`，配不配 Schema 都改不了，属既有 cosmetic。）

## 1b. ⛔ 产出型节点：声明可绑 Data 字段（Spec C `config.capture` 模型）

只要节点产出**别人可能想消费的值**，就必须让用户能拿到。**⚠ Spec C T4（`33fa43f`，2026-06-15）已整体删除旧的 `Capture<字段>` 输入框 + `node.Capture` 助手 + `Semantic:"capture"`** —— 捕获现在全由框架做，**节点零捕获代码**。别照老记忆加捕获框（已不存在，会复活删掉的东西 = 踩二号铁律）：

- [ ] **产出 = 在 exec 出口声明 `OutputSpec.Data []DataField`**（如 `Found` 带 `Point`/`Conf`），`Run()` 里 `ctx.Out(exit).Set(field, 值)...Fire()`。**不加** `Semantic:"capture"` 输入框，**不调** `node.Capture`（这俩已不存在）。
- [ ] 捕获到变量由框架自动：用户在 Inspector 把 Data 字段绑到变量 → 存进 `node.config.capture`（`map[字段]→变量`）→ fire 时 `dispatch_v5.applyCaptures` 自动写（scope `auto`；只写该 exit **实际带的**字段，未带不写、变量留旧值——稀疏 data 天然保证）。
- [ ] 可绑字段 = `nodepkg.BindableFields(spec)`（从 exec 出口 Data 字段派生）；`internal/services/container/validator_capture_refs.go` 校验绑定（字段须可绑 + 变量须已声明）。
- [ ] 前端「输出」组按 Data 字段自动渲染绑定 UI——你只管在 Spec 声明 Data + i18n 给 Data 字段 label。
- [ ] 纯数据节点（`IsPureData`）输出本就能直连数据线 / 被 GetVar 读，与此节无关。
- 源码锚点：`runtime/dispatch_v5.go applyCaptures` · `validator_capture_refs.go` · `nodepkg.BindableFields`。**范式：`internal/nodes/detect/detect_color_blobs.go`**（只声明 Data + Set，无捕获框）。

> 完整数据流模型（config.capture vs exec-data vs 纯数据直连）看 [node-data-flow.md](node-data-flow.md)。

## 2. 后端 — 包要被 blank-import (否则 init 不跑、节点不存在)

- [ ] 新 `internal/nodes/<category>` 包：确认 `main.go` + `internal/services/container/runtime/dispatch_v5_test.go` 里有 `_ "github.com/yottaapp/yotta/internal/nodes/<category>"`。已有 category 包加节点则无需动。

## 3. i18n — 文案 + 重新生成 catalog (catalog drift 测试会卡)

- [ ] `frontend/src/i18n/zh.ts` + `en.ts` 加 `node.<Kind>` 块：`label` / `description` / `input.<pin>.label`（dropdown 选项加 `input.<pin>.option.<value>`）。zh/en **对称**（parity 测试）。**文案规范看 [node-spec-style §10](node-spec-style.md)**：zh 人话不夹黑话、en sentence-case、option 要翻译、输出捕获 `<字段>→变量`、时间 `(ms)`。
- [ ] ⚠ **出口 + Data 字段也要译，别只译输入**（本批 3 节点踩过，2026-06-18 补）：`output.<出口或Data字段名>.label` —— 每个 exec 出口（`Found`/`NotFound`/`Timeout`…）**和**每个 `OutputSpec.Data` 字段（`Matches`/`PrimaryPoint`/`Conf`/`Text`…）都要给 label，否则图节点出口引脚 + Inspector「输出」组显**英文裸字段名**。这些 key 经 `gen:node-i18n` 抽进 `node-i18n.json`（覆盖图节点 PIN_SPECS + inspector）。结构化输入的子字段 label 见 §1 结构化输入条 + [node-spec-style §10](node-spec-style.md)。
- [ ] vue-i18n 文案含 `{` `}` `|` `@` `$` 要转义（见 [vue-i18n-message-compiler-traps.md](../frontend/vue-i18n-message-compiler-traps.md)）；改完 `pnpm i18n:check` 的 `[compile]` 段会兜。
- [ ] **跑 `cd frontend && pnpm gen:node-i18n`** 重新生成 `internal/catalog/node-i18n.json`（从 zh.ts 抽取）。漏跑 → `go test ./internal/catalog/` 的 drift 守卫 FAIL。

## 4. 前端 — 渲染

- [ ] **普通节点**：无需写组件。`ContainerFlowNode` 按 backend Spec（pin/group/默认值经 RPC + adapter）自动渲染。确认 `pinSpec.ts` 的 `PIN_SPECS` 有这个 kind（多数自动派生）。
- [ ] **自定义渲染**（极少，如 CommentBox sticky note）：写独立组件，在 `ContainerEditorView.vue` 的 `nodeTypes` 里映射 `<Kind>: YourComponent`，并在 `PIN_SPECS` 派生 nodeTypes 处把它从共享 `ContainerFlowNode` 排除。

## 5. 面板 / 选择器 — ⚠ 有 3 个，各自独立过滤 (最容易漏)

节点要能被用户加，得在**全部** 3 个选择器里出现。它们**统一只按 `excludeFromPalette` 过滤**：

- `components/containers/NodePalette.vue`（侧边面板）
- `components/containers/NodeExplorerModal.vue`（explorer 弹窗）
- `components/containers/InlineContextMenu.vue`（画布右键菜单）

- [ ] **普通节点**：`adapter.ts` 不会给它置 `excludeFromPalette` → 三处自动显示，无需动。
- [ ] **visual-only / marker 节点想进面板**（如 CommentBox）：在 `nodeRegistry/adapter.ts` 的 `excludeFromPalette` 赋值处**例外放行该 kind**（`(isGraphMarker || isVisualOnly) && kind !== 'X'`）。**不要**在某个选择器里单独加 `kind !== 'X'`——`excludeFromPalette` 是唯一权威，三处都只读它。（踩过坑：只改了 NodePalette，explorer/右键菜单仍不显示。）
- [ ] 反之，要**隐藏**某节点（专用 UI 创建，如 SubgraphInput）：靠 `isGraphMarker`/`isVisualOnly` → adapter 自动置 `excludeFromPalette`。

## 6. 分组标签

- [ ] `Category` → group 经 `adapter.ts` 的 `GROUP_MAP`。group 标签 i18n 走 `nodeGroup.*`：`NodePalette.vue` 的 `GROUP_LABEL` + `useNodeGroupColor.ts` 的 `GROUP_I18N_KEY` **都要指向 zh/en 里真存在的 `nodeGroup.<key>`**。（踩过坑：`GROUP_I18N_KEY` 指了不存在的 `nodeGroup.system_subgraph` → 组标题显原始 key。）

## 7. 校验（按需）

- [ ] **编辑期校验**（NodeInspector 红错）写在 `internal/services/container/validator.go` 的 `checkGraphPerKind` kind switch → `validateXxx(n)`。**不是**节点的 `Validate()` 方法（那只在 engine runtime 跑）。详见 [node-validation-pipeline-bifurcation.md](node-validation-pipeline-bifurcation.md)。

## 8. 验证（全绿才算完）

- [ ] `go build ./... && go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... -count=1`
- [ ] `cd frontend && pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`
- [ ] `task build`
- [ ] 真机 smoke：在**侧边面板 + 右键菜单 + explorer 弹窗**都能找到并加进去；默认值/文案/渲染/分组标签都对。
