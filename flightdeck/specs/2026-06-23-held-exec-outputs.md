---
status: active
summary: 框架数据流改进(UE 式 held output): exec 节点 fire 时自动把出口 Data 字段存进本次运行的输出缓存(nodeID.field→值), 下游数据线可从任意距离直连读取, 免 GetVar、免紧邻约束。pullDataPin 对 exec 出口字段改读缓存; 缓存在 applyCaptures 旁填; 收编单跳 exec-data + 移除 EXEC_DATA_NOT_ADJACENT 警告; capture+GetVar 保留(命名/跨域)。触发自 AI 结构化输出 red/white 多消费的繁琐。
last_updated: 2026-06-23
graduate: true
---

# exec 节点输出可被任意下游直连 (held output, 免 GetVar)

## 0. 定位 / 触发

**这是框架数据流模型改进, 惠及所有 exec 节点(不只 AI)。** 触发自 ② AI 节点真机:
结构化输出 `red`/`white` 要给多个/远处节点消费时, 现状要么"绑变量 + GetVar"(繁琐),
要么 exec-data 单跳直连(只能紧邻下游, 跨跳运行时 `REQUIRED_FIELD_MISSING`)。用户对比
UE 蓝图: **非纯节点(带 exec)执行后输出被"存住", 任意下游直接读** —— 我们缺这个。

本 spec 让 exec 节点的出口 Data 字段也具备 UE 式 "held output" 语义: **fire 时自动存,
下游数据线任意距离直连读**, 去掉 GetVar 与紧邻约束这两道摩擦。

## 1. 目标 / 非目标

**目标(In)**
- exec 节点 fire 某出口时, 该出口携带的 **Data 字段值自动存进"本次运行输出缓存"**(键 `nodeID.field`)。
- 下游**数据线**从该 exec 出口字段拉值时, 读这个缓存 —— **任意距离**(不限紧邻), **免 GetVar**。
- 语义 = "上次该节点 fire 该字段的值"(loop 里即最新一轮; 同变量)。
- **收编单跳 exec-data**(`applyExecDataEdges`): 缓存是其超集(含紧邻), 单跳机制被替代。
- vision 图像输入(`Capture.Image → AI`)等现走单跳的, 改走缓存, 行为不变。
- **移除** `EXEC_DATA_NOT_ADJACENT` 编辑期警告(约束消失, 警告失去意义)。

**非目标(Out)**
- 不动**纯数据节点**(GetVar/Now/Expr/PureFunc)的按需重算 pull —— 那套已对(同 UE 纯函数)。
- 不删 **capture + GetVar** 机制: 仍保留给"显式命名变量 / 跨作用域(子图 RequiredGlobals) /
  可读性"场景; 它与直连读是两种正交用法(见 §8)。
- 不引入"重拉 exec 节点"语义(exec 节点永不因 pull 被重跑 —— 读已存的值, 同 UE)。
- 不改 exec **控制流**线规则。

## 2. 现状(三套机制, 本 spec 改第 3 套)

| 机制 | 谁能当数据线源 | 现状 | 本 spec |
|---|---|---|---|
| ① 纯数据 pull | `IsPureData` 节点(GetVar/Expr/Now/PureFunc) | 按需重算, 任意距离直连 ✓ | 不动 |
| ② 捕获到变量 | exec 节点产出 → `config.capture` 绑变量 → `GetVar` 读 | 任意距离, 但要手动命名变量 + 加 GetVar 节点 | 保留(命名/跨域) |
| ③ exec 节点出口 Data 字段直连 | exec 出口的 Data 字段(`Fail.Code` / AI `red`) | 仅**单跳** exec-data(紧邻下游), 跨跳运行时 `REQUIRED_FIELD_MISSING` | **改成 held 缓存(任意距离)** |

