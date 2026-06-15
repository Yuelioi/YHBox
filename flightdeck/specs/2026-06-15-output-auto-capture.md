---
status: active
summary: "输出自动捕获 + Inspector 输出组统一绑定 (Spec C)。取消'逐节点手声明 Semantic:capture 输入框'这套, 捕获绑定改存 config.capture{字段:变量名}, 前端 Inspector「输出」组统一一行式绑定 (方案 A: 按钮绑+chip, 翻译统一)。**两条写路径** (节点形态决定, 非统一): ① fire-time 自动捕获 — 出口 Data 字段值在 dispatch routeResult 由框架自动写进绑定变量 (~11 个检测/截图/脚本节点, 零节点代码); ② region per-iteration 显式捕获 — Loop/ForEach 的 Index/Item 在 RunRegion 每轮由节点调 helper 读 config.capture 写 (不经 routeResult)。模板三件套 Found 布尔补成显式 Data 字段。**消费者审计** (config.capture 是新 var-ref 站): useVarMutations 5 处 (rename/count/listUsageNodeIDs/deleteVar-cascade/listUsageRefs) + 后端 validator + referrers 全改读 config.capture。迁移条件化 + per-node 映射。边界: 不碰 vue-flow 画布/节点/连线/pin。"
last_updated: 2026-06-15
---

# 输出自动捕获 + Inspector 输出组统一绑定 (Spec C)

> 承接 Spec B (已 done 归档)。2026-06-15 brainstorm 拍定 (用户: "为什么不直接把所有出口都允许设置变量, 统一好管理" + 嫌捕获交互不优雅 + 输出速览没翻译)。**已吸收 3 份外部 AI review (ds/claude; gpt 空)**: 它们正确指出 region 循环捕获不走 OutputData (返工风险) + config.capture 消费者审计缺口; 据此本 spec 从"单一自动路径"改为"两条写路径 + 完整消费者审计"。reviewer 误判项 (留旧值=缺陷 / 纯数据该可绑 / 多用户迁移风险) 已逐条核源码驳回, 见 §附:review 处置。

## 背景 / 目标

现在"把节点产出存进变量"叫**捕获框**: 节点 Spec 逐个手声明 `Semantic:"capture"` String 输入(`CaptureCount`…), `Run()` 里手调 `node.Capture(ctx, in, 框名, 值)` 写变量(`internal/node/capture.go`, scope 固定 auto)。三毛病(用户提):
1. **不统一/漏声明**: 逐节点手工活 → PlayClip 的 `Fail` 出口带 `Error`/`Code` Data 字段但**没声明捕获框** → 绑不了(违反 [node-data-flow](../checklists/2026-06-05-node-data-flow.md) 硬约束)。
2. **没翻译/费解**: Part 2「输出速览」取 `pinsFor().dataOut` 原始 pin 名(Count/Center); 捕获框取翻译 label(命中像素数)→ 一物两名。
3. **交互不优雅**: 捕获框 = 每个"标题+文本框+hint", 占地大。

**目标**: 捕获绑定不再靠手声明的输入框, 改 `config.capture` 存储 + 前端「输出」组统一一行式绑定; 出口 Data 产出由框架**自动捕获**(零节点代码), 覆盖现在漏声明的(PlayClip)。

## 核心机制 (读源码确认)

- 捕获助手 `node.Capture(ctx, in, captureField, value)`(`internal/node/capture.go`): trim 非空则 `ctx.Vars().SetScoped(name, "auto", value)`。
- 出口 fire 带数据 `ctx.Out("exit").Set("字段", 值).Fire()` → `Outputs{exitName, data}`(`outputs.go`)。
- dispatch `routeResult`(`dispatch_v5.go` L182)有 `result.ExitName` + `result.OutputData`(fire 出口 data map), L262 喂 exec-data 连线 `nextWithData(node.ID+"."+ExitName, ..., OutputData)`。
- 前端 `adapter.ts splitOutputs`(L133)把"exec 出口携带 Data 字段"flatten 进 `dataOut` → `getSpec(kind).dataOut` 含全部 Data 字段(PlayClip.Error/Code 在内)。
- **关键差异 (reviewer 验出, 已读源码确认)**: `internal/nodes/control/loop.go` / `foreach.go` 的 `CaptureIndex`/`CaptureItem` 在 `RunRegion` 里**每轮迭代** `node.Capture(ctx, in, ..., i/el)`, 值**不是任何出口的 Data 字段**, 也**不经 routeResult**(RunRegion 直接调 `body()`, Body 出口不逐轮 route)。→ 不能用 OutputData 路径捕获。

## 两条写路径 (节点形态决定, 不是设计随意分叉)

