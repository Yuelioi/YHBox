# 稳定代码命名与显式版本属性恢复

Status: completed

## Outcome

删除由产品 release 3.1 派生出的结构性代码命名；Go/TypeScript 模块按职责保持稳定，应用版本、artifact format generation 与单个 Node 版本分别由显式属性承载。

## Completion criterion

- 先生成全仓 impact map，区分产品版本、wire/artifact format 版本、Node 实体版本和纯代码模块名，禁止再把这些维度混为一个 “31” 后缀。
- 将 `internal/nodes31`、`internal/nodes31runtime`、`internal/services/workflow31` 迁移为无 release 后缀的稳定职责名；同步所有 Go imports、package declarations、composition root、commands、tests、architecture guards 与生成入口。
- 将 `frontend/src/contracts/node31.ts` 等结构性文件/identifier 改为稳定语义名；同步前端 imports、tests 与生成脚本。
- Node identity 使用稳定 `nodeTypeId` 加显式版本属性；若现有 contract 只靠 URI/path 后缀表达版本，先冻结新的 NodeRef/schema/hash preimage，再同步 Catalog、Program、manifest、Go/TS projection 与 golden fixtures。
- artifact/wire generation 继续使用显式 `format` / `version` 和独立 hash domain；产品 release 只由 `pkg/version` 与 release metadata 管理。不得用代码目录、package、type、service 或 filename 承载产品 SemVer。
- 全仓 `git grep` 对结构性 `31` 命名归零；只允许 serialized contract version、fixture、compatibility statement 和历史 Flightdeck/commit evidence 中出现 3.1。
- 更新架构/Wayfinder/plan 中会诱导“Node System 3.1 = nodes31 package”的表述；本 Slice 与 plugin hosts/SDK 完成后统一执行阶段门禁。

## Blocked by

无。node-package-signing-trust 已由 ab57d572 完成。

## Verification

- 022bc360 将 `internal/nodes31`、`internal/nodes31runtime`、`internal/services/workflow31`、`node31.ts`、`workflow31.ts` 与 `source31_test.go` 迁移为稳定职责名，没有 alias、兼容 package 或双 import path。
- `NodeRef`、`Draft` 与 `MachineContract` 新增必填严格 SemVer `version`；`nodeTypeId` 改为稳定 canonical URI 并拒绝 `/vN` 尾段。节点版本进入 semantic preimage，Catalog、Workflow Source、Node Package、Go/TS schema、projection、golden artifact 与 fixture 已同步。
- `3.1` 继续只表示 Node Contract / Workflow Source artifact generation；生成的 TypeScript `YottaWorkflowSource31` 等 format-bound 类型属于允许项。
- Wails bindings 无 warning 生成；`go test -run '^$' ./...` 暴露的稳定别名冲突已修复，受影响包 compile-only 通过；`pnpm -C frontend typecheck` 通过。
- `go test ./internal/nodecontract ./internal/nodepackage ./internal/nodes ./internal/nodecatalog ./internal/workflow/schema` 通过；最终 Node contract 定向测试通过。
- 按阶段制约定没有运行 `task check`、跨平台矩阵或真实 GUI/plugin smoke。

## Out of scope

- 改变 Yotta 当前产品 release 号。
- 保留 `nodes31` alias、兼容 package、双 import path 或长期 shim。
- 顺带重写节点行为、runtime 调度或插件 host。
- 删除合法的 serialized contract format/version、artifact hash domain version 或发布 metadata。

## Result

Completed by 022bc360. 稳定代码名与显式 Node version contract 已恢复；阶段验收延后到 plugin hosts/SDK/conformance 完成后批量执行。
