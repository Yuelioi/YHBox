package winutil

import (
	"context"
	"regexp"
	"testing"
	"time"
)

func TestIsEmptyMatch(t *testing.T) {
	cases := []struct {
		name string
		spec MatchSpec
		want bool
	}{
		{"all empty", MatchSpec{}, true},
		{"title only", MatchSpec{Title: "异环"}, false},
		{"class only", MatchSpec{Class: "Unreal"}, false},
		{"process only", MatchSpec{ProcessName: "game.exe"}, false},
		{"regex dot star alone", MatchSpec{Title: ".*", TitleMatch: "regex"}, true},
		{"regex dot plus alone", MatchSpec{Title: ".+", TitleMatch: "regex"}, true},
		{"regex anchored dot star alone", MatchSpec{Title: "^.*$", TitleMatch: "regex"}, true},
		{"regex .* with class", MatchSpec{Title: ".*", TitleMatch: "regex", Class: "Unreal"}, false},
		{"regex specific", MatchSpec{Title: "异环", TitleMatch: "regex"}, false},
		{"exact .*", MatchSpec{Title: ".*", TitleMatch: "exact"}, false}, // exact 模式下 .* 字面意义不是万能
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsEmptyMatch(c.spec); got != c.want {
				t.Errorf("IsEmptyMatch(%+v) = %v, want %v", c.spec, got, c.want)
			}
		})
	}
}

func TestCompileTitle(t *testing.T) {
	if _, err := CompileTitle(MatchSpec{Title: "异环", TitleMatch: "exact"}); err != nil {
		t.Errorf("exact mode should not compile: %v", err)
	}
	if r, err := CompileTitle(MatchSpec{Title: "异环.*", TitleMatch: "regex"}); err != nil || r == nil {
		t.Errorf("valid regex should compile, got err=%v r=%v", err, r)
	}
	if _, err := CompileTitle(MatchSpec{Title: "[invalid", TitleMatch: "regex"}); err == nil {
		t.Errorf("invalid regex should fail")
	}
}

func TestEnumTopWindows_ReturnsVisibleWindows(t *testing.T) {
	got := EnumTopWindows()
	// 测试进程跑在桌面会话里, 至少应枚到若干窗口 (CI headless 可能为空 → 只断言不 panic + 字段完整性).
	for _, w := range got {
		if w.HWND == 0 {
			t.Fatalf("EnumTopWindows 返回了 HWND==0 的条目: %+v", w)
		}
	}
}

func TestResolveWindow_CtxCancelledReturnsPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消
	spec := MatchSpec{Title: "definitely-no-such-window-zzz", TitleMatch: "exact"}
	start := time.Now()
	_, err := ResolveWindow(ctx, spec, 3*time.Second, 500*time.Millisecond)
	if err == nil {
		t.Fatal("want error on cancelled ctx, got nil")
	}
	if time.Since(start) > 1*time.Second {
		t.Fatalf("ResolveWindow ignored ctx cancel, waited %v", time.Since(start))
	}
}

func TestExecutableWindowQueryUsesRegistryToken(t *testing.T) {
	want := &executableWindowQuery{title: "Editor", class: "Window"}
	var got *executableWindowQuery
	var state uintptr
	withExecutableWindowQuery(want, func(token uintptr) {
		state = token
		got = executableWindowQueryFromState(token)
	})
	if got != want {
		t.Fatalf("callback query = %p, want %p", got, want)
	}
	if query := executableWindowQueryFromState(state); query != nil {
		t.Fatalf("callback query remained registered: %p", query)
	}
}

func TestExecutableWindowTitleMatcherSupportsExactAndRegex(t *testing.T) {
	exact := &executableWindowQuery{title: "异环  ", titleMatch: "exact"}
	if !exact.matchesTitle("异环  ") || exact.matchesTitle("异环") {
		t.Fatal("exact title mode did not preserve raw Win32 whitespace")
	}
	pattern, err := regexp.Compile(`^异环\s*$`)
	if err != nil {
		t.Fatal(err)
	}
	regex := &executableWindowQuery{title: `^异环\s*$`, titleMatch: "regex", titleRegex: pattern}
	if !regex.matchesTitle("异环  ") || regex.matchesTitle("异环 - 登录") {
		t.Fatal("regex title mode did not use the installed expression")
	}
}
