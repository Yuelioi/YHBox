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

	"github.com/yottaapp/yotta/internal/node"
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

// ===== RegexMatch / RegexExtract =====
// 错误路径契约: 非法 pattern 返安全值 (false/"") + ctx.Log().Warn — 不返 error
// (pure-data error 被数据线路径静默吞). 编辑期 validator 对 literal pattern 报红
// (validator.go::validateRegexPattern), 运行时 Warn 兜动态 (连线) pattern.

type RegexMatch struct{}

func (RegexMatch) Spec() node.Spec {
	return withRuntimeCapabilities(specBuilder("RegexMatch", []node.InputSpec{strTextIn("Text"), strTextIn("Pattern")}, "Bool"), node.RuntimeCapabilityLog)
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
	return withRuntimeCapabilities(specBuilder("RegexExtract", []node.InputSpec{strTextIn("Text"), strTextIn("Pattern")}, "String"), node.RuntimeCapabilityLog)
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
