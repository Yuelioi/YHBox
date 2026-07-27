//go:build windows

package pluginhost

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/pluginfixture"
)

func TestInstalledPackageFixturesProjectBothHostsAndRevokeExistingAdapters(t *testing.T) {
	ctx := context.Background()
	wasmModule, err := pluginfixture.MinimalWasmModule()
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := pluginfixture.Create(ctx, filepath.Join(t.TempDir(), "packages"), []byte("MZ-process-fixture"), wasmModule)
	if err != nil {
		t.Fatal(err)
	}
	packages, err := fixtures.Store.RuntimePackages(ctx, nodepackage.RuntimeHost{APIGeneration: "1.0", OperatingSystem: "windows", Architecture: "amd64"})
	if err != nil || len(packages) != 2 {
		t.Fatalf("RuntimePackages = %#v, %v", packages, err)
	}
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	projection, err := MergeCatalog(builtins.Catalog, packages, nodes.GeneratorVersion)
	if err != nil {
		t.Fatal(err)
	}
	processHost, err := NewProcessHost(projection.Catalog, ProcessHostOptions{})
	if err != nil {
		t.Fatal(err)
	}
	testExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	wasmHost, err := NewWasmHost(projection.Catalog, WasmHostOptions{RunnerExecutable: testExecutable})
	if err != nil {
		t.Fatal(err)
	}
	processAdapters, err := processHost.Adapters(packages)
	if err != nil || len(processAdapters) != 1 {
		t.Fatalf("process adapters = %#v, %v", processAdapters, err)
	}
	wasmAdapters, err := wasmHost.Adapters(packages)
	if err != nil || len(wasmAdapters) != 1 {
		t.Fatalf("Wasm adapters = %#v, %v", wasmAdapters, err)
	}
	if _, err := fixtures.Store.Disable(fixtures.ProcessManifest.PackageID()); err != nil {
		t.Fatal(err)
	}
	assertAdaptersRevoked(t, processAdapters)
	if err := fixtures.Store.Uninstall(fixtures.WasmManifest.PackageID()); err != nil {
		t.Fatal(err)
	}
	assertAdaptersRevoked(t, wasmAdapters)
}

func assertAdaptersRevoked(t *testing.T, adapters map[string]nodeadapter.InstalledAdapter) {
	t.Helper()
	for _, adapter := range adapters {
		_, err := adapter.Run(context.Background(), nodeadapter.Invocation{
			InvocationID: "invocation-1", Attempt: 1, GraphID: "main", NodeID: "plugin", Config: map[string]any{},
			Inputs: map[string]datatype.ValueEnvelope{}, ObservedAt: time.Now().UTC(),
		})
		if err == nil {
			t.Fatal("revoked package adapter remained executable")
		}
	}
}
