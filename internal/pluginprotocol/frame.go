package pluginprotocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	Protocol                      = "yotta.plugin/1"
	ProcessIsolationHostFeatureID = "https://schemas.yotta.dev/host-features/plugin-process-isolation/lpac-appcontainer-job/v1"
	WasmIsolationHostFeatureID    = "https://schemas.yotta.dev/host-features/plugin-wasm-isolation/lpac-appcontainer-job-wazero/v1"
	MaxFrameBytes                 = 8 << 20
	MaxConfigBytes                = 1 << 20
	MaxHostPayloadBytes           = 4 << 20
	MaxFailureTextBytes           = 2 << 10
	MaxPortValues                 = 4_096
	MaxExecOutputs                = 4_096
	MaxCounters                   = 256
	MaxOperations                 = 64
	MaxHostCalls                  = 4_096
	MaxStatusEvents               = 4_096
	MaxEntropyBytes               = 4_096
	MaxWaitMillis                 = 86_400_000
	MaxFacts                      = 256
	MaxFactValueBytes             = 2 << 10
)

var identityPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,255}$`)
var operationPattern = regexp.MustCompile(`^[a-z][a-z0-9._/-]{0,63}$`)
var codePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._/-][a-z0-9]+)+$`)

func WriteFrame(writer io.Writer, frame *Frame) error {
	if writer == nil {
		return errors.New("plugin frame writer is required")
	}
	payload, err := MarshalFrame(frame)
	if err != nil {
		return err
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if err := writeAll(writer, header[:]); err != nil {
		return fmt.Errorf("write plugin frame header: %w", err)
	}
	if err := writeAll(writer, payload); err != nil {
		return fmt.Errorf("write plugin frame payload: %w", err)
	}
	return nil
}

func ReadFrame(reader io.Reader) (*Frame, error) {
	if reader == nil {
		return nil, errors.New("plugin frame reader is required")
	}
	var header [4]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return nil, fmt.Errorf("read plugin frame header: %w", err)
	}
	size := int(binary.BigEndian.Uint32(header[:]))
	if size <= 0 || size > MaxFrameBytes {
		return nil, fmt.Errorf("plugin frame length must be within 1..%d", MaxFrameBytes)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, fmt.Errorf("read plugin frame payload: %w", err)
	}
	return UnmarshalFrame(payload)
}

// MarshalFrame returns the canonical payload without the stream length prefix.
func MarshalFrame(frame *Frame) ([]byte, error) {
	if err := ValidateFrame(frame); err != nil {
		return nil, err
	}
	payload, err := (proto.MarshalOptions{Deterministic: true}).Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("encode plugin frame: %w", err)
	}
	if len(payload) == 0 || len(payload) > MaxFrameBytes {
		return nil, fmt.Errorf("plugin frame exceeds %d bytes", MaxFrameBytes)
	}
	return payload, nil
}

// UnmarshalFrame opens a canonical payload without a stream length prefix.
func UnmarshalFrame(payload []byte) (*Frame, error) {
	if len(payload) == 0 || len(payload) > MaxFrameBytes {
		return nil, fmt.Errorf("plugin frame payload must be within 1..%d bytes", MaxFrameBytes)
	}
	frame := &Frame{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, frame); err != nil {
		return nil, fmt.Errorf("decode plugin frame: %w", err)
	}
	if err := rejectUnknown(frame.ProtoReflect()); err != nil {
		return nil, err
	}
	canonical, err := (proto.MarshalOptions{Deterministic: true}).Marshal(frame)
	if err != nil || !bytes.Equal(canonical, payload) {
		return nil, errors.New("plugin frame is not canonical deterministic Protobuf")
	}
	if err := ValidateFrame(frame); err != nil {
		return nil, err
	}
	return frame, nil
}

