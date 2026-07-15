package nodes31

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/yottaapp/yotta/internal/nodeauthoring"
)

const (
	GeneratorVersion = "v1"
)

type GeneratedArtifacts struct {
	Catalog       []byte
	Authoring     []byte
	Documentation []byte
}

func GenerateArtifacts() (GeneratedArtifacts, error) {
	builtins, err := Build()
	if err != nil {
		return GeneratedArtifacts{}, err
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: GeneratorVersion,
	})
	if err != nil {
		return GeneratedArtifacts{}, err
	}
	documentation, err := generateDocumentation(builtins, projection)
	if err != nil {
		return GeneratedArtifacts{}, err
	}
	return GeneratedArtifacts{
		Catalog:       builtins.Catalog.Bytes(),
		Authoring:     projection.Bytes(),
		Documentation: []byte(strings.TrimRight(documentation, "\n") + "\n"),
	}, nil
}

func generateDocumentation(builtins Builtins, authoring nodeauthoring.Snapshot) (string, error) {
	var builder strings.Builder
	builder.WriteString("# Yotta 3.1 built-in nodes\n\n")
	fmt.Fprintf(&builder, "Generated from the strict Node Authoring Projection `%s`. Do not edit.\n\n", authoring.Digest())
	for _, contract := range builtins.Contracts {
		projected, ok := authoring.Node(contract.NodeRef().NodeTypeID)
		if !ok {
			return "", fmt.Errorf("documentation projection missing node %q", contract.NodeRef().NodeTypeID)
		}
		fmt.Fprintf(&builder, "## `%s`\n\n", projected.NodeRef.NodeTypeID)
		fmt.Fprintf(&builder, "- Authoring projection: `%s`\n", authoring.Digest())
		fmt.Fprintf(&builder, "- Title key: `%s`\n", projected.TitleKey)
		fmt.Fprintf(&builder, "- Availability: `%s`\n", projected.Availability)
		fmt.Fprintf(&builder, "- Execution: `%s` / `%s` / cache `%s`\n", projected.Execution.Class, projected.Execution.Determinism, projected.Execution.Cache)
		if len(projected.Capabilities) == 0 {
			builder.WriteString("- Capabilities: none\n\n")
		} else {
			builder.WriteString("- Capabilities:\n")
			for _, requirement := range projected.Capabilities {
				fmt.Fprintf(&builder, "  - `%s`: `%s`; target `%s`; risk `%s`; consent `%s`; operations `%s`\n",
					requirement.RequirementID, requirement.Capability.CapabilityID, requirement.TargetSlot,
					requirement.Risk, requirement.Consent, strings.Join(requirement.Operations, "`, `"))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |\n")
		builder.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
		for _, port := range projected.DataInputs {
			fmt.Fprintf(&builder, "| input | `%s` | `%s` | `%s` | `%s` | `%s` | %s |\n", port.ID, port.Type.Label, port.Type.Lifecycle, port.Carrier, port.Binding, portDefaultLabel(port))
		}
		for _, port := range projected.DataOutputs {
			fmt.Fprintf(&builder, "| output | `%s` | `%s` | `%s` | `%s` | `%s` | — |\n", port.ID, port.Type.Label, port.Type.Lifecycle, port.Carrier, port.Binding)
		}
		builder.WriteString("\n")
		if len(projected.ConfigFields) == 0 {
			builder.WriteString("Configuration fields: none.\n\n")
		} else {
			builder.WriteString("| Configuration field | Control | Required | Constraints |\n")
			builder.WriteString("| --- | --- | --- | --- |\n")
			for _, field := range projected.ConfigFields {
				required := "no"
				if field.Required {
					required = "yes"
				}
				fmt.Fprintf(&builder, "| `%s` | `%s` | %s | `%s` |\n", field.ID, field.Control, required, constraintLabel(field))
			}
			builder.WriteString("\n")
		}
		if len(projected.Signals) == 0 {
			builder.WriteString("Exec, Error, and Status ports: none.\n")
		} else {
			builder.WriteString("| Signal channel | Direction | Port |\n")
			builder.WriteString("| --- | --- | --- |\n")
			for _, signal := range projected.Signals {
				fmt.Fprintf(&builder, "| `%s` | `%s` | `%s` |\n", signal.Channel, signal.Direction, signal.ID)
			}
		}
		builder.WriteString("\n")
	}
	return builder.String(), nil
}

func portDefaultLabel(port nodeauthoring.PortProjection) string {
	if !port.HasDefault {
		return "—"
	}
	return "`" + string(port.Default) + "`"
}

func constraintLabel(field nodeauthoring.FieldProjection) string {
	parts := make([]string, 0, 8)
	constraints := field.Constraints
	if constraints.MinLength != nil {
		parts = append(parts, "minLength: "+strconv.Itoa(*constraints.MinLength))
	}
	if constraints.MaxLength != nil {
		parts = append(parts, "maxLength: "+strconv.Itoa(*constraints.MaxLength))
	}
	if len(constraints.Minimum) != 0 {
		parts = append(parts, "minimum: "+string(constraints.Minimum))
	}
	if len(constraints.Maximum) != 0 {
		parts = append(parts, "maximum: "+string(constraints.Maximum))
	}
	if constraints.MinItems != nil {
		parts = append(parts, "minItems: "+strconv.Itoa(*constraints.MinItems))
	}
	if constraints.MaxItems != nil {
		parts = append(parts, "maxItems: "+strconv.Itoa(*constraints.MaxItems))
	}
	if constraints.Pattern != "" {
		parts = append(parts, "pattern: "+constraints.Pattern)
	}
	if len(constraints.Enum) != 0 {
		values := make([]string, 0, len(constraints.Enum))
		for _, value := range constraints.Enum {
			encoded, _ := json.Marshal(json.RawMessage(value))
			values = append(values, string(encoded))
		}
		parts = append(parts, "enum: "+strings.Join(values, ", "))
	}
	if field.HasDefault {
		parts = append(parts, "default hint: "+string(field.Default))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, ", ")
}
