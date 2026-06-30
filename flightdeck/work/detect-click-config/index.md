# Index — detect-click-config

## State

代码已完成，当前只等用户真机 smoke。本 topic 覆盖 Vision/ClickTemplate/新节点群/WaitWindowGone/Point 手填四阶段；另有两个已修 bug 待重测：InputText 改 WM_CHAR targeted，短图日志 LogMerger finalize 时 emit。

## Next

用户真机 smoke：
- Phase 1-4 各项回归。
- 重点重测 InputText：记事本 / VS Code 打字进目标窗口。
- 重点重测短图日志：启用节点日志后短 run 也能在 GUI 面板看到日志。
- 重测 WaitWindowGone。
- 重测 Swipe 手填 Point 与连点行为。

全过后归档本 topic。

## Read now

- knowledge/build/build.md — build / smoke 前置约定与当前验证基线。

## Read if

- design.md — 如果 smoke 暴露节点能力范围、语义或配置契约问题。
- plans/plan-phase1-vision-foundation.md — 如果排查 Vision / MatchHit / ROI / MatchMode 清理。
- plans/plan-phase2-clicktemplate.md — 如果排查 ClickTemplate 锚点、偏移、多命中、ROI、Keys、ClickCount。
- plans/plan-phase3-new-nodes.md — 如果排查 WaitTemplateGone / Swipe / InputText / Scroll 横向 / StopApp / ClickAt 扩展。
- plans/plan-phase4-point-entry-waitwindowgone.md — 如果排查 Point 手填或 WaitWindowGone。
- knowledge/input/postmessage-typetext-uses-wm-char.md — 如果 InputText 后台输入异常。
- knowledge/logging/short-run-flush-loses-dump.md — 如果短图节点日志仍不显示。

## Progress

Done:
- Phase 1 Vision 基础层重构。
- Phase 2 ClickTemplate 全家桶。
- Phase 3 新节点群与后端。
- Phase 4 Point 手填控件与 WaitWindowGone。
- InputText WM_CHAR targeted 修复。
- 短图日志 finalize emit 修复。

Remaining:
- 用户真机 smoke。

## Open questions

- 已知局限不阻塞 smoke：Swipe 在 sendinput 后端仍走 PostMessage，RawInput/DirectInput 游戏可能收不到拖拽；pkg/input SendInput 原语不查注入数，失败上报不到节点层。
