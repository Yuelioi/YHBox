# Node Package manifest, trust and runtime authority

当前 Node Package 的 manifest、archive、signature/trust、local lifecycle 与 runtime projection 位于 internal/nodepackage；Catalog merge 与隔离执行宿主位于 internal/pluginhost。

## Manifest、signature 与 registry

- manifest 使用 yotta.node-package / 1；canonical JSON 经 yotta/node-package-manifest/v1 domain hash 得到 digest。publisher namespace 是 canonical HTTPS URI，package/release 使用严格身份与 SemVer。
- payload 由 portable path、raw SHA-256、size、media type 锁定；archive 拒绝 traversal、collision、symlink/special file、额外/缺失 entry、zip bomb 与 mode drift。
- Ed25519 signature 绑定 publisher key、exact namespace、package ID 与 manifest digest；TrustPolicy 是本地主机显式 bootstrap 的 namespace authority。
- revoked key、revoked/quarantined manifest 在 install、enable、rollback、reopen 与 runtime read 均 fail closed。
- immutable generation 先 durable publish，canonical registry v2 最后提交，才产生可执行 authority。

## Runtime projection 与撤权

- RuntimePackages 只投影当前 enabled、host-compatible 且重新验证通过的 generation。
- RuntimePayload 每次 Read 都重新检查 registry 当前授权和 exact payload identity；旧 adapter 在 disable、quarantine、rollback 或 uninstall 后不能继续执行。
- 程序启动只用 OpenStoreIfPresent 打开已经存在的 registry；没有 registry 或 runtime packages 时不创建 trust root、Process/Wasm host 或任意代码加载面。
- pluginhost.MergeCatalog 拒绝 type/capability/node/entrypoint 冲突；ImplementationLock 精确绑定 package ID、manifest digest、ABI 与 entrypoint。

## Host 与 SDK

- Process/Wasm 节点必须声明对应 isolation host feature，composition 只安装与 merged Catalog exact lock 一致的 adapter。
- 两个 host 共用 yotta.plugin/1 deterministic Protobuf、canonical Value Envelope、单调 sequence、deadline、byte/count/host-call/status budget 与 resource/state/entropy/wait/action/status/result 语义。
- Windows launcher 只有 LPAC/AppContainer + atomic Job List + explicit inherited handles；Wasm 不实例化 WASI，只导入 yotta_plugin_v1.exchange，并仍位于独立受限 runner 进程。
- resource call 只进入 invocation 已绑定的 Run Session；guest 不获得 path、credential、native handle、service pointer 或 capability 旁路。
- sdk/plugin、contracts/plugin/v1 与 cmd/plugin-sdk 生成和校验 Proto/WIT/Go/TypeScript contract、Node reference 与 conformance vectors；不支持插件 Go ABI、frontend JavaScript、Vue 或 DOM。
