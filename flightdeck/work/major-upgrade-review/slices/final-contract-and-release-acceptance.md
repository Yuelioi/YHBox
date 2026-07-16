# Final contract and release acceptance

Status: completed

## Outcome

对 Yotta 3.1 全部实现完成总审计，集中修复仓库工程缺口，并给出可复核 completion verdict。结论严格区分工程完成、可构建 candidate 与公开 stable。

## Completion criterion

- Slice registry 中全部 implementation Slice completed，仓库不存在旧 Container runtime、双执行路径或 release-number package/type name。
- stable nodeTypeId、显式 SemVer version、semanticDigest、Catalog/Program/package locks 与 Go/TS/schema/reference/golden 一致且 drift gate 可重建。
- GUI、headless、AI、MCP、schedule/debug 只消费唯一 compiler/Program/Run application path；第三方执行只来自已签名启用 package。
- Standards/Spec review 的仓库工程 finding 已修复；公开发布治理前置单独列明。
- 最终 acceptance matrix 以阶段批量方式执行并通过。
- candidate staging/manifest/archive/smoke 基于同一冻结 payload，sign 入口不重建。
- 当前 source-available LICENSE 未被描述为 OSI open source。

## Work completed

1. 完成 plan/design/review → code/tests/docs completion matrix 与 Standards/Spec 双轴 review。
2. 删除 `YHFISH_ADB_PATH` fallback、旧 v3 example 与 release-number generated type name。
3. 将根 desktop composition 收入 `internal/desktopapp`，生产依赖改为 constructor-time options；根 `main.go` 保持薄入口。
4. 新增复用同一 `appbootstrap.Runtime` 的 `cmd/yotta` headless validate/compile/inspect/run。
5. AI authoring review 增加容量、active/terminal TTL 与 PreparedPatch 释放；TestProfile DTO 不再携带明文 API key。
6. 刷新 architecture/compatibility/contribution/release 文档与受审计 Wails RPC contract。
7. package 链增加 CLI、冻结 manifest 文件集/size/hash、staged worker/plugin/CLI/desktop smoke；签名链只签冻结 payload 后 restage/smoke。
8. 拆分 Workflow editor/inspector 大组件，父组件回到计划尺寸阈值以内。

## Verification

- `task package`: PASS；包含唯一完整 `task check`、production build、stage、archive、frozen candidate smoke、前后 clean worktree。
- Go: global 65.0%，root 75.0%，CLI 65.3%；package floors、vet、staticcheck PASS。
- Frontend: format/lint/typecheck/i18n/bindings/Vitest/build PASS；28 files/106 tests，1269 keys，Wails 14/94/109。
- Bundle: entry 262837 gzip bytes ≤350000；editor 97277 ≤200000/target 125000。
- Candidate: manifest exact file set/size/SHA-256、staged ScriptWorker、Process/Wasm plugin isolation、CLI strict legacy rejection、desktop startup PASS。
- Race: CI race-sensitive Windows package group PASS。
- Fuzz: MCP patch、Workflow parser、CompileDraft、OpenProgram 各 10s PASS。
- Portable compile: 31 packages `go test -c` for linux/amd64 and darwin/arm64 PASS；不宣称在 Windows 运行外平台测试。
- WebView: catalog click 0→1、drag 1→2、AI review、console safety PASS；PNG 已人工检查。
- Native Linux/macOS GUI/portable execution: 保持由 CI 原生 matrix 提供权威结果。

## Blocked by

无仓库工程阻塞。

公开 stable 外部前置：OSI LICENSE、canonical public identity/repository、真实多维护者与 owner settings、Authenticode certificate/timestamp、原生宿主与 installer smoke。未经授权未 push、未创建公开仓库、未改 owner 设置。

## Result

**Verdict: major upgrade engineering complete.**

仓库可以构建、冻结并 smoke 一个 unsigned Windows engineering candidate。它尚不能被称为 OSI open source 或公开 3.1 stable；外部治理与签名条件应进入后续独立 release/governance Topic，而不是继续延长本工程迁移。
