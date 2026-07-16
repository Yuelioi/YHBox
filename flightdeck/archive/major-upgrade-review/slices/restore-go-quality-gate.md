# 恢复 Go quality gate

Status: completed (27e01b17)

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

- 初始主工作树 profile 为 12808/21495=59.6%，低于 global 65% 1164 statements；隔离 c8d8b540 在同一 Go 1.25.12 工具链并补入 frontend/dist embed 输入后为 65.3%，证明预算引入时有效。
- 新测试覆盖 Node 3.1 extended evaluators、contract schema budgets、compiler scheduler instructions、Workflow 3.1 service lifecycle、authoring 原子 patch、recording/input/vision 与 Wails composition；没有降低阈值、隐藏 package 或加入无断言测试。
- 最终预算脚本 global 65.2%、根包 34.7%、recording 46.4%、input 33.9%，其余 package floors 全部通过。
- pkg/winutil callback state 使用同步 token registry，internal/automation/installed 使用同构 WindowHandle 转换；定向 tests、CGO_ENABLED=0 compile、全仓 go vet 与 staticcheck 通过。
- task check 于 2026-07-17 完整通过：Go/frontend tests、coverage、vet/staticcheck、bindings contract、format/lint/typecheck/i18n、production build、bundle 与供应链门禁全绿。
- coverage.out、.task/root-coverage.out 与临时 baseline worktree 均已清理；实现提交为 27e01b17。

## Out of scope

- 编辑器交互、桌面权限和 smoke runner 的功能扩展。
- Node Package signing、plugin host 与 AI-native 设计实施。
- 降低质量标准以追平当前数字。

## Result

完成。coverage 漂移根因是 3.1 destructive migration 用低覆盖新栈替换旧栈后未持续守住全局预算；门槛与聚合脚本本身有效。27e01b17 通过高价值回归恢复 65.2% global coverage，同时消除 vet/staticcheck 问题并让完整 task check 回到可信绿色。