package pluginprotocol

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestFrameRoundTripUsesCanonicalStrictProtobuf(t *testing.T) {
	frame := testInvocationFrame()
	var stream bytes.Buffer
	if err := WriteFrame(&stream, frame); err != nil {
		t.Fatal(err)
	}
	opened, err := ReadFrame(&stream)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(opened, frame) {
		t.Fatalf("opened frame = %#v, want %#v", opened, frame)
	}

	payload, err := (proto.MarshalOptions{Deterministic: true}).Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	payload = protowire.AppendTag(payload, 99, protowire.VarintType)
	payload = protowire.AppendVarint(payload, 1)
	stream.Reset()
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	stream.Write(header[:])
	stream.Write(payload)
	if _, err := ReadFrame(&stream); err == nil || !strings.Contains(err.Error(), "unknown Protobuf") {
		t.Fatalf("unknown field error = %v", err)
	}
}

func TestFrameValidationAcceptsAllHostResponses(t *testing.T) {
	frames := []*Frame{
		{Protocol: Protocol, Sequence: 1, Payload: &Frame_HostOpenResponse{HostOpenResponse: &HostOpenResponse{RequestId: "open", HandleJson: []byte(`{}`)}}},
		{Protocol: Protocol, Sequence: 2, Payload: &Frame_HostInvokeResponse{HostInvokeResponse: &HostInvokeResponse{RequestId: "invoke", Payload: []byte("ok")}}},
		{Protocol: Protocol, Sequence: 3, Payload: &Frame_HostDropResponse{HostDropResponse: &HostDropResponse{RequestId: "drop"}}},
		{Protocol: Protocol, Sequence: 4, Payload: &Frame_HostEntropyResponse{HostEntropyResponse: &HostEntropyResponse{RequestId: "entropy", Entropy: []byte{1}}}},
		{Protocol: Protocol, Sequence: 5, Payload: &Frame_HostWaitResponse{HostWaitResponse: &HostWaitResponse{RequestId: "wait"}}},
		{Protocol: Protocol, Sequence: 6, Payload: &Frame_StateReadResponse{StateReadResponse: &StateReadResponse{RequestId: "read", ValueEnvelope: []byte(`{}`), Revision: 1}}},
		{Protocol: Protocol, Sequence: 7, Payload: &Frame_StateWriteResponse{StateWriteResponse: &StateWriteResponse{RequestId: "write", ValueEnvelope: []byte(`{}`), Revision: 2}}},
		{Protocol: Protocol, Sequence: 8, Payload: &Frame_Cancel{Cancel: &Cancel{Reason: "budget_exceeded"}}},
		{Protocol: Protocol, Sequence: 9, Payload: &Frame_Result{Result: &Result{Outcome: Outcome_OUTCOME_FAILED, Failure: &Failure{Code: "plugin.failed", Message: "failed"}, TerminationStrength: "process_crash"}}},
	}
	for _, frame := range frames {
		if err := ValidateFrame(frame); err != nil {
			t.Fatalf("ValidateFrame(%T) error = %v", frame.Payload, err)
		}
	}
}

func TestFrameIORejectsMissingAndShortStreams(t *testing.T) {
	if err := WriteFrame(nil, testInvocationFrame()); err == nil {
		t.Fatal("WriteFrame accepted nil writer")
	}
	if _, err := ReadFrame(nil); err == nil {
		t.Fatal("ReadFrame accepted nil reader")
	}
	if _, err := ReadFrame(bytes.NewReader([]byte{0, 0, 0, 0})); err == nil {
		t.Fatal("ReadFrame accepted zero length")
	}
	if err := WriteFrame(zeroWriter{}, testInvocationFrame()); err == nil {
		t.Fatal("WriteFrame accepted a zero-byte writer")
	}
}

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) { return 0, nil }

var _ io.Writer = zeroWriter{}

func TestFrameValidationRejectsInvalidBudgetOrderingAndOutcome(t *testing.T) {
	frame := testInvocationFrame()
	frame.GetInvocation().Budget.MaxHostCalls = 0
	if err := ValidateFrame(frame); err == nil {
		t.Fatal("accepted invocation without a host-call budget")
	}

	result := &Frame{Protocol: Protocol, Sequence: 2, Payload: &Frame_Result{Result: &Result{
		Outcome: Outcome_OUTCOME_SUCCEEDED, TerminationStrength: "cooperative",
		ExecOutputs: []string{"out", "out"},
	}}}
	if err := ValidateFrame(result); err == nil {
		t.Fatal("accepted duplicate exec outputs")
	}
	result.GetResult().ExecOutputs = nil
	result.GetResult().Failure = &Failure{Code: "plugin.failed", Message: "failed"}
	if err := ValidateFrame(result); err == nil {
		t.Fatal("accepted a successful result with a failure")
	}
}

func TestFrameValidationCoversMediatedExecutionCalls(t *testing.T) {
	frames := []*Frame{
		{Protocol: Protocol, Sequence: 1, Payload: &Frame_HostEntropyRequest{HostEntropyRequest: &HostEntropyRequest{RequestId: "entropy-1", ByteCount: 32}}},
		{Protocol: Protocol, Sequence: 2, Payload: &Frame_HostWaitRequest{HostWaitRequest: &HostWaitRequest{RequestId: "wait-1", DurationMillis: 5}}},
		{Protocol: Protocol, Sequence: 3, Payload: &Frame_StateReadRequest{StateReadRequest: &StateReadRequest{RequestId: "state-1", AccessId: "cache"}}},
		{Protocol: Protocol, Sequence: 4, Payload: &Frame_StateWriteRequest{StateWriteRequest: &StateWriteRequest{RequestId: "state-2", AccessId: "cache", ValueEnvelope: []byte(`{}`)}}},
		{Protocol: Protocol, Sequence: 5, Payload: &Frame_Action{Action: &ActionEvent{
			EffectId: "write", Action: "plugin.write", Outcome: "succeeded", SummaryCode: "plugin.write_completed",
			Counters: []*Counter{{Key: "items", Value: 1}}, Facts: []*Fact{{Key: "target", Value: "redacted"}},
		}}},
	}
	for _, frame := range frames {
		if err := ValidateFrame(frame); err != nil {
			t.Fatalf("ValidateFrame(%T) error = %v", frame.Payload, err)
		}
	}
	frames[len(frames)-1].GetAction().Facts = []*Fact{{Key: "target"}, {Key: "target"}}
	if err := ValidateFrame(frames[len(frames)-1]); err == nil {
		t.Fatal("accepted duplicate action facts")
	}
}

func testInvocationFrame() *Frame {
	return &Frame{Protocol: Protocol, Sequence: 1, Payload: &Frame_Invocation{Invocation: &Invocation{
		RequestId: "request-1", InvocationId: "invocation-1", GraphId: "main", NodeId: "node-1", Attempt: 1,
		ObservedUnixMillis: 1, DeadlineUnixMillis: 2,
		NodeRefJson: []byte(`{}`), ImplementationLockJson: []byte(`{}`), ConfigJson: []byte(`{}`),
		Inputs: []*PortValue{{PortId: "value", ValueEnvelope: []byte(`{}`)}},
		Budget: &Budget{MaxFrameBytes: MaxFrameBytes, MaxOutputBytes: MaxFrameBytes, MaxHostCalls: 8, MaxStatusEvents: 8},
	}}}
}
