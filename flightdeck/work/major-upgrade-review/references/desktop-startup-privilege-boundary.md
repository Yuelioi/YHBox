# 桌面启动权限边界

Status: completed (c3cab6e4)

## Outcome

Yotta 桌面 composition root 以调用用户身份启动，不再启动即触发 UAC；高完整性目标只能由明确 capability/provider 支持，缺失时 fail closed。

## Completion criterion

- main composition root 不调用 EnsureAdmin 或等价进程级 runas。
- 不保留对外可调用的 pkg/platform 全局提权 helper。
- Windows 与非 Windows构建保持 console/platform seam 可编译。
- 回归锁定 composition root 不得重新引入进程级提权。
- dev-only WebView options 平台隔离，production options 恒为空。
- 受影响 Go tests、production-tag tests、production build 与普通 Windows EXE 冷启动 smoke 通过。
- 源码、测试、threat model 与 lifecycle Knowledge 独立 commit。

## Blocked by

无。

## Verification

- go test -count=1 . ./pkg/platform 通过。
- go test -count=1 -tags production . 通过。
- task build 通过并生成 production bin/Yotta.exe。
- 隔离 production EXE 冷启动无需 UAC，窗口标题 Yotta 2.0.0、bounds 1180x720。
- 20260717-011619 production-window-print.png 已人工检查：首屏直接显示工作流列表、新建入口和日志面板，无黑屏。

## Out of scope

- 为具体高完整性目标实现 elevated provider。
- Workflow smoke runner 业务逻辑。
- 插件执行 host 或系统级 sandbox。

## Result

完成并提交为 c3cab6e4 fix(app): keep desktop process unprivileged。
