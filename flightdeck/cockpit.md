# Cockpit — YHFish

Focus: **所有既有实现 topic 已于 2026-07-04 按用户确认标记为已实现并归档；当前在扩展 Yotta 通用节点能力，首批 File / JSON / Fetch 节点已实现并通过验证。** 本地 `main` 仍待发布/推送决策。

## In flight

- [work/type-aware-inline-node-menu/](work/type-aware-inline-node-menu/) — **已实现并验证**。pin 拖到空白时按 exec/data、方向和 pin 类型过滤可创建节点；覆盖 `number` / `bool` / `string` / `point` / `any` / `list` / `file` 全类型测试。候选列表使用 strict 兼容，只收精确类型或 `any`，不会因为 `number -> string/bool` 这类 warning 转换显示大量无关节点。exec 的 `Done` / `Fail` 等口只显示可执行节点，排除纯数据/参数转换/视觉/marker 节点；普通画布菜单保留 `CommentBox`。自动连线使用 `pinTypeCompat` 并优先精确 pin 匹配，已跑相关 Vitest 和 `pnpm typecheck`。
- [work/nte-packet-capture-research/](work/nte-packet-capture-research/) — **PacketNTE 公开 API 已验证可用**。已拉取 MaaNTE / MaaNTE-Map / MaaNTE-Web / MaaNTE-PPH / PacketNTE 到 `flightdeck/references`；确认地图端只消费本机 `ws://127.0.0.1:14514` 的 `navi-state`，实际抓包接口是 PacketNTE 的 `nte_coordinate_api`。上游 `MaaNTE` 仓库不带坐标核心源码或 `thirdparty/` 产物，CI 从 `1pineappleduck/PacketNTE` release 下载 `nte_coordinate_api-v1.2.0.cp312-win_amd64.zip`；本机已下载到 ignored `flightdeck/references/MaaNTE/thirdparty`，安装 `scapy` / `pktmon-interface` 后，以 `CoordinateCapture(refresh_rate=0, capture_backend="pcap")` 成功读出 `(x, y, z, pitch, heading)` 实时样本。
- [work/node-type-market-research/](work/node-type-market-research/) — **调研完成**。调研 Unreal Blueprint / Unity Visual Scripting / Node-RED / n8n / Power Automate / UiPath / Make / ComfyUI / Blender / LabVIEW / Godot，结论是先补通用 `JSON` 语义、文件读取、JSON 路径和 HTTP 请求节点；首批已落地。
- [work/node-io-json-fetch-plan/](work/node-io-json-fetch-plan/) — **首批节点已实现并验证**。新增 `ReadTextFile`、`ReadJsonFile`、`ParseJSON`、`ToJSON`、`JsonPath`、`Fetch`；`JSON` pin 语义改为任意 JSON 值并保留旧 object helper；已跑 `go build ./...`、节点/目录测试、`pnpm typecheck`、`pnpm i18n:check`、`task build`。

## Open questions

- `PixelAt` 是否要升级为显式坐标输入/target-aware API，取代当前 Win32 鼠标 HUD 心智。
- Android 输入是否需要继续研究 minitouch/maatouch/MuMu IPC；当前先不做，ADB 通用路径已能覆盖主要用户流程。
- 前端生产构建仍有既有大 chunk / plugin timing warning，当前为非阻塞基线。
- MCP 目前固定监听 `127.0.0.1:8765` 且只用 arm 闸控制危险操作；是否再加“完全关闭服务”和端口配置，等 smoke 后决定。
- `BrowserTarget` 已按产品判断删除；底层 Browser CDP controller/client 可作为内部能力保留，但不要作为面向普通用户的节点恢复。
- 通用节点下一批候选：`WriteTextFile`、`WriteJsonFile`、`WatchFile`、Fetch cURL 导入/HTML 提取、JSON merge/map/filter。
