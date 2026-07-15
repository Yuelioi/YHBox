package appbootstrap_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/appbootstrap"
	app31 "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodes31"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/services/workflow31"
)

func TestBuildComposesWorkflowServiceThroughProductionProgramChain(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	policy, err := appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	events := make(chan app31.RunEvent, 16)
	runtime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: t.TempDir(), Limits: testLimits(), Policy: policy, GrantTTL: 5 * time.Minute,
		OwnerCloseTimeout: time.Second, Now: func() time.Time { return now },
		OnRunEvent: func(event app31.RunEvent) { events <- event },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Close(ctx); err != nil {
			t.Errorf("Close = %v", err)
		}
	})
	service, err := workflow31.NewService(runtime.Application)
	if err != nil {
		t.Fatal(err)
	}
	source := concatSource(runtime.Builtins)
	compiled, err := service.CompileDraft(string(source))
	if err != nil || !compiled.ProgramHash.Valid() || len(compiled.Diagnostics) != 0 {
		t.Fatalf("CompileDraft = %#v, %v", compiled, err)
	}
	saved, err := service.SaveSource(string(source), -1)
	if err != nil || saved.WorkflowID != "wf-bootstrap" || saved.SourceJSON == "" || saved.SourceHash != compiled.SourceHash {
		t.Fatalf("SaveSource = %#v, %v", saved, err)
	}
	if listed := service.ListSources(); len(listed) != 1 || listed[0].SourceJSON != "" || listed[0].SourceHash != saved.SourceHash {
		t.Fatalf("ListSources = %#v", listed)
	}
	started, err := service.StartRun(saved.WorkflowID)
	if err != nil || started.Run == nil || started.Run.Status != run31.StatusQueued || started.ProgramHash != compiled.ProgramHash {
		t.Fatalf("StartRun = %#v, %v", started, err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-events:
			if event.RunID == started.Run.RunID && event.Status == run31.StatusSucceeded {
				timeline, err := service.GetRunTimeline(event.RunID)
				if err != nil || timeline.Status != run31.StatusSucceeded || len(timeline.Timeline) != 2 || timeline.Failure != nil {
					t.Fatalf("GetRunTimeline = %#v, %v", timeline, err)
				}
				if catalog := service.GetCatalog(); !strings.Contains(catalog, `"version":"3.1"`) {
					t.Fatalf("GetCatalog = %s", catalog)
				}
				return
			}
		case <-deadline:
			t.Fatal("production Run did not succeed")
		}
	}
}

func TestBuiltinPolicyRejectsUninstalledProviderIdentity(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 30, 0, 0, time.UTC)
	policy, err := appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{{
		ProviderID: "third-party", ProviderArtifactDigest: testDigest(t, "forged"), ProviderABI: "https://example.test/abi/v1",
		TargetID: "remote", TargetKind: "remote", PluginInstanceID: "plugin",
	}}})
	if err != nil || decision.Outcome != admission.PolicyDenied {
		t.Fatalf("Authorize = %#v, %v", decision, err)
	}
}

func testLimits() appbootstrap.Limits {
	return appbootstrap.Limits{
		MaxSources: 8, MaxPrograms: 8, MaxRuns: 8,
		MaxBlobBytes: 1 << 20, MaxTotalBlobBytes: 8 << 20, MaxResourcePayloadBytes: 1 << 20,
		BlobChunkBytes: 64 << 10, BlobQueueCapacity: 2, StreamCapacity: 4, StreamChunkBytes: 64 << 10,
	}
}

func concatSource(builtins nodes31.Builtins) []byte {
	ref := builtins.ConcatContract.NodeRef()
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-bootstrap","name":"Bootstrap"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"concat","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},
			 "bindings":{"a":{"kind":"value","value":"a"},"b":{"kind":"value","value":"b"}}}
		],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, ref.NodeTypeID, ref.SemanticDigest))
}

func testDigest(t *testing.T, label string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("yotta/test/appbootstrap/v1", []byte(label))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
