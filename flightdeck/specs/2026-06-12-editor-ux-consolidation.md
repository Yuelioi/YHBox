---
status: active
summary: 编辑器 UX 收口 — 删主界面库 tab(能力全收进编辑器子图库面板, 本地/在线两 tab + 全套增删改查) + 删"新窗口打开"模式(只维护主窗口一条路径) + 面包屑节点计数删除 + 主窗口补未保存标记
last_updated: 2026-06-12
---

# 编辑器 UX 收口: 删库 tab + 子图库面板全套 CRUD + 删新窗口模式

**拍板记录 (2026-06-12)**: 用户定 ①删主界面「库」tab — 绝大部分场景在容器编辑器, 库能力收进编辑器子图库面板, 面板改成**本地/在线**两 tab (在线占位, 为后续在线库留口); ②编辑器子图库本地 tab 补**全套增删改查** (插引用/复制为新/改名+改描述标签/删除带引用警告/复制 ID/详情); ③删面包屑「N 节点」计数 (什么都不点就能看概况, 冗余); ④**整删「新窗口打开」功能** — 不修两路径行为分叉 (tab 开节点浏览器不支持/未保存标记不一致), 直接只维护主窗口一条路径; ⑤主窗口编辑器**补未保存标记** (删新窗口后全 app 失去未保存提示, 在面包屑容器名旁补回)。

## ① 删「库」tab + 编辑器子图库面板升级

**删除对象** (整删, 二号铁律):

- 路由 `/library` (frontend/src/router/index.ts:11) + 侧边栏入口 (AppSidebar.vue:115)
- LibraryView.vue / library/LibraryCard.vue / library/LibraryDetailPanel.vue / library/ComingSoonSection.vue
- 相关 i18n key (library.* 中只被以上组件消费的)
- stores/library.ts **保留** (编辑器面板复用其 list/delete/duplicate/referrersOf/missingGlobalsFor)

**LibraryExplorerModal 升级** (frontend/src/components/containers/LibraryExplorerModal.vue):

- 顶部 **本地/在线** 两 tab。在线 = 占位 (从 ComingSoonSection 搬文案精神, 不搬组件)。
- 本地 tab 保留现有: 搜索 / 按主标签分组 / 单击插引用+缺变量自动补 (useNodeCreation.onPickLibrarySubgraph 不动)。
- 补右键菜单: 插入引用 / 复制为新子图 / 改名…(含描述/标签编辑) / 删除 (先 referrersOf 扫引用, 有引用弹「被 N 个容器使用」确认) / 复制 ID。
- 补详情: 选中项展示 描述/标签/节点数/引用计数 (异步拉 referrers, 参考原 LibraryDetailPanel 的交互, 形态融入面板不照搬组件)。
- 后端 RPC 全现成: subgraphs.update(id, patchJSON, baseRev) 改名改元数据 / delete_(id, baseRev) / duplicate(id) / referrers(id) (backend.ts:236-250)。**无后端改动**。

## ② 删面包屑节点计数

ContainerEditorBreadcrumb.vue:30 `editor.breadcrumb.node_count` 展示删除, i18n key (zh.ts:221 / en.ts:206) 一并清。

## ③ 删「新窗口打开」(调研清单 2026-06-12, 两路径合一)

**前端入口**: ContainersTab.vue:118-125 卡片按钮 + :284-291 onEditInWindow; ContainerEditorToolbar.vue:169 菜单项 + ContainerEditorView.vue:575-585 onOpenNewWindow。
**后端**: service.go OpenEditorWindow(74-84) / OpenInWindow(86-94) / WindowOpener 接口(25-29) / windows 字段(35) / SetWindowOpener(65-69); wire_container_editor.go **整文件删**; main.go:454 SetWindowOpener 调用行删。
**子窗专属 UI** (ContainerEditorView.vue): standalone 顶栏 (返回/容器名/未保存/window controls) + goBack/onClose/onMinimise/onToggleMaximise + useWindowControls 调用。**dirty 关闭确认 modal 保留** — 实现期源码核出它被嵌入态路由守卫 (onBeforeRouteLeave/Update → guardDirty) 共用, 调研报告误判为子窗专属; editor.dirty.* 文案随之保留。
**isStandalone 分支全清**: App.vue (保留 route.meta.standalone — 其他工具窗仍用, 只删 query.standalone 这一支); ContainerEditorView; ContainerEditorToolbar; 路由 query。
**i18n 删**: editor.header.back / editor.window.*(否——AppTitleBar 主窗在用, 留) / editor.toolbar.open_new_window / containers.open_new_window_tip / containers.editing_locked_tip / toast.open_window_failed。editor.header.dirty_dot 留给⑤复用。
**useWindowControls.ts**: AppTitleBar 主窗也在用 → **保留文件**, 只删编辑器侧消费。
**编辑占用锁连根删** (实现期扩展): editingContainerIDs/isEditing (stores/containers.ts, 订阅 wire_container_editor 的 editing-acquired/released 事件) + ContainersTab 卡片 disable/锁图标 — 事件源删除后永远 false, 整套清。
**顺手死代码清除** (实现期发现, 全是 /library 或旧拖放链的引用者): NodePalette.vue (编辑器 UX v2 后无人挂载) + useFlowInteraction.ts (legacy MIME 拖放链, 生产者只剩 NodePalette) + TabToolbar.vue (LibraryView 是最后消费者) + useLibraryViewMode.ts; LibraryDetailPanel 移居 components/containers/ (modal 复用)。
**不受影响**: 录屏/截图/校准/悬浮启动器等独立工具窗 (route.meta.standalone 体系, 不走 WindowOpener)。

## ⑤ 主窗口未保存标记

dirty 源: useContainerDraft.ts:56 (deep watch draft+activeGraph 置脏, 保存后清)。在 ContainerEditorBreadcrumb 容器名旁渲染小「未保存」标记 (复用 editor.header.dirty_dot 文案或等价新 key), dirty 经 ContainerEditorView 现有链路传入。

## 验收

1. 主界面侧边栏无「库」; 编辑器子图库面板有本地/在线两 tab, 本地 tab 右键能 复制为新/改名/删除(被引用时弹警告), 选中能看引用计数。
2. 面包屑无节点计数; 改动图内容后容器名旁出现未保存标记, 保存后消失。
3. 容器列表与编辑器内无任何「新窗口打开」入口; `go build ./...` + vue-tsc + 测试基线全绿 (预存红按 cockpit 在册)。
