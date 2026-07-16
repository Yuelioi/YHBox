# Plugin hosts、SDK 与 conformance

Status: completed (625a1326)

## Outcome

把已签名、已启用的 Node Package 变成唯一可信的第三方执行来源：Store 产生精确 runtime projection，Catalog 与 Program 锁定 payload/ABI/entrypoint，Process 与 Wasm host 通过同一 typed execution contract 运行，并由生成 SDK、示例包和共享 conformance 证明一致性。

## Completion criterion

- Store 只投影当前 enabled、未撤销、未 quarantine 且重新验证通过的 generation；runtime read 继续检查 registry authority 与 payload identity。
- Catalog/Program lock 精确绑定 package generation、ABI、entrypoint 与 NodeRef。
- common deterministic Protobuf 固定 canonical Value Envelope、config、trigger、resource/state/entropy/wait/action/status/result、deadline、budget、cancel 与 terminal strength，并拒绝 unknown/oversize/non-canonical frame。
- Process host 每 attempt 一进程；Windows 只允许零 capability LPAC/AppContainer + Job + explicit handle list，隔离不可用时 fail closed。
- Wasm 不实例化 WASI 或 ambient filesystem/network/process/GUI imports；限制 linear memory、wall time、frame/output/host-call，并继续运行在隔离 runner 进程内。
- 两种 host 都生成 exact compiler.InstalledAdapter，只消费 Run Session/Resource Broker authority。
- generator 输出 versioned Proto/WIT、Go/TypeScript SDK contract、Node reference、conformance vectors 与 drift gate。
- Process/Wasm 示例、签名 fixture 和共享 conformance 覆盖成功与主要失败边界。
- 没有第三方包时 composition 不创建 trust root、host 或任意代码加载面；disable/quarantine/rollback/uninstall 立即撤销旧 adapter 的 payload read。
- 阶段末统一运行 task check、跨平台 core/GUI build、真实 Windows 双插件链路与 WebView smoke。

## Implementation batches

1. common execution contract、Store runtime projection、Catalog merge：完成（310d8afd）。
2. Process protocol、sandbox launcher、adapter：完成（613bc654、1483e908）。
3. Wasm runner、无 ambient imports、adapter：完成（623ebd44）。
4. SDK/generator、示例、共享 conformance、composition：完成（b9871cf3）。
5. 阶段批量验收与 final acceptance handoff：完成（625a1326）。

## Blocked by

无。

## Verification

- task check 全绿：global Go coverage 65.0%，go vet/staticcheck 通过；frontend 28 files / 106 tests，production entry 262852 gzip bytes、editor 96843 gzip bytes。
- task build 通过，正式 Windows production build 同时生成 Yotta.exe、ScriptWorker 与 WasmPluginRunner。
- task windows:smoke:plugins 通过，签名 Process 与 Wasm 示例均走真实 Windows LPAC/AppContainer + Job 隔离链。
- Linux amd64 与 macOS arm64 portable core cross-build 通过；原生 production GUI 继续由 CI gui-build matrix 验证。
- WebView smoke 通过：100 catalog nodes、2 canvas nodes、AI review panel 可达，无 window error/rejection/console.error；截图已人工确认真实渲染。
- 覆盖率门禁合并共享 conformance profile、按 block 去重并排除标准 Code generated 文件；65% 阈值未降低。

## Out of scope

- marketplace、远程发现或自动更新服务。
- 第三方 Go plugin/shared library ABI。
- 插件 JavaScript/Vue/DOM、自定义前端 bundle 或绕过内置 Editor Adapter。
- 缺少平台隔离时退回普通 subprocess 或主进程 Wasm。
- 旧 package/ABI/protocol 兼容 reader、shim 或 fallback。

## Result

完成。已签名 package 是第三方执行的唯一来源；Process/Wasm host、SDK、示例、conformance、composition、撤销语义和阶段运行证据闭合。公开发布与跨平台 stable 承诺留给最终总审计，不扩大本 Slice。
