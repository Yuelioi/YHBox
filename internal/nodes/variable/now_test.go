package variable

import (
	"context"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

func TestNow_ReturnsLiveUnixMillis(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&Now{})
	rn, _ := registry.Get("Now")

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
