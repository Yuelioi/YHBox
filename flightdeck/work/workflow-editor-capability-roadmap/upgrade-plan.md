# Yotta 3.1 产品升级执行计划

## 完成定义

“3.1 major upgrade 完成”必须同时满足：

- 旧产品能力已按入口、管理、创作、运行四层完成机械对比。
- Windows 与 Android 的安装、创作和运行闭环完整；未来 macOS 只新增 Adapter。
- 工作流可安全导入导出，资产库可随规模增长管理。
- Source 的多 graph 结构拥有真实 subgraph 调用/创作语义，comment/reroute 不依赖旧 Container 模型。
- Browser CDP 在 installation、Settings、policy、provider/admission、Catalog 和健康诊断完整后声明支持。
- 所有阶段按批量验收原则留下可信 build、test 与宿主 smoke 证据。

2026-07-18 架构审计确认以上定义**尚未满足**；此前完成判定已撤销。执行内核升级已形成，但产品连续性、外围模块边界和真实宿主纵向验收仍需按 Slice 27 恢复。

## 执行原则

- 一个 Stage 包含多个相邻 Slice；Slice 与本地 commit 只作为实现、恢复和回滚边界。
- Slice 内仅运行支撑继续开发的定向测试、typecheck 或 compile。
- Stage 完成后统一执行相关聚合测试、`task check`、`task build` 和触发的真实宿主 smoke。
- 保存等原地动作成功不 toast；失败使用统一 Nuxt UI。
- 不恢复旧 Container runtime，不创建第二套 Workflow Source/compiler/runtime。
- 不恢复任意宿主 JS/Wails/yt console；脚本只能走 capability-admitted 隔离执行。
- 3.0 历史 Knowledge 在恢复期间保留并明确标记，R5 通过后统一 promotion、归档或删除，不长期混在主动 Knowledge 中。

## 当前执行路线

| Recovery Stage | Slices | 交付边界 | Stage gate |
| --- | --- | --- | --- |
| R0 事实重置 | 29 | 固定 3.0 oracle、capability ledger、G01–G17、dirty ownership、历史 Knowledge 退役 registry | 文档/链接/Flightdeck 校验；不跑产品门禁 |
| R1 外围深模块 | 30–33 | Typed RPC、Installation Manifest/Target Runtime、Recording Session、Asset Picker Query | contract/conformance + 相关聚合测试 + `task check`；形成一个阶段 commit |
| R2 Windows 闭环 | 34 | UAC、F9、exact/regex、多窗口、键鼠、窗口、截图、模板、录制/回放 | G02–G08/G11/G13/G15 + production build + Windows native smoke；一个 commit |
| R3 创作能力 | 35 | 搜索式 picker、typed State/转换、编辑器基础旅程、P0/P1 节点族 | G01/G06/G09–G12 + editor/integration + 人工 UX；一个 commit |
| R4 平台连续性 | 36 | Android/ADB、Browser CDP、macOS seam proof | G16/G17 + emulator/browser smoke + cross-platform compile；一个 commit |
| R5 发布 | 37 | 全矩阵、规模、生命周期、workspace 恢复与 3.0 Knowledge 退役 | G01–G17、完整 `task check`/build/native matrix、用户验收 |

当前阶段：R1 已于 2026-07-18 完成并形成统一 `task check` 证据；下一阶段是 R2 / Slice 34，不能用 R1 contract 结果替代 Windows native smoke。

具体 completion、blocked-by、verification 和 out-of-scope 只在对应 Slice 维护；[Slice 27](slices/27-architecture-recovery.md) 是恢复设计 umbrella，[Slice registry](slices/map.md) 是完整状态表。

## 历史实现证据

Slices 1–26 只证明相应组件和历史批次曾实现，不能直接作为 3.1 发布证据。它们的可复用实现必须重新映射到 [capability ledger](artifacts/capability-ledger.md)，并通过对应 [golden journeys](artifacts/golden-journeys.md)。

## 发布阈值

本计划定义的 3.1 major upgrade 发布阈值**尚未满足**。后续以 capability ledger、G01–G17 和 Slice 37 的真实宿主矩阵为唯一完成依据；签名、安装包和 license 表述仍属于发布工程，但不得替代产品能力验收。

