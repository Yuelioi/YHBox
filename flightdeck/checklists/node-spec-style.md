---
status: active
last_updated: 2026-05-25
when_to_read: before writing / editing 任何 nodepkg.Spec (`internal/nodes/**/*.go`) — 含 Inputs/Outputs/Default
applies_to: [node-spec, pin-naming, convention, go]
when_to_update: 改 nodepkg.Spec 的 pin 命名约定 / Inputs·Outputs·Default 结构时
---
# Node Spec Style Playbook

写/改任何 `internal/nodes/**/*.go` 节点 Spec 前**前置**读这份. P2.1 锁定的统一约定 — 防止 4 套混 (PascalCase / lowercase / 老 vs 新) 再次漂移.

## 1. Pin Name 大小写

| 类别                                                                                                      | 规则                           | 例子                                                                                                     |
| --------------------------------------------------------------------------------------------------------- | ------------------------------ | -------------------------------------------------------------------------------------------------------- |
| **User-facing data pin** (Inspector 可见 / JSON 序列化的 config key)                                | **PascalCase**           | `Template`, `VarName`, `Scope`, `Duration`, `Threshold`                                        |
| **Framework exec in pin** (单 in 入口)                                                              | 常量 `In` = `"In"`         | `{Name: pinIn, Type: TypeExec}`                                                                        |
| **Framework exec out pin** (单 done 出口)                                                           | 常量 `Done` = `"Done"`     | `{Name: pinDone, Type: TypeExec}` (显示文案走 i18n `output.Done.label`, 后端无 `DisplayName` 字段) |
| **多 exec out 出口** (语义专选: Loop body/done, Try out/catch, If True/False, Switch CaseN/Default) | **PascalCase, 语义命名** | `Body`/`Done`, `Out`/`Catch`, `True`/`False`, `Case1`..`Case16`/`Default`              |
| **数据 out pin** (purefunc.result, GetVar.value 等)                                                 | **PascalCase**           | `Result`, `Value`, `Point`, `Confidence`                                                         |

**⚠ 修正 (2026-06-08, 实证 `frontend/.../pinSpec.ts:84`)**: exec-out **一律 PascalCase**, 入口/fire-only 节点也不例外 — Start 用 `"Done"`, EventTick 用 `"Out"`, 子图入口 marker `SubgraphInput` 用 `"Done"`. **绝不要用小写 `"out"`**: 前端 `pinSpec.ts` 明写"用 'out' 会让边渲染错位 + 运行时不播种" (runtime 从 `entryID+".Done"` 播种, 主图从 `start.Done` 播种). 旧版本这里写的"fire-only 用小写 out"是**错的, 已废** — 照它写会炸渲染 + 不播种.

历史 lowercase user pin (`varName` / `scope` / `delta` / `value` / `a` / `b` / `result`) 全应迁 PascalCase. 不接受 fallback.

## 2. Default 值类型

Spec.InputSpec.Default 序列化进 JSON 走 `encoding/json`, 反序列化拿 `any`. 类型不一致 → 反序列化路径杂. 统一规则:

| InputSpec.Type                                | Default 用                                                       | 不许用                                                                                         |
| --------------------------------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `"Number"` / `"Integer"` / `"Duration"` | `json.Number("0")` (精度安全)                                  | 裸 `float64(0)` / `int(0)`                                                                 |
| `"Bool"`                                    | `false`                                                        | 字符串 `"false"`                                                                             |
| `"String"`                                  | `""`                                                           | 不许 `nil` (可选 pin 不设 `Required` 即可 — `InputSpec` **无** `Optional` 字段) |
| `"Point"`                                   | `map[string]any{"x": json.Number("0"), "y": json.Number("0")}` | 裸 struct                                                                                      |
| `"Rect"`                                    | 同 Point 风格                                                    | 裸 struct                                                                                      |
| `"JSON"`                                    | `map[string]any{}`                                             | `nil` (会被反序列化跳过)                                                                     |

**理由**: `json.Number` 保 int64 精度 — float64 mantissa 53 bit, 大 int 会丢精度. Inspector slider 改值后写回 JSON 再读出, 类型仍稳定.

## 3. Exec Exit 命名

| 节点形态                                                                                  | exec out pin name                    | 例子                                                                                                                                               |
| ----------------------------------------------------------------------------------------- | ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| 单 done 出口 (KeyPress / Sleep / SetVar / IncVar / Log / WindowTarget / MouseCalibration) | `Done`                             | `{Name: "Done", Type: TypeExec}`                                                                                                                 |
| 多语义出口 (If/Try/Loop/CheckTemplate)                                                    | 语义专名 PascalCase                  | `True`/`False`, `Out`/`Catch`, `Body`/`Done`, `Found`/`NotFound`                                                                   |
| 入口 / Fire-only (Start / EventTick / SubgraphInput — 无 exec in, 出口即"trigger")       | **PascalCase** (不是小写 out!) | `Start.Done`, `EventTick.Out`, `SubgraphInput.Done`                                                                                          |
| Switch 兜底出口 (例外)                                                                    | 小写 `default`                     | validator 把 `"default"` 当保留兜底关键字 (`validate.go` / `validator.go`), 跟用户 case 值同命名空间 — **有意保留, 别迁 `Default`** |

