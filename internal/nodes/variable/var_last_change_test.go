package variable

import (
	"context"
	"testing"

	"yotta/internal/node"
)

// stampVarStore — full VarStore stub returning a fixed LastChange stamp.
type stampVarStore struct{ stamp int64 }

func (stampVarStore) Get(string) (any, bool)               { return nil, false }
func (stampVarStore) Set(string, any)                      {}
func (stampVarStore) Inc(string, float64) float64          { return 0 }
func (stampVarStore) GetScoped(string, string) (any, bool) { return nil, false }
func (stampVarStore) SetScoped(string, string, any)        {}
func (stampVarStore) IncScoped(string, string, float64) float64 { return 0 }
func (s stampVarStore) LastChange(string) int64            { return s.stamp }

func TestVarLastChange_ReadsLiveStamp(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&VarLastChange{})
	rn, _ := node.Get("VarLastChange")

	services := node.StubServices()
	services.Vars = stampVarStore{stamp: 1717000000000}

	v, err := node.EvaluatePureData(context.Background(), rn,
		nil,
		map[string]any{"VarName": "hp"},
		services,
	)
	if err != nil {
		t.Fatal(err)
	}
	if v != float64(1717000000000) {
		t.Errorf("expected 1717000000000, got %v", v)
	}
}

func TestVarLastChange_UnsetReturnsZero(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&VarLastChange{})
	rn, _ := node.Get("VarLastChange")

	services := node.StubServices()
	services.Vars = stampVarStore{stamp: 0}

	v, err := node.EvaluatePureData(context.Background(), rn,
		nil,
		map[string]any{"VarName": "hp"},
		services,
	)
	if err != nil {
		t.Fatal(err)
	}
	if v != float64(0) {
		t.Errorf("expected 0, got %v", v)
	}
}
