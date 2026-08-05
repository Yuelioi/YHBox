# 知识审计与收口决定

## 审计范围

- `docs/`：33 个 Markdown，混有现行架构、3.1 ADR、研究稿和 Wayfinder ticket。
- `flightdeck/knowledge/`：91 个 Markdown，混有任务指南、架构说明、一次性故障复盘和已解决问题时间线。
- 引用者：根 `README.md`、`CONTRIBUTING.md`、`SECURITY.md`、`AGENTS.md`、`scripts/README.md`、
  `flightdeck/deck.md` 与 Finished Work。
- 代码事实：composition、Application/Workflow 运行链、storage root 与 SQLite schema、节点 Catalog、
  configured target、Wails transport、Vue Flow 编辑器边界、Task/CI 入口。

## 最小结构

`docs/` 保留面向开发者的当前事实：

- `docs/README.md`：总入口与知识职责边界；
- `docs/architecture/README.md`：系统图和关键代码地图；
- `docs/architecture/runtime.md`：composition、Source → Program → Run、节点与 Target 运行链；
- `docs/architecture/storage.md`：本地 profile、数据 authority、备份与凭据位置；
- `docs/architecture/threat-model.md`：当前信任与运行边界；
- `docs/compatibility.md`、`docs/platform-support.md`、`docs/open-source-readiness.md`：对外稳定合同。

`flightdeck/knowledge/` 保留可独立执行的任务指南：

- `README.md`：按任务路由；
- `build/build.md`：增量/full 门禁、生成物、build/package/smoke 触发条件；
- `frontend/ui.md`：当前 UI 约定与视觉验收；
- `frontend/workflow-editor.md`：EditorSession、Vue Flow authority 与画布手势；
- `nodes/development.md`：节点/类型/adapter 全链开发；
- `automation/input-and-capture.md`：Target、输入/捕获语义与真机验证；
- `wails/services.md`：Go service、composition、bindings、transport 与 error/event 契约。

## 删除与合并决定

- 删除 `docs/adr/`、`docs/research/`、`docs/wayfinder/`、`docs/agents/`；仍成立的当前原则写入架构或兼容文档，历史由 Git 与 Finished Work 保存。
- 删除拆分的 application/network/automation/node architecture 文档；把当前运行事实合并进
  `architecture/runtime.md`，代码地图合并进 `architecture/README.md`。
- 删除所有单次 bug 复盘、旧节点/Container/Subgraph 范式、旧工具/API 技巧与重复架构 Knowledge；
  仍成立的正向规则合并到上述六类任务指南。
- 产品版本维护合并进 build 指南；版本和合同的事实仍由 `VERSION`、module 常量、schema 与
  `task versions:*` / `task contracts:*` 负责。

## 代码核对摘要

- Windows 默认 profile 是 `%LOCALAPPDATA%\Yotta\Yotta`；显式 root、`YOTTA_ROOT`、平台默认按此优先级解析。
- root layout 为 3；Content Catalog 是 `catalog/content.db` schema 8，Run Ledger 是
  `state/runs.db` schema 2；Workflow Source 在 Catalog，Program 只在 `cache/programs`，Blob bytes
  在 `objects/sha256`。
- `main.go → internal/desktopapp → internal/localruntime → internal/appbootstrap → internal/application`
  是唯一 composition/command 链；GUI、CLI、Schedule、MCP/AI authoring 不创建第二套 runtime。
- Node/Data Contract 由 `internal/datatype`、`internal/nodecontract`、`internal/nodes` 显式组装，
  Catalog/Projection/Compiler/runtime adapter 分别由 `nodecatalog`、`nodeauthoring`、
  `workflow/compiler`、`noderuntime` 拥有。
- Vue Flow 的瞬时选择/拖拽位置属于内部 store；持久 Source 由 `EditorSession` 通过 revision-CAS command
  更新。当前空白左键拖拽框选，Space+左键或中/右键平移，Ctrl 追加选择。
- `ALT` 等 modifier-only key chord 是有效值；短按 chord 用 Press Keys，跨节点保持按下状态用
  Hold Keys → effect → Release Held Input，并由 Run owner 在取消/失败/结束时兜底释放。