历史 lowercase out (`out` / `done` / `complete` / `yes` / `no` / `Fire`) 全清成上表风格. 单 done 节点用 `Done`. Loop 用 `Body`/`Done` (老 `body`/`complete` 已迁).

## 4. 节点 const 风格

每个节点文件 const block 命名:

```go
const (
    pinIn       = "In"          // exec in (固定 framework 名)
    pinDone     = "Done"        // exec done out (固定 framework 名)
    inVarName   = "VarName"     // 用户 data in
    inScope     = "Scope"
    outValue    = "Value"       // 用户 data out
)
```

短前缀缩写 (`sv`/`iv`/`gv` 等) 已不推荐 — 节点文件 scope 已经 isolation enough, 用 `inXxx`/`outXxx`/`pinIn`/`pinDone` 统一可读.

## 5. Display 风格

⚠ **当前后端没有 `Displayer` 接口** (全仓 grep `Display(` 仅命中无主实现) — 节点摘要由前端按 Spec 渲染. **别给 Go 节点加 `Display` 方法**, 是死代码. (历史遗留条目, 待真要做 backend display hook 时再补.)

## 6. Lint 守护

`internal/node/spec_consistency_test.go::TestSpecConsistency_PinNamingConvention` 解析所有 `Get`/`All()` 注册 spec, 验:

1. 所有 data pin Name (`Type != "Exec"`) 首字母大写. (exec-out 名 lint **不查**大小写 — 靠人工保持 PascalCase, 见上.)
2. 所有 Exec **in** pin Name == `"In"`. fire-only 节点无 exec in, 自然不触发 (无 lowercase-out 白名单).
3. 所有 Number/Integer/Duration Default 是 `json.Number` 类型.
4. 所有 String Default 非 `nil`.

任何节点违反 → test fail 给出 (Kind, pin) 定位. 添加新节点 = 必跑这测试.

## 7. 不能这样

❌ `{Name: "varName", Type: "String", ...}` (lowercase 用户 pin)
❌ `{Name: "in", Type: TypeExec, ...}` (lowercase 框架 pin, 必须 `pinIn`)
❌ `Default: 0.0` (Number 必须 `json.Number("0")`)
❌ `Default: nil` (String 必须 `""`)
❌ exec exit `"complete"`/`"yes"`/`"no"` (用 `Done`/`Found`/`NotFound` 等语义专名)

## 8. 迁移现状1

| 包                                                                | 风格             | 状态                                               |
| ----------------------------------------------------------------- | ---------------- | -------------------------------------------------- |
| detect, input, system, io, stopwatch, variable, purefunc, control | PascalCase       | 已 OK (2026-06-08 复核: data pin / exec-in 全合规) |
| event, mock, mockerror                                            | PascalCase / mix | event OK; mock 视 fishing-v2 不引用                |

exec-out 名 (2026-06-08 统一): 单出口 `Done`, 语义出口 PascalCase, 入口/fire-only 也 `Done`/`Out` (非小写 out). 例外: Switch 兜底 `default` (保留关键字). 历史脏名 `Fire` 已清 (WindowTarget/MouseCalibration → `Done`); `Yes`/`No`/`Missing` 命中分支已统一 `Found`/`NotFound` (DetectColor/DetectColorHSV/DualColorBarTrack); 区域输入 `Region`/`Roi` 已统一 `ROI`.

迁完跑 `go test ./internal/node/... -run TestSpecConsistency -count=1` 确认.

## 9. Canonical pin 词汇表 (同概念必须同名 — 2026-06-08 全节点审计锁定)

⚠ **加新节点前先查这张表**: 同一语义概念**必须复用既有 pin 名**, 别另起 (历史教训: 屏幕区域曾有 `Region`/`ROI`/`Roi` 三个名, 命中分支有 `Yes/No`/`Found/NotFound`/`Missing` 三套, 已全部收敛). lint 测不了"同概念异名", 全靠这张表 + review.

