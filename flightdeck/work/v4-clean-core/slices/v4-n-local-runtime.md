# V4-N 本地运行环境单一装配

## Goal

删除 desktop、CLI 与 automation 热替换对 storage、settings、provider 和 execution environment 的
重复装配，让一次安装事实只产生一组彼此一致的运行事实和一个明确生命周期 owner。

## Status

Finished

## Changes

- 新增 concrete `internal/localruntime.Runtime`，统一打开并逆序关闭 storage profile、catalog、
  settings/log、AI/HTTP/application/automation installations、Blob、Script、Node Package 与
  Workflow runtime。
- Desktop 与 CLI 只保留各自 presentation/encoding wiring，统一调用 `localruntime.Open/Close` 并直接
  启动其唯一 Application owner；
  源码约束测试禁止两侧重新出现重复装配入口。
- `appbootstrap` 新增 concrete sealed execution environment factory，从同一 installation snapshot
  一次生成 Host Profile、Policy、Provider snapshot 与 automation generation。
- 首次启动和 settings 热替换复用同一派生路径；发布失败整体回滚，旧 Run 继续持有旧 generation
  lease，空闲后再回收。
- Desktop 不再另行关闭 settings；`localruntime` 是 storage-backed 资源的唯一 owner。

## Verification

- `go test ./internal/localruntime ./internal/appbootstrap ./internal/desktopapp ./cmd/yotta -count=1`
- execution environment 一致性、热替换回滚、profile lease 释放和 desktop/CLI 源码边界测试通过。
- 本轮中途 `task check` 已完成并通过 19 个受影响 Go 包、93 个前端文件 / 394 项测试；后续中间
  切片不再重复运行，Windows build 与完整旅程留到整个 Go 清扫最终收尾。

## Next

继续 [运行边界校正](v4-o-runtime-boundaries.md)。
