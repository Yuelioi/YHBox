package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	promptManifestDigestDomain = "yotta/ai-prompt-manifest/v1"
	MaxPromptInstructionsBytes = 64 << 10
	MaxPromptBlocks            = 256
)

var (
	promptManifestIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._/-][a-z0-9]+)*$`)
	promptVersionPattern    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	promptOwnerPattern      = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)
)

type PromptManifestDraft struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Owner        string `json:"owner"`
	Instructions string `json:"instructions"`
}

type promptManifestState struct {
	digest   artifact.Digest
	document PromptManifestDraft
	bytes    []byte
}

type PromptManifest struct{ state *promptManifestState }

func SealPromptManifest(draft PromptManifestDraft) (PromptManifest, error) {
	if !promptManifestIDPattern.MatchString(draft.ID) || !promptVersionPattern.MatchString(draft.Version) ||
		!promptOwnerPattern.MatchString(draft.Owner) || draft.Instructions == "" || len(draft.Instructions) > MaxPromptInstructionsBytes {
		return PromptManifest{}, errors.New("invalid AI prompt manifest identity or instruction budget")
	}
	raw, err := artifact.Marshal(draft)
	if err != nil {
		return PromptManifest{}, err
	}
	digest, err := artifact.Sum(promptManifestDigestDomain, raw)
	if err != nil {
		return PromptManifest{}, err
	}
	return PromptManifest{state: &promptManifestState{digest: digest, document: draft, bytes: raw}}, nil
}

func OpenPromptManifest(raw []byte, digest artifact.Digest) (PromptManifest, error) {
	if !digest.Valid() || len(raw) == 0 || len(raw) > MaxPromptInstructionsBytes+4096 {
		return PromptManifest{}, errors.New("invalid AI prompt manifest artifact")
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return PromptManifest{}, errors.New("AI prompt manifest is not canonical")
	}
	var draft PromptManifestDraft
	if err := decodeExactJSON(raw, &draft); err != nil {
		return PromptManifest{}, fmt.Errorf("decode AI prompt manifest: %w", err)
	}
	sealed, err := SealPromptManifest(draft)
	if err != nil || sealed.Digest() != digest || !bytes.Equal(sealed.Bytes(), raw) {
		return PromptManifest{}, errors.New("AI prompt manifest digest mismatch")
	}
	return sealed, nil
}

func (p PromptManifest) Valid() bool { return p.state != nil && p.state.digest.Valid() }

func (p PromptManifest) Digest() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.digest
}

func (p PromptManifest) Bytes() []byte {
	if !p.Valid() {
		return nil
	}
	return append([]byte(nil), p.state.bytes...)
}

func (p PromptManifest) Machine() PromptManifestDraft {
	if !p.Valid() {
		return PromptManifestDraft{}
	}
	return p.state.document
}

type PromptBlockKind string

const (
	PromptBlockUser       PromptBlockKind = "user"
	PromptBlockContext    PromptBlockKind = "context"
	PromptBlockToolResult PromptBlockKind = "tool-result"
)

type PromptBlock struct {
	Kind     PromptBlockKind `json:"kind"`
	SourceID string          `json:"sourceId,omitempty"`
	Content  string          `json:"content"`
}

func (b PromptBlock) Validate() error {
	switch b.Kind {
	case PromptBlockUser, PromptBlockContext:
		if b.SourceID != "" {
			return errors.New("AI user/context prompt block cannot claim a tool source")
		}
	case PromptBlockToolResult:
		if !attemptIDPattern.MatchString(b.SourceID) {
			return errors.New("AI tool-result prompt block requires a valid source")
		}
	default:
		return errors.New("invalid AI prompt block kind")
	}
	if b.Content == "" || len(b.Content) > MaxPromptBytes {
		return errors.New("AI prompt block exceeds its content budget")
	}
	return nil
}

type RenderedPrompt struct {
	ManifestDigest artifact.Digest `json:"manifestDigest"`
	Manifest       json.RawMessage `json:"manifest"`
	Blocks         []PromptBlock   `json:"blocks"`
}

func RenderPrompt(manifest PromptManifest, blocks []PromptBlock) (RenderedPrompt, error) {
	if !manifest.Valid() || len(blocks) == 0 || len(blocks) > MaxPromptBlocks {
		return RenderedPrompt{}, errors.New("AI rendered prompt requires a trusted manifest and bounded blocks")
	}
	result := RenderedPrompt{
		ManifestDigest: manifest.Digest(), Manifest: manifest.Bytes(), Blocks: append([]PromptBlock(nil), blocks...),
	}
	if err := result.Validate(); err != nil {
		return RenderedPrompt{}, err
	}
	return result, nil
}

func (p RenderedPrompt) Validate() error {
	if _, err := p.OpenManifest(); err != nil {
		return err
	}
	if len(p.Blocks) == 0 || len(p.Blocks) > MaxPromptBlocks {
		return errors.New("AI rendered prompt block budget is invalid")
	}
	total := len(p.Manifest)
	for _, block := range p.Blocks {
		if err := block.Validate(); err != nil {
			return err
		}
		total += len(block.Content) + len(block.SourceID)
		if total > MaxPromptBytes {
			return errors.New("AI rendered prompt exceeds its byte budget")
		}
	}
	return nil
}

func (p RenderedPrompt) OpenManifest() (PromptManifest, error) {
	return OpenPromptManifest(p.Manifest, p.ManifestDigest)
}

func (p RenderedPrompt) ProviderInput() (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	if len(p.Blocks) == 1 && p.Blocks[0].Kind == PromptBlockUser {
		return p.Blocks[0].Content, nil
	}
	raw, err := artifact.Marshal(p.Blocks)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
