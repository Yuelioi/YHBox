// validate-fishing-v2 加载 bin/data/library/subgraphs/fishing-v2/*.json 并跑 container ValidateContainer.
//
// 用法: go run ./cmd/validate-fishing-v2
//
// 当前阶段 F4 增量编写 14 个 subgraph, 主图 fishing-v2.json 尚未 ship — 临时把已 ship 的
// helper subgraphs 装进一个 minimal main graph (Start → 调用每个 helper → Stop) 跑 validator,
// 抓 INVALID_PIN / DANGLING_EDGE / MISSING_TEMPLATE / etc 早期错误.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"yhbox/internal/services/container"
)

const fishingV2Dir = "bin/data/library/subgraphs/fishing-v2"

// 当 fishing-v2.json (主图) 不存在时, 用 stub 把已 ship 的 helper subgraphs 各调用一次,
// 让 validator 走每一个内部节点的校验路径 (而不是孤立看 subgraph).
func buildStubMain(subgraphs []container.Subgraph) container.Container {
	c := container.Container{
		SchemaVersion: 1,
		ID:            "fishing-v2-validation-stub",
		Name:          "fishing-v2-validation-stub",
		Vars: []container.VarDecl{
			{Name: "_pressEscCleared", Type: "bool", Default: false},
			{Name: "state", Type: "string", Default: "IDLE"},
			{Name: "controlDir", Type: "number", Default: 0.0},
			{Name: "_hookFFound", Type: "bool", Default: false},
			{Name: "_inspectPhaseResult", Type: "string", Default: "unknown"},
			{Name: "waitingStart", Type: "number", Default: 0.0},
			{Name: "hookStreak", Type: "number", Default: 0.0},
			{Name: "fishingStart", Type: "number", Default: 0.0},
			{Name: "emptyCastStreak", Type: "number", Default: 0.0},
		},
		Subgraphs: subgraphs,
	}
	nodes := []container.GraphNode{
		{ID: "start", Kind: "Start", CreatedAt: time.Now()},
		{ID: "winTarget", Kind: "WindowTarget", Config: map[string]any{
			"match":   map[string]any{"title": "stub", "titleMatch": "exact"},
			"runtime": map[string]any{"inputBackend": "postmessage", "captureBackend": "auto"},
		}, CreatedAt: time.Now()},
	}
	edges := []container.GraphEdge{}
	prevID := "start"
	prevPin := "out"
	for i, sg := range subgraphs {
		callID := fmt.Sprintf("call_%d", i)
		// isAnonymous → 必须用 CollapsedNode; 否则 (常规子图) 用 Subgraph
		callKind := "Subgraph"
		if sg.IsAnonymous {
			callKind = "CollapsedNode"
		}
		nodes = append(nodes, container.GraphNode{
			ID:   callID,
			Kind: callKind,
			Config: map[string]any{
				"subgraphId": sg.ID,
			},
			CreatedAt: time.Now(),
		})
		edges = append(edges, container.GraphEdge{
			From: prevID + "." + prevPin,
			To:   callID + ".in",
		})
		// 子图每个 declared output pin 都接出, 让 validator 看见
		if len(sg.OutputPins) == 0 {
			fmt.Fprintf(os.Stderr, "warning: subgraph %q has no OutputPins — call site has no out edge\n", sg.ID)
			prevID = callID
			prevPin = "out" // dummy, 边可能 dangling
			continue
		}
		// 用第一个 output pin 串下去
		prevID = callID
		prevPin = sg.OutputPins[0].ID
	}
	nodes = append(nodes, container.GraphNode{ID: "stop", Kind: "Stop", CreatedAt: time.Now()})
	edges = append(edges, container.GraphEdge{From: prevID + "." + prevPin, To: "stop.in"})

	c.Graph = container.Graph{
		ID:      "fishing-v2-stub-graph",
		Version: 1,
		Nodes:   nodes,
		Edges:   edges,
	}
	return c
}

func main() {
	entries, err := os.ReadDir(fishingV2Dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open dir: %v\n", err)
		os.Exit(2)
	}
	var subgraphs []container.Subgraph
	var mainContainer *container.Container
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(fishingV2Dir, e.Name())
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", path, rerr)
			os.Exit(2)
		}
		if e.Name() == "fishing-v2.json" {
			var c container.Container
			if err := json.Unmarshal(data, &c); err != nil {
				fmt.Fprintf(os.Stderr, "unmarshal main %s: %v\n", path, err)
				os.Exit(2)
			}
			mainContainer = &c
		} else {
			var sg container.Subgraph
			if err := json.Unmarshal(data, &sg); err != nil {
				fmt.Fprintf(os.Stderr, "unmarshal subgraph %s: %v\n", path, err)
				os.Exit(2)
			}
			subgraphs = append(subgraphs, sg)
		}
	}

	sort.Slice(subgraphs, func(i, j int) bool { return subgraphs[i].ID < subgraphs[j].ID })

	if mainContainer != nil {
		mainContainer.Subgraphs = subgraphs
	} else {
		stub := buildStubMain(subgraphs)
		mainContainer = &stub
		fmt.Fprintf(os.Stderr, "INFO: fishing-v2.json missing — using stub main with %d helper subgraphs\n", len(subgraphs))
	}

	errs := container.ValidateContainer(mainContainer)
	if len(errs) == 0 {
		fmt.Println("✅ All clean — 0 validation errors")
		return
	}
	// 按 severity 排
	sort.SliceStable(errs, func(i, j int) bool {
		if errs[i].Severity != errs[j].Severity {
			return errs[i].Severity > errs[j].Severity
		}
		return errs[i].Code < errs[j].Code
	})
	hasError := false
	for _, e := range errs {
		if e.Severity == container.SeverityError {
			hasError = true
		}
		path := strings.Join(e.GraphPath, "/")
		if path == "" {
			path = "(root)"
		}
		msg := e.Message
		if msg == "" && len(e.Params) > 0 {
			b, _ := json.Marshal(e.Params)
			msg = string(b)
		}
		fmt.Printf("- [%s] %s @ %s/%s — %s\n", e.Severity, e.Code, path, e.NodeID, msg)
	}
	if hasError {
		os.Exit(1)
	}
}
