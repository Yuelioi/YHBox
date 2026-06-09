# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (子图 `__missing__` 反复 bug **根治(第4次复发, 真机验过)**: ① 防火墙 onConnect 拦哨兵 pin、永不进存盘边 ② onActivated 切回容器重拉子图。教训进 [incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md)。｜前: modal+HUD 风格统一(小 HUD 彩色面板 / modal BaseModal 纯黑平铺)。)
**Active focus**: 无专项进行中。近期完成: **子图 `__missing__` 根治(真机验)**、资产子系统(→ [docs/asset-subsystem.md](docs/asset-subsystem.md))、modal+HUD 风格统一。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。候选见下。

## 进行中

- 无专项进行中。(modal+HUD 风格统一已完成, 见 Active focus + [ui.md](checklists/ui.md) / [standalone-window-style.md](checklists/standalone-window-style.md)。)

## 下一步

候选(用户拍, 无紧迫): ① 搜索面板(CommandPalette·NodeSearch)/ 大复合(TemplatePicker·TemplateManager)modal 是否收进 BaseModal(目前有意没收: 结构特殊) ② idea 池(cv-perception · editor-footgun · misc-tools)。注: **截屏 ScreenPicker / 鼠标检测 MouseHUDView 是大面板, 不套小 HUD 彩色状态方案**(用户 2026-06-10 确认)。已知预存失败(非回归): runtime 缺 fish fixture, 见 [build.md](checklists/build.md)。

## Hanging tasks

- [ ] 无阻塞待办。（子图 "(子图未找到)/__missing__" 反复 bug 已根治+真机验，全程入 [keepalive incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md) + [import-cache incident](incidents/2026-06-09-import-bypasses-container-store-cache.md)。原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。）
