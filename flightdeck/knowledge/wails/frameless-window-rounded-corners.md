# Frameless 工具窗圆角与透明背景
## 当前结论（v3.0.0-alpha2.117）

- Windows 实现仍没有 `DWMWA_WINDOW_CORNER_PREFERENCE` / `DWMWCP`，`CornerRadius` 仍只用于 macOS Liquid Glass；Win11 原生 DWM 圆角若要用，仍需项目自己的 Windows adapter。
- 旧 alpha.91 会给透明/半透明窗口强制加 `WS_EX_TRANSPARENT`，导致 HUD 点击穿透。alpha2.117 已修正：只有显式设置 `IgnoreMouseEvents` 才加该 style；普通透明/半透明窗改走 DirectComposition (`WS_EX_NOREDIRECTIONBITMAP`)。
- 因而 CSS `border-radius` + `BackgroundTypeTransparent` 的路线不再被源码层直接否决，但还没有在 Win10 19045 对拖动、阴影、点击、缩放与多窗口做真机 smoke，不能直接宣称已可发布。
- `BackgroundTypeTranslucent` 是 backdrop/材质路线，不等同于纯透明圆角；Win10 与 Win11 的视觉和性能差异仍需分别验证。

## 使用约束

1. 当前生产窗口继续使用实心背景与方角，直到透明路线完成 Win10/Win11 smoke。
2. 做实验时必须保持 `IgnoreMouseEvents=false`，逐项验证按钮点击、拖拽区、焦点、AlwaysOnTop、DPI 缩放和窗口关闭。
3. 只想统一 HUD 质感时，优先走 [standalone-window-style.md](standalone-window-style.md) 的标题栏、内描边与状态色，不把系统圆角当成前置条件。

## 历史说明

2026-06-15 基于 alpha.91 得出的“透明窗必然 click-through，停止调研”在当时成立；项目升级到 alpha2.117 后源码条件已改变，该结论不再作为当前约束。
