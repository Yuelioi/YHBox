# Cockpit — YHFish

Focus: **正在执行 Go 后端面向大型、多平台开源项目的升级方案；application lifecycle 已闭合，下一步进入 Settings/Container durability。** 本地 `main` 已分批提交，尚未推送。

## In flight

- [work/go-backend-architecture-review/](work/go-backend-architecture-review/) — **升级实施中**。批次 D 已闭合 backend 的 Win32/Wails seam、收窄真实 RPC 并定义 Linux/macOS GUI compile gate；批次 E 已闭合后台、交互、debug held-input 与 App/log 的对称关闭。下一步进入 Settings/Container durability，并等待 GUI job 首次远端运行。
- [work/type-aware-inline-node-menu/](work/type-aware-inline-node-menu/) — **已实现并验证**。pin 拖到空白时按 exec/data、方向和 pin 类型过滤可创建节点；覆盖 `number` / `bool` / `string` / `point` / `any` / `list` / `file` 全类型测试。候选列表使用 strict 兼容，只收精确类型或 `any`，不会因为 `number -> string/bool` 这类 warning 转换显示大量无关节点。exec 的 `Done` / `Fail` 等口只显示可执行节点，排除纯数据/参数转换/视觉/marker 节点；普通画布菜单保留 `CommentBox`。自动连线使用 `pinTypeCompat` 并优先精确 pin 匹配，已跑相关 Vitest 和 `pnpm typecheck`。
- [work/node-io-json-fetch-plan/](work/node-io-json-fetch-plan/) — **首批节点已实现并验证**。新增 `ReadTextFile`、`ReadJsonFile`、`ParseJSON`、`ToJSON`、`JsonPath`、`Fetch`；`JSON` pin 语义改为任意 JSON 值并保留旧 object helper；已跑 `go build ./...`、节点/目录测试、`pnpm typecheck`、`pnpm i18n:check`、`task build`。

## Open questions

- `PixelAt` 是否要升级为显式坐标输入/target-aware API，取代当前 Win32 鼠标 HUD 心智。
- Android 输入是否需要继续研究 minitouch/maatouch/MuMu IPC；当前先不做，ADB 通用路径已能覆盖主要用户流程。
- 前端生产构建仍有既有大 chunk / plugin timing warning，当前为非阻塞基线。
- MCP 目前固定监听 `127.0.0.1:8765` 且只用 arm 闸控制危险操作；是否再加“完全关闭服务”和端口配置，等 smoke 后决定。
- `BrowserTarget` 已按产品判断删除；底层 Browser CDP controller/client 可作为内部能力保留，但不要作为面向普通用户的节点恢复。
- 通用节点下一批候选：`WriteTextFile`、`WriteJsonFile`、`WatchFile`、Fetch cURL 导入/HTML 提取、JSON merge/map/filter。
