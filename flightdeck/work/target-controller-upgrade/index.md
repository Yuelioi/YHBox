# Index — target-controller-upgrade

## State

破坏性大升级 topic。先完成调研与总体设计，尚未进入代码实现。核心决策：Go 保持主运行时，Rust 只作为 Win32/native controller hot path；先引入 `Target / Controller / CoordinateSpace / Trace`，再迁移节点、Android、浏览器和输入后端矩阵。

## Next

按 `plan.md` 执行 Phase 1：新增 `internal/automation/target` 与 `internal/automation/controller`，用现有 Win32 input/capture/window 能力包一层兼容 adapter，不改容器 JSON，不重写全部节点。

推荐执行方式：Subagent-Driven，逐 task 实现、逐 task review。

## Read now

- design.md
- plan.md

## Read if

- ../../knowledge/architecture/automation-framework-survey.md — 需要回看 ok-script / MaaFramework / Airtest / RPA 调研结论。
- ../../knowledge/architecture/target-controller-upgrade-guide.md — 需要回看长期升级路线、Go/Rust 分工、Android/Win32/Browser 策略。
- ../../knowledge/nodes/node-system-architecture.md — 迁移节点或 runtime service 前。
- ../../knowledge/subgraph/asset-subsystem.md — 改截图取点、资产 capture、模板变体前。
- ../../knowledge/input/sendinput-primitive-size-and-return.md — 调 Win32 SendInput primitive 前。

## Progress

Done:
- 市面框架调研。
- 总体设计 spec。
- Phase 1 implementation plan。
- cockpit 中加入恢复入口。

Current:
- 未实现代码。

## Open questions

- 当前工作区已有 AE 修复、i18n 清理、runtime test skip 等未提交改动；开工前需要决定是一起保留、拆提交，还是先清理工作区。
- Phase 1 计划中的接口名可在执行前按源码现实微调，但不得扩大范围到 Android、Browser 或 Trace UI。
