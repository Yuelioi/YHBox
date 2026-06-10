---
status: done
summary: 字符串函数 10 节点 + Length 改 rune + 正则编辑期校验的实现计划 (TDD 分任务) — 已实现 (3ceb6ae..f1d4a5d), 终审 SHIP
last_updated: 2026-06-10
implements: specs/2026-06-10-string-nodes.md
---

# 字符串函数节点 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 purefunc 包补 Replace/Substring/Trim/ToUpper/ToLower/IndexOf/StartsWith/EndsWith/RegexMatch/RegexExtract 十个字符串节点；把现有 Length 改成 rune 计数；正则 literal pattern 编辑期校验红错。

**Architecture:** 零框架改动。节点进现有 `internal/nodes/purefunc` 包（新文件 `strings.go`），Category `PureFunc`。位置/长度语义全 rune-based（CJK 正确）。正则非法 pattern 运行时返安全值 + `ctx.Log().Warn`（pure-data 的 Evaluate error 会被数据线路径吞，不返 error）；编辑期 `validator.go` 镜像 `validateCronConfig` 范式对 literal pattern 报红。

**Tech Stack:** Go（`strings`/`regexp`/`unicode/utf8`）、vue-i18n、Taskfile。

**实现依据见 spec：** `flightdeck/specs/2026-06-10-string-nodes.md`。配套规范 `flightdeck/checklists/add-node.md`、`flightdeck/checklists/vue-i18n-message-compiler-traps.md`。

---

## File Structure

**新建：**
- `internal/nodes/purefunc/strings.go` — 10 个字符串节点（注册在 purefunc.go init 追加）。
- `internal/nodes/purefunc/strings_test.go` — 单测（CJK + 边界 + 正则安全值）。

**修改（后端）：**
- `internal/nodes/purefunc/purefunc.go` — `Length.Evaluate` 改 rune；init() 追加 10 个；包 doc 31→41。
- `internal/nodes/purefunc/purefunc_test.go` — Length 表加 CJK 行。
- `internal/services/container/validator.go` — `CodeInvalidRegexPattern` + `validateRegexPattern` + switch 两 case。
- `internal/services/container/validator_test.go`（或该文件实际测试所在处）— 校验用例。

**修改（前端 / i18n / docs）：**
- `frontend/src/i18n/zh.ts` + `en.ts` — 10 个 `node.<Kind>` 块 + `validation.INVALID_REGEX_PATTERN`。
- `flightdeck/docs/node-system-reference.md` — PureFunc 行 32→42。

**已核源码事实（实现者不必重查，撞不一致时停下报告）：**
- `in.String` 是纯类型断言（inputs.go:67，非 string→`""`）；`in.Bool`（:135，非 bool→false）；`in.Int`（:114，float64 截断）。
- `ctx.Log()` 返回 `LogService`，有 `Warn(format string, args ...any)`（interfaces.go:213-217）。**pure-data Evaluate 测试必须走 `node.EvaluatePureData(..., node.StubServices())`**（math_test.go 的 `evalMathNode` helper 范式）——裸 `Evaluate(nil, in)` 会让 `ctx.Log()` panic。StubServices 提供 stdout LogService。
- 现有 `Length`（purefunc.go:352-361）用 `len(formatValue(in.Raw("S")))` = 字节长度；全仓只有 `TestEvaluate_22PureFuncs` 一行 ASCII 用例依赖（`"hello"→5`，byte==rune 无感）。
- validator 范式（validator.go:935-953 `validateCronConfig`）：`PinString(n, "Expression")` 读 literal → 空返 nil（空 = 准备连上游/没填）→ parse 失败 emit `ValidationError{Severity: SeverityError, Code: ..., NodeID: n.ID, Params: {...}}`；动态来源（连线）解析失败由 runtime 兜——**正则镜像此范式即满足 spec"连线跳过校验"的标准行为**。switch 在 `checkGraphPerKind`（:661）。Code 常量块在 :56-71。
- 前端红错文案：`frontend/src/i18n/zh.ts:1591` / `en.ts:1571` 的 `INVALID_CRON_EXPR: 'Cron 表达式无效: {expr} ({parseErr})'` 所在 `validation` 块加同款条目（`{pattern}`/`{parseErr}` 是 vue-i18n 插值，**此处的花括号是合法用法**）。
- 新 pin 名 Text/Old/New/All/Start/Length/Sub/Prefix/Suffix/Pattern 经 catalog 全量扫描**零撞名**（Start/Length/Sub 只是节点 kind 名，分裂守卫只比 pin 名）。Substring 的 `Length`/`Start` 是全新 pin 名用 Integer 无分裂。
- `evalMathNode`/`wantNum`/`wantNaN` helper 已在 math_test.go（同包可直接用）。

---

## Task 1: Length 改 rune 计数

**Files:**
- Modify: `internal/nodes/purefunc/purefunc.go`（Length.Evaluate + import）
- Modify: `internal/nodes/purefunc/purefunc_test.go`（CJK 行）

