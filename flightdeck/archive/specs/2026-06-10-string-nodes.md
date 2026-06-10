---
status: done
summary: 字符串函数节点 Replace/Substring/Trim/Upper/Lower/IndexOf/StartsWith/EndsWith/Regex (加节点路线图阶段4) — 已实现 (含 Length rune 化 + INVALID_REGEX_PATTERN 编辑期校验)
last_updated: 2026-06-10
related: [specs/2026-06-10-random-nodes.md]
---

# 字符串函数节点（加节点路线图 · 阶段 4）

## 背景与定位

加节点路线图（[random-nodes spec](2026-06-10-random-nodes.md)）的**阶段 4**（末批）。补 PureFunc 缺的字符串函数。本框架（视觉+输入自动化）字符串需求最弱，属补完整性。**无框架改动、无新 palette 分类**——进 `internal/nodes/purefunc` 包、Category `PureFunc`，跟现有 Concat/Contains/Length 并列。

## 已验证的源码事实

- **现有字符串节点只有 Concat/Contains/Length**（`purefunc.go`）。其余全缺。
- **`Length` 现用 `len(formatValue(...))` = 字节长度**（中文按 UTF-8 字节算，`"中文"→6`）——本 spec **改为 rune-based**（见「byte vs rune 判断」）。
- **`in.String` 是纯类型断言**（`inputs.go:67`：非 string → 返 `""`，**不做数字/bool 转换**）——故新字符串节点对非字符串输入得 `""`。（`formatValue` 是 Concat/ToString 用的软转，与此不同。）`in.Bool`/`in.Int` 同理各自取值。

## byte vs rune 判断（关键，已拍）

用户用 CJK（中文）文本。**涉及位置/长度语义的节点一律 rune-based**（Substring 的 `[]rune` 切、IndexOf 的 rune 下标、Length 的字符数）。`StartsWith/EndsWith/Replace/Trim/Upper/Lower` 内部仍用 Go string/regexp（字节实现），但不涉及"第几个字符"，结果对用户无差——故不强说"全系统 rune"，只保证**用户能感知的位置/长度语义统一按字符**。

**⛔ 同步修现有 `Length` 为 rune-based**：现 `Length` 用 `len(formatValue(...))`=**字节**长度（`Length("中文")=6`），与新节点 rune 语义冲突——`Length→Substring.Length` 会拿错结果（reviewer 标 HIGH）。改为 `utf8.RuneCountInString`。**项目未发布，按[二号铁律]直接改**。**风险已核实极低**：全仓 grep，Length 仅节点定义 + 1 个 ASCII 测试（`"hello"→5`，byte==rune），**无 fixture/快照/示例 graph 依赖字节语义**。回归仅需把那 1 个测试加个 CJK 断言。

## 节点清单（10 个，加进 `purefunc` 包，Category `PureFunc`）

> 全 `IsPureData:true` + Evaluator。底层 `strings` / `regexp` 包。

| 节点 | 输入 | 输出 | 实现 |
|---|---|---|---|
> String 输入经 `in.String`（**非字符串→`""`**，不 panic）；Bool 输入（Replace 的 `All`）经 `in.Bool`；Integer 经 `in.Int`。空串输入：见各行。
>
> ⚠ **正则非法 pattern 不返 error**（pure-data 的 Evaluate error 会被数据线构建路径 `buildDataWireFor` 静默吞掉）→ 运行时返安全值（false/""）**并 `ctx.Log().Warn` 记一行**（让动态 pattern 写错也能在日志看到，不致纯静默），**编辑期 validator 对 literal pattern 报红**（见落地清单）。

