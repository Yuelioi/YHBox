# Wails/WebView 自调试 smoke

Status: completed (ab5b644f)

## Outcome

Agent 能通过单一仓库命令启动隔离 dev Wails/WebView，采集 console/page error，执行 Workflow 编辑器交互，断言 Source/画布变化并保存可人工检查的 UI 截图；production build 不暴露调试选项。

## Completion criterion

- Windows 非 production build 可通过显式环境变量设置 loopback CDP port 与独立 WebView profile。
- production 与非 Windows debug options 恒为空。
- smoke runner 使用正式 Wails DEV build 入口、独立 exe/data/profile，且只清理自己拥有的进程。
- smoke 创建临时 Workflow，断言目录单击 0→1、拖放 1→2，并拒绝 window error、unhandled rejection 与 console.error。
- smoke 保存 stdout/stderr、PNG 和隔离数据；agent 使用图片查看工具检查实际布局。
- build Knowledge 记录可重复命令、截图 settle 和已知非阻断 Wails dev 请求。
- dev smoke 与 production build/smoke 通过后独立 commit。

## Blocked by

无。

## Verification

- go test -count=1 ./cmd/workflow-editor-smoke 编译通过。
- scripts/smoke-workflow-editor.ps1 完整运行通过：99 catalog nodes、2 canvas nodes、status passed。
- 20260717-011449 workflow-editor.png 已用原始分辨率人工检查，完整编辑器 UI 可见。
- task build 通过；production options 回归与 production-tag root test 通过。
- 隔离 production 冷启动与 PrintWindow UI 检查通过。

## Out of scope

- 长驻远程调试端口或 production 调试开关。
- 通用浏览器自动化框架。
- Node Package trust、host 与最终发布验收。

## Result

完成并提交为 ab5b644f test(app): automate workflow editor WebView smoke。
