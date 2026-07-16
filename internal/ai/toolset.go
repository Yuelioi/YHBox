package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	toolSetDigestDomain = "yotta/ai-tool-set/v1"
	MaxToolSetTools     = 128
	MaxToolSetBytes     = 1 << 20
)

var toolNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

type ToolManifestDraft struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Authority    ToolAuthority   `json:"authority"`
	Capability   artifact.Digest `json:"capability,omitempty"`
	InputSchema  json.RawMessage `json:"inputSchema"`
	OutputSchema json.RawMessage `json:"outputSchema"`
}

type ToolAuthority string

const (
	ToolAuthorityPure       ToolAuthority = "pure"
	ToolAuthorityCapability ToolAuthority = "capability"
)

type ToolSetDraft struct {
	ID      string              `json:"id"`
	Version string              `json:"version"`
	Owner   string              `json:"owner"`
	Tools   []ToolManifestDraft `json:"tools"`
}

type toolSetState struct {
	digest   artifact.Digest
	document ToolSetDraft
	bytes    []byte
}

type ToolSet struct{ state *toolSetState }

func SealToolSet(draft ToolSetDraft) (ToolSet, error) {
	if !promptManifestIDPattern.MatchString(draft.ID) || !promptVersionPattern.MatchString(draft.Version) ||
		!promptOwnerPattern.MatchString(draft.Owner) || len(draft.Tools) == 0 || len(draft.Tools) > MaxToolSetTools {
		return ToolSet{}, errors.New("invalid AI tool set identity or tool budget")
	}
	tools := append([]ToolManifestDraft(nil), draft.Tools...)
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
	previous := ""
	for index := range tools {
		tool := &tools[index]
		if !toolNamePattern.MatchString(tool.Name) || tool.Name <= previous || tool.Description == "" || len(tool.Description) > 4096 {
			return ToolSet{}, errors.New("invalid or duplicate AI tool manifest")
		}
		switch tool.Authority {
		case ToolAuthorityPure:
			if tool.Capability != "" {
				return ToolSet{}, errors.New("pure AI tool cannot claim capability authority")
			}
		case ToolAuthorityCapability:
			if !tool.Capability.Valid() {
				return ToolSet{}, errors.New("capability AI tool requires approved authority identity")
			}
		default:
			return ToolSet{}, errors.New("invalid AI tool authority")
		}
		previous = tool.Name
		input, err := CompileStructuredOutput(tool.Name+"_input", tool.InputSchema)
		if err != nil {
			return ToolSet{}, fmt.Errorf("compile AI tool %q input: %w", tool.Name, err)
		}
		output, err := CompileStructuredOutput(tool.Name+"_output", tool.OutputSchema)
		if err != nil {
			return ToolSet{}, fmt.Errorf("compile AI tool %q output: %w", tool.Name, err)
		}
		tool.InputSchema = append(json.RawMessage(nil), input.Schema...)
		tool.OutputSchema = append(json.RawMessage(nil), output.Schema...)
	}
	draft.Tools = tools
	raw, err := artifact.Marshal(draft)
	if err != nil {
		return ToolSet{}, err
	}
	if len(raw) > MaxToolSetBytes {
		return ToolSet{}, errors.New("AI tool set exceeds its byte budget")
	}
	digest, err := artifact.Sum(toolSetDigestDomain, raw)
	if err != nil {
		return ToolSet{}, err
	}
	return ToolSet{state: &toolSetState{digest: digest, document: draft, bytes: raw}}, nil
}

func OpenToolSet(raw []byte, digest artifact.Digest) (ToolSet, error) {
	if !digest.Valid() || len(raw) == 0 || len(raw) > MaxToolSetBytes {
		return ToolSet{}, errors.New("invalid AI tool set artifact")
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return ToolSet{}, errors.New("AI tool set is not canonical")
	}
	var draft ToolSetDraft
	if err := decodeExactJSON(raw, &draft); err != nil {
		return ToolSet{}, fmt.Errorf("decode AI tool set: %w", err)
	}
	sealed, err := SealToolSet(draft)
	if err != nil || sealed.Digest() != digest || !bytes.Equal(sealed.Bytes(), raw) {
		return ToolSet{}, errors.New("AI tool set digest mismatch")
	}
	return sealed, nil
}

func (s ToolSet) Valid() bool { return s.state != nil && s.state.digest.Valid() }

func (s ToolSet) Digest() artifact.Digest {
	if !s.Valid() {
		return ""
	}
	return s.state.digest
}

func (s ToolSet) Bytes() []byte {
	if !s.Valid() {
		return nil
	}
	return append([]byte(nil), s.state.bytes...)
}

func (s ToolSet) Machine() ToolSetDraft {
	if !s.Valid() {
		return ToolSetDraft{}
	}
	clone := s.state.document
	clone.Tools = append([]ToolManifestDraft(nil), clone.Tools...)
	for index := range clone.Tools {
		clone.Tools[index].InputSchema = append(json.RawMessage(nil), clone.Tools[index].InputSchema...)
		clone.Tools[index].OutputSchema = append(json.RawMessage(nil), clone.Tools[index].OutputSchema...)
	}
	return clone
}
