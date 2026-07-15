package ai

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

const (
	ProviderABI                 = "https://schemas.yotta.dev/provider-abi/resource/v1"
	KindModelSession            = "ai/model-session"
	OperationGenerate           = "generate"
	OperationGenerateStructured = "generate-structured"
	resourceImplementation      = "provider-native-generation/v1"
)

type CredentialStore interface {
	Get(string) (string, error)
}

type CapabilityScope struct {
	Retention  RetentionRequirement `json:"retention"`
	Structured bool                 `json:"structured"`
}

type resourceProvider struct {
	profile     ModelProfile
	native      Provider
	credentials CredentialStore
}

type modelSession struct {
	mu           sync.Mutex
	credentialID string
	scope        CapabilityScope
	closed       bool
}

func NewResourceProvider(profile ModelProfile, native Provider, credentials CredentialStore) (resource.Provider, error) {
	if !profile.Valid() || native == nil || credentials == nil {
		return nil, errors.New("AI resource provider requires profile, native adapter, and credential store")
	}
	return &resourceProvider{profile: profile, native: native, credentials: credentials}, nil
}

func ProviderArtifactDigest(profile ModelProfile) (artifact.Digest, error) {
	if !profile.Valid() {
		return "", errors.New("AI provider artifact requires a model profile")
	}
	manifest, err := artifact.Marshal(map[string]any{
		"providerAbi": ProviderABI, "implementation": resourceImplementation,
		"profileDigest": profile.Digest(), "profile": json.RawMessage(profile.Bytes()),
	})
	if err != nil {
		return "", err
	}
	return artifact.Sum("yotta/provider-implementation-manifest/v1", manifest)
}

func InstallationID(prefix string, profile ModelProfile) (string, error) {
	if prefix == "" || !profile.Valid() {
		return "", errors.New("AI installation identity is invalid")
	}
	digest := profile.Digest().String()
	const digestPrefix = "sha256:"
	if len(digest) != len(digestPrefix)+64 || digest[:len(digestPrefix)] != digestPrefix {
		return "", errors.New("AI profile digest is invalid")
	}
	if _, err := hex.DecodeString(digest[len(digestPrefix):]); err != nil {
		return "", err
	}
	return prefix + "-" + digest[len(digestPrefix):len(digestPrefix)+32], nil
}

func (p *resourceProvider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	if request.Kind != KindModelSession || request.CredentialBindingID == "" || len(request.Operations) == 0 {
		return nil, errors.New("invalid AI model session request")
	}
	for _, operation := range request.Operations {
		if operation != OperationGenerate && operation != OperationGenerateStructured {
			return nil, errors.New("AI model session requested an unsupported operation")
		}
	}
	var config map[string]any
	if err := decodeExactJSON(request.Config, &config); err != nil || len(config) != 0 {
		return nil, errors.New("AI model session config must be an empty object")
	}
	var scope CapabilityScope
	if err := decodeExactJSON(request.CapabilityScope, &scope); err != nil {
		return nil, errors.New("invalid AI capability scope")
	}
	if scope.Retention != RetentionProviderDefault && scope.Retention != RetentionNoApplicationState && scope.Retention != RetentionZeroRequired {
		return nil, errors.New("invalid AI retention scope")
	}
	if scope.Structured && !containsOperation(request.Operations, OperationGenerateStructured) {
		return nil, errors.New("AI structured scope lacks its operation")
	}
	return &modelSession{credentialID: request.CredentialBindingID, scope: scope}, nil
}

func (p *resourceProvider) Invoke(ctx context.Context, object any, operation string, payload []byte) ([]byte, error) {
	session, ok := object.(*modelSession)
	if !ok {
		return nil, errors.New("AI resource object has the wrong type")
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil, errors.New("AI model session is closed")
	}
	if operation != OperationGenerate && operation != OperationGenerateStructured {
		return nil, errors.New("AI model session operation is unsupported")
	}
	var request GenerateRequest
	if err := decodeExactJSON(payload, &request); err != nil {
		return nil, fmt.Errorf("decode AI generation request: %w", err)
	}
	if request.Retention != session.scope.Retention {
		return nil, errors.New("AI request retention does not match the granted scope")
	}
	wantsStructured := request.Output != nil
	if wantsStructured != session.scope.Structured || wantsStructured != (operation == OperationGenerateStructured) {
		return nil, errors.New("AI request structured mode does not match the granted operation")
	}
	credential, err := p.credentials.Get(session.credentialID)
	if err != nil || credential == "" {
		return nil, errors.New("AI credential is unavailable")
	}
	outcome, err := p.native.Generate(ctx, credential, request)
	if err != nil {
		return nil, err
	}
	return artifact.Marshal(outcome)
}

func (p *resourceProvider) Close(_ context.Context, object any) error {
	session, ok := object.(*modelSession)
	if !ok {
		return errors.New("AI resource object has the wrong type")
	}
	session.mu.Lock()
	session.closed = true
	session.credentialID = ""
	session.mu.Unlock()
	return nil
}

func decodeExactJSON(raw []byte, target any) error {
	if len(raw) == 0 || len(raw) > MaxProviderResponseBytes {
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
		return errors.New("JSON payload contains trailing values")
	}
	return nil
}

func containsOperation(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
