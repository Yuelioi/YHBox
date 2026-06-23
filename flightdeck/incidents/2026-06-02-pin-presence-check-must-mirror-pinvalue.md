---
when_to_read: 写/改任何读或判定节点 pin 值的容器侧逻辑(validator / 扫描器 / 迁移 / 导出);加"必填缺失/未设值"类静态校验;撞"validator 报某 pin 没值但 runtime 明明能跑"
applies_to: [container, validator, pin-value, config-literal, node-framework, false-positive]
last_updated: 2026-06-02
status: active
---

# pin 值的存在性判定必须镜像 PinValue 的两级回退

**现象**: 给容器加 `MISSING_REQUIRED_PIN` 静态校验时,只查了 `config["literal"][pin]`。结果对真实 fishing-v2 容器的**每个** Subgraph 调用节点误报 —— 它们把 `SubgraphID` 存在**顶层** `config["SubgraphID"]`(不是 `config.literal` 下)。validator 说"没值",但 runtime 跑得好好的,两者对 pin 是否有值的判定不一致。

**根因**: pin 值的正源读取走 `container.PinValue(n, name)`(`pin_value.go`),它是**两级**的:先 `config["literal"][name]`,**取不到再回退顶层 `config[name]`**。runtime 的 `pullDataPin`、`PinString`/`PinFloat`/`PinStringList`、editor 全走这条。任何只看其中一级的"有没有值"判定都会跟正源脱节 → 对用另一级存值的容器误报/漏判。

**怎么做**:
- 判定"pin 有没有值来源"时,有值 = **连入 data 边 ∨ `config.literal` 有该 key ∨ 顶层 `config` 有该 key ∨ Spec 有非 nil Default**。四者任一即满足。
- 别假设"画布只写 config.literal":约定上 literal 是正源(见 [[storage-convention-consumer-audit-gap]]),但**历史/真实容器**(fishing-v2)把值存在顶层 config,且 runtime 接受。校验是跑在所有现存容器上的共享设施,必须容两种。
- 验证别只跑单测:**拿真实容器跑**(`go run ./cmd/validate-fishing-v2`),误报一眼现形。本坑就是 single-test 绿、真实容器红。
- 反过来,只针对 `config.literal` 子 map 的检查(如 `UNKNOWN_LITERAL_PIN` 查未知 literal key)是有意只看一级 —— 那是检查 literal 内容本身,不是判定"有没有值",别混。

**相关**: [[storage-convention-consumer-audit-gap]](改 pin 值读写约定前 exhaustive grep 全消费者);本质同源 —— 动 pin 值的读法前,先找全"正源到底怎么读"。
