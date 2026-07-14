package container

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/yottaapp/yotta/internal/services/container/dependency"
)

// BuildYottaLock generates yotta-lock.json content from the portable package files.
func BuildYottaLock(manifest PackageManifest, graph Graph, closure dependency.ClosureResult, generatedAt string) (YottaLock, error) {
	deps := LockDependencies{
		Templates:   sortedStrings(closure.Templates),
		Clips:       sortedStrings(closure.Clips),
		Subgraphs:   sortedStrings(closure.Subgraphs),
		AISlots:     deriveAISlots(graph.Nodes),
		TargetSlots: deriveTargetSlots(graph.Nodes),
	}
	closureHash, err := hashJSON(deps)
	if err != nil {
		return YottaLock{}, err
	}
	manifestHash, err := hashJSON(manifest)
	if err != nil {
		return YottaLock{}, err
	}
	graphHash, err := hashJSON(graph)
	if err != nil {
		return YottaLock{}, err
	}
	return YottaLock{
		SchemaVersion: LockSchemaVersion,
		PackageID:     manifest.Yotta.PackageID,
		PackageName:   manifest.Name,
		Version:       manifest.Version,
		ManifestHash:  manifestHash,
		GraphHash:     graphHash,
		ClosureHash:   closureHash,
		GeneratedAt:   generatedAt,
		Dependencies:  deps,
	}, nil
}

func deriveTargetSlots(nodes []GraphNode) []string {
	seen := map[string]bool{}
	for _, n := range nodes {
		if n.Kind != "Win32WindowTarget" && n.Kind != "AndroidTarget" {
			continue
		}
		slot := configString(n.Config, portableTargetBindingKey)
		if slot != "" {
			seen[slot] = true
		}
	}
	return sortedKeys(seen)
}

func deriveAISlots(nodes []GraphNode) []string {
	seen := map[string]bool{}
	for _, n := range nodes {
		if n.Kind != "AI" {
			continue
		}
		slot := configString(n.Config, "Connection")
		if slot != "" {
			seen[slot] = true
		}
	}
	return sortedKeys(seen)
}

func configString(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	if v, ok := config[key].(string); ok {
		return v
	}
	literal, ok := config["literal"].(map[string]any)
	if !ok {
		return ""
	}
	v, _ := literal[key].(string)
	return v
}

func hashJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return "sha256-" + hex.EncodeToString(sum[:]), nil
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func sortedKeys(in map[string]bool) []string {
	out := make([]string, 0, len(in))
	for key := range in {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
