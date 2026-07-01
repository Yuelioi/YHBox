# Index — version-bump-script

## State

实现完成，待人工按下一次真实版本号执行发布脚本。

## Next

- 发布时在干净工作区运行 `task version:bump VERSION=<next>`，确认生成 release commit 和 `v<next>` tag。
- 如未来恢复 MSIX 打包，再把 `build/windows/msix/*.xml` 纳入同一脚本。

## Read now

- flightdeck/knowledge/build/version-bump.md

## Read if

- flightdeck/knowledge/build/build.md — 如果要做正式打包验证。
- flightdeck/knowledge/frontend/ui.md — 如果继续调整标题栏版本展示。

## Progress

Done:
- 新增 `scripts/bump-version.ps1`：校验 `X.Y.Z`，dry-run，检查 tag 冲突，真实执行时要求工作区干净，更新版本文件，提交 release bump，并创建 `vX.Y.Z` tag。
- 新增 `task version:bump VERSION=...` 入口；补齐 `version:sync` 对 NSIS installer 版本的同步。
- 修复现存版本漂移：`main.go` 改为使用 `pkg/version.Version`，`build/windows/nsis/wails_tools.nsh` 同步到当前 `2.0.0`。
- 标题栏品牌旁显示 `AppInfoService` 返回的版本号；关于页已有同源版本展示，保持不重复改造。

Verified:
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\bump-version.ps1 -Version 2.0.1 -DryRun`
- 在临时 git repo 中实跑 `scripts/bump-version.ps1 -Version 9.8.7`，确认生成 commit 和 `v9.8.7` tag。
- `go test .`
- `pnpm --dir frontend typecheck`

## Open questions

- 当前 MSIX 模板仍是旧占位版本；项目现阶段 Windows package 走 Wails/NSIS，暂不纳入脚本。
