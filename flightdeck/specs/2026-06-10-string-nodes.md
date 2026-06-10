---
status: active
summary: 字符串函数节点 Replace/Substring/Trim/Upper/Lower/IndexOf/StartsWith/EndsWith/Regex (加节点路线图阶段4)
last_updated: 2026-06-10
related: [specs/2026-06-10-random-nodes.md]
---

# 字符串函数节点（加节点路线图 · 阶段 4）

## 背景与定位

加节点路线图（[random-nodes spec](2026-06-10-random-nodes.md)）的**阶段 4**（末批）。补 PureFunc 缺的字符串函数。本框架（视觉+输入自动化）字符串需求最弱，属补完整性。**无框架改动、无新 palette 分类**——进 `internal/nodes/purefunc` 包、Category `PureFunc`，跟现有 Concat/Contains/Length 并列。

## 已验证的源码事实

- **现有字符串节点只有 Concat/Contains/Length**（`purefunc.go`）。其余全缺。
- **`Length` 用 `len(formatValue(...))` = 字节长度**（多字节字符如中文会按 UTF-8 字节算）——这是已存在行为，本 spec 的新节点按下方「byte vs rune」判断处理，不改 Length。
- **`formatValue`**（purefunc 包内未导出）软转任意值为 string；`in.String`/`in.Int` 取值。

## byte vs rune 判断（关键，已拍）

用户用 CJK（中文）文本。**全字符串系统一律 rune-based**（按字符数，中文正确）。位置/长度类（Substring/IndexOf）`[]rune` 切 / rune 下标；`StartsWith/EndsWith/Replace/Trim/Upper/Lower` 不涉及位置、按字符串操作天然正确。

**⛔ 同步修现有 `Length` 为 rune-based**：现 `Length` 用 `len(formatValue(...))`=**字节**长度（`Length("中文")=6`），与新节点 rune 语义冲突——用户 `Length→Substring.Length` 会拿错结果（reviewer 标 HIGH）。改为 `utf8.RuneCountInString`。**项目未发布，按[二号铁律]直接改不留兼容**（字节长度对 CJK 本就无用）。这是本 spec 的一部分，不另开。回归：更新 `Length` 现有测试为 rune 预期。

## 节点清单（10 个，加进 `purefunc` 包，Category `PureFunc`）

> 全 `IsPureData:true` + Evaluator。底层 `strings` / `regexp` 包。

| 节点 | 输入 | 输出 | 实现 |
|---|---|---|---|
> 输入全经 `in.String`（**非字符串→`""`**，不 panic）。空串输入：见各行。
>
> ⚠ **正则非法 pattern 不返 error**（pure-data 的 Evaluate error 会被 `buildDataWireFor` 静默吞掉，line 57-60）→ 运行时返安全值（false/""），**靠编辑期 validator** 报红（见落地清单）。

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
| **RegexMatch** | `Text`(String)、`Pattern`(String) | `Result`(Bool) | `regexp.MatchString`——**搜索/包含**匹配（`"abc"`+`"b"`→true；全文匹配写 `^…$`）。非法 Pattern→**false**（编辑期校验报红）|
| **RegexExtract** | `Text`(String)、`Pattern`(String) | `Result`(String) | `regexp.Compile`+`FindStringSubmatch`：`len(m)>1`（有捕获组）→`m[1]`（即便空串；多组只返组1、`(?:)` 不计组）；否则→`m[0]`整匹配；无匹配/非法 Pattern→`""` |

## 设计判断（已拍，待复核）

- **进 purefunc 包、Category PureFunc**——字符串就是纯函数，不新建包/分类，零前端 Category 工作。
- **全字符串系统 rune-based**（含改 Length）——CJK 正确、内部统一，不留字节/字符混用陷阱。
- **Replace 带 `All` 开关**（默认 true=全替、false=替第一个）——覆盖高频"替第一个"，免逼用户上正则。
- **Substring 用 Start+Length**（负=到末尾默认/0=空/正=N），与 [collection spec](2026-06-10-collection-nodes.md) ListSlice **同款约定**。注：`Length` 默认 -1，需 number widget 允许负输入（标准数字框允许；impl 验一下）。
- **正则错误走编辑期校验而非运行时 error**——因 pure-data Evaluate error 被框架静默吞（源码核 `buildDataWireFor:57`），返 error 没意义；改 validator 编译 literal pattern 报红。运行时非法→false/""。
- **正则节点全错误路径 = 安全值**（与现有 PureFunc 节点"从不 error"契约一致）。

## 非目标（YAGNI）

- 不做 RegexReplace/RegexFindAll（多匹配要 List，且 Filter/Map 同属高阶；按需再加）。
- 不做 PadLeft/PadRight/Repeat/Format(模板)——无 demand。Format 真要做走 Expr 或另设计。
- 不做大小写区域设置（locale）——`strings.ToUpper/Lower` 的 Unicode 默认够用。
- RegexExtract 的"无匹配"与"非法 pattern"在输出层都是 `""`（不区分；非法已被编辑期校验拦在前面，运行时几乎只剩"无匹配"）。

## 落地清单（按 add-node.md，要点）

1. **后端**：`purefunc.go` 加 10 个 node type + `init()` 注册追加；**改现有 `Length` 为 `utf8.RuneCountInString`** + 更新其测试为 rune 预期。正则非法 pattern 运行时返 false/""（**不返 error**）。无新包、无 blank-import 改动。
2. **编辑期校验**：`validator.go::checkGraphPerKind` 加 RegexMatch/RegexExtract 分支——对 **literal** Pattern 跑 `regexp.Compile`，失败 → NodeInspector 红错（add-node §7；wired-in pattern 无法静态校验，运行时安全返空）。
3. **测试**：必含 **CJK 用例**（`Substring("中文abc",0,2)="中文"`、`IndexOf("a中文","文")=2`、`Length("中文")=2`）+ 空串/边界（IndexOf 空 Sub→0、Substring 越界、Replace All 开关、RegexExtract 多组/空组/无匹配）。
4. **i18n**：zh/en 加 10 个 `node.<Kind>` + Replace 的 `input.All.label`、RegexMatch 搜索语义/全文提示。**⛔ vue-i18n 转义硬步骤**：正则节点描述**尽量不放裸正则元字符**（`\ . * + ? ^ $ { } ( ) | [ ]`）；非放不可必转义，见 [[vue-i18n-message-compiler-traps]]；`pnpm i18n:check` 的 `[compile]` 段是最后兜底，**必须绿**。跑 `pnpm gen:node-i18n`。无 nodeGroup 改动。
5. **验证**：`go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... ./internal/services/container/... -count=1`（含改 Length 后的回归）；`pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`；`task nodes` 见 10 节点在 PureFunc。
