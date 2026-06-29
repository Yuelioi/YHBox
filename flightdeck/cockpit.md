# Cockpit — YHFish

Focus: **节点逐步调试 + target/controller/Android + MCP 对外暴露大周期已 landing 到本地 `main`，视觉/坐标后处理节点小周期也已 landing**。当前 `main` 顶部为 `24dfd9f feat(nodes): add visual roi pickers`；本地还未 push。下一步优先做人工复核/发布决策。

## In flight

- [work/node-step-debugging/](work/node-step-debugging/) — **已 landed，待最终人工复核/推送**。V1 节点调试已贯通 runtime、service/RPC、Wails bindings、编辑器 toolbar、右键 Debug from here、Step/Continue/Pause/Stop、画布高亮、状态面板、状态重同步和 Wails event payload 兼容。近期修复：AndroidTarget step 后能进入下一节点；禁用节点不会卡住 Step；去掉运行/调试成功 toast；toolbar 改成 IDE 风格左/中/右分区，保存/检测/调试/运行集中在中间。
- [work/target-controller-upgrade/](work/target-controller-upgrade/) — **Phase 1-73 已 landed，进入人工复核/后续小切片阶段**。Go 保持主运行时，Rust 只下沉 Win32/native hot path；Target/Controller/CoordinateSpace/Trace、Win32/Android controller、target-aware vision、ADB discovery、Android picker 截图、AndroidStartApp/AndroidStopApp、内置 ADB、支持目标 badge、能力校验和大量 guard 已落地。`BrowserTarget` 用户入口已删除，不要恢复为普通节点。当前遗留边界：`PixelAt` 仍是 Win32 MouseHUD 语义，Android 坐标取点需要新 API 再做。
- [work/window-control/](work/window-control/) — **代码和关键 smoke 问题已并入主线，作为历史 topic 保留**。双 `Win32WindowTarget` 默认配置共享、StopApp 完整 exe 路径失败都已修复；后续只作为回归参考。
- [work/detect-click-config/](work/detect-click-config/) — **代码完成，作为历史/回归 topic 保留**。Vision/ClickTemplate/WaitWindowGone/Point 手填、新节点群、InputText WM_CHAR targeted、短图日志 finalize emit 已完成；近期补丁已落地：模板/颜色检测失败输出统一、`CheckTemplate/WaitTemplate/WaitTemplateGone` 支持 ROI，新增 `PickMatchPoint/PickBlobPoint`、`OffsetPoint/PointDistance/ROIAroundPoint`、`PickMatchROI/PickBlobROI` 等纯数据后处理节点。
- [work/mcp-node-exec/](work/mcp-node-exec/) — **已实现，待人工 smoke**。GUI 内置 Streamable HTTP MCP server 已接入 `http://127.0.0.1:8765/mcp`，包含 list_nodes / list_windows / find_window / run_node / save_container、arm 写入/执行闸、busy 闸和设置页 MCP tab。未 arm 时只读工具可用，run_node/save_container 拒绝。

## Next

1. **发布/推送决策** —— 当前 `main` 本地 ahead 远端，若人工复核无问题，再 `git push origin main`。
2. **人工复核建议** —— 用已测容器抽样跑：`AndroidTarget -> AndroidStartApp`、Android 截图取点/范围、调试 Step/Continue/Pause/Stop、禁用节点跳过、toolbar 保存/检测/运行/调试布局；视觉节点可抽样跑 `FindTemplateAll -> PickMatchPoint/PickMatchROI -> ClickAt/DetectColorBlobs` 和 `DetectColorBlobs -> PickBlobPoint/PickBlobROI`。
3. **MCP smoke** —— 设置页确认 URL 和“允许执行和写入”开关；MCP 客户端连 `http://127.0.0.1:8765/mcp`，先测只读工具，再测未 arm 拒绝，最后 arm 后跑 Capture/ClickAt 等低风险节点。

## Open questions

- `PixelAt` 是否要升级为显式坐标输入/target-aware API，取代当前 Win32 鼠标 HUD 心智。
- Android 输入是否需要继续研究 minitouch/maatouch/MuMu IPC；当前先不做，ADB 通用路径已能覆盖主要用户流程。
- 前端生产构建仍有既有大 chunk / plugin timing warning，当前为非阻塞基线。
- MCP 目前固定监听 `127.0.0.1:8765` 且只用 arm 闸控制危险操作；是否再加“完全关闭服务”和端口配置，等 smoke 后决定。
- `BrowserTarget` 已按产品判断删除；底层 Browser CDP controller/client 可作为内部能力保留，但不要作为面向普通用户的节点恢复。

## Verification Baseline

最新节点小周期 landing 后验证已过：

- `go test ./...`
- `pnpm --dir frontend i18n:check`
- `pnpm --dir frontend test`
- `pnpm --dir frontend typecheck`
