package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	gort "runtime"
	"testing"
	"time"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
)

func loadFishingV2Main(t *testing.T) *container.Container {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	containerDir := filepath.Join(root, "bin", "data", "containers", "fishing-v2")

	mainData, err := os.ReadFile(filepath.Join(containerDir, "container.json"))
	if err != nil {
		t.Fatalf("read container.json: %v", err)
	}
	var c container.Container
	if err := json.Unmarshal(mainData, &c); err != nil {
		t.Fatalf("unmarshal container.json: %v", err)
	}

	sgDir := filepath.Join(containerDir, "subgraphs")
	entries, err := os.ReadDir(sgDir)
	if err != nil {
		t.Fatalf("read subgraphs dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(sgDir, e.Name()))
		if rerr != nil {
			t.Fatalf("read %s: %v", e.Name(), rerr)
		}
		var sg container.Subgraph
		if err := json.Unmarshal(data, &sg); err != nil {
			t.Fatalf("unmarshal %s: %v", e.Name(), err)
		}
		c.Subgraphs = append(c.Subgraphs, sg)
	}
	// B2: 老 fishing-v2 JSON 含 SubgraphInput/Output 节点, 走 Normalize 迁移到 metadata 后 validator 才认.
	c.Normalize()
	return &c
}

func TestFishingV2Main_Validates(t *testing.T) {
	c := loadFishingV2Main(t)
	errs := container.ValidateContainer(c)
	for _, e := range errs {
		if e.Severity == container.SeverityError {
			t.Errorf("validator error: code=%s nodeID=%s params=%v", e.Code, e.NodeID, e.Params)
		}
	}
}

func TestFishingV2Main_StateCycleSmoke(t *testing.T) {
	c := loadFishingV2Main(t)
	for i, n := range c.Graph.Nodes {
		// Subgraph 参数 override (state machine internal delays).
		if n.Kind == "Subgraph" {
			sgID, _ := n.Config["SubgraphID"].(string)
			switch sgID {
			case "state_IDLE":
				c.Graph.Nodes[i].Config["literal"] = map[string]any{
					"baitProbeDelayMs":     1.0,
					"castRemainingDelayMs": 1.0,
				}
			case "state_SETUP":
				c.Graph.Nodes[i].Config["literal"] = map[string]any{
					"postClickDelayMs": 1.0,
				}
			}
		}
		// Main loop Sleep override — 老 fixture 用 sleepIdle.Duration=500ms 但 test 总 timeout
		// 也 500ms, 一轮没跑完. 改 1ms 让 IDLE→SETUP 同 cycle 跑完.
		if n.Kind == "Sleep" {
			lit, _ := c.Graph.Nodes[i].Config["literal"].(map[string]any)
			if lit == nil {
				lit = map[string]any{}
				c.Graph.Nodes[i].Config["literal"] = lit
			}
			lit["Duration"] = 1.0
		}
	}

	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	spy := &spyInputBackend{}
	rt.Input = spy
	rt.Matcher = &mockMatcher{HitTemplates: map[string]bool{"fishing.start_fish": true}}
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx)

	if spy.clicks == 0 {
		t.Errorf("StateCycleSmoke: expected at least 1 click (state_SETUP ran), got 0")
	}
	finalState, _ := rt.Vars()["state"].(string)
	if finalState != "IDLE" && finalState != "SETUP" {
		t.Errorf("StateCycleSmoke: expected state ∈ {IDLE, SETUP}, got %q", finalState)
	}
	t.Logf("StateCycleSmoke: clicks=%d finalState=%q (after 500ms)", spy.clicks, finalState)
}
