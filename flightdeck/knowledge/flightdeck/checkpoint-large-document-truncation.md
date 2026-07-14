---
kind: trap
summary: "把受终端输出预算截断的大文档内容送入 Flightdeck checkpoint 会静默截尾；必须分页读取并在落账前后核对完整性。"
activation: action
read_when: "before checkpointing a large Flightdeck topic or research document assembled from shell output"
recheck_when: "Flightdeck checkpoint accepts local paths or a non-truncating file resource instead of caller-supplied full document content"
---
# Avoid truncating large Flightdeck documents during checkpoint

Flightdeck checkpoint 接收的是调用方提供的完整 document content。终端工具展示了文件开头并不代表返回值包含全文：大文件可能已被输出预算截断。若把这段结果直接作为 checkpoint document，materialize 会把原文件静默替换成截尾版本。

安全流程：

1. 优先使用 Flightdeck resume/resource 返回的完整文档；不要从一次大型 shell 输出复制全文。
2. 必须从 shell 读取时，先取得文件字符数或字节数，再用固定小页分页读取；每页保持在工具输出预算内，并在内存中按顺序拼接。
3. checkpoint 前断言拼接后的长度等于读取前长度；对关键大文档同时记录行数。
4. checkpoint 后立即检查 `git diff --check`、文件行数和 diff 尾部，确认没有意外 EOF、半句或大段删除。
5. 一旦发现截尾，停止继续编辑；从 checkpoint 前的可信内容恢复全文，再重新应用小范围变更。

不要把“工具输出里出现了 truncation 提示”当作可以忽略的显示问题；此时内容不能用于 materialize。
