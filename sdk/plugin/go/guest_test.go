package pluginsdk

import (
	"bytes"
	"net"
	"testing"

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

func testInvocationFrame() *pluginprotocol.Frame {
	return &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 1,
		Payload: &pluginprotocol.Frame_Invocation{Invocation: &pluginprotocol.Invocation{
			RequestId: "request-1", InvocationId: "invocation-1", GraphId: "main", NodeId: "node-1", Attempt: 1,
			ObservedUnixMillis: 1, DeadlineUnixMillis: 2, NodeRefJson: []byte(`{}`), ImplementationLockJson: []byte(`{}`), ConfigJson: []byte(`{}`),
			Budget: &pluginprotocol.Budget{MaxFrameBytes: pluginprotocol.MaxFrameBytes, MaxOutputBytes: pluginprotocol.MaxFrameBytes, MaxHostCalls: 8, MaxStatusEvents: 8},
		}}}
}
