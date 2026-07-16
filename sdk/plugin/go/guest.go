package pluginsdk

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/pluginprotocol"
)

// Guest owns one process-plugin invocation stream. Its methods serialize all
// traffic, assign the next sequence, and reject mismatched host responses.
type Guest struct {
	input     io.Reader
	output    io.Writer
	mu        sync.Mutex
	next      uint64
	completed bool
}

func NewGuest(input io.Reader, output io.Writer) (*Guest, error) {
	if input == nil || output == nil {
		return nil, errors.New("plugin guest requires protocol streams")
	}
	return &Guest{input: input, output: output, next: 1}, nil
}

func (guest *Guest) ReceiveInvocation() (*Invocation, error) {
	guest.mu.Lock()
	defer guest.mu.Unlock()
	if guest.next != 1 || guest.completed {
		return nil, errors.New("plugin invocation was already received")
	}
	frame, err := pluginprotocol.ReadFrame(guest.input)
	if err != nil {
		return nil, err
	}
	if frame.Sequence != 1 || frame.GetInvocation() == nil {
		return nil, errors.New("plugin guest expected invocation sequence 1")
	}
	guest.next = 2
	return frame.GetInvocation(), nil
}

func (guest *Guest) Open(requestID, requirementID string, operations []string, configJSON []byte) (*HostOpenResponse, error) {
	request := &pluginprotocol.HostOpenRequest{
		RequestId: requestID, RequirementId: requirementID, Operations: append([]string(nil), operations...), ConfigJson: append([]byte(nil), configJSON...),
	}
	slices.Sort(request.Operations)
	response, err := guest.exchange(&pluginprotocol.Frame{Payload: &pluginprotocol.Frame_HostOpenRequest{HostOpenRequest: request}})
	if err != nil {
		return nil, err
	}
	opened := response.GetHostOpenResponse()
	if opened == nil || opened.RequestId != requestID {
		return nil, errors.New("plugin host returned a mismatched open response")
	}
	return opened, nil
}

func (guest *Guest) Invoke(requestID, requirementID string, handleJSON []byte, operation string, payload []byte) (*HostInvokeResponse, error) {
	response, err := guest.exchange(&pluginprotocol.Frame{Payload: &pluginprotocol.Frame_HostInvokeRequest{HostInvokeRequest: &pluginprotocol.HostInvokeRequest{
		RequestId: requestID, RequirementId: requirementID, HandleJson: append([]byte(nil), handleJSON...), Operation: operation, Payload: append([]byte(nil), payload...),
	}}})
	if err != nil {
		return nil, err
	}
	opened := response.GetHostInvokeResponse()
	if opened == nil || opened.RequestId != requestID {
		return nil, errors.New("plugin host returned a mismatched invoke response")
	}
	return opened, nil
}

func (guest *Guest) Drop(requestID, requirementID string, handleJSON []byte) (*HostDropResponse, error) {
	response, err := guest.exchange(&pluginprotocol.Frame{Payload: &pluginprotocol.Frame_HostDropRequest{HostDropRequest: &pluginprotocol.HostDropRequest{
		RequestId: requestID, RequirementId: requirementID, HandleJson: append([]byte(nil), handleJSON...),
	}}})
	if err != nil {
		return nil, err
	}
	opened := response.GetHostDropResponse()
	if opened == nil || opened.RequestId != requestID {
		return nil, errors.New("plugin host returned a mismatched drop response")
	}
	return opened, nil
}

func (guest *Guest) Entropy(requestID string, byteCount uint32) (*HostEntropyResponse, error) {
	response, err := guest.exchange(&pluginprotocol.Frame{Payload: &pluginprotocol.Frame_HostEntropyRequest{HostEntropyRequest: &pluginprotocol.HostEntropyRequest{
		RequestId: requestID, ByteCount: byteCount,
	}}})
	if err != nil {
		return nil, err
	}
	opened := response.GetHostEntropyResponse()
	if opened == nil || opened.RequestId != requestID {
		return nil, errors.New("plugin host returned a mismatched entropy response")
	}
	return opened, nil
}

func (guest *Guest) Wait(requestID string, duration time.Duration) (*HostWaitResponse, error) {
	response, err := guest.exchange(&pluginprotocol.Frame{Payload: &pluginprotocol.Frame_HostWaitRequest{HostWaitRequest: &pluginprotocol.HostWaitRequest{
		RequestId: requestID, DurationMillis: uint64(duration / time.Millisecond),
	}}})
	if err != nil {
		return nil, err
	}
	opened := response.GetHostWaitResponse()
	if opened == nil || opened.RequestId != requestID {
		return nil, errors.New("plugin host returned a mismatched wait response")
	}
	return opened, nil
}

func (guest *Guest) ReadState(requestID, accessID string) (*StateReadResponse, error) {
	response, err := guest.exchange(&pluginprotocol.Frame{Payload: &pluginprotocol.Frame_StateReadRequest{StateReadRequest: &pluginprotocol.StateReadRequest{
		RequestId: requestID, AccessId: accessID,
	}}})
	if err != nil {
		return nil, err
	}
	opened := response.GetStateReadResponse()
	if opened == nil || opened.RequestId != requestID {
		return nil, errors.New("plugin host returned a mismatched state-read response")
	}
	return opened, nil
}

