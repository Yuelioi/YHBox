package wasmrunner

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"

	"github.com/yottaapp/yotta/internal/pluginprotocol"
)

func TestRunExecutesNarrowExchangeABIWithoutWASI(t *testing.T) {
	resultFrame := &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 2,
		Payload: &pluginprotocol.Frame_Result{Result: &pluginprotocol.Result{
			Outcome: pluginprotocol.Outcome_OUTCOME_SUCCEEDED, TerminationStrength: "cooperative",
		}}}
	resultPayload, err := pluginprotocol.MarshalFrame(resultFrame)
	if err != nil {
		t.Fatal(err)
	}
	module := exchangeFixtureModule(resultPayload)
	invocation := &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 1,
		Payload: &pluginprotocol.Frame_Invocation{Invocation: &pluginprotocol.Invocation{
			RequestId: "request-1", InvocationId: "invocation-1", GraphId: "main", NodeId: "node-1", Attempt: 1,
			ObservedUnixMillis: 1, DeadlineUnixMillis: 2, NodeRefJson: []byte(`{}`), ImplementationLockJson: []byte(`{}`), ConfigJson: []byte(`{}`),
			Budget: &pluginprotocol.Budget{MaxFrameBytes: pluginprotocol.MaxFrameBytes, MaxOutputBytes: pluginprotocol.MaxFrameBytes, MaxHostCalls: 1, MaxStatusEvents: 1},
		}}}
	var input, output bytes.Buffer
	if err := pluginprotocol.WriteWasmBootstrap(&input, pluginprotocol.WasmBootstrap{MemoryLimitPages: 2, Module: module}); err != nil {
		t.Fatal(err)
	}
	if err := pluginprotocol.WriteFrame(&input, invocation); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), &input, &output); err != nil {
		t.Fatal(err)
	}
	got, err := pluginprotocol.ReadFrame(&output)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sequence != 2 || got.GetResult() == nil {
		t.Fatalf("runner output = %#v", got)
	}
}

func exchangeFixtureModule(frame []byte) []byte {
	module := []byte{'\x00', 'a', 's', 'm', '\x01', '\x00', '\x00', '\x00'}
	// Types: exchange, alloc, run.
	types := []byte{3,
		0x60, 4, 0x7f, 0x7f, 0x7f, 0x7f, 1, 0x7e,
		0x60, 1, 0x7f, 1, 0x7f,
		0x60, 2, 0x7f, 0x7f, 1, 0x7f,
	}
	module = appendSection(module, 1, types)
	imports := append([]byte{1}, wasmName(ImportModule)...)
	imports = append(imports, wasmName(ExchangeFunction)...)
	imports = append(imports, 0, 0) // function import, type 0
	module = appendSection(module, 2, imports)
	module = appendSection(module, 3, []byte{2, 1, 2})
	module = appendSection(module, 5, []byte{1, 0, 2}) // one memory, min two pages
	exports := []byte{3}
	exports = append(exports, wasmName("memory")...)
	exports = append(exports, 2, 0)
	exports = append(exports, wasmName(AllocateExport)...)
	exports = append(exports, 0, 1)
	exports = append(exports, wasmName(RunExport)...)
	exports = append(exports, 0, 2)
	module = appendSection(module, 7, exports)
	allocateBody := []byte{0, 0x41, 0x80, 0x08, 0x0b} // i32.const 1024
	runBody := []byte{0, 0x41, 0}
	runBody = append(runBody, 0x41)
	runBody = append(runBody, uleb(uint32(len(frame)))...)
	runBody = append(runBody, 0x41, 0, 0x41, 0, 0x10, 0, 0x1a, 0x41, 0, 0x0b)
	code := []byte{2}
	code = append(code, uleb(uint32(len(allocateBody)))...)
	code = append(code, allocateBody...)
	code = append(code, uleb(uint32(len(runBody)))...)
	code = append(code, runBody...)
	module = appendSection(module, 10, code)
	data := []byte{1, 0, 0x41, 0, 0x0b}
	data = append(data, uleb(uint32(len(frame)))...)
	data = append(data, frame...)
	return appendSection(module, 11, data)
}

func appendSection(module []byte, id byte, payload []byte) []byte {
	module = append(module, id)
	module = append(module, uleb(uint32(len(payload)))...)
	return append(module, payload...)
}

func wasmName(value string) []byte {
	result := uleb(uint32(len(value)))
	return append(result, value...)
}

func uleb(value uint32) []byte {
	var scratch [binary.MaxVarintLen32]byte
	count := binary.PutUvarint(scratch[:], uint64(value))
	return append([]byte(nil), scratch[:count]...)
}
