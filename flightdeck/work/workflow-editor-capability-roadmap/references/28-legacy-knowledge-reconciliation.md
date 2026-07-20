# Slice 28：旧版 Knowledge 与架构文档复查

## Outcome / Question

在执行 Slice 27 R0 前，用用户指定 3.0 基线与当前代码复查 `flightdeck/knowledge`、`docs` 中的新旧架构信息，纠正会误导后续恢复工作的现行知识，并保留可用的旧产品行为证据。

## Completion criterion

- 高风险 current-sounding 旧知识不再把 Container/nodepkg/Expr/yt/nodes31 当现行实现。
- 平台、input selector、installation generation、UAC、录制、资产 picker 和验收边界与当前证据一致。
- 旧行为与 3.1 替代关系写入 topic-local context。
- 不删除仍被 archived Topic 引用的历史知识；路由被收窄到旧基线取证。

## Blocked by

无。

## Verification

- 全仓搜索不存在的 `internal/node`、`internal/services/container`、`nodes31runtime`、旧 exact-only/macOS-only-Adapter/UAC 声明。
- 检查所有修改 Knowledge 的严格 frontmatter 和链接。
- `git diff --check` 与 Flightdeck validation；本 Slice 不运行代码测试/build。

## Out of scope

- 不修改业务代码、workflow 数据或 runtime 状态。
- 不执行 Slice 27 的旧版 worktree、capability ledger 或黄金旅程实现。
- 不重写研究/Wayfinder provenance，也不删除 archived Topic。

## Result

- 当前与历史资料的读取层级、旧版有用行为、新旧差异和框架健康结论已写入 [`../references/knowledge-and-docs-review.md`](../references/knowledge-and-docs-review.md)。
- 现行架构/节点/录制/资产/build Knowledge 已升级；旧 Container/Expr/yt/subgraph trap 被标为历史 3.0 并收窄路由。
- `docs/architecture` 已纠正 platform installation seam、capability 二次投影、Windows admin threat 和 Android/Browser 支持等级。
- 下一步回到 Slice 27 Stage R0。

