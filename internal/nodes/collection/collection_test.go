package collection

import (
	"context"
	"testing"

	"yotta/internal/node"
)

// evalNode — EvaluatePureData 路径 (同 purefunc math_test 范式).
func evalNode(t *testing.T, n node.Node, dataWire map[string]any) any {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(n)
	rn, ok := node.Get(n.Spec().Kind)
	if !ok {
		t.Fatalf("kind %q not registered", n.Spec().Kind)
	}
	got, err := node.EvaluatePureData(context.Background(), rn, dataWire, nil, node.StubServices())
	if err != nil {
		t.Fatalf("EvaluatePureData err: %v", err)
	}
	return got
}

func wantList(t *testing.T, got any, want []any) {
	t.Helper()
	l, ok := got.([]any)
	if !ok {
		t.Fatalf("want []any, got %T(%v)", got, got)
	}
	if len(l) != len(want) {
		t.Fatalf("len = %d (%v), want %d (%v)", len(l), l, len(want), want)
	}
	for i := range l {
		if l[i] != want[i] {
			t.Fatalf("[%d] = %v, want %v", i, l[i], want[i])
		}
	}
}

func TestSplit(t *testing.T) {
	wantList(t, evalNode(t, &Split{}, map[string]any{"Text": "a,b,c", "Separator": ","}), []any{"a", "b", "c"})
	// Text="" → 空列表 (刻意偏离 Go Split("",sep)=[""])
	wantList(t, evalNode(t, &Split{}, map[string]any{"Text": "", "Separator": ","}), []any{})
	// Separator="" → 按 UTF-8 字符逐个拆 (CJK 安全)
	wantList(t, evalNode(t, &Split{}, map[string]any{"Text": "中a", "Separator": ""}), []any{"中", "a"})
	// 默认分隔符 ","
	wantList(t, evalNode(t, &Split{}, map[string]any{"Text": "x,y"}), []any{"x", "y"})
}

func TestJoin(t *testing.T) {
	got := evalNode(t, &Join{}, map[string]any{"List": []any{"a", 1.5, true, nil}, "Separator": "-"})
	if got != "a-1.5-true-null" {
		t.Fatalf("Join = %v, want a-1.5-true-null", got)
	}
	if got := evalNode(t, &Join{}, map[string]any{"List": []any{}}); got != "" {
		t.Fatalf("empty list join = %v, want \"\"", got)
	}
}

func TestListLength(t *testing.T) {
	if got := evalNode(t, &ListLength{}, map[string]any{"List": []any{1, 2, 3}}); got != 3.0 {
		t.Fatalf("len = %v, want 3", got)
	}
	// 非列表 → in.List 返 nil → 0
	if got := evalNode(t, &ListLength{}, map[string]any{"List": "not a list"}); got != 0.0 {
		t.Fatalf("non-list len = %v, want 0", got)
	}
}

func TestListGet(t *testing.T) {
	lst := []any{"a", nil, "c"}
	if got := evalNode(t, &ListGet{}, map[string]any{"List": lst, "Index": 0}); got != "a" {
		t.Fatalf("get[0] = %v", got)
	}
	// 元素本身是 nil → nil (与越界同输出, spec 已声明歧义)
	if got := evalNode(t, &ListGet{}, map[string]any{"List": lst, "Index": 1}); got != nil {
		t.Fatalf("get[1] = %v, want nil", got)
	}
	// 越界 / 负索引 → nil
	if got := evalNode(t, &ListGet{}, map[string]any{"List": lst, "Index": 99}); got != nil {
		t.Fatalf("get[99] = %v, want nil", got)
	}
	if got := evalNode(t, &ListGet{}, map[string]any{"List": lst, "Index": -1}); got != nil {
		t.Fatalf("get[-1] = %v, want nil (不做负索引)", got)
	}
}

func TestCollection_SpecShape(t *testing.T) {
	for _, n := range []node.Node{&Split{}, &Join{}, &ListLength{}, &ListGet{}} {
		s := n.Spec()
		if !s.IsPureData || s.Category != "List" {
			t.Fatalf("%s: must be IsPureData + Category List, got %+v", s.Kind, s)
		}
	}
}
