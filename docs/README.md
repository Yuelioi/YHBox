# Yotta project knowledge

这里是面向开发者的稳定知识入口。文档只描述当前系统，不保存实施阶段、已解决故障或旧版本方案。

| 要找什么 | 从这里开始 |
| --- | --- |
| 领域术语与核心运行模型 | [Repository context](../CONTEXT.md) |
| Workflow Source、图、节点、Snippet 与导入导出 | [Workflow 与创作](product/workflows.md) |
| Target Slot、Configured Target、Capability 与资源 | [Target 与资源](product/targets-and-resources.md) |
| Run 生命周期、Debug、Timeline 与 Schedule | [Run 与 Schedule](product/runs-and-schedules.md) |
| 软件架构、关键代码、模块所有权 | [架构与代码地图](architecture/README.md) |
| Workflow 如何保存、编译和运行 | [运行链](architecture/runtime.md) |
| Data/Node/Program/Run/RPC 合同如何关联 | [合同与生成投影](architecture/contracts.md) |
| 本地数据存在哪里、哪些可以重建 | [本地存储](architecture/storage.md) |
| 信任边界与高风险能力 | [威胁模型](architecture/threat-model.md) |
| 版本域、兼容与迁移 | [兼容策略](compatibility.md) |
| Windows/Linux/macOS/Target 支持程度 | [平台支持](platform-support.md) |
| Headless CLI 的 validate/compile/run/health/migrate | [CLI reference](reference/cli.md) |
| 许可与公开发布差距 | [发布就绪](open-source-readiness.md) |
| 构建、节点、编辑器、自动化、Wails 修改方法 | [任务知识](../flightdeck/knowledge/README.md) |

## 知识职责

- `docs/`：当前产品模型、架构、数据、兼容、安全和操作边界。
- `flightdeck/knowledge/`：完成一类修改时可直接执行的项目指南。
- `flightdeck/work/`：仍在进行或已经结束的工作上下文，不是当前架构权威。
- 代码、schema、Task、测试和生成合同：最终事实来源。文档与它们冲突时先以实现为准，再修正文档。
- Git：历史和被替换方案的长期记录；不要把 bug 时间线重新复制回核心知识。
