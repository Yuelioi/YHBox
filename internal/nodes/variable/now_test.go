package variable

import (
	"context"
	"testing"
	"time"

	"yotta/internal/node"
)

func TestNow_ReturnsLiveUnixMillis(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Now{})
	rn, _ := node.Get("Now")

	services := node.StubServices()

	before := time.Now().UnixMilli()
	v, err := node.EvaluatePureData(context.Background(), rn, nil, nil, services)
	after := time.Now().UnixMilli()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := v.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", v)
	}
	ms := int64(got)
	if ms < before || ms > after {
		t.Errorf("expected unix ms in [%d, %d], got %d", before, after, ms)
	}
}