- [ ] **Step 1: 写失败测试**

`purefunc_test.go::TestEvaluate_22PureFuncs` 的 cases 表里，紧跟现有 `{"Length", map[string]any{"S": "hello"}, 5.0},` 行后加：

```go
		{"Length", map[string]any{"S": "中文abc"}, 5.0}, // rune 计数 (字节是 9) — specs/2026-06-10-string-nodes.md
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/purefunc/ -run TestEvaluate_22PureFuncs -v`
Expected: FAIL（`Length#01` 子测试得 9.0，want 5.0）。

- [ ] **Step 3: 改实现**

`purefunc.go::Length.Evaluate` 改为：

```go
// Evaluate — rune 计数 (CJK 一个字算 1, 非字节). 与 Substring/IndexOf 的位置语义统一,
// 见 specs/2026-06-10-string-nodes.md "byte vs rune 判断".
func (Length) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return float64(utf8.RuneCountInString(formatValue(in.Raw("S")))), nil
}
```

purefunc.go import 块加 `"unicode/utf8"`。

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/purefunc/ -count=1`
Expected: PASS。

- [ ] **Step 5: 全仓回归（Length 消费方）**

Run: `go build ./... && go test ./internal/catalog/... ./internal/services/container/... -count=1`
Expected: PASS（已知 runtime fish-fixture 预存失败除外）。

- [ ] **Step 6: Commit**

```bash
git add internal/nodes/purefunc/purefunc.go internal/nodes/purefunc/purefunc_test.go
git commit -m "fix(purefunc): Length 改 rune 计数 (CJK 字符数, 与阶段4 位置语义统一)"
```

---

## Task 2: strings.go — 非正则 8 节点

**Files:**
- Create: `internal/nodes/purefunc/strings.go`
- Create: `internal/nodes/purefunc/strings_test.go`

- [ ] **Step 1: 写失败测试**

`internal/nodes/purefunc/strings_test.go`：

```go
package purefunc

import (
	"testing"

	"yotta/internal/node"
)

// wantStr / wantBool — strings 节点断言 helper (evalMathNode 在 math_test.go, 同包复用).
func wantStr(t *testing.T, got any, want string) {
	t.Helper()
	s, ok := got.(string)
	if !ok || s != want {
		t.Fatalf("want %q, got %T(%v)", want, got, got)
	}
}

func wantBool(t *testing.T, got any, want bool) {
	t.Helper()
	b, ok := got.(bool)
	if !ok || b != want {
		t.Fatalf("want %v, got %T(%v)", want, got, got)
	}
}

func TestReplace(t *testing.T) {
	// All 默认 true → 全替
	wantStr(t, evalMathNode(t, &Replace{}, map[string]any{"Text": "a-b-c", "Old": "-", "New": "+"}), "a+b+c")
	// All=false → 只替第一个
	wantStr(t, evalMathNode(t, &Replace{}, map[string]any{"Text": "a-b-c", "Old": "-", "New": "+", "All": false}), "a+b-c")
	// Old="" → 原样返 (刻意偏离 Go ReplaceAll 的逐字符插入)
	wantStr(t, evalMathNode(t, &Replace{}, map[string]any{"Text": "abc", "Old": "", "New": "x"}), "abc")
	wantStr(t, evalMathNode(t, &Replace{}, map[string]any{"Text": "abc", "Old": "", "New": "x", "All": false}), "abc")
}

func TestSubstring_RuneBased(t *testing.T) {
	// CJK: 前 2 个字符
	wantStr(t, evalMathNode(t, &Substring{}, map[string]any{"Text": "中文abc", "Start": 0, "Length": 2}), "中文")
	// Length 默认 -1 → 取到末尾
	wantStr(t, evalMathNode(t, &Substring{}, map[string]any{"Text": "中文abc", "Start": 2}), "abc")
	// Length=0 → 空串
	wantStr(t, evalMathNode(t, &Substring{}, map[string]any{"Text": "abc", "Start": 1, "Length": 0}), "")
	// Start 越界 → 空串; Start 负 → clamp 0
	wantStr(t, evalMathNode(t, &Substring{}, map[string]any{"Text": "abc", "Start": 99}), "")
	wantStr(t, evalMathNode(t, &Substring{}, map[string]any{"Text": "abc", "Start": -5, "Length": 2}), "ab")
	// Length 超尾 → 到末尾
	wantStr(t, evalMathNode(t, &Substring{}, map[string]any{"Text": "abc", "Start": 1, "Length": 99}), "bc")
	// 空文本
	wantStr(t, evalMathNode(t, &Substring{}, map[string]any{"Text": "", "Start": 0, "Length": 3}), "")
}

// rune 一致性: Length(s) 喂 Substring(s, 0, Length(s)) 应得整串 (验两节点同按 rune, 无字节混用).
func TestRuneConsistency_LengthFeedsSubstring(t *testing.T) {
	s := "中文abc"
	n := evalMathNode(t, &Length{}, map[string]any{"S": s})
	length, ok := n.(float64)
	if !ok || length != 5 {
		t.Fatalf("Length(%q) = %v, want 5", s, n)
	}
	wantStr(t, evalMathNode(t, &Substring{}, map[string]any{"Text": s, "Start": 0, "Length": int(length)}), s)
}

