# Tab menu and snippet shortcuts

## Outcome / Question

恢复面向大型节点目录的上下文快速添加：用户在画布按 Tab 即可按分类或搜索节点/Snippet，并在当前鼠标位置插入；常用 Snippet 可绑定安全快捷键。

## Completion criterion

- 画布非输入态按 Tab 打开快速添加面板并聚焦搜索；Escape 关闭，上下键移动，Enter 插入。
- 空查询展示类别 → 节点的两级结构；输入查询跨本地化标题、描述、tags、type ID 和 Snippet 元数据匹配。
- 选择节点或 Snippet 后插入到唤起时鼠标对应的 flow coordinate；子图上下文继续排除 run-root。
- Snippet schema、summary、编辑 Modal 和列表恢复可选 shortcut；服务拒绝保留快捷键与重复绑定。
- Snippet 快捷键只在工作流编辑器画布上下文触发，输入框、Modal 与系统/编辑器保留组合不触发。
- 左侧目录继续保留浏览/拖拽能力；快速添加不是第二份 Catalog owner。

## Blocked by

- Slice 12 durable Snippet 与现有 Catalog Projection。
- Slice 15 确保工作流内资源/离开流稳定。

## Verification

- Go 测试覆盖 shortcut trim、长度、保留组合和重复冲突。
- 前端测试覆盖快捷键标准化、输入态守门、Tab 分类/搜索和鼠标落点。
- WebView 用 Tab 搜索添加点击模板，再用自定义快捷键插入已配置 Snippet。

## Out of scope

- 饼菜单：可增长节点全集不适合径向固定槽位。
- 恢复 3.0 registry、Container 或 localStorage Snippet store。
- 全局 OS 快捷键直接修改工作流；快捷键限定当前编辑器上下文。

## Result

已完成。工作流画布记录最近鼠标位置，非输入/非 Modal 状态按 Tab 打开分类与跨节点/Snippet 搜索面板；上下键、Enter、Esc 完成键盘流，并在唤起坐标插入。Snippet schema/summary/Modal/Dock 增加可选快捷键，Go 服务规范化组合并拒绝编辑器保留项、无修饰普通键和重复绑定；快捷键仅在画布上下文触发。Go 服务测试、前端快捷键/快速添加/创作基础测试、TypeScript 类型检查与 i18n 检查通过。
