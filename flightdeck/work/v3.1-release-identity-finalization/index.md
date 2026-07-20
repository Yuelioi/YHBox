# Yotta v3.1 Release Identity Finalization

## Goal

把产品 release identity 统一为 3.1.0，并纠正最终候选的版本与验收证据。

## Status

Finished

## Current

版本已经统一为 3.1.0，并仅存在于 version 属性、binary metadata、manifest、artifact 和 tag；没有
重新引入 release-number package、type、runtime 或模块名称。公开 stable 的许可证、签名、公开身份
和 owner settings 属于新的 release/governance Work。

## Next

None.

## Progress

- 使用正式 bump 脚本同步七个权威版本消费者。
- version verification 确认 3.1.0 一致。
- `task package`、production build、stage/archive、manifest 和 frozen candidate smoke 通过。
- Candidate manifest 记录 source commit、完整文件集合、大小与 SHA-256。
- WebView smoke 的有效截图人工确认显示 `Yotta v3.1.0`。

## References

- [Build gates](../../knowledge/build/build.md) — 当前打包和 smoke 触发条件。
- [Version bump](../../knowledge/build/version-bump.md) — 版本权威消费者。
- [Major upgrade acceptance](../major-upgrade-review/references/final-contract-and-release-acceptance.md) — 工程候选证据。
