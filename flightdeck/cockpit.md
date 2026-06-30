# Cockpit — YHFish

Focus: **节点逐步调试、Target/Controller/Android、Window control 已通过并归档；旧 `knowledge/architecture` 已清理，剩余知识库已复审；正在执行容器 package schema 重设计**。本地 `main` 仍待发布/推送决策。

## In flight

- [work/detect-click-config/](work/detect-click-config/) — **代码完成，待确认是否也归档**。Vision/ClickTemplate/WaitWindowGone/Point 手填、新节点群、InputText WM_CHAR targeted、短图日志 finalize emit 已完成；近期补丁已落地：模板/颜色检测失败输出统一、`CheckTemplate/WaitTemplate/WaitTemplateGone` 支持 ROI，新增 `PickMatchPoint/PickBlobPoint`、`OffsetPoint/PointDistance/ROIAroundPoint`、`PickMatchROI/PickBlobROI` 等纯数据后处理节点。
- [work/mcp-node-exec/](work/mcp-node-exec/) — **已实现，待人工 smoke**。GUI 内置 Streamable HTTP MCP server 已接入 `http://127.0.0.1:8765/mcp`，包含 list_nodes / list_windows / find_window / run_node / save_container、arm 写入/执行闸、busy 闸和设置页 MCP tab。未 arm 时只读工具可用，run_node/save_container 拒绝。
- [work/container-package-schema/](work/container-package-schema/) — **阶段 1-2 基础已完成，下一步阶段 3 Store**。已新增 package/installation/lock 模型类型，把 `Graph.version` 破坏式改为 `Graph.schemaVersion`，并新增闭包拆分与 `yotta-lock.json` 构建；后续切多文件 Store。

## Next

1. **MCP smoke** —— 设置页确认 URL 和“允许执行和写入”开关；MCP 客户端连 `http://127.0.0.1:8765/mcp`，先测只读工具，再测未 arm 拒绝，最后 arm 后跑 Capture/ClickAt 等低风险节点。
2. **detect-click-config 归档决策** —— 如果视觉/坐标后处理节点已人工确认通过，把该 topic 也移出 `work/`。
3. **容器 package schema 阶段 3** —— 按 [work/container-package-schema/plan.md](work/container-package-schema/plan.md) 把 Store 从单 `container.json` 切到 package/graph/installation/lock 四件套。
4. **发布/推送决策** —— 若剩余 smoke 无问题，再 `git push origin main`。

## Open questions

- `PixelAt` 是否要升级为显式坐标输入/target-aware API，取代当前 Win32 鼠标 HUD 心智。
- Android 输入是否需要继续研究 minitouch/maatouch/MuMu IPC；当前先不做，ADB 通用路径已能覆盖主要用户流程。
- 前端生产构建仍有既有大 chunk / plugin timing warning，当前为非阻塞基线。
- MCP 目前固定监听 `127.0.0.1:8765` 且只用 arm 闸控制危险操作；是否再加“完全关闭服务”和端口配置，等 smoke 后决定。
- `BrowserTarget` 已按产品判断删除；底层 Browser CDP controller/client 可作为内部能力保留，但不要作为面向普通用户的节点恢复。

## Knowledge audit

2026-06-30 复审后，项目本地 `flightdeck/knowledge/` 保留 60 个文件：

- 删除重复知识 `frontend/success-feedback-inline-not-toast.md`，成功反馈规则由 `frontend/ui.md` 承载。
- 修正过期事实：节点目录现值改为命令查询口径，NeedsTarget/NeedsWindow 口径对齐当前 Target/Android 边界，SendInput failure note 收窄到仍未检查注入数的路径。
- 修正旧验证基线：Go/前端测试当前应绿，不再套 runtime fixture / i18n residue / lint 旧红豁免。
- 标准 markdown 断链和 wiki-style 交叉引用已清理。

## Verification Baseline

最新节点小周期 landing 后验证已过：

- `go test ./...`
- `pnpm --dir frontend i18n:check`
- `pnpm --dir frontend test`
- `pnpm --dir frontend typecheck`
