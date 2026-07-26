# Keep repository agent instructions tracked and canonical

仓库级开发 agent 指令是公开工程 contract，必须由受版本控制的 `AGENTS.md` 单点持有，不能依赖
维护者机器上的 ignored 文件或另一个 provider-specific 副本。Canonical 文件应保持短小，只包含：

1. 仓库事实与架构导航；
2. 唯一 build/test/generate 入口；
3. 生成物、凭据、Git 和破坏性操作边界；
4. Flightdeck/ADR/领域知识入口；
5. 无法由 lint/schema/test 机械执行的少量判断规则。

provider-specific 文件若工具确实要求，只做薄适配并指向 canonical contract，不复制整份规则。
不再需要的 wrapper 直接删除。可机械验证的要求放 CI/linter；产品运行时 PromptManifest、贡献者
agent 指令和 Flightdeck Knowledge 分别由不同 owner 维护，禁止相互复制。

评审指令变更时至少验证：新 agent 能找到真实入口、不会编辑生成物、会运行当前 required gates、不会依据旧项目名/旧分支策略行动。仅靠“文本看起来完整”不能证明指令有效。
