// Package schema defines the only durable Workflow Source contract accepted by Yotta 3.1.
package schema

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	Format         = "yotta.workflow"
	Version        = "3.1"
	MaxRevision    = 9_007_199_254_740_991
	MaxDiagnostics = 10_000
	MaxVariables   = 4_096
	MaxSecretRefs  = 4_096
	MaxGraphDepth  = 32
	MaxGraphPath   = MaxGraphDepth*2 - 1
)

type GraphKind string

const (
	GraphKindMain     GraphKind = "main"
	GraphKindSubgraph GraphKind = "subgraph"
)

type WorkflowSource struct {
	Format         string          `json:"format" jsonschema:"required,enum=yotta.workflow"`
	Version        string          `json:"version" jsonschema:"required,enum=3.1"`
	Workflow       Workflow        `json:"workflow" jsonschema:"required"`
	Revision       int64           `json:"revision" jsonschema:"required,minimum=0,maximum=9007199254740991"`
	EntryGraph     string          `json:"entryGraph" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Graphs         []Graph         `json:"graphs" jsonschema:"required,minItems=1,maxItems=256"`
	TargetDefaults []TargetDefault `json:"targetDefaults,omitempty" jsonschema:"maxItems=64"`
	Variables      []Variable      `json:"variables" jsonschema:"required,maxItems=4096"`
	SecretRefs     []SecretRef     `json:"secretRefs" jsonschema:"required,maxItems=4096"`
}

type TargetDefault struct {
	Target string `json:"target" jsonschema:"required,maxLength=128,pattern=^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$"`
	Slot   string `json:"slot" jsonschema:"required,maxLength=128,pattern=^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$"`
}

type Workflow struct {
	ID          string   `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Name        string   `json:"name" jsonschema:"required,minLength=1,maxLength=256"`
	Description string   `json:"description,omitempty" jsonschema:"maxLength=4096"`
	Category    string   `json:"category,omitempty" jsonschema:"maxLength=128"`
	Tags        []string `json:"tags,omitempty" jsonschema:"maxItems=64"`
	CreatedAt   string   `json:"createdAt,omitempty" jsonschema:"maxLength=64"`
	UpdatedAt   string   `json:"updatedAt,omitempty" jsonschema:"maxLength=64"`
}

func (w Workflow) validateTimestamps() error {
	createdAt, created, err := parseWorkflowTimestamp("createdAt", w.CreatedAt)
	if err != nil {
		return err
	}
	updatedAt, updated, err := parseWorkflowTimestamp("updatedAt", w.UpdatedAt)
	if err != nil {
		return err
	}
	if created && updated && updatedAt.Before(createdAt) {
		return fmt.Errorf("updatedAt must not be before createdAt")
	}
	return nil
}

func parseWorkflowTimestamp(field, value string) (time.Time, bool, error) {
	if value == "" {
		return time.Time{}, false, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("%s must be RFC 3339: %w", field, err)
	}
	return parsed, true, nil
}

type Graph struct {
	ID          string       `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Name        string       `json:"name,omitempty" jsonschema:"maxLength=256"`
	Kind        GraphKind    `json:"kind" jsonschema:"required,enum=main,enum=subgraph"`
	Nodes       []Node       `json:"nodes" jsonschema:"required,maxItems=4096"`
	Calls       []GraphCall  `json:"calls,omitempty" jsonschema:"maxItems=4096"`
	Edges       []Edge       `json:"edges" jsonschema:"required,maxItems=16384"`
	Inputs      []GraphPort  `json:"inputs" jsonschema:"required,maxItems=4096"`
	Outputs     []GraphPort  `json:"outputs" jsonschema:"required,maxItems=4096"`
	Entries     []Endpoint   `json:"entries,omitempty" jsonschema:"maxItems=64"`
	Exits       []GraphExit  `json:"exits,omitempty" jsonschema:"maxItems=64"`
	Annotations []Annotation `json:"annotations,omitempty" jsonschema:"maxItems=4096"`
}

type GraphPort struct {
	ID     string                  `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Type   datatype.TypeExpression `json:"type" jsonschema:"required"`
	NodeID string                  `json:"nodeId" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	PortID string                  `json:"portId" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
}

type GraphCall struct {
	ID       string                  `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	GraphID  string                  `json:"graphId" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Label    string                  `json:"label,omitempty" jsonschema:"maxLength=1024"`
	Position Position                `json:"position" jsonschema:"required"`
	Bindings map[string]InputBinding `json:"bindings" jsonschema:"required"`
}

type GraphExit struct {
	ID       string      `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Channel  EdgeChannel `json:"channel" jsonschema:"required,enum=exec,enum=error"`
	Endpoint Endpoint    `json:"endpoint" jsonschema:"required"`
}

type Annotation struct {
	ID       string   `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Text     string   `json:"text" jsonschema:"required,maxLength=16384"`
	Color    string   `json:"color,omitempty" jsonschema:"maxLength=32"`
	Position Position `json:"position" jsonschema:"required"`
	Size     Size     `json:"size" jsonschema:"required"`
}

type Size struct {
	Width  float64 `json:"width" jsonschema:"required"`
	Height float64 `json:"height" jsonschema:"required"`
}

type BindingKind string

const (
	BindingValue   BindingKind = "value"
	BindingDefault BindingKind = "default"
	BindingBlob    BindingKind = "blob"
)

type InputBinding struct {
	Kind  BindingKind     `json:"kind" jsonschema:"required,enum=value,enum=default,enum=blob"`
	Value json.RawMessage `json:"value,omitempty"`
	Blob  *blob.BlobRef   `json:"blob,omitempty"`
}

type Node struct {
	ID       string                  `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	NodeRef  nodecontract.NodeRef    `json:"nodeRef" jsonschema:"required"`
	Label    string                  `json:"label,omitempty"`
	Position Position                `json:"position" jsonschema:"required"`
	Config   map[string]any          `json:"config" jsonschema:"required"`
	Bindings map[string]InputBinding `json:"bindings" jsonschema:"required"`
	Disabled bool                    `json:"disabled,omitempty"`
}

type Position struct {
	X float64 `json:"x" jsonschema:"required"`
	Y float64 `json:"y" jsonschema:"required"`
}

type EdgeChannel string

const (
	EdgeData  EdgeChannel = "data"
	EdgeExec  EdgeChannel = "exec"
	EdgeError EdgeChannel = "error"
)

type Endpoint struct {
	NodeID string `json:"nodeId" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	PortID string `json:"portId" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
}

type Edge struct {
	Channel      EdgeChannel      `json:"channel" jsonschema:"required,enum=data,enum=exec,enum=error"`
	From         Endpoint         `json:"from" jsonschema:"required"`
	To           Endpoint         `json:"to" jsonschema:"required"`
	Presentation EdgePresentation `json:"presentation,omitempty"`
}

type EdgePresentation struct {
	Reroutes []Position `json:"reroutes,omitempty" jsonschema:"maxItems=64"`
}

type Variable struct {
	Name    string                  `json:"name" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Type    datatype.TypeExpression `json:"type" jsonschema:"required"`
	Default json.RawMessage         `json:"default" jsonschema:"required"`
}

type SecretRef struct {
	ID      string `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Purpose string `json:"purpose" jsonschema:"required,minLength=1"`
}
