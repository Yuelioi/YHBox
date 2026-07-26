package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/yottaapp/yotta/internal/pluginprotocol"
)

func main() {
	write := flag.Bool("write", false, "update generated plugin SDK artifacts")
	flag.Parse()
	if err := run(*write); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(write bool) error {
	protoSource, err := os.ReadFile(filepath.FromSlash("contracts/plugin/v1/plugin.proto"))
	if err != nil {
		return err
	}
	protoDigest := sha256.Sum256(protoSource)
	outputs, err := generatedOutputs(hex.EncodeToString(protoDigest[:]))
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(outputs))
	for path := range outputs {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := compareOrWrite(path, outputs[path], write); err != nil {
			return err
		}
	}
	return checkGeneratedProto(write)
}

func generatedOutputs(protoDigest string) (map[string][]byte, error) {
	goContract, err := format.Source([]byte(fmt.Sprintf(goContractTemplate, protoDigest)))
	if err != nil {
		return nil, err
	}
	vectors, err := conformanceVectors()
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		"contracts/plugin/v1/plugin.wit":               []byte(witContract),
		"contracts/plugin/v1/conformance/vectors.json": vectors,
		"sdk/plugin/go/contract_gen.go":                goContract,
		"sdk/plugin/typescript/contract.ts":            []byte(fmt.Sprintf(typeScriptContractTemplate, protoDigest)),
		"sdk/plugin/node-reference.md":                 []byte(nodeReference),
	}, nil
}

func compareOrWrite(path string, want []byte, write bool) error {
	path = filepath.FromSlash(path)
	if write {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, want, 0o644)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("generated plugin artifact %s is missing: %w", path, err)
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("generated plugin artifact %s drifted; run task plugins:update", path)
	}
	return nil
}

func checkGeneratedProto(write bool) error {
	temporary, err := os.MkdirTemp("", "yotta-plugin-proto-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	command := exec.Command("protoc", "--go_out="+temporary, "--go_opt=module=github.com/yottaapp/yotta", "contracts/plugin/v1/plugin.proto")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("generate plugin Protobuf: %w\n%s", err, output)
	}
	generated := filepath.Join(temporary, "internal", "pluginprotocol", "plugin.pb.go")
	want, err := os.ReadFile(generated)
	if err != nil {
		return err
	}
	return compareOrWrite("internal/pluginprotocol/plugin.pb.go", want, write)
}

type vector struct {
	Name        string `json:"name"`
	Direction   string `json:"direction"`
	FrameBase64 string `json:"frameBase64"`
	SHA256      string `json:"sha256"`
}

func conformanceVectors() ([]byte, error) {
	frames := []struct {
		name, direction string
		frame           *pluginprotocol.Frame
	}{
		{"minimal-invocation", "host-to-guest", &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 1,
			Payload: &pluginprotocol.Frame_Invocation{Invocation: &pluginprotocol.Invocation{
				RequestId: "request-1", InvocationId: "invocation-1", GraphId: "main", NodeId: "node-1", Attempt: 1,
				ObservedUnixMillis: 1, DeadlineUnixMillis: 2, NodeRefJson: []byte(`{}`), ImplementationLockJson: []byte(`{}`), ConfigJson: []byte(`{}`),
				Budget: &pluginprotocol.Budget{MaxFrameBytes: pluginprotocol.MaxFrameBytes, MaxOutputBytes: pluginprotocol.MaxFrameBytes, MaxHostCalls: 8, MaxStatusEvents: 8},
			}}}},
		{"successful-result", "guest-to-host", &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 2,
			Payload: &pluginprotocol.Frame_Result{Result: &pluginprotocol.Result{Outcome: pluginprotocol.Outcome_OUTCOME_SUCCEEDED, TerminationStrength: "cooperative"}}}},
		{"capability-denied", "host-to-guest", &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 3,
			Payload: &pluginprotocol.Frame_HostOpenResponse{HostOpenResponse: &pluginprotocol.HostOpenResponse{
				RequestId: "open-1", Failure: &pluginprotocol.Failure{Code: "plugin.host_call_failed", Message: "resource open was denied"},
			}}}},
		{"deadline-cancel", "host-to-guest", &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 4,
			Payload: &pluginprotocol.Frame_Cancel{Cancel: &pluginprotocol.Cancel{Reason: "deadline_exceeded"}}}},
	}
	result := make([]vector, 0, len(frames))
	for _, item := range frames {
		raw, err := pluginprotocol.MarshalFrame(item.frame)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(raw)
		result = append(result, vector{
			Name: item.name, Direction: item.direction, FrameBase64: base64.StdEncoding.EncodeToString(raw), SHA256: hex.EncodeToString(digest[:]),
		})
	}
	raw, err := json.MarshalIndent(struct {
		Protocol string   `json:"protocol"`
		Vectors  []vector `json:"vectors"`
	}{pluginprotocol.Protocol, result}, "", "  ")
	return append(raw, '\n'), err
}

