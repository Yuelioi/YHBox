// internal/nodes/detect/find_template_all_test.go
package detect

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func runFTA(t *testing.T, vision *mockVision, cfg map[string]any) node.RunResult {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&FindTemplateAll{})
	rn, _ := node.Get("FindTemplateAll")
	base := map[string]any{ftaInTemplates: []string{"tpl.a"}}
	for k, v := range cfg {
		base[k] = v
	}
	return node.RunNode(context.Background(), rn, nil, base, nil, withVision(vision), false)
}

func ftaMatch(x, y, conf float64, key string) node.TemplateMatch {
	return node.TemplateMatch{Point: node.Point{X: x, Y: y}, Conf: conf, TemplateKey: key}
}

func threeMatches() []node.TemplateMatch {
	return []node.TemplateMatch{
		ftaMatch(0.5, 0.5, 0.95, "a"),
		ftaMatch(0.2, 0.3, 0.90, "a"),
		ftaMatch(0.8, 0.7, 0.88, "b"),
	}
}

func TestFindTemplateAll_FoundCountAndPrimary(t *testing.T) {
	r := runFTA(t, &mockVision{matchAllResults: threeMatches()}, nil)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != ftaOutFound {
		t.Fatalf("exit=%q want Found", r.ExitName)
	}
	if r.OutputData[ftaDataCount] != 3 {
		t.Errorf("Count=%v want 3", r.OutputData[ftaDataCount])
	}
	pp, ok := r.OutputData[ftaDataPrimaryPt].(node.Point)
	if !ok || pp.X != 0.5 {
		t.Errorf("PrimaryPoint=%v want highest-conf (0.5,0.5)", r.OutputData[ftaDataPrimaryPt])
	}
	if r.OutputData[ftaDataPrimaryCf] != 0.95 {
		t.Errorf("PrimaryConf=%v want 0.95", r.OutputData[ftaDataPrimaryCf])
	}
}

func TestFindTemplateAll_MaxResultsTruncatesListNotCount(t *testing.T) {
	r := runFTA(t, &mockVision{matchAllResults: threeMatches()}, map[string]any{ftaInMaxResults: 2})
	if r.OutputData[ftaDataCount] != 3 {
		t.Errorf("Count=%v want 3 (total, NOT truncated by MaxResults)", r.OutputData[ftaDataCount])
	}
	ms, ok := r.OutputData[ftaDataMatches].([]node.TemplateMatch)
	if !ok || len(ms) != 2 {
		t.Fatalf("Matches len=%d want 2 (list truncated to MaxResults)", len(ms))
	}
	// primary (最高分) 仍在截断后列表首位
	if ms[0].Point.X != 0.5 {
		t.Errorf("Matches[0]=%v want primary (0.5,0.5)", ms[0].Point)
	}
}

func TestFindTemplateAll_NotFoundWhenEmpty(t *testing.T) {
	r := runFTA(t, &mockVision{matchAllResults: nil}, nil)
	if r.ExitName != ftaOutNotFound {
		t.Fatalf("exit=%q want NotFound", r.ExitName)
	}
	if r.OutputData[ftaDataCount] != 0 {
		t.Errorf("Count=%v want 0", r.OutputData[ftaDataCount])
	}
}

func TestFindTemplateAll_ErrorPropagates(t *testing.T) {
	r := runFTA(t, &mockVision{matchAllErr: errors.New("boom")}, nil)
	if r.Error == nil {
		t.Error("expected error propagation")
	}
}
