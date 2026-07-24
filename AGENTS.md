# Yotta repository guide

本文件是仓库级 agent contract。保持简短；可机械执行的规则放进 `task check`、CI、schema 或测试，不在这里复制长 prompt。

## 仓库事实

- Yotta 是 Wails v3 桌面应用：Go 后端位于仓库根、`internal/` 和 `pkg/`，Vue 3/TypeScript 前端位于 `frontend/`。
- Windows 是完整支持平台；Linux/macOS 目前只承诺平台中立核心可测试、GUI 可编译且为预览级。
- 当前 `LICENSE` 是 source-available、不是 OSI 开源许可证；许可证替换前不要对外称为 OSI open source。

## 架构导航

- `main.go`：只保留进程入口与嵌入资源；桌面 composition 位于 `internal/desktopapp/`，生命周期实现位于 `internal/appruntime/`。
- `internal/nodecontract/` 与 `internal/datatype/`：节点/数据契约；`internal/workflow/compiler/` 与 `internal/noderuntime/`：唯一工作流执行路径。
- `internal/automation/`：平台中立 target/controller contract；平台能力通过 adapter 接入。
- `internal/services/`：应用服务；`pkg/`：可复用 adapter/helper；`cmd/`：仓库工具。
- `frontend/src/`：UI 与编辑器；`docs/architecture/README.md` 是架构文档入口。

## 验证入口

交付前从仓库根目录运行按 Git 变更范围选择的本地门禁：

```powershell
task check
```

该门禁通常超过 60 秒。Agent 必须从首次执行起使用可续接、可轮询的等待方式，并为命令保留足够的
内部超时；外层调用窗口超时不等于门禁失败，不得因此重复启动。重试前必须确认原进程已经结束并取得
真实退出码。

`task check:full` 是 CI、发布和明确要求完整验收时的全量门禁，不作为普通修改的默认收尾。不要在其他
文档维护平行的命令清单；增量路由、full、race、跨平台 GUI build、打包和真机 smoke 的触发条件见
`flightdeck/knowledge/build/build.md`。

## 生成物与修改边界

- `frontend/bindings/` 由 Wails 生成且被 gitignore；不要手改。改 Go 导出 API 后通过正式 Task/Wails 入口重新生成。
- `frontend/dist/`、`bin/`、`coverage.out`、`.task/`、`logs/`、`data/`、`settings.json` 是本地产物或用户数据，不提交。
- 保留工作区中不属于当前任务的改动；不要顺手格式化、重命名或提交无关文件。

## Git 与安全

- 未经用户明确授权，不 push、不改写历史、不丢弃现有改动，也不绕过 hook。
- 不提交 token、API key、凭据、用户数据或私有样本；日志、错误和 fixture 必须脱敏。
- 把 workflow/package/MCP、文件、网络和进程输入视为不可信；不得绕过 capability、授权或 arm 边界。

## Flightdeck

- 恢复长期工作时先读 `flightdeck/deck.md` 的 Focus，再完整读取对应 Work 的 `index.md`、
  `context.md` 和可选 `plan.md`，并核对实时 Git 状态。
- `deck.md` 只保存 Open Work 列表和稳定项目链接；存在 Open Work 时必须且只能标记一个 Focus。
  Work 的 Goal、Status、Current、Next 与最近 Progress 写入 `flightdeck/work/<id>/index.md`，稳定
  目标上下文写入同目录 `context.md`。
- `Next` 只直接列出当前动作所需的最多三个本地链接；其余资料按需从 References 读取，不创建
  单独的 `Read now` 路由区块。
- `flightdeck/knowledge/` 是按主题组织的普通 Markdown 指南，不使用 frontmatter、路由字段、
  revision、陷阱分类或复查账本；只有当前任务确实需要时才按主题搜索。
- 保存进度是重写可见 handoff，不自动 commit。完成 Work 时原地改为 `Finished`，不移动目录。
