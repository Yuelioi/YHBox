// Package wasmrunner implements the one-shot trusted process which embeds an
// untrusted Wasm plugin without WASI or any ambient host imports.
package wasmrunner

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/yottaapp/yotta/internal/pluginprotocol"
)

const (
	WorkerArgument       = "--yotta-wasm-plugin-runner-v1"
	WorkerExecutableName = "Yotta.WasmPluginRunner.exe"
	ImportModule         = "yotta_plugin_v1"
	ExchangeFunction     = "exchange"
	AllocateExport       = "yotta_alloc"
	RunExport            = "yotta_run"
	ExitOK               = 0
	ExitInvalid          = 40
	ExitRuntime          = 41
)

func Run(ctx context.Context, input io.Reader, output io.Writer) error {
	if ctx == nil || input == nil || output == nil {
		return errors.New("Wasm runner requires context and protocol streams")
	}
	bootstrap, err := pluginprotocol.ReadWasmBootstrap(input)
	if err != nil {
		return err
	}
	invocation, err := pluginprotocol.ReadFrame(input)
	if err != nil {
		return fmt.Errorf("read Wasm invocation: %w", err)
	}
	if invocation.Sequence != 1 || invocation.GetInvocation() == nil {
		return errors.New("Wasm runner requires invocation frame sequence 1")
	}
	invocationPayload, err := pluginprotocol.MarshalFrame(invocation)
	if err != nil {
		return err
	}

	configuration := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(bootstrap.MemoryLimitPages).
		WithCloseOnContextDone(true).
		WithDebugInfoEnabled(false)
	runtime := wazero.NewRuntimeWithConfig(ctx, configuration)
	defer runtime.Close(ctx)
	exchange := &exchangeState{input: input, output: output, nextSequence: 2, responses: map[[32]byte][]byte{}}
	if _, err := runtime.NewHostModuleBuilder(ImportModule).
		NewFunctionBuilder().WithFunc(exchange.call).Export(ExchangeFunction).
		Instantiate(ctx); err != nil {
		return fmt.Errorf("instantiate Yotta Wasm imports: %w", err)
	}
	compiled, err := runtime.CompileModule(ctx, bootstrap.Module)
	if err != nil {
		return fmt.Errorf("compile Wasm plugin: %w", err)
	}
	defer compiled.Close(ctx)
	module, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName("yotta-plugin"))
	if err != nil {
		return fmt.Errorf("instantiate Wasm plugin: %w", err)
	}
	defer module.Close(ctx)
	allocate := module.ExportedFunction(AllocateExport)
	run := module.ExportedFunction(RunExport)
	if module.Memory() == nil || allocate == nil || run == nil {
		return errors.New("Wasm plugin is missing memory, yotta_alloc, or yotta_run")
	}
	allocated, err := allocate.Call(ctx, uint64(len(invocationPayload)))
	if err != nil || len(allocated) != 1 || allocated[0] > math.MaxUint32 {
		return errors.New("Wasm plugin failed to allocate invocation memory")
	}
	pointer := uint32(allocated[0])
	if !module.Memory().Write(pointer, invocationPayload) {
		return errors.New("Wasm plugin invocation memory is out of bounds")
	}
	outcome, err := run.Call(ctx, uint64(pointer), uint64(len(invocationPayload)))
	if err != nil {
		return fmt.Errorf("execute Wasm plugin: %w", err)
	}
	if len(outcome) != 1 || uint32(outcome[0]) != 0 {
		return errors.New("Wasm plugin returned a failing runner status")
	}
	if err := exchange.failure(); err != nil {
		return err
	}
	if !exchange.completed() {
		return errors.New("Wasm plugin returned without emitting a result")
	}
	return nil
}

type exchangeState struct {
	mu           sync.Mutex
	input        io.Reader
	output       io.Writer
	nextSequence uint64
	responses    map[[32]byte][]byte
	result       bool
	err          error
}