func (guest *Guest) WriteState(requestID, accessID string, envelope []byte) (*StateWriteResponse, error) {
	response, err := guest.exchange(&pluginprotocol.Frame{Payload: &pluginprotocol.Frame_StateWriteRequest{StateWriteRequest: &pluginprotocol.StateWriteRequest{
		RequestId: requestID, AccessId: accessID, ValueEnvelope: append([]byte(nil), envelope...),
	}}})
	if err != nil {
		return nil, err
	}
	opened := response.GetStateWriteResponse()
	if opened == nil || opened.RequestId != requestID {
		return nil, errors.New("plugin host returned a mismatched state-write response")
	}
	return opened, nil
}

func (guest *Guest) Status(code string, counters map[string]int64) error {
	items := make([]*pluginprotocol.Counter, 0, len(counters))
	for key, value := range counters {
		items = append(items, &pluginprotocol.Counter{Key: key, Value: value})
	}
	slices.SortFunc(items, func(left, right *pluginprotocol.Counter) int { return strings.Compare(left.Key, right.Key) })
	return guest.emit(&pluginprotocol.Frame{Payload: &pluginprotocol.Frame_Status{Status: &pluginprotocol.StatusEvent{Code: code, Counters: items}}})
}

func (guest *Guest) Record(action Action) error {
	action.Counters = append([]*pluginprotocol.Counter(nil), action.Counters...)
	action.Facts = append([]*pluginprotocol.Fact(nil), action.Facts...)
	slices.SortFunc(action.Counters, func(left, right *pluginprotocol.Counter) int { return strings.Compare(left.Key, right.Key) })
	slices.SortFunc(action.Facts, func(left, right *pluginprotocol.Fact) int { return strings.Compare(left.Key, right.Key) })
	return guest.emit(&pluginprotocol.Frame{Payload: &pluginprotocol.Frame_Action{Action: &action}})
}

func (guest *Guest) Succeed(outputs map[string][]byte, execOutputs []string) error {
	values := make([]*pluginprotocol.PortValue, 0, len(outputs))
	for portID, envelope := range outputs {
		values = append(values, &pluginprotocol.PortValue{PortId: portID, ValueEnvelope: append([]byte(nil), envelope...)})
	}
	slices.SortFunc(values, func(left, right *pluginprotocol.PortValue) int { return strings.Compare(left.PortId, right.PortId) })
	routes := append([]string(nil), execOutputs...)
	slices.Sort(routes)
	return guest.complete(&pluginprotocol.Result{
		Outcome: pluginprotocol.Outcome_OUTCOME_SUCCEEDED, Outputs: values, ExecOutputs: routes, TerminationStrength: "cooperative",
	})
}

func (guest *Guest) Fail(code, output, message string) error {
	return guest.complete(&pluginprotocol.Result{
		Outcome: pluginprotocol.Outcome_OUTCOME_FAILED, Failure: &pluginprotocol.Failure{Code: code, Output: output, Message: message},
		TerminationStrength: "cooperative",
	})
}

func (guest *Guest) exchange(frame *pluginprotocol.Frame) (*pluginprotocol.Frame, error) {
	guest.mu.Lock()
	defer guest.mu.Unlock()
	if err := guest.write(frame); err != nil {
		return nil, err
	}
	response, err := pluginprotocol.ReadFrame(guest.input)
	if err != nil {
		return nil, err
	}
	if response.Sequence != guest.next {
		return nil, fmt.Errorf("plugin host response sequence %d, want %d", response.Sequence, guest.next)
	}
	guest.next++
	if cancel := response.GetCancel(); cancel != nil {
		return nil, fmt.Errorf("plugin invocation cancelled: %s", cancel.Reason)
	}
	return response, nil
}

func (guest *Guest) emit(frame *pluginprotocol.Frame) error {
	guest.mu.Lock()
	defer guest.mu.Unlock()
	return guest.write(frame)
}

func (guest *Guest) complete(result *pluginprotocol.Result) error {
	guest.mu.Lock()
	defer guest.mu.Unlock()
	if guest.completed {
		return errors.New("plugin result was already emitted")
	}
	if err := guest.write(&pluginprotocol.Frame{Payload: &pluginprotocol.Frame_Result{Result: result}}); err != nil {
		return err
	}
	guest.completed = true
	return nil
}

func (guest *Guest) write(frame *pluginprotocol.Frame) error {
	if guest.completed || guest.next < 2 {
		return errors.New("plugin guest is not in an active invocation")
	}
	frame.Protocol = pluginprotocol.Protocol
	frame.Sequence = guest.next
	if err := pluginprotocol.WriteFrame(guest.output, frame); err != nil {
		return err
	}
	guest.next++
	return nil
}
