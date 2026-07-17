# 旧版能力与 3.1 产品连续性审计

## 修正后的结论

3.1 重建了 Workflow Source、Authoring Projection、compiler/scheduler、内容寻址 Node/Data 契约、exact installation/capability admission、durable run journal 和同一调度器上的调试控制。这些是必要且有价值的底层升级。

但把当前状态称为“major upgrade 已完成”不准确。旧产品栈在 9fce7870 中整体移除后，许多用户能力没有在新架构上迁完；此前审计又过度聚焦节点编辑器。准确说法是：**底层架构升级已形成，新产品迁移尚未完成，当前 3.1 对用户存在明显能力回归。**

旧代码虽然从当前树删除，但完整保留在 Git 历史。迁移审计现固定以 9fce7870^ 为旧基线，逐项比较路由、视图、store、Wails 调用和 runtime；详细表见 [artifacts/legacy-product-capability-diff.md](artifacts/legacy-product-capability-diff.md)。

不回滚旧 Container 栈，不复制第二套编辑器或运行时；需要保留的能力必须接入 3.1 唯一契约与执行路径。

## 审计口径

一项旧能力只有同时满足以下四层，才能称为“已恢复”：

1. 可见入口：用户能发现并进入，空状态、取消和失败可恢复。
2. 管理流程：能创建、查看、编辑、删除、批量处理和处理规模增长。
3. 创作绑定：Workflow Source/编辑器能引用 exact installation、BlobRef 或目标 slot。
4. 运行闭环：Catalog、compiler、admission、provider/runtime 和 journal 使用同一契约。

后端函数、store、路由、build-tag 文件或 controller 单独存在，只能说明“底层仍在”，不能说明产品能力已保留。

## 平台架构结论

当前平台中立程度不一致：

- 正确 seam：pkg/input、pkg/capture、tools hotkey 的 build-tagged Adapter，以及 internal/automation/controller 的 Controller/能力 Interface。
- 错误 seam：生产使用的 internal/automation/installed 从 ProfileDraft 到 provider 都是 Win32 窗口模型；Settings、节点 targetKinds、appbootstrap policy 与前端又直接依赖 win32Targets/win32-window。
- 结果：未来直接增加 macOS 会跨层重写，而不是只新增一个 Adapter。

因此 Slice 13 必须先把 installed automation 深化成按 target kind 注册 Adapter 的平台中立模块，再恢复 Android；未来 macOS 只新增 bundle/window identity、input/capture Adapter 和 profile editor，不修改 Workflow Source、通用节点、compiler、scheduler 或 policy。

## 决策矩阵

| 能力 | 决策 | 3.1 适配 | 优先级 |
| --- | --- | --- | --- |
| 点击/连线不误移节点 | 已恢复 | 稳定手势投影、原子命令 | 完成 |
| 拖线推荐、hover、自动连接 | 已恢复 | 共用权威兼容规则 | 完成 |
| 多选/Delete/clipboard/对齐/布局 | 已恢复 | 实测尺寸、ELK、单次 undo | 完成 |
| 诊断、运行轨迹、真调试 | 已恢复 | 同一 compiler/scheduler/journal | 完成 |
| WaitTemplate/WaitTemplateGone/ClickTemplate | 已恢复 | exact target + BlobRef | 完成 |
| 资源缩略图、失效诊断、录制转草稿 | 已恢复 | 受限 Blob API、预览后单次 undo | 完成 |
| 平台中立 automation installation | 恢复重构 | target-kind Adapter registry + typed profile | P0 |
| Android/ADB 目标 | 恢复重做 | 通过通用 installation seam 接入 | P0 |
| 桌面应用取消与 F9 | 修复/恢复 | 取消 no-op；捕获回填 exact identity | P0 |
| 工作流删除/批量/搜索/分页 | 恢复 | service/store 查询与引用语义 | P0/P1 |
| 工作流 import/export | 3.1 发布前恢复 | canonical Source + exact blobs；不恢复旧 Container zip | P0 / Slice 16 |
| 资产批量/分页/维护 | 3.1 发布前恢复 | QueryAssets + 批量 metadata；GC 必须先有完整 BlobRef roots | P1 / Slice 17 |
| AI API URL | 恢复为安装属性 | endpoint 属于可信 AI installation | P0 |
| 悬浮窗启动入口 | 恢复入口 | 主壳调用既有 OpenLauncher | P0 |
| 画布节点定位/命令面板 | 定位已恢复；palette 由现有入口替代 | Source-native 搜索/聚焦 | 完成 |
| subgraph/comment/reroute | 3.1 发布前恢复/重做 | 完整调用契约/annotation/edge presentation | P1 / Slice 18 |
| JS/yt 任意脚本入口 | 明确不恢复 | typed nodes/Node Package；未来仅 sandboxed Script Node | 不恢复 |
| Browser CDP | 3.1 发布前恢复 | exact installation Adapter；controller 不等于 product support | P1 / Slice 19 |
| 旧 Container UI/第二运行时 | 明确不恢复 | 保持 3.1 唯一路径 | 不恢复 |

## 2026-07-17 发布边界纠偏

3.1 尚未发布，不能把已纳入升级范围的缺失能力改称 post-3.1 后宣称完成。Slice 15 已重新打开；Source portability、资产规模化、subgraph/comment 与 Browser CDP 分别进入 Slices 16–19。只有旧双运行时、任意宿主脚本入口等明确不符合 3.1 安全架构的能力继续不恢复。

## Stage 7 事实修正

- 旧树只有 ExportPackage，没有 ImportPackage 产品闭环；因此 3.1 不接收旧 Container zip，也不把“导入缺失”误写成已丢失的旧能力。
- 旧 Asset Maintenance 调用 previewCleanup/cleanupUnused 清理 subgraph，并非可证明安全的 Blob GC；3.1 metadata 删除不会破坏已绑定的 immutable BlobRef。
- 当前 Source 虽包含 graphs/GraphKindSubgraph 字段，compiler/program 仍要求唯一 main entry graph；subgraph 不能因 schema 字段存在就宣称可用。
- Browser CDP controller、discovery 与 client 只是底层构件；缺 installation/Settings/policy/provider/admission/Catalog 时不属于产品能力。
- 画布节点定位已恢复；通用 command palette 没有独立 domain value，由可见工具栏、选择工具条、目录搜索与快捷键替代。

## 分阶段路线

- Stage 1 图编辑：完成并批量验收。
- Stage 2 运行认知与调试：完成并批量验收。
- Stage 3 自动化创作闭环：完成并批量验收。
- Stage 4 基础产品可用性与桌面连续性：Slice 14 与 Slices 9–12，阶段末批量验收。
- Stage 5 平台目标架构与 Android：先 Slice 13，再 Slice 8；用 conformance、跨平台 compile、ADB emulator 与 Windows GUI 批量验收。
- Stage 7 高级能力决策与迁移收口：完成并批量验收。
- macOS runtime：不在当前 Slice 实现；Slice 13 后应成为新增 Adapter，而非再次改造核心。

## 跨阶段约束

- 产品版本只进入 version/manifest/binary metadata，不创建 nodes31 一类包名。
- 所有批量编辑必须是一个原子 undo/redo。
- 调试不得绕过 compiler、target、capability、租约或副作用审计。
- 工作流只引用安装 slot；path、PID、HWND、CGWindowID、API key 和 endpoint 不进入图。
- 原地动作成功不 toast；失败才使用 Nuxt UI，禁止浏览器 alert。
- Slice 内只做继续开发所需的定向检查；阶段末统一批量验收。
