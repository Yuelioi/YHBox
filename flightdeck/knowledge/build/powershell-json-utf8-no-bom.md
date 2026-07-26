# ⚠ Windows PowerShell 5 读取 UTF-8 no-BOM JSON 会按 ANSI 解码
2026-07-10，`verify-version.ps1` 在 pwsh 可解析 `build/windows/info.json`，但经 Task 调用 Windows PowerShell 5 时，`Get-Content -Raw` 按系统 ANSI 解码 UTF-8 no-BOM 文件，中文字段损坏并连带破坏 JSON 引号，最终 `ConvertFrom-Json` 失败。

仓库 PowerShell 脚本读取 UTF-8 JSON/文本统一使用：

```powershell
$text = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath $path))
$json = $text | ConvertFrom-Json
```

不要为兼容旧 PowerShell 给源码/JSON 重新加 BOM；Go coverage 等工具会被 BOM 阻断。
