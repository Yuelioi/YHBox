package pluginhost

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/pluginprotocol"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func TestProcessSessionMediatesHostCallsStatusActionsAndResult(t *testing.T) {
	hostSide, guestSide := net.Pipe()
	defer hostSide.Close()
	defer guestSide.Close()
	var statusCalls, actionCalls int32
	session := processSession{
		invocation: compiler.Invocation{
			ReadEntropy: func(buffer []byte) error {
				for index := range buffer {
					buffer[index] = byte(index + 1)
				}
				return nil
			},
			EmitStatus: func(_ context.Context, code string, counters map[string]int64) error {
				if code != "plugin.progress" || counters["items"] != 2 {
					t.Fatalf("unexpected status %q %#v", code, counters)
				}
				atomic.AddInt32(&statusCalls, 1)
				return nil
			},
			RecordAction: func(_ context.Context, action compiler.AdapterAction) error {
				if action.EffectID != "write" || action.Action != "plugin.write" || action.SummaryCode != "plugin.write_completed" {
					t.Fatalf("unexpected action %#v", action)
				}
				atomic.AddInt32(&actionCalls, 1)
				return nil
			},
		},
		reader: hostSide, writer: hostSide, nextSequence: 2, maxHostCalls: 4, maxStatusEvents: 4,
	}
	guestError := make(chan error, 1)
	go func() {
		entropy := &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 2,
			Payload: &pluginprotocol.Frame_HostEntropyRequest{HostEntropyRequest: &pluginprotocol.HostEntropyRequest{RequestId: "entropy-1", ByteCount: 4}}}
		if err := pluginprotocol.WriteFrame(guestSide, entropy); err != nil {
			guestError <- err
			return
		}
		response, err := pluginprotocol.ReadFrame(guestSide)
		if err != nil {
			guestError <- err
			return
		}
		if response.Sequence != 3 || string(response.GetHostEntropyResponse().Entropy) != "\x01\x02\x03\x04" {
			guestError <- errors.New("unexpected entropy response")
			return
		}
		frames := []*pluginprotocol.Frame{
			{Protocol: pluginprotocol.Protocol, Sequence: 4, Payload: &pluginprotocol.Frame_Status{Status: &pluginprotocol.StatusEvent{
				Code: "plugin.progress", Counters: []*pluginprotocol.Counter{{Key: "items", Value: 2}},
			}}},
			{Protocol: pluginprotocol.Protocol, Sequence: 5, Payload: &pluginprotocol.Frame_Action{Action: &pluginprotocol.ActionEvent{
				EffectId: "write", Action: "plugin.write", Outcome: "succeeded", SummaryCode: "plugin.write_completed",
			}}},
			{Protocol: pluginprotocol.Protocol, Sequence: 6, Payload: &pluginprotocol.Frame_Result{Result: &pluginprotocol.Result{
				Outcome: pluginprotocol.Outcome_OUTCOME_SUCCEEDED, TerminationStrength: "cooperative",
			}}},
		}
		for _, frame := range frames {
			if err := pluginprotocol.WriteFrame(guestSide, frame); err != nil {
				guestError <- err
				return
			}
		}
		guestError <- nil
	}()
	result, err := session.serve(context.Background(), &testProcessControl{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != pluginprotocol.Outcome_OUTCOME_SUCCEEDED || atomic.LoadInt32(&statusCalls) != 1 || atomic.LoadInt32(&actionCalls) != 1 {
		t.Fatalf("result/calls = %#v/%d/%d", result, statusCalls, actionCalls)
	}
	if err := <-guestError; err != nil {
		t.Fatal(err)
	}
}

func TestProcessSessionChecksBudgetBeforeExecutingHostCall(t *testing.T) {
	hostSide, guestSide := net.Pipe()
	defer hostSide.Close()
	defer guestSide.Close()
	var entropyCalls int32
	session := processSession{
		invocation: compiler.Invocation{ReadEntropy: func([]byte) error {
			atomic.AddInt32(&entropyCalls, 1)
			return nil
		}},
		reader: hostSide, writer: hostSide, nextSequence: 2, maxHostCalls: 0, maxStatusEvents: 1,
	}
	go func() {
		_ = pluginprotocol.WriteFrame(guestSide, &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 2,
			Payload: &pluginprotocol.Frame_HostEntropyRequest{HostEntropyRequest: &pluginprotocol.HostEntropyRequest{RequestId: "entropy-1", ByteCount: 1}}})
	}()
	_, err := session.serve(context.Background(), &testProcessControl{})
	if err == nil || !strings.Contains(err.Error(), "host-call budget") {
		t.Fatalf("serve error = %v", err)
	}
	if atomic.LoadInt32(&entropyCalls) != 0 {
		t.Fatal("over-budget host call executed before rejection")
	}
}

