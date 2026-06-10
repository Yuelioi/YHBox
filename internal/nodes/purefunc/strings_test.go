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
