package capability

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	PlanFormat       = "yotta.capability-plan"
	PlanVersion      = "1"
	MaxPlanBytes     = 4 << 20
	planDigestDomain = "yotta/capability-plan/v1"
)

type PlanEntry struct {
	GraphID     string      `json:"graphId"`
	NodeID      string      `json:"nodeId"`
	Requirement Requirement `json:"requirement"`
}

type planDocument struct {
	Format  string          `json:"format"`
	Version string          `json:"version"`
	Digest  artifact.Digest `json:"digest"`
	Entries []PlanEntry     `json:"entries"`
}

type planState struct {
	document planDocument
	bytes    []byte
}
type Plan struct{ state *planState }

func SealPlan(source []PlanEntry) (Plan, error) {
	if len(source) > 16384 {
		return Plan{}, errors.New("capability plan exceeds entry budget")
	}
	entries := append([]PlanEntry(nil), source...)
	for index := range entries {
		entry := &entries[index]
		if !validAttributionID(entry.GraphID) || !validAttributionID(entry.NodeID) {
			return Plan{}, errors.New("invalid attributed capability plan entry")
		}
		normalized, err := NormalizeRequirementSyntax(entry.Requirement)
		if err != nil {
			return Plan{}, fmt.Errorf("invalid attributed capability requirement: %w", err)
		}
		entry.Requirement = normalized
	}
	sort.Slice(entries, func(i, j int) bool {
		left, right := entries[i], entries[j]
		if left.GraphID != right.GraphID {
			return left.GraphID < right.GraphID
		}
		if left.NodeID != right.NodeID {
			return left.NodeID < right.NodeID
		}
		return left.Requirement.ID < right.Requirement.ID
	})
	for index := 1; index < len(entries); index++ {
		left, right := entries[index-1], entries[index]
		if left.GraphID == right.GraphID && left.NodeID == right.NodeID && left.Requirement.ID == right.Requirement.ID {
			return Plan{}, errors.New("duplicate attributed capability requirement")
		}
	}
	body, err := artifact.Marshal(entries)
	if err != nil {
		return Plan{}, err
	}
	digest, err := artifact.Sum(planDigestDomain, body)
	if err != nil {
		return Plan{}, err
	}
	document := planDocument{Format: PlanFormat, Version: PlanVersion, Digest: digest, Entries: entries}
	raw, err := artifact.Marshal(document)
	if err != nil {
		return Plan{}, err
	}
	if len(raw) > MaxPlanBytes {
		return Plan{}, errors.New("capability plan exceeds byte budget")
	}
	return Plan{state: &planState{document: document, bytes: raw}}, nil
}

func OpenPlan(raw []byte) (Plan, error) {
	if len(raw) == 0 || len(raw) > MaxPlanBytes {
		return Plan{}, errors.New("capability plan exceeds byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, 96, 524288, 1<<20); err != nil {
		return Plan{}, err
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Plan{}, errors.New("capability plan is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document planDocument
	if err := decoder.Decode(&document); err != nil {
		return Plan{}, fmt.Errorf("decode capability plan: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Plan{}, errors.New("capability plan contains trailing values")
	}
	if document.Format != PlanFormat || document.Version != PlanVersion {
		return Plan{}, errors.New("unsupported capability plan")
	}
	sealed, err := SealPlan(document.Entries)
	if err != nil {
		return Plan{}, err
	}
	if sealed.Digest() != document.Digest || !bytes.Equal(sealed.Bytes(), raw) {
		return Plan{}, errors.New("capability plan digest mismatch")
	}
	return sealed, nil
}

func (p Plan) Valid() bool { return p.state != nil && p.state.document.Digest.Valid() }
func (p Plan) Digest() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.document.Digest
}
func (p Plan) Bytes() []byte {
	if !p.Valid() {
		return nil
	}
	return append([]byte(nil), p.state.bytes...)
}
func (p Plan) Entries() []PlanEntry {
	if !p.Valid() {
		return nil
	}
	raw, err := json.Marshal(p.state.document.Entries)
	if err != nil {
		panic(err)
	}
	var result []PlanEntry
	if err := json.Unmarshal(raw, &result); err != nil {
		panic(err)
	}
	return result
}
