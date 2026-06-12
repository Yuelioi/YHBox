# Cockpit — YHFish

**Last updated**: 2026-06-12 by 月离 (子图库美化 v3 代码全绿: hover 多选框+分页(条数记忆)+底部双态工具栏+批量改分类+右栏就地编辑, 批量面板/编辑弹窗连根删。plan 已归档, 待真机。)
**Active focus**: **四案代码全绿待真机验收**: 编辑器 UX 收口 + 子图库三连改(v2 交互/微修/v3 美化, 批量与编辑以 v3 清单为准)。spec 都待真机过后拍 done。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-12-editor-ux-consolidation.md](specs/2026-06-12-editor-ux-consolidation.md) — 编辑器 UX 收口 — 删主界面库 tab(能力全收进编辑器子图库面板, 本地/在线两 tab + 全套增删改查) + 删"新窗口打开"模式(只维护主窗口一条路径) + 面包屑节点计数删除 + 主窗口补未保存标记
- [2026-06-12-library-modal-interaction.md](specs/2026-06-12-library-modal-interaction.md) — 子图库 modal 交互改版 — 单击选中出详情(删悬停)+双击/按钮插入 + Ctrl/Shift 多选 + 批量删除/加标签 + Subgraph 新增 category 字段(分组按分类) + 分类/标签过滤器
- [2026-06-12-library-modal-polish.md](specs/2026-06-12-library-modal-polish.md) — hover 多选框 + 底部工具栏(双态: 计数/分页 ↔ 批量删/加标签/改分类) + 分页(每页条数记忆) + 右栏就地编辑(名称/描述双击, 标签/分类行内控件, 删编辑弹窗) + 插入引用单主 CTA + 删除/复制弱化 + LibraryBatchPanel 连根删
- [2026-06-12-library-selection-and-fold-naming.md](specs/2026-06-12-library-selection-and-fold-naming.md) — 选中行 primary 高亮(bg-primary/15+ring) + 折叠/裸建子图默认名改「子图 N」序号递增(弃时间戳, 删折叠前缀键) + ConfirmDialog 输入模式打开时全选默认值
<!-- /AUTO -->

## 下一步

真机验收四案一次过(清单 = 待验证四条; 子图库批量/编辑形态以 v3 美化清单为准, 旧清单已标注取代项), 过了把四个 spec 拍 done; 顺手扫一眼 待复核 的 variable-system(大概率误报, 确认即销)。其余候选(无紧迫): 修 2a0ff140 测试容器的预存悬空引用 sg-0d53b1bb(删那个节点即可, 顺手活); WaitTemplate 孤儿边原子性硬化(真机再现再修); 复发#5 promotion 候选(前台容器全局指针 onMounted+onActivated 升 checklist); 脚本 SubgraphID 容错(未拍板, validator 全局校验已覆盖大半); 搜索/大复合 modal 收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 18 错; residue 28 处; runtime fixture 缺失红(build.md 在册)。

## 待复核

- ⚠ [docs/variable-system.md](docs/variable-system.md) — 改动路径命中 applies_to(internal/services/container/model.go), 自动拍 stale。实际改动只是 Subgraph 加 category 字段, 没碰变量系统 — 大概率误报, 扫一眼确认即拍回 active。

## 待验证

- ⚠ [archive/plans/2026-06-12-library-modal-polish.md](archive/plans/2026-06-12-library-modal-polish.md) — 子图库美化 v3 真机验收: ①行首 hover 出勾选框, 勾选常显, 批量不用按 Ctrl; ②底部工具栏双态: 无选中显「共 N 个」+分页, 选中显 批量删除/加标签/改分类/取消; ③每页条数 20/50/100 可调且重开记住, 过滤变化回第 1 页, >1 页出页码; ④批量改分类生效且分组随之变; ⑤右栏名称/描述双击可改(回车/失焦存, Esc 撤), 标签/分类行内直接改, 改完即存; ⑥「编辑信息」按钮/弹窗/右键项全无; ⑦插入引用是右栏唯一大绿钮, 复制/删除缩到底部小钮; ⑧双击行插入+右键其余菜单+过滤器+缺变量自动补全不回归。
- ⚠ [archive/plans/2026-06-12-library-selection-and-fold-naming.md](archive/plans/2026-06-12-library-selection-and-fold-naming.md) — 选中态+命名真机验收: ①子图库选中行 primary 高亮, 与悬停灰阶一眼可分; ②折叠弹框默认名「子图 N」且全选, 直接打字即替换, 回车接受, 连续折叠序号递增不重名; ③裸拖 Subgraph 节点自动名同为「子图 N」; ④其它带输入的确认框打开同样聚焦全选不炸。
- ⚠ [archive/plans/2026-06-12-library-modal-interaction.md](archive/plans/2026-06-12-library-modal-interaction.md) — 子图库 modal 真机验收(原批量面板/编辑按钮形态已被 v3 美化取代, 以上条清单为准): ①单击=选中出详情不插入, 悬停不再变右栏; ②双击插入引用+缺变量自动补不回归; ⑦分组按分类(空=未分类), 分类下拉+标签多选+文本搜索三过滤叠加; ⑧编辑器内子图属性面板能改分类, 与库内互通; ⑨右键菜单(插入/复制为新/复制ID/删除)与在线 tab 占位照旧。
- ⚠ [archive/plans/2026-06-12-editor-ux-consolidation.md](archive/plans/2026-06-12-editor-ux-consolidation.md) — 编辑器 UX 收口真机验收(③④的单击插入/悬停详情交互已被上条改版取代, 只验 CRUD/警告/tab 仍在): ①侧边栏无「库」, 容器/日程/设置导航正常; ②编辑器子图库 modal 有 本地/在线 两 tab(在线是占位文案); ③右键有 插入引用/复制为新/编辑信息/复制 ID/删除, 删被引用的子图弹「被 N 个容器使用」警告; ⑤面包屑无「N 节点」计数, 改图后容器名旁出「未保存」, 保存后消失, 离开编辑器有未保存时仍弹确认; ⑥容器列表与编辑器内无任何「新窗口打开」入口; ⑦录屏/截图/校准/悬浮启动器等工具窗照常能开(回归面)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
