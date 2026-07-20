# Preserve complete documents during Flightdeck handoff

大文档不能通过一次可能被截断的终端输出往返重写。Flightdeck 以仓库中的普通 Markdown 为事实源；
优先对原文件做小范围 patch，避免复制工具输出后覆盖全文。

需要读取或重写大文档时：

1. 先记录文件字节数和行数，再按固定小页读取，直到明确到达 EOF。
2. 若必须生成完整替换内容，写入前确认拼接后的长度和页序与原始读取一致。
3. 修改后检查 `git diff --check`、文件行数、diff 首尾和标题结构，确认没有意外截尾。
4. 看到任何 truncation 提示时，把该次输出视为不完整证据，不用于全文替换。
5. 发现截尾后停止追加编辑，从 Git 与仍在工作区中的可信内容恢复，再重新应用最小 patch。