| 概念 | canonical pin | 类型 | 备注 / 反例 |
| --- | --- | --- | --- |
| 屏幕区域 / ROI | `ROI` | Geometry | ❌ 别用 `Region` / `Roi` |
| 超时 | `TimeoutMs` | Number | — |
| 轮询间隔 | `PollIntervalMs` | Number | — |
| 等待时长 (纯 Sleep) | `Duration` | Duration | 用专属 Duration widget |
| 动作时长 (按键/点击 hold) | `DurationMs` | Number | 跟 `MoveMs` 可共存于同节点 |
| 鼠标移动耗时 | `MoveMs` | Number | — |
| 抖动 | `JitterPct` | Number | — |
| 绝对位置比例 | `XRatio` / `YRatio` | Number | client-area [0,1] |
| 相对像素位移 | `Dx` / `Dy` | Number | 跟 XRatio/YRatio 是两个概念 |
| 鼠标键 | `Button` | String (left/right/middle) | — |
| 虚拟键 | `VK` | String | — |
| 变量名 / 作用域 | `VarName` / `Scope` | String | Scope 选项 auto/local/global |
| 模板匹配阈值 | `Threshold` | Number | WaitStable/Change 用 `StableThreshold`/`ChangeThreshold` (语义前缀, 有意保留) |
| 多模板匹配模式 | `MatchMode` | String (any/all) | — |
| 秒表 key | `Key` | String | — |
| 命中 / 未命中分支 | `Found` / `NotFound` | Exec | ❌ 别用 `Yes`/`No`/`Missing` |
| 颜色范围 (tuple, 支持 rgb+hsv) | `Range` | JSON + `dcRangeSchema` | c1Min..c3Max |
| 颜色范围 (hsv-only obj) | `HSV` | JSON + `hsvObjSchema` | 双色用 `InnerColor`/`OuterColor` |
| 捕获到变量 | `Capture<DataField>` | String, Advanced, Semantic:"capture" | 一对一映射 OutputSpec.Data 字段名 |
| 单成功出口 | `Done` | Exec | — |
| 错误出口 | `Fail` | Exec, Semantic:"error" | 带 Data `Error`/`Code` |

DataField (exec 出口携带数据) 命名:
- 置信度 → **`Conf`** (模板族已用; 新节点统一 Conf, 别用 `Confidence`).
- 命中位置 → `Point` (模板锚点) 或 `Center` (颜色质心) — 语义不同各自保留, 新节点按语义选。
- 计数 → 带域前缀: `PixelCount` / `BlobCount` / `ClusterCount` (`Count` 太泛, 仅 DetectColor 历史保留).

**实时核对 / 揪分裂**: 上面是手维护的"规矩", 会跟代码漂。跑 `task nodes:pins` (= `go run ./cmd/node-catalog pins`) 按名合并**全节点实际 pin 名** + 列用量, 末尾「命名分裂告警」揪形近撞名 (`Roi` vs `ROI`) 与同名不同具体类型 —— **告警段为空 = 没分裂, 表与代码一致**。加 pin 后跑一次核对。

## 10. i18n 文案规范 (`frontend/src/i18n/{zh,en}.ts` 的 `node.<Kind>` 块)

- **zh 一律人话中文**, label/hint **不准夹内部黑话**: ❌ `runner`/`callee`/`SubgraphInput`/`framework`/`stringify`/`wildcard`/`dynamic Inputs[]`/`isAnonymous`. 合法保留缩写: HSV / RGB / ROI / ms / ID / URL / px / DPI。
- **en label 用 sentence case** (`Color blob locate`), 不用 Title Case (`Color Blob Locate`)。
- **捕获框 label 风格**统一 `<字段>→变量` / `<field> → variable`。
- **时间单位**统一 `(ms)`, 不用 `(毫秒)`。
- **区域 label**: zh `区域 (ratio)`, en `ROI (ratio)`。
- **dropdown option 值要翻译** (精确/包含/前缀...), 不留 raw enum (`exact`/`contains`)。例外 (zh/en 一致地保留标识符): `Scope` 的 auto/local/global、`Log.Level` 的 debug/info/warn。
- 结构化 JSON 输入 (colorRange / ObjSchema) 的子字段 label 嵌在 `input.<Pin>.<fieldKey>.label` (前端 `StructuredInput.vue` 按 `fieldPath.<key>` 取), 别漏。
- zh/en **键结构必须对称** (parity 测试守; 加节点跑 `pnpm i18n:check`)。
- 改完必跑 `cd frontend && pnpm gen:node-i18n` 重生 `internal/catalog/node-i18n.json` (catalog drift 测试守)。

## 11. 前端命名约定 + 已知待办

- **exec pin 名前端别硬编码小写** `'in'`/`'out'`: 后端是 `In`/`Done` (PascalCase)。FE 从 backend Spec 派生 pin 名时**原样保留大小写**, 别 lowercase。
- **Expr 节点 config 形状** (实证 `internal/services/container/configtypes.go ParseExprConfig`): 表达式存 **`config.literal.Expression`** (pin literal); 动态输入声明存 **`config.Inputs[]`** 的 `{Name, Type}` (PascalCase)。FE 读写必须用这个形状 — ❌ 别用 `config.expr` / `config.inputs` / `.name` / `.type` (2026-06-08 修过 `pinLiterals.ts` / `useExprFusion.ts` 的此类小写 desync)。
- **⏳ 待修 (记档防忘)**: `useFlowInteraction.ts` 的 `isSingleInOut` 判 `execOut[0]==='out'` (小写) 永不匹配 → "拖节点到线上自动断接"功能一直是死的; line 111-112 还用 `${id}.in`/`.out` 拼边。修法: 改判 count (`execIn.length===1 && execOut.length===1`) + 用真实 pin 名拼边。**启用=改行为, 需单独验证, 别顺手开**。同类小写 fallback (`useGraphMutations ?? 'out'`、`useInlineMenu ?? 'out'`、`pinSpec.ts:120`) 仅兜未知 kind, 低优先。
