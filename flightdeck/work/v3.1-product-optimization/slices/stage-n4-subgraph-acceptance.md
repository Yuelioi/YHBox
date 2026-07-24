# N4 — 子图真实旅程与阶段验收

## Goal

用真实 store、compiler、production build 和 Windows WebView 证明 Stage N 不是只在前端内存中成立。

## Status

Finished

## Result

- 新增 workflow service 纵向测试：创建带单入口和命名 exec/error exits 的子图及两处调用，保存后从磁盘重开并
  编译；展开一处调用后再次重开/编译；最后原子移除剩余调用和定义，确认磁盘 Source 物理只剩主图并再次编译。
- 修正 WebView smoke 的真实交互与等待协议：节点清理点击标题栏并先断言选中、框选携带 Shift、自动推导显式确认
  预览、调试完成同时等待 Run 回到可再次启动状态，冷启动 hydration 使用独立 45 秒窗口。
- 移除 smoke 中已经被产品明确删除的“节点右键 → 视觉模板”旧旅程；资源库仍由独立真实页面截图覆盖。
- 2026-07-24 `task webview:smoke` 最终退出码 0。实际检查工作流列表、编辑器、子图、资源库和计划编辑器截图；
  子图入口/出口没有裁切、被工具栏遮挡或与节点端口重叠，页面无 JS error/rejection/console.error。
- 子图管理器、GraphCall Inspector、接口面板和节点 Inspector 改为低频异步 chunk；production editor gzip 从
  224827 降到 205329 字节，低于 220000 门槛。
- `task build` 退出码 0：生成 `bin/Yotta.exe`，验证产品版本 3.1.0、Windows GUI subsystem，并通过隔离桌面
  启动 smoke，进程持续存活至少 5 秒。

## Verification

- `go test ./cmd/workflow-editor-smoke -count=1`
- `go test ./internal/services/workflow -run TestServicePersistsCompilesAndPhysicallyDeletesSubgraphLifecycle -count=1`
- `task check`
- `task webview:smoke`
- `task build`

## References

- [Stage N plan](../plan.md)
- [N3 lifecycle](stage-n3-subgraph-lifecycle.md)
- [Build and acceptance](../../../knowledge/build/build.md)
