package installed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/resource"
	pkginput "github.com/yottaapp/yotta/pkg/input"
)

const (
	ProviderABI = "https://schemas.yotta.dev/provider-abi/resource/v1"
	TargetKind  = target.KindWin32Window
	KindInput   = "automation/input-session"

	OperationClick        = "click"
	OperationMove         = "move"
	OperationScroll       = "scroll"
	OperationDrag         = "drag"
	OperationMoveRelative = "move-relative"
	OperationPressKeys    = "press-keys"
	OperationTypeText     = "type-text"

	CodeInvalidRequest    = "automation.invalid_request"
	CodeIdentityChanged   = "automation.identity_changed"
	CodeTargetNotFound    = "automation.target_not_found"
	CodeTargetAmbiguous   = "automation.target_ambiguous"
	CodeInputFailed       = "automation.input_failed"
	CodeUnsupportedHost   = "automation.unsupported_host"
	CodeContractViolation = "automation.contract_violation"

	providerImplementation = "exact-installed-window-input/v1"
	MaxInputDurationMs     = int64(60_000)
)

var operations = []string{
	OperationClick, OperationDrag, OperationMove, OperationMoveRelative,
	OperationPressKeys, OperationScroll, OperationTypeText,
}

func Operations() []string { return append([]string(nil), operations...) }

type Failure struct {
	Code  string
	Cause error
}

func (e *Failure) Error() string {
	if e == nil {
		return "automation failure"
	}
	if e.Cause == nil {
		return e.Code
	}
	return e.Code + ": " + e.Cause.Error()
}
func (e *Failure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type CapabilityScope struct {
	Operation string `json:"operation"`
}
type Point struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Unit string  `json:"unit"`
}
type ClickRequest struct {
	Point                Point  `json:"point"`
	Button               string `json:"button"`
	DurationMilliseconds int64  `json:"durationMilliseconds"`
}
type MoveRequest struct {
	Point Point `json:"point"`
}
type ScrollRequest struct {
	Point      Point `json:"point"`
	Notches    int64 `json:"notches"`
	Horizontal bool  `json:"horizontal"`
}
type DragRequest struct {
	From                 Point  `json:"from"`
	To                   Point  `json:"to"`
	Button               string `json:"button"`
	DurationMilliseconds int64  `json:"durationMilliseconds"`
}
type RelativeMoveRequest struct {
	DeltaX               int64 `json:"deltaX"`
	DeltaY               int64 `json:"deltaY"`
	DurationMilliseconds int64 `json:"durationMilliseconds"`
}
type PressKeysRequest struct {
	Keys                 []string `json:"keys"`
	DurationMilliseconds int64    `json:"durationMilliseconds"`
}
type TypeTextRequest struct {
	Text string `json:"text"`
}

type driver interface {
	Execute(context.Context, string, any) error
	Close() error
}

type provider struct {
	profile   Profile
	driver    driver
	closeOnce sync.Once
	closeErr  error
}
type session struct {
	mu        sync.Mutex
	operation string
	closed    bool
}

func newProvider(profile Profile) (*provider, error) {
	if !profile.Valid() {
		return nil, errors.New("automation provider requires a profile")
	}
	driver, err := newPlatformDriver(profile)
	if err != nil {
		return nil, err
	}
	return &provider{profile: profile, driver: driver}, nil
}

func (p *provider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	if request.Kind != KindInput || request.CredentialBindingID != "" || len(request.Operations) != 1 || !slices.Contains(operations, request.Operations[0]) {
		return nil, failure(CodeContractViolation, errors.New("invalid automation input session request"))
	}
	var config map[string]any
	if err := decodeExact(request.Config, &config, 1024); err != nil || len(config) != 0 {
		return nil, failure(CodeContractViolation, errors.New("automation input session config must be empty"))
	}
	var scope CapabilityScope
	if err := decodeExact(request.CapabilityScope, &scope, 1024); err != nil || scope.Operation != request.Operations[0] {
		return nil, failure(CodeContractViolation, errors.New("automation input capability scope is invalid"))
	}
	return &session{operation: scope.Operation}, nil
}

func (p *provider) Invoke(ctx context.Context, object any, operation string, payload []byte) ([]byte, error) {
	opened, ok := object.(*session)
	if !ok {
		return nil, failure(CodeContractViolation, errors.New("automation input object has the wrong type"))
	}
	opened.mu.Lock()
	defer opened.mu.Unlock()
	if opened.closed || operation != opened.operation {
		return nil, failure(CodeContractViolation, errors.New("automation input session is closed or operation is not granted"))
	}
	request, err := decodeOperationRequest(operation, payload)
	if err != nil {
		return nil, failure(CodeInvalidRequest, err)
	}
	if err := VerifyProfile(p.profile); err != nil {
		return nil, failure(CodeIdentityChanged, err)
	}
	if err := p.driver.Execute(ctx, operation, request); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		var classified *Failure
		if errors.As(err, &classified) {
			return nil, err
		}
		return nil, failure(CodeInputFailed, err)
	}
	return artifact.Marshal(struct{}{})
}

