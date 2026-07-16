# Node Package 签名与 publisher trust

Status: ready

## Outcome

Node Package Store 只接受可验证的签名 envelope 和明确的 publisher/namespace authority，并能根据 revocation 或 quarantine 输入 fail closed；local-artifact 精确 digest approval 不会扩张成 namespace ownership。

## Completion criterion

- 定义内容寻址签名 envelope、签名覆盖范围和 canonical verification preimage。
- publisher key identity 与 package namespace ownership 有明确绑定和冲突规则。
- Store install/open/update 在赋予 registry authority 前验证签名、ownership 与 trust 状态。
- revocation/quarantine 输入及其持久化、重开和回滚行为有明确 fail-closed contract。
- 测试覆盖有效、篡改、未知 key、namespace 劫持、撤销、quarantine、rollback 与 reopen。
- 受影响 Go test/race/vet/staticcheck 与跨平台 core build 通过，并独立 commit。

## Blocked by

restore-go-quality-gate。先恢复可信仓库门禁，再继续 Wave E trust。

## Verification

尚未实施。进入本 Slice 时先读取 node-package-manifest、node-package local store 与 threat-model 相关 Knowledge，并先写阶段 plan。

## Out of scope

- Catalog merge 与 runtime execution host。
- Wasm/Process sandbox、SDK 与 conformance。
- Go plugin 或第三方 JavaScript/Vue/DOM。
- 最终发布签名、installer signing 与公开 release。

## Result

Ready，未开始。