| 节点 | 输入 | 输出 | 实现 |
|---|---|---|---|
| **Replace** | `Text`、`Old`、`New`(String)、`All`(Bool,默认 true) | `Result`(String) | `All`=true→`strings.ReplaceAll`；false→`strings.Replace(Text,Old,New,1)`（只替第一个）。`Old==""`→原样返（不插入）|
| **Substring** | `Text`(String)、`Start`(Integer,默认0)、`Length`(Integer,默认 **-1**) | `Result`(String) | **rune-based**：`r:=[]rune(Text)`；Start clamp `[0,len]`（负→0、≥len→空串）；`Length<0`→取到末尾(默认)、`==0`→空串、`>0`→取 N rune；end=`min(start+Length,len)`；返 `string(r[start:end])`。`Text==""`→`""` |
| **Trim** | `Text`(String) | `Result`(String) | `strings.TrimSpace`（两端空白；`""`→`""`）|
| **ToUpper** | `Text`(String) | `Result`(String) | `strings.ToUpper` |
| **ToLower** | `Text`(String) | `Result`(String) | `strings.ToLower` |
| **IndexOf** | `Text`(String)、`Sub`(String) | `Result`(Number) | **rune 下标**：`b:=strings.Index(Text,Sub); b<0→-1; 否则→utf8.RuneCountInString(Text[:b])`。`Sub==""`→`0`（Go 语义）；`Text==""&&Sub!=""`→-1 |
| **StartsWith** | `Text`(String)、`Prefix`(String) | `Result`(Bool) | `strings.HasPrefix`（`Prefix==""`→true）|
| **EndsWith** | `Text`(String)、`Suffix`(String) | `Result`(Bool) | `strings.HasSuffix`（`Suffix==""`→true）|
| **RegexMatch** | `Text`(String)、`Pattern`(String) | `Result`(Bool) | `regexp.MatchString`——**搜索/包含**匹配（`"abc"`+`"b"`→true；全文匹配写 `^…$`）。非法 Pattern→**false** + Log.Warn（编辑期对 literal 报红）|
| **RegexExtract** | `Text`(String)、`Pattern`(String) | `Result`(String) | `regexp.Compile`+`FindStringSubmatch`：`len(m)>1`（有捕获组，含命名组 `(?P<n>)` 但**按位置取**）→`m[1]`（**多组只组1**、`(?:)` 不计组、空匹配亦返空串）；否则→`m[0]`整匹配；无匹配/非法→`""` + Log.Warn。**`""` 不区分"匹配到空"与"没匹配"**——要判存在先用 RegexMatch |

## 设计判断（已拍，待复核）

- **进 purefunc 包、Category PureFunc**——字符串就是纯函数，不新建包/分类，零前端 Category 工作。
- **位置/长度语义 rune-based**（含改 Length）——CJK 正确；非位置节点（Replace/StartsWith…）字节实现但用户无感，不强说"全系统 rune"。
- **Replace 带 `All` 开关**（默认 **true=全替**——脚本场景"替全部"更常用，同 Python `str.replace`；false=替第一个为 opt-in。注：JS `replace` 默认替首个是历史特例，不作依据）。**`Old==""`→原样返**（刻意偏离 Go `ReplaceAll` 的"每字符间插入"；`All=false` 时空串"第一个位置"无定义、同样原样返）。
- **Substring 用 Start+Length**（负=到末尾默认/0=空/正=N），与 [collection spec](2026-06-10-collection-nodes.md) ListSlice **同款约定**。`Length` 默认 -1 是**程序设定的 Spec.Default**（自动进 config.literal），"取到末尾"默认行为与"widget 是否允许手输负数"**无关**、永远生效；仅用户想手动改回 -1 才需数字框接受负数（标准框支持，小 UX 边角）。
- **正则错误走编辑期校验 + 运行时 Log.Warn，不走 Evaluate error**——pure-data 的 error 被框架数据线路径静默吞（源码核 `buildDataWireFor`），返 error 没意义。**已知窗口**：wired-in（动态）pattern 无法编辑期校验，非法时运行时返 false/"" + Log.Warn（刻意，非遗漏）。
- **正则节点全错误路径 = 安全值**（与现有 PureFunc 节点"从不 error"契约一致）。
- **IndexOf/Length 输出 Number**（非 Integer）——与现有 PureFunc 数值输出风格一致；下游 in.Int 宽松转。

## 非目标（YAGNI）

