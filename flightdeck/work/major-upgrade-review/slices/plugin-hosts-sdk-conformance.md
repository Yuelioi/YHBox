# Plugin hosts、SDK 与 conformance

Status: current

## Outcome

把已签名、已启用的 Node Package 变成唯一可信的第三方执行来源：Store 产生精确 runtime projection，Catalog 与 Program 锁定 payload/ABI/entrypoint，Process 与 Wasm host 通过同一 typed execution contract 运行，并由生成 SDK、示例包和共享 conformance 证明一致性。

## Completion criterion

- Node Package Store 只为当前 enabled、未撤销、未 quarantine 且重新验证通过的 generation 产生 runtime projection；路径、payload digest、manifest digest、ABI、platform 与 NodeRef 全部 host-owned。
- Catalog merge 显式合并 built-in 与 package contributions，拒绝 Node/Data Type/Capability/entrypoint 冲突；Program implementation lock 精确绑定 package generation 与 executable payload。
- common plugin protocol 固定 invocation、canonical Value Envelope、config、trigger、structured error/status、deadline、budget、取消与 terminal strength；所有 frame 有协议版本、byte/depth/count budget 和严格 unknown-field 拒绝。
- Process host 使用受限、一次 attempt 一进程的 length-delimited binary protocol；崩溃/协议错版只失败当前 attempt。Windows 必须在零 ambient capability 的 LPAC/AppContainer + Job + explicit handle list 下运行；隔离不可用时 fail closed。
- Wasm host 默认不实例化 WASI，不提供 filesystem/network/process/GUI ambient imports；限制 linear memory、wall time、frame/output 与 host-call budget，并在受限 runner 进程内执行。
- 两种 host 都转换为 exact `compiler.InstalledAdapter`，消费现有 Run Session/Resource Broker authority，不建立本地 capability 旁路，不信任 guest 自报 implementation identity。
- generator 输出 versioned Proto/WIT、Go/TypeScript SDK contract、Node reference、conformance vectors 与 drift gate；不生成或加载插件 Go ABI、JavaScript、Vue、DOM。
- 提供一个 Wasm 与一个 Process 示例 package/release fixture；共享 conformance 覆盖成功、schema violation、digest mismatch、protocol mismatch、cancel、timeout、crash、oversize、capability denial 与 handle cleanup。
- 没有第三方包时 composition 不创建任意代码加载面；安装/启用、compile/run、disable/uninstall 的可执行状态严格跟随 registry authority。
- 全部实现完成后一次性运行 `task check`、跨平台 core/GUI build、真实 Windows 两种插件链路与 WebView smoke。

## Implementation batches

1. common execution contract、Store runtime projection、Catalog merge。
2. Process protocol、sandbox launcher、adapter。
3. Wasm runner、无 ambient imports、adapter。
4. SDK/generator、两个示例、共享 conformance、composition。
5. 阶段批量验收与 final acceptance handoff。

## Blocked by

无。stable-code-names-explicit-versions 已由 022bc360 完成。

## Verification

实现期间只做能继续集成的最小 package compile/test 或生成 drift 反馈；不得把这些写成阶段验收。完整 gate 与真实 smoke 只在全部 batches 完成后运行一次。

## Out of scope

- marketplace、远程发现或自动更新服务。
- 第三方 Go plugin/shared library ABI。
- 插件 JavaScript/Vue/DOM、自定义前端 bundle 或绕过内置 Editor Adapter。
- 在缺少平台隔离能力时退回普通 subprocess 或主进程 Wasm。
- 为旧 package/ABI/protocol 保留兼容 reader、shim 或 fallback。

## Result

Current。先实现 common contract、Store runtime projection 与 Catalog merge。
