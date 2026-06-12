---
status: done
summary: 编辑器 UX 收口执行计划 — 7 任务: 删新窗口模式(Go→前端→分支清扫) → 未保存标记 → 删节点计数 → 子图库面板升级(tabs+CRUD) → 删库 tab → 全套验证
implements: specs/2026-06-12-editor-ux-consolidation.md
last_updated: 2026-06-12
verify: 编辑器 UX 收口真机验收 — ①侧边栏无「库」, 容器/日程/设置导航正常 ②编辑器子图库 modal 有 本地/在线 两 tab(在线是占位文案) ③本地 tab 单击仍是插入引用+缺变量自动补(不回归), 右键有 插入引用/复制为新/编辑信息/复制 ID/删除, 删被引用的子图弹「被 N 个容器使用」警告 ④悬停条目右栏出详情 ⑤面包屑无「N 节点」计数, 改图后容器名旁出「未保存」, 保存后消失, 离开编辑器有未保存时仍弹确认 ⑥容器列表与编辑器内无任何「新窗口打开」入口 ⑦录屏/截图/校准/悬浮启动器等工具窗照常能开。注: ③④的单击插入/悬停详情交互已被 2026-06-12-library-modal-interaction 后续改版取代, 以新清单为准, 此条只验 CRUD/警告/tab 仍在
---

# 编辑器 UX 收口 — 执行计划

依据: [spec](../specs/2026-06-12-editor-ux-consolidation.md)。全程前置读 checklists: code-style / ui / comments / build / vue-i18n-message-compiler-traps; incidents: draft-subgraphs-phantom-field (子图内容读取走 subgraphsFor, 别读 draft.subgraphs), keepalive-singleton-subgraph-store-stale (动 editorStore 全局指针前必读)。

## 任务

1. **删「新窗口打开」Go 侧**: service.go 删 OpenEditorWindow/OpenInWindow/WindowOpener 接口/windows 字段/SetWindowOpener; wire_container_editor.go 整文件删; main.go SetWindowOpener 调用行删。验: `go build ./...` 绿。
2. **regen bindings + 删前端 RPC wrapper**: backend.ts 删 containers.openEditorWindow/openInWindow; bindings 按 build.md 走 task 链路 regenerate (改了 Go 导出符号)。验: 前端无对两 RPC 的引用 (git grep)。
3. **删前端入口 + isStandalone 分支 + 子窗专属 UI**: ContainersTab 卡片按钮+onEditInWindow; ContainerEditorToolbar 菜单项+isStandalone prop; ContainerEditorView standalone 顶栏(6-55)/dirty 关闭 modal(57-91)/goBack/onClose/onMinimise/onToggleMaximise/onOpenNewWindow/isStandalone computed/useWindowControls 消费; App.vue query.standalone 支删 (meta.standalone 留给工具窗)。i18n: editor.window.* / editor.dirty.* / editor.header.back / editor.toolbar.open_new_window / containers.open_new_window_tip 清 (zh+en)。验: git grep 'standalone=1' 零命中; vue-tsc 绿。
4. **主窗口未保存标记**: ContainerEditorBreadcrumb 容器名旁渲染 dirty 小标记 (复用 editor.header.dirty_dot), dirty 从 ContainerEditorView 传入。验: 视觉自检 (headless-ui-verify 或 task dev)。
5. **删面包屑节点计数**: ContainerEditorBreadcrumb.vue:30 + editor.breadcrumb.node_count (zh/en)。验: vue-tsc 绿。
6. **子图库面板升级** (LibraryExplorerModal): ①顶部 本地/在线 tab, 在线占位文案; ②本地 tab 右键菜单 插入引用/复制为新/改名…(名+描述+标签 modal, subgraphs.update 带 baseRev)/删除(referrersOf 预扫, 被引用弹确认)/复制 ID; ③选中详情区 (描述/标签/节点数/引用计数异步)。复用 stores/library.ts。验: vue-tsc + 前端测试绿; 视觉自检。
7. **删库 tab + 全套验证**: 删 /library 路由 + AppSidebar 入口 + LibraryView/LibraryCard/LibraryDetailPanel/ComingSoonSection + 孤儿 i18n (library.* 按消费者清, sidebar.library); 全仓 grep 断链 (move-delete-update-referrers)。验: `go build ./...` + `go test ./...` + vue-tsc + `pnpm test` + lint, 按 cockpit 在册预存红判基线; 真机 smoke 留用户。

## Progress

- [x] 1-7 全部完成 (2026-06-12)。验证: go build/test (9 红全为 build.md 在册 fixture 缺失预存) · vue-tsc 绿 · vitest 274/274 · i18n parity 1929 OK + compile OK + residue 39 预存 · oxlint 18 错全预存 (在册基线 16 系数字漂移, 已实证 HEAD 同款) · vite build 绿。真机验收留用户。
- 实现期修正: dirty 关闭确认 modal 非子窗专属 (路由守卫共用), 保留; 顺手清死代码 NodePalette / useFlowInteraction / TabToolbar / useLibraryViewMode + 编辑占用锁机制 (详见 spec ③)。
