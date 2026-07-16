# 恢复 Go quality gate

Status: current

## Outcome

恢复 task check 的可信绿色基线：coverage、go vet 和 staticcheck 都对当前仓库真实通过，且没有降低阈值、隐藏 package 或绕过 hook。

## Completion criterion

- 用确定性脚本重现并解释 tracked 65% coverage budget 与 detached HEAD 53e6d8a9 实测 59.8% 的差异来源。
- 修复方式保持有意义的回归压力；不得通过 blanket exclude、新增无断言测试或静默降低 globalMinimum 伪造绿色。
- go test -count=1 -covermode=atomic -coverprofile=coverage.out ./... 与 check-go-coverage.ps1 通过。
- 修复 pkg/winutil/window_windows.go:385 的 go vet unsafe.Pointer 警告，并以邻近测试锁定安全转换。
- 修复 internal/automation/installed/platform_windows.go:64 的 staticcheck S1016，不改变 handle identity 语义。
- task check 完整通过；清理 coverage.out 和临时基线 worktree，并独立 commit。

## Blocked by

无。

## Verification

当前红灯证据：

- 主工作树 task check：全仓 Go tests 通过，global statement coverage 59.6% < 65%，随后停止。
- 去掉 cmd/workflow-editor-smoke 的 113 条 statement 后约为 59.9%，不足以解释 65% 差距。
- detached HEAD 53e6d8a9 在相同 Go 1.25.12 工具链和现有 frontend/dist 下实测 59.8% < 65%。
- go vet ./...：pkg/winutil/window_windows.go:385:36 possible misuse of unsafe.Pointer。
- staticcheck ./...：internal/automation/installed/platform_windows.go:64:9 S1016。
- frontend check、task build、dev WebView smoke 和 production EXE smoke 已独立通过。

## Out of scope

- 编辑器交互、桌面权限和 smoke runner 的功能扩展。
- Node Package signing、plugin host 与 AI-native 设计实施。
- 降低质量标准以追平当前数字。

## Result

已建立稳定红灯和基线对照，根因修复尚未开始。下个对话从 coverage 预算/汇总历史与最小缺口归属审计开始。
