package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/yottaapp/yotta/internal/artifact"
)

const profileDigestDomain = "yotta/ai-model-profile/v1"

type ProviderKind string

const (
	ProviderOpenAIResponses   ProviderKind = "openai-responses"
	ProviderAnthropicMessages ProviderKind = "anthropic-messages"
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
	Model            string              `json:"model"`
	Capabilities     ProfileCapabilities `json:"capabilities"`
	MaxOutputTokens  int64               `json:"maxOutputTokens"`
	Pricing          TokenPricing        `json:"pricing"`
	Evaluation       EvaluationStatus    `json:"evaluation"`
	EvaluationSuite  artifact.Digest     `json:"evaluationSuite,omitempty"`
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
	if (draft.Provider != ProviderOpenAIResponses && draft.Provider != ProviderAnthropicMessages) || !modelPattern.MatchString(draft.Model) ||
		draft.MaxOutputTokens <= 0 || draft.MaxOutputTokens > 1_000_000 {
		return ModelProfile{}, errors.New("invalid AI model profile identity or budget")
	}
	if draft.Capabilities.ParallelTools && !draft.Capabilities.ToolCalling {
		return ModelProfile{}, errors.New("parallel AI tool calls require tool calling")
	}
	if draft.Capabilities.ToolCalling {
		if err := draft.Pricing.Validate(); err != nil {
			return ModelProfile{}, err
		}
	}
	switch draft.Evaluation {
	case EvaluationUnverified:
		if draft.EvaluationSuite != "" {
			return ModelProfile{}, errors.New("unverified AI profile cannot claim an evaluation suite")
		}
	case EvaluationApproved, EvaluationRejected:
		if !draft.EvaluationSuite.Valid() {
			return ModelProfile{}, errors.New("evaluated AI profile requires a suite digest")
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
