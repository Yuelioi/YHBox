# Version bump checklist
版本号唯一来源是 `pkg/version/version.go` 的 `version.Version`。Go 运行时窗口标题、托盘和 `AppInfoService` 都必须读这个常量；前端通过 `backend.appInfo.info()` 获取版本，不能硬编码。

发布 bump 用：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\bump-version.ps1 -Version 3.1.1
```

或：

```powershell
task version:bump VERSION=3.1.1
```

脚本要求工作区干净，然后更新：

- `pkg/version/version.go`
- `build/config.yml`
- `build/windows/info.json`
- `build/windows/nsis/wails_tools.nsh`
- `build/windows/wails.exe.manifest`
- `build/windows/wails.dev.manifest`
- `frontend/package.json`

随后脚本会提交 `chore(release): bump version to vX.Y.Z`，再创建 `vX.Y.Z` tag。不要手动只改其中一处，也不要在未提交版本文件前打 tag；tag 必须指向版本 bump commit。

CI 与 release 的只读一致性检查使用：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-version.ps1
```

release 额外传 `-ExpectedVersion <tag 去掉 v>`，tag 与任一版本元数据不一致就失败。`verify-version.ps1` 只验证、不写文件；真正 bump 仍只走 `bump-version.ps1`。

检查脚本行为但不写文件：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\bump-version.ps1 -Version 3.1.1 -DryRun
```
