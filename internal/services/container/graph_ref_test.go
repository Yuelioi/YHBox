package container

import (
	"encoding/json"
	"testing"
)

func TestGraphRefSerialization(t *testing.T) {
	cases := []struct {
		name string
		ref  GraphRef
		want string
	}{
		{"main", MainGraphRef(), `{"kind":"main","id":""}`},
		{"subgraph", SubgraphGraphRef("sg-abc"), `{"kind":"subgraph","id":"sg-abc"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.ref)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(b) != tc.want {
				t.Errorf("got %s want %s", string(b), tc.want)
			}
			var back GraphRef
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if back != tc.ref {
				t.Errorf("round-trip: got %+v want %+v", back, tc.ref)
			}
		})
	}
}

func TestGraphRefIsMain(t *testing.T) {
	if !MainGraphRef().IsMain() {
		t.Errorf("MainGraphRef should be main")
	}
	if SubgraphGraphRef("x").IsMain() {
		t.Errorf("subgraph should not be main")
	}
}
