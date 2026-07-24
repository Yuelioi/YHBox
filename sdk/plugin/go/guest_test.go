package pluginsdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/pluginprotocol"
)

func TestGuestOrdersInvocationStatusAndResult(t *testing.T) {
	invocation := testInvocationFrame()
	var input, output bytes.Buffer
	if err := pluginprotocol.WriteFrame(&input, invocation); err != nil {
		t.Fatal(err)
	}
	guest, err := NewGuest(&input, &output)
	if err != nil {
		t.Fatal(err)
	}
	if opened, err := guest.ReceiveInvocation(); err != nil || opened.InvocationId != "invocation-1" {
		t.Fatalf("ReceiveInvocation = %#v, %v", opened, err)
	}
	if err := guest.Status("plugin.progress", map[string]int64{"items": 1}); err != nil {
		t.Fatal(err)
	}
	if err := guest.Succeed(map[string][]byte{}, nil); err != nil {
		t.Fatal(err)
	}
	status, err := pluginprotocol.ReadFrame(&output)
	if err != nil {
		t.Fatal(err)
	}
	result, err := pluginprotocol.ReadFrame(&output)
	if err != nil {
		t.Fatal(err)
	}
	if status.Sequence != 2 || status.GetStatus() == nil || result.Sequence != 3 || result.GetResult() == nil {
		t.Fatalf("status/result = %#v / %#v", status, result)
	}
}

func TestGuestHostCallRejectsMismatchedResponse(t *testing.T) {
	guestSide, hostSide := net.Pipe()
	defer guestSide.Close()
	defer hostSide.Close()
	guest, err := NewGuest(guestSide, guestSide)
	if err != nil {
		t.Fatal(err)
	}
	guest.next = 2
	go func() {
		request, _ := pluginprotocol.ReadFrame(hostSide)
		_ = pluginprotocol.WriteFrame(hostSide, &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: request.Sequence + 1,
			Payload: &pluginprotocol.Frame_HostEntropyResponse{HostEntropyResponse: &pluginprotocol.HostEntropyResponse{RequestId: "wrong", Entropy: []byte{1}}}})
	}()
	if _, err := guest.Entropy("entropy-1", 1); err == nil {
		t.Fatal("Guest accepted a mismatched host response")
	}
}

