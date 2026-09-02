package runprepare

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type countingStore struct {
	*blob.Store
	puts int
}

func (store *countingStore) Put(ctx context.Context, mediaType string, source io.Reader) (blob.BlobRef, error) {
	store.puts++
	return store.Store.Put(ctx, mediaType, source)
}

func (store *countingStore) PutRetained(ctx context.Context, mediaType string, source io.Reader) (blob.BlobRef, blob.Retention, error) {
	store.puts++
	return store.Store.PutRetained(ctx, mediaType, source)
}

func TestPlannerSelectsExactVariantAndScalesFallbackOnce(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	ref1080 := putPattern(t, store, 30, 15)
	ref2K := putPattern(t, store, 40, 20)
	planner, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	source := sourceArtifact(t, []schema.ImageResourceVariant{
		{ID: "1080p", Resolution: [2]int{1920, 1080}, BBox: [4]int{0, 0, 30, 15}, Blob: ref1080},
		{ID: "2k", Resolution: [2]int{2560, 1440}, BBox: [4]int{0, 0, 40, 20}, Blob: ref2K},
	})

	exact, err := planner.Prepare(ctx, source, map[string][2]int{"game": {2560, 1440}})
	if err != nil || len(exact.Overrides) != 1 || exact.Overrides[0].Blob != ref2K || store.puts != 2 {
		t.Fatalf("exact overrides=%+v puts=%d err=%v", exact, store.puts, err)
	}

	fallbackSource := sourceArtifact(t, []schema.ImageResourceVariant{
		{ID: "1080p", Resolution: [2]int{1920, 1080}, BBox: [4]int{0, 0, 30, 15}, Blob: ref1080},
	})
	first, err := planner.Prepare(ctx, fallbackSource, map[string][2]int{"game": {2560, 1440}})
	if err != nil || len(first.Overrides) != 1 || first.Overrides[0].Blob == ref1080 || store.puts != 3 {
		t.Fatalf("fallback overrides=%+v puts=%d err=%v", first, store.puts, err)
	}
	second, err := planner.Prepare(ctx, fallbackSource, map[string][2]int{"game": {2560, 1440}})
	if err != nil || second.Overrides[0].Blob != first.Overrides[0].Blob || store.puts != 4 {
		t.Fatalf("cached overrides=%+v puts=%d err=%v", second, store.puts, err)
	}
	raw, err := store.ReadRange(ctx, first.Overrides[0].Blob, 0, first.Overrides[0].Blob.Size)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(raw))
	if err != nil || decoded.Bounds().Dx() != 40 || decoded.Bounds().Dy() != 20 {
		t.Fatalf("scaled bounds=%v err=%v", decoded.Bounds(), err)
	}
	exact.Release()
	first.Release()
	second.Release()
}

func TestPlannerRejectsAspectRatioDistortion(t *testing.T) {
	store := openTestStore(t)
	ref := putPattern(t, store, 30, 15)
	planner, _ := New(store)
	source := sourceArtifact(t, []schema.ImageResourceVariant{{
		ID: "1080p", Resolution: [2]int{1920, 1080}, BBox: [4]int{0, 0, 30, 15}, Blob: ref,
	}})
	if _, err := planner.Prepare(context.Background(), source, map[string][2]int{"game": {3440, 1440}}); err == nil {
		t.Fatal("aspect-ratio mismatch was accepted")
	}
}

func openTestStore(t *testing.T) *countingStore {
	t.Helper()
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return &countingStore{Store: store}
}

func putPattern(t *testing.T, store *countingStore, width, height int) blob.BlobRef {
	t.Helper()
	value := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			value.SetRGBA(x, y, color.RGBA{R: uint8(x * 7), G: uint8(y * 11), B: uint8((x + y) * 5), A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, value); err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), "image/png", bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func sourceArtifact(t *testing.T, variants []schema.ImageResourceVariant) []byte {
	t.Helper()
	source := schema.WorkflowSource{
		Format: schema.Format, Version: schema.Version,
		Workflow: schema.Workflow{ID: "adaptive", Name: "Adaptive"},
		Revision: 0, EntryGraph: "main",
		Graphs: []schema.Graph{{ID: "main", Kind: schema.GraphKindMain, Nodes: []schema.Node{{
			ID: "match", NodeRef: nodecontract.NodeRef{NodeTypeID: "https://schemas.yotta.dev/nodes/test/image", Version: "1.0.0", SemanticDigest: artifact.Digest("sha256:1111111111111111111111111111111111111111111111111111111111111111")},
			Position: schema.Position{}, Config: map[string]any{"slot": "game"}, Bindings: map[string]schema.InputBinding{
				"template": {Kind: schema.BindingResource, Resource: &schema.ResourceBinding{ResourceID: "template", VariantID: variants[0].ID}},
			},
		}}, Edges: []schema.Edge{}, Inputs: []schema.GraphPort{}, Outputs: []schema.GraphPort{}}},
		Resources:                []schema.WorkflowResource{{ID: "template", Kind: schema.ResourceImage, Name: "Template", Image: &schema.ImageResource{Variants: variants}}},
		TargetProfileDefinitions: []schema.TargetProfileDefinition{}, CredentialRequirements: []schema.CredentialRequirement{}, Dependencies: []schema.NodePackageDependency{}, Variables: []schema.Variable{},
	}
	raw, err := artifact.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
