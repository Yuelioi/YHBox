// Package apperr projects domain errors into one locale-free Wails envelope.
// Domain packages keep their own typed errors and expose safe transport facts
// through EnvelopeProvider; the envelope is not a second domain error model.
package apperr

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/google/uuid"
)

var (
	observerMu sync.RWMutex
	observer   func(Envelope, error)
)

const (
	CategoryValidation     = "validation"
	CategoryDomain         = "domain"
	CategoryPolicy         = "policy"
	CategoryAdapter        = "adapter"
	CategoryInfrastructure = "infrastructure"

	IDUnexpected = "system.unexpected"
)

// Envelope is the only problem representation allowed across the Wails seam.
// ID and Params are locale-free. Params must not contain credentials, raw
// implementation errors, private paths, or ambient host authority.
type Envelope struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Params      any    `json:"params,omitempty"`
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
	ID     string         `json:"id"`
	Params map[string]any `json:"params,omitempty"`
}

// Error only returns the stable ID. Params remain structured and Cause belongs
// to domain errors that wrap this problem; neither becomes product text.
func (e *Error) Error() string { return e.ID }

// New 构造一个 *Error。params 可为 nil。
func New(id string, params map[string]any) *Error {
	return &Error{ID: id, Params: params}
}

func (e *Error) RPCErrorEnvelope() Envelope {
	if e == nil {
		return Envelope{ID: IDUnexpected, Category: CategoryInfrastructure}
	}
	return Envelope{
		ID: e.ID, Category: CategoryDomain, Params: cloneParams(e.Params),
	}
}

// Marshal is installed once as application.Options.MarshalError. It preserves
// typed domain errors and gives every remaining Go error the same stable JSON
// shape, so frontend transports never need to guess whether Wails returned an
// object, a validation struct, or a raw message.
func Marshal(err error) []byte {
	envelope := From(err)
	observe(envelope, err)
	encoded, marshalErr := json.Marshal(envelope)
	if marshalErr == nil {
		return encoded
	}
	return []byte(`{"id":"system.unexpected","category":"infrastructure","retryable":false}`)
}

// SetObserver installs the process composition's safe correlation sink. The
// observer receives the process-local cause, but product transports never do.
func SetObserver(next func(Envelope, error)) func() {
	observerMu.Lock()
	previous := observer
	observer = next
	observerMu.Unlock()
	return func() {
		observerMu.Lock()
		observer = previous
		observerMu.Unlock()
	}
}

func observe(envelope Envelope, err error) {
	if envelope.ID != IDUnexpected || err == nil {
		return
	}
	observerMu.RLock()
	current := observer
	observerMu.RUnlock()
	if current != nil {
		current(envelope, err)
	}
}

func From(err error) Envelope {
	if err == nil {
		return completeEnvelope(Envelope{ID: IDUnexpected, Category: CategoryInfrastructure})
	}
	var provider EnvelopeProvider
	if errors.As(err, &provider) {
		envelope := provider.RPCErrorEnvelope()
		if envelope.ID != "" && envelope.Category != "" {
			return completeEnvelope(envelope)
		}
	}
	return completeEnvelope(Envelope{
		ID:        IDUnexpected,
		Category:  CategoryInfrastructure,
		Retryable: errors.Is(err, context.DeadlineExceeded),
	})
}

func completeEnvelope(envelope Envelope) Envelope {
	if envelope.OperationID == "" {
		envelope.OperationID = uuid.NewString()
	}
	return envelope
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
