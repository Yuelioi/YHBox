// Package schema defines the only durable Workflow Source contract accepted by Yotta 3.
package schema

const (
	Format      = "yotta.workflow"
	Version     = 3
	MaxRevision = 9_007_199_254_740_991
)

type GraphKind string

const (
	GraphKindMain     GraphKind = "main"
	GraphKindSubgraph GraphKind = "subgraph"
)

type Capability string

type WorkflowSource struct {
	Format                string       `json:"format" jsonschema:"required,enum=yotta.workflow"`
	Version               int          `json:"version" jsonschema:"required,enum=3"`
	Workflow              Workflow     `json:"workflow" jsonschema:"required"`
	Revision              int64        `json:"revision" jsonschema:"required,minimum=0,maximum=9007199254740991"`
	EntryGraph            string       `json:"entryGraph" jsonschema:"required,minLength=1"`
	Graphs                []Graph      `json:"graphs" jsonschema:"required,minItems=1"`
	Variables             []Variable   `json:"variables" jsonschema:"required"`
	SecretRefs            []SecretRef  `json:"secretRefs" jsonschema:"required"`
	RequestedCapabilities []Capability `json:"requestedCapabilities" jsonschema:"required"`
}

type Workflow struct {
	ID   string `json:"id" jsonschema:"required,minLength=1"`
	Name string `json:"name" jsonschema:"required,minLength=1"`
}

type Graph struct {
	ID      string      `json:"id" jsonschema:"required,minLength=1"`
	Kind    GraphKind   `json:"kind" jsonschema:"required,enum=main,enum=subgraph"`
	Nodes   []Node      `json:"nodes" jsonschema:"required"`
	Edges   []Edge      `json:"edges" jsonschema:"required"`
	Inputs  []GraphPort `json:"inputs" jsonschema:"required"`
	Outputs []GraphPort `json:"outputs" jsonschema:"required"`
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