func TestGuestExercisesMediatedCallsAndCanonicalResults(t *testing.T) {
	var input, output bytes.Buffer
	frames := []*pluginprotocol.Frame{
		testInvocationFrame(),
		{Protocol: pluginprotocol.Protocol, Sequence: 3, Payload: &pluginprotocol.Frame_HostOpenResponse{HostOpenResponse: &pluginprotocol.HostOpenResponse{RequestId: "open-1", HandleJson: []byte(`{}`)}}},
		{Protocol: pluginprotocol.Protocol, Sequence: 5, Payload: &pluginprotocol.Frame_HostInvokeResponse{HostInvokeResponse: &pluginprotocol.HostInvokeResponse{RequestId: "invoke-1", Payload: []byte("ok")}}},
		{Protocol: pluginprotocol.Protocol, Sequence: 7, Payload: &pluginprotocol.Frame_HostDropResponse{HostDropResponse: &pluginprotocol.HostDropResponse{RequestId: "drop-1"}}},
		{Protocol: pluginprotocol.Protocol, Sequence: 9, Payload: &pluginprotocol.Frame_HostEntropyResponse{HostEntropyResponse: &pluginprotocol.HostEntropyResponse{RequestId: "entropy-1", Entropy: []byte{1}}}},
		{Protocol: pluginprotocol.Protocol, Sequence: 11, Payload: &pluginprotocol.Frame_HostWaitResponse{HostWaitResponse: &pluginprotocol.HostWaitResponse{RequestId: "wait-1"}}},
		{Protocol: pluginprotocol.Protocol, Sequence: 13, Payload: &pluginprotocol.Frame_StateReadResponse{StateReadResponse: &pluginprotocol.StateReadResponse{RequestId: "read-1", ValueEnvelope: []byte(`{}`), Revision: 1}}},
		{Protocol: pluginprotocol.Protocol, Sequence: 15, Payload: &pluginprotocol.Frame_StateWriteResponse{StateWriteResponse: &pluginprotocol.StateWriteResponse{RequestId: "write-1", ValueEnvelope: []byte(`{}`), Revision: 2}}},
	}
	for _, frame := range frames {
		if err := pluginprotocol.WriteFrame(&input, frame); err != nil {
			t.Fatal(err)
		}
	}
	guest, err := NewGuest(&input, &output)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := guest.ReceiveInvocation(); err != nil {
		t.Fatal(err)
	}
	if _, err := guest.Open("open-1", "filesystem", []string{"write", "read"}, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := guest.Invoke("invoke-1", "filesystem", []byte(`{}`), "read", []byte("request")); err != nil {
		t.Fatal(err)
	}
	if _, err := guest.Drop("drop-1", "filesystem", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := guest.Entropy("entropy-1", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := guest.Wait("wait-1", time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if _, err := guest.ReadState("read-1", "cache"); err != nil {
		t.Fatal(err)
	}
	if _, err := guest.WriteState("write-1", "cache", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := guest.Record(&Action{EffectId: "effect", Action: "plugin.write", Outcome: "succeeded", SummaryCode: "plugin.write_completed", Counters: []*pluginprotocol.Counter{{Key: "z"}, {Key: "a"}}, Facts: []*pluginprotocol.Fact{{Key: "z"}, {Key: "a"}}}); err != nil {
		t.Fatal(err)
	}
	if err := guest.Fail("plugin.failed", "", "failed"); err != nil {
		t.Fatal(err)
	}
	if err := guest.Fail("plugin.failed", "", "again"); err == nil {
		t.Fatal("Guest emitted a second terminal result")
	}
}

func TestInlineValueHelpersVerifyAndResealDigest(t *testing.T) {
	document := envelopeDocument{Format: valueEnvelopeFormat, Version: valueEnvelopeVersion, Type: json.RawMessage(`{}`), Repr: "inline-json", Codec: "yotta.jcs/v1", Value: json.RawMessage(`{"a":1}`)}
	var err error
	document.ValueDigest, err = envelopeDigest(document)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := artifact.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	value, err := InlineJSON(envelope)
	if err != nil || string(value) != `{"a":1}` {
		t.Fatalf("InlineJSON = %s, %v", value, err)
	}
	replaced, err := ReplaceInlineJSON(envelope, []byte(`{"b":2,"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	value, err = InlineJSON(replaced)
	if err != nil || string(value) != `{"a":1,"b":2}` {
		t.Fatalf("replaced value = %s, %v", value, err)
	}
	tampered := append([]byte(nil), envelope...)
	tampered[len(tampered)-2] = '2'
	if _, err := InlineJSON(tampered); err == nil {
		t.Fatal("InlineJSON accepted a tampered envelope")
	}
	if _, err := ReplaceInlineJSON(envelope, []byte(`{`)); err == nil {
		t.Fatal("ReplaceInlineJSON accepted invalid JSON")
	}
}

func TestGuestRejectsInvalidConstructionAndRepeatedInvocation(t *testing.T) {
	if _, err := NewGuest(nil, &bytes.Buffer{}); err == nil {
		t.Fatal("NewGuest accepted nil input")
	}
	var input, output bytes.Buffer
	if err := pluginprotocol.WriteFrame(&input, testInvocationFrame()); err != nil {
		t.Fatal(err)
	}
	guest, err := NewGuest(&input, &output)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := guest.ReceiveInvocation(); err != nil {
		t.Fatal(err)
	}
	if _, err := guest.ReceiveInvocation(); err == nil {
		t.Fatal("Guest accepted a repeated invocation")
	}
}

func TestGuestRejectsWrongResponseKindsForEveryMediatedCall(t *testing.T) {
	newGuest := func(t *testing.T) *Guest {
		t.Helper()
		var input bytes.Buffer
		if err := pluginprotocol.WriteFrame(&input, &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 3, Payload: &pluginprotocol.Frame_HostEntropyResponse{HostEntropyResponse: &pluginprotocol.HostEntropyResponse{RequestId: "wrong", Entropy: []byte{1}}}}); err != nil {
			t.Fatal(err)
		}
		guest, err := NewGuest(&input, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		guest.next = 2
		return guest
	}
	calls := []func(*Guest) error{
		func(guest *Guest) error {
			_, err := guest.Open("open", "resource", []string{"read"}, []byte(`{}`))
			return err
		},
		func(guest *Guest) error {
			_, err := guest.Invoke("invoke", "resource", []byte(`{}`), "read", nil)
			return err
		},
		func(guest *Guest) error { _, err := guest.Drop("drop", "resource", []byte(`{}`)); return err },
		func(guest *Guest) error { _, err := guest.Wait("wait", time.Millisecond); return err },
		func(guest *Guest) error { _, err := guest.ReadState("read", "state"); return err },
		func(guest *Guest) error { _, err := guest.WriteState("write", "state", []byte(`{}`)); return err },
	}
	for _, call := range calls {
		if err := call(newGuest(t)); err == nil {
			t.Fatal("Guest accepted a response of the wrong kind")
		}
	}
}

func TestGuestRejectsInactiveCancelledAndBrokenStreams(t *testing.T) {
	guest, err := NewGuest(bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := guest.Status("plugin.progress", nil); err == nil {
		t.Fatal("inactive Guest emitted status")
	}
	if err := guest.Record(nil); err == nil {
		t.Fatal("Guest accepted a nil action")
	}
	if _, err := guest.ReceiveInvocation(); err == nil {
		t.Fatal("Guest accepted an empty invocation stream")
	}
	var wrongInput bytes.Buffer
	if err := pluginprotocol.WriteFrame(&wrongInput, &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 1, Payload: &pluginprotocol.Frame_Cancel{Cancel: &pluginprotocol.Cancel{Reason: "user_cancelled"}}}); err != nil {
		t.Fatal(err)
	}
	wrongGuest, _ := NewGuest(&wrongInput, &bytes.Buffer{})
	if _, err := wrongGuest.ReceiveInvocation(); err == nil {
		t.Fatal("Guest accepted a non-invocation first frame")
	}
	var cancelInput bytes.Buffer
	if err := pluginprotocol.WriteFrame(&cancelInput, &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 3, Payload: &pluginprotocol.Frame_Cancel{Cancel: &pluginprotocol.Cancel{Reason: "budget_exceeded"}}}); err != nil {
		t.Fatal(err)
	}
	cancelled, _ := NewGuest(&cancelInput, &bytes.Buffer{})
	cancelled.next = 2
	if _, err := cancelled.Entropy("entropy", 1); err == nil {
		t.Fatal("Guest ignored host cancellation")
	}
	missingResponse, _ := NewGuest(bytes.NewReader(nil), &bytes.Buffer{})
	missingResponse.next = 2
	if _, err := missingResponse.Entropy("entropy", 1); err == nil {
		t.Fatal("Guest accepted a missing host response")
	}
	broken, _ := NewGuest(bytes.NewReader(nil), sdkFailingWriter{})
	broken.next = 2
	if err := broken.Fail("plugin.failed", "", "failed"); err == nil {
		t.Fatal("Guest ignored a broken output stream")
	}
}

type sdkFailingWriter struct{}

func (sdkFailingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func testInvocationFrame() *pluginprotocol.Frame {
	return &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 1,
		Payload: &pluginprotocol.Frame_Invocation{Invocation: &pluginprotocol.Invocation{
			RequestId: "request-1", InvocationId: "invocation-1", GraphId: "main", NodeId: "node-1", Attempt: 1,
			ObservedUnixMillis: 1, DeadlineUnixMillis: 2, NodeRefJson: []byte(`{}`), ImplementationLockJson: []byte(`{}`), ConfigJson: []byte(`{}`),
			Budget: &pluginprotocol.Budget{MaxFrameBytes: pluginprotocol.MaxFrameBytes, MaxOutputBytes: pluginprotocol.MaxFrameBytes, MaxHostCalls: 8, MaxStatusEvents: 8},
		}}}
}
