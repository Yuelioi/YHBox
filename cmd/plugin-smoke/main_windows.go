//go:build windows

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/hostapi"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/pluginfixture"
	"github.com/yottaapp/yotta/internal/pluginhost"
	"github.com/yottaapp/yotta/internal/wasmrunner"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stdout, "process and Wasm plugin smoke passed")
}

func run() error {
	root, err := os.MkdirTemp("", "yotta-plugin-smoke-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	processExecutable := filepath.Join(root, "ProcessUppercase.exe")
	wasmRunnerExecutable := os.Getenv("YOTTA_WASM_PLUGIN_RUNNER")
	builds := map[string]string{processExecutable: "./examples/plugins/process-uppercase"}
	if wasmRunnerExecutable == "" {
		wasmRunnerExecutable = filepath.Join(root, wasmrunner.WorkerExecutableName)
		builds[wasmRunnerExecutable] = "./cmd/yotta-wasm-plugin-runner"
	}
	for output, source := range builds {
		command := exec.Command("go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags=-w -s -H windowsgui", "-o", output, source)
		if combined, buildErr := command.CombinedOutput(); buildErr != nil {
			return fmt.Errorf("build %s: %w\n%s", source, buildErr, combined)
		}
	}
	processPayload, err := os.ReadFile(processExecutable)
	if err != nil {
		return err
	}
	wasmPayload, err := pluginfixture.MinimalWasmModule()
	if err != nil {
		return err
	}
	fixtures, err := pluginfixture.Create(context.Background(), filepath.Join(root, "packages"), processPayload, wasmPayload)
	if err != nil {
		return err
	}
	packages, err := fixtures.Store.RuntimePackages(context.Background(), nodepackage.RuntimeHost{
		APIGeneration: hostapi.Current, OperatingSystem: "windows", Architecture: "amd64",
	})
	if err != nil {
		return err
	}
	builtins, err := nodes.Build()
	if err != nil {
		return err
	}
	projection, err := pluginhost.MergeCatalog(builtins.Catalog, packages, nodes.GeneratorVersion)
	if err != nil {
		return err
	}
	processHost, err := pluginhost.NewProcessHost(projection.Catalog, pluginhost.ProcessHostOptions{})
	if err != nil {
		return err
	}
	wasmHost, err := pluginhost.NewWasmHost(projection.Catalog, pluginhost.WasmHostOptions{RunnerExecutable: wasmRunnerExecutable})
	if err != nil {
		return err
	}
	processAdapters, err := processHost.Adapters(packages)
	if err != nil {
		return err
	}
	wasmAdapters, err := wasmHost.Adapters(packages)
	if err != nil {
		return err
	}
	input, err := datatype.SealInlineJSON(projection.Catalog, datatype.RefResolvedType(fixtures.StringType.TypeRef()), []byte(`"hello"`))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	processAdapter, err := adapterFor(packages, processAdapters, nodecontract.ABIProcess)
	if err != nil {
		return err
	}
	processResult, err := processAdapter.Run(ctx, invocation("process-1", map[string]datatype.ValueEnvelope{"value": input}))
	if err != nil {
		return err
	}
	output := processResult.Outputs["result"]
	if !output.Valid() || !bytes.Equal(output.InlineJSON(), []byte(`"HELLO"`)) {
		return fmt.Errorf("process plugin output = %s", output.InlineJSON())
	}
	wasmAdapter, err := adapterFor(packages, wasmAdapters, nodecontract.ABIWIT)
	if err != nil {
		return err
	}
	wasmResult, err := wasmAdapter.Run(ctx, invocation("wasm-1", map[string]datatype.ValueEnvelope{}))
	if err != nil {
		return err
	}
	if len(wasmResult.Outputs) != 0 || len(wasmResult.ExecOutputs) != 0 {
		return fmt.Errorf("minimal Wasm plugin returned unexpected outputs")
	}
	return nil
}

func adapterFor(packages []nodepackage.RuntimePackage, adapters map[string]compiler.InstalledAdapter, kind nodecontract.ABIKind) (compiler.InstalledAdapter, error) {
	for _, runtimePackage := range packages {
		for _, node := range runtimePackage.Nodes {
			if node.Implementation.ABI.Kind == kind {
				adapter, ok := adapters[node.Lock.Entrypoint]
				if ok {
					return adapter, nil
				}
			}
		}
	}
	return compiler.InstalledAdapter{}, fmt.Errorf("%s fixture adapter is unavailable", kind)
}

func invocation(id string, inputs map[string]datatype.ValueEnvelope) compiler.Invocation {
	return compiler.Invocation{
		InvocationID: id, Attempt: 1, GraphID: "main", NodeID: id, Config: map[string]any{}, Inputs: inputs,
		ObservedAt: time.Now().UTC(), ReadEntropy: func(buffer []byte) error { return nil },
		Wait: func(ctx context.Context, duration time.Duration) error {
			select {
			case <-time.After(duration):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		EmitStatus:   func(context.Context, string, map[string]int64) error { return nil },
		RecordAction: func(context.Context, compiler.AdapterAction) error { return nil },
	}
}
