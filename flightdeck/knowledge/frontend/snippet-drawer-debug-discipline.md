# Instrument frontend state before attempting a second fix

前端出现“状态没传到、值始终不变、事件没有触发”时，先用源码确认预期数据流，再在状态边界增加
最小诊断信号。第一次修复若没有改变症状，不继续叠加猜测；先取得运行时证据。

## Procedure

1. 从用户动作沿 component、prop/event、composable/store 到持久化 owner 画出最短数据路径。
2. 在路径两端记录同一份身份和值，必要时同时记录事件顺序；日志不得包含 credential 或用户数据。
3. 用一次可重复操作采集证据，确认第一个与预期不一致的边界。
4. 只修该边界并增加能长期证明根因的测试；临时日志在验证后删除。
5. 若必须让用户协助复现，先把操作步骤和需要返回的最小日志说明清楚，避免多轮无信息的 reload。

SaveSnippetDrawer 曾因 `editingID` 与模板 camelize 后的 `editingId` 不匹配而无法预填；运行时 props
立即暴露了第一个错误边界。具体命名规则见 [Vue prop camelize asymmetry](vue-prop-camelize-asymmetry.md)。
