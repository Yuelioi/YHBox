package nodes31

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	PresentationFormat  = "yotta.node-presentation"
	PresentationVersion = "3.1"
	GeneratorVersion    = "v1"
)

type presentationBody struct {
	GeneratorVersion string            `json:"generatorVersion"`
	Types            []json.RawMessage `json:"types"`
	Nodes            []json.RawMessage `json:"nodes"`
}

type presentationDocument struct {
	Format             string           `json:"format"`
	Version            string           `json:"version"`
	PresentationDigest artifact.Digest  `json:"presentationDigest"`
	Body               presentationBody `json:"body"`
}

type GeneratedArtifacts struct {
	Catalog       []byte
	Presentation  []byte
	Documentation []byte
}

func GenerateArtifacts() (GeneratedArtifacts, error) {
	builtins, err := Build()
	if err != nil {
		return GeneratedArtifacts{}, err
	}
	body := presentationBody{
		GeneratorVersion: GeneratorVersion,
		Types:            make([]json.RawMessage, 0, len(builtins.Types)),
		Nodes:            make([]json.RawMessage, 0, len(builtins.Contracts)),
	}
	for _, definition := range builtins.Types {
		body.Types = append(body.Types, definition.Bytes())
	}
	for _, contract := range builtins.Contracts {
		body.Nodes = append(body.Nodes, contract.Bytes())
	}
	bodyBytes, err := artifact.Marshal(body)
	if err != nil {
		return GeneratedArtifacts{}, err
	}
	digest, err := artifact.Sum("yotta/node-presentation/v1", bodyBytes)
	if err != nil {
		return GeneratedArtifacts{}, err
	}
	presentation, err := artifact.Marshal(presentationDocument{
		Format: PresentationFormat, Version: PresentationVersion,
		PresentationDigest: digest, Body: body,
	})
	if err != nil {
		return GeneratedArtifacts{}, err
	}
	documentation := generateDocumentation(builtins)
	return GeneratedArtifacts{
		Catalog:       builtins.Catalog.Bytes(),
		Presentation:  append(presentation, '\n'),
		Documentation: []byte(strings.TrimRight(documentation, "\n") + "\n"),
	}, nil
}

func generateDocumentation(builtins Builtins) string {
	var builder strings.Builder
	builder.WriteString("# Yotta 3.1 built-in nodes\n\n")
	builder.WriteString("Generated from sealed Data Type and Node Contract artifacts. Do not edit.\n\n")
	for _, contract := range builtins.Contracts {
		machine := contract.Machine()
		authoring := contract.Authoring()
		fmt.Fprintf(&builder, "## `%s`\n\n", machine.NodeTypeID)
		fmt.Fprintf(&builder, "- Title key: `%s`\n", authoring.TitleKey)
		fmt.Fprintf(&builder, "- Execution: `%s` / `%s` / cache `%s`\n", machine.Execution.Class, machine.Execution.Determinism, machine.Execution.Cache)
		if len(machine.CapabilityRequirements) == 0 {
			builder.WriteString("- Capabilities: none\n\n")
		} else {
			builder.WriteString("- Capabilities:\n")
			for _, requirement := range machine.CapabilityRequirements {
				fmt.Fprintf(&builder, "  - `%s`: `%s` operations `%s`\n", requirement.ID, requirement.Capability.CapabilityID, strings.Join(requirement.Operations, "`, `"))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("| Channel | Direction | Port | Type | Required | Resource lease |\n")
		builder.WriteString("| --- | --- | --- | --- | --- | --- |\n")
		for _, port := range machine.Ports.DataInputs {
			fmt.Fprintf(&builder, "| data | input | `%s` | `%s` | %t | %s |\n", port.ID, typeExpressionLabel(port.Type), port.Required, leaseLabel(port.ResourceLease))
		}
		for _, port := range machine.Ports.DataOutputs {
			fmt.Fprintf(&builder, "| data | output | `%s` | `%s` | — | %s |\n", port.ID, typeExpressionLabel(port.Type), leaseLabel(port.ResourceLease))
		}
		for _, port := range machine.Ports.ErrorOutputs {
			fmt.Fprintf(&builder, "| error | output | `%s` | — | — | — |\n", port.ID)
		}
		for _, port := range machine.Ports.StatusOutputs {
			fmt.Fprintf(&builder, "| status | output | `%s` | — | — | — |\n", port.ID)
		}
		if len(machine.Ports.ExecInputs)+len(machine.Ports.ExecOutputs)+len(machine.Ports.ErrorOutputs)+len(machine.Ports.StatusOutputs) == 0 {
			builder.WriteString("\nExec, Error, and Status ports: none.\n")
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func leaseLabel(lease *nodecontract.ResourceLeaseBinding) string {
	if lease == nil {
		return "—"
	}
	return fmt.Sprintf("`%s` (`%s`)", lease.RequirementID, strings.Join(lease.Operations, "`, `"))
}

func typeExpressionLabel(expression datatype.TypeExpression) string {
	switch expression.Kind {
	case datatype.TypeExpressionRef:
		if expression.Ref != nil {
			return expression.Ref.TypeID
		}
	case datatype.TypeExpressionList:
		if expression.Element != nil {
			return "list<" + typeExpressionLabel(*expression.Element) + ">"
		}
	case datatype.TypeExpressionUnion:
		members := make([]string, 0, len(expression.Members))
		for _, member := range expression.Members {
			members = append(members, typeExpressionLabel(member))
		}
		return strings.Join(members, " | ")
	case datatype.TypeExpressionVariable:
		return "$" + expression.Variable
	}
	return "invalid"
}