func ValidateFrame(frame *Frame) error {
	if frame == nil || frame.Protocol != Protocol || frame.Sequence == 0 {
		return errors.New("plugin frame protocol or sequence is invalid")
	}
	switch payload := frame.Payload.(type) {
	case *Frame_Invocation:
		return validateInvocation(payload.Invocation)
	case *Frame_HostOpenRequest:
		return validateHostOpenRequest(payload.HostOpenRequest)
	case *Frame_HostOpenResponse:
		return validateHostOpenResponse(payload.HostOpenResponse)
	case *Frame_HostInvokeRequest:
		return validateHostInvokeRequest(payload.HostInvokeRequest)
	case *Frame_HostInvokeResponse:
		return validateHostInvokeResponse(payload.HostInvokeResponse)
	case *Frame_HostDropRequest:
		return validateHostDropRequest(payload.HostDropRequest)
	case *Frame_HostDropResponse:
		return validateHostDropResponse(payload.HostDropResponse)
	case *Frame_Status:
		return validateStatus(payload.Status)
	case *Frame_Result:
		return validateResult(payload.Result)
	case *Frame_Cancel:
		if payload.Cancel == nil || !validCancelReason(payload.Cancel.Reason) {
			return errors.New("plugin cancel frame is invalid")
		}
		return nil
	case *Frame_HostEntropyRequest:
		if value := payload.HostEntropyRequest; value == nil || !validIdentity(value.RequestId) || value.ByteCount == 0 || value.ByteCount > MaxEntropyBytes {
			return errors.New("plugin host entropy request is invalid")
		}
		return nil
	case *Frame_HostEntropyResponse:
		return validateEntropyResponse(payload.HostEntropyResponse)
	case *Frame_HostWaitRequest:
		if value := payload.HostWaitRequest; value == nil || !validIdentity(value.RequestId) || value.DurationMillis == 0 || value.DurationMillis > MaxWaitMillis {
			return errors.New("plugin host wait request is invalid")
		}
		return nil
	case *Frame_HostWaitResponse:
		return validateFailureOnlyResponse(payload.HostWaitResponse.GetRequestId(), payload.HostWaitResponse.GetFailure(), payload.HostWaitResponse != nil)
	case *Frame_StateReadRequest:
		return validateStateReadRequest(payload.StateReadRequest)
	case *Frame_StateReadResponse:
		return validateStateResponse(payload.StateReadResponse.GetRequestId(), payload.StateReadResponse.GetValueEnvelope(), payload.StateReadResponse.GetRevision(), payload.StateReadResponse.GetFailure(), payload.StateReadResponse != nil)
	case *Frame_StateWriteRequest:
		return validateStateWriteRequest(payload.StateWriteRequest)
	case *Frame_StateWriteResponse:
		return validateStateResponse(payload.StateWriteResponse.GetRequestId(), payload.StateWriteResponse.GetValueEnvelope(), payload.StateWriteResponse.GetRevision(), payload.StateWriteResponse.GetFailure(), payload.StateWriteResponse != nil)
	case *Frame_Action:
		return validateAction(payload.Action)
	default:
		return errors.New("plugin frame payload is missing or unsupported")
	}
}

func validateEntropyResponse(value *HostEntropyResponse) error {
	if value == nil || !validIdentity(value.RequestId) || (len(value.Entropy) == 0) == (value.Failure == nil) || len(value.Entropy) > MaxEntropyBytes {
		return errors.New("plugin host entropy response is invalid")
	}
	if value.Failure != nil {
		return validateFailure(value.Failure)
	}
	return nil
}

func validateFailureOnlyResponse(requestID string, failure *Failure, present bool) error {
	if !present || !validIdentity(requestID) {
		return errors.New("plugin host response is invalid")
	}
	if failure != nil {
		return validateFailure(failure)
	}
	return nil
}

func validateStateReadRequest(value *StateReadRequest) error {
	if value == nil || !validIdentity(value.RequestId) || !validIdentity(value.AccessId) {
		return errors.New("plugin state read request is invalid")
	}
	return nil
}

func validateStateWriteRequest(value *StateWriteRequest) error {
	if value == nil || !validIdentity(value.RequestId) || !validIdentity(value.AccessId) {
		return errors.New("plugin state write request is invalid")
	}
	if err := validateCanonicalJSON(value.ValueEnvelope, datatype.MaxValueEnvelopeBytes); err != nil {
		return fmt.Errorf("plugin state write value: %w", err)
	}
	return nil
}

func validateStateResponse(requestID string, envelope []byte, revision int64, failure *Failure, present bool) error {
	if !present || !validIdentity(requestID) || revision < 0 || (len(envelope) == 0) == (failure == nil) {
		return errors.New("plugin state response is invalid")
	}
	if failure != nil {
		return validateFailure(failure)
	}
	return validateCanonicalJSON(envelope, datatype.MaxValueEnvelopeBytes)
}

func validateAction(value *ActionEvent) error {
	if value == nil || !validIdentity(value.EffectId) || !operationPattern.MatchString(value.Action) ||
		(value.Outcome != "succeeded" && value.Outcome != "failed" && value.Outcome != "cancelled") ||
		!codePattern.MatchString(value.SummaryCode) || len(value.Counters) > MaxCounters || len(value.Facts) > MaxFacts {
		return errors.New("plugin action event is invalid")
	}
	if value.Outcome == "failed" {
		if !codePattern.MatchString(value.ErrorCode) {
			return errors.New("failed plugin action requires an error code")
		}
	} else if value.ErrorCode != "" {
		return errors.New("non-failed plugin action contains an error code")
	}
	previous := ""
	for _, counter := range value.Counters {
		if counter == nil || !validIdentity(counter.Key) || counter.Key <= previous {
			return errors.New("plugin action counters are invalid or not strictly ordered")
		}
		previous = counter.Key
	}
	previous = ""
	for _, fact := range value.Facts {
		if fact == nil || !validIdentity(fact.Key) || fact.Key <= previous || len(fact.Value) > MaxFactValueBytes || !utf8.ValidString(fact.Value) {
			return errors.New("plugin action facts are invalid or not strictly ordered")
		}
		previous = fact.Key
	}
	return nil
}

