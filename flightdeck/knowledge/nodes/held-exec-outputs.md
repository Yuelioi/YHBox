---
kind: note
summary: "历史 3.0 ContainerRunner held-output 缓存；3.1 使用 Program data plan 与 activation-scoped push output。"
activation: action
read_when: "仅在审查 3.0 ContainerRunner.execOutputs，或对比 3.1 activation-scoped data output 时"
recheck_when: "改 ContainerRunner.execOutputs 键格式/生命周期 / routeResult 的写钩子 / pullDataPin 对 exec 出口字段的读路径 / IsExecOutputDataFieldNode 判定 / 重新引入紧邻约束时"
---
# held exec output — exec 出口 Data 字段任意距离直连（免 GetVar）

> 历史知识：`ContainerRunner.execOutputs`、GetVar 与旧 exec DataField 已删除。当前规则见 `nodes/node-data-flow.md`，不得把全局 held cache 回接到 3.1。
exec 节点出口携带的 **Data 字段**（`Fail.Code`/`Fail.Error`、AI 结构化输出声明的 `red`/`white`、`Capture.Image` 等）具备 **UE 式 "held output" 语义**：节点 fire 某出口时该出口的每个 Data 字段自动存进本次运行的缓存，下游**数据线可从任意距离直连读**，不需要 GetVar、不要求源是紧邻 exec 上游。

## 机制（写 → 读）

- **缓存**：`ContainerRunner.execOutputs map[string]any`，键 `"<nodeID>.<field>"`，per-run 生命周期（`NewContainerRunner` 初始化，主图/子图/listener 子流程共用一张；子图调用切 `nodesByID/dataEdges` 表但**不切 runner**，故缓存跨 frame 持续）。
- **写**：`routeResult`（`dispatch_v5.go`）在 fire 出口时调 `captureExecOutputs(node, data)`，把出口 OutputData 每个字段写进缓存。两处接入：成功出口（`result.OutputData`）+ 失败出口（`failData{Error,Code}`），紧挨各自的 `applyCaptures` 调用。**稀疏写**：只写本次 fire 实际带的字段，未带的保留上次值（同 `applyCaptures` 语义）。
- **读**：`pullDataPin`（`data_pull.go`）解析数据线源时，若源是 exec 出口 Data 字段（`container.IsExecOutputDataFieldNode`）→ 读 `execOutputs["<srcID>.<srcPin>"]`：命中返值（`coerceToType` 按消费 pin 类型转），未命中（源还没 fire）返 `nil` → 消费方走自己的默认。**exec 节点永不因 pull 被重跑**——只读已存值，不触发副作用重放。

键全局唯一靠 node ID = `kind_<6位base36随机>`（`frontend/src/composables/containerEditor/ids.ts`），容器内不碰撞，故 flat 共享 map 安全、不需 per-frame 命名空间。

## 语义

| 场景 | 行为 |
|---|---|
| 跨跳 | 源先 fire（写缓存），任意距离的下游后跑（读缓存）→ 命中 |
| fan-out | 源一次 fire，多个并联下游各自读同一缓存值 |
| loop | 循环体内每轮 fire 覆盖写，循环外读最后一轮值 |
| 稀疏 | 某次 fire 未带某字段 → 缓存留旧值（要区分就同时读一个 gate 字段） |
| 未 fire | 缓存无键 → `nil` → 消费方默认 |

## 三套数据线来源机制（并存、正交）

1. **纯数据 pull**（GetVar/Expr/Now/22 purefunc）：`IsPureData` 源，按需重算，任意距离。不变。
2. **held output 直连**（本机制）：默认、零样板，适合「产出 → 直接喂下游」。
3. **capture + GetVar**：仍保留——显式把产出**命名成变量**（多处 `$名`/GetVar 复用）、**跨子图作用域**（RequiredGlobals 白名单）、图上可读性。与直连读不互斥；同一字段可既直连又绑变量。

另有一条正交路径：`token.ExecData` 直传 `RunNode` 的 `execData` 参数，走 `in.X("k")` **原始 key-match**（消费 pin 名 == 源字段名时，无需数据线）——held output 不替代它（priority `dataWire > config > execData > default`，见 `internal/node/inputs.go`）。

## 历史（破坏性切净）

本机制收编了旧的**单跳 exec-data 数据线桥** `applyExecDataEdges`（只能紧邻下游，跨跳运行时 `REQUIRED_FIELD_MISSING`）——缓存是其超集，故 `applyExecDataEdges` 连同编辑期 `EXEC_DATA_NOT_ADJACENT` 警告（`validateExecDataAdjacency`）一并删除，约束消失。历史实现记录在 cold archive `2026-06-23-held-exec-outputs`;当前规则以本知识和源码为准。
