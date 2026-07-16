package pluginhost

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/pluginprotocol"
	"github.com/yottaapp/yotta/internal/processsandbox"
	"github.com/yottaapp/yotta/internal/wasmrunner"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const WasmIsolationHostFeatureID = pluginprotocol.WasmIsolationHostFeatureID

type WasmHostOptions struct {
	RunnerExecutable string
	Execution        ProcessHostOptions
	MemoryLimitPages uint32
	MaxModuleBytes   int64
}

type WasmHost struct {
	executionHost
	runner         *processsandbox.Runner
	runnerImage    processsandbox.Image
	memoryPages    uint32
	maxModuleBytes int64
}

func NewWasmHost(catalog nodecatalog.Snapshot, options WasmHostOptions) (*WasmHost, error) {
	if !catalog.Valid() {
		return nil, errors.New("Wasm plugin host requires a valid Catalog")
	}
	execution, err := normalizeExecutionOptions(options.Execution)
	if err != nil {
		return nil, err
	}
	if options.MemoryLimitPages == 0 {
		options.MemoryLimitPages = 1_024
	}
	if options.MaxModuleBytes == 0 {
		options.MaxModuleBytes = pluginprotocol.MaxWasmModuleBytes
	}
	if options.MemoryLimitPages < pluginprotocol.MinWasmMemoryPages || options.MemoryLimitPages > pluginprotocol.MaxWasmMemoryPages ||
		options.MaxModuleBytes <= 0 || options.MaxModuleBytes > pluginprotocol.MaxWasmModuleBytes {
		return nil, errors.New("Wasm plugin host module budgets are invalid")
	}
	image, err := processsandbox.OpenImage(options.RunnerExecutable)
	if err != nil {
		return nil, fmt.Errorf("open trusted Wasm runner image: %w", err)
	}
	runner, err := processsandbox.New(processsandbox.Options{
		ProfileName: "Yotta.Plugin.Wasm.V1", DisplayName: "Yotta Wasm Plugin",
		Description:        "Isolated third-party Yotta WebAssembly node",
		ProcessMemoryBytes: execution.ProcessMemoryBytes, JobMemoryBytes: execution.JobMemoryBytes,
	})
	if err != nil {
		return nil, err
	}
	return &WasmHost{
		executionHost: executionHost{catalog: catalog, options: execution}, runner: runner, runnerImage: image,
		memoryPages: options.MemoryLimitPages, maxModuleBytes: options.MaxModuleBytes,
	}, nil
}

func (host *WasmHost) HostFeatures() []string {
	if host == nil || host.runner == nil || !host.runner.Available() {
		return []string{}
	}
	return []string{WasmIsolationHostFeatureID}
}

func (host *WasmHost) Adapters(packages []nodepackage.RuntimePackage) (map[string]compiler.InstalledAdapter, error) {
	if host == nil || host.runner == nil || !host.catalog.Valid() {
		return nil, errors.New("Wasm plugin host is not initialized")
	}
	result := map[string]compiler.InstalledAdapter{}
	for _, runtimePackage := range packages {
		for _, node := range runtimePackage.Nodes {
			if node.Implementation.ABI.Kind != nodecontract.ABIWIT {
				continue
			}
			if err := host.validateRuntimeNode(runtimePackage, node, nodecontract.ABIWIT); err != nil {
				return nil, err
			}
			if node.Payload.Metadata().MediaType != "application/wasm" {
				return nil, fmt.Errorf("Wasm plugin %q does not contain an application/wasm payload", node.Lock.Entrypoint)
			}
			if _, duplicate := result[node.Lock.Entrypoint]; duplicate {
				return nil, fmt.Errorf("duplicate Wasm plugin entrypoint %q", node.Lock.Entrypoint)
			}
			pinned := node
			result[node.Lock.Entrypoint] = compiler.InstalledAdapter{
				Implementation: node.Lock,
				Run: func(ctx context.Context, invocation compiler.Invocation) (compiler.AdapterResult, error) {
					return host.invokeWasm(ctx, pinned, invocation)
				},
			}
		}
	}
	return result, nil
}

func (host *WasmHost) invokeWasm(parent context.Context, node nodepackage.RuntimeNode, invocation compiler.Invocation) (compiler.AdapterResult, error) {
	initial, deadline, err := host.invocationFrame(parent, node, invocation)
	if err != nil {
		return compiler.AdapterResult{}, err
	}
	module, err := node.Payload.Read(parent, host.maxModuleBytes)
	if err != nil {
		return compiler.AdapterResult{}, fmt.Errorf("read Wasm plugin payload: %w", err)
	}
	executionContext, cancelExecution := context.WithDeadline(parent, deadline)
	defer cancelExecution()
	session := &processSession{
		catalog: host.catalog, invocation: invocation,
		nextSequence: 2, maxHostCalls: host.options.MaxHostCalls, maxStatusEvents: host.options.MaxStatusEvents,
	}
	result, err := executeSandboxed(executionContext, host.runner, processsandbox.Request{
		Image: host.runnerImage, Args: []string{wasmrunner.WorkerArgument}, Timeout: time.Until(deadline),
	}, session, wasmrunner.ExitOK, "Wasm plugin runner", func(writer io.Writer) error {
		if err := pluginprotocol.WriteWasmBootstrap(writer, pluginprotocol.WasmBootstrap{
			MemoryLimitPages: host.memoryPages, Module: module,
		}); err != nil {
			return err
		}
		return pluginprotocol.WriteFrame(writer, initial)
	})
	if err != nil {
		return compiler.AdapterResult{}, err
	}
	return host.openResult(result)
}