func validateInvocation(value *Invocation) error {
	if value == nil || !validIdentity(value.RequestId) || !validIdentity(value.InvocationId) ||
		!validIdentity(value.GraphId) || !validIdentity(value.NodeId) || value.Attempt == 0 ||
		value.DeadlineUnixMillis <= value.ObservedUnixMillis || value.Budget == nil {
		return errors.New("plugin invocation identity or deadline is invalid")
	}
	for name, raw := range map[string][]byte{
		"nodeRef": value.NodeRefJson, "implementationLock": value.ImplementationLockJson, "config": value.ConfigJson,
	} {
		if err := validateCanonicalJSON(raw, MaxConfigBytes); err != nil {
			return fmt.Errorf("plugin invocation %s: %w", name, err)
		}
	}
	if value.Budget.MaxFrameBytes == 0 || value.Budget.MaxFrameBytes > MaxFrameBytes ||
		value.Budget.MaxOutputBytes == 0 || value.Budget.MaxOutputBytes > MaxFrameBytes ||
		value.Budget.MaxHostCalls == 0 || value.Budget.MaxHostCalls > MaxHostCalls ||
		value.Budget.MaxStatusEvents > MaxStatusEvents {
		return errors.New("plugin invocation budget is invalid")
	}
	if err := validatePortValues(value.Inputs, MaxFrameBytes); err != nil {
		return fmt.Errorf("plugin invocation inputs: %w", err)
	}
	if trigger := value.Trigger; trigger != nil {
		if !validIdentity(trigger.Channel) || !validIdentity(trigger.InputPort) ||
			!validOptionalIdentity(trigger.SourceNodeId) || !validOptionalIdentity(trigger.SourcePortId) {
			return errors.New("plugin invocation trigger is invalid")
		}
		if len(trigger.FailureJson) != 0 {
			if err := validateCanonicalJSON(trigger.FailureJson, MaxConfigBytes); err != nil {
				return fmt.Errorf("plugin invocation trigger failure: %w", err)
			}
		}
	}
	return nil
}

func validatePortValues(values []*PortValue, maxTotal int) error {
	if len(values) > MaxPortValues {
		return errors.New("port value count exceeds its budget")
	}
	previous, total := "", 0
	for _, value := range values {
		if value == nil || !validIdentity(value.PortId) || value.PortId <= previous || len(value.ValueEnvelope) == 0 || len(value.ValueEnvelope) > datatype.MaxValueEnvelopeBytes {
			return errors.New("port values are invalid or not strictly ordered")
		}
		total += len(value.ValueEnvelope)
		if total > maxTotal {
			return errors.New("port values exceed their aggregate byte budget")
		}
		if err := validateCanonicalJSON(value.ValueEnvelope, datatype.MaxValueEnvelopeBytes); err != nil {
			return err
		}
		previous = value.PortId
	}
	return nil
}

func validateHostOpenRequest(value *HostOpenRequest) error {
	if value == nil || !validIdentity(value.RequestId) || !validIdentity(value.RequirementId) || len(value.Operations) == 0 || len(value.Operations) > MaxOperations {
		return errors.New("plugin host open request is invalid")
	}
	if !strictOperations(value.Operations) {
		return errors.New("plugin host open operations are invalid or not strictly ordered")
	}
	return validateCanonicalJSON(value.ConfigJson, MaxConfigBytes)
}

func validateHostOpenResponse(value *HostOpenResponse) error {
	if value == nil || !validIdentity(value.RequestId) || (len(value.HandleJson) == 0) == (value.Failure == nil) {
		return errors.New("plugin host open response is invalid")
	}
	if value.Failure != nil {
		return validateFailure(value.Failure)
	}
	return validateCanonicalJSON(value.HandleJson, MaxConfigBytes)
}

func validateHostInvokeRequest(value *HostInvokeRequest) error {
	if value == nil || !validIdentity(value.RequestId) || !validIdentity(value.RequirementId) || !operationPattern.MatchString(value.Operation) || len(value.Payload) > MaxHostPayloadBytes {
		return errors.New("plugin host invoke request is invalid")
	}
	return validateCanonicalJSON(value.HandleJson, MaxConfigBytes)
}