func TestTrimUpperLower(t *testing.T) {
	wantStr(t, evalMathNode(t, &Trim{}, map[string]any{"Text": "  hi\t\n"}), "hi")
	wantStr(t, evalMathNode(t, &Trim{}, map[string]any{"Text": ""}), "")
	wantStr(t, evalMathNode(t, &ToUpper{}, map[string]any{"Text": "abC中"}), "ABC中")
	wantStr(t, evalMathNode(t, &ToLower{}, map[string]any{"Text": "AbC中"}), "abc中")
}

func TestIndexOf_RuneIndex(t *testing.T) {
	cases := []struct {
		name      string
		text, sub string
		want      float64
	}{
		{"ascii", "hello", "ll", 2},
		{"cjk_index", "a中文", "文", 2},  // 字节下标是 4, rune 下标 2
		{"not_found", "abc", "x", -1},
		{"empty_sub", "abc", "", 0},    // Go 语义, 刻意保留
		{"empty_text", "", "x", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalMathNode(t, &IndexOf{}, map[string]any{"Text": tc.text, "Sub": tc.sub})
			f, ok := got.(float64)
			if !ok || f != tc.want {
				t.Fatalf("IndexOf(%q,%q) = %T(%v), want %v", tc.text, tc.sub, got, got, tc.want)
			}
		})
	}
}

func TestStartsEndsWith(t *testing.T) {
	wantBool(t, evalMathNode(t, &StartsWith{}, map[string]any{"Text": "中文abc", "Prefix": "中"}), true)
	wantBool(t, evalMathNode(t, &StartsWith{}, map[string]any{"Text": "abc", "Prefix": "b"}), false)
	wantBool(t, evalMathNode(t, &StartsWith{}, map[string]any{"Text": "abc", "Prefix": ""}), true) // 空前缀恒真
	wantBool(t, evalMathNode(t, &EndsWith{}, map[string]any{"Text": "abc文", "Suffix": "文"}), true)
	wantBool(t, evalMathNode(t, &EndsWith{}, map[string]any{"Text": "abc", "Suffix": ""}), true)
}

func TestStringNodes_SpecShape(t *testing.T) {
	for _, n := range []node.Node{&Replace{}, &Substring{}, &Trim{}, &ToUpper{}, &ToLower{}, &IndexOf{}, &StartsWith{}, &EndsWith{}} {
		s := n.Spec()
		if !s.IsPureData || s.Category != "PureFunc" {
			t.Fatalf("%s: must be IsPureData + PureFunc, got %+v", s.Kind, s)
		}
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/purefunc/ -run "TestReplace|TestSubstring|TestRune|TestTrim|TestIndexOf|TestStartsEnds|TestStringNodes" -v`
Expected: 编译失败（`Replace` 等未定义）。

- [ ] **Step 3: 写实现**

`internal/nodes/purefunc/strings.go`：

```go
// strings.go — 字符串函数节点 (10 个): Replace/Substring/Trim/ToUpper/ToLower/
// IndexOf/StartsWith/EndsWith/RegexMatch/RegexExtract. 见 specs/2026-06-10-string-nodes.md.
// 注册在 purefunc.go::init() "字符串函数 (10)" 组.
// 位置/长度语义 rune-based (CJK 一个字算 1); String 输入经 in.String (非字符串 → "").
// 正则非法 pattern 不返 error (pure-data error 被数据线路径吞) → 安全值 + ctx.Log().Warn.
package purefunc

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"yotta/internal/node"
)

// strTextIn 单个 String text 输入.
func strTextIn(name string) node.InputSpec {
	return node.InputSpec{Name: name, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}}
}

// ===== Replace =====

type Replace struct{}

func (Replace) Spec() node.Spec {
	return specBuilder("Replace", []node.InputSpec{
		strTextIn("Text"), strTextIn("Old"), strTextIn("New"),
		{Name: "All", Type: "Bool", Default: true, Widget: node.WidgetSpec{Kind: "checkbox"}},
	}, "String")
}

// Evaluate — Old=="" 原样返 (刻意偏离 Go ReplaceAll 的逐字符插入; All=false 时"第一个位置"同样无定义).
func (Replace) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	text, oldS, newS := in.String("Text"), in.String("Old"), in.String("New")
	if oldS == "" {
		return text, nil
	}
	if in.Bool("All") {
		return strings.ReplaceAll(text, oldS, newS), nil
	}
	return strings.Replace(text, oldS, newS, 1), nil
}

// ===== Substring =====

type Substring struct{}

