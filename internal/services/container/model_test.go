package container

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestContainer_JSONRoundTrip(t *testing.T) {
	src := Container{
		SchemaVersion: 1, ID: "uuid-1", Name: "战斗主循环",
		Description: "test", Tags: []string{"副本", "自动"},
		Hotkey: "Ctrl+Shift+1",
		Vars:   []VarDecl{{Name: "enemyHp", Type: "number", Default: float64(100)}},
		Graph: Graph{
			Nodes: []GraphNode{{ID: "n1", Kind: "Start", X: 100, Y: 100, Config: map[string]any{}}},
			Edges: []GraphEdge{{From: "n1.Done", To: "n2.In"}},
		},
		CreatedAt: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
	}
	b, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Container
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != src.Name || got.Hotkey != src.Hotkey {
		t.Errorf("string fields mismatch")
	}
	if len(got.Tags) != 2 || got.Tags[0] != "副本" {
		t.Errorf("tags mismatch: %v", got.Tags)
	}
	if len(got.Vars) != 1 || got.Vars[0].Name != "enemyHp" {
		t.Errorf("vars mismatch: %v", got.Vars)
	}
	if len(got.Graph.Nodes) != 1 || got.Graph.Nodes[0].Kind != "Start" {
		t.Errorf("graph nodes mismatch")
	}
	if len(got.Graph.Edges) != 1 || got.Graph.Edges[0].From != "n1.Done" {
		t.Errorf("graph edges mismatch")
	}
}

// 验证当前 graph schema：Graph 必须有 ID + schemaVersion；Container 不再有 Category；GraphNode 有 CreatedAt。
func TestContainerGraphSchemaFields(t *testing.T) {
	c := Container{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "x",
		Name:          "test",
		Graph: Graph{
			ID:            "g-main",
			SchemaVersion: GraphSchemaVersion,
			Nodes: []GraphNode{
				{ID: "n1", Kind: "Start", CreatedAt: time.Now().UTC()},
			},
			Edges: nil,
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes.Contains(b, []byte(`"category"`)) {
		t.Errorf("expected no 'category' field in JSON, got: %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"id":"g-main"`)) {
		t.Errorf("expected Graph.ID in JSON, got: %s", string(b))
	}
	if bytes.Contains(b, []byte(`"version"`)) {
		t.Errorf("expected no legacy Graph.SchemaVersion in JSON, got: %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"schemaVersion":1`)) {
		t.Errorf("expected Graph.schemaVersion=1 in JSON, got: %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"createdAt"`)) {
		t.Errorf("expected node.createdAt in JSON, got: %s", string(b))
	}

	var c2 Container
	if err := json.Unmarshal(b, &c2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c2.Graph.ID != "g-main" {
		t.Errorf("Graph.ID lost: %q", c2.Graph.ID)
	}
	if c2.Graph.SchemaVersion != GraphSchemaVersion {
		t.Errorf("Graph.SchemaVersion lost: %d", c2.Graph.SchemaVersion)
	}
	if c2.Graph.Nodes[0].CreatedAt.IsZero() {
		t.Errorf("node.CreatedAt lost")
	}
}

func TestPackageManifestJSONShape(t *testing.T) {
	m := PackageManifest{
		SchemaVersion: PackageSchemaVersion,
		Kind:          PackageKindContainer,
		Name:          "@yotta/daily-fishing",
		DisplayName:   "每日钓鱼",
		Version:       "0.1.0",
		Description:   "自动完成每日钓鱼循环",
		Summary:       "每日钓鱼自动化",
		Keywords:      []string{"fishing", "daily", "mumu"},
		Category:      "daily",
		License:       "MIT",
		Author:        PackagePerson{Name: "yl"},
		Publisher:     PackagePublisher{ID: "yotta", Name: "Yotta"},
		Repository:    &PackageLink{Type: "git"},
		Yotta: PackageYotta{
			PackageID:  "pkg_01jz_daily_fishing",
			EntryGraph: "graph.json",
			Publication: Publication{
				State:      PublicationDraft,
				Visibility: VisibilityPrivate,
			},
			Sources: []SourceRef{},
			Vars:    []VarDecl{{Name: "state", Type: "string", Default: "IDLE"}},
			Targets: map[string]TargetSlot{
				"game": {Kind: "win32-window", DisplayName: "游戏窗口"},
			},
			AI: map[string]AISlot{
				"main": {
					DisplayName:  "默认 AI",
					ProviderHint: "openai-compatible",
					Capabilities: []string{"vision", "json"},
				},
			},
		},
	}

	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes.Contains(b, []byte(`"createdAt"`)) || bytes.Contains(b, []byte(`"updatedAt"`)) {
		t.Fatalf("package manifest must not contain local time fields: %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"packageId":"pkg_01jz_daily_fishing"`)) {
		t.Fatalf("packageId missing from yotta manifest: %s", string(b))
	}
	if bytes.Contains(b, []byte(`"uid"`)) {
		t.Fatalf("manifest must use packageId instead of uid: %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"targets":{"game"`)) || !bytes.Contains(b, []byte(`"ai":{"main"`)) {
		t.Fatalf("logical binding slots missing: %s", string(b))
	}
}

func TestPackageManifestJSONOmitsEmptyLinks(t *testing.T) {
	manifest := containerToPackageManifest(Container{ID: "local-id", Name: "Local"})
	b, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, field := range []string{"repository", "bugs", "docs", "changelog"} {
		if bytes.Contains(b, []byte(`"`+field+`"`)) {
			t.Errorf("empty %s must be omitted from package.json: %s", field, string(b))
		}
	}
}

func TestInstallationJSONShape(t *testing.T) {
	hotkey := "Ctrl+Shift+1"
	inst := Installation{
		SchemaVersion:    InstallationSchemaVersion,
		InstanceID:       "local_uuid",
		PackageID:        "pkg_01jz_daily_fishing",
		PackageName:      "@yotta/daily-fishing",
		InstalledVersion: "0.1.0",
		Display: InstallationDisplay{
			Favorite: true,
		},
		RuntimeOverrides: RuntimeOverrides{
			Hotkey: &hotkey,
		},
		TargetBindings: map[string]TargetBinding{
			"game": {
				Kind: "win32-window",
				Match: map[string]any{
					"title":      "Blue Archive",
					"titleMatch": "contains",
				},
			},
		},
		AIBindings: map[string]AIBinding{
			"main": {ConnectionID: "local-openai-connection"},
		},
		Updates: InstallationUpdates{
			AutoCheck: true,
		},
	}

	b, err := json.Marshal(inst)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !bytes.Contains(b, []byte(`"packageId":"pkg_01jz_daily_fishing"`)) {
		t.Fatalf("installation must keep stable packageId link: %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"packageName":"@yotta/daily-fishing"`)) {
		t.Fatalf("installation should keep packageName display cache: %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"inputBackend":null`)) || !bytes.Contains(b, []byte(`"scaleTolerance":null`)) {
		t.Fatalf("unset runtime overrides must marshal as null: %s", string(b))
	}
}
