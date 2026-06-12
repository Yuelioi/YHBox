# Cockpit — YHFish

**Last updated**: 2026-06-12 by 月离 (编辑器 UX 收口代码全绿: 库 tab 整删+能力收进编辑器子图库 modal(本地/在线 tab + 全套增删改查) + 新窗口模式连根删 + 未保存标记入面包屑 + 顺手清四件死代码。待真机验收。)
**Active focus**: **编辑器 UX 收口代码+验证全绿, 待真机验收**(清单见 待验证)。plan 已归档([spec](specs/2026-06-12-editor-ux-consolidation.md) 待真机过后拍 done)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-12-editor-ux-consolidation.md](specs/2026-06-12-editor-ux-consolidation.md) — 编辑器 UX 收口 — 删主界面库 tab(能力全收进编辑器子图库面板, 本地/在线两 tab + 全套增删改查) + 删"新窗口打开"模式(只维护主窗口一条路径) + 面包屑节点计数删除 + 主窗口补未保存标记
<!-- /AUTO -->

## 下一步

真机验收编辑器 UX 收口(清单 = 待验证第一条), 过了 spec 拍 done。其余候选(无紧迫): 修 2a0ff140 测试容器的预存悬空引用 sg-0d53b1bb(删那个节点即可, 顺手活); WaitTemplate 孤儿边原子性硬化(真机再现再修); 复发#5 promotion 候选(前台容器全局指针 onMounted+onActivated 升 checklist); 脚本 SubgraphID 容错(未拍板, validator 全局校验已覆盖大半); 搜索/大复合 modal 收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 18 错; residue 28 处; runtime fixture 缺失红(build.md 在册)。

## 待复核

- 无。(vars.\* 删漏 2026-06-12 用户拍板补删, 已销。)

## 待验证

- ⚠ [archive/plans/2026-06-12-editor-ux-consolidation.md](archive/plans/2026-06-12-editor-ux-consolidation.md) — 编辑器 UX 收口真机验收: ①侧边栏无「库」, 容器/日程/设置导航正常; ②编辑器子图库 modal 有 本地/在线 两 tab(在线是占位文案); ③本地 tab: 单击仍是插入引用+缺变量自动补(不回归), 右键有 插入引用/复制为新/编辑信息(名/描述/标签)/复制 ID/删除, 删被引用的子图弹「被 N 个容器使用」警告; ④悬停条目右栏出详情(描述/标签/节点数/引用计数); ⑤面包屑无「N 节点」计数, 改图后容器名旁出「未保存」, 保存后消失, 离开编辑器有未保存时仍弹确认; ⑥容器列表与编辑器内无任何「新窗口打开」入口; ⑦录屏/截图/校准/悬浮启动器等工具窗照常能开(回归面)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
