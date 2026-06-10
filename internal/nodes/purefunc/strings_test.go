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
