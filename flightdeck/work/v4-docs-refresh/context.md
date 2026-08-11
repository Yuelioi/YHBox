# Context

## Authority

- 当前代码、schema、Task、测试和正式生成合同是事实来源；它们彼此冲突时继续追到实际生产调用路径和
  门禁，不以文档投票决定结果。
- `README.md` 与 `docs/` 描述当前产品和系统；`flightdeck/knowledge/` 只提供完成具体修改时可执行的工程
  方法。两者都可能过期，不能互相引用来证明事实。
- Work 记录保存目标、决策和进度，不是当前架构权威；历史实现只留在 Git 或 Finished Work 中。

## Documentation rules

- 先从实现得到术语、入口、所有者、数据流、版本和平台边界，再决定文档结构；不为了保留旧目录而编造内容。
- 稳定文档不硬编码容易漂移的节点数、测试数、RPC 数、bundle 大小或短期门禁输出；需要枚举时链接正式
  generator/Task 或解释如何生成。
- 命令必须来自当前 Taskfile 或 package scripts，文件路径必须真实存在，平台和许可结论必须分别受当前
  adapter/build tag/CI 与 `LICENSE` 支持。
- Knowledge 保留能服务另一项独立任务的正向操作方法；一次性故障、完成日志、旧版本路线和未验证建议留在
  Work 或删除。
- Configured Target 的文档必须遵守根 `AGENTS.md`：Network、Application、Automation 配置即授权并保持
  per-Run direct invocation，不得把已删除的逐节点安全层重新包装成推荐架构。

## Scope and boundaries

- 覆盖根 README、`docs/`、`flightdeck/knowledge/`、相关导航与文档真实性检查。
- 可按代码证据新增缺失主题、合并重复主题或删除失实内容；不在本 Work 顺手重构生产代码。
- Owner 已授权删除仅代表旧 YHBox 界面的 `preview/fish.png` 与 `preview/piano.png`。
- 不 commit、push、改 remote、改历史或发布；保留工作区其它未提交修改。
