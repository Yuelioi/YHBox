---
status: idea
summary: 杂项小工具 backlog — i18n residue 清理 (悬浮窗工具→转正 floating-launcher; 截图 UI 美化已做; 从 scratch-backlog 抢救保留)
---

# 杂项小工具 backlog

2026-06-06 大清理时，从已归档的 [scratch-backlog](../archive/specs/scratch-backlog.md) 里只保留下面两条；其余编辑器/输入/审计类零散项一并随各 backlog 归档。

**触发**：随手记，没承诺立刻干。

---

- ~~**悬浮窗工具**：支持置顶，后续往里塞小工具。~~ → **2026-06-07 已落地归档** [2026-06-06-floating-launcher](../archive/specs/2026-06-06-floating-launcher.md)（最终入口在主程序 titlebar、配置进独立设置页、自适应高度+显示模式；偏离见该 spec 文末「归档落地说明」）。
- ~~**截图 UI 美化**：参考录屏的样式。~~ ✅ **2026-06-06 已做**：截图窗外壳对齐录屏家族 HUD + 内部重做（放大镜取点 / 实时取色 / 可编辑数值 / 滚轮缩放·右键平移 4 功能 + 现代化卡片侧栏）。连带把全项目 HSV 约定 S·V 0-255→0-100（H 0-360 不动）切齐主流取色器，取色读数能直接填检测节点阈值。
- **i18n residue 清理**：`useAutoFocusOnOpen.ts:39` 一个 DEV-only 中文 `console.warn`（parity 绿、residue 红）。要清再说。
- **`SettingsLauncher.vue` 正文 i18n**：悬浮窗启动器设置页正文仍中文硬编码，`pnpm i18n:check` 持续标红（非回归，floating-launcher 归档时延后项）。要清再说。
- **4 个独立工具窗换 `HudShell` 统一风格**：录屏/截图/鼠标检测/校准，详见 checklist [standalone-window-style](../checklists/standalone-window-style.md)。floating-launcher 归档时的延后项。
- **WindowTarget TitleMatch 下拉名不副实 (latent bug)**：`window_target.go` 下拉列了 `exact/contains/prefix/suffix/regex` 五项，但 `pkg/winutil/window.go::CompileTitle` 只在 `regex` 时编译正则，其余全落到 `ResolveWindow` 的精确比对 (`title != spec.Title`)——即 `contains/prefix/suffix` 实际全按 `exact` 匹配，用户选了无效。`MatchSpec` 注释本身也只写 `"exact" | "regex"`。修法二选一：要么 `CompileTitle` 把 contains/prefix/suffix 转成对应正则，要么砍掉下拉那三项。（2026-06-07 加 WaitWindow 节点时发现；WaitWindow 已只暴露诚实的 exact/regex。无 user demand，先记。）
