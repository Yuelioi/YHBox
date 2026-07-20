# Yotta 3.1 Major Upgrade

## Goal

完成并验证 AI-native、destructive 的 Yotta 3.1 架构和工程发布计划。

## Status

Finished

## Current

3.1 major upgrade engineering 已完成。唯一 application/runtime path、3.1 contracts、AI authoring、
Node Package Process/Wasm host、GUI/headless seam 与冻结候选链均有最终证据。当前产物是 unsigned
engineering candidate；公开 stable 仍受许可证、签名、canonical identity 和真实维护者权限约束。

## Next

None.

## Progress

- 完成稳定 contract、desktop composition、constructor-complete 装配和 headless CLI。
- 完成 AI review budget、prompt/tool provenance、Node Package 与 Process/Wasm host。
- 冻结 candidate manifest、桌面/CLI/worker/runner、capture、ADB 和 license payload。
- Windows race、fuzz、portable-core Linux/Darwin compile 与 WebView smoke 通过。
- `task package` 覆盖完整 Go、前端、bindings、bundle 和 frozen-payload smoke。
- 外部 stable 发布前置被明确隔离，不再伪装成代码迁移未完成。

## References

- [Upgrade design](design.md) — 目标架构与破坏性迁移设计。
- [AI-native design](ai-native-design.md) — AI authoring 与运行边界。
- [Architecture review](review.md) — 最终工程审查。
- [Execution plan](plan.md) — 实施和验收批次。
- [Final contract and release acceptance](references/final-contract-and-release-acceptance.md) — 冻结候选证据。
- [OSS governance research](research/oss-governance.md) — 公开 stable 前置。
