---
kind: trap
summary: "Yotta 本地被 gitignore 的 `CLAUDE.md` 曾保留旧项目名 `YHFish`、直接提交 main 和过时的硬审批流程；它既无法随 PR 评审，也会与 Flightdeck/真实 CI/当前协作规则冲突。"
activation: symptom
read_when: "新增或修改 AGENTS.md、CLAUDE.md、Codex/Claude 指令、repo skill、贡献者自动化或“让 agent 更懂仓库”的 prompt 时"
recheck_when: "tracked `AGENTS.md` 落地、Flightdeck 入口变化、CI 主命令变化或 provider-specific instruction 文件增删后"
---
# ⚠ 未跟踪的 agent 指令会与仓库事实静默漂移
仓库级开发 agent 指令是公开工程 contract，不能依赖每个维护者机器上的 ignored 文件。canonical 文件应受版本控制、短小且只包含：

1. 仓库事实与架构导航；
2. 唯一 build/test/generate 入口；
3. 生成物、凭据、Git 和破坏性操作边界；
4. Flightdeck/ADR/领域知识入口；
5. 无法由 lint/schema/test 机械执行的少量判断规则。

provider-specific 文件若必须存在，只做薄适配并指向 canonical contract，不复制整份规则。可机械验证的要求放 CI/linter；产品运行时 PromptManifest、贡献者 agent 指令和 Flightdeck 知识分别由不同 owner 维护，禁止相互复制。

评审指令变更时至少验证：新 agent 能找到真实入口、不会编辑生成物、会运行当前 required gates、不会依据旧项目名/旧分支策略行动。仅靠“文本看起来完整”不能证明指令有效。
