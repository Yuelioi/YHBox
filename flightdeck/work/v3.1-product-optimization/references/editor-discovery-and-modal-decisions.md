# 编辑器发现与 Modal 取舍

## Research Read

问题是如何在节点数量增长、画布保持紧凑且键盘优先的桌面自动化编辑器中，提高节点/Snippet 发现效率，同时让工作流资源编辑、离开恢复和计划创建保持上下文。约束是继续使用 3.1 Catalog、Source、Asset、Snippet 与 Schedule 单一 owner，不复制 3.0 runtime/store。

## Source Matrix

- 本地 3.1：WorkflowNode 同时展示全部 pins，并把最多三个 lightweight inline candidate 放入节点；点击模板的多个 common duration 因此膨胀。WorkflowResourceDock 可创建和使用 Macro，但没有编辑事件。onBeforeRouteLeave 使用二选一 Confirm。SchedulesView 用 editing 状态替换整个列表。
- 本地 3.0 reference：NodeLibraryPanel 是 Houdini 式分类树，Tab 切换并聚焦；ContainerEditorView 记录最后鼠标位置，Snippet 有 shortcut 并在编辑器上下文插入。可复用交互成立，但旧 registry、Container 与 localStorage store 不恢复。
- [Houdini Tab menu](https://www.sidefx.com/docs/houdini/basics/tabmenu.html)：Tab 在当前 network context 打开可分类、可键入过滤、可方向键/Enter 操作的节点菜单。
- [Houdini Network editor](https://www.sidefx.com/docs/houdini/network/nodes.html)：选中节点类型后在 network 中放置；从 connector 发起时可按上下文缩小候选。
- [Houdini Network editor menus](https://www.sidefx.com/docs/houdini/network/menus.html)：参数编辑器与画布是独立 pane；节点可使用不同 LOD，只显示连接或必要端口。
- [Blender Menu Search](https://docs.blender.org/manual/en/3.4/interface/controls/templates/operator_search.html)：F3 搜索强调键入过滤、上下键与 Enter；菜单路径帮助用户理解分类。
- [Blender Pie Menus](https://docs.blender.org/manual/en/4.2/interface/controls/buttons/menus.html#pie-menus)：径向菜单依赖固定方向和肌肉记忆，适合少量熟练动作。
- [VS Code Snippets](https://code.visualstudio.com/docs/editing/userdefinedsnippets)：Snippet 可通过 prefix/Tab 或显式 keybinding 插入，并用上下文条件限制生效范围。
- Nuxt UI：现有 BaseModal、AdaptiveSelect、UButton/UInput 和语义 token 已提供 Modal、表单与键盘焦点基础，不引入另一套视觉系统。

## Patterns

1. Tab 快速添加只读取当前 Catalog/Graph context；分类是空查询的导航，搜索是已知目标的捷径。
2. 快速添加与左侧目录互补：前者键盘/鼠标位置优先，后者持续浏览、拖拽和教学。
3. 画布节点使用低 LOD：signals、required/primary/connected pins 常显，其余 pins 可展开；完整值编辑在 Inspector。
4. Snippet shortcut 是显式、可见、可冲突检查的编辑器上下文命令；不注册为 OS 级全局修改操作。
5. 离开脏文档必须同时提供恢复、破坏和取消路径；保存失败不得继续导航。
6. 创建/编辑单个管理对象且无需深链时使用 Modal 保留列表上下文；长生命周期或多页面任务才使用路由。
7. 饼菜单不承载节点全集或 Snippet 集合；它只适合未来少量固定画布命令。

## Local Application

- 不改变现有 emerald/zinc dark tokens，不采用外部工具建议的紫色主题。
- 3.1 Catalog、SnippetService、MacroService、ScheduleStore 和 EditorSession 继续是唯一事实源。
- Tab 面板复用 Catalog Projection 与 Snippet summaries；不保存第二份分类索引。
- optional pin 展开是纯 UI 状态；Source、bindings 和 Compiler 完全不变。
- 计划 Modal 复用 ScheduleEditorPanel；不新增 route 或 draft store。

## Next Step

按 Slice 14 → 15 → 16 → 17 实施，Stage I 末用点击模板、已有 Macro、自定义 Snippet 快捷键和计划 Modal 组成真实 WebView 用户旅程，再统一跑完整门禁。