| 路径 | 适用 | 写在哪 | 节点代码 |
|---|---|---|---|
| **① fire-time 自动** | 出口 Data 字段(检测/模板/截图/脚本/秒表…约 11 节点) | 框架在 `routeResult`: 对 `config.capture` 里绑了且 `字段 ∈ result.OutputData` 的字段 → `SetScoped(var,"auto",OutputData[字段])` | **零** (删 node.Capture 调用) |
| **② region per-iteration** | Loop(`Index`)/ForEach(`Item`/`Index`)循环迭代值 | 节点在 RunRegion 每轮调 helper `node.CaptureOutput(ctx, "字段", 值)`(读 `config.capture[字段]` 写变量) | 保留每轮一行显式调用 (只 2 节点; 唯节点知道每轮值/时机) |

> 用户层仍**统一**: 两路都绑 `config.capture` + 同一前端「输出」组绑定 UI。差异只在后端写时机, 对用户不可见。`node.Capture` 不全删 —— 重命名/改造成 `CaptureOutput`(读 `config.capture` 而非旧捕获输入 pin), 仅 region 节点保留调用; 11 个 fire-time 节点的调用删掉、框架接管。

## ⛔ 核心不变量 (实现第一步逐节点核对, 不是写完再补)

`grep -rn 'Semantic: "capture"' internal/nodes` = **13 文件 27 点**。逐点归类 + 处理:
- **fire-time 类**: 被捕获值**必须**作为出口 Data 字段被 `.Set()`(框架从 OutputData 读)。逐点确认; 缺的补 Data 字段。**已知要补**: 模板三件套 `Found` 布尔(`wait_template`/`check_template`/`click_template`)—— 它不是 Data 字段, "命中与否"靠走哪个出口编码。**补法**: 加 `Found`(`Bool`)Data 字段, **命中出口 Set(Found,true)、未命中/超时出口 Set(Found,false), 两出口都带**。
- **region 类**: Loop/ForEach 的 Index/Item → 走路径 ②, 不要求是 Data 字段。前端为能列出它们绑定, 在节点 Spec 用 Body 出口的 Data 字段声明(`Loop.Body: Index`; `ForEach.Body: Item, Index`)作为**列举元数据**(前端 dataOut 统一读到), 实际写由路径 ② 每轮 helper 完成。
- 其余检测/截图/脚本/秒表节点的捕获多是真数据值(Count/Center/Point/Conf/Ratio/Clusters/Path/Elapsed/Result/Inner*X/Outer*X/Px…), 大概率已 `Set`, **仍须逐个核**。

## 设计决策

### 1. 后端

- **存储**: 节点 `config.capture: map[string]string`(`字段名 → 变量名`), 跟 `config.literal` 平级。空/缺省=无绑定。
- **路径 ① (routeResult hook)**: 拿到 `result.ExitName`+`result.OutputData` 后, 读 `config.capture`, 对每个 `字段→变量` 且 `字段 ∈ OutputData`(本次 fire 出口实际带该字段, **稀疏**: OutputData 只含节点 `.Set()` 过的字段, 不是全字段补 null)→ `SetScoped(变量,"auto",值)`。**只在成功 route 路径写; error 分支(节点返 err)不 route 也不写**。
- **路径 ② (region helper)**: `node.CaptureOutput(ctx, field, value)` 读 `config.capture[field]`, 非空则 `SetScoped`。Loop/ForEach RunRegion 每轮调(替原 `node.Capture(ctx,in,"CaptureIndex",i)`)。
- **删干净**(二号铁律): 移除 13 文件全部 `Semantic:"capture"` 输入 pin 声明; 删 11 个 fire-time 节点的 `node.Capture()` 调用; `node.Capture`(读输入 pin 版)删, 留/改 `node.CaptureOutput`(读 config.capture 版)。`spec_capture_test.go`(CaptureType 校验框)随框删而调整; `InputSpec.CaptureType` 若无消费者一并删(`grep CaptureType` 核)。
- **写/注入顺序无依赖**: 自动捕获写变量 与 exec-data 连线注入下游 是**两条独立 channel**(写 VarStore vs 注入下游 input), 顺序不影响正确性。`SetScoped` 本身不失败(只写 map); 变量名合法性在编辑/校验期查, 非运行期 → 无运行期写失败路径。

### 2. 消费者审计 (config.capture 是新 var-ref 站 — review 验出的真缺口, 必做)

