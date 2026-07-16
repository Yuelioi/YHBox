# Node Package 签名与 publisher trust

Status: completed (ab57d572)

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

无。

## Verification

ab57d572 增加 canonical TrustPolicy 与 Ed25519 SignatureEnvelope；preimage 绑定 algorithm、publisher key ID、exact namespace、package ID 与 manifest digest。CreateStore 显式建立本地 trust root，后续 policy 必须 revision+previousDigest 单调扩展且不能移除/重分配既有 publisher authority。

registry v2 在同一 canonical commit 中持有 trust policy、signature evidence、revocation/quarantine 和 generation pointers。Store 只安装 signed archive；未知/撤销 key、namespace mismatch、manifest revoke/quarantine、policy rollback 与受阻 generation enable/rollback/reopen 全部 fail closed。

定向 Go test/vet/staticcheck/race、Windows/Linux/macOS core compile 与 internal/nodepackage 75.1% coverage 均通过。全量门禁属于扩展平台阶段末批量验收，不作为本 Slice 独立 acceptance gate。

## Out of scope

- Catalog merge 与 runtime execution host。
- Wasm/Process sandbox、SDK 与 conformance。
- Go plugin 或第三方 JavaScript/Vue/DOM。
- 最终发布签名、installer signing 与公开 release。

## Result

Completed in ab57d572。Store admission 已从 local exact digest approval 切到 verified publisher signature + exact namespace authority；trust update 和 package authority 共享 registry-last commit。
