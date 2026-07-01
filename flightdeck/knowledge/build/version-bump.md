# Version bump checklist
SUMMARY: Always bump Yotta releases through scripts/bump-version.ps1 so pkg/version, build metadata, frontend package metadata, installer metadata, commit, and tag stay aligned.
READ WHEN: before changing app version, release metadata, installer version, git release tag, about/titlebar version display, or version sync tasks.

---

版本号唯一来源是 `pkg/version/version.go` 的 `version.Version`。Go 运行时窗口标题、托盘和 `AppInfoService` 都必须读这个常量；前端通过 `backend.appInfo.info()` 获取版本，不能硬编码。

发布 bump 用：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\bump-version.ps1 -Version 2.0.1
```

或：

```powershell
task version:bump VERSION=2.0.1
```

脚本要求工作区干净，然后更新：

- `pkg/version/version.go`
- `build/config.yml`
- `build/windows/info.json`
- `build/windows/nsis/wails_tools.nsh`
- `build/windows/wails.exe.manifest`
- `frontend/package.json`

随后脚本会提交 `chore(release): bump version to vX.Y.Z`，再创建 `vX.Y.Z` tag。不要手动只改其中一处，也不要在未提交版本文件前打 tag；tag 必须指向版本 bump commit。

检查脚本行为但不写文件：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\bump-version.ps1 -Version 2.0.1 -DryRun
```
