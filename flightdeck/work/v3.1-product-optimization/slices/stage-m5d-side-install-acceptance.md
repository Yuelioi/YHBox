# M5d — 派生隔离与旁装验收

## Journey

本地派生 Source 始终是独立 workflow identity。即使它后来通过 trust boundary 成为 verified Release，
也不能作为原 Installation 的 update/merge candidate；它只能显式安装为新 Installation。任一 immutable
Release 均可重复安装为使用不同本机配置的实例。

## Verification

- 从 Installation 派生 Source、把派生 artifact 投影为 verified Release 后，原 Installation 的
  `PrepareUpdate` 因 workflow identity 不同而 fail closed。
- 同一个 derived Release 可由 `InstallVerified` 旁装为独立 Installation；原 Installation/current Release/
  configuration 保持不变。
- M5a–M5d 定向测试、最终 `task check` 与 Windows WebView smoke 通过。

## Status

Finished.

## Evidence

- `TestDerivedReleaseCannotOverwriteOriginAndCanSideInstall` 覆盖 derive → verified Release →
  origin update rejection → independent side install，且原 Installation/current Release 不变。
- Stage 最终 `task check` 退出 0，34 个受影响 Go 包通过。
- Windows WebView smoke `20260726-123823` 退出 0；显式“编辑副本”进入现有 Source 编辑器，
  截图已目检。
