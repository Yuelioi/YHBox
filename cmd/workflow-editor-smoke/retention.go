package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

func seedRecoveryFixture(ctx context.Context, root, retentionSource, retentionBlobs string) error {
	const originalName = "damaged-workflow.json"
	raw := []byte(`{"format":"yotta.workflow","version":"1",`)
	identityInput := make([]byte, 0, len(originalName)+1+len(raw))
	identityInput = append(identityInput, originalName...)
	identityInput = append(identityInput, 0)
	identityInput = append(identityInput, raw...)
	recoveryID, err := artifact.Sum("yotta/workflow-source-recovery/v1", identityInput)
	if err != nil {
		return fmt.Errorf("identify workflow recovery fixture: %w", err)
	}
	profile, err := storage.Open(ctx, storage.OpenOptions{Root: root})
	if err != nil {
		return fmt.Errorf("open smoke storage profile: %w", err)
	}
	defer profile.Close()
	foundation, err := catalog.Open(ctx, profile.Roots)
	if err != nil {
		return fmt.Errorf("open smoke Catalog: %w", err)
	}
	defer foundation.Close()
	if err := foundation.Workflows().PutQuarantine(ctx, catalog.WorkflowQuarantineRecord{
		ID: recoveryID, OriginalName: originalName,
		Reason: "synthetic invalid JSON", Artifact: raw, CreatedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("seed workflow recovery fixture: %w", err)
	}
	if retentionSource == "" {
		if err := seedAuthoringFixture(ctx, foundation); err != nil {
			return err
		}
	} else {
		if err := seedRetentionFixture(
			ctx, profile.Roots, foundation, retentionSource, retentionBlobs,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedAuthoringFixture(ctx context.Context, foundation *catalog.Foundation) error {
	builtins, err := nodes.Build()
	if err != nil {
		return fmt.Errorf("build smoke node catalog: %w", err)
	}
	runStarted, ok := builtins.Definition(nodes.RunStartedNodeID)
	if !ok {
		return errors.New("smoke node catalog omitted RunStarted")
	}
	store, err := workflowstore.OpenSourceStore(foundation.Workflows(), workflowstore.SourceStoreOptions{MaxSources: 16, Now: time.Now})
	if err != nil {
		return fmt.Errorf("open smoke Workflow Source store: %w", err)
	}
	source := schema.WorkflowSource{
		Format: schema.Format, Version: schema.Version,
		Workflow: schema.Workflow{ID: uuid.NewString(), Name: "Visual acceptance"},
		Revision: 0, EntryGraph: "main",
		Graphs: []schema.Graph{{
			ID: "main", Kind: schema.GraphKindMain,
			Nodes: []schema.Node{{
				ID: "run-started", NodeRef: runStarted.Contract.NodeRef(), Position: schema.Position{X: 120, Y: 160},
				Config: map[string]any{}, Bindings: map[string]schema.InputBinding{},
			}},
			Edges: []schema.Edge{}, Inputs: []schema.GraphPort{}, Outputs: []schema.GraphPort{},
		}},
		Resources: []schema.WorkflowResource{}, TargetProfileDefinitions: []schema.TargetProfileDefinition{},
		CredentialRequirements: []schema.CredentialRequirement{}, Dependencies: []schema.NodePackageDependency{}, Variables: []schema.Variable{},
	}
	raw, err := artifact.Marshal(source)
	if err != nil {
		return err
	}
	if _, err := store.Save(ctx, raw, -1); err != nil {
		return fmt.Errorf("seed smoke Workflow Source: %w", err)
	}
	return nil
}

type retentionReport struct {
	Status         string `json:"status"`
	WorkflowID     string `json:"workflowId"`
	SourceHash     string `json:"sourceHash"`
	Graphs         int    `json:"graphs"`
	Nodes          int    `json:"nodes"`
	Resources      int    `json:"resources"`
	BlobReferences int    `json:"blobReferences"`
	BlobBytes      int64  `json:"blobBytes"`
	TargetDefaults int    `json:"targetDefaults"`
}

type retentionFixture struct {
	source    schema.WorkflowSource
	canonical []byte
	refs      []blob.BlobRef
	report    retentionReport
	layout    blob.LayoutInspection
}

func checkWorkflowRetention(
	ctx context.Context,
	sourcePath string,
	blobRoot string,
) (retentionReport, error) {
	fixture, err := loadRetentionFixture(ctx, sourcePath, blobRoot)
	if err != nil {
		return retentionReport{}, err
	}
	build, err := compiler.BuildDigest()
	if err != nil {
		return retentionReport{}, fmt.Errorf("identify current compiler: %w", err)
	}
	builtinNodes, err := nodes.Build()
	if err != nil {
		return retentionReport{}, fmt.Errorf("build current node catalog: %w", err)
	}
	compileSource, err := migrateRetentionContracts(fixture.source, builtinNodes)
	if err != nil {
		return retentionReport{}, err
	}
	compiled, err := compiler.New(build, builtinNodes.ConfigValidators).CompileDraft(
		ctx,
		compiler.CompileRequest{
			SourceJSON: compileSource,
			Catalog:    builtinNodes.Catalog,
			BlobVerifier: compiler.BlobVerifierFunc(func(
				verifyCtx context.Context,
				ref blob.BlobRef,
			) error {
				return verifyRetentionBlob(verifyCtx, blobRoot, fixture.layout.Version, ref)
			}),
		},
	)
	if err != nil {
		return retentionReport{}, fmt.Errorf("compile golden Workflow: %w", err)
	}
	if schema.HasErrors(compiled.Diagnostics) {
		encoded, _ := json.Marshal(compiled.Diagnostics)
		return retentionReport{}, fmt.Errorf("golden Workflow has compile errors: %s", encoded)
	}
	if _, ok := compiled.Program(); !ok {
		return retentionReport{}, errors.New("golden Workflow compile produced no executable program")
	}
	return fixture.report, nil
}

func migrateRetentionContracts(
	source schema.WorkflowSource,
	builtinNodes nodes.Builtins,
) ([]byte, error) {
	bindings := builtinNodes.Catalog.Bindings()
	contracts := make([]nodecontract.Contract, 0, len(bindings))
	for _, binding := range bindings {
		contracts = append(contracts, binding.Contract)
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtinNodes.Catalog, Types: builtinNodes.Catalog.Types(),
		Capabilities: builtinNodes.Catalog.Capabilities(), Contracts: contracts,
		GeneratorVersion: nodes.GeneratorVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("build current authoring projection: %w", err)
	}
	engine, err := authoring.New(builtinNodes.Catalog, projection, nil)
	if err != nil {
		return nil, fmt.Errorf("open current authoring engine: %w", err)
	}
	commands := make([]authoring.Command, 0)
	for _, graph := range source.Graphs {
		for _, node := range graph.Nodes {
			current, ok := projection.Node(node.NodeRef.NodeTypeID)
			if !ok || current.NodeRef == node.NodeRef ||
				current.NodeRef.Version != node.NodeRef.Version {
				continue
			}
			command := authoring.Command{
				Kind: authoring.CommandUpgradeNodeContract,
				UpgradeNodeContract: &authoring.NodeCommand{
					GraphID: graph.ID,
					NodeID:  node.ID,
				},
			}
			if _, err := engine.Apply(source, []authoring.Command{command}); err == nil {
				commands = append(commands, command)
			}
		}
	}
	if len(commands) == 0 {
		raw, err := artifact.Marshal(source)
		if err != nil {
			return nil, fmt.Errorf("encode golden Workflow Source: %w", err)
		}
		return raw, nil
	}
	result, err := engine.Apply(source, commands)
	if err != nil {
		return nil, fmt.Errorf("migrate golden Workflow node contracts: %w", err)
	}
	return result.Artifact, nil
}

func loadRetentionFixture(
	ctx context.Context,
	sourcePath string,
	blobRoot string,
) (retentionFixture, error) {
	if strings.TrimSpace(sourcePath) == "" || strings.TrimSpace(blobRoot) == "" {
		return retentionFixture{}, errors.New("golden Workflow requires Source and Blob Store paths")
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return retentionFixture{}, fmt.Errorf("read golden Workflow Source: %w", err)
	}
	source, canonical, sourceHash, diagnostics, err := schema.CanonicalSource(raw)
	if err != nil {
		return retentionFixture{}, fmt.Errorf("canonicalize golden Workflow Source: %w", err)
	}
	if schema.HasErrors(diagnostics) {
		encoded, _ := json.Marshal(diagnostics)
		return retentionFixture{}, fmt.Errorf("golden Workflow Source is invalid: %s", encoded)
	}
	refs, err := schema.BlobReferences(source)
	if err != nil {
		return retentionFixture{}, fmt.Errorf("inventory golden Workflow blobs: %w", err)
	}
	layout, err := blob.InspectLayout(ctx, blobRoot)
	if err != nil {
		return retentionFixture{}, fmt.Errorf("inspect golden Blob Store: %w", err)
	}
	if layout.Version != "1" && layout.Version != blob.LayoutVersion {
		return retentionFixture{}, fmt.Errorf("golden Blob Store layout %q is unsupported", layout.Version)
	}
	var nodeCount int
	for _, graph := range source.Graphs {
		nodeCount += len(graph.Nodes)
	}
	var blobBytes int64
	for _, ref := range refs {
		if err := verifyRetentionBlob(ctx, blobRoot, layout.Version, ref); err != nil {
			return retentionFixture{}, fmt.Errorf("verify golden Workflow blob %s: %w", ref.Digest, err)
		}
		blobBytes += ref.Size
	}
	report := retentionReport{
		Status:         "retained",
		WorkflowID:     source.Workflow.ID,
		SourceHash:     sourceHash.String(),
		Graphs:         len(source.Graphs),
		Nodes:          nodeCount,
		Resources:      len(source.Resources),
		BlobReferences: len(refs),
		BlobBytes:      blobBytes,
		TargetDefaults: len(source.TargetDefaults),
	}
	if source.Workflow.ID != "fishing-v2" ||
		report.Graphs < 7 ||
		report.Nodes < 60 ||
		report.Resources < 18 ||
		report.BlobReferences < 36 {
		encoded, _ := json.Marshal(report)
		return retentionFixture{}, fmt.Errorf("golden Workflow retention baseline drifted: %s", encoded)
	}
	return retentionFixture{
		source: source, canonical: canonical, refs: refs, report: report, layout: layout,
	}, nil
}

func verifyRetentionBlob(
	ctx context.Context,
	root string,
	layout string,
	ref blob.BlobRef,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	name := strings.TrimPrefix(ref.Digest.String(), "sha256:")
	path := filepath.Join(root, name)
	if layout == blob.LayoutVersion {
		path = filepath.Join(root, name[:2], name[2:4], name)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != ref.Size {
		return errors.New("blob physical identity does not match the Source reference")
	}
	hash := sha256.New()
	written, err := io.Copy(hash, &contextReader{ctx: ctx, reader: file})
	if err != nil {
		return err
	}
	if written != ref.Size || hex.EncodeToString(hash.Sum(nil)) != name {
		return errors.New("blob digest does not match the Source reference")
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

func seedRetentionFixture(
	ctx context.Context,
	roots storage.Roots,
	foundation *catalog.Foundation,
	sourcePath string,
	sourceBlobRoot string,
) error {
	fixture, err := loadRetentionFixture(ctx, sourcePath, sourceBlobRoot)
	if err != nil {
		return err
	}
	targetStore, err := blob.Open(
		roots.Objects,
		blob.Limits{MaxBlobBytes: 256 << 20, MaxTotalBytes: 4 << 30},
		foundation.Objects(),
	)
	if err != nil {
		return fmt.Errorf("open isolated golden Blob Store: %w", err)
	}
	for _, ref := range fixture.refs {
		name := strings.TrimPrefix(ref.Digest.String(), "sha256:")
		path := filepath.Join(sourceBlobRoot, name)
		if fixture.layout.Version == blob.LayoutVersion {
			path = filepath.Join(sourceBlobRoot, name[:2], name[2:4], name)
		}
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open golden Workflow blob %s: %w", ref.Digest, err)
		}
		copied, putErr := targetStore.Put(ctx, ref.MediaType, file)
		closeErr := file.Close()
		if putErr != nil || closeErr != nil {
			return errors.Join(putErr, closeErr)
		}
		if copied != ref {
			return fmt.Errorf("golden Workflow blob %s changed while seeding", ref.Digest)
		}
	}

	// An isolated smoke profile begins at revision zero while retaining the
	// complete user-authored graph and resource content.
	fixture.source.Revision = 0
	raw, err := json.Marshal(fixture.source)
	if err != nil {
		return fmt.Errorf("encode isolated golden Workflow: %w", err)
	}
	sourceStore, err := workflowstore.OpenSourceStore(
		foundation.Workflows(),
		workflowstore.SourceStoreOptions{MaxSources: 100},
	)
	if err != nil {
		return fmt.Errorf("open isolated Workflow Source Store: %w", err)
	}
	if _, err := sourceStore.Save(ctx, raw, -1); err != nil {
		return fmt.Errorf("seed isolated golden Workflow: %w", err)
	}
	return nil
}
