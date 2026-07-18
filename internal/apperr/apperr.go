// Package apperr projects domain errors into one locale-free Wails envelope.
// Domain packages keep their own typed errors and expose safe transport facts
// through EnvelopeProvider; the envelope is not a second domain error model.
package apperr

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

const (
	CategoryValidation     = "validation"
	CategoryDomain         = "domain"
	CategoryPolicy         = "policy"
	CategoryAdapter        = "adapter"
	CategoryInfrastructure = "infrastructure"

	CodeUnclassified = "rpc.unclassified"
)

// Envelope is the only error representation allowed across the Wails seam.
// Message is locale-free diagnostic text; the frontend localises Code when a
// matching message exists. Details must not contain credentials or ambient
// host authority.
type Envelope struct {
	Code        string `json:"code"`
	Category    string `json:"category"`
	Message     string `json:"message"`
	Details     any    `json:"details,omitempty"`
	OperationID string `json:"operationId,omitempty"`
	RunID       string `json:"runId,omitempty"`
	Retryable   bool   `json:"retryable"`
}

// EnvelopeProvider lets a domain error preserve its code and safe details
// without teaching the Wails composition root about every domain package.
type EnvelopeProvider interface {
	RPCErrorEnvelope() Envelope
}

// Error 携带一个 i18n code 与可选插值参数。
type Error struct {
	Code   string         `json:"code"`
	Params map[string]any `json:"params,omitempty"`
}

// Error 只返 Code (供后端 log/debug)。要 params 走结构化日志字段, 不进 Error()。
func (e *Error) Error() string { return e.Code }

// New 构造一个 *Error。params 可为 nil。
func New(code string, params map[string]any) *Error {
	return &Error{Code: code, Params: params}
}

func (e *Error) RPCErrorEnvelope() Envelope {
	if e == nil {
		return Envelope{Code: CodeUnclassified, Category: CategoryInfrastructure, Message: "unknown error"}
	}
	return Envelope{
		Code: e.Code, Category: CategoryDomain, Message: e.Code, Details: cloneParams(e.Params),
	}
}

// Marshal is installed once as application.Options.MarshalError. It preserves
// typed domain errors and gives every remaining Go error the same stable JSON
// shape, so frontend transports never need to guess whether Wails returned an
// object, a validation struct, or a raw message.
func Marshal(err error) []byte {
	envelope := From(err)
	encoded, marshalErr := json.Marshal(envelope)
	if marshalErr == nil {
		return encoded
	}
	return []byte(`{"code":"rpc.unclassified","category":"infrastructure","message":"error envelope serialization failed","retryable":false}`)
}

func From(err error) Envelope {
	if err == nil {
		return completeEnvelope(Envelope{Code: CodeUnclassified, Category: CategoryInfrastructure, Message: "unknown error"})
	}
	var provider EnvelopeProvider
	if errors.As(err, &provider) {
		envelope := provider.RPCErrorEnvelope()
		if envelope.Code != "" && envelope.Category != "" && envelope.Message != "" {
			return completeEnvelope(envelope)
		}
	}
	return completeEnvelope(Envelope{
		Code: CodeUnclassified, Category: CategoryInfrastructure, Message: err.Error(),
		Details:   marshalConcreteDetails(err),
		Retryable: errors.Is(err, context.DeadlineExceeded),
	})
}

func completeEnvelope(envelope Envelope) Envelope {
	if envelope.OperationID == "" {
		envelope.OperationID = uuid.NewString()
	}
	return envelope
}

func marshalConcreteDetails(err error) any {
	encoded, marshalErr := json.Marshal(err)
	if marshalErr != nil || string(encoded) == "{}" || string(encoded) == "null" {
		return nil
	}
	var details any
	if json.Unmarshal(encoded, &details) != nil {
		return nil
	}
	return details
}

func cloneParams(params map[string]any) map[string]any {
	if params == nil {
		return nil
	}
	result := make(map[string]any, len(params))
	for key, value := range params {
		result[key] = value
	}
	return result
}

// 首批 code 常量 (与 FE error.* i18n 一一对应)。
const (
	CodeWailsNotReady                = "WAILS_NOT_READY"
	CodeAutomationTargetSlotRequired = "AUTOMATION_TARGET_SLOT_REQUIRED"
	CodeRecordingTargetUnavailable   = "RECORDING_TARGET_UNAVAILABLE"
	CodeRecordingModeRequired        = "RECORDING_MODE_REQUIRED"
	CodeRecordingCalibrationRequired = "RECORDING_CALIBRATION_REQUIRED"
	CodeRecordingSessionBusy         = "RECORDING_SESSION_BUSY"
	CodeAssetQueryInvalid            = "ASSET_QUERY_INVALID"
)
