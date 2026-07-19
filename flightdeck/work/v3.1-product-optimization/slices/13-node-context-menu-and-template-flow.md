---
slice: "13"
title: 节点上下文菜单与模板创作流
status: completed
---

## Outcome / Question

把节点右键从直接执行“保存 Snippet”升级为专业编辑器的标准上下文菜单；视觉模板在当前工作流内完成选择或截图，并复用同一资产与 typed BlobRef 绑定链路，不再要求用户退出编辑器寻找入口。

## Completion criterion

- 右键普通工作流节点打开可键盘导航、可关闭且层级稳定的 Nuxt UI 上下文菜单，不再直接弹出 Snippet 保存框。
- 菜单至少提供复制、剪切、复制节点、启用/禁用、断点、折叠为子图、保存为 Snippet、视觉模板与删除；危险操作独立分组并使用错误语义色。
- 右键未选中节点时先把它设为当前节点；右键已处于多选中的节点保留多选，因此批量复制、剪切、复制和删除继续作用于现有选择。
- 视觉模板子菜单提供“从资源库选择”和“截图新模板”：前者在编辑器资源工作区直接打开视觉模板分页，后者在当前工作流发起截图。
- 截图完成后，兼容节点立即绑定模板 BlobRef；不兼容节点按既有 3.1 节点契约插入点击模板节点。保存、搜索、选择和运行仍走唯一 Asset/EditorSession 链路。
- Snippet 只是复用分组中的一个菜单功能，保存 payload、持久化和插入契约保持不变。

## Blocked by

- Slice 12 已提供 durable Snippet 服务和节点快照契约。
- 工作区资源与模板截图、Asset 查询、BlobRef binding 和 click-template 插入链路必须保持单一 owner，不创建上下文菜单专用副本。

## Verification

- 组件测试锁定 Nuxt UI 菜单、完整分组、危险色、模板资源直达和不再直接响应 contextmenu 保存 Snippet。
- 定向前端测试覆盖右键选择语义、模板页签外部控制与既有资源绑定。
- 真实 Wails/WebView 中右键节点，验证菜单和 Snippet 二级动作，再展开视觉模板子菜单并进入当前工作流的模板资源分页。
- Stage H 末统一运行聚合前端测试、完整门禁、真实 WebView smoke 和 production build。

## Out of scope

- 复制 3.0 Container、模板 store 或旧按 node kind 分发。
- 在本 Slice 引入完整画布空白处菜单、连线菜单或自由停靠面板。
- 让多节点 Snippet 混入单节点 Snippet payload；多节点复用继续使用 Source-native 子图。

## Result

已完成。普通工作流节点使用基于现有 Nuxt UI DropdownMenu 的鼠标位置锚点菜单：复制、剪切、复制节点、启用/禁用、断点、折叠为子图、视觉模板、保存为 Snippet 与删除按任务分组；快捷键提示、键盘焦点、portal 碰撞处理和错误色危险项沿用统一组件。右键未选中节点会建立单节点选择；右键现有多选中的节点保留整个选择，因此批量命令语义不变。

视觉模板作为菜单子任务进入工作流：选择动作把左侧工作区切到受控的 template 分页，保留搜索、分类、排序、分页和资源使用；截图动作复用现有目标选择与 Screen Picker。保存结果继续交给 `useWorkspaceResource`：有兼容 template BlobRef 输入时绑定当前节点，否则插入正式 click-template 节点，不复制资产 store、BlobRef 或节点创建状态。

初次采用 UContextMenu 使 editor gzip 达 221850B，超过 220000B 门禁 1850B；实现改为复用编辑器已加载的 UDropdownMenu 后降至 218398B，未放宽预算。真实 WebView journey 通过，截图位于 `.task/workflow-editor-smoke/20260720-025424/node-context-menu.png` 与 `node-template-menu.png`。最终 `task check:frontend` 通过：格式、oxlint、ESLint、typecheck、i18n、Wails contract、63 个测试文件 / 253 项测试、production build 和 bundle budget 全绿；`task webview:smoke` 通过。

完整 `task check` 第二轮已越过本次修改的 `cmd/workflow-editor-smoke`，仅 `pkg/platform TestHighResTimer_TickerAccuracy` 在 coverage 高负载下以 631ms 超过 600ms 上限；该仓库已知实时调度 flaky 随后的定向 `go test -count=1 ./pkg/platform -run TestHighResTimer_TickerAccuracy` 通过。
