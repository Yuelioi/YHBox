# V4 发布兼容性封板计划

## Outcome

V4 首发格式与节点合同可冻结、可检查、可升级；后续演进有自动化门禁和真实磁盘回放测试保护。

## Stage 1 — 代码事实清单

- [x] 枚举产品版本、持久化 artifact、节点/数据合同、协议与 store layout 的代码 owner。
- [x] 标出用户资产、可重建派生物和进程内协议，记录 reader、writer、migration 与错误传播事实。
- [x] 对照现有 inventory、schema、Task 和测试，列出发布阻断与可延后债务。

## Stage 2 — 节点与工作流兼容底线

- [x] 冻结 V4 节点合同基线，禁止同版本 semantic digest 原地漂移。
- [x] 验证旧 NodeRef 的相邻迁移、边/参数保持和编译诊断，覆盖真实磁盘工作流重开。
- [x] 补齐 Workflow Source 的版本迁移发布与恢复测试，确保失败不静默吞掉。

## Stage 3 — 其他持久化域

- [x] 核对 Schedule、Macro、Settings、Run、Blob、Asset/Resource 等格式升级的原子写回与重开行为。
- [x] 核对 package、worker、plugin、MCP/Host 等协议协商与不兼容错误。
- [x] 把跨域升级顺序收进少数深 module，避免调用方各自实现迁移。

## Stage 4 — 自动化与发布验收

- [x] 将兼容基线、migration coverage 和版本漂移检查接入正式 Task/CI。
- [x] 用旧样本执行迁移、重开、编译与运行准备回放，并核对用户可操作错误。
- [x] 更新代码支持的兼容文档，运行发布范围门禁并保存最终结论。

## Stage 5 — External release handoff

- [x] 确认 canonical `github.com/yuelioi/yotta` 可访问并切换本地 `origin`。
- [ ] Review、commit 当前变更，从 clean worktree 完成 `task package` 与 frozen candidate smoke。
