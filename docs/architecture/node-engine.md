# Node engine

节点由声明式 `Spec` 与恰好一种执行能力组成：`Runnable`、`RegionRunner` 或 `Evaluator`。视觉/图标记节点可以没有执行能力。注册期验证 capability invariant、pin contract 和 runtime capability；执行前验证实际 `ServiceBundle`，缺失能力返回 typed `AssemblyError`。

Registry 可实例化。生产 `init()` 只写默认实例；Store、validator、dependency scanner、catalog、runner、Script 和 MCP 使用同一个不可变 snapshot。测试使用局部 Registry，可安全并行。

`Ctx` 只暴露执行核心和窄化的 Services view。新增节点应在 `internal/nodes/<category>` 内贡献；当前不承诺动态 Go plugin 或 public ABI。

