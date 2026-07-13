// Package schema defines the only durable Workflow Source contract accepted by Yotta 3.
package schema

import contractjsonschema "github.com/invopop/jsonschema"

const (
	Format                   = "yotta.workflow"
	Version                  = 3
	MaxRevision              = 9_007_199_254_740_991
	MaxDiagnostics           = 10_000
	MaxVariables             = 4_096
	MaxSecretRefs            = 4_096
	MaxRequestedCapabilities = 4_096
)

type GraphKind string

const (
	GraphKindMain     GraphKind = "main"
	GraphKindSubgraph GraphKind = "subgraph"
)

type Capability string

func (Capability) JSONSchema() *contractjsonschema.Schema {
	minimum := uint64(1)
	return &contractjsonschema.Schema{Type: "string", MinLength: &minimum}
}

type WorkflowSource struct {
	Format                string       `json:"format" jsonschema:"required,enum=yotta.workflow"`
	Version               int          `json:"version" jsonschema:"required,enum=3"`
	Workflow              Workflow     `json:"workflow" jsonschema:"required"`
	Revision              int64        `json:"revision" jsonschema:"required,minimum=0,maximum=9007199254740991"`
	EntryGraph            string       `json:"entryGraph" jsonschema:"required,minLength=1"`
	Graphs                []Graph      `json:"graphs" jsonschema:"required,minItems=1,maxItems=256"`
	Variables             []Variable   `json:"variables" jsonschema:"required,maxItems=4096"`
	SecretRefs            []SecretRef  `json:"secretRefs" jsonschema:"required,maxItems=4096"`
	RequestedCapabilities []Capability `json:"requestedCapabilities" jsonschema:"required,maxItems=4096"`
}

type Workflow struct {
	ID   string `json:"id" jsonschema:"required,minLength=1"`
	Name string `json:"name" jsonschema:"required,minLength=1"`
}

type Graph struct {
	ID      string      `json:"id" jsonschema:"required,minLength=1"`
	Kind    GraphKind   `json:"kind" jsonschema:"required,enum=main,enum=subgraph"`
	Nodes   []Node      `json:"nodes" jsonschema:"required,maxItems=4096"`
	Edges   []Edge      `json:"edges" jsonschema:"required,maxItems=16384"`
	Inputs  []GraphPort `json:"inputs" jsonschema:"required,maxItems=4096"`
	Outputs []GraphPort `json:"outputs" jsonschema:"required,maxItems=4096"`
}

type GraphPort struct {
	ID     string `json:"id" jsonschema:"required,minLength=1"`
	Name   string `json:"name" jsonschema:"required,minLength=1"`
	Type   string `json:"type" jsonschema:"required,minLength=1"`
	NodeID string `json:"nodeId" jsonschema:"required,minLength=1"`
}

type Node struct {
	ID       string         `json:"id" jsonschema:"required,minLength=1"`
	Kind     string         `json:"kind" jsonschema:"required,minLength=1"`
	Label    string         `json:"label,omitempty"`
	Position Position       `json:"position" jsonschema:"required"`
	Config   map[string]any `json:"config" jsonschema:"required"`
	Disabled bool           `json:"disabled,omitempty"`
}

type Position struct {
	X float64 `json:"x" jsonschema:"required"`
	Y float64 `json:"y" jsonschema:"required"`
}

type Edge struct {
	From string `json:"from" jsonschema:"required,minLength=1"`
	To   string `json:"to" jsonschema:"required,minLength=1"`
}

type Variable struct {
	Name    string `json:"name" jsonschema:"required,minLength=1"`
	Type    string `json:"type" jsonschema:"required,minLength=1"`
	Default any    `json:"default,omitempty"`
}

type SecretRef struct {
	ID      string `json:"id" jsonschema:"required,minLength=1"`
	Purpose string `json:"purpose" jsonschema:"required,minLength=1"`
}