源码锚点(现状):
- `runtime/data_pull.go pullDataPin`: 数据线源是 exec 出口 Data 字段(`IsExecOutputDataFieldNode`)→ **返 nil**(line ~79), 让单跳 exec-data 兜。
- `runtime/dispatch_v5.go applyExecDataEdges`: 单跳注入(遍历有效输入列表, 从触发 token 的 exec-data 注入)。
- `runtime/dispatch_v5.go applyCaptures`: fire 时按 `config.capture` 写绑定变量。
- `container/validate.go IsExecOutputDataFieldNode / IsDataOutPinNode`: 已 config-aware(认动态输出字段)→ 数据边校验已放行 exec 出口字段当 data-out。
- `container/validator.go validateExecDataAdjacency`: 当前编辑期"跨跳"警告(本 spec 移除)。

## 3. 核心设计:per-run exec 输出缓存

**存**:`ContainerRunner` 持一个本次运行的 `execOutputs map[string]any`(键 `"<nodeID>.<field>"`)。
任何 exec 节点 fire 某出口时, 把该出口 `OutputData` 的**每个字段**写进去:
`execOutputs["<nodeID>.<field>"] = 值`。落点 = `applyCaptures` 旁(同一 fire 钩子, 已遍历出口 data)。
稀疏写: 只写该次 fire 实际带的字段(同 `applyCaptures` 语义), 未带的保留上次值。

**读**:`pullDataPin` 解析数据线源时, 若源是 exec 出口 Data 字段(`IsExecOutputDataFieldNode`):
**读 `execOutputs["<srcID>.<srcPin>"]`**(命中返值; 未命中 = 源还没 fire → 返 nil, 消费方走默认),
不再返 nil 等单跳。`coerceToType` 按消费 pin 声明类型转(同现有动态输入注入)。

**时序**:正向 exec 流里源节点先 fire(写缓存)、下游后跑(读缓存)→ 命中。fan-out / loop / 跨多跳
全覆盖(读"上次 fire 值")。

## 4. 关键决策

| 决策 | 选择 | 理由 |
|---|---|---|
| held 缓存粒度 | **per-run, 键 `nodeID.field`** | 同变量生命周期; loop 里覆盖写=最新一轮 |
| 读时机 | **pullDataPin 直读缓存**(不进 `evalDataSource` pure-data 闸) | exec 出口字段非 pure-data, 不该走重算路径; 直读已存值 |
| 单跳 exec-data | **收编/移除 `applyExecDataEdges`** | 缓存是超集; 一套机制更清晰。**plan 须验 vision(Capture.Image→AI)+ Fail.Code 等回归** |
| 紧邻警告 | **移除 `validateExecDataAdjacency` + i18n `EXEC_DATA_NOT_ADJACENT`** | 约束消失 |
| capture+GetVar | **保留** | 显式命名 / 子图 RequiredGlobals 跨域 / 可读性; 与直连读正交 |
| 重算语义 | **exec 节点永不因 pull 重跑** | 读已存值(同 UE held output); 不触发副作用重放 |
| per-tick 一致性 | **直读 live 缓存**(暂不进 per-tick snapshot) | 一次 dispatch 内源不会 re-fire, 多消费读同值; plan 验是否需 snapshot 集成 |

## 5. 实现触点(给 plan)

1. `runtime/runner.go`(ContainerRunner 结构): 加 `execOutputs map[string]any` 字段 + 每次容器 Run 起跑重置; listener 子流程共用主 runner 缓存 vs 各自一份 → plan 定(倾向主图一份; 子图调用 push frame 的可见性见 §10)。
2. `runtime/dispatch_v5.go`: fire 钩子(`applyCaptures` 旁 / `routeResult` 路由前)把出口每个 data 字段写 `execOutputs["nodeID.field"]`。
3. `runtime/data_pull.go pullDataPin`: 源是 `IsExecOutputDataFieldNode` → 读 `execOutputs`, 不返 nil。
4. `runtime/dispatch_v5.go`: **删 `applyExecDataEdges`**(及其调用)—— 被缓存替代; AI 节点收 Image / Fail.Code→Switch 等改走缓存读。
5. `container/validator.go`: **删 `validateExecDataAdjacency` + 其调用**;`frontend/src/i18n`(zh/en)删 `EXEC_DATA_NOT_ADJACENT`。
6. `data_pull.go evalDataSource` / `resolveDataPinV5`: 复核 exec 出口字段不误入 pure-data 闸(现有 `IsExecOutputDataFieldNode` 短路在 pullDataPin; resolveDataPinV5 分支确认)。

