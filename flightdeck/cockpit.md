# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (modal + HUD 风格统一完成 + 看板清理。定调: **小 HUD 面板=彩色状态面板**(参考鼠标录制)、**内容多的 modal=BaseModal 纯黑平铺**, 约定落 [ui.md](checklists/ui.md) / [standalone-window-style.md](checklists/standalone-window-style.md)。)
**Active focus**: 无专项进行中。近期完成: 资产子系统(→ [docs/asset-subsystem.md](docs/asset-subsystem.md))、modal+HUD 风格统一。**本仓内测期: 默认不 push, 用户明确说推才推**(commits.md 铁律)。下一步候选见下。

## 进行中

- 无专项进行中。(modal+HUD 风格统一已完成, 见 Active focus + [ui.md](checklists/ui.md) / [standalone-window-style.md](checklists/standalone-window-style.md)。)

## 下一步

候选(用户拍, 无紧迫): ① 其它 frameless HUD(截屏 ScreenPicker / 鼠标检测 MouseHUDView)是否也上**彩色状态面板**(跟校准/录制 HUD 一致) ② 搜索面板(CommandPalette·NodeSearch)/ 大复合(TemplatePicker·TemplateManager)modal 是否收进 BaseModal(目前有意没收: 结构特殊) ③ 子图切换 smoke 复验(Hanging) ④ idea 池(cv-perception · editor-footgun · misc-tools)。已知预存失败(非回归): runtime 缺 fish fixture, 见 [build.md](checklists/build.md)。

## Hanging tasks

- [ ] **子图系统问题（大部分已根治，剩真机 smoke 复验）**。修了三条真 bug，前两条同症状 "(子图未找到)" 不同根因：① 库导入绕过容器 Store 内存缓存 → [incident](incidents/2026-06-09-import-bypasses-container-store-cache.md)（library SetContainerReloader + 回归测试）；② keep-alive 多容器编辑器共享全局单例子图 store 切回污染 → [incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md)。② 已**根治**（commit 20e25a9）：store 状态按容器隔离（subgraphsByContainer / editorPathByContainer keyed by containerID），activeContainerID 降级成前台指针，对外 API 不变、单测 59 绿；顺带消除未落盘子图编辑/层级切换丢失 + id 碰撞 mergeSubgraphs 取错版本。先前的 onActivated 补丁(9ccccbf)已被根治替换。**待用户真机 smoke 复验**：容器2 折叠子图 → 切容器3 → 切回容器2，子图节点应仍正常 + 分享成功。（原记的 2 个预存 vue-tsc 红已在 2026-06-10 import-strategy 收口随资产子系统清零。）
- [ ] 无阻塞待办。（原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。）
