package container

import (
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func FuzzGraphRewriter(f *testing.F) {
	f.Add([]byte(`{"nodes":[{"id":"old","kind":"Start"}],"edges":[{"from":"old.Done","to":"next.In"}]}`), "old", "new")
	f.Add([]byte(`{"nodes":[],"edges":[]}`), "", "x")
	f.Fuzz(func(t *testing.T, raw []byte, oldID, newID string) {
		if len(raw) > 64<<10 || len(oldID) > 256 || len(newID) > 256 {
			t.Skip()
		}
		var graph Graph
		if json.Unmarshal(raw, &graph) != nil {
			return
		}
		rewriter := NewGraphRewriter()
		rewriter.RenameNodeID(oldID, newID)
		rewriter.Apply(&graph)
		if _, err := json.Marshal(graph); err != nil {
			t.Fatalf("rewritten graph cannot be encoded: %v", err)
		}
	})
}

func FuzzPackageMetadataValidation(f *testing.F) {
	f.Add(
		[]byte(`{"name":"sample","version":"1.0.0","yotta":{"packageId":"pkg_sample"}}`),
		[]byte(`{"schemaVersion":1,"nodes":[],"edges":[]}`),
		[]byte(`{"schemaVersion":2,"packageId":"pkg_sample"}`),
	)
	f.Fuzz(func(t *testing.T, manifestRaw, graphRaw, lockRaw []byte) {
		if len(manifestRaw)+len(graphRaw)+len(lockRaw) > 192<<10 {
			t.Skip()
		}
		var manifest PackageManifest
		var graph Graph
		var lock YottaLock
		if json.Unmarshal(manifestRaw, &manifest) != nil || json.Unmarshal(graphRaw, &graph) != nil || json.Unmarshal(lockRaw, &lock) != nil {
			return
		}
		_ = validatePackageLock(node.DefaultRegistrySnapshot(), manifest, graph, nil, lock)
	})
}
