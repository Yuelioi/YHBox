package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/catalog"
)

func TestDoExport_ProducesMachineAndHumanCatalogs(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		output := captureStdout(t, func() { doExport(false) })
		var nodes []catalog.Node
		if err := json.Unmarshal([]byte(output), &nodes); err != nil {
			t.Fatalf("doExport(false) produced invalid JSON: %v", err)
		}
		if len(nodes) == 0 {
			t.Fatal("JSON catalog is empty")
		}
		if !containsNodeWithText(nodes, "WindowState") {
			t.Fatal("WindowState is missing its user-facing catalog text")
		}
	})

	t.Run("markdown", func(t *testing.T) {
		output := captureStdout(t, func() { doExport(true) })
		for _, want := range []string{"# 节点速查表", "### WindowState", "| pin | 类型 | 必填 | 默认 | 说明 |"} {
			if !strings.Contains(output, want) {
				t.Fatalf("Markdown catalog missing %q", want)
			}
		}
	})
}

func TestDoPins_ProducesDeterministicNamingReference(t *testing.T) {
	first := captureStdout(t, doPins)
	second := captureStdout(t, doPins)
	if first != second {
		t.Fatal("pin catalog output is not deterministic")
	}
	for _, want := range []string{
		"# 节点 pin 命名参考 (合并)",
		"## 输入参数 (config)",
		"## 输入 exec 口",
		"## 出口 exec",
		"## 出口数据字段 (exec-data)",
		"## 纯数据输出",
		"## ⚠ 命名分裂告警",
		"| `In` |",
		"(无)",
	} {
		if !strings.Contains(first, want) {
			t.Fatalf("pin catalog missing %q", want)
		}
	}
}

func TestAddPin_AggregatesTypesNodesAndFirstLabel(t *testing.T) {
	pins := newPinMap()
	addPin(pins, "Point", "Point", "坐标", "DetectColor")
	addPin(pins, "Point", "String", "later label", "DetectColor")
	addPin(pins, "Point", "Point", "", "DetectColor")

	got := pins["Point"]
	if got == nil {
		t.Fatal("Point aggregate missing")
	}
	if got.label != "坐标" {
		t.Fatalf("label = %q, want first non-empty label", got.label)
	}
	if !reflect.DeepEqual(sortedSet(got.types), []string{"Point", "String"}) {
		t.Fatalf("types = %v", sortedSet(got.types))
	}
	if !reflect.DeepEqual(sortedSet(got.nodes), []string{"DetectColor"}) {
		t.Fatalf("nodes = %v", sortedSet(got.nodes))
	}
	if !reflect.DeepEqual(sortedPinKeys(pins), []string{"Point"}) {
		t.Fatalf("keys = %v", sortedPinKeys(pins))
	}
}

func containsNodeWithText(nodes []catalog.Node, kind string) bool {
	for _, node := range nodes {
		if node.Kind == kind {
			return node.Label != "" && node.Description != ""
		}
	}
	return false
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	file, err := os.CreateTemp(t.TempDir(), "stdout-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = file
	defer func() { os.Stdout = original }()

	fn()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = original
	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
