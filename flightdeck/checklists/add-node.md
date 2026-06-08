---
status: active
last_updated: 2026-06-07
when_to_read: 新增 / 改一个节点 kind 前 (backend Spec → 前端渲染 → 面板 → i18n 全链路)
applies_to: [node, add-node, nodepkg, spec, palette, i18n, registry, frontend, backend]
when_to_update: 改节点新增链路任一环 (nodepkg.Spec 结构 / registry 注册 / palette 面板 / 前端渲染映射 / i18n 注入流程) 时
portable: false
---

# 加一个节点 kind — 全链路 Checklist

新增节点是**跨 Go + 前端多处**的事，漏一处就"代码在、能渲染、但用户加不进去 / 没翻译 / 没默认值"。按这份走，别凭记忆。

> 配套：pin 命名规范看 [node-spec-style.md](node-spec-style.md)；校验该写哪条管线看 incident [[2026-06-04-node-validation-pipeline-bifurcation]]；Geometry pin 值形状看 [[2026-06-04-geometry-pin-value-pct-shape]]。

## 1. 后端 — 节点实现 (`internal/nodes/<category>/<name>.go`)

- [ ] `func init() { node.Register(&X{}) }`。
- [ ] `Spec()`：`Kind`（PascalCase）、`Category`、`Inputs`/`Outputs`。exec-in pin 必须叫 `"In"`；pin 名 PascalCase（守卫测试 `TestSpecConsistency_*`）。Number Default 用 `json.Number("...")`。
- [ ] **同概念必须复用既有 pin 名** — 加 pin 前先查 [node-spec-style §9 Canonical pin 词汇表](node-spec-style.md)（屏幕区域=`ROI`、超时=`TimeoutMs`、命中分支=`Found`/`NotFound`、捕获=`Capture<字段>` 等）。lint 测不出"同概念异名"，全靠这张表 + 加完跑 `task nodes:pins` 核对（「命名分裂告警」段为空 = 没撞名）。
- [ ] **恰好一种 capability**：实现 `Runnable`(Run) / `RegionRunner`(RunRegion) / `Evaluator`(Evaluate) **之一**；或纯展示设 `IsVisualOnly: true`、图标记设 `IsGraphMarker: true`（这两类可零 capability）。注册时 `registry.go` 会校验。
- [ ] 字段默认值：`InputSpec.Default` —— 前端建节点时会经 `deriveDefaults` 收成 `{ literal: {...} }` 自动填进 `config.literal`。要有默认值就在这写。

## 1b. ⛔ 产出型节点必须加"输出捕获"框（硬约束，防 $sys 重生）

只要节点产出了**别人可能想消费的值**（exec 出口的 Data 字段 / 检测结果 / 计算值），就必须让用户能拿到。`$sys` 的病根是"产出藏在框架硬编码表里、跟节点声明脱钩"——焊死在节点 Spec 上：

- [ ] **新增产出型节点 / 给节点加 OutputData 时，必须同步决定该字段是否可捕获；默认可捕获。** 不允许"有产出但没有任何消费路径"。
- [ ] 捕获框 = 可选 String 输入，**只能一一对应 `OutputSpec.Data` 字段**（不许另立隐藏捕获字段）：
  ```go
  {Name: "Capture<字段>", Type: "String", Advanced: true, Semantic: "capture",
      Widget: node.WidgetSpec{Kind: "text"}}
  ```
- [ ] `Run()`（RegionRunner 在 `RunRegion`）在成功 `Fire()` 前，对该 exit **实际带的**每个值调 `node.Capture(ctx, in, "Capture<字段>", 值)`（`internal/node/capture.go`：trim 后非空才 `SetScoped(name,"auto",值)`）。error 早返不写；某 exit 没带的字段不写。
- [ ] 前端会按 `Semantic=="capture"` 把这些框聚成默认折叠的「输出捕获」分组（带数量徽章）——你只管在 Spec 声明 + i18n 给 label，渲染自动。
- [ ] 纯数据节点（`IsPureData`）不需要捕获框——它的输出本就能直连数据线 / 被 GetVar 读。

> 完整数据流模型（捕获 vs exec-data vs 纯数据直连）看 [2026-06-05-node-data-flow.md](2026-06-05-node-data-flow.md)。

## 2. 后端 — 包要被 blank-import (否则 init 不跑、节点不存在)

- [ ] 新 `internal/nodes/<category>` 包：确认 `main.go` + `internal/services/container/runtime/dispatch_v5_test.go` 里有 `_ "yotta/internal/nodes/<category>"`。已有 category 包加节点则无需动。

## 3. i18n — 文案 + 重新生成 catalog (catalog drift 测试会卡)

- [ ] `frontend/src/i18n/zh.ts` + `en.ts` 加 `node.<Kind>` 块：`label` / `description` / `input.<pin>.label`（dropdown 选项加 `input.<pin>.option.<value>`）。zh/en **对称**（parity 测试）。**文案规范看 [node-spec-style §10](node-spec-style.md)**：zh 人话不夹黑话、en sentence-case、option 要翻译、捕获框 `<字段>→变量`、时间 `(ms)`。
- [ ] vue-i18n 文案含 `{` `}` `|` `@` `$` 要转义（见 checklist [[vue-i18n-message-compiler-traps]]）；改完 `pnpm i18n:check` 的 `[compile]` 段会兜。
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

- [ ] **编辑期校验**（NodeInspector 红错）写在 `internal/services/container/validator.go` 的 `checkGraphPerKind` kind switch → `validateXxx(n)`。**不是**节点的 `Validate()` 方法（那只在 engine runtime 跑）。详见 incident [[2026-06-04-node-validation-pipeline-bifurcation]]。

## 8. 验证（全绿才算完）

- [ ] `go build ./... && go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... -count=1`
- [ ] `cd frontend && pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`
- [ ] `task build`
- [ ] 真机 smoke：在**侧边面板 + 右键菜单 + explorer 弹窗**都能找到并加进去；默认值/文案/渲染/分组标签都对。
