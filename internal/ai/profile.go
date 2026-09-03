package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
)

const profileDigestDomain = "yotta/ai-model-profile/v2"

type ProviderKind string

const (
	ProviderOpenAIResponses       ProviderKind = "openai-responses"
	ProviderOpenAIChatCompletions ProviderKind = "openai-chat-completions"
	ProviderAnthropicMessages     ProviderKind = "anthropic-messages"
	ProviderCodexSubscription     ProviderKind = "codex-subscription"
)

type EvaluationStatus string

const (
	EvaluationUnverified EvaluationStatus = "unverified"
	EvaluationApproved   EvaluationStatus = "approved"
	EvaluationRejected   EvaluationStatus = "rejected"
)

type ProfileCapabilities struct {
	StructuredOutput bool `json:"structuredOutput"`
	ToolCalling      bool `json:"toolCalling"`
	ParallelTools    bool `json:"parallelTools"`
	Background       bool `json:"background"`
	ZeroRetention    bool `json:"zeroRetention"`
}

type ModelProfileDraft struct {
	Provider         ProviderKind        `json:"provider"`
	Endpoint         string              `json:"endpoint"`
	AllowLocalHTTP   bool                `json:"allowLocalHttp"`
	Model            string              `json:"model"`
	Capabilities     ProfileCapabilities `json:"capabilities"`
	MaxOutputTokens  int64               `json:"maxOutputTokens"`
	Pricing          TokenPricing        `json:"pricing"`
	Evaluation       EvaluationStatus    `json:"evaluation"`
	EvaluationSuite  artifact.Digest     `json:"evaluationSuite,omitempty"`
	EvaluationReport artifact.Digest     `json:"evaluationReport,omitempty"`
	ProviderMetadata json.RawMessage     `json:"providerMetadata"`
}

type modelProfileState struct {
	digest   artifact.Digest
	document ModelProfileDraft
	bytes    []byte
}

type ModelProfile struct{ state *modelProfileState }

var modelPattern = regexp.MustCompile(`^[^\x00-\x1f\x7f]{1,256}$`)

func SealModelProfile(draft ModelProfileDraft) (ModelProfile, error) {
	if (draft.Provider != ProviderOpenAIResponses && draft.Provider != ProviderOpenAIChatCompletions && draft.Provider != ProviderAnthropicMessages && draft.Provider != ProviderCodexSubscription) || !modelPattern.MatchString(draft.Model) ||
		draft.MaxOutputTokens < 0 || draft.MaxOutputTokens > 1_000_000 {
		return ModelProfile{}, errors.New("invalid AI model profile identity or budget")
	}
	if draft.Capabilities.ParallelTools && !draft.Capabilities.ToolCalling {
		return ModelProfile{}, errors.New("parallel AI tool calls require tool calling")
	}
	if draft.Provider == ProviderOpenAIChatCompletions && draft.Capabilities.ToolCalling {
		return ModelProfile{}, errors.New("OpenAI Chat Completions agent tool calling is not supported")
	}
	endpoint, err := NormalizeProviderEndpoint(draft.Provider, draft.Endpoint, draft.AllowLocalHTTP)
	if err != nil {
		return ModelProfile{}, err
	}
	draft.Endpoint = endpoint
	if strings.HasPrefix(endpoint, "https://") {
		draft.AllowLocalHTTP = false
	}
	if draft.Capabilities.ToolCalling && draft.Provider != ProviderCodexSubscription {
		if err := draft.Pricing.Validate(); err != nil {
			return ModelProfile{}, err
		}
	}
	switch draft.Evaluation {
	case EvaluationUnverified:
		if draft.EvaluationSuite != "" || draft.EvaluationReport != "" {
			return ModelProfile{}, errors.New("unverified AI profile cannot claim evaluation evidence")
		}
	case EvaluationApproved, EvaluationRejected:
		if !draft.EvaluationSuite.Valid() || !draft.EvaluationReport.Valid() {
			return ModelProfile{}, errors.New("evaluated AI profile requires suite and report digests")
		}
	default:
		return ModelProfile{}, errors.New("invalid AI model evaluation status")
	}
	metadata := draft.ProviderMetadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	if len(metadata) > 64<<10 {
		return ModelProfile{}, errors.New("AI provider metadata exceeds byte budget")
	}
	canonicalMetadata, err := artifact.Canonicalize(metadata)
	if err != nil {
		return ModelProfile{}, errors.New("AI provider metadata must be canonical JSON")
	}
	draft.ProviderMetadata = canonicalMetadata
	raw, err := artifact.Marshal(draft)
	if err != nil {
		return ModelProfile{}, err
	}
	digest, err := artifact.Sum(profileDigestDomain, raw)
	if err != nil {
		return ModelProfile{}, err
	}
	return ModelProfile{state: &modelProfileState{digest: digest, document: draft, bytes: raw}}, nil
}

