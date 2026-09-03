package localruntime_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/localruntime"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/securestore"
	"github.com/yottaapp/yotta/internal/services"
)

func TestOpenOwnsTheCompleteLocalRuntimeLifecycle(t *testing.T) {
	root := t.TempDir()
	config := localruntime.Config{
		StorageRoot: root,
		Executable:  filepath.Join(root, "Yotta.exe"),
		RootLog:     zerolog.New(io.Discard),
		WorkflowLog: discardWorkflowLog{},
		Now:         time.Now,
	}
	opened, err := localruntime.Open(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Settings == nil || opened.Workflow == nil || opened.Workflow.BlobStore == nil ||
		opened.Assets == nil || opened.Objects == nil || opened.Roots.Root == "" {
		t.Fatalf("local runtime omitted an owned result: %#v", opened)
	}
	if err := opened.Workflow.Application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := opened.Close(closeCtx); err != nil {
		t.Fatal(err)
	}

	reopened, err := localruntime.Open(context.Background(), config)
	if err != nil {
		t.Fatalf("local runtime did not release the profile lease: %v", err)
	}
	if err := reopened.Close(closeCtx); err != nil {
		t.Fatal(err)
	}
}

func TestOpenResolvesAnEmptyStorageRootThroughTheCanonicalRootSet(t *testing.T) {
	root := t.TempDir()
	t.Setenv("YOTTA_ROOT", root)
	opened, err := localruntime.Open(context.Background(), localruntime.Config{
		Executable:  filepath.Join(root, "Yotta.exe"),
		RootLog:     zerolog.New(io.Discard),
		WorkflowLog: discardWorkflowLog{},
		Now:         time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if opened.Roots.Root != filepath.Clean(root) {
		t.Fatalf("resolved storage root = %q, want %q", opened.Roots.Root, filepath.Clean(root))
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := opened.Close(closeCtx); err != nil {
		t.Fatal(err)
	}
}

func TestOpenDoesNotRequireOptionalCodexCLI(t *testing.T) {
	root := t.TempDir()
	config := localruntime.Config{
		StorageRoot: root, Executable: filepath.Join(root, "Yotta.exe"),
		RootLog: zerolog.New(io.Discard), WorkflowLog: discardWorkflowLog{}, Now: time.Now,
		AISecrets: services.NewAISecrets(securestore.New()),
	}
	opened, err := localruntime.Open(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = opened.Settings.MutateSettings(func(settings *services.Settings) error {
		settings.AI.Profiles = []services.AIModelSettings{{
			Slot: "codex", Label: "Codex", Provider: ai.ProviderCodexSubscription,
			Endpoint: "codex://subscription", Model: "gpt-5.6-sol", MaxOutputTokens: 4096,
			Capabilities: ai.ProfileCapabilities{StructuredOutput: true, ToolCalling: true, ParallelTools: true},
			Evaluation:   ai.EvaluationUnverified,
		}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", t.TempDir())
	reopened, err := localruntime.Open(context.Background(), config)
	if err != nil {
		t.Fatalf("configured optional Codex profile blocked local runtime startup: %v", err)
	}
	if err := reopened.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestDesktopAndCLIShareTheLocalRuntimeSeam(t *testing.T) {
	for name, path := range map[string]string{
		"desktop": filepath.Join("..", "desktopapp", "desktop.go"),
		"cli":     filepath.Join("..", "..", "cmd", "yotta", "main.go"),
	} {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(source)
		if !strings.Contains(text, "localruntime.Open(") {
			t.Fatalf("%s composition does not use localruntime.Open", name)
		}
		for _, duplicate := range []string{
			"storage.Open(",
			"catalog.Open(",
			"appbootstrap.Build(",
			"ai.Install(",
			"httpegress.Install(",
			"appcontrol.Install(",
			"automationinstalled.Install(",
			"scriptengine.NewRuntime(",
		} {
			if strings.Contains(text, duplicate) {
				t.Fatalf("%s composition duplicates local runtime decision %q", name, duplicate)
			}
		}
	}
}

type discardWorkflowLog struct{}

func (discardWorkflowLog) EmitWorkflowLog(context.Context, noderuntime.LogEntry) error {
	return nil
}
