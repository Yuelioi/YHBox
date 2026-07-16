---
format: 4
project: yotta
focus: major-upgrade-review
---

# Flightdeck

## Conventions

- 对话、spec、plan 一律使用中文。
- 大型任务先定义包含多个相邻 Slices 的可交付阶段；Slice 与本地 commit 只作为实现、恢复和回滚边界，不各自触发全量验收。
- 阶段实施中只运行支撑继续开发所需的最小定向 test/compile/static check；除非用户明确要求或存在必须立即验证的高风险边界，不重复运行 `task check`、完整 race、跨平台矩阵、production build 或真实 GUI/plugin smoke。
- 阶段全部实现后再批量验收一次：先运行阶段相关聚合测试，再运行 `task check`，并按触发条件运行 cross-platform build、真实 Windows WebView/plugin smoke 和人工视觉检查。
- 阶段中遇到无关 flaky failure 时先做定向复现并记录，不立即反复重跑整套门禁；在阶段末 acceptance gate 中统一要求可信绿色结果。

## Open questions

- 发布继续延期：未明确授权前不创建或推送 `yottaapp/yotta`；先决定 OSI 许可证、canonical identity 与本地领先历史的安全公开方式。