## 6. 语义细节(设计/排查按此)

- **读"上次 fire 值"**: 源没 fire 过 → 缓存无键 → nil → 消费方用自己默认(同变量没设过)。
- **稀疏**: 某次 fire 未带某字段(如 AI 走 Fail 出口不带 red)→ 不写该字段, 缓存留旧值(要区分就同时读一个 gate 字段, 同 `applyCaptures` 既有语义)。
- **loop**: 源在循环体内每轮 fire 覆盖写; 循环外消费读最后一轮值。
- **fan-out**: 源一次 fire, 多个并联下游各自读同一缓存值。
- **顺序**: 必须源先 fire 再被读(正向 exec 流天然保证); 逆序/未跑到 → nil。

## 7. 校验改动

- exec 出口 Data 字段当 data-out 源**已放行**(`IsDataOutPinNode` config-aware, 上一轮已落), 不用再改边类型校验。
- **删** `EXEC_DATA_NOT_ADJACENT` 警告(§5.5)。
- 不新增校验(held 语义下任意距离合法)。

## 8. 与现有机制的关系(三套并存)

- **纯数据 pull**(GetVar/Expr…): 不变, 按需重算。
- **直连读 exec 出口字段**(本 spec): 默认、零样板, 适合"产出 → 直接喂下游"。
- **capture + GetVar**: 仍用于 —— 显式把产出**命名成变量**(别处 `$名`/GetVar 多处复用)、**跨子图作用域**(RequiredGlobals 白名单)、图上可读性。两者正交, 不互斥; 同一字段可既直连又绑变量。

## 9. 测试策略

- **缓存读写**: 假 exec 节点 fire 出口字段 → execOutputs 写入; 下游 pullDataPin 读到。
- **跨跳直连**: `AI → log1 → log2`, `AI.red→log1.Msg`(紧邻)+ `AI.white→log2.Msg`(跨跳)→ **都拿到值**(替代旧 REQUIRED_FIELD_MISSING)。复用真机踩坑场景。
- **fan-out**: AI 扇出两并联 log, red/white 各注入。
- **loop**: 循环体内 exec 节点产出, 循环外读最新一轮。
- **回归(收编单跳)**: vision `Capture.Image → AI` 仍识图; `Fail.Code → Switch.Value`(`TestExecDataEdge_FailCodeIntoDataPin`)仍通; Expr/Script 动态输入不破。
- **稀疏**: Fail 出口不带 red → 读旧值。
- **未 fire**: 源未跑 → 消费方默认。
- **校验**: 删 `validateExecDataAdjacency` 后跨跳不再警告; i18n parity。
- 预存失败基线照 `checklists/build.md`。

## 10. 风险 / plan 待验(不脑补)

- **单跳收编**: 删 `applyExecDataEdges` 后, 所有原走单跳的(vision Image / Fail.Code)须经缓存端到端验, 别只假设。
- **子图 / push-frame 可见性**: 子图调用切表跑, exec 输出缓存键用 `nodeID` 在子图作用域是否仍唯一可读 → plan 实测; 必要时缓存随 frame。
- **per-tick 一致性**: 现读 live; 若某场景需一次 dispatch 内快照一致(同变量 snapshot)再加, 默认先不加。
- **缓存生命周期**: per-run 重置时机(容器 Run 起跑清空), listener 子流程共用 vs 独立 → plan 定。
- **内存**: 按 `nodeID.field` 累积, 上限 = 图节点产出字段数(小), 非问题。
- **跟 capture 写顺序**: fire 时既写变量(applyCaptures)又写 execOutputs, 两写独立无序依赖。