func (Substring) Spec() node.Spec {
	return specBuilder("Substring", []node.InputSpec{
		strTextIn("Text"),
		{Name: "Start", Type: "Integer", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
		// Length 默认 -1 = 取到末尾 (Spec.Default 自动进 config.literal, 与 widget 无关永远生效).
		{Name: "Length", Type: "Integer", Default: json.Number("-1"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "String")
}

// Evaluate — rune-based: Start clamp [0,len]; Length<0 到末尾 / ==0 空串 / >0 取 N rune.
func (Substring) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	r := []rune(in.String("Text"))
	start := in.Int("Start")
	if start < 0 {
		start = 0
	}
	if start > len(r) {
		start = len(r)
	}
	length := in.Int("Length")
	if length == 0 {
		return "", nil
	}
	end := len(r)
	if length > 0 {
		end = start + length
		if end > len(r) {
			end = len(r)
		}
	}
	return string(r[start:end]), nil
}

// ===== Trim / ToUpper / ToLower =====

type Trim struct{}

func (Trim) Spec() node.Spec {
	return specBuilder("Trim", []node.InputSpec{strTextIn("Text")}, "String")
}
func (Trim) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return strings.TrimSpace(in.String("Text")), nil
}

type ToUpper struct{}

func (ToUpper) Spec() node.Spec {
	return specBuilder("ToUpper", []node.InputSpec{strTextIn("Text")}, "String")
}
func (ToUpper) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return strings.ToUpper(in.String("Text")), nil
}

type ToLower struct{}

func (ToLower) Spec() node.Spec {
	return specBuilder("ToLower", []node.InputSpec{strTextIn("Text")}, "String")
}
func (ToLower) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return strings.ToLower(in.String("Text")), nil
}

// ===== IndexOf =====

type IndexOf struct{}

func (IndexOf) Spec() node.Spec {
	return specBuilder("IndexOf", []node.InputSpec{strTextIn("Text"), strTextIn("Sub")}, "Number")
}

// Evaluate — rune 下标 (CJK 正确). 无匹配 -1; Sub=="" → 0 (Go 语义, 判"包含"用 Contains 节点).
func (IndexOf) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	text := in.String("Text")
	b := strings.Index(text, in.String("Sub"))
	if b < 0 {
		return float64(-1), nil
	}
	return float64(utf8.RuneCountInString(text[:b])), nil
}

// ===== StartsWith / EndsWith =====

type StartsWith struct{}

func (StartsWith) Spec() node.Spec {
	return specBuilder("StartsWith", []node.InputSpec{strTextIn("Text"), strTextIn("Prefix")}, "Bool")
}
func (StartsWith) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return strings.HasPrefix(in.String("Text"), in.String("Prefix")), nil
}

type EndsWith struct{}

func (EndsWith) Spec() node.Spec {
	return specBuilder("EndsWith", []node.InputSpec{strTextIn("Text"), strTextIn("Suffix")}, "Bool")
}
func (EndsWith) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return strings.HasSuffix(in.String("Text"), in.String("Suffix")), nil
}
```

（`regexp` import 本 Task 还没用到——若编译报 unused，先删掉、Task 3 再加回；或 Task 2 干脆不写 `regexp` 行，Task 3 加。）

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/purefunc/ -count=1`
Expected: PASS（全部）。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/purefunc/strings.go internal/nodes/purefunc/strings_test.go
git commit -m "feat(purefunc): 字符串节点 Replace/Substring/Trim/ToUpper/ToLower/IndexOf/StartsWith/EndsWith (rune-based)"
```

---

## Task 3: RegexMatch + RegexExtract

**Files:**
- Modify: `internal/nodes/purefunc/strings.go`
- Test: `internal/nodes/purefunc/strings_test.go`

- [ ] **Step 1: 写失败测试**

追加到 `strings_test.go`：

```go
func TestRegexMatch(t *testing.T) {
	// 搜索/包含语义 (非全文): "abc" 配 "b" 也中
	wantBool(t, evalMathNode(t, &RegexMatch{}, map[string]any{"Text": "abc", "Pattern": "b"}), true)
	wantBool(t, evalMathNode(t, &RegexMatch{}, map[string]any{"Text": "abc", "Pattern": "^b$"}), false)
	wantBool(t, evalMathNode(t, &RegexMatch{}, map[string]any{"Text": "户外 23 度", "Pattern": `\d+`}), true)
	// 非法 pattern → false (安全值, 不 error 不 panic; Warn 走 StubServices stdout)
	wantBool(t, evalMathNode(t, &RegexMatch{}, map[string]any{"Text": "abc", "Pattern": "("}), false)
}

