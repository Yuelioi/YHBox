package snippet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestServicePersistsExactNodeTemplateWithoutInstanceState(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	value, err := service.Save(&Snippet{
		Name: "Configured click", Category: "Automation", Tags: []string{"game", "GAME"},
		Payload: NodeTemplate{
			NodeRef: nodecontract.NodeRef{NodeTypeID: "https://schemas.yotta.dev/nodes/automation/click-pointer", Version: "1.0.0", SemanticDigest: artifact.Digest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
			Label:   "Confirm", Config: map[string]any{"target": "window-target"},
			Bindings: map[string]schema.InputBinding{"button": {Kind: schema.BindingValue, Value: []byte(`"left"`)}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.ID == "" || value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() {
		t.Fatalf("saved metadata incomplete: %+v", value)
	}
	if len(value.Tags) != 1 || value.Tags[0] != "game" {
		t.Fatalf("tags = %#v", value.Tags)
	}
	used, err := service.MarkUsed(value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if used.UsageCount != 1 || used.LastUsedAt == nil {
		t.Fatalf("usage metadata = %+v", used)
	}
	reopened, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	loaded, ok := reopened.Get(value.ID)
	if !ok || loaded.Payload.Label != "Confirm" || loaded.Payload.Config["target"] != "window-target" || loaded.UsageCount != 1 || loaded.LastUsedAt == nil {
		t.Fatalf("loaded = %+v", loaded)
	}
	content, err := os.ReadFile(filepath.Join(root, value.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{`"position"`, `"nodeId"`} {
		if contains := string(content); len(contains) > 0 && stringContains(contains, forbidden) {
			t.Fatalf("persisted snippet contains %s: %s", forbidden, contains)
		}
	}
}

func TestServiceMigratesStaleNodeRefAndDurablyReopens(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := NewService(store).Save(&Snippet{
		Name: "Legacy move pointer",
		Payload: NodeTemplate{
			NodeRef: nodecontract.NodeRef{
				NodeTypeID:     "https://schemas.yotta.dev/nodes/automation/move-pointer",
				Version:        "1.0.0",
				SemanticDigest: artifact.Digest("sha256:2bf1f8059f1269e407d2aedf4f717cc6c0b860eb46b92abd1794a3aa3bf559af"),
			},
			Config:   map[string]any{"slot": "desktop-window"},
			Bindings: map[string]schema.InputBinding{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	projection := currentAuthoringProjection(t)
	migrated, err := NewServiceWithAuthoring(store, projection).Get(legacy.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, ok := projection.Node(legacy.Payload.NodeRef.NodeTypeID)
	if !ok || migrated.Payload.NodeRef != current.NodeRef {
		t.Fatalf("migrated NodeRef = %+v, current = %+v", migrated.Payload.NodeRef, current.NodeRef)
	}
	if got := string(migrated.Payload.Bindings["duration"].Value); got != "300" {
		t.Fatalf("legacy duration = %s", got)
	}
	if got := string(migrated.Payload.Bindings["motion"].Value); got != `"linear"` {
		t.Fatalf("legacy motion = %s", got)
	}
	reopened, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	persisted, ok := reopened.Get(legacy.ID)
	if !ok || persisted.Payload.NodeRef != current.NodeRef {
		t.Fatalf("persisted migrated snippet = %+v", persisted)
	}
}

func TestStoreIsolatesCorruptItems(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	result := store.List()
	if len(result.Items) != 0 || len(result.Warnings) != 1 || result.Warnings[0].File != "broken.json" {
		t.Fatalf("list result = %+v", result)
	}
}

func TestServiceRejectsSensitiveRuntimeFields(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		config   map[string]any
		bindings map[string]schema.InputBinding
	}{
		{name: "config", config: map[string]any{"runtimeHandle": "native"}, bindings: map[string]schema.InputBinding{}},
		{name: "binding", config: map[string]any{}, bindings: map[string]schema.InputBinding{"target": {Kind: schema.BindingValue, Value: []byte(`{"credential":"unsafe"}`)}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewService(store).Save(&Snippet{
				Name: "unsafe",
				Payload: NodeTemplate{
					NodeRef: nodecontract.NodeRef{NodeTypeID: "https://example.com/node", Version: "1.0.0", SemanticDigest: artifact.Digest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
					Config:  test.config, Bindings: test.bindings,
				},
			})
			if err == nil {
				t.Fatal("expected sensitive runtime field rejection")
			}
		})
	}
}

func TestServiceNormalizesAndRejectsUnsafeSnippetShortcuts(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	base := func(name, shortcut string) *Snippet {
		return &Snippet{
			Name: name, Shortcut: shortcut,
			Payload: NodeTemplate{
				NodeRef: nodecontract.NodeRef{NodeTypeID: "https://example.com/node", Version: "1.0.0", SemanticDigest: artifact.Digest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
				Config:  map[string]any{}, Bindings: map[string]schema.InputBinding{},
			},
		}
	}
	first, err := service.Save(base("First", " shift + ctrl + k "))
	if err != nil {
		t.Fatal(err)
	}
	if first.Shortcut != "Ctrl+Shift+K" || service.List().Items[0].Shortcut != "Ctrl+Shift+K" {
		t.Fatalf("shortcut was not normalized: %+v", first)
	}
	if _, err := service.Save(base("Duplicate", "ctrl+shift+k")); err == nil {
		t.Fatal("expected duplicate shortcut rejection")
	}
	for _, shortcut := range []string{"Ctrl+C", "K", "Ctrl+Shift"} {
		if _, err := service.Save(base("Unsafe", shortcut)); err == nil {
			t.Fatalf("expected %q to be rejected", shortcut)
		}
	}
}

func stringContains(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}

func currentAuthoringProjection(t *testing.T) nodeauthoring.Snapshot {
	t.Helper()
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: nodes.GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return projection
}
