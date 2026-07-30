package appbootstrap

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appcontrol"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/resource"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/stream"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

func TestExecutionEnvironmentFactorySealsOneConsistentSnapshot(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	scriptRuntime, err := scriptengine.NewRuntime(scriptengine.RuntimeOptions{
		Executable:         filepath.Join(t.TempDir(), scriptengine.WorkerExecutableName),
		ProcessMemoryBytes: scriptengine.DefaultMemoryBytes,
		JobMemoryBytes:     scriptengine.DefaultMemoryBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	blobDigest, err := blob.ProviderArtifactDigest()
	if err != nil {
		t.Fatal(err)
	}
	streamDigest, err := stream.ProviderArtifactDigest()
	if err != nil {
		t.Fatal(err)
	}
	workspaceDigest, err := workspacefs.ProviderArtifactDigest()
	if err != nil {
		t.Fatal(err)
	}
	aiInstallations, err := ai.Install(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	httpInstallations, err := httpegress.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	applications, err := appcontrol.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	automation, err := automationinstalled.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	factory, err := newExecutionEnvironmentFactory(executionEnvironmentFactory{
		builtins: builtins, blobDigest: blobDigest, streamDigest: streamDigest, workspaceFileDigest: workspaceDigest,
		scriptRuntime: scriptRuntime,
		ai:            aiInstallations,
		http:          httpInstallations,
		baseProviders: map[string]run.InstalledProvider{
			blob.ProviderID: {
				ArtifactDigest: blobDigest, ABI: blob.ProviderABI, Provider: inertProvider{},
			},
			stream.ProviderID: {
				ArtifactDigest: streamDigest, ABI: stream.ProviderABI, Provider: inertProvider{},
			},
			workspacefs.ProviderID: {
				ArtifactDigest: workspaceDigest, ABI: workspacefs.ProviderABI, Provider: inertProvider{},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	environment, err := factory.seal(applications, automation)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := environment.generation.Retire(); err != nil {
			t.Errorf("retire environment: %v", err)
		}
	})

	if !environment.profile.Valid() || environment.policy == nil ||
		len(environment.providers) != 3 || !environment.generation.Valid() {
		t.Fatalf("sealed environment = %#v", environment)
	}
	_, release, err := environment.acquire()
	if err != nil {
		t.Fatal(err)
	}
	release()
}

type inertProvider struct{}

func (inertProvider) Open(context.Context, resource.ProviderOpenRequest) (any, error) {
	return struct{}{}, nil
}
func (inertProvider) Invoke(context.Context, any, string, []byte) ([]byte, error) {
	return nil, nil
}
func (inertProvider) Close(context.Context, any) error { return nil }

var _ resource.Provider = inertProvider{}