func TestProcessSessionReturnsCapabilityDenialWithoutGrantingAuthority(t *testing.T) {
	hostSide, guestSide := net.Pipe()
	defer hostSide.Close()
	defer guestSide.Close()
	session := processSession{
		invocation: compiler.Invocation{Sessions: map[string]*run.Session{}},
		reader:     hostSide, writer: hostSide, nextSequence: 2, maxHostCalls: 1, maxStatusEvents: 1,
	}
	guestError := make(chan error, 1)
	go func() {
		request := &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 2,
			Payload: &pluginprotocol.Frame_HostOpenRequest{HostOpenRequest: &pluginprotocol.HostOpenRequest{
				RequestId: "open-1", RequirementId: "filesystem", Operations: []string{"read"}, ConfigJson: []byte(`{}`),
			}}}
		if err := pluginprotocol.WriteFrame(guestSide, request); err != nil {
			guestError <- err
			return
		}
		response, err := pluginprotocol.ReadFrame(guestSide)
		if err != nil {
			guestError <- err
			return
		}
		if response.GetHostOpenResponse().GetFailure() == nil || len(response.GetHostOpenResponse().GetHandleJson()) != 0 {
			guestError <- errors.New("capability denial returned authority")
			return
		}
		guestError <- pluginprotocol.WriteFrame(guestSide, &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 4,
			Payload: &pluginprotocol.Frame_Result{Result: &pluginprotocol.Result{Outcome: pluginprotocol.Outcome_OUTCOME_SUCCEEDED, TerminationStrength: "cooperative"}}})
	}()
	if _, err := session.serve(context.Background(), &testProcessControl{}); err != nil {
		t.Fatal(err)
	}
	if err := <-guestError; err != nil {
		t.Fatal(err)
	}
}

func TestProcessSessionCancelsAndTerminatesUnresponsiveGuest(t *testing.T) {
	hostSide, guestSide := net.Pipe()
	defer hostSide.Close()
	defer guestSide.Close()
	control := &testProcessControl{}
	session := processSession{reader: hostSide, writer: hostSide, nextSequence: 2, maxHostCalls: 1, maxStatusEvents: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := session.serve(ctx, control)
	if !errors.Is(err, context.DeadlineExceeded) || atomic.LoadInt32(&control.terminated) != 1 {
		t.Fatalf("serve error/terminated = %v/%d", err, control.terminated)
	}
}

func TestSharedResultBoundaryRejectsSchemaViolationAndOutputOversize(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	host := executionHost{catalog: builtins.Catalog, options: ProcessHostOptions{MaxOutputBytes: pluginprotocol.MaxFrameBytes}}
	invalid := &pluginprotocol.Result{
		Outcome: pluginprotocol.Outcome_OUTCOME_SUCCEEDED, TerminationStrength: "cooperative",
		Outputs: []*pluginprotocol.PortValue{{PortId: "result", ValueEnvelope: []byte(`{}`)}},
	}
	if _, err := host.openResult(invalid); err == nil {
		t.Fatal("result boundary accepted a schema-invalid Value Envelope")
	}
	host.options.MaxOutputBytes = 1
	if _, err := host.openResult(invalid); err == nil || !strings.Contains(err.Error(), "output byte budget") {
		t.Fatalf("oversize result error = %v", err)
	}
}

func TestSharedSessionContainsGuestCrashToCurrentInvocation(t *testing.T) {
	session := processSession{
		invocation: compiler.Invocation{}, reader: bytes.NewReader(nil), writer: io.Discard,
		nextSequence: 2, maxHostCalls: 1, maxStatusEvents: 1,
	}
	if _, err := session.serve(context.Background(), &testProcessControl{}); err == nil || !strings.Contains(err.Error(), "read process plugin frame") {
		t.Fatalf("guest crash error = %v", err)
	}
}

type testProcessControl struct{ terminated int32 }

func (process *testProcessControl) Terminate() error {
	atomic.AddInt32(&process.terminated, 1)
	return nil
}
