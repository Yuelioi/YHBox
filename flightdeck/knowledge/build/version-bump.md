# Product version maintenance

产品版本的唯一可编辑来源是仓库根目录 `VERSION`。它使用 numeric SemVer `MAJOR.MINOR.PATCH`；持久化合同、
Host Interface、wire protocol、store layout 和 private frontend package 不跟随产品 bump。

查看当前产品版本和独立版本域：

```powershell
task version:show
task versions:inventory
```

提升 patch、minor、major，或指定精确版本：

```powershell
task version:bump BUMP=patch
task version:bump BUMP=minor
task version:bump BUMP=major
task version:bump BUMP=3.2.0
```

升级命令只修改 `VERSION`，并同步 Wails、Windows VERSIONINFO、manifest 和 NSIS 的静态投影。它不提交、
不打 tag，也不要求工作区事先干净；调用者应自行审阅并按发布流程提交。只预览时使用：

```powershell
go run ./cmd/yotta-versions bump --dry-run patch
```

手动编辑 `VERSION` 后可运行：

```powershell
task version:sync
task versions:check
```

`versions:check` 是只读门禁：校验 SemVer、Windows 数值范围、所有产品投影，以及当前源码中已退役的统一合同
版本字面量。运行时版本通过 Go linker `-X` 注入；未经过正式构建入口的本地 `go run` 显示 `dev`。

`task build` 完成后会自动运行 `task versions:check:binary`，验证 EXE 的 fixed/string 版本资源和
`WINDOWS_GUI` subsystem；这同时防止产品版本遗漏和双击时出现控制台黑框。

release 验证可继续调用 `scripts/verify-version.ps1 -ExpectedVersion <tag 去掉 v>`。tag 和 commit 由发布流程显式
创建，版本工具不会隐式改写 Git 历史。
