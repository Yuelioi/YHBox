# Contributing to Yotta

感谢你改进 Yotta。当前优先接受可移植后端、automation adapter、节点、可靠性测试、文档和无障碍 UI 改进。

## 开发环境

- Go 版本以 `go.mod` 为准。
- Node 22、pnpm 11；安装前端依赖使用 `pnpm -C frontend install --frozen-lockfile`。
- Wails CLI 版本必须通过 `scripts/verify-wails-version.ps1 -CheckInstalled`。
- Windows 完整构建需要 Rust、Task 和 WGC DLL 工具链。

## 提交前检查

```powershell
gofmt -w <changed-go-files>
go test ./...
go vet ./...
staticcheck ./...
pnpm -C frontend test
pnpm -C frontend typecheck
pnpm -C frontend i18n:check
pnpm -C frontend format:check
```

并发、生命周期或持久化改动还应对受影响包运行 `go test -race`。Parser、导入和非可信 JSON 边界应补 seed corpus 或 fuzz test。

## 架构约束

- 平台中立核心不得直接 import Win32 实现；新增平台能力通过 automation controller/capability adapter。
- 后台资源必须有明确 owner，`Start` 失败可回滚，`Close` 幂等且逆序。
- 用户数据写入使用原子快照或可验证的 lock-last transaction。
- 节点声明 `RuntimeCapabilities`；缺失装配在执行前返回 typed `AssemblyError`。
- 测试使用局部 `node.NewRegistry()`；不要清空默认全局 registry。
- 当前只承诺 in-tree 节点贡献，不承诺 Go plugin ABI。

详细背景见 [docs/architecture](docs/architecture/README.md)。行为或数据格式变更需同步[兼容策略](docs/compatibility.md)和迁移测试。

## Pull request

PR 请说明问题、设计取舍、平台影响、验证命令和用户数据兼容性。避免把格式化、重命名和行为变更混在一个巨大提交中。安全问题不要公开提 issue，请按 [SECURITY.md](SECURITY.md) 私下报告。

## 许可提示

贡献者提交代码即确认自己有权提交，并同意项目当前仓库许可。当前许可不是 OSI 开源许可证；维护者改变项目许可证前需要完成法律与贡献者授权确认。

