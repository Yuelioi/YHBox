# Cockpit — Yotta

Focus: **制定面向大型开源项目的破坏性升级方案。** 全仓审查架构、接口、工程门禁与发布治理，不保留旧契约、不添加兼容层或兜底。

## In flight

- `major-upgrade-review` — 全仓审查与 Yotta 3.0 九阶段方案已完成，等待确认实施入口。
- `recording-asset-lifecycle` — 实现与自动化验证已完成；用 `bin/Yotta.exe` 验收三条用户路径后归档。

## Open questions

- 发布已延期：不创建或推送 `yottaapp/yotta`；继续保留本地历史与旧 origin，避免误推。
- 真正公开前需决定维持 source-available 或切换 OSI 许可证，启用 GitHub private vulnerability reporting，并观察三平台 CI 首跑。

- `PixelAt` 是否要升级为显式坐标输入/target-aware API，取代当前 Win32 鼠标 HUD 心智。
- Android 输入是否需要继续研究 minitouch/maatouch/MuMu IPC；当前先不做，ADB 通用路径已能覆盖主要用户流程。
- 前端生产构建仍有既有大 chunk / plugin timing warning，当前为非阻塞基线。
- MCP 目前固定监听 `127.0.0.1:8765` 且只用 arm 闸控制危险操作；是否再加“完全关闭服务”和端口配置，等 smoke 后决定。
- `BrowserTarget` 已按产品判断删除；底层 Browser CDP controller/client 可作为内部能力保留，但不要作为面向普通用户的节点恢复。
- 通用节点下一批候选：`WriteTextFile`、`WriteJsonFile`、`WatchFile`、Fetch cURL 导入/HTML 提取、JSON merge/map/filter。
