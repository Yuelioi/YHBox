// specs_test.go — P0.2 验 4 个 extractor 读 PascalCase cfg key (atomic #3 redraw 后).
package specs

import (
	"reflect"
	"testing"

	"yhbox/internal/services/container/dependency"
)

func TestSubgraphExtractor_PascalCase(t *testing.T) {
	ext := dependency.GetExtractor("Subgraph")
	if ext == nil {
		t.Fatal("Subgraph extractor not registered")
	}
	got := ext.Extract(map[string]any{"SubgraphID": "callee-x"})
	want := []dependency.Dependency{{Kind: dependency.KindSubgraph, Key: "callee-x"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// lowercase 老 key 不再认 (atomic #5 follow-up 切干净)
	if dep := ext.Extract(map[string]any{"subgraphId": "old"}); len(dep) != 0 {
		t.Errorf("lowercase subgraphId should not match, got %v", dep)
	}
}

func TestTemplateRefExtractor_PascalCase(t *testing.T) {
	for _, kind := range []string{"WaitTemplate", "CheckTemplate", "ClickTemplate"} {
		ext := dependency.GetExtractor(kind)
		if ext == nil {
			t.Fatalf("%s extractor not registered", kind)
		}
		got := ext.Extract(map[string]any{"Template": "fishing.hook_icon"})
		want := []dependency.Dependency{{Kind: dependency.KindTemplate, Key: "fishing.hook_icon"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v, want %v", kind, got, want)
		}
		if dep := ext.Extract(map[string]any{"template": "old"}); len(dep) != 0 {
			t.Errorf("%s lowercase template should not match, got %v", kind, dep)
		}
	}
}

func TestOnEventExtractor_PascalCase(t *testing.T) {
	ext := dependency.GetExtractor("OnEvent")
	if ext == nil {
		t.Fatal("OnEvent extractor not registered")
	}
	// kind=template_appeared + Template wired → 抽 template 依赖
	got := ext.Extract(map[string]any{"Kind": "template_appeared", "Template": "fishing.hook_icon"})
	want := []dependency.Dependency{{Kind: dependency.KindTemplate, Key: "fishing.hook_icon"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// 别的 kind → 不抽
	if dep := ext.Extract(map[string]any{"Kind": "stopwatch_done", "Template": "ignored"}); len(dep) != 0 {
		t.Errorf("non-template_appeared kind should not extract, got %v", dep)
	}
	// 老 lowercase 不认
	if dep := ext.Extract(map[string]any{"kind": "template_appeared", "template": "old"}); len(dep) != 0 {
		t.Errorf("lowercase kind/template should not match, got %v", dep)
	}
}

func TestPlayClipExtractor_PascalCase(t *testing.T) {
	ext := dependency.GetExtractor("PlayClip")
	if ext == nil {
		t.Fatal("PlayClip extractor not registered")
	}
	got := ext.Extract(map[string]any{"ClipID": "intro-recording"})
	want := []dependency.Dependency{{Kind: dependency.KindClip, Key: "intro-recording"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if dep := ext.Extract(map[string]any{"clipID": "old"}); len(dep) != 0 {
		t.Errorf("lowercase clipID should not match, got %v", dep)
	}
}
