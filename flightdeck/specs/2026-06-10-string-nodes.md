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

用户用 CJK（中文）文本。**新增的"位置/长度"类节点一律 rune-based**（按字符数，中文正确），`[]rune(s)` 切。`StartsWith/EndsWith/Replace/Trim/Upper/Lower` 不涉及位置、按字节/字符串操作天然正确。**已知不一致**：现有 `Length` 是字节长度（非回归，不在本 spec 改；如要统一另开）。

## 节点清单（10 个，加进 `purefunc` 包，Category `PureFunc`）

> 全 `IsPureData:true` + Evaluator。底层 `strings` / `regexp` 包。

| 节点 | 输入 | 输出 | 实现 |
|---|---|---|---|
| **Replace** | `Text`(String)、`Old`(String)、`New`(String) | `Result`(String) | `strings.ReplaceAll`（替换全部）|
| **Substring** | `Text`(String)、`Start`(Integer,默认0)、`Length`(Integer,默认 **-1**) | `Result`(String) | **rune-based**：`r:=[]rune(Text)`；Start clamp `[0,len]`（负 Start→0）；`Length<0`→取到末尾(默认)、`Length==0`→**空串**(可表达取0)、`Length>0`→取 Length 个 rune；end=`min(start+Length,len)`；返 `string(r[start:end])` |
| **Trim** | `Text`(String) | `Result`(String) | `strings.TrimSpace`（两端空白）|
| **ToUpper** | `Text`(String) | `Result`(String) | `strings.ToUpper` |
| **ToLower** | `Text`(String) | `Result`(String) | `strings.ToLower` |
| **IndexOf** | `Text`(String)、`Sub`(String) | `Result`(Number) | **rune index**：找不到→-1；找到→子串首次出现的**字符**下标（由 `strings.Index` 字节位转 rune 数）|
| **StartsWith** | `Text`(String)、`Prefix`(String) | `Result`(Bool) | `strings.HasPrefix` |
| **EndsWith** | `Text`(String)、`Suffix`(String) | `Result`(Bool) | `strings.HasSuffix` |
| **RegexMatch** | `Text`(String)、`Pattern`(String) | `Result`(Bool) | `regexp.MatchString`；**Pattern 非法→返 error**（节点 Fail/编辑期校验可接）|
| **RegexExtract** | `Text`(String)、`Pattern`(String) | `Result`(String) | `regexp.Compile`+`FindStringSubmatch`：有捕获组→返组1；无组→返整个匹配；无匹配→`""`；Pattern 非法→error |

## 设计判断（已拍，待复核）

- **进 purefunc 包、Category PureFunc**——字符串就是纯函数，不新建包/分类，零前端 Category 工作。
- **位置类 rune-based**（Substring/IndexOf）——CJK 正确。
- **Replace = ReplaceAll**（替换全部，最常用）；要"只替一次/正则替换"将来按需加 ReplaceN/RegexReplace。
- **Substring 用 Start+Length**（Length<0=到末尾默认、0=空、>0=N 个），与 [collection spec](2026-06-10-collection-nodes.md) 的 ListSlice **同款约定**（负=rest/0=空/正=N），0 能表达"取0个"，绕开"End 默认值"歧义。
- **Regex 非法 Pattern → error**（不静默返 false/空）——让用户知道正则写错了。

## 非目标（YAGNI）

- 不做 RegexReplace/RegexFindAll（多匹配要 List，且 Filter/Map 同属高阶；按需再加）。
- 不做 PadLeft/PadRight/Repeat/Format(模板)——无 demand。Format 真要做走 Expr 或另设计。
- 不改现有 `Length` 的字节语义。
- 不做大小写区域设置（locale）——`strings.ToUpper/Lower` 的 Unicode 默认够用。

## 落地清单（按 add-node.md，要点）

1. **后端**：`purefunc.go` 加 10 个 node type + `init()` 注册列表追加。无新包、无 blank-import 改动。`regexp` 非法 pattern 在 Evaluate 里 `Compile` 失败返 `fmt.Errorf`。
2. **i18n**：zh/en 加 10 个 `node.<Kind>`（label/description/pin label）。**注意 vue-i18n 转义**：description/示例里如含 `{` `}` `|` `@` `$`（正则节点描述很可能举正则例子）必须转义，见 checklist [[vue-i18n-message-compiler-traps]]。跑 `pnpm gen:node-i18n`。**无 nodeGroup 改动**。
3. **验证**：`go test ./internal/nodes/... ./internal/catalog/... -count=1`；`pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`（`[compile]` 段兜正则文案转义）；`task nodes` 见 10 节点在 PureFunc。