func (p *provider) Close(_ context.Context, object any) error {
	opened, ok := object.(*session)
	if !ok {
		return failure(CodeContractViolation, errors.New("automation input object has the wrong type"))
	}
	opened.mu.Lock()
	opened.closed = true
	opened.mu.Unlock()
	return nil
}

func (p *provider) CloseHost() error {
	p.closeOnce.Do(func() { p.closeErr = p.driver.Close() })
	return p.closeErr
}

func ProviderArtifactDigest(profile Profile) (artifact.Digest, error) {
	if !profile.Valid() {
		return "", errors.New("automation provider artifact requires a profile")
	}
	raw, err := artifact.Marshal(map[string]any{"implementation": providerImplementation, "profileDigest": profile.Digest(), "providerAbi": ProviderABI})
	if err != nil {
		return "", err
	}
	return artifact.Sum("yotta/automation-provider-artifact/v1", raw)
}

func OpenEffectResponse(raw []byte) error {
	var response struct{}
	if err := decodeExact(raw, &response, 32); err != nil {
		return failure(CodeContractViolation, errors.New("invalid automation input response"))
	}
	canonical, err := artifact.Marshal(response)
	if err != nil || !bytes.Equal(canonical, raw) {
		return failure(CodeContractViolation, errors.New("automation input response is not canonical"))
	}
	return nil
}

func decodeOperationRequest(operation string, raw []byte) (any, error) {
	if len(raw) == 0 || len(raw) > 64<<10 {
		return nil, errors.New("automation input request exceeds byte budget")
	}
	decode := func(target any) error { return decodeExact(raw, target, 64<<10) }
	switch operation {
	case OperationClick:
		var request ClickRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		return request, validateClick(request)
	case OperationMove:
		var request MoveRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		return request, validatePoint(request.Point)
	case OperationScroll:
		var request ScrollRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if err := validatePoint(request.Point); err != nil || request.Notches == 0 || request.Notches < -100 || request.Notches > 100 {
			return nil, errors.New("automation scroll request is invalid")
		}
		return request, nil
	case OperationDrag:
		var request DragRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if err := validatePoint(request.From); err != nil {
			return nil, err
		}
		if err := validatePoint(request.To); err != nil || !validButton(request.Button) || !validDuration(request.DurationMilliseconds) {
			return nil, errors.New("automation drag request is invalid")
		}
		return request, nil
	case OperationMoveRelative:
		var request RelativeMoveRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if request.DeltaX < -1_000_000 || request.DeltaX > 1_000_000 || request.DeltaY < -1_000_000 || request.DeltaY > 1_000_000 || !validDuration(request.DurationMilliseconds) {
			return nil, errors.New("automation relative move request is invalid")
		}
		return request, nil
	case OperationPressKeys:
		var request PressKeysRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if len(request.Keys) == 0 || len(request.Keys) > 16 || !validDuration(request.DurationMilliseconds) {
			return nil, errors.New("automation key request is invalid")
		}
		seen := map[string]struct{}{}
		for _, key := range request.Keys {
			if strings.TrimSpace(key) != key || pkginput.VK(key) == 0 {
				return nil, errors.New("automation key request is invalid")
			}
			folded := strings.ToUpper(key)
			if _, exists := seen[folded]; exists {
				return nil, errors.New("automation key request contains duplicates")
			}
			seen[folded] = struct{}{}
		}
		return request, nil
	case OperationTypeText:
		var request TypeTextRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if request.Text == "" || len(request.Text) > 4096 || !utf8.ValidString(request.Text) || bytes.IndexByte([]byte(request.Text), 0) >= 0 {
			return nil, errors.New("automation text request is invalid")
		}
		return request, nil
	default:
		return nil, errors.New("unsupported automation input operation")
	}
}

func validateClick(request ClickRequest) error {
	if err := validatePoint(request.Point); err != nil || !validButton(request.Button) || !validDuration(request.DurationMilliseconds) {
		return errors.New("automation click request is invalid")
	}
	return nil
}
func validatePoint(point Point) error {
	if point.Unit == "ratio" && point.X >= 0 && point.X <= 1 && point.Y >= 0 && point.Y <= 1 {
		return nil
	}
	if point.Unit == "px" && point.X >= 0 && point.X <= 1_000_000 && point.Y >= 0 && point.Y <= 1_000_000 {
		return nil
	}
	return errors.New("automation point is invalid")
}
func validButton(button string) bool {
	return button == "left" || button == "right" || button == "middle"
}
func validDuration(value int64) bool { return value >= 0 && value <= MaxInputDurationMs }
func decodeExact(raw []byte, target any, maximum int) error {
	if len(raw) == 0 || len(raw) > maximum {
		return errors.New("JSON document exceeds byte budget")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("JSON document contains trailing values")
	}
	return nil
}
func failure(code string, cause error) error { return &Failure{Code: code, Cause: cause} }

var _ resource.Provider = (*provider)(nil)