func validateHostInvokeResponse(value *HostInvokeResponse) error {
	if value == nil || !validIdentity(value.RequestId) || (value.Failure != nil && len(value.Payload) != 0) || len(value.Payload) > MaxHostPayloadBytes {
		return errors.New("plugin host invoke response is invalid")
	}
	if value.Failure != nil {
		return validateFailure(value.Failure)
	}
	return nil
}

func validateHostDropRequest(value *HostDropRequest) error {
	if value == nil || !validIdentity(value.RequestId) || !validIdentity(value.RequirementId) {
		return errors.New("plugin host drop request is invalid")
	}
	return validateCanonicalJSON(value.HandleJson, MaxConfigBytes)
}

func validateHostDropResponse(value *HostDropResponse) error {
	if value == nil || !validIdentity(value.RequestId) {
		return errors.New("plugin host drop response is invalid")
	}
	if value.Failure != nil {
		return validateFailure(value.Failure)
	}
	return nil
}

func validateStatus(value *StatusEvent) error {
	if value == nil || !codePattern.MatchString(value.Code) || len(value.Counters) > MaxCounters {
		return errors.New("plugin status event is invalid")
	}
	previous := ""
	for _, counter := range value.Counters {
		if counter == nil || !validIdentity(counter.Key) || counter.Key <= previous {
			return errors.New("plugin status counters are invalid or not strictly ordered")
		}
		previous = counter.Key
	}
	return nil
}

func validateResult(value *Result) error {
	if value == nil || !validTerminationStrength(value.TerminationStrength) || len(value.ExecOutputs) > MaxExecOutputs {
		return errors.New("plugin result is invalid")
	}
	if !slices.IsSorted(value.ExecOutputs) || hasDuplicates(value.ExecOutputs) {
		return errors.New("plugin exec outputs are not strictly ordered")
	}
	switch value.Outcome {
	case Outcome_OUTCOME_SUCCEEDED:
		if value.Failure != nil {
			return errors.New("successful plugin result contains a failure")
		}
		return validatePortValues(value.Outputs, MaxFrameBytes)
	case Outcome_OUTCOME_FAILED:
		if len(value.Outputs) != 0 || len(value.ExecOutputs) != 0 || value.Failure == nil {
			return errors.New("failed plugin result contains successful outputs")
		}
		return validateFailure(value.Failure)
	default:
		return errors.New("plugin result outcome is invalid")
	}
}

func validateFailure(value *Failure) error {
	if value == nil || !codePattern.MatchString(value.Code) || !validOptionalIdentity(value.Output) || value.Message == "" ||
		len(value.Message) > MaxFailureTextBytes || !utf8.ValidString(value.Message) {
		return errors.New("plugin failure is invalid")
	}
	return nil
}

func validateCanonicalJSON(raw []byte, maxBytes int) error {
	if len(raw) == 0 || len(raw) > maxBytes {
		return errors.New("canonical JSON is empty or exceeds its byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, 96, 262_144, maxBytes); err != nil {
		return err
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return errors.New("value is not canonical RFC 8785 JSON")
	}
	return nil
}

func rejectUnknown(message protoreflect.Message) error {
	if len(message.GetUnknown()) != 0 {
		return errors.New("plugin frame contains unknown Protobuf fields")
	}
	var nestedErr error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.IsList() && field.Kind() == protoreflect.MessageKind {
			list := value.List()
			for index := 0; index < list.Len(); index++ {
				if err := rejectUnknown(list.Get(index).Message()); err != nil {
					nestedErr = err
					return false
				}
			}
		} else if field.Kind() == protoreflect.MessageKind {
			nestedErr = rejectUnknown(value.Message())
		}
		return nestedErr == nil
	})
	return nestedErr
}

func strictOperations(values []string) bool {
	for index, value := range values {
		if !operationPattern.MatchString(value) || index > 0 && value <= values[index-1] {
			return false
		}
	}
	return true
}

func hasDuplicates(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return true
		}
	}
	return false
}

func validIdentity(value string) bool         { return identityPattern.MatchString(value) }
func validOptionalIdentity(value string) bool { return value == "" || validIdentity(value) }
func validCancelReason(value string) bool {
	return value == "user_cancelled" || value == "deadline_exceeded" || value == "run_shutdown" || value == "budget_exceeded"
}
func validTerminationStrength(value string) bool {
	return value == "cooperative" || value == "engine_interrupt" || value == "job_terminate" || value == "process_crash"
}

func writeAll(writer io.Writer, value []byte) error {
	for len(value) != 0 {
		written, err := writer.Write(value)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		value = value[written:]
	}
	return nil
}