func DefaultProviderEndpoint(provider ProviderKind) string {
	switch provider {
	case ProviderOpenAIResponses:
		return OpenAIResponsesEndpoint
	case ProviderOpenAIChatCompletions:
		return OpenAIChatCompletionsBaseURL
	case ProviderAnthropicMessages:
		return AnthropicMessagesEndpoint
	case ProviderCodexSubscription:
		return "codex://subscription"
	default:
		return ""
	}
}

// NormalizeProviderEndpoint seals the configured provider URL without
// inventing or requiring a provider-specific request path. It permits plain
// HTTP only for an explicitly acknowledged loopback installation.
func NormalizeProviderEndpoint(provider ProviderKind, raw string, allowLocalHTTP bool) (string, error) {
	if provider == ProviderCodexSubscription {
		if raw == "" || raw == "codex://subscription" {
			return "codex://subscription", nil
		}
		return "", errors.New("codex subscription provider endpoint is managed by Codex")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = DefaultProviderEndpoint(provider)
	}
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" || parsed.Opaque != "" {
		return "", errors.New("AI endpoint must be an absolute URL without credentials, query, or fragment")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	if parsed.Scheme != "https" {
		if parsed.Scheme != "http" || !allowLocalHTTP || !isLoopbackHost(parsed.Hostname()) {
			return "", errors.New("AI endpoint requires HTTPS; HTTP is allowed only for an explicitly enabled loopback host")
		}
	}
	if strings.Contains(parsed.Path, "\\") {
		return "", errors.New("AI endpoint path is invalid")
	}
	parsed.RawPath = ""
	return parsed.String(), nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func OpenModelProfile(raw []byte, digest artifact.Digest) (ModelProfile, error) {
	if !digest.Valid() || len(raw) == 0 || len(raw) > 128<<10 {
		return ModelProfile{}, errors.New("invalid AI model profile artifact")
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return ModelProfile{}, errors.New("AI model profile is not canonical")
	}
	var draft ModelProfileDraft
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&draft); err != nil {
		return ModelProfile{}, err
	}
	sealed, err := SealModelProfile(draft)
	if err != nil || sealed.Digest() != digest || !bytes.Equal(sealed.Bytes(), raw) {
		return ModelProfile{}, errors.New("AI model profile digest mismatch")
	}
	return sealed, nil
}

func (p ModelProfile) Valid() bool { return p.state != nil && p.state.digest.Valid() }
func (p ModelProfile) Digest() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.digest
}
func (p ModelProfile) Bytes() []byte {
	if !p.Valid() {
		return nil
	}
	return append([]byte(nil), p.state.bytes...)
}
func (p ModelProfile) Machine() ModelProfileDraft {
	if !p.Valid() {
		return ModelProfileDraft{}
	}
	clone := p.state.document
	clone.ProviderMetadata = append(json.RawMessage(nil), clone.ProviderMetadata...)
	return clone
}