config.capture 引用变量名。**现有** `frontend/src/composables/containerEditor/useVarMutations.ts` 已把捕获绑定当 var-ref 消费者(`captureFieldsOf(kind)` 从 `NODE_FIELD_SCHEMAS` semantic==='capture' 派生, 读 `config.literal[field]`), 被 5 处用: `renameVar` / `countUsage` / `listUsageNodeIDs` / `deleteVar`(cascade 清空) / `listUsageRefs`(FindReferences)。改存储后**这 5 处全要改**: "可绑字段"来源从 NODE_FIELD_SCHEMAS-capture 改成节点 dataOut(可绑字段); "绑定值"从 `config.literal[field]` 改成 `config.capture[field]`。漏改 = 删变量不清绑定(悬空)/重命名漏改/查引用漏列 —— 正是 incident [[2026-05-29-storage-convention-consumer-audit-gap]] 的坑。
- 后端校验: `config.capture` 变量名走变量引用校验(`validator_var_refs.go` 类, 码 `INVALID_VAR_REF`; scope auto 同捕获旧语义); 字段名须是该 kind 合法可绑字段。
- 后端 referrers(`backend.subgraphs.referrers`/var 引用扫描)若扫捕获绑定, 同步改读 config.capture。

### 3. 前端 · Inspector「输出」组统一 (方案 A: 按钮绑 + chip)

合并 Part 2「输出速览(只读)」+「输出捕获(VarNameInput)」两套 → 一个列表(`NodeInspector.vue`):
- **exec 出口**(完成/失败/命中…): 顶部只读参考行, 不可绑。
- **可绑产出**(`非 IsPureData` 节点的 dataOut 字段, 含 region 节点 Body 声明的 Index/Item): 每行 `翻译名 类型 [+绑变量]`; 绑了显 `→ $var ✕`。点「+绑变量」走现有 `VarNameInput`(复用其现有: 选现有变量 / 现场新建→emit declare-var→draft.vars 立即入草稿、保存时落盘; 重名走 `validateVarName`; 捕获 scope 固定 auto)。新建变量类型取该字段类型(Number/Point/Bool/String/any; list 不在白名单, 维持)。写回 `config.capture[字段]`。
- **纯数据节点**(`IsPureData`: GetVar/Now/VarLastChange/Expr/PureFunc): 输出是连线源, 只读参考行, **不可绑**。理由(reviewer 质疑过): 纯数据节点是 Evaluator, **无 exec fire / 无 routeResult hook**, 机制上无处写; "把取出的变量再存到另一变量"用 `SetVar` 节点(本就该这么做), 非捕获场景。
- 删 `NodeInspector` 现 `captureLiterals`/`captureOpen`/折叠头(9d66558 那段)→ 换统一列表; `outPins` 速览并入。
- **Found 字段语义提示**: Found 两出口都 Set(始终反映本次命中态), 其余数据字段仅命中出口 Set(未命中留旧值, 靠 Found/Count gate)。这是**有意**的(node-data-flow 既定: 未命中不写、留旧值), 前端 hint 一句说明 Found 是"本次是否命中", 别让用户以为它跟 Center 同语义。

### 4. 翻译

Data 字段加 i18n label `node.<kind>.output.<字段>`(如 `node.DetectColor.output.Count`="命中像素数"); 旧捕获框中文 label 搬过来复用, 删旧捕获 i18n 键。输出行 label 走这套, 不显原始英文 pin 名。

### 5. 迁移 (条件化 + per-node 映射)

- 本项目**单机单用户**(deck: 单机使用; CLAUDE.md: 未发布无外部用户)→ grep dev 机容器存档 = **全量覆盖**, 不存在"漏掉别的用户工作流"。
- 实现时 grep 现有存档有无非空 `config.literal["Capture<X>"]`。**有** → 迁移前**先备份容器目录**(或确认 git 可回滚), 写一次性脚本搬到 `config.capture`; **映射 per-node 显式**(不假设统一"去 Capture 前缀": 用每个节点已知的 `捕获输入名→Data字段名` 对照表, e.g. detect_color `CaptureCount→Count`/`CaptureCenter→Center`; 混合情况=逐字段搬, 部分绑定天然支持)。跑完删旧键。**没有**(用户印象里没绑过)→ 跳过。

## 边界与风险

- **边界**: 不碰 vue-flow 画布/`ContainerFlowNode`/连线/pin/右键/inline(沿用 Spec B 边界)。绑定全在 Inspector。画布侧 Data 字段仍是"画得出连不上"footgun(本 spec 不解决画布, 只给 Inspector 绑定路径)。
- **风险 1 — dispatch 热路径 + 性能**: routeResult 每次 fire 调一次。新增 = 读 `config.capture`(内存小 map)+ 遍历绑定字段(通常 0–2)+ SetScoped。`O(绑定数)`, 相对节点 Run(视觉/输入)开销可忽略。**不需** benchmark; 若日后循环内高频出口顾虑大再加。
- **风险 2 — 并发态势不变**: 自动捕获写发生在 dispatch(跟现 `node.Capture`(在 Run 内、由 dispatch 调)**同写、同上下文**), **不引入新并发**。多分支(Parallel/Race)并发写同名变量的隐患是**既有**的, 已有前端 `useConcurrencyWarning` 警告, 继续适用; `VarStore.SetScoped` 线程安全性沿用现状(本 spec 不改写其同步语义)。
- **风险 3 — 不变量漏核 / region 路径**: 见 §核心不变量。两条路径已分清; 实现第一步逐节点核, 漏一个 = 该产出绑了不写。
- **风险 4 — capture 节点测试**: 13 文件改 Spec → 各 `*_test.go` 里"捕获写入"断言改成"绑 config.capture + 验写入"(fire-time 走 dispatch 测; region 走 RunRegion 测)。
- **风险 5 — 消费者审计漏改** (见 §2): useVarMutations 5 处 + 校验 + referrers, 漏一处即悬空/漏列。

