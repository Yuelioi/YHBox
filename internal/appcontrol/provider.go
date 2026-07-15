package appcontrol

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

const (
	ProviderABI        = "https://schemas.yotta.dev/provider-abi/resource/v1"
	KindApplication    = "application/lifecycle"
	TargetKind         = "installed-application"
	OperationLaunch    = "launch"
	OperationTerminate = "terminate"
	providerImpl       = "exact-installed-application-lifecycle/v1"

	CodeInvalidRequest    = "application.invalid_request"
	CodeIdentityChanged   = "application.identity_changed"
	CodeLaunchFailed      = "application.launch_failed"
	CodeTerminateFailed   = "application.terminate_failed"
	CodeUnsupportedHost   = "application.unsupported_host"
	CodeContractViolation = "application.contract_violation"
)

type Failure struct {
	Code  string
	Cause error
}

func (e *Failure) Error() string {
	if e == nil {
		return "application control failure"
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
type LaunchResponse struct {
	ProcessID uint32 `json:"processId"`
}
type TerminateResponse struct {
	TerminatedCount int `json:"terminatedCount"`
}

type platformController interface {
	Launch(context.Context, Profile) (uint32, error)
	Terminate(context.Context, Profile) (int, error)
}

type provider struct {
	profile  Profile
	platform platformController
}
type session struct {
	mu        sync.Mutex
	operation string
	closed    bool
}

func NewProvider(profile Profile) (resource.Provider, error) {
	if err := VerifyProfile(profile); err != nil {
		return nil, err
	}
	return &provider{profile: profile, platform: newPlatformController()}, nil
}

func (p *provider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	if request.Kind != KindApplication || request.CredentialBindingID != "" || len(request.Operations) != 1 || !validOperation(request.Operations[0]) {
		return nil, failure(CodeContractViolation, errors.New("invalid application lifecycle session request"))
	}
	var config map[string]any
	if err := decodeExact(request.Config, &config, 1024); err != nil || len(config) != 0 {
		return nil, failure(CodeContractViolation, errors.New("application lifecycle session config must be empty"))
	}
	var scope CapabilityScope
	if err := decodeExact(request.CapabilityScope, &scope, 1024); err != nil || scope.Operation != request.Operations[0] {
		return nil, failure(CodeContractViolation, errors.New("application lifecycle capability scope is invalid"))
	}
	return &session{operation: scope.Operation}, nil
}

func (p *provider) Invoke(ctx context.Context, object any, operation string, payload []byte) ([]byte, error) {
	opened, ok := object.(*session)
	if !ok {
		return nil, failure(CodeContractViolation, errors.New("application lifecycle object has the wrong type"))
	}
	opened.mu.Lock()
	defer opened.mu.Unlock()
	if opened.closed || operation != opened.operation || !validOperation(operation) {
		return nil, failure(CodeContractViolation, errors.New("application lifecycle session is closed or operation is unsupported"))
	}
	var request map[string]any
	if err := decodeExact(payload, &request, 1024); err != nil || len(request) != 0 {
		return nil, failure(CodeInvalidRequest, errors.New("application lifecycle request must be empty"))
	}
	if err := VerifyProfile(p.profile); err != nil {
		return nil, failure(CodeIdentityChanged, err)
	}
	switch operation {
	case OperationLaunch:
		pid, err := p.platform.Launch(ctx, p.profile)
		if err != nil {
			return nil, mapPlatformFailure(CodeLaunchFailed, err)
		}
		return artifact.Marshal(LaunchResponse{ProcessID: pid})
	case OperationTerminate:
		count, err := p.platform.Terminate(ctx, p.profile)
		if err != nil {
			return nil, mapPlatformFailure(CodeTerminateFailed, err)
		}
		return artifact.Marshal(TerminateResponse{TerminatedCount: count})
	default:
		return nil, failure(CodeContractViolation, errors.New("application lifecycle operation is unsupported"))
	}
}

func (p *provider) Close(_ context.Context, object any) error {
	opened, ok := object.(*session)
	if !ok {
		return failure(CodeContractViolation, errors.New("application lifecycle object has the wrong type"))
	}
	opened.mu.Lock()
	opened.closed = true
	opened.mu.Unlock()
	return nil
}

func ProviderArtifactDigest(profile Profile) (artifact.Digest, error) {
	if !profile.Valid() {
		return "", errors.New("application provider artifact requires a profile")
	}
	manifest, err := artifact.Marshal(map[string]any{"providerAbi": ProviderABI, "implementation": providerImpl, "profileDigest": profile.Digest(), "profile": json.RawMessage(profile.Bytes())})
	if err != nil {
		return "", err
	}
	return artifact.Sum("yotta/provider-implementation-manifest/v1", manifest)
}

func OpenLaunchResponse(raw []byte) (LaunchResponse, error) {
	var response LaunchResponse
	if decodeExact(raw, &response, 4096) != nil || response.ProcessID == 0 {
		return LaunchResponse{}, failure(CodeContractViolation, errors.New("invalid application launch response"))
	}
	canonical, err := artifact.Marshal(response)
	if err != nil || !bytes.Equal(canonical, raw) {
		return LaunchResponse{}, failure(CodeContractViolation, errors.New("application launch response is not canonical"))
	}
	return response, nil
}
func OpenTerminateResponse(raw []byte) (TerminateResponse, error) {
	var response TerminateResponse
	if decodeExact(raw, &response, 4096) != nil || response.TerminatedCount < 0 {
		return TerminateResponse{}, failure(CodeContractViolation, errors.New("invalid application terminate response"))
	}
	canonical, err := artifact.Marshal(response)
	if err != nil || !bytes.Equal(canonical, raw) {
		return TerminateResponse{}, failure(CodeContractViolation, errors.New("application terminate response is not canonical"))
	}
	return response, nil
}

func validOperation(operation string) bool {
	return operation == OperationLaunch || operation == OperationTerminate
}
func mapPlatformFailure(code string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, errUnsupportedHost) {
		code = CodeUnsupportedHost
	}
	return failure(code, err)
}
func decodeExact(raw []byte, target any, maximum int) error {
	if len(raw) == 0 || len(raw) > maximum {
		return errors.New("JSON payload exceeds byte budget")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("JSON payload contains trailing values")
	}
	return nil
}
func failure(code string, cause error) error { return &Failure{Code: code, Cause: cause} }

var _ resource.Provider = (*provider)(nil)