const goContractTemplate = `// Code generated by cmd/plugin-sdk; DO NOT EDIT.

package pluginsdk

import "github.com/yottaapp/yotta/internal/pluginprotocol"

const (
	Protocol = pluginprotocol.Protocol
	ProcessGuestArgument = "--yotta-plugin-process-v1"
	ProtoSHA256 = "%s"
)

type Invocation = pluginprotocol.Invocation
type PortValue = pluginprotocol.PortValue
type Trigger = pluginprotocol.Trigger
type Budget = pluginprotocol.Budget
type Counter = pluginprotocol.Counter
type Fact = pluginprotocol.Fact
type Failure = pluginprotocol.Failure
type Action = pluginprotocol.ActionEvent
type HostOpenResponse = pluginprotocol.HostOpenResponse
type HostInvokeResponse = pluginprotocol.HostInvokeResponse
type HostDropResponse = pluginprotocol.HostDropResponse
type HostEntropyResponse = pluginprotocol.HostEntropyResponse
type HostWaitResponse = pluginprotocol.HostWaitResponse
type StateReadResponse = pluginprotocol.StateReadResponse
type StateWriteResponse = pluginprotocol.StateWriteResponse
`

const witContract = `package yotta:plugin@1.0.0;

interface execution {
  type bytes = list<u8>;
  record invocation { frame: bytes }
  record exchange-request { frame: bytes, response-capacity: u32 }
  variant exchange-result { emitted, response(bytes), resize(u32), rejected }
  exchange: func(request: exchange-request) -> exchange-result;
}

world node-plugin {
  import execution;
  export run: func(invocation: execution.invocation) -> result<_, string>;
}
`

const typeScriptContractTemplate = `// Code generated by cmd/plugin-sdk; DO NOT EDIT.
export const protocol = "yotta.plugin/1" as const;
export const protoSHA256 = "%s" as const;
export type CanonicalJSON = Uint8Array & { readonly __canonicalJSON: unique symbol };
export type ValueEnvelope = CanonicalJSON & { readonly __valueEnvelope: unique symbol };
export interface NodeRef { nodeTypeId: string; version: string; semanticDigest: string }
export interface ImplementationLock { packageId: string; artifactDigest: string; abi: { kind: "process" | "wit"; version: "v1" }; entrypoint: string }
export interface Budget { maxFrameBytes: bigint; maxOutputBytes: bigint; maxHostCalls: number; maxStatusEvents: number }
export interface PortValue { portId: string; valueEnvelope: ValueEnvelope }
export interface Invocation { requestId: string; invocationId: string; graphId: string; nodeId: string; attempt: number; observedUnixMillis: bigint; deadlineUnixMillis: bigint; nodeRefJSON: CanonicalJSON; implementationLockJSON: CanonicalJSON; configJSON: CanonicalJSON; inputs: readonly PortValue[]; budget: Budget }
export type TerminationStrength = "cooperative" | "engine_interrupt" | "job_terminate" | "process_crash";
export interface Failure { code: string; output?: string; message: string }
`

const nodeReference = `# Yotta plugin node reference

A package contributes immutable Node Contracts. Node identity is the stable nodeTypeId plus an explicit SemVer version and semanticDigest. The runtime never derives a node name from the Yotta application version.

Each implementation is pinned by packageId, manifest artifactDigest, ABI kind/version, and entrypoint. Process v1 payloads are Windows PE executables; WIT v1 payloads are application/wasm modules executed by the trusted runner without WASI.

Guests receive canonical Value Envelopes and must return the exact declared output ports. Resource, state, entropy, wait, status, and action operations cross the mediated protocol; filesystem paths, credentials, native handles, frontend JavaScript, Vue, and DOM access are not plugin APIs.
`
