# Cockpit — Yotta

Focus: **录制坐标、折叠子图空内容及 package manifest 空链接均已完成代码修复。** 新生产二进制已构建，等待重启后的本地 Wails smoke。

## In flight

- `recording-subgraph-critical-bugs` — 折叠已确认；录制与 package manifest 等待新二进制重启、保存及复录验证。

## Open questions

- 发布已延期：不创建或推送 `yottaapp/yotta`；继续保留本地历史与旧 origin，避免误推。
- 真正公开前需决定维持 source-available 或切换 OSI 许可证，启用 GitHub private vulnerability reporting，并观察三平台 CI 首跑。

- `PixelAt` 是否要升级为显式坐标输入/target-aware API，取代当前 Win32 鼠标 HUD 心智。
- Android 输入是否需要继续研究 minitouch/maatouch/MuMu IPC；当前先不做，ADB 通用路径已能覆盖主要用户流程。
- 前端生产构建仍有既有大 chunk / plugin timing warning，当前为非阻塞基线。
- MCP 目前固定监听 `127.0.0.1:8765` 且只用 arm 闸控制危险操作；是否再加“完全关闭服务”和端口配置，等 smoke 后决定。
- `BrowserTarget` 已按产品判断删除；底层 Browser CDP controller/client 可作为内部能力保留，但不要作为面向普通用户的节点恢复。
- 通用节点下一批候选：`WriteTextFile`、`WriteJsonFile`、`WatchFile`、Fetch cURL 导入/HTML 提取、JSON merge/map/filter。
