# Cockpit — YHFish

Focus: **节点逐步调试、Target/Controller/Android、Window control 已通过并归档；旧 `knowledge/architecture` 已清理**。当前 `main` 顶部为 `cac3d8c fix(node-explorer): label pure data nodes as common`；本地仍待发布/推送决策。`work/` 只保留视觉/坐标后处理回归包和 MCP smoke 包。

## In flight

- [work/detect-click-config/](work/detect-click-config/) — **代码完成，待确认是否也归档**。Vision/ClickTemplate/WaitWindowGone/Point 手填、新节点群、InputText WM_CHAR targeted、短图日志 finalize emit 已完成；近期补丁已落地：模板/颜色检测失败输出统一、`CheckTemplate/WaitTemplate/WaitTemplateGone` 支持 ROI，新增 `PickMatchPoint/PickBlobPoint`、`OffsetPoint/PointDistance/ROIAroundPoint`、`PickMatchROI/PickBlobROI` 等纯数据后处理节点。
- [work/mcp-node-exec/](work/mcp-node-exec/) — **已实现，待人工 smoke**。GUI 内置 Streamable HTTP MCP server 已接入 `http://127.0.0.1:8765/mcp`，包含 list_nodes / list_windows / find_window / run_node / save_container、arm 写入/执行闸、busy 闸和设置页 MCP tab。未 arm 时只读工具可用，run_node/save_container 拒绝。

## Next

1. **MCP smoke** —— 设置页确认 URL 和“允许执行和写入”开关；MCP 客户端连 `http://127.0.0.1:8765/mcp`，先测只读工具，再测未 arm 拒绝，最后 arm 后跑 Capture/ClickAt 等低风险节点。
2. **detect-click-config 归档决策** —— 如果视觉/坐标后处理节点已人工确认通过，把该 topic 也移出 `work/`。
3. **发布/推送决策** —— 若剩余 smoke 无问题，再 `git push origin main`。

## Open questions

- `PixelAt` 是否要升级为显式坐标输入/target-aware API，取代当前 Win32 鼠标 HUD 心智。
- Android 输入是否需要继续研究 minitouch/maatouch/MuMu IPC；当前先不做，ADB 通用路径已能覆盖主要用户流程。
- 前端生产构建仍有既有大 chunk / plugin timing warning，当前为非阻塞基线。
- MCP 目前固定监听 `127.0.0.1:8765` 且只用 arm 闸控制危险操作；是否再加“完全关闭服务”和端口配置，等 smoke 后决定。
- `BrowserTarget` 已按产品判断删除；底层 Browser CDP controller/client 可作为内部能力保留，但不要作为面向普通用户的节点恢复。

## Archived this turn

- `node-step-debugging`
- `target-controller-upgrade`
- `window-control`

## Verification Baseline

最新节点小周期 landing 后验证已过：

- `go test ./...`
- `pnpm --dir frontend i18n:check`
- `pnpm --dir frontend test`
- `pnpm --dir frontend typecheck`
