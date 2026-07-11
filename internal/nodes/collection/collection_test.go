package collection

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

// evalNode — EvaluatePureData 路径 (同 purefunc math_test 范式).
func evalNode(t *testing.T, n node.Node, dataWire map[string]any) any {
	t.Helper()
	registry := node.NewRegistry()
	registry.Register(n)
	rn, ok := registry.Get(n.Spec().Kind)
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
	for _, n := range []node.Node{&Split{}, &Join{}, &ListLength{}, &ListGet{}, &ListContains{}, &ListAppend{}, &ListSlice{}} {
		s := n.Spec()
		if !s.IsPureData || s.Category != "List" {
			t.Fatalf("%s: must be IsPureData + Category List, got %+v", s.Kind, s)
		}
	}
}

func TestListContains(t *testing.T) {
	lst := []any{1.0, "b", nil}
	cases := []struct {
		name  string
		value any
		want  bool
	}{
		{"same_type", "b", true},
		{"cross_type_strcmp", "1", true}, // 与 Eq 节点同语义: 跨类型串比
		{"nil_element", nil, true},       // LooseEqual(nil,nil)=true
		{"absent", "x", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalNode(t, &ListContains{}, map[string]any{"List": lst, "Value": tc.value})
			if got != tc.want {
				t.Fatalf("Contains(%v) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
	// 嵌套子列表元素 — 防护回归 (直比 panic 路径), 串比语义
	nested := []any{[]any{1, 2}}
	if got := evalNode(t, &ListContains{}, map[string]any{"List": nested, "Value": []any{1, 2}}); got != true {
		t.Fatalf("nested contains = %v, want true (串比)", got)
	}
	// nil vs "" 不等 (FormatValue(nil)="null")
	if got := evalNode(t, &ListContains{}, map[string]any{"List": []any{nil}, "Value": ""}); got != false {
		t.Fatalf(`Contains([nil], "") = %v, want false`, got)
	}
}

func TestListAppend(t *testing.T) {
	orig := []any{"a"}
	got := evalNode(t, &ListAppend{}, map[string]any{"List": orig, "Item": "b"})
	wantList(t, got, []any{"a", "b"})
	// 必 copy: 原列表不被改 (防 append 别名上游切片)
	if len(orig) != 1 || orig[0] != "a" {
		t.Fatalf("orig mutated: %v", orig)
	}
	// 空/非列表 → 单元素新列表
	wantList(t, evalNode(t, &ListAppend{}, map[string]any{"List": nil, "Item": "x"}), []any{"x"})
}

func TestListSlice(t *testing.T) {
	lst := []any{"a", "b", "c", "d"}
	// Count 默认 -1 → 取到末尾
	wantList(t, evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": 1}), []any{"b", "c", "d"})
	// Count=0 → 空
	wantList(t, evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": 1, "Count": 0}), []any{})
	// Count>0 → N 个, 超尾截断
	wantList(t, evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": 2, "Count": 99}), []any{"c", "d"})
	// Start>=len → 恒空 (Count 忽略)
	wantList(t, evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": 9, "Count": 2}), []any{})
	// 负 Start → 0
	wantList(t, evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": -3, "Count": 2}), []any{"a", "b"})
	// copy: 改结果不影响原列表 — 用长度断言间接验 (返回的是新底层数组)
	got := evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": 0, "Count": 2}).([]any)
	got[0] = "Z"
	if lst[0] != "a" {
		t.Fatalf("slice aliased original: %v", lst)
	}
}
