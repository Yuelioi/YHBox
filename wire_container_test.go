package main

import (
	"path/filepath"
	"testing"

	"yhbox/internal/services/template"
)

func TestMatcherInvalidate_PicksUpNewVariant(t *testing.T) {
	root := t.TempDir()
	cid := "c1"
	tplDir := filepath.Join(root, "containers", cid, "templates")
	m := newTemplateMatcherAdapter(root, nil, nil)

	// matcher 先缓存一个空 store (模拟用户存模板前已跑过一次容器).
	st1, err := m.storeFor(cid)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st1.PickBest("ns.k", 1920, 1080); ok {
		t.Fatal("empty store should miss")
	}

	// 经独立 store 往磁盘存一个 variant (模拟 templateSvc 在另一 Store 实例上存模板).
	saver, err := template.NewStore(tplDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := saver.SaveMeta("ns.k", template.KeyMeta{Name: "K"}); err != nil {
		t.Fatal(err)
	}
	if _, err := saver.SaveVariant("ns.k", []byte{0x89, 0x50}, template.VariantMeta{Resolution: [2]int{1920, 1080}}); err != nil {
		t.Fatal(err)
	}

	// 复现 staleness: matcher 仍拿旧缓存 store → 还是 miss.
	st2, _ := m.storeFor(cid)
	if _, ok := st2.PickBest("ns.k", 1920, 1080); ok {
		t.Fatal("stale cached store should still miss before Invalidate")
	}

	// 失效后重读磁盘 → 命中.
	m.Invalidate(cid)
	st3, _ := m.storeFor(cid)
	if _, ok := st3.PickBest("ns.k", 1920, 1080); !ok {
		t.Error("after Invalidate, store should pick up the newly-saved variant")
	}
}

func TestLongEdgeScale(t *testing.T) {
	cases := []struct {
		fw, fh, vw, vh int
		want           float64
	}{
		{1920, 1080, 1920, 1080, 1.0},
		{2560, 1440, 1280, 720, 2.0},
		{1280, 720, 2560, 1440, 0.5},
		{1366, 768, 1280, 720, 1366.0 / 1280},
		{1920, 1080, 0, 0, 0},
	}
	for _, c := range cases {
		if got := longEdgeScale(c.fw, c.fh, c.vw, c.vh); got != c.want {
			t.Errorf("longEdgeScale(%d,%d,%d,%d) = %v, want %v", c.fw, c.fh, c.vw, c.vh, got, c.want)
		}
	}
}

func TestWithinScaleTolerance(t *testing.T) {
	cases := []struct {
		scale, k float64
		want     bool
	}{
		{1.0, 2.0, true},
		{2.0, 2.0, true},
		{0.5, 2.0, true},
		{2.5, 2.0, false},
		{0.4, 2.0, false},
		{1.2, 1.0, false},
		{1.0, 1.0, true},
		{0, 2.0, false},
		{1.0, 0.5, true},  // k<1 归一到 1 → 仅精确放行
		{1.2, 0.5, false}, // k<1 归一到 1 → 非精确判否
	}
	for _, c := range cases {
		if got := withinScaleTolerance(c.scale, c.k); got != c.want {
			t.Errorf("withinScaleTolerance(%v,%v) = %v, want %v", c.scale, c.k, got, c.want)
		}
	}
}
