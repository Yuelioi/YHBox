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
	dir := filepath.Join(root, "bin", "data", "library", "subgraphs", "fishing-v2")

	mainPath := filepath.Join(dir, "fishing-v2.json")
	mainData, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read fishing-v2.json: %v", err)
	}
	var c container.Container
	if err := json.Unmarshal(mainData, &c); err != nil {
		t.Fatalf("unmarshal fishing-v2.json: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() == "fishing-v2.json" {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			t.Fatalf("read %s: %v", e.Name(), rerr)
		}
		var sg container.Subgraph
		if err := json.Unmarshal(data, &sg); err != nil {
			t.Fatalf("unmarshal %s: %v", e.Name(), err)
		}
		c.Subgraphs = append(c.Subgraphs, sg)
	}
	return &c
}

func TestFishingV2Main_Validates(t *testing.T) {
	c := loadFishingV2Main(t)
	errs := container.ValidateContainer(c)
	for _, e := range errs {
		if e.Severity == container.SeverityError {
			t.Errorf("validator error: code=%s nodeID=%s msg=%s", e.Code, e.NodeID, e.Message)
		}
	}
}

func TestFishingV2Main_StateCycleSmoke(t *testing.T) {
	c := loadFishingV2Main(t)
	for i, n := range c.Graph.Nodes {
		if n.Kind != "Subgraph" {
			continue
		}
		sgID, _ := n.Config["subgraphId"].(string)
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
