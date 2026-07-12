# Cockpit — Yotta

Focus: **实施 Yotta 3.0 AI-native 大型开源破坏性升级。** Wave 1 已完成实现与双轴 review；下一工程入口是 Wave 2 工具链/供应链 pin，同时等待 OSI/identity/source-open 决策。

## In flight

- `major-upgrade-review` — 方案与 Wave 1 完成：format、agent contract、task check、CI/lint/coverage/race/fuzz、三平台 compile、bindings contract、bundle gate、ELK 与 icon-index lazy split；进入 Wave 2。
- `recording-asset-lifecycle` — 实现与自动化验证已完成；用 `bin/Yotta.exe` 验收三条用户路径后归档。

## Open questions

- 发布继续延期：未明确授权前不创建或推送 `yottaapp/yotta`；先决定 OSI 许可证、canonical identity 与本地领先历史的安全公开方式。
- 真正公开前必须启用并验证 main/tag rulesets、第二管理员、private vulnerability reporting、完整 required checks 与可信 release 链。
- Yotta 3.0 已决定 MCP 默认关闭、优先 stdio；HTTP 只允许 authenticated loopback。旧的固定 `127.0.0.1:8765` + arm 方案不再是目标设计。

- `PixelAt` 是否要升级为显式坐标输入/target-aware API，取代当前 Win32 鼠标 HUD 心智。
- Android 输入是否需要继续研究 minitouch/maatouch/MuMu IPC；当前先不做，ADB 通用路径已能覆盖主要用户流程。
- 前端生产构建仍有既有大 chunk / plugin timing warning，当前为非阻塞基线。
- `BrowserTarget` 已按产品判断删除；底层 Browser CDP controller/client 可作为内部能力保留，但不要作为面向普通用户的节点恢复。
- 通用节点下一批候选：`WriteTextFile`、`WriteJsonFile`、`WatchFile`、Fetch cURL 导入/HTML 提取、JSON merge/map/filter。
