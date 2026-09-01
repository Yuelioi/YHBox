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
	OperationAgentStart         = "agent-start"
	OperationAgentContinue      = "agent-continue"
	resourceImplementation      = "provider-native-generation/v2"
)

type CredentialStore interface {
	Get(string) (string, error)
}

type CapabilityScope struct {
	Retention  RetentionRequirement `json:"retention"`
	Structured bool                 `json:"structured"`
	Agent      bool                 `json:"agent"`
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
	agentState   any
	agentTools   ToolSet
	pendingCalls map[string]string
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
		if operation != OperationGenerate && operation != OperationGenerateStructured && operation != OperationAgentStart && operation != OperationAgentContinue {
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
	if scope.Agent && scope.Structured {
		return nil, errors.New("AI agent and structured scopes are distinct")
	}
	hasAgentOperation := containsOperation(request.Operations, OperationAgentStart) || containsOperation(request.Operations, OperationAgentContinue)
	if scope.Agent && (!containsOperation(request.Operations, OperationAgentStart) || !containsOperation(request.Operations, OperationAgentContinue) ||
		containsOperation(request.Operations, OperationGenerate) || containsOperation(request.Operations, OperationGenerateStructured)) {
		return nil, errors.New("AI agent scope requires only start and continuation operations")
	}
	if !scope.Agent && hasAgentOperation {
		return nil, errors.New("AI agent operations require agent scope")
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
	if operation != OperationGenerate && operation != OperationGenerateStructured && operation != OperationAgentStart && operation != OperationAgentContinue {
		return nil, errors.New("AI model session operation is unsupported")
	}
	if operation == OperationAgentStart || operation == OperationAgentContinue {
		return p.invokeAgent(ctx, session, operation, payload)
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
	credential := "codex-subscription"
	if p.profile.Machine().Provider != ProviderCodexSubscription {
		var err error
		credential, err = p.credentials.Get(session.credentialID)
		if err != nil || credential == "" {
			return nil, errors.New("AI credential is unavailable")
		}
	}
	outcome, err := p.native.Generate(ctx, credential, request)
	if err != nil {
		return nil, err
	}
	return artifact.Marshal(outcome)
}

type nativeAgentProvider interface {
	StartAgent(context.Context, string, AgentStartRequest) (Outcome, any, error)
	ContinueAgent(context.Context, string, any, AgentContinueRequest) (Outcome, any, error)
}

func (p *resourceProvider) invokeAgent(ctx context.Context, session *modelSession, operation string, payload []byte) ([]byte, error) {
	if !session.scope.Agent || session.scope.Structured {
		return nil, errors.New("AI request agent mode does not match the granted scope")
	}
	provider, ok := p.native.(nativeAgentProvider)
	if !ok || !p.profile.Machine().Capabilities.ToolCalling {
		return nil, errors.New("installed AI model does not support native tool calling")
	}
	credential := "codex-subscription"
	if p.profile.Machine().Provider != ProviderCodexSubscription {
		var credentialErr error
		credential, credentialErr = p.credentials.Get(session.credentialID)
		if credentialErr != nil || credential == "" {
			return nil, errors.New("AI credential is unavailable")
		}
	}
	var outcome Outcome
	var next any
	var err error
	switch operation {
	case OperationAgentStart:
		if session.agentState != nil {
			return nil, errors.New("AI agent session is already active")
		}
		var request AgentStartRequest
		if err := decodeExactJSON(payload, &request); err != nil {
			return nil, fmt.Errorf("decode AI agent start request: %w", err)
		}
		if err := request.Validate(); err != nil {
			return nil, err
		}
		if request.Retention != session.scope.Retention {
			return nil, errors.New("AI agent retention does not match the granted scope")
		}
		toolSet, openErr := request.ToolSet.Open()
		if openErr != nil {
			return nil, openErr
		}
		outcome, next, err = provider.StartAgent(ctx, credential, request)
		if err == nil {
			session.agentTools = toolSet
		}
	case OperationAgentContinue:
		if session.agentState == nil || len(session.pendingCalls) == 0 {
			return nil, errors.New("AI agent session has no pending tool calls")
		}
		var request AgentContinueRequest
		if err := decodeExactJSON(payload, &request); err != nil {
			return nil, fmt.Errorf("decode AI agent continuation request: %w", err)
		}
		if err := request.Validate(); err != nil {
			return nil, err
		}
		if err := matchPendingToolResults(session.pendingCalls, request.Results); err != nil {
			return nil, err
		}
		if err := validateAgentToolResults(session.agentTools, request.Results); err != nil {
			return nil, err
		}
		outcome, next, err = provider.ContinueAgent(ctx, credential, session.agentState, request)
	}
	if err != nil {
		return nil, err
	}
	pending, err := pendingToolCalls(outcome)
	if err != nil {
		return nil, err
	}
	if len(pending) > 0 && next == nil {
		return nil, errors.New("AI provider omitted continuation state for pending tool calls")
	}
	session.agentState = next
	session.pendingCalls = pending
	if outcome.Finish.Kind != FinishToolCalls {
		session.agentState = nil
		session.agentTools = ToolSet{}
	}
	return artifact.Marshal(outcome)
}

func validateAgentToolResults(toolSet ToolSet, results []ToolResult) error {
	if !toolSet.Valid() {
		return errors.New("AI agent session has no exact ToolSet")
	}
	tools := make(map[string]ToolManifestDraft)
	for _, tool := range toolSet.Machine().Tools {
		tools[tool.Name] = tool
	}
	for _, result := range results {
		tool, ok := tools[result.Name]
		if !ok {
			return ErrAgentUnknownTool
		}
		output, err := CompileStructuredOutput(result.Name+"_output", tool.OutputSchema)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrAgentToolSchema, err)
		}
		canonical, err := output.ValidateValue(result.Value)
		if err != nil || !bytes.Equal(canonical, result.Value) {
			return ErrAgentToolSchema
		}
	}
	return nil
}

func pendingToolCalls(outcome Outcome) (map[string]string, error) {
	if err := outcome.Validate(); err != nil {
		return nil, err
	}
	pending := make(map[string]string)
	for _, item := range outcome.Items {
		if item.Kind != OutputToolCall {
			continue
		}
		if _, duplicate := pending[item.ToolCall.CallID]; duplicate {
			return nil, errors.New("AI provider returned duplicate tool call identity")
		}
		pending[item.ToolCall.CallID] = item.ToolCall.Name
	}
	if (outcome.Finish.Kind == FinishToolCalls) != (len(pending) > 0) {
		return nil, errors.New("AI provider tool calls do not match finish state")
	}
	return pending, nil
}

func matchPendingToolResults(pending map[string]string, results []ToolResult) error {
	if len(results) != len(pending) {
		return errors.New("AI agent continuation must resolve every pending tool call exactly once")
	}
	for _, result := range results {
		if pending[result.CallID] != result.Name {
			return errors.New("AI agent tool result does not match its pending call")
		}
	}
	return nil
}

func (p *resourceProvider) Close(_ context.Context, object any) error {
	session, ok := object.(*modelSession)
	if !ok {
		return errors.New("AI resource object has the wrong type")
	}
	session.mu.Lock()
	session.closed = true
	session.credentialID = ""
	session.agentState = nil
	session.agentTools = ToolSet{}
	session.pendingCalls = nil
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