func TestRegexExtract(t *testing.T) {
	// 无捕获组 → 整匹配
	wantStr(t, evalMathNode(t, &RegexExtract{}, map[string]any{"Text": "x=42;", "Pattern": `\d+`}), "42")
	// 有捕获组 → 组1 (多组只组1)
	wantStr(t, evalMathNode(t, &RegexExtract{}, map[string]any{"Text": "x=42;y=7", "Pattern": `x=(\d+);y=(\d+)`}), "42")
	// (?:) 不计组 → 整匹配
	wantStr(t, evalMathNode(t, &RegexExtract{}, map[string]any{"Text": "ab", "Pattern": "(?:a)b"}), "ab")
	// 空捕获组 → 空串 (与"无匹配"同输出, spec 已声明不区分)
	wantStr(t, evalMathNode(t, &RegexExtract{}, map[string]any{"Text": "ab", "Pattern": "a(x?)b"}), "")
	// 无匹配 → 空串
	wantStr(t, evalMathNode(t, &RegexExtract{}, map[string]any{"Text": "abc", "Pattern": `\d+`}), "")
	// 非法 pattern → 空串
	wantStr(t, evalMathNode(t, &RegexExtract{}, map[string]any{"Text": "abc", "Pattern": "("}), "")
}

func TestRegexNodes_SpecShape(t *testing.T) {
	m := (RegexMatch{}).Spec()
	if m.Outputs[0].Type != "Bool" {
		t.Fatalf("RegexMatch output want Bool, got %s", m.Outputs[0].Type)
	}
	e := (RegexExtract{}).Spec()
	if e.Outputs[0].Type != "String" {
		t.Fatalf("RegexExtract output want String, got %s", e.Outputs[0].Type)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/purefunc/ -run TestRegex -v`
Expected: 编译失败（`RegexMatch` 未定义）。

- [ ] **Step 3: 写实现**

追加到 `strings.go`（import 块补 `"regexp"`）：

```go
// ===== RegexMatch / RegexExtract =====
// 错误路径契约: 非法 pattern 返安全值 (false/"") + ctx.Log().Warn — 不返 error
// (pure-data error 被数据线路径静默吞). 编辑期 validator 对 literal pattern 报红
// (validator.go::validateRegexPattern), 运行时 Warn 兜动态 (连线) pattern.

type RegexMatch struct{}

func (RegexMatch) Spec() node.Spec {
	return specBuilder("RegexMatch", []node.InputSpec{strTextIn("Text"), strTextIn("Pattern")}, "Bool")
}

// Evaluate — 搜索/包含匹配 ("abc"+"b"→true); 全文匹配用户自己写 ^...$.
func (RegexMatch) Evaluate(ctx node.Ctx, in node.Inputs) (any, error) {
	pat := in.String("Pattern")
	ok, err := regexp.MatchString(pat, in.String("Text"))
	if err != nil {
		ctx.Log().Warn("RegexMatch: 非法 pattern %q: %v", pat, err)
		return false, nil
	}
	return ok, nil
}

type RegexExtract struct{}

func (RegexExtract) Spec() node.Spec {
	return specBuilder("RegexExtract", []node.InputSpec{strTextIn("Text"), strTextIn("Pattern")}, "String")
}

// Evaluate — 有捕获组取组1 (多组只组1, 命名组也按位置), 无组取整匹配; 无匹配/非法 → "".
// "" 不区分"匹配到空"与"没匹配" — 要判存在先用 RegexMatch (spec 已声明).
func (RegexExtract) Evaluate(ctx node.Ctx, in node.Inputs) (any, error) {
	pat := in.String("Pattern")
	re, err := regexp.Compile(pat)
	if err != nil {
		ctx.Log().Warn("RegexExtract: 非法 pattern %q: %v", pat, err)
		return "", nil
	}
	m := re.FindStringSubmatch(in.String("Text"))
	if m == nil {
		return "", nil
	}
	if len(m) > 1 {
		return m[1], nil
	}
	return m[0], nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/purefunc/ -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/purefunc/strings.go internal/nodes/purefunc/strings_test.go
git commit -m "feat(purefunc): RegexMatch/RegexExtract (安全值 + Log.Warn, 不返 error)"
```

---

## Task 4: init 注册 + 包 doc + i18n + catalog 绿

**Files:**
- Modify: `internal/nodes/purefunc/purefunc.go`
- Modify: `frontend/src/i18n/zh.ts`、`frontend/src/i18n/en.ts`

- [ ] **Step 1: init() 注册追加**

`purefunc.go::init()` 列表 `// 数学 (9)` 行后加：

```go
		// 字符串函数 (10)
		&Replace{}, &Substring{}, &Trim{}, &ToUpper{}, &ToLower{},
		&IndexOf{}, &StartsWith{}, &EndsWith{}, &RegexMatch{}, &RegexExtract{},
```

- [ ] **Step 2: 包 doc 数量更新**

`purefunc.go` 第 1 行 `31 个纯函数节点 (Add/Sub/.../Select + 数学 Abs/.../Sqrt)` 改为：

```go
// Package purefunc 41 个纯函数节点 (Add/.../Select + 数学 Abs/.../Sqrt + 字符串 Replace/.../RegexExtract) + Expr.
```

- [ ] **Step 3: zh.ts 加 10 节点块**

在 `// math` 区之后（`// random` 区之前）新建 `// string` 区：

```ts
    Replace: {
      label: '替换文本', description: '把文本里的 Old 换成 New。「全部替换」开着换所有，关掉只换第一处。Old 留空时原样返回。',
      input: { Text: { label: '文本' }, Old: { label: '找什么' }, New: { label: '换成什么' }, All: { label: '全部替换' } },
      output: { Result: { label: '结果' } },
    },
    Substring: {
      label: '截取文本', description: '从第 Start 个字符开始截 Length 个字符（中文一个字算 1 个）。Length 填 -1（默认）截到末尾，0 得空串。Start 超出范围得空串。',
      input: { Text: { label: '文本' }, Start: { label: '起点' }, Length: { label: '长度' } },
      output: { Result: { label: '结果' } },
    },
    Trim: {
      label: '去首尾空白', description: '去掉文本开头和结尾的空格、换行、制表符。',
      input: { Text: { label: '文本' } },
      output: { Result: { label: '结果' } },
    },
    ToUpper: {
      label: '转大写', description: '把英文字母全部转成大写。',
      input: { Text: { label: '文本' } },
      output: { Result: { label: '结果' } },
    },
    ToLower: {
      label: '转小写', description: '把英文字母全部转成小写。',
      input: { Text: { label: '文本' } },
      output: { Result: { label: '结果' } },
    },
    IndexOf: {
      label: '查找位置', description: 'Sub 在文本里第一次出现的位置（从 0 数，中文一个字算 1 个）。找不到得 -1。只想判断"包含吗"请用「包含」节点。',
      input: { Text: { label: '文本' }, Sub: { label: '找什么' } },
      output: { Result: { label: '位置' } },
    },
    StartsWith: {
      label: '开头是', description: '判断文本是否以 Prefix 开头。Prefix 留空恒为真。',
      input: { Text: { label: '文本' }, Prefix: { label: '开头' } },
      output: { Result: { label: '结果' } },
    },
    EndsWith: {
      label: '结尾是', description: '判断文本是否以 Suffix 结尾。Suffix 留空恒为真。',
      input: { Text: { label: '文本' }, Suffix: { label: '结尾' } },
      output: { Result: { label: '结果' } },
    },
    RegexMatch: {
      label: '正则匹配', description: `判断文本里是否有匹配正则表达式的部分（是"包含"式：abc 用 b 也算中）。要整串完全匹配，给表达式首尾加 ^ 和 {'$'}。表达式写错时结果恒为否，并记一条警告日志。`,
      input: { Text: { label: '文本' }, Pattern: { label: '正则表达式' } },
      output: { Result: { label: '结果' } },
    },
    RegexExtract: {
      label: '正则提取', description: '从文本里提取第一段匹配正则表达式的内容；表达式带括号分组时取第 1 组。没匹配到、或表达式写错时得空串（写错另记警告日志）。',
      input: { Text: { label: '文本' }, Pattern: { label: '正则表达式' } },
      output: { Result: { label: '结果' } },
    },
```

⚠ RegexMatch 的 description 含 `{'$'}` —— 这是 vue-i18n 对 `$` 的**转义写法**（见 checklist vue-i18n-message-compiler-traps），且整条用**反引号模板串**包（与现有 Expr.description 同款）。其余条目无特殊字符。照抄。

- [ ] **Step 4: en.ts 加镜像块**

```ts
    Replace: {
      label: 'Replace', description: 'Replace Old with New in the text. With "Replace all" on it replaces every occurrence, off only the first. Empty Old returns the text unchanged.',
      input: { Text: { label: 'Text' }, Old: { label: 'Find' }, New: { label: 'Replace with' }, All: { label: 'Replace all' } },
      output: { Result: { label: 'Result' } },
    },
    Substring: {
      label: 'Substring', description: 'Take Length characters starting at Start (a CJK character counts as 1). Length -1 (default) takes to the end, 0 gives an empty string. Out-of-range Start gives an empty string.',
      input: { Text: { label: 'Text' }, Start: { label: 'Start' }, Length: { label: 'Length' } },
      output: { Result: { label: 'Result' } },
    },
    Trim: {
      label: 'Trim', description: 'Remove spaces, newlines and tabs from both ends of the text.',
      input: { Text: { label: 'Text' } },
      output: { Result: { label: 'Result' } },
    },
    ToUpper: {
      label: 'To Upper', description: 'Convert letters to uppercase.',
      input: { Text: { label: 'Text' } },
      output: { Result: { label: 'Result' } },
    },
    ToLower: {
      label: 'To Lower', description: 'Convert letters to lowercase.',
      input: { Text: { label: 'Text' } },
      output: { Result: { label: 'Result' } },
    },
    IndexOf: {
      label: 'Index Of', description: 'Position of the first occurrence of Sub in the text (counting from 0, a CJK character counts as 1). -1 when not found. To just test "contains", use the Contains node.',
      input: { Text: { label: 'Text' }, Sub: { label: 'Find' } },
      output: { Result: { label: 'Position' } },
    },
    StartsWith: {
      label: 'Starts With', description: 'Whether the text starts with Prefix. Empty Prefix is always true.',
      input: { Text: { label: 'Text' }, Prefix: { label: 'Prefix' } },
      output: { Result: { label: 'Result' } },
    },
    EndsWith: {
      label: 'Ends With', description: 'Whether the text ends with Suffix. Empty Suffix is always true.',
      input: { Text: { label: 'Text' }, Suffix: { label: 'Suffix' } },
      output: { Result: { label: 'Result' } },
    },
    RegexMatch: {
      label: 'Regex Match', description: `Whether any part of the text matches the regular expression (search semantics: b matches abc). For a full match wrap the pattern in ^ and {'$'}. An invalid pattern always gives false and logs a warning.`,
      input: { Text: { label: 'Text' }, Pattern: { label: 'Pattern' } },
      output: { Result: { label: 'Result' } },
    },
    RegexExtract: {
      label: 'Regex Extract', description: 'Extract the first match of the regular expression; with capture groups, group 1 is taken. No match or an invalid pattern gives an empty string (invalid patterns also log a warning).',
      input: { Text: { label: 'Text' }, Pattern: { label: 'Pattern' } },
      output: { Result: { label: 'Result' } },
    },
```

- [ ] **Step 5: 生成 + 校验**

Run: `cd frontend && pnpm gen:node-i18n` → 期望 96 节点（86+10）。
Run: `cd frontend && pnpm i18n:check` → **`[compile]` 段必绿**（RegexMatch 的 `{'$'}` 转义正确性靠它兜）；parity OK；residue 28 已知。
Run: `go test ./internal/nodes/purefunc/ ./internal/catalog/... ./internal/node/... -count=1` → PASS（`TestNoPinNameSplit` 无新分裂）。

- [ ] **Step 6: Commit**

```bash
git add internal/nodes/purefunc/purefunc.go frontend/src/i18n/ internal/catalog/node-i18n.json
git commit -m "feat(purefunc): 注册 10 字符串节点 + zh/en i18n"
```

---

## Task 5: 编辑期正则校验 (validator)

**Files:**
- Modify: `internal/services/container/validator.go`
- Modify: `frontend/src/i18n/zh.ts` + `en.ts`（validation 块）
- Test: validator 测试所在文件（先 grep `CodeInvalidCronExpr` 或 `INVALID_CRON_EXPR` 找到现有 per-kind 校验测试文件与范式，镜像写）

- [ ] **Step 1: 写失败测试**

先找到现有 validator per-kind 测试（grep `validateCronConfig\|INVALID_CRON_EXPR` in `internal/services/container/*_test.go`），在同文件按既有范式追加（下面是意图模板，按实际范式适配——构造 GraphNode 的方式/断言 helper 以现有测试为准）：

```go
func TestValidateRegexPattern(t *testing.T) {
	// 非法 literal pattern → SeverityError + INVALID_REGEX_PATTERN
	bad := GraphNode{ID: "r1", Kind: "RegexMatch", Config: map[string]any{
		"literal": map[string]any{"Pattern": "("},
	}}
	errs := checkGraphPerKind([]GraphNode{bad}, []string{"main"}, true)
	if len(errs) != 1 || errs[0].Code != CodeInvalidRegexPattern || errs[0].Severity != SeverityError {
		t.Fatalf("want 1 INVALID_REGEX_PATTERN error, got %+v", errs)
	}

	// 合法 pattern → 无错
	good := GraphNode{ID: "r2", Kind: "RegexExtract", Config: map[string]any{
		"literal": map[string]any{"Pattern": `\d+`},
	}}
	if errs := checkGraphPerKind([]GraphNode{good}, []string{"main"}, true); len(errs) != 0 {
		t.Fatalf("valid pattern should pass, got %+v", errs)
	}

	// 空 pattern → 跳过 (准备连上游/没填, 同 Cron 惯例)
	empty := GraphNode{ID: "r3", Kind: "RegexMatch", Config: map[string]any{}}
	if errs := checkGraphPerKind([]GraphNode{empty}, []string{"main"}, true); len(errs) != 0 {
		t.Fatalf("empty pattern should be skipped, got %+v", errs)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/services/container/ -run TestValidateRegexPattern -v`
Expected: 编译失败（`CodeInvalidRegexPattern` 未定义）。

- [ ] **Step 3: validator.go 实现**

1. Code 常量块（validator.go:56-71 区域）加：

```go
	CodeInvalidRegexPattern  = "INVALID_REGEX_PATTERN"
```

2. `checkGraphPerKind` 的 switch 加（`case "Cron":` 后）：

```go
		case "RegexMatch", "RegexExtract":
			nodeErrs = validateRegexPattern(n)
```

3. `validateCronConfig` 附近加 helper（import 块补 `"regexp"`）：

```go
// validateRegexPattern 静态校验 RegexMatch/RegexExtract 的 inline literal Pattern.
// 空 = 用户准备连上游 / 还没填 → 跳过 (同 validateCronConfig 惯例);
// 动态来源 (上游 data edge) 编辑期不可知, 运行时节点自身 Log.Warn + 安全值兜.
func validateRegexPattern(n *GraphNode) []ValidationError {
	s := PinString(n, "Pattern")
	if s == "" {
		return nil
	}
	if _, err := regexp.Compile(s); err != nil {
		return []ValidationError{{
			Severity: SeverityError,
			Code:     CodeInvalidRegexPattern,
			NodeID:   n.ID,
			Params:   map[string]any{"pattern": s, "parseErr": err.Error()},
		}}
	}
	return nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/services/container/ -run TestValidateRegexPattern -v` → PASS。
Run: `go test ./internal/services/container/... -count=1` → 无新失败（fish-fixture 预存除外）。

- [ ] **Step 5: 前端 validation i18n**

`zh.ts` validation 块（`INVALID_CRON_EXPR` 行旁）加：

```ts
    INVALID_REGEX_PATTERN: '正则表达式无效: {pattern} ({parseErr})',
```

`en.ts` 同位置：

```ts
    INVALID_REGEX_PATTERN: 'Regular expression invalid: {pattern} ({parseErr})',
```

（此处 `{pattern}`/`{parseErr}` 是 vue-i18n 插值参数，与 INVALID_CRON_EXPR 同款，**合法**。）

- [ ] **Step 6: 前端校验**

Run: `cd frontend && pnpm typecheck && pnpm i18n:check`
Expected: typecheck 零错; parity/compile OK（residue 28 已知）。

- [ ] **Step 7: Commit**

```bash
git add internal/services/container/validator.go internal/services/container/*_test.go frontend/src/i18n/
git commit -m "feat(validator): RegexMatch/RegexExtract literal pattern 编辑期校验 (INVALID_REGEX_PATTERN)"
```

---

## Task 6: docs 同步 + 全量验证

**Files:**
- Modify: `flightdeck/docs/node-system-reference.md`（PureFunc 行）

- [ ] **Step 1: 节点目录更新**

PureFunc 行 32→42，按字母序插入 10 个新名：

```markdown
| **PureFunc** (42) | Abs, Add, And, Ceil, Clamp, Concat, Contains, Div, EndsWith, Eq, Expr, Floor, Gt, GtEq, IndexOf, Length, Lt, LtEq, Max, Min, Mod, Mul, Neg, Not, NotEq, Or, Pow, RegexExtract, RegexMatch, Replace, Round, Select, Sqrt, StartsWith, Sub, Substring, ToBool, ToLower, ToNumber, ToString, ToUpper, Trim — **全 PureData (Evaluator)** |
```

- [ ] **Step 2: 后端全绿**

Run: `go build ./... && go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... ./internal/services/container/... -count=1`
Expected: PASS（fish-fixture 预存除外）。

- [ ] **Step 3: catalog 自查**

Run: `task nodes`
Expected: 10 字符串节点在 PureFunc, pin/默认值对（Substring.Length 默认 -1）。

- [ ] **Step 4: 构建**

Run: `task build`
Expected: 成功。

- [ ] **Step 5: 真机 smoke（人工，留给用户）**

- **看什么**：「纯函数」分组出现 10 个字符串节点。
- **怎么验**：① 拖 Substring → Text=`中文abc`、起点 0、长度 2 → 连 Log → 跑 → 输出 `中文`。② 拖 RegexMatch → Pattern 填 `(` → 节点上出现红错「正则表达式无效」；改成 `\d+` 红错消失。
- **什么算过**：两条都对。

- [ ] **Step 6: Commit**

```bash
git add flightdeck/docs/node-system-reference.md
git commit -m "docs(nodes): node-system-reference PureFunc 目录 32→42 (字符串 10 节点)"
```

---

## Self-Review（写完计划的自查结论）

- **Spec 覆盖**：10 节点表逐行→Task 2/3（Replace All 开关+Old 空、Substring rune+Start/Length 全边界、IndexOf rune 下标+空 Sub、空前后缀恒真、Regex 搜索语义/组1/空组/非法安全值）；Length 改 rune+CJK 回归+rune 一致性集成用例→Task 1+2；正则编辑期校验（literal-only、空跳过、动态 runtime 兜）→Task 5（镜像 validateCronConfig，满足 spec"标准行为"）；运行时 Log.Warn→Task 3；i18n（含 `{'$'}` 转义与 compile 兜底）→Task 4；验证命令→Task 6。全覆盖。
- **类型一致**：`strTextIn` Task 2 定义、Task 3 复用；`evalMathNode`/`wantStr`/`wantBool` 跨文件同包复用；`CodeInvalidRegexPattern` 常量↔switch↔helper↔前端 key 串一致；IndexOf/Length 输出 Number（float64）与现有风格一致。
- **无占位符**：唯 Task 5 Step 1 标注"按现有 validator 测试范式适配"——GraphNode 构造与断言已给出完整可编译模板，适配仅限 helper 形式差异，意图三用例（非法/合法/空）固定。
- **已知非回归**：fish-fixture、i18n residue 28，已在步骤标注。
