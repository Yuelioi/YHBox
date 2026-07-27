# V4-R Go 清扫最终交付

## Goal

清理 `cmd/workflow-editor-smoke` 对旧装配和旧产品术语的残留，校准架构文档与真实依赖，随后在整个
Go 清扫完成点一次性执行范围门禁、Windows production build、`fishing-v2` 和完整 WebView 旅程。

## Status

Finished

## Constraints

- 中途只跑修改点相关 package/test，不运行 `task check` 或全量 build/smoke。
- 最终门禁从首次启动起使用可续接轮询，不因外层等待窗口超时重复启动。
- 保留工作区其他既有修改；不提交、不 push。

## Delivered

- `cmd/workflow-editor-smoke` 从 2964 行入口拆为 page state、retention、journey、editor exercise
  与 browser helper 五个职责文件，入口只保留参数解析和旅程调度。
- Desktop 与 CLI 的 production composition 由架构测试约束为共同进入 `localruntime`；插件 smoke
  迁到 `nodeadapter` ABI，不再复制 compiler 内部类型。
- 架构文档与兼容 ledger 对齐最终依赖、Source/Program 语义、直接升级下限和退役条件。
- 最终门禁发现并修复两个真实接缝：Windows plugin smoke 残留旧 ABI，以及空 `StorageRoot`
  未交给 canonical `storage.Resolve` 解析。

## Verification

- `task check`：退出码 0，144.6 秒；32 个受影响 Go 包通过，Wails 16 服务 / 140 方法 /
  223 models 契约通过，前端 93 个测试文件 / 394 项测试通过。
- `task build`：退出码 0，43.4 秒；production frontend、Yotta、CLI、ScriptWorker、WasmRunner、
  PE metadata 与隔离 desktop startup 通过。
- `task workflow:retention`：退出码 0；`fishing-v2` 保留 7 Graph、60 Node、18 Resource、
  36 Blob 引用与 229225 Blob bytes。
- `task webview:smoke:fishing`：退出码 0；首屏 2318ms，启动 5663ms，结果目录
  `.task/workflow-editor-smoke/20260727-211517`。
- `task webview:smoke:full`：退出码 0；启动 3820ms，结果目录
  `.task/workflow-editor-smoke/20260727-211632`；编辑器、子图、资源、计划和管理页面截图目检通过。
