// Package schema defines the only durable Workflow Source contract accepted by Yotta 3.1.
package schema

import (
	"encoding/json"

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
)

type GraphKind string

const (
	GraphKindMain     GraphKind = "main"
	GraphKindSubgraph GraphKind = "subgraph"
)

type WorkflowSource struct {
	Format     string      `json:"format" jsonschema:"required,enum=yotta.workflow"`
	Version    string      `json:"version" jsonschema:"required,enum=3.1"`
	Workflow   Workflow    `json:"workflow" jsonschema:"required"`
	Revision   int64       `json:"revision" jsonschema:"required,minimum=0,maximum=9007199254740991"`
	EntryGraph string      `json:"entryGraph" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Graphs     []Graph     `json:"graphs" jsonschema:"required,minItems=1,maxItems=256"`
	Variables  []Variable  `json:"variables" jsonschema:"required,maxItems=4096"`
	SecretRefs []SecretRef `json:"secretRefs" jsonschema:"required,maxItems=4096"`
}

type Workflow struct {
	ID   string `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Name string `json:"name" jsonschema:"required,minLength=1"`
}

type Graph struct {
	ID      string      `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Kind    GraphKind   `json:"kind" jsonschema:"required,enum=main,enum=subgraph"`
	Nodes   []Node      `json:"nodes" jsonschema:"required,maxItems=4096"`
	Edges   []Edge      `json:"edges" jsonschema:"required,maxItems=16384"`
	Inputs  []GraphPort `json:"inputs" jsonschema:"required,maxItems=4096"`
	Outputs []GraphPort `json:"outputs" jsonschema:"required,maxItems=4096"`
}

type GraphPort struct {
	ID     string                  `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Type   datatype.TypeExpression `json:"type" jsonschema:"required"`
	NodeID string                  `json:"nodeId" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	PortID string                  `json:"portId" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
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
	EdgeData   EdgeChannel = "data"
	EdgeExec   EdgeChannel = "exec"
	EdgeError  EdgeChannel = "error"
	EdgeStatus EdgeChannel = "status"
)

type Endpoint struct {
	NodeID string `json:"nodeId" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	PortID string `json:"portId" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
}

type Edge struct {
	Channel EdgeChannel `json:"channel" jsonschema:"required,enum=data,enum=exec,enum=error,enum=status"`
	From    Endpoint    `json:"from" jsonschema:"required"`
	To      Endpoint    `json:"to" jsonschema:"required"`
}

type Variable struct {
	Name    string                  `json:"name" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Type    datatype.TypeExpression `json:"type" jsonschema:"required"`
	Default json.RawMessage         `json:"default,omitempty"`
}

type SecretRef struct {
	ID      string `json:"id" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	Purpose string `json:"purpose" jsonschema:"required,minLength=1"`
}
