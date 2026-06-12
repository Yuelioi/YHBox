# Cockpit — YHFish

**Last updated**: 2026-06-12 by 月离 (编辑器外壳重排真机过 + 用户拍板微调落地: 录制从 rail 底部移回 toolbar 右区最左(检查左边, 紧凑三态单控件); rail 定稿 4 项(变量/Snippets drawer + 节点库/子图库 modal)。跨容器串修复真机过。)
**Active focus**: **编辑器外壳重排已真机过**(rail 4 项 + 录制在 toolbar 右区 + 面包屑去冗余) + 跨容器状态串根治已真机过。子图转脚本真机过。剩录制新位置看一眼 + 删被引用模板真机债。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->

<!-- /AUTO -->

## 下一步

清真机验证债(见 待验证: 录制新位置扫一眼; 库里删被脚本引用的模板 → 弹 referrer 警告)。外壳重排/跨容器修复/子图转脚本真机均已过, 销。之后无在飞项目, 候选(无紧迫): WaitTemplate「先连边再建节点」失败留孤儿边的原子性硬化(本次根因修了暂不触发, 真机再现再修); 复发#5 promotion 候选(前台容器全局指针 onMounted+onActivated 规则升 checklist); 脚本 SubgraphID 容错(运行时报错附现有子图列表 / 编辑期校验字面 SubgraphID 存在性, 2026-06-12 提出未拍板); 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错清债; residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册); StateCycleSmoke 本机预存红排查(build.md 在册, mock 模板没命中)。

## 待复核

- 无。(vars.\* 删漏 2026-06-12 用户拍板补删, 已销。)

## 待验证

- ⚠ 录制新位置真机扫一眼(2026-06-12 用户拍板移位后未看): toolbar 右区最左(检查左边)「录制」按钮 — 空闲点开下拉精准/简易; 倒计时变「取消(n)」; 录制中变红「停止录制」(目标容器+F12 进 tooltip)。(外壳重排其余项 + 跨容器修复三步 用户 2026-06-12 真机已过, 销。)
- ⚠ [archive/specs/2026-06-11-script-template-dep-extraction.md](archive/specs/2026-06-11-script-template-dep-extraction.md) — 库里删一个被某脚本引用的模板,确认弹「被引用」referrer 警告 + gcBlobs 不回收其 blob(单测已覆盖提取+扫描器接线,差集成/真机这一验)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
