---
slice: "15"
title: 高级能力决策与迁移收口
status: completed
---

# Slice 15：高级能力决策与迁移收口

## Outcome / Question

把旧版仍未迁移的高级能力逐项收口为恢复、现有替代、明确延期或不恢复，消除“后端还在但产品不知道是否保留”的灰区，并形成诚实的 3.1 发布边界。

## Completion criterion

- Workflow Source/package import/export：决定安全格式、冲突/身份/版本语义及是否进入 3.1 发布。
- 资产批量选择、分页、variant 维护与清理：按 immutable BlobRef 和引用安全决定恢复范围。
- 画布节点定位与命令面板：决定恢复为现有 Source 内搜索/聚焦还是由目录搜索替代。
- subgraph authoring、comment、reroute：分别决定恢复、延期或不恢复，不把旧 Container 格式直接复制进 Source。
- snippet、任意表达式与 yt console：明确安全替代或不恢复，不扩大不可信代码执行面。
- Browser CDP：决定是否形成安装目标 Slice；controller 存在不等于产品支持。
- 更新 capability audit、升级计划、release 边界与 topic 状态；不得保留“以后再说”。

## Blocked by

Resolved：Stage 6 已完成。

## Verification

Stage 7 的 Source-native 节点定位实现后，整阶段统一通过 task check、正式 task build 与 Windows WebView smoke；145 个前端测试、1595 个 i18n keys、14/106/134 Wails 契约、65.4% Go 总覆盖均通过。smoke 确认 104 个目录节点、3 个画布节点，截图已人工检查。

## Out of scope

复制旧 Container runtime、恢复任意宿主 JS/yt console、以已有 controller 或隐藏后端冒充可用产品能力。

## Final decisions

| 能力 | 旧版事实 | 3.1 决策 | 发布影响 |
| --- | --- | --- | --- |
| Workflow import/export/package | 旧版只有 Container zip 导出，没有 ImportPackage 闭环；包排除 installation | 3.1 不恢复旧 Container zip，也不谎称能迁移。canonical Source portability 另做独立 post-3.1 工作包：只打包 source、manifest 与 exact referenced blobs；禁止 secrets/installations/executable payload。导入必须限额、hash/schema/NodeRef 校验、私有 staging 原子发布，默认分配新 workflow ID；显式覆盖才允许 CAS。 | 3.1 release notes 明示当前无导入导出；不是现有运行闭环阻断 |
| 资产批量/分页/维护 | 旧版有分页/选择条；所谓 Maintenance 实际只清理旧 subgraph | 现有搜索、单删、meta、variant 增删已保留。批量元数据/删除与 QueryAssets 分页列入独立 post-3.1 规模化工作包。不得把删 metadata 当 Blob GC；GC 只有在能枚举 Source、Run、package 等全部 BlobRef roots 后才设计。 | 延期，P2；不影响 exact BlobRef 运行 |
| 画布节点定位 | 旧版 Ctrl+F NodeSearchModal 可跨主图/子图定位 | 已恢复为 Source-native 搜索：工具栏入口、Ctrl/⌘+F、名称/ID/NodeRef/graph 搜索，选择后切图、选中并居中。 | 已恢复 |
| 通用命令面板 | 旧版聚合现有编辑/视图动作 | 不单独恢复旧 registry；3.1 的工具栏、选择工具条、目录搜索与快捷键已覆盖实际动作。未来只有动作数量再次失控时才增加由同一 command handler 驱动的 palette。 | 由现有入口替代 |
| subgraph authoring | Source schema 有 graph 结构，但 compiler 明确只接受唯一 main entry graph | 不冒充已支持。独立 post-3.1 深模块工作包必须同时定义 call-node contract、graph ports、递归/深度预算、compiler/program/journal graph path 与创作 UI，不能只解锁隐藏字段。 | 延期，P3 |
| comment | 旧版用画布节点表达注释 | 不作为可执行 Node 恢复；未来以 authoring-only annotation projection 设计，不能进入 scheduler 或 capability。 | 延期，P3 |
| reroute | 旧版作为图编辑辅助 | 3.1 不恢复为 Source 语义节点；如以后需要，只能作为 edge presentation metadata，自动布局仍忽略它。 | 明确不恢复旧模型 |
| snippet/任意表达式/yt console | 可执行旧表达式或宿主脚本，安全边界与 3.1 不兼容 | 明确不恢复任意宿主 JS/Wails/yt console。现有 typed nodes、变量和 Node Package 覆盖确定性逻辑；未来脚本只能是有资源预算、capability admission、隔离 worker、确定输入输出契约的 sandboxed Script Node。 | 不恢复旧入口 |
| Browser CDP | controller/discovery/client 存在，但没有 installation、Settings、policy、provider/admission 或 Catalog 产品链 | 3.1 不声明支持。另做实验性 post-3.1 Adapter 工作包：exact loopback endpoint + page identity、健康/漂移、consent、operation set、target resolution 和浏览器启动责任必须完整后才出现入口。 | 延期；controller 仅为底层资产 |

## Result

Completed。唯一应立即恢复的日常回归“画布节点定位”已接入 3.1 Source；其余项目均有明确替代、延期边界或不恢复理由。3.1 major upgrade 的既定 Stage 1–7 已执行完毕，但 release notes 必须诚实列出 Source portability、资产规模化、subgraph authoring 与 Browser CDP 尚未作为产品能力交付。