## 验证

- **后端** `go test ./internal/...`: ① 自动捕获 dispatch 单测(绑定字段 fire 后变量被写 / **未带字段不写、变量留旧值**(reviewer 关注的负面用例)/ error 不写 / 多字段); ② region 单测(Loop 每轮 Index 写入递增 / ForEach Item+Index 逐轮); ③ 13 节点测试改写; ④ 模板 Found 双出口写 true/false。
- **前端** `pnpm typecheck`/`pnpm test`(useVarMutations 改后 rename/count/delete-cascade/find-refs **对 config.capture** 的单测 — 消费者审计回归)/`pnpm build:dev`; 禁用色扫描零行。
- **真机 smoke**: 给 ① DetectColor(命中像素数/中心点)② PlayClip(Error/Code, 旧绑不了的)③ Loop(Index)各绑变量 → 跑 → GetVar/日志确认写入正确值; 未命中出口不覆盖旧值; 删一个被捕获绑定的变量 → 确认绑定被清(不悬空); 翻译名一致。

## 实现分期 (plan 切分建议)

- **Part 1 · 后端**: config.capture + routeResult 自动捕获(路径①) + CaptureOutput helper(路径②, Loop/ForEach) + 不变量核对/补 Data 字段(Found 等) + 删 13 文件捕获框 + 校验 + 后端测试。(结构大头/风险集中, 先做先验。)
- **Part 2 · 前端 + 消费者审计**: 方案 A 输出组 UI + 翻译 i18n + useVarMutations 5 处改读 config.capture + FindReferences + 删旧 captureLiterals 那套 + 前端测试。
- **Part 3 · 迁移(条件)+ 收尾真机 smoke**。

(Part 1 必须先于 Part 2 真机验; 消费者审计(Part 2)与后端校验(Part 1)对 config.capture 形态须一致。)

## 附: review 处置 (3 份外部 AI, 逐条核源码)

**接受 (改了 spec)**:
- *region 捕获不走 OutputData*(ds#2/claude#1): **真**。读 loop.go/foreach.go 证实每轮 `node.Capture` 非 Data 字段 → 改成两条写路径。(ds 说的具体 API `ctx.SetReturn` 不存在 — 实为 `node.Capture` 每轮调; substance 对、细节错。)
- *config.capture 消费者审计*(claude#3): **真**。读 useVarMutations 证实 5 处消费捕获绑定 → §2 列为必做。
- *Found 双出口语义不一致*(claude#2): 接受为**澄清** — Found 始终写(反映本次命中), 其余留旧值; §3 加 hint 说明。
- *迁移别假设统一前缀剥离 / 混合情况*(claude#4): 接受 → §5 改 per-node 映射 + 备份。
- *OutputData 稀疏语义 / 写入顺序 / SetScoped 失败*(ds#5/claude#5): 接受为澄清 → §1 写明稀疏、双 channel 无序依赖、无运行期写失败。

**驳回 (reviewer 不懂项目既有语义/全貌, 已核源码)**:
- *"未命中留旧值"是缺陷, 应清空*(ds#1/#9): **驳**。这是 node-data-flow 既定**有意**语义(留旧值 + 靠 Found/Count gate), 非缺陷; 改成"清空"反而是行为回归。负面用例(未命中不覆盖)已入验证。
- *纯数据节点该可绑*(ds#3): **驳**。Evaluator 无 fire/routeResult hook, 机制上无处写; "再存一份"用 SetVar 节点。§3 已注明理由。
- *迁移高风险无回滚/覆盖不全*(ds#4): **驳前提**。单机单用户, grep=全量覆盖; git 可回滚 + 迁移前备份(§5)已够, 不需企业级回滚方案。
- *性能需 benchmark*(ds#7): **降级**。O(绑定数)可忽略, 不需 benchmark(§风险1)。
- *并发安全未讨论*(ds#8): **澄清非新风险**。同写同上下文, 态势不变, 既有并发警告适用(§风险2)。
- *VarNameInput 新建时机/作用域/重名未定义*(ds#6): **复用既有**。VarNameInput 现成行为(declare-var→草稿、保存落盘; validateVarName 重名; scope auto), 非新决策(§3 已注)。