func (state *exchangeState) call(_ context.Context, module api.Module, requestPointer, requestLength, responsePointer, responseCapacity uint32) uint64 {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.err != nil || state.result || requestLength == 0 || requestLength > pluginprotocol.MaxFrameBytes {
		state.fail(errors.New("Wasm plugin exchange is invalid or already complete"))
		return math.MaxUint64
	}
	view, ok := module.Memory().Read(requestPointer, requestLength)
	if !ok {
		state.fail(errors.New("Wasm plugin exchange request escapes linear memory"))
		return math.MaxUint64
	}
	requestBytes := append([]byte(nil), view...)
	key := sha256.Sum256(requestBytes)
	if cached, ok := state.responses[key]; ok {
		return state.copyResponse(module, key, cached, responsePointer, responseCapacity)
	}
	request, err := pluginprotocol.UnmarshalFrame(requestBytes)
	if err != nil {
		state.fail(fmt.Errorf("Wasm plugin emitted an invalid frame at sequence %d: %w", state.nextSequence, err))
		return math.MaxUint64
	}
	if request.Sequence != state.nextSequence || !guestPayload(request) {
		state.fail(fmt.Errorf("Wasm plugin emitted an invalid frame at sequence %d", state.nextSequence))
		return math.MaxUint64
	}
	state.nextSequence++
	if err := pluginprotocol.WriteFrame(state.output, request); err != nil {
		state.fail(err)
		return math.MaxUint64
	}
	if request.GetResult() != nil {
		state.result = true
		return 0
	}
	if !requiresResponse(request) {
		return 0
	}
	response, err := pluginprotocol.ReadFrame(state.input)
	if err != nil {
		state.fail(fmt.Errorf("Wasm plugin host response is invalid at sequence %d: %w", state.nextSequence, err))
		return math.MaxUint64
	}
	if response.Sequence != state.nextSequence || !matchesResponse(request, response) {
		state.fail(fmt.Errorf("Wasm plugin host response is invalid at sequence %d", state.nextSequence))
		return math.MaxUint64
	}
	state.nextSequence++
	responseBytes, err := pluginprotocol.MarshalFrame(response)
	if err != nil {
		state.fail(err)
		return math.MaxUint64
	}
	state.responses[key] = responseBytes
	return state.copyResponse(module, key, responseBytes, responsePointer, responseCapacity)
}

func (state *exchangeState) copyResponse(module api.Module, key [32]byte, response []byte, pointer, capacity uint32) uint64 {
	if uint32(len(response)) > capacity {
		return uint64(-int64(len(response)))
	}
	if len(response) != 0 && !module.Memory().Write(pointer, response) {
		state.fail(errors.New("Wasm plugin exchange response escapes linear memory"))
		return math.MaxUint64
	}
	delete(state.responses, key)
	return uint64(len(response))
}

func (state *exchangeState) fail(err error) {
	if state.err == nil {
		state.err = err
	}
}

func (state *exchangeState) failure() error {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.err
}

func (state *exchangeState) completed() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.result
}

func guestPayload(frame *pluginprotocol.Frame) bool {
	switch frame.Payload.(type) {
	case *pluginprotocol.Frame_HostOpenRequest, *pluginprotocol.Frame_HostInvokeRequest, *pluginprotocol.Frame_HostDropRequest,
		*pluginprotocol.Frame_HostEntropyRequest, *pluginprotocol.Frame_HostWaitRequest,
		*pluginprotocol.Frame_StateReadRequest, *pluginprotocol.Frame_StateWriteRequest,
		*pluginprotocol.Frame_Status, *pluginprotocol.Frame_Action, *pluginprotocol.Frame_Result:
		return true
	default:
		return false
	}
}

func requiresResponse(frame *pluginprotocol.Frame) bool {
	switch frame.Payload.(type) {
	case *pluginprotocol.Frame_HostOpenRequest, *pluginprotocol.Frame_HostInvokeRequest, *pluginprotocol.Frame_HostDropRequest,
		*pluginprotocol.Frame_HostEntropyRequest, *pluginprotocol.Frame_HostWaitRequest,
		*pluginprotocol.Frame_StateReadRequest, *pluginprotocol.Frame_StateWriteRequest:
		return true
	default:
		return false
	}
}

func matchesResponse(request, response *pluginprotocol.Frame) bool {
	switch requestPayload := request.Payload.(type) {
	case *pluginprotocol.Frame_HostOpenRequest:
		return response.GetHostOpenResponse().GetRequestId() == requestPayload.HostOpenRequest.RequestId
	case *pluginprotocol.Frame_HostInvokeRequest:
		return response.GetHostInvokeResponse().GetRequestId() == requestPayload.HostInvokeRequest.RequestId
	case *pluginprotocol.Frame_HostDropRequest:
		return response.GetHostDropResponse().GetRequestId() == requestPayload.HostDropRequest.RequestId
	case *pluginprotocol.Frame_HostEntropyRequest:
		return response.GetHostEntropyResponse().GetRequestId() == requestPayload.HostEntropyRequest.RequestId
	case *pluginprotocol.Frame_HostWaitRequest:
		return response.GetHostWaitResponse().GetRequestId() == requestPayload.HostWaitRequest.RequestId
	case *pluginprotocol.Frame_StateReadRequest:
		return response.GetStateReadResponse().GetRequestId() == requestPayload.StateReadRequest.RequestId
	case *pluginprotocol.Frame_StateWriteRequest:
		return response.GetStateWriteResponse().GetRequestId() == requestPayload.StateWriteRequest.RequestId
	default:
		return false
	}
}