- 不做 RegexReplace/RegexFindAll（多匹配要 List，且 Filter/Map 同属高阶；按需再加）。
- 不做 PadLeft/PadRight/Repeat/Format(模板)——无 demand。Format 真要做走 Expr 或另设计。
- 不做大小写区域设置（locale）——`strings.ToUpper/Lower` 的 Unicode 默认够用。
- **RegexExtract 不支持返回全部捕获组 / 按组名取值**——永远只返组1（位置）；要多组/命名组按需另开（避免被当缺陷）。
- RegexExtract 的"无匹配""非法 pattern""匹配到空串"在输出层都是 `""`（不区分；非法已被编辑期校验+Log.Warn 兜，运行时几乎只剩"无匹配/匹配空"）。
- IndexOf `Sub==""→0`（Go 语义，刻意保留）。**别用 `IndexOf(...)>=0` 实现"包含"判断**——既与现有 `Contains` 节点能力重复，且 Sub 为空时恒真（注：`Contains` 对空子串同样恒真，这是 Go `strings.Contains` 语义，非 IndexOf 独有）。判包含直接用 `Contains` 节点。
- **正则非法 Pattern 的 Log.Warn 不做限流**（YAGNI）：高频动态非法 pattern 会刷日志，但那是持续性 bug、该修 pattern 本身；不为它加去重/限流。
- **validator 随图改重跑**：用户给 Pattern pin 连/断线 = 图变更 → 编辑期校验自动重评（连线后跳过校验、残留红错清除），标准行为无需特殊处理。
- **Length 改 rune 的残余风险**：项目按 CLAUDE.md 是**单机内测、无外部用户**，grep + 用户自己的 graph 即全部面，可接受残余（理论上仓库外本地 graph 不在 grep 范围，但内测期等同用户自有）。

## 落地清单（按 add-node.md，要点）

1. **后端**：`purefunc.go` 加 10 个 node type + `init()` 注册追加；**改现有 `Length` 为 `utf8.RuneCountInString`**（grep 确认仅 1 处 ASCII 测试依赖，加 CJK 断言即可）。正则非法 pattern 运行时返 false/"" + `ctx.Log().Warn`（**不返 error**）。无新包、无 blank-import 改动。
2. **编辑期校验**：`validator.go::checkGraphPerKind` 加 RegexMatch/RegexExtract 分支——**仅当 Pattern pin 无 incoming data edge 且 config.literal["Pattern"] 存在**（=静态 literal）时跑 `regexp.Compile`，失败 → NodeInspector 红错（add-node §7）。有连线（动态）则跳过，运行时 Log.Warn 兜。
3. **测试**：必含 **CJK**（`Substring("中文abc",0,2)="中文"`、`IndexOf("a中文","文")=2`、`Length("中文")=2`）+ **rune 一致性集成用例**（`Length(s)` 的结果喂 `Substring(s,0,Length(s))` 应得整串——验 Length/Substring 都按 rune、无字节混用）+ 边界：IndexOf `Sub=""→0`、Substring `Start≥len`/`Length=0→""`/`Length<0` 到末尾、Replace `All=false` 多次只替首个 + `Old=""→原样`、RegexExtract 多组只组1/空捕获组/无匹配、RegexMatch 包含语义。
4. **i18n**：zh/en 加 10 个 `node.<Kind>` + Replace 的 `input.All.label`、RegexMatch 搜索语义/全文提示。**⛔ vue-i18n 转义**：真正会炸编译的是 **`{ } | @ $`**（非 `\ . * + ?`——所以 `\d+`、`.*` 这类示例**安全**）；描述若用到 `{ }`(如 `\d{2,3}`)、`|`(或)、`$`(行尾锚) 才需转义，见 [[vue-i18n-message-compiler-traps]]；`pnpm i18n:check` 的 `[compile]` 段兜底**必绿**。跑 `pnpm gen:node-i18n`。无 nodeGroup 改动。
5. **验证**：`go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... ./internal/services/container/... -count=1`（含改 Length 回归）；`pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`；`task nodes` 见 10 节点在 PureFunc。
