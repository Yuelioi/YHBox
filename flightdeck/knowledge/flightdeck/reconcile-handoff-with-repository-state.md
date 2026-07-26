# Reconcile the Flightdeck handoff with repository state

Work 页面是恢复入口，不是可以脱离仓库事实执行的命令。开始 Focus Work 前，应把 `index.md` 的
Current/Next 与当前源码、测试证据、Git 状态和最近历史交叉核对。

当 handoff 与仓库现实不一致时：

1. 保留工作区未提交内容，先查看 `git status`、相关 diff、近期 commits 和当前实现，不直接用 HEAD 覆盖。
2. 判断差异是较新的未保存进展、过期描述，还是尚未解决的真实冲突；证据不足时保留歧义并向用户说明。
3. 已完成的交付不得因旧 Next 再次执行；同样，未提交的新进展也不得因 Git 较旧而被丢弃。
4. 事实明确后，重写 Work 的 Current、Next 和最近 Progress，使下一会话只看到仍然成立的恢复语义。
5. 用户指出“这个已经完成”或类似漂移信号时，先重新核对实现和验证证据，再继续执行。
