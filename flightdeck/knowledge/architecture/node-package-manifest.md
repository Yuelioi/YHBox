---
kind: note
summary: "Node Package manifest v1、Ed25519 publisher trust、namespace authority 与 registry-last lifecycle 的当前边界"
activation: action
read_when: "实现/修改 Node Package 安装、信任、升级、Catalog merge、Wasm/Process host、SDK/conformance 或插件 payload 解包前"
recheck_when: "改 internal/nodepackage manifest/signature/trust、Node Contract implementation ABI、package artifact lock、publisher namespace、host API generation 或 payload path/digest 规则时"
---
# Node Package manifest and trust

当前 Node Package 的单一 manifest、archive、signature/trust 与 local lifecycle 核心在 `internal/nodepackage`。它建立可验证、可 reopen 的内容寻址包身份和安装 authority；Catalog merge 与第三方执行 host 尚未开放。

## Manifest 与 archive

- manifest format/version 是 `yotta.node-package` / `1`；canonical JSON 经 `yotta/node-package-manifest/v1` domain hash 得到 manifest digest。
- publisher namespace 必须是 canonical HTTPS URI；package ID 以及包贡献的 Data Type、Capability、Node Type 必须位于获准 namespace。package release version 是严格 SemVer。
- manifest 内嵌 exact Data Type / Capability / Node semantic artifacts；实现只允许 Node Contract 声明的 `wit/vN` 或 `process/vN` ABI。
- executable/documentation bytes 由 portable path、raw SHA-256、size、media type 锁定。archive 拒绝额外/缺失/重复 entry、traversal、case collision、symlink/special file、zip bomb 与 mode drift。
- `ExtractArchive` 只做结构、内容和安全解包；Store admission 额外要求 signature/trust，不能把成功解包当作可安装或可执行 authority。

## Publisher signature 与 trust

- signature envelope 使用 Ed25519；canonical preimage 绑定 algorithm、publisher key ID、exact publisher namespace、package ID 与 manifest digest。key ID 是 public key 的 domain-separated digest。
- TrustPolicy 由本地主机显式 bootstrap；每个 exact namespace 绑定一组 publisher keys。未知 key、namespace mismatch、自报 owner 或仅有合法 JSON 均不构成信任。
- policy update 必须 revision 递增一、previousDigest 精确匹配，且不能删除或重分配既有 key/namespace authority；key rotation 通过保留旧 key 并新增 key，撤销状态单调累积。
- revoked key、revoked/quarantined manifest 在 install、enable、rollback 与 reopen 均 fail closed。local exact manifest digest 不再是 Store admission trust。

## Registry-last lifecycle

- immutable generation 位于 `generations/<manifest-digest>/`；canonical registry v2 是 trust policy、signature evidence、revocation/quarantine、package pointers 的唯一 authority 与 commit record。
- generation 先 durable publish，registry-last 才使 install/update 生效；incoming/orphan bytes 不可运行且 reopen 时清理。
- Store reopen、generation 复用和 rollback 重验 manifest、精确 file set、size、raw SHA-256、host-owned mode、stored signature evidence、namespace authority 与当前 trust status。
- lifecycle 支持 snapshot-only list/get、signed install/update、enable/disable、quarantine、单步 rollback 与 uninstall。quarantine 强制禁用；rollback 只选已验证且 active 的 generation 并保持 disabled。

## 与后续 host 的关系

- `nodecontract` 决定一个 Node Type 允许哪些 ABI；manifest 和 publisher trust 都不能扩大它。
- `nodecatalog.ImplementationLock` 和 Program Snapshot 必须从已验证/已启用的 package generation 生成，不接受 UI、workflow 或 plugin 自报。
- `admission`、`capability` 与 `resource` 已有 plugin instance/session authority；Wasm/Process host 必须消费这些边界，不建立本地可信旁路。
- Catalog merge、Program package lock、Wasm Component/WIT host、Process protobuf host、sandbox/quota、SDK 和 conformance fixtures 仍属于后续阶段。

在 host 与 conformance 完成前，主程序必须保持“没有第三方包时无任意代码加载面”；不要把 manifest、signature 或安装存在误写成插件可运行。
