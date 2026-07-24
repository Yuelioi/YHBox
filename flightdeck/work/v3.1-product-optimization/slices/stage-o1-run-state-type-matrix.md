# O1 — Run 状态类型矩阵与真实旅程

## Goal

让用户无需逐项手测：自动证明每一种公开 Run 状态都能添加、编辑初值、保存重开、编译，并通过对应
状态节点进入统一 runtime。

## Status

Finished

## Result

- 组件红灯精确复现并修复“文件元数据点击 + 不添加”：复合初值被 Vue 转为 reactive proxy 后传给
  `structuredClone`，抛出 `DataCloneError`，命令未发出；现在 clone 前显式解包 reactive value。
- 完整 UI 类型矩阵覆盖 18 个 Catalog 命名类型和按键组合列表，添加与已有复合初值更新均通过。
- Authoring Projection 新增后端验证的 `stateInitial`；Data Type examples 在封装时必须通过 pinned schema，
  前端不再按 control 猜测初值。
- runtime 矩阵捕获并修正 `list<KeyCode>` 与 State Read/Write 泛型的合同冲突；状态声明负责 durable，
  Read/Write 绑定槽的冻结具体类型，Increment 仍只接受 numeric。
- service/store 矩阵覆盖全部类型的 authoring patch、保存重开与 compiler；noderuntime 矩阵逐类型
  执行 State Write 并核对输出。
- JSON 初值编辑不再静默吞掉非法输入，会保留草稿并显示中文行内错误。

## Verification

- 前端定向矩阵通过：4 个测试文件、62 项测试。
- Go 定向矩阵通过：`internal/datatype`、`internal/nodeauthoring`、`internal/nodes`、
  `internal/noderuntime`、`internal/services/workflow`。
- `task webview:smoke` 通过；真实 Windows WebView 自动选择“文件元数据”、添加 `smoke_metadata`
  并确认状态计数，证据位于 `.task/workflow-editor-smoke/20260724-215401/run-state.png`。
- 增量 `task check` 通过：42 个相关 Go 包、79 个前端测试文件/339 项测试；ESLint 基线保持 23。
- `pnpm -C frontend build` 和 bundle budget 通过：entry 247071/350000、editor
  205314/220000 gzip bytes。
